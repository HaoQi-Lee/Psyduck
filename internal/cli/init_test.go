package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// chdir changes the working directory to dir and returns a restore function.
func chdir(t *testing.T, dir string) {
	t.Helper()
	orig, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(dir))
	t.Cleanup(func() { os.Chdir(orig) })
}

func TestInit_Basic(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)

	var out bytes.Buffer
	root := NewRootCmd(&out, &out)
	root.SetArgs([]string{"init"})
	require.NoError(t, root.Execute())

	require.DirExists(t, filepath.Join(dir, ".psy"))
}

func TestInit_CreatesClaudeMd(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)

	var out bytes.Buffer
	root := NewRootCmd(&out, &out)
	root.SetArgs([]string{"init"})
	require.NoError(t, root.Execute())

	claudeMd := filepath.Join(dir, "CLAUDE.md")
	require.FileExists(t, claudeMd)
	data, err := os.ReadFile(claudeMd)
	require.NoError(t, err)
	require.Contains(t, string(data), "<!-- psyduck -->")
	require.Contains(t, string(data), "SPEC.md")
}

func TestInit_AppendsToExistingClaudeMd(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)
	claudeMd := filepath.Join(dir, "CLAUDE.md")
	require.NoError(t, os.WriteFile(claudeMd, []byte("# My Project\n\nSome existing content.\n"), 0o644))

	var out bytes.Buffer
	root := NewRootCmd(&out, &out)
	root.SetArgs([]string{"init"})
	require.NoError(t, root.Execute())

	data, err := os.ReadFile(claudeMd)
	require.NoError(t, err)
	require.Contains(t, string(data), "# My Project")
	require.Contains(t, string(data), "<!-- psyduck -->")
}

func TestInit_DoesNotDuplicateClaudeMdSection(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)

	var out bytes.Buffer
	root := NewRootCmd(&out, &out)
	root.SetArgs([]string{"init"})
	require.NoError(t, root.Execute())

	// Run init again on a fresh psyduck instance (simulate by removing .psy).
	os.RemoveAll(filepath.Join(dir, ".psy"))
	root2 := NewRootCmd(&out, &out)
	root2.SetArgs([]string{"init"})
	require.NoError(t, root2.Execute())

	data, err := os.ReadFile(filepath.Join(dir, "CLAUDE.md"))
	require.NoError(t, err)
	// The marker should appear exactly once.
	require.Equal(t, 1, bytes.Count(data, []byte("<!-- psyduck -->")))
}

func TestInit_RefusesIfAlreadyInitialized(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".psy"), 0o755))

	var out bytes.Buffer
	root := NewRootCmd(&out, &out)
	root.SetArgs([]string{"init"})
	err := root.Execute()
	require.Error(t, err)
	require.Contains(t, err.Error(), "already initialized")
}

func TestInit_InstallPlugins_CreatesPluginDirs(t *testing.T) {
	workDir := t.TempDir()
	chdir(t, workDir)

	// Point HOME at a temp dir so we don't pollute the real home.
	homeDir := filepath.Join(workDir, "home")
	require.NoError(t, os.MkdirAll(homeDir, 0o755))
	t.Setenv("HOME", homeDir)
	t.Setenv("USERPROFILE", homeDir)

	var out bytes.Buffer
	root := NewRootCmd(&out, &out)
	root.SetArgs([]string{"init", "--install-plugins"})
	require.NoError(t, root.Execute())

	skillsBase := filepath.Join(homeDir, ".claude", "skills")
	for _, name := range []string{"psy-sync", "psy-sync-all"} {
		skillPath := filepath.Join(skillsBase, name, "SKILL.md")
		require.FileExists(t, skillPath, "plugin %s/SKILL.md should exist", name)
	}

	// Removed skills should NOT be installed.
	for _, gone := range []string{"psy-syncing", "psy-archiving", "psy-reading-context"} {
		_, err := os.Stat(filepath.Join(skillsBase, gone))
		require.True(t, os.IsNotExist(err), "plugin %s should not be installed", gone)
	}

	require.Contains(t, out.String(), "installed 2 plugin(s)")
}

func TestInit_InstallPlugins_SkipsExisting(t *testing.T) {
	workDir := t.TempDir()
	chdir(t, workDir)

	homeDir := filepath.Join(workDir, "home")
	require.NoError(t, os.MkdirAll(homeDir, 0o755))
	t.Setenv("HOME", homeDir)
	t.Setenv("USERPROFILE", homeDir)

	// First run.
	var out bytes.Buffer
	root := NewRootCmd(&out, &out)
	root.SetArgs([]string{"init", "--install-plugins"})
	require.NoError(t, root.Execute())

	// Modify one SKILL.md to prove it's NOT overwritten on second run.
	targetPath := filepath.Join(homeDir, ".claude", "skills", "psy-sync", "SKILL.md")
	require.NoError(t, os.WriteFile(targetPath, []byte("custom content"), 0o644))

	// Remove .psy so init won't fail with "already initialized".
	os.RemoveAll(filepath.Join(workDir, ".psy"))

	// Second run.
	out.Reset()
	root = NewRootCmd(&out, &out)
	root.SetArgs([]string{"init", "--install-plugins"})
	require.NoError(t, root.Execute())

	data, err := os.ReadFile(targetPath)
	require.NoError(t, err)
	require.Equal(t, "custom content", string(data))
	require.Contains(t, out.String(), "skipped")
}

func TestInit_InstallPlugins_ListsInstalled(t *testing.T) {
	workDir := t.TempDir()
	chdir(t, workDir)

	homeDir := filepath.Join(workDir, "home")
	require.NoError(t, os.MkdirAll(homeDir, 0o755))
	t.Setenv("HOME", homeDir)
	t.Setenv("USERPROFILE", homeDir)

	var out bytes.Buffer
	root := NewRootCmd(&out, &out)
	root.SetArgs([]string{"init", "--install-plugins"})
	require.NoError(t, root.Execute())

	output := out.String()
	require.Contains(t, output, "psy-sync")
	require.Contains(t, output, "psy-sync-all")
	require.NotContains(t, output, "psy-syncing")
	require.NotContains(t, output, "psy-archiving")
	require.NotContains(t, output, "psy-reading-context")
}

func TestInit_ClaudeMdAdvertisesNewTriggers(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)

	var out bytes.Buffer
	root := NewRootCmd(&out, &out)
	root.SetArgs([]string{"init"})
	require.NoError(t, root.Execute())

	data, err := os.ReadFile(filepath.Join(dir, "CLAUDE.md"))
	require.NoError(t, err)
	content := string(data)

	// The CLAUDE.md section advertises only the automatic /psy-sync trigger.
	// Use backtick-quoted form to disambiguate from /psy-sync-all.
	require.Contains(t, content, "`/psy-sync`")
	require.Contains(t, content, "superpowers:executing-plans")
	require.Contains(t, content, "MANDATORY")

	// /psy-sync-all is a manual-only command and must NOT be advertised in CLAUDE.md.
	require.NotContains(t, content, "psy-sync-all")

	// The retired skills must not be advertised.
	require.NotContains(t, content, "psy-syncing")
	require.NotContains(t, content, "psy-archiving")
	require.NotContains(t, content, "psy-reading-context")
}
