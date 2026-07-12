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

`main.go` 经 `cli.ExitCodeFromErr` 把命令错误映射为退出码：`*cli.ExitError` 用其携带的码（如 `psy check` 漂移→1），其他错误→`cli.ExitInternal`（70）；非空 `Msg` 打到 stderr。所有子命令错误均由 cobra 的 `RunE` 返回，cobra 通过 `SilenceErrors: true` 抑制重复输出。
