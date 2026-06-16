package cli

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
)

// proposalWithdrawnBody is a representative withdrawProposal 200 body: the single-object
// {data: Proposal} envelope for a proposal that has just been withdrawn — now back in
// `draft`, with proposed_at/response_deadline CLEARED (null), the prior responses deleted
// server-side (response_summary zeroed), and the updated available_transitions (withdraw
// is gone, propose now offered again). The secret token appears nowhere.
const proposalWithdrawnBody = `{"data":{"id":"prp_0123","type":"proposal","status":"draft","tension_id":"ten_0123","circle_id":"role_0123","proposer_id":"per_0123","changes":[{"id":"chg_1","type":"CreateRole","name":"Scribe"}],"response_summary":{"total":0,"no_objection":0,"bring_to_meeting":0},"expected_response_count":0,"received_response_count":0,"available_transitions":["propose"],"proposed_at":null,"response_deadline":null,"accepted_at":null,"created_at":"2026-01-01T00:00:00Z","updated_at":"2026-06-15T12:00:00Z"}}`

// runProposalWithdrawOver drives the pure runProposalWithdraw over a fake seam, returning
// the outcome and captured stdout/stderr, and failing if the token leaks.
func runProposalWithdrawOver(t *testing.T, seam proposalSeam, cfg proposalWithdrawConfig) (Outcome, string, string) {
	t.Helper()
	var out, errb bytes.Buffer
	cfg.seam = seam
	cfg.reqCtx = context.Background()
	cfg.stdout = &out
	cfg.stderr = &errb
	outcome, _ := runProposalWithdraw(cfg)
	if strings.Contains(out.String()+errb.String(), meSecretToken) {
		t.Fatalf("the token leaked into output: %q", out.String()+errb.String())
	}
	return outcome, out.String(), errb.String()
}

// --- happy path: the bodyless POST shape + decode-and-render ----------------

// TestRunProposalWithdraw_PostsBodylessAndRenders pins the request shape (one bodyless
// POST to the /withdraw sub-path, no body, no Content-Type, no If-Match, no prior GET)
// and that the withdrawn proposal is decoded and rendered back in `draft` on success.
func TestRunProposalWithdraw_PostsBodylessAndRenders(t *testing.T) {
	tr := &tensionTransport{status: 200, body: proposalWithdrawnBody}
	seam := &fakeProposalSeam{fakeMeSeam: &fakeMeSeam{ctx: validMeContext(), transport: tr}}

	outcome, stdout, stderr := runProposalWithdrawOver(t, seam, proposalWithdrawConfig{proposalID: "prp_0123"})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success\nstderr: %s", outcome, stderr)
	}
	if tr.calls != 1 {
		t.Fatalf("a withdraw is exactly one request (no prior GET), got %d", tr.calls)
	}
	if tr.lastMethod != http.MethodPost {
		t.Errorf("method = %q, want POST", tr.lastMethod)
	}
	if !strings.HasSuffix(tr.lastPath, "/proposals/prp_0123/withdraw") {
		t.Errorf("path = %q, want a /proposals/prp_0123/withdraw suffix", tr.lastPath)
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
	if !strings.Contains(stdout, "draft") {
		t.Errorf("stdout should render the withdrawn proposal's draft status:\n%s", stdout)
	}
	// The withdraw is destructive but server-owned — the command narrates none of the
	// deleted responses (plan ADR-2). The status word "draft" is data; a "deleted"
	// narration is not.
	if strings.Contains(strings.ToLower(stdout+stderr), "deleted") {
		t.Errorf("the command must not narrate the deleted responses:\nstdout: %s\nstderr: %s", stdout, stderr)
	}
}

// TestRunProposalWithdraw_StructuredJSONEmitsRawPayload pins that -o json emits the raw
// {data: Proposal} document verbatim, not the human projection — carrying the cleared
// deadline (null) and the updated transitions.
func TestRunProposalWithdraw_StructuredJSONEmitsRawPayload(t *testing.T) {
	tr := &tensionTransport{status: 200, body: proposalWithdrawnBody}
	seam := &fakeProposalSeam{fakeMeSeam: &fakeMeSeam{ctx: validMeContext(), transport: tr}}

	outcome, stdout, _ := runProposalWithdrawOver(t, seam, proposalWithdrawConfig{proposalID: "prp_0123", outputFlag: "json", outputPresent: true})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success", outcome)
	}
	for _, want := range []string{`"data"`, "prp_0123", `"status"`, `"draft"`, `"response_deadline"`, "null", `"available_transitions"`, "propose"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("structured json should carry the raw {data: Proposal} payload, missing %q:\n%s", want, stdout)
		}
	}
	// The raw payload must not carry the human projection's block labels.
	if strings.Contains(stdout, "Transitions:") {
		t.Errorf("structured json must not render the human projection:\n%s", stdout)
	}
}

// TestRunProposalWithdraw_HumanShowsClearedDeadlineAndTransitions pins that the human
// render surfaces the withdrawn proposal back in `draft` with the deadline cleared and the
// server's updated transitions — the result the agent reads.
func TestRunProposalWithdraw_HumanShowsClearedDeadlineAndTransitions(t *testing.T) {
	tr := &tensionTransport{status: 200, body: proposalWithdrawnBody}
	seam := &fakeProposalSeam{fakeMeSeam: &fakeMeSeam{ctx: validMeContext(), transport: tr}}

	outcome, stdout, stderr := runProposalWithdrawOver(t, seam, proposalWithdrawConfig{proposalID: "prp_0123"})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success\nstderr: %s", outcome, stderr)
	}
	if !strings.Contains(stdout, "draft") {
		t.Errorf("the printed proposal should show status draft:\n%s", stdout)
	}
	// The withdrawn proposal offers `propose` again; the human render surfaces the
	// server's available transitions.
	if !strings.Contains(stdout, "propose") {
		t.Errorf("the printed proposal should carry the server's updated transitions (propose):\n%s", stdout)
	}
}

// --- 404/422 are REAL failures (the inverse of discard's 404-as-success) ----

// TestRunProposalWithdraw_DisallowedTransitionIsAPIError pins ADR-1: a 422 (transition
// not allowed — already `draft`, or `withdraw` not offered) is a real failure routed
// through the shared classifier — never folded into success — naming the HTTP status,
// with nothing rendered on stdout.
func TestRunProposalWithdraw_DisallowedTransitionIsAPIError(t *testing.T) {
	tr := &tensionTransport{status: 422, body: `{"detail":"transition not allowed"}`}
	seam := &fakeProposalSeam{fakeMeSeam: &fakeMeSeam{ctx: validMeContext(), transport: tr}}

	outcome, stdout, stderr := runProposalWithdrawOver(t, seam, proposalWithdrawConfig{proposalID: "prp_0123"})
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

// TestRunProposalWithdraw_UnknownProposalIsAPIError pins ADR-1: a 404 (unknown/invisible
// proposal) is a real failure — explicitly NOT swallowed as success the way discard folds
// its 404 — naming the HTTP status, with nothing rendered on stdout.
func TestRunProposalWithdraw_UnknownProposalIsAPIError(t *testing.T) {
	tr := &tensionTransport{status: 404, body: `{"detail":"Proposal not found"}`}
	seam := &fakeProposalSeam{fakeMeSeam: &fakeMeSeam{ctx: validMeContext(), transport: tr}}

	outcome, stdout, stderr := runProposalWithdrawOver(t, seam, proposalWithdrawConfig{proposalID: "prp_ffff"})
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

// TestRunProposalWithdraw_PremiumDeniedIsPlanLimit pins that a 403 on this gated
// operation keeps PermissionError(4) — the category and exit code 061 does NOT
// change — while Plan-Limit Signal (061) now refines the wording to name the gating
// feature and frame it as a possibility (superseding the pre-061 "stays generic"
// assertion). The request is still issued with no client-side Premium pre-check, and
// the wording never instructs an upgrade.
func TestRunProposalWithdraw_PremiumDeniedIsPlanLimit(t *testing.T) {
	tr := &tensionTransport{status: 403, body: `{"detail":"async proposals not enabled"}`}
	seam := &fakeProposalSeam{fakeMeSeam: &fakeMeSeam{ctx: validMeContext(), transport: tr}}

	outcome, _, stderr := runProposalWithdrawOver(t, seam, proposalWithdrawConfig{proposalID: "prp_0123"})
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

func TestRunProposalWithdraw_NoCredentialsIsUsageError(t *testing.T) {
	tr := &tensionTransport{status: 200, body: proposalWithdrawnBody}
	seam := &fakeProposalSeam{fakeMeSeam: &fakeMeSeam{ctx: noTokenContext(), transport: tr}}

	outcome, stdout, stderr := runProposalWithdrawOver(t, seam, proposalWithdrawConfig{proposalID: "prp_0123"})
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

func TestRunProposalWithdraw_BadOutputUsageErrorNoRequest(t *testing.T) {
	tr := &tensionTransport{status: 200, body: proposalWithdrawnBody}
	seam := &fakeProposalSeam{fakeMeSeam: &fakeMeSeam{ctx: validMeContext(), transport: tr}}

	outcome, _, stderr := runProposalWithdrawOver(t, seam, proposalWithdrawConfig{proposalID: "prp_0123", outputFlag: "xml", outputPresent: true})
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

// TestRunProposalWithdraw_RateLimitSurfacedNotRetried pins §133: a POST 429 is surfaced
// on the first occurrence and never auto-retried (017's isSafeMethod gate), so a withdraw
// cannot double-fire — exactly one request is sent.
func TestRunProposalWithdraw_RateLimitSurfacedNotRetried(t *testing.T) {
	tr := &tensionTransport{status: 429, body: `{"detail":"rate limited"}`}
	seam := &fakeProposalSeam{fakeMeSeam: &fakeMeSeam{ctx: validMeContext(), transport: tr}}

	outcome, _, stderr := runProposalWithdrawOver(t, seam, proposalWithdrawConfig{proposalID: "prp_0123"})
	if outcome != RateLimited || ExitCode(outcome) != 5 {
		t.Fatalf("outcome=%v exit=%d, want RateLimited/5\nstderr: %s", outcome, ExitCode(outcome), stderr)
	}
	if tr.calls != 1 {
		t.Errorf("a POST 429 must not be retried (no double-withdraw), want 1 call, got %d", tr.calls)
	}
}

func TestRunProposalWithdraw_TransportErrorIsNetworkUnavailable(t *testing.T) {
	tr := &tensionTransport{netErr: errors.New("dial tcp: connection refused")}
	seam := &fakeProposalSeam{fakeMeSeam: &fakeMeSeam{ctx: validMeContext(), transport: tr}}

	outcome, _, stderr := runProposalWithdrawOver(t, seam, proposalWithdrawConfig{proposalID: "prp_0123"})
	if outcome != NetworkUnavailable || ExitCode(outcome) != 6 {
		t.Fatalf("outcome=%v exit=%d, want NetworkUnavailable/6\nstderr: %s", outcome, ExitCode(outcome), stderr)
	}
}

// --- id path-escaping ------------------------------------------------------

// TestRunProposalWithdraw_EscapesIDAsSinglePathSegment pins that a `/` in the id is
// escaped so it cannot create extra path segments or redirect the request.
func TestRunProposalWithdraw_EscapesIDAsSinglePathSegment(t *testing.T) {
	tr := &tensionTransport{status: 200, body: proposalWithdrawnBody}
	seam := &fakeProposalSeam{fakeMeSeam: &fakeMeSeam{ctx: validMeContext(), transport: tr}}

	_, _, _ = runProposalWithdrawOver(t, seam, proposalWithdrawConfig{proposalID: "prp_x/evil"})
	if strings.Contains(tr.lastPath, "prp_x/evil/withdraw") {
		t.Errorf("a `/` in the id must not create extra path segments: %q", tr.lastPath)
	}
	if !strings.Contains(tr.lastPath, "prp_x%2Fevil") {
		t.Errorf("the id should be escaped as a single path segment, got %q", tr.lastPath)
	}
}

// --- command-level wiring --------------------------------------------------

// TestProposalCommand_WithdrawResolvesThroughGroup pins that the `withdraw` leaf is
// attached to the shared `proposal` group and resolves through it.
func TestProposalCommand_WithdrawResolvesThroughGroup(t *testing.T) {
	seam := &fakeProposalSeam{fakeMeSeam: &fakeMeSeam{ctx: validMeContext(), transport: &tensionTransport{status: 200, body: proposalWithdrawnBody}}}
	root := NewRootCommand()
	if err := Register(root, newProposalCommand(seam)); err != nil {
		t.Fatalf("the proposal group should register under the guard, got %v", err)
	}
	group, _, err := root.Find([]string{"proposal"})
	if err != nil {
		t.Fatalf("`proposal` did not resolve: %v", err)
	}
	if withdraw, _, err := group.Find([]string{"withdraw"}); err != nil || withdraw.Name() != "withdraw" {
		t.Errorf("`proposal withdraw` should resolve through the group: %v", err)
	}
}

// TestProposalWithdrawCommand_SendsBodylessPostEndToEnd pins the bodyless POST shape
// through a real invocation.
func TestProposalWithdrawCommand_SendsBodylessPostEndToEnd(t *testing.T) {
	tr := &tensionTransport{status: 200, body: proposalWithdrawnBody}
	seam := &fakeProposalSeam{fakeMeSeam: &fakeMeSeam{ctx: validMeContext(), transport: tr}}

	root := NewRootCommand()
	MustRegister(root, newProposalCommand(seam))
	var out, errb bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&errb)
	outcome, _ := Run(root, []string{"proposal", "withdraw", "prp_0123"})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success\nstderr: %s", outcome, errb.String())
	}
	if tr.lastMethod != http.MethodPost || tr.lastBody != "" || tr.lastContentType != "" {
		t.Errorf("want a bodyless POST with no Content-Type, got method=%q body=%q ct=%q", tr.lastMethod, tr.lastBody, tr.lastContentType)
	}
	if !strings.HasSuffix(tr.lastPath, "/proposals/prp_0123/withdraw") {
		t.Errorf("path = %q, want a /proposals/prp_0123/withdraw suffix", tr.lastPath)
	}
}

// TestProposalWithdrawCommand_RequiresExactlyOneArg pins ExactArgs(1): zero positionals
// (and more than one) is a usage error that sends no request.
func TestProposalWithdrawCommand_RequiresExactlyOneArg(t *testing.T) {
	for _, args := range [][]string{
		{"proposal", "withdraw"},                         // zero positional
		{"proposal", "withdraw", "prp_0123", "prp_0456"}, // two positionals
	} {
		tr := &tensionTransport{status: 200, body: proposalWithdrawnBody}
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

// TestProposalWithdrawCommand_StrayFlagIsUsageErrorNoRequest pins the structural guard and
// plan ADR-2's no-confirmation/no---force stance: the flagless leaf rejects a stray
// --force (and any other flag) as a cobra unknown-flag usage error, no request.
func TestProposalWithdrawCommand_StrayFlagIsUsageErrorNoRequest(t *testing.T) {
	for _, stray := range [][]string{
		{"proposal", "withdraw", "prp_0123", "--force"},
		{"proposal", "withdraw", "prp_0123", "--yes"},
		{"proposal", "withdraw", "prp_0123", "--changes", `[{"type":"X"}]`},
	} {
		tr := &tensionTransport{status: 200, body: proposalWithdrawnBody}
		seam := &fakeProposalSeam{fakeMeSeam: &fakeMeSeam{ctx: validMeContext(), transport: tr}}
		root := NewRootCommand()
		MustRegister(root, newProposalCommand(seam))
		root.SetOut(&bytes.Buffer{})
		root.SetErr(&bytes.Buffer{})
		outcome, _ := Run(root, stray)
		if outcome != UsageError {
			t.Errorf("a stray flag %v should be a UsageError, got %v", stray, outcome)
		}
		if tr.calls != 0 {
			t.Errorf("a stray flag must send no request, got %d calls", tr.calls)
		}
	}
}
