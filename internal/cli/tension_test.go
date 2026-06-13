package cli

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

// tensionCreatedBody is a representative createTension 201 body: the single-object
// {data: Tension} envelope carrying the ten_ id, the server-computed status, the
// sensing role/person, and null nullable fields. The secret token appears nowhere.
const tensionCreatedBody = `{"data":{"id":"ten_0123","type":"tension","body":"We ship faster than we update the roadmap.","status":"unprocessed","role_id":"role_0123","sensed_by_id":"per_0123","created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z","label":null,"meeting_type":null,"parent_role_id":null}}`

// tensionTransport is a fake base http.RoundTripper for the write path: like
// cannedTransport it returns a canned response (or a wire error) and counts calls,
// but it also records the request METHOD, BODY, and Content-Type header so a test
// can assert the POST shape (the marshalled {tension:{…}} envelope and the
// application/json content type the 042 ADR-1 seam sets). As a base transport it
// sits under 007's AuthTransport, so it is reached only on the authenticated branch.
type tensionTransport struct {
	calls           int
	lastMethod      string
	lastPath        string
	lastBody        string
	lastContentType string
	lastIfMatch     string
	status          int
	body            string
	netErr          error
}

func (c *tensionTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	c.calls++
	c.lastMethod = req.Method
	c.lastPath = req.URL.Path
	c.lastContentType = req.Header.Get("Content-Type")
	c.lastIfMatch = req.Header.Get("If-Match")
	if req.Body != nil {
		b, _ := io.ReadAll(req.Body)
		c.lastBody = string(b)
	}
	if c.netErr != nil {
		return nil, c.netErr
	}
	return &http.Response{
		StatusCode: c.status,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(c.body)),
	}, nil
}

// runTensionCreateOver drives the pure runTensionCreate over a fake seam, returning
// the outcome and captured stdout/stderr, and failing if the token leaks.
func runTensionCreateOver(t *testing.T, seam tensionSeam, cfg tensionCreateConfig) (Outcome, string, string) {
	t.Helper()
	var out, errb bytes.Buffer
	cfg.seam = seam
	cfg.reqCtx = context.Background()
	cfg.stdout = &out
	cfg.stderr = &errb
	outcome, _ := runTensionCreate(cfg)
	if strings.Contains(out.String()+errb.String(), meSecretToken) {
		t.Fatalf("the token leaked into output: %q", out.String()+errb.String())
	}
	return outcome, out.String(), errb.String()
}

// --- validateMeetingType (pure) --------------------------------------------

func TestValidateMeetingType(t *testing.T) {
	if err := validateMeetingType(""); err != nil {
		t.Errorf("an absent --meeting-type should be valid, got %v", err)
	}
	for _, ok := range []string{"tactical", "governance"} {
		if err := validateMeetingType(ok); err != nil {
			t.Errorf("%q should be valid, got %v", ok, err)
		}
	}
	err := validateMeetingType("weekly")
	if err == nil {
		t.Fatal("an unsupported --meeting-type should be rejected")
	}
	if !strings.Contains(err.Error(), "weekly") {
		t.Errorf("the error should name the unsupported value:\n%v", err)
	}
	for _, want := range []string{"tactical", "governance"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the error should list the supported set (missing %q):\n%v", want, err)
		}
	}
}

// --- happy path: the POST shape --------------------------------------------

func TestRunTensionCreate_BodyOnlyPostsTension(t *testing.T) {
	tr := &tensionTransport{status: 201, body: tensionCreatedBody}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, stderr := runTensionCreateOver(t, seam, tensionCreateConfig{
		id:   "role_0123",
		body: "We ship faster than we update the roadmap.",
	})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success\nstderr: %s", outcome, stderr)
	}
	if tr.calls != 1 {
		t.Fatalf("a capture is exactly one request, got %d", tr.calls)
	}
	if tr.lastMethod != http.MethodPost {
		t.Errorf("method = %q, want POST", tr.lastMethod)
	}
	if !strings.HasSuffix(tr.lastPath, "/roles/role_0123/tensions") {
		t.Errorf("path = %q, want /roles/role_0123/tensions", tr.lastPath)
	}
	if tr.lastContentType != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", tr.lastContentType)
	}
	if tr.lastBody != `{"tension":{"body":"We ship faster than we update the roadmap."}}` {
		t.Errorf("body = %s, want a body-only tension envelope", tr.lastBody)
	}
	// The created tension is printed with its ten_ id and computed status.
	for _, want := range []string{"ten_0123", "unprocessed"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("stdout should print the created tension's %q:\n%s", want, stdout)
		}
	}
}

func TestRunTensionCreate_LabelAndMeetingTypeInBody(t *testing.T) {
	tr := &tensionTransport{status: 201, body: tensionCreatedBody}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, _, stderr := runTensionCreateOver(t, seam, tensionCreateConfig{
		id:             "role_0123",
		body:           "a tension",
		label:          "Roadmap drift",
		labelSet:       true,
		meetingType:    "governance",
		meetingTypeSet: true,
	})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success\nstderr: %s", outcome, stderr)
	}
	want := `{"tension":{"body":"a tension","label":"Roadmap drift","meeting_type":"governance"}}`
	if tr.lastBody != want {
		t.Errorf("body = %s, want %s", tr.lastBody, want)
	}
}

func TestRunTensionCreate_OmittedLabelMeetingTypeSendNothing(t *testing.T) {
	// Omitted (labelSet=false, meetingTypeSet=false) AND present-but-empty both omit
	// the field — omitempty drops a still-empty value.
	for _, cfg := range []tensionCreateConfig{
		{id: "role_0123", body: "a tension"},
		{id: "role_0123", body: "a tension", label: "", labelSet: true, meetingType: "", meetingTypeSet: true},
	} {
		tr := &tensionTransport{status: 201, body: tensionCreatedBody}
		seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}
		if _, _, stderr := runTensionCreateOver(t, seam, cfg); strings.TrimSpace(stderr) != "" {
			t.Fatalf("unexpected stderr: %s", stderr)
		}
		for _, forbidden := range []string{"label", "meeting_type", "status", "sensed_by"} {
			if strings.Contains(tr.lastBody, forbidden) {
				t.Errorf("body must not carry %q, got %s", forbidden, tr.lastBody)
			}
		}
	}
}

// --- fail-fast input validation: no request --------------------------------

func TestRunTensionCreate_MissingBodyUsageErrorNoRequest(t *testing.T) {
	tr := &tensionTransport{status: 201, body: tensionCreatedBody}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, stderr := runTensionCreateOver(t, seam, tensionCreateConfig{id: "role_0123", body: ""})
	if outcome != UsageError || ExitCode(outcome) != 2 {
		t.Fatalf("outcome=%v exit=%d, want UsageError/2", outcome, ExitCode(outcome))
	}
	if !strings.Contains(stderr, "--body") {
		t.Errorf("stderr should name --body as required:\n%s", stderr)
	}
	if tr.calls != 0 {
		t.Errorf("a missing --body must be rejected before any request, got %d calls", tr.calls)
	}
	if strings.TrimSpace(stdout) != "" {
		t.Errorf("nothing should be printed on a rejected body, got:\n%s", stdout)
	}
}

func TestRunTensionCreate_WhitespaceBodyUsageErrorNoRequest(t *testing.T) {
	tr := &tensionTransport{status: 201, body: tensionCreatedBody}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, _, stderr := runTensionCreateOver(t, seam, tensionCreateConfig{id: "role_0123", body: "   "})
	if outcome != UsageError || ExitCode(outcome) != 2 {
		t.Fatalf("outcome=%v exit=%d, want UsageError/2", outcome, ExitCode(outcome))
	}
	if !strings.Contains(stderr, "--body") {
		t.Errorf("stderr should name --body as required:\n%s", stderr)
	}
	if tr.calls != 0 {
		t.Errorf("a whitespace-only --body must be rejected before any request, got %d calls", tr.calls)
	}
}

func TestRunTensionCreate_UnsupportedMeetingTypeNoRequest(t *testing.T) {
	tr := &tensionTransport{status: 201, body: tensionCreatedBody}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, _, stderr := runTensionCreateOver(t, seam, tensionCreateConfig{
		id: "role_0123", body: "a tension", meetingType: "weekly", meetingTypeSet: true,
	})
	if outcome != UsageError || ExitCode(outcome) != 2 {
		t.Fatalf("outcome=%v exit=%d, want UsageError/2", outcome, ExitCode(outcome))
	}
	if !strings.Contains(stderr, "weekly") {
		t.Errorf("stderr should name the unsupported value:\n%s", stderr)
	}
	if !strings.Contains(stderr, "governance") {
		t.Errorf("stderr should list the supported set:\n%s", stderr)
	}
	if tr.calls != 0 {
		t.Errorf("an unsupported --meeting-type must be rejected before any request, got %d calls", tr.calls)
	}
}

func TestRunTensionCreate_BadOutputUsageErrorNoRequest(t *testing.T) {
	tr := &tensionTransport{status: 201, body: tensionCreatedBody}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, _, _ := runTensionCreateOver(t, seam, tensionCreateConfig{id: "role_0123", body: "a tension", outputFlag: "xml", outputPresent: true})
	if outcome != UsageError || ExitCode(outcome) != 2 {
		t.Fatalf("outcome=%v exit=%d, want UsageError/2", outcome, ExitCode(outcome))
	}
	if tr.calls != 0 {
		t.Errorf("a bad --output must be rejected before any request, got %d calls", tr.calls)
	}
}

// --- error classification (via the shared classifier) ----------------------

func TestRunTensionCreate_NoCredentialsIsUsageError(t *testing.T) {
	tr := &tensionTransport{status: 201, body: tensionCreatedBody}
	seam := &fakeMeSeam{ctx: noTokenContext(), transport: tr}

	outcome, stdout, stderr := runTensionCreateOver(t, seam, tensionCreateConfig{id: "role_0123", body: "a tension"})
	if outcome != UsageError || ExitCode(outcome) != 2 {
		t.Fatalf("outcome=%v exit=%d, want UsageError/2", outcome, ExitCode(outcome))
	}
	if !strings.Contains(strings.ToLower(stderr), "not authenticated") {
		t.Errorf("stderr should report not authenticated:\n%s", stderr)
	}
	if strings.TrimSpace(stdout) != "" {
		t.Errorf("no tension should be printed on a credential failure, got:\n%s", stdout)
	}
}

func TestRunTensionCreate_UnknownRoleSurfacesAPIStatus(t *testing.T) {
	tr := &tensionTransport{status: 404, body: `{"detail":"Role not found"}`}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, stderr := runTensionCreateOver(t, seam, tensionCreateConfig{id: "role_ffff", body: "a tension"})
	if outcome != APIError || ExitCode(outcome) != 3 {
		t.Fatalf("an unknown role should surface APIError/3, got %v/%d\nstderr: %s", outcome, ExitCode(outcome), stderr)
	}
	if !strings.Contains(stderr, "404") {
		t.Errorf("stderr should name the HTTP status (404):\n%s", stderr)
	}
	if strings.TrimSpace(stdout) != "" {
		t.Errorf("nothing should be printed on a not-found, got:\n%s", stdout)
	}
}

// TestRunTensionCreate_RateLimitSurfacedNotRetried pins §133: a POST 429 is
// surfaced on the first occurrence and never auto-retried (017's isSafeMethod
// gate), so capture cannot double-create — exactly one request is sent.
func TestRunTensionCreate_RateLimitSurfacedNotRetried(t *testing.T) {
	tr := &tensionTransport{status: 429, body: `{"detail":"rate limited"}`}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, _, stderr := runTensionCreateOver(t, seam, tensionCreateConfig{id: "role_0123", body: "a tension"})
	if outcome != RateLimited || ExitCode(outcome) != 5 {
		t.Fatalf("outcome=%v exit=%d, want RateLimited/5\nstderr: %s", outcome, ExitCode(outcome), stderr)
	}
	if tr.calls != 1 {
		t.Errorf("a POST 429 must not be retried (no duplicate create), want 1 call, got %d", tr.calls)
	}
}

func TestRunTensionCreate_TransportErrorIsNetworkUnavailable(t *testing.T) {
	tr := &tensionTransport{netErr: errors.New("dial tcp: connection refused")}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, _, stderr := runTensionCreateOver(t, seam, tensionCreateConfig{id: "role_0123", body: "a tension"})
	if outcome != NetworkUnavailable || ExitCode(outcome) != 6 {
		t.Fatalf("outcome=%v exit=%d, want NetworkUnavailable/6\nstderr: %s", outcome, ExitCode(outcome), stderr)
	}
}

// --- structured output: raw {data: Tension} verbatim -----------------------

func TestRunTensionCreate_StructuredJSONEmitsRawPayload(t *testing.T) {
	tr := &tensionTransport{status: 201, body: tensionCreatedBody}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, _ := runTensionCreateOver(t, seam, tensionCreateConfig{id: "role_0123", body: "a tension", outputFlag: "json", outputPresent: true})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success", outcome)
	}
	for _, want := range []string{`"data"`, "ten_0123", `"role_id"`, `"sensed_by_id"`} {
		if !strings.Contains(stdout, want) {
			t.Errorf("structured json should carry the raw {data: Tension} payload, missing %q:\n%s", want, stdout)
		}
	}
	// The raw payload must not carry the human projection's block labels.
	if strings.Contains(stdout, "Sensing role:") {
		t.Errorf("structured json must not render the human projection:\n%s", stdout)
	}
}

// --- command-level wiring --------------------------------------------------

// TestTensionCommand_GroupRegistersUnderGuard pins ADR-2: the `tension` group is a
// valid non-runnable group (≥1 child) and accepted by the registration guard.
func TestTensionCommand_GroupRegistersUnderGuard(t *testing.T) {
	seam := &fakeMeSeam{ctx: validMeContext(), transport: &tensionTransport{status: 201, body: tensionCreatedBody}}
	root := NewRootCommand()
	if err := Register(root, newTensionCommand(seam)); err != nil {
		t.Fatalf("the tension group should register under the guard, got %v", err)
	}
	group, _, err := root.Find([]string{"tension"})
	if err != nil || group.Name() != "tension" {
		t.Fatalf("`tension` did not resolve: %v", err)
	}
	if group.RunE != nil || group.Run != nil {
		t.Error("the tension group must be non-runnable (no action)")
	}
	if create, _, err := group.Find([]string{"create"}); err != nil || create.Name() != "create" {
		t.Errorf("`tension create` should resolve through the group: %v", err)
	}
}

// TestTensionCreateCommand_SendsBodyEndToEnd pins the Changed()-gating and POST
// shape through a real invocation.
func TestTensionCreateCommand_SendsBodyEndToEnd(t *testing.T) {
	tr := &tensionTransport{status: 201, body: tensionCreatedBody}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	root := NewRootCommand()
	MustRegister(root, newTensionCommand(seam))
	var out, errb bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&errb)
	outcome, _ := Run(root, []string{"tension", "create", "role_0123", "--body", "a tension", "--label", "L", "--meeting-type", "governance"})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success\nstderr: %s", outcome, errb.String())
	}
	want := `{"tension":{"body":"a tension","label":"L","meeting_type":"governance"}}`
	if tr.lastBody != want {
		t.Errorf("body = %s, want %s", tr.lastBody, want)
	}
}

// TestTensionCreateCommand_RequiresExactlyOneArg pins ExactArgs(1): zero args is a
// usage error and sends no request.
func TestTensionCreateCommand_RequiresExactlyOneArg(t *testing.T) {
	tr := &tensionTransport{status: 201, body: tensionCreatedBody}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	root := NewRootCommand()
	MustRegister(root, newTensionCommand(seam))
	root.SetOut(&bytes.Buffer{})
	root.SetErr(&bytes.Buffer{})
	outcome, _ := Run(root, []string{"tension", "create", "--body", "a tension"})
	if outcome != UsageError {
		t.Errorf("zero positional args should be a UsageError, got %v", outcome)
	}
	if tr.calls != 0 {
		t.Errorf("a wrong arg count must send no request, got %d calls", tr.calls)
	}
}
