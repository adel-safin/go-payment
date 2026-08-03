//go:build integration

package postgres_test

import (
	"context"
	"testing"
	"time"

	identitymigrations "github.com/adel-safin/go-payment/db/migrations/identity"
	"github.com/adel-safin/go-payment/internal/identity/adapters/postgres"
	"github.com/adel-safin/go-payment/internal/identity/app"
	pkgauth "github.com/adel-safin/go-payment/pkg/auth"
	"github.com/adel-safin/go-payment/pkg/migrate"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestRegisterLoginIntegration(t *testing.T) {
	ctx := context.Background()
	pg, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("identity"),
		postgres.WithUsername("payment"),
		postgres.WithPassword("payment"),
		testcontainers.WithWaitStrategy(wait.ForListeningPort("5432/tcp")),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = pg.Terminate(ctx) })

	url, err := pg.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)
	require.NoError(t, migrate.UpFS(url, identitymigrations.FS, "."))

	pool, err := pgxpool.New(ctx, url)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	svc := app.NewService(postgres.NewUserRepo(pool), pkgauth.NewTokenManager("s", "iss", time.Hour))
	_, err = svc.Register(ctx, "int@example.com", "password1")
	require.NoError(t, err)
	login, err := svc.Login(ctx, "int@example.com", "password1")
	require.NoError(t, err)
	require.NotEmpty(t, login.Token)
}
