package build

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
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
	w := &selfContainmentWorld{tempDir: t.TempDir}
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

// selfContainmentWorld is the per-scenario state. The surface-content half
// holds the swept drafting skill (whitespace-normalized, per the operator-path
// BDD convention) and the authority-question deferral extracted from it. The
// detection half holds a fixture surface laid under a fresh temp root — the
// detection steps drive the production scan against that fixture only, never
// against the real plugin/ (the live pass belongs to the guard test).
type selfContainmentWorld struct {
	skill    string
	deferral string

	tempDir     func() string
	fixtureRoot string
	seeded      map[string]string // plugin/-relative path -> seeded content
	addedFile   string            // plugin/-relative path of the post-layout addition
	scan        *SurfaceScan
	scanErr     error
}

func (w *selfContainmentWorld) register(sc *godog.ScenarioContext) {
	sc.Before(func(ctx context.Context, _ *godog.Scenario) (context.Context, error) {
		*w = selfContainmentWorld{tempDir: w.tempDir}
		return ctx, nil
	})

	// Rule: Follow the surface's handoffs with only the plugin and the CLI
	sc.Step(`^the swept operating surface was read on a machine with only the plugin and the CLI$`, w.givenSurfaceRead)
	sc.Step(`^the proposal-drafting skill's authority-question deferral is read$`, w.whenDeferralRead)
	sc.Step(`^it will name the constraint-discovery path as the receiving component$`, w.thenNamesConstraintDiscovery)
	sc.Step(`^no spec number will be needed to follow the handoff$`, w.thenNoSpecNumberNeeded)
	sc.Step(`^a surface file contained the example id "([^"]*)", the version string "([^"]*)", and the phrase "([^"]*)"$`, w.givenKnownSafeFile)
	sc.Step(`^the operating-world references will remain intact$`, w.thenOperatingWorldIntact)

	// Rule: Development residue cannot merge into the shipped surface
	sc.Step(`^every file under the operating surface referenced only in-surface components and the glassfrog CLI$`, w.givenConformingSurface)
	sc.Step(`^the merge-gating verification run executes$`, w.whenScanRuns)
	sc.Step(`^the self-containment check will pass$`, w.thenCheckPasses)
	sc.Step(`^it will report zero violations$`, w.thenZeroViolations)
	sc.Step(`^a new file had been added under the operating surface$`, w.givenFileAdded)
	sc.Step(`^the new file will be among the files checked$`, w.thenAddedFileChecked)
	sc.Step(`^no list or configuration will have been updated to include it$`, w.thenNoRegistrationStep)
	sc.Step(`^a surface file contained the reference "([^"]*)"$`, w.givenFileContaining)
	sc.Step(`^the self-containment check runs over that surface$`, w.whenScanRuns)
	sc.Step(`^it will fail naming the file and the line$`, w.thenFailsNamingFileAndLine)
	sc.Step(`^the report will carry the matched text "([^"]*)" as a resolvable-reference violation$`, w.thenCarriesResolvableViolation)
	sc.Step(`^the report will state the remedy: replace with the in-plugin component name, or remove the reference$`, w.thenStatesResolvableRemedy)
	sc.Step(`^a surface file contained the phrase "([^"]*)"$`, w.givenFileContaining)
	sc.Step(`^it will fail on that line$`, w.thenFailsNamingFileAndLine)
	sc.Step(`^the report will name the repo-machinery phrase family as the violated rule$`, w.thenNamesMachineryFamily)
	sc.Step(`^an operating surface whose walk found zero files$`, w.givenEmptySurface)
	sc.Step(`^the self-containment check runs$`, w.whenScanRuns)
	sc.Step(`^it will fail reporting the surface as missing or empty$`, w.thenFailsMissingOrEmpty)
	sc.Step(`^it will not report success over a vacuously clean set$`, w.thenNoVacuousPass)
	sc.Step(`^a surface file referenced "([^"]*)"$`, w.givenFileContaining)
	sc.Step(`^it will fail reporting the dangling path$`, w.thenFailsDanglingPath)
	sc.Step(`^the report will state the remedy: correct the path to the existing in-surface file, or remove the reference$`, w.thenStatesDanglingRemedy)
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

// --- Rule: development residue cannot merge (fixture surfaces only) ----------

// layFixture writes plugin/-relative files under a fresh temp root and records
// the seeded contents, so intactness can be re-checked after the scan.
func (w *selfContainmentWorld) layFixture(files map[string]string) error {
	w.fixtureRoot = w.tempDir()
	w.seeded = files
	for rel, content := range files {
		abs := filepath.Join(w.fixtureRoot, OperatingSurfaceRoot, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			return fmt.Errorf("laying fixture surface: %w", err)
		}
		if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
			return fmt.Errorf("laying fixture surface: %w", err)
		}
	}
	return nil
}

func (w *selfContainmentWorld) givenConformingSurface() error {
	return w.layFixture(map[string]string{
		"skills/example/SKILL.md": "Run `glassfrog proposal list` to situate; the write-safety gate\n(plugin/hooks/gate.sh) asks before a governance write.\n",
		"hooks/gate.sh":           "#!/usr/bin/env bash\n# asks before a governance write\n",
	})
}

func (w *selfContainmentWorld) givenFileAdded() error {
	if err := w.givenConformingSurface(); err != nil {
		return err
	}
	w.addedFile = "plugin/skills/future/SKILL.md"
	abs := filepath.Join(w.fixtureRoot, filepath.FromSlash(w.addedFile))
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return fmt.Errorf("adding a future surface file: %w", err)
	}
	return os.WriteFile(abs, []byte("a future path's guidance, in surface-only terms\n"), 0o644)
}

func (w *selfContainmentWorld) givenFileContaining(text string) error {
	return w.layFixture(map[string]string{
		"skills/checked/SKILL.md": "surface prose above\nthe checked line carries " + text + " inside it\n",
	})
}

func (w *selfContainmentWorld) givenKnownSafeFile(id, version, phrase string) error {
	return w.layFixture(map[string]string{
		"skills/safe/SKILL.md": fmt.Sprintf("read %s again, ship %s, and consult %s for the enum\n", id, version, phrase),
	})
}

func (w *selfContainmentWorld) givenEmptySurface() error {
	w.fixtureRoot = w.tempDir()
	w.seeded = nil
	return os.MkdirAll(filepath.Join(w.fixtureRoot, OperatingSurfaceRoot), 0o755)
}

func (w *selfContainmentWorld) whenScanRuns() error {
	if w.fixtureRoot == "" {
		return fmt.Errorf("no fixture surface was laid — the detection steps never scan the real plugin/")
	}
	w.scan, w.scanErr = ScanOperatingSurface(w.fixtureRoot)
	return nil
}

func (w *selfContainmentWorld) violations() string {
	if w.scan == nil {
		return ""
	}
	return strings.Join(w.scan.Violations, "\n")
}

func (w *selfContainmentWorld) thenCheckPasses() error {
	if w.scanErr != nil {
		return fmt.Errorf("the check failed on a conforming surface: %v", w.scanErr)
	}
	return nil
}

func (w *selfContainmentWorld) thenZeroViolations() error {
	if w.scanErr != nil {
		return fmt.Errorf("the check errored: %v", w.scanErr)
	}
	if len(w.scan.Violations) != 0 {
		return fmt.Errorf("the check reported violations:\n%s", w.violations())
	}
	return nil
}

func (w *selfContainmentWorld) thenAddedFileChecked() error {
	if w.scanErr != nil {
		return fmt.Errorf("the check errored: %v", w.scanErr)
	}
	for _, f := range w.scan.Files {
		if f == w.addedFile {
			return nil
		}
	}
	return fmt.Errorf("the added file %q is not among the files checked: %v", w.addedFile, w.scan.Files)
}

// thenNoRegistrationStep asserts the walk-derived coverage: the added file was
// only created on disk — no other fixture file names it, so nothing resembling
// a list or configuration was updated to include it.
func (w *selfContainmentWorld) thenNoRegistrationStep() error {
	base := filepath.Base(filepath.FromSlash(w.addedFile))
	dir := "future"
	for rel, content := range w.seeded {
		if strings.Contains(content, base) || strings.Contains(content, dir) {
			return fmt.Errorf("fixture file %q references the added file — coverage would rest on registration, not the walk", rel)
		}
	}
	return nil
}

func (w *selfContainmentWorld) thenFailsNamingFileAndLine() error {
	if w.scanErr != nil {
		return fmt.Errorf("the check errored instead of reporting: %v", w.scanErr)
	}
	if !strings.Contains(w.violations(), "plugin/skills/checked/SKILL.md:2:") {
		return fmt.Errorf("no violation names the file and the line; got:\n%s", w.violations())
	}
	return nil
}

func (w *selfContainmentWorld) thenCarriesResolvableViolation(text string) error {
	joined := w.violations()
	if !strings.Contains(joined, fmt.Sprintf("forbidden reference %q", text)) {
		return fmt.Errorf("no violation carries the matched text %q; got:\n%s", text, joined)
	}
	if !containsFold(joined, "resolvable reference") {
		return fmt.Errorf("the violation is not reported as a resolvable-reference violation; got:\n%s", joined)
	}
	return nil
}

func (w *selfContainmentWorld) thenStatesResolvableRemedy() error {
	if !strings.Contains(w.violations(), "Remedy: replace with the in-plugin component name, or remove the reference.") {
		return fmt.Errorf("the report does not state the resolvable-reference remedy; got:\n%s", w.violations())
	}
	return nil
}

func (w *selfContainmentWorld) thenNamesMachineryFamily() error {
	if !containsFold(w.violations(), "repo-machinery phrase") {
		return fmt.Errorf("the report does not name the repo-machinery phrase family; got:\n%s", w.violations())
	}
	return nil
}

func (w *selfContainmentWorld) thenFailsMissingOrEmpty() error {
	if w.scanErr == nil {
		return fmt.Errorf("the check passed over a surface whose walk found zero files")
	}
	if !strings.Contains(w.scanErr.Error(), "missing or empty") {
		return fmt.Errorf("the failure does not report the surface as missing or empty: %v", w.scanErr)
	}
	return nil
}

func (w *selfContainmentWorld) thenNoVacuousPass() error {
	if w.scan != nil && w.scanErr == nil {
		return fmt.Errorf("the check reported success over a vacuously clean set")
	}
	return nil
}

func (w *selfContainmentWorld) thenFailsDanglingPath() error {
	if w.scanErr != nil {
		return fmt.Errorf("the check errored instead of reporting: %v", w.scanErr)
	}
	joined := w.violations()
	if !strings.Contains(joined, "dangling in-surface path") || !strings.Contains(joined, "plugin/hooks/does-not-exist.txt") {
		return fmt.Errorf("no violation reports the dangling path; got:\n%s", joined)
	}
	return nil
}

func (w *selfContainmentWorld) thenStatesDanglingRemedy() error {
	if !strings.Contains(w.violations(), "Remedy: correct the path to the existing in-surface file, or remove the reference.") {
		return fmt.Errorf("the report does not state the dangling-path remedy; got:\n%s", w.violations())
	}
	return nil
}

func (w *selfContainmentWorld) thenOperatingWorldIntact() error {
	for rel, want := range w.seeded {
		got, err := os.ReadFile(filepath.Join(w.fixtureRoot, OperatingSurfaceRoot, filepath.FromSlash(rel)))
		if err != nil {
			return fmt.Errorf("re-reading the fixture surface: %w", err)
		}
		if string(got) != want {
			return fmt.Errorf("the scan altered %q — operating-world references did not remain intact", rel)
		}
	}
	return nil
}
