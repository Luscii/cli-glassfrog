package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// version is the CLI's version string. It can be overridden at build time with
// -ldflags "-X github.com/Luscii/cli-glassfrog/internal/cli.version=...".
var version = "0.0.0-dev"

// newVersionCommand returns the `version` leaf. Its output format is
// intentionally minimal here — richer --version presentation is the Help &
// Version capability's concern (a later spec).
func newVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the glassfrog version",
		// version takes no positional arguments; reject any so unexpected input
		// is a usage error rather than silently ignored (dispatch's Invalid-input
		// accord, 002).
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			fmt.Fprintln(cmd.OutOrStdout(), version)
			return nil
		},
	}
}
