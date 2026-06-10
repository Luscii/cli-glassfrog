package cli

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
)

// --- canned GET /roles/{id}/policies bodies --------------------------------
//
// They carry the grown Policy shape (role_id/domain_id/created_at/updated_at) in
// the API's snake_case names, one role-level policy (null domain_id) to exercise
// the explicit-absence render, and the secret token nowhere.

const policiesPageComplete = `{"data":[
  {"id":"pol_1","title":"All PRs require two approvals","body":"Every PR needs two approvals","role_id":"role_0123","domain_id":"dom_1","created_at":"2024-01-01T00:00:00Z","updated_at":"2024-01-01T00:00:00Z"},
  {"id":"pol_2","title":"Spending limit","body":"Spend under 1k","role_id":"role_0123","domain_id":null,"created_at":"2024-02-01T00:00:00Z","updated_at":"2024-02-01T00:00:00Z"}
],"meta":{"pagination":{"per_page":100,"has_next_page":false,"next_cursor":""}}}`

const policiesPageEmpty = `{"data":[],"meta":{"pagination":{"per_page":100,"has_next_page":false,"next_cursor":""}}}`

// policyDocumentBody is a representative GET /policies/{id} body: the single-object
// {data: Policy} envelope carrying the full body and the grown scope/timestamps.
const policyDocumentBody = `{"data":{"id":"pol_0123","title":"All PRs require two approvals","body":"<p>Every PR needs <strong>two</strong> approvals.</p>","role_id":"role_0123","domain_id":"dom_1","created_at":"2024-01-02T03:04:05Z","updated_at":"2024-05-06T07:08:09Z"}}`

// policiesPage builds a one-policy page; a non-empty nextCursor marks more pages.
func policiesPage(id, title, nextCursor string) string {
	hasNext := "false"
	if nextCursor != "" {
		hasNext = "true"
	}
	return `{"data":[{"id":"` + id + `","title":"` + title + `","body":"b","role_id":"role_0123","domain_id":"dom_1","created_at":"2024-01-01T00:00:00Z","updated_at":"2024-01-01T00:00:00Z"}],` +
		`"meta":{"pagination":{"per_page":1,"has_next_page":` + hasNext + `,"next_cursor":"` + nextCursor + `"}}}`
}

// runPolicyOver drives the pure runPolicyGet over a fake seam, returning the
// outcome and captured stdout/stderr, and failing if the token leaks.
func runPolicyOver(t *testing.T, seam policiesSeam, cfg policyConfig, id string) (Outcome, string, string) {
	t.Helper()
	var out, errb bytes.Buffer
	cfg.seam = seam
	cfg.reqCtx = context.Background()
	cfg.stdout = &out
	cfg.stderr = &errb
	outcome, _ := runPolicyGet(cfg, id)
	if strings.Contains(out.String()+errb.String(), meSecretToken) {
		t.Fatalf("the token leaked into output: %q", out.String()+errb.String())
	}
	return outcome, out.String(), errb.String()
}

// runPoliciesOver drives the pure runPoliciesList over a fake seam, returning the
// outcome and captured stdout/stderr, and failing if the token leaks.
func runPoliciesOver(t *testing.T, seam policiesSeam, cfg policiesConfig) (Outcome, string, string) {
	t.Helper()
	var out, errb bytes.Buffer
	cfg.seam = seam
	cfg.reqCtx = context.Background()
	cfg.stdout = &out
	cfg.stderr = &errb
	outcome, _ := runPoliciesList(cfg)
	if strings.Contains(out.String()+errb.String(), meSecretToken) {
		t.Fatalf("the token leaked into output: %q", out.String()+errb.String())
	}
	return outcome, out.String(), errb.String()
}

// --- list walk branches ----------------------------------------------------

func TestRunPolicies_ListSuccessWalksAndProjects(t *testing.T) {
	tr := &cannedTransport{status: 200, body: policiesPageComplete}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, stderr := runPoliciesOver(t, seam, policiesConfig{id: "role_0123"})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success\nstderr: %s", outcome, stderr)
	}
	for _, want := range []string{
		"All PRs require two approvals (pol_1)",
		"Every PR needs two approvals",
		"Spending limit (pol_2)",
		"(whole-role — no domain)", // pol_2 has null domain_id
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
	if got := tr.lastPath; !strings.HasSuffix(got, "/roles/role_0123/policies") {
		t.Errorf("path = %q, want it to target /roles/role_0123/policies", got)
	}
}

func TestRunPolicies_EmptyIsCleanSuccess(t *testing.T) {
	tr := &cannedTransport{status: 200, body: policiesPageEmpty}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, stderr := runPoliciesOver(t, seam, policiesConfig{id: "role_0123"})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success", outcome)
	}
	if strings.TrimRight(stdout, "\n") != "No policies." {
		t.Errorf("a role with no policies should print exactly `No policies.`, got %q", stdout)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Errorf("an empty list is a clean success; stderr should be empty, got %q", stderr)
	}
}

func TestRunPolicies_WalksEveryPageToCompletion(t *testing.T) {
	tr := &seqMeTransport{steps: []seqMeResp{
		{status: 200, body: policiesPage("pol_1", "Page One", "c1")},
		{status: 200, body: policiesPage("pol_2", "Page Two", "c2")},
		{status: 200, body: policiesPage("pol_3", "Page Three", "")},
	}}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, _ := runPoliciesOver(t, seam, policiesConfig{id: "role_0123"})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success", outcome)
	}
	if tr.calls != 3 {
		t.Errorf("the walk should issue three page requests, got %d", tr.calls)
	}
	for _, want := range []string{"Page One", "Page Two", "Page Three"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("every page's policies should print, missing %q:\n%s", want, stdout)
		}
	}
}

// --- --query --------------------------------------------------------------

func TestRunPolicies_QuerySentWhenSetAndNonEmpty(t *testing.T) {
	tr := &cannedTransport{status: 200, body: policiesPageComplete}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	_, _, _ = runPoliciesOver(t, seam, policiesConfig{id: "role_0123", query: "approvals", querySet: true})
	if got := tr.lastQuery.Get("q"); got != "approvals" {
		t.Errorf("q = %q, want \"approvals\"", got)
	}
}

func TestRunPolicies_QueryEmptyOrUnsetSendsNoQ(t *testing.T) {
	// Omitted (querySet=false).
	tr1 := &cannedTransport{status: 200, body: policiesPageComplete}
	seam1 := &fakeMeSeam{ctx: validMeContext(), transport: tr1}
	_, _, _ = runPoliciesOver(t, seam1, policiesConfig{id: "role_0123"})
	if _, present := tr1.lastQuery["q"]; present {
		t.Errorf("an omitted --query must not send q, got %v", tr1.lastQuery)
	}

	// Present but empty (querySet=true, query="").
	tr2 := &cannedTransport{status: 200, body: policiesPageComplete}
	seam2 := &fakeMeSeam{ctx: validMeContext(), transport: tr2}
	_, _, _ = runPoliciesOver(t, seam2, policiesConfig{id: "role_0123", query: "", querySet: true})
	if _, present := tr2.lastQuery["q"]; present {
		t.Errorf("`--query \"\"` must behave as no filter (no q), got %v", tr2.lastQuery)
	}
}

// --- --first-page ----------------------------------------------------------

func TestRunPolicies_FirstPageStopsAndSignals(t *testing.T) {
	tr := &cannedTransport{status: 200, body: policiesPage("pol_1", "First Page Policy", "c1")}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, stderr := runPoliciesOver(t, seam, policiesConfig{id: "role_0123", firstPage: true})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success", outcome)
	}
	if !strings.Contains(stdout, "First Page Policy") {
		t.Errorf("the first page should print:\n%s", stdout)
	}
	if tr.calls != 1 {
		t.Errorf("--first-page must not walk, want 1 call, got %d", tr.calls)
	}
	if !strings.Contains(stderr, "more policies exist") {
		t.Errorf("stderr should note more policies exist:\n%s", stderr)
	}
}

func TestRunPolicies_PerPageSizesWalk(t *testing.T) {
	tr := &cannedTransport{status: 200, body: policiesPageComplete}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	_, _, _ = runPoliciesOver(t, seam, policiesConfig{id: "role_0123", perPage: 7, perPageSet: true})
	if got := tr.lastQuery.Get("per_page"); got != "7" {
		t.Errorf("per_page = %q, want \"7\" (WithPageSize passed through)", got)
	}
}

// --- mid-walk failure ------------------------------------------------------

func TestRunPolicies_MidWalkFailurePartialAndIncomplete(t *testing.T) {
	tr := &seqMeTransport{steps: []seqMeResp{
		{status: 200, body: policiesPage("pol_1", "Gathered Policy", "c1")},
		{status: 500, body: `{"detail":"boom"}`},
	}}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, stderr := runPoliciesOver(t, seam, policiesConfig{id: "role_0123"})
	if outcome == Success {
		t.Fatalf("a mid-walk failure must be non-zero, got Success")
	}
	if ExitCode(outcome) == 0 {
		t.Errorf("a mid-walk failure must exit non-zero, got exit 0 (outcome %v)", outcome)
	}
	if !strings.Contains(stdout, "Gathered Policy") {
		t.Errorf("the partial set gathered so far should print:\n%s", stdout)
	}
	if !strings.Contains(stderr, "incomplete") {
		t.Errorf("stderr should note the result is incomplete and name the cause:\n%s", stderr)
	}
}

// --- error classification (via the shared classifier) ----------------------

func TestRunPolicies_NoCredentialsIsUsageError(t *testing.T) {
	tr := &cannedTransport{status: 200, body: policiesPageComplete}
	seam := &fakeMeSeam{ctx: noTokenContext(), transport: tr}

	outcome, stdout, stderr := runPoliciesOver(t, seam, policiesConfig{id: "role_0123"})
	if outcome != UsageError || ExitCode(outcome) != 2 {
		t.Fatalf("outcome=%v exit=%d, want UsageError/2", outcome, ExitCode(outcome))
	}
	if !strings.Contains(strings.ToLower(stderr), "not authenticated") {
		t.Errorf("stderr should report not authenticated:\n%s", stderr)
	}
	if strings.TrimSpace(stdout) != "" {
		t.Errorf("no policy data should be printed on a credential failure, got:\n%s", stdout)
	}
}

func TestRunPolicies_TransportErrorIsNetworkUnavailable(t *testing.T) {
	tr := &cannedTransport{netErr: errors.New("dial tcp: connection refused")}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, _, stderr := runPoliciesOver(t, seam, policiesConfig{id: "role_0123"})
	if outcome != NetworkUnavailable || ExitCode(outcome) != 6 {
		t.Fatalf("outcome=%v exit=%d, want NetworkUnavailable/6\nstderr: %s", outcome, ExitCode(outcome), stderr)
	}
}

func TestRunPolicies_Non2xxClassifies(t *testing.T) {
	cases := []struct {
		status int
		want   Outcome
		code   int
	}{
		{403, PermissionError, 4},
		{429, RateLimited, 5},
		{500, APIError, 3},
	}
	for _, c := range cases {
		tr := &cannedTransport{status: c.status, body: `{"detail":"x"}`}
		seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}
		outcome, _, stderr := runPoliciesOver(t, seam, policiesConfig{id: "role_0123"})
		if outcome != c.want || ExitCode(outcome) != c.code {
			t.Errorf("status %d: outcome=%v exit=%d, want %v/%d\nstderr: %s", c.status, outcome, ExitCode(outcome), c.want, c.code, stderr)
		}
	}
}

// --- resolve-before-call: a bad --output costs no request ------------------

func TestRunPolicies_BadOutputIsUsageErrorNoRequest(t *testing.T) {
	tr := &cannedTransport{status: 200, body: policiesPageComplete}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, _, stderr := runPoliciesOver(t, seam, policiesConfig{id: "role_0123", outputFlag: "xml"})
	if outcome != UsageError || ExitCode(outcome) != 2 {
		t.Fatalf("outcome=%v exit=%d, want UsageError/2\nstderr: %s", outcome, ExitCode(outcome), stderr)
	}
	if tr.calls != 0 {
		t.Errorf("a bad --output must be rejected before any request, got %d calls", tr.calls)
	}
}

// --- structured output emits the raw payload -------------------------------

func TestRunPolicies_StructuredJSONEmitsRawPayload(t *testing.T) {
	tr := &cannedTransport{status: 200, body: policiesPageComplete}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, _ := runPoliciesOver(t, seam, policiesConfig{id: "role_0123", outputFlag: "json"})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success", outcome)
	}
	for _, want := range []string{`"data"`, "pol_1", `"role_id"`, `"created_at"`} {
		if !strings.Contains(stdout, want) {
			t.Errorf("structured json should carry the raw payload, missing %q:\n%s", want, stdout)
		}
	}
	// Structured output must not carry the human projection's block labels.
	if strings.Contains(stdout, "(whole-role — no domain)") {
		t.Errorf("structured json must not render the human projection:\n%s", stdout)
	}
}

// --- command-level wiring --------------------------------------------------

// TestPoliciesCommand_QueryFlagSendsQ pins the Changed()-gating end to end: a
// real `policies <id> --query approvals` invocation sends q=approvals.
func TestPoliciesCommand_QueryFlagSendsQ(t *testing.T) {
	tr := &cannedTransport{status: 200, body: policiesPageComplete}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	root := NewRootCommand()
	MustRegister(root, newPoliciesCommand(seam))
	var out, errb bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&errb)
	outcome, _ := Run(root, []string{"policies", "role_0123", "--query", "approvals"})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success\nstderr: %s", outcome, errb.String())
	}
	if got := tr.lastQuery.Get("q"); got != "approvals" {
		t.Errorf("q = %q, want \"approvals\"", got)
	}
}

// TestPoliciesCommand_RequiresExactlyOneArg pins ExactArgs(1): zero args is a
// usage error and sends no request.
func TestPoliciesCommand_RequiresExactlyOneArg(t *testing.T) {
	tr := &cannedTransport{status: 200, body: policiesPageComplete}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	root := NewRootCommand()
	MustRegister(root, newPoliciesCommand(seam))
	root.SetOut(&bytes.Buffer{})
	root.SetErr(&bytes.Buffer{})
	outcome, _ := Run(root, []string{"policies"})
	if outcome != UsageError {
		t.Errorf("zero args should be a UsageError, got %v", outcome)
	}
	if tr.calls != 0 {
		t.Errorf("a wrong arg count must send no request, got %d calls", tr.calls)
	}
}

// --- single policy read (T004) ---------------------------------------------

func TestRunPolicy_SingleReadPrintsTitleAndFullBody(t *testing.T) {
	tr := &cannedTransport{status: 200, body: policyDocumentBody}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, stderr := runPolicyOver(t, seam, policyConfig{}, "pol_0123")
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success\nstderr: %s", outcome, stderr)
	}
	for _, want := range []string{
		"All PRs require two approvals (pol_0123)",
		"<p>Every PR needs <strong>two</strong> approvals.</p>", // full body verbatim
		"role_0123", "dom_1",
		"2024-01-02T03:04:05Z", "2024-05-06T07:08:09Z",
	} {
		if !strings.Contains(stdout, want) {
			t.Errorf("the single policy should show %q:\n%s", want, stdout)
		}
	}
	if tr.calls != 1 {
		t.Errorf("a single read is one call, got %d", tr.calls)
	}
	if got := tr.lastPath; !strings.HasSuffix(got, "/policies/pol_0123") {
		t.Errorf("path = %q, want it to target /policies/pol_0123", got)
	}
}

func TestRunPolicy_UnknownIdSurfacesAPIStatus(t *testing.T) {
	tr := &cannedTransport{status: 404, body: `{"detail":"Policy not found"}`}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, stderr := runPolicyOver(t, seam, policyConfig{}, "pol_ffff")
	if outcome != APIError || ExitCode(outcome) != 3 {
		t.Fatalf("an unknown id should surface APIError/3, got %v/%d\nstderr: %s", outcome, ExitCode(outcome), stderr)
	}
	if !strings.Contains(stderr, "404") {
		t.Errorf("stderr should name the HTTP status (404):\n%s", stderr)
	}
	if strings.TrimSpace(stdout) != "" {
		t.Errorf("nothing should be printed to stdout on a not-found, got:\n%s", stdout)
	}
}

func TestRunPolicy_StructuredJSONEmitsRawPayload(t *testing.T) {
	tr := &cannedTransport{status: 200, body: policyDocumentBody}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, _ := runPolicyOver(t, seam, policyConfig{outputFlag: "json"}, "pol_0123")
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success", outcome)
	}
	for _, want := range []string{`"data"`, "pol_0123", `"role_id"`, `"updated_at"`} {
		if !strings.Contains(stdout, want) {
			t.Errorf("structured json should carry the raw single-policy payload, missing %q:\n%s", want, stdout)
		}
	}
}

// TestPolicyCommand_ListFlagRejectedNoRequest pins the structural list-only guard
// (ADR-1): a list-only flag on `policy` is a cobra unknown-flag UsageError before
// any request — the transport tripwire confirms nothing is sent.
func TestPolicyCommand_ListFlagRejectedNoRequest(t *testing.T) {
	for _, flag := range []string{"--query", "--first-page", "--per-page"} {
		tr := &cannedTransport{status: 200, body: policyDocumentBody}
		seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

		root := NewRootCommand()
		MustRegister(root, newPolicyCommand(seam))
		var out, errb bytes.Buffer
		root.SetOut(&out)
		root.SetErr(&errb)
		// --query/--per-page take a value; --first-page is a bool. Pass a value for the
		// value-taking flags so cobra's failure is unknown-flag, not missing-value.
		args := []string{"policy", "pol_0123", flag}
		if flag != "--first-page" {
			args = append(args, "x")
		}
		outcome, _ := Run(root, args)
		if outcome != UsageError {
			t.Errorf("%s on `policy` should be a UsageError, got %v", flag, outcome)
		}
		if tr.calls != 0 {
			t.Errorf("%s on `policy` must send no request, got %d calls", flag, tr.calls)
		}
	}
}

// TestPolicyCommand_RequiresExactlyOneArg pins ExactArgs(1) on the single read.
func TestPolicyCommand_RequiresExactlyOneArg(t *testing.T) {
	tr := &cannedTransport{status: 200, body: policyDocumentBody}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	root := NewRootCommand()
	MustRegister(root, newPolicyCommand(seam))
	root.SetOut(&bytes.Buffer{})
	root.SetErr(&bytes.Buffer{})
	outcome, _ := Run(root, []string{"policy"})
	if outcome != UsageError {
		t.Errorf("zero args should be a UsageError, got %v", outcome)
	}
	if tr.calls != 0 {
		t.Errorf("a wrong arg count must send no request, got %d calls", tr.calls)
	}
}
