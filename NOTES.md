# 本次修改记录（多语言/版权/提示词）

## 目标与范围
- 多语言：仅前端界面文案的语言切换，不是文档内容翻译。
- 版权信息：开源版也允许自定义版权文案/显示隐藏。
- AI 提示词：支持在管理端编辑并通过新接口读写。
- 文档数量：移除上限校验；后台不展示“文档数量”能力项。

## 多语言策略（重点）
- 语言来源：基于站点设置 `app.settings.language`（管理端“网站语言”）。
- 可选值：`en-US`、`zh-CN`、`auto`。
- `auto`：根据浏览器/请求头 `Accept-Language` 解析语言（前端 SSR 用请求头，客户端用 `navigator.language`）。
- 默认语言：`en-US`（前台 UI 默认英文）。
- 文档内容不参与语言切换，不会自动翻译；多语言只影响界面文案、提示、默认版权等。

## 后端改动摘要
- `AppSettings` 增加 `language` 字段并下发至 Web/Widget info。
- 开源版默认放开自定义版权限制（`AllowCustomCopyright: true`）。
- 新增 Prompt 读写接口：`/api/pro/v1/prompt` (GET/POST)。
  - 读：无记录时返回默认提示词。
  - 写：落地 `settings` 表 `system_prompt` 项，JSON 结构 `{content, summary_content}`。
- 文档数量上限校验已移除（创建节点不再检查 MaxNode）。

## 前端改动摘要
- 管理端新增“网站语言”卡片（Web 设置页）。
- 版权与提示词卡片取消版本限制（开源版可用）。
- Web/App 增加轻量 i18n：
  - `resolveLanguage` 统一解析语言；`dayjs` 语言跟随解析结果。
  - 关键 UI 文案使用 `useI18n()`。
  - 默认版权、提示、状态文案支持中英文。
- 后台隐藏“每个 Wiki 的文档数量”显示（移除 label）。

## 关键文件列表
- 后端：
  - `backend/domain/app.go`（language 字段）
  - `backend/domain/setting.go`（默认语言常量）
  - `backend/domain/license.go`（默认许可放开版权限制）
  - `backend/domain/prompt.go`（Prompt DTO）
  - `backend/repo/pg/prompt.go`（Prompt settings 读写）
  - `backend/usecase/llm.go`（Prompt settings 读写接口）
  - `backend/handler/v1/knowledge_base.go`（prompt API）
  - `backend/usecase/app.go`（language & 默认免责声明）
  - `backend/repo/pg/node.go`（移除文档数上限校验）
- 前端（Web App）：
  - `web/app/src/i18n/*`（locale/messages/useI18n）
  - `web/app/src/provider/index.tsx`（dayjs locale 切换）
  - `web/app/src/app/layout.tsx`（html lang）
  - `web/app/src/components/header/index.tsx`（登出文案）
  - `web/app/src/components/QaModal/*`（问答状态/版权文案）
  - `web/app/src/views/widget/*`（Widget 文案/版权）
  - `web/app/src/views/node/DocContent.tsx`（移除硬编码 dayjs locale）
  - `web/app/src/assets/type/index.ts`（language 字段）
- 前端（Admin）：
  - `web/admin/src/pages/setting/component/CardLanguage.tsx`
  - `web/admin/src/pages/setting/component/CardWeb.tsx`
  - `web/admin/src/pages/setting/component/CardAI.tsx`
  - `web/admin/src/pages/setting/component/CardQaCopyright.tsx`
  - `web/admin/src/pages/setting/component/CardRobot/WebComponent/index.tsx`
  - `web/admin/src/constant/version.ts`（隐藏文档数量 label）

## 注意事项与后续扩展
- 目前 i18n 是轻量字典方式，适合少量文案；如果未来需要大规模翻译，可考虑统一接入 i18next。
- `language` 字段仍由 Web App 的设置读取；Widget 通过 Web App settings 携带语言。
- 若新增文案，记得补齐 `web/app/src/i18n/messages.ts`。
- 若新增 SSR 页面需要语言，统一用 `resolveLanguage` 解析。

## 追加记录（前台文案补齐 i18n）
- 搜索结果页（QA/Widget）补齐结果统计、空状态、摘要缺失、校验失败提示的 i18n。
- 文档贡献编辑页补齐保存/提交/占位/校验提示等文案 i18n。
- 反馈弹窗、Mermaid/Markdown 渲染错误、图片加载占位文案接入 i18n。
- 上传错误提示改为根据当前语言返回（`upload.*` 文案）。
- Widget “敬请期待”提示改为 i18n。
