package app_test

import (
	"context"
	"sync"
	"testing"

	"github.com/adel-safin/go-payment/internal/wallet/app"
	"github.com/adel-safin/go-payment/internal/wallet/domain"
	pkgerrors "github.com/adel-safin/go-payment/pkg/errors"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type memRepo struct {
	mu      sync.Mutex
	wallets map[uuid.UUID]domain.Wallet
	bal     map[uuid.UUID]domain.Balance
	entries map[string]domain.LedgerEntry
}

func newMem() *memRepo {
	return &memRepo{
		wallets: map[uuid.UUID]domain.Wallet{},
		bal:     map[uuid.UUID]domain.Balance{},
		entries: map[string]domain.LedgerEntry{},
	}
}

func (m *memRepo) CreateWallet(_ context.Context, w domain.Wallet) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.wallets[w.ID] = w
	m.bal[w.ID] = domain.Balance{WalletID: w.ID}
	return nil
}

func (m *memRepo) GetWallet(_ context.Context, id uuid.UUID) (domain.Wallet, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	w, ok := m.wallets[id]
	if !ok {
		return domain.Wallet{}, pkgerrors.ErrNotFound
	}
	return w, nil
}

func (m *memRepo) GetBalance(_ context.Context, walletID uuid.UUID) (domain.Balance, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	b, ok := m.bal[walletID]
	if !ok {
		return domain.Balance{}, pkgerrors.ErrNotFound
	}
	return b, nil
}

func (m *memRepo) GetEntryByIdempotency(_ context.Context, key string) (domain.LedgerEntry, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	e, ok := m.entries[key]
	return e, ok, nil
}

func (m *memRepo) MutateBalance(_ context.Context, walletID uuid.UUID, expectedVersion int64, newBalance int64, entry domain.LedgerEntry) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	b, ok := m.bal[walletID]
	if !ok {
		return pkgerrors.ErrNotFound
	}
	if b.Version != expectedVersion {
		return domain.ErrVersionConflict
	}
	if _, exists := m.entries[entry.IdempotencyKey]; exists {
		return pkgerrors.ErrAlreadyExists
	}
	b.BalanceMinor = newBalance
	b.Version++
	m.bal[walletID] = b
	m.entries[entry.IdempotencyKey] = entry
	return nil
}

func TestCreditDebitIdempotent(t *testing.T) {
	svc := app.NewService(newMem())
	ctx := context.Background()
	w, err := svc.CreateWallet(ctx, uuid.NewString(), "RUB")
	require.NoError(t, err)

	tid := uuid.NewString()
	res, err := svc.Credit(ctx, w.ID.String(), 1000, tid, "credit-1")
	require.NoError(t, err)
	require.Equal(t, int64(1000), res.Balance.BalanceMinor)

	res2, err := svc.Credit(ctx, w.ID.String(), 1000, tid, "credit-1")
	require.NoError(t, err)
	require.Equal(t, res.EntryID, res2.EntryID)
	require.Equal(t, int64(1000), res2.Balance.BalanceMinor)

	_, err = svc.Debit(ctx, w.ID.String(), 400, uuid.NewString(), "debit-1")
	require.NoError(t, err)
	b, _, err := svc.GetBalance(ctx, w.ID.String())
	require.NoError(t, err)
	require.Equal(t, int64(600), b.BalanceMinor)

	_, err = svc.Debit(ctx, w.ID.String(), 9999, uuid.NewString(), "debit-2")
	require.ErrorIs(t, err, domain.ErrInsufficientFunds)
}
