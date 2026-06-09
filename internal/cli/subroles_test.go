package cli

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
)

// --- subroles fixtures -----------------------------------------------------

// subrolesPageComplete is a single complete page of two child RoleDetails.
const subrolesPageComplete = `{
  "data": [
    {"id": "role_press", "type": "role", "name": "Press Officer", "purpose": "Press that lands",
     "parent_role_id": "role_0123", "has_subroles": false, "flags": [],
     "policies": [{"id": "pol_1", "title": "All PRs require two approvals", "body": "b"}]},
    {"id": "role_events", "type": "role", "name": "Events Lead", "purpose": null,
     "parent_role_id": "role_0123", "has_subroles": true, "flags": []}
  ],
  "meta": {"pagination": {"per_page": 100, "has_next_page": false, "next_cursor": ""}}
}`

// subrolesPageEmpty is a leaf role's empty child set.
const subrolesPageEmpty = `{"data": [], "meta": {"pagination": {"per_page": 100, "has_next_page": false, "next_cursor": ""}}}`

// subrolesPage builds a one-child page for a named child, optionally signalling a
// next page with the given cursor — for assembling a multi-page walk.
func subrolesPage(id, name, nextCursor string) string {
	hasNext := "false"
	if nextCursor != "" {
		hasNext = "true"
	}
	return `{"data":[{"id":"` + id + `","type":"role","name":"` + name + `","purpose":"p",` +
		`"parent_role_id":"role_0123","has_subroles":false,"flags":[]}],` +
		`"meta":{"pagination":{"per_page":1,"has_next_page":` + hasNext + `,"next_cursor":"` + nextCursor + `"}}}`
}

// runSubrolesOver drives the pure runSubroles over a fake seam, returning the
// outcome and captured stdout/stderr, and failing if the token leaks.
func runSubrolesOver(t *testing.T, seam subrolesSeam, cfg subrolesConfig) (Outcome, string, string) {
	t.Helper()
	var out, errb bytes.Buffer
	cfg.seam = seam
	cfg.reqCtx = context.Background()
	cfg.stdout = &out
	cfg.stderr = &errb
	if cfg.id == "" {
		cfg.id = "role_0123"
	}
	outcome, _ := runSubroles(cfg)
	if strings.Contains(out.String()+errb.String(), meSecretToken) {
		t.Fatalf("the token leaked into output: %q", out.String()+errb.String())
	}
	return outcome, out.String(), errb.String()
}

// --- default walk -----------------------------------------------------------

func TestRunSubroles_ListSuccessWalksAndProjects(t *testing.T) {
	tr := &cannedTransport{status: 200, body: subrolesPageComplete}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, stderr := runSubrolesOver(t, seam, subrolesConfig{})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success\nstderr: %s", outcome, stderr)
	}
	if got := tr.lastPath; !strings.HasSuffix(got, "/roles/role_0123/subroles") {
		t.Errorf("path = %q, want /roles/role_0123/subroles", got)
	}
	for _, want := range []string{"Press Officer (role_press)", "Events Lead (role_events)"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("each child should print as a projection; missing %q:\n%s", want, stdout)
		}
	}
	if strings.TrimSpace(stderr) != "" {
		t.Errorf("a complete success writes nothing to stderr, got %q", stderr)
	}
}

func TestRunSubroles_LeafRolePrintsNoSubroles(t *testing.T) {
	tr := &cannedTransport{status: 200, body: subrolesPageEmpty}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, stderr := runSubrolesOver(t, seam, subrolesConfig{})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success", outcome)
	}
	if strings.TrimRight(stdout, "\n") != "No subroles." {
		t.Errorf("a leaf role should print exactly `No subroles.`, got %q", stdout)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Errorf("a leaf role is a clean success; stderr should be empty, got %q", stderr)
	}
}

func TestRunSubroles_WalksEveryPageToCompletion(t *testing.T) {
	tr := &seqMeTransport{steps: []seqMeResp{
		{status: 200, body: subrolesPage("role_p1", "Page One Child", "c1")},
		{status: 200, body: subrolesPage("role_p2", "Page Two Child", "c2")},
		{status: 200, body: subrolesPage("role_p3", "Page Three Child", "")},
	}}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, stderr := runSubrolesOver(t, seam, subrolesConfig{})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success\nstderr: %s", outcome, stderr)
	}
	if tr.calls != 3 {
		t.Errorf("the walk should issue three page requests, got %d", tr.calls)
	}
	for _, want := range []string{"Page One Child", "Page Two Child", "Page Three Child"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("every page's children should print, missing %q:\n%s", want, stdout)
		}
	}
}

// --- include ----------------------------------------------------------------

func TestRunSubroles_IncludeSendsParamAndEmbedsPerChild(t *testing.T) {
	tr := &cannedTransport{status: 200, body: subrolesPageComplete}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, stderr := runSubrolesOver(t, seam, subrolesConfig{
		include: []string{"policies"},
	})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success\nstderr: %s", outcome, stderr)
	}
	if got := tr.lastQuery.Get("include"); got != "policies" {
		t.Errorf("include = %q, want policies", got)
	}
	for _, want := range []string{"Policies:", "All PRs require two approvals"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("the requested include should embed inline per child; missing %q:\n%s", want, stdout)
		}
	}
}

func TestRunSubroles_UnsupportedIncludeRejectedBeforeRequest(t *testing.T) {
	tr := &cannedTransport{status: 200, body: subrolesPageComplete}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, stderr := runSubrolesOver(t, seam, subrolesConfig{include: []string{"members"}})
	if outcome != UsageError {
		t.Fatalf("outcome = %v, want UsageError", outcome)
	}
	if tr.calls != 0 {
		t.Errorf("an unsupported --include must send no request (tripwire), got %d", tr.calls)
	}
	if strings.TrimSpace(stdout) != "" {
		t.Errorf("no data should print, got %q", stdout)
	}
	// "members" is a TREE include value, not a subroles one — it must be rejected
	// here, naming the subroles set.
	if !strings.Contains(stderr, "members") {
		t.Errorf("stderr should name the unsupported value:\n%s", stderr)
	}
	for _, want := range []string{"assignments", "parent_role", "skills"} {
		if !strings.Contains(stderr, want) {
			t.Errorf("stderr should name the subroles include set; missing %q:\n%s", want, stderr)
		}
	}
}

// --- depth rejected on subroles --------------------------------------------

func TestRunSubroles_DepthRejectedNoRequest(t *testing.T) {
	tr := &cannedTransport{status: 200, body: subrolesPageComplete}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, stderr := runSubrolesOver(t, seam, subrolesConfig{depthSet: true})
	if outcome != UsageError {
		t.Fatalf("outcome = %v, want UsageError", outcome)
	}
	if tr.calls != 0 {
		t.Errorf("--depth on subroles must send no request (tripwire), got %d", tr.calls)
	}
	if strings.TrimSpace(stdout) != "" {
		t.Errorf("no data should print, got %q", stdout)
	}
	if !strings.Contains(stderr, "--depth") {
		t.Errorf("stderr should name the --depth misuse:\n%s", stderr)
	}
}

// --- first-page opt-out + completeness -------------------------------------

func TestRunSubroles_FirstPageStopsAtOnePageAndSignalsMore(t *testing.T) {
	tr := &cannedTransport{status: 200, body: subrolesPage("role_p1", "First Page Child", "c1")}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, stderr := runSubrolesOver(t, seam, subrolesConfig{firstPage: true})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success\nstderr: %s", outcome, stderr)
	}
	if tr.calls != 1 {
		t.Errorf("--first-page must not walk; want 1 call, got %d", tr.calls)
	}
	if !strings.Contains(stdout, "First Page Child") {
		t.Errorf("the first page should print:\n%s", stdout)
	}
	if !strings.Contains(stderr, "more subroles exist") {
		t.Errorf("stderr should note more subroles exist:\n%s", stderr)
	}
}

func TestRunSubroles_MidWalkFailureYieldsPartialFlaggedIncomplete(t *testing.T) {
	tr := &seqMeTransport{steps: []seqMeResp{
		{status: 200, body: subrolesPage("role_p1", "Gathered Child", "c1")},
		{status: 500, body: `{"detail":"boom"}`},
	}}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, stderr := runSubrolesOver(t, seam, subrolesConfig{})
	if outcome == Success {
		t.Fatalf("a mid-walk failure must exit non-zero, got Success")
	}
	if !strings.Contains(stdout, "Gathered Child") {
		t.Errorf("the partial set gathered so far should print:\n%s", stdout)
	}
	if !strings.Contains(stderr, "incomplete") {
		t.Errorf("stderr should note the result is incomplete and name the cause:\n%s", stderr)
	}
}

// --- error classification ---------------------------------------------------

func TestRunSubroles_UnknownIdSurfacesAPIStatus(t *testing.T) {
	tr := &cannedTransport{status: 404, body: `{"detail":"Role not found"}`}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}
	outcome, stdout, stderr := runSubrolesOver(t, seam, subrolesConfig{id: "role_ffff"})
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

func TestRunSubroles_TransportErrorIsNetworkUnavailable(t *testing.T) {
	tr := &cannedTransport{netErr: errors.New("dial tcp: connection refused")}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}
	outcome, _, stderr := runSubrolesOver(t, seam, subrolesConfig{})
	if outcome != NetworkUnavailable {
		t.Fatalf("outcome = %v, want NetworkUnavailable", outcome)
	}
	if strings.TrimSpace(stderr) == "" {
		t.Error("a transport failure should be named on stderr")
	}
}

func TestRunSubroles_NoTokenIsUsageError(t *testing.T) {
	seam := &fakeMeSeam{ctx: noTokenContext(), transport: &cannedTransport{status: 200, body: subrolesPageComplete}}
	outcome, stdout, stderr := runSubrolesOver(t, seam, subrolesConfig{})
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

func TestRunSubroles_StructuredEmitsRawPayload(t *testing.T) {
	tr := &cannedTransport{status: 200, body: subrolesPageComplete}
	seam := &fakeMeSeam{ctx: validMeContext(), envOutput: "json", transport: tr}
	outcome, stdout, stderr := runSubrolesOver(t, seam, subrolesConfig{})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success\nstderr: %s", outcome, stderr)
	}
	for _, want := range []string{`"data"`, "role_press"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("structured output should emit the raw payload; missing %q:\n%s", want, stdout)
		}
	}
}

func TestRunSubroles_PerPageSizesTheWalk(t *testing.T) {
	tr := &cannedTransport{status: 200, body: subrolesPageComplete}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}
	outcome, _, stderr := runSubrolesOver(t, seam, subrolesConfig{perPage: 50, perPageSet: true})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success\nstderr: %s", outcome, stderr)
	}
	if got := tr.lastQuery.Get("per_page"); got != "50" {
		t.Errorf("per_page query = %q, want 50", got)
	}
}
