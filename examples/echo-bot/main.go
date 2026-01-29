// Package main демонстрирует echo bot на базе goro.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"

	"github.com/udisondev/sprut/pkg/client"
	"github.com/udisondev/sprut/pkg/identity"
)

var msgCounter atomic.Int64

func main() {
	addr := flag.String("addr", "localhost:8443", "server address")
	keysPath := flag.String("keys", "", "path to keys file (will be generated if not exists)")
	insecure := flag.Bool("insecure", false, "skip TLS verification")
	caCert := flag.String("ca-cert", "", "path to CA certificate for server verification")
	flag.Parse()

	if *keysPath == "" {
		log.Fatal("keys path is required")
	}

	// Загружаем или генерируем ключи
	keys, err := identity.LoadOrGenerate(*keysPath)
	if err != nil {
		log.Fatalf("load keys: %v", err)
	}

	fmt.Printf("Echo bot public key: %s\n", keys.PublicKeyHex())

	// Настраиваем опции клиента
	opts := []client.ConnectOption{
		client.WithKeys(keys),
		client.WithOnError(func(err error) {
			fmt.Printf("Error: %v\n", err)
		}),
	}
	if *insecure {
		opts = append(opts, client.WithInsecureSkipVerify())
	}
	if *caCert != "" {
		opts = append(opts, client.WithCACertFile(*caCert))
	}

	// Создаём канал для отправки
	send := make(chan client.OutgoingMessage, 100)

	// Подключаемся к серверу
	recv, err := client.Connect(*addr, send, opts...)
	if err != nil {
		log.Fatalf("connect: %v", err)
	}

	fmt.Println("Echo bot is running. Press Ctrl+C to exit.")

	// Обрабатываем входящие сообщения
	go func() {
		for msg := range recv {
			fmt.Printf("[RECV] From %s: %s\n", msg.From[:16]+"...", string(msg.Payload))

			// Echo back
			reply := fmt.Sprintf("Echo: %s", string(msg.Payload))
			msgID := fmt.Sprintf("echo-%d", msgCounter.Add(1))

			send <- client.OutgoingMessage{
				To:      msg.From,
				MsgID:   msgID,
				Payload: []byte(reply),
			}
			fmt.Printf("[SENT] To %s: %s\n", msg.From[:16]+"...", reply)
		}
	}()

	// Ждём сигнала
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	fmt.Println("\nShutting down...")
	close(send)
}
