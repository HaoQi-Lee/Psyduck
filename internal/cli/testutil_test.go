package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func setupInitialized(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	chdir(t, dir)
	var out bytes.Buffer
	root := NewRootCmd(&out, &out)
	root.SetArgs([]string{"init"})
	require.NoError(t, root.Execute())
	return dir
}

func writeValidSpec(t *testing.T, root, pkg string) {
	t.Helper()
	mustWriteFile(t, filepath.Join(root, pkg, "SPEC.md"), `---
psy_kind: factual
psy_version: 1
package: `+pkg+`
created: 2026-06-05
---

# Purpose

x.

# Public API

x

# Dependencies

x
`)
}

func mustWriteFile(t *testing.T, path, body string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(body), 0o644))
}
