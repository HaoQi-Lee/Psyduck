# `psy check` 实施计划总览

> **For agentic workers:** REQUIRED SUB-SKILL: 用 `superpowers:subagent-driven-development`（推荐）或 `superpowers:executing-plans` 逐任务执行本计划。步骤用 checkbox（`- [ ]`）追踪。

**目标：** 新增 `psy check` 子命令——只读检测 `SPEC.md` 与仓库现实之间的漂移（结构漂移 + 时序提示），作为 CI/pre-commit 门禁，退出码表达成败。

**架构：** 新建纯逻辑包 `internal/spec`（SPEC 解析、git 抽象、检测编排、`Report`），`internal/cli/check.go` 作薄 cobra 封装。引入 `ExitError` 类型 + `main.go` 重构，区分「漂移=1」与「错误=70」。git 经 `os/exec` 调系统 `git`，抽象为 `VCS` 接口以便用 fake 注入单测核心逻辑。

**技术栈：** Go 1.22、`spf13/cobra`、`stretchr/testify`、`rogpeppe/go-internal/testscript`、系统 `git`。

**开发分支：** `feature/psy-check`（所有任务在此分支，禁止在 `main`/`master` 上开发）。

**设计依据：** `docs/superpowers/specs/2026-07-10-psy-check-design.md`（已审核通过）。

---

## 步骤索引

| 步骤 | 名称 | 文件 | 状态 |
|------|------|------|------|
| 1 | 退出码基础设施 | [step-1-exit-code-infra.md](step-1-exit-code-infra.md) | [ ] 未开始 |
| 2 | SPEC 解析 + git 助手 | [step-2-parsing-and-git.md](step-2-parsing-and-git.md) | [ ] 未开始 |
| 3 | 检测编排（结构漂移 + 时序） | [step-3-check-orchestration.md](step-3-check-orchestration.md) | [ ] 未开始 |
| 4 | CLI 命令 + 渲染 + E2E | [step-4-cli-and-e2e.md](step-4-cli-and-e2e.md) | [ ] 未开始 |

## 执行顺序

严格按步骤顺序执行（DAG，无前向依赖）。每步完成、`go test` 通过后再进入下一步。每步内部任务亦严格线性。

## 全局约束

- **TDD 严格**：每个任务先写失败测试，跑红，再写最小实现，跑绿，再提交。
- **每任务一次提交**，提交信息用 conventional commits（`feat:` / `refactor:` / `test:` / `docs:`）。
- **不引入新依赖**：仅用 go.mod 现有依赖（cobra、testify、testscript、标准库）。
- **SPEC 解析不引 YAML 库**：仅需 `package:` 字段与 `# 文件` bullet，行解析即可（贴合「薄」）。
- **git 是 `check` 的硬依赖**：非 git 仓库 → 退出码 70，不降级。
- **`check` 只读**：不写任何文件、不联网、不调 LLM。
- **零参数零 flag**：`psy check` 不接受任何参数与开关；失败语义固定（结构漂移→1，时序→0）。
- 所有步骤完成后：运行 `make test` 与 `make lint`；随后执行 `/psy-sync`（MANDATORY，见 CLAUDE.md）更新受影响包的 `SPEC.md`（`internal/cli`、新增 `internal/spec`）。

## 设计要点回顾（执行者必读）

- **退出码契约**：`ExitOK=0`、`ExitDrift=1`、`ExitInternal=70`。`check` 的 RunE 自行打印结果，漂移时返回 `&ExitError{Code: ExitDrift}`（非空 Msg 才打 stderr）。
- **结构漂移**：声明集 D（`# 文件` 解析）vs 实际集 A（`git ls-files <包目录>`，剔除 SPEC.md 自身与含子 SPEC.md 的子目录，重定位为包目录相对）。
- **时序提示**：源文件最新提交时间 > SPEC.md 时间 → 提示，恒不影响退出码。
- **包发现**：`git ls-files` 全量过滤 basename==`SPEC.md`；包目录=SPEC.md 所在目录；`package:` 值与之不符记为结构问题。
- **`VCS` 接口保留**（设计 §7）：核心检测逻辑 `checkWith` 用 fake VCS 单测（快、确定性、不依赖真 git）；`gitVCS` 单测用真实临时 git 仓库。

## 类型/签名一致性清单（跨步骤共享，勿改名）

`internal/spec`：
- `type Spec struct { Path, Package string; HasPackage bool; PkgDir string; Files []string; HasFilesSection bool }`
- `func Parse(path string, data []byte) Spec`
- `type VCS interface { ListFiles(relDir string) ([]string, error); LastCommitTime(relPath string) (time.Time, bool, error) }`
- `func RepoRoot() (string, error)`
- `type Report struct { Root string; Packages []PackageReport }`
- `type PackageReport struct { Package, SpecPath, PkgDir string; PackageMismatch, MissingFileSection bool; ListedButGone, Undocumented []string; Timing []TimingHint }`
- `type TimingHint struct { File string; FileTime, SpecTime time.Time }`
- `func (PackageReport) HasDrift() bool` / `func (Report) DriftCount() int` / `func (Report) TimingCount() int`
- `func Check(repoRoot string) (Report, error)`

`internal/cli`：
- `const ExitDrift = 1`
- `type ExitError struct { Code int; Msg string }` + `func (e *ExitError) Error() string`
- `func ExitCodeFromErr(err error) (int, string)`
- `func newCheckCmd() *cobra.Command` / `func runCheck(cmd *cobra.Command) error` / `func renderReport(w io.Writer, r spec.Report)`
