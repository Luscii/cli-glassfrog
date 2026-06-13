package cli

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
)

// tensionUpdatedBody is a representative updateTension 200 body: the single-object
// {data: Tension} envelope after an edit (status archived to exercise the
// transition). The secret token appears nowhere.
const tensionUpdatedBody = `{"data":{"id":"ten_0123","type":"tension","body":"Roadmap updates lag behind shipped work.","status":"archived","role_id":"role_0123","sensed_by_id":"per_0123","created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-02T00:00:00Z","label":null,"meeting_type":null,"parent_role_id":null}}`

// runTensionUpdateOver drives the pure runTensionUpdate over a fake seam, returning
// the outcome and captured stdout/stderr, and failing if the token leaks.
func runTensionUpdateOver(t *testing.T, seam tensionSeam, cfg tensionUpdateConfig) (Outcome, string, string) {
	t.Helper()
	var out, errb bytes.Buffer
	cfg.seam = seam
	cfg.reqCtx = context.Background()
	cfg.stdout = &out
	cfg.stderr = &errb
	outcome, _ := runTensionUpdate(cfg)
	if strings.Contains(out.String()+errb.String(), meSecretToken) {
		t.Fatalf("the token leaked into output: %q", out.String()+errb.String())
	}
	return outcome, out.String(), errb.String()
}

// --- happy path: the PATCH shape -------------------------------------------

func TestRunTensionUpdate_BodyOnlyPatchesTension(t *testing.T) {
	tr := &tensionTransport{status: 200, body: tensionUpdatedBody}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, stderr := runTensionUpdateOver(t, seam, tensionUpdateConfig{
		id: "ten_0123", body: "X", bodySet: true,
	})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success\nstderr: %s", outcome, stderr)
	}
	if tr.calls != 1 {
		t.Fatalf("an update is exactly one request, got %d", tr.calls)
	}
	if tr.lastMethod != http.MethodPatch {
		t.Errorf("method = %q, want PATCH", tr.lastMethod)
	}
	if !strings.HasSuffix(tr.lastPath, "/tensions/ten_0123") {
		t.Errorf("path = %q, want /tensions/ten_0123", tr.lastPath)
	}
	if tr.lastContentType != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", tr.lastContentType)
	}
	if tr.lastIfMatch != "" {
		t.Errorf("no If-Match must be sent (last-write-wins), got %q", tr.lastIfMatch)
	}
	if tr.lastBody != `{"tension":{"body":"X"}}` {
		t.Errorf("body = %s, want a body-only partial envelope", tr.lastBody)
	}
	if !strings.Contains(stdout, "ten_0123") {
		t.Errorf("the updated tension should be printed:\n%s", stdout)
	}
}

func TestRunTensionUpdate_StatusArchivedSent(t *testing.T) {
	tr := &tensionTransport{status: 200, body: tensionUpdatedBody}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, _, stderr := runTensionUpdateOver(t, seam, tensionUpdateConfig{
		id: "ten_0123", status: "archived", statusSet: true,
	})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success\nstderr: %s", outcome, stderr)
	}
	if tr.lastBody != `{"tension":{"status":"archived"}}` {
		t.Errorf("body = %s, want status-only partial envelope", tr.lastBody)
	}
}

func TestRunTensionUpdate_LabelAndMeetingTypeOnly(t *testing.T) {
	tr := &tensionTransport{status: 200, body: tensionUpdatedBody}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, _, stderr := runTensionUpdateOver(t, seam, tensionUpdateConfig{
		id:             "ten_0123",
		label:          "Roadmap drift",
		labelSet:       true,
		meetingType:    "governance",
		meetingTypeSet: true,
	})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success\nstderr: %s", outcome, stderr)
	}
	want := `{"tension":{"label":"Roadmap drift","meeting_type":"governance"}}`
	if tr.lastBody != want {
		t.Errorf("body = %s, want %s", tr.lastBody, want)
	}
	for _, forbidden := range []string{`"body"`, `"status"`} {
		if strings.Contains(tr.lastBody, forbidden) {
			t.Errorf("body must not carry %s, got %s", forbidden, tr.lastBody)
		}
	}
}

// --- fail-fast input validation: no request --------------------------------

func TestRunTensionUpdate_NoEditableFieldUsageErrorNoRequest(t *testing.T) {
	tr := &tensionTransport{status: 200, body: tensionUpdatedBody}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, stderr := runTensionUpdateOver(t, seam, tensionUpdateConfig{id: "ten_0123"})
	if outcome != UsageError || ExitCode(outcome) != 2 {
		t.Fatalf("outcome=%v exit=%d, want UsageError/2", outcome, ExitCode(outcome))
	}
	for _, want := range []string{"--body", "--label", "--status", "--meeting-type"} {
		if !strings.Contains(stderr, want) {
			t.Errorf("stderr should name the editable flags (missing %q):\n%s", want, stderr)
		}
	}
	if tr.calls != 0 {
		t.Errorf("no editable field must be rejected before any request, got %d calls", tr.calls)
	}
	if strings.TrimSpace(stdout) != "" {
		t.Errorf("nothing should be printed on a rejected no-op, got:\n%s", stdout)
	}
}

func TestRunTensionUpdate_EmptyOnlyFlagRejectedNoRequest(t *testing.T) {
	// --label "" alone resolves to no field sent — the send-set is empty, so it is
	// rejected by the at-least-one-field precondition, NOT a no-op PATCH.
	tr := &tensionTransport{status: 200, body: tensionUpdatedBody}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, _, stderr := runTensionUpdateOver(t, seam, tensionUpdateConfig{
		id: "ten_0123", label: "", labelSet: true,
	})
	if outcome != UsageError || ExitCode(outcome) != 2 {
		t.Fatalf("outcome=%v exit=%d, want UsageError/2", outcome, ExitCode(outcome))
	}
	if !strings.Contains(stderr, "at least one") {
		t.Errorf("stderr should report the at-least-one-field precondition:\n%s", stderr)
	}
	if tr.calls != 0 {
		t.Errorf("an empty-valued only flag must send no request, got %d calls", tr.calls)
	}
}

func TestRunTensionUpdate_BlankBodyRejectedSpecificallyNoRequest(t *testing.T) {
	// A supplied whitespace-only --body gets the SPECIFIC message, not the generic
	// precondition, and sends no request.
	tr := &tensionTransport{status: 200, body: tensionUpdatedBody}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, _, stderr := runTensionUpdateOver(t, seam, tensionUpdateConfig{
		id: "ten_0123", body: "   ", bodySet: true,
	})
	if outcome != UsageError || ExitCode(outcome) != 2 {
		t.Fatalf("outcome=%v exit=%d, want UsageError/2", outcome, ExitCode(outcome))
	}
	if !strings.Contains(stderr, "--body") || !strings.Contains(stderr, "must not be empty") {
		t.Errorf("stderr should report that --body must not be empty:\n%s", stderr)
	}
	if strings.Contains(stderr, "at least one") {
		t.Errorf("a blank --body should get the specific message, not the generic precondition:\n%s", stderr)
	}
	if tr.calls != 0 {
		t.Errorf("a blank --body must be rejected before any request, got %d calls", tr.calls)
	}
}

func TestRunTensionUpdate_UnsupportedStatusNoRequest(t *testing.T) {
	tr := &tensionTransport{status: 200, body: tensionUpdatedBody}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, _, stderr := runTensionUpdateOver(t, seam, tensionUpdateConfig{
		id: "ten_0123", status: "open", statusSet: true,
	})
	if outcome != UsageError || ExitCode(outcome) != 2 {
		t.Fatalf("outcome=%v exit=%d, want UsageError/2", outcome, ExitCode(outcome))
	}
	if !strings.Contains(stderr, "open") {
		t.Errorf("stderr should name the unsupported value:\n%s", stderr)
	}
	for _, want := range []string{"unprocessed", "processed", "archived"} {
		if !strings.Contains(stderr, want) {
			t.Errorf("stderr should list the supported status set (missing %q):\n%s", want, stderr)
		}
	}
	if tr.calls != 0 {
		t.Errorf("an unsupported --status must be rejected before any request, got %d calls", tr.calls)
	}
}

func TestRunTensionUpdate_UnsupportedMeetingTypeNoRequest(t *testing.T) {
	tr := &tensionTransport{status: 200, body: tensionUpdatedBody}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, _, stderr := runTensionUpdateOver(t, seam, tensionUpdateConfig{
		id: "ten_0123", meetingType: "sync", meetingTypeSet: true,
	})
	if outcome != UsageError || ExitCode(outcome) != 2 {
		t.Fatalf("outcome=%v exit=%d, want UsageError/2", outcome, ExitCode(outcome))
	}
	if !strings.Contains(stderr, "sync") {
		t.Errorf("stderr should name the unsupported value:\n%s", stderr)
	}
	if tr.calls != 0 {
		t.Errorf("an unsupported --meeting-type must be rejected before any request, got %d calls", tr.calls)
	}
}

func TestRunTensionUpdate_BadOutputUsageErrorNoRequest(t *testing.T) {
	tr := &tensionTransport{status: 200, body: tensionUpdatedBody}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, _, _ := runTensionUpdateOver(t, seam, tensionUpdateConfig{
		id: "ten_0123", body: "X", bodySet: true, outputFlag: "xml", outputPresent: true,
	})
	if outcome != UsageError || ExitCode(outcome) != 2 {
		t.Fatalf("outcome=%v exit=%d, want UsageError/2", outcome, ExitCode(outcome))
	}
	if tr.calls != 0 {
		t.Errorf("a bad --output must be rejected before any request, got %d calls", tr.calls)
	}
}

// --- error classification (via the shared classifier) ----------------------

func TestRunTensionUpdate_NoCredentialsIsUsageError(t *testing.T) {
	tr := &tensionTransport{status: 200, body: tensionUpdatedBody}
	seam := &fakeMeSeam{ctx: noTokenContext(), transport: tr}

	outcome, stdout, stderr := runTensionUpdateOver(t, seam, tensionUpdateConfig{id: "ten_0123", body: "X", bodySet: true})
	if outcome != UsageError || ExitCode(outcome) != 2 {
		t.Fatalf("outcome=%v exit=%d, want UsageError/2", outcome, ExitCode(outcome))
	}
	if !strings.Contains(strings.ToLower(stderr), "not authenticated") {
		t.Errorf("stderr should report not authenticated:\n%s", stderr)
	}
	if strings.TrimSpace(stdout) != "" {
		t.Errorf("no tension should be printed on a credential failure, got:\n%s", stdout)
	}
}

func TestRunTensionUpdate_UnknownIDSurfacesAPIStatus(t *testing.T) {
	tr := &tensionTransport{status: 404, body: `{"detail":"Tension not found"}`}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, stderr := runTensionUpdateOver(t, seam, tensionUpdateConfig{id: "ten_ffff", body: "X", bodySet: true})
	if outcome != APIError || ExitCode(outcome) != 3 {
		t.Fatalf("an unknown id should surface APIError/3, got %v/%d\nstderr: %s", outcome, ExitCode(outcome), stderr)
	}
	if !strings.Contains(stderr, "404") {
		t.Errorf("stderr should name the HTTP status (404):\n%s", stderr)
	}
	if tr.lastIfMatch != "" {
		t.Errorf("no If-Match must be sent even on a not-found, got %q", tr.lastIfMatch)
	}
	if strings.TrimSpace(stdout) != "" {
		t.Errorf("nothing should be printed on a not-found, got:\n%s", stdout)
	}
}

// TestRunTensionUpdate_RateLimitSurfacedNotRetried pins §133: a PATCH 429 is
// surfaced on the first occurrence and never auto-retried (017's isSafeMethod
// gate), so an update cannot be silently re-sent — exactly one request.
func TestRunTensionUpdate_RateLimitSurfacedNotRetried(t *testing.T) {
	tr := &tensionTransport{status: 429, body: `{"detail":"rate limited"}`}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, _, stderr := runTensionUpdateOver(t, seam, tensionUpdateConfig{id: "ten_0123", body: "X", bodySet: true})
	if outcome != RateLimited || ExitCode(outcome) != 5 {
		t.Fatalf("outcome=%v exit=%d, want RateLimited/5\nstderr: %s", outcome, ExitCode(outcome), stderr)
	}
	if tr.calls != 1 {
		t.Errorf("a PATCH 429 must not be retried, want 1 call, got %d", tr.calls)
	}
}

func TestRunTensionUpdate_TransportErrorIsNetworkUnavailable(t *testing.T) {
	tr := &tensionTransport{netErr: errors.New("dial tcp: connection refused")}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, _, stderr := runTensionUpdateOver(t, seam, tensionUpdateConfig{id: "ten_0123", body: "X", bodySet: true})
	if outcome != NetworkUnavailable || ExitCode(outcome) != 6 {
		t.Fatalf("outcome=%v exit=%d, want NetworkUnavailable/6\nstderr: %s", outcome, ExitCode(outcome), stderr)
	}
}

// --- structured output: raw {data: Tension} verbatim -----------------------

func TestRunTensionUpdate_StructuredJSONEmitsRawPayload(t *testing.T) {
	tr := &tensionTransport{status: 200, body: tensionUpdatedBody}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, _ := runTensionUpdateOver(t, seam, tensionUpdateConfig{
		id: "ten_0123", body: "X", bodySet: true, outputFlag: "json", outputPresent: true,
	})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success", outcome)
	}
	for _, want := range []string{`"data"`, "ten_0123", `"role_id"`} {
		if !strings.Contains(stdout, want) {
			t.Errorf("structured json should carry the raw {data: Tension} payload, missing %q:\n%s", want, stdout)
		}
	}
	if strings.Contains(stdout, "Sensing role:") {
		t.Errorf("structured json must not render the human projection:\n%s", stdout)
	}
}

// --- command-level wiring --------------------------------------------------

// TestTensionUpdateCommand_AttachedToGroup pins ADR-2: `update` resolves as a leaf
// of the existing `tension` group beside create/list/get.
func TestTensionUpdateCommand_AttachedToGroup(t *testing.T) {
	seam := &fakeMeSeam{ctx: validMeContext(), transport: &tensionTransport{status: 200, body: tensionUpdatedBody}}
	root := NewRootCommand()
	MustRegister(root, newTensionCommand(seam))
	group, _, err := root.Find([]string{"tension"})
	if err != nil {
		t.Fatalf("`tension` did not resolve: %v", err)
	}
	if update, _, err := group.Find([]string{"update"}); err != nil || update.Name() != "update" {
		t.Errorf("`tension update` should resolve through the group: %v", err)
	}
}

// TestTensionUpdateCommand_SendsBodyEndToEnd pins the Changed()-gating and PATCH
// shape through a real invocation.
func TestTensionUpdateCommand_SendsBodyEndToEnd(t *testing.T) {
	tr := &tensionTransport{status: 200, body: tensionUpdatedBody}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	root := NewRootCommand()
	MustRegister(root, newTensionCommand(seam))
	var out, errb bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&errb)
	outcome, _ := Run(root, []string{"tension", "update", "ten_0123", "--label", "L", "--meeting-type", "governance"})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success\nstderr: %s", outcome, errb.String())
	}
	want := `{"tension":{"label":"L","meeting_type":"governance"}}`
	if tr.lastBody != want {
		t.Errorf("body = %s, want %s", tr.lastBody, want)
	}
}

// TestTensionUpdateCommand_RequiresExactlyOneArg pins ExactArgs(1): zero args is a
// usage error and sends no request.
func TestTensionUpdateCommand_RequiresExactlyOneArg(t *testing.T) {
	tr := &tensionTransport{status: 200, body: tensionUpdatedBody}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	root := NewRootCommand()
	MustRegister(root, newTensionCommand(seam))
	root.SetOut(&bytes.Buffer{})
	root.SetErr(&bytes.Buffer{})
	outcome, _ := Run(root, []string{"tension", "update", "--body", "X"})
	if outcome != UsageError {
		t.Errorf("zero positional args should be a UsageError, got %v", outcome)
	}
	if tr.calls != 0 {
		t.Errorf("a wrong arg count must send no request, got %d calls", tr.calls)
	}
}

// TestTensionUpdateCommand_EditFlagsRejectedOnGet pins the structural guard: the
// editable flags live only on `update`, so passing --status to `get` is a cobra
// unknown-flag usage error with no request.
func TestTensionUpdateCommand_EditFlagsRejectedOnGet(t *testing.T) {
	tr := &tensionTransport{status: 200, body: tensionUpdatedBody}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	root := NewRootCommand()
	MustRegister(root, newTensionCommand(seam))
	root.SetOut(&bytes.Buffer{})
	root.SetErr(&bytes.Buffer{})
	outcome, _ := Run(root, []string{"tension", "get", "ten_0123", "--status", "archived"})
	if outcome != UsageError {
		t.Errorf("an editable flag on `get` should be a cobra usage error, got %v", outcome)
	}
	if tr.calls != 0 {
		t.Errorf("an unknown flag must send no request, got %d calls", tr.calls)
	}
}
