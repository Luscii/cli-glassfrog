package build

import (
	"fmt"
	"regexp"
	"strings"

	"sigs.k8s.io/yaml"
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

// Required bold-labelled fields per surface (interface-spec anatomy rows 2–6).
// These label sets are the structural contract the interface skill pinned — a
// checked-in contract fact, not a value derivable from another file — so
// pinning them here is correct rather than a second source of truth. A missing
// or empty one is guard condition 2.
var (
	routingHeaderRequiredFields         = []string{"Owner", "Contract citations"}
	routingCitationRequiredFields       = []string{"Premise", "Circle indicator", "Root signal"}
	routingRuleRequiredFields           = []string{"Mechanism", "Own-circle consequence", "Circle Lead exception", "Root circle"}
	routingClassificationRequiredFields = []string{"Test", "Resolved by", "Parent resolution"}
	routingProcedureRequiredFields      = []string{"Answer shape", "All anchors named", "Gap reporting", "Uncertainty"}
)

// RoleCitation is one `Role.<field>` schema anchor the record cites, paired
// with the section citing it — so a dropped Role field fails AT the section
// citing it (guard condition 9).
type RoleCitation struct {
	Field   string
	Section string
}

// CircleRoutingRuleRecord is the parsed record: the leading empirical marker
// text, the document-header field lines (Owner, Contract citations), each
// required section's bold-labelled fields keyed by label, the named-reads
// block's leaves in declaration order, and the cited schema anchors (the
// premise's property set and every `Role.<field>` citation). Every record-side
// fact the guard and the content scenarios need is derived here from the file;
// nothing about the record's content is hard-coded elsewhere (plan ADR-4).
type CircleRoutingRuleRecord struct {
	Raw                  string
	Marker               string
	HeaderFields         map[string]string
	CitationFields       map[string]string
	RuleFields           map[string]string
	ClassificationFields map[string]string
	ProcedureFields      map[string]string
	NamedReads           []string
	CitedPremiseProps    []string
	RoleCitations        []RoleCitation
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
	citations := parseLabelledFields(routingSectionBody(raw, routingSectionCitations))
	return CircleRoutingRuleRecord{
		Raw:                  raw,
		Marker:               parseLeadingMarker(raw),
		HeaderFields:         parseLabelledFields(routingHeaderRegion(raw)),
		CitationFields:       citations,
		RuleFields:           parseLabelledFields(routingSectionBody(raw, routingSectionRule)),
		ClassificationFields: parseLabelledFields(routingSectionBody(raw, routingSectionClassification)),
		ProcedureFields:      parseLabelledFields(procedure),
		NamedReads:           parseNamedReadsBlock(procedure),
		CitedPremiseProps:    parseCitedPremiseProps(citations["Premise"]),
		RoleCitations:        parseRoleCitations(raw),
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

// premisePropRE matches a backticked snake_case property name — the shape of a
// request property (`tension_id`, `changes`). It deliberately requires the
// whole backticked span to be a lowercase identifier, so the dotted schema
// anchor (`CreateProposalRequest.properties.proposal`) does not match: that is
// the citation path, not a cited property.
var premisePropRE = regexp.MustCompile("`([a-z][a-z0-9_]*)`")

// parseCitedPremiseProps extracts the property names the record's Premise
// citation says the create request's proposal object carries — the record-side
// half of the premise tripwire (guard condition 8). Derived from the record,
// never hard-coded, so the guard compares the record's own claim against the
// vendored spec.
func parseCitedPremiseProps(premise string) []string {
	var out []string
	seen := map[string]bool{}
	for _, m := range premisePropRE.FindAllStringSubmatch(premise, -1) {
		if !seen[m[1]] {
			seen[m[1]] = true
			out = append(out, m[1])
		}
	}
	return out
}

// roleCitationRE matches a backticked `Role.<field>` schema anchor.
var roleCitationRE = regexp.MustCompile("`Role\\.([A-Za-z0-9_]+)`")

// parseRoleCitations collects every `Role.<field>` anchor the record cites,
// paired with the required section citing it, in section order. Derived from
// the record so the guard checks exactly the anchors the record rests on —
// no schema field names are hard-coded (plan ADR-4).
func parseRoleCitations(raw string) []RoleCitation {
	sections := []string{
		routingSectionCitations,
		routingSectionRule,
		routingSectionClassification,
		routingSectionProcedure,
	}
	var cites []RoleCitation
	for _, section := range sections {
		body := routingSectionBody(raw, section)
		seen := map[string]bool{}
		for _, m := range roleCitationRE.FindAllStringSubmatch(body, -1) {
			if !seen[m[1]] {
				seen[m[1]] = true
				cites = append(cites, RoleCitation{Field: m[1], Section: section})
			}
		}
	}
	return cites
}

// RoutingMarkerIsWellFormed reports whether the leading marker carries the
// required phrase and states the cite-versus-observe split (interface-spec
// anatomy row 1 / condition 3): the absent circle parameter is published
// contract, while where the proposal lands is observed behaviour. A marker
// absent or degraded on any of these is not well-formed.
func RoutingMarkerIsWellFormed(marker string) bool {
	if !strings.Contains(marker, "Empirical record") {
		return false
	}
	low := strings.ToLower(marker)
	// The cite half: the circle parameter's absence is published contract.
	citesContract := strings.Contains(low, "circle parameter") && strings.Contains(low, "contract")
	// The observe half: where the proposal lands is observed behaviour.
	statesObserved := strings.Contains(low, "observed") && (strings.Contains(low, "lands") || strings.Contains(low, "landing"))
	return citesContract && statesObserved
}

// --- The spec side: the published contract the record cites --------------

// routingSpecSchema is the narrow view of the vendored spec the guard needs:
// the create request's proposal property set (the premise tripwire's spec
// side) and the Role schema's property set (the classification anchors).
// sigs.k8s.io/yaml converts YAML→JSON, so the json tags drive the decode;
// every other field in the large spec is ignored.
type routingSpecSchema struct {
	Components struct {
		Schemas struct {
			CreateProposalRequest struct {
				Properties struct {
					Proposal struct {
						Properties map[string]interface{} `json:"properties"`
					} `json:"proposal"`
				} `json:"properties"`
			} `json:"CreateProposalRequest"`
			Role struct {
				Properties map[string]interface{} `json:"properties"`
			} `json:"Role"`
		} `json:"schemas"`
	} `json:"components"`
}

// LoadSpecRoutingAnchors reads the vendored spec from the repository root and
// returns the two spec-side sets the guard compares the record against: the
// property-name set of CreateProposalRequest.properties.proposal, and the
// property-name set of the Role schema. A missing or unparseable spec is an
// error; an empty set on either side is too (the anchors the record cites must
// resolve to something).
func LoadSpecRoutingAnchors() (proposalProps, roleProps []string, err error) {
	raw, err := readRepoFile(VendoredSpecPath)
	if err != nil {
		return nil, nil, err
	}
	var doc routingSpecSchema
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return nil, nil, fmt.Errorf("parsing %s: %w", VendoredSpecPath, err)
	}
	for name := range doc.Components.Schemas.CreateProposalRequest.Properties.Proposal.Properties {
		proposalProps = append(proposalProps, name)
	}
	if len(proposalProps) == 0 {
		return nil, nil, fmt.Errorf("%s: CreateProposalRequest.properties.proposal has no properties — the record's premise anchor does not resolve", VendoredSpecPath)
	}
	for name := range doc.Components.Schemas.Role.Properties {
		roleProps = append(roleProps, name)
	}
	if len(roleProps) == 0 {
		return nil, nil, fmt.Errorf("%s: the Role schema has no properties — the record's classification anchors do not resolve", VendoredSpecPath)
	}
	return sortedStrings(proposalProps), sortedStrings(roleProps), nil
}

// --- The guard: the record checked against itself and the contract -------

// CheckCircleRoutingRule returns every violation of the record's invariants,
// each message naming the invariant, the offending element, AND which
// resolution path applies (interface-spec § Error Communication, conditions
// 1–6 and 8–9). An empty result means the record agrees with itself, resolves
// on the shipped CLI, and still rests on a contract that carries its premise
// and anchors. Every side is derived at test time — the record's sections,
// fields, named reads, and cited anchors from the record; the property sets
// from the vendored spec; the live surfaces from the CLI sources. The guard
// hard-codes no read names, no property sets, and no schema field values
// (plan ADR-4).
//
// Condition 7 — every named read present in the drafting path's composed-leaf
// registry — will be added when the composed surface widens (073 phase 2);
// its reverse direction will stay unchecked by design (see the residues).
//
// EXPLICITLY PARTIAL (stated, not silent), three residues:
//  1. Semantic drift is undetectable — the server could change where a
//     proposal lands while the request schema stays byte-identical; the
//     empirical marker is what tells a consumer the landing behaviour is
//     observed rather than contracted.
//  2. The reverse direction of the record↔registry agreement is unchecked by
//     design — a registry leaf the record does not name is legitimate (the
//     registry carries the drafting path's other composed leaves); asserting
//     set-equality would invent a routing-read delimiter in the registry and
//     make the guard a second source of truth for the procedure's reads.
//  3. Flags are unguarded — `me roles`'s pagination limitation, which the
//     Uncertainty field's hedge rests on, has no machine anchor; the hedge is
//     safe under either resolution.
//
// The check never stops at the first failure: it collects every violation it
// can still evaluate, so one broken invariant does not mask the rest.
func CheckCircleRoutingRule(rec CircleRoutingRuleRecord, proposalProps, roleProps, liveTop, liveMe, liveTension []string) []string {
	var v []string

	// Condition 1: every required section present.
	sections := []string{
		routingSectionCitations,
		routingSectionRule,
		routingSectionClassification,
		routingSectionProcedure,
	}
	present := map[string]bool{}
	for _, name := range sections {
		if strings.TrimSpace(routingSectionBody(rec.Raw, name)) == "" {
			v = append(v, fmt.Sprintf("the record is missing its required %q section — add the section; the record is incomplete, not merely terse", name))
		} else {
			present[name] = true
		}
	}

	// Condition 2: every required field label present with a non-empty value.
	// Fields are only checked inside sections that exist — a missing section
	// already failed condition 1, and repeating it per field would bury it.
	fieldChecks := []struct {
		surface string
		fields  map[string]string
		labels  []string
		gate    bool
	}{
		{"document header", rec.HeaderFields, routingHeaderRequiredFields, true},
		{routingSectionCitations + " section", rec.CitationFields, routingCitationRequiredFields, present[routingSectionCitations]},
		{routingSectionRule + " section", rec.RuleFields, routingRuleRequiredFields, present[routingSectionRule]},
		{routingSectionClassification + " section", rec.ClassificationFields, routingClassificationRequiredFields, present[routingSectionClassification]},
		{routingSectionProcedure + " section", rec.ProcedureFields, routingProcedureRequiredFields, present[routingSectionProcedure]},
	}
	for _, fc := range fieldChecks {
		if !fc.gate {
			continue
		}
		for _, label := range fc.labels {
			if strings.TrimSpace(fc.fields[label]) == "" {
				v = append(v, fmt.Sprintf("the %s is missing required field %q or its value is empty — supply the field", fc.surface, label))
			}
		}
	}

	// Condition 3: the empirical marker present and carrying the split.
	if !RoutingMarkerIsWellFormed(rec.Marker) {
		v = append(v, "the empirical marker is absent or missing its required phrase — restore the leading blockquote carrying \"Empirical record\" and the cite-versus-observe split: the absent circle parameter is published contract, while where the proposal lands is observed behaviour")
	}

	// Conditions 4–6: the named reads declared, resolving, and anchorable.
	v = append(v, checkRoutingNamedReads(rec.NamedReads, liveTop, liveMe, liveTension)...)

	// Condition 8: the premise tripwire — the create request's whole proposal
	// property set must equal the set the record's Premise citation claims.
	// Deliberately a set-equality rather than a search for circle-like key
	// names: a circle parameter could arrive under any spelling, and a
	// name-matching check would miss exactly the change that matters.
	if !stringSetsEqual(rec.CitedPremiseProps, proposalProps) {
		v = append(v, fmt.Sprintf("CreateProposalRequest.properties.proposal's property set %v is not exactly the record's cited premise set %v — an added parameter dissolves the rule's premise; re-derive the rule against the new parameter, or retire the record", sortedStrings(proposalProps), sortedStrings(rec.CitedPremiseProps)))
	}

	// Condition 9: every cited Role field still on the Role schema.
	roleSet := toStringSet(roleProps)
	for _, cite := range rec.RoleCitations {
		if !roleSet[cite.Field] {
			v = append(v, fmt.Sprintf("the record cites Role.%s in its %s section but the Role schema no longer carries that field — re-derive the citation against the new anchor, or retire the record; a contract that cannot distinguish a circle from a role cannot support the classification test", cite.Field, cite.Section))
		}
	}

	return v
}

// checkRoutingNamedReads resolves every read the record's named-reads block
// declares against the shipped CLI's live surfaces, with a four-way shape:
// top-level for single-token leaves, `me <sub>` and `tension <sub>` for the
// two-token groups the guard anchors, and an unanchorable-default arm that
// reports rather than skips (no silent caps). An absent or empty block is
// condition 4; an unresolved read is condition 5; an unanchorable path is
// condition 6.
func checkRoutingNamedReads(reads, liveTop, liveMe, liveTension []string) []string {
	if len(reads) == 0 {
		return []string{"the record declares no reads — its named-reads block is absent or empty; declare the reads the procedure names, because a procedure that names none is not a procedure"}
	}
	topSet := toStringSet(liveTop)
	meSet := toStringSet(liveMe)
	tensionSet := toStringSet(liveTension)
	var v []string
	for _, read := range reads {
		switch parts := strings.Fields(read); {
		case len(parts) == 1:
			if !topSet[parts[0]] {
				v = append(v, fmt.Sprintf("named read %q does not resolve on the shipped CLI (surface searched: top-level) — fix the record, or restore the command", read))
			}
		case len(parts) == 2 && parts[0] == "me":
			if !meSet[parts[1]] {
				v = append(v, fmt.Sprintf("named read %q does not resolve on the shipped CLI (surface searched: me) — fix the record, or restore the command", read))
			}
		case len(parts) == 2 && parts[0] == "tension":
			if !tensionSet[parts[1]] {
				v = append(v, fmt.Sprintf("named read %q does not resolve on the shipped CLI (surface searched: tension) — fix the record, or restore the command", read))
			}
		default:
			v = append(v, fmt.Sprintf("named read %q carries a command path the guard cannot anchor (supported forms: a top-level command, `me <sub>`, `tension <sub>`) — extend the guard or fix the record, never skipped silently", read))
		}
	}
	return v
}
