package ports

import (
	"context"

	"github.com/adel-safin/go-payment/internal/transfer/domain"
	"github.com/adel-safin/go-payment/pkg/outbox"
	"github.com/google/uuid"
)

type TransferRepository interface {
	GetByIdempotencyKey(ctx context.Context, key string) (domain.Transfer, bool, error)
	GetByID(ctx context.Context, id uuid.UUID) (domain.Transfer, error)
	InsertCompletedWithOutbox(ctx context.Context, tr domain.Transfer, ev outbox.Event) error
}

type WalletClient interface {
	Debit(ctx context.Context, walletID string, amount int64, transferID, idemKey string) error
	Credit(ctx context.Context, walletID string, amount int64, transferID, idemKey string) error
}

type IdempotencyCache interface {
	Get(ctx context.Context, key string) (string, bool, error)
	Set(ctx context.Context, key, transferID string) error
}
