package build

import "strings"

// Proposal Impact Review Path (069) is the sixth and last operator *path* the
// Claude plugin (062) established, and the third to cross the Write-Safety
// Guardrail (063) — with a structural novelty: the review runs in a pure-read
// `proposal-impact-reviewer` agent under plugin/agents/, while the path's one
// gated write (`proposal respond`) is the thin `proposal-impact-review` skill's
// caller-context step, taken only after the operator decides (plan ADR-3's
// announced divergence from 067/068's in-subagent write locus). Like 062–068 it
// adds NO code to the Go CLI — the artifacts are declarative (a skill, an
// agent, and a single-sourced composed-leaf registry). This file gives
// internal/build read + validation access so the BDD suite can assert the
// artifacts are well-formed and the drift guard can keep the composed leaves
// truthful to the shipped CLI's command surface AND satisfy the one-in-nine-out
// gate-membership invariant plan ADR-3/ADR-5 depend on: the respond is a member
// of 063's gated set, and the nine review reads are not.
//
// internal/build stays cli-free by deliberate convention (see
// VersionInjectionTarget / operatororientation.go): the CLI's command surface is
// matched as strings against the CLI sources (LiveProposalSubcommands,
// LiveMeSubcommands, LiveTopLevelCommands) rather than importing internal/cli
// and inverting the dependency.

// Repo-relative locations of the proposal-impact-review-path artifacts
// (forward-slash; joined through filepath so the reads are OS-agnostic).
const (
	// ProposalImpactReviewSkillPath is the thin, discoverable entry point the
	// host loads on demand. Its frontmatter description is the trigger surface;
	// its body owns the workflow's two interaction moments (pick a pending
	// proposal, receive the operator's response decision) and the caller-context
	// respond step.
	ProposalImpactReviewSkillPath = "plugin/skills/proposal-impact-review/SKILL.md"

	// ProposalImpactReviewerAgentPath is the pure-read subagent the skill
	// delegates the review to. It is auto-discovered from plugin/agents/ by
	// directory convention — no `agents` key is added to plugin.json (063's
	// hooks.json and 064–068's agents confirmed directory auto-discovery;
	// ManifestDemandsNoSetup still forbids the key).
	ProposalImpactReviewerAgentPath = "plugin/agents/proposal-impact-reviewer.md"

	// ProposalImpactReviewCommandsPath is the single source of the command-path
	// leaves the path composes, read by the agent artifact (whose "Composed
	// reads" section names exactly the nine read leaves), the skill (whose
	// decision-and-respond step runs the one gated write leaf), and the drift
	// guard (which checks each leaf still resolves in the CLI and that the
	// one-in-nine-out gate-membership invariant holds). Mirrors 063's
	// gated-commands.txt and the 064–068 registry files (plan ADR-5).
	ProposalImpactReviewCommandsPath = "plugin/agents/proposal-impact-review-commands.txt"
)

// ProposalImpactReviewGatedWrite is 069's one gated governance write — the
// single composed leaf that MUST be a member of 063's gated set and always run
// through the confirmed write flow (spec/ADR-3: the operator-chosen response is
// recorded in the caller's context, through the guardrail's confirmed flow). It
// is an explicit per-leaf contract anchor, checked in like 067's
// ProposalDraftingGatedWrite and 068's ProposalCirculationGatedWrites, because
// the gate-membership invariant needs an independent statement of WHICH leaf
// must be gated: the grounding/situating reads (proposal get/list) share the
// `proposal` group with the respond, so deriving the write from the gated set
// alone — or counting "exactly one composed leaf is gated" — cannot tell "the
// respond is gated" from "a read was swapped in for the respond" (a swap
// preserves the count and the group), the very "response ships unconfirmed"
// regression the guard exists to catch. The guard cross-checks this anchor
// against the composed leaf list, so it can never silently name a leaf the path
// does not actually compose.
const ProposalImpactReviewGatedWrite = "proposal respond"

// ReadProposalImpactReviewSkill reads the committed proposal-impact-review
// SKILL.md (frontmatter + body).
func ReadProposalImpactReviewSkill() (string, error) {
	raw, err := readRepoFile(ProposalImpactReviewSkillPath)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

// ReadProposalImpactReviewerAgent reads the committed proposal-impact-reviewer
// agent (frontmatter + body).
func ReadProposalImpactReviewerAgent() (string, error) {
	raw, err := readRepoFile(ProposalImpactReviewerAgentPath)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

// ReadProposalImpactReviewCommands reads the single-sourced composed-leaf
// registry, returning the command-path leaves it lists (proposal get, me roles,
// policies, …). Comment (#) and blank lines are ignored; each remaining line is
// one leaf, interior whitespace collapsed — the same line format as 063's
// gated-commands.txt, so the membership check compares like with like. Unlike
// the 067/068 registries the leaves are mixed-shape: two-token `proposal <sub>`
// / `me <sub>` paths, the bare `me`, and single-token top-level reads.
func ReadProposalImpactReviewCommands() ([]string, error) {
	raw, err := readRepoFile(ProposalImpactReviewCommandsPath)
	if err != nil {
		return nil, err
	}
	return parseProposalImpactReviewCommands(string(raw)), nil
}

// parseProposalImpactReviewCommands extracts the composed leaves from registry
// content. Split out so the comment/blank-line skipping is unit-testable
// without a filesystem read. Shares the exact parsing shape of
// parseTensionCommands / parseProposalDraftingCommands /
// parseProposalCirculationCommands / parseGatedRegistry so all six
// single-source registries read identically.
func parseProposalImpactReviewCommands(content string) []string {
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
