package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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

	// The command scenarios: the seam the run drives (its transport is a tripwire —
	// no scenario here may reach it), the invocation, and what the run produced.
	seam       *fakeProposalSeam
	invocation string
	outcome    Outcome
	outcomeSet bool
	exitCode   int
	stdout     string
	stderr     string
	outputs    []string // stdout per run, for the repeatability comparison

	// The fact a structured-output scenario read back.
	readFact    grammar.Fact
	readFactSet bool

	// The write-gate scenario: armed by its Given, evaluated by the run.
	gateInstalled bool
	gateRan       bool
	gateDecision  string
	gateMessage   string

	// The retirement scenario's rewritten record.
	editedRecord    string
	editedRecordSet bool
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
	sc.Step(`^every change type will appear with its placement class$`, w.thenEveryTypeCarriesItsPlacement)
	sc.Step(`^the nesting rule will be stated once with its wrapper types$`, w.thenNestingRuleStatedOnce)
	sc.Step(`^each fact will carry its title, shape, disposition, and symptom$`, w.thenEachFactCarriesEveryField)
	sc.Step(`^the contract-derived content will be visibly separated from the empirical observations$`, w.thenProvenanceVisiblySeparated)
	sc.Step(`^every change type will appear with its placement class in condensed form$`, w.thenEveryTypeCondensedByPlacement)
	sc.Step(`^each fact will appear as one line carrying its id, disposition, and title$`, w.thenEachFactOnOneLine)
	sc.Step(`^the contract-derived change-type vocabulary will still render in full$`, w.thenVocabularyStillRendersInFull)
	sc.Step(`^the output will state that no empirical residue is currently recorded$`, w.thenStatesNoEmpiricalResidue)

	// --- The command and its conduct (T005) ---
	sc.Step(`^an AI agent was about to assemble a change set$`, w.givenNothingStaged)
	sc.Step(`^the implemented "glassfrog proposal grammar" command$`, w.givenNothingStaged)
	sc.Step(`^a machine with the CLI installed and no credential configured$`, w.givenNoCredentialConfigured)
	sc.Step(`^the operating surface's write gate was installed$`, w.givenWriteGateInstalled)
	sc.Step(`^a successful "glassfrog ([^"]*)" run$`, w.givenSuccessfulRun)
	sc.Step(`^(?:a practitioner|an agent|it) runs "glassfrog (.+)"$`, w.whenRunsInvocation)
	sc.Step(`^the same binary runs the same command again$`, w.whenRunsTheSameCommandAgain)
	sc.Step(`^a consumer reads the rendered fact "([^"]*)"$`, w.whenConsumerReadsFact)
	sc.Step(`^a consumer reads the rendered structure$`, w.whenConsumerReadsStructure)
	sc.Step(`^the "change_types" array will carry every change type the vendored contract enumerates, each with its placement$`, w.thenStructuredVocabularyMatchesTheContract)
	sc.Step(`^the "facts" array will carry "([^"]*)" and "([^"]*)" with the symptom each produces$`, w.thenStructuredFactsCarrySymptoms)
	sc.Step(`^no API request will have been made to learn either$`, w.thenNoAPIRequestMade)
	sc.Step(`^no API request will be attempted$`, w.thenNoAPIRequestMade)
	sc.Step(`^the run will fail as a usage error with exit code (\d+)$`, w.thenFailsAsUsageErrorWithCode)
	sc.Step(`^the failure will express no verdict on the change set's validity$`, w.thenFailureExpressesNoVerdict)
	sc.Step(`^its disposition will read "([^"]*)" rather than "([^"]*)"$`, w.thenFactDispositionReads)
	sc.Step(`^its symptom will state that a returned prp_ id is a dead draft, not a successful change$`, w.thenFactSymptomStatesDeadDraft)
	sc.Step(`^every "change_types" entry will carry the provenance token "([^"]*)"$`, w.thenEveryChangeTypeCarriesToken)
	sc.Step(`^every "facts" entry will carry the provenance token "([^"]*)"$`, w.thenEveryFactCarriesToken)
	sc.Step(`^the full reference will render successfully$`, w.thenFullReferenceRendered)
	sc.Step(`^the gate will pass the command ungated as a recognized proposal read$`, w.thenGatePassesUngated)
	sc.Step(`^no confirmation will be asked$`, w.thenNoConfirmationAsked)
	sc.Step(`^the structured output will be identical across the two runs$`, w.thenStructuredOutputIdenticalAcrossRuns)

	// --- The retired-fact edge (T005, exercising T001's regeneration) ---
	sc.Step(`^the fact "([^"]*)" retired from the record together with its manifest entry$`, w.givenFactRetiredFromTheRecord)
	sc.Step(`^the grammar artifact was regenerated$`, w.givenArtifactRegenerated)
	sc.Step(`^the reference next renders$`, w.whenReferenceNextRenders)
	sc.Step(`^"([^"]*)" will no longer appear among the rendered facts$`, w.thenFactNoLongerRendered)
	sc.Step(`^the "([^"]*)" type will remain in the contract-derived vocabulary$`, w.thenTypeRemainsInVocabulary)
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

// --- The command and its conduct (T005) -------------------------------------

// givenNothingstaged is the no-op premise a few scenarios open with: the command
// is the landed one and nothing needs staging, because it reads no credential, no
// file, and no network.
func (w *grammarRefWorld) givenNothingStaged() error { return nil }

// givenNoCredentialConfigured stages the credential-free machine: the connection
// context carries no usable token and the transport is a tripwire that records any
// use. Neither can affect the outcome — the command never assembles a connection —
// which is exactly what the Thens assert.
func (w *grammarRefWorld) givenNoCredentialConfigured() error {
	w.seam = &fakeProposalSeam{fakeMeSeam: &fakeMeSeam{
		ctx:       noTokenContext(),
		transport: &cannedTransport{status: 500, body: `{"detail":"the grammar read must never reach a transport"}`},
	}}
	return nil
}

// givenWriteGateInstalled asserts the shipped gate script is present, and arms the
// run step to evaluate it alongside the command — modelling an agent running the
// command on a machine where the operating surface's gate is installed.
func (w *grammarRefWorld) givenWriteGateInstalled() error {
	if _, err := build.ReadGateScript(); err != nil {
		return fmt.Errorf("the write gate script is not readable: %w", err)
	}
	if _, err := exec.LookPath("bash"); err != nil {
		return fmt.Errorf("bash is not on PATH — the gate is a bash hook: %w", err)
	}
	w.gateInstalled = true
	return nil
}

// grammarSeam returns the seam the run uses, defaulting to one whose transport is a
// tripwire: every scenario in this suite must reach exit 0 (or a usage error)
// WITHOUT a request, so a default that would answer a request could hide a
// regression that started making one.
func (w *grammarRefWorld) grammarSeam() *fakeProposalSeam {
	if w.seam == nil {
		w.seam = &fakeProposalSeam{fakeMeSeam: &fakeMeSeam{
			ctx:       validMeContext(),
			transport: &cannedTransport{status: 500, body: `{"detail":"the grammar read must never reach a transport"}`},
		}}
	}
	return w.seam
}

// whenRunsInvocation dispatches the invocation through a real root with the real
// `proposal` group attached — the landed leaf, its cobra argument validation, the
// output-format resolution, and the render dispatch, end to end. Nothing is stubbed
// but the seam, which exists so the run never touches the real ~/.glassfrogrc or a
// real socket.
func (w *grammarRefWorld) whenRunsInvocation(invocation string) error {
	root := NewRootCommand()
	seam := w.grammarSeam()
	MustRegister(root, newProposalCommand(seam))

	var out, errb bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&errb)
	outcome, _ := Run(root, strings.Fields(invocation))

	w.invocation = invocation
	w.outcome, w.outcomeSet = outcome, true
	w.exitCode = ExitCode(outcome)
	w.stdout, w.stderr = out.String(), errb.String()
	w.outputs = append(w.outputs, w.stdout)
	// The human-format Thens (T004) read the rendered text, so a human invocation
	// feeds them the command's own stdout rather than a re-render.
	if !strings.Contains(invocation, "json") && !strings.Contains(invocation, "yaml") {
		w.humanOutput, w.rendered = w.stdout, true
	}

	if w.gateInstalled {
		decision, message, err := grammarRunGateScript("glassfrog " + invocation)
		if err != nil {
			return fmt.Errorf("running the write gate on %q: %w", invocation, err)
		}
		w.gateDecision, w.gateMessage, w.gateRan = decision, message, true
	}
	return nil
}

// givenSuccessfulRun is the same run, with success asserted as the premise — the
// scenarios that read the structured document open with it.
func (w *grammarRefWorld) givenSuccessfulRun(invocation string) error {
	if err := w.whenRunsInvocation(invocation); err != nil {
		return err
	}
	if w.exitCode != 0 {
		return fmt.Errorf("the run %q was not successful: exit %d, stderr %q", invocation, w.exitCode, w.stderr)
	}
	return nil
}

func (w *grammarRefWorld) whenRunsTheSameCommandAgain() error {
	if w.invocation == "" {
		return fmt.Errorf("no earlier invocation to repeat")
	}
	return w.whenRunsInvocation(w.invocation)
}

// structured decodes the run's stdout as the rendered grammar structure, asserting
// the accord's top-level shape: exactly the two keys, both present.
func (w *grammarRefWorld) structured() (grammar.Grammar, error) {
	if !w.outcomeSet {
		return grammar.Grammar{}, fmt.Errorf("no invocation has run")
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal([]byte(w.stdout), &raw); err != nil {
		return grammar.Grammar{}, fmt.Errorf("the structured output is not a JSON object (%w): %s", err, w.stdout)
	}
	for _, key := range []string{"change_types", "facts"} {
		if _, ok := raw[key]; !ok {
			return grammar.Grammar{}, fmt.Errorf("the structured output omits the %q key: %s", key, w.stdout)
		}
	}
	var g grammar.Grammar
	if err := json.Unmarshal([]byte(w.stdout), &g); err != nil {
		return grammar.Grammar{}, fmt.Errorf("the structured output does not decode as the grammar structure (%w): %s", err, w.stdout)
	}
	return g, nil
}

func (w *grammarRefWorld) whenConsumerReadsStructure() error {
	_, err := w.structured()
	return err
}

func (w *grammarRefWorld) whenConsumerReadsFact(id string) error {
	g, err := w.structured()
	if err != nil {
		return err
	}
	for _, f := range g.Facts {
		if f.ID == id {
			w.readFact, w.readFactSet = f, true
			return nil
		}
	}
	return fmt.Errorf("the structured output carries no fact %q; it carries %v", id, grammarFactIDs(g))
}

// thenStructuredVocabularyMatchesTheContract set-compares the rendered vocabulary
// against the vendored contract's enum, derived from the contract at assertion time
// — the rendered side must not be checked against itself, and a hard-coded expected
// list would become a third source of truth.
func (w *grammarRefWorld) thenStructuredVocabularyMatchesTheContract() error {
	g, err := w.structured()
	if err != nil {
		return err
	}
	enum, nestedOnly, _, err := grammarContractSides()
	if err != nil {
		return err
	}
	rendered := map[string]string{}
	for _, ct := range g.ChangeTypes {
		if ct.Placement == "" {
			return fmt.Errorf("rendered change type %q carries no placement", ct.Type)
		}
		rendered[ct.Type] = ct.Placement
	}
	for _, t := range enum {
		if _, ok := rendered[t]; !ok {
			return fmt.Errorf("the contract enumerates %q but the rendered vocabulary omits it", t)
		}
	}
	for t := range rendered {
		if !grammarContains(enum, t) {
			return fmt.Errorf("the rendered vocabulary carries %q, which the contract does not enumerate", t)
		}
	}
	// Each type's placement must be the one the contract's own rule implies.
	for _, t := range enum {
		want := grammar.PlacementTopLevel
		if grammarContains(nestedOnly, t) {
			want = grammar.PlacementNestedOnly
		}
		if rendered[t] != want {
			return fmt.Errorf("rendered change type %q carries placement %q, want %q per the contract's nested-only rule", t, rendered[t], want)
		}
	}
	return nil
}

func (w *grammarRefWorld) thenStructuredFactsCarrySymptoms(first, second string) error {
	g, err := w.structured()
	if err != nil {
		return err
	}
	for _, want := range []string{first, second} {
		found := false
		for _, f := range g.Facts {
			if f.ID != want {
				continue
			}
			found = true
			if strings.TrimSpace(f.Symptom) == "" {
				return fmt.Errorf("fact %q carries no symptom", want)
			}
		}
		if !found {
			return fmt.Errorf("the rendered residue omits fact %q; it carries %v", want, grammarFactIDs(g))
		}
	}
	return nil
}

// thenNoAPIRequestMade asserts the client-less conduct from three independent
// angles: the connection was never assembled, no client was ever built, and the
// tripwire transport was never used. Any one of them alone could pass while the
// command had started making a request some other way.
func (w *grammarRefWorld) thenNoAPIRequestMade() error {
	seam := w.grammarSeam()
	if seam.assembleCalled {
		return fmt.Errorf("the command assembled a connection context — it must resolve no base URL and no token")
	}
	if seam.newClientCalled {
		return fmt.Errorf("the command built an API client — the grammar read is client-less")
	}
	tr, ok := seam.transport.(*cannedTransport)
	if !ok {
		return fmt.Errorf("this scenario's transport does not record calls")
	}
	if tr.calls != 0 {
		return fmt.Errorf("the command sent %d request(s); the grammar read must send none", tr.calls)
	}
	return nil
}

func (w *grammarRefWorld) thenFailsAsUsageErrorWithCode(code int) error {
	if !w.outcomeSet {
		return fmt.Errorf("no invocation has run")
	}
	if w.outcome != UsageError {
		return fmt.Errorf("the run produced outcome %v, want a usage error", w.outcome)
	}
	if w.exitCode != code {
		return fmt.Errorf("the run exited %d, want %d", w.exitCode, code)
	}
	return nil
}

// thenFailureExpressesNoVerdict is the accord's refusal boundary: a change set
// offered for checking is refused because there is no input path, NOT because the
// CLI judged it. Any validity vocabulary in the failure would be exactly the
// verdict the command must never express.
func (w *grammarRefWorld) thenFailureExpressesNoVerdict() error {
	combined := strings.ToLower(w.stdout + w.stderr)
	for _, verdict := range []string{"invalid", "valid", "malformed", "rejected", "not accepted", "well-formed", "schema"} {
		if strings.Contains(combined, verdict) {
			return fmt.Errorf("the usage failure uses the validity vocabulary %q, which reads as a verdict on the change set:\nstdout: %s\nstderr: %s", verdict, w.stdout, w.stderr)
		}
	}
	if strings.TrimSpace(w.stderr) == "" {
		return fmt.Errorf("the usage failure said nothing on stderr")
	}
	return nil
}

func (w *grammarRefWorld) thenFactDispositionReads(want, notWant string) error {
	if !w.readFactSet {
		return fmt.Errorf("no fact was read")
	}
	if w.readFact.Disposition != want {
		return fmt.Errorf("fact %s carries disposition %q, want %q (and not %q)", w.readFact.ID, w.readFact.Disposition, want, notWant)
	}
	return nil
}

func (w *grammarRefWorld) thenFactSymptomStatesDeadDraft() error {
	if !w.readFactSet {
		return fmt.Errorf("no fact was read")
	}
	symptom := grammarCollapse(w.readFact.Symptom)
	for _, want := range []string{"prp_", "not", "successful governance change"} {
		if !strings.Contains(symptom, want) {
			return fmt.Errorf("fact %s's symptom does not state that a returned prp_ id is not a successful change (missing %q): %q", w.readFact.ID, want, symptom)
		}
	}
	return nil
}

func (w *grammarRefWorld) thenEveryChangeTypeCarriesToken(token string) error {
	g, err := w.structured()
	if err != nil {
		return err
	}
	if len(g.ChangeTypes) == 0 {
		return fmt.Errorf("the rendered vocabulary is empty, so no entry carries a token")
	}
	for _, ct := range g.ChangeTypes {
		if ct.Provenance != token {
			return fmt.Errorf("change type %q carries provenance %q, want %q", ct.Type, ct.Provenance, token)
		}
	}
	return nil
}

func (w *grammarRefWorld) thenEveryFactCarriesToken(token string) error {
	g, err := w.structured()
	if err != nil {
		return err
	}
	if len(g.Facts) == 0 {
		return fmt.Errorf("the rendered residue is empty, so no entry carries a token")
	}
	for _, f := range g.Facts {
		if f.Provenance != token {
			return fmt.Errorf("fact %q carries provenance %q, want %q", f.ID, f.Provenance, token)
		}
	}
	return nil
}

// thenFullReferenceRendered asserts the whole reference arrived — both halves, not
// merely a zero exit with empty output.
func (w *grammarRefWorld) thenFullReferenceRendered() error {
	if w.exitCode != 0 {
		return fmt.Errorf("the run exited %d, want 0; stderr: %s", w.exitCode, w.stderr)
	}
	if err := w.thenEveryTypeCarriesItsPlacement(); err != nil {
		return err
	}
	return w.thenEachFactCarriesEveryField()
}

func (w *grammarRefWorld) thenGatePassesUngated() error {
	if !w.gateRan {
		return fmt.Errorf("the gate was never evaluated")
	}
	if w.gateDecision != "" {
		return fmt.Errorf("the gate returned permissionDecision %q for a read (%s); it must pass through: %s", w.gateDecision, w.invocation, w.gateMessage)
	}
	// Pass-through alone is not enough: an unrecognized subcommand fails CLOSED to
	// `ask`, so a silent pass-through would mean the leaf was recognized. Assert the
	// recognition is deliberate — the leaf is in the script's read set.
	script, err := build.ReadGateScript()
	if err != nil {
		return err
	}
	reads, ok := grammarGateReadSet(script)
	if !ok {
		return fmt.Errorf("could not read PROPOSAL_READS out of the gate script")
	}
	if !grammarContains(reads, "grammar") {
		return fmt.Errorf("the gate script's recognized-read set is %v — it does not list `grammar`, so the pass-through is accidental rather than deliberate", reads)
	}
	return nil
}

func (w *grammarRefWorld) thenNoConfirmationAsked() error {
	if !w.gateRan {
		return fmt.Errorf("the gate was never evaluated")
	}
	if w.gateDecision == "ask" || strings.TrimSpace(w.gateMessage) != "" {
		return fmt.Errorf("the gate asked for confirmation on a read: decision %q, message %q", w.gateDecision, w.gateMessage)
	}
	return nil
}

func (w *grammarRefWorld) thenStructuredOutputIdenticalAcrossRuns() error {
	if len(w.outputs) < 2 {
		return fmt.Errorf("only %d invocation(s) ran; the comparison needs two", len(w.outputs))
	}
	first, second := w.outputs[len(w.outputs)-2], w.outputs[len(w.outputs)-1]
	if first != second {
		return fmt.Errorf("two runs of the same command produced different output:\nfirst:  %q\nsecond: %q", first, second)
	}
	if strings.TrimSpace(first) == "" {
		return fmt.Errorf("both runs produced empty output, which would make them trivially identical")
	}
	return nil
}

// --- The retired-fact edge (T005, exercising T001's regeneration) -------------

// givenFactRetiredFromTheRecord rewrites the record in memory the way a retirement
// rewrites it on disk: the fact's section is deleted AND its id leaves the
// Live-facts manifest. Both halves matter — 072's guard rejects either one alone.
func (w *grammarRefWorld) givenFactRetiredFromTheRecord(id string) error {
	recordRaw, err := build.ReadGrammarFactsRecord()
	if err != nil {
		return fmt.Errorf("reading the grammar record: %w", err)
	}
	if !strings.Contains(recordRaw, "## "+id) {
		return fmt.Errorf("the record carries no %q section to retire", id)
	}
	edited, err := grammarRetireFact(recordRaw, id)
	if err != nil {
		return err
	}
	if strings.Contains(edited, "## "+id) {
		return fmt.Errorf("the %q section survived the retirement edit", id)
	}
	w.editedRecord, w.editedRecordSet = edited, true
	return nil
}

// givenArtifactRegenerated re-derives the artifact from the UNCHANGED contract and
// the rewritten record, through the same derivation the generator and the drift
// guard use. That is what makes this scenario an exercise of the regeneration step
// rather than a hand-built payload.
func (w *grammarRefWorld) givenArtifactRegenerated() error {
	if !w.editedRecordSet {
		return fmt.Errorf("no record edit was staged, so there is nothing to regenerate from")
	}
	specRaw, err := grammarContractBytes()
	if err != nil {
		return err
	}
	artifact, err := build.BuildGrammarFromSources(specRaw, w.editedRecord)
	if err != nil {
		return fmt.Errorf("regenerating the artifact from the rewritten record: %w", err)
	}
	w.payload, w.payloadSet = artifact.Grammar, true
	return nil
}

func (w *grammarRefWorld) whenReferenceNextRenders() error {
	return w.renderIn(humanFormat(output.DefaultFormat))
}

func (w *grammarRefWorld) thenFactNoLongerRendered(id string) error {
	g, err := w.grammarPayload()
	if err != nil {
		return err
	}
	for _, f := range g.Facts {
		if f.ID == id {
			return fmt.Errorf("fact %q is still in the rendered residue", id)
		}
	}
	if err := w.requireRendered(); err != nil {
		return err
	}
	if strings.Contains(w.humanOutput, id) {
		return fmt.Errorf("fact %q still appears in the rendering:\n%s", id, w.humanOutput)
	}
	return nil
}

func (w *grammarRefWorld) thenTypeRemainsInVocabulary(changeType string) error {
	g, err := w.grammarPayload()
	if err != nil {
		return err
	}
	for _, ct := range g.ChangeTypes {
		if ct.Type != changeType {
			continue
		}
		if err := w.requireRendered(); err != nil {
			return err
		}
		if !strings.Contains(w.humanOutput, changeType) {
			return fmt.Errorf("type %q is in the payload but not in the rendering:\n%s", changeType, w.humanOutput)
		}
		return nil
	}
	return fmt.Errorf("type %q left the contract-derived vocabulary when a fact retired", changeType)
}

// --- Suite helpers ----------------------------------------------------------

// grammarRetireFact removes a fact's section and its manifest entry from the
// record's text, modelling the retirement edit a human makes.
func grammarRetireFact(recordRaw, id string) (string, error) {
	start := strings.Index(recordRaw, "## "+id)
	if start < 0 {
		return "", fmt.Errorf("no %q section in the record", id)
	}
	end := len(recordRaw)
	if next := strings.Index(recordRaw[start+1:], "\n## "); next >= 0 {
		end = start + 1 + next + 1
	}
	edited := recordRaw[:start] + recordRaw[end:]

	// Drop the id from the Live-facts manifest line, leaving the remaining ids.
	const manifestLabel = "**Live facts**:"
	at := strings.Index(edited, manifestLabel)
	if at < 0 {
		return "", fmt.Errorf("the record has no Live-facts manifest line")
	}
	lineEnd := strings.Index(edited[at:], "\n")
	if lineEnd < 0 {
		lineEnd = len(edited) - at
	}
	line := edited[at : at+lineEnd]
	var kept []string
	for _, part := range strings.Split(strings.TrimPrefix(line, manifestLabel), ",") {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" && trimmed != id {
			kept = append(kept, trimmed)
		}
	}
	return edited[:at] + manifestLabel + " " + strings.Join(kept, ", ") + edited[at+lineEnd:], nil
}

// grammarContractBytes reads the vendored contract from the repository root — the
// unchanged side of the retirement scenario.
func grammarContractBytes() ([]byte, error) {
	root, err := build.RepoRoot()
	if err != nil {
		return nil, err
	}
	return os.ReadFile(filepath.Join(root, filepath.FromSlash(build.VendoredSpecPath)))
}

// grammarContractSides derives the contract's enum and nested-only set at
// assertion time, so a scenario compares the rendering against the contract rather
// than against a checked-in expectation.
func grammarContractSides() (enum, nestedOnly []string, description string, err error) {
	raw, err := grammarContractBytes()
	if err != nil {
		return nil, nil, "", err
	}
	return build.ParseSpecChangeTypes(raw)
}

// grammarGateReadSet extracts the gate script's recognized-read leaves from its
// PROPOSAL_READS assignment, so the gate scenario reads the script's own set rather
// than restating it.
func grammarGateReadSet(script string) ([]string, bool) {
	const label = "PROPOSAL_READS="
	at := strings.Index(script, label)
	if at < 0 {
		return nil, false
	}
	rest := script[at+len(label):]
	open := strings.Index(rest, `"`)
	if open < 0 {
		return nil, false
	}
	rest = rest[open+1:]
	close := strings.Index(rest, `"`)
	if close < 0 {
		return nil, false
	}
	return strings.Fields(rest[:close]), true
}

// grammarRunGateScript feeds the real gate script a Bash tool call carrying command
// and returns the permission decision ("" when the gate passes through) plus the
// message the host would show.
func grammarRunGateScript(command string) (decision, message string, err error) {
	root, err := build.RepoRoot()
	if err != nil {
		return "", "", err
	}
	script := filepath.Join(root, filepath.FromSlash(build.GateScriptPath))
	raw, err := json.Marshal(struct {
		ToolName  string `json:"tool_name"`
		ToolInput struct {
			Command string `json:"command"`
		} `json:"tool_input"`
	}{ToolName: "Bash", ToolInput: struct {
		Command string `json:"command"`
	}{Command: command}})
	if err != nil {
		return "", "", err
	}
	cmd := exec.Command("bash", script)
	cmd.Stdin = bytes.NewReader(raw)
	var out, errBuf bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errBuf
	if runErr := cmd.Run(); runErr != nil {
		return "", "", fmt.Errorf("gate script failed: %v (stderr: %s)", runErr, errBuf.String())
	}
	trimmed := strings.TrimSpace(out.String())
	if trimmed == "" {
		return "", "", nil // pass-through
	}
	var g struct {
		HookSpecificOutput struct {
			PermissionDecision       string `json:"permissionDecision"`
			PermissionDecisionReason string `json:"permissionDecisionReason"`
		} `json:"hookSpecificOutput"`
		SystemMessage string `json:"systemMessage"`
	}
	if uerr := json.Unmarshal([]byte(trimmed), &g); uerr != nil {
		return "", "", fmt.Errorf("gate emitted non-JSON output %q: %w", trimmed, uerr)
	}
	msg := g.SystemMessage
	if msg == "" {
		msg = g.HookSpecificOutput.PermissionDecisionReason
	}
	return g.HookSpecificOutput.PermissionDecision, msg, nil
}

func grammarContains(set []string, want string) bool {
	for _, s := range set {
		if s == want {
			return true
		}
	}
	return false
}

// grammarCollapse drops markdown emphasis and collapses whitespace, so a phrase
// reads the same whether or not the record's author bolded a word inside it.
func grammarCollapse(s string) string {
	s = strings.NewReplacer("*", "", "`", "").Replace(s)
	return strings.Join(strings.Fields(s), " ")
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
