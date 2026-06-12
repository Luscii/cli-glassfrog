package cli

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
)

// --- canned GET /actors bodies ---------------------------------------------
//
// They carry the Actor shape (id/name/kind + the timestamps the directory does
// not project) in the API's snake_case names, a mixed human/agent page to
// exercise both id prefixes and kind badges, and the secret token nowhere.

const actorsPageComplete = `{"data":[
  {"id":"per_0123","name":"Alice Smith","kind":"human","created_at":"t","updated_at":"t"},
  {"id":"agt_0456","name":"Claude","kind":"agent","created_at":"t","updated_at":"t"}
],"meta":{"pagination":{"per_page":100,"has_next_page":false,"next_cursor":""}}}`

const actorsPageEmpty = `{"data":[],"meta":{"pagination":{"per_page":100,"has_next_page":false,"next_cursor":""}}}`

// actorsAgentsOnlyPage is the agents-only page the --kind agent scenario expects
// the (filtered) API to return: only agent rows, no humans.
const actorsAgentsOnlyPage = `{"data":[
  {"id":"agt_0456","name":"Claude","kind":"agent","created_at":"t","updated_at":"t"}
],"meta":{"pagination":{"per_page":100,"has_next_page":false,"next_cursor":""}}}`

// actorsPage builds a one-actor page; a non-empty nextCursor marks more pages.
func actorsPage(id, name, kind, nextCursor string) string {
	hasNext := "false"
	if nextCursor != "" {
		hasNext = "true"
	}
	return `{"data":[{"id":"` + id + `","name":"` + name + `","kind":"` + kind + `","created_at":"t","updated_at":"t"}],` +
		`"meta":{"pagination":{"per_page":1,"has_next_page":` + hasNext + `,"next_cursor":"` + nextCursor + `"}}}`
}

// runActorsOver drives the pure runActorsList over a fake seam, returning the
// outcome and captured stdout/stderr, and failing if the token leaks.
func runActorsOver(t *testing.T, seam actorsSeam, cfg actorsConfig) (Outcome, string, string) {
	t.Helper()
	var out, errb bytes.Buffer
	cfg.seam = seam
	cfg.reqCtx = context.Background()
	cfg.stdout = &out
	cfg.stderr = &errb
	outcome, _ := runActorsList(cfg)
	if strings.Contains(out.String()+errb.String(), meSecretToken) {
		t.Fatalf("the token leaked into output: %q", out.String()+errb.String())
	}
	return outcome, out.String(), errb.String()
}

// --- list walk branches ----------------------------------------------------

func TestRunActors_ListSuccessWalksAndProjects(t *testing.T) {
	tr := &cannedTransport{status: 200, body: actorsPageComplete}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, stderr := runActorsOver(t, seam, actorsConfig{})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success\nstderr: %s", outcome, stderr)
	}
	// The default human format is `full`: an id+kind line, then an indented Name line.
	for _, want := range []string{
		"per_0123  [human]",
		"  Name:  Alice Smith",
		"agt_0456  [agent]",
		"  Name:  Claude",
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
	if got := tr.lastPath; !strings.HasSuffix(got, "/actors") {
		t.Errorf("path = %q, want it to target /actors", got)
	}
	// With no filter, nothing is sent.
	for _, param := range []string{"kind", "role_id", "q"} {
		if _, present := tr.lastQuery[param]; present {
			t.Errorf("no filter should be sent by default, got %q in %v", param, tr.lastQuery)
		}
	}
}

func TestRunActors_EmptyIsCleanSuccess(t *testing.T) {
	tr := &cannedTransport{status: 200, body: actorsPageEmpty}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, stderr := runActorsOver(t, seam, actorsConfig{})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success", outcome)
	}
	if strings.TrimRight(stdout, "\n") != "no actors" {
		t.Errorf("an org/filter with no actor should print exactly `no actors`, got %q", stdout)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Errorf("an empty list is a clean success; stderr should be empty, got %q", stderr)
	}
}

func TestRunActors_WalksEveryPageToCompletion(t *testing.T) {
	tr := &seqMeTransport{steps: []seqMeResp{
		{status: 200, body: actorsPage("per_1", "Page One", "human", "c1")},
		{status: 200, body: actorsPage("agt_2", "Page Two", "agent", "c2")},
		{status: 200, body: actorsPage("per_3", "Page Three", "human", "")},
	}}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, _ := runActorsOver(t, seam, actorsConfig{})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success", outcome)
	}
	if tr.calls != 3 {
		t.Errorf("the walk should issue three page requests, got %d", tr.calls)
	}
	for _, want := range []string{"Page One", "Page Two", "Page Three"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("every page's actors should print, missing %q:\n%s", want, stdout)
		}
	}
}

// --- --kind / --role-id / --query filters ----------------------------------

func TestRunActors_KindSentWhenSetAndNonEmpty(t *testing.T) {
	tr := &cannedTransport{status: 200, body: actorsAgentsOnlyPage}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, _ := runActorsOver(t, seam, actorsConfig{kind: "agent", kindSet: true})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success", outcome)
	}
	if got := tr.lastQuery.Get("kind"); got != "agent" {
		t.Errorf("kind = %q, want \"agent\"", got)
	}
	// Only agents are printed (the filtered API response carries no humans).
	if !strings.Contains(stdout, "agt_0456") || strings.Contains(stdout, "per_") {
		t.Errorf("only agents should print for --kind agent:\n%s", stdout)
	}
}

func TestRunActors_RoleIDSentWhenSetAndNonEmpty(t *testing.T) {
	tr := &cannedTransport{status: 200, body: actorsPageComplete}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	_, _, _ = runActorsOver(t, seam, actorsConfig{roleID: "role_abc", roleSet: true})
	if got := tr.lastQuery.Get("role_id"); got != "role_abc" {
		t.Errorf("role_id = %q, want \"role_abc\"", got)
	}
}

func TestRunActors_QuerySentWhenSetAndNonEmpty(t *testing.T) {
	tr := &cannedTransport{status: 200, body: actorsPageComplete}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	_, _, _ = runActorsOver(t, seam, actorsConfig{query: "alice", querySet: true})
	if got := tr.lastQuery.Get("q"); got != "alice" {
		t.Errorf("q = %q, want \"alice\"", got)
	}
}

func TestRunActors_ThreeFiltersCombine(t *testing.T) {
	tr := &cannedTransport{status: 200, body: actorsPageComplete}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	_, _, _ = runActorsOver(t, seam, actorsConfig{
		kind: "human", kindSet: true,
		roleID: "role_abc", roleSet: true,
		query: "x", querySet: true,
	})
	if got := tr.lastQuery.Get("kind"); got != "human" {
		t.Errorf("kind = %q, want \"human\"", got)
	}
	if got := tr.lastQuery.Get("role_id"); got != "role_abc" {
		t.Errorf("role_id = %q, want \"role_abc\"", got)
	}
	if got := tr.lastQuery.Get("q"); got != "x" {
		t.Errorf("q = %q, want \"x\"", got)
	}
}

func TestRunActors_OmittedOrEmptyFiltersSendNothing(t *testing.T) {
	// Omitted (all *Set=false).
	tr1 := &cannedTransport{status: 200, body: actorsPageComplete}
	seam1 := &fakeMeSeam{ctx: validMeContext(), transport: tr1}
	_, _, _ = runActorsOver(t, seam1, actorsConfig{})
	for _, param := range []string{"kind", "role_id", "q"} {
		if _, present := tr1.lastQuery[param]; present {
			t.Errorf("an omitted filter must not send %q, got %v", param, tr1.lastQuery)
		}
	}

	// Present but empty (--role-id ""/--query "" set; --kind "" set is a no-op too —
	// validateKind treats empty as no constraint).
	tr2 := &cannedTransport{status: 200, body: actorsPageComplete}
	seam2 := &fakeMeSeam{ctx: validMeContext(), transport: tr2}
	_, _, _ = runActorsOver(t, seam2, actorsConfig{kind: "", kindSet: true, roleID: "", roleSet: true, query: "", querySet: true})
	for _, param := range []string{"kind", "role_id", "q"} {
		if _, present := tr2.lastQuery[param]; present {
			t.Errorf("an empty filter must behave as no filter (no %q), got %v", param, tr2.lastQuery)
		}
	}
}

// TestRunActors_FiltersRetainedOnEveryPage pins the plan Risk: the filters must
// ride EVERY page request of a multi-page walk, not just the first.
func TestRunActors_FiltersRetainedOnEveryPage(t *testing.T) {
	tr := &recordingSeqTransport{steps: []seqMeResp{
		{status: 200, body: actorsPage("agt_1", "Page One", "agent", "c1")},
		{status: 200, body: actorsPage("agt_2", "Page Two", "agent", "")},
	}}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, _, _ := runActorsOver(t, seam, actorsConfig{kind: "agent", kindSet: true})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success", outcome)
	}
	if len(tr.queries) < 2 {
		t.Fatalf("expected the walk to span more than one page, got %d requests", len(tr.queries))
	}
	for i, q := range tr.queries {
		if got := q.Get("kind"); got != "agent" {
			t.Errorf("page-%d request kind = %q, want \"agent\" (must ride every page)", i+1, got)
		}
	}
}

// --- --kind validation (the one closed-enum input) -------------------------

func TestRunActors_UnsupportedKindIsUsageErrorNoRequest(t *testing.T) {
	tr := &cannedTransport{status: 200, body: actorsPageComplete}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, _, stderr := runActorsOver(t, seam, actorsConfig{kind: "robot", kindSet: true})
	if outcome != UsageError || ExitCode(outcome) != 2 {
		t.Fatalf("outcome=%v exit=%d, want UsageError/2", outcome, ExitCode(outcome))
	}
	if !strings.Contains(stderr, "robot") {
		t.Errorf("stderr should name the unsupported value:\n%s", stderr)
	}
	for _, want := range []string{"agent", "human"} {
		if !strings.Contains(stderr, want) {
			t.Errorf("stderr should list the supported set (%q):\n%s", want, stderr)
		}
	}
	if tr.calls != 0 {
		t.Errorf("an unsupported --kind must be rejected before any request, got %d calls", tr.calls)
	}
}

// --- --first-page ----------------------------------------------------------

func TestRunActors_FirstPageStopsAndSignals(t *testing.T) {
	tr := &cannedTransport{status: 200, body: actorsPage("per_1", "First Page Actor", "human", "c1")}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, stderr := runActorsOver(t, seam, actorsConfig{firstPage: true})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success", outcome)
	}
	if !strings.Contains(stdout, "First Page Actor") {
		t.Errorf("the first page should print:\n%s", stdout)
	}
	if tr.calls != 1 {
		t.Errorf("--first-page must not walk, want 1 call, got %d", tr.calls)
	}
	if !strings.Contains(stderr, "more actors exist") {
		t.Errorf("stderr should note more actors exist:\n%s", stderr)
	}
}

func TestRunActors_PerPageSizesWalk(t *testing.T) {
	tr := &cannedTransport{status: 200, body: actorsPageComplete}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	_, _, _ = runActorsOver(t, seam, actorsConfig{perPage: 7, perPageSet: true})
	if got := tr.lastQuery.Get("per_page"); got != "7" {
		t.Errorf("per_page = %q, want \"7\" (WithPageSize passed through)", got)
	}
}

// --- mid-walk failure ------------------------------------------------------

func TestRunActors_MidWalkFailurePartialAndIncomplete(t *testing.T) {
	tr := &seqMeTransport{steps: []seqMeResp{
		{status: 200, body: actorsPage("per_1", "Gathered Actor", "human", "c1")},
		{status: 500, body: `{"detail":"boom"}`},
	}}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, stderr := runActorsOver(t, seam, actorsConfig{})
	if outcome == Success {
		t.Fatalf("a mid-walk failure must be non-zero, got Success")
	}
	if ExitCode(outcome) == 0 {
		t.Errorf("a mid-walk failure must exit non-zero, got exit 0 (outcome %v)", outcome)
	}
	if !strings.Contains(stdout, "Gathered Actor") {
		t.Errorf("the partial set gathered so far should print:\n%s", stdout)
	}
	if !strings.Contains(stderr, "incomplete") {
		t.Errorf("stderr should note the result is incomplete and name the cause:\n%s", stderr)
	}
}

// --- error classification (via the shared classifier) ----------------------

func TestRunActors_NoCredentialsIsUsageError(t *testing.T) {
	tr := &cannedTransport{status: 200, body: actorsPageComplete}
	seam := &fakeMeSeam{ctx: noTokenContext(), transport: tr}

	outcome, stdout, stderr := runActorsOver(t, seam, actorsConfig{})
	if outcome != UsageError || ExitCode(outcome) != 2 {
		t.Fatalf("outcome=%v exit=%d, want UsageError/2", outcome, ExitCode(outcome))
	}
	if !strings.Contains(strings.ToLower(stderr), "not authenticated") {
		t.Errorf("stderr should report not authenticated:\n%s", stderr)
	}
	if strings.TrimSpace(stdout) != "" {
		t.Errorf("no actor data should be printed on a credential failure, got:\n%s", stdout)
	}
}

func TestRunActors_TransportErrorIsNetworkUnavailable(t *testing.T) {
	tr := &cannedTransport{netErr: errors.New("dial tcp: connection refused")}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, _, stderr := runActorsOver(t, seam, actorsConfig{})
	if outcome != NetworkUnavailable || ExitCode(outcome) != 6 {
		t.Fatalf("outcome=%v exit=%d, want NetworkUnavailable/6\nstderr: %s", outcome, ExitCode(outcome), stderr)
	}
}

// A malformed --role-id is passed through to the API and surfaces its status (a
// local rejection would defeat the pass-through contract — plan ADR-3).
func TestRunActors_MalformedRoleIDSurfacesAPIStatus(t *testing.T) {
	tr := &cannedTransport{status: 400, body: `{"detail":"invalid role id"}`}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, _, stderr := runActorsOver(t, seam, actorsConfig{roleID: "not-a-role", roleSet: true})
	if outcome != APIError || ExitCode(outcome) != 3 {
		t.Fatalf("a malformed role_id should surface APIError/3, got %v/%d\nstderr: %s", outcome, ExitCode(outcome), stderr)
	}
	if !strings.Contains(stderr, "400") {
		t.Errorf("stderr should name the HTTP status (400):\n%s", stderr)
	}
	if tr.calls != 1 {
		t.Errorf("a free identifier must be passed through (one request issued), got %d calls", tr.calls)
	}
}

func TestRunActors_Non2xxClassifies(t *testing.T) {
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
		outcome, _, stderr := runActorsOver(t, seam, actorsConfig{})
		if outcome != c.want || ExitCode(outcome) != c.code {
			t.Errorf("status %d: outcome=%v exit=%d, want %v/%d\nstderr: %s", c.status, outcome, ExitCode(outcome), c.want, c.code, stderr)
		}
	}
}

// --- resolve-before-call: a bad --output costs no request ------------------

func TestRunActors_BadOutputIsUsageErrorNoRequest(t *testing.T) {
	tr := &cannedTransport{status: 200, body: actorsPageComplete}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, _, stderr := runActorsOver(t, seam, actorsConfig{outputFlag: "xml", outputPresent: true})
	if outcome != UsageError || ExitCode(outcome) != 2 {
		t.Fatalf("outcome=%v exit=%d, want UsageError/2\nstderr: %s", outcome, ExitCode(outcome), stderr)
	}
	if tr.calls != 0 {
		t.Errorf("a bad --output must be rejected before any request, got %d calls", tr.calls)
	}
}

// TestRunActors_OutputResolvedBeforeKind pins the output-first precedence
// (interface § Interactions): when BOTH --output and --kind are invalid, the
// --output error is reported, and neither costs a request.
func TestRunActors_OutputResolvedBeforeKind(t *testing.T) {
	tr := &cannedTransport{status: 200, body: actorsPageComplete}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, _, stderr := runActorsOver(t, seam, actorsConfig{outputFlag: "xml", outputPresent: true, kind: "robot", kindSet: true})
	if outcome != UsageError {
		t.Fatalf("outcome = %v, want UsageError", outcome)
	}
	if strings.Contains(stderr, "robot") {
		t.Errorf("the --output error should be reported first, not the --kind error:\n%s", stderr)
	}
	if tr.calls != 0 {
		t.Errorf("both checks are pre-assembly; no request must be sent, got %d calls", tr.calls)
	}
}

// --- structured output emits the aggregated raw document --------------------

func TestRunActors_StructuredJSONEmitsAggregatedRawPayload(t *testing.T) {
	tr := &cannedTransport{status: 200, body: actorsPageComplete}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, _ := runActorsOver(t, seam, actorsConfig{outputFlag: "json", outputPresent: true})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success", outcome)
	}
	for _, want := range []string{`"data"`, "per_0123", `"kind"`, "agt_0456"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("structured json should carry the raw payload, missing %q:\n%s", want, stdout)
		}
	}
	// Structured output must not carry the human projection's kind badge nor the
	// per-page meta envelope.
	if strings.Contains(stdout, "[human]") {
		t.Errorf("structured json must not render the human projection:\n%s", stdout)
	}
	if strings.Contains(stdout, `"pagination"`) {
		t.Errorf("the aggregated document must drop the per-page meta envelope:\n%s", stdout)
	}
}

// --- validateKind unit (the SOT validator) ---------------------------------

func TestValidateKind(t *testing.T) {
	for _, ok := range []string{"", "human", "agent"} {
		if err := validateKind(ok); err != nil {
			t.Errorf("validateKind(%q) should be valid, got %v", ok, err)
		}
	}
	err := validateKind("robot")
	if err == nil {
		t.Fatal("validateKind(\"robot\") should reject")
	}
	for _, want := range []string{"--kind", "robot", "agent", "human"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("validateKind error should name %q, got %q", want, err.Error())
		}
	}
}

// --- command-level wiring --------------------------------------------------

// TestActorsCommand_FiltersSendParams pins the Changed()-gating end to end: a real
// `actors --kind human --role-id role_abc --query x` invocation sends all three.
func TestActorsCommand_FiltersSendParams(t *testing.T) {
	tr := &cannedTransport{status: 200, body: actorsPageComplete}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	root := NewRootCommand()
	MustRegister(root, newActorsCommand(seam))
	var out, errb bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&errb)
	outcome, _ := Run(root, []string{"actors", "--kind", "human", "--role-id", "role_abc", "--query", "x"})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success\nstderr: %s", outcome, errb.String())
	}
	if got := tr.lastQuery.Get("kind"); got != "human" {
		t.Errorf("kind = %q, want \"human\"", got)
	}
	if got := tr.lastQuery.Get("role_id"); got != "role_abc" {
		t.Errorf("role_id = %q, want \"role_abc\"", got)
	}
	if got := tr.lastQuery.Get("q"); got != "x" {
		t.Errorf("q = %q, want \"x\"", got)
	}
}

// TestActorsCommand_ShortQueryAlias pins the -q alias for --query.
func TestActorsCommand_ShortQueryAlias(t *testing.T) {
	tr := &cannedTransport{status: 200, body: actorsPageComplete}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	root := NewRootCommand()
	MustRegister(root, newActorsCommand(seam))
	root.SetOut(&bytes.Buffer{})
	root.SetErr(&bytes.Buffer{})
	outcome, _ := Run(root, []string{"actors", "-q", "alice"})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success", outcome)
	}
	if got := tr.lastQuery.Get("q"); got != "alice" {
		t.Errorf("-q should map to q, got %q", got)
	}
}

// TestActorsCommand_UnsupportedKindNoRequest pins fail-fast --kind validation at
// the command level: a real invocation with a bad kind sends no request.
func TestActorsCommand_UnsupportedKindNoRequest(t *testing.T) {
	tr := &cannedTransport{status: 200, body: actorsPageComplete}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	root := NewRootCommand()
	MustRegister(root, newActorsCommand(seam))
	root.SetOut(&bytes.Buffer{})
	root.SetErr(&bytes.Buffer{})
	outcome, _ := Run(root, []string{"actors", "--kind", "robot"})
	if outcome != UsageError {
		t.Errorf("an unsupported --kind should be a UsageError, got %v", outcome)
	}
	if tr.calls != 0 {
		t.Errorf("an unsupported --kind must send no request, got %d calls", tr.calls)
	}
}

// TestActorsCommand_PositionalIsUsageErrorNoRequest pins cobra.NoArgs: a positional
// is a usage error before any request (no hand-rolled guard).
func TestActorsCommand_PositionalIsUsageErrorNoRequest(t *testing.T) {
	tr := &cannedTransport{status: 200, body: actorsPageComplete}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	root := NewRootCommand()
	MustRegister(root, newActorsCommand(seam))
	root.SetOut(&bytes.Buffer{})
	root.SetErr(&bytes.Buffer{})
	outcome, _ := Run(root, []string{"actors", "per_0123"})
	if outcome != UsageError {
		t.Errorf("a positional should be a UsageError via NoArgs, got %v", outcome)
	}
	if tr.calls != 0 {
		t.Errorf("a positional must send no request, got %d calls", tr.calls)
	}
}
