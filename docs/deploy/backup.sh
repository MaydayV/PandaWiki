#!/usr/bin/env bash
# PandaWiki PostgreSQL 数据备份脚本（stack-slim / compose 部署）
#
# 用法：
#   ./backup.sh                    # 发布包根目录执行（备份到 ./data/backups/）
#   ./docs/deploy/backup.sh        # 源码仓库根目录执行（备份到 ./docs/deploy/data/backups/）
#   ./backup.sh 14                 # 保留最近 14 份（默认 7）
#
# 定时备份（每天 3:30，发布包安装目录示例）：
#   30 3 * * * /opt/pandawiki/backup.sh >> /opt/pandawiki/data/backups/backup.log 2>&1
#
# 恢复：
#   gzip -dc ./data/backups/pandawiki-YYYYmmdd-HHMMSS.sql.gz | \
#     docker compose -f docker-compose.image.yml exec -T panda-wiki-postgres \
#     psql -U panda-wiki -d panda-wiki
#   （恢复到全新库前先 DROP/重建或清空数据卷）
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
COMPOSE_FILE="${COMPOSE_FILE:-${SCRIPT_DIR}/docker-compose.image.yml}"
ENV_FILE="${ENV_FILE:-${SCRIPT_DIR}/.env}"
KEEP="${1:-7}"
BACKUP_DIR="${BACKUP_DIR:-${SCRIPT_DIR}/data/backups}"
STAMP="$(date +%Y%m%d-%H%M%S)"
TARGET="$BACKUP_DIR/pandawiki-$STAMP.sql.gz"

COMPOSE_ARGS=()
if [[ -f "$ENV_FILE" ]]; then
  COMPOSE_ARGS+=(--env-file "$ENV_FILE")
fi
COMPOSE_ARGS+=(-f "$COMPOSE_FILE")

mkdir -p "$BACKUP_DIR"

echo "[$(date '+%F %T')] dumping database..."
docker compose "${COMPOSE_ARGS[@]}" exec -T panda-wiki-postgres \
  pg_dump -U panda-wiki -d panda-wiki --no-owner --no-privileges \
  | gzip > "$TARGET"

SIZE="$(du -h "$TARGET" | cut -f1)"
echo "[$(date '+%F %T')] done: $TARGET ($SIZE)"

# 清理过期备份
if [ "$KEEP" -gt 0 ]; then
  count=$(ls -1t "$BACKUP_DIR"/pandawiki-*.sql.gz 2>/dev/null | wc -l | tr -d ' ')
  if [ "$count" -gt "$KEEP" ]; then
    ls -1t "$BACKUP_DIR"/pandawiki-*.sql.gz | tail -n +$((KEEP + 1)) | while read -r f; do
      rm -f "$f"
      echo "[$(date '+%F %T')] pruned old backup: $f"
    done
  fi
fi
