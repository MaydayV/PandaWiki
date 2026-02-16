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

## 部署记录（2026-02-14，Debian VM）
- 目标机器：`deploy@10.211.55.5`
- 线上源码目录：`/home/deploy/pandawiki-src`
- 线上编排目录：`/home/deploy/pandawiki-src/deploy`

### 本次采用方式
- 使用 Git 同步源码到虚拟机（非本机构建部署）。
- 本机创建快照提交后，推送到虚拟机 bare 仓库：
  - `ssh://deploy@10.211.55.5/home/deploy/pandawiki-sync.git`
- 虚拟机工作目录通过 `git fetch + reset` 更新到该提交。

### 关键命令（可复用）
```bash
# 本机：推送当前提交到虚拟机 bare 仓库
git push ssh://deploy@10.211.55.5/home/deploy/pandawiki-sync.git HEAD:refs/heads/deploy-sync

# 虚拟机：更新工作目录到 deploy-sync
git -C /home/deploy/pandawiki-src init
git -C /home/deploy/pandawiki-src remote add sync /home/deploy/pandawiki-sync.git
git -C /home/deploy/pandawiki-src fetch sync deploy-sync
git -C /home/deploy/pandawiki-src reset --hard sync/deploy-sync

# 虚拟机：先构建 admin 前端产物（关键，admin 镜像只 COPY web/admin/dist）
cd /home/deploy/pandawiki-src/web/admin
pnpm install --frozen-lockfile
NODE_OPTIONS=--max-old-space-size=4096 pnpm build

# 虚拟机：重建并拉起核心服务
cd /home/deploy/pandawiki-src/deploy
sudo docker compose build panda-wiki-api panda-wiki-consumer panda-wiki-app panda-wiki-admin
sudo docker compose up -d --force-recreate panda-wiki-api panda-wiki-consumer panda-wiki-app panda-wiki-admin
```

### 本次验收结果
- `docker compose ps` 显示核心服务 `panda-wiki-api / consumer / app / admin` 均为 `Up`。
- `https://127.0.0.1:2443` 返回 `HTTP/2 200`（管理端 HTTPS 正常）。
- `http://127.0.0.1:3010` 返回 `HTTP/1.1 200`（前台正常）。
- `http://127.0.0.1:8000/share/v1/app/web/info` 返回 `HTTP/1.1 200`（API 可用）。

## 追加记录（2026-02-15，前台多语言与样式修复）
- 修复欢迎页配色保存后不生效问题：`/share/v1/app/web/info` 与 `/share/v1/app/widget/info` 在服务端读取改为 `no-store`，避免 SSR 缓存旧配置导致主题不刷新。
- 修复文档页英文元信息显示：
  - 统一改为 `Created {{time}} / Updated {{time}}`。
  - 移除创建者/编辑者账号展示。
  - 修复复制尾巴文案硬编码中文，改为走 i18n。
- 修复英文环境残留中文：
  - 顶部“智能问答”按钮文案改为可传入（header/welcomeHeader/banner）。
  - 前台接入 `qa.chatTab` 作为按钮文案。
  - 底部品牌文案改为可传入，并由前台语言配置输出（默认 `Powered by PandaWiki`）。
  - 兼容历史默认中文版权文案（`本网站由 PandaWiki 提供技术支持`），英文环境自动替换为英文默认值。
  - 目录卡片“查看更多”改为可国际化文案。
- 修复左侧目录英文换行问题：目录标题区域改为单行省略显示，避免 `Catalog` 被错误折行。
- 新增设置项：`设置 -> 前台网站样式个性化` 增加“前台 Logo”上传（PNG），直接写入 `settings.icon`，前台头部和问答弹窗 logo 统一生效。

## 追加记录（2026-02-15，Logo 上传与显示修复）
- 修复 Logo 上传缺少比例校验的问题：
  - `UploadFile` 组件新增可选参数 `requireSquare/squareTolerance/squareErrorMessage`。
  - 前台 Logo 上传启用 1:1 校验，非正方形 PNG 直接提示并阻止上传。
- 修复 Logo 上传后前台不显示的问题：
  - `web/app/.env` 增加 `STATIC_FILE_TARGET=http://panda-wiki-minio:9000`，确保 `next build` 时生成 `/static-file` 正确转发。
  - `web/app/next.config.ts` 的 `/static-file` 转发改为优先使用 `STATIC_FILE_TARGET`，缺失时回退 `TARGET`。
- 前台 Logo 展示稳定性优化：
  - 前台头部与欢迎页头部 Logo 改为固定 `36x36`，并使用 `object-fit: contain`，避免透明 PNG 被裁切或撑变形。
- 虚拟机构建注意：
  - `web/admin` 在 Debian 小内存环境下构建可能 OOM，需使用：
    - `NODE_OPTIONS=--max-old-space-size=4096 pnpm --filter panda-wiki-admin build`

## 追加记录（2026-02-16，安全设置页网络异常修复）
- 根因：安全设置页“内容合规”初始化请求 `GET /api/pro/v1/block`，后端缺少对应路由，页面打开触发 404 并提示“网络异常”。
- 修复：
  - 新增后端接口：
    - `GET /api/pro/v1/block?kb_id=...` 读取屏蔽词
    - `POST /api/pro/v1/block` 保存屏蔽词
  - 屏蔽词保存逻辑落到 `settings` 表 `key=block_words`，并在写入时去重、去空白。
- 验证（VM）：
  - 修复前：`/api/pro/v1/block` 返回 `404`
  - 修复后：登录后 `GET` 返回 `200`，`POST` 保存后再次 `GET` 可读回数据。

## 追加记录（2026-02-16，文档元信息显示开关）
- 新增设置项：`设置 -> 前台网站样式个性化 -> 文档元信息显示`
  - 显示创建时间
  - 显示更新时间
  - 显示字数
- 三个开关默认全开：历史数据未配置时，前台默认按开启处理。
- 前台文档详情页元信息渲染改为按开关动态拼装，支持任意组合显示，分隔符 `·` 自动处理，不会出现多余分隔符。
- 配置持久化字段：
  - `settings.node_meta_settings.show_created_at`
  - `settings.node_meta_settings.show_updated_at`
  - `settings.node_meta_settings.show_word_count`

## 追加记录（2026-02-16，管理员入口与发布页版本查看增强）
- 管理员设置页：
  - “新建用户”按钮移动到“选择已有用户”区域右上角，入口更直观。
- 文档属性弹窗“网络异常”修复：
  - 移除不存在的接口 `GET /api/pro/v1/auth/group/list` 调用，避免 404 触发全局错误提示。
  - 分组选择改为优先读取当前文档已有分组并作为候选项回填，保证属性弹窗可稳定打开与保存。
- 发布页面增强：
  - 新增后端接口 `GET /api/v1/knowledge_base/release/docs`，返回指定版本文档快照及与上一版本对比结果（新增/删除/修改/未变更）。
  - 发布页新增“查看版本”“文档对比”操作，支持在弹窗中查看版本内文档清单、版本差异统计与逐项对比。
  - 本次按复杂度评估暂不新增“回退发布版本”功能，避免引入跨版本状态回写风险。
- 迁移兼容：
  - 在仅保留 `full_fresh_deploy.sql` 的场景下，自动迁移启动逻辑改为“若不存在 `migration/*.up.sql` 则跳过增量迁移”，避免 API 因缺失历史迁移文件启动失败。

## 追加记录（2026-02-16，复制尾巴可配置与连续提问间隔落地）
- 安全设置 -> 内容复制：
  - “增加内容尾巴”新增可编辑文本框，支持自定义复制尾巴内容并持久化到 `settings.copy_append_content`。
  - 默认模板为：
    - `-----------------------------------------`
    - `{{content_from}} {{url}}`
  - 支持变量替换：`{{url}} / {url}`、`{{content_from}} / {content_from}`。
- 前台复制行为：
  - 文档页“复制 Markdown”与全局复制后缀统一改为读取 `copy_append_content`，不再硬编码固定尾巴文案。
  - 当模板为空时自动回退到默认模板，保证兼容历史配置。
- 问答设置：
  - “连续提问时间间隔（敬请期待）”改为可用配置，支持 `0-300` 秒（`0` 表示不限制）。
  - 配置落地字段：`settings.conversation_setting.ask_interval_seconds`。
- 分享问答限流：
  - `/share/v1/chat/message` 与 `/share/v1/chat/widget` 增加连续提问间隔校验。
  - 校验维度：同一知识库 + 应用 + 来源 IP，按最近一次用户消息时间计算剩余等待秒数。
  - 命中时返回：`提问过于频繁，请 N 秒后再试`。
- SQL 现状：
  - 当前仓库仅保留 1 个完整部署 SQL：
    - `backend/store/pg/migration/full_fresh_deploy.sql`
  - 未保留任何增量迁移 SQL 文件。
