---
name: long-no-reply-timeout-reply
overview: 在后台 /admin/quick-reply 自动回复配置中新增"长时间未回复文案"（no_reply_timeout_reply）：当用户距最后一条客服消息超时（默认5min，从 config.toml 读取）未回复时，以当前已分配客服口吻发送该文案，随后自动关闭会话触发"会话关闭文案"。
todos:
  - id: quickreply-config-field
    content: 在 schema.sql、quickreply models/queries/quickreply.go、cmd/quickreply.go 中新增 no_reply_timeout_reply 配置字段
    status: completed
  - id: no-reply-timeout-logic
    content: 实现 get-stale-agent-conversations 查询、CloseConversation、HandleNoReplyTimeout 及防重入标记，ReopenAndUnassignConversation 清理标记
    status: completed
    dependencies:
      - quickreply-config-field
  - id: config-wiring
    content: 在 cmd/main.go 读取 no_reply_timeout 配置并启动 RunNoReplyTimeoutRefresher，更新 config.sample.toml 与 config.toml
    status: completed
    dependencies:
      - no-reply-timeout-logic
  - id: admin-form-i18n
    content: 在 QuickReplyConfig.vue 新增长时间未回复文案表单字段，补充 zh-CN/en-US i18n 文案
    status: completed
    dependencies:
      - quickreply-config-field
  - id: verify-build
    content: 执行 go build/vet 与前端构建验证，用 [skill:accessibility-auditor] 审计新增表单字段
    status: completed
    dependencies:
      - admin-form-i18n
      - config-wiring
---

## 产品概述

在后台系统的"自动回复文案"（`/admin/quick-reply`）中新增一项"长时间未回复文案"。当客服最后一条消息发送后，用户在配置的超时时间内（默认 5 分钟，从 config.toml 读取）未再发送消息，系统自动以当前已分配客服的口吻发送预设文案，随后自动将会话置为关闭并触发现有"会话关闭文案"。

## 核心功能

1. **后台配置项**：在自动回复文案配置表单中新增"长时间未回复文案"输入框（文本域），随其他文案一并保存/回显。
2. **超时检测**：周期扫描"已分配给具体客服、状态为打开、最后一条消息为客服消息且距今超过超时时间"的会话；客服尚未回复用户时不触发。
3. **以客服口吻发送**：超时后先以当前已分配客服身份发送"长时间未回复文案"（沿用现有 SendReplyAsUser 机制，widget 端显示该客服头像与名称）。
4. **自动关闭会话**：文案发送后自动将会话置为关闭，通过既有状态变更链路触发现有"会话关闭文案"（closed_reply），保证两条文案按顺序发送。
5. **超时时间可配置**：从 `config.toml` 的 `[conversation]` 段读取（如 `no_reply_timeout = "5m"`），支持热配置重启生效。
6. **防重复触发**：每个会话仅触发一次（会话 meta 记录 `no_reply_timeout_sent` 标记），重新打开会话后可再次触发。

## 技术栈

- 后端：Go + PostgreSQL（`inbox_quick_reply_configs` 表新增列、`conversations.meta` JSONB 防重入标记、`conversation_messages` 消息时间筛选）
- 前端：Vue 3 + VeeValidate/Zod 表单（既有后台管理组件体系）
- 配置：koanf（`config.toml`），沿用 `cmp.Or(ko.Duration(...), 默认值)` 读取模式

## 实现方案

### 后端：配置持久化（no_reply_timeout_reply）

1. `schema.sql`：`inbox_quick_reply_configs` 表新增列 `no_reply_timeout_reply TEXT NULL`。
2. `internal/quickreply/models/models.go`：`InboxQuickReplyConfig` 增加 `NoReplyTimeoutReply string \`db:"no_reply_timeout_reply" json:"no_reply_timeout_reply"\``。
3. `internal/quickreply/queries.sql`：`get-config` 增加该列；`upsert-config` 插入/更新列（参数由 $1-$7 扩为 $1-$8，新增参数为 `noReplyTimeoutReply`）。
4. `internal/quickreply/quickreply.go`：`UpsertConfig` 签名增加 `noReplyTimeoutReply string` 参数并透传。
5. `cmd/quickreply.go`：`quickReplyConfigRequest` 增加 `NoReplyTimeoutReply string \`json:"no_reply_timeout_reply"\``，加入长度校验（复用 `maxQuickReplyFieldLength`），`UpsertConfig` 调用传新参。

### 后端：超时检测与处理

1. **新查询** `internal/conversation/queries.sql` 新增 `get-stale-agent-conversations`（参考 `get-waiting-human-conversations` 模式），返回 `uuid, inbox_id, contact_id, assigned_user_id`，条件：

- `assigned_user_id IS NOT NULL`（仅具体客服分配）
- 状态为 `Open`（join `conversation_statuses` 按 name 过滤，排除 Snoozed/Resolved/Closed）
- `meta->>'no_reply_timeout_sent' IS DISTINCT FROM 'true'`（防重入）
- 最后一条消息 `sender_type='agent'`（客服回复后用户未回）
- 最后一条**非 System 用户**的 agent 消息时间 `< NOW() - $1::interval`（排除系统自动文案，避免转人工等系统消息误触发）

2. **常量** `internal/conversation/models/models.go` 新增 `ConversationMetaNoReplyTimeoutSent = "no_reply_timeout_sent"`。
3. **接口扩展** `internal/quickreply/quickreply.go`：`ConversationService` 接口新增 `CloseConversation(uuid string) error`。
4. **conversation.Manager 实现**：

- `CloseConversation(uuid)`：内部获取系统用户（`userStore.GetSystemUser()`，已有先例 L131/L1015），调用 `UpdateConversationStatus(uuid, 0, models.StatusClosed, "", systemUser)`——该函数在 status=="Closed" 时自动触发 `quickReply.HandleConversationClosed`（L1210-1214），从而发送"会话关闭文案"，无需查 statusID。
- 新查询结构体字段注册（queries 结构新增 `GetStaleAgentConversations`）。

5. **HandleNoReplyTimeout**（quickreply.Manager 新增）：

- 读取 inbox config；读取 conversation meta 检查 `no_reply_timeout_sent`，已标记则跳过；
- 先 `UpdateConversationMeta(uuid, {"no_reply_timeout_sent": true})` 防重入（失败则返回错误跳过本轮）；
- `cfg.Enabled` 且文案非空时：`AssignedUserID.Valid` → `SendReplyAsUser(assignedUserID, ...)` 以客服口吻发送；否则兜底 `SendAutoReply`；发送失败仅记录日志（不阻止关闭）；
- 最后调用 `CloseConversation(uuid)` 自动关闭（顺序保证：先"长时间未回复文案"，后"会话关闭文案"）；文案为空时跳过发送直接关闭。

6. **周期任务** `internal/conversation/queue_refresher.go` 新增 `RunNoReplyTimeoutRefresher(ctx, timeout, interval)`（仿照 `RunQueueInfoRefresher` ticker 模式）：每周期执行新查询获取超时会话列表 → 对每个 uuid `GetConversation(0, uuid, "")` → `c.quickReply.HandleNoReplyTimeout(conversation)`，错误记录日志。

### 配置接入

- `cmd/main.go`：新增 `noReplyTimeout = cmp.Or(ko.Duration("conversation.no_reply_timeout"), 5*time.Minute)`、`noReplyTimeoutScanInterval = cmp.Or(ko.Duration("conversation.no_reply_timeout_scan_interval"), 30*time.Second)`，并启动 `go conversation.RunNoReplyTimeoutRefresher(ctx, noReplyTimeout, noReplyTimeoutScanInterval)`。
- `config.sample.toml`、`config.toml`：`[conversation]` 段新增 `no_reply_timeout = "5m"` 与 `no_reply_timeout_scan_interval = "30s"`。

### 前端后台管理

- `frontend/apps/main/src/views/admin/quickreply/QuickReplyConfig.vue`：在 `closed_reply` 字段之后新增 `no_reply_timeout_reply` FormField（Textarea rows="2"）；zod schema、`loadConfig`、`saveConfig` payload 同步增加该字段。
- i18n：`i18n/zh-CN.json` 新增 `admin.quickReply.noReplyTimeoutReply`（"长时间未回复文案"）与 `.description`（"客服最后回复后，用户长时间未回复时以客服口吻自动发送，随后自动关闭会话。"）；`i18n/en-US.json` 对应英文文案。

## 性能与可靠性

- 超时扫描为低频周期任务（默认 30s），查询按 `conversation_messages.conversation_id` 索引子查询取最后一条消息，会话数级开销可忽略；仅命中超时会话才执行 meta 更新与发送。
- 防重入通过 meta 标记保证单次触发；发送失败不阻塞关闭（确保"会话关闭文案"一定触发）；meta 标记失败则跳过本轮避免并发重复发送。
- 边界：仅处理 `assigned_user_id` 非空且状态 Open 的会话；团队分配会话跳过（`HandleNoReplyTimeout` 内兜底用系统口吻）；重新打开会话（ReopenAndUnassignConversation 路径）后 meta 标记仍在，需在 `ReopenAndUnassignConversation` 同步 `DeleteConversationMetaKey(uuid, ConversationMetaNoReplyTimeoutSent)` 以支持再次触发。
- widget 端无需改动：以客服身份发送的消息自动按真实客服渲染（头像+名称）。

## 目录结构

```
schema.sql                                          [MODIFY] inbox_quick_reply_configs 增加 no_reply_timeout_reply 列
internal/quickreply/models/models.go                [MODIFY] InboxQuickReplyConfig 增加 NoReplyTimeoutReply 字段
internal/quickreply/queries.sql                     [MODIFY] get-config/upsert-config 增加该列
internal/quickreply/quickreply.go                   [MODIFY] UpsertConfig 加参；ConversationService 接口加 CloseConversation；新增 HandleNoReplyTimeout
internal/conversation/models/models.go              [MODIFY] 新增 ConversationMetaNoReplyTimeoutSent 常量
internal/conversation/queries.sql                   [MODIFY] 新增 get-stale-agent-conversations 查询
internal/conversation/conversation.go               [MODIFY] 实现 CloseConversation；queries 结构注册新查询
internal/conversation/queue_refresher.go            [MODIFY] 新增 RunNoReplyTimeoutRefresher
cmd/main.go                                         [MODIFY] 读取 no_reply_timeout 配置并启动 refresher
cmd/quickreply.go                                   [MODIFY] quickReplyConfigRequest 加字段、校验、传参
config.sample.toml                                  [MODIFY] [conversation] 增加 no_reply_timeout / no_reply_timeout_scan_interval
config.toml                                         [MODIFY] 同上（本地运行配置）
frontend/apps/main/src/views/admin/quickreply/QuickReplyConfig.vue  [MODIFY] 新增 no_reply_timeout_reply 表单字段
i18n/zh-CN.json                                     [MODIFY] 新增 admin.quickReply.noReplyTimeoutReply 文案
i18n/en-US.json                                     [MODIFY] 新增对应英文文案
```

## 关键接口定义

```
// quickreply.ConversationService 新增
CloseConversation(uuid string) error

// conversation.Manager 实现
func (c *Manager) CloseConversation(uuid string) error {
    systemUser, err := c.userStore.GetSystemUser()
    if err != nil { return err }
    return c.UpdateConversationStatus(uuid, 0, models.StatusClosed, "", systemUser)
}

// quickreply.Manager 新增
func (m *Manager) HandleNoReplyTimeout(conversation cmodels.Conversation) error

// conversation 包新查询（queries.sql）
-- name: get-stale-agent-conversations
SELECT c.uuid, c.inbox_id, c.contact_id, c.assigned_user_id
FROM conversations c
JOIN conversation_statuses cs ON cs.id = c.status_id
WHERE c.assigned_user_id IS NOT NULL
  AND cs.name = 'Open'
  AND (c.meta->>'no_reply_timeout_sent') IS DISTINCT FROM 'true'
  AND (SELECT cm.sender_type FROM conversation_messages cm
       WHERE cm.conversation_id = c.id ORDER BY cm.id DESC LIMIT 1) = 'agent'
  AND (SELECT MAX(cm.created_at) FROM conversation_messages cm
       JOIN users u ON u.id = cm.sender_id
       WHERE cm.conversation_id = c.id AND cm.sender_type = 'agent' AND u.email <> 'System'
      ) < NOW() - $1::interval
```

## Agent Extensions

### Skill

- **accessibility-auditor**
- 目的：对后台"自动回复文案"表单新增的"长时间未回复文案"输入项进行 WCAG 2.2 可访问性审计（FormLabel/FormControl 关联、错误提示、键盘可操作性与视觉对比度）。
- 预期结果：输出审计结论与必要的修复建议，确保新增表单字段满足可访问性要求后再交付。