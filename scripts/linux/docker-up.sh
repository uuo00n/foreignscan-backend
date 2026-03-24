#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "${ROOT_DIR}"

MODE="${1:-dev}"
if [[ "${MODE}" != "dev" && "${MODE}" != "prod" ]]; then
  echo "Usage: $0 [dev|prod]" >&2
  exit 1
fi
shift || true

ENV_FILE="${ENV_FILE:-.env.docker}"
if [[ ! -f "${ENV_FILE}" ]]; then
  if [[ -f ".env.docker.example" ]]; then
    cp ".env.docker.example" "${ENV_FILE}"
    echo "Created ${ENV_FILE} from .env.docker.example"
  else
    echo "Error: ${ENV_FILE} not found and .env.docker.example is missing." >&2
    exit 1
  fi
fi

OVERRIDE_FILE="compose.dev.yml"
if [[ "${MODE}" == "prod" ]]; then
  OVERRIDE_FILE="compose.prod.yml"
fi

COMPOSE=(docker compose --env-file "${ENV_FILE}" -f compose.yml -f "${OVERRIDE_FILE}")
POSTGRES_HEALTH_TIMEOUT="${POSTGRES_HEALTH_TIMEOUT:-90}"

echo "[1/3] Starting postgres (${MODE})..."
"${COMPOSE[@]}" up -d postgres

echo "[2/3] Waiting postgres to become healthy (timeout: ${POSTGRES_HEALTH_TIMEOUT}s)..."
START_TS="$(date +%s)"
while true; do
  CID="$("${COMPOSE[@]}" ps -q postgres)"
  if [[ -n "${CID}" ]]; then
    HEALTH="$(docker inspect -f '{{if .State.Health}}{{.State.Health.Status}}{{else}}{{.State.Status}}{{end}}' "${CID}" 2>/dev/null || true)"
    if [[ "${HEALTH}" == "healthy" ]]; then
      echo "Postgres is healthy."
      break
    fi
  fi

  NOW_TS="$(date +%s)"
  if (( NOW_TS - START_TS >= POSTGRES_HEALTH_TIMEOUT )); then
    echo "Error: Timed out waiting for postgres to become healthy." >&2
    "${COMPOSE[@]}" logs --tail=100 postgres || true
    exit 1
  fi
  sleep 2
done

echo "[3/3] Starting api (${MODE})..."
"${COMPOSE[@]}" up --build -d api "$@"

echo
echo "Service status:"
"${COMPOSE[@]}" ps

if command -v curl >/dev/null 2>&1; then
  echo
  echo "Health checks:"
  if ! curl -fsS "http://localhost:3000/health"; then
    echo "Warning: /health is not reachable yet."
  fi
  echo
  if ! curl -fsS "http://localhost:3000/ready"; then
    echo "Warning: /ready is not reachable yet."
  fi
  echo

  DETECT_URL="$(
    awk -F= '
      $1=="FS_DETECT_URL" {
        gsub(/^[ \t"]+|[ \t"]+$/, "", $2);
        print $2;
        exit
      }
    ' "${ENV_FILE}" 2>/dev/null || true
  )"
  if [[ -n "${DETECT_URL}" ]] && ! curl -fsS --max-time 2 "${DETECT_URL}" >/dev/null 2>&1; then
    echo "Warning: FS_DETECT_URL is unreachable from host: ${DETECT_URL}"
  fi
fi
