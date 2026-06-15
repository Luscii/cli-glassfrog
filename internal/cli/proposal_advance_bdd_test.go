package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/Luscii/cli-glassfrog/internal/apiclient"
	"github.com/cucumber/godog"
)

// TestProposalProposeFeatures runs the executable acceptance for Advance to Circulation
// (057): the `proposal propose <prp-id>` transition, driven through the shared
// proposalSeam over a fake base transport so every scenario runs offline (no real
// network, ~/.glassfrogrc, pipe, or filesystem). Its Paths name ONLY this spec's feature
// file — never the features/ directory — so the suite reports its own independent
// scenario count and cannot disturb another suite (LEARNINGS: a suite points at its own
// feature file). The 4 @validation scenarios stay @wip (held for the validate skill) and
// are skipped by the ~@wip filter.
func TestProposalProposeFeatures(t *testing.T) {
	suite := godog.TestSuite{
		ScenarioInitializer: initializeProposalProposeScenario,
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"../../features/proposal-write-flow/advance-to-circulation.feature"},
			Tags:     "~@wip",
			TestingT: t,
		},
	}
	if suite.Run() != 0 {
		t.Fatal("non-zero status: advance-to-circulation feature scenarios failed")
	}
}

// proposalProposeWorld is the per-scenario state: the connection context and fake
// transport assembled from the Given steps, plus the captured outcome/exit-code/streams
// of the When run. Everything is injected — no step touches the real network, env, or
// home. The transport is the concrete tensionTransport so a step can read the recorded
// method/body/Content-Type/If-Match/call-count.
type proposalProposeWorld struct {
	ctx       apiclient.ConnectionContext
	transport *tensionTransport
	secret    string

	outcome  Outcome
	exitCode int
	stdout   string
	stderr   string
}

func initializeProposalProposeScenario(sc *godog.ScenarioContext) {
	w := &proposalProposeWorld{}
	sc.Before(func(ctx context.Context, _ *godog.Scenario) (context.Context, error) {
		*w = proposalProposeWorld{
			// A 200 advance (the proposal now proposed_outside_meeting) is the default;
			// per-scenario Given steps override the transport/context (transition-not-allowed
			// 422, unknown 404, Premium 403, unreachable, rate-limit 429, no token).
			transport: &tensionTransport{status: http.StatusOK, body: proposalAdvancedBody},
			secret:    meSecretToken,
		}
		w.ctx = validMeContext()
		return ctx, nil
	})

	// --- Givens ---
	sc.Step(`^a complete connection context with a stored token$`, w.completeContext)
	sc.Step(`^a draft proposal "([^"]*)" whose available transitions include "([^"]*)"$`, w.draftCanBeProposedWithTransitions)
	sc.Step(`^a draft proposal "([^"]*)" that can be proposed$`, w.draftCanBeProposed)
	sc.Step(`^a proposal "([^"]*)" for which "([^"]*)" is not currently allowed$`, w.transitionNotAllowed)
	sc.Step(`^no proposal "([^"]*)" exists$`, w.proposalNotFound)
	sc.Step(`^the organization does not have async proposals enabled$`, w.premiumNotEnabled)
	sc.Step(`^the API endpoint is unreachable$`, w.endpointUnreachable)
	sc.Step(`^the API answers the propose request with 429$`, w.rateLimited)
	sc.Step(`^no usable token is available to the CLI$`, w.noToken)

	// --- When ---
	sc.Step(`^(?:an agent|a practitioner) runs "glassfrog (.+)"$`, w.runCommand)

	// --- Thens ---
	sc.Step(`^the request will POST to the proposal's propose transition with no body$`, w.postedBodylessToProposeTransition)
	sc.Step(`^the advanced proposal will be printed with status "([^"]*)"$`, w.advancedPrintedWithStatus)
	sc.Step(`^the API will answer (\d+)$`, w.apiAnswered)
	sc.Step(`^the returned proposal will be printed as JSON$`, w.printedAsJSON)
	sc.Step(`^the printed proposal will carry the server-set response deadline$`, w.carriesResponseDeadline)
	sc.Step(`^its response summary will reflect the proposer's implicit no-objection$`, w.reflectsImplicitNoObjection)
	sc.Step(`^the command will not narrate the notification side effects$`, w.noNotificationNarration)
	sc.Step(`^the command will exit with code (\d+)$`, w.exitWithCode)
	sc.Step(`^the command will exit with a non-zero API-error code$`, w.exitNonZeroAPIError)
	sc.Step(`^the command will exit with a non-zero permission code$`, w.exitPermissionCode)
	sc.Step(`^the command will exit with a non-zero network-unavailable code$`, w.exitNetworkUnavailable)
	sc.Step(`^the command will exit with a non-zero rate-limit code$`, w.exitRateLimitCode)
	sc.Step(`^stderr will report that the advance failed and name the HTTP status$`, w.stderrAdvanceFailedNamesStatus)
	sc.Step(`^stderr will report "([^"]*)" and point to "([^"]*)"$`, w.stderrReportsAndPointsTo)
	sc.Step(`^stderr will report a usage error naming the required proposal id$`, w.stderrUsageNamingProposalID)
	sc.Step(`^stderr will report a usage error and name the rejected output value "([^"]*)"$`, w.stderrReportsRejectedOutput)
	sc.Step(`^stderr will report the transport failure by name$`, w.transportFailureNamed)
	sc.Step(`^no plan-specific message will be added$`, w.noPlanSpecificMessage)
	sc.Step(`^no request will be sent$`, w.noRequestSent)
	sc.Step(`^the POST will not be automatically retried$`, w.postNotRetried)
}

// --- Given implementations ---

func (w *proposalProposeWorld) completeContext() error { w.ctx = validMeContext(); return nil }

func (w *proposalProposeWorld) draftCanBeProposedWithTransitions(_, _ string) error {
	w.transport = &tensionTransport{status: http.StatusOK, body: proposalAdvancedBody}
	return nil
}

func (w *proposalProposeWorld) draftCanBeProposed(_ string) error {
	w.transport = &tensionTransport{status: http.StatusOK, body: proposalAdvancedBody}
	return nil
}

func (w *proposalProposeWorld) transitionNotAllowed(_, _ string) error {
	w.transport = &tensionTransport{status: http.StatusUnprocessableEntity, body: `{"detail":"transition not allowed"}`}
	return nil
}

func (w *proposalProposeWorld) proposalNotFound(_ string) error {
	w.transport = &tensionTransport{status: http.StatusNotFound, body: `{"detail":"Proposal not found"}`}
	return nil
}

func (w *proposalProposeWorld) premiumNotEnabled() error {
	w.transport = &tensionTransport{status: http.StatusForbidden, body: `{"detail":"async proposals not enabled"}`}
	return nil
}

func (w *proposalProposeWorld) endpointUnreachable() error {
	w.transport = &tensionTransport{netErr: errors.New("dial tcp: connection refused")}
	return nil
}

func (w *proposalProposeWorld) rateLimited() error {
	w.transport = &tensionTransport{status: http.StatusTooManyRequests, body: `{"detail":"rate limited"}`}
	return nil
}

func (w *proposalProposeWorld) noToken() error {
	w.ctx = noTokenContext()
	return nil
}

// --- When implementation ---

// runCommand parses the captured invocation with the package's quote-aware splitArgs
// (propose takes no quoted args, but the shared splitter handles the simple token list
// too), after unescaping the feature file's \" inner quotes, and dispatches it through a
// real root with the `proposal` group attached over a fake seam. It asserts the secret
// token never leaks into output.
func (w *proposalProposeWorld) runCommand(invocation string) error {
	args := splitArgs(strings.ReplaceAll(invocation, `\"`, `"`))
	root := NewRootCommand()
	seam := &fakeProposalSeam{fakeMeSeam: &fakeMeSeam{ctx: w.ctx, transport: w.transport}}
	MustRegister(root, newProposalCommand(seam))

	var out, errb bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&errb)
	w.outcome, _ = Run(root, args)
	w.exitCode = ExitCode(w.outcome)
	w.stdout, w.stderr = out.String(), errb.String()

	if w.secret != "" && strings.Contains(w.stdout+w.stderr, w.secret) {
		return fmt.Errorf("the token leaked into output: stdout=%q stderr=%q", w.stdout, w.stderr)
	}
	return nil
}

// --- Then implementations ---

// postedBodylessToProposeTransition pins ADR-1/ADR-3: exactly one bodyless POST to the
// /proposals/{id}/propose sub-path, no Content-Type, no If-Match, and NO prior GET (the
// transition is server-authorized — the command does not pre-read available_transitions).
func (w *proposalProposeWorld) postedBodylessToProposeTransition() error {
	if w.transport.calls != 1 {
		return fmt.Errorf("an advance is exactly one request (no prior GET), got %d", w.transport.calls)
	}
	if w.transport.lastMethod != http.MethodPost {
		return fmt.Errorf("the advance should POST, got method %q", w.transport.lastMethod)
	}
	if !strings.HasSuffix(w.transport.lastPath, "/propose") || !strings.Contains(w.transport.lastPath, "/proposals/") {
		return fmt.Errorf("the request should target /proposals/{id}/propose, got %q", w.transport.lastPath)
	}
	if w.transport.lastBody != "" {
		return fmt.Errorf("a bodyless POST must send no body, got %q", w.transport.lastBody)
	}
	if w.transport.lastContentType != "" {
		return fmt.Errorf("a bodyless POST must send no Content-Type, got %q", w.transport.lastContentType)
	}
	if w.transport.lastIfMatch != "" {
		return fmt.Errorf("a transition must send no If-Match, got %q", w.transport.lastIfMatch)
	}
	return nil
}

func (w *proposalProposeWorld) advancedPrintedWithStatus(status string) error {
	if !strings.Contains(w.stdout, status) {
		return fmt.Errorf("the advanced proposal should be printed with status %q:\n%s", status, w.stdout)
	}
	return nil
}

// apiAnswered confirms the scenario's canned status reached the wire exactly once — so
// the 404/422/403 paths genuinely exercise a non-2xx response (not a skipped request).
func (w *proposalProposeWorld) apiAnswered(code int) error {
	if w.transport.status != code {
		return fmt.Errorf("the canned API response should be %d, got %d", code, w.transport.status)
	}
	if w.transport.calls != 1 {
		return fmt.Errorf("the API should have been called exactly once, got %d", w.transport.calls)
	}
	return nil
}

func (w *proposalProposeWorld) printedAsJSON() error {
	var doc struct {
		Data struct {
			ID               string `json:"id"`
			Status           string `json:"status"`
			ResponseDeadline string `json:"response_deadline"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(w.stdout), &doc); err != nil {
		return fmt.Errorf("stdout should be valid JSON, got error %v:\n%s", err, w.stdout)
	}
	if doc.Data.ID == "" || doc.Data.Status != "proposed_outside_meeting" {
		return fmt.Errorf("-o json should carry the advanced {data: Proposal} (id + proposed_outside_meeting status), got:\n%s", w.stdout)
	}
	// The raw payload must not carry the human projection's block labels.
	if strings.Contains(w.stdout, "Transitions:") {
		return fmt.Errorf("structured json must not render the human projection:\n%s", w.stdout)
	}
	return nil
}

func (w *proposalProposeWorld) carriesResponseDeadline() error {
	// proposalAdvancedBody's server-set response_deadline, surfaced by the full render.
	if !strings.Contains(w.stdout, "2026-06-22T12:00:00Z") {
		return fmt.Errorf("the printed proposal should carry the server-set response deadline:\n%s", w.stdout)
	}
	return nil
}

func (w *proposalProposeWorld) reflectsImplicitNoObjection() error {
	// proposalAdvancedBody records the proposer's implicit no_objection (1/1), surfaced
	// by the full render's Responses line.
	if !strings.Contains(w.stdout, "1/1 no-objection") {
		return fmt.Errorf("the response summary should reflect the proposer's implicit no-objection (1/1):\n%s", w.stdout)
	}
	return nil
}

func (w *proposalProposeWorld) noNotificationNarration() error {
	// The command surfaces only the returned proposal — it narrates none of the side
	// effects (notifications, the deadline computation). The deadline VALUE is data the
	// server returned (allowed); a "notif…" narration is not.
	if strings.Contains(strings.ToLower(w.stdout+w.stderr), "notif") {
		return fmt.Errorf("the command must not narrate the notification side effects:\nstdout: %s\nstderr: %s", w.stdout, w.stderr)
	}
	return nil
}

func (w *proposalProposeWorld) exitWithCode(code int) error {
	if w.exitCode != code {
		return fmt.Errorf("exit code = %d, want %d (outcome %v)\nstderr: %s", w.exitCode, code, w.outcome, w.stderr)
	}
	return nil
}

func (w *proposalProposeWorld) exitNonZeroAPIError() error {
	if w.outcome != APIError || w.exitCode != 3 {
		return fmt.Errorf("outcome=%v exit=%d, want APIError/3\nstderr: %s", w.outcome, w.exitCode, w.stderr)
	}
	return nil
}

func (w *proposalProposeWorld) exitPermissionCode() error {
	if w.outcome != PermissionError || w.exitCode != 4 {
		return fmt.Errorf("outcome=%v exit=%d, want PermissionError/4\nstderr: %s", w.outcome, w.exitCode, w.stderr)
	}
	return nil
}

func (w *proposalProposeWorld) exitNetworkUnavailable() error {
	if w.outcome != NetworkUnavailable || w.exitCode != 6 {
		return fmt.Errorf("outcome=%v exit=%d, want NetworkUnavailable/6\nstderr: %s", w.outcome, w.exitCode, w.stderr)
	}
	return nil
}

func (w *proposalProposeWorld) exitRateLimitCode() error {
	if w.outcome != RateLimited || w.exitCode != 5 {
		return fmt.Errorf("outcome=%v exit=%d, want RateLimited/5\nstderr: %s", w.outcome, w.exitCode, w.stderr)
	}
	return nil
}

// stderrAdvanceFailedNamesStatus asserts the diagnostic names the HTTP status the
// scenario's transport answered — generic over 422/404/403/429 (the inverse-of-discard
// behaviour: every non-2xx is a real, status-named failure).
func (w *proposalProposeWorld) stderrAdvanceFailedNamesStatus() error {
	if w.transport.status == 0 {
		return fmt.Errorf("the scenario installed no status-bearing transport")
	}
	want := fmt.Sprintf("%d", w.transport.status)
	if !strings.Contains(w.stderr, want) {
		return fmt.Errorf("stderr should name the HTTP status (%s):\n%s", want, w.stderr)
	}
	return nil
}

func (w *proposalProposeWorld) stderrReportsAndPointsTo(report, pointer string) error {
	if !strings.Contains(strings.ToLower(w.stderr), strings.ToLower(report)) {
		return fmt.Errorf("stderr should report %q:\n%s", report, w.stderr)
	}
	if !strings.Contains(w.stderr, pointer) {
		return fmt.Errorf("stderr should point to %q:\n%s", pointer, w.stderr)
	}
	return nil
}

func (w *proposalProposeWorld) stderrUsageNamingProposalID() error {
	if w.outcome != UsageError || w.exitCode != 2 {
		return fmt.Errorf("outcome=%v exit=%d, want UsageError/2\nstderr: %s", w.outcome, w.exitCode, w.stderr)
	}
	// cobra's ExactArgs(1) rejection names the argument requirement ("accepts 1 arg(s),
	// received 0") on the SilenceErrors leaf via dispatch's surfacing.
	if !strings.Contains(w.stderr, "arg") {
		return fmt.Errorf("stderr should report the missing required proposal id argument:\n%s", w.stderr)
	}
	return nil
}

func (w *proposalProposeWorld) stderrReportsRejectedOutput(value string) error {
	if w.outcome != UsageError || w.exitCode != 2 {
		return fmt.Errorf("outcome=%v exit=%d, want UsageError/2\nstderr: %s", w.outcome, w.exitCode, w.stderr)
	}
	if !strings.Contains(w.stderr, value) {
		return fmt.Errorf("stderr should name the rejected output value %q:\n%s", value, w.stderr)
	}
	return nil
}

func (w *proposalProposeWorld) transportFailureNamed() error {
	if w.outcome != NetworkUnavailable {
		return fmt.Errorf("outcome = %v, want NetworkUnavailable\nstderr: %s", w.outcome, w.stderr)
	}
	if strings.TrimSpace(w.stderr) == "" {
		return errors.New("a transport failure should be named on stderr")
	}
	return nil
}

// noPlanSpecificMessage pins ADR-3: the Premium 403 stays a generic permission refusal —
// the command adds no bespoke "not available on your plan" / "upgrade" narration (the
// server's RFC 9457 detail is surfaced unchanged; only a command-added plan-gate message
// is forbidden).
func (w *proposalProposeWorld) noPlanSpecificMessage() error {
	low := strings.ToLower(w.stderr)
	for _, banned := range []string{"not available on your plan", "upgrade", "premium plan"} {
		if strings.Contains(low, banned) {
			return fmt.Errorf("the Premium 403 must stay generic — no plan-specific message, but stderr contains %q:\n%s", banned, w.stderr)
		}
	}
	return nil
}

func (w *proposalProposeWorld) noRequestSent() error {
	if w.transport.calls != 0 {
		return fmt.Errorf("no request should be sent, but the transport was called %d times", w.transport.calls)
	}
	return nil
}

func (w *proposalProposeWorld) postNotRetried() error {
	if w.transport.calls != 1 {
		return fmt.Errorf("a POST 429 must not be retried (no double-advance), want exactly 1 call, got %d", w.transport.calls)
	}
	return nil
}
