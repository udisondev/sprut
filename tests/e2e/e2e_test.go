// Package e2e содержит end-to-end тесты для goro.
package e2e

import (
	"context"
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
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/udisondev/sprut/internal/config"
	"github.com/udisondev/sprut/internal/router"
	"github.com/udisondev/sprut/pkg/client"
	"github.com/udisondev/sprut/pkg/identity"
	"github.com/udisondev/sprut/pkg/message"
)

const (
	// testServerPort порт для goro сервера в тестах.
	testServerPort = 18443
	// testServerAddr адрес сервера в тестах.
	testServerAddr = "127.0.0.1:18443"
)

// TestMessageExchange запускает сервер, подключает клиентов и обменивается сообщениями.
func TestMessageExchange(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// 1. NATS
	natsURL := startNATS(t, ctx)
	t.Logf("NATS: %s", natsURL)

	// 2. TLS certs
	certFile, keyFile := generateCerts(t)

	// 3. Goro server
	serverCtx, serverCancel := context.WithCancel(ctx)
	defer serverCancel()

	ready := startServer(t, serverCtx, natsURL, certFile, keyFile)
	select {
	case <-ready:
	case <-time.After(30 * time.Second):
		t.Fatal("server didn't start")
	}
	t.Logf("Server: %s", testServerAddr)

	// 4. Alice
	alice := connect(t, testServerAddr)
	defer alice.close()
	t.Logf("Alice: %s", alice.pubKey[:16])

	// 5. Bob
	bob := connect(t, testServerAddr)
	defer bob.close()
	t.Logf("Bob: %s", bob.pubKey[:16])

	// 6. Alice -> Bob
	alice.send <- client.OutgoingMessage{
		To:      bob.pubKey,
		MsgID:   "msg-1",
		Payload: []byte("Hello Bob!"),
	}

	msg := waitMsg(t, bob.recv, 10*time.Second)
	require.Equal(t, alice.pubKey, msg.From)
	require.Equal(t, "Hello Bob!", string(msg.Payload))
	t.Logf("Bob got: %s", string(msg.Payload))

	// 7. Bob -> Alice
	bob.send <- client.OutgoingMessage{
		To:      alice.pubKey,
		MsgID:   "msg-2",
		Payload: []byte("Hello Alice!"),
	}

	msg = waitMsg(t, alice.recv, 10*time.Second)
	require.Equal(t, bob.pubKey, msg.From)
	require.Equal(t, "Hello Alice!", string(msg.Payload))
	t.Logf("Alice got: %s", string(msg.Payload))

	t.Log("OK")
}

type testClient struct {
	send   chan client.OutgoingMessage
	recv   <-chan *message.Message
	pubKey string
}

func (c *testClient) close() {
	close(c.send)
}

func startNATS(t *testing.T, ctx context.Context) string {
	t.Helper()

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "nats:latest",
			ExposedPorts: []string{"4222/tcp"},
			WaitingFor:   wait.ForListeningPort("4222/tcp").WithStartupTimeout(30 * time.Second),
		},
		Started: true,
	})
	require.NoError(t, err)

	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = container.Terminate(ctx)
	})

	host, err := container.Host(ctx)
	require.NoError(t, err)

	port, err := container.MappedPort(ctx, "4222")
	require.NoError(t, err)

	return fmt.Sprintf("nats://%s:%s", host, port.Port())
}

func generateCerts(t *testing.T) (string, string) {
	t.Helper()

	dir := t.TempDir()
	certFile := filepath.Join(dir, "cert.pem")
	keyFile := filepath.Join(dir, "key.pem")

	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	tmpl := x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "localhost"},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1")},
	}

	der, err := x509.CreateCertificate(rand.Reader, &tmpl, &tmpl, &priv.PublicKey, priv)
	require.NoError(t, err)

	cf, err := os.Create(certFile)
	require.NoError(t, err)
	require.NoError(t, pem.Encode(cf, &pem.Block{Type: "CERTIFICATE", Bytes: der}))
	require.NoError(t, cf.Close())

	kf, err := os.Create(keyFile)
	require.NoError(t, err)
	kb, err := x509.MarshalECPrivateKey(priv)
	require.NoError(t, err)
	require.NoError(t, pem.Encode(kf, &pem.Block{Type: "EC PRIVATE KEY", Bytes: kb}))
	require.NoError(t, kf.Close())

	return certFile, keyFile
}

func startServer(t *testing.T, ctx context.Context, natsURL, certFile, keyFile string) <-chan struct{} {
	t.Helper()

	lis, err := net.Listen("tcp", testServerAddr)
	require.NoError(t, err)

	ready := make(chan struct{})

	cfg := &config.Config{
		Server: config.ServerConfig{
			Host:     "127.0.0.1",
			Port:     testServerPort,
			ServerID: "test",
		},
		TLS: config.TLSConfig{CertFile: certFile, KeyFile: keyFile},
		NATS: config.NATSConfig{
			URLs:          []string{natsURL},
			ReconnectWait: time.Second,
			MaxReconnects: 5,
		},
		Limits: config.LimitsConfig{
			MaxConnections:  100,
			MaxMessageSize:  65536,
			RateLimitPerSec: 1000,
			RateLimitBurst:  100,
			AuthTimeout:     10 * time.Second,
			ChallengeTTL:    60 * time.Second,
		},
		Ready: ready,
	}

	go func() {
		if err := router.Serve(ctx, cfg, lis); err != nil && ctx.Err() == nil {
			t.Errorf("router.Serve: %v", err)
		}
	}()

	return ready
}

func connect(t *testing.T, addr string) *testClient {
	t.Helper()

	keys, err := identity.Generate()
	require.NoError(t, err)

	send := make(chan client.OutgoingMessage, 10)

	recv, err := client.Connect(addr, send,
		client.WithKeys(keys),
		client.WithInsecureSkipVerify(),
		client.WithDialTimeout(10*time.Second),
		client.WithReadTimeout(30*time.Second),
		client.WithWriteTimeout(10*time.Second),
	)
	require.NoError(t, err)

	return &testClient{
		send:   send,
		recv:   recv,
		pubKey: keys.PublicKeyHex(),
	}
}

func waitMsg(t *testing.T, ch <-chan *message.Message, timeout time.Duration) *message.Message {
	t.Helper()
	select {
	case m := <-ch:
		return m
	case <-time.After(timeout):
		t.Fatal("timeout")
		return nil
	}
}
