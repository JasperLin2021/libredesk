---
name: widget-queue-footer-and-agent-assigned-reply
overview: 实现 widget 端两个需求：1) 用户点击"我要转人工"后，排队人数提示以底部固定条持续显示并实时刷新，直到会话被分配给客服；2) 会话被分配给客服时，后台预设的"分配后自动回复"以该客服身份发送，widget 显示该客服头像和名称。
design:
  architecture:
    framework: vue
  styleKeywords:
    - Minimalism
    - Amber accent
    - Clean compact bar
    - Status chip
  fontSystem:
    fontFamily: PingFang SC
    heading:
      size: 16px
      weight: 600
    subheading:
      size: 13px
      weight: 500
    body:
      size: 12px
      weight: 400
  colorSystem:
    primary:
      - "#D97706"
      - "#B45309"
    background:
      - "#FFFBEB"
      - "#FFFFFF"
      - "#FEF3C7"
    text:
      - "#92400E"
      - "#78350F"
    functional:
      - "#16A34A"
      - "#DC2626"
      - "#2563EB"
todos:
  - id: backend-meta-persistence
    content: 持久化 queue_info 到会话 meta，widget 查询返回 c.meta，并同步清理点删除 queue_info
    status: completed
  - id: backend-realtime-refresh
    content: 新增等待会话查询、RefreshWaitingQueueInfo 与 RunQueueInfoRefresher，接入 main/config 并即时触发
    status: completed
    dependencies:
      - backend-meta-persistence
  - id: backend-assigned-as-agent
    content: 实现 SendReplyAsUser，HandleUserAssigned 以客服身份发送并清理 queue_info，补充团队分配清理
    status: completed
    dependencies:
      - backend-meta-persistence
  - id: widget-sticky-footer
    content: 在 ChatMessages.vue 新增常驻底部排队条及显隐逻辑，用 accessibility-auditor 审计
    status: completed
    dependencies:
      - backend-realtime-refresh
      - backend-assigned-as-agent
---

## 产品概述

优化 widget 端"转人工"排队体验与分配通知：排队人数在聊天区底部常驻显示并实时刷新，直到会话被分配给客服；会话分配后，后台预设的接入人工文案改为以该客服身份发送，widget 中展示客服头像与名称。

## 核心功能

1. **排队人数常驻底部显示**：用户点击"我要转人工"后，聊天区底部固定显示"当前排队人数为 x 人"，不随消息滚动消失，直到该会话被分配给客服。
2. **排队人数实时刷新**：当队列中其他会话被分配、关闭或新增排队会话时，底部排队人数自动更新（周期刷新 + 分配事件即时刷新）。
3. **分配后以客服身份自动回复**：会话分配给客服后，仍发送后台预设的 inbox 级"接入人工"文案（assigned_reply），但消息以被分配客服身份发出，widget 中显示该客服的头像和名称；同时底部排队条自动消失。
4. **历史消息保留**：原"排队人数"消息气泡保留在消息列表中，新增的常驻底部条是独立展示。

## 技术栈

- 后端：Go + PostgreSQL（conversations.meta JSONB）+ WebSocket（现有 livechat 通道）
- 前端：Vue 3 widget（现有 Tailwind 组件体系）

## 实现方案

### 后端

1. **queue_info 持久化**：`quickreply.handleTransferRequest` 在发送排队回复后，把 `queue_info`（含 count）写入会话 meta（`UpdateConversationMeta`），与 `bot_human_requested=true` 一并保存。新增常量 `ConversationMetaQueueInfo = "queue_info"`（models.go）。
2. **widget 查询返回 meta**：`get-chat-conversation`、`get-contact-chat-conversations` 增加 `c.meta` 列；`ChatConversation` 增加 `Meta json.RawMessage` 字段（db/json tag 均为 `meta`），使 widget 首次加载即可读取 `meta.queue_info`。
3. **实时刷新**：

- 新增查询 `get-waiting-human-conversations`：Open、`assigned_user_id/assigned_team_id` 均为 NULL、`meta->>'bot_human_requested'='true'`，返回 uuid/contact_id/inbox_id。
- 新增方法 `RefreshWaitingQueueInfo()`：重算 `CountOpenUnassignedConversations()`，对每个等待会话执行 `UpdateConversationMeta(uuid, {"queue_info": {"count": N}})` 并 `BroadcastConversationToWidget(uuid, contactID, inboxID, {"queue_info": {"count": N}})`。
- 新增 `RunQueueInfoRefresher(ctx, interval)`（参照现有 `RunUnsnoozer` 模式），在 `cmd/main.go` 启动，间隔取新配置键 `conversation.queue_info_refresh_interval`（默认 30s，config.sample.toml 补充）。
- 触发点：`handleTransferRequest` 末尾立即调用一次（保证底部条即时出现）；`afterUserAssignedHooks` 中调用（分配后其余等待会话计数即时减一）。

4. **分配后以客服身份发送**：

- `quickreply.ConversationService` 接口新增 `SendReplyAsUser(userID, inboxID, contactID int, conversationUUID, content string, metaMap) (cmodels.Message, error)` 与 `RefreshWaitingQueueInfo() error`；`conversation.Manager` 实现 `SendReplyAsUser`（内部直接调用 `QueueReply(nil, inboxID, userID, ...)`）。
- `HandleUserAssigned`：`conversation.AssignedUserID.Valid` 时用 `SendReplyAsUser(agentID, ...)` 发送 `cfg.AssignedReply`，否则回退 `SendAutoReply`；发送后 `DeleteConversationMetaKey(uuid, "queue_info")` 清理 meta。
- `afterUserAssignedHooks` 的 assignee 广播数据中附带 `"meta": {"queue_info": nil}`，widget 通过 deepMerge 把 `meta.queue_info` 置 null，底部条立即隐藏（同时依赖 assignee 判断）。
- `UpdateConversationTeamAssignee`（仅团队分配、无具体客服）补充：删除 `queue_info` meta 并广播 `{"meta": {"queue_info": null}}`，避免底部条残留。

5. **清理点同步**：`ReopenAndUnassignConversation`、`handleChatInit`、`handleGetConversations` 三处清除 `bot_human_requested` 的位置，同步 `DeleteConversationMetaKey(uuid, "queue_info")`。
6. **头像签名**：`broadcastMessageToWidgetClients` 已调用 `SignAvatarURL`，以客服身份发送的消息头像自动签名，无需额外改动。

### 前端 widget（ChatMessages.vue）

- 在根容器（`flex flex-col relative flex-1 min-h-0`）的滚动区之后、ScrollToBottomButton 之前，新增常驻底部条：`v-if="showQueueInfoFooter"`，样式沿用现有 queue_info 琥珀色 chip（Clock 图标 + `$t('widget.queuePosition', {count})`），加 `border-t` 与浅琥珀背景。
- 新增 computed：`showQueueInfoFooter` = 会话存在且 `status==='Open'` 且 `assignee` 为空 且 `meta?.queue_info?.count` 为数字；`queueInfoCount` 取 `meta.queue_info.count`。
- 数据流：`loadConversation`/`handleChatInit` 响应中的 `conversation.meta` 提供初始值；`RefreshWaitingQueueInfo` 的 CONVERSATION_UPDATE 广播经 `updateCurrentConversation` deepMerge 更新 count；分配广播（assignee + `meta.queue_info=null`）驱动隐藏。历史 queue_info 气泡渲染保持不变。

## 性能与可靠性

- `RefreshWaitingQueueInfo`：1 次等待会话查询 + 1 次全局计数 + N 次 meta 更新/广播，N 为排队会话数（通常很小），30s 周期低频执行，开销可忽略；分配事件即时刷新保证实时性。
- deepMerge 对 `{"queue_info": null}` 会直接置空目标键，隐藏逻辑可靠；广播仅发往对应会话的 widget 客户端，无全局推送。
- 边界：会话关闭（status 非 Open）或团队分配时底部条隐藏；历史 queue_info 气泡不删除，仅新增常驻条。

## 目录结构

```
internal/quickreply/quickreply.go          [MODIFY] ConversationService 接口新增 SendReplyAsUser/RefreshWaitingQueueInfo；handleTransferRequest 持久化 queue_info 并立即刷新；HandleUserAssigned 以客服身份发送并清理 queue_info
internal/conversation/conversation.go      [MODIFY] 实现 SendReplyAsUser、RefreshWaitingQueueInfo；queries 结构新增 GetWaitingHumanConversations；afterUserAssignedHooks 广播 queue_info null 并刷新；ReopenAndUnassignConversation 清理 queue_info；UpdateConversationTeamAssignee 清理并广播
internal/conversation/queries.sql          [MODIFY] 两个 chat 查询加 c.meta；新增 get-waiting-human-conversations 查询
internal/conversation/models/models.go     [MODIFY] ChatConversation 增加 Meta 字段；新增常量 ConversationMetaQueueInfo
internal/conversation/queue_refresher.go   [NEW] RunQueueInfoRefresher 周期任务（参照 unsnoozer.go 模式）
cmd/main.go                                [MODIFY] 读取 queue_info_refresh_interval 配置并启动 RunQueueInfoRefresher
cmd/chat.go                                [MODIFY] handleChatInit / handleGetConversations 清理 queue_info
config.sample.toml                         [MODIFY] [conversation] 增加 queue_info_refresh_interval = "30s"
frontend/apps/widget/src/components/ChatMessages.vue  [MODIFY] 新增常驻底部排队条与 computed 逻辑
```

## 设计说明

widget 整体视觉保持不变，仅新增一个常驻底部排队信息条。位置：消息滚动区下方、输入框上方，全宽横条，顶部细分隔线，浅琥珀底色 + 琥珀文字，左侧 Clock 图标 + "当前排队人数为 x 人"文案，与消息列表中已有的 queue_info 琥珀色 chip 视觉一致，确保用户认知连续。条内居中布局、紧凑留白，不干扰消息阅读；会话分配后整条淡出隐藏，无动画过度设计。

## 页面与区块

- 消息区（不变）：历史 queue_info 气泡保留
- 底部排队条（新增）：v-if 条件渲染，status/assignee/meta 三条件控制显隐，role="status" 提升可访问性

## Agent Extensions

### Skill

- **accessibility-auditor**
- 目的：对 widget 新增的常驻排队底部条进行 WCAG 2.2 可访问性审计，重点核查 role="status" 的 ARIA 用法、颜色对比度（琥珀文字 #92400E 在 #FFFBEB 背景上）与键盘/读屏语义。
- 预期结果：输出审计结论与必要的修复建议，确保新增 UI 满足可访问性要求后再交付。