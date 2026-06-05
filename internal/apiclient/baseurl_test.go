package apiclient

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Luscii/cli-glassfrog/internal/auth"
)

// stubBaseURLEnv overrides the env seam so resolution sees a controlled
// GLASSFROG_BASE_URL without touching the real process environment. An empty val
// models both an empty and an unset variable.
func stubBaseURLEnv(t *testing.T, val string) {
	t.Helper()
	orig := getenv
	t.Cleanup(func() { getenv = orig })
	getenv = func(key string) string {
		if key == EnvVarBaseURL {
			return val
		}
		return ""
	}
}

// seedBaseURLFile writes a .glassfrogrc holding base_url=val into dir (created if
// needed) and returns the file path. Confined to the caller's temp tree.
func seedBaseURLFile(t *testing.T, dir, val string) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}
	path := filepath.Join(dir, auth.CredentialsFileName)
	if err := os.WriteFile(path, []byte("base_url="+val+"\n"), 0o600); err != nil {
		t.Fatalf("seeding %s: %v", path, err)
	}
	return path
}

// tripwireDir places an unreadable directory at the .glassfrogrc path in dir, so
// any unintended file read errors and a "no file was read" guarantee becomes
// load-bearing rather than implied by output (LEARNINGS).
func tripwireDir(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(dir, auth.CredentialsFileName), 0o755); err != nil {
		t.Fatal(err)
	}
}

func TestResolveBaseURL_FlagWinsAndConsultsNoOtherSource(t *testing.T) {
	// A malformed env value and an unreadable file at the nearest location are
	// tripwires: if resolution consulted either despite a usable flag, it would
	// error. A valid flag must short-circuit both.
	stubBaseURLEnv(t, "not-a-url")
	start := t.TempDir()
	tripwireDir(t, start)

	got, err := ResolveBaseURL("https://staging.example.com/api/v5", start, t.TempDir())
	if err != nil {
		t.Fatalf("a usable flag must not consult any other source, but got: %v", err)
	}
	if got.Source != SourceFlag {
		t.Errorf("Source = %v, want Flag", got.Source)
	}
	if got.Value != "https://staging.example.com/api/v5" {
		t.Errorf("Value = %q, want the flag value", got.Value)
	}
	if got.Path != "" {
		t.Errorf("Path = %q, want empty (flag source carries no path)", got.Path)
	}
}

func TestResolveBaseURL_MalformedFlagFailsLoudNoFallThrough(t *testing.T) {
	// A valid env value sits below the flag: a malformed flag must NOT fall
	// through to it — it fails loud naming the flag.
	stubBaseURLEnv(t, "https://env.example.com/api/v5")

	_, err := ResolveBaseURL("api.glassfrog.com", t.TempDir(), t.TempDir())
	if err == nil {
		t.Fatalf("expected a BaseURLError for a scheme-less flag, got nil")
	}
	var be *BaseURLError
	if !errors.As(err, &be) {
		t.Fatalf("expected *BaseURLError, got %T: %v", err, err)
	}
	if !strings.Contains(be.Source, FlagBaseURL) {
		t.Errorf("error source %q does not name the flag %q", be.Source, FlagBaseURL)
	}
}

func TestResolveBaseURL_EnvWinsAndReadsNoFile(t *testing.T) {
	stubBaseURLEnv(t, "https://env.example.com/api/v5")
	start := t.TempDir()
	// A usable file sits below the env, and a tripwire at the nearest location:
	// the env hit must read no file at all.
	tripwireDir(t, start)

	got, err := ResolveBaseURL("", start, t.TempDir())
	if err != nil {
		t.Fatalf("an env hit must not read any file, but got: %v", err)
	}
	if got.Source != SourceEnvironment {
		t.Errorf("Source = %v, want Environment", got.Source)
	}
	if got.Value != "https://env.example.com/api/v5" {
		t.Errorf("Value = %q, want the env value", got.Value)
	}
	if got.Path != "" {
		t.Errorf("Path = %q, want empty (env source carries no path)", got.Path)
	}
}

func TestResolveBaseURL_MalformedEnvNamesTheVariableNoFallThrough(t *testing.T) {
	stubBaseURLEnv(t, "ftp://glassfrog.com/api/v5")
	start := t.TempDir()
	seedBaseURLFile(t, start, "https://file.example.com/api/v5") // a usable file below

	_, err := ResolveBaseURL("", start, t.TempDir())
	if err == nil {
		t.Fatalf("expected a BaseURLError for a non-http env scheme, got nil")
	}
	var be *BaseURLError
	if !errors.As(err, &be) {
		t.Fatalf("expected *BaseURLError, got %T: %v", err, err)
	}
	if !strings.Contains(be.Source, EnvVarBaseURL) {
		t.Errorf("error source %q does not name %q", be.Source, EnvVarBaseURL)
	}
}

func TestResolveBaseURL_EmptyEnvFallsThroughToFile(t *testing.T) {
	stubBaseURLEnv(t, "") // unset / empty both look like ""
	start := t.TempDir()
	want := seedBaseURLFile(t, start, "https://file.example.com/api/v5")

	got, err := ResolveBaseURL("", start, t.TempDir())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Source != SourceFile || got.Value != "https://file.example.com/api/v5" || got.Path != want {
		t.Errorf("got %+v, want File/https://file.example.com/api/v5/%s", got, want)
	}
}

func TestResolveBaseURL_WhitespaceOnlyEnvFallsThrough(t *testing.T) {
	stubBaseURLEnv(t, "   \t ")
	start := t.TempDir()
	want := seedBaseURLFile(t, start, "https://file.example.com/api/v5")

	got, err := ResolveBaseURL("", start, t.TempDir())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Source != SourceFile || got.Path != want {
		t.Errorf("got %+v, want the file source at %s (whitespace env ignored)", got, want)
	}
}

func TestResolveBaseURL_WhitespaceOnlyFlagFallsThrough(t *testing.T) {
	stubBaseURLEnv(t, "https://env.example.com/api/v5")

	got, err := ResolveBaseURL("   ", t.TempDir(), t.TempDir())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Source != SourceEnvironment {
		t.Errorf("Source = %v, want Environment (whitespace flag ignored)", got.Source)
	}
}

func TestResolveBaseURL_NearestFileWinsOverHome(t *testing.T) {
	stubBaseURLEnv(t, "")
	start := t.TempDir()
	home := t.TempDir()
	want := seedBaseURLFile(t, start, "https://project.example.com/api/v5")
	seedBaseURLFile(t, home, "https://home.example.com/api/v5")

	got, err := ResolveBaseURL("", start, home)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Source != SourceFile || got.Value != "https://project.example.com/api/v5" || got.Path != want {
		t.Errorf("got %+v, want the project file %s", got, want)
	}
}

func TestResolveBaseURL_BaseURLlessFileFallsThroughToDefault(t *testing.T) {
	stubBaseURLEnv(t, "")
	start := t.TempDir()
	// A token-only file holds no base_url: the walk continues and, with nothing
	// else, the default backstops.
	if err := os.WriteFile(filepath.Join(start, auth.CredentialsFileName), []byte("token=gf_only\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	got, err := ResolveBaseURL("", start, t.TempDir())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Source != SourceDefault || got.Value != DefaultBaseURL {
		t.Errorf("got %+v, want the built-in default %q", got, DefaultBaseURL)
	}
}

func TestResolveBaseURL_DefaultBackstopWhenNothingConfigured(t *testing.T) {
	stubBaseURLEnv(t, "")
	start := filepath.Join(t.TempDir(), "deep", "dir")
	if err := os.MkdirAll(start, 0o755); err != nil {
		t.Fatal(err)
	}

	got, err := ResolveBaseURL("", start, t.TempDir())
	if err != nil {
		t.Fatalf("the default must not be an error, got: %v", err)
	}
	if got.Source != SourceDefault {
		t.Errorf("Source = %v, want Default", got.Source)
	}
	if got.Value != DefaultBaseURL {
		t.Errorf("Value = %q, want the pinned default %q", got.Value, DefaultBaseURL)
	}
	if got.Path != "" {
		t.Errorf("Path = %q, want empty for the default source", got.Path)
	}
}

func TestResolveBaseURL_MalformedFileNamesFileNoFallThrough(t *testing.T) {
	stubBaseURLEnv(t, "")
	start := t.TempDir()
	want := seedBaseURLFile(t, start, "api.glassfrog.com") // scheme-less

	_, err := ResolveBaseURL("", start, t.TempDir())
	if err == nil {
		t.Fatalf("expected a BaseURLError for a scheme-less file value, got nil")
	}
	var be *BaseURLError
	if !errors.As(err, &be) {
		t.Fatalf("expected *BaseURLError, got %T: %v", err, err)
	}
	if be.Source != want {
		t.Errorf("error source = %q, want the file path %q", be.Source, want)
	}
}

func TestResolveBaseURL_UnreadableFileFailsLoudNoDefault(t *testing.T) {
	stubBaseURLEnv(t, "")
	start := t.TempDir()
	tripwireDir(t, start) // a directory at .glassfrogrc → an unreadable file

	_, err := ResolveBaseURL("", start, t.TempDir())
	if err == nil {
		t.Fatalf("expected a read error, got nil (must not fall through to the default)")
	}
	var re *auth.ReadError
	if !errors.As(err, &re) {
		t.Fatalf("expected *auth.ReadError, got %T: %v", err, err)
	}
}

func TestResolveBaseURL_ValuePassedThroughVerbatim(t *testing.T) {
	stubBaseURLEnv(t, "")
	start := t.TempDir()
	// A trailing slash must survive verbatim — the resolver never normalizes.
	seedBaseURLFile(t, start, "https://glassfrog.com/api/v5/")

	got, err := ResolveBaseURL("", start, t.TempDir())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Value != "https://glassfrog.com/api/v5/" {
		t.Errorf("Value = %q, want the trailing slash preserved verbatim", got.Value)
	}
}

func TestResolveBaseURL_Deterministic(t *testing.T) {
	stubBaseURLEnv(t, "")
	start := t.TempDir()
	seedBaseURLFile(t, start, "https://file.example.com/api/v5")
	home := t.TempDir()

	first, err1 := ResolveBaseURL("", start, home)
	second, err2 := ResolveBaseURL("", start, home)
	if err1 != nil || err2 != nil {
		t.Fatalf("unexpected error: %v / %v", err1, err2)
	}
	if first != second {
		t.Errorf("resolution not deterministic: %+v vs %+v", first, second)
	}
}

func TestResolveBaseURL_BaseURLErrorCarriesNoSecret(t *testing.T) {
	// Even when the malformed file also holds a token, the error names only the
	// source/path and never the token (secret hygiene; ADR-3/ADR-4).
	stubBaseURLEnv(t, "")
	secret := "gf_super_secret_token"
	start := t.TempDir()
	path := filepath.Join(start, auth.CredentialsFileName)
	content := "token=" + secret + "\nbase_url=api.glassfrog.com\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := ResolveBaseURL("", start, t.TempDir())
	if err == nil {
		t.Fatalf("expected a BaseURLError, got nil")
	}
	if strings.Contains(err.Error(), secret) {
		t.Errorf("the error leaked the token value: %q", err.Error())
	}
}

func TestIsUsableURL_AcceptRejectBoundary(t *testing.T) {
	cases := []struct {
		value string
		want  bool
	}{
		{"https://glassfrog.com/api/v5", true},
		{"http://localhost:8080", true},
		{"https://glassfrog.com", true},
		{"api.glassfrog.com", false},          // scheme-less host
		{"ftp://glassfrog.com/api/v5", false}, // non-http(s) scheme
		{"https://", false},                   // scheme but no host
		{"://glassfrog.com", false},           // no scheme
		{"glassfrog", false},                  // bare word
		{"http://", false},                    // no host
	}
	for _, c := range cases {
		if got := isUsableURL(c.value); got != c.want {
			t.Errorf("isUsableURL(%q) = %v, want %v", c.value, got, c.want)
		}
	}
}

func TestBaseURLSource_String(t *testing.T) {
	cases := map[BaseURLSource]string{
		SourceFlag:        "flag",
		SourceEnvironment: "environment",
		SourceFile:        "file",
		SourceDefault:     "default",
	}
	for src, want := range cases {
		if got := src.String(); got != want {
			t.Errorf("%d.String() = %q, want %q", src, got, want)
		}
	}
}

func TestResolveBaseURLFromOS_BindsSeams(t *testing.T) {
	// The production seam binds os.Getwd/os.UserHomeDir/os.Getenv. Stub all three
	// and confirm the wrapper feeds them into the pure resolver.
	stubBaseURLEnv(t, "")
	start := t.TempDir()
	want := seedBaseURLFile(t, start, "https://file.example.com/api/v5")
	home := t.TempDir()

	origWd, origHome := getwd, userHomeDir
	t.Cleanup(func() { getwd, userHomeDir = origWd, origHome })
	getwd = func() (string, error) { return start, nil }
	userHomeDir = func() (string, error) { return home, nil }

	got, err := ResolveBaseURLFromOS("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Source != SourceFile || got.Path != want {
		t.Errorf("got %+v, want the file source at %s", got, want)
	}
}
