---
psy_kind: factual
psy_version: 1
package: cmd/psy
created: 2026-06-05
---

# 概述

`psy` 二进制入口点。将 `internal/cli.NewRootCmd` 接入 OS 标准流，执行命令并在出错时以 `ExitInternal` 退出码退出。

# 文件

- `main.go` — 程序入口，构建根命令并执行。

# API

- `main()` — 程序入口函数。

# 依赖

- `internal/cli` — 根命令构建与子命令注册。
- `fmt`、`os` — 标准库。

# 设计重点

## 错误处理策略

`main.go` 不区分错误类型，统一以 `cli.ExitInternal`（70）退出。所有子命令错误均由 cobra 的 `RunE` 返回，cobra 本身通过 `SilenceErrors: true` 抑制重复输出，由 `main` 自行格式化并打印到 stderr。
