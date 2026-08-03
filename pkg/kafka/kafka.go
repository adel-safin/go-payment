package kafka

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/segmentio/kafka-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

// Writer wraps kafka-go writer with W3C trace injection into headers.
type Writer struct {
	w *kafka.Writer
}

func NewWriter(brokers, topic string) *Writer {
	return &Writer{w: &kafka.Writer{
		Addr:         kafka.TCP(strings.Split(brokers, ",")...),
		Topic:        topic,
		Balancer:     &kafka.LeastBytes{},
		RequiredAcks: kafka.RequireOne,
		Async:        false,
	}}
}

func (w *Writer) Publish(ctx context.Context, key string, value []byte) error {
	carrier := propagation.MapCarrier{}
	otel.GetTextMapPropagator().Inject(ctx, carrier)
	headers := make([]kafka.Header, 0, len(carrier))
	for k, v := range carrier {
		headers = append(headers, kafka.Header{Key: k, Value: []byte(v)})
	}
	err := w.w.WriteMessages(ctx, kafka.Message{
		Key:     []byte(key),
		Value:   value,
		Headers: headers,
		Time:    time.Now().UTC(),
	})
	if err != nil {
		return fmt.Errorf("kafka publish: %w", err)
	}
	return nil
}

func (w *Writer) Close() error { return w.w.Close() }

// Reader wraps kafka-go reader with W3C extraction.
type Reader struct {
	r *kafka.Reader
}

func NewReader(brokers, topic, group string) *Reader {
	return &Reader{r: kafka.NewReader(kafka.ReaderConfig{
		Brokers:        strings.Split(brokers, ","),
		Topic:          topic,
		GroupID:        group,
		MinBytes:       1,
		MaxBytes:       10e6,
		CommitInterval: time.Second,
		StartOffset:    kafka.FirstOffset,
	})}
}

func (r *Reader) Fetch(ctx context.Context) (kafka.Message, context.Context, error) {
	msg, err := r.r.FetchMessage(ctx)
	if err != nil {
		return kafka.Message{}, ctx, err
	}
	carrier := propagation.MapCarrier{}
	for _, h := range msg.Headers {
		carrier[h.Key] = string(h.Value)
	}
	ctx = otel.GetTextMapPropagator().Extract(ctx, carrier)
	return msg, ctx, nil
}

func (r *Reader) Commit(ctx context.Context, msg kafka.Message) error {
	return r.r.CommitMessages(ctx, msg)
}

func (r *Reader) Close() error { return r.r.Close() }
