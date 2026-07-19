package build

import "testing"

// TestProposalImpactReviewDriftGuard is the best-effort drift guard for the
// Proposal Impact Review Path (069, plan ADR-5). It anchors the single-sourced
// composed-leaf registry (plugin/agents/proposal-impact-review-commands.txt) to
// the CLI's actual command surface and to 063's gated proposal-write registry
// at once:
//
//   - every composed leaf must still resolve to a real command in the shipped
//     CLI — against the anchor its shape names: `proposal <sub>` pairs on the
//     proposal subcommand surface, `me <sub>` pairs on the `me` subcommand
//     surface, the bare `me` on that command's root wiring (it is
//     variable-wired, so LiveTopLevelCommands deliberately does not see it;
//     LiveMeSubcommands proves the registration), and the top-level reads
//     (`roles`, `domains`, `policies` — top-level commands, not `roles`
//     subcommands) on the top-level surface. A silent-drift artifact reads as
//     authoritative while being wrong;
//   - the one-in-nine-out gate-membership invariant ADR-3 depends on cannot
//     silently break: `proposal respond` (ProposalImpactReviewGatedWrite) must
//     stay a member of 063's gated set (plugin/hooks/gated-commands.txt) —
//     the confirmed-write-flow promise pinned structurally — and the nine
//     review reads must stay out of it (the review must not start prompting).
//
// This extends the 067/068 gate-posture pattern to a one-in-nine-out path:
// 066 asserts all-out, 067 one-in, 068 two-in-two-out, 069 one-in-NINE-out —
// closing the family (069 is the last operator path). If the respond ever left
// the gated registry — or were swapped for a read, which shares the `proposal`
// group and preserves any count — the response would ship unconfirmed, the
// very failure this surface is prone to, and this guard turns that into a
// build failure.
//
// The registries are derived from source — the composed leaves from
// proposal-impact-review-commands.txt, the live surfaces from the CLI sources,
// the gated set from 063's registry — so the guard hard-codes none of the
// SETS. The one thing it must name is the identity of the gated write
// (ProposalImpactReviewGatedWrite): a read and the respond share the
// `proposal` group, so the write cannot be derived as "the composed leaf 063
// gates" — and a count-only membership check ("exactly one composed leaf is
// gated") would be satisfied by a read/write swap across the gate. The anchor
// is pinned per leaf, like 067's ProposalDraftingGatedWrite and 068's
// ProposalCirculationGatedWrites, and cross-checked against the composed list.
// Hard-coding a whole enumerable SET would be the anti-pattern; naming the
// per-leaf contract-fact the derivation genuinely cannot recover is not
// (LEARNINGS: a drift guard must not hard-code the value it guards — meaning
// the guarded sets, not the anchors those sets cannot express).
//
// COVERAGE (explicitly partial, per plan ADR-5 — stated, not silent):
//   - every composed leaf resolves against its shape's anchor (bare `me`,
//     `me <sub>`, `proposal <sub>`, top-level command word); any other shape
//     is reported as unanchorable rather than silently skipped;
//   - the gated write (ProposalImpactReviewGatedWrite) is named in the
//     composed list and is a member of 063's gated set; every other composed
//     leaf (the nine reads) is absent from it — the gated-membership
//     invariant, both directions, anchored per write leaf so a read swapped in
//     for the respond cannot pass;
//   - each consumer still names its side of the single source: the
//     proposal-impact-reviewer agent every read leaf, the skill the respond
//     leaf — the prose stays a genuine consumer, not a divergent copy.
//
// NOT COVERED (no machine source to anchor against; left to review + the BDD
// suite): the commands' flags (deferred to `glassfrog <cmd> --help`), the
// impact-picture prose and the skill's workflow steps (the BDD
// content-inspection scenarios cover their required phrases), the
// footprint-honesty and reads-inform-never-decide disciplines (prompt-level;
// held-out validation scenarios inspect them), the split write locus itself
// (prompt-level; the BDD guardrail scenario exercises 063's real gate), and
// the gate script's command-string parsing robustness (063's own suite covers
// it). The guard pins the enumerable surface and its gate-membership, never
// the synthesis; it is not total coverage.
func TestProposalImpactReviewDriftGuard(t *testing.T) {
	composed, err := ReadProposalImpactReviewCommands()
	if err != nil {
		t.Fatalf("could not read the composed-leaf registry %s: %v", ProposalImpactReviewCommandsPath, err)
	}
	// Sanity-check the single source itself: an empty registry would make the
	// guard vacuously pass while the artifacts name commands nobody checked.
	if len(composed) == 0 {
		t.Fatalf("the composed-leaf registry %s lists no leaves — the guard would check nothing", ProposalImpactReviewCommandsPath)
	}

	liveTop, err := LiveTopLevelCommands()
	if err != nil {
		t.Fatalf("could not extract the CLI's top-level command surface: %v", err)
	}
	if len(liveTop) == 0 {
		t.Fatal("extracted no top-level commands — the top-level surface anchor could not be read")
	}

	liveMe, err := LiveMeSubcommands()
	if err != nil {
		t.Fatalf("could not extract the CLI's `me` subcommand surface: %v", err)
	}
	if len(liveMe) == 0 {
		t.Fatal("extracted no `me` subcommands — the me surface anchor could not be read")
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
	// An empty gated set would make the membership half of the guard vacuous;
	// 063 guarantees the registry is non-empty (its own guard pins the four
	// proposal writes), so an empty read here means the anchor broke.
	if len(gated) == 0 {
		t.Fatalf("063's gated registry %s lists no leaves — the gated-membership check would be vacuous", GatedRegistryPath)
	}

	skill, err := ReadProposalImpactReviewSkill()
	if err != nil {
		t.Fatalf("could not read the proposal-impact-review skill %s: %v", ProposalImpactReviewSkillPath, err)
	}
	agent, err := ReadProposalImpactReviewerAgent()
	if err != nil {
		t.Fatalf("could not read the proposal-impact-reviewer agent %s: %v", ProposalImpactReviewerAgentPath, err)
	}

	if drift := CheckProposalImpactReviewDrift(composed, liveTop, liveMe, liveProposal, gated, skill, agent); len(drift) != 0 {
		t.Fatalf("the proposal-impact-review path drifted from the shipped CLI or the guardrail boundary:\n  - %s", joinDrift(drift))
	}
}

// TestProposalImpactReviewKeepsManifestAutoDiscovered confirms the design
// choice that makes 069 safe for 062: the proposal-impact-reviewer agent is
// auto-discovered from plugin/agents/, NOT declared as an `agents` key in
// plugin.json, so the manifest stays free of every setup-forcing key (062's
// ManifestDemandsNoSetup, which forbids `agents`). Adding the key would
// silently break Operator Orientation's "no configuration beyond the CLI's
// credential setup" contract; this fails fast if a future change reaches for
// it. The plugin tree also stays pure data, so an agent-load failure can never
// break a glassfrog command.
func TestProposalImpactReviewKeepsManifestAutoDiscovered(t *testing.T) {
	_, raw, err := ReadOrientationManifest()
	if err != nil {
		t.Fatalf("could not read the plugin manifest: %v", err)
	}
	if !ManifestDemandsNoSetup(raw) {
		t.Fatal("plugin.json now declares a setup-forcing key (mcpServers/hooks/commands/agents/skills) — 069's reviewer must be auto-discovered from plugin/agents/, not a manifest key, to keep 062's no-setup contract")
	}
	clean, err := OrientationPluginHasNoGoCode()
	if err != nil {
		t.Fatalf("could not inspect the plugin tree: %v", err)
	}
	if !clean {
		t.Fatal("plugin tree now carries Go code — an agent-load failure could no longer be isolated from the CLI")
	}
}
