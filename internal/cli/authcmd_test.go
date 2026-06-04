package cli

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Luscii/cli-glassfrog/internal/auth"
	"github.com/spf13/cobra"
)

// --- shared test doubles (reused by the 006 BDD steps) ---

// fakeInteractor scripts the interactive responses and records which terminal
// interactions were requested, so a test asserts both behavior and the fact
// that a prompt/confirm/choice was offered. Never touches a real terminal.
type fakeInteractor struct {
	promptValue string
	promptErr   error
	confirm     bool
	confirmErr  error
	chosen      []string
	chooseErr   error

	promptCalled  bool
	confirmCalled bool
	chooseCalled  bool
	choseHome     string
	choseCwd      string
}

func (f *fakeInteractor) promptToken() (string, error) {
	f.promptCalled = true
	return f.promptValue, f.promptErr
}

func (f *fakeInteractor) confirmReplace(string) (bool, error) {
	f.confirmCalled = true
	return f.confirm, f.confirmErr
}

func (f *fakeInteractor) chooseLocations(homePath, cwdPath string) ([]string, error) {
	f.chooseCalled = true
	f.choseHome, f.choseCwd = homePath, cwdPath
	if f.chooseErr != nil {
		return nil, f.chooseErr
	}
	if f.chosen != nil {
		return f.chosen, nil
	}
	return []string{homePath}, nil
}

// fakeSeam injects pre-built inputs, temp-dir roots, and a fake interactor. The
// positional argument still arrives through cobra (merged here), so a
// command-level test exercises real flag/arg parsing.
type fakeSeam struct {
	inputs tokenInputs
	home   string
	start  string
	inter  interactor
}

func (f *fakeSeam) gatherInputs(args []string) (tokenInputs, error) {
	in := f.inputs
	if len(args) >= 1 {
		in.arg, in.argGiven = args[0], true
	}
	return in, nil
}

func (f *fakeSeam) homeDir() (string, error)  { return f.home, nil }
func (f *fakeSeam) startDir() (string, error) { return f.start, nil }
func (f *fakeSeam) interactor() interactor    { return f.inter }

// runLoginCapture runs runLogin with captured stdout/stderr.
func runLoginCapture(cfg loginConfig) (Outcome, string, string) {
	var out, errb bytes.Buffer
	cfg.stdout, cfg.stderr = &out, &errb
	outcome, _ := runLogin(cfg)
	return outcome, out.String(), errb.String()
}

// --- runLogin: token sources and the happy path ---

func TestRunLogin_ArgToHome_Success(t *testing.T) {
	home, start := t.TempDir(), t.TempDir()
	outcome, stdout, stderr := runLoginCapture(loginConfig{
		inputs:  tokenInputs{arg: "gf_new_token", argGiven: true},
		homeDir: home, startDir: start,
		interact: &fakeInteractor{},
	})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success (stderr: %s)", outcome, stderr)
	}
	homePath := filepath.Join(home, auth.CredentialsFileName)
	tok, has, err := auth.ReadCredentialsFile(homePath)
	if err != nil || !has || tok != "gf_new_token" {
		t.Fatalf("home file token = %q has=%v err=%v", tok, has, err)
	}
	if !strings.Contains(stdout, homePath) {
		t.Fatalf("stdout should name the written path, got %q", stdout)
	}
	if strings.Contains(stdout+stderr, "gf_new_token") {
		t.Fatalf("the token must never appear in output, got stdout=%q stderr=%q", stdout, stderr)
	}
}

func TestRunLogin_EnvPersisted_Success(t *testing.T) {
	home, start := t.TempDir(), t.TempDir()
	outcome, _, _ := runLoginCapture(loginConfig{
		inputs:  tokenInputs{env: "gf_env_token"},
		homeDir: home, startDir: start,
		interact: &fakeInteractor{},
	})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success", outcome)
	}
	tok, _, _ := auth.ReadCredentialsFile(filepath.Join(home, auth.CredentialsFileName))
	if tok != "gf_env_token" {
		t.Fatalf("home token = %q, want gf_env_token", tok)
	}
}

func TestRunLogin_PipedToCwd_Success(t *testing.T) {
	home, start := t.TempDir(), t.TempDir()
	outcome, _, _ := runLoginCapture(loginConfig{
		inputs:  tokenInputs{stdin: "gf_project_token\n", stdinPiped: true},
		homeDir: home, startDir: start, cwd: true,
		interact: &fakeInteractor{},
	})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success", outcome)
	}
	tok, _, _ := auth.ReadCredentialsFile(filepath.Join(start, auth.CredentialsFileName))
	if tok != "gf_project_token" {
		t.Fatalf("cwd token = %q, want gf_project_token (trailing newline trimmed)", tok)
	}
	// Home must be untouched.
	if _, statErr := os.Stat(filepath.Join(home, auth.CredentialsFileName)); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatal("home file should not have been written under --cwd")
	}
}

// --- runLogin: usage outcomes (no write) ---

func TestRunLogin_NoTokenNonInteractive_UsageError(t *testing.T) {
	home, start := t.TempDir(), t.TempDir()
	outcome, _, stderr := runLoginCapture(loginConfig{
		inputs:  tokenInputs{isTTY: false},
		homeDir: home, startDir: start,
		interact: &fakeInteractor{},
	})
	if outcome != UsageError {
		t.Fatalf("outcome = %v, want UsageError", outcome)
	}
	if !strings.Contains(stderr, "no token to store") {
		t.Fatalf("stderr should report no token, got %q", stderr)
	}
	assertNoFile(t, filepath.Join(home, auth.CredentialsFileName))
}

func TestRunLogin_BlankToken_UsageError(t *testing.T) {
	home, start := t.TempDir(), t.TempDir()
	outcome, _, _ := runLoginCapture(loginConfig{
		inputs:  tokenInputs{arg: "   ", argGiven: true},
		homeDir: home, startDir: start,
		interact: &fakeInteractor{},
	})
	if outcome != UsageError {
		t.Fatalf("outcome = %v, want UsageError", outcome)
	}
	assertNoFile(t, filepath.Join(home, auth.CredentialsFileName))
}

func TestRunLogin_ExistingNonInteractiveNoOverwrite_UsageError(t *testing.T) {
	home, start := t.TempDir(), t.TempDir()
	homePath := filepath.Join(home, auth.CredentialsFileName)
	if err := os.WriteFile(homePath, []byte("token=gf_old_token\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	outcome, _, stderr := runLoginCapture(loginConfig{
		inputs:  tokenInputs{arg: "gf_new_token", argGiven: true, isTTY: false},
		homeDir: home, startDir: start, overwrite: false,
		interact: &fakeInteractor{},
	})
	if outcome != UsageError {
		t.Fatalf("outcome = %v, want UsageError", outcome)
	}
	if !strings.Contains(stderr, homePath) || !strings.Contains(stderr, "--overwrite") {
		t.Fatalf("stderr should name the path and --overwrite, got %q", stderr)
	}
	// The existing credential is unchanged.
	tok, _, _ := auth.ReadCredentialsFile(homePath)
	if tok != "gf_old_token" {
		t.Fatalf("existing token changed to %q, want gf_old_token", tok)
	}
}

func TestRunLogin_ExistingNonInteractiveOverwrite_Success(t *testing.T) {
	home, start := t.TempDir(), t.TempDir()
	homePath := filepath.Join(home, auth.CredentialsFileName)
	if err := os.WriteFile(homePath, []byte("token=gf_old_token\nkeep=me\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	outcome, _, _ := runLoginCapture(loginConfig{
		inputs:  tokenInputs{arg: "gf_new_token", argGiven: true, isTTY: false},
		homeDir: home, startDir: start, overwrite: true,
		interact: &fakeInteractor{},
	})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success", outcome)
	}
	got, _ := os.ReadFile(homePath)
	if !strings.Contains(string(got), "token=gf_new_token") || !strings.Contains(string(got), "keep=me") {
		t.Fatalf("overwrite merge wrong, got:\n%s", got)
	}
}

// --- runLogin: runtime outcomes ---

func TestRunLogin_MalformedExisting_RuntimeErrorNotOverwritten(t *testing.T) {
	home, start := t.TempDir(), t.TempDir()
	homePath := filepath.Join(home, auth.CredentialsFileName)
	original := "garbage line without an equals\n"
	if err := os.WriteFile(homePath, []byte(original), 0o600); err != nil {
		t.Fatal(err)
	}
	outcome, _, stderr := runLoginCapture(loginConfig{
		inputs:  tokenInputs{arg: "gf_new_token", argGiven: true},
		homeDir: home, startDir: start,
		interact: &fakeInteractor{},
	})
	if outcome != RuntimeError {
		t.Fatalf("outcome = %v, want RuntimeError", outcome)
	}
	if !strings.Contains(stderr, "format error") || !strings.Contains(stderr, homePath) {
		t.Fatalf("stderr should report a format error naming the file, got %q", stderr)
	}
	if got, _ := os.ReadFile(homePath); string(got) != original {
		t.Fatalf("malformed file must not be overwritten, got:\n%s", got)
	}
}

// An unreadable existing file fails at the pre-write guard (a read error, not a
// write or format error): the message must not claim "write error" or that the
// filesystem is unchanged in write terms — no write was attempted.
func TestRunLogin_UnreadableExisting_ReadErrorNotWriteError(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root bypasses file permission checks")
	}
	home, start := t.TempDir(), t.TempDir()
	homePath := filepath.Join(home, auth.CredentialsFileName)
	if err := os.WriteFile(homePath, []byte("token=gf_old_token\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(homePath, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(homePath, 0o600) })

	outcome, _, stderr := runLoginCapture(loginConfig{
		inputs:  tokenInputs{arg: "gf_new_token", argGiven: true},
		homeDir: home, startDir: start,
		interact: &fakeInteractor{},
	})
	if outcome != RuntimeError {
		t.Fatalf("outcome = %v, want RuntimeError", outcome)
	}
	if strings.Contains(stderr, "write error") {
		t.Fatalf("a read failure must not be reported as a write error, got %q", stderr)
	}
	if !strings.Contains(stderr, homePath) {
		t.Fatalf("stderr should name the unreadable file %q, got %q", homePath, stderr)
	}
}

// --- runLogin: interactive paths through the fake interactor ---

func TestRunLogin_NeedsPrompt_UsesInteractorAndWritesHome(t *testing.T) {
	home, start := t.TempDir(), t.TempDir()
	fi := &fakeInteractor{promptValue: "gf_prompted_token"}
	outcome, _, stderr := runLoginCapture(loginConfig{
		inputs:  tokenInputs{isTTY: true},
		homeDir: home, startDir: start,
		interact: fi,
	})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success (stderr %q)", outcome, stderr)
	}
	if !fi.promptCalled {
		t.Fatal("the prompt was not used for a missing token in an interactive session")
	}
	tok, _, _ := auth.ReadCredentialsFile(filepath.Join(home, auth.CredentialsFileName))
	if tok != "gf_prompted_token" {
		t.Fatalf("home token = %q, want the prompted value", tok)
	}
}

func TestRunLogin_InteractiveExisting_ConfirmsAndChoosesLocation(t *testing.T) {
	home, start := t.TempDir(), t.TempDir()
	homePath := filepath.Join(home, auth.CredentialsFileName)
	cwdPath := filepath.Join(start, auth.CredentialsFileName)
	if err := os.WriteFile(homePath, []byte("token=gf_home_old\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cwdPath, []byte("token=gf_cwd_old\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	fi := &fakeInteractor{confirm: true, chosen: []string{homePath}}
	outcome, _, _ := runLoginCapture(loginConfig{
		inputs:  tokenInputs{arg: "gf_new_token", argGiven: true, isTTY: true},
		homeDir: home, startDir: start,
		interact: fi,
	})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success", outcome)
	}
	if !fi.confirmCalled {
		t.Fatal("an interactive existing-token store must confirm before changing it")
	}
	if !fi.chooseCalled || fi.choseHome != homePath || fi.choseCwd != cwdPath {
		t.Fatalf("must offer home and cwd as locations, got home=%q cwd=%q called=%v", fi.choseHome, fi.choseCwd, fi.chooseCalled)
	}
}

// --- command wiring: resolves, accepts one positional, maps usage → code 2 ---

func TestAuthLoginCommand_ResolvesInAssembledTree(t *testing.T) {
	root := newTestRootWithAuth(&fakeSeam{home: t.TempDir(), start: t.TempDir(), inter: &fakeInteractor{}})
	got, _, err := root.Find([]string{"auth", "login"})
	if err != nil || got == nil || got.Name() != "login" {
		t.Fatalf("auth login did not resolve: got %v err %v", got, err)
	}
}

func TestAuthLogin_NoTokenNonInteractive_MapsToUsageCode2(t *testing.T) {
	home, start := t.TempDir(), t.TempDir()
	root := newTestRootWithAuth(&fakeSeam{home: home, start: start, inter: &fakeInteractor{}, inputs: tokenInputs{isTTY: false}})
	outcome, _, output := runCapture(root, "auth", "login")
	if outcome != UsageError {
		t.Fatalf("outcome = %v, want UsageError", outcome)
	}
	if code := ExitCode(outcome); code != codeUsageError {
		t.Fatalf("exit code = %d, want %d (usage)", code, codeUsageError)
	}
	if !strings.Contains(output, "no token to store") {
		t.Fatalf("operator output should report no token, got %q", output)
	}
}

func TestAuthLogin_ExtraPositional_UsageError(t *testing.T) {
	root := newTestRootWithAuth(&fakeSeam{home: t.TempDir(), start: t.TempDir(), inter: &fakeInteractor{}})
	outcome, _, _ := runCapture(root, "auth", "login", "tok1", "tok2")
	if outcome != UsageError {
		t.Fatalf("a second positional must be a usage error, got %v", outcome)
	}
}

func TestAuthLogin_WriteErrorMapsToInternalCode1(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root bypasses directory permission checks")
	}
	home, start := t.TempDir(), t.TempDir()
	if err := os.Chmod(home, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(home, 0o700) })
	root := newTestRootWithAuth(&fakeSeam{home: home, start: start, inter: &fakeInteractor{}, inputs: tokenInputs{arg: "x", argGiven: true}})
	outcome, _, output := runCapture(root, "auth", "login", "gf_new_token")
	if outcome != RuntimeError {
		t.Fatalf("outcome = %v, want RuntimeError", outcome)
	}
	if code := ExitCode(outcome); code != codeInternalError {
		t.Fatalf("exit code = %d, want %d (internal)", code, codeInternalError)
	}
	if !strings.Contains(output, "write error") {
		t.Fatalf("operator output should report a write error, got %q", output)
	}
}

// --- helpers ---

// newTestRootWithAuth assembles a root carrying only the auth group wired with
// the given seam, so command tests drive real cobra parsing + dispatch without
// the production seam reading the real os.
func newTestRootWithAuth(seam loginSeam) *cobra.Command {
	root := NewRootCommand()
	MustRegister(root, newAuthCommand(seam))
	return root
}

func assertNoFile(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected no file at %s, stat err = %v", path, err)
	}
}
