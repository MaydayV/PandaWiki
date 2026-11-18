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

## 轮次安排

| 轮次 | 主题 | 提交信息前缀 |
|---|---|---|
| 轮 0 | 计划文档 | docs(hardening) |
| 轮 1 | P0 歼灭：rag_jobs UUID、微信死锁、Recover+panic、JWT 启动校验 | fix(stability) |
| 轮 2 | 后端稳定性：SSE ctx 保护、HTTP 超时批量、body 泄漏、优雅退出、连接池、server 超时、panic 边界 | fix(stability) |
| 轮 3 | 安全加固：凭据日志、BodyLimit+KB 权限、fail-open、CAS TLS、常量时间比较、session SETNX、SSRF 覆盖 | fix(security) |
| 轮 4 | 数据库：补索引、rag_jobs reaper+退避+清理调度、schema 检查增强、pgvector 检索 | fix(db) |
| 轮 5 | 部署便利性：quickstart 原子导入、compose healthcheck、.dockerignore、config 快速失败、native 修复 | fix(deploy) |
| 轮 6 | 前端性能：markdown2 增量渲染、admin chunk 拆分、死代码清理、SSE 超时 | perf(frontend) |
| 轮 7 | 收尾：全量回归、文档更新、总结 | docs(hardening) |

每轮完成后运行对应验证（go build/test/golangci-lint；admin/app pnpm build/lint），全部通过才提交。
