package auth

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// stubEnv overrides the env seam so resolution sees a controlled GLASSFROG_TOKEN
// without touching the real process environment. An empty val models both an
// empty and an unset variable (resolve treats them identically — both fall
// through to files).
func stubEnv(t *testing.T, val string) {
	t.Helper()
	orig := getenv
	t.Cleanup(func() { getenv = orig })
	getenv = func(key string) string {
		if key == envTokenVar {
			return val
		}
		return ""
	}
}

// seedToken writes a .glassfrogrc holding token=val into dir (created if needed)
// and returns the file path. Confined to the caller's temp tree.
func seedToken(t *testing.T, dir, val string) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}
	path := filepath.Join(dir, credentialsFileName)
	if err := os.WriteFile(path, []byte("token="+val+"\n"), 0o600); err != nil {
		t.Fatalf("seeding %s: %v", path, err)
	}
	return path
}

func TestResolve_EnvFirstShortCircuitsFiles(t *testing.T) {
	stubEnv(t, "gf_env_token")
	home := t.TempDir()
	seedToken(t, home, "gf_home_token") // present, but must not be read

	got, err := resolve(filepath.Join(home, "work"), home)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Source != SourceEnvironment {
		t.Errorf("Source = %v, want Environment", got.Source)
	}
	if got.Token != "gf_env_token" {
		t.Errorf("Token = %q, want gf_env_token", got.Token)
	}
	if got.Path != "" {
		t.Errorf("Path = %q, want empty (no file read on env hit)", got.Path)
	}
}

func TestResolve_EmptyEnvFallsThroughToFile(t *testing.T) {
	stubEnv(t, "") // set-but-empty / unset both look like ""
	start := t.TempDir()
	want := seedToken(t, start, "gf_file_token")

	got, err := resolve(start, t.TempDir())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Source != SourceFile || got.Token != "gf_file_token" || got.Path != want {
		t.Errorf("got %+v, want File/gf_file_token/%s", got, want)
	}
}

func TestResolve_WhitespaceOnlyEnvFallsThroughToFile(t *testing.T) {
	// A whitespace-only GLASSFROG_TOKEN is treated as absent (not a usable
	// credential) and must not short-circuit the file search.
	stubEnv(t, "   \t ")
	start := t.TempDir()
	want := seedToken(t, start, "gf_file_token")

	got, err := resolve(start, t.TempDir())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Source != SourceFile || got.Token != "gf_file_token" || got.Path != want {
		t.Errorf("got %+v, want File/gf_file_token/%s (whitespace env ignored)", got, want)
	}
}

func TestResolve_NearestWinsOverHome(t *testing.T) {
	stubEnv(t, "")
	start := t.TempDir()
	home := t.TempDir()
	want := seedToken(t, start, "gf_project_token")
	seedToken(t, home, "gf_home_token")

	got, err := resolve(start, home)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Token != "gf_project_token" || got.Path != want {
		t.Errorf("got %+v, want the project file %s", got, want)
	}
}

func TestResolve_WalkUpReachesAncestor(t *testing.T) {
	stubEnv(t, "")
	base := t.TempDir()
	ancestor := base
	start := filepath.Join(base, "a", "b") // two directories below the ancestor
	if err := os.MkdirAll(start, 0o755); err != nil {
		t.Fatal(err)
	}
	want := seedToken(t, ancestor, "gf_ancestor_token")

	got, err := resolve(start, t.TempDir())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Source != SourceFile || got.Token != "gf_ancestor_token" || got.Path != want {
		t.Errorf("got %+v, want the ancestor file %s", got, want)
	}
}

func TestResolve_HomeFallbackWhenNoProjectFile(t *testing.T) {
	stubEnv(t, "")
	home := t.TempDir()
	want := seedToken(t, home, "gf_home_token")
	// A start directory deep under a different tree with no .glassfrogrc.
	start := filepath.Join(t.TempDir(), "x", "y")
	if err := os.MkdirAll(start, 0o755); err != nil {
		t.Fatal(err)
	}

	got, err := resolve(start, home)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Source != SourceFile || got.Token != "gf_home_token" || got.Path != want {
		t.Errorf("got %+v, want the home file %s", got, want)
	}
}

func TestResolve_HomeOnAscentPathReadOnce(t *testing.T) {
	stubEnv(t, "")
	home := t.TempDir()
	start := filepath.Join(home, "projects", "app") // home is an ancestor of start
	if err := os.MkdirAll(start, 0o755); err != nil {
		t.Fatal(err)
	}
	want := seedToken(t, home, "gf_home_token")

	got, err := resolve(start, home)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Found during the walk-up at the home directory's position: a File source
	// with the home file's path (not a duplicate read or a None).
	if got.Source != SourceFile || got.Token != "gf_home_token" || got.Path != want {
		t.Errorf("got %+v, want a File source at %s", got, want)
	}
}

func TestResolve_TokenlessFileIsSkipped(t *testing.T) {
	stubEnv(t, "")
	start := t.TempDir()
	home := t.TempDir()
	// A tokenless (but valid) file in the nearest location must not shadow the
	// home file that has a token.
	if err := os.WriteFile(filepath.Join(start, credentialsFileName), []byte("# no token here\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	want := seedToken(t, home, "gf_home_token")

	got, err := resolve(start, home)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Token != "gf_home_token" || got.Path != want {
		t.Errorf("got %+v, want the home file %s (tokenless nearest skipped)", got, want)
	}
}

func TestResolve_NoCredentialsAnywhereIsNone(t *testing.T) {
	stubEnv(t, "")
	start := filepath.Join(t.TempDir(), "deep", "dir")
	if err := os.MkdirAll(start, 0o755); err != nil {
		t.Fatal(err)
	}

	got, err := resolve(start, t.TempDir())
	if err != nil {
		t.Fatalf("absence must not be an error, got: %v", err)
	}
	if got.Source != SourceNone {
		t.Errorf("Source = %v, want None", got.Source)
	}
	if got.Token != "" || got.Path != "" {
		t.Errorf("None resolution carried data: %+v", got)
	}
}

func TestResolve_UnreadableFileFailsLoud(t *testing.T) {
	stubEnv(t, "")
	start := t.TempDir()
	home := t.TempDir()
	seedToken(t, home, "gf_home_token") // a usable source exists further along
	// A directory at the .glassfrogrc path makes os.ReadFile fail
	// deterministically across platforms (and as root) — a path-only
	// PathError, not os.ErrNotExist — exercising the fail-loud branch without
	// 0o000 permission semantics that vary by OS / privilege. os.ReadFile
	// never reads content into its error, so the read path cannot leak a
	// token; content-leak hygiene is covered by the format-error test.
	bad := filepath.Join(start, credentialsFileName)
	if err := os.Mkdir(bad, 0o755); err != nil {
		t.Fatal(err)
	}

	_, err := resolve(start, home)
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

func TestResolve_MalformedFileFailsLoudNotAbsence(t *testing.T) {
	stubEnv(t, "")
	start := t.TempDir()
	bad := filepath.Join(start, credentialsFileName)
	if err := os.WriteFile(bad, []byte("this is not a key=value pair\noops\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := resolve(start, t.TempDir())
	if err == nil {
		t.Fatalf("expected a format error, got nil (must not report absence)")
	}
	var fe *FormatError
	if !errors.As(err, &fe) {
		t.Fatalf("expected *FormatError, got %T: %v", err, err)
	}
	if fe.Path != bad {
		t.Errorf("FormatError.Path = %q, want %q", fe.Path, bad)
	}
}

// candidateDirs is the de-duplicated search order. These cases pin the two
// correctness traps the plan flags: a home directory that lies on the ascent
// path is listed once, and the walk-up stops at the filesystem root (no
// infinite loop, no duplicate root entry).
func TestCandidateDirs_HomeOnAscentPathDeduped(t *testing.T) {
	home := filepath.Clean(t.TempDir())
	start := filepath.Join(home, "a", "b")

	dirs := candidateDirs(start, home)

	if count := countOccurrences(dirs, home); count != 1 {
		t.Errorf("home %s appears %d times in %v, want exactly 1", home, count, dirs)
	}
	// home is consulted once, at its natural ascent position — the walk-up then
	// continues above it to the root, so home is NOT the final entry here.
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

	// The ascent terminates at the filesystem root exactly once.
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
