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

## 遗留事项（第四轮迭代后更新，2026-11-19：清零）

第四轮迭代（轮 13~14）将第三轮"评估后不动"的 3 项**全部解决**：

1. ~~API token / SimpleAuth 口令明文存储~~ → **已解决**：新增 `ENCRYPTION_KEY`
   配置（AES-256-GCM），配置后落库加密（`enc:v1:` 前缀密文），未配置 = 明文
   兼容既有部署、零行为变化；`api_tokens` 加 `token_index` HMAC 确定性索引列
   支撑等值查询；repo 返回调用方前解密（校验/展示/缓存交互不变）；密钥轮换/
   密文损坏解密 fail-closed 返回空串。附加密工具单测 4 个（往返/防二次加密/
   确定性索引/fail-closed）。
2. ~~用户消息原文进 Info 日志~~ → **已解决**：7 处用户消息/请求体日志降级为
   Debug（生产默认 Info 级不输出，隐私合规；排障时 `LOG_LEVEL=-4` 可见）。
3. ~~larkws select{} 空转 goroutine~~ → **已解决**：本地 fork 上游 SDK
   （`third_party/larksuite-oapi-sdk-go-v3` + go.mod replace，271 个 Go 文件），
   给 ws.Client 增加 `Close()` 优雅关闭（closedCh + 断开连接）并让 Start 响应
   ctx 取消（原本 select{} 永久阻塞）、reconnect 循环补 ctx 检查。
   `FeishuClient.Stop` 现在真正断开 ws 连接——空转 goroutine 与幽灵连接
   彻底消除（closed 屏蔽保留为关闭竞态窗口的双保险）。

**遗留项清零。** 勘察清单中的全部问题（4 P0 / 30+ P1 / 40+ P2）均已修复或闭环。

## 最终状态（第五轮复核后，2026-11-19）

第五轮复核（轮 15~17）对四轮全部改动做完整性检查，发现并修复 7 处缺陷：

**后端加密链路 4 处**（0c48c82c）：
- P0 CreateAPIToken 返回密文（repo.Create 就地改 Token，响应必须用明文）
- P0 generate-env.sh 漏配 ENCRYPTION_KEY（补 rand_hex 生成）
- P1 ENCRYPTION_KEY 开启后旧明文 token 失效（token_index 回退明文查询 + 懒迁移）
- P1 isChange 密文误判（cmp.Equal 对比密文，nonce 随机导致每次保存都踢会话）

**前端复核 3 处**（a5a9ac07、5646a0b4，P0 零）：
- P1 图片流式加载窗口：createRoot 重复挂载泄漏 + 重复 fetch
  （root 注册表复用/unmount + in-flight 去重）
- P1 AiQaContent updater 原地改引用只修了 data 分支（error/done/
  chunk_result/onCancel 共 8 处补齐浅拷贝）
- admin 全量 lint 首次建立基线：8 个存量 error 清零 + vite 正则
  Windows 分隔符兼容

复核确认项：增量渲染序列化对称性/降级路径安全、SimpleAuth 唯一消费者走
解密路径、无其他 token 查询点、SSE 心跳前端按规范忽略注释行、限频窗口
语义正确、TRUST_PROXY 链路 caddy set 覆盖语义正确、SQL 多行语句完整。

## 第六轮迭代（轮 18~20，2026-11-20）：新勘察面闭环

对**未系统勘察过**的两个面（前端 SSE 消费端、后端 bot 消息链路）重新勘察，
发现 1 P0 + 10 P1 + 20+ P2，已修复（2a5dac57）：

**后端 bot 链路**（共同放大器：消费端放弃 channel 即级联卡死泵+LLM 槽）：
- lark 裸 goroutine 无 recover、公众号 GetUserInfo nil 解引用崩进程、
  wechat_service/dingtalk 更新失败放弃消费（排干）、discord 2000 截断
  （rune 安全，push 链同步）、wecom 死锁地雷（锁外+超时）、
  SendMessage 失败当成功、SendQuestionToAI 的 close/Delete 移 defer
  （panic 路径消费者不阻塞、条目不泄漏）

**前端 SSE 消费端**：
- P0 app fetch.ts 缺 raw 模式（编辑器润色面板永远空白）
- P1 onError 缺失（HTTP 错误 loading 卡死 5 分钟）、unsubscribe 误污染
  上一条回答、新对话幻影请求、AIGenerate 取消后 loading 卡死

复核确认：conversationID 每次新生成使重复消息路径不可达、
hasActiveStreamRef 复位覆盖全部出口、raw 模式默认值保持现有行为。

- 后端：golangci-lint 0 issues、go test ./... 全绿
- 前端：app lint/build、admin build 全部通过
- 全部改动分 20 个提交在本地 main，未推送（可用 git revert 逐轮回退）

## 部署提示

- 生产建议配置 `ENCRYPTION_KEY`（`docs/deploy/.env.example` 有说明）：启用后
  新写入的凭据为密文，旧明文数据仍可读取；配置后请勿更换密钥。
- SDK fork 说明：`third_party/larksuite-oapi-sdk-go-v3` 是上游 v3.4.20 的本地
  补丁副本（ws/client.go 的 Close 注释标明改动点），升级上游时需重新套用补丁。

## 第七轮深度迭代（轮 21~30，2026-11-20）

六个新勘察面（权限矩阵/i18n/goroutine/发布状态机/admin 数据流/app 数据流/
repo 层/web/packages/收尾角落）持续闭环，修复 **2 个新 P0**（跨 KB 会话
泄露、RAG doc_id 断裂致检索全坏）+ **30+ P1** + 大量 P2：

| 提交 | 主题 |
|---|---|
| 22b609eb | 权限矩阵（P0 会话泄露/摘要泄露/MoveNode 跨 KB/user list）+ i18n + goroutine 加固×10 |
| 7fbd267b | 发布状态机（P0 doc_id 断裂/KB 级联/发布补偿）+ admin 数据流竞态 |
| a959c42d | 剩余 handler（敏感词缓存/解绑 session/口令限速）+ app 数据流 |
| 9b77237a | 复核轮 22/23——恢复误删的 nodes 级联 + 限速 key 隔离 |
| ea1d2c21 | 复核前端——release 页残留竞态 + setting 文案 |
| b13ea55f+86491c49 | repo 层（PV 双计/投稿审批抢占式）+ 主题 token 补齐 + 时区修正 |
| 4a0f7d20 | 收尾角落（RAG 入队分批/上传嗅探/密钥日志/cert 权限） |

**轮 29（复核轮 26-28 diff）与轮 30（独立抽查+全量门禁）连续两轮零发现，
满足停止条件。**

## 最终状态（轮 30）

- 后端：golangci-lint 0 issues、go test ./... 全绿（含加密/DFA/rag_job 单测）
- 前端：app eslint 0 问题、admin eslint error 0、app/admin/themes build 全部成功
- 累计 **28 个修复提交**（1d562991..4a0f7d20）在本地 main，未推送
- 勘察面覆盖率：backend 全部目录 + web 三端全部目录 + 共享包 + 部署链路

## 第八轮迭代（轮 31~35，2026-11-20）：并发专项 + 漏洞扫描 + 投稿链路

新勘察面（govulncheck/pnpm audit/全局共享状态并发/api-pro+投稿链路），
修复 1 P0 + 2 P1 + 10 P2：

| 提交 | 主题 |
|---|---|
| 5def4bb0 | next 16.4.6（pnpm audit 清零） |
| be88c5ab | 并发专项（IsDone 裸读/wsCli atomic/cron CAS/全局 slice 拷贝/dingtalk 挂死缓解/discord 顺序）+ govulncheck 缓解确认 |
| 2cfc9516 | 投稿链路（P0 nav_id 丢失致文档树不可见/审批校验前置+失败回写 pending） |
| （本轮） | DocModal 空组提示收尾 |

轮 34/35 连续两轮零新发现，二次达成停止条件。累计 **34 个提交**
（1d562991..本提交）。
