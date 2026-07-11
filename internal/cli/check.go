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
