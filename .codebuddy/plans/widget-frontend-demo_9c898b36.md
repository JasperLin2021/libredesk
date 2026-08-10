---
name: widget-frontend-demo
overview: 在 e:\libredesk\widget-frontend 创建一个独立的 Vue 3 + Vite 示例网页，演示如何将 LibreDesk 在线客服 widget 嵌入到外部网站中。
design:
  architecture:
    framework: vue
  styleKeywords:
    - 现代简约
    - 科技感
    - 渐变背景
    - 玻璃拟态
    - 卡片布局
  fontSystem:
    fontFamily: Inter
    heading:
      size: 48px
      weight: 700
    subheading:
      size: 20px
      weight: 500
    body:
      size: 16px
      weight: 400
  colorSystem:
    primary:
      - "#4F46E5"
      - "#7C3AED"
      - "#6366F1"
    background:
      - "#FFFFFF"
      - "#F8FAFC"
      - "#0F172A"
    text:
      - "#0F172A"
      - "#475569"
      - "#F1F5F9"
    functional:
      - "#22C55E"
      - "#EF4444"
      - "#F59E0B"
todos:
  - id: init-vue-project
    content: 使用 Vite 初始化 Vue 3 项目，创建 package.json、vite.config.js、index.html、main.js
    status: completed
  - id: create-index-html
    content: 编写 index.html，预留 widget.js 动态加载占位，设置页面 meta 和全局样式
    status: completed
    dependencies:
      - init-vue-project
  - id: create-app-root
    content: 创建 App.vue 根组件，管理配置状态（localStorage）、widget 生命周期、配置面板与演示页面的条件渲染切换
    status: completed
    dependencies:
      - init-vue-project
  - id: create-use-widget
    content: 创建 src/composables/useWidget.js，封装动态加载 widget.js、initLibredesk 初始化、API 代理、状态监听、销毁清理逻辑
    status: completed
    dependencies:
      - init-vue-project
  - id: create-config-panel
    content: 创建 ConfigPanel.vue 配置面板组件，包含 baseURL 和 inboxID 输入框、表单验证、保存到 localStorage、美观的玻璃拟态弹窗样式
    status: completed
    dependencies:
      - create-app-root
  - id: create-demo-page
    content: 创建 DemoPage.vue 主演示页面，组装 Navbar、HeroSection、FeatureCards、Footer、WidgetControls 组件
    status: completed
    dependencies:
      - create-app-root
  - id: create-demo-sections
    content: 创建 Navbar.vue、HeroSection.vue、FeatureCards.vue、Footer.vue 四个页面区块组件，实现仿真公司官网视觉
    status: completed
    dependencies:
      - create-demo-page
  - id: create-widget-controls
    content: 创建 WidgetControls.vue，展示 widget 状态指示器、API 控制按钮组（show/hide/setUser/logout/destroy）、未读消息数实时显示
    status: completed
    dependencies:
      - create-use-widget
      - create-demo-page
---

## 产品概述

一个独立的 Vue 3 演示网页，模拟科技公司官网，并在右下角嵌入 LibreDesk 在线客服 widget。用户可以通过配置面板输入后端地址和 inbox ID，即时体验在线客服功能。同时展示 widget JS API 的调用示例。

## 核心功能

- **仿真公司官网**：包含导航栏、Hero 区域、功能卡片展示、页脚，模拟真实网站场景
- **在线客服 Widget 嵌入**：通过加载 LibreDesk 后端的 widget.js 脚本，在右下角显示浮动聊天按钮
- **配置面板**：提供 baseURL 和 inboxID 输入框，支持本地持久化存储（localStorage），方便开发和演示
- **Widget API 演示**：提供按钮调用 widget 的 show()、hide()、setUser()、logout()、destroy() 等 API，直观展示控制能力
- **状态指示器**：实时显示 widget 连接状态、未读消息数、可见性状态

## 技术栈

- **前端框架**: Vue 3 (Composition API) + Vite
- **样式方案**: CSS Scoped + CSS Variables
- **额外依赖**: 无需，纯 Vue 3 原生能力
- **持久化**: localStorage 存储配置

## 实现方案

### 整体策略

创建一个独立的 Vite + Vue 3 项目，完全不依赖 LibreDesk 的 monorepo 结构。采用单一页面架构，通过条件渲染切换"配置面板"和"演示页面"两个视图。

### Widget 嵌入机制

LibreDesk 的 widget.js 是一个自执行的 IIFE 脚本，通过以下方式嵌入：

**自动初始化（推荐）**：

```html
<script src="http://localhost:9000/widget.js"></script>
<script>
  window.LibredeskSettings = {
    baseURL: 'http://localhost:9000',
    inboxID: 'xxx-uuid-xxx'
  };
</script>
```

**手动初始化 + API 控制**：

```html
<script src="http://localhost:9000/widget.js"></script>
<script>
  const widget = window.initLibredesk({
    baseURL: 'http://localhost:9000',
    inboxID: 'xxx-uuid-xxx'
  });
  // 提供完整的 JS API
</script>
```

本项目采用手动初始化方式，以便于控制 widget 的生命周期。

### 关键设计决策

- **不引入额外依赖**：widget.js 本身就是纯 JS 脚本，Vue 3 + Vite 足够
- **localStorage 持久化配置**：避免每次刷新都要重新输入 baseURL 和 inboxID
- **响应式状态**：通过 Vue 响应式变量追踪 widget 的可见性、未读消息数等状态
- **组件通信**：配置面板通过 emit/props 将配置传递给父组件，父组件负责 widget 的初始化与销毁

## 实现细节

### 项目结构

所有文件均为新建，放在 `e:\libredesk\widget-frontend\` 目录下：

```
widget-frontend/
├── index.html              # [NEW] HTML 入口，加载 widget.js 脚本
├── package.json            # [NEW] 项目配置，仅依赖 vue 和 vite
├── vite.config.js          # [NEW] Vite 配置，开发服务器端口 5173
├── src/
│   ├── main.js             # [NEW] Vue 应用入口
│   ├── App.vue             # [NEW] 根组件，管理 widget 生命周期
│   ├── components/
│   │   ├── ConfigPanel.vue # [NEW] 配置面板：baseURL + inboxID 输入、保存按钮
│   │   ├── DemoPage.vue    # [NEW] 演示页面：仿公司官网 + widget API 控制区
│   │   ├── Navbar.vue      # [NEW] 网站导航栏
│   │   ├── HeroSection.vue # [NEW] Hero 区域（大标题 + 副标题 + CTA 按钮）
│   │   ├── FeatureCards.vue# [NEW] 功能卡片展示区
│   │   ├── Footer.vue      # [NEW] 页脚
│   │   └── WidgetControls.vue # [NEW] Widget API 控制面板（show/hide/logout 等按钮 + 状态显示）
│   └── composables/
│       └── useWidget.js    # [NEW] Widget 管理 composable（初始化、销毁、API 封装）
```

### 数据流

```
用户输入配置 → ConfigPanel emit → App.vue 保存到 localStorage
                                    ↓
                              useWidget composable 加载 widget.js
                                    ↓
                              创建 widget 实例 (window.initLibredesk)
                                    ↓
                              监听 widget 事件 (onShow/onHide/onUnreadCountChange)
                                    ↓
                              更新响应式状态 → WidgetControls 组件展示
```

### 关键代码结构

**useWidget composable** 封装 widget 初始化逻辑：

- `initWidget(baseURL, inboxID)` → 动态加载 widget.js 脚本，调用 `initLibredesk()`，注册回调
- `destroyWidget()` → 调用 widget.destroy()，移除脚本标签
- `show() / hide() / setUser(jwt) / logout()` → 代理调用 widget API
- 返回响应式状态：`isVisible`, `unreadCount`, `isReady`

**Widget 脚本动态加载**：
使用 `document.createElement('script')` 动态注入 widget.js，避免在不需要时阻塞页面加载。监听 `load` 事件确认加载完成后再调用 `initLibredesk()`。

## 设计风格

采用现代简约企业官网风格，以白色和浅灰为主色调，配合蓝紫色渐变点缀，营造专业、可信赖的科技公司形象。配置面板使用深色半透明弹窗，与主页面形成视觉层次。

## 页面规划

### 页面一：演示主页面（DemoPage）

**导航栏**：深色半透明背景，左侧 Logo 占位，右侧导航链接（首页、产品、定价、关于、联系我们）
**Hero 区域**：全宽渐变背景（深蓝到紫色），居中大标题 + 副标题 + CTA 按钮"开始使用"
**功能卡片区**：4 列网格布局，每张卡片包含图标占位符、标题、描述文字，悬浮时轻微上浮 + 阴影增强
**页脚**：浅灰背景，分栏显示公司信息、产品链接、社交媒体图标

### Widget 控制面板（WidgetControls）

固定在页面左侧的垂直面板，半透明玻璃拟态背景。包含：

- Widget 状态指示器（已连接/未连接 绿/红点）
- 未读消息数徽章
- API 控制按钮组：显示/隐藏、展开、销毁、重新初始化
- JWT 登录模拟按钮（用于 setUser API 演示）

### 配置面板（ConfigPanel）

页面加载时自动检测 localStorage，无配置则显示全屏遮罩配置面板：

- 居中卡片，深色玻璃拟态风格
- 输入框：backend URL（默认 http://localhost:9000）、Inbox ID
- 保存按钮 + 输入验证（URL 格式校验）