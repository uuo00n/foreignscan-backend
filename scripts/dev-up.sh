#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ENV_FILE="${ENV_FILE:-${ROOT_DIR}/.env.docker}"
BACKEND_BASE="${BACKEND_BASE:-http://127.0.0.1:3000}"
YOLO_BASE="${YOLO_BASE:-http://127.0.0.1:8077}"

trim() {
  local s="${1:-}"
  s="${s#"${s%%[![:space:]]*}"}"
  s="${s%"${s##*[![:space:]]}"}"
  printf '%s' "${s}"
}

extract_detect_url() {
  if [[ ! -f "${ENV_FILE}" ]]; then
    return 0
  fi
  awk -F= '
    $1=="FS_DETECT_URL" {
      gsub(/^[ \t"]+|[ \t"]+$/, "", $2);
      print $2;
      exit
    }
  ' "${ENV_FILE}" 2>/dev/null || true
}

compact_body() {
  local body="${1:-}"
  body="$(printf '%s' "${body}" | tr '\n' ' ' | tr '\r' ' ')"
  body="$(printf '%s' "${body}" | sed 's/[[:space:]]\+/ /g')"
  printf '%s' "${body:0:180}"
}

request() {
  local url="${1:?url is required}"
  local tmp
  tmp="$(mktemp)"
  local code
  if ! code="$(curl -sS --max-time 3 -o "${tmp}" -w '%{http_code}' "${url}")"; then
    rm -f "${tmp}"
    echo "ERR"
    echo ""
    return 1
  fi
  local body
  body="$(cat "${tmp}")"
  rm -f "${tmp}"
  echo "${code}"
  echo "${body}"
  return 0
}

assert_status_200() {
  local name="${1:?name is required}"
  local url="${2:?url is required}"

  local resp code body
  if ! resp="$(request "${url}")"; then
    echo "[contract][FAIL] ${name}: 无法访问 ${url}" >&2
    return 1
  fi
  code="$(printf '%s\n' "${resp}" | sed -n '1p')"
  body="$(printf '%s\n' "${resp}" | sed -n '2,$p')"

  if [[ "${code}" != "200" ]]; then
    local brief
    brief="$(compact_body "${body}")"
    if [[ "${code}" == "404" ]]; then
      echo "[contract][FAIL] ${name}: HTTP 404（后端运行版本可能缺少该接口） ${url}" >&2
    else
      echo "[contract][FAIL] ${name}: HTTP ${code} ${url}" >&2
    fi
    if [[ -n "${brief}" ]]; then
      echo "[contract][DETAIL] ${brief}" >&2
    fi
    return 1
  fi

  echo "[contract][OK] ${name}: ${url}"
  return 0
}

normalize_detect_probe_url() {
  local detect_url
  detect_url="$(trim "${1:-}")"
  if [[ -z "${detect_url}" ]]; then
    echo ""
    return 0
  fi
  local base="${detect_url%/}"
  if [[ "${base}" == */api ]]; then
    echo "${base}/room-models"
    return 0
  fi
  echo "${base}/api/room-models"
}

run_contract_checks() {
  if ! command -v curl >/dev/null 2>&1; then
    echo "[contract][FAIL] 缺少 curl，无法执行联调契约检查" >&2
    return 1
  fi

  echo
  echo "[contract] 开始联调契约检查..."

  assert_status_200 "backend health" "${BACKEND_BASE%/}/health"
  assert_status_200 "backend room-models api" "${BACKEND_BASE%/}/api/room-models"
  assert_status_200 "yolo room-models api" "${YOLO_BASE%/}/api/room-models"

  local detect_url
  detect_url="$(extract_detect_url)"
  detect_url="$(trim "${detect_url}")"
  if [[ -z "${detect_url}" ]]; then
    echo "[contract][FAIL] ${ENV_FILE} 未配置 FS_DETECT_URL" >&2
    return 1
  fi

  local probe_url
  probe_url="$(normalize_detect_probe_url "${detect_url}")"
  if [[ -z "${probe_url}" ]]; then
    echo "[contract][FAIL] FS_DETECT_URL 无法解析: ${detect_url}" >&2
    return 1
  fi

  local host_probe_url="${probe_url/host.docker.internal/127.0.0.1}"
  assert_status_200 "FS_DETECT_URL(host probe)" "${host_probe_url}"

  echo "[contract][OK] 联调契约检查通过"
}

"${ROOT_DIR}/scripts/docker-up.sh" dev "$@"
run_contract_checks
