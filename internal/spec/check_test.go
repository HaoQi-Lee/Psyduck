package spec

import (
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestReport_Counts(t *testing.T) {
	r := Report{Packages: []PackageReport{
		{ListedButGone: []string{"a"}}, // drift
		{Timing: []TimingHint{{}}},     // timing only, no drift
		{},                             // clean
	}}
	require.Equal(t, 1, r.DriftCount())
	require.Equal(t, 1, r.TimingCount())
}

func TestPackageReport_HasDrift(t *testing.T) {
	require.True(t, PackageReport{PackageMismatch: true}.HasDrift())
	require.True(t, PackageReport{MissingFileSection: true}.HasDrift())
	require.True(t, PackageReport{ListedButGone: []string{"x"}}.HasDrift())
	require.True(t, PackageReport{Undocumented: []string{"x"}}.HasDrift())
	require.False(t, PackageReport{}.HasDrift())
	require.False(t, PackageReport{Timing: []TimingHint{{}}}.HasDrift())
}

// fakeVCS is an in-memory VCS for testing checkWith without real git.
type fakeVCS struct {
	files   map[string][]string  // relDir -> files (repo-root-relative)
	times   map[string]time.Time // relPath -> time (absent = no history)
	listErr error                // if set, ListFiles returns this error
	timeErr error                // if set, LastCommitTime returns this error
}

func (f *fakeVCS) ListFiles(relDir string) ([]string, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	return f.files[relDir], nil
}
func (f *fakeVCS) LastCommitTime(relPath string) (time.Time, bool, error) {
	if f.timeErr != nil {
		return time.Time{}, false, f.timeErr
	}
	t, ok := f.times[relPath]
	return t, ok, nil
}

func TestCheck_StructuralDrift(t *testing.T) {
	root := t.TempDir()
	writeSpec(t, root, "pkg", "pkg", "- `root.go` — r\n- `old.go` — gone\n")
	writeFile(t, filepath.Join(root, "pkg", "root.go"), "package pkg\n")
	writeFile(t, filepath.Join(root, "pkg", "new.go"), "package pkg\n")

	v := &fakeVCS{files: map[string][]string{
		"":    {"pkg/SPEC.md"},
		"pkg": {"pkg/SPEC.md", "pkg/root.go", "pkg/new.go"},
	}}
	rep, err := checkWith(root, v)
	require.NoError(t, err)
	require.Len(t, rep.Packages, 1)
	pr := rep.Packages[0]
	require.Equal(t, []string{"old.go"}, pr.ListedButGone)
	require.Equal(t, []string{"new.go"}, pr.Undocumented)
	require.True(t, pr.HasDrift())
	require.Equal(t, 1, rep.DriftCount())
}

func TestCheck_CleanPackage(t *testing.T) {
	root := t.TempDir()
	writeSpec(t, root, "pkg", "pkg", "- `root.go` — r\n")
	writeFile(t, filepath.Join(root, "pkg", "root.go"), "package pkg\n")
	v := &fakeVCS{files: map[string][]string{
		"":    {"pkg/SPEC.md"},
		"pkg": {"pkg/SPEC.md", "pkg/root.go"},
	}}
	rep, err := checkWith(root, v)
	require.NoError(t, err)
	require.Len(t, rep.Packages, 1)
	require.False(t, rep.Packages[0].HasDrift())
}

func TestCheck_NestedPackageExcluded(t *testing.T) {
	root := t.TempDir()
	writeSpec(t, root, "pkg", "pkg", "- `parent.go` — p\n")
	writeFile(t, filepath.Join(root, "pkg", "parent.go"), "package pkg\n")
	writeSpec(t, root, "pkg/child", "pkg/child", "- `child.go` — c\n")
	writeFile(t, filepath.Join(root, "pkg", "child", "child.go"), "package child\n")
	v := &fakeVCS{files: map[string][]string{
		"":          {"pkg/SPEC.md", "pkg/child/SPEC.md"},
		"pkg":       {"pkg/SPEC.md", "pkg/parent.go", "pkg/child/SPEC.md", "pkg/child/child.go"},
		"pkg/child": {"pkg/child/SPEC.md", "pkg/child/child.go"},
	}}
	rep, err := checkWith(root, v)
	require.NoError(t, err)
	require.Len(t, rep.Packages, 2)
	byDir := map[string]PackageReport{}
	for _, p := range rep.Packages {
		byDir[p.PkgDir] = p
	}
	require.False(t, byDir["pkg"].HasDrift(), "parent must not see child files")
	require.Empty(t, byDir["pkg"].Undocumented)
	require.False(t, byDir["pkg/child"].HasDrift())
}

func TestCheck_PackageMismatch(t *testing.T) {
	root := t.TempDir()
	writeSpec(t, root, "pkg", "wrong/path", "- `root.go` — r\n")
	writeFile(t, filepath.Join(root, "pkg", "root.go"), "package pkg\n")
	v := &fakeVCS{files: map[string][]string{"": {"pkg/SPEC.md"}, "pkg": {"pkg/SPEC.md", "pkg/root.go"}}}
	rep, err := checkWith(root, v)
	require.NoError(t, err)
	require.True(t, rep.Packages[0].PackageMismatch)
}

func TestCheck_MissingFilesSection(t *testing.T) {
	root := t.TempDir()
	writeSpec(t, root, "pkg", "pkg", "") // no # 文件
	writeFile(t, filepath.Join(root, "pkg", "root.go"), "package pkg\n")
	v := &fakeVCS{files: map[string][]string{"": {"pkg/SPEC.md"}, "pkg": {"pkg/SPEC.md", "pkg/root.go"}}}
	rep, err := checkWith(root, v)
	require.NoError(t, err)
	require.True(t, rep.Packages[0].MissingFileSection)
	require.Empty(t, rep.Packages[0].ListedButGone)
	require.Empty(t, rep.Packages[0].Undocumented)
}

func TestCheck_TimingHint(t *testing.T) {
	root := t.TempDir()
	writeSpec(t, root, "pkg", "pkg", "- `root.go` — r\n")
	writeFile(t, filepath.Join(root, "pkg", "root.go"), "package pkg\n")
	specT := time.Unix(1_700_000_000, 0) // older
	fileT := time.Unix(1_710_000_000, 0) // newer
	v := &fakeVCS{
		files: map[string][]string{"": {"pkg/SPEC.md"}, "pkg": {"pkg/SPEC.md", "pkg/root.go"}},
		times: map[string]time.Time{"pkg/SPEC.md": specT, "pkg/root.go": fileT},
	}
	rep, err := checkWith(root, v)
	require.NoError(t, err)
	require.Len(t, rep.Packages[0].Timing, 1)
	require.Equal(t, "root.go", rep.Packages[0].Timing[0].File)
	require.Equal(t, 1, rep.TimingCount())
	// timing is advisory: not drift
	require.False(t, rep.Packages[0].HasDrift())
	require.Equal(t, 0, rep.DriftCount())
}

func TestCheck_TimingSpecUntracked(t *testing.T) {
	root := t.TempDir()
	writeSpec(t, root, "pkg", "pkg", "- `root.go` — r\n")
	writeFile(t, filepath.Join(root, "pkg", "root.go"), "package pkg\n")
	// spec absent from times (no commit history) -> timing skipped, no error
	v := &fakeVCS{
		files: map[string][]string{"": {"pkg/SPEC.md"}, "pkg": {"pkg/SPEC.md", "pkg/root.go"}},
		times: map[string]time.Time{"pkg/root.go": time.Unix(1_710_000_000, 0)},
	}
	rep, err := checkWith(root, v)
	require.NoError(t, err)
	require.Empty(t, rep.Packages[0].Timing)
}

func TestCheck_ListFilesErrorPropagated(t *testing.T) {
	root := t.TempDir()
	v := &fakeVCS{listErr: errors.New("list boom")}
	_, err := checkWith(root, v)
	require.Error(t, err)
	require.Contains(t, err.Error(), "list boom")
}

func TestCheck_LastCommitTimeErrorPropagated(t *testing.T) {
	root := t.TempDir()
	writeSpec(t, root, "pkg", "pkg", "- `root.go` — r\n")
	writeFile(t, filepath.Join(root, "pkg", "root.go"), "package pkg\n")
	v := &fakeVCS{
		files:   map[string][]string{"": {"pkg/SPEC.md"}, "pkg": {"pkg/SPEC.md", "pkg/root.go"}},
		timeErr: errors.New("time boom"),
	}
	_, err := checkWith(root, v)
	require.Error(t, err)
	require.Contains(t, err.Error(), "time boom")
}

func TestCheck_RootLevelSpec(t *testing.T) {
	root := t.TempDir()
	// package: "" parses to an empty name (trimQuotes), matching the root
	// PkgDir ("") so PackageMismatch stays false.
	writeSpec(t, root, "", `""`, "- `main.go` — m\n")
	writeFile(t, filepath.Join(root, "main.go"), "package main\n")
	v := &fakeVCS{files: map[string][]string{
		"": {"SPEC.md", "main.go"},
	}}
	rep, err := checkWith(root, v)
	require.NoError(t, err)
	require.Len(t, rep.Packages, 1)
	require.Equal(t, "", rep.Packages[0].PkgDir)
	require.False(t, rep.Packages[0].HasDrift())
}
