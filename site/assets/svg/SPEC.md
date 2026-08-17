---
psy_kind: factual
psy_version: 1
package: assets/svg
---

# 概述

站点所有自绘 SVG 视觉资源所在目录。包含两套素材：**主视觉**（Hero 区使用的渐变 + 辉光抽象图形，200×200）与 **8-bit 像素角标**（项目区使用的 16×16 sticky sprite）。全部为原创自绘，**不使用任何官方宝可梦素材**，纯抽象致敬，无版权风险。

# 文件

- `psyduck.svg` — Hero 区主视觉：黄色困惑鸭子抽象造型，viewBox 200×200。使用 `psyduckBody` 径向渐变（淡黄 → 鸭黄 → 暗金）+ `psyduckGlow` 辉光滤镜。包含头顶三根呆毛、橙色嘴喙、标志性小圆点眼睛、左右两只捧头小手（带 ±20° 旋转）。
- `grid-bg.svg` — Hero 区共享的背景网格图案，80×80 viewBox。使用 SVG `pattern` 元素生成无限重复的细线方格，stroke 为 `rgba(255,255,255,0.08)`。`aria-hidden="true"`（纯装饰）。
- `sprites/psyduck-pixel.svg` — Psyduck 项目区 sticky 角标：16×16 viewBox 黄色鸭子像素图。使用 `shape-rendering="crispEdges"` 保持像素感（禁用反锯齿）。鸭黄身体（`#ffd23f`）+ 深色像素点眼 + 橙色嘴部条纹。

# API

SVG 文件以静态资源形式被 `index.html` 通过 `<img src="...">` 标签引用：

| 文件 | 在 index.html 中的引用 | 在 CSS 中的引用 |
|---|---|---|
| `psyduck.svg` | `<img class="hero-mascot is-psyduck" data-mascot="psyduck">` | — |
| `grid-bg.svg` | — | `hero.css` 中 `background-image: url('../svg/grid-bg.svg')` |
| `sprites/psyduck-pixel.svg` | `<img class="project-sprite is-right">` | — |

**无障碍契约：**
- 有语义的主视觉 SVG（`psyduck.svg`）必须使用 `role="img"` + `<title>` + `aria-labelledby` 提供文本替代；项目区 sticky sprite 通过 `<img alt="" aria-hidden="true">` 在 HTML 层面声明为装饰
- 纯装饰 SVG（`grid-bg.svg`）使用 `aria-hidden="true"`

# 依赖

- 仅依赖 SVG 1.1 标准（`<radialGradient>` / `<filter>` / `<feGaussianBlur>` / `<feMerge>` / `<pattern>`）
- 不依赖任何外部 SVG 库 / 工具链 / 图标系统（Lucide / Heroicons / FontAwesome 等）
- 不依赖任何字体（所有视觉为几何图形 + 路径）

# 设计重点

**两套素材的视觉节奏**：Hero 区使用 200×200 高质量渐变 + 辉光的"主视觉"建立角色辨识；项目区使用 16×16 像素 sprite 作为"游戏机般"的极客锚点，呼应站点的赛博朋克 / CLI 终端调性。两套素材共享同一 Psyduck 调色板（鸭黄 `#ffd23f`），保证视觉一致性。

**SVG 不引用 CSS 变量**：所有颜色直接以十六进制 / RGB 字面量写入 SVG 标签内。这是因为 SVG `fill` / `stroke` 属性在 `<img src="*.svg">` 引用模式下无法读取外部 CSS 自定义属性（需要 inline SVG 嵌入才能引用）。当前牺牲了主题切换灵活性，换取性能（浏览器可缓存 SVG 资源）与简洁。

**像素 sprite 的 `crispEdges` 渲染**：sprite SVG 显式声明 `shape-rendering="crispEdges"`，禁用浏览器对边缘的反锯齿处理，确保 16×16 像素图被 CSS 放大到 64×64（桌面）或 40×40（移动端）时保持锯齿状的像素感。CSS 层面在 `.project-sprite` 上配合 `image-rendering: pixelated` 进一步强化。

**辉光滤镜的封装**：角色 SVG 内部定义自己的 `filter` ID（`psyduckGlow`），避免跨 SVG 的 ID 冲突。这样即使未来加载多个 SVG 在同一文档中时不会因为同名 filter 互相覆盖。
