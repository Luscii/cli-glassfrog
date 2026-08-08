package build

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/cucumber/godog"
)

// TestGrammarFactsGuardFeatures runs the executable acceptance for the guard
// (072, change-set-grammar-guard.feature). The scenarios drive CheckGrammarFacts
// over a fixture record and real spec-side sets, mutating one to trip a
// condition or retiring a fact to prove a complete retirement passes while a
// partial one fails. Runs with the ~@wip filter.
func TestGrammarFactsGuardFeatures(t *testing.T) {
	w := &grammarGuardWorld{}
	suite := godog.TestSuite{
		ScenarioInitializer: func(sc *godog.ScenarioContext) { w.register(sc) },
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"../../features/unguided-change-construction/change-set-grammar-guard.feature"},
			Tags:     "~@wip",
			TestingT: t,
		},
	}
	if suite.Run() != 0 {
		t.Fatal("non-zero status: change-set-grammar-guard feature scenarios failed")
	}
}

// grammarGuardWorld is the per-scenario state: the fixture record, the spec-side
// sets (loaded real, then mutated to model a refresh), and the guard's findings.
type grammarGuardWorld struct {
	record     string
	specEnum   []string
	specNested []string
	violations []string
	ran        bool
}

func (w *grammarGuardWorld) register(sc *godog.ScenarioContext) {
	sc.Before(func(ctx context.Context, _ *godog.Scenario) (context.Context, error) {
		enum, nested, err := LoadSpecChangeTypes()
		if err != nil {
			return ctx, fmt.Errorf("could not load spec-side sets: %w", err)
		}
		*w = grammarGuardWorld{
			record:     validGrammarRecordFixture(),
			specEnum:   enum,
			specNested: nested,
		}
		return ctx, nil
	})

	// Givens
	sc.Step(`^the record's nested-only citation list matched the vendored spec's nested-only set$`, w.givenNestedOnlyMatched)
	sc.Step(`^a record carrying the fact "([^"]*)"$`, w.givenRecordCarrying)
	sc.Step(`^a record whose manifest declared "([^"]*)"$`, w.givenManifestDeclared)
	sc.Step(`^a record carrying the fact "([^"]*)" in both its manifest and its fact sections$`, w.givenRecordCarrying)
	sc.Step(`^every recorded fact had been absorbed by the published contract$`, w.givenAllAbsorbed)

	// Whens
	sc.Step(`^a refreshed spec no longer carries the same nested-only set$`, w.whenSpecRefreshedNested)
	sc.Step(`^(.+) is introduced$`, w.whenDefectIntroduced)
	sc.Step(`^a contract-absorbed fact is retired by deleting its section and dropping its id from the manifest$`, w.whenCompleteRetirement)
	sc.Step(`^only (.+) is removed$`, w.whenPartialRemoval)
	sc.Step(`^the record is left in place with an empty manifest and no fact sections$`, w.whenEmptyShell)

	// Thens
	sc.Step(`^the guard will fail the build naming both sets$`, w.thenFailNamingBothSets)
	sc.Step(`^the failure will demand re-derivation or retirement before the change lands$`, w.thenDemandRederiveOrRetire)
	sc.Step(`^the guard will fail naming the fact "([^"]*)"$`, w.thenFailNamingFact)
	sc.Step(`^it will name (.+)$`, w.thenNameDetail)
	sc.Step(`^the guard will pass$`, w.thenGuardPasses)
	sc.Step(`^the surviving fact will still be recorded$`, w.thenSurvivingFactRecorded)
	sc.Step(`^the guard will fail naming "([^"]*)"$`, w.thenFailNamingFact)
	sc.Step(`^the failure will name the resolution path$`, w.thenNameResolutionPath)
	sc.Step(`^the guard will fail reporting the record as empty$`, w.thenFailEmpty)
	sc.Step(`^the failure will direct the maintainer to delete the record rather than keep a shell$`, w.thenDirectDelete)
}

// run evaluates the guard over the current fixture + spec sides, once.
func (w *grammarGuardWorld) run() {
	w.violations = CheckGrammarFacts(ParseGrammarFactsRecord(w.record), w.specEnum, w.specNested)
	w.ran = true
}

// ensureRan runs the guard if a Then is reached without an explicit When having
// driven it (the record is already in its Given state).
func (w *grammarGuardWorld) ensureRan() {
	if !w.ran {
		w.run()
	}
}

// --- Givens ---

func (w *grammarGuardWorld) givenNestedOnlyMatched() error {
	// Baseline: the fixture's nested-only citation set-equals the real spec's.
	if !stringSetsEqual(ParseGrammarFactsRecord(w.record).NestedOnly, w.specNested) {
		return fmt.Errorf("fixture nested-only citation does not match the spec's set")
	}
	return nil
}

func (w *grammarGuardWorld) givenRecordCarrying(id string) error {
	if !recordHasFact(w.record, id) {
		return fmt.Errorf("fixture does not carry the fact %q", id)
	}
	return nil
}

func (w *grammarGuardWorld) givenManifestDeclared(ids string) error {
	got := ParseGrammarFactsRecord(w.record).ManifestIDs
	if !stringSetsEqual(got, splitIDList(ids)) {
		return fmt.Errorf("fixture manifest is %v, scenario declares %q", got, ids)
	}
	return nil
}

func (w *grammarGuardWorld) givenAllAbsorbed() error { return nil }

// --- Whens ---

func (w *grammarGuardWorld) whenSpecRefreshedNested() error {
	// A refreshed spec drops a nested-only type — the record's citation no longer
	// set-equals the contract.
	if len(w.specNested) == 0 {
		return fmt.Errorf("spec nested-only set is empty; nothing to drop")
	}
	w.specNested = w.specNested[:len(w.specNested)-1]
	w.run()
	return nil
}

func (w *grammarGuardWorld) whenDefectIntroduced(defect string) error {
	switch {
	case strings.Contains(defect, "empty Evidence"):
		w.record = strings.Replace(w.record, "- **Evidence**: prp_ebe2815f", "- **Evidence**:", 1)
	case strings.Contains(defect, `Disposition of "probably-fine"`):
		w.record = strings.Replace(w.record, "- **Disposition**: accepted\n", "- **Disposition**: probably-fine\n", 1)
	default:
		return fmt.Errorf("unhandled defect %q", defect)
	}
	w.run()
	return nil
}

func (w *grammarGuardWorld) whenCompleteRetirement() error {
	w.record = dropManifestID(w.record, "CSG-2")
	w.record = removeFactSection(w.record, "CSG-2")
	w.run()
	return nil
}

func (w *grammarGuardWorld) whenPartialRemoval(half string) error {
	switch {
	case strings.Contains(half, "fact section"):
		w.record = removeFactSection(w.record, "CSG-1")
	case strings.Contains(half, "manifest entry"):
		w.record = dropManifestID(w.record, "CSG-1")
	default:
		return fmt.Errorf("unhandled removed half %q", half)
	}
	w.run()
	return nil
}

func (w *grammarGuardWorld) whenEmptyShell() error {
	w.record = emptyShellGrammarRecordFixture()
	w.run()
	return nil
}

// --- Thens ---

func (w *grammarGuardWorld) thenFailNamingBothSets() error {
	w.ensureRan()
	if len(w.violations) == 0 {
		return fmt.Errorf("guard passed; expected a nested-only mismatch failure")
	}
	joined := strings.Join(w.violations, "\n")
	if !containsFold(joined, "nested-only") {
		return fmt.Errorf("failure does not name the nested-only invariant: %s", joined)
	}
	// Both sets readable from the failure: the dropped type names the difference.
	if !strings.Contains(joined, "RemoveDomain") {
		return fmt.Errorf("failure does not make both sets readable (missing the differing type): %s", joined)
	}
	return nil
}

func (w *grammarGuardWorld) thenDemandRederiveOrRetire() error {
	joined := strings.Join(w.violations, "\n")
	if !containsFold(joined, "re-derive") || !containsFold(joined, "retire") {
		return fmt.Errorf("failure does not demand re-derivation or retirement: %s", joined)
	}
	return nil
}

func (w *grammarGuardWorld) thenFailNamingFact(id string) error {
	w.ensureRan()
	if len(w.violations) == 0 {
		return fmt.Errorf("guard passed; expected a failure naming %q", id)
	}
	if !strings.Contains(strings.Join(w.violations, "\n"), id) {
		return fmt.Errorf("failure does not name the fact %q: %v", id, w.violations)
	}
	return nil
}

func (w *grammarGuardWorld) thenNameDetail(detail string) error {
	joined := strings.Join(w.violations, "\n")
	var want string
	switch {
	case strings.Contains(detail, "field label"):
		want = "Evidence"
	case strings.Contains(detail, "unrecognized value"):
		want = "probably-fine"
	default:
		return fmt.Errorf("unhandled detail %q", detail)
	}
	if !strings.Contains(joined, want) {
		return fmt.Errorf("failure does not name %q (for detail %q): %s", want, detail, joined)
	}
	return nil
}

func (w *grammarGuardWorld) thenGuardPasses() error {
	w.ensureRan()
	if len(w.violations) != 0 {
		return fmt.Errorf("guard failed after a complete retirement:\n  - %s", strings.Join(w.violations, "\n  - "))
	}
	return nil
}

func (w *grammarGuardWorld) thenSurvivingFactRecorded() error {
	if !recordHasFact(w.record, "CSG-1") {
		return fmt.Errorf("the surviving fact CSG-1 is no longer recorded")
	}
	return nil
}

func (w *grammarGuardWorld) thenNameResolutionPath() error {
	joined := strings.Join(w.violations, "\n")
	if !containsFold(joined, "complete the retirement") && !containsFold(joined, "add the id to the manifest") {
		return fmt.Errorf("failure does not name a resolution path: %s", joined)
	}
	return nil
}

func (w *grammarGuardWorld) thenFailEmpty() error {
	w.ensureRan()
	joined := strings.Join(w.violations, "\n")
	if !containsFold(joined, "empty") || !containsFold(joined, "no fact sections") {
		return fmt.Errorf("failure does not report the record as empty: %s", joined)
	}
	return nil
}

func (w *grammarGuardWorld) thenDirectDelete() error {
	joined := strings.Join(w.violations, "\n")
	if !containsFold(joined, "delete the record") {
		return fmt.Errorf("failure does not direct the maintainer to delete the record: %s", joined)
	}
	return nil
}

// --- fixture surgery helpers ---

// recordHasFact reports whether the record carries a `## <id> —` fact section.
func recordHasFact(record, id string) bool {
	for _, f := range ParseGrammarFactsRecord(record).Facts {
		if f.ID == id {
			return true
		}
	}
	return false
}

// removeFactSection deletes the `## <id> — …` section (heading through the line
// before the next `## ` heading or EOF), leaving the manifest untouched — the
// partial-retirement / complete-retirement surgery.
func removeFactSection(record, id string) string {
	lines := strings.Split(record, "\n")
	start, end := -1, len(lines)
	for i, line := range lines {
		if strings.HasPrefix(line, "## "+id+" ") {
			start = i
			continue
		}
		if start >= 0 && strings.HasPrefix(line, "## ") {
			end = i
			break
		}
	}
	if start < 0 {
		return record
	}
	kept := append(append([]string{}, lines[:start]...), lines[end:]...)
	return strings.Join(kept, "\n")
}

// dropManifestID rewrites the `Live facts` line without the given id, leaving
// every fact section untouched — the other half of the retirement surgery.
func dropManifestID(record, id string) string {
	lines := strings.Split(record, "\n")
	for i, line := range lines {
		if !strings.Contains(line, "**Live facts**:") {
			continue
		}
		var kept []string
		for _, cur := range splitIDList(strings.SplitN(line, ":", 2)[1]) {
			if cur != id {
				kept = append(kept, cur)
			}
		}
		lines[i] = "- **Live facts**: " + strings.Join(kept, ", ")
		break
	}
	return strings.Join(lines, "\n")
}
