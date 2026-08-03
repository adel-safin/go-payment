package app_test

import (
	"context"
	"sync"
	"testing"

	"github.com/adel-safin/go-payment/internal/transfer/app"
	"github.com/adel-safin/go-payment/internal/transfer/domain"
	pkgerrors "github.com/adel-safin/go-payment/pkg/errors"
	"github.com/adel-safin/go-payment/pkg/outbox"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type memRepo struct {
	mu   sync.Mutex
	byID map[uuid.UUID]domain.Transfer
	byKey map[string]uuid.UUID
	outbox []outbox.Event
}

func newRepo() *memRepo {
	return &memRepo{byID: map[uuid.UUID]domain.Transfer{}, byKey: map[string]uuid.UUID{}}
}

func (m *memRepo) GetByIdempotencyKey(_ context.Context, key string) (domain.Transfer, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	id, ok := m.byKey[key]
	if !ok {
		return domain.Transfer{}, false, nil
	}
	return m.byID[id], true, nil
}

func (m *memRepo) GetByID(_ context.Context, id uuid.UUID) (domain.Transfer, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	tr, ok := m.byID[id]
	if !ok {
		return domain.Transfer{}, pkgerrors.ErrNotFound
	}
	return tr, nil
}

func (m *memRepo) InsertCompletedWithOutbox(_ context.Context, tr domain.Transfer, ev outbox.Event) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.byKey[tr.IdempotencyKey]; ok {
		return pkgerrors.ErrAlreadyExists
	}
	m.byID[tr.ID] = tr
	m.byKey[tr.IdempotencyKey] = tr.ID
	m.outbox = append(m.outbox, ev)
	return nil
}

type memWallet struct {
	mu       sync.Mutex
	balances map[string]int64
}

func (w *memWallet) Debit(_ context.Context, walletID string, amount int64, _, _ string) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.balances[walletID] < amount {
		return pkgerrors.Wrap(pkgerrors.ErrInsufficientFunds, "debit")
	}
	w.balances[walletID] -= amount
	return nil
}

func (w *memWallet) Credit(_ context.Context, walletID string, amount int64, _, _ string) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.balances[walletID] += amount
	return nil
}

type memCache struct {
	mu sync.Mutex
	m  map[string]string
}

func (c *memCache) Get(_ context.Context, key string) (string, bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	v, ok := c.m[key]
	return v, ok, nil
}

func (c *memCache) Set(_ context.Context, key, transferID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.m[key] = transferID
	return nil
}

func TestCreateIdempotent(t *testing.T) {
	from, to := uuid.NewString(), uuid.NewString()
	wallet := &memWallet{balances: map[string]int64{from: 1000, to: 0}}
	repo := newRepo()
	cache := &memCache{m: map[string]string{}}
	svc := app.NewService(repo, wallet, cache)

	res, err := svc.Create(context.Background(), from, to, 200, "RUB", "idem-1", uuid.NewString())
	require.NoError(t, err)
	require.False(t, res.IdempotentReplay)
	require.Equal(t, domain.StatusCompleted, res.Transfer.Status)
	require.Len(t, repo.outbox, 1)

	res2, err := svc.Create(context.Background(), from, to, 200, "RUB", "idem-1", uuid.NewString())
	require.NoError(t, err)
	require.True(t, res2.IdempotentReplay)
	require.Equal(t, res.Transfer.ID, res2.Transfer.ID)
	require.Equal(t, int64(800), wallet.balances[from])
	require.Equal(t, int64(200), wallet.balances[to])
}
