package cli

import (
	"bytes"
	"fmt"
	"runtime/debug"
	"strings"

	"github.com/Luscii/cli-glassfrog/internal/build"
	"github.com/cucumber/godog"
)

// veState is the per-scenario fixture for Version Embedding (023): the resolver
// inputs a Given sets up and the value resolveVersion produces, plus the
// build-config and its guard violations for the injection-seam scenario. It
// lives on the shared world like credState, so the shared struct gains one
// field, not many.
type veState struct {
	injected   string
	info       *debug.BuildInfo
	infoOK     bool
	resolved   string
	cfg        build.Config
	violations []string
}

// assembleAndRun assembles a fresh root (with the current package version var)
// and runs args, returning the trimmed combined output. Used by the end-to-end
// scenarios that assert what the assembled CLI reports for --version and the
// version command.
func assembleAndRun(args ...string) string {
	root := Assemble()
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetErr(buf)
	_, _ = Run(root, args)
	return strings.TrimSpace(buf.String())
}

// registerVersionEmbeddingSteps wires the Version Embedding (023) Given/When/Then
// steps. They exercise resolveVersion directly with crafted build info — the
// resolution precedence is a pure function, so the scenarios run offline with no
// binary build. The T002 build-config and end-to-end --version scenarios are
// wired separately (and stay @wip until T002).
func (w *world) registerVersionEmbeddingSteps(sc *godog.ScenarioContext) {
	// --- Givens ---
	sc.Step(`^a binary had the version "([^"]*)" supplied at build time$`, w.veGivenInjected)
	sc.Step(`^the same binary carried module build info recording a different version$`, w.veGivenDifferentBuildInfo)
	sc.Step(`^the CLI was installed from a tagged module version "([^"]*)" with no version supplied to the build$`, w.veGivenSourceInstall)
	sc.Step(`^the CLI was installed from an untagged commit so Go recorded a pseudo-version "([^"]*)"$`, w.veGivenSourceInstall)
	sc.Step(`^a binary was produced by a plain local build with no version supplied$`, w.veGivenNoInjection)
	sc.Step(`^Go recorded the module version as "([^"]*)"$`, w.veGivenRecordedBuildInfo)
	sc.Step(`^a binary was built with no version supplied and with no module build info available$`, w.veGivenNoInjectionNoInfo)

	// --- Whens ---
	sc.Step(`^(?:it|the installed binary|the produced binary) is asked for its version$`, w.veWhenAskedForVersion)

	// --- Thens ---
	sc.Step(`^it will report "([^"]*)"(?: from the module build info Go recorded| verbatim)?$`, w.veThenReports)
	sc.Step(`^it will report a clear, non-empty development placeholder$`, w.veThenReportsPlaceholder)
	sc.Step(`^it will not report an empty string$`, w.veThenNotEmpty)
	sc.Step(`^the reported value will identify the exact commit$`, w.veThenNotEmpty)
	sc.Step(`^the recorded build info will be ignored in favor of the embedded version$`, w.veThenInjectedWon)

	// --- T002: release/pre-release reporting (end-to-end) and the injection guard ---
	sc.Step(`^a build had the release version "([^"]*)" supplied$`, w.veGivenReleaseVersion)
	sc.Step(`^the produced binary is asked for its version via "--version"$`, w.veWhenAskedViaFlag)
	sc.Step(`^the "version" command will report the same "([^"]*)"$`, w.veThenVersionCommandSame)
	sc.Step(`^the pre-release suffix will be preserved exactly$`, w.veThenInjectedWon)
	sc.Step(`^the build configuration had its version-injection seam blanked so no version is stamped$`, w.veGivenBlankedSeam)
	sc.Step(`^the build-config assertion runs$`, w.veWhenConfigAssertionRuns)
	sc.Step(`^it will fail$`, w.veThenAssertionFailed)
	sc.Step(`^it will report that the configuration no longer injects the version variable$`, w.veThenNamesTarget)
}

// --- Given implementations ---

func (w *world) veGivenInjected(v string) error {
	w.ve.injected = v
	return nil
}

func (w *world) veGivenDifferentBuildInfo() error {
	// A version distinct from the injected one, so "injected wins" is observable.
	w.ve.info = buildInfo("v0.0.0-20200101000000-000000000000")
	w.ve.infoOK = true
	return nil
}

func (w *world) veGivenSourceInstall(recorded string) error {
	// No build-time injection; Go recorded the module version (a real tag or a
	// pseudo-version) in the binary's build info.
	w.ve.injected = ""
	w.ve.info = buildInfo(recorded)
	w.ve.infoOK = true
	return nil
}

func (w *world) veGivenNoInjection() error {
	w.ve.injected = ""
	return nil
}

func (w *world) veGivenRecordedBuildInfo(recorded string) error {
	w.ve.info = buildInfo(recorded)
	w.ve.infoOK = true
	return nil
}

func (w *world) veGivenNoInjectionNoInfo() error {
	w.ve.injected = ""
	w.ve.info = nil
	w.ve.infoOK = false
	return nil
}

// --- When implementation ---

func (w *world) veWhenAskedForVersion() error {
	// The production resolution path, with the build info this scenario staged —
	// no network or VCS lookup, exactly as the binary does at runtime.
	w.ve.resolved = resolveVersion(w.ve.injected, w.ve.info, w.ve.infoOK)
	return nil
}

// --- Then implementations ---

func (w *world) veThenReports(want string) error {
	if w.ve.resolved != want {
		return fmt.Errorf("reported version = %q, want %q", w.ve.resolved, want)
	}
	return nil
}

func (w *world) veThenReportsPlaceholder() error {
	if w.ve.resolved != placeholderVersion {
		return fmt.Errorf("reported version = %q, want the placeholder %q", w.ve.resolved, placeholderVersion)
	}
	if w.ve.resolved == "" {
		return fmt.Errorf("placeholder must be non-empty")
	}
	return nil
}

func (w *world) veThenNotEmpty() error {
	if w.ve.resolved == "" {
		return fmt.Errorf("reported version must be a non-empty string")
	}
	return nil
}

func (w *world) veThenInjectedWon() error {
	if w.ve.resolved != w.ve.injected {
		return fmt.Errorf("expected the embedded version %q to win, but reported %q", w.ve.injected, w.ve.resolved)
	}
	return nil
}

// --- T002 implementations ---

func (w *world) veGivenReleaseVersion(v string) error {
	// A supplied release version drives BOTH paths: the package var feeds the
	// end-to-end --version/version run (reset() restores it after the scenario),
	// and ve.injected feeds the pure-resolver When. The pre-release scenario uses
	// the resolver When; the stamped-version scenario uses the --version When.
	version = v
	w.ve.injected = v
	return nil
}

func (w *world) veWhenAskedViaFlag() error {
	w.ve.resolved = assembleAndRun("--version")
	return nil
}

func (w *world) veThenVersionCommandSame(want string) error {
	cmdOut := assembleAndRun("version")
	if cmdOut != want {
		return fmt.Errorf("version command reported %q, want %q", cmdOut, want)
	}
	if cmdOut != w.ve.resolved {
		return fmt.Errorf("version command (%q) and --version (%q) must be byte-identical", cmdOut, w.ve.resolved)
	}
	return nil
}

func (w *world) veGivenBlankedSeam() error {
	// A single build whose ldflags inject nothing — the 021-era empty seam, which
	// the guard must reject now that 023 owns it.
	w.ve.cfg = build.Config{Builds: []build.Build{{Ldflags: []string{""}}}}
	return nil
}

func (w *world) veWhenConfigAssertionRuns() error {
	w.ve.violations = build.CheckVersionInjection(w.ve.cfg)
	return nil
}

func (w *world) veThenAssertionFailed() error {
	if len(w.ve.violations) == 0 {
		return fmt.Errorf("expected the config assertion to fail on a blanked seam, but it passed")
	}
	return nil
}

func (w *world) veThenNamesTarget() error {
	if !strings.Contains(strings.Join(w.ve.violations, " "), build.VersionInjectionTarget) {
		return fmt.Errorf("violation should name the version variable %q, got: %v",
			build.VersionInjectionTarget, w.ve.violations)
	}
	return nil
}
