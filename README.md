# Go Payment / Ledger Platform

Event-driven payment platform in Go: identity, wallet (double-entry ledger),
transfer (idempotent + transactional outbox), notification consumer.

## Quick start

```bash
make up                 # infrastructure + services via Docker Compose
make test               # unit tests
make test-race          # unit + race detector
make test-integration   # testcontainers integration tests
make e2e                # requires running stack (GATEWAY_URL)
make seed               # register user + create wallets
```

Gateway: `http://localhost:8080`  
Jaeger: `http://localhost:16686`  
Grafana: `http://localhost:3000`

## Example flow

```bash
# register + login
curl -s -X POST localhost:8080/v1/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"email":"alice@example.com","password":"secret123"}'

TOKEN=$(curl -s -X POST localhost:8080/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"alice@example.com","password":"secret123"}' | jq -r .token)

# create wallets
curl -s -X POST localhost:8080/v1/wallets \
  -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"currency":"RUB"}'
```

Fund wallets via wallet `Credit` gRPC, then:

```bash
curl -s -X POST localhost:8080/v1/transfers \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -H "Idempotency-Key: $(uuidgen)" \
  -d '{"from_wallet_id":"...","to_wallet_id":"...","amount_minor":100,"currency":"RUB"}'
```

## Testing pyramid

| Level | Command |
|-------|---------|
| Unit | `make test` |
| Race | `make test-race` |
| Integration | `make test-integration` |
| E2E | `make up && make e2e` |
| Load | `make load` (needs k6) |
| Chaos | `bash scripts/chaos/outbox_recovery.sh` |

## Layout

See [docs/architecture.md](docs/architecture.md).
