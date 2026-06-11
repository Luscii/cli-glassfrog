package resolve

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

// TestResolveFeatures runs the Source-Composed Resolution (039) executable
// acceptance scenarios against the pure resolver, composing sources over an
// injected env lookup, temp directories, and in-memory stdin readers — no real
// network, home directory, or terminal.
//
// godog binds steps per-suite, so this suite names only its own feature file
// (LEARNINGS 2026-06-04). The three @validation scenarios stay @wip — held out
// for the validate skill, not implemented by the Builder — so the suite filters
// them with Tags "~@wip".
func TestResolveFeatures(t *testing.T) {
	suite := godog.TestSuite{
		ScenarioInitializer: initializeResolveScenario,
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"../../features/duplicated-setting-resolution/source-composed-resolution.feature"},
			Tags:     "~@wip",
			TestingT: t,
		},
	}
	if suite.Run() != 0 {
		t.Fatal("non-zero status: source-composed-resolution feature scenarios failed")
	}
}

const (
	envVarName   = "GLASSFROG_SETTING" // a setting-agnostic placeholder env name
	fileKey      = "setting"           // a setting-agnostic .glassfrogrc key
	defaultValue = "the-default"
)

// resolveWorld is the per-scenario state. composition selects how whenResolve
// assembles the source list; the remaining fields parameterize the individual
// sources. The injected env lookup and temp directories keep every scenario
// hermetic.
type resolveWorld struct {
	base    string // root temp dir, removed in After
	currDir string
	homeDir string

	composition string // selects the source list whenResolve builds

	flag       Flag   // the single flag source (full composition)
	aliasFlags []Flag // the alias flag source (list-valued composition)

	envValue string // "" models unset/empty (both fall through)

	filePath  string // seeded .glassfrogrc path, for assertion
	malformed bool

	stdinReadFailed bool
	stdinDrained    bool // tripwire: set if a stdin reader is ever invoked

	result   Resolution
	err      error
	panicked bool
	panicMsg string
}

func (w *resolveWorld) lookup(name string) string {
	if name == envVarName {
		return w.envValue
	}
	return ""
}

func (w *resolveWorld) mkdir(dir string) (string, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("mkdir %s: %w", dir, err)
	}
	return dir, nil
}

func (w *resolveWorld) seed(dir, content string) (string, error) {
	if _, err := w.mkdir(dir); err != nil {
		return "", err
	}
	path := filepath.Join(dir, rcfile.FileName)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		return "", fmt.Errorf("seed %s: %w", path, err)
	}
	return path, nil
}

func initializeResolveScenario(sc *godog.ScenarioContext) {
	w := &resolveWorld{}

	sc.Before(func(ctx context.Context, _ *godog.Scenario) (context.Context, error) {
		base, err := os.MkdirTemp("", "resolve-bdd-")
		if err != nil {
			return ctx, err
		}
		*w = resolveWorld{base: base, flag: Flag{Name: "--flag"}}
		if w.currDir, err = w.mkdir(filepath.Join(base, "work", "nested")); err != nil {
			return ctx, err
		}
		if w.homeDir, err = w.mkdir(filepath.Join(base, "home")); err != nil {
			return ctx, err
		}
		return ctx, nil
	})

	sc.After(func(ctx context.Context, _ *godog.Scenario, _ error) (context.Context, error) {
		if w.base != "" {
			os.RemoveAll(w.base)
		}
		return ctx, nil
	})

	// --- Givens: composition ---
	sc.Step(`^a resolver composed of a flag source, an environment source, a file source, and a trailing default$`, w.givenFullComposition)
	sc.Step(`^a resolver composed of an environment source and a file source with no trailing default$`, w.givenEnvFileComposition)
	sc.Step(`^a flag source composed over the aliases "([^"]*)" and "([^"]*)"$`, w.givenAliasComposition)
	sc.Step(`^a resolver composed with two stdin sources$`, w.givenTwoStdinComposition)
	sc.Step(`^a resolver whose file source reads "\.glassfrogrc"$`, w.givenFileOnlyComposition)
	sc.Step(`^a resolver whose highest-precedence source reads piped stdin through an injected reader$`, w.givenStdinFirstComposition)

	// --- Givens: source state ---
	sc.Step(`^the flag had been supplied with a value$`, w.givenFlagSupplied)
	sc.Step(`^the flag had not been supplied$`, w.givenFlagNotSupplied)
	sc.Step(`^the environment variable had been set to a non-empty value$`, w.givenEnvSet)
	sc.Step(`^no flag, environment variable, or "\.glassfrogrc" key had supplied a value$`, noop)
	sc.Step(`^neither the variable nor any "\.glassfrogrc" had carried the key$`, noop)
	sc.Step(`^"([^"]*)" had not been supplied but "([^"]*)" had been supplied with a value$`, w.givenAliasState)
	sc.Step(`^no higher-precedence source had yielded$`, noop)
	sc.Step(`^the nearest "\.glassfrogrc" was unreadable or malformed$`, w.givenMalformedFile)
	sc.Step(`^reading the piped stdin had failed$`, w.givenStdinReadFailed)

	// --- When ---
	sc.Step(`^the setting is resolved$`, w.whenResolve)

	// --- Thens ---
	sc.Step(`^it will return the flag's value$`, w.thenReturnsFlagValue)
	sc.Step(`^it will report the provenance as the flag$`, w.thenProvenanceFlag)
	sc.Step(`^it will skip the empty flag source$`, w.thenSkippedFlag)
	sc.Step(`^it will return the environment value$`, w.thenReturnsEnvValue)
	sc.Step(`^it will report the provenance as the environment variable$`, w.thenProvenanceEnv)
	sc.Step(`^it will return the default value$`, w.thenReturnsDefault)
	sc.Step(`^it will report the provenance as the default$`, w.thenProvenanceDefault)
	sc.Step(`^it will report the provenance as nothing-found$`, w.thenProvenanceNone)
	sc.Step(`^it will report no error$`, w.thenNoError)
	sc.Step(`^it will return the value from "([^"]*)"$`, w.thenReturnsAliasValue)
	sc.Step(`^it will report the provenance origin as "([^"]*)"$`, w.thenProvenanceOrigin)
	sc.Step(`^it will fail loudly as a composition error$`, w.thenComposistionPanic)
	sc.Step(`^it will not drain the stream for the first reader$`, w.thenStreamNotDrained)
	sc.Step(`^it will surface a resolution error naming that file path$`, w.thenResolutionErrorNamingFile)
	sc.Step(`^it will not fall through to a lower-precedence source$`, w.thenNoFallThrough)
	sc.Step(`^it will surface the read error directly, aborting resolution the same way a config-file failure does$`, w.thenSurfacesReadError)
}

func noop() error { return nil }

// --- composition Givens ---

func (w *resolveWorld) givenFullComposition() error    { w.composition = "full"; return nil }
func (w *resolveWorld) givenEnvFileComposition() error { w.composition = "env-file"; return nil }
func (w *resolveWorld) givenTwoStdinComposition() error {
	w.composition = "two-stdin"
	return nil
}
func (w *resolveWorld) givenFileOnlyComposition() error   { w.composition = "file-only"; return nil }
func (w *resolveWorld) givenStdinFirstComposition() error { w.composition = "stdin-first"; return nil }

func (w *resolveWorld) givenAliasComposition(first, second string) error {
	w.composition = "alias"
	w.aliasFlags = []Flag{{Name: first}, {Name: second}}
	return nil
}

// --- source-state Givens ---

func (w *resolveWorld) givenFlagSupplied() error {
	w.flag.Present = true
	w.flag.Value = "flag-value"
	return nil
}

func (w *resolveWorld) givenFlagNotSupplied() error { w.flag.Present = false; return nil }
func (w *resolveWorld) givenEnvSet() error          { w.envValue = "env-value"; return nil }

func (w *resolveWorld) givenAliasState(notSupplied, supplied string) error {
	for i := range w.aliasFlags {
		switch w.aliasFlags[i].Name {
		case notSupplied:
			w.aliasFlags[i].Present = false
		case supplied:
			w.aliasFlags[i].Present = true
			w.aliasFlags[i].Value = "alias-value"
		}
	}
	return nil
}

func (w *resolveWorld) givenMalformedFile() error {
	// A non-blank, non-comment line with no '=' makes rcfile.Resolve return a
	// typed *FormatError naming the path — fail-loud without leaking contents.
	path, err := w.seed(w.currDir, "not-a-key-value-line\n")
	if err != nil {
		return err
	}
	w.filePath = path
	w.malformed = true
	return nil
}

func (w *resolveWorld) givenStdinReadFailed() error { w.stdinReadFailed = true; return nil }

// --- When ---

func (w *resolveWorld) whenResolve() error {
	defer func() {
		if r := recover(); r != nil {
			w.panicked = true
			w.panicMsg = fmt.Sprint(r)
		}
	}()
	w.result, w.err = Resolve(w.buildSources()...)
	return nil
}

// buildSources assembles the precedence list for the scenario's composition from
// the world's per-source state.
func (w *resolveWorld) buildSources() []Source {
	switch w.composition {
	case "full":
		return []Source{
			FromFlags(w.flag),
			FromEnv(w.lookup, envVarName),
			FromFile(w.currDir, w.homeDir, fileKey),
			Default(defaultValue),
		}
	case "env-file":
		return []Source{
			FromEnv(w.lookup, envVarName),
			FromFile(w.currDir, w.homeDir, fileKey),
		}
	case "alias":
		return []Source{FromFlags(w.aliasFlags...)}
	case "two-stdin":
		read := func() (string, error) { w.stdinDrained = true; return "x", nil }
		return []Source{FromStdin(read, false), FromStdin(read, false)}
	case "file-only":
		// A trailing default proves the malformed-file error aborts the walk
		// rather than falling through to a lower-precedence source.
		return []Source{
			FromFile(w.currDir, w.homeDir, fileKey),
			Default(defaultValue),
		}
	case "stdin-first":
		read := func() (string, error) {
			w.stdinDrained = true
			if w.stdinReadFailed {
				return "", errStdinBoom
			}
			return "", nil
		}
		return []Source{FromStdin(read, false), Default(defaultValue)}
	default:
		return nil
	}
}

var errStdinBoom = errors.New("stdin read failed")

// --- Thens ---

func (w *resolveWorld) thenReturnsFlagValue() error {
	if w.err != nil {
		return fmt.Errorf("unexpected error: %v", w.err)
	}
	if w.result.Provenance.Kind != KindFlag || w.result.Value != "flag-value" {
		return fmt.Errorf("got %+v, want the flag's value", w.result)
	}
	return nil
}

func (w *resolveWorld) thenProvenanceFlag() error {
	if w.result.Provenance.Kind != KindFlag {
		return fmt.Errorf("provenance kind = %v, want flag", w.result.Provenance.Kind)
	}
	return nil
}

func (w *resolveWorld) thenSkippedFlag() error {
	if w.result.Provenance.Kind == KindFlag {
		return errors.New("the empty flag source won; it should have been skipped")
	}
	return nil
}

func (w *resolveWorld) thenReturnsEnvValue() error {
	if w.err != nil {
		return fmt.Errorf("unexpected error: %v", w.err)
	}
	if w.result.Provenance.Kind != KindEnv || w.result.Value != "env-value" {
		return fmt.Errorf("got %+v, want the environment value", w.result)
	}
	return nil
}

func (w *resolveWorld) thenProvenanceEnv() error {
	if w.result.Provenance.Kind != KindEnv || w.result.Provenance.Origin != envVarName {
		return fmt.Errorf("got %+v, want KindEnv with Origin %q", w.result.Provenance, envVarName)
	}
	return nil
}

func (w *resolveWorld) thenReturnsDefault() error {
	if w.err != nil {
		return fmt.Errorf("unexpected error: %v", w.err)
	}
	if w.result.Provenance.Kind != KindDefault || w.result.Value != defaultValue {
		return fmt.Errorf("got %+v, want the default value", w.result)
	}
	return nil
}

func (w *resolveWorld) thenProvenanceDefault() error {
	if w.result.Provenance.Kind != KindDefault {
		return fmt.Errorf("provenance kind = %v, want default", w.result.Provenance.Kind)
	}
	return nil
}

func (w *resolveWorld) thenProvenanceNone() error {
	if w.result.Found() || w.result.Provenance.Kind != KindNone {
		return fmt.Errorf("got %+v, want a KindNone nothing-found result", w.result)
	}
	return nil
}

func (w *resolveWorld) thenNoError() error {
	if w.err != nil {
		return fmt.Errorf("expected no error, got %v", w.err)
	}
	return nil
}

func (w *resolveWorld) thenReturnsAliasValue(alias string) error {
	if w.err != nil {
		return fmt.Errorf("unexpected error: %v", w.err)
	}
	if w.result.Value != "alias-value" || w.result.Provenance.Origin != alias {
		return fmt.Errorf("got %+v, want the value from alias %q", w.result, alias)
	}
	return nil
}

func (w *resolveWorld) thenProvenanceOrigin(origin string) error {
	if w.result.Provenance.Origin != origin {
		return fmt.Errorf("provenance origin = %q, want %q", w.result.Provenance.Origin, origin)
	}
	return nil
}

func (w *resolveWorld) thenComposistionPanic() error {
	if !w.panicked {
		return errors.New("expected a composition-error panic, but Resolve returned normally")
	}
	if !strings.Contains(w.panicMsg, "Stdin") {
		return fmt.Errorf("panic message %q does not name the Stdin misuse", w.panicMsg)
	}
	return nil
}

func (w *resolveWorld) thenStreamNotDrained() error {
	if w.stdinDrained {
		return errors.New("a stdin source was evaluated; the stream was drained before the guard fired")
	}
	return nil
}

func (w *resolveWorld) thenResolutionErrorNamingFile() error {
	if w.err == nil {
		return errors.New("expected a resolution error, got nil")
	}
	var fe *rcfile.FormatError
	var re *rcfile.ReadError
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
		return fmt.Errorf("err = %T %v, want a typed rcfile error naming the file", w.err, w.err)
	}
	return nil
}

func (w *resolveWorld) thenNoFallThrough() error {
	if w.err == nil {
		return errors.New("expected an error, but resolution fell through")
	}
	if w.result.Value != "" {
		return fmt.Errorf("resolution returned %q despite the failure (fell through)", w.result.Value)
	}
	return nil
}

func (w *resolveWorld) thenSurfacesReadError() error {
	if !errors.Is(w.err, errStdinBoom) {
		return fmt.Errorf("err = %v, want the verbatim stdin read error", w.err)
	}
	if w.result.Value != "" {
		return fmt.Errorf("resolution returned %q despite the read failure (fell through)", w.result.Value)
	}
	return nil
}
