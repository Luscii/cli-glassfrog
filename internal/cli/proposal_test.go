package cli

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
)

// proposalCreatedBody is a representative createProposal 201 body: the single-object
// {data: Proposal} envelope carrying the prp_ id, the server-set draft status, the
// anchor tension, the response summary, and the available transitions. The secret
// token appears nowhere.
const proposalCreatedBody = `{"data":{"id":"prp_0123","type":"proposal","status":"draft","tension_id":"ten_0123","circle_id":"role_0123","proposer_id":"per_0123","changes":[{"id":"chg_1","type":"CreateRole","name":"Scribe"}],"response_summary":{"total":0,"no_objection":0,"bring_to_meeting":0},"expected_response_count":0,"received_response_count":0,"available_transitions":["propose"],"proposed_at":null,"response_deadline":null,"accepted_at":null,"created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"}}`

// fakeProposalSeam is the proposalSeam test double: it reuses fakeMeSeam for the
// 042 tensionSeam shape (assemble/newClient/sleep/resolveSelection/readTemplateSource)
// and adds readChangesSource over injected bytes — so the source read is exercised
// with no real pipe or filesystem. By default readChangesSource returns the flag value
// as inline bytes (the inline case); a test sets changesBytes to model a file/stdin
// source, or changesErr to model a bad source.
type fakeProposalSeam struct {
	*fakeMeSeam
	changesBytes []byte
	changesErr   error
	changesValue string // records the value readChangesSource was called with
}

func (s *fakeProposalSeam) readChangesSource(value string) ([]byte, error) {
	s.changesValue = value
	if s.changesErr != nil {
		return nil, s.changesErr
	}
	if s.changesBytes != nil {
		return s.changesBytes, nil
	}
	return []byte(value), nil
}

// runProposalCreateOver drives the pure runProposalCreate over a fake seam, returning
// the outcome and captured stdout/stderr, and failing if the token leaks.
func runProposalCreateOver(t *testing.T, seam proposalSeam, cfg proposalCreateConfig) (Outcome, string, string) {
	t.Helper()
	var out, errb bytes.Buffer
	cfg.seam = seam
	cfg.reqCtx = context.Background()
	cfg.stdout = &out
	cfg.stderr = &errb
	outcome, _ := runProposalCreate(cfg)
	if strings.Contains(out.String()+errb.String(), meSecretToken) {
		t.Fatalf("the token leaked into output: %q", out.String()+errb.String())
	}
	return outcome, out.String(), errb.String()
}

// --- happy path: the POST shape + verbatim changes -------------------------

func TestRunProposalCreate_PostsProposalVerbatim(t *testing.T) {
	tr := &tensionTransport{status: 201, body: proposalCreatedBody}
	seam := &fakeProposalSeam{fakeMeSeam: &fakeMeSeam{ctx: validMeContext(), transport: tr}}

	outcome, stdout, stderr := runProposalCreateOver(t, seam, proposalCreateConfig{
		tensionID:    "ten_0123",
		changesValue: `[{"type":"CreateRole","name":"Scribe"}]`,
	})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success\nstderr: %s", outcome, stderr)
	}
	if tr.calls != 1 {
		t.Fatalf("a create is exactly one request, got %d", tr.calls)
	}
	if tr.lastMethod != http.MethodPost {
		t.Errorf("method = %q, want POST", tr.lastMethod)
	}
	if !strings.HasSuffix(tr.lastPath, "/proposals") {
		t.Errorf("path = %q, want a /proposals suffix", tr.lastPath)
	}
	if tr.lastContentType != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", tr.lastContentType)
	}
	if tr.lastIfMatch != "" {
		t.Errorf("a create must send NO If-Match, got %q", tr.lastIfMatch)
	}
	want := `{"proposal":{"tension_id":"ten_0123","changes":[{"type":"CreateRole","name":"Scribe"}]}}`
	if tr.lastBody != want {
		t.Errorf("body = %s, want %s", tr.lastBody, want)
	}
	for _, w := range []string{"prp_0123", "draft"} {
		if !strings.Contains(stdout, w) {
			t.Errorf("stdout should print the created proposal's %q:\n%s", w, stdout)
		}
	}
}

// TestRunProposalCreate_PreservesExtraChangeKeys pins that command-specific keys
// beyond `type` are carried byte-for-byte into the request body.
func TestRunProposalCreate_PreservesExtraChangeKeys(t *testing.T) {
	tr := &tensionTransport{status: 201, body: proposalCreatedBody}
	seam := &fakeProposalSeam{fakeMeSeam: &fakeMeSeam{ctx: validMeContext(), transport: tr}}

	inline := `[{"type":"CreateRole","name":"Scribe","accountabilities":["taking minutes"]}]`
	outcome, _, stderr := runProposalCreateOver(t, seam, proposalCreateConfig{tensionID: "ten_0123", changesValue: inline})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success\nstderr: %s", outcome, stderr)
	}
	want := `{"proposal":{"tension_id":"ten_0123","changes":` + inline + `}}`
	if tr.lastBody != want {
		t.Errorf("changes were reshaped:\n got: %s\nwant: %s", tr.lastBody, want)
	}
}

// TestRunProposalCreate_FileSource pins that a file source (injected bytes) is POSTed
// verbatim.
func TestRunProposalCreate_FileSource(t *testing.T) {
	tr := &tensionTransport{status: 201, body: proposalCreatedBody}
	seam := &fakeProposalSeam{
		fakeMeSeam:   &fakeMeSeam{ctx: validMeContext(), transport: tr},
		changesBytes: []byte(`[{"type":"UpdateRole"}]`),
	}

	outcome, _, stderr := runProposalCreateOver(t, seam, proposalCreateConfig{tensionID: "ten_0123", changesValue: "changes.json"})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success\nstderr: %s", outcome, stderr)
	}
	if seam.changesValue != "changes.json" {
		t.Errorf("the seam should receive the raw flag value, got %q", seam.changesValue)
	}
	if tr.lastBody != `{"proposal":{"tension_id":"ten_0123","changes":[{"type":"UpdateRole"}]}}` {
		t.Errorf("file-sourced changes not POSTed verbatim: %s", tr.lastBody)
	}
}

// TestRunProposalCreate_StdinSource pins that a stdin source (injected bytes) is
// POSTed verbatim.
func TestRunProposalCreate_StdinSource(t *testing.T) {
	tr := &tensionTransport{status: 201, body: proposalCreatedBody}
	seam := &fakeProposalSeam{
		fakeMeSeam:   &fakeMeSeam{ctx: validMeContext(), transport: tr},
		changesBytes: []byte(`[{"type":"CreateRole"}]`),
	}

	outcome, _, stderr := runProposalCreateOver(t, seam, proposalCreateConfig{tensionID: "ten_0123", changesValue: "stdin"})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success\nstderr: %s", outcome, stderr)
	}
	if tr.lastBody != `{"proposal":{"tension_id":"ten_0123","changes":[{"type":"CreateRole"}]}}` {
		t.Errorf("stdin-sourced changes not POSTed verbatim: %s", tr.lastBody)
	}
}

// --- fail-fast input validation: no request --------------------------------

func TestRunProposalCreate_MissingChangesUsageErrorNoRequest(t *testing.T) {
	tr := &tensionTransport{status: 201, body: proposalCreatedBody}
	seam := &fakeProposalSeam{fakeMeSeam: &fakeMeSeam{ctx: validMeContext(), transport: tr}}

	outcome, stdout, stderr := runProposalCreateOver(t, seam, proposalCreateConfig{tensionID: "ten_0123", changesValue: ""})
	if outcome != UsageError || ExitCode(outcome) != 2 {
		t.Fatalf("outcome=%v exit=%d, want UsageError/2", outcome, ExitCode(outcome))
	}
	if !strings.Contains(stderr, "--changes") {
		t.Errorf("stderr should name --changes as required:\n%s", stderr)
	}
	if tr.calls != 0 {
		t.Errorf("a missing --changes must be rejected before any request, got %d calls", tr.calls)
	}
	if strings.TrimSpace(stdout) != "" {
		t.Errorf("nothing should be printed on a rejected change set, got:\n%s", stdout)
	}
}

func TestRunProposalCreate_EmptyArrayUsageErrorNoRequest(t *testing.T) {
	tr := &tensionTransport{status: 201, body: proposalCreatedBody}
	seam := &fakeProposalSeam{fakeMeSeam: &fakeMeSeam{ctx: validMeContext(), transport: tr}}

	outcome, _, stderr := runProposalCreateOver(t, seam, proposalCreateConfig{tensionID: "ten_0123", changesValue: `[]`})
	if outcome != UsageError || ExitCode(outcome) != 2 {
		t.Fatalf("outcome=%v exit=%d, want UsageError/2", outcome, ExitCode(outcome))
	}
	if !strings.Contains(stderr, "at least one change") {
		t.Errorf("stderr should say at least one change is required:\n%s", stderr)
	}
	if tr.calls != 0 {
		t.Errorf("an empty change set must be rejected before any request, got %d calls", tr.calls)
	}
}

func TestRunProposalCreate_NonArrayUsageErrorNoRequest(t *testing.T) {
	tr := &tensionTransport{status: 201, body: proposalCreatedBody}
	seam := &fakeProposalSeam{fakeMeSeam: &fakeMeSeam{ctx: validMeContext(), transport: tr}}

	outcome, _, _ := runProposalCreateOver(t, seam, proposalCreateConfig{tensionID: "ten_0123", changesValue: `not json`})
	if outcome != UsageError || ExitCode(outcome) != 2 {
		t.Fatalf("outcome=%v exit=%d, want UsageError/2", outcome, ExitCode(outcome))
	}
	if tr.calls != 0 {
		t.Errorf("an unparseable change set must be rejected before any request, got %d calls", tr.calls)
	}
}

func TestRunProposalCreate_TypelessElementUsageErrorNoRequest(t *testing.T) {
	tr := &tensionTransport{status: 201, body: proposalCreatedBody}
	seam := &fakeProposalSeam{fakeMeSeam: &fakeMeSeam{ctx: validMeContext(), transport: tr}}

	outcome, _, stderr := runProposalCreateOver(t, seam, proposalCreateConfig{tensionID: "ten_0123", changesValue: `[{"name":"Scribe"}]`})
	if outcome != UsageError || ExitCode(outcome) != 2 {
		t.Fatalf("outcome=%v exit=%d, want UsageError/2", outcome, ExitCode(outcome))
	}
	if !strings.Contains(stderr, "type") {
		t.Errorf(`stderr should report every change must carry a "type":\n%s`, stderr)
	}
	if tr.calls != 0 {
		t.Errorf("a typeless change must be rejected before any request, got %d calls", tr.calls)
	}
}

func TestRunProposalCreate_BadSourceUsageErrorNoRequest(t *testing.T) {
	tr := &tensionTransport{status: 201, body: proposalCreatedBody}
	seam := &fakeProposalSeam{
		fakeMeSeam: &fakeMeSeam{ctx: validMeContext(), transport: tr},
		changesErr: errors.New("could not read the change set file \"changes.json\": permission denied"),
	}

	outcome, _, stderr := runProposalCreateOver(t, seam, proposalCreateConfig{tensionID: "ten_0123", changesValue: "changes.json"})
	if outcome != UsageError || ExitCode(outcome) != 2 {
		t.Fatalf("outcome=%v exit=%d, want UsageError/2", outcome, ExitCode(outcome))
	}
	if !strings.Contains(stderr, "changes.json") {
		t.Errorf("stderr should name the bad source:\n%s", stderr)
	}
	if tr.calls != 0 {
		t.Errorf("a bad source must be rejected before any request, got %d calls", tr.calls)
	}
}

func TestRunProposalCreate_BadOutputUsageErrorNoRequest(t *testing.T) {
	tr := &tensionTransport{status: 201, body: proposalCreatedBody}
	seam := &fakeProposalSeam{fakeMeSeam: &fakeMeSeam{ctx: validMeContext(), transport: tr}}

	outcome, _, _ := runProposalCreateOver(t, seam, proposalCreateConfig{tensionID: "ten_0123", changesValue: `[{"type":"X"}]`, outputFlag: "xml", outputPresent: true})
	if outcome != UsageError || ExitCode(outcome) != 2 {
		t.Fatalf("outcome=%v exit=%d, want UsageError/2", outcome, ExitCode(outcome))
	}
	if tr.calls != 0 {
		t.Errorf("a bad --output must be rejected before any request, got %d calls", tr.calls)
	}
}

// --- error classification (via the shared classifier) ----------------------

func TestRunProposalCreate_NoCredentialsIsUsageError(t *testing.T) {
	tr := &tensionTransport{status: 201, body: proposalCreatedBody}
	seam := &fakeProposalSeam{fakeMeSeam: &fakeMeSeam{ctx: noTokenContext(), transport: tr}}

	outcome, stdout, stderr := runProposalCreateOver(t, seam, proposalCreateConfig{tensionID: "ten_0123", changesValue: `[{"type":"X"}]`})
	if outcome != UsageError || ExitCode(outcome) != 2 {
		t.Fatalf("outcome=%v exit=%d, want UsageError/2", outcome, ExitCode(outcome))
	}
	if !strings.Contains(strings.ToLower(stderr), "not authenticated") {
		t.Errorf("stderr should report not authenticated:\n%s", stderr)
	}
	if strings.TrimSpace(stdout) != "" {
		t.Errorf("no proposal should be printed on a credential failure, got:\n%s", stdout)
	}
}

// TestRunProposalCreate_PremiumDeniedIsPlanLimit pins that a 403 on this gated
// operation keeps PermissionError(4) — the category and exit code 061 does NOT
// change — while Plan-Limit Signal (061) now refines the wording to name the gating
// feature and frame it as a possibility (superseding the pre-061 "stays generic"
// assertion). The request is still issued with no client-side Premium pre-check, and
// the wording never instructs an upgrade.
func TestRunProposalCreate_PremiumDeniedIsPlanLimit(t *testing.T) {
	tr := &tensionTransport{status: 403, body: `{"detail":"async proposals not enabled"}`}
	seam := &fakeProposalSeam{fakeMeSeam: &fakeMeSeam{ctx: validMeContext(), transport: tr}}

	outcome, _, stderr := runProposalCreateOver(t, seam, proposalCreateConfig{tensionID: "ten_0123", changesValue: `[{"type":"X"}]`})
	if outcome != PermissionError || ExitCode(outcome) != 4 {
		t.Fatalf("a 403 should surface PermissionError/4 (unchanged by 061), got %v/%d\nstderr: %s", outcome, ExitCode(outcome), stderr)
	}
	if tr.calls != 1 {
		t.Errorf("the command must issue the request (no client-side Premium pre-check), got %d calls", tr.calls)
	}
	if !strings.Contains(stderr, "Premium async proposals") {
		t.Errorf("the plan-limit signal should name the gating feature:\n%s", stderr)
	}
	if !strings.Contains(stderr, "may not") {
		t.Errorf("the plan-limit signal should frame the limit as a possibility:\n%s", stderr)
	}
	if strings.Contains(strings.ToLower(stderr), "upgrade") {
		t.Errorf("the plan-limit signal must never instruct an upgrade:\n%s", stderr)
	}
}

func TestRunProposalCreate_UnknownTensionSurfacesAPIStatus(t *testing.T) {
	tr := &tensionTransport{status: 404, body: `{"detail":"Tension not found"}`}
	seam := &fakeProposalSeam{fakeMeSeam: &fakeMeSeam{ctx: validMeContext(), transport: tr}}

	outcome, stdout, stderr := runProposalCreateOver(t, seam, proposalCreateConfig{tensionID: "ten_ffff", changesValue: `[{"type":"X"}]`})
	if outcome != APIError || ExitCode(outcome) != 3 {
		t.Fatalf("an unknown tension should surface APIError/3, got %v/%d\nstderr: %s", outcome, ExitCode(outcome), stderr)
	}
	if !strings.Contains(stderr, "404") {
		t.Errorf("stderr should name the HTTP status (404):\n%s", stderr)
	}
	if strings.TrimSpace(stdout) != "" {
		t.Errorf("nothing should be printed on a not-found, got:\n%s", stdout)
	}
}

// TestRunProposalCreate_RateLimitSurfacedNotRetried pins §133: a POST 429 is surfaced
// on the first occurrence and never auto-retried (017's isSafeMethod gate), so a
// create cannot double-submit — exactly one request is sent.
func TestRunProposalCreate_RateLimitSurfacedNotRetried(t *testing.T) {
	tr := &tensionTransport{status: 429, body: `{"detail":"rate limited"}`}
	seam := &fakeProposalSeam{fakeMeSeam: &fakeMeSeam{ctx: validMeContext(), transport: tr}}

	outcome, _, stderr := runProposalCreateOver(t, seam, proposalCreateConfig{tensionID: "ten_0123", changesValue: `[{"type":"X"}]`})
	if outcome != RateLimited || ExitCode(outcome) != 5 {
		t.Fatalf("outcome=%v exit=%d, want RateLimited/5\nstderr: %s", outcome, ExitCode(outcome), stderr)
	}
	if tr.calls != 1 {
		t.Errorf("a POST 429 must not be retried (no duplicate proposal), want 1 call, got %d", tr.calls)
	}
}

func TestRunProposalCreate_TransportErrorIsNetworkUnavailable(t *testing.T) {
	tr := &tensionTransport{netErr: errors.New("dial tcp: connection refused")}
	seam := &fakeProposalSeam{fakeMeSeam: &fakeMeSeam{ctx: validMeContext(), transport: tr}}

	outcome, _, stderr := runProposalCreateOver(t, seam, proposalCreateConfig{tensionID: "ten_0123", changesValue: `[{"type":"X"}]`})
	if outcome != NetworkUnavailable || ExitCode(outcome) != 6 {
		t.Fatalf("outcome=%v exit=%d, want NetworkUnavailable/6\nstderr: %s", outcome, ExitCode(outcome), stderr)
	}
}

// --- structured output: raw {data: Proposal} verbatim ----------------------

func TestRunProposalCreate_StructuredJSONEmitsRawPayload(t *testing.T) {
	tr := &tensionTransport{status: 201, body: proposalCreatedBody}
	seam := &fakeProposalSeam{fakeMeSeam: &fakeMeSeam{ctx: validMeContext(), transport: tr}}

	outcome, stdout, _ := runProposalCreateOver(t, seam, proposalCreateConfig{tensionID: "ten_0123", changesValue: `[{"type":"X"}]`, outputFlag: "json", outputPresent: true})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success", outcome)
	}
	for _, want := range []string{`"data"`, "prp_0123", `"status"`, "draft"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("structured json should carry the raw {data: Proposal} payload, missing %q:\n%s", want, stdout)
		}
	}
	// The raw payload must not carry the human projection's block labels.
	if strings.Contains(stdout, "Transitions:") {
		t.Errorf("structured json must not render the human projection:\n%s", stdout)
	}
}

// --- command-level wiring --------------------------------------------------

// TestProposalCommand_GroupRegistersUnderGuard pins ADR-1: the `proposal` group is a
// valid non-runnable group (≥1 child) and accepted by the registration guard.
func TestProposalCommand_GroupRegistersUnderGuard(t *testing.T) {
	seam := &fakeProposalSeam{fakeMeSeam: &fakeMeSeam{ctx: validMeContext(), transport: &tensionTransport{status: 201, body: proposalCreatedBody}}}
	root := NewRootCommand()
	if err := Register(root, newProposalCommand(seam)); err != nil {
		t.Fatalf("the proposal group should register under the guard, got %v", err)
	}
	group, _, err := root.Find([]string{"proposal"})
	if err != nil || group.Name() != "proposal" {
		t.Fatalf("`proposal` did not resolve: %v", err)
	}
	if group.RunE != nil || group.Run != nil {
		t.Error("the proposal group must be non-runnable (no action)")
	}
	if create, _, err := group.Find([]string{"create"}); err != nil || create.Name() != "create" {
		t.Errorf("`proposal create` should resolve through the group: %v", err)
	}
}

// TestProposalCreateCommand_SendsBodyEndToEnd pins the POST shape through a real
// invocation (the single-quoted inline JSON is passed as one argv token here).
func TestProposalCreateCommand_SendsBodyEndToEnd(t *testing.T) {
	tr := &tensionTransport{status: 201, body: proposalCreatedBody}
	seam := &fakeProposalSeam{fakeMeSeam: &fakeMeSeam{ctx: validMeContext(), transport: tr}}

	root := NewRootCommand()
	MustRegister(root, newProposalCommand(seam))
	var out, errb bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&errb)
	outcome, _ := Run(root, []string{"proposal", "create", "ten_0123", "--changes", `[{"type":"CreateRole","name":"Scribe"}]`})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success\nstderr: %s", outcome, errb.String())
	}
	want := `{"proposal":{"tension_id":"ten_0123","changes":[{"type":"CreateRole","name":"Scribe"}]}}`
	if tr.lastBody != want {
		t.Errorf("body = %s, want %s", tr.lastBody, want)
	}
}

// TestProposalCreateCommand_RequiresExactlyOneArg pins ExactArgs(1): zero positionals
// (and more than one) is a usage error that sends no request.
func TestProposalCreateCommand_RequiresExactlyOneArg(t *testing.T) {
	for _, args := range [][]string{
		{"proposal", "create", "--changes", `[{"type":"X"}]`},                         // zero positional
		{"proposal", "create", "ten_0123", "ten_0456", "--changes", `[{"type":"X"}]`}, // two positionals
	} {
		tr := &tensionTransport{status: 201, body: proposalCreatedBody}
		seam := &fakeProposalSeam{fakeMeSeam: &fakeMeSeam{ctx: validMeContext(), transport: tr}}
		root := NewRootCommand()
		MustRegister(root, newProposalCommand(seam))
		root.SetOut(&bytes.Buffer{})
		root.SetErr(&bytes.Buffer{})
		outcome, _ := Run(root, args)
		if outcome != UsageError {
			t.Errorf("args %v should be a UsageError, got %v", args, outcome)
		}
		if tr.calls != 0 {
			t.Errorf("a wrong arg count must send no request, got %d calls", tr.calls)
		}
	}
}
