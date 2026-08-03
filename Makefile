MODULE := github.com/adel-safin/go-payment
GOBIN := $(shell go env GOPATH)/bin
export PATH := $(GOBIN):$(PATH)

.PHONY: gen migrate test test-integration test-race lint up down build seed e2e load

gen:
	@command -v protoc >/dev/null || (echo "protoc required"; exit 1)
	protoc -I api/proto \
		--go_out=api/gen --go_opt=paths=source_relative \
		--go-grpc_out=api/gen --go-grpc_opt=paths=source_relative \
		api/proto/identity/v1/identity.proto \
		api/proto/wallet/v1/wallet.proto \
		api/proto/transfer/v1/transfer.proto
	@command -v sqlc >/dev/null && sqlc generate || true

migrate:
	@echo "migrations run on service startup via goose"

test:
	go test ./... -count=1

test-race:
	go test ./... -race -count=1

test-integration:
	go test ./... -tags=integration -count=1 -timeout 10m

lint:
	@command -v golangci-lint >/dev/null && golangci-lint run ./... || go vet ./...

up:
	docker compose -f deploy/compose/docker-compose.yml up -d --build

down:
	docker compose -f deploy/compose/docker-compose.yml down -v

build:
	go build -o bin/gateway ./cmd/gateway
	go build -o bin/identity ./cmd/identity
	go build -o bin/wallet ./cmd/wallet
	go build -o bin/transfer ./cmd/transfer
	go build -o bin/notification ./cmd/notification

seed:
	go run ./scripts/seed

e2e:
	go test ./tests/e2e -tags=e2e -count=1 -timeout 5m

load:
	k6 run scripts/load/transfer.js
