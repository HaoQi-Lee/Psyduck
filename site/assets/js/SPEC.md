---
psy_kind: factual
psy_version: 1
package: assets/js
---

# 概述

站点全部原生 JavaScript 交互模块所在目录。每个模块用 IIFE 包裹自身实现，仅将公开 API 挂载到 `window` 上的对应全局命名空间。`main.js` 作为入口，按依赖顺序调用各模块的 `init*` 函数。**有意不使用 ES Modules**，从而消除 `file://` 协议下的 CORS 限制，支持本地双击 `index.html` 直接运行。

# 文件

- `typewriter.js` — 打字机效果模块：单元素逐字符填充 + CLI 多行链式打字。
- `scroll-reveal.js` — 滚动揭示模块：基于 `IntersectionObserver` 监听 `.reveal` 元素，进入视口时切换 `.is-visible` class；同时为 `.cli-window` 元素触发 CLI 打字。
- `cursor-glow.js` — 鼠标跟随光晕：动态创建 `.cursor-glow` 元素并在 `mousemove` 时（`requestAnimationFrame` 节流）更新位置。仅桌面（细指针设备）启用。
- `parallax.js` — 背景视差：监听 `scroll` 事件（`requestAnimationFrame` 节流），将 `#hero` 的 `background-position-y` 设置为 `-scrollY * 0.4`。仅桌面 + 非 `prefers-reduced-motion` 启用。
- `mascot.js` — 吉祥物点击彩蛋：对 `[data-mascot]` 元素绑定 `click`，触发 `mascot-poke` 动画（强制 reflow 重置）。
- `main.js` — 入口模块：在 `DOMContentLoaded` 时按依赖顺序调用 `Typewriter.initTypewriters()` → `ScrollReveal.initScrollReveal()` → `CursorGlow.initCursorGlow()` → `Parallax.initParallax()` → `Mascot.initMascots()`。每次调用前通过 `if (window.X)` 守卫，确保模块缺失时不报错。通过 `document.readyState` 检测确保即使 defer 时机不同也能正确启动。

# API

所有模块通过 `window.<Namespace>` 暴露 API。

## `window.Typewriter`

```js
typewriterPlay(element, { text, speed = 55, startDelay = 0 }): Promise<void>
```
逐字符将 `text` 写入 `element.textContent`。`speed` 单位为毫秒（每字符）。

```js
typewriterPlayLines(container, { lines, speed = 35, lineDelay = 250 }): Promise<void>
```
多行链式打字。`container` 会被清空，依次创建 `<span class="cli-line is-typing">` 并填充每行内容，每行完成后移除 `is-typing`（关闭闪烁光标），行间停顿 `lineDelay`。

```js
initTypewriters(root = document): void
```
扫描 `root` 下所有 `[data-typewriter]` 元素，读取 `data-typewriter` 文本、`data-typewriter-speed`（默认 55）、`data-typewriter-delay`（默认 0），立即开始播放。Hero 标题与副标题用此机制。

```js
playCliWindow(element): Promise<void>
```
解析 `element` 的 `data-cli-lines` 属性（JSON 数组字符串），调用 `typewriterPlayLines` 播放。播放后在 `element.dataset.cliPlayed = '1'`，防止重复触发。JSON 解析失败时仅 `console.warn`，不抛出。

## `window.ScrollReveal`

```js
initScrollReveal({ root = null, rootMargin = '0px 0px -10% 0px', threshold = 0.1 } = {}): void
```
对 `document` 中所有 `.reveal` 创建 `IntersectionObserver`。元素进入视口时：

1. 添加 `.is-visible` class
2. 如果元素含 `.cli-window` class，查找其内部 `[data-cli-lines]` 子元素并调用 `Typewriter.playCliWindow()`
3. `unobserve(el)` 释放观察

不支持 `IntersectionObserver` 的浏览器降级为"全部立即可见 + 所有 CLI 立即播放"。

## `window.CursorGlow`

```js
initCursorGlow(): void
```
- `matchMedia('(pointer: coarse)')` 命中时立即 `return`（移动端）
- 创建 `<div class="cursor-glow is-hidden" aria-hidden="true">` 追加到 `body`
- 监听 `window.mousemove`（`{ passive: true }`），首次移动时移除 `.is-hidden`；用 `requestAnimationFrame` 节流，将 `transform: translate3d(x-200, y-200, 0)` 应用到光晕元素（400×400，居中对齐光标）
- 监听 `window.mouseleave`，重新挂上 `.is-hidden`

## `window.Parallax`

```js
initParallax(): void
```
- `pointer: coarse` 或 `prefers-reduced-motion: reduce` 命中时立即 `return`
- 选取 `#hero` 作为视差目标
- 监听 `window.scroll`（`{ passive: true }`），`requestAnimationFrame` 节流，每帧将 `backgroundPositionY = (-scrollY * 0.4) + 'px'` 应用到目标

## `window.Mascot`

```js
initMascots(): void
```
对所有 `[data-mascot]` 元素绑定 `click`：移除 `.is-poked` → 读取 `offsetWidth` 强制 reflow → 重新加上 `.is-poked`，触发 `mascot-poke` 关键帧（600ms 摇摆）。

# 依赖

- 仅依赖浏览器原生 API：`document.querySelectorAll`、`IntersectionObserver`、`requestAnimationFrame`、`matchMedia`、`Promise` / `async-await`、`JSON.parse`
- 不依赖任何 npm 包、jQuery、GSAP、framer-motion 等
- 模块之间的依赖：`scroll-reveal.js` 依赖 `window.Typewriter`（CLI 触发），`main.js` 依赖所有 5 个模块

# 设计重点

**避免 ES Modules 的协议限制**：每个模块用 `(function () { 'use strict'; ... })()` 包裹，将公开 API 挂载到 `window.Typewriter` 等命名空间。HTML 通过 `<script src="..." defer></script>` 顺序加载，`defer` 保证脚本在 DOM 解析完成后才执行且按声明顺序运行，因此 `main.js` 中 `window.Typewriter` 等全局对象一定已存在。这种设计的代价是失去模块系统的静态依赖分析，但收益是可以本地 `file://` 协议直接打开（典型用户场景：作者本地预览或邮件 ZIP 分享）。

**rAF 节流的标准模式**：`cursor-glow.js` 与 `parallax.js` 均使用 `ticking` 布尔标志 + `requestAnimationFrame(update)` 的组合：事件触发时若 `ticking` 为 `false` 才 `rAF`，`update()` 末尾将 `ticking` 重置为 `false`。这样无论事件触发频率多高，每帧最多执行一次写入。所有滚动 / 鼠标监听都使用 `{ passive: true }` 避免阻塞主线程。

**`IntersectionObserver` 的"一次性触发"策略**：揭示后立即 `unobserve(el)`，避免观察器无限累积。`playCliWindow` 还通过 `element.dataset.cliPlayed === '1'` 做幂等守卫，即使元素被重复 observe 也只会播放一次。

**移动端 / 减弱动画的早期短路**：cursor-glow 与 parallax 在函数入口就检测 `matchMedia` 并直接 `return`，不创建任何 DOM 也不绑定任何事件。这比"先创建再隐藏"更彻底——既节省内存也避免移动 Safari 上 `background-attachment: fixed` 的性能问题。

**CLI 数据契约**：CLI 终端通过 `[data-cli-lines]` 属性传递内容，值必须是合法 JSON 数组字符串（外层使用单引号 HTML 属性，内层用双引号），例如 `data-cli-lines='["$ cmd", "output"]'`。`typewriter.js` 在 `JSON.parse` 失败时仅 `console.warn` 不抛出，保证数据异常时站点其余功能不受影响。
