---
psy_kind: factual
psy_version: 1
package: assets/css
---

# 概述

站点全部样式表所在目录。按"reset → 设计令牌 → 全局基础 → 各区段 → 主题覆盖 → 动效 → footer"分层组织，每个文件只承担一项职责，避免相互污染。所有颜色 / 字体 / 间距 / 圆角 / 动效曲线均通过 `tokens.css` 中的 CSS 变量集中管理；项目区主题色通过 `.theme-psyduck` class 复合选择器覆盖局部变量实现。

# 文件

- `reset.css` — 极简浏览器默认样式重置，包含 `box-sizing: border-box`、移除默认 margin/padding、`img/svg/button/a/ul/ol` 归一化、`:focus-visible` 描边。引用了 `--color-neon-cyan` 并提供 `#00f0ff` fallback，保证 reset.css 可独立使用。
- `tokens.css` — 全局设计令牌，定义 `:root` 下所有 CSS 自定义属性：基底色、霓虹色、Psyduck 主题色、字体栈、字号、间距、圆角、动效曲线 / 时长、z-index 层级。
- `base.css` — 全局基础样式：body 背景与字体、`section` 默认 padding 与 `overflow: hidden`、`.container` 1024px 居中容器、`h1/h2/h3` 字号、`.reveal` 初始态与 `.is-visible` 揭示态、`.visually-hidden` 无障碍工具类。
- `hero.css` — Hero 区段：`#hero` 全屏高度 + 固定背景网格、`.hero-handle` 大标题、`.hero-tagline` 副标题、`.hero-bottom` 构图、`.hero-mascot.is-psyduck` 吉祥物尺寸与位移、`.hero-scroll-hint` 底部滚动提示。内含 `@keyframes float` / `bounce` 两个 idle 动画。文件末尾包含移动端（`< 768px`）与平板（`768-1023px`）`@media` 降级。
- `project.css` — Psyduck 项目区**完整结构**。定义 `.project-section` 容器并初始化 4 个局部主题变量（`--theme-bg/-glow/-accent/-grid`）的默认值。包含 `.project-sprite`（sticky 像素角标）、`.project-title/-subtitle/-intro`、`.feature-grid` + `.feature-card`、`.cli-window` + `.cli-titlebar/-body/-line` + `.cli-dot.is-{red,yellow,green}`、`.cta-group` + `.cta-btn.is-primary/.is-secondary`、`.download-group` + `.download-label` + `.download-btns` + `.download-btn` + `.download-os/-name/-arch`。内含 `@keyframes pulse-glow`。文件末尾包含平板 2 列 + 移动端 1 列降级。
- `theme-psyduck.css` — Psyduck 主题覆盖：通过 `.theme-psyduck.project-section` 复合选择器覆盖 `--theme-*` 局部变量为 Psyduck 深蓝底 + 鸭黄光晕 + 水青强调色 + 水青网格；同时设置该区段背景的 linear-gradient 网格图案。
- `animations.css` — 共享动效资源：`.cursor-glow` 鼠标跟随光晕容器（含 `mix-blend-mode: screen`）、`.cli-line.is-typing::after` 闪烁方块光标、`@keyframes blink`、`@keyframes mascot-poke`、`.hero-mascot.is-poked` 点击彩蛋动画、`@media (prefers-reduced-motion: reduce)` 全局禁用所有动画。
- `footer.css` — Footer 区段：`#site-footer` 顶部细线分隔、`.footer-tagline` 标语、`.neon-heart` 粉色心形辉光、`.footer-social` 社交链接行、`.footer-copyright` 版权小字。

# API

CSS 包通过以下"对外契约"被 `index.html` 与 `assets/js/` 消费：

| 类别 | 名称 | 用途 |
|---|---|---|
| **全局令牌** | `--color-bg-base` / `--color-neon-pink` / `--color-neon-cyan` / `--color-text-primary` / `--color-text-muted` | 基础调色板 |
| | `--color-bg-psyduck` / `--color-psyduck-glow` / `--color-psyduck-accent` | Psyduck 主题色 |
| | `--font-display` / `--font-mono` | 字体栈 |
| | `--fs-hero/-section/-h3/-body/-small/-cli` | 字号 |
| | `--space-xs/-sm/-md/-lg/-xl` | 间距 |
| | `--radius-sm/-md/-pill` | 圆角 |
| | `--ease-out/-in-out` / `--duration-fast/-base/-slow` | 动效 |
| | `--z-bg/-base/-sticky/-cursor` | z-index 层级 |
| **局部令牌** | `--theme-bg/-glow/-accent/-grid` | 在 `.project-section` 中初始化，由 `.theme-psyduck` class 覆盖 |
| **状态 class** | `.reveal` / `.reveal.is-visible` | scroll-reveal.js 切换 |
| | `.cli-line.is-typing` | typewriter.js 在每行打字时挂载，结束后移除 |
| | `.cursor-glow` / `.cursor-glow.is-hidden` | cursor-glow.js 动态创建并切换 |
| | `.hero-mascot.is-poked` | mascot.js 点击时挂载，动画结束自动失效 |
| **结构 class** | `.container` / `.project-section` / `.feature-grid` / `.feature-card` / `.cli-window` / `.cli-titlebar` / `.cli-dot.is-{red,yellow,green}` / `.cli-body` / `.cta-group` / `.cta-btn.is-primary/.is-secondary` / `.download-group` / `.download-btns` / `.download-btn` / `.hero-mascot.is-psyduck` / `.hero-handle` / `.hero-tagline` / `.footer-*` | 直接在 `index.html` 中使用 |
| **无障碍工具** | `.visually-hidden` | 屏幕阅读器专属文本 |

# 依赖

- 仅依赖标准 CSS3（自定义属性、Grid、Flex、`@keyframes`、`backdrop-filter`、`@media`、`@supports` 等浏览器原生特性）
- 字体通过 `index.html` 的 `<link>` 引入 Google Fonts，本目录文件不直接 `@import`
- 不依赖任何 CSS 预处理器（Sass / Less / PostCSS）

# 设计重点

**主题覆盖通过 CSS 变量级联**：`.project-section` 在 `project.css` 中以 `0,1,0` 特异性初始化 4 个 `--theme-*` 局部变量；`.theme-psyduck.project-section` 在主题文件中以 `0,2,0` 特异性覆盖。CSS link 加载顺序保证主题文件晚于 project.css，确保级联生效。所有 `.project-section` 内部样式只引用局部变量，因此切换主题无需重写任何结构样式。

**关键帧分布策略**：仅在 Hero 使用的 `float` / `bounce` 放在 `hero.css` 内；项目区按钮辉光 `pulse-glow` 放在 `project.css` 内（与 `.cta-btn.is-primary` 紧邻）；跨区段共享的 `blink`（CLI 光标）与 `mascot-poke`（吉祥物点击）放在 `animations.css`。这样每个区段的动画与其结构样式共定位，移除该区段时关键帧也一起带走。

**移动端降级在文件末尾就近声明**：每个区段 CSS 文件末尾追加 `@media (max-width: 767px)` 与可选平板 `@media`，而非集中到 `mobile.css`。这样阅读单一区段样式时可以一并看到其响应式行为，降低跨文件跳转成本。

**`prefers-reduced-motion` 在 `animations.css` 中通过 `*, *::before, *::after { animation-duration: 0.01ms !important; transition-duration: 0.01ms !important }` 实现"近乎瞬移落位"**，等价于禁用所有动画但保留最终状态。这是当前最简洁可靠的全局禁用模式。
