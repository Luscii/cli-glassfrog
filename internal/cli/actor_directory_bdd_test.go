package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/Luscii/cli-glassfrog/internal/apiclient"
	"github.com/cucumber/godog"
)

// TestActorDirectoryFeatures runs the executable acceptance for Actor Directory
// (048): the `actors` command driven through its seam over a fake base transport,
// so every scenario runs offline (no real network, no real ~/.glassfrogrc). Its
// Paths name ONLY this spec's feature file — never the features/ directory — so the
// suite reports its own independent scenario count and un-@wip-ping these scenarios
// cannot disturb another suite (LEARNINGS: a suite points at its own feature file).
// The @validation scenarios stay @wip (held for the validate skill) and are skipped
// by the ~@wip filter.
func TestActorDirectoryFeatures(t *testing.T) {
	suite := godog.TestSuite{
		ScenarioInitializer: initializeActorDirectoryScenario,
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"../../features/actors-disconnected-from-governance/actor-directory.feature"},
			Tags:     "~@wip",
			TestingT: t,
		},
	}
	if suite.Run() != 0 {
		t.Fatal("non-zero status: actor-directory feature scenarios failed")
	}
}

// actorWorld is the per-scenario state: the connection context and fake transport
// assembled from the Given steps, plus the captured outcome/exit-code/streams.
// Everything is injected — no step touches the real network, env, or home.
type actorWorld struct {
	ctx       apiclient.ConnectionContext
	transport http.RoundTripper
	secret    string

	outcome  Outcome
	exitCode int
	stdout   string
	stderr   string
}

func initializeActorDirectoryScenario(sc *godog.ScenarioContext) {
	w := &actorWorld{}
	sc.Before(func(ctx context.Context, _ *godog.Scenario) (context.Context, error) {
		*w = actorWorld{
			// A single complete mixed-kind page is the default; the per-scenario Given
			// steps override the transport/context as needed.
			transport: &cannedTransport{status: 200, body: actorsPageComplete},
			secret:    meSecretToken,
		}
		w.ctx = validMeContext()
		return ctx, nil
	})

	// --- Givens ---
	sc.Step(`^a complete connection context with a stored token$`, w.completeContext)
	sc.Step(`^a role that two actors fill$`, w.roleTwoActorsFill)
	sc.Step(`^the API cannot parse the submitted role-id filter$`, w.apiRejectsRoleID)
	sc.Step(`^no usable token is available to the CLI$`, w.noToken)
	sc.Step(`^the organization has several actors$`, w.severalActors)
	sc.Step(`^no actor's name matches "([^"]*)"$`, w.noActorMatches)
	sc.Step(`^the organization has people and agents$`, w.peopleAndAgents)
	sc.Step(`^the organization's actors span more than one page$`, w.actorsSpanMoreThanOnePage)
	sc.Step(`^the page walk fails after retrieving the first page$`, w.walkFailsAfterFirstPage)

	// --- Whens ---
	sc.Step(`^an agent runs "glassfrog (.+)"$`, w.runCommand)

	// --- Thens ---
	sc.Step(`^the request will carry "([^"]*)" set to "([^"]*)"$`, w.requestCarriesParam)
	sc.Step(`^the request will carry no kind, role, or query filter$`, w.requestCarriesNoFilter)
	sc.Step(`^both actors will be printed as a list$`, w.bothActorsPrinted)
	sc.Step(`^every actor will be printed as a list$`, w.everyActorPrinted)
	sc.Step(`^only the agents will be printed as a list$`, w.onlyAgentsPrinted)
	sc.Step(`^the command will exit with code (\d+)$`, w.exitWithCode)
	sc.Step(`^the command will exit with a non-zero API-error code$`, w.exitNonZeroAPIError)
	sc.Step(`^the command will exit with a non-zero code$`, w.exitNonZero)
	sc.Step(`^stderr will report that the read failed and name the HTTP status$`, w.stderrNamesHTTPStatus)
	sc.Step(`^stderr will report "([^"]*)" and point to "([^"]*)"$`, w.stderrReportsAndPointsTo)
	sc.Step(`^no actors will be printed$`, w.noActorsPrinted)
	sc.Step(`^"([^"]*)" will be printed to stdout$`, w.literalPrintedToStdout)
	sc.Step(`^stderr will report the unsupported value and list the supported set$`, w.stderrNamesUnsupportedKind)
	sc.Step(`^no API request will be sent$`, w.noRequestSent)
	sc.Step(`^only the first page of actors will be printed$`, w.onlyFirstPagePrinted)
	sc.Step(`^stderr will note that more actors exist$`, w.stderrNotesMoreExist)
	sc.Step(`^every page will be walked and the complete set of actors will be printed$`, w.everyPageWalked)
	sc.Step(`^the actors retrieved so far will be printed$`, w.partialActorsPrinted)
	sc.Step(`^stderr will note the result is incomplete and name the cause$`, w.stderrNotesIncomplete)
}

// --- multi-page fixtures ---
//
// Page one carries Alice (human), page two carries Bob (agent); the distinct names
// let the first-page scenario assert page two is NOT printed while the walk
// scenario asserts both are.
func actorsMultiPage() *recordingSeqTransport {
	return &recordingSeqTransport{steps: []seqMeResp{
		{status: 200, body: actorsPage("per_1", "Alice Page One", "human", "c1")},
		{status: 200, body: actorsPage("agt_2", "Bob Page Two", "agent", "")},
	}}
}

// --- Given implementations ---

func (w *actorWorld) completeContext() error { w.ctx = validMeContext(); return nil }

func (w *actorWorld) roleTwoActorsFill() error {
	w.transport = &cannedTransport{status: 200, body: actorsPageComplete}
	return nil
}

func (w *actorWorld) apiRejectsRoleID() error {
	w.transport = &cannedTransport{status: 400, body: `{"detail":"invalid role id"}`}
	return nil
}

func (w *actorWorld) noToken() error {
	w.ctx = noTokenContext()
	return nil
}

func (w *actorWorld) severalActors() error {
	w.transport = &cannedTransport{status: 200, body: actorsPageComplete}
	return nil
}

func (w *actorWorld) noActorMatches(_ string) error {
	w.transport = &cannedTransport{status: 200, body: actorsPageEmpty}
	return nil
}

func (w *actorWorld) peopleAndAgents() error {
	// The API applies the --kind agent filter and returns only agents.
	w.transport = &cannedTransport{status: 200, body: actorsAgentsOnlyPage}
	return nil
}

func (w *actorWorld) actorsSpanMoreThanOnePage() error {
	w.transport = actorsMultiPage()
	return nil
}

func (w *actorWorld) walkFailsAfterFirstPage() error {
	w.transport = &seqMeTransport{steps: []seqMeResp{
		{status: 200, body: actorsPage("per_1", "Gathered Actor", "human", "c1")},
		{status: 500, body: `{"detail":"boom"}`},
	}}
	return nil
}

// --- When implementation ---

// runCommand parses the captured invocation (quote-aware, reusing the search
// suite's splitArgs so a quoted value stays one argument) and dispatches it through
// a real root with only the `actors` leaf attached over a fake seam. It asserts the
// token never leaks into either stream.
func (w *actorWorld) runCommand(invocation string) error {
	args := splitArgs(invocation)
	root := NewRootCommand()
	seam := &fakeMeSeam{ctx: w.ctx, transport: w.transport}
	MustRegister(root, newActorsCommand(seam))

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
func (w *actorWorld) transportCalls() int {
	switch t := w.transport.(type) {
	case *cannedTransport:
		return t.calls
	case *seqMeTransport:
		return t.calls
	case *recordingSeqTransport:
		return t.calls
	default:
		return -1
	}
}

// lastQuery reads the most-recent request's query off whichever recording fake the
// scenario installed.
func (w *actorWorld) lastQuery() (url.Values, error) {
	switch t := w.transport.(type) {
	case *cannedTransport:
		return t.lastQuery, nil
	case *recordingSeqTransport:
		if len(t.queries) == 0 {
			return nil, errors.New("no request was recorded")
		}
		return t.queries[len(t.queries)-1], nil
	default:
		return nil, errors.New("this scenario's transport does not record the query")
	}
}

// --- Then implementations ---

func (w *actorWorld) requestCarriesParam(param, value string) error {
	q, err := w.lastQuery()
	if err != nil {
		return err
	}
	if got := q.Get(param); got != value {
		return fmt.Errorf("request %s = %q, want %q", param, got, value)
	}
	return nil
}

func (w *actorWorld) requestCarriesNoFilter() error {
	q, err := w.lastQuery()
	if err != nil {
		return err
	}
	for _, param := range []string{"kind", "role_id", "q"} {
		if _, present := q[param]; present {
			return fmt.Errorf("expected no filter, but the request carried %s=%v", param, q[param])
		}
	}
	return nil
}

func (w *actorWorld) bothActorsPrinted() error {
	for _, want := range []string{"Alice Smith", "Claude"} {
		if !strings.Contains(w.stdout, want) {
			return fmt.Errorf("both actors should print; missing %q:\n%s", want, w.stdout)
		}
	}
	return nil
}

func (w *actorWorld) everyActorPrinted() error {
	for _, want := range []string{"per_0123", "agt_0456"} {
		if !strings.Contains(w.stdout, want) {
			return fmt.Errorf("every actor should print; missing %q:\n%s", want, w.stdout)
		}
	}
	return nil
}

func (w *actorWorld) onlyAgentsPrinted() error {
	if !strings.Contains(w.stdout, "agt_0456") {
		return fmt.Errorf("the agent should print:\n%s", w.stdout)
	}
	if strings.Contains(w.stdout, "per_") {
		return fmt.Errorf("only agents should print for --kind agent (no per_ rows):\n%s", w.stdout)
	}
	return nil
}

func (w *actorWorld) exitWithCode(code int) error {
	if w.exitCode != code {
		return fmt.Errorf("exit code = %d, want %d (outcome %v)\nstderr: %s", w.exitCode, code, w.outcome, w.stderr)
	}
	return nil
}

func (w *actorWorld) exitNonZeroAPIError() error {
	if w.outcome != APIError || w.exitCode != 3 {
		return fmt.Errorf("outcome=%v exit=%d, want APIError/3\nstderr: %s", w.outcome, w.exitCode, w.stderr)
	}
	return nil
}

func (w *actorWorld) exitNonZero() error {
	if w.exitCode == 0 {
		return fmt.Errorf("exit code = 0, want a non-zero code (outcome %v)\nstderr: %s", w.outcome, w.stderr)
	}
	return nil
}

func (w *actorWorld) stderrNamesHTTPStatus() error {
	if !strings.Contains(w.stderr, "400") {
		return fmt.Errorf("stderr should report the read failed and name the HTTP status (400):\n%s", w.stderr)
	}
	return nil
}

func (w *actorWorld) stderrReportsAndPointsTo(report, pointer string) error {
	if !strings.Contains(strings.ToLower(w.stderr), strings.ToLower(report)) {
		return fmt.Errorf("stderr should report %q:\n%s", report, w.stderr)
	}
	if !strings.Contains(w.stderr, pointer) {
		return fmt.Errorf("stderr should point to %q:\n%s", pointer, w.stderr)
	}
	return nil
}

func (w *actorWorld) noActorsPrinted() error {
	// A not-authenticated failure prints no actor rows (the human projection lines).
	if strings.Contains(w.stdout, "per_") || strings.Contains(w.stdout, "agt_") {
		return fmt.Errorf("no actors should be printed, got stdout:\n%s", w.stdout)
	}
	return nil
}

func (w *actorWorld) literalPrintedToStdout(literal string) error {
	if !strings.Contains(w.stdout, literal) {
		return fmt.Errorf("stdout should contain %q:\n%s", literal, w.stdout)
	}
	return nil
}

func (w *actorWorld) stderrNamesUnsupportedKind() error {
	if !strings.Contains(w.stderr, "robot") {
		return fmt.Errorf("stderr should name the unsupported value:\n%s", w.stderr)
	}
	for _, want := range []string{"agent", "human"} {
		if !strings.Contains(w.stderr, want) {
			return fmt.Errorf("stderr should list the supported set; missing %q:\n%s", want, w.stderr)
		}
	}
	return nil
}

func (w *actorWorld) noRequestSent() error {
	if w.transportCalls() != 0 {
		return fmt.Errorf("no API request should be sent, but the transport was called %d times", w.transportCalls())
	}
	return nil
}

func (w *actorWorld) onlyFirstPagePrinted() error {
	if !strings.Contains(w.stdout, "Alice Page One") {
		return fmt.Errorf("the first page should print:\n%s", w.stdout)
	}
	if strings.Contains(w.stdout, "Bob Page Two") {
		return fmt.Errorf("--first-page must not print later pages:\n%s", w.stdout)
	}
	if w.transportCalls() != 1 {
		return fmt.Errorf("--first-page must not walk; want 1 call, got %d", w.transportCalls())
	}
	return nil
}

func (w *actorWorld) stderrNotesMoreExist() error {
	if !strings.Contains(w.stderr, "more actors exist") {
		return fmt.Errorf("stderr should note more actors exist:\n%s", w.stderr)
	}
	return nil
}

func (w *actorWorld) everyPageWalked() error {
	if w.transportCalls() < 2 {
		return fmt.Errorf("the walk should issue more than one page request, got %d", w.transportCalls())
	}
	for _, want := range []string{"Alice Page One", "Bob Page Two"} {
		if !strings.Contains(w.stdout, want) {
			return fmt.Errorf("the complete set across pages should print; missing %q:\n%s", want, w.stdout)
		}
	}
	return nil
}

func (w *actorWorld) partialActorsPrinted() error {
	if !strings.Contains(w.stdout, "Gathered Actor") {
		return fmt.Errorf("the partial set gathered so far should print:\n%s", w.stdout)
	}
	return nil
}

func (w *actorWorld) stderrNotesIncomplete() error {
	if !strings.Contains(w.stderr, "incomplete") {
		return fmt.Errorf("stderr should note the result is incomplete and name the cause:\n%s", w.stderr)
	}
	return nil
}
