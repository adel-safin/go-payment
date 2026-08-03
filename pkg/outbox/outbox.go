package outbox

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Event is a row in the transactional outbox.
type Event struct {
	ID            uuid.UUID
	AggregateType string
	AggregateID   string
	EventType     string
	Payload       []byte
	CreatedAt     time.Time
	PublishedAt   *time.Time
}

// Store persists outbox events inside the caller's transaction when possible.
type Store interface {
	Insert(ctx context.Context, e Event) error
	FetchUnpublished(ctx context.Context, limit int) ([]Event, error)
	MarkPublished(ctx context.Context, id uuid.UUID, at time.Time) error
}

// Publisher sends payloads to a broker.
type Publisher interface {
	Publish(ctx context.Context, key string, value []byte) error
}
