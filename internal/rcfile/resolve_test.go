package rcfile

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// seed writes a .glassfrogrc holding the given content into dir (created if
// needed) and returns the file path. Confined to the caller's temp tree.
func seed(t *testing.T, dir, content string) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}
	path := filepath.Join(dir, FileName)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("seeding %s: %v", path, err)
	}
	return path
}

func seedKey(t *testing.T, dir, key, value string) string {
	t.Helper()
	return seed(t, dir, key+"="+value+"\n")
}

func TestResolve_NearestWinsOverHome(t *testing.T) {
	start := t.TempDir()
	home := t.TempDir()
	want := seedKey(t, start, "base_url", "https://project.example.com/api/v5")
	seedKey(t, home, "base_url", "https://home.example.com/api/v5")

	value, path, found, err := Resolve(start, home, "base_url")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !found || value != "https://project.example.com/api/v5" || path != want {
		t.Errorf("got (%q, %q, %v), want the project file %q", value, path, found, want)
	}
}

func TestResolve_WalkUpReachesAncestor(t *testing.T) {
	base := t.TempDir()
	start := filepath.Join(base, "a", "b")
	if err := os.MkdirAll(start, 0o755); err != nil {
		t.Fatal(err)
	}
	want := seedKey(t, base, "token", "gf_ancestor")

	value, path, found, err := Resolve(start, t.TempDir(), "token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !found || value != "gf_ancestor" || path != want {
		t.Errorf("got (%q, %q, %v), want the ancestor file %q", value, path, found, want)
	}
}

func TestResolve_KeylessFileIsSkipped(t *testing.T) {
	start := t.TempDir()
	home := t.TempDir()
	// A file that exists and parses but lacks the requested key must not shadow a
	// lower file that has it.
	seed(t, start, "token=gf_only\n# no base_url here\n")
	want := seedKey(t, home, "base_url", "https://home.example.com/api/v5")

	value, path, found, err := Resolve(start, home, "base_url")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !found || value != "https://home.example.com/api/v5" || path != want {
		t.Errorf("got (%q, %q, %v), want the home file %q (key-less nearest skipped)", value, path, found, want)
	}
}

func TestResolve_HomeFallbackWhenNoProjectFile(t *testing.T) {
	home := t.TempDir()
	want := seedKey(t, home, "token", "gf_home")
	start := filepath.Join(t.TempDir(), "x", "y")
	if err := os.MkdirAll(start, 0o755); err != nil {
		t.Fatal(err)
	}

	value, path, found, err := Resolve(start, home, "token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !found || value != "gf_home" || path != want {
		t.Errorf("got (%q, %q, %v), want the home file %q", value, path, found, want)
	}
}

func TestResolve_HomeOnAscentPathReadOnce(t *testing.T) {
	home := t.TempDir()
	start := filepath.Join(home, "projects", "app")
	if err := os.MkdirAll(start, 0o755); err != nil {
		t.Fatal(err)
	}
	want := seedKey(t, home, "token", "gf_home")

	value, path, found, err := Resolve(start, home, "token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !found || value != "gf_home" || path != want {
		t.Errorf("got (%q, %q, %v), want a single read at the home file %q", value, path, found, want)
	}
}

func TestResolve_NoneFoundIsNotAnError(t *testing.T) {
	start := filepath.Join(t.TempDir(), "deep", "dir")
	if err := os.MkdirAll(start, 0o755); err != nil {
		t.Fatal(err)
	}

	value, path, found, err := Resolve(start, t.TempDir(), "token")
	if err != nil {
		t.Fatalf("absence must not be an error, got: %v", err)
	}
	if found || value != "" || path != "" {
		t.Errorf("a not-found resolution carried data: value=%q path=%q found=%v", value, path, found)
	}
}

func TestResolve_UnreadableFileFailsLoud(t *testing.T) {
	start := t.TempDir()
	home := t.TempDir()
	seedKey(t, home, "token", "gf_home") // a usable source exists further along
	// A directory at the .glassfrogrc path makes os.ReadFile fail deterministically
	// across platforms (a path-only error, not os.ErrNotExist) — the fail-loud
	// branch without OS-dependent 0o000 semantics (LEARNINGS).
	bad := filepath.Join(start, FileName)
	if err := os.Mkdir(bad, 0o755); err != nil {
		t.Fatal(err)
	}

	_, _, _, err := Resolve(start, home, "token")
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

func TestResolve_MalformedFileFailsLoudNotSkip(t *testing.T) {
	start := t.TempDir()
	bad := seed(t, start, "this is not a key=value pair\noops\n")

	_, _, _, err := Resolve(start, t.TempDir(), "token")
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

func TestResolve_ReturnsOnlyRequestedKey(t *testing.T) {
	// Even when the winning file holds a token beside the base_url, resolving
	// base_url returns only the base URL — the token never crosses the walk.
	secret := "gf_super_secret_token"
	start := t.TempDir()
	want := seed(t, start, "token="+secret+"\nbase_url=https://project.example.com/api/v5\n")

	value, path, found, err := Resolve(start, t.TempDir(), "base_url")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !found || path != want {
		t.Fatalf("got (found=%v, path=%q), want (true, %q)", found, path, want)
	}
	if strings.Contains(value, secret) {
		t.Errorf("the token leaked into the resolved base URL: %q", value)
	}
}

// candidateDirs is the de-duplicated search order. These cases pin the two
// correctness traps: a home directory that lies on the ascent path is listed
// once, and the walk-up stops at the filesystem root (no infinite loop, no
// duplicate root entry).
func TestCandidateDirs_HomeOnAscentPathDeduped(t *testing.T) {
	home := filepath.Clean(t.TempDir())
	start := filepath.Join(home, "a", "b")

	dirs := candidateDirs(start, home)

	if count := countOccurrences(dirs, home); count != 1 {
		t.Errorf("home %s appears %d times in %v, want exactly 1", home, count, dirs)
	}
	if hasDuplicates(dirs) {
		t.Errorf("candidate list has duplicates: %v", dirs)
	}
}

func TestCandidateDirs_HomeAppendedWhenDistinct(t *testing.T) {
	start := filepath.Clean(t.TempDir())
	home := filepath.Clean(t.TempDir())

	dirs := candidateDirs(start, home)

	if dirs[0] != start {
		t.Errorf("first candidate = %q, want start %q", dirs[0], start)
	}
	if dirs[len(dirs)-1] != home {
		t.Errorf("last candidate = %q, want home %q", dirs[len(dirs)-1], home)
	}
}

func TestCandidateDirs_StopsAtRootNoDuplicates(t *testing.T) {
	start := filepath.Clean(t.TempDir())

	dirs := candidateDirs(start, "")

	root := "/"
	for d := start; ; {
		parent := filepath.Dir(d)
		if parent == d {
			root = d
			break
		}
		d = parent
	}
	if countOccurrences(dirs, root) != 1 {
		t.Errorf("root %s should appear exactly once in %v", root, dirs)
	}
	if hasDuplicates(dirs) {
		t.Errorf("candidate list has duplicates: %v", dirs)
	}
}

func countOccurrences(s []string, target string) int {
	n := 0
	for _, v := range s {
		if v == target {
			n++
		}
	}
	return n
}

func hasDuplicates(s []string) bool {
	seen := map[string]bool{}
	for _, v := range s {
		if seen[v] {
			return true
		}
		seen[v] = true
	}
	return false
}
