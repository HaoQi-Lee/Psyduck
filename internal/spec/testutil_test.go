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
