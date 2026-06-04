package auth

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// Writer tests (Credential Storage 006). The shared reader's own behaviour is
// covered by credentials_test.go (Credential Discovery 005); here we exercise
// WriteCredentials and the write→read round-trip that pins the shared format.

// seedCredFile drops content at path, failing the test on error. Confined to
// the test's temp dir by callers — never the real home. (Named to avoid
// colliding with credentials_test.go's writeFile helper in this package.)
func seedCredFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("seeding %s: %v", path, err)
	}
}

func readBack(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	return string(data)
}

// --- WriteCredentials: new-file creation ---

func TestWriteCredentials_AbsentPath_CreatesFileWithToken(t *testing.T) {
	path := filepath.Join(t.TempDir(), CredentialsFileName)

	if err := WriteCredentials(path, "gf_new_token"); err != nil {
		t.Fatalf("WriteCredentials on absent path: %v", err)
	}

	got := readBack(t, path)
	if !strings.Contains(got, "token=gf_new_token") {
		t.Fatalf("written file missing token line, got:\n%s", got)
	}
}

func TestWriteCredentials_AbsentPath_OwnerOnlyPermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX permission bits not meaningful on Windows")
	}
	path := filepath.Join(t.TempDir(), CredentialsFileName)

	if err := WriteCredentials(path, "gf_new_token"); err != nil {
		t.Fatalf("WriteCredentials: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Fatalf("permissions = %o, want 0600 (owner read/write only)", perm)
	}
}

// --- WriteCredentials: line-preserving merge ---

func TestWriteCredentials_PresentFile_ReplacesOnlyTokenPreservesOthers(t *testing.T) {
	path := filepath.Join(t.TempDir(), CredentialsFileName)
	seedCredFile(t, path, "# glassfrog credentials\ntoken=gf_old_token\nother=keep_me\n")

	if err := WriteCredentials(path, "gf_new_token"); err != nil {
		t.Fatalf("WriteCredentials over existing file: %v", err)
	}

	got := readBack(t, path)
	if !strings.Contains(got, "token=gf_new_token") {
		t.Fatalf("token value not replaced, got:\n%s", got)
	}
	if strings.Contains(got, "gf_old_token") {
		t.Fatalf("old token still present, got:\n%s", got)
	}
	if !strings.Contains(got, "# glassfrog credentials") {
		t.Fatalf("comment not preserved, got:\n%s", got)
	}
	if !strings.Contains(got, "other=keep_me") {
		t.Fatalf("unrelated entry not preserved, got:\n%s", got)
	}
}

func TestWriteCredentials_PresentFileNoToken_AppendsTokenPreservesOthers(t *testing.T) {
	path := filepath.Join(t.TempDir(), CredentialsFileName)
	seedCredFile(t, path, "# header\nother=keep_me\n")

	if err := WriteCredentials(path, "gf_new_token"); err != nil {
		t.Fatalf("WriteCredentials appending token: %v", err)
	}

	got := readBack(t, path)
	if !strings.Contains(got, "token=gf_new_token") {
		t.Fatalf("token line not appended, got:\n%s", got)
	}
	if !strings.Contains(got, "other=keep_me") || !strings.Contains(got, "# header") {
		t.Fatalf("existing lines not preserved, got:\n%s", got)
	}
}

// A file with multiple token= lines collapses to a single one holding the new
// value, with no stale secret left behind — matching the reader's last-wins
// rule so the written token is the effective one.
func TestWriteCredentials_MultipleTokenLines_CollapsedToOne(t *testing.T) {
	path := filepath.Join(t.TempDir(), CredentialsFileName)
	seedCredFile(t, path, "token=gf_first\nother=keep_me\ntoken=gf_second\n")

	if err := WriteCredentials(path, "gf_new_token"); err != nil {
		t.Fatalf("WriteCredentials: %v", err)
	}

	got := readBack(t, path)
	if n := strings.Count(got, "token="); n != 1 {
		t.Fatalf("expected exactly one token= line, got %d:\n%s", n, got)
	}
	if !strings.Contains(got, "token=gf_new_token") {
		t.Fatalf("new token not written, got:\n%s", got)
	}
	if strings.Contains(got, "gf_first") || strings.Contains(got, "gf_second") {
		t.Fatalf("a stale token value remained, got:\n%s", got)
	}
	if !strings.Contains(got, "other=keep_me") {
		t.Fatalf("unrelated entry not preserved, got:\n%s", got)
	}
	// The reader (last-wins) returns the value just written.
	tok, found, err := ReadCredentialsFile(path)
	if err != nil || !found || tok != "gf_new_token" {
		t.Fatalf("reader after collapse: token=%q found=%v err=%v", tok, found, err)
	}
}

// --- WriteCredentials: malformed existing file → format error, no write ---

func TestWriteCredentials_MalformedExisting_FormatErrorNoWrite(t *testing.T) {
	path := filepath.Join(t.TempDir(), CredentialsFileName)
	original := "this line has no equals sign\n"
	seedCredFile(t, path, original)

	err := WriteCredentials(path, "gf_new_token")
	if err == nil {
		t.Fatal("expected a format error for a malformed existing file, got nil")
	}
	var fe *FormatError
	if !errors.As(err, &fe) {
		t.Fatalf("expected *FormatError, got %T: %v", err, err)
	}
	if fe.Path != path {
		t.Fatalf("FormatError should name the path %q, named %q", path, fe.Path)
	}
	if strings.Contains(err.Error(), "gf_new_token") {
		t.Fatalf("error must not contain the token value: %v", err)
	}
	if got := readBack(t, path); got != original {
		t.Fatalf("malformed file must not be overwritten; got:\n%s", got)
	}
}

// --- WriteCredentials: a multi-line token is rejected at the API boundary ---

func TestWriteCredentials_MultiLineToken_RejectedNoWrite(t *testing.T) {
	for _, tc := range []struct {
		name  string
		token string
	}{
		{"newline", "gf_tok\nevil=injected"},
		{"carriage-return", "gf_tok\revil=injected"},
		{"crlf", "gf_tok\r\nevil=injected"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), CredentialsFileName)

			err := WriteCredentials(path, tc.token)
			if !errors.Is(err, ErrTokenNotSingleLine) {
				t.Fatalf("expected ErrTokenNotSingleLine, got %T: %v", err, err)
			}
			if strings.Contains(err.Error(), "gf_tok") || strings.Contains(err.Error(), "injected") {
				t.Fatalf("error must not contain the token value: %v", err)
			}
			if _, statErr := os.Stat(path); !errors.Is(statErr, os.ErrNotExist) {
				t.Fatalf("a rejected token must not create the file, stat err = %v", statErr)
			}
		})
	}
}

// --- WriteCredentials: failed write leaves filesystem unchanged, no temp left ---

func TestWriteCredentials_UnwritableDir_WriteErrorOriginalUnchanged(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("directory write-permission semantics differ on Windows")
	}
	if os.Geteuid() == 0 {
		t.Skip("running as root bypasses directory permission checks")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, CredentialsFileName)
	if err := os.Chmod(dir, 0o500); err != nil {
		t.Fatalf("chmod dir read-only: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o700) })

	err := WriteCredentials(path, "gf_new_token")
	if err == nil {
		t.Fatal("expected a write error for an unwritable directory, got nil")
	}
	var we *WriteError
	if !errors.As(err, &we) {
		t.Fatalf("expected *WriteError, got %T: %v", err, err)
	}
	if strings.Contains(err.Error(), "gf_new_token") {
		t.Fatalf("error must not contain the token value: %v", err)
	}
	if _, statErr := os.Stat(path); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("target should not exist after a failed write, stat err = %v", statErr)
	}
	// No temp file left behind in the (now restored-readable) directory.
	_ = os.Chmod(dir, 0o700)
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("readdir: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("a temp file was left behind: %v", entries)
	}
}

// --- Round-trip: a written token is read back unchanged through the shared reader ---

func TestWriteCredentials_RoundTripsThroughReader(t *testing.T) {
	path := filepath.Join(t.TempDir(), CredentialsFileName)

	if err := WriteCredentials(path, "gf_round_trip"); err != nil {
		t.Fatalf("WriteCredentials: %v", err)
	}

	token, found, err := ReadCredentialsFile(path)
	if err != nil {
		t.Fatalf("ReadCredentialsFile: %v", err)
	}
	if !found {
		t.Fatal("reader reports no token after a write")
	}
	if token != "gf_round_trip" {
		t.Fatalf("round-trip token = %q, want %q", token, "gf_round_trip")
	}
}

func TestWriteCredentials_RoundTripsAfterMerge(t *testing.T) {
	path := filepath.Join(t.TempDir(), CredentialsFileName)
	seedCredFile(t, path, "# comment\nother=keep\ntoken=gf_old\n")

	if err := WriteCredentials(path, "gf_merged"); err != nil {
		t.Fatalf("WriteCredentials: %v", err)
	}
	token, found, err := ReadCredentialsFile(path)
	if err != nil || !found {
		t.Fatalf("ReadCredentialsFile after merge: token=%q found=%v err=%v", token, found, err)
	}
	if token != "gf_merged" {
		t.Fatalf("merged round-trip token = %q, want %q", token, "gf_merged")
	}
}
