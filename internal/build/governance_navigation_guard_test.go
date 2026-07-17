package build

import "testing"

// TestGovernanceNavigationDriftGuard is the best-effort drift guard for the
// Governance Navigation Path (064, plan ADR-4). It anchors the single-sourced
// composed-read registry (plugin/agents/composed-reads.txt) to the CLI's actual
// top-level command surface, so a renamed or dropped read cannot leave the
// navigation artifacts naming a command the CLI no longer exposes — a
// silent-drift artifact reads as authoritative while being wrong.
//
// Both sides are derived from source — the leaves from composed-reads.txt, the
// live surface from app.go's Assemble — so the guard hard-codes neither. A
// hard-coded expectation would become a second source of truth edited on every
// legitimate change (LEARNINGS: a drift guard must not hard-code the value it
// guards).
//
// COVERAGE (explicitly partial, per plan ADR-4 — stated, not silent):
//   - every composed read leaf (search, roles, tree, fillers, subrole-actors,
//     domains, policies) still resolves to a top-level command in the shipped CLI;
//   - the navigator agent artifact still names every composed leaf, so the prose
//     stays a genuine consumer of the single source, not a divergent copy.
//
// NOT COVERED (no machine source to anchor against; left to review + the BDD
// suite): the commands' flags (deferred to `glassfrog <command> --help`), the
// read-vs-write classification of a leaf (a write leaf added to the registry would
// still "exist" — the read-only property is guarded separately by the BDD "only
// reads, never writes" scenario and the agent's Write/Edit-withheld tool grant),
// the synthesized-picture prose, and the traversal's relevance judgment. The guard
// pins the enumerable surface, never the synthesis; it is not total coverage.
func TestGovernanceNavigationDriftGuard(t *testing.T) {
	composed, err := ReadComposedReads()
	if err != nil {
		t.Fatalf("could not read the composed-read registry %s: %v", ComposedReadsPath, err)
	}
	// Sanity-check the single source itself: an empty registry would make the guard
	// vacuously pass while the artifacts name reads nobody checked.
	if len(composed) == 0 {
		t.Fatalf("the composed-read registry %s lists no leaves — the guard would check nothing", ComposedReadsPath)
	}

	live, err := LiveTopLevelCommands()
	if err != nil {
		t.Fatalf("could not extract the CLI's top-level command surface: %v", err)
	}
	// Sanity-check the extraction, so a regression in LiveTopLevelCommands fails loudly
	// rather than silently reporting an empty surface as "no drift".
	if len(live) == 0 {
		t.Fatal("extracted no top-level commands — the surface anchor could not be read")
	}

	agent, err := ReadNavigatorAgent()
	if err != nil {
		t.Fatalf("could not read the navigator agent %s: %v", NavigatorAgentPath, err)
	}

	if drift := CheckNavigationDrift(composed, live, agent); len(drift) != 0 {
		t.Fatalf("the navigation path drifted from the shipped CLI:\n  - %s", joinDrift(drift))
	}
}

// TestNavigationKeepsManifestAutoDiscovered confirms the design choice that makes
// 064 safe for 062: the navigator agent is auto-discovered from plugin/agents/,
// NOT declared as an `agents` key in plugin.json, so the manifest stays free of
// every setup-forcing key (062's ManifestDemandsNoSetup, which forbids `agents`).
// Adding the key would silently break Operator Orientation's "no configuration
// beyond the CLI's credential setup" contract; this fails fast if a future change
// reaches for it. The plugin tree also stays pure data, so an agent-load failure
// can never break a glassfrog command.
func TestNavigationKeepsManifestAutoDiscovered(t *testing.T) {
	_, raw, err := ReadOrientationManifest()
	if err != nil {
		t.Fatalf("could not read the plugin manifest: %v", err)
	}
	if !ManifestDemandsNoSetup(raw) {
		t.Fatal("plugin.json now declares a setup-forcing key (mcpServers/hooks/commands/agents/skills) — 064's navigator must be auto-discovered from plugin/agents/, not a manifest key, to keep 062's no-setup contract")
	}
	clean, err := OrientationPluginHasNoGoCode()
	if err != nil {
		t.Fatalf("could not inspect the plugin tree: %v", err)
	}
	if !clean {
		t.Fatal("plugin tree now carries Go code — an agent-load failure could no longer be isolated from the CLI")
	}
}
