package cli

import "github.com/spf13/cobra"

// Assemble builds the fully-wired root command: it creates the root and
// attaches every top-level command and group through the registration guard.
//
// This is the single, explicit wiring site (ADR-4). The entrypoint calls it
// and executes the result; commands never self-register via package init.
// Adding a command means one MustRegister line here plus the command's own
// constructor — no existing command is edited.
func Assemble() *cobra.Command {
	root := NewRootCommand()
	// Top-level commands and groups are wired here — one MustRegister line
	// each. Adding a command does not touch the others.
	MustRegister(root, newVersionCommand())
	MustRegister(root, newRolesCommand())
	// Credential Storage (006): the auth group + login leaf, delegating the file
	// write to internal/auth through the production input seam.
	MustRegister(root, newAuthCommand(productionSeam{}))
	// Help & Version (003): tune the assembled root's help/version rendering —
	// unify --version with the version command, hide framework built-ins, keep
	// standard alphabetical listing. Applied after wiring so it configures the
	// final command set.
	configureHelpAndVersion(root)
	return root
}
