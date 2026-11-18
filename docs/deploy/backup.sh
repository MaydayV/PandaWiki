#!/usr/bin/env bash
# PandaWiki PostgreSQL 数据备份脚本（stack-slim / compose 部署）
#
# 用法：
#   ./backup.sh                    # 备份到 ./data/backups/
#   ./backup.sh 14                 # 保留最近 14 份（默认 7）
#
# 定时备份（每天 3:30）：
#   30 3 * * * cd /path/to/pandawiki && ./docs/deploy/backup.sh >> ./data/backups/backup.log 2>&1
#
# 恢复：
#   cat ./data/backups/pandawiki-YYYYmmdd-HHMMSS.sql.gz | \
#     docker compose -f docs/deploy/docker-compose.image.yml exec -T postgres \
#     psql -U panda-wiki -d panda-wiki
#   （恢复到全新库前先 DROP/重建或清空数据卷）
set -euo pipefail

COMPOSE_FILE="docs/deploy/docker-compose.image.yml"
KEEP="${1:-7}"
BACKUP_DIR="data/backups"
STAMP="$(date +%Y%m%d-%H%M%S)"
TARGET="$BACKUP_DIR/pandawiki-$STAMP.sql.gz"

mkdir -p "$BACKUP_DIR"

echo "[$(date '+%F %T')] dumping database..."
docker compose -f "$COMPOSE_FILE" exec -T postgres \
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
