package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/Luscii/cli-glassfrog/internal/apiclient"
	"github.com/cucumber/godog"
)

// TestResponseRecordingFeatures runs the executable acceptance for Response Recording
// (058): the `proposal respond <prp-id> --response <value>` consume/respond write, driven
// through the shared proposalSeam over a fake base transport so every scenario runs
// offline (no real network, ~/.glassfrogrc, pipe, or filesystem). Its Paths name ONLY
// this spec's feature file — never the features/ directory — so the suite reports its own
// independent scenario count and cannot disturb another suite (LEARNINGS: a suite points
// at its own feature file). The 3 @validation scenarios stay @wip (held for the validate
// skill) and are skipped by the ~@wip filter.
func TestResponseRecordingFeatures(t *testing.T) {
	suite := godog.TestSuite{
		ScenarioInitializer: initializeResponseRecordingScenario,
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"../../features/proposal-write-flow/response-recording.feature"},
			Tags:     "~@wip",
			TestingT: t,
		},
	}
	if suite.Run() != 0 {
		t.Fatal("non-zero status: response-recording feature scenarios failed")
	}
}

// responseRecordingWorld is the per-scenario state: the connection context and fake
// transport assembled from the Given steps, plus the captured outcome/exit-code/streams
// of the When run. Everything is injected — no step touches the real network, env, or
// home. The transport is the concrete tensionTransport so a step can read the recorded
// method/body/Content-Type/If-Match/call-count.
type responseRecordingWorld struct {
	ctx       apiclient.ConnectionContext
	transport *tensionTransport
	secret    string
	args      []string // the parsed invocation, so a Then can recover the --response value

	outcome  Outcome
	exitCode int
	stdout   string
	stderr   string
}

func initializeResponseRecordingScenario(sc *godog.ScenarioContext) {
	w := &responseRecordingWorld{}
	sc.Before(func(ctx context.Context, _ *godog.Scenario) (context.Context, error) {
		*w = responseRecordingWorld{
			// A 201 recorded response (the proposal still circulating) is the default;
			// per-scenario Given steps override the transport/context (accepted 201, 422
			// already-responded, 403 Premium gate, 404 unknown, 429 rate-limit, no token).
			transport: &tensionTransport{status: http.StatusCreated, body: proposalVoteRecordedBody},
			secret:    meSecretToken,
		}
		w.ctx = validMeContext()
		return ctx, nil
	})

	// --- Givens ---
	sc.Step(`^a complete connection context with a stored token$`, w.completeContext)
	sc.Step(`^the proposal "([^"]*)" is circulating for acceptance$`, w.proposalCirculating)
	sc.Step(`^the proposal "([^"]*)" is awaiting only this member's response$`, w.proposalAwaitingOnlyThis)
	sc.Step(`^the responses endpoint answers that this person has already responded$`, w.alreadyResponded)
	sc.Step(`^the responses endpoint answers that async proposals are not enabled$`, w.premiumNotEnabled)
	sc.Step(`^no proposal "([^"]*)" is visible to the caller$`, w.proposalNotVisible)
	sc.Step(`^the responses endpoint answers the recording with a rate-limit response$`, w.rateLimited)
	sc.Step(`^no usable token is available to the CLI$`, w.noToken)

	// --- When ---
	sc.Step(`^(?:an agent|a practitioner) runs "glassfrog (.+)"$`, w.runCommand)
	sc.Step(`^an agent records a response with "glassfrog (.+)"$`, w.runCommand)

	// --- Thens ---
	sc.Step(`^the request will post the response to the proposal's responses endpoint$`, w.postedToResponsesEndpoint)
	sc.Step(`^the request body will carry "([^"]*)" set to "([^"]*)"$`, w.bodyCarriesKeyValue)
	sc.Step(`^the recorded response will be printed with its "([^"]*)" id and the proposal status$`, w.printedWithIDAndStatus)
	sc.Step(`^the recorded response will be printed$`, w.recordedResponsePrinted)
	sc.Step(`^the structured result will carry the parent proposal status as "([^"]*)"$`, w.structuredStatusIs)
	sc.Step(`^the request will be sent with a JSON content type$`, w.requestJSONContentType)
	sc.Step(`^the request will carry no If-Match precondition$`, w.requestNoIfMatch)
	sc.Step(`^the command will exit with code (\d+)$`, w.exitWithCode)
	sc.Step(`^the command will exit with a non-zero API-error code$`, w.exitNonZeroAPIError)
	sc.Step(`^the command will exit with the permission code$`, w.exitPermissionCode)
	sc.Step(`^the command will exit with the rate-limit code$`, w.exitRateLimitCode)
	sc.Step(`^stderr will report that the recording failed and name the HTTP status$`, w.stderrRecordingFailedNamesStatus)
	sc.Step(`^stderr will report "([^"]*)" and point to "([^"]*)"$`, w.stderrReportsAndPointsTo)
	sc.Step(`^stderr will report that "([^"]*)" is required and list the supported set$`, w.stderrRequiredAndLists)
	sc.Step(`^stderr will report the unsupported value and list the supported set$`, w.stderrUnsupportedAndLists)
	sc.Step(`^stderr will report a usage error and name the rejected output value "([^"]*)"$`, w.stderrReportsRejectedOutput)
	sc.Step(`^no plan-limit-specific interpretation will be added$`, w.noPlanSpecificMessage)
	sc.Step(`^no request will be sent$`, w.noRequestSent)
	sc.Step(`^the recording will not be retried$`, w.recordingNotRetried)
	sc.Step(`^the recording will not be retried, so no duplicate response is recorded$`, w.recordingNotRetried)
	sc.Step(`^the rate-limit will be surfaced on the first occurrence$`, w.rateLimitSurfacedFirst)
}

// --- Given implementations ---

func (w *responseRecordingWorld) completeContext() error { w.ctx = validMeContext(); return nil }

func (w *responseRecordingWorld) proposalCirculating(_ string) error {
	w.transport = &tensionTransport{status: http.StatusCreated, body: proposalVoteRecordedBody}
	return nil
}

// proposalAwaitingOnlyThis installs the 201 body whose parent proposal_status is
// `accepted` — this response closed the consent window (the auto-acceptance signal).
func (w *responseRecordingWorld) proposalAwaitingOnlyThis(_ string) error {
	w.transport = &tensionTransport{status: http.StatusCreated, body: proposalVoteAcceptedBody}
	return nil
}

func (w *responseRecordingWorld) alreadyResponded() error {
	w.transport = &tensionTransport{status: http.StatusUnprocessableEntity, body: `{"detail":"already responded"}`}
	return nil
}

func (w *responseRecordingWorld) premiumNotEnabled() error {
	w.transport = &tensionTransport{status: http.StatusForbidden, body: `{"detail":"async proposals not enabled"}`}
	return nil
}

func (w *responseRecordingWorld) proposalNotVisible(_ string) error {
	w.transport = &tensionTransport{status: http.StatusNotFound, body: `{"detail":"Proposal not found"}`}
	return nil
}

func (w *responseRecordingWorld) rateLimited() error {
	w.transport = &tensionTransport{status: http.StatusTooManyRequests, body: `{"detail":"rate limited"}`}
	return nil
}

func (w *responseRecordingWorld) noToken() error {
	w.ctx = noTokenContext()
	return nil
}

// --- When implementation ---

// runCommand parses the captured invocation with the package's quote-aware splitArgs,
// after unescaping the feature file's \" inner quotes, and dispatches it through a real
// root with the `proposal` group attached over a fake seam. It asserts the secret token
// never leaks into output.
func (w *responseRecordingWorld) runCommand(invocation string) error {
	w.args = splitArgs(strings.ReplaceAll(invocation, `\"`, `"`))
	root := NewRootCommand()
	seam := &fakeProposalSeam{fakeMeSeam: &fakeMeSeam{ctx: w.ctx, transport: w.transport}}
	MustRegister(root, newProposalCommand(seam))

	var out, errb bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&errb)
	w.outcome, _ = Run(root, w.args)
	w.exitCode = ExitCode(w.outcome)
	w.stdout, w.stderr = out.String(), errb.String()

	if w.secret != "" && strings.Contains(w.stdout+w.stderr, w.secret) {
		return fmt.Errorf("the token leaked into output: stdout=%q stderr=%q", w.stdout, w.stderr)
	}
	return nil
}

// --- Then implementations ---

// postedToResponsesEndpoint pins the request shape: exactly one POST to the
// /proposals/{id}/responses sub-path.
func (w *responseRecordingWorld) postedToResponsesEndpoint() error {
	if w.transport.calls != 1 {
		return fmt.Errorf("recording a response is exactly one request, got %d", w.transport.calls)
	}
	if w.transport.lastMethod != http.MethodPost {
		return fmt.Errorf("the recording should POST, got method %q", w.transport.lastMethod)
	}
	if !strings.HasSuffix(w.transport.lastPath, "/responses") || !strings.Contains(w.transport.lastPath, "/proposals/") {
		return fmt.Errorf("the request should target /proposals/{id}/responses, got %q", w.transport.lastPath)
	}
	return nil
}

// bodyCarriesKeyValue pins the nested {response:{value}} body carries the given key set
// to the given value, with no person field.
func (w *responseRecordingWorld) bodyCarriesKeyValue(key, value string) error {
	want := fmt.Sprintf(`"%s":"%s"`, key, value)
	if !strings.Contains(w.transport.lastBody, want) {
		return fmt.Errorf("the request body should carry %s, got %q", want, w.transport.lastBody)
	}
	for _, banned := range []string{"person", "responder", "actor"} {
		if strings.Contains(w.transport.lastBody, banned) {
			return fmt.Errorf("the body must carry no person field, but contains %q: %s", banned, w.transport.lastBody)
		}
	}
	return nil
}

func (w *responseRecordingWorld) printedWithIDAndStatus(idPrefix string) error {
	if !strings.Contains(w.stdout, idPrefix) {
		return fmt.Errorf("the recorded response should be printed with its %q id:\n%s", idPrefix, w.stdout)
	}
	// proposalVoteRecordedBody's parent proposal_status, surfaced by the render.
	if !strings.Contains(w.stdout, "proposed_outside_meeting") {
		return fmt.Errorf("the recorded response should surface the proposal status:\n%s", w.stdout)
	}
	return nil
}

func (w *responseRecordingWorld) recordedResponsePrinted() error {
	if !strings.Contains(w.stdout, "prr_") {
		return fmt.Errorf("the recorded response should be printed (its prr_ id):\n%s", w.stdout)
	}
	return nil
}

// structuredStatusIs pins that the structured (-o json) result carries the parent
// proposal_status — the load-bearing auto-acceptance signal — set to the given value.
func (w *responseRecordingWorld) structuredStatusIs(status string) error {
	// The structured render pretty-prints, so match the field and value
	// whitespace-tolerantly (`"proposal_status": "accepted"`) rather than as a fixed pair.
	if !strings.Contains(w.stdout, "proposal_status") || !strings.Contains(w.stdout, status) {
		return fmt.Errorf("the structured result should carry proposal_status %q:\n%s", status, w.stdout)
	}
	if strings.Contains(w.stdout, "Proposal status:") {
		return fmt.Errorf("a structured result must not render the human projection:\n%s", w.stdout)
	}
	return nil
}

func (w *responseRecordingWorld) requestJSONContentType() error {
	if w.transport.lastContentType != "application/json" {
		return fmt.Errorf("Content-Type = %q, want application/json", w.transport.lastContentType)
	}
	return nil
}

func (w *responseRecordingWorld) requestNoIfMatch() error {
	if w.transport.lastIfMatch != "" {
		return fmt.Errorf("an append-create must carry no If-Match, got %q", w.transport.lastIfMatch)
	}
	return nil
}

func (w *responseRecordingWorld) exitWithCode(code int) error {
	if w.exitCode != code {
		return fmt.Errorf("exit code = %d, want %d (outcome %v)\nstderr: %s", w.exitCode, code, w.outcome, w.stderr)
	}
	return nil
}

func (w *responseRecordingWorld) exitNonZeroAPIError() error {
	if w.outcome != APIError || w.exitCode != 3 {
		return fmt.Errorf("outcome=%v exit=%d, want APIError/3\nstderr: %s", w.outcome, w.exitCode, w.stderr)
	}
	return nil
}

func (w *responseRecordingWorld) exitPermissionCode() error {
	if w.outcome != PermissionError || w.exitCode != 4 {
		return fmt.Errorf("outcome=%v exit=%d, want PermissionError/4\nstderr: %s", w.outcome, w.exitCode, w.stderr)
	}
	return nil
}

func (w *responseRecordingWorld) exitRateLimitCode() error {
	if w.outcome != RateLimited || w.exitCode != 5 {
		return fmt.Errorf("outcome=%v exit=%d, want RateLimited/5\nstderr: %s", w.outcome, w.exitCode, w.stderr)
	}
	return nil
}

// stderrRecordingFailedNamesStatus asserts the diagnostic names the HTTP status the
// scenario's transport answered — generic over 422/404/403 (every non-2xx is a real,
// status-named failure).
func (w *responseRecordingWorld) stderrRecordingFailedNamesStatus() error {
	if w.transport.status == 0 {
		return errors.New("the scenario installed no status-bearing transport")
	}
	want := fmt.Sprintf("%d", w.transport.status)
	if !strings.Contains(w.stderr, want) {
		return fmt.Errorf("stderr should name the HTTP status (%s):\n%s", want, w.stderr)
	}
	return nil
}

func (w *responseRecordingWorld) stderrReportsAndPointsTo(report, pointer string) error {
	if !strings.Contains(strings.ToLower(w.stderr), strings.ToLower(report)) {
		return fmt.Errorf("stderr should report %q:\n%s", report, w.stderr)
	}
	if !strings.Contains(w.stderr, pointer) {
		return fmt.Errorf("stderr should point to %q:\n%s", pointer, w.stderr)
	}
	return nil
}

// stderrRequiredAndLists pins the missing-value rejection: a UsageError(2) naming the
// flag as required and listing the supported set.
func (w *responseRecordingWorld) stderrRequiredAndLists(flag string) error {
	if w.outcome != UsageError || w.exitCode != 2 {
		return fmt.Errorf("outcome=%v exit=%d, want UsageError/2\nstderr: %s", w.outcome, w.exitCode, w.stderr)
	}
	if !strings.Contains(w.stderr, flag) || !strings.Contains(strings.ToLower(w.stderr), "required") {
		return fmt.Errorf("stderr should report %q as required:\n%s", flag, w.stderr)
	}
	return w.assertSupportedSetListed()
}

// stderrUnsupportedAndLists pins the unsupported-value rejection: a UsageError(2) naming
// the offending value (recovered from the invocation) and listing the supported set.
func (w *responseRecordingWorld) stderrUnsupportedAndLists() error {
	if w.outcome != UsageError || w.exitCode != 2 {
		return fmt.Errorf("outcome=%v exit=%d, want UsageError/2\nstderr: %s", w.outcome, w.exitCode, w.stderr)
	}
	if value := w.responseFlagValue(); value != "" && !strings.Contains(w.stderr, value) {
		return fmt.Errorf("stderr should name the unsupported value %q:\n%s", value, w.stderr)
	}
	return w.assertSupportedSetListed()
}

// assertSupportedSetListed checks both consent values appear in the usage message.
func (w *responseRecordingWorld) assertSupportedSetListed() error {
	for _, name := range []string{"no_objection", "bring_to_meeting"} {
		if !strings.Contains(w.stderr, name) {
			return fmt.Errorf("stderr should list the supported set (missing %q):\n%s", name, w.stderr)
		}
	}
	return nil
}

// responseFlagValue recovers the --response value from the parsed invocation, so the
// unsupported-value assertion can name it without the step capturing it.
func (w *responseRecordingWorld) responseFlagValue() string {
	for i, a := range w.args {
		if a == "--response" && i+1 < len(w.args) {
			return w.args[i+1]
		}
	}
	return ""
}

func (w *responseRecordingWorld) stderrReportsRejectedOutput(value string) error {
	if w.outcome != UsageError || w.exitCode != 2 {
		return fmt.Errorf("outcome=%v exit=%d, want UsageError/2\nstderr: %s", w.outcome, w.exitCode, w.stderr)
	}
	if !strings.Contains(w.stderr, value) {
		return fmt.Errorf("stderr should name the rejected output value %q:\n%s", value, w.stderr)
	}
	return nil
}

// noPlanSpecificMessage pins the cross-cutting rule: the Premium 403 stays a generic
// permission refusal — the command adds no bespoke plan-limit narration (the server's
// RFC 9457 detail is surfaced unchanged; only a command-added plan-gate message is
// forbidden).
func (w *responseRecordingWorld) noPlanSpecificMessage() error {
	low := strings.ToLower(w.stderr)
	for _, banned := range []string{"not available on your plan", "upgrade", "premium plan"} {
		if strings.Contains(low, banned) {
			return fmt.Errorf("the Premium 403 must stay generic — no plan-specific message, but stderr contains %q:\n%s", banned, w.stderr)
		}
	}
	return nil
}

func (w *responseRecordingWorld) noRequestSent() error {
	if w.transport.calls != 0 {
		return fmt.Errorf("no request should be sent, but the transport was called %d times", w.transport.calls)
	}
	return nil
}

// recordingNotRetried pins §133: a POST is never auto-retried (017's isSafeMethod gate),
// so a rejected/rate-limited recording cannot double-fire — exactly one request is sent.
func (w *responseRecordingWorld) recordingNotRetried() error {
	if w.transport.calls != 1 {
		return fmt.Errorf("a POST must not be retried (no double-record), want exactly 1 call, got %d", w.transport.calls)
	}
	return nil
}

func (w *responseRecordingWorld) rateLimitSurfacedFirst() error {
	if w.outcome != RateLimited {
		return fmt.Errorf("outcome = %v, want RateLimited\nstderr: %s", w.outcome, w.stderr)
	}
	if w.transport.calls != 1 {
		return fmt.Errorf("the rate-limit should surface on the first occurrence (exactly 1 call), got %d", w.transport.calls)
	}
	return nil
}
