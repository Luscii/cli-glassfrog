package build

import "testing"

// TestTensionProcessingDriftGuard is the best-effort drift guard for the Tension
// Processing Path (066, plan ADR-5). It anchors the single-sourced composed-leaf
// registry (plugin/agents/tension-processing-commands.txt) to two machine
// sources at once:
//
//   - the CLI's actual `tension` subcommand surface, so a renamed or dropped
//     tension command cannot leave the processing artifacts naming a command the
//     CLI no longer exposes — a silent-drift artifact reads as authoritative
//     while being wrong;
//   - 063's gated proposal-write registry (plugin/hooks/gated-commands.txt), so
//     the ungated-writes invariant ADR-3 depends on cannot silently break in
//     either direction — a tension leaf pulled into the gated set would make the
//     processor's writes start prompting, and a proposal leaf leaking into the
//     composed set would be wrongly executed by the subagent.
//
// Every side is derived from source — the composed leaves from
// tension-processing-commands.txt, the live surface from newTensionCommand, the
// gated set from 063's registry — so the guard hard-codes none of them. A
// hard-coded expectation would become a second source of truth edited on every
// legitimate change (LEARNINGS: a drift guard must not hard-code the value it
// guards).
//
// COVERAGE (explicitly partial, per plan ADR-5 — stated, not silent):
//   - every composed leaf (tension list/get/subroles/create/update/discard) is a
//     `tension <sub>` pair whose subcommand still resolves on the CLI's tension
//     command;
//   - the composed set is disjoint from 063's gated proposal-write set, in both
//     directions (no composed leaf gated; no `tension` leaf in the gated
//     registry);
//   - the tension-processor agent artifact still names every composed leaf, so
//     the prose stays a genuine consumer of the single source, not a divergent
//     copy.
//
// NOT COVERED (no machine source to anchor against; left to review + the BDD
// suite): the commands' flags (deferred to `glassfrog tension <sub> --help`),
// the tension-record prose and the skill's workflow wording (the BDD
// content-inspection scenarios cover their required phrases), the gate script's
// command-string parsing robustness (063's own suite covers it), and the
// read-vs-write classification of a tension leaf (a new write leaf added to the
// registry would still "exist" — the fence is guarded by the disjointness check
// plus the agent's Write/Edit-withheld grant and prompt scope, asserted by the
// BDD suite). The guard pins the enumerable surface and its gate-membership,
// never the synthesis; it is not total coverage.
func TestTensionProcessingDriftGuard(t *testing.T) {
	composed, err := ReadTensionCommands()
	if err != nil {
		t.Fatalf("could not read the composed-leaf registry %s: %v", TensionCommandsPath, err)
	}
	// Sanity-check the single source itself: an empty registry would make the guard
	// vacuously pass while the artifacts name commands nobody checked.
	if len(composed) == 0 {
		t.Fatalf("the composed-leaf registry %s lists no leaves — the guard would check nothing", TensionCommandsPath)
	}

	live, err := LiveTensionSubcommands()
	if err != nil {
		t.Fatalf("could not extract the CLI's tension subcommand surface: %v", err)
	}
	// Sanity-check the extraction, so a regression in LiveTensionSubcommands fails
	// loudly rather than silently reporting an empty surface as "no drift".
	if len(live) == 0 {
		t.Fatal("extracted no tension subcommands — the surface anchor could not be read")
	}

	gated, err := ReadGatedRegistry()
	if err != nil {
		t.Fatalf("could not read 063's gated-command registry %s: %v", GatedRegistryPath, err)
	}
	// An empty gated set would make the disjointness half of the guard vacuous;
	// 063 guarantees the registry is non-empty (its own guard pins the four
	// proposal writes), so an empty read here means the anchor broke.
	if len(gated) == 0 {
		t.Fatalf("063's gated registry %s lists no leaves — the ungated-invariant check would be vacuous", GatedRegistryPath)
	}

	agent, err := ReadTensionProcessorAgent()
	if err != nil {
		t.Fatalf("could not read the tension-processor agent %s: %v", TensionProcessorAgentPath, err)
	}

	if drift := CheckTensionProcessingDrift(composed, live, gated, agent); len(drift) != 0 {
		t.Fatalf("the tension-processing path drifted from the shipped CLI or the guardrail boundary:\n  - %s", joinDrift(drift))
	}
}

// TestTensionProcessingKeepsManifestAutoDiscovered confirms the design choice
// that makes 066 safe for 062: the tension-processor agent is auto-discovered
// from plugin/agents/, NOT declared as an `agents` key in plugin.json, so the
// manifest stays free of every setup-forcing key (062's ManifestDemandsNoSetup,
// which forbids `agents`). Adding the key would silently break Operator
// Orientation's "no configuration beyond the CLI's credential setup" contract;
// this fails fast if a future change reaches for it. The plugin tree also stays
// pure data, so an agent-load failure can never break a glassfrog command.
func TestTensionProcessingKeepsManifestAutoDiscovered(t *testing.T) {
	_, raw, err := ReadOrientationManifest()
	if err != nil {
		t.Fatalf("could not read the plugin manifest: %v", err)
	}
	if !ManifestDemandsNoSetup(raw) {
		t.Fatal("plugin.json now declares a setup-forcing key (mcpServers/hooks/commands/agents/skills) — 066's processor must be auto-discovered from plugin/agents/, not a manifest key, to keep 062's no-setup contract")
	}
	clean, err := OrientationPluginHasNoGoCode()
	if err != nil {
		t.Fatalf("could not inspect the plugin tree: %v", err)
	}
	if !clean {
		t.Fatal("plugin tree now carries Go code — an agent-load failure could no longer be isolated from the CLI")
	}
}
