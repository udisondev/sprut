package router

import (
	"context"
	"crypto/tls"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"sync"
	"time"

	"github.com/udisondev/sprut/pkg/broker"
	"github.com/udisondev/sprut/pkg/config"
	"github.com/udisondev/sprut/pkg/protocol"
)

// Константы для роутера.
const (
	// WriteBufferSize - размер буфера исходящих сообщений на клиента.
	// При переполнении клиент отключается как slow consumer.
	WriteBufferSize = 1000

	// WriteTimeout - таймаут записи сообщения.
	WriteTimeout = 30 * time.Second
)

// Run создаёт TCP listener и запускает роутер с TLS.
// Аналог http.ListenAndServeTLS.
func Run(ctx context.Context, cfg *config.Config) error {
	addr := cfg.Server.Addr()
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}

	return Serve(ctx, cfg, lis)
}

// Serve запускает роутер на переданном TCP listener.
// Аналог http.ServeTLS — принимает plain TCP listener и оборачивает в TLS.
func Serve(ctx context.Context, cfg *config.Config, lis net.Listener) error {
	tlsConfig, err := buildTLSConfig(cfg.TLS)
	if err != nil {
		return fmt.Errorf("build TLS config: %w", err)
	}

	tlsLis := tls.NewListener(lis, tlsConfig)
	defer func() {
		if err := tlsLis.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
			slog.Error("close TLS listener", "error", err)
		}
	}()

	addr := lis.Addr().String()

	// Graceful shutdown listener
	go func() {
		<-ctx.Done()
		if err := tlsLis.Close(); err != nil {
			slog.Error("close listener", "error", err)
		}
	}()

	// NATS брокер
	brk, err := broker.New(broker.Config{
		URLs:          cfg.NATS.URLs,
		ReconnectWait: cfg.NATS.ReconnectWait,
		MaxReconnects: cfg.NATS.MaxReconnects,
	})
	if err != nil {
		return fmt.Errorf("create broker: %w", err)
	}
	defer func() {
		if err := brk.Close(); err != nil {
			slog.Error("close broker", "error", err)
		}
	}()

	// ServerID в байтах для записи в буферы
	var serverID [protocol.ServerIDSize]byte
	serverIDBytes := []byte(cfg.Server.ServerID)
	if len(serverIDBytes) > protocol.ServerIDSize {
		return fmt.Errorf("server_id too long: max %d bytes, got %d", protocol.ServerIDSize, len(serverIDBytes))
	}
	copy(serverID[:], serverIDBytes)

	// Семафор-с-буфером: одна операция для лимита соединений И получения auth буфера
	authSem := make(chan []byte, cfg.Limits.MaxConnections)
	for range cfg.Limits.MaxConnections {
		buf := make([]byte, AuthBufSize)
		copy(buf[offServerID:offServerID+protocol.ServerIDSize], serverID[:])
		authSem <- buf
	}

	// sync.Pool для буферов сообщений (хранит *[]byte для избежания аллокаций)
	msgPool := &sync.Pool{New: func() any {
		buf := make([]byte, cfg.Limits.MaxMessageSize)
		return &buf
	}}

	// sync.Map для пиров
	var peers sync.Map

	slog.Info("router started", "addr", addr)
	slog.Info("router: configuration",
		"max_connections", cfg.Limits.MaxConnections,
		"max_message_size", cfg.Limits.MaxMessageSize,
		"rate_limit_per_sec", cfg.Limits.RateLimitPerSec,
		"rate_limit_burst", cfg.Limits.RateLimitBurst,
		"auth_timeout", cfg.Limits.AuthTimeout,
		"challenge_ttl", cfg.Limits.ChallengeTTL,
	)

	// Сигнализируем что сервер готов
	if cfg.Ready != nil {
		close(cfg.Ready)
	}

	// Accept loop
	for {
		conn, err := tlsLis.Accept()
		if err != nil {
			if ctx.Err() != nil {
				slog.Info("router shutting down")
				return nil
			}
			if errors.Is(err, net.ErrClosed) {
				return nil
			}
			slog.Error("accept connection", "error", err)
			continue
		}

		select {
		case authBuf := <-authSem:
			slog.Debug("router: auth buffer acquired", "remote", conn.RemoteAddr())
			go func(c net.Conn, buf []byte) {
				defer func() { authSem <- buf }()
				handleConn(c, &peers, buf, msgPool, brk, cfg)
			}(conn, authBuf)
		default:
			slog.Warn("router: connection limit reached", "remote", conn.RemoteAddr())
			if err := conn.Close(); err != nil {
				slog.Error("router: close connection on limit failed", "error", err)
			}
		}
	}
}

// handleConn обрабатывает одно соединение.
func handleConn(
	conn net.Conn,
	peers *sync.Map,
	authBuf []byte,
	msgPool *sync.Pool,
	brk *broker.Broker,
	cfg *config.Config,
) {
	remoteAddr := conn.RemoteAddr().String()
	defer func() {
		if err := conn.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
			slog.Error("close connection", "error", err, "remote", remoteAddr)
		}
	}()

	slog.Debug("new connection", "remote", remoteAddr)

	// Включаем TCP_NODELAY для снижения latency (Nagle's algorithm отключен)
	if tlsConn, ok := conn.(*tls.Conn); ok {
		if tcpConn, ok := tlsConn.NetConn().(*net.TCPConn); ok {
			_ = tcpConn.SetNoDelay(true)
		}
	}

	// 1. Аутентификация (буфер с serverID уже получен из семафора)
	if err := authenticate(conn, cfg.Limits.AuthTimeout, cfg.Limits.ChallengeTTL, authBuf); err != nil {
		if !errors.Is(err, io.EOF) {
			slog.Warn("authentication failed", "error", err, "remote", remoteAddr)
		}
		return
	}

	// PeerID уже в буфере после authenticate()
	var id PeerID
	copy(id[:], authBuf[offPubKey:offPubKey+protocol.PublicKeySize])
	pubKeyHex := hex.EncodeToString(id[:])
	slog.Info("client authenticated", "client", pubKeyHex, "remote", remoteAddr)

	// 2. Создаём peer
	peer, err := newPeer(
		conn, id, brk,
		WriteBufferSize, WriteTimeout,
		cfg.Limits.RateLimitPerSec, cfg.Limits.RateLimitBurst,
	)
	if err != nil {
		slog.Error("router: create peer failed", "error", err, "client", pubKeyHex)
		return
	}
	slog.Debug("router: peer created", "client", pubKeyHex, "remote", remoteAddr)

	// 3. Закрываем старое соединение если есть (reconnect case)
	if old, loaded := peers.LoadAndDelete(id); loaded {
		oldPeer := old.(*Peer)
		slog.Info("closing old connection", "client", pubKeyHex)
		oldPeer.Close()
	}
	peers.Store(id, peer)

	defer func() {
		peers.Delete(id)
		peer.Close()
		slog.Info("client disconnected", "client", pubKeyHex)
	}()

	// 4. Запускаем write loop
	slog.Debug("router: starting read/write loops", "client", pubKeyHex)
	go peer.writeLoop()

	// 5. Read loop (блокирующий)
	for {
		select {
		case <-peer.closeCh:
			return
		default:
		}

		// Rate limiting: проверяем перед чтением сообщения
		if !peer.AllowMessage() {
			slog.Warn("rate limit exceeded, disconnecting client", "client", pubKeyHex)
			return
		}

		if err := handleMessage(peer, msgPool, cfg.Limits.MaxMessageSize); err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, net.ErrClosed) {
				slog.Debug("peer disconnected gracefully", "client", pubKeyHex)
			} else {
				slog.Error("handle message", "error", err, "client", pubKeyHex)
			}
			return
		}
	}
}
