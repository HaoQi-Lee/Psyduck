package cli

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/psyduck/psyduck/internal/version"
)

func TestVersionCommand_PrintsVersion(t *testing.T) {
	var stdout, stderr bytes.Buffer
	root := NewRootCmd(&stdout, &stderr)
	root.SetArgs([]string{"version"})

	err := root.Execute()
	require.NoError(t, err)
	require.Equal(t, version.Version+"\n", stdout.String())
	require.Empty(t, stderr.String())
}
