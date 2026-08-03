package metrics

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
)

type TransferMetrics struct {
	created metric.Int64Counter
	failed  metric.Int64Counter
}

func NewTransferMetrics(service string) (*TransferMetrics, error) {
	meter := otel.Meter(service)
	created, err := meter.Int64Counter("transfer_created_total")
	if err != nil {
		return nil, fmt.Errorf("counter created: %w", err)
	}
	failed, err := meter.Int64Counter("transfer_failed_total")
	if err != nil {
		return nil, fmt.Errorf("counter failed: %w", err)
	}
	return &TransferMetrics{created: created, failed: failed}, nil
}

func (m *TransferMetrics) Created(ctx context.Context) {
	if m != nil {
		m.created.Add(ctx, 1)
	}
}

func (m *TransferMetrics) Failed(ctx context.Context) {
	if m != nil {
		m.failed.Add(ctx, 1)
	}
}
