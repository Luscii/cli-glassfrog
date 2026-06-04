package auth

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeFile creates a file under t.TempDir() holding content and returns its
// path. Confined to the per-test temp tree — the reader is never pointed at the
// developer's real ~/.glassfrogrc.
func writeFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), credentialsFileName)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("seeding fixture: %v", err)
	}
	return path
}

func TestReadCredentialsFile_ValidToken(t *testing.T) {
	path := writeFile(t, "# glassfrog credentials\ntoken=gf_x\n")

	token, found, err := readCredentialsFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !found {
		t.Fatalf("found = false, want true for a file holding token=gf_x")
	}
	if token != "gf_x" {
		t.Errorf("token = %q, want %q", token, "gf_x")
	}
}

func TestReadCredentialsFile_TrimsKeyAndValue(t *testing.T) {
	// Surrounding whitespace around key and value is trimmed; the split is on
	// the first '=' so a value may itself contain '='.
	path := writeFile(t, "  token  =  gf_a=b  \n")

	token, found, err := readCredentialsFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !found || token != "gf_a=b" {
		t.Errorf("got (%q, %v), want (%q, true)", token, found, "gf_a=b")
	}
}

func TestReadCredentialsFile_NoTokenKey(t *testing.T) {
	// A file that parses cleanly but carries no token key is a normal "not
	// found" — no error.
	path := writeFile(t, "# nothing useful here\nother=value\n")

	token, found, err := readCredentialsFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found {
		t.Errorf("found = true, want false for a tokenless file")
	}
	if token != "" {
		t.Errorf("token = %q, want empty", token)
	}
}

func TestReadCredentialsFile_WhitespaceOnlyValueIsNoToken(t *testing.T) {
	path := writeFile(t, "token=   \n")

	_, found, err := readCredentialsFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found {
		t.Errorf("found = true, want false for a whitespace-only token value")
	}
}

func TestReadCredentialsFile_CommentsAndBlanksIgnored(t *testing.T) {
	// Leading-whitespace comments and blank lines are skipped; unknown keys are
	// ignored without error.
	content := "\n   \n   # indented comment\nunknown=ignored\n\ntoken=gf_y\n"
	path := writeFile(t, content)

	token, found, err := readCredentialsFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !found || token != "gf_y" {
		t.Errorf("got (%q, %v), want (%q, true)", token, found, "gf_y")
	}
}

func TestReadCredentialsFile_MalformedLineIsFormatError(t *testing.T) {
	secret := "gf_super_secret"
	path := writeFile(t, "token="+secret+"\nthis line has no equals sign\n")

	_, _, err := readCredentialsFile(path)
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
	// Secret hygiene: the token value must never appear in the error text.
	if strings.Contains(err.Error(), secret) {
		t.Errorf("format error leaked the token value: %q", err.Error())
	}
}

func TestReadCredentialsFile_MissingFileIsReadError(t *testing.T) {
	// A missing file surfaces a typed ReadError that unwraps to fs.ErrNotExist,
	// so the resolver can distinguish "skip" from a genuine read failure.
	path := filepath.Join(t.TempDir(), credentialsFileName)

	_, _, err := readCredentialsFile(path)
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
