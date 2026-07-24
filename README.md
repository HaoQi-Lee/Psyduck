# PsyDuck

> 面向 Claude Code 工作流的轻量 **Spec 生命周期管理框架**。
> **只管 spec 的生命周期**——内容生成交给 `superpowers` 等技能。

## Quick start

```bash
# Build (Go 1.22+)
make build        # → bin/psy
# 或
go build -o bin/psy ./cmd/psy

# In your repo (cd to repo root first)
./psy init --install-plugins   # create .psy/ + install skills as /slash commands
```

`psy` 二进制零网络调用、零 LLM 依赖，只读写 `.psy/` 目录和各 package 的 `SPEC.md`。

---

## API Reference

### `psy init`

初始化 `.psy/` 目录，并追加 SPEC.md 读取规则到 `CLAUDE.md`。在仓库根目录下运行。

```bash
psy init [--install-plugins]
```

| Flag | Description |
|---|---|
| `--install-plugins` | 把内置 skill 安装到 `~/.claude/skills/<name>/SKILL.md`（全局 `/` 斜杠命令）。已有同名目录会跳过，不覆盖用户自定义 |

```bash
# 最小初始化（只建 .psy/ + 更新 CLAUDE.md）
psy init

# 同时安装全局 /slash 命令
psy init --install-plugins
```

### `psy version`

```bash
psy version
# 0.0.0-dev
```

版本号通过 `go build -ldflags "-X github.com/psyduck/psyduck/internal/version.Version=vX.Y.Z"` 注入。

### `psy check`

只读检测每个 `SPEC.md` **自上次同步以来**的漂移，作为 CI / pre-commit 门禁。零参数、零 flag。

检测以每个 `SPEC.md` 的**最近一次提交**为锚点，用 `git diff` 取该包目录内自锚点以来的净变更，分类为漂移与过期：

- **结构漂移**（退出码 1，失败门禁）：
  - **新增**了一个源文件（类型属 SPEC 已声明、但 `# 文件` 未列）；
  - **删除**了一个 `# 文件` 仍列出的文件；
  - 或 `package:` 路径与实际位置不符，或缺少 `# 文件` 章节。
- **时序提示**（仅打印，退出码 0）：源文件自同步以来**仅内容修改**，提示 spec 可能过期。

变更取自 `git diff --name-status <锚点> HEAD -- <包目录>`（git 为硬依赖；**只看已提交的变更**，未提交的工作区改动不计）。diff 条目剔除：

- `SPEC.md` 自身；
- 含子 `SPEC.md` 的嵌套包子目录（归子包，不在父包重复计入）；
- **非代码目录**：`testdata`、`vendor`、`node_modules`，以及点前缀目录（`.idea`、`.git`、`.vscode` 等）——按任意一级目录段匹配。

> **资源文件豁免**：`# 文件` 只列源文件。新增或修改的资源文件（`*.png`、`*.yaml`、locale 等）只要其类型不在该包 SPEC 已声明的扩展名词汇内，就既不算漂移也不算过期——「哪些算源码」由每个包自己的 SPEC 定义，check 不猜测目录/扩展名。

> **只报增量**：上次同步**之前**就存在的历史不一致不再报告（不在锚点之后）。`/psy-sync` 重写 SPEC 后该包立即归零。

> **根目录恒非包**：仓库根的 `SPEC.md`（即使存在）被跳过——根级文件（`go.mod`、`README.md` 等）与独立非包目录天然不参与 check。

```bash
psy check
# pkg: NG
#   + new.go (added)
#   - old.go (removed)
#   ~ root.go (3d newer than spec)
# summary: 1 drift, 1 stale   (exit 1)
```

| 退出码 | 含义 |
|---|---|
| `0` | 无结构漂移（时序提示不影响） |
| `1` | 检测到结构漂移 |
| `70` | 内部错误（非 git 仓库 / git 缺失 / 读取或 git 调用失败） |

> `psy check` 只读：不写文件、不联网、不调 LLM。修复漂移请用 `/psy-sync`。

---

## Skills

通过 `psy init --install-plugins` 安装到 `~/.claude/skills/`，作为 `/` 斜杠命令使用。

### `/psy-sync` — 变更集同步

**触发时机（两种均强制）：**

1. **自动**——`superpowers:executing-plans` 执行完成之后，立即调用，作为关闭实施流程的一步。
2. **手动**——用户主动运行 `/psy-sync` 时。

**执行步骤：**

| 步骤 | 动作 |
|---|---|
| 1 | 确定变更集：从刚执行的计划上下文、`git status`、或近期提交推断受影响文件 |
| 2 | 按代码包分组：Go package / JS/TS 模块 / Python package / Rust crate 等 |
| 3 | 对每个包读源码，生成或更新 `SPEC.md`（中文），保持已有 front-matter |
| 4 | 归档本次变更产生的设计文档（`docs/superpowers/specs/*.md`）到 `.psy/<YYYY-MM-DD>/<basename>.md`，冲突追加 `-2`、`-3`…… |
| 5 | 同步仓库根 `README.md`：校正代码派生的事实（命令/flag/退出码/目录结构/构建测试命令/依赖/skill 步骤），并按现有结构为新增暴露面补写中性条目；编辑性散文保持不动 |

**核心规则：**
- 文档现实，不写期望
- 归档是 `git mv`，非复制
- 变更集为空时提示用户，不扩展为全量扫描
- `executing-plans` 之后不可跳过

### `/psy-sync-all` — 全量同步

**仅手动触发**——运行 `/psy-sync-all` 时。

**适用场景：**
- 在已有代码库上初次接入 psyduck（bootstrap）
- 怀疑存在未被近期变更触及的 spec 漂移（drift recovery）
- 需要一个干净基线再开始使用变更集工作流

**执行步骤：**

| 步骤 | 动作 |
|---|---|
| 1 | 遍历整个仓库，识别所有代码包 |
| 2 | 对每个包读源码，生成或更新 `SPEC.md`（中文） |
| 3 | 归档 `docs/superpowers/specs/` 下所有未归档的文档到 `.psy/<YYYY-MM-DD>/<basename>.md` |
| 4 | 同步仓库根 `README.md`：按整库现状重新核对所有代码派生章节，修正过时事实并为缺失暴露面按现有结构补写条目 |

首次运行会产生大量 diff，建议批量审查后提交。

---

## SPEC.md Format

每个代码包的 `SPEC.md` 由 skill 自动维护，结构如下：

```markdown
---
psy_kind: factual
psy_version: 1
package: internal/cli
---

# 概述

<!-- 一段话描述这个代码包的主要作用和职责 -->

# 文件

<!-- 罗列代码包内的所有源文件，每个附一句简短说明 -->

# API

<!-- 对外暴露的接口（导出函数、类型等），每个附一行中文描述 -->

# 依赖

<!-- 该包依赖的其他内部包以及第三方库 -->

# 设计重点 *(可选)*

<!-- 核心设计决策、架构模式、实现细节等 -->
```

---

## Directory Structure

```
<repo-root>/
├── .psy/                                        # psyduck 元数据
│   └── YYYY-MM-DD/                              # 按日期归档
│       └── <doc-name>.md                         # 归档的设计文档
│
├── internal/auth/
│   ├── auth.go
│   └── SPEC.md                                  # factual spec
└── cmd/server/
    ├── main.go
    └── SPEC.md
```

---

## Implementation

### Tech Stack

| 关注点 | 选型 |
|---|---|
| Language | Go 1.22（单二进制、零运行时、跨平台） |
| CLI | `spf13/cobra` |
| Git | `os/exec` 调系统 `git` |
| Testing | `testing` + `testify` + `rogpeppe/go-internal/testscript` |
| Embedded Assets | `//go:embed`（skills、CLAUDE.md section） |
| Version | `go build -ldflags` 注入 |

### Dependencies

| Package | 用途 |
|---|---|
| `spf13/cobra` | CLI 框架 |
| `stretchr/testify` | 测试断言 |
| `rogpeppe/go-internal` | testscript E2E 测试 |

### Project Layout

```
psyduck/
├── cmd/psy/                   # 入口 main.go
├── internal/
│   ├── cli/                   # 子命令 (root/init/version/check) + 退出码
│   │   ├── claudemd/          #   嵌入的 CLAUDE.md section
│   │   └── skills/            #   嵌入的 skill 文件 (psy-sync, psy-sync-all)
│   ├── spec/                  # check 漂移检测纯逻辑：解析 + git 抽象 + 变更分类
│   └── version/               # 构建期注入版本号
├── testdata/script/           # testscript E2E 测试
├── docs/                      # 设计文档暂存（归档后清空）
└── .psy/                      # psyduck 自身的 spec（dogfooding）
```

### Build & Test

```bash
make build     # go build -o bin/psy ./cmd/psy
make test      # go test ./...
make lint      # go vet ./...
make tidy      # go mod tidy
```

### Design Principles

- **薄**：文件即数据，git 即状态，psy 是这两者之上的薄封装
- **零侵入**：不改源码、不加依赖、不联网、不调 LLM，只读写 `.psy/` 和 `SPEC.md`
- **grep 友好**：`SPEC.md` 是标准 markdown + YAML front-matter
- **不抢戏**：psyduck 不生成内容、不调 LLM、不生成代码
