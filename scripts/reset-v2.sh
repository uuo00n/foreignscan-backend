#!/usr/bin/env bash
set -euo pipefail

# 一次性重建脚本：清空历史数据 + 清空上传目录（不可恢复）
# 依赖：FS_POSTGRES_DSN 已配置，或 .env 中存在。

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT_DIR"

if [[ -z "${FS_POSTGRES_DSN:-}" ]]; then
  if [[ -f .env ]]; then
    # shellcheck disable=SC1091
    source .env
  fi
fi

if [[ -z "${FS_POSTGRES_DSN:-}" ]]; then
  echo "FS_POSTGRES_DSN is required"
  exit 1
fi

echo "[1/3] Truncating tables..."
psql "$FS_POSTGRES_DSN" <<'SQL'
BEGIN;
DO $$
DECLARE
  t text;
BEGIN
  FOREACH t IN ARRAY ARRAY[
    'detection_runs',
    'images',
    'style_images',
    'points',
    'rooms',
    'scenes'
  ]
  LOOP
    IF to_regclass(t) IS NOT NULL THEN
      EXECUTE format('TRUNCATE TABLE %I RESTART IDENTITY CASCADE', t);
    END IF;
  END LOOP;
END $$;
COMMIT;
SQL

echo "[2/3] Cleaning uploads directories..."
UPLOAD_DIR="${FS_UPLOAD_DIR:-$ROOT_DIR/cmd/server/uploads}"
rm -rf "$UPLOAD_DIR/images" "$UPLOAD_DIR/styles" "$UPLOAD_DIR/labels"
mkdir -p "$UPLOAD_DIR/images" "$UPLOAD_DIR/styles" "$UPLOAD_DIR/labels"

echo "[3/3] Done."
echo "Data reset completed: DB tables cleared and uploads directory rebuilt."
