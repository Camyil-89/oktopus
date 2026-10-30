#!/usr/bin/env bash
# CI: tarball for GitHub Release (compose + stack scripts + ClickHouse init).
set -euo pipefail

VERSION="${1:?usage: ci-release-bundle.sh <semver>}"
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
DIST="$ROOT/dist"
OUT="$DIST/oktopus-${VERSION}-deploy.tar.gz"

mkdir -p "$DIST"
tar -czf "$OUT" \
  -C "$ROOT" \
  docker-compose.yml \
  docker-compose.desktop.yml \
  docker-compose.portainer.yml \
  scripts/stack-up.sh \
  scripts/stack-up.bat \
  scripts/stack-up-env.ps1 \
  scripts/stack-up-banner.ps1 \
  deploy/clickhouse/init

echo "bundle=$OUT"
