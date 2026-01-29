package testsprut

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"time"
)

// Certs содержит пути к сгенерированным TLS сертификатам.
type Certs struct {
	// CertFile путь к файлу сертификата (PEM).
	CertFile string
	// KeyFile путь к файлу приватного ключа (PEM).
	KeyFile string
	// CACert содержимое CA сертификата (для клиентов).
	CACert []byte
	// dir временная директория с файлами.
	dir string
}

// GenerateCerts генерирует самоподписанные TLS сертификаты для тестов.
// Сертификаты сохраняются во временную директорию.
// Вызывающий код отвечает за удаление директории после использования.
func GenerateCerts() (*Certs, error) {
	dir, err := os.MkdirTemp("", "sprut-test-certs-*")
	if err != nil {
		return nil, fmt.Errorf("create temp dir: %w", err)
	}

	certFile := filepath.Join(dir, "cert.pem")
	keyFile := filepath.Join(dir, "key.pem")

	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		os.RemoveAll(dir)
		return nil, fmt.Errorf("generate key: %w", err)
	}

	tmpl := x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "localhost"},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1")},
		DNSNames:              []string{"localhost"},
	}

	der, err := x509.CreateCertificate(rand.Reader, &tmpl, &tmpl, &priv.PublicKey, priv)
	if err != nil {
		os.RemoveAll(dir)
		return nil, fmt.Errorf("create certificate: %w", err)
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	if err := os.WriteFile(certFile, certPEM, 0600); err != nil {
		os.RemoveAll(dir)
		return nil, fmt.Errorf("write cert file: %w", err)
	}

	kb, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		os.RemoveAll(dir)
		return nil, fmt.Errorf("marshal private key: %w", err)
	}

	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: kb})
	if err := os.WriteFile(keyFile, keyPEM, 0600); err != nil {
		os.RemoveAll(dir)
		return nil, fmt.Errorf("write key file: %w", err)
	}

	return &Certs{
		CertFile: certFile,
		KeyFile:  keyFile,
		CACert:   certPEM,
		dir:      dir,
	}, nil
}

// Cleanup удаляет временные файлы сертификатов.
func (c *Certs) Cleanup() error {
	if c.dir == "" {
		return nil
	}
	return os.RemoveAll(c.dir)
}
