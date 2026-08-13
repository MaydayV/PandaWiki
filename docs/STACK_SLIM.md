# 乘风版部署精简说明

> 目标环境：**Linux 单机服务器**（不是开发机、不是 Mac mini）。
> 目标：**降低内存/CPU 占用**，并让部署变成「一条 compose 命令」。
> 状态：Phase 1–4 已落地；对象存储默认 **阿里云 OSS**（无 MinIO 容器）。全新部署用 `full_fresh_deploy.sql`；默认 pgvector RAG、**6 容器**。见 [plans/stack-slim-plan.md](./plans/stack-slim-plan.md)。

## 1. 为什么要改

当前一次部署要拉起约 **11 个容器**：

| 层 | 组件 | 问题 |
|---|---|---|
| 应用 | API、Consumer、Next 前台、Admin Nginx | 进程重复、端口多 |
| 网关 | Caddy（host 网络 + Unix Socket） | 动态路由难配，但不能删 |
| 数据 | Postgres、Redis | Redis 很轻；附件走 **OSS**（无本地 MinIO） |
| RAG | NATS、Qdrant、Raglite | **最占内存**，且 Raglite 还依赖上面三套中间件 |

Raglite 启动依赖 Postgres、MinIO、Qdrant、NATS。只要保留这套 RAG，NATS / Qdrant / MinIO 就去不干净。

部署痛点不只是「容器多」，还包括：服务器上先 `pnpm build`、一堆必填密码、Caddy host 网络、Admin 额外 `2443` 端口。

## 2. 明确不改什么

| 项 | 原因 |
|---|---|
| Postgres | 主库，必须留 |
| Redis | Session、登录锁定、企微/钉钉 token、跨请求锁。单机也只占几十 MB，**不换成内存** |
| Caddy | 按知识库动态写域名/端口/证书并注入 `X-KB-ID`，用 Nginx 平替等于重做网关 |
| 前台 Next.js | SSR / SEO 需要 Node 进程，现阶段不改成纯静态 |
| 后台 Vite + 现有 UI | 不把 admin/app 合成一个前端 |
| 嵌入 / 重排模型 | 继续用后台已配置的 embedding、rerank（自动模式默认 rerank 为 `bge-reranker-v2-m3`） |

## 3. 目标形态（Linux 单机）

```
现在：Caddy + Postgres + Redis + MinIO + NATS + Qdrant + Raglite
      + API + Consumer + App + Admin
      ≈ 11 容器

目标：Caddy + Postgres + Redis + NATS + API（含 Consumer/Admin）+ App
      + OSS（外部）
      = 6 容器
```

Phase 1–5 已完成：**6 容器**（Caddy + Postgres + Redis + NATS + API + App）。

流量仍走 Caddy → App / API。Admin 静态资源改由 Caddy 或 API 托管，不再单独起 Nginx。

## 4. RAG 换成什么

**Postgres `pgvector` + 现有 embedding/rerank HTTP API，检索做进 API 进程。**

当前问答链路：

```
文档 → Raglite 切块并向量化 → Qdrant
提问 → Raglite 检索（可改写、阈值、权限组）→ chunk → LLM 回答
```

效果主要由 **切块质量、嵌入模型、rerank、查询改写** 决定，不是 Qdrant 这个库。Wiki 规模（千～万级文档）下，pgvector 与 Qdrant 的检索差异通常体感不明显。

| 能力 | 现在 | 换成 pgvector |
|---|---|---|
| 向量存储 | Qdrant | 与业务同一 Postgres |
| 嵌入 / 重排 | 后台配置的模型 | **同一套模型** |
| 权限组 | metadata `group_ids` | SQL `WHERE` |
| 多轮改写 | Raglite 吃对话历史 | 检索前调一次 chat |
| 切块 | Raglite 内部 | API 内按标题/token 切 |
| 中文短词 | 主要靠向量 | 向量 + Postgres 全文（`pg_trgm` 或简单分词） |

代码已有 `RAGService` 接口（`backend/store/rag`）。默认 `pg` 实现；compose 不再包含 Qdrant/Raglite。

### 对问答效果的预期

- **模型和切块、rerank、改写都对齐**：体感可接近现状。
- **必须做的**：全库重新「学习」（向量不能从 Qdrant 直接搬）。
- **会掉效果的偷工**：切块过粗/过碎、去掉 rerank、不做查询改写、不做权限过滤。
- **可能变好的点**：专有名词、短关键词（补了全文检索之后）。

不采用：Milvus / Weaviate / ES（更重）、SQLite 向量（以后不好扩）、RAGFlow / Dify（服务更多）。

## 5. 其它精简

**Consumer 并进 API**  
pg 模式下文档向量任务写入 Postgres `rag_jobs` 表，API 内 goroutine 消费；Anydoc 导出完成仍走 NATS。去 NATS 需等 Anydoc 也改任务表。

**Admin 不再独立容器**  
`web/admin` 构建产物挂到 Caddy 或由 API 提供静态文件，去掉 `2443` 和 Admin Nginx。

**对象存储（OSS）**  
默认使用 S3 兼容 API 连接阿里云 OSS。API 反代 `/static-file/*` 到 OSS；Caddy 静态路由指向 API。桶需预先创建并配置公共读或 CDN。

**Redis**  
保留。去 Redis 省不下有意义的内存，却要重做 Session 和限流。

## 6. 资源与部署预期

| 指标 | 现在 | 目标 |
|---|---|---|
| 容器数 | ~11 | 6 |
| 内存（经验） | Qdrant+Raglite 常占 1GB+，整体轻松 3GB+ | 主要剩 Postgres + Next + API |
| 必填密钥 | Postgres/Redis/S3/NATS/Qdrant/JWT/Admin | Postgres/Redis/S3/OSS/NATS/JWT/Admin |

Linux 建议仍按现有文档：**4C8G 起步，8C16G 更从容**。精简后 4C8G 会明显好过现在。

## 7. 风险

1. 与上游 Raglite 分叉：以后合上游时要保住 `RAGService` 适配层，不要把 pg 实现写进 handler。
2. 切换窗口：旧向量不可用，需要后台「全量重新学习」。
3. Anydoc 爬虫目前也走 NATS；去 NATS 时必须一起改成任务表，否则导入文档会断。
4. Caddy 动态路由第一期不动，避免网关与 RAG 两场重构叠在一起。

## 8. 相关文件

- 实现计划：[plans/stack-slim-plan.md](./plans/stack-slim-plan.md)
- 现有部署：[DEPLOYMENT.md](./DEPLOYMENT.md)
- RAG 接口：`backend/store/rag/rag.go`
- 现实现：`backend/store/rag/ct.go`（Raglite）
