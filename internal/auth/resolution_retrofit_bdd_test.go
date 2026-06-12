package auth

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/cucumber/godog"
)

// TestResolutionRetrofitFeatures runs the token slice of the Resolution
// Call-Site Retrofit (040) executable acceptance scenarios against the
// production seam (Resolve), driving it over temp directory trees and a
// controlled GLASSFROG_TOKEN — never the developer's real home directory or
// working directory.
//
// 040's feature file is cross-cutting (token, base URL, output). godog binds
// steps per-suite and a feature file's scenarios need different package seams
// to stay hermetic (the token walk must rebind auth's own getwd/userHomeDir to
// avoid reading the real ~/.glassfrogrc — CONSTITUTION IV), so each domain owns
// its slice via a domain tag: this suite runs only the @token scenarios. The
// @base-url / @output / @validation scenarios stay out of scope here (filtered
// by "@token && ~@wip").
func TestResolutionRetrofitFeatures(t *testing.T) {
	suite := godog.TestSuite{
		ScenarioInitializer: initializeRetrofitTokenScenario,
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"../../features/duplicated-setting-resolution/resolution-call-site-retrofit.feature"},
			Tags:     "@token && ~@wip",
			TestingT: t,
		},
	}
	if suite.Run() != 0 {
		t.Fatal("non-zero status: token retrofit feature scenarios failed")
	}
}

// retrofitTokenWorld is the per-scenario state. It builds a temp filesystem and
// drives the production seam (Resolve) by binding getwd/userHomeDir to
// scenario-chosen temp directories and setting/unsetting the real
// GLASSFROG_TOKEN.
type retrofitTokenWorld struct {
	base    string // root temp dir for this scenario, removed in After
	homeDir string
	currDir string

	res Resolution
	err error
}

func initializeRetrofitTokenScenario(sc *godog.ScenarioContext) {
	w := &retrofitTokenWorld{}

	// Capture the ambient GLASSFROG_TOKEN and the OS seams once, restoring them
	// after every scenario so the suite is hermetic and order-independent.
	origToken, tokenWasSet := os.LookupEnv(envTokenVar)
	origGetwd, origHome := getwd, userHomeDir

	sc.Before(func(ctx context.Context, _ *godog.Scenario) (context.Context, error) {
		base, err := os.MkdirTemp("", "token-retrofit-bdd-")
		if err != nil {
			return ctx, err
		}
		*w = retrofitTokenWorld{base: base}
		if w.homeDir, err = mkdirAll(filepath.Join(base, "home")); err != nil {
			return ctx, err
		}
		// currDir sits in a sibling subtree, so home is the final fallback (not an
		// ancestor) — matching the precedence-chain semantics the scenarios assert.
		if w.currDir, err = mkdirAll(filepath.Join(base, "work", "nested")); err != nil {
			return ctx, err
		}
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

	// --- Givens ---
	sc.Step(`^GLASSFROG_TOKEN had been set to a non-empty value$`, w.givenTokenEnv)
	sc.Step(`^GLASSFROG_TOKEN was unset and no "\.glassfrogrc" carried a token key$`, w.givenNothingAnywhere)

	// --- When ---
	sc.Step(`^the token is resolved$`, w.whenResolve)

	// --- Thens ---
	sc.Step(`^it will return that token with the source reported as the environment$`, w.thenTokenFromEnv)
	sc.Step(`^it will not read any "\.glassfrogrc"$`, w.thenNoFileConsulted)
	sc.Step(`^it will report the source as none with no error$`, w.thenNoneNoError)
	sc.Step(`^the resolution will carry no token value$`, w.thenNoToken)
}

// retrofitTokenEnvValue is a non-empty, recognisable token the env-walk scenario
// asserts against.
const retrofitTokenEnvValue = "gf_env_token_value"

func (w *retrofitTokenWorld) givenTokenEnv() error {
	// Seed a decoy file-token too: the env hit must short-circuit it (proving the
	// file walk is never consulted when the environment yields).
	if _, err := seedRC(w.currDir, "token=gf_decoy_file_token\n"); err != nil {
		return err
	}
	return os.Setenv(envTokenVar, retrofitTokenEnvValue)
}

func (w *retrofitTokenWorld) givenNothingAnywhere() error {
	// Before unsets the env and seeds no files, so nothing yields anywhere.
	return nil
}

func (w *retrofitTokenWorld) whenResolve() error {
	w.res, w.err = Resolve()
	return nil
}

func (w *retrofitTokenWorld) thenTokenFromEnv() error {
	if w.err != nil {
		return fmt.Errorf("unexpected error: %v", w.err)
	}
	if w.res.Source != SourceEnvironment {
		return fmt.Errorf("Source = %v, want Environment", w.res.Source)
	}
	if w.res.Token != retrofitTokenEnvValue {
		return fmt.Errorf("Token = %q, want the GLASSFROG_TOKEN value %q", w.res.Token, retrofitTokenEnvValue)
	}
	return nil
}

func (w *retrofitTokenWorld) thenNoFileConsulted() error {
	// An environment hit consults no file, so it carries no file path even though
	// a decoy .glassfrogrc was seeded in the current directory.
	if w.res.Path != "" {
		return fmt.Errorf("a .glassfrogrc was consulted: Path = %q, want empty", w.res.Path)
	}
	return nil
}

func (w *retrofitTokenWorld) thenNoneNoError() error {
	if w.err != nil {
		return fmt.Errorf("absence reported as an error: %v", w.err)
	}
	if w.res.Source != SourceNone {
		return fmt.Errorf("Source = %v, want None", w.res.Source)
	}
	return nil
}

func (w *retrofitTokenWorld) thenNoToken() error {
	if w.res.Token != "" {
		return fmt.Errorf("a token was carried despite no source: %q", w.res.Token)
	}
	return nil
}

// mkdirAll and seedRC are file helpers local to this suite (the 040 token slice),
// returning errors rather than panicking so a failure keeps godog on its normal
// failure path and the After hook still restores the seams and removes the tree.
func mkdirAll(dir string) (string, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("mkdir %s: %w", dir, err)
	}
	return dir, nil
}

func seedRC(dir, content string) (string, error) {
	if _, err := mkdirAll(dir); err != nil {
		return "", err
	}
	path := filepath.Join(dir, credentialsFileName)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		return "", fmt.Errorf("seed %s: %w", path, err)
	}
	return path, nil
}
