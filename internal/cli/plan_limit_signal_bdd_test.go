package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/Luscii/cli-glassfrog/internal/apiclient"
	"github.com/Luscii/cli-glassfrog/internal/output"
	"github.com/cucumber/godog"
)

// TestPlanLimitSignalFeatures runs the executable acceptance for Plan-Limit
// Signal (061). Every scenario drives the single failure chokepoint reportFailure
// directly over a crafted *ResponseError carrying the failed operation's
// method/path (T001) and a resolved format — the same "behavioral BDD over the
// chokepoint" strategy the sibling diagnostic/rendering suites use — so each runs
// offline with no real network or ~/.glassfrogrc. Paths name ONLY this spec's
// feature file (never the features/ directory), so un-@wip-ping these scenarios
// cannot disturb another internal/cli suite (LEARNINGS). The two @validation
// scenarios stay @wip (held for the validate skill) and are skipped by the ~@wip
// filter.
func TestPlanLimitSignalFeatures(t *testing.T) {
	suite := godog.TestSuite{
		ScenarioInitializer: initializePlanLimitSignalScenario,
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"../../features/unsignalled-plan-limits/plan-limit-signal.feature"},
			Tags:     "~@wip",
			TestingT: t,
		},
	}
	if suite.Run() != 0 {
		t.Fatal("non-zero status: plan-limit-signal feature scenarios failed")
	}
}

// planLimitWorld is the per-scenario state. err is the crafted command-execution
// failure (a *ResponseError carrying the gated operation's method/path); renders
// accumulates one capture per format the scenario rendered under (renderCapture is
// shared with the sibling failure-rendering suite — same package).
type planLimitWorld struct {
	err     error
	renders map[output.OutputFormat]renderCapture
}

// planLimitFormats is every --output format value (020): the two human formats
// (full/compact → stderr) and the two structured formats (json/yaml → stdout).
var planLimitFormats = []output.OutputFormat{
	output.FormatFull, output.FormatCompact, output.FormatJSON, output.FormatYAML,
}

func initializePlanLimitSignalScenario(sc *godog.ScenarioContext) {
	w := &planLimitWorld{}
	sc.Before(func(ctx context.Context, _ *godog.Scenario) (context.Context, error) {
		*w = planLimitWorld{renders: map[output.OutputFormat]renderCapture{}}
		return ctx, nil
	})

	// --- Givens ---
	// A rejection by method + path + status (optionally noting an invalid change for
	// the 422 case). The method/path are the concrete request identity 060's
	// recognizer matches against; the status drives whether the 403 branch fires.
	sc.Step(`^the [\w-]+ operation (GET|POST) (\S+) had been rejected with HTTP status (\d+)(?: for an invalid change)?$`, w.givenRejected)
	// "recognized as a plan-limit 403" is the same crafted failure: a 403 from a
	// gated operation (recognition is keyed on the operation, not a flag).
	sc.Step(`^the [\w-]+ operation (GET|POST) (\S+) had been recognized as a plan-limit 403$`, w.givenRecognizedPlanLimit)
	sc.Step(`^the operation is a known plan-gated operation$`, w.noop)
	sc.Step(`^the rejection was a genuine permission denial unrelated to the plan$`, w.noop)
	sc.Step(`^the ai_integration gate kind is modeled but reached by no command$`, w.givenAIIntegrationModeled)

	// --- Whens ---
	sc.Step(`^the failure is rendered$`, w.renderAcrossChannels)
	sc.Step(`^the failure is rendered under the full format$`, w.renderUnderFull)
	sc.Step(`^the failure is rendered under the json format$`, w.renderUnderJSON)
	sc.Step(`^the failure is rendered under each output format$`, w.renderUnderEachFormat)
	sc.Step(`^commands are run$`, w.renderAcrossChannels)

	// --- Thens ---
	sc.Step(`^the diagnostic will name Premium async proposals as the gating feature$`, w.diagnosticNamesGate)
	sc.Step(`^it will state the operation may not be available on the organization's plan$`, w.statesMayNotBeAvailable)
	sc.Step(`^it will point the caller to verify the plan includes Premium async proposals$`, w.pointsToVerifyPlan)
	sc.Step(`^it will frame the plan limit as a possibility, not a certainty$`, w.framesPossibility)
	sc.Step(`^the diagnostic will be the generic permission denial$`, w.diagnosticIsGenericPermission)
	sc.Step(`^it will name no gating feature$`, w.namesNoGatingFeature)
	sc.Step(`^the structured envelope will carry no feature element$`, w.envelopeHasNoFeature)
	sc.Step(`^the diagnostic will carry no plan-limit wording$`, w.carriesNoPlanLimitWording)
	sc.Step(`^every invocation will terminate with the permission exit code (\d+)$`, w.everyFormatExitsWith)
	sc.Step(`^only the rendered presentation will differ between formats$`, w.onlyPresentationDiffers)
	sc.Step(`^no failure will render an ai_integration plan-limit message today$`, w.noAIIntegrationMessageToday)
	sc.Step(`^the signal will be ready to name that gate if such a command is later added$`, w.signalReadyForAIIntegration)
	sc.Step(`^the error envelope on stdout will carry a distinct feature element naming Premium async proposals$`, w.envelopeHasDistinctFeature)
	sc.Step(`^the gate name will not be folded only into the message text$`, w.gateNotFoldedOnlyIntoMessage)
	sc.Step(`^the diagnostic will frame the plan limit as a possibility$`, w.framesPossibility)
	sc.Step(`^it will note the rejection may instead be a permission issue$`, w.notesMayBePermission)
	sc.Step(`^it will not state the plan is certainly insufficient$`, w.neverAssertsCertainInsufficiency)
}

func (w *planLimitWorld) noop() error { return nil }

// renderOnce drives the chokepoint over the crafted failure in one format and
// records the capture (mirrors the sibling failure-rendering suite).
func (w *planLimitWorld) renderOnce(f output.OutputFormat) renderCapture {
	var out, errb bytes.Buffer
	outcome, _ := reportFailure(&out, &errb, f, w.err)
	cap := renderCapture{stdout: out.String(), stderr: errb.String(), outcome: outcome}
	w.renders[f] = cap
	return cap
}

// --- Given implementations ---

func (w *planLimitWorld) givenRejected(method, path, status string) error {
	code, err := strconv.Atoi(status)
	if err != nil {
		return fmt.Errorf("unparseable HTTP status %q: %w", status, err)
	}
	return w.craft(method, path, code)
}

func (w *planLimitWorld) givenRecognizedPlanLimit(method, path string) error {
	return w.craft(method, path, 403)
}

func (w *planLimitWorld) givenAIIntegrationModeled() error {
	// The ai_integration gate is modeled but reached by no registered operation
	// (060 ADR-3). Craft a 403 from a plausible agent endpoint that is NOT in the
	// gated registry, so recognition returns GateNone and no ai_integration
	// plan-limit message is produced today.
	return w.craft("POST", "/agents", 403)
}

func (w *planLimitWorld) craft(method, path string, status int) error {
	body := fmt.Sprintf(`{"type":"about:blank","title":"Error","detail":"status %d"}`, status)
	w.err = &apiclient.ResponseError{
		StatusCode: status,
		Method:     method,
		Path:       path,
		Body:       []byte(body),
	}
	return nil
}

// --- When implementations ---

// renderAcrossChannels renders the crafted failure under one human format (full →
// stderr) and one structured format (json → stdout), so a scenario's Thens can
// inspect whichever channel they describe from a single "the failure is rendered".
func (w *planLimitWorld) renderAcrossChannels() error {
	if w.err == nil {
		return errors.New("no crafted failure was set up")
	}
	w.renderOnce(output.FormatFull)
	w.renderOnce(output.FormatJSON)
	return nil
}

func (w *planLimitWorld) renderUnderFull() error {
	if w.err == nil {
		return errors.New("no crafted failure was set up")
	}
	w.renderOnce(output.FormatFull)
	return nil
}

func (w *planLimitWorld) renderUnderJSON() error {
	if w.err == nil {
		return errors.New("no crafted failure was set up")
	}
	w.renderOnce(output.FormatJSON)
	return nil
}

func (w *planLimitWorld) renderUnderEachFormat() error {
	if w.err == nil {
		return errors.New("no crafted failure was set up")
	}
	for _, f := range planLimitFormats {
		w.renderOnce(f)
	}
	return nil
}

// --- Then helpers ---

// humanLine returns the cause-plus-next-step line written to stderr under the full
// (human) format.
func (w *planLimitWorld) humanLine() (string, error) {
	cap, ok := w.renders[output.FormatFull]
	if !ok {
		return "", errors.New("no full (human) render was captured")
	}
	return cap.stderr, nil
}

// jsonEnvelope decodes the structured error envelope rendered to stdout under json.
func (w *planLimitWorld) jsonEnvelope() (map[string]any, string, error) {
	cap, ok := w.renders[output.FormatJSON]
	if !ok {
		return nil, "", errors.New("no json render was captured")
	}
	var doc struct {
		Error map[string]any `json:"error"`
	}
	if err := json.Unmarshal([]byte(cap.stdout), &doc); err != nil {
		return nil, cap.stdout, fmt.Errorf("stdout is not a valid JSON envelope: %v\n%s", err, cap.stdout)
	}
	return doc.Error, cap.stdout, nil
}

// --- Then implementations ---

func (w *planLimitWorld) diagnosticNamesGate() error {
	line, err := w.humanLine()
	if err != nil {
		return err
	}
	if !strings.Contains(line, "Premium async proposals") {
		return fmt.Errorf("the diagnostic should name the gating feature:\n%s", line)
	}
	return nil
}

func (w *planLimitWorld) statesMayNotBeAvailable() error {
	line, err := w.humanLine()
	if err != nil {
		return err
	}
	if !strings.Contains(line, "may not") || !strings.Contains(line, "plan") {
		return fmt.Errorf("the cause should state the operation may not be available on the org's plan:\n%s", line)
	}
	return nil
}

func (w *planLimitWorld) pointsToVerifyPlan() error {
	line, err := w.humanLine()
	if err != nil {
		return err
	}
	if !strings.Contains(line, "verify") || !strings.Contains(line, "plan includes Premium async proposals") {
		return fmt.Errorf("the next step should point the caller to verify the plan includes the feature:\n%s", line)
	}
	return nil
}

func (w *planLimitWorld) framesPossibility() error {
	line, err := w.humanLine()
	if err != nil {
		return err
	}
	if !strings.Contains(line, "may not") {
		return fmt.Errorf("the diagnostic should hedge ('may not'), framing the limit as a possibility:\n%s", line)
	}
	return w.neverAssertsCertainInsufficiency()
}

func (w *planLimitWorld) diagnosticIsGenericPermission() error {
	line, err := w.humanLine()
	if err != nil {
		return err
	}
	if !strings.Contains(line, "check that the configured identity has the required role membership / permission") {
		return fmt.Errorf("a non-recognized 403 should keep the generic permission diagnostic:\n%s", line)
	}
	return nil
}

func (w *planLimitWorld) namesNoGatingFeature() error {
	line, err := w.humanLine()
	if err != nil {
		return err
	}
	if strings.Contains(line, "Premium async proposals") {
		return fmt.Errorf("the diagnostic should name no gating feature:\n%s", line)
	}
	return nil
}

func (w *planLimitWorld) envelopeHasNoFeature() error {
	env, raw, err := w.jsonEnvelope()
	if err != nil {
		return err
	}
	if _, present := env["feature"]; present {
		return fmt.Errorf("the structured envelope should carry no feature element:\n%s", raw)
	}
	return nil
}

func (w *planLimitWorld) carriesNoPlanLimitWording() error {
	line, err := w.humanLine()
	if err != nil {
		return err
	}
	if strings.Contains(line, "Premium async proposals") || strings.Contains(line, "may not include") {
		return fmt.Errorf("a non-403 on a gated op should carry no plan-limit wording:\n%s", line)
	}
	return nil
}

func (w *planLimitWorld) everyFormatExitsWith(code int) error {
	for _, f := range planLimitFormats {
		cap, ok := w.renders[f]
		if !ok {
			return fmt.Errorf("no render captured for format %v", f)
		}
		if cap.outcome != PermissionError || ExitCode(cap.outcome) != code {
			return fmt.Errorf("format %v: outcome=%v exit=%d, want PermissionError/%d", f, cap.outcome, ExitCode(cap.outcome), code)
		}
	}
	return nil
}

func (w *planLimitWorld) onlyPresentationDiffers() error {
	human := []output.OutputFormat{output.FormatFull, output.FormatCompact}
	structured := []output.OutputFormat{output.FormatJSON, output.FormatYAML}
	// Same outcome across every format — only the channel/content differs.
	for _, f := range planLimitFormats {
		if w.renders[f].outcome != PermissionError {
			return fmt.Errorf("format %v changed the outcome to %v", f, w.renders[f].outcome)
		}
	}
	for _, f := range human {
		cap := w.renders[f]
		if cap.stdout != "" || cap.stderr == "" {
			return fmt.Errorf("human format %v should render to stderr with empty stdout: stdout=%q stderr=%q", f, cap.stdout, cap.stderr)
		}
	}
	for _, f := range structured {
		cap := w.renders[f]
		if cap.stderr != "" || cap.stdout == "" {
			return fmt.Errorf("structured format %v should render to stdout with empty stderr: stdout=%q stderr=%q", f, cap.stdout, cap.stderr)
		}
	}
	return nil
}

func (w *planLimitWorld) noAIIntegrationMessageToday() error {
	line, err := w.humanLine()
	if err != nil {
		return err
	}
	if strings.Contains(line, "AI Integration") {
		return fmt.Errorf("no ai_integration plan-limit message should render today (no command reaches that gate):\n%s", line)
	}
	// And it falls through to the generic permission diagnostic, not a plan-limit one.
	if strings.Contains(line, "may not include") {
		return fmt.Errorf("an unregistered operation must not produce plan-limit wording:\n%s", line)
	}
	return nil
}

func (w *planLimitWorld) signalReadyForAIIntegration() error {
	// The display-name mapping already carries the ai_integration gate, so the
	// signal can name it the moment a command reaches it (061 owns the wording,
	// 060 ADR-3 models the kind).
	if got := featureGateDisplayName(apiclient.GateAIIntegration); got == "" {
		return errors.New("featureGateDisplayName(GateAIIntegration) is empty — the signal is not ready to name that gate")
	}
	return nil
}

func (w *planLimitWorld) envelopeHasDistinctFeature() error {
	env, raw, err := w.jsonEnvelope()
	if err != nil {
		return err
	}
	feat, ok := env["feature"].(string)
	if !ok {
		return fmt.Errorf("the envelope should carry a distinct feature element:\n%s", raw)
	}
	if feat != "Premium async proposals" {
		return fmt.Errorf("the feature element = %q, want %q:\n%s", feat, "Premium async proposals", raw)
	}
	return nil
}

func (w *planLimitWorld) gateNotFoldedOnlyIntoMessage() error {
	env, raw, err := w.jsonEnvelope()
	if err != nil {
		return err
	}
	// The gate must be readable as its own parseable element, not ONLY recoverable
	// by parsing the message prose (ADR-4). message also names the gate, but feature
	// stands alone.
	if _, present := env["feature"]; !present {
		return fmt.Errorf("the gate must be its own element, not folded only into the message:\n%s", raw)
	}
	msg, _ := env["message"].(string)
	if msg == "" {
		return fmt.Errorf("the message should still carry the cause prose:\n%s", raw)
	}
	return nil
}

func (w *planLimitWorld) notesMayBePermission() error {
	line, err := w.humanLine()
	if err != nil {
		return err
	}
	if !strings.Contains(line, "may instead") || !strings.Contains(strings.ToLower(line), "permission") {
		return fmt.Errorf("the diagnostic should note the rejection may instead be a permission issue:\n%s", line)
	}
	return nil
}

func (w *planLimitWorld) neverAssertsCertainInsufficiency() error {
	line, err := w.humanLine()
	if err != nil {
		return err
	}
	for _, certain := range []string{"upgrade", "not available on your plan", "is not available", "certainly", "is insufficient", "must purchase"} {
		if strings.Contains(strings.ToLower(line), certain) {
			return fmt.Errorf("the diagnostic must never assert a certain insufficiency or instruct an upgrade (found %q):\n%s", certain, line)
		}
	}
	return nil
}
