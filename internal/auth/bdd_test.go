package auth

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/cucumber/godog"
)

// TestFeatures runs the Credential Discovery (005) executable acceptance
// scenarios in features/unauthenticated-access.feature against the production
// seam (Resolve) over temp directory trees and a controlled GLASSFROG_TOKEN.
// @wip scenarios are skipped — the three @validation scenarios stay @wip
// because they are held out for independent verification (the validate skill),
// not implemented by the Builder.
//
// This suite owns unauthenticated-access.feature; the cli package's suite owns
// no-runnable-cli.feature. Per-package ownership keeps each suite pointed only
// at the feature whose steps it defines.
func TestFeatures(t *testing.T) {
	suite := godog.TestSuite{
		ScenarioInitializer: initializeScenario,
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"../../features/unauthenticated-access.feature"},
			Tags:     "~@wip",
			TestingT: t,
		},
	}
	if suite.Run() != 0 {
		t.Fatal("non-zero status: feature scenarios failed")
	}
}

// authWorld is the per-scenario state. It builds a temp filesystem and drives
// the production seam (Resolve) by binding getwd/userHomeDir to scenario-chosen
// temp directories and setting/unsetting the real GLASSFROG_TOKEN — never the
// developer's real home directory or working directory.
type authWorld struct {
	base    string // root temp dir for this scenario, removed in After
	homeDir string
	currDir string

	homePath     string // seeded home .glassfrogrc, when present
	currPath     string // seeded current .glassfrogrc with a token, when present
	ancestorPath string // seeded .glassfrogrc two dirs above current, when present
	tokenlessDir string // current dir holding a tokenless file, when present

	res Resolution
	err error
}

// mkdir and seed return errors rather than panicking: a panic inside a step
// would skip godog's After hook, leaving the getwd/userHomeDir/GLASSFROG_TOKEN
// globals mutated and the temp tree undeleted, which leaks into later
// scenarios. Returning the error keeps every step on godog's normal failure
// path, so cleanup always runs.
func (w *authWorld) mkdir(dir string) (string, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("mkdir %s: %w", dir, err)
	}
	return dir, nil
}

func (w *authWorld) seed(dir, content string, mode os.FileMode) (string, error) {
	if _, err := w.mkdir(dir); err != nil {
		return "", err
	}
	path := filepath.Join(dir, credentialsFileName)
	if err := os.WriteFile(path, []byte(content), mode); err != nil {
		return "", fmt.Errorf("seed %s: %w", path, err)
	}
	return path, nil
}

func initializeScenario(sc *godog.ScenarioContext) {
	w := &authWorld{}

	// Capture the ambient GLASSFROG_TOKEN and the OS seams once, restoring them
	// after every scenario so the suite is hermetic and order-independent.
	origToken, tokenWasSet := os.LookupEnv(envTokenVar)
	origGetwd, origHome := getwd, userHomeDir

	sc.Before(func(ctx context.Context, _ *godog.Scenario) (context.Context, error) {
		base, err := os.MkdirTemp("", "auth-bdd-")
		if err != nil {
			return ctx, err
		}
		*w = authWorld{base: base}
		if w.homeDir, err = w.mkdir(filepath.Join(base, "home")); err != nil {
			return ctx, err
		}
		// Default current directory is independent of home (a sibling subtree),
		// so home is the final fallback rather than an ancestor unless a
		// scenario reconfigures it.
		if w.currDir, err = w.mkdir(filepath.Join(base, "work", "nested")); err != nil {
			return ctx, err
		}

		// Bind the seams to this scenario's directories; reads happen live so a
		// later Given that moves currDir takes effect.
		getwd = func() (string, error) { return w.currDir, nil }
		userHomeDir = func() (string, error) { return w.homeDir, nil }
		os.Unsetenv(envTokenVar)
		return ctx, nil
	})

	sc.After(func(ctx context.Context, _ *godog.Scenario, _ error) (context.Context, error) {
		getwd, userHomeDir = origGetwd, origHome
		if tokenWasSet {
			os.Setenv(envTokenVar, origToken)
		} else {
			os.Unsetenv(envTokenVar)
		}
		if w.base != "" {
			os.RemoveAll(w.base)
		}
		return ctx, nil
	})

	// --- Givens: environment ---
	sc.Step(`^GLASSFROG_TOKEN was not set$`, func() error { os.Unsetenv(envTokenVar); return nil })
	sc.Step(`^GLASSFROG_TOKEN was set to "([^"]*)"$`, func(v string) error { return os.Setenv(envTokenVar, v) })
	sc.Step(`^GLASSFROG_TOKEN was set to an empty value$`, func() error { return os.Setenv(envTokenVar, "") })

	// --- Givens: files ---
	sc.Step(`^a "\.glassfrogrc" in the home directory held the token "([^"]*)"$`, w.givenHomeToken)
	sc.Step(`^a "\.glassfrogrc" in the current directory held the token "([^"]*)"$`, w.givenCurrentToken)
	sc.Step(`^a "\.glassfrogrc" in the current directory existed with no token entry$`, w.givenCurrentTokenless)
	sc.Step(`^a "\.glassfrogrc" two directories above held the token "([^"]*)"$`, w.givenAncestorToken)
	sc.Step(`^the home directory was an ancestor of the current directory$`, w.givenHomeIsAncestor)
	sc.Step(`^the only "\.glassfrogrc" lived in the home directory holding "([^"]*)"$`, w.givenHomeToken)
	sc.Step(`^the nearest "\.glassfrogrc" existed but could not be read$`, w.givenNearestUnreadable)
	sc.Step(`^the nearest "\.glassfrogrc" held a line that was neither blank, a comment, nor a "key=value" pair$`, w.givenNearestMalformed)

	// --- Givens: absence (no-ops — Before seeds nothing) ---
	noFile := func() error { return nil }
	sc.Step(`^no "\.glassfrogrc" existed in the current directory or any ancestor$`, noFile)
	sc.Step(`^no "\.glassfrogrc" existed in the current directory, any ancestor, or the home directory$`, noFile)
	sc.Step(`^no "\.glassfrogrc" existed in the current directory$`, noFile)
	sc.Step(`^no "\.glassfrogrc" existed in the home directory$`, noFile)

	// --- Whens ---
	sc.Step(`^the CLI resolves the credential$`, w.whenResolve)
	sc.Step(`^the CLI resolves the credential from the current directory$`, w.whenResolve)

	// --- Thens ---
	sc.Step(`^it will use the token from the home file$`, w.thenTokenFromHome)
	sc.Step(`^it will skip the tokenless file$`, w.thenSkippedTokenless)
	sc.Step(`^it will report that no credentials were found$`, w.thenNone)
	sc.Step(`^it will not fabricate a token$`, w.thenNoToken)
	sc.Step(`^it will not raise an error of its own$`, w.thenNoError)
	sc.Step(`^it will report a read error naming that file$`, w.thenReadErrorNaming)
	sc.Step(`^it will not fall through to another source$`, w.thenNoFallThrough)
	sc.Step(`^it will report a format error naming that file$`, w.thenFormatErrorNaming)
	sc.Step(`^it will not report that no credentials were found$`, w.thenNotNone)
	sc.Step(`^it will use the token from GLASSFROG_TOKEN$`, w.thenTokenFromEnv)
	sc.Step(`^it will report the source as the environment$`, w.thenSourceEnvironment)
	sc.Step(`^it will not read any credentials file$`, w.thenNoFileRead)
	sc.Step(`^it will not treat the empty variable as a token$`, w.thenNotEnvSource)
	sc.Step(`^it will use the token from the credentials file$`, w.thenTokenFromCurrentFile)
	sc.Step(`^it will use the token from the current directory's file$`, w.thenTokenFromCurrentFile)
	sc.Step(`^it will use the token from the ancestor file$`, w.thenTokenFromAncestor)
	sc.Step(`^it will report a File source with that file's path$`, w.thenFileSourceWithHomePath)
}

// --- Given implementations ---

func (w *authWorld) givenHomeToken(v string) error {
	path, err := w.seed(w.homeDir, "token="+v+"\n", 0o600)
	if err != nil {
		return err
	}
	w.homePath = path
	return nil
}

func (w *authWorld) givenCurrentToken(v string) error {
	path, err := w.seed(w.currDir, "token="+v+"\n", 0o600)
	if err != nil {
		return err
	}
	w.currPath = path
	return nil
}

func (w *authWorld) givenCurrentTokenless() error {
	w.tokenlessDir = w.currDir
	_, err := w.seed(w.currDir, "# a comment but no token entry\n", 0o600)
	return err
}

func (w *authWorld) givenAncestorToken(v string) error {
	// currDir is base/work/nested; two directories above is base.
	path, err := w.seed(w.base, "token="+v+"\n", 0o600)
	if err != nil {
		return err
	}
	w.ancestorPath = path
	return nil
}

func (w *authWorld) givenHomeIsAncestor() error {
	// Reconfigure the working directory to sit beneath the home directory so
	// the home file is encountered during the walk-up rather than as the final
	// fallback.
	dir, err := w.mkdir(filepath.Join(w.homeDir, "projects", "app"))
	if err != nil {
		return err
	}
	w.currDir = dir
	return nil
}

func (w *authWorld) givenNearestUnreadable() error {
	// Create a directory at the .glassfrogrc path. os.ReadFile fails on a
	// directory deterministically across platforms (and as root), so this
	// exercises the resolver's fail-loud ReadError branch without relying on
	// 0o000 permission semantics that vary by OS / privilege.
	path := filepath.Join(w.currDir, credentialsFileName)
	if _, err := w.mkdir(path); err != nil {
		return err
	}
	w.currPath = path
	return nil
}

func (w *authWorld) givenNearestMalformed() error {
	// No '=' anywhere on the line, so it is genuinely unparseable.
	path, err := w.seed(w.currDir, "this line is neither blank a comment nor a pair\n", 0o600)
	if err != nil {
		return err
	}
	w.currPath = path
	return nil
}

// --- When implementation ---

func (w *authWorld) whenResolve() error {
	w.res, w.err = Resolve()
	return nil
}

// --- Then implementations ---

func (w *authWorld) thenTokenFromHome() error {
	if w.err != nil {
		return fmt.Errorf("unexpected error: %v", w.err)
	}
	if w.res.Source != SourceFile || w.res.Path != w.homePath {
		return fmt.Errorf("got %+v, want a File source at the home file %s", w.res, w.homePath)
	}
	return nil
}

func (w *authWorld) thenSkippedTokenless() error {
	if w.err != nil {
		return fmt.Errorf("unexpected error: %v", w.err)
	}
	tokenlessPath := filepath.Join(w.tokenlessDir, credentialsFileName)
	if w.res.Path == tokenlessPath {
		return fmt.Errorf("resolution used the tokenless file %s instead of skipping it", tokenlessPath)
	}
	return nil
}

func (w *authWorld) thenNone() error {
	if w.err != nil {
		return fmt.Errorf("absence reported as an error: %v", w.err)
	}
	if w.res.Source != SourceNone {
		return fmt.Errorf("Source = %v, want None", w.res.Source)
	}
	return nil
}

func (w *authWorld) thenNoToken() error {
	if w.res.Token != "" {
		return fmt.Errorf("a token was fabricated: %q", w.res.Token)
	}
	return nil
}

func (w *authWorld) thenNoError() error {
	if w.err != nil {
		return fmt.Errorf("resolution raised an error of its own: %v", w.err)
	}
	return nil
}

func (w *authWorld) thenReadErrorNaming() error {
	var re *ReadError
	if !errors.As(w.err, &re) {
		return fmt.Errorf("expected a *ReadError, got %T: %v", w.err, w.err)
	}
	if re.Path != w.currPath {
		return fmt.Errorf("ReadError.Path = %q, want %q", re.Path, w.currPath)
	}
	return nil
}

func (w *authWorld) thenNoFallThrough() error {
	if w.err == nil {
		return errors.New("expected an error, but resolution fell through to another source")
	}
	if w.res.Token != "" || w.res.Source != SourceNone {
		return fmt.Errorf("resolution returned a usable result %+v despite the read failure", w.res)
	}
	return nil
}

func (w *authWorld) thenFormatErrorNaming() error {
	var fe *FormatError
	if !errors.As(w.err, &fe) {
		return fmt.Errorf("expected a *FormatError, got %T: %v", w.err, w.err)
	}
	if fe.Path != w.currPath {
		return fmt.Errorf("FormatError.Path = %q, want %q", fe.Path, w.currPath)
	}
	return nil
}

func (w *authWorld) thenNotNone() error {
	if w.err == nil && w.res.Source == SourceNone {
		return errors.New("a broken file was reported as absence (None, no error)")
	}
	return nil
}

func (w *authWorld) thenTokenFromEnv() error {
	if w.err != nil {
		return fmt.Errorf("unexpected error: %v", w.err)
	}
	if w.res.Token != os.Getenv(envTokenVar) {
		return fmt.Errorf("Token = %q, want the GLASSFROG_TOKEN value %q", w.res.Token, os.Getenv(envTokenVar))
	}
	return nil
}

func (w *authWorld) thenSourceEnvironment() error {
	if w.res.Source != SourceEnvironment {
		return fmt.Errorf("Source = %v, want Environment", w.res.Source)
	}
	return nil
}

func (w *authWorld) thenNoFileRead() error {
	// On an environment hit no file is consulted, so Path is empty.
	if w.res.Source != SourceEnvironment || w.res.Path != "" {
		return fmt.Errorf("got %+v, want an Environment source with no file path", w.res)
	}
	return nil
}

func (w *authWorld) thenNotEnvSource() error {
	if w.res.Source == SourceEnvironment {
		return errors.New("an empty GLASSFROG_TOKEN was treated as a token")
	}
	return nil
}

func (w *authWorld) thenTokenFromCurrentFile() error {
	if w.err != nil {
		return fmt.Errorf("unexpected error: %v", w.err)
	}
	if w.res.Source != SourceFile || w.res.Path != w.currPath {
		return fmt.Errorf("got %+v, want a File source at the current-directory file %s", w.res, w.currPath)
	}
	return nil
}

func (w *authWorld) thenTokenFromAncestor() error {
	if w.err != nil {
		return fmt.Errorf("unexpected error: %v", w.err)
	}
	if w.res.Source != SourceFile || w.res.Path != w.ancestorPath {
		return fmt.Errorf("got %+v, want a File source at the ancestor file %s", w.res, w.ancestorPath)
	}
	return nil
}

func (w *authWorld) thenFileSourceWithHomePath() error {
	if w.err != nil {
		return fmt.Errorf("unexpected error: %v", w.err)
	}
	if w.res.Source != SourceFile || w.res.Path != w.homePath {
		return fmt.Errorf("got %+v, want a File source with the home file's path %s", w.res, w.homePath)
	}
	return nil
}
