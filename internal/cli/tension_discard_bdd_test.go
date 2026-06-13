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

// TestTensionDiscardFeatures runs the executable acceptance for Tension Discard
// (045): the `tension discard <ten-id>` soft-delete, driven through the shared
// tensionSeam over a fake base transport so every scenario runs offline (no real
// network, no real ~/.glassfrogrc). Its Paths name ONLY this spec's feature file —
// never the features/ directory — so the suite reports its own independent scenario
// count and cannot disturb another suite (LEARNINGS: a suite points at its own
// feature file). The 3 @validation scenarios stay @wip (held for the validate skill)
// and are skipped by the ~@wip filter.
func TestTensionDiscardFeatures(t *testing.T) {
	suite := godog.TestSuite{
		ScenarioInitializer: initializeTensionDiscardScenario,
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"../../features/tension-capture/tension-discard.feature"},
			Tags:     "~@wip",
			TestingT: t,
		},
	}
	if suite.Run() != 0 {
		t.Fatal("non-zero status: tension-discard feature scenarios failed")
	}
}

// tensionDiscardWorld is the per-scenario state: the connection context and fake
// transport assembled from the Given steps, plus the captured
// outcome/exit-code/streams of the When run. Everything is injected — no step
// touches the real network, env, or home. The transport is the concrete
// tensionTransport so a step can read the recorded method/body/Content-Type/call-count.
type tensionDiscardWorld struct {
	ctx       apiclient.ConnectionContext
	transport *tensionTransport
	secret    string

	outcome  Outcome
	exitCode int
	stdout   string
	stderr   string
}

func initializeTensionDiscardScenario(sc *godog.ScenarioContext) {
	w := &tensionDiscardWorld{}
	sc.Before(func(ctx context.Context, _ *godog.Scenario) (context.Context, error) {
		*w = tensionDiscardWorld{
			// A 204 live discard is the default; per-scenario Given steps override the
			// transport/context (already-gone 404, forbidden 403, rate-limit, no token).
			transport: &tensionTransport{status: http.StatusNoContent, body: ""},
			secret:    meSecretToken,
		}
		w.ctx = validMeContext()
		return ctx, nil
	})

	// --- Givens ---
	sc.Step(`^a complete connection context with a stored token$`, w.completeContext)
	sc.Step(`^a tension "([^"]*)" exists$`, w.tensionExists)
	sc.Step(`^a tension "([^"]*)" that has already been discarded$`, w.tensionAlreadyDiscarded)
	sc.Step(`^the caller may not delete the tension "([^"]*)"$`, w.callerMayNotDelete)
	sc.Step(`^the tensions endpoint cannot be reached$`, w.endpointUnreachable)
	sc.Step(`^the tensions endpoint answers the discard with a rate-limit response$`, w.rateLimited)
	sc.Step(`^no usable token is available to the CLI$`, w.noToken)

	// --- When ---
	sc.Step(`^(?:an agent|a practitioner) runs "glassfrog (.+)"$`, w.runCommand)

	// --- Thens ---
	sc.Step(`^the request will DELETE the tension with no body$`, w.deletedNoBody)
	sc.Step(`^the API will answer (\d+)$`, w.apiAnswered)
	sc.Step(`^the discard result will be printed as the result$`, w.resultPrinted)
	sc.Step(`^stderr will note that the tension was discarded$`, w.stderrNotesDiscarded)
	sc.Step(`^stderr will note that the tension was already gone$`, w.stderrNotesAlreadyGone)
	sc.Step(`^the outcome will be treated as success$`, w.outcomeTreatedAsSuccess)
	sc.Step(`^the synthesized discard result will be rendered as JSON$`, w.renderedAsJSON)
	sc.Step(`^the command will exit with code (\d+)$`, w.exitWithCode)
	sc.Step(`^the command will exit with a non-zero permission code$`, w.exitPermissionCode)
	sc.Step(`^the command will exit with the network-unavailable code$`, w.exitNetworkUnavailable)
	sc.Step(`^the command will exit with the rate-limit code$`, w.exitRateLimitCode)
	sc.Step(`^stderr will report a usage error naming the required "([^"]*)"$`, w.stderrUsageErrorNamingRequired)
	sc.Step(`^stderr will report a usage error$`, w.stderrReportsUsageError)
	sc.Step(`^stderr will report a usage error and name the rejected output value "([^"]*)"$`, w.stderrReportsRejectedOutput)
	sc.Step(`^stderr will report "([^"]*)" and point to "([^"]*)"$`, w.stderrReportsAndPointsTo)
	sc.Step(`^stderr will report that the discard failed and name the HTTP status$`, w.stderrDiscardFailedNamesStatus)
	sc.Step(`^stderr will report the transport failure by name$`, w.transportFailureNamed)
	sc.Step(`^no request will be sent$`, w.noRequestSent)
	sc.Step(`^the rate-limit will be surfaced on the first occurrence$`, w.rateLimitSurfaced)
	sc.Step(`^the discard will not be retried$`, w.notRetried)
}

// --- Given implementations ---

func (w *tensionDiscardWorld) completeContext() error { w.ctx = validMeContext(); return nil }

func (w *tensionDiscardWorld) tensionExists(_ string) error {
	w.transport = &tensionTransport{status: http.StatusNoContent, body: ""}
	return nil
}

func (w *tensionDiscardWorld) tensionAlreadyDiscarded(_ string) error {
	w.transport = &tensionTransport{status: http.StatusNotFound, body: `{"detail":"Tension not found"}`}
	return nil
}

func (w *tensionDiscardWorld) callerMayNotDelete(_ string) error {
	w.transport = &tensionTransport{status: http.StatusForbidden, body: `{"detail":"forbidden"}`}
	return nil
}

func (w *tensionDiscardWorld) endpointUnreachable() error {
	w.transport = &tensionTransport{netErr: errors.New("dial tcp: connection refused")}
	return nil
}

func (w *tensionDiscardWorld) rateLimited() error {
	w.transport = &tensionTransport{status: http.StatusTooManyRequests, body: `{"detail":"rate limited"}`}
	return nil
}

func (w *tensionDiscardWorld) noToken() error {
	w.ctx = noTokenContext()
	return nil
}

// --- When implementation ---

// runCommand parses the captured invocation with the package's quote-aware
// splitArgs, after unescaping the feature file's \" inner quotes, and dispatches it
// through a real root with the `tension` group attached over a fake seam. It asserts
// the secret token never leaks into output.
func (w *tensionDiscardWorld) runCommand(invocation string) error {
	args := splitArgs(strings.ReplaceAll(invocation, `\"`, `"`))
	root := NewRootCommand()
	seam := &fakeMeSeam{ctx: w.ctx, transport: w.transport}
	MustRegister(root, newTensionCommand(seam))

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

func (w *tensionDiscardWorld) deletedNoBody() error {
	if w.transport.calls != 1 {
		return fmt.Errorf("a discard is exactly one request, got %d", w.transport.calls)
	}
	if w.transport.lastMethod != http.MethodDelete {
		return fmt.Errorf("the discard should DELETE, got method %q", w.transport.lastMethod)
	}
	if !strings.Contains(w.transport.lastPath, "/tensions/") {
		return fmt.Errorf("the request should target /tensions/{id}, got %q", w.transport.lastPath)
	}
	if w.transport.lastBody != "" {
		return fmt.Errorf("a bodyless DELETE must send no body, got %q", w.transport.lastBody)
	}
	if w.transport.lastContentType != "" {
		return fmt.Errorf("a bodyless DELETE must send no Content-Type, got %q", w.transport.lastContentType)
	}
	if w.transport.lastIfMatch != "" {
		return fmt.Errorf("no If-Match must be sent (last-write-wins), got %q", w.transport.lastIfMatch)
	}
	return nil
}

func (w *tensionDiscardWorld) apiAnswered(code int) error {
	if w.transport.status != code {
		return fmt.Errorf("the canned API response should be %d, got %d", code, w.transport.status)
	}
	if w.transport.calls != 1 {
		return fmt.Errorf("the API should have been called exactly once, got %d", w.transport.calls)
	}
	return nil
}

func (w *tensionDiscardWorld) resultPrinted() error {
	if !strings.Contains(w.stdout, "[discarded]") || !strings.Contains(w.stdout, "ten_") {
		return fmt.Errorf("the synthesized discard result should be printed (the ten_ id with [discarded]):\n%s", w.stdout)
	}
	return nil
}

func (w *tensionDiscardWorld) stderrNotesDiscarded() error {
	if !strings.Contains(w.stderr, "discarded tension") {
		return fmt.Errorf("stderr should note the tension was discarded:\n%s", w.stderr)
	}
	return nil
}

func (w *tensionDiscardWorld) stderrNotesAlreadyGone() error {
	if !strings.Contains(w.stderr, "already discarded") {
		return fmt.Errorf("stderr should note the tension was already gone:\n%s", w.stderr)
	}
	// A 404-as-success must not leak a not-found error.
	if strings.Contains(w.stderr, "404") || strings.Contains(strings.ToLower(w.stderr), "not found") {
		return fmt.Errorf("an already-gone discard must leak no not-found error:\n%s", w.stderr)
	}
	return nil
}

func (w *tensionDiscardWorld) outcomeTreatedAsSuccess() error {
	if w.outcome != Success {
		return fmt.Errorf("the outcome should be treated as success, got %v\nstderr: %s", w.outcome, w.stderr)
	}
	return nil
}

func (w *tensionDiscardWorld) renderedAsJSON() error {
	var doc struct {
		Data struct {
			ID        string `json:"id"`
			Discarded bool   `json:"discarded"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(w.stdout), &doc); err != nil {
		return fmt.Errorf("stdout should be valid JSON, got error %v:\n%s", err, w.stdout)
	}
	if doc.Data.ID == "" || !doc.Data.Discarded {
		return fmt.Errorf(`-o json should carry {"data":{"id":…,"discarded":true}}, got:\n%s`, w.stdout)
	}
	if strings.Contains(w.stdout, "discarded_at") {
		return fmt.Errorf("the result must carry no server-owned field (discarded_at):\n%s", w.stdout)
	}
	return nil
}

func (w *tensionDiscardWorld) exitWithCode(code int) error {
	if w.exitCode != code {
		return fmt.Errorf("exit code = %d, want %d (outcome %v)\nstderr: %s", w.exitCode, code, w.outcome, w.stderr)
	}
	return nil
}

func (w *tensionDiscardWorld) exitPermissionCode() error {
	if w.outcome != PermissionError || w.exitCode != 4 {
		return fmt.Errorf("outcome=%v exit=%d, want PermissionError/4\nstderr: %s", w.outcome, w.exitCode, w.stderr)
	}
	return nil
}

func (w *tensionDiscardWorld) exitNetworkUnavailable() error {
	if w.outcome != NetworkUnavailable || w.exitCode != 6 {
		return fmt.Errorf("outcome=%v exit=%d, want NetworkUnavailable/6\nstderr: %s", w.outcome, w.exitCode, w.stderr)
	}
	return nil
}

func (w *tensionDiscardWorld) exitRateLimitCode() error {
	if w.outcome != RateLimited || w.exitCode != 5 {
		return fmt.Errorf("outcome=%v exit=%d, want RateLimited/5\nstderr: %s", w.outcome, w.exitCode, w.stderr)
	}
	return nil
}

func (w *tensionDiscardWorld) stderrUsageErrorNamingRequired(_ string) error {
	if w.outcome != UsageError || w.exitCode != 2 {
		return fmt.Errorf("outcome=%v exit=%d, want UsageError/2\nstderr: %s", w.outcome, w.exitCode, w.stderr)
	}
	// cobra's ExactArgs(1) rejection names the argument requirement ("accepts 1
	// arg(s), received 0") on the SilenceErrors leaf via dispatch's surfacing.
	if !strings.Contains(w.stderr, "arg") {
		return fmt.Errorf("stderr should report the missing required argument:\n%s", w.stderr)
	}
	return nil
}

func (w *tensionDiscardWorld) stderrReportsUsageError() error {
	if w.outcome != UsageError || w.exitCode != 2 {
		return fmt.Errorf("outcome=%v exit=%d, want UsageError/2\nstderr: %s", w.outcome, w.exitCode, w.stderr)
	}
	if strings.TrimSpace(w.stderr) == "" {
		return errors.New("a usage error should be reported on stderr")
	}
	return nil
}

func (w *tensionDiscardWorld) stderrReportsRejectedOutput(value string) error {
	if w.outcome != UsageError || w.exitCode != 2 {
		return fmt.Errorf("outcome=%v exit=%d, want UsageError/2\nstderr: %s", w.outcome, w.exitCode, w.stderr)
	}
	if !strings.Contains(w.stderr, value) {
		return fmt.Errorf("stderr should name the rejected output value %q:\n%s", value, w.stderr)
	}
	return nil
}

func (w *tensionDiscardWorld) stderrReportsAndPointsTo(report, pointer string) error {
	if !strings.Contains(strings.ToLower(w.stderr), strings.ToLower(report)) {
		return fmt.Errorf("stderr should report %q:\n%s", report, w.stderr)
	}
	if !strings.Contains(w.stderr, pointer) {
		return fmt.Errorf("stderr should point to %q:\n%s", pointer, w.stderr)
	}
	return nil
}

func (w *tensionDiscardWorld) stderrDiscardFailedNamesStatus() error {
	if !strings.Contains(w.stderr, "403") {
		return fmt.Errorf("stderr should name the HTTP status (403):\n%s", w.stderr)
	}
	return nil
}

func (w *tensionDiscardWorld) transportFailureNamed() error {
	if w.outcome != NetworkUnavailable {
		return fmt.Errorf("outcome = %v, want NetworkUnavailable\nstderr: %s", w.outcome, w.stderr)
	}
	if strings.TrimSpace(w.stderr) == "" {
		return errors.New("a transport failure should be named on stderr")
	}
	return nil
}

func (w *tensionDiscardWorld) noRequestSent() error {
	if w.transport.calls != 0 {
		return fmt.Errorf("no request should be sent, but the transport was called %d times", w.transport.calls)
	}
	return nil
}

func (w *tensionDiscardWorld) rateLimitSurfaced() error {
	if w.outcome != RateLimited {
		return fmt.Errorf("the rate-limit should be surfaced (outcome RateLimited), got %v\nstderr: %s", w.outcome, w.stderr)
	}
	if w.transport.calls < 1 {
		return fmt.Errorf("the rate-limit must be surfaced on the first occurrence (the request was sent), got %d calls", w.transport.calls)
	}
	return nil
}

func (w *tensionDiscardWorld) notRetried() error {
	if w.transport.calls != 1 {
		return fmt.Errorf("a DELETE 429 must not be retried, want exactly 1 call, got %d", w.transport.calls)
	}
	return nil
}
