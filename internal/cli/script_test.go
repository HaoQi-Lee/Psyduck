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
	err := root.Execute()
	if err == nil {
		return ExitOK
	}
	fmt.Fprintln(os.Stderr, "psy:", err)
	return ExitInternal
}

func TestScripts(t *testing.T) {
	testscript.Run(t, testscript.Params{
		Dir: "../../testdata/script",
	})
}
