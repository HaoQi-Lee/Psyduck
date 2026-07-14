---
psy_kind: factual
psy_version: 1
package: internal/spec
created: 2026-07-12
---

# 概述

`psy check` 漂移检测的纯逻辑核心。解析每个 `SPEC.md` 的 `package:` front-matter 与 `# 文件` 章节，**以该 SPEC 最近一次提交为锚点**，用 `git diff` 取包目录内自上次同步以来的净变更，分类为结构漂移（增/删与 SPEC 冲突）与时序过期（仅内容修改）。无 cobra 依赖，便于注入 fake VCS 单测。

# 文件

- `spec.go` — 纯解析：从 SPEC.md 字节流提取 `package:` 值与 `# 文件` bullet 列表（`Parse`、`Spec`）。
- `git.go` — `VCS` 接口、`NameStatus` 类型、`gitVCS` 生产实现（`os/exec` 调系统 git：`ls-files`、`log`、`diff --name-status`）、`RepoRoot`、`parseNameStatus`。
- `check.go` — 检测编排：`Check`/`checkWith`/`checkPackage`、变更分类 `classify`、`Report`/`PackageReport`/`TimingHint` 类型与计数方法。
- `spec_test.go` — `Parse` 解析边界单测。
- `git_test.go` — `gitVCS` 单测（真实临时 git 仓库，含 `LastCommit`/`DiffNameStatus`）。
- `classify_test.go` — `classify` 纯函数表驱动单测（A/D/M/R/C/T 分类规则）。
- `check_test.go` — 检测编排单测（`fakeVCS` 注入，覆盖漂移/资源豁免/嵌套/时序/未提交/错误传播）。
- `testutil_test.go` — 测试辅助（`initGitRepo`、`commitAllAt`、`writeSpec` 等）。

# API

- `func Parse(path string, data []byte) Spec` — 解析 SPEC.md，返回 `package:` 值与 `# 文件` 文件列表。
- `type Spec struct` — 解析结果（`Package`、`HasPackage`、`PkgDir`、`Files`、`HasFilesSection`）。
- `type VCS interface { ListFiles(relDir string) ([]string, error); LastCommitTime(relPath string) (time.Time, bool, error); LastCommit(relPath string) (string, bool, error); DiffNameStatus(fromCommit, relDir string) ([]NameStatus, error) }` — 源码控制抽象；生产为 `gitVCS`，测试注入 fake。
- `type NameStatus struct` — 一条 `git diff --name-status` 净变更（`Status`、新 `Path`、`OldPath` 仅 R/C）。
- `func RepoRoot() (string, error)` — 当前工作目录的 git 仓库根（`git rev-parse --show-toplevel`）。
- `func Check(repoRoot string) (Report, error)` — 扫描仓库内所有 SPEC.md，返回漂移报告（只读，首个错误即中止）。
- `type Report struct` — 全仓库报告，含 `Packages []PackageReport`。
- `type PackageReport struct` — 单包结果（`PackageMismatch`、`MissingFileSection`、`Added`、`Removed`、`SpecUntracked`、`Timing`）。
- `type TimingHint struct` — 时序提示（自同步以来修改过的源文件）。
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

## 漂移语义（基于变更）

检测以每个 `SPEC.md` 的**最近一次提交**为锚点（`LastCommit`，`git log -1 --format=%H`），用 `DiffNameStatus(anchor, pkgDir)`（`git diff --no-renames --name-status <anchor> HEAD -- <pkg>`）取包目录内**自上次同步以来的净变更**，剔除 SPEC.md 自身、含子 SPEC.md 的嵌套包子目录、非代码目录（`isNonCode`）下的条目，重定位为包目录相对后交 `classify` 分类：

- **Added（漂移）**：新增的**源类型**文件（类型属 SPEC 已声明词汇）且 SPEC 未列。资源文件（`*.png`/`*.yaml` 等未声明类型）豁免，见下。
- **Removed（漂移）**：自同步以来删除、但 SPEC 仍列出的文件。
- **Modified（stale）**：仅内容修改（`M`）、类型变更（`T`）或未知状态——SPEC 文字可能过期，仅提示，恒不影响退出码。

`gitVCS` 用 `--no-renames`，故重命名表现为「旧删 + 新增」，与显式 `R` 经 `classify` 得到相同的「removed 旧 + added 新」。`package:` 值与位置不符或缺 `# 文件` 章节亦记为漂移；SPEC 无提交历史（`SpecUntracked`）则跳过检测（不可度量，不算漂移）。

**与旧快照模型的区别**：旧版比对「声明集 vs 当前跟踪集」的全量快照；新版只看「自上次同步以来的增量」。代价：上次同步之前就存在的历史不一致不再报告（不在锚点之后）。收益：信号聚焦于「改了代码却没 re-sync」，且 `/psy-sync` 重写 SPEC 后该包立即归零。

**根目录恒非包**：仓库根的 `SPEC.md`（`path.Dir == "."`）在发现阶段被跳过——否则其 `PkgDir=""` 会让 `DiffNameStatus(anchor, "")` 对全仓做 diff。故根级文件（`go.mod`、`README.md` 等）与独立非包目录天然不参与 check。

**非代码目录排除**：diff 条目按**任意一级目录段**自动剔除 `testdata`、`vendor`、`node_modules` 及点前缀目录（`.idea`、`.git`、`.vscode` 等，编辑器/VCS 噪声），见 `isNonCode`。仅按目录段判定，故点文件（如 `.gitignore`）保留。`static`/`assets`/`dist`/`build` 等项目目录不全局排除——它们的归属由「文件类型词汇」判定，而非全局黑名单。

## 文件类型词汇（资源豁免）

`psy-sync` 的 `# 文件` 只列源文件；新增的资源文件（`*.png`/`*.yaml`/locale 等）类型不在 SPEC 词汇内。`classify` 对**新增**条目仅当 `typeKey(f)`（扩展名小写带点）落入 SPEC 已声明词汇（`typeKeys`）且未声明时才记 `Added`：

- 资源类型（SPEC 从未声明）自动豁免——`assets/logo.png`、`app.yaml` 新增不算漂移；
- 同一声明类型的漏列仍算漂移——SPEC 列了 `skills/embed.md`，则未列的 `skills/extra.md` 新增是真实遗漏。

删除方向（`Removed`）不按类型过滤——SPEC 列了却已删即为漂移。这把「哪些算源码」的定义权交给每个包自己的 SPEC，check 不再猜测目录/扩展名类型。
