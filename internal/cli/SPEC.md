---
psy_kind: factual
psy_version: 1
package: internal/cli
created: 2026-06-05
---

# 概述

`psy` 二进制的 Cobra 命令树。每个子命令一个文件，提供 `init`、`version`、`check` 等子命令的实现，以及嵌入式资源（CLAUDE.md 片段、skill 文件）。

# 文件

- `root.go` — 构建根命令，注册所有子命令。
- `init.go` — `psy init` 子命令，初始化 `.psy/` 目录并追加 CLAUDE.md 片段；`--install-plugins` 标志将嵌入 skill 安装到 `~/.claude/skills/`。
- `version.go` — `psy version` 子命令，打印版本号。
- `check.go` — `psy check` 子命令，调用 `internal/spec.Check` 渲染纯文本报告，漂移返回 `ExitError{Code: ExitDrift}`。
- `exitcodes.go` — 退出码常量（`ExitOK`=0、`ExitDrift`=1、`ExitInternal`=70）、`ExitError` 类型、`ExitCodeFromErr` 错误→退出码映射。
- `skills_embed.go` — 通过 `//go:embed` 嵌入 `skills/*.md` 为 `embed.FS`。
- `claudemd/section.md` — 嵌入的 CLAUDE.md 片段，包含 psyduck 生命周期说明。
- `skills/psy-sync.md` — 嵌入的 `psy-sync` skill 文件。
- `skills/psy-sync-all.md` — 嵌入的 `psy-sync-all` skill 文件。
- `init_test.go` — `init` 子命令的单元测试。
- `version_test.go` — `version` 子命令的单元测试。
- `check_test.go` — `renderReport` 纯函数单测 + `runCheck` 退出码集成测试（真实临时 git 仓库）。
- `exitcodes_test.go` — `ExitCodeFromErr` 表驱动单测。
- `script_test.go` — 基于 `testscript` 的集成测试入口，将 `psy` 命令注册为 `testscript` 内置命令。
- `testutil_test.go` — 测试辅助函数（`chdir`、`setupInitialized`、`writeValidSpec`、`mustWriteFile`）。

# API

- `NewRootCmd(stdout, stderr io.Writer) *cobra.Command` — 构建并返回 `psy` 根命令，注入 IO 流用于测试。
- `ExitOK = 0` — 正常退出码。
- `ExitDrift = 1` — `psy check` 检测到结构漂移（门禁失败）。
- `ExitInternal = 70` — 内部错误退出码。
- `ExitError struct { Code int; Msg string }` — 携带退出码的错误；非空 `Msg` 由 `main` 打到 stderr。
- `ExitCodeFromErr(err error) (int, string)` — 把命令错误映射为退出码与可选 stderr 信息。

# 子命令

- `psy init` — 在当前目录创建 `.psy/` 目录，追加 psyduck 生命周期说明到 `CLAUDE.md`（幂等，通过 `<!-- psyduck -->` 标记检测重复）。
- `psy init --install-plugins` — 将嵌入的 skill 文件安装到 `~/.claude/skills/<name>/SKILL.md`，已存在的目录跳过不覆盖。
- `psy version` — 输出版本号。
- `psy check` — 只读检测每个 `SPEC.md` 与现实的漂移（结构漂移退出 1，时序仅提示退出 0），零参数零 flag。

# 依赖

- `github.com/spf13/cobra` — 命令行框架。
- `github.com/rogpeppe/go-internal/testscript` — 集成测试引擎（仅测试文件）。
- `github.com/stretchr/testify` — 断言库（仅测试文件）。
- `internal/version` — 版本号字符串（仅 `version.go`）。
- `internal/spec` — 漂移检测核心（仅 `check.go`）。

# 设计重点

## 嵌入式资源管理

`claudemd/section.md` 通过 `//go:embed` 嵌入为字符串变量 `claudeMdSection`，`skills/*.md` 嵌入为 `embed.FS` 变量 `skillFiles`。这种方式将资源与二进制打包，安装时无需外部文件。

## 测试架构

测试分两层：`*_test.go` 单元测试覆盖各子命令逻辑；`testdata/script/` 下的 `testscript` 文件提供端到端集成测试。`TestMain` 将 `psy` 命令注册为 testscript 内置命令，无需预编译二进制。

## 初始化幂等性

`ensureClaudeMd` 通过 `<!-- psyduck -->` 标记检测 CLAUDE.md 中是否已包含 psyduck 片段，确保重复调用不会产生重复内容。`init` 命令本身通过检测 `.psy/` 目录存在来防止重复初始化。

## 插件安装策略

`installPluginsToDir` 以目录为单位管理 skill：如果目标 skill 目录已存在则跳过，保护用户自定义内容不被覆盖。安装位置为 `~/.claude/skills/<name>/SKILL.md`，符合 Claude Code 全局 skill 发现规范。

## 退出码与门禁

`ExitError` + `ExitCodeFromErr` 让 `psy check` 区分「漂移=1」（结果，非错误）与「内部错误=70」。`main` 与 testscript 入口共用 `ExitCodeFromErr`，保证生产与测试退出码一致。`psy check` 只读、零参数零 flag，适合 CI / pre-commit 门禁。
