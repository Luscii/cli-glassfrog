package cli

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

// Unit coverage for the `tension subroles <role-id>` roll-up leaf (046). It reuses
// the landed runTensionList runner with tensionsConfig.subroles set (plan ADR-3), so
// these tests pin the ONE data difference — the swapped request path
// (/roles/{id}/subroles/tensions) — plus the behaviors that must stay distinct from
// `tension list`: a leaf-anchor 404 is a read failure, NOT an empty success. The
// status/paging/render/error branches are exercised by tension_reads_test.go through
// the same runner; here we confirm the subroles wiring drives them.

// --- path swap (the one data difference) -----------------------------------

func TestRunTensionSubroles_SuccessWalksSubrolesEndpoint(t *testing.T) {
	tr := &cannedTransport{status: 200, body: tensionsPageComplete}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, stderr := runTensionListOver(t, seam, tensionsConfig{id: "role_0123", subroles: true})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success\nstderr: %s", outcome, stderr)
	}
	if got := tr.lastPath; !strings.HasSuffix(got, "/roles/role_0123/subroles/tensions") {
		t.Errorf("path = %q, want /roles/role_0123/subroles/tensions", got)
	}
	for _, want := range []string{"ten_1  [unprocessed]  Roadmap drift", "ten_2  [processed]"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("each sub-role tension should print as a projection, missing %q:\n%s", want, stdout)
		}
	}
}

// TestRunTensionList_DefaultPathIsRoleOwnTensions pins that an unset subroles flag
// keeps the role's-own-tensions path — the path helper's two branches stay distinct.
func TestRunTensionList_DefaultPathIsRoleOwnTensions(t *testing.T) {
	tr := &cannedTransport{status: 200, body: tensionsPageComplete}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	_, _, _ = runTensionListOver(t, seam, tensionsConfig{id: "role_0123"})
	if got := tr.lastPath; !strings.HasSuffix(got, "/roles/role_0123/tensions") {
		t.Errorf("path = %q, want /roles/role_0123/tensions (subroles unset)", got)
	}
}

// --- leaf-404 is a failure, distinct from empty-200 ------------------------

// TestRunTensionSubroles_LeafAnchor404IsFailure pins the leaf-anchor 404 surfacing as
// a non-zero read failure naming the status — with NO "no sub-roles" special case
// (plan ADR-3).
func TestRunTensionSubroles_LeafAnchor404IsFailure(t *testing.T) {
	tr := &cannedTransport{status: 404, body: `{"detail":"Role not found"}`}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, stderr := runTensionListOver(t, seam, tensionsConfig{id: "role_0123", subroles: true})
	if outcome != APIError || ExitCode(outcome) != 3 {
		t.Fatalf("a leaf-anchor 404 should surface APIError/3, got %v/%d\nstderr: %s", outcome, ExitCode(outcome), stderr)
	}
	if !strings.Contains(stderr, "404") {
		t.Errorf("stderr should name the HTTP status (404):\n%s", stderr)
	}
	if strings.Contains(strings.ToLower(stderr), "no sub-roles") {
		t.Errorf("the 404 must NOT be re-interpreted as a no-sub-roles message:\n%s", stderr)
	}
	if strings.TrimSpace(stdout) != "" {
		t.Errorf("a failed roll-up prints no tension data, got stdout:\n%s", stdout)
	}
}

// TestRunTensionSubroles_EmptyIsCleanSuccess pins the genuinely-empty roll-up
// (sub-roles exist but carry no tensions) as a zero-exit success — distinct from the
// leaf-anchor 404 above. Both empty-ish outcomes must never conflate (plan ADR-3).
func TestRunTensionSubroles_EmptyIsCleanSuccess(t *testing.T) {
	tr := &cannedTransport{status: 200, body: tensionsPageEmpty}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, stderr := runTensionListOver(t, seam, tensionsConfig{id: "role_0123", subroles: true})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success", outcome)
	}
	if strings.TrimRight(stdout, "\n") != "no tensions" {
		t.Errorf("an empty roll-up should print exactly `no tensions`, got %q", stdout)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Errorf("an empty roll-up is a clean success; stderr should be empty, got %q", stderr)
	}
}

// --- status filter, paging, validation through the shared runner -----------

func TestRunTensionSubroles_StatusSentWhenSupplied(t *testing.T) {
	tr := &cannedTransport{status: 200, body: tensionsPageComplete}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	_, _, _ = runTensionListOver(t, seam, tensionsConfig{id: "role_0123", subroles: true, status: "unprocessed"})
	if got := tr.lastQuery.Get("status"); got != "unprocessed" {
		t.Errorf("status = %q, want \"unprocessed\"", got)
	}
	if got := tr.lastPath; !strings.HasSuffix(got, "/roles/role_0123/subroles/tensions") {
		t.Errorf("the filtered roll-up should still hit the subroles endpoint, got %q", got)
	}
}

func TestRunTensionSubroles_UnsupportedStatusIsUsageErrorNoRequest(t *testing.T) {
	tr := &cannedTransport{status: 200, body: tensionsPageComplete}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, _, stderr := runTensionListOver(t, seam, tensionsConfig{id: "role_0123", subroles: true, status: "open"})
	if outcome != UsageError || ExitCode(outcome) != 2 {
		t.Fatalf("outcome=%v exit=%d, want UsageError/2", outcome, ExitCode(outcome))
	}
	if !strings.Contains(stderr, "open") {
		t.Errorf("stderr should name the unsupported value:\n%s", stderr)
	}
	if tr.calls != 0 {
		t.Errorf("an unsupported --status must be rejected before any request, got %d calls", tr.calls)
	}
}

func TestRunTensionSubroles_WalksEveryPageToCompletion(t *testing.T) {
	tr := &seqMeTransport{steps: []seqMeResp{
		{status: 200, body: tensionsPage("ten_1", "Page One", "c1")},
		{status: 200, body: tensionsPage("ten_2", "Page Two", "c2")},
		{status: 200, body: tensionsPage("ten_3", "Page Three", "")},
	}}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, _ := runTensionListOver(t, seam, tensionsConfig{id: "role_0123", subroles: true})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success", outcome)
	}
	if tr.calls != 3 {
		t.Errorf("the roll-up should walk three pages, got %d", tr.calls)
	}
	for _, want := range []string{"ten_1", "ten_2", "ten_3"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("every page should print, missing %q:\n%s", want, stdout)
		}
	}
	if got := tr.lastPath; !strings.HasSuffix(got, "/roles/role_0123/subroles/tensions") {
		t.Errorf("the walk should target the subroles endpoint, got %q", got)
	}
}

func TestRunTensionSubroles_FirstPageStopsAndSignals(t *testing.T) {
	tr := &cannedTransport{status: 200, body: tensionsPage("ten_1", "First Page", "c1")}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, stderr := runTensionListOver(t, seam, tensionsConfig{id: "role_0123", subroles: true, firstPage: true})
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

func TestRunTensionSubroles_MidWalkFailurePartialAndIncomplete(t *testing.T) {
	tr := &seqMeTransport{steps: []seqMeResp{
		{status: 200, body: tensionsPage("ten_1", "Gathered", "c1")},
		{status: 500, body: `{"detail":"boom"}`},
	}}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, stderr := runTensionListOver(t, seam, tensionsConfig{id: "role_0123", subroles: true})
	if ExitCode(outcome) == 0 {
		t.Fatalf("a mid-walk failure must exit non-zero, got exit 0 (outcome %v)", outcome)
	}
	if !strings.Contains(stdout, "ten_1") {
		t.Errorf("the partial set gathered so far should print:\n%s", stdout)
	}
	if !strings.Contains(stderr, "incomplete") {
		t.Errorf("stderr should note the result is incomplete and name the cause:\n%s", stderr)
	}
}

func TestRunTensionSubroles_NoCredentialsIsUsageError(t *testing.T) {
	tr := &cannedTransport{status: 200, body: tensionsPageComplete}
	seam := &fakeMeSeam{ctx: noTokenContext(), transport: tr}

	outcome, stdout, stderr := runTensionListOver(t, seam, tensionsConfig{id: "role_0123", subroles: true})
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

func TestRunTensionSubroles_TransportErrorIsNetworkUnavailable(t *testing.T) {
	tr := &cannedTransport{netErr: errors.New("dial tcp: connection refused")}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, _, stderr := runTensionListOver(t, seam, tensionsConfig{id: "role_0123", subroles: true})
	if outcome != NetworkUnavailable || ExitCode(outcome) != 6 {
		t.Fatalf("outcome=%v exit=%d, want NetworkUnavailable/6\nstderr: %s", outcome, ExitCode(outcome), stderr)
	}
}

func TestRunTensionSubroles_BadOutputIsUsageErrorNoRequest(t *testing.T) {
	tr := &cannedTransport{status: 200, body: tensionsPageComplete}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, _, _ := runTensionListOver(t, seam, tensionsConfig{id: "role_0123", subroles: true, outputFlag: "xml", outputPresent: true})
	if outcome != UsageError || ExitCode(outcome) != 2 {
		t.Fatalf("outcome=%v exit=%d, want UsageError/2", outcome, ExitCode(outcome))
	}
	if tr.calls != 0 {
		t.Errorf("a bad --output must be rejected before any request, got %d calls", tr.calls)
	}
}

func TestRunTensionSubroles_StructuredJSONEmitsAggregatedRawPayload(t *testing.T) {
	tr := &cannedTransport{status: 200, body: tensionsPageComplete}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, _ := runTensionListOver(t, seam, tensionsConfig{id: "role_0123", subroles: true, outputFlag: "json", outputPresent: true})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success", outcome)
	}
	for _, want := range []string{`"data"`, "ten_1", `"role_id"`} {
		if !strings.Contains(stdout, want) {
			t.Errorf("structured json should carry the raw payload, missing %q:\n%s", want, stdout)
		}
	}
	if strings.Contains(stdout, "sensing role:") {
		t.Errorf("structured json must not render the human projection:\n%s", stdout)
	}
	if strings.Contains(stdout, `"pagination"`) {
		t.Errorf("the aggregated document must drop the per-page meta envelope:\n%s", stdout)
	}
}

// --- command-level wiring under the `tension` group ------------------------

// TestTensionSubrolesCommand_HitsSubrolesEndpoint pins the end-to-end wiring under the
// `tension` group: a real `tension subroles <id>` hits the roll-up endpoint.
func TestTensionSubrolesCommand_HitsSubrolesEndpoint(t *testing.T) {
	tr := &cannedTransport{status: 200, body: tensionsPageComplete}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	root := NewRootCommand()
	MustRegister(root, newTensionCommand(seam))
	var out, errb bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&errb)
	outcome, _ := Run(root, []string{"tension", "subroles", "role_0123"})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success\nstderr: %s", outcome, errb.String())
	}
	if got := tr.lastPath; !strings.HasSuffix(got, "/roles/role_0123/subroles/tensions") {
		t.Errorf("path = %q, want /roles/role_0123/subroles/tensions", got)
	}
}

// TestTensionSubrolesCommand_StatusSendsParam pins --status through the group.
func TestTensionSubrolesCommand_StatusSendsParam(t *testing.T) {
	tr := &cannedTransport{status: 200, body: tensionsPageComplete}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	root := NewRootCommand()
	MustRegister(root, newTensionCommand(seam))
	root.SetOut(&bytes.Buffer{})
	root.SetErr(&bytes.Buffer{})
	outcome, _ := Run(root, []string{"tension", "subroles", "role_0123", "--status", "unprocessed"})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success", outcome)
	}
	if got := tr.lastQuery.Get("status"); got != "unprocessed" {
		t.Errorf("status = %q, want \"unprocessed\"", got)
	}
}

// TestTensionSubrolesCommand_UnsupportedStatusNoRequest pins fail-fast --status
// validation at the command level under the group.
func TestTensionSubrolesCommand_UnsupportedStatusNoRequest(t *testing.T) {
	tr := &cannedTransport{status: 200, body: tensionsPageComplete}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	root := NewRootCommand()
	MustRegister(root, newTensionCommand(seam))
	root.SetOut(&bytes.Buffer{})
	root.SetErr(&bytes.Buffer{})
	outcome, _ := Run(root, []string{"tension", "subroles", "role_0123", "--status", "open"})
	if outcome != UsageError {
		t.Errorf("an unsupported --status should be a UsageError, got %v", outcome)
	}
	if tr.calls != 0 {
		t.Errorf("an unsupported --status must send no request, got %d calls", tr.calls)
	}
}

// TestTensionSubrolesCommand_RequiresExactlyOneArg pins ExactArgs(1): zero args is a
// usage error and sends no request.
func TestTensionSubrolesCommand_RequiresExactlyOneArg(t *testing.T) {
	tr := &cannedTransport{status: 200, body: tensionsPageComplete}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	root := NewRootCommand()
	MustRegister(root, newTensionCommand(seam))
	root.SetOut(&bytes.Buffer{})
	root.SetErr(&bytes.Buffer{})
	outcome, _ := Run(root, []string{"tension", "subroles"})
	if outcome != UsageError {
		t.Errorf("zero args should be a UsageError, got %v", outcome)
	}
	if tr.calls != 0 {
		t.Errorf("a wrong arg count must send no request, got %d calls", tr.calls)
	}
}
