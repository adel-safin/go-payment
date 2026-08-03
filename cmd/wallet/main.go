package main

import (
	"context"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	walletv1 "github.com/adel-safin/go-payment/api/gen/wallet/v1"
	walletmigrations "github.com/adel-safin/go-payment/db/migrations/wallet"
	grpcadapter "github.com/adel-safin/go-payment/internal/wallet/adapters/grpc"
	"github.com/adel-safin/go-payment/internal/wallet/adapters/postgres"
	"github.com/adel-safin/go-payment/internal/wallet/app"
	"github.com/adel-safin/go-payment/pkg/config"
	"github.com/adel-safin/go-payment/pkg/health"
	"github.com/adel-safin/go-payment/pkg/logger"
	"github.com/adel-safin/go-payment/pkg/migrate"
	otelx "github.com/adel-safin/go-payment/pkg/otel"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
)

func main() {
	cfg := config.Load("wallet")
	log := logger.New(cfg.LogLevel)
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	shutdownTelemetry, err := otelx.Setup(ctx, cfg.ServiceName, cfg.OTLPEndpoint)
	if err != nil {
		log.Warn("otel setup failed, continuing without telemetry", "err", err)
		shutdownTelemetry = func(context.Context) error { return nil }
	}

	if err := migrate.UpFS(cfg.DatabaseURL, walletmigrations.FS, "."); err != nil {
		log.Error("migrate failed", "err", err)
		os.Exit(1)
	}

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Error("db connect failed", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	svc := app.NewService(postgres.NewRepo(pool))
	server := grpc.NewServer(grpc.StatsHandler(otelgrpc.NewServerHandler()))
	walletv1.RegisterWalletServiceServer(server, grpcadapter.NewServer(svc))

	h := health.New()
	h.Register("postgres", func(c context.Context) error { return pool.Ping(c) })
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", h.Healthz())
	mux.HandleFunc("/readyz", h.Readyz())
	httpSrv := &http.Server{Addr: cfg.HTTPAddr, Handler: mux}

	go func() {
		log.Info("wallet http listening", "addr", cfg.HTTPAddr)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("http failed", "err", err)
			stop()
		}
	}()

	lis, err := net.Listen("tcp", cfg.GRPCAddr)
	if err != nil {
		log.Error("listen failed", "err", err)
		os.Exit(1)
	}
	go func() {
		log.Info("wallet grpc listening", "addr", cfg.GRPCAddr)
		if err := server.Serve(lis); err != nil {
			log.Error("grpc failed", "err", err)
			stop()
		}
	}()

	<-ctx.Done()
	server.GracefulStop()
	shCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = httpSrv.Shutdown(shCtx)
	_ = shutdownTelemetry(shCtx)
	log.Info("wallet stopped")
}
