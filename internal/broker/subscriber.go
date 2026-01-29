package broker

import (
	"fmt"

	"github.com/nats-io/nats.go"
)

// Subscriber управляет подпиской на сообщения для клиента.
type Subscriber struct {
	sub *nats.Subscription
}

// NewSubscriber создаёт подписчика для указанного публичного ключа.
func NewSubscriber(broker *Broker, pubKeyHex string, handler nats.MsgHandler) (*Subscriber, error) {
	subject := subjectForClient(pubKeyHex)
	sub, err := broker.conn.Subscribe(subject, handler)
	if err != nil {
		return nil, fmt.Errorf("subscribe to %s: %w", subject, err)
	}

	return &Subscriber{
		sub: sub,
	}, nil
}

// Unsubscribe отписывается от топика.
func (s *Subscriber) Unsubscribe() error {
	return s.sub.Unsubscribe()
}

// subjectForClient возвращает NATS subject для клиента.
func subjectForClient(pubKeyHex string) string {
	return "goro.msg." + pubKeyHex
}
