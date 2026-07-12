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

// renderReport writes a compact, plain-text, grep-friendly summary: one line
// per package ("<path>: OK|NG"), with indented detail lines only when needed.
// No ANSI colors, no Chinese.
func renderReport(w io.Writer, r spec.Report) {
	fmt.Fprintln(w, "psy check")
	for _, p := range r.Packages {
		label := p.PkgDir
		if label == "" {
			label = "."
		}
		status := "OK"
		if p.HasDrift() {
			status = "NG"
		}
		fmt.Fprintf(w, "%s: %s\n", label, status)
		for _, f := range p.Undocumented {
			fmt.Fprintf(w, "  + %s (undocumented)\n", f)
		}
		for _, f := range p.ListedButGone {
			fmt.Fprintf(w, "  - %s (missing)\n", f)
		}
		if p.PackageMismatch {
			fmt.Fprintf(w, "  ! package mismatch (declared %q, at %q)\n", p.Package, p.PkgDir)
		}
		if p.MissingFileSection {
			fmt.Fprintln(w, "  ! missing files section")
		}
		for _, th := range p.Timing {
			days := int(th.FileTime.Sub(th.SpecTime).Hours() / 24)
			fmt.Fprintf(w, "  ~ %s (%dd newer than spec)\n", th.File, days)
		}
	}
	fmt.Fprintf(w, "summary: %d drift, %d stale\n", r.DriftCount(), r.TimingCount())
}
