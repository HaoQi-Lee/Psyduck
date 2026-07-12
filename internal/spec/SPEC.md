---
psy_kind: factual
psy_version: 1
package: internal/spec
created: 2026-07-12
---

# 概述

`psy check` 漂移检测的纯逻辑核心。解析 `SPEC.md` 的 `package:` front-matter 与 `# 文件` 章节，经 `git` 取仓库现实（跟踪文件、最新提交时间），比对声明集与实际集，产出结构漂移发现与（仅提示的）时序过期。无 cobra 依赖，便于注入 fake VCS 单测。

# 文件

- `spec.go` — 纯解析：从 SPEC.md 字节流提取 `package:` 值与 `# 文件` bullet 列表（`Parse`、`Spec`）。
- `git.go` — `VCS` 接口、`gitVCS` 生产实现（`os/exec` 调系统 git）、`RepoRoot`。
- `check.go` — 检测编排：`Check`/`checkWith`/`checkPackage`，`Report`/`PackageReport`/`TimingHint` 类型与计数方法。
- `spec_test.go` — `Parse` 解析边界单测。
- `git_test.go` — `gitVCS` 单测（真实临时 git 仓库）。
- `check_test.go` — 检测编排单测（`fakeVCS` 注入，覆盖漂移/嵌套/时序/错误传播）。
- `testutil_test.go` — 测试辅助（`initGitRepo`、`commitAllAt`、`writeSpec` 等）。

# API

- `func Parse(path string, data []byte) Spec` — 解析 SPEC.md，返回 `package:` 值与 `# 文件` 文件列表。
- `type Spec struct` — 解析结果（`Package`、`HasPackage`、`PkgDir`、`Files`、`HasFilesSection`）。
- `type VCS interface { ListFiles(relDir string) ([]string, error); LastCommitTime(relPath string) (time.Time, bool, error) }` — 源码控制抽象；生产为 `gitVCS`，测试注入 fake。
- `func RepoRoot() (string, error)` — 当前工作目录的 git 仓库根（`git rev-parse --show-toplevel`）。
- `func Check(repoRoot string) (Report, error)` — 扫描仓库内所有 SPEC.md，返回漂移报告（只读，首个错误即中止）。
- `type Report struct` — 全仓库报告，含 `Packages []PackageReport`。
- `type PackageReport struct` — 单包结果（`PackageMismatch`、`MissingFileSection`、`ListedButGone`、`Undocumented`、`Timing`）。
- `type TimingHint struct` — 时序提示（源文件比 SPEC 新）。
- `func (PackageReport) HasDrift() bool` / `func (Report) DriftCount() int` / `func (Report) TimingCount() int` — 计数（时序不计入漂移）。

# 依赖

- 标准库（`os`、`os/exec`、`path`、`path/filepath`、`regexp`、`sort`、`strconv`、`time` 等）。
- 系统 `git`（运行期硬依赖）。
- `github.com/stretchr/testify`、`github.com/rogpeppe/go-internal/testscript`（仅测试）。

# 设计重点

## VCS 抽象与可测试性

`Check`（公开，用真实 `gitVCS`）委托给 `checkWith(repoRoot, vcs)`（私有，接受注入的 `VCS`）。单测用 `fakeVCS` 注入，无需真 git 即可覆盖发现/漂移/时序/错误传播；`gitVCS` 则用真实临时 git 仓库单独验证。

## 跨平台路径处理

git 永远返回正斜杠路径。切分 git 返回的字符串必须用 `path` 包（OS 无关），`filepath` 仅用于 `os.ReadFile` 前把正斜杠转回 OS 分隔符——否则在 Windows 上 `filepath.Base("pkg/SPEC.md")` 会整体当作 basename。

## 漂移语义

声明集 D（`# 文件`，包目录相对）vs 实际集 A（`git ls-files`，剔除 SPEC.md 自身与含子 SPEC.md 的嵌套包子目录，重定位为包目录相对）。`ListedButGone`（D∖A）、`Undocumented`（A∖D）；`package:` 值与位置不符或缺 `# 文件` 章节亦记为漂移。时序提示恒不影响漂移判定（仅提示）。
