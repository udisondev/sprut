// Package e2e содержит end-to-end тесты для Sprut.
package e2e

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/udisondev/sprut/pkg/identity"
	"github.com/udisondev/sprut/pkg/message"
	"github.com/udisondev/sprut/pkg/testsprut"
)

// TestMessageExchange запускает сервер, подключает клиентов и обменивается сообщениями.
func TestMessageExchange(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// 1. Запускаем тестовое окружение
	env, err := testsprut.Start(ctx)
	require.NoError(t, err)
	defer env.Close(ctx)
	t.Logf("NATS: %s", env.NATSUrl)
	t.Logf("Server: %s", env.SprutAddr)

	// 2. Alice
	aliceKeys, err := identity.Generate()
	require.NoError(t, err)
	alice, err := env.NewClient(ctx, aliceKeys)
	require.NoError(t, err)
	defer alice.Close()
	t.Logf("Alice: %s", alice.PubKeyHex()[:16])

	// 3. Bob
	bobKeys, err := identity.Generate()
	require.NoError(t, err)
	bob, err := env.NewClient(ctx, bobKeys)
	require.NoError(t, err)
	defer bob.Close()
	t.Logf("Bob: %s", bob.PubKeyHex()[:16])

	// 4. Alice -> Bob
	alice.SendMessage(bob.PubKeyHex(), "msg-1", []byte("Hello Bob!"))

	msg := waitMsg(t, bob.Recv(), 10*time.Second)
	require.Equal(t, alice.PubKeyHex(), msg.From)
	require.Equal(t, "Hello Bob!", string(msg.Payload))
	t.Logf("Bob got: %s", string(msg.Payload))

	// 5. Bob -> Alice
	bob.SendMessage(alice.PubKeyHex(), "msg-2", []byte("Hello Alice!"))

	msg = waitMsg(t, alice.Recv(), 10*time.Second)
	require.Equal(t, bob.PubKeyHex(), msg.From)
	require.Equal(t, "Hello Alice!", string(msg.Payload))
	t.Logf("Alice got: %s", string(msg.Payload))

	t.Log("OK")
}

// TestMultipleClients проверяет работу с несколькими клиентами.
func TestMultipleClients(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	env, err := testsprut.Start(ctx)
	require.NoError(t, err)
	defer env.Close(ctx)

	// Создаём несколько клиентов
	clients := make([]*testsprut.Client, 5)
	for i := range clients {
		keys, err := identity.Generate()
		require.NoError(t, err)
		c, err := env.NewClient(ctx, keys)
		require.NoError(t, err)
		clients[i] = c
		defer c.Close()
	}

	// Каждый клиент отправляет сообщение первому
	for i := 1; i < len(clients); i++ {
		clients[i].SendMessage(clients[0].PubKeyHex(), "msg", []byte("ping"))
	}

	// Первый клиент должен получить 4 сообщения
	for range 4 {
		msg := waitMsg(t, clients[0].Recv(), 10*time.Second)
		require.Equal(t, "ping", string(msg.Payload))
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
