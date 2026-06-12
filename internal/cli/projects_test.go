package cli

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
)

// --- canned GET /roles/{id}/projects bodies --------------------------------
//
// They carry the Project shape (status/description/role_id/tags + the detail
// fields) in the API's snake_case names, one individual-initiative project (null
// role_id) to exercise the no-role marker, and the secret token nowhere.

const projectsPageComplete = `{"data":[
  {"id":"proj_1","status":"current","description":"Ship onboarding","role_id":"role_0123","tags":["q3"],"has_sub_projects":true,"has_actions":false},
  {"id":"proj_2","status":"scheduled","description":"Audit billing","role_id":null,"tags":[],"has_sub_projects":false,"has_actions":true}
],"meta":{"pagination":{"per_page":100,"has_next_page":false,"next_cursor":""}}}`

const projectsPageEmpty = `{"data":[],"meta":{"pagination":{"per_page":100,"has_next_page":false,"next_cursor":""}}}`

// projectDocumentBody is a representative GET /projects/{id} body: the
// single-object {data: Project} envelope carrying the full detail (status,
// description, owning role, parent, timestamps, link, note).
const projectDocumentBody = `{"data":{"id":"proj_0123","status":"current","description":"Ship the new onboarding flow","role_id":"role_0123","parent_project_id":"proj_root","tags":["q3"],"has_sub_projects":true,"has_actions":true,"created_at":"2024-01-02T03:04:05Z","updated_at":"2024-05-06T07:08:09Z","link":"https://example.com/p/123","note":"Blocked on design review"}}`

// runProjectOver drives the pure runProjectGet over a fake seam, returning the
// outcome and captured stdout/stderr, and failing if the token leaks.
func runProjectOver(t *testing.T, seam projectsSeam, cfg projectConfig, id string) (Outcome, string, string) {
	t.Helper()
	var out, errb bytes.Buffer
	cfg.seam = seam
	cfg.reqCtx = context.Background()
	cfg.stdout = &out
	cfg.stderr = &errb
	outcome, _ := runProjectGet(cfg, id)
	if strings.Contains(out.String()+errb.String(), meSecretToken) {
		t.Fatalf("the token leaked into output: %q", out.String()+errb.String())
	}
	return outcome, out.String(), errb.String()
}

// projectsPage builds a one-project page; a non-empty nextCursor marks more pages.
func projectsPage(id, description, nextCursor string) string {
	hasNext := "false"
	if nextCursor != "" {
		hasNext = "true"
	}
	return `{"data":[{"id":"` + id + `","status":"current","description":"` + description + `","role_id":"role_0123","tags":[],"has_sub_projects":false,"has_actions":false}],` +
		`"meta":{"pagination":{"per_page":1,"has_next_page":` + hasNext + `,"next_cursor":"` + nextCursor + `"}}}`
}

// runProjectsOver drives the pure runProjectsList over a fake seam, returning the
// outcome and captured stdout/stderr, and failing if the token leaks.
func runProjectsOver(t *testing.T, seam projectsSeam, cfg projectsConfig) (Outcome, string, string) {
	t.Helper()
	var out, errb bytes.Buffer
	cfg.seam = seam
	cfg.reqCtx = context.Background()
	cfg.stdout = &out
	cfg.stderr = &errb
	outcome, _ := runProjectsList(cfg)
	if strings.Contains(out.String()+errb.String(), meSecretToken) {
		t.Fatalf("the token leaked into output: %q", out.String()+errb.String())
	}
	return outcome, out.String(), errb.String()
}

// --- list walk branches ----------------------------------------------------

func TestRunProjects_ListSuccessWalksAndProjects(t *testing.T) {
	tr := &cannedTransport{status: 200, body: projectsPageComplete}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, stderr := runProjectsOver(t, seam, projectsConfig{id: "role_0123"})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success\nstderr: %s", outcome, stderr)
	}
	for _, want := range []string{
		"proj_1  [current]  Ship onboarding",
		"proj_2  [scheduled]  Audit billing",
		"role: —", // proj_2 has null role_id → the individual-initiative marker
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
	if got := tr.lastPath; !strings.HasSuffix(got, "/roles/role_0123/projects") {
		t.Errorf("path = %q, want it to target /roles/role_0123/projects", got)
	}
}

func TestRunProjects_EmptyIsCleanSuccess(t *testing.T) {
	tr := &cannedTransport{status: 200, body: projectsPageEmpty}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, stderr := runProjectsOver(t, seam, projectsConfig{id: "role_0123"})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success", outcome)
	}
	if strings.TrimRight(stdout, "\n") != "no projects" {
		t.Errorf("a role with no projects should print exactly `no projects`, got %q", stdout)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Errorf("an empty list is a clean success; stderr should be empty, got %q", stderr)
	}
}

func TestRunProjects_WalksEveryPageToCompletion(t *testing.T) {
	tr := &seqMeTransport{steps: []seqMeResp{
		{status: 200, body: projectsPage("proj_1", "Page One", "c1")},
		{status: 200, body: projectsPage("proj_2", "Page Two", "c2")},
		{status: 200, body: projectsPage("proj_3", "Page Three", "")},
	}}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, _ := runProjectsOver(t, seam, projectsConfig{id: "role_0123"})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success", outcome)
	}
	if tr.calls != 3 {
		t.Errorf("the walk should issue three page requests, got %d", tr.calls)
	}
	for _, want := range []string{"Page One", "Page Two", "Page Three"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("every page's projects should print, missing %q:\n%s", want, stdout)
		}
	}
}

// --- --query / --status / --tag --------------------------------------------

func TestRunProjects_QuerySentWhenSetAndNonEmpty(t *testing.T) {
	tr := &cannedTransport{status: 200, body: projectsPageComplete}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	_, _, _ = runProjectsOver(t, seam, projectsConfig{id: "role_0123", query: "onboarding", querySet: true})
	if got := tr.lastQuery.Get("q"); got != "onboarding" {
		t.Errorf("q = %q, want \"onboarding\"", got)
	}
}

func TestRunProjects_StatusSentWhenSupplied(t *testing.T) {
	tr := &cannedTransport{status: 200, body: projectsPageComplete}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	_, _, _ = runProjectsOver(t, seam, projectsConfig{id: "role_0123", status: "current"})
	if got := tr.lastQuery.Get("status"); got != "current" {
		t.Errorf("status = %q, want \"current\"", got)
	}
}

func TestRunProjects_TagSentWhenSetAndNonEmpty(t *testing.T) {
	tr := &cannedTransport{status: 200, body: projectsPageComplete}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	_, _, _ = runProjectsOver(t, seam, projectsConfig{id: "role_0123", tag: "growth", tagSet: true})
	if got := tr.lastQuery.Get("tag"); got != "growth" {
		t.Errorf("tag = %q, want \"growth\"", got)
	}
}

func TestRunProjects_ThreeFiltersCombine(t *testing.T) {
	tr := &cannedTransport{status: 200, body: projectsPageComplete}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	_, _, _ = runProjectsOver(t, seam, projectsConfig{
		id: "role_0123", query: "x", querySet: true, status: "current", tag: "t", tagSet: true,
	})
	if got := tr.lastQuery.Get("q"); got != "x" {
		t.Errorf("q = %q, want \"x\"", got)
	}
	if got := tr.lastQuery.Get("status"); got != "current" {
		t.Errorf("status = %q, want \"current\"", got)
	}
	if got := tr.lastQuery.Get("tag"); got != "t" {
		t.Errorf("tag = %q, want \"t\"", got)
	}
}

func TestRunProjects_OmittedOrEmptyFiltersSendNothing(t *testing.T) {
	// Omitted (querySet=false, tagSet=false, status="").
	tr1 := &cannedTransport{status: 200, body: projectsPageComplete}
	seam1 := &fakeMeSeam{ctx: validMeContext(), transport: tr1}
	_, _, _ = runProjectsOver(t, seam1, projectsConfig{id: "role_0123"})
	for _, param := range []string{"q", "status", "tag"} {
		if _, present := tr1.lastQuery[param]; present {
			t.Errorf("an omitted filter must not send %q, got %v", param, tr1.lastQuery)
		}
	}

	// Present but empty (querySet/tagSet=true with empty values).
	tr2 := &cannedTransport{status: 200, body: projectsPageComplete}
	seam2 := &fakeMeSeam{ctx: validMeContext(), transport: tr2}
	_, _, _ = runProjectsOver(t, seam2, projectsConfig{id: "role_0123", query: "", querySet: true, tag: "", tagSet: true})
	for _, param := range []string{"q", "tag"} {
		if _, present := tr2.lastQuery[param]; present {
			t.Errorf("an empty filter must behave as no filter (no %q), got %v", param, tr2.lastQuery)
		}
	}
}

// --- --status validation (the one closed-enum input) -----------------------

func TestRunProjects_UnsupportedStatusIsUsageErrorNoRequest(t *testing.T) {
	tr := &cannedTransport{status: 200, body: projectsPageComplete}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, _, stderr := runProjectsOver(t, seam, projectsConfig{id: "role_0123", status: "active"})
	if outcome != UsageError || ExitCode(outcome) != 2 {
		t.Fatalf("outcome=%v exit=%d, want UsageError/2", outcome, ExitCode(outcome))
	}
	if !strings.Contains(stderr, "active") {
		t.Errorf("stderr should name the unsupported value:\n%s", stderr)
	}
	if !strings.Contains(stderr, "current") {
		t.Errorf("stderr should list the supported set:\n%s", stderr)
	}
	if tr.calls != 0 {
		t.Errorf("an unsupported --status must be rejected before any request, got %d calls", tr.calls)
	}
}

// --- --first-page ----------------------------------------------------------

func TestRunProjects_FirstPageStopsAndSignals(t *testing.T) {
	tr := &cannedTransport{status: 200, body: projectsPage("proj_1", "First Page Project", "c1")}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, stderr := runProjectsOver(t, seam, projectsConfig{id: "role_0123", firstPage: true})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success", outcome)
	}
	if !strings.Contains(stdout, "First Page Project") {
		t.Errorf("the first page should print:\n%s", stdout)
	}
	if tr.calls != 1 {
		t.Errorf("--first-page must not walk, want 1 call, got %d", tr.calls)
	}
	if !strings.Contains(stderr, "more projects exist") {
		t.Errorf("stderr should note more projects exist:\n%s", stderr)
	}
}

func TestRunProjects_PerPageSizesWalk(t *testing.T) {
	tr := &cannedTransport{status: 200, body: projectsPageComplete}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	_, _, _ = runProjectsOver(t, seam, projectsConfig{id: "role_0123", perPage: 7, perPageSet: true})
	if got := tr.lastQuery.Get("per_page"); got != "7" {
		t.Errorf("per_page = %q, want \"7\" (WithPageSize passed through)", got)
	}
}

// --- mid-walk failure ------------------------------------------------------

func TestRunProjects_MidWalkFailurePartialAndIncomplete(t *testing.T) {
	tr := &seqMeTransport{steps: []seqMeResp{
		{status: 200, body: projectsPage("proj_1", "Gathered Project", "c1")},
		{status: 500, body: `{"detail":"boom"}`},
	}}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, stderr := runProjectsOver(t, seam, projectsConfig{id: "role_0123"})
	if outcome == Success {
		t.Fatalf("a mid-walk failure must be non-zero, got Success")
	}
	if ExitCode(outcome) == 0 {
		t.Errorf("a mid-walk failure must exit non-zero, got exit 0 (outcome %v)", outcome)
	}
	if !strings.Contains(stdout, "Gathered Project") {
		t.Errorf("the partial set gathered so far should print:\n%s", stdout)
	}
	if !strings.Contains(stderr, "incomplete") {
		t.Errorf("stderr should note the result is incomplete and name the cause:\n%s", stderr)
	}
}

// --- error classification (via the shared classifier) ----------------------

func TestRunProjects_NoCredentialsIsUsageError(t *testing.T) {
	tr := &cannedTransport{status: 200, body: projectsPageComplete}
	seam := &fakeMeSeam{ctx: noTokenContext(), transport: tr}

	outcome, stdout, stderr := runProjectsOver(t, seam, projectsConfig{id: "role_0123"})
	if outcome != UsageError || ExitCode(outcome) != 2 {
		t.Fatalf("outcome=%v exit=%d, want UsageError/2", outcome, ExitCode(outcome))
	}
	if !strings.Contains(strings.ToLower(stderr), "not authenticated") {
		t.Errorf("stderr should report not authenticated:\n%s", stderr)
	}
	if strings.TrimSpace(stdout) != "" {
		t.Errorf("no project data should be printed on a credential failure, got:\n%s", stdout)
	}
}

func TestRunProjects_TransportErrorIsNetworkUnavailable(t *testing.T) {
	tr := &cannedTransport{netErr: errors.New("dial tcp: connection refused")}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, _, stderr := runProjectsOver(t, seam, projectsConfig{id: "role_0123"})
	if outcome != NetworkUnavailable || ExitCode(outcome) != 6 {
		t.Fatalf("outcome=%v exit=%d, want NetworkUnavailable/6\nstderr: %s", outcome, ExitCode(outcome), stderr)
	}
}

func TestRunProjects_Non2xxClassifies(t *testing.T) {
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
		outcome, _, stderr := runProjectsOver(t, seam, projectsConfig{id: "role_0123"})
		if outcome != c.want || ExitCode(outcome) != c.code {
			t.Errorf("status %d: outcome=%v exit=%d, want %v/%d\nstderr: %s", c.status, outcome, ExitCode(outcome), c.want, c.code, stderr)
		}
	}
}

// --- resolve-before-call: a bad --output costs no request ------------------

func TestRunProjects_BadOutputIsUsageErrorNoRequest(t *testing.T) {
	tr := &cannedTransport{status: 200, body: projectsPageComplete}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, _, stderr := runProjectsOver(t, seam, projectsConfig{id: "role_0123", outputFlag: "xml", outputPresent: true})
	if outcome != UsageError || ExitCode(outcome) != 2 {
		t.Fatalf("outcome=%v exit=%d, want UsageError/2\nstderr: %s", outcome, ExitCode(outcome), stderr)
	}
	if tr.calls != 0 {
		t.Errorf("a bad --output must be rejected before any request, got %d calls", tr.calls)
	}
}

// --- structured output emits the aggregated raw document --------------------

func TestRunProjects_StructuredJSONEmitsAggregatedRawPayload(t *testing.T) {
	tr := &cannedTransport{status: 200, body: projectsPageComplete}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, _ := runProjectsOver(t, seam, projectsConfig{id: "role_0123", outputFlag: "json", outputPresent: true})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success", outcome)
	}
	for _, want := range []string{`"data"`, "proj_1", `"role_id"`, `"has_sub_projects"`} {
		if !strings.Contains(stdout, want) {
			t.Errorf("structured json should carry the raw payload, missing %q:\n%s", want, stdout)
		}
	}
	// Structured output must not carry the human projection's block labels nor the
	// per-page meta envelope.
	if strings.Contains(stdout, "sub-projects:") {
		t.Errorf("structured json must not render the human projection:\n%s", stdout)
	}
	if strings.Contains(stdout, `"pagination"`) {
		t.Errorf("the aggregated document must drop the per-page meta envelope:\n%s", stdout)
	}
}

// --- command-level wiring --------------------------------------------------

// TestProjectsCommand_FiltersSendParams pins the Changed()-gating end to end: a
// real `projects <id> --query x --status current --tag t` invocation sends all three.
func TestProjectsCommand_FiltersSendParams(t *testing.T) {
	tr := &cannedTransport{status: 200, body: projectsPageComplete}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	root := NewRootCommand()
	MustRegister(root, newProjectsCommand(seam))
	var out, errb bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&errb)
	outcome, _ := Run(root, []string{"projects", "role_0123", "--query", "x", "--status", "current", "--tag", "t"})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success\nstderr: %s", outcome, errb.String())
	}
	if got := tr.lastQuery.Get("q"); got != "x" {
		t.Errorf("q = %q, want \"x\"", got)
	}
	if got := tr.lastQuery.Get("status"); got != "current" {
		t.Errorf("status = %q, want \"current\"", got)
	}
	if got := tr.lastQuery.Get("tag"); got != "t" {
		t.Errorf("tag = %q, want \"t\"", got)
	}
}

// TestProjectsCommand_UnsupportedStatusNoRequest pins fail-fast --status validation
// at the command level: a real invocation with a bad status sends no request.
func TestProjectsCommand_UnsupportedStatusNoRequest(t *testing.T) {
	tr := &cannedTransport{status: 200, body: projectsPageComplete}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	root := NewRootCommand()
	MustRegister(root, newProjectsCommand(seam))
	root.SetOut(&bytes.Buffer{})
	root.SetErr(&bytes.Buffer{})
	outcome, _ := Run(root, []string{"projects", "role_0123", "--status", "active"})
	if outcome != UsageError {
		t.Errorf("an unsupported --status should be a UsageError, got %v", outcome)
	}
	if tr.calls != 0 {
		t.Errorf("an unsupported --status must send no request, got %d calls", tr.calls)
	}
}

// TestProjectsCommand_RequiresExactlyOneArg pins ExactArgs(1): zero args is a usage
// error and sends no request.
func TestProjectsCommand_RequiresExactlyOneArg(t *testing.T) {
	tr := &cannedTransport{status: 200, body: projectsPageComplete}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	root := NewRootCommand()
	MustRegister(root, newProjectsCommand(seam))
	root.SetOut(&bytes.Buffer{})
	root.SetErr(&bytes.Buffer{})
	outcome, _ := Run(root, []string{"projects"})
	if outcome != UsageError {
		t.Errorf("zero args should be a UsageError, got %v", outcome)
	}
	if tr.calls != 0 {
		t.Errorf("a wrong arg count must send no request, got %d calls", tr.calls)
	}
}

// --- single project read (T003) --------------------------------------------

func TestRunProject_SingleReadPrintsDetail(t *testing.T) {
	tr := &cannedTransport{status: 200, body: projectDocumentBody}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, stderr := runProjectOver(t, seam, projectConfig{}, "proj_0123")
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success\nstderr: %s", outcome, stderr)
	}
	for _, want := range []string{
		"proj_0123  [current]",
		"Ship the new onboarding flow", // description
		"role_0123",                    // owning role
		"proj_root",                    // parent
		"Blocked on design review",     // note, verbatim
		"2024-01-02T03:04:05Z",         // created
	} {
		if !strings.Contains(stdout, want) {
			t.Errorf("the single project should show %q:\n%s", want, stdout)
		}
	}
	if tr.calls != 1 {
		t.Errorf("a single read is one call, got %d", tr.calls)
	}
	if got := tr.lastPath; !strings.HasSuffix(got, "/projects/proj_0123") {
		t.Errorf("path = %q, want it to target /projects/proj_0123", got)
	}
}

func TestRunProject_UnknownIdSurfacesAPIStatus(t *testing.T) {
	tr := &cannedTransport{status: 404, body: `{"detail":"Project not found"}`}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, stderr := runProjectOver(t, seam, projectConfig{}, "proj_ffff")
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

func TestRunProject_StructuredJSONEmitsRawPayload(t *testing.T) {
	tr := &cannedTransport{status: 200, body: projectDocumentBody}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, _ := runProjectOver(t, seam, projectConfig{outputFlag: "json", outputPresent: true}, "proj_0123")
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success", outcome)
	}
	for _, want := range []string{`"data"`, "proj_0123", `"role_id"`, `"updated_at"`} {
		if !strings.Contains(stdout, want) {
			t.Errorf("structured json should carry the raw single-project payload, missing %q:\n%s", want, stdout)
		}
	}
	// The single read emits the raw {data: Project} body, not the human projection.
	if strings.Contains(stdout, "Sub-projects:") {
		t.Errorf("structured json must not render the human projection:\n%s", stdout)
	}
}

// TestProjectCommand_ListFlagRejectedNoRequest pins the structural list-only guard
// (ADR-1): a list-only flag on `project` is a cobra unknown-flag UsageError before
// any request — the transport tripwire confirms nothing is sent.
func TestProjectCommand_ListFlagRejectedNoRequest(t *testing.T) {
	for _, flag := range []string{"--query", "--status", "--tag", "--first-page", "--per-page"} {
		tr := &cannedTransport{status: 200, body: projectDocumentBody}
		seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

		root := NewRootCommand()
		MustRegister(root, newProjectCommand(seam))
		var out, errb bytes.Buffer
		root.SetOut(&out)
		root.SetErr(&errb)
		// The value-taking flags need a value so cobra's failure is unknown-flag, not
		// missing-value; --first-page is a bool.
		args := []string{"project", "proj_0123", flag}
		if flag != "--first-page" {
			args = append(args, "x")
		}
		outcome, _ := Run(root, args)
		if outcome != UsageError {
			t.Errorf("%s on `project` should be a UsageError, got %v", flag, outcome)
		}
		if tr.calls != 0 {
			t.Errorf("%s on `project` must send no request, got %d calls", flag, tr.calls)
		}
	}
}

// TestProjectCommand_RequiresExactlyOneArg pins ExactArgs(1) on the single read.
func TestProjectCommand_RequiresExactlyOneArg(t *testing.T) {
	tr := &cannedTransport{status: 200, body: projectDocumentBody}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	root := NewRootCommand()
	MustRegister(root, newProjectCommand(seam))
	root.SetOut(&bytes.Buffer{})
	root.SetErr(&bytes.Buffer{})
	outcome, _ := Run(root, []string{"project"})
	if outcome != UsageError {
		t.Errorf("zero args should be a UsageError, got %v", outcome)
	}
	if tr.calls != 0 {
		t.Errorf("a wrong arg count must send no request, got %d calls", tr.calls)
	}
}
