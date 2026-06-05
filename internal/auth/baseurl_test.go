package auth

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// seedBaseURL writes a .glassfrogrc holding base_url=val into dir (created if
// needed) and returns the file path. Confined to the caller's temp tree.
func seedBaseURL(t *testing.T, dir, val string) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}
	path := filepath.Join(dir, credentialsFileName)
	if err := os.WriteFile(path, []byte("base_url="+val+"\n"), 0o600); err != nil {
		t.Fatalf("seeding %s: %v", path, err)
	}
	return path
}

// seedTokenAndBaseURL writes a .glassfrogrc holding both a token and a base_url
// into dir and returns the file path. Used to prove the token never crosses the
// base-URL read seam.
func seedTokenAndBaseURL(t *testing.T, dir, token, baseURL string) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}
	path := filepath.Join(dir, credentialsFileName)
	content := "token=" + token + "\nbase_url=" + baseURL + "\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("seeding %s: %v", path, err)
	}
	return path
}

func TestReadBaseURLFile_ValidValue(t *testing.T) {
	path := writeFile(t, "# glassfrog config\nbase_url=https://glassfrog.com/api/v5\n")

	value, found, err := readBaseURLFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !found {
		t.Fatalf("found = false, want true for a file holding base_url=…")
	}
	if value != "https://glassfrog.com/api/v5" {
		t.Errorf("value = %q, want %q", value, "https://glassfrog.com/api/v5")
	}
}

func TestReadBaseURLFile_NoBaseURLKey(t *testing.T) {
	// A file that parses cleanly but carries no base_url key is a normal "not
	// found" — no error. A token-only file must read as base-URL-less.
	path := writeFile(t, "token=gf_x\n")

	value, found, err := readBaseURLFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found {
		t.Errorf("found = true, want false for a file with no base_url entry")
	}
	if value != "" {
		t.Errorf("value = %q, want empty", value)
	}
}

func TestReadBaseURLFile_WhitespaceOnlyValueIsNotFound(t *testing.T) {
	path := writeFile(t, "base_url=   \n")

	_, found, err := readBaseURLFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found {
		t.Errorf("found = true, want false for a whitespace-only base_url value")
	}
}

func TestReadBaseURLFile_MalformedLineIsFormatError(t *testing.T) {
	secret := "gf_super_secret"
	path := writeFile(t, "token="+secret+"\nbase_url=https://glassfrog.com\nthis line has no equals sign\n")

	_, _, err := readBaseURLFile(path)
	if err == nil {
		t.Fatalf("expected a format error, got nil")
	}
	var fe *FormatError
	if !errors.As(err, &fe) {
		t.Fatalf("expected *FormatError, got %T: %v", err, err)
	}
	if fe.Path != path {
		t.Errorf("FormatError.Path = %q, want %q", fe.Path, path)
	}
	// Secret hygiene: the token value must never appear in the error text even
	// though the file held one alongside the base_url.
	if strings.Contains(err.Error(), secret) {
		t.Errorf("format error leaked the token value: %q", err.Error())
	}
}

func TestReadBaseURLFile_MissingFileIsReadError(t *testing.T) {
	path := filepath.Join(t.TempDir(), credentialsFileName)

	_, _, err := readBaseURLFile(path)
	if err == nil {
		t.Fatalf("expected a read error for a missing file, got nil")
	}
	var re *ReadError
	if !errors.As(err, &re) {
		t.Fatalf("expected *ReadError, got %T: %v", err, err)
	}
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("missing-file ReadError should unwrap to os.ErrNotExist: %v", err)
	}
	if re.Path != path {
		t.Errorf("ReadError.Path = %q, want %q", re.Path, path)
	}
}

func TestReadBaseURLFile_TokenNeverReturned(t *testing.T) {
	// A file carrying both keys: the base-URL read must surface only the base_url
	// value, never the token (ADR-3 secret hygiene).
	secret := "gf_super_secret_token"
	path := seedTokenAndBaseURL(t, t.TempDir(), secret, "https://glassfrog.com/api/v5")

	value, found, err := readBaseURLFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !found || value != "https://glassfrog.com/api/v5" {
		t.Fatalf("got (%q, %v), want (%q, true)", value, found, "https://glassfrog.com/api/v5")
	}
	if strings.Contains(value, secret) {
		t.Errorf("the token value leaked into the base_url read: %q", value)
	}
}

func TestResolveBaseURLFile_NearestWinsOverHome(t *testing.T) {
	start := t.TempDir()
	home := t.TempDir()
	want := seedBaseURL(t, start, "https://project.example.com/api/v5")
	seedBaseURL(t, home, "https://home.example.com/api/v5")

	value, path, found, err := ResolveBaseURLFile(start, home)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !found {
		t.Fatalf("found = false, want true")
	}
	if value != "https://project.example.com/api/v5" || path != want {
		t.Errorf("got (%q, %q), want the project file (%q, %q)", value, path, "https://project.example.com/api/v5", want)
	}
}

func TestResolveBaseURLFile_WalkUpReachesAncestor(t *testing.T) {
	base := t.TempDir()
	start := filepath.Join(base, "a", "b")
	if err := os.MkdirAll(start, 0o755); err != nil {
		t.Fatal(err)
	}
	want := seedBaseURL(t, base, "https://ancestor.example.com/api/v5")

	value, path, found, err := ResolveBaseURLFile(start, t.TempDir())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !found || value != "https://ancestor.example.com/api/v5" || path != want {
		t.Errorf("got (%q, %q, %v), want the ancestor file %q", value, path, found, want)
	}
}

func TestResolveBaseURLFile_BaseURLlessFileIsSkipped(t *testing.T) {
	start := t.TempDir()
	home := t.TempDir()
	// A file that exists and parses but holds no base_url in the nearest location
	// must not shadow the home file that has one — the walk continues.
	if err := os.WriteFile(filepath.Join(start, credentialsFileName), []byte("token=gf_only\n# no base url\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	want := seedBaseURL(t, home, "https://home.example.com/api/v5")

	value, path, found, err := ResolveBaseURLFile(start, home)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !found || value != "https://home.example.com/api/v5" || path != want {
		t.Errorf("got (%q, %q, %v), want the home file %q (base_url-less nearest skipped)", value, path, found, want)
	}
}

func TestResolveBaseURLFile_HomeFallbackWhenNoProjectFile(t *testing.T) {
	home := t.TempDir()
	want := seedBaseURL(t, home, "https://home.example.com/api/v5")
	start := filepath.Join(t.TempDir(), "x", "y")
	if err := os.MkdirAll(start, 0o755); err != nil {
		t.Fatal(err)
	}

	value, path, found, err := ResolveBaseURLFile(start, home)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !found || value != "https://home.example.com/api/v5" || path != want {
		t.Errorf("got (%q, %q, %v), want the home file %q", value, path, found, want)
	}
}

func TestResolveBaseURLFile_HomeOnAscentPathReadOnce(t *testing.T) {
	home := t.TempDir()
	start := filepath.Join(home, "projects", "app")
	if err := os.MkdirAll(start, 0o755); err != nil {
		t.Fatal(err)
	}
	want := seedBaseURL(t, home, "https://home.example.com/api/v5")

	value, path, found, err := ResolveBaseURLFile(start, home)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !found || value != "https://home.example.com/api/v5" || path != want {
		t.Errorf("got (%q, %q, %v), want a single read at the home file %q", value, path, found, want)
	}
}

func TestResolveBaseURLFile_NoneFoundIsNotAnError(t *testing.T) {
	start := filepath.Join(t.TempDir(), "deep", "dir")
	if err := os.MkdirAll(start, 0o755); err != nil {
		t.Fatal(err)
	}

	value, path, found, err := ResolveBaseURLFile(start, t.TempDir())
	if err != nil {
		t.Fatalf("absence must not be an error, got: %v", err)
	}
	if found {
		t.Errorf("found = true, want false when no base_url exists anywhere")
	}
	if value != "" || path != "" {
		t.Errorf("a not-found resolution carried data: value=%q path=%q", value, path)
	}
}

func TestResolveBaseURLFile_UnreadableFileFailsLoud(t *testing.T) {
	start := t.TempDir()
	home := t.TempDir()
	seedBaseURL(t, home, "https://home.example.com/api/v5") // a usable source exists further along
	// A directory at the .glassfrogrc path makes os.ReadFile fail deterministically
	// across platforms (a path-only PathError, not os.ErrNotExist) — exercising the
	// fail-loud branch without OS-dependent 0o000 permission semantics (LEARNINGS).
	bad := filepath.Join(start, credentialsFileName)
	if err := os.Mkdir(bad, 0o755); err != nil {
		t.Fatal(err)
	}

	_, _, _, err := ResolveBaseURLFile(start, home)
	if err == nil {
		t.Fatalf("expected a read error, got nil (must not fall through to home)")
	}
	var re *ReadError
	if !errors.As(err, &re) {
		t.Fatalf("expected *ReadError, got %T: %v", err, err)
	}
	if errors.Is(err, os.ErrNotExist) {
		t.Errorf("a directory at the path must not be reported as absence: %v", err)
	}
	if re.Path != bad {
		t.Errorf("ReadError.Path = %q, want %q", re.Path, bad)
	}
}

func TestResolveBaseURLFile_MalformedFileFailsLoudNotSkip(t *testing.T) {
	start := t.TempDir()
	bad := filepath.Join(start, credentialsFileName)
	if err := os.WriteFile(bad, []byte("this is not a key=value pair\noops\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, _, _, err := ResolveBaseURLFile(start, t.TempDir())
	if err == nil {
		t.Fatalf("expected a format error, got nil (must not report not-found)")
	}
	var fe *FormatError
	if !errors.As(err, &fe) {
		t.Fatalf("expected *FormatError, got %T: %v", err, err)
	}
	if fe.Path != bad {
		t.Errorf("FormatError.Path = %q, want %q", fe.Path, bad)
	}
}

func TestResolveBaseURLFile_TokenNeverInResultOrError(t *testing.T) {
	// Even when the winning file holds a token beside the base_url, the resolved
	// value carries only the base URL and the token never appears.
	secret := "gf_super_secret_token"
	start := t.TempDir()
	want := seedTokenAndBaseURL(t, start, secret, "https://project.example.com/api/v5")

	value, path, found, err := ResolveBaseURLFile(start, t.TempDir())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !found || path != want {
		t.Fatalf("got (found=%v, path=%q), want (true, %q)", found, path, want)
	}
	if strings.Contains(value, secret) {
		t.Errorf("the token value leaked into the resolved base URL: %q", value)
	}
}

// TestParseCredentials_TokenUnaffectedByBaseURLKey pins that generalizing the
// shared parser to also capture base_url did not widen what the token reader
// exposes: a base_url-only file still reads as tokenless.
func TestReadCredentialsFile_BaseURLOnlyFileIsTokenless(t *testing.T) {
	path := writeFile(t, "base_url=https://glassfrog.com/api/v5\n")

	token, found, err := readCredentialsFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found || token != "" {
		t.Errorf("got (%q, %v), want (\"\", false) — a base_url-only file has no token", token, found)
	}
}
