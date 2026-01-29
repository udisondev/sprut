package client

import (
	"bufio"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"sync"
	"time"

	"github.com/udisondev/sprut/pkg/identity"
	"github.com/udisondev/sprut/pkg/message"
	"github.com/udisondev/sprut/pkg/protocol"
	"google.golang.org/protobuf/proto"
)

// OutgoingMessage сообщение для отправки.
type OutgoingMessage struct {
	To      string
	MsgID   string
	Payload []byte
}

// buildTLSConfig создаёт TLS конфигурацию на основе опций.
func (cfg *connectConfig) buildTLSConfig() (*tls.Config, error) {
	// Если указан полный TLS config — используем его как есть
	if cfg.tlsConfig != nil {
		return cfg.tlsConfig, nil
	}

	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS13,
	}

	// Загрузка CA из файлов
	if len(cfg.caCertPaths) > 0 {
		pool := x509.NewCertPool()
		for _, path := range cfg.caCertPaths {
			caCert, err := os.ReadFile(path)
			if err != nil {
				return nil, fmt.Errorf("read CA cert %s: %w", path, err)
			}
			if !pool.AppendCertsFromPEM(caCert) {
				return nil, fmt.Errorf("parse CA cert %s: invalid PEM", path)
			}
		}
		tlsConfig.RootCAs = pool
	} else if cfg.rootCAs != nil {
		tlsConfig.RootCAs = cfg.rootCAs
	}

	if cfg.serverName != "" {
		tlsConfig.ServerName = cfg.serverName
	}

	if cfg.insecureSkipVerify {
		tlsConfig.InsecureSkipVerify = true
	}

	return tlsConfig, nil
}

// Connect устанавливает соединение с сервером и возвращает канал входящих сообщений.
//
// Параметры:
//   - addr: адрес сервера (host:port)
//   - send: канал исходящих сообщений. Закрытие канала завершает соединение.
//   - opts: опции подключения
//
// Возвращает канал входящих сообщений. Канал закрывается при завершении соединения.
func Connect(addr string, send <-chan OutgoingMessage, opts ...ConnectOption) (<-chan *message.Message, error) {
	// 1. Дефолтные значения
	keys, err := identity.Generate()
	if err != nil {
		return nil, fmt.Errorf("generate keys: %w", err)
	}

	cfg := &connectConfig{
		keys:         keys,
		localAddr:    DefaultLocalAddr,
		dialTimeout:  DefaultDialTimeout,
		writeTimeout: DefaultWriteTimeout,
		readBufSize:  DefaultReadBufSize,
	}

	// 2. Применяем опции
	for _, opt := range opts {
		opt(cfg)
	}

	// 3. Настраиваем TLS
	tlsConfig, err := cfg.buildTLSConfig()
	if err != nil {
		return nil, fmt.Errorf("build TLS config: %w", err)
	}

	// 4. Подключаемся
	dialer := &net.Dialer{
		Timeout:   cfg.dialTimeout,
		LocalAddr: cfg.localAddr,
	}
	conn, err := tls.DialWithDialer(dialer, "tcp", addr, tlsConfig)
	if err != nil {
		return nil, fmt.Errorf("dial: %w", err)
	}

	// 5. Проходим аутентификацию
	if err := authenticate(conn, cfg.keys, cfg.dialTimeout); err != nil {
		_ = conn.Close() // ошибка Close() не важна, возвращаем ошибку authenticate
		return nil, fmt.Errorf("authenticate: %w", err)
	}

	// 6. Запускаем цикл обработки
	recv := make(chan *message.Message, cfg.readBufSize)
	go runLoop(conn, cfg, send, recv)

	return recv, nil
}

func authenticate(conn *tls.Conn, keys *identity.KeyPair, timeout time.Duration) error {
	if err := conn.SetDeadline(time.Now().Add(timeout)); err != nil {
		return fmt.Errorf("set deadline: %w", err)
	}
	defer func() {
		_ = conn.SetDeadline(time.Time{})
	}()

	reader := bufio.NewReader(conn)

	// 1. Отправляем ClientHello
	hello := &protocol.ClientHello{}
	copy(hello.PubKey[:], keys.PublicKey)
	if err := hello.Encode(conn); err != nil {
		return fmt.Errorf("send client hello: %w", err)
	}

	// 2. Получаем ServerChallenge
	msgType, err := protocol.ReadMessageType(reader)
	if err != nil {
		return fmt.Errorf("read challenge type: %w", err)
	}
	if msgType != protocol.TypeServerChallenge {
		return fmt.Errorf("unexpected message type: %d", msgType)
	}

	challenge, err := protocol.DecodeServerChallenge(reader)
	if err != nil {
		return fmt.Errorf("decode challenge: %w", err)
	}

	// 3. Подписываем и отправляем ClientResponse
	signature, err := signChallenge(keys, challenge, conn)
	if err != nil {
		return fmt.Errorf("sign challenge: %w", err)
	}

	response := &protocol.ClientResponse{Signature: signature}
	if err := response.Encode(conn); err != nil {
		return fmt.Errorf("send response: %w", err)
	}

	// 4. Получаем AuthResult (синхронизация с сервером)
	msgType, err = protocol.ReadMessageType(reader)
	if err != nil {
		return fmt.Errorf("read result type: %w", err)
	}
	if msgType != protocol.TypeAuthResult {
		return fmt.Errorf("unexpected message type: %d", msgType)
	}

	result, err := protocol.DecodeAuthResult(reader)
	if err != nil {
		return fmt.Errorf("decode result: %w", err)
	}

	if result.Status != protocol.AuthStatusOK {
		return fmt.Errorf("%w: %s", protocol.ErrAuthFailed, result.ErrorMsg)
	}

	return nil
}

// runLoop управляет соединением: читает и пишет сообщения.
func runLoop(conn *tls.Conn, cfg *connectConfig, send <-chan OutgoingMessage, recv chan<- *message.Message) {
	var wg sync.WaitGroup
	closeCh := make(chan struct{})
	var closeOnce sync.Once

	closeAll := func() {
		closeOnce.Do(func() {
			close(closeCh)
			if err := conn.Close(); err != nil {
				handleError(cfg, fmt.Errorf("close connection: %w", err))
			}
		})
	}

	// Запускаем читающую горутину
	wg.Add(1)
	go func() {
		defer wg.Done()
		readLoop(conn, cfg, recv, closeCh, closeAll)
	}()

	// Запускаем пишущую горутину
	wg.Add(1)
	go func() {
		defer wg.Done()
		writeLoop(conn, cfg, send, closeCh, closeAll)
	}()

	// Ждём завершения обеих горутин
	wg.Wait()

	// Закрываем канал получения
	close(recv)
}

func readLoop(conn *tls.Conn, cfg *connectConfig, recv chan<- *message.Message, closeCh <-chan struct{}, closeAll func()) {
	defer closeAll()

	reader := bufio.NewReader(conn)

	for {
		select {
		case <-closeCh:
			return
		default:
		}

		serverMsg, err := protocol.DecodeServerMessage(reader)
		if err != nil {
			// EOF или closed — нормальное завершение
			if errors.Is(err, io.EOF) || errors.Is(err, net.ErrClosed) {
				return
			}
			// Проверяем closeCh перед логированием
			select {
			case <-closeCh:
				return
			default:
				handleError(cfg, fmt.Errorf("decode server message: %w", err))
				return
			}
		}

		msg := &message.Message{}
		if err := proto.Unmarshal(serverMsg.Data, msg); err != nil {
			handleError(cfg, fmt.Errorf("unmarshal message: %w", err))
			continue
		}

		select {
		case recv <- msg:
		case <-closeCh:
			return
		}
	}
}

func writeLoop(conn *tls.Conn, cfg *connectConfig, send <-chan OutgoingMessage, closeCh <-chan struct{}, closeAll func()) {
	for {
		select {
		case <-closeCh:
			return
		case msg, ok := <-send:
			if !ok {
				// Канал закрыт — завершаем соединение
				closeAll()
				return
			}
			if err := sendMessage(conn, cfg, &msg); err != nil {
				handleError(cfg, fmt.Errorf("send message: %w", err))
			}
		}
	}
}

func sendMessage(conn *tls.Conn, cfg *connectConfig, msg *OutgoingMessage) error {
	if cfg.writeTimeout > 0 {
		if err := conn.SetWriteDeadline(time.Now().Add(cfg.writeTimeout)); err != nil {
			return fmt.Errorf("set write deadline: %w", err)
		}
	}

	clientMsg := &protocol.ClientMessage{
		To:      msg.To,
		MsgID:   msg.MsgID,
		Payload: msg.Payload,
	}

	return clientMsg.Encode(conn)
}

func handleError(cfg *connectConfig, err error) {
	if cfg.onError != nil {
		cfg.onError(err)
	}
}
