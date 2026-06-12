package apiclient

import (
	"errors"
	"testing"

	"github.com/Luscii/cli-glassfrog/internal/auth"
	"github.com/Luscii/cli-glassfrog/internal/rcfile"
)

// fakeBaseURL / fakeCred build single-outcome resolver funcs for the table tests.
func fakeBaseURL(b BaseURL, err error) func() (BaseURL, error) {
	return func() (BaseURL, error) { return b, err }
}

func fakeCred(r auth.Resolution, err error) func() (auth.Resolution, error) {
	return func() (auth.Resolution, error) { return r, err }
}

func TestAssemble_PacksEachOutcomeCombination(t *testing.T) {
	usableBaseURL := BaseURL{Value: "https://glassfrog.com/api/v5", Source: SourceFile, Path: "/home/me/.glassfrogrc"}
	baseURLErr := &BaseURLError{Source: "--" + FlagBaseURL}
	presentCred := auth.Resolution{Token: "gf_x", Source: auth.SourceEnvironment}
	absentCred := auth.Resolution{Source: auth.SourceNone}
	credErr := &rcfile.ReadError{Path: "/etc/.glassfrogrc", Err: errors.New("permission denied")}

	tests := []struct {
		name         string
		baseURL      BaseURL
		baseURLErr   error
		cred         auth.Resolution
		credErr      error
		wantComplete bool
		wantProblems int
	}{
		{
			name:         "complete",
			baseURL:      usableBaseURL,
			cred:         presentCred,
			wantComplete: true,
			wantProblems: 0,
		},
		{
			name:         "credential absent",
			baseURL:      BaseURL{Value: DefaultBaseURL, Source: SourceDefault},
			cred:         absentCred,
			wantComplete: false,
			wantProblems: 1,
		},
		{
			name:         "base-URL error",
			baseURLErr:   baseURLErr,
			cred:         presentCred,
			wantComplete: false,
			wantProblems: 1,
		},
		{
			name:         "credential error",
			baseURL:      usableBaseURL,
			credErr:      credErr,
			wantComplete: false,
			wantProblems: 1,
		},
		{
			name:         "both problems",
			baseURLErr:   baseURLErr,
			cred:         absentCred,
			wantComplete: false,
			wantProblems: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := Assemble(
				fakeBaseURL(tt.baseURL, tt.baseURLErr),
				fakeCred(tt.cred, tt.credErr),
			)

			// Each (value, error) outcome is packed verbatim into the matching field.
			if ctx.BaseURL != tt.baseURL {
				t.Errorf("BaseURL = %+v, want %+v", ctx.BaseURL, tt.baseURL)
			}
			if !errors.Is(ctx.BaseURLErr, tt.baseURLErr) {
				t.Errorf("BaseURLErr = %v, want %v", ctx.BaseURLErr, tt.baseURLErr)
			}
			if ctx.Cred != tt.cred {
				t.Errorf("Cred = %+v, want %+v", ctx.Cred, tt.cred)
			}
			if !errors.Is(ctx.CredErr, tt.credErr) {
				t.Errorf("CredErr = %v, want %v", ctx.CredErr, tt.credErr)
			}
			if got := ctx.Complete(); got != tt.wantComplete {
				t.Errorf("Complete() = %v, want %v", got, tt.wantComplete)
			}
			if got := len(ctx.Problems()); got != tt.wantProblems {
				t.Errorf("len(Problems()) = %d, want %d (%v)", got, tt.wantProblems, ctx.Problems())
			}
		})
	}
}

// TestAssemble_CallsBothResolversEvenWhenBaseURLErrors is the carry-both tripwire:
// a base-URL resolver error must NOT short-circuit the credential walk. The
// negative property is pinned by recording invocation, not just inspecting output.
func TestAssemble_CallsBothResolversEvenWhenBaseURLErrors(t *testing.T) {
	var baseURLCalls, credCalls int
	resolveBaseURL := func() (BaseURL, error) {
		baseURLCalls++
		return BaseURL{}, &BaseURLError{Source: "--" + FlagBaseURL}
	}
	resolveCred := func() (auth.Resolution, error) {
		credCalls++
		return auth.Resolution{Source: auth.SourceNone}, nil
	}

	ctx := Assemble(resolveBaseURL, resolveCred)

	if baseURLCalls != 1 {
		t.Errorf("base-URL resolver called %d times, want exactly 1", baseURLCalls)
	}
	if credCalls != 1 {
		t.Errorf("credential resolver called %d times, want exactly 1 (carry-both must not short-circuit)", credCalls)
	}
	// Both problems surfaced — proof the credential outcome was carried, not dropped.
	if len(ctx.Problems()) != 2 {
		t.Errorf("Problems() = %v, want both the base-URL and credential parts", ctx.Problems())
	}
}

func TestAssemble_Deterministic(t *testing.T) {
	baseURL := BaseURL{Value: "https://glassfrog.com/api/v5", Source: SourceFile, Path: "/home/me/.glassfrogrc"}
	cred := auth.Resolution{Token: "gf_x", Source: auth.SourceFile, Path: "/home/me/.glassfrogrc"}

	first := Assemble(fakeBaseURL(baseURL, nil), fakeCred(cred, nil))
	second := Assemble(fakeBaseURL(baseURL, nil), fakeCred(cred, nil))

	if first != second {
		t.Errorf("Assemble not deterministic: %+v vs %+v", first, second)
	}
	if first.Complete() != second.Complete() {
		t.Errorf("readiness differs: %v vs %v", first.Complete(), second.Complete())
	}
}

func TestAssemble_NilResolverPanics(t *testing.T) {
	okBaseURL := fakeBaseURL(BaseURL{Value: DefaultBaseURL, Source: SourceDefault}, nil)
	okCred := fakeCred(auth.Resolution{Source: auth.SourceNone}, nil)

	t.Run("nil base-URL resolver", func(t *testing.T) {
		defer func() {
			if recover() == nil {
				t.Error("expected a panic on a nil base-URL resolver (fail-fast), got none")
			}
		}()
		Assemble(nil, okCred)
	})

	t.Run("nil credential resolver", func(t *testing.T) {
		defer func() {
			if recover() == nil {
				t.Error("expected a panic on a nil credential resolver (fail-fast), got none")
			}
		}()
		Assemble(okBaseURL, nil)
	})
}

// TestAssembleFromOS_BindsResolvers confirms the production seam binds
// ResolveBaseURLFromOS(flagValue, flagPresent) and auth.Resolve and delegates to Assemble. It
// is driven hermetically: apiclient's OS seams are stubbed to a config-free temp
// tree (→ built-in default base URL), and GLASSFROG_TOKEN is set so auth.Resolve
// short-circuits at the environment rung without touching the filesystem.
func TestAssembleFromOS_BindsResolvers(t *testing.T) {
	t.Setenv("GLASSFROG_TOKEN", "gf_fromos_test")

	origGetenv, origWd, origHome := getenv, getwd, userHomeDir
	t.Cleanup(func() { getenv, getwd, userHomeDir = origGetenv, origWd, origHome })
	getenv = func(string) string { return "" } // no GLASSFROG_BASE_URL → fall through to default
	emptyDir := t.TempDir()
	homeDir := t.TempDir()
	getwd = func() (string, error) { return emptyDir, nil }
	userHomeDir = func() (string, error) { return homeDir, nil }

	ctx := AssembleFromOS("", false)

	if ctx.BaseURLErr != nil {
		t.Fatalf("unexpected base-URL error: %v", ctx.BaseURLErr)
	}
	if ctx.BaseURL.Source != SourceDefault {
		t.Errorf("BaseURL.Source = %v, want the built-in default (ResolveBaseURLFromOS bound)", ctx.BaseURL.Source)
	}
	if ctx.CredErr != nil {
		t.Fatalf("unexpected credential error: %v", ctx.CredErr)
	}
	if ctx.Cred.Source != auth.SourceEnvironment || ctx.Cred.Token != "gf_fromos_test" {
		t.Errorf("Cred = %s, want the GLASSFROG_TOKEN value from the environment (auth.Resolve bound)", ctx.Cred)
	}
	if !ctx.Complete() {
		t.Errorf("Complete() = false, want true for default base URL + present env token; problems: %v", ctx.Problems())
	}
}
