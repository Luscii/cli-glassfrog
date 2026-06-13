package cli

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
)

// --- canned GET /roles/{id}/assignments bodies -----------------------------
//
// They carry the Assignment shape (id/actor_id/role_id/focus/elected_until + the
// embedded actor{id,name,kind} from the default include) in the API's snake_case
// names, a person and an agent to exercise both id prefixes, and the secret token
// nowhere.

const fillersPageComplete = `{"data":[
  {"id":"asgn_1","actor_id":"per_0123","role_id":"role_0123","focus":"Keep the lights on","elected_until":"2026-12-31","actor":{"id":"per_0123","name":"Alice Smith","kind":"human"}},
  {"id":"asgn_2","actor_id":"agt_0456","role_id":"role_0123","focus":"","elected_until":"","actor":{"id":"agt_0456","name":"Claude","kind":"agent"}}
],"meta":{"pagination":{"per_page":100,"has_next_page":false,"next_cursor":""}}}`

const fillersPageEmpty = `{"data":[],"meta":{"pagination":{"per_page":100,"has_next_page":false,"next_cursor":""}}}`

// fillersPage builds a one-filler page; a non-empty nextCursor marks more pages.
func fillersPage(actorID, name, kind, nextCursor string) string {
	hasNext := "false"
	if nextCursor != "" {
		hasNext = "true"
	}
	return `{"data":[{"id":"asgn_x","actor_id":"` + actorID + `","role_id":"role_0123","focus":"","elected_until":"",` +
		`"actor":{"id":"` + actorID + `","name":"` + name + `","kind":"` + kind + `"}}],` +
		`"meta":{"pagination":{"per_page":1,"has_next_page":` + hasNext + `,"next_cursor":"` + nextCursor + `"}}}`
}

// runFillersOver drives the pure runFillersList over a fake seam, returning the
// outcome and captured stdout/stderr, and failing if the token leaks.
func runFillersOver(t *testing.T, seam fillersSeam, cfg fillersConfig) (Outcome, string, string) {
	t.Helper()
	var out, errb bytes.Buffer
	cfg.seam = seam
	cfg.reqCtx = context.Background()
	cfg.stdout = &out
	cfg.stderr = &errb
	outcome, _ := runFillersList(cfg)
	if strings.Contains(out.String()+errb.String(), meSecretToken) {
		t.Fatalf("the token leaked into output: %q", out.String()+errb.String())
	}
	return outcome, out.String(), errb.String()
}

// --- list walk branches ----------------------------------------------------

func TestRunFillers_ListSuccessWalksAndProjects(t *testing.T) {
	tr := &cannedTransport{status: 200, body: fillersPageComplete}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, stderr := runFillersOver(t, seam, fillersConfig{id: "role_0123"})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success\nstderr: %s", outcome, stderr)
	}
	for _, want := range []string{
		"per_0123  [human]", // person row leads with the filling actor + kind
		"Alice Smith",       // the actor name from the default include
		"agt_0456  [agent]", // agent row
		"Claude",
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
	if got := tr.lastPath; !strings.HasSuffix(got, "/roles/role_0123/assignments") {
		t.Errorf("path = %q, want it to target /roles/role_0123/assignments", got)
	}
}

// The actor's name + kind come from the endpoint's default include — the command
// declares no --include flag (plan ADR-3).
func TestRunFillers_NameAndKindShownWithoutIncludeFlag(t *testing.T) {
	tr := &cannedTransport{status: 200, body: fillersPageComplete}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	_, stdout, _ := runFillersOver(t, seam, fillersConfig{id: "role_0123"})
	if !strings.Contains(stdout, "Alice Smith") || !strings.Contains(stdout, "[human]") {
		t.Errorf("the filler's name and kind should be shown from the default include:\n%s", stdout)
	}
	// No filter/include query param is sent — the request is the bare default.
	for _, param := range []string{"include", "kind", "q", "role_id"} {
		if _, present := tr.lastQuery[param]; present {
			t.Errorf("the command sends no %q param (no filters / no --include), got %v", param, tr.lastQuery)
		}
	}
}

// A person and an agent filler are distinguished by id prefix and kind badge.
func TestRunFillers_PersonAndAgentDistinguished(t *testing.T) {
	tr := &cannedTransport{status: 200, body: fillersPageComplete}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	_, stdout, _ := runFillersOver(t, seam, fillersConfig{id: "role_0123"})
	if !strings.Contains(stdout, "per_0123") || !strings.Contains(stdout, "[human]") {
		t.Errorf("the person filler should be distinguished by per_ id + [human] badge:\n%s", stdout)
	}
	if !strings.Contains(stdout, "agt_0456") || !strings.Contains(stdout, "[agent]") {
		t.Errorf("the agent filler should be distinguished by agt_ id + [agent] badge:\n%s", stdout)
	}
}

// A filler's focus and election expiry are projected under the default human format.
func TestRunFillers_FocusAndElectionProjected(t *testing.T) {
	tr := &cannedTransport{status: 200, body: fillersPageComplete}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	_, stdout, _ := runFillersOver(t, seam, fillersConfig{id: "role_0123"})
	for _, want := range []string{"Keep the lights on", "2026-12-31"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("focus/election must be projected, missing %q:\n%s", want, stdout)
		}
	}
	// The focus-less / non-elected agent shows the explicit-absence markers.
	if !strings.Contains(stdout, "(none)") || !strings.Contains(stdout, "(not an elected seat)") {
		t.Errorf("an absent focus/election must show explicit-absence markers:\n%s", stdout)
	}
}

func TestRunFillers_EmptyIsCleanSuccess(t *testing.T) {
	tr := &cannedTransport{status: 200, body: fillersPageEmpty}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, stderr := runFillersOver(t, seam, fillersConfig{id: "role_0123"})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success", outcome)
	}
	if strings.TrimRight(stdout, "\n") != "no fillers" {
		t.Errorf("a role with no fillers should print exactly `no fillers`, got %q", stdout)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Errorf("an empty list is a clean success; stderr should be empty, got %q", stderr)
	}
}

func TestRunFillers_WalksEveryPageToCompletion(t *testing.T) {
	tr := &seqMeTransport{steps: []seqMeResp{
		{status: 200, body: fillersPage("per_1", "Page One", "human", "c1")},
		{status: 200, body: fillersPage("per_2", "Page Two", "human", "c2")},
		{status: 200, body: fillersPage("per_3", "Page Three", "human", "")},
	}}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, _ := runFillersOver(t, seam, fillersConfig{id: "role_0123"})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success", outcome)
	}
	if tr.calls != 3 {
		t.Errorf("the walk should issue three page requests, got %d", tr.calls)
	}
	for _, want := range []string{"Page One", "Page Two", "Page Three"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("every page's fillers should print, missing %q:\n%s", want, stdout)
		}
	}
}

// --- --first-page ----------------------------------------------------------

func TestRunFillers_FirstPageStopsAndSignals(t *testing.T) {
	tr := &cannedTransport{status: 200, body: fillersPage("per_1", "First Page Filler", "human", "c1")}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, stderr := runFillersOver(t, seam, fillersConfig{id: "role_0123", firstPage: true})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success", outcome)
	}
	if !strings.Contains(stdout, "First Page Filler") {
		t.Errorf("the first page should print:\n%s", stdout)
	}
	if tr.calls != 1 {
		t.Errorf("--first-page must not walk, want 1 call, got %d", tr.calls)
	}
	if !strings.Contains(stderr, "more fillers exist") {
		t.Errorf("stderr should note more fillers exist:\n%s", stderr)
	}
}

func TestRunFillers_PerPageSizesWalk(t *testing.T) {
	tr := &cannedTransport{status: 200, body: fillersPageComplete}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	_, _, _ = runFillersOver(t, seam, fillersConfig{id: "role_0123", perPage: 7, perPageSet: true})
	if got := tr.lastQuery.Get("per_page"); got != "7" {
		t.Errorf("per_page = %q, want \"7\" (WithPageSize passed through)", got)
	}
}

// --- mid-walk failure ------------------------------------------------------

func TestRunFillers_MidWalkFailurePartialAndIncomplete(t *testing.T) {
	tr := &seqMeTransport{steps: []seqMeResp{
		{status: 200, body: fillersPage("per_1", "Gathered Filler", "human", "c1")},
		{status: 500, body: `{"detail":"boom"}`},
	}}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, stderr := runFillersOver(t, seam, fillersConfig{id: "role_0123"})
	if outcome == Success {
		t.Fatalf("a mid-walk failure must be non-zero, got Success")
	}
	if ExitCode(outcome) == 0 {
		t.Errorf("a mid-walk failure must exit non-zero, got exit 0 (outcome %v)", outcome)
	}
	if !strings.Contains(stdout, "Gathered Filler") {
		t.Errorf("the partial set gathered so far should print:\n%s", stdout)
	}
	if !strings.Contains(stderr, "incomplete") {
		t.Errorf("stderr should note the result is incomplete and name the cause:\n%s", stderr)
	}
}

// --- error classification (via the shared classifier) ----------------------

func TestRunFillers_NoCredentialsIsUsageError(t *testing.T) {
	tr := &cannedTransport{status: 200, body: fillersPageComplete}
	seam := &fakeMeSeam{ctx: noTokenContext(), transport: tr}

	outcome, stdout, stderr := runFillersOver(t, seam, fillersConfig{id: "role_0123"})
	if outcome != UsageError || ExitCode(outcome) != 2 {
		t.Fatalf("outcome=%v exit=%d, want UsageError/2", outcome, ExitCode(outcome))
	}
	if !strings.Contains(strings.ToLower(stderr), "not authenticated") {
		t.Errorf("stderr should report not authenticated:\n%s", stderr)
	}
	if strings.TrimSpace(stdout) != "" {
		t.Errorf("no filler data should be printed on a credential failure, got:\n%s", stdout)
	}
}

func TestRunFillers_UnknownRoleSurfacesAPIStatus(t *testing.T) {
	tr := &cannedTransport{status: 404, body: `{"detail":"Role not found"}`}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, stderr := runFillersOver(t, seam, fillersConfig{id: "role_ffff"})
	if outcome != APIError || ExitCode(outcome) != 3 {
		t.Fatalf("an unknown role id should surface APIError/3, got %v/%d\nstderr: %s", outcome, ExitCode(outcome), stderr)
	}
	if !strings.Contains(stderr, "404") {
		t.Errorf("stderr should name the HTTP status (404):\n%s", stderr)
	}
	if strings.TrimSpace(stdout) != "" {
		t.Errorf("nothing should be printed to stdout on a not-found, got:\n%s", stdout)
	}
}

func TestRunFillers_TransportErrorIsNetworkUnavailable(t *testing.T) {
	tr := &cannedTransport{netErr: errors.New("dial tcp: connection refused")}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, _, stderr := runFillersOver(t, seam, fillersConfig{id: "role_0123"})
	if outcome != NetworkUnavailable || ExitCode(outcome) != 6 {
		t.Fatalf("outcome=%v exit=%d, want NetworkUnavailable/6\nstderr: %s", outcome, ExitCode(outcome), stderr)
	}
}

func TestRunFillers_Non2xxClassifies(t *testing.T) {
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
		outcome, _, stderr := runFillersOver(t, seam, fillersConfig{id: "role_0123"})
		if outcome != c.want || ExitCode(outcome) != c.code {
			t.Errorf("status %d: outcome=%v exit=%d, want %v/%d\nstderr: %s", c.status, outcome, ExitCode(outcome), c.want, c.code, stderr)
		}
	}
}

// --- resolve-before-call: a bad --output costs no request ------------------

func TestRunFillers_BadOutputIsUsageErrorNoRequest(t *testing.T) {
	tr := &cannedTransport{status: 200, body: fillersPageComplete}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, _, stderr := runFillersOver(t, seam, fillersConfig{id: "role_0123", outputFlag: "xml", outputPresent: true})
	if outcome != UsageError || ExitCode(outcome) != 2 {
		t.Fatalf("outcome=%v exit=%d, want UsageError/2\nstderr: %s", outcome, ExitCode(outcome), stderr)
	}
	if tr.calls != 0 {
		t.Errorf("a bad --output must be rejected before any request, got %d calls", tr.calls)
	}
}

// --- structured output emits the aggregated raw document --------------------

func TestRunFillers_StructuredJSONEmitsAggregatedRawPayload(t *testing.T) {
	tr := &cannedTransport{status: 200, body: fillersPageComplete}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, _ := runFillersOver(t, seam, fillersConfig{id: "role_0123", outputFlag: "json", outputPresent: true})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success", outcome)
	}
	for _, want := range []string{`"data"`, "asgn_1", `"actor_id"`, `"elected_until"`, `"focus"`} {
		if !strings.Contains(stdout, want) {
			t.Errorf("structured json should carry the raw payload, missing %q:\n%s", want, stdout)
		}
	}
	// Structured output must not carry the human projection's block labels nor the
	// per-page meta envelope.
	if strings.Contains(stdout, "Elected until:") {
		t.Errorf("structured json must not render the human projection:\n%s", stdout)
	}
	if strings.Contains(stdout, `"pagination"`) {
		t.Errorf("the aggregated document must drop the per-page meta envelope:\n%s", stdout)
	}
}

// --- command-level wiring --------------------------------------------------

// TestFillersCommand_TargetsAssignmentsEndpoint pins the end-to-end path: a real
// `fillers role_0123` invocation reads /roles/{id}/assignments.
func TestFillersCommand_TargetsAssignmentsEndpoint(t *testing.T) {
	tr := &cannedTransport{status: 200, body: fillersPageComplete}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	root := NewRootCommand()
	MustRegister(root, newFillersCommand(seam))
	var out, errb bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&errb)
	outcome, _ := Run(root, []string{"fillers", "role_0123"})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success\nstderr: %s", outcome, errb.String())
	}
	if got := tr.lastPath; !strings.HasSuffix(got, "/roles/role_0123/assignments") {
		t.Errorf("path = %q, want it to target /roles/role_0123/assignments", got)
	}
}

// TestFillersCommand_RequiresExactlyOneArg pins ExactArgs(1): zero args is a usage
// error and sends no request (the transport tripwire).
func TestFillersCommand_RequiresExactlyOneArg(t *testing.T) {
	for _, args := range [][]string{{"fillers"}, {"fillers", "role_0123", "role_0456"}} {
		tr := &cannedTransport{status: 200, body: fillersPageComplete}
		seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

		root := NewRootCommand()
		MustRegister(root, newFillersCommand(seam))
		root.SetOut(&bytes.Buffer{})
		root.SetErr(&bytes.Buffer{})
		outcome, _ := Run(root, args)
		if outcome != UsageError {
			t.Errorf("%v should be a UsageError, got %v", args, outcome)
		}
		if tr.calls != 0 {
			t.Errorf("%v must send no request, got %d calls", args, tr.calls)
		}
	}
}

// TestFillersCommand_UnknownFlagRejectedNoRequest pins that the command declares no
// filter flags / no --include: an unknown flag is a cobra usage error before any
// request (plan ADR-3).
func TestFillersCommand_UnknownFlagRejectedNoRequest(t *testing.T) {
	for _, flag := range []string{"--include", "--kind", "--status", "--query"} {
		tr := &cannedTransport{status: 200, body: fillersPageComplete}
		seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

		root := NewRootCommand()
		MustRegister(root, newFillersCommand(seam))
		root.SetOut(&bytes.Buffer{})
		root.SetErr(&bytes.Buffer{})
		outcome, _ := Run(root, []string{"fillers", "role_0123", flag, "x"})
		if outcome != UsageError {
			t.Errorf("%s on `fillers` should be a UsageError, got %v", flag, outcome)
		}
		if tr.calls != 0 {
			t.Errorf("%s on `fillers` must send no request, got %d calls", flag, tr.calls)
		}
	}
}
