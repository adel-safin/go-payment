package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrInvalidTransfer = errors.New("invalid transfer")
	ErrSameWallet      = errors.New("from and to wallets must differ")
)

type Status string

const (
	StatusPending   Status = "pending"
	StatusCompleted Status = "completed"
	StatusFailed    Status = "failed"
)

type Transfer struct {
	ID             uuid.UUID
	FromWalletID   uuid.UUID
	ToWalletID     uuid.UUID
	AmountMinor    int64
	Currency       string
	Status         Status
	UserID         uuid.UUID
	IdempotencyKey string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func NewTransfer(from, to uuid.UUID, amount int64, currency, idemKey string, userID uuid.UUID) (Transfer, error) {
	if from == to {
		return Transfer{}, ErrSameWallet
	}
	if amount <= 0 {
		return Transfer{}, ErrInvalidTransfer
	}
	if currency == "" || idemKey == "" {
		return Transfer{}, ErrInvalidTransfer
	}
	now := time.Now().UTC()
	return Transfer{
		ID:             uuid.New(),
		FromWalletID:   from,
		ToWalletID:     to,
		AmountMinor:    amount,
		Currency:       currency,
		Status:         StatusPending,
		UserID:         userID,
		IdempotencyKey: idemKey,
		CreatedAt:      now,
		UpdatedAt:      now,
	}, nil
}
