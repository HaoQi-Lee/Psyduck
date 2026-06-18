---
psy_kind: factual
psy_version: 1
package: internal/version
created: 2026-06-05
---

# 概述

持有构建时可注入的 `Version` 字符串。独立为极小包，使 `cmd/psy` 和 `internal/cli` 可共同依赖而不会引入其他内部包。

# 文件

- `version.go` — 定义 `Version` 变量。

# API

- `Version string` — CLI 版本号，默认值 `"0.0.0-dev"`，可通过 ldflags `-X github.com/psyduck/psyduck/internal/version.Version=vX.Y.Z` 在构建时覆盖。

# 依赖

- 无。
