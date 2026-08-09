package build

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/cucumber/godog"
)

// TestSurfaceSelfContainmentFeatures runs the executable acceptance for
// operating-surface self-containment (076,
// operating-surface-self-containment.feature). Like the sibling build-side
// suites its Paths name ONLY this spec's feature file and it runs with the
// ~@wip filter, so only the scenarios implemented so far execute.
//
// The suite carries two step families. Surface-content steps read the real
// swept surface read-only, asserting its handoffs name in-plugin components.
// Detection steps (the guard scenarios) drive the self-containment production
// functions against t.TempDir() fixture surfaces only — never against the real
// plugin/, so a seeded violation is never introduced into the checkout. The
// invariant across both: the suite never writes to plugin/ and never runs the
// walker against it — the live pass belongs to the guard test.
func TestSurfaceSelfContainmentFeatures(t *testing.T) {
	w := &selfContainmentWorld{}
	suite := godog.TestSuite{
		ScenarioInitializer: func(sc *godog.ScenarioContext) { w.register(sc) },
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"../../features/unequipped-agent-operators/operating-surface-self-containment.feature"},
			Tags:     "~@wip",
			TestingT: t,
		},
	}
	if suite.Run() != 0 {
		t.Fatal("non-zero status: operating-surface-self-containment feature scenarios failed")
	}
}

// specNumberPattern matches a spec-number id (a repo cross-reference like
// "(067)" or "063's"). It is the surface-content steps' own needle for "no
// spec number is needed": no development-spec catalog exists where the
// operator stands, so a bare 0NN token is a repo id, never a surface name.
var specNumberPattern = regexp.MustCompile(`\b0\d{2}\b`)

// selfContainmentWorld is the per-scenario state: the swept drafting skill
// (whitespace-normalized, per the operator-path BDD convention) and the
// authority-question deferral passage extracted from it.
type selfContainmentWorld struct {
	skill    string
	deferral string
}

func (w *selfContainmentWorld) register(sc *godog.ScenarioContext) {
	sc.Before(func(ctx context.Context, _ *godog.Scenario) (context.Context, error) {
		*w = selfContainmentWorld{}
		return ctx, nil
	})

	// Rule: Follow the surface's handoffs with only the plugin and the CLI
	sc.Step(`^the swept operating surface was read on a machine with only the plugin and the CLI$`, w.givenSurfaceRead)
	sc.Step(`^the proposal-drafting skill's authority-question deferral is read$`, w.whenDeferralRead)
	sc.Step(`^it will name the constraint-discovery path as the receiving component$`, w.thenNamesConstraintDiscovery)
	sc.Step(`^no spec number will be needed to follow the handoff$`, w.thenNoSpecNumberNeeded)
}

// --- Rule: handoffs name in-plugin components --------------------------------

// givenSurfaceRead loads the drafting skill exactly as a shipped plugin carries
// it — read-only, nothing repo-side needed to interpret it.
func (w *selfContainmentWorld) givenSurfaceRead() error {
	skill, err := ReadProposalDraftingSkill()
	if err != nil {
		return fmt.Errorf("proposal-drafting skill did not load: %w", err)
	}
	w.skill = normalizeWS(skill)
	return nil
}

// whenDeferralRead extracts the authority-question deferral — the sentence
// that hands "am I allowed to do X?" onward — from the skill's boundary prose.
func (w *selfContainmentWorld) whenDeferralRead() error {
	if w.skill == "" {
		if err := w.givenSurfaceRead(); err != nil {
			return err
		}
	}
	start := strings.Index(w.skill, "am I allowed")
	if start < 0 {
		return fmt.Errorf("the drafting skill carries no authority-question deferral to read")
	}
	rest := w.skill[start:]
	end := strings.Index(rest, ".")
	if end < 0 {
		return fmt.Errorf("the authority-question deferral never ends — no sentence boundary found")
	}
	w.deferral = rest[:end+1]
	return nil
}

func (w *selfContainmentWorld) thenNamesConstraintDiscovery() error {
	if !containsFold(w.deferral, "Constraint Discovery Path") {
		return fmt.Errorf("the deferral does not name the constraint-discovery path as the receiving component: %q", w.deferral)
	}
	return nil
}

func (w *selfContainmentWorld) thenNoSpecNumberNeeded() error {
	if m := specNumberPattern.FindString(w.deferral); m != "" {
		return fmt.Errorf("the handoff still needs the spec number %q to follow: %q", m, w.deferral)
	}
	return nil
}
