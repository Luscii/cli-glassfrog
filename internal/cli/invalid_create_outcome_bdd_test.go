package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/cucumber/godog"
)

// TestInvalidCreateOutcomeFeatures runs the executable acceptance for
// Invalid-Create Outcome (078): the accepted-but-invalid create failing with its
// own exit code across the output formats, the three verdict states that stay
// successes, and the exit-code registry's extension. Its Paths name ONLY this
// spec's feature file, so the suite reports its own scenario count and cannot
// disturb another suite. The 4 @validation scenarios stay @wip (held for the
// validate skill) and are skipped by the ~@wip filter.
//
// The world EMBEDS the sibling suite's postCreateValidityWorld, because roughly
// two dozen of this file's step texts are byte-identical to that suite's. Godog
// matches steps by text, so a near-duplicate implementation here would be a second
// definition of the same sentence, free to drift from the first. Embedding makes
// every shared text resolve to the one implementation, and this file defines only
// the steps that are genuinely new.
func TestInvalidCreateOutcomeFeatures(t *testing.T) {
	suite := godog.TestSuite{
		ScenarioInitializer: initializeInvalidCreateOutcomeScenario,
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"../../features/success-reported-for-a-dead-proposal/invalid-create-outcome.feature"},
			Tags:     "~@wip",
			TestingT: t,
		},
	}
	if suite.Run() != 0 {
		t.Fatal("non-zero status: invalid-create-outcome feature scenarios failed")
	}
}

// invalidCreateWorld is the per-scenario state. It embeds the sibling suite's
// world so the shared Given/When/Then implementations operate on the same fields
// (the scripted exchanges, the injected templates, the captured streams), and adds
// only what this feature's own steps need.
type invalidCreateWorld struct {
	postCreateValidityWorld

	// registryBefore is the code each category held before the invalid-create
	// category was added — the snapshot the registry-extension scenario compares
	// against.
	registryBefore map[Outcome]int
	// registryAfter is the code the invalid-create category took.
	registryAfter int
}

func initializeInvalidCreateOutcomeScenario(sc *godog.ScenarioContext) {
	w := &invalidCreateWorld{}
	sc.Before(func(ctx context.Context, _ *godog.Scenario) (context.Context, error) {
		*w = invalidCreateWorld{
			postCreateValidityWorld: postCreateValidityWorld{
				createStep: proposalSeqStep{status: 201, body: proposalCreatedBody},
				// The default read-back carries no validity field; per-scenario
				// Givens override it with the verdict the scenario states.
				readStep: proposalSeqStep{status: 200, body: proposalCreatedBody},
				secret:   meSecretToken,
			},
		}
		w.ctx = validMeContext()
		return ctx, nil
	})

	// --- Givens shared with the sibling suite (one implementation, reused) ---
	sc.Step(`^a complete connection context with a stored token$`, w.completeContext)
	sc.Step(`^the created proposal reads back as valid with no validation alerts$`, w.readsBackValidNoAlerts)
	sc.Step(`^the created proposal reads back as not valid with the alert "([^"]*)" and no available transitions$`, w.readsBackNotValidWithAlertNoTransitions)
	sc.Step(`^the created proposal reads back as not valid with one validation alert$`, w.readsBackNotValidOneAlert)
	sc.Step(`^the created proposal reads back carrying no validity field$`, w.readsBackNoValidityField)
	sc.Step(`^the created proposal reads back as valid with one alert of severity "([^"]*)"$`, w.readsBackValidWithAlertSeverity)
	sc.Step(`^the proposals endpoint rejects the create$`, w.createRejected)
	sc.Step(`^the create succeeds but the read of the created proposal cannot reach the server$`, w.readBackUnreachable)

	// --- Givens new to this feature ---
	sc.Step(`^the created proposal reads back as not valid with no validation alerts$`, w.readsBackNotValidNoAlerts)
	sc.Step(`^the created proposal reads back as not valid with one validation alert and a non-empty transition set$`, w.readsBackNotValidWithTransitions)
	// Deliberately NOT the sibling's "…only the proposal fields the create rendered
	// before the verdict existed": that step asserts the pre-verdict field paths
	// still RESOLVE, this one asserts the template is never invoked at all. Two
	// different claims, so two wordings and two definitions — both stay.
	sc.Step(`^a user template referencing only proposal fields$`, w.markedUserTemplate)
	sc.Step(`^the exit-code registry with its existing assigned codes$`, w.registrySnapshot)

	// --- Whens ---
	sc.Step(`^an agent runs "glassfrog (.+)"$`, w.runCommand)
	sc.Step(`^an agent creates a proposal with that template selected$`, w.runCreateWithTemplate)
	sc.Step(`^the invalid-create category is added to it$`, w.registryAddInvalidCreate)

	// --- Thens shared with the sibling suite ---
	sc.Step(`^the created proposal will be printed with its "([^"]*)" id and "([^"]*)" status$`, w.createdPrintedWithIDAndStatus)
	sc.Step(`^the created proposal will be printed with its "([^"]*)" id$`, w.createdPrintedWithID)
	sc.Step(`^the result will report the validity as "([^"]*)"$`, w.validityReportedAs)
	sc.Step(`^the result will carry the alert with its severity, path, and message$`, w.alertCarriedWithSeverityAndPath)
	sc.Step(`^the create will have been followed by exactly one read of the created proposal$`, w.exactlyOneReadFollowed)
	sc.Step(`^the command will exit with code (\d+)$`, w.exitWithCode)
	sc.Step(`^the command will exit with a non-zero API-error code$`, w.exitNonZeroAPIError)
	sc.Step(`^the result will report that the server stated no verdict on the draft$`, w.noVerdictStatedReported)
	sc.Step(`^the structured result will contain the created proposal's "([^"]*)" id$`, w.structuredContainsID)
	sc.Step(`^the structured result will carry "([^"]*)", "([^"]*)", and "([^"]*)"$`, w.structuredCarriesKeys)
	sc.Step(`^the advisory will be rendered in the selected machine format, carrying "([^"]*)" as (true|false)$`, w.machineAdvisoryCarries)
	sc.Step(`^the compact line will carry the created "([^"]*)" id, its status, and the change count$`, w.compactCarriesIDStatusCount)
	sc.Step(`^it will carry the validity token and the alert count$`, w.compactCarriesValidityAndAlertCount)

	// --- Thens new to this feature ---
	sc.Step(`^stderr will carry the created "([^"]*)" id$`, w.stderrCarriesCreatedID)
	sc.Step(`^stderr will carry each alert with its severity, path, and message$`, w.stderrCarriesEachAlert)
	sc.Step(`^stderr will name creating a corrected proposal as the next step$`, w.stderrNamesCorrectedProposal)
	sc.Step(`^stdout will be empty$`, w.stdoutEmpty)
	sc.Step(`^stdout will carry the failure envelope with kind "([^"]*)"$`, w.stdoutCarriesEnvelopeWithKind)
	sc.Step(`^stdout will carry the failure envelope for the rejected create$`, w.stdoutCarriesEnvelopeForRejectedCreate)
	sc.Step(`^the envelope will carry "([^"]*)" and "([^"]*)"$`, w.envelopeCarriesBothKeys)
	sc.Step(`^the envelope will carry "([^"]*)"$`, w.envelopeCarriesKey)
	sc.Step(`^the envelope will not carry a "([^"]*)" key$`, w.envelopeLacksKey)
	sc.Step(`^the envelope will carry neither "([^"]*)" nor "([^"]*)"$`, w.envelopeCarriesNeitherKey)
	sc.Step(`^the envelope will carry the cause naming the created "([^"]*)" id$`, w.envelopeCauseNamesID)
	sc.Step(`^the envelope will carry the remedy as its own field$`, w.envelopeCarriesRemedyField)
	sc.Step(`^no server proposal document will be emitted$`, w.noServerDocumentEmitted)
	sc.Step(`^no verdict advisory will be emitted$`, w.noVerdictAdvisoryEmitted)
	sc.Step(`^no failure envelope will be emitted$`, w.noFailureEnvelopeEmitted)
	sc.Step(`^the template will not be rendered$`, w.templateNotRendered)
	sc.Step(`^the category will take a previously-unused code$`, w.categoryTookUnusedCode)
	sc.Step(`^every existing category will keep the code it had before$`, w.existingCategoriesKeptCodes)
}

// --- Given implementations ---

// readsBackNotValidNoAlerts is the invalid draft the server attached NO alerts to:
// it still fails, and its envelope omits validation_alerts entirely rather than
// rendering an empty array.
func (w *invalidCreateWorld) readsBackNotValidNoAlerts() error {
	w.wantValidity, w.wantAlertCount = "not valid", 0
	w.readStep = proposalSeqStep{status: 200, body: readBackBodyWith("draft", `"valid":false,"validation_alerts":[],`, `[]`)}
	return nil
}

// readsBackNotValidWithTransitions pairs the unfavourable verdict with a NON-EMPTY
// transition set — the fixture that proves the trigger keys on the verdict alone.
// 074's invalid fixture had both an unfavourable verdict and no transitions, so it
// could not tell the two apart.
func (w *invalidCreateWorld) readsBackNotValidWithTransitions() error {
	w.wantAlertMessage = "Can't update the Cloud Foundations role during this meeting."
	w.wantValidity, w.wantAlertCount = "not valid", 1
	alerts := fmt.Sprintf(`"valid":false,"validation_alerts":[{"severity":"error","path":"name","message":%q}],`, w.wantAlertMessage)
	w.readStep = proposalSeqStep{status: 200, body: readBackBodyWith("draft", alerts, `["propose","withdraw"]`)}
	return nil
}

// markedUserTemplate registers a template whose output carries an unmistakable
// marker, so the Then step can assert the template was never INVOKED rather than
// merely that its output looks empty.
func (w *invalidCreateWorld) markedUserTemplate() error {
	w.tmplFiles = map[string]string{
		"pre074.tmpl": "TEMPLATE-WAS-RENDERED {{.Proposal.ID}} {{.Proposal.Status}}",
	}
	return nil
}

// registrySnapshot captures the code every category held BEFORE the invalid-create
// category existed. The pairs are written out because that is what "the code it had
// before" means — there is no live source to derive a historical assignment from,
// and freezing it here is the same discipline as exitcode_test.go's
// TestExitCodeConstants_ExactValues change-detector. If a future capability
// renumbers one of these, both guards fail.
func (w *invalidCreateWorld) registrySnapshot() error {
	w.registryBefore = map[Outcome]int{
		Success:            0,
		RuntimeError:       1,
		UsageError:         2,
		APIError:           3,
		PermissionError:    4,
		RateLimited:        5,
		NetworkUnavailable: 6,
		StaleWrite:         7,
	}
	return nil
}

// --- When implementations ---

func (w *invalidCreateWorld) registryAddInvalidCreate() error {
	if w.registryBefore == nil {
		return fmt.Errorf("no registry snapshot was taken by the Given")
	}
	w.registryAfter = ExitCode(InvalidCreate)
	return nil
}

// --- Then implementations ---

func (w *invalidCreateWorld) stderrCarriesCreatedID(idPrefix string) error {
	if !strings.Contains(w.stderr, idPrefix) {
		return fmt.Errorf("stderr should carry the created %s id:\n%s", idPrefix, w.stderr)
	}
	return nil
}

// stderrCarriesEachAlert pins the human alert block: one line per alert carrying
// the severity, the path, and the server's own message. The expected message comes
// from what the read-back Given set up, never re-derived here.
func (w *invalidCreateWorld) stderrCarriesEachAlert() error {
	if w.wantAlertMessage == "" {
		return fmt.Errorf("no alert message was set up by the Given")
	}
	want := "  error name: " + w.wantAlertMessage
	if !strings.Contains(w.stderr, want) {
		return fmt.Errorf("stderr should carry the alert line %q:\n%s", want, w.stderr)
	}
	return nil
}

func (w *invalidCreateWorld) stderrNamesCorrectedProposal() error {
	if !strings.Contains(w.stderr, "create a corrected proposal from the same tension") {
		return fmt.Errorf("stderr should name creating a corrected proposal as the next step:\n%s", w.stderr)
	}
	return nil
}

func (w *invalidCreateWorld) stdoutEmpty() error {
	if w.stdout != "" {
		return fmt.Errorf("stdout should be empty on a human-format failure, got:\n%s", w.stdout)
	}
	return nil
}

// failureEnvelope decodes stdout as the shared failure envelope. It is the one
// decode every envelope assertion below goes through, so no step re-parses.
func (w *invalidCreateWorld) failureEnvelope() (map[string]json.RawMessage, error) {
	var env struct {
		Error map[string]json.RawMessage `json:"error"`
	}
	if err := json.Unmarshal([]byte(w.stdout), &env); err != nil {
		return nil, fmt.Errorf("stdout should be the failure envelope: %v\n%s", err, w.stdout)
	}
	if env.Error == nil {
		return nil, fmt.Errorf("stdout carries no error envelope:\n%s", w.stdout)
	}
	return env.Error, nil
}

func (w *invalidCreateWorld) stdoutCarriesEnvelopeWithKind(kind string) error {
	detail, err := w.failureEnvelope()
	if err != nil {
		return err
	}
	var got string
	if raw, ok := detail["kind"]; !ok || json.Unmarshal(raw, &got) != nil {
		return fmt.Errorf("the envelope should carry a kind:\n%s", w.stdout)
	}
	if got != kind {
		return fmt.Errorf("envelope kind = %q, want %q", got, kind)
	}
	return nil
}

func (w *invalidCreateWorld) stdoutCarriesEnvelopeForRejectedCreate() error {
	detail, err := w.failureEnvelope()
	if err != nil {
		return err
	}
	// A rejected create is an exchange failure, so its envelope carries the status
	// the server answered with — the create's own rejection, unchanged by 078.
	if _, ok := detail["status"]; !ok {
		return fmt.Errorf("a rejected create's envelope should carry the response status:\n%s", w.stdout)
	}
	return nil
}

func (w *invalidCreateWorld) envelopeCarriesKey(key string) error {
	detail, err := w.failureEnvelope()
	if err != nil {
		return err
	}
	if _, ok := detail[key]; !ok {
		return fmt.Errorf("the envelope should carry %q:\n%s", key, w.stdout)
	}
	return nil
}

func (w *invalidCreateWorld) envelopeCarriesBothKeys(first, second string) error {
	for _, key := range []string{first, second} {
		if err := w.envelopeCarriesKey(key); err != nil {
			return err
		}
	}
	return nil
}

func (w *invalidCreateWorld) envelopeLacksKey(key string) error {
	detail, err := w.failureEnvelope()
	if err != nil {
		return err
	}
	if raw, ok := detail[key]; ok {
		return fmt.Errorf("the %q key must be absent, not present as %s:\n%s", key, raw, w.stdout)
	}
	return nil
}

func (w *invalidCreateWorld) envelopeCarriesNeitherKey(first, second string) error {
	for _, key := range []string{first, second} {
		if err := w.envelopeLacksKey(key); err != nil {
			return err
		}
	}
	return nil
}

func (w *invalidCreateWorld) envelopeCauseNamesID(idPrefix string) error {
	detail, err := w.failureEnvelope()
	if err != nil {
		return err
	}
	var message string
	if raw, ok := detail["message"]; !ok || json.Unmarshal(raw, &message) != nil {
		return fmt.Errorf("the envelope should carry a message:\n%s", w.stdout)
	}
	if !strings.Contains(message, idPrefix) {
		return fmt.Errorf("the cause should name the created %s id, got %q", idPrefix, message)
	}
	// The cause does not enumerate the alerts — they have their own key.
	if w.wantAlertMessage != "" && strings.Contains(message, w.wantAlertMessage) {
		return fmt.Errorf("the cause must not enumerate the alerts: %q", message)
	}
	return nil
}

func (w *invalidCreateWorld) envelopeCarriesRemedyField() error {
	detail, err := w.failureEnvelope()
	if err != nil {
		return err
	}
	var nextStep string
	if raw, ok := detail["next_step"]; !ok || json.Unmarshal(raw, &nextStep) != nil || nextStep == "" {
		return fmt.Errorf("the envelope should carry the remedy in its own next_step field:\n%s", w.stdout)
	}
	if !strings.Contains(nextStep, "create a corrected proposal from the same tension") {
		return fmt.Errorf("next_step = %q, want it to name the remedy", nextStep)
	}
	return nil
}

func (w *invalidCreateWorld) noServerDocumentEmitted() error {
	if strings.Contains(w.stdout, `"data"`) {
		return fmt.Errorf("no server proposal document may ride stdout on the failure:\n%s", w.stdout)
	}
	return nil
}

func (w *invalidCreateWorld) noVerdictAdvisoryEmitted() error {
	if strings.Contains(w.stderr, "verdict_source") || strings.Contains(w.stderr, "the validity verdict was read back from proposal") {
		return fmt.Errorf("no verdict advisory may accompany the failure:\n%s", w.stderr)
	}
	return nil
}

// noFailureEnvelopeEmitted pins the non-trigger states: stdout carries no failure
// envelope at all, in either the human or the machine rendering.
func (w *invalidCreateWorld) noFailureEnvelopeEmitted() error {
	if strings.Contains(w.stdout, `"error"`) || strings.Contains(w.stdout, "invalid-create") {
		return fmt.Errorf("a success state must emit no failure envelope:\n%s", w.stdout)
	}
	if w.outcome != Success {
		return fmt.Errorf("outcome = %v, want Success\nstderr: %s", w.outcome, w.stderr)
	}
	return nil
}

func (w *invalidCreateWorld) templateNotRendered() error {
	if len(w.tmplFiles) == 0 {
		return fmt.Errorf("no user template was set up by the Given")
	}
	if strings.Contains(w.stdout+w.stderr, "TEMPLATE-WAS-RENDERED") {
		return fmt.Errorf("the template must never be invoked on the failure path:\nstdout: %s\nstderr: %s", w.stdout, w.stderr)
	}
	if w.stdout != "" {
		return fmt.Errorf("the failure leaves stdout empty, template or not, got:\n%s", w.stdout)
	}
	return nil
}

func (w *invalidCreateWorld) categoryTookUnusedCode() error {
	if w.registryBefore == nil {
		return fmt.Errorf("no registry snapshot was taken by the Given")
	}
	for category, code := range w.registryBefore {
		if code == w.registryAfter {
			return fmt.Errorf("the invalid-create code %d is already assigned to %v — it must be previously unused", w.registryAfter, category)
		}
	}
	// And it is one code, reached through the same registry every category uses.
	if w.registryAfter != ExitCode(InvalidCreate) {
		return fmt.Errorf("the category must map to exactly one code, got %d and %d", w.registryAfter, ExitCode(InvalidCreate))
	}
	return nil
}

func (w *invalidCreateWorld) existingCategoriesKeptCodes() error {
	if w.registryBefore == nil {
		return fmt.Errorf("no registry snapshot was taken by the Given")
	}
	for category, before := range w.registryBefore {
		if now := ExitCode(category); now != before {
			return fmt.Errorf("%v was renumbered from %d to %d — adding a category must renumber nothing", category, before, now)
		}
	}
	return nil
}
