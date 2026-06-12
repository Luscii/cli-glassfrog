package cli

import (
	"bytes"
	"context"
	"errors"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Luscii/cli-glassfrog/internal/apiclient"
	"github.com/Luscii/cli-glassfrog/internal/auth"
	"github.com/Luscii/cli-glassfrog/internal/output"
	"github.com/Luscii/cli-glassfrog/internal/rcfile"
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
	lastPath  string
	status    int
	body      string
	netErr    error
}

func (c *cannedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	c.calls++
	c.lastQuery = req.URL.Query()
	c.lastPath = req.URL.Path
	if c.netErr != nil {
		return nil, c.netErr
	}
	return &http.Response{
		StatusCode: c.status,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(c.body)),
	}, nil
}

// seqMeResp is one canned reply a seqMeTransport returns on a single attempt.
type seqMeResp struct {
	status int
	header http.Header
	body   string
}

// seqMeTransport is a fake base transport that returns canned replies in order
// (the i-th attempt gets steps[i], repeating the last once exhausted), so a 429
// retry through runMe can be exercised end-to-end over a fake — no real network.
type seqMeTransport struct {
	calls    int
	lastPath string
	steps    []seqMeResp
}

func (s *seqMeTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	s.calls++
	s.lastPath = req.URL.Path
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

// fakeMeSeam binds a fixed ConnectionContext and a fake base transport, so every
// runMe branch runs offline (ADR-5). newClientErr stands in for a base-URL error
// surfaced at client construction.
type fakeMeSeam struct {
	ctx          apiclient.ConnectionContext
	newClientErr error
	transport    http.RoundTripper

	assembledBaseURL    string
	assembledBaseURLSet bool // the presence bit (cobra Changed()) the RunE threaded to assemble (040 ADR-2)
	assembleCalled      bool
	newClientCalled     bool

	slept []time.Duration // the 017 backoff waits the recording fake-sleep observed

	// Output Format Selection (020) injected sources for resolveFormat: the env and
	// .glassfrogrc output rungs the fake feeds to output.ResolveFormat alongside the
	// flag value. All zero by default — env absent, file absent, no error — so a
	// default-constructed fake resolves to FormatFull (the standing full path every
	// pre-020 test exercises). A 020 test sets these to drive json/compact/yaml or a
	// resolution error.
	envOutput  string
	fileOutput string
	filePath   string
	fileFound  bool
	fileErr    error

	// User-Defined Template Output (035) injected template-source behavior for
	// readTemplateSource, exercised over the real readTemplateSourceFrom logic with
	// NO real filesystem/stdin: tmplFiles maps a TemplateFile path to its content (a
	// path absent from the map reads as not-found, the missing-file fail-fast case);
	// tmplStdin is the piped stdin text and tmplStdinPiped says whether a pipe is
	// present (isTTY = !tmplStdinPiped — an un-piped stdin is the TTY fail-fast case).
	// All zero by default — a built-in-format test never touches them.
	tmplFiles      map[string]string
	tmplStdin      string
	tmplStdinPiped bool
}

func (s *fakeMeSeam) assemble(baseURL string, baseURLPresent bool) apiclient.ConnectionContext {
	s.assembleCalled = true
	s.assembledBaseURL = baseURL
	s.assembledBaseURLSet = baseURLPresent
	return s.ctx
}

func (s *fakeMeSeam) newClient(ctx apiclient.ConnectionContext) (*apiclient.Client, error) {
	s.newClientCalled = true
	if s.newClientErr != nil {
		return nil, s.newClientErr
	}
	return apiclient.NewClient(ctx, s.transport)
}

// sleep binds a recording fake-sleep that never blocks, so a 429 retry through
// runMe is asserted in milliseconds (CONSTITUTION IV — no real sleep).
func (s *fakeMeSeam) sleep() func(time.Duration) {
	return func(d time.Duration) { s.slept = append(s.slept, d) }
}

// resolveSelection drives the real output.ResolveSelectionFromOS composing entry
// (020 widened by 035, retrofit 040) so a test exercises genuine parsing, precedence,
// and the flag-rung template classification. The 040 retrofit folded the 6-arg
// pre-fetched-source core into ResolveSelectionFromOS and removed it, so the fake now
// feeds its injected sources through the OS-shaped seam HERMETICALLY: the env rung via
// the real GLASSFROG_OUTPUT (set/unset for this call, then restored) and the file rung
// via a temp .glassfrogrc seeded from the injected file source — never the developer's
// real environment or ~/.glassfrogrc. A default-constructed fake (all sources absent)
// yields FormatFull — the standing full path every pre-020 test relies on.
func (s *fakeMeSeam) resolveSelection(flagValue string, flagPresent bool) (output.Selection, error) {
	dir, err := os.MkdirTemp("", "cli-sel-fake-")
	if err != nil {
		return output.Selection{Format: output.DefaultFormat}, err
	}
	defer os.RemoveAll(dir)

	// Env rung: set the injected value (or unset, so an absent injection falls through
	// regardless of the developer's ambient environment). Restored on return.
	prev, had := os.LookupEnv(output.EnvVarOutput)
	if strings.TrimSpace(s.envOutput) != "" {
		os.Setenv(output.EnvVarOutput, s.envOutput)
	} else {
		os.Unsetenv(output.EnvVarOutput)
	}
	defer func() {
		if had {
			os.Setenv(output.EnvVarOutput, prev)
		} else {
			os.Unsetenv(output.EnvVarOutput)
		}
	}()

	// File rung: translate the injected file source into a real temp .glassfrogrc.
	// An injected read error is modelled by a directory at the file path, which makes
	// rcfile fail loud with a real *ReadError naming that .glassfrogrc (no fall-through).
	rcPath := filepath.Join(dir, rcfile.FileName)
	switch {
	case s.fileErr != nil:
		_ = os.Mkdir(rcPath, 0o755)
	case s.fileFound:
		_ = os.WriteFile(rcPath, []byte("output="+s.fileOutput+"\n"), 0o600)
	}

	return output.ResolveSelectionFromOS(flagValue, flagPresent, dir, dir)
}

// readTemplateSource exercises the production readTemplateSourceFrom logic (035
// ADR-4) over injected sources: a file read resolves against tmplFiles (a path not
// present reads as not-found — the missing-file fail-fast case), and stdin reads
// tmplStdin guarded by !tmplStdinPiped (an un-piped stdin is the TTY fail-fast case).
// No real filesystem or os.Stdin is touched, so every fail-fast case is hermetic.
func (s *fakeMeSeam) readTemplateSource(ref output.TemplateRef) (string, error) {
	readFile := func(path string) ([]byte, error) {
		if content, ok := s.tmplFiles[path]; ok {
			return []byte(content), nil
		}
		return nil, fs.ErrNotExist
	}
	return readTemplateSourceFrom(ref, readFile, !s.tmplStdinPiped, strings.NewReader(s.tmplStdin))
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

// The `me` full projection is now rendered through internal/render (019); its
// byte-equivalence with the pre-019 formatMe output is pinned by that package's
// goldens (TestRender_MeFull_*). The end-to-end success path — including the
// roles embed and the empty-embed omission — stays covered by the runMe tests
// below and the identity-read BDD suite.

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
	if !strings.Contains(err.Error(), `"actions"`) {
		t.Errorf("the usage error should quote the unsupported target, got %q", err.Error())
	}
	if !strings.Contains(err.Error(), "target ") || strings.Contains(err.Error(), "targets ") {
		t.Errorf("a single unsupported target should use the singular noun, got %q", err.Error())
	}

	// --include is a string slice: multiple unsupported targets must each be
	// quoted individually with a plural noun — not the whole list quoted as one
	// bogus target.
	multi := validateInclude([]string{"projects", "actions"})
	if multi == nil {
		t.Fatal("multiple unsupported targets should be rejected")
	}
	msg := multi.Error()
	if !strings.Contains(msg, "targets ") {
		t.Errorf("multiple unsupported targets should use the plural noun, got %q", msg)
	}
	if !strings.Contains(msg, `"actions"`) || !strings.Contains(msg, `"projects"`) {
		t.Errorf("each unsupported target should be quoted individually, got %q", msg)
	}
	if strings.Contains(msg, `"projects, actions"`) || strings.Contains(msg, `"actions, projects"`) {
		t.Errorf("the comma-joined list should not be quoted as a single target, got %q", msg)
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

// A me run whose transport returns 429-then-200 rides out the throttle through
// 017's RetryExecutor: it renders the identity projection after one bounded wait
// (recording fake-sleep — no real delay), exits Success, sends exactly twice, and
// emits a secret-free progress note to stderr (T003 wiring).
func TestRunMe_RetriesOn429ThenSucceeds(t *testing.T) {
	tr := &seqMeTransport{steps: []seqMeResp{
		{status: 429, header: http.Header{"Retry-After": {"2"}}, body: `{"error":"rate limited"}`},
		{status: 200, body: meBodyAlice},
	}}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, stderr := runMeOver(t, seam)
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success after a bounded retry", outcome)
	}
	if !strings.Contains(stdout, "Alice Smith") {
		t.Errorf("the identity projection should render after the retry, got %q", stdout)
	}
	if tr.calls != 2 {
		t.Errorf("transport called %d times, want 2 (429 then 200)", tr.calls)
	}
	if !strings.HasSuffix(tr.lastPath, "/me") {
		t.Errorf("request path = %q, want it to target the /me endpoint", tr.lastPath)
	}
	if len(seam.slept) != 1 || seam.slept[0] != 2*time.Second {
		t.Errorf("waited %v, want exactly [2s] (the Retry-After interval, via the injected fake)", seam.slept)
	}
	if !strings.Contains(stderr, "rate limited") {
		t.Errorf("a progress note should be written to stderr before the retry, got %q", stderr)
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
	// A genuinely generic non-2xx (500) — 401/403/429 now split into
	// PermissionError/RateLimited (API Error Extraction 015), so a 5xx is the
	// faithful representative of the residual generic APIError bucket.
	tr := &cannedTransport{status: 500, body: `{"error":"server error"}`}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, stderr := runMeOver(t, seam)
	if outcome != APIError {
		t.Fatalf("outcome = %v, want APIError", outcome)
	}
	if strings.TrimSpace(stdout) != "" {
		t.Errorf("no projection should print on a non-2xx, got stdout %q", stdout)
	}
	if !strings.Contains(stderr, "500") {
		t.Errorf("stderr should surface the status code, got %q", stderr)
	}
	// Action Transparency: the message names the cause (status) AND a concrete,
	// generic next step — without interpreting the status (no per-status meaning).
	if !strings.Contains(stderr, "retry") {
		t.Errorf("the non-2xx message should carry a generic next step, got %q", stderr)
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

// 031 ADR-2: an undecodable 2xx body is an API-exchange problem (APIError → exit
// 3), not a CLI-internal fault (was RuntimeError → exit 1). The cause/next-step
// wording is unchanged.
func TestRunMe_UndecodableBodyIsAPIError(t *testing.T) {
	tr := &cannedTransport{status: 200, body: `not json at all`}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, stderr := runMeOver(t, seam)
	if outcome != APIError {
		t.Fatalf("outcome = %v, want APIError", outcome)
	}
	if strings.TrimSpace(stdout) != "" {
		t.Errorf("no projection should print on a decode failure, got %q", stdout)
	}
	if strings.TrimSpace(stderr) == "" {
		t.Error("a decode failure should be reported on stderr")
	}
	// Action Transparency: the message names the cause (shape mismatch) AND the
	// next step (report it).
	if !strings.Contains(stderr, "report it") {
		t.Errorf("the decode-failure message should carry a next step, got %q", stderr)
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

// A base-URL error surfaced as an rcfile read/format error (an unreadable or
// malformed .glassfrogrc base_url) is classified UsageError AND must carry the
// same next-step correction hint as a *BaseURLError — the two are symmetric in
// classifyClientError, so formatClientErrorMessage must give both the hint
// rather than printing a bare err.Error(). Pins both rcfile shapes.
func TestRunMe_BaseURLRcfileErrorIsUsageErrorWithNextStep(t *testing.T) {
	for name, rcErr := range map[string]error{
		"format": &rcfile.FormatError{Path: "/home/u/.glassfrogrc"},
		"read":   &rcfile.ReadError{Path: "/home/u/.glassfrogrc", Err: errors.New("permission denied")},
	} {
		t.Run(name, func(t *testing.T) {
			tr := &cannedTransport{status: 200, body: meBodyAlice}
			seam := &fakeMeSeam{ctx: apiclient.ConnectionContext{}, newClientErr: rcErr, transport: tr}

			outcome, _, stderr := runMeOver(t, seam)
			if outcome != UsageError {
				t.Fatalf("outcome = %v, want UsageError", outcome)
			}
			if !strings.Contains(stderr, ".glassfrogrc") {
				t.Errorf("stderr should name the configured file, got %q", stderr)
			}
			if !strings.Contains(stderr, "--base-url") || !strings.Contains(stderr, "GLASSFROG_BASE_URL") {
				t.Errorf("stderr should carry the base-URL correction next step, got %q", stderr)
			}
			if tr.calls != 0 {
				t.Errorf("transport was called %d times; a base-URL error must not send", tr.calls)
			}
		})
	}
}

// --- API Error Extraction (015): refined-error message + classification ---

// reportFailure refines a generic non-2xx *ResponseError into a typed
// *ProblemError (once), so the message surfaces the API's own detail and the
// returned error that travels up the chain IS the typed value (ADR-4). Pins (on
// the human path, unchanged by 032): the API detail appears
// (DetailSynthesized==false); a synthesized fallback shows the "status N"
// wording, NOT the synthesized text; the per-class next-step hints render on
// stderr; stdout stays empty; the returned error is a *ProblemError.
func TestReportFailure_SurfacesDetailAndClassifies(t *testing.T) {
	cases := []struct {
		name        string
		re          *apiclient.ResponseError
		wantOutcome Outcome
		wantInMsg   []string
		notInMsg    []string
	}{
		{
			name:        "404-detail-surfaced",
			re:          &apiclient.ResponseError{StatusCode: 404, Body: []byte(`{"detail":"Token lacks access to this circle"}`)},
			wantOutcome: APIError,
			wantInMsg:   []string{"404", "Token lacks access to this circle"},
		},
		{
			// 031: the 401 next step is split from the old combined permission hint —
			// it points at verifying the configured API token (authentication).
			name:        "401-permission-hint",
			re:          &apiclient.ResponseError{StatusCode: 401, Body: []byte(`{"detail":"Unauthorized"}`)},
			wantOutcome: PermissionError,
			wantInMsg:   []string{"401", "Unauthorized", "verify the configured API token"},
		},
		{
			// 031: the 403 next step points at the identity's role membership /
			// permission (authorization), distinct from the 401 token hint.
			name:        "403-permission-hint",
			re:          &apiclient.ResponseError{StatusCode: 403, Body: []byte(`{"detail":"Forbidden"}`)},
			wantOutcome: PermissionError,
			wantInMsg:   []string{"403", "required role membership / permission"},
		},
		{
			// 031: the 429 next step references the reset window (the Retry-After /
			// X-RateLimit-Reset headers), refining the old bare "retry later".
			name:        "429-rate-limit-hint",
			re:          &apiclient.ResponseError{StatusCode: 429, Body: []byte(`{"detail":"Too Many Requests"}`)},
			wantOutcome: RateLimited,
			wantInMsg:   []string{"429", "wait for the rate-limit window to reset", "retry"},
		},
		{
			name:        "synthesized-detail-shows-fallback-not-synthesized-text",
			re:          &apiclient.ResponseError{StatusCode: 500, Body: []byte(`<html>boom</html>`)},
			wantOutcome: APIError,
			wantInMsg:   []string{"the API returned a non-2xx response: status 500", "retry"},
			// the status-derived fallback "Internal Server Error" must NOT be
			// presented as the API's own detail wording.
			notInMsg: []string{"Internal Server Error"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var outb, errb bytes.Buffer
			// The human path (full) is unchanged by 032: the cause-plus-next-step
			// line lands on stderr and stdout stays empty.
			outcome, retErr := reportFailure(&outb, &errb, output.FormatFull, tc.re)
			if outb.Len() != 0 {
				t.Errorf("the human path must leave stdout empty, got %q", outb.String())
			}
			if outcome != tc.wantOutcome {
				t.Errorf("outcome = %v, want %v", outcome, tc.wantOutcome)
			}
			// The returned error must be the refined *ProblemError (travels up the chain).
			var pe *apiclient.ProblemError
			if !errors.As(retErr, &pe) {
				t.Errorf("returned error should be a *ProblemError, got %T", retErr)
			}
			msg := errb.String()
			for _, want := range tc.wantInMsg {
				if !strings.Contains(msg, want) {
					t.Errorf("message should contain %q:\n%s", want, msg)
				}
			}
			for _, notWant := range tc.notInMsg {
				if strings.Contains(msg, notWant) {
					t.Errorf("message should NOT contain %q:\n%s", notWant, msg)
				}
			}
			if strings.Contains(msg, meSecretToken) {
				t.Errorf("token leaked into the message: %q", msg)
			}
		})
	}
}

// reportFailure must refine ONCE: feeding it an already-refined
// *ProblemError must not double-wrap, and the outcome/message stay stable.
func TestReportFailure_RefinesOnce(t *testing.T) {
	pe := apiclient.ExtractProblem(&apiclient.ResponseError{StatusCode: 403, Body: []byte(`{"detail":"Forbidden"}`)})
	var outb, errb bytes.Buffer
	outcome, retErr := reportFailure(&outb, &errb, output.FormatFull, pe)
	if outcome != PermissionError {
		t.Errorf("outcome = %v, want PermissionError", outcome)
	}
	// The returned value should still be the SAME *ProblemError (no re-wrap).
	if retErr != error(pe) {
		t.Errorf("an already-refined *ProblemError must not be re-refined; got a different value")
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

// TestMeCommand_ThreadsBaseURLPresence pins the 040 presence threading (ADR-2):
// the RunE forwards cobra Changed() for --base-url to the seam's assemble, and
// Changed() reports the inherited persistent root flag as supplied regardless of
// whether it sits before or after the `me` subcommand — including an explicit
// empty value (`--base-url ""`). An unsupplied flag threads presence=false. This
// is the cli-side companion to the resolver-level @base-url scenarios (the
// command-path-position scenario in particular), exercising the real cobra
// parse that those resolver tests model.
func TestMeCommand_ThreadsBaseURLPresence(t *testing.T) {
	run := func(args ...string) *fakeMeSeam {
		t.Helper()
		seam := &fakeMeSeam{ctx: validMeContext(), transport: &cannedTransport{status: 200, body: meBodyAlice}}
		root := NewRootCommand()
		MustRegister(root, newMeCommand(seam))
		var out, errb bytes.Buffer
		root.SetOut(&out)
		root.SetErr(&errb)
		_, _ = Run(root, args)
		return seam
	}

	t.Run("flag before subcommand (empty value, supplied)", func(t *testing.T) {
		seam := run("--base-url", "", "me")
		if !seam.assembledBaseURLSet {
			t.Errorf("presence = false, want true (--base-url \"\" before `me` is supplied)")
		}
		if seam.assembledBaseURL != "" {
			t.Errorf("base URL = %q, want the empty supplied value", seam.assembledBaseURL)
		}
	})

	t.Run("flag after subcommand (empty value, supplied)", func(t *testing.T) {
		seam := run("me", "--base-url", "")
		if !seam.assembledBaseURLSet {
			t.Errorf("presence = false, want true (--base-url \"\" after `me` is supplied)")
		}
	})

	t.Run("flag with explicit =empty (supplied)", func(t *testing.T) {
		seam := run("me", "--base-url=")
		if !seam.assembledBaseURLSet {
			t.Errorf("presence = false, want true (--base-url= is supplied)")
		}
	})

	t.Run("flag unsupplied", func(t *testing.T) {
		seam := run("me")
		if seam.assembledBaseURLSet {
			t.Errorf("presence = true, want false (no --base-url supplied)")
		}
	})
}
