package cli

import (
	"io"

	"github.com/spf13/cobra"
)

// NewRootCmd builds the `psy` root command. stdout and stderr are injected
// for testability; production callers pass os.Stdout / os.Stderr.
func NewRootCmd(stdout, stderr io.Writer) *cobra.Command {
	root := &cobra.Command{
		Use:           "psy",
		Short:         "PsyDuck — spec lifecycle for Claude Code workflows",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.SetOut(stdout)
	root.SetErr(stderr)
	root.CompletionOptions.DisableDefaultCmd = true
	root.AddCommand(newVersionCmd())
	root.AddCommand(newInitCmd())
	root.AddCommand(newCheckCmd())
	return root
}
