package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/Luscii/cli-glassfrog/internal/apiclient"
	"github.com/Luscii/cli-glassfrog/internal/auth"
	"github.com/cucumber/godog"
)

// TestIdentityReadFeatures runs the executable acceptance for Identity Read
// (011): the `me` command driven through its seam over a fake base transport, so
// every scenario runs offline (no real network, no real ~/.glassfrogrc). Its
// Paths name ONLY this spec's feature file — never the features/ directory — so
// un-@wip-ping these scenarios cannot disturb another suite, and each
// internal/cli suite reports its own independent scenario count (LEARNINGS: a
// suite points at its own feature file). The 4 @validation scenarios stay @wip
// (held for the validate skill) and are skipped by the ~@wip filter.
func TestIdentityReadFeatures(t *testing.T) {
	suite := godog.TestSuite{
		ScenarioInitializer: initializeIdentityReadScenario,
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"../../features/self-service-reads/identity-read.feature"},
			Tags:     "~@wip",
			TestingT: t,
		},
	}
	if suite.Run() != 0 {
		t.Fatal("non-zero status: identity-read feature scenarios failed")
	}
}

// meWorld is the per-scenario state for the identity-read suite: the connection
// context and fake transport assembled from the Given steps, plus the captured
// outcome/exit-code/streams of the When run. Everything is injected — no step
// touches the real network, env, or home.
type meWorld struct {
	ctx          apiclient.ConnectionContext
	newClientErr error
	transport    *cannedTransport
	secret       string

	outcome  Outcome
	exitCode int
	stdout   string
	stderr   string
}

func initializeIdentityReadScenario(sc *godog.ScenarioContext) {
	w := &meWorld{}
	sc.Before(func(ctx context.Context, _ *godog.Scenario) (context.Context, error) {
		*w = meWorld{
			// A 2xx body is the default transport response; error scenarios override
			// status/netErr in their Given steps.
			transport: &cannedTransport{status: 200, body: meBodyAlice},
			secret:    meSecretToken,
		}
		w.ctx = validMeContext()
		return ctx, nil
	})

	// --- Givens ---
	sc.Step(`^a complete connection context with a present, valid token$`, w.completeContext)
	sc.Step(`^the API would return the actor "([^"]*)", organization "([^"]*)", and access level "([^"]*)"$`, w.apiReturnsIdentity)
	sc.Step(`^a complete connection context whose token resolves to an agent actor$`, w.agentActor)
	sc.Step(`^the agent actor has an "([^"]*)" id$`, w.agentActorHasID)
	sc.Step(`^the actor fills the roles "([^"]*)" and "([^"]*)"$`, w.actorFillsRoles)
	sc.Step(`^a complete connection context whose actor fills no roles$`, w.actorFillsNoRoles)
	sc.Step(`^a complete connection context whose token is expired or wrong$`, w.completeContext)
	sc.Step(`^the API would answer with a (\d+) response$`, w.apiAnswersStatus)
	sc.Step(`^a complete connection context with a usable base URL$`, w.completeContext)
	sc.Step(`^the API could not be reached$`, w.apiUnreachable)
	sc.Step(`^a connection context with a usable base URL but no usable token$`, w.contextNoToken)
	sc.Step(`^the API would return a (\d+) response whose body does not match the identity shape$`, w.apiReturnsUndecodable)
	sc.Step(`^a connection context carrying a base-URL error from a malformed configured value$`, w.contextBaseURLError)
	sc.Step(`^a connection context whose credentials file is malformed$`, w.contextMalformedCredFile)

	// --- Whens ---
	sc.Step(`^the operator runs the me command$`, w.runMeCmd)
	sc.Step(`^the operator runs the me command with the roles embed requested$`, w.runMeCmdWithRoles)
	sc.Step(`^the operator runs the me command requesting an include target the spec does not define$`, w.runMeCmdBadInclude)

	// --- Thens ---
	sc.Step(`^the projection will name the actor "([^"]*)" with its id and kind$`, w.projectionNamesActor)
	sc.Step(`^it will name the organization "([^"]*)" with its id$`, w.projectionNamesOrg)
	sc.Step(`^it will report the access level "([^"]*)"$`, w.projectionReportsAccess)
	sc.Step(`^the command will exit successfully$`, w.exitedSuccessfully)
	sc.Step(`^the projection will report the actor kind as agent$`, w.projectionKindAgent)
	sc.Step(`^it will surface the "([^"]*)" id as the actionable handle$`, w.projectionSurfacesID)
	sc.Step(`^the request will carry the include-roles query parameter$`, w.requestCarriedIncludeRoles)
	sc.Step(`^the projection will list each role's id and name alongside the identity facts$`, w.projectionListsRoles)
	sc.Step(`^the projection will print the identity facts$`, w.projectionPrintsIdentity)
	sc.Step(`^it will omit the roles section rather than printing an empty list$`, w.projectionOmitsRoles)
	sc.Step(`^the command will reject the input as a usage error naming the unsupported target$`, w.rejectedUnsupportedInclude)
	sc.Step(`^no request will reach the API$`, w.noRequestReachedAPI)
	sc.Step(`^the command will surface the non-2xx outcome with its status code$`, w.surfacedNon2xxWithStatus)
	sc.Step(`^it will print no identity projection$`, w.printedNoProjection)
	sc.Step(`^it will exit with a non-success result$`, w.exitedNonSuccess)
	sc.Step(`^the command will exit with a non-success result$`, w.exitedNonSuccess)
	sc.Step(`^the command will surface a transport failure naming the cause$`, w.surfacedTransportFailure)
	sc.Step(`^it will not retry$`, w.didNotRetry)
	sc.Step(`^the authenticated transport's fail-safe will refuse the call$`, w.failSafeRefused)
	sc.Step(`^no unauthenticated request will be sent$`, w.noRequestReachedAPI)
	sc.Step(`^the command will surface a decode failure$`, w.surfacedDecodeFailure)
	sc.Step(`^it will exit with the internal-error result rather than a success$`, w.exitedInternalError)
	sc.Step(`^the command will surface the base-URL problem as a usage error$`, w.surfacedBaseURLUsageError)
	sc.Step(`^the message will explain the next step to correct the configured base URL$`, w.messageExplainsBaseURLNextStep)
	sc.Step(`^the command will surface a credential-file error naming the file$`, w.surfacedCredFileError)
	sc.Step(`^the message will explain the next step to fix or re-create the file$`, w.messageExplainsCredNextStep)
	sc.Step(`^it will exit with the internal-error result$`, w.exitedInternalError)
}

// --- Given implementations ---

func (w *meWorld) completeContext() error { w.ctx = validMeContext(); return nil }

func (w *meWorld) apiReturnsIdentity(actorName, orgName, access string) error {
	w.transport = &cannedTransport{status: 200, body: fmt.Sprintf(`{
      "actor": {"id": "per_0123456789abcdef0123456789abcdef", "name": %q, "kind": "human"},
      "organization": {"id": "org_0123456789abcdef0123456789abcdef", "name": %q},
      "membership": {"access_level": %q}
    }`, actorName, orgName, access)}
	return nil
}

func (w *meWorld) agentActor() error {
	w.ctx = validMeContext()
	w.transport = &cannedTransport{status: 200, body: meBodyAgentWithRoles}
	return nil
}

func (w *meWorld) agentActorHasID(_ string) error { return nil } // the agt_ id is in the body

func (w *meWorld) actorFillsRoles(_, _ string) error {
	w.transport = &cannedTransport{status: 200, body: meBodyAgentWithRoles}
	return nil
}

func (w *meWorld) actorFillsNoRoles() error {
	w.ctx = validMeContext()
	w.transport = &cannedTransport{status: 200, body: meBodyAlice} // no roles field
	return nil
}

func (w *meWorld) apiAnswersStatus(status int) error {
	w.transport = &cannedTransport{status: status, body: `{"error":"unauthorized"}`}
	return nil
}

func (w *meWorld) apiUnreachable() error {
	w.transport = &cannedTransport{netErr: errors.New("dial tcp: connection refused")}
	return nil
}

func (w *meWorld) contextNoToken() error {
	w.ctx = apiclient.ConnectionContext{
		BaseURL: apiclient.BaseURL{Value: "https://example.test/api/v5", Source: apiclient.SourceFlag},
		Cred:    auth.Resolution{Source: auth.SourceNone},
	}
	return nil
}

func (w *meWorld) apiReturnsUndecodable(_ int) error {
	w.transport = &cannedTransport{status: 200, body: `this is not the identity shape`}
	return nil
}

func (w *meWorld) contextBaseURLError() error {
	w.ctx = apiclient.ConnectionContext{}
	w.newClientErr = &apiclient.BaseURLError{Source: "--" + apiclient.FlagBaseURL}
	return nil
}

func (w *meWorld) contextMalformedCredFile() error {
	// The CredErr surfaces through the AuthTransport as *AuthError{CredentialError}
	// wrapping this cause; the cause names the file (path-only, never the token).
	w.ctx = apiclient.ConnectionContext{
		BaseURL: apiclient.BaseURL{Value: "https://example.test/api/v5", Source: apiclient.SourceFlag},
		Cred:    auth.Resolution{Source: auth.SourceNone},
		CredErr: errors.New("the .glassfrogrc credentials file is malformed"),
	}
	return nil
}

// --- When implementations ---

func (w *meWorld) run(args ...string) error {
	root := NewRootCommand()
	seam := &fakeMeSeam{ctx: w.ctx, newClientErr: w.newClientErr, transport: w.transport}
	MustRegister(root, newMeCommand(seam))

	var out, errb bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&errb)
	w.outcome, _ = Run(root, append([]string{"me"}, args...))
	w.exitCode = ExitCode(w.outcome)
	w.stdout, w.stderr = out.String(), errb.String()

	// Secret-hygiene invariant across every scenario: the token never appears in
	// any produced output, success or failure.
	if w.secret != "" && strings.Contains(w.stdout+w.stderr, w.secret) {
		return fmt.Errorf("the token leaked into output: stdout=%q stderr=%q", w.stdout, w.stderr)
	}
	return nil
}

func (w *meWorld) runMeCmd() error          { return w.run() }
func (w *meWorld) runMeCmdWithRoles() error { return w.run("--include", "roles") }
func (w *meWorld) runMeCmdBadInclude() error {
	return w.run("--include", "tensions-that-do-not-exist")
}

// --- Then implementations ---

func (w *meWorld) projectionNamesActor(name string) error {
	if !strings.Contains(w.stdout, name) {
		return fmt.Errorf("projection should name the actor %q:\n%s", name, w.stdout)
	}
	// "with its id and kind": a per_/agt_ handle and a (kind) marker.
	if !strings.Contains(w.stdout, "per_") && !strings.Contains(w.stdout, "agt_") {
		return fmt.Errorf("projection should include the actor id handle:\n%s", w.stdout)
	}
	if !strings.Contains(w.stdout, "(human)") && !strings.Contains(w.stdout, "(agent)") {
		return fmt.Errorf("projection should include the actor kind:\n%s", w.stdout)
	}
	return nil
}

func (w *meWorld) projectionNamesOrg(name string) error {
	if !strings.Contains(w.stdout, name) || !strings.Contains(w.stdout, "org_") {
		return fmt.Errorf("projection should name the organization %q with its org_ id:\n%s", name, w.stdout)
	}
	return nil
}

func (w *meWorld) projectionReportsAccess(level string) error {
	if !strings.Contains(w.stdout, level) {
		return fmt.Errorf("projection should report access level %q:\n%s", level, w.stdout)
	}
	return nil
}

func (w *meWorld) exitedSuccessfully() error {
	if w.outcome != Success || w.exitCode != 0 {
		return fmt.Errorf("outcome=%v exit=%d, want Success/0\nstderr: %s", w.outcome, w.exitCode, w.stderr)
	}
	return nil
}

func (w *meWorld) projectionKindAgent() error {
	if !strings.Contains(w.stdout, "(agent)") {
		return fmt.Errorf("projection should report the actor kind as agent:\n%s", w.stdout)
	}
	return nil
}

func (w *meWorld) projectionSurfacesID(prefix string) error {
	if !strings.Contains(w.stdout, prefix) {
		return fmt.Errorf("projection should surface the %s id:\n%s", prefix, w.stdout)
	}
	return nil
}

func (w *meWorld) requestCarriedIncludeRoles() error {
	if got := w.transport.lastQuery.Get("include"); got != "roles" {
		return fmt.Errorf("request should carry include=roles, got %q", got)
	}
	return nil
}

func (w *meWorld) projectionListsRoles() error {
	for _, want := range []string{"roles:", "Marketing Lead", "role_", "Treasurer"} {
		if !strings.Contains(w.stdout, want) {
			return fmt.Errorf("projection should list each role's id and name, missing %q:\n%s", want, w.stdout)
		}
	}
	return nil
}

func (w *meWorld) projectionPrintsIdentity() error {
	for _, want := range []string{"actor:", "organization:", "access:"} {
		if !strings.Contains(w.stdout, want) {
			return fmt.Errorf("projection should print the identity facts, missing %q:\n%s", want, w.stdout)
		}
	}
	return nil
}

func (w *meWorld) projectionOmitsRoles() error {
	if strings.Contains(w.stdout, "roles:") {
		return fmt.Errorf("an empty roles embed should omit the roles section:\n%s", w.stdout)
	}
	return nil
}

func (w *meWorld) rejectedUnsupportedInclude() error {
	if w.outcome != UsageError || w.exitCode != 2 {
		return fmt.Errorf("outcome=%v exit=%d, want UsageError/2", w.outcome, w.exitCode)
	}
	if !strings.Contains(w.stderr, "tensions-that-do-not-exist") {
		return fmt.Errorf("the usage error should name the unsupported target:\n%s", w.stderr)
	}
	return nil
}

func (w *meWorld) noRequestReachedAPI() error {
	if w.transport.calls != 0 {
		return fmt.Errorf("no request should reach the API, but the transport was called %d times", w.transport.calls)
	}
	return nil
}

func (w *meWorld) surfacedNon2xxWithStatus() error {
	if w.outcome != APIError || w.exitCode != 3 {
		return fmt.Errorf("outcome=%v exit=%d, want APIError/3", w.outcome, w.exitCode)
	}
	if !strings.Contains(w.stderr, fmt.Sprintf("%d", w.transport.status)) {
		return fmt.Errorf("stderr should surface the status code %d:\n%s", w.transport.status, w.stderr)
	}
	return nil
}

func (w *meWorld) printedNoProjection() error {
	if strings.TrimSpace(w.stdout) != "" {
		return fmt.Errorf("no identity projection should print, got stdout:\n%s", w.stdout)
	}
	return nil
}

func (w *meWorld) exitedNonSuccess() error {
	if w.outcome == Success || w.exitCode == 0 {
		return fmt.Errorf("should exit non-success, got outcome=%v exit=%d", w.outcome, w.exitCode)
	}
	return nil
}

func (w *meWorld) surfacedTransportFailure() error {
	if w.outcome != NetworkUnavailable || w.exitCode != 6 {
		return fmt.Errorf("outcome=%v exit=%d, want NetworkUnavailable/6", w.outcome, w.exitCode)
	}
	if strings.TrimSpace(w.stderr) == "" {
		return errors.New("a transport failure should name the cause on stderr")
	}
	return nil
}

func (w *meWorld) didNotRetry() error {
	if w.transport.calls != 1 {
		return fmt.Errorf("the read must not retry, but the transport was called %d times", w.transport.calls)
	}
	return nil
}

func (w *meWorld) failSafeRefused() error {
	// The fail-safe refusal surfaces as a non-success outcome with no request
	// sent; UsageError is the NoCredentials mapping (002 invalid-input).
	if w.outcome != UsageError || w.exitCode != 2 {
		return fmt.Errorf("outcome=%v exit=%d, want UsageError/2 (NoCredentials fail-safe)", w.outcome, w.exitCode)
	}
	return nil
}

func (w *meWorld) surfacedDecodeFailure() error {
	if strings.TrimSpace(w.stderr) == "" {
		return errors.New("a decode failure should be reported on stderr")
	}
	return nil
}

func (w *meWorld) exitedInternalError() error {
	if w.outcome != RuntimeError || w.exitCode != 1 {
		return fmt.Errorf("outcome=%v exit=%d, want RuntimeError/1 (internal error)\nstderr: %s", w.outcome, w.exitCode, w.stderr)
	}
	return nil
}

func (w *meWorld) surfacedBaseURLUsageError() error {
	if w.outcome != UsageError || w.exitCode != 2 {
		return fmt.Errorf("outcome=%v exit=%d, want UsageError/2", w.outcome, w.exitCode)
	}
	if !strings.Contains(strings.ToLower(w.stderr), "base url") && !strings.Contains(w.stderr, "base-url") {
		return fmt.Errorf("stderr should surface the base-URL problem:\n%s", w.stderr)
	}
	return nil
}

func (w *meWorld) messageExplainsBaseURLNextStep() error {
	if !strings.Contains(w.stderr, "--"+apiclient.FlagBaseURL) {
		return fmt.Errorf("the message should explain correcting --%s:\n%s", apiclient.FlagBaseURL, w.stderr)
	}
	return nil
}

func (w *meWorld) surfacedCredFileError() error {
	if !strings.Contains(w.stderr, ".glassfrogrc") {
		return fmt.Errorf("stderr should name the credentials file:\n%s", w.stderr)
	}
	return nil
}

func (w *meWorld) messageExplainsCredNextStep() error {
	if !strings.Contains(w.stderr, "auth login") {
		return fmt.Errorf("the message should explain fixing/re-creating the file via `glassfrog auth login`:\n%s", w.stderr)
	}
	return nil
}
