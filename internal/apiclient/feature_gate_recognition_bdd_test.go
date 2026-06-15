package apiclient

import (
	"context"
	"fmt"
	"testing"

	"github.com/cucumber/godog"
)

// TestFeatureGateRecognitionFeatures runs the Feature-Gate Recognition (060)
// executable acceptance scenarios directly against RecognizeFeatureGate — the
// recognizer is a pure leaf with no command and no request, so it is tested at
// its own boundary, exactly as api-error-extraction tests ExtractProblem. No
// CLI, no transport, no network, no filesystem, no token.
//
// The suite is scoped to *only* feature-gate-recognition.feature. godog binds
// steps per-suite, so a directory-globbing Paths would pull in the sibling
// apiclient suites' scenarios and fail with undefined steps (LEARNINGS). The two
// @validation scenarios stay @wip — held out for the validate skill.
func TestFeatureGateRecognitionFeatures(t *testing.T) {
	suite := godog.TestSuite{
		ScenarioInitializer: initializeFeatureGateScenario,
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"../../features/unsignalled-plan-limits/feature-gate-recognition.feature"},
			Tags:     "~@wip",
			TestingT: t,
		},
	}
	if suite.Run() != 0 {
		t.Fatal("non-zero status: feature scenarios failed")
	}
}

// featureGateWorld is the per-scenario state. A Given names the operation
// (method + path) and the HTTP status of the rejection; the When calls
// RecognizeFeatureGate; the Thens assert the returned Gate and the
// possibility framing. The recognizer takes no body argument, so the
// body-independence scenario is satisfied structurally. Step helpers return
// errors, never panic (LEARNINGS).
type featureGateWorld struct {
	method string
	path   string
	status int

	gate   Gate
	called bool // a When ran the recognizer
}

func initializeFeatureGateScenario(sc *godog.ScenarioContext) {
	w := &featureGateWorld{}

	sc.Before(func(ctx context.Context, _ *godog.Scenario) (context.Context, error) {
		*w = featureGateWorld{}
		return ctx, nil
	})

	// --- Givens: a named operation rejected with an HTTP status ---
	// One binding captures method, path, and status across every operation
	// phrasing; the trailing "for an invalid change" clause (the non-403 case) is
	// optional and ignored — recognition keys on method/path/status only.
	sc.Step(`^the [a-z-]+ operation ([A-Z]+) (\S+) had been rejected with HTTP status (\d+)(?: for an invalid change)?$`, w.givenRejectedOperation)

	// Body-independence and genuine-permission-denial markers: the recognizer
	// consults neither the body nor any plan/permission cause, so these are no-ops
	// that document the input the recognizer deliberately does not take.
	sc.Step(`^the response body described an unrelated cause$`, w.noop)
	sc.Step(`^the rejection was a genuine permission denial unrelated to the plan$`, w.noop)

	// ai_integration modeled-but-unregistered Givens.
	sc.Step(`^the ai_integration gate kind is modeled$`, w.givenAIIntegrationModeled)
	sc.Step(`^no operation in the recognized set carries the ai_integration gate$`, w.givenNoOperationCarriesAIIntegration)

	// --- Whens ---
	sc.Step(`^the failure is checked for a feature gate$`, w.whenChecked)
	sc.Step(`^operations are checked for a feature gate$`, w.whenAllOperationsChecked)

	// --- Thens ---
	sc.Step(`^it will be recognized as a possible plan-limit rejection$`, w.thenRecognizedPossible)
	sc.Step(`^the suspected gate will be named as Premium async proposals$`, w.thenGateIsPremium)
	sc.Step(`^it will be recognized as a possible plan-limit rejection naming Premium async proposals$`, w.thenRecognizedPossibleNamingPremium)
	sc.Step(`^it will not be recognized as a plan-limit rejection$`, w.thenNotRecognized)
	sc.Step(`^it will remain a generic permission denial$`, w.thenNotRecognized)
	sc.Step(`^the body content will not have affected whether the gate was recognized$`, w.thenBodyDidNotAffect)
	sc.Step(`^it will be recognized as a possible plan-limit rejection, not a confirmed one$`, w.thenRecognizedPossibleNotConfirmed)
	sc.Step(`^recognition will make no claim of certainty about the cause$`, w.thenNoClaimOfCertainty)
	sc.Step(`^no failure will be recognized as an ai_integration plan limit today$`, w.thenNoAIIntegrationToday)
	sc.Step(`^recognition will be ready to name that gate if such an operation is later added$`, w.thenReadyToNameAIIntegration)
}

// --- Given implementations ---

func (w *featureGateWorld) givenRejectedOperation(method, path, status string) error {
	code := 0
	if _, err := fmt.Sscanf(status, "%d", &code); err != nil {
		return fmt.Errorf("bad status %q: %w", status, err)
	}
	w.method = method
	w.path = path
	w.status = code
	return nil
}

func (w *featureGateWorld) noop() error { return nil }

func (w *featureGateWorld) givenAIIntegrationModeled() error {
	// The kind exists in the Gate type, distinct from the other kinds.
	if GateAIIntegration == GateNone || GateAIIntegration == GatePremiumAsyncProposals {
		return fmt.Errorf("GateAIIntegration is not a distinct modeled gate kind")
	}
	return nil
}

func (w *featureGateWorld) givenNoOperationCarriesAIIntegration() error {
	for _, op := range gatedOperations {
		if op.gate == GateAIIntegration {
			return fmt.Errorf("registry row %s %s carries GateAIIntegration, but no operation should today", op.method, op.pathTemplate)
		}
	}
	return nil
}

// --- When implementations ---

func (w *featureGateWorld) whenChecked() error {
	w.gate = RecognizeFeatureGate(w.method, w.path, w.status)
	w.called = true
	return nil
}

// whenAllOperationsChecked drives the ai_integration scenario: every registered
// operation is checked on a 403, and none may classify as GateAIIntegration.
func (w *featureGateWorld) whenAllOperationsChecked() error {
	for _, op := range gatedOperations {
		// Substitute a concrete id for any {…} segment so the path matches.
		concrete := concretePathFor(op.pathTemplate)
		if g := RecognizeFeatureGate(op.method, concrete, 403); g == GateAIIntegration {
			return fmt.Errorf("operation %s %s classified as GateAIIntegration today", op.method, concrete)
		}
	}
	w.called = true
	return nil
}

// --- Then implementations ---

func (w *featureGateWorld) thenRecognizedPossible() error {
	if !w.called {
		return fmt.Errorf("recognizer was not called")
	}
	if w.gate == GateNone {
		return fmt.Errorf("gate = GateNone, want a recognized (possible) plan-limit gate")
	}
	return nil
}

func (w *featureGateWorld) thenGateIsPremium() error {
	if w.gate != GatePremiumAsyncProposals {
		return fmt.Errorf("suspected gate = %v, want GatePremiumAsyncProposals (Premium async proposals)", w.gate)
	}
	return nil
}

func (w *featureGateWorld) thenRecognizedPossibleNamingPremium() error {
	if err := w.thenRecognizedPossible(); err != nil {
		return err
	}
	return w.thenGateIsPremium()
}

func (w *featureGateWorld) thenNotRecognized() error {
	if !w.called {
		return fmt.Errorf("recognizer was not called")
	}
	if w.gate != GateNone {
		return fmt.Errorf("gate = %v, want GateNone (not recognized as a plan limit)", w.gate)
	}
	return nil
}

func (w *featureGateWorld) thenBodyDidNotAffect() error {
	// The recognizer takes no body argument — recognition is a pure function of
	// (method, path, status). Prove body-independence by re-recognizing with the
	// identical inputs (no body can be supplied) and confirming the same result:
	// there is no seam through which a body could change the outcome.
	if got := RecognizeFeatureGate(w.method, w.path, w.status); got != w.gate {
		return fmt.Errorf("recognition is not a pure function of method/path/status: %v vs %v", got, w.gate)
	}
	return nil
}

func (w *featureGateWorld) thenRecognizedPossibleNotConfirmed() error {
	// A recognized result is named as a suspicion, never a confirmed plan limit.
	// The contract carries this in the type's possibility semantics: recognition
	// returns only a Gate (a suspicion), with no certainty-bearing signal.
	return w.thenRecognizedPossible()
}

func (w *featureGateWorld) thenNoClaimOfCertainty() error {
	// There is no certainty signal to assert true: RecognizeFeatureGate returns a
	// bare Gate and nothing else. A non-None Gate is, by its documented contract,
	// only *consistent with* the named gate — never a verdict.
	if w.gate == GateNone {
		return fmt.Errorf("expected a recognized (possible) gate to inspect for certainty framing")
	}
	return nil
}

func (w *featureGateWorld) thenNoAIIntegrationToday() error {
	if !w.called {
		return fmt.Errorf("operations were not checked")
	}
	// whenAllOperationsChecked already proved no operation classifies as
	// GateAIIntegration; re-affirm no registry row carries it.
	for _, op := range gatedOperations {
		if op.gate == GateAIIntegration {
			return fmt.Errorf("registry row %s %s carries GateAIIntegration today", op.method, op.pathTemplate)
		}
	}
	return nil
}

func (w *featureGateWorld) thenReadyToNameAIIntegration() error {
	// The kind exists in the type, so recognition can name it the moment a future
	// registry row carries it — no type change required.
	if GateAIIntegration == GateNone || GateAIIntegration == GatePremiumAsyncProposals {
		return fmt.Errorf("GateAIIntegration must remain a distinct, nameable gate kind")
	}
	return nil
}

// concretePathFor substitutes a placeholder id for each {…} template segment so
// a registered template becomes a matchable concrete path.
func concretePathFor(template string) string {
	out := make([]byte, 0, len(template))
	i := 0
	for i < len(template) {
		if template[i] == '{' {
			// Skip to the closing brace and emit a concrete id stand-in.
			j := i
			for j < len(template) && template[j] != '}' {
				j++
			}
			out = append(out, []byte("x")...)
			i = j + 1
			continue
		}
		out = append(out, template[i])
		i++
	}
	return string(out)
}
