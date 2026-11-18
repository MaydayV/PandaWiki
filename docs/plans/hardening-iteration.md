# 加固迭代计划（2026-11-19）

> 目标：部署更方便、性能更好、更稳定。每轮独立提交，便于回退；只提交不推送。
> 勘察来源：两轮全量勘察（后端稳定性/安全 257 个 Go 文件 + 数据库/部署/前端），所有条目均有文件:行号依据。

## 勘察结论摘要

### P0（已确认，立即修）
1. **rag_jobs 入队必然失败**：`domain/rag_job.go` 的 `ID string` 主键无 `default` tag，GORM 对零值 string PK 显式插入 `''`，撞 UUID 列报错 → pgvector 模式 RAG 全链路瘫痪。
2. **微信 SSE 硬死锁**：`pkg/bot/wechat{,_service}/wechat.go` 持 `state.Mutex` 向无缓冲 `NotificationChan` 发送；消费端断开后 defer 再抢锁 → 两 goroutine 永久互锁，`llmSemaphore` 槽位泄漏，累计后全站对话瘫痪。
3. **主 API 无 Recover + bot goroutine panic**：`val.(*domain.ConversationState)` 对 nil 断言在裸 goroutine 中 panic → 整个进程崩溃（对比 feishu 已有 recover）。
4. **JWT_SECRET 空不拦截**：空 key 可伪造管理员 token，且容器"健康"启动。

### P1 稳定性
- Chat SSE `eventCh` 裸发送，消费端断开后级联卡死（`usecase/chat.go` 多处 + 中间泵 goroutine）。
- `http.Client`/DefaultClient 无超时（wecom/dingtalk/wechat/cas/github 共 15+ 处）；CAS 还带 `InsecureSkipVerify`。
- `repo/pg/knowledge_base.go:328` caddy 同步 resp.Body 从不关闭。
- `cmd/api/main.go` 无优雅退出（Fatal 直接 os.Exit，SIGTERM 硬切 worker）。
- PG 连接池零配置（无上限、无生命周期；API 与内嵌 worker 共享同一池）。
- echo server 无 ReadTimeout/IdleTimeout（slowloris）。
- rag_jobs：无 reaper（processing 僵尸堆积）、`PurgeDone` 从未调度、失败重试无退避。
- `handler/share/wechat.go:112-122` 公开端点对 `Messages[0]/[1]` 无长度检查 → panic。
- `ensureSchemaInitialized` 只查 knowledge_bases，老库缺 rag_chunks/rag_jobs 也能启动。

### P1 安全
- `/api/v1/file/upload` 无 KB 级权限校验（任何登录用户可向任意 KB 前缀写文件）+ 主 API 无 BodyLimit。
- 凭据进日志：wechat/discord access token、CAS ticket（Info 级）。
- Echo RealIP 信任 XFF 头 → 登录限速/对话限频可伪造绕过。
- `checkDeniedExtension` fail-open（设置读取失败则黑名单失效）。
- widget LLM 端点无 captcha/限频兜底；feedback 可并发重复投票。
- `middleware/session.go` GetOrSet 非原子（应 SETNX）；ap_token/SimpleAuth 非常量时间比较。
- SSRF 工具完备但覆盖不足：`tools.go UploadMediaFromURL`、push webhook host 白名单（`strings.Contains`）。

### 性能
- **app 流式卡顿根因**：`incrementalRenderer.tsx` 前缀比较失配 → 每 token 全量重建 DOM + markdown-it 全文 O(n²) 重解析；mermaid 每 token 重渲染。
- admin 最大 chunk 3.0MB（@ctzhian/modelkit 未拆分）、echarts 全量引入、hljs 全量语言集、vendor-editor 2MB。
- `middleware/jwt.go GetKbID` 对所有写请求全量读 body+unmarshal 提取 kb_id。
- N+1：stat 聚合每 KB 串行 9 查询等 5 处。
- `fetch.ts` SSE 硬编码 5 分钟超时。

### 数据库
- 索引缺口：nodes(kb_id,nav_id)、node_releases(node_id,updated_at DESC)、conversation_messages(kb_id,created_at)、comments(kb_id,created_at)、auths/contributes/navs/mcp_calls 零索引、stat_pages 缺 created_at。
- pgvector 检索：过滤条件无法下推 HNSW → 多 KB 混存时欠返回。

### 部署
- quickstart.sh 导入 SQL 无 `ON_ERROR_STOP`（半套 schema 上线）。
- compose：caddy/api/app 全无 healthcheck，depends_on 全是 started。
- 根目录无 `.dockerignore` → 构建上下文携带 2.9GB node_modules；`COPY web/` 在依赖安装前（缓存失效）。
- config 无启动校验（空 JWT/空 S3 密钥照常启动）。
- generate-env.sh FORCE=1 会覆盖已初始化 DB 的密码。

## 轮次安排（已全部完成，2026-11-19）

| 轮次 | 主题 | 提交 | 状态 |
|---|---|---|---|
| 轮 0 | 计划文档 | 1d562991 docs(hardening) | ✅ |
| 轮 1 | P0 歼灭：rag_jobs UUID、微信死锁、Recover+panic、JWT 启动校验 | 1366dbc8 fix(stability) | ✅ |
| 轮 2 | 后端稳定性：SSE 排干×10、HTTP 超时、body 泄漏、优雅退出、连接池、server 超时 | 22433107 fix(stability) | ✅ |
| 轮 3 | 安全加固：凭据日志、BodyLimit+KB 权限、fail-open、CAS TLS、常量时间比较、session SETNX、SSRF 覆盖 | 2d078412 fix(security) | ✅ |
| 轮 4 | 数据库：9 索引、rag_jobs reaper+退避+清理调度、schema 检查增强、pgvector 召回 | 9fcf572a fix(db) | ✅ |
| 轮 5 | 部署便利性：quickstart 原子导入、compose healthcheck、.dockerignore（2.4GB→67MB）、Dockerfile 缓存、native 修复 | 8cb9c740 fix(deploy) | ✅ |
| 轮 6 | 前端性能：markdown2 块级增量渲染、admin 首屏 4.1MB→1.65MB、死代码、飞书卡片 JSON 注入 | 78615324 perf(frontend) | ✅ |
| 轮 6.5 | P2 清尾：SSE 错误泄露、SQL 参数化、admin 存量告警、死依赖 | c8039d08 fix(hardening) | ✅ |
| 轮 7 | 收尾：全量回归（后端 test/lint、app/admin build 全绿）、文档更新 | 本提交 | ✅ |

## 遗留事项（评估后暂不处理，含理由）

1. **larkws.Start goroutine 泄漏**（pkg/bot/feishu/stream.go）：上游 SDK 限制（成功路径永久阻塞 select{}），已有注释说明；钉钉/Discord 流有 ctx 退出机制。等上游修复。
2. **widget LLM 端点无 captcha**：需要前端 widget 配合改造（传 captcha token），涉及产品交互决策；现状有可选的提问间隔限频兜底。
3. **Echo RealIP 信任 XFF**：需要"可信代理"跨组件设计（caddy 配置由 API 动态下发），简单开关（TRUST_PROXY env）可后续加。
4. **API token / SimpleAuth 口令明文存储**：加密存储需要迁移方案与密钥管理设计，比较处已改常量时间。
5. **用户消息原文进 Info 日志**（handler/share 多处）：涉及产品排障需求取舍，建议引入日志脱敏中间件统一处理。
6. **jwt GetKbID 对写请求全量读 body**：性能优化收益中等，改动触及所有写请求路径，单独任务处理。
7. **nodes name/content LIKE 无 trigram 索引**：管理端低频操作，数据量大后再加。
8. **Dockerfile.postgres 零引用**：手动兜底镜像保留，注释已说明离线环境需预下载 pgvector 源码。

## 最终状态

- 后端：golangci-lint 0 issues、go test ./... 全绿
- 前端：app pnpm lint 0 问题、app/admin pnpm build 成功
- 全部改动分 9 个提交在本地 main，未推送（可用 git revert 逐轮回退）
