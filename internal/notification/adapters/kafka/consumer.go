package consumer

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/adel-safin/go-payment/internal/notification/app"
	pkgkafka "github.com/adel-safin/go-payment/pkg/kafka"
)

type Runner struct {
	reader *pkgkafka.Reader
	svc    *app.Service
	log    *slog.Logger
}

func New(reader *pkgkafka.Reader, svc *app.Service, log *slog.Logger) *Runner {
	return &Runner{reader: reader, svc: svc, log: log}
}

func (r *Runner) Run(ctx context.Context) error {
	for {
		msg, msgCtx, err := r.reader.Fetch(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return fmt.Errorf("fetch: %w", err)
		}
		eventID := string(msg.Key)
		if eventID == "" {
			eventID = fmt.Sprintf("%s-%d-%d", msg.Topic, msg.Partition, msg.Offset)
		}
		if err := r.svc.Handle(msgCtx, eventID, msg.Value); err != nil {
			r.log.Error("handle failed", "err", err, "event_id", eventID)
			continue
		}
		if err := r.reader.Commit(ctx, msg); err != nil {
			return fmt.Errorf("commit: %w", err)
		}
	}
}
