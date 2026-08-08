package build

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/cucumber/godog"
)

// TestCircleRoutingRuleGuard is the best-effort guard for the circle-routing
// record (073, plan ADR-4). It holds the record to its own structure, to the
// shipped CLI, and to the contract it cites — deriving every side from its
// file at test time. See CheckCircleRoutingRule for the invariants and the
// three explicitly-partial residues.
func TestCircleRoutingRuleGuard(t *testing.T) {
	raw, err := ReadCircleRoutingRuleRecord()
	if err != nil {
		t.Fatalf("could not read the circle-routing record (%s): %v", CircleRoutingRulePath, err)
	}
	rec := ParseCircleRoutingRuleRecord(raw)

	proposalProps, roleProps, err := LoadSpecRoutingAnchors()
	if err != nil {
		t.Fatalf("could not derive the spec-side sets from %s: %v", VendoredSpecPath, err)
	}
	liveTop, liveMe, liveTension, err := loadRoutingLiveSurfaces()
	if err != nil {
		t.Fatal(err)
	}

	// Sanity-check the extraction so a regression in the parsers fails loudly
	// rather than passing vacuously.
	if len(rec.NamedReads) == 0 {
		t.Fatal("no named reads parsed from the record — the parser or the record regressed")
	}
	if len(rec.CitedPremiseProps) == 0 {
		t.Fatal("no cited premise properties parsed from the record — the parser or the record regressed")
	}
	if len(rec.RoleCitations) == 0 {
		t.Fatal("no Role citations parsed from the record — the parser or the record regressed")
	}

	// The committed record must agree with itself, the CLI, and the contract.
	if v := CheckCircleRoutingRule(rec, proposalProps, roleProps, liveTop, liveMe, liveTension); len(v) != 0 {
		t.Fatalf("the committed circle-routing record drifted:\n  - %s", strings.Join(v, "\n  - "))
	}
}

// TestCircleRoutingRuleGuardConditions proves each violation condition fires
// loudly, naming the offending element and its resolution path — a guard that
// never fails is no guard. Each case mutates a valid fixture (or the passed
// spec/live sides) to trip exactly one condition, then asserts the message
// names what the interface-spec § Error Communication table requires.
// Condition 7 (record↔registry agreement) does not exist yet; it lands with
// the composed-surface widening (073 phase 2).
func TestCircleRoutingRuleGuardConditions(t *testing.T) {
	// Fixture sides the record fixture is built against.
	proposalProps := []string{"changes", "tension_id"}
	roleProps := []string{"has_subroles", "id", "name", "parent_role_id"}
	liveTop := []string{"roles", "tree"}
	liveMe := []string{"actions", "roles"}
	liveTension := []string{"list", "show"}

	check := func(record string, pp, rp, lt, lm, ltn []string) []string {
		return CheckCircleRoutingRule(ParseCircleRoutingRuleRecord(record), pp, rp, lt, lm, ltn)
	}

	// Baseline must pass — otherwise the mutations prove nothing.
	if v := check(validRoutingRecordFixture(), proposalProps, roleProps, liveTop, liveMe, liveTension); len(v) != 0 {
		t.Fatalf("baseline fixture is not clean:\n  - %s", strings.Join(v, "\n  - "))
	}

	cases := []struct {
		name          string
		record        string   // fixture (default valid)
		proposalProps []string // default fixture set
		roleProps     []string // default fixture set
		liveTension   []string // default fixture set
		wantNames     []string // substrings the failure must name
	}{
		{
			name:      "1: a required section is absent",
			record:    removeRoutingSection(validRoutingRecordFixture(), "Classification test"),
			wantNames: []string{"Classification test", "add the section", "not merely terse"},
		},
		{
			name:      "2: a required field label is missing",
			record:    strings.Replace(validRoutingRecordFixture(), "- **Root circle**: `Role.parent_role_id` null means no parent circle to route to; not resolved.\n", "", 1),
			wantNames: []string{"Rule section", "Root circle", "supply the field"},
		},
		{
			name:      "2: a header line is missing",
			record:    strings.Replace(validRoutingRecordFixture(), "- **Owner**: the proposal-drafting skill; consumed via a symbolic link, never a copy.\n", "", 1),
			wantNames: []string{"document header", "Owner", "supply the field"},
		},
		{
			name:      "3: the empirical marker is absent",
			record:    stripRoutingMarker(validRoutingRecordFixture()),
			wantNames: []string{"empirical marker", "restore the leading blockquote", "Empirical record"},
		},
		{
			name:      "4: the named-reads block is absent or empty",
			record:    strings.Replace(validRoutingRecordFixture(), "```\nme roles\ntension list\nroles\n```\n\n", "", 1),
			wantNames: []string{"declares no reads", "declare the reads"},
		},
		{
			name:        "5: a named read the CLI no longer exposes",
			liveTension: []string{"show"}, // `tension list` dropped by a rename
			wantNames:   []string{"tension list", "surface searched: tension", "restore the command"},
		},
		{
			name:      "6: an unanchorable command path",
			record:    strings.Replace(validRoutingRecordFixture(), "```\nme roles\ntension list\nroles\n```", "```\nme roles\ntension list all\nroles\n```", 1),
			wantNames: []string{"tension list all", "cannot anchor", "supported forms", "extend the guard"},
		},
		{
			name:          "8: the premise tripwire on an added property",
			proposalProps: []string{"changes", "circle_id", "tension_id"},
			wantNames:     []string{"circle_id", "tension_id", "changes", "dissolves the rule's premise", "re-derive the rule", "retire the record"},
		},
		{
			name:      "9: a cited Role field dropped from the schema",
			roleProps: []string{"id", "name", "parent_role_id"}, // has_subroles dropped
			wantNames: []string{"Role.has_subroles", "Contract citations", "re-derive the citation", "retire the record"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			record := tc.record
			if record == "" {
				record = validRoutingRecordFixture()
			}
			pp := tc.proposalProps
			if pp == nil {
				pp = proposalProps
			}
			rp := tc.roleProps
			if rp == nil {
				rp = roleProps
			}
			ltn := tc.liveTension
			if ltn == nil {
				ltn = liveTension
			}
			v := check(record, pp, rp, liveTop, liveMe, ltn)
			if len(v) == 0 {
				t.Fatalf("condition did not fire; expected a violation naming %v", tc.wantNames)
			}
			joined := strings.Join(v, "\n")
			for _, want := range tc.wantNames {
				if !strings.Contains(joined, want) {
					t.Errorf("failure did not name %q; got:\n%s", want, joined)
				}
			}
		})
	}
}

// loadRoutingLiveSurfaces loads the three live CLI surfaces the named-reads
// resolution anchors against, reusing the family's extractors (064, 065, 066).
func loadRoutingLiveSurfaces() (liveTop, liveMe, liveTension []string, err error) {
	if liveTop, err = LiveTopLevelCommands(); err != nil {
		return nil, nil, nil, fmt.Errorf("could not extract the live top-level surface: %w", err)
	}
	if liveMe, err = LiveMeSubcommands(); err != nil {
		return nil, nil, nil, fmt.Errorf("could not extract the live me surface: %w", err)
	}
	if liveTension, err = LiveTensionSubcommands(); err != nil {
		return nil, nil, nil, fmt.Errorf("could not extract the live tension surface: %w", err)
	}
	return liveTop, liveMe, liveTension, nil
}

// TestCircleRoutingGuardFeatures runs the executable acceptance for the guard
// (073, circle-routing-guard.feature). The scenarios drive
// CheckCircleRoutingRule over a fixture record and real spec/live sides,
// mutating one side to trip a condition. Runs with the ~@wip filter; the
// condition-7 and composed-surface scenarios stay @wip until phase 2, the
// retirement scenario is process-inexecutable, and the three @validation
// scenarios are held for /score:validate.
func TestCircleRoutingGuardFeatures(t *testing.T) {
	w := &circleRoutingGuardWorld{}
	suite := godog.TestSuite{
		ScenarioInitializer: func(sc *godog.ScenarioContext) { w.register(sc) },
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"../../features/proposal-circle-not-choosable/circle-routing-guard.feature"},
			Tags:     "~@wip",
			TestingT: t,
		},
	}
	if suite.Run() != 0 {
		t.Fatal("non-zero status: circle-routing-guard feature scenarios failed")
	}
}

// circleRoutingGuardWorld is the per-scenario state: the fixture record, the
// spec-side and live-side sets (loaded real, then mutated to model a refresh
// or a CLI change), and the guard's findings.
type circleRoutingGuardWorld struct {
	record        string
	proposalProps []string
	roleProps     []string
	liveTop       []string
	liveMe        []string
	liveTension   []string
	realRaw       string   // the committed record, for the marker scenario
	added         string   // property added by the modelled refresh
	dropped       string   // Role field dropped by the modelled refresh
	badLeaf       string   // the unanchorable leaf introduced
	wantNamed     []string // what the outline example expects the failure to name
	wantResolve   string   // the resolution phrase the outline example expects
	violations    []string
	ran           bool
}

func (w *circleRoutingGuardWorld) register(sc *godog.ScenarioContext) {
	sc.Before(func(ctx context.Context, _ *godog.Scenario) (context.Context, error) {
		proposalProps, roleProps, err := LoadSpecRoutingAnchors()
		if err != nil {
			return ctx, fmt.Errorf("could not load spec-side sets: %w", err)
		}
		liveTop, liveMe, liveTension, err := loadRoutingLiveSurfaces()
		if err != nil {
			return ctx, err
		}
		*w = circleRoutingGuardWorld{
			record:        validRoutingRecordFixture(),
			proposalProps: proposalProps,
			roleProps:     roleProps,
			liveTop:       liveTop,
			liveMe:        liveMe,
			liveTension:   liveTension,
		}
		return ctx, nil
	})

	// Givens
	sc.Step(`^the record's premise cited "([^"]*)" as carrying no circle property$`, w.givenPremiseCited)
	sc.Step(`^the record cited "Role\.has_subroles" as the circle indicator and "Role\.parent_role_id" as the root signal$`, w.givenRoleCitations)
	sc.Step(`^the record named "([^"]*)" in its named-reads block$`, w.givenNamedRead)
	sc.Step(`^the record named a read carrying a command path of three tokens$`, w.givenThreeTokenRead)
	sc.Step(`^a record missing (a required section|a required field label|the named-reads block)$`, w.givenRecordMissing)
	sc.Step(`^the record's landing behaviour was observed rather than published$`, w.givenRealRecord)

	// Whens
	sc.Step(`^a spec refresh adds any property to that object beyond "tension_id" and "changes"$`, w.whenSpecAddsProperty)
	sc.Step(`^a spec refresh drops either field from the Role schema$`, w.whenSpecDropsRoleField)
	sc.Step(`^the CLI drops or renames that subcommand$`, w.whenCLIDropsTensionList)
	sc.Step(`^the guard resolves the named-reads block$`, w.whenGuardRuns)
	sc.Step(`^the guard evaluates the record$`, w.whenGuardRuns)
	sc.Step(`^the record is checked for its leading marker$`, w.whenMarkerChecked)

	// Thens
	sc.Step(`^the guard will fail naming both property sets so the addition is readable from the failure$`, w.thenFailNamingBothPropertySets)
	sc.Step(`^the message will name re-deriving the rule or retiring the record as the two resolution paths$`, w.thenNameRederiveRuleOrRetire)
	sc.Step(`^the guard will fail naming the missing field and the section citing it$`, w.thenFailNamingFieldAndSection)
	sc.Step(`^the message will name re-deriving the citation or retiring the record as the two resolution paths$`, w.thenNameRederiveCitationOrRetire)
	sc.Step(`^the guard will fail naming the leaf and which surface was searched$`, w.thenFailNamingLeafAndSurface)
	sc.Step(`^the record will not be able to name a read the CLI no longer exposes$`, w.thenBuildFails)
	sc.Step(`^it will fail naming the leaf and the supported forms$`, w.thenFailNamingLeafAndForms)
	sc.Step(`^it will not silently skip the leaf it cannot anchor$`, w.thenLeafNotSkipped)
	sc.Step(`^it will fail naming (.+)$`, w.thenFailNamingExpected)
	sc.Step(`^the message will name supplying the missing element as the resolution path$`, w.thenNameSupplyResolution)
	sc.Step(`^the marker will state that the absent circle parameter is contract while the landing is observed behaviour$`, w.thenRealMarkerStatesSplit)
	sc.Step(`^a record missing that marker will fail the guard$`, w.thenStrippedMarkerFailsGuard)
}

// run evaluates the guard over the current fixture + sides, once.
func (w *circleRoutingGuardWorld) run() {
	w.violations = CheckCircleRoutingRule(ParseCircleRoutingRuleRecord(w.record), w.proposalProps, w.roleProps, w.liveTop, w.liveMe, w.liveTension)
	w.ran = true
}

// ensureRan runs the guard if a Then is reached without an explicit When
// having driven it (the record is already in its Given state).
func (w *circleRoutingGuardWorld) ensureRan() {
	if !w.ran {
		w.run()
	}
}

func (w *circleRoutingGuardWorld) joined() string { return strings.Join(w.violations, "\n") }

// --- Givens ---

func (w *circleRoutingGuardWorld) givenPremiseCited(anchor string) error {
	if !strings.Contains(w.record, anchor) {
		return fmt.Errorf("fixture does not cite the premise anchor %q", anchor)
	}
	if got := ParseCircleRoutingRuleRecord(w.record).CitedPremiseProps; len(got) == 0 {
		return fmt.Errorf("fixture's premise cites no properties")
	}
	return nil
}

func (w *circleRoutingGuardWorld) givenRoleCitations() error {
	cited := map[string]bool{}
	for _, c := range ParseCircleRoutingRuleRecord(w.record).RoleCitations {
		cited[c.Field] = true
	}
	if !cited["has_subroles"] || !cited["parent_role_id"] {
		return fmt.Errorf("fixture does not cite both Role anchors; cited %v", cited)
	}
	return nil
}

func (w *circleRoutingGuardWorld) givenNamedRead(leaf string) error {
	for _, r := range ParseCircleRoutingRuleRecord(w.record).NamedReads {
		if r == leaf {
			return nil
		}
	}
	return fmt.Errorf("fixture's named-reads block does not name %q", leaf)
}

func (w *circleRoutingGuardWorld) givenThreeTokenRead() error {
	w.badLeaf = "tension list all"
	w.record = strings.Replace(w.record, "```\nme roles\ntension list\nroles\n```", "```\nme roles\n"+w.badLeaf+"\nroles\n```", 1)
	return nil
}

func (w *circleRoutingGuardWorld) givenRecordMissing(element string) error {
	switch element {
	case "a required section":
		w.record = removeRoutingSection(w.record, "Classification test")
		w.wantNamed = []string{"Classification test"}
		w.wantResolve = "add the section"
	case "a required field label":
		w.record = strings.Replace(w.record, "- **Root circle**: `Role.parent_role_id` null means no parent circle to route to; not resolved.\n", "", 1)
		w.wantNamed = []string{"Rule section", "Root circle"}
		w.wantResolve = "supply the field"
	case "the named-reads block":
		w.record = strings.Replace(w.record, "```\nme roles\ntension list\nroles\n```\n\n", "", 1)
		w.wantNamed = []string{"declares no reads"}
		w.wantResolve = "declare the reads"
	default:
		return fmt.Errorf("unhandled missing element %q", element)
	}
	return nil
}

func (w *circleRoutingGuardWorld) givenRealRecord() error {
	raw, err := ReadCircleRoutingRuleRecord()
	if err != nil {
		return fmt.Errorf("record did not load: %w", err)
	}
	w.realRaw = raw
	return nil
}

// --- Whens ---

func (w *circleRoutingGuardWorld) whenSpecAddsProperty() error {
	// The modelled refresh adds a circle parameter. The tripwire is a
	// set-equality, so the spelling is irrelevant — this concrete instance
	// stands for any added property.
	w.added = "circle_id"
	w.proposalProps = append(append([]string(nil), w.proposalProps...), w.added)
	w.run()
	return nil
}

func (w *circleRoutingGuardWorld) whenSpecDropsRoleField() error {
	w.dropped = "has_subroles"
	var kept []string
	for _, f := range w.roleProps {
		if f != w.dropped {
			kept = append(kept, f)
		}
	}
	w.roleProps = kept
	w.run()
	return nil
}

func (w *circleRoutingGuardWorld) whenCLIDropsTensionList() error {
	var kept []string
	for _, s := range w.liveTension {
		if s != "list" {
			kept = append(kept, s)
		}
	}
	w.liveTension = kept
	w.run()
	return nil
}

func (w *circleRoutingGuardWorld) whenGuardRuns() error {
	w.run()
	return nil
}

func (w *circleRoutingGuardWorld) whenMarkerChecked() error {
	if w.realRaw == "" {
		return fmt.Errorf("the real record was not loaded by the Given")
	}
	return nil
}

// --- Thens ---

func (w *circleRoutingGuardWorld) thenFailNamingBothPropertySets() error {
	w.ensureRan()
	if len(w.violations) == 0 {
		return fmt.Errorf("guard passed; expected the premise tripwire to fire")
	}
	if w.added == "" {
		return fmt.Errorf("no property was added; the When did not run")
	}
	// Both sets readable: the added property (spec side) and the cited premise
	// properties (record side) all appear in the failure.
	for _, want := range append([]string{w.added}, ParseCircleRoutingRuleRecord(w.record).CitedPremiseProps...) {
		if !strings.Contains(w.joined(), want) {
			return fmt.Errorf("failure does not name %q, so both sets are not readable: %s", want, w.joined())
		}
	}
	return nil
}

func (w *circleRoutingGuardWorld) thenNameRederiveRuleOrRetire() error {
	if !containsFold(w.joined(), "re-derive the rule") || !containsFold(w.joined(), "retire the record") {
		return fmt.Errorf("failure does not name both resolution paths: %s", w.joined())
	}
	return nil
}

func (w *circleRoutingGuardWorld) thenFailNamingFieldAndSection() error {
	w.ensureRan()
	if len(w.violations) == 0 {
		return fmt.Errorf("guard passed; expected the classification-anchor condition to fire")
	}
	if w.dropped == "" {
		return fmt.Errorf("no Role field was dropped; the When did not run")
	}
	if !strings.Contains(w.joined(), "Role."+w.dropped) {
		return fmt.Errorf("failure does not name the missing field Role.%s: %s", w.dropped, w.joined())
	}
	if !strings.Contains(w.joined(), "section") || !strings.Contains(w.joined(), "Contract citations") {
		return fmt.Errorf("failure does not name the section citing the field: %s", w.joined())
	}
	return nil
}

func (w *circleRoutingGuardWorld) thenNameRederiveCitationOrRetire() error {
	if !containsFold(w.joined(), "re-derive the citation") || !containsFold(w.joined(), "retire the record") {
		return fmt.Errorf("failure does not name both resolution paths: %s", w.joined())
	}
	return nil
}

func (w *circleRoutingGuardWorld) thenFailNamingLeafAndSurface() error {
	w.ensureRan()
	if !strings.Contains(w.joined(), `"tension list"`) {
		return fmt.Errorf("failure does not name the leaf: %s", w.joined())
	}
	if !strings.Contains(w.joined(), "surface searched: tension") {
		return fmt.Errorf("failure does not name the surface searched: %s", w.joined())
	}
	return nil
}

func (w *circleRoutingGuardWorld) thenBuildFails() error {
	w.ensureRan()
	if len(w.violations) == 0 {
		return fmt.Errorf("guard passed; the record could still name a read the CLI no longer exposes")
	}
	return nil
}

func (w *circleRoutingGuardWorld) thenFailNamingLeafAndForms() error {
	w.ensureRan()
	if w.badLeaf == "" {
		return fmt.Errorf("no unanchorable leaf was introduced; the Given did not run")
	}
	if !strings.Contains(w.joined(), w.badLeaf) {
		return fmt.Errorf("failure does not name the leaf %q: %s", w.badLeaf, w.joined())
	}
	if !strings.Contains(w.joined(), "supported forms") || !strings.Contains(w.joined(), "top-level") || !strings.Contains(w.joined(), "me <sub>") || !strings.Contains(w.joined(), "tension <sub>") {
		return fmt.Errorf("failure does not name the supported forms: %s", w.joined())
	}
	return nil
}

func (w *circleRoutingGuardWorld) thenLeafNotSkipped() error {
	// Reported rather than skipped: the unanchorable leaf produced a violation
	// of its own, not a silent pass.
	for _, v := range w.violations {
		if strings.Contains(v, w.badLeaf) {
			return nil
		}
	}
	return fmt.Errorf("the unanchorable leaf %q was silently skipped: %v", w.badLeaf, w.violations)
}

func (w *circleRoutingGuardWorld) thenFailNamingExpected(named string) error {
	w.ensureRan()
	if len(w.violations) == 0 {
		return fmt.Errorf("guard passed; expected a failure naming %q", named)
	}
	if len(w.wantNamed) == 0 {
		return fmt.Errorf("no expectation was set by the Given for %q", named)
	}
	for _, want := range w.wantNamed {
		if !strings.Contains(w.joined(), want) {
			return fmt.Errorf("failure does not name %q: %s", want, w.joined())
		}
	}
	return nil
}

func (w *circleRoutingGuardWorld) thenNameSupplyResolution() error {
	if w.wantResolve == "" {
		return fmt.Errorf("no resolution expectation was set by the Given")
	}
	if !containsFold(w.joined(), w.wantResolve) {
		return fmt.Errorf("failure does not name the resolution %q: %s", w.wantResolve, w.joined())
	}
	return nil
}

func (w *circleRoutingGuardWorld) thenRealMarkerStatesSplit() error {
	marker := parseLeadingMarker(w.realRaw)
	if !RoutingMarkerIsWellFormed(marker) {
		return fmt.Errorf("the committed record's marker does not state the cite-versus-observe split: %q", marker)
	}
	return nil
}

// thenStrippedMarkerFailsGuard strips the leading marker from the real record
// and runs the guard against the real sides — the marker's absence must
// surface as a condition-3 violation.
func (w *circleRoutingGuardWorld) thenStrippedMarkerFailsGuard() error {
	idx := strings.Index(w.realRaw, "# Circle Routing Rule")
	if idx < 0 {
		return fmt.Errorf("could not locate the document header to strip the marker")
	}
	stripped := ParseCircleRoutingRuleRecord(w.realRaw[idx:])
	v := CheckCircleRoutingRule(stripped, w.proposalProps, w.roleProps, w.liveTop, w.liveMe, w.liveTension)
	for _, msg := range v {
		if containsFold(msg, "marker") {
			return nil
		}
	}
	return fmt.Errorf("guard did not fail on the missing marker; got %v", v)
}

// --- fixtures and surgery helpers ---

// validRoutingRecordFixture is a minimal well-formed record — leading marker
// with the cite-versus-observe split, document header, and the four required
// sections with every required field and the named-reads block. Its cited
// premise set, Role anchors, and named reads match the real contract and the
// real CLI, so the baseline passes against real sides too. Shared by the
// guard condition table and the guard BDD suite.
func validRoutingRecordFixture() string {
	return "> Empirical record. That `proposal create` carries no circle parameter is published contract; where the proposal lands is observed server behaviour.\n\n" +
		"# Circle Routing Rule\n\n" +
		"- **Owner**: the proposal-drafting skill; consumed via a symbolic link, never a copy.\n" +
		"- **Contract citations**: `spec/glassfrog-api-v5.yaml` — by schema and property name.\n\n" +
		"## Contract citations\n\n" +
		"- **Premise**: `CreateProposalRequest.properties.proposal` requires only `tension_id`, optionally carries `changes`, and has no circle property.\n" +
		"- **Circle indicator**: `Role.has_subroles` resolves whether a target is a circle.\n" +
		"- **Root signal**: `Role.parent_role_id` is nullable; null is the root signal.\n\n" +
		"## Rule\n\n" +
		"- **Mechanism**: a proposal inherits the circle of its anchor tension's sensing role.\n" +
		"- **Own-circle consequence**: anchored in the circle's parent.\n" +
		"- **Circle Lead exception**: the circle-role itself is a valid anchor site.\n" +
		"- **Root circle**: `Role.parent_role_id` null means no parent circle to route to; not resolved.\n\n" +
		"## Classification test\n\n" +
		"- **Test**: classify from the change target alone.\n" +
		"- **Resolved by**: `Role.has_subroles`.\n" +
		"- **Parent resolution**: `Role.parent_role_id` names the containing role.\n\n" +
		"## Procedure\n\n" +
		"```\nme roles\ntension list\nroles\n```\n\n" +
		"- **Answer shape**: the circle by `role_` id; each anchor by `ten_` id.\n" +
		"- **All anchors named**: all eligible anchors named, none chosen.\n" +
		"- **Gap reporting**: name capture on that specific role.\n" +
		"- **Uncertainty**: none found in `me roles`; the read does not follow pagination.\n"
}

// removeRoutingSection deletes the `## <heading>` section (heading through the
// line before the next `## ` heading or EOF) — the missing-section surgery.
func removeRoutingSection(record, heading string) string {
	lines := strings.Split(record, "\n")
	start, end := -1, len(lines)
	for i, line := range lines {
		if strings.TrimSpace(line) == "## "+heading {
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

// stripRoutingMarker removes the leading blockquote, leaving the record to
// open with its document header — the missing-marker surgery.
func stripRoutingMarker(record string) string {
	idx := strings.Index(record, "# Circle Routing Rule")
	if idx < 0 {
		return record
	}
	return record[idx:]
}
