#!/usr/bin/env bash
# Debian 原生部署（无 Docker）：Postgres + Redis + NATS + MinIO + API + App + Nginx
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"
ENV_FILE="${ENV_FILE:-$REPO_ROOT/docs/deploy/.env}"
INSTALL_ROOT="${INSTALL_ROOT:-/opt/pandawiki}"
RUN_USER="${RUN_USER:-deploy}"
SUDO="${SUDO:-sudo}"
if [[ -n "${SUDO_PASSWORD:-}" ]]; then
  sudo() { echo "$SUDO_PASSWORD" | command sudo -S "$@"; }
fi

write_as_root() {
  local dest="$1"
  local tmp
  tmp="$(mktemp)"
  cat > "$tmp"
  $SUDO cp "$tmp" "$dest"
  rm -f "$tmp"
}

die() { echo "错误: $*" >&2; exit 1; }

[[ -f "$ENV_FILE" ]] || die "缺少 $ENV_FILE，请先 cp docs/deploy/.env.example 并填写"

# shellcheck disable=SC1090
set -a
source "$ENV_FILE"
set +a

ARCH="$(uname -m)"
case "$ARCH" in
  aarch64|arm64) GO_ARCH=arm64; MINIO_ARCH=arm64; NATS_ARCH=arm64 ;;
  x86_64|amd64) GO_ARCH=amd64; MINIO_ARCH=amd64; NATS_ARCH=amd64 ;;
  *) die "不支持的架构: $ARCH" ;;
esac

echo ">>> 停止 Docker 版 PandaWiki（如有）"
pkill -f "docker compose.*panda-wiki" 2>/dev/null || true
if [[ -f "$REPO_ROOT/docs/deploy/docker-compose.build.yml" ]]; then
  $SUDO docker compose -f "$REPO_ROOT/docs/deploy/docker-compose.build.yml" down --remove-orphans 2>/dev/null || true
fi

echo ">>> 安装系统依赖"
export DEBIAN_FRONTEND=noninteractive
$SUDO apt-get update -qq
$SUDO apt-get install -y -qq \
  redis-server curl ca-certificates build-essential git \
  postgresql-server-dev-15 postgresql-contrib

install_pgvector() {
  if $SUDO -u postgres psql -d postgres -tAc "SELECT 1 FROM pg_available_extensions WHERE name='vector';" 2>/dev/null | grep -q 1; then
    echo "pgvector 已可用"
    return 0
  fi
  echo ">>> 编译安装 pgvector（Debian 默认源无 pgvector 包）"
  PG_CONFIG="$(command -v pg_config || echo /usr/lib/postgresql/15/bin/pg_config)"
  workdir="$(mktemp -d)"
  git clone --depth 1 --branch v0.8.0 https://github.com/pgvector/pgvector.git "$workdir/pgvector"
  make -C "$workdir/pgvector" clean
  make -C "$workdir/pgvector" OPTFLAGS="" vector.so
  cp "$workdir/pgvector/sql/vector.sql" "$workdir/pgvector/sql/vector--0.8.0.sql"
  $SUDO install -m 755 "$workdir/pgvector/vector.so" "$("$PG_CONFIG" --pkglibdir)/vector.so"
  $SUDO install -d "$("$PG_CONFIG" --sharedir)/extension"
  $SUDO install -m 644 "$workdir/pgvector/sql/vector--0.8.0.sql" "$workdir/pgvector/vector.control" \
    "$("$PG_CONFIG" --sharedir)/extension/"
  rm -rf "$workdir"
}

install_pgvector

$SUDO systemctl enable --now postgresql redis-server

echo ">>> 安装 Go（若无）"
if ! command -v go >/dev/null 2>&1 || ! go version | grep -q 'go1.2[6-9]'; then
  GO_VER=1.26.0
  GO_TAR="go${GO_VER}.linux-${GO_ARCH}.tar.gz"
  curl -fsSL "https://go.dev/dl/${GO_TAR}" -o "/tmp/${GO_TAR}"
  $SUDO rm -rf /usr/local/go
  $SUDO tar -C /usr/local -xzf "/tmp/${GO_TAR}"
  $SUDO ln -sf /usr/local/go/bin/go /usr/local/bin/go
fi
export PATH="/usr/local/go/bin:$PATH"
export GOTOOLCHAIN=auto

echo ">>> 安装 NATS / MinIO 二进制（若无）"
$SUDO mkdir -p "$INSTALL_ROOT/bin"
if [[ ! -x "$INSTALL_ROOT/bin/nats-server" ]]; then
  NATS_VER=2.10.29
  curl -fsSL "https://github.com/nats-io/nats-server/releases/download/v${NATS_VER}/nats-server-v${NATS_VER}-linux-${NATS_ARCH}.tar.gz" \
    | tar -xz -C /tmp
  $SUDO cp "/tmp/nats-server-v${NATS_VER}-linux-${NATS_ARCH}/nats-server" "$INSTALL_ROOT/bin/"
  $SUDO chmod +x "$INSTALL_ROOT/bin/nats-server"
fi
if [[ ! -x "$INSTALL_ROOT/bin/minio" ]]; then
  curl -fsSL "https://dl.min.io/server/minio/release/linux-${MINIO_ARCH}/minio" -o /tmp/minio
  $SUDO install -m 755 /tmp/minio "$INSTALL_ROOT/bin/minio"
fi

echo ">>> 初始化 PostgreSQL"
$SUDO -u postgres psql -v ON_ERROR_STOP=0 -c "CREATE USER \"panda-wiki\" WITH PASSWORD '${POSTGRES_PASSWORD}';" 2>/dev/null || true
$SUDO -u postgres psql -v ON_ERROR_STOP=0 -c "ALTER USER \"panda-wiki\" WITH PASSWORD '${POSTGRES_PASSWORD}';" 2>/dev/null || true
$SUDO -u postgres psql -v ON_ERROR_STOP=0 -c "CREATE DATABASE \"panda-wiki\" OWNER \"panda-wiki\";" 2>/dev/null || true
$SUDO -u postgres psql -d panda-wiki -v ON_ERROR_STOP=1 -c "CREATE EXTENSION IF NOT EXISTS vector;"

SQL_FILE="$REPO_ROOT/backend/store/pg/migration/full_fresh_deploy.sql"
TABLE_COUNT="$($SUDO -u postgres psql -d panda-wiki -tAc \
  "SELECT count(*) FROM information_schema.tables WHERE table_schema='public' AND table_name='knowledge_bases';" \
  2>/dev/null | tr -d '[:space:]' || echo 0)"
if [[ "$TABLE_COUNT" == "0" ]]; then
  echo ">>> 导入 full_fresh_deploy.sql"
  $SUDO -u postgres psql -d panda-wiki -v ON_ERROR_STOP=1 -f "$SQL_FILE"
fi
$SUDO -u postgres psql -d panda-wiki -v ON_ERROR_STOP=1 -c \
  'GRANT ALL ON SCHEMA public TO "panda-wiki"; GRANT ALL ON ALL TABLES IN SCHEMA public TO "panda-wiki"; GRANT ALL ON ALL SEQUENCES IN SCHEMA public TO "panda-wiki";'

echo ">>> 配置 Redis 密码"
REDIS_CONF="/etc/redis/redis.conf"
redis_tmp="$(mktemp)"
echo "requirepass ${REDIS_PASSWORD}" > "$redis_tmp"
$SUDO sed -i '/^requirepass /d' "$REDIS_CONF"
$SUDO sh -c "cat '$redis_tmp' >> '$REDIS_CONF'"
rm -f "$redis_tmp"
$SUDO systemctl restart redis-server

echo ">>> 写入 systemd 单元"
$SUDO mkdir -p "$INSTALL_ROOT"/{bin,data/minio,logs,run}
$SUDO mkdir -p /app/etc/nginx/ssl /data
$SUDO chown -R "$RUN_USER:$RUN_USER" /app/etc/nginx/ssl "$INSTALL_ROOT" /data

write_as_root /etc/systemd/system/pandawiki-nats.service <<UNIT
[Unit]
Description=PandaWiki NATS
After=network.target

[Service]
Type=simple
User=$RUN_USER
ExecStart=$INSTALL_ROOT/bin/nats-server -js -m 8222 --user panda-wiki --pass ${NATS_PASSWORD}
Restart=always

[Install]
WantedBy=multi-user.target
UNIT

write_as_root /etc/systemd/system/pandawiki-minio.service <<UNIT
[Unit]
Description=PandaWiki MinIO (测试用 OSS 模拟)
After=network.target

[Service]
Type=simple
User=$RUN_USER
Environment=MINIO_ROOT_USER=${S3_ACCESS_KEY}
Environment=MINIO_ROOT_PASSWORD=${S3_SECRET_KEY}
ExecStart=$INSTALL_ROOT/bin/minio server $INSTALL_ROOT/data/minio --address :9000 --console-address :9001
Restart=always

[Install]
WantedBy=multi-user.target
UNIT

# 原生环境变量（覆盖 docker 主机名）
NATIVE_ENV="$INSTALL_ROOT/pandawiki.env"
write_as_root "$NATIVE_ENV" <<ENV
POSTGRES_PASSWORD=${POSTGRES_PASSWORD}
PG_DSN=host=127.0.0.1 user=panda-wiki password=${POSTGRES_PASSWORD} dbname=panda-wiki port=5432 sslmode=disable TimeZone=Asia/Shanghai
MQ_NATS_SERVER=nats://127.0.0.1:4222
NATS_PASSWORD=${NATS_PASSWORD}
REDIS_ADDR=127.0.0.1:6379
REDIS_PASSWORD=${REDIS_PASSWORD}
S3_ENDPOINT=127.0.0.1:9000
S3_ACCESS_KEY=${S3_ACCESS_KEY}
S3_SECRET_KEY=${S3_SECRET_KEY}
S3_BUCKET=${S3_BUCKET:-static-file}
S3_REGION=${S3_REGION:-oss-cn-hangzhou}
S3_USE_SSL=0
S3_PUBLIC_BASE_URL=${S3_PUBLIC_BASE_URL:-http://10.211.55.5:9000/static-file}
JWT_SECRET=${JWT_SECRET}
ADMIN_PASSWORD=${ADMIN_PASSWORD}
SENTRY_ENABLED=false
RUN_WORKER=${RUN_WORKER:-1}
ADMIN_ENABLED=${ADMIN_ENABLED:-1}
ADMIN_PORT=2443
ADMIN_DIST_DIR=${REPO_ROOT}/web/admin/dist
RAG_PROVIDER=${RAG_PROVIDER:-pg}
RAG_PG_EMBEDDING_DIM=${RAG_PG_EMBEDDING_DIM:-1024}
CADDY_API=disabled
ENV

write_as_root /etc/systemd/system/pandawiki-api.service <<UNIT
[Unit]
Description=PandaWiki API
After=network.target postgresql.service redis-server.service pandawiki-nats.service pandawiki-minio.service
Wants=postgresql.service redis-server.service pandawiki-nats.service pandawiki-minio.service

[Service]
Type=simple
User=$RUN_USER
WorkingDirectory=$REPO_ROOT/backend
EnvironmentFile=$NATIVE_ENV
ExecStart=$INSTALL_ROOT/bin/panda-wiki-api
Restart=always
LimitNOFILE=65535

[Install]
WantedBy=multi-user.target
UNIT

APP_ENV="$INSTALL_ROOT/pandawiki-app.env"

echo ">>> 编译 Admin + App"
cd "$REPO_ROOT/web"
if [[ "${ADMIN_SKIP_BUILD:-}" != "1" || "${APP_SKIP_BUILD:-}" != "1" ]]; then
  pnpm install --frozen-lockfile
fi

build_admin() {
  NODE_OPTIONS=--max-old-space-size=4096 pnpm --filter panda-wiki-admin build
}

if [[ "${ADMIN_SKIP_BUILD:-}" == "1" ]]; then
  echo "跳过 Admin 编译 (ADMIN_SKIP_BUILD=1)"
elif ! build_admin; then
  echo "Admin 编译失败，尝试从本地旧镜像提取 dist..."
  if command -v docker >/dev/null 2>&1 && $SUDO docker images -q deploy-panda-wiki-admin:latest | grep -q .; then
    tmp_cid="$($SUDO docker create deploy-panda-wiki-admin:latest)"
    rm -rf "$REPO_ROOT/web/admin/dist"
    mkdir -p "$REPO_ROOT/web/admin/dist"
    $SUDO docker cp "$tmp_cid:/usr/share/nginx/html/." "$REPO_ROOT/web/admin/dist/"
    $SUDO docker rm "$tmp_cid" >/dev/null
    $SUDO chown -R "$RUN_USER:$RUN_USER" "$REPO_ROOT/web/admin/dist"
    echo "已从 deploy-panda-wiki-admin:latest 提取 admin dist"
  else
    die "Admin 编译失败且无可用 dist"
  fi
fi

build_app() {
  NODE_OPTIONS=--max-old-space-size=4096 pnpm --filter panda-wiki-app build
  APP_DIST="$REPO_ROOT/web/app/dist"
  mkdir -p "$APP_DIST/standalone/app/dist"
  cp -r "$APP_DIST/static" "$APP_DIST/standalone/app/dist/"
  [[ -d "$REPO_ROOT/web/app/public" ]] && cp -r "$REPO_ROOT/web/app/public" "$APP_DIST/standalone/app/"
}

if [[ "${APP_SKIP_BUILD:-}" == "1" ]]; then
  echo "跳过 App 编译 (APP_SKIP_BUILD=1)"
  APP_RUN_DIR="$INSTALL_ROOT/app"
  if [[ ! -f "$APP_RUN_DIR/app/server.js" ]]; then
    die "APP_SKIP_BUILD=1 但 $APP_RUN_DIR/app/server.js 不存在，请先提取或编译 App"
  fi
elif ! build_app; then
  echo "App 编译失败，尝试从本地旧镜像提取 standalone..."
  if command -v docker >/dev/null 2>&1 && $SUDO docker images -q deploy-panda-wiki-app:latest | grep -q .; then
    tmp_cid="$($SUDO docker create deploy-panda-wiki-app:latest)"
    rm -rf "$INSTALL_ROOT/app"
    $SUDO docker cp "$tmp_cid:/app" "$INSTALL_ROOT/app"
    $SUDO docker rm "$tmp_cid" >/dev/null
    $SUDO chown -R "$RUN_USER:$RUN_USER" "$INSTALL_ROOT/app"
    APP_RUN_DIR="$INSTALL_ROOT/app"
    echo "已从 deploy-panda-wiki-app:latest 提取 app"
  else
    die "App 编译失败且无可用 standalone"
  fi
else
  APP_RUN_DIR="$REPO_ROOT/web/app/dist/standalone"
fi

write_as_root "$APP_ENV" <<ENV
NODE_ENV=production
PORT=3010
HOSTNAME=127.0.0.1
TARGET=http://127.0.0.1:8000
STATIC_FILE_TARGET=http://127.0.0.1:8000
DEV_KB_ID=${DEV_KB_ID:-}
ENV

write_as_root /etc/systemd/system/pandawiki-app.service <<UNIT
[Unit]
Description=PandaWiki App (Next.js)
After=pandawiki-api.service
Wants=pandawiki-api.service

[Service]
Type=simple
User=$RUN_USER
WorkingDirectory=$APP_RUN_DIR
EnvironmentFile=$APP_ENV
ExecStart=/usr/bin/node app/server.js
Restart=always

[Install]
WantedBy=multi-user.target
UNIT

echo ">>> 编译 API"
cd "$REPO_ROOT/backend"
go build -ldflags "-s -w" -o "$INSTALL_ROOT/bin/panda-wiki-api" cmd/api/main.go cmd/api/wire_gen.go
$SUDO chown "$RUN_USER:$RUN_USER" "$INSTALL_ROOT/bin/panda-wiki-api"

echo ">>> 配置 Nginx 反代（端口 8888）"
NGINX_CONF="/etc/nginx/conf.d/pandawiki.conf"
if [[ -d /www/server/panel/vhost/nginx ]]; then
  NGINX_CONF="/www/server/panel/vhost/nginx/pandawiki.conf"
elif [[ -d /www/server/nginx/conf/vhost ]]; then
  NGINX_CONF="/www/server/nginx/conf/vhost/pandawiki.conf"
fi
write_as_root "$NGINX_CONF" <<'NGINX'
server {
    listen 8888;
    server_name _;

    client_max_body_size 100m;

    location / {
        proxy_pass http://127.0.0.1:3010;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
NGINX
if [[ -x /www/server/nginx/sbin/nginx ]]; then
  $SUDO /www/server/nginx/sbin/nginx -t && $SUDO /www/server/nginx/sbin/nginx -s reload
else
  $SUDO nginx -t && $SUDO systemctl reload nginx
fi

echo ">>> 启动服务"
$SUDO systemctl unmask pandawiki-nats pandawiki-minio pandawiki-api pandawiki-app 2>/dev/null || true
$SUDO systemctl daemon-reload
for svc in pandawiki-nats pandawiki-minio pandawiki-api pandawiki-app; do
  $SUDO systemctl enable "$svc"
  $SUDO systemctl restart "$svc"
done

echo ">>> 等待 API 就绪"
for i in $(seq 1 60); do
  if curl -sf http://127.0.0.1:8000/health >/dev/null 2>&1; then
    echo ""
    echo "部署完成。"
    echo "  前台: http://$(hostname -I | awk '{print $1}'):8888"
    echo "  后台: https://$(hostname -I | awk '{print $1}'):2443  （自签证书，浏览器需跳过警告）"
    echo "  API:  http://127.0.0.1:8000/health"
    echo "  管理员密码: ${ADMIN_PASSWORD}"
    echo ""
    echo "首次使用：登录后台创建知识库后，把 KB ID 写入 $APP_ENV 的 DEV_KB_ID 并 systemctl restart pandawiki-app"
    exit 0
  fi
  sleep 2
done

echo "API 启动超时，查看日志:"
$SUDO journalctl -u pandawiki-api -n 50 --no-pager
exit 1
