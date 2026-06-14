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

// TestActorAssignmentsFeatures runs the executable acceptance for Actor Assignments
// (050): the `assignments` command driven through its seam over a fake base
// transport, so every scenario runs offline (no real network, no real
// ~/.glassfrogrc). Its Paths name ONLY this spec's feature file — never the
// features/ directory — so the suite reports its own independent scenario count, and
// un-@wip-ing these scenarios cannot disturb another suite (LEARNINGS: a suite points
// at its own feature file). The @validation scenarios stay @wip (held for the
// validate skill) and are skipped by the ~@wip filter.
func TestActorAssignmentsFeatures(t *testing.T) {
	suite := godog.TestSuite{
		ScenarioInitializer: initializeActorAssignmentsScenario,
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"../../features/an-actors-governance-footprint/actor-assignments.feature"},
			Tags:     "~@wip",
			TestingT: t,
		},
	}
	if suite.Run() != 0 {
		t.Fatal("non-zero status: actor-assignments feature scenarios failed")
	}
}

// assignmentWorld is the per-scenario state: the connection context and fake
// transport assembled from the Given steps, plus the captured
// outcome/exit-code/streams. Everything is injected — no step touches the real
// network, env, or home. It is the actor-end sibling of role-fillers' fillerWorld.
type assignmentWorld struct {
	ctx       apiclient.ConnectionContext
	transport http.RoundTripper
	secret    string

	outcome  Outcome
	exitCode int
	stdout   string
	stderr   string
}

func initializeActorAssignmentsScenario(sc *godog.ScenarioContext) {
	w := &assignmentWorld{}
	sc.Before(func(ctx context.Context, _ *godog.Scenario) (context.Context, error) {
		*w = assignmentWorld{
			// A single complete two-role page is the default; the per-scenario Given
			// steps override the transport/context as needed.
			transport: &cannedTransport{status: 200, body: assignmentsPageComplete},
			secret:    meSecretToken,
		}
		w.ctx = validMeContext()
		return ctx, nil
	})

	// --- Givens (shared phrasings reused verbatim from the roles/projects/fillers
	// suites where the text matches) ---
	sc.Step(`^a complete connection context with a stored token$`, w.completeContext)
	sc.Step(`^the actor "([^"]*)" fills two roles$`, w.actorFillsTwoRoles)
	sc.Step(`^the person "([^"]*)" and the agent "([^"]*)" each fill a role$`, w.personAndAgentEachFillARole)
	sc.Step(`^no actor "([^"]*)" exists$`, w.noSuchActor)
	sc.Step(`^no usable token is available to the CLI$`, w.noToken)
	sc.Step(`^the actor "([^"]*)" fills no roles$`, w.actorFillsNoRoles)
	sc.Step(`^the actor "([^"]*)" fills a role through an assignment with a focus and an election date$`, w.actorFillsRoleWithFocusAndElection)
	sc.Step(`^the actor "([^"]*)" has assignments spanning more than one page$`, w.assignmentsSpanMoreThanOnePage)
	sc.Step(`^the assignment list walk fails after retrieving the first page$`, w.walkFailsAfterFirstPage)

	// --- Whens (an agent or a practitioner — same dispatch) ---
	sc.Step(`^an agent runs "glassfrog (.+)"$`, w.runCommand)
	sc.Step(`^a practitioner runs "glassfrog (.+)"$`, w.runCommand)

	// --- Thens ---
	sc.Step(`^the request will read the actor's assignments endpoint$`, w.requestReadsAssignments)
	sc.Step(`^each assignment will be printed as a projection naming its filled role$`, w.eachAssignmentNamesFilledRole)
	sc.Step(`^the agent's assignments will be printed$`, w.agentAssignmentsPrinted)
	sc.Step(`^the assignment's focus and election expiry will be printed$`, w.focusAndElectionPrinted)
	sc.Step(`^the command will exit with code (\d+)$`, w.exitWithCode)
	sc.Step(`^the command will exit with a non-zero API-error code$`, w.exitNonZeroAPIError)
	sc.Step(`^the command will exit with a non-zero code$`, w.exitNonZero)
	sc.Step(`^stderr will report that the read failed and name the HTTP status$`, w.stderrNamesHTTPStatus)
	sc.Step(`^stderr will report "([^"]*)" and point to "([^"]*)"$`, w.stderrReportsAndPointsTo)
	sc.Step(`^no assignment data will be printed$`, w.noAssignmentDataPrinted)
	sc.Step(`^"([^"]*)" will be printed to stdout$`, w.literalPrintedToStdout)
	sc.Step(`^stderr will report a usage error$`, w.stderrReportsUsageError)
	sc.Step(`^no API request will be sent$`, w.noRequestSent)
	sc.Step(`^only the first page of assignments will be printed$`, w.onlyFirstPagePrinted)
	sc.Step(`^stderr will note that more assignments exist$`, w.stderrNotesMoreExist)
	sc.Step(`^the assignments retrieved so far will be printed$`, w.partialAssignmentsPrinted)
	sc.Step(`^stderr will note the result is incomplete and name the cause$`, w.stderrNotesIncomplete)
}

// --- multi-page fixture ---
//
// Page one carries the first role, page two a distinctly-named second role; the
// distinct names let the first-page scenario assert page two is NOT printed.
func assignmentsMultiPage() *seqMeTransport {
	return &seqMeTransport{steps: []seqMeResp{
		{status: 200, body: assignmentsPage("per_0123", "role_1", "First Page Role", "c1")},
		{status: 200, body: assignmentsPage("per_0123", "role_2", "Second Page Role", "")},
	}}
}

// --- Given implementations ---

func (w *assignmentWorld) completeContext() error { w.ctx = validMeContext(); return nil }

func (w *assignmentWorld) actorFillsTwoRoles(_ string) error {
	w.transport = &cannedTransport{status: 200, body: assignmentsPageComplete}
	return nil
}

func (w *assignmentWorld) personAndAgentEachFillARole(_, _ string) error {
	// The When reads the agent's assignments endpoint, so the canned body is the
	// agent's single-role page (the person is read at the role end / a sibling read).
	w.transport = &cannedTransport{status: 200, body: assignmentsPage("agt_0456", "role_a", "Marketing Lead", "")}
	return nil
}

func (w *assignmentWorld) noSuchActor(_ string) error {
	w.transport = &cannedTransport{status: 404, body: `{"detail":"Actor not found"}`}
	return nil
}

func (w *assignmentWorld) noToken() error {
	w.ctx = noTokenContext()
	return nil
}

func (w *assignmentWorld) actorFillsNoRoles(_ string) error {
	w.transport = &cannedTransport{status: 200, body: assignmentsPageEmpty}
	return nil
}

func (w *assignmentWorld) actorFillsRoleWithFocusAndElection(_ string) error {
	// assignmentsPageComplete's first row carries a focus ("Keep the lights on")
	// and an election date ("2026-12-31").
	w.transport = &cannedTransport{status: 200, body: assignmentsPageComplete}
	return nil
}

func (w *assignmentWorld) assignmentsSpanMoreThanOnePage(_ string) error {
	w.transport = assignmentsMultiPage()
	return nil
}

func (w *assignmentWorld) walkFailsAfterFirstPage() error {
	w.transport = &seqMeTransport{steps: []seqMeResp{
		{status: 200, body: assignmentsPage("per_0123", "role_1", "Gathered Role", "c1")},
		{status: 500, body: `{"detail":"boom"}`},
	}}
	return nil
}

// --- When implementation ---

// runCommand parses the captured invocation (quote-aware, reusing the search suite's
// splitArgs) and dispatches it through a real root with only the `assignments` leaf
// attached over a fake seam. It asserts the token never leaks into either stream.
func (w *assignmentWorld) runCommand(invocation string) error {
	args := splitArgs(invocation)
	root := NewRootCommand()
	seam := &fakeMeSeam{ctx: w.ctx, transport: w.transport}
	MustRegister(root, newAssignmentsCommand(seam))

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
func (w *assignmentWorld) transportCalls() int {
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
func (w *assignmentWorld) transportLastPath() (string, error) {
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

// requestReadsAssignments confirms the request hit the actor-scoped assignments
// endpoint. The actor id varies by scenario (per_0123 / agt_0456), so it matches the
// shape /actors/<id>/assignments rather than one fixed id.
func (w *assignmentWorld) requestReadsAssignments() error {
	path, err := w.transportLastPath()
	if err != nil {
		return err
	}
	if !strings.Contains(path, "/actors/") || !strings.HasSuffix(path, "/assignments") {
		return fmt.Errorf("the request should read the actor's assignments endpoint, got path %q", path)
	}
	return nil
}

func (w *assignmentWorld) eachAssignmentNamesFilledRole() error {
	// Each row leads with the filled role id and names the role — the answer to
	// "which roles does this actor fill?".
	for _, want := range []string{"role_a", "Marketing Lead", "role_b", "General Company Circle"} {
		if !strings.Contains(w.stdout, want) {
			return fmt.Errorf("each assignment should name its filled role; missing %q:\n%s", want, w.stdout)
		}
	}
	return nil
}

func (w *assignmentWorld) agentAssignmentsPrinted() error {
	if !strings.Contains(w.stdout, "Marketing Lead") || !strings.Contains(w.stdout, "role_a") {
		return fmt.Errorf("the agent's assignments should print:\n%s", w.stdout)
	}
	return nil
}

func (w *assignmentWorld) focusAndElectionPrinted() error {
	for _, want := range []string{"Keep the lights on", "2026-12-31"} {
		if !strings.Contains(w.stdout, want) {
			return fmt.Errorf("the assignment's focus and election expiry should print; missing %q:\n%s", want, w.stdout)
		}
	}
	return nil
}

func (w *assignmentWorld) exitWithCode(code int) error {
	if w.exitCode != code {
		return fmt.Errorf("exit code = %d, want %d (outcome %v)\nstderr: %s", w.exitCode, code, w.outcome, w.stderr)
	}
	return nil
}

func (w *assignmentWorld) exitNonZeroAPIError() error {
	if w.outcome != APIError || w.exitCode != 3 {
		return fmt.Errorf("outcome=%v exit=%d, want APIError/3\nstderr: %s", w.outcome, w.exitCode, w.stderr)
	}
	return nil
}

func (w *assignmentWorld) exitNonZero() error {
	if w.exitCode == 0 {
		return fmt.Errorf("exit code = 0, want a non-zero code (outcome %v)\nstderr: %s", w.outcome, w.stderr)
	}
	return nil
}

func (w *assignmentWorld) stderrNamesHTTPStatus() error {
	if !strings.Contains(w.stderr, "404") {
		return fmt.Errorf("stderr should report the read failed and name the HTTP status (404):\n%s", w.stderr)
	}
	return nil
}

func (w *assignmentWorld) stderrReportsAndPointsTo(report, pointer string) error {
	if !strings.Contains(strings.ToLower(w.stderr), strings.ToLower(report)) {
		return fmt.Errorf("stderr should report %q:\n%s", report, w.stderr)
	}
	if !strings.Contains(w.stderr, pointer) {
		return fmt.Errorf("stderr should point to %q:\n%s", pointer, w.stderr)
	}
	return nil
}

func (w *assignmentWorld) noAssignmentDataPrinted() error {
	// A not-authenticated failure prints no assignment rows (the human projection
	// leads every row with a role_ id and names the role).
	if strings.Contains(w.stdout, "role_a") || strings.Contains(w.stdout, "Marketing Lead") {
		return fmt.Errorf("no assignment data should be printed, got stdout:\n%s", w.stdout)
	}
	return nil
}

func (w *assignmentWorld) literalPrintedToStdout(literal string) error {
	if !strings.Contains(w.stdout, literal) {
		return fmt.Errorf("stdout should contain %q:\n%s", literal, w.stdout)
	}
	return nil
}

func (w *assignmentWorld) stderrReportsUsageError() error {
	if w.outcome != UsageError || w.exitCode != 2 {
		return fmt.Errorf("outcome=%v exit=%d, want UsageError/2\nstderr: %s", w.outcome, w.exitCode, w.stderr)
	}
	if strings.TrimSpace(w.stderr) == "" {
		return errors.New("a usage error should be reported on stderr")
	}
	return nil
}

func (w *assignmentWorld) noRequestSent() error {
	if w.transportCalls() != 0 {
		return fmt.Errorf("no API request should be sent, but the transport was called %d times", w.transportCalls())
	}
	return nil
}

func (w *assignmentWorld) onlyFirstPagePrinted() error {
	if !strings.Contains(w.stdout, "First Page Role") {
		return fmt.Errorf("the first page should print:\n%s", w.stdout)
	}
	if strings.Contains(w.stdout, "Second Page Role") {
		return fmt.Errorf("--first-page must not print later pages:\n%s", w.stdout)
	}
	if w.transportCalls() != 1 {
		return fmt.Errorf("--first-page must not walk; want 1 call, got %d", w.transportCalls())
	}
	return nil
}

func (w *assignmentWorld) stderrNotesMoreExist() error {
	if !strings.Contains(w.stderr, "more assignments exist") {
		return fmt.Errorf("stderr should note more assignments exist:\n%s", w.stderr)
	}
	return nil
}

func (w *assignmentWorld) partialAssignmentsPrinted() error {
	if !strings.Contains(w.stdout, "Gathered Role") {
		return fmt.Errorf("the partial set gathered so far should print:\n%s", w.stdout)
	}
	return nil
}

func (w *assignmentWorld) stderrNotesIncomplete() error {
	if !strings.Contains(w.stderr, "incomplete") {
		return fmt.Errorf("stderr should note the result is incomplete and name the cause:\n%s", w.stderr)
	}
	return nil
}
