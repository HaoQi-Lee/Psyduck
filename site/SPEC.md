---
psy_kind: factual
psy_version: 1
package: .
---

# 概述

Psyduck Geek Site 是一个二次元极客风格的单页静态产品落地页，为 **Psyduck**（面向 Claude Code 的 SPEC 生命周期管理框架）打造。页面展示框架三大特性、CLI 演示终端、多平台二进制下载入口。整站零构建步骤、零第三方运行时 JavaScript 库，可直接通过任意静态托管部署。

# 文件

- `index.html` — 单页入口，三个区块自上而下：Hero / Psyduck Project / Footer。包含 SEO meta、Open Graph、Twitter Card、JSON-LD 结构化数据。
- `README.md` — 项目说明，含本地运行与部署指引。
- `assets/og-image.png` — 社交分享卡片图（1200×630）。
- `assets/css/` — 样式包，详见 [`assets/css/SPEC.md`](assets/css/SPEC.md)。
- `assets/js/` — 原生 JS 交互模块包，详见 [`assets/js/SPEC.md`](assets/js/SPEC.md)。
- `assets/svg/` — SVG 视觉资源包，详见 [`assets/svg/SPEC.md`](assets/svg/SPEC.md)。
- `CLAUDE.md` — psyduck spec 生命周期约定（项目级 AI 指令）。
- `.psy/` — psyduck 工作流归档目录。

# API

本项目"对外"等同于浏览器加载 `index.html`。HTML 与 JS 模块之间通过以下 `data-*` 钩子契约耦合（详见各子包 SPEC）：

| 钩子 | 用途 | 消费方 |
|---|---|---|
| `[data-typewriter]` / `data-typewriter-speed` / `data-typewriter-delay` | 元素被打字机逐字符填充 | `assets/js/typewriter.js` |
| `[data-mascot]` | 吉祥物点击彩蛋（poke 动画） | `assets/js/mascot.js` |
| `.reveal` → `.is-visible` | 进入视口时淡入上滑 | `assets/js/scroll-reveal.js` |
| `.cli-window` + `[data-cli-lines]`（JSON 数组） | CLI 终端逐行打字效果 | `assets/js/scroll-reveal.js` → `assets/js/typewriter.js` |

# 依赖

- **运行时**：浏览器原生 API（HTML / CSS / ES2020+ JS）
- **外部资源**：Google Fonts CDN（`Orbitron` + `JetBrains Mono`，`display=swap`）
- **构建依赖**：无（零构建步骤）
- **npm/pip 等包**：无

# 核心架构

**单页 + 区段主题**：所有内容在 `index.html` 一个文件内，通过 `<section>` / `<footer>` 分段。Psyduck 项目区带 `.theme-psyduck` class，对 `.project-section` 内的局部 CSS 变量（`--theme-bg` / `--theme-glow` / `--theme-accent` / `--theme-grid`）进行覆盖，形成独立的主题色。

**CSS 加载顺序固定**：reset → tokens → base → 各区段（hero / project / theme-psyduck）→ animations → footer。tokens.css 集中所有设计令牌（颜色 / 字体 / 间距 / 圆角 / 动效曲线 / z-index）。

**JS 全局命名空间 + defer 装配**：6 个 JS 模块各自 IIFE 包裹并将公开 API 挂载到 `window.Typewriter` / `ScrollReveal` / `CursorGlow` / `Parallax` / `Mascot`；`main.js` 通过 `defer` 顺序加载，在 `DOMContentLoaded` 时按依赖顺序调用各模块的 `init*`。**不使用 ES Modules**，从而避免 `file://` 协议下的 CORS 限制，支持本地直接双击打开。

# 响应式断点

| 断点 | 范围 | 调整 |
|---|---|---|
| 桌面 | `>= 1024px` | 完整体验 |
| 平板 | `768px ~ 1023px` | 字号缩小，特性卡片 3 → 2 列 |
| 移动 | `< 768px` | 关闭鼠标光晕、视差、sticky sprite、`background-attachment: fixed`；Hero 改竖排堆叠；特性卡片 1 列；CLI 字号 0.75rem；下载按钮竖排 |

移动端降级通过 `@media` 内联在各模块 CSS 文件末尾实现，不独立 `mobile.css`。

# 无障碍与性能

- 所有装饰性 SVG 使用 `aria-hidden="true"`；语义性 SVG 使用 `<title>` + `aria-labelledby`
- `prefers-reduced-motion: reduce` 在 `animations.css` 中全局禁用所有 keyframes 与 transitions
- 所有动画仅使用 `transform` 与 `opacity`（GPU 加速）
- 鼠标 / 滚动事件使用 `requestAnimationFrame` 节流
- 移动端（`pointer: coarse`）关闭 cursor-glow 与 parallax
- `IntersectionObserver` 在元素揭示后立即 `unobserve` 释放
- `:focus-visible` 描边使用霓虹青色 2px

# 部署

1. 推送到 `master` / `main` 分支
2. 在静态托管服务配置根目录
3. GitHub Pages：Settings → Pages → 选择分支根目录
4. 访问 `https://<user>.github.io/<repo>/`
