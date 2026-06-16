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

// TestSubroleFillerRollUpFeatures runs the executable acceptance for Subrole Filler
// Roll-up (051): the `subrole-actors` command driven through its seam over a fake
// base transport, so every scenario runs offline (no real network, no real
// ~/.glassfrogrc). Its Paths name ONLY this spec's feature file — never the
// features/ directory — so the suite reports its own independent scenario count, and
// un-@wip-ing these scenarios cannot disturb another suite (LEARNINGS: a suite points
// at its own feature file). The @validation scenarios stay @wip (held for the
// validate skill) and are skipped by the ~@wip filter.
func TestSubroleFillerRollUpFeatures(t *testing.T) {
	suite := godog.TestSuite{
		ScenarioInitializer: initializeSubroleFillerRollUpScenario,
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"../../features/who-to-contact-for-a-role/subrole-filler-roll-up.feature"},
			Tags:     "~@wip",
			TestingT: t,
		},
	}
	if suite.Run() != 0 {
		t.Fatal("non-zero status: subrole-filler-roll-up feature scenarios failed")
	}
}

// subroleActorWorld is the per-scenario state: the connection context and fake
// transport assembled from the Given steps, plus the captured
// outcome/exit-code/streams. Everything is injected — no step touches the real
// network, env, or home. It is the roll-up sibling of actor-directory's actorWorld.
type subroleActorWorld struct {
	ctx       apiclient.ConnectionContext
	transport http.RoundTripper
	secret    string

	outcome  Outcome
	exitCode int
	stdout   string
	stderr   string
}

func initializeSubroleFillerRollUpScenario(sc *godog.ScenarioContext) {
	w := &subroleActorWorld{}
	sc.Before(func(ctx context.Context, _ *godog.Scenario) (context.Context, error) {
		*w = subroleActorWorld{
			// A single complete mixed-kind page is the default; the per-scenario Given
			// steps override the transport/context as needed.
			transport: &cannedTransport{status: 200, body: actorsPageComplete},
			secret:    meSecretToken,
		}
		w.ctx = validMeContext()
		return ctx, nil
	})

	// --- Givens (shared phrasings reused verbatim from the actor-directory /
	// assignments suites where the text matches; the roll-up-specific Givens are new) ---
	sc.Step(`^a complete connection context with a stored token$`, w.completeContext)
	sc.Step(`^the role "([^"]*)" has direct sub-roles filled by several actors$`, w.subrolesFilledBySeveral)
	sc.Step(`^the role "([^"]*)" has no sub-roles$`, w.roleHasNoSubroles)
	sc.Step(`^no usable token is available to the CLI$`, w.noToken)
	sc.Step(`^the role "([^"]*)" has direct sub-roles filled by no actors$`, w.subrolesFilledByNone)
	sc.Step(`^the role "([^"]*)" has direct sub-roles filled by both people and agents$`, w.subrolesFilledByPeopleAndAgents)
	sc.Step(`^the role "([^"]*)" has sub-role fillers spanning more than one page$`, w.subroleFillersSpanMoreThanOnePage)
	sc.Step(`^the subrole filler roll-up walk fails after retrieving the first page$`, w.walkFailsAfterFirstPage)

	// --- Whens (an agent or a practitioner — same dispatch) ---
	sc.Step(`^an agent runs "glassfrog (.+)"$`, w.runCommand)
	sc.Step(`^a practitioner runs "glassfrog (.+)"$`, w.runCommand)

	// --- Thens ---
	sc.Step(`^the request will read the role's subroles actors endpoint$`, w.requestReadsSubrolesActors)
	sc.Step(`^each sub-role filler will be printed as a projection$`, w.eachFillerPrinted)
	sc.Step(`^the command will exit with code (\d+)$`, w.exitWithCode)
	sc.Step(`^the command will exit with a non-zero API-error code$`, w.exitNonZeroAPIError)
	sc.Step(`^the command will exit with a non-zero code$`, w.exitNonZero)
	sc.Step(`^stderr will report that the read failed and name the HTTP status$`, w.stderrNamesHTTPStatus)
	sc.Step(`^no "this role has no sub-roles" message will be added$`, w.noSubrolesMessageAdded)
	sc.Step(`^stderr will report "([^"]*)" and point to "([^"]*)"$`, w.stderrReportsAndPointsTo)
	sc.Step(`^no actor data will be printed$`, w.noActorDataPrinted)
	sc.Step(`^"([^"]*)" will be printed to stdout$`, w.literalPrintedToStdout)
	sc.Step(`^the request will carry "([^"]*)" set to "([^"]*)"$`, w.requestCarriesParam)
	sc.Step(`^only the agents will be printed as a list$`, w.onlyAgentsPrinted)
	sc.Step(`^stderr will report the unsupported value and list the supported set$`, w.stderrNamesUnsupportedKind)
	sc.Step(`^no API request will be sent$`, w.noRequestSent)
	sc.Step(`^every page of sub-role fillers will be walked$`, w.everyPageWalked)
	sc.Step(`^the complete set will be printed$`, w.completeSetPrinted)
	sc.Step(`^only the first page of actors will be printed$`, w.onlyFirstPagePrinted)
	sc.Step(`^stderr will note that more actors exist$`, w.stderrNotesMoreExist)
	sc.Step(`^the actors retrieved so far will be printed$`, w.partialActorsPrinted)
	sc.Step(`^stderr will note the result is incomplete and name the cause$`, w.stderrNotesIncomplete)
}

// --- Given implementations ---

func (w *subroleActorWorld) completeContext() error { w.ctx = validMeContext(); return nil }

func (w *subroleActorWorld) subrolesFilledBySeveral(_ string) error {
	w.transport = &cannedTransport{status: 200, body: actorsPageComplete}
	return nil
}

// roleHasNoSubroles installs a leaf-anchor 404: the endpoint is only available on
// expanded roles, so a leaf role answers 404 (plan ADR-3). It is genuinely distinct
// from the empty-200 success (sub-roles exist but carry no fillers).
func (w *subroleActorWorld) roleHasNoSubroles(_ string) error {
	w.transport = &cannedTransport{status: 404, body: `{"detail":"role has no subroles"}`}
	return nil
}

func (w *subroleActorWorld) noToken() error {
	w.ctx = noTokenContext()
	return nil
}

func (w *subroleActorWorld) subrolesFilledByNone(_ string) error {
	w.transport = &cannedTransport{status: 200, body: actorsPageEmpty}
	return nil
}

func (w *subroleActorWorld) subrolesFilledByPeopleAndAgents(_ string) error {
	// The API applies the --kind agent filter and returns only agents.
	w.transport = &cannedTransport{status: 200, body: actorsAgentsOnlyPage}
	return nil
}

func (w *subroleActorWorld) subroleFillersSpanMoreThanOnePage(_ string) error {
	w.transport = actorsMultiPage()
	return nil
}

func (w *subroleActorWorld) walkFailsAfterFirstPage() error {
	w.transport = &seqMeTransport{steps: []seqMeResp{
		{status: 200, body: actorsPage("per_1", "Gathered Actor", "human", "c1")},
		{status: 500, body: `{"detail":"boom"}`},
	}}
	return nil
}

// --- When implementation ---

// runCommand parses the captured invocation (quote-aware, reusing the search suite's
// splitArgs) and dispatches it through a real root with only the `subrole-actors`
// leaf attached over a fake seam. It asserts the token never leaks into either stream.
func (w *subroleActorWorld) runCommand(invocation string) error {
	args := splitArgs(invocation)
	root := NewRootCommand()
	seam := &fakeMeSeam{ctx: w.ctx, transport: w.transport}
	MustRegister(root, newSubroleActorsCommand(seam))

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

func (w *subroleActorWorld) transportCalls() int {
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

func (w *subroleActorWorld) lastQuery() (url.Values, error) {
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

func (w *subroleActorWorld) transportLastPath() (string, error) {
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

// requestReadsSubrolesActors confirms the request hit the role-scoped subroles actors
// endpoint (shape /roles/<id>/subroles/actors) rather than the org-wide /actors
// directory.
func (w *subroleActorWorld) requestReadsSubrolesActors() error {
	path, err := w.transportLastPath()
	if err != nil {
		return err
	}
	if !strings.Contains(path, "/roles/") || !strings.HasSuffix(path, "/subroles/actors") {
		return fmt.Errorf("the request should read the role's subroles actors endpoint, got path %q", path)
	}
	return nil
}

func (w *subroleActorWorld) eachFillerPrinted() error {
	for _, want := range []string{"per_0123", "agt_0456"} {
		if !strings.Contains(w.stdout, want) {
			return fmt.Errorf("each sub-role filler should print as a projection; missing %q:\n%s", want, w.stdout)
		}
	}
	return nil
}

func (w *subroleActorWorld) exitWithCode(code int) error {
	if w.exitCode != code {
		return fmt.Errorf("exit code = %d, want %d (outcome %v)\nstderr: %s", w.exitCode, code, w.outcome, w.stderr)
	}
	return nil
}

func (w *subroleActorWorld) exitNonZeroAPIError() error {
	if w.outcome != APIError || w.exitCode != 3 {
		return fmt.Errorf("outcome=%v exit=%d, want APIError/3\nstderr: %s", w.outcome, w.exitCode, w.stderr)
	}
	return nil
}

func (w *subroleActorWorld) exitNonZero() error {
	if w.exitCode == 0 {
		return fmt.Errorf("exit code = 0, want a non-zero code (outcome %v)\nstderr: %s", w.outcome, w.stderr)
	}
	return nil
}

func (w *subroleActorWorld) stderrNamesHTTPStatus() error {
	if !strings.Contains(w.stderr, "404") {
		return fmt.Errorf("stderr should report the read failed and name the HTTP status (404):\n%s", w.stderr)
	}
	return nil
}

// noSubrolesMessageAdded confirms the CLI adds NO "this role has no sub-roles"
// interpretation on a leaf 404 — the status is surfaced as the shared read failure
// (plan ADR-3, VISION Exclusion 1).
func (w *subroleActorWorld) noSubrolesMessageAdded() error {
	if strings.Contains(strings.ToLower(w.stdout+w.stderr), "no sub-roles") {
		return fmt.Errorf("no \"this role has no sub-roles\" message must be added:\nstdout: %s\nstderr: %s", w.stdout, w.stderr)
	}
	return nil
}

func (w *subroleActorWorld) stderrReportsAndPointsTo(report, pointer string) error {
	if !strings.Contains(strings.ToLower(w.stderr), strings.ToLower(report)) {
		return fmt.Errorf("stderr should report %q:\n%s", report, w.stderr)
	}
	if !strings.Contains(w.stderr, pointer) {
		return fmt.Errorf("stderr should point to %q:\n%s", pointer, w.stderr)
	}
	return nil
}

func (w *subroleActorWorld) noActorDataPrinted() error {
	// A not-authenticated failure prints no actor rows (the human projection lines).
	if strings.Contains(w.stdout, "per_") || strings.Contains(w.stdout, "agt_") {
		return fmt.Errorf("no actor data should be printed, got stdout:\n%s", w.stdout)
	}
	return nil
}

func (w *subroleActorWorld) literalPrintedToStdout(literal string) error {
	if !strings.Contains(w.stdout, literal) {
		return fmt.Errorf("stdout should contain %q:\n%s", literal, w.stdout)
	}
	return nil
}

func (w *subroleActorWorld) requestCarriesParam(param, value string) error {
	q, err := w.lastQuery()
	if err != nil {
		return err
	}
	if got := q.Get(param); got != value {
		return fmt.Errorf("request %s = %q, want %q", param, got, value)
	}
	return nil
}

func (w *subroleActorWorld) onlyAgentsPrinted() error {
	if !strings.Contains(w.stdout, "agt_0456") {
		return fmt.Errorf("the agent should print:\n%s", w.stdout)
	}
	if strings.Contains(w.stdout, "per_") {
		return fmt.Errorf("only agents should print for --kind agent (no per_ rows):\n%s", w.stdout)
	}
	return nil
}

func (w *subroleActorWorld) stderrNamesUnsupportedKind() error {
	if !strings.Contains(w.stderr, "robot") {
		return fmt.Errorf("stderr should name the unsupported value:\n%s", w.stderr)
	}
	// The supported set is alphabetically sorted via validateClosedFlagSet — assert by
	// membership, never by a hard-coded order (LEARNINGS).
	for _, want := range []string{"agent", "human"} {
		if !strings.Contains(w.stderr, want) {
			return fmt.Errorf("stderr should list the supported set; missing %q:\n%s", want, w.stderr)
		}
	}
	return nil
}

func (w *subroleActorWorld) noRequestSent() error {
	if w.transportCalls() != 0 {
		return fmt.Errorf("no API request should be sent, but the transport was called %d times", w.transportCalls())
	}
	return nil
}

func (w *subroleActorWorld) everyPageWalked() error {
	if w.transportCalls() < 2 {
		return fmt.Errorf("the walk should issue more than one page request, got %d", w.transportCalls())
	}
	return nil
}

func (w *subroleActorWorld) completeSetPrinted() error {
	for _, want := range []string{"Alice Page One", "Bob Page Two"} {
		if !strings.Contains(w.stdout, want) {
			return fmt.Errorf("the complete set across pages should print; missing %q:\n%s", want, w.stdout)
		}
	}
	return nil
}

func (w *subroleActorWorld) onlyFirstPagePrinted() error {
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

func (w *subroleActorWorld) stderrNotesMoreExist() error {
	if !strings.Contains(w.stderr, "more actors exist") {
		return fmt.Errorf("stderr should note more actors exist:\n%s", w.stderr)
	}
	return nil
}

func (w *subroleActorWorld) partialActorsPrinted() error {
	if !strings.Contains(w.stdout, "Gathered Actor") {
		return fmt.Errorf("the partial set gathered so far should print:\n%s", w.stdout)
	}
	return nil
}

func (w *subroleActorWorld) stderrNotesIncomplete() error {
	if !strings.Contains(w.stderr, "incomplete") {
		return fmt.Errorf("stderr should note the result is incomplete and name the cause:\n%s", w.stderr)
	}
	return nil
}
