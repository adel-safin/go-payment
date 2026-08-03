package app

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/adel-safin/go-payment/internal/wallet/domain"
	"github.com/adel-safin/go-payment/internal/wallet/ports"
	pkgerrors "github.com/adel-safin/go-payment/pkg/errors"
	"github.com/google/uuid"
)

type Service struct {
	repo ports.WalletRepository
}

func NewService(repo ports.WalletRepository) *Service {
	return &Service{repo: repo}
}

func (s *Service) CreateWallet(ctx context.Context, userID, currency string) (domain.Wallet, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return domain.Wallet{}, pkgerrors.ErrInvalidArgument
	}
	w, err := domain.NewWallet(uid, currency)
	if err != nil {
		return domain.Wallet{}, err
	}
	if err := s.repo.CreateWallet(ctx, w); err != nil {
		return domain.Wallet{}, err
	}
	return w, nil
}

func (s *Service) GetBalance(ctx context.Context, walletID string) (domain.Balance, domain.Wallet, error) {
	id, err := uuid.Parse(walletID)
	if err != nil {
		return domain.Balance{}, domain.Wallet{}, pkgerrors.ErrInvalidArgument
	}
	w, err := s.repo.GetWallet(ctx, id)
	if err != nil {
		return domain.Balance{}, domain.Wallet{}, err
	}
	b, err := s.repo.GetBalance(ctx, id)
	if err != nil {
		return domain.Balance{}, domain.Wallet{}, err
	}
	return b, w, nil
}

type MutateResult struct {
	Balance domain.Balance
	EntryID uuid.UUID
}

func (s *Service) Credit(ctx context.Context, walletID string, amount int64, transferID, idemKey string) (MutateResult, error) {
	return s.mutate(ctx, walletID, amount, transferID, idemKey, domain.EntryCredit)
}

func (s *Service) Debit(ctx context.Context, walletID string, amount int64, transferID, idemKey string) (MutateResult, error) {
	return s.mutate(ctx, walletID, amount, transferID, idemKey, domain.EntryDebit)
}

func (s *Service) mutate(ctx context.Context, walletID string, amount int64, transferID, idemKey string, typ domain.EntryType) (MutateResult, error) {
	if amount <= 0 {
		return MutateResult{}, domain.ErrInvalidAmount
	}
	wid, err := uuid.Parse(walletID)
	if err != nil {
		return MutateResult{}, pkgerrors.ErrInvalidArgument
	}
	tid, err := uuid.Parse(transferID)
	if err != nil {
		return MutateResult{}, pkgerrors.ErrInvalidArgument
	}
	if idemKey == "" {
		return MutateResult{}, pkgerrors.ErrInvalidArgument
	}

	if existing, ok, err := s.repo.GetEntryByIdempotency(ctx, idemKey); err != nil {
		return MutateResult{}, err
	} else if ok {
		b, err := s.repo.GetBalance(ctx, wid)
		if err != nil {
			return MutateResult{}, err
		}
		return MutateResult{Balance: b, EntryID: existing.ID}, nil
	}

	const maxAttempts = 8
	for attempt := 0; attempt < maxAttempts; attempt++ {
		b, err := s.repo.GetBalance(ctx, wid)
		if err != nil {
			return MutateResult{}, err
		}
		var next domain.Balance
		switch typ {
		case domain.EntryCredit:
			next, err = domain.ApplyCredit(b, amount)
		case domain.EntryDebit:
			next, err = domain.ApplyDebit(b, amount)
		default:
			return MutateResult{}, pkgerrors.ErrInvalidArgument
		}
		if err != nil {
			return MutateResult{}, err
		}
		entry := domain.LedgerEntry{
			ID:             uuid.New(),
			WalletID:       wid,
			TransferID:     tid,
			Type:           typ,
			AmountMinor:    amount,
			IdempotencyKey: idemKey,
			CreatedAt:      time.Now().UTC(),
		}
		err = s.repo.MutateBalance(ctx, wid, b.Version, next.BalanceMinor, entry)
		if err == nil {
			return MutateResult{Balance: next, EntryID: entry.ID}, nil
		}
		if errors.Is(err, domain.ErrVersionConflict) || errors.Is(err, pkgerrors.ErrVersionConflict) {
			continue
		}
		if errors.Is(err, pkgerrors.ErrAlreadyExists) {
			existing, ok, gerr := s.repo.GetEntryByIdempotency(ctx, idemKey)
			if gerr != nil {
				return MutateResult{}, gerr
			}
			if ok {
				bal, berr := s.repo.GetBalance(ctx, wid)
				if berr != nil {
					return MutateResult{}, berr
				}
				return MutateResult{Balance: bal, EntryID: existing.ID}, nil
			}
		}
		return MutateResult{}, err
	}
	return MutateResult{}, fmt.Errorf("%w: exhausted retries", domain.ErrVersionConflict)
}
