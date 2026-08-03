package app_test

import (
	"context"
	"log/slog"
	"os"
	"sync"
	"testing"

	"github.com/adel-safin/go-payment/internal/notification/app"
	"github.com/adel-safin/go-payment/internal/notification/ports"
	"github.com/stretchr/testify/require"
)

type memStore struct {
	mu   sync.Mutex
	seen map[string]struct{}
	notes []ports.Notification
}

func (m *memStore) TryClaimEvent(_ context.Context, eventID string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.seen[eventID]; ok {
		return false, nil
	}
	m.seen[eventID] = struct{}{}
	return true, nil
}

func (m *memStore) SaveNotification(_ context.Context, n ports.Notification) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.notes = append(m.notes, n)
	return nil
}

func (m *memStore) CountNotifications(_ context.Context) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return int64(len(m.notes)), nil
}

func TestHandleIdempotent(t *testing.T) {
	store := &memStore{seen: map[string]struct{}{}}
	svc := app.NewService(store, slog.New(slog.NewTextHandler(os.Stdout, nil)))
	payload := []byte(`{"transfer_id":"t1","user_id":"u1","amount_minor":100,"currency":"RUB","status":"completed"}`)

	require.NoError(t, svc.Handle(context.Background(), "e1", payload))
	require.NoError(t, svc.Handle(context.Background(), "e1", payload))
	n, err := store.CountNotifications(context.Background())
	require.NoError(t, err)
	require.Equal(t, int64(1), n)
}
