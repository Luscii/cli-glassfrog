package build

import "strings"

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
