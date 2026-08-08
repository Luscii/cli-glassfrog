package build

import (
	"fmt"
	"strings"
)

// Proposal Drafting Path (067) is the fourth operator *path* the Claude plugin
// (062) established, and the first to cross the Write-Safety Guardrail (063): a
// thin `proposal-drafting` skill that delegates to a write-capable-but-fenced
// `proposal-drafter` agent under plugin/agents/. Like 062–066 it adds NO code to
// the Go CLI — the artifacts are declarative (a skill, an agent, and a
// single-sourced composed-leaf registry). This file gives internal/build read +
// validation access so the BDD suite can assert the artifacts are well-formed and
// the drift guard can keep the composed leaves truthful to the shipped CLI's
// command surface AND satisfy the gated-membership invariant plan ADR-3/ADR-5
// depends on: the one write (`proposal create`) is a member of 063's gated set,
// and the situating reads are not.
//
// internal/build stays cli-free by deliberate convention (see
// VersionInjectionTarget / operatororientation.go): the CLI's command surface is
// matched as strings against the CLI sources (LiveTensionSubcommands,
// LiveProposalSubcommands, LiveTopLevelCommands, LiveMeSubcommands) rather than
// importing internal/cli and inverting the dependency.

// Repo-relative locations of the proposal-drafting-path artifacts (forward-slash;
// joined through filepath so the reads are OS-agnostic).
const (
	// ProposalDraftingSkillPath is the thin, discoverable entry point the host
	// loads on demand. Its frontmatter description is the trigger surface.
	ProposalDraftingSkillPath = "plugin/skills/proposal-drafting/SKILL.md"

	// ProposalDrafterAgentPath is the write-capable-but-fenced subagent the skill
	// delegates drafting to. It is auto-discovered from plugin/agents/ by directory
	// convention — no `agents` key is added to plugin.json (063's hooks.json and
	// 064/065/066's agents confirmed directory auto-discovery; ManifestDemandsNoSetup
	// still forbids the key).
	ProposalDrafterAgentPath = "plugin/agents/proposal-drafter.md"

	// ProposalDraftingCommandsPath is the single source of the `<group> <sub>` leaves
	// the drafter composes, read by BOTH the agent artifact (which names exactly
	// these leaves) and the drift guard (which checks each still resolves in the CLI
	// and that the gated-membership invariant holds). Mirrors 063's gated-commands.txt,
	// 064's composed-reads.txt, and 066's tension-processing-commands.txt (plan ADR-5).
	ProposalDraftingCommandsPath = "plugin/agents/proposal-drafting-commands.txt"
)

// ReadProposalDraftingSkill reads the committed proposal-drafting SKILL.md
// (frontmatter + body).
func ReadProposalDraftingSkill() (string, error) {
	raw, err := readRepoFile(ProposalDraftingSkillPath)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

// ReadProposalDrafterAgent reads the committed proposal-drafter agent
// (frontmatter + body).
func ReadProposalDrafterAgent() (string, error) {
	raw, err := readRepoFile(ProposalDrafterAgentPath)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

// ReadProposalDraftingCommands reads the single-sourced composed-leaf registry,
// returning the `<group> <sub>` leaves it lists (tension get, proposal create, …).
// Comment (#) and blank lines are ignored; each remaining line is one leaf,
// interior whitespace collapsed — the same line format as 063's gated-commands.txt,
// so the membership check compares like with like.
func ReadProposalDraftingCommands() ([]string, error) {
	raw, err := readRepoFile(ProposalDraftingCommandsPath)
	if err != nil {
		return nil, err
	}
	return parseProposalDraftingCommands(string(raw)), nil
}

// parseProposalDraftingCommands extracts the composed leaves from registry
// content. Split out so the comment/blank-line skipping is unit-testable without a
// filesystem read. Shares the exact parsing shape of parseTensionCommands /
// parseGatedRegistry so all four single-source registries read identically.
func parseProposalDraftingCommands(content string) []string {
	var leaves []string
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		leaves = append(leaves, strings.Join(strings.Fields(trimmed), " "))
	}
	return leaves
}

// --- Drift guard (T002) ----------------------------------------------------
//
// The drift guard anchors the single-sourced composed-leaf registry to the CLI's
// actual command surface, so a renamed or dropped command cannot leave the
// drafting artifacts naming a command the CLI no longer exposes — and to 063's
// gated proposal-write registry, so the gated-membership invariant (plan ADR-3:
// the one write is confirmed BECAUSE it stays in the gated set; the situating
// reads never prompt BECAUSE they stay out of it) cannot silently break in either
// direction.
//
// It is the INVERSE of 066's disjointness assertion. 066 requires every composed
// tension leaf to be ABSENT from the gated set (all its writes are ungated); 067
// requires its one gated write (ProposalDraftingGatedWrite) to be PRESENT in the
// gated set and every other composed leaf (the situating reads) to be ABSENT. The
// three registries are source-derived — the composed leaves from the registry,
// the gated set from 063's gated-commands.txt, the live surfaces from the CLI
// sources. The one thing that MUST be named is the identity of the gated write: it
// cannot be derived as "the single composed leaf 063 gates", because a read and
// the create both live in the `proposal` group, so that derivation would pass if
// `proposal create` were dropped from the gated set and a read added in its place
// (both leave exactly one gated composed leaf, both in the `proposal` group) — the
// very "create ships unconfirmed" regression the guard exists to catch. So the
// write leaf is an explicit contract anchor (ProposalDraftingGatedWrite), pinned
// like 063's expectedProposalSurface, and cross-checked against the composed leaf
// list so a drift in either surfaces.
//
// It is best-effort and explicitly PARTIAL (plan ADR-5, stated not silent): it
// pins the EXISTENCE of the composed leaves and their GATE-MEMBERSHIP only. It
// deliberately does NOT verify their flags (deferred to `glassfrog <group> <sub>
// --help`), the draft-record prose, the confirmation-narration wording (the BDD
// content-inspection scenarios cover those required phrases), or the gate script's
// command-string parsing robustness (063's own suite covers it). That gap is
// stated here rather than left silent (no silent caps).

// ProposalDraftingGatedWrite is 067's one gated governance write — the single
// composed leaf that MUST be a member of 063's gated set and always run through
// the confirmed write flow (spec/ADR-3: "the path performs exactly one gated
// write — the proposal create"). It is an explicit contract anchor, checked in
// like 063's expectedProposalSurface, because the gate-membership invariant needs
// an independent statement of WHICH leaf must be gated: a situating read
// (proposal get/list) shares the `proposal` group with the create, so deriving
// the write from the gated set alone cannot tell "the create is gated" from "a
// read was swapped in for the create" — the exact regression that would ship the
// create unconfirmed. The guard cross-checks this anchor against the composed leaf
// list, so it can never silently name a leaf the path does not actually compose.
const ProposalDraftingGatedWrite = "proposal create"

// CheckProposalDraftingDrift returns one finding per way the composed-leaf
// registry, the CLI's command surface, and 063's gated proposal-write registry
// have diverged. Empty means truthful. Each finding names the offending leaf so a
// CI failure points straight at it.
//
//	(a+b) every composed leaf must resolve on the shipped CLI through a four-way
//	    leaf resolution (transplanted from 065's CheckConstraintDrift when 073
//	    widened the composed surface to the three routing reads): a single-token
//	    leaf against the top-level surface, a `me <sub>` leaf against the `me`
//	    subcommands, and a `tension <sub>`/`proposal <sub>` leaf against the
//	    matching group — with an unanchorable-default arm that REPORTS a command
//	    path the guard cannot anchor rather than silently skipping it (no silent
//	    caps); else the artifacts name a command the CLI dropped or renamed;
//	(c) the gated-membership invariant, anchored on the one gated write
//	    (ProposalDraftingGatedWrite): the write anchor is named in the composed
//	    list AND is a member of 063's gated set (else the create ships
//	    unconfirmed), and every OTHER composed leaf is absent from the gated set
//	    (else a read would start prompting);
//	(d) the proposal-drafter agent must name every composed leaf — so the artifact
//	    prose stays a genuine consumer of the single source and cannot silently
//	    drop one.
func CheckProposalDraftingDrift(composedLeaves, liveTop, liveMe, liveTension, liveProposal, gatedLeaves []string, agent string) []string {
	var findings []string

	liveByGroup := map[string]map[string]bool{
		"tension":  setOf(liveTension),
		"proposal": setOf(liveProposal),
	}
	topSet := setOf(liveTop)
	meSet := setOf(liveMe)
	gatedSet := setOf(gatedLeaves)
	composedSet := setOf(composedLeaves)

	for _, leaf := range composedLeaves {
		fields := strings.Fields(leaf)

		// (a+b) four-way leaf resolution against the live surfaces, with the
		// unanchorable default reporting rather than skipping.
		switch {
		case len(fields) == 1:
			if !topSet[leaf] {
				findings = append(findings, fmt.Sprintf("composed leaf %q no longer exists as a top-level command in the CLI — the drafting artifacts name a command the CLI dropped or renamed; fix the artifact or restore the command", leaf))
			}
		case len(fields) == 2 && fields[0] == "me":
			if !meSet[fields[1]] {
				findings = append(findings, fmt.Sprintf("composed leaf %q no longer exists as a subcommand of `me` in the CLI — the drafting artifacts name a command the CLI dropped or renamed; fix the artifact or restore the command", leaf))
			}
		case len(fields) == 2 && (fields[0] == "tension" || fields[0] == "proposal"):
			if !liveByGroup[fields[0]][fields[1]] {
				findings = append(findings, fmt.Sprintf("composed leaf %q no longer exists as a subcommand of the CLI's %s command — the drafting artifacts name a command the CLI dropped or renamed; fix the artifact or restore the command", leaf, fields[0]))
			}
		default:
			findings = append(findings, fmt.Sprintf("composed leaf %q is a command path the drift guard cannot anchor (only top-level commands, `me <sub>`, `tension <sub>`, and `proposal <sub>` are supported) — extend the guard or fix the registry (registry: %s)", leaf, ProposalDraftingCommandsPath))
			continue
		}

		// (c, read side) every composed leaf OTHER than the one gated write must be
		// absent from 063's gated set — a read that entered the gated set would
		// start prompting.
		if leaf != ProposalDraftingGatedWrite && gatedSet[leaf] {
			findings = append(findings, fmt.Sprintf("composed leaf %q is a read but appears in 063's gated registry (%s) — a read would start prompting; remove it from the gated set", leaf, GatedRegistryPath))
		}

		// (d) the agent artifact must name every composed leaf — the prose is a
		// genuine consumer of the single source, not a divergent copy.
		if !mentionsToken(agent, leaf) {
			findings = append(findings, fmt.Sprintf("the proposal-drafter agent no longer names the composed leaf %q that %s lists — the artifact prose drifted from the single source", leaf, ProposalDraftingCommandsPath))
		}
	}

	// (c, write side) the one gated write must be a composed leaf AND a member of
	// 063's gated set. The composed cross-check guards the anchor against a drifted
	// leaf list; the gated check is what catches `proposal create` being dropped
	// from the registry (or swapped for a read) — the create would then ship
	// unconfirmed.
	if !composedSet[ProposalDraftingGatedWrite] {
		findings = append(findings, fmt.Sprintf("the composed-leaf registry (%s) no longer names the path's one gated write %q — either the artifact dropped it or the write leaf changed; reconcile the anchor with the composed list", ProposalDraftingCommandsPath, ProposalDraftingGatedWrite))
	}
	if !gatedSet[ProposalDraftingGatedWrite] {
		findings = append(findings, fmt.Sprintf("the path's one gated write %q is not a member of 063's gated registry (%s) — the create would ship unconfirmed; restore it to the gated set", ProposalDraftingGatedWrite, GatedRegistryPath))
	}

	return findings
}

// setOf builds a lookup set from a slice — the small helper the drift check uses
// so its membership tests read as set operations.
func setOf(items []string) map[string]bool {
	set := make(map[string]bool, len(items))
	for _, item := range items {
		set[item] = true
	}
	return set
}
