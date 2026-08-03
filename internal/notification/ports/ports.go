package ports

import (
	"context"

	"github.com/google/uuid"
)

type Notification struct {
	ID         uuid.UUID
	EventID    string
	TransferID string
	Channel    string
	Payload    []byte
}

type Store interface {
	TryClaimEvent(ctx context.Context, eventID string) (claimed bool, err error)
	SaveNotification(ctx context.Context, n Notification) error
	CountNotifications(ctx context.Context) (int64, error)
}
