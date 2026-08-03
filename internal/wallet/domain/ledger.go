package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrInvalidAmount     = errors.New("amount must be positive")
	ErrInsufficientFunds = errors.New("insufficient funds")
	ErrInvalidCurrency   = errors.New("invalid currency")
	ErrVersionConflict   = errors.New("version conflict")
)

type EntryType string

const (
	EntryDebit  EntryType = "debit"
	EntryCredit EntryType = "credit"
)

type Wallet struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	Currency  string
	CreatedAt time.Time
}

type Balance struct {
	WalletID     uuid.UUID
	BalanceMinor int64
	Version      int64
}

type LedgerEntry struct {
	ID             uuid.UUID
	WalletID       uuid.UUID
	TransferID     uuid.UUID
	Type           EntryType
	AmountMinor    int64
	IdempotencyKey string
	CreatedAt      time.Time
}

func NewWallet(userID uuid.UUID, currency string) (Wallet, error) {
	if currency == "" || len(currency) != 3 {
		return Wallet{}, ErrInvalidCurrency
	}
	return Wallet{
		ID:        uuid.New(),
		UserID:    userID,
		Currency:  currency,
		CreatedAt: time.Now().UTC(),
	}, nil
}

func ApplyCredit(b Balance, amount int64) (Balance, error) {
	if amount <= 0 {
		return b, ErrInvalidAmount
	}
	b.BalanceMinor += amount
	b.Version++
	return b, nil
}

func ApplyDebit(b Balance, amount int64) (Balance, error) {
	if amount <= 0 {
		return b, ErrInvalidAmount
	}
	if b.BalanceMinor < amount {
		return b, ErrInsufficientFunds
	}
	b.BalanceMinor -= amount
	b.Version++
	return b, nil
}

// BalancedPair validates double-entry equality for a transfer.
func BalancedPair(debit, credit int64) bool {
	return debit > 0 && debit == credit
}
