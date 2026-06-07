package cli

import (
	"github.com/Luscii/cli-glassfrog/internal/apiclient"
	"github.com/spf13/cobra"
)

// NewRootCommand returns the top of the known command set — the `glassfrog`
// root command. It carries no action of its own; invoked with no subcommand it
// resolves to itself and prints help (the bare-group behavior from
// interface-cli.md, which the root shares).
//
// The returned command is the parent that subcommands are attached to via the
// registration guard during explicit startup wiring (see Assemble).
//
// It declares the global connection flags every API command inherits. Today
// that is --base-url, the highest-precedence rung of base-URL resolution (008:
// flag → GLASSFROG_BASE_URL → .glassfrogrc base_url → built-in default). It is a
// PERSISTENT flag on the root so it is registered once and inherited by every
// current and future API command (Identity Read 011, ADR-2); API commands read
// its value via cobra flag inheritance and pass it to AssembleFromOS, rather
// than each re-registering it. The flag NAME comes from apiclient.FlagBaseURL so
// the precedence-chain rung and the registered flag can't drift; the usage
// string is an implementation detail kept consistent with the spec. The flag
// appears (inert) on non-API commands like version/auth — an accepted wart
// (ADR-2).
func NewRootCommand() *cobra.Command {
	root := &cobra.Command{
		Use:   "glassfrog",
		Short: "Command-line surface over the Glassfrog v5 governance API",
		Long: "glassfrog is a faithful command-line surface over the Glassfrog v5 " +
			"Holacracy governance API, built for AI agents and practitioners.",
		// No Run: a bare invocation resolves to the root and prints help.
		// SilenceUsage keeps usage noise out of error paths; error-to-exit-code
		// mapping is the Exit-Code Convention capability's concern.
		SilenceUsage: true,
	}
	root.PersistentFlags().String(
		apiclient.FlagBaseURL, "",
		"Glassfrog API base URL (overrides GLASSFROG_BASE_URL, the .glassfrogrc base_url, and the built-in default)",
	)
	return root
}
