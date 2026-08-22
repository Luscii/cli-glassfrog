package build

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/cucumber/godog"
)

// TestPreAssemblyRoutingApplicationFeatures runs the executable acceptance for
// the applied routing (079, pre-assembly-routing-application.feature) — the
// routing determination the drafting workflow's first step now runs. It shares
// preAssemblyGateWorld with the grammar-consultation suite; the routing
// premises are grounded in the circle-routing record itself (named reads and
// the root-circle decline read from the record, never hard-coded), because the
// artifacts name the record and run its reads without restating its rule. Runs
// with the ~@wip filter; the one @validation scenario stays held for
// /score:validate.
func TestPreAssemblyRoutingApplicationFeatures(t *testing.T) {
	w := &preAssemblyGateWorld{}
	suite := godog.TestSuite{
		ScenarioInitializer: func(sc *godog.ScenarioContext) { w.registerRouting(sc) },
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"../../features/proposal-circle-not-choosable/pre-assembly-routing-application.feature"},
			Tags:     "~@wip",
			TestingT: t,
		},
	}
	if suite.Run() != 0 {
		t.Fatal("non-zero status: pre-assembly-routing-application feature scenarios failed")
	}
}

func (w *preAssemblyGateWorld) registerRouting(sc *godog.ScenarioContext) {
	sc.Before(func(ctx context.Context, _ *godog.Scenario) (context.Context, error) {
		*w = preAssemblyGateWorld{}
		return ctx, nil
	})

	// Scenario: Routing names the target circle and every eligible anchor, choosing none
	sc.Step(`^an intended change and no anchor settled on$`, w.givenLoadedArtifacts)
	sc.Step(`^the drafter runs the recorded routing procedure's reads in their order$`, w.whenRoutingReadsInOrder)
	sc.Step(`^it will report the target circle's role_ id and every eligible anchor's ten_ id$`, w.thenReportsTargetAndAnchors)
	sc.Step(`^it will return action named-anchors, choosing none — the choice is the practitioner's$`, w.thenNamedAnchorsChoosingNone)

	// Scenario: A mismatched handed-in anchor is reported, not drafted on silently
	sc.Step(`^a handed-in anchor whose determination landed the change outside the target circle$`, w.givenMismatchedAnchor)
	sc.Step(`^the drafter evaluates the handed-in anchor$`, w.whenLoadedArtifacts)
	sc.Step(`^it will return action surfaced-routing-mismatch naming the eligible anchors that reach the target circle$`, w.thenRoutingMismatchNamesAnchors)
	sc.Step(`^drafting will proceed on the handed-in anchor where the practitioner directs it — the mismatch is reported, not enforced$`, w.thenMismatchReportedNotEnforced)

	// Scenario: An empty eligible set names capture as the closing step
	sc.Step(`^a target circle where the operator filled a role but no tension was sensed on it$`, w.givenLoadedArtifacts)
	sc.Step(`^the routing determination reports$`, w.whenLoadedArtifacts)
	sc.Step(`^it will return action named-anchors with an empty eligible set$`, w.thenNamedAnchorsEmptySet)
	sc.Step(`^the capture-gap note will name capture on that specific role in that specific circle as the step that closes the gap, handed onward rather than performed$`, w.thenCaptureGapHandedOnward)

	// Scenario: An incomplete routing walk continues flagged, inventing nothing
	sc.Step(`^a routing determination whose reads failed before the procedure completed$`, w.givenRoutingReadFailed)
	sc.Step(`^the drafter reports its answer$`, w.whenLoadedArtifacts)
	sc.Step(`^it will name what failed and present the determination as incomplete in the consultation element$`, w.thenIncompleteInConsultation)
	sc.Step(`^it will continue on what was established, neither inventing the unread part nor abandoning it$`, w.thenContinuesInventingNothing)

	// Scenario: A root circle's missing parent is declined, not resolved
	sc.Step(`^a change to the governance of a circle whose parent_role_id was null$`, w.givenNullParentCase)
	sc.Step(`^the routing part of the consultation element will carry the record's decline that no target is resolved for that case$`, w.thenCarriesRecordsDecline)
	sc.Step(`^no target circle will be invented or chosen in its place$`, w.thenNoTargetInvented)

	// Scenario: Both artifact descriptions state the routed entry
	sc.Step(`^the proposal-drafting skill and the proposal-drafter agent after the gate landed$`, w.givenLoadedArtifacts)
	sc.Step(`^their frontmatter descriptions are read$`, w.whenDescriptionsRead)
	sc.Step(`^each description will state that the path determines where the change lands before an anchor is settled on$`, w.thenDescriptionsStateRoutedEntry)
	sc.Step(`^every boundary sentence the descriptions carried before will still be present$`, w.thenBoundarySentencesRetained)
}

// --- Scenario: routing answers, choosing none ---------------------------------

// whenRoutingReadsInOrder asserts the workflow's Route step runs the reads the
// record names, in the record's order — both sides derived: the reads and
// their order from the record's named-reads block, the step body from the
// parsed workflow. The in-order listing is matched as the comma-joined
// sequence so a read's token appearing inside a sibling leaf (roles inside
// me roles) cannot satisfy the order by accident.
func (w *preAssemblyGateWorld) whenRoutingReadsInOrder() error {
	if err := w.ensureRouting(); err != nil {
		return err
	}
	named := w.routing.NamedReads
	if len(named) == 0 {
		return fmt.Errorf("the routing record declares no named reads")
	}
	w.steps = DraftingWorkflowSteps(w.skillRaw)
	route, err := w.stepIndex("Route")
	if err != nil {
		return err
	}
	body := grammarNorm(w.steps[route].Body)
	if !strings.Contains(body, strings.Join(named, ", ")) {
		return fmt.Errorf("the Route step does not run the record's reads in the record's order %v: %q", named, body)
	}
	if !containsFold(body, "in the record's order") {
		return fmt.Errorf("the Route step does not state it runs the procedure in the record's order: %q", body)
	}
	return nil
}

func (w *preAssemblyGateWorld) thenReportsTargetAndAnchors() error {
	return requirePhrases("the artifacts", w.combined(),
		"the target circle's role_ id", "every eligible anchor's ten_ id")
}

func (w *preAssemblyGateWorld) thenNamedAnchorsChoosingNone() error {
	if err := w.assertActionVocabulary(); err != nil {
		return err
	}
	if !mentionsToken(w.combined(), "named-anchors") {
		return fmt.Errorf("the artifacts never return action named-anchors")
	}
	if !containsFold(w.combined(), "choosing none") && !containsFold(w.combined(), "none chosen") {
		return fmt.Errorf("the artifacts do not state the routing answer chooses no anchor")
	}
	return requirePhrases("the artifacts", w.combined(), "the anchor choice is the practitioner's")
}

// --- Scenario: mismatched handed-in anchor -------------------------------------

func (w *preAssemblyGateWorld) givenMismatchedAnchor() error {
	if err := w.ensureArtifacts(); err != nil {
		return err
	}
	return requirePhrases("the artifacts", w.combined(), "outside the target circle")
}

func (w *preAssemblyGateWorld) thenRoutingMismatchNamesAnchors() error {
	if err := w.assertActionVocabulary(); err != nil {
		return err
	}
	return requirePhrases("the agent", w.agent,
		"surfaced-routing-mismatch", "eligible anchors that reach the target circle")
}

func (w *preAssemblyGateWorld) thenMismatchReportedNotEnforced() error {
	return requirePhrases("the agent", w.agent,
		"proceeds on the handed-in anchor where the practitioner directs it",
		"reported, never enforced")
}

// --- Scenario: empty eligible set ------------------------------------------------

func (w *preAssemblyGateWorld) thenNamedAnchorsEmptySet() error {
	if !mentionsToken(w.agent, "named-anchors") {
		return fmt.Errorf("the agent never returns action named-anchors")
	}
	return requirePhrases("the agent", w.agent, "empty eligible set", "capture-gap")
}

func (w *preAssemblyGateWorld) thenCaptureGapHandedOnward() error {
	return requirePhrases("the agent", w.agent,
		"capture on that specific role in that specific circle",
		"closes the gap", "handed onward", "never performed")
}

// --- Scenario: incomplete routing walk -------------------------------------------

func (w *preAssemblyGateWorld) givenRoutingReadFailed() error {
	if err := w.ensureArtifacts(); err != nil {
		return err
	}
	return requirePhrases("the agent", w.agent, "fails before the procedure completes")
}

func (w *preAssemblyGateWorld) thenIncompleteInConsultation() error {
	if !mentionsToken(w.agent, "consultation") {
		return fmt.Errorf("the agent's record carries no consultation element")
	}
	return requirePhrases("the agent", w.agent, "name what failed", "incomplete")
}

func (w *preAssemblyGateWorld) thenContinuesInventingNothing() error {
	return requirePhrases("the agent", w.agent,
		"continue on what was established", "inventing", "abandon")
}

// --- Scenario: root circle declined ----------------------------------------------

// givenNullParentCase grounds the premise in the record: the Root circle field
// must cite parent_role_id as the signal and state the case is not resolved —
// the decline the drafter's routing part carries verbatim in posture.
func (w *preAssemblyGateWorld) givenNullParentCase() error {
	if err := w.ensureArtifacts(); err != nil {
		return err
	}
	if err := w.ensureRouting(); err != nil {
		return err
	}
	root := grammarNorm(w.routing.RuleFields["Root circle"])
	if root == "" {
		return fmt.Errorf("the routing record carries no Root circle field")
	}
	if !strings.Contains(root, "parent_role_id") {
		return fmt.Errorf("the Root circle field does not cite parent_role_id as the signal: %q", root)
	}
	if !containsFold(root, "not resolved") {
		return fmt.Errorf("the Root circle field does not state the case is not resolved: %q", root)
	}
	return nil
}

func (w *preAssemblyGateWorld) thenCarriesRecordsDecline() error {
	if !mentionsToken(w.agent, "routing") || !mentionsToken(w.agent, "consultation") {
		return fmt.Errorf("the agent's record carries no routing part in a consultation element")
	}
	return requirePhrases("the agent", w.agent,
		"the record's decline", "no containing circle")
}

func (w *preAssemblyGateWorld) thenNoTargetInvented() error {
	return requirePhrases("the agent", w.agent,
		"invent or choose no target circle in its place")
}

// --- Scenario: widened descriptions ----------------------------------------------

func (w *preAssemblyGateWorld) whenDescriptionsRead() error {
	for _, artifact := range []struct{ label, raw string }{
		{"the skill", w.skillRaw},
		{"the agent", w.agentRaw},
	} {
		if FrontmatterDescription(artifact.raw) == "" {
			return fmt.Errorf("%s carries no frontmatter description", artifact.label)
		}
	}
	return nil
}

func (w *preAssemblyGateWorld) thenDescriptionsStateRoutedEntry() error {
	for _, artifact := range []struct{ label, raw string }{
		{"the skill description", w.skillRaw},
		{"the agent description", w.agentRaw},
	} {
		desc := grammarNorm(FrontmatterDescription(artifact.raw))
		if err := requirePhrases(artifact.label, desc,
			"determines where the change lands", "before an anchor is settled on"); err != nil {
			return err
		}
	}
	return nil
}

// thenBoundarySentencesRetained pins every boundary sentence the two
// descriptions carried before the widening — the discovery-surface fences that
// keep the sibling paths' triggers complementary. These are contract phrases
// (what the descriptions promised before 079), so they are pinned literally.
func (w *preAssemblyGateWorld) thenBoundarySentencesRetained() error {
	skillDesc := grammarNorm(FrontmatterDescription(w.skillRaw))
	if err := requirePhrases("the skill description", skillDesc,
		"Reach for this whenever a ready tension should become a draft proposal",
		"It is not for capturing, refining, or retiring a tension (that is the Tension Processing Path)",
		"it does not judge whether an action is allowed or needs a proposal (that is the Constraint Discovery Path)",
		"it does not explain the governance around a concern (that is governance navigation)",
		"it never advances or withdraws a circulating proposal (that is the Proposal Circulation Path)",
		"records a response on one (that is the proposal-impact-review path)",
	); err != nil {
		return err
	}
	agentDesc := grammarNorm(FrontmatterDescription(w.agentRaw))
	return requirePhrases("the agent description", agentDesc,
		"Never advances, responds to, or withdraws a proposal",
		"never a tension write",
		"never an authority verdict",
		"The proposal-drafting skill delegates drafting here",
	)
}
