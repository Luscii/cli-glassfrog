package cli

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
)

// TestValidateProposalResponse_Supported pins that both spec values pass.
func TestValidateProposalResponse_Supported(t *testing.T) {
	for _, value := range []string{"no_objection", "bring_to_meeting"} {
		if err := validateProposalResponse(value); err != nil {
			t.Errorf("validateProposalResponse(%q) = %v, want nil", value, err)
		}
	}
}

// TestValidateProposalResponse_EmptyIsRequiredError pins that an empty value (the flag
// absent or explicitly blank) is the REQUIRED error — not a "no constraint" no-op like
// the optional filter validators — naming --response and listing the supported set in
// sorted order.
func TestValidateProposalResponse_EmptyIsRequiredError(t *testing.T) {
	err := validateProposalResponse("")
	if err == nil {
		t.Fatal("validateProposalResponse(\"\") = nil, want a required error")
	}
	msg := err.Error()
	if !strings.Contains(msg, "--response is required") {
		t.Errorf("error should name --response as required, got %q", msg)
	}
	// The supported set is listed in sorted order.
	if !strings.Contains(msg, "bring_to_meeting, no_objection") {
		t.Errorf("error should list the supported set in sorted order, got %q", msg)
	}
}

// TestValidateProposalResponse_UnsupportedNamesValueAndSet pins that an unsupported
// non-empty value is rejected, naming the offending value and the supported set (sorted).
func TestValidateProposalResponse_UnsupportedNamesValueAndSet(t *testing.T) {
	err := validateProposalResponse("abstain")
	if err == nil {
		t.Fatal("validateProposalResponse(\"abstain\") = nil, want an error")
	}
	msg := err.Error()
	if !strings.Contains(msg, "abstain") {
		t.Errorf("error should name the unsupported value, got %q", msg)
	}
	if !strings.Contains(msg, "bring_to_meeting, no_objection") {
		t.Errorf("error should list the supported set in sorted order, got %q", msg)
	}
}

// TestSupportedProposalResponseNames_SingleSourcedAndSorted pins that the names helper
// is single-sourced from the map and returns sorted order (deterministic usage text).
func TestSupportedProposalResponseNames_SingleSourcedAndSorted(t *testing.T) {
	names := supportedProposalResponseNames()
	if len(names) != len(supportedProposalResponses) {
		t.Fatalf("names (%d) must cover the whole set (%d)", len(names), len(supportedProposalResponses))
	}
	for i := 1; i < len(names); i++ {
		if names[i-1] > names[i] {
			t.Errorf("names not sorted: %v", names)
		}
	}
}

// proposalVoteRecordedBody is a representative createProposalResponse 201 body: the
// single-object {data: ProposalVote} envelope for a recorded no_objection response on a
// still-circulating proposal. The secret token appears nowhere.
const proposalVoteRecordedBody = `{"data":{"id":"prr_0123","type":"proposal_response","proposal_id":"prp_0123","proposal_status":"proposed_outside_meeting","value":"no_objection","created_at":"2026-06-15T12:00:00Z","updated_at":"2026-06-15T12:00:00Z"}}`

// proposalVoteAcceptedBody is the 201 body when THIS response triggered auto-acceptance:
// the parent proposal_status reads `accepted` (the consent window closed).
const proposalVoteAcceptedBody = `{"data":{"id":"prr_0123","type":"proposal_response","proposal_id":"prp_0123","proposal_status":"accepted","value":"no_objection","created_at":"2026-06-15T12:00:00Z","updated_at":"2026-06-15T12:00:00Z"}}`

// runProposalRespondOver drives the pure runProposalRespond over a fake seam, returning
// the outcome and captured stdout/stderr, and failing if the token leaks.
func runProposalRespondOver(t *testing.T, seam proposalSeam, cfg proposalRespondConfig) (Outcome, string, string) {
	t.Helper()
	var out, errb bytes.Buffer
	cfg.seam = seam
	cfg.reqCtx = context.Background()
	cfg.stdout = &out
	cfg.stderr = &errb
	outcome, _ := runProposalRespond(cfg)
	if strings.Contains(out.String()+errb.String(), meSecretToken) {
		t.Fatalf("the token leaked into output: %q", out.String()+errb.String())
	}
	return outcome, out.String(), errb.String()
}

// --- happy path: the {response:{value}} POST shape + decode-and-render ------

// TestRunProposalRespond_PostsResponseAndRenders pins the request shape (one POST to the
// /responses sub-path carrying {response:{value}}, Content-Type application/json, NO
// If-Match, no person field) and that the recorded vote is decoded and rendered.
func TestRunProposalRespond_PostsResponseAndRenders(t *testing.T) {
	tr := &tensionTransport{status: 201, body: proposalVoteRecordedBody}
	seam := &fakeProposalSeam{fakeMeSeam: &fakeMeSeam{ctx: validMeContext(), transport: tr}}

	outcome, stdout, stderr := runProposalRespondOver(t, seam, proposalRespondConfig{proposalID: "prp_0123", response: "no_objection"})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success\nstderr: %s", outcome, stderr)
	}
	if tr.calls != 1 {
		t.Fatalf("recording a response is exactly one request, got %d", tr.calls)
	}
	if tr.lastMethod != http.MethodPost {
		t.Errorf("method = %q, want POST", tr.lastMethod)
	}
	if !strings.HasSuffix(tr.lastPath, "/proposals/prp_0123/responses") {
		t.Errorf("path = %q, want a /proposals/prp_0123/responses suffix", tr.lastPath)
	}
	if tr.lastContentType != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", tr.lastContentType)
	}
	if tr.lastIfMatch != "" {
		t.Errorf("an append-create must send NO If-Match, got %q", tr.lastIfMatch)
	}
	if tr.lastBody != `{"response":{"value":"no_objection"}}` {
		t.Errorf("body = %q, want {response:{value:no_objection}}", tr.lastBody)
	}
	// The body carries no person/responder field — the server derives the person.
	for _, banned := range []string{"person", "responder", "actor"} {
		if strings.Contains(tr.lastBody, banned) {
			t.Errorf("the body must carry no person field, but contains %q: %s", banned, tr.lastBody)
		}
	}
	if !strings.Contains(stdout, "prr_0123") || !strings.Contains(stdout, "proposed_outside_meeting") {
		t.Errorf("stdout should render the recorded vote's prr_ id and proposal status:\n%s", stdout)
	}
}

// TestRunProposalRespond_BringToMeetingSends pins that --response bring_to_meeting sends
// that value and succeeds.
func TestRunProposalRespond_BringToMeetingSends(t *testing.T) {
	tr := &tensionTransport{status: 201, body: `{"data":{"id":"prr_1","type":"proposal_response","proposal_id":"prp_0123","proposal_status":"proposed_outside_meeting","value":"bring_to_meeting","created_at":"t","updated_at":"t"}}`}
	seam := &fakeProposalSeam{fakeMeSeam: &fakeMeSeam{ctx: validMeContext(), transport: tr}}

	outcome, stdout, stderr := runProposalRespondOver(t, seam, proposalRespondConfig{proposalID: "prp_0123", response: "bring_to_meeting"})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success\nstderr: %s", outcome, stderr)
	}
	if tr.lastBody != `{"response":{"value":"bring_to_meeting"}}` {
		t.Errorf("body = %q, want {response:{value:bring_to_meeting}}", tr.lastBody)
	}
	if !strings.Contains(stdout, "bring_to_meeting") {
		t.Errorf("stdout should render the recorded value:\n%s", stdout)
	}
}

// TestRunProposalRespond_StructuredJSONCarriesAcceptedStatus pins that -o json emits the
// raw {data: ProposalVote} document verbatim, including the load-bearing proposal_status
// = accepted (the auto-acceptance signal), not the human projection.
func TestRunProposalRespond_StructuredJSONCarriesAcceptedStatus(t *testing.T) {
	tr := &tensionTransport{status: 201, body: proposalVoteAcceptedBody}
	seam := &fakeProposalSeam{fakeMeSeam: &fakeMeSeam{ctx: validMeContext(), transport: tr}}

	outcome, stdout, _ := runProposalRespondOver(t, seam, proposalRespondConfig{proposalID: "prp_0123", response: "no_objection", outputFlag: "json", outputPresent: true})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success", outcome)
	}
	for _, want := range []string{`"data"`, "prr_0123", `"proposal_status"`, "accepted"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("structured json should carry the raw {data: ProposalVote} payload, missing %q:\n%s", want, stdout)
		}
	}
	if strings.Contains(stdout, "Proposal status:") {
		t.Errorf("structured json must not render the human projection:\n%s", stdout)
	}
}

// --- required/unsupported --response: fail-fast, no request ----------------

// TestRunProposalRespond_MissingResponseIsUsageErrorNoRequest pins that an omitted
// --response is a UsageError(2) naming it required + listing the set, with NO request.
func TestRunProposalRespond_MissingResponseIsUsageErrorNoRequest(t *testing.T) {
	tr := &tensionTransport{status: 201, body: proposalVoteRecordedBody}
	seam := &fakeProposalSeam{fakeMeSeam: &fakeMeSeam{ctx: validMeContext(), transport: tr}}

	outcome, stdout, stderr := runProposalRespondOver(t, seam, proposalRespondConfig{proposalID: "prp_0123", response: ""})
	if outcome != UsageError || ExitCode(outcome) != 2 {
		t.Fatalf("outcome=%v exit=%d, want UsageError/2", outcome, ExitCode(outcome))
	}
	if tr.calls != 0 {
		t.Errorf("a missing --response must be rejected before any request, got %d calls", tr.calls)
	}
	if !strings.Contains(stderr, "--response is required") {
		t.Errorf("stderr should name --response as required:\n%s", stderr)
	}
	if strings.TrimSpace(stdout) != "" {
		t.Errorf("nothing should be printed on a rejected response, got:\n%s", stdout)
	}
}

// TestRunProposalRespond_UnsupportedResponseIsUsageErrorNoRequest pins that an
// unsupported --response is a UsageError(2) naming the value + set, with NO request.
func TestRunProposalRespond_UnsupportedResponseIsUsageErrorNoRequest(t *testing.T) {
	tr := &tensionTransport{status: 201, body: proposalVoteRecordedBody}
	seam := &fakeProposalSeam{fakeMeSeam: &fakeMeSeam{ctx: validMeContext(), transport: tr}}

	outcome, _, stderr := runProposalRespondOver(t, seam, proposalRespondConfig{proposalID: "prp_0123", response: "abstain"})
	if outcome != UsageError || ExitCode(outcome) != 2 {
		t.Fatalf("outcome=%v exit=%d, want UsageError/2", outcome, ExitCode(outcome))
	}
	if tr.calls != 0 {
		t.Errorf("an unsupported --response must be rejected before any request, got %d calls", tr.calls)
	}
	if !strings.Contains(stderr, "abstain") {
		t.Errorf("stderr should name the unsupported value:\n%s", stderr)
	}
}

// TestRunProposalRespond_BadOutputUsageErrorNoRequest pins that an invalid --output is a
// UsageError(2) rejected before any request (resolved before --response validation).
func TestRunProposalRespond_BadOutputUsageErrorNoRequest(t *testing.T) {
	tr := &tensionTransport{status: 201, body: proposalVoteRecordedBody}
	seam := &fakeProposalSeam{fakeMeSeam: &fakeMeSeam{ctx: validMeContext(), transport: tr}}

	outcome, _, stderr := runProposalRespondOver(t, seam, proposalRespondConfig{proposalID: "prp_0123", response: "no_objection", outputFlag: "xml", outputPresent: true})
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

// --- 403 Premium gate / 422 already-responded / 404 unknown ----------------

// TestRunProposalRespond_PremiumDeniedIsPermission pins that a 403 (async proposals not
// enabled) surfaces as PermissionError(4) with NO plan-specific message, issued without
// any client-side Premium pre-check.
func TestRunProposalRespond_PremiumDeniedIsPermission(t *testing.T) {
	tr := &tensionTransport{status: 403, body: `{"detail":"async proposals not enabled"}`}
	seam := &fakeProposalSeam{fakeMeSeam: &fakeMeSeam{ctx: validMeContext(), transport: tr}}

	outcome, _, stderr := runProposalRespondOver(t, seam, proposalRespondConfig{proposalID: "prp_0123", response: "no_objection"})
	if outcome != PermissionError || ExitCode(outcome) != 4 {
		t.Fatalf("a 403 should surface PermissionError/4, got %v/%d\nstderr: %s", outcome, ExitCode(outcome), stderr)
	}
	if tr.calls != 1 {
		t.Errorf("the command must issue the request (no client-side Premium pre-check), got %d calls", tr.calls)
	}
	if !strings.Contains(stderr, "403") {
		t.Errorf("stderr should name the HTTP status (403):\n%s", stderr)
	}
	for _, banned := range []string{"plan", "Premium", "premium", "upgrade"} {
		if strings.Contains(stderr, banned) {
			t.Errorf("the Premium 403 must stay generic — no plan-specific message, but stderr contains %q:\n%s", banned, stderr)
		}
	}
}

// TestRunProposalRespond_SecondResponseIsAPIErrorNotRetried pins that a 422 (already
// responded) is a real APIError(3) — never retried, never folded into success — naming
// the HTTP status, with nothing on stdout.
func TestRunProposalRespond_SecondResponseIsAPIErrorNotRetried(t *testing.T) {
	tr := &tensionTransport{status: 422, body: `{"detail":"already responded"}`}
	seam := &fakeProposalSeam{fakeMeSeam: &fakeMeSeam{ctx: validMeContext(), transport: tr}}

	outcome, stdout, stderr := runProposalRespondOver(t, seam, proposalRespondConfig{proposalID: "prp_0123", response: "no_objection"})
	if outcome != APIError || ExitCode(outcome) != 3 {
		t.Fatalf("a 422 should surface APIError/3 (not success), got %v/%d\nstderr: %s", outcome, ExitCode(outcome), stderr)
	}
	if tr.calls != 1 {
		t.Errorf("a 422 must not be retried, want 1 call, got %d", tr.calls)
	}
	if !strings.Contains(stderr, "422") {
		t.Errorf("stderr should name the HTTP status (422):\n%s", stderr)
	}
	if strings.TrimSpace(stdout) != "" {
		t.Errorf("a 422 must NOT be folded into success (no print), got:\n%s", stdout)
	}
}

// TestRunProposalRespond_UnknownProposalIsAPIError pins that a 404 (unknown/invisible
// proposal) is a real APIError(3), naming the HTTP status.
func TestRunProposalRespond_UnknownProposalIsAPIError(t *testing.T) {
	tr := &tensionTransport{status: 404, body: `{"detail":"Proposal not found"}`}
	seam := &fakeProposalSeam{fakeMeSeam: &fakeMeSeam{ctx: validMeContext(), transport: tr}}

	outcome, stdout, stderr := runProposalRespondOver(t, seam, proposalRespondConfig{proposalID: "prp_ffff", response: "no_objection"})
	if outcome != APIError || ExitCode(outcome) != 3 {
		t.Fatalf("a 404 should surface APIError/3, got %v/%d\nstderr: %s", outcome, ExitCode(outcome), stderr)
	}
	if !strings.Contains(stderr, "404") {
		t.Errorf("stderr should name the HTTP status (404):\n%s", stderr)
	}
	if strings.TrimSpace(stdout) != "" {
		t.Errorf("an unknown proposal must NOT be reported as success, got:\n%s", stdout)
	}
}

// --- not-authenticated / transport / rate limit ----------------------------

func TestRunProposalRespond_NoCredentialsIsUsageError(t *testing.T) {
	tr := &tensionTransport{status: 201, body: proposalVoteRecordedBody}
	seam := &fakeProposalSeam{fakeMeSeam: &fakeMeSeam{ctx: noTokenContext(), transport: tr}}

	outcome, stdout, stderr := runProposalRespondOver(t, seam, proposalRespondConfig{proposalID: "prp_0123", response: "no_objection"})
	if outcome != UsageError || ExitCode(outcome) != 2 {
		t.Fatalf("outcome=%v exit=%d, want UsageError/2", outcome, ExitCode(outcome))
	}
	if !strings.Contains(strings.ToLower(stderr), "not authenticated") {
		t.Errorf("stderr should report not authenticated:\n%s", stderr)
	}
	if strings.TrimSpace(stdout) != "" {
		t.Errorf("nothing should be printed on a credential failure, got:\n%s", stdout)
	}
}

func TestRunProposalRespond_TransportErrorIsNetworkUnavailable(t *testing.T) {
	tr := &tensionTransport{netErr: errors.New("dial tcp: connection refused")}
	seam := &fakeProposalSeam{fakeMeSeam: &fakeMeSeam{ctx: validMeContext(), transport: tr}}

	outcome, _, stderr := runProposalRespondOver(t, seam, proposalRespondConfig{proposalID: "prp_0123", response: "no_objection"})
	if outcome != NetworkUnavailable || ExitCode(outcome) != 6 {
		t.Fatalf("outcome=%v exit=%d, want NetworkUnavailable/6\nstderr: %s", outcome, ExitCode(outcome), stderr)
	}
}

// TestRunProposalRespond_RateLimitSurfacedNotRetried pins §133: a POST 429 is surfaced
// once and never auto-retried (017's isSafeMethod gate), so a recording cannot
// double-fire — exactly one request is sent.
func TestRunProposalRespond_RateLimitSurfacedNotRetried(t *testing.T) {
	tr := &tensionTransport{status: 429, body: `{"detail":"rate limited"}`}
	seam := &fakeProposalSeam{fakeMeSeam: &fakeMeSeam{ctx: validMeContext(), transport: tr}}

	outcome, _, stderr := runProposalRespondOver(t, seam, proposalRespondConfig{proposalID: "prp_0123", response: "no_objection"})
	if outcome != RateLimited || ExitCode(outcome) != 5 {
		t.Fatalf("outcome=%v exit=%d, want RateLimited/5\nstderr: %s", outcome, ExitCode(outcome), stderr)
	}
	if tr.calls != 1 {
		t.Errorf("a POST 429 must not be retried (no double-record), want 1 call, got %d", tr.calls)
	}
}

// --- id path-escaping ------------------------------------------------------

// TestRunProposalRespond_EscapesIDAsSinglePathSegment pins that a `/` in the id is
// escaped so it cannot create extra path segments or redirect the request.
func TestRunProposalRespond_EscapesIDAsSinglePathSegment(t *testing.T) {
	tr := &tensionTransport{status: 201, body: proposalVoteRecordedBody}
	seam := &fakeProposalSeam{fakeMeSeam: &fakeMeSeam{ctx: validMeContext(), transport: tr}}

	_, _, _ = runProposalRespondOver(t, seam, proposalRespondConfig{proposalID: "prp_x/evil", response: "no_objection"})
	if strings.Contains(tr.lastPath, "prp_x/evil/responses") {
		t.Errorf("a `/` in the id must not create extra path segments: %q", tr.lastPath)
	}
	if !strings.Contains(tr.lastPath, "prp_x%2Fevil") {
		t.Errorf("the id should be escaped as a single path segment, got %q", tr.lastPath)
	}
}

// --- command-level wiring --------------------------------------------------

// TestProposalCommand_RespondResolvesThroughGroup pins that the `respond` leaf is
// attached to the shared `proposal` group and resolves through it.
func TestProposalCommand_RespondResolvesThroughGroup(t *testing.T) {
	seam := &fakeProposalSeam{fakeMeSeam: &fakeMeSeam{ctx: validMeContext(), transport: &tensionTransport{status: 201, body: proposalVoteRecordedBody}}}
	root := NewRootCommand()
	if err := Register(root, newProposalCommand(seam)); err != nil {
		t.Fatalf("the proposal group should register under the guard, got %v", err)
	}
	group, _, err := root.Find([]string{"proposal"})
	if err != nil {
		t.Fatalf("`proposal` did not resolve: %v", err)
	}
	if respond, _, err := group.Find([]string{"respond"}); err != nil || respond.Name() != "respond" {
		t.Errorf("`proposal respond` should resolve through the group: %v", err)
	}
}

// TestProposalRespondCommand_RequiresExactlyOneArg pins ExactArgs(1): zero positionals
// (and more than one) is a usage error that sends no request.
func TestProposalRespondCommand_RequiresExactlyOneArg(t *testing.T) {
	for _, args := range [][]string{
		{"proposal", "respond", "--response", "no_objection"},
		{"proposal", "respond", "prp_0123", "prp_0456", "--response", "no_objection"},
	} {
		tr := &tensionTransport{status: 201, body: proposalVoteRecordedBody}
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

// TestProposalRespondCommand_StrayFlagIsUsageErrorNoRequest pins the structural guard:
// the leaf declares only --response, so a stray flag is a cobra unknown-flag usage
// error, no request.
func TestProposalRespondCommand_StrayFlagIsUsageErrorNoRequest(t *testing.T) {
	tr := &tensionTransport{status: 201, body: proposalVoteRecordedBody}
	seam := &fakeProposalSeam{fakeMeSeam: &fakeMeSeam{ctx: validMeContext(), transport: tr}}
	root := NewRootCommand()
	MustRegister(root, newProposalCommand(seam))
	root.SetOut(&bytes.Buffer{})
	root.SetErr(&bytes.Buffer{})
	outcome, _ := Run(root, []string{"proposal", "respond", "prp_0123", "--response", "no_objection", "--changes", `[{"type":"X"}]`})
	if outcome != UsageError {
		t.Errorf("a stray flag should be a UsageError, got %v", outcome)
	}
	if tr.calls != 0 {
		t.Errorf("a stray flag must send no request, got %d calls", tr.calls)
	}
}

// TestProposalRespondCommand_SendsResponsePostEndToEnd pins the {response:{value}} POST
// shape with Content-Type set and no If-Match through a real invocation.
func TestProposalRespondCommand_SendsResponsePostEndToEnd(t *testing.T) {
	tr := &tensionTransport{status: 201, body: proposalVoteRecordedBody}
	seam := &fakeProposalSeam{fakeMeSeam: &fakeMeSeam{ctx: validMeContext(), transport: tr}}

	root := NewRootCommand()
	MustRegister(root, newProposalCommand(seam))
	var out, errb bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&errb)
	outcome, _ := Run(root, []string{"proposal", "respond", "prp_0123", "--response", "no_objection"})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success\nstderr: %s", outcome, errb.String())
	}
	if tr.lastMethod != http.MethodPost || tr.lastContentType != "application/json" || tr.lastIfMatch != "" {
		t.Errorf("want a POST with application/json and no If-Match, got method=%q ct=%q ifmatch=%q", tr.lastMethod, tr.lastContentType, tr.lastIfMatch)
	}
	if tr.lastBody != `{"response":{"value":"no_objection"}}` {
		t.Errorf("body = %q, want {response:{value:no_objection}}", tr.lastBody)
	}
	if !strings.HasSuffix(tr.lastPath, "/proposals/prp_0123/responses") {
		t.Errorf("path = %q, want a /proposals/prp_0123/responses suffix", tr.lastPath)
	}
}
