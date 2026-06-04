package apiclient

import (
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/Luscii/cli-glassfrog/internal/auth"
)

// fakeTransport is a base http.RoundTripper that records whether it was called,
// how many times, and the X-Auth-Token header on the request it received. It
// returns a sentinel response so callers can assert the auth transport returns
// the base result unchanged.
type fakeTransport struct {
	calls     int
	gotHeader string
	sawHeader bool
	resp      *http.Response
	err       error
}

func (f *fakeTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	f.calls++
	f.gotHeader = req.Header.Get(AuthHeaderName)
	_, f.sawHeader = req.Header[AuthHeaderName]
	return f.resp, f.err
}

// countingResolver returns a resolver func that yields the given outcome and a
// pointer to its invocation count, so tests can assert resolve-once.
func countingResolver(res auth.Resolution, err error) (func() (auth.Resolution, error), *int) {
	calls := 0
	return func() (auth.Resolution, error) {
		calls++
		return res, err
	}, &calls
}

func newRequest(t *testing.T) *http.Request {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, "http://example.test/api/v5/me", nil)
	if err != nil {
		t.Fatalf("building request: %v", err)
	}
	return req
}

func TestRoundTrip_AttachesTokenAndDelegates(t *testing.T) {
	sentinel := &http.Response{StatusCode: 200, Body: http.NoBody}
	base := &fakeTransport{resp: sentinel}
	resolve, _ := countingResolver(auth.Resolution{Token: secretToken, Source: auth.SourceEnvironment}, nil)
	transport := NewAuthTransport(base, resolve)

	resp, err := transport.RoundTrip(newRequest(t))

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp != sentinel {
		t.Fatalf("response = %v, want the base transport's response unchanged", resp)
	}
	if base.calls != 1 {
		t.Fatalf("base called %d times, want 1", base.calls)
	}
	if base.gotHeader != secretToken {
		t.Fatalf("X-Auth-Token = %q, want the resolved token", base.gotHeader)
	}
}

func TestRoundTrip_AttachesTokenVerbatim(t *testing.T) {
	const weird = "gf_/+=%20\t weird-Token.value"
	base := &fakeTransport{resp: &http.Response{StatusCode: 200, Body: http.NoBody}}
	resolve, _ := countingResolver(auth.Resolution{Token: weird, Source: auth.SourceFile, Path: "/home/dev/.glassfrogrc"}, nil)
	transport := NewAuthTransport(base, resolve)

	if _, err := transport.RoundTrip(newRequest(t)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if base.gotHeader != weird {
		t.Fatalf("X-Auth-Token = %q, want the token verbatim %q", base.gotHeader, weird)
	}
}

func TestRoundTrip_DoesNotMutateCallersRequest(t *testing.T) {
	base := &fakeTransport{resp: &http.Response{StatusCode: 200, Body: http.NoBody}}
	resolve, _ := countingResolver(auth.Resolution{Token: secretToken, Source: auth.SourceEnvironment}, nil)
	transport := NewAuthTransport(base, resolve)

	req := newRequest(t)
	if _, err := transport.RoundTrip(req); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, present := req.Header[AuthHeaderName]; present {
		t.Fatal("RoundTrip mutated the caller's request header; it must clone before setting X-Auth-Token")
	}
}

func TestRoundTrip_NoCredentialsRefusesWithoutCallingBase(t *testing.T) {
	base := &fakeTransport{resp: &http.Response{StatusCode: 200, Body: http.NoBody}}
	resolve, _ := countingResolver(auth.Resolution{Source: auth.SourceNone}, nil)
	transport := NewAuthTransport(base, resolve)

	resp, err := transport.RoundTrip(newRequest(t))

	if resp != nil {
		t.Fatalf("response = %v, want nil on a refused call", resp)
	}
	var authErr *AuthError
	if !errors.As(err, &authErr) || authErr.Kind != NoCredentials {
		t.Fatalf("err = %v, want AuthError{NoCredentials}", err)
	}
	if base.calls != 0 {
		t.Fatalf("base called %d times, want 0 — no unauthenticated send", base.calls)
	}
}

func TestRoundTrip_CredentialErrorRefusesWithoutCallingBase(t *testing.T) {
	base := &fakeTransport{resp: &http.Response{StatusCode: 200, Body: http.NoBody}}
	cause := &auth.FormatError{Path: "/home/dev/.glassfrogrc"}
	resolve, _ := countingResolver(auth.Resolution{}, cause)
	transport := NewAuthTransport(base, resolve)

	resp, err := transport.RoundTrip(newRequest(t))

	if resp != nil {
		t.Fatalf("response = %v, want nil on a refused call", resp)
	}
	var authErr *AuthError
	if !errors.As(err, &authErr) || authErr.Kind != CredentialError {
		t.Fatalf("err = %v, want AuthError{CredentialError}", err)
	}
	var fe *auth.FormatError
	if !errors.As(err, &fe) || fe.Path != cause.Path {
		t.Fatalf("err does not name the file %q: %v", cause.Path, err)
	}
	if base.calls != 0 {
		t.Fatalf("base called %d times, want 0 — broken credential must not send", base.calls)
	}
}

func TestRoundTrip_ResolvesOncePerInvocation(t *testing.T) {
	base := &fakeTransport{resp: &http.Response{StatusCode: 200, Body: http.NoBody}}
	resolve, calls := countingResolver(auth.Resolution{Token: secretToken, Source: auth.SourceFile, Path: "/home/dev/.glassfrogrc"}, nil)
	transport := NewAuthTransport(base, resolve)

	for i := 0; i < 3; i++ {
		if _, err := transport.RoundTrip(newRequest(t)); err != nil {
			t.Fatalf("request %d: unexpected error: %v", i, err)
		}
	}

	if *calls != 1 {
		t.Fatalf("resolver invoked %d times, want once per invocation", *calls)
	}
	if base.calls != 3 {
		t.Fatalf("base called %d times, want 3", base.calls)
	}
	if base.gotHeader != secretToken {
		t.Fatalf("later request carried %q, want the same identity %q", base.gotHeader, secretToken)
	}
}

func TestActiveIdentity_ReportsSourceAndPathNeverToken(t *testing.T) {
	const path = "/home/dev/.glassfrogrc"
	base := &fakeTransport{resp: &http.Response{StatusCode: 200, Body: http.NoBody}}
	resolve, _ := countingResolver(auth.Resolution{Token: secretToken, Source: auth.SourceFile, Path: path}, nil)
	transport := NewAuthTransport(base, resolve)

	id, authErr := transport.ActiveIdentity()
	if authErr != nil {
		t.Fatalf("unexpected error: %v", authErr)
	}
	if id.Source != auth.SourceFile || id.Path != path {
		t.Fatalf("identity = %+v, want File source at %q", id, path)
	}
	// The identity value, formatted any common way, must not leak the token.
	for _, rendered := range []string{id.String(), id.Path, id.Source.String()} {
		if strings.Contains(rendered, secretToken) {
			t.Fatalf("active identity leaked the token: %q", rendered)
		}
	}
}

func TestActiveIdentity_SurfacesAuthFailure(t *testing.T) {
	base := &fakeTransport{resp: &http.Response{StatusCode: 200, Body: http.NoBody}}
	resolve, _ := countingResolver(auth.Resolution{Source: auth.SourceNone}, nil)
	transport := NewAuthTransport(base, resolve)

	if _, authErr := transport.ActiveIdentity(); authErr == nil || authErr.Kind != NoCredentials {
		t.Fatalf("ActiveIdentity error = %v, want NoCredentials", authErr)
	}
}
