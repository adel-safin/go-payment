# Go Payment / Ledger Platform

Event-driven payment platform in Go: identity, wallet (double-entry ledger),
transfer (idempotent + transactional outbox), notification consumer.

## Quick start

```bash
make up          # infrastructure + services via Docker Compose
make test        # unit tests
make test-integration  # integration tests (testcontainers)
```

Gateway: `http://localhost:8080`  
Jaeger: `http://localhost:16686`  
Grafana: `http://localhost:3000`

## Example flow

```bash
# register
curl -s -X POST localhost:8080/v1/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"email":"alice@example.com","password":"secret123"}'

# login
TOKEN=$(curl -s -X POST localhost:8080/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"alice@example.com","password":"secret123"}' | jq -r .token)

# create wallets / transfer — see docs/architecture.md
```

## Layout

See `docs/architecture.md` for service boundaries and patterns.
