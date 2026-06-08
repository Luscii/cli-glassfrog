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

// TestMeRolesFeatures runs the executable acceptance for My Roles (012): the
// `me roles` command driven through its seam over a fake base transport, so every
// scenario runs offline (no real network, no real ~/.glassfrogrc). Its Paths name
// ONLY this spec's feature file — never the features/ directory — so un-@wip-ping
// these scenarios cannot disturb another suite, and the suite reports its own
// independent scenario count (LEARNINGS: a suite points at its own feature file).
// The 2 @validation scenarios stay @wip (held for the validate skill) and are
// skipped by the ~@wip filter.
func TestMeRolesFeatures(t *testing.T) {
	suite := godog.TestSuite{
		ScenarioInitializer: initializeMeRolesScenario,
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"../../features/self-service-reads/my-roles.feature"},
			Tags:     "~@wip",
			TestingT: t,
		},
	}
	if suite.Run() != 0 {
		t.Fatal("non-zero status: my-roles feature scenarios failed")
	}
}

// meRolesWorld is the per-scenario state for the my-roles suite: the connection
// context and fake transport assembled from the Given steps, plus the captured
// outcome/exit-code/streams of the When run. Everything is injected — no step
// touches the real network, env, or home.
type meRolesWorld struct {
	ctx          apiclient.ConnectionContext
	newClientErr error
	transport    *cannedTransport
	secret       string

	outcome  Outcome
	exitCode int
	stdout   string
	stderr   string
}

// roleEssentialsBody is a single role "Marketing Lead" with a purpose, two
// domains, and three accountabilities — the fixture for the essentials scenario.
const roleEssentialsBody = `{
  "data": [
    {"id": "role_0123456789abcdef0123456789abcdef", "name": "Marketing Lead",
     "purpose": "A market that knows us",
     "domains": [{"description": "The marketing budget"}, {"description": "The brand guidelines"}],
     "accountabilities": [
       {"description": "Defining the quarterly campaign"},
       {"description": "Reporting reach to the circle"},
       {"description": "Maintaining the press list"}
     ],
     "fillers": [{"id": "per_x", "name": "Someone"}],
     "tags": ["secret-tag"],
     "is_core": true}
  ],
  "meta": {"pagination": {"per_page": 25, "has_next_page": false, "next_cursor": ""}}
}`

func initializeMeRolesScenario(sc *godog.ScenarioContext) {
	w := &meRolesWorld{}
	sc.Before(func(ctx context.Context, _ *godog.Scenario) (context.Context, error) {
		*w = meRolesWorld{
			// A multi-role 2xx body is the default; error and shape scenarios override
			// status/netErr/body in their Given steps.
			transport: &cannedTransport{status: 200, body: rolesBodyMulti},
			secret:    meSecretToken,
		}
		w.ctx = validMeContext()
		return ctx, nil
	})

	// --- Givens ---
	sc.Step(`^a complete connection context with a stored token$`, w.completeContext)
	sc.Step(`^the API would return the roles the practitioner fills$`, w.apiReturnsRoles)
	sc.Step(`^the API would return no roles for the practitioner$`, w.apiReturnsNoRoles)
	sc.Step(`^no usable token is available to the CLI$`, w.contextNoToken)
	sc.Step(`^the API could not be reached$`, w.apiUnreachable)
	sc.Step(`^the API would return a (\d+) response$`, w.apiAnswersStatus)
	sc.Step(`^a connection context carrying a base-URL configuration error$`, w.contextBaseURLError)
	sc.Step(`^the API would return a (\d+) response whose body cannot be parsed$`, w.apiReturnsUnparseable)
	sc.Step(`^the API would return a role named "([^"]*)" with a purpose, two domains, and three accountabilities$`, w.apiReturnsRoleEssentials)
	sc.Step(`^the API would return a first page reporting that more roles exist$`, w.apiReturnsFirstPageHasNext)

	// --- Whens ---
	sc.Step(`^the practitioner runs "glassfrog (.+)"$`, w.runCommand)

	// --- Thens ---
	sc.Step(`^each role will be printed as a projection$`, w.eachRolePrinted)
	sc.Step(`^the command will exit with code (\d+)$`, w.exitWithCode)
	sc.Step(`^"([^"]*)" will be printed to stdout$`, w.literalPrintedToStdout)
	sc.Step(`^stderr will report "([^"]*)" and point to "([^"]*)"$`, w.stderrReportsAndPointsTo)
	sc.Step(`^no role data will be printed$`, w.noRoleDataPrinted)
	sc.Step(`^the transport failure will be named on stderr$`, w.transportFailureNamed)
	sc.Step(`^stderr will report that the read failed and name the (\d+) status$`, w.stderrNamesStatus)
	sc.Step(`^the invocation will be rejected as a usage error$`, w.rejectedAsUsageError)
	sc.Step(`^no request will reach the API$`, w.noRequestReachedAPI)
	sc.Step(`^stderr will name the malformed base URL and its source$`, w.stderrNamesBaseURL)
	sc.Step(`^stderr will report that the response could not be parsed$`, w.stderrReportsDecodeFailure)
	sc.Step(`^the projection will show the role name, its identifier, its purpose, its domains, then its accountabilities$`, w.projectionShowsEssentials)
	sc.Step(`^the role's fillers, tags, and classification flags will not be shown$`, w.projectionHidesNonEssentials)
	sc.Step(`^the roles from the response will be printed to stdout$`, w.rolesPrintedToStdout)
	sc.Step(`^an incomplete-result note will be written to stderr$`, w.incompleteNoteOnStderr)
}

// --- Given implementations ---

func (w *meRolesWorld) completeContext() error { w.ctx = validMeContext(); return nil }

func (w *meRolesWorld) apiReturnsRoles() error {
	w.transport = &cannedTransport{status: 200, body: rolesBodyMulti}
	return nil
}

func (w *meRolesWorld) apiReturnsNoRoles() error {
	w.transport = &cannedTransport{status: 200, body: rolesBodyEmpty}
	return nil
}

func (w *meRolesWorld) contextNoToken() error {
	w.ctx = apiclient.ConnectionContext{
		BaseURL: apiclient.BaseURL{Value: "https://example.test/api/v5", Source: apiclient.SourceFlag},
		Cred:    auth.Resolution{Source: auth.SourceNone},
	}
	return nil
}

func (w *meRolesWorld) apiUnreachable() error {
	w.transport = &cannedTransport{netErr: errors.New("dial tcp: connection refused")}
	return nil
}

func (w *meRolesWorld) apiAnswersStatus(status int) error {
	w.transport = &cannedTransport{status: status, body: `{"error":"forbidden"}`}
	return nil
}

func (w *meRolesWorld) contextBaseURLError() error {
	w.ctx = apiclient.ConnectionContext{}
	w.newClientErr = &apiclient.BaseURLError{Source: "--" + apiclient.FlagBaseURL}
	return nil
}

func (w *meRolesWorld) apiReturnsUnparseable(_ int) error {
	w.transport = &cannedTransport{status: 200, body: `this is not the roles shape`}
	return nil
}

func (w *meRolesWorld) apiReturnsRoleEssentials(_ string) error {
	w.transport = &cannedTransport{status: 200, body: roleEssentialsBody}
	return nil
}

func (w *meRolesWorld) apiReturnsFirstPageHasNext() error {
	w.transport = &cannedTransport{status: 200, body: rolesBodyHasNext}
	return nil
}

// --- When implementation ---

// runCommand parses the captured "me roles …" invocation and dispatches it
// through a real root with `me` (runnable) and the `roles` leaf attached, over a
// fake seam. It asserts the secret token never leaks into any produced output.
func (w *meRolesWorld) runCommand(invocation string) error {
	root := NewRootCommand()
	seam := &fakeMeSeam{ctx: w.ctx, newClientErr: w.newClientErr, transport: w.transport}
	meCmd := newMeCommand(seam)
	MustRegister(root, meCmd)
	MustRegister(meCmd, newMeRolesCommand(seam))

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

// --- Then implementations ---

func (w *meRolesWorld) eachRolePrinted() error {
	// The default multi-role body carries Marketing Lead and Treasurer, each as a
	// projection block with its id and the section headers.
	for _, want := range []string{"Marketing Lead (role_", "Treasurer", "Purpose:", "Domains:", "Accountabilities:"} {
		if !strings.Contains(w.stdout, want) {
			return fmt.Errorf("each role should be printed as a projection, missing %q:\n%s", want, w.stdout)
		}
	}
	return nil
}

func (w *meRolesWorld) exitWithCode(code int) error {
	if w.exitCode != code {
		return fmt.Errorf("exit code = %d, want %d (outcome %v)\nstderr: %s", w.exitCode, code, w.outcome, w.stderr)
	}
	return nil
}

func (w *meRolesWorld) literalPrintedToStdout(literal string) error {
	if !strings.Contains(w.stdout, literal) {
		return fmt.Errorf("stdout should contain %q:\n%s", literal, w.stdout)
	}
	return nil
}

func (w *meRolesWorld) stderrReportsAndPointsTo(report, pointer string) error {
	// "not authenticated" is matched leniently (the message names the cause); the
	// pointer ("glassfrog auth login") must appear verbatim.
	if !strings.Contains(strings.ToLower(w.stderr), strings.ToLower(report)) {
		return fmt.Errorf("stderr should report %q:\n%s", report, w.stderr)
	}
	if !strings.Contains(w.stderr, pointer) {
		return fmt.Errorf("stderr should point to %q:\n%s", pointer, w.stderr)
	}
	return nil
}

func (w *meRolesWorld) noRoleDataPrinted() error {
	if strings.TrimSpace(w.stdout) != "" {
		return fmt.Errorf("no role data should be printed, got stdout:\n%s", w.stdout)
	}
	return nil
}

func (w *meRolesWorld) transportFailureNamed() error {
	if w.outcome != NetworkUnavailable {
		return fmt.Errorf("outcome = %v, want NetworkUnavailable", w.outcome)
	}
	if strings.TrimSpace(w.stderr) == "" {
		return errors.New("a transport failure should be named on stderr")
	}
	return nil
}

func (w *meRolesWorld) stderrNamesStatus(status int) error {
	if w.outcome != APIError {
		return fmt.Errorf("outcome = %v, want APIError", w.outcome)
	}
	if !strings.Contains(w.stderr, fmt.Sprintf("%d", status)) {
		return fmt.Errorf("stderr should name the %d status:\n%s", status, w.stderr)
	}
	return nil
}

func (w *meRolesWorld) rejectedAsUsageError() error {
	if w.outcome != UsageError || w.exitCode != 2 {
		return fmt.Errorf("outcome=%v exit=%d, want UsageError/2", w.outcome, w.exitCode)
	}
	return nil
}

func (w *meRolesWorld) noRequestReachedAPI() error {
	if w.transport.calls != 0 {
		return fmt.Errorf("no request should reach the API, but the transport was called %d times", w.transport.calls)
	}
	return nil
}

func (w *meRolesWorld) stderrNamesBaseURL() error {
	if w.outcome != UsageError {
		return fmt.Errorf("outcome = %v, want UsageError", w.outcome)
	}
	if !strings.Contains(strings.ToLower(w.stderr), "base url") && !strings.Contains(w.stderr, "base-url") {
		return fmt.Errorf("stderr should name the malformed base URL:\n%s", w.stderr)
	}
	if !strings.Contains(w.stderr, "--"+apiclient.FlagBaseURL) {
		return fmt.Errorf("stderr should name the source (--%s):\n%s", apiclient.FlagBaseURL, w.stderr)
	}
	return nil
}

func (w *meRolesWorld) stderrReportsDecodeFailure() error {
	if w.outcome != RuntimeError {
		return fmt.Errorf("outcome = %v, want RuntimeError", w.outcome)
	}
	if strings.TrimSpace(w.stderr) == "" {
		return errors.New("a decode failure should be reported on stderr")
	}
	return nil
}

func (w *meRolesWorld) projectionShowsEssentials() error {
	for _, want := range []string{
		"Marketing Lead", "role_0123456789abcdef0123456789abcdef",
		"Purpose: A market that knows us",
		"Domains:", "The marketing budget", "The brand guidelines",
		"Accountabilities:", "Defining the quarterly campaign", "Maintaining the press list",
	} {
		if !strings.Contains(w.stdout, want) {
			return fmt.Errorf("the projection should show %q:\n%s", want, w.stdout)
		}
	}
	// Order: name → purpose → domains → accountabilities.
	iName := strings.Index(w.stdout, "Marketing Lead")
	iPurpose := strings.Index(w.stdout, "Purpose:")
	iDomains := strings.Index(w.stdout, "Domains:")
	iAcc := strings.Index(w.stdout, "Accountabilities:")
	if !(iName < iPurpose && iPurpose < iDomains && iDomains < iAcc) {
		return fmt.Errorf("the projection should be ordered name → purpose → domains → accountabilities:\n%s", w.stdout)
	}
	return nil
}

func (w *meRolesWorld) projectionHidesNonEssentials() error {
	// The essentials fixture carries a filler name, a tag, and a flag the
	// projection must never surface.
	for _, forbidden := range []string{"Someone", "secret-tag", "is_core", "filler", "Filler", "tag", "Tag", "flag", "Flag"} {
		if strings.Contains(w.stdout, forbidden) {
			return fmt.Errorf("the projection must not surface %q:\n%s", forbidden, w.stdout)
		}
	}
	return nil
}

func (w *meRolesWorld) rolesPrintedToStdout() error {
	if !strings.Contains(w.stdout, "Marketing Lead") {
		return fmt.Errorf("the roles from the response should print to stdout:\n%s", w.stdout)
	}
	return nil
}

func (w *meRolesWorld) incompleteNoteOnStderr() error {
	// interface-cli pins the note text verbatim; compare the full stderr line
	// against the constant (trailing newline trimmed) so wording drift fails the
	// acceptance, not just an absent substring.
	if strings.TrimRight(w.stderr, "\n") != incompleteRolesNote {
		return fmt.Errorf("stderr should be exactly the pinned incompleteness note %q, got %q", incompleteRolesNote, w.stderr)
	}
	if strings.Contains(w.stdout, "incomplete") {
		return fmt.Errorf("the incompleteness note must not appear on stdout:\n%s", w.stdout)
	}
	return nil
}
