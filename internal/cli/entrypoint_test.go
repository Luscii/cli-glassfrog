package cli

import (
	"bytes"
	"errors"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/spf13/cobra"
)

// codeTree builds a small assembled tree for entrypoint tests: a `version` leaf
// that succeeds, a `boom` leaf whose action returns an error, and a `panicker`
// leaf whose action panics — so a test can drive each exit-code path through
// the recover+map core that Main runs.
func codeTree() *cobra.Command {
	root := NewRootCommand()
	leaf := func(name string, run func(*cobra.Command, []string) error) *cobra.Command {
		return &cobra.Command{Use: name, Short: "the " + name + " command", Args: cobra.NoArgs, RunE: run}
	}
	MustRegister(root, leaf("version", func(*cobra.Command, []string) error { return nil }))
	MustRegister(root, leaf("boom", func(*cobra.Command, []string) error { return errors.New("kaboom") }))
	MustRegister(root, leaf("panicker", func(*cobra.Command, []string) error { panic("nil deref") }))
	return root
}

// runToExitCode maps each producer-backed outcome to its code: a clean run is 0,
// an unknown command is 2, and a resolved command whose action fails is 1.
func TestRunToExitCode_MapsOutcomes(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want int
	}{
		{"success", []string{"version"}, 0},
		{"unknown command", []string{"nope"}, 2},
		{"runtime failure", []string{"boom"}, 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := codeTree()
			root.SetOut(&bytes.Buffer{})
			root.SetErr(&bytes.Buffer{})
			if got := runToExitCode(root, tc.args); got != tc.want {
				t.Fatalf("runToExitCode(%v) = %d, want %d", tc.args, got, tc.want)
			}
		})
	}
}

// A panic in a resolved command's action is recovered and mapped to 1 — never
// Go's default panic exit status 2, which would collide with UsageError (ADR-4).
// The panic value is written to stderr so the crash stays diagnosable.
func TestRunToExitCode_PanicYieldsOneNotTwo(t *testing.T) {
	stderr, restore := captureStderr(t)
	defer restore()

	root := codeTree()
	root.SetOut(&bytes.Buffer{})
	root.SetErr(&bytes.Buffer{})
	got := runToExitCode(root, []string{"panicker"})

	if got == 2 {
		t.Fatal("panic exited 2 — it must not collide with the usage code")
	}
	if got != 1 {
		t.Fatalf("panic exit code = %d, want 1", got)
	}
	if out := stderr(); !strings.Contains(out, "nil deref") {
		t.Fatalf("panic value should be written to stderr for diagnosability, got: %q", out)
	}
}

// Main wires the real assembled tree to os.Args and returns the mapped code.
// A successful invocation yields 0 — pinning the Assemble()+os.Args wiring that
// runToExitCode's tests drive with a custom tree.
func TestMain_SuccessfulInvocationYieldsZero(t *testing.T) {
	defer swapArgs(t, []string{"glassfrog", "version"})()
	defer silenceStdout(t)() // Main writes the real command's output to os.Stdout
	if got := Main(); got != 0 {
		t.Fatalf("Main() for a successful invocation = %d, want 0", got)
	}
}

// captureStderr redirects os.Stderr to a temp file for the duration of a test,
// returning a getter for what was written and a restore func. A temp file
// (rather than an os.Pipe) is used deliberately: the code under test writes the
// panic diagnostic synchronously, with no concurrent reader, so a pipe could
// block and deadlock once a large write filled the OS pipe buffer. A file has
// no such bound and its descriptor is closed on restore.
//
// restore is idempotent (guarded by sync.Once), so calling it more than once —
// e.g. a `defer restore()` alongside a get() that restores internally — is
// safe and never double-closes the file. get() stops capturing (it calls
// restore) before reading, so it returns everything written up to that point.
func captureStderr(t *testing.T) (func() string, func()) {
	t.Helper()
	orig := os.Stderr
	f, err := os.CreateTemp(t.TempDir(), "stderr-*")
	if err != nil {
		t.Fatalf("create temp stderr: %v", err)
	}
	os.Stderr = f
	var once sync.Once
	restore := func() {
		once.Do(func() {
			os.Stderr = orig
			f.Close()
		})
	}
	get := func() string {
		restore()
		out, err := os.ReadFile(f.Name())
		if err != nil {
			t.Fatalf("read captured stderr: %v", err)
		}
		return string(out)
	}
	return get, restore
}

// swapArgs replaces os.Args for a test and returns a restore func.
func swapArgs(t *testing.T, args []string) func() {
	t.Helper()
	orig := os.Args
	os.Args = args
	return func() { os.Args = orig }
}

// silenceStdout discards os.Stdout for the duration of a test (so a real
// command's output does not leak into the test log) and returns a restore func.
func silenceStdout(t *testing.T) func() {
	t.Helper()
	orig := os.Stdout
	devnull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		t.Fatalf("open %s: %v", os.DevNull, err)
	}
	os.Stdout = devnull
	return func() {
		os.Stdout = orig
		devnull.Close()
	}
}
