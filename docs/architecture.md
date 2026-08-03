# Architecture

## Services

- **gateway** — REST edge, JWT auth, fans out to gRPC services
- **identity** — users, password hashing, JWT issuance
- **wallet** — double-entry ledger, optimistic locking
- **transfer** — idempotent transfers, transactional outbox → Kafka
- **notification** — Kafka consumer with processed_events idempotency

## Patterns

Transactional outbox, idempotency keys, optimistic concurrency, W3C trace propagation
across HTTP, gRPC, and Kafka headers. Kafka publish retries with DLQ fallback.
Wallet gRPC client uses a consecutive-failure circuit breaker.

## Money

All amounts are `int64` minor units (e.g. kopecks). No floating point.

## Transfer sequence

```mermaid
sequenceDiagram
  participant Client
  participant Gateway
  participant Transfer
  participant Wallet
  participant Outbox
  participant Kafka
  participant Notification

  Client->>Gateway: POST /v1/transfers Idempotency-Key
  Gateway->>Transfer: CreateTransfer
  Transfer->>Wallet: Debit
  Transfer->>Wallet: Credit
  Transfer->>Outbox: insert transfer + outbox_events (one TX)
  Transfer-->>Gateway: transfer_id
  Gateway-->>Client: 200
  Outbox->>Kafka: publish transfer.completed
  Kafka->>Notification: consume
  Notification->>Notification: processed_events claim
```

## Observability

- OpenTelemetry traces/metrics via OTLP → Collector → Jaeger / Prometheus
- Jaeger UI: http://localhost:16686
- Grafana: http://localhost:3000
- Graceful shutdown on SIGINT/SIGTERM for all services
