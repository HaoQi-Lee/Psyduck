package main

import (
	"fmt"
	"os"

	"github.com/psyduck/psyduck/internal/cli"
)

func main() {
	root := cli.NewRootCmd(os.Stdout, os.Stderr)
	code, msg := cli.ExitCodeFromErr(root.Execute())
	if msg != "" {
		fmt.Fprintln(os.Stderr, "psy:", msg)
	}
	os.Exit(code)
}
