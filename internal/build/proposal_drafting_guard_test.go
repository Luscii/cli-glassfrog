package build

import "testing"

// TestProposalDraftingDriftGuard is the best-effort drift guard for the Proposal
// Drafting Path (067, plan ADR-5). It anchors the single-sourced composed-leaf
// registry (plugin/agents/proposal-drafting-commands.txt) to three machine
// sources at once:
//
//   - the CLI's actual `tension` and `proposal` subcommand surfaces, so a renamed
//     or dropped command cannot leave the drafting artifacts naming a command the
//     CLI no longer exposes — a silent-drift artifact reads as authoritative while
//     being wrong;
//   - 063's gated proposal-write registry (plugin/hooks/gated-commands.txt), so
//     the gated-membership invariant ADR-3 depends on cannot silently break: the
//     one write (`proposal create`) must stay a member of the gated set, and the
//     situating reads must stay out of it.
//
// This is the INVERSE of 066's disjointness guard. 066 asserts NO composed leaf is
// gated (all its tension writes are ungated by design); 067 asserts its one gated
// write (ProposalDraftingGatedWrite) IS gated and every other composed leaf is
// not. If `proposal create` ever left the gated registry — or were swapped for a
// situating read, which shares the `proposal` group — the create would ship
// unconfirmed, the very failure this surface is prone to, and this guard turns
// that into a build failure.
//
// Every side is derived from source — the composed leaves from
// proposal-drafting-commands.txt, the live surfaces from newTensionCommand /
// newProposalCommand, the gated set from 063's registry — so the guard hard-codes
// none of them, including the identity of the write leaf (derived as "the single
// composed leaf 063 gates", never named literally). A hard-coded expectation would
// become a second source of truth edited on every legitimate change (LEARNINGS: a
// drift guard must not hard-code the value it guards).
//
// COVERAGE (explicitly partial, per plan ADR-5 — stated, not silent):
//   - every composed leaf (tension get; proposal list/get/create) is a `<group>
//     <sub>` pair whose subcommand still resolves on the CLI's matching command;
//   - the one gated write (ProposalDraftingGatedWrite) is named in the composed
//     list and is a member of 063's gated set; every other composed leaf (the
//     situating reads) is absent from it — the gated-membership invariant, both
//     directions, anchored on the write leaf so a read swapped in for the create
//     cannot pass;
//   - the proposal-drafter agent artifact still names every composed leaf, so the
//     prose stays a genuine consumer of the single source, not a divergent copy.
//
// NOT COVERED (no machine source to anchor against; left to review + the BDD
// suite): the commands' flags (deferred to `glassfrog <group> <sub> --help`), the
// draft-record prose, the confirmation-narration wording and the skill's workflow
// steps (the BDD content-inspection scenarios cover their required phrases), and
// the gate script's command-string parsing robustness (063's own suite covers it).
// The guard pins the enumerable surface and its gate-membership, never the
// synthesis; it is not total coverage.
func TestProposalDraftingDriftGuard(t *testing.T) {
	composed, err := ReadProposalDraftingCommands()
	if err != nil {
		t.Fatalf("could not read the composed-leaf registry %s: %v", ProposalDraftingCommandsPath, err)
	}
	// Sanity-check the single source itself: an empty registry would make the guard
	// vacuously pass while the artifacts name commands nobody checked.
	if len(composed) == 0 {
		t.Fatalf("the composed-leaf registry %s lists no leaves — the guard would check nothing", ProposalDraftingCommandsPath)
	}

	liveTension, err := LiveTensionSubcommands()
	if err != nil {
		t.Fatalf("could not extract the CLI's tension subcommand surface: %v", err)
	}
	if len(liveTension) == 0 {
		t.Fatal("extracted no tension subcommands — the tension surface anchor could not be read")
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

	agent, err := ReadProposalDrafterAgent()
	if err != nil {
		t.Fatalf("could not read the proposal-drafter agent %s: %v", ProposalDrafterAgentPath, err)
	}

	if drift := CheckProposalDraftingDrift(composed, liveTension, liveProposal, gated, agent); len(drift) != 0 {
		t.Fatalf("the proposal-drafting path drifted from the shipped CLI or the guardrail boundary:\n  - %s", joinDrift(drift))
	}
}

// TestProposalDraftingKeepsManifestAutoDiscovered confirms the design choice that
// makes 067 safe for 062: the proposal-drafter agent is auto-discovered from
// plugin/agents/, NOT declared as an `agents` key in plugin.json, so the manifest
// stays free of every setup-forcing key (062's ManifestDemandsNoSetup, which
// forbids `agents`). Adding the key would silently break Operator Orientation's
// "no configuration beyond the CLI's credential setup" contract; this fails fast
// if a future change reaches for it. The plugin tree also stays pure data, so an
// agent-load failure can never break a glassfrog command.
func TestProposalDraftingKeepsManifestAutoDiscovered(t *testing.T) {
	_, raw, err := ReadOrientationManifest()
	if err != nil {
		t.Fatalf("could not read the plugin manifest: %v", err)
	}
	if !ManifestDemandsNoSetup(raw) {
		t.Fatal("plugin.json now declares a setup-forcing key (mcpServers/hooks/commands/agents/skills) — 067's drafter must be auto-discovered from plugin/agents/, not a manifest key, to keep 062's no-setup contract")
	}
	clean, err := OrientationPluginHasNoGoCode()
	if err != nil {
		t.Fatalf("could not inspect the plugin tree: %v", err)
	}
	if !clean {
		t.Fatal("plugin tree now carries Go code — an agent-load failure could no longer be isolated from the CLI")
	}
}
