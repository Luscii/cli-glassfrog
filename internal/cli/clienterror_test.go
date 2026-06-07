package cli

import (
	"errors"
	"fmt"
	"testing"

	"github.com/Luscii/cli-glassfrog/internal/apiclient"
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
		{"response-is-api-error", &apiclient.ResponseError{StatusCode: 404}, APIError},
		{"decode-is-runtime", &apiclient.DecodeError{StatusCode: 200}, RuntimeError},
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
	wantProduced := []Outcome{Success, UsageError, RuntimeError, NetworkUnavailable, APIError}
	if len(produced) != len(wantProduced) {
		t.Errorf("classifier produced %d distinct outcomes across the table, want %d (a category lost or gained coverage)", len(produced), len(wantProduced))
	}
	for _, o := range wantProduced {
		if !produced[o] {
			t.Errorf("no table row produces %v — the classifier's coverage of that category is untested", o)
		}
	}
}

// The AuthError arm must be discriminated before the rcfile arms: a
// CredentialError wraps an rcfile read/format error (via Unwrap), so without the
// ordering it would be mislabelled a base-URL UsageError instead of the
// RuntimeError a malformed credentials file warrants. Pin a wrapped
// CredentialError carrying a real rcfile cause.
func TestClassifyClientError_AuthBeforeRcfile(t *testing.T) {
	wrapped := fmt.Errorf("auth resolution failed: %w",
		&apiclient.AuthError{Kind: apiclient.CredentialError})
	if got := classifyClientError(wrapped); got != RuntimeError {
		t.Errorf("a wrapped CredentialError = %v, want RuntimeError (AuthError must win over the rcfile arms)", got)
	}
}
