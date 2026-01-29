// Package router реализует высокопроизводительный TCP роутер с TLS.
package router

import (
	"crypto/tls"
	"fmt"

	"github.com/udisondev/sprut/internal/config"
)

// buildTLSConfig создаёт production-ready TLS конфигурацию.
func buildTLSConfig(cfg config.TLSConfig) (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("load certificates: %w", err)
	}

	minVersion := tls.VersionTLS12
	if cfg.MinVersion == "1.3" {
		minVersion = tls.VersionTLS13
	}

	tlsCfg := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   uint16(minVersion),
		// CipherSuites игнорируются для TLS 1.3 (Go выбирает автоматически)
		// Для TLS 1.2 указываем явно безопасные cipher suites
		CipherSuites: []uint16{
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		},
		// PreferServerCipherSuites deprecated since Go 1.17, Go автоматически
		// выбирает наиболее безопасные cipher suites.
		// Session tickets отключены для сохранения forward secrecy.
		// При компрометации ticket key без ротации нарушается PFS.
		SessionTicketsDisabled: true,
	}

	return tlsCfg, nil
}
