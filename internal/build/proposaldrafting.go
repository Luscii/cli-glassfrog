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
// LiveProposalSubcommands) rather than importing internal/cli and inverting the
// dependency.

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
// requires EXACTLY ONE composed leaf to be PRESENT in the gated set (its one
// gated create) and the rest absent (the situating reads). Both are fully
// source-derived — the composed leaves from the registry, the gated set from
// 063's gated-commands.txt, the live surfaces from the CLI sources — so the guard
// hard-codes none of them, including the identity of the write leaf: it is
// derived as "the single composed leaf 063 gates", never named literally
// (LEARNINGS: a drift guard must not hard-code the value it guards).
//
// It is best-effort and explicitly PARTIAL (plan ADR-5, stated not silent): it
// pins the EXISTENCE of the composed leaves and their GATE-MEMBERSHIP only. It
// deliberately does NOT verify their flags (deferred to `glassfrog <group> <sub>
// --help`), the draft-record prose, the confirmation-narration wording (the BDD
// content-inspection scenarios cover those required phrases), or the gate script's
// command-string parsing robustness (063's own suite covers it). That gap is
// stated here rather than left silent (no silent caps).

// CheckProposalDraftingDrift returns one finding per way the composed-leaf
// registry, the CLI's command surface, and 063's gated proposal-write registry
// have diverged. Empty means truthful. Each finding names the offending leaf so a
// CI failure points straight at it.
//
//	(a) every composed leaf must be a `tension <sub>` or `proposal <sub>` pair —
//	    a leaf under any other group is reported, not silently accepted;
//	(b) every composed leaf's subcommand must still exist on the matching CLI
//	    command (tension leaves against liveTension, proposal leaves against
//	    liveProposal) — else the artifacts name a command the CLI dropped or renamed;
//	(c) the gated-membership invariant: EXACTLY ONE composed leaf may be a member
//	    of 063's gated set (the one gated create), and it must be a `proposal`
//	    write. Zero gated → the create left the gated registry and would ship
//	    unconfirmed; more than one → a situating read entered the gated set and
//	    would start prompting;
//	(d) the proposal-drafter agent must name every composed leaf — so the artifact
//	    prose stays a genuine consumer of the single source and cannot silently
//	    drop one.
func CheckProposalDraftingDrift(composedLeaves, liveTension, liveProposal, gatedLeaves []string, agent string) []string {
	var findings []string

	liveByGroup := map[string]map[string]bool{
		"tension":  setOf(liveTension),
		"proposal": setOf(liveProposal),
	}
	gatedSet := setOf(gatedLeaves)

	var gatedComposed []string
	for _, leaf := range composedLeaves {
		fields := strings.Fields(leaf)

		// (a) the composed set holds `tension <sub>` / `proposal <sub>` pairs only.
		if len(fields) != 2 || (fields[0] != "tension" && fields[0] != "proposal") {
			findings = append(findings, fmt.Sprintf("composed leaf %q is not a `tension <sub>` or `proposal <sub>` pair — an unexpected command must not enter the drafter's composed set (registry: %s)", leaf, ProposalDraftingCommandsPath))
			continue
		}

		// (b) the subcommand must still exist on the matching CLI command.
		if !liveByGroup[fields[0]][fields[1]] {
			findings = append(findings, fmt.Sprintf("composed leaf %q no longer exists as a subcommand of the CLI's %s command — the drafting artifacts name a command the CLI dropped or renamed; fix the artifact or restore the command", leaf, fields[0]))
		}

		// Collect the composed leaves 063 gates — the raw material for (c).
		if gatedSet[leaf] {
			gatedComposed = append(gatedComposed, leaf)
		}

		// (d) the agent artifact must name every composed leaf — the prose is a
		// genuine consumer of the single source, not a divergent copy.
		if !mentionsToken(agent, leaf) {
			findings = append(findings, fmt.Sprintf("the proposal-drafter agent no longer names the composed leaf %q that %s lists — the artifact prose drifted from the single source", leaf, ProposalDraftingCommandsPath))
		}
	}

	// (c) the gated-membership invariant — exactly one composed leaf is gated (the
	// one create), and it is a proposal write. Derived from source: the write leaf
	// is "the composed leaf 063 gates", never named literally here.
	switch {
	case len(gatedComposed) == 0:
		findings = append(findings, fmt.Sprintf("no composed leaf is a member of 063's gated proposal-write registry (%s) — the path's one gated write (its create) has left the gated set and would ship unconfirmed; restore it to the registry or fix the artifact", GatedRegistryPath))
	case len(gatedComposed) > 1:
		findings = append(findings, fmt.Sprintf("more than one composed leaf is gated by 063 (%v) — the path performs exactly one gated write, so a situating read has entered the gated registry (%s) and would start prompting; remove it from the gated set", gatedComposed, GatedRegistryPath))
	default:
		if fields := strings.Fields(gatedComposed[0]); len(fields) == 2 && fields[0] != "proposal" {
			findings = append(findings, fmt.Sprintf("the single gated composed leaf %q is not a `proposal` write — the path's one gated write must be a governance (proposal) write (registry: %s)", gatedComposed[0], GatedRegistryPath))
		}
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
