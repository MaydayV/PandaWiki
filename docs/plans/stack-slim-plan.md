# Linux 单机栈精简 — 开发计划

> 说明文档：[../STACK_SLIM.md](../STACK_SLIM.md)
> 分支：`feat/stack-slim`
> 状态（2026-08-13）：Phase 1–4 已落地（全新部署路径；默认 pgvector + 7 容器）。Phase 5 / 去 NATS（Anydoc）为可选后续。
> 约束：保留 Redis；保留 Caddy；RAG 换 pgvector + 现有 embedding/rerank；目标 Linux 服务器。

## 一、原则

1. 业务逻辑仍在 `usecase/`，新 RAG 只加 `store/rag` 的实现，不把检索写进 handler。
2. 过渡期 `RAG_PROVIDER=ct|pg` 可切换；**默认 `pg`**。回退 Raglite 需启用 compose profile `legacy-rag` 并设 `RAG_PROVIDER=ct`。
3. 不把「导出/SEO 那批未提交改动」混进本分支（已 stash）。
4. 每阶段可单独上线；阶段 3 之前不要求用户全量重新学习。

## 二、阶段

### Phase 1 — 部署减负（不换架构）

**目标**：现有 11 容器也能「填最少变量就启动」。

- [x] `docs/deploy/generate-env.sh` 自动生成随机密钥
- [x] `docs/deploy/quickstart.sh` Image 模式一键部署
- [x] `docs/deploy/.env.example` 注释与 `RUN_WORKER`
- [x] compose 健康检查（Redis/MinIO/NATS/Qdrant）与启动顺序
- [x] `docs/DEPLOYMENT.md` 更新

验收：新服务器按文档用镜像 compose 拉起，不必在服务器上跑 `pnpm build`。

### Phase 2 — 减进程

**2.1 Consumer 并进 API** — [x]

- [x] `RUN_WORKER` 环境变量（默认 `1`）
- [x] API wire 注入 MQ handlers + CronHandler
- [x] `CronHandler.Start()` 延迟启动，避免 API 未开 worker 时跑 cron
- [x] compose 默认不启 Consumer；`split-worker` profile 保留回退

**2.2 Admin 静态合并** — [x]

- [x] API 内嵌 HTTPS 管理端（`server/http/admin_server.go`，端口 `2443`）
- [x] 复用 `/app/etc/nginx/ssl` 自签证书（`setup.CheckInitCert`）
- [x] `Dockerfile.api` 打包 `web/admin/dist`；compose 默认映射 `2443` 到 API
- [x] 独立 `panda-wiki-admin` 改为 profile `legacy-admin`（端口 `2444` 避免冲突）

验收：浏览器访问 `https://<server>:2443` 打开后台；`docker compose ps` 默认无 Admin 容器。

### Phase 3 — pgvector RAG（与 Raglite 并存）— [x]

- [x] `full_fresh_deploy.sql` 含 `vector` 扩展 + `rag_chunks` + `rag_jobs`（无增量 `*.up.sql`）
- [x] Postgres 镜像换 `pgvector/pgvector:pg16`
- [x] `store/rag/pg.go` 实现 `RAGService`（切块、嵌入、向量检索、rerank、多轮改写）
- [x] `RAG_PROVIDER=ct|pg`（**默认 `pg`**）；`RAG_PG_EMBEDDING_DIM`（默认 1024）
- [x] `store/rag/chunk.go` Markdown 切块 + 单测
- [x] 全量重新学习 API + Admin 入口（Phase 4）
- [ ] `pg_trgm` 短词融合（可选增强）

验收：同一 embedding/rerank 配置下，`pg` 模式发布文档后可检索；切换需全量 re-ingest（Phase 4 切流）。

### Phase 4 — 切流并下线 Raglite 依赖 — [x]（NATS 任务表待后续）

1. [x] 后台「全量重新学习」：`POST /api/v1/knowledge_base/rag/reindex` + Admin 设置页按钮
2. [x] 默认 `RAG_PROVIDER=pg`
3. [x] compose 将 `panda-wiki-qdrant`、`panda-wiki-raglite` 移入 profile `legacy-rag`（新部署默认 **7 容器**）
4. [x] `RAG_PROVIDER=pg` 时不再创建 `raglite` 数据库（`backend/store/pg/pg.go`）
5. [x] pg 模式向量任务改为 Postgres `rag_jobs` 表（`SKIP LOCKED` 轮询）；Anydoc 导出仍走 NATS

验收：新部署无 Qdrant/Raglite；从 Raglite 迁移或新建 pg 索引后，全量重新学习完成则问答可用。

### Phase 5 — 对象存储改用 OSS（已完成）

- [x] 默认栈去掉 MinIO 容器；`store/s3` 支持 S3 兼容 OSS（DNS 寻址、公网 URL）。
- [x] API/Caddy/Admin/App 的 `/static-file/*` 统一经 API 反代 OSS。
- [x] Legacy Raglite（`legacy-rag` profile）仍保留 MinIO 供 Raglite 使用。

验收：compose 默认 6 容器；配置 `S3_*` 后上传与静态访问走 OSS。

## 三、不在本计划内

- 把后端改成 Node/Python
- 去掉 Redis / 改纯内存
- 用 Nginx 替换 Caddy
- 合并 Next 前台与 Vite 后台为一个应用
- 引入 Milvus、Weaviate、ES、RAGFlow、Dify

## 四、建议顺序与工作量

| 阶段 | 工作量（约） | 省资源 | 更好部署 |
|---|---|---|---|
| 1 部署减负 | 小 | 无 | 高 |
| 2 减进程 | 中 | 中（少一个 Go + 一个 Nginx） | 高 |
| 3 pgvector 并存 | 大 | 尚无（两套 RAG 会暂时更重） | 低 |
| 4 下线 Raglite | 中 | **最高** | 高 |
| 5 OSS 对象存储 | 中 | 中（无 MinIO 容器） | 高 |

Linux 生产建议：**1 → 2 → 3（灰度）→ 4 → 5**。OSS 与 pg RAG 可并行配置。

## 五、关键代码锚点

| 模块 | 路径 |
|---|---|
| RAG 接口 | `backend/store/rag/rag.go` |
| Raglite 实现 | `backend/store/rag/ct.go` |
| 检索调用 | `backend/usecase/llm.go` `GetRankNodes` |
| 向量任务投递 | `backend/repo/mq/rag.go` |
| Consumer | `backend/cmd/consumer`、`backend/handler/mq` |
| 模型同步到 RAG | `backend/usecase/model.go` `UpsertModel` |
| 创建 raglite 库 | `backend/store/pg/pg.go` |
| Admin Nginx | `web/admin/Dockerfile`、`web/admin/server.conf` |
| Compose | `docs/deploy/docker-compose.image.yml` |

## 六、测试与验收清单

- [x] `go test` 覆盖切块（`store/rag/chunk_test.go`）
- [x] 切换 `RAG_PROVIDER` 不影响 chat/embedding 配置读写（默认 pg）
- [x] RAG job 失败重试与节点 `rag_info` 状态同步
- [x] compose `split-worker` 在 pg 模式可解析
- [x] 空库未导入 SQL 时 API fail-fast 提示
- [x] `Dockerfile.api` 多阶段构建 admin
- [ ] 权限组外的文档不会出现在检索结果（需验收）
- [ ] 抽样问答无明显回退（需验收）
- [x] Image compose 新部署无 Qdrant/Raglite/MinIO（profile 默认不启用）
- [x] 默认对象存储走 OSS（`S3_*` + API `/static-file` 反代）
- [ ] `make lint`（本机未装 golangci-lint/swag；`go test ./...` 已通过）
