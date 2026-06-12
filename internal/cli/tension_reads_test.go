package cli

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
)

// --- canned GET /roles/{role_id}/tensions bodies (T003) --------------------
//
// They carry the Tension shape in the API's snake_case names, one tension with a
// null label and null role_id to exercise the explicit-absence markers, and the
// secret token nowhere.

const tensionsPageComplete = `{"data":[
  {"id":"ten_1","type":"tension","body":"We ship faster than we update the roadmap.","status":"unprocessed","role_id":"role_0123","sensed_by_id":"per_0123","label":"Roadmap drift","meeting_type":null,"parent_role_id":null},
  {"id":"ten_2","type":"tension","body":"Audit billing edge cases.","status":"processed","role_id":null,"sensed_by_id":"per_0123","label":null,"meeting_type":null,"parent_role_id":null}
],"meta":{"pagination":{"per_page":100,"has_next_page":false,"next_cursor":""}}}`

const tensionsPageEmpty = `{"data":[],"meta":{"pagination":{"per_page":100,"has_next_page":false,"next_cursor":""}}}`

// tensionDocumentBody is a representative GET /tensions/{id} body: the single-object
// {data: Tension} envelope carrying the full detail (status, body, sensing role,
// sensed-by, meeting type, timestamps).
const tensionDocumentBody = `{"data":{"id":"ten_0123","type":"tension","body":"We ship faster than we update the roadmap.","status":"unprocessed","role_id":"role_0123","sensed_by_id":"per_0123","created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-02T00:00:00Z","label":"Roadmap drift","meeting_type":"governance","parent_role_id":null}}`

// runTensionGetOver drives the pure runTensionGet over a fake seam, returning the
// outcome and captured stdout/stderr, and failing if the token leaks.
func runTensionGetOver(t *testing.T, seam tensionSeam, cfg tensionGetConfig, id string) (Outcome, string, string) {
	t.Helper()
	var out, errb bytes.Buffer
	cfg.seam = seam
	cfg.reqCtx = context.Background()
	cfg.stdout = &out
	cfg.stderr = &errb
	outcome, _ := runTensionGet(cfg, id)
	if strings.Contains(out.String()+errb.String(), meSecretToken) {
		t.Fatalf("the token leaked into output: %q", out.String()+errb.String())
	}
	return outcome, out.String(), errb.String()
}

// tensionsPage builds a one-tension page; a non-empty nextCursor marks more pages.
func tensionsPage(id, label, nextCursor string) string {
	hasNext := "false"
	if nextCursor != "" {
		hasNext = "true"
	}
	return `{"data":[{"id":"` + id + `","type":"tension","body":"a tension","status":"unprocessed","role_id":"role_0123","sensed_by_id":"per_0123","label":"` + label + `","meeting_type":null,"parent_role_id":null}],` +
		`"meta":{"pagination":{"per_page":1,"has_next_page":` + hasNext + `,"next_cursor":"` + nextCursor + `"}}}`
}

// runTensionListOver drives the pure runTensionList over a fake seam, returning the
// outcome and captured stdout/stderr, and failing if the token leaks.
func runTensionListOver(t *testing.T, seam tensionSeam, cfg tensionsConfig) (Outcome, string, string) {
	t.Helper()
	var out, errb bytes.Buffer
	cfg.seam = seam
	cfg.reqCtx = context.Background()
	cfg.stdout = &out
	cfg.stderr = &errb
	outcome, _ := runTensionList(cfg)
	if strings.Contains(out.String()+errb.String(), meSecretToken) {
		t.Fatalf("the token leaked into output: %q", out.String()+errb.String())
	}
	return outcome, out.String(), errb.String()
}

// --- validateTensionStatus (pure, T002) ------------------------------------
//
// A NEW validator over the tension status set (unprocessed/processed/archived),
// distinct from the action/project validateStatus set — reusing that would accept
// invalid tension statuses and reject valid ones (plan ADR-3).

func TestValidateTensionStatus(t *testing.T) {
	if err := validateTensionStatus(""); err != nil {
		t.Errorf("an absent --status should be valid (no filter), got %v", err)
	}
	for _, ok := range []string{"unprocessed", "processed", "archived"} {
		if err := validateTensionStatus(ok); err != nil {
			t.Errorf("%q should be valid, got %v", ok, err)
		}
	}
	err := validateTensionStatus("open")
	if err == nil {
		t.Fatal("an unsupported --status should be rejected")
	}
	if !strings.Contains(err.Error(), "open") {
		t.Errorf("the error should name the unsupported value:\n%v", err)
	}
	for _, want := range []string{"archived", "processed", "unprocessed"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the error should list the supported set (missing %q):\n%v", want, err)
		}
	}
}

// TestValidateTensionStatus_RejectsActionVocabulary pins that the action/project
// statuses are NOT accepted here — the new set is distinct (plan ADR-3).
func TestValidateTensionStatus_RejectsActionVocabulary(t *testing.T) {
	for _, wrong := range []string{"current", "completed"} {
		if err := validateTensionStatus(wrong); err == nil {
			t.Errorf("%q is an action/project status and must be rejected for tensions", wrong)
		}
	}
}

// TestSupportedTensionStatusNames_Sorted pins the deterministic sorted order used
// in the usage message.
func TestSupportedTensionStatusNames_Sorted(t *testing.T) {
	got := supportedTensionStatusNames()
	want := []string{"archived", "processed", "unprocessed"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v (sorted)", got, want)
		}
	}
}

// --- tension list walk branches (T003) -------------------------------------

func TestRunTensionList_SuccessWalksAndProjects(t *testing.T) {
	tr := &cannedTransport{status: 200, body: tensionsPageComplete}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, stderr := runTensionListOver(t, seam, tensionsConfig{id: "role_0123"})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success\nstderr: %s", outcome, stderr)
	}
	for _, want := range []string{
		"ten_1  [unprocessed]  Roadmap drift",
		"We ship faster than we update the roadmap.",
		"ten_2  [processed]",
		"sensing role: —", // ten_2 has null role_id → the absence marker
	} {
		if !strings.Contains(stdout, want) {
			t.Errorf("stdout missing %q:\n%s", want, stdout)
		}
	}
	if strings.TrimSpace(stderr) != "" {
		t.Errorf("a complete success should write nothing to stderr, got %q", stderr)
	}
	if tr.calls != 1 {
		t.Errorf("a single complete page should be one call, got %d", tr.calls)
	}
	if got := tr.lastPath; !strings.HasSuffix(got, "/roles/role_0123/tensions") {
		t.Errorf("path = %q, want it to target /roles/role_0123/tensions", got)
	}
}

func TestRunTensionList_EmptyIsCleanSuccess(t *testing.T) {
	tr := &cannedTransport{status: 200, body: tensionsPageEmpty}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, stderr := runTensionListOver(t, seam, tensionsConfig{id: "role_0123"})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success", outcome)
	}
	if strings.TrimRight(stdout, "\n") != "no tensions" {
		t.Errorf("a role with no tensions should print exactly `no tensions`, got %q", stdout)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Errorf("an empty list is a clean success; stderr should be empty, got %q", stderr)
	}
}

func TestRunTensionList_WalksEveryPageToCompletion(t *testing.T) {
	tr := &seqMeTransport{steps: []seqMeResp{
		{status: 200, body: tensionsPage("ten_1", "Page One", "c1")},
		{status: 200, body: tensionsPage("ten_2", "Page Two", "c2")},
		{status: 200, body: tensionsPage("ten_3", "Page Three", "")},
	}}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, _ := runTensionListOver(t, seam, tensionsConfig{id: "role_0123"})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success", outcome)
	}
	if tr.calls != 3 {
		t.Errorf("the walk should issue three page requests, got %d", tr.calls)
	}
	for _, want := range []string{"ten_1", "ten_2", "ten_3", "Page One", "Page Two", "Page Three"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("every page's tensions should print, missing %q:\n%s", want, stdout)
		}
	}
}

// --- --status filter --------------------------------------------------------

func TestRunTensionList_StatusSentWhenSupplied(t *testing.T) {
	tr := &cannedTransport{status: 200, body: tensionsPageComplete}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	_, _, _ = runTensionListOver(t, seam, tensionsConfig{id: "role_0123", status: "unprocessed"})
	if got := tr.lastQuery.Get("status"); got != "unprocessed" {
		t.Errorf("status = %q, want \"unprocessed\"", got)
	}
}

func TestRunTensionList_OmittedOrEmptyStatusSendsNothing(t *testing.T) {
	// Omitted (status="").
	tr1 := &cannedTransport{status: 200, body: tensionsPageComplete}
	seam1 := &fakeMeSeam{ctx: validMeContext(), transport: tr1}
	_, _, _ = runTensionListOver(t, seam1, tensionsConfig{id: "role_0123"})
	if _, present := tr1.lastQuery["status"]; present {
		t.Errorf("an omitted --status must not send the param, got %v", tr1.lastQuery)
	}

	// Present but empty behaves as no filter.
	tr2 := &cannedTransport{status: 200, body: tensionsPageComplete}
	seam2 := &fakeMeSeam{ctx: validMeContext(), transport: tr2}
	_, _, _ = runTensionListOver(t, seam2, tensionsConfig{id: "role_0123", status: ""})
	if _, present := tr2.lastQuery["status"]; present {
		t.Errorf("an empty --status must behave as no filter, got %v", tr2.lastQuery)
	}
}

// --- --status validation (the one closed-enum input) -----------------------

func TestRunTensionList_UnsupportedStatusIsUsageErrorNoRequest(t *testing.T) {
	tr := &cannedTransport{status: 200, body: tensionsPageComplete}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, _, stderr := runTensionListOver(t, seam, tensionsConfig{id: "role_0123", status: "open"})
	if outcome != UsageError || ExitCode(outcome) != 2 {
		t.Fatalf("outcome=%v exit=%d, want UsageError/2", outcome, ExitCode(outcome))
	}
	if !strings.Contains(stderr, "open") {
		t.Errorf("stderr should name the unsupported value:\n%s", stderr)
	}
	for _, want := range []string{"archived", "processed", "unprocessed"} {
		if !strings.Contains(stderr, want) {
			t.Errorf("stderr should list the supported set (missing %q):\n%s", want, stderr)
		}
	}
	if tr.calls != 0 {
		t.Errorf("an unsupported --status must be rejected before any request, got %d calls", tr.calls)
	}
}

// --- --first-page -----------------------------------------------------------

func TestRunTensionList_FirstPageStopsAndSignals(t *testing.T) {
	tr := &cannedTransport{status: 200, body: tensionsPage("ten_1", "First Page", "c1")}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, stderr := runTensionListOver(t, seam, tensionsConfig{id: "role_0123", firstPage: true})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success", outcome)
	}
	if !strings.Contains(stdout, "ten_1") {
		t.Errorf("the first page should print:\n%s", stdout)
	}
	if tr.calls != 1 {
		t.Errorf("--first-page must not walk, want 1 call, got %d", tr.calls)
	}
	if !strings.Contains(stderr, "more tensions exist") {
		t.Errorf("stderr should note more tensions exist:\n%s", stderr)
	}
}

func TestRunTensionList_PerPageSizesWalk(t *testing.T) {
	tr := &cannedTransport{status: 200, body: tensionsPageComplete}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	_, _, _ = runTensionListOver(t, seam, tensionsConfig{id: "role_0123", perPage: 7, perPageSet: true})
	if got := tr.lastQuery.Get("per_page"); got != "7" {
		t.Errorf("per_page = %q, want \"7\" (WithPageSize passed through)", got)
	}
}

// --- mid-walk failure -------------------------------------------------------

func TestRunTensionList_MidWalkFailurePartialAndIncomplete(t *testing.T) {
	tr := &seqMeTransport{steps: []seqMeResp{
		{status: 200, body: tensionsPage("ten_1", "Gathered", "c1")},
		{status: 500, body: `{"detail":"boom"}`},
	}}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, stderr := runTensionListOver(t, seam, tensionsConfig{id: "role_0123"})
	if outcome == Success {
		t.Fatalf("a mid-walk failure must be non-zero, got Success")
	}
	if ExitCode(outcome) == 0 {
		t.Errorf("a mid-walk failure must exit non-zero, got exit 0 (outcome %v)", outcome)
	}
	if !strings.Contains(stdout, "ten_1") {
		t.Errorf("the partial set gathered so far should print:\n%s", stdout)
	}
	if !strings.Contains(stderr, "incomplete") {
		t.Errorf("stderr should note the result is incomplete and name the cause:\n%s", stderr)
	}
}

// --- error classification (via the shared classifier) ----------------------

func TestRunTensionList_NoCredentialsIsUsageError(t *testing.T) {
	tr := &cannedTransport{status: 200, body: tensionsPageComplete}
	seam := &fakeMeSeam{ctx: noTokenContext(), transport: tr}

	outcome, stdout, stderr := runTensionListOver(t, seam, tensionsConfig{id: "role_0123"})
	if outcome != UsageError || ExitCode(outcome) != 2 {
		t.Fatalf("outcome=%v exit=%d, want UsageError/2", outcome, ExitCode(outcome))
	}
	if !strings.Contains(strings.ToLower(stderr), "not authenticated") {
		t.Errorf("stderr should report not authenticated:\n%s", stderr)
	}
	if strings.TrimSpace(stdout) != "" {
		t.Errorf("no tension data should be printed on a credential failure, got:\n%s", stdout)
	}
}

func TestRunTensionList_TransportErrorIsNetworkUnavailable(t *testing.T) {
	tr := &cannedTransport{netErr: errors.New("dial tcp: connection refused")}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, _, stderr := runTensionListOver(t, seam, tensionsConfig{id: "role_0123"})
	if outcome != NetworkUnavailable || ExitCode(outcome) != 6 {
		t.Fatalf("outcome=%v exit=%d, want NetworkUnavailable/6\nstderr: %s", outcome, ExitCode(outcome), stderr)
	}
}

func TestRunTensionList_Non2xxClassifies(t *testing.T) {
	cases := []struct {
		status int
		want   Outcome
		code   int
	}{
		{403, PermissionError, 4},
		{429, RateLimited, 5},
		{500, APIError, 3},
	}
	for _, c := range cases {
		tr := &cannedTransport{status: c.status, body: `{"detail":"x"}`}
		seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}
		outcome, _, stderr := runTensionListOver(t, seam, tensionsConfig{id: "role_0123"})
		if outcome != c.want || ExitCode(outcome) != c.code {
			t.Errorf("status %d: outcome=%v exit=%d, want %v/%d\nstderr: %s", c.status, outcome, ExitCode(outcome), c.want, c.code, stderr)
		}
	}
}

// --- resolve-before-call: a bad --output costs no request ------------------

func TestRunTensionList_BadOutputIsUsageErrorNoRequest(t *testing.T) {
	tr := &cannedTransport{status: 200, body: tensionsPageComplete}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, _, stderr := runTensionListOver(t, seam, tensionsConfig{id: "role_0123", outputFlag: "xml"})
	if outcome != UsageError || ExitCode(outcome) != 2 {
		t.Fatalf("outcome=%v exit=%d, want UsageError/2\nstderr: %s", outcome, ExitCode(outcome), stderr)
	}
	if tr.calls != 0 {
		t.Errorf("a bad --output must be rejected before any request, got %d calls", tr.calls)
	}
}

// --- structured output emits the aggregated raw document --------------------

func TestRunTensionList_StructuredJSONEmitsAggregatedRawPayload(t *testing.T) {
	tr := &cannedTransport{status: 200, body: tensionsPageComplete}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, _ := runTensionListOver(t, seam, tensionsConfig{id: "role_0123", outputFlag: "json"})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success", outcome)
	}
	for _, want := range []string{`"data"`, "ten_1", `"role_id"`, `"sensed_by_id"`} {
		if !strings.Contains(stdout, want) {
			t.Errorf("structured json should carry the raw payload, missing %q:\n%s", want, stdout)
		}
	}
	// Structured output must not carry the human projection's block labels nor the
	// per-page meta envelope.
	if strings.Contains(stdout, "sensing role:") {
		t.Errorf("structured json must not render the human projection:\n%s", stdout)
	}
	if strings.Contains(stdout, `"pagination"`) {
		t.Errorf("the aggregated document must drop the per-page meta envelope:\n%s", stdout)
	}
}

// --- command-level wiring (T003) -------------------------------------------

// TestTensionListCommand_StatusSendsParam pins the end-to-end --status send under
// the `tension` group: a real `tension list <id> --status unprocessed` sends it.
func TestTensionListCommand_StatusSendsParam(t *testing.T) {
	tr := &cannedTransport{status: 200, body: tensionsPageComplete}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	root := NewRootCommand()
	MustRegister(root, newTensionCommand(seam))
	var out, errb bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&errb)
	outcome, _ := Run(root, []string{"tension", "list", "role_0123", "--status", "unprocessed"})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success\nstderr: %s", outcome, errb.String())
	}
	if got := tr.lastQuery.Get("status"); got != "unprocessed" {
		t.Errorf("status = %q, want \"unprocessed\"", got)
	}
}

// TestTensionListCommand_UnsupportedStatusNoRequest pins fail-fast --status
// validation at the command level under the `tension` group.
func TestTensionListCommand_UnsupportedStatusNoRequest(t *testing.T) {
	tr := &cannedTransport{status: 200, body: tensionsPageComplete}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	root := NewRootCommand()
	MustRegister(root, newTensionCommand(seam))
	root.SetOut(&bytes.Buffer{})
	root.SetErr(&bytes.Buffer{})
	outcome, _ := Run(root, []string{"tension", "list", "role_0123", "--status", "open"})
	if outcome != UsageError {
		t.Errorf("an unsupported --status should be a UsageError, got %v", outcome)
	}
	if tr.calls != 0 {
		t.Errorf("an unsupported --status must send no request, got %d calls", tr.calls)
	}
}

// TestTensionListCommand_RequiresExactlyOneArg pins ExactArgs(1): zero args is a
// usage error and sends no request.
func TestTensionListCommand_RequiresExactlyOneArg(t *testing.T) {
	tr := &cannedTransport{status: 200, body: tensionsPageComplete}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	root := NewRootCommand()
	MustRegister(root, newTensionCommand(seam))
	root.SetOut(&bytes.Buffer{})
	root.SetErr(&bytes.Buffer{})
	outcome, _ := Run(root, []string{"tension", "list"})
	if outcome != UsageError {
		t.Errorf("zero args should be a UsageError, got %v", outcome)
	}
	if tr.calls != 0 {
		t.Errorf("a wrong arg count must send no request, got %d calls", tr.calls)
	}
}

// --- single tension read (T004) --------------------------------------------

func TestRunTensionGet_SingleReadPrintsDetail(t *testing.T) {
	tr := &cannedTransport{status: 200, body: tensionDocumentBody}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, stderr := runTensionGetOver(t, seam, tensionGetConfig{}, "ten_0123")
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success\nstderr: %s", outcome, stderr)
	}
	for _, want := range []string{
		"ten_0123  [unprocessed]",                    // header: id + status
		"We ship faster than we update the roadmap.", // body, verbatim
		"role_0123",            // sensing role
		"per_0123",             // sensed by
		"governance",           // meeting type
		"2026-01-01T00:00:00Z", // created
	} {
		if !strings.Contains(stdout, want) {
			t.Errorf("the single tension should show %q:\n%s", want, stdout)
		}
	}
	if tr.calls != 1 {
		t.Errorf("a single read is one call, got %d", tr.calls)
	}
	if got := tr.lastPath; !strings.HasSuffix(got, "/tensions/ten_0123") {
		t.Errorf("path = %q, want it to target /tensions/ten_0123", got)
	}
}

func TestRunTensionGet_UnknownIdSurfacesAPIStatus(t *testing.T) {
	tr := &cannedTransport{status: 404, body: `{"detail":"Tension not found"}`}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, stderr := runTensionGetOver(t, seam, tensionGetConfig{}, "ten_ffff")
	if outcome != APIError || ExitCode(outcome) != 3 {
		t.Fatalf("an unknown id should surface APIError/3, got %v/%d\nstderr: %s", outcome, ExitCode(outcome), stderr)
	}
	if !strings.Contains(stderr, "404") {
		t.Errorf("stderr should name the HTTP status (404):\n%s", stderr)
	}
	if strings.TrimSpace(stdout) != "" {
		t.Errorf("nothing should be printed to stdout on a not-found, got:\n%s", stdout)
	}
}

func TestRunTensionGet_StructuredJSONEmitsRawPayload(t *testing.T) {
	tr := &cannedTransport{status: 200, body: tensionDocumentBody}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, _ := runTensionGetOver(t, seam, tensionGetConfig{outputFlag: "json"}, "ten_0123")
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success", outcome)
	}
	for _, want := range []string{`"data"`, "ten_0123", `"role_id"`, `"updated_at"`} {
		if !strings.Contains(stdout, want) {
			t.Errorf("structured json should carry the raw single-tension payload, missing %q:\n%s", want, stdout)
		}
	}
	// The single read emits the raw {data: Tension} body, not the human projection.
	if strings.Contains(stdout, "Sensing role:") {
		t.Errorf("structured json must not render the human projection:\n%s", stdout)
	}
}

// TestTensionGetCommand_ListFlagRejectedNoRequest pins the structural list-only
// guard (ADR-1): a list-only flag on `get` is a cobra unknown-flag UsageError before
// any request — the transport tripwire confirms nothing is sent.
func TestTensionGetCommand_ListFlagRejectedNoRequest(t *testing.T) {
	for _, flag := range []string{"--status", "--first-page", "--per-page"} {
		tr := &cannedTransport{status: 200, body: tensionDocumentBody}
		seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

		root := NewRootCommand()
		MustRegister(root, newTensionCommand(seam))
		var out, errb bytes.Buffer
		root.SetOut(&out)
		root.SetErr(&errb)
		// The value-taking flags need a value so cobra's failure is unknown-flag, not
		// missing-value; --first-page is a bool.
		args := []string{"tension", "get", "ten_0123", flag}
		if flag != "--first-page" {
			args = append(args, "x")
		}
		outcome, _ := Run(root, args)
		if outcome != UsageError {
			t.Errorf("%s on `tension get` should be a UsageError, got %v", flag, outcome)
		}
		if tr.calls != 0 {
			t.Errorf("%s on `tension get` must send no request, got %d calls", flag, tr.calls)
		}
	}
}

// TestTensionGetCommand_RequiresExactlyOneArg pins ExactArgs(1) on the single read.
func TestTensionGetCommand_RequiresExactlyOneArg(t *testing.T) {
	tr := &cannedTransport{status: 200, body: tensionDocumentBody}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	root := NewRootCommand()
	MustRegister(root, newTensionCommand(seam))
	root.SetOut(&bytes.Buffer{})
	root.SetErr(&bytes.Buffer{})
	outcome, _ := Run(root, []string{"tension", "get"})
	if outcome != UsageError {
		t.Errorf("zero args should be a UsageError, got %v", outcome)
	}
	if tr.calls != 0 {
		t.Errorf("a wrong arg count must send no request, got %d calls", tr.calls)
	}
}
