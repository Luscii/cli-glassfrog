package cli

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/Luscii/cli-glassfrog/internal/apiclient"
	"github.com/Luscii/cli-glassfrog/internal/output"
	"github.com/Luscii/cli-glassfrog/internal/rcfile"
)

// Diagnose is the single source of truth for an API-client error's category. Its
// table pins the error→category mapping for every family; the exhaustiveness
// guard (len + comma-ok) pins that the table covers every category Diagnose can
// produce, so a dropped or added arm fails loud rather than silently (PR #10
// LEARNINGS). classifyClientError now delegates here, so this also pins the
// delegate.
func TestDiagnose_Category_Table(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want Outcome
	}{
		{"nil-is-success", nil, Success},
		{"no-credentials-is-usage", &apiclient.AuthError{Kind: apiclient.NoCredentials}, UsageError},
		{"credential-error-is-runtime", &apiclient.AuthError{Kind: apiclient.CredentialError}, RuntimeError},
		{"transport-is-network-unavailable", &apiclient.TransportError{}, NetworkUnavailable},
		{"response-404-is-api-error", &apiclient.ResponseError{StatusCode: 404}, APIError},
		{"response-500-is-api-error", &apiclient.ResponseError{StatusCode: 500}, APIError},
		{"response-401-is-permission", &apiclient.ResponseError{StatusCode: 401}, PermissionError},
		{"response-403-is-permission", &apiclient.ResponseError{StatusCode: 403}, PermissionError},
		{"response-429-is-rate-limited", &apiclient.ResponseError{StatusCode: 429}, RateLimited},
		{"problem-401-is-permission", apiclient.ExtractProblem(&apiclient.ResponseError{StatusCode: 401}), PermissionError},
		{"problem-403-is-permission", apiclient.ExtractProblem(&apiclient.ResponseError{StatusCode: 403}), PermissionError},
		{"problem-429-is-rate-limited", apiclient.ExtractProblem(&apiclient.ResponseError{StatusCode: 429}), RateLimited},
		{"problem-404-is-api-error", apiclient.ExtractProblem(&apiclient.ResponseError{StatusCode: 404}), APIError},
		{"response-412-is-stale-write", &apiclient.ResponseError{StatusCode: 412}, StaleWrite},
		{"problem-412-is-stale-write", apiclient.ExtractProblem(&apiclient.ResponseError{StatusCode: 412}), StaleWrite},
		{"decode-is-api-error", &apiclient.DecodeError{StatusCode: 200}, APIError}, // 031 ADR-2: was RuntimeError
		{"base-url-error-is-usage", &apiclient.BaseURLError{Source: "--base-url"}, UsageError},
		{"rcfile-read-error-is-usage", &rcfile.ReadError{}, UsageError},
		{"rcfile-format-error-is-usage", &rcfile.FormatError{}, UsageError},
		{"format-error-is-usage", &output.FormatError{}, UsageError},
		{"unknown-error-is-runtime-failsafe", errors.New("something unexpected"), RuntimeError},
	}

	produced := map[Outcome]bool{}
	for _, tc := range cases {
		got := Diagnose(tc.err).Category
		if got != tc.want {
			t.Errorf("Diagnose(%s).Category = %v, want %v", tc.name, got, tc.want)
		}
		// The delegate must agree with Diagnose for every row.
		if c := classifyClientError(tc.err); c != got {
			t.Errorf("classifyClientError(%s) = %v, but Diagnose(...).Category = %v — the delegate drifted", tc.name, c, got)
		}
		produced[got] = true
	}

	// Exhaustiveness guard: the table must exercise every Outcome Diagnose can
	// produce. If a future edit adds a category but no row, or drops a category's
	// only row, this list and the produced set diverge and the test fails loud
	// (the comma-ok half names a missing category explicitly).
	wantProduced := []Outcome{Success, UsageError, RuntimeError, NetworkUnavailable, APIError, PermissionError, RateLimited, StaleWrite}
	if len(produced) != len(wantProduced) {
		t.Errorf("Diagnose produced %d distinct categories across the table, want %d (a category lost or gained coverage)", len(produced), len(wantProduced))
	}
	for _, o := range wantProduced {
		if !produced[o] {
			t.Errorf("no table row produces %v — Diagnose's coverage of that category is untested", o)
		}
	}
}

// renderDiagnostic(Diagnose(err)) must reproduce the pre-consolidation
// formatClientErrorMessage(err) byte-for-byte for every failure family — the
// human stderr surface does not drift on the refactor. These golden strings were
// captured from formatClientErrorMessage before it was folded into Diagnose.
func TestRenderDiagnostic_ByteEquivalence(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want string
	}{
		{
			"auth-nocred",
			&apiclient.AuthError{Kind: apiclient.NoCredentials},
			"not authenticated — run `glassfrog auth login` or set GLASSFROG_TOKEN",
		},
		{
			"auth-cred",
			&apiclient.AuthError{Kind: apiclient.CredentialError},
			"cannot authenticate: <nil> — fix or re-create the credentials file with `glassfrog auth login`",
		},
		{
			"transport",
			&apiclient.TransportError{},
			"request failed: <nil> — check connectivity; the API may be unreachable",
		},
		{
			"problem-401",
			apiclient.ExtractProblem(&apiclient.ResponseError{StatusCode: 401}),
			"the API returned a non-2xx response: status 401 — verify the configured API token",
		},
		{
			"problem-403",
			apiclient.ExtractProblem(&apiclient.ResponseError{StatusCode: 403}),
			"the API returned a non-2xx response: status 403 — check that the configured identity has the required role membership / permission",
		},
		{
			"problem-429",
			apiclient.ExtractProblem(&apiclient.ResponseError{StatusCode: 429}),
			"the API returned a non-2xx response: status 429 — wait for the rate-limit window to reset (per the `Retry-After` / `X-RateLimit-Reset` headers) and retry",
		},
		{
			"problem-404",
			apiclient.ExtractProblem(&apiclient.ResponseError{StatusCode: 404}),
			"the API returned a non-2xx response: status 404 — the API rejected the read; check that the token has access and retry, or consult the status code",
		},
		{
			"problem-500-detail",
			&apiclient.ProblemError{StatusCode: 500, Detail: "boom", Title: "Server"},
			"the API returned a non-2xx response: status 500: Server: boom — the API rejected the read; check that the token has access and retry, or consult the status code",
		},
		{
			"response-404",
			&apiclient.ResponseError{StatusCode: 404},
			"the API returned a non-2xx response: status 404 — the API rejected the read; check that the token has access and retry, or consult the status code",
		},
		{
			"decode",
			&apiclient.DecodeError{StatusCode: 200},
			"the API response did not match the expected shape — this may be an API change; report it (decoding the 200 response body failed: <nil>)",
		},
		{
			"baseurl",
			&apiclient.BaseURLError{Source: "--base-url"},
			"base URL from --base-url is not a valid absolute http(s) URL — correct --base-url, GLASSFROG_BASE_URL, or the .glassfrogrc base_url",
		},
		{
			"rcread",
			&rcfile.ReadError{Path: "/x"},
			"settings file /x could not be read: <nil> — correct --base-url, GLASSFROG_BASE_URL, or the .glassfrogrc base_url",
		},
		{
			"rcformat",
			&rcfile.FormatError{Path: "/x"},
			"settings file /x is malformed: a non-comment line is not a key=value pair — correct --base-url, GLASSFROG_BASE_URL, or the .glassfrogrc base_url",
		},
		{
			"format",
			&output.FormatError{},
			`unsupported output value "" from  — supported: full, compact, json, yaml`,
		},
	}
	for _, tc := range cases {
		got := renderDiagnostic(Diagnose(tc.err))
		if got != tc.want {
			t.Errorf("renderDiagnostic(Diagnose(%s)):\n got %q\nwant %q", tc.name, got, tc.want)
		}
	}
}

// reportFailure delegates to Diagnose: on the human path (unchanged by 032) the
// stderr line it prints is exactly renderDiagnostic(Diagnose(refined)), stdout
// stays empty, and the returned Outcome is the Diagnostic's Category, computed
// from the same refined value.
func TestReportFailure_Delegates(t *testing.T) {
	// A bare *ResponseError is refined to a *ProblemError inside reportFailure,
	// so both the message and the category must reflect the refined value.
	err := &apiclient.ResponseError{StatusCode: 403}
	var outb, buf bytes.Buffer
	gotOutcome, _ := reportFailure(&outb, &buf, output.FormatFull, err)

	wantLine := "the API returned a non-2xx response: status 403 — check that the configured identity has the required role membership / permission\n"
	if buf.String() != wantLine {
		t.Errorf("reportFailure stderr = %q, want %q", buf.String(), wantLine)
	}
	if outb.Len() != 0 {
		t.Errorf("the human path must leave stdout empty, got %q", outb.String())
	}
	if gotOutcome != PermissionError {
		t.Errorf("reportFailure outcome = %v, want PermissionError", gotOutcome)
	}
}

// Stale-Write Surfacing (054): a 412 is branched out of the generic APIError
// bucket into its own StaleWrite category (→ code 7) with a re-read/retry next
// step and a cause that names the precondition failure. This pins all four
// observable fields of the diagnostic for both 412 shapes: the API supplied its
// own detail (its words win, unchanged problemCause) and the API supplied none
// (a status-derived precondition-failure cause, not the bare "status 412" line).
func TestDiagnose_StaleWrite_412(t *testing.T) {
	t.Run("api-supplied-detail", func(t *testing.T) {
		err := apiclient.ExtractProblem(&apiclient.ResponseError{
			StatusCode: 412,
			Body:       []byte(`{"detail":"version mismatch on tension ten_123"}`),
		})
		d := Diagnose(err)
		if d.Category != StaleWrite {
			t.Errorf("category = %v, want StaleWrite", d.Category)
		}
		if ExitCode(d.Category) != 7 {
			t.Errorf("ExitCode(%v) = %d, want 7", d.Category, ExitCode(d.Category))
		}
		// The API's own words win — its detail is surfaced verbatim.
		if !strings.Contains(d.Cause, "version mismatch on tension ten_123") {
			t.Errorf("cause %q does not surface the API's own detail", d.Cause)
		}
		// The next step is the re-read/retry recovery, never the generic token step.
		lower := strings.ToLower(d.NextStep)
		if !strings.Contains(lower, "re-read") || !strings.Contains(lower, "retry") {
			t.Errorf("next step %q does not tell the operator to re-read and retry", d.NextStep)
		}
		if strings.Contains(lower, "token has access") {
			t.Errorf("next step %q must not be the generic check-token-access step", d.NextStep)
		}
	})

	t.Run("no-readable-detail", func(t *testing.T) {
		// An empty body forces ExtractProblem to synthesize (DetailSynthesized=true).
		err := apiclient.ExtractProblem(&apiclient.ResponseError{StatusCode: 412, Body: []byte("")})
		d := Diagnose(err)
		if d.Category != StaleWrite {
			t.Errorf("category = %v, want StaleWrite", d.Category)
		}
		// The cause is derived from the 412 status (not invented): it names the
		// precondition failure / changed-since-read, not the bare generic line.
		if !strings.Contains(d.Cause, "412") {
			t.Errorf("cause %q is not derived from the 412 status", d.Cause)
		}
		lower := strings.ToLower(d.Cause)
		if !strings.Contains(lower, "precondition") && !strings.Contains(lower, "changed since it was read") {
			t.Errorf("cause %q does not name the precondition failure / changed-since-read", d.Cause)
		}
		if strings.Contains(d.Cause, "the API returned a non-2xx response: status 412") && !strings.Contains(lower, "precondition") {
			t.Errorf("cause %q is the bare generic status line — it must be 412-aware", d.Cause)
		}
		// The category, code, and re-read next step are still assigned for the
		// synthesized shape.
		if ExitCode(d.Category) != 7 {
			t.Errorf("ExitCode(%v) = %d, want 7", d.Category, ExitCode(d.Category))
		}
		if !strings.Contains(strings.ToLower(d.NextStep), "re-read") {
			t.Errorf("next step %q does not tell the operator to re-read", d.NextStep)
		}
	})
}

// No existing status's surfacing drifts when the 412 arm is added: 401/403/404/
// 429/500 keep their exact category, exit code, cause, and next step. This pins
// the no-drift guarantee the additive 412 case must preserve (the @validation
// "No existing surfacing drifts" scenario's mechanism, held out at the BDD level).
func TestDiagnose_412_DoesNotDriftOtherStatuses(t *testing.T) {
	cases := []struct {
		status   int
		category Outcome
		code     int
	}{
		{401, PermissionError, 4},
		{403, PermissionError, 4},
		{404, APIError, 3},
		{429, RateLimited, 5},
		{500, APIError, 3},
	}
	for _, tc := range cases {
		err := apiclient.ExtractProblem(&apiclient.ResponseError{StatusCode: tc.status})
		d := Diagnose(err)
		if d.Category != tc.category {
			t.Errorf("status %d: category = %v, want %v (drifted)", tc.status, d.Category, tc.category)
		}
		if got := ExitCode(d.Category); got != tc.code {
			t.Errorf("status %d: exit code = %d, want %d (drifted)", tc.status, got, tc.code)
		}
		// None of the unchanged statuses may inherit the stale-write surfacing.
		if d.Category == StaleWrite {
			t.Errorf("status %d must not be classified as a stale write", tc.status)
		}
		if strings.Contains(strings.ToLower(d.NextStep), "re-read") {
			t.Errorf("status %d next step %q must not carry the 412 re-read hint", tc.status, d.NextStep)
		}
	}
}

// No Diagnose arm may emit the X-Auth-Token: every Cause, every NextStep, and the
// rendered line is response/path/status only. The sentinel is the real token a
// production seam would carry on the request header (meSecretToken) — it must
// appear in none of the three observable surfaces for any family (mirrors
// 010/015's token-free tests).
func TestDiagnose_NeverEmitsToken(t *testing.T) {
	arms := []struct {
		name string
		err  error
	}{
		{"auth-nocred", &apiclient.AuthError{Kind: apiclient.NoCredentials}},
		{"auth-cred", &apiclient.AuthError{Kind: apiclient.CredentialError}},
		{"transport", &apiclient.TransportError{}},
		{"problem-401", apiclient.ExtractProblem(&apiclient.ResponseError{StatusCode: 401})},
		{"problem-403", apiclient.ExtractProblem(&apiclient.ResponseError{StatusCode: 403})},
		{"problem-429", apiclient.ExtractProblem(&apiclient.ResponseError{StatusCode: 429})},
		{"problem-404", apiclient.ExtractProblem(&apiclient.ResponseError{StatusCode: 404})},
		{"problem-412", apiclient.ExtractProblem(&apiclient.ResponseError{StatusCode: 412})},
		{"response-500", &apiclient.ResponseError{StatusCode: 500}},
		{"decode", &apiclient.DecodeError{StatusCode: 200}},
		{"baseurl", &apiclient.BaseURLError{Source: "--base-url"}},
		{"rcread", &rcfile.ReadError{Path: "/x"}},
		{"rcformat", &rcfile.FormatError{Path: "/x"}},
		{"format", &output.FormatError{}},
		{"failsafe", errors.New("some unexpected failure")},
	}
	for _, a := range arms {
		d := Diagnose(a.err)
		for field, val := range map[string]string{
			"Cause":    d.Cause,
			"NextStep": d.NextStep,
			"rendered": renderDiagnostic(d),
		} {
			if strings.Contains(val, meSecretToken) {
				t.Errorf("Diagnose(%s).%s leaked the token: %q", a.name, field, val)
			}
		}
	}
}
