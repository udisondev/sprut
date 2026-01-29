// Package main демонстрирует простое использование клиентской библиотеки goro.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
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

	fmt.Printf("My public key: %s\n", keys.PublicKeyHex())

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

	fmt.Println("Connected! Press Ctrl+C to exit.")
	fmt.Println("Usage: <recipient_pubkey_hex> <message>")
	fmt.Println()

	// Обрабатываем входящие сообщения
	go func() {
		for msg := range recv {
			fmt.Printf("[%s] From %s: %s\n", msg.Id, msg.From[:16]+"...", string(msg.Payload))
		}
	}()

	// Запускаем чтение команд в отдельной горутине
	go readCommands(send)

	// Ждём сигнала
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	fmt.Println("\nShutting down...")
	close(send)
}

func readCommands(send chan<- client.OutgoingMessage) {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, " ", 2)
		if len(parts) < 2 {
			fmt.Println("Usage: <recipient_pubkey_hex> <message>")
			continue
		}

		to := parts[0]
		msg := parts[1]

		if len(to) != 64 {
			fmt.Println("Invalid recipient pubkey (must be 64 hex chars)")
			continue
		}

		send <- client.OutgoingMessage{
			To:      to,
			MsgID:   fmt.Sprintf("msg-%d", msgCounter.Add(1)),
			Payload: []byte(msg),
		}
	}
}
