package build

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/cucumber/godog"
)

// TestHomebrewTapFeatures runs the executable acceptance for the Homebrew Tap
// (036). Like 021/022, its Paths name ONLY this spec's feature file and it runs
// with the ~@wip filter, so only the scenarios implemented in T003 execute.
//
// A live `brew install`/`brew upgrade` cannot run in a unit test (it needs a
// published release, a real tap, and a network download), so the install /
// upgrade / pre-release scenarios stay @wip — they are the manual post-first-
// release validation the plan calls out, and the @validation ones are held for
// /score:validate. T003 implements the two scenarios that ARE provable in
// process against the declarative artifacts:
//   - the config-guard catches a blanked/retargeted brews block (CheckConfigGuard);
//   - the published formula's checksums match the release's checksums file
//     (the offline render-and-inspect — skipped cleanly when goreleaser is absent).
func TestHomebrewTapFeatures(t *testing.T) {
	w := &homebrewWorld{}
	suite := godog.TestSuite{
		ScenarioInitializer: func(sc *godog.ScenarioContext) { w.register(sc) },
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"../../features/runtime-dependent-distribution/homebrew-tap.feature"},
			Tags:     "~@wip",
			TestingT: t,
		},
	}
	if suite.Run() != 0 {
		t.Fatal("non-zero status: homebrew-tap feature scenarios failed")
	}
}

// homebrewWorld is the per-scenario state: the (possibly mutated) config and the
// guard result for the config-guard scenario, and the rendered formula for the
// checksum scenario.
type homebrewWorld struct {
	cfg        Config
	violations []string
	formula    renderedFormula
}

func (w *homebrewWorld) register(sc *godog.ScenarioContext) {
	sc.Before(func(ctx context.Context, _ *godog.Scenario) (context.Context, error) {
		*w = homebrewWorld{}
		return ctx, nil
	})

	// Config-guard scenario.
	sc.Step(`^the "\.goreleaser\.yaml" no longer has a "brews" entry targeting the "([^"]*)" tap$`, w.givenBrewsBlanked)
	sc.Step(`^the config-guard test runs in PR validation$`, w.whenGuardRuns)
	sc.Step(`^it will fail before any release is cut$`, w.thenGuardFails)
	sc.Step(`^it will name the missing or drifted brews configuration$`, w.thenNamesBrews)

	// Checksum-match scenario (offline render-and-inspect).
	sc.Step(`^a stable release whose formula has been published to the tap$`, w.givenFormulaRendered)
	sc.Step(`^each archive recorded in the formula is checked against the release's "([^"]*)"$`, w.whenChecksumsChecked)
	sc.Step(`^every recorded sha256 will equal the matching checksums-file entry$`, w.thenChecksumsMatch)
}

// --- Config-guard scenario -------------------------------------------------

// givenBrewsBlanked loads the real config and removes its brews entry, modelling
// "the .goreleaser.yaml no longer has a brews entry targeting the tap".
func (w *homebrewWorld) givenBrewsBlanked(_ string) error {
	cfg, _, err := LoadConfig()
	if err != nil {
		return fmt.Errorf("loading the real config: %w", err)
	}
	cfg.Brews = nil
	w.cfg = cfg
	return nil
}

func (w *homebrewWorld) whenGuardRuns() error {
	w.violations = CheckConfigGuard(w.cfg)
	return nil
}

func (w *homebrewWorld) thenGuardFails() error {
	if len(w.violations) == 0 {
		return fmt.Errorf("the config-guard must fail on a blanked brews block, but it reported no violations")
	}
	return nil
}

func (w *homebrewWorld) thenNamesBrews() error {
	if !strings.Contains(strings.ToLower(strings.Join(w.violations, "\n")), "brews") {
		return fmt.Errorf("a guard violation must name the brews configuration, got: %v", w.violations)
	}
	return nil
}

// --- Checksum-match scenario -----------------------------------------------

// givenFormulaRendered renders the formula offline. When goreleaser is absent
// (e.g. PR validation runs `go test ./...` without it) the scenario is skipped
// rather than failed — the render is exercised in CI's release path and as a
// documented local invocation.
func (w *homebrewWorld) givenFormulaRendered() error {
	rf, err, available := getRenderedFormula()
	if !available {
		return godog.ErrSkip
	}
	if err != nil {
		return fmt.Errorf("rendering the formula offline: %w", err)
	}
	w.formula = rf
	return nil
}

// whenChecksumsChecked pins the checksums filename from the scenario text and
// confirms the formula recorded the four archive checksums to compare.
func (w *homebrewWorld) whenChecksumsChecked(_ string) error {
	if len(w.formula.shas) != 4 {
		return fmt.Errorf("expected the formula to record four archive checksums, got %d", len(w.formula.shas))
	}
	return nil
}

func (w *homebrewWorld) thenChecksumsMatch() error {
	for filename, formulaSHA := range w.formula.shas {
		wantSHA, ok := w.formula.checksums[filename]
		if !ok {
			return fmt.Errorf("formula references %s but it is absent from the checksums file", filename)
		}
		if formulaSHA != wantSHA {
			return fmt.Errorf("sha256 mismatch for %s: formula records %s, checksums file has %s", filename, formulaSHA, wantSHA)
		}
	}
	return nil
}
