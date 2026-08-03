package main

import (
	"context"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	transferv1 "github.com/adel-safin/go-payment/api/gen/transfer/v1"
	walletv1 "github.com/adel-safin/go-payment/api/gen/wallet/v1"
	transfermigrations "github.com/adel-safin/go-payment/db/migrations/transfer"
	grpcadapter "github.com/adel-safin/go-payment/internal/transfer/adapters/grpc"
	"github.com/adel-safin/go-payment/internal/transfer/adapters/postgres"
	rediscache "github.com/adel-safin/go-payment/internal/transfer/adapters/redis"
	"github.com/adel-safin/go-payment/internal/transfer/adapters/walletclient"
	"github.com/adel-safin/go-payment/internal/transfer/app"
	"github.com/adel-safin/go-payment/pkg/config"
	"github.com/adel-safin/go-payment/pkg/health"
	pkgkafka "github.com/adel-safin/go-payment/pkg/kafka"
	"github.com/adel-safin/go-payment/pkg/logger"
	"github.com/adel-safin/go-payment/pkg/migrate"
	otelx "github.com/adel-safin/go-payment/pkg/otel"
	"github.com/adel-safin/go-payment/pkg/outbox"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	cfg := config.Load("transfer")
	log := logger.New(cfg.LogLevel)
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	shutdownTelemetry, err := otelx.Setup(ctx, cfg.ServiceName, cfg.OTLPEndpoint)
	if err != nil {
		log.Warn("otel setup failed, continuing without telemetry", "err", err)
		shutdownTelemetry = func(context.Context) error { return nil }
	}

	if err := migrate.UpFS(cfg.DatabaseURL, transfermigrations.FS, "."); err != nil {
		log.Error("migrate failed", "err", err)
		os.Exit(1)
	}

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Error("db connect failed", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	wConn, err := grpc.NewClient(cfg.WalletGRPCAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
	)
	if err != nil {
		log.Error("dial wallet", "err", err)
		os.Exit(1)
	}
	defer wConn.Close()

	cache := rediscache.New(cfg.RedisAddr, cfg.RedisPassword)
	svc := app.NewService(
		postgres.NewRepo(pool),
		walletclient.New(walletv1.NewWalletServiceClient(wConn)),
		cache,
	)

	server := grpc.NewServer(grpc.StatsHandler(otelgrpc.NewServerHandler()))
	transferv1.RegisterTransferServiceServer(server, grpcadapter.NewServer(svc))

	writer := pkgkafka.NewWriter(cfg.KafkaBrokers, cfg.KafkaTopic)
	defer writer.Close()
	worker := outbox.NewWorker(postgres.NewOutboxStore(pool), writer, log)
	go worker.Run(ctx)

	h := health.New()
	h.Register("postgres", func(c context.Context) error { return pool.Ping(c) })
	h.Register("redis", func(c context.Context) error { return cache.Client().Ping(c).Err() })
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", h.Healthz())
	mux.HandleFunc("/readyz", h.Readyz())
	httpSrv := &http.Server{Addr: cfg.HTTPAddr, Handler: mux}

	go func() {
		log.Info("transfer http listening", "addr", cfg.HTTPAddr)
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
		log.Info("transfer grpc listening", "addr", cfg.GRPCAddr)
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
	log.Info("transfer stopped")
}
