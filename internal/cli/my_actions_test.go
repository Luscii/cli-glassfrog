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

// Canned GET /me/actions bodies for the command/branch tests. They use the API's
// snake_case names and carry the secret token nowhere (it rides the request
// header, asserted absent from output by runMyActionsOver).
const (
	actionsBodyMulti = `{
      "data": [
        {"id": "actn_0123456789abcdef0123456789abcdef", "type": "action",
         "description": "Review PR #6818", "status": "current",
         "role_id": "role_0123456789abcdef0123456789abcdef",
         "individual_initiative": false, "parent_project_id": null,
         "tags": ["marketing", "q2"], "created_at": "", "updated_at": ""},
        {"id": "actn_00000000000000000000000000000001", "type": "action",
         "description": null, "status": "waiting",
         "role_id": "role_00000000000000000000000000000001",
         "individual_initiative": true, "parent_project_id": null,
         "tags": [], "created_at": "", "updated_at": ""}
      ],
      "meta": {"pagination": {"per_page": 25, "has_next_page": false, "next_cursor": ""}}
    }`

	actionsBodyEmpty = `{"data": [], "meta": {"pagination": {"per_page": 25, "has_next_page": false, "next_cursor": ""}}}`

	actionsBodyHasNext = `{
      "data": [
        {"id": "actn_0123456789abcdef0123456789abcdef", "type": "action",
         "description": "Review PR #6818", "status": "current",
         "role_id": "role_0123456789abcdef0123456789abcdef",
         "individual_initiative": false, "parent_project_id": null,
         "tags": [], "created_at": "", "updated_at": ""}
      ],
      "meta": {"pagination": {"per_page": 1, "has_next_page": true, "next_cursor": "abc"}}
    }`
)

// runMyActionsOver drives the pure runMyActions over a fake seam, returning the
// outcome and the captured stdout/stderr, and failing if the token leaks.
func runMyActionsOver(t *testing.T, seam meSeam, status string) (Outcome, string, string) {
	t.Helper()
	var out, errb bytes.Buffer
	outcome, _ := runMyActions(myActionsConfig{
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

// --- formatMyActions (pure) ----------------------------------------------

func TestFormatMyActions_RendersFieldsPerAction(t *testing.T) {
	resp := glassfrog.MyActionsResponse{
		Data: []glassfrog.Action{
			{ID: "actn_aaa", Status: "current", Description: "Review PR #6818", RoleID: "role_aaa", Tags: []string{"marketing", "q2"}},
			{ID: "actn_bbb", Status: "waiting", Description: "", RoleID: "role_bbb"},
		},
	}
	out := formatMyActions(resp)
	for _, want := range []string{
		"actn_aaa", "current", "Review PR #6818", "role_aaa", "marketing", "q2",
		"actn_bbb", "waiting", "role_bbb",
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

// An empty list is the valid "you own no matching actions" answer and renders an
// explicit empty-result line, not nothing.
func TestFormatMyActions_EmptyListRendersExplicitLine(t *testing.T) {
	out := formatMyActions(glassfrog.MyActionsResponse{})
	if strings.TrimSpace(out) == "" {
		t.Error("an empty list should render an explicit line, got blank output")
	}
	if strings.TrimRight(out, "\n") != "No actions." {
		t.Errorf("an empty list should render exactly `No actions.`, got %q", out)
	}
}

// formatMyActions renders only the projection; the more-available signal is the
// command's concern (stderr), so it must not leak into the projection body.
func TestFormatMyActions_NoSignalInProjectionBody(t *testing.T) {
	resp := glassfrog.MyActionsResponse{
		Data: []glassfrog.Action{{ID: "actn_aaa", Status: "current", Description: "x", RoleID: "role_aaa"}},
	}
	resp.Meta.Pagination.HasNextPage = true
	if strings.Contains(formatMyActions(resp), "more results") {
		t.Errorf("the more-available signal must not appear in the projection body:\n%s", formatMyActions(resp))
	}
}

// --- runMyActions branches -----------------------------------------------

func TestRunMyActions_SuccessMultiAction(t *testing.T) {
	tr := &cannedTransport{status: 200, body: actionsBodyMulti}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, stderr := runMyActionsOver(t, seam, "")
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success", outcome)
	}
	for _, want := range []string{
		"actn_0123456789abcdef0123456789abcdef", "current", "Review PR #6818",
		"role_0123456789abcdef0123456789abcdef", "marketing",
		"actn_00000000000000000000000000000001", "waiting",
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
	// No --status: no query parameters.
	if got := tr.lastQuery.Encode(); got != "" {
		t.Errorf("with no --status, no query should be sent, got %q", got)
	}
}

func TestRunMyActions_EmptyListIsCleanSuccess(t *testing.T) {
	tr := &cannedTransport{status: 200, body: actionsBodyEmpty}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, stderr := runMyActionsOver(t, seam, "")
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success", outcome)
	}
	if strings.TrimRight(stdout, "\n") != "No actions." {
		t.Errorf("an empty list should print exactly `No actions.`, got %q", stdout)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Errorf("an empty list is not incomplete; stderr should be empty, got %q", stderr)
	}
}

func TestRunMyActions_SupportedStatusFiltersRequest(t *testing.T) {
	tr := &cannedTransport{status: 200, body: actionsBodyMulti}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, _, _ := runMyActionsOver(t, seam, "current")
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
func TestRunMyActions_UnsupportedStatusRejectedBeforeAnyRequest(t *testing.T) {
	tr := &cannedTransport{status: 200, body: actionsBodyMulti}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, _, stderr := runMyActionsOver(t, seam, "done")
	if outcome != UsageError {
		t.Fatalf("outcome = %v, want UsageError", outcome)
	}
	if !strings.Contains(stderr, "done") {
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

func TestRunMyActions_HasNextPageSignalsMoreOnStderr(t *testing.T) {
	tr := &cannedTransport{status: 200, body: actionsBodyHasNext}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, stderr := runMyActionsOver(t, seam, "")
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success (incompleteness is still a success)", outcome)
	}
	if !strings.Contains(stdout, "actn_0123456789abcdef0123456789abcdef") {
		t.Errorf("the received first page should still print to stdout:\n%s", stdout)
	}
	// The more-available signal is on stderr (012's convention), pinned verbatim.
	if strings.TrimRight(stderr, "\n") != incompleteActionsNote {
		t.Errorf("stderr should be exactly the pinned more-available note %q, got %q", incompleteActionsNote, stderr)
	}
	// Exactly one request — the next page is signalled, never fetched.
	if tr.calls != 1 {
		t.Errorf("transport called %d times; a further page must be signalled, not fetched", tr.calls)
	}
}

func TestRunMyActions_NoCredentialsIsUsageErrorNoRequest(t *testing.T) {
	tr := &cannedTransport{status: 200, body: actionsBodyMulti}
	ctx := apiclient.ConnectionContext{
		BaseURL: apiclient.BaseURL{Value: "https://example.test/api/v5", Source: apiclient.SourceFlag},
		Cred:    auth.Resolution{Source: auth.SourceNone},
	}
	seam := &fakeMeSeam{ctx: ctx, transport: tr}

	outcome, stdout, stderr := runMyActionsOver(t, seam, "")
	if outcome != UsageError {
		t.Fatalf("outcome = %v, want UsageError", outcome)
	}
	if !strings.Contains(stderr, "auth login") {
		t.Errorf("stderr should point at `glassfrog auth login`, got %q", stderr)
	}
	if strings.TrimSpace(stdout) != "" {
		t.Errorf("no action data should print on a no-token failure, got %q", stdout)
	}
	if tr.calls != 0 {
		t.Errorf("an unauthenticated request must not be sent, got %d calls", tr.calls)
	}
}

func TestRunMyActions_CredentialErrorIsRuntimeError(t *testing.T) {
	tr := &cannedTransport{status: 200, body: actionsBodyMulti}
	ctx := apiclient.ConnectionContext{
		BaseURL: apiclient.BaseURL{Value: "https://example.test/api/v5", Source: apiclient.SourceFlag},
		Cred:    auth.Resolution{Source: auth.SourceNone},
		CredErr: errors.New("the .glassfrogrc credentials file is malformed"),
	}
	seam := &fakeMeSeam{ctx: ctx, transport: tr}

	outcome, _, stderr := runMyActionsOver(t, seam, "")
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

func TestRunMyActions_TransportFailureIsNetworkUnavailableNoRetry(t *testing.T) {
	tr := &cannedTransport{netErr: errors.New("dial tcp: connection refused")}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, stderr := runMyActionsOver(t, seam, "")
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

func TestRunMyActions_NonStatus2xxIsAPIError(t *testing.T) {
	tr := &cannedTransport{status: 403, body: `{"error":"forbidden"}`}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, stderr := runMyActionsOver(t, seam, "")
	if outcome != APIError {
		t.Fatalf("outcome = %v, want APIError", outcome)
	}
	if strings.TrimSpace(stdout) != "" {
		t.Errorf("no projection should print on a non-2xx, got %q", stdout)
	}
	if !strings.Contains(stderr, "403") {
		t.Errorf("stderr should name the 403 status, got %q", stderr)
	}
	// The non-2xx message names the status AND a generic next step — without
	// interpreting the status into a specific meaning (015/017's concern).
	if !strings.Contains(stderr, "retry") {
		t.Errorf("the non-2xx message should carry a generic next step, got %q", stderr)
	}
}

func TestRunMyActions_UndecodableBodyIsRuntimeError(t *testing.T) {
	tr := &cannedTransport{status: 200, body: `not json at all`}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, stderr := runMyActionsOver(t, seam, "")
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

func TestRunMyActions_BaseURLErrorIsUsageErrorNothingSent(t *testing.T) {
	tr := &cannedTransport{status: 200, body: actionsBodyMulti}
	seam := &fakeMeSeam{
		ctx:          apiclient.ConnectionContext{},
		newClientErr: &apiclient.BaseURLError{Source: "--" + apiclient.FlagBaseURL},
		transport:    tr,
	}

	outcome, _, stderr := runMyActionsOver(t, seam, "")
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

// --- newMyActionsCommand integration (outcome → exit code, registration) --

// runMyActionsCommand registers `me` under a real root (with the persistent
// --base-url flag) and the `actions` leaf under `me`, then dispatches
// `me actions [args]` through Run — pinning the command wiring AND the
// outcomeError → ExitCode path (3/6) the dispatch carrier enables. The seam is
// shared by both `me` and `me actions` (productionSeam binds the real one).
func runMyActionsCommand(t *testing.T, seam meSeam, args ...string) (Outcome, int, string, string) {
	t.Helper()
	root := NewRootCommand()
	meCmd := newMeCommand(seam)
	MustRegister(root, meCmd)
	MustRegister(meCmd, newMyActionsCommand(seam))
	var out, errb bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&errb)
	outcome, _ := Run(root, append([]string{"me", "actions"}, args...))
	return outcome, ExitCode(outcome), out.String(), errb.String()
}

func TestMyActionsCommand_ExitCodesAcrossOutcomes(t *testing.T) {
	cases := []struct {
		name     string
		tr       *cannedTransport
		ctx      apiclient.ConnectionContext
		seamErr  error
		args     []string
		outcome  Outcome
		exitCode int
	}{
		{"success", &cannedTransport{status: 200, body: actionsBodyMulti}, validMeContext(), nil, nil, Success, 0},
		{"empty-success", &cannedTransport{status: 200, body: actionsBodyEmpty}, validMeContext(), nil, nil, Success, 0},
		{"has-next-success", &cannedTransport{status: 200, body: actionsBodyHasNext}, validMeContext(), nil, nil, Success, 0},
		{"filtered-success", &cannedTransport{status: 200, body: actionsBodyMulti}, validMeContext(), nil, []string{"--status", "current"}, Success, 0},
		{"unsupported-status", &cannedTransport{status: 200, body: actionsBodyMulti}, validMeContext(), nil, []string{"--status", "done"}, UsageError, 2},
		{"api-error", &cannedTransport{status: 403, body: `{}`}, validMeContext(), nil, nil, APIError, 3},
		{"network-unavailable", &cannedTransport{netErr: errors.New("connection refused")}, validMeContext(), nil, nil, NetworkUnavailable, 6},
		{"decode-error", &cannedTransport{status: 200, body: `nope`}, validMeContext(), nil, nil, RuntimeError, 1},
		{"stray-arg", &cannedTransport{status: 200, body: actionsBodyMulti}, validMeContext(), nil, []string{"extra"}, UsageError, 2},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			seam := &fakeMeSeam{ctx: tc.ctx, newClientErr: tc.seamErr, transport: tc.tr}
			outcome, code, stdout, stderr := runMyActionsCommand(t, seam, tc.args...)
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
func TestMyActionsCommand_StrayArgSendsNothing(t *testing.T) {
	tr := &cannedTransport{status: 200, body: actionsBodyMulti}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}
	outcome, code, _, _ := runMyActionsCommand(t, seam, "extra-argument")
	if outcome != UsageError || code != 2 {
		t.Fatalf("stray arg: outcome=%v code=%d, want UsageError/2", outcome, code)
	}
	if tr.calls != 0 {
		t.Errorf("a rejected invocation must not send a request, got %d calls", tr.calls)
	}
}

// The persistent --base-url value reaches the seam's assemble (inherited from the
// root through `me` to `me actions`).
func TestMyActionsCommand_InheritsBaseURLFlag(t *testing.T) {
	tr := &cannedTransport{status: 200, body: actionsBodyMulti}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}
	_, _, _, _ = runMyActionsCommand(t, seam, "--base-url", "https://flag.test/api/v5")
	if seam.assembledBaseURL != "https://flag.test/api/v5" {
		t.Errorf("assemble received base URL %q, want the inherited flag value", seam.assembledBaseURL)
	}
}

// The `actions` leaf declares no --base-url flag of its own — it inherits the
// root's persistent one. A locally-declared flag would shadow the inherited
// value; this pins that it does not.
func TestMyActionsCommand_DeclaresNoOwnBaseURLFlag(t *testing.T) {
	cmd := newMyActionsCommand(&fakeMeSeam{})
	if cmd.Flags().Lookup(apiclient.FlagBaseURL) != nil {
		t.Errorf("the actions leaf must not declare its own --%s flag; it is inherited", apiclient.FlagBaseURL)
	}
}

// The `actions` leaf declares a local --status flag.
func TestMyActionsCommand_DeclaresLocalStatusFlag(t *testing.T) {
	cmd := newMyActionsCommand(&fakeMeSeam{})
	if cmd.Flags().Lookup("status") == nil {
		t.Error("the actions leaf should declare a local --status flag")
	}
}

// `me` remains BOTH runnable (its own RunE) AND a parent of the `actions` leaf,
// as it already is for `roles` — the guard validates a command at its own
// registration, so the second sibling attaches without tripping leaf-xor-group.
func TestMyActionsCommand_MeIsRunnableWithActionsChild(t *testing.T) {
	root := NewRootCommand()
	meCmd := newMeCommand(&fakeMeSeam{})
	MustRegister(root, meCmd)
	MustRegister(meCmd, newMyActionsCommand(&fakeMeSeam{}))

	if meCmd.RunE == nil {
		t.Error("me should remain runnable (its own RunE) after gaining a child")
	}
	var actionsChild *cobra.Command
	for _, c := range meCmd.Commands() {
		if c.Name() == "actions" {
			actionsChild = c
		}
	}
	if actionsChild == nil {
		t.Fatal("me should have an `actions` child registered under it")
	}
	if actionsChild.RunE == nil {
		t.Error("the actions child should carry an action")
	}
}

// The full Assemble wiring must not panic — pins that the production wiring of
// `me` + `me actions` (alongside `me roles`) passes the registration guard end
// to end, and that `me actions` resolves through the guard.
func TestAssemble_WiresMeActionsWithoutPanic(t *testing.T) {
	root := Assemble()
	actionsCmd, _, err := root.Find([]string{"me", "actions"})
	if err != nil || actionsCmd == nil || actionsCmd.Name() != "actions" {
		t.Fatalf("Assemble should wire `me actions`, got %v (err %v)", actionsCmd, err)
	}
	// The sibling `me roles` must still resolve — wiring one leaf does not
	// disturb the other under `me`.
	rolesCmd, _, err := root.Find([]string{"me", "roles"})
	if err != nil || rolesCmd == nil || rolesCmd.Name() != "roles" {
		t.Fatalf("Assemble should still wire `me roles` alongside `me actions`, got %v (err %v)", rolesCmd, err)
	}
}
