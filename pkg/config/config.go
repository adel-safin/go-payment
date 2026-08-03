package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds process-wide settings loaded from environment variables.
type Config struct {
	ServiceName    string
	HTTPAddr       string
	GRPCAddr       string
	DatabaseURL    string
	RedisAddr      string
	RedisPassword  string
	KafkaBrokers   string
	KafkaTopic     string
	KafkaDLQTopic  string
	OTLPEndpoint   string
	JWTSecret      string
	JWTIssuer      string
	JWTTTL         time.Duration
	WalletGRPCAddr string
	IdentityGRPC   string
	TransferGRPC   string
	LogLevel       string
}

func Load(serviceName string) Config {
	return Config{
		ServiceName:    serviceName,
		HTTPAddr:       getenv("HTTP_ADDR", ":8080"),
		GRPCAddr:       getenv("GRPC_ADDR", ":9090"),
		DatabaseURL:    getenv("DATABASE_URL", "postgres://payment:payment@localhost:5432/payment?sslmode=disable"),
		RedisAddr:      getenv("REDIS_ADDR", "localhost:6379"),
		RedisPassword:  getenv("REDIS_PASSWORD", ""),
		KafkaBrokers:   getenv("KAFKA_BROKERS", "localhost:19092"),
		KafkaTopic:     getenv("KAFKA_TOPIC", "transfer.events"),
		KafkaDLQTopic:  getenv("KAFKA_DLQ_TOPIC", "transfer.events.dlq"),
		OTLPEndpoint:   getenv("OTEL_EXPORTER_OTLP_ENDPOINT", "localhost:4317"),
		JWTSecret:      getenv("JWT_SECRET", "dev-secret-change-me"),
		JWTIssuer:      getenv("JWT_ISSUER", "go-payment"),
		JWTTTL:         durationEnv("JWT_TTL", 24*time.Hour),
		WalletGRPCAddr: getenv("WALLET_GRPC_ADDR", "localhost:9092"),
		IdentityGRPC:   getenv("IDENTITY_GRPC_ADDR", "localhost:9091"),
		TransferGRPC:   getenv("TRANSFER_GRPC_ADDR", "localhost:9093"),
		LogLevel:       getenv("LOG_LEVEL", "info"),
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func durationEnv(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
		if sec, err := strconv.Atoi(v); err == nil {
			return time.Duration(sec) * time.Second
		}
	}
	return fallback
}
