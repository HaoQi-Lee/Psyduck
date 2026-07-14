package spec

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Report is the result of checking a whole repository.
type Report struct {
	Root     string
	Packages []PackageReport
}

// PackageReport is the result of checking one SPEC.md.
type PackageReport struct {
	Package            string       // front-matter package: value
	SpecPath           string       // repo-root-relative SPEC.md path
	PkgDir             string       // repo-root-relative dir of the SPEC.md
	PackageMismatch    bool         // package: value != PkgDir (or absent)
	MissingFileSection bool         // no # 文件 heading found
	Added              []string     // source-type files added since sync, unlisted (pkg-relative)
	Removed            []string     // files deleted since sync, still listed (pkg-relative)
	SpecUntracked      bool         // SPEC.md has no commit history; drift not measurable
	Timing             []TimingHint // modified since sync (advisory)
}

// TimingHint flags a source file whose latest commit is newer than the spec's.
type TimingHint struct {
	File     string
	FileTime time.Time
	SpecTime time.Time
}

// HasDrift reports any structural drift (excludes timing, which is advisory).
func (p PackageReport) HasDrift() bool {
	return p.PackageMismatch || p.MissingFileSection ||
		len(p.Added) > 0 || len(p.Removed) > 0
}

// DriftCount is the number of packages with structural drift.
func (r Report) DriftCount() int {
	n := 0
	for _, p := range r.Packages {
		if p.HasDrift() {
			n++
		}
	}
	return n
}

// TimingCount is the total number of timing hints across all packages.
func (r Report) TimingCount() int {
	n := 0
	for _, p := range r.Packages {
		n += len(p.Timing)
	}
	return n
}

// Check scans repoRoot for every SPEC.md and reports drift against git. It is
// read-only. Equivalent to checkWith with a real gitVCS. Check aborts on the
// first per-package error (fail-fast).
func Check(repoRoot string) (Report, error) {
	return checkWith(repoRoot, newGitVCS(repoRoot))
}

// checkWith is the testable core; vcs is injected so logic can be exercised
// without a real git repository.
func checkWith(repoRoot string, vcs VCS) (Report, error) {
	rep := Report{Root: repoRoot}
	all, err := vcs.ListFiles("")
	if err != nil {
		return rep, err
	}
	var specPaths []string
	specDirs := map[string]bool{}
	for _, f := range all {
		if path.Base(f) != "SPEC.md" {
			continue
		}
		// Root is never a package: a SPEC.md at the repo root has PkgDir "" and
		// DiffNameStatus(anchor, "") would diff the whole repo. Skip it.
		dir := path.Dir(f)
		if dir == "." || dir == "" {
			continue
		}
		specPaths = append(specPaths, f)
		specDirs[dir] = true
	}
	for _, sp := range specPaths {
		pr, err := checkPackage(repoRoot, sp, vcs, specDirs)
		if err != nil {
			return rep, err
		}
		rep.Packages = append(rep.Packages, pr)
	}
	sort.Slice(rep.Packages, func(i, j int) bool {
		return rep.Packages[i].SpecPath < rep.Packages[j].SpecPath
	})
	return rep, nil
}

func checkPackage(repoRoot, specPath string, vcs VCS, specDirs map[string]bool) (PackageReport, error) {
	data, err := os.ReadFile(filepath.Join(repoRoot, filepath.FromSlash(specPath)))
	if err != nil {
		return PackageReport{}, fmt.Errorf("read %s: %w", specPath, err)
	}
	sp := Parse(specPath, data)
	pr := PackageReport{
		Package:  sp.Package,
		SpecPath: specPath,
		PkgDir:   sp.PkgDir,
	}
	if !sp.HasPackage || sp.Package != sp.PkgDir {
		pr.PackageMismatch = true
	}
	if !sp.HasFilesSection {
		// Without # 文件 there is no declared set or type vocabulary to compare
		// against; this is itself drift (the SPEC is incomplete).
		pr.MissingFileSection = true
		return pr, nil
	}

	// Anchor = the SPEC's last commit. Only changes after it are post-sync.
	anchor, ok, err := vcs.LastCommit(specPath)
	if err != nil {
		return pr, err
	}
	if !ok {
		// SPEC uncommitted: cannot measure post-sync changes. Not drift.
		pr.SpecUntracked = true
		return pr, nil
	}

	changes, err := vcs.DiffNameStatus(anchor, sp.PkgDir)
	if err != nil {
		return pr, err
	}

	pkgPrefix := sp.PkgDir
	if pkgPrefix != "" {
		pkgPrefix += "/"
	}
	// Filter diff entries (SPEC.md itself, nested-package subtrees, non-code
	// dirs) and relativize to the package directory before classifying.
	var rel []NameStatus
	for _, c := range changes {
		if isExcluded(c.Path, specPath, specDirs) {
			continue
		}
		rel = append(rel, NameStatus{
			Status:  c.Status,
			Path:    strings.TrimPrefix(c.Path, pkgPrefix),
			OldPath: strings.TrimPrefix(c.OldPath, pkgPrefix),
		})
	}

	added, removed, modified := classify(rel, toSet(sp.Files), typeKeys(sp.Files))
	pr.Added = added
	pr.Removed = removed

	// Stale: files modified since sync (advisory, never affects exit code).
	specTime, specOK, err := vcs.LastCommitTime(specPath)
	if err != nil {
		return pr, err
	}
	if specOK {
		for _, f := range modified {
			ft, mok, err := vcs.LastCommitTime(pkgPrefix + f)
			if err != nil {
				return pr, err
			}
			if mok && ft.After(specTime) {
				pr.Timing = append(pr.Timing, TimingHint{File: f, FileTime: ft, SpecTime: specTime})
			}
		}
		sort.Slice(pr.Timing, func(i, j int) bool { return pr.Timing[i].File < pr.Timing[j].File })
	}
	return pr, nil
}

// isExcluded reports whether a repo-root-relative diff path should be ignored:
// the SPEC.md itself, a file under a non-code dir, or a file in a nested
// package subtree (which belongs to the child SPEC, not this one).
func isExcluded(repoRel, specPath string, specDirs map[string]bool) bool {
	if repoRel == specPath {
		return true
	}
	if isNonCode(repoRel) {
		return true
	}
	return inNestedSpecDir(repoRel, specPath, specDirs)
}

// nonCodeDirs are directory names that never constitute package source: test
// fixtures (testdata) and dependency trees (vendor, node_modules). A file
// anywhere under such a directory is excluded from the actual-file set, as is
// anything under a dot-directory (.idea, .git, .vscode, ...). Only directory
// segments are considered, so dot-files like .gitignore are kept.
var nonCodeDirs = map[string]bool{
	"testdata":     true,
	"vendor":       true,
	"node_modules": true,
}

// isNonCode reports whether repoRelPath lives under a non-code or dot directory.
func isNonCode(repoRelPath string) bool {
	segs := strings.Split(repoRelPath, "/")
	for _, seg := range segs[:len(segs)-1] { // directory segments only
		if nonCodeDirs[seg] || strings.HasPrefix(seg, ".") {
			return true
		}
	}
	return false
}

func toSet(xs []string) map[string]bool {
	m := map[string]bool{}
	for _, x := range xs {
		m[x] = true
	}
	return m
}

// typeKey is the lower-cased extension (with leading dot) of a path, or "" if
// it has none. It classifies a file's type for drift scoping: "logo.PNG" ->
// ".png", "Makefile" -> "".
func typeKey(p string) string {
	return strings.ToLower(filepath.Ext(p))
}

// typeKeys is the set of file-type keys among the declared files. Undocumented
// reporting is restricted to files whose type the SPEC already declares, so a
// package's resource files (icons, configs, locales, ...) — of types its # 文件
// never mentions — are not drift, just outside the declared vocabulary.
func typeKeys(files []string) map[string]bool {
	m := map[string]bool{}
	for _, f := range files {
		m[typeKey(f)] = true
	}
	return m
}

// classify sorts the net changes since the SPEC's last commit (pkg-dir-relative)
// into drift and staleness, given the SPEC's declared file set and its type
// vocabulary:
//   - Added  (drift): a source-type file (type in vocab) added since sync that
//     the SPEC does not list. Resource files of unlisted types are ignored.
//   - Removed (drift): a file deleted since sync that the SPEC still lists.
//   - Modified (stale): any content change (M), type change (T), or unknown
//     status — something changed, the SPEC's prose may be stale.
//
// A rename (R) splits into its old path (removed if declared) and new path
// (added if source-type and unlisted); a copy (C) is treated as an add of the
// new path. Each returned slice is sorted and deduped.
func classify(changes []NameStatus, declared, vocab map[string]bool) (added, removed, modified []string) {
	seenAdd, seenRem, seenMod := map[string]bool{}, map[string]bool{}, map[string]bool{}
	push := func(dst *[]string, seen map[string]bool, s string) {
		if s == "" || seen[s] {
			return
		}
		seen[s] = true
		*dst = append(*dst, s)
	}
	for _, c := range changes {
		switch c.Status {
		case "A", "C":
			if vocab[typeKey(c.Path)] && !declared[c.Path] {
				push(&added, seenAdd, c.Path)
			}
		case "D":
			if declared[c.Path] {
				push(&removed, seenRem, c.Path)
			}
		case "R":
			if declared[c.OldPath] {
				push(&removed, seenRem, c.OldPath)
			}
			if vocab[typeKey(c.Path)] && !declared[c.Path] {
				push(&added, seenAdd, c.Path)
			}
		default: // M, T, and any unknown status -> stale (conservative)
			push(&modified, seenMod, c.Path)
		}
	}
	sort.Strings(added)
	sort.Strings(removed)
	sort.Strings(modified)
	return added, removed, modified
}

// inNestedSpecDir reports whether file f lives under a proper subdirectory of
// specPath's dir that itself contains a SPEC.md (a nested package).
func inNestedSpecDir(f, specPath string, specDirs map[string]bool) bool {
	specDir := path.Dir(specPath)
	d := path.Dir(f)
	for d != specDir && d != "." && d != "" {
		if specDirs[d] {
			return true
		}
		parent := path.Dir(d)
		if parent == d {
			break
		}
		d = parent
	}
	return false
}
