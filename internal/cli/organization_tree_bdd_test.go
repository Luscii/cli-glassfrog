package cli

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"testing"

	"github.com/Luscii/cli-glassfrog/internal/apiclient"
	"github.com/cucumber/godog"
)

// TestOrganizationTreeFeatures runs the executable acceptance for Organization
// Tree (026): the `tree` (whole-org + rooted) and `subroles` commands driven
// through their seams over a fake base transport, so every scenario runs offline
// (no real network, no real ~/.glassfrogrc). Its Paths name ONLY this spec's
// feature file — never the features/ directory — so the suite reports its own
// independent scenario count and un-@wip-ping these scenarios cannot disturb
// another suite (LEARNINGS: a suite points at its own feature file). The
// @validation scenarios stay @wip (held for the validate skill) and are skipped
// by the ~@wip filter.
func TestOrganizationTreeFeatures(t *testing.T) {
	suite := godog.TestSuite{
		ScenarioInitializer: initializeOrgTreeScenario,
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"../../features/governance-reads/organization-tree.feature"},
			Tags:     "~@wip",
			TestingT: t,
		},
	}
	if suite.Run() != 0 {
		t.Fatal("non-zero status: organization-tree feature scenarios failed")
	}
}

// orgTreeWorld is the per-scenario state: the connection context and fake
// transport assembled from the Given steps, plus the captured
// outcome/exit-code/streams of the When run. Everything is injected — no step
// touches the real network, env, or home.
type orgTreeWorld struct {
	ctx       apiclient.ConnectionContext
	transport http.RoundTripper
	secret    string

	outcome  Outcome
	exitCode int
	stdout   string
	stderr   string
}

// --- BDD-local fixtures (rich bodies the Then steps assert against) ---------

// bddRootedTreeWithIncludes is a rooted subtree (root + one child) carrying
// accountabilities/domains/fillers on each node, so the same Given serves both
// the plain-subtree scenario (sections gated off without --include) and the
// per-node-include scenario (sections rendered with --include).
const bddRootedTreeWithIncludes = `{
  "data": {
    "id": "role_0123", "type": "circle", "name": "Marketing", "purpose": "Markets that know us",
    "parent_role_id": "role_anchor", "has_subroles": true, "flags": ["structural"],
    "accountabilities": [{"id": "acc_1", "description": "Defining the campaign"}],
    "domains": [{"id": "dom_1", "description": "The marketing budget"}],
    "fillers": [{"id": "per_x", "name": "Alice Smith", "kind": "human"}],
    "children": [
      {"id": "role_press", "type": "role", "name": "Press Officer", "purpose": "p",
       "parent_role_id": "role_0123", "has_subroles": false, "flags": [],
       "accountabilities": [{"id": "acc_2", "description": "Holding the press list"}],
       "domains": [{"id": "dom_2", "description": "The newsroom"}],
       "children": []}
    ]
  }
}`

// bddSubrolesWithIncludes is a single page of two children carrying assignments
// AND policies, so the per-child-include scenario asserts both render inline.
const bddSubrolesWithIncludes = `{
  "data": [
    {"id": "role_press", "type": "role", "name": "Press Officer", "purpose": "p",
     "parent_role_id": "role_0123", "has_subroles": false, "flags": [],
     "assignments": [{"id": "asgn_1", "actor_id": "per_x", "role_id": "role_press", "actor": {"id": "per_x", "name": "Alice Smith", "kind": "human"}}],
     "policies": [{"id": "pol_1", "title": "All PRs require two approvals", "body": "b"}]},
    {"id": "role_events", "type": "role", "name": "Events Lead", "purpose": "p",
     "parent_role_id": "role_0123", "has_subroles": false, "flags": [],
     "assignments": [{"id": "asgn_2", "actor_id": "per_y", "role_id": "role_events", "actor": {"id": "per_y", "name": "Bob Jones", "kind": "human"}}],
     "policies": [{"id": "pol_2", "title": "Events need sign-off", "body": "b"}]}
  ],
  "meta": {"pagination": {"per_page": 100, "has_next_page": false, "next_cursor": ""}}
}`

func initializeOrgTreeScenario(sc *godog.ScenarioContext) {
	w := &orgTreeWorld{}
	sc.Before(func(ctx context.Context, _ *godog.Scenario) (context.Context, error) {
		*w = orgTreeWorld{
			transport: &cannedTransport{status: 200, body: orgTreeBody},
			secret:    meSecretToken,
		}
		w.ctx = validMeContext()
		return ctx, nil
	})

	// --- Givens ---
	sc.Step(`^a complete connection context with a stored token$`, w.completeContext)
	sc.Step(`^no usable token is available to the CLI$`, w.contextNoToken)
	sc.Step(`^the API would return a nested role tree for the organization$`, w.apiReturnsOrgTree)
	sc.Step(`^a role "([^"]*)" exists in the organization$`, w.apiReturnsRootedTree)
	sc.Step(`^a role "([^"]*)" exists with no child roles$`, w.apiReturnsLeafTree)
	sc.Step(`^no role "([^"]*)" exists in the organization$`, w.apiReturnsNotFound)
	sc.Step(`^the circle role "([^"]*)" has several child roles$`, w.apiReturnsSubrolesWithIncludes)
	sc.Step(`^the role "([^"]*)" has no child roles$`, w.apiReturnsEmptySubroles)
	sc.Step(`^the subroles of "([^"]*)" span more than one page$`, w.apiReturnsSubrolesFirstOfMany)
	sc.Step(`^the subroles of "([^"]*)" span three pages of API responses$`, w.apiReturnsSubrolesThreePages)
	sc.Step(`^the subroles walk for "([^"]*)" fails after retrieving the first page$`, w.apiSubrolesFailsMidWalk)
	sc.Step(`^the organization has a deep role hierarchy$`, w.apiReturnsCappedTree)
	sc.Step(`^the organization is deeper than one level$`, w.apiReturnsCappedTree)
	sc.Step(`^a direct child "([^"]*)" itself contains subroles$`, w.noopGiven)

	// --- Whens ---
	sc.Step(`^an agent runs "glassfrog (.+)"$`, w.runCommand)

	// --- Thens (shared exit/stderr phrasings) ---
	sc.Step(`^the command will exit with code (\d+)$`, w.exitWithCode)
	sc.Step(`^the command will exit with a non-zero API-error code$`, w.exitNonZeroAPIError)
	sc.Step(`^the command will exit with a non-zero code$`, w.exitNonZero)
	sc.Step(`^stderr will report "([^"]*)" and point to "([^"]*)"$`, w.stderrReportsAndPointsTo)
	sc.Step(`^stderr will report that the read failed and name the HTTP status$`, w.stderrNamesHTTPStatus)
	sc.Step(`^stderr will report a usage error$`, w.stderrReportsUsageError)
	sc.Step(`^no API request will be sent$`, w.noRequestSent)
	sc.Step(`^the request will carry "([^"]*)"$`, w.requestCarriesRawParam)
	sc.Step(`^"([^"]*)" will be printed to stdout$`, w.literalPrintedToStdout)

	// --- Thens (tree) ---
	sc.Step(`^the tree will be printed as a nested projection rooted at the anchor role$`, w.nestedTreePrinted)
	sc.Step(`^no tree data will be printed$`, w.noTreeDataPrinted)
	sc.Step(`^the tree will be printed rooted at and including "([^"]*)"$`, w.treeRootedAt)
	sc.Step(`^each node will carry its accountabilities and domains inline$`, w.nodesCarryIncludes)
	sc.Step(`^stderr will name the unsupported value and the tree read's supported set$`, w.stderrNamesTreeIncludeSet)
	sc.Step(`^a single-node tree with no children will be printed$`, w.singleNodeTreePrinted)
	sc.Step(`^the anchor role and only its direct children will be printed$`, w.anchorAndDirectChildrenPrinted)
	sc.Step(`^"([^"]*)" will be marked as having subroles below the returned tree$`, w.nodeMarkedBelowDepth)
	sc.Step(`^a true leaf child will be marked as having none$`, w.trueLeafNotMarked)
	sc.Step(`^no invented count of omitted descendants will be printed$`, w.noInventedCount)

	// --- Thens (subroles) ---
	sc.Step(`^each direct child role will be printed as a projection$`, w.eachChildPrinted)
	sc.Step(`^the assignments and policies will be printed inline on each child role$`, w.assignmentsAndPoliciesInline)
	sc.Step(`^only the first page of child roles will be printed$`, w.onlyFirstPageChildrenPrinted)
	sc.Step(`^stderr will note that more subroles exist$`, w.stderrNotesMoreSubroles)
	sc.Step(`^the command will walk every page to completion$`, w.walkedEveryPage)
	sc.Step(`^all child roles across the pages will be printed$`, w.allPagesChildrenPrinted)
	sc.Step(`^the child roles retrieved so far will be printed$`, w.partialChildrenPrinted)
	sc.Step(`^stderr will note the result is incomplete and name the cause$`, w.stderrNotesIncomplete)
}

// --- Given implementations ---

func (w *orgTreeWorld) completeContext() error   { w.ctx = validMeContext(); return nil }
func (w *orgTreeWorld) contextNoToken() error    { w.ctx = noTokenContext(); return nil }
func (w *orgTreeWorld) noopGiven(_ string) error { return nil }

func (w *orgTreeWorld) apiReturnsOrgTree() error {
	w.transport = &cannedTransport{status: 200, body: orgTreeBody}
	return nil
}

func (w *orgTreeWorld) apiReturnsRootedTree(_ string) error {
	w.transport = &cannedTransport{status: 200, body: bddRootedTreeWithIncludes}
	return nil
}

func (w *orgTreeWorld) apiReturnsLeafTree(_ string) error {
	w.transport = &cannedTransport{status: 200, body: leafTreeBody}
	return nil
}

func (w *orgTreeWorld) apiReturnsNotFound(_ string) error {
	w.transport = &cannedTransport{status: 404, body: `{"detail":"Role not found"}`}
	return nil
}

func (w *orgTreeWorld) apiReturnsSubrolesWithIncludes(_ string) error {
	w.transport = &cannedTransport{status: 200, body: bddSubrolesWithIncludes}
	return nil
}

func (w *orgTreeWorld) apiReturnsEmptySubroles(_ string) error {
	w.transport = &cannedTransport{status: 200, body: subrolesPageEmpty}
	return nil
}

func (w *orgTreeWorld) apiReturnsSubrolesFirstOfMany(_ string) error {
	w.transport = &cannedTransport{status: 200, body: subrolesPage("role_p1", "First Page Child", "c1")}
	return nil
}

func (w *orgTreeWorld) apiReturnsSubrolesThreePages(_ string) error {
	w.transport = &seqMeTransport{steps: []seqMeResp{
		{status: 200, body: subrolesPage("role_p1", "Page One Child", "c1")},
		{status: 200, body: subrolesPage("role_p2", "Page Two Child", "c2")},
		{status: 200, body: subrolesPage("role_p3", "Page Three Child", "")},
	}}
	return nil
}

func (w *orgTreeWorld) apiSubrolesFailsMidWalk(_ string) error {
	w.transport = &seqMeTransport{steps: []seqMeResp{
		{status: 200, body: subrolesPage("role_p1", "Gathered Child", "c1")},
		{status: 500, body: `{"detail":"boom"}`},
	}}
	return nil
}

func (w *orgTreeWorld) apiReturnsCappedTree() error {
	w.transport = &cannedTransport{status: 200, body: cappedTreeBody}
	return nil
}

// --- When implementation ---

// runCommand dispatches the captured invocation through a real root with the
// `tree` and `subroles` leaves attached over a fake seam, asserting the secret
// token never leaks into output.
func (w *orgTreeWorld) runCommand(invocation string) error {
	root := NewRootCommand()
	seam := &fakeMeSeam{ctx: w.ctx, transport: w.transport}
	MustRegister(root, newTreeCommand(seam))
	MustRegister(root, newSubrolesCommand(seam))

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

func (w *orgTreeWorld) transportCalls() int {
	switch t := w.transport.(type) {
	case *cannedTransport:
		return t.calls
	case *seqMeTransport:
		return t.calls
	default:
		return -1
	}
}

// --- Then implementations (shared) ---

func (w *orgTreeWorld) exitWithCode(code int) error {
	if w.exitCode != code {
		return fmt.Errorf("exit code = %d, want %d (outcome %v)\nstderr: %s", w.exitCode, code, w.outcome, w.stderr)
	}
	return nil
}

func (w *orgTreeWorld) exitNonZeroAPIError() error {
	if w.outcome != APIError || w.exitCode != 3 {
		return fmt.Errorf("outcome=%v exit=%d, want APIError/3\nstderr: %s", w.outcome, w.exitCode, w.stderr)
	}
	return nil
}

func (w *orgTreeWorld) exitNonZero() error {
	if w.exitCode == 0 {
		return fmt.Errorf("exit code = 0, want a non-zero code (outcome %v)\nstderr: %s", w.outcome, w.stderr)
	}
	return nil
}

func (w *orgTreeWorld) stderrReportsAndPointsTo(report, pointer string) error {
	if !strings.Contains(strings.ToLower(w.stderr), strings.ToLower(report)) {
		return fmt.Errorf("stderr should report %q:\n%s", report, w.stderr)
	}
	if !strings.Contains(w.stderr, pointer) {
		return fmt.Errorf("stderr should point to %q:\n%s", pointer, w.stderr)
	}
	return nil
}

func (w *orgTreeWorld) stderrNamesHTTPStatus() error {
	if !strings.Contains(w.stderr, "404") {
		return fmt.Errorf("stderr should name the HTTP status (404):\n%s", w.stderr)
	}
	return nil
}

func (w *orgTreeWorld) stderrReportsUsageError() error {
	if w.outcome != UsageError || w.exitCode != 2 {
		return fmt.Errorf("outcome=%v exit=%d, want UsageError/2\nstderr: %s", w.outcome, w.exitCode, w.stderr)
	}
	if strings.TrimSpace(w.stderr) == "" {
		return fmt.Errorf("a usage error should be reported on stderr")
	}
	return nil
}

func (w *orgTreeWorld) noRequestSent() error {
	if w.transportCalls() != 0 {
		return fmt.Errorf("no API request should be sent, but the transport was called %d times", w.transportCalls())
	}
	return nil
}

func (w *orgTreeWorld) requestCarriesRawParam(param string) error {
	t, ok := w.transport.(*cannedTransport)
	if !ok {
		return fmt.Errorf("this scenario's transport does not record the query")
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

func (w *orgTreeWorld) literalPrintedToStdout(literal string) error {
	if !strings.Contains(w.stdout, literal) {
		return fmt.Errorf("stdout should contain %q:\n%s", literal, w.stdout)
	}
	return nil
}

// --- Then implementations (tree) ---

func (w *orgTreeWorld) nestedTreePrinted() error {
	for _, want := range []string{
		"General Company Circle (role_anchor)",
		"  Marketing (role_mkt)",
		"    Press Officer (role_press)",
	} {
		if !strings.Contains(w.stdout, want) {
			return fmt.Errorf("the tree should be a nested projection; missing %q:\n%s", want, w.stdout)
		}
	}
	return nil
}

func (w *orgTreeWorld) noTreeDataPrinted() error {
	if strings.TrimSpace(w.stdout) != "" {
		return fmt.Errorf("no tree data should be printed, got stdout:\n%s", w.stdout)
	}
	return nil
}

func (w *orgTreeWorld) treeRootedAt(id string) error {
	if !strings.Contains(w.stdout, "("+id+")") {
		return fmt.Errorf("the tree should be rooted at and include %q:\n%s", id, w.stdout)
	}
	return nil
}

func (w *orgTreeWorld) nodesCarryIncludes() error {
	for _, want := range []string{"Accountabilities:", "Defining the campaign", "Domains:", "The marketing budget"} {
		if !strings.Contains(w.stdout, want) {
			return fmt.Errorf("each node should carry its includes inline; missing %q:\n%s", want, w.stdout)
		}
	}
	return nil
}

func (w *orgTreeWorld) stderrNamesTreeIncludeSet() error {
	if !strings.Contains(w.stderr, "nonsense") {
		return fmt.Errorf("stderr should name the unsupported value:\n%s", w.stderr)
	}
	for _, want := range []string{"accountabilities", "domains", "members"} {
		if !strings.Contains(w.stderr, want) {
			return fmt.Errorf("stderr should name the tree read's supported set; missing %q:\n%s", want, w.stderr)
		}
	}
	return nil
}

func (w *orgTreeWorld) singleNodeTreePrinted() error {
	if !strings.Contains(w.stdout, "Solo Role (role_0123)") {
		return fmt.Errorf("a single-node tree should print its one node:\n%s", w.stdout)
	}
	if strings.Contains(w.stdout, "(+ subroles below depth)") {
		return fmt.Errorf("a true leaf must not carry the depth marker:\n%s", w.stdout)
	}
	return nil
}

func (w *orgTreeWorld) anchorAndDirectChildrenPrinted() error {
	for _, want := range []string{"Anchor (role_anchor)", "Deep Branch (role_0456)", "True Leaf (role_leaf)"} {
		if !strings.Contains(w.stdout, want) {
			return fmt.Errorf("the anchor and its direct children should print; missing %q:\n%s", want, w.stdout)
		}
	}
	return nil
}

func (w *orgTreeWorld) nodeMarkedBelowDepth(id string) error {
	re := regexp.MustCompile(`\(` + regexp.QuoteMeta(id) + `\)\s+\(\+ subroles below depth\)`)
	if !re.MatchString(w.stdout) {
		return fmt.Errorf("%q should be marked as having subroles below the returned tree:\n%s", id, w.stdout)
	}
	return nil
}

func (w *orgTreeWorld) trueLeafNotMarked() error {
	if !strings.Contains(w.stdout, "True Leaf (role_leaf)") {
		return fmt.Errorf("the true leaf should print:\n%s", w.stdout)
	}
	if strings.Contains(w.stdout, "True Leaf (role_leaf)  (+ subroles below depth)") {
		return fmt.Errorf("the true leaf must be marked as having none (no marker):\n%s", w.stdout)
	}
	return nil
}

func (w *orgTreeWorld) noInventedCount() error {
	// The marker carries no descendant count: it is exactly "(+ subroles below
	// depth)" — never a number of omitted descendants.
	if regexp.MustCompile(`below depth\)\s*\(?\s*\d`).MatchString(w.stdout) ||
		regexp.MustCompile(`\d+\s+(more|omitted|hidden)\s+subroles`).MatchString(w.stdout) {
		return fmt.Errorf("the marker must not invent a count of omitted descendants:\n%s", w.stdout)
	}
	return nil
}

// --- Then implementations (subroles) ---

func (w *orgTreeWorld) eachChildPrinted() error {
	for _, want := range []string{"Press Officer (role_press)", "Events Lead (role_events)"} {
		if !strings.Contains(w.stdout, want) {
			return fmt.Errorf("each direct child should print as a projection; missing %q:\n%s", want, w.stdout)
		}
	}
	return nil
}

func (w *orgTreeWorld) assignmentsAndPoliciesInline() error {
	for _, want := range []string{"Assignments:", "Alice Smith (per_x)", "Policies:", "All PRs require two approvals"} {
		if !strings.Contains(w.stdout, want) {
			return fmt.Errorf("the requested includes should print inline per child; missing %q:\n%s", want, w.stdout)
		}
	}
	return nil
}

func (w *orgTreeWorld) onlyFirstPageChildrenPrinted() error {
	if !strings.Contains(w.stdout, "First Page Child") {
		return fmt.Errorf("the first page should print:\n%s", w.stdout)
	}
	if w.transportCalls() != 1 {
		return fmt.Errorf("--first-page must not walk, want 1 call, got %d", w.transportCalls())
	}
	return nil
}

func (w *orgTreeWorld) stderrNotesMoreSubroles() error {
	if !strings.Contains(w.stderr, "more subroles exist") {
		return fmt.Errorf("stderr should note more subroles exist:\n%s", w.stderr)
	}
	return nil
}

func (w *orgTreeWorld) walkedEveryPage() error {
	if w.transportCalls() != 3 {
		return fmt.Errorf("the walk should issue three page requests, got %d", w.transportCalls())
	}
	return nil
}

func (w *orgTreeWorld) allPagesChildrenPrinted() error {
	for _, want := range []string{"Page One Child", "Page Two Child", "Page Three Child"} {
		if !strings.Contains(w.stdout, want) {
			return fmt.Errorf("every page's children should print, missing %q:\n%s", want, w.stdout)
		}
	}
	return nil
}

func (w *orgTreeWorld) partialChildrenPrinted() error {
	if !strings.Contains(w.stdout, "Gathered Child") {
		return fmt.Errorf("the partial set gathered so far should print:\n%s", w.stdout)
	}
	return nil
}

func (w *orgTreeWorld) stderrNotesIncomplete() error {
	if !strings.Contains(w.stderr, "incomplete") {
		return fmt.Errorf("stderr should note the result is incomplete and name the cause:\n%s", w.stderr)
	}
	return nil
}
