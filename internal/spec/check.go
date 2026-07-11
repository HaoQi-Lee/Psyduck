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
	ListedButGone      []string     // declared but not on disk (pkg-relative)
	Undocumented       []string     // on disk but not declared (pkg-relative)
	Timing             []TimingHint // source newer than spec (advisory)
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
		len(p.ListedButGone) > 0 || len(p.Undocumented) > 0
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
// read-only. Equivalent to checkWith with a real gitVCS.
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
		if path.Base(f) == "SPEC.md" {
			specPaths = append(specPaths, f)
			specDirs[path.Dir(f)] = true
		}
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

	actualRel, err := actualFiles(sp.PkgDir, specPath, specDirs, vcs)
	if err != nil {
		return pr, err
	}

	if sp.HasFilesSection {
		declared := toSet(sp.Files)
		for f := range declared {
			if !actualRel[f] {
				pr.ListedButGone = append(pr.ListedButGone, f)
			}
		}
		for f := range actualRel {
			if !declared[f] {
				pr.Undocumented = append(pr.Undocumented, f)
			}
		}
		sort.Strings(pr.ListedButGone)
		sort.Strings(pr.Undocumented)
	} else {
		pr.MissingFileSection = true
	}
	return pr, nil
}

// actualFiles returns the set of pkg-dir-relative files tracked under pkgDir,
// excluding the SPEC.md itself and nested-package subtrees.
func actualFiles(pkgDir, specPath string, specDirs map[string]bool, vcs VCS) (map[string]bool, error) {
	listed, err := vcs.ListFiles(pkgDir)
	if err != nil {
		return nil, err
	}
	pkgPrefix := pkgDir
	if pkgPrefix != "" {
		pkgPrefix += "/"
	}
	out := map[string]bool{}
	for _, f := range listed {
		if f == specPath {
			continue
		}
		if inNestedSpecDir(f, specPath, specDirs) {
			continue
		}
		out[strings.TrimPrefix(f, pkgPrefix)] = true
	}
	return out, nil
}

func toSet(xs []string) map[string]bool {
	m := map[string]bool{}
	for _, x := range xs {
		m[x] = true
	}
	return m
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
