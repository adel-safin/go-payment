package kafka

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

// ReliablePublisher retries publish and sends to DLQ after exhausting attempts.
type ReliablePublisher struct {
	primary *Writer
	dlq     *Writer
	log     *slog.Logger
	retries int
}

func NewReliablePublisher(brokers, topic, dlqTopic string, log *slog.Logger) *ReliablePublisher {
	return &ReliablePublisher{
		primary: NewWriter(brokers, topic),
		dlq:     NewWriter(brokers, dlqTopic),
		log:     log,
		retries: 3,
	}
}

func (p *ReliablePublisher) Publish(ctx context.Context, key string, value []byte) error {
	var err error
	backoff := 100 * time.Millisecond
	for i := 0; i < p.retries; i++ {
		err = p.primary.Publish(ctx, key, value)
		if err == nil {
			return nil
		}
		p.log.Warn("kafka publish retry", "attempt", i+1, "err", err)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff):
			backoff *= 2
		}
	}
	if dlqErr := p.dlq.Publish(ctx, key, value); dlqErr != nil {
		return fmt.Errorf("publish failed and dlq failed: primary=%v dlq=%w", err, dlqErr)
	}
	p.log.Error("message sent to DLQ", "key", key, "err", err)
	return nil
}

func (p *ReliablePublisher) Close() error {
	_ = p.primary.Close()
	return p.dlq.Close()
}
