package cli

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/Luscii/cli-glassfrog/internal/apiclient"
	"github.com/Luscii/cli-glassfrog/internal/auth"
	"github.com/spf13/cobra"
)

// Canned GET /me/roles bodies for the command/branch tests. They use the API's
// snake_case names and carry the secret token nowhere (it rides the request
// header, asserted absent from output by runMeRolesOver).
const (
	rolesBodyMulti = `{
      "data": [
        {"id": "role_0123456789abcdef0123456789abcdef", "name": "Marketing Lead",
         "purpose": "A market that knows us",
         "domains": [{"description": "The marketing budget"}],
         "accountabilities": [{"description": "Defining the campaign"}]},
        {"id": "role_00000000000000000000000000000001", "name": "Treasurer",
         "purpose": null, "domains": [], "accountabilities": []}
      ],
      "meta": {"pagination": {"per_page": 25, "has_next_page": false, "next_cursor": ""}}
    }`

	rolesBodyEmpty = `{"data": [], "meta": {"pagination": {"per_page": 25, "has_next_page": false, "next_cursor": ""}}}`

	rolesBodyHasNext = `{
      "data": [
        {"id": "role_0123456789abcdef0123456789abcdef", "name": "Marketing Lead",
         "purpose": "p", "domains": [], "accountabilities": []}
      ],
      "meta": {"pagination": {"per_page": 1, "has_next_page": true, "next_cursor": "abc"}}
    }`
)

// runMeRolesOver drives the pure runMeRoles over a fake seam, returning the
// outcome and the captured stdout/stderr, and failing if the token leaks.
func runMeRolesOver(t *testing.T, seam meSeam) (Outcome, string, string) {
	t.Helper()
	var out, errb bytes.Buffer
	outcome, _ := runMeRoles(meRolesConfig{
		seam:   seam,
		reqCtx: context.Background(),
		stdout: &out,
		stderr: &errb,
	})
	if strings.Contains(out.String()+errb.String(), meSecretToken) {
		t.Fatalf("the token leaked into output: %q", out.String()+errb.String())
	}
	return outcome, out.String(), errb.String()
}

// --- runMeRoles branches ---------------------------------------------------

func TestRunMeRoles_SuccessMultiRole(t *testing.T) {
	tr := &cannedTransport{status: 200, body: rolesBodyMulti}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, stderr := runMeRolesOver(t, seam)
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success", outcome)
	}
	for _, want := range []string{
		"Marketing Lead (role_0123456789abcdef0123456789abcdef)",
		"Purpose: A market that knows us",
		"Domains:", "The marketing budget",
		"Accountabilities:", "Defining the campaign",
		"Treasurer", "(no purpose set)",
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
	// The path is /me/roles; no query parameters this slice.
	if got := tr.lastQuery.Encode(); got != "" {
		t.Errorf("me roles should send no query parameters, got %q", got)
	}
}

func TestRunMeRoles_EmptyListIsCleanSuccess(t *testing.T) {
	tr := &cannedTransport{status: 200, body: rolesBodyEmpty}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, stderr := runMeRolesOver(t, seam)
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success", outcome)
	}
	if strings.TrimRight(stdout, "\n") != "No roles." {
		t.Errorf("an empty list should print exactly `No roles.`, got %q", stdout)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Errorf("an empty list is not incomplete; stderr should be empty, got %q", stderr)
	}
}

func TestRunMeRoles_HasNextPageSignalsIncompleteOnStderr(t *testing.T) {
	tr := &cannedTransport{status: 200, body: rolesBodyHasNext}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, stderr := runMeRolesOver(t, seam)
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success (incompleteness is still a success)", outcome)
	}
	// Projection on stdout.
	if !strings.Contains(stdout, "Marketing Lead") {
		t.Errorf("the received roles should still print to stdout:\n%s", stdout)
	}
	// Incompleteness note on stderr, never interleaved into stdout. Pin the EXACT
	// note text (interface-cli specifies it verbatim) so wording can't drift
	// silently — Fprintln appends a trailing newline, so the line equals the
	// constant plus "\n".
	if stderr != incompleteRolesNote+"\n" {
		t.Errorf("stderr should be exactly the pinned incompleteness note, got %q", stderr)
	}
	if strings.Contains(stdout, "incomplete") {
		t.Errorf("the incompleteness note must not appear on stdout:\n%s", stdout)
	}
}

func TestRunMeRoles_NoCredentialsIsUsageErrorNoRequest(t *testing.T) {
	tr := &cannedTransport{status: 200, body: rolesBodyMulti}
	ctx := apiclient.ConnectionContext{
		BaseURL: apiclient.BaseURL{Value: "https://example.test/api/v5", Source: apiclient.SourceFlag},
		Cred:    auth.Resolution{Source: auth.SourceNone},
	}
	seam := &fakeMeSeam{ctx: ctx, transport: tr}

	outcome, stdout, stderr := runMeRolesOver(t, seam)
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

func TestRunMeRoles_CredentialErrorIsRuntimeError(t *testing.T) {
	tr := &cannedTransport{status: 200, body: rolesBodyMulti}
	ctx := apiclient.ConnectionContext{
		BaseURL: apiclient.BaseURL{Value: "https://example.test/api/v5", Source: apiclient.SourceFlag},
		Cred:    auth.Resolution{Source: auth.SourceNone},
		CredErr: errors.New("the .glassfrogrc credentials file is malformed"),
	}
	seam := &fakeMeSeam{ctx: ctx, transport: tr}

	outcome, _, stderr := runMeRolesOver(t, seam)
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

func TestRunMeRoles_TransportFailureIsNetworkUnavailableNoRetry(t *testing.T) {
	tr := &cannedTransport{netErr: errors.New("dial tcp: connection refused")}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, stderr := runMeRolesOver(t, seam)
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

func TestRunMeRoles_NonStatus2xxIsAPIError(t *testing.T) {
	// A genuinely generic non-2xx (500): 401/403/429 now split into
	// PermissionError/RateLimited (API Error Extraction 015), so a 5xx represents
	// the residual generic APIError bucket this test pins.
	tr := &cannedTransport{status: 500, body: `{"error":"server error"}`}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, stderr := runMeRolesOver(t, seam)
	if outcome != APIError {
		t.Fatalf("outcome = %v, want APIError", outcome)
	}
	if strings.TrimSpace(stdout) != "" {
		t.Errorf("no projection should print on a non-2xx, got %q", stdout)
	}
	if !strings.Contains(stderr, "500") {
		t.Errorf("stderr should name the 500 status, got %q", stderr)
	}
	// F-1 (validate round 1): the non-2xx message names the status AND a concrete,
	// generic next step — without interpreting the status into a specific meaning.
	if !strings.Contains(stderr, "retry") {
		t.Errorf("the non-2xx message should carry a generic next step, got %q", stderr)
	}
}

// 031 ADR-2: an undecodable 2xx body now classifies as APIError (exit 3), not
// RuntimeError (exit 1); the cause/next-step message is unchanged.
func TestRunMeRoles_UndecodableBodyIsAPIError(t *testing.T) {
	tr := &cannedTransport{status: 200, body: `not json at all`}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, stderr := runMeRolesOver(t, seam)
	if outcome != APIError {
		t.Fatalf("outcome = %v, want APIError", outcome)
	}
	if strings.TrimSpace(stdout) != "" {
		t.Errorf("no projection should print on a decode failure, got %q", stdout)
	}
	if strings.TrimSpace(stderr) == "" {
		t.Error("a decode failure should be reported on stderr")
	}
	// F-1 (validate round 1): the decode-failure message names the cause (shape
	// mismatch) AND the next step (report it).
	if !strings.Contains(stderr, "report it") {
		t.Errorf("the decode-failure message should carry a next step, got %q", stderr)
	}
}

func TestRunMeRoles_BaseURLErrorIsUsageErrorNothingSent(t *testing.T) {
	tr := &cannedTransport{status: 200, body: rolesBodyMulti}
	seam := &fakeMeSeam{
		ctx:          apiclient.ConnectionContext{},
		newClientErr: &apiclient.BaseURLError{Source: "--" + apiclient.FlagBaseURL},
		transport:    tr,
	}

	outcome, _, stderr := runMeRolesOver(t, seam)
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

// --- newMeRolesCommand integration (outcome → exit code, registration) -----

// runMeRolesCommand registers `me` under a real root (with the persistent
// --base-url flag) and the `roles` leaf under `me`, then dispatches
// `me roles [args]` through Run — pinning the command wiring AND the
// outcomeError → ExitCode path (3/6) the dispatch carrier enables. The seam is
// shared by both `me` and `me roles` (productionSeam binds the real one).
func runMeRolesCommand(t *testing.T, seam meSeam, args ...string) (Outcome, int, string, string) {
	t.Helper()
	root := NewRootCommand()
	meCmd := newMeCommand(seam)
	MustRegister(root, meCmd)
	MustRegister(meCmd, newMeRolesCommand(seam))
	var out, errb bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&errb)
	outcome, _ := Run(root, append([]string{"me", "roles"}, args...))
	return outcome, ExitCode(outcome), out.String(), errb.String()
}

func TestMeRolesCommand_ExitCodesAcrossOutcomes(t *testing.T) {
	cases := []struct {
		name     string
		tr       *cannedTransport
		ctx      apiclient.ConnectionContext
		seamErr  error
		args     []string
		outcome  Outcome
		exitCode int
	}{
		{"success", &cannedTransport{status: 200, body: rolesBodyMulti}, validMeContext(), nil, nil, Success, 0},
		{"empty-success", &cannedTransport{status: 200, body: rolesBodyEmpty}, validMeContext(), nil, nil, Success, 0},
		{"has-next-success", &cannedTransport{status: 200, body: rolesBodyHasNext}, validMeContext(), nil, nil, Success, 0},
		{"api-error", &cannedTransport{status: 500, body: `{}`}, validMeContext(), nil, nil, APIError, 3},
		{"network-unavailable", &cannedTransport{netErr: errors.New("connection refused")}, validMeContext(), nil, nil, NetworkUnavailable, 6},
		{"decode-error", &cannedTransport{status: 200, body: `nope`}, validMeContext(), nil, nil, APIError, 3},
		{"stray-arg", &cannedTransport{status: 200, body: rolesBodyMulti}, validMeContext(), nil, []string{"extra"}, UsageError, 2},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			seam := &fakeMeSeam{ctx: tc.ctx, newClientErr: tc.seamErr, transport: tc.tr}
			outcome, code, stdout, stderr := runMeRolesCommand(t, seam, tc.args...)
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
func TestMeRolesCommand_StrayArgSendsNothing(t *testing.T) {
	tr := &cannedTransport{status: 200, body: rolesBodyMulti}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}
	outcome, code, _, _ := runMeRolesCommand(t, seam, "extra-argument")
	if outcome != UsageError || code != 2 {
		t.Fatalf("stray arg: outcome=%v code=%d, want UsageError/2", outcome, code)
	}
	if tr.calls != 0 {
		t.Errorf("a rejected invocation must not send a request, got %d calls", tr.calls)
	}
}

// The persistent --base-url value reaches the seam's assemble (inherited from the
// root through `me` to `me roles`).
func TestMeRolesCommand_InheritsBaseURLFlag(t *testing.T) {
	tr := &cannedTransport{status: 200, body: rolesBodyMulti}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}
	_, _, _, _ = runMeRolesCommand(t, seam, "--base-url", "https://flag.test/api/v5")
	if seam.assembledBaseURL != "https://flag.test/api/v5" {
		t.Errorf("assemble received base URL %q, want the inherited flag value", seam.assembledBaseURL)
	}
}

// The `roles` leaf declares no --base-url flag of its own — it inherits the
// root's persistent one. A locally-declared flag would shadow the inherited
// value; this pins that it does not.
func TestMeRolesCommand_DeclaresNoOwnBaseURLFlag(t *testing.T) {
	cmd := newMeRolesCommand(&fakeMeSeam{})
	if cmd.Flags().Lookup(apiclient.FlagBaseURL) != nil {
		t.Errorf("the roles leaf must not declare its own --%s flag; it is inherited", apiclient.FlagBaseURL)
	}
}

// The registration guard permits `me` to be BOTH runnable (its own RunE) AND a
// parent (the roles child), because the guard validates a command at its own
// registration: `me` registers as a leaf first (childless), then `roles`
// attaches under it. This pins ADR-1's runnable-with-children resolution.
func TestMeRolesCommand_MeIsRunnableWithChildren(t *testing.T) {
	root := NewRootCommand()
	meCmd := newMeCommand(&fakeMeSeam{})
	MustRegister(root, meCmd) // me as a runnable leaf — must not panic
	MustRegister(meCmd, newMeRolesCommand(&fakeMeSeam{}))

	if meCmd.RunE == nil {
		t.Error("me should remain runnable (its own RunE) after gaining a child")
	}
	var rolesChild *cobra.Command
	for _, c := range meCmd.Commands() {
		if c.Name() == "roles" {
			rolesChild = c
		}
	}
	if rolesChild == nil {
		t.Fatal("me should have a `roles` child registered under it")
	}
	if rolesChild.RunE == nil {
		t.Error("the roles child should carry an action")
	}
}

// The full Assemble wiring must not panic — pins that the production wiring of
// `me` + `me roles` passes the registration guard end to end.
func TestAssemble_WiresMeRolesWithoutPanic(t *testing.T) {
	root := Assemble()
	me, _, err := root.Find([]string{"me"})
	if err != nil || me == nil || me.Name() != "me" {
		t.Fatalf("Assemble should wire a `me` command, got %v (err %v)", me, err)
	}
	rolesCmd, _, err := root.Find([]string{"me", "roles"})
	if err != nil || rolesCmd == nil || rolesCmd.Name() != "roles" {
		t.Fatalf("Assemble should wire `me roles`, got %v (err %v)", rolesCmd, err)
	}
}
