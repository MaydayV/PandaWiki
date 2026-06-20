# 乘风版更新日志

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
