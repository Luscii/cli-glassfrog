package cli

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/Luscii/cli-glassfrog/internal/apiclient"
	"github.com/Luscii/cli-glassfrog/internal/auth"
	"github.com/Luscii/cli-glassfrog/internal/glassfrog"
	"github.com/spf13/cobra"
)

// Canned GET /me/projects bodies for the command/branch tests. They use the API's
// snake_case names and carry the secret token nowhere (it rides the request
// header, asserted absent from output by runMeProjectsOver).
const (
	projectsBodyMulti = `{
      "data": [
        {"id": "proj_0123456789abcdef0123456789abcdef", "type": "project",
         "description": "Rebuild onboarding flow", "status": "current",
         "role_id": "role_0123456789abcdef0123456789abcdef",
         "individual_initiative": false, "has_sub_projects": true, "has_actions": true,
         "parent_project_id": null, "tags": ["marketing", "q2"], "created_at": "", "updated_at": ""},
        {"id": "proj_00000000000000000000000000000001", "type": "project",
         "description": "Audit vendor list", "status": "scheduled",
         "role_id": "role_00000000000000000000000000000001",
         "individual_initiative": false, "has_sub_projects": false, "has_actions": false,
         "parent_project_id": null, "tags": [], "created_at": "", "updated_at": ""}
      ],
      "meta": {"pagination": {"per_page": 25, "has_next_page": false, "next_cursor": ""}}
    }`

	projectsBodyEmpty = `{"data": [], "meta": {"pagination": {"per_page": 25, "has_next_page": false, "next_cursor": ""}}}`

	// A project with a null role_id (a non-role-owned, individual-initiative
	// project) — the projection renders the explicit no-role marker.
	projectsBodyNoRole = `{
      "data": [
        {"id": "proj_00000000000000000000000000000002", "type": "project",
         "description": "Personal spike", "status": "current",
         "role_id": null, "individual_initiative": true,
         "has_sub_projects": false, "has_actions": false,
         "parent_project_id": null, "tags": [], "created_at": "", "updated_at": ""}
      ],
      "meta": {"pagination": {"per_page": 25, "has_next_page": false, "next_cursor": ""}}
    }`

	projectsBodyHasNext = `{
      "data": [
        {"id": "proj_0123456789abcdef0123456789abcdef", "type": "project",
         "description": "Rebuild onboarding flow", "status": "current",
         "role_id": "role_0123456789abcdef0123456789abcdef",
         "individual_initiative": false, "has_sub_projects": false, "has_actions": false,
         "parent_project_id": null, "tags": [], "created_at": "", "updated_at": ""}
      ],
      "meta": {"pagination": {"per_page": 1, "has_next_page": true, "next_cursor": "abc"}}
    }`
)

// runMeProjectsOver drives the pure runMeProjects over a fake seam, returning the
// outcome and the captured stdout/stderr, and failing if the token leaks.
func runMeProjectsOver(t *testing.T, seam meSeam, status string) (Outcome, string, string) {
	t.Helper()
	var out, errb bytes.Buffer
	outcome, _ := runMeProjects(meProjectsConfig{
		seam:   seam,
		status: status,
		reqCtx: context.Background(),
		stdout: &out,
		stderr: &errb,
	})
	if strings.Contains(out.String()+errb.String(), meSecretToken) {
		t.Fatalf("the token leaked into output: %q", out.String()+errb.String())
	}
	return outcome, out.String(), errb.String()
}

// --- formatMeProjects (pure) ---------------------------------------------

func TestFormatMeProjects_RendersFieldsPerProject(t *testing.T) {
	resp := glassfrog.MyProjectsResponse{
		Data: []glassfrog.Project{
			{ID: "proj_aaa", Status: "current", Description: "Rebuild onboarding flow", RoleID: "role_aaa", HasSubProjects: true, HasActions: true, Tags: []string{"marketing", "q2"}},
			{ID: "proj_bbb", Status: "scheduled", Description: "", RoleID: "role_bbb"},
		},
	}
	out := formatMeProjects(resp)
	for _, want := range []string{
		"proj_aaa", "current", "Rebuild onboarding flow", "role_aaa", "marketing", "q2",
		"proj_bbb", "scheduled", "role_bbb",
		"sub-projects: yes", "actions: yes", "sub-projects: no", "actions: no",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("projection missing %q:\n%s", want, out)
		}
	}
	// A null/empty description renders the em-dash placeholder, never blank.
	if !strings.Contains(out, "—") {
		t.Errorf("a null description should render the em-dash placeholder:\n%s", out)
	}
}

// A null role_id (a non-role-owned project) renders the explicit no-role marker
// in the role slot, while still surfacing the project's id/status/description.
func TestFormatMeProjects_NullRoleRendersNoRoleMarker(t *testing.T) {
	resp := glassfrog.MyProjectsResponse{
		Data: []glassfrog.Project{
			{ID: "proj_ccc", Status: "current", Description: "Personal spike", RoleID: ""},
		},
	}
	out := formatMeProjects(resp)
	if !strings.Contains(out, "role: "+noRoleMarker) {
		t.Errorf("a null role_id should render the no-role marker %q in the role slot:\n%s", noRoleMarker, out)
	}
	for _, want := range []string{"proj_ccc", "current", "Personal spike"} {
		if !strings.Contains(out, want) {
			t.Errorf("a no-role project should still surface %q:\n%s", want, out)
		}
	}
}

// An empty list is the valid "you own no matching projects" answer and renders an
// explicit empty-result line, not nothing.
func TestFormatMeProjects_EmptyListRendersExplicitLine(t *testing.T) {
	out := formatMeProjects(glassfrog.MyProjectsResponse{})
	if strings.TrimSpace(out) == "" {
		t.Error("an empty list should render an explicit line, got blank output")
	}
	if strings.TrimRight(out, "\n") != "no projects" {
		t.Errorf("an empty list should render exactly `no projects`, got %q", out)
	}
}

// formatMeProjects renders only the projection; the more-available signal is the
// command's concern (stderr), so it must not leak into the projection body.
func TestFormatMeProjects_NoSignalInProjectionBody(t *testing.T) {
	resp := glassfrog.MyProjectsResponse{
		Data: []glassfrog.Project{{ID: "proj_aaa", Status: "current", Description: "x", RoleID: "role_aaa"}},
	}
	resp.Meta.Pagination.HasNextPage = true
	if strings.Contains(formatMeProjects(resp), "more projects") {
		t.Errorf("the more-available signal must not appear in the projection body:\n%s", formatMeProjects(resp))
	}
}

// --- runMeProjects branches ----------------------------------------------

func TestRunMeProjects_SuccessMultiProject(t *testing.T) {
	tr := &cannedTransport{status: 200, body: projectsBodyMulti}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, stderr := runMeProjectsOver(t, seam, "")
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success", outcome)
	}
	for _, want := range []string{
		"proj_0123456789abcdef0123456789abcdef", "current", "Rebuild onboarding flow",
		"role_0123456789abcdef0123456789abcdef", "marketing",
		"proj_00000000000000000000000000000001", "scheduled",
	} {
		if !strings.Contains(stdout, want) {
			t.Errorf("stdout missing %q:\n%s", want, stdout)
		}
	}
	if strings.TrimSpace(stderr) != "" {
		t.Errorf("a complete success should write nothing to stderr, got %q", stderr)
	}
	if tr.calls != 1 {
		t.Errorf("transport called %d times, want exactly 1", tr.calls)
	}
	// The request goes to the /me/projects endpoint.
	if !strings.HasSuffix(tr.lastPath, "/me/projects") {
		t.Errorf("request path = %q, want it to end with /me/projects", tr.lastPath)
	}
	// No --status: no query parameters.
	if got := tr.lastQuery.Encode(); got != "" {
		t.Errorf("with no --status, no query should be sent, got %q", got)
	}
}

func TestRunMeProjects_NullRoleRendersNoRoleMarker(t *testing.T) {
	tr := &cannedTransport{status: 200, body: projectsBodyNoRole}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, _ := runMeProjectsOver(t, seam, "")
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success", outcome)
	}
	if !strings.Contains(stdout, "role: "+noRoleMarker) {
		t.Errorf("a null role_id should render the no-role marker:\n%s", stdout)
	}
	for _, want := range []string{"proj_00000000000000000000000000000002", "current", "Personal spike"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("a no-role project should still surface %q:\n%s", want, stdout)
		}
	}
}

func TestRunMeProjects_EmptyListIsCleanSuccess(t *testing.T) {
	tr := &cannedTransport{status: 200, body: projectsBodyEmpty}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, stderr := runMeProjectsOver(t, seam, "")
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success", outcome)
	}
	if strings.TrimRight(stdout, "\n") != "no projects" {
		t.Errorf("an empty list should print exactly `no projects`, got %q", stdout)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Errorf("an empty list is not incomplete; stderr should be empty, got %q", stderr)
	}
}

func TestRunMeProjects_SupportedStatusFiltersRequest(t *testing.T) {
	tr := &cannedTransport{status: 200, body: projectsBodyMulti}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, _, _ := runMeProjectsOver(t, seam, "current")
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success", outcome)
	}
	if tr.calls != 1 {
		t.Errorf("transport called %d times, want exactly 1", tr.calls)
	}
	if got := tr.lastQuery.Get("status"); got != "current" {
		t.Errorf("--status current should add ?status=current, got query %v", tr.lastQuery)
	}
}

// An unsupported --status is rejected before any request: the tripwire transport
// is never reached, and assembly/build never happen (validateStatus runs first).
func TestRunMeProjects_UnsupportedStatusRejectedBeforeAnyRequest(t *testing.T) {
	tr := &cannedTransport{status: 200, body: projectsBodyMulti}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, _, stderr := runMeProjectsOver(t, seam, "in-progress")
	if outcome != UsageError {
		t.Fatalf("outcome = %v, want UsageError", outcome)
	}
	if !strings.Contains(stderr, "in-progress") {
		t.Errorf("stderr should name the unsupported value, got %q", stderr)
	}
	if !strings.Contains(stderr, "current") {
		t.Errorf("stderr should list the supported set, got %q", stderr)
	}
	if tr.calls != 0 {
		t.Errorf("an unsupported --status must issue no request, got %d calls", tr.calls)
	}
	if seam.assembleCalled || seam.newClientCalled {
		t.Errorf("validateStatus must run before assembly/build (assembled=%v built=%v)", seam.assembleCalled, seam.newClientCalled)
	}
}

func TestRunMeProjects_HasNextPageSignalsMoreOnStderr(t *testing.T) {
	tr := &cannedTransport{status: 200, body: projectsBodyHasNext}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, stderr := runMeProjectsOver(t, seam, "")
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success (incompleteness is still a success)", outcome)
	}
	if !strings.Contains(stdout, "proj_0123456789abcdef0123456789abcdef") {
		t.Errorf("the received first page should still print to stdout:\n%s", stdout)
	}
	// The more-available signal is on stderr (012's convention), pinned verbatim.
	if strings.TrimRight(stderr, "\n") != incompleteProjectsNote {
		t.Errorf("stderr should be exactly the pinned more-available note %q, got %q", incompleteProjectsNote, stderr)
	}
	// Exactly one request — the next page is signalled, never fetched.
	if tr.calls != 1 {
		t.Errorf("transport called %d times; a further page must be signalled, not fetched", tr.calls)
	}
}

func TestRunMeProjects_NoCredentialsIsUsageErrorNoRequest(t *testing.T) {
	tr := &cannedTransport{status: 200, body: projectsBodyMulti}
	ctx := apiclient.ConnectionContext{
		BaseURL: apiclient.BaseURL{Value: "https://example.test/api/v5", Source: apiclient.SourceFlag},
		Cred:    auth.Resolution{Source: auth.SourceNone},
	}
	seam := &fakeMeSeam{ctx: ctx, transport: tr}

	outcome, stdout, stderr := runMeProjectsOver(t, seam, "")
	if outcome != UsageError {
		t.Fatalf("outcome = %v, want UsageError", outcome)
	}
	if !strings.Contains(stderr, "auth login") {
		t.Errorf("stderr should point at `glassfrog auth login`, got %q", stderr)
	}
	if strings.TrimSpace(stdout) != "" {
		t.Errorf("no project data should print on a no-token failure, got %q", stdout)
	}
	if tr.calls != 0 {
		t.Errorf("an unauthenticated request must not be sent, got %d calls", tr.calls)
	}
}

func TestRunMeProjects_CredentialErrorIsRuntimeError(t *testing.T) {
	tr := &cannedTransport{status: 200, body: projectsBodyMulti}
	ctx := apiclient.ConnectionContext{
		BaseURL: apiclient.BaseURL{Value: "https://example.test/api/v5", Source: apiclient.SourceFlag},
		Cred:    auth.Resolution{Source: auth.SourceNone},
		CredErr: errors.New("the .glassfrogrc credentials file is malformed"),
	}
	seam := &fakeMeSeam{ctx: ctx, transport: tr}

	outcome, _, stderr := runMeProjectsOver(t, seam, "")
	if outcome != RuntimeError {
		t.Fatalf("outcome = %v, want RuntimeError", outcome)
	}
	if strings.TrimSpace(stderr) == "" {
		t.Error("a credential-file error should be reported on stderr")
	}
	if tr.calls != 0 {
		t.Errorf("a credential error must not send, got %d calls", tr.calls)
	}
}

func TestRunMeProjects_TransportFailureIsNetworkUnavailableNoRetry(t *testing.T) {
	tr := &cannedTransport{netErr: errors.New("dial tcp: connection refused")}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, stderr := runMeProjectsOver(t, seam, "")
	if outcome != NetworkUnavailable {
		t.Fatalf("outcome = %v, want NetworkUnavailable", outcome)
	}
	if strings.TrimSpace(stdout) != "" {
		t.Errorf("no projection should print on a transport failure, got %q", stdout)
	}
	if strings.TrimSpace(stderr) == "" {
		t.Error("a transport failure should report a cause on stderr")
	}
	if tr.calls != 1 {
		t.Errorf("the read must not retry, got %d calls", tr.calls)
	}
}

func TestRunMeProjects_NonStatus2xxIsAPIError(t *testing.T) {
	// A genuinely generic non-2xx (500): 401/403/429 now split into
	// PermissionError/RateLimited (API Error Extraction 015), so a 5xx represents
	// the residual generic APIError bucket this test pins.
	tr := &cannedTransport{status: 500, body: `{"error":"server error"}`}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, stderr := runMeProjectsOver(t, seam, "")
	if outcome != APIError {
		t.Fatalf("outcome = %v, want APIError", outcome)
	}
	if strings.TrimSpace(stdout) != "" {
		t.Errorf("no projection should print on a non-2xx, got %q", stdout)
	}
	if !strings.Contains(stderr, "500") {
		t.Errorf("stderr should name the 500 status, got %q", stderr)
	}
	// The non-2xx message names the status AND a generic next step — without
	// interpreting the status into a specific meaning (015/017's concern).
	if !strings.Contains(stderr, "retry") {
		t.Errorf("the non-2xx message should carry a generic next step, got %q", stderr)
	}
	for _, forbidden := range []string{"permission", "unauthorized", "forbidden", "rate limit"} {
		if strings.Contains(strings.ToLower(stderr), forbidden) {
			t.Errorf("the non-2xx message must not interpret the status (%q present): %q", forbidden, stderr)
		}
	}
}

func TestRunMeProjects_UndecodableBodyIsRuntimeError(t *testing.T) {
	tr := &cannedTransport{status: 200, body: `not json at all`}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, stderr := runMeProjectsOver(t, seam, "")
	if outcome != RuntimeError {
		t.Fatalf("outcome = %v, want RuntimeError", outcome)
	}
	if strings.TrimSpace(stdout) != "" {
		t.Errorf("no projection should print on a decode failure, got %q", stdout)
	}
	if !strings.Contains(stderr, "report it") {
		t.Errorf("the decode-failure message should carry a next step, got %q", stderr)
	}
}

func TestRunMeProjects_BaseURLErrorIsUsageErrorNothingSent(t *testing.T) {
	tr := &cannedTransport{status: 200, body: projectsBodyMulti}
	seam := &fakeMeSeam{
		ctx:          apiclient.ConnectionContext{},
		newClientErr: &apiclient.BaseURLError{Source: "--" + apiclient.FlagBaseURL},
		transport:    tr,
	}

	outcome, _, stderr := runMeProjectsOver(t, seam, "")
	if outcome != UsageError {
		t.Fatalf("outcome = %v, want UsageError", outcome)
	}
	if !strings.Contains(stderr, "base-url") && !strings.Contains(strings.ToLower(stderr), "base url") {
		t.Errorf("stderr should name the base-URL problem, got %q", stderr)
	}
	if tr.calls != 0 {
		t.Errorf("a base-URL error must not send, got %d calls", tr.calls)
	}
}

// --- newMeProjectsCommand integration (outcome → exit code, registration) -

// runMeProjectsCommand registers `me` under a real root (with the persistent
// --base-url flag) and the `projects` leaf under `me`, then dispatches
// `me projects [args]` through Run — pinning the command wiring AND the
// outcomeError → ExitCode path (3/6) the dispatch carrier enables. The seam is
// shared by both `me` and `me projects` (productionSeam binds the real one).
func runMeProjectsCommand(t *testing.T, seam meSeam, args ...string) (Outcome, int, string, string) {
	t.Helper()
	root := NewRootCommand()
	meCmd := newMeCommand(seam)
	MustRegister(root, meCmd)
	MustRegister(meCmd, newMeProjectsCommand(seam))
	var out, errb bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&errb)
	outcome, _ := Run(root, append([]string{"me", "projects"}, args...))
	return outcome, ExitCode(outcome), out.String(), errb.String()
}

func TestMeProjectsCommand_ExitCodesAcrossOutcomes(t *testing.T) {
	cases := []struct {
		name     string
		tr       *cannedTransport
		ctx      apiclient.ConnectionContext
		seamErr  error
		args     []string
		outcome  Outcome
		exitCode int
	}{
		{"success", &cannedTransport{status: 200, body: projectsBodyMulti}, validMeContext(), nil, nil, Success, 0},
		{"empty-success", &cannedTransport{status: 200, body: projectsBodyEmpty}, validMeContext(), nil, nil, Success, 0},
		{"no-role-success", &cannedTransport{status: 200, body: projectsBodyNoRole}, validMeContext(), nil, nil, Success, 0},
		{"has-next-success", &cannedTransport{status: 200, body: projectsBodyHasNext}, validMeContext(), nil, nil, Success, 0},
		{"filtered-success", &cannedTransport{status: 200, body: projectsBodyMulti}, validMeContext(), nil, []string{"--status", "current"}, Success, 0},
		{"unsupported-status", &cannedTransport{status: 200, body: projectsBodyMulti}, validMeContext(), nil, []string{"--status", "in-progress"}, UsageError, 2},
		{"api-error", &cannedTransport{status: 500, body: `{}`}, validMeContext(), nil, nil, APIError, 3},
		{"network-unavailable", &cannedTransport{netErr: errors.New("connection refused")}, validMeContext(), nil, nil, NetworkUnavailable, 6},
		{"decode-error", &cannedTransport{status: 200, body: `nope`}, validMeContext(), nil, nil, RuntimeError, 1},
		{"stray-arg", &cannedTransport{status: 200, body: projectsBodyMulti}, validMeContext(), nil, []string{"extra"}, UsageError, 2},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			seam := &fakeMeSeam{ctx: tc.ctx, newClientErr: tc.seamErr, transport: tc.tr}
			outcome, code, stdout, stderr := runMeProjectsCommand(t, seam, tc.args...)
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

// A stray positional argument is rejected by cobra.NoArgs before any API call.
func TestMeProjectsCommand_StrayArgSendsNothing(t *testing.T) {
	tr := &cannedTransport{status: 200, body: projectsBodyMulti}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}
	outcome, code, _, _ := runMeProjectsCommand(t, seam, "extra-argument")
	if outcome != UsageError || code != 2 {
		t.Fatalf("stray arg: outcome=%v code=%d, want UsageError/2", outcome, code)
	}
	if tr.calls != 0 {
		t.Errorf("a rejected invocation must not send a request, got %d calls", tr.calls)
	}
}

// The persistent --base-url value reaches the seam's assemble (inherited from the
// root through `me` to `me projects`).
func TestMeProjectsCommand_InheritsBaseURLFlag(t *testing.T) {
	tr := &cannedTransport{status: 200, body: projectsBodyMulti}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}
	_, _, _, _ = runMeProjectsCommand(t, seam, "--base-url", "https://flag.test/api/v5")
	if seam.assembledBaseURL != "https://flag.test/api/v5" {
		t.Errorf("assemble received base URL %q, want the inherited flag value", seam.assembledBaseURL)
	}
}

// The `projects` leaf declares no --base-url flag of its own — it inherits the
// root's persistent one.
func TestMeProjectsCommand_DeclaresNoOwnBaseURLFlag(t *testing.T) {
	cmd := newMeProjectsCommand(&fakeMeSeam{})
	if cmd.Flags().Lookup(apiclient.FlagBaseURL) != nil {
		t.Errorf("the projects leaf must not declare its own --%s flag; it is inherited", apiclient.FlagBaseURL)
	}
}

// The `projects` leaf declares a local --status flag.
func TestMeProjectsCommand_DeclaresLocalStatusFlag(t *testing.T) {
	cmd := newMeProjectsCommand(&fakeMeSeam{})
	if cmd.Flags().Lookup("status") == nil {
		t.Error("the projects leaf should declare a local --status flag")
	}
}

// /me/projects offers no include parameter (ADR-2), so the leaf declares NO
// --include flag (unlike `me --include roles`).
func TestMeProjectsCommand_DeclaresNoIncludeFlag(t *testing.T) {
	cmd := newMeProjectsCommand(&fakeMeSeam{})
	if cmd.Flags().Lookup("include") != nil {
		t.Error("the projects leaf must not declare an --include flag (ADR-2: /me/projects offers no include)")
	}
}

// No ?include is ever added to the request, on any path.
func TestRunMeProjects_NeverSendsInclude(t *testing.T) {
	tr := &cannedTransport{status: 200, body: projectsBodyMulti}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}
	_, _, _ = runMeProjectsOver(t, seam, "current")
	if got := tr.lastQuery.Get("include"); got != "" {
		t.Errorf("no ?include should ever be sent, got include=%q", got)
	}
}

// `me` remains BOTH runnable (its own RunE) AND a parent of the `projects` leaf,
// as it already is for `roles`/`actions` — the guard validates a command at its
// own registration, so a third sibling attaches without tripping leaf-xor-group.
func TestMeProjectsCommand_MeIsRunnableWithProjectsChild(t *testing.T) {
	root := NewRootCommand()
	meCmd := newMeCommand(&fakeMeSeam{})
	MustRegister(root, meCmd)
	MustRegister(meCmd, newMeProjectsCommand(&fakeMeSeam{}))

	if meCmd.RunE == nil {
		t.Error("me should remain runnable (its own RunE) after gaining a child")
	}
	var projectsChild *cobra.Command
	for _, c := range meCmd.Commands() {
		if c.Name() == "projects" {
			projectsChild = c
		}
	}
	if projectsChild == nil {
		t.Fatal("me should have a `projects` child registered under it")
	}
	if projectsChild.RunE == nil {
		t.Error("the projects child should carry an action")
	}
}

// The full Assemble wiring must not panic — pins that the production wiring of
// `me projects` (alongside `me roles` and `me actions`) passes the registration
// guard end to end, and that `me projects` resolves through the guard.
func TestAssemble_WiresMeProjectsWithoutPanic(t *testing.T) {
	root := Assemble()
	projectsCmd, _, err := root.Find([]string{"me", "projects"})
	if err != nil || projectsCmd == nil || projectsCmd.Name() != "projects" {
		t.Fatalf("Assemble should wire `me projects`, got %v (err %v)", projectsCmd, err)
	}
	// The sibling leaves must still resolve — wiring one leaf does not disturb the
	// others under `me`.
	for _, sibling := range []string{"roles", "actions"} {
		cmd, _, err := root.Find([]string{"me", sibling})
		if err != nil || cmd == nil || cmd.Name() != sibling {
			t.Fatalf("Assemble should still wire `me %s` alongside `me projects`, got %v (err %v)", sibling, cmd, err)
		}
	}
}
