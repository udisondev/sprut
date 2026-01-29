package testsprut_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/udisondev/sprut/pkg/identity"
	"github.com/udisondev/sprut/pkg/message"
	"github.com/udisondev/sprut/pkg/testsprut"
)

func TestEnvironment_Start(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	env, err := testsprut.Start(ctx)
	require.NoError(t, err, "Start должен успешно завершиться")
	require.NotEmpty(t, env.NATSUrl, "NATSUrl должен быть заполнен")
	require.NotEmpty(t, env.SprutAddr, "SprutAddr должен быть заполнен")
	require.NotEmpty(t, env.CACert, "CACert должен быть заполнен")

	err = env.Close(ctx)
	require.NoError(t, err, "Close должен успешно завершиться")
}

func TestEnvironment_NewClient(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	env, err := testsprut.Start(ctx)
	require.NoError(t, err)
	defer env.Close(ctx)

	keys, err := identity.Generate()
	require.NoError(t, err, "Generate keys должен успешно завершиться")

	client, err := env.NewClient(ctx, keys)
	require.NoError(t, err, "NewClient должен успешно завершиться")
	require.NotNil(t, client)
	require.Equal(t, keys.PublicKeyHex(), client.PubKeyHex())

	client.Close()
}

func TestEnvironment_MessageExchange(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// 1. Запускаем окружение
	env, err := testsprut.Start(ctx)
	require.NoError(t, err)
	defer env.Close(ctx)

	// 2. Создаём Alice
	aliceKeys, err := identity.Generate()
	require.NoError(t, err)
	alice, err := env.NewClient(ctx, aliceKeys)
	require.NoError(t, err)
	defer alice.Close()

	// 3. Создаём Bob
	bobKeys, err := identity.Generate()
	require.NoError(t, err)
	bob, err := env.NewClient(ctx, bobKeys)
	require.NoError(t, err)
	defer bob.Close()

	// 4. Alice -> Bob
	alice.SendMessage(bob.PubKeyHex(), "msg-1", []byte("Hello Bob!"))

	msg := waitMessage(t, bob.Recv(), 10*time.Second)
	require.Equal(t, alice.PubKeyHex(), msg.From)
	require.Equal(t, "Hello Bob!", string(msg.Payload))

	// 5. Bob -> Alice
	bob.SendMessage(alice.PubKeyHex(), "msg-2", []byte("Hello Alice!"))

	msg = waitMessage(t, alice.Recv(), 10*time.Second)
	require.Equal(t, bob.PubKeyHex(), msg.From)
	require.Equal(t, "Hello Alice!", string(msg.Payload))
}

func TestEnvironment_WithOptions(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	env, err := testsprut.Start(ctx,
		testsprut.WithMaxConnections(50),
		testsprut.WithMaxMessageSize(32768),
		testsprut.WithRateLimit(500, 50),
		testsprut.WithServerID("custom-test-server"),
	)
	require.NoError(t, err)
	defer env.Close(ctx)

	// Проверяем что сервер работает с кастомными настройками
	keys, err := identity.Generate()
	require.NoError(t, err)

	client, err := env.NewClient(ctx, keys)
	require.NoError(t, err)
	client.Close()
}

func TestCerts_GenerateAndCleanup(t *testing.T) {
	certs, err := testsprut.GenerateCerts()
	require.NoError(t, err)
	require.NotEmpty(t, certs.CertFile)
	require.NotEmpty(t, certs.KeyFile)
	require.NotEmpty(t, certs.CACert)

	err = certs.Cleanup()
	require.NoError(t, err)
}

// waitMessage ожидает сообщение из канала с таймаутом.
func waitMessage(t *testing.T, ch <-chan *message.Message, timeout time.Duration) *message.Message {
	t.Helper()
	select {
	case msg := <-ch:
		return msg
	case <-time.After(timeout):
		t.Fatal("timeout waiting for message")
		return nil
	}
}

// Пример использования для документации.
func Example() {
	ctx := context.Background()

	// Запускаем тестовое окружение
	env, err := testsprut.Start(ctx)
	if err != nil {
		panic(err)
	}
	defer env.Close(ctx)

	// Создаём клиента
	keys, _ := identity.Generate()
	c, err := env.NewClient(ctx, keys)
	if err != nil {
		panic(err)
	}
	defer c.Close()

	// Отправляем сообщение (самому себе для примера)
	c.SendMessage(keys.PublicKeyHex(), "test-msg", []byte("Hello!"))
}
