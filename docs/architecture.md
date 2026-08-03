# Architecture

## Services

- **gateway** — REST edge, JWT auth, fans out to gRPC services
- **identity** — users, password hashing, JWT issuance
- **wallet** — double-entry ledger, optimistic locking
- **transfer** — idempotent transfers, transactional outbox → Kafka
- **notification** — Kafka consumer with processed_events idempotency

## Patterns

Transactional outbox, idempotency keys, optimistic concurrency, W3C trace propagation
across HTTP, gRPC, and Kafka headers.

## Money

All amounts are `int64` minor units (e.g. kopecks). No floating point.
