package build

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"sigs.k8s.io/yaml"
)

// GrammarFactsPath is the change-set grammar record, relative to the repository
// root — a hand-authored knowledge artifact owned by the proposal-drafting
// skill (072, plan ADR-1), carrying the two empirical change-set shapes the
// published v5 contract does not (CSG-1, CSG-2). It is the shared constant the
// guard reads and future consumers (#75/#77/#83) grep; single-sourced here
// alongside the family's other path constants.
const GrammarFactsPath = "plugin/skills/proposal-drafting/references/change-set-grammar-facts.md"

// GrammarDispositionVocabulary is the closed set of Disposition values a fact
// may carry (interface-spec § per-fact contract). A value outside this set is a
// guard violation (condition 5). Recorded here as a checked-in contract fact —
// it is the closed vocabulary the spec pins, not a value derived from another
// file, so pinning it is correct rather than a second source of truth.
var GrammarDispositionVocabulary = []string{"accepted", "rejected", "accepted-but-invalid"}

// GrammarRequiredFields is the five labelled fields every fact section must
// carry, non-empty (interface-spec § per-fact contract). The labels are the
// machine anchors; a missing or empty one is guard condition 4.
var GrammarRequiredFields = []string{"Shape", "Disposition", "Symptom", "Evidence", "Provenance"}

// GrammarFact is one recorded fact parsed from the record: its id and title
// (from the `## CSG-<n> — <title>` heading), the five labelled fields keyed by
// label, and the change-type names its Shape field mentions (for the
// citation-integrity check against the spec enum).
type GrammarFact struct {
	ID              string
	Title           string
	Fields          map[string]string
	ShapeCitedTypes []string
}

// GrammarFactsRecord is the parsed record: the leading empirical marker text,
// the `Live facts` manifest ids, the nested-only citation list, and the fact
// sections. Every side the guard needs is derived here from the file; the guard
// hard-codes no fact ids, enum values, or type names (plan ADR-3).
type GrammarFactsRecord struct {
	Raw         string
	Marker      string
	ManifestIDs []string
	NestedOnly  []string
	Facts       []GrammarFact
}

// ReadGrammarFactsRecord reads the record from the repository root. A missing
// or unreadable file is an error.
func ReadGrammarFactsRecord() (string, error) {
	raw, err := readRepoFile(GrammarFactsPath)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

// ParseGrammarFactsRecord derives every record-side fact the guard and the
// content scenarios need from the raw markdown. It never reads the spec — the
// spec side is derived separately (SpecChangeTypeEnum / SpecNestedOnlyTypes) so
// the two sources stay independent.
func ParseGrammarFactsRecord(raw string) GrammarFactsRecord {
	return GrammarFactsRecord{
		Raw:         raw,
		Marker:      parseLeadingMarker(raw),
		ManifestIDs: parseLiveFactsManifest(raw),
		NestedOnly:  parseNestedOnlyCitation(raw),
		Facts:       parseGrammarFactSections(raw),
	}
}

// parseLeadingMarker returns the text of the leading blockquote — the
// consecutive `>` lines at the top of the file, before any `#` heading. Empty
// if the file opens with a heading (no marker) rather than a blockquote.
func parseLeadingMarker(raw string) string {
	var lines []string
	for _, line := range strings.Split(raw, "\n") {
		t := strings.TrimSpace(line)
		if t == "" && len(lines) == 0 {
			continue // skip leading blank lines
		}
		if strings.HasPrefix(t, "#") {
			break // a heading before any blockquote: no leading marker
		}
		if strings.HasPrefix(t, ">") {
			lines = append(lines, strings.TrimSpace(strings.TrimPrefix(t, ">")))
			continue
		}
		if len(lines) > 0 {
			break // blockquote ended
		}
	}
	return strings.Join(lines, " ")
}

// MarkerIsWellFormed reports whether the leading marker carries the required
// empirical phrase and states both halves of its claim: every fact is observed
// behavior, and none is part of the published contract (interface-spec anatomy
// row 1 / condition 8). A marker that is absent or degraded on any of these is
// not well-formed.
func MarkerIsWellFormed(marker string) bool {
	if !strings.Contains(marker, "Empirical record") {
		return false
	}
	low := strings.ToLower(marker)
	statesObserved := strings.Contains(low, "observed")
	// The disclaimer half: names the published contract and negates membership,
	// accepting the natural phrasings ("none of it is part of the … contract",
	// "is not part of the published contract").
	negated := strings.Contains(low, "none") || strings.Contains(low, "not part") || strings.Contains(low, "not published")
	statesNotContract := strings.Contains(low, "contract") && negated
	return statesObserved && statesNotContract
}

// liveFactsRE matches the `Live facts` manifest header line, capturing the
// comma-separated id list.
var liveFactsRE = regexp.MustCompile(`(?m)^\s*[-*]?\s*\*\*Live facts\*\*:\s*(.+)$`)

// parseLiveFactsManifest returns the fact ids declared on the `Live facts`
// header line, in declaration order. Empty if the line is absent or declares no
// ids (an empty manifest — a valid parse of an empty-shell record the guard
// then rejects).
func parseLiveFactsManifest(raw string) []string {
	m := liveFactsRE.FindStringSubmatch(raw)
	if m == nil {
		return nil
	}
	return splitIDList(m[1])
}

// nestedOnlyRE captures the type list that follows the explicit
// "Nested-only types:" label, up to the sentence-terminating period. The label
// is the parse anchor: `CreateRole`/`UpdateRole` appear earlier in the same
// bullet as the wrapper parts, so only the labelled list is the nested-only
// set.
var nestedOnlyRE = regexp.MustCompile(`Nested-only types:\s*([^.]+)\.`)

// parseNestedOnlyCitation returns the nested-only change-type names the record
// cites, from the labelled list in the contract-citations section. This is the
// set the guard compares against the spec's nested-only set (condition 7).
func parseNestedOnlyCitation(raw string) []string {
	m := nestedOnlyRE.FindStringSubmatch(raw)
	if m == nil {
		return nil
	}
	return backtickedCamelTokens(m[1])
}

// factHeadingRE matches a fact section heading `## CSG-<n> — <title>`, capturing
// the id and the title. The em dash separates id from title; a plain hyphen is
// accepted too so an authoring slip degrades to a parse, not a crash.
var factHeadingRE = regexp.MustCompile(`(?m)^##\s+(CSG-\d+)\s*[—-]\s*(.+?)\s*$`)

// parseGrammarFactSections parses each `## CSG-<n> — <title>` section into a
// GrammarFact, reading its labelled fields and the change-type names its Shape
// mentions. Sections run from one fact heading to the next fact heading (or EOF).
func parseGrammarFactSections(raw string) []GrammarFact {
	locs := factHeadingRE.FindAllStringSubmatchIndex(raw, -1)
	var facts []GrammarFact
	for i, loc := range locs {
		id := raw[loc[2]:loc[3]]
		title := raw[loc[4]:loc[5]]
		bodyStart := loc[1]
		bodyEnd := len(raw)
		if i+1 < len(locs) {
			bodyEnd = locs[i+1][0]
		}
		body := raw[bodyStart:bodyEnd]
		fields := parseLabelledFields(body)
		facts = append(facts, GrammarFact{
			ID:              id,
			Title:           strings.TrimSpace(title),
			Fields:          fields,
			ShapeCitedTypes: backtickedCamelTokens(fields["Shape"]),
		})
	}
	return facts
}

// labelledFieldRE matches the start of a `- **Label**: value` list item,
// capturing the label and the first line of its value.
var labelledFieldRE = regexp.MustCompile(`^\s*[-*]\s*\*\*([^*]+)\*\*:\s*(.*)$`)

// parseLabelledFields reads the bold-labelled list items in a fact section into
// a label→value map. A field value runs from its label line through any
// continuation lines (a wrapped list item indents its continuation) up to the
// next labelled item, heading, or blank line — so a multi-line Shape or Symptom
// is captured whole, not truncated to its first line.
func parseLabelledFields(body string) map[string]string {
	fields := map[string]string{}
	var curLabel string
	var curValue []string
	flush := func() {
		if curLabel != "" {
			fields[curLabel] = strings.TrimSpace(strings.Join(curValue, " "))
		}
		curLabel, curValue = "", nil
	}
	for _, line := range strings.Split(body, "\n") {
		if m := labelledFieldRE.FindStringSubmatch(line); m != nil {
			flush()
			curLabel = strings.TrimSpace(m[1])
			curValue = []string{strings.TrimSpace(m[2])}
			continue
		}
		if strings.HasPrefix(strings.TrimSpace(line), "#") {
			break // a heading ends the section's fields
		}
		if strings.TrimSpace(line) == "" {
			flush() // a blank line ends the current field
			continue
		}
		if curLabel != "" {
			curValue = append(curValue, strings.TrimSpace(line))
		}
	}
	flush()
	return fields
}

// lineCitationRE matches a line-number citation form the record must never use
// — a `<file>.yaml:NNN` suffix or the words "line NNN" — because line numbers
// shift on every spec refresh (interface-spec: cite by name, never by line).
var lineCitationRE = regexp.MustCompile(`(?i)\.yaml:\d+|\bline\s+\d+`)

// camelTokenRE matches a backticked UpperCamelCase identifier — the shape of a
// governance change-type name (`CreatePolicy`, `UpdateRole`). It deliberately
// requires the whole backticked span to be the identifier, so a dotted anchor
// like `ProposalChange.properties.type.enum` does not match (it is a citation
// path, not a change type).
var camelTokenRE = regexp.MustCompile("`([A-Z][A-Za-z]+)`")

// backtickedCamelTokens extracts the distinct backticked UpperCamelCase tokens
// from a string, in first-seen order.
func backtickedCamelTokens(s string) []string {
	var out []string
	seen := map[string]bool{}
	for _, m := range camelTokenRE.FindAllStringSubmatch(s, -1) {
		if !seen[m[1]] {
			seen[m[1]] = true
			out = append(out, m[1])
		}
	}
	return out
}

// Disposition returns the fact's Disposition value normalized for comparison
// against the closed vocabulary: trailing sentence punctuation and surrounding
// whitespace stripped, so `accepted-but-invalid.` reads as `accepted-but-invalid`.
func (f GrammarFact) Disposition() string {
	return strings.TrimRight(strings.TrimSpace(f.Fields["Disposition"]), ".")
}

// sectionRE matches a `## ` heading line, capturing the heading text.
var sectionRE = regexp.MustCompile(`(?m)^##\s+(.+?)\s*$`)

// ContractCitationsSection returns the body of the `## Contract citations`
// section (heading excluded), up to the next `## ` heading or EOF. Empty if the
// section is absent. Used to bound where the record legitimately names
// change-type tokens as citations rather than restatements.
func ContractCitationsSection(raw string) string {
	locs := sectionRE.FindAllStringSubmatchIndex(raw, -1)
	for i, loc := range locs {
		heading := strings.TrimSpace(raw[loc[2]:loc[3]])
		if !strings.EqualFold(heading, "Contract citations") {
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

// RestatedTypesBeyondCitations returns any backticked change-type token in the
// record whose value is not accounted for by a legitimate citation source —
// the contract-citations section (where naming a type IS a citation) or a
// fact's own Shape. A pasted enum brings in types (e.g. `RemoveRole`,
// `ElectRoleFiller`) absent from both, which surface here as restatements. The
// check is self-contained: it derives the allowed set from the record's own
// citation surfaces, never from a hard-coded enum.
func RestatedTypesBeyondCitations(rec GrammarFactsRecord) []string {
	allowed := map[string]bool{}
	for _, t := range backtickedCamelTokens(ContractCitationsSection(rec.Raw)) {
		allowed[t] = true
	}
	for _, f := range rec.Facts {
		for _, t := range f.ShapeCitedTypes {
			allowed[t] = true
		}
	}
	var extra []string
	for _, t := range backtickedCamelTokens(rec.Raw) {
		if !allowed[t] {
			extra = append(extra, t)
		}
	}
	return extra
}

// splitIDList splits a comma-separated id list, trimming whitespace and any
// surrounding backticks; empty items are dropped.
func splitIDList(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ",") {
		id := strings.TrimSpace(part)
		id = strings.Trim(id, "`")
		id = strings.TrimSpace(id)
		if id != "" {
			out = append(out, id)
		}
	}
	return out
}

// --- The spec side: the published contract the record cites --------------

// VendoredSpecPath is the vendored Glassfrog API v5 contract, relative to the
// repository root — the source the record's citations are checked against. Not
// restated anywhere; the enum and the nested-only set are derived from it at
// test time.
const VendoredSpecPath = "spec/glassfrog-api-v5.yaml"

// proposalChangeSchema is the narrow view of the ProposalChange schema the guard
// needs: the change-type enum and the free-text description carrying the
// nested-only rule. sigs.k8s.io/yaml converts YAML→JSON, so the json tags drive
// the decode; every other field in the large spec is ignored.
type proposalChangeSchema struct {
	Components struct {
		Schemas struct {
			ProposalChange struct {
				Description string `json:"description"`
				Properties  struct {
					Type struct {
						Enum []string `json:"enum"`
					} `json:"type"`
				} `json:"properties"`
			} `json:"ProposalChange"`
		} `json:"schemas"`
	} `json:"components"`
}

// LoadSpecChangeTypes reads the vendored spec from the repository root and
// returns the two spec-side sets the guard compares the record against: the
// full change-type enum at ProposalChange.properties.type.enum, and the
// nested-only type set named in the ProposalChange schema description. A missing
// or unparseable spec is an error; an empty enum is too (the anchor the record
// cites must resolve to something).
func LoadSpecChangeTypes() (enum []string, nestedOnly []string, err error) {
	raw, err := readRepoFile(VendoredSpecPath)
	if err != nil {
		return nil, nil, err
	}
	var doc proposalChangeSchema
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return nil, nil, fmt.Errorf("parsing %s: %w", VendoredSpecPath, err)
	}
	enum = doc.Components.Schemas.ProposalChange.Properties.Type.Enum
	if len(enum) == 0 {
		return nil, nil, fmt.Errorf("%s: ProposalChange.properties.type.enum is empty or absent — the record's enum citation anchor does not resolve", VendoredSpecPath)
	}
	nestedOnly = specNestedOnlyTypes(doc.Components.Schemas.ProposalChange.Description)
	return enum, nestedOnly, nil
}

// specNestedOnlyRE captures the parenthesized nested-only type list from the
// ProposalChange description sentence ("Nested-only types (`CreateAccountability`,
// …) must appear as children of `CreateRole` or `UpdateRole`…"). The parens
// bound the set: `CreateRole`/`UpdateRole` are named after them as the wrapper
// parts, so a whole-description scan would wrongly fold them in.
var specNestedOnlyRE = regexp.MustCompile(`Nested-only types\s*\(([^)]*)\)`)

// specNestedOnlyTypes extracts the nested-only change-type names from the
// ProposalChange schema description.
func specNestedOnlyTypes(description string) []string {
	m := specNestedOnlyRE.FindStringSubmatch(description)
	if m == nil {
		return nil
	}
	return backtickedCamelTokens(m[1])
}

// --- The guard: the record checked against itself and the contract -------

// CheckGrammarFacts returns every citation-integrity and internal-consistency
// violation in the record, each message naming the invariant, the offending
// element, AND which resolution path applies (interface-spec § Error
// Communication, conditions 1–8). An empty result means the record agrees with
// itself and with the contract it cites. The marker (condition 8) is checked
// against the parsed record; every other side is derived — the guard hard-codes
// no fact ids, enum values, or type names (plan ADR-3).
//
// The check never stops at the first failure: it collects every violation it
// can still evaluate, so one broken invariant does not mask the rest.
func CheckGrammarFacts(rec GrammarFactsRecord, specEnum, specNestedOnly []string) []string {
	var v []string

	factIDs := map[string]bool{}
	for _, f := range rec.Facts {
		factIDs[f.ID] = true
	}
	manifestIDs := map[string]bool{}
	for _, id := range rec.ManifestIDs {
		manifestIDs[id] = true
	}

	// Condition 3: an empty record is not a valid state (checked first so the
	// empty-shell message leads, but the other checks still run).
	if len(rec.Facts) == 0 {
		v = append(v, "the record has no fact sections — an empty record is not a valid state; delete the record and record the supersession rather than keeping a shell")
	}

	// Condition 1: manifest declares an id with no matching section.
	for _, id := range rec.ManifestIDs {
		if !factIDs[id] {
			v = append(v, fmt.Sprintf("the Live-facts manifest declares %q but no ## %s section exists — complete the retirement (drop the id), or restore the section if the deletion was unintended", id, id))
		}
	}

	// Condition 2: a fact section's id is absent from the manifest.
	for _, f := range rec.Facts {
		if !manifestIDs[f.ID] {
			v = append(v, fmt.Sprintf("fact section %q is absent from the Live-facts manifest — add the id to the manifest, or finish the deletion if the fact was meant to retire", f.ID))
		}
	}

	// Condition 4/5: per-fact required fields and closed disposition vocabulary.
	for _, f := range rec.Facts {
		for _, label := range GrammarRequiredFields {
			if strings.TrimSpace(f.Fields[label]) == "" {
				v = append(v, fmt.Sprintf("fact %s is missing or has an empty required field %q — supply the field", f.ID, label))
			}
		}
		if disp := f.Disposition(); disp != "" && !inStringSet(disp, GrammarDispositionVocabulary) {
			v = append(v, fmt.Sprintf("fact %s carries Disposition %q outside the closed vocabulary — use one of %s", f.ID, disp, strings.Join(GrammarDispositionVocabulary, " / ")))
		}
	}

	// Condition 6: a change type the record names in a fact Shape must exist in
	// the spec's enum.
	enumSet := toStringSet(specEnum)
	for _, f := range rec.Facts {
		for _, t := range f.ShapeCitedTypes {
			if !enumSet[t] {
				v = append(v, fmt.Sprintf("fact %s names change type %q, absent from the spec enum at ProposalChange.properties.type.enum — re-derive the citation, or retire the fact that names it", f.ID, t))
			}
		}
	}

	// Condition 7: the record's nested-only citation must set-equal the spec's.
	if !stringSetsEqual(rec.NestedOnly, specNestedOnly) {
		v = append(v, fmt.Sprintf("the record's nested-only citation %v is not set-equal to the spec's nested-only set %v (ProposalChange description) — re-derive the citation, or retire the fact it supports", sortedStrings(rec.NestedOnly), sortedStrings(specNestedOnly)))
	}

	// Condition 8: the empirical marker must be present and well-formed.
	if !MarkerIsWellFormed(rec.Marker) {
		v = append(v, "the empirical marker is absent or degraded — restore the leading blockquote stating that every fact is observed behavior and none is part of the published contract (must carry the phrase \"Empirical record\")")
	}

	return v
}

// inStringSet reports whether s is a member of set.
func inStringSet(s string, set []string) bool {
	for _, x := range set {
		if x == s {
			return true
		}
	}
	return false
}

// toStringSet builds a membership set from a slice.
func toStringSet(xs []string) map[string]bool {
	set := make(map[string]bool, len(xs))
	for _, x := range xs {
		set[x] = true
	}
	return set
}

// stringSetsEqual reports whether two slices carry the same set of values,
// order- and duplicate-insensitive.
func stringSetsEqual(a, b []string) bool {
	sa, sb := toStringSet(a), toStringSet(b)
	if len(sa) != len(sb) {
		return false
	}
	for k := range sa {
		if !sb[k] {
			return false
		}
	}
	return true
}

// sortedCopy returns a sorted copy for stable failure messages.
func sortedStrings(xs []string) []string {
	out := append([]string(nil), xs...)
	sort.Strings(out)
	return out
}
