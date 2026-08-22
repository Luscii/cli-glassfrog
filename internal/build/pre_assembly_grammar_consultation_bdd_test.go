package build

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/cucumber/godog"
)

// TestPreAssemblyGrammarConsultationFeatures runs the executable acceptance
// for the pre-assembly gate (079, pre-assembly-grammar-consultation.feature).
// Like the sibling build-side suites its Paths name ONLY this feature file and
// it runs with the ~@wip filter; the four @validation scenarios stay @wip for
// /score:validate.
//
// The deliverable is an in-place widening of the drafting path's declarative
// artifacts (the skill's nine-step gate workflow, the drafter's eight-leaf
// fence + consultation element, the registry's consultation-read line and
// rewritten routing annotation), so the scenarios assert against artifact
// content: phrase assertions read a whitespace-collapsed, marker-stripped copy
// (grammarNorm), structural checks (frontmatter, step order, registry
// membership) read the raw or parsed forms. The dead-shape scenarios ground
// their premises in the grammar-facts record itself (CSG ids read from the
// record, never hard-coded content), because the artifacts consult the
// rendering and never restate a shape.
func TestPreAssemblyGrammarConsultationFeatures(t *testing.T) {
	w := &preAssemblyGateWorld{}
	suite := godog.TestSuite{
		ScenarioInitializer: func(sc *godog.ScenarioContext) { w.registerConsultation(sc) },
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"../../features/unguided-change-construction/pre-assembly-grammar-consultation.feature"},
			Tags:     "~@wip",
			TestingT: t,
		},
	}
	if suite.Run() != 0 {
		t.Fatal("non-zero status: pre-assembly-grammar-consultation feature scenarios failed")
	}
}

// preAssemblyGateWorld is the per-scenario state shared by the two 079 suites
// (this file and the routing-application suite): the loaded artifacts in
// normalized and raw form, the parsed workflow steps, the registries, and the
// two consulted records.
type preAssemblyGateWorld struct {
	skill       string // grammarNorm'd for phrase assertions
	agent       string
	skillRaw    string
	agentRaw    string
	steps       []DraftingWorkflowStep
	composed    []string
	gated       []string
	registryRaw string
	facts       GrammarFactsRecord
	routing     CircleRoutingRuleRecord
}

func (w *preAssemblyGateWorld) registerConsultation(sc *godog.ScenarioContext) {
	sc.Before(func(ctx context.Context, _ *godog.Scenario) (context.Context, error) {
		*w = preAssemblyGateWorld{}
		return ctx, nil
	})

	// Scenario: The workflow orders the gate ahead of the gated create
	sc.Step(`^the proposal-drafting skill's workflow was read end to end$`, w.givenWorkflowRead)
	sc.Step(`^its steps are compared against the gate's order$`, w.whenStepsCompared)
	sc.Step(`^the routing determination will precede the grammar consult, the consult will precede assembly, and the match against recorded dead shapes will follow assembly with the routing answer in hand$`, w.thenGateOrderHolds)
	sc.Step(`^every gate step will precede the confirmed create$`, w.thenGatePrecedesCreate)

	// Scenario: A recognized dead shape is named before the write
	sc.Step(`^an assembled change set that matched the recorded dead shape CSG-1's refused wrapper form$`, w.givenMatchedFact("CSG-1", "wrapper"))
	sc.Step(`^the drafter reaches the gated create$`, w.whenLoadedArtifacts)
	sc.Step(`^it will surface the fact's handle, shape, and symptom before the write$`, w.thenSurfacesHandleShapeSymptom)
	sc.Step(`^it will return action surfaced-dead-shape awaiting the practitioner's direction$`, w.thenReturnsSurfacedDeadShape)
	sc.Step(`^it will express no verdict on the change set's validity$`, w.thenNoVerdict)

	// Scenario: A failed grammar read is recorded and drafting continues
	sc.Step(`^a grammar read that failed$`, w.givenFailedGrammarRead)
	sc.Step(`^the drafter continues$`, w.whenLoadedArtifacts)
	sc.Step(`^the consultation element will record that the grammar was not consulted, naming the failure$`, w.thenRecordsNotConsulted)
	sc.Step(`^drafting will continue rather than being withheld$`, w.thenDraftingNotWithheld)
	sc.Step(`^assembly will not be presented as consulted$`, w.thenAssemblyNotPresentedConsulted)

	// Scenario: The self-targeting shape is recognized with the routing answer in hand
	sc.Step(`^a change set whose role operation targeted the circle the proposal would be anchored in$`, w.givenMatchedFact("CSG-2", "self-targeting"))
	sc.Step(`^the drafter matches the assembled set against the recorded dead shapes$`, w.whenMatchStepReached)
	sc.Step(`^it will recognize the self-targeting shape recorded as CSG-2 and name its symptom before the write$`, w.thenRecognizesSelfTargeting)
	sc.Step(`^the recognition will rest on both the change's target and the circle the proposal would be anchored in$`, w.thenRecognitionRestsOnBoth)

	// Scenario: Proceeding past a surfaced dead shape runs the create unchanged
	sc.Step(`^a recognized dead shape surfaced with action surfaced-dead-shape$`, w.givenSurfacedDeadShape)
	sc.Step(`^the practitioner directs the drafter to proceed past that fact$`, w.whenProceedPastDirected)
	sc.Step(`^the re-delegated run will act on the direction and run the create through the confirmed write flow unchanged$`, w.thenActsAndCreatesUnchanged)
	sc.Step(`^the change set will not be altered$`, w.thenChangeSetNotAltered)

	// Scenario: A change set matching nothing recorded implies nothing about validity
	sc.Step(`^an assembled change set matching no recorded dead shape$`, w.givenLoadedArtifacts)
	sc.Step(`^the drafter reaches the write$`, w.whenLoadedArtifacts)
	sc.Step(`^the consultation element will state that no recorded shape matched$`, w.thenStatesNoShapeMatched)
	sc.Step(`^nothing about the set's validity will be implied$`, w.thenNothingImplied)

	// Scenario: A re-delegation carrying direction does not re-surface the same decision
	sc.Step(`^a re-delegation whose input carried the settled anchor and the proceed-past instruction naming the surfaced fact$`, w.givenReDelegationWithDirection)
	sc.Step(`^the drafter runs the gate from the top$`, w.whenGateFromTheTop)
	sc.Step(`^it will act on the direction rather than returning the same decision again$`, w.thenActsNotReAsks)

	// Scenario: The consultation read joins the composed surface as an ungated read
	sc.Step(`^the composed-leaf registry and the write gate's registries$`, w.givenRegistries)
	sc.Step(`^the eight composed leaves are checked against the gate's membership$`, w.whenEightLeavesChecked)
	sc.Step(`^proposal grammar will be listed as a composed read and absent from the gated set$`, w.thenGrammarComposedUngated)
	sc.Step(`^proposal create will remain the only composed leaf in the gated set$`, w.thenCreateOnlyGated)
}

// --- Loading ------------------------------------------------------------------

func (w *preAssemblyGateWorld) ensureArtifacts() error {
	if w.skillRaw == "" {
		skill, err := ReadProposalDraftingSkill()
		if err != nil {
			return fmt.Errorf("proposal-drafting skill did not load: %w", err)
		}
		w.skillRaw, w.skill = skill, grammarNorm(skill)
	}
	if w.agentRaw == "" {
		agent, err := ReadProposalDrafterAgent()
		if err != nil {
			return fmt.Errorf("proposal-drafter agent did not load: %w", err)
		}
		w.agentRaw, w.agent = agent, grammarNorm(agent)
	}
	return nil
}

func (w *preAssemblyGateWorld) ensureFacts() error {
	if w.facts.Raw != "" {
		return nil
	}
	raw, err := ReadGrammarFactsRecord()
	if err != nil {
		return fmt.Errorf("grammar-facts record did not load: %w", err)
	}
	w.facts = ParseGrammarFactsRecord(raw)
	return nil
}

func (w *preAssemblyGateWorld) ensureRouting() error {
	if w.routing.Raw != "" {
		return nil
	}
	raw, err := ReadCircleRoutingRuleRecord()
	if err != nil {
		return fmt.Errorf("circle-routing record did not load: %w", err)
	}
	w.routing = ParseCircleRoutingRuleRecord(raw)
	return nil
}

func (w *preAssemblyGateWorld) givenLoadedArtifacts() error { return w.ensureArtifacts() }
func (w *preAssemblyGateWorld) whenLoadedArtifacts() error  { return w.ensureArtifacts() }

// combined returns the normalized skill and agent together — behavioral
// conduct is described across both artifacts.
func (w *preAssemblyGateWorld) combined() string { return w.skill + " " + w.agent }

// fact returns the recorded fact with the given id, or an error naming what
// the record carries instead. The id comes from the scenario's concrete data;
// the fact's content is always read from the record, never hard-coded.
func (w *preAssemblyGateWorld) fact(id string) (GrammarFact, error) {
	if err := w.ensureFacts(); err != nil {
		return GrammarFact{}, err
	}
	for _, f := range w.facts.Facts {
		if f.ID == id {
			return f, nil
		}
	}
	var have []string
	for _, f := range w.facts.Facts {
		have = append(have, f.ID)
	}
	return GrammarFact{}, fmt.Errorf("the grammar-facts record carries no fact %s (has %v)", id, have)
}

// stepIndex returns the position of the named step in the parsed workflow, or
// an error naming what the workflow parses to.
func (w *preAssemblyGateWorld) stepIndex(name string) (int, error) {
	for i, s := range w.steps {
		if s.Name == name {
			return i, nil
		}
	}
	var have []string
	for _, s := range w.steps {
		have = append(have, s.Name)
	}
	return -1, fmt.Errorf("the workflow carries no %q step (steps: %v)", name, have)
}

// requirePhrases asserts every phrase is present in the normalized text,
// failing on the first absence with the label naming which surface was read.
func requirePhrases(label, text string, phrases ...string) error {
	for _, p := range phrases {
		if !containsFold(text, p) {
			return fmt.Errorf("%s does not state %q", label, p)
		}
	}
	return nil
}

// --- Scenario: gate order -------------------------------------------------------

func (w *preAssemblyGateWorld) givenWorkflowRead() error {
	if err := w.ensureArtifacts(); err != nil {
		return err
	}
	w.steps = DraftingWorkflowSteps(w.skillRaw)
	if len(w.steps) == 0 {
		return fmt.Errorf("the skill's workflow section parses to no steps")
	}
	return nil
}

// whenStepsCompared pins the whole enumeration: the workflow's steps are
// exactly the gate's order — the contract anchor DraftingGateOrder — so a
// reordered, dropped, or smuggled-in step fails by name.
func (w *preAssemblyGateWorld) whenStepsCompared() error {
	if len(w.steps) != len(DraftingGateOrder) {
		return fmt.Errorf("the workflow carries %d steps, want the gate's %d", len(w.steps), len(DraftingGateOrder))
	}
	for i, want := range DraftingGateOrder {
		if w.steps[i].Name != want {
			return fmt.Errorf("workflow step %d is %q, want %q", i+1, w.steps[i].Name, want)
		}
	}
	return nil
}

func (w *preAssemblyGateWorld) thenGateOrderHolds() error {
	route, err := w.stepIndex("Route")
	if err != nil {
		return err
	}
	consult, err := w.stepIndex("Consult")
	if err != nil {
		return err
	}
	assemble, err := w.stepIndex("Assemble")
	if err != nil {
		return err
	}
	match, err := w.stepIndex("Match")
	if err != nil {
		return err
	}
	if !(route < consult && consult < assemble && assemble < match) {
		return fmt.Errorf("gate order violated: Route=%d Consult=%d Assemble=%d Match=%d", route, consult, assemble, match)
	}
	if !containsFold(grammarNorm(w.steps[match].Body), "with the routing answer in hand") {
		return fmt.Errorf("the Match step does not run with the routing answer in hand: %q", w.steps[match].Body)
	}
	return nil
}

func (w *preAssemblyGateWorld) thenGatePrecedesCreate() error {
	create, err := w.stepIndex("Confirm & create")
	if err != nil {
		return err
	}
	for i, s := range w.steps {
		if s.Name == "Confirm & create" || s.Name == "Hand off" {
			continue
		}
		if i > create {
			return fmt.Errorf("gate step %q follows the confirmed create", s.Name)
		}
	}
	return nil
}

// --- Scenario: recognized dead shape ---------------------------------------------

// givenMatchedFact grounds a dead-shape premise in the record itself: the
// named fact must exist, carry the anchor word in its heading or Shape, and
// state a non-empty Shape and Symptom — the substance the artifacts promise
// to surface. The artifacts themselves never restate the shape (consulted,
// never copied), so the premise is checked against the record, not the prose.
func (w *preAssemblyGateWorld) givenMatchedFact(id, anchor string) func() error {
	return func() error {
		if err := w.ensureArtifacts(); err != nil {
			return err
		}
		f, err := w.fact(id)
		if err != nil {
			return err
		}
		if !containsFold(f.Title+" "+f.Fields["Shape"], anchor) {
			return fmt.Errorf("fact %s does not describe the %q form: %q", id, anchor, f.Title)
		}
		for _, field := range []string{"Shape", "Symptom"} {
			if strings.TrimSpace(f.Fields[field]) == "" {
				return fmt.Errorf("fact %s carries no %s to surface", id, field)
			}
		}
		return nil
	}
}

func (w *preAssemblyGateWorld) thenSurfacesHandleShapeSymptom() error {
	return requirePhrases("the artifacts", w.combined(),
		"handle", "shape", "symptom", "before the write")
}

// assertActionVocabulary pins the output contract's full action enumeration
// and the consultation element with its three named parts — the load-bearing
// record shape every awaiting-direction return relies on.
func (w *preAssemblyGateWorld) assertActionVocabulary() error {
	for _, action := range []string{
		"created", "surfaced-existing", "declined", "none",
		"surfaced-routing-mismatch", "named-anchors", "surfaced-dead-shape",
	} {
		if !mentionsToken(w.agent, action) {
			return fmt.Errorf("the agent's action vocabulary omits %q", action)
		}
	}
	if !mentionsToken(w.agent, "consultation") {
		return fmt.Errorf("the agent's output contract omits the consultation element")
	}
	for _, part := range []string{"grammar", "routing", "match"} {
		if !mentionsToken(w.agent, part) {
			return fmt.Errorf("the consultation element omits its %q part", part)
		}
	}
	if !containsFold(w.agent, "every") || !containsFold(w.agent, "action path") {
		return fmt.Errorf("the consultation element is not stated as present on every action path")
	}
	return nil
}

func (w *preAssemblyGateWorld) thenReturnsSurfacedDeadShape() error {
	if err := w.assertActionVocabulary(); err != nil {
		return err
	}
	return requirePhrases("the artifacts", w.combined(),
		"surfaced-dead-shape", "awaiting", "direction")
}

func (w *preAssemblyGateWorld) thenNoVerdict() error {
	return requirePhrases("the artifacts", w.combined(), "no verdict", "validity")
}

// --- Scenario: failed grammar read -----------------------------------------------

func (w *preAssemblyGateWorld) givenFailedGrammarRead() error {
	if err := w.ensureArtifacts(); err != nil {
		return err
	}
	if !containsFold(w.agent, "grammar read fails") {
		return fmt.Errorf("the agent carries no defensive entry for a failed grammar read")
	}
	return nil
}

func (w *preAssemblyGateWorld) thenRecordsNotConsulted() error {
	if !mentionsToken(w.agent, "consultation") {
		return fmt.Errorf("the agent's record carries no consultation element")
	}
	return requirePhrases("the agent", w.agent, "not consulted", "naming the failure")
}

func (w *preAssemblyGateWorld) thenDraftingNotWithheld() error {
	return requirePhrases("the artifacts", w.combined(), "never withheld")
}

func (w *preAssemblyGateWorld) thenAssemblyNotPresentedConsulted() error {
	return requirePhrases("the agent", w.agent, "never presented as consulted")
}

// --- Scenario: self-targeting shape ----------------------------------------------

func (w *preAssemblyGateWorld) whenMatchStepReached() error {
	w.steps = DraftingWorkflowSteps(w.skillRaw)
	if _, err := w.stepIndex("Match"); err != nil {
		return err
	}
	return nil
}

// thenRecognizesSelfTargeting asserts the conduct, grounded in the record: the
// fact was verified by the Given (recorded, self-targeting, symptom stated);
// the artifacts promise to surface a recognized fact's handle, shape, and
// symptom before the write — naming CSG-2 in the prose would violate the
// consulted-never-restated posture, so the recognition is asserted as the
// conduct plus the record's substance, not a copied id.
func (w *preAssemblyGateWorld) thenRecognizesSelfTargeting() error {
	return requirePhrases("the artifacts", w.combined(),
		"handle", "symptom", "before the write")
}

func (w *preAssemblyGateWorld) thenRecognitionRestsOnBoth() error {
	return requirePhrases("the skill", w.skill,
		"both the change's target and the circle the proposal would be anchored in")
}

// --- Scenario: proceed past a surfaced dead shape --------------------------------

func (w *preAssemblyGateWorld) givenSurfacedDeadShape() error {
	if err := w.ensureArtifacts(); err != nil {
		return err
	}
	if !mentionsToken(w.agent, "surfaced-dead-shape") {
		return fmt.Errorf("the agent never returns action surfaced-dead-shape")
	}
	return nil
}

func (w *preAssemblyGateWorld) whenProceedPastDirected() error {
	return requirePhrases("the artifacts", w.combined(), "proceed-past instruction")
}

func (w *preAssemblyGateWorld) thenActsAndCreatesUnchanged() error {
	return requirePhrases("the agent", w.agent,
		"acts on it", "confirmed write flow", "unchanged")
}

func (w *preAssemblyGateWorld) thenChangeSetNotAltered() error {
	return requirePhrases("the agent", w.agent, "not altered")
}

// --- Scenario: no recorded shape matches ------------------------------------------

func (w *preAssemblyGateWorld) thenStatesNoShapeMatched() error {
	if !mentionsToken(w.agent, "consultation") {
		return fmt.Errorf("the agent's record carries no consultation element")
	}
	return requirePhrases("the agent", w.agent, "no recorded shape matched")
}

func (w *preAssemblyGateWorld) thenNothingImplied() error {
	return requirePhrases("the artifacts", w.combined(), "implies nothing about", "validity")
}

// --- Scenario: re-delegation with direction ---------------------------------------

func (w *preAssemblyGateWorld) givenReDelegationWithDirection() error {
	if err := w.ensureArtifacts(); err != nil {
		return err
	}
	return requirePhrases("the artifacts", w.combined(),
		"settled anchor", "proceed-past instruction naming the surfaced fact")
}

func (w *preAssemblyGateWorld) whenGateFromTheTop() error {
	return requirePhrases("the artifacts", w.combined(), "from the top")
}

func (w *preAssemblyGateWorld) thenActsNotReAsks() error {
	return requirePhrases("the agent", w.agent,
		"acts on it", "do not re-surface the same decision")
}

// --- Scenario: ungated consultation read ------------------------------------------

// givenRegistries loads the composed-leaf registry (parsed and raw) and 063's
// gated set. It also pins the annotation rewrite the widening promised: no
// sentence in the registry or the agent fence may still claim the routing
// reads sit ahead of their use — the retired claim's fragments must be gone,
// and the surfaces must describe reads a routing step runs. (The held
// @validation scenario re-verifies this at the validate stage; the pin lives
// here so the suite fails the moment the claim regresses.)
func (w *preAssemblyGateWorld) givenRegistries() error {
	if err := w.ensureArtifacts(); err != nil {
		return err
	}
	composed, err := ReadProposalDraftingCommands()
	if err != nil {
		return fmt.Errorf("could not read the composed-leaf registry: %w", err)
	}
	w.composed = composed
	registryRaw, err := ReadProposalDraftingRegistryRaw()
	if err != nil {
		return fmt.Errorf("could not read the registry raw: %w", err)
	}
	w.registryRaw = registryRaw
	gated, err := ReadGatedRegistry()
	if err != nil {
		return fmt.Errorf("could not read the write gate's gated registry: %w", err)
	}
	if len(gated) == 0 {
		return fmt.Errorf("the gated registry lists no leaves — nothing to check membership against")
	}
	w.gated = gated

	for _, surface := range []struct{ label, text string }{
		{"the registry", grammarNorm(registryRaw)},
		{"the agent fence", w.agent},
	} {
		for _, retired := range []string{
			"no workflow step consults",
			"ahead of a workflow that uses them",
			"a later change to this path",
			"not an implied routing step",
			"do not infer a routing step",
		} {
			if containsFold(surface.text, retired) {
				return fmt.Errorf("%s still claims the routing reads are ahead of their use: %q", surface.label, retired)
			}
		}
		if !containsFold(surface.text, "routing step") {
			return fmt.Errorf("%s does not describe the reads as the routing step's reads", surface.label)
		}
	}
	return nil
}

func (w *preAssemblyGateWorld) whenEightLeavesChecked() error {
	if len(w.composed) != 8 {
		return fmt.Errorf("the registry lists %d leaves, want the eight composed leaves: %v", len(w.composed), w.composed)
	}
	return nil
}

func (w *preAssemblyGateWorld) thenGrammarComposedUngated() error {
	composedSet := setOf(w.composed)
	if !composedSet["proposal grammar"] {
		return fmt.Errorf("the registry does not list proposal grammar as a composed read: %v", w.composed)
	}
	if setOf(w.gated)["proposal grammar"] {
		return fmt.Errorf("proposal grammar is a member of the gated set — the consultation would start prompting")
	}
	// The gate script itself agrees: the consultation read passes ungated.
	dec, _, err := runGateScript(`glassfrog proposal grammar`)
	if err != nil {
		return fmt.Errorf("the gate script errored on the consultation read: %v", err)
	}
	if dec == "ask" {
		return fmt.Errorf("the gate script gated proposal grammar (ask) — consultation must never prompt")
	}
	return nil
}

func (w *preAssemblyGateWorld) thenCreateOnlyGated() error {
	gatedSet := setOf(w.gated)
	var gatedComposed []string
	for _, leaf := range w.composed {
		if gatedSet[leaf] {
			gatedComposed = append(gatedComposed, leaf)
		}
	}
	if len(gatedComposed) != 1 || gatedComposed[0] != ProposalDraftingGatedWrite {
		return fmt.Errorf("the gated composed leaves are %v, want exactly [%s]", gatedComposed, ProposalDraftingGatedWrite)
	}
	return nil
}
