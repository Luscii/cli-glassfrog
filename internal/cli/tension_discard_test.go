package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/Luscii/cli-glassfrog/internal/apiclient"
)

// runTensionDiscardOver drives the pure runTensionDiscard over a fake seam,
// returning the outcome and captured stdout/stderr, and failing if the token leaks.
func runTensionDiscardOver(t *testing.T, seam tensionSeam, cfg tensionDiscardConfig) (Outcome, string, string) {
	t.Helper()
	var out, errb bytes.Buffer
	cfg.seam = seam
	cfg.reqCtx = context.Background()
	cfg.stdout = &out
	cfg.stderr = &errb
	outcome, _ := runTensionDiscard(cfg)
	if strings.Contains(out.String()+errb.String(), meSecretToken) {
		t.Fatalf("the token leaked into output: %q", out.String()+errb.String())
	}
	return outcome, out.String(), errb.String()
}

// --- happy path (204): one bodyless DELETE, synthesized result, stderr advisory --

func TestRunTensionDiscard_LiveTensionBodylessDelete(t *testing.T) {
	tr := &tensionTransport{status: http.StatusNoContent, body: ""}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, stderr := runTensionDiscardOver(t, seam, tensionDiscardConfig{id: "ten_0123"})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success\nstderr: %s", outcome, stderr)
	}
	if tr.calls != 1 {
		t.Fatalf("a discard is exactly one request, got %d", tr.calls)
	}
	if tr.lastMethod != http.MethodDelete {
		t.Errorf("method = %q, want DELETE", tr.lastMethod)
	}
	if !strings.HasSuffix(tr.lastPath, "/tensions/ten_0123") {
		t.Errorf("path = %q, want /tensions/ten_0123", tr.lastPath)
	}
	if tr.lastBody != "" {
		t.Errorf("a bodyless DELETE must send no body, got %q", tr.lastBody)
	}
	if tr.lastContentType != "" {
		t.Errorf("a bodyless DELETE must send no Content-Type, got %q", tr.lastContentType)
	}
	if tr.lastIfMatch != "" {
		t.Errorf("no If-Match must be sent (last-write-wins), got %q", tr.lastIfMatch)
	}
	if strings.TrimSpace(stdout) != "ten_0123  [discarded]" {
		t.Errorf("stdout should carry the synthesized confirmation line, got:\n%q", stdout)
	}
	if !strings.Contains(stderr, "discarded tension ten_0123") {
		t.Errorf("stderr should note the tension was discarded:\n%s", stderr)
	}
}

// --- 404 as success: identical stdout, "already gone" advisory, no error --------

func TestRunTensionDiscard_AlreadyGoneIsSuccess(t *testing.T) {
	tr := &tensionTransport{status: http.StatusNotFound, body: `{"detail":"Tension not found"}`}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, stderr := runTensionDiscardOver(t, seam, tensionDiscardConfig{id: "ten_0123"})
	if outcome != Success || ExitCode(outcome) != 0 {
		t.Fatalf("a 404 must be folded into success (exit 0), got %v/%d\nstderr: %s", outcome, ExitCode(outcome), stderr)
	}
	if tr.calls != 1 {
		t.Fatalf("a discard is exactly one request, got %d", tr.calls)
	}
	if strings.TrimSpace(stdout) != "ten_0123  [discarded]" {
		t.Errorf("stdout should be IDENTICAL to the 204 path, got:\n%q", stdout)
	}
	if !strings.Contains(stderr, "already discarded") {
		t.Errorf("stderr should note the tension was already gone:\n%s", stderr)
	}
	// No not-found error must leak: not the status, not "not found".
	if strings.Contains(stderr, "404") || strings.Contains(strings.ToLower(stderr), "not found") {
		t.Errorf("a 404-as-success must leak no not-found error to stderr:\n%s", stderr)
	}
}

// TestRunTensionDiscard_204And404StdoutIdentical pins ADR-3/ADR-4: the synthesized
// stdout result is byte-identical whether the API answered 204 or 404 — the
// distinction rides only the stderr advisory.
func TestRunTensionDiscard_204And404StdoutIdentical(t *testing.T) {
	run := func(status int) string {
		tr := &tensionTransport{status: status, body: ""}
		seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}
		_, stdout, _ := runTensionDiscardOver(t, seam, tensionDiscardConfig{id: "ten_0123"})
		return stdout
	}
	if got204, got404 := run(http.StatusNoContent), run(http.StatusNotFound); got204 != got404 {
		t.Errorf("stdout must be identical for 204 and 404:\n204: %q\n404: %q", got204, got404)
	}
}

// --- structured output: synthesized {data:{id,discarded}} -----------------------

func TestRunTensionDiscard_StructuredJSON(t *testing.T) {
	tr := &tensionTransport{status: http.StatusNoContent, body: ""}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, stderr := runTensionDiscardOver(t, seam, tensionDiscardConfig{
		id: "ten_0123", outputFlag: "json", outputPresent: true,
	})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success\nstderr: %s", outcome, stderr)
	}
	var doc struct {
		Data struct {
			ID        string `json:"id"`
			Discarded bool   `json:"discarded"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(stdout), &doc); err != nil {
		t.Fatalf("stdout should be valid JSON, got error %v:\n%s", err, stdout)
	}
	if doc.Data.ID != "ten_0123" || !doc.Data.Discarded {
		t.Errorf(`-o json should emit {"data":{"id":"ten_0123","discarded":true}}, got:\n%s`, stdout)
	}
	// The synthesized result claims nothing the server did not return.
	if strings.Contains(stdout, "discarded_at") {
		t.Errorf("the result must carry no server-owned field (discarded_at):\n%s", stdout)
	}
}

// TestRunTensionDiscard_StructuredJSON_404Identical pins that the structured result
// is identical for the 404-as-success path.
func TestRunTensionDiscard_StructuredJSON_404Identical(t *testing.T) {
	run := func(status int) string {
		tr := &tensionTransport{status: status, body: ""}
		seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}
		_, stdout, _ := runTensionDiscardOver(t, seam, tensionDiscardConfig{
			id: "ten_0123", outputFlag: "json", outputPresent: true,
		})
		return stdout
	}
	if got204, got404 := run(http.StatusNoContent), run(http.StatusNotFound); got204 != got404 {
		t.Errorf("structured stdout must be identical for 204 and 404:\n204: %q\n404: %q", got204, got404)
	}
}

// --- fail-fast: bad --output is a usage error with NO request -------------------

func TestRunTensionDiscard_BadOutputUsageErrorNoRequest(t *testing.T) {
	tr := &tensionTransport{status: http.StatusNoContent, body: ""}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, _ := runTensionDiscardOver(t, seam, tensionDiscardConfig{
		id: "ten_0123", outputFlag: "xml", outputPresent: true,
	})
	if outcome != UsageError || ExitCode(outcome) != 2 {
		t.Fatalf("outcome=%v exit=%d, want UsageError/2", outcome, ExitCode(outcome))
	}
	if tr.calls != 0 {
		t.Errorf("a bad --output must be rejected before any request, got %d calls", tr.calls)
	}
	if strings.TrimSpace(stdout) != "" {
		t.Errorf("nothing should be printed on a rejected --output, got:\n%s", stdout)
	}
}

// --- error classification (only 404 is folded into success) ---------------------

func TestRunTensionDiscard_NoCredentialsIsUsageError(t *testing.T) {
	tr := &tensionTransport{status: http.StatusNoContent, body: ""}
	seam := &fakeMeSeam{ctx: noTokenContext(), transport: tr}

	outcome, stdout, stderr := runTensionDiscardOver(t, seam, tensionDiscardConfig{id: "ten_0123"})
	if outcome != UsageError || ExitCode(outcome) != 2 {
		t.Fatalf("outcome=%v exit=%d, want UsageError/2\nstderr: %s", outcome, ExitCode(outcome), stderr)
	}
	if !strings.Contains(strings.ToLower(stderr), "not authenticated") {
		t.Errorf("stderr should report not authenticated:\n%s", stderr)
	}
	if strings.TrimSpace(stdout) != "" {
		t.Errorf("nothing should be printed on a credential failure, got:\n%s", stdout)
	}
}

// TestRunTensionDiscard_ForbiddenIsPermissionError pins that a 403 on an existing
// tension is NEVER swallowed — only 404 is folded into success (ADR-2).
func TestRunTensionDiscard_ForbiddenIsPermissionError(t *testing.T) {
	tr := &tensionTransport{status: http.StatusForbidden, body: `{"detail":"forbidden"}`}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, stderr := runTensionDiscardOver(t, seam, tensionDiscardConfig{id: "ten_0123"})
	if outcome != PermissionError || ExitCode(outcome) != 4 {
		t.Fatalf("a 403 should surface PermissionError/4, got %v/%d\nstderr: %s", outcome, ExitCode(outcome), stderr)
	}
	if !strings.Contains(stderr, "403") {
		t.Errorf("stderr should name the HTTP status (403):\n%s", stderr)
	}
	if strings.TrimSpace(stdout) != "" {
		t.Errorf("nothing should be printed on a permission failure, got:\n%s", stdout)
	}
}

// TestRunTensionDiscard_OtherNon2xxIsAPIError pins that a non-404 4xx/5xx routes
// through the shared classifier (here a 500 → APIError/3), not into success.
func TestRunTensionDiscard_OtherNon2xxIsAPIError(t *testing.T) {
	tr := &tensionTransport{status: http.StatusInternalServerError, body: `{"detail":"boom"}`}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, _, stderr := runTensionDiscardOver(t, seam, tensionDiscardConfig{id: "ten_0123"})
	if outcome != APIError || ExitCode(outcome) != 3 {
		t.Fatalf("a 500 should surface APIError/3, got %v/%d\nstderr: %s", outcome, ExitCode(outcome), stderr)
	}
	if !strings.Contains(stderr, "500") {
		t.Errorf("stderr should name the HTTP status (500):\n%s", stderr)
	}
}

// TestRunTensionDiscard_RateLimitSurfacedNotRetried pins §133: a DELETE 429 is
// surfaced on the first occurrence and never auto-retried (017's isSafeMethod
// gate), so a discard cannot be silently re-sent — exactly one request.
func TestRunTensionDiscard_RateLimitSurfacedNotRetried(t *testing.T) {
	tr := &tensionTransport{status: http.StatusTooManyRequests, body: `{"detail":"rate limited"}`}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, _, stderr := runTensionDiscardOver(t, seam, tensionDiscardConfig{id: "ten_0123"})
	if outcome != RateLimited || ExitCode(outcome) != 5 {
		t.Fatalf("outcome=%v exit=%d, want RateLimited/5\nstderr: %s", outcome, ExitCode(outcome), stderr)
	}
	if tr.calls != 1 {
		t.Errorf("a DELETE 429 must not be retried, want 1 call, got %d", tr.calls)
	}
}

func TestRunTensionDiscard_TransportErrorIsNetworkUnavailable(t *testing.T) {
	tr := &tensionTransport{netErr: errors.New("dial tcp: connection refused")}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, _, stderr := runTensionDiscardOver(t, seam, tensionDiscardConfig{id: "ten_0123"})
	if outcome != NetworkUnavailable || ExitCode(outcome) != 6 {
		t.Fatalf("outcome=%v exit=%d, want NetworkUnavailable/6\nstderr: %s", outcome, ExitCode(outcome), stderr)
	}
}

// --- isNotFound: keys on the EXACT 404 status of a *ResponseError ---------------

func TestIsNotFound_OnlyExact404(t *testing.T) {
	for _, status := range []int{http.StatusUnauthorized, http.StatusForbidden, http.StatusTooManyRequests, http.StatusInternalServerError} {
		if isNotFound(&apiclient.ResponseError{StatusCode: status}) {
			t.Errorf("isNotFound must not match status %d", status)
		}
	}
	if !isNotFound(&apiclient.ResponseError{StatusCode: http.StatusNotFound}) {
		t.Error("isNotFound should match an exact 404 *ResponseError")
	}
	// errors.As walks the chain, so a wrapped ResponseError is still found.
	if !isNotFound(fmt.Errorf("wrapped: %w", &apiclient.ResponseError{StatusCode: http.StatusNotFound})) {
		t.Error("isNotFound should match a wrapped 404 *ResponseError")
	}
	if isNotFound(errors.New("a plain error")) {
		t.Error("isNotFound must not match a non-ResponseError")
	}
}

// --- command-level wiring -------------------------------------------------------

// TestTensionDiscardCommand_AttachedToGroup pins ADR-1: `discard` resolves as a leaf
// of the existing `tension` group beside create/list/get/update.
func TestTensionDiscardCommand_AttachedToGroup(t *testing.T) {
	seam := &fakeMeSeam{ctx: validMeContext(), transport: &tensionTransport{status: http.StatusNoContent}}
	root := NewRootCommand()
	MustRegister(root, newTensionCommand(seam))
	group, _, err := root.Find([]string{"tension"})
	if err != nil {
		t.Fatalf("`tension` did not resolve: %v", err)
	}
	if discard, _, err := group.Find([]string{"discard"}); err != nil || discard.Name() != "discard" {
		t.Errorf("`tension discard` should resolve through the group: %v", err)
	}
	// The other leaves remain reachable (the group was extended, not redefined).
	for _, leaf := range []string{"create", "list", "get", "update"} {
		if c, _, err := group.Find([]string{leaf}); err != nil || c.Name() != leaf {
			t.Errorf("`tension %s` should still resolve through the group: %v", leaf, err)
		}
	}
}

// TestTensionDiscardCommand_BodylessDeleteEndToEnd pins the bodyless DELETE shape
// and the synthesized result through a real invocation.
func TestTensionDiscardCommand_BodylessDeleteEndToEnd(t *testing.T) {
	tr := &tensionTransport{status: http.StatusNoContent, body: ""}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	root := NewRootCommand()
	MustRegister(root, newTensionCommand(seam))
	var out, errb bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&errb)
	outcome, _ := Run(root, []string{"tension", "discard", "ten_0123"})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success\nstderr: %s", outcome, errb.String())
	}
	if tr.lastMethod != http.MethodDelete || tr.lastBody != "" || tr.lastContentType != "" {
		t.Errorf("expected a bodyless DELETE with no Content-Type, got method=%q body=%q ct=%q", tr.lastMethod, tr.lastBody, tr.lastContentType)
	}
	if !strings.Contains(out.String(), "ten_0123  [discarded]") {
		t.Errorf("the synthesized confirmation should be printed:\n%s", out.String())
	}
}

// TestTensionDiscardCommand_RequiresExactlyOneArg pins ExactArgs(1): zero or more
// than one positional id is a usage error and sends no request.
func TestTensionDiscardCommand_RequiresExactlyOneArg(t *testing.T) {
	for _, args := range [][]string{
		{"tension", "discard"},
		{"tension", "discard", "ten_0123", "ten_0456"},
	} {
		tr := &tensionTransport{status: http.StatusNoContent}
		seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}
		root := NewRootCommand()
		MustRegister(root, newTensionCommand(seam))
		root.SetOut(&bytes.Buffer{})
		root.SetErr(&bytes.Buffer{})
		outcome, _ := Run(root, args)
		if outcome != UsageError {
			t.Errorf("%v should be a UsageError, got %v", args, outcome)
		}
		if tr.calls != 0 {
			t.Errorf("%v: a wrong arg count must send no request, got %d calls", args, tr.calls)
		}
	}
}

// TestTensionDiscardCommand_RejectsFieldFlag pins the structural guard: discard has
// no editable-field flags, so a stray --body is a cobra unknown-flag usage error
// with no request.
func TestTensionDiscardCommand_RejectsFieldFlag(t *testing.T) {
	tr := &tensionTransport{status: http.StatusNoContent}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}
	root := NewRootCommand()
	MustRegister(root, newTensionCommand(seam))
	root.SetOut(&bytes.Buffer{})
	root.SetErr(&bytes.Buffer{})
	outcome, _ := Run(root, []string{"tension", "discard", "ten_0123", "--body", "X"})
	if outcome != UsageError {
		t.Errorf("a stray field flag should be a cobra usage error, got %v", outcome)
	}
	if tr.calls != 0 {
		t.Errorf("an unknown flag must send no request, got %d calls", tr.calls)
	}
}
