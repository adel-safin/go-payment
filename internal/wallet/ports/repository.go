package ports

import (
	"context"

	"github.com/adel-safin/go-payment/internal/wallet/domain"
	"github.com/google/uuid"
)

type WalletRepository interface {
	CreateWallet(ctx context.Context, w domain.Wallet) error
	GetWallet(ctx context.Context, id uuid.UUID) (domain.Wallet, error)
	GetBalance(ctx context.Context, walletID uuid.UUID) (domain.Balance, error)
	GetEntryByIdempotency(ctx context.Context, key string) (domain.LedgerEntry, bool, error)
	MutateBalance(ctx context.Context, walletID uuid.UUID, expectedVersion int64, newBalance int64, entry domain.LedgerEntry) error
}
