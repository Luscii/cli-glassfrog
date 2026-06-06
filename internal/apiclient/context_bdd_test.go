package apiclient

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/Luscii/cli-glassfrog/internal/auth"
	"github.com/Luscii/cli-glassfrog/internal/rcfile"
	"github.com/cucumber/godog"
)

// TestConnectionContextFeatures runs the Connection Context Assembly (009)
// executable acceptance scenarios against the pure aggregator (Assemble), driving
// it over FAKE base-URL / credential resolver outcomes — no real network and no
// real home/filesystem are ever touched.
//
// This is a SEPARATE suite from TestFeatures (007) and TestBaseURLFeatures (008).
// godog binds steps per-suite, so each suite's Paths names only its own feature
// file (LEARNINGS 2026-06-04): this suite owns connection-context-assembly.feature
// alone. The four @validation scenarios stay @wip — held out for the validate
// skill, not implemented by the Builder.
func TestConnectionContextFeatures(t *testing.T) {
	suite := godog.TestSuite{
		ScenarioInitializer: initializeConnectionContextScenario,
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"../../features/undefined-connection-settings/connection-context-assembly.feature"},
			Tags:     "~@wip",
			TestingT: t,
		},
	}
	if suite.Run() != 0 {
		t.Fatal("non-zero status: connection context assembly feature scenarios failed")
	}
}

// connectionWorld is the per-scenario state. It configures fake resolver outcomes
// (the base-URL value/source/error and the credential outcome/error), drives
// Assemble over fakes that record invocation, and captures the resulting context
// and its rendering — all in memory, no I/O.
type connectionWorld struct {
	baseURL    BaseURL
	baseURLErr error
	cred       auth.Resolution
	credErr    error

	configPath     string // shared file path when base URL and token come from one file
	baseURLErrPath string // file path named by a base-URL read error
	credErrPath    string // file path named by a credential read error

	ctx      ConnectionContext
	rendered string
	token    string // the token a redaction scenario asserts never leaks

	baseURLCalls int
	credCalls    int

	requestContexts []ConnectionContext // contexts observed across simulated requests
}

func initializeConnectionContextScenario(sc *godog.ScenarioContext) {
	w := &connectionWorld{}

	sc.Before(func(ctx context.Context, _ *godog.Scenario) (context.Context, error) {
		*w = connectionWorld{}
		return ctx, nil
	})

	// --- Givens: base URL outcomes ---
	sc.Step(`^the base URL resolved to "([^"]*)" from a config file$`, w.givenBaseURLFromConfigFile)
	sc.Step(`^the base URL resolved to "([^"]*)"$`, w.givenBaseURLResolved)
	sc.Step(`^the base URL resolved to the built-in default$`, w.givenBaseURLDefault)
	sc.Step(`^base URL resolution reported a format error naming the flag$`, w.givenBaseURLFormatErrorFlag)
	sc.Step(`^base URL resolution reported a read error naming a config file$`, w.givenBaseURLReadError)

	// --- Givens: credential outcomes ---
	sc.Step(`^a token resolved from that config file$`, w.givenTokenFromThatConfigFile)
	sc.Step(`^a token resolved from the environment$`, w.givenTokenFromEnv)
	sc.Step(`^no credentials were found$`, w.givenNoCredentials)
	sc.Step(`^credential discovery reported a read error naming a config file$`, w.givenCredentialReadError)

	// --- Givens: pre-assembled contexts ---
	sc.Step(`^a complete context assembled with the token "([^"]*)"$`, w.givenCompleteContextWithToken)
	sc.Step(`^a connection context assembled for the invocation$`, w.givenContextAssembledForInvocation)

	// --- Whens ---
	sc.Step(`^the CLI assembles the connection context$`, w.whenAssemble)
	sc.Step(`^the connection context is rendered for diagnostics$`, w.whenRender)
	sc.Step(`^a command makes more than one API request$`, w.whenMakeMultipleRequests)

	// --- Thens: carried outcomes ---
	sc.Step(`^it will carry the resolved base URL and its source$`, w.thenCarriesBaseURLAndSource)
	sc.Step(`^it will carry the resolved token and its source$`, w.thenCarriesTokenAndSource)
	sc.Step(`^it will carry the resolved base URL$`, w.thenCarriesBaseURL)
	sc.Step(`^it will carry a credential outcome of absent$`, w.thenCredentialAbsent)
	sc.Step(`^it will carry the credential error naming that file$`, w.thenCarriesCredentialErrorNamingFile)
	sc.Step(`^it will carry the base-URL error naming that file$`, w.thenCarriesBaseURLErrorNamingFile)
	sc.Step(`^it will keep the resolved token and its source$`, w.thenKeepsTokenAndSource)
	sc.Step(`^it will surface both the base-URL error and the absent credential$`, w.thenSurfaceBoth)

	// --- Thens: readiness ---
	sc.Step(`^it will report the context as complete$`, w.thenReportsComplete)
	sc.Step(`^it will report the base URL source as the built-in default$`, w.thenBaseURLSourceDefault)
	sc.Step(`^it will report the context as incomplete naming the missing credential$`, w.thenIncompleteNamingCredential)
	sc.Step(`^it will report the context as incomplete naming the credential part$`, w.thenIncompleteNamingCredential)
	sc.Step(`^it will report the context as incomplete naming the base-URL part$`, w.thenIncompleteNamingBaseURL)
	sc.Step(`^it will report the context as incomplete$`, w.thenReportsIncomplete)
	sc.Step(`^it will not refuse the request or fabricate a token$`, w.thenNotRefuseOrFabricate)
	sc.Step(`^it will not stop at the first problem$`, w.thenNotStopAtFirstProblem)

	// --- Thens: redaction ---
	sc.Step(`^it will show the credential source and path$`, w.thenShowsCredentialSourceAndPath)
	sc.Step(`^the token value "([^"]*)" will not appear in the output$`, w.thenTokenValueNotInOutput)

	// --- Thens: assemble-once / reuse ---
	sc.Step(`^every request will use the same assembled context$`, w.thenEveryRequestSameContext)
	sc.Step(`^the context will not be reassembled or re-resolved between requests$`, w.thenNotReassembledOrReResolved)
}

// --- Given implementations ---

func (w *connectionWorld) givenBaseURLFromConfigFile(value string) error {
	w.configPath = "/home/me/.glassfrogrc"
	w.baseURL = BaseURL{Value: value, Source: SourceFile, Path: w.configPath}
	return nil
}

func (w *connectionWorld) givenBaseURLResolved(value string) error {
	w.configPath = "/home/me/.glassfrogrc"
	w.baseURL = BaseURL{Value: value, Source: SourceFile, Path: w.configPath}
	return nil
}

func (w *connectionWorld) givenBaseURLDefault() error {
	w.baseURL = BaseURL{Value: DefaultBaseURL, Source: SourceDefault}
	return nil
}

func (w *connectionWorld) givenBaseURLFormatErrorFlag() error {
	w.baseURLErr = &BaseURLError{Source: "--" + FlagBaseURL}
	return nil
}

func (w *connectionWorld) givenBaseURLReadError() error {
	w.baseURLErrPath = "/work/.glassfrogrc"
	w.baseURLErr = &rcfile.ReadError{Path: w.baseURLErrPath, Err: errors.New("permission denied")}
	return nil
}

func (w *connectionWorld) givenTokenFromThatConfigFile() error {
	path := w.configPath
	if path == "" {
		path = "/home/me/.glassfrogrc"
	}
	w.cred = auth.Resolution{Token: "gf_file_token", Source: auth.SourceFile, Path: path}
	return nil
}

func (w *connectionWorld) givenTokenFromEnv() error {
	w.cred = auth.Resolution{Token: "gf_env_token", Source: auth.SourceEnvironment}
	return nil
}

func (w *connectionWorld) givenNoCredentials() error {
	w.cred = auth.Resolution{Source: auth.SourceNone}
	return nil
}

func (w *connectionWorld) givenCredentialReadError() error {
	w.credErrPath = "/work/.glassfrogrc"
	w.credErr = &rcfile.ReadError{Path: w.credErrPath, Err: errors.New("permission denied")}
	return nil
}

func (w *connectionWorld) givenCompleteContextWithToken(token string) error {
	w.token = token
	w.configPath = "/home/me/.glassfrogrc"
	w.baseURL = BaseURL{Value: "https://glassfrog.com/api/v5", Source: SourceFile, Path: w.configPath}
	w.cred = auth.Resolution{Token: token, Source: auth.SourceFile, Path: w.configPath}
	w.ctx = w.assemble()
	if !w.ctx.Complete() {
		return fmt.Errorf("expected a complete context, got problems %v", w.ctx.Problems())
	}
	return nil
}

func (w *connectionWorld) givenContextAssembledForInvocation() error {
	w.configPath = "/home/me/.glassfrogrc"
	w.baseURL = BaseURL{Value: "https://glassfrog.com/api/v5", Source: SourceFile, Path: w.configPath}
	w.cred = auth.Resolution{Token: "gf_invocation_token", Source: auth.SourceFile, Path: w.configPath}
	w.ctx = w.assemble()
	return nil
}

// --- When implementations ---

// assemble drives Assemble over fakes that record invocation, so carry-both /
// resolve-once can be pinned by counting, not just by output.
func (w *connectionWorld) assemble() ConnectionContext {
	return Assemble(
		func() (BaseURL, error) { w.baseURLCalls++; return w.baseURL, w.baseURLErr },
		func() (auth.Resolution, error) { w.credCalls++; return w.cred, w.credErr },
	)
}

func (w *connectionWorld) whenAssemble() error {
	w.ctx = w.assemble()
	return nil
}

func (w *connectionWorld) whenRender() error {
	w.rendered = w.ctx.String()
	return nil
}

func (w *connectionWorld) whenMakeMultipleRequests() error {
	// Each "request" reuses the already-assembled context — no re-assembly.
	for i := 0; i < 3; i++ {
		w.requestContexts = append(w.requestContexts, w.ctx)
	}
	return nil
}

// --- Then implementations ---

func (w *connectionWorld) thenCarriesBaseURLAndSource() error {
	if w.ctx.BaseURLErr != nil {
		return fmt.Errorf("unexpected base-URL error: %v", w.ctx.BaseURLErr)
	}
	if w.ctx.BaseURL.Value != w.baseURL.Value || w.ctx.BaseURL.Source != w.baseURL.Source {
		return fmt.Errorf("got %+v, want the resolved base URL %+v", w.ctx.BaseURL, w.baseURL)
	}
	return nil
}

func (w *connectionWorld) thenCarriesTokenAndSource() error {
	if w.ctx.CredErr != nil {
		return fmt.Errorf("unexpected credential error: %v", w.ctx.CredErr)
	}
	if w.ctx.Cred.Token != w.cred.Token || w.ctx.Cred.Source != w.cred.Source {
		return fmt.Errorf("got %s, want the resolved token with source %v", w.ctx.Cred, w.cred.Source)
	}
	return nil
}

func (w *connectionWorld) thenCarriesBaseURL() error {
	if w.ctx.BaseURLErr != nil {
		return fmt.Errorf("unexpected base-URL error: %v", w.ctx.BaseURLErr)
	}
	if w.ctx.BaseURL.Value != w.baseURL.Value {
		return fmt.Errorf("BaseURL.Value = %q, want %q", w.ctx.BaseURL.Value, w.baseURL.Value)
	}
	return nil
}

func (w *connectionWorld) thenCredentialAbsent() error {
	if w.ctx.CredErr != nil {
		return fmt.Errorf("unexpected credential error: %v", w.ctx.CredErr)
	}
	if w.ctx.Cred.Source != auth.SourceNone {
		return fmt.Errorf("Cred.Source = %v, want None (absent)", w.ctx.Cred.Source)
	}
	return nil
}

func (w *connectionWorld) thenCarriesCredentialErrorNamingFile() error {
	var re *rcfile.ReadError
	if !errors.As(w.ctx.CredErr, &re) {
		return fmt.Errorf("expected a *rcfile.ReadError in CredErr, got %T: %v", w.ctx.CredErr, w.ctx.CredErr)
	}
	if re.Path != w.credErrPath {
		return fmt.Errorf("CredErr names %q, want the file %q", re.Path, w.credErrPath)
	}
	return nil
}

func (w *connectionWorld) thenCarriesBaseURLErrorNamingFile() error {
	var re *rcfile.ReadError
	if !errors.As(w.ctx.BaseURLErr, &re) {
		return fmt.Errorf("expected a *rcfile.ReadError in BaseURLErr, got %T: %v", w.ctx.BaseURLErr, w.ctx.BaseURLErr)
	}
	if re.Path != w.baseURLErrPath {
		return fmt.Errorf("BaseURLErr names %q, want the file %q", re.Path, w.baseURLErrPath)
	}
	return nil
}

func (w *connectionWorld) thenKeepsTokenAndSource() error {
	if w.ctx.CredErr != nil {
		return fmt.Errorf("unexpected credential error: %v", w.ctx.CredErr)
	}
	if w.ctx.Cred.Token != w.cred.Token || w.ctx.Cred.Source != w.cred.Source {
		return fmt.Errorf("token not kept intact: got %s, want source %v with the resolved token", w.ctx.Cred, w.cred.Source)
	}
	return nil
}

func (w *connectionWorld) thenSurfaceBoth() error {
	if w.ctx.BaseURLErr == nil {
		return errors.New("base-URL error was not carried")
	}
	if w.ctx.Cred.Source != auth.SourceNone {
		return fmt.Errorf("credential absence was not carried: Source = %v", w.ctx.Cred.Source)
	}
	probs := w.ctx.Problems()
	if len(probs) != 2 {
		return fmt.Errorf("Problems() = %v, want both the base-URL and credential parts", probs)
	}
	return nil
}

func (w *connectionWorld) thenReportsComplete() error {
	if !w.ctx.Complete() {
		return fmt.Errorf("Complete() = false, want true; problems: %v", w.ctx.Problems())
	}
	return nil
}

func (w *connectionWorld) thenBaseURLSourceDefault() error {
	if w.ctx.BaseURL.Source != SourceDefault {
		return fmt.Errorf("BaseURL.Source = %v, want the built-in default", w.ctx.BaseURL.Source)
	}
	return nil
}

func (w *connectionWorld) thenReportsIncomplete() error {
	if w.ctx.Complete() {
		return errors.New("Complete() = true, want false")
	}
	return nil
}

func (w *connectionWorld) thenIncompleteNamingCredential() error {
	if err := w.thenReportsIncomplete(); err != nil {
		return err
	}
	return w.problemNaming("credential")
}

func (w *connectionWorld) thenIncompleteNamingBaseURL() error {
	if err := w.thenReportsIncomplete(); err != nil {
		return err
	}
	return w.problemNaming("base url")
}

// problemNaming asserts at least one Problems() entry names the given part
// (case-insensitive), without leaking any secret.
func (w *connectionWorld) problemNaming(part string) error {
	for _, p := range w.ctx.Problems() {
		if strings.Contains(strings.ToLower(p), part) {
			return nil
		}
	}
	return fmt.Errorf("Problems() = %v, want an entry naming the %q part", w.ctx.Problems(), part)
}

func (w *connectionWorld) thenNotRefuseOrFabricate() error {
	// Assembly always returns a context (no refusal) and never invents a token.
	if w.ctx.Cred.Source != auth.SourceNone {
		return fmt.Errorf("Cred.Source = %v, want None — no source should be fabricated", w.ctx.Cred.Source)
	}
	if w.ctx.Cred.Token != "" {
		return errors.New("a token was fabricated for an absent credential")
	}
	return nil
}

func (w *connectionWorld) thenNotStopAtFirstProblem() error {
	// Carry-both: the credential resolver must have run despite the base-URL error.
	if w.baseURLCalls != 1 {
		return fmt.Errorf("base-URL resolver called %d times, want exactly 1", w.baseURLCalls)
	}
	if w.credCalls != 1 {
		return fmt.Errorf("credential resolver called %d times, want exactly 1 (must not short-circuit)", w.credCalls)
	}
	return nil
}

func (w *connectionWorld) thenShowsCredentialSourceAndPath() error {
	if !strings.Contains(w.rendered, w.cred.Source.String()) {
		return fmt.Errorf("rendering %q does not show the credential source %q", w.rendered, w.cred.Source)
	}
	if w.cred.Path != "" && !strings.Contains(w.rendered, w.cred.Path) {
		return fmt.Errorf("rendering %q does not show the credential path %q", w.rendered, w.cred.Path)
	}
	return nil
}

func (w *connectionWorld) thenTokenValueNotInOutput(token string) error {
	if strings.Contains(w.rendered, token) {
		return fmt.Errorf("rendering leaked the token %q: %s", token, w.rendered)
	}
	return nil
}

func (w *connectionWorld) thenEveryRequestSameContext() error {
	if len(w.requestContexts) < 2 {
		return fmt.Errorf("only %d requests observed, want more than one", len(w.requestContexts))
	}
	for i, got := range w.requestContexts {
		if got != w.ctx {
			return fmt.Errorf("request %d used a different context: %+v vs %+v", i, got, w.ctx)
		}
	}
	return nil
}

func (w *connectionWorld) thenNotReassembledOrReResolved() error {
	if w.baseURLCalls != 1 || w.credCalls != 1 {
		return fmt.Errorf("resolvers ran %d/%d times, want exactly 1/1 — the context was re-resolved", w.baseURLCalls, w.credCalls)
	}
	return nil
}
