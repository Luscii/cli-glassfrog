package build

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/cucumber/godog"
)

// TestGrammarFactsFeatures runs the executable acceptance for the change-set
// grammar record (072, change-set-grammar-facts.feature). Like the other
// build-side suites its Paths name ONLY this spec's feature file and it runs
// with the ~@wip filter, so only the scenarios implemented so far execute.
//
// The deliverable is a committed markdown knowledge artifact, so the content
// scenarios assert against that file, following the operator-path convention:
// content assertions read a whitespace-collapsed copy (grammarNorm), structural
// checks read the raw file. The three @validation scenarios and the
// process-inexecutable retirement scenario stay @wip.
func TestGrammarFactsFeatures(t *testing.T) {
	w := &grammarFactsWorld{}
	suite := godog.TestSuite{
		ScenarioInitializer: func(sc *godog.ScenarioContext) { w.register(sc) },
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"../../features/unguided-change-construction/change-set-grammar-facts.feature"},
			Tags:     "~@wip",
			TestingT: t,
		},
	}
	if suite.Run() != 0 {
		t.Fatal("non-zero status: change-set-grammar-facts feature scenarios failed")
	}
}

// grammarFactsWorld is the per-scenario state: the raw record, its parse, and
// the fact id named by a Given (for the single-fact scenarios).
type grammarFactsWorld struct {
	raw    string
	record GrammarFactsRecord
	factID string
}

func (w *grammarFactsWorld) register(sc *godog.ScenarioContext) {
	sc.Before(func(ctx context.Context, _ *godog.Scenario) (context.Context, error) {
		*w = grammarFactsWorld{}
		return ctx, nil
	})

	// Givens
	sc.Step(`^the record carried the fact "([^"]*)"$`, w.givenRecordCarriesFact)
	sc.Step(`^an assembler was about to build a change set touching a circle's own governance$`, w.givenAssembler)
	sc.Step(`^the published contract carried the change-type enum at "([^"]*)"$`, w.givenContractEnum)
	sc.Step(`^the landed record$`, w.givenLandedRecord)

	// Whens — every "consult" just ensures the record is loaded.
	sc.Step(`^the record is consulted for changing a circle's own policy$`, w.whenConsult)
	sc.Step(`^the record is consulted for a role update that self-targets the circle from inside its own governance$`, w.whenConsult)
	sc.Step(`^it consults the record before assembling$`, w.whenConsult)
	sc.Step(`^a consumer reads the fact's disposition$`, w.whenConsult)
	sc.Step(`^the record names the enumerated change types or the nested-only rule$`, w.whenConsult)
	sc.Step(`^each fact section is read$`, w.whenConsult)
	sc.Step(`^the record's facts were observed only from live server behavior$`, w.givenLandedRecord)
	sc.Step(`^the record is checked for its empirical marker$`, w.whenConsult)

	// Thens
	sc.Step(`^it will state the change as a top-level "([^"]*)" part with no "([^"]*)" wrapper$`, w.thenTopLevelNoWrapper)
	sc.Step(`^its Symptom field will state that a wrapped shape is refused while the web UI generates exactly the top-level form$`, w.thenSymptomWrappedRefused)
	sc.Step(`^it will state that the server accepts the create but returns the proposal invalid with a blocking alert and no available transitions$`, w.thenAcceptedInvalid)
	sc.Step(`^its Disposition field will read "([^"]*)" rather than "([^"]*)"$`, w.thenDispositionReads)
	sc.Step(`^it will find the accepted shape to use in "([^"]*)"$`, w.thenAcceptedShapeIn)
	sc.Step(`^it will find the dead shape to avoid in "([^"]*)"$`, w.thenDeadShapeIn)
	sc.Step(`^no round-trip against the server will be needed to learn either$`, w.thenNoRoundTrip)
	sc.Step(`^the record will distinguish the server returning a created prp_ id from the proposal being valid$`, w.thenDistinguishCreatedFromValid)
	sc.Step(`^it will not let a returned id read as a successful governance change$`, w.thenIdNotSuccess)
	sc.Step(`^it will cite the contract anchor by schema and property name$`, w.thenCitesAnchor)
	sc.Step(`^it will restate no enum values beyond its citation lists$`, w.thenNoRestatement)
	sc.Step(`^every fact will carry all five required fields$`, w.thenFiveFields)
	sc.Step(`^its Evidence will name the live proposal the fact was verified against$`, w.thenEvidenceNamesProposal)
	sc.Step(`^its Provenance will name the LEARNINGS entry it supersedes$`, w.thenProvenanceNamesLearnings)
	sc.Step(`^a leading marker will state that every fact is observed behavior and none is part of the published contract$`, w.thenMarkerWellFormed)
	sc.Step(`^a record missing that marker will fail the guard$`, w.thenMissingMarkerFailsGuard)
}

// ensureRecord loads and parses the record once per scenario.
func (w *grammarFactsWorld) ensureRecord() error {
	if w.raw != "" {
		return nil
	}
	raw, err := ReadGrammarFactsRecord()
	if err != nil {
		return fmt.Errorf("record did not load: %w", err)
	}
	w.raw = raw
	w.record = ParseGrammarFactsRecord(raw)
	return nil
}

// fact returns the parsed fact with the given id, or an error naming it.
func (w *grammarFactsWorld) fact(id string) (GrammarFact, error) {
	for _, f := range w.record.Facts {
		if f.ID == id {
			return f, nil
		}
	}
	return GrammarFact{}, fmt.Errorf("record carries no fact %q; has %v", id, w.factIDs())
}

func (w *grammarFactsWorld) factIDs() []string {
	var ids []string
	for _, f := range w.record.Facts {
		ids = append(ids, f.ID)
	}
	return ids
}

// --- Givens ---

func (w *grammarFactsWorld) givenRecordCarriesFact(id string) error {
	if err := w.ensureRecord(); err != nil {
		return err
	}
	if _, err := w.fact(id); err != nil {
		return err
	}
	w.factID = id
	return nil
}

func (w *grammarFactsWorld) givenAssembler() error    { return w.ensureRecord() }
func (w *grammarFactsWorld) givenLandedRecord() error { return w.ensureRecord() }

func (w *grammarFactsWorld) givenContractEnum(anchor string) error {
	if err := w.ensureRecord(); err != nil {
		return err
	}
	// The scenario names the enum anchor; the record must cite exactly that
	// schema.property path (structural check against the raw file).
	if !strings.Contains(w.raw, anchor) {
		return fmt.Errorf("record does not cite the enum anchor %q", anchor)
	}
	return nil
}

func (w *grammarFactsWorld) whenConsult() error { return w.ensureRecord() }

// --- Thens ---

func (w *grammarFactsWorld) thenTopLevelNoWrapper(createType, wrapperType string) error {
	f, err := w.fact("CSG-1")
	if err != nil {
		return err
	}
	shape := f.Fields["Shape"]
	if !strings.Contains(shape, createType) {
		return fmt.Errorf("CSG-1 Shape does not name the top-level %q part: %q", createType, shape)
	}
	if !containsFold(shape, "top-level") {
		return fmt.Errorf("CSG-1 Shape does not state the shape is top-level: %q", shape)
	}
	if !strings.Contains(shape, wrapperType) || !containsFold(shape, "wrap") {
		return fmt.Errorf("CSG-1 Shape does not state there is no %q wrapper: %q", wrapperType, shape)
	}
	return nil
}

func (w *grammarFactsWorld) thenSymptomWrappedRefused() error {
	f, err := w.fact("CSG-1")
	if err != nil {
		return err
	}
	sym := grammarNorm(f.Fields["Symptom"])
	for _, want := range []string{"refused", "web ui", "top-level"} {
		if !containsFold(sym, want) {
			return fmt.Errorf("CSG-1 Symptom is missing %q: %q", want, sym)
		}
	}
	return nil
}

func (w *grammarFactsWorld) thenAcceptedInvalid() error {
	f, err := w.fact("CSG-2")
	if err != nil {
		return err
	}
	sym := grammarNorm(f.Fields["Symptom"])
	if !containsFold(sym, "accept") {
		return fmt.Errorf("CSG-2 Symptom does not state the server accepts the create: %q", sym)
	}
	if !containsFold(sym, "invalid") && !containsFold(sym, "valid: false") {
		return fmt.Errorf("CSG-2 Symptom does not state the proposal is returned invalid: %q", sym)
	}
	if !containsFold(sym, "blocking alert") {
		return fmt.Errorf("CSG-2 Symptom does not name the blocking alert: %q", sym)
	}
	if !containsFold(sym, "available_transitions") {
		return fmt.Errorf("CSG-2 Symptom does not name the empty available_transitions: %q", sym)
	}
	return nil
}

func (w *grammarFactsWorld) thenDispositionReads(want, notWant string) error {
	f, err := w.fact("CSG-2")
	if err != nil {
		return err
	}
	if got := f.Disposition(); got != want {
		return fmt.Errorf("CSG-2 Disposition reads %q, want %q (not %q)", got, want, notWant)
	}
	return nil
}

func (w *grammarFactsWorld) thenAcceptedShapeIn(id string) error {
	f, err := w.fact(id)
	if err != nil {
		return err
	}
	if f.Disposition() != "accepted" {
		return fmt.Errorf("%s is not the accepted shape (Disposition %q)", id, f.Disposition())
	}
	return nil
}

func (w *grammarFactsWorld) thenDeadShapeIn(id string) error {
	f, err := w.fact(id)
	if err != nil {
		return err
	}
	if f.Disposition() != "accepted-but-invalid" {
		return fmt.Errorf("%s is not the dead shape (Disposition %q)", id, f.Disposition())
	}
	return nil
}

func (w *grammarFactsWorld) thenNoRoundTrip() error {
	// The knowledge is readable without a server round-trip iff both shapes are
	// fully present in the record — each with a Shape and a Symptom to read.
	for _, id := range []string{"CSG-1", "CSG-2"} {
		f, err := w.fact(id)
		if err != nil {
			return err
		}
		if strings.TrimSpace(f.Fields["Shape"]) == "" || strings.TrimSpace(f.Fields["Symptom"]) == "" {
			return fmt.Errorf("%s is not fully readable without a round-trip (Shape/Symptom empty)", id)
		}
	}
	return nil
}

func (w *grammarFactsWorld) thenDistinguishCreatedFromValid() error {
	f, err := w.fact("CSG-2")
	if err != nil {
		return err
	}
	sym := grammarNorm(f.Fields["Symptom"])
	if !containsFold(sym, "created") || !containsFold(sym, "valid") {
		return fmt.Errorf("CSG-2 Symptom does not distinguish a created id from validity: %q", sym)
	}
	return nil
}

func (w *grammarFactsWorld) thenIdNotSuccess() error {
	f, err := w.fact("CSG-2")
	if err != nil {
		return err
	}
	sym := grammarNorm(f.Fields["Symptom"])
	if !containsFold(sym, "not a successful governance change") {
		return fmt.Errorf("CSG-2 Symptom does not deny that a returned id is a successful governance change: %q", sym)
	}
	return nil
}

func (w *grammarFactsWorld) thenCitesAnchor() error {
	if !strings.Contains(w.raw, "ProposalChange.properties.type.enum") {
		return fmt.Errorf("record does not cite the enum by schema.property anchor")
	}
	if !strings.Contains(w.raw, "`ProposalChange`") {
		return fmt.Errorf("record does not cite the nested-only rule by the ProposalChange schema name")
	}
	// Cite by name, never by line number: no `<file>.yaml:NNN` or "line NNN".
	if lineCitationRE.MatchString(w.raw) {
		return fmt.Errorf("record cites by line number, which drifts on every refresh")
	}
	return nil
}

func (w *grammarFactsWorld) thenNoRestatement() error {
	if extra := RestatedTypesBeyondCitations(w.record); len(extra) != 0 {
		return fmt.Errorf("record restates enum values beyond its citation lists: %v", extra)
	}
	return nil
}

func (w *grammarFactsWorld) thenFiveFields() error {
	if len(w.record.Facts) == 0 {
		return fmt.Errorf("record carries no fact sections")
	}
	for _, f := range w.record.Facts {
		for _, label := range GrammarRequiredFields {
			if strings.TrimSpace(f.Fields[label]) == "" {
				return fmt.Errorf("%s is missing required field %q", f.ID, label)
			}
		}
	}
	return nil
}

func (w *grammarFactsWorld) thenEvidenceNamesProposal() error {
	for _, f := range w.record.Facts {
		if !strings.Contains(f.Fields["Evidence"], "prp_") {
			return fmt.Errorf("%s Evidence does not name a live proposal (prp_…): %q", f.ID, f.Fields["Evidence"])
		}
	}
	return nil
}

func (w *grammarFactsWorld) thenProvenanceNamesLearnings() error {
	for _, f := range w.record.Facts {
		prov := f.Fields["Provenance"]
		if !containsFold(prov, "LEARNINGS") || !strings.Contains(prov, "2026-08-05") {
			return fmt.Errorf("%s Provenance does not name the LEARNINGS 2026-08-05 entry: %q", f.ID, prov)
		}
	}
	return nil
}

func (w *grammarFactsWorld) thenMarkerWellFormed() error {
	if !MarkerIsWellFormed(w.record.Marker) {
		return fmt.Errorf("the leading marker is absent or does not state observed-behavior / not-part-of-contract: %q", w.record.Marker)
	}
	return nil
}

// thenMissingMarkerFailsGuard strips the leading marker from the real record and
// runs the guard against the real spec — the marker's absence must surface as a
// condition-8 violation.
func (w *grammarFactsWorld) thenMissingMarkerFailsGuard() error {
	idx := strings.Index(w.raw, "# Change-Set Grammar Facts")
	if idx < 0 {
		return fmt.Errorf("could not locate the document header to strip the marker")
	}
	stripped := ParseGrammarFactsRecord(w.raw[idx:])
	enum, nested, err := LoadSpecChangeTypes()
	if err != nil {
		return fmt.Errorf("could not load spec-side sets: %w", err)
	}
	v := CheckGrammarFacts(stripped, enum, nested)
	for _, msg := range v {
		if containsFold(msg, "marker") {
			return nil
		}
	}
	return fmt.Errorf("guard did not fail on the missing marker; got %v", v)
}

// grammarNorm produces the whitespace-normalized copy content assertions read
// (operator-path BDD convention): markdown emphasis/code markers (`*` and
// backticks) are dropped and runs of whitespace collapse to single spaces, so a
// phrase reads the same whether or not the author bolded a word inside it.
// Structural checks read the raw file instead.
func grammarNorm(s string) string {
	s = strings.NewReplacer("*", "", "`", "").Replace(s)
	return strings.Join(strings.Fields(s), " ")
}
