#!/usr/bin/env bash
# Linux 生产：Image 模式一键拉起（不在服务器上 pnpm build）
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

COMPOSE_FILE="${COMPOSE_FILE:-docker-compose.image.yml}"
ENV_FILE="${ENV_FILE:-${SCRIPT_DIR}/.env}"
SQL_FILE="${SQL_FILE:-${SCRIPT_DIR}/../../backend/store/pg/migration/full_fresh_deploy.sql}"

if [[ ! -f "$ENV_FILE" ]]; then
  echo ">>> 生成 .env"
  FORCE=1 "$SCRIPT_DIR/generate-env.sh" "$ENV_FILE"
fi

# shellcheck disable=SC1090
set -a
source "$ENV_FILE"
set +a

mkdir -p ./data/caddy/caddy_config ./data/caddy/caddy_data ./data/caddy/run

echo ">>> 拉取镜像"
docker compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" pull

echo ">>> 启动 PostgreSQL"
docker compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" up -d panda-wiki-postgres

echo ">>> 等待 PostgreSQL 就绪"
for i in $(seq 1 60); do
  if docker compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" exec -T panda-wiki-postgres \
    pg_isready -U panda-wiki >/dev/null 2>&1; then
    break
  fi
  if [[ "$i" -eq 60 ]]; then
    echo "错误: PostgreSQL 未在预期时间内就绪，请检查: docker compose -f $COMPOSE_FILE logs panda-wiki-postgres"
    exit 1
  fi
  sleep 2
done

TABLE_COUNT="$(
  docker compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" exec -T panda-wiki-postgres \
    psql -U panda-wiki -d panda-wiki -tAc "SELECT count(*) FROM information_schema.tables WHERE table_schema='public' AND table_name='knowledge_bases';" \
    2>/dev/null | tr -d '[:space:]' || echo 0
)"

if [[ "$TABLE_COUNT" == "0" ]]; then
  if [[ ! -f "$SQL_FILE" ]]; then
    echo "错误: 找不到初始化 SQL: $SQL_FILE"
    exit 1
  fi
  echo ">>> 首次部署：导入 full_fresh_deploy.sql"
  cat "$SQL_FILE" | docker compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" exec -T panda-wiki-postgres \
    psql -U panda-wiki -d panda-wiki
else
  echo ">>> 数据库已初始化，跳过 SQL 导入"
fi

echo ">>> 启动全部服务"
docker compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" up -d

echo ">>> 等待 API 健康检查"
for i in $(seq 1 90); do
  if curl -sf http://127.0.0.1:8000/health >/dev/null 2>&1; then
    echo ""
    echo "部署完成。"
    echo "  前台: http://<服务器IP>:3010  （经 Caddy）"
    echo "  后台: https://<服务器IP>:2443"
    echo "  API:  http://127.0.0.1:8000/health"
    echo ""
    echo "管理员密码见 .env 中 ADMIN_PASSWORD"
    exit 0
  fi
  sleep 2
done

echo "警告: API 健康检查超时。查看日志:"
echo "  docker compose -f $COMPOSE_FILE logs panda-wiki-api --tail 80"
exit 1
