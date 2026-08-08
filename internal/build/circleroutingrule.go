package build

import (
	"regexp"
	"strings"
)

// CircleRoutingRulePath is the circle-routing record, relative to the
// repository root — a hand-authored knowledge artifact owned by the
// proposal-drafting skill (073, plan ADR-1), sibling to the change-set grammar
// record in the same references/ directory. It carries the routing rule (a
// proposal inherits the circle of its anchor tension's sensing role), the
// classification test, and the procedure with its named-reads block. It is the
// shared constant the guard reads and the pre-assembly gate (#77) will load;
// single-sourced here alongside the family's other path constants.
const CircleRoutingRulePath = "plugin/skills/proposal-drafting/references/circle-routing-rule.md"

// Required section headings of the record (interface-spec anatomy rows 3–6).
// The headings are the machine anchors the parser and the guard share.
const (
	routingSectionCitations      = "Contract citations"
	routingSectionRule           = "Rule"
	routingSectionClassification = "Classification test"
	routingSectionProcedure      = "Procedure"
)

// CircleRoutingRuleRecord is the parsed record: the leading empirical marker
// text, the document-header field lines (Owner, Contract citations), each
// required section's bold-labelled fields keyed by label, and the named-reads
// block's leaves in declaration order. Every record-side fact the guard and
// the content scenarios need is derived here from the file; nothing about the
// record's content is hard-coded elsewhere (plan ADR-4).
type CircleRoutingRuleRecord struct {
	Raw                  string
	Marker               string
	HeaderFields         map[string]string
	CitationFields       map[string]string
	RuleFields           map[string]string
	ClassificationFields map[string]string
	ProcedureFields      map[string]string
	NamedReads           []string
}

// ReadCircleRoutingRuleRecord reads the record from the repository root. A
// missing or unreadable file is an error.
func ReadCircleRoutingRuleRecord() (string, error) {
	raw, err := readRepoFile(CircleRoutingRulePath)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

// ParseCircleRoutingRuleRecord derives every record-side fact from the raw
// markdown. It never reads the vendored spec — the spec side is derived
// separately so the two sources stay independent.
func ParseCircleRoutingRuleRecord(raw string) CircleRoutingRuleRecord {
	procedure := routingSectionBody(raw, routingSectionProcedure)
	return CircleRoutingRuleRecord{
		Raw:                  raw,
		Marker:               parseLeadingMarker(raw),
		HeaderFields:         parseLabelledFields(routingHeaderRegion(raw)),
		CitationFields:       parseLabelledFields(routingSectionBody(raw, routingSectionCitations)),
		RuleFields:           parseLabelledFields(routingSectionBody(raw, routingSectionRule)),
		ClassificationFields: parseLabelledFields(routingSectionBody(raw, routingSectionClassification)),
		ProcedureFields:      parseLabelledFields(procedure),
		NamedReads:           parseNamedReadsBlock(procedure),
	}
}

// routingTitleRE matches the document-header title line.
var routingTitleRE = regexp.MustCompile(`(?m)^#\s+Circle Routing Rule\s*$`)

// routingHeaderRegion returns the document-header region — from the line after
// the `# Circle Routing Rule` title to the first `## ` section heading — where
// the Owner and Contract citations field lines live. Empty if the title is
// absent.
func routingHeaderRegion(raw string) string {
	loc := routingTitleRE.FindStringIndex(raw)
	if loc == nil {
		return ""
	}
	region := raw[loc[1]:]
	if next := sectionRE.FindStringIndex(region); next != nil {
		region = region[:next[0]]
	}
	return region
}

// routingSectionBody returns the body of the `## <heading>` section (heading
// line excluded), up to the next `## ` heading or EOF. Empty if the section is
// absent.
func routingSectionBody(raw, heading string) string {
	locs := sectionRE.FindAllStringSubmatchIndex(raw, -1)
	for i, loc := range locs {
		if !strings.EqualFold(strings.TrimSpace(raw[loc[2]:loc[3]]), heading) {
			continue
		}
		start := loc[1]
		end := len(raw)
		if i+1 < len(locs) {
			end = locs[i+1][0]
		}
		return raw[start:end]
	}
	return ""
}

// parseNamedReadsBlock returns the read leaves declared in the procedure
// section's fenced code block, one per line in declaration order — the
// record's declaration surface (plan ADR-2), using the *-commands.txt token
// grammar: a single token for a top-level command, two tokens for
// `<group> <sub>`. Empty if the section carries no fenced block or the block
// is empty.
func parseNamedReadsBlock(procedureBody string) []string {
	lines := strings.Split(procedureBody, "\n")
	var reads []string
	inFence := false
	for _, line := range lines {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "```") {
			if inFence {
				break // closing fence: the first block is the declaration
			}
			inFence = true
			continue
		}
		if inFence && t != "" {
			reads = append(reads, t)
		}
	}
	return reads
}
