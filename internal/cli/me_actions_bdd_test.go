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

// TestMeActionsFeatures runs the executable acceptance for My Actions (013): the
// `me actions` command driven through its seam over a fake base transport, so
// every scenario runs offline (no real network, no real ~/.glassfrogrc). Its
// Paths name ONLY this spec's feature file — never the features/ directory — so
// un-@wip-ping these scenarios cannot disturb another suite, and the suite
// reports its own independent scenario count (LEARNINGS: a suite points at its
// own feature file). The @validation scenarios stay @wip (held for the validate
// skill) and are skipped by the ~@wip filter.
func TestMeActionsFeatures(t *testing.T) {
	suite := godog.TestSuite{
		ScenarioInitializer: initializeMeActionsScenario,
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"../../features/self-service-reads/my-actions.feature"},
			Tags:     "~@wip",
			TestingT: t,
		},
	}
	if suite.Run() != 0 {
		t.Fatal("non-zero status: my-actions feature scenarios failed")
	}
}

// meActionsWorld is the per-scenario state for the my-actions suite: the
// connection context and fake transport assembled from the Given steps, plus the
// captured outcome/exit-code/streams of the When run. Everything is injected — no
// step touches the real network, env, or home.
type meActionsWorld struct {
	ctx          apiclient.ConnectionContext
	newClientErr error
	transport    *cannedTransport
	secret       string

	outcome  Outcome
	exitCode int
	stdout   string
	stderr   string
}

func initializeMeActionsScenario(sc *godog.ScenarioContext) {
	w := &meActionsWorld{}
	sc.Before(func(ctx context.Context, _ *godog.Scenario) (context.Context, error) {
		*w = meActionsWorld{
			// A multi-action 2xx body is the default; error and shape scenarios
			// override status/netErr/body in their Given steps.
			transport: &cannedTransport{status: 200, body: actionsBodyMulti},
			secret:    meSecretToken,
		}
		w.ctx = validMeContext()
		return ctx, nil
	})

	// --- Givens ---
	sc.Step(`^a complete connection context with a present, valid token$`, w.completeContext)
	sc.Step(`^a complete connection context with a usable base URL$`, w.completeContext)
	sc.Step(`^the API would return one page of actions the practitioner owns$`, w.apiReturnsActions)
	sc.Step(`^the practitioner owns no actions matching the request$`, w.apiReturnsNoActions)
	sc.Step(`^a connection context with a usable base URL but no usable token$`, w.contextNoToken)
	sc.Step(`^the API would answer the my-actions read with a non-2xx response$`, w.apiAnswersNon2xx)
	sc.Step(`^the API could not be reached$`, w.apiUnreachable)
	sc.Step(`^the API would return a 200 response whose body does not match the actions shape$`, w.apiReturnsUnparseable)
	sc.Step(`^a connection context carrying a base-URL error from a malformed configured value$`, w.contextBaseURLError)
	sc.Step(`^a connection context whose credentials file is malformed$`, w.contextCredentialError)
	sc.Step(`^the API would return a first page reporting that more results are available$`, w.apiReturnsFirstPageHasNext)

	// --- Whens ---
	sc.Step(`^the operator runs the my-actions command with no status filter$`, w.runNoFilter)
	sc.Step(`^the operator runs the my-actions command with the status filter "([^"]*)"$`, w.runWithStatus)
	sc.Step(`^the operator runs the my-actions command with a status value outside the spec's vocabulary$`, w.runWithUnsupportedStatus)
	sc.Step(`^the operator runs the my-actions command$`, w.runNoFilter)

	// --- Thens ---
	sc.Step(`^the request will go to the my-actions endpoint$`, w.requestWentToEndpoint)
	sc.Step(`^the projection will list each action with its id, status, description, and owning role$`, w.projectionListsActions)
	sc.Step(`^the command will exit successfully$`, w.exitSuccess)
	sc.Step(`^the projection will report an empty list$`, w.projectionEmptyList)
	sc.Step(`^the authenticated transport's fail-safe will refuse the call$`, w.failSafeRefused)
	sc.Step(`^the command will exit with a non-success result$`, w.exitNonSuccess)
	sc.Step(`^it will exit with a non-success result$`, w.exitNonSuccess)
	sc.Step(`^no unauthenticated request will be sent$`, w.noRequestReachedAPI)
	sc.Step(`^the command will surface the generic non-2xx outcome carrying the status$`, w.surfacesGenericNon2xx)
	sc.Step(`^it will not turn it into a specific, interpreted API error message$`, w.notInterpretedAPIError)
	sc.Step(`^the command will surface a transport failure naming the cause$`, w.surfacesTransportFailure)
	sc.Step(`^it will not retry$`, w.didNotRetry)
	sc.Step(`^the command will surface a decode failure$`, w.surfacesDecodeFailure)
	sc.Step(`^it will exit with the internal-error result rather than a success$`, w.exitInternalError)
	sc.Step(`^the command will surface the base-URL problem as a usage error$`, w.surfacesBaseURLUsageError)
	sc.Step(`^the message will explain the next step to correct the configured base URL$`, w.baseURLNextStep)
	sc.Step(`^no request will reach the API$`, w.noRequestReachedAPI)
	sc.Step(`^the command will surface a credential-file error naming the file$`, w.surfacesCredentialFileError)
	sc.Step(`^the message will explain the next step to fix or re-create the file$`, w.credentialFileNextStep)
	sc.Step(`^it will exit with the internal-error result$`, w.exitInternalError)
	sc.Step(`^the value "([^"]*)" will be accepted as a supported status$`, w.statusAccepted)
	sc.Step(`^the request will carry the status filter "([^"]*)"$`, w.requestCarriesStatus)
	sc.Step(`^only the actions the API returns for that filter will be rendered$`, w.filteredActionsRendered)
	sc.Step(`^the command will reject the input as a usage error naming the unsupported value and the supported set$`, w.rejectsUnsupportedStatus)
	sc.Step(`^the projection will render only the first page$`, w.rendersFirstPage)
	sc.Step(`^it will surface a clear "more results are available" signal$`, w.surfacesMoreAvailableSignal)
	sc.Step(`^it will not request a second page$`, w.didNotRequestSecondPage)
}

// --- Given implementations ---

func (w *meActionsWorld) completeContext() error { w.ctx = validMeContext(); return nil }

func (w *meActionsWorld) apiReturnsActions() error {
	w.transport = &cannedTransport{status: 200, body: actionsBodyMulti}
	return nil
}

func (w *meActionsWorld) apiReturnsNoActions() error {
	w.transport = &cannedTransport{status: 200, body: actionsBodyEmpty}
	return nil
}

func (w *meActionsWorld) contextNoToken() error {
	w.ctx = apiclient.ConnectionContext{
		BaseURL: apiclient.BaseURL{Value: "https://example.test/api/v5", Source: apiclient.SourceFlag},
		Cred:    auth.Resolution{Source: auth.SourceNone},
	}
	return nil
}

func (w *meActionsWorld) apiAnswersNon2xx() error {
	// A genuinely generic non-2xx (500): API Error Extraction (015) split
	// 401/403→permission(4) and 429→rate-limit(5), so a 5xx is the faithful
	// representative of "a non-2xx response" surfaced as the generic APIError.
	w.transport = &cannedTransport{status: 500, body: `{"error":"server error"}`}
	return nil
}

func (w *meActionsWorld) apiUnreachable() error {
	w.transport = &cannedTransport{netErr: errors.New("dial tcp: connection refused")}
	return nil
}

func (w *meActionsWorld) apiReturnsUnparseable() error {
	w.transport = &cannedTransport{status: 200, body: `this is not the actions shape`}
	return nil
}

func (w *meActionsWorld) contextBaseURLError() error {
	w.ctx = apiclient.ConnectionContext{}
	w.newClientErr = &apiclient.BaseURLError{Source: "--" + apiclient.FlagBaseURL}
	return nil
}

func (w *meActionsWorld) contextCredentialError() error {
	w.ctx = apiclient.ConnectionContext{
		BaseURL: apiclient.BaseURL{Value: "https://example.test/api/v5", Source: apiclient.SourceFlag},
		Cred:    auth.Resolution{Source: auth.SourceNone},
		CredErr: errors.New("read /home/practitioner/.glassfrogrc: malformed credentials file"),
	}
	return nil
}

func (w *meActionsWorld) apiReturnsFirstPageHasNext() error {
	w.transport = &cannedTransport{status: 200, body: actionsBodyHasNext}
	return nil
}

// --- When implementations ---

// run builds a real root with `me` (runnable) and the `actions` leaf attached
// over the fake seam, dispatches `me actions [args]`, and captures the
// outcome/exit-code/streams. It asserts the secret token never leaks into any
// produced output (the cross-read invariant, pinned on every When).
func (w *meActionsWorld) run(args ...string) error {
	root := NewRootCommand()
	seam := &fakeMeSeam{ctx: w.ctx, newClientErr: w.newClientErr, transport: w.transport}
	meCmd := newMeCommand(seam)
	MustRegister(root, meCmd)
	MustRegister(meCmd, newMeActionsCommand(seam))

	var out, errb bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&errb)
	w.outcome, _ = Run(root, append([]string{"me", "actions"}, args...))
	w.exitCode = ExitCode(w.outcome)
	w.stdout, w.stderr = out.String(), errb.String()

	if w.secret != "" && strings.Contains(w.stdout+w.stderr, w.secret) {
		return fmt.Errorf("the token leaked into output: stdout=%q stderr=%q", w.stdout, w.stderr)
	}
	return nil
}

func (w *meActionsWorld) runNoFilter() error { return w.run() }

func (w *meActionsWorld) runWithStatus(status string) error { return w.run("--status", status) }

func (w *meActionsWorld) runWithUnsupportedStatus() error {
	return w.run("--status", "definitely-not-a-status")
}

// --- Then implementations ---

func (w *meActionsWorld) requestWentToEndpoint() error {
	if w.transport.calls != 1 {
		return fmt.Errorf("exactly one request should reach the my-actions endpoint, got %d calls", w.transport.calls)
	}
	if !strings.HasSuffix(w.transport.lastPath, "/me/actions") {
		return fmt.Errorf("the request should go to the /me/actions endpoint, got path %q", w.transport.lastPath)
	}
	return nil
}

func (w *meActionsWorld) projectionListsActions() error {
	for _, want := range []string{
		"actn_0123456789abcdef0123456789abcdef", "current", "Review PR #6818",
		"role_0123456789abcdef0123456789abcdef", "waiting",
	} {
		if !strings.Contains(w.stdout, want) {
			return fmt.Errorf("the projection should list each action's id/status/description/role, missing %q:\n%s", want, w.stdout)
		}
	}
	return nil
}

func (w *meActionsWorld) exitSuccess() error {
	if w.outcome != Success || w.exitCode != 0 {
		return fmt.Errorf("the command should exit successfully, got outcome=%v code=%d\nstderr: %s", w.outcome, w.exitCode, w.stderr)
	}
	return nil
}

func (w *meActionsWorld) projectionEmptyList() error {
	if strings.TrimRight(w.stdout, "\n") != "No actions." {
		return fmt.Errorf("an empty list should render exactly `No actions.`, got %q", w.stdout)
	}
	return nil
}

func (w *meActionsWorld) failSafeRefused() error {
	// The AuthTransport fail-safe refuses a no-token call → UsageError (not a
	// transport outcome). No projection prints.
	if w.outcome != UsageError {
		return fmt.Errorf("the no-token fail-safe should refuse the call as a usage error, got %v", w.outcome)
	}
	if strings.TrimSpace(w.stdout) != "" {
		return fmt.Errorf("no action data should print when the call is refused, got %q", w.stdout)
	}
	return nil
}

func (w *meActionsWorld) exitNonSuccess() error {
	if w.outcome == Success || w.exitCode == 0 {
		return fmt.Errorf("the command should exit with a non-success result, got outcome=%v code=%d", w.outcome, w.exitCode)
	}
	return nil
}

func (w *meActionsWorld) noRequestReachedAPI() error {
	if w.transport.calls != 0 {
		return fmt.Errorf("no request should reach the API, but the transport was called %d times", w.transport.calls)
	}
	return nil
}

func (w *meActionsWorld) surfacesGenericNon2xx() error {
	if w.outcome != APIError {
		return fmt.Errorf("a non-2xx should surface the generic APIError outcome, got %v", w.outcome)
	}
	if !strings.Contains(w.stderr, "500") {
		return fmt.Errorf("the message should carry the status (500), got %q", w.stderr)
	}
	return nil
}

func (w *meActionsWorld) notInterpretedAPIError() error {
	// The generic non-2xx message must not interpret the status into a specific
	// meaning (no "forbidden"/"permission"/"unauthorized"/"rate limit" wording —
	// that is 015/017's concern). It names the status and a generic next step.
	lower := strings.ToLower(w.stderr)
	for _, forbidden := range []string{"permission", "unauthorized", "forbidden", "rate limit", "rate-limit"} {
		if strings.Contains(lower, forbidden) {
			return fmt.Errorf("the non-2xx message must not interpret the status (%q present):\n%s", forbidden, w.stderr)
		}
	}
	return nil
}

func (w *meActionsWorld) surfacesTransportFailure() error {
	if w.outcome != NetworkUnavailable {
		return fmt.Errorf("a wire failure should surface NetworkUnavailable, got %v", w.outcome)
	}
	if strings.TrimSpace(w.stderr) == "" {
		return errors.New("a transport failure should name the cause on stderr")
	}
	return nil
}

func (w *meActionsWorld) didNotRetry() error {
	if w.transport.calls != 1 {
		return fmt.Errorf("the read must not retry, got %d calls", w.transport.calls)
	}
	return nil
}

func (w *meActionsWorld) surfacesDecodeFailure() error {
	if w.outcome != RuntimeError {
		return fmt.Errorf("an undecodable 2xx should surface RuntimeError, got %v", w.outcome)
	}
	if strings.TrimSpace(w.stderr) == "" {
		return errors.New("a decode failure should be reported on stderr")
	}
	return nil
}

func (w *meActionsWorld) exitInternalError() error {
	if w.outcome != RuntimeError || w.exitCode != 1 {
		return fmt.Errorf("the internal-error result is RuntimeError/1, got outcome=%v code=%d", w.outcome, w.exitCode)
	}
	return nil
}

func (w *meActionsWorld) surfacesBaseURLUsageError() error {
	if w.outcome != UsageError {
		return fmt.Errorf("a base-URL problem should surface a usage error, got %v", w.outcome)
	}
	if !strings.Contains(w.stderr, "base-url") && !strings.Contains(strings.ToLower(w.stderr), "base url") {
		return fmt.Errorf("stderr should name the base-URL problem, got %q", w.stderr)
	}
	return nil
}

func (w *meActionsWorld) baseURLNextStep() error {
	if !strings.Contains(w.stderr, "--"+apiclient.FlagBaseURL) {
		return fmt.Errorf("the message should explain how to correct the base URL (name --%s), got %q", apiclient.FlagBaseURL, w.stderr)
	}
	return nil
}

func (w *meActionsWorld) surfacesCredentialFileError() error {
	if w.outcome != RuntimeError {
		return fmt.Errorf("a malformed credentials file should surface RuntimeError, got %v", w.outcome)
	}
	if !strings.Contains(w.stderr, ".glassfrogrc") {
		return fmt.Errorf("the message should name the credentials file, got %q", w.stderr)
	}
	return nil
}

func (w *meActionsWorld) credentialFileNextStep() error {
	if !strings.Contains(w.stderr, "auth login") && !strings.Contains(strings.ToLower(w.stderr), "re-create") {
		return fmt.Errorf("the message should explain how to fix or re-create the file, got %q", w.stderr)
	}
	return nil
}

func (w *meActionsWorld) statusAccepted(status string) error {
	// "current" was accepted: the request was issued (not rejected before send).
	if w.outcome != Success {
		return fmt.Errorf("status %q should be accepted and the read succeed, got %v\nstderr: %s", status, w.outcome, w.stderr)
	}
	if w.transport.calls != 1 {
		return fmt.Errorf("an accepted status should issue exactly one request, got %d calls", w.transport.calls)
	}
	return nil
}

func (w *meActionsWorld) requestCarriesStatus(status string) error {
	if got := w.transport.lastQuery.Get("status"); got != status {
		return fmt.Errorf("the request should carry ?status=%s, got query %v", status, w.transport.lastQuery)
	}
	return nil
}

func (w *meActionsWorld) filteredActionsRendered() error {
	// The fake returns the multi-action body for the filter; the projection
	// renders those actions.
	if !strings.Contains(w.stdout, "actn_0123456789abcdef0123456789abcdef") {
		return fmt.Errorf("the actions the API returned for the filter should be rendered:\n%s", w.stdout)
	}
	return nil
}

func (w *meActionsWorld) rejectsUnsupportedStatus() error {
	if w.outcome != UsageError || w.exitCode != 2 {
		return fmt.Errorf("an unsupported status should be a usage error (exit 2), got outcome=%v code=%d", w.outcome, w.exitCode)
	}
	if !strings.Contains(w.stderr, "definitely-not-a-status") {
		return fmt.Errorf("the message should name the unsupported value, got %q", w.stderr)
	}
	// The supported set is listed so the operator can self-correct.
	for _, s := range []string{"current", "waiting", "completed"} {
		if !strings.Contains(w.stderr, s) {
			return fmt.Errorf("the message should list the supported set (missing %q), got %q", s, w.stderr)
		}
	}
	return nil
}

func (w *meActionsWorld) rendersFirstPage() error {
	if !strings.Contains(w.stdout, "actn_0123456789abcdef0123456789abcdef") {
		return fmt.Errorf("the first page should be rendered to stdout:\n%s", w.stdout)
	}
	return nil
}

func (w *meActionsWorld) surfacesMoreAvailableSignal() error {
	// The signal rides stderr (012's convention), pinned verbatim against the
	// rendering constant so wording drift fails the acceptance.
	if strings.TrimRight(w.stderr, "\n") != incompleteActionsNote {
		return fmt.Errorf("stderr should be exactly the pinned more-available note %q, got %q", incompleteActionsNote, w.stderr)
	}
	return nil
}

func (w *meActionsWorld) didNotRequestSecondPage() error {
	if w.transport.calls != 1 {
		return fmt.Errorf("a further page must be signalled, not fetched; got %d requests", w.transport.calls)
	}
	return nil
}
