# Linux 单机栈精简 — 开发计划

> 说明文档：[../STACK_SLIM.md](../STACK_SLIM.md)
> 分支：`feat/stack-slim`
> 状态（2026-08-13）：**Phase 1–5 已完成**。默认 6 容器 + OSS + pgvector RAG。可选后续：去 NATS（Anydoc）、`pg_trgm` 增强。
> 约束：保留 Redis；保留 Caddy；目标 Linux 服务器。

## 一、原则

1. 业务逻辑仍在 `usecase/`，RAG 实现在 `store/rag/pg.go`。
2. **仅 `RAG_PROVIDER=pg`**；已移除 Raglite/Qdrant/MinIO 及独立 Consumer/Admin 容器。
3. 每阶段可单独上线；pg 模式切换需全量重新学习。

## 二、阶段（均已完成）

### Phase 1 — 部署减负

- [x] `generate-env.sh` / `quickstart.sh` / `.env.example`
- [x] compose 健康检查与启动顺序
- [x] `DEPLOYMENT.md`

### Phase 2 — 减进程

- [x] Consumer 并进 API（`RUN_WORKER=1`）
- [x] Admin 内嵌 API HTTPS（2443）
- [x] 移除独立 Consumer / Admin compose 服务

### Phase 3 — pgvector RAG

- [x] `full_fresh_deploy.sql`（vector + `rag_chunks` + `rag_jobs`）
- [x] `store/rag/pg.go` + 切块单测
- [ ] `pg_trgm` 短词融合（可选增强）

### Phase 4 — 下线 Raglite

- [x] 全量重新学习 API + Admin 入口
- [x] `rag_jobs` 表 worker（`SKIP LOCKED`）
- [x] 移除 Qdrant/Raglite 代码与 compose

### Phase 5 — OSS 对象存储

- [x] 默认 OSS；API 反代 `/static-file/*`
- [x] compose 固定 6 容器

## 三、不在本计划内

- 去掉 Redis / 改纯内存
- 用 Nginx 替换 Caddy
- 合并 Next 前台与 Vite 后台
- Milvus / Weaviate / ES / RAGFlow / Dify

## 四、当前 compose（6 容器）

`Caddy + Postgres + Redis + NATS + API + App`

## 五、关键代码锚点

| 模块 | 路径 |
|---|---|
| RAG 接口 | `backend/store/rag/rag.go` |
| pgvector 实现 | `backend/store/rag/pg.go` |
| 向量任务 worker | `backend/handler/mq/rag_job_worker.go` |
| OSS / S3 | `backend/store/s3/`、`backend/config/s3.go` |
| Compose | `docs/deploy/docker-compose.image.yml` |

## 六、测试与验收清单

- [x] `go test ./...`（切块、S3 URL 等）
- [x] compose 6 容器、无 legacy profile
- [x] OSS `/static-file` 反代链路（代码就绪）
- [ ] 权限组检索隔离（需部署验收）
- [ ] 抽样问答效果（需部署验收）
- [ ] `make lint`（CI `backend_check.yml` 可跑 golangci-lint）
