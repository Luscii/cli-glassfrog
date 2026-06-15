package cli

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/Luscii/cli-glassfrog/internal/apiclient"
	"github.com/cucumber/godog"
)

// TestProposalCreationFeatures runs the executable acceptance for Proposal Creation
// (055): the `proposal create <tension-id> --changes <src>` write, driven through the
// shared proposalSeam over a fake base transport so every scenario runs offline (no
// real network, ~/.glassfrogrc, pipe, or filesystem). Its Paths name ONLY this spec's
// feature file — never the features/ directory — so the suite reports its own
// independent scenario count and cannot disturb another suite (LEARNINGS: a suite
// points at its own feature file). The 3 @validation scenarios stay @wip (held for the
// validate skill) and are skipped by the ~@wip filter.
func TestProposalCreationFeatures(t *testing.T) {
	suite := godog.TestSuite{
		ScenarioInitializer: initializeProposalCreationScenario,
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"../../features/proposal-write-flow/proposal-creation.feature"},
			Tags:     "~@wip",
			TestingT: t,
		},
	}
	if suite.Run() != 0 {
		t.Fatal("non-zero status: proposal-creation feature scenarios failed")
	}
}

// splitArgsPOSIX splits a command line into arguments with POSIX-style quote grouping:
// BOTH single and double quotes group a token, and a quote of one kind is preserved
// literally inside the other kind. The landed splitArgs (cross_model_search_bdd_test)
// toggles inQuote on every double-quote and so corrupts a single-quoted inline JSON
// (`--changes '[{"type":...}]'`) by splitting on its embedded double quotes. This
// splitter keeps the embedded double quotes intact: inside single quotes a `"` is a
// literal character, so the JSON survives as ONE argv token. (The feature file's `\"`
// inner quotes must be unescaped to `"` BEFORE splitting — see runCommand.)
func splitArgsPOSIX(s string) []string {
	var args []string
	var cur strings.Builder
	inSingle, inDouble, hasToken := false, false, false
	flush := func() {
		if hasToken {
			args = append(args, cur.String())
			cur.Reset()
			hasToken = false
		}
	}
	for _, r := range s {
		switch {
		case r == '\'' && !inDouble:
			inSingle = !inSingle
			hasToken = true // an empty '' is still a (blank) token
		case r == '"' && !inSingle:
			inDouble = !inDouble
			hasToken = true
		case r == ' ' && !inSingle && !inDouble:
			flush()
		default:
			cur.WriteRune(r)
			hasToken = true
		}
	}
	flush()
	return args
}

// TestSplitArgsPOSIX_SingleQuotedInlineJSON pins the splitter's load-bearing
// behaviour: a single-quoted inline --changes array parses into exactly one token
// whose value is the literal JSON with its embedded double quotes preserved — proving
// the suite does single-quote grouping, not the naive double-quote toggle.
func TestSplitArgsPOSIX_SingleQuotedInlineJSON(t *testing.T) {
	raw := `proposal create ten_0123 --changes '[{"type":"CreateRole"}]'`
	args := splitArgsPOSIX(raw)
	want := []string{"proposal", "create", "ten_0123", "--changes", `[{"type":"CreateRole"}]`}
	if len(args) != len(want) {
		t.Fatalf("got %d args %q, want %d %q", len(args), args, len(want), want)
	}
	for i := range want {
		if args[i] != want[i] {
			t.Errorf("arg[%d] = %q, want %q", i, args[i], want[i])
		}
	}
}

// proposalCreationWorld is the per-scenario state: the connection context and fake
// transport assembled from the Given steps, the change-source bytes a file/stdin Given
// injects, plus the captured outcome/exit-code/streams of the When run. Everything is
// injected — no step touches the real network, env, home, pipe, or filesystem.
type proposalCreationWorld struct {
	ctx          apiclient.ConnectionContext
	transport    *tensionTransport
	changesBytes []byte // non-nil when a file/stdin Given injected the source; nil → inline
	wantChanges  string // the change array a file/stdin Given expects POSTed verbatim
	secret       string

	outcome  Outcome
	exitCode int
	stdout   string
	stderr   string
}

func initializeProposalCreationScenario(sc *godog.ScenarioContext) {
	w := &proposalCreationWorld{}
	sc.Before(func(ctx context.Context, _ *godog.Scenario) (context.Context, error) {
		*w = proposalCreationWorld{
			// A 201-created proposal is the default; per-scenario Given steps override
			// the transport/context (unknown tension, permission-denied, rate-limit, no
			// token) or inject a file/stdin change source.
			transport: &tensionTransport{status: 201, body: proposalCreatedBody},
			secret:    meSecretToken,
		}
		w.ctx = validMeContext()
		return ctx, nil
	})

	// --- Givens ---
	sc.Step(`^a complete connection context with a stored token$`, w.completeContext)
	sc.Step(`^the tension "([^"]*)" exists$`, w.tensionExists)
	sc.Step(`^no tension "([^"]*)" exists$`, w.tensionNotFound)
	sc.Step(`^the proposals endpoint answers the create with a permission-denied response$`, w.permissionDenied)
	sc.Step(`^the proposals endpoint answers the create with a rate-limit response$`, w.rateLimited)
	sc.Step(`^no usable token is available to the CLI$`, w.noToken)
	sc.Step(`^a file "([^"]*)" holding a JSON array of changes$`, w.fileSource)
	sc.Step(`^a JSON array of changes piped on standard input$`, w.stdinSource)

	// --- When ---
	sc.Step(`^an agent runs "glassfrog (.+)"$`, w.runCommand)

	// --- Thens ---
	sc.Step(`^the request will post the proposal to the proposals endpoint$`, w.requestPostedToProposalsEndpoint)
	sc.Step(`^the request body will carry the anchor "([^"]*)" and the changes array verbatim$`, w.bodyCarriesAnchorAndChanges)
	sc.Step(`^the request body will carry the changes read from the file verbatim$`, w.bodyCarriesInjectedChanges)
	sc.Step(`^the request body will carry the changes read from stdin verbatim$`, w.bodyCarriesInjectedChanges)
	sc.Step(`^the created proposal will be printed with its "([^"]*)" id and "([^"]*)" status$`, w.createdPrintedWithIDAndStatus)
	sc.Step(`^the created proposal will be printed$`, w.createdPrinted)
	sc.Step(`^the structured result will contain the created proposal's "([^"]*)" id and a "([^"]*)" status$`, w.structuredContainsIDAndStatus)
	sc.Step(`^the command will exit with code (\d+)$`, w.exitWithCode)
	sc.Step(`^the command will exit with a non-zero API-error code$`, w.exitNonZeroAPIError)
	sc.Step(`^the command will exit with the permission code$`, w.exitPermissionCode)
	sc.Step(`^the command will exit with the rate-limit code$`, w.exitRateLimitCode)
	sc.Step(`^stderr will report "([^"]*)" and point to "([^"]*)"$`, w.stderrReportsAndPointsTo)
	sc.Step(`^stderr will report that the create failed and name the HTTP status$`, w.stderrNamesHTTPStatus)
	sc.Step(`^stderr will report a usage error and name the rejected output value "([^"]*)"$`, w.stderrReportsRejectedOutput)
	sc.Step(`^stderr will report that "([^"]*)" is required$`, w.stderrReportsRequired)
	sc.Step(`^stderr will report that at least one change is required$`, w.stderrReportsAtLeastOneChange)
	sc.Step(`^stderr will report a usage error naming the change source$`, w.stderrReportsUsageNamingSource)
	sc.Step(`^stderr will report that every change must carry a "([^"]*)"$`, w.stderrReportsEveryChangeNeedsType)
	sc.Step(`^no request will be sent$`, w.noRequestSent)
	sc.Step(`^the rate-limit will be surfaced on the first occurrence$`, w.rateLimitSurfaced)
	sc.Step(`^the create will not be retried, so no duplicate proposal is created$`, w.notRetried)
}

// --- Given implementations ---

func (w *proposalCreationWorld) completeContext() error { w.ctx = validMeContext(); return nil }

func (w *proposalCreationWorld) tensionExists(_ string) error {
	w.transport = &tensionTransport{status: 201, body: proposalCreatedBody}
	return nil
}

func (w *proposalCreationWorld) tensionNotFound(_ string) error {
	w.transport = &tensionTransport{status: 404, body: `{"detail":"Tension not found"}`}
	return nil
}

func (w *proposalCreationWorld) permissionDenied() error {
	w.transport = &tensionTransport{status: 403, body: `{"detail":"async proposals not enabled"}`}
	return nil
}

func (w *proposalCreationWorld) rateLimited() error {
	w.transport = &tensionTransport{status: 429, body: `{"detail":"rate limited"}`}
	return nil
}

func (w *proposalCreationWorld) noToken() error {
	w.ctx = noTokenContext()
	return nil
}

func (w *proposalCreationWorld) fileSource(_ string) error {
	w.wantChanges = `[{"type":"CreateRole","name":"Scribe"}]`
	w.changesBytes = []byte(w.wantChanges)
	return nil
}

func (w *proposalCreationWorld) stdinSource() error {
	w.wantChanges = `[{"type":"CreateRole","name":"Scribe"}]`
	w.changesBytes = []byte(w.wantChanges)
	return nil
}

// --- When implementation ---

// runCommand parses the captured invocation with the suite's single-quote-aware
// splitArgsPOSIX (so a single-quoted inline `--changes '[{"type":...}]'` reaches cobra
// as ONE argv token), AFTER unescaping the feature file's \" inner quotes to ", and
// dispatches it through a real root with the `proposal` group attached over a fake
// seam. A file/stdin Given injects the change-source bytes; otherwise the seam reads
// the flag value as inline JSON. It asserts the secret token never leaks into output.
func (w *proposalCreationWorld) runCommand(invocation string) error {
	args := splitArgsPOSIX(strings.ReplaceAll(invocation, `\"`, `"`))
	root := NewRootCommand()
	seam := &fakeProposalSeam{
		fakeMeSeam:   &fakeMeSeam{ctx: w.ctx, transport: w.transport},
		changesBytes: w.changesBytes,
	}
	MustRegister(root, newProposalCommand(seam))

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

// --- Then implementations ---

func (w *proposalCreationWorld) requestPostedToProposalsEndpoint() error {
	if w.transport.calls != 1 {
		return fmt.Errorf("a create is exactly one request, got %d", w.transport.calls)
	}
	if w.transport.lastMethod != "POST" {
		return fmt.Errorf("the create should POST, got method %q", w.transport.lastMethod)
	}
	if !strings.HasSuffix(w.transport.lastPath, "/proposals") {
		return fmt.Errorf("the request should target /proposals, got %q", w.transport.lastPath)
	}
	if w.transport.lastContentType != "application/json" {
		return fmt.Errorf("the body should be sent as application/json, got %q", w.transport.lastContentType)
	}
	if w.transport.lastIfMatch != "" {
		return fmt.Errorf("a create must send NO If-Match, got %q", w.transport.lastIfMatch)
	}
	return nil
}

func (w *proposalCreationWorld) bodyCarriesAnchorAndChanges(anchorField string) error {
	if !strings.Contains(w.transport.lastBody, fmt.Sprintf(`"%s":"ten_0123"`, anchorField)) {
		return fmt.Errorf("the body should carry the anchor %q, got %s", anchorField, w.transport.lastBody)
	}
	if !strings.Contains(w.transport.lastBody, `"changes":[{"type":"CreateRole","name":"Scribe"}]`) {
		return fmt.Errorf("the body should carry the changes array verbatim, got %s", w.transport.lastBody)
	}
	return nil
}

func (w *proposalCreationWorld) bodyCarriesInjectedChanges() error {
	if w.wantChanges == "" {
		return fmt.Errorf("no injected change source was set up by the Given")
	}
	if !strings.Contains(w.transport.lastBody, `"changes":`+w.wantChanges) {
		return fmt.Errorf("the body should carry the injected changes %q verbatim, got %s", w.wantChanges, w.transport.lastBody)
	}
	return nil
}

func (w *proposalCreationWorld) createdPrintedWithIDAndStatus(idPrefix, status string) error {
	for _, want := range []string{idPrefix, status} {
		if !strings.Contains(w.stdout, want) {
			return fmt.Errorf("the created proposal should print %q:\n%s", want, w.stdout)
		}
	}
	return nil
}

func (w *proposalCreationWorld) createdPrinted() error {
	if !strings.Contains(w.stdout, "prp_") {
		return fmt.Errorf("the created proposal should be printed (its prp_ id present):\n%s", w.stdout)
	}
	return nil
}

func (w *proposalCreationWorld) structuredContainsIDAndStatus(idPrefix, status string) error {
	for _, want := range []string{idPrefix, status} {
		if !strings.Contains(w.stdout, want) {
			return fmt.Errorf("the structured result should contain %q:\n%s", want, w.stdout)
		}
	}
	// Structured output is the raw {data: …} payload, not the human projection.
	if strings.Contains(w.stdout, "Transitions:") {
		return fmt.Errorf("structured output must not render the human projection:\n%s", w.stdout)
	}
	return nil
}

func (w *proposalCreationWorld) exitWithCode(code int) error {
	if w.exitCode != code {
		return fmt.Errorf("exit code = %d, want %d (outcome %v)\nstderr: %s", w.exitCode, code, w.outcome, w.stderr)
	}
	return nil
}

func (w *proposalCreationWorld) exitNonZeroAPIError() error {
	if w.outcome != APIError || w.exitCode != 3 {
		return fmt.Errorf("outcome=%v exit=%d, want APIError/3\nstderr: %s", w.outcome, w.exitCode, w.stderr)
	}
	return nil
}

func (w *proposalCreationWorld) exitPermissionCode() error {
	if w.outcome != PermissionError || w.exitCode != 4 {
		return fmt.Errorf("outcome=%v exit=%d, want PermissionError/4\nstderr: %s", w.outcome, w.exitCode, w.stderr)
	}
	return nil
}

func (w *proposalCreationWorld) exitRateLimitCode() error {
	if w.outcome != RateLimited || w.exitCode != 5 {
		return fmt.Errorf("outcome=%v exit=%d, want RateLimited/5\nstderr: %s", w.outcome, w.exitCode, w.stderr)
	}
	return nil
}

func (w *proposalCreationWorld) stderrReportsAndPointsTo(report, pointer string) error {
	if !strings.Contains(strings.ToLower(w.stderr), strings.ToLower(report)) {
		return fmt.Errorf("stderr should report %q:\n%s", report, w.stderr)
	}
	if !strings.Contains(w.stderr, pointer) {
		return fmt.Errorf("stderr should point to %q:\n%s", pointer, w.stderr)
	}
	return nil
}

func (w *proposalCreationWorld) stderrNamesHTTPStatus() error {
	if w.transport.status == 0 {
		return fmt.Errorf("the scenario installed no status-bearing transport")
	}
	want := fmt.Sprintf("%d", w.transport.status)
	if !strings.Contains(w.stderr, want) {
		return fmt.Errorf("stderr should name the HTTP status (%s):\n%s", want, w.stderr)
	}
	return nil
}

func (w *proposalCreationWorld) stderrReportsRejectedOutput(value string) error {
	if w.outcome != UsageError || w.exitCode != 2 {
		return fmt.Errorf("outcome=%v exit=%d, want UsageError/2\nstderr: %s", w.outcome, w.exitCode, w.stderr)
	}
	if !strings.Contains(w.stderr, value) {
		return fmt.Errorf("stderr should name the rejected output value %q:\n%s", value, w.stderr)
	}
	return nil
}

func (w *proposalCreationWorld) stderrReportsRequired(name string) error {
	if !strings.Contains(w.stderr, name) {
		return fmt.Errorf("stderr should report that %q is required:\n%s", name, w.stderr)
	}
	return nil
}

func (w *proposalCreationWorld) stderrReportsAtLeastOneChange() error {
	if !strings.Contains(w.stderr, "at least one change") {
		return fmt.Errorf("stderr should report that at least one change is required:\n%s", w.stderr)
	}
	return nil
}

func (w *proposalCreationWorld) stderrReportsUsageNamingSource() error {
	if w.outcome != UsageError || w.exitCode != 2 {
		return fmt.Errorf("outcome=%v exit=%d, want UsageError/2\nstderr: %s", w.outcome, w.exitCode, w.stderr)
	}
	// The unparseable-inline message names the --changes source and the JSON-array
	// expectation.
	if !strings.Contains(w.stderr, "--changes") {
		return fmt.Errorf("stderr should name the change source (--changes):\n%s", w.stderr)
	}
	return nil
}

func (w *proposalCreationWorld) stderrReportsEveryChangeNeedsType(key string) error {
	if !strings.Contains(w.stderr, key) {
		return fmt.Errorf("stderr should report that every change must carry a %q:\n%s", key, w.stderr)
	}
	return nil
}

func (w *proposalCreationWorld) noRequestSent() error {
	if w.transport.calls != 0 {
		return fmt.Errorf("no request should be sent, but the transport was called %d times", w.transport.calls)
	}
	return nil
}

func (w *proposalCreationWorld) rateLimitSurfaced() error {
	if w.outcome != RateLimited {
		return fmt.Errorf("the rate-limit should be surfaced (outcome RateLimited), got %v\nstderr: %s", w.outcome, w.stderr)
	}
	if w.transport.calls < 1 {
		return fmt.Errorf("the rate-limit must be surfaced on the first occurrence (the request was sent), got %d calls", w.transport.calls)
	}
	return nil
}

func (w *proposalCreationWorld) notRetried() error {
	if w.transport.calls != 1 {
		return fmt.Errorf("a POST 429 must not be retried (no duplicate proposal), want exactly 1 call, got %d", w.transport.calls)
	}
	return nil
}
