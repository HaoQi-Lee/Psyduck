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

func TestGitVCS_LastCommitAndDiff(t *testing.T) {
	root := t.TempDir()
	initGitRepo(t, root)
	writeFile(t, filepath.Join(root, "pkg", "keep.go"), "package pkg\n")
	writeFile(t, filepath.Join(root, "pkg", "gone.go"), "package pkg\n")
	writeFile(t, filepath.Join(root, "pkg", "SPEC.md"), "---\npackage: pkg\n---\n")
	commitAllAt(t, root, "init", "2026-06-05T00:00:00")

	v := newGitVCS(root)

	anchor, ok, err := v.LastCommit("pkg/SPEC.md")
	require.NoError(t, err)
	require.True(t, ok)
	require.Len(t, anchor, 40) // SHA-1 hex

	_, ok, err = v.LastCommit("never.go")
	require.NoError(t, err)
	require.False(t, ok) // no commit history

	// No change since the SPEC's last commit.
	changes, err := v.DiffNameStatus(anchor, "pkg")
	require.NoError(t, err)
	require.Empty(t, changes)

	// Second commit: add new.go, delete gone.go, modify keep.go (SPEC untouched).
	writeFile(t, filepath.Join(root, "pkg", "new.go"), "package pkg\n")
	require.NoError(t, os.Remove(filepath.Join(root, "pkg", "gone.go")))
	writeFile(t, filepath.Join(root, "pkg", "keep.go"), "package pkg\n// changed\n")
	commitAllAt(t, root, "edit", "2026-07-01T00:00:00")

	changes, err = v.DiffNameStatus(anchor, "pkg")
	require.NoError(t, err)
	byPath := map[string]string{}
	for _, c := range changes {
		byPath[c.Path] = c.Status
	}
	require.Equal(t, "A", byPath["pkg/new.go"], "new.go added since sync")
	require.Equal(t, "D", byPath["pkg/gone.go"], "gone.go deleted since sync")
	require.Equal(t, "M", byPath["pkg/keep.go"], "keep.go modified since sync")
}
