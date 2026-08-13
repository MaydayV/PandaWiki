# 乘风版部署指南（Docker 多方案）

本文提供三种可落地的 Docker 部署方案：

1. 手动安装环境 + 服务器源码构建部署（Build 模式）
2. 方案 B：预构建镜像部署（Image 模式，推荐生产）
3. 外层 Nginx + 内层 Caddy（推荐公网生产）

本文描述 Docker 部署的三种方式。**乘风版默认编排**（`feat/stack-slim`）为 **6 容器**：Caddy + Postgres + Redis + NATS + API（含 Consumer/Admin）+ App；向量检索在 Postgres pgvector，附件走 OSS。详见 [STACK_SLIM.md](./STACK_SLIM.md)。

## 1. 方式对比

| 方式 | 适用场景 | 优点 | 注意事项 |
| --- | --- | --- | --- |
| 手动安装环境 + Build 模式 | 开发、联调、快速验证 | 改完代码即可在服务器本地构建 | 首次配置步骤较多；构建耗时较长 |
| 方案 B（预构建镜像） | 生产环境、稳定交付 | 发布快、可回滚、服务器负载低 | 需要先在 CI 产出镜像 |
| 外层 Nginx + 内层 Caddy | 生产环境、统一 80/443 出口 | 可以复用现有 Nginx 证书与网关体系 | 不能移除 Caddy，Nginx 仅做外层入口 |

## 2. 环境清单与推荐版本

### 2.1 服务器资源建议

- 最低配置：`4 vCPU / 8 GB RAM / 80 GB SSD`
- 推荐配置：`8 vCPU / 16 GB RAM / 160 GB SSD`
- 操作系统：`Debian 12` 或 `Ubuntu 22.04+`

### 2.2 组件版本

| 组件 | 推荐版本 | 说明 |
| --- | --- | --- |
| Docker Engine | `24.x+` | 三种部署方式都需要 |
| Docker Compose Plugin | `v2.24+` | 使用 `docker compose` 命令 |
| Git | `2.30+` | 拉取代码 |
| Node.js | `22.x` | 仅 Build 模式需要 |
| pnpm | `10.x` | 仅 Build 模式需要 |
| PostgreSQL | `pgvector/pgvector:pg16`（容器） | 主数据库 + 向量检索（默认 RAG） |
| Redis | `7-alpine`（容器） | 缓存/限流 |
| NATS | `2.10-alpine`（容器） | Anydoc 导出完成事件；pg 模式向量任务已改 `rag_jobs` 表 |
| 阿里云 OSS | 外部服务 | 图片/附件对象存储（S3 兼容 API） |
| Caddy | `2.10-alpine`（容器） | 知识库访问规则与动态路由 |

## 3. 通用准备

### 3.1 拉取代码

```bash
git clone https://github.com/MaydayV/PandaWiki.git
cd PandaWiki
```

### 3.2 准备部署变量

**推荐（Linux 生产 Image 模式）：**

```bash
cd docs/deploy
./generate-env.sh          # 自动生成随机密钥写入 .env
# 按需编辑 PANDAWIKI_IMAGE_TAG 等
./quickstart.sh            # 拉镜像 → 初始化库 → 启动全部服务
```

手动方式：

```bash
cd docs/deploy
cp .env.example .env
```

修改 `.env` 至少包含以下值（`generate-env.sh` 会自动生成）：

- `POSTGRES_PASSWORD`
- `REDIS_PASSWORD`
- `S3_ACCESS_KEY`、`S3_SECRET_KEY`、`S3_ENDPOINT`、`S3_BUCKET`
- `S3_PUBLIC_BASE_URL`（OSS 桶公网 URL 或 CDN 域名）
- `NATS_PASSWORD`
- `JWT_SECRET`
- `ADMIN_PASSWORD`

可选：

- `DEV_KB_ID` — 生产经 Caddy 注入 `X-KB-ID` 时可留空
- `RUN_WORKER` — 默认 `1`，API 内嵌 MQ 消费者与定时任务
- `ADMIN_ENABLED` — 默认 `1`，API 内嵌 HTTPS 管理端（2443）
- `RAG_PG_EMBEDDING_DIM` — pgvector 嵌入维度，默认 `1024`（须与 embedding 模型一致）

仅 Image 模式额外需要：

- `PANDAWIKI_IMAGE_REPO`
- `PANDAWIKI_IMAGE_TAG`

### 3.3 首次部署初始化数据库（仅首次）

> 当前项目使用**单一**完整部署 SQL（无增量迁移文件）：`backend/store/pg/migration/full_fresh_deploy.sql`  
> 推荐：`docs/deploy/quickstart.sh` 检测空库后自动导入。

先启动 PostgreSQL：

```bash
docker compose -f docker-compose.build.yml up -d panda-wiki-postgres
```

导入完整 SQL：

```bash
cat ../../backend/store/pg/migration/full_fresh_deploy.sql | \
docker compose -f docker-compose.build.yml exec -T panda-wiki-postgres \
psql -U panda-wiki -d panda-wiki
```

如果使用 Image 模式，可将命令中的 `docker-compose.build.yml` 替换为 `docker-compose.image.yml`。

### 3.4 端口职责与流量路径（当前默认）

- `3010`：前台入口，由 `panda-wiki-caddy` 监听并反向代理到 `api/app`；`/static-file/*` 经 API 反代 OSS。
- `2443`：后台管理入口（内嵌于 `panda-wiki-api`，HTTPS）。
- `8000`：API 直连入口（通常用于健康检查、联调）。
- `5432/6379/4222`：数据库与中间件内部端口，默认不对公网暴露。

说明：`panda-wiki-app` 当前只在容器网络 `expose 3010`，不会直接映射宿主机 `3010`，避免与 Caddy 冲突。

### 3.5 从旧编排升级到当前网络模型（执行一次）

如果你之前使用过旧版本编排（`app` 直映射 `3010`），升级到当前版本建议先执行一次完整重建：

```bash
docker compose -f docker-compose.build.yml down --remove-orphans
docker compose -f docker-compose.build.yml up -d --build
```

Image 模式同理：

```bash
docker compose -f docker-compose.image.yml down --remove-orphans
docker compose -f docker-compose.image.yml pull
docker compose -f docker-compose.image.yml up -d
```

## 4. 方式一：手动安装环境 + 服务器源码构建部署（Build 模式）

### 4.1 安装基础环境（Debian/Ubuntu）

```bash
sudo apt-get update
sudo apt-get install -y ca-certificates curl gnupg lsb-release git
```

安装 Docker：

```bash
sudo install -m 0755 -d /etc/apt/keyrings
curl -fsSL https://download.docker.com/linux/debian/gpg | \
  sudo gpg --dearmor -o /etc/apt/keyrings/docker.gpg
sudo chmod a+r /etc/apt/keyrings/docker.gpg

echo \
  "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] \
  https://download.docker.com/linux/debian \
  $(. /etc/os-release && echo \"$VERSION_CODENAME\") stable" | \
  sudo tee /etc/apt/sources.list.d/docker.list > /dev/null

sudo apt-get update
sudo apt-get install -y docker-ce docker-ce-cli containerd.io docker-compose-plugin
sudo systemctl enable docker
sudo systemctl start docker
```

安装 Node.js 22 与 pnpm：

```bash
curl -fsSL https://deb.nodesource.com/setup_22.x | sudo -E bash -
sudo apt-get install -y nodejs
corepack enable
corepack prepare pnpm@10.12.1 --activate
```

### 4.2 构建前端产物（必须）

> `web/admin` 与 `web/app` 的 Dockerfile 会复制构建产物，因此先构建前端。

```bash
cd ../../web
pnpm install --frozen-lockfile
NODE_OPTIONS=--max-old-space-size=4096 pnpm --filter panda-wiki-admin build
pnpm --filter panda-wiki-app build
cd ../docs/deploy
```

如果遇到 `ERR_PNPM_LOCKFILE_CONFIG_MISMATCH`：

```bash
pnpm install --no-frozen-lockfile
```

然后继续构建流程。

### 4.3 启动全部服务（Build）

```bash
docker compose -f docker-compose.build.yml up -d --build
```

### 4.4 验证

```bash
docker compose -f docker-compose.build.yml ps
curl -sS --retry 10 --retry-delay 2 --retry-connrefused http://127.0.0.1:8000/health
curl -k -I https://127.0.0.1:2443 | head -n 5
curl -I http://127.0.0.1:3010 | head -n 5
```

### 4.5 日常更新

```bash
cd ../..
git pull origin main
cd web
pnpm install --frozen-lockfile
NODE_OPTIONS=--max-old-space-size=4096 pnpm --filter panda-wiki-admin build
pnpm --filter panda-wiki-app build
cd ../docs/deploy
docker compose -f docker-compose.build.yml up -d --build
```

## 5. 方式二：方案 B（预构建镜像部署，推荐生产）

方案 B 使用 `docs/deploy/docker-compose.image.yml`，仅拉取镜像，不在服务器编译。

### 5.1 准备镜像变量

编辑 `docs/deploy/.env`：

- `PANDAWIKI_IMAGE_REPO=docker.io/caodanv`
- `PANDAWIKI_IMAGE_TAG=<发布标签>`

例如：

```env
PANDAWIKI_IMAGE_REPO=docker.io/caodanv
PANDAWIKI_IMAGE_TAG=FV2.6.14.2111
```

如镜像仓库为私有，先登录：

```bash
docker login
```

### 5.2 启动

一键（推荐）：

```bash
cd docs/deploy
./quickstart.sh
```

或手动（**必须先导入 SQL**，空库不能直接 `up -d`）：

```bash
cd docs/deploy
./generate-env.sh   # 首次
docker compose -f docker-compose.image.yml pull
docker compose -f docker-compose.image.yml up -d panda-wiki-postgres
# 等待 postgres 就绪后，空库导入：
cat ../../backend/store/pg/migration/full_fresh_deploy.sql | \
  docker compose -f docker-compose.image.yml exec -T panda-wiki-postgres \
  psql -U panda-wiki -d panda-wiki
docker compose -f docker-compose.image.yml up -d
```

推荐仍用 `./quickstart.sh`，会自动检测空库并导入 SQL。

默认 **6 个容器**（Caddy + Postgres + Redis + NATS + API + App；向量在 pgvector，附件在 OSS）。

### 5.3 升级发布

1. 修改 `.env` 中 `PANDAWIKI_IMAGE_TAG` 为新版本。
2. 执行：

```bash
docker compose -f docker-compose.image.yml pull
docker compose -f docker-compose.image.yml up -d
```

### 5.4 回滚

1. 将 `PANDAWIKI_IMAGE_TAG` 改回上一版本。
2. 执行：

```bash
docker compose -f docker-compose.image.yml pull
docker compose -f docker-compose.image.yml up -d
```

### 5.5 CI 自动发布到 Docker Hub（推荐）

当前仓库已配置 GitHub Actions 自动推送四个镜像（`api/consumer/app/admin`）到 Docker Hub。

先在 GitHub 仓库设置 Secrets：

- `DOCKERHUB_USERNAME`：Docker Hub 用户名（例如 `caodanv`）
- `DOCKERHUB_TOKEN`：Docker Hub Access Token（不要用明文密码）

发布步骤：

```bash
git checkout main
git pull origin main
git tag v2.6.2
git push origin v2.6.2
```

推送完成后会自动发布：

- `docker.io/caodanv/pandawiki-api:v2.6.2`
- `docker.io/caodanv/pandawiki-consumer:v2.6.2`
- `docker.io/caodanv/pandawiki-app:v2.6.2`
- `docker.io/caodanv/pandawiki-admin:v2.6.2`

## 6. 方式三：外层 Nginx + 内层 Caddy（推荐公网生产）

适用场景：你需要统一 80/443 出口、复用现有 Nginx 证书和 WAF/CDN 策略。

### 6.1 核心原则

- 继续使用当前 Docker 编排（Build 或 Image 均可）。
- 保留 `panda-wiki-caddy`，不要替换掉它。
- Nginx 仅做外层入口，将流量转发到本机 `3010/2443`。

### 6.2 Nginx 参考配置

```nginx
server {
    listen 80;
    server_name wiki.example.com;
    return 301 https://$host$request_uri;
}

server {
    listen 443 ssl http2;
    server_name wiki.example.com;

    ssl_certificate     /etc/nginx/certs/wiki.crt;
    ssl_certificate_key /etc/nginx/certs/wiki.key;

    location / {
        proxy_pass http://127.0.0.1:3010;
        proxy_set_header Host $host;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    }
}

server {
    listen 443 ssl http2;
    server_name admin.example.com;

    ssl_certificate     /etc/nginx/certs/admin.crt;
    ssl_certificate_key /etc/nginx/certs/admin.key;

    location / {
        proxy_pass https://127.0.0.1:2443;
        proxy_ssl_verify off;
        proxy_set_header Host $host;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    }
}
```

### 6.3 为什么不能直接用 Nginx 平替 Caddy

后端会按知识库配置动态下发访问规则，并依赖路由层自动注入 `X-KB-ID`。当前实现直接调用 Caddy Admin API（Unix Socket）动态更新配置。若直接移除 Caddy，需要额外开发一套 Nginx 动态配置与热更新机制。

## 7. 访问说明

直连模式（无外层 Nginx）：

- 后台管理：`https://<server-ip>:2443`
- 前台站点：`http://<server-ip>:3010`
- API 健康检查：`http://<server-ip>:8000/health`

外层 Nginx 模式（推荐）：

- 前台站点：`https://wiki.example.com`
- 后台管理：`https://admin.example.com`

## 8. 安全建议（生产必做）

1. `.env` 中全部密码改为高强度随机值，禁止使用示例密码。
2. 外层 Nginx 模式下，建议只对公网开放 `80/443`，限制 `8000/2443/3010` 仅本机或内网访问。
3. 为对外域名配置真实 TLS 证书，不使用自签证书直接暴露公网。
4. 定期备份：PostgreSQL 数据卷、OSS 桶数据、`docs/deploy/.env`。
5. 按 AGPL-3.0 要求提供当前运行版本对应源码链接。

## 9. 相关文件

- Build 模式编排：`docs/deploy/docker-compose.build.yml`
- Image 模式编排：`docs/deploy/docker-compose.image.yml`
- 部署变量模板：`docs/deploy/.env.example`
- 一键生成密钥：`docs/deploy/generate-env.sh`
- Image 模式一键部署：`docs/deploy/quickstart.sh`
- 首次完整 SQL：`backend/store/pg/migration/full_fresh_deploy.sql`
