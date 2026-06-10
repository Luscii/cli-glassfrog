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
	wantProduced := []Outcome{Success, UsageError, RuntimeError, NetworkUnavailable, APIError, PermissionError, RateLimited}
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

// reportClientError delegates to Diagnose: the stderr line it prints is exactly
// renderDiagnostic(Diagnose(refined)) and the returned Outcome is the
// Diagnostic's Category, computed from the same refined value.
func TestReportClientError_Delegates(t *testing.T) {
	// A bare *ResponseError is refined to a *ProblemError inside reportClientError,
	// so both the message and the category must reflect the refined value.
	err := &apiclient.ResponseError{StatusCode: 403}
	var buf bytes.Buffer
	gotOutcome, _ := reportClientError(&buf, err)

	wantLine := "the API returned a non-2xx response: status 403 — check that the configured identity has the required role membership / permission\n"
	if buf.String() != wantLine {
		t.Errorf("reportClientError stderr = %q, want %q", buf.String(), wantLine)
	}
	if gotOutcome != PermissionError {
		t.Errorf("reportClientError outcome = %v, want PermissionError", gotOutcome)
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
