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

// TestProposalReadsFeatures runs the executable acceptance for Proposal Reads (056):
// the global `proposal list` and the single `proposal get <prp-id>` read, driven
// through the shared proposalSeam over a fake base transport so every scenario runs
// offline (no real network, no real ~/.glassfrogrc). Its Paths name ONLY this spec's
// feature file — never the features/ directory — so the suite reports its own
// independent scenario count and un-@wip-ping these scenarios cannot disturb another
// suite (LEARNINGS: a suite points at its own feature file). The 4 @validation scenarios
// stay @wip (held for the validate skill) and are skipped by the ~@wip filter.
func TestProposalReadsFeatures(t *testing.T) {
	suite := godog.TestSuite{
		ScenarioInitializer: initializeProposalReadsScenario,
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"../../features/proposal-write-flow/proposal-reads.feature"},
			Tags:     "~@wip",
			TestingT: t,
		},
	}
	if suite.Run() != 0 {
		t.Fatal("non-zero status: proposal-reads feature scenarios failed")
	}
}

// proposalReadsWorld is the per-scenario state: the connection context and fake
// transport assembled from the Given steps, plus the captured outcome/exit-code/streams
// of the When run. Everything is injected — no step touches the real network, env, or
// home.
type proposalReadsWorld struct {
	ctx       apiclient.ConnectionContext
	transport http.RoundTripper
	secret    string

	outcome  Outcome
	exitCode int
	stdout   string
	stderr   string
}

func initializeProposalReadsScenario(sc *godog.ScenarioContext) {
	w := &proposalReadsWorld{}
	sc.Before(func(ctx context.Context, _ *godog.Scenario) (context.Context, error) {
		*w = proposalReadsWorld{
			// A two-proposal single-page body is the default; per-scenario Given steps
			// override the transport/context as needed.
			transport: &cannedTransport{status: 200, body: proposalsPageComplete},
			secret:    meSecretToken,
		}
		w.ctx = validMeContext()
		return ctx, nil
	})

	// --- Givens ---
	sc.Step(`^a complete connection context with a stored token$`, w.completeContext)
	sc.Step(`^several proposals are visible to the caller$`, w.severalProposalsVisible)
	sc.Step(`^no proposals are visible to the caller$`, w.noProposalsVisible)
	sc.Step(`^the visible proposals span more than one page$`, w.proposalsSpanMultiplePages)
	sc.Step(`^no usable token is available to the CLI$`, w.noToken)
	sc.Step(`^a proposal "([^"]*)" exists$`, w.proposalExists)
	sc.Step(`^no proposal "([^"]*)" exists$`, w.proposalNotFound)

	// --- Whens --- (an agent, a practitioner, or a proposer all drive the same run) ---
	sc.Step(`^(?:an agent|a practitioner|a proposer) runs "glassfrog (.+)"$`, w.runCommand)

	// --- Thens ---
	sc.Step(`^the request will read the proposals endpoint$`, w.requestReadProposalsEndpoint)
	sc.Step(`^each proposal will be printed as a projection$`, w.eachProposalPrinted)
	sc.Step(`^the command will exit with code (\d+)$`, w.exitWithCode)
	sc.Step(`^the command will exit with a non-zero API-error code$`, w.exitNonZeroAPIError)
	sc.Step(`^the request will send "([^"]*)" as "([^"]*)" and "([^"]*)" as "([^"]*)"$`, w.requestSendsTwoParams)
	sc.Step(`^the request will send "([^"]*)" as "([^"]*)"$`, w.requestSendsParam)
	sc.Step(`^only the matching proposals will be printed$`, w.matchingProposalsPrinted)
	sc.Step(`^only that proposer's proposals will be printed$`, w.matchingProposalsPrinted)
	sc.Step(`^"([^"]*)" will be printed to stdout$`, w.literalPrintedToStdout)
	sc.Step(`^stderr will report "([^"]*)" and point to "([^"]*)"$`, w.stderrReportsAndPointsTo)
	sc.Step(`^no proposal data will be printed$`, w.noProposalDataPrinted)
	sc.Step(`^the proposal's status, changes, response summary, and available transitions will be printed$`, w.proposalDetailPrinted)
	sc.Step(`^stderr will report that the read failed and name the HTTP status$`, w.stderrNamesHTTPStatus)
	sc.Step(`^stderr will report a usage error$`, w.stderrReportsUsageError)
	sc.Step(`^stderr will report a usage error and name the rejected output value "([^"]*)"$`, w.stderrReportsRejectedOutput)
	sc.Step(`^stderr will report a usage error naming the value "([^"]*)" and the supported set$`, w.stderrReportsUnsupportedStatus)
	sc.Step(`^no request will be sent$`, w.noRequestSent)
	sc.Step(`^only the first page of proposals will be printed$`, w.onlyFirstPagePrinted)
	sc.Step(`^stderr will note that more proposals exist than shown$`, w.stderrNotesMoreExist)
}

// --- Given implementations ---

func (w *proposalReadsWorld) completeContext() error { w.ctx = validMeContext(); return nil }

func (w *proposalReadsWorld) severalProposalsVisible() error {
	w.transport = &cannedTransport{status: 200, body: proposalsPageComplete}
	return nil
}

func (w *proposalReadsWorld) noProposalsVisible() error {
	w.transport = &cannedTransport{status: 200, body: proposalsPageEmpty}
	return nil
}

func (w *proposalReadsWorld) proposalsSpanMultiplePages() error {
	w.transport = &cannedTransport{status: 200, body: proposalsPage("prp_1", "draft", "c1")}
	return nil
}

func (w *proposalReadsWorld) noToken() error {
	w.ctx = noTokenContext()
	return nil
}

func (w *proposalReadsWorld) proposalExists(_ string) error {
	w.transport = &cannedTransport{status: 200, body: proposalDocumentBody}
	return nil
}

func (w *proposalReadsWorld) proposalNotFound(_ string) error {
	w.transport = &cannedTransport{status: 404, body: `{"detail":"Proposal not found"}`}
	return nil
}

// --- When implementation ---

// runCommand dispatches the captured invocation through a real root with the `proposal`
// group attached over a fake seam — the group parents create (055), list, and get, so a
// single suite drives both reads. The invocations carry only simple tokens (no inline
// JSON), so strings.Fields splits them faithfully. It asserts the secret token never
// leaks.
func (w *proposalReadsWorld) runCommand(invocation string) error {
	root := NewRootCommand()
	seam := &fakeProposalSeam{fakeMeSeam: &fakeMeSeam{ctx: w.ctx, transport: w.transport}}
	MustRegister(root, newProposalCommand(seam))

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

// transportCalls reads the request count off the fake transport the scenario installed.
func (w *proposalReadsWorld) transportCalls() int {
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

func (w *proposalReadsWorld) requestReadProposalsEndpoint() error {
	t, ok := w.transport.(*cannedTransport)
	if !ok {
		return errors.New("this scenario's transport does not record the path")
	}
	if !strings.HasSuffix(t.lastPath, "/proposals") {
		return fmt.Errorf("the request should target /proposals, got %q", t.lastPath)
	}
	return nil
}

func (w *proposalReadsWorld) eachProposalPrinted() error {
	for _, want := range []string{"prp_1  [draft]", "prp_2  [accepted]"} {
		if !strings.Contains(w.stdout, want) {
			return fmt.Errorf("each proposal should print as a projection, missing %q:\n%s", want, w.stdout)
		}
	}
	return nil
}

func (w *proposalReadsWorld) exitWithCode(code int) error {
	if w.exitCode != code {
		return fmt.Errorf("exit code = %d, want %d (outcome %v)\nstderr: %s", w.exitCode, code, w.outcome, w.stderr)
	}
	return nil
}

func (w *proposalReadsWorld) exitNonZeroAPIError() error {
	if w.outcome != APIError || w.exitCode != 3 {
		return fmt.Errorf("outcome=%v exit=%d, want APIError/3\nstderr: %s", w.outcome, w.exitCode, w.stderr)
	}
	return nil
}

func (w *proposalReadsWorld) paramSetTo(param, value string) error {
	t, ok := w.transport.(*cannedTransport)
	if !ok {
		return errors.New("this scenario's transport does not record the query")
	}
	if got := t.lastQuery.Get(param); got != value {
		return fmt.Errorf("the request query should carry %q=%q, got %q (query %q)", param, value, got, t.lastQuery.Encode())
	}
	return nil
}

func (w *proposalReadsWorld) requestSendsParam(param, value string) error {
	return w.paramSetTo(param, value)
}

func (w *proposalReadsWorld) requestSendsTwoParams(p1, v1, p2, v2 string) error {
	if err := w.paramSetTo(p1, v1); err != nil {
		return err
	}
	return w.paramSetTo(p2, v2)
}

func (w *proposalReadsWorld) matchingProposalsPrinted() error {
	// The API does the filtering; the fake returns the (matching) set. The genuine
	// assertion that the right filter was sent lives in the request-param step above;
	// here we assert the matching proposals are projected to stdout.
	if !strings.Contains(w.stdout, "prp_1") {
		return fmt.Errorf("the matching proposals should be printed:\n%s", w.stdout)
	}
	return nil
}

func (w *proposalReadsWorld) literalPrintedToStdout(literal string) error {
	if !strings.Contains(w.stdout, literal) {
		return fmt.Errorf("stdout should contain %q:\n%s", literal, w.stdout)
	}
	return nil
}

func (w *proposalReadsWorld) stderrReportsAndPointsTo(report, pointer string) error {
	if !strings.Contains(strings.ToLower(w.stderr), strings.ToLower(report)) {
		return fmt.Errorf("stderr should report %q:\n%s", report, w.stderr)
	}
	if !strings.Contains(w.stderr, pointer) {
		return fmt.Errorf("stderr should point to %q:\n%s", pointer, w.stderr)
	}
	return nil
}

func (w *proposalReadsWorld) noProposalDataPrinted() error {
	if strings.TrimSpace(w.stdout) != "" {
		return fmt.Errorf("no proposal data should be printed, got stdout:\n%s", w.stdout)
	}
	return nil
}

func (w *proposalReadsWorld) proposalDetailPrinted() error {
	for _, want := range []string{
		"[proposed_outside_meeting]", // status
		"CreateRole",                 // a change by type
		"2 total — 1 no-objection, 1 bring-to-meeting", // aggregate response summary
		"propose, withdraw", // available transitions
	} {
		if !strings.Contains(w.stdout, want) {
			return fmt.Errorf("the single proposal should print its status, changes, response summary, and transitions, missing %q:\n%s", want, w.stdout)
		}
	}
	return nil
}

func (w *proposalReadsWorld) stderrNamesHTTPStatus() error {
	if !strings.Contains(w.stderr, "404") {
		return fmt.Errorf("stderr should name the HTTP status (404):\n%s", w.stderr)
	}
	return nil
}

func (w *proposalReadsWorld) stderrReportsUsageError() error {
	if w.outcome != UsageError || w.exitCode != 2 {
		return fmt.Errorf("outcome=%v exit=%d, want UsageError/2\nstderr: %s", w.outcome, w.exitCode, w.stderr)
	}
	if strings.TrimSpace(w.stderr) == "" {
		return errors.New("a usage error should be reported on stderr")
	}
	return nil
}

func (w *proposalReadsWorld) stderrReportsRejectedOutput(value string) error {
	if err := w.stderrReportsUsageError(); err != nil {
		return err
	}
	if !strings.Contains(w.stderr, value) {
		return fmt.Errorf("stderr should name the rejected output value %q:\n%s", value, w.stderr)
	}
	return nil
}

func (w *proposalReadsWorld) stderrReportsUnsupportedStatus(value string) error {
	if err := w.stderrReportsUsageError(); err != nil {
		return err
	}
	if !strings.Contains(w.stderr, value) {
		return fmt.Errorf("stderr should name the unsupported value %q:\n%s", value, w.stderr)
	}
	// Lists at least one supported status (the sorted set).
	if !strings.Contains(w.stderr, "draft") {
		return fmt.Errorf("stderr should list the supported set:\n%s", w.stderr)
	}
	return nil
}

func (w *proposalReadsWorld) noRequestSent() error {
	if w.transportCalls() != 0 {
		return fmt.Errorf("no request should be sent, but the transport was called %d times", w.transportCalls())
	}
	return nil
}

func (w *proposalReadsWorld) onlyFirstPagePrinted() error {
	if !strings.Contains(w.stdout, "prp_1") {
		return fmt.Errorf("the first page should print:\n%s", w.stdout)
	}
	if w.transportCalls() != 1 {
		return fmt.Errorf("--first-page must not walk, want 1 call, got %d", w.transportCalls())
	}
	return nil
}

func (w *proposalReadsWorld) stderrNotesMoreExist() error {
	if !strings.Contains(w.stderr, "more proposals exist") {
		return fmt.Errorf("stderr should note more proposals exist:\n%s", w.stderr)
	}
	return nil
}
