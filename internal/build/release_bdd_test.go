package build

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/cucumber/godog"
)

// TestAutomatedReleasePipelineFeatures runs the executable acceptance for
// Automated Release Pipeline (022). Like 021's suite, its Paths name ONLY this
// spec's feature file, and it runs with the ~@wip filter so only the scenarios
// implemented so far execute.
//
// A release workflow is not unit-testable end-to-end (it runs on GitHub
// Actions), so — exactly as 021 used the config-guard as the in-process proxy
// for the build matrix — these scenarios are proven against the parsed
// declarative artifacts: the extended .goreleaser.yaml (CheckConfigGuard, for
// "build the four binaries / attach four archives + checksums") and the
// release workflow (CheckReleaseWorkflow, for trigger gating, the abort gate,
// idempotent --clobber upload, and assets-only status preservation). Actual
// artifact production is verified by the documented `goreleaser release
// --snapshot` invocation, and cross-target execution by the verify matrix
// (T003) and CI.
//
// Scenarios still held @wip: the three @validation scenarios (for /score:validate),
// the consumer-checksum scenario (needs a real download), and — until T003 —
// the self-containment-verification-abort scenario.
func TestAutomatedReleasePipelineFeatures(t *testing.T) {
	suite := godog.TestSuite{
		ScenarioInitializer: initializeReleasePipelineScenario,
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"../../features/runtime-dependent-distribution/automated-release-pipeline.feature"},
			Tags:     "~@wip",
			TestingT: t,
		},
	}
	if suite.Run() != 0 {
		t.Fatal("non-zero status: automated-release-pipeline feature scenarios failed")
	}
}

// releaseWorld is the per-scenario state: the parsed declarative artifacts plus
// the guard results computed when "the pipeline runs".
type releaseWorld struct {
	wf        Workflow
	wfLoaded  bool
	cfg       Config
	cfgLoaded bool

	workflowViolations []string
	configViolations   []string
	ran                bool
}

func initializeReleasePipelineScenario(sc *godog.ScenarioContext) {
	w := &releaseWorld{}
	sc.Before(func(ctx context.Context, _ *godog.Scenario) (context.Context, error) {
		*w = releaseWorld{}
		return ctx, nil
	})

	// --- Givens (all establish the same context: the real artifacts are loaded) ---
	sc.Step(`^a release for tag "([^"]*)" had been drafted$`, w.givenArtifactsLoaded)
	sc.Step(`^a release for tag "([^"]*)" had been published$`, w.givenArtifactsLoaded)
	sc.Step(`^the build for one target platform fails$`, w.givenNoop)
	sc.Step(`^commits had been merged to main and a draft release was updated$`, w.givenArtifactsLoadedNoArg)
	sc.Step(`^a "([^"]*)" release already had the artifact set attached$`, w.givenArtifactsLoaded)
	sc.Step(`^a maintainer had created and published a "([^"]*)" release by hand without the draft-release flow$`, w.givenArtifactsLoaded)
	sc.Step(`^a release for tag "([^"]*)" had been published and marked as a pre-release$`, w.givenArtifactsLoaded)
	sc.Step(`^the four target archives had been built for a published "([^"]*)" release$`, w.givenArtifactsLoaded)
	sc.Step(`^one target binary fails the self-containment check on its own platform$`, w.givenNoop)

	// --- Whens ---
	sc.Step(`^a maintainer publishes the release$`, w.whenPipelineRuns)
	sc.Step(`^the pipeline runs$`, w.whenPipelineRuns)
	sc.Step(`^the pipeline runs again for the "([^"]*)" release$`, w.whenPipelineRunsTag)
	sc.Step(`^no release is published$`, w.givenNoop)

	// --- Thens ---
	sc.Step(`^the pipeline will build the macOS amd64, macOS arm64, Linux amd64, and Linux arm64 binaries$`, w.thenBuildsFourBinaries)
	sc.Step(`^it will attach four "([^"]*)" archives and one checksums file to the "([^"]*)" release$`, w.thenAttachesFourArchivesAndChecksums)
	sc.Step(`^it will abort$`, w.thenAbortsViaGate)
	sc.Step(`^it will attach no archives and no checksums file$`, w.thenAbortsViaGate)
	sc.Step(`^the pipeline will not run$`, w.thenTriggerExcludesRoutine)
	sc.Step(`^no archives or checksums file will be built or attached$`, w.thenTriggerExcludesRoutine)
	sc.Step(`^it will converge on a single attached artifact set$`, w.thenUploadIsIdempotent)
	sc.Step(`^it will not create duplicate or conflicting archives$`, w.thenUploadIsIdempotent)
	sc.Step(`^it will build and attach the full artifact set exactly as for a drafted release$`, w.thenHandledIdentically)
	sc.Step(`^it will build and attach the same artifact set$`, w.thenAttachesSameSet)
	sc.Step(`^the release will remain marked as a pre-release$`, w.thenStatusPreserved)
	sc.Step(`^it will abort before attaching anything$`, w.thenVerifyGateAborts)
	sc.Step(`^the release will receive no archives and no checksums file$`, w.thenVerifyGateAborts)
}

// --- Given implementations -------------------------------------------------

func (w *releaseWorld) givenNoop() error { return nil }

func (w *releaseWorld) givenArtifactsLoaded(_ string) error { return w.load() }

func (w *releaseWorld) givenArtifactsLoadedNoArg() error { return w.load() }

// load reads the real .goreleaser.yaml and release workflow. A scenario that
// can't even load the artifacts has nothing to assert against, so a load error
// fails the step.
func (w *releaseWorld) load() error {
	cfg, _, err := LoadConfig()
	if err != nil {
		return fmt.Errorf("loading the build/release config: %w", err)
	}
	w.cfg = cfg
	w.cfgLoaded = true

	wf, _, err := LoadWorkflow()
	if err != nil {
		return fmt.Errorf("loading the release workflow: %w", err)
	}
	w.wf = wf
	w.wfLoaded = true
	return nil
}

// --- When implementations --------------------------------------------------

// whenPipelineRuns is the in-process proxy for "the pipeline runs": it evaluates
// both guards against the loaded artifacts. A correctly-configured pipeline
// reports no violations; the Then steps assert the specific property each
// scenario cares about. whenPipelineRunsTag is the one-capture-group variant
// ("runs again for the X release") — the tag is narrative, so it delegates.
func (w *releaseWorld) whenPipelineRunsTag(_ string) error { return w.whenPipelineRuns() }

func (w *releaseWorld) whenPipelineRuns() error {
	if !w.wfLoaded || !w.cfgLoaded {
		if err := w.load(); err != nil {
			return err
		}
	}
	w.configViolations = CheckConfigGuard(w.cfg)
	w.workflowViolations = CheckReleaseWorkflow(w.wf)
	w.ran = true
	return nil
}

// --- Then implementations --------------------------------------------------

// thenBuildsFourBinaries asserts the build matrix is exactly the four supported
// targets (the config-guard passing is the in-process proof a build produces one
// binary per target) and that the build job actually runs goreleaser.
func (w *releaseWorld) thenBuildsFourBinaries() error {
	if !w.cfgLoaded {
		return fmt.Errorf("the config was not loaded")
	}
	if len(w.configViolations) != 0 {
		return fmt.Errorf("the build/release config is not exactly the four targets + release sections:\n  %s",
			strings.Join(w.configViolations, "\n  "))
	}
	gotGoos := sortedCopy(w.cfg.Builds[0].Goos)
	gotGoarch := sortedCopy(w.cfg.Builds[0].Goarch)
	if strings.Join(gotGoos, ",") != "darwin,linux" || strings.Join(gotGoarch, ",") != "amd64,arm64" {
		return fmt.Errorf("expected darwin/linux × amd64/arm64, got goos=%v goarch=%v", gotGoos, gotGoarch)
	}
	if goreleaserArgs(w.wf.Jobs["build"]) == "" {
		return fmt.Errorf("the build job must run goreleaser to produce the binaries")
	}
	return nil
}

// thenAttachesFourArchivesAndChecksums asserts the config produces one tar.gz per
// target + one checksums file (config-guard) and that the publish job uploads
// exactly those, to the triggering tag. The format arg pins tar.gz from the
// scenario text.
func (w *releaseWorld) thenAttachesFourArchivesAndChecksums(format, _ string) error {
	if len(w.configViolations) != 0 {
		return fmt.Errorf("the archives/checksum config drifted:\n  %s", strings.Join(w.configViolations, "\n  "))
	}
	if format != ArchiveFormat {
		return fmt.Errorf("the scenario expects %q archives, guard pins %q", format, ArchiveFormat)
	}
	upload := uploadStepRun(w.wf.Jobs["publish"])
	if !strings.Contains(upload, "dist/*.tar.gz") || !strings.Contains(upload, "checksums.txt") {
		return fmt.Errorf("the publish job must attach the archives and the checksums file, got upload:\n%s", upload)
	}
	if !strings.Contains(upload, "github.event.release.tag_name") {
		return fmt.Errorf("the publish job must attach to the triggering release tag, got upload:\n%s", upload)
	}
	return nil
}

// thenAbortsViaGate asserts the atomicity gate: publish depends on build, so a
// build failure (this scenario's premise) skips publish — nothing is attached.
func (w *releaseWorld) thenAbortsViaGate() error {
	if err := w.requireValidWorkflow(); err != nil {
		return err
	}
	if !needsContains(w.wf.Jobs["publish"].Needs, "build") {
		return fmt.Errorf("publish must `needs: build` so a build failure aborts before any attach")
	}
	return nil
}

// thenTriggerExcludesRoutine asserts the trigger is the published-release event
// only, with no routine-activity trigger — so a merge / draft update / tag push
// does not run the pipeline.
func (w *releaseWorld) thenTriggerExcludesRoutine() error {
	if !w.wfLoaded {
		if err := w.load(); err != nil {
			return err
		}
	}
	if v := checkTrigger(w.wf.On); len(v) != 0 {
		return fmt.Errorf("the trigger must be published-release-only with no routine trigger:\n  %s",
			strings.Join(v, "\n  "))
	}
	return nil
}

// thenUploadIsIdempotent asserts the upload uses --clobber, so re-running (or
// recovering from a partial upload) converges on one asset set with no
// duplicates.
func (w *releaseWorld) thenUploadIsIdempotent() error {
	if err := w.requireValidWorkflow(); err != nil {
		return err
	}
	if !strings.Contains(uploadStepRun(w.wf.Jobs["publish"]), "--clobber") {
		return fmt.Errorf("the publish upload must use --clobber to converge on one asset set")
	}
	return nil
}

// thenHandledIdentically asserts the pipeline keys on the published event with no
// job gating on the release's source (draft-flow vs. hand-created): same trigger,
// no `if:` discriminator, and the same artifact set is produced.
func (w *releaseWorld) thenHandledIdentically() error {
	if err := w.thenTriggerExcludesRoutine(); err != nil {
		return err
	}
	for _, name := range []string{"build", "publish"} {
		if cond := strings.TrimSpace(w.wf.Jobs[name].If); cond != "" {
			return fmt.Errorf("job %q must not gate on the release source (found if: %q) — a hand-created release is handled identically", name, cond)
		}
	}
	return w.thenBuildsFourBinaries()
}

// thenAttachesSameSet asserts the same full artifact set is produced and uploaded
// (config-guard clean + the publish upload references archives and checksums).
func (w *releaseWorld) thenAttachesSameSet() error {
	if len(w.configViolations) != 0 {
		return fmt.Errorf("the artifact set drifted:\n  %s", strings.Join(w.configViolations, "\n  "))
	}
	upload := uploadStepRun(w.wf.Jobs["publish"])
	if !strings.Contains(upload, "dist/*.tar.gz") || !strings.Contains(upload, "checksums.txt") {
		return fmt.Errorf("the publish job must attach the same archives + checksums set, got upload:\n%s", upload)
	}
	return nil
}

// thenStatusPreserved asserts the publish step only adds assets — it uses
// `gh release upload` and never `gh release edit`/`create` or a --prerelease/
// --latest flag, so the pre-release/latest status the publisher chose is left
// untouched.
func (w *releaseWorld) thenStatusPreserved() error {
	if err := w.requireValidWorkflow(); err != nil {
		return err
	}
	for _, s := range w.wf.Jobs["publish"].Steps {
		if strings.Contains(s.Run, "gh release edit") || strings.Contains(s.Run, "gh release create") {
			return fmt.Errorf("the publish job must not author/edit the release (found %q) — status and notes are the publisher's", strings.TrimSpace(s.Run))
		}
		if strings.Contains(s.Run, "--prerelease") || strings.Contains(s.Run, "--latest") {
			return fmt.Errorf("the publish job must not set --prerelease/--latest — it honors the existing status")
		}
	}
	if !strings.Contains(uploadStepRun(w.wf.Jobs["publish"]), "gh release upload") {
		return fmt.Errorf("the publish job must attach assets via `gh release upload` (assets-only)")
	}
	return nil
}

// thenVerifyGateAborts asserts the cross-target gate: publish depends on verify
// (and on build), so a self-containment failure on any target leg skips publish —
// nothing is attached. The CheckVerifyGate guard passing on the loaded workflow
// is the in-process proof the gate is wired.
func (w *releaseWorld) thenVerifyGateAborts() error {
	if err := w.requireValidWorkflow(); err != nil {
		return err
	}
	if v := CheckVerifyGate(w.wf); len(v) != 0 {
		return fmt.Errorf("the verify gate is not wired (a self-containment failure would not abort):\n  %s",
			strings.Join(v, "\n  "))
	}
	if !needsContains(w.wf.Jobs["publish"].Needs, "verify") {
		return fmt.Errorf("publish must `needs: verify` so a self-containment failure aborts before any attach")
	}
	return nil
}

// requireValidWorkflow guards the structural Thens: if the workflow itself does
// not pass the guard, the scenario's property cannot be relied on, so surface the
// violations rather than asserting against a broken workflow.
func (w *releaseWorld) requireValidWorkflow() error {
	if !w.ran {
		if err := w.whenPipelineRuns(); err != nil {
			return err
		}
	}
	if len(w.workflowViolations) != 0 {
		return fmt.Errorf("the release workflow does not pass the guard:\n  %s",
			strings.Join(w.workflowViolations, "\n  "))
	}
	return nil
}
