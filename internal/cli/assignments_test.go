package cli

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
)

// --- canned GET /actors/{id}/assignments bodies ----------------------------
//
// They carry the actor-end Assignment shape (id/actor_id/role_id/focus/elected_until
// + the embedded role{id,type,name,purpose,parent_role_id} from the default
// include=role) in the API's snake_case names, two filled roles to exercise the row
// projection, and the secret token nowhere.

const assignmentsPageComplete = `{"data":[
  {"id":"asgn_1","actor_id":"per_0123","role_id":"role_a","focus":"Keep the lights on","elected_until":"2026-12-31","role":{"id":"role_a","type":"role","name":"Marketing Lead","purpose":"A market that knows us","parent_role_id":"role_parent"}},
  {"id":"asgn_2","actor_id":"per_0123","role_id":"role_b","focus":"","elected_until":"","role":{"id":"role_b","type":"circle","name":"General Company Circle","purpose":"","parent_role_id":""}}
],"meta":{"pagination":{"per_page":100,"has_next_page":false,"next_cursor":""}}}`

const assignmentsPageEmpty = `{"data":[],"meta":{"pagination":{"per_page":100,"has_next_page":false,"next_cursor":""}}}`

// assignmentsPage builds a one-assignment page; a non-empty nextCursor marks more
// pages. The actor id rides actor_id; the filled role's name distinguishes pages.
func assignmentsPage(actorID, roleID, roleName, nextCursor string) string {
	hasNext := "false"
	if nextCursor != "" {
		hasNext = "true"
	}
	return `{"data":[{"id":"asgn_x","actor_id":"` + actorID + `","role_id":"` + roleID + `","focus":"","elected_until":"",` +
		`"role":{"id":"` + roleID + `","type":"role","name":"` + roleName + `","purpose":"","parent_role_id":""}}],` +
		`"meta":{"pagination":{"per_page":1,"has_next_page":` + hasNext + `,"next_cursor":"` + nextCursor + `"}}}`
}

// runAssignmentsOver drives the pure runAssignmentsList over a fake seam, returning
// the outcome and captured stdout/stderr, and failing if the token leaks.
func runAssignmentsOver(t *testing.T, seam assignmentsSeam, cfg assignmentsConfig) (Outcome, string, string) {
	t.Helper()
	var out, errb bytes.Buffer
	cfg.seam = seam
	cfg.reqCtx = context.Background()
	cfg.stdout = &out
	cfg.stderr = &errb
	outcome, _ := runAssignmentsList(cfg)
	if strings.Contains(out.String()+errb.String(), meSecretToken) {
		t.Fatalf("the token leaked into output: %q", out.String()+errb.String())
	}
	return outcome, out.String(), errb.String()
}

// --- list walk branches ----------------------------------------------------

func TestRunAssignments_ListSuccessWalksAndProjects(t *testing.T) {
	tr := &cannedTransport{status: 200, body: assignmentsPageComplete}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, stderr := runAssignmentsOver(t, seam, assignmentsConfig{id: "per_0123"})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success\nstderr: %s", outcome, stderr)
	}
	for _, want := range []string{
		"role_a",         // the row leads with the filled role id
		"Marketing Lead", // the filled role name from the default include
		"role_b",
		"General Company Circle",
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
	if got := tr.lastPath; !strings.HasSuffix(got, "/actors/per_0123/assignments") {
		t.Errorf("path = %q, want it to target /actors/per_0123/assignments", got)
	}
}

// An agent actor is read from the same endpoint as a person — the id is passed
// through unchanged (the person/agent scenario).
func TestRunAssignments_AgentReadFromSameEndpoint(t *testing.T) {
	tr := &cannedTransport{status: 200, body: assignmentsPage("agt_0456", "role_a", "Marketing Lead", "")}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, _ := runAssignmentsOver(t, seam, assignmentsConfig{id: "agt_0456"})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success", outcome)
	}
	if got := tr.lastPath; !strings.HasSuffix(got, "/actors/agt_0456/assignments") {
		t.Errorf("an agent id should read /actors/agt_0456/assignments, got %q", got)
	}
	if !strings.Contains(stdout, "Marketing Lead") {
		t.Errorf("the agent's assignment should print:\n%s", stdout)
	}
}

// The filled role's name + id come from the endpoint's default include=role — the
// command declares no --include flag and sends no filter param (plan ADR-3).
func TestRunAssignments_RoleNameShownWithoutIncludeFlag(t *testing.T) {
	tr := &cannedTransport{status: 200, body: assignmentsPageComplete}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	_, stdout, _ := runAssignmentsOver(t, seam, assignmentsConfig{id: "per_0123"})
	if !strings.Contains(stdout, "Marketing Lead") || !strings.Contains(stdout, "role_a") {
		t.Errorf("the filled role's name and id should be shown from the default include:\n%s", stdout)
	}
	// No filter/include query param is sent — the request is the bare default.
	for _, param := range []string{"include", "kind", "q", "role_id"} {
		if _, present := tr.lastQuery[param]; present {
			t.Errorf("the command sends no %q param (no filters / no --include), got %v", param, tr.lastQuery)
		}
	}
}

// An assignment's focus and election expiry are projected under the default human
// format; an absent one shows the explicit-absence markers.
func TestRunAssignments_FocusAndElectionProjected(t *testing.T) {
	tr := &cannedTransport{status: 200, body: assignmentsPageComplete}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	_, stdout, _ := runAssignmentsOver(t, seam, assignmentsConfig{id: "per_0123"})
	for _, want := range []string{"Keep the lights on", "2026-12-31"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("focus/election must be projected, missing %q:\n%s", want, stdout)
		}
	}
	// The focus-less / non-elected / top-level second row shows the absence markers.
	for _, want := range []string{"(none)", "(not an elected seat)", "(top-level)"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("an absent focus/election/parent must show explicit-absence markers, missing %q:\n%s", want, stdout)
		}
	}
}

func TestRunAssignments_EmptyIsCleanSuccess(t *testing.T) {
	tr := &cannedTransport{status: 200, body: assignmentsPageEmpty}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, stderr := runAssignmentsOver(t, seam, assignmentsConfig{id: "per_0123"})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success", outcome)
	}
	if strings.TrimRight(stdout, "\n") != "no assignments" {
		t.Errorf("an actor with no assignments should print exactly `no assignments`, got %q", stdout)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Errorf("an empty list is a clean success; stderr should be empty, got %q", stderr)
	}
}

func TestRunAssignments_WalksEveryPageToCompletion(t *testing.T) {
	tr := &seqMeTransport{steps: []seqMeResp{
		{status: 200, body: assignmentsPage("per_0123", "role_1", "Role One", "c1")},
		{status: 200, body: assignmentsPage("per_0123", "role_2", "Role Two", "c2")},
		{status: 200, body: assignmentsPage("per_0123", "role_3", "Role Three", "")},
	}}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, _ := runAssignmentsOver(t, seam, assignmentsConfig{id: "per_0123"})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success", outcome)
	}
	if tr.calls != 3 {
		t.Errorf("the walk should issue three page requests, got %d", tr.calls)
	}
	for _, want := range []string{"Role One", "Role Two", "Role Three"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("every page's assignments should print, missing %q:\n%s", want, stdout)
		}
	}
}

// --- --first-page ----------------------------------------------------------

func TestRunAssignments_FirstPageStopsAndSignals(t *testing.T) {
	tr := &cannedTransport{status: 200, body: assignmentsPage("per_0123", "role_1", "First Page Role", "c1")}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, stderr := runAssignmentsOver(t, seam, assignmentsConfig{id: "per_0123", firstPage: true})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success", outcome)
	}
	if !strings.Contains(stdout, "First Page Role") {
		t.Errorf("the first page should print:\n%s", stdout)
	}
	if tr.calls != 1 {
		t.Errorf("--first-page must not walk, want 1 call, got %d", tr.calls)
	}
	if !strings.Contains(stderr, "more assignments exist") {
		t.Errorf("stderr should note more assignments exist:\n%s", stderr)
	}
}

func TestRunAssignments_PerPageSizesWalk(t *testing.T) {
	tr := &cannedTransport{status: 200, body: assignmentsPageComplete}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	_, _, _ = runAssignmentsOver(t, seam, assignmentsConfig{id: "per_0123", perPage: 7, perPageSet: true})
	if got := tr.lastQuery.Get("per_page"); got != "7" {
		t.Errorf("per_page = %q, want \"7\" (WithPageSize passed through)", got)
	}
}

// --- mid-walk failure ------------------------------------------------------

func TestRunAssignments_MidWalkFailurePartialAndIncomplete(t *testing.T) {
	tr := &seqMeTransport{steps: []seqMeResp{
		{status: 200, body: assignmentsPage("per_0123", "role_1", "Gathered Role", "c1")},
		{status: 500, body: `{"detail":"boom"}`},
	}}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, stderr := runAssignmentsOver(t, seam, assignmentsConfig{id: "per_0123"})
	if outcome == Success {
		t.Fatalf("a mid-walk failure must be non-zero, got Success")
	}
	if ExitCode(outcome) == 0 {
		t.Errorf("a mid-walk failure must exit non-zero, got exit 0 (outcome %v)", outcome)
	}
	if !strings.Contains(stdout, "Gathered Role") {
		t.Errorf("the partial set gathered so far should print:\n%s", stdout)
	}
	if !strings.Contains(stderr, "incomplete") {
		t.Errorf("stderr should note the result is incomplete and name the cause:\n%s", stderr)
	}
}

// --- error classification (via the shared classifier) ----------------------

func TestRunAssignments_NoCredentialsIsUsageError(t *testing.T) {
	tr := &cannedTransport{status: 200, body: assignmentsPageComplete}
	seam := &fakeMeSeam{ctx: noTokenContext(), transport: tr}

	outcome, stdout, stderr := runAssignmentsOver(t, seam, assignmentsConfig{id: "per_0123"})
	if outcome != UsageError || ExitCode(outcome) != 2 {
		t.Fatalf("outcome=%v exit=%d, want UsageError/2", outcome, ExitCode(outcome))
	}
	if !strings.Contains(strings.ToLower(stderr), "not authenticated") {
		t.Errorf("stderr should report not authenticated:\n%s", stderr)
	}
	if strings.TrimSpace(stdout) != "" {
		t.Errorf("no assignment data should be printed on a credential failure, got:\n%s", stdout)
	}
}

func TestRunAssignments_UnknownActorSurfacesAPIStatus(t *testing.T) {
	tr := &cannedTransport{status: 404, body: `{"detail":"Actor not found"}`}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, stderr := runAssignmentsOver(t, seam, assignmentsConfig{id: "per_ffff"})
	if outcome != APIError || ExitCode(outcome) != 3 {
		t.Fatalf("an unknown actor id should surface APIError/3, got %v/%d\nstderr: %s", outcome, ExitCode(outcome), stderr)
	}
	if !strings.Contains(stderr, "404") {
		t.Errorf("stderr should name the HTTP status (404):\n%s", stderr)
	}
	if strings.TrimSpace(stdout) != "" {
		t.Errorf("nothing should be printed to stdout on a not-found, got:\n%s", stdout)
	}
}

func TestRunAssignments_TransportErrorIsNetworkUnavailable(t *testing.T) {
	tr := &cannedTransport{netErr: errors.New("dial tcp: connection refused")}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, _, stderr := runAssignmentsOver(t, seam, assignmentsConfig{id: "per_0123"})
	if outcome != NetworkUnavailable || ExitCode(outcome) != 6 {
		t.Fatalf("outcome=%v exit=%d, want NetworkUnavailable/6\nstderr: %s", outcome, ExitCode(outcome), stderr)
	}
}

func TestRunAssignments_Non2xxClassifies(t *testing.T) {
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
		outcome, _, stderr := runAssignmentsOver(t, seam, assignmentsConfig{id: "per_0123"})
		if outcome != c.want || ExitCode(outcome) != c.code {
			t.Errorf("status %d: outcome=%v exit=%d, want %v/%d\nstderr: %s", c.status, outcome, ExitCode(outcome), c.want, c.code, stderr)
		}
	}
}

// --- resolve-before-call: a bad --output costs no request ------------------

func TestRunAssignments_BadOutputIsUsageErrorNoRequest(t *testing.T) {
	tr := &cannedTransport{status: 200, body: assignmentsPageComplete}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, _, stderr := runAssignmentsOver(t, seam, assignmentsConfig{id: "per_0123", outputFlag: "xml", outputPresent: true})
	if outcome != UsageError || ExitCode(outcome) != 2 {
		t.Fatalf("outcome=%v exit=%d, want UsageError/2\nstderr: %s", outcome, ExitCode(outcome), stderr)
	}
	if tr.calls != 0 {
		t.Errorf("a bad --output must be rejected before any request, got %d calls", tr.calls)
	}
}

// --- structured output emits the aggregated raw document --------------------

func TestRunAssignments_StructuredJSONEmitsAggregatedRawPayload(t *testing.T) {
	tr := &cannedTransport{status: 200, body: assignmentsPageComplete}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, _ := runAssignmentsOver(t, seam, assignmentsConfig{id: "per_0123", outputFlag: "json", outputPresent: true})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success", outcome)
	}
	for _, want := range []string{`"data"`, "asgn_1", `"actor_id"`, `"elected_until"`, `"focus"`, `"role"`} {
		if !strings.Contains(stdout, want) {
			t.Errorf("structured json should carry the raw payload, missing %q:\n%s", want, stdout)
		}
	}
	// Structured output must not carry the human projection's block labels nor the
	// per-page meta envelope.
	if strings.Contains(stdout, "Elected until:") {
		t.Errorf("structured json must not render the human projection:\n%s", stdout)
	}
	if strings.Contains(stdout, `"pagination"`) {
		t.Errorf("the aggregated document must drop the per-page meta envelope:\n%s", stdout)
	}
}

// --- command-level wiring --------------------------------------------------

// TestAssignmentsCommand_TargetsAssignmentsEndpoint pins the end-to-end path: a real
// `assignments per_0123` invocation reads /actors/{id}/assignments.
func TestAssignmentsCommand_TargetsAssignmentsEndpoint(t *testing.T) {
	tr := &cannedTransport{status: 200, body: assignmentsPageComplete}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	root := NewRootCommand()
	MustRegister(root, newAssignmentsCommand(seam))
	var out, errb bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&errb)
	outcome, _ := Run(root, []string{"assignments", "per_0123"})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success\nstderr: %s", outcome, errb.String())
	}
	if got := tr.lastPath; !strings.HasSuffix(got, "/actors/per_0123/assignments") {
		t.Errorf("path = %q, want it to target /actors/per_0123/assignments", got)
	}
}

// TestAssignmentsCommand_RequiresExactlyOneArg pins ExactArgs(1): zero/extra args is
// a usage error and sends no request (the transport tripwire).
func TestAssignmentsCommand_RequiresExactlyOneArg(t *testing.T) {
	for _, args := range [][]string{{"assignments"}, {"assignments", "per_0123", "per_0456"}} {
		tr := &cannedTransport{status: 200, body: assignmentsPageComplete}
		seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

		root := NewRootCommand()
		MustRegister(root, newAssignmentsCommand(seam))
		root.SetOut(&bytes.Buffer{})
		root.SetErr(&bytes.Buffer{})
		outcome, _ := Run(root, args)
		if outcome != UsageError {
			t.Errorf("%v should be a UsageError, got %v", args, outcome)
		}
		if tr.calls != 0 {
			t.Errorf("%v must send no request, got %d calls", args, tr.calls)
		}
	}
}

// TestAssignmentsCommand_UnknownFlagRejectedNoRequest pins that the command declares
// no filter flags / no --include: an unknown flag is a cobra usage error before any
// request (plan ADR-3).
func TestAssignmentsCommand_UnknownFlagRejectedNoRequest(t *testing.T) {
	for _, flag := range []string{"--include", "--kind", "--status", "--query"} {
		tr := &cannedTransport{status: 200, body: assignmentsPageComplete}
		seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

		root := NewRootCommand()
		MustRegister(root, newAssignmentsCommand(seam))
		root.SetOut(&bytes.Buffer{})
		root.SetErr(&bytes.Buffer{})
		outcome, _ := Run(root, []string{"assignments", "per_0123", flag, "x"})
		if outcome != UsageError {
			t.Errorf("%s on `assignments` should be a UsageError, got %v", flag, outcome)
		}
		if tr.calls != 0 {
			t.Errorf("%s on `assignments` must send no request, got %d calls", flag, tr.calls)
		}
	}
}
