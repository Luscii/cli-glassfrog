package apiclient

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/cucumber/godog"
)

// TestGuardedWritesFeatures runs the Guarded Writes (053) executable acceptance
// scenarios against Client.Execute + the new Request.IfMatch field, driving them
// over the package's fake base (respondingBase) — no real network, sleep, home, or
// filesystem. The base records the outbound If-Match (and Content-Type) header so a
// scenario can assert the send contract.
//
// The suite is scoped to *only* guarded-writes.feature. godog binds steps
// per-suite, so a directory-globbing Paths would pull in the sibling apiclient
// suites' scenarios and fail with undefined steps (LEARNINGS 2026-06-04). This is
// the seventh apiclient suite (007/008/009/010/017/052 → this). The two
// @validation scenarios stay @wip — held out for the validate skill, not
// implemented by the Builder.
func TestGuardedWritesFeatures(t *testing.T) {
	suite := godog.TestSuite{
		ScenarioInitializer: initializeGuardedWriteScenario,
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"../../features/clobbered-changes/guarded-writes.feature"},
			Tags:     "~@wip",
			TestingT: t,
		},
	}
	if suite.Run() != 0 {
		t.Fatal("non-zero status: feature scenarios failed")
	}
}

// guardedWriteWorld is the per-scenario state. A Given builds the write Request
// (method, optional content type, and the captured version threaded into IfMatch);
// a When runs one send through Execute over the fake base; the Thens assert on the
// outbound If-Match header the base recorded and on the unchanged refusal contract.
// Step helpers return errors, never panic (LEARNINGS).
type guardedWriteWorld struct {
	base    *respondingBase
	req     Request
	resp    *Response
	execErr error

	setIfMatch     string // the IfMatch value a Given installed, for verbatim cross-checks
	rejectedStatus int    // the non-2xx status a refused-write When configured
}

// execute runs one send through a client built over the configured base and
// captures the *Response and any error. The client is built from a complete
// context so the AuthTransport attaches the (secret) token — proving the
// precondition send is orthogonal to authentication. out is nil: 053 asserts on
// the outbound request and the refusal path, not a decoded body.
func (w *guardedWriteWorld) execute() error {
	client, err := NewClient(completeContext(secretToken), w.base)
	if err != nil {
		return fmt.Errorf("building client: %w", err)
	}
	w.resp, w.execErr = client.Execute(context.Background(), w.req, nil)
	return nil
}

func initializeGuardedWriteScenario(sc *godog.ScenarioContext) {
	w := &guardedWriteWorld{}

	sc.Before(func(ctx context.Context, _ *godog.Scenario) (context.Context, error) {
		*w = guardedWriteWorld{
			base: &respondingBase{status: 200, body: bodyOf("{}")},
			req:  Request{Method: http.MethodPatch, Path: "/tensions/ten_1"},
		}
		return ctx, nil
	})

	// --- Givens ---
	sc.Step(`^a write request carrying a captured version of "([^"]*)"$`, w.givenWriteWithVersion)
	sc.Step(`^a write request carrying a version that no longer matches the resource$`, w.givenWriteWithStaleVersion)
	sc.Step(`^a write request that carries no captured version$`, w.givenWriteWithoutVersion)
	sc.Step(`^a delete request carrying a captured version of "([^"]*)"$`, w.givenDeleteWithVersion)
	sc.Step(`^a write request whose captured version is empty$`, w.givenWriteWithEmptyVersion)
	sc.Step(`^a write request carrying a captured version that is a quoted weak validator$`, w.givenWriteWithWeakValidator)
	sc.Step(`^a write request that carries both a body media type and a captured version$`, w.givenWriteWithContentTypeAndVersion)

	// --- Whens ---
	sc.Step(`^the request is sent$`, func() error { return w.execute() })
	sc.Step(`^the request is sent and the server responds with status (\d+)$`, w.whenSentServerResponds)

	// --- Thens ---
	sc.Step(`^the outbound request will carry an If-Match header of "([^"]*)"$`, w.thenIfMatchHeaderIs)
	sc.Step(`^the outbound request will carry no If-Match header$`, w.thenNoIfMatchHeader)
	sc.Step(`^the write will proceed unconditionally, exactly as before this capability existed$`, w.thenWriteProceedsUnconditionally)
	sc.Step(`^the precondition will be attached identically regardless of the request method$`, w.thenMethodAgnostic)
	sc.Step(`^the write will not be refused for a malformed precondition$`, w.thenNotRefusedForMalformedPrecondition)
	sc.Step(`^the If-Match header will preserve the weak-validator prefix and the surrounding quotes$`, w.thenWeakValidatorPreserved)
	sc.Step(`^no part of the token will be stripped or normalized$`, w.thenTokenNotNormalized)
	sc.Step(`^the outbound request will carry both its Content-Type header and the If-Match header$`, w.thenBothHeadersPresent)
	sc.Step(`^neither header will displace the other$`, w.thenNeitherDisplacesTheOther)
	sc.Step(`^the refusal will not be interpreted or relabeled$`, w.thenRefusalNotInterpreted)
	sc.Step(`^the outcome will flow through the existing diagnostic message and exit code unchanged$`, w.thenFlowsThroughExistingPath)
}

// --- Given implementations ---

func (w *guardedWriteWorld) givenWriteWithVersion(version string) error {
	w.req = Request{Method: http.MethodPatch, Path: "/tensions/ten_1", IfMatch: version}
	w.setIfMatch = version
	return nil
}

func (w *guardedWriteWorld) givenWriteWithStaleVersion() error {
	const stale = "stale-version"
	w.req = Request{Method: http.MethodPatch, Path: "/tensions/ten_1", IfMatch: stale}
	w.setIfMatch = stale
	return nil
}

func (w *guardedWriteWorld) givenWriteWithoutVersion() error {
	// An ordinary write with no captured version — IfMatch zero-values to "".
	w.req = Request{Method: http.MethodPatch, Path: "/tensions/ten_1"}
	w.setIfMatch = ""
	return nil
}

func (w *guardedWriteWorld) givenDeleteWithVersion(version string) error {
	w.req = Request{Method: http.MethodDelete, Path: "/tensions/ten_1", IfMatch: version}
	w.setIfMatch = version
	return nil
}

func (w *guardedWriteWorld) givenWriteWithEmptyVersion() error {
	// An explicitly empty captured version is indistinguishable from unset.
	w.req = Request{Method: http.MethodPatch, Path: "/tensions/ten_1", IfMatch: ""}
	w.setIfMatch = ""
	return nil
}

func (w *guardedWriteWorld) givenWriteWithWeakValidator() error {
	const weak = `W/"abc123"`
	w.req = Request{Method: http.MethodPatch, Path: "/tensions/ten_1", IfMatch: weak}
	w.setIfMatch = weak
	return nil
}

func (w *guardedWriteWorld) givenWriteWithContentTypeAndVersion() error {
	const version = "a1b2c3"
	w.req = Request{
		Method:      http.MethodPatch,
		Path:        "/tensions/ten_1",
		Body:        strings.NewReader(`{"tension":{"body":"x"}}`),
		ContentType: "application/json",
		IfMatch:     version,
	}
	w.setIfMatch = version
	return nil
}

// --- When implementations ---

func (w *guardedWriteWorld) whenSentServerResponds(status string) error {
	code := 0
	if _, err := fmt.Sscanf(status, "%d", &code); err != nil {
		return fmt.Errorf("bad status %q: %w", status, err)
	}
	w.rejectedStatus = code
	w.base = &respondingBase{status: code, body: bodyOf(`{"error":"precondition failed"}`)}
	return w.execute()
}

// --- Then implementations ---

func (w *guardedWriteWorld) thenIfMatchHeaderIs(want string) error {
	if w.execErr != nil {
		return fmt.Errorf("unexpected error: %v", w.execErr)
	}
	if !w.base.ifMatchSet {
		return errors.New("the outbound request carried no If-Match header")
	}
	if w.base.gotIfMatch != want {
		return fmt.Errorf("If-Match = %q, want %q", w.base.gotIfMatch, want)
	}
	return nil
}

func (w *guardedWriteWorld) thenNoIfMatchHeader() error {
	if w.execErr != nil {
		return fmt.Errorf("unexpected error: %v", w.execErr)
	}
	if w.base.ifMatchSet {
		return fmt.Errorf("the outbound request carried an If-Match header %q, want none", w.base.gotIfMatch)
	}
	return nil
}

func (w *guardedWriteWorld) thenWriteProceedsUnconditionally() error {
	// The write was still sent (one attempt reached the base) and carried no
	// precondition — last-write-wins, exactly as before this capability existed.
	if w.base.calls != 1 {
		return fmt.Errorf("base reached %d times, want exactly 1 unconditional write", w.base.calls)
	}
	if w.base.ifMatchSet {
		return errors.New("an unconditional write carried an If-Match header")
	}
	return nil
}

func (w *guardedWriteWorld) thenMethodAgnostic() error {
	if w.execErr != nil {
		return fmt.Errorf("unexpected error: %v", w.execErr)
	}
	if !w.base.ifMatchSet {
		return errors.New("the guarded request carried no If-Match header")
	}
	// The send depends only on the field, not the method. Replay the SAME version on
	// a PATCH and confirm an identical If-Match send.
	patchWorld := &guardedWriteWorld{base: &respondingBase{status: 200, body: bodyOf("{}")}}
	patchWorld.req = Request{Method: http.MethodPatch, Path: "/tensions/ten_1", IfMatch: w.setIfMatch}
	if err := patchWorld.execute(); err != nil {
		return err
	}
	if patchWorld.execErr != nil {
		// execute() stores the Execute outcome in execErr (it returns only build
		// errors), so surface an unexpected replay error as the real cause rather
		// than letting it fall through to a misleading header assertion.
		return fmt.Errorf("the method-agnostic PATCH replay errored: %v", patchWorld.execErr)
	}
	if !patchWorld.base.ifMatchSet {
		return errors.New("a PATCH with the same version set no If-Match header; the precondition must be method-agnostic")
	}
	if patchWorld.base.gotIfMatch != w.base.gotIfMatch {
		return fmt.Errorf("a %s sent If-Match %q but a PATCH sent %q; the precondition must be method-agnostic", w.req.Method, w.base.gotIfMatch, patchWorld.base.gotIfMatch)
	}
	return nil
}

func (w *guardedWriteWorld) thenNotRefusedForMalformedPrecondition() error {
	// An empty version sends no precondition, so the server is never handed a
	// malformed token: the request was sent and produced no client-side error.
	if w.execErr != nil {
		return fmt.Errorf("the empty-version write errored: %v", w.execErr)
	}
	if w.base.ifMatchSet {
		return errors.New("an empty version still sent an If-Match header (a malformed precondition)")
	}
	return nil
}

func (w *guardedWriteWorld) thenWeakValidatorPreserved() error {
	if w.execErr != nil {
		return fmt.Errorf("unexpected error: %v", w.execErr)
	}
	if w.base.gotIfMatch != w.setIfMatch {
		return fmt.Errorf("If-Match = %q, want the weak validator preserved verbatim %q", w.base.gotIfMatch, w.setIfMatch)
	}
	return nil
}

func (w *guardedWriteWorld) thenTokenNotNormalized() error {
	// Verbatim: the sent value equals the IfMatch byte-for-byte — the W/ prefix and
	// the surrounding quotes are intact, nothing stripped.
	if w.base.gotIfMatch != w.setIfMatch {
		return fmt.Errorf("If-Match = %q, want it byte-for-byte unchanged from %q", w.base.gotIfMatch, w.setIfMatch)
	}
	return nil
}

func (w *guardedWriteWorld) thenBothHeadersPresent() error {
	if w.execErr != nil {
		return fmt.Errorf("unexpected error: %v", w.execErr)
	}
	if !w.base.contentTypeSet {
		return errors.New("the outbound request carried no Content-Type header")
	}
	if !w.base.ifMatchSet {
		return errors.New("the outbound request carried no If-Match header")
	}
	return nil
}

func (w *guardedWriteWorld) thenNeitherDisplacesTheOther() error {
	// If-Match and Content-Type are separate fields set by separate blocks; each
	// carries its own value, neither overwriting the other.
	if w.base.gotContentType != w.req.ContentType {
		return fmt.Errorf("Content-Type = %q, want %q (displaced by If-Match)", w.base.gotContentType, w.req.ContentType)
	}
	if w.base.gotIfMatch != w.setIfMatch {
		return fmt.Errorf("If-Match = %q, want %q (displaced by Content-Type)", w.base.gotIfMatch, w.setIfMatch)
	}
	return nil
}

func (w *guardedWriteWorld) thenRefusalNotInterpreted() error {
	// The refusal surfaces as the existing generic *ResponseError — 053 does not
	// classify, relabel, or synthesize a *Response for it.
	var respErr *ResponseError
	if !errors.As(w.execErr, &respErr) {
		return fmt.Errorf("err = %v, want the existing generic *ResponseError (the 412 not interpreted or relabeled)", w.execErr)
	}
	if respErr.StatusCode != w.rejectedStatus {
		return fmt.Errorf("status = %d, want %d carried unchanged", respErr.StatusCode, w.rejectedStatus)
	}
	if w.resp != nil {
		return errors.New("a refused write produced a *Response; the 412 must ride the error path")
	}
	return nil
}

func (w *guardedWriteWorld) thenFlowsThroughExistingPath() error {
	// The 412 is the existing generic *ResponseError carrying its status — the same
	// input the landed 004/015 diagnostic + exit-code mapping already consumes. 053
	// adds no new Outcome, exit code, or diagnostic; distinct surfacing is 054.
	var respErr *ResponseError
	if !errors.As(w.execErr, &respErr) {
		return fmt.Errorf("err = %v, want the existing *ResponseError unchanged", w.execErr)
	}
	if respErr.StatusCode != w.rejectedStatus {
		return fmt.Errorf("status = %d, want %d surfaced unchanged", respErr.StatusCode, w.rejectedStatus)
	}
	return nil
}
