package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/adel-safin/go-payment/internal/notification/ports"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func (s *Store) TryClaimEvent(ctx context.Context, eventID string) (bool, error) {
	_, err := s.pool.Exec(ctx, `INSERT INTO processed_events (event_id) VALUES ($1)`, eventID)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return false, nil
		}
		return false, fmt.Errorf("claim event: %w", err)
	}
	return true, nil
}

func (s *Store) SaveNotification(ctx context.Context, n ports.Notification) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO notifications (id, event_id, transfer_id, channel, payload)
		VALUES ($1, $2, $3, $4, $5)`,
		n.ID, n.EventID, n.TransferID, n.Channel, n.Payload,
	)
	return err
}

func (s *Store) CountNotifications(ctx context.Context) (int64, error) {
	var n int64
	err := s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM notifications`).Scan(&n)
	return n, err
}
