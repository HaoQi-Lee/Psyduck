# 步骤 1：退出码基础设施

**本步目标：** 引入 `ExitError` 类型与 `ExitCodeFromErr` 辅助函数，重构 `main.go` 与 testscript 入口，使「漂移=1 / 错误=70 / 正常=0」可干净表达。本步不实现 `check`，仅为后续铺路；改动向后兼容（现有 `init`/`version` 行为不变）。

**前置：** 在 `main` 上创建并切换开发分支。

- [ ] **Step 0: 创建开发分支**

```bash
git checkout -b feature/psy-check
git status   # 确认在 feature/psy-check，工作区干净（设计文档已在 main）
```

预期：`On branch feature/psy-check`，无未提交改动。

---

## Task 1.1：`ExitError` + `ExitCodeFromErr`（TDD）

**Files:**
- Modify: `internal/cli/exitcodes.go`
- Test: `internal/cli/exitcodes_test.go`（新建）

- [ ] **Step 1: 写失败测试**

创建 `internal/cli/exitcodes_test.go`：

```go
package cli

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExitCodeFromErr(t *testing.T) {
	tests := []struct {
		name string
		err  error
		code int
		msg  string
	}{
		{"nil is ok", nil, ExitOK, ""},
		{"drift ExitError no msg", &ExitError{Code: ExitDrift}, ExitDrift, ""},
		{"internal ExitError with msg", &ExitError{Code: ExitInternal, Msg: "boom"}, ExitInternal, "boom"},
		{"wrapped ExitError still resolves", fmt.Errorf("wrap: %w", &ExitError{Code: ExitDrift}), ExitDrift, ""},
		{"plain error maps to internal", errors.New("nope"), ExitInternal, "nope"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			code, msg := ExitCodeFromErr(tc.err)
			require.Equal(t, tc.code, code)
			require.Equal(t, tc.msg, msg)
		})
	}
}
```

- [ ] **Step 2: 跑测试，确认失败**

Run: `go test ./internal/cli/ -run TestExitCodeFromErr`
Expected: 编译失败——`ExitDrift`、`ExitError`、`ExitCodeFromErr` 未定义。

- [ ] **Step 3: 写最小实现**

把 `internal/cli/exitcodes.go` 整体替换为：

```go
package cli

import (
	"errors"
	"fmt"
)

// Exit codes used across all psy subcommands.
const (
	ExitOK       = 0
	ExitDrift    = 1  // psy check: drift detected (gate failure)
	ExitInternal = 70
)

// ExitError carries a process exit code distinct from ExitInternal. An empty
// Msg suppresses the "psy: <msg>" line that main otherwise prints to stderr.
// It lets `psy check` report drift (exit 1) without looking like an error
// (exit 70).
type ExitError struct {
	Code int
	Msg  string
}

func (e *ExitError) Error() string {
	if e.Msg != "" {
		return e.Msg
	}
	return fmt.Sprintf("exit code %d", e.Code)
}

// ExitCodeFromErr maps a command error to a process exit code plus an
// optional stderr message.
//   nil             -> (ExitOK, "")
//   *ExitError      -> (Code, Msg)
//   any other error -> (ExitInternal, err.Error())
func ExitCodeFromErr(err error) (int, string) {
	if err == nil {
		return ExitOK, ""
	}
	var ec *ExitError
	if errors.As(err, &ec) {
		return ec.Code, ec.Msg
	}
	return ExitInternal, err.Error()
}
```

- [ ] **Step 4: 跑测试，确认通过**

Run: `go test ./internal/cli/ -run TestExitCodeFromErr -v`
Expected: PASS（5 个子测试全绿）。

- [ ] **Step 5: 提交**

```bash
git add internal/cli/exitcodes.go internal/cli/exitcodes_test.go
git commit -m "refactor(cli): add ExitError and ExitCodeFromErr for multi-code exits"
```

---

## Task 1.2：`main.go` 用 `ExitCodeFromErr`

**Files:**
- Modify: `cmd/psy/main.go`

- [ ] **Step 1: 改写 main.go**

把 `cmd/psy/main.go` 整体替换为：

```go
package main

import (
	"fmt"
	"os"

	"github.com/psyduck/psyduck/internal/cli"
)

func main() {
	root := cli.NewRootCmd(os.Stdout, os.Stderr)
	code, msg := cli.ExitCodeFromErr(root.Execute())
	if msg != "" {
		fmt.Fprintln(os.Stderr, "psy:", msg)
	}
	os.Exit(code)
}
```

- [ ] **Step 2: 验证 build 与既有测试不回归**

Run: `go build ./... && go test ./...`
Expected: build 成功；所有既有测试 PASS（`TestInit_*`、`TestVersionCommand_*`、`TestScripts`）。`init` 仍返回普通 error → 经 `ExitCodeFromErr` 映射为 70，行为不变。

- [ ] **Step 3: 提交**

```bash
git add cmd/psy/main.go
git commit -m "refactor(main): route command errors through ExitCodeFromErr"
```

---

## Task 1.3：testscript 入口同步退出码

**Files:**
- Modify: `internal/cli/script_test.go`

- [ ] **Step 1: 改写 runPsyForTestscript**

把 `internal/cli/script_test.go` 整体替换为：

```go
package cli

import (
	"fmt"
	"os"
	"testing"

	"github.com/rogpeppe/go-internal/testscript"
)

// TestMain wires the `psy` binary into testscript so script fixtures can
// invoke it directly (no shelling out to a separately-built binary).
func TestMain(m *testing.M) {
	os.Exit(testscript.RunMain(m, map[string]func() int{
		"psy": runPsyForTestscript,
	}))
}

// runPsyForTestscript mirrors cmd/psy/main.go but returns an exit code
// instead of calling os.Exit, so testscript can capture it.
func runPsyForTestscript() int {
	root := NewRootCmd(os.Stdout, os.Stderr)
	code, msg := ExitCodeFromErr(root.Execute())
	if msg != "" {
		fmt.Fprintln(os.Stderr, "psy:", msg)
	}
	return code
}

func TestScripts(t *testing.T) {
	testscript.Run(t, testscript.Params{
		Dir: "../../testdata/script",
	})
}
```

- [ ] **Step 2: 验证 testscript E2E 仍通过**

Run: `go test ./internal/cli/ -run TestScripts -v`
Expected: PASS（`init_basic.txt` 的 `! psy init` 仍因 `already initialized` 退出非 0，`stderr` 断言成立）。

- [ ] **Step 3: 提交**

```bash
git add internal/cli/script_test.go
git commit -m "refactor(cli): testscript entry honors ExitCodeFromErr"
```

---

## 步骤 1 验证清单

- [ ] `go build ./...` 成功
- [ ] `go test ./...` 全绿
- [ ] `main` 上无改动（所有改动在 `feature/psy-check`）
- [ ] 现有 `init`/`version` 行为零回归
