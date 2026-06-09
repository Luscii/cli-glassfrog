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
	"github.com/Luscii/cli-glassfrog/internal/auth"
	"github.com/cucumber/godog"
)

// TestRoleReadsFeatures runs the executable acceptance for Role Reads (025): the
// org-wide `roles` command (list + single read) driven through its seam over a
// fake base transport, so every scenario runs offline (no real network, no real
// ~/.glassfrogrc). Its Paths name ONLY this spec's feature file — never the
// features/ directory — so the suite reports its own independent scenario count
// and un-@wip-ping these scenarios cannot disturb another suite (LEARNINGS: a
// suite points at its own feature file). The 3 @validation scenarios stay @wip
// (held for the validate skill) and are skipped by the ~@wip filter.
func TestRoleReadsFeatures(t *testing.T) {
	suite := godog.TestSuite{
		ScenarioInitializer: initializeRoleReadsScenario,
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"../../features/governance-reads/role-reads.feature"},
			Tags:     "~@wip",
			TestingT: t,
		},
	}
	if suite.Run() != 0 {
		t.Fatal("non-zero status: role-reads feature scenarios failed")
	}
}

// roleReadsWorld is the per-scenario state: the connection context and fake
// transport assembled from the Given steps, plus the captured
// outcome/exit-code/streams of the When run. Everything is injected — no step
// touches the real network, env, or home.
type roleReadsWorld struct {
	ctx          apiclient.ConnectionContext
	newClientErr error
	transport    http.RoundTripper
	secret       string

	outcome  Outcome
	exitCode int
	stdout   string
	stderr   string
}

func initializeRoleReadsScenario(sc *godog.ScenarioContext) {
	w := &roleReadsWorld{}
	sc.Before(func(ctx context.Context, _ *godog.Scenario) (context.Context, error) {
		*w = roleReadsWorld{
			// A multi-role single-page body is the default; the per-scenario Given
			// steps override the transport/context as needed.
			transport: &cannedTransport{status: 200, body: orgRolesPageComplete},
			secret:    meSecretToken,
		}
		w.ctx = validMeContext()
		return ctx, nil
	})

	// --- Givens ---
	sc.Step(`^a complete connection context with a stored token$`, w.completeContext)
	sc.Step(`^the API would return several roles in the organization$`, w.apiReturnsSeveralRoles)
	sc.Step(`^a role "([^"]*)" exists in the organization$`, w.apiReturnsRoleDetail)
	sc.Step(`^no usable token is available to the CLI$`, w.contextNoToken)
	sc.Step(`^the API is unreachable at the wire$`, w.apiUnreachable)
	sc.Step(`^no role "([^"]*)" exists in the organization$`, w.apiReturnsNotFound)
	sc.Step(`^the API would return no roles$`, w.apiReturnsNoRoles)
	sc.Step(`^the parent role "([^"]*)" contains several roles$`, w.apiReturnsParentRoles)
	sc.Step(`^the organization's roles span three pages of API responses$`, w.apiReturnsThreePages)
	sc.Step(`^the organization's roles span more than one page$`, w.apiReturnsFirstOfMany)
	sc.Step(`^the role list walk fails after retrieving the first page$`, w.apiFailsMidWalk)

	// --- Whens --- (both "an agent" and "the practitioner" drive the same run) ---
	sc.Step(`^(?:an agent|the practitioner) runs "glassfrog (.+)"$`, w.runCommand)

	// --- Thens ---
	sc.Step(`^each role will be printed as a projection$`, w.eachRolePrinted)
	sc.Step(`^the command will exit with code (\d+)$`, w.exitWithCode)
	sc.Step(`^the command will exit with a non-zero API-error code$`, w.exitNonZeroAPIError)
	sc.Step(`^the command will exit with a non-zero code$`, w.exitNonZero)
	sc.Step(`^the role's name, purpose, accountabilities, domains, and fillers will be printed$`, w.singleRolePrinted)
	sc.Step(`^stderr will report "([^"]*)" and point to "([^"]*)"$`, w.stderrReportsAndPointsTo)
	sc.Step(`^no role data will be printed$`, w.noRoleDataPrinted)
	sc.Step(`^stderr will name the transport failure$`, w.transportFailureNamed)
	sc.Step(`^stderr will report that the read failed and name the HTTP status$`, w.stderrNamesHTTPStatus)
	sc.Step(`^"([^"]*)" will be printed to stdout$`, w.literalPrintedToStdout)
	sc.Step(`^the request will carry the "([^"]*)" filter$`, w.requestCarriesFilter)
	sc.Step(`^only roles under that parent will be printed$`, w.rolesPrinted)
	sc.Step(`^stderr will report a usage error$`, w.stderrReportsUsageError)
	sc.Step(`^no API request will be sent$`, w.noRequestSent)
	sc.Step(`^the request will carry "([^"]*)"$`, w.requestCarriesRawParam)
	sc.Step(`^the policies and subroles will be printed inline within the role$`, w.policiesAndSubrolesInline)
	sc.Step(`^stderr will name the unsupported value and the supported set$`, w.stderrNamesUnsupportedInclude)
	sc.Step(`^the command will walk every page to completion$`, w.walkedEveryPage)
	sc.Step(`^all roles across the pages will be printed$`, w.allPagesPrinted)
	sc.Step(`^only the first page of roles will be printed$`, w.onlyFirstPagePrinted)
	sc.Step(`^stderr will note that more roles exist$`, w.stderrNotesMoreExist)
	sc.Step(`^the roles retrieved so far will be printed$`, w.partialRolesPrinted)
	sc.Step(`^stderr will note the result is incomplete and name the cause$`, w.stderrNotesIncomplete)
}

// --- Given implementations ---

func (w *roleReadsWorld) completeContext() error { w.ctx = validMeContext(); return nil }

func (w *roleReadsWorld) apiReturnsSeveralRoles() error {
	w.transport = &cannedTransport{status: 200, body: orgRolesPageComplete}
	return nil
}

func (w *roleReadsWorld) apiReturnsRoleDetail(_ string) error {
	w.transport = &cannedTransport{status: 200, body: roleDetailBody}
	return nil
}

func (w *roleReadsWorld) contextNoToken() error {
	w.ctx = apiclient.ConnectionContext{
		BaseURL: apiclient.BaseURL{Value: "https://example.test/api/v5", Source: apiclient.SourceFlag},
		Cred:    auth.Resolution{Source: auth.SourceNone},
	}
	return nil
}

func (w *roleReadsWorld) apiUnreachable() error {
	w.transport = &cannedTransport{netErr: errors.New("dial tcp: connection refused")}
	return nil
}

func (w *roleReadsWorld) apiReturnsNotFound(_ string) error {
	w.transport = &cannedTransport{status: 404, body: `{"detail":"Role not found"}`}
	return nil
}

func (w *roleReadsWorld) apiReturnsNoRoles() error {
	w.transport = &cannedTransport{status: 200, body: orgRolesPageEmpty}
	return nil
}

func (w *roleReadsWorld) apiReturnsParentRoles(_ string) error {
	w.transport = &cannedTransport{status: 200, body: orgRolesPageComplete}
	return nil
}

func (w *roleReadsWorld) apiReturnsThreePages() error {
	w.transport = &seqMeTransport{steps: []seqMeResp{
		{status: 200, body: orgRolesPage("role_00000000000000000000000000000001", "Page One Role", "c1")},
		{status: 200, body: orgRolesPage("role_00000000000000000000000000000002", "Page Two Role", "c2")},
		{status: 200, body: orgRolesPage("role_00000000000000000000000000000003", "Page Three Role", "")},
	}}
	return nil
}

func (w *roleReadsWorld) apiReturnsFirstOfMany() error {
	w.transport = &cannedTransport{status: 200, body: orgRolesPage("role_00000000000000000000000000000001", "First Page Role", "c1")}
	return nil
}

func (w *roleReadsWorld) apiFailsMidWalk() error {
	w.transport = &seqMeTransport{steps: []seqMeResp{
		{status: 200, body: orgRolesPage("role_00000000000000000000000000000001", "Gathered Role", "c1")},
		{status: 500, body: `{"detail":"boom"}`},
	}}
	return nil
}

// --- When implementation ---

// runCommand parses the captured "roles …" invocation and dispatches it through a
// real root with the `roles` leaf attached over a fake seam. It asserts the
// secret token never leaks into any produced output.
func (w *roleReadsWorld) runCommand(invocation string) error {
	root := NewRootCommand()
	seam := &fakeMeSeam{ctx: w.ctx, newClientErr: w.newClientErr, transport: w.transport}
	MustRegister(root, newRolesCommand(seam))

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

// transportCalls reads the request count off whichever fake transport the
// scenario installed.
func (w *roleReadsWorld) transportCalls() int {
	switch t := w.transport.(type) {
	case *cannedTransport:
		return t.calls
	case *seqMeTransport:
		return t.calls
	default:
		return -1
	}
}

// lastQuery reads the recorded query off the cannedTransport (the only fake that
// records it; the filter/include scenarios all use it).
func (w *roleReadsWorld) lastQuery() (string, error) {
	t, ok := w.transport.(*cannedTransport)
	if !ok {
		return "", fmt.Errorf("this scenario's transport does not record the query")
	}
	return t.lastQuery.Encode(), nil
}

// --- Then implementations ---

func (w *roleReadsWorld) eachRolePrinted() error {
	for _, want := range []string{"Marketing Lead (role_", "Anchor Circle", "Purpose:"} {
		if !strings.Contains(w.stdout, want) {
			return fmt.Errorf("each role should print as a projection, missing %q:\n%s", want, w.stdout)
		}
	}
	return nil
}

func (w *roleReadsWorld) exitWithCode(code int) error {
	if w.exitCode != code {
		return fmt.Errorf("exit code = %d, want %d (outcome %v)\nstderr: %s", w.exitCode, code, w.outcome, w.stderr)
	}
	return nil
}

func (w *roleReadsWorld) exitNonZeroAPIError() error {
	if w.outcome != APIError || w.exitCode != 3 {
		return fmt.Errorf("outcome=%v exit=%d, want APIError/3\nstderr: %s", w.outcome, w.exitCode, w.stderr)
	}
	return nil
}

func (w *roleReadsWorld) exitNonZero() error {
	if w.exitCode == 0 {
		return fmt.Errorf("exit code = 0, want a non-zero code (outcome %v)\nstderr: %s", w.outcome, w.stderr)
	}
	return nil
}

func (w *roleReadsWorld) singleRolePrinted() error {
	for _, want := range []string{
		"Marketing Lead (role_0123456789abcdef0123456789abcdef)",
		"Purpose: A market that knows us",
		"Accountabilities:", "Defining the campaign",
		"Domains:", "The marketing budget",
		"Fillers:", "Alice Smith (per_x)",
	} {
		if !strings.Contains(w.stdout, want) {
			return fmt.Errorf("the single role should show %q:\n%s", want, w.stdout)
		}
	}
	return nil
}

func (w *roleReadsWorld) stderrReportsAndPointsTo(report, pointer string) error {
	if !strings.Contains(strings.ToLower(w.stderr), strings.ToLower(report)) {
		return fmt.Errorf("stderr should report %q:\n%s", report, w.stderr)
	}
	if !strings.Contains(w.stderr, pointer) {
		return fmt.Errorf("stderr should point to %q:\n%s", pointer, w.stderr)
	}
	return nil
}

func (w *roleReadsWorld) noRoleDataPrinted() error {
	if strings.TrimSpace(w.stdout) != "" {
		return fmt.Errorf("no role data should be printed, got stdout:\n%s", w.stdout)
	}
	return nil
}

func (w *roleReadsWorld) transportFailureNamed() error {
	if w.outcome != NetworkUnavailable {
		return fmt.Errorf("outcome = %v, want NetworkUnavailable", w.outcome)
	}
	if strings.TrimSpace(w.stderr) == "" {
		return errors.New("a transport failure should be named on stderr")
	}
	return nil
}

func (w *roleReadsWorld) stderrNamesHTTPStatus() error {
	if !strings.Contains(w.stderr, "404") {
		return fmt.Errorf("stderr should name the HTTP status (404):\n%s", w.stderr)
	}
	return nil
}

func (w *roleReadsWorld) literalPrintedToStdout(literal string) error {
	if !strings.Contains(w.stdout, literal) {
		return fmt.Errorf("stdout should contain %q:\n%s", literal, w.stdout)
	}
	return nil
}

func (w *roleReadsWorld) requestCarriesFilter(param string) error {
	q, err := w.lastQuery()
	if err != nil {
		return err
	}
	if !strings.Contains(q, param+"=") {
		return fmt.Errorf("the request query %q should carry the %q filter", q, param)
	}
	return nil
}

func (w *roleReadsWorld) rolesPrinted() error {
	if !strings.Contains(w.stdout, "role_") {
		return fmt.Errorf("roles should be printed to stdout:\n%s", w.stdout)
	}
	return nil
}

func (w *roleReadsWorld) stderrReportsUsageError() error {
	if w.outcome != UsageError || w.exitCode != 2 {
		return fmt.Errorf("outcome=%v exit=%d, want UsageError/2\nstderr: %s", w.outcome, w.exitCode, w.stderr)
	}
	if strings.TrimSpace(w.stderr) == "" {
		return errors.New("a usage error should be reported on stderr")
	}
	return nil
}

func (w *roleReadsWorld) noRequestSent() error {
	if w.transportCalls() != 0 {
		return fmt.Errorf("no API request should be sent, but the transport was called %d times", w.transportCalls())
	}
	return nil
}

func (w *roleReadsWorld) requestCarriesRawParam(param string) error {
	// param is like `include=policies,subroles`; the recorded query encodes the
	// comma, so compare on the decoded key/value via lastQuery.Get.
	t, ok := w.transport.(*cannedTransport)
	if !ok {
		return errors.New("this scenario's transport does not record the query")
	}
	kv := strings.SplitN(param, "=", 2)
	if len(kv) != 2 {
		return fmt.Errorf("malformed expected param %q", param)
	}
	if got := t.lastQuery.Get(kv[0]); got != kv[1] {
		return fmt.Errorf("request %s = %q, want %q", kv[0], got, kv[1])
	}
	return nil
}

func (w *roleReadsWorld) policiesAndSubrolesInline() error {
	for _, want := range []string{"Policies:", "All PRs require two approvals", "Subroles:", "Press Officer"} {
		if !strings.Contains(w.stdout, want) {
			return fmt.Errorf("the requested includes should print inline; missing %q:\n%s", want, w.stdout)
		}
	}
	return nil
}

func (w *roleReadsWorld) stderrNamesUnsupportedInclude() error {
	if !strings.Contains(w.stderr, "nonsense") {
		return fmt.Errorf("stderr should name the unsupported value:\n%s", w.stderr)
	}
	if !strings.Contains(w.stderr, "policies") {
		return fmt.Errorf("stderr should name the supported set:\n%s", w.stderr)
	}
	return nil
}

func (w *roleReadsWorld) walkedEveryPage() error {
	if w.transportCalls() != 3 {
		return fmt.Errorf("the walk should issue three page requests, got %d", w.transportCalls())
	}
	return nil
}

func (w *roleReadsWorld) allPagesPrinted() error {
	for _, want := range []string{"Page One Role", "Page Two Role", "Page Three Role"} {
		if !strings.Contains(w.stdout, want) {
			return fmt.Errorf("every page's roles should print, missing %q:\n%s", want, w.stdout)
		}
	}
	return nil
}

func (w *roleReadsWorld) onlyFirstPagePrinted() error {
	if !strings.Contains(w.stdout, "First Page Role") {
		return fmt.Errorf("the first page should print:\n%s", w.stdout)
	}
	if w.transportCalls() != 1 {
		return fmt.Errorf("--first-page must not walk, want 1 call, got %d", w.transportCalls())
	}
	return nil
}

func (w *roleReadsWorld) stderrNotesMoreExist() error {
	if !strings.Contains(w.stderr, "more roles exist") {
		return fmt.Errorf("stderr should note more roles exist:\n%s", w.stderr)
	}
	return nil
}

func (w *roleReadsWorld) partialRolesPrinted() error {
	if !strings.Contains(w.stdout, "Gathered Role") {
		return fmt.Errorf("the partial set gathered so far should print:\n%s", w.stdout)
	}
	return nil
}

func (w *roleReadsWorld) stderrNotesIncomplete() error {
	if !strings.Contains(w.stderr, "incomplete") {
		return fmt.Errorf("stderr should note the result is incomplete and name the cause:\n%s", w.stderr)
	}
	return nil
}
