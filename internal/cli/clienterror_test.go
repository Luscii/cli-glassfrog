package cli

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/Luscii/cli-glassfrog/internal/apiclient"
	"github.com/Luscii/cli-glassfrog/internal/auth"
	"github.com/Luscii/cli-glassfrog/internal/rcfile"
)

// The classifier's contract is a table: each API-client error → its Outcome
// category (plan ADR-4 / interface-spec Error Communication). Asserting every
// row pins the mapping; the exhaustiveness guard below pins that the table
// covers every category the classifier can produce, so a dropped or added arm
// fails loud rather than silently (PR #10 LEARNINGS).
func TestClassifyClientError_Table(t *testing.T) {
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
		{"unknown-error-is-runtime-failsafe", errors.New("something unexpected"), RuntimeError},
	}

	produced := map[Outcome]bool{}
	for _, tc := range cases {
		got := classifyClientError(tc.err)
		if got != tc.want {
			t.Errorf("classifyClientError(%s) = %v, want %v", tc.name, got, tc.want)
		}
		produced[got] = true
	}

	// Exhaustiveness guard: the table must exercise every Outcome the classifier
	// can produce. If a future edit adds a category but no row, or drops a
	// category's only row, this list and the produced set diverge and the test
	// fails loud (the comma-ok half: a category missing from `produced` is named
	// explicitly rather than passing silently).
	wantProduced := []Outcome{Success, UsageError, RuntimeError, NetworkUnavailable, APIError, PermissionError, RateLimited}
	if len(produced) != len(wantProduced) {
		t.Errorf("classifier produced %d distinct outcomes across the table, want %d (a category lost or gained coverage)", len(produced), len(wantProduced))
	}
	for _, o := range wantProduced {
		if !produced[o] {
			t.Errorf("no table row produces %v — the classifier's coverage of that category is untested", o)
		}
	}
}

// The AuthError arm must be discriminated before the rcfile arms. In production a
// CredentialError IS an *AuthError whose Unwrap() exposes the rcfile read/format
// error the credential resolver returned — so errors.As can see BOTH an
// *AuthError and an *rcfile.ReadError in the same chain. If the classifier
// checked the rcfile arms first, that error would be mislabelled a base-URL
// UsageError instead of the RuntimeError a malformed credentials file warrants.
//
// Build the REAL shape via the authenticated-transport path (the same way Execute
// surfaces it): a context carrying an rcfile CredErr, run through NewClient +
// Execute, whose auth fail-safe wraps the rcfile error as *AuthError{CredentialError}
// before any request is sent. This is what makes the test meaningful — a
// nil-cause AuthError (the earlier version) matched neither rcfile arm and passed
// trivially regardless of ordering.
func TestClassifyClientError_AuthBeforeRcfile(t *testing.T) {
	rcErr := &rcfile.ReadError{Path: "/home/u/.glassfrogrc", Err: errors.New("permission denied")}
	ctx := apiclient.ConnectionContext{
		BaseURL: apiclient.BaseURL{Value: "https://example.test/api/v5", Source: apiclient.SourceFlag},
		Cred:    auth.Resolution{Source: auth.SourceNone},
		CredErr: rcErr, // AuthTransport wraps this as *AuthError{CredentialError}
	}
	// The base transport is required by NewClient but never reached — the auth
	// fail-safe fires before any send.
	client, err := apiclient.NewClient(ctx, &cannedTransport{})
	if err != nil {
		t.Fatalf("NewClient errored: %v", err)
	}
	_, execErr := client.Execute(context.Background(), apiclient.Request{Method: http.MethodGet, Path: "/me"}, nil)
	if execErr == nil {
		t.Fatal("Execute should surface the credential fail-safe error")
	}

	// The chain genuinely matches BOTH types — without this overlap the ordering
	// test would be vacuous.
	var authErr *apiclient.AuthError
	var rcReadErr *rcfile.ReadError
	if !errors.As(execErr, &authErr) {
		t.Fatalf("error chain should contain an *AuthError, got %v", execErr)
	}
	if !errors.As(execErr, &rcReadErr) {
		t.Fatalf("error chain should ALSO unwrap to the *rcfile.ReadError, got %v", execErr)
	}

	if got := classifyClientError(execErr); got != RuntimeError {
		t.Errorf("classifyClientError = %v, want RuntimeError (AuthError must win over the rcfile arms)", got)
	}
}
