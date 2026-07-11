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
