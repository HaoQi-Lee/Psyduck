// Package spec parses psyduck SPEC.md files and detects drift between a
// spec's declared file list and the files git actually tracks.
package spec

import (
	"bufio"
	"bytes"
	"path/filepath"
	"regexp"
	"strings"
)

// Spec holds the fields parsed from a SPEC.md that psy check cares about.
type Spec struct {
	Path            string   // repo-root-relative path of the SPEC.md
	Package         string   // front-matter package: value
	HasPackage      bool     // whether a package: field was present
	PkgDir          string   // repo-root-relative dir containing the SPEC.md
	Files           []string // # 文件 entries, pkg-dir-relative, slash, cleaned
	HasFilesSection bool     // whether a # 文件 / # Files heading was found
}

var packageRe = regexp.MustCompile(`(?m)^\s*package:\s*(.+?)\s*$`)

// Parse extracts the package front-matter value and the # 文件 file list from a
// SPEC.md. path is the repo-root-relative path (used to derive PkgDir).
// Parsing is intentionally line-based — only package: and file bullets are
// needed, so no YAML dependency.
func Parse(path string, data []byte) Spec {
	s := Spec{Path: path, PkgDir: dirOf(path)}
	if m := packageRe.FindSubmatch(data); m != nil {
		s.HasPackage = true
		s.Package = trimQuotes(string(m[1]))
	}
	s.Files, s.HasFilesSection = parseFilesSection(data)
	return s
}

func dirOf(path string) string {
	if i := strings.LastIndexByte(path, '/'); i >= 0 {
		return path[:i]
	}
	return ""
}

func trimQuotes(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 {
		first, last := s[0], s[len(s)-1]
		if (first == '"' && last == '"') || (first == '\'' && last == '\'') {
			return s[1 : len(s)-1]
		}
	}
	return s
}

// parseFilesSection scans for a # 文件 / # Files heading and collects the
// backtick-quoted path of each "- " bullet until the next heading.
func parseFilesSection(data []byte) (files []string, found bool) {
	sc := bufio.NewScanner(bytes.NewReader(data))
	inSection := false
	seen := map[string]bool{}
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if isHeading(line) {
			if inSection {
				return files, found // next heading ends the section
			}
			if isFilesHeading(line) {
				inSection = true
				found = true
			}
			continue
		}
		if inSection && strings.HasPrefix(line, "- ") {
			if p := bulletPath(line); p != "" {
				clean := filepath.ToSlash(filepath.Clean(p))
				if !seen[clean] {
					seen[clean] = true
					files = append(files, clean)
				}
			}
		}
	}
	return files, found
}

func isHeading(line string) bool { return strings.HasPrefix(line, "#") }

func isFilesHeading(line string) bool {
	if !isHeading(line) {
		return false
	}
	t := strings.TrimSpace(strings.TrimLeft(line, "#"))
	return t == "文件" || t == "Files"
}

// bulletPath extracts the first backtick-quoted token from a bullet line like
// "- `root.go` — desc". Returns "" if none.
func bulletPath(line string) string {
	i := strings.IndexByte(line, '`')
	if i < 0 {
		return ""
	}
	rest := line[i+1:]
	j := strings.IndexByte(rest, '`')
	if j < 0 {
		return ""
	}
	return rest[:j]
}
