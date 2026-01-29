package broker

import (
	"fmt"
)

// Publisher публикует сообщения в NATS.
type Publisher struct {
	broker *Broker
}

// NewPublisher создаёт издателя.
func NewPublisher(broker *Broker) *Publisher {
	return &Publisher{broker: broker}
}

// Publish публикует сообщение для указанного получателя.
func (p *Publisher) Publish(toPubKeyHex string, data []byte) error {
	subject := subjectForClient(toPubKeyHex)
	if err := p.broker.conn.Publish(subject, data); err != nil {
		return fmt.Errorf("publish to %s: %w", subject, err)
	}
	return nil
}
