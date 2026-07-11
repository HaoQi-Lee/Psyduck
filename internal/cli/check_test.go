package cli

import (
	"bytes"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/psyduck/psyduck/internal/spec"
)

func TestRenderReport_DriftAndTiming(t *testing.T) {
	var b bytes.Buffer
	renderReport(&b, spec.Report{
		Root: "/r",
		Packages: []spec.PackageReport{
			{
				Package: "pkg", SpecPath: "pkg/SPEC.md", PkgDir: "pkg",
				ListedButGone: []string{"old.go"},
				Undocumented:  []string{"new.go"},
				Timing: []spec.TimingHint{{
					File:     "root.go",
					FileTime: time.Unix(1_750_000_000, 0),
					SpecTime: time.Unix(1_740_000_000, 0),
				}},
			},
			{Package: "clean", SpecPath: "clean/SPEC.md", PkgDir: "clean"},
		},
	})
	out := b.String()
	require.Contains(t, out, "✗ 结构漂移")
	require.Contains(t, out, "- 已过期:   old.go")
	require.Contains(t, out, "+ 未文档化: new.go")
	require.Contains(t, out, "⚠ 时序提示")
	require.Contains(t, out, "root.go 比 SPEC 新")
	require.Contains(t, out, "✓ 无漂移")
	require.Contains(t, out, "发现: 1 处结构漂移, 1 条时序提示")
}
