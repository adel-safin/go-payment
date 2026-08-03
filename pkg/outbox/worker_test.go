package outbox_test

import (
	"context"
	"testing"
	"time"

	"github.com/adel-safin/go-payment/pkg/outbox"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type memStore struct {
	events []outbox.Event
}

func (m *memStore) Insert(_ context.Context, e outbox.Event) error {
	m.events = append(m.events, e)
	return nil
}

func (m *memStore) FetchUnpublished(_ context.Context, limit int) ([]outbox.Event, error) {
	var out []outbox.Event
	for _, e := range m.events {
		if e.PublishedAt == nil {
			out = append(out, e)
			if len(out) >= limit {
				break
			}
		}
	}
	return out, nil
}

func (m *memStore) MarkPublished(_ context.Context, id uuid.UUID, at time.Time) error {
	for i := range m.events {
		if m.events[i].ID == id {
			m.events[i].PublishedAt = &at
		}
	}
	return nil
}

type memPub struct{ n int }

func (m *memPub) Publish(context.Context, string, []byte) error {
	m.n++
	return nil
}

func TestWorkerTick(t *testing.T) {
	store := &memStore{events: []outbox.Event{{
		ID: uuid.New(), AggregateID: "a", Payload: []byte(`{}`), CreatedAt: time.Now().UTC(),
	}}}
	pub := &memPub{}
	w := outbox.NewWorker(store, pub, nil)
	require.NoError(t, w.Tick(context.Background()))
	require.Equal(t, 1, pub.n)
	require.NoError(t, w.Tick(context.Background()))
	require.Equal(t, 1, pub.n)
}
