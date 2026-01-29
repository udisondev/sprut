// Package broker реализует интеграцию с NATS.
package broker

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/nats-io/nats.go"
)

// Broker управляет соединением с NATS.
type Broker struct {
	conn *nats.Conn
}

// Config конфигурация NATS.
type Config struct {
	URLs          []string
	ReconnectWait time.Duration
	MaxReconnects int
}

// New создаёт новый брокер.
func New(cfg Config) (*Broker, error) {
	opts := []nats.Option{
		nats.ReconnectWait(cfg.ReconnectWait),
		nats.MaxReconnects(cfg.MaxReconnects),
		nats.DisconnectErrHandler(func(_ *nats.Conn, err error) {
			if err != nil {
				slog.Warn("NATS disconnected", "error", err)
			}
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			slog.Info("NATS reconnected", "url", nc.ConnectedUrl())
		}),
		nats.ClosedHandler(func(_ *nats.Conn) {
			slog.Info("NATS connection closed")
		}),
	}

	// NATS поддерживает URL через запятую
	url := nats.DefaultURL
	if len(cfg.URLs) > 0 {
		url = strings.Join(cfg.URLs, ",")
	}

	conn, err := nats.Connect(url, opts...)
	if err != nil {
		return nil, fmt.Errorf("connect to NATS: %w", err)
	}

	return &Broker{conn: conn}, nil
}

// Conn возвращает соединение NATS.
func (b *Broker) Conn() *nats.Conn {
	return b.conn
}

// Close закрывает соединение.
func (b *Broker) Close() error {
	return b.conn.Drain()
}
