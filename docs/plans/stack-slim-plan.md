# Linux 单机栈精简 — 开发计划

> 说明文档：[../STACK_SLIM.md](../STACK_SLIM.md)
> 分支：`feat/stack-slim`
> 状态（2026-08-13）：Phase 1–2 已落地；Phase 3（pgvector）未开始。
> 约束：保留 Redis；保留 Caddy；RAG 换 pgvector + 现有 embedding/rerank；目标 Linux 服务器。

## 一、原则

1. 业务逻辑仍在 `usecase/`，新 RAG 只加 `store/rag` 的实现，不把检索写进 handler。
2. 过渡期 `RAG_PROVIDER=ct|pg` 可切换，默认先 `ct`，切流完成后再改默认并下线 Raglite。
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

### Phase 3 — pgvector RAG（与 Raglite 并存）

**3.1 存储**

- Postgres 启用 `vector` 扩展。
- 表设计（名称可微调）：

| 表 | 用途 |
|---|---|
| `rag_chunks` | `id`, `kb_id`/`dataset_id`, `doc_id`, `node_id`, `content`, `embedding vector`, `group_ids`, `tags`, `seq` |
| `rag_jobs` | 异步切块/嵌入任务（`SKIP LOCKED`），替代 `VectorTaskTopic` |

- 向量维度跟当前 embedding 模型走（写入时记录 `dim`，换模型则全量重学）。

**3.2 实现 `store/rag/pg.go`**

满足现有 `RAGService`：

- `CreateKnowledgeBase` / `DeleteKnowledgeBase`
- `UpsertRecords`：HTML→MD（复用现有 converter）→ 按标题/token 切块 → 调 embedding API → upsert
- `QueryRecords`：问题 embedding → pgvector 近邻 → 可选全文召回合并 → **调用现有 rerank 模型** → 权限组过滤 → `TopK` / `MaxChunksPerDoc` / `SimilarityThreshold`
- 多轮：`HistoryMsgs` 非空时先用 chat 模型改写 query（对齐 Raglite `ChatHistory`）
- `UpsertModel`：pg 实现只校验并缓存模型配置，不再写入 Raglite

配置：`RAG_PROVIDER=pg` 时走新实现；`ct` 走旧 `ct.go`。

**3.3 切块与检索基线（保效果）**

必须同时具备，禁止「只做向量 TopK」上线：

1. 切块：优先 Markdown 标题，其次 token 上限（已有 `SplitByTokenLimit` 可作底）
2. rerank：沿用后台配置的 rerank 模型
3. 查询改写：多轮对话场景
4. `group_ids` 过滤（与现网权限一致）
5. 短词补充：`pg_trgm` 或简单全文，与向量结果融合后再 rerank

验收：同一知识库、同一 embedding/rerank，用固定 20～50 条问答抽样，人工对比 `ct` vs `pg` 的命中文档；明显回退则修切块/融合，不靠换库硬上。

### Phase 4 — 切流并下线 Raglite 依赖

1. 后台提供「全量重新学习」（遍历已发布文档写入 `rag_jobs`）。
2. 默认 `RAG_PROVIDER=pg`。
3. compose 去掉 `panda-wiki-qdrant`、`panda-wiki-raglite`。
4. 停止创建 `raglite` 数据库（`backend/store/pg/pg.go`）。
5. NATS：
   - 向量任务、Raglite 进度事件改为 `rag_jobs` + 文档状态字段。
   - Anydoc 导出完成改为任务表或短轮询；确认无主题后再从 compose 去掉 NATS。

验收：新部署无 Qdrant/Raglite；旧数据学习完成后问答可用；文档导入/发布仍异步可查进度。

### Phase 5 — 可选：本地文件替换 MinIO

- `store/s3` 增加 `file` backend（或 MinIO 指向本地路径）。
- 单机 Linux 默认本地盘；需要对象存储时仍可用 S3/MinIO。
- 与 Phase 3/4 解耦，避免文件迁移和向量迁移同时炸。

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
| 5 本地盘 | 中 | 中 | 中 |

Linux 生产建议：**1 → 2 → 3（灰度）→ 4**。第 5 步按需。

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

- [ ] `go test` 覆盖切块、权限过滤、空库检索
- [ ] 切换 `RAG_PROVIDER` 不影响 chat/embedding 配置读写
- [ ] 发布文档后任务成功，chunk 可查
- [ ] 权限组外的文档不会出现在检索结果
- [ ] 抽样问答与 Raglite 对比无明显回退
- [ ] Image compose 在干净 Linux 上可启动（阶段 4 后无 Qdrant/Raglite）
- [ ] `cd backend && make lint` 在涉及 Go 的阶段结束时通过
