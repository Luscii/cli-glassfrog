// Package grammar carries the change-set grammar reference the CLI serves to an
// assembler before it builds a proposal's changes[] array (077): the
// contract-enumerated change types with their placement rules, and the empirical
// residue the published contract does not carry, every shape marked by
// provenance.
//
// The knowledge has two sources, both of which live in the repository and
// NEITHER of which ships with the binary: the vendored Glassfrog API v5 contract
// (spec/glassfrog-api-v5.yaml) and the change-set grammar record owned by the
// proposal-drafting skill (plugin/skills/proposal-drafting/references/). A
// dev-time generator derives one structured artifact from both and commits it
// here; this package embeds it, so the reference is served by the binary itself
// on any machine, before credentials and with no repository access (plan ADR-1,
// ADR-3).
//
// The package is deliberately a leaf: it decodes one embedded JSON document and
// hands the payload out. It parses no YAML and no markdown — a record-format
// quirk is a repository-side generation failure, never a runtime failure on an
// operator's machine. It imports nothing beyond the stdlib, so both the shipped
// binary and the dev-time generator/guard can depend on it without a cycle.
package grammar

// The committed artifact is machine-produced from the vendored contract and the
// grammar record — never hand-edited. This is the regeneration step every guard
// failure and the artifact's own marker name.
//
//go:generate go run ./gen

// The two provenance tokens every rendered shape carries (interface-cli §
// "Provenance tokens"). They are compared literally by agents, so they are
// constants here — the one place that spells them — rather than string literals
// at each construction site.
const (
	// ProvenancePublishedContract marks a shape the published v5 contract
	// enumerates: the change-type vocabulary and the nested-only rule.
	ProvenancePublishedContract = "published-contract"
	// ProvenanceEmpiricalObservation marks a shape verified against live server
	// behavior and recorded in the grammar record — true when observed, and
	// subject to change when the server changes.
	ProvenanceEmpiricalObservation = "empirical-observation"
)

// The two placement classes a change type can carry (interface-cli §
// "change_types[] entry").
const (
	// PlacementTopLevel means the type may sit directly in the changes[] array.
	PlacementTopLevel = "top-level"
	// PlacementNestedOnly means the type must ride inside a wrapper part; the
	// entry's Wrappers field names which.
	PlacementNestedOnly = "nested-only"
)

// Artifact is the committed generated document: the rendered payload wrapped in
// a maintenance envelope (interface-cli § "The embedded artifact"). Generated
// carries the do-not-edit marker naming the regeneration step; it is repo-facing
// maintenance metadata and never reaches command output — the accessor hands out
// Grammar and nothing else.
type Artifact struct {
	Generated string  `json:"generated"`
	Grammar   Grammar `json:"grammar"`
}

// Grammar is the rendered structure — the ONE definition that does triple duty
// (interface-cli § "One schema, three consumers"): the generator writes it, the
// drift guard regenerates and byte-compares it, and the command serializes it as
// its json/yaml output and renders it through the human templates.
//
// Both keys are always present. Facts is `[]` — never omitted, never null — when
// the record's residue is empty, so a consumer never has to distinguish "no
// facts" from "the key moved".
type Grammar struct {
	ChangeTypes []ChangeType `json:"change_types"`
	Facts       []Fact       `json:"facts"`
}

// ChangeType is one entry of the contract-enumerated change-type vocabulary.
// Wrappers is present ONLY when Placement is PlacementNestedOnly — it names the
// role-operation parts the type must ride inside — so a top-level entry carries
// no empty array to interpret.
type ChangeType struct {
	Type       string   `json:"type"`
	Placement  string   `json:"placement"`
	Wrappers   []string `json:"wrappers,omitempty"`
	Provenance string   `json:"provenance"`
}

// Fact is one entry of the empirical residue, projected from a fact section of
// the grammar record. Title/Shape/Disposition/Symptom are the record's own text,
// carried verbatim. The record's Evidence and lineage fields are excluded by
// design (interface-cli § "Excluded by design"): they are maintenance metadata,
// and the lineage text names development-repository history that is meaningless
// on an operator's machine.
type Fact struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Shape       string `json:"shape"`
	Disposition string `json:"disposition"`
	Symptom     string `json:"symptom"`
	Provenance  string `json:"provenance"`
}
