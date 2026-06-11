package resolve

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Luscii/cli-glassfrog/internal/rcfile"
)

// --- T001: core types ---

func TestSourceKindString(t *testing.T) {
	cases := map[SourceKind]string{
		KindNone:    "none",
		KindFlag:    "flag",
		KindEnv:     "env",
		KindFile:    "file",
		KindStdin:   "stdin",
		KindDefault: "default",
	}
	if len(cases) != 6 {
		t.Fatalf("expected 6 kinds, mapped %d", len(cases))
	}
	for k, want := range cases {
		if got := k.String(); got != want {
			t.Errorf("SourceKind(%d).String() = %q, want %q", k, got, want)
		}
	}
	if KindNone != 0 {
		t.Errorf("KindNone = %d, want 0 (the zero value)", KindNone)
	}
}

func TestResolutionFound(t *testing.T) {
	if (Resolution{}).Found() {
		t.Error("zero Resolution{}.Found() = true, want false (nothing found)")
	}
	if !(Resolution{Provenance: Provenance{Kind: KindDefault}}).Found() {
		t.Error("a KindDefault resolution should report Found() == true")
	}
	if !(Resolution{Provenance: Provenance{Kind: KindFlag}}).Found() {
		t.Error("a KindFlag resolution should report Found() == true")
	}
}

func TestSourceKindWithoutEvaluating(t *testing.T) {
	// Kind() must not run eval — the Stdin guard reads it before the walk.
	s := FromEnv(func(string) string {
		t.Fatal("Kind() evaluated the source")
		return ""
	}, "X")
	if s.Kind() != KindEnv {
		t.Errorf("Kind() = %v, want KindEnv", s.Kind())
	}
}

// --- T002: the Resolve walk ---

func TestResolveFirstYieldWins(t *testing.T) {
	res, err := Resolve(
		FromFlags(Flag{Name: "--flag", Present: true, Value: "fromflag"}),
		FromEnv(func(string) string { return "fromenv" }, "X"),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Value != "fromflag" || res.Provenance.Kind != KindFlag {
		t.Errorf("got %+v, want the flag value with KindFlag", res)
	}
}

func TestResolveLazyShortCircuit(t *testing.T) {
	// Once the flag yields, the lower env source's lookup must never run.
	tripwire := FromEnv(func(string) string {
		t.Fatal("a lower source was evaluated after a higher source yielded")
		return ""
	}, "X")
	if _, err := Resolve(
		FromFlags(Flag{Name: "--flag", Present: true, Value: "v"}),
		tripwire,
	); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResolveNoneFound(t *testing.T) {
	res, err := Resolve(
		FromFlags(Flag{Name: "--flag", Present: false}),
		FromEnv(func(string) string { return "" }, "X"),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Found() || res.Provenance.Kind != KindNone {
		t.Errorf("got %+v, want a KindNone not-found result", res)
	}
}

func TestResolveErrorAbortsNoFallThrough(t *testing.T) {
	boom := errors.New("boom")
	res, err := Resolve(
		errSource(boom),
		FromFlags(Flag{Name: "--flag", Present: true, Value: "lower"}),
	)
	if !errors.Is(err, boom) {
		t.Fatalf("err = %v, want the verbatim boom error", err)
	}
	if res.Value != "" {
		t.Errorf("res.Value = %q, want empty (no fall-through to a lower source)", res.Value)
	}
}

func TestResolvePanicsOnMultipleStdin(t *testing.T) {
	drained := false
	read := func() (string, error) { drained = true; return "x", nil }
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected a panic on two Stdin sources")
		}
		if msg, _ := r.(string); !strings.Contains(msg, "Stdin") {
			t.Errorf("panic message %q does not name the Stdin misuse", r)
		}
		if drained {
			t.Error("a stdin source was evaluated before the guard panicked")
		}
	}()
	// The guard panics before returning, so the result is never produced; the
	// blank assignment satisfies errcheck without implying a checkable outcome.
	_, _ = Resolve(FromStdin(read, false), FromStdin(read, false))
}

func TestResolvePanicsOnZeroValueSource(t *testing.T) {
	// A caller that bypasses the constructors can only produce a zero-value
	// Source (nil eval). The guard must fail fast with a clear message before the
	// walk dereferences eval, and before any other source is evaluated.
	tripwire := FromEnv(func(string) string {
		t.Fatal("a source was evaluated before the zero-value guard panicked")
		return ""
	}, "X")
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected a panic on a zero-value Source")
		}
		if msg, _ := r.(string); !strings.Contains(msg, "zero-value Source") {
			t.Errorf("panic message %q does not name the zero-value misuse", r)
		}
	}()
	_, _ = Resolve(Source{}, tripwire)
}

// errSource is a test-only Source whose eval returns err — exercises the
// abort-and-surface path without a real I/O failure.
func errSource(err error) Source {
	return Source{kind: KindFile, eval: func() (string, string, bool, error) {
		return "", "", false, err
	}}
}

// --- T003: value-only constructors ---

func TestFromFlagsPresentEmptyValueYields(t *testing.T) {
	res, err := Resolve(FromFlags(Flag{Name: "--flag", Present: true, Value: ""}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Provenance.Kind != KindFlag || res.Provenance.Origin != "--flag" || res.Value != "" {
		t.Errorf("got %+v, want a presence-based empty-value flag yield with Origin --flag", res)
	}
}

func TestFromFlagsAliasWalk(t *testing.T) {
	res, err := Resolve(FromFlags(
		Flag{Name: "--output", Present: false},
		Flag{Name: "-o", Present: true, Value: "json"},
	))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Value != "json" || res.Provenance.Origin != "-o" {
		t.Errorf("got %+v, want -o's value json with Origin -o", res)
	}
}

func TestFromEnvSkipsWhitespaceAndReportsName(t *testing.T) {
	lookup := func(name string) string {
		switch name {
		case "A":
			return "   "
		case "B":
			return "value-b"
		default:
			return ""
		}
	}
	res, err := Resolve(FromEnv(lookup, "A", "B"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Value != "value-b" || res.Provenance.Kind != KindEnv || res.Provenance.Origin != "B" {
		t.Errorf("got %+v, want value-b from env name B", res)
	}
}

func TestDefaultAlwaysYields(t *testing.T) {
	res, err := Resolve(Default("the-default"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Value != "the-default" || res.Provenance.Kind != KindDefault || res.Provenance.Origin != "" {
		t.Errorf("got %+v, want the-default with KindDefault and empty Origin", res)
	}
}

// --- T004: the file source ---

func TestFromFileYieldsWithResolvedPath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, rcfile.FileName)
	if err := os.WriteFile(path, []byte("base_url=https://example.test\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	res, err := Resolve(FromFile(dir, "", "base_url"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Value != "https://example.test" || res.Provenance.Kind != KindFile || res.Provenance.Origin != path {
		t.Errorf("got %+v, want the file value with Origin %q", res, path)
	}
}

func TestFromFileMissingOrKeyLessDoesNotYield(t *testing.T) {
	dir := t.TempDir() // no .glassfrogrc seeded
	res, err := Resolve(FromFile(dir, "", "base_url"), Default("backstop"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Provenance.Kind != KindDefault {
		t.Errorf("got %+v, want fall-through to the default (file did not yield)", res)
	}
}

func TestFromFileMalformedSurfacesTypedErrorAndAborts(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, rcfile.FileName)
	if err := os.WriteFile(path, []byte("not-a-key-value-line\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	res, err := Resolve(FromFile(dir, "", "base_url"), Default("backstop"))
	var fe *rcfile.FormatError
	if !errors.As(err, &fe) {
		t.Fatalf("err = %T %v, want a verbatim *rcfile.FormatError", err, err)
	}
	if fe.Path != path {
		t.Errorf("FormatError.Path = %q, want %q", fe.Path, path)
	}
	if res.Value != "" {
		t.Errorf("res.Value = %q, want empty (no fall-through to the default)", res.Value)
	}
}

// --- T005: the stdin source ---

func TestFromStdinPipedYieldsTrimmed(t *testing.T) {
	res, err := Resolve(FromStdin(func() (string, error) { return "  piped\n", nil }, false))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Value != "piped" || res.Provenance.Kind != KindStdin || res.Provenance.Origin != "" {
		t.Errorf("got %+v, want trimmed piped content with KindStdin and empty Origin", res)
	}
}

func TestFromStdinTTYNeverReads(t *testing.T) {
	read := func() (string, error) {
		t.Fatal("read was invoked on a TTY")
		return "", nil
	}
	res, err := Resolve(FromStdin(read, true), Default("backstop"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Provenance.Kind != KindDefault {
		t.Errorf("got %+v, want fall-through to the default (TTY stdin does not yield)", res)
	}
}

func TestFromStdinEmptyDoesNotYield(t *testing.T) {
	res, err := Resolve(FromStdin(func() (string, error) { return "   \n", nil }, false), Default("backstop"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Provenance.Kind != KindDefault {
		t.Errorf("got %+v, want fall-through to the default (whitespace-only stdin does not yield)", res)
	}
}

func TestFromStdinReadFailureAborts(t *testing.T) {
	boom := errors.New("read boom")
	res, err := Resolve(FromStdin(func() (string, error) { return "", boom }, false), Default("backstop"))
	if !errors.Is(err, boom) {
		t.Fatalf("err = %v, want the verbatim read error", err)
	}
	if res.Value != "" {
		t.Errorf("res.Value = %q, want empty (read failure aborts, no fall-through)", res.Value)
	}
}

func TestReadBoundedStdinUnderAndOverCap(t *testing.T) {
	got, err := readBoundedStdin(strings.NewReader("small"))
	if err != nil || got != "small" {
		t.Fatalf("under-cap read: got %q, err %v", got, err)
	}
	over := strings.NewReader(strings.Repeat("x", maxStdinBytes+1))
	if _, err := readBoundedStdin(over); err == nil {
		t.Error("over-cap read returned no error; want an error (no silent truncation)")
	}
}

// --- T006: OS binding ---

func TestEnvFromOSMatchesPureConstructor(t *testing.T) {
	t.Setenv("RESOLVE_OS_BINDING_TEST", "from-os")
	res, err := Resolve(EnvFromOS("RESOLVE_OS_BINDING_TEST"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Value != "from-os" || res.Provenance.Kind != KindEnv || res.Provenance.Origin != "RESOLVE_OS_BINDING_TEST" {
		t.Errorf("got %+v, want the OS env value identical to the pure constructor", res)
	}
}

func TestOSRootsReturnsWorkingDirectory(t *testing.T) {
	startDir, _, err := OSRoots()
	if err != nil {
		t.Fatalf("OSRoots error: %v", err)
	}
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd error: %v", err)
	}
	if startDir != wd {
		t.Errorf("startDir = %q, want the working directory %q", startDir, wd)
	}
	// homeDir is best-effort: "" when undeterminable is acceptable, never an error.
}
