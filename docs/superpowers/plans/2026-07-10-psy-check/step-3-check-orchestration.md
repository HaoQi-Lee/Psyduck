# 步骤 3：检测编排（结构漂移 + 时序）

**本步目标：** 在 `internal/spec` 实现 `Check`——发现仓库内所有 `SPEC.md`，比对声明集（`# 文件`）与实际集（`git ls-files`），产出结构漂移与时序提示，返回结构化 `Report`。核心逻辑 `checkWith` 用 fake VCS 单测（不依赖真 git）。

**前置：** 步骤 2 完成（`Spec`/`Parse`/`VCS`/`gitVCS` 就绪）。

---

## Task 3.1：`Report` 类型与计数方法（TDD）

**Files:**
- Create: `internal/spec/check.go`
- Test: `internal/spec/check_test.go`（新建）

- [ ] **Step 1: 写失败测试**

创建 `internal/spec/check_test.go`：

```go
package spec

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestReport_Counts(t *testing.T) {
	r := Report{Packages: []PackageReport{
		{ListedButGone: []string{"a"}}, // drift
		{Timing: []TimingHint{{}}},     // timing only, no drift
		{},                             // clean
	}}
	require.Equal(t, 1, r.DriftCount())
	require.Equal(t, 1, r.TimingCount())
}

func TestPackageReport_HasDrift(t *testing.T) {
	require.True(t, PackageReport{PackageMismatch: true}.HasDrift())
	require.True(t, PackageReport{MissingFileSection: true}.HasDrift())
	require.True(t, PackageReport{ListedButGone: []string{"x"}}.HasDrift())
	require.True(t, PackageReport{Undocumented: []string{"x"}}.HasDrift())
	require.False(t, PackageReport{}.HasDrift())
	require.False(t, PackageReport{Timing: []TimingHint{{}}}.HasDrift())
}

// keep the time import meaningful until timing tests arrive in 3.3
var _ = time.Unix
```

- [ ] **Step 2: 跑测试，确认失败**

Run: `go test ./internal/spec/ -run 'TestReport_Counts|TestPackageReport_HasDrift'`
Expected: 编译失败——`Report`/`PackageReport`/`TimingHint`/方法未定义。

- [ ] **Step 3: 写最小实现**

创建 `internal/spec/check.go`：

```go
package spec

import "time"

// Report is the result of checking a whole repository.
type Report struct {
	Root     string
	Packages []PackageReport
}

// PackageReport is the result of checking one SPEC.md.
type PackageReport struct {
	Package            string       // front-matter package: value
	SpecPath           string       // repo-root-relative SPEC.md path
	PkgDir             string       // repo-root-relative dir of the SPEC.md
	PackageMismatch    bool         // package: value != PkgDir (or absent)
	MissingFileSection bool         // no # 文件 heading found
	ListedButGone      []string     // declared but not on disk (pkg-relative)
	Undocumented       []string     // on disk but not declared (pkg-relative)
	Timing             []TimingHint // source newer than spec (advisory)
}

// TimingHint flags a source file whose latest commit is newer than the spec's.
type TimingHint struct {
	File     string
	FileTime time.Time
	SpecTime time.Time
}

// HasDrift reports any structural drift (excludes timing, which is advisory).
func (p PackageReport) HasDrift() bool {
	return p.PackageMismatch || p.MissingFileSection ||
		len(p.ListedButGone) > 0 || len(p.Undocumented) > 0
}

// DriftCount is the number of packages with structural drift.
func (r Report) DriftCount() int {
	n := 0
	for _, p := range r.Packages {
		if p.HasDrift() {
			n++
		}
	}
	return n
}

// TimingCount is the total number of timing hints across all packages.
func (r Report) TimingCount() int {
	n := 0
	for _, p := range r.Packages {
		n += len(p.Timing)
	}
	return n
}
```

- [ ] **Step 4: 跑测试，确认通过**

Run: `go test ./internal/spec/ -run 'TestReport_Counts|TestPackageReport_HasDrift' -v`
Expected: PASS。

- [ ] **Step 5: 提交**

```bash
git add internal/spec/check.go internal/spec/check_test.go
git commit -m "feat(spec): add Report types and drift/timing counters"
```

---

## Task 3.2：结构漂移检测（TDD）

**Files:**
- Modify: `internal/spec/check.go`（追加 `Check`/`checkWith`/`checkPackage`/helpers）
- Modify: `internal/spec/check_test.go`（追加 fakeVCS + 结构测试）

- [ ] **Step 1: 追加失败测试（结构漂移）**

在 `internal/spec/check_test.go` 顶部 import 块加入 `"path/filepath"`（若尚无），并删除 Task 3.1 临时加的 `var _ = time.Unix` 行（time 已被 TimingHint 使用）。然后在文件末尾追加：

```go
// fakeVCS is an in-memory VCS for testing checkWith without real git.
type fakeVCS struct {
	files map[string][]string  // relDir -> files (repo-root-relative)
	times map[string]time.Time // relPath -> time (absent = no history)
}

func (f *fakeVCS) ListFiles(relDir string) ([]string, error) {
	return f.files[relDir], nil
}
func (f *fakeVCS) LastCommitTime(relPath string) (time.Time, bool, error) {
	t, ok := f.times[relPath]
	return t, ok, nil
}

func TestCheck_StructuralDrift(t *testing.T) {
	root := t.TempDir()
	writeSpec(t, root, "pkg", "pkg", "- `root.go` — r\n- `old.go` — gone\n")
	writeFile(t, filepath.Join(root, "pkg", "root.go"), "package pkg\n")
	writeFile(t, filepath.Join(root, "pkg", "new.go"), "package pkg\n")

	v := &fakeVCS{files: map[string][]string{
		"":     {"pkg/SPEC.md"},
		"pkg":  {"pkg/SPEC.md", "pkg/root.go", "pkg/new.go"},
	}}
	rep, err := checkWith(root, v)
	require.NoError(t, err)
	require.Len(t, rep.Packages, 1)
	pr := rep.Packages[0]
	require.Equal(t, []string{"old.go"}, pr.ListedButGone)
	require.Equal(t, []string{"new.go"}, pr.Undocumented)
	require.True(t, pr.HasDrift())
	require.Equal(t, 1, rep.DriftCount())
}

func TestCheck_CleanPackage(t *testing.T) {
	root := t.TempDir()
	writeSpec(t, root, "pkg", "pkg", "- `root.go` — r\n")
	writeFile(t, filepath.Join(root, "pkg", "root.go"), "package pkg\n")
	v := &fakeVCS{files: map[string][]string{
		"":    {"pkg/SPEC.md"},
		"pkg": {"pkg/SPEC.md", "pkg/root.go"},
	}}
	rep, err := checkWith(root, v)
	require.NoError(t, err)
	require.Len(t, rep.Packages, 1)
	require.False(t, rep.Packages[0].HasDrift())
}

func TestCheck_NestedPackageExcluded(t *testing.T) {
	root := t.TempDir()
	writeSpec(t, root, "pkg", "pkg", "- `parent.go` — p\n")
	writeFile(t, filepath.Join(root, "pkg", "parent.go"), "package pkg\n")
	writeSpec(t, root, "pkg/child", "pkg/child", "- `child.go` — c\n")
	writeFile(t, filepath.Join(root, "pkg", "child", "child.go"), "package child\n")
	v := &fakeVCS{files: map[string][]string{
		"":          {"pkg/SPEC.md", "pkg/child/SPEC.md"},
		"pkg":       {"pkg/SPEC.md", "pkg/parent.go", "pkg/child/SPEC.md", "pkg/child/child.go"},
		"pkg/child": {"pkg/child/SPEC.md", "pkg/child/child.go"},
	}}
	rep, err := checkWith(root, v)
	require.NoError(t, err)
	require.Len(t, rep.Packages, 2)
	byDir := map[string]PackageReport{}
	for _, p := range rep.Packages {
		byDir[p.PkgDir] = p
	}
	require.False(t, byDir["pkg"].HasDrift(), "parent must not see child files")
	require.Empty(t, byDir["pkg"].Undocumented)
	require.False(t, byDir["pkg/child"].HasDrift())
}

func TestCheck_PackageMismatch(t *testing.T) {
	root := t.TempDir()
	writeSpec(t, root, "pkg", "wrong/path", "- `root.go` — r\n")
	writeFile(t, filepath.Join(root, "pkg", "root.go"), "package pkg\n")
	v := &fakeVCS{files: map[string][]string{"": {"pkg/SPEC.md"}, "pkg": {"pkg/SPEC.md", "pkg/root.go"}}}
	rep, err := checkWith(root, v)
	require.NoError(t, err)
	require.True(t, rep.Packages[0].PackageMismatch)
}

func TestCheck_MissingFilesSection(t *testing.T) {
	root := t.TempDir()
	writeSpec(t, root, "pkg", "pkg", "") // no # 文件
	writeFile(t, filepath.Join(root, "pkg", "root.go"), "package pkg\n")
	v := &fakeVCS{files: map[string][]string{"": {"pkg/SPEC.md"}, "pkg": {"pkg/SPEC.md", "pkg/root.go"}}}
	rep, err := checkWith(root, v)
	require.NoError(t, err)
	require.True(t, rep.Packages[0].MissingFileSection)
	require.Empty(t, rep.Packages[0].ListedButGone)
	require.Empty(t, rep.Packages[0].Undocumented)
}
```

- [ ] **Step 2: 跑测试，确认失败**

Run: `go test ./internal/spec/ -run TestCheck_`
Expected: 编译失败——`checkWith` 未定义。

- [ ] **Step 3: 追加结构漂移实现（尚不含时序）**

在 `internal/spec/check.go` 末尾追加：

```go
import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Check scans repoRoot for every SPEC.md and reports drift against git. It is
// read-only. Equivalent to checkWith with a real gitVCS.
func Check(repoRoot string) (Report, error) {
	return checkWith(repoRoot, newGitVCS(repoRoot))
}

// checkWith is the testable core; vcs is injected so logic can be exercised
// without a real git repository.
func checkWith(repoRoot string, vcs VCS) (Report, error) {
	rep := Report{Root: repoRoot}
	all, err := vcs.ListFiles("")
	if err != nil {
		return rep, err
	}
	var specPaths []string
	specDirs := map[string]bool{}
	for _, f := range all {
		if path.Base(f) == "SPEC.md" {
			specPaths = append(specPaths, f)
			specDirs[path.Dir(f)] = true
		}
	}
	for _, sp := range specPaths {
		pr, err := checkPackage(repoRoot, sp, vcs, specDirs)
		if err != nil {
			return rep, err
		}
		rep.Packages = append(rep.Packages, pr)
	}
	sort.Slice(rep.Packages, func(i, j int) bool {
		return rep.Packages[i].SpecPath < rep.Packages[j].SpecPath
	})
	return rep, nil
}

func checkPackage(repoRoot, specPath string, vcs VCS, specDirs map[string]bool) (PackageReport, error) {
	data, err := os.ReadFile(filepath.Join(repoRoot, filepath.FromSlash(specPath)))
	if err != nil {
		return PackageReport{}, fmt.Errorf("read %s: %w", specPath, err)
	}
	sp := Parse(specPath, data)
	pr := PackageReport{
		Package:  sp.Package,
		SpecPath: specPath,
		PkgDir:   sp.PkgDir,
	}
	if !sp.HasPackage || sp.Package != sp.PkgDir {
		pr.PackageMismatch = true
	}

	actualRel, err := actualFiles(sp.PkgDir, specPath, specDirs, vcs)
	if err != nil {
		return pr, err
	}

	if sp.HasFilesSection {
		declared := toSet(sp.Files)
		for f := range declared {
			if !actualRel[f] {
				pr.ListedButGone = append(pr.ListedButGone, f)
			}
		}
		for f := range actualRel {
			if !declared[f] {
				pr.Undocumented = append(pr.Undocumented, f)
			}
		}
		sort.Strings(pr.ListedButGone)
		sort.Strings(pr.Undocumented)
	} else {
		pr.MissingFileSection = true
	}
	return pr, nil
}

// actualFiles returns the set of pkg-dir-relative files tracked under pkgDir,
// excluding the SPEC.md itself and nested-package subtrees.
func actualFiles(pkgDir, specPath string, specDirs map[string]bool, vcs VCS) (map[string]bool, error) {
	listed, err := vcs.ListFiles(pkgDir)
	if err != nil {
		return nil, err
	}
	pkgPrefix := pkgDir
	if pkgPrefix != "" {
		pkgPrefix += "/"
	}
	out := map[string]bool{}
	for _, f := range listed {
		if f == specPath {
			continue
		}
		if inNestedSpecDir(f, specPath, specDirs) {
			continue
		}
		out[strings.TrimPrefix(f, pkgPrefix)] = true
	}
	return out, nil
}

func toSet(xs []string) map[string]bool {
	m := map[string]bool{}
	for _, x := range xs {
		m[x] = true
	}
	return m
}

// inNestedSpecDir reports whether file f lives under a proper subdirectory of
// specPath's dir that itself contains a SPEC.md (a nested package).
func inNestedSpecDir(f, specPath string, specDirs map[string]bool) bool {
	specDir := path.Dir(specPath)
	d := path.Dir(f)
	for d != specDir && d != "." && d != "" {
		if specDirs[d] {
			return true
		}
		parent := path.Dir(d)
		if parent == d {
			break
		}
		d = parent
	}
	return false
}
```

> 注意：`check.go` 现在有两个 import 块会冲突。把 Step 3 的 `import (...)` 合并进文件顶部已有的 `import "time"` 块，得到单一 import 块：
> ```go
> import (
> 	"fmt"
> 	"os"
> 	"path"
> 	"path/filepath"
> 	"sort"
> 	"strings"
> 	"time"
> )
> ```
> 即：编辑 `check.go`，将顶部 `import "time"` 替换为上面的合并块，并删除末尾追加段落里的 `import (...)`。
>
> **跨平台要点（本仓库跑在 Windows）：** git 返回的路径永远是**正斜杠**。`filepath.Base`/`filepath.Dir` 按 OS 分隔符切分，在 Windows 上会把 `"pkg/SPEC.md"` 整体当作 basename——所以对 git 返回的字符串必须用 `path` 包（OS 无关、按 `/` 切）。`filepath` 仅保留给 `os.ReadFile` 前 `filepath.FromSlash(specPath)` 把正斜杠转回 OS 分隔符这一处。

- [ ] **Step 4: 跑测试，确认通过**

Run: `go test ./internal/spec/ -run TestCheck_ -v`
Expected: PASS（5 个结构测试全绿）。

- [ ] **Step 5: 提交**

```bash
git add internal/spec/check.go internal/spec/check_test.go
git commit -m "feat(spec): detect structural drift (declared vs git-tracked files)"
```

---

## Task 3.3：时序提示（TDD）

**Files:**
- Modify: `internal/spec/check.go`（在 `checkPackage` 末尾 `return pr, nil` 前插入时序块；加 `sortedKeys` helper）
- Modify: `internal/spec/check_test.go`（追加时序测试）

- [ ] **Step 1: 追加失败测试（时序）**

在 `internal/spec/check_test.go` 末尾追加：

```go
func TestCheck_TimingHint(t *testing.T) {
	root := t.TempDir()
	writeSpec(t, root, "pkg", "pkg", "- `root.go` — r\n")
	writeFile(t, filepath.Join(root, "pkg", "root.go"), "package pkg\n")
	specT := time.Unix(1_700_000_000, 0) // older
	fileT := time.Unix(1_710_000_000, 0) // newer
	v := &fakeVCS{
		files: map[string][]string{"": {"pkg/SPEC.md"}, "pkg": {"pkg/SPEC.md", "pkg/root.go"}},
		times: map[string]time.Time{"pkg/SPEC.md": specT, "pkg/root.go": fileT},
	}
	rep, err := checkWith(root, v)
	require.NoError(t, err)
	require.Len(t, rep.Packages[0].Timing, 1)
	require.Equal(t, "root.go", rep.Packages[0].Timing[0].File)
	require.Equal(t, 1, rep.TimingCount())
	// timing is advisory: not drift
	require.False(t, rep.Packages[0].HasDrift())
	require.Equal(t, 0, rep.DriftCount())
}

func TestCheck_TimingSpecUntracked(t *testing.T) {
	root := t.TempDir()
	writeSpec(t, root, "pkg", "pkg", "- `root.go` — r\n")
	writeFile(t, filepath.Join(root, "pkg", "root.go"), "package pkg\n")
	// spec absent from times (no commit history) -> timing skipped, no error
	v := &fakeVCS{
		files: map[string][]string{"": {"pkg/SPEC.md"}, "pkg": {"pkg/SPEC.md", "pkg/root.go"}},
		times: map[string]time.Time{"pkg/root.go": time.Unix(1_710_000_000, 0)},
	}
	rep, err := checkWith(root, v)
	require.NoError(t, err)
	require.Empty(t, rep.Packages[0].Timing)
}
```

- [ ] **Step 2: 跑测试，确认失败**

Run: `go test ./internal/spec/ -run TestCheck_Timing`
Expected: FAIL——`Timing` 始终为空（时序逻辑尚未实现），`require.Len(...,1)` 不满足。

- [ ] **Step 3: 插入时序实现**

在 `internal/spec/check.go` 的 `checkPackage` 中，把末尾的：

```go
	} else {
		pr.MissingFileSection = true
	}
	return pr, nil
}
```

替换为：

```go
	} else {
		pr.MissingFileSection = true
	}

	// Timing hints are advisory and never affect the exit code.
	if specTime, specOK, err := vcs.LastCommitTime(specPath); err != nil {
		return pr, err
	} else if specOK {
		pkgPrefix := sp.PkgDir
		if pkgPrefix != "" {
			pkgPrefix += "/"
		}
		for _, f := range sortedKeys(actualRel) {
			ft, ok, err := vcs.LastCommitTime(pkgPrefix + f)
			if err != nil {
				return pr, err
			}
			if ok && ft.After(specTime) {
				pr.Timing = append(pr.Timing, TimingHint{File: f, FileTime: ft, SpecTime: specTime})
			}
		}
	}
	return pr, nil
}
```

并在文件末尾（`inNestedSpecDir` 之后）追加 helper：

```go
func sortedKeys(m map[string]bool) []string {
	ks := make([]string, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	return ks
}
```

- [ ] **Step 4: 跑测试，确认通过**

Run: `go test ./internal/spec/ -v`
Expected: PASS（全部测试，含时序）。

- [ ] **Step 5: 提交**

```bash
git add internal/spec/check.go internal/spec/check_test.go
git commit -m "feat(spec): add advisory timing hints (source newer than spec)"
```

---

## 步骤 3 验证清单

- [ ] `go build ./...` 成功
- [ ] `go test ./internal/spec/ -v` 全绿（结构 + 时序 + 计数）
- [ ] `go test ./...` 全绿（无回归）
- [ ] fake VCS 覆盖：漂移、干净、嵌套包剔除、package 不符、缺 # 文件、时序、SPEC 未跟踪
