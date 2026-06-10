package cli

import (
	"bytes"
	"context"
	"errors"
	"strconv"
	"strings"
	"testing"

	"github.com/Luscii/cli-glassfrog/internal/apiclient"
	"github.com/Luscii/cli-glassfrog/internal/auth"
	"github.com/Luscii/cli-glassfrog/internal/output"
	"github.com/spf13/cobra"
)

// Canned GET /roles bodies for the org-wide list tests. They carry the grown
// Role shape (type/parent_role_id/has_subroles/flags/fillers/tags) in the API's
// snake_case names, and the secret token nowhere (it rides the request header,
// asserted absent from output by runRolesOver).
const (
	orgRolesPageComplete = `{
      "data": [
        {"id": "role_0123456789abcdef0123456789abcdef", "type": "role", "name": "Marketing Lead",
         "purpose": "A market that knows us", "parent_role_id": "role_aaaa000000000000000000000000aaaa",
         "has_subroles": true, "flags": ["structural"],
         "domains": [{"id": "dom_1", "description": "The marketing budget"}],
         "accountabilities": [{"id": "acct_1", "description": "Defining the campaign"}],
         "fillers": [{"id": "per_x", "name": "Alice Smith", "kind": "human"}], "tags": ["marketing"]},
        {"id": "role_00000000000000000000000000000001", "type": "role", "name": "Anchor Circle",
         "purpose": null, "parent_role_id": null, "has_subroles": false, "flags": [],
         "domains": [], "accountabilities": [], "fillers": [], "tags": []}
      ],
      "meta": {"pagination": {"per_page": 500, "has_next_page": false, "next_cursor": ""}}
    }`

	orgRolesPageEmpty = `{"data": [], "meta": {"pagination": {"per_page": 500, "has_next_page": false, "next_cursor": ""}}}`
)

// orgRolesPage builds a one-role page body for a named role, optionally signalling
// a next page with the given cursor — for assembling a multi-page walk over the
// seqMeTransport.
func orgRolesPage(id, name, nextCursor string) string {
	hasNext := "false"
	if nextCursor != "" {
		hasNext = "true"
	}
	return `{"data":[{"id":"` + id + `","type":"role","name":"` + name + `","purpose":"p",` +
		`"parent_role_id":null,"has_subroles":false,"flags":[],"domains":[],"accountabilities":[],"fillers":[],"tags":[]}],` +
		`"meta":{"pagination":{"per_page":1,"has_next_page":` + hasNext + `,"next_cursor":"` + nextCursor + `"}}}`
}

// runRolesOver drives the pure runRoles over a fake seam, returning the outcome
// and captured stdout/stderr, and failing if the token leaks.
func runRolesOver(t *testing.T, seam rolesSeam, cfg rolesConfig) (Outcome, string, string) {
	t.Helper()
	var out, errb bytes.Buffer
	cfg.seam = seam
	cfg.reqCtx = context.Background()
	cfg.stdout = &out
	cfg.stderr = &errb
	outcome, _ := runRoles(cfg)
	if strings.Contains(out.String()+errb.String(), meSecretToken) {
		t.Fatalf("the token leaked into output: %q", out.String()+errb.String())
	}
	return outcome, out.String(), errb.String()
}

// --- runRolesList branches -------------------------------------------------

func TestRunRoles_ListSuccessWalksAndProjects(t *testing.T) {
	tr := &cannedTransport{status: 200, body: orgRolesPageComplete}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, stderr := runRolesOver(t, seam, rolesConfig{})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success\nstderr: %s", outcome, stderr)
	}
	for _, want := range []string{
		"Marketing Lead (role_0123456789abcdef0123456789abcdef)",
		"Purpose: A market that knows us",
		"The marketing budget", "Defining the campaign",
		"Anchor Circle", "(no purpose set)",
	} {
		if !strings.Contains(stdout, want) {
			t.Errorf("stdout missing %q:\n%s", want, stdout)
		}
	}
	if strings.TrimSpace(stderr) != "" {
		t.Errorf("a complete success should write nothing to stderr, got %q", stderr)
	}
	if tr.calls != 1 {
		t.Errorf("a single complete page should be one call, got %d", tr.calls)
	}
	if got := tr.lastPath; !strings.HasSuffix(got, "/roles") {
		t.Errorf("path = %q, want it to target /roles", got)
	}
}

func TestRunRoles_ListEmptyIsCleanSuccess(t *testing.T) {
	tr := &cannedTransport{status: 200, body: orgRolesPageEmpty}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, stderr := runRolesOver(t, seam, rolesConfig{})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success", outcome)
	}
	if strings.TrimRight(stdout, "\n") != "No roles." {
		t.Errorf("an empty org should print exactly `No roles.`, got %q", stdout)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Errorf("an empty list is a clean success; stderr should be empty, got %q", stderr)
	}
}

func TestRunRoles_ListWalksEveryPageToCompletion(t *testing.T) {
	tr := &seqMeTransport{steps: []seqMeResp{
		{status: 200, body: orgRolesPage("role_00000000000000000000000000000001", "Page One Role", "c1")},
		{status: 200, body: orgRolesPage("role_00000000000000000000000000000002", "Page Two Role", "c2")},
		{status: 200, body: orgRolesPage("role_00000000000000000000000000000003", "Page Three Role", "")},
	}}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, stderr := runRolesOver(t, seam, rolesConfig{})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success\nstderr: %s", outcome, stderr)
	}
	for _, want := range []string{"Page One Role", "Page Two Role", "Page Three Role"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("stdout missing %q (the walk must concatenate every page):\n%s", want, stdout)
		}
	}
	if tr.calls != 3 {
		t.Errorf("a three-page walk should be three calls, got %d", tr.calls)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Errorf("a completed walk writes nothing to stderr, got %q", stderr)
	}
}

func TestRunRoles_ListNoCredentialsIsUsageErrorNoRequest(t *testing.T) {
	tr := &cannedTransport{status: 200, body: orgRolesPageComplete}
	ctx := apiclient.ConnectionContext{
		BaseURL: apiclient.BaseURL{Value: "https://example.test/api/v5", Source: apiclient.SourceFlag},
		Cred:    auth.Resolution{Source: auth.SourceNone},
	}
	seam := &fakeMeSeam{ctx: ctx, transport: tr}

	outcome, stdout, stderr := runRolesOver(t, seam, rolesConfig{})
	if outcome != UsageError {
		t.Fatalf("outcome = %v, want UsageError", outcome)
	}
	if !strings.Contains(stderr, "auth login") {
		t.Errorf("stderr should point at `glassfrog auth login`, got %q", stderr)
	}
	if strings.TrimSpace(stdout) != "" {
		t.Errorf("no role data should print on a no-token failure, got %q", stdout)
	}
	if tr.calls != 0 {
		t.Errorf("an unauthenticated request must not be sent, got %d calls", tr.calls)
	}
}

func TestRunRoles_ListTransportFailureIsNetworkUnavailable(t *testing.T) {
	tr := &cannedTransport{netErr: errors.New("dial tcp: connection refused")}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, stderr := runRolesOver(t, seam, rolesConfig{})
	if outcome != NetworkUnavailable {
		t.Fatalf("outcome = %v, want NetworkUnavailable", outcome)
	}
	if strings.TrimSpace(stdout) != "" {
		t.Errorf("no projection should print on a first-page transport failure, got %q", stdout)
	}
	if strings.TrimSpace(stderr) == "" {
		t.Error("a transport failure should report a cause on stderr")
	}
}

func TestRunRoles_ListNon2xxIsAPIError(t *testing.T) {
	tr := &cannedTransport{status: 500, body: `{"error":"server error"}`}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, stderr := runRolesOver(t, seam, rolesConfig{})
	if outcome != APIError {
		t.Fatalf("outcome = %v, want APIError", outcome)
	}
	if strings.TrimSpace(stdout) != "" {
		t.Errorf("no projection should print on a first-page non-2xx, got %q", stdout)
	}
	if !strings.Contains(stderr, "500") {
		t.Errorf("stderr should name the 500 status, got %q", stderr)
	}
}

// 031 ADR-2: an undecodable 2xx body now classifies as APIError (exit 3), not
// RuntimeError (exit 1); the cause/next-step message is unchanged.
func TestRunRoles_ListUndecodableBodyIsAPIError(t *testing.T) {
	tr := &cannedTransport{status: 200, body: `not json at all`}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, _ := runRolesOver(t, seam, rolesConfig{})
	if outcome != APIError {
		t.Fatalf("outcome = %v, want APIError", outcome)
	}
	if strings.TrimSpace(stdout) != "" {
		t.Errorf("no projection should print on a decode failure, got %q", stdout)
	}
}

func TestRunRoles_ListBaseURLErrorIsUsageErrorNothingSent(t *testing.T) {
	tr := &cannedTransport{status: 200, body: orgRolesPageComplete}
	seam := &fakeMeSeam{
		ctx:          apiclient.ConnectionContext{},
		newClientErr: &apiclient.BaseURLError{Source: "--" + apiclient.FlagBaseURL},
		transport:    tr,
	}

	outcome, _, stderr := runRolesOver(t, seam, rolesConfig{})
	if outcome != UsageError {
		t.Fatalf("outcome = %v, want UsageError", outcome)
	}
	if !strings.Contains(strings.ToLower(stderr), "base-url") && !strings.Contains(strings.ToLower(stderr), "base url") {
		t.Errorf("stderr should name the base-URL problem, got %q", stderr)
	}
	if tr.calls != 0 {
		t.Errorf("a base-URL error must not send, got %d calls", tr.calls)
	}
}

func TestRunRoles_ListInvalidOutputIsUsageErrorNothingSent(t *testing.T) {
	tr := &cannedTransport{status: 200, body: orgRolesPageComplete}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr, envOutput: "bogus"}

	outcome, _, stderr := runRolesOver(t, seam, rolesConfig{})
	if outcome != UsageError {
		t.Fatalf("outcome = %v, want UsageError", outcome)
	}
	if !strings.Contains(stderr, "bogus") {
		t.Errorf("stderr should name the bad format value, got %q", stderr)
	}
	if tr.calls != 0 {
		t.Errorf("an invalid --output must not send, got %d calls", tr.calls)
	}
}

// -o json emits the aggregated {data:[…]} document with each role's raw bytes
// preserved (no human projection, no per-page meta — completeness rides stderr).
func TestRunRoles_ListStructuredEmitsAggregatedData(t *testing.T) {
	tr := &cannedTransport{status: 200, body: orgRolesPageComplete}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr, envOutput: "json"}

	outcome, stdout, _ := runRolesOver(t, seam, rolesConfig{})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success", outcome)
	}
	for _, want := range []string{`"data"`, "Marketing Lead", "Anchor Circle", `"has_subroles"`} {
		if !strings.Contains(stdout, want) {
			t.Errorf("structured output should carry the role data field %q:\n%s", want, stdout)
		}
	}
	// The synthesized envelope drops per-page pagination meta (an aggregate has no
	// single page's meta); completeness is signalled on stderr instead.
	if strings.Contains(stdout, `"meta"`) || strings.Contains(stdout, "has_next_page") {
		t.Errorf("the aggregated document should not carry per-page meta:\n%s", stdout)
	}
	// The human projection header must not appear under a structured format.
	if strings.Contains(stdout, "Purpose:") {
		t.Errorf("structured output must not render the human projection:\n%s", stdout)
	}
}

// json and human fetch the SAME set: a structured default read walks every page
// to completion, just like the human default — the format changes rendering, not
// fetch depth.
func TestRunRoles_ListStructuredWalksEveryPage(t *testing.T) {
	tr := &seqMeTransport{steps: []seqMeResp{
		{status: 200, body: orgRolesPage("role_00000000000000000000000000000001", "Page One Role", "c1")},
		{status: 200, body: orgRolesPage("role_00000000000000000000000000000002", "Page Two Role", "c2")},
		{status: 200, body: orgRolesPage("role_00000000000000000000000000000003", "Page Three Role", "")},
	}}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr, envOutput: "json"}

	outcome, stdout, stderr := runRolesOver(t, seam, rolesConfig{})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success\nstderr: %s", outcome, stderr)
	}
	for _, want := range []string{"Page One Role", "Page Two Role", "Page Three Role"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("structured output must include every page's roles, missing %q:\n%s", want, stdout)
		}
	}
	if tr.calls != 3 {
		t.Errorf("the structured walk should issue three requests, got %d", tr.calls)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Errorf("a completed structured walk writes nothing to stderr, got %q", stderr)
	}
}

// A mid-walk failure under -o json emits the partial set as a valid document,
// flags incompleteness on stderr, and exits non-zero — identical signalling to
// the human path.
func TestRunRoles_ListStructuredMidWalkFailureIsPartialAndIncomplete(t *testing.T) {
	tr := &seqMeTransport{steps: []seqMeResp{
		{status: 200, body: orgRolesPage("role_00000000000000000000000000000001", "Gathered Role", "c1")},
		{status: 500, body: `{"detail":"boom"}`},
	}}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr, envOutput: "json"}

	outcome, stdout, stderr := runRolesOver(t, seam, rolesConfig{})
	if outcome != APIError {
		t.Fatalf("outcome = %v, want APIError", outcome)
	}
	if !strings.Contains(stdout, "Gathered Role") || !strings.Contains(stdout, `"data"`) {
		t.Errorf("the partial set should be emitted as a structured document:\n%s", stdout)
	}
	if !strings.Contains(stderr, "incomplete") {
		t.Errorf("stderr should flag the structured result incomplete, got %q", stderr)
	}
}

// An empty org under -o json emits a valid empty list ({"data":[]}), not null.
func TestRunRoles_ListStructuredEmptyIsEmptyArray(t *testing.T) {
	tr := &cannedTransport{status: 200, body: orgRolesPageEmpty}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr, envOutput: "json"}

	outcome, stdout, _ := runRolesOver(t, seam, rolesConfig{})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success", outcome)
	}
	if !strings.Contains(stdout, `"data": []`) {
		t.Errorf("an empty org should emit {\"data\": []}, got:\n%s", stdout)
	}
}

// --- list filters, --first-page, --per-page, completeness (T003) -----------

func TestRunRoles_FilterParentSendsParamAndProjects(t *testing.T) {
	tr := &cannedTransport{status: 200, body: orgRolesPageComplete}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, _, stderr := runRolesOver(t, seam, rolesConfig{parent: "role_aaaa000000000000000000000000aaaa"})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success\nstderr: %s", outcome, stderr)
	}
	if got := tr.lastQuery.Get("parent_role_id"); got != "role_aaaa000000000000000000000000aaaa" {
		t.Errorf("parent_role_id = %q, want the parent id", got)
	}
}

func TestRunRoles_PersonAndTagFiltersSendParams(t *testing.T) {
	tr := &cannedTransport{status: 200, body: orgRolesPageComplete}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	_, _, _ = runRolesOver(t, seam, rolesConfig{person: "per_x", tag: "marketing"})
	if got := tr.lastQuery.Get("person_id"); got != "per_x" {
		t.Errorf("person_id = %q, want per_x", got)
	}
	if got := tr.lastQuery.Get("tag"); got != "marketing" {
		t.Errorf("tag = %q, want marketing", got)
	}
}

func TestRunRoles_HasSubrolesIsTriState(t *testing.T) {
	tru, fls := true, false
	cases := []struct {
		name    string
		ptr     *bool
		wantKey bool
		wantVal string
	}{
		{"omitted-sends-nothing", nil, false, ""},
		{"true-sends-true", &tru, true, "true"},
		{"false-sends-false", &fls, true, "false"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tr := &cannedTransport{status: 200, body: orgRolesPageComplete}
			seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}
			_, _, _ = runRolesOver(t, seam, rolesConfig{hasSubroles: tc.ptr})
			_, present := tr.lastQuery["has_subroles"]
			if present != tc.wantKey {
				t.Errorf("has_subroles present = %v, want %v (omitted ≠ false)", present, tc.wantKey)
			}
			if tc.wantKey && tr.lastQuery.Get("has_subroles") != tc.wantVal {
				t.Errorf("has_subroles = %q, want %q", tr.lastQuery.Get("has_subroles"), tc.wantVal)
			}
		})
	}
}

// A filter combined with a role id is a usage error with NO request sent
// (transport tripwire).
func TestRunRoles_FilterWithIdIsUsageErrorNoRequest(t *testing.T) {
	tr := &cannedTransport{status: 200, body: orgRolesPageComplete}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, _, stderr := runRolesOver(t, seam, rolesConfig{args: []string{"role_0123"}, tag: "marketing"})
	if outcome != UsageError {
		t.Fatalf("outcome = %v, want UsageError", outcome)
	}
	if !strings.Contains(stderr, "--tag") {
		t.Errorf("stderr should name the offending --tag flag, got %q", stderr)
	}
	if tr.calls != 0 {
		t.Errorf("a rejected combination must send nothing (tripwire), got %d calls", tr.calls)
	}
	if seam.assembleCalled {
		t.Errorf("validation must run before assembly, assembled=%v", seam.assembleCalled)
	}
}

// --first-page against a multi-page org prints only the first page, writes the
// "more roles exist" note, and exits 0 — one request, no walk.
func TestRunRoles_FirstPageStopsAndSignals(t *testing.T) {
	tr := &cannedTransport{status: 200, body: orgRolesPage("role_00000000000000000000000000000001", "Only Page Shown", "c1")}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, stderr := runRolesOver(t, seam, rolesConfig{firstPage: true})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success (the opt-out is not an error)", outcome)
	}
	if !strings.Contains(stdout, "Only Page Shown") {
		t.Errorf("the first page should print, got %q", stdout)
	}
	if !strings.Contains(stderr, "more roles exist") {
		t.Errorf("stderr should note more roles exist, got %q", stderr)
	}
	if tr.calls != 1 {
		t.Errorf("--first-page must not walk, want 1 call, got %d", tr.calls)
	}
}

// --first-page on a single-page org prints the page and writes NO note (exit 0).
func TestRunRoles_FirstPageNoNoteWhenComplete(t *testing.T) {
	tr := &cannedTransport{status: 200, body: orgRolesPageComplete}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, _, stderr := runRolesOver(t, seam, rolesConfig{firstPage: true})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success", outcome)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Errorf("a single-page first-page run writes no note, got %q", stderr)
	}
}

// A mid-walk failure renders the partial set, writes the incomplete note naming
// the cause, and exits non-zero (classified from the Stop error).
func TestRunRoles_MidWalkFailureYieldsPartialFlaggedIncomplete(t *testing.T) {
	tr := &seqMeTransport{steps: []seqMeResp{
		{status: 200, body: orgRolesPage("role_00000000000000000000000000000001", "Gathered Role", "c1")},
		{status: 500, body: `{"detail":"boom"}`},
	}}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, stderr := runRolesOver(t, seam, rolesConfig{})
	if outcome != APIError {
		t.Fatalf("outcome = %v, want APIError (the walk stopped on a 500)", outcome)
	}
	if !strings.Contains(stdout, "Gathered Role") {
		t.Errorf("the partial set gathered so far should print, got %q", stdout)
	}
	if !strings.Contains(stderr, "incomplete") || !strings.Contains(stderr, "partial set") {
		t.Errorf("stderr should flag the result incomplete and name a partial set, got %q", stderr)
	}
	if tr.calls != 2 {
		t.Errorf("the walk should issue two requests before stopping, got %d", tr.calls)
	}
}

func TestRunRoles_PerPageSizesTheWalk(t *testing.T) {
	tr := &cannedTransport{status: 200, body: orgRolesPageComplete}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	_, _, _ = runRolesOver(t, seam, rolesConfig{perPage: 10, perPageSet: true})
	if got := tr.lastQuery.Get("per_page"); got != "10" {
		t.Errorf("per_page = %q, want 10 (--per-page sizes the walk)", got)
	}
}

// --per-page is keyed on presence, not value: a provided 0 or negative value is
// passed through to the API as-is (no client-side clamp) rather than silently
// ignored, so an out-of-range value surfaces the API's rejection.
func TestRunRoles_PerPageProvidedValuePassedThrough(t *testing.T) {
	for _, n := range []int{0, -1} {
		tr := &cannedTransport{status: 200, body: orgRolesPageComplete}
		seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}
		_, _, _ = runRolesOver(t, seam, rolesConfig{perPage: n, perPageSet: true})
		if got := tr.lastQuery.Get("per_page"); got != strconv.Itoa(n) {
			t.Errorf("per_page = %q, want %d (a provided value is passed through, not ignored)", got, n)
		}
	}
}

// --per-page combined with a role id is a usage error keyed on flag presence, so
// even --per-page=0 (the int zero value) alongside an id is rejected with no
// request sent (the perPageSet=Changed fix; a value-based `!= 0` check would miss it).
func TestRunRoles_PerPageWithIdIsUsageErrorNoRequest(t *testing.T) {
	for name, n := range map[string]int{"zero": 0, "positive": 25, "negative": -1} {
		t.Run(name, func(t *testing.T) {
			tr := &cannedTransport{status: 200, body: roleDetailBody}
			seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}
			outcome, _, stderr := runRolesOver(t, seam, rolesConfig{
				args: []string{"role_0123456789abcdef0123456789abcdef"}, perPage: n, perPageSet: true,
			})
			if outcome != UsageError {
				t.Fatalf("outcome = %v, want UsageError", outcome)
			}
			if !strings.Contains(stderr, "--per-page") {
				t.Errorf("stderr should name the offending --per-page flag, got %q", stderr)
			}
			if tr.calls != 0 {
				t.Errorf("a rejected combination must send nothing (tripwire), got %d calls", tr.calls)
			}
		})
	}
}

// A filter that matches nothing prints `No roles.` and exits 0.
func TestRunRoles_NoMatchFilterIsCleanSuccess(t *testing.T) {
	tr := &cannedTransport{status: 200, body: orgRolesPageEmpty}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, _ := runRolesOver(t, seam, rolesConfig{tag: "no-such-tag"})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success", outcome)
	}
	if strings.TrimRight(stdout, "\n") != "No roles." {
		t.Errorf("a no-match filter should print `No roles.`, got %q", stdout)
	}
}

// --- single-role read (T004) ----------------------------------------------

// roleDetailBody is a GET /roles/{id} body with policies + subroles embedded
// (as ?include=policies,subroles would return) plus the base role fields.
const roleDetailBody = `{
  "data": {
    "id": "role_0123456789abcdef0123456789abcdef", "type": "role", "name": "Marketing Lead",
    "purpose": "A market that knows us", "parent_role_id": null, "has_subroles": true, "flags": [],
    "domains": [{"id": "dom_1", "description": "The marketing budget"}],
    "accountabilities": [{"id": "acct_1", "description": "Defining the campaign"}],
    "fillers": [{"id": "per_x", "name": "Alice Smith", "kind": "human"}], "tags": [],
    "policies": [{"id": "pol_1", "title": "All PRs require two approvals", "body": "b"}],
    "subroles": [{"id": "role_sub00000000000000000000000000sub", "type": "role", "name": "Press Officer"}]
  }
}`

func TestRunRoles_SingleReadProjectsBaseFields(t *testing.T) {
	tr := &cannedTransport{status: 200, body: roleDetailBody}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, stderr := runRolesOver(t, seam, rolesConfig{args: []string{"role_0123456789abcdef0123456789abcdef"}})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success\nstderr: %s", outcome, stderr)
	}
	for _, want := range []string{
		"Marketing Lead (role_0123456789abcdef0123456789abcdef)",
		"Purpose: A market that knows us",
		"The marketing budget", "Defining the campaign",
		"Fillers:", "Alice Smith (per_x)",
	} {
		if !strings.Contains(stdout, want) {
			t.Errorf("stdout missing %q:\n%s", want, stdout)
		}
	}
	if got := tr.lastPath; !strings.HasSuffix(got, "/roles/role_0123456789abcdef0123456789abcdef") {
		t.Errorf("path = %q, want it to target /roles/{id}", got)
	}
	// No --include: an unrequested section is omitted entirely (not even a marker).
	if strings.Contains(stdout, "Policies:") || strings.Contains(stdout, "Subroles:") {
		t.Errorf("unrequested include sections must be omitted:\n%s", stdout)
	}
}

func TestRunRoles_SingleReadIncludeEmbedsInlineAndSendsParam(t *testing.T) {
	tr := &cannedTransport{status: 200, body: roleDetailBody}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, stderr := runRolesOver(t, seam, rolesConfig{
		args:    []string{"role_0123456789abcdef0123456789abcdef"},
		include: []string{"policies", "subroles"},
	})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success\nstderr: %s", outcome, stderr)
	}
	if got := tr.lastQuery.Get("include"); got != "policies,subroles" {
		t.Errorf("include = %q, want comma-joined policies,subroles", got)
	}
	for _, want := range []string{"Policies:", "All PRs require two approvals", "Subroles:", "Press Officer"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("requested include should render inline; missing %q:\n%s", want, stdout)
		}
	}
}

// A requested-but-empty include renders its explicit-absence marker; an
// unrequested sibling is omitted entirely.
func TestRunRoles_SingleReadRequestedEmptyIncludeShowsMarker(t *testing.T) {
	body := `{"data":{"id":"role_0123456789abcdef0123456789abcdef","type":"role","name":"Bare","purpose":"p","parent_role_id":null,"has_subroles":false,"flags":[],"domains":[],"accountabilities":[],"fillers":[],"tags":[],"policies":[]}}`
	tr := &cannedTransport{status: 200, body: body}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, _ := runRolesOver(t, seam, rolesConfig{
		args:    []string{"role_0123456789abcdef0123456789abcdef"},
		include: []string{"policies"},
	})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success", outcome)
	}
	if !strings.Contains(stdout, "Policies:") || !strings.Contains(stdout, "(none)") {
		t.Errorf("a requested-but-empty include should show its section with (none):\n%s", stdout)
	}
	if strings.Contains(stdout, "Notes:") || strings.Contains(stdout, "Skills:") {
		t.Errorf("unrequested sections must stay omitted:\n%s", stdout)
	}
}

// An unsupported --include value is a usage error naming the value and the
// supported set, with NO request sent (transport tripwire).
func TestRunRoles_UnsupportedIncludeRejectedBeforeAnyRequest(t *testing.T) {
	tr := &cannedTransport{status: 200, body: roleDetailBody}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, _, stderr := runRolesOver(t, seam, rolesConfig{
		args:    []string{"role_0123456789abcdef0123456789abcdef"},
		include: []string{"nonsense"},
	})
	if outcome != UsageError {
		t.Fatalf("outcome = %v, want UsageError", outcome)
	}
	if !strings.Contains(stderr, "nonsense") || !strings.Contains(stderr, "policies") {
		t.Errorf("stderr should name the bad value and the supported set, got %q", stderr)
	}
	if tr.calls != 0 {
		t.Errorf("an unsupported include must send nothing (tripwire), got %d calls", tr.calls)
	}
	if seam.assembleCalled {
		t.Errorf("include validation must run before assembly, assembled=%v", seam.assembleCalled)
	}
}

// An unknown role id is passed through to the API and surfaces as its status —
// the id is NOT validated locally (ADR-4).
func TestRunRoles_UnknownIdSurfacesAPIStatus(t *testing.T) {
	tr := &cannedTransport{status: 404, body: `{"detail":"Role not found"}`}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, stderr := runRolesOver(t, seam, rolesConfig{args: []string{"role_ffffffffffffffffffffffffffffffff"}})
	if outcome != APIError {
		t.Fatalf("outcome = %v, want APIError", outcome)
	}
	if strings.TrimSpace(stdout) != "" {
		t.Errorf("no projection should print on an unknown id, got %q", stdout)
	}
	if !strings.Contains(stderr, "404") {
		t.Errorf("stderr should name the 404 status, got %q", stderr)
	}
	if tr.calls != 1 {
		t.Errorf("the id should be passed through (one request), got %d calls", tr.calls)
	}
}

// The role id is escaped as a single path segment: a `/` in the id must not
// create extra path segments (endpoint redirection) — it is passed through
// unvalidated (ADR-4) but never allowed to change the targeted endpoint.
func TestRunRoles_SingleReadEscapesIdInPath(t *testing.T) {
	tr := &cannedTransport{status: 200, body: roleDetailBody}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	_, _, _ = runRolesOver(t, seam, rolesConfig{args: []string{"role_x/subroles"}})
	if strings.Contains(tr.lastPath, "roles/role_x/subroles") {
		t.Errorf("a `/` in the id must not create extra path segments (endpoint redirection): %q", tr.lastPath)
	}
	if !strings.Contains(tr.lastPath, "role_x%2Fsubroles") {
		t.Errorf("the id should be escaped as a single path segment, got %q", tr.lastPath)
	}
	// A valid role_… id is unaffected (PathEscape is a no-op for unreserved chars).
	tr2 := &cannedTransport{status: 200, body: roleDetailBody}
	seam2 := &fakeMeSeam{ctx: validMeContext(), transport: tr2}
	_, _, _ = runRolesOver(t, seam2, rolesConfig{args: []string{"role_0123456789abcdef0123456789abcdef"}})
	if !strings.HasSuffix(tr2.lastPath, "/roles/role_0123456789abcdef0123456789abcdef") {
		t.Errorf("a valid id should pass through unchanged, got %q", tr2.lastPath)
	}
}

func TestRunRoles_SingleReadStructuredEmitsRawPayload(t *testing.T) {
	tr := &cannedTransport{status: 200, body: roleDetailBody}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr, envOutput: "json"}

	outcome, stdout, _ := runRolesOver(t, seam, rolesConfig{
		args:    []string{"role_0123456789abcdef0123456789abcdef"},
		include: []string{"policies"},
	})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success", outcome)
	}
	if !strings.Contains(stdout, `"data"`) || !strings.Contains(stdout, "All PRs require two approvals") {
		t.Errorf("structured single read should emit the raw {data:…} payload:\n%s", stdout)
	}
	if strings.Contains(stdout, "Purpose:") {
		t.Errorf("structured output must not render the human projection:\n%s", stdout)
	}
}

// --- validateRolesInclude (pure) -------------------------------------------

func TestValidateRolesInclude(t *testing.T) {
	if err := validateRolesInclude(nil); err != nil {
		t.Errorf("absent --include is valid, got %v", err)
	}
	if err := validateRolesInclude([]string{"policies", "subroles", "parent_role", "notes", "skills", "assignments"}); err != nil {
		t.Errorf("all six supported values should pass, got %v", err)
	}
	err := validateRolesInclude([]string{"nonsense"})
	if err == nil || !strings.Contains(err.Error(), `"nonsense"`) {
		t.Errorf("an unsupported value should be quoted in the error, got %v", err)
	}
	if !strings.Contains(err.Error(), "value ") || strings.Contains(err.Error(), "values ") {
		t.Errorf("a single bad value should use the singular noun, got %q", err.Error())
	}
	multi := validateRolesInclude([]string{"bogus", "nope"})
	if multi == nil || !strings.Contains(multi.Error(), "values ") {
		t.Errorf("multiple bad values should use the plural noun, got %v", multi)
	}
}

// reportIncompleteWalk must return the REFINED error (not the original Stop), so
// the returned value agrees with the classified outcome and a downstream
// errors.As sees the extracted *ProblemError on a mid-walk non-2xx.
func TestReportIncompleteWalk_ReturnsRefinedError(t *testing.T) {
	var errb bytes.Buffer
	stop := &apiclient.ResponseError{StatusCode: 403, Body: []byte(`{"detail":"Forbidden"}`)}
	outcome, retErr := reportIncompleteWalk(&errb, stop)
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

// --- validateRolesFlags (pure) ---------------------------------------------

func TestValidateRolesFlags(t *testing.T) {
	if err := validateRolesFlags(false, rolesFlagState{}); err != nil {
		t.Errorf("a bare list is valid, got %v", err)
	}
	if err := validateRolesFlags(true, rolesFlagState{}); err != nil {
		t.Errorf("a bare single read is valid, got %v", err)
	}
	// A list filter with an id is a usage error naming the misuse.
	err := validateRolesFlags(true, rolesFlagState{tagSet: true})
	if err == nil || !strings.Contains(err.Error(), "--tag") {
		t.Errorf("a filter with an id should be rejected naming --tag, got %v", err)
	}
	// --include without an id is a usage error.
	err = validateRolesFlags(false, rolesFlagState{includeSet: true})
	if err == nil || !strings.Contains(err.Error(), "--include") {
		t.Errorf("--include without an id should be rejected, got %v", err)
	}
}

// --- newRolesCommand integration (outcome → exit code, wiring) -------------

// runRolesCommand registers `roles` under a real root (with the persistent
// --base-url + --output flags) and dispatches `roles [args]` through Run.
func runRolesCommand(t *testing.T, seam rolesSeam, args ...string) (Outcome, int, string, string) {
	t.Helper()
	root := NewRootCommand()
	MustRegister(root, newRolesCommand(seam))
	var out, errb bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&errb)
	outcome, _ := Run(root, append([]string{"roles"}, args...))
	return outcome, ExitCode(outcome), out.String(), errb.String()
}

func TestRolesCommand_ExitCodesAcrossOutcomes(t *testing.T) {
	cases := []struct {
		name     string
		tr       *cannedTransport
		ctx      apiclient.ConnectionContext
		seamErr  error
		args     []string
		outcome  Outcome
		exitCode int
	}{
		{"list-success", &cannedTransport{status: 200, body: orgRolesPageComplete}, validMeContext(), nil, nil, Success, 0},
		{"list-empty-success", &cannedTransport{status: 200, body: orgRolesPageEmpty}, validMeContext(), nil, nil, Success, 0},
		{"api-error", &cannedTransport{status: 500, body: `{}`}, validMeContext(), nil, nil, APIError, 3},
		{"network-unavailable", &cannedTransport{netErr: errors.New("refused")}, validMeContext(), nil, nil, NetworkUnavailable, 6},
		{"decode-error", &cannedTransport{status: 200, body: `nope`}, validMeContext(), nil, nil, APIError, 3},
		{"too-many-args", &cannedTransport{status: 200, body: orgRolesPageComplete}, validMeContext(), nil, []string{"role_a", "role_b"}, UsageError, 2},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			seam := &fakeMeSeam{ctx: tc.ctx, newClientErr: tc.seamErr, transport: tc.tr}
			outcome, code, stdout, stderr := runRolesCommand(t, seam, tc.args...)
			if outcome != tc.outcome {
				t.Errorf("outcome = %v, want %v\nstderr: %s", outcome, tc.outcome, stderr)
			}
			if code != tc.exitCode {
				t.Errorf("exit code = %d, want %d", code, tc.exitCode)
			}
			if strings.Contains(stdout+stderr, meSecretToken) {
				t.Errorf("token leaked into output: %q", stdout+stderr)
			}
		})
	}
}

// More than one positional id is rejected by cobra.MaximumNArgs(1) before any
// API call.
func TestRolesCommand_TooManyArgsSendsNothing(t *testing.T) {
	tr := &cannedTransport{status: 200, body: orgRolesPageComplete}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}
	outcome, code, _, _ := runRolesCommand(t, seam, "role_a", "role_b")
	if outcome != UsageError || code != 2 {
		t.Fatalf("too many args: outcome=%v code=%d, want UsageError/2", outcome, code)
	}
	if tr.calls != 0 {
		t.Errorf("a rejected invocation must not send a request, got %d calls", tr.calls)
	}
}

// The persistent --base-url value reaches the seam's assemble (inherited from the
// root).
func TestRolesCommand_InheritsBaseURLFlag(t *testing.T) {
	tr := &cannedTransport{status: 200, body: orgRolesPageComplete}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}
	_, _, _, _ = runRolesCommand(t, seam, "--base-url", "https://flag.test/api/v5")
	if seam.assembledBaseURL != "https://flag.test/api/v5" {
		t.Errorf("assemble received base URL %q, want the inherited flag value", seam.assembledBaseURL)
	}
}

// The `roles` leaf declares no --base-url flag of its own — it inherits the
// root's persistent one.
func TestRolesCommand_DeclaresNoOwnBaseURLFlag(t *testing.T) {
	cmd := newRolesCommand(&fakeMeSeam{})
	if cmd.Flags().Lookup(apiclient.FlagBaseURL) != nil {
		t.Errorf("the roles leaf must not declare its own --%s flag; it is inherited", apiclient.FlagBaseURL)
	}
}

// `roles` is a runnable leaf (an optional positional id), NOT a group — it
// replaces the earlier stub `roles list`/`roles get` group.
func TestRolesCommand_IsRunnableLeafReplacingStub(t *testing.T) {
	cmd := newRolesCommand(&fakeMeSeam{})
	if cmd.RunE == nil {
		t.Error("roles should be a runnable leaf (RunE set)")
	}
	if len(cmd.Commands()) != 0 {
		t.Errorf("roles should have no subcommands (the list/get stubs are removed), got %v", cmd.Commands())
	}
	if cmd.Args == nil {
		t.Error("roles should declare an Args validator (MaximumNArgs(1))")
	}
}

// The full Assemble wiring must not panic and must wire `roles` as a top-level
// runnable command.
func TestAssemble_WiresRolesWithoutPanic(t *testing.T) {
	root := Assemble()
	rolesCmd, _, err := root.Find([]string{"roles"})
	if err != nil || rolesCmd == nil || rolesCmd.Name() != "roles" {
		t.Fatalf("Assemble should wire a top-level `roles` command, got %v (err %v)", rolesCmd, err)
	}
	if rolesCmd.RunE == nil {
		t.Error("the wired `roles` should be runnable")
	}
}

// Guard against an accidental coupling: the `roles` constructor takes a seam.
var _ = func() *cobra.Command { return newRolesCommand(productionSeam{}) }

// Sanity: the org-roles render key resolves in both formats (the registry guard
// in internal/render is the authority; this pins the cli-side resource constant).
func TestRolesCommand_OrgRolesRenderKeyExists(t *testing.T) {
	for _, f := range []output.OutputFormat{output.FormatFull, output.FormatCompact} {
		tr := &cannedTransport{status: 200, body: orgRolesPageComplete}
		seam := &fakeMeSeam{ctx: validMeContext(), transport: tr, envOutput: f.String()}
		outcome, stdout, stderr := runRolesOver(t, seam, rolesConfig{})
		if outcome != Success {
			t.Fatalf("format %s: outcome=%v stderr=%s", f, outcome, stderr)
		}
		if !strings.Contains(stdout, "role_0123456789abcdef0123456789abcdef") {
			t.Errorf("format %s should render the role id:\n%s", f, stdout)
		}
	}
}
