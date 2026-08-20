---
name: widget-message-avatars
overview: 在 widget 端 `ChatMessages.vue` 中为每条消息气泡加上头像：用户消息显示用户头像，agent 真人回复显示客服头像，AI/system/macro 等非真人回复显示收件箱头像。布局参考截图：客服/系统消息头像在左气泡在左，用户消息头像在右气泡在右。
design:
  architecture:
    framework: vue
  styleKeywords:
    - Clean
    - Conversational
    - Avatar-Bubble Layout
    - Right/Left Aligned
  fontSystem:
    fontFamily: inherit
    heading:
      size: text-base
      weight: .nan
    subheading:
      size: text-sm
      weight: .nan
    body:
      size: text-sm
      weight: .nan
  colorSystem:
    primary:
      - "#0A6AFF"
    background:
      - "#FFFFFF"
      - "#F4F4F5"
    text:
      - "#0A0A0A"
      - "#71717A"
    functional:
      - "#22C55E"
      - "#EAB308"
      - "#EF4444"
todos:
  - id: add-avatar-helpers
    content: 在 ChatMessages.vue 中新增 isUserMessage / isSpecialMessage / getMessageAvatarUrl / getMessageAvatarFallback 四个纯函数，并导入 Avatar/AvatarImage/AvatarFallback
    status: completed
  - id: restructure-message-row
    content: 重构消息外层 div 为 flex row + 头像列 + 消息子列（按 author.type 决定 row-reverse / items-end / items-start），保留原气泡、引用、附件、元信息全部内容
    status: completed
    dependencies:
      - add-avatar-helpers
  - id: typing-indicator-avatar
    content: 为 typing 指示器容器加一个收件箱头像（self-end，与客服消息视觉一致）
    status: completed
    dependencies:
      - add-avatar-helpers
  - id: verify-build
    content: 运行 vite build 验证 widget 编译通过，检查 ChatMessages.vue 无新增 lint 错误
    status: completed
    dependencies:
      - restructure-message-row
      - typing-indicator-avatar
---

## Product Overview

在 widget 端的聊天消息列表中，为每条消息气泡加上头像，形成「头像 + 气泡 + 元信息」的左/右对话式布局，与主流即时通讯界面一致。

## Core Features

- 用户消息（contact / visitor）：头像在右，对齐气泡底部；无头像时回退显示用户首字母（默认 V）
- 客服消息（agent）：显示对应客服的真实头像在左，无头像时回退客服首字母
- 系统/AI/macro 等自动回复消息：统一显示收件箱自定义头像（`config.avatar_url` → `config.launcher.logo_url`）在左，无头像时回退品牌首字母
- 头像大小约 32px（size-8），与气泡底部对齐；不挤压气泡文字区
- 快速回复（QuickReply）、排队提示（queue_info）、满意度（CSAT）三类特殊消息保持原样，不加头像
- typing 指示器同步加上收件箱头像在左，保持会话视觉一致

## Tech Stack

- Vue 3.5 + `<script setup>` + Composition API
- 复用现有 `@shared-ui/components/ui/avatar`（`Avatar` / `AvatarImage` / `AvatarFallback`）
- 复用 `widgetStore.config.avatar_url`、`widgetStore.config.brand_name`、`userStore.firstName` 等已有状态
- 不新增依赖，不改后端，不改 schema，不改 i18n

## Implementation Approach

- 仅修改 `frontend/apps/widget/src/components/ChatMessages.vue` 一个文件，模板与 script 同步调整
- 把「消息外层 `<div>`」从 `flex flex-col items-end/start` 改为 `flex flex-row gap-2`，按作者类型决定 `flex-row` 或 `flex-row-reverse`，在 row 中并列渲染头像与原气泡
- 原气泡、引用展开、附件、元信息整段作为消息子列（`flex flex-col`），保持其内部 `max-w-[85%]`、颜色、状态指示等现有样式与逻辑
- 新增两个纯函数 `getMessageAvatarUrl(message)` / `getMessageAvatarFallback(message)`：基于 `message.author.type` 决策：
- `contact` / `visitor` → 用户侧（`message.author.avatar_url`，pending 消息时取 `userStore.firstName` 首字母）
- `agent` → 客服侧（`message.author.avatar_url` + `first_name` 首字母）
- 其它（system、ai_assistant、macro 等自动回复）→ 收件箱侧（`config.avatar_url || config.launcher?.logo_url` + `brand_name` 首字母）
- typing 指示器当前是单独的 `flex flex-col items-start` 容器，同步加一个收件箱头像，结构与普通 agent 消息保持一致

## Implementation Notes

- 复用现有 `ConversationsList.vue` 已有的 Avatar 使用方式（`:src` + `:initials` 模式），保持视觉一致
- 不动 `author.type` 解析逻辑（`store/chat.js:55-67` 已正确填充 `avatar_url` / `first_name` / `type`）
- 不改 pending 消息结构，确保发送中/失败的本地消息也能正确显示用户头像
- 不触碰 `getMessageTime`、`isQuotedTextVisible`、`handleCSATSubmitted` 等无关逻辑，回归面最小
- 后端 `MessageAuthor.AvatarURL` 字段（contact / agent 都有）已存在且 widget store 已读取，无需任何后端改动

## Key Code Structures

```js
// 头像 URL 选择（按消息 author 类型分层）
function isUserMessage (message) {
  return message.author?.type === 'contact' || message.author?.type === 'visitor'
}

function getMessageAvatarUrl (message) {
  if (isUserMessage(message)) return message.author?.avatar_url || ''
  if (message.author?.type === 'agent') return message.author?.avatar_url || ''
  // system / ai_assistant / macro 等自动回复 → 收件箱头像
  return widgetStore.config.avatar_url || widgetStore.config.launcher?.logo_url || ''
}

function getMessageAvatarFallback (message) {
  if (isUserMessage(message)) {
    const name = message.author?.first_name || userStore.firstName || 'V'
    return (name || 'V').charAt(0).toUpperCase()
  }
  if (message.author?.type === 'agent') {
    return (message.author?.first_name || 'A').charAt(0).toUpperCase()
  }
  return (widgetStore.config.brand_name || 'L').charAt(0).toUpperCase()
}
```

模板侧消息行结构调整为：

```
<div
  v-for="message in chatStore.getCurrentConversationMessages"
  :key="message.uuid"
  :class="[
    'flex gap-2',
    isUserMessage(message) ? 'flex-row-reverse' : 'flex-row'
  ]"
>
  <Avatar
    v-if="!isSpecialMessage(message)"
    class="size-8 flex-shrink-0 self-end"
  >
    <AvatarImage :src="getMessageAvatarUrl(message)" />
    <AvatarFallback>{{ getMessageAvatarFallback(message) }}</AvatarFallback>
  </Avatar>
  <div
    :class="['flex flex-col gap-1 min-w-0', isUserMessage(message) ? 'items-end' : 'items-start']"
  >
    <!-- 保留原 bubble / quote / attachment / meta 全部内容 -->
  </div>
</div>
```

## Design Style

延续 widget 现行的「清爽卡片式」风格，仅在消息行增加头像列，不引入新色板或动效。气泡颜色保持现状（用户 = primary，客服/system = muted），头像通过 `bg-muted`/`bg-secondary` 默认底色与气泡形成层级；与示例图保持一致：左列客服/system 头像（圆形），右列用户头像（圆形），头像底部与气泡底部对齐。

## Single Page Block Design Rules

仅改动 `ChatMessages.vue` 的消息列表区。整体滚动容器、间隔、padding、字体、颜色变量全部复用现有 token，无新增区块。