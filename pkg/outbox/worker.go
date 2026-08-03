package outbox

import (
	"context"
	"log/slog"
	"time"
)

// Worker polls unpublished events and publishes them.
type Worker struct {
	store     Store
	publisher Publisher
	log       *slog.Logger
	interval  time.Duration
	batch     int
}

func NewWorker(store Store, publisher Publisher, log *slog.Logger) *Worker {
	return &Worker{
		store:     store,
		publisher: publisher,
		log:       log,
		interval:  time.Second,
		batch:     50,
	}
}

func (w *Worker) Run(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()
	for {
		if err := w.Tick(ctx); err != nil && ctx.Err() == nil {
			if w.log != nil {
				w.log.Error("outbox tick failed", "err", err)
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// Tick processes one batch of unpublished events.
func (w *Worker) Tick(ctx context.Context) error {
	events, err := w.store.FetchUnpublished(ctx, w.batch)
	if err != nil {
		return err
	}
	for _, e := range events {
		if err := w.publisher.Publish(ctx, e.AggregateID, e.Payload); err != nil {
			return err
		}
		now := time.Now().UTC()
		if err := w.store.MarkPublished(ctx, e.ID, now); err != nil {
			return err
		}
		if w.log != nil {
			w.log.Info("outbox published", "id", e.ID, "type", e.EventType)
		}
	}
	return nil
}
