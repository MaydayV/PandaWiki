# 飞书机器人增强 + 知识库更新主动推送 — 开发计划

> 状态（2026-08-01）：Phase 1–3 已落地；Phase 4 单元测试已补齐（模板渲染 / Webhook 签名 / 带密钥发送）。
> Discord / 企业微信推送未实现（当前支持飞书、钉钉 Webhook）。
> 签名校验配置格式：`WebhookURL|密钥`。

## 一、现状分析

### 1.1 飞书机器人已有实现

飞书机器人**已经存在**于代码库中，核心文件：

| 文件 | 功能 |
|------|------|
| `backend/pkg/bot/feishu/stream.go` | 飞书机器人核心客户端，WebSocket 事件监听 |
| `backend/pkg/bot/lark/client.go` | Lark 国际版客户端，HTTP 回调模式 |
| `backend/pkg/feishu/feishu.go` | 飞书 OAuth 登录（后台管理用） |
| `backend/domain/app.go` | 配置类型：`FeishuBotIsEnabled`、`FeishuBotAppID`、`FeishuBotAppSecret` |
| `backend/usecase/app.go` | 生命周期管理：`updateFeishuBot()`、`updateLarkBot()` |

已实现的功能：
- ✅ WebSocket 长连接事件监听（飞书）/ HTTP 回调（Lark 国际版）
- ✅ 互动卡片（Interactive Card）流式输出回答
- ✅ 单聊（p2p）和群聊（group @机器人）支持
- ✅ 消息去重（sync.Map + 5 分钟 TTL）
- ✅ 获取用户信息（open_id → 姓名/头像/UserId）
- ✅ Markdown 格式回答

### 1.2 与钉钉机器人的差异

| 功能点 | 钉钉 | 飞书 | 差距 |
|--------|------|------|------|
| 流式卡片回复 | ✅ | ✅ | — |
| 单聊/群聊 | ✅ | ✅ | — |
| 用户信息获取 | ✅ userid/姓名/头像/邮箱/工号 | ✅ open_id/姓名/头像/UserId | 飞书缺少邮箱字段提取 |
| 消息去重 | ✅ inFlight + TTL + 可重试 | ✅ sync.Map + TTL | 钉钉更完善：处理中不可清理、失败可重试 |
| 异步处理 | ✅ goroutine + panic recovery | ✅ goroutine | 飞书缺少 panic recovery |
| 错误兜底 | ✅ 卡片更新失败回退"出错了" | ❌ 失败直接 return | 需补齐 |
| @提及替换 | N/A | ✅ Lark 版有 `replaceMentions` | 飞书 WS 版缺失 |
| **知识库更新主动推送** | ❌ | ❌ | 全新功能 |

### 1.3 结论

飞书机器人基础功能**已基本完备**，本次工作聚焦于：

1. **补齐飞书与钉钉的细节差距**（错误兜底、panic recovery、邮箱字段）
2. **新增知识库更新主动推送功能**（全机器人平台通用，含后台配置开关）

---

## 二、开发计划

### Phase 1：飞书机器人健壮性增强

#### 1.1 增加错误兜底与 panic recovery

**文件**：`backend/pkg/bot/feishu/stream.go`

- `sendQACard` 方法中，卡片更新失败时回退发送纯文本消息"出错了，请稍后再试"
- 外层 `processMessageAsync` 增加 `defer recover()`，防止 panic 导致 goroutine 泄漏
- QA 流式输出循环中，单次卡片更新失败不中断整体回答（记录日志继续）

#### 1.2 补充用户信息字段

**文件**：`backend/pkg/bot/feishu/stream.go`

- `sendQACard` 中调用 `GetUserInfo` 后，补充提取 `Email`（飞书 User 结构体已有此字段）
- 写入 `convInfo.UserInfo.Email`

#### 1.3 飞书 WS 版群聊 @提及文本清理

**文件**：`backend/pkg/bot/feishu/stream.go`

- 群聊消息中，飞书 WS 事件的 `content.text` 可能包含 `@_user_N` 占位符
- 参照 Lark 版的 `replaceMentions`，增加提及文本清理逻辑

---

### Phase 2：知识库更新主动推送

#### 2.1 域模型扩展

**文件**：`backend/domain/app.go`

在 `AppSettings` 中新增推送配置字段：

```go
// KnowledgeBaseUpdatePush 知识库更新推送配置
KBUpdatePushEnabled   bool   `json:"kb_update_push_enabled,omitempty"`    // 推送总开关
KBUpdatePushContent   string `json:"kb_update_push_content,omitempty"`   // 推送模板内容（支持变量）
KBUpdatePushChatIDs   string `json:"kb_update_push_chat_ids,omitempty"`  // 推送目标群聊 ID 列表（逗号分隔）
```

#### 2.2 推送消息模板

支持变量：
- `{kb_name}` — 知识库名称
- `{version}` — 版本号
- `{doc_count}` — 本次发布的文档数
- `{added}` — 新增文档数
- `{updated}` — 更新文档数
- `{removed}` — 删除文档数
- `{release_time}` — 发布时间

默认模板：
```
📚 知识库「{kb_name}」已更新
版本：{version} | 文档：{doc_count} 篇
新增 {added} · 更新 {updated} · 删除 {removed}
发布时间：{release_time}
```

#### 2.3 推送触发点

**文件**：`backend/usecase/knowledge_base.go` — `CreateKBRelease` 方法

在 release 创建成功后，异步触发推送：
- 检查该知识库关联的 App 设置中 `KBUpdatePushEnabled` 是否开启
- 遍历 `KBUpdatePushChatIDs`，向每个目标群聊发送推送消息
- 推送失败不影响 release 创建流程（仅记录日志）

#### 2.4 多平台推送实现

**新增文件**：`backend/pkg/bot/push.go` — 统一推送接口

```go
type PushNotifier interface {
    SendTextMessage(ctx context.Context, chatID string, content string) error
    SendRichMessage(ctx context.Context, chatID string, title, content string) error
}
```

各平台实现：

##### 飞书推送（已验证 API）

- **文件**：`backend/pkg/bot/feishu/push.go`
- **API**：`POST https://open.feishu.cn/open-apis/im/v1/messages?receive_id_type=chat_id`
- **权限**：`im:message:send_as_bot`
- **认证**：Bearer `tenant_access_token`（SDK 自动管理）
- **限流**：同群 5 QPS
- **消息类型**：`msg_type=text`（纯文本）或 `msg_type=post`（富文本，支持链接和加粗）
- **已有 SDK**：直接复用 `larkim.NewCreateMessageReqBuilder()`，代码中已有此模式

```go
// 飞书推送核心代码（复用现有 SDK client）
func (c *FeishuClient) SendTextMessage(ctx context.Context, chatID, content string) error {
    msgContent, _ := json.Marshal(map[string]string{"text": content})
    resp, err := c.client.Im.Message.Create(ctx,
        larkim.NewCreateMessageReqBuilder().
            ReceiveIdType("chat_id").
            Body(larkim.NewCreateMessageReqBodyBuilder().
                MsgType("text").
                ReceiveId(chatID).
                Content(string(msgContent)).
                Build()).
            Build())
    // handle resp ...
}
```

##### 钉钉推送

- **文件**：`backend/pkg/bot/dingtalk/push.go`
- **方案 A（企业内部机器人）**：`POST https://api.dingtalk.com/v1.0/robot/oToMessages/batchSend`（单聊）或群聊 API
- **方案 B（自定义 Webhook）**：`POST https://oapi.dingtalk.com/robot/send?access_token=XXX`，支持 HMAC-SHA256 签名
- **推荐**：方案 A（与现有 DingTalkClient 复用同一 access_token），回退方案 B（Webhook 更简单但需额外配置）
- **消息类型**：`msgtype=markdown`（支持标题 + 正文）

##### 企业微信推送

- **文件**：`backend/pkg/bot/wechat_service/push.go`
- **API**：群聊 Webhook `POST https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=XXX`
- **消息类型**：`msgtype=markdown`

##### Discord 推送

- **文件**：`backend/pkg/bot/discord/push.go`
- **API**：`POST /channels/{channel_id}/messages`
- **消息类型**：`content`（纯文本，支持 Markdown）

#### 2.5 推送服务编排

**新增文件**：`backend/usecase/push.go` — 推送编排逻辑

```go
type PushUsecase struct {
    appRepo   repo.AppRepo
    logger    *log.Logger
    // 各平台 notifier 按 appID 缓存
}

func (u *PushUsecase) NotifyKBUpdate(ctx context.Context, kbID string, release *domain.KBRelease) error {
    // 1. 根据 kbID 找到关联的 App 列表
    // 2. 过滤出 KBUpdatePushEnabled=true 的 App
    // 3. 根据 App 类型选择对应 PushNotifier
    // 4. 渲染模板并发送到 KBUpdatePushChatIDs 中的每个群聊
}
```

---

### Phase 3：后台配置界面

#### 3.1 前端配置组件

**文件**：`web/admin/src/pages/setting/component/CardPush.tsx`（新增）

配置项：
- **推送总开关** — Switch 组件
- **推送目标群聊** — 输入框（支持多个 chat_id，逗号分隔）
- **推送模板** — 多行文本框，带变量提示
- **测试发送** — 按钮，发送测试消息到指定群聊

#### 3.2 设置页集成

**文件**：`web/admin/src/pages/setting/index.tsx`

在设置页中新增"消息推送"Tab/Section，放置 `CardPush` 组件。

#### 3.3 API 接口

复用已有的 App Settings 更新接口（`PUT /api/v1/app/settings`），新字段通过 `settings` 对象传递。

---

### Phase 4：测试与验证

#### 4.1 单元测试

- `backend/pkg/bot/feishu/stream_test.go` — 消息去重、panic recovery
- `backend/usecase/push_test.go` — 模板渲染、推送编排逻辑
- `backend/pkg/bot/dingtalk/push_test.go` — 钉钉推送消息构造

#### 4.2 集成验证

- 飞书机器人单聊问答 → 验证流式卡片
- 飞书机器人群聊 @问答 → 验证提及清理 + 卡片
- 发布知识库新版本 → 验证推送到配置的群聊
- 关闭推送开关 → 验证不推送
- 推送失败 → 验证不影响 release 流程

---

## 三、关键文件清单

### 需修改的文件

| 文件 | 修改内容 |
|------|----------|
| `backend/domain/app.go` | 新增 `KBUpdatePush*` 配置字段 |
| `backend/pkg/bot/feishu/stream.go` | 错误兜底、panic recovery、邮箱提取、@提及清理 |
| `backend/usecase/knowledge_base.go` | `CreateKBRelease` 中触发推送 |
| `backend/usecase/app.go` | 注册推送相关依赖 |
| `web/admin/src/pages/setting/index.tsx` | 集成推送配置组件 |

### 需新增的文件

| 文件 | 内容 |
|------|------|
| `backend/pkg/bot/push.go` | `PushNotifier` 接口定义 |
| `backend/pkg/bot/feishu/push.go` | 飞书推送实现 |
| `backend/pkg/bot/dingtalk/push.go` | 钉钉推送实现 |
| `backend/usecase/push.go` | 推送编排逻辑 |
| `web/admin/src/pages/setting/component/CardPush.tsx` | 推送配置 UI |
| `backend/pkg/bot/feishu/stream_test.go` | 飞书机器人测试 |
| `backend/usecase/push_test.go` | 推送逻辑测试 |

---

## 四、风险与注意事项

1. **飞书权限**：主动发送消息需要 `im:message:send_as_bot` 权限，需在飞书开放平台配置
2. **群聊 ID 获取**：用户需在后台手动填入目标群聊 chat_id（可通过飞书机器人收到的群聊事件日志获取）
3. **推送频率**：高频发布时需考虑限流，单次推送间隔不低于 1 秒
4. **钉钉群聊 ID**：钉钉群聊推送需要 `chat_id`，获取方式与飞书不同（通过 Webhook 或 API）
5. **消息格式**：飞书和钉钉的 Markdown 语法不完全一致，模板需做平台适配
