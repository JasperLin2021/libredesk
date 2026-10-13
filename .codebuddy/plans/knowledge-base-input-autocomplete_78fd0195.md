---
name: knowledge-base-input-autocomplete
overview: 在回复编辑器的输入框中实现知识库关键词自动补全：用户输入关键词时自动弹出匹配的知识库文章，选择后可直接发送正文。
todos:
  - id: create-kb-suggestion-list
    content: 创建 KnowledgeSuggestionList.vue 知识库下拉列表组件
    status: completed
  - id: integrate-editor-autocomplete
    content: 在 TextEditor.vue 中实现关键词监听、防抖搜索与下拉定位
    status: completed
    dependencies:
      - create-kb-suggestion-list
  - id: wire-replybox-content
    content: 在 ReplyBoxContent.vue 中启用知识库自动补全并限制仅回复模式
    status: completed
    dependencies:
      - integrate-editor-autocomplete
  - id: verify-lint-tests
    content: 运行 lint 与单元测试，验证无回归并修复告警
    status: completed
    dependencies:
      - wire-replybox-content
---

## 产品概述

在主应用会话回复编辑器的输入框中，增加“知识库关键词自动补全”功能。客服人员在回复编辑区输入关键词时，系统自动从已发布的知识库文章中搜索匹配项，并以悬浮下拉列表展示文章标题与摘要；用户可通过鼠标点击或键盘上下键选择文章，确认后直接把文章完整正文作为回复内容发送。

## 核心功能

- 关键词触发：在回复编辑器（TipTap 富文本编辑器）中输入字符，当光标所在单词长度 ≥ 2 时，自动触发知识库搜索。
- 下拉展示：展示最多 5-8 条匹配知识库文章，显示标题与内容摘要，匹配项按发布时间或相关度排序。
- 键盘操作：支持 ↑/↓ 切换选中项、Enter 确认选择、Esc 关闭列表。
- 选择发送：确认选择后，从后端拉取完整正文，并复用现有 `KB_SEND_CONTENT` 事件，由 `ReplyBox.vue` 直接发送消息。
- 模式适配：仅在“回复”模式下启用；私密备注模式不触发自动补全，避免误发送。
- 防抖与性能：输入后 300ms 防抖，取消上一次未完成的请求，避免频繁请求后端。

## 技术栈

- 前端框架：Vue 3 + Composition API
- 编辑器：TipTap 2（@tiptap/vue-3、@tiptap/suggestion）
- 构建工具：Vite
- 样式：Tailwind CSS + 项目内 shadcn/ui 组件
- 状态与通信：Pinia + mitt（event emitter）
- API：项目内已封装的 Axios 模块（`frontend/apps/main/src/api/index.js`）

## 实现方案

### 总体策略

不新增复杂的 ProseMirror 扩展，而是在 `TextEditor.vue` 内部监听 `onUpdate` 与 `selectionUpdate` 事件，提取光标前当前单词，经防抖后调用已有 API `listAgentKbArticles` 搜索；通过 `editor.view.coordsAtPos` 计算光标坐标，使用一个独立的 Vue 悬浮层展示下拉列表。选中文章后，调用 `getAgentKbArticle` 获取完整正文，并通过 `emitter.emit(EMITTER_EVENTS.KB_SEND_CONTENT, content)` 交给 `ReplyBox.vue` 复用现有发送管线。

### 关键决策

- **不新建 TipTap 扩展**：避免与现有 Mention 扩展在触发字符、键盘事件上产生冲突，同时保持实现轻量、可维护。
- **复用现有发送事件**：`KB_SEND_CONTENT` 已在 `ReplyBox.vue` 中处理，包含 `htmlContent/textContent` 同步、发送、失败回滚等逻辑，复用可减少重复代码与状态不一致风险。
- **仅在回复模式启用**：通过 `messageType` prop 控制，私密备注模式下关闭自动补全，符合业务场景。

### 性能与可靠性

- 搜索使用 300ms 防抖，并复用 axios 的 AbortController 取消上一次未完成的请求。
- 下拉列表最多展示 5-8 条，摘要最多 2 行，避免 DOM 与网络开销。
- 仅当编辑器获得焦点且光标位于文本末尾或单词中间时触发，避免全局监听。
- 错误处理：搜索失败时静默关闭下拉框，不阻断用户输入。

### 架构设计

```
ReplyBox.vue (监听 KB_SEND_CONTENT 并发送)
    │
    ▼
ReplyBoxContent.vue (启用 kbSuggestions，传递 messageType)
    │
    ▼
TextEditor.vue (检测当前单词、搜索、渲染下拉框)
    ├── 内部状态：suggestionQuery / articles / selectedIndex / popupPos
    ├── 防抖搜索：api.listAgentKbArticles
    └── 选中回调：getAgentKbArticle → emitter KB_SEND_CONTENT
        │
        ▼
KnowledgeSuggestionList.vue (下拉 UI，键盘导航)
```

## 目录结构

```
frontend/apps/main/src/
├── components/
│   └── editor/
│       ├── TextEditor.vue              # [MODIFY] 集成知识库自动补全触发、搜索、定位与选中逻辑
│       └── KnowledgeSuggestionList.vue # [NEW]  知识库文章下拉列表 UI，支持标题/摘要展示与键盘导航
├── features/
│   └── conversation/
│       ├── ReplyBoxContent.vue         # [MODIFY] 向编辑器启用知识库建议并透传 messageType
│       └── sidebar/
│           └── KnowledgePanel.vue      # [REFERENCE] 现有发送逻辑参考，不修改
├── api/
│   └── index.js                        # [REFERENCE] 已提供 listAgentKbArticles / getAgentKbArticle
└── constants/
    └── emitterEvents.js                # [REFERENCE] 已定义 KB_SEND_CONTENT
```

## 执行注意事项

- 光标坐标获取：使用 `editor.view.coordsAtPos(editor.state.selection.from)`，并在滚动或窗口 resize 时重新定位。
- 单词提取：基于 `editor.state.doc.textBetween(0, selection.from)` 与正则 `/[\p{L}\p{N}_]+$/u` 截取光标前最后一个连续单词。
- 与 Mention 下拉互斥：当检测到 `@` 触发的 Mention 下拉处于活动状态时，不显示知识库下拉。
- 私密备注模式：通过 `props.messageType !== 'reply'` 直接禁用监听。
- 无障碍：下拉列表使用 `role="listbox"`，选项使用 `role="option"` 与 `aria-selected`，并确保焦点管理不跳出编辑器。