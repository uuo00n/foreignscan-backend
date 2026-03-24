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

docker compose --env-file "${ENV_FILE}" -f compose.yml -f "${OVERRIDE_FILE}" down "$@"
