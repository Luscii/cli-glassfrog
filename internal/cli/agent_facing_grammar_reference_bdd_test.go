package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/Luscii/cli-glassfrog/internal/build"
	"github.com/Luscii/cli-glassfrog/internal/grammar"
	"github.com/cucumber/godog"
)

// TestAgentFacingGrammarReferenceFeatures runs the executable acceptance for the
// Agent-Facing Grammar Reference (077). Its Paths name ONLY this spec's feature
// file — never the features/ directory — so the suite reports its own independent
// scenario count and un-@wip-ping these scenarios cannot disturb another suite.
//
// The feature spans two surfaces: the shipped command (this package) and the
// repository-side drift guard (internal/build). The repo convention is one suite
// per feature file — a second suite over the same file would need per-scenario
// tags the file does not carry — so this suite owns both, importing internal/build
// for the guard scenario the way the Version Embedding suite already does. The
// dependency direction is unchanged: internal/build still never imports
// internal/cli.
//
// The three @validation scenarios stay @wip (held for the validate skill) and are
// skipped by the ~@wip filter.
func TestAgentFacingGrammarReferenceFeatures(t *testing.T) {
	suite := godog.TestSuite{
		ScenarioInitializer: initializeGrammarReferenceScenario,
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"../../features/unguided-change-construction/agent-facing-grammar-reference.feature"},
			Tags:     "~@wip",
			TestingT: t,
		},
	}
	if suite.Run() != 0 {
		t.Fatal("non-zero status: agent-facing-grammar-reference feature scenarios failed")
	}
}

// grammarRefWorld is the per-scenario state. Nothing in it touches the network,
// the real environment, or a real credential file — the command under test is
// client-less, and the guard scenario perturbs its inputs in memory so the
// committed artifact on disk stays truthful for the next run.
type grammarRefWorld struct {
	// The drift-guard scenario's three sides.
	committed   []byte
	regenerated []byte
	manifestIDs []string
	findings    []string
	guardRan    bool
}

func initializeGrammarReferenceScenario(sc *godog.ScenarioContext) {
	w := &grammarRefWorld{}
	sc.Before(func(ctx context.Context, _ *godog.Scenario) (context.Context, error) {
		*w = grammarRefWorld{}
		return ctx, nil
	})

	// --- The drift guard (T003) ---
	sc.Step(`^a vendored-contract refresh changed the change-type enum$`, w.givenContractRefreshChangedTheEnum)
	sc.Step(`^the committed grammar artifact was not regenerated$`, w.givenArtifactNotRegenerated)
	sc.Step(`^the repository's merge-gating verification runs$`, w.whenMergeGatingVerificationRuns)
	sc.Step(`^it will fail naming the divergence between the contract and the rendered vocabulary$`, w.thenFailsNamingContractDivergence)
	sc.Step(`^the failure will name regeneration as the remedy$`, w.thenFailureNamesRegeneration)
}

// --- The drift guard (T003) -------------------------------------------------

// loadGuardSides reads the three sides the guard compares. A side that will not
// load is a step failure, not a guard finding: the guard checks agreement between
// sides, and an unloadable side is a different problem.
func (w *grammarRefWorld) loadGuardSides() error {
	if w.committed != nil {
		return nil
	}
	committed, err := build.ReadGrammarArtifact()
	if err != nil {
		return fmt.Errorf("reading the committed grammar artifact: %w", err)
	}
	regenerated, err := build.RenderGrammarArtifact()
	if err != nil {
		return fmt.Errorf("re-deriving the grammar artifact from its sources: %w", err)
	}
	raw, err := build.ReadGrammarFactsRecord()
	if err != nil {
		return fmt.Errorf("reading the grammar record: %w", err)
	}
	w.committed, w.regenerated = committed, regenerated
	w.manifestIDs = build.ParseGrammarFactsRecord(raw).ManifestIDs
	return nil
}

// givenContractRefreshChangedTheEnum models a refreshed vendored contract by
// changing what a FRESH DERIVATION would produce — a new enum member appears in
// the vocabulary — without touching the vendored file or the committed artifact.
// That is exactly the state a refresh PR is in before the generator runs.
func (w *grammarRefWorld) givenContractRefreshChangedTheEnum() error {
	if err := w.loadGuardSides(); err != nil {
		return err
	}
	var artifact grammar.Artifact
	if err := json.Unmarshal(w.regenerated, &artifact); err != nil {
		return fmt.Errorf("decoding the fresh derivation: %w", err)
	}
	artifact.Grammar.ChangeTypes = append(artifact.Grammar.ChangeTypes, grammar.ChangeType{
		Type:       "ZzArchiveRole", // sorts last, so the refresh is an addition rather than a reorder
		Placement:  grammar.PlacementTopLevel,
		Provenance: grammar.ProvenancePublishedContract,
	})
	refreshed, err := build.MarshalGrammarArtifact(artifact)
	if err != nil {
		return fmt.Errorf("encoding the refreshed derivation: %w", err)
	}
	w.regenerated = refreshed
	return nil
}

// givenArtifactNotRegenerated is the absence of an action: the committed bytes
// stay exactly as checked in. Asserted rather than assumed, so the scenario states
// what it depends on.
func (w *grammarRefWorld) givenArtifactNotRegenerated() error {
	if err := w.loadGuardSides(); err != nil {
		return err
	}
	if string(w.committed) == string(w.regenerated) {
		return fmt.Errorf("the committed artifact already matches the refreshed derivation — the scenario's premise (no regeneration) did not hold")
	}
	return nil
}

func (w *grammarRefWorld) whenMergeGatingVerificationRuns() error {
	if err := w.loadGuardSides(); err != nil {
		return err
	}
	w.findings = build.CheckGrammarArtifact(w.committed, w.regenerated, w.manifestIDs)
	w.guardRan = true
	return nil
}

func (w *grammarRefWorld) thenFailsNamingContractDivergence() error {
	if !w.guardRan {
		return fmt.Errorf("the guard was never run")
	}
	if len(w.findings) == 0 {
		return fmt.Errorf("the guard reported no findings — a contract refresh that outran the artifact passed the merge gate")
	}
	joined := strings.Join(w.findings, " | ")
	if !strings.Contains(joined, "CONTRACT-DERIVED") {
		return fmt.Errorf("no finding names the contract-derived vocabulary as the diverged half: %s", joined)
	}
	if !strings.Contains(joined, build.VendoredSpecPath) {
		return fmt.Errorf("no finding names the vendored contract %q: %s", build.VendoredSpecPath, joined)
	}
	return nil
}

func (w *grammarRefWorld) thenFailureNamesRegeneration() error {
	if len(w.findings) == 0 {
		return fmt.Errorf("the guard reported no findings, so no remedy was named")
	}
	for _, f := range w.findings {
		if !strings.Contains(f, build.GrammarRegenerationStep) {
			return fmt.Errorf("finding %q does not name the regeneration step %q as the remedy", f, build.GrammarRegenerationStep)
		}
	}
	return nil
}
