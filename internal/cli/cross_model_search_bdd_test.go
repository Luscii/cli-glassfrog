package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/Luscii/cli-glassfrog/internal/apiclient"
	"github.com/cucumber/godog"
)

// TestCrossModelSearchFeatures runs the executable acceptance for Cross-Model
// Search (041): the `search <query>` command driven through its seam over a fake
// base transport, so every scenario runs offline (no real network, no real
// ~/.glassfrogrc). Its Paths name ONLY this spec's feature file — never the
// features/ directory — so the suite reports its own independent scenario count
// and un-@wip-ping these scenarios cannot disturb another suite (LEARNINGS: a
// suite points at its own feature file). The @validation scenarios stay @wip
// (held for the validate skill) and are skipped by the ~@wip filter.
func TestCrossModelSearchFeatures(t *testing.T) {
	suite := godog.TestSuite{
		ScenarioInitializer: initializeCrossModelSearchScenario,
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"../../features/undiscoverable-governance/cross-model-search.feature"},
			Tags:     "~@wip",
			TestingT: t,
		},
	}
	if suite.Run() != 0 {
		t.Fatal("non-zero status: cross-model-search feature scenarios failed")
	}
}

// searchRoleProjectPage is a single complete page with a role hit and a project
// hit (for the --types role,project scoping scenario): both carry an owning role
// id and an excerpt.
const searchRoleProjectPage = `{
  "data": [
    {"type":"role","id":"role_b","title":"Budget Owner","excerpt":"owns budget","rank":0.9,"role_id":"role_b"},
    {"type":"project","id":"proj_b","title":"Budget Review","excerpt":"q2 budget","rank":0.7,"role_id":"role_b"}
  ],
  "meta":{"pagination":{"per_page":100,"has_next_page":false,"next_cursor":""}}
}`

// searchWorld is the per-scenario state: the connection context and fake transport
// assembled from the Given steps, plus the captured outcome/exit-code/streams and
// the parsed query of the When run. Everything is injected — no step touches the
// real network, env, or home.
type searchWorld struct {
	ctx       apiclient.ConnectionContext
	transport http.RoundTripper
	secret    string

	query    string // the positional query parsed from the invocation (for verbatim assertions)
	outcome  Outcome
	exitCode int
	stdout   string
	stderr   string
}

func initializeCrossModelSearchScenario(sc *godog.ScenarioContext) {
	w := &searchWorld{}
	sc.Before(func(ctx context.Context, _ *godog.Scenario) (context.Context, error) {
		*w = searchWorld{
			// A single complete page of mixed-type results is the default; the
			// per-scenario Given steps override the transport/context as needed.
			transport: &cannedTransport{status: 200, body: searchPageComplete},
			secret:    meSecretToken,
		}
		w.ctx = validMeContext()
		return ctx, nil
	})

	// --- Givens ---
	sc.Step(`^a complete connection context with a stored token$`, w.completeContext)
	sc.Step(`^several resources of different types match "([^"]*)"$`, w.resourcesOfDifferentTypesMatch)
	sc.Step(`^no usable token is available to the CLI$`, w.noToken)
	sc.Step(`^the API cannot process the submitted query$`, w.apiRejectsQuery)
	sc.Step(`^no resource matches "([^"]*)"$`, w.noResourceMatches)
	sc.Step(`^the organization has roles and projects matching "([^"]*)"$`, w.rolesAndProjectsMatch)
	sc.Step(`^a query that matches at least one role$`, w.queryMatchesARole)
	sc.Step(`^the query "([^"]*)" matches results spanning more than one page$`, w.querySpansMoreThanOnePage)
	sc.Step(`^the result walk fails after retrieving the first page$`, w.walkFailsAfterFirstPage)
	sc.Step(`^the query "([^"]*)" scoped to "([^"]*)" spans more than one page$`, w.scopedQuerySpansMoreThanOnePage)

	// --- Whens --- (the no-query form first; its anchored suffix keeps the general
	// form from matching it) ---
	sc.Step(`^an agent runs "glassfrog (.+)" with no query argument$`, w.runCommandNoQuery)
	sc.Step(`^an agent runs "glassfrog (.+)"$`, w.runCommand)

	// --- Thens ---
	sc.Step(`^the request will carry "([^"]*)" set to "([^"]*)" with no type scope$`, w.requestCarriesParamNoTypes)
	sc.Step(`^the request will carry "([^"]*)" set to "([^"]*)"$`, w.requestCarriesParam)
	sc.Step(`^the request will carry "([^"]*)" as the whole string unmodified$`, w.requestCarriesQueryVerbatim)
	sc.Step(`^the matching results will be printed in relevance order$`, w.resultsPrintedInRelevanceOrder)
	sc.Step(`^the command will exit with code (\d+)$`, w.exitWithCode)
	sc.Step(`^the command will exit with a non-zero API-error code$`, w.exitNonZeroAPIError)
	sc.Step(`^the command will exit with a non-zero code$`, w.exitNonZero)
	sc.Step(`^stderr will report "([^"]*)" and point to "([^"]*)"$`, w.stderrReportsAndPointsTo)
	sc.Step(`^no results will be printed$`, w.noResultsPrinted)
	sc.Step(`^stderr will report that the search failed and name the HTTP status$`, w.stderrNamesHTTPStatus)
	sc.Step(`^"([^"]*)" will be printed to stdout$`, w.literalPrintedToStdout)
	sc.Step(`^stderr will report a usage error$`, w.stderrReportsUsageError)
	sc.Step(`^no API request will be sent$`, w.noRequestSent)
	sc.Step(`^only role and project results will be printed$`, w.onlyRoleAndProjectPrinted)
	sc.Step(`^stderr will report the unsupported value and list the supported set$`, w.stderrNamesUnsupportedType)
	sc.Step(`^each result will carry its type, id, title, excerpt, and rank$`, w.eachResultCarriesBridgeFields)
	sc.Step(`^a role result will also carry its owning role id$`, w.roleResultCarriesRoleID)
	sc.Step(`^every page will be walked and the complete relevance-ordered set will be printed$`, w.everyPageWalked)
	sc.Step(`^only the first page of results will be printed$`, w.onlyFirstPagePrinted)
	sc.Step(`^stderr will note that more results exist$`, w.stderrNotesMoreExist)
	sc.Step(`^the results retrieved so far will be printed$`, w.partialResultsPrinted)
	sc.Step(`^stderr will note the result is incomplete and name the cause$`, w.stderrNotesIncomplete)
	sc.Step(`^every page request of the walk will retain "([^"]*)" set to "([^"]*)" and "([^"]*)" set to "([^"]*)"$`, w.everyPageRetainsParams)
}

// --- Given implementations ---

func (w *searchWorld) completeContext() error { w.ctx = validMeContext(); return nil }

func (w *searchWorld) resourcesOfDifferentTypesMatch(_ string) error {
	w.transport = &cannedTransport{status: 200, body: searchPageComplete}
	return nil
}

func (w *searchWorld) noToken() error {
	w.ctx = noTokenContext()
	return nil
}

func (w *searchWorld) apiRejectsQuery() error {
	w.transport = &cannedTransport{status: 400, body: `{"detail":"malformed query"}`}
	return nil
}

func (w *searchWorld) noResourceMatches(_ string) error {
	w.transport = &cannedTransport{status: 200, body: searchPageEmpty}
	return nil
}

func (w *searchWorld) rolesAndProjectsMatch(_ string) error {
	w.transport = &cannedTransport{status: 200, body: searchRoleProjectPage}
	return nil
}

func (w *searchWorld) queryMatchesARole() error {
	w.transport = &cannedTransport{status: 200, body: searchPageComplete}
	return nil
}

func (w *searchWorld) querySpansMoreThanOnePage(_ string) error {
	w.transport = &recordingSeqTransport{steps: []seqMeResp{
		{status: 200, body: searchPage("role", "role_1", "Page One Hit", "c1")},
		{status: 200, body: searchPage("note", "note_2", "Page Two Hit", "")},
	}}
	return nil
}

func (w *searchWorld) walkFailsAfterFirstPage() error {
	w.transport = &seqMeTransport{steps: []seqMeResp{
		{status: 200, body: searchPage("role", "role_1", "Gathered Hit", "c1")},
		{status: 500, body: `{"detail":"boom"}`},
	}}
	return nil
}

func (w *searchWorld) scopedQuerySpansMoreThanOnePage(_, _ string) error {
	w.transport = &recordingSeqTransport{steps: []seqMeResp{
		{status: 200, body: searchPage("role", "role_1", "Page One Hit", "c1")},
		{status: 200, body: searchPage("role", "role_2", "Page Two Hit", "")},
	}}
	return nil
}

// --- When implementations ---

// runCommand parses the captured invocation with a QUOTE-AWARE splitter (so a
// quoted multi-word query reaches cobra as ONE positional — strings.Fields would
// shred it) and dispatches it through a real root with only the `search` leaf
// attached over a fake seam. It records the parsed query (the first positional
// after `search`) for the verbatim assertions, and asserts the token never leaks.
func (w *searchWorld) runCommand(invocation string) error {
	args := splitArgs(invocation)
	if len(args) > 1 {
		w.query = args[1]
	}
	return w.dispatch(args)
}

// runCommandNoQuery handles the `"glassfrog search" with no query argument` form:
// the invocation is just `search`, with no positional — exercising the cobra
// ExactArgs(1) rejection.
func (w *searchWorld) runCommandNoQuery(invocation string) error {
	return w.dispatch(splitArgs(invocation))
}

func (w *searchWorld) dispatch(args []string) error {
	root := NewRootCommand()
	seam := &fakeMeSeam{ctx: w.ctx, transport: w.transport}
	MustRegister(root, newSearchCommand(seam))

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

// splitArgs splits a command line into arguments, honoring double-quoted segments
// so a quoted multi-word query (`search "strategy review -archived"`) becomes ONE
// argument. The existing whitespace-splitting invocation step (strings.Fields)
// cannot do this — exactly the gap this feature requires (T003 reuse caveat).
func splitArgs(s string) []string {
	var args []string
	var cur strings.Builder
	inQuote, hasToken := false, false
	flush := func() {
		if hasToken {
			args = append(args, cur.String())
			cur.Reset()
			hasToken = false
		}
	}
	for _, r := range s {
		switch {
		case r == '"':
			inQuote = !inQuote
			hasToken = true // an empty "" is still a (blank) token
		case r == ' ' && !inQuote:
			flush()
		default:
			cur.WriteRune(r)
			hasToken = true
		}
	}
	flush()
	return args
}

// --- helpers ---

// transportCalls reads the request count off whichever fake transport the
// scenario installed.
func (w *searchWorld) transportCalls() int {
	switch t := w.transport.(type) {
	case *cannedTransport:
		return t.calls
	case *seqMeTransport:
		return t.calls
	case *recordingSeqTransport:
		return t.calls
	default:
		return -1
	}
}

// lastQuery reads the most-recent request's query off whichever recording fake the
// scenario installed.
func (w *searchWorld) lastQuery() (url.Values, error) {
	switch t := w.transport.(type) {
	case *cannedTransport:
		return t.lastQuery, nil
	case *recordingSeqTransport:
		if len(t.queries) == 0 {
			return nil, errors.New("no request was recorded")
		}
		return t.queries[len(t.queries)-1], nil
	default:
		return nil, errors.New("this scenario's transport does not record the query")
	}
}

// --- Then implementations ---

func (w *searchWorld) requestCarriesParamNoTypes(param, value string) error {
	if err := w.requestCarriesParam(param, value); err != nil {
		return err
	}
	q, err := w.lastQuery()
	if err != nil {
		return err
	}
	if _, present := q["types"]; present {
		return fmt.Errorf("expected no type scope, but the request carried types=%v", q["types"])
	}
	return nil
}

func (w *searchWorld) requestCarriesParam(param, value string) error {
	q, err := w.lastQuery()
	if err != nil {
		return err
	}
	if got := q.Get(param); got != value {
		return fmt.Errorf("request %s = %q, want %q", param, got, value)
	}
	return nil
}

func (w *searchWorld) requestCarriesQueryVerbatim(param string) error {
	q, err := w.lastQuery()
	if err != nil {
		return err
	}
	if got := q.Get(param); got != w.query {
		return fmt.Errorf("%s should be forwarded byte-for-byte\n got: %q\nwant: %q (the unmodified input)", param, got, w.query)
	}
	return nil
}

func (w *searchWorld) resultsPrintedInRelevanceOrder() error {
	// searchPageComplete lists the role hit before the note hit; the render must
	// preserve that order (no re-sort).
	roleAt := strings.Index(w.stdout, "Onboarding Lead")
	noteAt := strings.Index(w.stdout, "Onboarding retro")
	if roleAt < 0 || noteAt < 0 {
		return fmt.Errorf("both results should be printed:\n%s", w.stdout)
	}
	if roleAt > noteAt {
		return fmt.Errorf("results should print in the API's relevance order (role before note):\n%s", w.stdout)
	}
	return nil
}

func (w *searchWorld) exitWithCode(code int) error {
	if w.exitCode != code {
		return fmt.Errorf("exit code = %d, want %d (outcome %v)\nstderr: %s", w.exitCode, code, w.outcome, w.stderr)
	}
	return nil
}

func (w *searchWorld) exitNonZeroAPIError() error {
	if w.outcome != APIError || w.exitCode != 3 {
		return fmt.Errorf("outcome=%v exit=%d, want APIError/3\nstderr: %s", w.outcome, w.exitCode, w.stderr)
	}
	return nil
}

func (w *searchWorld) exitNonZero() error {
	if w.exitCode == 0 {
		return fmt.Errorf("exit code = 0, want a non-zero code (outcome %v)\nstderr: %s", w.outcome, w.stderr)
	}
	return nil
}

func (w *searchWorld) stderrReportsAndPointsTo(report, pointer string) error {
	if !strings.Contains(strings.ToLower(w.stderr), strings.ToLower(report)) {
		return fmt.Errorf("stderr should report %q:\n%s", report, w.stderr)
	}
	if !strings.Contains(w.stderr, pointer) {
		return fmt.Errorf("stderr should point to %q:\n%s", pointer, w.stderr)
	}
	return nil
}

func (w *searchWorld) noResultsPrinted() error {
	if strings.TrimSpace(w.stdout) != "" {
		return fmt.Errorf("no results should be printed, got stdout:\n%s", w.stdout)
	}
	return nil
}

func (w *searchWorld) stderrNamesHTTPStatus() error {
	if !strings.Contains(w.stderr, "400") {
		return fmt.Errorf("stderr should report the search failed and name the HTTP status (400):\n%s", w.stderr)
	}
	return nil
}

func (w *searchWorld) literalPrintedToStdout(literal string) error {
	if !strings.Contains(w.stdout, literal) {
		return fmt.Errorf("stdout should contain %q:\n%s", literal, w.stdout)
	}
	return nil
}

func (w *searchWorld) stderrReportsUsageError() error {
	if w.outcome != UsageError || w.exitCode != 2 {
		return fmt.Errorf("outcome=%v exit=%d, want UsageError/2\nstderr: %s", w.outcome, w.exitCode, w.stderr)
	}
	if strings.TrimSpace(w.stderr) == "" {
		return errors.New("a usage error should be reported on stderr")
	}
	return nil
}

func (w *searchWorld) noRequestSent() error {
	if w.transportCalls() != 0 {
		return fmt.Errorf("no API request should be sent, but the transport was called %d times", w.transportCalls())
	}
	return nil
}

func (w *searchWorld) onlyRoleAndProjectPrinted() error {
	for _, want := range []string{"[role]", "Budget Owner", "[project]", "Budget Review"} {
		if !strings.Contains(w.stdout, want) {
			return fmt.Errorf("the role and project results should print; missing %q:\n%s", want, w.stdout)
		}
	}
	return nil
}

func (w *searchWorld) stderrNamesUnsupportedType() error {
	if !strings.Contains(w.stderr, "nonsense") {
		return fmt.Errorf("stderr should name the unsupported value:\n%s", w.stderr)
	}
	for _, want := range []string{"role", "policy", "domain"} {
		if !strings.Contains(w.stderr, want) {
			return fmt.Errorf("stderr should list the supported set; missing %q:\n%s", want, w.stderr)
		}
	}
	return nil
}

func (w *searchWorld) eachResultCarriesBridgeFields() error {
	// The role hit in searchPageComplete carries type+id+title+excerpt+rank.
	for _, want := range []string{"[role]", "role_0123", "Onboarding Lead", "owns onboarding", "rank 0.99"} {
		if !strings.Contains(w.stdout, want) {
			return fmt.Errorf("each result should carry type/id/title/excerpt/rank; missing %q:\n%s", want, w.stdout)
		}
	}
	return nil
}

func (w *searchWorld) roleResultCarriesRoleID() error {
	if !strings.Contains(w.stdout, "Role: role_0123") {
		return fmt.Errorf("a role result should carry its owning role id (Role: line):\n%s", w.stdout)
	}
	return nil
}

func (w *searchWorld) everyPageWalked() error {
	if w.transportCalls() < 2 {
		return fmt.Errorf("the walk should issue more than one page request, got %d", w.transportCalls())
	}
	for _, want := range []string{"Page One Hit", "Page Two Hit"} {
		if !strings.Contains(w.stdout, want) {
			return fmt.Errorf("the complete set across pages should print; missing %q:\n%s", want, w.stdout)
		}
	}
	return nil
}

func (w *searchWorld) onlyFirstPagePrinted() error {
	if !strings.Contains(w.stdout, "Page One Hit") {
		return fmt.Errorf("the first page should print:\n%s", w.stdout)
	}
	if strings.Contains(w.stdout, "Page Two Hit") {
		return fmt.Errorf("--first-page must not print later pages:\n%s", w.stdout)
	}
	if w.transportCalls() != 1 {
		return fmt.Errorf("--first-page must not walk; want 1 call, got %d", w.transportCalls())
	}
	return nil
}

func (w *searchWorld) stderrNotesMoreExist() error {
	if !strings.Contains(w.stderr, "more results exist") {
		return fmt.Errorf("stderr should note more results exist:\n%s", w.stderr)
	}
	return nil
}

func (w *searchWorld) partialResultsPrinted() error {
	if !strings.Contains(w.stdout, "Gathered Hit") {
		return fmt.Errorf("the partial set gathered so far should print:\n%s", w.stdout)
	}
	return nil
}

func (w *searchWorld) stderrNotesIncomplete() error {
	if !strings.Contains(w.stderr, "incomplete") {
		return fmt.Errorf("stderr should note the result is incomplete and name the cause:\n%s", w.stderr)
	}
	return nil
}

func (w *searchWorld) everyPageRetainsParams(p1, v1, p2, v2 string) error {
	t, ok := w.transport.(*recordingSeqTransport)
	if !ok {
		return errors.New("this scenario's transport does not record per-page queries")
	}
	if len(t.queries) < 2 {
		return fmt.Errorf("expected the walk to span more than one page, got %d requests", len(t.queries))
	}
	for i, q := range t.queries {
		if got := q.Get(p1); got != v1 {
			return fmt.Errorf("page-%d request %s = %q, want %q (must ride every page)", i+1, p1, got, v1)
		}
		if got := q.Get(p2); got != v2 {
			return fmt.Errorf("page-%d request %s = %q, want %q (must ride every page)", i+1, p2, got, v2)
		}
	}
	return nil
}
