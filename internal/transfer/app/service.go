package app

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/adel-safin/go-payment/internal/transfer/domain"
	"github.com/adel-safin/go-payment/internal/transfer/ports"
	pkgerrors "github.com/adel-safin/go-payment/pkg/errors"
	"github.com/adel-safin/go-payment/pkg/outbox"
	"github.com/google/uuid"
)

type Service struct {
	repo   ports.TransferRepository
	wallet ports.WalletClient
	cache  ports.IdempotencyCache
}

func NewService(repo ports.TransferRepository, wallet ports.WalletClient, cache ports.IdempotencyCache) *Service {
	return &Service{repo: repo, wallet: wallet, cache: cache}
}

type CreateResult struct {
	Transfer         domain.Transfer
	IdempotentReplay bool
}

func (s *Service) Create(ctx context.Context, fromID, toID string, amount int64, currency, idemKey, userID string) (CreateResult, error) {
	if idemKey == "" {
		return CreateResult{}, pkgerrors.ErrInvalidArgument
	}
	if id, ok, err := s.cache.Get(ctx, idemKey); err != nil {
		return CreateResult{}, err
	} else if ok {
		tid, err := uuid.Parse(id)
		if err != nil {
			return CreateResult{}, err
		}
		tr, err := s.repo.GetByID(ctx, tid)
		if err != nil {
			return CreateResult{}, err
		}
		return CreateResult{Transfer: tr, IdempotentReplay: true}, nil
	}

	if existing, ok, err := s.repo.GetByIdempotencyKey(ctx, idemKey); err != nil {
		return CreateResult{}, err
	} else if ok {
		_ = s.cache.Set(ctx, idemKey, existing.ID.String())
		return CreateResult{Transfer: existing, IdempotentReplay: true}, nil
	}

	from, err := uuid.Parse(fromID)
	if err != nil {
		return CreateResult{}, pkgerrors.ErrInvalidArgument
	}
	to, err := uuid.Parse(toID)
	if err != nil {
		return CreateResult{}, pkgerrors.ErrInvalidArgument
	}
	uid, err := uuid.Parse(userID)
	if err != nil {
		return CreateResult{}, pkgerrors.ErrInvalidArgument
	}
	if currency == "" {
		currency = "RUB"
	}

	tr, err := domain.NewTransfer(from, to, amount, currency, idemKey, uid)
	if err != nil {
		return CreateResult{}, err
	}

	debitKey := fmt.Sprintf("transfer:%s:debit", tr.ID)
	creditKey := fmt.Sprintf("transfer:%s:credit", tr.ID)

	if err := s.wallet.Debit(ctx, fromID, amount, tr.ID.String(), debitKey); err != nil {
		return CreateResult{}, err
	}
	if err := s.wallet.Credit(ctx, toID, amount, tr.ID.String(), creditKey); err != nil {
		// compensate debit
		_ = s.wallet.Credit(ctx, fromID, amount, tr.ID.String(), debitKey+":compensate")
		return CreateResult{}, err
	}

	tr.Status = domain.StatusCompleted
	tr.UpdatedAt = time.Now().UTC()

	payload, _ := json.Marshal(map[string]any{
		"transfer_id":    tr.ID.String(),
		"from_wallet_id": tr.FromWalletID.String(),
		"to_wallet_id":   tr.ToWalletID.String(),
		"amount_minor":   tr.AmountMinor,
		"currency":       tr.Currency,
		"user_id":        tr.UserID.String(),
		"status":         string(tr.Status),
	})
	ev := outbox.Event{
		ID:            uuid.New(),
		AggregateType: "transfer",
		AggregateID:   tr.ID.String(),
		EventType:     "transfer.completed",
		Payload:       payload,
		CreatedAt:     time.Now().UTC(),
	}
	if err := s.repo.InsertCompletedWithOutbox(ctx, tr, ev); err != nil {
		return CreateResult{}, err
	}
	_ = s.cache.Set(ctx, idemKey, tr.ID.String())
	return CreateResult{Transfer: tr}, nil
}

func (s *Service) Get(ctx context.Context, id string) (domain.Transfer, error) {
	tid, err := uuid.Parse(id)
	if err != nil {
		return domain.Transfer{}, pkgerrors.ErrInvalidArgument
	}
	return s.repo.GetByID(ctx, tid)
}
