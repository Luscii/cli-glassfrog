package render

import (
	"strings"
	"testing"

	"github.com/Luscii/cli-glassfrog/internal/grammar"
)

// Golden tests for the change-set grammar's human formats (077 T004). They pin
// bytes, so a template edit that changes the shape of the reference an agent reads
// has to be a deliberate change to the expectation.

// grammarFixture is a small stand-in vocabulary and residue: two top-level types,
// one nested-only type, and two facts covering both ends of the disposition
// vocabulary. Small enough to pin byte-exactly, and it exercises every branch the
// real 21-type artifact does.
func grammarFixture() grammar.Grammar {
	return grammar.Grammar{
		ChangeTypes: []grammar.ChangeType{
			{Type: "CreateAccountability", Placement: grammar.PlacementNestedOnly, Wrappers: []string{"CreateRole", "UpdateRole"}, Provenance: grammar.ProvenancePublishedContract},
			{Type: "CreatePolicy", Placement: grammar.PlacementTopLevel, Provenance: grammar.ProvenancePublishedContract},
			{Type: "UpdateRole", Placement: grammar.PlacementTopLevel, Provenance: grammar.ProvenancePublishedContract},
		},
		Facts: []grammar.Fact{
			{ID: "CSG-1", Title: "An own-circle policy is top-level", Shape: "a top-level CreatePolicy part", Disposition: "accepted", Symptom: "a wrapped shape is refused", Provenance: grammar.ProvenanceEmpiricalObservation},
			{ID: "CSG-2", Title: "A self-targeting UpdateRole comes back invalid", Shape: "an UpdateRole targeting the circle itself", Disposition: "accepted-but-invalid", Symptom: "a returned prp_ id is a dead draft, not a successful change", Provenance: grammar.ProvenanceEmpiricalObservation},
		},
	}
}

// grammarEmptyResidueFixture is the same vocabulary with the residue retired to
// nothing — the "no live residue" edge the record can genuinely reach.
func grammarEmptyResidueFixture() grammar.Grammar {
	g := grammarFixture()
	g.Facts = []grammar.Fact{}
	return g
}

func TestRender_GrammarFull_Golden(t *testing.T) {
	want := "Change-set grammar — the shapes a proposal's changes[] may carry.\n" +
		"  Change types (3) — provenance: published-contract\n" +
		"    CreateAccountability  [nested-only]\n" +
		"    CreatePolicy  [top-level]\n" +
		"    UpdateRole  [top-level]\n" +
		"  Nesting rule — provenance: published-contract\n" +
		"    A nested-only type must appear as a child of CreateRole or UpdateRole, never as a top-level change.\n" +
		"    Nested-only types: CreateAccountability\n" +
		"  Empirical residue (2) — provenance: empirical-observation\n" +
		"    CSG-1  [accepted]  An own-circle policy is top-level\n" +
		"      Shape:    a top-level CreatePolicy part\n" +
		"      Symptom:  a wrapped shape is refused\n" +
		"    CSG-2  [accepted-but-invalid]  A self-targeting UpdateRole comes back invalid\n" +
		"      Shape:    an UpdateRole targeting the circle itself\n" +
		"      Symptom:  a returned prp_ id is a dead draft, not a successful change\n"
	assertRender(t, ResourceGrammar, FormatFull, NewGrammarView(grammarFixture()), want)
}

func TestRender_GrammarCompact_Golden(t *testing.T) {
	want := "top-level: CreatePolicy, UpdateRole\n" +
		"nested-only (inside CreateRole or UpdateRole): CreateAccountability\n" +
		"CSG-1  [accepted]  An own-circle policy is top-level\n" +
		"CSG-2  [accepted-but-invalid]  A self-targeting UpdateRole comes back invalid\n"
	assertRender(t, ResourceGrammar, FormatCompact, NewGrammarView(grammarFixture()), want)
}

// TestRender_GrammarFull_EmptyResidue_Golden pins the empty-residue variant: the
// contract vocabulary renders in full and the residue section states its absence
// explicitly. A vanished section would read as "there is nothing to know here",
// which is a different claim from "nothing is currently recorded".
func TestRender_GrammarFull_EmptyResidue_Golden(t *testing.T) {
	want := "Change-set grammar — the shapes a proposal's changes[] may carry.\n" +
		"  Change types (3) — provenance: published-contract\n" +
		"    CreateAccountability  [nested-only]\n" +
		"    CreatePolicy  [top-level]\n" +
		"    UpdateRole  [top-level]\n" +
		"  Nesting rule — provenance: published-contract\n" +
		"    A nested-only type must appear as a child of CreateRole or UpdateRole, never as a top-level change.\n" +
		"    Nested-only types: CreateAccountability\n" +
		"  Empirical residue (0) — provenance: empirical-observation\n" +
		"    no empirical residue is currently recorded\n"
	assertRender(t, ResourceGrammar, FormatFull, NewGrammarView(grammarEmptyResidueFixture()), want)
}

func TestRender_GrammarCompact_EmptyResidue_Golden(t *testing.T) {
	want := "top-level: CreatePolicy, UpdateRole\n" +
		"nested-only (inside CreateRole or UpdateRole): CreateAccountability\n" +
		"no empirical residue is currently recorded\n"
	assertRender(t, ResourceGrammar, FormatCompact, NewGrammarView(grammarEmptyResidueFixture()), want)
}

// TestRender_Grammar_NoNestedOnlyTypesOmitsTheRule: a contract with no nested-only
// type has no nesting rule to state, and stating one anyway would invent a
// constraint. The vocabulary still renders.
func TestRender_Grammar_NoNestedOnlyTypesOmitsTheRule(t *testing.T) {
	g := grammar.Grammar{
		ChangeTypes: []grammar.ChangeType{{Type: "CreateRole", Placement: grammar.PlacementTopLevel, Provenance: grammar.ProvenancePublishedContract}},
		Facts:       []grammar.Fact{},
	}
	for _, format := range []Format{FormatFull, FormatCompact} {
		got, err := Render(ResourceGrammar, format, NewGrammarView(g))
		if err != nil {
			t.Fatalf("render %s: %v", format, err)
		}
		if strings.Contains(got, "Nesting rule") || strings.Contains(got, "nested-only") {
			t.Errorf("%s states a nesting rule for a contract that has no nested-only type:\n%s", format, got)
		}
		if !strings.Contains(got, "CreateRole") {
			t.Errorf("%s dropped the vocabulary:\n%s", format, got)
		}
	}
}

// TestRender_Grammar_RendersEveryTypeAndFactOfTheRealArtifact runs both formats
// over the artifact the binary actually carries — the golden fixtures pin the
// shape, this pins the coverage, so a template that drops a branch on the real
// 21-type vocabulary is caught.
func TestRender_Grammar_RendersEveryTypeAndFactOfTheRealArtifact(t *testing.T) {
	g, err := grammar.Load()
	if err != nil {
		t.Fatalf("loading the embedded grammar: %v", err)
	}
	view := NewGrammarView(g)
	for _, format := range []Format{FormatFull, FormatCompact} {
		got, err := Render(ResourceGrammar, format, view)
		if err != nil {
			t.Fatalf("render %s: %v", format, err)
		}
		for _, ct := range g.ChangeTypes {
			if !strings.Contains(got, ct.Type) {
				t.Errorf("%s does not render change type %q:\n%s", format, ct.Type, got)
			}
		}
		for _, f := range g.Facts {
			if !strings.Contains(got, f.ID) {
				t.Errorf("%s does not render fact %q:\n%s", format, f.ID, got)
			}
			if !strings.Contains(got, f.Disposition) {
				t.Errorf("%s does not render fact %s's disposition %q:\n%s", format, f.ID, f.Disposition, got)
			}
		}
	}
}

// TestRender_Grammar_FullSeparatesProvenance is the accord's provenance-marking
// requirement for `full`: a reader can tell contract-published content from a
// verified observation. Both tokens appear, and the vocabulary is marked with the
// contract token rather than the observation one.
func TestRender_Grammar_FullSeparatesProvenance(t *testing.T) {
	got, err := Render(ResourceGrammar, FormatFull, NewGrammarView(grammarFixture()))
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	contractAt := strings.Index(got, grammar.ProvenancePublishedContract)
	observedAt := strings.Index(got, grammar.ProvenanceEmpiricalObservation)
	if contractAt < 0 || observedAt < 0 {
		t.Fatalf("full must mark both provenances; got:\n%s", got)
	}
	if contractAt > observedAt {
		t.Errorf("the contract-published sections must precede the empirical ones:\n%s", got)
	}
	// The vocabulary lines must sit under the contract marking, not the empirical
	// one: a type listed below the residue's header would read as an observation.
	typeAt := strings.Index(got, "CreatePolicy  [top-level]")
	if typeAt < 0 || typeAt > observedAt {
		t.Errorf("the change-type vocabulary must render above the empirical residue:\n%s", got)
	}
}

// TestRender_Grammar_NeverRendersMaintenanceMetadata: the record's Evidence and
// lineage are excluded by design, and the artifact's envelope is repo-facing. A
// template that reached for either would leak a development-repository pointer onto
// an operator's machine.
func TestRender_Grammar_NeverRendersMaintenanceMetadata(t *testing.T) {
	g, err := grammar.Load()
	if err != nil {
		t.Fatalf("loading the embedded grammar: %v", err)
	}
	view := NewGrammarView(g)
	for _, format := range []Format{FormatFull, FormatCompact} {
		got, err := Render(ResourceGrammar, format, view)
		if err != nil {
			t.Fatalf("render %s: %v", format, err)
		}
		for _, banned := range []string{"Evidence", "prp_ebe2815f", "prp_c76cd6bf", "supersedes", "DO NOT EDIT", "go generate", "internal/", "spec/glassfrog-api-v5.yaml", "plugin/skills"} {
			if strings.Contains(got, banned) {
				t.Errorf("%s renders the excluded/maintenance content %q:\n%s", format, banned, got)
			}
		}
	}
}

// TestNewGrammarView_PreservesArtifactOrdering: the view must not re-sort, or the
// human formats would disagree with the structured output about the reference's
// order — and determinism is a pinned contract, not a nicety.
func TestNewGrammarView_PreservesArtifactOrdering(t *testing.T) {
	g := grammar.Grammar{
		ChangeTypes: []grammar.ChangeType{
			{Type: "Zulu", Placement: grammar.PlacementTopLevel, Provenance: grammar.ProvenancePublishedContract},
			{Type: "Alpha", Placement: grammar.PlacementTopLevel, Provenance: grammar.ProvenancePublishedContract},
		},
		Facts: []grammar.Fact{
			{ID: "CSG-2", Title: "second", Shape: "s", Disposition: "accepted", Symptom: "y", Provenance: grammar.ProvenanceEmpiricalObservation},
			{ID: "CSG-1", Title: "first", Shape: "s", Disposition: "accepted", Symptom: "y", Provenance: grammar.ProvenanceEmpiricalObservation},
		},
	}
	view := NewGrammarView(g)
	if view.ChangeTypes[0].Type != "Zulu" || view.ChangeTypes[1].Type != "Alpha" {
		t.Errorf("the view re-sorted the vocabulary: %+v", view.ChangeTypes)
	}
	if view.Facts[0].ID != "CSG-2" || view.Facts[1].ID != "CSG-1" {
		t.Errorf("the view re-sorted the residue: %+v", view.Facts)
	}
}

// TestNewGrammarView_UnionsTheWrapperSet: taking the first entry's wrappers would
// under-state the rule if entries ever carried different sets.
func TestNewGrammarView_UnionsTheWrapperSet(t *testing.T) {
	g := grammar.Grammar{
		ChangeTypes: []grammar.ChangeType{
			{Type: "CreateDomain", Placement: grammar.PlacementNestedOnly, Wrappers: []string{"UpdateRole"}, Provenance: grammar.ProvenancePublishedContract},
			{Type: "CreateAccountability", Placement: grammar.PlacementNestedOnly, Wrappers: []string{"CreateRole"}, Provenance: grammar.ProvenancePublishedContract},
		},
		Facts: []grammar.Fact{},
	}
	view := NewGrammarView(g)
	if len(view.Wrappers) != 2 || view.Wrappers[0] != "CreateRole" || view.Wrappers[1] != "UpdateRole" {
		t.Errorf("the wrapper set must be the sorted union across nested-only entries; got %v", view.Wrappers)
	}
}
