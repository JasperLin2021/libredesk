---
name: quick-reply-excel-import-export
overview: 为 /admin/quick-reply 页面的主题与问题模块实现 Excel 导入导出功能：前端用 xlsx 库生成/解析单 Sheet 表格并预览确认，后端新增批量导入接口以事务方式按主题名称做追加合并写入。
design:
  architecture:
    framework: vue
    component: shadcn
  styleKeywords:
    - 企业级后台
    - 简洁清晰
    - shadcn 风格
    - 轻量交互
    - 数据表格预览
  fontSystem:
    fontFamily: PingFang SC
    heading:
      size: 16px
      weight: 600
    subheading:
      size: 14px
      weight: 500
    body:
      size: 13px
      weight: 400
  colorSystem:
    primary:
      - "#2563EB"
      - "#1D4ED8"
    background:
      - "#FFFFFF"
      - "#F8FAFC"
    text:
      - "#0F172A"
      - "#64748B"
    functional:
      - "#EF4444"
      - "#10B981"
      - "#F59E0B"
todos:
  - id: backend-import-method
    content: 在 internal/quickreply/quickreply.go 为 Manager 增加 db 字段，实现 ImportTopics 事务方法（追加合并、返回统计）
    status: completed
  - id: backend-import-api
    content: 在 cmd/handlers.go 与 cmd/quickreply.go 新增批量导入路由与 handler，补充 i18n 导入文案
    status: completed
    dependencies:
      - backend-import-method
  - id: frontend-deps-api
    content: 在 frontend/package.json 新增 xlsx 依赖，并在 api/index.js 新增 importQuickReplyTopics 方法
    status: completed
  - id: frontend-export
    content: 在 TopicQuestionManager.vue 实现导出按钮与展平生成 xlsx 下载
    status: completed
    dependencies:
      - frontend-deps-api
  - id: frontend-import
    content: 在 TopicQuestionManager.vue 实现导入 Dialog（上传解析、归并预览、确认提交、toast 统计刷新）
    status: completed
    dependencies:
      - frontend-export
      - backend-import-api
  - id: verify-sample
    content: 使用 [skill:xlsx] 生成样例 Excel 模板验证导入解析逻辑与列结构兼容性
    status: completed
    dependencies:
      - frontend-import
---

## 产品概述

为后台管理页面 /admin/quick-reply 的主题与问题模块新增 Excel 导入导出能力，运营人员可通过编辑 Excel 表格批量维护快捷回复的主题、问题与答案数据，替代逐条手工录入。

## 核心功能

- **导出**：将当前收件箱的全部主题、问题、答案、别名、提示语导出为单个 Sheet 的 Excel 文件（列：主题名称 | 别名 | 提示语 | 问题 | 答案；同一主题的多个问题展开为多行，无问题的主题保留一行且问题列为空）。
- **导入**：上传 .xlsx/.xls 文件，前端解析并按"主题名称"归并多行为同一主题；导入前在页面预览解析出的主题/问题/答案，确认无误后再写入数据库。
- **追加合并策略**：按主题名称（含别名）匹配，已存在的主题更新其别名/提示语，并按问题文本更新或新增问题；不存在的主题整体新增；不删除任何现有数据。
- **字段规范**：与后端模型一致——主题（name、names 别名数组、hint_message、sort_order）、问题（question、answer、sort_order）；别名在 Excel 中用分号/逗号分隔。
- **结果反馈**：导入完成后提示统计结果（新增/更新主题数、新增/更新问题数），并刷新页面数据。

## 边界约定

- 导入仅作用于当前选中的收件箱，不跨收件箱。
- 导入保持现有名称唯一性约束（主名称与别名在收件箱内唯一）。
- 导入采用数据库事务，任一行失败则整体回滚，不产生半写入状态。

## 技术栈选择

- 后端：Go + fasthttp/fastglue + sqlx + goyesql（沿用项目现有架构，不引入新框架）
- 前端：Vue 3 + Vite + shadcn-vue 风格组件（@shared-ui/components/ui/*）+ lucide-vue-next 图标
- Excel 处理：前端新增 `xlsx`（SheetJS）依赖用于生成/解析 Excel；后端不引入 excelize，仅新增一个批量导入 JSON 接口
- i18n：前后端共用根目录 `i18n/*.json`（前端经 `getLanguage` API 加载），新增文案需同步 en-US 与 zh-CN 等语言文件

## 实现方案

### 总体策略

采用"前端处理 Excel、后端负责事务写入"的分工：

1. **导出**：前端复用现有 `GET /api/v1/inboxes/{id}/quick-reply-topics`（返回 topics 含 questions）获取完整数据，展平为单 Sheet 后用 SheetJS 生成 xlsx 并触发浏览器下载，无需新增后端接口。
2. **导入**：前端用 SheetJS 解析 Excel → 按主题名称归并 → 在 Dialog 中预览 → 确认后以结构化 JSON（topics 数组含 questions）提交新增的后端批量导入接口，由后端在数据库事务内按追加合并策略写入并返回统计。

**理由**：导入必须在导入前预览确认，前端解析天然契合；避免后端引入 excelize 依赖；导出数据源接口已存在，后端改动最小化。

### 后端设计

1. **Manager 事务支持**（`internal/quickreply/quickreply.go`）：

- `Manager` 结构体新增 `db *sqlx.DB` 字段，`New()` 中保存 `opts.DB`（参照 `internal/aiagent` 的 `m.db` 模式，第 266-293 行事务写法）。
- 新增 `ImportTopics(inboxID int, topics []ImportTopicItem) (ImportStats, error)`：
    - 事务模式：`tx, err := m.db.Beginx()` + `defer tx.Rollback()` + `tx.Commit()`。
    - 事务内用 `tx.Stmtx(m.q.GetTopicByName)` 按主名/别名匹配主题（现有 `names && ARRAY[$2::text]` 查询天然支持别名匹配）：
    - **已存在**：合并 names（沿用 `buildNames` 去重逻辑，主名称在前）、hint_message 非空则更新；逐条问题按 question 文本与 `tx.Stmtx(m.q.GetQuestions)` 拉取的现有问题匹配——命中则更新 answer，未命中则新增。
    - **不存在**：`InsertTopic` 新增主题，再逐条 `InsertQuestion` 新增问题。
    - 名称唯一性校验在导入时同样执行（复用 `TopicNameExists`/`validateNames` 逻辑，冲突返回 ConflictError）。
    - 返回统计 `ImportStats{CreatedTopics, UpdatedTopics, CreatedQuestions, UpdatedQuestions int}`。

2. **批量导入接口**（`cmd/handlers.go` + `cmd/quickreply.go`）：

- 新增路由 `POST /api/v1/inboxes/{id}/quick-reply-topics/import`，权限 `inboxes:manage`（加在 `cmd/handlers.go` 第 183-192 行 quick-reply 路由组之后）。
- 新增 `handleImportQuickReplyTopics`：解析 `{"topics":[{name, names, hint_message, questions:[{question, answer}]}]}`，逐项校验（name 非空且 ≤500、question 非空且 ≤500、answer ≤10000，复用 `maxQuickReplyFieldLength`、`buildNames`、`validateNames`），调用 `ImportTopics`，成功返回 `r.SendEnvelope(stats)`；错误走 `sendErrorEnvelope`。

3. **i18n 文案**（根目录 `i18n/en-US.json`、`zh-CN.json` 等 11 个语言文件）：新增导入相关错误提示与统计文案（至少 en-US + zh-CN）。

### 前端设计

1. **依赖与 API 封装**：

- `frontend/package.json` 新增 `xlsx` 依赖（pnpm@9.15.3 管理，注意与 lockfile 同步）。
- `frontend/apps/main/src/api/index.js` 新增 `importQuickReplyTopics(id, data)` → `POST /api/v1/inboxes/${id}/quick-reply-topics/import`（沿用第 420-433 行现有 headers 写法）。

2. **TopicQuestionManager.vue 导出**：

- CardHeader（第 4-10 行"新增主题"按钮旁）新增"导出"按钮（lucide `Download` 图标，`Button size="sm" variant="outline"`）。
- 逻辑：`getQuickReplyTopics(props.inboxId)` → 每个 topic 按 questions 逐行展平（无问题也输出一行，问题列留空）→ 列：主题名称、别名（分号分隔）、提示语、问题、答案 → `XLSX.utils.json_to_sheet` + `XLSX.writeFile` 生成 `quick-reply-{inboxId}.xlsx` 下载。

3. **TopicQuestionManager.vue 导入**：

- CardHeader 新增"导入"按钮（lucide `Upload` 图标），打开导入 Dialog。
- Dialog 内自上而下：文件选择区（accept `.xlsx,.xls`，选中后显示文件名）→ 解析统计条（"主题 N 个 / 问题 M 个"，错误行以 destructive 色提示行号）→ 预览表格（归并后的主题名称、别名、提示语、问题、答案，答案超长截断 hover 显示完整）→ DialogFooter（取消 / 确认导入，导入中显示 spinner 并禁用）。
- 解析逻辑：`XLSX.read` + `sheet_to_json` → 按主题名称归并（同名行合并、别名去重、hint_message 取非空值、问题按序收集）→ 跳过完全空行，主题名/问题为空的错误行高亮并阻止提交。
- 表头兼容中英文：主题名称/别名/提示语/问题/答案（topic name/aliases/hint message/question/answer）；别名支持 `;` `；` `,` `，` 分隔。
- 提交：确认后调 `importQuickReplyTopics` → 成功 toast 展示统计（沿用 `useEmitter` SHOW_TOAST + `handleHTTPError` 模式）并 `loadTopics()` 刷新列表。

### 性能与可靠性

- 数据量级为快捷回复配置（通常数百行内），前端解析与后端逐条事务写入无性能瓶颈；导入整体包在单个事务中，失败自动回滚，杜绝半写入状态。
- 后端校验与现有单条创建接口完全一致（长度、唯一性），保证导入数据与手工录入质量一致。
- 不删除任何现有数据，符合"追加合并"语义，降低误操作风险。

### 架构说明

```
[前端 TopicQuestionManager.vue]
  导出: getQuickReplyTopics ──► 展平 ──► SheetJS ──► .xlsx 下载
  导入: 上传 .xlsx ──► SheetJS 解析 ──► 归并/预览 ──► POST /import (JSON)
                                                       │
[后端] cmd/quickreply.go: handleImportQuickReplyTopics（校验）
        └──► internal/quickreply: ImportTopics（Beginx 事务内追加合并）
                └──► queries.sql 现有命名查询（GetTopicByName/InsertTopic/InsertQuestion/UpdateQuestion 等）
```

- 复用点：`GetTopicByName` 别名匹配、`buildNames`/`validateNames`、`maxQuickReplyFieldLength`、envelope 错误体系、`tx.Stmtx` 事务模式（参照 aiagent）、shadcn 组件与 toast 模式，均不引入新架构范式。

## 目录结构与文件清单

```
e:/libredesk/
├── cmd/
│   ├── handlers.go                        # [MODIFY] 新增 POST /api/v1/inboxes/{id}/quick-reply-topics/import 路由（权限 inboxes:manage）
│   └── quickreply.go                      # [MODIFY] 新增 handleImportQuickReplyTopics：请求解析、长度/唯一性校验、调用 ImportTopics、返回统计
├── internal/quickreply/
│   ├── quickreply.go                      # [MODIFY] Manager 新增 db 字段；新增 ImportTopicItem/ImportStats 类型与 ImportTopics 事务方法（追加合并）
│   └── queries.sql                        # [MODIFY] 如事务内查询不满足则补充命名查询（预期可完全复用现有查询，仅需确认）
├── i18n/
│   ├── en-US.json                         # [MODIFY] 新增导入导出相关文案（错误、统计提示）
│   └── zh-CN.json                         # [MODIFY] 新增导入导出相关文案（其余语言文件可后续补充）
└── frontend/
    ├── package.json                       # [MODIFY] 新增 xlsx（SheetJS）依赖
    └── apps/main/src/
        ├── api/index.js                   # [MODIFY] 新增 importQuickReplyTopics(id, data)
        └── features/admin/quickreply/
            └── TopicQuestionManager.vue   # [MODIFY] CardHeader 新增"导入/导出"按钮；导出展平生成 xlsx；导入 Dialog（上传→解析归并→预览确认→提交→toast 统计并刷新）
```

## 设计说明

在现有 shadcn-vue 风格基础上做增量 UI 设计，保持与当前 quick-reply 页面视觉完全一致，不改变既有布局结构。

- **入口**：CardHeader 右侧"新增主题"按钮旁并列放置"导入"（Upload 图标）与"导出"（Download 图标）按钮，均采用 `Button size="sm" variant="outline"`，hover 时边框与底色轻微加深，保持工具栏视觉节奏一致。
- **导入 Dialog**：复用现有 Dialog 组件（参照删除确认 Dialog 写法，lucide 图标 + Card/Table 样式）。内容自上而下四段：

1. 文件选择区：虚线边框可点击拖放区（accept .xlsx/.xls），选中后显示文件名与解析状态；
2. 解析统计条：浅色背景卡片展示"主题 N 个 / 问题 M 个"，若有错误行以 destructive 色提示行号；
3. 预览表格：紧凑表格展示归并后的主题名称、别名、提示语、问题、答案（答案超长截断，hover 显示完整），无数据时显示空态文案；窄屏横向滚动；
4. 底部 DialogFooter：取消 / "确认导入"（primary 按钮，导入中显示 spinner 并禁用）。

- **交互反馈**：导入成功 toast 展示"新增主题 X / 更新主题 Y / 新增问题 M / 更新问题 N"；失败 toast 展示后端错误信息（沿用 handleHTTPError 模式）。
- **响应式**：预览表格窄屏横向滚动（overflow-x-auto），文件选择区与按钮在移动端纵向堆叠。

## Agent Extensions

### Skill

- **xlsx**
- 用途：在导入导出功能实现后，生成一份与前端 SheetJS 导出一致的样例 .xlsx 模板，验证列结构（主题名称/别名/提示语/问题/答案）、中英文表头兼容性与中文单元格编码，确保导入解析逻辑与真实 Excel 文件格式一致。
- 预期产出：一份可被导入逻辑正确解析的样例 Excel 文件及验证结论，用于手工回归测试。