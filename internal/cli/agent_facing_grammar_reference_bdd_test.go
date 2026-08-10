package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/Luscii/cli-glassfrog/internal/build"
	"github.com/Luscii/cli-glassfrog/internal/grammar"
	"github.com/Luscii/cli-glassfrog/internal/output"
	"github.com/Luscii/cli-glassfrog/internal/render"
	"github.com/cucumber/godog"
)

// TestAgentFacingGrammarReferenceFeatures runs the executable acceptance for the
// Agent-Facing Grammar Reference (077). Its Paths name ONLY this spec's feature
// file — never the features/ directory — so the suite reports its own independent
// scenario count and un-@wip-ping these scenarios cannot disturb another suite.
//
// The feature spans two surfaces: the shipped command (this package) and the
// repository-side drift guard (internal/build). The repo convention is one suite
// per feature file — a second suite over the same file would need per-scenario
// tags the file does not carry — so this suite owns both, importing internal/build
// for the guard scenario the way the Version Embedding suite already does. The
// dependency direction is unchanged: internal/build still never imports
// internal/cli.
//
// The three @validation scenarios stay @wip (held for the validate skill) and are
// skipped by the ~@wip filter.
func TestAgentFacingGrammarReferenceFeatures(t *testing.T) {
	suite := godog.TestSuite{
		ScenarioInitializer: initializeGrammarReferenceScenario,
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"../../features/unguided-change-construction/agent-facing-grammar-reference.feature"},
			Tags:     "~@wip",
			TestingT: t,
		},
	}
	if suite.Run() != 0 {
		t.Fatal("non-zero status: agent-facing-grammar-reference feature scenarios failed")
	}
}

// grammarRefWorld is the per-scenario state. Nothing in it touches the network,
// the real environment, or a real credential file — the command under test is
// client-less, and the guard scenario perturbs its inputs in memory so the
// committed artifact on disk stays truthful for the next run.
type grammarRefWorld struct {
	// The drift-guard scenario's three sides.
	committed   []byte
	regenerated []byte
	manifestIDs []string
	findings    []string
	guardRan    bool

	// The rendering scenarios: the payload under test (the embedded grammar, or a
	// perturbation of it a Given stages) and the human text a When produced.
	payload     grammar.Grammar
	payloadSet  bool
	humanOutput string
	rendered    bool
}

func initializeGrammarReferenceScenario(sc *godog.ScenarioContext) {
	w := &grammarRefWorld{}
	sc.Before(func(ctx context.Context, _ *godog.Scenario) (context.Context, error) {
		*w = grammarRefWorld{}
		return ctx, nil
	})

	// --- The drift guard (T003) ---
	sc.Step(`^a vendored-contract refresh changed the change-type enum$`, w.givenContractRefreshChangedTheEnum)
	sc.Step(`^the committed grammar artifact was not regenerated$`, w.givenArtifactNotRegenerated)
	sc.Step(`^the repository's merge-gating verification runs$`, w.whenMergeGatingVerificationRuns)
	sc.Step(`^it will fail naming the divergence between the contract and the rendered vocabulary$`, w.thenFailsNamingContractDivergence)
	sc.Step(`^the failure will name regeneration as the remedy$`, w.thenFailureNamesRegeneration)

	// --- The human formats (T004) ---
	sc.Step(`^the record carried the facts "([^"]*)" and "([^"]*)"$`, w.givenRecordCarriesFacts)
	sc.Step(`^the grammar record carried no live facts$`, w.givenRecordCarriesNoLiveFacts)
	sc.Step(`^the reference is rendered in the default human format$`, w.whenRenderedInDefaultHumanFormat)
	sc.Step(`^a practitioner runs "glassfrog proposal grammar --output (full|compact)"$`, w.whenRenderedInNamedHumanFormat)
	sc.Step(`^every change type will appear with its placement class$`, w.thenEveryTypeCarriesItsPlacement)
	sc.Step(`^the nesting rule will be stated once with its wrapper types$`, w.thenNestingRuleStatedOnce)
	sc.Step(`^each fact will carry its title, shape, disposition, and symptom$`, w.thenEachFactCarriesEveryField)
	sc.Step(`^the contract-derived content will be visibly separated from the empirical observations$`, w.thenProvenanceVisiblySeparated)
	sc.Step(`^every change type will appear with its placement class in condensed form$`, w.thenEveryTypeCondensedByPlacement)
	sc.Step(`^each fact will appear as one line carrying its id, disposition, and title$`, w.thenEachFactOnOneLine)
	sc.Step(`^the contract-derived change-type vocabulary will still render in full$`, w.thenVocabularyStillRendersInFull)
	sc.Step(`^the output will state that no empirical residue is currently recorded$`, w.thenStatesNoEmpiricalResidue)
}

// --- The drift guard (T003) -------------------------------------------------

// loadGuardSides reads the three sides the guard compares. A side that will not
// load is a step failure, not a guard finding: the guard checks agreement between
// sides, and an unloadable side is a different problem.
func (w *grammarRefWorld) loadGuardSides() error {
	if w.committed != nil {
		return nil
	}
	committed, err := build.ReadGrammarArtifact()
	if err != nil {
		return fmt.Errorf("reading the committed grammar artifact: %w", err)
	}
	regenerated, err := build.RenderGrammarArtifact()
	if err != nil {
		return fmt.Errorf("re-deriving the grammar artifact from its sources: %w", err)
	}
	raw, err := build.ReadGrammarFactsRecord()
	if err != nil {
		return fmt.Errorf("reading the grammar record: %w", err)
	}
	w.committed, w.regenerated = committed, regenerated
	w.manifestIDs = build.ParseGrammarFactsRecord(raw).ManifestIDs
	return nil
}

// givenContractRefreshChangedTheEnum models a refreshed vendored contract by
// changing what a FRESH DERIVATION would produce — a new enum member appears in
// the vocabulary — without touching the vendored file or the committed artifact.
// That is exactly the state a refresh PR is in before the generator runs.
func (w *grammarRefWorld) givenContractRefreshChangedTheEnum() error {
	if err := w.loadGuardSides(); err != nil {
		return err
	}
	var artifact grammar.Artifact
	if err := json.Unmarshal(w.regenerated, &artifact); err != nil {
		return fmt.Errorf("decoding the fresh derivation: %w", err)
	}
	artifact.Grammar.ChangeTypes = append(artifact.Grammar.ChangeTypes, grammar.ChangeType{
		Type:       "ZzArchiveRole", // sorts last, so the refresh is an addition rather than a reorder
		Placement:  grammar.PlacementTopLevel,
		Provenance: grammar.ProvenancePublishedContract,
	})
	refreshed, err := build.MarshalGrammarArtifact(artifact)
	if err != nil {
		return fmt.Errorf("encoding the refreshed derivation: %w", err)
	}
	w.regenerated = refreshed
	return nil
}

// givenArtifactNotRegenerated is the absence of an action: the committed bytes
// stay exactly as checked in. Asserted rather than assumed, so the scenario states
// what it depends on.
func (w *grammarRefWorld) givenArtifactNotRegenerated() error {
	if err := w.loadGuardSides(); err != nil {
		return err
	}
	if string(w.committed) == string(w.regenerated) {
		return fmt.Errorf("the committed artifact already matches the refreshed derivation — the scenario's premise (no regeneration) did not hold")
	}
	return nil
}

func (w *grammarRefWorld) whenMergeGatingVerificationRuns() error {
	if err := w.loadGuardSides(); err != nil {
		return err
	}
	w.findings = build.CheckGrammarArtifact(w.committed, w.regenerated, w.manifestIDs)
	w.guardRan = true
	return nil
}

func (w *grammarRefWorld) thenFailsNamingContractDivergence() error {
	if !w.guardRan {
		return fmt.Errorf("the guard was never run")
	}
	if len(w.findings) == 0 {
		return fmt.Errorf("the guard reported no findings — a contract refresh that outran the artifact passed the merge gate")
	}
	joined := strings.Join(w.findings, " | ")
	if !strings.Contains(joined, "CONTRACT-DERIVED") {
		return fmt.Errorf("no finding names the contract-derived vocabulary as the diverged half: %s", joined)
	}
	if !strings.Contains(joined, build.VendoredSpecPath) {
		return fmt.Errorf("no finding names the vendored contract %q: %s", build.VendoredSpecPath, joined)
	}
	return nil
}

func (w *grammarRefWorld) thenFailureNamesRegeneration() error {
	if len(w.findings) == 0 {
		return fmt.Errorf("the guard reported no findings, so no remedy was named")
	}
	for _, f := range w.findings {
		if !strings.Contains(f, build.GrammarRegenerationStep) {
			return fmt.Errorf("finding %q does not name the regeneration step %q as the remedy", f, build.GrammarRegenerationStep)
		}
	}
	return nil
}

// --- The human formats (T004) -----------------------------------------------

// grammarPayload is the structure under test: the artifact the binary carries,
// unless a Given staged a perturbation of it (the retired-residue edge).
func (w *grammarRefWorld) grammarPayload() (grammar.Grammar, error) {
	if w.payloadSet {
		return w.payload, nil
	}
	g, err := grammar.Load()
	if err != nil {
		return grammar.Grammar{}, fmt.Errorf("loading the embedded grammar: %w", err)
	}
	w.payload, w.payloadSet = g, true
	return w.payload, nil
}

// givenRecordCarriesFacts asserts the premise rather than staging it: the named
// facts must be the ones the shipped artifact actually renders, so a retirement
// that outruns the scenario surfaces here instead of silently weakening it.
func (w *grammarRefWorld) givenRecordCarriesFacts(first, second string) error {
	g, err := w.grammarPayload()
	if err != nil {
		return err
	}
	for _, want := range []string{first, second} {
		found := false
		for _, f := range g.Facts {
			if f.ID == want {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("the rendered residue carries no fact %q; it carries %v", want, grammarFactIDs(g))
		}
	}
	return nil
}

// givenRecordCarriesNoLiveFacts stages the retired-residue edge: every fact has
// left the record, so the artifact's residue is the empty list. The shipped record
// carries two facts, so this state is reachable only by perturbing the payload —
// which is why it is asserted at the rendering rather than through a real
// invocation.
func (w *grammarRefWorld) givenRecordCarriesNoLiveFacts() error {
	g, err := w.grammarPayload()
	if err != nil {
		return err
	}
	g.Facts = []grammar.Fact{}
	w.payload, w.payloadSet = g, true
	return nil
}

// renderIn renders the staged payload through the production render path — the
// same render.Render call the command makes — and stores the text for the Thens.
func (w *grammarRefWorld) renderIn(format render.Format) error {
	g, err := w.grammarPayload()
	if err != nil {
		return err
	}
	text, err := render.Render(render.ResourceGrammar, format, render.NewGrammarView(g))
	if err != nil {
		return fmt.Errorf("rendering the grammar in %s: %w", format, err)
	}
	w.humanOutput, w.rendered = text, true
	return nil
}

// whenRenderedInDefaultHumanFormat renders in whatever the production default
// resolves to, rather than naming `full` — so a change to the default cannot leave
// this scenario asserting a format nobody gets.
func (w *grammarRefWorld) whenRenderedInDefaultHumanFormat() error {
	return w.renderIn(humanFormat(output.DefaultFormat))
}

// whenRenderedInNamedHumanFormat resolves the invocation's --output value through
// the production selection chain and renders in the format it names. It goes
// through render rather than the cobra leaf because these three scenarios pin the
// human formats' CONTENT, which is the rendering's contract; the command's own
// dispatch — that `--output full` reaches this renderer, that a positional is a
// usage error, that no request is made — is asserted by the command scenarios,
// which run the real leaf end to end.
func (w *grammarRefWorld) whenRenderedInNamedHumanFormat(name string) error {
	format, err := output.ParseFormat(name)
	if err != nil {
		return fmt.Errorf("parsing the --output value %q: %w", name, err)
	}
	return w.renderIn(humanFormat(format))
}

func grammarFactIDs(g grammar.Grammar) []string {
	ids := make([]string, 0, len(g.Facts))
	for _, f := range g.Facts {
		ids = append(ids, f.ID)
	}
	return ids
}

// grammarOutputLines splits the rendered text into non-empty lines.
func (w *grammarRefWorld) grammarOutputLines() []string {
	var lines []string
	for _, l := range strings.Split(w.humanOutput, "\n") {
		if strings.TrimSpace(l) != "" {
			lines = append(lines, l)
		}
	}
	return lines
}

func (w *grammarRefWorld) requireRendered() error {
	if !w.rendered {
		return fmt.Errorf("nothing was rendered")
	}
	return nil
}

func (w *grammarRefWorld) thenEveryTypeCarriesItsPlacement() error {
	if err := w.requireRendered(); err != nil {
		return err
	}
	g, err := w.grammarPayload()
	if err != nil {
		return err
	}
	if len(g.ChangeTypes) == 0 {
		return fmt.Errorf("the payload carries no change types to render")
	}
	for _, ct := range g.ChangeTypes {
		// The type and its placement must appear TOGETHER — a type listed
		// somewhere and a placement class listed somewhere else would satisfy a
		// two-substring check while telling a reader nothing.
		if !grammarLineWith(w.grammarOutputLines(), ct.Type, ct.Placement) {
			return fmt.Errorf("no line carries change type %q with its placement %q:\n%s", ct.Type, ct.Placement, w.humanOutput)
		}
	}
	return nil
}

func (w *grammarRefWorld) thenNestingRuleStatedOnce() error {
	if err := w.requireRendered(); err != nil {
		return err
	}
	g, err := w.grammarPayload()
	if err != nil {
		return err
	}
	var nested, wrappers []string
	wrapperSeen := map[string]bool{}
	for _, ct := range g.ChangeTypes {
		if ct.Placement != grammar.PlacementNestedOnly {
			continue
		}
		nested = append(nested, ct.Type)
		for _, wr := range ct.Wrappers {
			if !wrapperSeen[wr] {
				wrapperSeen[wr] = true
				wrappers = append(wrappers, wr)
			}
		}
	}
	if len(nested) == 0 {
		return fmt.Errorf("the payload carries no nested-only type, so there is no nesting rule to state")
	}
	// Stated ONCE: exactly one line introduces the rule, rather than the wrapper
	// pair being repeated on every nested-only entry.
	ruleLines := 0
	var ruleLine string
	for _, l := range w.grammarOutputLines() {
		if strings.Contains(l, "must appear as a child of") {
			ruleLines++
			ruleLine = l
		}
	}
	if ruleLines != 1 {
		return fmt.Errorf("the nesting rule must be stated exactly once; found %d statements:\n%s", ruleLines, w.humanOutput)
	}
	for _, wr := range wrappers {
		if !strings.Contains(ruleLine, wr) {
			return fmt.Errorf("the nesting rule does not name the wrapper type %q: %q", wr, ruleLine)
		}
	}
	for _, n := range nested {
		if !strings.Contains(w.humanOutput, n) {
			return fmt.Errorf("the nested-only type %q is not named:\n%s", n, w.humanOutput)
		}
	}
	return nil
}

func (w *grammarRefWorld) thenEachFactCarriesEveryField() error {
	if err := w.requireRendered(); err != nil {
		return err
	}
	g, err := w.grammarPayload()
	if err != nil {
		return err
	}
	if len(g.Facts) == 0 {
		return fmt.Errorf("the payload carries no facts to render")
	}
	for _, f := range g.Facts {
		for label, value := range map[string]string{
			"title":       f.Title,
			"shape":       f.Shape,
			"disposition": f.Disposition,
			"symptom":     f.Symptom,
		} {
			if !strings.Contains(w.humanOutput, value) {
				return fmt.Errorf("fact %s's %s is not rendered (%q):\n%s", f.ID, label, value, w.humanOutput)
			}
		}
	}
	return nil
}

func (w *grammarRefWorld) thenProvenanceVisiblySeparated() error {
	if err := w.requireRendered(); err != nil {
		return err
	}
	contractAt := strings.Index(w.humanOutput, grammar.ProvenancePublishedContract)
	observedAt := strings.Index(w.humanOutput, grammar.ProvenanceEmpiricalObservation)
	if contractAt < 0 {
		return fmt.Errorf("the output carries no %q marking:\n%s", grammar.ProvenancePublishedContract, w.humanOutput)
	}
	if observedAt < 0 {
		return fmt.Errorf("the output carries no %q marking:\n%s", grammar.ProvenanceEmpiricalObservation, w.humanOutput)
	}
	g, err := w.grammarPayload()
	if err != nil {
		return err
	}
	// Separation, not just presence: every contract-derived type must render above
	// the empirical marking, and every fact below it. A type listed under the
	// residue's heading would read as an observation.
	for _, ct := range g.ChangeTypes {
		if at := strings.Index(w.humanOutput, ct.Type+"  ["); at >= 0 && at > observedAt {
			return fmt.Errorf("change type %q renders below the empirical marking, so it reads as an observation:\n%s", ct.Type, w.humanOutput)
		}
	}
	for _, f := range g.Facts {
		at := strings.Index(w.humanOutput, f.ID)
		if at < 0 || at < observedAt {
			return fmt.Errorf("fact %q does not render under the empirical marking:\n%s", f.ID, w.humanOutput)
		}
	}
	return nil
}

func (w *grammarRefWorld) thenEveryTypeCondensedByPlacement() error {
	if err := w.thenEveryTypeCarriesItsPlacement(); err != nil {
		return err
	}
	g, err := w.grammarPayload()
	if err != nil {
		return err
	}
	// Condensed: the whole vocabulary rides on one line per placement class, not
	// one line per type. Counting the distinct classes derives the bound from the
	// data rather than pinning a literal.
	classes := map[string]bool{}
	for _, ct := range g.ChangeTypes {
		classes[ct.Placement] = true
	}
	vocabularyLines := 0
	for _, l := range w.grammarOutputLines() {
		for class := range classes {
			if strings.HasPrefix(strings.TrimSpace(l), class) {
				vocabularyLines++
				break
			}
		}
	}
	if vocabularyLines != len(classes) {
		return fmt.Errorf("the compact vocabulary must be one line per placement class (%d expected); found %d:\n%s", len(classes), vocabularyLines, w.humanOutput)
	}
	return nil
}

func (w *grammarRefWorld) thenEachFactOnOneLine() error {
	if err := w.requireRendered(); err != nil {
		return err
	}
	g, err := w.grammarPayload()
	if err != nil {
		return err
	}
	if len(g.Facts) == 0 {
		return fmt.Errorf("the payload carries no facts to render")
	}
	for _, f := range g.Facts {
		var matched []string
		for _, l := range w.grammarOutputLines() {
			if strings.Contains(l, f.ID) {
				matched = append(matched, l)
			}
		}
		if len(matched) != 1 {
			return fmt.Errorf("fact %s must render as exactly one line; found %d:\n%s", f.ID, len(matched), w.humanOutput)
		}
		for label, value := range map[string]string{"disposition": f.Disposition, "title": f.Title} {
			if !strings.Contains(matched[0], value) {
				return fmt.Errorf("fact %s's one line does not carry its %s (%q): %q", f.ID, label, value, matched[0])
			}
		}
	}
	return nil
}

func (w *grammarRefWorld) thenVocabularyStillRendersInFull() error {
	return w.thenEveryTypeCarriesItsPlacement()
}

func (w *grammarRefWorld) thenStatesNoEmpiricalResidue() error {
	if err := w.requireRendered(); err != nil {
		return err
	}
	const statement = "no empirical residue is currently recorded"
	if !strings.Contains(w.humanOutput, statement) {
		return fmt.Errorf("the output does not state %q:\n%s", statement, w.humanOutput)
	}
	return nil
}

// grammarLineWith reports whether any single line carries every one of the given
// substrings — the "these facts belong together" check a pair of independent
// Contains calls cannot make.
func grammarLineWith(lines []string, substrings ...string) bool {
	for _, l := range lines {
		all := true
		for _, s := range substrings {
			if !strings.Contains(l, s) {
				all = false
				break
			}
		}
		if all {
			return true
		}
	}
	return false
}
