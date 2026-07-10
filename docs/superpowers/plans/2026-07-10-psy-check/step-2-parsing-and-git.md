# 步骤 2：SPEC 解析 + git 助手

**本步目标：** 在新包 `internal/spec` 中实现两块纯/薄逻辑——(a) 从 `SPEC.md` 字节流解析出 `package:` 与 `# 文件` 列表；(b) 经 `os/exec` 调系统 `git` 的薄封装 `gitVCS` + 仓库根探测。二者是步骤 3 检测编排的输入。

**包路径：** `github.com/psyduck/psyduck/internal/spec`

---

## Task 2.1：测试工具（临时 git 仓库辅助）

**Files:**
- Test: `internal/spec/testutil_test.go`（新建）

- [ ] **Step 1: 写测试工具**

创建 `internal/spec/testutil_test.go`：

```go
package spec

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// initGitRepo turns dir into a git repo with a dummy author and no GPG
// signing. Fails the test if git is unavailable.
func initGitRepo(t *testing.T, dir string) {
	t.Helper()
	mustGit(t, dir, "init")
	mustGit(t, dir, "config", "user.email", "t@example.com")
	mustGit(t, dir, "config", "user.name", "test")
	mustGit(t, dir, "config", "commit.gpgsign", "false")
}

// commitAllAt stages everything in dir and commits with an explicit commit
// date (ISO 8601, e.g. "2026-07-01T00:00:00") so timing tests are deterministic.
func commitAllAt(t *testing.T, dir, msg, dateISO string) {
	t.Helper()
	mustGit(t, dir, "add", "-A")
	cmd := exec.Command("git", "commit", "-m", msg)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_DATE="+dateISO,
		"GIT_COMMITTER_DATE="+dateISO,
	)
	out, err := cmd.CombinedOutput()
	require.NoErrorf(t, err, "git commit in %s: %s", dir, out)
}

// commitAll is commitAllAt without pinning the date.
func commitAll(t *testing.T, dir, msg string) {
	t.Helper()
	commitAllAt(t, dir, msg, "")
}

func writeFile(t *testing.T, path, body string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(body), 0o644))
}

// writeSpec writes a minimal SPEC.md at <root>/<pkgDir>/SPEC.md with the given
// package: value and # 文件 body. filesBody is inserted verbatim under the
// 文件 heading (pass "" to omit the section entirely).
func writeSpec(t *testing.T, root, pkgDir, pkg, filesBody string) {
	t.Helper()
	var b strings.Builder
	b.WriteString("---\npsy_kind: factual\npsy_version: 1\npackage: " + pkg + "\ncreated: 2026-06-05\n---\n\n# 概述\n\nx\n\n")
	if filesBody == "" {
		// intentionally no # 文件 section
	} else {
		b.WriteString("# 文件\n\n")
		b.WriteString(filesBody)
		b.WriteString("\n")
	}
	writeFile(t, filepath.Join(root, pkgDir, "SPEC.md"), b.String())
}

func mustGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	require.NoErrorf(t, err, "git %s in %s: %s", strings.Join(args, " "), dir, out)
}
```

- [ ] **Step 2: 确认编译（尚无被测代码，仅工具）**

Run: `go vet ./internal/spec/`
Expected: PASS（测试工具文件编译通过；包内暂无非测试 .go 文件属正常，`go vet` 仅校验测试文件语法）。

> 注：此时 `go test ./internal/spec/` 会因包内无非测试文件而报 "no non-test Go files"——下一步 Task 2.2 加入 `spec.go` 后即解决。本步不单独提交工具文件，与 Task 2.2 一起提交。

---

## Task 2.2：SPEC 解析（TDD）

**Files:**
- Create: `internal/spec/spec.go`
- Test: `internal/spec/spec_test.go`（新建）

- [ ] **Step 1: 写失败测试**

创建 `internal/spec/spec_test.go`：

```go
package spec

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParse_Package(t *testing.T) {
	data := []byte("---\npsy_kind: factual\npackage: internal/cli\ncreated: 2026-06-05\n---\n\n# 概述\n\nx\n")
	s := Parse("internal/cli/SPEC.md", data)
	require.True(t, s.HasPackage)
	require.Equal(t, "internal/cli", s.Package)
	require.Equal(t, "internal/cli", s.PkgDir)
}

func TestParse_PackageQuotedAndMissing(t *testing.T) {
	quoted := []byte("---\npackage: \"internal/version\"\n---\n")
	require.Equal(t, "internal/version", Parse("a/b/SPEC.md", quoted).Package)

	none := []byte("---\npsy_kind: factual\n---\n")
	s := Parse("a/SPEC.md", none)
	require.False(t, s.HasPackage)
	require.Equal(t, "", s.Package)
}

func TestParse_FilesSection(t *testing.T) {
	data := []byte("---\npackage: pkg\n---\n\n# 概述\n\nx\n\n# 文件\n\n- `root.go` — 根\n- `claudemd/section.md` — 片段\n\n# API\n\n- x\n")
	s := Parse("pkg/SPEC.md", data)
	require.True(t, s.HasFilesSection)
	require.Equal(t, []string{"root.go", "claudemd/section.md"}, s.Files)
}

func TestParse_FilesSectionEnglishFallback(t *testing.T) {
	data := []byte("---\npackage: pkg\n---\n\n# Files\n\n- `a.go` — a\n")
	s := Parse("pkg/SPEC.md", data)
	require.True(t, s.HasFilesSection)
	require.Equal(t, []string{"a.go"}, s.Files)
}

func TestParse_NoFilesSection(t *testing.T) {
	data := []byte("---\npackage: pkg\n---\n\n# 概述\n\nx\n\n# API\n\nx\n")
	s := Parse("pkg/SPEC.md", data)
	require.False(t, s.HasFilesSection)
	require.Empty(t, s.Files)
}

func TestParse_RootLevelSpecPkgDir(t *testing.T) {
	s := Parse("SPEC.md", []byte("---\npackage: ''\n---\n"))
	require.Equal(t, "", s.PkgDir)
}
```

- [ ] **Step 2: 跑测试，确认失败**

Run: `go test ./internal/spec/ -run TestParse`
Expected: 编译失败——`Parse`、`Spec` 未定义。

- [ ] **Step 3: 写最小实现**

创建 `internal/spec/spec.go`：

```go
// Package spec parses psyduck SPEC.md files and detects drift between a
// spec's declared file list and the files git actually tracks.
package spec

import (
	"bufio"
	"bytes"
	"path/filepath"
	"regexp"
	"strings"
)

// Spec holds the fields parsed from a SPEC.md that psy check cares about.
type Spec struct {
	Path            string   // repo-root-relative path of the SPEC.md
	Package         string   // front-matter package: value
	HasPackage      bool     // whether a package: field was present
	PkgDir          string   // repo-root-relative dir containing the SPEC.md
	Files           []string // # 文件 entries, pkg-dir-relative, slash, cleaned
	HasFilesSection bool     // whether a # 文件 / # Files heading was found
}

var packageRe = regexp.MustCompile(`(?m)^\s*package:\s*(.+?)\s*$`)

// Parse extracts the package front-matter value and the # 文件 file list from a
// SPEC.md. path is the repo-root-relative path (used to derive PkgDir).
// Parsing is intentionally line-based — only package: and file bullets are
// needed, so no YAML dependency.
func Parse(path string, data []byte) Spec {
	s := Spec{Path: path, PkgDir: dirOf(path)}
	if m := packageRe.FindSubmatch(data); m != nil {
		s.HasPackage = true
		s.Package = trimQuotes(string(m[1]))
	}
	s.Files, s.HasFilesSection = parseFilesSection(data)
	return s
}

func dirOf(path string) string {
	if i := strings.LastIndexByte(path, '/'); i >= 0 {
		return path[:i]
	}
	return ""
}

func trimQuotes(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 {
		first, last := s[0], s[len(s)-1]
		if (first == '"' && last == '"') || (first == '\'' && last == '\'') {
			return s[1 : len(s)-1]
		}
	}
	return s
}

// parseFilesSection scans for a # 文件 / # Files heading and collects the
// backtick-quoted path of each "- " bullet until the next heading.
func parseFilesSection(data []byte) (files []string, found bool) {
	sc := bufio.NewScanner(bytes.NewReader(data))
	inSection := false
	seen := map[string]bool{}
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if isHeading(line) {
			if inSection {
				return files, found // next heading ends the section
			}
			if isFilesHeading(line) {
				inSection = true
				found = true
			}
			continue
		}
		if inSection && strings.HasPrefix(line, "- ") {
			if p := bulletPath(line); p != "" {
				clean := filepath.ToSlash(filepath.Clean(p))
				if !seen[clean] {
					seen[clean] = true
					files = append(files, clean)
				}
			}
		}
	}
	return files, found
}

func isHeading(line string) bool { return strings.HasPrefix(line, "#") }

func isFilesHeading(line string) bool {
	if !isHeading(line) {
		return false
	}
	t := strings.TrimSpace(strings.TrimLeft(line, "#"))
	return t == "文件" || t == "Files"
}

// bulletPath extracts the first backtick-quoted token from a bullet line like
// "- `root.go` — desc". Returns "" if none.
func bulletPath(line string) string {
	i := strings.IndexByte(line, '`')
	if i < 0 {
		return ""
	}
	rest := line[i+1:]
	j := strings.IndexByte(rest, '`')
	if j < 0 {
		return ""
	}
	return rest[:j]
}
```

- [ ] **Step 4: 跑测试，确认通过**

Run: `go test ./internal/spec/ -run TestParse -v`
Expected: PASS（6 个测试全绿）。

- [ ] **Step 5: 提交**

```bash
git add internal/spec/spec.go internal/spec/spec_test.go internal/spec/testutil_test.go
git commit -m "feat(spec): parse package front-matter and # 文件 section"
```

---

## Task 2.3：`gitVCS` + `RepoRoot`（TDD）

**Files:**
- Create: `internal/spec/git.go`
- Test: `internal/spec/git_test.go`（新建）

- [ ] **Step 1: 写失败测试**

创建 `internal/spec/git_test.go`：

```go
package spec

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRepoRoot_NotARepo(t *testing.T) {
	dir := t.TempDir()
	wd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(dir))
	t.Cleanup(func() { os.Chdir(wd) })

	_, err = RepoRoot()
	require.Error(t, err)
}

func TestGitVCS_ListFilesAndTimes(t *testing.T) {
	root := t.TempDir()
	initGitRepo(t, root)
	writeFile(t, filepath.Join(root, "a.go"), "package p\n")
	writeFile(t, filepath.Join(root, "pkg", "b.go"), "package pkg\n")
	commitAllAt(t, root, "first", "2026-06-05T00:00:00")

	v := newGitVCS(root)

	all, err := v.ListFiles("")
	require.NoError(t, err)
	require.Equal(t, []string{"a.go", "pkg/b.go"}, all)

	sub, err := v.ListFiles("pkg")
	require.NoError(t, err)
	require.Equal(t, []string{"pkg/b.go"}, sub)

	ts, ok, err := v.LastCommitTime("a.go")
	require.NoError(t, err)
	require.True(t, ok)
	require.False(t, ts.IsZero())

	_, ok, err = v.LastCommitTime("never.go")
	require.NoError(t, err)
	require.False(t, ok) // no commit history
}
```

- [ ] **Step 2: 跑测试，确认失败**

Run: `go test ./internal/spec/ -run 'TestRepoRoot|TestGitVCS'`
Expected: 编译失败——`RepoRoot`、`newGitVCS` 未定义。

- [ ] **Step 3: 写最小实现**

创建 `internal/spec/git.go`：

```go
package spec

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// VCS is the source-control seam used by Check. Methods take repo-root-
// relative paths. Production impl is gitVCS; tests inject fakes.
type VCS interface {
	// ListFiles returns git-tracked files under relDir ("" = whole repo),
	// repo-root-relative and slash-separated.
	ListFiles(relDir string) ([]string, error)
	// LastCommitTime returns the latest commit time touching relPath.
	// ok is false when the path has no commit history.
	LastCommitTime(relPath string) (t time.Time, ok bool, err error)
}

// RepoRoot returns the absolute repository root of the current working
// directory via `git rev-parse --show-toplevel`.
func RepoRoot() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("not a git repository (or git unavailable): %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

type gitVCS struct{ root string }

// newGitVCS builds a VCS backed by system git, rooted at root.
func newGitVCS(root string) VCS { return &gitVCS{root: root} }

func (g *gitVCS) ListFiles(relDir string) ([]string, error) {
	args := []string{"ls-files"}
	if relDir != "" {
		args = append(args, relDir)
	}
	out, err := runGit(g.root, args...)
	if err != nil {
		return nil, err
	}
	var files []string
	for _, line := range strings.Split(string(out), "\n") {
		if line = strings.TrimSpace(line); line != "" {
			files = append(files, filepath.ToSlash(line))
		}
	}
	return files, nil
}

func (g *gitVCS) LastCommitTime(relPath string) (time.Time, bool, error) {
	out, err := runGit(g.root, "log", "-1", "--format=%ct", "--", relPath)
	if err != nil {
		return time.Time{}, false, err
	}
	s := strings.TrimSpace(string(out))
	if s == "" {
		return time.Time{}, false, nil
	}
	sec, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return time.Time{}, false, fmt.Errorf("parse %q: %w", s, err)
	}
	return time.Unix(sec, 0), true, nil
}

// runGit runs git in repoRoot and returns stdout.
func runGit(repoRoot string, args ...string) ([]byte, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = repoRoot
	out, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok && len(ee.Stderr) > 0 {
			return nil, fmt.Errorf("git %s: %s", strings.Join(args, " "), strings.TrimSpace(string(ee.Stderr)))
		}
		return nil, fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return out, nil
}
```

- [ ] **Step 4: 跑测试，确认通过**

Run: `go test ./internal/spec/ -run 'TestRepoRoot|TestGitVCS' -v`
Expected: PASS。

- [ ] **Step 5: 提交**

```bash
git add internal/spec/git.go internal/spec/git_test.go
git commit -m "feat(spec): add VCS interface and gitVCS implementation"
```

---

## 步骤 2 验证清单

- [ ] `go build ./...` 成功
- [ ] `go test ./internal/spec/` 全绿
- [ ] `go test ./...` 全绿（无回归）
- [ ] 未引入新第三方依赖（`go.mod` 不变）
