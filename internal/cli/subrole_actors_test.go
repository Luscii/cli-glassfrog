package cli

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
)

// runSubroleActorsOver drives the pure runSubroleActors orchestration over a fake
// seam, returning the outcome and captured stdout/stderr, and failing if the token
// leaks. It defaults the anchor role id to role_0123 (the feature's fixture id) so a
// test that does not exercise the id can leave it unset. It is the roll-up sibling of
// actors_test.go's runActorsOver.
func runSubroleActorsOver(t *testing.T, seam actorsSeam, cfg subroleActorsConfig) (Outcome, string, string) {
	t.Helper()
	var out, errb bytes.Buffer
	cfg.seam = seam
	cfg.reqCtx = context.Background()
	cfg.stdout = &out
	cfg.stderr = &errb
	if cfg.roleID == "" {
		cfg.roleID = "role_0123"
	}
	outcome, _ := runSubroleActors(cfg)
	if strings.Contains(out.String()+errb.String(), meSecretToken) {
		t.Fatalf("the token leaked into output: %q", out.String()+errb.String())
	}
	return outcome, out.String(), errb.String()
}

// --- list walk branches ----------------------------------------------------

func TestRunSubroleActors_RollsUpAndProjectsAtSubrolesPath(t *testing.T) {
	tr := &cannedTransport{status: 200, body: actorsPageComplete}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, stderr := runSubroleActorsOver(t, seam, subroleActorsConfig{})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success\nstderr: %s", outcome, stderr)
	}
	// Bare actor projection (id/name/kind) — the same `actors` render key 048 ships;
	// NO focus / elected_until (that is 047's assignment shape).
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
	for _, absent := range []string{"focus", "elected_until"} {
		if strings.Contains(stdout, absent) {
			t.Errorf("the roll-up is actor-shaped, not assignment-shaped; stdout must not project %q:\n%s", absent, stdout)
		}
	}
	if strings.TrimSpace(stderr) != "" {
		t.Errorf("a complete success should write nothing to stderr, got %q", stderr)
	}
	if tr.calls != 1 {
		t.Errorf("a single complete page should be one call, got %d", tr.calls)
	}
	if got := tr.lastPath; !strings.HasSuffix(got, "/roles/role_0123/subroles/actors") {
		t.Errorf("path = %q, want it to target /roles/role_0123/subroles/actors", got)
	}
	// With no --kind, nothing is sent.
	if _, present := tr.lastQuery["kind"]; present {
		t.Errorf("no filter should be sent by default, got kind=%v", tr.lastQuery["kind"])
	}
}

// TestRunSubroleActors_EmptyIsCleanSuccess pins the empty-200 outcome (sub-roles
// exist but carry no fillers) as a zero-exit success — distinct from a leaf 404.
func TestRunSubroleActors_EmptyIsCleanSuccess(t *testing.T) {
	tr := &cannedTransport{status: 200, body: actorsPageEmpty}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, stderr := runSubroleActorsOver(t, seam, subroleActorsConfig{})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success", outcome)
	}
	if strings.TrimRight(stdout, "\n") != "no actors" {
		t.Errorf("sub-roles with no fillers should print exactly `no actors`, got %q", stdout)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Errorf("an empty list is a clean success; stderr should be empty, got %q", stderr)
	}
}

func TestRunSubroleActors_WalksEveryPageToCompletion(t *testing.T) {
	tr := &seqMeTransport{steps: []seqMeResp{
		{status: 200, body: actorsPage("per_1", "Page One", "human", "c1")},
		{status: 200, body: actorsPage("agt_2", "Page Two", "agent", "c2")},
		{status: 200, body: actorsPage("per_3", "Page Three", "human", "")},
	}}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, _ := runSubroleActorsOver(t, seam, subroleActorsConfig{})
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

// --- --kind filter (the one closed-enum input) -----------------------------

func TestRunSubroleActors_KindSentWhenSetAndNonEmpty(t *testing.T) {
	tr := &cannedTransport{status: 200, body: actorsAgentsOnlyPage}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, _ := runSubroleActorsOver(t, seam, subroleActorsConfig{kind: "agent", kindSet: true})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success", outcome)
	}
	if got := tr.lastQuery.Get("kind"); got != "agent" {
		t.Errorf("kind = %q, want \"agent\"", got)
	}
	if !strings.Contains(stdout, "agt_0456") || strings.Contains(stdout, "per_") {
		t.Errorf("only agents should print for --kind agent:\n%s", stdout)
	}
}

func TestRunSubroleActors_OmittedOrEmptyKindSendsNothing(t *testing.T) {
	// Omitted (kindSet=false).
	tr1 := &cannedTransport{status: 200, body: actorsPageComplete}
	seam1 := &fakeMeSeam{ctx: validMeContext(), transport: tr1}
	_, _, _ = runSubroleActorsOver(t, seam1, subroleActorsConfig{})
	if _, present := tr1.lastQuery["kind"]; present {
		t.Errorf("an omitted --kind must not send kind, got %v", tr1.lastQuery)
	}

	// Present but empty (--kind "" set) — validateKind treats empty as no constraint.
	tr2 := &cannedTransport{status: 200, body: actorsPageComplete}
	seam2 := &fakeMeSeam{ctx: validMeContext(), transport: tr2}
	_, _, _ = runSubroleActorsOver(t, seam2, subroleActorsConfig{kind: "", kindSet: true})
	if _, present := tr2.lastQuery["kind"]; present {
		t.Errorf("`--kind \"\"` must behave as no filter, got %v", tr2.lastQuery)
	}
}

// TestRunSubroleActors_KindRetainedOnEveryPage pins that the kind filter rides every
// page request of a multi-page walk (paging.All threads the base request's query).
func TestRunSubroleActors_KindRetainedOnEveryPage(t *testing.T) {
	tr := &recordingSeqTransport{steps: []seqMeResp{
		{status: 200, body: actorsPage("per_1", "Page One", "human", "c1")},
		{status: 200, body: actorsPage("per_2", "Page Two", "human", "")},
	}}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, _, _ := runSubroleActorsOver(t, seam, subroleActorsConfig{kind: "human", kindSet: true})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success", outcome)
	}
	if len(tr.queries) < 2 {
		t.Fatalf("expected the walk to span more than one page, got %d requests", len(tr.queries))
	}
	for i, q := range tr.queries {
		if got := q.Get("kind"); got != "human" {
			t.Errorf("page-%d request kind = %q, want \"human\" (must ride every page)", i+1, got)
		}
	}
	// Every page must also stay on the subroles endpoint.
	for i, p := range tr.paths {
		if !strings.HasSuffix(p, "/roles/role_0123/subroles/actors") {
			t.Errorf("page-%d path = %q, want the subroles actors endpoint", i+1, p)
		}
	}
}

func TestRunSubroleActors_UnsupportedKindIsUsageErrorNoRequest(t *testing.T) {
	tr := &cannedTransport{status: 200, body: actorsPageComplete}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, _, stderr := runSubroleActorsOver(t, seam, subroleActorsConfig{kind: "robot", kindSet: true})
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

// --- --first-page / --per-page ---------------------------------------------

func TestRunSubroleActors_FirstPageStopsAndSignals(t *testing.T) {
	tr := &cannedTransport{status: 200, body: actorsPage("per_1", "First Page Actor", "human", "c1")}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, stderr := runSubroleActorsOver(t, seam, subroleActorsConfig{firstPage: true})
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

func TestRunSubroleActors_PerPageSizesWalk(t *testing.T) {
	tr := &cannedTransport{status: 200, body: actorsPageComplete}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	_, _, _ = runSubroleActorsOver(t, seam, subroleActorsConfig{perPage: 7, perPageSet: true})
	if got := tr.lastQuery.Get("per_page"); got != "7" {
		t.Errorf("per_page = %q, want \"7\" (WithPageSize passed through)", got)
	}
}

// --- mid-walk failure ------------------------------------------------------

func TestRunSubroleActors_MidWalkFailurePartialAndIncomplete(t *testing.T) {
	tr := &seqMeTransport{steps: []seqMeResp{
		{status: 200, body: actorsPage("per_1", "Gathered Actor", "human", "c1")},
		{status: 500, body: `{"detail":"boom"}`},
	}}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, stderr := runSubroleActorsOver(t, seam, subroleActorsConfig{})
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

func TestRunSubroleActors_NoCredentialsIsUsageError(t *testing.T) {
	tr := &cannedTransport{status: 200, body: actorsPageComplete}
	seam := &fakeMeSeam{ctx: noTokenContext(), transport: tr}

	outcome, stdout, stderr := runSubroleActorsOver(t, seam, subroleActorsConfig{})
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

func TestRunSubroleActors_TransportErrorIsNetworkUnavailable(t *testing.T) {
	tr := &cannedTransport{netErr: errors.New("dial tcp: connection refused")}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, _, stderr := runSubroleActorsOver(t, seam, subroleActorsConfig{})
	if outcome != NetworkUnavailable || ExitCode(outcome) != 6 {
		t.Fatalf("outcome=%v exit=%d, want NetworkUnavailable/6\nstderr: %s", outcome, ExitCode(outcome), stderr)
	}
}

// TestRunSubroleActors_LeafAnchor404SurfacesStatus pins the leaf-anchor 404 as a
// surfaced read failure naming the status — with NO "this role has no sub-roles"
// interpretation (plan ADR-3). It is genuinely distinct from the empty-200 success.
func TestRunSubroleActors_LeafAnchor404SurfacesStatus(t *testing.T) {
	tr := &cannedTransport{status: 404, body: `{"detail":"role has no subroles"}`}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, stderr := runSubroleActorsOver(t, seam, subroleActorsConfig{})
	if outcome != APIError || ExitCode(outcome) != 3 {
		t.Fatalf("a leaf-anchor 404 should surface APIError/3, got %v/%d\nstderr: %s", outcome, ExitCode(outcome), stderr)
	}
	if !strings.Contains(stderr, "404") {
		t.Errorf("stderr should name the HTTP status (404):\n%s", stderr)
	}
	if strings.Contains(strings.ToLower(stderr+stdout), "no sub-roles") {
		t.Errorf("no \"this role has no sub-roles\" interpretation must be added:\nstderr: %s\nstdout: %s", stderr, stdout)
	}
	// Distinct from the empty-200 success: a 404 prints no `no actors` empty line.
	if strings.Contains(stdout, "no actors") {
		t.Errorf("a leaf 404 must NOT render the empty-list success line:\n%s", stdout)
	}
	if tr.calls != 1 {
		t.Errorf("the anchor id is passed through (one request issued), got %d calls", tr.calls)
	}
}

func TestRunSubroleActors_Non2xxClassifies(t *testing.T) {
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
		outcome, _, stderr := runSubroleActorsOver(t, seam, subroleActorsConfig{})
		if outcome != c.want || ExitCode(outcome) != c.code {
			t.Errorf("status %d: outcome=%v exit=%d, want %v/%d\nstderr: %s", c.status, outcome, ExitCode(outcome), c.want, c.code, stderr)
		}
	}
}

// --- resolve-before-call: a bad --output / output-first precedence ----------

func TestRunSubroleActors_BadOutputIsUsageErrorNoRequest(t *testing.T) {
	tr := &cannedTransport{status: 200, body: actorsPageComplete}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, _, stderr := runSubroleActorsOver(t, seam, subroleActorsConfig{outputFlag: "xml", outputPresent: true})
	if outcome != UsageError || ExitCode(outcome) != 2 {
		t.Fatalf("outcome=%v exit=%d, want UsageError/2\nstderr: %s", outcome, ExitCode(outcome), stderr)
	}
	if tr.calls != 0 {
		t.Errorf("a bad --output must be rejected before any request, got %d calls", tr.calls)
	}
}

// TestRunSubroleActors_OutputResolvedBeforeKind pins the output-first precedence
// (interface § Interactions): when BOTH --output and --kind are invalid, the
// --output error is reported, and neither costs a request.
func TestRunSubroleActors_OutputResolvedBeforeKind(t *testing.T) {
	tr := &cannedTransport{status: 200, body: actorsPageComplete}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, _, stderr := runSubroleActorsOver(t, seam, subroleActorsConfig{outputFlag: "xml", outputPresent: true, kind: "robot", kindSet: true})
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

func TestRunSubroleActors_StructuredJSONEmitsAggregatedRawPayload(t *testing.T) {
	tr := &cannedTransport{status: 200, body: actorsPageComplete}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, _ := runSubroleActorsOver(t, seam, subroleActorsConfig{outputFlag: "json", outputPresent: true})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success", outcome)
	}
	for _, want := range []string{`"data"`, "per_0123", `"kind"`, "agt_0456"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("structured json should carry the raw payload, missing %q:\n%s", want, stdout)
		}
	}
	if strings.Contains(stdout, "[human]") {
		t.Errorf("structured json must not render the human projection:\n%s", stdout)
	}
	if strings.Contains(stdout, `"pagination"`) {
		t.Errorf("the aggregated document drops per-page meta, got pagination:\n%s", stdout)
	}
}

// --- anchor id pass-through + path escaping --------------------------------

// TestRunSubroleActors_RoleIDEscapedAsOneSegment pins that a `/` in the anchor id is
// escaped to one opaque path segment (the 025 runRoleGet discipline) — a raw `/`
// cannot redirect the request to a different endpoint.
func TestRunSubroleActors_RoleIDEscapedAsOneSegment(t *testing.T) {
	tr := &cannedTransport{status: 200, body: actorsPageComplete}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	_, _, _ = runSubroleActorsOver(t, seam, subroleActorsConfig{roleID: "role_x/extra"})
	if strings.Contains(tr.lastPath, "role_x/extra/subroles") {
		t.Errorf("a `/` in the id must not create extra path segments: %q", tr.lastPath)
	}
	if !strings.Contains(tr.lastPath, "role_x%2Fextra") {
		t.Errorf("the id should be escaped as a single path segment, got %q", tr.lastPath)
	}
}

// --- command surface -------------------------------------------------------

// TestNewSubroleActorsCommand_SurfaceOnlyKind pins the leaf's flag/arg surface: an
// ExactArgs(1) anchor, --kind/--first-page/--per-page declared, and deliberately NO
// --role-id and NO --query (the endpoint offers neither — plan ADR-2).
func TestNewSubroleActorsCommand_SurfaceOnlyKind(t *testing.T) {
	cmd := newSubroleActorsCommand(&fakeMeSeam{})

	if cmd.Use != "subrole-actors <role-id>" {
		t.Errorf("Use = %q, want \"subrole-actors <role-id>\"", cmd.Use)
	}
	if strings.TrimSpace(cmd.Short) == "" {
		t.Error("the leaf must declare a non-empty Short")
	}
	if !cmd.SilenceErrors || !cmd.SilenceUsage {
		t.Error("the leaf must set SilenceErrors and SilenceUsage so its runner owns the messages")
	}
	if cmd.Args == nil {
		t.Error("the leaf must declare an Args validator (ExactArgs(1))")
	} else {
		if err := cmd.Args(cmd, []string{}); err == nil {
			t.Error("ExactArgs(1) must reject zero positionals")
		}
		if err := cmd.Args(cmd, []string{"a", "b"}); err == nil {
			t.Error("ExactArgs(1) must reject a second positional")
		}
		if err := cmd.Args(cmd, []string{"role_0123"}); err != nil {
			t.Errorf("ExactArgs(1) must accept exactly one positional, got %v", err)
		}
	}
	for _, want := range []string{"kind", "first-page", "per-page"} {
		if cmd.Flags().Lookup(want) == nil {
			t.Errorf("the leaf must declare the --%s flag", want)
		}
	}
	for _, absent := range []string{"role-id", "query"} {
		if cmd.Flags().Lookup(absent) != nil {
			t.Errorf("the leaf must NOT declare --%s (the endpoint offers no such filter)", absent)
		}
	}
}
