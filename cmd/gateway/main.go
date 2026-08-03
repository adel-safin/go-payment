package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	identityv1 "github.com/adel-safin/go-payment/api/gen/identity/v1"
	transferv1 "github.com/adel-safin/go-payment/api/gen/transfer/v1"
	walletv1 "github.com/adel-safin/go-payment/api/gen/wallet/v1"
	"github.com/adel-safin/go-payment/internal/gateway/httpapi"
	pkgauth "github.com/adel-safin/go-payment/pkg/auth"
	"github.com/adel-safin/go-payment/pkg/config"
	"github.com/adel-safin/go-payment/pkg/logger"
	otelx "github.com/adel-safin/go-payment/pkg/otel"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	cfg := config.Load("gateway")
	log := logger.New(cfg.LogLevel)
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	shutdownTelemetry, err := otelx.Setup(ctx, cfg.ServiceName, cfg.OTLPEndpoint)
	if err != nil {
		log.Warn("otel setup failed, continuing without telemetry", "err", err)
		shutdownTelemetry = func(context.Context) error { return nil }
	}

	dialOpts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
	}

	idConn, err := grpc.NewClient(cfg.IdentityGRPC, dialOpts...)
	if err != nil {
		log.Error("dial identity", "err", err)
		os.Exit(1)
	}
	defer idConn.Close()

	var walletClient walletv1.WalletServiceClient
	var transferClient transferv1.TransferServiceClient
	if wConn, err := grpc.NewClient(cfg.WalletGRPCAddr, dialOpts...); err == nil {
		defer wConn.Close()
		walletClient = walletv1.NewWalletServiceClient(wConn)
	} else {
		log.Warn("wallet dial failed", "err", err)
	}
	if tConn, err := grpc.NewClient(cfg.TransferGRPC, dialOpts...); err == nil {
		defer tConn.Close()
		transferClient = transferv1.NewTransferServiceClient(tConn)
	} else {
		log.Warn("transfer dial failed", "err", err)
	}

	tokens := pkgauth.NewTokenManager(cfg.JWTSecret, cfg.JWTIssuer, cfg.JWTTTL)
	handler := httpapi.New(
		identityv1.NewIdentityServiceClient(idConn),
		walletClient,
		transferClient,
		tokens,
	)

	srv := &http.Server{Addr: cfg.HTTPAddr, Handler: handler}
	go func() {
		log.Info("gateway listening", "addr", cfg.HTTPAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("http failed", "err", err)
			stop()
		}
	}()

	<-ctx.Done()
	shCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(shCtx)
	_ = shutdownTelemetry(shCtx)
	log.Info("gateway stopped")
}
