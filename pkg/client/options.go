package client

import (
	"crypto/tls"
	"crypto/x509"
	"net"
	"time"

	"github.com/udisondev/sprut/pkg/identity"
)

// Константы по умолчанию.
const (
	DefaultDialTimeout  = 10 * time.Second
	DefaultReadTimeout  = 30 * time.Second
	DefaultWriteTimeout = 30 * time.Second
	DefaultReadBufSize  = 100
)

// DefaultLocalAddr адрес для исходящих соединений по умолчанию.
// Можно переопределить через WithLocalAddr(nil) для системного выбора.
var DefaultLocalAddr = &net.TCPAddr{IP: net.ParseIP("127.0.0.1")}

type connectConfig struct {
	keys *identity.KeyPair

	tlsConfig *tls.Config

	// TLS builder fields
	rootCAs            *x509.CertPool
	caCertPaths        []string
	serverName         string
	insecureSkipVerify bool

	localAddr    *net.TCPAddr
	onError      func(error)
	dialTimeout  time.Duration
	readTimeout  time.Duration
	writeTimeout time.Duration

	readBufSize int
}

// ConnectOption конфигурирует соединение.
type ConnectOption func(*connectConfig)

// WithKeys устанавливает ключи идентификации.
// Если не указано, ключи генерируются автоматически.
func WithKeys(keys *identity.KeyPair) ConnectOption {
	return func(c *connectConfig) {
		c.keys = keys
	}
}

// WithTLSConfig устанавливает TLS конфигурацию.
func WithTLSConfig(cfg *tls.Config) ConnectOption {
	return func(c *connectConfig) {
		c.tlsConfig = cfg
	}
}

// WithOnError устанавливает обработчик ошибок.
func WithOnError(handler func(error)) ConnectOption {
	return func(c *connectConfig) {
		c.onError = handler
	}
}

// WithDialTimeout устанавливает таймаут подключения.
func WithDialTimeout(d time.Duration) ConnectOption {
	return func(c *connectConfig) {
		c.dialTimeout = d
	}
}

// WithReadBufSize устанавливает размер буфера входящих сообщений
func WithReadBufSize(n int) ConnectOption {
	return func(c *connectConfig) {
		c.readBufSize = n
	}
}

// WithReadTimeout устанавливает таймаут чтения.
func WithReadTimeout(d time.Duration) ConnectOption {
	return func(c *connectConfig) {
		c.readTimeout = d
	}
}

// WithWriteTimeout устанавливает таймаут записи.
func WithWriteTimeout(d time.Duration) ConnectOption {
	return func(c *connectConfig) {
		c.writeTimeout = d
	}
}

// WithRootCAs устанавливает пул CA сертификатов для проверки сервера.
func WithRootCAs(pool *x509.CertPool) ConnectOption {
	return func(c *connectConfig) {
		c.rootCAs = pool
	}
}

// WithCACertFile добавляет CA сертификат из PEM-файла.
// Можно вызывать несколько раз для добавления нескольких CA.
func WithCACertFile(path string) ConnectOption {
	return func(c *connectConfig) {
		c.caCertPaths = append(c.caCertPaths, path)
	}
}

// WithServerName устанавливает ServerName для SNI и проверки сертификата.
func WithServerName(name string) ConnectOption {
	return func(c *connectConfig) {
		c.serverName = name
	}
}

// WithInsecureSkipVerify отключает проверку сертификата сервера.
// Использовать только для разработки и тестирования.
func WithInsecureSkipVerify() ConnectOption {
	return func(c *connectConfig) {
		c.insecureSkipVerify = true
	}
}

// WithLocalAddr устанавливает локальный адрес для исходящих соединений.
// По умолчанию используется DefaultLocalAddr (127.0.0.1).
// Передайте nil чтобы использовать системный выбор адреса.
func WithLocalAddr(addr *net.TCPAddr) ConnectOption {
	return func(c *connectConfig) {
		c.localAddr = addr
	}
}
