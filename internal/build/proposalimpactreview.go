package build

import (
	"fmt"
	"strings"
)

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

// --- Drift guard (T002) ----------------------------------------------------
//
// The drift guard anchors the single-sourced composed-leaf registry to the
// CLI's actual command surface, so a renamed or dropped command cannot leave
// the impact-review artifacts naming a command the CLI no longer exposes — and
// to 063's gated proposal-write registry, so the one-in-nine-out
// gate-membership invariant (plan ADR-3: the respond is confirmed BECAUSE it
// stays in the gated set; the nine review reads never prompt BECAUSE they stay
// out of it) cannot silently break in either direction.
//
// It extends the 067/068 gate-posture pattern to a one-in-nine-out path. 066
// requires every composed leaf ABSENT from the gated set (all-out); 067
// requires its one gated write PRESENT and the rest absent (one-in); 068
// requires both transitions present and both reads absent (two-in-two-out);
// 069 requires the respond (ProposalImpactReviewGatedWrite) present and all
// NINE reads absent (one-in-nine-out). The registries are source-derived — the
// composed leaves from the registry file, the gated set from 063's
// gated-commands.txt, the live surfaces from the CLI sources. The one thing
// that MUST be named is the identity of the gated write: it cannot be derived
// as "the composed leaf 063 gates", because the grounding/situating reads
// (proposal get/list) share the `proposal` group with the respond, so that
// derivation — or a count ("exactly one composed leaf is gated") — would pass
// if a read were swapped in for the respond (a swap preserves the count and
// the group), the very "response ships unconfirmed" regression the guard
// exists to catch. So the write leaf is an explicit per-leaf contract anchor
// (ProposalImpactReviewGatedWrite), pinned like 067's
// ProposalDraftingGatedWrite and 068's ProposalCirculationGatedWrites, and
// cross-checked against the composed leaf list so a drift in either surfaces.
//
// Unlike the 067/068 registries the 069 leaves are MIXED-SHAPE — the guard
// resolves each against its own anchor:
//   - `proposal <sub>` pairs against the CLI's proposal subcommand surface
//     (LiveProposalSubcommands);
//   - `me <sub>` pairs against the `me` subcommand surface (LiveMeSubcommands,
//     which also confirms `me` is still registered on root);
//   - the bare `me` leaf against that same wiring: `me` is variable-wired
//     (`MustRegister(root, meCmd)`), so LiveTopLevelCommands deliberately does
//     not see it — a non-empty liveMe surface is the proof the command
//     resolves, because LiveMeSubcommands errors when the root registration is
//     gone;
//   - single-token leaves (`roles`, `domains`, `policies` — top-level
//     commands, not `roles` subcommands) against the top-level surface
//     (LiveTopLevelCommands).
//
// Any other shape is reported as unanchorable rather than silently skipped
// (no silent caps).
//
// It is best-effort and explicitly PARTIAL (plan ADR-5, stated not silent): it
// pins the EXISTENCE of the composed leaves and their GATE-MEMBERSHIP only. It
// deliberately does NOT verify their flags (deferred to `glassfrog <cmd>
// --help`), the impact-picture prose, the footprint-honesty or
// reads-inform-never-decide disciplines (prompt-level; the BDD validation
// scenarios inspect them), or the gate script's command-string parsing
// robustness (063's own suite covers it). That gap is stated here rather than
// left silent (no silent caps).

// CheckProposalImpactReviewDrift returns one finding per way the composed-leaf
// registry, the CLI's command surface, and 063's gated proposal-write registry
// have diverged. Empty means truthful. Each finding names the offending leaf
// so a CI failure points straight at it.
//
//	(a) every composed leaf must resolve against its shape's anchor — the bare
//	    `me`, a `me <sub>` / `proposal <sub>` pair, or a top-level command
//	    word; a leaf of any other shape is reported, not silently accepted;
//	(b) the gated-membership invariant, anchored on the one gated write
//	    (ProposalImpactReviewGatedWrite): the write anchor is named in the
//	    composed list AND is a member of 063's gated set (else the response
//	    ships unconfirmed), and every OTHER composed leaf (the nine reads) is
//	    absent from the gated set (else the review would start prompting);
//	(c) each consumer still names its side of the single source: the
//	    proposal-impact-reviewer agent names every read leaf (its "Composed
//	    reads" section is a genuine consumer, not a divergent copy) and the
//	    skill names the respond leaf (its caller-context write step) — so the
//	    artifact prose cannot silently drop a leaf.
func CheckProposalImpactReviewDrift(composedLeaves, liveTop, liveMe, liveProposal, gatedLeaves []string, skill, agent string) []string {
	var findings []string

	topSet := setOf(liveTop)
	meSet := setOf(liveMe)
	proposalSet := setOf(liveProposal)
	gatedSet := setOf(gatedLeaves)
	composedSet := setOf(composedLeaves)

	for _, leaf := range composedLeaves {
		fields := strings.Fields(leaf)

		// (a) each leaf resolves against the anchor its shape names.
		switch {
		case leaf == "me":
			// The bare `me` is variable-wired on root, invisible to
			// LiveTopLevelCommands; a non-empty me-subcommand surface proves the
			// root registration (LiveMeSubcommands errors without it).
			if len(liveMe) == 0 {
				findings = append(findings, fmt.Sprintf("composed leaf %q could not be anchored — the `me` command's root registration (and its subcommand surface) was not resolved from %s", leaf, cliWiringSource))
			}
		case len(fields) == 1:
			if !topSet[leaf] {
				findings = append(findings, fmt.Sprintf("composed leaf %q no longer exists as a top-level command in the CLI — the impact-review artifacts name a read the CLI dropped or renamed; fix the artifact or restore the command (anchor: %s)", leaf, cliWiringSource))
			}
		case len(fields) == 2 && fields[0] == "me":
			if !meSet[fields[1]] {
				findings = append(findings, fmt.Sprintf("composed leaf %q no longer exists as a subcommand of `me` in the CLI — the impact-review artifacts name a read the CLI dropped or renamed; fix the artifact or restore the command (anchor: %s)", leaf, cliWiringSource))
			}
		case len(fields) == 2 && fields[0] == "proposal":
			if !proposalSet[fields[1]] {
				findings = append(findings, fmt.Sprintf("composed leaf %q no longer exists as a subcommand of the CLI's proposal command — the impact-review artifacts name a command the CLI dropped or renamed; fix the artifact or restore the command", leaf))
			}
		default:
			// The registry carries a command path the guard has no anchor for —
			// report it rather than silently skipping (no silent caps).
			findings = append(findings, fmt.Sprintf("composed leaf %q is a command path the drift guard cannot anchor (only bare `me`, `me <sub>`, `proposal <sub>`, and top-level commands are supported) — extend the guard or fix the registry (registry: %s)", leaf, ProposalImpactReviewCommandsPath))
			continue
		}

		// (b, read side) every composed leaf OTHER than the one gated write must
		// be absent from 063's gated set — a read that entered the gated set
		// would make the review start prompting.
		if leaf != ProposalImpactReviewGatedWrite && gatedSet[leaf] {
			findings = append(findings, fmt.Sprintf("composed leaf %q is a review read but appears in 063's gated registry (%s) — the review would start prompting; remove it from the gated set", leaf, GatedRegistryPath))
		}

		// (c) each consumer names its side of the single source: the agent the
		// read leaves (never the respond — that is the skill's caller-context
		// step), the skill the respond leaf.
		if leaf == ProposalImpactReviewGatedWrite {
			if !mentionsToken(skill, leaf) {
				findings = append(findings, fmt.Sprintf("the proposal-impact-review skill no longer names the gated write %q that %s lists — the caller-context respond step drifted from the single source", leaf, ProposalImpactReviewCommandsPath))
			}
		} else if !mentionsToken(agent, leaf) {
			findings = append(findings, fmt.Sprintf("the proposal-impact-reviewer agent no longer names the composed read leaf %q that %s lists — the artifact prose drifted from the single source", leaf, ProposalImpactReviewCommandsPath))
		}
	}

	// (b, write side) the one gated write must be a composed leaf AND a member
	// of 063's gated set. The composed cross-check guards the anchor against a
	// drifted leaf list; the gated check is what catches `proposal respond`
	// being dropped from the registry (or swapped for a read) — the response
	// would then ship unconfirmed.
	if !composedSet[ProposalImpactReviewGatedWrite] {
		findings = append(findings, fmt.Sprintf("the composed-leaf registry (%s) no longer names the path's one gated write %q — either the artifact dropped it or the write leaf changed; reconcile the anchor with the composed list", ProposalImpactReviewCommandsPath, ProposalImpactReviewGatedWrite))
	}
	if !gatedSet[ProposalImpactReviewGatedWrite] {
		findings = append(findings, fmt.Sprintf("the path's one gated write %q is not a member of 063's gated registry (%s) — the response would ship unconfirmed; restore it to the gated set", ProposalImpactReviewGatedWrite, GatedRegistryPath))
	}

	return findings
}
