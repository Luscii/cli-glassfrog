package apiclient

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Luscii/cli-glassfrog/internal/rcfile"
	"github.com/cucumber/godog"
)

// TestBaseURLFeatures runs the Base URL Resolution (008) executable acceptance
// scenarios against the pure resolver (ResolveBaseURL), driving it over temp
// directory trees with injected roots, a controlled GLASSFROG_BASE_URL (the env
// seam), and a supplied flag value — no real network and no real home directory.
//
// This is a SEPARATE suite from TestFeatures (007). godog binds steps per-suite,
// so each suite's Paths names only its own feature file (LEARNINGS 2026-06-04):
// TestFeatures owns request-authentication.feature, this suite owns
// base-url-resolution.feature. The four @validation scenarios stay @wip — held
// out for the validate skill, not implemented by the Builder.
func TestBaseURLFeatures(t *testing.T) {
	suite := godog.TestSuite{
		ScenarioInitializer: initializeBaseURLScenario,
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"../../features/undefined-connection-settings/base-url-resolution.feature"},
			Tags:     "~@wip",
			TestingT: t,
		},
	}
	if suite.Run() != 0 {
		t.Fatal("non-zero status: base URL feature scenarios failed")
	}
}

// baseURLWorld is the per-scenario state. It builds a temp filesystem and drives
// the pure resolver by passing scenario-chosen temp directories as the injected
// roots and reading GLASSFROG_BASE_URL through the stubbed env seam — never the
// developer's real home directory, working directory, or environment.
type baseURLWorld struct {
	base    string // root temp dir for this scenario, removed in After
	currDir string // the "current directory"
	homeDir string

	flagValue   string
	flagPresent bool   // cobra Changed() for --base-url: the flag rung is presence-based (040 ADR-2)
	envValue    string // "" models both unset and empty (the resolver treats them alike)

	currPath string // seeded current-dir .glassfrogrc, when present
	homePath string // seeded home-dir .glassfrogrc, when present

	result BaseURL
	err    error
}

// mkdir and seed return errors rather than panicking: a panic inside a step
// would skip godog's After hook, leaving the env seam stubbed and the temp tree
// undeleted, which leaks into later scenarios (LEARNINGS).
func (w *baseURLWorld) mkdir(dir string) (string, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("mkdir %s: %w", dir, err)
	}
	return dir, nil
}

func (w *baseURLWorld) seed(dir, content string) (string, error) {
	if _, err := w.mkdir(dir); err != nil {
		return "", err
	}
	path := filepath.Join(dir, rcfile.FileName)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		return "", fmt.Errorf("seed %s: %w", path, err)
	}
	return path, nil
}

func initializeBaseURLScenario(sc *godog.ScenarioContext) {
	w := &baseURLWorld{}

	origGetenv := getenv

	sc.Before(func(ctx context.Context, _ *godog.Scenario) (context.Context, error) {
		base, err := os.MkdirTemp("", "baseurl-bdd-")
		if err != nil {
			return ctx, err
		}
		*w = baseURLWorld{base: base}
		// currDir sits in a sibling subtree from home, so home is the final
		// fallback (not an ancestor) unless a scenario reconfigures it — matching
		// the precedence-chain semantics the scenarios assert.
		if w.currDir, err = w.mkdir(filepath.Join(base, "work", "nested")); err != nil {
			return ctx, err
		}
		if w.homeDir, err = w.mkdir(filepath.Join(base, "home")); err != nil {
			return ctx, err
		}
		// Stub the env seam to read this scenario's value live (a later Given sets
		// it after Before runs). Empty means unset/empty — both fall through.
		getenv = func(key string) string {
			if key == EnvVarBaseURL {
				return w.envValue
			}
			return ""
		}
		return ctx, nil
	})

	sc.After(func(ctx context.Context, _ *godog.Scenario, _ error) (context.Context, error) {
		getenv = origGetenv
		if w.base != "" {
			os.RemoveAll(w.base)
		}
		return ctx, nil
	})

	// --- Givens: the flag ---
	sc.Step(`^the base-URL flag was set to "([^"]*)"$`, w.givenFlag)
	sc.Step(`^the base-URL flag was set to "([^"]*)" with no scheme$`, w.givenFlag)
	sc.Step(`^the base-URL flag was not set$`, w.givenNoFlag)

	// --- Givens: the environment ---
	sc.Step(`^GLASSFROG_BASE_URL was set to "([^"]*)"$`, w.givenEnv)
	sc.Step(`^GLASSFROG_BASE_URL was not set$`, w.givenNoEnv)
	sc.Step(`^GLASSFROG_BASE_URL was set to an empty value$`, w.givenNoEnv)

	// --- Givens: files ---
	sc.Step(`^a "\.glassfrogrc" in the current directory held the base URL "([^"]*)"$`, w.givenCurrentBaseURL)
	sc.Step(`^a "\.glassfrogrc" in the current directory held the base URL "([^"]*)" with no scheme$`, w.givenCurrentBaseURL)
	sc.Step(`^a "\.glassfrogrc" in the home directory held the base URL "([^"]*)"$`, w.givenHomeBaseURL)
	sc.Step(`^a "\.glassfrogrc" in the current directory existed with no base URL entry$`, w.givenCurrentNoBaseURL)
	sc.Step(`^the nearest "\.glassfrogrc" existed but could not be read$`, w.givenNearestUnreadable)

	// --- Givens: absence (no-ops — Before seeds nothing) ---
	sc.Step(`^no "\.glassfrogrc" held a base URL in the current directory, any ancestor, or the home directory$`, func() error { return nil })

	// --- When ---
	sc.Step(`^the CLI resolves the base URL$`, w.whenResolve)

	// --- Thens ---
	sc.Step(`^it will use the base URL from the flag$`, w.thenValueFromFlag)
	sc.Step(`^it will report the source as the flag$`, w.thenSourceFlag)
	sc.Step(`^it will consult no other source$`, w.thenConsultedNoOtherSource)
	sc.Step(`^it will report a format error naming the flag$`, w.thenFormatErrorNamingFlag)
	sc.Step(`^it will not fall through to another source$`, w.thenNoFallThrough)
	sc.Step(`^it will use the built-in default base URL$`, w.thenValueIsDefault)
	sc.Step(`^it will report the source as the built-in default$`, w.thenSourceDefault)
	sc.Step(`^it will use the base URL from the current directory's file$`, w.thenValueFromCurrentFile)
	sc.Step(`^it will use the base URL from GLASSFROG_BASE_URL$`, w.thenValueFromEnv)
	sc.Step(`^it will not read any config file$`, w.thenNoFileRead)
	sc.Step(`^it will skip the file with no base URL$`, w.thenSkippedBaseURLlessFile)
	sc.Step(`^it will use the base URL from the home file$`, w.thenValueFromHomeFile)
	sc.Step(`^it will not treat the empty variable as a base URL$`, w.thenNotEnvSource)
	sc.Step(`^it will report a read error naming that file$`, w.thenReadErrorNamingFile)
	sc.Step(`^it will not fall through to the built-in default$`, w.thenNoFallThrough)
	sc.Step(`^it will report a format error naming that file$`, w.thenFormatErrorNamingFile)
	sc.Step(`^it will report a format error naming GLASSFROG_BASE_URL$`, w.thenFormatErrorNamingEnv)
}

// --- Given implementations ---

func (w *baseURLWorld) givenFlag(v string) error { w.flagValue = v; w.flagPresent = true; return nil }
func (w *baseURLWorld) givenNoFlag() error       { w.flagValue = ""; w.flagPresent = false; return nil }
func (w *baseURLWorld) givenEnv(v string) error  { w.envValue = v; return nil }
func (w *baseURLWorld) givenNoEnv() error        { w.envValue = ""; return nil }

func (w *baseURLWorld) givenCurrentBaseURL(v string) error {
	path, err := w.seed(w.currDir, "base_url="+v+"\n")
	if err != nil {
		return err
	}
	w.currPath = path
	return nil
}

func (w *baseURLWorld) givenHomeBaseURL(v string) error {
	path, err := w.seed(w.homeDir, "base_url="+v+"\n")
	if err != nil {
		return err
	}
	w.homePath = path
	return nil
}

func (w *baseURLWorld) givenCurrentNoBaseURL() error {
	// A file that exists and parses but holds no base_url (a token only): it must
	// be skipped, not shadow a lower source.
	path, err := w.seed(w.currDir, "token=gf_only\n# no base url here\n")
	if err != nil {
		return err
	}
	w.currPath = path
	return nil
}

func (w *baseURLWorld) givenNearestUnreadable() error {
	// A directory at the .glassfrogrc path makes os.ReadFile fail deterministically
	// across platforms (a path-only error, not os.ErrNotExist) — the fail-loud read
	// branch without OS-dependent 0o000 semantics (LEARNINGS).
	path := filepath.Join(w.currDir, rcfile.FileName)
	if _, err := w.mkdir(path); err != nil {
		return err
	}
	w.currPath = path
	return nil
}

// --- When implementation ---

func (w *baseURLWorld) whenResolve() error {
	w.result, w.err = ResolveBaseURL(w.flagValue, w.flagPresent, w.currDir, w.homeDir)
	return nil
}

// --- Then implementations ---

func (w *baseURLWorld) thenValueFromFlag() error {
	if w.err != nil {
		return fmt.Errorf("unexpected error: %v", w.err)
	}
	if w.result.Source != SourceFlag || w.result.Value != w.flagValue {
		return fmt.Errorf("got %+v, want the flag value %q", w.result, w.flagValue)
	}
	return nil
}

func (w *baseURLWorld) thenSourceFlag() error {
	if w.result.Source != SourceFlag {
		return fmt.Errorf("Source = %v, want Flag", w.result.Source)
	}
	return nil
}

func (w *baseURLWorld) thenConsultedNoOtherSource() error {
	if w.err != nil {
		return fmt.Errorf("a lower source surfaced an error, so another source was consulted: %v", w.err)
	}
	// A flag hit reads no file and consults no env value, so it carries no file
	// path — the env/file rungs never contributed.
	if w.result.Source != SourceFlag || w.result.Path != "" {
		return fmt.Errorf("got %+v, want a Flag source with no file path", w.result)
	}
	return nil
}

func (w *baseURLWorld) thenFormatErrorNamingFlag() error {
	var be *BaseURLError
	if !errors.As(w.err, &be) {
		return fmt.Errorf("expected a *BaseURLError, got %T: %v", w.err, w.err)
	}
	if !strings.Contains(be.Source, FlagBaseURL) {
		return fmt.Errorf("error source %q does not name the flag %q", be.Source, FlagBaseURL)
	}
	return nil
}

func (w *baseURLWorld) thenNoFallThrough() error {
	if w.err == nil {
		return errors.New("expected an error, but resolution fell through to another source")
	}
	// On any fail-loud outcome the resolver returns the zero BaseURL — no
	// lower-precedence source (or the default) produced a value.
	if w.result.Value != "" {
		return fmt.Errorf("resolution returned a usable value %q despite the failure", w.result.Value)
	}
	return nil
}

func (w *baseURLWorld) thenValueIsDefault() error {
	if w.err != nil {
		return fmt.Errorf("unexpected error: %v", w.err)
	}
	if w.result.Value != DefaultBaseURL {
		return fmt.Errorf("Value = %q, want the built-in default %q", w.result.Value, DefaultBaseURL)
	}
	return nil
}

func (w *baseURLWorld) thenSourceDefault() error {
	if w.result.Source != SourceDefault {
		return fmt.Errorf("Source = %v, want Default", w.result.Source)
	}
	return nil
}

func (w *baseURLWorld) thenValueFromCurrentFile() error {
	if w.err != nil {
		return fmt.Errorf("unexpected error: %v", w.err)
	}
	if w.result.Source != SourceFile || w.result.Path != w.currPath {
		return fmt.Errorf("got %+v, want a File source at the current-directory file %s", w.result, w.currPath)
	}
	return nil
}

func (w *baseURLWorld) thenValueFromEnv() error {
	if w.err != nil {
		return fmt.Errorf("unexpected error: %v", w.err)
	}
	if w.result.Source != SourceEnvironment || w.result.Value != w.envValue {
		return fmt.Errorf("got %+v, want the GLASSFROG_BASE_URL value %q", w.result, w.envValue)
	}
	return nil
}

func (w *baseURLWorld) thenNoFileRead() error {
	// An env hit consults no file, so it carries no file path.
	if w.result.Source != SourceEnvironment || w.result.Path != "" {
		return fmt.Errorf("got %+v, want an Environment source with no file path", w.result)
	}
	return nil
}

func (w *baseURLWorld) thenSkippedBaseURLlessFile() error {
	if w.err != nil {
		return fmt.Errorf("unexpected error: %v", w.err)
	}
	if w.result.Path == w.currPath {
		return fmt.Errorf("resolution used the base_url-less file %s instead of skipping it", w.currPath)
	}
	return nil
}

func (w *baseURLWorld) thenValueFromHomeFile() error {
	if w.err != nil {
		return fmt.Errorf("unexpected error: %v", w.err)
	}
	if w.result.Source != SourceFile || w.result.Path != w.homePath {
		return fmt.Errorf("got %+v, want a File source at the home file %s", w.result, w.homePath)
	}
	return nil
}

func (w *baseURLWorld) thenNotEnvSource() error {
	if w.result.Source == SourceEnvironment {
		return errors.New("an empty GLASSFROG_BASE_URL was treated as a base URL")
	}
	return nil
}

func (w *baseURLWorld) thenReadErrorNamingFile() error {
	var re *rcfile.ReadError
	if !errors.As(w.err, &re) {
		return fmt.Errorf("expected a *rcfile.ReadError, got %T: %v", w.err, w.err)
	}
	if re.Path != w.currPath {
		return fmt.Errorf("ReadError.Path = %q, want %q", re.Path, w.currPath)
	}
	return nil
}

func (w *baseURLWorld) thenFormatErrorNamingFile() error {
	var be *BaseURLError
	if !errors.As(w.err, &be) {
		return fmt.Errorf("expected a *BaseURLError, got %T: %v", w.err, w.err)
	}
	if be.Source != w.currPath {
		return fmt.Errorf("error source = %q, want the file path %q", be.Source, w.currPath)
	}
	return nil
}

func (w *baseURLWorld) thenFormatErrorNamingEnv() error {
	var be *BaseURLError
	if !errors.As(w.err, &be) {
		return fmt.Errorf("expected a *BaseURLError, got %T: %v", w.err, w.err)
	}
	if be.Source != EnvVarBaseURL {
		return fmt.Errorf("error source = %q, want %q", be.Source, EnvVarBaseURL)
	}
	return nil
}
