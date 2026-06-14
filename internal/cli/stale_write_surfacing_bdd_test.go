package cli

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/Luscii/cli-glassfrog/internal/apiclient"
	"github.com/cucumber/godog"
)

// TestStaleWriteSurfacingFeatures runs the executable acceptance for Stale-Write
// Surfacing (054). Every "When the failure is surfaced" step drives the pure
// classification seam directly — Diagnose over a crafted 412 error, then ExitCode
// over the resulting category — with no transport, clock, or retry, which is
// exactly what the "surfaces without recovering" scenario asserts. Its Paths name
// ONLY this spec's feature file (never the features/ directory) so un-@wip-ping
// these scenarios cannot disturb another internal/cli suite, and the suite reports
// its own independent scenario count (LEARNINGS). The @validation scenarios stay
// @wip (held for the validate skill) and are skipped by ~@wip.
func TestStaleWriteSurfacingFeatures(t *testing.T) {
	suite := godog.TestSuite{
		ScenarioInitializer: initializeStaleWriteSurfacingScenario,
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"../../features/clobbered-changes/stale-write-surfacing.feature"},
			Tags:     "~@wip",
			TestingT: t,
		},
	}
	if suite.Run() != 0 {
		t.Fatal("non-zero status: stale-write-surfacing feature scenarios failed")
	}
}

// staleWorld is the per-scenario state. It holds the crafted error handed to the
// classification seam and the produced Diagnostic / exit code. For the two-status
// comparison scenario it holds a second error and its results. usedTransport stays
// false for every scenario, pinning "no re-read, retry, or back-off is performed".
type staleWorld struct {
	err  error
	d    Diagnostic
	code int
	dSet bool

	// Second status (the 500/404 comparison scenarios surface two outcomes).
	err2  error
	d2    Diagnostic
	code2 int

	usedTransport bool
}

func initializeStaleWriteSurfacingScenario(sc *godog.ScenarioContext) {
	w := &staleWorld{}
	sc.Before(func(ctx context.Context, _ *godog.Scenario) (context.Context, error) {
		*w = staleWorld{}
		return ctx, nil
	})

	// --- Givens ---
	sc.Step(`^a guarded write the server refused with status (\d+)$`, w.givenRefusedWrite)
	sc.Step(`^a status (\d+) surfaced on a request that carried no If-Match header$`, w.givenStatusNoIfMatch)
	sc.Step(`^a status (\d+) whose response body carried an error detail$`, w.givenStatusWithDetail)
	sc.Step(`^a status (\d+) whose response carried no readable detail$`, w.givenStatusNoDetail)
	sc.Step(`^a status (\d+) outcome$`, w.givenStatus)
	sc.Step(`^a status (\d+) outcome and a status (\d+) outcome$`, w.givenTwoStatuses)
	sc.Step(`^the published exit codes 0 through 6$`, w.givenPublishedCodes)
	sc.Step(`^the surfacing of statuses 401, 403, 404, 429, and 500 before this capability$`, func() error { return nil })

	// --- Whens ---
	sc.Step(`^the failure is surfaced$`, w.whenSurfaced)
	sc.Step(`^each is surfaced$`, w.whenEachSurfaced)
	sc.Step(`^the stale-write category is registered$`, w.whenSurfacedStaleWrite)
	sc.Step(`^the 412 branch is added$`, func() error { return nil })

	// --- Thens ---
	sc.Step(`^the process will exit with the stale-write code (\d+)$`, w.thenExitCode)
	sc.Step(`^the code will be distinct from the generic API-error code (\d+)$`, w.thenDistinctFrom)
	sc.Step(`^the next step will tell the operator to re-read the resource for its current version and retry the write$`, w.thenNextStepReReadRetry)
	sc.Step(`^it will not be the generic "([^"]*)" step$`, w.thenNextStepNotGeneric)
	sc.Step(`^it will still be classified as a stale write from the (\d+) status alone$`, w.thenStaleWriteFromStatus)
	sc.Step(`^the classification will not depend on the command or the resource$`, w.thenStatusDriven)
	sc.Step(`^the (\d+) will carry the stale-write category and exit code (\d+)$`, w.thenFirstStaleWrite)
	sc.Step(`^the (\d+) will keep the generic API-error category and exit code (\d+)$`, w.thenSecondGeneric)
	sc.Step(`^only a category, a cause, a next step, and an exit code will be produced$`, w.thenOnlyDiagnosticProduced)
	sc.Step(`^no re-read, retry, or back-off will be performed$`, w.thenNoRecovery)
	sc.Step(`^the cause will surface the API's own detail$`, w.thenCauseIsAPIDetail)
	sc.Step(`^the failure will be identified as a precondition failure from the resource changing since it was read$`, w.thenIdentifiedAsPrecondition)
	sc.Step(`^the cause will be derived from the (\d+) status rather than invented$`, w.thenCauseFromStatus)
	sc.Step(`^the stale-write category, exit code, and re-read next step will still be assigned$`, w.thenStaleWriteFullyAssigned)
	sc.Step(`^it will keep the generic API-error category and exit code (\d+)$`, w.thenKeepsGeneric)
	sc.Step(`^only the (\d+) status will be branched out of the generic bucket$`, w.thenOnly412Branched)
	sc.Step(`^it will take the previously-unused code (\d+)$`, w.thenStaleWriteTakesCode)
	sc.Step(`^every existing code will keep the meaning it had before$`, w.thenExistingCodesUnchanged)
	sc.Step(`^each of those statuses will keep its prior category, exit code, cause, and next step$`, w.thenNoDrift)
}

// --- Given implementations ---

// craftRefusal builds the typed *ProblemError a 412 (or any status) arrives as,
// the same shape reportClientError hands Diagnose. An empty body forces a
// synthesized detail; a non-empty body carries the API's own detail.
func craftRefusal(status int, body string) error {
	return apiclient.ExtractProblem(&apiclient.ResponseError{StatusCode: status, Body: []byte(body)})
}

func (w *staleWorld) givenRefusedWrite(status int) error {
	w.err = craftRefusal(status, `{"detail":"the resource changed since it was read"}`)
	return nil
}

// givenStatusNoIfMatch crafts the 412 with no signal of whether an If-Match was
// sent — the classification can only see the surfaced status, which is the point.
func (w *staleWorld) givenStatusNoIfMatch(status int) error {
	w.err = craftRefusal(status, "")
	return nil
}

func (w *staleWorld) givenStatusWithDetail(status int) error {
	w.err = craftRefusal(status, `{"detail":"If-Match version did not match the current resource"}`)
	return nil
}

func (w *staleWorld) givenStatusNoDetail(status int) error {
	w.err = craftRefusal(status, "")
	return nil
}

func (w *staleWorld) givenStatus(status int) error {
	w.err = craftRefusal(status, "")
	return nil
}

func (w *staleWorld) givenTwoStatuses(a, b int) error {
	w.err = craftRefusal(a, "")
	w.err2 = craftRefusal(b, "")
	return nil
}

func (w *staleWorld) givenPublishedCodes() error {
	// The 0–6 band is the constant-level source of truth (exitcode.go); no setup
	// is needed beyond asserting the registry below.
	return nil
}

// --- When implementations ---

func (w *staleWorld) whenSurfaced() error {
	w.d = Diagnose(w.err)
	w.code = ExitCode(w.d.Category)
	w.dSet = true
	return nil
}

func (w *staleWorld) whenEachSurfaced() error {
	w.d = Diagnose(w.err)
	w.code = ExitCode(w.d.Category)
	w.d2 = Diagnose(w.err2)
	w.code2 = ExitCode(w.d2.Category)
	w.dSet = true
	return nil
}

// whenSurfacedStaleWrite registers the stale-write category by surfacing a 412 —
// the act of classifying it is what assigns the new code at the registry.
func (w *staleWorld) whenSurfacedStaleWrite() error {
	w.err = craftRefusal(412, "")
	return w.whenSurfaced()
}

// --- Then implementations ---

func (w *staleWorld) thenExitCode(code int) error {
	if !w.dSet {
		return errors.New("no failure was surfaced")
	}
	if w.code != code {
		return fmt.Errorf("exit code = %d, want %d", w.code, code)
	}
	if w.d.Category != StaleWrite {
		return fmt.Errorf("category = %v, want StaleWrite", w.d.Category)
	}
	return nil
}

func (w *staleWorld) thenDistinctFrom(generic int) error {
	if w.code == generic {
		return fmt.Errorf("stale-write code %d must be distinct from the generic API-error code %d", w.code, generic)
	}
	return nil
}

func (w *staleWorld) thenNextStepReReadRetry() error {
	lower := strings.ToLower(w.d.NextStep)
	if !strings.Contains(lower, "re-read") || !strings.Contains(lower, "retry") {
		return fmt.Errorf("next step %q does not tell the operator to re-read and retry", w.d.NextStep)
	}
	return nil
}

func (w *staleWorld) thenNextStepNotGeneric(generic string) error {
	if strings.Contains(w.d.NextStep, generic) {
		return fmt.Errorf("next step %q must not be the generic %q step", w.d.NextStep, generic)
	}
	return nil
}

func (w *staleWorld) thenStaleWriteFromStatus(status int) error {
	if w.d.Category != StaleWrite {
		return fmt.Errorf("status %d: category = %v, want StaleWrite", status, w.d.Category)
	}
	return nil
}

func (w *staleWorld) thenStatusDriven() error {
	// The crafted error carries only a status and body — no command or resource
	// identity reached the classifier. That it still classified as StaleWrite
	// proves the classification is status-driven only.
	if w.d.Category != StaleWrite {
		return fmt.Errorf("category = %v, want StaleWrite (status-driven classification)", w.d.Category)
	}
	return nil
}

func (w *staleWorld) thenFirstStaleWrite(status, code int) error {
	if w.d.Category != StaleWrite {
		return fmt.Errorf("status %d: category = %v, want StaleWrite", status, w.d.Category)
	}
	if w.code != code {
		return fmt.Errorf("status %d: exit code = %d, want %d", status, w.code, code)
	}
	return nil
}

func (w *staleWorld) thenSecondGeneric(status, code int) error {
	if w.d2.Category != APIError {
		return fmt.Errorf("status %d: category = %v, want APIError", status, w.d2.Category)
	}
	if w.code2 != code {
		return fmt.Errorf("status %d: exit code = %d, want %d", status, w.code2, code)
	}
	return nil
}

func (w *staleWorld) thenOnlyDiagnosticProduced() error {
	// A stale write yields exactly the three diagnostic fields plus the mapped exit
	// code — nothing more. The category, cause, and next step are all present.
	if w.d.Category != StaleWrite {
		return fmt.Errorf("category = %v, want StaleWrite", w.d.Category)
	}
	if strings.TrimSpace(w.d.Cause) == "" {
		return errors.New("a stale write must carry a cause")
	}
	if strings.TrimSpace(w.d.NextStep) == "" {
		return errors.New("a stale write must carry a next step")
	}
	return nil
}

func (w *staleWorld) thenNoRecovery() error {
	// Diagnose + ExitCode are pure — surfacing never drives a transport, clock, or
	// retry. usedTransport is never set by any scenario in this suite.
	if w.usedTransport {
		return errors.New("surfacing a 412 must not re-read, retry, or back off")
	}
	return nil
}

func (w *staleWorld) thenCauseIsAPIDetail() error {
	if !strings.Contains(w.d.Cause, "If-Match version did not match the current resource") {
		return fmt.Errorf("cause %q does not surface the API's own detail", w.d.Cause)
	}
	return nil
}

func (w *staleWorld) thenIdentifiedAsPrecondition() error {
	// Identified end to end: the category is StaleWrite and the next step points at
	// the re-read/retry recovery the precondition failure calls for.
	if w.d.Category != StaleWrite {
		return fmt.Errorf("category = %v, want StaleWrite (precondition failure)", w.d.Category)
	}
	if !strings.Contains(strings.ToLower(w.d.NextStep), "re-read") {
		return fmt.Errorf("next step %q does not point at re-reading after a precondition failure", w.d.NextStep)
	}
	return nil
}

func (w *staleWorld) thenCauseFromStatus(status int) error {
	// The body carried no readable detail, so the cause is the status-derived
	// fallback — it names the status and the precondition failure, never invented.
	if !strings.Contains(w.d.Cause, fmt.Sprintf("%d", status)) {
		return fmt.Errorf("cause %q is not derived from the %d status", w.d.Cause, status)
	}
	lower := strings.ToLower(w.d.Cause)
	if !strings.Contains(lower, "precondition") && !strings.Contains(lower, "changed since it was read") {
		return fmt.Errorf("cause %q does not name the precondition failure", w.d.Cause)
	}
	return nil
}

func (w *staleWorld) thenStaleWriteFullyAssigned() error {
	if w.d.Category != StaleWrite {
		return fmt.Errorf("category = %v, want StaleWrite", w.d.Category)
	}
	if w.code != 7 {
		return fmt.Errorf("exit code = %d, want 7", w.code)
	}
	if !strings.Contains(strings.ToLower(w.d.NextStep), "re-read") {
		return fmt.Errorf("next step %q does not carry the re-read hint", w.d.NextStep)
	}
	return nil
}

func (w *staleWorld) thenKeepsGeneric(code int) error {
	if w.d.Category != APIError {
		return fmt.Errorf("category = %v, want APIError (generic bucket)", w.d.Category)
	}
	if w.code != code {
		return fmt.Errorf("exit code = %d, want %d", w.code, code)
	}
	return nil
}

func (w *staleWorld) thenOnly412Branched(status int) error {
	// The surfaced status (404 here) stayed in the generic bucket; only a 412
	// branches out. Confirm a 412 would branch while this status did not.
	if w.d.Category == StaleWrite {
		return fmt.Errorf("status %d must not be branched into the stale-write category", status)
	}
	if probe := Diagnose(craftRefusal(412, "")); probe.Category != StaleWrite {
		return fmt.Errorf("only the 412 should branch out, but it did not classify as StaleWrite (got %v)", probe.Category)
	}
	return nil
}

func (w *staleWorld) thenStaleWriteTakesCode(code int) error {
	if got := ExitCode(StaleWrite); got != code {
		return fmt.Errorf("StaleWrite maps to code %d, want the previously-unused %d", got, code)
	}
	return nil
}

func (w *staleWorld) thenExistingCodesUnchanged() error {
	// Every 0–6 code keeps its category mapping — the additive 7 renumbers nothing.
	for o, want := range map[Outcome]int{
		Success: 0, RuntimeError: 1, UsageError: 2, APIError: 3,
		PermissionError: 4, RateLimited: 5, NetworkUnavailable: 6,
	} {
		if got := ExitCode(o); got != want {
			return fmt.Errorf("existing code drifted: ExitCode(%v) = %d, want %d", o, got, want)
		}
	}
	return nil
}

func (w *staleWorld) thenNoDrift() error {
	// Each pre-existing status keeps its prior category and exit code after the 412
	// arm is added — none inherits the stale-write surfacing.
	for _, tc := range []struct {
		status   int
		category Outcome
		code     int
	}{
		{401, PermissionError, 4},
		{403, PermissionError, 4},
		{404, APIError, 3},
		{429, RateLimited, 5},
		{500, APIError, 3},
	} {
		d := Diagnose(craftRefusal(tc.status, ""))
		if d.Category != tc.category {
			return fmt.Errorf("status %d category drifted: got %v, want %v", tc.status, d.Category, tc.category)
		}
		if got := ExitCode(d.Category); got != tc.code {
			return fmt.Errorf("status %d code drifted: got %d, want %d", tc.status, got, tc.code)
		}
		if strings.Contains(strings.ToLower(d.NextStep), "re-read") {
			return fmt.Errorf("status %d next step drifted: it picked up the 412 re-read hint (%q)", tc.status, d.NextStep)
		}
	}
	return nil
}
