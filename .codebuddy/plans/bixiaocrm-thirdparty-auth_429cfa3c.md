---
name: bixiaocrm-thirdparty-auth
overview: 前端页面简化后，新增后端路由对接必小秘第三方API进行用户认证，认证成功则创建联系人并进入聊天。
todos:
  - id: create-bixiao-handler
    content: 在 cmd/chat_bixiao.go 中创建 handleBixiaoAuth handler：验证 inbox、调用 bixiaocrm API、提取 nickname/tlId、创建联系人、保存 custom_attributes、生成 session token
    status: completed
  - id: register-route
    content: 在 cmd/handlers.go 中注册新路由 POST /api/v1/widget/chat/auth/bixiao，使用 validateWidgetInbox 中间件
    status: completed
    dependencies:
      - create-bixiao-handler
  - id: modify-usewidget
    content: 修改 widget-frontend/src/composables/useWidget.js：替换 verifyToken 为 verifyBixiaoAuth，接收 token/device/versionSeq/inbox 四个参数，调用新 endpoint
    status: completed
  - id: modify-app-vue
    content: 修改 widget-frontend/src/App.vue：将 JWT 单字段模态框改为 token/device/versionSeq/inbox 四字段模态框，inbox 预填配置值
    status: completed
    dependencies:
      - modify-usewidget
  - id: build-verify
    content: 构建后端和前端，验证编译通过
    status: completed
    dependencies:
      - register-route
      - modify-app-vue
---

## 用户需求

用户点击"我的客服"按钮后，弹出模态框填入 **token**、**device**、**versionSeq**、**inbox** 四个字段，提交到后端新路由。后端将 token/device/versionSeq 作为 HTTP header 转发到第三方接口 `https://api-app.bixiaocrm.com/personnel/index.json` 进行验证。

验证失败时返回"非法用户"错误提示；验证成功时从返回数据中提取 `nickname` 作为联系人名字（first_name），提取 `tlId` 作为 `external_user_id` 存入 `custom_attributes` JSONB（key: `tl_id`），创建或更新联系人，生成 session token，前端收到后打开聊天窗口。

## 核心功能

1. **前端四字段模态框**：替换现有 JWT 单字段输入，改为 token、device、versionSeq、inbox 四个字段，inbox 预填已保存配置值
2. **后端新路由**：`POST /api/v1/widget/chat/auth/bixiao`，使用 `validateWidgetInbox` 中间件验证 inbox
3. **第三方 API 调用**：后端以 HTTP POST 转发到 bixiaocrm，header 附带 token/device/versionSeq
4. **联系人自动创建**：以 tlId 为 `external_user_id` 通过 UPSERT 创建联系人，nickname 作为 first_name
5. **Session 生成**：复用现有 `generateSessionToken()`，返回 session_token 和用户信息

## 技术栈

- **后端**: Go + fastglue（fasthttp）+ Redis + PostgreSQL
- **前端**: Vue 3 + Tailwind CSS + lucide-vue-next（现有 widget-frontend）
- **外部调用**: `net/http` 标准库（参考 `cmd/oauth.go` 模式）

## 实现方案

### 整体架构

```
用户点击"我的客服" → 弹出4字段模态框
  → POST /api/v1/widget/chat/auth/bixiao (X-Libredesk-Inbox-ID header)
  → validateWidgetInbox 中间件验证 inbox
  → handleBixiaoAuth handler:
      1) POST https://api-app.bixiaocrm.com/personnel/index.json (headers: token, device, versionSeq)
      2) 解析响应 → 提取 nickname / tlId
      3) app.user.CreateContact({ ExternalUserID: tlId, FirstName: nickname })
      4) app.user.SaveCustomAttributes(contactID, { tl_id: tlId })
      5) generateSessionToken() → Redis
      6) 返回 { session_token, user }
  → 前端收到 session_token → setUser() → show() → 聊天窗口打开
```

### 关键设计决策

1. **tlId 双重作用**：既是 `external_user_id`（用于联系人 UPSERT 去重），也存入 `custom_attributes.tl_id`（供后台查询）
2. **完全新的 handler**：不复用 `handleAuthExchange`，因为其核心逻辑是验证本地签发的 HMAC JWT，而此处需转发到第三方 API
3. **复用现有基础设施**：`validateWidgetInbox` 中间件、`CreateContact()` UPSERT 逻辑、`generateSessionToken()`、`getSessionDuration()`、`SaveCustomAttributes()` 全部直接复用
4. **无需数据库 migration**：`custom_attributes` 是 JSONB 字段，新增 `tl_id` key 无需 DDL

## 实现细节

### 后端：`cmd/chat_bixiao.go`（新建）

Handler 结构：

```
func handleBixiaoAuth(r *fastglue.Request) error {
    // 1. 从中间件上下文获取 inbox + config
    // 2. 解析请求体 { token, device, versionSeq, inbox }
    // 3. HTTP POST 到 bixiaocrm API，header 带 token/device/versionSeq
    // 4. 解析响应：成功则提取 nickname + tlId，失败则返回错误
    // 5. 用 tlId 创建/更新联系人（external_user_id）
    // 6. SaveCustomAttributes({ tl_id: tlId })
    // 7. generateSessionToken() 生成 token 存 Redis
    // 8. 返回 { session_token, user: { user_id, first_name, ... } }
}
```

- HTTP 客户端使用 `&http.Client{Timeout: 10 * time.Second}`（参考 `cmd/oauth.go`）
- 第三方 API 响应结构：`{ nickname: string, tlId: string }`（只提取这两个字段，其他字段忽略）
- 联系人创建：`umodels.User{ FirstName: nickname, ExternalUserID: tlId, CustomAttributes: {"tl_id": tlId} }` → `app.user.CreateContact(&user)`
- Session 生成后写入 Redis 反向索引 key：`widget_user:{inboxID}:{contactID}` → token（复用 `handleAuthExchange` 的模式）

### 后端：`cmd/handlers.go`（修改）

在 widget APIs 区域（第 323 行附近）新增一行路由：

```
g.POST("/api/v1/widget/chat/auth/bixiao", rateLimit(validateWidgetInbox(handleBixiaoAuth), "widget"))
```

### 前端：`src/composables/useWidget.js`（修改）

- 将 `verifyToken(jwt)` 替换为 `verifyBixiaoAuth({ token, device, versionSeq, inbox })`
- 请求体从 `{ jwt }` 改为 `{ token, device, versionSeq, inbox }`
- API endpoint 从 `/api/v1/widget/chat/auth/exchange` 改为 `/api/v1/widget/chat/auth/bixiao`
- 验证通过后仍需调用 `window.Libredesk.show()` 打开聊天窗口

### 前端：`src/App.vue`（修改）

- 模态框从单个 textarea 改为 4 个 input 字段：
- **token**（text input，必填）
- **device**（text input，必填）
- **versionSeq**（text input，必填）
- **inbox**（text input，预填 config.inboxID，可修改）
- `handleVerify()` 改为调用 `verifyBixiaoAuth()` 传 4 个参数
- 连接状态时按钮文案显示"继续咨询"，认证后按钮文案也正确处理

## 文件变更清单

```
e:\libredesk\
├── cmd/
│   ├── chat_bixiao.go          # [NEW] handleBixiaoAuth handler
│   └── handlers.go             # [MODIFY] 注册新路由
└── widget-frontend/
    └── src/
        ├── App.vue             # [MODIFY] 四字段模态框替换单字段
        └── composables/
            └── useWidget.js    # [MODIFY] verifyBixiaoAuth 替换 verifyToken
```