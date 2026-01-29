package broker

import (
	"fmt"
	"log/slog"

	"github.com/nats-io/nats.go"
)

// Subscriber управляет подпиской на сообщения для клиента.
type Subscriber struct {
	sub *nats.Subscription
}

// NewSubscriber создаёт подписчика для указанного публичного ключа.
func NewSubscriber(broker *Broker, pubKeyHex string, handler nats.MsgHandler) (*Subscriber, error) {
	subject := subjectForClient(pubKeyHex)
	slog.Debug("subscriber: creating", "subject", subject)

	sub, err := broker.conn.Subscribe(subject, handler)
	if err != nil {
		slog.Error("subscriber: subscribe failed", "subject", subject, "error", err)
		return nil, fmt.Errorf("subscribe to %s: %w", subject, err)
	}

	slog.Info("subscriber: subscribed", "subject", subject)

	return &Subscriber{
		sub: sub,
	}, nil
}

// Unsubscribe отписывается от топика.
func (s *Subscriber) Unsubscribe() error {
	subject := s.sub.Subject
	slog.Debug("subscriber: unsubscribing", "subject", subject)
	if err := s.sub.Unsubscribe(); err != nil {
		slog.Error("subscriber: unsubscribe failed", "subject", subject, "error", err)
		return err
	}
	return nil
}

// subjectForClient возвращает NATS subject для клиента.
func subjectForClient(pubKeyHex string) string {
	return "goro.msg." + pubKeyHex
}
