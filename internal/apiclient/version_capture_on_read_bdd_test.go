package apiclient

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/cucumber/godog"
)

// TestVersionCaptureOnReadFeatures runs the Version Capture on Read (052)
// executable acceptance scenarios against Client.Execute + the new
// Response.Version() accessor, driving them over the package's fake base
// (respondingBase) — no real network, sleep, home, or filesystem.
//
// The suite is scoped to *only* version-capture-on-read.feature. godog binds
// steps per-suite, so a directory-globbing Paths would pull in the sibling
// apiclient suites' scenarios and fail with undefined steps (LEARNINGS
// 2026-06-04). This is the sixth apiclient suite (007/008/009/010/017 → this).
// The two @validation scenarios stay @wip — held out for the validate skill, not
// implemented by the Builder.
func TestVersionCaptureOnReadFeatures(t *testing.T) {
	suite := godog.TestSuite{
		ScenarioInitializer: initializeVersionCaptureScenario,
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"../../features/clobbered-changes/version-capture-on-read.feature"},
			Tags:     "~@wip",
			TestingT: t,
		},
	}
	if suite.Run() != 0 {
		t.Fatal("non-zero status: feature scenarios failed")
	}
}

// versionCaptureWorld is the per-scenario state. A Given installs the fake base's
// canned single-resource (or list) response and the ETag it carried; a When runs
// one read through Execute and captures the result; the Thens assert on the
// version the Response.Version() accessor reports and on the unchanged read
// contract. Step helpers return errors, never panic (LEARNINGS).
type versionCaptureWorld struct {
	base    *respondingBase
	resp    *Response
	out     map[string]any
	execErr error

	setETag        string // the ETag value a Given installed, for cross-checks
	rejectedStatus int    // the non-2xx status a rejected-read Given configured
}

// execute runs one GET read through a client built over the configured base and
// captures the *Response, the decoded body, and any error. The client is built
// from a complete context so the AuthTransport attaches the (secret) token —
// proving capture is orthogonal to authentication.
func (w *versionCaptureWorld) execute() error {
	client, err := NewClient(completeContext(secretToken), w.base)
	if err != nil {
		return fmt.Errorf("building client: %w", err)
	}
	w.out = map[string]any{}
	w.resp, w.execErr = client.Execute(context.Background(), Request{Method: http.MethodGet, Path: "/me"}, &w.out)
	return nil
}

func initializeVersionCaptureScenario(sc *godog.ScenarioContext) {
	w := &versionCaptureWorld{}

	sc.Before(func(ctx context.Context, _ *godog.Scenario) (context.Context, error) {
		*w = versionCaptureWorld{
			base: &respondingBase{status: 200, body: bodyOf(`{"id":"x"}`)},
		}
		return ctx, nil
	})

	// --- Givens ---
	sc.Step(`^a single-resource read of a tension whose response carried an ETag of "([^"]*)"$`, w.givenSingleResourceWithETag)
	sc.Step(`^a single-resource read of a role whose response carried an ETag of "([^"]*)"$`, w.givenSingleResourceWithETag)
	sc.Step(`^a single-resource read whose response carried an ETag of "([^"]*)"$`, w.givenSingleResourceWithETag)
	sc.Step(`^a single-resource read whose response carried no ETag header$`, w.givenSingleResourceNoETag)
	sc.Step(`^a single-resource read that the server rejected with status (\d+)$`, w.givenRejectedRead)
	sc.Step(`^a single-resource read whose ETag was a quoted weak validator$`, w.givenWeakValidatorETag)
	sc.Step(`^a read that returned a list of tensions with a collection-level ETag$`, w.givenListReadWithCollectionETag)

	// --- Whens (all run one read; phrased per scenario intent) ---
	sc.Step(`^the read completes$`, func() error { return w.execute() })
	sc.Step(`^the read is rendered in any output format$`, func() error { return w.execute() })
	sc.Step(`^the failure is handled$`, func() error { return w.execute() })

	// --- Thens ---
	sc.Step(`^the captured version will be "([^"]*)"$`, w.thenCapturedVersionIs)
	sc.Step(`^no version will be captured$`, w.thenNoVersionCaptured)
	sc.Step(`^the read will still succeed and render normally$`, w.thenReadSucceedsAndRenders)
	sc.Step(`^the existing diagnostic message and exit code will be unchanged$`, w.thenFailureContractUnchanged)
	sc.Step(`^the capture will behave identically to a tension read$`, w.thenBehavesIdenticallyToTension)
	sc.Step(`^the rendered output will be byte-for-byte what it was before version capture existed$`, w.thenRenderedOutputUnchanged)
	sc.Step(`^the captured version will be present only on the in-process result$`, w.thenVersionOnlyInProcess)
	sc.Step(`^no per-resource version will be captured for any item in the list$`, w.thenNoPerItemVersion)
	sc.Step(`^the captured version will preserve the weak-validator prefix and the surrounding quotes$`, w.thenWeakValidatorPreserved)
	sc.Step(`^no part of the token will be stripped or normalized$`, w.thenTokenNotNormalized)
}

// --- Given implementations ---

func (w *versionCaptureWorld) givenSingleResourceWithETag(etag string) error {
	header := make(http.Header)
	header.Set("ETag", etag)
	w.setETag = etag
	w.base = &respondingBase{status: 200, header: header, body: bodyOf(`{"id":"x"}`)}
	return nil
}

func (w *versionCaptureWorld) givenSingleResourceNoETag() error {
	w.setETag = ""
	w.base = &respondingBase{status: 200, body: bodyOf(`{"id":"x"}`)}
	return nil
}

func (w *versionCaptureWorld) givenRejectedRead(status string) error {
	code := 0
	if _, err := fmt.Sscanf(status, "%d", &code); err != nil {
		return fmt.Errorf("bad status %q: %w", status, err)
	}
	w.rejectedStatus = code
	// Even if the server set an ETag, a non-2xx yields no *Response to capture on.
	header := make(http.Header)
	header.Set("ETag", "would-not-be-captured")
	w.base = &respondingBase{status: code, header: header, body: bodyOf(`{"error":"not found"}`)}
	return nil
}

func (w *versionCaptureWorld) givenWeakValidatorETag() error {
	const weak = `W/"abc123"`
	header := make(http.Header)
	header.Set("ETag", weak)
	w.setETag = weak
	w.base = &respondingBase{status: 200, header: header, body: bodyOf(`{"id":"x"}`)}
	return nil
}

func (w *versionCaptureWorld) givenListReadWithCollectionETag() error {
	header := make(http.Header)
	header.Set("ETag", "collection-etag")
	// An enveloped list body: the items themselves carry no version field — there
	// is no per-resource version seam, only a single response-level ETag.
	w.base = &respondingBase{
		status: 200,
		header: header,
		body:   bodyOf(`{"tensions":[{"id":"ten_1"},{"id":"ten_2"}]}`),
	}
	return nil
}

// --- Then implementations ---

func (w *versionCaptureWorld) thenCapturedVersionIs(want string) error {
	if w.execErr != nil {
		return fmt.Errorf("unexpected error: %v", w.execErr)
	}
	if w.resp == nil {
		return errors.New("no *Response to capture a version from")
	}
	if got := w.resp.Version(); got != want {
		return fmt.Errorf("captured version = %q, want %q", got, want)
	}
	return nil
}

func (w *versionCaptureWorld) thenNoVersionCaptured() error {
	// Two routes to "nothing captured": a successful read with no ETag (resp set,
	// Version() == ""), or a failed read (no *Response exists at all).
	if w.resp == nil {
		return nil
	}
	if got := w.resp.Version(); got != "" {
		return fmt.Errorf("captured version = %q, want \"\" (no version captured)", got)
	}
	return nil
}

func (w *versionCaptureWorld) thenReadSucceedsAndRenders() error {
	if w.execErr != nil {
		return fmt.Errorf("read failed: %v", w.execErr)
	}
	if w.resp == nil || w.resp.StatusCode != 200 {
		return fmt.Errorf("resp = %v, want a successful 200 read", w.resp)
	}
	// The body decoded into the caller's target exactly as before — capture did
	// not interfere with rendering.
	if w.out["id"] != "x" {
		return fmt.Errorf("decoded body = %v, want the read body to render normally", w.out)
	}
	return nil
}

func (w *versionCaptureWorld) thenFailureContractUnchanged() error {
	// The existing failure path is untouched: a non-2xx still surfaces the generic
	// *ResponseError carrying its status (the input to the existing diagnostic +
	// exit-code mapping), and there is no *Response to capture a version on.
	var respErr *ResponseError
	if !errors.As(w.execErr, &respErr) {
		return fmt.Errorf("err = %v, want the existing *ResponseError unchanged", w.execErr)
	}
	if respErr.StatusCode != w.rejectedStatus {
		return fmt.Errorf("status = %d, want %d surfaced unchanged", respErr.StatusCode, w.rejectedStatus)
	}
	if w.resp != nil {
		return errors.New("a failed read produced a *Response; capture must not synthesize one")
	}
	return nil
}

func (w *versionCaptureWorld) thenBehavesIdenticallyToTension() error {
	if w.execErr != nil {
		return fmt.Errorf("unexpected error: %v", w.execErr)
	}
	if w.resp == nil {
		return errors.New("no *Response to compare a captured version from")
	}
	// The accessor takes no resource-type input — it reads only the header. Prove
	// resource-agnosticism by replaying the SAME ETag as a tension read and
	// confirming an identical capture.
	captured := w.resp.Version()
	tensionWorld := &versionCaptureWorld{}
	if err := tensionWorld.givenSingleResourceWithETag(w.setETag); err != nil {
		return err
	}
	if err := tensionWorld.execute(); err != nil {
		return err
	}
	if tensionWorld.resp == nil {
		return errors.New("the comparison tension read produced no *Response")
	}
	if got := tensionWorld.resp.Version(); got != captured {
		return fmt.Errorf("a tension read captured %q but the role read captured %q; capture must be resource-agnostic", got, captured)
	}
	return nil
}

func (w *versionCaptureWorld) thenRenderedOutputUnchanged() error {
	if w.execErr != nil {
		return fmt.Errorf("unexpected error: %v", w.execErr)
	}
	// The decoded body the render path consumes carries no version field — the
	// captured version never entered the rendered model, so output is unchanged.
	if _, leaked := w.out["version"]; leaked {
		return errors.New("a version field leaked into the decoded body (rendered output)")
	}
	if _, leaked := w.out["etag"]; leaked {
		return errors.New("an etag field leaked into the decoded body (rendered output)")
	}
	if w.out["id"] != "x" {
		return fmt.Errorf("decoded body = %v, want the original read body unchanged", w.out)
	}
	return nil
}

func (w *versionCaptureWorld) thenVersionOnlyInProcess() error {
	// The version is reachable only via the in-process *Response accessor, never
	// the decoded body that gets rendered.
	if w.resp == nil || w.resp.Version() == "" {
		return errors.New("the version was not captured on the in-process result")
	}
	if _, leaked := w.out["version"]; leaked {
		return errors.New("the version is present in the rendered body, not only in-process")
	}
	return nil
}

func (w *versionCaptureWorld) thenNoPerItemVersion() error {
	if w.execErr != nil {
		return fmt.Errorf("unexpected error: %v", w.execErr)
	}
	items, ok := w.out["tensions"].([]any)
	if !ok {
		return fmt.Errorf("decoded list = %v, want a tensions array", w.out)
	}
	// There is no per-resource version seam: no list item carries a captured
	// version. The accessor is response-level only (a single collection ETag),
	// never attached per item.
	for i, raw := range items {
		item, ok := raw.(map[string]any)
		if !ok {
			return fmt.Errorf("item %d = %v, want an object", i, raw)
		}
		if _, has := item["version"]; has {
			return fmt.Errorf("item %d carries a per-resource version; a list read captures none", i)
		}
		if _, has := item["etag"]; has {
			return fmt.Errorf("item %d carries a per-resource etag; a list read captures none", i)
		}
	}
	return nil
}

func (w *versionCaptureWorld) thenWeakValidatorPreserved() error {
	if w.resp == nil {
		return errors.New("no *Response to capture a version from")
	}
	if got := w.resp.Version(); got != w.setETag {
		return fmt.Errorf("captured version = %q, want the weak validator preserved verbatim %q", got, w.setETag)
	}
	return nil
}

func (w *versionCaptureWorld) thenTokenNotNormalized() error {
	if w.resp == nil {
		return errors.New("no *Response to capture a version from")
	}
	// Verbatim: the captured value equals the raw ETag byte-for-byte — the W/
	// prefix and the surrounding quotes are intact, nothing stripped.
	got := w.resp.Version()
	if got != w.setETag {
		return fmt.Errorf("captured version = %q, want it byte-for-byte unchanged from %q", got, w.setETag)
	}
	return nil
}
