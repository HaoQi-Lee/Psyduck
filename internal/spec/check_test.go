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

func TestCheck_ModifiedFileNoCommitTime(t *testing.T) {
	// A modified file whose LastCommitTime returns ok=false (no history) is
	// silently skipped — no Timing entry, no error.
	root := t.TempDir()
	writeSpec(t, root, "pkg", "pkg", "- `root.go` — r\n")
	writeFile(t, filepath.Join(root, "pkg", "root.go"), "package pkg\n")
	v := &fakeVCS{
		files:   map[string][]string{"": {"pkg/SPEC.md"}},
		commits: map[string]string{"pkg/SPEC.md": "c1"},
		diffs:   map[string][]NameStatus{"pkg": {{Status: "M", Path: "pkg/root.go"}}},
		times:   map[string]time.Time{"pkg/SPEC.md": time.Unix(1_700_000_000, 0)}, // root.go absent -> ok=false
	}
	rep, err := checkWith(root, v)
	require.NoError(t, err)
	require.Empty(t, rep.Packages[0].Timing)
	require.False(t, rep.Packages[0].HasDrift())
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

func TestCheck_MultiPackageMixedStates(t *testing.T) {
	root := t.TempDir()
	// pkgA clean; pkgB drift (added); pkgC stale (modified); pkgD untracked SPEC.
	writeSpec(t, root, "pkgA", "pkgA", "- `a.go` — a\n")
	writeFile(t, filepath.Join(root, "pkgA", "a.go"), "package pkgA\n")
	writeSpec(t, root, "pkgB", "pkgB", "- `b.go` — b\n")
	writeFile(t, filepath.Join(root, "pkgB", "b.go"), "package pkgB\n")
	writeSpec(t, root, "pkgC", "pkgC", "- `c.go` — c\n")
	writeFile(t, filepath.Join(root, "pkgC", "c.go"), "package pkgC\n")
	writeSpec(t, root, "pkgD", "pkgD", "- `d.go` — d\n")
	writeFile(t, filepath.Join(root, "pkgD", "d.go"), "package pkgD\n")
	v := &fakeVCS{
		files: map[string][]string{"": {"pkgA/SPEC.md", "pkgB/SPEC.md", "pkgC/SPEC.md", "pkgD/SPEC.md"}},
		commits: map[string]string{"pkgA/SPEC.md": "a", "pkgB/SPEC.md": "b", "pkgC/SPEC.md": "c"},
		diffs: map[string][]NameStatus{
			"pkgB": {{Status: "A", Path: "pkgB/new.go"}},
			"pkgC": {{Status: "M", Path: "pkgC/c.go"}},
		},
		times: map[string]time.Time{
			"pkgC/SPEC.md": time.Unix(1_700_000_000, 0),
			"pkgC/c.go":    time.Unix(1_710_000_000, 0),
		},
	}
	rep, err := checkWith(root, v)
	require.NoError(t, err)
	require.Len(t, rep.Packages, 4)
	byDir := map[string]PackageReport{}
	for _, p := range rep.Packages {
		byDir[p.PkgDir] = p
	}
	require.False(t, byDir["pkgA"].HasDrift())
	require.True(t, byDir["pkgB"].HasDrift())
	require.Equal(t, []string{"new.go"}, byDir["pkgB"].Added)
	require.False(t, byDir["pkgC"].HasDrift())
	require.Len(t, byDir["pkgC"].Timing, 1)
	require.True(t, byDir["pkgD"].SpecUntracked)
	require.False(t, byDir["pkgD"].HasDrift())
	require.Equal(t, 1, rep.DriftCount())
	require.Equal(t, 1, rep.TimingCount())
}

func TestCheck_PackagesSortedBySpecPath(t *testing.T) {
	root := t.TempDir()
	for _, d := range []string{"zeta", "alpha", "mid"} {
		writeSpec(t, root, d, d, "- `x.go` — x\n")
		writeFile(t, filepath.Join(root, d, "x.go"), "package "+d+"\n")
	}
	v := &fakeVCS{
		files:   map[string][]string{"": {"zeta/SPEC.md", "alpha/SPEC.md", "mid/SPEC.md"}},
		commits: map[string]string{"zeta/SPEC.md": "1", "alpha/SPEC.md": "1", "mid/SPEC.md": "1"},
	}
	rep, err := checkWith(root, v)
	require.NoError(t, err)
	require.Len(t, rep.Packages, 3)
	require.Equal(t, []string{"alpha", "mid", "zeta"}, []string{
		rep.Packages[0].PkgDir, rep.Packages[1].PkgDir, rep.Packages[2].PkgDir,
	})
}

func TestCheck_PackageMismatchAndDriftCoexist(t *testing.T) {
	root := t.TempDir()
	writeSpec(t, root, "pkg", "wrong/path", "- `root.go` — r\n")
	writeFile(t, filepath.Join(root, "pkg", "root.go"), "package pkg\n")
	v := &fakeVCS{
		files:   map[string][]string{"": {"pkg/SPEC.md"}},
		commits: map[string]string{"pkg/SPEC.md": "c1"},
		diffs:   map[string][]NameStatus{"pkg": {{Status: "A", Path: "pkg/new.go"}}},
	}
	rep, err := checkWith(root, v)
	require.NoError(t, err)
	pr := rep.Packages[0]
	require.True(t, pr.PackageMismatch)
	require.Equal(t, []string{"new.go"}, pr.Added)
	require.True(t, pr.HasDrift())
}

func TestCheck_MissingFilesSectionBeforeAnchor(t *testing.T) {
	root := t.TempDir()
	writeSpec(t, root, "pkg", "pkg", "") // no # 文件
	writeFile(t, filepath.Join(root, "pkg", "root.go"), "package pkg\n")
	// SPEC is also untracked, but MissingFileSection returns before the anchor
	// lookup, so SpecUntracked must stay false.
	v := &fakeVCS{
		files:   map[string][]string{"": {"pkg/SPEC.md"}},
		commits: map[string]string{},
	}
	rep, err := checkWith(root, v)
	require.NoError(t, err)
	pr := rep.Packages[0]
	require.True(t, pr.MissingFileSection)
	require.False(t, pr.SpecUntracked, "MissingFileSection returns before the anchor check")
	require.True(t, pr.HasDrift())
}

func TestCheck_DeepNestedPackages(t *testing.T) {
	root := t.TempDir()
	writeSpec(t, root, "pkg", "pkg", "- `top.go` — t\n")
	writeFile(t, filepath.Join(root, "pkg", "top.go"), "package pkg\n")
	writeSpec(t, root, "pkg/mid", "pkg/mid", "- `mid.go` — m\n")
	writeFile(t, filepath.Join(root, "pkg", "mid", "mid.go"), "package mid\n")
	writeSpec(t, root, "pkg/mid/leaf", "pkg/mid/leaf", "- `leaf.go` — l\n")
	writeFile(t, filepath.Join(root, "pkg", "mid", "leaf", "leaf.go"), "package leaf\n")
	v := &fakeVCS{
		files:   map[string][]string{"": {"pkg/SPEC.md", "pkg/mid/SPEC.md", "pkg/mid/leaf/SPEC.md"}},
		commits: map[string]string{"pkg/SPEC.md": "1", "pkg/mid/SPEC.md": "2", "pkg/mid/leaf/SPEC.md": "3"},
		diffs: map[string][]NameStatus{
			"pkg":          {{Status: "A", Path: "pkg/mid/leaf/leaf.go"}},
			"pkg/mid":      {{Status: "A", Path: "pkg/mid/leaf/leaf.go"}},
			"pkg/mid/leaf": {},
		},
	}
	rep, err := checkWith(root, v)
	require.NoError(t, err)
	require.Len(t, rep.Packages, 3)
	byDir := map[string]PackageReport{}
	for _, p := range rep.Packages {
		byDir[p.PkgDir] = p
	}
	require.Empty(t, byDir["pkg"].Added, "top must not see deeply nested files")
	require.Empty(t, byDir["pkg/mid"].Added, "mid must not see leaf files")
	require.False(t, byDir["pkg/mid/leaf"].HasDrift())
}

func TestCheck_ModifiedAtSpecTimeNotStale(t *testing.T) {
	root := t.TempDir()
	writeSpec(t, root, "pkg", "pkg", "- `root.go` — r\n")
	writeFile(t, filepath.Join(root, "pkg", "root.go"), "package pkg\n")
	// root.go last commit time == spec commit time; strict After -> not stale.
	same := time.Unix(1_700_000_000, 0)
	v := &fakeVCS{
		files:   map[string][]string{"": {"pkg/SPEC.md"}},
		commits: map[string]string{"pkg/SPEC.md": "c1"},
		diffs:   map[string][]NameStatus{"pkg": {{Status: "M", Path: "pkg/root.go"}}},
		times:   map[string]time.Time{"pkg/SPEC.md": same, "pkg/root.go": same},
	}
	rep, err := checkWith(root, v)
	require.NoError(t, err)
	require.Empty(t, rep.Packages[0].Timing, "file commit == spec commit (not strictly after) -> not stale")
}

func TestCheck_TimingSortedByFile(t *testing.T) {
	root := t.TempDir()
	writeSpec(t, root, "pkg", "pkg", "- `a.go` — a\n")
	writeFile(t, filepath.Join(root, "pkg", "a.go"), "package pkg\n")
	specT := time.Unix(1_700_000_000, 0)
	v := &fakeVCS{
		files:   map[string][]string{"": {"pkg/SPEC.md"}},
		commits: map[string]string{"pkg/SPEC.md": "c1"},
		diffs: map[string][]NameStatus{"pkg": {
			{Status: "M", Path: "pkg/zeta.go"},
			{Status: "M", Path: "pkg/alpha.go"},
			{Status: "M", Path: "pkg/mid.go"},
		}},
		times: map[string]time.Time{
			"pkg/SPEC.md":  specT,
			"pkg/zeta.go":  specT.Add(time.Hour),
			"pkg/alpha.go": specT.Add(time.Hour),
			"pkg/mid.go":   specT.Add(time.Hour),
		},
	}
	rep, err := checkWith(root, v)
	require.NoError(t, err)
	var files []string
	for _, th := range rep.Packages[0].Timing {
		files = append(files, th.File)
	}
	require.Equal(t, []string{"alpha.go", "mid.go", "zeta.go"}, files)
}

func TestCheck_SpecItselfInDiffFiltered(t *testing.T) {
	root := t.TempDir()
	// SPEC declares .md (via skills/embed.md), so SPEC.md's own .md type IS in
	// vocab — only isExcluded (== specPath) keeps it out of stale/drift.
	writeSpec(t, root, "pkg", "pkg", "- `root.go` — r\n- `skills/embed.md` — e\n")
	writeFile(t, filepath.Join(root, "pkg", "root.go"), "package pkg\n")
	v := &fakeVCS{
		files:   map[string][]string{"": {"pkg/SPEC.md"}},
		commits: map[string]string{"pkg/SPEC.md": "c1"},
		diffs:   map[string][]NameStatus{"pkg": {{Status: "M", Path: "pkg/SPEC.md"}}},
		times:   map[string]time.Time{"pkg/SPEC.md": time.Unix(1_700_000_000, 0)},
	}
	rep, err := checkWith(root, v)
	require.NoError(t, err)
	pr := rep.Packages[0]
	require.Empty(t, pr.Added)
	require.Empty(t, pr.Removed)
	require.Empty(t, pr.Timing)
	require.False(t, pr.HasDrift())
}
