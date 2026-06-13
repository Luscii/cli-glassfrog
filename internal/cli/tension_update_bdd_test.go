package cli

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/Luscii/cli-glassfrog/internal/apiclient"
	"github.com/cucumber/godog"
)

// TestTensionUpdateFeatures runs the executable acceptance for Tension Update
// (044): the `tension update <ten-id>` edit, driven through the shared tensionSeam
// over a fake base transport so every scenario runs offline (no real network, no
// real ~/.glassfrogrc). Its Paths name ONLY this spec's feature file — never the
// features/ directory — so the suite reports its own independent scenario count and
// cannot disturb another suite (LEARNINGS: a suite points at its own feature file).
// The 3 @validation scenarios stay @wip (held for the validate skill) and are
// skipped by the ~@wip filter.
func TestTensionUpdateFeatures(t *testing.T) {
	suite := godog.TestSuite{
		ScenarioInitializer: initializeTensionUpdateScenario,
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"../../features/tension-capture/tension-update.feature"},
			Tags:     "~@wip",
			TestingT: t,
		},
	}
	if suite.Run() != 0 {
		t.Fatal("non-zero status: tension-update feature scenarios failed")
	}
}

// tensionUpdateWorld is the per-scenario state: the connection context and fake
// transport assembled from the Given steps, plus the captured
// outcome/exit-code/streams of the When run. Everything is injected — no step
// touches the real network, env, or home. The transport is the concrete
// tensionTransport so a step can read the recorded method/body/call-count.
type tensionUpdateWorld struct {
	ctx       apiclient.ConnectionContext
	transport *tensionTransport
	secret    string

	outcome  Outcome
	exitCode int
	stdout   string
	stderr   string
}

func initializeTensionUpdateScenario(sc *godog.ScenarioContext) {
	w := &tensionUpdateWorld{}
	sc.Before(func(ctx context.Context, _ *godog.Scenario) (context.Context, error) {
		*w = tensionUpdateWorld{
			// A 200-updated tension is the default; per-scenario Given steps override
			// the transport/context (unknown id, rate-limit, no token).
			transport: &tensionTransport{status: 200, body: tensionUpdatedBody},
			secret:    meSecretToken,
		}
		w.ctx = validMeContext()
		return ctx, nil
	})

	// --- Givens ---
	sc.Step(`^a complete connection context with a stored token$`, w.completeContext)
	sc.Step(`^a tension "([^"]*)" exists$`, w.tensionExists)
	sc.Step(`^no tension "([^"]*)" exists$`, w.tensionNotFound)
	sc.Step(`^the tensions endpoint answers the update with a rate-limit response$`, w.rateLimited)
	sc.Step(`^no usable token is available to the CLI$`, w.noToken)

	// --- When ---
	sc.Step(`^(?:an agent|a practitioner) runs "glassfrog (.+)"$`, w.runCommand)

	// --- Thens ---
	sc.Step(`^the request will PATCH the tension with the new body only$`, w.patchedBodyOnly)
	sc.Step(`^the updated tension will be printed as the result$`, w.updatedPrinted)
	sc.Step(`^the command will exit with code (\d+)$`, w.exitWithCode)
	sc.Step(`^the command will exit with a non-zero API-error code$`, w.exitNonZeroAPIError)
	sc.Step(`^the command will exit with the rate-limit code$`, w.exitRateLimitCode)
	sc.Step(`^stderr will report that the update failed and name the HTTP status$`, w.stderrNamesHTTPStatus)
	sc.Step(`^stderr will report that "([^"]*)" must not be empty$`, w.stderrReportsMustNotBeEmpty)
	sc.Step(`^stderr will report "([^"]*)" and point to "([^"]*)"$`, w.stderrReportsAndPointsTo)
	sc.Step(`^stderr will report the unsupported value and list the supported set$`, w.stderrReportsUnsupportedStatus)
	sc.Step(`^stderr will report a usage error naming that at least one field is required$`, w.stderrReportsAtLeastOneField)
	sc.Step(`^stderr will report a usage error and name the rejected output value "([^"]*)"$`, w.stderrReportsRejectedOutput)
	sc.Step(`^no request will be sent$`, w.noRequestSent)
	sc.Step(`^the value will be accepted as a supported status$`, w.statusAccepted)
	sc.Step(`^the request will carry "([^"]*)" set to "([^"]*)"$`, w.requestCarriesFieldSetTo)
	sc.Step(`^the request will carry "([^"]*)" set to "([^"]*)" and "([^"]*)" set to "([^"]*)" and no other fields$`, w.requestCarriesTwoFieldsOnly)
	sc.Step(`^the rate-limit will be surfaced on the first occurrence$`, w.rateLimitSurfaced)
	sc.Step(`^the update will not be retried$`, w.notRetried)
}

// --- Given implementations ---

func (w *tensionUpdateWorld) completeContext() error { w.ctx = validMeContext(); return nil }

func (w *tensionUpdateWorld) tensionExists(_ string) error {
	w.transport = &tensionTransport{status: 200, body: tensionUpdatedBody}
	return nil
}

func (w *tensionUpdateWorld) tensionNotFound(_ string) error {
	w.transport = &tensionTransport{status: 404, body: `{"detail":"Tension not found"}`}
	return nil
}

func (w *tensionUpdateWorld) rateLimited() error {
	w.transport = &tensionTransport{status: 429, body: `{"detail":"rate limited"}`}
	return nil
}

func (w *tensionUpdateWorld) noToken() error {
	w.ctx = noTokenContext()
	return nil
}

// --- When implementation ---

// runCommand parses the captured invocation with the package's quote-aware
// splitArgs (so `--body "new text"` reaches cobra as ONE flag value), after
// unescaping the feature file's \" inner quotes, and dispatches it through a real
// root with the `tension` group attached over a fake seam. It asserts the secret
// token never leaks into output.
func (w *tensionUpdateWorld) runCommand(invocation string) error {
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

func (w *tensionUpdateWorld) patchedBodyOnly() error {
	if w.transport.calls != 1 {
		return fmt.Errorf("an update is exactly one request, got %d", w.transport.calls)
	}
	if w.transport.lastMethod != "PATCH" {
		return fmt.Errorf("the update should PATCH, got method %q", w.transport.lastMethod)
	}
	if !strings.Contains(w.transport.lastPath, "/tensions/") {
		return fmt.Errorf("the request should target /tensions/{id}, got %q", w.transport.lastPath)
	}
	if !strings.Contains(w.transport.lastBody, `"body"`) {
		return fmt.Errorf("the body should carry the new body, got %s", w.transport.lastBody)
	}
	for _, forbidden := range []string{`"label"`, `"status"`, `"meeting_type"`} {
		if strings.Contains(w.transport.lastBody, forbidden) {
			return fmt.Errorf("a body-only update must not carry %s, got %s", forbidden, w.transport.lastBody)
		}
	}
	if w.transport.lastIfMatch != "" {
		return fmt.Errorf("no If-Match must be sent (last-write-wins), got %q", w.transport.lastIfMatch)
	}
	return nil
}

func (w *tensionUpdateWorld) updatedPrinted() error {
	if !strings.Contains(w.stdout, "ten_") {
		return fmt.Errorf("the updated tension should be printed (its ten_ id present):\n%s", w.stdout)
	}
	return nil
}

func (w *tensionUpdateWorld) exitWithCode(code int) error {
	if w.exitCode != code {
		return fmt.Errorf("exit code = %d, want %d (outcome %v)\nstderr: %s", w.exitCode, code, w.outcome, w.stderr)
	}
	return nil
}

func (w *tensionUpdateWorld) exitNonZeroAPIError() error {
	if w.outcome != APIError || w.exitCode != 3 {
		return fmt.Errorf("outcome=%v exit=%d, want APIError/3\nstderr: %s", w.outcome, w.exitCode, w.stderr)
	}
	return nil
}

func (w *tensionUpdateWorld) exitRateLimitCode() error {
	if w.outcome != RateLimited || w.exitCode != 5 {
		return fmt.Errorf("outcome=%v exit=%d, want RateLimited/5\nstderr: %s", w.outcome, w.exitCode, w.stderr)
	}
	return nil
}

func (w *tensionUpdateWorld) stderrNamesHTTPStatus() error {
	if !strings.Contains(w.stderr, "404") {
		return fmt.Errorf("stderr should name the HTTP status (404):\n%s", w.stderr)
	}
	return nil
}

func (w *tensionUpdateWorld) stderrReportsMustNotBeEmpty(name string) error {
	if !strings.Contains(w.stderr, name) {
		return fmt.Errorf("stderr should name %q:\n%s", name, w.stderr)
	}
	if !strings.Contains(w.stderr, "must not be empty") {
		return fmt.Errorf("stderr should report that the body must not be empty:\n%s", w.stderr)
	}
	return nil
}

func (w *tensionUpdateWorld) stderrReportsAndPointsTo(report, pointer string) error {
	if !strings.Contains(strings.ToLower(w.stderr), strings.ToLower(report)) {
		return fmt.Errorf("stderr should report %q:\n%s", report, w.stderr)
	}
	if !strings.Contains(w.stderr, pointer) {
		return fmt.Errorf("stderr should point to %q:\n%s", pointer, w.stderr)
	}
	return nil
}

func (w *tensionUpdateWorld) stderrReportsUnsupportedStatus() error {
	if !strings.Contains(w.stderr, "open") {
		return fmt.Errorf("stderr should name the unsupported value:\n%s", w.stderr)
	}
	for _, want := range []string{"unprocessed", "processed", "archived"} {
		if !strings.Contains(w.stderr, want) {
			return fmt.Errorf("stderr should list the supported status set (missing %q):\n%s", want, w.stderr)
		}
	}
	return nil
}

func (w *tensionUpdateWorld) stderrReportsAtLeastOneField() error {
	if !strings.Contains(w.stderr, "at least one") {
		return fmt.Errorf("stderr should report the at-least-one-field precondition:\n%s", w.stderr)
	}
	return nil
}

func (w *tensionUpdateWorld) stderrReportsRejectedOutput(value string) error {
	if w.outcome != UsageError || w.exitCode != 2 {
		return fmt.Errorf("outcome=%v exit=%d, want UsageError/2\nstderr: %s", w.outcome, w.exitCode, w.stderr)
	}
	if !strings.Contains(w.stderr, value) {
		return fmt.Errorf("stderr should name the rejected output value %q:\n%s", value, w.stderr)
	}
	return nil
}

func (w *tensionUpdateWorld) noRequestSent() error {
	if w.transport.calls != 0 {
		return fmt.Errorf("no request should be sent, but the transport was called %d times", w.transport.calls)
	}
	return nil
}

func (w *tensionUpdateWorld) statusAccepted() error {
	// A supported --status is not rejected locally: the request is sent and the
	// command succeeds (the server's recomputed status is rendered as returned).
	if w.outcome != Success {
		return fmt.Errorf("a supported status should be accepted (outcome Success), got %v\nstderr: %s", w.outcome, w.stderr)
	}
	if w.transport.calls != 1 {
		return fmt.Errorf("an accepted status should reach the API in exactly one request, got %d", w.transport.calls)
	}
	return nil
}

func (w *tensionUpdateWorld) requestCarriesFieldSetTo(field, value string) error {
	want := fmt.Sprintf(`"%s":"%s"`, field, value)
	if !strings.Contains(w.transport.lastBody, want) {
		return fmt.Errorf("the request body should carry %s, got %s", want, w.transport.lastBody)
	}
	return nil
}

func (w *tensionUpdateWorld) requestCarriesTwoFieldsOnly(f1, v1, f2, v2 string) error {
	for _, want := range []string{fmt.Sprintf(`"%s":"%s"`, f1, v1), fmt.Sprintf(`"%s":"%s"`, f2, v2)} {
		if !strings.Contains(w.transport.lastBody, want) {
			return fmt.Errorf("the request body should carry %s, got %s", want, w.transport.lastBody)
		}
	}
	for _, forbidden := range []string{`"body"`, `"status"`} {
		if strings.Contains(w.transport.lastBody, forbidden) {
			return fmt.Errorf("the body should carry no other fields, but found %s in %s", forbidden, w.transport.lastBody)
		}
	}
	return nil
}

func (w *tensionUpdateWorld) rateLimitSurfaced() error {
	if w.outcome != RateLimited {
		return fmt.Errorf("the rate-limit should be surfaced (outcome RateLimited), got %v\nstderr: %s", w.outcome, w.stderr)
	}
	if w.transport.calls < 1 {
		return fmt.Errorf("the rate-limit must be surfaced on the first occurrence (the request was sent), got %d calls", w.transport.calls)
	}
	return nil
}

func (w *tensionUpdateWorld) notRetried() error {
	if w.transport.calls != 1 {
		return fmt.Errorf("a PATCH 429 must not be retried, want exactly 1 call, got %d", w.transport.calls)
	}
	return nil
}
