package cli

import "github.com/spf13/cobra"

// configureHelpAndVersion tunes the assembled root's help and version
// rendering to match the Help & Version accord (003). It is applied once, by
// Assemble, after the command tree is wired. It registers no guard-registered
// (domain) command — it configures the framework's standard rendering over the
// guard-registered set. It does replace cobra's built-in `help` command with a
// hidden, non-resolvable placeholder (a framework-level command, not a
// domain one); see the SetHelpCommand call below.
func configureHelpAndVersion(root *cobra.Command) {
	// ADR-3 — version unify: the --version flag and the `version` command read
	// one source of truth and emit the same bare value. That source is the
	// RESOLVED version (resolvedVersion: injected → Go build-info → placeholder,
	// spec 023), not the raw injected var — so the flag reports the same value
	// the version command prints, across release, source-install, and local
	// builds. cobra enables the --version flag when Version is non-empty;
	// resolvedVersion is always non-empty, so the flag is always present. The
	// template overrides cobra's default "Name version X" form so the flag
	// output is byte-identical to the version command's Fprintln.
	root.Version = resolvedVersion()
	root.SetVersionTemplate("{{.Version}}\n")

	// ADR-2 — hide built-ins: replace cobra's auto `help` command with a hidden
	// placeholder that carries no name (empty Use), and disable the
	// `completion` command. cobra always attaches *some* help command when the
	// root has subcommands, and Hidden:true alone only hides it from listings —
	// it does not stop `glassfrog help` from resolving. A nameless placeholder
	// occupies that slot without introducing any typeable, resolvable token:
	// `glassfrog help` is an unknown command, and there is no invented path to
	// invoke (even a degenerate empty argument is rejected as unknown). The
	// --help flag is untouched and stays available everywhere.
	root.SetHelpCommand(&cobra.Command{Hidden: true})
	root.CompletionOptions.DisableDefaultCmd = true

	// ADR-1 — standard rendering + sorting: no custom help/usage template;
	// alphabetical listing is cobra's default (cobra.EnableCommandSorting ==
	// true). Nothing to set here — the default is kept and pinned by a
	// regression test.
}
