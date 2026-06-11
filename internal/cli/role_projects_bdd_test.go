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

// TestRoleProjectsFeatures runs the executable acceptance for Role Projects (038):
// the `projects <role-id>` list and `project <proj-id>` single read, driven through
// the shared projectsSeam over a fake base transport so every scenario runs offline
// (no real network, no real ~/.glassfrogrc). Its Paths name ONLY this spec's feature
// file — never the features/ directory — so the suite reports its own independent
// scenario count and un-@wip-ping these scenarios cannot disturb another suite
// (LEARNINGS: a suite points at its own feature file). The 3 @validation scenarios
// stay @wip (held for the validate skill) and are skipped by the ~@wip filter.
func TestRoleProjectsFeatures(t *testing.T) {
	suite := godog.TestSuite{
		ScenarioInitializer: initializeRoleProjectsScenario,
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"../../features/governance-reads/role-projects.feature"},
			Tags:     "~@wip",
			TestingT: t,
		},
	}
	if suite.Run() != 0 {
		t.Fatal("non-zero status: role-projects feature scenarios failed")
	}
}

// roleProjectsWorld is the per-scenario state: the connection context and fake
// transport assembled from the Given steps, plus the captured
// outcome/exit-code/streams of the When run. Everything is injected — no step
// touches the real network, env, or home.
type roleProjectsWorld struct {
	ctx       apiclient.ConnectionContext
	transport http.RoundTripper
	secret    string

	outcome  Outcome
	exitCode int
	stdout   string
	stderr   string
}

func initializeRoleProjectsScenario(sc *godog.ScenarioContext) {
	w := &roleProjectsWorld{}
	sc.Before(func(ctx context.Context, _ *godog.Scenario) (context.Context, error) {
		*w = roleProjectsWorld{
			// A two-project single-page body is the default; the per-scenario Given
			// steps override the transport/context as needed.
			transport: &cannedTransport{status: 200, body: projectsPageComplete},
			secret:    meSecretToken,
		}
		w.ctx = validMeContext()
		return ctx, nil
	})

	// --- Givens ---
	sc.Step(`^a complete connection context with a stored token$`, w.completeContext)
	sc.Step(`^the role "([^"]*)" owns several projects$`, w.roleOwnsSeveralProjects)
	sc.Step(`^the role "([^"]*)" owns no projects$`, w.roleOwnsNoProjects)
	sc.Step(`^the role "([^"]*)" owns projects in several statuses$`, w.roleOwnsProjectsInStatuses)
	sc.Step(`^the role "([^"]*)" has projects spanning more than one page$`, w.roleHasMultiPageProjects)
	sc.Step(`^the project list walk fails after retrieving the first page$`, w.walkFailsMidway)
	sc.Step(`^no usable token is available to the CLI$`, w.noToken)
	sc.Step(`^a project "([^"]*)" exists$`, w.projectExists)
	sc.Step(`^no project "([^"]*)" exists$`, w.projectNotFound)

	// --- Whens --- (both "an agent" and "a practitioner" drive the same run) ---
	sc.Step(`^(?:an agent|a practitioner) runs "glassfrog (.+)"$`, w.runCommand)

	// --- Thens ---
	sc.Step(`^the request will read the role's projects endpoint$`, w.requestHitProjectsEndpoint)
	sc.Step(`^each project will be printed as a projection$`, w.eachProjectPrinted)
	sc.Step(`^the command will exit with code (\d+)$`, w.exitWithCode)
	sc.Step(`^the command will exit with a non-zero API-error code$`, w.exitNonZeroAPIError)
	sc.Step(`^the command will exit with a non-zero code$`, w.exitNonZero)
	sc.Step(`^"([^"]*)" will be printed to stdout$`, w.literalPrintedToStdout)
	sc.Step(`^stderr will report "([^"]*)" and point to "([^"]*)"$`, w.stderrReportsAndPointsTo)
	sc.Step(`^no project data will be printed$`, w.noProjectDataPrinted)
	sc.Step(`^the project's status, description, and owning role will be printed$`, w.statusDescriptionRolePrinted)
	sc.Step(`^stderr will report that the read failed and name the HTTP status$`, w.stderrNamesHTTPStatus)
	sc.Step(`^stderr will report a usage error$`, w.stderrReportsUsageError)
	sc.Step(`^no API request will be sent$`, w.noRequestSent)
	sc.Step(`^the request will carry the "([^"]*)" parameter set to "([^"]*)"$`, w.requestCarriesParamSetTo)
	sc.Step(`^only the current projects will be printed$`, w.onlyCurrentProjectsPrinted)
	sc.Step(`^stderr will report the unsupported value and list the supported set$`, w.stderrReportsUnsupportedStatus)
	sc.Step(`^only the first page of projects will be printed$`, w.onlyFirstPagePrinted)
	sc.Step(`^stderr will note that more projects exist$`, w.stderrNotesMoreExist)
	sc.Step(`^the projects retrieved so far will be printed$`, w.partialProjectsPrinted)
	sc.Step(`^stderr will note the result is incomplete and name the cause$`, w.stderrNotesIncomplete)
}

// --- Given implementations ---

func (w *roleProjectsWorld) completeContext() error { w.ctx = validMeContext(); return nil }

func (w *roleProjectsWorld) roleOwnsSeveralProjects(_ string) error {
	w.transport = &cannedTransport{status: 200, body: projectsPageComplete}
	return nil
}

func (w *roleProjectsWorld) roleOwnsNoProjects(_ string) error {
	w.transport = &cannedTransport{status: 200, body: projectsPageEmpty}
	return nil
}

func (w *roleProjectsWorld) roleOwnsProjectsInStatuses(_ string) error {
	// A single current project so "only the current projects will be printed" is a
	// genuine assertion (the API does the filtering; the fake returns the filtered set).
	w.transport = &cannedTransport{status: 200, body: projectsPage("proj_1", "Ship onboarding", "")}
	return nil
}

func (w *roleProjectsWorld) roleHasMultiPageProjects(_ string) error {
	w.transport = &cannedTransport{status: 200, body: projectsPage("proj_1", "First Page Project", "c1")}
	return nil
}

func (w *roleProjectsWorld) walkFailsMidway() error {
	w.transport = &seqMeTransport{steps: []seqMeResp{
		{status: 200, body: projectsPage("proj_1", "Gathered Project", "c1")},
		{status: 500, body: `{"detail":"boom"}`},
	}}
	return nil
}

func (w *roleProjectsWorld) noToken() error {
	w.ctx = noTokenContext()
	return nil
}

func (w *roleProjectsWorld) projectExists(_ string) error {
	w.transport = &cannedTransport{status: 200, body: projectDocumentBody}
	return nil
}

func (w *roleProjectsWorld) projectNotFound(_ string) error {
	w.transport = &cannedTransport{status: 404, body: `{"detail":"Project not found"}`}
	return nil
}

// --- When implementation ---

// runCommand parses the captured invocation and dispatches it through a real root
// with BOTH the `projects` and `project` leaves attached over a fake seam, so a
// single suite drives both reads. It asserts the secret token never leaks.
func (w *roleProjectsWorld) runCommand(invocation string) error {
	root := NewRootCommand()
	seam := &fakeMeSeam{ctx: w.ctx, transport: w.transport}
	MustRegister(root, newProjectsCommand(seam))
	MustRegister(root, newProjectCommand(seam))

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
func (w *roleProjectsWorld) transportCalls() int {
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

func (w *roleProjectsWorld) requestHitProjectsEndpoint() error {
	t, ok := w.transport.(*cannedTransport)
	if !ok {
		return errors.New("this scenario's transport does not record the path")
	}
	if !strings.HasSuffix(t.lastPath, "/projects") || !strings.Contains(t.lastPath, "/roles/") {
		return fmt.Errorf("the request should target /roles/{id}/projects, got %q", t.lastPath)
	}
	return nil
}

func (w *roleProjectsWorld) eachProjectPrinted() error {
	for _, want := range []string{"proj_1  [current]  Ship onboarding", "proj_2  [scheduled]  Audit billing"} {
		if !strings.Contains(w.stdout, want) {
			return fmt.Errorf("each project should print as a projection, missing %q:\n%s", want, w.stdout)
		}
	}
	return nil
}

func (w *roleProjectsWorld) exitWithCode(code int) error {
	if w.exitCode != code {
		return fmt.Errorf("exit code = %d, want %d (outcome %v)\nstderr: %s", w.exitCode, code, w.outcome, w.stderr)
	}
	return nil
}

func (w *roleProjectsWorld) exitNonZeroAPIError() error {
	if w.outcome != APIError || w.exitCode != 3 {
		return fmt.Errorf("outcome=%v exit=%d, want APIError/3\nstderr: %s", w.outcome, w.exitCode, w.stderr)
	}
	return nil
}

func (w *roleProjectsWorld) exitNonZero() error {
	if w.exitCode == 0 {
		return fmt.Errorf("exit code = 0, want a non-zero code (outcome %v)\nstderr: %s", w.outcome, w.stderr)
	}
	return nil
}

func (w *roleProjectsWorld) literalPrintedToStdout(literal string) error {
	if !strings.Contains(w.stdout, literal) {
		return fmt.Errorf("stdout should contain %q:\n%s", literal, w.stdout)
	}
	return nil
}

func (w *roleProjectsWorld) stderrReportsAndPointsTo(report, pointer string) error {
	if !strings.Contains(strings.ToLower(w.stderr), strings.ToLower(report)) {
		return fmt.Errorf("stderr should report %q:\n%s", report, w.stderr)
	}
	if !strings.Contains(w.stderr, pointer) {
		return fmt.Errorf("stderr should point to %q:\n%s", pointer, w.stderr)
	}
	return nil
}

func (w *roleProjectsWorld) noProjectDataPrinted() error {
	if strings.TrimSpace(w.stdout) != "" {
		return fmt.Errorf("no project data should be printed, got stdout:\n%s", w.stdout)
	}
	return nil
}

func (w *roleProjectsWorld) statusDescriptionRolePrinted() error {
	for _, want := range []string{
		"[current]",                    // status
		"Ship the new onboarding flow", // description
		"role_0123",                    // owning role
	} {
		if !strings.Contains(w.stdout, want) {
			return fmt.Errorf("the single project should print its status, description, and owning role, missing %q:\n%s", want, w.stdout)
		}
	}
	return nil
}

func (w *roleProjectsWorld) stderrNamesHTTPStatus() error {
	if !strings.Contains(w.stderr, "404") {
		return fmt.Errorf("stderr should name the HTTP status (404):\n%s", w.stderr)
	}
	return nil
}

func (w *roleProjectsWorld) stderrReportsUsageError() error {
	if w.outcome != UsageError || w.exitCode != 2 {
		return fmt.Errorf("outcome=%v exit=%d, want UsageError/2\nstderr: %s", w.outcome, w.exitCode, w.stderr)
	}
	if strings.TrimSpace(w.stderr) == "" {
		return errors.New("a usage error should be reported on stderr")
	}
	return nil
}

func (w *roleProjectsWorld) noRequestSent() error {
	if w.transportCalls() != 0 {
		return fmt.Errorf("no API request should be sent, but the transport was called %d times", w.transportCalls())
	}
	return nil
}

func (w *roleProjectsWorld) requestCarriesParamSetTo(param, value string) error {
	t, ok := w.transport.(*cannedTransport)
	if !ok {
		return errors.New("this scenario's transport does not record the query")
	}
	if got := t.lastQuery.Get(param); got != value {
		return fmt.Errorf("the request query should carry %q=%q, got %q (query %q)", param, value, got, t.lastQuery.Encode())
	}
	return nil
}

func (w *roleProjectsWorld) onlyCurrentProjectsPrinted() error {
	if !strings.Contains(w.stdout, "[current]") || !strings.Contains(w.stdout, "Ship onboarding") {
		return fmt.Errorf("the current projects should be printed:\n%s", w.stdout)
	}
	return nil
}

func (w *roleProjectsWorld) stderrReportsUnsupportedStatus() error {
	if w.outcome != UsageError || w.exitCode != 2 {
		return fmt.Errorf("outcome=%v exit=%d, want UsageError/2\nstderr: %s", w.outcome, w.exitCode, w.stderr)
	}
	// Names the rejected value and lists at least one supported status.
	if !strings.Contains(w.stderr, "active") {
		return fmt.Errorf("stderr should name the unsupported value:\n%s", w.stderr)
	}
	if !strings.Contains(w.stderr, "current") {
		return fmt.Errorf("stderr should list the supported set:\n%s", w.stderr)
	}
	return nil
}

func (w *roleProjectsWorld) onlyFirstPagePrinted() error {
	if !strings.Contains(w.stdout, "First Page Project") {
		return fmt.Errorf("the first page should print:\n%s", w.stdout)
	}
	if w.transportCalls() != 1 {
		return fmt.Errorf("--first-page must not walk, want 1 call, got %d", w.transportCalls())
	}
	return nil
}

func (w *roleProjectsWorld) stderrNotesMoreExist() error {
	if !strings.Contains(w.stderr, "more projects exist") {
		return fmt.Errorf("stderr should note more projects exist:\n%s", w.stderr)
	}
	return nil
}

func (w *roleProjectsWorld) partialProjectsPrinted() error {
	if !strings.Contains(w.stdout, "Gathered Project") {
		return fmt.Errorf("the partial set gathered so far should print:\n%s", w.stdout)
	}
	return nil
}

func (w *roleProjectsWorld) stderrNotesIncomplete() error {
	if !strings.Contains(w.stderr, "incomplete") {
		return fmt.Errorf("stderr should note the result is incomplete and name the cause:\n%s", w.stderr)
	}
	return nil
}
