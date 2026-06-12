package apiclient

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/Luscii/cli-glassfrog/internal/rcfile"
	"github.com/cucumber/godog"
)

// TestResolutionRetrofitBaseURLFeatures runs the base-URL slice of the Resolution
// Call-Site Retrofit (040) executable acceptance scenarios against the pure
// resolver (ResolveBaseURL), driving it over temp directory trees with injected
// roots, a controlled GLASSFROG_BASE_URL (the env seam), and a supplied flag
// value + presence bit — no real network and no real home directory.
//
// 040's feature file is cross-cutting (token / base URL / output); each setting
// is owned by its own package suite so the resolver stays hermetic over that
// package's own seams (see LEARNINGS). This suite filters "@base-url && ~@wip".
// The command-path-position scenario is driven here at the resolver level — both
// arg orders make cobra Changed() report the flag as supplied, so both resolve
// with presence true; the cobra position-independence itself is pinned by a
// RunE-level test in internal/cli.
func TestResolutionRetrofitBaseURLFeatures(t *testing.T) {
	suite := godog.TestSuite{
		ScenarioInitializer: initializeRetrofitBaseURLScenario,
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"../../features/duplicated-setting-resolution/resolution-call-site-retrofit.feature"},
			Tags:     "@base-url && ~@wip",
			TestingT: t,
		},
	}
	if suite.Run() != 0 {
		t.Fatal("non-zero status: base-URL retrofit feature scenarios failed")
	}
}

type retrofitBaseURLWorld struct {
	base    string
	currDir string
	homeDir string

	flagValue   string
	flagPresent bool
	envValue    string // "" models unset/empty (both fall through)

	currPath string // seeded current-dir .glassfrogrc, when present

	result BaseURL
	err    error

	// Second resolution for the command-path-position scenario.
	result2 BaseURL
	err2    error
}

func initializeRetrofitBaseURLScenario(sc *godog.ScenarioContext) {
	w := &retrofitBaseURLWorld{}

	origGetenv := getenv

	sc.Before(func(ctx context.Context, _ *godog.Scenario) (context.Context, error) {
		base, err := os.MkdirTemp("", "baseurl-retrofit-bdd-")
		if err != nil {
			return ctx, err
		}
		*w = retrofitBaseURLWorld{base: base}
		if w.currDir, err = mkdirAllBaseURL(filepath.Join(base, "work", "nested")); err != nil {
			return ctx, err
		}
		if w.homeDir, err = mkdirAllBaseURL(filepath.Join(base, "home")); err != nil {
			return ctx, err
		}
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

	// --- Givens ---
	sc.Step(`^the "--base-url" flag had not been supplied and GLASSFROG_BASE_URL was unset$`, w.givenNoFlagNoEnv)
	sc.Step(`^the nearest "\.glassfrogrc" carried "base_url = (.+)"$`, w.givenFileBaseURL)
	sc.Step(`^"--base-url (.+)" had been supplied$`, w.givenFlagSupplied)
	sc.Step(`^"--base-url" had been supplied with an empty or whitespace value$`, w.givenFlagSuppliedEmpty)
	sc.Step(`^"glassfrog --base-url .*both invoked$`, w.givenBothPositions)

	// --- Whens ---
	sc.Step(`^the base URL is resolved$`, w.whenResolve)
	sc.Step(`^each command resolves the base URL$`, w.whenResolveBoth)

	// --- Thens ---
	sc.Step(`^it will return that file value with the source reported as the file and its path$`, w.thenFromFile)
	sc.Step(`^it will return that value with the source reported as the flag$`, w.thenFromFlag)
	sc.Step(`^it will not consult the environment or any "\.glassfrogrc"$`, w.thenConsultedNoOtherSource)
	sc.Step(`^it will fail with a usage error naming "--base-url"$`, w.thenUsageErrorNamingFlag)
	sc.Step(`^no lower-precedence source will be consulted$`, w.thenNoFallThrough)
	sc.Step(`^the flag will win its rung by its presence$`, w.thenFlagWonByPresence)
	sc.Step(`^it will not fall through to the environment$`, w.thenNoFallThrough)
	sc.Step(`^both will detect the flag as supplied and fail with the same usage error$`, w.thenBothSameUsageError)
}

// --- Given implementations ---

func (w *retrofitBaseURLWorld) givenNoFlagNoEnv() error {
	w.flagValue, w.flagPresent, w.envValue = "", false, ""
	return nil
}

func (w *retrofitBaseURLWorld) givenFileBaseURL(v string) error {
	path, err := seedBaseURLRC(w.currDir, "base_url="+v+"\n")
	if err != nil {
		return err
	}
	w.currPath = path
	return nil
}

func (w *retrofitBaseURLWorld) givenFlagSupplied(v string) error {
	w.flagValue, w.flagPresent = v, true
	return nil
}

func (w *retrofitBaseURLWorld) givenFlagSuppliedEmpty() error {
	// An explicit empty/whitespace flag: supplied (present) with a blank value.
	w.flagValue, w.flagPresent = "   ", true
	// Seed a usable env value so a (wrong) fall-through would be observable.
	w.envValue = "https://env.example.com/api/v5"
	return nil
}

func (w *retrofitBaseURLWorld) givenBothPositions() error {
	// cobra Changed() reports the flag as supplied regardless of where it sits on
	// the command path, so both invocations resolve with presence true and a blank
	// value. Seed an env value so a fall-through would be observable.
	w.flagValue, w.flagPresent = "", true
	w.envValue = "https://env.example.com/api/v5"
	return nil
}

// --- When implementations ---

func (w *retrofitBaseURLWorld) whenResolve() error {
	w.result, w.err = ResolveBaseURL(w.flagValue, w.flagPresent, w.currDir, w.homeDir)
	return nil
}

func (w *retrofitBaseURLWorld) whenResolveBoth() error {
	// Both arg orders make Changed() report the flag as supplied — model both as
	// present, then assert they fail identically.
	w.result, w.err = ResolveBaseURL("", true, w.currDir, w.homeDir)
	w.result2, w.err2 = ResolveBaseURL("", true, w.currDir, w.homeDir)
	return nil
}

// --- Then implementations ---

func (w *retrofitBaseURLWorld) thenFromFile() error {
	if w.err != nil {
		return fmt.Errorf("unexpected error: %v", w.err)
	}
	if w.result.Source != SourceFile || w.result.Path != w.currPath {
		return fmt.Errorf("got %+v, want a File source at %s", w.result, w.currPath)
	}
	return nil
}

func (w *retrofitBaseURLWorld) thenFromFlag() error {
	if w.err != nil {
		return fmt.Errorf("unexpected error: %v", w.err)
	}
	if w.result.Source != SourceFlag || w.result.Value != w.flagValue {
		return fmt.Errorf("got %+v, want the flag value %q", w.result, w.flagValue)
	}
	return nil
}

func (w *retrofitBaseURLWorld) thenConsultedNoOtherSource() error {
	if w.err != nil {
		return fmt.Errorf("a lower source surfaced an error: %v", w.err)
	}
	if w.result.Source != SourceFlag || w.result.Path != "" {
		return fmt.Errorf("got %+v, want a Flag source with no file path", w.result)
	}
	return nil
}

func (w *retrofitBaseURLWorld) thenUsageErrorNamingFlag() error {
	var be *BaseURLError
	if !errors.As(w.err, &be) {
		return fmt.Errorf("expected a *BaseURLError, got %T: %v", w.err, w.err)
	}
	if be.Source != "--"+FlagBaseURL {
		return fmt.Errorf("error source = %q, want %q", be.Source, "--"+FlagBaseURL)
	}
	return nil
}

func (w *retrofitBaseURLWorld) thenNoFallThrough() error {
	if w.err == nil {
		return errors.New("expected an error, but resolution fell through to another source")
	}
	if w.result.Value != "" {
		return fmt.Errorf("resolution returned a usable value %q despite the failure", w.result.Value)
	}
	return nil
}

func (w *retrofitBaseURLWorld) thenFlagWonByPresence() error {
	// The flag won its rung even though its value is blank: a *BaseURLError naming
	// the flag (not the env) is the proof — the walk stopped at the flag rung.
	var be *BaseURLError
	if !errors.As(w.err, &be) {
		return fmt.Errorf("expected the flag to win and fail loud (*BaseURLError), got %T: %v", w.err, w.err)
	}
	if be.Source != "--"+FlagBaseURL {
		return fmt.Errorf("error source = %q, want the flag %q (it did not win its rung by presence)", be.Source, "--"+FlagBaseURL)
	}
	return nil
}

func (w *retrofitBaseURLWorld) thenBothSameUsageError() error {
	for i, e := range []error{w.err, w.err2} {
		var be *BaseURLError
		if !errors.As(e, &be) {
			return fmt.Errorf("invocation %d: expected a *BaseURLError, got %T: %v", i+1, e, e)
		}
		if be.Source != "--"+FlagBaseURL {
			return fmt.Errorf("invocation %d: error source = %q, want %q", i+1, be.Source, "--"+FlagBaseURL)
		}
	}
	if w.err.Error() != w.err2.Error() {
		return fmt.Errorf("the two invocations produced different errors: %q vs %q", w.err, w.err2)
	}
	return nil
}

// --- file helpers (040 base-URL slice) ---

func mkdirAllBaseURL(dir string) (string, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("mkdir %s: %w", dir, err)
	}
	return dir, nil
}

func seedBaseURLRC(dir, content string) (string, error) {
	if _, err := mkdirAllBaseURL(dir); err != nil {
		return "", err
	}
	path := filepath.Join(dir, rcfile.FileName)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		return "", fmt.Errorf("seed %s: %w", path, err)
	}
	return path, nil
}
