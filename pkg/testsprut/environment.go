package testsprut

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/udisondev/sprut/pkg/client"
	"github.com/udisondev/sprut/pkg/config"
	"github.com/udisondev/sprut/pkg/identity"
	"github.com/udisondev/sprut/pkg/message"
	"github.com/udisondev/sprut/pkg/router"
)

// Environment представляет тестовое окружение Sprut.
type Environment struct {
	// NATSUrl URL для подключения к NATS.
	NATSUrl string
	// SprutAddr адрес Sprut сервера (host:port).
	SprutAddr string
	// CACert CA сертификат для TLS клиентов.
	CACert []byte

	nats      *natsContainer
	certs     *Certs
	listener  net.Listener
	cancelCtx context.CancelFunc
	serverErr chan error
}

// Option опция конфигурации окружения.
type Option func(*options)

type options struct {
	maxConnections  int
	maxMessageSize  int
	rateLimitPerSec float64
	rateLimitBurst  int
	authTimeout     time.Duration
	challengeTTL    time.Duration
	serverID        string
}

func defaultOptions() *options {
	return &options{
		maxConnections:  100,
		maxMessageSize:  65536,
		rateLimitPerSec: 1000,
		rateLimitBurst:  100,
		authTimeout:     10 * time.Second,
		challengeTTL:    60 * time.Second,
		serverID:        "test-sprut",
	}
}

// WithMaxConnections устанавливает максимальное количество соединений.
func WithMaxConnections(n int) Option {
	return func(o *options) { o.maxConnections = n }
}

// WithMaxMessageSize устанавливает максимальный размер сообщения.
func WithMaxMessageSize(n int) Option {
	return func(o *options) { o.maxMessageSize = n }
}

// WithRateLimit устанавливает лимиты для rate limiter.
func WithRateLimit(perSec float64, burst int) Option {
	return func(o *options) {
		o.rateLimitPerSec = perSec
		o.rateLimitBurst = burst
	}
}

// WithAuthTimeout устанавливает таймаут аутентификации.
func WithAuthTimeout(d time.Duration) Option {
	return func(o *options) { o.authTimeout = d }
}

// WithChallengeTTL устанавливает TTL для challenge.
func WithChallengeTTL(d time.Duration) Option {
	return func(o *options) { o.challengeTTL = d }
}

// WithServerID устанавливает идентификатор сервера.
func WithServerID(id string) Option {
	return func(o *options) { o.serverID = id }
}

// Start запускает тестовое окружение: NATS контейнер + Sprut сервер.
func Start(ctx context.Context, opts ...Option) (*Environment, error) {
	o := defaultOptions()
	for _, opt := range opts {
		opt(o)
	}

	// 1. Запускаем NATS
	nats, err := startNATS(ctx)
	if err != nil {
		return nil, fmt.Errorf("start NATS: %w", err)
	}

	// 2. Генерируем TLS сертификаты
	certs, err := GenerateCerts()
	if err != nil {
		nats.Terminate(ctx)
		return nil, fmt.Errorf("generate certs: %w", err)
	}

	// 3. Создаём TCP listener на случайном порту
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		certs.Cleanup()
		nats.Terminate(ctx)
		return nil, fmt.Errorf("create listener: %w", err)
	}

	addr := lis.Addr().String()
	host, port, _ := net.SplitHostPort(addr)

	// 4. Конфигурация Sprut
	ready := make(chan struct{})
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host:     host,
			Port:     mustAtoi(port),
			ServerID: o.serverID,
		},
		TLS: config.TLSConfig{
			CertFile: certs.CertFile,
			KeyFile:  certs.KeyFile,
		},
		NATS: config.NATSConfig{
			URLs:          []string{nats.URL()},
			ReconnectWait: time.Second,
			MaxReconnects: 5,
		},
		Limits: config.LimitsConfig{
			MaxConnections:  o.maxConnections,
			MaxMessageSize:  o.maxMessageSize,
			RateLimitPerSec: o.rateLimitPerSec,
			RateLimitBurst:  o.rateLimitBurst,
			AuthTimeout:     o.authTimeout,
			ChallengeTTL:    o.challengeTTL,
		},
		Ready: ready,
	}

	// 5. Запускаем Sprut сервер в горутине
	serverCtx, cancelCtx := context.WithCancel(ctx)
	serverErr := make(chan error, 1)

	go func() {
		serverErr <- router.Serve(serverCtx, cfg, lis)
	}()

	// 6. Ждём готовности сервера
	select {
	case <-ready:
		// Сервер готов
	case err := <-serverErr:
		cancelCtx()
		certs.Cleanup()
		nats.Terminate(ctx)
		return nil, fmt.Errorf("server failed to start: %w", err)
	case <-time.After(30 * time.Second):
		cancelCtx()
		certs.Cleanup()
		nats.Terminate(ctx)
		return nil, fmt.Errorf("server start timeout")
	}

	return &Environment{
		NATSUrl:   nats.URL(),
		SprutAddr: addr,
		CACert:    certs.CACert,
		nats:      nats,
		certs:     certs,
		listener:  lis,
		cancelCtx: cancelCtx,
		serverErr: serverErr,
	}, nil
}

// Close останавливает тестовое окружение.
func (e *Environment) Close(ctx context.Context) error {
	// Останавливаем Sprut сервер
	if e.cancelCtx != nil {
		e.cancelCtx()
	}

	// Ждём завершения сервера
	select {
	case <-e.serverErr:
	case <-time.After(5 * time.Second):
	}

	// Очищаем ресурсы
	var errs []error

	if e.certs != nil {
		if err := e.certs.Cleanup(); err != nil {
			errs = append(errs, fmt.Errorf("cleanup certs: %w", err))
		}
	}

	if e.nats != nil {
		if err := e.nats.Terminate(ctx); err != nil {
			errs = append(errs, fmt.Errorf("terminate NATS: %w", err))
		}
	}

	if len(errs) > 0 {
		return errs[0]
	}
	return nil
}

// NewClient создаёт клиент Sprut с указанными ключами.
// Клиент подключён и готов к использованию.
// Вызывающий код отвечает за закрытие канала отправки.
func (e *Environment) NewClient(ctx context.Context, keys *identity.KeyPair) (*Client, error) {
	send := make(chan client.OutgoingMessage, 10)

	recv, err := client.Connect(e.SprutAddr, send,
		client.WithKeys(keys),
		client.WithInsecureSkipVerify(),
		client.WithDialTimeout(10*time.Second),
		client.WithReadTimeout(30*time.Second),
		client.WithWriteTimeout(10*time.Second),
	)
	if err != nil {
		close(send)
		return nil, fmt.Errorf("connect to Sprut: %w", err)
	}

	return &Client{
		send:      send,
		recv:      recv,
		pubKeyHex: keys.PublicKeyHex(),
	}, nil
}

// Client обёртка над клиентом Sprut для тестов.
type Client struct {
	send      chan client.OutgoingMessage
	recv      <-chan *message.Message
	pubKeyHex string
}

// Send возвращает канал для отправки сообщений.
func (c *Client) Send() chan<- client.OutgoingMessage {
	return c.send
}

// Recv возвращает канал для получения сообщений.
func (c *Client) Recv() <-chan *message.Message {
	return c.recv
}

// PubKeyHex возвращает hex-представление публичного ключа клиента.
func (c *Client) PubKeyHex() string {
	return c.pubKeyHex
}

// Close закрывает клиент.
func (c *Client) Close() {
	close(c.send)
}

// SendMessage отправляет сообщение получателю.
func (c *Client) SendMessage(to, msgID string, payload []byte) {
	c.send <- client.OutgoingMessage{
		To:      to,
		MsgID:   msgID,
		Payload: payload,
	}
}

func mustAtoi(s string) int {
	var n int
	for _, c := range s {
		n = n*10 + int(c-'0')
	}
	return n
}
