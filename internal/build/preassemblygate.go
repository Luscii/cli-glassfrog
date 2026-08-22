package build

import (
	"regexp"
	"strings"
)

// Pre-Assembly Grammar Consultation (079) wires the consultation gate into the
// Proposal Drafting Path's shipped artifacts (067): the skill's workflow
// becomes the nine-step gate order, the drafter's fence grows the consultation
// read, and the draft record gains the consultation element. Like the sibling
// operator-path features it adds NO code to the Go CLI — every behavioral
// requirement lands as edits to the declarative plugin artifacts. This file
// gives internal/build the parsing access the BDD content-inspection suite
// needs to assert those artifacts' load-bearing structure: the ordered
// workflow steps, the frontmatter descriptions, and the raw registry text
// (comments included, for the annotation assertions). Parsing lives here in
// production source, per the operator-path discipline, so the helpers are
// shared and unit-testable rather than duplicated across test files.

// DraftingGateOrder is the checked-in contract anchor for the gate's step
// order — the canonical enumeration of the drafting workflow's nine steps in
// the order they must run. It cannot be derived from repo state (the skill
// artifact is the side under test, and the interface accord that pins the
// order is a pipeline artifact the operating surface never references), so it
// is pinned here like ProposalDraftingGatedWrite. The property each position
// encodes: routing precedes everything (the determination cannot be gated on
// its own answer), the consult precedes assembly (the grammar informs the
// build, it does not audit it), the match follows assembly with the routing
// answer in hand (the anchor-dependent shape needs both), and every gate step
// precedes the one confirmed create.
var DraftingGateOrder = []string{
	"Route",
	"Ground",
	"Situate",
	"Duplicate check",
	"Consult",
	"Assemble",
	"Match",
	"Confirm & create",
	"Hand off",
}

// DraftingWorkflowStep is one ordered step of the proposal-drafting skill's
// "The workflow" section: its bold name and the prose body that follows it
// (up to the next numbered step or the end of the numbered list).
type DraftingWorkflowStep struct {
	Name string
	Body string
}

// draftingStepRE matches a numbered workflow step's opening line:
// `1. **Route** — …`. The bold span is the step name.
var draftingStepRE = regexp.MustCompile(`(?m)^\d+\.\s+\*\*([^*]+)\*\*`)

// DraftingWorkflowSteps parses the ordered, numbered steps of the skill's
// "The workflow" section. It returns them in document order; an absent
// section or an empty list returns nil, which callers treat as a loud
// failure (a workflow that parses to nothing cannot satisfy any order).
func DraftingWorkflowSteps(skillRaw string) []DraftingWorkflowStep {
	section := draftingWorkflowSection(skillRaw)
	if section == "" {
		return nil
	}
	matches := draftingStepRE.FindAllStringSubmatchIndex(section, -1)
	var steps []DraftingWorkflowStep
	for i, m := range matches {
		end := len(section)
		if i+1 < len(matches) {
			end = matches[i+1][0]
		}
		steps = append(steps, DraftingWorkflowStep{
			Name: strings.TrimSpace(section[m[2]:m[3]]),
			Body: strings.TrimSpace(section[m[3]:end]),
		})
	}
	return steps
}

// draftingWorkflowSection returns the body of the `## The workflow` section:
// from its heading to the next `## `-level heading (a `### ` subsection like
// the relay stays inside the section, but the numbered list ends at any
// heading — draftingStepRE only matches numbered lines, so the subsection
// prose never reads as a step).
func draftingWorkflowSection(skillRaw string) string {
	const heading = "\n## The workflow"
	start := strings.Index(skillRaw, heading)
	if start < 0 {
		return ""
	}
	body := skillRaw[start+len(heading):]
	if end := strings.Index(body, "\n## "); end >= 0 {
		body = body[:end]
	}
	return body
}

// FrontmatterDescription returns the `description:` value of the artifact's
// leading YAML frontmatter block, or "" when the block or the field is
// absent. The plugin artifacts keep the description on a single line (the
// host's discovery surface), so a single-line read is the contract here.
func FrontmatterDescription(raw string) string {
	front, ok := frontmatterBlock(raw)
	if !ok {
		return ""
	}
	for _, line := range strings.Split(front, "\n") {
		if rest, isDesc := strings.CutPrefix(strings.TrimSpace(line), "description:"); isDesc {
			return strings.TrimSpace(rest)
		}
	}
	return ""
}

// ReadProposalDraftingRegistryRaw reads the composed-leaf registry verbatim —
// comment lines included — so content assertions can inspect the registry's
// annotations, not just the leaves ReadProposalDraftingCommands parses out.
func ReadProposalDraftingRegistryRaw() (string, error) {
	raw, err := readRepoFile(ProposalDraftingCommandsPath)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}
