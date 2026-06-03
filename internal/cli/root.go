package cli

import "github.com/spf13/cobra"

// NewRootCommand returns the top of the known command set — the `glassfrog`
// root command. It carries no action of its own; invoked with no subcommand it
// resolves to itself and prints help (the bare-group behavior from
// interface-cli.md, which the root shares).
//
// The returned command is the parent that subcommands are attached to via the
// registration guard during explicit startup wiring (see Assemble).
func NewRootCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "glassfrog",
		Short: "Command-line surface over the Glassfrog v5 governance API",
		Long: "glassfrog is a faithful command-line surface over the Glassfrog v5 " +
			"Holacracy governance API, built for AI agents and practitioners.",
		// No Run: a bare invocation resolves to the root and prints help.
		// SilenceUsage keeps usage noise out of error paths; error-to-exit-code
		// mapping is the Exit-Code Convention capability's concern.
		SilenceUsage: true,
	}
}
