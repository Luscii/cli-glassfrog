package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/Luscii/cli-glassfrog/internal/apiclient"
	"github.com/cucumber/godog"
)

// TestAPIErrorExtractionFeatures runs the executable acceptance for API Error
// Extraction (015). The extraction-level scenarios drive apiclient.ExtractProblem
// directly over crafted *ResponseError values; the consumer-observable scenarios
// (detail-in-message, 401/403→4, 429→5, 404→3) drive the `me` read command over
// a fake base transport returning a canned non-2xx — so every scenario runs
// offline (no real network, no real ~/.glassfrogrc). Its Paths name ONLY this
// spec's feature file — never the features/ directory — so un-@wip-ping these
// scenarios cannot disturb another internal/cli suite, and each suite reports
// its own independent scenario count (LEARNINGS). The four @validation scenarios
// stay @wip (held for the validate skill) and are skipped by the ~@wip filter.
func TestAPIErrorExtractionFeatures(t *testing.T) {
	suite := godog.TestSuite{
		ScenarioInitializer: initializeAPIErrorExtractionScenario,
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"../../features/opaque-failures/api-error-extraction.feature"},
			Tags:     "~@wip",
			TestingT: t,
		},
	}
	if suite.Run() != 0 {
		t.Fatal("non-zero status: api-error-extraction feature scenarios failed")
	}
}

// problemWorld is the per-scenario state. For extraction scenarios it holds the
// crafted *ResponseError (re), the produced *ProblemError (pe), and the expected
// crafted members. For consumer scenarios it holds the canned transport
// status/body and the captured exit code / stderr of a `me` run. usedTransport
// stays false across the pure-extraction scenarios, which pins "no backoff /
// retry while interpreting".
type problemWorld struct {
	re *apiclient.ResponseError
	pe *apiclient.ProblemError

	wantType        string
	wantTitle       string
	wantDetail      string
	extensionMarker string

	cmdStatus     int
	cmdBody       string
	exitCode      int
	stderr        string
	usedTransport bool
}

func initializeAPIErrorExtractionScenario(sc *godog.ScenarioContext) {
	w := &problemWorld{}
	sc.Before(func(ctx context.Context, _ *godog.Scenario) (context.Context, error) {
		*w = problemWorld{}
		return ctx, nil
	})

	// --- Givens (extraction-level: craft a *ResponseError) ---
	sc.Step(`^a non-2xx response had a valid RFC 9457 Problem Details body$`, w.givenValidProblemBody)
	sc.Step(`^a non-2xx response had status (\d+) and a detail of "([^"]*)"$`, w.givenStatusAndDetail)
	sc.Step(`^a non-2xx response had status (\d+) with rate-limit headers$`, w.givenStatusWithRateLimitHeaders)
	sc.Step(`^a non-2xx response had status (\d+) from a plan-gated endpoint$`, w.givenStatusPlanGated)
	sc.Step(`^a non-2xx Problem Details body had extension members beyond the standard four$`, w.givenExtensionMembers)
	sc.Step(`^a non-2xx response had HTTP status (\d+) and a body status of (\d+)$`, w.givenStatusMismatch)
	sc.Step(`^a non-2xx response had status (\d+) and no body$`, w.givenStatusEmptyBody)
	sc.Step(`^a non-2xx response had status (\d+) with an HTML body$`, w.givenStatusHTMLBody)

	// --- Givens (consumer-level: canned transport for a `me` run) ---
	sc.Step(`^a command received a non-2xx response with a detail of "([^"]*)"$`, w.givenCommandDetail)
	sc.Step(`^a command received a non-2xx response with status (\d+)$`, w.givenCommandStatus)

	// --- Whens ---
	sc.Step(`^the system interprets the non-2xx outcome$`, w.interpretOutcome)
	sc.Step(`^the command reports the failure to the operator$`, w.runCommand)
	sc.Step(`^the command maps the failure to an exit code$`, w.runCommand)

	// --- Thens (extraction) ---
	sc.Step(`^a typed API error carrying the HTTP status will be returned$`, w.carriesHTTPStatus)
	sc.Step(`^the error will carry the extracted detail, title, and type$`, w.carriesDetailTitleType)
	sc.Step(`^a typed API error carrying status (\d+) will be returned$`, w.carriesStatus)
	sc.Step(`^the error will carry the detail "([^"]*)"$`, w.carriesDetail)
	sc.Step(`^a typed API error carrying status (\d+) and the response headers will be returned$`, w.carriesStatusAndHeaders)
	sc.Step(`^the system will not sleep, back off, or retry$`, w.noBackoff)
	sc.Step(`^a typed API error carrying status (\d+) and the API's detail will be returned$`, w.carriesStatusAndAPIDetail)
	sc.Step(`^the system will not translate it into plan-availability guidance$`, w.noPlanGuidance)
	sc.Step(`^only the detail, title, and type will be surfaced as named fields$`, w.onlyStandardMembersSurfaced)
	sc.Step(`^the raw body will be preserved so the extension members remain available$`, w.rawBodyPreservesExtensions)
	sc.Step(`^the typed error's authoritative status will be (\d+)$`, w.authoritativeStatusIs)
	sc.Step(`^the body status (\d+) will be carried as metadata only$`, w.bodyStatusIsMetadataOnly)
	sc.Step(`^a fallback detail derived from the status will be supplied$`, w.fallbackDetailSupplied)
	sc.Step(`^the raw body will be preserved$`, w.rawBodyPreserved)
	sc.Step(`^a typed API error carrying status (\d+) with a fallback detail will be returned$`, w.carriesStatusWithFallback)
	sc.Step(`^the parsing will not be conditioned on the response content type$`, w.parsingNotContentTypeGated)

	// --- Thens (consumer) ---
	sc.Step(`^the failure message will contain "([^"]*)"$`, w.failureMessageContains)
	sc.Step(`^the command will exit with code (\d+)$`, w.commandExitsWithCode)
}

// --- Given implementations (extraction) ---

func (w *problemWorld) givenValidProblemBody() error {
	w.wantType, w.wantTitle, w.wantDetail = "https://errors.glassfrog.com/not-found", "Not Found", "No such role"
	w.re = &apiclient.ResponseError{
		StatusCode: 404,
		Header:     http.Header{},
		Body:       []byte(fmt.Sprintf(`{"type":%q,"title":%q,"status":404,"detail":%q}`, w.wantType, w.wantTitle, w.wantDetail)),
	}
	return nil
}

func (w *problemWorld) givenStatusAndDetail(status int, detail string) error {
	w.wantDetail = detail
	w.re = &apiclient.ResponseError{StatusCode: status, Body: []byte(fmt.Sprintf(`{"detail":%q}`, detail))}
	return nil
}

func (w *problemWorld) givenStatusWithRateLimitHeaders(status int) error {
	hdr := http.Header{}
	hdr.Set("Retry-After", "60")
	hdr.Set("X-RateLimit-Remaining", "0")
	w.re = &apiclient.ResponseError{StatusCode: status, Header: hdr, Body: []byte(`{"detail":"Too Many Requests"}`)}
	return nil
}

func (w *problemWorld) givenStatusPlanGated(status int) error {
	w.wantDetail = "You do not have access to this circle"
	w.re = &apiclient.ResponseError{StatusCode: status, Body: []byte(fmt.Sprintf(`{"detail":%q}`, w.wantDetail))}
	return nil
}

func (w *problemWorld) givenExtensionMembers() error {
	w.wantType, w.wantTitle, w.wantDetail = "about:blank", "Unprocessable", "validation failed"
	w.extensionMarker = "trace-xyz-789"
	w.re = &apiclient.ResponseError{
		StatusCode: 422,
		Body: []byte(fmt.Sprintf(
			`{"type":%q,"title":%q,"detail":%q,"status":422,"errors":[{"field":"name"}],"trace_id":%q}`,
			w.wantType, w.wantTitle, w.wantDetail, w.extensionMarker,
		)),
	}
	return nil
}

func (w *problemWorld) givenStatusMismatch(httpStatus, bodyStatus int) error {
	w.re = &apiclient.ResponseError{
		StatusCode: httpStatus,
		Body:       []byte(fmt.Sprintf(`{"status":%d,"detail":"Forbidden"}`, bodyStatus)),
	}
	return nil
}

func (w *problemWorld) givenStatusEmptyBody(status int) error {
	w.re = &apiclient.ResponseError{StatusCode: status, Body: []byte("")}
	return nil
}

func (w *problemWorld) givenStatusHTMLBody(status int) error {
	hdr := http.Header{}
	hdr.Set("Content-Type", "text/html")
	w.re = &apiclient.ResponseError{StatusCode: status, Header: hdr, Body: []byte("<html><body>502 Bad Gateway</body></html>")}
	return nil
}

// --- Given implementations (consumer) ---

func (w *problemWorld) givenCommandDetail(detail string) error {
	// Any non-permission, non-rate-limit status carries the detail to the message;
	// 404 keeps it in the residual generic APIError bucket.
	w.cmdStatus = 404
	w.cmdBody = fmt.Sprintf(`{"detail":%q}`, detail)
	w.wantDetail = detail
	return nil
}

func (w *problemWorld) givenCommandStatus(status int) error {
	w.cmdStatus = status
	w.cmdBody = `{"detail":"the API rejected the call"}`
	return nil
}

// --- When implementations ---

func (w *problemWorld) interpretOutcome() error {
	if w.re == nil {
		return errors.New("no crafted non-2xx response was set up")
	}
	w.pe = apiclient.ExtractProblem(w.re)
	if w.pe == nil {
		return errors.New("ExtractProblem returned nil — it must always produce a typed error")
	}
	return nil
}

// runCommand drives the `me` read over a fake base transport returning the canned
// non-2xx, capturing the mapped exit code and stderr. It mirrors the meWorld run
// harness (NewRootCommand + newMeCommand(fake seam) + Run + ExitCode).
func (w *problemWorld) runCommand() error {
	w.usedTransport = true
	root := NewRootCommand()
	tr := &cannedTransport{status: w.cmdStatus, body: w.cmdBody}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}
	MustRegister(root, newMeCommand(seam))

	var out, errb bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&errb)
	outcome, _ := Run(root, []string{"me"})
	w.exitCode = ExitCode(outcome)
	w.stderr = errb.String()

	if strings.Contains(out.String()+errb.String(), meSecretToken) {
		return fmt.Errorf("the token leaked into output: stdout=%q stderr=%q", out.String(), errb.String())
	}
	return nil
}

// --- Then implementations (extraction) ---

func (w *problemWorld) carriesHTTPStatus() error {
	if w.pe.StatusCode != w.re.StatusCode {
		return fmt.Errorf("StatusCode = %d, want the HTTP status %d", w.pe.StatusCode, w.re.StatusCode)
	}
	return nil
}

func (w *problemWorld) carriesDetailTitleType() error {
	if w.pe.DetailSynthesized {
		return errors.New("a valid Problem Details body must yield the API's own detail (DetailSynthesized=false)")
	}
	if w.pe.Detail != w.wantDetail {
		return fmt.Errorf("Detail = %q, want %q", w.pe.Detail, w.wantDetail)
	}
	if w.pe.Title != w.wantTitle {
		return fmt.Errorf("Title = %q, want %q", w.pe.Title, w.wantTitle)
	}
	if w.pe.Type != w.wantType {
		return fmt.Errorf("Type = %q, want %q", w.pe.Type, w.wantType)
	}
	return nil
}

func (w *problemWorld) carriesStatus(status int) error {
	if w.pe.StatusCode != status {
		return fmt.Errorf("StatusCode = %d, want %d", w.pe.StatusCode, status)
	}
	return nil
}

func (w *problemWorld) carriesDetail(detail string) error {
	if w.pe.Detail != detail {
		return fmt.Errorf("Detail = %q, want %q", w.pe.Detail, detail)
	}
	if w.pe.DetailSynthesized {
		return fmt.Errorf("the detail %q is the API's own, not a synthesized fallback", detail)
	}
	return nil
}

func (w *problemWorld) carriesStatusAndHeaders(status int) error {
	if err := w.carriesStatus(status); err != nil {
		return err
	}
	var re *apiclient.ResponseError
	if !errors.As(error(w.pe), &re) {
		return errors.New("the response headers must stay reachable via the wrapped *ResponseError")
	}
	if re.Header.Get("Retry-After") != w.re.Header.Get("Retry-After") {
		return fmt.Errorf("the rate-limit headers were not carried through: got %v", re.Header)
	}
	return nil
}

func (w *problemWorld) noBackoff() error {
	// Interpretation is a pure function over the handed-in outcome — no transport,
	// clock, or retry is involved. usedTransport is only ever set by runCommand.
	if w.usedTransport {
		return errors.New("interpreting a non-2xx must not touch a transport (no retry/backoff)")
	}
	if w.pe == nil {
		return errors.New("interpretation must still produce a typed error")
	}
	return nil
}

func (w *problemWorld) carriesStatusAndAPIDetail(status int) error {
	if err := w.carriesStatus(status); err != nil {
		return err
	}
	if w.pe.DetailSynthesized {
		return errors.New("a plan-gated 403 with a body detail must surface the API's own detail")
	}
	if w.pe.Detail != w.wantDetail {
		return fmt.Errorf("Detail = %q, want the API's own %q", w.pe.Detail, w.wantDetail)
	}
	return nil
}

func (w *problemWorld) noPlanGuidance() error {
	// 015 carries the 403 + detail generically; it must not synthesize plan-
	// availability guidance (that is the Unsignalled Plan Limits problem).
	lower := strings.ToLower(w.pe.Detail + " " + w.pe.Error())
	for _, forbidden := range []string{"not available on your plan", "upgrade your plan", "premium"} {
		if strings.Contains(lower, forbidden) {
			return fmt.Errorf("the typed error must not add plan-availability guidance (%q present): %q", forbidden, w.pe.Detail)
		}
	}
	return nil
}

func (w *problemWorld) onlyStandardMembersSurfaced() error {
	if w.pe.Type != w.wantType || w.pe.Title != w.wantTitle || w.pe.Detail != w.wantDetail {
		return fmt.Errorf("the standard members were not surfaced as fields: type=%q title=%q detail=%q", w.pe.Type, w.pe.Title, w.pe.Detail)
	}
	// The extension members are NOT promoted to any named field.
	surfaced := w.pe.Type + "\x00" + w.pe.Title + "\x00" + w.pe.Detail + "\x00" + w.pe.Error()
	if w.extensionMarker != "" && strings.Contains(surfaced, w.extensionMarker) {
		return fmt.Errorf("an extension member (%q) leaked into a named field / the message", w.extensionMarker)
	}
	return nil
}

func (w *problemWorld) rawBodyPreservesExtensions() error {
	var re *apiclient.ResponseError
	if !errors.As(error(w.pe), &re) {
		return errors.New("the raw body must stay reachable via the wrapped *ResponseError")
	}
	if w.extensionMarker != "" && !strings.Contains(string(re.Body), w.extensionMarker) {
		return fmt.Errorf("the raw body should retain the extension member %q, body=%q", w.extensionMarker, re.Body)
	}
	return nil
}

func (w *problemWorld) authoritativeStatusIs(status int) error {
	if w.pe.StatusCode != status {
		return fmt.Errorf("authoritative StatusCode = %d, want %d", w.pe.StatusCode, status)
	}
	return nil
}

func (w *problemWorld) bodyStatusIsMetadataOnly(bodyStatus int) error {
	if w.pe.BodyStatus == nil {
		return fmt.Errorf("BodyStatus should carry the disagreeing body status %d as metadata, got nil", bodyStatus)
	}
	if *w.pe.BodyStatus != bodyStatus {
		return fmt.Errorf("BodyStatus = %d, want %d", *w.pe.BodyStatus, bodyStatus)
	}
	if w.pe.StatusCode == bodyStatus {
		return fmt.Errorf("the body status %d must NOT override the authoritative StatusCode", bodyStatus)
	}
	return nil
}

func (w *problemWorld) fallbackDetailSupplied() error {
	if !w.pe.DetailSynthesized {
		return errors.New("an unreadable body must yield a synthesized fallback detail (DetailSynthesized=true)")
	}
	if strings.TrimSpace(w.pe.Detail) == "" {
		return errors.New("the fallback detail must be non-empty")
	}
	if want := http.StatusText(w.pe.StatusCode); want != "" && w.pe.Detail != want {
		return fmt.Errorf("fallback Detail = %q, want the status-derived %q", w.pe.Detail, want)
	}
	return nil
}

func (w *problemWorld) rawBodyPreserved() error {
	var re *apiclient.ResponseError
	if !errors.As(error(w.pe), &re) {
		return errors.New("the raw body must stay reachable via the wrapped *ResponseError")
	}
	if string(re.Body) != string(w.re.Body) {
		return fmt.Errorf("raw body = %q, want the original %q", re.Body, w.re.Body)
	}
	return nil
}

func (w *problemWorld) carriesStatusWithFallback(status int) error {
	if err := w.carriesStatus(status); err != nil {
		return err
	}
	return w.fallbackDetailSupplied()
}

func (w *problemWorld) parsingNotContentTypeGated() error {
	// This scenario's body was delivered with Content-Type: text/html, yet a typed
	// error was still produced — extraction never inspected the Content-Type.
	if w.re.Header.Get("Content-Type") == "" {
		return errors.New("the scenario should deliver a non-problem+json Content-Type to prove no gating")
	}
	if w.pe == nil {
		return errors.New("extraction must produce a typed error regardless of Content-Type")
	}
	// Direct proof: a VALID problem+json body delivered with text/html still
	// extracts the API's detail — so the parse is genuinely Content-Type-blind.
	probeHdr := http.Header{}
	probeHdr.Set("Content-Type", "text/html")
	probe := apiclient.ExtractProblem(&apiclient.ResponseError{
		StatusCode: 400, Header: probeHdr, Body: []byte(`{"detail":"parsed despite text/html"}`),
	})
	if probe.DetailSynthesized || probe.Detail != "parsed despite text/html" {
		return fmt.Errorf("a problem+json body under a text/html Content-Type should still parse, got detail=%q synthesized=%v", probe.Detail, probe.DetailSynthesized)
	}
	return nil
}

// --- Then implementations (consumer) ---

func (w *problemWorld) failureMessageContains(want string) error {
	if !strings.Contains(w.stderr, want) {
		return fmt.Errorf("the failure message should contain %q:\n%s", want, w.stderr)
	}
	return nil
}

func (w *problemWorld) commandExitsWithCode(code int) error {
	if w.exitCode != code {
		return fmt.Errorf("exit code = %d, want %d\nstderr: %s", w.exitCode, code, w.stderr)
	}
	return nil
}
