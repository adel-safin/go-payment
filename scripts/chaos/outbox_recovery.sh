#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
COMPOSE="$ROOT/deploy/compose/docker-compose.yml"

echo "Stopping redpanda for 10s to exercise outbox recovery..."
docker compose -f "$COMPOSE" stop redpanda
sleep 10
echo "Starting redpanda again..."
docker compose -f "$COMPOSE" start redpanda
echo "Wait for health..."
sleep 15
echo "Outbox worker should republish unpublished events. Check transfer logs and Jaeger."
