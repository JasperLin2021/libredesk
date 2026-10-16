---
name: livechat-handoff-settings-tab
overview: 在 Livechat 收件箱表单"安装"选项卡右侧新增"转人工设置"选项卡：包含"选择时间段"下拉（数据来自工作时间列表，默认"不限制"）和"只允许上班时间/只允许下班时间"单选项；后端 Inbox 新增 handoff_business_hours_id、handoff_hours_mode 字段并持久化（含 migration），保存后可回显。
design:
  architecture:
    framework: vue
    component: shadcn
  styleKeywords:
    - 简洁表单
    - 与现有后台一致
  fontSystem:
    fontFamily: PingFang SC
    heading:
      size: 14px
      weight: 600
    subheading:
      size: 13px
      weight: 500
    body:
      size: 13px
      weight: 400
  colorSystem:
    primary:
      - "#2563EB"
    background:
      - "#FFFFFF"
      - "#F8FAFC"
    text:
      - "#0F172A"
      - "#64748B"
    functional:
      - "#3B82F6"
      - "#94A3B8"
todos:
  - id: backend-migration
    content: 新增 migration v2.17.0：inboxes 表加 handoff_business_hours_id、handoff_hours_mode 列，并注册到 cmd/upgrade.go
    status: completed
  - id: backend-api
    content: 后端透传：models.go 新增字段、cmd/inboxes.go 校验、internal/inbox/inbox.go Create/Update/GetAll SQL 补充新字段
    status: completed
  - id: frontend-schema
    content: livechatFormSchema.js 新增 handoff 字段校验与默认值，补充 tabs 等相关 i18n 文案
    status: completed
  - id: frontend-tab-ui
    content: LivechatInboxForm.vue 在“安装”右侧新增“转人工设置”选项卡：下拉（getAllBusinessHours，含“不限制”）+ 上班/下班 RadioGroup，按截图布局并联动置灰
    status: completed
  - id: frontend-submit-echo
    content: 完成 FIELD_TAB 映射、onSubmit 清洗（未选时间段置 null）与初始值 watch 回显，验证 EditInbox/NewInbox payload 透传
    status: completed
  - id: build-verify
    content: 运行前端 lint/build 与 Go 编译，验证保存-回显链路
    status: completed
---

## 产品概述

在 Livechat 收件箱编辑表单的顶部选项卡中，于“安装”右侧新增“转人工设置”选项卡，用于配置转人工生效的时间段。

## 核心功能

- **转人工时间段**：内容区标题，下方提供“选择时间段”下拉框，选项来自系统“工作时间”（business hours）列表，默认含“不限制”空选项；未选择时转人工时间不生效
- **时间模式单选**：下拉框下方显示 ⓘ 提示 + 两个互斥单选项（圆形单选钮样式）
- “只允许上班时间”：按所选时间段内的上班时段判断
- “只允许下班时间”：对所选时间段取反（即仅下班时段允许）
- **持久化与回显**：配置保存到数据库（Inbox 新增关联字段与模式字段），编辑时回显；未选择时间段时模式单选项置灰或仅提示不生效
- **范围**：仅 Livechat 渠道表单；本计划只覆盖设置界面与持久化回显，不含转人工触发时的实际时间拦截逻辑

## 技术栈

- 后端：Go（现有 internal/inbox 模块、internal/migrations Go 版迁移机制、cmd/ HTTP handler）
- 前端：Vue 3 + vee-validate + zod（现有 LivechatInboxForm 模式），shadcn 风格组件（Tabs / Select / RadioGroup / Label / Info 提示）

## 实现方案

### 数据库

- 新增 migration（internal/migrations/v2.17.0.go，导出 V2_17_0(db, fs, ko)），为 inboxes 表新增：
- handoff_business_hours_id INTEGER NULL REFERENCES business_hours(id)（NULL=不限制）
- handoff_hours_mode TEXT NOT NULL DEFAULT 'business'（枚举：business=只允许上班时间，off_business=只允许下班时间）
- 在 cmd/upgrade.go migrations 切片中注册 v2.17.0

### 后端透传

- internal/inbox/models/models.go：Inbox struct 新增 HandoffBusinessHoursID *int64、HandoffHoursMode string
- cmd/inboxes.go：validateInbox 放行/校验新字段（business_hours 外键存在性校验，允许空）
- internal/inbox/inbox.go：Create/Update/GetAll 的 SQL 与扫描补充新字段

### 前端

- livechatFormSchema.js：新增 handoff_business_hours_id（可空）、handoff_hours_mode（默认 'business'）校验
- LivechatInboxForm.vue：
- Tabs 在 installation 后新增 handoff Tab（TabsTrigger + v-show 内容区），按截图布局
- 复用 api.getAllBusinessHours() 拉取工作时间列表（参照 TeamForm.vue 第 214 行模式），下拉含“不限制”空选项
- 下拉变更联动：选择时间段后启用单选项；未选择时提示“不限制”
- FIELD_TAB 注册新字段 → 新 Tab 映射；onSubmit 清洗（未选时间段时 handoff_business_hours_id 置 null）；初始值 watch 回填
- EditInbox.vue / NewInbox.vue：payload 自然透传新字段（无需特殊处理，验证字段未被过滤）

## 架构

```mermaid
flowchart LR
  A[LivechatInboxForm 新 Tab] -->|vee-validate + zod| B[EditInbox/NewInbox payload]
  B -->|PUT/POST /api/v1/inboxes| C[cmd/inboxes.go]
  C --> D[internal/inbox Manager]
  D --> E[(inboxes 新列)]
  F[GET /api/v1/business-hours] --> A
```

## 注意事项

- 复用现有 Tabs/v-show、FIELD_TAB 跳转、回显 watch 模式，不引入新状态管理
- migration 需同时注册到 upgrade.go，否则不会执行；新列必须可空以兼容存量数据
- 不改动无关渠道（Email 表单）与转人工触发逻辑

## 设计风格

严格遵循现有管理后台表单风格，按截图实现：内容区为浅色卡片区块，标题“转人工时间段”；其下一行为左右布局——左侧标签“选择时间段”，右侧 Select 下拉（占位/选中显示工作时间名称，含“不限制”选项）；再下方一行：ⓘ 蓝色信息图标 + 提示文案 + 两个圆形单选钮（RadioGroup）：“只允许上班时间”“只允许下班时间”。顶部 Tabs 沿用现有组件，新 Tab 位于“安装”右侧，激活态样式与现有 Tab 一致。