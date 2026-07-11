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
