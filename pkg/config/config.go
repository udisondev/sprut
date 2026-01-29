// Package config реализует загрузку конфигурации.
package config

import (
	"errors"
	"fmt"
	"os"
	"time"
)

// Config конфигурация сервера.
type Config struct {
	Server ServerConfig `yaml:"server"`
	TLS    TLSConfig    `yaml:"tls"`
	NATS   NATSConfig   `yaml:"nats"`
	Limits LimitsConfig `yaml:"limits"`
	Log    LogConfig    `yaml:"log"`

	// Ready закрывается когда сервер полностью готов к приёму соединений.
	// Опциональное поле, используется для тестов.
	Ready chan struct{} `yaml:"-"`
}

// ServerConfig конфигурация TCP сервера.
type ServerConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	ServerID string `yaml:"server_id"`
}

// Addr возвращает адрес сервера в формате host:port.
func (c ServerConfig) Addr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

// TLSConfig конфигурация TLS.
type TLSConfig struct {
	CertFile   string `yaml:"cert_file"`
	KeyFile    string `yaml:"key_file"`
	MinVersion string `yaml:"min_version"`
}

// NATSConfig конфигурация NATS.
type NATSConfig struct {
	URLs          []string      `yaml:"urls"`
	ReconnectWait time.Duration `yaml:"reconnect_wait"`
	MaxReconnects int           `yaml:"max_reconnects"`
}

// LimitsConfig конфигурация лимитов.
type LimitsConfig struct {
	MaxConnections  int           `yaml:"max_connections"`
	MaxMessageSize  int           `yaml:"max_message_size"`
	RateLimitPerSec float64       `yaml:"rate_limit_per_sec"`
	RateLimitBurst  int           `yaml:"rate_limit_burst"`
	AuthTimeout     time.Duration `yaml:"auth_timeout"`
	ChallengeTTL    time.Duration `yaml:"challenge_ttl"`
}

// LogConfig конфигурация логирования.
type LogConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
	File   string `yaml:"file"` // путь к файлу логов (пустой = stdout)
}

// Validate проверяет корректность конфигурации.
func (c *Config) Validate() error {
	var errs []error

	// Server
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		errs = append(errs, fmt.Errorf("invalid server port: %d", c.Server.Port))
	}
	if c.Server.ServerID == "" {
		errs = append(errs, fmt.Errorf("server_id is required"))
	}

	// TLS
	if c.TLS.CertFile == "" {
		errs = append(errs, fmt.Errorf("tls.cert_file is required"))
	} else if _, err := os.Stat(c.TLS.CertFile); err != nil {
		errs = append(errs, fmt.Errorf("tls.cert_file: %w", err))
	}
	if c.TLS.KeyFile == "" {
		errs = append(errs, fmt.Errorf("tls.key_file is required"))
	} else if _, err := os.Stat(c.TLS.KeyFile); err != nil {
		errs = append(errs, fmt.Errorf("tls.key_file: %w", err))
	}

	// NATS
	if len(c.NATS.URLs) == 0 {
		errs = append(errs, fmt.Errorf("nats.urls is required"))
	}

	// Limits
	if c.Limits.MaxConnections < 1 {
		errs = append(errs, fmt.Errorf("limits.max_connections must be positive"))
	}
	if c.Limits.MaxMessageSize < 1 {
		errs = append(errs, fmt.Errorf("limits.max_message_size must be positive"))
	}
	if c.Limits.AuthTimeout <= 0 {
		errs = append(errs, fmt.Errorf("limits.auth_timeout must be positive"))
	}
	if c.Limits.ChallengeTTL <= 0 {
		errs = append(errs, fmt.Errorf("limits.challenge_ttl must be positive"))
	}

	return errors.Join(errs...)
}

// Default возвращает конфигурацию по умолчанию.
func Default() *Config {
	return &Config{
		Server: ServerConfig{
			Host:     "0.0.0.0",
			Port:     8443,
			ServerID: "goro-1",
		},
		TLS: TLSConfig{
			MinVersion: "1.3",
		},
		NATS: NATSConfig{
			URLs:          []string{"nats://localhost:4222"},
			ReconnectWait: 2 * time.Second,
			MaxReconnects: -1,
		},
		Limits: LimitsConfig{
			MaxConnections:  10000,
			MaxMessageSize:  65536,
			RateLimitPerSec: 100,
			RateLimitBurst:  10,
			AuthTimeout:     10 * time.Second,
			ChallengeTTL:    60 * time.Second,
		},
		Log: LogConfig{
			Level:  "info",
			Format: "json",
		},
	}
}
