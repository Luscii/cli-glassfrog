package rcfile

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeFile creates a .glassfrogrc under t.TempDir() holding content and returns
// its path. Confined to the per-test temp tree.
func writeFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), FileName)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("seeding fixture: %v", err)
	}
	return path
}

func TestReadValue_Present(t *testing.T) {
	path := writeFile(t, "# glassfrog config\ntoken=gf_x\nbase_url=https://glassfrog.com/api/v5\n")

	for _, c := range []struct{ key, want string }{
		{"token", "gf_x"},
		{"base_url", "https://glassfrog.com/api/v5"},
	} {
		value, found, err := ReadValue(path, c.key)
		if err != nil {
			t.Fatalf("ReadValue(%q): %v", c.key, err)
		}
		if !found || value != c.want {
			t.Errorf("ReadValue(%q) = (%q, %v), want (%q, true)", c.key, value, found, c.want)
		}
	}
}

func TestReadValue_OnlyReturnsRequestedKey(t *testing.T) {
	// Secret hygiene, now structural: asking for base_url never returns the token,
	// and vice versa — each ReadValue surfaces exactly one key's value.
	secret := "gf_super_secret_token"
	path := writeFile(t, "token="+secret+"\nbase_url=https://glassfrog.com/api/v5\n")

	value, found, err := ReadValue(path, "base_url")
	if err != nil || !found {
		t.Fatalf("ReadValue(base_url) = (%q, %v, %v)", value, found, err)
	}
	if strings.Contains(value, secret) {
		t.Errorf("the token leaked into the base_url read: %q", value)
	}
}

func TestReadValue_TrimsKeyAndValue(t *testing.T) {
	// Surrounding whitespace around key and value is trimmed; the split is on the
	// first '=' so a value may itself contain '='.
	path := writeFile(t, "  token  =  gf_a=b  \n")

	value, found, err := ReadValue(path, "token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !found || value != "gf_a=b" {
		t.Errorf("got (%q, %v), want (%q, true)", value, found, "gf_a=b")
	}
}

func TestReadValue_MissingKeyIsNotFound(t *testing.T) {
	path := writeFile(t, "# nothing useful here\nother=value\n")

	value, found, err := ReadValue(path, "token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found || value != "" {
		t.Errorf("got (%q, %v), want (\"\", false) for an absent key", value, found)
	}
}

func TestReadValue_WhitespaceOnlyValueIsNotFound(t *testing.T) {
	path := writeFile(t, "token=   \n")

	_, found, err := ReadValue(path, "token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found {
		t.Errorf("found = true, want false for a whitespace-only value")
	}
}

func TestReadValue_CommentsAndBlanksIgnored(t *testing.T) {
	content := "\n   \n   # indented comment\nunknown=ignored\n\ntoken=gf_y\n"
	path := writeFile(t, content)

	value, found, err := ReadValue(path, "token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !found || value != "gf_y" {
		t.Errorf("got (%q, %v), want (%q, true)", value, found, "gf_y")
	}
}

func TestReadValue_LastOccurrenceWins(t *testing.T) {
	path := writeFile(t, "token=first\ntoken=second\n")

	value, _, err := ReadValue(path, "token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if value != "second" {
		t.Errorf("value = %q, want the last occurrence %q", value, "second")
	}
}

func TestReadValue_MalformedLineIsFormatError(t *testing.T) {
	secret := "gf_super_secret"
	path := writeFile(t, "token="+secret+"\nthis line has no equals sign\n")

	_, _, err := ReadValue(path, "token")
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
	// The error names only the path, never any value on another line.
	if strings.Contains(err.Error(), secret) {
		t.Errorf("format error leaked a value: %q", err.Error())
	}
}

func TestReadValue_MissingFileIsReadError(t *testing.T) {
	path := filepath.Join(t.TempDir(), FileName)

	_, _, err := ReadValue(path, "token")
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

func TestSettings_ValueUsabilityRule(t *testing.T) {
	s := Settings{"a": "x", "blank": "", "ws": "   \t "}
	if v, ok := s.Value("a"); !ok || v != "x" {
		t.Errorf("Value(a) = (%q, %v), want (x, true)", v, ok)
	}
	if _, ok := s.Value("blank"); ok {
		t.Error("Value(blank) found = true, want false for an empty value")
	}
	// A whitespace-only value is treated as absent even when Settings is built
	// directly (not via Parse, which would have trimmed it) — the method enforces
	// its own "usable after trimming" contract.
	if _, ok := s.Value("ws"); ok {
		t.Error("Value(ws) found = true, want false for a whitespace-only value")
	}
	if _, ok := s.Value("absent"); ok {
		t.Error("Value(absent) found = true, want false for a missing key")
	}
}
