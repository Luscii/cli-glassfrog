package cli

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

// --- domains fixtures -------------------------------------------------------

// domainsPageComplete is a single complete page of two domains: one bound to a
// controlling role, one with a null role_id. Neither carries a policies embed
// (the list never includes it).
const domainsPageComplete = `{
  "data": [
    {"id": "dom_budget", "type": "domain", "description": "The marketing budget",
     "role_id": "role_0123", "created_at": "2024-01-01T00:00:00Z", "updated_at": "2024-01-01T00:00:00Z"},
    {"id": "dom_brand", "type": "domain", "description": "The brand guidelines",
     "role_id": null, "created_at": "2024-01-01T00:00:00Z", "updated_at": "2024-01-01T00:00:00Z"}
  ],
  "meta": {"pagination": {"per_page": 100, "has_next_page": false, "next_cursor": ""}}
}`

// domainsPageEmpty is a role that controls no domains.
const domainsPageEmpty = `{"data": [], "meta": {"pagination": {"per_page": 100, "has_next_page": false, "next_cursor": ""}}}`

// domainsPage builds a one-domain page for a named domain, optionally signalling a
// next page with the given cursor — for assembling a multi-page walk.
func domainsPage(id, desc, nextCursor string) string {
	hasNext := "false"
	if nextCursor != "" {
		hasNext = "true"
	}
	return `{"data":[{"id":"` + id + `","type":"domain","description":"` + desc + `",` +
		`"role_id":"role_0123","created_at":"t","updated_at":"t"}],` +
		`"meta":{"pagination":{"per_page":1,"has_next_page":` + hasNext + `,"next_cursor":"` + nextCursor + `"}}}`
}

// recordingSeqTransport is a fake base transport that returns canned replies in
// order (like seqMeTransport) but ALSO records every request's query — so a test
// can assert the `q` search term rides EVERY page of the walk, not just the first
// (033 risk: q must ride every walked page).
type recordingSeqTransport struct {
	calls   int
	queries []url.Values
	paths   []string
	steps   []seqMeResp
}

func (s *recordingSeqTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	s.calls++
	s.queries = append(s.queries, req.URL.Query())
	s.paths = append(s.paths, req.URL.Path)
	i := s.calls - 1
	if i >= len(s.steps) {
		i = len(s.steps) - 1
	}
	step := s.steps[i]
	header := step.header
	if header == nil {
		header = make(http.Header)
	}
	return &http.Response{
		StatusCode: step.status,
		Header:     header,
		Body:       io.NopCloser(strings.NewReader(step.body)),
	}, nil
}

// runDomainsOver drives the pure runDomains over a fake seam, returning the
// outcome and captured stdout/stderr, and failing if the token leaks.
func runDomainsOver(t *testing.T, seam domainsSeam, cfg domainsConfig) (Outcome, string, string) {
	t.Helper()
	var out, errb bytes.Buffer
	cfg.seam = seam
	cfg.reqCtx = context.Background()
	cfg.stdout = &out
	cfg.stderr = &errb
	if cfg.id == "" {
		cfg.id = "role_0123"
	}
	outcome, _ := runDomains(cfg)
	if strings.Contains(out.String()+errb.String(), meSecretToken) {
		t.Fatalf("the token leaked into output: %q", out.String()+errb.String())
	}
	return outcome, out.String(), errb.String()
}

// --- default walk -----------------------------------------------------------

func TestRunDomains_ListSuccessWalksAndProjects(t *testing.T) {
	tr := &cannedTransport{status: 200, body: domainsPageComplete}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, stderr := runDomainsOver(t, seam, domainsConfig{})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success\nstderr: %s", outcome, stderr)
	}
	if got := tr.lastPath; !strings.HasSuffix(got, "/roles/role_0123/domains") {
		t.Errorf("path = %q, want /roles/role_0123/domains", got)
	}
	for _, want := range []string{"The marketing budget (dom_budget)", "The brand guidelines (dom_brand)"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("each domain should print as a projection; missing %q:\n%s", want, stdout)
		}
	}
	if strings.TrimSpace(stderr) != "" {
		t.Errorf("a complete success writes nothing to stderr, got %q", stderr)
	}
}

func TestRunDomains_NoDomainsPrintsNoDomains(t *testing.T) {
	tr := &cannedTransport{status: 200, body: domainsPageEmpty}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, stderr := runDomainsOver(t, seam, domainsConfig{})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success", outcome)
	}
	if strings.TrimRight(stdout, "\n") != "No domains." {
		t.Errorf("a role with no domains should print exactly `No domains.`, got %q", stdout)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Errorf("an empty list is a clean success; stderr should be empty, got %q", stderr)
	}
}

func TestRunDomains_WalksEveryPageToCompletion(t *testing.T) {
	tr := &seqMeTransport{steps: []seqMeResp{
		{status: 200, body: domainsPage("dom_1", "Page One Domain", "c1")},
		{status: 200, body: domainsPage("dom_2", "Page Two Domain", "c2")},
		{status: 200, body: domainsPage("dom_3", "Page Three Domain", "")},
	}}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, stderr := runDomainsOver(t, seam, domainsConfig{})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success\nstderr: %s", outcome, stderr)
	}
	if tr.calls != 3 {
		t.Errorf("the walk should issue three page requests, got %d", tr.calls)
	}
	for _, want := range []string{"Page One Domain", "Page Two Domain", "Page Three Domain"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("every page's domains should print, missing %q:\n%s", want, stdout)
		}
	}
}

// --- search (--query) -------------------------------------------------------

func TestRunDomains_QuerySetsQParam(t *testing.T) {
	tr := &cannedTransport{status: 200, body: domainsPageComplete}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, _, stderr := runDomainsOver(t, seam, domainsConfig{query: "review"})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success\nstderr: %s", outcome, stderr)
	}
	if got := tr.lastQuery.Get("q"); got != "review" {
		t.Errorf("q = %q, want review", got)
	}
}

func TestRunDomains_QueryRidesEveryWalkedPage(t *testing.T) {
	tr := &recordingSeqTransport{steps: []seqMeResp{
		{status: 200, body: domainsPage("dom_1", "Review process", "c1")},
		{status: 200, body: domainsPage("dom_2", "Review board", "")},
	}}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, _, stderr := runDomainsOver(t, seam, domainsConfig{query: "review"})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success\nstderr: %s", outcome, stderr)
	}
	if tr.calls != 2 {
		t.Fatalf("the walk should issue two page requests, got %d", tr.calls)
	}
	for i, q := range tr.queries {
		if got := q.Get("q"); got != "review" {
			t.Errorf("page %d carried q = %q, want review (q must ride every walked page)", i+1, got)
		}
	}
}

func TestRunDomains_BlankQuerySendsNoQ(t *testing.T) {
	tr := &cannedTransport{status: 200, body: domainsPageComplete}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, _, _ := runDomainsOver(t, seam, domainsConfig{query: "   "})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success", outcome)
	}
	if _, present := tr.lastQuery["q"]; present {
		t.Errorf("a blank/whitespace --query must send no q, but q was present: %v", tr.lastQuery)
	}
}

// --- include rejected on the list ------------------------------------------

func TestRunDomains_IncludeRejectedBeforeRequest(t *testing.T) {
	tr := &cannedTransport{status: 200, body: domainsPageComplete}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, stderr := runDomainsOver(t, seam, domainsConfig{includeSet: true})
	if outcome != UsageError {
		t.Fatalf("outcome = %v, want UsageError", outcome)
	}
	if tr.calls != 0 {
		t.Errorf("--include on the list must send no request (tripwire), got %d", tr.calls)
	}
	if strings.TrimSpace(stdout) != "" {
		t.Errorf("no data should print, got %q", stdout)
	}
	if !strings.Contains(stderr, "--include") {
		t.Errorf("stderr should name the --include misuse:\n%s", stderr)
	}
}

// --- first-page opt-out + completeness -------------------------------------

func TestRunDomains_FirstPageStopsAtOnePageAndSignalsMore(t *testing.T) {
	tr := &cannedTransport{status: 200, body: domainsPage("dom_1", "First Page Domain", "c1")}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, stderr := runDomainsOver(t, seam, domainsConfig{firstPage: true})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success\nstderr: %s", outcome, stderr)
	}
	if tr.calls != 1 {
		t.Errorf("--first-page must not walk; want 1 call, got %d", tr.calls)
	}
	if !strings.Contains(stdout, "First Page Domain") {
		t.Errorf("the first page should print:\n%s", stdout)
	}
	if !strings.Contains(stderr, "more domains exist") {
		t.Errorf("stderr should note more domains exist:\n%s", stderr)
	}
}

func TestRunDomains_MidWalkFailureYieldsPartialFlaggedIncomplete(t *testing.T) {
	tr := &seqMeTransport{steps: []seqMeResp{
		{status: 200, body: domainsPage("dom_1", "Gathered Domain", "c1")},
		{status: 500, body: `{"detail":"boom"}`},
	}}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, stderr := runDomainsOver(t, seam, domainsConfig{})
	if outcome == Success {
		t.Fatalf("a mid-walk failure must exit non-zero, got Success")
	}
	if !strings.Contains(stdout, "Gathered Domain") {
		t.Errorf("the partial set gathered so far should print:\n%s", stdout)
	}
	if !strings.Contains(stderr, "incomplete") {
		t.Errorf("stderr should note the result is incomplete and name the cause:\n%s", stderr)
	}
}

// --- error classification ---------------------------------------------------

func TestRunDomains_UnknownIdSurfacesAPIStatus(t *testing.T) {
	tr := &cannedTransport{status: 404, body: `{"detail":"Role not found"}`}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}
	outcome, stdout, stderr := runDomainsOver(t, seam, domainsConfig{id: "role_ffff"})
	if outcome != APIError {
		t.Fatalf("outcome = %v, want APIError", outcome)
	}
	if strings.TrimSpace(stdout) != "" {
		t.Errorf("no data should print on an API error, got %q", stdout)
	}
	if !strings.Contains(stderr, "404") {
		t.Errorf("stderr should name the HTTP status:\n%s", stderr)
	}
}

func TestRunDomains_TransportErrorIsNetworkUnavailable(t *testing.T) {
	tr := &cannedTransport{netErr: errors.New("dial tcp: connection refused")}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}
	outcome, _, stderr := runDomainsOver(t, seam, domainsConfig{})
	if outcome != NetworkUnavailable {
		t.Fatalf("outcome = %v, want NetworkUnavailable", outcome)
	}
	if strings.TrimSpace(stderr) == "" {
		t.Error("a transport failure should be named on stderr")
	}
}

func TestRunDomains_NoTokenIsUsageError(t *testing.T) {
	seam := &fakeMeSeam{ctx: noTokenContext(), transport: &cannedTransport{status: 200, body: domainsPageComplete}}
	outcome, stdout, stderr := runDomainsOver(t, seam, domainsConfig{})
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

// --- structured output ------------------------------------------------------

func TestRunDomains_StructuredEmitsRawPayload(t *testing.T) {
	tr := &cannedTransport{status: 200, body: domainsPageComplete}
	seam := &fakeMeSeam{ctx: validMeContext(), envOutput: "json", transport: tr}
	outcome, stdout, stderr := runDomainsOver(t, seam, domainsConfig{})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success\nstderr: %s", outcome, stderr)
	}
	for _, want := range []string{`"data"`, "dom_budget"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("structured output should emit the raw payload; missing %q:\n%s", want, stdout)
		}
	}
}

func TestRunDomains_PerPageSizesTheWalk(t *testing.T) {
	tr := &cannedTransport{status: 200, body: domainsPageComplete}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}
	outcome, _, stderr := runDomainsOver(t, seam, domainsConfig{perPage: 50, perPageSet: true})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success\nstderr: %s", outcome, stderr)
	}
	if got := tr.lastQuery.Get("per_page"); got != "50" {
		t.Errorf("per_page query = %q, want 50", got)
	}
}
