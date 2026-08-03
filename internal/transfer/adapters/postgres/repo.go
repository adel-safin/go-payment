package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/adel-safin/go-payment/internal/transfer/domain"
	pkgerrors "github.com/adel-safin/go-payment/pkg/errors"
	"github.com/adel-safin/go-payment/pkg/outbox"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repo struct {
	pool *pgxpool.Pool
}

func NewRepo(pool *pgxpool.Pool) *Repo {
	return &Repo{pool: pool}
}

func (r *Repo) GetByIdempotencyKey(ctx context.Context, key string) (domain.Transfer, bool, error) {
	tr, err := r.scan(ctx, `SELECT id, from_wallet_id, to_wallet_id, amount_minor, currency, status, user_id, idempotency_key, created_at, updated_at
		FROM transfers WHERE idempotency_key = $1`, key)
	if errors.Is(err, pkgerrors.ErrNotFound) {
		return domain.Transfer{}, false, nil
	}
	if err != nil {
		return domain.Transfer{}, false, err
	}
	return tr, true, nil
}

func (r *Repo) GetByID(ctx context.Context, id uuid.UUID) (domain.Transfer, error) {
	return r.scan(ctx, `SELECT id, from_wallet_id, to_wallet_id, amount_minor, currency, status, user_id, idempotency_key, created_at, updated_at
		FROM transfers WHERE id = $1`, id)
}

func (r *Repo) scan(ctx context.Context, q string, arg any) (domain.Transfer, error) {
	var tr domain.Transfer
	var status string
	err := r.pool.QueryRow(ctx, q, arg).Scan(
		&tr.ID, &tr.FromWalletID, &tr.ToWalletID, &tr.AmountMinor, &tr.Currency,
		&status, &tr.UserID, &tr.IdempotencyKey, &tr.CreatedAt, &tr.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Transfer{}, pkgerrors.ErrNotFound
	}
	if err != nil {
		return domain.Transfer{}, err
	}
	tr.Status = domain.Status(status)
	return tr, nil
}

func (r *Repo) InsertCompletedWithOutbox(ctx context.Context, tr domain.Transfer, ev outbox.Event) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
		INSERT INTO transfers (id, from_wallet_id, to_wallet_id, amount_minor, currency, status, user_id, idempotency_key, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		tr.ID, tr.FromWalletID, tr.ToWalletID, tr.AmountMinor, tr.Currency, string(tr.Status),
		tr.UserID, tr.IdempotencyKey, tr.CreatedAt, tr.UpdatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return pkgerrors.ErrAlreadyExists
		}
		return fmt.Errorf("insert transfer: %w", err)
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO outbox_events (id, aggregate_type, aggregate_id, event_type, payload, created_at)
		VALUES ($1,$2,$3,$4,$5,$6)`,
		ev.ID, ev.AggregateType, ev.AggregateID, ev.EventType, ev.Payload, ev.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert outbox: %w", err)
	}
	return tx.Commit(ctx)
}

// OutboxStore implements pkg/outbox.Store against the same pool.
type OutboxStore struct {
	pool *pgxpool.Pool
}

func NewOutboxStore(pool *pgxpool.Pool) *OutboxStore {
	return &OutboxStore{pool: pool}
}

func (s *OutboxStore) Insert(ctx context.Context, e outbox.Event) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO outbox_events (id, aggregate_type, aggregate_id, event_type, payload, created_at)
		VALUES ($1,$2,$3,$4,$5,$6)`,
		e.ID, e.AggregateType, e.AggregateID, e.EventType, e.Payload, e.CreatedAt)
	return err
}

func (s *OutboxStore) FetchUnpublished(ctx context.Context, limit int) ([]outbox.Event, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, aggregate_type, aggregate_id, event_type, payload, created_at, published_at
		FROM outbox_events
		WHERE published_at IS NULL
		ORDER BY created_at
		LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []outbox.Event
	for rows.Next() {
		var e outbox.Event
		if err := rows.Scan(&e.ID, &e.AggregateType, &e.AggregateID, &e.EventType, &e.Payload, &e.CreatedAt, &e.PublishedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (s *OutboxStore) MarkPublished(ctx context.Context, id uuid.UUID, at time.Time) error {
	_, err := s.pool.Exec(ctx, `UPDATE outbox_events SET published_at = $1 WHERE id = $2`, at, id)
	return err
}
