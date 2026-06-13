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

// TestTensionReadsFeatures runs the executable acceptance for Tension Reads (043):
// the `tension list <role-id>` list and `tension get <ten-id>` single read, driven
// through the shared tensionSeam over a fake base transport so every scenario runs
// offline (no real network, no real ~/.glassfrogrc). Its Paths name ONLY this spec's
// feature file — never the features/ directory — so the suite reports its own
// independent scenario count and un-@wip-ping these scenarios cannot disturb another
// suite (LEARNINGS: a suite points at its own feature file). The 3 @validation
// scenarios stay @wip (held for the validate skill) and are skipped by the ~@wip
// filter.
func TestTensionReadsFeatures(t *testing.T) {
	suite := godog.TestSuite{
		ScenarioInitializer: initializeTensionReadsScenario,
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"../../features/tension-capture/tension-reads.feature"},
			Tags:     "~@wip",
			TestingT: t,
		},
	}
	if suite.Run() != 0 {
		t.Fatal("non-zero status: tension-reads feature scenarios failed")
	}
}

// tensionReadsWorld is the per-scenario state: the connection context and fake
// transport assembled from the Given steps, plus the captured
// outcome/exit-code/streams of the When run. Everything is injected — no step
// touches the real network, env, or home.
type tensionReadsWorld struct {
	ctx       apiclient.ConnectionContext
	transport http.RoundTripper
	secret    string

	outcome  Outcome
	exitCode int
	stdout   string
	stderr   string
}

func initializeTensionReadsScenario(sc *godog.ScenarioContext) {
	w := &tensionReadsWorld{}
	sc.Before(func(ctx context.Context, _ *godog.Scenario) (context.Context, error) {
		*w = tensionReadsWorld{
			// A two-tension single-page body is the default; the per-scenario Given
			// steps override the transport/context as needed.
			transport: &cannedTransport{status: 200, body: tensionsPageComplete},
			secret:    meSecretToken,
		}
		w.ctx = validMeContext()
		return ctx, nil
	})

	// --- Givens ---
	sc.Step(`^a complete connection context with a stored token$`, w.completeContext)
	sc.Step(`^the role "([^"]*)" carries several tensions$`, w.roleCarriesSeveralTensions)
	sc.Step(`^the role "([^"]*)" carries no tensions$`, w.roleCarriesNoTensions)
	sc.Step(`^the role "([^"]*)" carries tensions in several statuses$`, w.roleCarriesTensionsInStatuses)
	sc.Step(`^the role "([^"]*)" has tensions spanning more than one page$`, w.roleHasMultiPageTensions)
	sc.Step(`^the tension list walk fails after retrieving the first page$`, w.walkFailsMidway)
	sc.Step(`^no usable token is available to the CLI$`, w.noToken)
	sc.Step(`^a tension "([^"]*)" exists$`, w.tensionExists)
	sc.Step(`^no tension "([^"]*)" exists$`, w.tensionNotFound)

	// --- Whens --- (both "an agent" and "a practitioner" drive the same run) ---
	sc.Step(`^(?:an agent|a practitioner) runs "glassfrog (.+)"$`, w.runCommand)

	// --- Thens ---
	sc.Step(`^the request will read the role's tensions endpoint$`, w.requestHitTensionsEndpoint)
	sc.Step(`^each tension will be printed as a projection$`, w.eachTensionPrinted)
	sc.Step(`^the command will exit with code (\d+)$`, w.exitWithCode)
	sc.Step(`^the command will exit with a non-zero API-error code$`, w.exitNonZeroAPIError)
	sc.Step(`^the command will exit with a non-zero code$`, w.exitNonZero)
	sc.Step(`^"([^"]*)" will be printed to stdout$`, w.literalPrintedToStdout)
	sc.Step(`^stderr will report "([^"]*)" and point to "([^"]*)"$`, w.stderrReportsAndPointsTo)
	sc.Step(`^no tension data will be printed$`, w.noTensionDataPrinted)
	sc.Step(`^the tension's status, body, and sensing role will be printed$`, w.statusBodyRolePrinted)
	sc.Step(`^stderr will report that the read failed and name the HTTP status$`, w.stderrNamesHTTPStatus)
	sc.Step(`^stderr will report a usage error$`, w.stderrReportsUsageError)
	sc.Step(`^stderr will report a usage error and name the rejected output value "([^"]*)"$`, w.stderrReportsRejectedOutput)
	sc.Step(`^no request will be sent$`, w.noRequestSent)
	sc.Step(`^the request will carry the "([^"]*)" parameter set to "([^"]*)"$`, w.requestCarriesParamSetTo)
	sc.Step(`^only the unprocessed tensions will be printed$`, w.onlyUnprocessedTensionsPrinted)
	sc.Step(`^stderr will report the unsupported value and list the supported set$`, w.stderrReportsUnsupportedStatus)
	sc.Step(`^only the first page of tensions will be printed$`, w.onlyFirstPagePrinted)
	sc.Step(`^stderr will note that more tensions exist$`, w.stderrNotesMoreExist)
	sc.Step(`^the tensions retrieved so far will be printed$`, w.partialTensionsPrinted)
	sc.Step(`^stderr will note the result is incomplete and name the cause$`, w.stderrNotesIncomplete)
}

// --- Given implementations ---

func (w *tensionReadsWorld) completeContext() error { w.ctx = validMeContext(); return nil }

func (w *tensionReadsWorld) roleCarriesSeveralTensions(_ string) error {
	w.transport = &cannedTransport{status: 200, body: tensionsPageComplete}
	return nil
}

func (w *tensionReadsWorld) roleCarriesNoTensions(_ string) error {
	w.transport = &cannedTransport{status: 200, body: tensionsPageEmpty}
	return nil
}

func (w *tensionReadsWorld) roleCarriesTensionsInStatuses(_ string) error {
	// A single unprocessed tension so "only the unprocessed tensions will be printed"
	// is a genuine assertion (the API does the filtering; the fake returns the
	// filtered set).
	w.transport = &cannedTransport{status: 200, body: tensionsPage("ten_1", "Unprocessed One", "")}
	return nil
}

func (w *tensionReadsWorld) roleHasMultiPageTensions(_ string) error {
	w.transport = &cannedTransport{status: 200, body: tensionsPage("ten_1", "First Page Tension", "c1")}
	return nil
}

func (w *tensionReadsWorld) walkFailsMidway() error {
	w.transport = &seqMeTransport{steps: []seqMeResp{
		{status: 200, body: tensionsPage("ten_1", "Gathered Tension", "c1")},
		{status: 500, body: `{"detail":"boom"}`},
	}}
	return nil
}

func (w *tensionReadsWorld) noToken() error {
	w.ctx = noTokenContext()
	return nil
}

func (w *tensionReadsWorld) tensionExists(_ string) error {
	w.transport = &cannedTransport{status: 200, body: tensionDocumentBody}
	return nil
}

func (w *tensionReadsWorld) tensionNotFound(_ string) error {
	w.transport = &cannedTransport{status: 404, body: `{"detail":"Tension not found"}`}
	return nil
}

// --- When implementation ---

// runCommand parses the captured invocation and dispatches it through a real root
// with the `tension` group attached over a fake seam — the group parents create
// (042), list, and get, so a single suite drives both reads. It asserts the secret
// token never leaks.
func (w *tensionReadsWorld) runCommand(invocation string) error {
	root := NewRootCommand()
	seam := &fakeMeSeam{ctx: w.ctx, transport: w.transport}
	MustRegister(root, newTensionCommand(seam))

	var out, errb bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&errb)
	w.outcome, _ = Run(root, strings.Fields(invocation))
	w.exitCode = ExitCode(w.outcome)
	w.stdout, w.stderr = out.String(), errb.String()

	if w.secret != "" && strings.Contains(w.stdout+w.stderr, w.secret) {
		return fmt.Errorf("the token leaked into output: stdout=%q stderr=%q", w.stdout, w.stderr)
	}
	return nil
}

// transportCalls reads the request count off whichever fake transport the scenario
// installed.
func (w *tensionReadsWorld) transportCalls() int {
	switch t := w.transport.(type) {
	case *cannedTransport:
		return t.calls
	case *seqMeTransport:
		return t.calls
	default:
		return -1
	}
}

// --- Then implementations ---

func (w *tensionReadsWorld) requestHitTensionsEndpoint() error {
	t, ok := w.transport.(*cannedTransport)
	if !ok {
		return errors.New("this scenario's transport does not record the path")
	}
	if !strings.HasSuffix(t.lastPath, "/tensions") || !strings.Contains(t.lastPath, "/roles/") {
		return fmt.Errorf("the request should target /roles/{id}/tensions, got %q", t.lastPath)
	}
	return nil
}

func (w *tensionReadsWorld) eachTensionPrinted() error {
	for _, want := range []string{"ten_1  [unprocessed]  Roadmap drift", "ten_2  [processed]"} {
		if !strings.Contains(w.stdout, want) {
			return fmt.Errorf("each tension should print as a projection, missing %q:\n%s", want, w.stdout)
		}
	}
	return nil
}

func (w *tensionReadsWorld) exitWithCode(code int) error {
	if w.exitCode != code {
		return fmt.Errorf("exit code = %d, want %d (outcome %v)\nstderr: %s", w.exitCode, code, w.outcome, w.stderr)
	}
	return nil
}

func (w *tensionReadsWorld) exitNonZeroAPIError() error {
	if w.outcome != APIError || w.exitCode != 3 {
		return fmt.Errorf("outcome=%v exit=%d, want APIError/3\nstderr: %s", w.outcome, w.exitCode, w.stderr)
	}
	return nil
}

func (w *tensionReadsWorld) exitNonZero() error {
	if w.exitCode == 0 {
		return fmt.Errorf("exit code = 0, want a non-zero code (outcome %v)\nstderr: %s", w.outcome, w.stderr)
	}
	return nil
}

func (w *tensionReadsWorld) literalPrintedToStdout(literal string) error {
	if !strings.Contains(w.stdout, literal) {
		return fmt.Errorf("stdout should contain %q:\n%s", literal, w.stdout)
	}
	return nil
}

func (w *tensionReadsWorld) stderrReportsAndPointsTo(report, pointer string) error {
	if !strings.Contains(strings.ToLower(w.stderr), strings.ToLower(report)) {
		return fmt.Errorf("stderr should report %q:\n%s", report, w.stderr)
	}
	if !strings.Contains(w.stderr, pointer) {
		return fmt.Errorf("stderr should point to %q:\n%s", pointer, w.stderr)
	}
	return nil
}

func (w *tensionReadsWorld) noTensionDataPrinted() error {
	if strings.TrimSpace(w.stdout) != "" {
		return fmt.Errorf("no tension data should be printed, got stdout:\n%s", w.stdout)
	}
	return nil
}

func (w *tensionReadsWorld) statusBodyRolePrinted() error {
	for _, want := range []string{
		"[unprocessed]", // status
		"We ship faster than we update the roadmap.", // body
		"role_0123", // sensing role
	} {
		if !strings.Contains(w.stdout, want) {
			return fmt.Errorf("the single tension should print its status, body, and sensing role, missing %q:\n%s", want, w.stdout)
		}
	}
	return nil
}

func (w *tensionReadsWorld) stderrNamesHTTPStatus() error {
	if !strings.Contains(w.stderr, "404") {
		return fmt.Errorf("stderr should name the HTTP status (404):\n%s", w.stderr)
	}
	return nil
}

func (w *tensionReadsWorld) stderrReportsUsageError() error {
	if w.outcome != UsageError || w.exitCode != 2 {
		return fmt.Errorf("outcome=%v exit=%d, want UsageError/2\nstderr: %s", w.outcome, w.exitCode, w.stderr)
	}
	if strings.TrimSpace(w.stderr) == "" {
		return errors.New("a usage error should be reported on stderr")
	}
	return nil
}

func (w *tensionReadsWorld) stderrReportsRejectedOutput(value string) error {
	// Reuse the shared usage-error contract (UsageError/exit-2 + non-empty stderr) so
	// it has a single source — a change to that contract updates one place — then add
	// the message assertion this step is about: the rejected value is named.
	if err := w.stderrReportsUsageError(); err != nil {
		return err
	}
	if !strings.Contains(w.stderr, value) {
		return fmt.Errorf("stderr should name the rejected output value %q:\n%s", value, w.stderr)
	}
	return nil
}

func (w *tensionReadsWorld) noRequestSent() error {
	if w.transportCalls() != 0 {
		return fmt.Errorf("no request should be sent, but the transport was called %d times", w.transportCalls())
	}
	return nil
}

func (w *tensionReadsWorld) requestCarriesParamSetTo(param, value string) error {
	t, ok := w.transport.(*cannedTransport)
	if !ok {
		return errors.New("this scenario's transport does not record the query")
	}
	if got := t.lastQuery.Get(param); got != value {
		return fmt.Errorf("the request query should carry %q=%q, got %q (query %q)", param, value, got, t.lastQuery.Encode())
	}
	return nil
}

func (w *tensionReadsWorld) onlyUnprocessedTensionsPrinted() error {
	if !strings.Contains(w.stdout, "[unprocessed]") || !strings.Contains(w.stdout, "Unprocessed One") {
		return fmt.Errorf("the unprocessed tensions should be printed:\n%s", w.stdout)
	}
	return nil
}

func (w *tensionReadsWorld) stderrReportsUnsupportedStatus() error {
	if w.outcome != UsageError || w.exitCode != 2 {
		return fmt.Errorf("outcome=%v exit=%d, want UsageError/2\nstderr: %s", w.outcome, w.exitCode, w.stderr)
	}
	// Names the rejected value and lists at least one supported status.
	if !strings.Contains(w.stderr, "open") {
		return fmt.Errorf("stderr should name the unsupported value:\n%s", w.stderr)
	}
	if !strings.Contains(w.stderr, "unprocessed") {
		return fmt.Errorf("stderr should list the supported set:\n%s", w.stderr)
	}
	return nil
}

func (w *tensionReadsWorld) onlyFirstPagePrinted() error {
	if !strings.Contains(w.stdout, "First Page Tension") {
		return fmt.Errorf("the first page should print:\n%s", w.stdout)
	}
	if w.transportCalls() != 1 {
		return fmt.Errorf("--first-page must not walk, want 1 call, got %d", w.transportCalls())
	}
	return nil
}

func (w *tensionReadsWorld) stderrNotesMoreExist() error {
	if !strings.Contains(w.stderr, "more tensions exist") {
		return fmt.Errorf("stderr should note more tensions exist:\n%s", w.stderr)
	}
	return nil
}

func (w *tensionReadsWorld) partialTensionsPrinted() error {
	if !strings.Contains(w.stdout, "Gathered Tension") {
		return fmt.Errorf("the partial set gathered so far should print:\n%s", w.stdout)
	}
	return nil
}

func (w *tensionReadsWorld) stderrNotesIncomplete() error {
	if !strings.Contains(w.stderr, "incomplete") {
		return fmt.Errorf("stderr should note the result is incomplete and name the cause:\n%s", w.stderr)
	}
	return nil
}
