package main

import (
	"fmt"
	"os"

	"github.com/psyduck/psyduck/internal/cli"
)

func main() {
	root := cli.NewRootCmd(os.Stdout, os.Stderr)
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "psy:", err)
		os.Exit(cli.ExitInternal)
	}
}
