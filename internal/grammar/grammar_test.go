package grammar

import (
	"encoding/json"
	"sort"
	"strings"
	"testing"
)

// The accessor half of the Agent-Facing Grammar Reference (077 T002), exercised
// against the committed artifact — the same bytes the shipped binary carries.

func TestLoadReturnsTheCommittedVocabularyAndResidue(t *testing.T) {
	g, err := Load()
	if err != nil {
		t.Fatalf("loading the embedded grammar: %v", err)
	}
	if len(g.ChangeTypes) == 0 {
		t.Fatal("the loaded grammar carries no change types")
	}
	if g.Facts == nil {
		t.Fatal("the loaded grammar's facts slice is nil — the key must always be an array")
	}
}

// TestLoadCarriesEveryFieldOnEveryEntry pins the accord's field tables: an entry
// missing a field would render a blank line rather than fail, so the completeness
// check belongs here rather than in the templates.
func TestLoadCarriesEveryFieldOnEveryEntry(t *testing.T) {
	g, err := Load()
	if err != nil {
		t.Fatalf("loading the embedded grammar: %v", err)
	}
	for _, ct := range g.ChangeTypes {
		if ct.Type == "" {
			t.Fatal("a change-type entry carries no type")
		}
		switch ct.Placement {
		case PlacementTopLevel:
			if len(ct.Wrappers) != 0 {
				t.Fatalf("top-level entry %q carries wrappers %v — the field is present only when nested-only", ct.Type, ct.Wrappers)
			}
		case PlacementNestedOnly:
			if len(ct.Wrappers) == 0 {
				t.Fatalf("nested-only entry %q names no wrapper parts", ct.Type)
			}
		default:
			t.Fatalf("entry %q carries placement %q, outside the closed vocabulary", ct.Type, ct.Placement)
		}
		if ct.Provenance != ProvenancePublishedContract {
			t.Fatalf("entry %q carries provenance %q, want %q", ct.Type, ct.Provenance, ProvenancePublishedContract)
		}
	}
	for _, f := range g.Facts {
		if f.ID == "" || f.Title == "" || f.Shape == "" || f.Disposition == "" || f.Symptom == "" {
			t.Fatalf("fact %+v is missing a field", f)
		}
		if f.Provenance != ProvenanceEmpiricalObservation {
			t.Fatalf("fact %s carries provenance %q, want %q", f.ID, f.Provenance, ProvenanceEmpiricalObservation)
		}
	}
}

// TestLoadOrdersChangeTypesAlphabetically pins half of the determinism contract
// (interface-cli § "Ordering is deterministic"); the facts' manifest order is
// pinned build-side, where the manifest is readable.
func TestLoadOrdersChangeTypesAlphabetically(t *testing.T) {
	g, err := Load()
	if err != nil {
		t.Fatalf("loading the embedded grammar: %v", err)
	}
	types := make([]string, len(g.ChangeTypes))
	for i, ct := range g.ChangeTypes {
		types[i] = ct.Type
	}
	sorted := append([]string(nil), types...)
	sort.Strings(sorted)
	for i := range types {
		if types[i] != sorted[i] {
			t.Fatalf("change types are not alphabetical: got %v", types)
		}
	}
}

// TestLoadNeverExposesTheMaintenanceEnvelope is the accord's repo-facing boundary:
// the marker names repository paths, so a path from the accessor to the marker
// would be a leaked repository pointer on an operator's machine.
func TestLoadNeverExposesTheMaintenanceEnvelope(t *testing.T) {
	g, err := Load()
	if err != nil {
		t.Fatalf("loading the embedded grammar: %v", err)
	}
	doc, err := json.Marshal(g)
	if err != nil {
		t.Fatalf("marshalling the payload: %v", err)
	}
	for _, banned := range []string{"generated", "DO NOT EDIT", "go generate", "internal/build", "spec/glassfrog-api-v5.yaml", "plugin/skills"} {
		if strings.Contains(string(doc), banned) {
			t.Fatalf("the payload leaks the maintenance envelope's %q: %s", banned, doc)
		}
	}
}

// TestLoadReturnsAnIndependentCopy guards the cached decode: a caller that
// reorders or clears what it got must not change what the next caller sees.
func TestLoadReturnsAnIndependentCopy(t *testing.T) {
	first, err := Load()
	if err != nil {
		t.Fatalf("loading: %v", err)
	}
	wantTypes := len(first.ChangeTypes)
	wantFirstType := first.ChangeTypes[0].Type
	first.ChangeTypes[0].Type = "mutated"
	first.ChangeTypes = first.ChangeTypes[:0]
	for i := range first.Facts {
		first.Facts[i].Symptom = "mutated"
	}

	second, err := Load()
	if err != nil {
		t.Fatalf("re-loading: %v", err)
	}
	if len(second.ChangeTypes) != wantTypes || second.ChangeTypes[0].Type != wantFirstType {
		t.Fatalf("a caller's mutation reached the cached decode: got %d entries starting %q", len(second.ChangeTypes), second.ChangeTypes[0].Type)
	}
	for _, f := range second.Facts {
		if f.Symptom == "mutated" {
			t.Fatalf("a caller's mutation reached the cached fact %s", f.ID)
		}
	}
}

// TestLoadIsRepeatable is the interface's byte-identical-output contract read at
// the accessor: the same binary always hands out the same document.
func TestLoadIsRepeatable(t *testing.T) {
	first, err := Load()
	if err != nil {
		t.Fatalf("loading: %v", err)
	}
	second, err := Load()
	if err != nil {
		t.Fatalf("re-loading: %v", err)
	}
	a, _ := json.Marshal(first)
	b, _ := json.Marshal(second)
	if string(a) != string(b) {
		t.Fatalf("two loads produced different documents:\n%s\n%s", a, b)
	}
}

// TestDecodeArtifactBytesFailsWithoutPanicking covers the exit-1 corrupt-build
// path the drift guard makes unshippable — no scenario can reach it (it would have
// to fabricate a corrupt binary), so this is where it is covered (tasks.md §
// Scenario Disposition).
func TestDecodeArtifactBytesFailsWithoutPanicking(t *testing.T) {
	cases := map[string]struct {
		raw    string
		wantIn string
	}{
		"truncated json":      {raw: `{"grammar": {"change_types": [`, wantIn: "could not be decoded"},
		"not an object":       {raw: `["change_types"]`, wantIn: "could not be decoded"},
		"empty document":      {raw: `{}`, wantIn: "carries no change types"},
		"empty vocabulary":    {raw: `{"grammar":{"change_types":[],"facts":[]}}`, wantIn: "carries no change types"},
		"wrong payload shape": {raw: `{"grammar":{"change_types":"CreateRole"}}`, wantIn: "could not be decoded"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got, err := decodeArtifactBytes([]byte(tc.raw))
			if err == nil {
				t.Fatalf("expected a decode failure; got %+v", got)
			}
			if !strings.Contains(err.Error(), tc.wantIn) {
				t.Fatalf("error %q does not name %q", err, tc.wantIn)
			}
			if got.ChangeTypes != nil || got.Facts != nil {
				t.Fatalf("a failed decode must yield the zero payload; got %+v", got)
			}
		})
	}
}

// TestDecodeArtifactBytesNormalizesAnAbsentResidue: a `facts` key that decoded to
// nil would serialize back out as null, breaking the always-an-array promise.
func TestDecodeArtifactBytesNormalizesAnAbsentResidue(t *testing.T) {
	for _, raw := range []string{
		`{"grammar":{"change_types":[{"type":"CreateRole","placement":"top-level","provenance":"published-contract"}]}}`,
		`{"grammar":{"change_types":[{"type":"CreateRole","placement":"top-level","provenance":"published-contract"}],"facts":null}}`,
	} {
		g, err := decodeArtifactBytes([]byte(raw))
		if err != nil {
			t.Fatalf("decoding %s: %v", raw, err)
		}
		if g.Facts == nil {
			t.Fatalf("facts stayed nil for %s", raw)
		}
		doc, _ := json.Marshal(g)
		if !strings.Contains(string(doc), `"facts":[]`) {
			t.Fatalf("an absent residue must serialize as []; got %s", doc)
		}
	}
}
