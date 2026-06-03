// Package cli hosts the Glassfrog CLI's command tree and the registration
// guard that builds it.
//
// The command set is realized on cobra (see ADR-2): cobra's command tree is
// the registry. Commands are attached through the fail-loud Register /
// MustRegister guard (see ADR-3) rather than cobra's AddCommand directly, and
// the tree is assembled explicitly at startup (see ADR-4) — never via package
// init side effects.
//
// This file establishes the package as the home for the root command
// (root.go), the registration guard (registry.go), and the explicit wiring of
// individual commands. The entrypoint lives in the repository-root main
// package, which executes the assembled root command.
package cli
