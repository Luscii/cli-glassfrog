package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/Luscii/cli-glassfrog/internal/apiclient"
	"github.com/cucumber/godog"
)

// TestDiagnosticNormalizationFeatures runs the executable acceptance for
// Diagnostic Normalization (031). Every "When the failure is normalized" step
// drives the pure Diagnose normalizer directly over a crafted error — no
// transport, clock, or retry — which is exactly what "no additional wait or
// retry" and "the most-specific category wins" assert. The decode→exit-3 scenario
// drives the `me` read over a fake base transport returning a 2xx body that will
// not decode, so the failure→exit-code path runs end-to-end offline. Its Paths
// name ONLY this spec's feature file (never the features/ directory) so
// un-@wip-ping these scenarios cannot disturb another internal/cli suite, and the
// suite reports its own independent scenario count (LEARNINGS). The @validation
// scenarios stay @wip (held for the validate skill) and are skipped by ~@wip.
func TestDiagnosticNormalizationFeatures(t *testing.T) {
	suite := godog.TestSuite{
		ScenarioInitializer: initializeDiagnosticNormalizationScenario,
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"../../features/opaque-failures/diagnostic-normalization.feature"},
			Tags:     "~@wip",
			TestingT: t,
		},
	}
	if suite.Run() != 0 {
		t.Fatal("non-zero status: diagnostic-normalization feature scenarios failed")
	}
}

// diagWorld is the per-scenario state. It holds the crafted error(s) handed to
// Diagnose, the produced Diagnostic(s), and — for the exit-code scenario — the
// `me` run's captured exit code. usedTransport stays false for every pure-Diagnose
// scenario, pinning "no wait or retry while normalizing".
type diagWorld struct {
	err  error
	d    Diagnostic
	dSet bool

	// 401-vs-403 pair: two errors normalized in one scenario.
	err401, err403 error
	d401, d403     Diagnostic

	exitCode      int
	usedTransport bool
}

func initializeDiagnosticNormalizationScenario(sc *godog.ScenarioContext) {
	w := &diagWorld{}
	sc.Before(func(ctx context.Context, _ *godog.Scenario) (context.Context, error) {
		*w = diagWorld{}
		return ctx, nil
	})

	// --- Givens ---
	sc.Step(`^a command's request had failed with a connection-refused transport error$`, w.givenTransportError)
	sc.Step(`^a typed API error had HTTP status (\d+) with a detail of "([^"]*)"$`, w.givenTypedStatusDetail)
	sc.Step(`^a typed API error had HTTP status (\d+)$`, w.givenTypedStatus)
	sc.Step(`^a typed API error had status (\d+)$`, w.givenTypedStatus)
	sc.Step(`^a typed API error had HTTP status (\d+) with no detail or title$`, w.givenTypedStatusNoDetail)
	sc.Step(`^a typed API error had HTTP status (\d+) surfaced after rate-limit retries were exhausted$`, w.givenTypedStatus)
	sc.Step(`^one typed API error had status (\d+) and another had status (\d+)$`, w.givenTwoTypedStatuses)
	sc.Step(`^a decode error had occurred on a 2xx response whose body would not decode$`, w.givenDecodeError)
	sc.Step(`^a command had received a 2xx response whose body would not decode$`, w.givenCommandUndecodable2xx)
	sc.Step(`^a successful 2xx outcome had reached the normalizer$`, w.givenSuccess)
	sc.Step(`^a failure the normalizer cannot map to a transport, decode, typed-API, or usage family$`, w.givenUnrecognizedFailure)

	// --- Whens ---
	sc.Step(`^the failure is normalized$`, w.whenNormalized)
	sc.Step(`^each failure is normalized$`, w.whenEachNormalized)
	sc.Step(`^it is processed$`, w.whenNormalized)
	sc.Step(`^the command maps the failure to an exit code$`, w.whenCommandMapsExitCode)

	// --- Thens ---
	sc.Step(`^the diagnostic's category will be network-unavailable$`, w.thenCategory(NetworkUnavailable))
	sc.Step(`^the diagnostic's category will be permission/authorization$`, w.thenCategory(PermissionError))
	sc.Step(`^the diagnostic's category will be general API error$`, w.thenCategory(APIError))
	sc.Step(`^the diagnostic's category will be rate-limited$`, w.thenCategory(RateLimited))
	sc.Step(`^the cause will name that the API could not be reached$`, w.thenCauseContains("request failed"))
	sc.Step(`^the next step will point the caller to check connectivity and the configured endpoint$`, w.thenNextStepContains("connectivity"))
	sc.Step(`^the cause will be the API's detail text$`, w.thenCauseIsAPIDetail)
	sc.Step(`^the next step will point the caller toward the required membership or permission$`, w.thenNextStepMentionsMembership)
	sc.Step(`^the cause will be derived from the HTTP status rather than invented$`, w.thenCauseFromStatus)
	sc.Step(`^no fabricated next step will be attached$`, w.thenNoFabricatedNextStep)
	sc.Step(`^the next step will point the caller to wait for the rate-limit window to reset and retry$`, w.thenNextStepResetWindow)
	sc.Step(`^no additional wait or retry will be performed$`, w.thenNoWaitOrRetry)
	sc.Step(`^both diagnostics will share the permission/authorization category$`, w.thenBothPermission)
	sc.Step(`^the 401 next step will point the caller to verify the configured API token$`, w.then401NextStepToken)
	sc.Step(`^the 403 next step will point the caller to the required role membership or permission$`, w.then403NextStepMembership)
	sc.Step(`^the cause will name that the API responded but its body could not be read as expected$`, w.thenCauseShapeMismatch)
	sc.Step(`^the category will not be general API error$`, w.thenCategoryNot(APIError))
	sc.Step(`^the command will exit with code (\d+)$`, w.thenExitCode)
	sc.Step(`^no diagnostic will be produced$`, w.thenNoDiagnostic)
	sc.Step(`^the success will pass through untouched$`, w.thenSuccessUntouched)
	sc.Step(`^a diagnostic with the internal-error category will be produced$`, w.thenCategory(RuntimeError))
	sc.Step(`^the cause will be the failure's own message$`, w.thenCauseIsOwnMessage)
	sc.Step(`^no stack trace will be written$`, w.thenNoStackTrace)
}

// --- Given implementations ---

func (w *diagWorld) givenTransportError() error {
	// The wire cause is unexported; a zero-value *TransportError still renders the
	// "request failed: …" cause Diagnose surfaces, which is what this scenario pins.
	w.err = &apiclient.TransportError{}
	return nil
}

// givenTypedStatusDetail crafts a refined *ProblemError (as reportClientError
// would hand Diagnose) carrying the API's own detail.
func (w *diagWorld) givenTypedStatusDetail(status int, detail string) error {
	w.err = apiclient.ExtractProblem(&apiclient.ResponseError{
		StatusCode: status,
		Body:       []byte(fmt.Sprintf(`{"detail":%q}`, detail)),
	})
	return nil
}

func (w *diagWorld) givenTypedStatus(status int) error {
	w.err = apiclient.ExtractProblem(&apiclient.ResponseError{
		StatusCode: status,
		Body:       []byte(`{"detail":"the API rejected the call"}`),
	})
	return nil
}

// givenTypedStatusNoDetail crafts a non-2xx whose body carries no detail/title,
// so ExtractProblem must synthesize a status-derived fallback (DetailSynthesized).
func (w *diagWorld) givenTypedStatusNoDetail(status int) error {
	w.err = apiclient.ExtractProblem(&apiclient.ResponseError{StatusCode: status, Body: []byte("")})
	return nil
}

func (w *diagWorld) givenTwoTypedStatuses(a, b int) error {
	w.err401 = apiclient.ExtractProblem(&apiclient.ResponseError{StatusCode: a, Body: []byte(`{"detail":"Unauthorized"}`)})
	w.err403 = apiclient.ExtractProblem(&apiclient.ResponseError{StatusCode: b, Body: []byte(`{"detail":"Forbidden"}`)})
	return nil
}

func (w *diagWorld) givenDecodeError() error {
	w.err = &apiclient.DecodeError{StatusCode: 200}
	return nil
}

func (w *diagWorld) givenCommandUndecodable2xx() error {
	// A 2xx whose body is not the JSON the `me` projection expects → a DecodeError
	// surfaces from renderResult and funnels through reportClientError.
	w.err = nil
	return nil
}

func (w *diagWorld) givenSuccess() error {
	w.err = nil
	return nil
}

func (w *diagWorld) givenUnrecognizedFailure() error {
	w.err = errors.New("an unexpected internal failure")
	return nil
}

// --- When implementations ---

func (w *diagWorld) whenNormalized() error {
	w.d = Diagnose(w.err)
	w.dSet = true
	return nil
}

func (w *diagWorld) whenEachNormalized() error {
	w.d401 = Diagnose(w.err401)
	w.d403 = Diagnose(w.err403)
	return nil
}

// whenCommandMapsExitCode drives the `me` read over a fake base transport that
// returns a 2xx body the projection cannot decode, then maps the resulting
// Outcome through the production ExitCode registry — the failure→exit-code path
// end-to-end, offline.
func (w *diagWorld) whenCommandMapsExitCode() error {
	w.usedTransport = true
	root := NewRootCommand()
	tr := &cannedTransport{status: 200, body: "this is not the JSON the me projection expects"}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}
	MustRegister(root, newMeCommand(seam))

	var out, errb bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&errb)
	outcome, _ := Run(root, []string{"me"})
	w.exitCode = ExitCode(outcome)

	if strings.Contains(out.String()+errb.String(), meSecretToken) {
		return fmt.Errorf("the token leaked into output: stdout=%q stderr=%q", out.String(), errb.String())
	}
	return nil
}

// --- Then helpers ---

func (w *diagWorld) thenCategory(want Outcome) func() error {
	return func() error {
		if !w.dSet {
			return errors.New("no diagnostic was produced")
		}
		if w.d.Category != want {
			return fmt.Errorf("category = %v, want %v", w.d.Category, want)
		}
		return nil
	}
}

func (w *diagWorld) thenCategoryNot(notWant Outcome) func() error {
	return func() error {
		if w.d.Category == notWant {
			return fmt.Errorf("category = %v, but it must NOT be %v (a more specific category should win)", w.d.Category, notWant)
		}
		return nil
	}
}

func (w *diagWorld) thenCauseContains(sub string) func() error {
	return func() error {
		if !strings.Contains(w.d.Cause, sub) {
			return fmt.Errorf("cause %q does not mention %q", w.d.Cause, sub)
		}
		return nil
	}
}

func (w *diagWorld) thenNextStepContains(sub string) func() error {
	return func() error {
		if !strings.Contains(w.d.NextStep, sub) {
			return fmt.Errorf("next step %q does not mention %q", w.d.NextStep, sub)
		}
		return nil
	}
}

// --- Then implementations ---

func (w *diagWorld) thenCauseIsAPIDetail() error {
	if !strings.Contains(w.d.Cause, "You are not a member of this circle") {
		return fmt.Errorf("cause %q does not carry the API's own detail", w.d.Cause)
	}
	return nil
}

func (w *diagWorld) thenNextStepMentionsMembership() error {
	lower := strings.ToLower(w.d.NextStep)
	if !strings.Contains(lower, "membership") && !strings.Contains(lower, "permission") {
		return fmt.Errorf("next step %q does not point toward membership or permission", w.d.NextStep)
	}
	return nil
}

func (w *diagWorld) thenCauseFromStatus() error {
	// The 500 had no body detail, so ExtractProblem synthesized a status-derived
	// fallback ("status 500") — the cause is derived from the status, not invented.
	if !strings.Contains(w.d.Cause, "500") {
		return fmt.Errorf("cause %q is not derived from the HTTP status", w.d.Cause)
	}
	return nil
}

func (w *diagWorld) thenNoFabricatedNextStep() error {
	// A residual generic non-2xx keeps the existing generic line; what matters for
	// "no fabricated next step" is that nothing invented beyond the generic recourse
	// is attached. The generic line references the status code only.
	if strings.Contains(strings.ToLower(w.d.NextStep), "membership") {
		return fmt.Errorf("a generic API error must not fabricate a membership hint: %q", w.d.NextStep)
	}
	return nil
}

func (w *diagWorld) thenNextStepResetWindow() error {
	lower := strings.ToLower(w.d.NextStep)
	if !strings.Contains(lower, "reset") {
		return fmt.Errorf("next step %q does not point at the rate-limit window resetting", w.d.NextStep)
	}
	if !strings.Contains(lower, "retry") {
		return fmt.Errorf("next step %q does not mention retrying", w.d.NextStep)
	}
	return nil
}

func (w *diagWorld) thenNoWaitOrRetry() error {
	// Diagnose is pure — a "normalize" step never drives a transport, clock, or
	// retry. usedTransport is only ever set by the exit-code scenario's command run.
	if w.usedTransport {
		return errors.New("normalizing a 429 must not touch a transport (no wait or retry)")
	}
	return nil
}

func (w *diagWorld) thenBothPermission() error {
	if w.d401.Category != PermissionError {
		return fmt.Errorf("401 category = %v, want PermissionError", w.d401.Category)
	}
	if w.d403.Category != PermissionError {
		return fmt.Errorf("403 category = %v, want PermissionError", w.d403.Category)
	}
	return nil
}

func (w *diagWorld) then401NextStepToken() error {
	if !strings.Contains(strings.ToLower(w.d401.NextStep), "token") {
		return fmt.Errorf("401 next step %q does not point at verifying the API token", w.d401.NextStep)
	}
	return nil
}

func (w *diagWorld) then403NextStepMembership() error {
	lower := strings.ToLower(w.d403.NextStep)
	if !strings.Contains(lower, "membership") && !strings.Contains(lower, "permission") {
		return fmt.Errorf("403 next step %q does not point at role membership / permission", w.d403.NextStep)
	}
	// The distinction is load-bearing: the two next steps must actually differ.
	if w.d401.NextStep == w.d403.NextStep {
		return fmt.Errorf("401 and 403 carry identical next steps (%q) — the split did not take", w.d401.NextStep)
	}
	return nil
}

func (w *diagWorld) thenCauseShapeMismatch() error {
	if !strings.Contains(w.d.Cause, "did not match the expected shape") {
		return fmt.Errorf("cause %q does not name the body-shape mismatch", w.d.Cause)
	}
	return nil
}

func (w *diagWorld) thenExitCode(code int) error {
	if w.exitCode != code {
		return fmt.Errorf("exit code = %d, want %d", w.exitCode, code)
	}
	return nil
}

func (w *diagWorld) thenNoDiagnostic() error {
	// A successful outcome normalizes to the Success category — the "no diagnostic"
	// signal — with no cause and no next step.
	if w.d.Category != Success {
		return fmt.Errorf("a success must normalize to Success (no diagnostic), got %v", w.d.Category)
	}
	if w.d.Cause != "" || w.d.NextStep != "" {
		return fmt.Errorf("a success carries no cause/next step, got cause=%q next=%q", w.d.Cause, w.d.NextStep)
	}
	return nil
}

func (w *diagWorld) thenSuccessUntouched() error {
	if w.err != nil {
		return errors.New("the success input was a non-nil error — the scenario set-up is wrong")
	}
	return nil
}

func (w *diagWorld) thenCauseIsOwnMessage() error {
	if w.d.Cause != w.err.Error() {
		return fmt.Errorf("cause = %q, want the failure's own message %q", w.d.Cause, w.err.Error())
	}
	return nil
}

func (w *diagWorld) thenNoStackTrace() error {
	// The fail-safe surfaces the error's own message only — never a goroutine dump.
	combined := w.d.Cause + "\n" + w.d.NextStep
	if strings.Contains(combined, "goroutine ") || strings.Contains(combined, ".go:") {
		return fmt.Errorf("the fail-safe diagnostic leaked a stack trace: %q", combined)
	}
	return nil
}
