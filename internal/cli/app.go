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
	// Identity Read (011): the `me` command — the first end-to-end API read.
	// productionSeam binds AssembleFromOS + NewClientFromOS (the real transport);
	// me reads the persistent --base-url flag the root declares. me is registered
	// as a leaf FIRST (while childless), so the guard sees a valid runnable leaf;
	// My Roles' `roles` child is attached AFTER, so the guard validates the child
	// and `me` becomes both runnable and a parent without tripping the
	// leaf-xor-group rule (which only fires at a command's own registration).
	meCmd := newMeCommand(productionSeam{})
	MustRegister(root, meCmd)
	// My Roles (012): the `roles` leaf under the runnable `me` command. `me roles`
	// sends GET /me/roles and prints the roles the practitioner fills. Reuses the
	// same productionSeam (assemble + newClient) and the inherited --base-url.
	MustRegister(meCmd, newMyRolesCommand(productionSeam{}))
	// Help & Version (003): tune the assembled root's help/version rendering —
	// unify --version with the version command, hide framework built-ins, keep
	// standard alphabetical listing. Applied after wiring so it configures the
	// final command set.
	configureHelpAndVersion(root)
	return root
}
