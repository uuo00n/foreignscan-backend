#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

MODE="${1:-dev}"
if [[ "${MODE}" != "dev" && "${MODE}" != "prod" ]]; then
  echo "Usage: $0 [dev|prod] [extra up args...]" >&2
  exit 1
fi
shift || true

echo "[rebuild] Stopping existing services (${MODE})..."
"${ROOT_DIR}/scripts/linux/docker-down.sh" "${MODE}" --remove-orphans

echo
echo "[rebuild] Rebuilding and starting services (${MODE})..."
exec "${ROOT_DIR}/scripts/linux/docker-up.sh" "${MODE}" "$@"
