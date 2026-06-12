package cli

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/Luscii/cli-glassfrog/internal/apiclient"
)

// --- search fixtures --------------------------------------------------------

// searchPageComplete is a single complete page mixing a role hit (every field
// populated) and a note hit (null excerpt, no role_id), in descending-rank order.
const searchPageComplete = `{
  "data": [
    {"type":"role","id":"role_0123","title":"Onboarding Lead","excerpt":"owns onboarding","rank":0.99,"role_id":"role_0123"},
    {"type":"note","id":"note_0456","title":"Onboarding retro","excerpt":null,"rank":0.8}
  ],
  "meta":{"pagination":{"per_page":100,"has_next_page":false,"next_cursor":""}}
}`

// searchPageEmpty is a query that matched nothing.
const searchPageEmpty = `{"data":[],"meta":{"pagination":{"per_page":100,"has_next_page":false,"next_cursor":""}}}`

// searchPage builds a one-result page of the given type, optionally signalling a
// next page with the given cursor — for assembling a multi-page walk.
func searchPage(typ, id, title, nextCursor string) string {
	hasNext := "false"
	if nextCursor != "" {
		hasNext = "true"
	}
	return `{"data":[{"type":"` + typ + `","id":"` + id + `","title":"` + title + `",` +
		`"excerpt":"x","rank":0.5,"role_id":"role_0123"}],` +
		`"meta":{"pagination":{"per_page":100,"has_next_page":` + hasNext + `,"next_cursor":"` + nextCursor + `"}}}`
}

// runSearchOver drives the pure runSearch over a fake seam, returning the outcome
// and captured stdout/stderr, and failing if the token leaks. An empty cfg.query
// defaults to "onboarding".
func runSearchOver(t *testing.T, seam searchSeam, cfg searchConfig) (Outcome, string, string) {
	t.Helper()
	var out, errb bytes.Buffer
	cfg.seam = seam
	cfg.reqCtx = context.Background()
	cfg.stdout = &out
	cfg.stderr = &errb
	if cfg.query == "" {
		cfg.query = "onboarding"
	}
	outcome, _ := runSearch(cfg)
	if strings.Contains(out.String()+errb.String(), meSecretToken) {
		t.Fatalf("the token leaked into output: %q", out.String()+errb.String())
	}
	return outcome, out.String(), errb.String()
}

// --- default walk: relevance order, completeness ----------------------------

func TestRunSearch_ListSuccessWalksAndProjects(t *testing.T) {
	tr := &cannedTransport{status: 200, body: searchPageComplete}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, stderr := runSearchOver(t, seam, searchConfig{})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success\nstderr: %s", outcome, stderr)
	}
	if got := tr.lastPath; !strings.HasSuffix(got, "/search") {
		t.Errorf("path = %q, want it to end in /search", got)
	}
	// Both hits print, role before note (the API's relevance order, preserved).
	roleAt := strings.Index(stdout, "Onboarding Lead")
	noteAt := strings.Index(stdout, "Onboarding retro")
	if roleAt < 0 || noteAt < 0 || roleAt > noteAt {
		t.Errorf("results should print in API relevance order (role then note):\n%s", stdout)
	}
	// The type badge bridges each hit; a null excerpt renders as the marker; the
	// note (no role_id) omits the Role line.
	for _, want := range []string{"[role]", "[note]", "Excerpt: —"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("missing %q:\n%s", want, stdout)
		}
	}
	if strings.TrimSpace(stderr) != "" {
		t.Errorf("a complete success writes nothing to stderr, got %q", stderr)
	}
}

func TestRunSearch_EmptyResultPrintsNoResults(t *testing.T) {
	tr := &cannedTransport{status: 200, body: searchPageEmpty}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, stderr := runSearchOver(t, seam, searchConfig{query: "zxqv"})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success\nstderr: %s", outcome, stderr)
	}
	if strings.TrimSpace(stdout) != "No results." {
		t.Errorf("an empty result should print exactly `No results.`, got %q", stdout)
	}
}

func TestRunSearch_MultiPageWalksToCompletion(t *testing.T) {
	tr := &recordingSeqTransport{steps: []seqMeResp{
		{status: 200, body: searchPage("role", "role_1", "Page One Hit", "c1")},
		{status: 200, body: searchPage("note", "note_2", "Page Two Hit", "c2")},
		{status: 200, body: searchPage("project", "proj_3", "Page Three Hit", "")},
	}}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, stderr := runSearchOver(t, seam, searchConfig{})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success\nstderr: %s", outcome, stderr)
	}
	if tr.calls != 3 {
		t.Errorf("the walk should issue three page requests, got %d", tr.calls)
	}
	for _, want := range []string{"Page One Hit", "Page Two Hit", "Page Three Hit"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("every page's results should print; missing %q:\n%s", want, stdout)
		}
	}
}

// --- query forwarding: verbatim ---------------------------------------------

func TestRunSearch_QueryForwardedByteForByte(t *testing.T) {
	tr := &cannedTransport{status: 200, body: searchPageComplete}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	const raw = `strategy review -archived or "budget cuts"`
	outcome, _, stderr := runSearchOver(t, seam, searchConfig{query: raw})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success\nstderr: %s", outcome, stderr)
	}
	if got := tr.lastQuery.Get("query"); got != raw {
		t.Errorf("query must be forwarded byte-for-byte\n got: %q\nwant: %q", got, raw)
	}
}

// --- per_page = 100 (the /search max), not paging's generic 500 -------------

func TestRunSearch_DefaultWalkRequestsPerPage100(t *testing.T) {
	tr := &cannedTransport{status: 200, body: searchPageComplete}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	if _, _, stderr := runSearchOver(t, seam, searchConfig{}); strings.TrimSpace(stderr) != "" {
		t.Fatalf("unexpected stderr: %s", stderr)
	}
	if got := tr.lastQuery.Get("per_page"); got != "100" {
		t.Errorf("the default walk must request per_page=100 (the /search max, not paging's 500), got %q", got)
	}
}

func TestRunSearch_PerPageOverridesDefault(t *testing.T) {
	tr := &cannedTransport{status: 200, body: searchPageComplete}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	runSearchOver(t, seam, searchConfig{perPage: 50, perPageSet: true})
	if got := tr.lastQuery.Get("per_page"); got != "50" {
		t.Errorf("--per-page 50 should override the default; per_page = %q, want 50", got)
	}
}

// --- types scoping ----------------------------------------------------------

func TestRunSearch_TypesScopeSentWhenSet(t *testing.T) {
	tr := &cannedTransport{status: 200, body: searchPageComplete}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	runSearchOver(t, seam, searchConfig{types: []string{"role", "project"}})
	if got := tr.lastQuery.Get("types"); got != "role,project" {
		t.Errorf("types = %q, want role,project", got)
	}
}

func TestRunSearch_TypesOmittedSendsNoTypesParam(t *testing.T) {
	tr := &cannedTransport{status: 200, body: searchPageComplete}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	runSearchOver(t, seam, searchConfig{})
	if _, present := tr.lastQuery["types"]; present {
		t.Errorf("omitting --types must send no types param, got %v", tr.lastQuery["types"])
	}
}

func TestRunSearch_UnsupportedTypeRejectedBeforeRequest(t *testing.T) {
	tr := &cannedTransport{status: 200, body: searchPageComplete}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, stderr := runSearchOver(t, seam, searchConfig{types: []string{"nonsense"}})
	if outcome != UsageError {
		t.Fatalf("outcome = %v, want UsageError", outcome)
	}
	if tr.calls != 0 {
		t.Errorf("an unsupported --types value must send no request (tripwire), got %d calls", tr.calls)
	}
	if strings.TrimSpace(stdout) != "" {
		t.Errorf("no data should print, got %q", stdout)
	}
	if !strings.Contains(stderr, "nonsense") {
		t.Errorf("stderr should name the unsupported value:\n%s", stderr)
	}
	if !strings.Contains(stderr, "--types") {
		t.Errorf("stderr should name the --types flag (not --include):\n%s", stderr)
	}
	for _, want := range []string{"role", "policy", "domain"} {
		if !strings.Contains(stderr, want) {
			t.Errorf("stderr should name the supported set; missing %q:\n%s", want, stderr)
		}
	}
}

// --- first-page opt-out + completeness signal -------------------------------

func TestRunSearch_FirstPageStopsAtOnePageAndSignalsMore(t *testing.T) {
	tr := &cannedTransport{status: 200, body: searchPage("role", "role_1", "First Page Hit", "c1")}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, stderr := runSearchOver(t, seam, searchConfig{firstPage: true})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success\nstderr: %s", outcome, stderr)
	}
	if tr.calls != 1 {
		t.Errorf("--first-page must not walk; want 1 call, got %d", tr.calls)
	}
	if !strings.Contains(stdout, "First Page Hit") {
		t.Errorf("the first page should print:\n%s", stdout)
	}
	if !strings.Contains(stderr, "more results exist") {
		t.Errorf("stderr should note more results exist:\n%s", stderr)
	}
	if got := tr.lastQuery.Get("per_page"); got != "100" {
		t.Errorf("the --first-page request should also carry per_page=100, got %q", got)
	}
}

func TestRunSearch_MidWalkFailureYieldsPartialFlaggedIncomplete(t *testing.T) {
	tr := &seqMeTransport{steps: []seqMeResp{
		{status: 200, body: searchPage("role", "role_1", "Gathered Hit", "c1")},
		{status: 500, body: `{"detail":"boom"}`},
	}}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, stderr := runSearchOver(t, seam, searchConfig{})
	if outcome == Success {
		t.Fatalf("a mid-walk failure must exit non-zero, got Success")
	}
	if !strings.Contains(stdout, "Gathered Hit") {
		t.Errorf("the partial set gathered so far should print:\n%s", stdout)
	}
	if !strings.Contains(stderr, "incomplete") {
		t.Errorf("stderr should note the result is incomplete and name the cause:\n%s", stderr)
	}
}

// --- query + types carried on EVERY page of the walk ------------------------

func TestRunSearch_QueryAndTypesCarriedOnEveryPage(t *testing.T) {
	tr := &recordingSeqTransport{steps: []seqMeResp{
		{status: 200, body: searchPage("role", "role_1", "Page One Hit", "c1")},
		{status: 200, body: searchPage("role", "role_2", "Page Two Hit", "")},
	}}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, _, stderr := runSearchOver(t, seam, searchConfig{query: "onboarding", types: []string{"role"}})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success\nstderr: %s", outcome, stderr)
	}
	if len(tr.queries) != 2 {
		t.Fatalf("expected 2 page requests, got %d", len(tr.queries))
	}
	// The page-2 request must retain BOTH query and types — not just page 1.
	page2 := tr.queries[1]
	if got := page2.Get("query"); got != "onboarding" {
		t.Errorf("page-2 request query = %q, want onboarding (must ride every page)", got)
	}
	if got := page2.Get("types"); got != "role" {
		t.Errorf("page-2 request types = %q, want role (must ride every page)", got)
	}
}

// --- error classification ---------------------------------------------------

func TestRunSearch_NoTokenIsUsageError(t *testing.T) {
	seam := &fakeMeSeam{ctx: noTokenContext(), transport: &cannedTransport{status: 200, body: searchPageComplete}}
	outcome, stdout, stderr := runSearchOver(t, seam, searchConfig{})
	if outcome != UsageError {
		t.Fatalf("outcome = %v, want UsageError\nstderr: %s", outcome, stderr)
	}
	if strings.TrimSpace(stdout) != "" {
		t.Errorf("no data should print, got %q", stdout)
	}
	if !strings.Contains(strings.ToLower(stderr), "not authenticated") {
		t.Errorf("stderr should report not authenticated:\n%s", stderr)
	}
}

func TestRunSearch_MalformedQueryStatusIsAPIError(t *testing.T) {
	tr := &cannedTransport{status: 400, body: `{"detail":"malformed query"}`}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}
	outcome, stdout, stderr := runSearchOver(t, seam, searchConfig{})
	if outcome != APIError {
		t.Fatalf("outcome = %v, want APIError", outcome)
	}
	if strings.TrimSpace(stdout) != "" {
		t.Errorf("no data should print on an API error, got %q", stdout)
	}
	if !strings.Contains(stderr, "400") {
		t.Errorf("stderr should name the HTTP status (400):\n%s", stderr)
	}
}

func TestRunSearch_TransportErrorIsNetworkUnavailable(t *testing.T) {
	tr := &cannedTransport{netErr: errors.New("dial tcp: connection refused")}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}
	outcome, _, stderr := runSearchOver(t, seam, searchConfig{})
	if outcome != NetworkUnavailable {
		t.Fatalf("outcome = %v, want NetworkUnavailable", outcome)
	}
	if strings.TrimSpace(stderr) == "" {
		t.Error("a transport failure should be named on stderr")
	}
}

func TestRunSearch_UndecodableBodyIsAPIError(t *testing.T) {
	tr := &cannedTransport{status: 200, body: `this is not json`}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}
	outcome, _, _ := runSearchOver(t, seam, searchConfig{})
	if outcome != APIError {
		t.Fatalf("a 2xx body in the wrong shape is a *DecodeError → APIError(3), got %v", outcome)
	}
}

func TestRunSearch_InvalidOutputFormatIsUsageErrorNoRequest(t *testing.T) {
	tr := &cannedTransport{status: 200, body: searchPageComplete}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}
	outcome, _, stderr := runSearchOver(t, seam, searchConfig{outputFlag: "verbose", outputPresent: true})
	if outcome != UsageError {
		t.Fatalf("outcome = %v, want UsageError", outcome)
	}
	if tr.calls != 0 {
		t.Errorf("an invalid --output must fail before any request, got %d calls", tr.calls)
	}
	if strings.TrimSpace(stderr) == "" {
		t.Error("the invalid format should be named on stderr")
	}
}

// reportIncompleteSearchWalk must return the REFINED error (not the original
// Stop), so the returned value agrees with the classified outcome and a downstream
// errors.As sees the extracted *ProblemError on a mid-walk non-2xx.
func TestReportIncompleteSearchWalk_ReturnsRefinedError(t *testing.T) {
	var errb bytes.Buffer
	stop := &apiclient.ResponseError{StatusCode: 403, Body: []byte(`{"detail":"Forbidden"}`)}
	outcome, retErr := reportIncompleteSearchWalk(&errb, stop)
	if outcome != PermissionError {
		t.Errorf("outcome = %v, want PermissionError (403 → refined classification)", outcome)
	}
	var pe *apiclient.ProblemError
	if !errors.As(retErr, &pe) {
		t.Errorf("returned error should be the refined *ProblemError, got %T", retErr)
	}
	if !strings.Contains(errb.String(), "incomplete") {
		t.Errorf("stderr should carry the incomplete note, got %q", errb.String())
	}
}

// --- structured output ------------------------------------------------------

func TestRunSearch_StructuredEmitsRawPayload(t *testing.T) {
	tr := &cannedTransport{status: 200, body: searchPageComplete}
	seam := &fakeMeSeam{ctx: validMeContext(), envOutput: "json", transport: tr}
	outcome, stdout, stderr := runSearchOver(t, seam, searchConfig{})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success\nstderr: %s", outcome, stderr)
	}
	for _, want := range []string{`"data"`, "role_0123"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("structured output should emit the raw payload; missing %q:\n%s", want, stdout)
		}
	}
}

// --- validateTypes (pure) ---------------------------------------------------

func TestValidateTypes(t *testing.T) {
	if err := validateTypes(nil); err != nil {
		t.Errorf("no --types is valid (requests all types), got %v", err)
	}
	if err := validateTypes([]string{"role", "policy", "domain"}); err != nil {
		t.Errorf("all-supported types should pass, got %v", err)
	}
	single := validateTypes([]string{"widget"})
	if single == nil || !strings.Contains(single.Error(), "--types") || !strings.Contains(single.Error(), `"widget"`) {
		t.Errorf("a bad value should name --types and the value, got %v", single)
	}
	if strings.Contains(single.Error(), "values ") {
		t.Errorf("a single bad value should use the singular noun, got %v", single)
	}
	multi := validateTypes([]string{"widget", "gadget"})
	if multi == nil || !strings.Contains(multi.Error(), "values ") {
		t.Errorf("multiple bad values should use the plural noun, got %v", multi)
	}
}
