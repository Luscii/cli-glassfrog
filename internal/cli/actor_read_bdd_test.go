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

// TestActorReadFeatures runs the executable acceptance for Actor Read (049): the
// single-actor `actors <id>` read driven through its seam over a fake base
// transport, so every scenario runs offline (no real network, no real
// ~/.glassfrogrc). Its Paths name ONLY this spec's feature file — never the
// features/ directory — so the suite reports its own independent scenario count and
// un-@wip-ping these scenarios cannot disturb another suite (LEARNINGS: a suite
// points at its own feature file). The @validation scenarios stay @wip (held for
// the validate skill) and are skipped by the ~@wip filter.
func TestActorReadFeatures(t *testing.T) {
	suite := godog.TestSuite{
		ScenarioInitializer: initializeActorReadScenario,
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"../../features/actors-disconnected-from-governance/actor-read.feature"},
			Tags:     "~@wip",
			TestingT: t,
		},
	}
	if suite.Run() != 0 {
		t.Fatal("non-zero status: actor-read feature scenarios failed")
	}
}

// --- canned GET /actors/{id} {data: ActorDetail} bodies (id-matched to the
// feature's per_abc / agt_def invocations) ----------------------------------

const actorReadPerAbcRoles = `{"data":{"id":"per_abc","name":"Alice Smith","kind":"human",
  "roles":[{"id":"role_x","type":"role","name":"Marketing Lead","purpose":"A market that knows us",
    "domains":[{"id":"dom_1","description":"The marketing budget"}],
    "accountabilities":[{"id":"acc_1","description":"Defining the campaign"}]}]}}`

const actorReadPerAbcAssignments = `{"data":{"id":"per_abc","name":"Alice Smith","kind":"human",
  "assignments":[{"id":"asgn_1","actor_id":"per_abc","role_id":"role_x","focus":"Campaigns"}]}}`

const actorReadAgtDef = `{"data":{"id":"agt_def","name":"Claude","kind":"agent"}}`

// actorReadWorld is the per-scenario state: the connection context and fake
// transport assembled from the Given steps, plus the captured
// outcome/exit-code/streams. Everything is injected — no step touches the real
// network, env, or home.
type actorReadWorld struct {
	ctx       apiclient.ConnectionContext
	transport *cannedTransport
	secret    string

	outcome  Outcome
	exitCode int
	stdout   string
	stderr   string
}

func initializeActorReadScenario(sc *godog.ScenarioContext) {
	w := &actorReadWorld{}
	sc.Before(func(ctx context.Context, _ *godog.Scenario) (context.Context, error) {
		*w = actorReadWorld{
			// A bare single-actor document is the default; per-scenario Given steps
			// override the transport/context as needed.
			transport: &cannedTransport{status: 200, body: actorReadPerAbcRoles},
			secret:    meSecretToken,
		}
		w.ctx = validMeContext()
		return ctx, nil
	})

	// --- Givens ---
	sc.Step(`^a complete connection context with a stored token$`, w.completeContext)
	sc.Step(`^an actor that exists in the organization$`, w.actorExists)
	sc.Step(`^an actor who fills several roles$`, w.actorWithAssignments)
	sc.Step(`^an agent that exists in the organization$`, w.agentExists)
	sc.Step(`^a token without the ai_integration feature$`, w.tokenWithoutAIIntegration)
	sc.Step(`^the organization has several actors$`, w.severalActors)
	sc.Step(`^no usable token is available to the CLI$`, w.noToken)
	sc.Step(`^the API answers the actor read with a 404$`, w.apiAnswers404)

	// --- Whens ---
	sc.Step(`^an agent runs "glassfrog (.+)"$`, w.runCommand)

	// --- Thens ---
	sc.Step(`^the request will carry "([^"]*)" set to "([^"]*)"$`, w.requestCarriesParam)
	sc.Step(`^the request will read "([^"]*)"$`, w.requestReadsPath)
	sc.Step(`^the request will read the unified "([^"]*)" endpoint$`, w.requestReadsPath)
	sc.Step(`^it will not route through the ai_integration-gated "([^"]*)" alias$`, w.requestNotThroughAlias)
	sc.Step(`^the actor will be printed with each role's name, purpose, accountabilities, and domains$`, w.footprintPrinted)
	sc.Step(`^the actor will be printed with its assignments embedded$`, w.assignmentsPrinted)
	sc.Step(`^the actor's id, name, and kind will be printed$`, w.identityPrinted)
	sc.Step(`^the agent will be printed as a single actor$`, w.agentPrinted)
	sc.Step(`^every actor will be printed as a list$`, w.everyActorPrinted)
	sc.Step(`^stderr will report the unsupported value and list the supported set$`, w.stderrNamesUnsupportedInclude)
	sc.Step(`^stderr will report that --include applies only to a single actor read$`, w.stderrIncludeSingleOnly)
	sc.Step(`^stderr will report that the filter applies only to the directory list$`, w.stderrFilterListOnly)
	sc.Step(`^stderr will report "([^"]*)" and point to "([^"]*)"$`, w.stderrReportsAndPointsTo)
	sc.Step(`^stderr will report that the read failed and name the HTTP status$`, w.stderrNamesHTTPStatus)
	sc.Step(`^no actor will be printed$`, w.noActorPrinted)
	sc.Step(`^no API request will be sent$`, w.noRequestSent)
	sc.Step(`^the command will exit with code (\d+)$`, w.exitWithCode)
	sc.Step(`^the command will exit with a non-zero API-error code$`, w.exitNonZeroAPIError)
}

// --- Given implementations ---

func (w *actorReadWorld) completeContext() error { w.ctx = validMeContext(); return nil }

func (w *actorReadWorld) actorExists() error {
	w.transport = &cannedTransport{status: 200, body: actorReadPerAbcRoles}
	return nil
}

func (w *actorReadWorld) actorWithAssignments() error {
	w.transport = &cannedTransport{status: 200, body: actorReadPerAbcAssignments}
	return nil
}

func (w *actorReadWorld) agentExists() error {
	w.transport = &cannedTransport{status: 200, body: actorReadAgtDef}
	return nil
}

func (w *actorReadWorld) tokenWithoutAIIntegration() error {
	// The CLI never gates on a feature flag: an agt_ id reads the ungated unified
	// endpoint with any valid token. The fake carries the standard usable context.
	w.ctx = validMeContext()
	w.transport = &cannedTransport{status: 200, body: actorReadAgtDef}
	return nil
}

func (w *actorReadWorld) severalActors() error {
	w.transport = &cannedTransport{status: 200, body: actorsPageComplete}
	return nil
}

func (w *actorReadWorld) noToken() error {
	w.ctx = noTokenContext()
	return nil
}

func (w *actorReadWorld) apiAnswers404() error {
	w.transport = &cannedTransport{status: 404, body: `{"detail":"Actor not found"}`}
	return nil
}

// --- When implementation ---

// runCommand parses the captured invocation (quote-aware, reusing the search
// suite's splitArgs) and dispatches it through a real root with only the `actors`
// leaf attached over a fake seam. It asserts the token never leaks into either
// stream.
func (w *actorReadWorld) runCommand(invocation string) error {
	args := splitArgs(invocation)
	root := NewRootCommand()
	seam := &fakeMeSeam{ctx: w.ctx, transport: w.transport}
	MustRegister(root, newActorsCommand(seam))

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

func (w *actorReadWorld) requestCarriesParam(param, value string) error {
	if got := w.transport.lastQuery.Get(param); got != value {
		return fmt.Errorf("request %s = %q, want %q", param, got, value)
	}
	return nil
}

func (w *actorReadWorld) requestReadsPath(path string) error {
	if !strings.HasSuffix(w.transport.lastPath, path) {
		return fmt.Errorf("request path = %q, want it to read %q", w.transport.lastPath, path)
	}
	return nil
}

func (w *actorReadWorld) requestNotThroughAlias(alias string) error {
	if strings.Contains(w.transport.lastPath, alias) {
		return fmt.Errorf("the request must not route through the %q alias, got %q", alias, w.transport.lastPath)
	}
	return nil
}

func (w *actorReadWorld) footprintPrinted() error {
	for _, want := range []string{"Roles:", "Marketing Lead (role_x)", "A market that knows us", "The marketing budget", "Defining the campaign"} {
		if !strings.Contains(w.stdout, want) {
			return fmt.Errorf("the governance footprint should print; missing %q:\n%s", want, w.stdout)
		}
	}
	return nil
}

func (w *actorReadWorld) assignmentsPrinted() error {
	for _, want := range []string{"Assignments:", "role_x", "Campaigns"} {
		if !strings.Contains(w.stdout, want) {
			return fmt.Errorf("the assignments should embed; missing %q:\n%s", want, w.stdout)
		}
	}
	return nil
}

func (w *actorReadWorld) identityPrinted() error {
	for _, want := range []string{"per_abc", "[human]", "Alice Smith"} {
		if !strings.Contains(w.stdout, want) {
			return fmt.Errorf("the actor's id, name, and kind should print; missing %q:\n%s", want, w.stdout)
		}
	}
	return nil
}

func (w *actorReadWorld) agentPrinted() error {
	for _, want := range []string{"agt_def", "[agent]", "Claude"} {
		if !strings.Contains(w.stdout, want) {
			return fmt.Errorf("the agent should print as a single actor; missing %q:\n%s", want, w.stdout)
		}
	}
	return nil
}

func (w *actorReadWorld) everyActorPrinted() error {
	for _, want := range []string{"per_0123", "agt_0456"} {
		if !strings.Contains(w.stdout, want) {
			return fmt.Errorf("every actor should print as a list; missing %q:\n%s", want, w.stdout)
		}
	}
	return nil
}

func (w *actorReadWorld) stderrNamesUnsupportedInclude() error {
	if !strings.Contains(w.stderr, "nonsense") {
		return fmt.Errorf("stderr should name the unsupported value:\n%s", w.stderr)
	}
	for _, want := range []string{"assignments", "roles"} {
		if !strings.Contains(w.stderr, want) {
			return fmt.Errorf("stderr should list the supported set; missing %q:\n%s", want, w.stderr)
		}
	}
	return nil
}

func (w *actorReadWorld) stderrIncludeSingleOnly() error {
	if !strings.Contains(w.stderr, "--include") {
		return fmt.Errorf("stderr should report that --include applies only to a single actor read:\n%s", w.stderr)
	}
	return nil
}

func (w *actorReadWorld) stderrFilterListOnly() error {
	if !strings.Contains(w.stderr, "--kind") || !strings.Contains(w.stderr, "directory list") {
		return fmt.Errorf("stderr should report that the filter applies only to the directory list:\n%s", w.stderr)
	}
	return nil
}

func (w *actorReadWorld) stderrReportsAndPointsTo(report, pointer string) error {
	if !strings.Contains(strings.ToLower(w.stderr), strings.ToLower(report)) {
		return fmt.Errorf("stderr should report %q:\n%s", report, w.stderr)
	}
	if !strings.Contains(w.stderr, pointer) {
		return fmt.Errorf("stderr should point to %q:\n%s", pointer, w.stderr)
	}
	return nil
}

func (w *actorReadWorld) stderrNamesHTTPStatus() error {
	if !strings.Contains(w.stderr, "404") {
		return fmt.Errorf("stderr should report the read failed and name the HTTP status (404):\n%s", w.stderr)
	}
	return nil
}

func (w *actorReadWorld) noActorPrinted() error {
	if strings.Contains(w.stdout, "per_") || strings.Contains(w.stdout, "agt_") {
		return fmt.Errorf("no actor should be printed, got stdout:\n%s", w.stdout)
	}
	return nil
}

func (w *actorReadWorld) noRequestSent() error {
	if w.transport.calls != 0 {
		return fmt.Errorf("no API request should be sent, but the transport was called %d times", w.transport.calls)
	}
	return nil
}

func (w *actorReadWorld) exitWithCode(code int) error {
	if w.exitCode != code {
		return fmt.Errorf("exit code = %d, want %d (outcome %v)\nstderr: %s", w.exitCode, code, w.outcome, w.stderr)
	}
	return nil
}

func (w *actorReadWorld) exitNonZeroAPIError() error {
	if w.outcome != APIError || w.exitCode != 3 {
		return fmt.Errorf("outcome=%v exit=%d, want APIError/3\nstderr: %s", w.outcome, w.exitCode, w.stderr)
	}
	return nil
}
