package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/adel-safin/go-payment/internal/wallet/domain"
	pkgerrors "github.com/adel-safin/go-payment/pkg/errors"
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

func (r *Repo) CreateWallet(ctx context.Context, w domain.Wallet) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
		INSERT INTO wallets (id, user_id, currency, created_at)
		VALUES ($1, $2, $3, $4)`, w.ID, w.UserID, w.Currency, w.CreatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return pkgerrors.ErrAlreadyExists
		}
		return fmt.Errorf("insert wallet: %w", err)
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO account_balances (wallet_id, balance_minor, version)
		VALUES ($1, 0, 0)`, w.ID)
	if err != nil {
		return fmt.Errorf("insert balance: %w", err)
	}
	return tx.Commit(ctx)
}

func (r *Repo) GetWallet(ctx context.Context, id uuid.UUID) (domain.Wallet, error) {
	var w domain.Wallet
	err := r.pool.QueryRow(ctx, `
		SELECT id, user_id, currency, created_at FROM wallets WHERE id = $1`, id).
		Scan(&w.ID, &w.UserID, &w.Currency, &w.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Wallet{}, pkgerrors.ErrNotFound
	}
	if err != nil {
		return domain.Wallet{}, err
	}
	return w, nil
}

func (r *Repo) GetBalance(ctx context.Context, walletID uuid.UUID) (domain.Balance, error) {
	var b domain.Balance
	err := r.pool.QueryRow(ctx, `
		SELECT wallet_id, balance_minor, version FROM account_balances WHERE wallet_id = $1`, walletID).
		Scan(&b.WalletID, &b.BalanceMinor, &b.Version)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Balance{}, pkgerrors.ErrNotFound
	}
	if err != nil {
		return domain.Balance{}, err
	}
	return b, nil
}

func (r *Repo) GetEntryByIdempotency(ctx context.Context, key string) (domain.LedgerEntry, bool, error) {
	var e domain.LedgerEntry
	var typ string
	err := r.pool.QueryRow(ctx, `
		SELECT id, wallet_id, transfer_id, entry_type, amount_minor, idempotency_key, created_at
		FROM ledger_entries WHERE idempotency_key = $1`, key).
		Scan(&e.ID, &e.WalletID, &e.TransferID, &typ, &e.AmountMinor, &e.IdempotencyKey, &e.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.LedgerEntry{}, false, nil
	}
	if err != nil {
		return domain.LedgerEntry{}, false, err
	}
	e.Type = domain.EntryType(typ)
	return e, true, nil
}

func (r *Repo) MutateBalance(ctx context.Context, walletID uuid.UUID, expectedVersion int64, newBalance int64, entry domain.LedgerEntry) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	tag, err := tx.Exec(ctx, `
		UPDATE account_balances
		SET balance_minor = $1, version = version + 1
		WHERE wallet_id = $2 AND version = $3`,
		newBalance, walletID, expectedVersion)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrVersionConflict
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO ledger_entries (id, wallet_id, transfer_id, entry_type, amount_minor, idempotency_key, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		entry.ID, entry.WalletID, entry.TransferID, string(entry.Type), entry.AmountMinor, entry.IdempotencyKey, entry.CreatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return pkgerrors.ErrAlreadyExists
		}
		return err
	}
	return tx.Commit(ctx)
}
