---
name: 对话侧栏新增知识库Tab
overview: 在客服对话右侧栏"常用语"tab 右侧新增"知识库"tab（共4个tab），列表数据来自客服知识库（新增 agent 侧只读接口），支持按标题/内容搜索（后端 keyword_field），点击"发送正文"直接把知识正文作为客服消息发送给用户，底部提供"显示更多"分页加载按钮。
design:
  architecture:
    framework: vue
    component: shadcn
  styleKeywords:
    - 极简紧凑
    - shadcn 设计语言
    - 与现有侧栏一致
  fontSystem:
    fontFamily: PingFang-SC
    heading:
      size: 12px
      weight: 500
    subheading:
      size: 12px
      weight: 400
    body:
      size: 12px
      weight: 400
  colorSystem:
    primary:
      - "#0F172A"
    background:
      - "#FFFFFF"
      - "#F4F4F5"
    text:
      - "#09090B"
      - "#71717A"
    functional:
      - "#2563EB"
      - "#DC2626"
todos:
  - id: backend-agent-kb-api
    content: 新增 cmd/agent_kb.go handler 并在 cmd/handlers.go 注册 agent 侧只读路由（列表强制 status=published，详情接口，权限 messages:read）
    status: completed
  - id: frontend-api-i18n
    content: 在 api/index.js 新增 listAgentKbArticles/getAgentKbArticle，emitterEvents.js 新增 KB_SEND_CONTENT，三语言 i18n 文件新增 conversation.sidebar.knowledgeBase.* key
    status: completed
    dependencies:
      - backend-agent-kb-api
  - id: knowledge-panel
    content: 新建 sidebar/KnowledgePanel.vue：搜索范围下拉+关键词防抖后端搜索、条目列表（标题+摘要+展开正文）、发送正文按钮、底部显示更多分页
    status: completed
    dependencies:
      - frontend-api-i18n
  - id: sidebar-tab-integration
    content: 修改 ConversationSideBar.vue：TabsList 改 grid-cols-4，新增"知识库" TabsTrigger 与 TabsContent 挂载 KnowledgePanel
    status: completed
    dependencies:
      - knowledge-panel
  - id: send-content-wiring
    content: 修改 ReplyBox.vue 监听 KB_SEND_CONTENT：填充正文（private_note 切 reply）后调用 processSend() 完成直接发送
    status: completed
    dependencies:
      - knowledge-panel
  - id: verify-build
    content: 验证：go build、前端构建与功能自测（搜索/分页/发送正文/草稿不泄露）
    status: completed
    dependencies:
      - sidebar-tab-integration
      - send-content-wiring
---

## Product Overview

在客服对话工作台右侧边栏中，"常用语”tab 右侧新增一个“知识库”tab（侧栏由 3 个 tab 变为 4 个）。该面板让客服在处理对话时可直接查阅并复用管理后台“客服知识库”中的知识条目。

## Core Features

- **知识库 Tab**：位于“常用语”右侧，所有具备对话处理权限的客服可见（不受“知识库管理”权限限制）
- **列表展示**：数据来源于客服知识库（仅已发布状态），每条显示标题、内容摘要（去 HTML 标签截断），点击可展开查看完整正文
- **搜索**：搜索区包含范围下拉（全部 / 标题 / 内容）+ 关键词输入框，为后端搜索（防抖触发），与截图设计一致
- **发送正文**：条目上的“发送正文”按钮，点击后获取完整正文并立即作为一条客服消息直接发送给用户（不经过回复框，复用现有发送管线以保留乐观更新、警告确认与错误回滚）
- **显示更多**：列表底部有“显示更多”按钮，分页加载更多条目；已全部加载时隐藏
- **本期不做**：发送附件功能暂不实现

## Tech Stack

- 后端：Go (fastglue + sqlx)，复用 `internal/knowledgebase` Manager
- 前端：Vue 3 (Composition API) + Pinia + `@shared-ui` 组件（shadcn 风格）+ vue-i18n
- 不引入新依赖，完全遵循项目现有模式

## Implementation Approach

1. **后端新增 agent 侧只读接口**（不复用受 `knowledge_base:manage` 保护的管理端 `/api/v1/kb`）：

- `GET /api/v1/agent/kb/articles`（权限 `messages:read`）：复用 `parseKnowledgeArticleFilter` 解析 `keyword_field`（all/title/content）、`keywords`、`page`、`per_page`，强制 `status=published`，调用 `Manager.List`
- `GET /api/v1/agent/kb/articles/{id}`（权限 `messages:read`）：复用 `Manager.GetArticle`，供"发送正文"取完整 content（列表仅返回 summary）
- handler 放新文件 `cmd/agent_kb.go`（参照 `cmd/agent_canned.go` 惯例），路由注册到 `cmd/handlers.go` agent 路由区

2. **前端新增面板**：`KnowledgePanel.vue`（sidebar 目录），结构与样式参照 `CannedResponsesPanel.vue`（顶部搜索区、滚动列表区）。搜索为后端分页搜索：范围下拉 + 关键词输入 300ms 防抖后重置 page=1 请求；"显示更多"请求下一页追加；`total > 已加载数` 时显示按钮
3. **发送正文**：新增 emitter 事件 `KB_SEND_CONTENT`。面板点击按钮 → 调 agent 详情接口取完整 content → emit 事件；`ReplyBox.vue` 监听：若当前是 private_note 模式切换为 reply，设置 `htmlContent` 后立即调用 `processSend()`。完整复用现有发送管线（pending message 乐观显示、email/tags 警告对话框、错误回滚、macro/draft 清理），避免重复实现发送逻辑
4. **Tab 接入**：`ConversationSideBar.vue` 的 TabsList 由 `grid-cols-3` 改 `grid-cols-4`，新增 value="knowledge" 的 TabsTrigger 与 TabsContent，label 走 i18n
5. **i18n**：`i18n/zh-CN.json`、`en-US.json`、`da-DK.json` 同步新增 `conversation.sidebar.knowledgeBase.*` 系列 key（标题、搜索占位符、全部/标题/内容、发送正文、显示更多、加载中、空状态、发送成功/失败等）

### Performance

- 列表分页（默认 20 条/页），仅传 summary，不加载全量 content；关键词过滤在 SQL 层（`ilike` + `unnest`，与现有管理端一致）
- 搜索防抖 300ms；翻页仅追加渲染；无 N+1（列表查询一次 + count 一次）
- 面板数据在 tab 激活/挂载时才拉取，不预热

### 关键执行细节

- agent 列表接口必须强制覆盖 `status=published`，忽略客户端传入的 status，防止草稿泄露
- `processSend` 为 async，监听器中调用需处理发送中重复点击（面板按钮置 loading 状态）
- 事件监听需在 `onMounted`/`onUnmounted` 成对注册/注销（参照 COPILOT_INSERT_REPLY 写法）
- v-html 渲染知识正文前不做额外净化（与管理端预览一致，数据源为内部录入）

## Directory Structure

```
libredesk/
├── cmd/
│   ├── agent_kb.go          # [NEW] agent 侧知识库只读 handler：handleGetAgentKbArticles（复用 parseKnowledgeArticleFilter，强制 status=published）、handleGetAgentKbArticle（复用 Manager.GetArticle）
│   └── handlers.go          # [MODIFY] agent 路由区新增 GET /api/v1/agent/kb/articles、GET /api/v1/agent/kb/articles/{id}（perm(..., "messages:read")）
├── frontend/apps/main/src/
│   ├── features/conversation/sidebar/
│   │   ├── KnowledgePanel.vue        # [NEW] 知识库面板：顶部搜索区（范围 Select + Input + Search 图标）、列表（标题+摘要+发送正文按钮，点击展开完整正文 v-html）、底部"显示更多"按钮、空/加载状态
│   │   └── ConversationSideBar.vue   # [MODIFY] TabsList 改 grid-cols-4，新增 knowledge TabsTrigger/TabsContent 引入 KnowledgePanel
│   ├── features/conversation/
│   │   └── ReplyBox.vue              # [MODIFY] 新增 KB_SEND_CONTENT 监听：设置 htmlContent（private_note 切 reply）后调用 processSend()
│   ├── api/index.js                  # [MODIFY] 新增 listAgentKbArticles(params)、getAgentKbArticle(id) 并加入导出清单
│   └── constants/emitterEvents.js    # [MODIFY] 新增 KB_SEND_CONTENT 事件常量
└── i18n/
    ├── zh-CN.json            # [MODIFY] 新增 conversation.sidebar.knowledgeBase.* key
    ├── en-US.json            # [MODIFY] 同步新增
    └── da-DK.json            # [MODIFY] 同步新增
```

## 设计说明

新面板完全遵循现有对话侧栏设计语言（shadcn 风格、text-xs 紧凑布局），与"常用语"tab 视觉一致：

- **顶部搜索区**：border-b 分隔，左侧为范围下拉（全部/标题/内容，小号 Select），右侧为带 Search 图标的关键词输入框（h-8 text-xs，placeholder 为"搜索知识库”）
- **条目列表**：垂直滚动区，每条目为卡片式行（hover:bg-accent），左侧标题（font-medium text-xs）+ 摘要两行截断（text-xs text-muted-foreground），右侧或条目底部为“发送正文”小按钮（outline variant, h-6 text-xs）；点击条目展开完整正文（v-html 渲染富文本，缩进区域区分）
- **底部**：居中“显示更多”ghost 按钮（加载中显示 Loader2 旋转），全部加载后隐藏；搜索无结果显示空状态提示文案
- **Tab 头**：4 等分 grid，保持现有 text-xs 触发器样式