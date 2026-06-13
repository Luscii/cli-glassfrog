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

// TestRoleFillersFeatures runs the executable acceptance for Role Fillers (047):
// the `fillers` command driven through its seam over a fake base transport, so
// every scenario runs offline (no real network, no real ~/.glassfrogrc). Its Paths
// name ONLY this spec's feature file — never the features/ directory — so the suite
// reports its own independent scenario count and un-@wip-ping these scenarios cannot
// disturb another suite (LEARNINGS: a suite points at its own feature file). The
// @validation scenarios stay @wip (held for the validate skill) and are skipped by
// the ~@wip filter.
func TestRoleFillersFeatures(t *testing.T) {
	suite := godog.TestSuite{
		ScenarioInitializer: initializeRoleFillersScenario,
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"../../features/who-to-contact-for-a-role/role-fillers.feature"},
			Tags:     "~@wip",
			TestingT: t,
		},
	}
	if suite.Run() != 0 {
		t.Fatal("non-zero status: role-fillers feature scenarios failed")
	}
}

// fillerWorld is the per-scenario state: the connection context and fake transport
// assembled from the Given steps, plus the captured outcome/exit-code/streams.
// Everything is injected — no step touches the real network, env, or home.
type fillerWorld struct {
	ctx       apiclient.ConnectionContext
	transport http.RoundTripper
	secret    string

	outcome  Outcome
	exitCode int
	stdout   string
	stderr   string
}

func initializeRoleFillersScenario(sc *godog.ScenarioContext) {
	w := &fillerWorld{}
	sc.Before(func(ctx context.Context, _ *godog.Scenario) (context.Context, error) {
		*w = fillerWorld{
			// A single complete person+agent page is the default; the per-scenario Given
			// steps override the transport/context as needed.
			transport: &cannedTransport{status: 200, body: fillersPageComplete},
			secret:    meSecretToken,
		}
		w.ctx = validMeContext()
		return ctx, nil
	})

	// --- Givens ---
	sc.Step(`^a complete connection context with a stored token$`, w.completeContext)
	sc.Step(`^the role "([^"]*)" is filled by two actors$`, w.roleFilledByTwo)
	sc.Step(`^the role "([^"]*)" is filled by one person and one agent$`, w.roleFilledByPersonAndAgent)
	sc.Step(`^no role "([^"]*)" exists$`, w.noSuchRole)
	sc.Step(`^no usable token is available to the CLI$`, w.noToken)
	sc.Step(`^the role "([^"]*)" is filled by no actors$`, w.roleFilledByNone)
	sc.Step(`^the role "([^"]*)" is filled by an actor whose assignment has a focus and an election date$`, w.roleFilledByActorWithFocusAndElection)
	sc.Step(`^the role "([^"]*)" has fillers spanning more than one page$`, w.fillersSpanMoreThanOnePage)
	sc.Step(`^the filler list walk fails after retrieving the first page$`, w.walkFailsAfterFirstPage)

	// --- Whens (an agent or a practitioner — same dispatch) ---
	sc.Step(`^an agent runs "glassfrog (.+)"$`, w.runCommand)
	sc.Step(`^a practitioner runs "glassfrog (.+)"$`, w.runCommand)

	// --- Thens ---
	sc.Step(`^the request will read the role's assignments endpoint$`, w.requestReadsAssignments)
	sc.Step(`^each filler will be printed as a projection$`, w.eachFillerPrinted)
	sc.Step(`^both fillers will be printed$`, w.bothFillersPrinted)
	sc.Step(`^each filler will show whether it is a person or an agent$`, w.eachFillerShowsKind)
	sc.Step(`^the filler's focus and election expiry will be printed$`, w.focusAndElectionPrinted)
	sc.Step(`^the command will exit with code (\d+)$`, w.exitWithCode)
	sc.Step(`^the command will exit with a non-zero API-error code$`, w.exitNonZeroAPIError)
	sc.Step(`^the command will exit with a non-zero code$`, w.exitNonZero)
	sc.Step(`^stderr will report that the read failed and name the HTTP status$`, w.stderrNamesHTTPStatus)
	sc.Step(`^stderr will report "([^"]*)" and point to "([^"]*)"$`, w.stderrReportsAndPointsTo)
	sc.Step(`^no filler data will be printed$`, w.noFillerDataPrinted)
	sc.Step(`^"([^"]*)" will be printed to stdout$`, w.literalPrintedToStdout)
	sc.Step(`^stderr will report a usage error$`, w.stderrReportsUsageError)
	sc.Step(`^no API request will be sent$`, w.noRequestSent)
	sc.Step(`^only the first page of fillers will be printed$`, w.onlyFirstPagePrinted)
	sc.Step(`^stderr will note that more fillers exist$`, w.stderrNotesMoreExist)
	sc.Step(`^the fillers retrieved so far will be printed$`, w.partialFillersPrinted)
	sc.Step(`^stderr will note the result is incomplete and name the cause$`, w.stderrNotesIncomplete)
}

// --- multi-page fixture ---
//
// Page one carries the first filler, page two a distinctly-named second filler; the
// distinct names let the first-page scenario assert page two is NOT printed.
func fillersMultiPage() *seqMeTransport {
	return &seqMeTransport{steps: []seqMeResp{
		{status: 200, body: fillersPage("per_1", "First Page Filler", "human", "c1")},
		{status: 200, body: fillersPage("per_2", "Second Page Filler", "human", "")},
	}}
}

// --- Given implementations ---

func (w *fillerWorld) completeContext() error { w.ctx = validMeContext(); return nil }

func (w *fillerWorld) roleFilledByTwo(_ string) error {
	w.transport = &cannedTransport{status: 200, body: fillersPageComplete}
	return nil
}

func (w *fillerWorld) roleFilledByPersonAndAgent(_ string) error {
	// fillersPageComplete carries a person (per_0123 / human) and an agent
	// (agt_0456 / agent).
	w.transport = &cannedTransport{status: 200, body: fillersPageComplete}
	return nil
}

func (w *fillerWorld) noSuchRole(_ string) error {
	w.transport = &cannedTransport{status: 404, body: `{"detail":"Role not found"}`}
	return nil
}

func (w *fillerWorld) noToken() error {
	w.ctx = noTokenContext()
	return nil
}

func (w *fillerWorld) roleFilledByNone(_ string) error {
	w.transport = &cannedTransport{status: 200, body: fillersPageEmpty}
	return nil
}

func (w *fillerWorld) roleFilledByActorWithFocusAndElection(_ string) error {
	// fillersPageComplete's first row (Alice) carries a focus ("Keep the lights on")
	// and an election date ("2026-12-31").
	w.transport = &cannedTransport{status: 200, body: fillersPageComplete}
	return nil
}

func (w *fillerWorld) fillersSpanMoreThanOnePage(_ string) error {
	w.transport = fillersMultiPage()
	return nil
}

func (w *fillerWorld) walkFailsAfterFirstPage() error {
	w.transport = &seqMeTransport{steps: []seqMeResp{
		{status: 200, body: fillersPage("per_1", "Gathered Filler", "human", "c1")},
		{status: 500, body: `{"detail":"boom"}`},
	}}
	return nil
}

// --- When implementation ---

// runCommand parses the captured invocation (quote-aware, reusing the search
// suite's splitArgs) and dispatches it through a real root with only the `fillers`
// leaf attached over a fake seam. It asserts the token never leaks into either
// stream.
func (w *fillerWorld) runCommand(invocation string) error {
	args := splitArgs(invocation)
	root := NewRootCommand()
	seam := &fakeMeSeam{ctx: w.ctx, transport: w.transport}
	MustRegister(root, newFillersCommand(seam))

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

// --- helpers ---

// transportCalls reads the request count off whichever fake transport the scenario
// installed.
func (w *fillerWorld) transportCalls() int {
	switch t := w.transport.(type) {
	case *cannedTransport:
		return t.calls
	case *seqMeTransport:
		return t.calls
	default:
		return -1
	}
}

// transportLastPath reads the most-recent request's path off whichever fake the
// scenario installed.
func (w *fillerWorld) transportLastPath() (string, error) {
	switch t := w.transport.(type) {
	case *cannedTransport:
		return t.lastPath, nil
	case *seqMeTransport:
		return t.lastPath, nil
	default:
		return "", errors.New("this scenario's transport does not record the path")
	}
}

// --- Then implementations ---

func (w *fillerWorld) requestReadsAssignments() error {
	path, err := w.transportLastPath()
	if err != nil {
		return err
	}
	if !strings.HasSuffix(path, "/roles/role_0123/assignments") {
		return fmt.Errorf("the request should read the role's assignments endpoint, got path %q", path)
	}
	return nil
}

func (w *fillerWorld) eachFillerPrinted() error {
	for _, want := range []string{"per_0123", "agt_0456"} {
		if !strings.Contains(w.stdout, want) {
			return fmt.Errorf("each filler should print as a projection; missing %q:\n%s", want, w.stdout)
		}
	}
	return nil
}

func (w *fillerWorld) bothFillersPrinted() error {
	for _, want := range []string{"Alice Smith", "Claude"} {
		if !strings.Contains(w.stdout, want) {
			return fmt.Errorf("both fillers should print; missing %q:\n%s", want, w.stdout)
		}
	}
	return nil
}

func (w *fillerWorld) eachFillerShowsKind() error {
	for _, want := range []string{"[human]", "[agent]"} {
		if !strings.Contains(w.stdout, want) {
			return fmt.Errorf("each filler should show whether it is a person or an agent; missing %q:\n%s", want, w.stdout)
		}
	}
	return nil
}

func (w *fillerWorld) focusAndElectionPrinted() error {
	for _, want := range []string{"Keep the lights on", "2026-12-31"} {
		if !strings.Contains(w.stdout, want) {
			return fmt.Errorf("the filler's focus and election expiry should print; missing %q:\n%s", want, w.stdout)
		}
	}
	return nil
}

func (w *fillerWorld) exitWithCode(code int) error {
	if w.exitCode != code {
		return fmt.Errorf("exit code = %d, want %d (outcome %v)\nstderr: %s", w.exitCode, code, w.outcome, w.stderr)
	}
	return nil
}

func (w *fillerWorld) exitNonZeroAPIError() error {
	if w.outcome != APIError || w.exitCode != 3 {
		return fmt.Errorf("outcome=%v exit=%d, want APIError/3\nstderr: %s", w.outcome, w.exitCode, w.stderr)
	}
	return nil
}

func (w *fillerWorld) exitNonZero() error {
	if w.exitCode == 0 {
		return fmt.Errorf("exit code = 0, want a non-zero code (outcome %v)\nstderr: %s", w.outcome, w.stderr)
	}
	return nil
}

func (w *fillerWorld) stderrNamesHTTPStatus() error {
	if !strings.Contains(w.stderr, "404") {
		return fmt.Errorf("stderr should report the read failed and name the HTTP status (404):\n%s", w.stderr)
	}
	return nil
}

func (w *fillerWorld) stderrReportsAndPointsTo(report, pointer string) error {
	if !strings.Contains(strings.ToLower(w.stderr), strings.ToLower(report)) {
		return fmt.Errorf("stderr should report %q:\n%s", report, w.stderr)
	}
	if !strings.Contains(w.stderr, pointer) {
		return fmt.Errorf("stderr should point to %q:\n%s", pointer, w.stderr)
	}
	return nil
}

func (w *fillerWorld) noFillerDataPrinted() error {
	// A not-authenticated failure prints no filler rows (the human projection leads
	// every row with a per_/agt_ id).
	if strings.Contains(w.stdout, "per_") || strings.Contains(w.stdout, "agt_") {
		return fmt.Errorf("no filler data should be printed, got stdout:\n%s", w.stdout)
	}
	return nil
}

func (w *fillerWorld) literalPrintedToStdout(literal string) error {
	if !strings.Contains(w.stdout, literal) {
		return fmt.Errorf("stdout should contain %q:\n%s", literal, w.stdout)
	}
	return nil
}

func (w *fillerWorld) stderrReportsUsageError() error {
	if w.outcome != UsageError || w.exitCode != 2 {
		return fmt.Errorf("outcome=%v exit=%d, want UsageError/2\nstderr: %s", w.outcome, w.exitCode, w.stderr)
	}
	if strings.TrimSpace(w.stderr) == "" {
		return errors.New("a usage error should be reported on stderr")
	}
	return nil
}

func (w *fillerWorld) noRequestSent() error {
	if w.transportCalls() != 0 {
		return fmt.Errorf("no API request should be sent, but the transport was called %d times", w.transportCalls())
	}
	return nil
}

func (w *fillerWorld) onlyFirstPagePrinted() error {
	if !strings.Contains(w.stdout, "First Page Filler") {
		return fmt.Errorf("the first page should print:\n%s", w.stdout)
	}
	if strings.Contains(w.stdout, "Second Page Filler") {
		return fmt.Errorf("--first-page must not print later pages:\n%s", w.stdout)
	}
	if w.transportCalls() != 1 {
		return fmt.Errorf("--first-page must not walk; want 1 call, got %d", w.transportCalls())
	}
	return nil
}

func (w *fillerWorld) stderrNotesMoreExist() error {
	if !strings.Contains(w.stderr, "more fillers exist") {
		return fmt.Errorf("stderr should note more fillers exist:\n%s", w.stderr)
	}
	return nil
}

func (w *fillerWorld) partialFillersPrinted() error {
	if !strings.Contains(w.stdout, "Gathered Filler") {
		return fmt.Errorf("the partial set gathered so far should print:\n%s", w.stdout)
	}
	return nil
}

func (w *fillerWorld) stderrNotesIncomplete() error {
	if !strings.Contains(w.stderr, "incomplete") {
		return fmt.Errorf("stderr should note the result is incomplete and name the cause:\n%s", w.stderr)
	}
	return nil
}
