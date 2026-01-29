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
				slog.Warn("broker: NATS disconnected", "error", err)
			}
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			slog.Info("broker: NATS reconnected", "url", nc.ConnectedUrl())
		}),
		nats.ClosedHandler(func(_ *nats.Conn) {
			slog.Info("broker: NATS connection closed")
		}),
	}

	// NATS поддерживает URL через запятую
	url := nats.DefaultURL
	if len(cfg.URLs) > 0 {
		url = strings.Join(cfg.URLs, ",")
	}

	slog.Debug("broker: connecting", "urls", url)

	conn, err := nats.Connect(url, opts...)
	if err != nil {
		slog.Error("broker: connect failed", "urls", url, "error", err)
		return nil, fmt.Errorf("connect to NATS: %w", err)
	}

	slog.Debug("broker: connection established", "server_id", conn.ConnectedServerId(), "url", conn.ConnectedUrl())

	return &Broker{conn: conn}, nil
}

// Conn возвращает соединение NATS.
func (b *Broker) Conn() *nats.Conn {
	return b.conn
}

// Close закрывает соединение.
func (b *Broker) Close() error {
	slog.Debug("broker: closing connection")
	if err := b.conn.Drain(); err != nil {
		slog.Error("broker: drain failed", "error", err)
		return err
	}
	return nil
}
