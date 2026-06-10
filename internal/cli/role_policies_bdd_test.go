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

// TestRolePoliciesFeatures runs the executable acceptance for Role Policies (034):
// the `policies <role-id>` list and `policy <pol-id>` single read, driven through
// the shared policiesSeam over a fake base transport so every scenario runs
// offline (no real network, no real ~/.glassfrogrc). Its Paths name ONLY this
// spec's feature file — never the features/ directory — so the suite reports its
// own independent scenario count and un-@wip-ping these scenarios cannot disturb
// another suite (LEARNINGS: a suite points at its own feature file). The 2
// @validation scenarios stay @wip (held for the validate skill) and are skipped by
// the ~@wip filter.
func TestRolePoliciesFeatures(t *testing.T) {
	suite := godog.TestSuite{
		ScenarioInitializer: initializeRolePoliciesScenario,
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"../../features/governance-reads/role-policies.feature"},
			Tags:     "~@wip",
			TestingT: t,
		},
	}
	if suite.Run() != 0 {
		t.Fatal("non-zero status: role-policies feature scenarios failed")
	}
}

// rolePoliciesWorld is the per-scenario state: the connection context and fake
// transport assembled from the Given steps, plus the captured
// outcome/exit-code/streams of the When run. Everything is injected — no step
// touches the real network, env, or home.
type rolePoliciesWorld struct {
	ctx       apiclient.ConnectionContext
	transport http.RoundTripper
	secret    string

	outcome  Outcome
	exitCode int
	stdout   string
	stderr   string
}

func initializeRolePoliciesScenario(sc *godog.ScenarioContext) {
	w := &rolePoliciesWorld{}
	sc.Before(func(ctx context.Context, _ *godog.Scenario) (context.Context, error) {
		*w = rolePoliciesWorld{
			// A two-policy single-page body is the default; the per-scenario Given
			// steps override the transport/context as needed.
			transport: &cannedTransport{status: 200, body: policiesPageComplete},
			secret:    meSecretToken,
		}
		w.ctx = validMeContext()
		return ctx, nil
	})

	// --- Givens ---
	sc.Step(`^a complete connection context with a stored token$`, w.completeContext)
	sc.Step(`^the role "([^"]*)" is governed by several policies$`, w.roleHasSeveralPolicies)
	sc.Step(`^the role "([^"]*)" is governed by no policies$`, w.roleHasNoPolicies)
	sc.Step(`^the role "([^"]*)" is governed by a policy titled "([^"]*)"$`, w.roleHasPolicyTitled)
	sc.Step(`^the role "([^"]*)" has policies spanning more than one page$`, w.roleHasMultiPagePolicies)
	sc.Step(`^the policy list walk fails after retrieving the first page$`, w.walkFailsMidway)
	sc.Step(`^no usable token is available to the CLI$`, w.noToken)
	sc.Step(`^a policy "([^"]*)" exists$`, w.policyExists)
	sc.Step(`^no policy "([^"]*)" exists$`, w.policyNotFound)

	// --- Whens --- (both "an agent" and "a practitioner" drive the same run) ---
	sc.Step(`^(?:an agent|a practitioner) runs "glassfrog (.+)"$`, w.runCommand)

	// --- Thens ---
	sc.Step(`^the request will read the role's policies endpoint$`, w.requestHitPoliciesEndpoint)
	sc.Step(`^each policy will be printed as a projection$`, w.eachPolicyPrinted)
	sc.Step(`^the command will exit with code (\d+)$`, w.exitWithCode)
	sc.Step(`^the command will exit with a non-zero API-error code$`, w.exitNonZeroAPIError)
	sc.Step(`^the command will exit with a non-zero code$`, w.exitNonZero)
	sc.Step(`^"([^"]*)" will be printed to stdout$`, w.literalPrintedToStdout)
	sc.Step(`^stderr will report "([^"]*)" and point to "([^"]*)"$`, w.stderrReportsAndPointsTo)
	sc.Step(`^no policy data will be printed$`, w.noPolicyDataPrinted)
	sc.Step(`^the policy's title and full body will be printed$`, w.titleAndFullBodyPrinted)
	sc.Step(`^stderr will report that the read failed and name the HTTP status$`, w.stderrNamesHTTPStatus)
	sc.Step(`^stderr will report a usage error$`, w.stderrReportsUsageError)
	sc.Step(`^no API request will be sent$`, w.noRequestSent)
	sc.Step(`^the request will carry the "([^"]*)" search parameter$`, w.requestCarriesSearchParam)
	sc.Step(`^only the matching policies will be printed$`, w.matchingPoliciesPrinted)
	sc.Step(`^only the first page of policies will be printed$`, w.onlyFirstPagePrinted)
	sc.Step(`^stderr will note that more policies exist$`, w.stderrNotesMoreExist)
	sc.Step(`^the policies retrieved so far will be printed$`, w.partialPoliciesPrinted)
	sc.Step(`^stderr will note the result is incomplete and name the cause$`, w.stderrNotesIncomplete)
}

// --- Given implementations ---

func (w *rolePoliciesWorld) completeContext() error { w.ctx = validMeContext(); return nil }

func (w *rolePoliciesWorld) roleHasSeveralPolicies(_ string) error {
	w.transport = &cannedTransport{status: 200, body: policiesPageComplete}
	return nil
}

func (w *rolePoliciesWorld) roleHasNoPolicies(_ string) error {
	w.transport = &cannedTransport{status: 200, body: policiesPageEmpty}
	return nil
}

func (w *rolePoliciesWorld) roleHasPolicyTitled(_, title string) error {
	w.transport = &cannedTransport{status: 200, body: policiesPage("pol_1", title, "")}
	return nil
}

func (w *rolePoliciesWorld) roleHasMultiPagePolicies(_ string) error {
	w.transport = &cannedTransport{status: 200, body: policiesPage("pol_1", "First Page Policy", "c1")}
	return nil
}

func (w *rolePoliciesWorld) walkFailsMidway() error {
	w.transport = &seqMeTransport{steps: []seqMeResp{
		{status: 200, body: policiesPage("pol_1", "Gathered Policy", "c1")},
		{status: 500, body: `{"detail":"boom"}`},
	}}
	return nil
}

func (w *rolePoliciesWorld) noToken() error {
	w.ctx = noTokenContext()
	return nil
}

func (w *rolePoliciesWorld) policyExists(_ string) error {
	w.transport = &cannedTransport{status: 200, body: policyDocumentBody}
	return nil
}

func (w *rolePoliciesWorld) policyNotFound(_ string) error {
	w.transport = &cannedTransport{status: 404, body: `{"detail":"Policy not found"}`}
	return nil
}

// --- When implementation ---

// runCommand parses the captured invocation and dispatches it through a real root
// with BOTH the `policies` and `policy` leaves attached over a fake seam, so a
// single suite drives both reads. It asserts the secret token never leaks.
func (w *rolePoliciesWorld) runCommand(invocation string) error {
	root := NewRootCommand()
	seam := &fakeMeSeam{ctx: w.ctx, transport: w.transport}
	MustRegister(root, newPoliciesCommand(seam))
	MustRegister(root, newPolicyCommand(seam))

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
func (w *rolePoliciesWorld) transportCalls() int {
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

func (w *rolePoliciesWorld) requestHitPoliciesEndpoint() error {
	t, ok := w.transport.(*cannedTransport)
	if !ok {
		return errors.New("this scenario's transport does not record the path")
	}
	if !strings.HasSuffix(t.lastPath, "/policies") || !strings.Contains(t.lastPath, "/roles/") {
		return fmt.Errorf("the request should target /roles/{id}/policies, got %q", t.lastPath)
	}
	return nil
}

func (w *rolePoliciesWorld) eachPolicyPrinted() error {
	for _, want := range []string{"All PRs require two approvals (pol_1)", "Spending limit (pol_2)"} {
		if !strings.Contains(w.stdout, want) {
			return fmt.Errorf("each policy should print as a projection, missing %q:\n%s", want, w.stdout)
		}
	}
	return nil
}

func (w *rolePoliciesWorld) exitWithCode(code int) error {
	if w.exitCode != code {
		return fmt.Errorf("exit code = %d, want %d (outcome %v)\nstderr: %s", w.exitCode, code, w.outcome, w.stderr)
	}
	return nil
}

func (w *rolePoliciesWorld) exitNonZeroAPIError() error {
	if w.outcome != APIError || w.exitCode != 3 {
		return fmt.Errorf("outcome=%v exit=%d, want APIError/3\nstderr: %s", w.outcome, w.exitCode, w.stderr)
	}
	return nil
}

func (w *rolePoliciesWorld) exitNonZero() error {
	if w.exitCode == 0 {
		return fmt.Errorf("exit code = 0, want a non-zero code (outcome %v)\nstderr: %s", w.outcome, w.stderr)
	}
	return nil
}

func (w *rolePoliciesWorld) literalPrintedToStdout(literal string) error {
	if !strings.Contains(w.stdout, literal) {
		return fmt.Errorf("stdout should contain %q:\n%s", literal, w.stdout)
	}
	return nil
}

func (w *rolePoliciesWorld) stderrReportsAndPointsTo(report, pointer string) error {
	if !strings.Contains(strings.ToLower(w.stderr), strings.ToLower(report)) {
		return fmt.Errorf("stderr should report %q:\n%s", report, w.stderr)
	}
	if !strings.Contains(w.stderr, pointer) {
		return fmt.Errorf("stderr should point to %q:\n%s", pointer, w.stderr)
	}
	return nil
}

func (w *rolePoliciesWorld) noPolicyDataPrinted() error {
	if strings.TrimSpace(w.stdout) != "" {
		return fmt.Errorf("no policy data should be printed, got stdout:\n%s", w.stdout)
	}
	return nil
}

func (w *rolePoliciesWorld) titleAndFullBodyPrinted() error {
	for _, want := range []string{
		"All PRs require two approvals (pol_0123)",
		"<p>Every PR needs <strong>two</strong> approvals.</p>", // full body, verbatim
	} {
		if !strings.Contains(w.stdout, want) {
			return fmt.Errorf("the single policy should print its title and full body, missing %q:\n%s", want, w.stdout)
		}
	}
	return nil
}

func (w *rolePoliciesWorld) stderrNamesHTTPStatus() error {
	if !strings.Contains(w.stderr, "404") {
		return fmt.Errorf("stderr should name the HTTP status (404):\n%s", w.stderr)
	}
	return nil
}

func (w *rolePoliciesWorld) stderrReportsUsageError() error {
	if w.outcome != UsageError || w.exitCode != 2 {
		return fmt.Errorf("outcome=%v exit=%d, want UsageError/2\nstderr: %s", w.outcome, w.exitCode, w.stderr)
	}
	if strings.TrimSpace(w.stderr) == "" {
		return errors.New("a usage error should be reported on stderr")
	}
	return nil
}

func (w *rolePoliciesWorld) noRequestSent() error {
	if w.transportCalls() != 0 {
		return fmt.Errorf("no API request should be sent, but the transport was called %d times", w.transportCalls())
	}
	return nil
}

func (w *rolePoliciesWorld) requestCarriesSearchParam(param string) error {
	t, ok := w.transport.(*cannedTransport)
	if !ok {
		return errors.New("this scenario's transport does not record the query")
	}
	if t.lastQuery.Get(param) == "" {
		return fmt.Errorf("the request query %q should carry the %q search parameter", t.lastQuery.Encode(), param)
	}
	return nil
}

func (w *rolePoliciesWorld) matchingPoliciesPrinted() error {
	if !strings.Contains(w.stdout, "All PRs require two approvals") {
		return fmt.Errorf("the matching policy should be printed:\n%s", w.stdout)
	}
	return nil
}

func (w *rolePoliciesWorld) onlyFirstPagePrinted() error {
	if !strings.Contains(w.stdout, "First Page Policy") {
		return fmt.Errorf("the first page should print:\n%s", w.stdout)
	}
	if w.transportCalls() != 1 {
		return fmt.Errorf("--first-page must not walk, want 1 call, got %d", w.transportCalls())
	}
	return nil
}

func (w *rolePoliciesWorld) stderrNotesMoreExist() error {
	if !strings.Contains(w.stderr, "more policies exist") {
		return fmt.Errorf("stderr should note more policies exist:\n%s", w.stderr)
	}
	return nil
}

func (w *rolePoliciesWorld) partialPoliciesPrinted() error {
	if !strings.Contains(w.stdout, "Gathered Policy") {
		return fmt.Errorf("the partial set gathered so far should print:\n%s", w.stdout)
	}
	return nil
}

func (w *rolePoliciesWorld) stderrNotesIncomplete() error {
	if !strings.Contains(w.stderr, "incomplete") {
		return fmt.Errorf("stderr should note the result is incomplete and name the cause:\n%s", w.stderr)
	}
	return nil
}
