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
	// Role Reads (025): the org-wide `roles` command (list + single read). It
	// REPLACES the earlier stub `roles` group. productionSeam binds the real
	// transport (AssembleFromOS + NewClientFromOS) and clock; `roles` reads the
	// inherited persistent --base-url/--output flags.
	MustRegister(root, newRolesCommand(productionSeam{}))
	// Organization Tree (026): the `tree` command — the whole-org tree (no id) or
	// the subtree rooted at a role (one id), one unpaginated nested document. A
	// sibling of `roles`, not a child (ADR-1). productionSeam binds the real
	// transport + clock; `tree` reads the inherited persistent --base-url/--output.
	MustRegister(root, newTreeCommand(productionSeam{}))
	// Organization Tree (026): the `subroles <id>` command — a role's immediate
	// children, paginated and walked to completion by default (reusing 025's walk +
	// --first-page opt-out). Also a sibling of `roles` (ADR-1).
	MustRegister(root, newSubrolesCommand(productionSeam{}))
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
	MustRegister(meCmd, newMeRolesCommand(productionSeam{}))
	// My Actions (013): the `actions` leaf under `me`, a sibling of `roles`.
	// `me actions [--status <s>]` sends GET /me/actions and prints the actions the
	// practitioner's roles own, optionally filtered by status. (The 013 spec prose
	// says "my actions"; the implemented convention is `me actions` under `me`,
	// mirroring `me roles` — see .score/memory/LEARNINGS.md.) Reuses the same
	// productionSeam and the inherited --base-url.
	MustRegister(meCmd, newMeActionsCommand(productionSeam{}))
	// My Projects (014): the `projects` leaf under `me`, a sibling of `roles` and
	// `actions`. `me projects [--status <s>]` sends GET /me/projects and prints the
	// projects the practitioner's roles own, optionally filtered by status. (The
	// 014 spec prose says "my projects"; the implemented convention is `me projects`
	// under `me`, mirroring `me roles`/`me actions` — see .score/memory/LEARNINGS.md.)
	// Reuses the same productionSeam and the inherited --base-url; it adds no
	// --include flag (/me/projects offers no include — ADR-2).
	MustRegister(meCmd, newMeProjectsCommand(productionSeam{}))
	// Help & Version (003): tune the assembled root's help/version rendering —
	// unify --version with the version command, hide framework built-ins, keep
	// standard alphabetical listing. Applied after wiring so it configures the
	// final command set.
	configureHelpAndVersion(root)
	return root
}
