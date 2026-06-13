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

// TestTensionCaptureFeatures runs the executable acceptance for Tension Capture
// (042): the `tension create <role-id>` write, driven through the shared
// tensionSeam over a fake base transport so every scenario runs offline (no real
// network, no real ~/.glassfrogrc). Its Paths name ONLY this spec's feature file —
// never the features/ directory — so the suite reports its own independent scenario
// count and cannot disturb another suite (LEARNINGS: a suite points at its own
// feature file). The 2 @validation scenarios stay @wip (held for the validate
// skill) and are skipped by the ~@wip filter.
func TestTensionCaptureFeatures(t *testing.T) {
	suite := godog.TestSuite{
		ScenarioInitializer: initializeTensionCaptureScenario,
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"../../features/tension-capture/tension-capture.feature"},
			Tags:     "~@wip",
			TestingT: t,
		},
	}
	if suite.Run() != 0 {
		t.Fatal("non-zero status: tension-capture feature scenarios failed")
	}
}

// tensionCaptureWorld is the per-scenario state: the connection context and fake
// transport assembled from the Given steps, plus the captured
// outcome/exit-code/streams of the When run. Everything is injected — no step
// touches the real network, env, or home. The transport is the concrete
// tensionTransport so a step can read the recorded method/body/call-count.
type tensionCaptureWorld struct {
	ctx       apiclient.ConnectionContext
	transport *tensionTransport
	secret    string

	outcome  Outcome
	exitCode int
	stdout   string
	stderr   string
}

func initializeTensionCaptureScenario(sc *godog.ScenarioContext) {
	w := &tensionCaptureWorld{}
	sc.Before(func(ctx context.Context, _ *godog.Scenario) (context.Context, error) {
		*w = tensionCaptureWorld{
			// A 201-created tension is the default; per-scenario Given steps override
			// the transport/context (unknown role, rate-limit, no token).
			transport: &tensionTransport{status: 201, body: tensionCreatedBody},
			secret:    meSecretToken,
		}
		w.ctx = validMeContext()
		return ctx, nil
	})

	// --- Givens ---
	sc.Step(`^a complete connection context with a stored token$`, w.completeContext)
	sc.Step(`^the role "([^"]*)" exists$`, w.roleExists)
	sc.Step(`^no role "([^"]*)" exists$`, w.roleNotFound)
	sc.Step(`^the tensions endpoint answers the capture with a rate-limit response$`, w.rateLimited)
	sc.Step(`^no usable token is available to the CLI$`, w.noToken)

	// --- When ---
	sc.Step(`^an agent runs "glassfrog (.+)"$`, w.runCommand)

	// --- Thens ---
	sc.Step(`^the request will post the tension to the role's tensions endpoint$`, w.requestPostedToTensionsEndpoint)
	sc.Step(`^the created tension will be printed with its "([^"]*)" id and computed status$`, w.createdPrintedWithIDAndStatus)
	sc.Step(`^the created tension will be printed$`, w.createdPrinted)
	sc.Step(`^the command will exit with code (\d+)$`, w.exitWithCode)
	sc.Step(`^the command will exit with a non-zero API-error code$`, w.exitNonZeroAPIError)
	sc.Step(`^the command will exit with the rate-limit code$`, w.exitRateLimitCode)
	sc.Step(`^stderr will report "([^"]*)" and point to "([^"]*)"$`, w.stderrReportsAndPointsTo)
	sc.Step(`^stderr will report that the capture failed and name the HTTP status$`, w.stderrNamesHTTPStatus)
	sc.Step(`^stderr will report the unsupported value and list the supported set$`, w.stderrReportsUnsupportedMeetingType)
	sc.Step(`^stderr will report a usage error and name the rejected output value "([^"]*)"$`, w.stderrReportsRejectedOutput)
	sc.Step(`^stderr will report that "([^"]*)" is required$`, w.stderrReportsRequired)
	sc.Step(`^no request will be sent$`, w.noRequestSent)
	sc.Step(`^the rate-limit will be surfaced on the first occurrence$`, w.rateLimitSurfaced)
	sc.Step(`^the capture will not be retried, so no duplicate tension is created$`, w.notRetried)
	sc.Step(`^the structured result will contain the created tension's "([^"]*)" id$`, w.structuredContainsID)
	sc.Step(`^the request body will carry the body, the label, and "([^"]*)" set to "([^"]*)"$`, w.bodyCarriesLabelAndMeetingType)
}

// --- Given implementations ---

func (w *tensionCaptureWorld) completeContext() error { w.ctx = validMeContext(); return nil }

func (w *tensionCaptureWorld) roleExists(_ string) error {
	w.transport = &tensionTransport{status: 201, body: tensionCreatedBody}
	return nil
}

func (w *tensionCaptureWorld) roleNotFound(_ string) error {
	w.transport = &tensionTransport{status: 404, body: `{"detail":"Role not found"}`}
	return nil
}

func (w *tensionCaptureWorld) rateLimited() error {
	w.transport = &tensionTransport{status: 429, body: `{"detail":"rate limited"}`}
	return nil
}

func (w *tensionCaptureWorld) noToken() error {
	w.ctx = noTokenContext()
	return nil
}

// --- When implementation ---

// runCommand parses the captured invocation with the package's quote-aware
// splitArgs (so `--body "a tension"` reaches cobra as ONE flag value), after
// unescaping the feature file's \" inner quotes, and dispatches it through a real
// root with the `tension` group attached over a fake seam. It asserts the secret
// token never leaks into output.
func (w *tensionCaptureWorld) runCommand(invocation string) error {
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

func (w *tensionCaptureWorld) requestPostedToTensionsEndpoint() error {
	if w.transport.calls != 1 {
		return fmt.Errorf("a capture is exactly one request, got %d", w.transport.calls)
	}
	if w.transport.lastMethod != "POST" {
		return fmt.Errorf("the capture should POST, got method %q", w.transport.lastMethod)
	}
	if !strings.Contains(w.transport.lastPath, "/roles/") || !strings.HasSuffix(w.transport.lastPath, "/tensions") {
		return fmt.Errorf("the request should target /roles/{id}/tensions, got %q", w.transport.lastPath)
	}
	return nil
}

func (w *tensionCaptureWorld) createdPrintedWithIDAndStatus(idPrefix string) error {
	if !strings.Contains(w.stdout, "ten_") {
		return fmt.Errorf("the created tension should print its %q id:\n%s", idPrefix, w.stdout)
	}
	if !strings.Contains(w.stdout, "unprocessed") {
		return fmt.Errorf("the created tension should print its computed status:\n%s", w.stdout)
	}
	return nil
}

func (w *tensionCaptureWorld) createdPrinted() error {
	if !strings.Contains(w.stdout, "ten_") {
		return fmt.Errorf("the created tension should be printed (its ten_ id present):\n%s", w.stdout)
	}
	return nil
}

func (w *tensionCaptureWorld) exitWithCode(code int) error {
	if w.exitCode != code {
		return fmt.Errorf("exit code = %d, want %d (outcome %v)\nstderr: %s", w.exitCode, code, w.outcome, w.stderr)
	}
	return nil
}

func (w *tensionCaptureWorld) exitNonZeroAPIError() error {
	if w.outcome != APIError || w.exitCode != 3 {
		return fmt.Errorf("outcome=%v exit=%d, want APIError/3\nstderr: %s", w.outcome, w.exitCode, w.stderr)
	}
	return nil
}

func (w *tensionCaptureWorld) exitRateLimitCode() error {
	if w.outcome != RateLimited || w.exitCode != 5 {
		return fmt.Errorf("outcome=%v exit=%d, want RateLimited/5\nstderr: %s", w.outcome, w.exitCode, w.stderr)
	}
	return nil
}

func (w *tensionCaptureWorld) stderrReportsAndPointsTo(report, pointer string) error {
	if !strings.Contains(strings.ToLower(w.stderr), strings.ToLower(report)) {
		return fmt.Errorf("stderr should report %q:\n%s", report, w.stderr)
	}
	if !strings.Contains(w.stderr, pointer) {
		return fmt.Errorf("stderr should point to %q:\n%s", pointer, w.stderr)
	}
	return nil
}

func (w *tensionCaptureWorld) stderrNamesHTTPStatus() error {
	if !strings.Contains(w.stderr, "404") {
		return fmt.Errorf("stderr should name the HTTP status (404):\n%s", w.stderr)
	}
	return nil
}

func (w *tensionCaptureWorld) stderrReportsUnsupportedMeetingType() error {
	if !strings.Contains(w.stderr, "weekly") {
		return fmt.Errorf("stderr should name the unsupported value:\n%s", w.stderr)
	}
	for _, want := range []string{"tactical", "governance"} {
		if !strings.Contains(w.stderr, want) {
			return fmt.Errorf("stderr should list the supported set (missing %q):\n%s", want, w.stderr)
		}
	}
	return nil
}

func (w *tensionCaptureWorld) stderrReportsRejectedOutput(value string) error {
	if w.outcome != UsageError || w.exitCode != 2 {
		return fmt.Errorf("outcome=%v exit=%d, want UsageError/2\nstderr: %s", w.outcome, w.exitCode, w.stderr)
	}
	if !strings.Contains(w.stderr, value) {
		return fmt.Errorf("stderr should name the rejected output value %q:\n%s", value, w.stderr)
	}
	return nil
}

func (w *tensionCaptureWorld) stderrReportsRequired(name string) error {
	if !strings.Contains(w.stderr, name) {
		return fmt.Errorf("stderr should report that %q is required:\n%s", name, w.stderr)
	}
	return nil
}

func (w *tensionCaptureWorld) noRequestSent() error {
	if w.transport.calls != 0 {
		return fmt.Errorf("no request should be sent, but the transport was called %d times", w.transport.calls)
	}
	return nil
}

func (w *tensionCaptureWorld) rateLimitSurfaced() error {
	if w.outcome != RateLimited {
		return fmt.Errorf("the rate-limit should be surfaced (outcome RateLimited), got %v\nstderr: %s", w.outcome, w.stderr)
	}
	if w.transport.calls < 1 {
		return fmt.Errorf("the rate-limit must be surfaced on the first occurrence (the request was sent), got %d calls", w.transport.calls)
	}
	return nil
}

func (w *tensionCaptureWorld) notRetried() error {
	if w.transport.calls != 1 {
		return fmt.Errorf("a POST 429 must not be retried (no duplicate tension), want exactly 1 call, got %d", w.transport.calls)
	}
	return nil
}

func (w *tensionCaptureWorld) structuredContainsID(idPrefix string) error {
	if !strings.Contains(w.stdout, "ten_") {
		return fmt.Errorf("the structured result should contain the created tension's %q id:\n%s", idPrefix, w.stdout)
	}
	// Structured output is the raw {data: …} payload, not the human projection.
	if strings.Contains(w.stdout, "Sensing role:") {
		return fmt.Errorf("structured output must not render the human projection:\n%s", w.stdout)
	}
	return nil
}

func (w *tensionCaptureWorld) bodyCarriesLabelAndMeetingType(field, value string) error {
	for _, want := range []string{`"body"`, `"label"`, fmt.Sprintf(`"%s":"%s"`, field, value)} {
		if !strings.Contains(w.transport.lastBody, want) {
			return fmt.Errorf("the request body should carry %s, got %s", want, w.transport.lastBody)
		}
	}
	return nil
}
