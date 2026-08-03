package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	notificationmigrations "github.com/adel-safin/go-payment/db/migrations/notification"
	kafkaconsumer "github.com/adel-safin/go-payment/internal/notification/adapters/kafka"
	"github.com/adel-safin/go-payment/internal/notification/adapters/postgres"
	"github.com/adel-safin/go-payment/internal/notification/app"
	"github.com/adel-safin/go-payment/pkg/config"
	"github.com/adel-safin/go-payment/pkg/health"
	pkgkafka "github.com/adel-safin/go-payment/pkg/kafka"
	"github.com/adel-safin/go-payment/pkg/logger"
	"github.com/adel-safin/go-payment/pkg/migrate"
	otelx "github.com/adel-safin/go-payment/pkg/otel"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	cfg := config.Load("notification")
	log := logger.New(cfg.LogLevel)
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	shutdownTelemetry, err := otelx.Setup(ctx, cfg.ServiceName, cfg.OTLPEndpoint)
	if err != nil {
		log.Warn("otel setup failed, continuing without telemetry", "err", err)
		shutdownTelemetry = func(context.Context) error { return nil }
	}

	if err := migrate.UpFS(cfg.DatabaseURL, notificationmigrations.FS, "."); err != nil {
		log.Error("migrate failed", "err", err)
		os.Exit(1)
	}

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Error("db connect failed", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	svc := app.NewService(postgres.NewStore(pool), log)
	reader := pkgkafka.NewReader(cfg.KafkaBrokers, cfg.KafkaTopic, "notification-service")
	defer reader.Close()
	runner := kafkaconsumer.New(reader, svc, log)

	h := health.New()
	h.Register("postgres", func(c context.Context) error { return pool.Ping(c) })
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", h.Healthz())
	mux.HandleFunc("/readyz", h.Readyz())
	httpSrv := &http.Server{Addr: cfg.HTTPAddr, Handler: mux}

	go func() {
		log.Info("notification http listening", "addr", cfg.HTTPAddr)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("http failed", "err", err)
			stop()
		}
	}()

	go func() {
		log.Info("notification consumer started", "topic", cfg.KafkaTopic)
		if err := runner.Run(ctx); err != nil {
			log.Error("consumer failed", "err", err)
			stop()
		}
	}()

	<-ctx.Done()
	shCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = httpSrv.Shutdown(shCtx)
	_ = shutdownTelemetry(shCtx)
	log.Info("notification stopped")
}
