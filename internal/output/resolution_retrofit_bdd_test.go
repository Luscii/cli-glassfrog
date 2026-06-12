package output

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

// TestResolutionRetrofitOutputFeatures runs the output slice of the Resolution
// Call-Site Retrofit (040) executable acceptance scenarios against the composing
// entry ResolveSelectionFromOS, driving it over the package getenv seam and
// temp-dir .glassfrogrc files — never the real environment or ~/.glassfrogrc
// (ADR-4). 040's feature file is cross-cutting; each setting is owned by its own
// package suite (see LEARNINGS), so this suite filters "@output && ~@wip".
func TestResolutionRetrofitOutputFeatures(t *testing.T) {
	suite := godog.TestSuite{
		ScenarioInitializer: initializeRetrofitOutputScenario,
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"../../features/duplicated-setting-resolution/resolution-call-site-retrofit.feature"},
			Tags:     "@output && ~@wip",
			TestingT: t,
		},
	}
	if suite.Run() != 0 {
		t.Fatal("non-zero status: output retrofit feature scenarios failed")
	}
}

type retrofitOutputWorld struct {
	base    string
	currDir string
	homeDir string

	flagValue   string
	flagPresent bool
	envValue    string // "" models unset; whitespace models a present-but-blank env value

	filePath string // seeded .glassfrogrc, when present

	sel Selection
	err error
}

func initializeRetrofitOutputScenario(sc *godog.ScenarioContext) {
	w := &retrofitOutputWorld{}
	origGetenv := getenv

	sc.Before(func(ctx context.Context, _ *godog.Scenario) (context.Context, error) {
		base, err := os.MkdirTemp("", "output-retrofit-bdd-")
		if err != nil {
			return ctx, err
		}
		*w = retrofitOutputWorld{base: base}
		if w.currDir, err = os.MkdirTemp(base, "work"); err != nil {
			return ctx, err
		}
		if w.homeDir, err = os.MkdirTemp(base, "home"); err != nil {
			return ctx, err
		}
		getenv = func(key string) string {
			if key == EnvVarOutput {
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
	sc.Step(`^no "--output" flag, no GLASSFROG_OUTPUT, and no "\.glassfrogrc" output key$`, w.givenAllAbsent)
	sc.Step(`^no "--output" flag and no GLASSFROG_OUTPUT$`, w.givenFlagEnvAbsent)
	sc.Step(`^the nearest "\.glassfrogrc" was malformed$`, w.givenMalformedFile)
	sc.Step(`^the "--output" flag had not been supplied and GLASSFROG_OUTPUT was set to whitespace only$`, w.givenWhitespaceEnv)
	sc.Step(`^no "\.glassfrogrc" output key was present$`, func() error { return nil })

	// --- When ---
	sc.Step(`^the output selection is resolved$`, w.whenResolve)

	// --- Thens ---
	sc.Step(`^it will return the built-in default "full"$`, w.thenDefaultFull)
	sc.Step(`^it will report no source on success \(output surfaces provenance only on a format error\)$`, w.thenNoSourceOnSuccess)
	sc.Step(`^it will surface the typed config read error naming that file$`, w.thenTypedConfigError)
	sc.Step(`^it will not fall through to the built-in default$`, w.thenDidNotUseDefault)
	sc.Step(`^it will treat the environment value as absent$`, w.thenEnvTreatedAbsent)
}

// --- Given implementations ---

func (w *retrofitOutputWorld) givenAllAbsent() error     { return nil } // Before seeds nothing
func (w *retrofitOutputWorld) givenFlagEnvAbsent() error { return nil }

func (w *retrofitOutputWorld) givenMalformedFile() error {
	// A line that is neither blank, a comment, nor a key=value pair is unparseable,
	// so rcfile fails loud with a typed *FormatError naming the file.
	path := filepath.Join(w.currDir, rcfile.FileName)
	if err := os.WriteFile(path, []byte("this line has no equals sign\n"), 0o600); err != nil {
		return err
	}
	w.filePath = path
	return nil
}

func (w *retrofitOutputWorld) givenWhitespaceEnv() error {
	w.envValue = "   "
	return nil
}

// --- When implementation ---

func (w *retrofitOutputWorld) whenResolve() error {
	w.sel, w.err = ResolveSelectionFromOS(w.flagValue, w.flagPresent, w.currDir, w.homeDir)
	return nil
}

// --- Then implementations ---

func (w *retrofitOutputWorld) thenDefaultFull() error {
	if w.err != nil {
		return fmt.Errorf("unexpected error: %v", w.err)
	}
	if _, ok := w.sel.AsTemplate(); ok {
		return fmt.Errorf("expected the built-in default, got a template: %+v", w.sel)
	}
	if w.sel.Format != FormatFull {
		return fmt.Errorf("resolved %v, want the built-in default full", w.sel.Format)
	}
	return nil
}

func (w *retrofitOutputWorld) thenNoSourceOnSuccess() error {
	// Output surfaces provenance only on a *FormatError; a successful selection is a
	// bare format with no source to report. The success + format check is the proof.
	if w.err != nil {
		return fmt.Errorf("a successful resolution carried an error: %v", w.err)
	}
	return nil
}

func (w *retrofitOutputWorld) thenTypedConfigError() error {
	var re *rcfile.ReadError
	var fe *rcfile.FormatError
	switch {
	case errors.As(w.err, &fe):
		if fe.Path != w.filePath {
			return fmt.Errorf("FormatError.Path = %q, want %q", fe.Path, w.filePath)
		}
	case errors.As(w.err, &re):
		if re.Path != w.filePath {
			return fmt.Errorf("ReadError.Path = %q, want %q", re.Path, w.filePath)
		}
	default:
		return fmt.Errorf("expected a typed rcfile read/format error naming the file, got %T: %v", w.err, w.err)
	}
	return nil
}

func (w *retrofitOutputWorld) thenDidNotUseDefault() error {
	if w.err == nil {
		return errors.New("expected a config error, but resolution fell through to the default")
	}
	// On the fail-loud path the placeholder Selection is the zero format; the error,
	// not the value, is what the caller acts on.
	return nil
}

func (w *retrofitOutputWorld) thenEnvTreatedAbsent() error {
	if w.err != nil {
		return fmt.Errorf("a whitespace env value should fall through, not error: %v", w.err)
	}
	if _, ok := w.sel.AsTemplate(); ok {
		return fmt.Errorf("expected the default format, got a template: %+v", w.sel)
	}
	if w.sel.Format != FormatFull {
		return fmt.Errorf("whitespace env should fall through to the default full, resolved %v", w.sel.Format)
	}
	return nil
}
