// Package router реализует высокопроизводительный TCP роутер с TLS.
package router

import (
	"crypto/tls"
	"fmt"
	"log/slog"

	"github.com/udisondev/sprut/pkg/config"
)

// buildTLSConfig создаёт production-ready TLS конфигурацию.
func buildTLSConfig(cfg config.TLSConfig) (*tls.Config, error) {
	slog.Debug("tls: loading certificates", "cert_file", cfg.CertFile, "key_file", cfg.KeyFile)

	cert, err := tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
	if err != nil {
		slog.Error("tls: load certificates failed", "error", err, "cert_file", cfg.CertFile, "key_file", cfg.KeyFile)
		return nil, fmt.Errorf("load certificates: %w", err)
	}

	slog.Info("tls: certificates loaded", "cert_file", cfg.CertFile)

	minVersion := tls.VersionTLS12
	minVersionStr := "1.2"
	if cfg.MinVersion == "1.3" {
		minVersion = tls.VersionTLS13
		minVersionStr = "1.3"
	}

	slog.Debug("tls: configuration built", "min_version", minVersionStr, "session_tickets_disabled", true)

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
