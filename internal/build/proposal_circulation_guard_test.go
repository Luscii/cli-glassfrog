package build

import "testing"

// TestProposalCirculationDriftGuard is the best-effort drift guard for the
// Proposal Circulation Path (068, plan ADR-5). It anchors the single-sourced
// composed-leaf registry (plugin/agents/proposal-circulation-commands.txt) to
// three machine sources at once:
//
//   - the CLI's actual `proposal` subcommand surface, so a renamed or dropped
//     command cannot leave the circulation artifacts naming a command the CLI no
//     longer exposes — a silent-drift artifact reads as authoritative while
//     being wrong;
//   - 063's gated proposal-write registry (plugin/hooks/gated-commands.txt), so
//     the two-write gated-membership invariant ADR-3 depends on cannot silently
//     break: both transitions (`proposal propose`, `proposal withdraw`) must
//     each stay a member of the gated set, and the reads (`proposal get`,
//     `proposal list`) must stay out of it.
//
// This extends 067's gate-posture pattern to a two-in-two-out path. 066 asserts
// NO composed leaf is gated (all-out); 067 asserts its one gated write IS gated
// and the rest are not (one-in); 068 asserts BOTH transitions are gated and BOTH
// reads are not (two-in-two-out). If either transition ever left the gated
// registry — or were swapped for a read, which shares the `proposal` group and
// preserves any count — that transition would ship unconfirmed, the very failure
// this surface is prone to, and this guard turns that into a build failure.
//
// The three registries are derived from source — the composed leaves from
// proposal-circulation-commands.txt, the live surface from newProposalCommand,
// the gated set from 063's registry — so the guard hard-codes none of the SETS.
// The one thing it must name is the identity of the two gated writes
// (ProposalCirculationGatedWrites): a read and a transition share the `proposal`
// group, so the writes cannot be derived as "the composed leaves 063 gates" —
// and a count-only membership check ("exactly two composed leaves are gated")
// would be satisfied by a read/write swap across the gate. Each anchor is pinned
// per leaf, like 067's ProposalDraftingGatedWrite, and cross-checked against the
// composed list. Hard-coding a whole enumerable SET would be the anti-pattern;
// naming the per-leaf contract-facts the derivation genuinely cannot recover is
// not (LEARNINGS: a drift guard must not hard-code the value it guards — meaning
// the guarded sets, not the anchors those sets cannot express).
//
// COVERAGE (explicitly partial, per plan ADR-5 — stated, not silent):
//   - every composed leaf (proposal get/list/propose/withdraw) is a
//     `proposal <sub>` pair whose subcommand still resolves on the CLI's
//     proposal command;
//   - each gated write (ProposalCirculationGatedWrites) is named in the composed
//     list and is a member of 063's gated set; every other composed leaf (the
//     reads) is absent from it — the gated-membership invariant, both
//     directions, anchored per write leaf so a read swapped in for a transition
//     cannot pass;
//   - the proposal-circulator agent artifact still names every composed leaf, so
//     the prose stays a genuine consumer of the single source, not a divergent
//     copy.
//
// NOT COVERED (no machine source to anchor against; left to review + the BDD
// suite): the commands' flags (deferred to `glassfrog proposal <sub> --help`),
// the circulation-record prose, the confirmation-narration wording and the
// skill's workflow steps (the BDD content-inspection scenarios cover their
// required phrases), the reads-inform-never-gate discipline (prompt-level; a
// held-out validation scenario inspects for a client-side gate), and the gate
// script's command-string parsing robustness (063's own suite covers it). The
// guard pins the enumerable surface and its gate-membership, never the
// synthesis; it is not total coverage.
func TestProposalCirculationDriftGuard(t *testing.T) {
	composed, err := ReadProposalCirculationCommands()
	if err != nil {
		t.Fatalf("could not read the composed-leaf registry %s: %v", ProposalCirculationCommandsPath, err)
	}
	// Sanity-check the single source itself: an empty registry would make the guard
	// vacuously pass while the artifacts name commands nobody checked.
	if len(composed) == 0 {
		t.Fatalf("the composed-leaf registry %s lists no leaves — the guard would check nothing", ProposalCirculationCommandsPath)
	}

	liveProposal, err := LiveProposalSubcommands()
	if err != nil {
		t.Fatalf("could not extract the CLI's proposal subcommand surface: %v", err)
	}
	if len(liveProposal) == 0 {
		t.Fatal("extracted no proposal subcommands — the proposal surface anchor could not be read")
	}

	gated, err := ReadGatedRegistry()
	if err != nil {
		t.Fatalf("could not read 063's gated-command registry %s: %v", GatedRegistryPath, err)
	}
	// An empty gated set would make the membership half of the guard vacuous; 063
	// guarantees the registry is non-empty (its own guard pins the four proposal
	// writes), so an empty read here means the anchor broke.
	if len(gated) == 0 {
		t.Fatalf("063's gated registry %s lists no leaves — the gated-membership check would be vacuous", GatedRegistryPath)
	}

	agent, err := ReadProposalCirculatorAgent()
	if err != nil {
		t.Fatalf("could not read the proposal-circulator agent %s: %v", ProposalCirculatorAgentPath, err)
	}

	if drift := CheckProposalCirculationDrift(composed, liveProposal, gated, agent); len(drift) != 0 {
		t.Fatalf("the proposal-circulation path drifted from the shipped CLI or the guardrail boundary:\n  - %s", joinDrift(drift))
	}
}

// TestProposalCirculationKeepsManifestAutoDiscovered confirms the design choice
// that makes 068 safe for 062: the proposal-circulator agent is auto-discovered
// from plugin/agents/, NOT declared as an `agents` key in plugin.json, so the
// manifest stays free of every setup-forcing key (062's ManifestDemandsNoSetup,
// which forbids `agents`). Adding the key would silently break Operator
// Orientation's "no configuration beyond the CLI's credential setup" contract;
// this fails fast if a future change reaches for it. The plugin tree also stays
// pure data, so an agent-load failure can never break a glassfrog command.
func TestProposalCirculationKeepsManifestAutoDiscovered(t *testing.T) {
	_, raw, err := ReadOrientationManifest()
	if err != nil {
		t.Fatalf("could not read the plugin manifest: %v", err)
	}
	if !ManifestDemandsNoSetup(raw) {
		t.Fatal("plugin.json now declares a setup-forcing key (mcpServers/hooks/commands/agents/skills) — 068's circulator must be auto-discovered from plugin/agents/, not a manifest key, to keep 062's no-setup contract")
	}
	clean, err := OrientationPluginHasNoGoCode()
	if err != nil {
		t.Fatalf("could not inspect the plugin tree: %v", err)
	}
	if !clean {
		t.Fatal("plugin tree now carries Go code — an agent-load failure could no longer be isolated from the CLI")
	}
}
