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

// TestSubrolesTensionRollUpFeatures runs the executable acceptance for Subroles
// Tension Roll-up (046): the `tension subroles <role-id>` cross-role roll-up, driven
// through the shared tensionSeam over a fake base transport so every scenario runs
// offline (no real network, no real ~/.glassfrogrc). Its Paths name ONLY this spec's
// feature file — never the features/ directory — so the suite reports its own
// independent scenario count and un-@wip-ping these scenarios cannot disturb another
// suite (LEARNINGS: a suite points at its own feature file). The 4 @validation
// scenarios stay @wip (held for the validate skill) and are skipped by the ~@wip
// filter; their unique steps are therefore left undefined here, by design.
func TestSubrolesTensionRollUpFeatures(t *testing.T) {
	suite := godog.TestSuite{
		ScenarioInitializer: initializeSubrolesTensionRollUpScenario,
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"../../features/tension-capture/subroles-tension-roll-up.feature"},
			Tags:     "~@wip",
			TestingT: t,
		},
	}
	if suite.Run() != 0 {
		t.Fatal("non-zero status: subroles-tension-roll-up feature scenarios failed")
	}
}

// subrolesTensionWorld is the per-scenario state: the connection context and fake
// transport assembled from the Given steps, plus the captured
// outcome/exit-code/streams of the When run. Everything is injected — no step touches
// the real network, env, or home. It mirrors tensionReadsWorld (043), reusing the
// sibling-suite step phrasings (godog matches by text) so the shared steps behave
// identically; only the subroles-specific Givens/Thens are new.
type subrolesTensionWorld struct {
	ctx       apiclient.ConnectionContext
	transport http.RoundTripper
	secret    string

	outcome  Outcome
	exitCode int
	stdout   string
	stderr   string
}

func initializeSubrolesTensionRollUpScenario(sc *godog.ScenarioContext) {
	w := &subrolesTensionWorld{}
	sc.Before(func(ctx context.Context, _ *godog.Scenario) (context.Context, error) {
		*w = subrolesTensionWorld{
			// A two-tension single-page body is the default; the per-scenario Given
			// steps override the transport/context as needed.
			transport: &cannedTransport{status: 200, body: tensionsPageComplete},
			secret:    meSecretToken,
		}
		w.ctx = validMeContext()
		return ctx, nil
	})

	// --- Givens --- (shared phrasing reused from the tension-reads suite) ---
	sc.Step(`^a complete connection context with a stored token$`, w.completeContext)
	sc.Step(`^no usable token is available to the CLI$`, w.noToken)
	// --- Givens --- (subroles-specific) ---
	sc.Step(`^the role "([^"]*)" has direct sub-roles carrying several tensions$`, w.subrolesCarrySeveralTensions)
	sc.Step(`^the role "([^"]*)" has no sub-roles$`, w.anchorHasNoSubroles)
	sc.Step(`^the role "([^"]*)" has direct sub-roles carrying no tensions$`, w.subrolesCarryNoTensions)
	sc.Step(`^the role "([^"]*)" has direct sub-roles carrying tensions in several statuses$`, w.subrolesCarryTensionsInStatuses)
	sc.Step(`^the role "([^"]*)" has sub-role tensions spanning more than one page$`, w.subrolesHaveMultiPageTensions)
	sc.Step(`^the subroles tension roll-up walk fails after retrieving the first page$`, w.walkFailsMidway)

	// --- Whens --- (both "an agent" and "a practitioner" drive the same run) ---
	sc.Step(`^(?:an agent|a practitioner) runs "glassfrog (.+)"$`, w.runCommand)

	// --- Thens --- (shared phrasing reused from the tension-reads suite) ---
	sc.Step(`^the command will exit with code (\d+)$`, w.exitWithCode)
	sc.Step(`^the command will exit with a non-zero API-error code$`, w.exitNonZeroAPIError)
	sc.Step(`^the command will exit with a non-zero code$`, w.exitNonZero)
	sc.Step(`^"([^"]*)" will be printed to stdout$`, w.literalPrintedToStdout)
	sc.Step(`^stderr will report "([^"]*)" and point to "([^"]*)"$`, w.stderrReportsAndPointsTo)
	sc.Step(`^no tension data will be printed$`, w.noTensionDataPrinted)
	sc.Step(`^stderr will report that the read failed and name the HTTP status$`, w.stderrNamesHTTPStatus)
	sc.Step(`^no request will be sent$`, w.noRequestSent)
	sc.Step(`^the request will carry the "([^"]*)" parameter set to "([^"]*)"$`, w.requestCarriesParamSetTo)
	sc.Step(`^only the unprocessed tensions will be printed$`, w.onlyUnprocessedTensionsPrinted)
	sc.Step(`^stderr will report the unsupported value and list the supported set$`, w.stderrReportsUnsupportedStatus)
	sc.Step(`^only the first page of tensions will be printed$`, w.onlyFirstPagePrinted)
	sc.Step(`^stderr will note that more tensions exist$`, w.stderrNotesMoreExist)
	sc.Step(`^the tensions retrieved so far will be printed$`, w.partialTensionsPrinted)
	sc.Step(`^stderr will note the result is incomplete and name the cause$`, w.stderrNotesIncomplete)
	// --- Thens --- (subroles-specific) ---
	sc.Step(`^the request will read the role's subroles tensions endpoint$`, w.requestHitSubrolesEndpoint)
	sc.Step(`^each sub-role tension will be printed as a projection$`, w.eachTensionPrinted)
	sc.Step(`^no "this role has no sub-roles" message will be added$`, w.noLeafInterpretationAdded)
	sc.Step(`^every page of sub-role tensions will be walked$`, w.everyPageWalked)
	sc.Step(`^the complete set will be printed$`, w.completeSetPrinted)
}

// --- Given implementations ---

func (w *subrolesTensionWorld) completeContext() error { w.ctx = validMeContext(); return nil }

func (w *subrolesTensionWorld) noToken() error { w.ctx = noTokenContext(); return nil }

func (w *subrolesTensionWorld) subrolesCarrySeveralTensions(_ string) error {
	w.transport = &cannedTransport{status: 200, body: tensionsPageComplete}
	return nil
}

// anchorHasNoSubroles installs the leaf-anchor 404 the API answers for a role with no
// sub-roles. It is a genuinely different fake from the empty-200 success (plan ADR-3):
// a non-2xx that must surface as a read failure, never a clean empty roll-up.
func (w *subrolesTensionWorld) anchorHasNoSubroles(_ string) error {
	w.transport = &cannedTransport{status: 404, body: `{"detail":"Role not found"}`}
	return nil
}

func (w *subrolesTensionWorld) subrolesCarryNoTensions(_ string) error {
	w.transport = &cannedTransport{status: 200, body: tensionsPageEmpty}
	return nil
}

func (w *subrolesTensionWorld) subrolesCarryTensionsInStatuses(_ string) error {
	// A single unprocessed tension so "only the unprocessed tensions will be printed"
	// is a genuine assertion (the API does the filtering; the fake returns the
	// filtered set).
	w.transport = &cannedTransport{status: 200, body: tensionsPage("ten_1", "Unprocessed One", "")}
	return nil
}

// subrolesHaveMultiPageTensions installs a three-page sequence whose first page
// reports more exist. It serves both the full-walk scenario (all three pages consumed)
// and the --first-page opt-out (only the first page, more-exist note) — the run
// command, not the fixture, picks the path.
func (w *subrolesTensionWorld) subrolesHaveMultiPageTensions(_ string) error {
	w.transport = &seqMeTransport{steps: []seqMeResp{
		{status: 200, body: tensionsPage("ten_1", "First Page Tension", "c1")},
		{status: 200, body: tensionsPage("ten_2", "Second Page Tension", "c2")},
		{status: 200, body: tensionsPage("ten_3", "Third Page Tension", "")},
	}}
	return nil
}

func (w *subrolesTensionWorld) walkFailsMidway() error {
	w.transport = &seqMeTransport{steps: []seqMeResp{
		{status: 200, body: tensionsPage("ten_1", "Gathered Tension", "c1")},
		{status: 500, body: `{"detail":"boom"}`},
	}}
	return nil
}

// --- When implementation ---

// runCommand parses the captured invocation and dispatches it through a real root with
// the `tension` group attached over a fake seam — the group parents subroles (046)
// alongside create/list/get/update/discard. It asserts the secret token never leaks.
func (w *subrolesTensionWorld) runCommand(invocation string) error {
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
func (w *subrolesTensionWorld) transportCalls() int {
	switch t := w.transport.(type) {
	case *cannedTransport:
		return t.calls
	case *seqMeTransport:
		return t.calls
	default:
		return -1
	}
}

// lastPath reads the most-recent request path off whichever fake the scenario installed.
func (w *subrolesTensionWorld) lastPath() string {
	switch t := w.transport.(type) {
	case *cannedTransport:
		return t.lastPath
	case *seqMeTransport:
		return t.lastPath
	default:
		return ""
	}
}

// --- Then implementations ---

func (w *subrolesTensionWorld) requestHitSubrolesEndpoint() error {
	got := w.lastPath()
	if !strings.HasSuffix(got, "/subroles/tensions") || !strings.Contains(got, "/roles/") {
		return fmt.Errorf("the request should target /roles/{id}/subroles/tensions, got %q", got)
	}
	return nil
}

func (w *subrolesTensionWorld) eachTensionPrinted() error {
	for _, want := range []string{"ten_1  [unprocessed]  Roadmap drift", "ten_2  [processed]"} {
		if !strings.Contains(w.stdout, want) {
			return fmt.Errorf("each sub-role tension should print as a projection, missing %q:\n%s", want, w.stdout)
		}
	}
	return nil
}

func (w *subrolesTensionWorld) exitWithCode(code int) error {
	if w.exitCode != code {
		return fmt.Errorf("exit code = %d, want %d (outcome %v)\nstderr: %s", w.exitCode, code, w.outcome, w.stderr)
	}
	return nil
}

func (w *subrolesTensionWorld) exitNonZeroAPIError() error {
	if w.outcome != APIError || w.exitCode != 3 {
		return fmt.Errorf("outcome=%v exit=%d, want APIError/3\nstderr: %s", w.outcome, w.exitCode, w.stderr)
	}
	return nil
}

func (w *subrolesTensionWorld) exitNonZero() error {
	if w.exitCode == 0 {
		return fmt.Errorf("exit code = 0, want a non-zero code (outcome %v)\nstderr: %s", w.outcome, w.stderr)
	}
	return nil
}

func (w *subrolesTensionWorld) literalPrintedToStdout(literal string) error {
	if !strings.Contains(w.stdout, literal) {
		return fmt.Errorf("stdout should contain %q:\n%s", literal, w.stdout)
	}
	return nil
}

func (w *subrolesTensionWorld) stderrReportsAndPointsTo(report, pointer string) error {
	if !strings.Contains(strings.ToLower(w.stderr), strings.ToLower(report)) {
		return fmt.Errorf("stderr should report %q:\n%s", report, w.stderr)
	}
	if !strings.Contains(w.stderr, pointer) {
		return fmt.Errorf("stderr should point to %q:\n%s", pointer, w.stderr)
	}
	return nil
}

func (w *subrolesTensionWorld) noTensionDataPrinted() error {
	if strings.TrimSpace(w.stdout) != "" {
		return fmt.Errorf("no tension data should be printed, got stdout:\n%s", w.stdout)
	}
	return nil
}

func (w *subrolesTensionWorld) stderrNamesHTTPStatus() error {
	if !strings.Contains(w.stderr, "404") {
		return fmt.Errorf("stderr should name the HTTP status (404):\n%s", w.stderr)
	}
	return nil
}

// noLeafInterpretationAdded pins plan ADR-3's no-special-case rule: the leaf-anchor 404
// is surfaced verbatim as a read failure, never re-worded into a "no sub-roles" empty
// success.
func (w *subrolesTensionWorld) noLeafInterpretationAdded() error {
	if strings.Contains(strings.ToLower(w.stderr), "no sub-roles") {
		return fmt.Errorf("the 404 must not be re-interpreted as a no-sub-roles message:\n%s", w.stderr)
	}
	return nil
}

func (w *subrolesTensionWorld) noRequestSent() error {
	if w.transportCalls() != 0 {
		return fmt.Errorf("no request should be sent, but the transport was called %d times", w.transportCalls())
	}
	return nil
}

func (w *subrolesTensionWorld) requestCarriesParamSetTo(param, value string) error {
	t, ok := w.transport.(*cannedTransport)
	if !ok {
		return errors.New("this scenario's transport does not record the query")
	}
	if got := t.lastQuery.Get(param); got != value {
		return fmt.Errorf("the request query should carry %q=%q, got %q (query %q)", param, value, got, t.lastQuery.Encode())
	}
	return nil
}

func (w *subrolesTensionWorld) onlyUnprocessedTensionsPrinted() error {
	if !strings.Contains(w.stdout, "[unprocessed]") || !strings.Contains(w.stdout, "Unprocessed One") {
		return fmt.Errorf("the unprocessed tensions should be printed:\n%s", w.stdout)
	}
	return nil
}

func (w *subrolesTensionWorld) stderrReportsUnsupportedStatus() error {
	if w.outcome != UsageError || w.exitCode != 2 {
		return fmt.Errorf("outcome=%v exit=%d, want UsageError/2\nstderr: %s", w.outcome, w.exitCode, w.stderr)
	}
	// Names the rejected value and lists at least one supported status. The supported
	// set is alphabetically sorted; assert membership, never a hard-coded order.
	if !strings.Contains(w.stderr, "open") {
		return fmt.Errorf("stderr should name the unsupported value:\n%s", w.stderr)
	}
	if !strings.Contains(w.stderr, "unprocessed") {
		return fmt.Errorf("stderr should list the supported set:\n%s", w.stderr)
	}
	return nil
}

func (w *subrolesTensionWorld) onlyFirstPagePrinted() error {
	if !strings.Contains(w.stdout, "First Page Tension") {
		return fmt.Errorf("the first page should print:\n%s", w.stdout)
	}
	if w.transportCalls() != 1 {
		return fmt.Errorf("--first-page must not walk, want 1 call, got %d", w.transportCalls())
	}
	return nil
}

func (w *subrolesTensionWorld) stderrNotesMoreExist() error {
	if !strings.Contains(w.stderr, "more tensions exist") {
		return fmt.Errorf("stderr should note more tensions exist:\n%s", w.stderr)
	}
	return nil
}

func (w *subrolesTensionWorld) partialTensionsPrinted() error {
	if !strings.Contains(w.stdout, "Gathered Tension") {
		return fmt.Errorf("the partial set gathered so far should print:\n%s", w.stdout)
	}
	return nil
}

func (w *subrolesTensionWorld) stderrNotesIncomplete() error {
	if !strings.Contains(w.stderr, "incomplete") {
		return fmt.Errorf("stderr should note the result is incomplete and name the cause:\n%s", w.stderr)
	}
	return nil
}

// everyPageWalked pins the full walk: the three-page sequence is consumed to
// completion, so the walker issues exactly three requests.
func (w *subrolesTensionWorld) everyPageWalked() error {
	if w.transportCalls() != 3 {
		return fmt.Errorf("the roll-up should walk every page (3 requests), got %d", w.transportCalls())
	}
	return nil
}

func (w *subrolesTensionWorld) completeSetPrinted() error {
	for _, want := range []string{"ten_1", "ten_2", "ten_3"} {
		if !strings.Contains(w.stdout, want) {
			return fmt.Errorf("the complete set across every page should print, missing %q:\n%s", want, w.stdout)
		}
	}
	return nil
}
