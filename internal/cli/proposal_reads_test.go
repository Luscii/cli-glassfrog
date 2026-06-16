package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

// --- canned GET /proposals bodies ------------------------------------------
//
// They carry the Proposal shape in the API's snake_case names, with one proposal whose
// nullable anchors are null to exercise the explicit-absence markers, and the secret
// token nowhere.

const proposalsPageComplete = `{"data":[
  {"id":"prp_1","type":"proposal","status":"draft","tension_id":"ten_1","circle_id":"role_1","proposer_id":"per_1","changes":[{"id":"chg_1","type":"CreateRole","name":"Scribe"}],"response_summary":{"total":1,"no_objection":1,"bring_to_meeting":0},"available_transitions":["propose"],"proposed_at":null,"response_deadline":null,"accepted_at":null,"created_at":"t","updated_at":"t"},
  {"id":"prp_2","type":"proposal","status":"accepted","tension_id":null,"circle_id":null,"proposer_id":null,"changes":[],"response_summary":{"total":0,"no_objection":0,"bring_to_meeting":0},"available_transitions":[],"proposed_at":null,"response_deadline":null,"accepted_at":null,"created_at":"t","updated_at":"t"}
],"meta":{"pagination":{"per_page":100,"has_next_page":false,"next_cursor":""}}}`

const proposalsPageEmpty = `{"data":[],"meta":{"pagination":{"per_page":100,"has_next_page":false,"next_cursor":""}}}`

// proposalDocumentBody is a representative GET /proposals/{id} body: the single-object
// {data: Proposal} envelope carrying the full detail — changes (with a free-form key),
// the aggregate response_summary, and available_transitions.
const proposalDocumentBody = `{"data":{"id":"prp_0123","type":"proposal","status":"proposed_outside_meeting","tension_id":"ten_0123","circle_id":"role_0123","proposer_id":"per_0123","changes":[{"id":"chg_1","type":"CreateRole","name":"Scribe"}],"response_summary":{"total":2,"no_objection":1,"bring_to_meeting":1},"expected_response_count":3,"received_response_count":2,"available_transitions":["propose","withdraw"],"proposed_at":"2026-01-03T00:00:00Z","response_deadline":"2026-01-10T00:00:00Z","accepted_at":null,"created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-02T00:00:00Z"}}`

// proposalsPage builds a one-proposal page; a non-empty nextCursor marks more pages.
func proposalsPage(id, status, nextCursor string) string {
	hasNext := "false"
	if nextCursor != "" {
		hasNext = "true"
	}
	return `{"data":[{"id":"` + id + `","type":"proposal","status":"` + status + `","tension_id":"ten_1","circle_id":"role_1","proposer_id":"per_1","changes":[{"id":"chg_1","type":"CreateRole"}],"response_summary":{"total":0,"no_objection":0,"bring_to_meeting":0},"available_transitions":["propose"],"created_at":"t","updated_at":"t"}],` +
		`"meta":{"pagination":{"per_page":1,"has_next_page":` + hasNext + `,"next_cursor":"` + nextCursor + `"}}}`
}

// runProposalListOver drives the pure runProposalList over a fake seam, returning the
// outcome and captured stdout/stderr, and failing if the token leaks.
func runProposalListOver(t *testing.T, seam proposalSeam, cfg proposalsConfig) (Outcome, string, string) {
	t.Helper()
	var out, errb bytes.Buffer
	cfg.seam = seam
	cfg.reqCtx = context.Background()
	cfg.stdout = &out
	cfg.stderr = &errb
	outcome, _ := runProposalList(cfg)
	if strings.Contains(out.String()+errb.String(), meSecretToken) {
		t.Fatalf("the token leaked into output: %q", out.String()+errb.String())
	}
	return outcome, out.String(), errb.String()
}

// runProposalGetOver drives the pure runProposalGet over a fake seam, returning the
// outcome and captured stdout/stderr, and failing if the token leaks.
func runProposalGetOver(t *testing.T, seam proposalSeam, cfg proposalGetConfig, id string) (Outcome, string, string) {
	t.Helper()
	var out, errb bytes.Buffer
	cfg.seam = seam
	cfg.reqCtx = context.Background()
	cfg.stdout = &out
	cfg.stderr = &errb
	outcome, _ := runProposalGet(cfg, id)
	if strings.Contains(out.String()+errb.String(), meSecretToken) {
		t.Fatalf("the token leaked into output: %q", out.String()+errb.String())
	}
	return outcome, out.String(), errb.String()
}

// --- T003: validateProposalStatus (pure) -----------------------------------
//
// A NEW validator over the proposal status set, distinct from the action/project
// validateStatus AND the tension validateTensionStatus sets — reusing either would
// accept invalid proposal statuses and reject valid ones (plan ADR-3).

func TestValidateProposalStatus_AcceptsSupportedAndEmpty(t *testing.T) {
	for _, s := range []string{"draft", "proposed_outside_meeting", "escalated", "accepted", "draft_with_conflicts", ""} {
		if err := validateProposalStatus(s); err != nil {
			t.Errorf("validateProposalStatus(%q) = %v, want nil", s, err)
		}
	}
}

// TestValidateProposalStatus_RejectsUnsupported pins that an unsupported value is
// rejected with a message naming the value and listing the sorted supported set.
func TestValidateProposalStatus_RejectsUnsupported(t *testing.T) {
	err := validateProposalStatus("open")
	if err == nil {
		t.Fatal("validateProposalStatus(\"open\") = nil, want an error")
	}
	if !strings.Contains(err.Error(), "open") {
		t.Errorf("error should name the unsupported value: %v", err)
	}
	// Lists the supported set in sorted order (accepted first, ...).
	if !strings.Contains(err.Error(), "accepted, draft, draft_with_conflicts, escalated, proposed_outside_meeting") {
		t.Errorf("error should list the sorted supported set: %v", err)
	}
}

// TestValidateProposalStatus_IncludesDraftWithConflicts pins draft_with_conflicts is in
// the set (the value the FEATURE-MODEL prose omits — easily missed).
func TestValidateProposalStatus_IncludesDraftWithConflicts(t *testing.T) {
	if !supportedProposalStatuses["draft_with_conflicts"] {
		t.Error("draft_with_conflicts must be in the supported proposal status set")
	}
	if err := validateProposalStatus("draft_with_conflicts"); err != nil {
		t.Errorf("draft_with_conflicts should validate, got %v", err)
	}
}

// --- T004: runProposalList -------------------------------------------------

// TestRunProposalList_WalksAndPrintsProjection pins the default walk: GET /proposals to
// completion, each proposal printed as a projection, exit 0.
func TestRunProposalList_WalksAndPrintsProjection(t *testing.T) {
	tr := &cannedTransport{status: 200, body: proposalsPageComplete}
	seam := &fakeProposalSeam{fakeMeSeam: &fakeMeSeam{ctx: validMeContext(), transport: tr}}

	outcome, stdout, stderr := runProposalListOver(t, seam, proposalsConfig{})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success\nstderr: %s", outcome, stderr)
	}
	if !strings.HasSuffix(tr.lastPath, "/proposals") {
		t.Errorf("the request should target /proposals, got %q", tr.lastPath)
	}
	for _, want := range []string{"prp_1  [draft]", "prp_2  [accepted]"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("each proposal should print as a projection, missing %q:\n%s", want, stdout)
		}
	}
}

// TestRunProposalList_EmptyIsCleanSuccess pins that no visible proposals prints
// `no proposals` and exits 0.
func TestRunProposalList_EmptyIsCleanSuccess(t *testing.T) {
	tr := &cannedTransport{status: 200, body: proposalsPageEmpty}
	seam := &fakeProposalSeam{fakeMeSeam: &fakeMeSeam{ctx: validMeContext(), transport: tr}}

	outcome, stdout, stderr := runProposalListOver(t, seam, proposalsConfig{})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success\nstderr: %s", outcome, stderr)
	}
	if !strings.Contains(stdout, "no proposals") {
		t.Errorf("an empty visible set should print `no proposals`:\n%s", stdout)
	}
}

// TestRunProposalList_FiltersSentOnlyWhenSupplied pins each filter is sent as its query
// parameter when supplied, and an omitted filter sends nothing.
func TestRunProposalList_FiltersSentOnlyWhenSupplied(t *testing.T) {
	tr := &cannedTransport{status: 200, body: proposalsPageComplete}
	seam := &fakeProposalSeam{fakeMeSeam: &fakeMeSeam{ctx: validMeContext(), transport: tr}}

	outcome, _, stderr := runProposalListOver(t, seam, proposalsConfig{
		status:        "proposed_outside_meeting",
		roleID:        "role_0123",
		proposerID:    "per_0123",
		proposedAfter: "2026-01-01T00:00:00Z",
		// acceptedAfter omitted — must send nothing.
	})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success\nstderr: %s", outcome, stderr)
	}
	for param, want := range map[string]string{
		"status":         "proposed_outside_meeting",
		"role_id":        "role_0123",
		"proposer_id":    "per_0123",
		"proposed_after": "2026-01-01T00:00:00Z",
	} {
		if got := tr.lastQuery.Get(param); got != want {
			t.Errorf("query %q = %q, want %q (full query %q)", param, got, want, tr.lastQuery.Encode())
		}
	}
	if _, present := tr.lastQuery["accepted_after"]; present {
		t.Errorf("an omitted --accepted-after must send no parameter, got %v", tr.lastQuery["accepted_after"])
	}
}

// TestRunProposalList_UnsupportedStatusNoRequest pins the transport tripwire: an
// unsupported --status is a UsageError(2) naming the value + supported set, with NO
// request sent. The four other filters are not validated locally.
func TestRunProposalList_UnsupportedStatusNoRequest(t *testing.T) {
	tr := &cannedTransport{status: 200, body: proposalsPageComplete}
	seam := &fakeProposalSeam{fakeMeSeam: &fakeMeSeam{ctx: validMeContext(), transport: tr}}

	outcome, _, stderr := runProposalListOver(t, seam, proposalsConfig{status: "open"})
	if outcome != UsageError || ExitCode(outcome) != 2 {
		t.Fatalf("outcome=%v exit=%d, want UsageError/2\nstderr: %s", outcome, ExitCode(outcome), stderr)
	}
	if tr.calls != 0 {
		t.Errorf("an unsupported --status must be rejected before any request, got %d calls", tr.calls)
	}
	if !strings.Contains(stderr, "open") || !strings.Contains(stderr, "draft") {
		t.Errorf("stderr should name the value and list the supported set:\n%s", stderr)
	}
}

// TestRunProposalList_BadOutputNoRequest pins an invalid --output is a UsageError(2)
// before any request (resolve-first ordering).
func TestRunProposalList_BadOutputNoRequest(t *testing.T) {
	tr := &cannedTransport{status: 200, body: proposalsPageComplete}
	seam := &fakeProposalSeam{fakeMeSeam: &fakeMeSeam{ctx: validMeContext(), transport: tr}}

	outcome, _, _ := runProposalListOver(t, seam, proposalsConfig{outputFlag: "xml", outputPresent: true})
	if outcome != UsageError || ExitCode(outcome) != 2 {
		t.Fatalf("outcome=%v exit=%d, want UsageError/2", outcome, ExitCode(outcome))
	}
	if tr.calls != 0 {
		t.Errorf("a bad --output must be rejected before any request, got %d calls", tr.calls)
	}
}

// TestRunProposalList_NoCredentialsIsUsageError pins that a missing token fails as a
// not-authenticated UsageError(2) with nothing printed.
func TestRunProposalList_NoCredentialsIsUsageError(t *testing.T) {
	tr := &cannedTransport{status: 200, body: proposalsPageComplete}
	seam := &fakeProposalSeam{fakeMeSeam: &fakeMeSeam{ctx: noTokenContext(), transport: tr}}

	outcome, stdout, stderr := runProposalListOver(t, seam, proposalsConfig{})
	if outcome != UsageError || ExitCode(outcome) != 2 {
		t.Fatalf("outcome=%v exit=%d, want UsageError/2\nstderr: %s", outcome, ExitCode(outcome), stderr)
	}
	if !strings.Contains(strings.ToLower(stderr), "not authenticated") {
		t.Errorf("stderr should report not authenticated:\n%s", stderr)
	}
	if strings.TrimSpace(stdout) != "" {
		t.Errorf("no proposal data should be printed on a credential failure:\n%s", stdout)
	}
}

// TestRunProposalList_StructuredEmitsAggregatedDocument pins -o json emits the
// aggregated {data:[…]} document, not the human projection.
func TestRunProposalList_StructuredEmitsAggregatedDocument(t *testing.T) {
	tr := &cannedTransport{status: 200, body: proposalsPageComplete}
	seam := &fakeProposalSeam{fakeMeSeam: &fakeMeSeam{ctx: validMeContext(), transport: tr}}

	outcome, stdout, _ := runProposalListOver(t, seam, proposalsConfig{outputFlag: "json", outputPresent: true})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success", outcome)
	}
	for _, want := range []string{`"data"`, "prp_1", "prp_2"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("structured json should carry the aggregated {data:[…]} document, missing %q:\n%s", want, stdout)
		}
	}
	if strings.Contains(stdout, "proposer:") {
		t.Errorf("structured json must not render the human projection:\n%s", stdout)
	}
}

// TestRunProposalList_FirstPageSignalsMore pins --first-page: one page, a "more
// proposals exist" stderr note, exit 0, exactly one request.
func TestRunProposalList_FirstPageSignalsMore(t *testing.T) {
	tr := &cannedTransport{status: 200, body: proposalsPage("prp_1", "draft", "c1")}
	seam := &fakeProposalSeam{fakeMeSeam: &fakeMeSeam{ctx: validMeContext(), transport: tr}}

	outcome, stdout, stderr := runProposalListOver(t, seam, proposalsConfig{firstPage: true})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success\nstderr: %s", outcome, stderr)
	}
	if tr.calls != 1 {
		t.Errorf("--first-page must not walk, want 1 call, got %d", tr.calls)
	}
	if !strings.Contains(stdout, "prp_1") {
		t.Errorf("the first page should print:\n%s", stdout)
	}
	if !strings.Contains(stderr, "more proposals exist") {
		t.Errorf("stderr should note more proposals exist:\n%s", stderr)
	}
}

// TestRunProposalList_MidWalkFailureIsPartialAndNonZero pins a mid-walk failure: the
// partial set is printed, an "incomplete" stderr note is written, and the command exits
// non-zero via the shared classifier.
func TestRunProposalList_MidWalkFailureIsPartialAndNonZero(t *testing.T) {
	tr := &seqMeTransport{steps: []seqMeResp{
		{status: 200, body: proposalsPage("prp_1", "draft", "c1")},
		{status: 500, body: `{"detail":"boom"}`},
	}}
	seam := &fakeProposalSeam{fakeMeSeam: &fakeMeSeam{ctx: validMeContext(), transport: tr}}

	outcome, stdout, stderr := runProposalListOver(t, seam, proposalsConfig{})
	if ExitCode(outcome) == 0 {
		t.Fatalf("a mid-walk failure should exit non-zero, got %v", outcome)
	}
	if !strings.Contains(stdout, "prp_1") {
		t.Errorf("the partial set gathered so far should print:\n%s", stdout)
	}
	if !strings.Contains(stderr, "incomplete") {
		t.Errorf("stderr should note the result is incomplete:\n%s", stderr)
	}
}

// TestProposalListCommand_RejectsPositional pins cobra.NoArgs: a positional to
// `proposal list` is a UsageError with no request sent (the list is global).
func TestProposalListCommand_RejectsPositional(t *testing.T) {
	tr := &cannedTransport{status: 200, body: proposalsPageComplete}
	seam := &fakeProposalSeam{fakeMeSeam: &fakeMeSeam{ctx: validMeContext(), transport: tr}}
	root := NewRootCommand()
	MustRegister(root, newProposalCommand(seam))
	root.SetOut(&bytes.Buffer{})
	root.SetErr(&bytes.Buffer{})

	outcome, _ := Run(root, []string{"proposal", "list", "role_0123"})
	if outcome != UsageError {
		t.Errorf("a positional to `proposal list` should be a UsageError, got %v", outcome)
	}
	if tr.calls != 0 {
		t.Errorf("a rejected positional must send no request, got %d calls", tr.calls)
	}
}

// --- T005: runProposalGet --------------------------------------------------

// TestRunProposalGet_PrintsFullDetail pins the single read: status, changes, response
// summary, and available transitions all printed, exit 0, GET /proposals/{id}.
func TestRunProposalGet_PrintsFullDetail(t *testing.T) {
	tr := &cannedTransport{status: 200, body: proposalDocumentBody}
	seam := &fakeProposalSeam{fakeMeSeam: &fakeMeSeam{ctx: validMeContext(), transport: tr}}

	outcome, stdout, stderr := runProposalGetOver(t, seam, proposalGetConfig{}, "prp_0123")
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success\nstderr: %s", outcome, stderr)
	}
	if !strings.HasSuffix(tr.lastPath, "/proposals/prp_0123") {
		t.Errorf("the request should target /proposals/{id}, got %q", tr.lastPath)
	}
	for _, want := range []string{
		"[proposed_outside_meeting]", // status
		"CreateRole",                 // change by type
		"2 total — 1 no-objection, 1 bring-to-meeting", // aggregate response summary
		"propose, withdraw", // available transitions
	} {
		if !strings.Contains(stdout, want) {
			t.Errorf("the single read should print %q:\n%s", want, stdout)
		}
	}
}

// TestRunProposalGet_UnknownIdSurfacesAPIStatus pins that an unknown/invisible id
// surfaces the API status (the id is not validated locally).
func TestRunProposalGet_UnknownIdSurfacesAPIStatus(t *testing.T) {
	tr := &cannedTransport{status: 404, body: `{"detail":"Proposal not found"}`}
	seam := &fakeProposalSeam{fakeMeSeam: &fakeMeSeam{ctx: validMeContext(), transport: tr}}

	outcome, stdout, stderr := runProposalGetOver(t, seam, proposalGetConfig{}, "prp_ffff")
	if outcome != APIError || ExitCode(outcome) != 3 {
		t.Fatalf("an unknown id should surface APIError/3, got %v/%d\nstderr: %s", outcome, ExitCode(outcome), stderr)
	}
	if !strings.Contains(stderr, "404") {
		t.Errorf("stderr should name the HTTP status (404):\n%s", stderr)
	}
	if strings.TrimSpace(stdout) != "" {
		t.Errorf("nothing should be printed on a not-found:\n%s", stdout)
	}
}

// TestRunProposalGet_StructuredEmitsRawPayload pins -o json emits the raw
// {data: Proposal} payload verbatim (faithful incl. all change keys), not the human
// projection.
func TestRunProposalGet_StructuredEmitsRawPayload(t *testing.T) {
	tr := &cannedTransport{status: 200, body: proposalDocumentBody}
	seam := &fakeProposalSeam{fakeMeSeam: &fakeMeSeam{ctx: validMeContext(), transport: tr}}

	outcome, stdout, _ := runProposalGetOver(t, seam, proposalGetConfig{outputFlag: "json", outputPresent: true}, "prp_0123")
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success", outcome)
	}
	for _, want := range []string{`"data"`, "prp_0123", `"available_transitions"`, "Scribe"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("structured json should carry the raw {data: Proposal} payload, missing %q:\n%s", want, stdout)
		}
	}
	if strings.Contains(stdout, "Transitions:") {
		t.Errorf("structured json must not render the human projection:\n%s", stdout)
	}
}

// TestProposalGetCommand_RejectsListFlags pins the structural list-only guard: each of
// the seven list flags on `get` is a cobra unknown-flag UsageError with NO request sent.
func TestProposalGetCommand_RejectsListFlags(t *testing.T) {
	listFlags := [][]string{
		{"--status", "draft"},
		{"--role-id", "role_0123"},
		{"--proposer-id", "per_0123"},
		{"--proposed-after", "2026-01-01T00:00:00Z"},
		{"--accepted-after", "2026-01-01T00:00:00Z"},
		{"--first-page"},
		{"--per-page", "10"},
	}
	for _, flag := range listFlags {
		tr := &cannedTransport{status: 200, body: proposalDocumentBody}
		seam := &fakeProposalSeam{fakeMeSeam: &fakeMeSeam{ctx: validMeContext(), transport: tr}}
		root := NewRootCommand()
		MustRegister(root, newProposalCommand(seam))
		root.SetOut(&bytes.Buffer{})
		root.SetErr(&bytes.Buffer{})

		args := append([]string{"proposal", "get", "prp_0123"}, flag...)
		outcome, _ := Run(root, args)
		if outcome != UsageError {
			t.Errorf("list flag %v on `get` should be a UsageError, got %v", flag, outcome)
		}
		if tr.calls != 0 {
			t.Errorf("a rejected list flag %v must send no request, got %d calls", flag, tr.calls)
		}
	}
}

// TestProposalGetCommand_RequiresExactlyOneArg pins ExactArgs(1): zero positionals (and
// more than one) is a usage error that sends no request.
func TestProposalGetCommand_RequiresExactlyOneArg(t *testing.T) {
	for _, args := range [][]string{
		{"proposal", "get"},                   // zero positional
		{"proposal", "get", "prp_1", "prp_2"}, // two positionals
	} {
		tr := &cannedTransport{status: 200, body: proposalDocumentBody}
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
