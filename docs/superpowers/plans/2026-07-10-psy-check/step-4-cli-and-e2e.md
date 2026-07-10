# 步骤 4：CLI 命令 + 渲染 + E2E

**本步目标：** 把 `spec.Check` 接入 cobra——`psy check`（零参数零 flag），渲染 `Report` 为纯文本，漂移时返回 `&ExitError{Code: ExitDrift}`；注册到根命令；补 testscript E2E 与 README。

**前置：** 步骤 3 完成（`spec.Check` 就绪）。

---

## Task 4.1：`renderReport`（纯函数，TDD）

**Files:**
- Create: `internal/cli/check.go`
- Test: `internal/cli/check_test.go`（新建）

- [ ] **Step 1: 写失败测试**

创建 `internal/cli/check_test.go`：

```go
package cli

import (
	"bytes"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/psyduck/psyduck/internal/spec"
)

func TestRenderReport_DriftAndTiming(t *testing.T) {
	var b bytes.Buffer
	renderReport(&b, spec.Report{
		Root: "/r",
		Packages: []spec.PackageReport{
			{
				Package: "pkg", SpecPath: "pkg/SPEC.md", PkgDir: "pkg",
				ListedButGone: []string{"old.go"},
				Undocumented:  []string{"new.go"},
				Timing: []spec.TimingHint{{
					File:     "root.go",
					FileTime: time.Unix(1_750_000_000, 0),
					SpecTime: time.Unix(1_740_000_000, 0),
				}},
			},
			{Package: "clean", SpecPath: "clean/SPEC.md", PkgDir: "clean"},
		},
	})
	out := b.String()
	require.Contains(t, out, "✗ 结构漂移")
	require.Contains(t, out, "- 已过期:   old.go")
	require.Contains(t, out, "+ 未文档化: new.go")
	require.Contains(t, out, "⚠ 时序提示")
	require.Contains(t, out, "root.go 比 SPEC 新")
	require.Contains(t, out, "✓ 无漂移")
	require.Contains(t, out, "发现: 1 处结构漂移, 1 条时序提示")
}
```

- [ ] **Step 2: 跑测试，确认失败**

Run: `go test ./internal/cli/ -run TestRenderReport`
Expected: 编译失败——`renderReport` 未定义。

- [ ] **Step 3: 写最小实现**

创建 `internal/cli/check.go`：

```go
package cli

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/psyduck/psyduck/internal/spec"
)

func newCheckCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "check",
		Short: "Check SPEC.md files for drift against the repo (read-only gate)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCheck(cmd)
		},
	}
}

func runCheck(cmd *cobra.Command) error {
	root, err := spec.RepoRoot()
	if err != nil {
		return &ExitError{Code: ExitInternal, Msg: err.Error()}
	}
	rep, err := spec.Check(root)
	if err != nil {
		return &ExitError{Code: ExitInternal, Msg: err.Error()}
	}
	renderReport(cmd.OutOrStdout(), rep)
	if rep.DriftCount() > 0 {
		return &ExitError{Code: ExitDrift}
	}
	return nil
}

// renderReport writes a plain-text, grep-friendly summary. No ANSI colors.
func renderReport(w io.Writer, r spec.Report) {
	fmt.Fprintf(w, "psy check — 检查 SPEC.md 与现实的漂移\n\n")
	for _, p := range r.Packages {
		label := p.PkgDir
		if label == "" {
			label = "(repo root)"
		}
		fmt.Fprintf(w, "%s  (SPEC.md)\n", label)
		if p.HasDrift() {
			fmt.Fprintln(w, "  ✗ 结构漂移")
			for _, f := range p.ListedButGone {
				fmt.Fprintf(w, "      - 已过期:   %s   （SPEC 列了，目录没有）\n", f)
			}
			for _, f := range p.Undocumented {
				fmt.Fprintf(w, "      + 未文档化: %s   （目录有，SPEC 未列）\n", f)
			}
			if p.PackageMismatch {
				fmt.Fprintln(w, "      ! package 路径与实际位置不符")
			}
			if p.MissingFileSection {
				fmt.Fprintln(w, "      ! 缺少 # 文件 章节")
			}
		} else {
			fmt.Fprintln(w, "  ✓ 无漂移")
		}
		if len(p.Timing) > 0 {
			fmt.Fprintln(w, "  ⚠ 时序提示")
			for _, th := range p.Timing {
				days := int(th.FileTime.Sub(th.SpecTime).Hours() / 24)
				fmt.Fprintf(w, "      %s 比 SPEC 新 %d 天（spec 可能过期）\n", th.File, days)
			}
		}
		fmt.Fprintln(w)
	}
	fmt.Fprintf(w, "发现: %d 处结构漂移, %d 条时序提示\n", r.DriftCount(), r.TimingCount())
}
```

- [ ] **Step 4: 跑测试，确认通过**

Run: `go test ./internal/cli/ -run TestRenderReport -v`
Expected: PASS。

- [ ] **Step 5: 提交**

```bash
git add internal/cli/check.go internal/cli/check_test.go
git commit -m "feat(cli): add psy check command and plain-text renderer"
```

---

## Task 4.2：注册到根命令 + 退出码接线测试（TDD）

**Files:**
- Modify: `internal/cli/root.go`
- Modify: `internal/cli/check_test.go`（追加 git 辅助 + runCheck 测试）

- [ ] **Step 1: 注册命令**

在 `internal/cli/root.go` 的 `NewRootCmd` 中，于 `root.AddCommand(newInitCmd())` 之后追加一行：

```go
	root.AddCommand(newCheckCmd())
```

（最终 `NewRootCmd` 末尾为：）
```go
	root.AddCommand(newVersionCmd())
	root.AddCommand(newInitCmd())
	root.AddCommand(newCheckCmd())
	return root
```

- [ ] **Step 2: 追加 runCheck 失败测试（真 git 仓库）**

在 `internal/cli/check_test.go` 顶部 import 块加入：
```go
	"os/exec"
	"path/filepath"
	"strings"
```
然后在文件末尾追加：

```go
func cliGitInit(t *testing.T, dir string) {
	t.Helper()
	cliGit(t, dir, "init")
	cliGit(t, dir, "config", "user.email", "t@example.com")
	cliGit(t, dir, "config", "user.name", "t")
	cliGit(t, dir, "config", "commit.gpgsign", "false")
}

func cliCommitAll(t *testing.T, dir, msg string) {
	t.Helper()
	cliGit(t, dir, "add", "-A")
	cliGit(t, dir, "commit", "-m", msg)
}

func cliGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	require.NoErrorf(t, err, "git %s: %s", strings.Join(args, " "), out)
}

const cliSpecBody = "---\npsy_kind: factual\npsy_version: 1\npackage: pkg\ncreated: 2026-06-05\n---\n\n# 概述\n\nx\n\n# 文件\n\n- `root.go` — r\n- `old.go` — gone\n"

func TestRunCheck_ExitDriftOnDrift(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)
	cliGitInit(t, dir)
	mustWriteFile(t, filepath.Join(dir, "pkg", "root.go"), "package pkg\n")
	mustWriteFile(t, filepath.Join(dir, "pkg", "new.go"), "package pkg\n")
	mustWriteFile(t, filepath.Join(dir, "pkg", "SPEC.md"), cliSpecBody)
	cliCommitAll(t, dir, "init")

	var out bytes.Buffer
	root := NewRootCmd(&out, &out)
	root.SetArgs([]string{"check"})
	err := root.Execute()
	require.Error(t, err)
	code, _ := ExitCodeFromErr(err)
	require.Equal(t, ExitDrift, code)
	require.Contains(t, out.String(), "结构漂移")
}

func TestRunCheck_ExitOKWhenClean(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)
	cliGitInit(t, dir)
	mustWriteFile(t, filepath.Join(dir, "pkg", "root.go"), "package pkg\n")
	mustWriteFile(t, filepath.Join(dir, "pkg", "SPEC.md"),
		"---\npackage: pkg\n---\n\n# 文件\n\n- `root.go` — r\n")
	cliCommitAll(t, dir, "init")

	var out bytes.Buffer
	root := NewRootCmd(&out, &out)
	root.SetArgs([]string{"check"})
	require.NoError(t, root.Execute())
	require.Contains(t, out.String(), "无漂移")
}
```

> `chdir` 与 `mustWriteFile` 已存在于 `internal/cli/testutil_test.go`（同包），可直接使用。

- [ ] **Step 3: 跑测试，确认通过**

Run: `go test ./internal/cli/ -run TestRunCheck -v`
Expected: PASS（漂移→ExitDrift，干净→nil）。

- [ ] **Step 4: 提交**

```bash
git add internal/cli/root.go internal/cli/check_test.go
git commit -m "feat(cli): register psy check and assert exit codes"
```

---

## Task 4.3：testscript E2E

**Files:**
- Create: `testdata/script/check_clean.txt`
- Create: `testdata/script/check_drift.txt`
- Create: `testdata/script/check_stale.txt`

> 输出全部在 **stdout**（漂移行也走 stdout；ExitError.Msg 为空不打 stderr）。

- [ ] **Step 1: 写 check_clean.txt**

创建 `testdata/script/check_clean.txt`：

```
# clean: declared set matches tracked files -> exit 0
exec git init
exec git config user.email t@example.com
exec git config user.name t
exec git config commit.gpgsign false

mkdir pkg
cat >pkg/root.go <<EOF
package pkg
EOF
cat >pkg/SPEC.md <<EOF
---
package: pkg
---

# 文件

- `root.go` — r
EOF
exec git add -A
exec git commit -m init

psy check
stdout '无漂移'
```

- [ ] **Step 2: 写 check_drift.txt**

创建 `testdata/script/check_drift.txt`：

```
# drift: old.go listed but gone, new.go undocumented -> exit 1
exec git init
exec git config user.email t@example.com
exec git config user.name t
exec git config commit.gpgsign false

mkdir pkg
cat >pkg/root.go <<EOF
package pkg
EOF
cat >pkg/new.go <<EOF
package pkg
EOF
cat >pkg/SPEC.md <<EOF
---
package: pkg
---

# 文件

- `root.go` — r
- `old.go` — gone
EOF
exec git add -A
exec git commit -m init

! psy check
stdout '已过期'
stdout 'old.go'
stdout '未文档化'
```

- [ ] **Step 3: 写 check_stale.txt**

创建 `testdata/script/check_stale.txt`：

```
# stale: source newer than spec -> hint printed, exit 0 (timing never fails)
exec git init
exec git config user.email t@example.com
exec git config user.name t
exec git config commit.gpgsign false

mkdir pkg
cat >pkg/root.go <<EOF
package pkg
EOF
cat >pkg/SPEC.md <<EOF
---
package: pkg
---

# 文件

- `root.go` — r
EOF
exec git add -A
env GIT_AUTHOR_DATE=2026-06-05T00:00:00
env GIT_COMMITTER_DATE=2026-06-05T00:00:00
exec git commit -m init

cat >pkg/root.go <<EOF
package pkg
// changed
EOF
exec git add -A
env GIT_AUTHOR_DATE=2026-07-01T00:00:00
env GIT_COMMITTER_DATE=2026-07-01T00:00:00
exec git commit -m update

psy check
stdout '时序提示'
stdout 'root.go 比 SPEC 新'
```

- [ ] **Step 4: 跑 E2E，确认通过**

Run: `go test ./internal/cli/ -run TestScripts -v`
Expected: PASS（`init_basic.txt` + 三个 `check_*.txt`）。

- [ ] **Step 5: 提交**

```bash
git add testdata/script/check_clean.txt testdata/script/check_drift.txt testdata/script/check_stale.txt
git commit -m "test(cli): add psy check testscript E2E (clean/drift/stale)"
```

---

## Task 4.4：README + 全量验证

**Files:**
- Modify: `README.md`

- [ ] **Step 1: 在 README 增加 `psy check` 小节**

在 `README.md` 的 `### psy version` 小节之后、`## Skills` 之前，插入：

```markdown
### `psy check`

只读检测每个 `SPEC.md` 与仓库现实的漂移，作为 CI / pre-commit 门禁。零参数、零 flag。

- **结构漂移**（退出码 1，失败门禁）：`# 文件` 章节列出的文件集合与 `git ls-files` 跟踪的实际文件集合不一致（多/少），或 `package:` 路径与实际位置不符，或缺少 `# 文件` 章节。
- **时序提示**（仅打印，退出码 0）：源文件最新提交比 `SPEC.md` 新，提示 spec 可能过期。

```bash
psy check
# 发现: 0 处结构漂移, 0 条时序提示   （退出码 0）
```

| 退出码 | 含义 |
|---|---|
| `0` | 无结构漂移（时序提示不影响） |
| `1` | 检测到结构漂移 |
| `70` | 内部错误（非 git 仓库 / git 缺失 / 解析异常） |

> `psy check` 只读：不写文件、不联网、不调 LLM。对账但重写请用 `/psy-sync-all`。
```

- [ ] **Step 2: 全量构建与测试**

Run: `make build && make test && make lint`
Expected: build 成功产出 `bin/psy`；`go test ./...` 全绿（含新增 spec 包与 check 测试）；`go vet ./...` 无告警。

- [ ] **Step 3: 手动冒烟（可选，在仓库自身上）**

Run: `./bin/psy check`（或 `go run ./cmd/psy check`）
Expected: 报告当前仓库各包；因 `internal/spec` 等新文件尚未写入对应 `SPEC.md`，可能显示时序/漂移——属预期（下一步 `/psy-sync` 会补齐）。

- [ ] **Step 4: 提交**

```bash
git add README.md
git commit -m "docs: document psy check in README"
```

---

## 步骤 4 验证清单

- [ ] `make build` 成功
- [ ] `make test` 全绿
- [ ] `make lint` 无告警
- [ ] `psy check` 已注册（`psy --help` 可见）
- [ ] E2E 覆盖 clean（0）/ drift（1）/ stale（0 + 提示）三种退出码路径

## 收尾（MANDATORY）

所有步骤完成后，按 CLAUDE.md 执行 `/psy-sync`：为本次变更集涉及的代码包生成/更新 `SPEC.md`（至少 `internal/cli`——新增 `check.go`；以及新包 `internal/spec`），并把本计划目录与设计文档归档到 `.psy/2026-07-11/`。
