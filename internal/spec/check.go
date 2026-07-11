package spec

import "time"

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
