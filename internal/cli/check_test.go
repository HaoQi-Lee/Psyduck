package cli

import (
	"bytes"
	"os/exec"
	"path/filepath"
	"strings"
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
