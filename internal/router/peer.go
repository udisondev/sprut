package router

import (
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"golang.org/x/time/rate"

	"github.com/udisondev/sprut/internal/broker"
	"github.com/udisondev/sprut/pkg/protocol"
)

// PeerID идентификатор пира (публичный ключ).
type PeerID [protocol.PublicKeySize]byte

// Peer представляет аутентифицированного клиента.
type Peer struct {
	id        PeerID
	conn      net.Conn
	pubKeyHex string

	publisher  *broker.Publisher
	subscriber *broker.Subscriber

	writeCh   chan []byte
	closeCh   chan struct{}
	closeOnce sync.Once

	writeTimeout time.Duration
	// lastDeadline используется для batch deadline updates -
	// обновляем deadline только каждые writeTimeout/2.
	lastDeadline time.Time

	// limiter ограничивает количество сообщений от клиента для защиты от DoS.
	limiter *rate.Limiter
}

// newPeer создаёт нового пира.
func newPeer(
	conn net.Conn,
	id PeerID,
	brk *broker.Broker,
	writeBufferSize int,
	writeTimeout time.Duration,
	rateLimitPerSec float64,
	rateLimitBurst int,
) (*Peer, error) {
	pubKeyHex := hex.EncodeToString(id[:])

	peer := &Peer{
		id:           id,
		conn:         conn,
		pubKeyHex:    pubKeyHex,
		publisher:    broker.NewPublisher(brk),
		writeCh:      make(chan []byte, writeBufferSize),
		closeCh:      make(chan struct{}),
		writeTimeout: writeTimeout,
		limiter:      rate.NewLimiter(rate.Limit(rateLimitPerSec), rateLimitBurst),
	}

	// Подписываемся на топик "goro.msg.{pubKeyHex}" для получения входящих сообщений
	subscriber, err := broker.NewSubscriber(brk, pubKeyHex, peer.handleNATSMessage)
	if err != nil {
		return nil, fmt.Errorf("create subscriber: %w", err)
	}
	peer.subscriber = subscriber

	return peer, nil
}

// PubKeyHex возвращает hex-представление публичного ключа.
func (p *Peer) PubKeyHex() string {
	return p.pubKeyHex
}

// AllowMessage проверяет, разрешено ли клиенту отправить сообщение (rate limiting).
// Возвращает true если разрешено, false если лимит превышен.
func (p *Peer) AllowMessage() bool {
	return p.limiter.Allow()
}

// Close закрывает соединение с пиром.
func (p *Peer) Close() {
	p.closeOnce.Do(func() {
		close(p.closeCh)
		if p.subscriber != nil {
			if err := p.subscriber.Unsubscribe(); err != nil {
				slog.Error("unsubscribe", "error", err, "client", p.pubKeyHex)
			}
		}
		if err := p.conn.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
			slog.Error("close connection", "error", err, "client", p.pubKeyHex)
		}
	})
}

// writeLoop обрабатывает исходящие сообщения.
func (p *Peer) writeLoop() {
	for {
		select {
		case <-p.closeCh:
			return
		case data := <-p.writeCh:
			if err := p.writeMessage(data); err != nil {
				slog.Error("write message", "error", err, "client", p.pubKeyHex)
				p.Close()
				return
			}
		}
	}
}

// writeMessage отправляет сообщение клиенту.
// Вызывается только из writeLoop, поэтому mutex не нужен.
func (p *Peer) writeMessage(data []byte) error {
	now := time.Now()
	// Batch deadline updates: обновляем только каждые writeTimeout/2
	// Это снижает количество syscall с 2 на сообщение до ~0.07 на сообщение
	if now.Sub(p.lastDeadline) > p.writeTimeout/2 {
		if err := p.conn.SetWriteDeadline(now.Add(p.writeTimeout)); err != nil {
			return fmt.Errorf("set write deadline: %w", err)
		}
		p.lastDeadline = now
	}

	// ServerMessage: Len(4) + Data
	serverMsg := &protocol.ServerMessage{Data: data}
	if err := serverMsg.Encode(p.conn); err != nil {
		return fmt.Errorf("encode server message: %w", err)
	}

	return nil
}

// handleNATSMessage обрабатывает входящие сообщения из NATS.
func (p *Peer) handleNATSMessage(msg *nats.Msg) {
	select {
	case <-p.closeCh:
		return
	case p.writeCh <- msg.Data:
		// OK - сообщение добавлено в очередь
	default:
		// Буфер переполнен - клиент не успевает обрабатывать (slow consumer)
		// Проверяем ещё раз closeCh для предотвращения race condition
		select {
		case <-p.closeCh:
			return
		default:
			slog.Warn("write buffer full, disconnecting slow client", "client", p.pubKeyHex)
			p.Close()
		}
	}
}
