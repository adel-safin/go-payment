//go:build integration

package postgres_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"

	walletmigrations "github.com/adel-safin/go-payment/db/migrations/wallet"
	"github.com/adel-safin/go-payment/internal/wallet/adapters/postgres"
	"github.com/adel-safin/go-payment/internal/wallet/app"
	"github.com/adel-safin/go-payment/pkg/migrate"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestConcurrentDebit(t *testing.T) {
	ctx := context.Background()
	pg, err := tcpostgres.Run(ctx,
		"postgres:16-alpine",
		tcpostgres.WithDatabase("wallet"),
		tcpostgres.WithUsername("payment"),
		tcpostgres.WithPassword("payment"),
		testcontainers.WithWaitStrategy(wait.ForListeningPort("5432/tcp")),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = pg.Terminate(ctx) })

	url, err := pg.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)
	require.NoError(t, migrate.UpFS(url, walletmigrations.FS, "."))

	pool, err := pgxpool.New(ctx, url)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	svc := app.NewService(postgres.NewRepo(pool))
	w, err := svc.CreateWallet(ctx, uuid.NewString(), "RUB")
	require.NoError(t, err)
	_, err = svc.Credit(ctx, w.ID.String(), 1000, uuid.NewString(), "seed")
	require.NoError(t, err)

	var okCount atomic.Int64
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, err := svc.Debit(ctx, w.ID.String(), 100, uuid.NewString(), uuid.NewString())
			if err == nil {
				okCount.Add(1)
			}
		}(i)
	}
	wg.Wait()
	require.Equal(t, int64(10), okCount.Load())
	b, _, err := svc.GetBalance(ctx, w.ID.String())
	require.NoError(t, err)
	require.Equal(t, int64(0), b.BalanceMinor)
}
