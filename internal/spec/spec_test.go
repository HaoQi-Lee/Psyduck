package spec

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParse_Package(t *testing.T) {
	data := []byte("---\npsy_kind: factual\npackage: internal/cli\ncreated: 2026-06-05\n---\n\n# 概述\n\nx\n")
	s := Parse("internal/cli/SPEC.md", data)
	require.True(t, s.HasPackage)
	require.Equal(t, "internal/cli", s.Package)
	require.Equal(t, "internal/cli", s.PkgDir)
}

func TestParse_PackageQuotedAndMissing(t *testing.T) {
	quoted := []byte("---\npackage: \"internal/version\"\n---\n")
	require.Equal(t, "internal/version", Parse("a/b/SPEC.md", quoted).Package)

	none := []byte("---\npsy_kind: factual\n---\n")
	s := Parse("a/SPEC.md", none)
	require.False(t, s.HasPackage)
	require.Equal(t, "", s.Package)
}

func TestParse_FilesSection(t *testing.T) {
	data := []byte("---\npackage: pkg\n---\n\n# 概述\n\nx\n\n# 文件\n\n- `root.go` — 根\n- `claudemd/section.md` — 片段\n\n# API\n\n- x\n")
	s := Parse("pkg/SPEC.md", data)
	require.True(t, s.HasFilesSection)
	require.Equal(t, []string{"root.go", "claudemd/section.md"}, s.Files)
}

func TestParse_FilesSectionEnglishFallback(t *testing.T) {
	data := []byte("---\npackage: pkg\n---\n\n# Files\n\n- `a.go` — a\n")
	s := Parse("pkg/SPEC.md", data)
	require.True(t, s.HasFilesSection)
	require.Equal(t, []string{"a.go"}, s.Files)
}

func TestParse_NoFilesSection(t *testing.T) {
	data := []byte("---\npackage: pkg\n---\n\n# 概述\n\nx\n\n# API\n\nx\n")
	s := Parse("pkg/SPEC.md", data)
	require.False(t, s.HasFilesSection)
	require.Empty(t, s.Files)
}

func TestParse_RootLevelSpecPkgDir(t *testing.T) {
	s := Parse("SPEC.md", []byte("---\npackage: ''\n---\n"))
	require.Equal(t, "", s.PkgDir)
}

func TestParse_FilesHeadingVariants(t *testing.T) {
	cases := map[string]string{
		"double-hash":   "---\npackage: p\n---\n\n## 文件\n\n- `a.go` — a\n",
		"no-space":      "---\npackage: p\n---\n\n#文件\n\n- `a.go` — a\n",
		"trailing-space": "---\npackage: p\n---\n\n# 文件 \n\n- `a.go` — a\n",
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			s := Parse("p/SPEC.md", []byte(body))
			require.True(t, s.HasFilesSection, name)
			require.Equal(t, []string{"a.go"}, s.Files)
		})
	}
}

func TestParse_FilesHeadingPrefixNoMatch(t *testing.T) {
	// "# 文件概述" is not the 文件 heading; bullets after it are not collected.
	data := []byte("---\npackage: p\n---\n\n# 文件概述\n\n- `a.go` — a\n")
	s := Parse("p/SPEC.md", data)
	require.False(t, s.HasFilesSection)
	require.Empty(t, s.Files)
}

func TestParse_CRLFLineEndings(t *testing.T) {
	data := []byte("---\r\npackage: p\r\n---\r\n\r\n# 文件\r\n\r\n- `a.go` — a\r\n- `b.go` — b\r\n")
	s := Parse("p/SPEC.md", data)
	require.True(t, s.HasPackage)
	require.Equal(t, "p", s.Package)
	require.True(t, s.HasFilesSection)
	require.Equal(t, []string{"a.go", "b.go"}, s.Files)
}

func TestParse_NonBacktickBulletSkipped(t *testing.T) {
	data := []byte("---\npackage: p\n---\n\n# 文件\n\n- root.go — no backticks\n- `kept.go` — k\n")
	s := Parse("p/SPEC.md", data)
	require.Equal(t, []string{"kept.go"}, s.Files)
}

func TestParse_DuplicateFileDedup(t *testing.T) {
	data := []byte("---\npackage: p\n---\n\n# 文件\n\n- `a.go` — a\n- `a.go` — dup\n")
	s := Parse("p/SPEC.md", data)
	require.Equal(t, []string{"a.go"}, s.Files)
}

func TestParse_EmptyFilesSection(t *testing.T) {
	data := []byte("---\npackage: p\n---\n\n# 文件\n\n# API\n\nx\n")
	s := Parse("p/SPEC.md", data)
	require.True(t, s.HasFilesSection)
	require.Empty(t, s.Files)
}

func TestParse_PackageFieldCaseSensitive(t *testing.T) {
	data := []byte("---\nPackage: p\n---\n\n# 文件\n\n- `a.go` — a\n")
	s := Parse("p/SPEC.md", data)
	require.False(t, s.HasPackage, "Package (capital) must not match the lowercase package: field")
}

func TestParse_PackageFrontMatterWins(t *testing.T) {
	// The package: regex scans the whole doc (multiline); the front-matter
	// occurrence is first, so it wins over a later body line starting with
	// "package:".
	data := []byte("---\npackage: real\n---\n\n# 概述\n\npackage: stray\n")
	s := Parse("p/SPEC.md", data)
	require.True(t, s.HasPackage)
	require.Equal(t, "real", s.Package)
}
