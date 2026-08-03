package app

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/adel-safin/go-payment/internal/notification/ports"
	"github.com/google/uuid"
)

type Service struct {
	store ports.Store
	log   *slog.Logger
}

func NewService(store ports.Store, log *slog.Logger) *Service {
	return &Service{store: store, log: log}
}

type TransferCompletedEvent struct {
	TransferID   string `json:"transfer_id"`
	FromWalletID string `json:"from_wallet_id"`
	ToWalletID   string `json:"to_wallet_id"`
	AmountMinor  int64  `json:"amount_minor"`
	Currency     string `json:"currency"`
	UserID       string `json:"user_id"`
	Status       string `json:"status"`
}

func (s *Service) Handle(ctx context.Context, eventID string, payload []byte) error {
	claimed, err := s.store.TryClaimEvent(ctx, eventID)
	if err != nil {
		return err
	}
	if !claimed {
		s.log.Info("duplicate event skipped", "event_id", eventID)
		return nil
	}

	var ev TransferCompletedEvent
	if err := json.Unmarshal(payload, &ev); err != nil {
		return err
	}

	n := ports.Notification{
		ID:         uuid.New(),
		EventID:    eventID,
		TransferID: ev.TransferID,
		Channel:    "log",
		Payload:    payload,
	}
	if err := s.store.SaveNotification(ctx, n); err != nil {
		return err
	}
	s.log.Info("notification delivered",
		"transfer_id", ev.TransferID,
		"user_id", ev.UserID,
		"amount_minor", ev.AmountMinor,
		"channel", "log",
	)
	return nil
}
