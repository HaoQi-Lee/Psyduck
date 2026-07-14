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
		{Added: []string{"a"}},     // drift
		{Timing: []TimingHint{{}}}, // timing only, no drift
		{},                         // clean
	}}
	require.Equal(t, 1, r.DriftCount())
	require.Equal(t, 1, r.TimingCount())
}

func TestPackageReport_HasDrift(t *testing.T) {
	require.True(t, PackageReport{PackageMismatch: true}.HasDrift())
	require.True(t, PackageReport{MissingFileSection: true}.HasDrift())
	require.True(t, PackageReport{Added: []string{"x"}}.HasDrift())
	require.True(t, PackageReport{Removed: []string{"x"}}.HasDrift())
	require.False(t, PackageReport{SpecUntracked: true}.HasDrift())
	require.False(t, PackageReport{}.HasDrift())
	require.False(t, PackageReport{Timing: []TimingHint{{}}}.HasDrift())
}

// fakeVCS is an in-memory VCS for testing checkWith without real git. Diff
// entries (diffs) and commit hashes (commits) are repo-root-relative; checkPackage
// relativizes them to the package dir before classifying.
type fakeVCS struct {
	files     map[string][]string   // relDir -> files (repo-root-relative)
	times     map[string]time.Time  // relPath -> time (absent = no history)
	commits   map[string]string     // relPath -> commit hash (absent = no history)
	diffs     map[string][]NameStatus // relDir -> changes since anchor (repo-root-relative)
	listErr   error                 // if set, ListFiles returns this error
	timeErr   error                 // if set, LastCommitTime returns this error
	commitErr error                 // if set, LastCommit returns this error
	diffErr   error                 // if set, DiffNameStatus returns this error
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
func (f *fakeVCS) LastCommit(relPath string) (string, bool, error) {
	if f.commitErr != nil {
		return "", false, f.commitErr
	}
	h, ok := f.commits[relPath]
	return h, ok, nil
}
func (f *fakeVCS) DiffNameStatus(fromCommit, relDir string) ([]NameStatus, error) {
	if f.diffErr != nil {
		return nil, f.diffErr
	}
	return f.diffs[relDir], nil
}

func TestCheck_DriftAddedAndRemoved(t *testing.T) {
	root := t.TempDir()
	// SPEC lists root.go and old.go; since sync, new.go was added and old.go deleted.
	writeSpec(t, root, "pkg", "pkg", "- `root.go` — r\n- `old.go` — o\n")
	writeFile(t, filepath.Join(root, "pkg", "root.go"), "package pkg\n")
	v := &fakeVCS{
		files:   map[string][]string{"": {"pkg/SPEC.md"}},
		commits: map[string]string{"pkg/SPEC.md": "c1"},
		diffs: map[string][]NameStatus{"pkg": {
			{Status: "A", Path: "pkg/new.go"},
			{Status: "D", Path: "pkg/old.go"},
		}},
	}
	rep, err := checkWith(root, v)
	require.NoError(t, err)
	require.Len(t, rep.Packages, 1)
	pr := rep.Packages[0]
	require.Equal(t, []string{"new.go"}, pr.Added)
	require.Equal(t, []string{"old.go"}, pr.Removed)
	require.True(t, pr.HasDrift())
	require.Equal(t, 1, rep.DriftCount())
}

func TestCheck_CleanPackage(t *testing.T) {
	root := t.TempDir()
	writeSpec(t, root, "pkg", "pkg", "- `root.go` — r\n")
	writeFile(t, filepath.Join(root, "pkg", "root.go"), "package pkg\n")
	v := &fakeVCS{
		files:   map[string][]string{"": {"pkg/SPEC.md"}},
		commits: map[string]string{"pkg/SPEC.md": "c1"},
		diffs:   map[string][]NameStatus{}, // no changes since sync
	}
	rep, err := checkWith(root, v)
	require.NoError(t, err)
	require.Len(t, rep.Packages, 1)
	require.False(t, rep.Packages[0].HasDrift())
	require.Empty(t, rep.Packages[0].Timing)
}

func TestCheck_ResourceTypeAddedIgnored(t *testing.T) {
	root := t.TempDir()
	// Since sync a resource file (png) and a config (yaml) were added; their
	// types are outside the SPEC's .go vocabulary, so not drift.
	writeSpec(t, root, "pkg", "pkg", "- `root.go` — r\n")
	writeFile(t, filepath.Join(root, "pkg", "root.go"), "package pkg\n")
	v := &fakeVCS{
		files:   map[string][]string{"": {"pkg/SPEC.md"}},
		commits: map[string]string{"pkg/SPEC.md": "c1"},
		diffs: map[string][]NameStatus{"pkg": {
			{Status: "A", Path: "pkg/assets/logo.png"},
			{Status: "A", Path: "pkg/app.yaml"},
		}},
	}
	rep, err := checkWith(root, v)
	require.NoError(t, err)
	pr := rep.Packages[0]
	require.Empty(t, pr.Added, "resource types are outside the declared vocabulary")
	require.False(t, pr.HasDrift())
}

func TestCheck_DeclaredTypeAddedFlagged(t *testing.T) {
	root := t.TempDir()
	// SPEC declares .md (via skills/embed.md); an unlisted .md added since sync
	// is drift within the declared vocabulary.
	writeSpec(t, root, "pkg", "pkg", "- `root.go` — r\n- `skills/embed.md` — e\n")
	writeFile(t, filepath.Join(root, "pkg", "root.go"), "package pkg\n")
	v := &fakeVCS{
		files:   map[string][]string{"": {"pkg/SPEC.md"}},
		commits: map[string]string{"pkg/SPEC.md": "c1"},
		diffs: map[string][]NameStatus{"pkg": {
			{Status: "A", Path: "pkg/skills/extra.md"},
		}},
	}
	rep, err := checkWith(root, v)
	require.NoError(t, err)
	require.Equal(t, []string{"skills/extra.md"}, rep.Packages[0].Added)
}

func TestCheck_NestedPackageExcluded(t *testing.T) {
	root := t.TempDir()
	writeSpec(t, root, "pkg", "pkg", "- `parent.go` — p\n")
	writeFile(t, filepath.Join(root, "pkg", "parent.go"), "package pkg\n")
	writeSpec(t, root, "pkg/child", "pkg/child", "- `child.go` — c\n")
	writeFile(t, filepath.Join(root, "pkg", "child", "child.go"), "package child\n")
	v := &fakeVCS{
		files:   map[string][]string{"": {"pkg/SPEC.md", "pkg/child/SPEC.md"}},
		commits: map[string]string{"pkg/SPEC.md": "c1", "pkg/child/SPEC.md": "c2"},
		diffs: map[string][]NameStatus{
			"pkg":       {{Status: "M", Path: "pkg/child/child.go"}}, // must be filtered from parent
			"pkg/child": {},
		},
	}
	rep, err := checkWith(root, v)
	require.NoError(t, err)
	require.Len(t, rep.Packages, 2)
	byDir := map[string]PackageReport{}
	for _, p := range rep.Packages {
		byDir[p.PkgDir] = p
	}
	require.False(t, byDir["pkg"].HasDrift(), "parent must not see child files")
	require.Empty(t, byDir["pkg"].Timing, "child-subtree changes are not parent staleness")
	require.Empty(t, byDir["pkg"].Added)
	require.False(t, byDir["pkg/child"].HasDrift())
}

func TestCheck_PackageMismatch(t *testing.T) {
	root := t.TempDir()
	writeSpec(t, root, "pkg", "wrong/path", "- `root.go` — r\n")
	writeFile(t, filepath.Join(root, "pkg", "root.go"), "package pkg\n")
	v := &fakeVCS{
		files:   map[string][]string{"": {"pkg/SPEC.md"}},
		commits: map[string]string{"pkg/SPEC.md": "c1"},
	}
	rep, err := checkWith(root, v)
	require.NoError(t, err)
	require.True(t, rep.Packages[0].PackageMismatch)
}

func TestCheck_MissingFilesSection(t *testing.T) {
	root := t.TempDir()
	writeSpec(t, root, "pkg", "pkg", "") // no # 文件
	writeFile(t, filepath.Join(root, "pkg", "root.go"), "package pkg\n")
	v := &fakeVCS{
		files:   map[string][]string{"": {"pkg/SPEC.md"}},
		commits: map[string]string{"pkg/SPEC.md": "c1"},
	}
	rep, err := checkWith(root, v)
	require.NoError(t, err)
	require.True(t, rep.Packages[0].MissingFileSection)
	require.Empty(t, rep.Packages[0].Added)
	require.Empty(t, rep.Packages[0].Removed)
}

func TestCheck_TimingHint(t *testing.T) {
	root := t.TempDir()
	writeSpec(t, root, "pkg", "pkg", "- `root.go` — r\n")
	writeFile(t, filepath.Join(root, "pkg", "root.go"), "package pkg\n")
	specT := time.Unix(1_700_000_000, 0) // older
	fileT := time.Unix(1_710_000_000, 0) // newer
	v := &fakeVCS{
		files:   map[string][]string{"": {"pkg/SPEC.md"}},
		commits: map[string]string{"pkg/SPEC.md": "c1"},
		diffs:   map[string][]NameStatus{"pkg": {{Status: "M", Path: "pkg/root.go"}}},
		times:   map[string]time.Time{"pkg/SPEC.md": specT, "pkg/root.go": fileT},
	}
	rep, err := checkWith(root, v)
	require.NoError(t, err)
	require.Len(t, rep.Packages[0].Timing, 1)
	require.Equal(t, "root.go", rep.Packages[0].Timing[0].File)
	require.Equal(t, 1, rep.TimingCount())
	require.False(t, rep.Packages[0].HasDrift())
	require.Equal(t, 0, rep.DriftCount())
}

func TestCheck_SpecUntracked(t *testing.T) {
	root := t.TempDir()
	writeSpec(t, root, "pkg", "pkg", "- `root.go` — r\n")
	writeFile(t, filepath.Join(root, "pkg", "root.go"), "package pkg\n")
	// SPEC has no commit history: drift cannot be measured, so nothing is reported.
	v := &fakeVCS{
		files:   map[string][]string{"": {"pkg/SPEC.md"}},
		commits: map[string]string{},
	}
	rep, err := checkWith(root, v)
	require.NoError(t, err)
	pr := rep.Packages[0]
	require.True(t, pr.SpecUntracked)
	require.Empty(t, pr.Added)
	require.Empty(t, pr.Timing)
	require.False(t, pr.HasDrift())
}

func TestCheck_ListFilesErrorPropagated(t *testing.T) {
	root := t.TempDir()
	v := &fakeVCS{listErr: errors.New("list boom")}
	_, err := checkWith(root, v)
	require.Error(t, err)
	require.Contains(t, err.Error(), "list boom")
}

func TestCheck_LastCommitErrorPropagated(t *testing.T) {
	root := t.TempDir()
	writeSpec(t, root, "pkg", "pkg", "- `root.go` — r\n")
	writeFile(t, filepath.Join(root, "pkg", "root.go"), "package pkg\n")
	v := &fakeVCS{
		files:     map[string][]string{"": {"pkg/SPEC.md"}},
		commitErr: errors.New("commit boom"),
	}
	_, err := checkWith(root, v)
	require.Error(t, err)
	require.Contains(t, err.Error(), "commit boom")
}

func TestCheck_DiffNameStatusErrorPropagated(t *testing.T) {
	root := t.TempDir()
	writeSpec(t, root, "pkg", "pkg", "- `root.go` — r\n")
	writeFile(t, filepath.Join(root, "pkg", "root.go"), "package pkg\n")
	v := &fakeVCS{
		files:   map[string][]string{"": {"pkg/SPEC.md"}},
		commits: map[string]string{"pkg/SPEC.md": "c1"},
		diffErr: errors.New("diff boom"),
	}
	_, err := checkWith(root, v)
	require.Error(t, err)
	require.Contains(t, err.Error(), "diff boom")
}

func TestCheck_LastCommitTimeErrorPropagated(t *testing.T) {
	root := t.TempDir()
	writeSpec(t, root, "pkg", "pkg", "- `root.go` — r\n")
	writeFile(t, filepath.Join(root, "pkg", "root.go"), "package pkg\n")
	v := &fakeVCS{
		files:   map[string][]string{"": {"pkg/SPEC.md"}},
		commits: map[string]string{"pkg/SPEC.md": "c1"},
		timeErr: errors.New("time boom"),
	}
	_, err := checkWith(root, v)
	require.Error(t, err)
	require.Contains(t, err.Error(), "time boom")
}

func TestCheck_RootSpecIgnored(t *testing.T) {
	root := t.TempDir()
	// A SPEC.md at the repo root is intentionally not a package: its PkgDir
	// would be "" and DiffNameStatus(anchor, "") would diff the whole repo.
	writeSpec(t, root, "", "root", "- `main.go` — m\n")
	writeFile(t, filepath.Join(root, "main.go"), "package main\n")
	writeSpec(t, root, "pkg", "pkg", "- `root.go` — r\n")
	writeFile(t, filepath.Join(root, "pkg", "root.go"), "package pkg\n")
	v := &fakeVCS{
		files:   map[string][]string{"": {"SPEC.md", "main.go", "pkg/SPEC.md"}},
		commits: map[string]string{"pkg/SPEC.md": "c1"},
	}
	rep, err := checkWith(root, v)
	require.NoError(t, err)
	require.Len(t, rep.Packages, 1, "root SPEC.md must be ignored")
	require.Equal(t, "pkg", rep.Packages[0].PkgDir)
	require.False(t, rep.Packages[0].HasDrift())
}

func TestCheck_NonCodeDirsExcluded(t *testing.T) {
	root := t.TempDir()
	writeSpec(t, root, "pkg", "pkg", "- `root.go` — r\n")
	writeFile(t, filepath.Join(root, "pkg", "root.go"), "package pkg\n")
	// Changes under non-code dirs are never package source and must be filtered.
	v := &fakeVCS{
		files:   map[string][]string{"": {"pkg/SPEC.md"}},
		commits: map[string]string{"pkg/SPEC.md": "c1"},
		diffs: map[string][]NameStatus{"pkg": {
			{Status: "A", Path: "pkg/testdata/fixture.txt"},
			{Status: "A", Path: "pkg/.idea/config.xml"},
			{Status: "A", Path: "pkg/vendor/lib.go"},   // .go, but under vendor/
			{Status: "A", Path: "pkg/node_modules/mod/index.js"},
		}},
	}
	rep, err := checkWith(root, v)
	require.NoError(t, err)
	pr := rep.Packages[0]
	require.Empty(t, pr.Added, "non-code dirs must be excluded")
	require.Empty(t, pr.Timing)
	require.False(t, pr.HasDrift())
}
