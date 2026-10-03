#!/usr/bin/env bash
# CI: сборка бэка, unit-тесты, attack-and-tests с минимальным стеком docker-compose.dev.yml.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

COMPOSE=(docker compose -f docker-compose.dev.yml)
SERVICES=(postgres openldap clickhouse)
OKTOPUS_BIN="${OKTOPUS_BIN:-${ROOT}/bin/oktopus}"
SERVE_LOG="${SERVE_LOG:-${ROOT}/bin/serve.log}"

cleanup() {
  if [[ -n "${SERVE_PID:-}" ]] && kill -0 "$SERVE_PID" 2>/dev/null; then
    kill "$SERVE_PID" 2>/dev/null || true
    wait "$SERVE_PID" 2>/dev/null || true
  fi
  if [[ "${CI_SKIP_COMPOSE:-}" != "1" ]]; then
    "${COMPOSE[@]}" down -v --remove-orphans 2>/dev/null || true
  fi
}
trap cleanup EXIT

export DATABASE_URL="${DATABASE_URL:-postgres://oktopus:oktopus@127.0.0.1:5434/oktopus?sslmode=disable}"
export CLICKHOUSE_ADDR="${CLICKHOUSE_ADDR:-127.0.0.1:9054}"
export CLICKHOUSE_DATABASE="${CLICKHOUSE_DATABASE:-oktopus}"
export CLICKHOUSE_USER="${CLICKHOUSE_USER:-oktopus}"
export CLICKHOUSE_PASSWORD="${CLICKHOUSE_PASSWORD:-oktopus}"
export OKTOPUS_JWT_SECRET="${OKTOPUS_JWT_SECRET:-ci-jwt-secret}"
export OKTOPUS_ADMIN_USERNAME="${OKTOPUS_ADMIN_USERNAME:-admin}"
export OKTOPUS_ADMIN_PASSWORD="${OKTOPUS_ADMIN_PASSWORD:-admin}"

if [[ "${CI_SKIP_COMPOSE:-}" != "1" ]]; then
  echo "==> docker compose: ${SERVICES[*]}"
  "${COMPOSE[@]}" up -d --wait --wait-timeout 300 "${SERVICES[@]}"

  mkdir -p bin
  echo "==> go build ./cmd"
  go build -o "$OKTOPUS_BIN" ./cmd

  echo "==> go test ./internal/..."
  go test ./internal/...
else
  echo "==> CI_SKIP_COMPOSE=1 (ожидаем postgres/openldap/clickhouse на хосте)"
  if [[ ! -x "$OKTOPUS_BIN" ]]; then
    echo "missing binary: $OKTOPUS_BIN"
    exit 1
  fi
fi

mkdir -p "$(dirname "$SERVE_LOG")"
echo "==> serve (background)"
"$OKTOPUS_BIN" serve -migrations "$ROOT/db/migrations" >"$SERVE_LOG" 2>&1 &
SERVE_PID=$!

echo "==> wait for API"
ready=0
for _ in $(seq 1 90); do
  if curl -sf -X POST "http://127.0.0.1:8000/api/auth/login" \
    -H "Content-Type: application/json" \
    -d "{\"username\":\"${OKTOPUS_ADMIN_USERNAME}\",\"password\":\"${OKTOPUS_ADMIN_PASSWORD}\"}" \
    | grep -q '"username"'; then
    ready=1
    break
  fi
  sleep 2
done
if [[ "$ready" -ne 1 ]]; then
  echo "serve did not become ready; log:"
  tail -n 80 "$SERVE_LOG" || true
  exit 1
fi

echo "==> attack-and-tests: run-all-acl"
go run ./attack-and-tests/run-all-acl

echo "==> attack-and-tests: mitm-host-mismatch"
go run ./attack-and-tests/mitm-host-mismatch all

echo "==> attack-and-tests: tunnel-dst-resolve"
go run ./attack-and-tests/tunnel-dst-resolve all

echo "==> backend integration: OK"
