package apiclient

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/Luscii/cli-glassfrog/internal/auth"
	"github.com/cucumber/godog"
)

// TestFeatures runs the Request Authentication (007) executable acceptance
// scenarios against the auth round-tripper, driving it with a fake base
// transport and a fake resolver — no real network and no real home directory.
//
// The suite is scoped to *only* request-authentication.feature: godog binds
// steps per-suite, so a directory-globbing Paths would pull in the auth and cli
// suites' scenarios and fail with undefined steps (LEARNINGS 2026-06-04). The
// three @validation scenarios stay @wip — held out for the validate skill, not
// implemented by the Builder.
func TestFeatures(t *testing.T) {
	suite := godog.TestSuite{
		ScenarioInitializer: initializeScenario,
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"../../features/unauthenticated-access/request-authentication.feature"},
			Tags:     "~@wip",
			TestingT: t,
		},
	}
	if suite.Run() != 0 {
		t.Fatal("non-zero status: feature scenarios failed")
	}
}

// recordingTransport is the fake base transport. It records the X-Auth-Token
// header (and whether it was present at all) for every request it receives, so
// steps can assert what was attached, whether the base was reached, and that
// the same identity rode every request. Reaching it at all means a request was
// "sent to the API".
type recordingTransport struct {
	calls      int
	headers    []string
	sawHeaders []bool
	resp       *http.Response
}

func (r *recordingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	r.calls++
	r.headers = append(r.headers, req.Header.Get(AuthHeaderName))
	_, ok := req.Header[AuthHeaderName]
	r.sawHeaders = append(r.sawHeaders, ok)
	return r.resp, nil
}

// authWorld is the per-scenario state. The injected resolver reads the world's
// canned outcome at call time (Givens run after Before), and counts its own
// invocations so the resolve-once scenario can assert a single call.
type authWorld struct {
	res    auth.Resolution
	resErr error
	calls  int // resolver invocations

	base      *recordingTransport
	transport *AuthTransport

	resolvedToken string // the token seeded by a Given, for verbatim/identity checks
	brokenPath    string // the broken .glassfrogrc path, for credential-error checks

	sendErr     error // error returned by the last RoundTrip
	identity    Identity
	identityErr *AuthError
	diagnostics string // an operator-facing request diagnostic, built from the reportable identity
}

func initializeScenario(sc *godog.ScenarioContext) {
	w := &authWorld{}

	sc.Before(func(ctx context.Context, _ *godog.Scenario) (context.Context, error) {
		*w = authWorld{
			base: &recordingTransport{resp: &http.Response{StatusCode: 200, Body: http.NoBody}},
		}
		resolver := func() (auth.Resolution, error) {
			w.calls++
			return w.res, w.resErr
		}
		w.transport = NewAuthTransport(w.base, resolver)
		return ctx, nil
	})

	// --- Givens: resolved credentials ---
	sc.Step(`^Credential Discovery resolved the token "([^"]*)" from a source$`, w.givenResolvedToken)
	sc.Step(`^Credential Discovery resolved the token "([^"]*)"$`, w.givenResolvedToken)
	sc.Step(`^Credential Discovery resolved a token containing unusual characters$`, w.givenResolvedUnusualToken)
	sc.Step(`^Credential Discovery was available as the credential source$`, w.givenAvailableSource)
	sc.Step(`^Credential Discovery resolved a token from the file "([^"]*)"$`, w.givenResolvedFromFile)
	sc.Step(`^request diagnostics were enabled$`, func() error { return nil })
	sc.Step(`^Credential Discovery resolved a token$`, w.givenResolvedAnyToken)

	// --- Givens: failure outcomes ---
	sc.Step(`^Credential Discovery reported that no credentials were found$`, w.givenNoCredentials)
	sc.Step(`^a "\.glassfrogrc" existed but could not be read or parsed$`, w.givenBrokenFile)
	sc.Step(`^Credential Discovery reported a credential error naming that file$`, w.givenCredentialError)

	// --- Whens ---
	sc.Step(`^the CLI sends an API request$`, w.whenSendOne)
	sc.Step(`^the CLI sends more than one API request in a single invocation$`, w.whenSendMany)
	sc.Step(`^the CLI prepares an API request$`, w.whenSendOne)
	sc.Step(`^the CLI authenticates an API request$`, w.whenAuthenticate)
	sc.Step(`^the CLI sends an API request carrying the "X-Auth-Token" header$`, w.whenSendWithDiagnostics)

	// --- Thens: attachment ---
	sc.Step(`^the request will carry an "X-Auth-Token" header set to "([^"]*)"$`, w.thenHeaderSetTo)
	sc.Step(`^the request will be sent to the API$`, w.thenRequestSent)
	sc.Step(`^the "X-Auth-Token" header value will exactly equal the resolved token$`, w.thenHeaderEqualsResolved)
	sc.Step(`^no characters will be added, removed, or re-encoded$`, w.thenHeaderEqualsResolved)
	sc.Step(`^every request will carry the same "X-Auth-Token" identity$`, w.thenSameIdentityEverywhere)
	sc.Step(`^the credential will be resolved only once$`, w.thenResolvedOnce)
	sc.Step(`^every request will reuse the resolved identity$`, w.thenSameIdentityEverywhere)

	// --- Thens: refusal ---
	sc.Step(`^the request will not be sent$`, w.thenRequestNotSent)
	sc.Step(`^the CLI will report that it cannot authenticate because no credentials were found$`, w.thenReportsNoCredentials)
	sc.Step(`^the CLI will report a credential error naming that file$`, w.thenReportsCredentialError)
	sc.Step(`^the outcome will be distinct from a no-credentials outcome$`, w.thenDistinctFromNoCredentials)

	// --- Thens: reporting / redaction ---
	sc.Step(`^the CLI will report the active identity source as that file path$`, w.thenIdentityIsFilePath)
	sc.Step(`^the token value will not appear in the output$`, w.thenTokenNotInOutput)
	sc.Step(`^the token value will be redacted from the diagnostic output$`, w.thenTokenRedactedFromDiagnostics)
}

// --- Given implementations ---

func (w *authWorld) givenResolvedToken(token string) error {
	w.res = auth.Resolution{Token: token, Source: auth.SourceEnvironment}
	w.resolvedToken = token
	return nil
}

func (w *authWorld) givenResolvedUnusualToken() error {
	// Printable but unusual: separators, encodings, and punctuation a naive
	// sanitizer might trim or re-encode. The fake base transport stores the
	// header value verbatim (no wire encoding), so a verbatim attach round-trips.
	const weird = `gf_/+=%20-token.value~with#unusual&chars`
	w.res = auth.Resolution{Token: weird, Source: auth.SourceFile, Path: "/home/dev/.glassfrogrc"}
	w.resolvedToken = weird
	return nil
}

func (w *authWorld) givenAvailableSource() error {
	w.res = auth.Resolution{Token: "gf_resolved_token", Source: auth.SourceFile, Path: "/home/dev/.glassfrogrc"}
	w.resolvedToken = "gf_resolved_token"
	return nil
}

func (w *authWorld) givenResolvedFromFile(path string) error {
	w.res = auth.Resolution{Token: secretToken, Source: auth.SourceFile, Path: path}
	w.resolvedToken = secretToken
	return nil
}

func (w *authWorld) givenResolvedAnyToken() error {
	w.res = auth.Resolution{Token: secretToken, Source: auth.SourceFile, Path: "/home/dev/.glassfrogrc"}
	w.resolvedToken = secretToken
	return nil
}

func (w *authWorld) givenNoCredentials() error {
	w.res = auth.Resolution{Source: auth.SourceNone}
	return nil
}

func (w *authWorld) givenBrokenFile() error {
	w.brokenPath = "/home/dev/.glassfrogrc"
	return nil
}

func (w *authWorld) givenCredentialError() error {
	if w.brokenPath == "" {
		w.brokenPath = "/home/dev/.glassfrogrc"
	}
	// Discovery's typed format error names only the path, never the token.
	w.resErr = &auth.FormatError{Path: w.brokenPath}
	return nil
}

// --- When implementations ---

func (w *authWorld) send() error {
	req, err := http.NewRequest(http.MethodGet, "http://example.test/api/v5/me", nil)
	if err != nil {
		return err
	}
	_, w.sendErr = w.transport.RoundTrip(req)
	return nil
}

func (w *authWorld) whenSendOne() error { return w.send() }

func (w *authWorld) whenSendMany() error {
	for i := 0; i < 3; i++ {
		if err := w.send(); err != nil {
			return err
		}
	}
	return nil
}

func (w *authWorld) whenAuthenticate() error {
	if err := w.send(); err != nil {
		return err
	}
	w.identity, w.identityErr = w.transport.ActiveIdentity()
	return nil
}

func (w *authWorld) whenSendWithDiagnostics() error {
	if err := w.send(); err != nil {
		return err
	}
	id, _ := w.transport.ActiveIdentity()
	// A request diagnostic an operator would see is built from the reportable
	// identity (Source/Path) — never the raw header — so the token is redacted
	// by construction. This mirrors how a verbose/trace line would be produced.
	w.diagnostics = fmt.Sprintf("→ GET /api/v5/me  acting as %s", id)
	return nil
}

// --- Then implementations ---

func (w *authWorld) thenHeaderSetTo(expected string) error {
	if w.sendErr != nil {
		return fmt.Errorf("unexpected error: %v", w.sendErr)
	}
	if len(w.base.headers) == 0 {
		return errors.New("no request reached the base transport")
	}
	if got := w.base.headers[len(w.base.headers)-1]; got != expected {
		return fmt.Errorf("X-Auth-Token = %q, want %q", got, expected)
	}
	return nil
}

func (w *authWorld) thenRequestSent() error {
	if w.base.calls == 0 {
		return errors.New("the request was not sent to the API")
	}
	return nil
}

func (w *authWorld) thenHeaderEqualsResolved() error {
	if w.sendErr != nil {
		return fmt.Errorf("unexpected error: %v", w.sendErr)
	}
	if len(w.base.headers) == 0 {
		return errors.New("no request reached the base transport")
	}
	if got := w.base.headers[len(w.base.headers)-1]; got != w.resolvedToken {
		return fmt.Errorf("X-Auth-Token = %q, want the resolved token %q (verbatim)", got, w.resolvedToken)
	}
	return nil
}

func (w *authWorld) thenSameIdentityEverywhere() error {
	if w.sendErr != nil {
		return fmt.Errorf("unexpected error: %v", w.sendErr)
	}
	if len(w.base.headers) < 2 {
		return fmt.Errorf("only %d requests reached the base transport, want more than one", len(w.base.headers))
	}
	for i, got := range w.base.headers {
		if got != w.resolvedToken {
			return fmt.Errorf("request %d carried %q, want the same identity %q", i, got, w.resolvedToken)
		}
	}
	return nil
}

func (w *authWorld) thenResolvedOnce() error {
	if w.calls != 1 {
		return fmt.Errorf("resolver invoked %d times, want exactly once per invocation", w.calls)
	}
	return nil
}

func (w *authWorld) thenRequestNotSent() error {
	if w.base.calls != 0 {
		return fmt.Errorf("the base transport was called %d times; no unauthenticated request may be sent", w.base.calls)
	}
	return nil
}

func (w *authWorld) thenReportsNoCredentials() error {
	var authErr *AuthError
	if !errors.As(w.sendErr, &authErr) {
		return fmt.Errorf("error %v is not an *AuthError", w.sendErr)
	}
	if authErr.Kind != NoCredentials {
		return fmt.Errorf("Kind = %v, want NoCredentials", authErr.Kind)
	}
	return nil
}

func (w *authWorld) thenReportsCredentialError() error {
	var authErr *AuthError
	if !errors.As(w.sendErr, &authErr) {
		return fmt.Errorf("error %v is not an *AuthError", w.sendErr)
	}
	if authErr.Kind != CredentialError {
		return fmt.Errorf("Kind = %v, want CredentialError", authErr.Kind)
	}
	if !strings.Contains(w.sendErr.Error(), w.brokenPath) {
		return fmt.Errorf("error %q does not name the broken file %q", w.sendErr.Error(), w.brokenPath)
	}
	return nil
}

func (w *authWorld) thenDistinctFromNoCredentials() error {
	var authErr *AuthError
	if !errors.As(w.sendErr, &authErr) {
		return fmt.Errorf("error %v is not an *AuthError", w.sendErr)
	}
	if authErr.Kind == NoCredentials {
		return errors.New("a broken credential was reported as a no-credentials outcome; they must be distinct")
	}
	return nil
}

func (w *authWorld) thenIdentityIsFilePath() error {
	if w.identityErr != nil {
		return fmt.Errorf("ActiveIdentity errored: %v", w.identityErr)
	}
	if w.identity.Source != auth.SourceFile {
		return fmt.Errorf("identity source = %v, want File", w.identity.Source)
	}
	if w.identity.Path != w.res.Path {
		return fmt.Errorf("identity path = %q, want %q", w.identity.Path, w.res.Path)
	}
	return nil
}

func (w *authWorld) thenTokenNotInOutput() error {
	if w.resolvedToken == "" {
		return errors.New("test setup error: no resolved token to check against")
	}
	outputs := map[string]string{
		"active identity": w.identity.String(),
		"diagnostics":     w.diagnostics,
	}
	if w.sendErr != nil {
		outputs["send error"] = w.sendErr.Error()
	}
	for where, text := range outputs {
		if strings.Contains(text, w.resolvedToken) {
			return fmt.Errorf("the token value leaked into the %s: %q", where, text)
		}
	}
	return nil
}

func (w *authWorld) thenTokenRedactedFromDiagnostics() error {
	if w.resolvedToken == "" {
		return errors.New("test setup error: no resolved token to check against")
	}
	// The header was actually attached (so we are testing redaction, not a
	// missing header), yet the diagnostic output omits the secret.
	if len(w.base.headers) == 0 || w.base.headers[len(w.base.headers)-1] != w.resolvedToken {
		return errors.New("the request did not carry the resolved token; nothing to redact")
	}
	if strings.Contains(w.diagnostics, w.resolvedToken) {
		return fmt.Errorf("the token value was not redacted from the diagnostic output: %q", w.diagnostics)
	}
	return nil
}
