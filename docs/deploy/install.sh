#!/usr/bin/env bash
# 乘风版一键安装：拉发布包 → 生成密钥 → 填写 OSS → 启动 5 容器
# 用法:
#   curl -fsSL https://github.com/MaydayV/PandaWiki/releases/latest/download/install.sh | bash
#   PANDAWIKI_VERSION=FV2.6.20.2111 INSTALL_DIR=/opt/pandawiki ./install.sh
set -euo pipefail

REPO="${PANDAWIKI_REPO:-MaydayV/PandaWiki}"
VERSION="${PANDAWIKI_VERSION:-latest}"
INSTALL_DIR="${INSTALL_DIR:-$PWD/pandawiki}"
IMAGE_REPO="${PANDAWIKI_IMAGE_REPO:-docker.io/caodanv}"

need_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "错误: 未找到命令 $1"
    exit 1
  fi
}

upsert_env() {
  local key="$1"
  local val="$2"
  local file="$3"
  local tmp
  tmp="$(mktemp)"
  awk -v k="$key" -v v="$val" '
    BEGIN { done=0 }
    $0 ~ "^" k "=" { print k "=" v; done=1; next }
    { print }
    END { if (!done) print k "=" v }
  ' "$file" >"$tmp"
  mv "$tmp" "$file"
}

is_deploy_dir() {
  [[ -f "$1/docker-compose.image.yml" && -f "$1/generate-env.sh" && -f "$1/quickstart.sh" ]]
}

is_placeholder() {
  case "${1:-}" in
    "" | your_oss_access_key | your_oss_secret_key | your-bucket-name | https://your-bucket.oss-cn-hangzhou.aliyuncs.com)
      return 0
      ;;
    *)
      return 1
      ;;
  esac
}

prompt_value() {
  local var_name="$1"
  local label="$2"
  local current="$3"
  local input=""
  if [[ -n "${!var_name:-}" ]] && ! is_placeholder "${!var_name}"; then
    return 0
  fi
  if is_placeholder "$current"; then
    current=""
  fi
  if [[ -t 0 ]]; then
    read -r -p "$label${current:+ [$current]}: " input
    printf -v "$var_name" '%s' "${input:-$current}"
  elif [[ -r /dev/tty ]]; then
    read -r -p "$label${current:+ [$current]}: " input </dev/tty
    printf -v "$var_name" '%s' "${input:-$current}"
  else
    printf -v "$var_name" '%s' "${!var_name:-$current}"
  fi
}

need_cmd curl
need_cmd tar
need_cmd docker
if ! docker compose version >/dev/null 2>&1; then
  echo "错误: 需要 Docker Compose v2（docker compose）"
  exit 1
fi

SRC_DIR=""
if [[ -n "${BASH_SOURCE[0]:-}" && -f "${BASH_SOURCE[0]}" ]]; then
  SRC_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
fi

if is_deploy_dir "$(pwd)"; then
  DEPLOY_DIR="$(pwd)"
elif [[ -n "$SRC_DIR" ]] && is_deploy_dir "$SRC_DIR"; then
  DEPLOY_DIR="$SRC_DIR"
else
  echo ">>> 下载发布包 $REPO@$VERSION"
  mkdir -p "$INSTALL_DIR"
  tmp="$(mktemp -d)"
  trap 'rm -rf "$tmp"' EXIT
  if [[ "$VERSION" == "latest" ]]; then
    PACK_URL="https://github.com/${REPO}/releases/latest/download/pandawiki-deploy.tar.gz"
  else
    PACK_URL="https://github.com/${REPO}/releases/download/${VERSION}/pandawiki-deploy.tar.gz"
  fi
  curl -fL "$PACK_URL" -o "$tmp/pandawiki-deploy.tar.gz"
  tar -xzf "$tmp/pandawiki-deploy.tar.gz" -C "$tmp"
  if [[ -d "$tmp/pandawiki-deploy" ]]; then
    cp -a "$tmp/pandawiki-deploy/." "$INSTALL_DIR/"
  else
    echo "错误: 发布包格式不正确"
    exit 1
  fi
  DEPLOY_DIR="$INSTALL_DIR"
fi

cd "$DEPLOY_DIR"
chmod +x generate-env.sh quickstart.sh backup.sh 2>/dev/null || true

if [[ ! -f .env ]]; then
  echo ">>> 生成 .env"
  FORCE=1 ./generate-env.sh .env
fi

IMAGE_TAG="$VERSION"
if [[ "$IMAGE_TAG" == "latest" && -f VERSION ]]; then
  IMAGE_TAG="$(tr -d '[:space:]' <VERSION)"
fi
if [[ -z "$IMAGE_TAG" ]]; then
  IMAGE_TAG="latest"
fi

upsert_env PANDAWIKI_IMAGE_REPO "$IMAGE_REPO" .env
upsert_env PANDAWIKI_IMAGE_TAG "$IMAGE_TAG" .env

# shellcheck disable=SC1091
set -a
source .env
set +a

echo ">>> 配置对象存储（OSS / S3）"
prompt_value S3_ACCESS_KEY "S3_ACCESS_KEY" "${S3_ACCESS_KEY:-}"
prompt_value S3_SECRET_KEY "S3_SECRET_KEY" "${S3_SECRET_KEY:-}"
prompt_value S3_ENDPOINT "S3_ENDPOINT" "${S3_ENDPOINT:-oss-cn-hangzhou.aliyuncs.com}"
prompt_value S3_BUCKET "S3_BUCKET" "${S3_BUCKET:-}"
prompt_value S3_REGION "S3_REGION" "${S3_REGION:-oss-cn-hangzhou}"
prompt_value S3_PUBLIC_BASE_URL "S3_PUBLIC_BASE_URL" "${S3_PUBLIC_BASE_URL:-}"
prompt_value S3_USE_SSL "S3_USE_SSL (1/0)" "${S3_USE_SSL:-1}"

upsert_env S3_ACCESS_KEY "$S3_ACCESS_KEY" .env
upsert_env S3_SECRET_KEY "$S3_SECRET_KEY" .env
upsert_env S3_ENDPOINT "$S3_ENDPOINT" .env
upsert_env S3_BUCKET "$S3_BUCKET" .env
upsert_env S3_REGION "$S3_REGION" .env
upsert_env S3_PUBLIC_BASE_URL "$S3_PUBLIC_BASE_URL" .env
upsert_env S3_USE_SSL "$S3_USE_SSL" .env

if [[ "$S3_ACCESS_KEY" == "your_oss_access_key" || -z "$S3_ACCESS_KEY" || -z "$S3_SECRET_KEY" || "$S3_BUCKET" == "your-bucket-name" || -z "$S3_BUCKET" ]]; then
  echo "错误: 请填写真实的 OSS/S3 凭证后再安装。"
  echo "  交互安装: bash $DEPLOY_DIR/install.sh"
  echo "  或先设置环境变量: S3_ACCESS_KEY S3_SECRET_KEY S3_BUCKET S3_PUBLIC_BASE_URL"
  exit 1
fi

echo ">>> 启动服务（镜像 $IMAGE_REPO/pandawiki-*:$IMAGE_TAG）"
./quickstart.sh
