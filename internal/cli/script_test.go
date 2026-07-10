package cli

import (
	"fmt"
	"os"
	"testing"

	"github.com/rogpeppe/go-internal/testscript"
)

// TestMain wires the `psy` binary into testscript so script fixtures can
// invoke it directly (no shelling out to a separately-built binary).
func TestMain(m *testing.M) {
	os.Exit(testscript.RunMain(m, map[string]func() int{
		"psy": runPsyForTestscript,
	}))
}

// runPsyForTestscript mirrors cmd/psy/main.go but returns an exit code
// instead of calling os.Exit, so testscript can capture it.
func runPsyForTestscript() int {
	root := NewRootCmd(os.Stdout, os.Stderr)
	code, msg := ExitCodeFromErr(root.Execute())
	if msg != "" {
		fmt.Fprintln(os.Stderr, "psy:", msg)
	}
	return code
}

func TestScripts(t *testing.T) {
	testscript.Run(t, testscript.Params{
		Dir: "../../testdata/script",
	})
}
