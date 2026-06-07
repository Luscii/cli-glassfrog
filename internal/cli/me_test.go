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

	"github.com/Luscii/cli-glassfrog/internal/apiclient"
	"github.com/Luscii/cli-glassfrog/internal/auth"
	"github.com/Luscii/cli-glassfrog/internal/glassfrog"
)

const meSecretToken = "gf_live_secret123"

// --- fakes ---------------------------------------------------------------

// cannedTransport is a fake base http.RoundTripper returning a canned response
// (or a wire error), recording its call count and the request URL/query so a
// test can assert one-attempt sends and the include=roles parameter. As a base
// transport it sits under 007's AuthTransport, so it is reached only on the
// authenticated branch — a no-credential context never calls it.
type cannedTransport struct {
	calls     int
	lastQuery url.Values
	status    int
	body      string
	netErr    error
}

func (c *cannedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	c.calls++
	c.lastQuery = req.URL.Query()
	if c.netErr != nil {
		return nil, c.netErr
	}
	return &http.Response{
		StatusCode: c.status,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(c.body)),
	}, nil
}

// fakeMeSeam binds a fixed ConnectionContext and a fake base transport, so every
// runMe branch runs offline (ADR-5). newClientErr stands in for a base-URL error
// surfaced at client construction.
type fakeMeSeam struct {
	ctx          apiclient.ConnectionContext
	newClientErr error
	transport    http.RoundTripper

	assembledBaseURL string
	assembleCalled   bool
	newClientCalled  bool
}

func (s *fakeMeSeam) assemble(baseURL string) apiclient.ConnectionContext {
	s.assembleCalled = true
	s.assembledBaseURL = baseURL
	return s.ctx
}

func (s *fakeMeSeam) newClient(ctx apiclient.ConnectionContext) (*apiclient.Client, error) {
	s.newClientCalled = true
	if s.newClientErr != nil {
		return nil, s.newClientErr
	}
	return apiclient.NewClient(ctx, s.transport)
}

// validMeContext is a complete, usable context: a parseable base URL and a
// present token, so NewClient succeeds and the AuthTransport authenticates the
// fake send. The token is the secret a token-never-in-output assertion guards.
func validMeContext() apiclient.ConnectionContext {
	return apiclient.ConnectionContext{
		BaseURL: apiclient.BaseURL{Value: "https://example.test/api/v5", Source: apiclient.SourceFlag},
		Cred:    auth.Resolution{Token: meSecretToken, Source: auth.SourceEnvironment},
	}
}

const meBodyAlice = `{
  "actor": {"id": "per_0123456789abcdef0123456789abcdef", "name": "Alice Smith", "kind": "human"},
  "organization": {"id": "org_0123456789abcdef0123456789abcdef", "name": "Acme"},
  "membership": {"access_level": "admin"}
}`

const meBodyAgentWithRoles = `{
  "actor": {"id": "agt_0123456789abcdef0123456789abcdef", "name": "Claude", "kind": "agent"},
  "organization": {"id": "org_0123456789abcdef0123456789abcdef", "name": "Acme"},
  "membership": {"access_level": "normal"},
  "roles": [
    {"id": "role_0123456789abcdef0123456789abcdef", "name": "Marketing Lead"},
    {"id": "role_00000000000000000000000000000001", "name": "Treasurer"}
  ]
}`

// runMeOver drives the pure runMe over a fake seam, returning the outcome and the
// captured stdout/stderr (in-memory buffers — no os.Pipe, no real streams).
func runMeOver(t *testing.T, seam meSeam, include ...string) (Outcome, string, string) {
	t.Helper()
	var out, errb bytes.Buffer
	outcome, _ := runMe(meConfig{
		seam:    seam,
		include: include,
		reqCtx:  context.Background(),
		stdout:  &out,
		stderr:  &errb,
	})
	combined := out.String() + errb.String()
	if strings.Contains(combined, meSecretToken) {
		t.Fatalf("the token leaked into output: %q", combined)
	}
	return outcome, out.String(), errb.String()
}

// --- formatMe (pure) -----------------------------------------------------

func TestFormatMe_IdentityOnly(t *testing.T) {
	me := glassfrog.MeResponse{
		Actor:        glassfrog.Actor{ID: "per_abc", Name: "Alice Smith", Kind: "human"},
		Organization: glassfrog.Organization{ID: "org_abc", Name: "Acme"},
		Membership:   glassfrog.Membership{AccessLevel: "admin"},
	}
	out := formatMe(me, false)
	for _, want := range []string{"Alice Smith", "(human)", "per_abc", "Acme", "org_abc", "admin"} {
		if !strings.Contains(out, want) {
			t.Errorf("projection missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "roles:") {
		t.Errorf("identity-only projection should not have a roles section:\n%s", out)
	}
}

func TestFormatMe_WithRoles(t *testing.T) {
	me := glassfrog.MeResponse{
		Actor:        glassfrog.Actor{ID: "agt_abc", Name: "Claude", Kind: "agent"},
		Organization: glassfrog.Organization{ID: "org_abc", Name: "Acme"},
		Membership:   glassfrog.Membership{AccessLevel: "normal"},
		Roles: []glassfrog.Role{
			{ID: "role_1", Name: "Marketing Lead"},
			{ID: "role_2", Name: "Treasurer"},
		},
	}
	out := formatMe(me, true)
	for _, want := range []string{"(agent)", "agt_abc", "roles:", "Marketing Lead", "role_1", "Treasurer", "role_2"} {
		if !strings.Contains(out, want) {
			t.Errorf("roles projection missing %q:\n%s", want, out)
		}
	}
}

// A roles embed requested but with no roles omits the section rather than
// printing an empty list (interface-cli).
func TestFormatMe_EmptyRolesEmbedOmitsSection(t *testing.T) {
	me := glassfrog.MeResponse{
		Actor:        glassfrog.Actor{ID: "per_abc", Name: "Alice", Kind: "human"},
		Organization: glassfrog.Organization{ID: "org_abc", Name: "Acme"},
		Membership:   glassfrog.Membership{AccessLevel: "admin"},
	}
	out := formatMe(me, true) // requested, but Roles is empty
	if strings.Contains(out, "roles:") {
		t.Errorf("an empty roles embed should omit the section:\n%s", out)
	}
}

// --- validateInclude (pure) ----------------------------------------------

func TestValidateInclude(t *testing.T) {
	if err := validateInclude(nil); err != nil {
		t.Errorf("absent --include should be valid, got %v", err)
	}
	if err := validateInclude([]string{"roles"}); err != nil {
		t.Errorf("--include roles should be valid, got %v", err)
	}
	err := validateInclude([]string{"actions"})
	if err == nil {
		t.Fatal("an unsupported --include target should be rejected")
	}
	if !strings.Contains(err.Error(), "actions") {
		t.Errorf("the usage error should name the unsupported target, got %q", err.Error())
	}
}

// --- runMe branches ------------------------------------------------------

func TestRunMe_SuccessIdentity(t *testing.T) {
	tr := &cannedTransport{status: 200, body: meBodyAlice}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, _ := runMeOver(t, seam)
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success", outcome)
	}
	for _, want := range []string{"Alice Smith", "(human)", "per_0123456789abcdef0123456789abcdef", "Acme", "admin"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("stdout missing %q:\n%s", want, stdout)
		}
	}
	if tr.calls != 1 {
		t.Errorf("transport called %d times, want exactly 1", tr.calls)
	}
	if tr.lastQuery.Get("include") != "" {
		t.Errorf("no --include should add no include query, got %q", tr.lastQuery.Get("include"))
	}
}

func TestRunMe_AgentTokenReportedAsAgentWithRolesEmbed(t *testing.T) {
	tr := &cannedTransport{status: 200, body: meBodyAgentWithRoles}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, _ := runMeOver(t, seam, "roles")
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success", outcome)
	}
	if tr.lastQuery.Get("include") != "roles" {
		t.Errorf("--include roles should add include=roles, got query %v", tr.lastQuery)
	}
	for _, want := range []string{"(agent)", "agt_0123456789abcdef0123456789abcdef", "roles:", "Marketing Lead", "Treasurer"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("stdout missing %q:\n%s", want, stdout)
		}
	}
}

func TestRunMe_EmptyRolesEmbedOmitsSection(t *testing.T) {
	tr := &cannedTransport{status: 200, body: meBodyAlice} // no roles in body
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, _ := runMeOver(t, seam, "roles")
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success", outcome)
	}
	if strings.Contains(stdout, "roles:") {
		t.Errorf("an empty roles embed should omit the section:\n%s", stdout)
	}
}

// An unsupported --include is rejected before any request: the tripwire base is
// never reached, and assembly never happens (validateInclude runs first).
func TestRunMe_UnsupportedIncludeRejectedBeforeAnyRequest(t *testing.T) {
	tr := &cannedTransport{status: 200, body: meBodyAlice}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, _, stderr := runMeOver(t, seam, "actions")
	if outcome != UsageError {
		t.Fatalf("outcome = %v, want UsageError", outcome)
	}
	if !strings.Contains(stderr, "actions") {
		t.Errorf("stderr should name the unsupported target, got %q", stderr)
	}
	if tr.calls != 0 {
		t.Errorf("transport was called %d times; an unsupported include must issue no request", tr.calls)
	}
	if seam.assembleCalled || seam.newClientCalled {
		t.Errorf("validateInclude must run before assembly/build (assembled=%v built=%v)", seam.assembleCalled, seam.newClientCalled)
	}
}

func TestRunMe_NonStatus2xxIsAPIError(t *testing.T) {
	tr := &cannedTransport{status: 401, body: `{"error":"unauthorized"}`}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, stderr := runMeOver(t, seam)
	if outcome != APIError {
		t.Fatalf("outcome = %v, want APIError", outcome)
	}
	if strings.TrimSpace(stdout) != "" {
		t.Errorf("no projection should print on a non-2xx, got stdout %q", stdout)
	}
	if !strings.Contains(stderr, "401") {
		t.Errorf("stderr should surface the status code, got %q", stderr)
	}
}

func TestRunMe_TransportFailureIsNetworkUnavailableNoRetry(t *testing.T) {
	tr := &cannedTransport{netErr: errors.New("dial tcp: connection refused")}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, stderr := runMeOver(t, seam)
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
		t.Errorf("transport called %d times; the read must not retry", tr.calls)
	}
}

// A no-token context surfaces *AuthError{NoCredentials} from the AuthTransport
// fail-safe → UsageError, with an actionable "auth login" message, and no
// request ever reaches the base transport.
func TestRunMe_NoCredentialsIsUsageErrorNoRequest(t *testing.T) {
	tr := &cannedTransport{status: 200, body: meBodyAlice}
	ctx := apiclient.ConnectionContext{
		BaseURL: apiclient.BaseURL{Value: "https://example.test/api/v5", Source: apiclient.SourceFlag},
		Cred:    auth.Resolution{Source: auth.SourceNone},
	}
	seam := &fakeMeSeam{ctx: ctx, transport: tr}

	outcome, _, stderr := runMeOver(t, seam)
	if outcome != UsageError {
		t.Fatalf("outcome = %v, want UsageError", outcome)
	}
	if !strings.Contains(stderr, "auth login") {
		t.Errorf("stderr should point at `glassfrog auth login`, got %q", stderr)
	}
	if tr.calls != 0 {
		t.Errorf("transport was called %d times; an unauthenticated request must not be sent", tr.calls)
	}
}

func TestRunMe_CredentialErrorIsRuntimeError(t *testing.T) {
	tr := &cannedTransport{status: 200, body: meBodyAlice}
	ctx := apiclient.ConnectionContext{
		BaseURL: apiclient.BaseURL{Value: "https://example.test/api/v5", Source: apiclient.SourceFlag},
		Cred:    auth.Resolution{Source: auth.SourceNone},
		CredErr: errors.New("the credentials file is malformed"),
	}
	seam := &fakeMeSeam{ctx: ctx, transport: tr}

	outcome, _, stderr := runMeOver(t, seam)
	if outcome != RuntimeError {
		t.Fatalf("outcome = %v, want RuntimeError", outcome)
	}
	if strings.TrimSpace(stderr) == "" {
		t.Error("a credential-file error should be reported on stderr")
	}
	if tr.calls != 0 {
		t.Errorf("transport was called %d times; a credential error must not send", tr.calls)
	}
}

func TestRunMe_UndecodableBodyIsRuntimeError(t *testing.T) {
	tr := &cannedTransport{status: 200, body: `not json at all`}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, stderr := runMeOver(t, seam)
	if outcome != RuntimeError {
		t.Fatalf("outcome = %v, want RuntimeError", outcome)
	}
	if strings.TrimSpace(stdout) != "" {
		t.Errorf("no projection should print on a decode failure, got %q", stdout)
	}
	if strings.TrimSpace(stderr) == "" {
		t.Error("a decode failure should be reported on stderr")
	}
}

func TestRunMe_BaseURLErrorIsUsageError(t *testing.T) {
	tr := &cannedTransport{status: 200, body: meBodyAlice}
	seam := &fakeMeSeam{
		ctx:          apiclient.ConnectionContext{},
		newClientErr: &apiclient.BaseURLError{Source: "--base-url"},
		transport:    tr,
	}

	outcome, _, stderr := runMeOver(t, seam)
	if outcome != UsageError {
		t.Fatalf("outcome = %v, want UsageError", outcome)
	}
	if !strings.Contains(stderr, "base-url") && !strings.Contains(stderr, "base URL") {
		t.Errorf("stderr should name the base-URL problem and next step, got %q", stderr)
	}
	if tr.calls != 0 {
		t.Errorf("transport was called %d times; a base-URL error must not send", tr.calls)
	}
}

// --- newMeCommand integration (outcome → exit code) ----------------------

// runMeCommand registers newMeCommand under a real root (with the persistent
// --base-url flag) and dispatches through Run, returning the outcome, the mapped
// process exit code, and the captured streams. This pins the command wiring AND
// the outcomeError → ExitCode path (3/6) the dispatch carrier enables.
func runMeCommand(t *testing.T, seam meSeam, args ...string) (Outcome, int, string, string) {
	t.Helper()
	root := NewRootCommand()
	MustRegister(root, newMeCommand(seam))
	var out, errb bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&errb)
	outcome, _ := Run(root, append([]string{"me"}, args...))
	return outcome, ExitCode(outcome), out.String(), errb.String()
}

func TestMeCommand_ExitCodesAcrossOutcomes(t *testing.T) {
	cases := []struct {
		name     string
		tr       *cannedTransport
		ctx      apiclient.ConnectionContext
		seamErr  error
		args     []string
		outcome  Outcome
		exitCode int
	}{
		{"success", &cannedTransport{status: 200, body: meBodyAlice}, validMeContext(), nil, nil, Success, 0},
		{"api-error", &cannedTransport{status: 404, body: `{}`}, validMeContext(), nil, nil, APIError, 3},
		{"network-unavailable", &cannedTransport{netErr: errors.New("connection refused")}, validMeContext(), nil, nil, NetworkUnavailable, 6},
		{"usage-unsupported-include", &cannedTransport{status: 200, body: meBodyAlice}, validMeContext(), nil, []string{"--include", "actions"}, UsageError, 2},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			seam := &fakeMeSeam{ctx: tc.ctx, newClientErr: tc.seamErr, transport: tc.tr}
			outcome, code, stdout, stderr := runMeCommand(t, seam, tc.args...)
			if outcome != tc.outcome {
				t.Errorf("outcome = %v, want %v", outcome, tc.outcome)
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

// An unexpected positional argument is a usage error (cobra.NoArgs); nothing runs.
func TestMeCommand_RejectsPositionalArgs(t *testing.T) {
	tr := &cannedTransport{status: 200, body: meBodyAlice}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}
	outcome, code, _, _ := runMeCommand(t, seam, "extra")
	if outcome != UsageError || code != 2 {
		t.Fatalf("positional arg: outcome=%v code=%d, want UsageError/2", outcome, code)
	}
	if tr.calls != 0 {
		t.Errorf("a rejected invocation must not send a request, got %d calls", tr.calls)
	}
}

// The persistent --base-url value reaches the seam's assemble (ADR-2 inheritance).
func TestMeCommand_PassesBaseURLFlagToAssemble(t *testing.T) {
	tr := &cannedTransport{status: 200, body: meBodyAlice}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}
	_, _, _, _ = runMeCommand(t, seam, "--base-url", "https://flag.test/api/v5")
	if seam.assembledBaseURL != "https://flag.test/api/v5" {
		t.Errorf("assemble received base URL %q, want the flag value", seam.assembledBaseURL)
	}
}
