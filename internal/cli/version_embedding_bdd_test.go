package cli

import (
	"fmt"
	"runtime/debug"

	"github.com/cucumber/godog"
)

// veState is the per-scenario fixture for Version Embedding (023): the resolver
// inputs a Given sets up and the value resolveVersion produces. It lives on the
// shared world like credState, so the shared struct gains one field, not many.
type veState struct {
	injected string
	info     *debug.BuildInfo
	infoOK   bool
	resolved string
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
	sc.Step(`^(?:it|the installed binary) is asked for its version$`, w.veWhenAskedForVersion)

	// --- Thens ---
	sc.Step(`^it will report "([^"]*)"(?: from the module build info Go recorded| verbatim)?$`, w.veThenReports)
	sc.Step(`^it will report a clear, non-empty development placeholder$`, w.veThenReportsPlaceholder)
	sc.Step(`^it will not report an empty string$`, w.veThenNotEmpty)
	sc.Step(`^the reported value will identify the exact commit$`, w.veThenNotEmpty)
	sc.Step(`^the recorded build info will be ignored in favor of the embedded version$`, w.veThenInjectedWon)
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
