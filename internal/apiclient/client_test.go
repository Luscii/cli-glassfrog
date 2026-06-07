package apiclient

import (
	"errors"
	"net/http"
	"testing"

	"github.com/Luscii/cli-glassfrog/internal/auth"
)

// fakeBase is a fake base http.RoundTripper used by the client tests. It records
// the X-Auth-Token header it observed (and whether the header was present) and
// counts its calls, so a test can assert the transport attached the credential
// and that the base was reached. The canned response (or error) it returns is
// set per test.
type fakeBase struct {
	calls    int
	gotToken string
	sawToken bool
	resp     *http.Response
	err      error
}

func (f *fakeBase) RoundTrip(req *http.Request) (*http.Response, error) {
	f.calls++
	f.gotToken = req.Header.Get(AuthHeaderName)
	_, f.sawToken = req.Header[AuthHeaderName]
	if f.err != nil {
		return nil, f.err
	}
	if f.resp != nil {
		return f.resp, nil
	}
	return &http.Response{StatusCode: 200, Header: make(http.Header), Body: http.NoBody}, nil
}

func TestNewClientRefusesOnBaseURLError(t *testing.T) {
	wantErr := &BaseURLError{Source: "--base-url"}
	ctx := ConnectionContext{
		BaseURLErr: wantErr,
		// A present credential proves NewClient never inspects the token on the
		// base-URL fail-fast branch — it must refuse regardless.
		Cred: auth.Resolution{Token: secretToken, Source: auth.SourceFile},
	}

	client, err := NewClient(ctx, &fakeBase{})

	if client != nil {
		t.Fatalf("client = %v, want nil on the base-URL fail-fast branch", client)
	}
	if !errors.Is(err, wantErr) {
		t.Fatalf("err = %v, want the carried BaseURLErr %v verbatim", err, wantErr)
	}
}

func TestNewClientBuildsAuthenticatedTransport(t *testing.T) {
	base := &fakeBase{}
	client, err := NewClient(completeContext(secretToken), base)
	if err != nil {
		t.Fatalf("NewClient errored on a usable base URL: %v", err)
	}
	if client == nil {
		t.Fatal("client is nil on the usable branch")
	}

	// Drive a request through the built client's transport: the AuthTransport
	// (not the client) must attach X-Auth-Token, observed on the fake base.
	req, err := http.NewRequest(http.MethodGet, "https://glassfrog.com/api/v5/me", nil)
	if err != nil {
		t.Fatalf("building request: %v", err)
	}
	if _, err := client.httpClient.Transport.RoundTrip(req); err != nil {
		t.Fatalf("RoundTrip errored: %v", err)
	}

	if base.calls != 1 {
		t.Fatalf("base reached %d times, want exactly 1", base.calls)
	}
	if base.gotToken != secretToken {
		t.Fatalf("base saw X-Auth-Token = %q, want %q attached by the transport", base.gotToken, secretToken)
	}
	// The client must not attach the header itself — the original request the
	// caller built must remain header-free.
	if _, ok := req.Header[AuthHeaderName]; ok {
		t.Fatalf("the original request carried %s; the client must not attach it itself", AuthHeaderName)
	}
}

func TestNewClientNilBasePanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("NewClient did not panic on a nil base")
		}
	}()
	_, _ = NewClient(completeContext(secretToken), nil)
}

func TestNewClientNilBasePanicsEvenWithBaseURLError(t *testing.T) {
	// The nil-base precondition is checked before the base-URL fail-fast, so a
	// wiring bug panics deterministically and is never masked by a BaseURLErr.
	defer func() {
		if recover() == nil {
			t.Fatal("NewClient did not panic on a nil base when BaseURLErr was also set")
		}
	}()
	ctx := ConnectionContext{BaseURLErr: &BaseURLError{Source: "--base-url"}}
	_, _ = NewClient(ctx, nil)
}

func TestNewClientSetsRequestTimeout(t *testing.T) {
	client, err := NewClient(completeContext(secretToken), &fakeBase{})
	if err != nil {
		t.Fatalf("NewClient errored: %v", err)
	}
	if client.httpClient.Timeout != requestTimeout {
		t.Fatalf("client timeout = %v, want %v", client.httpClient.Timeout, requestTimeout)
	}
}

func TestNewClientFromOSDelegates(t *testing.T) {
	// On a usable base URL it builds a real-transport-backed client.
	client, err := NewClientFromOS(completeContext(secretToken))
	if err != nil {
		t.Fatalf("NewClientFromOS errored on a usable base URL: %v", err)
	}
	if client == nil || client.httpClient == nil {
		t.Fatal("NewClientFromOS returned no usable client")
	}
	if client.httpClient.Timeout != requestTimeout {
		t.Fatalf("client timeout = %v, want %v", client.httpClient.Timeout, requestTimeout)
	}

	// On a base-URL error it delegates the fail-fast: the carried error verbatim,
	// no client.
	wantErr := &BaseURLError{Source: EnvVarBaseURL}
	c, err := NewClientFromOS(ConnectionContext{BaseURLErr: wantErr})
	if c != nil {
		t.Fatalf("client = %v, want nil on the base-URL fail-fast branch", c)
	}
	if !errors.Is(err, wantErr) {
		t.Fatalf("err = %v, want the carried BaseURLErr %v verbatim", err, wantErr)
	}
}
