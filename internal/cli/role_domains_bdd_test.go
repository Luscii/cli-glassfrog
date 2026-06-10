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

// TestRoleDomainsFeatures runs the executable acceptance for Role Domains (033):
// the `domains <role-id>` list and `domain <dom-id>` single read driven through
// their seams over a fake base transport, so every scenario runs offline (no real
// network, no real ~/.glassfrogrc). Its Paths name ONLY this spec's feature file
// — never the features/ directory — so the suite reports its own independent
// scenario count and un-@wip-ping these scenarios cannot disturb another suite
// (LEARNINGS: a suite points at its own feature file). The 3 @validation
// scenarios stay @wip (held for the validate skill) and are skipped by the ~@wip
// filter.
func TestRoleDomainsFeatures(t *testing.T) {
	suite := godog.TestSuite{
		ScenarioInitializer: initializeRoleDomainsScenario,
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"../../features/governance-reads/role-domains.feature"},
			Tags:     "~@wip",
			TestingT: t,
		},
	}
	if suite.Run() != 0 {
		t.Fatal("non-zero status: role-domains feature scenarios failed")
	}
}

// roleDomainsWorld is the per-scenario state: the connection context and fake
// transport assembled from the Given steps, plus the captured
// outcome/exit-code/streams of the When run. Everything is injected — no step
// touches the real network, env, or home.
type roleDomainsWorld struct {
	ctx          apiclient.ConnectionContext
	newClientErr error
	transport    http.RoundTripper
	secret       string

	outcome  Outcome
	exitCode int
	stdout   string
	stderr   string
}

func initializeRoleDomainsScenario(sc *godog.ScenarioContext) {
	w := &roleDomainsWorld{}
	sc.Before(func(ctx context.Context, _ *godog.Scenario) (context.Context, error) {
		*w = roleDomainsWorld{
			// A single complete page of domains is the default; the per-scenario Given
			// steps override the transport/context as needed.
			transport: &cannedTransport{status: 200, body: domainsPageComplete},
			secret:    meSecretToken,
		}
		w.ctx = validMeContext()
		return ctx, nil
	})

	// --- Givens ---
	sc.Step(`^a complete connection context with a stored token$`, w.completeContext)
	sc.Step(`^the role "([^"]*)" controls several domains$`, w.roleControlsSeveralDomains)
	sc.Step(`^no usable token is available to the CLI$`, w.contextNoToken)
	sc.Step(`^the API is unreachable at the wire$`, w.apiUnreachable)
	sc.Step(`^the role "([^"]*)" controls no domains$`, w.roleControlsNoDomains)
	sc.Step(`^a domain "([^"]*)" exists in the organization$`, w.domainExists)
	sc.Step(`^no domain "([^"]*)" exists in the organization$`, w.domainNotFound)
	sc.Step(`^the role "([^"]*)" controls a domain matching "([^"]*)"$`, w.roleControlsMatchingDomain)
	sc.Step(`^the role "([^"]*)" controls domains spanning three pages of API responses$`, w.domainsSpanThreePages)
	sc.Step(`^the role "([^"]*)" controls domains spanning more than one page$`, w.domainsSpanMoreThanOnePage)
	sc.Step(`^the domains list walk fails after retrieving the first page$`, w.domainsWalkFailsMidway)

	// --- Whens --- (both "an agent" and "the practitioner" drive the same run) ---
	sc.Step(`^(?:an agent|the practitioner) runs "glassfrog (.+)"$`, w.runCommand)

	// --- Thens ---
	sc.Step(`^each domain will be printed as a projection$`, w.eachDomainPrinted)
	sc.Step(`^the command will exit with code (\d+)$`, w.exitWithCode)
	sc.Step(`^the command will exit with a non-zero API-error code$`, w.exitNonZeroAPIError)
	sc.Step(`^the command will exit with a non-zero code$`, w.exitNonZero)
	sc.Step(`^stderr will report "([^"]*)" and point to "([^"]*)"$`, w.stderrReportsAndPointsTo)
	sc.Step(`^no domain data will be printed$`, w.noDomainDataPrinted)
	sc.Step(`^stderr will name the transport failure$`, w.transportFailureNamed)
	sc.Step(`^"([^"]*)" will be printed to stdout$`, w.literalPrintedToStdout)
	sc.Step(`^the domain's description and controlling role will be printed$`, w.singleDomainPrinted)
	sc.Step(`^the request will carry "([^"]*)"$`, w.requestCarriesRawParam)
	sc.Step(`^the domain's policies will be printed inline within the domain$`, w.policiesInline)
	sc.Step(`^stderr will report that the read failed and name the HTTP status$`, w.stderrNamesHTTPStatus)
	sc.Step(`^stderr will name the unsupported value and the supported set$`, w.stderrNamesUnsupportedInclude)
	sc.Step(`^no API request will be sent$`, w.noRequestSent)
	sc.Step(`^stderr will report a usage error$`, w.stderrReportsUsageError)
	sc.Step(`^the request will carry the "([^"]*)" search term$`, w.requestCarriesSearchTerm)
	sc.Step(`^only the matching domains will be printed$`, w.matchingDomainsPrinted)
	sc.Step(`^the command will walk every page to completion$`, w.walkedEveryPage)
	sc.Step(`^all domains across the pages will be printed$`, w.allPagesPrinted)
	sc.Step(`^only the first page of domains will be printed$`, w.onlyFirstPagePrinted)
	sc.Step(`^stderr will note that more domains exist$`, w.stderrNotesMoreExist)
	sc.Step(`^the domains retrieved so far will be printed$`, w.partialDomainsPrinted)
	sc.Step(`^stderr will note the result is incomplete and name the cause$`, w.stderrNotesIncomplete)
}

// --- Given implementations ---

func (w *roleDomainsWorld) completeContext() error { w.ctx = validMeContext(); return nil }

func (w *roleDomainsWorld) roleControlsSeveralDomains(_ string) error {
	w.transport = &cannedTransport{status: 200, body: domainsPageComplete}
	return nil
}

func (w *roleDomainsWorld) contextNoToken() error {
	w.ctx = apiclient.ConnectionContext{
		BaseURL: apiclient.BaseURL{Value: "https://example.test/api/v5", Source: apiclient.SourceFlag},
		Cred:    auth.Resolution{Source: auth.SourceNone},
	}
	return nil
}

func (w *roleDomainsWorld) apiUnreachable() error {
	w.transport = &cannedTransport{netErr: errors.New("dial tcp: connection refused")}
	return nil
}

func (w *roleDomainsWorld) roleControlsNoDomains(_ string) error {
	w.transport = &cannedTransport{status: 200, body: domainsPageEmpty}
	return nil
}

// domainExists returns a single-domain body that DOES carry policies, so the
// no-include single read renders description+role (policies omitted) and the
// --include policies read renders them inline — one Given serves both scenarios.
func (w *roleDomainsWorld) domainExists(_ string) error {
	w.transport = &cannedTransport{status: 200, body: getDomainBodyWithPolicies}
	return nil
}

func (w *roleDomainsWorld) domainNotFound(_ string) error {
	w.transport = &cannedTransport{status: 404, body: `{"detail":"Domain not found"}`}
	return nil
}

func (w *roleDomainsWorld) roleControlsMatchingDomain(_, term string) error {
	// The API does the full-text filtering; the fake returns a body that matches
	// the term so the run renders it, and the request-carries-q step asserts we
	// sent the search.
	w.transport = &cannedTransport{status: 200, body: domainsPage("dom_review", "Review "+term, "")}
	return nil
}

func (w *roleDomainsWorld) domainsSpanThreePages(_ string) error {
	w.transport = &seqMeTransport{steps: []seqMeResp{
		{status: 200, body: domainsPage("dom_1", "Page One Domain", "c1")},
		{status: 200, body: domainsPage("dom_2", "Page Two Domain", "c2")},
		{status: 200, body: domainsPage("dom_3", "Page Three Domain", "")},
	}}
	return nil
}

func (w *roleDomainsWorld) domainsSpanMoreThanOnePage(_ string) error {
	w.transport = &cannedTransport{status: 200, body: domainsPage("dom_1", "First Page Domain", "c1")}
	return nil
}

func (w *roleDomainsWorld) domainsWalkFailsMidway() error {
	w.transport = &seqMeTransport{steps: []seqMeResp{
		{status: 200, body: domainsPage("dom_1", "Gathered Domain", "c1")},
		{status: 500, body: `{"detail":"boom"}`},
	}}
	return nil
}

// --- When implementation ---

// runCommand parses the captured invocation and dispatches it through a real root
// with BOTH the `domains` and `domain` leaves attached over a fake seam (so the
// plural/singular pairing resolves exactly as in production). It asserts the
// secret token never leaks into any produced output.
func (w *roleDomainsWorld) runCommand(invocation string) error {
	root := NewRootCommand()
	seam := &fakeMeSeam{ctx: w.ctx, newClientErr: w.newClientErr, transport: w.transport}
	MustRegister(root, newDomainsCommand(seam))
	MustRegister(root, newDomainCommand(seam))

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
func (w *roleDomainsWorld) transportCalls() int {
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
// records it; the search/include scenarios all use it).
func (w *roleDomainsWorld) lastQuery() (string, error) {
	t, ok := w.transport.(*cannedTransport)
	if !ok {
		return "", fmt.Errorf("this scenario's transport does not record the query")
	}
	return t.lastQuery.Encode(), nil
}

// --- Then implementations ---

func (w *roleDomainsWorld) eachDomainPrinted() error {
	for _, want := range []string{"The marketing budget", "dom_budget", "The brand guidelines"} {
		if !strings.Contains(w.stdout, want) {
			return fmt.Errorf("each domain should print as a projection, missing %q:\n%s", want, w.stdout)
		}
	}
	return nil
}

func (w *roleDomainsWorld) exitWithCode(code int) error {
	if w.exitCode != code {
		return fmt.Errorf("exit code = %d, want %d (outcome %v)\nstderr: %s", w.exitCode, code, w.outcome, w.stderr)
	}
	return nil
}

func (w *roleDomainsWorld) exitNonZeroAPIError() error {
	if w.outcome != APIError || w.exitCode != 3 {
		return fmt.Errorf("outcome=%v exit=%d, want APIError/3\nstderr: %s", w.outcome, w.exitCode, w.stderr)
	}
	return nil
}

func (w *roleDomainsWorld) exitNonZero() error {
	if w.exitCode == 0 {
		return fmt.Errorf("exit code = 0, want a non-zero code (outcome %v)\nstderr: %s", w.outcome, w.stderr)
	}
	return nil
}

func (w *roleDomainsWorld) stderrReportsAndPointsTo(report, pointer string) error {
	if !strings.Contains(strings.ToLower(w.stderr), strings.ToLower(report)) {
		return fmt.Errorf("stderr should report %q:\n%s", report, w.stderr)
	}
	if !strings.Contains(w.stderr, pointer) {
		return fmt.Errorf("stderr should point to %q:\n%s", pointer, w.stderr)
	}
	return nil
}

func (w *roleDomainsWorld) noDomainDataPrinted() error {
	if strings.TrimSpace(w.stdout) != "" {
		return fmt.Errorf("no domain data should be printed, got stdout:\n%s", w.stdout)
	}
	return nil
}

func (w *roleDomainsWorld) transportFailureNamed() error {
	if w.outcome != NetworkUnavailable {
		return fmt.Errorf("outcome = %v, want NetworkUnavailable", w.outcome)
	}
	if strings.TrimSpace(w.stderr) == "" {
		return errors.New("a transport failure should be named on stderr")
	}
	return nil
}

func (w *roleDomainsWorld) literalPrintedToStdout(literal string) error {
	if !strings.Contains(w.stdout, literal) {
		return fmt.Errorf("stdout should contain %q:\n%s", literal, w.stdout)
	}
	return nil
}

func (w *roleDomainsWorld) singleDomainPrinted() error {
	for _, want := range []string{"The marketing budget", "role_0123"} {
		if !strings.Contains(w.stdout, want) {
			return fmt.Errorf("the single domain should show its description and controlling role; missing %q:\n%s", want, w.stdout)
		}
	}
	return nil
}

func (w *roleDomainsWorld) requestCarriesRawParam(param string) error {
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

func (w *roleDomainsWorld) policiesInline() error {
	for _, want := range []string{"Policies:", "Spend under $10k needs no approval"} {
		if !strings.Contains(w.stdout, want) {
			return fmt.Errorf("the requested policies should print inline within the domain; missing %q:\n%s", want, w.stdout)
		}
	}
	return nil
}

func (w *roleDomainsWorld) stderrNamesHTTPStatus() error {
	if !strings.Contains(w.stderr, "404") {
		return fmt.Errorf("stderr should name the HTTP status (404):\n%s", w.stderr)
	}
	return nil
}

func (w *roleDomainsWorld) stderrNamesUnsupportedInclude() error {
	if !strings.Contains(w.stderr, "nonsense") {
		return fmt.Errorf("stderr should name the unsupported value:\n%s", w.stderr)
	}
	if !strings.Contains(w.stderr, "policies") {
		return fmt.Errorf("stderr should name the supported set {policies}:\n%s", w.stderr)
	}
	return nil
}

func (w *roleDomainsWorld) noRequestSent() error {
	if w.transportCalls() != 0 {
		return fmt.Errorf("no API request should be sent, but the transport was called %d times", w.transportCalls())
	}
	return nil
}

func (w *roleDomainsWorld) stderrReportsUsageError() error {
	if w.outcome != UsageError || w.exitCode != 2 {
		return fmt.Errorf("outcome=%v exit=%d, want UsageError/2\nstderr: %s", w.outcome, w.exitCode, w.stderr)
	}
	if strings.TrimSpace(w.stderr) == "" {
		return errors.New("a usage error should be reported on stderr")
	}
	return nil
}

func (w *roleDomainsWorld) requestCarriesSearchTerm(param string) error {
	q, err := w.lastQuery()
	if err != nil {
		return err
	}
	if !strings.Contains(q, param+"=") {
		return fmt.Errorf("the request query %q should carry the %q search term", q, param)
	}
	return nil
}

func (w *roleDomainsWorld) matchingDomainsPrinted() error {
	if !strings.Contains(w.stdout, "Review") {
		return fmt.Errorf("only the matching domains should print:\n%s", w.stdout)
	}
	return nil
}

func (w *roleDomainsWorld) walkedEveryPage() error {
	if w.transportCalls() != 3 {
		return fmt.Errorf("the walk should issue three page requests, got %d", w.transportCalls())
	}
	return nil
}

func (w *roleDomainsWorld) allPagesPrinted() error {
	for _, want := range []string{"Page One Domain", "Page Two Domain", "Page Three Domain"} {
		if !strings.Contains(w.stdout, want) {
			return fmt.Errorf("every page's domains should print, missing %q:\n%s", want, w.stdout)
		}
	}
	return nil
}

func (w *roleDomainsWorld) onlyFirstPagePrinted() error {
	if !strings.Contains(w.stdout, "First Page Domain") {
		return fmt.Errorf("the first page should print:\n%s", w.stdout)
	}
	if w.transportCalls() != 1 {
		return fmt.Errorf("--first-page must not walk, want 1 call, got %d", w.transportCalls())
	}
	return nil
}

func (w *roleDomainsWorld) stderrNotesMoreExist() error {
	if !strings.Contains(w.stderr, "more domains exist") {
		return fmt.Errorf("stderr should note more domains exist:\n%s", w.stderr)
	}
	return nil
}

func (w *roleDomainsWorld) partialDomainsPrinted() error {
	if !strings.Contains(w.stdout, "Gathered Domain") {
		return fmt.Errorf("the partial set gathered so far should print:\n%s", w.stdout)
	}
	return nil
}

func (w *roleDomainsWorld) stderrNotesIncomplete() error {
	if !strings.Contains(w.stderr, "incomplete") {
		return fmt.Errorf("stderr should note the result is incomplete and name the cause:\n%s", w.stderr)
	}
	return nil
}
