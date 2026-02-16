# 乘风版部署指南（两种方式）

本文提供两种可落地部署方式：

1. 手动安装环境 + 服务器源码构建部署（Build 模式）
2. 方案 B：预构建镜像部署（Image 模式，推荐生产）

## 1. 方式对比

| 方式 | 适用场景 | 优点 | 注意事项 |
| --- | --- | --- | --- |
| 手动安装环境 + Build 模式 | 开发、联调、快速验证 | 改完代码即可在服务器本地构建 | 首次配置步骤较多；构建耗时较长 |
| 方案 B（预构建镜像） | 生产环境、稳定交付 | 发布快、可回滚、服务器负载低 | 需要先在 CI 产出镜像 |

## 2. 环境清单与推荐版本

### 2.1 服务器资源建议

- 最低配置：`4 vCPU / 8 GB RAM / 80 GB SSD`
- 推荐配置：`8 vCPU / 16 GB RAM / 160 GB SSD`
- 操作系统：`Debian 12` 或 `Ubuntu 22.04+`

### 2.2 组件版本

| 组件 | 推荐版本 | 说明 |
| --- | --- | --- |
| Docker Engine | `24.x+` | 两种部署方式都需要 |
| Docker Compose Plugin | `v2.24+` | 使用 `docker compose` 命令 |
| Git | `2.30+` | 拉取代码 |
| Node.js | `22.x` | Build 模式下构建前端产物 |
| pnpm | `10.x` | Build 模式下前端构建 |
| PostgreSQL | `16-alpine`（容器） | 主数据库 |
| Redis | `7-alpine`（容器） | 缓存/限流 |
| NATS | `2.10-alpine`（容器） | 消息队列 |
| MinIO | `latest`（容器） | 对象存储 |
| Qdrant | `v1.14.1`（容器） | 向量检索 |
| Raglite | `v2.14.1`（容器） | RAG 服务 |
| Caddy | `2.10-alpine`（容器） | 域名路由与访问入口配置 |

## 3. 通用准备

### 3.1 拉取代码

```bash
git clone https://github.com/MaydayV/PandaWiki.git
cd PandaWiki
```

### 3.2 准备部署变量

```bash
cd docs/deploy
cp .env.example .env
```

修改 `.env` 至少包含以下值：

- `POSTGRES_PASSWORD`
- `REDIS_PASSWORD`
- `S3_SECRET_KEY`
- `NATS_PASSWORD`
- `QDRANT_API_KEY`
- `JWT_SECRET`
- `ADMIN_PASSWORD`
- `DEV_KB_ID`

### 3.3 首次部署初始化数据库（仅首次）

> 当前项目使用完整部署 SQL：`backend/store/pg/migration/full_fresh_deploy.sql`

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

如果使用方案 B，可将命令中的 `docker-compose.build.yml` 替换为 `docker-compose.image.yml`。

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

> `web/admin` 与 `web/app` 的 Dockerfile 会直接复制 `dist`，因此先构建前端产物。

```bash
cd ../../web
pnpm install --frozen-lockfile
NODE_OPTIONS=--max-old-space-size=4096 pnpm --filter panda-wiki-admin build
pnpm --filter panda-wiki-app build
cd ../docs/deploy
```

### 4.3 启动全部服务（Build）

```bash
docker compose -f docker-compose.build.yml up -d --build
```

### 4.4 验证

```bash
docker compose -f docker-compose.build.yml ps
curl -sS http://127.0.0.1:8000/health
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

- `PANDAWIKI_IMAGE_REPO=ghcr.io/maydayv`
- `PANDAWIKI_IMAGE_TAG=<发布标签>`

例如：

```env
PANDAWIKI_IMAGE_REPO=ghcr.io/maydayv
PANDAWIKI_IMAGE_TAG=FV2.6.1.2111
```

如镜像仓库为私有，先登录：

```bash
docker login ghcr.io
```

### 5.2 启动

```bash
docker compose -f docker-compose.image.yml pull
docker compose -f docker-compose.image.yml up -d
```

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

## 6. 访问说明

- 后台管理：`https://<server-ip>:2443`
- 前台站点：`http://<server-ip>:3010`
- API 健康检查：`http://<server-ip>:8000/health`

## 7. 安全建议（生产必做）

1. `.env` 中全部密码改为高强度随机值，禁止使用示例密码。
2. 通过防火墙限制 `8000/3010/2443` 暴露范围，建议仅暴露反向代理端口。
3. 为后台域名配置真实 TLS 证书，不使用自签证书对外提供服务。
4. 定期备份：PostgreSQL 数据卷、MinIO 数据卷、`docs/deploy/.env`。
5. 按 AGPL-3.0 要求提供当前运行版本对应源码链接。

## 8. 相关文件

- Build 模式编排：`docs/deploy/docker-compose.build.yml`
- Image 模式编排：`docs/deploy/docker-compose.image.yml`
- 部署变量模板：`docs/deploy/.env.example`
- 首次完整 SQL：`backend/store/pg/migration/full_fresh_deploy.sql`
