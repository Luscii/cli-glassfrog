package cli

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
)

// proposalAdvancedBody is a representative proposeProposal 200 body: the single-object
// {data: Proposal} envelope for a proposal that has just advanced — now
// `proposed_outside_meeting`, carrying the server-set response_deadline, the proposer's
// auto-recorded implicit no_objection (reflected in response_summary), and the updated
// available_transitions (propose is gone, withdraw now present). The secret token
// appears nowhere.
const proposalAdvancedBody = `{"data":{"id":"prp_0123","type":"proposal","status":"proposed_outside_meeting","tension_id":"ten_0123","circle_id":"role_0123","proposer_id":"per_0123","changes":[{"id":"chg_1","type":"CreateRole","name":"Scribe"}],"response_summary":{"total":1,"no_objection":1,"bring_to_meeting":0},"expected_response_count":5,"received_response_count":1,"available_transitions":["withdraw"],"proposed_at":"2026-06-15T12:00:00Z","response_deadline":"2026-06-22T12:00:00Z","accepted_at":null,"created_at":"2026-01-01T00:00:00Z","updated_at":"2026-06-15T12:00:00Z"}}`

// runProposalProposeOver drives the pure runProposalPropose over a fake seam, returning
// the outcome and captured stdout/stderr, and failing if the token leaks.
func runProposalProposeOver(t *testing.T, seam proposalSeam, cfg proposalProposeConfig) (Outcome, string, string) {
	t.Helper()
	var out, errb bytes.Buffer
	cfg.seam = seam
	cfg.reqCtx = context.Background()
	cfg.stdout = &out
	cfg.stderr = &errb
	outcome, _ := runProposalPropose(cfg)
	if strings.Contains(out.String()+errb.String(), meSecretToken) {
		t.Fatalf("the token leaked into output: %q", out.String()+errb.String())
	}
	return outcome, out.String(), errb.String()
}

// --- happy path: the bodyless POST shape + decode-and-render ----------------

// TestRunProposalPropose_PostsBodylessAndRenders pins the request shape (one bodyless
// POST to the /propose sub-path, no body, no Content-Type, no If-Match, no prior GET)
// and that the advanced proposal is decoded and rendered on success.
func TestRunProposalPropose_PostsBodylessAndRenders(t *testing.T) {
	tr := &tensionTransport{status: 200, body: proposalAdvancedBody}
	seam := &fakeProposalSeam{fakeMeSeam: &fakeMeSeam{ctx: validMeContext(), transport: tr}}

	outcome, stdout, stderr := runProposalProposeOver(t, seam, proposalProposeConfig{proposalID: "prp_0123"})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success\nstderr: %s", outcome, stderr)
	}
	if tr.calls != 1 {
		t.Fatalf("an advance is exactly one request (no prior GET), got %d", tr.calls)
	}
	if tr.lastMethod != http.MethodPost {
		t.Errorf("method = %q, want POST", tr.lastMethod)
	}
	if !strings.HasSuffix(tr.lastPath, "/proposals/prp_0123/propose") {
		t.Errorf("path = %q, want a /proposals/prp_0123/propose suffix", tr.lastPath)
	}
	if tr.lastContentType != "" {
		t.Errorf("a bodyless POST must send NO Content-Type, got %q", tr.lastContentType)
	}
	if tr.lastBody != "" {
		t.Errorf("a bodyless POST must send NO body, got %q", tr.lastBody)
	}
	if tr.lastIfMatch != "" {
		t.Errorf("a transition must send NO If-Match, got %q", tr.lastIfMatch)
	}
	if !strings.Contains(stdout, "proposed_outside_meeting") {
		t.Errorf("stdout should render the advanced proposal's status:\n%s", stdout)
	}
}

// TestRunProposalPropose_StructuredJSONEmitsRawPayload pins that -o json emits the raw
// {data: Proposal} document verbatim, not the human projection.
func TestRunProposalPropose_StructuredJSONEmitsRawPayload(t *testing.T) {
	tr := &tensionTransport{status: 200, body: proposalAdvancedBody}
	seam := &fakeProposalSeam{fakeMeSeam: &fakeMeSeam{ctx: validMeContext(), transport: tr}}

	outcome, stdout, _ := runProposalProposeOver(t, seam, proposalProposeConfig{proposalID: "prp_0123", outputFlag: "json", outputPresent: true})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success", outcome)
	}
	for _, want := range []string{`"data"`, "prp_0123", `"status"`, "proposed_outside_meeting", `"response_deadline"`, "2026-06-22T12:00:00Z"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("structured json should carry the raw {data: Proposal} payload, missing %q:\n%s", want, stdout)
		}
	}
	// The raw payload must not carry the human projection's block labels.
	if strings.Contains(stdout, "Transitions:") {
		t.Errorf("structured json must not render the human projection:\n%s", stdout)
	}
}

// --- 404/422 are REAL failures (the inverse of discard's 404-as-success) ----

// TestRunProposalPropose_DisallowedTransitionIsAPIError pins ADR-3: a 422 (transition
// not allowed) is a real failure routed through the shared classifier — never folded
// into success — naming the HTTP status, with nothing rendered on stdout.
func TestRunProposalPropose_DisallowedTransitionIsAPIError(t *testing.T) {
	tr := &tensionTransport{status: 422, body: `{"detail":"transition not allowed"}`}
	seam := &fakeProposalSeam{fakeMeSeam: &fakeMeSeam{ctx: validMeContext(), transport: tr}}

	outcome, stdout, stderr := runProposalProposeOver(t, seam, proposalProposeConfig{proposalID: "prp_0123"})
	if outcome != APIError || ExitCode(outcome) != 3 {
		t.Fatalf("a 422 should surface APIError/3 (not success), got %v/%d\nstderr: %s", outcome, ExitCode(outcome), stderr)
	}
	if !strings.Contains(stderr, "422") {
		t.Errorf("stderr should name the HTTP status (422):\n%s", stderr)
	}
	if strings.TrimSpace(stdout) != "" {
		t.Errorf("a disallowed transition must NOT print a proposal (no 404/422-as-success), got:\n%s", stdout)
	}
}

// TestRunProposalPropose_UnknownProposalIsAPIError pins ADR-3: a 404 (unknown/invisible
// proposal) is a real failure — explicitly NOT swallowed as success the way discard
// folds its 404 — naming the HTTP status, with nothing rendered on stdout.
func TestRunProposalPropose_UnknownProposalIsAPIError(t *testing.T) {
	tr := &tensionTransport{status: 404, body: `{"detail":"Proposal not found"}`}
	seam := &fakeProposalSeam{fakeMeSeam: &fakeMeSeam{ctx: validMeContext(), transport: tr}}

	outcome, stdout, stderr := runProposalProposeOver(t, seam, proposalProposeConfig{proposalID: "prp_ffff"})
	if outcome != APIError || ExitCode(outcome) != 3 {
		t.Fatalf("a 404 should surface APIError/3 (not success), got %v/%d\nstderr: %s", outcome, ExitCode(outcome), stderr)
	}
	if !strings.Contains(stderr, "404") {
		t.Errorf("stderr should name the HTTP status (404):\n%s", stderr)
	}
	if strings.TrimSpace(stdout) != "" {
		t.Errorf("an unknown proposal must NOT be reported as success, got:\n%s", stdout)
	}
}

// --- the Premium 403 surfaces the plan-limit signal (061) ------------------

// TestRunProposalPropose_PremiumDeniedIsPlanLimit pins that a 403 on this gated
// operation (async proposals not enabled) keeps PermissionError(4) — the category
// and exit code 061 does NOT change — while Plan-Limit Signal (061) now refines the
// wording to name the gating feature and frame it as a possibility (superseding the
// pre-061 "stays generic" assertion). The request is still issued with no
// client-side Premium pre-check, and the wording never instructs an upgrade.
func TestRunProposalPropose_PremiumDeniedIsPlanLimit(t *testing.T) {
	tr := &tensionTransport{status: 403, body: `{"detail":"async proposals not enabled"}`}
	seam := &fakeProposalSeam{fakeMeSeam: &fakeMeSeam{ctx: validMeContext(), transport: tr}}

	outcome, _, stderr := runProposalProposeOver(t, seam, proposalProposeConfig{proposalID: "prp_0123"})
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

// --- not-authenticated, bad --output: fail-fast, no request ----------------

func TestRunProposalPropose_NoCredentialsIsUsageError(t *testing.T) {
	tr := &tensionTransport{status: 200, body: proposalAdvancedBody}
	seam := &fakeProposalSeam{fakeMeSeam: &fakeMeSeam{ctx: noTokenContext(), transport: tr}}

	outcome, stdout, stderr := runProposalProposeOver(t, seam, proposalProposeConfig{proposalID: "prp_0123"})
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

func TestRunProposalPropose_BadOutputUsageErrorNoRequest(t *testing.T) {
	tr := &tensionTransport{status: 200, body: proposalAdvancedBody}
	seam := &fakeProposalSeam{fakeMeSeam: &fakeMeSeam{ctx: validMeContext(), transport: tr}}

	outcome, _, stderr := runProposalProposeOver(t, seam, proposalProposeConfig{proposalID: "prp_0123", outputFlag: "xml", outputPresent: true})
	if outcome != UsageError || ExitCode(outcome) != 2 {
		t.Fatalf("outcome=%v exit=%d, want UsageError/2", outcome, ExitCode(outcome))
	}
	if tr.calls != 0 {
		t.Errorf("a bad --output must be rejected before any request, got %d calls", tr.calls)
	}
	if !strings.Contains(stderr, "xml") {
		t.Errorf("stderr should name the rejected output value:\n%s", stderr)
	}
}

// --- rate limit: surfaced once, never auto-retried (§133) ------------------

// TestRunProposalPropose_RateLimitSurfacedNotRetried pins §133: a POST 429 is surfaced
// on the first occurrence and never auto-retried (017's isSafeMethod gate), so an
// advance cannot double-fire — exactly one request is sent.
func TestRunProposalPropose_RateLimitSurfacedNotRetried(t *testing.T) {
	tr := &tensionTransport{status: 429, body: `{"detail":"rate limited"}`}
	seam := &fakeProposalSeam{fakeMeSeam: &fakeMeSeam{ctx: validMeContext(), transport: tr}}

	outcome, _, stderr := runProposalProposeOver(t, seam, proposalProposeConfig{proposalID: "prp_0123"})
	if outcome != RateLimited || ExitCode(outcome) != 5 {
		t.Fatalf("outcome=%v exit=%d, want RateLimited/5\nstderr: %s", outcome, ExitCode(outcome), stderr)
	}
	if tr.calls != 1 {
		t.Errorf("a POST 429 must not be retried (no double-advance), want 1 call, got %d", tr.calls)
	}
}

func TestRunProposalPropose_TransportErrorIsNetworkUnavailable(t *testing.T) {
	tr := &tensionTransport{netErr: errors.New("dial tcp: connection refused")}
	seam := &fakeProposalSeam{fakeMeSeam: &fakeMeSeam{ctx: validMeContext(), transport: tr}}

	outcome, _, stderr := runProposalProposeOver(t, seam, proposalProposeConfig{proposalID: "prp_0123"})
	if outcome != NetworkUnavailable || ExitCode(outcome) != 6 {
		t.Fatalf("outcome=%v exit=%d, want NetworkUnavailable/6\nstderr: %s", outcome, ExitCode(outcome), stderr)
	}
}

// --- id path-escaping ------------------------------------------------------

// TestRunProposalPropose_EscapesIDAsSinglePathSegment pins that a `/` in the id is
// escaped so it cannot create extra path segments or redirect the request.
func TestRunProposalPropose_EscapesIDAsSinglePathSegment(t *testing.T) {
	tr := &tensionTransport{status: 200, body: proposalAdvancedBody}
	seam := &fakeProposalSeam{fakeMeSeam: &fakeMeSeam{ctx: validMeContext(), transport: tr}}

	_, _, _ = runProposalProposeOver(t, seam, proposalProposeConfig{proposalID: "prp_x/evil"})
	if strings.Contains(tr.lastPath, "prp_x/evil/propose") {
		t.Errorf("a `/` in the id must not create extra path segments: %q", tr.lastPath)
	}
	if !strings.Contains(tr.lastPath, "prp_x%2Fevil") {
		t.Errorf("the id should be escaped as a single path segment, got %q", tr.lastPath)
	}
}

// --- command-level wiring --------------------------------------------------

// TestProposalCommand_ProposeResolvesThroughGroup pins that the `propose` leaf is
// attached to the shared `proposal` group and resolves through it.
func TestProposalCommand_ProposeResolvesThroughGroup(t *testing.T) {
	seam := &fakeProposalSeam{fakeMeSeam: &fakeMeSeam{ctx: validMeContext(), transport: &tensionTransport{status: 200, body: proposalAdvancedBody}}}
	root := NewRootCommand()
	if err := Register(root, newProposalCommand(seam)); err != nil {
		t.Fatalf("the proposal group should register under the guard, got %v", err)
	}
	group, _, err := root.Find([]string{"proposal"})
	if err != nil {
		t.Fatalf("`proposal` did not resolve: %v", err)
	}
	if propose, _, err := group.Find([]string{"propose"}); err != nil || propose.Name() != "propose" {
		t.Errorf("`proposal propose` should resolve through the group: %v", err)
	}
}

// TestProposalProposeCommand_SendsBodylessPostEndToEnd pins the bodyless POST shape
// through a real invocation.
func TestProposalProposeCommand_SendsBodylessPostEndToEnd(t *testing.T) {
	tr := &tensionTransport{status: 200, body: proposalAdvancedBody}
	seam := &fakeProposalSeam{fakeMeSeam: &fakeMeSeam{ctx: validMeContext(), transport: tr}}

	root := NewRootCommand()
	MustRegister(root, newProposalCommand(seam))
	var out, errb bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&errb)
	outcome, _ := Run(root, []string{"proposal", "propose", "prp_0123"})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success\nstderr: %s", outcome, errb.String())
	}
	if tr.lastMethod != http.MethodPost || tr.lastBody != "" || tr.lastContentType != "" {
		t.Errorf("want a bodyless POST with no Content-Type, got method=%q body=%q ct=%q", tr.lastMethod, tr.lastBody, tr.lastContentType)
	}
	if !strings.HasSuffix(tr.lastPath, "/proposals/prp_0123/propose") {
		t.Errorf("path = %q, want a /proposals/prp_0123/propose suffix", tr.lastPath)
	}
}

// TestProposalProposeCommand_RequiresExactlyOneArg pins ExactArgs(1): zero positionals
// (and more than one) is a usage error that sends no request.
func TestProposalProposeCommand_RequiresExactlyOneArg(t *testing.T) {
	for _, args := range [][]string{
		{"proposal", "propose"},                         // zero positional
		{"proposal", "propose", "prp_0123", "prp_0456"}, // two positionals
	} {
		tr := &tensionTransport{status: 200, body: proposalAdvancedBody}
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

// TestProposalProposeCommand_StrayFlagIsUsageErrorNoRequest pins the structural guard:
// the flagless leaf rejects a stray flag as a cobra unknown-flag usage error, no request.
func TestProposalProposeCommand_StrayFlagIsUsageErrorNoRequest(t *testing.T) {
	tr := &tensionTransport{status: 200, body: proposalAdvancedBody}
	seam := &fakeProposalSeam{fakeMeSeam: &fakeMeSeam{ctx: validMeContext(), transport: tr}}
	root := NewRootCommand()
	MustRegister(root, newProposalCommand(seam))
	root.SetOut(&bytes.Buffer{})
	root.SetErr(&bytes.Buffer{})
	outcome, _ := Run(root, []string{"proposal", "propose", "prp_0123", "--changes", `[{"type":"X"}]`})
	if outcome != UsageError {
		t.Errorf("a stray flag should be a UsageError, got %v", outcome)
	}
	if tr.calls != 0 {
		t.Errorf("a stray flag must send no request, got %d calls", tr.calls)
	}
}
