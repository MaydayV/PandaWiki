# 乘风版更新日志

## 版本 FV2.6.20 — 前台全链路双语闭环、访客语言切换与文档 AI 翻译入口

- **前台与共享组件双语闭环（`web/app` / `@panda-wiki/ui`）**：
  - 修复首屏默认语言为 `auto` 导致的 SSR/CSR 水合不一致问题，引入 `pw_locale` Cookie 与页头访客语言切换组件（`LanguageSwitch`），支持访客在 `en-US` / `zh-CN` 间切换。
  - 统一前台问答、反馈、Emoji 与导航组件文案的双语表达，并在 `en-US` 模式下将历史中文反馈标签与免责声明自动映射为英文。
- **分享接口与对话流多语言适配（`backend`）**：
  - 分享接口 CORS 放行 `X-Lang` 与 `Accept-Language` 请求头；分享站点信息、挂件限流/冷却提示、AI 对话 SSE 错误、文件上传提示与文档详情按请求语言返回对应中英文消息。
  - 文档自动翻译接口补充 `source_language` 参数，支持将中文后台维护的文档定向翻译为英文供前端访客浏览。
- **管理后台文档 AI 翻译入口（`web/admin`）**：
  - 保持后台界面全中文操作，在文档编辑器顶部新增「AI 翻译」按钮与「多语言 AI 翻译」配置弹窗。
- **文档与版本同步**：将发布版本号由 `FV2.6.19.2111` 升级至 `FV2.6.20.2111`，同步更新部署示例与版本说明。

## 版本 FV2.6.19 — 全栈稳定性与安全加固 + GitHub 私有仓库 MD 同步 + 发布包完善

### 一、新功能：GitHub 私有仓库 Markdown 同步
- 支持以 Git 仓库作为知识库唯一内容真源，定期轮询与手动触发全链路同步：
  - 按 `front-matter category` 与目录层级自动识别/创建栏目及多层目录，支持 `order` 排序。
  - 仓库相对路径图片按 blob SHA 复用并自动转存 OSS，彻底解决私有仓库图片前台不可访问问题。
  - 按文件 SHA 跳过未变更文章节省 embedding 成本，仅对本次新建/更新节点及孤儿删除自动发布 KB Release 并触发 RAG 向量更新。
  - 新增后台「GitHub MD 同步」设置卡片（仓库/分支格式与连通性校验、加密存储 PAT、手动立即同步、状态摘要回写）及 Redis 分布式锁防重入。

### 二、全栈稳定性与安全深度加固
- **安全与权限边界**：
  - 凭据落库支持 `ENCRYPTION_KEY`（AES-256-GCM）加密存储（涵盖 API Token、SimpleAuth 访问口令、GitHub 同步 PAT），旧明文数据懒迁移兼容，并自动幂等补齐存量数据库 `api_tokens.token_index`。
  - 收紧 `/api/v1/file/upload(+url)` KB 级权限校验与高危 HTML/JS MIME 降级防御，扩展 SSRF 保留 IPv6 网段拦截、常量时间比对及 `TRUST_PROXY` 可信代理链路防伪造限频头。
  - 保留 `/api/v1/user/list` 仅限系统管理员，新增 KB 协作者候选用户接口 `GET /api/v1/knowledge_base/user/candidates` 修复非管理员邀请协作者 403 问题。
- **核心链路与并发可靠性**：
  - 全链路补齐 SSE 心跳保活、断连排干防 goroutine 泄漏、飞书/钉钉/企微/Discord bot 优雅退出及上游 `larkws` 补丁、HTTP 客户端超时、数据库连接池与 `rag_jobs` 指数退避/僵尸回收及耗尽失败收敛。
  - 修复投稿审批回退 `pending`、pgvector 低版本 `SAVEPOINT` 降级保护（保留 `ef_search`）、知识库删除级联清理 `conversation_references`/`rag_jobs`、前台增量 Markdown 渲染占位图抖动与 admin 跨平台分包构建正则。
  - 前台 Next.js 升级至 `16.4.6` 并清零依赖已知高危/中危漏洞；补齐并发与单元测试覆盖。

### 三、部署体系、版本号与发布包完善
- 修复 `Dockerfile.api`（补充 `pnpm-lock.yaml` 与 `third_party/`）、`Dockerfile.api.pro` / `Dockerfile.consumer.pro` 构建上下文缺失。
- 统一管理端构建版本号注入逻辑为 `FV2.6.19.2111`；Compose 的 `TRUST_PROXY` 改为读取 `.env` 配置。
- GitHub Release 发布包正式纳入 `backup.sh` 定时备份脚本（支持发布包根目录与源码仓库直接运行），同步更新部署说明与 CI/CD 构建工作流。

## 版本 FV2.6.18 — 上游安全防御合入 + OSV 漏洞修复 + 系统配置 React #130 修复

### 一、安全加固与上游合并
- **SSRF 深度防御**：严格封禁私有、回环、保留及 IPv4-mapped IPv6 等内网地址，防范远程拉取图片及外部资源时的内网穿透风险。
- **远程图片校验**：校验安全响应格式与魔数（Magic Bytes），防止伪造扩展名注入恶意脚本。
- **Sitemap XML 转义**：对站点地图中 URL 实体符号进行合规转义，防范 XML 实体注入。
- **RAG 任务幂等锁**：修复节点 RAG 索引并发重试冲突，增加基于数据库状态与分布式锁的防重逻辑。

### 二、依赖安全漏洞清零（OSV 扫描验证通过）
- **前端 Next.js 升级**：升级至 Next.js 16.4.1，彻底清除 CVE-2025-29927、CVE-2025-29928、CVE-2025-55182 等所有已知安全漏洞。
- **后端 Go 依赖升级**：Echo 升级至 4.16.1，gRPC 升级至 1.83.2，gomarkdown 等核心模块同步升级至官方无漏洞安全修订版。

### 三、管理后台与前端修复
- **修复系统配置报错 Minified React error #130**：
  - 排查定位 `lottie-react`、`@emoji-mart/react`、`react-diff-viewer` 等 CommonJS 打包模块在 Vite / React 19 下导出嵌套对象导致 React 无法识别组件类型的问题。
  - 增强 `LottieIcon`、`EmojiPicker`、`MarkdownPreviewModal` 等组件的 `default` 展开兼容与运行时防御降级。
- **模型配置组件鲁棒性增强**：增强 `ModelConfig` 中 provider 图标的安全检索与防御，避免未知 provider 图标丢失时引发页面崩溃。

## 版本 FV2.6.17 — 飞书机器人增强 + 知识库更新推送 + 全系统深度审查

### 一、新功能

#### 1. 知识库更新主动推送
- 知识库发布新版本后，自动向配置的群聊发送更新通知
- 支持飞书和钉钉平台（Webhook 方式，配置 Webhook URL 即可）
- 后台「AI 机器人」设置页新增「知识库更新推送」配置卡片：
  - 启用/禁用开关（RadioGroup）
  - 推送目标群聊（支持多个 Webhook URL，逗号分隔）
  - 推送消息模板（支持 `{kb_name}` `{tag}` `{message}` `{release_time}` 变量）
  - 测试发送按钮（点击后发送测试消息到第一个群聊）
- 推送在发布成功后异步执行，失败不影响发布流程

#### 2. 飞书机器人健壮性增强
- 新增 panic recovery，防止消息处理 goroutine 泄漏
- 卡片流式更新失败时不再中断，继续尝试下一个 chunk
- QA 失败时回退发送纯文本错误消息"出错了，请稍后再试"
- 提取飞书用户手机号（Mobile）写入会话信息
- 群聊 @提及占位符 `@_user_N` 替换为实际用户名
- 消息解析提前到事件回调层，私聊/群聊共用用户信息获取逻辑
- 流式卡片 answer 改用 `strings.Builder` 消除 O(n²) 字符串拼接

#### 3. 测试推送后端接口
- 新增 `POST /api/v1/app/push/test` 端点
- 请求体：`{ "app_id": "xxx", "chat_id": "webhook_url" }`
- 需要管理员权限（FullControl）

#### 4. 前端 ErrorBoundary
- 新增根级 ErrorBoundary 组件，捕获渲染异常防止全页面白屏
- 错误页面显示错误信息和"返回首页"按钮

#### 5. useBotForm Hook
- 新增 `useBotForm` 通用 hook，封装机器人配置表单的 fetch/reset/submit/fieldChange 逻辑
- 供 9+ 个 CardRobot 组件复用，消除每组件 ~40 行重复代码

### 二、问题修复

#### 安全修复
- 钉钉 access_token 日志脱敏：掩码显示 `xxxx****xxxx`，不再明文暴露凭据
- 推送模板 Markdown 特殊字符转义：`*_#>[]|` 自动转义，防止钉钉 markdown 消息格式错乱

#### 数据一致性修复
- `KBUpdatePushEnabled` 移除 `omitempty`：`false` 值不再被 JSON 忽略
- 推送字段补充映射到 `AppSettingsResp`：前端可以正确读写推送配置
- `WechatServiceLogo` 字段补充映射到响应：修复字段丢失
- 合并上游时引入的重复 `promptJson` 结构体定义已移除

#### 性能优化
- `GetUsersAccountMap` 改为 `GetUsersAccountMapByIDs`：仅查询需要的 3 条用户，不再加载全表
- KB 缓存 `SetKB` 增加 10 分钟 TTL：防止永久脏数据
- echarts 从 `vendor-echarts` 手动分块中移除：随统计页懒加载，首屏减少 ~800KB

#### 并发安全
- `DFA.DeleteWordBatch` 移除并发 goroutine：改为顺序执行，消除数据竞争
- `ChatUsecase` 新增 `llmSemaphore`（容量 50）：限制并发 LLM 调用，防止资源耗尽

#### 机器人生命周期
- 飞书/钉钉 bot 新增 `done` channel：`Stop()` 后等待 goroutine 退出再重建，消除泄漏
- 飞书/钉钉/企微机器人禁用时调用 `UnregisterNotifier`：清理推送注册
- Lark（国际版飞书）注册飞书 Webhook 推送 notifier：修复 Lark 机器人推送无效
- 异步推送 goroutine 增加 60 秒超时 context：防止 goroutine 永久阻塞

#### 签名修复
- 飞书/钉钉 Webhook 签名恢复为正确的 key+空消息模式（与官方文档一致）

#### 类型安全
- 补充上游缺失的 `NodeStatsReq`、`NodeListGroupNavReq`、`NodeMoveNavReq` 类型定义
- 重新生成 Swagger 文档和前端 TypeScript 类型
- CardPush 移除全部 5 处 `@ts-expect-error`

#### 代码清理
- CardRobotFeishu、CardRobotDing 删除各 20 行注释掉的死代码

### 三、变更文件清单

**新增文件：**
- `backend/pkg/bot/push.go` — PushNotifier 统一推送接口
- `backend/pkg/bot/feishu/push.go` — 飞书 Webhook 推送实现
- `backend/pkg/bot/dingtalk/push.go` — 钉钉 Webhook 推送实现
- `backend/usecase/push.go` — 推送编排逻辑（模板渲染、多群聊分发、限流、测试）
- `web/admin/src/pages/setting/component/CardPush.tsx` — 推送配置 UI 组件
- `web/admin/src/components/ErrorBoundary.tsx` — 根级错误边界
- `web/admin/src/hooks/useBotForm.ts` — 机器人表单通用 hook

**修改文件：**
- `backend/domain/app.go` — 新增 KBUpdatePush 配置字段
- `backend/domain/chat.go` — 新增 Mobile 字段
- `backend/usecase/app.go` — 推送字段映射 + notifier 注册/注销 + bot 生命周期同步
- `backend/usecase/knowledge_base.go` — 发布后异步触发推送
- `backend/usecase/chat.go` — LLM 并发限制信号量
- `backend/usecase/push.go` — Markdown 转义
- `backend/usecase/node.go` — 用户查询精准化
- `backend/usecase/provider.go` — Wire 注册 PushUsecase
- `backend/handler/v1/app.go` — 测试推送端点
- `backend/pkg/bot/feishu/stream.go` — panic recovery + strings.Builder + done channel + replaceMentions
- `backend/pkg/bot/dingtalk/stream.go` — token 脱敏 + done channel
- `backend/pkg/bot/dingtalk/push.go` — 签名修正
- `backend/pkg/bot/feishu/push.go` — 签名修正
- `backend/repo/pg/user.go` — 新增 GetUsersAccountMapByIDs
- `backend/repo/pg/prompt.go` — 移除重复 promptJson
- `backend/repo/cache/kb.go` — KB 缓存增加 TTL
- `backend/utils/DFA.go` — 修复并发写 map
- `backend/api/node/v1/node.go` — 补充上游缺失类型
- `backend/docs/*` — Swagger 文档重生成
- `backend/cmd/api/wire_gen.go` — Wire DI 更新
- `backend/cmd/migrate/wire_gen.go` — Wire DI 更新
- `web/admin/src/main.tsx` — 集成 ErrorBoundary
- `web/admin/src/request/types.ts` — 新增推送字段类型
- `web/admin/src/pages/setting/component/CardRobot.tsx` — 集成 CardPush
- `web/admin/src/pages/setting/component/CardRobotFeishu.tsx` — 清理死代码
- `web/admin/src/pages/setting/component/CardRobotDing.tsx` — 清理死代码
- `web/admin/vite.config.ts` — echarts 移除 vendor 分块

### 四、审查统计

| 优先级 | 发现 | 已修复 |
|--------|------|--------|
| 🔴 严重（安全/数据一致性） | 6 | 6 ✅ |
| 🟠 高（可靠性/性能） | 8 | 8 ✅ |
| 🟡 中（可维护性/代码质量） | 4 | 4 ✅ |
| ✅ 误报（文档验证排除） | 3 | — |
| **合计** | **21** | **18 ✅ + 3 误报** |
