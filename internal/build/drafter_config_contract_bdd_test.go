package build

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/cucumber/godog"
	"sigs.k8s.io/yaml"
)

// TestDrafterConfigContractFeatures runs the executable acceptance for Drafter
// Config Migration (071). Like 022's suite, its Paths name ONLY this spec's
// feature file, and it runs with the ~@wip filter so only the scenarios the
// guards make observable execute: five of the thirteen — the guard verdicts
// over fixtures (superseded schema rejected by name, a changelog category
// losing its label, the declared fallback removed) and the coupling verdicts
// (a below-floor pinned major, an underivable ref).
//
// Eight scenarios stay held @wip, for TWO DISTINCT reasons that must not be
// conflated:
//
//   - Held for /score:validate — the four @validation scenarios ("No label is
//     invented or dropped", "The four label-contract assertions survive",
//     "The change claims no fix for the untagged-release failure", "Neither
//     side of the coupling verdict is a hard-coded literal"). Score's
//     convention holds validation scenarios out from the implementing agent
//     for independent verification — exactly as release_bdd_test.go holds
//     022's three.
//   - Not executable at all — the four scenarios asserting drafter runtime
//     output ("A drafting run reports no schema deprecations", "A feature
//     merge still bumps the draft to the next minor", "The declared fallback
//     supplies the patch bump", "The exclusion survives the realignment").
//     spec 071 § Non-Behaviors forecloses verifying drafter output (no
//     synthetic-pull-request harness), so these stay documentation-grade —
//     the same standing as their neighbours in release-drafting.feature. This
//     is a boundary, not a backlog: rewording them into config-shape checks
//     would make them green while destroying what they assert.
func TestDrafterConfigContractFeatures(t *testing.T) {
	suite := godog.TestSuite{
		ScenarioInitializer: initializeDrafterConfigContractScenario,
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"../../features/no-automated-pipeline/drafter-config-contract.feature"},
			Tags:     "~@wip",
			TestingT: t,
		},
	}
	if suite.Run() != 0 {
		t.Fatal("non-zero status: drafter-config-contract feature scenarios failed")
	}
}

// supersededDrafterYAML is a config still expressing the contract in the
// superseded shape: top-level version-resolver and exclude-labels keys plus a
// category-level labels shorthand — the input the ADR-4 by-name rejection
// exists for.
const supersededDrafterYAML = `
version-resolver:
  major:
    labels: [breaking]
  minor:
    labels: [features]
  patch:
    labels: [fixes]
  default: patch
categories:
  - title: "Features"
    labels: [features]
exclude-labels:
  - no-release-note
`

// drafterContractWorld is the per-scenario state: the fixture config and
// workflow under test plus the verdicts computed when a guard runs.
type drafterContractWorld struct {
	rd    ReleaseDrafterConfig
	rdSet bool
	wf    Workflow
	wfSet bool

	pinnedRef    string // the ref the workflow fixture pinned, for name-the-value Thens
	missingLabel string // the label the Given removed from the contract

	labelViolations    []string
	labelRan           bool
	couplingViolations []string
	couplingRan        bool
}

func initializeDrafterConfigContractScenario(sc *godog.ScenarioContext) {
	w := &drafterContractWorld{}
	sc.Before(func(ctx context.Context, _ *godog.Scenario) (context.Context, error) {
		*w = drafterContractWorld{}
		return ctx, nil
	})

	// --- Givens ---
	sc.Step(`^a drafter configuration still carrying top-level "version-resolver" and "exclude-labels" keys$`, w.givenSupersededConfig)
	sc.Step(`^a changelog category in the drafter configuration named no label in its condition$`, w.givenCategoryWithoutLabel)
	sc.Step(`^the condition-less version-resolver category was deleted from the drafter configuration$`, w.givenFallbackDeleted)
	sc.Step(`^the drafter configuration was written in the current schema$`, w.givenCurrentSchemaConfig)
	sc.Step(`^the drafting workflow pinned the drafter action at "([^"]*)"$`, w.givenWorkflowPinnedAt)
	sc.Step(`^the drafting workflow pinned the drafter action at a commit SHA$`, w.givenWorkflowPinnedAtSHA)

	// --- Whens ---
	sc.Step(`^the label-contract guard runs$`, w.whenLabelContractGuardRuns)
	sc.Step(`^the coupling guard runs$`, w.whenCouplingGuardRuns)

	// --- Thens ---
	sc.Step(`^the guard will fail$`, w.thenGuardFails)
	sc.Step(`^the violation will name the superseded schema and point at the migration$`, w.thenNamesSupersededSchema)
	sc.Step(`^it will not report the seven category labels as merely missing$`, w.thenNotMerelyMissingLabels)
	sc.Step(`^the violation will name the configuration file and the label missing from the contract$`, w.thenNamesFileAndMissingLabel)
	sc.Step(`^the violation will name the expected fallback increment "([^"]*)" as an absent declaration$`, w.thenNamesFallbackAsAbsent)
	sc.Step(`^the guard will not pass on the grounds that the action would have fallen back to patch anyway$`, w.thenRedundancyIsNoExcuse)
	sc.Step(`^the violation will name the pinned version and the schema floor the configuration requires$`, w.thenNamesPinAndFloor)
	sc.Step(`^the mismatch will be caught before merge rather than by a drafting run$`, w.thenCaughtPreMerge)
	sc.Step(`^the violation will state that the pinned major could not be determined$`, w.thenStatesMajorUnderivable)
	sc.Step(`^it will name the workflow file and the reference it read$`, w.thenNamesWorkflowFileAndRef)
}

// --- Given implementations -------------------------------------------------

func (w *drafterContractWorld) setDrafter(raw string) error {
	var rd ReleaseDrafterConfig
	if err := yaml.Unmarshal([]byte(raw), &rd); err != nil {
		return fmt.Errorf("parsing fixture release-drafter.yml: %w", err)
	}
	w.rd = rd
	w.rdSet = true
	return nil
}

func (w *drafterContractWorld) setWorkflow(raw string) error {
	var wf Workflow
	if err := yaml.Unmarshal([]byte(raw), &wf); err != nil {
		return fmt.Errorf("parsing fixture release-drafting.yml: %w", err)
	}
	w.wf = wf
	w.wfSet = true
	return nil
}

func (w *drafterContractWorld) givenSupersededConfig() error {
	return w.setDrafter(supersededDrafterYAML)
}

// givenCategoryWithoutLabel feeds a current-schema config whose Documentation
// changelog category names no label in its when — the concrete instance of "a
// category lost its label predicate".
func (w *drafterContractWorld) givenCategoryWithoutLabel() error {
	w.missingLabel = "docs"
	return w.setDrafter(strings.Replace(validDrafterYAML,
		"  - title: \"Documentation\"\n    when:\n      labels: [docs]\n",
		"  - title: \"Documentation\"\n    when:\n      labels: []\n", 1))
}

// givenFallbackDeleted removes the condition-less version-resolver entry. The
// anchor includes the patch bucket above it because `semver-increment:
// "patch"` alone matches the bucket entry first.
func (w *drafterContractWorld) givenFallbackDeleted() error {
	return w.setDrafter(strings.Replace(validDrafterYAML,
		"      labels: [fixes]\n  - type: \"version-resolver\"\n    semver-increment: \"patch\"\n",
		"      labels: [fixes]\n", 1))
}

func (w *drafterContractWorld) givenCurrentSchemaConfig() error {
	return w.setDrafter(validDrafterYAML)
}

func (w *drafterContractWorld) givenWorkflowPinnedAt(ref string) error {
	w.pinnedRef = ref
	return w.setWorkflow(strings.Replace(validDraftingWorkflowYAML,
		"@v7.7.0", "@"+ref, 1))
}

func (w *drafterContractWorld) givenWorkflowPinnedAtSHA() error {
	return w.givenWorkflowPinnedAt("1b38099d29f3144567d9aabbccddeeff00112233")
}

// --- When implementations --------------------------------------------------

// whenLabelContractGuardRuns evaluates CheckLabelContract over the fixture
// drafter config and the valid labeler/settings baselines — the drafter config
// is the one thing each scenario varies.
func (w *drafterContractWorld) whenLabelContractGuardRuns() error {
	if !w.rdSet {
		return fmt.Errorf("no drafter configuration was given")
	}
	var labeler LabelerConfig
	if err := yaml.Unmarshal([]byte(validLabelerYAML), &labeler); err != nil {
		return fmt.Errorf("parsing fixture labeler.yml: %w", err)
	}
	var settings SettingsConfig
	if err := yaml.Unmarshal([]byte(validSettingsYAML), &settings); err != nil {
		return fmt.Errorf("parsing fixture settings.yml: %w", err)
	}
	w.labelViolations = CheckLabelContract(w.rd, labeler, settings)
	w.labelRan = true
	return nil
}

// whenCouplingGuardRuns evaluates CheckDrafterSchemaCoupling over the fixture
// workflow and the (current-schema unless a Given said otherwise) config.
func (w *drafterContractWorld) whenCouplingGuardRuns() error {
	if !w.wfSet {
		return fmt.Errorf("no drafting workflow was given")
	}
	if !w.rdSet {
		if err := w.setDrafter(validDrafterYAML); err != nil {
			return err
		}
	}
	w.couplingViolations = CheckDrafterSchemaCoupling(w.rd, w.wf)
	w.couplingRan = true
	return nil
}

// --- Then implementations --------------------------------------------------

// ranViolations returns whichever guard's verdict this scenario computed.
func (w *drafterContractWorld) ranViolations() ([]string, error) {
	switch {
	case w.labelRan:
		return w.labelViolations, nil
	case w.couplingRan:
		return w.couplingViolations, nil
	default:
		return nil, fmt.Errorf("no guard was run")
	}
}

func (w *drafterContractWorld) thenGuardFails() error {
	violations, err := w.ranViolations()
	if err != nil {
		return err
	}
	if len(violations) == 0 {
		return fmt.Errorf("expected the guard to fail, but it reported no violations")
	}
	return nil
}

func (w *drafterContractWorld) thenNamesSupersededSchema() error {
	joined, err := w.joinedLabelViolations()
	if err != nil {
		return err
	}
	if !strings.Contains(joined, "superseded schema") {
		return fmt.Errorf("the violation must name the superseded schema, got:\n%s", joined)
	}
	if !strings.Contains(joined, "071") {
		return fmt.Errorf("the violation must point at the migration (071), got:\n%s", joined)
	}
	return nil
}

// thenNotMerelyMissingLabels asserts the failure is not readable as ONLY a
// pile of missing category labels: a by-name schema violation is present, so
// the obvious-but-wrong fix (re-adding labels at the superseded positions) is
// never the message.
func (w *drafterContractWorld) thenNotMerelyMissingLabels() error {
	joined, err := w.joinedLabelViolations()
	if err != nil {
		return err
	}
	if !strings.Contains(joined, "superseded schema") {
		return fmt.Errorf("without a schema-naming violation the failure reads as merely-missing labels, got:\n%s", joined)
	}
	return nil
}

func (w *drafterContractWorld) thenNamesFileAndMissingLabel() error {
	joined, err := w.joinedLabelViolations()
	if err != nil {
		return err
	}
	if w.missingLabel == "" {
		return fmt.Errorf("the scenario did not establish which label went missing")
	}
	if !strings.Contains(joined, "release-drafter.yml category") {
		return fmt.Errorf("the violation must name the configuration file's category contract, got:\n%s", joined)
	}
	want := fmt.Sprintf("required release-drafter.yml category label missing: %q", w.missingLabel)
	if !strings.Contains(joined, want) {
		return fmt.Errorf("the violation must name the missing label as %s, got:\n%s", want, joined)
	}
	return nil
}

func (w *drafterContractWorld) thenNamesFallbackAsAbsent(increment string) error {
	joined, err := w.joinedLabelViolations()
	if err != nil {
		return err
	}
	if !strings.Contains(joined, "version-resolver default") {
		return fmt.Errorf("the violation must name the version-resolver default, got:\n%s", joined)
	}
	if !strings.Contains(joined, fmt.Sprintf("%q", increment)) {
		return fmt.Errorf("the violation must name the expected increment %q, got:\n%s", increment, joined)
	}
	if !strings.Contains(joined, "no condition-less version-resolver category declares it") {
		return fmt.Errorf("the violation must read as an absent declaration, not an empty-string mismatch, got:\n%s", joined)
	}
	return nil
}

// thenRedundancyIsNoExcuse pins the reason this scenario exists: the action's
// built-in fallback is also patch, so removing the declaration changes no
// drafter output — the guard must fail anyway, on the declaration's absence.
func (w *drafterContractWorld) thenRedundancyIsNoExcuse() error {
	if !w.labelRan {
		return fmt.Errorf("the label-contract guard was not run")
	}
	if len(w.labelViolations) == 0 {
		return fmt.Errorf("the guard passed — behavioral redundancy must not excuse the absent declaration")
	}
	return nil
}

func (w *drafterContractWorld) thenNamesPinAndFloor() error {
	joined, err := w.joinedCouplingViolations()
	if err != nil {
		return err
	}
	if w.pinnedRef == "" {
		return fmt.Errorf("the scenario did not establish a pinned ref")
	}
	if !strings.Contains(joined, fmt.Sprintf("%q", w.pinnedRef)) {
		return fmt.Errorf("the violation must name the pinned version %q, got:\n%s", w.pinnedRef, joined)
	}
	if !strings.Contains(joined, fmt.Sprintf("requires (major %d)", DrafterSchemaMinMajor)) {
		return fmt.Errorf("the violation must name the schema floor (major %d), got:\n%s", DrafterSchemaMinMajor, joined)
	}
	return nil
}

// thenCaughtPreMerge asserts the mismatch verdict came from the guard as a
// pure function over parsed structures — this suite runs under `go test` in
// the 024 pre-merge matrix, with no drafting run involved. The drafting
// workflow itself triggers only on push to main and blocks nothing, so a
// verdict computed here IS the before-merge catch.
func (w *drafterContractWorld) thenCaughtPreMerge() error {
	if !w.couplingRan {
		return fmt.Errorf("the coupling guard was not run")
	}
	if len(w.couplingViolations) == 0 {
		return fmt.Errorf("no violation was produced for the pre-merge gate to catch")
	}
	return nil
}

func (w *drafterContractWorld) thenStatesMajorUnderivable() error {
	joined, err := w.joinedCouplingViolations()
	if err != nil {
		return err
	}
	if !strings.Contains(joined, "pinned major could not be determined") {
		return fmt.Errorf("the violation must state the pinned major could not be determined, got:\n%s", joined)
	}
	return nil
}

func (w *drafterContractWorld) thenNamesWorkflowFileAndRef() error {
	joined, err := w.joinedCouplingViolations()
	if err != nil {
		return err
	}
	if !strings.Contains(joined, DraftingWorkflowFileName) {
		return fmt.Errorf("the violation must name the workflow file %s, got:\n%s", DraftingWorkflowFileName, joined)
	}
	if w.pinnedRef == "" || !strings.Contains(joined, w.pinnedRef) {
		return fmt.Errorf("the violation must name the reference it read (%q), got:\n%s", w.pinnedRef, joined)
	}
	return nil
}

func (w *drafterContractWorld) joinedLabelViolations() (string, error) {
	if !w.labelRan {
		return "", fmt.Errorf("the label-contract guard was not run")
	}
	return strings.Join(w.labelViolations, "\n"), nil
}

func (w *drafterContractWorld) joinedCouplingViolations() (string, error) {
	if !w.couplingRan {
		return "", fmt.Errorf("the coupling guard was not run")
	}
	return strings.Join(w.couplingViolations, "\n"), nil
}
