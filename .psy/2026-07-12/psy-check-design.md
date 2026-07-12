# 设计文档：`psy check` 子命令（漂移检测 / 校验阶段）

- 日期：2026-07-10
- 状态：待审核
- 作者：李豪奇 + Claude
- 关联：`psy-sync`、`psy-sync-all` skill；`internal/cli`、`internal/version` 包

---

## 1. 背景与动机

psyduck 当前的「对账」手段只有 `psy-sync-all`——一个**全量重写**命令：它让 LLM 重读源码、重写每个包的 `SPEC.md`，会产生大量 diff、改动既有内容，无法直接当作只读的校验/门禁用。

缺失的是一个**只读、可作门禁（CI / pre-commit）的校验命令**：不写任何文件，只比对 `SPEC.md` 与仓库现实，报告漂移并以退出码表达成败。

本设计新增 `psy check` 子命令填补这一「校验」阶段，覆盖两个场景：

1. **结构漂移**——解析 `SPEC.md` 的 `package:` front-matter 定位目录，比对目录里实际源文件集合 vs `# 文件` 章节列出的文件集合，多/少即漂移。
2. **时序关系**——比对 git 日志里 `SPEC.md` 与包内源文件的最新提交时间，源比 spec 新则可能过期，**仅作 hint**。

---

## 2. 目标与非目标

### 目标

- 只读：不修改任何文件，不调 LLM，不联网。
- 可作门禁：明确的退出码语义，结构漂移即失败。
- 两个检测场景：结构漂移（失败级）+ 时序过期（提示级）。
- 纯文本人类可读输出；退出码承载机器可读的门禁信号。
- 纯逻辑可单测（git 抽象成接口）。

### 非目标（YAGNI，MVP 不做）

- 不做内容级漂移检测（API / 依赖描述是否准确）——需语义理解，留给 LLM skill。
- 不做 JSON / `--format` 模板 / SARIF 等机器可读输出格式——纯文本即可，门禁由退出码承载。
- 不做非 git 降级（git 是硬依赖，见 §5）。
- 不自动修复——修复是 `/psy-sync` 的职责，`check` 只读。

---

## 3. CLI 表面

```
psy check
```

**零参数、零 flag**。从 `git rev-parse --show-toplevel` 定位仓库根，扫描仓库内**每一个 `SPEC.md`**。行为固定：

- 只读，扫描全仓库，不接收路径参数。
- 输出为纯文本人类可读（无 ANSI 颜色，grep 友好，与本仓库 ethos 一致）。
- 结构漂移 → 退出码 1；时序过期 → 仅打印提示，退出码 0；内部错误 → 70（见 §4.4）。

> 不设 `--fail-on` / `--json` / `--no-color` 等开关：失败语义固定（结构漂移即失败），机器可读由**退出码**承载（CI 门禁只看退出码），颜色一律关闭以保持 grep 友好。YAGNI。

---

## 4. 退出码契约（核心，含 `main.go` 重构）

现有退出码只有 `ExitOK=0` 与 `ExitInternal=70`，且 `main.go` 对任何 `RunE` 错误一律 `os.Exit(70)`。这无法表达「漂移=1、错误=70」。需要引入「带退出码的错误」。

### 4.1 `exitcodes.go` 改动

```go
package cli

// Exit codes used across all psy subcommands.
const (
    ExitOK       = 0
    ExitDrift    = 1   // 新增：psy check 检测到漂移（门禁失败）
    ExitInternal = 70
)

// ExitError 携带退出码的错误。非空 Msg 会被 main 打到 stderr。
// 用于把「漂移（期望结果，退出 1）」与「内部错误（退出 70）」区分开。
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
```

### 4.2 `cmd/psy/main.go` 与 `runPsyForTestscript` 改动

把「错误→退出码」的映射抽成共享 helper `exitCodeFromErr(err) (code int, msg string)`：

- `err == nil` → `(0, "")`
- `err` 是 `*ExitError` → `(e.Code, e.Msg)`
- 其他 `error` → `(ExitInternal, err.Error())`

`main.go`：

```go
func main() {
    root := cli.NewRootCmd(os.Stdout, os.Stderr)
    code, msg := cli.ExitCodeFromErr(root.Execute())
    if msg != "" {
        fmt.Fprintln(os.Stderr, "psy:", msg)
    }
    os.Exit(code)
}
```

`runPsyForTestscript` 用同一 helper，返回相同 code（让 testscript 能断言退出码）。

### 4.3 向后兼容

现有 `init` 返回普通 `fmt.Errorf("init: already initialized ...")` → 非 `*ExitError` → 仍走 `(70, msg)`，**行为不变**。`version` 返回 nil → 0，不变。

### 4.4 `psy check` 退出码矩阵

| 情况 | 退出码 |
|---|---|
| 无漂移 | `0` |
| 结构漂移（文件多/少、`package:` 不符、缺 `# 文件` 章节） | `1` |
| 时序过期 | `0`（仅打印提示，恒不影响退出码） |
| 不是 git 仓库 / git 缺失 / 解析异常 | `70` |

> 设计要点：**漂移不是「错误」而是「结果」**。`check` 的 RunE 自行把结果打印到 stdout，然后返回 `&ExitError{Code: ExitDrift}`（或 nil），而不是把漂移当 error 抛出。这样 main 只负责把码翻译成进程退出。

---

## 5. 发现与检测逻辑

git 是 `check` 的**硬依赖**：按既定方案，实际源文件集合取自 `git ls-files`。若 `git rev-parse --show-toplevel` 失败（非 git 仓库 / 无 git），报错退出 70，**不降级**到 `os.ReadDir`（避免实际集定义分叉、引入噪音）。

### 5.1 包发现

- 仓库内每一个 `SPEC.md` 对应一个待检包。
- 包目录 = 该 `SPEC.md` 所在目录（相对仓库根）。
- 用 front-matter `package:` 交叉校验：若 `package:` 值 ≠ 包目录相对根路径 → 记一条**结构问题**「package 路径与实际位置不符」（属漂移类，失败门禁），但**仍继续**用目录做后续检测。

### 5.2 结构漂移（场景 1）

每个包：

1. **定位包目录**（并按 §5.1 校验 `package:`）。
2. **声明集 D** = 解析 `# 文件` 章节得到的文件路径集合（解析规范见 §6）。元素为相对包目录、`filepath.Clean` + `filepath.ToSlash` 归一后的路径。
3. **实际集 A** = `git ls-files <包目录>` 结果，剔除：
   - `SPEC.md` 自身；
   - 任何**含子 `SPEC.md` 的子目录**下的所有文件（嵌套包归子包，不在父包重复计入）。
   - （`git ls-files` 天然排除 untracked / generated / `.gitignore` 项，且能正确纳入 `.md` 等非代码资产，例如本仓库 `internal/cli` 列的 `claudemd/section.md`、`skills/*.md`。）
   - **统一基准**：`git ls-files` 返回仓库根相对路径（如 `internal/cli/root.go`），而 D 是包目录相对。比对前把 A 的每个元素去掉包目录前缀（如 `internal/cli/`）重定位为**包目录相对**（如 `root.go`、`claudemd/section.md`），再与 D 比对。
4. 比对产出发现：
   - **listed but gone**（`D \ A`）→ 「SPEC 列了但目录里没有」
   - **exists but undocumented**（`A \ D`）→ 「目录里有但 SPEC 没列」
   - `# 文件` 章节缺失 → 记「缺少 # 文件 章节」，跳过该包文件集比对（仍属漂移类）。

### 5.3 时序提示（场景 2，仅 hint）

- 对 `SPEC.md` 与实际集 A 中每个源文件，取 `git log -1 --format=%ct -- <path>` 的最新提交 Unix 时间戳。
- 若某源文件时间 > `SPEC.md` 时间 → 提示「`<file>` 比 SPEC 新 N 天，spec 可能过期」。
- **仅打印**，恒不影响退出码（只有结构漂移才失败）。
- 未提交文件不在 A 中，故不参与时序；`SPEC.md` 未提交（无提交记录）→ 跳过该包时序并附说明。

> 性能注记：MVP 按文件逐个 `git log -1`（包内文件数不大，可接受）。后续可用一次 `git log --format=%ct --name-only` 批量化，作为优化项，不在 MVP。

---

## 6. SPEC.md 解析规范

### 6.1 front-matter（取 `package:`）

- 文件须以 `---\n` 起、`---\n` 止构成 YAML front-matter 块。
- 在块内匹配 `^\s*package:\s*(.+?)\s*$`，取捕获组，去除首尾引号，即声明的包路径。
- 仅需此字段，**不引入 YAML 库**（贴合「薄」）。front-matter 缺失或无 `package:` → 记「SPEC 缺少 package front-matter」（漂移类）。

### 6.2 `# 文件` 章节

- 定位标题行：`^#\s*文件\b`（中文，与 skill 规定一致）。英文 `# Files` 作为**兜底**兼容（容错），两者都识别。
- 章节范围 = 该标题行到下一个 `^#\s` 标题行之间。
- 解析 bullet：每行形如 `- \`<path>\` — <desc>`，取**第一个反引号包起来的 token** 作为声明的文件路径。
- 路径相对包目录；归一为 `Clean` + `ToSlash` 后入集合 D。
- 找不到标题行 → 视为「缺少 # 文件 章节」。

---

## 7. VCS 接口与可测试性

把 git 抽成接口，让纯逻辑无需真 git 即可单测：

```go
// internal/spec
type VCS interface {
    // Root 返回仓库根（git rev-parse --show-toplevel）。
    Root() (string, error)
    // ListFiles 返回 dir 下 git 跟踪的文件，相对仓库根（git ls-files <dir>）。
    ListFiles(dir string) ([]string, error)
    // LastCommitTime 返回 path 最新提交时间（git log -1 --format=%ct -- <path>）。
    LastCommitTime(path string) (time.Time, error)
}
```

- 生产实现 `gitVCS{}` 走 `os/exec` 调系统 `git`。
- 检测入口 `Check(root string, vcs VCS) Report` 接收注入的 VCS；单测注入 fake VCS。

---

## 8. 包结构与文件清单（方案 A：纯逻辑包 + 薄 cli）

```
internal/spec/                       # 新包：纯逻辑，无 cobra 依赖
  spec.go          # SPEC.md 解析：front-matter (package:)、# 文件 bullet 列表
  git.go           # VCS 接口 + gitVCS 生产实现 (os/exec)
  check.go         # Check(root, vcs) Report：发现+漂移+时序编排；Report 数据结构
  spec_test.go     # 解析边界单测
  git_test.go      # gitVCS 单测（真实临时 git 仓库）
  check_test.go    # 检测编排单测（fake VCS）
internal/cli/
  check.go         # 新增：薄 cobra 命令（零参数零 flag）→ spec.Check → 渲染（人类纯文本）
  exitcodes.go     # 改：+ExitDrift、+ExitError、+ExitCodeFromErr
  root.go          # 改：注册 newCheckCmd()
cmd/psy/
  main.go          # 改：用 ExitCodeFromErr 翻译退出码
testdata/script/
  check_drift.txt   # 新增：git init 微型 fixture，声明集与实际集不符，断言退出码 1
  check_clean.txt   # 新增：无漂移断言退出码 0
  check_stale.txt   # 新增：源比 SPEC 新 → 打印时序提示但退出码 0
```

渲染（人类可读 + JSON）放在 `cli/check.go`，保持 `spec` 包「只有数据与逻辑、无展示」。

---

## 9. 输出格式

纯文本、无 ANSI 颜色（grep 友好）。示例：

```
psy check — 检查 SPEC.md 与现实的漂移

internal/cli  (SPEC.md)
  ✗ 结构漂移
      + 未文档化: check.go        （目录有，SPEC 未列）
      - 已过期:   old.go          （SPEC 列了，目录没有）
  ⚠ 时序提示
      root.go 比 SPEC 新 25 天（spec 可能过期）

internal/version  (SPEC.md)
  ✓ 无漂移

发现: 2 处结构漂移, 1 条时序提示
```

- `✗` 结构漂移（失败级，退出码 1）；`⚠` 时序提示（退出码 0）；`✓` 无漂移。
- 末行汇总。

> 内部仍以 `Report` 结构体承载检测结果（便于单测与渲染），只是不对外暴露 JSON。

---

## 10. 测试计划

### 单元测试（`internal/spec`）

- **解析**：front-matter 含/缺 `package:`、带引号；`# 文件` bullet 含子目录路径（`claudemd/section.md`）；章节缺失；英文 `# Files` 兜底；下一个 `#` 标题正确截断。
- **集合 diff**：listed-but-gone、exists-but-undocumented、完全一致。
- **嵌套包剔除**：父包目录下含子 `SPEC.md` 时，子目录文件不计入父包 A。
- **`package:` 不符**：报结构问题但不中断。
- **时序比较**：源新于 spec 触发提示；SPEC 未提交跳过。
- 全部用 **fake VCS** 注入，不依赖真 git。

### `gitVCS` 单测

- 在 `t.TempDir()` 内 `git init` + commit，验证 `Root` / `ListFiles` / `LastCommitTime` 返回值。

### testscript E2E（`testdata/script/`）

- `check_drift.txt`：`git init` 微型仓库，造一个声明集与实际集不符的包，断言 `psy check` 退出码 1、输出含未文档化/已过期。
- `check_clean.txt`：声明集与实际集一致，断言退出码 0。
- `check_stale.txt`：源比 SPEC 新 → 输出含时序提示，但退出码 0（时序恒不影响退出码）。
- 依赖 CI 环境装了 git（与 README「Git | os/exec 调系统 git」一致）。

---

## 11. 边界与风险

- **嵌套包边界**：靠「含子 `SPEC.md` 的子目录」划分。若某子目录是独立包却**没写** `SPEC.md`，其文件会被计入父包 A——但这正说明父包 spec 应当列出或子包补 spec，属可接受的「真实」信号。
- **`# 文件` 路径写法**：规范为相对包目录；若 spec 里写了 basename 而文件在子目录，会误报。归一规则已在 §6.2 固定（相对包目录），生成 spec 的 skill 须遵守同规则（现状一致）。
- **git 性能**：大仓库逐文件 `git log` 可能慢；MVP 接受，后续批量化。
- **退出码重构**：改 `main.go` 是本特性落地的前提；向后兼容（§4.3），现有 `init`/`version` 行为不变。

---

## 12. 待确认（审核时）

1. 退出码重构（引入 `ExitError` + 改 `main.go`）是否接受——这是「漂移=1、错误=70」干净落地的前提。
2. 英文 `# Files` 兜底解析是否需要（还是严格只认 `# 文件`）。
3. 人类输出格式（符号 `✗ ⚠ ✓`、分组、汇总行、无颜色）是否合意。
