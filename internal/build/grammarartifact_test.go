package build

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/Luscii/cli-glassfrog/internal/grammar"
)

// The derivation half of the Agent-Facing Grammar Reference (077 T001). These
// tests exercise the pure projection functions over synthesized inputs — the
// guard tests (grammarartifact_guard_test.go) pin the committed artifact against
// the real sources.

// realEnum is a stand-in contract enum: two nested-only types, two top-level, and
// the two wrapper parts. Deliberately unsorted so the alphabetical-ordering
// contract is actually exercised.
var realEnum = []string{"UpdateRole", "CreatePolicy", "CreateAccountability", "CreateRole", "RemoveDomain"}

func TestGrammarWrapperTypesDerivesTheWrapperPairFromContractProse(t *testing.T) {
	description := "Nested-only types (`CreateAccountability`, `RemoveDomain`) must appear as children of `UpdateRole` or `CreateRole`, not as top-level proposal changes."
	got, err := grammarWrapperTypes(description, realEnum)
	if err != nil {
		t.Fatalf("deriving the wrapper pair: %v", err)
	}
	// Sorted, so a prose reword that names the pair in the other order does not
	// move the artifact's bytes.
	want := []string{"CreateRole", "UpdateRole"}
	if len(got) != len(want) {
		t.Fatalf("derived wrappers %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("derived wrappers %v, want %v", got, want)
		}
	}
}

func TestGrammarWrapperTypesFailsLoudWhenTheProseChanges(t *testing.T) {
	cases := map[string]struct {
		description string
		wantIn      string
	}{
		"clause reworded away": {
			description: "Nested-only types (`CreateAccountability`) must be nested inside a role operation.",
			wantIn:      "could not locate the nested-only wrapper clause",
		},
		"clause names no type": {
			description: "Nested-only types (`CreateAccountability`) must appear as children of a role operation, not at top level.",
			wantIn:      "names no change type",
		},
		"clause names a type absent from the enum": {
			description: "Nested-only types (`CreateAccountability`) must appear as children of `ReviseRole`, not as top-level proposal changes.",
			wantIn:      "absent from ProposalChange.properties.type.enum",
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := grammarWrapperTypes(tc.description, realEnum)
			if err == nil {
				t.Fatalf("expected the derivation to fail; got no error")
			}
			if !strings.Contains(err.Error(), tc.wantIn) {
				t.Fatalf("error %q does not name %q", err, tc.wantIn)
			}
			// Every derivation failure must point at the code to re-derive in.
			if !strings.Contains(err.Error(), grammarArtifactSource) {
				t.Fatalf("error %q does not name the derivation site %q", err, grammarArtifactSource)
			}
		})
	}
}

func TestGrammarChangeTypesCarriesWrappersOnNestedOnlyEntriesOnly(t *testing.T) {
	entries, err := grammarChangeTypes(realEnum, []string{"CreateAccountability", "RemoveDomain"}, []string{"CreateRole", "UpdateRole"})
	if err != nil {
		t.Fatalf("projecting the vocabulary: %v", err)
	}
	if len(entries) != len(realEnum) {
		t.Fatalf("projected %d entries, want one per enum member (%d)", len(entries), len(realEnum))
	}
	// Alphabetical by type.
	for i := 1; i < len(entries); i++ {
		if entries[i-1].Type >= entries[i].Type {
			t.Fatalf("entries are not sorted alphabetically: %q then %q", entries[i-1].Type, entries[i].Type)
		}
	}
	for _, e := range entries {
		if e.Provenance != grammar.ProvenancePublishedContract {
			t.Fatalf("entry %q carries provenance %q, want %q", e.Type, e.Provenance, grammar.ProvenancePublishedContract)
		}
		nested := e.Type == "CreateAccountability" || e.Type == "RemoveDomain"
		switch {
		case nested && e.Placement != grammar.PlacementNestedOnly:
			t.Fatalf("entry %q has placement %q, want %q", e.Type, e.Placement, grammar.PlacementNestedOnly)
		case !nested && e.Placement != grammar.PlacementTopLevel:
			t.Fatalf("entry %q has placement %q, want %q", e.Type, e.Placement, grammar.PlacementTopLevel)
		}
		if nested && len(e.Wrappers) == 0 {
			t.Fatalf("nested-only entry %q carries no wrappers", e.Type)
		}
		if !nested && e.Wrappers != nil {
			t.Fatalf("top-level entry %q carries wrappers %v — the field must be absent unless nested-only", e.Type, e.Wrappers)
		}
	}
}

func TestGrammarChangeTypesRejectsANestedOnlyTypeAbsentFromTheEnum(t *testing.T) {
	_, err := grammarChangeTypes(realEnum, []string{"ReviseDomain"}, []string{"CreateRole"})
	if err == nil {
		t.Fatal("expected the projection to fail on a nested-only type outside the enum")
	}
	if !strings.Contains(err.Error(), "ReviseDomain") || !strings.Contains(err.Error(), "disagree") {
		t.Fatalf("error %q does not name the offending type and the disagreement", err)
	}
}

// factRecord builds a parsed record from a manifest and matching sections, so the
// projection can be exercised without writing a markdown fixture.
func factRecord(manifest []string, facts ...GrammarFact) GrammarFactsRecord {
	return GrammarFactsRecord{ManifestIDs: manifest, Facts: facts}
}

func fixtureFact(id string) GrammarFact {
	return GrammarFact{
		ID:    id,
		Title: id + " title",
		Fields: map[string]string{
			"Shape":       id + " shape",
			"Disposition": "accepted.",
			"Symptom":     id + " symptom",
			"Evidence":    "prp_deadbeef",
			"Provenance":  "supersedes a provisional note",
		},
	}
}

func TestGrammarFactEntriesFollowTheManifestOrderAndDropMaintenanceFields(t *testing.T) {
	// Sections in the opposite order from the manifest: the manifest is the
	// ordering source, not section order.
	rec := factRecord([]string{"CSG-1", "CSG-2"}, fixtureFact("CSG-2"), fixtureFact("CSG-1"))
	entries, err := grammarFactEntries(rec)
	if err != nil {
		t.Fatalf("projecting the residue: %v", err)
	}
	if len(entries) != 2 || entries[0].ID != "CSG-1" || entries[1].ID != "CSG-2" {
		t.Fatalf("entries do not follow the manifest order: %+v", entries)
	}
	for _, e := range entries {
		if e.Provenance != grammar.ProvenanceEmpiricalObservation {
			t.Fatalf("fact %s carries provenance %q, want %q", e.ID, e.Provenance, grammar.ProvenanceEmpiricalObservation)
		}
		if e.Disposition != "accepted" {
			t.Fatalf("fact %s disposition %q is not normalized off the record's sentence punctuation", e.ID, e.Disposition)
		}
	}
	// The record's Evidence and lineage are maintenance metadata: the projected
	// struct has no field to carry them, and the marshalled entry names neither.
	doc, err := json.Marshal(entries)
	if err != nil {
		t.Fatalf("marshalling entries: %v", err)
	}
	for _, banned := range []string{"evidence", "prp_deadbeef", "supersedes"} {
		if strings.Contains(strings.ToLower(string(doc)), banned) {
			t.Fatalf("projected entries leak the maintenance field %q: %s", banned, doc)
		}
	}
}

func TestGrammarFactEntriesEmptyManifestKeepsTheKeyPresent(t *testing.T) {
	entries, err := grammarFactEntries(factRecord(nil))
	if err != nil {
		t.Fatalf("projecting an empty residue: %v", err)
	}
	if entries == nil {
		t.Fatal("an empty manifest must yield a non-nil empty slice, so the key marshals as [] rather than null")
	}
	doc, err := MarshalGrammarArtifact(grammar.Artifact{Grammar: grammar.Grammar{ChangeTypes: []grammar.ChangeType{}, Facts: entries}})
	if err != nil {
		t.Fatalf("marshalling: %v", err)
	}
	if !strings.Contains(string(doc), `"facts": []`) {
		t.Fatalf("an empty residue must marshal as \"facts\": [] with the key present; got %s", doc)
	}
}

func TestGrammarFactEntriesRejectAManifestIDWithNoSection(t *testing.T) {
	rec := factRecord([]string{"CSG-1", "CSG-9"}, fixtureFact("CSG-1"))
	_, err := grammarFactEntries(rec)
	if err == nil {
		t.Fatal("expected the projection to fail on a manifest id with no section")
	}
	if !strings.Contains(err.Error(), "CSG-9") || !strings.Contains(err.Error(), GrammarRegenerationStep) {
		t.Fatalf("error %q does not name the missing fact and the regeneration step", err)
	}
}

func TestMarshalGrammarArtifactIsCanonicalAndNewlineTerminated(t *testing.T) {
	doc, err := MarshalGrammarArtifact(grammar.Artifact{
		Generated: "marker",
		Grammar:   grammar.Grammar{ChangeTypes: []grammar.ChangeType{}, Facts: []grammar.Fact{}},
	})
	if err != nil {
		t.Fatalf("marshalling: %v", err)
	}
	if !strings.HasSuffix(string(doc), "}\n") {
		t.Fatalf("the canonical encoding must end in a single trailing newline; got %q", string(doc))
	}
	if !strings.Contains(string(doc), "\n  \"grammar\": {") {
		t.Fatalf("the canonical encoding must be two-space indented; got %s", doc)
	}
}

// TestRenderGrammarArtifactIsByteDeterministic pins the interface's determinism
// contract at the source: the same two files always produce the same bytes, which
// is what makes the drift guard's byte-comparison a usable signal.
func TestRenderGrammarArtifactIsByteDeterministic(t *testing.T) {
	first, err := RenderGrammarArtifact()
	if err != nil {
		t.Fatalf("first derivation: %v", err)
	}
	second, err := RenderGrammarArtifact()
	if err != nil {
		t.Fatalf("second derivation: %v", err)
	}
	if string(first) != string(second) {
		t.Fatal("two derivations from the same sources produced different bytes")
	}
}
