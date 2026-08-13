#!/usr/bin/env bash
# 生成 docs/deploy/.env（随机密钥，Linux 生产 Image 模式一键用）
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ENV_FILE="${1:-${SCRIPT_DIR}/.env}"

rand_hex() {
  if command -v openssl >/dev/null 2>&1; then
    openssl rand -hex 16
  else
    head -c 16 /dev/urandom | od -An -tx1 | tr -d ' \n'
  fi
}

if [[ -f "$ENV_FILE" && "${FORCE:-}" != "1" ]]; then
  echo "已存在: $ENV_FILE"
  echo "如需覆盖: FORCE=1 $0"
  exit 0
fi

mkdir -p "$(dirname "$ENV_FILE")"

cat >"$ENV_FILE" <<EOF
# 由 generate-env.sh 生成 — 请妥善备份，丢失后无法恢复已加密数据
POSTGRES_PASSWORD=$(rand_hex)
REDIS_PASSWORD=$(rand_hex)
S3_ACCESS_KEY=s3panda-wiki
S3_SECRET_KEY=$(rand_hex)
NATS_PASSWORD=$(rand_hex)
QDRANT_API_KEY=$(rand_hex)
JWT_SECRET=$(rand_hex)
ADMIN_PASSWORD=$(rand_hex)
# 生产经 Caddy 注入 X-KB-ID 时可留空；仅直连 App 调试时需要
DEV_KB_ID=
PANDAWIKI_IMAGE_REPO=docker.io/caodanv
PANDAWIKI_IMAGE_TAG=latest
# API 内嵌 MQ 消费者与定时任务（Phase 2 默认开启；独立 Consumer 容器用 split-worker profile）
RUN_WORKER=1
ADMIN_ENABLED=1
RAG_PROVIDER=ct
RAG_PG_EMBEDDING_DIM=1024
EOF

chmod 600 "$ENV_FILE" 2>/dev/null || true
echo "已写入: $ENV_FILE"
