package broker

import (
	"fmt"
	"log/slog"
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
	slog.Debug("publisher: publishing", "subject", subject, "size", len(data))
	if err := p.broker.conn.Publish(subject, data); err != nil {
		slog.Error("publisher: failed", "subject", subject, "error", err)
		return fmt.Errorf("publish to %s: %w", subject, err)
	}
	return nil
}
