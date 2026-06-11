package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/Luscii/cli-glassfrog/internal/apiclient"
	"github.com/Luscii/cli-glassfrog/internal/output"
	"github.com/cucumber/godog"
	"sigs.k8s.io/yaml"
)

// TestOutputAwareFailureRenderingFeatures runs the executable acceptance for
// Output-Aware Failure Rendering (032). Most scenarios drive the single failure
// chokepoint reportFailure directly over a crafted failure and a resolved format —
// the plan's "behavioral BDD over the chokepoint" strategy — so each runs offline
// with no real network or ~/.glassfrogrc. The usage-error scenario drives the real
// dispatch path (Run over an unknown command) to prove the failure-render path
// does not wrap it; the mid-walk scenario drives the `roles` walk over a sequential
// fake transport to prove the partial-data reporter (reportIncompleteWalk) is
// unchanged. Paths name ONLY this spec's feature file — never the features/
// directory — so un-@wip-ping these scenarios cannot disturb another internal/cli
// suite. The three @validation scenarios stay @wip (held for the validate skill)
// and are skipped by the ~@wip filter.
func TestOutputAwareFailureRenderingFeatures(t *testing.T) {
	suite := godog.TestSuite{
		ScenarioInitializer: initializeOutputAwareFailureRenderingScenario,
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"../../features/opaque-failures/output-aware-failure-rendering.feature"},
			Tags:     "~@wip",
			TestingT: t,
		},
	}
	if suite.Run() != 0 {
		t.Fatal("non-zero status: output-aware-failure-rendering feature scenarios failed")
	}
}

// renderCapture holds the observable result of one reportFailure invocation.
type renderCapture struct {
	stdout  string
	stderr  string
	outcome Outcome
}

// failureRenderWorld is the per-scenario state. err is the crafted command-execution
// failure; renders accumulates one capture per format the scenario renders under.
type failureRenderWorld struct {
	err error
	// format is the scenario's resolved output format — the one the shared "When the
	// failure is rendered" step renders under. It defaults to JSON (set in the Before
	// reset) and is overridden by the yaml/full Givens, so the When exercises the
	// format the scenario actually describes rather than a hardcoded one.
	format  output.OutputFormat
	renders map[output.OutputFormat]renderCapture

	// usage-error scenario (dispatch path)
	usageStdout string
	usageExit   int

	// mid-walk scenario (roles walk path)
	walkStdout  string
	walkStderr  string
	walkOutcome Outcome
}

func initializeOutputAwareFailureRenderingScenario(sc *godog.ScenarioContext) {
	w := &failureRenderWorld{}
	sc.Before(func(ctx context.Context, _ *godog.Scenario) (context.Context, error) {
		// Default the resolved format to JSON: the single-render scenarios that mention
		// --output json drive the structured branch without each Given setting it. The
		// yaml/full Givens override w.format explicitly.
		*w = failureRenderWorld{format: output.FormatJSON, renders: map[output.OutputFormat]renderCapture{}}
		renderErrorFn = output.RenderError // restore the production seam between scenarios
		return ctx, nil
	})
	sc.After(func(ctx context.Context, _ *godog.Scenario, _ error) (context.Context, error) {
		renderErrorFn = output.RenderError
		return ctx, nil
	})

	// --- Givens ---
	sc.Step(`^a command run with --output json had failed with a 403 carrying a cause and a next step$`, w.given403WithCauseAndNextStep)
	sc.Step(`^a command run with --output json had failed with a transport error carrying no API body$`, w.givenTransportNoBody)
	sc.Step(`^a command run with --output json had failed with a non-2xx response carrying a JSON error body$`, w.givenNon2xxWithJSONBody)
	sc.Step(`^the same 403 permission failure had been rendered once under full and once under json$`, w.givenSame403UnderFullAndJSON)
	sc.Step(`^an unknown command had been invoked with --output json$`, w.givenUnknownCommandUnderJSON)
	sc.Step(`^a structured failure render had been unable to produce a complete document$`, w.givenRenderCannotComplete)
	sc.Step(`^a paginated read with --output json had rendered some pages then failed mid-walk$`, w.givenMidWalkPartialUnderJSON)
	sc.Step(`^a command run with --output json had failed with a response whose error body is not valid JSON$`, w.givenNon2xxWithNonJSONBody)
	sc.Step(`^a command run with --output yaml had failed with a 429 whose next step is to wait for the reset window and retry$`, w.given429UnderYAML)
	sc.Step(`^a failure whose diagnostic is the internal-error fallback with no next step$`, w.givenInternalFallback)
	sc.Step(`^a command run under the default full format had failed with a transport error$`, w.givenTransportUnderFull)

	// --- Whens ---
	sc.Step(`^the failure is rendered$`, w.renderTheFailure)
	sc.Step(`^each invocation terminates$`, w.noop)
	sc.Step(`^the usage error is reported$`, w.reportUsageError)
	sc.Step(`^the failure is reported$`, w.reportMidWalk)
	sc.Step(`^it is rendered under json and under full$`, w.renderUnderJSONAndFull)

	// --- Thens ---
	sc.Step(`^stdout will carry one unified error envelope as valid JSON$`, w.stdoutCarriesValidJSONEnvelope)
	sc.Step(`^stdout will carry the unified error envelope as valid JSON$`, w.stdoutCarriesValidJSONEnvelope)
	sc.Step(`^the envelope will carry the failure's message, kind, and originating status$`, w.envelopeCarriesMessageKindStatus)
	sc.Step(`^stderr will not also carry the human cause-plus-next-step line$`, w.stderrEmptyUnderJSON)
	sc.Step(`^the envelope will carry the message and kind "([^"]*)"$`, w.envelopeCarriesMessageAndKind)
	sc.Step(`^the envelope will omit the status and body fields$`, w.envelopeOmitsStatusAndBody)
	sc.Step(`^the raw error body will be nested verbatim within the envelope as structured data$`, w.bodyNestedVerbatim)
	sc.Step(`^the system will not re-classify or re-parse it$`, w.bodyNotReparsed)
	sc.Step(`^both will terminate with the same exit code (\d+)$`, w.bothExitWithCode)
	sc.Step(`^only the rendered presentation will differ between the two$`, w.onlyPresentationDiffers)
	sc.Step(`^it will keep its plain-text dispatch form rather than a structured envelope$`, w.usageStaysPlainText)
	sc.Step(`^the failure-render path will not wrap it, because it arose before a command executed in the resolved format$`, w.usageNotWrapped)
	sc.Step(`^nothing partial will be written to stdout$`, w.nothingPartialOnStdout)
	sc.Step(`^the invocation will map to the internal-error exit code (\d+)$`, w.invocationMapsToInternalError)
	sc.Step(`^stdout will carry the partial data document$`, w.midWalkStdoutHasPartial)
	sc.Step(`^the incompleteness note will be written to stderr rather than a second document on stdout$`, w.midWalkNoteOnStderr)
	sc.Step(`^the invocation will exit with the mid-walk failure's non-zero code$`, w.midWalkExitsNonZero)
	sc.Step(`^the envelope will still carry the message, kind, and status$`, w.envelopeCarriesMessageKindStatus)
	sc.Step(`^the envelope will omit the body field rather than failing the render$`, w.envelopeOmitsBodyOnly)
	sc.Step(`^the YAML document will convey that next step as its own distinct, parseable element$`, w.yamlNextStepDistinct)
	sc.Step(`^the cause will remain in its own element$`, w.yamlCauseInOwnElement)
	sc.Step(`^neither render will fabricate a next step$`, w.neitherRenderFabricatesNextStep)
	sc.Step(`^the structured render will omit the distinct next-step field rather than null-keying it$`, w.structuredOmitsNextStepKey)
	sc.Step(`^the cause-plus-next-step line will be written to stderr exactly as the CLI does today$`, w.humanLineOnStderr)
	sc.Step(`^stdout will stay empty$`, w.humanStdoutEmpty)
}

func (w *failureRenderWorld) noop() error { return nil }

// renderOnce drives the chokepoint over the crafted failure in one format and
// records the capture.
func (w *failureRenderWorld) renderOnce(f output.OutputFormat) renderCapture {
	var out, errb bytes.Buffer
	outcome, _ := reportFailure(&out, &errb, f, w.err)
	cap := renderCapture{stdout: out.String(), stderr: errb.String(), outcome: outcome}
	w.renders[f] = cap
	return cap
}

// --- Given implementations ---

func (w *failureRenderWorld) given403WithCauseAndNextStep() error {
	w.err = &apiclient.ResponseError{StatusCode: 403, Body: []byte(`{"detail":"You are not a member of this circle"}`)}
	return nil
}

func (w *failureRenderWorld) givenTransportNoBody() error {
	w.err = &apiclient.TransportError{}
	return nil
}

func (w *failureRenderWorld) givenNon2xxWithJSONBody() error {
	w.err = &apiclient.ResponseError{StatusCode: 500, Body: []byte(`{"title":"Server Error","detail":"boom","trace_id":"abc-123"}`)}
	return nil
}

func (w *failureRenderWorld) givenSame403UnderFullAndJSON() error {
	w.err = &apiclient.ResponseError{StatusCode: 403, Body: []byte(`{"detail":"Forbidden"}`)}
	w.renderOnce(output.FormatFull)
	w.renderOnce(output.FormatJSON)
	return nil
}

func (w *failureRenderWorld) givenUnknownCommandUnderJSON() error {
	// Recorded as the scenario's intent; the dispatch run happens in the When.
	return nil
}

func (w *failureRenderWorld) givenRenderCannotComplete() error {
	w.err = &apiclient.ResponseError{StatusCode: 500, Body: []byte(`{"detail":"boom"}`)}
	// Force the structured render to fail deterministically (the buffer-then-write
	// path), without contriving an un-encodable envelope.
	renderErrorFn = func(output.Format, output.ErrorEnvelope) ([]byte, error) {
		return nil, errors.New("rendering the error envelope failed")
	}
	return nil
}

func (w *failureRenderWorld) givenMidWalkPartialUnderJSON() error {
	// Driven through the real roles walk in the When (reportMidWalk).
	return nil
}

func (w *failureRenderWorld) givenNon2xxWithNonJSONBody() error {
	w.err = &apiclient.ResponseError{StatusCode: 500, Body: []byte(`<html>502 Bad Gateway</html>`)}
	return nil
}

func (w *failureRenderWorld) given429UnderYAML() error {
	w.format = output.FormatYAML
	w.err = &apiclient.ResponseError{StatusCode: 429, Body: []byte(`{"detail":"Too Many Requests"}`)}
	return nil
}

func (w *failureRenderWorld) givenInternalFallback() error {
	w.err = errors.New("an unexpected internal failure")
	return nil
}

func (w *failureRenderWorld) givenTransportUnderFull() error {
	w.format = output.FormatFull
	w.err = &apiclient.TransportError{}
	return nil
}

// --- When implementations ---

func (w *failureRenderWorld) renderTheFailure() error {
	if w.err == nil {
		return errors.New("no crafted failure was set up")
	}
	// Render under the format the scenario resolved (set by its Given), so the When
	// exercises exactly the format the scenario describes — the structured scenarios
	// render json/yaml, the human-line scenario renders full.
	w.renderOnce(w.format)
	return nil
}

func (w *failureRenderWorld) reportUsageError() error {
	root := NewRootCommand()
	var out, errb bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&errb)
	outcome, _ := Run(root, []string{"definitely-not-a-command", "--output", "json"})
	w.usageStdout = out.String() + errb.String()
	w.usageExit = ExitCode(outcome)
	return nil
}

func (w *failureRenderWorld) reportMidWalk() error {
	tr := &seqMeTransport{steps: []seqMeResp{
		{status: 200, body: orgRolesPage("role_00000000000000000000000000000001", "Gathered Role", "c1")},
		{status: 500, body: `{"detail":"boom"}`},
	}}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr, envOutput: "json"}
	var out, errb bytes.Buffer
	outcome, _ := runRoles(rolesConfig{
		seam:   seam,
		reqCtx: context.Background(),
		stdout: &out,
		stderr: &errb,
	})
	w.walkOutcome = outcome
	w.walkStdout = out.String()
	w.walkStderr = errb.String()
	return nil
}

func (w *failureRenderWorld) renderUnderJSONAndFull() error {
	if w.err == nil {
		return errors.New("no crafted failure was set up")
	}
	w.renderOnce(output.FormatJSON)
	w.renderOnce(output.FormatFull)
	return nil
}

// --- Then helpers ---

type envCapture struct {
	Message  string          `json:"message"`
	NextStep string          `json:"next_step"`
	Kind     string          `json:"kind"`
	Status   int             `json:"status"`
	Body     json.RawMessage `json:"body"`
}

func (w *failureRenderWorld) jsonEnvelope() (envCapture, string, error) {
	cap, ok := w.renders[output.FormatJSON]
	if !ok {
		return envCapture{}, "", errors.New("no json render was captured")
	}
	var doc struct {
		Error envCapture `json:"error"`
	}
	if err := json.Unmarshal([]byte(cap.stdout), &doc); err != nil {
		return envCapture{}, cap.stdout, fmt.Errorf("stdout is not a valid JSON envelope: %v\n%s", err, cap.stdout)
	}
	return doc.Error, cap.stdout, nil
}

// --- Then implementations ---

func (w *failureRenderWorld) stdoutCarriesValidJSONEnvelope() error {
	env, raw, err := w.jsonEnvelope()
	if err != nil {
		return err
	}
	if env.Message == "" || env.Kind == "" {
		return fmt.Errorf("the envelope must carry at least a message and kind:\n%s", raw)
	}
	if !strings.HasPrefix(strings.TrimSpace(raw), `{`) || !strings.Contains(raw, `"error"`) {
		return fmt.Errorf("stdout should be the {\"error\":{…}} envelope:\n%s", raw)
	}
	return nil
}

func (w *failureRenderWorld) envelopeCarriesMessageKindStatus() error {
	env, raw, err := w.jsonEnvelope()
	if err != nil {
		return err
	}
	if env.Message == "" {
		return fmt.Errorf("message missing:\n%s", raw)
	}
	if env.Kind == "" {
		return fmt.Errorf("kind missing:\n%s", raw)
	}
	if env.Status == 0 {
		return fmt.Errorf("originating status missing:\n%s", raw)
	}
	return nil
}

func (w *failureRenderWorld) stderrEmptyUnderJSON() error {
	cap := w.renders[output.FormatJSON]
	if cap.stderr != "" {
		return fmt.Errorf("stderr must stay empty under json, got %q", cap.stderr)
	}
	return nil
}

func (w *failureRenderWorld) envelopeCarriesMessageAndKind(wantKind string) error {
	env, raw, err := w.jsonEnvelope()
	if err != nil {
		return err
	}
	if env.Message == "" {
		return fmt.Errorf("message missing:\n%s", raw)
	}
	if env.Kind != wantKind {
		return fmt.Errorf("kind = %q, want %q:\n%s", env.Kind, wantKind, raw)
	}
	return nil
}

func (w *failureRenderWorld) envelopeOmitsStatusAndBody() error {
	_, raw, err := w.jsonEnvelope()
	if err != nil {
		return err
	}
	if strings.Contains(raw, "status") || strings.Contains(raw, "body") {
		return fmt.Errorf("status/body keys must be absent for a transport failure:\n%s", raw)
	}
	return nil
}

func (w *failureRenderWorld) bodyNestedVerbatim() error {
	env, raw, err := w.jsonEnvelope()
	if err != nil {
		return err
	}
	if len(env.Body) == 0 {
		return fmt.Errorf("the body must be carried:\n%s", raw)
	}
	// Nested as structured data (an object), not a quoted JSON string.
	if !strings.HasPrefix(strings.TrimSpace(string(env.Body)), "{") {
		return fmt.Errorf("the body must nest as structured data, not a quoted string: %s", env.Body)
	}
	if strings.Contains(raw, `"body": "`) {
		return fmt.Errorf("the body must not be a quoted string:\n%s", raw)
	}
	return nil
}

func (w *failureRenderWorld) bodyNotReparsed() error {
	env, _, err := w.jsonEnvelope()
	if err != nil {
		return err
	}
	// Prove no field was dropped, coerced, or re-classified: the rendered body
	// unmarshals to the same value as the API's original body.
	original := &apiclient.ResponseError{}
	if !errors.As(w.err, &original) {
		return errors.New("the crafted failure should wrap a *ResponseError")
	}
	var got, want any
	if err := json.Unmarshal(env.Body, &got); err != nil {
		return fmt.Errorf("rendered body is not valid JSON: %v", err)
	}
	if err := json.Unmarshal(original.Body, &want); err != nil {
		return fmt.Errorf("original body is not valid JSON: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		return fmt.Errorf("the body was re-parsed/altered: got %v, want %v", got, want)
	}
	return nil
}

func (w *failureRenderWorld) bothExitWithCode(code int) error {
	full := w.renders[output.FormatFull]
	js := w.renders[output.FormatJSON]
	if ExitCode(full.outcome) != code || ExitCode(js.outcome) != code {
		return fmt.Errorf("exit codes = full:%d json:%d, want both %d", ExitCode(full.outcome), ExitCode(js.outcome), code)
	}
	return nil
}

func (w *failureRenderWorld) onlyPresentationDiffers() error {
	full := w.renders[output.FormatFull]
	js := w.renders[output.FormatJSON]
	// Same outcome (so same exit code), different channel: human → stderr/empty
	// stdout; structured → stdout/empty stderr.
	if full.outcome != js.outcome {
		return fmt.Errorf("the outcome must not differ by format: full=%v json=%v", full.outcome, js.outcome)
	}
	if full.stdout != "" || full.stderr == "" {
		return fmt.Errorf("full should render to stderr with empty stdout: stdout=%q stderr=%q", full.stdout, full.stderr)
	}
	if js.stderr != "" || js.stdout == "" {
		return fmt.Errorf("json should render to stdout with empty stderr: stdout=%q stderr=%q", js.stdout, js.stderr)
	}
	return nil
}

func (w *failureRenderWorld) usageStaysPlainText() error {
	if strings.Contains(w.usageStdout, `"error"`) && json.Valid([]byte(strings.TrimSpace(w.usageStdout))) {
		return fmt.Errorf("a usage error must not render the structured envelope:\n%s", w.usageStdout)
	}
	return nil
}

func (w *failureRenderWorld) usageNotWrapped() error {
	// The usage error is a UsageError(2) on the dispatch path; reportFailure (the
	// command-execution chokepoint) is never reached, so no envelope appears.
	if w.usageExit != 2 {
		return fmt.Errorf("an unknown command must map to the usage exit code 2, got %d", w.usageExit)
	}
	if strings.Contains(w.usageStdout, `"next_step"`) || strings.Contains(w.usageStdout, `"kind"`) {
		return fmt.Errorf("the usage error must not be wrapped in the failure envelope:\n%s", w.usageStdout)
	}
	return nil
}

func (w *failureRenderWorld) nothingPartialOnStdout() error {
	cap := w.renders[output.FormatJSON]
	if cap.stdout != "" {
		return fmt.Errorf("a render that cannot complete must leave stdout empty, got %q", cap.stdout)
	}
	return nil
}

func (w *failureRenderWorld) invocationMapsToInternalError(code int) error {
	cap := w.renders[output.FormatJSON]
	if got := ExitCode(cap.outcome); got != code {
		return fmt.Errorf("exit code = %d, want %d (RuntimeError)", got, code)
	}
	if cap.outcome != RuntimeError {
		return fmt.Errorf("outcome = %v, want RuntimeError", cap.outcome)
	}
	return nil
}

func (w *failureRenderWorld) midWalkStdoutHasPartial() error {
	if !strings.Contains(w.walkStdout, "Gathered Role") || !strings.Contains(w.walkStdout, `"data"`) {
		return fmt.Errorf("stdout should carry the partial {\"data\":[…]} document:\n%s", w.walkStdout)
	}
	return nil
}

func (w *failureRenderWorld) midWalkNoteOnStderr() error {
	if !strings.Contains(w.walkStderr, "incomplete") {
		return fmt.Errorf("the incompleteness note should ride stderr, got %q", w.walkStderr)
	}
	// And there is no second document on stdout — only the partial {data:[…]}.
	if strings.Count(w.walkStdout, `"data"`) != 1 || strings.Contains(w.walkStdout, `"error"`) {
		return fmt.Errorf("stdout must hold exactly the one partial document, no error envelope:\n%s", w.walkStdout)
	}
	return nil
}

func (w *failureRenderWorld) midWalkExitsNonZero() error {
	if ExitCode(w.walkOutcome) == 0 {
		return fmt.Errorf("a mid-walk failure must exit non-zero, got outcome %v", w.walkOutcome)
	}
	return nil
}

func (w *failureRenderWorld) envelopeOmitsBodyOnly() error {
	env, raw, err := w.jsonEnvelope()
	if err != nil {
		return err
	}
	if env.Message == "" || env.Kind == "" || env.Status == 0 {
		return fmt.Errorf("message/kind/status must remain:\n%s", raw)
	}
	if strings.Contains(raw, `"body"`) || len(env.Body) != 0 {
		return fmt.Errorf("the body key must be absent for a non-JSON body:\n%s", raw)
	}
	return nil
}

func (w *failureRenderWorld) yamlNextStepDistinct() error {
	cap := w.renders[output.FormatYAML] // produced by the When (renderTheFailure under yaml)
	var doc struct {
		Error map[string]any `json:"error"`
	}
	if err := yaml.Unmarshal([]byte(cap.stdout), &doc); err != nil {
		return fmt.Errorf("the YAML failure document should parse: %v\n%s", err, cap.stdout)
	}
	ns, ok := doc.Error["next_step"]
	if !ok || fmt.Sprint(ns) == "" {
		return fmt.Errorf("next_step should be a distinct, populated element:\n%s", cap.stdout)
	}
	return nil
}

func (w *failureRenderWorld) yamlCauseInOwnElement() error {
	cap := w.renders[output.FormatYAML]
	var doc struct {
		Error map[string]any `json:"error"`
	}
	if err := yaml.Unmarshal([]byte(cap.stdout), &doc); err != nil {
		return fmt.Errorf("the YAML failure document should parse: %v", err)
	}
	msg, ok := doc.Error["message"]
	if !ok || fmt.Sprint(msg) == "" {
		return fmt.Errorf("the cause should remain in its own message element:\n%s", cap.stdout)
	}
	if fmt.Sprint(msg) == fmt.Sprint(doc.Error["next_step"]) {
		return errors.New("message and next_step must be distinct elements, not the same value")
	}
	return nil
}

func (w *failureRenderWorld) neitherRenderFabricatesNextStep() error {
	js := w.renders[output.FormatJSON]
	full := w.renders[output.FormatFull]
	if strings.Contains(js.stdout, "next_step") {
		return fmt.Errorf("the json render must not fabricate a next step:\n%s", js.stdout)
	}
	// The human line is the cause alone — no " — <next step>" separator.
	if strings.Contains(full.stderr, " — ") {
		return fmt.Errorf("the human render must not fabricate a next step:\n%s", full.stderr)
	}
	return nil
}

func (w *failureRenderWorld) structuredOmitsNextStepKey() error {
	js := w.renders[output.FormatJSON]
	if strings.Contains(js.stdout, "next_step") {
		return fmt.Errorf("the next_step key must be omitted (not null-keyed):\n%s", js.stdout)
	}
	return nil
}

func (w *failureRenderWorld) humanLineOnStderr() error {
	cap := w.renders[output.FormatFull] // produced by the When (renderTheFailure under full)
	if cap.stderr == "" {
		return errors.New("the cause-plus-next-step line should be written to stderr")
	}
	// Byte-stable with renderDiagnostic over the same refined diagnostic.
	d := Diagnose(refineClientError(w.err))
	want := renderDiagnostic(d) + "\n"
	if cap.stderr != want {
		return fmt.Errorf("stderr = %q, want the unchanged line %q", cap.stderr, want)
	}
	return nil
}

func (w *failureRenderWorld) humanStdoutEmpty() error {
	cap := w.renders[output.FormatFull]
	if cap.stdout != "" {
		return fmt.Errorf("stdout must stay empty on the human path, got %q", cap.stdout)
	}
	return nil
}
