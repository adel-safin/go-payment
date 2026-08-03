//go:build integration

package postgres_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	transfermigrations "github.com/adel-safin/go-payment/db/migrations/transfer"
	"github.com/adel-safin/go-payment/internal/transfer/adapters/postgres"
	"github.com/adel-safin/go-payment/internal/transfer/domain"
	"github.com/adel-safin/go-payment/pkg/migrate"
	"github.com/adel-safin/go-payment/pkg/outbox"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

type memPublisher struct {
	msgs [][]byte
}

func (m *memPublisher) Publish(_ context.Context, _ string, value []byte) error {
	m.msgs = append(m.msgs, append([]byte(nil), value...))
	return nil
}

func TestOutboxRecovery(t *testing.T) {
	ctx := context.Background()
	pg, err := tcpostgres.Run(ctx,
		"postgres:16-alpine",
		tcpostgres.WithDatabase("transfer"),
		tcpostgres.WithUsername("payment"),
		tcpostgres.WithPassword("payment"),
		testcontainers.WithWaitStrategy(wait.ForListeningPort("5432/tcp")),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = pg.Terminate(ctx) })

	url, err := pg.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)
	require.NoError(t, migrate.UpFS(url, transfermigrations.FS, "."))

	pool, err := pgxpool.New(ctx, url)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	repo := postgres.NewRepo(pool)
	store := postgres.NewOutboxStore(pool)

	tr := domain.Transfer{
		ID: uuid.New(), FromWalletID: uuid.New(), ToWalletID: uuid.New(),
		AmountMinor: 100, Currency: "RUB", Status: domain.StatusCompleted,
		UserID: uuid.New(), IdempotencyKey: uuid.NewString(),
		CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	}
	payload, _ := json.Marshal(map[string]string{"transfer_id": tr.ID.String()})
	ev := outbox.Event{
		ID: uuid.New(), AggregateType: "transfer", AggregateID: tr.ID.String(),
		EventType: "transfer.completed", Payload: payload, CreatedAt: time.Now().UTC(),
	}
	require.NoError(t, repo.InsertCompletedWithOutbox(ctx, tr, ev))

	// Simulate crash before publish: event is in DB unpublished.
	unpublished, err := store.FetchUnpublished(ctx, 10)
	require.NoError(t, err)
	require.Len(t, unpublished, 1)

	pub := &memPublisher{}
	for _, e := range unpublished {
		require.NoError(t, pub.Publish(ctx, e.AggregateID, e.Payload))
		require.NoError(t, store.MarkPublished(ctx, e.ID, time.Now().UTC()))
	}
	require.Len(t, pub.msgs, 1)

	again, err := store.FetchUnpublished(ctx, 10)
	require.NoError(t, err)
	require.Empty(t, again)
}
