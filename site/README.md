# Psyduck Geek Site

二次元极客风格产品落地页 — 为 [Psyduck](http://git.showcai.com.cn:7990/projects/TECH/repos/psyduck/browse) 打造的单页静态站，展示 SPEC 生命周期管理框架的特性、CLI 演示与下载入口。

## 页面结构

| 区域 | 内容 |
|------|------|
| **Hero** | 打字机动画标题 + 标语 + Psyduck 吉祥物 |
| **Project** | 特性卡片（SPEC 即文档 / 变更集同步 / 零侵入架构）+ CLI 终端模拟 + 下载按钮 |
| **Footer** | 社区标语 + 社交导航（GitHub / Docs / Issues） |

## 技术栈

- 纯 HTML + CSS + 原生 JS — 零构建工具、零第三方运行时依赖
- Google Fonts：`Orbitron`（展示字体）、`JetBrains Mono`（等宽字体）
- SEO：Open Graph、Twitter Card、JSON-LD 结构化数据
- 部署：任意静态托管（GitHub Pages、Nginx 等）

## 交互效果

| 特效 | 实现 |
|------|------|
| 打字机动画 | `typewriter.js` — 逐字输出 Hero 标题与副标题 |
| 滚动揭示 | `scroll-reveal.js` — IntersectionObserver 驱动的渐入动画 |
| 光标追踪光晕 | `cursor-glow.js` — 跟随鼠标的霓虹光效 |
| 视差背景 | `parallax.js` — 滚动驱动的背景位移动画 |
| 吉祥物动画 | `mascot.js` — 闲置摆动 + hover 放大交互 |

## 本地运行

```bash
# 任选其一
python -m http.server 8000
# 或
npx http-server -p 8000
```

浏览器打开 `http://localhost:8000`。

## 目录结构

```
├── index.html                    # 主页面
├── assets/
│   ├── css/
│   │   ├── reset.css             # CSS Reset
│   │   ├── tokens.css            # 设计令牌（颜色、字号、间距、动效）
│   │   ├── base.css              # 全局基础样式
│   │   ├── hero.css              # Hero 区域
│   │   ├── project.css           # 项目展示区
│   │   ├── theme-psyduck.css     # Psyduck 主题配色
│   │   ├── animations.css        # 关键帧动画集合
│   │   └── footer.css            # 页脚
│   ├── js/
│   │   ├── typewriter.js         # 打字机效果
│   │   ├── scroll-reveal.js      # 滚动揭示
│   │   ├── cursor-glow.js        # 鼠标光晕
│   │   ├── parallax.js           # 视差滚动
│   │   ├── mascot.js             # 吉祥物交互
│   │   └── main.js               # 入口，装配所有模块
│   ├── svg/
│   │   ├── psyduck.svg           # Hero 吉祥物
│   │   ├── grid-bg.svg           # 背景网格纹理
│   │   └── sprites/
│   │       └── psyduck-pixel.svg # 项目区像素风精灵图
│   └── og-image.png              # Open Graph 分享图（1200×630）
├── .psy/                         # Psyduck 归档目录
└── README.md
```

## 部署

1. 推送到 `main` / `master` 分支
2. 在静态托管服务配置根目录
3. GitHub Pages：Settings → Pages → 选择分支根目录
4. 访问 `https://<user>.github.io/<repo>/`

## 相关链接

- **仓库**：[git.showcai.com.cn/psyduck](http://git.showcai.com.cn:7990/projects/TECH/repos/psyduck/browse)
- **文档**：[cfu.showcai.com.cn/psyduck](http://cfu.showcai.com.cn:8090/display/architecture/I.3+PsyDuck)
