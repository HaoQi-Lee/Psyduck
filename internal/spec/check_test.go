package spec

import (
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

// keep the time import meaningful until timing tests arrive in 3.3
var _ = time.Unix
