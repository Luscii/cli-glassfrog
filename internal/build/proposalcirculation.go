package build

import (
	"fmt"
	"strings"
)

// Proposal Circulation Path (068) is the fifth operator *path* the Claude plugin
// (062) established, and the second to cross the Write-Safety Guardrail (063) —
// this time twice: a thin `proposal-circulation` skill that delegates to a
// write-capable-but-fenced `proposal-circulator` agent under plugin/agents/,
// whose two writes (`proposal propose`, `proposal withdraw`) are both gated
// bodyless transitions. Like 062–067 it adds NO code to the Go CLI — the
// artifacts are declarative (a skill, an agent, and a single-sourced
// composed-leaf registry). This file gives internal/build read + validation
// access so the BDD suite can assert the artifacts are well-formed and the drift
// guard can keep the composed leaves truthful to the shipped CLI's command
// surface AND satisfy the two-write gated-membership invariant plan ADR-3/ADR-5
// depend on: both transitions are members of 063's gated set, and the two reads
// are not.
//
// internal/build stays cli-free by deliberate convention (see
// VersionInjectionTarget / operatororientation.go): the CLI's command surface is
// matched as strings against the CLI sources (LiveProposalSubcommands) rather
// than importing internal/cli and inverting the dependency.

// Repo-relative locations of the proposal-circulation-path artifacts
// (forward-slash; joined through filepath so the reads are OS-agnostic).
const (
	// ProposalCirculationSkillPath is the thin, discoverable entry point the host
	// loads on demand. Its frontmatter description is the trigger surface.
	ProposalCirculationSkillPath = "plugin/skills/proposal-circulation/SKILL.md"

	// ProposalCirculatorAgentPath is the write-capable-but-fenced subagent the
	// skill delegates circulation to. It is auto-discovered from plugin/agents/ by
	// directory convention — no `agents` key is added to plugin.json (063's
	// hooks.json and 064/065/066/067's agents confirmed directory auto-discovery;
	// ManifestDemandsNoSetup still forbids the key).
	ProposalCirculatorAgentPath = "plugin/agents/proposal-circulator.md"

	// ProposalCirculationCommandsPath is the single source of the `<group> <sub>`
	// leaves the circulator composes, read by BOTH the agent artifact (which names
	// exactly these leaves) and the drift guard (which checks each still resolves
	// in the CLI and that the two-write gated-membership invariant holds). Mirrors
	// 063's gated-commands.txt, 064's composed-reads.txt, 066's
	// tension-processing-commands.txt, and 067's proposal-drafting-commands.txt
	// (plan ADR-5).
	ProposalCirculationCommandsPath = "plugin/agents/proposal-circulation-commands.txt"
)

// ProposalCirculationGatedWrites names 068's two gated governance writes — the
// composed leaves that MUST each be a member of 063's gated set and always run
// through the confirmed write flow (spec/ADR-3: "exactly two gated governance
// writes… each always through the guardrail's confirmed flow"). They are
// explicit per-leaf contract anchors, checked in like 067's
// ProposalDraftingGatedWrite, because the gate-membership invariant needs an
// independent statement of WHICH leaves must be gated: the reads (proposal
// get/list) share the `proposal` group with the transitions, so deriving the
// writes from the gated set alone — or counting "exactly two composed leaves are
// gated" — cannot tell "both transitions are gated" from "a read was swapped in
// for a transition" (a swap preserves the count). The guard cross-checks each
// anchor against the composed leaf list, so the anchors can never silently name
// leaves the path does not actually compose.
var ProposalCirculationGatedWrites = []string{"proposal propose", "proposal withdraw"}

// ReadProposalCirculationSkill reads the committed proposal-circulation SKILL.md
// (frontmatter + body).
func ReadProposalCirculationSkill() (string, error) {
	raw, err := readRepoFile(ProposalCirculationSkillPath)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

// ReadProposalCirculatorAgent reads the committed proposal-circulator agent
// (frontmatter + body).
func ReadProposalCirculatorAgent() (string, error) {
	raw, err := readRepoFile(ProposalCirculatorAgentPath)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

// ReadProposalCirculationCommands reads the single-sourced composed-leaf
// registry, returning the `<group> <sub>` leaves it lists (proposal get,
// proposal propose, …). Comment (#) and blank lines are ignored; each remaining
// line is one leaf, interior whitespace collapsed — the same line format as
// 063's gated-commands.txt, so the membership check compares like with like.
func ReadProposalCirculationCommands() ([]string, error) {
	raw, err := readRepoFile(ProposalCirculationCommandsPath)
	if err != nil {
		return nil, err
	}
	return parseProposalCirculationCommands(string(raw)), nil
}

// parseProposalCirculationCommands extracts the composed leaves from registry
// content. Split out so the comment/blank-line skipping is unit-testable without
// a filesystem read. Shares the exact parsing shape of parseTensionCommands /
// parseProposalDraftingCommands / parseGatedRegistry so all five single-source
// registries read identically.
func parseProposalCirculationCommands(content string) []string {
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
// circulation artifacts naming a command the CLI no longer exposes — and to 063's
// gated proposal-write registry, so the two-write gated-membership invariant
// (plan ADR-3: each transition is confirmed BECAUSE it stays in the gated set;
// the reads never prompt BECAUSE they stay out of it) cannot silently break in
// either direction.
//
// It extends 067's gate-posture pattern to a two-in-two-out path. 066 requires
// every composed leaf to be ABSENT from the gated set (all-out); 067 requires its
// one gated write PRESENT and the rest absent (one-in); 068 requires BOTH
// transitions (ProposalCirculationGatedWrites) present and BOTH reads absent
// (two-in-two-out). The three registries are source-derived — the composed
// leaves from the registry, the gated set from 063's gated-commands.txt, the
// live surface from the CLI sources. The one thing that MUST be named is the
// identity of the two gated writes: they cannot be derived as "the composed
// leaves 063 gates", and a count ("exactly two composed leaves are gated")
// cannot tell "both transitions are gated" from "a read was swapped in for a
// transition" (a swap preserves the count and the group) — the very "transition
// ships unconfirmed" regression the guard exists to catch. So each write leaf is
// an explicit per-leaf contract anchor (ProposalCirculationGatedWrites), pinned
// like 067's ProposalDraftingGatedWrite, and cross-checked against the composed
// leaf list so a drift in either surfaces.
//
// It is best-effort and explicitly PARTIAL (plan ADR-5, stated not silent): it
// pins the EXISTENCE of the composed leaves and their GATE-MEMBERSHIP only. It
// deliberately does NOT verify their flags (deferred to `glassfrog proposal
// <sub> --help`), the circulation-record prose, the reads-inform-never-gate
// discipline (prompt-level; the BDD validation scenarios inspect it), or the
// gate script's command-string parsing robustness (063's own suite covers it).
// That gap is stated here rather than left silent (no silent caps).

// CheckProposalCirculationDrift returns one finding per way the composed-leaf
// registry, the CLI's proposal command surface, and 063's gated proposal-write
// registry have diverged. Empty means truthful. Each finding names the offending
// leaf so a CI failure points straight at it.
//
//	(a) every composed leaf must be a `proposal <sub>` pair — the circulation
//	    path composes the proposal group only; a leaf under any other group is
//	    reported, not silently accepted;
//	(b) every composed leaf's subcommand must still exist on the CLI's proposal
//	    command — else the artifacts name a command the CLI dropped or renamed;
//	(c) the gated-membership invariant, anchored per write leaf
//	    (ProposalCirculationGatedWrites): each write anchor is named in the
//	    composed list AND is a member of 063's gated set (else that transition
//	    ships unconfirmed), and every OTHER composed leaf (the reads) is absent
//	    from the gated set (else monitoring would start prompting);
//	(d) the proposal-circulator agent must name every composed leaf — so the
//	    artifact prose stays a genuine consumer of the single source and cannot
//	    silently drop one.
func CheckProposalCirculationDrift(composedLeaves, liveProposal, gatedLeaves []string, agent string) []string {
	var findings []string

	liveSet := setOf(liveProposal)
	gatedSet := setOf(gatedLeaves)
	composedSet := setOf(composedLeaves)
	writeSet := setOf(ProposalCirculationGatedWrites)

	for _, leaf := range composedLeaves {
		fields := strings.Fields(leaf)

		// (a) the composed set holds `proposal <sub>` pairs only.
		if len(fields) != 2 || fields[0] != "proposal" {
			findings = append(findings, fmt.Sprintf("composed leaf %q is not a `proposal <sub>` pair — an unexpected command must not enter the circulator's composed set (registry: %s)", leaf, ProposalCirculationCommandsPath))
			continue
		}

		// (b) the subcommand must still exist on the CLI's proposal command.
		if !liveSet[fields[1]] {
			findings = append(findings, fmt.Sprintf("composed leaf %q no longer exists as a subcommand of the CLI's proposal command — the circulation artifacts name a command the CLI dropped or renamed; fix the artifact or restore the command", leaf))
		}

		// (c, read side) every composed leaf OTHER than the two gated writes must be
		// absent from 063's gated set — a read that entered the gated set would make
		// monitoring start prompting.
		if !writeSet[leaf] && gatedSet[leaf] {
			findings = append(findings, fmt.Sprintf("composed leaf %q is a read but appears in 063's gated registry (%s) — monitoring would start prompting; remove it from the gated set", leaf, GatedRegistryPath))
		}

		// (d) the agent artifact must name every composed leaf — the prose is a
		// genuine consumer of the single source, not a divergent copy.
		if !mentionsToken(agent, leaf) {
			findings = append(findings, fmt.Sprintf("the proposal-circulator agent no longer names the composed leaf %q that %s lists — the artifact prose drifted from the single source", leaf, ProposalCirculationCommandsPath))
		}
	}

	// (c, write side) each of the two gated writes must be a composed leaf AND a
	// member of 063's gated set. The composed cross-check guards each anchor
	// against a drifted leaf list; the gated check is what catches a transition
	// being dropped from the registry (or swapped for a read) — that transition
	// would then ship unconfirmed.
	for _, write := range ProposalCirculationGatedWrites {
		if !composedSet[write] {
			findings = append(findings, fmt.Sprintf("the composed-leaf registry (%s) no longer names the path's gated write %q — either the artifact dropped it or the write leaf changed; reconcile the anchors with the composed list", ProposalCirculationCommandsPath, write))
		}
		if !gatedSet[write] {
			findings = append(findings, fmt.Sprintf("the path's gated write %q is not a member of 063's gated registry (%s) — that transition would ship unconfirmed; restore it to the gated set", write, GatedRegistryPath))
		}
	}

	return findings
}
