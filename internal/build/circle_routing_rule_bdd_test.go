package build

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/cucumber/godog"
)

// TestCircleRoutingRuleFeatures runs the executable acceptance for the
// circle-routing record (073, circle-routing-rule.feature). Like the other
// build-side suites its Paths name ONLY this spec's content feature file and
// it runs with the ~@wip filter, so only the scenarios implemented so far
// execute. The sibling circle-routing-guard.feature is run by the guard suite.
//
// The deliverable is a committed markdown knowledge artifact, so the content
// scenarios assert against that file, following the operator-path convention:
// content assertions read a whitespace-collapsed copy (grammarNorm), structural
// checks read the raw file. The two @validation scenarios stay @wip for
// /score:validate.
func TestCircleRoutingRuleFeatures(t *testing.T) {
	w := &circleRoutingWorld{}
	suite := godog.TestSuite{
		ScenarioInitializer: func(sc *godog.ScenarioContext) { w.register(sc) },
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"../../features/proposal-circle-not-choosable/circle-routing-rule.feature"},
			Tags:     "~@wip",
			TestingT: t,
		},
	}
	if suite.Run() != 0 {
		t.Fatal("non-zero status: circle-routing-rule feature scenarios failed")
	}
}

// circleRoutingWorld is the per-scenario state: the raw record and its parse.
type circleRoutingWorld struct {
	raw    string
	record CircleRoutingRuleRecord
}

func (w *circleRoutingWorld) register(sc *godog.ScenarioContext) {
	sc.Before(func(ctx context.Context, _ *godog.Scenario) (context.Context, error) {
		*w = circleRoutingWorld{}
		return ctx, nil
	})

	// Givens
	sc.Step(`^the recorded routing content$`, w.givenRecord)
	sc.Step(`^the recorded routing content was read end to end$`, w.givenRecord)

	// Whens — every "consult"/"read" just ensures the record is loaded; the
	// record is declarative, so consulting it is reading it.
	sc.Step(`^it is consulted for a change to a circle's own domain or policy$`, w.whenConsult)
	sc.Step(`^its document header is read$`, w.whenConsult)
	sc.Step(`^its classification-test section is read$`, w.whenConsult)
	sc.Step(`^its named-reads block is read$`, w.whenConsult)
	sc.Step(`^it is consulted for a circle's own governance where the operator fills that circle's Circle Lead role$`, w.whenConsult)
	sc.Step(`^it is consulted for a change to a role inside a circle rather than to the circle itself$`, w.whenConsult)
	sc.Step(`^it is consulted for a change to the governance of a circle whose "([^"]*)" is null$`, w.whenConsultNullParent)
	sc.Step(`^its "Gap reporting" field is read$`, w.whenConsult)
	sc.Step(`^its Uncertainty field is read$`, w.whenConsult)
	sc.Step(`^it is inspected for what it would have a consumer refuse$`, w.whenConsult)

	// Thens
	sc.Step(`^its Own-circle consequence field will state the change must be anchored in that circle's parent$`, w.thenOwnCircleToParent)
	sc.Step(`^its Mechanism field will state the proposal lands in the circle of whichever tension anchors it$`, w.thenMechanismLands)
	sc.Step(`^its Owner line will name the proposal-drafting skill as the owning skill$`, w.thenOwnerNamesSkill)
	sc.Step(`^it will name symlink consumption as how any other skill would consume the record$`, w.thenOwnerNamesSymlink)
	sc.Step(`^its Contract citations line will name the published API specification$`, w.thenHeaderNamesSpec)
	sc.Step(`^its Test field will distinguish a change to a circle's own governance from a change to a role inside a circle$`, w.thenTestDistinguishes)
	sc.Step(`^its "Resolved by" field will name "([^"]*)" as what resolves whether a target is a circle$`, w.thenResolvedByNames)
	sc.Step(`^it will name "([^"]*)", "([^"]*)" and "([^"]*)" in the order the procedure runs them$`, w.thenNamedReadsInOrder)
	sc.Step(`^its "Answer shape" field will require the target circle named by its role_ id and each eligible anchor named by its ten_ id$`, w.thenAnswerShapeIDs)
	sc.Step(`^its "Circle Lead exception" field will state the circle-role itself is a valid anchor site$`, w.thenCircleLeadAnchors)
	sc.Step(`^it will not send the operator to the parent circle to find one$`, w.thenNoParentHopForLead)
	sc.Step(`^the stated Mechanism will route it to the circle containing that role$`, w.thenMechanismContainingCircle)
	sc.Step(`^no separate case will have to be looked up for it$`, w.thenNoSeparateCase)
	sc.Step(`^its "Root circle" field will state there is no parent circle to route to and that the case is not resolved$`, w.thenRootCircleLimit)
	sc.Step(`^it will name neither the circle itself nor any other circle as a default target$`, w.thenRootCircleNoDefault)
	sc.Step(`^it will require reporting that no eligible anchor exists yet$`, w.thenGapReportsNoAnchor)
	sc.Step(`^it will require naming capture on that specific role in that specific circle as the step that closes the gap$`, w.thenGapNamesCapture)
	sc.Step(`^it will require reporting "none found" and naming the read the search rested on$`, w.thenUncertaintyNoneFound)
	sc.Step(`^it will require marking the conclusion uncertain because the own-roles read does not follow pagination$`, w.thenUncertaintyPagination)
	sc.Step(`^nothing it prescribes will refuse, block, or delay a proposal create$`, w.thenNoRefusalPrescribed)
	sc.Step(`^the server will remain the judge of what it accepts$`, w.thenServerRemainsJudge)
}

// ensureRecord loads and parses the record once per scenario.
func (w *circleRoutingWorld) ensureRecord() error {
	if w.raw != "" {
		return nil
	}
	raw, err := ReadCircleRoutingRuleRecord()
	if err != nil {
		return fmt.Errorf("record did not load: %w", err)
	}
	w.raw = raw
	w.record = ParseCircleRoutingRuleRecord(raw)
	return nil
}

// field returns the named bold-labelled field from the given section's parse,
// normalized for content assertions, or an error naming what is missing.
func (w *circleRoutingWorld) field(fields map[string]string, section, label string) (string, error) {
	value := strings.TrimSpace(fields[label])
	if value == "" {
		return "", fmt.Errorf("the %s section carries no %q field", section, label)
	}
	return grammarNorm(value), nil
}

// --- Givens / Whens ---

func (w *circleRoutingWorld) givenRecord() error { return w.ensureRecord() }
func (w *circleRoutingWorld) whenConsult() error { return w.ensureRecord() }

// whenConsultNullParent additionally pins that the Root circle field cites the
// null-parent signal by the property name the scenario names.
func (w *circleRoutingWorld) whenConsultNullParent(property string) error {
	if err := w.ensureRecord(); err != nil {
		return err
	}
	root, err := w.field(w.record.RuleFields, "Rule", "Root circle")
	if err != nil {
		return err
	}
	if !strings.Contains(root, property) {
		return fmt.Errorf("Root circle does not cite %q as the signal: %q", property, root)
	}
	return nil
}

// --- Thens ---

func (w *circleRoutingWorld) thenOwnCircleToParent() error {
	f, err := w.field(w.record.RuleFields, "Rule", "Own-circle consequence")
	if err != nil {
		return err
	}
	if !containsFold(f, "anchored in") || !containsFold(f, "parent") {
		return fmt.Errorf("Own-circle consequence does not route the anchor to the parent circle: %q", f)
	}
	return nil
}

func (w *circleRoutingWorld) thenMechanismLands() error {
	f, err := w.field(w.record.RuleFields, "Rule", "Mechanism")
	if err != nil {
		return err
	}
	if !containsFold(f, "inherits the circle of its anchor tension's sensing role") {
		return fmt.Errorf("Mechanism does not state the proposal inherits its anchor's circle: %q", f)
	}
	if !containsFold(f, "the anchor choice is the routing choice") {
		return fmt.Errorf("Mechanism does not state the anchor choice is the routing choice: %q", f)
	}
	return nil
}

func (w *circleRoutingWorld) thenOwnerNamesSkill() error {
	f, err := w.field(w.record.HeaderFields, "document header", "Owner")
	if err != nil {
		return err
	}
	if !containsFold(f, "proposal-drafting skill") {
		return fmt.Errorf("Owner line does not name the proposal-drafting skill: %q", f)
	}
	return nil
}

func (w *circleRoutingWorld) thenOwnerNamesSymlink() error {
	f, err := w.field(w.record.HeaderFields, "document header", "Owner")
	if err != nil {
		return err
	}
	if !containsFold(f, "symbolic link") || !containsFold(f, "never a copy") {
		return fmt.Errorf("Owner line does not name symlink consumption (never a copy): %q", f)
	}
	return nil
}

func (w *circleRoutingWorld) thenHeaderNamesSpec() error {
	f, err := w.field(w.record.HeaderFields, "document header", "Contract citations")
	if err != nil {
		return err
	}
	if !containsFold(f, "Glassfrog API v5") {
		return fmt.Errorf("Contract citations line does not name the published API specification: %q", f)
	}
	return nil
}

func (w *circleRoutingWorld) thenTestDistinguishes() error {
	f, err := w.field(w.record.ClassificationFields, "classification-test", "Test")
	if err != nil {
		return err
	}
	if !containsFold(f, "circle's own governance") || !containsFold(f, "role inside a circle") {
		return fmt.Errorf("Test does not distinguish the two cases: %q", f)
	}
	if !containsFold(f, "change target alone") {
		return fmt.Errorf("Test does not classify from the change target alone: %q", f)
	}
	return nil
}

func (w *circleRoutingWorld) thenResolvedByNames(anchor string) error {
	f, err := w.field(w.record.ClassificationFields, "classification-test", "Resolved by")
	if err != nil {
		return err
	}
	if !strings.Contains(f, anchor) {
		return fmt.Errorf("Resolved by does not name %q: %q", anchor, f)
	}
	return nil
}

func (w *circleRoutingWorld) thenNamedReadsInOrder(first, second, third string) error {
	want := []string{first, second, third}
	got := w.record.NamedReads
	if len(got) != len(want) {
		return fmt.Errorf("named-reads block declares %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			return fmt.Errorf("named-reads block declares %v, want %v in that order", got, want)
		}
	}
	return nil
}

func (w *circleRoutingWorld) thenAnswerShapeIDs() error {
	f, err := w.field(w.record.ProcedureFields, "Procedure", "Answer shape")
	if err != nil {
		return err
	}
	if !strings.Contains(f, "role_") || !strings.Contains(f, "ten_") {
		return fmt.Errorf("Answer shape does not require role_ and ten_ ids: %q", f)
	}
	return nil
}

func (w *circleRoutingWorld) thenCircleLeadAnchors() error {
	f, err := w.field(w.record.RuleFields, "Rule", "Circle Lead exception")
	if err != nil {
		return err
	}
	if !containsFold(f, "circle-role itself is a valid anchor site") {
		return fmt.Errorf("Circle Lead exception does not state the circle-role itself anchors: %q", f)
	}
	return nil
}

func (w *circleRoutingWorld) thenNoParentHopForLead() error {
	f, err := w.field(w.record.RuleFields, "Rule", "Circle Lead exception")
	if err != nil {
		return err
	}
	if !containsFold(f, "need not go to the parent circle") {
		return fmt.Errorf("Circle Lead exception still sends the operator to the parent circle: %q", f)
	}
	return nil
}

func (w *circleRoutingWorld) thenMechanismContainingCircle() error {
	f, err := w.field(w.record.RuleFields, "Rule", "Mechanism")
	if err != nil {
		return err
	}
	if !containsFold(f, "lands in the circle containing that role") {
		return fmt.Errorf("Mechanism does not route to the circle containing the role: %q", f)
	}
	return nil
}

func (w *circleRoutingWorld) thenNoSeparateCase() error {
	f, err := w.field(w.record.RuleFields, "Rule", "Mechanism")
	if err != nil {
		return err
	}
	if !containsFold(f, "no separate case") {
		return fmt.Errorf("Mechanism does not state that no separate case exists: %q", f)
	}
	return nil
}

func (w *circleRoutingWorld) thenRootCircleLimit() error {
	f, err := w.field(w.record.RuleFields, "Rule", "Root circle")
	if err != nil {
		return err
	}
	if !containsFold(f, "no parent circle to route to") {
		return fmt.Errorf("Root circle does not state the limit: %q", f)
	}
	if !containsFold(f, "not resolved") {
		return fmt.Errorf("Root circle does not state the case is not resolved: %q", f)
	}
	return nil
}

func (w *circleRoutingWorld) thenRootCircleNoDefault() error {
	f, err := w.field(w.record.RuleFields, "Rule", "Root circle")
	if err != nil {
		return err
	}
	if !containsFold(f, "neither the circle itself nor any other circle") {
		return fmt.Errorf("Root circle does not decline both default targets: %q", f)
	}
	return nil
}

func (w *circleRoutingWorld) thenGapReportsNoAnchor() error {
	f, err := w.field(w.record.ProcedureFields, "Procedure", "Gap reporting")
	if err != nil {
		return err
	}
	if !containsFold(f, "no eligible anchor exists yet") {
		return fmt.Errorf("Gap reporting does not require reporting the missing anchor: %q", f)
	}
	return nil
}

func (w *circleRoutingWorld) thenGapNamesCapture() error {
	f, err := w.field(w.record.ProcedureFields, "Procedure", "Gap reporting")
	if err != nil {
		return err
	}
	if !containsFold(f, "capture on that specific role in that specific circle") {
		return fmt.Errorf("Gap reporting does not name capture on the specific role: %q", f)
	}
	if !containsFold(f, "closes the gap") {
		return fmt.Errorf("Gap reporting does not name capture as what closes the gap: %q", f)
	}
	return nil
}

func (w *circleRoutingWorld) thenUncertaintyNoneFound() error {
	f, err := w.field(w.record.ProcedureFields, "Procedure", "Uncertainty")
	if err != nil {
		return err
	}
	if !containsFold(f, "none found") {
		return fmt.Errorf("Uncertainty does not require reporting none found: %q", f)
	}
	if !strings.Contains(f, "me roles") {
		return fmt.Errorf("Uncertainty does not name the read the search rested on: %q", f)
	}
	return nil
}

func (w *circleRoutingWorld) thenUncertaintyPagination() error {
	f, err := w.field(w.record.ProcedureFields, "Procedure", "Uncertainty")
	if err != nil {
		return err
	}
	if !containsFold(f, "uncertain") {
		return fmt.Errorf("Uncertainty does not mark the conclusion uncertain: %q", f)
	}
	if !containsFold(f, "does not follow pagination") {
		return fmt.Errorf("Uncertainty does not ground the hedge in the pagination limitation: %q", f)
	}
	return nil
}

// thenNoRefusalPrescribed asserts by content inspection that the record
// prescribes reporting, never enforcement — the positive statement is the
// anchor, because the record must SAY it never refuses rather than merely
// omit refusal language.
func (w *circleRoutingWorld) thenNoRefusalPrescribed() error {
	norm := grammarNorm(w.raw)
	if !containsFold(norm, "nothing in this record refuses, blocks, or delays a proposal create") {
		return fmt.Errorf("the record does not state that it never refuses, blocks, or delays the write")
	}
	if !containsFold(norm, "reported, never enforced") {
		return fmt.Errorf("the record does not state that a routing gap is reported, never enforced")
	}
	return nil
}

func (w *circleRoutingWorld) thenServerRemainsJudge() error {
	if !containsFold(grammarNorm(w.raw), "the server remains the judge") {
		return fmt.Errorf("the record does not leave the server as the judge of what it accepts")
	}
	return nil
}
