# PandaWiki 商业版交付矩阵（2026-02-14）

## 1. 本轮决策与执行边界
- 仓库策略：仅在当前 fork 仓库实现与提交，不向上游提交。
- 管理台多语言：保持中文可用，非本轮重点。
- 多语言重点：前台知识库内容多语言能力（当前为重点后续项）。
- MCP 传输：采用 HTTP JSON-RPC（`/mcp`），优先降低客户接入复杂度。
- API 治理：以 token 维度做请求限流/配额与审计，先按“请求次数”治理。
- 防复制：保持简单可用策略（`allow/append/disabled`），不做过度细分。

## 2. 功能交付矩阵（10 项）

| 功能项 | 商业状态 | 本轮结果 | 管理入口 | 核心实现/接口 |
|---|---|---|---|---|
| 多语言支持（前台） | 部分实现 | 前台 UI 已支持 `zh-CN/en-US` 切换与跟随语言；知识库“内容自动翻译”未完成 | 管理台语言设置 | `web/app/src/i18n/*`、`backend/domain/app.go` `i18n_settings` |
| 自定义 AI 提示词 | 已交付（增强版） | 已支持提示词版本、详情、回滚与审计链路 | 管理台提示词设置 | `/api/pro/v1/prompt/version/*`、`backend/repo/pg/prompt.go` |
| 自定义版权信息 | 已交付（可继续统一） | 已有 `conversation_setting` 与 `brand_settings`；可按品牌策略继续统一 | 管理台设置 | `backend/domain/app.go` `brand_settings` |
| SEO 配置 | 已交付（本轮补齐） | 后台 SEO 面板补齐 canonical/robots/og/twitter/json-ld；前台 metadata/robots/sitemap 已渲染 | 管理台 SEO 卡片 | `web/admin/src/pages/setting/component/CardWebSEO.tsx`、`web/app/src/app/layout.tsx`、`web/app/src/app/robots.ts`、`web/app/src/app/sitemap.ts` |
| 访问流量分析 | 已交付（基础） | 具备统计写入与聚合，含 pv 开关拦截；高级漏斗可继续增强 | 管理台统计页 | `backend/handler/share/stat.go`、`backend/usecase/stat.go` |
| MCP Server | 已交付 | 提供 `/mcp` JSON-RPC：`initialize`、`tools/list`、`tools/call`，带鉴权与调用日志 | 管理台 MCP 设置 | `backend/handler/share/mcp.go` |
| 页面水印 | 已交付（基础） | 前台已有水印显示/隐藏能力；可继续增强溯源策略 | 管理台安全设置 | `web/app/src/components/watermark/WaterMarkProvider.tsx` |
| 内容不可复制 | 已交付（简化） | 维持简单策略：允许/追加来源/禁用复制；符合当前产品决策 | 管理台安全设置 | `web/app/src/hooks/useCopy.tsx` |
| 文档历史版本管理 | 已交付（增强版） | 已有 list/detail/rollback/diff 与审计，前端历史页面可走后端回滚 | 管理台文档历史 | `/api/pro/v1/node/release/*`、`backend/usecase/node.go` |
| API 调用能力与治理 | 已交付（本轮关键补齐） | `/share/v1/chat/completions` 增加 token 限流/配额（429 + OpenAI 错误类型）、usage 与审计 | API Token 管理 + OpenAI 兼容接口 | `backend/handler/share/chat.go`、`backend/repo/pg/api_call_audit.go`、`backend/domain/api_token.go` |

## 3. 本轮已修复项（对应你确认的问题）

### 3.1 后台 SEO 面板完整化
- 已补字段：`canonical_url`、`robots`、`og_image`、`twitter_card`、`json_ld`。
- 已保留：`desc`、`keyword`。
- 保存策略：`desc/keyword` 写顶层；高级字段写 `settings.seo_settings`。
- 关键文件：`web/admin/src/pages/setting/component/CardWebSEO.tsx`。

### 3.2 API token 级限流/配额治理
- `api_tokens` 新增字段：
  - `rate_limit_per_minute`（0=不限）
  - `daily_quota`（0=不限）
- 适配 create/list/update 全链路。
- `/share/v1/chat/completions`：
  - 超分钟限流：`error.type=rate_limit_error`，HTTP `429`。
  - 超日配额：`error.type=insufficient_quota`，HTTP `429`。
- 鉴权兼容：支持“OpenAI SecretKey 或 API Token”调用。
- 管理台已可在创建 token 时配置限流/配额并在列表显示。
- 关键文件：
  - `backend/domain/api_token.go`
  - `backend/repo/pg/ap_token.go`
  - `backend/repo/pg/api_call_audit.go`
  - `backend/handler/share/chat.go`
  - `web/admin/src/pages/setting/component/CardKB.tsx`
  - `backend/store/pg/migration/000040_add_api_token_governance_fields.up.sql`

### 3.3 MCP 传输方案选择
- 结论：当前保持 HTTP JSON-RPC（不强制 SSE）。
- 原因：客户技术接入更简单、调用门槛低、调试工具链成熟。
- 可扩展：后续若需要流式能力再补 SSE transport，不影响现有调用。

### 3.4 防复制策略
- 结论：保持简化策略（`allow/append/disabled`），当前实现可用。

## 4. “API 调用治理”的产品用途（给业务方）
- 成本控制：限制单 token 过量请求，避免模型费用失控。
- 稳定性保障：突发流量时保护服务，避免单客户挤占资源。
- 客户分级：不同 token 配不同限额，支持商业套餐策略。
- 审计追踪：按 token 记录调用，便于计费、排障、风控。

## 5. MCP 接入建议（对客户开发最简）
- Endpoint：`POST /mcp`
- Header：`X-KB-ID: <kb_id>`，必要时 `Authorization: Bearer <mcp_token>`
- JSON-RPC 方法：
  - `initialize`
  - `tools/list`
  - `tools/call`
- 建议：先走 HTTP JSON-RPC 直连；仅在明确需要流式推送时再评估 SSE。

## 6. 尚需继续推进（下一阶段）
- 前台“知识库内容多语言自动翻译/多语言内容发布”能力（本轮未完成）。
- 统计看板高级维度（漏斗/来源/转化）深化。
- 品牌与版权配置项进一步统一到单一品牌模型。
