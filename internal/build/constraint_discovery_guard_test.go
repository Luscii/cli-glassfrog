package build

import "testing"

// TestConstraintDiscoveryDriftGuard is the best-effort drift guard for the
// Constraint Discovery Path (065, plan ADR-2). It anchors the single-sourced
// composed-read registry (plugin/agents/constraint-discovery-composed-reads.txt)
// to the CLI's actual command surface, so a renamed or dropped read cannot
// leave the constraint-discovery artifacts naming a command the CLI no longer
// exposes — a silent-drift artifact reads as authoritative while being wrong.
//
// Both sides are derived from source — the leaves from the registry, the live
// surfaces from app.go's Assemble — so the guard hard-codes neither. A
// hard-coded expectation would become a second source of truth edited on every
// legitimate change (LEARNINGS: a drift guard must not hard-code the value it
// guards).
//
// COVERAGE (explicitly partial, per plan ADR-2 — stated, not silent):
//   - every single-token composed leaf (search, roles, tree, domains, policies,
//     policy) still resolves to a top-level command in the shipped CLI;
//   - the "me roles" leaf still resolves as the `roles` subcommand of the
//     root-registered `me` command (the one command-path leaf the registry
//     carries; `me` is variable-wired, so LiveTopLevelCommands cannot see it);
//   - the constraint-navigator agent artifact still names every composed leaf,
//     so the prose stays a genuine consumer of the single source, not a
//     divergent copy.
//
// NOT COVERED (no machine source to anchor against; left to review + the BDD
// suite): the commands' flags (deferred to `glassfrog <command> --help`), the
// synthesized-picture prose and the `characterization` wording (guarded by the
// BDD content scenarios), the read-vs-write classification of a leaf (a write
// leaf added to the registry would still "exist" — the read-only property is
// guarded separately by the BDD "only reads, never writes" scenario and the
// agent's Write/Edit-withheld tool grant), parser robustness, and the
// traversal's relevance judgment. The guard pins the enumerable surface, never
// the synthesis; it is not total coverage.
func TestConstraintDiscoveryDriftGuard(t *testing.T) {
	composed, err := ReadConstraintComposedReads()
	if err != nil {
		t.Fatalf("could not read the composed-read registry %s: %v", ConstraintComposedReadsPath, err)
	}
	// Sanity-check the single source itself: an empty registry would make the
	// guard vacuously pass while the artifacts name reads nobody checked.
	if len(composed) == 0 {
		t.Fatalf("the composed-read registry %s lists no leaves — the guard would check nothing", ConstraintComposedReadsPath)
	}

	liveTop, err := LiveTopLevelCommands()
	if err != nil {
		t.Fatalf("could not extract the CLI's top-level command surface: %v", err)
	}
	if len(liveTop) == 0 {
		t.Fatal("extracted no top-level commands — the surface anchor could not be read")
	}
	liveMe, err := LiveMeSubcommands()
	if err != nil {
		t.Fatalf("could not extract the `me` subcommand surface: %v", err)
	}
	if len(liveMe) == 0 {
		t.Fatal("extracted no `me` subcommands — the surface anchor could not be read")
	}

	agent, err := ReadConstraintNavigatorAgent()
	if err != nil {
		t.Fatalf("could not read the navigator agent %s: %v", ConstraintNavigatorAgentPath, err)
	}

	if drift := CheckConstraintDrift(composed, liveTop, liveMe, agent); len(drift) != 0 {
		t.Fatalf("the constraint discovery path drifted from the shipped CLI:\n  - %s", joinDrift(drift))
	}
}
