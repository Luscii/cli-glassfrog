package output

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/Luscii/cli-glassfrog/internal/rcfile"
)

// The 040 retrofit folded the 6-arg pre-fetched-source ResolveSelection core into
// the composing ResolveSelectionFromOS and removed it, so these cases now drive the
// single entry over the ADR-4 hermetic harness: the package getenv seam for the env
// rung and a temp-dir .glassfrogrc for the file rung — never the real environment or
// ~/.glassfrogrc. The flag rung is presence-based (cobra Changed()): a supplied flag
// passes flagPresent=true.

// stubOutputEnv points the package getenv seam at value for the env rung (EnvVarOutput
// only); an empty value models "unset" — so a test never reads the developer's real
// GLASSFROG_OUTPUT. Restored at test end.
func stubOutputEnv(t *testing.T, value string) {
	t.Helper()
	orig := getenv
	t.Cleanup(func() { getenv = orig })
	getenv = func(key string) string {
		if key == EnvVarOutput {
			return value
		}
		return ""
	}
}

// seedOutputRC writes a .glassfrogrc carrying output=value into dir and returns its
// path, so the file rung resolves over a real (temp) walk.
func seedOutputRC(t *testing.T, dir, value string) string {
	t.Helper()
	path := filepath.Join(dir, rcfile.FileName)
	if err := os.WriteFile(path, []byte("output="+value+"\n"), 0o600); err != nil {
		t.Fatalf("seed %s: %v", path, err)
	}
	return path
}

// TestResolveSelection_FlagFormatTokens confirms a reserved format token at the
// flag rung resolves to the matching built-in OutputFormat (any casing), never a
// template.
func TestResolveSelection_FlagFormatTokens(t *testing.T) {
	cases := map[string]OutputFormat{
		"full": FormatFull, "compact": FormatCompact, "json": FormatJSON, "yaml": FormatYAML,
		"JSON": FormatJSON, "  Yaml ": FormatYAML,
	}
	for flag, want := range cases {
		stubOutputEnv(t, "")
		dir := t.TempDir()
		sel, err := ResolveSelectionFromOS(flag, true, dir, dir)
		if err != nil {
			t.Errorf("%q: unexpected error %v", flag, err)
			continue
		}
		if _, ok := sel.AsTemplate(); ok {
			t.Errorf("%q: a format token must not resolve to a template", flag)
			continue
		}
		if sel.Format != want {
			t.Errorf("%q: resolved %v, want %v", flag, sel.Format, want)
		}
	}
}

// TestResolveSelection_FlagStdin confirms "stdin" at the flag rung resolves to a
// TemplateStdin ref.
func TestResolveSelection_FlagStdin(t *testing.T) {
	for _, flag := range []string{"stdin", "STDIN", " Stdin "} {
		stubOutputEnv(t, "")
		dir := t.TempDir()
		sel, err := ResolveSelectionFromOS(flag, true, dir, dir)
		if err != nil {
			t.Fatalf("%q: unexpected error %v", flag, err)
		}
		ref, ok := sel.AsTemplate()
		if !ok || ref.Kind != TemplateStdin {
			t.Errorf("%q: should resolve to TemplateStdin, got %+v (ok=%v)", flag, sel, ok)
		}
	}
}

// TestResolveSelection_FlagFile confirms any other non-empty flag value resolves to
// a TemplateFile ref carrying the raw path verbatim.
func TestResolveSelection_FlagFile(t *testing.T) {
	stubOutputEnv(t, "")
	dir := t.TempDir()
	sel, err := ResolveSelectionFromOS("./roles.tmpl", true, dir, dir)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	ref, ok := sel.AsTemplate()
	if !ok || ref.Kind != TemplateFile || ref.Path != "./roles.tmpl" {
		t.Errorf("a non-reserved value should be a TemplateFile with the path, got %+v (ok=%v)", sel, ok)
	}
}

// TestResolveSelection_FlagFileTrimsWhitespace confirms surrounding whitespace on a
// file-path flag value is trimmed, so `-o " ./x "` resolves to the same file as
// `-o ./x` — consistent with the reserved-token comparison (which also trims).
func TestResolveSelection_FlagFileTrimsWhitespace(t *testing.T) {
	stubOutputEnv(t, "")
	dir := t.TempDir()
	sel, err := ResolveSelectionFromOS("  ./roles.tmpl  ", true, dir, dir)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	ref, ok := sel.AsTemplate()
	if !ok || ref.Kind != TemplateFile || ref.Path != "./roles.tmpl" {
		t.Errorf("a whitespace-padded path should trim to %q, got %+v (ok=%v)", "./roles.tmpl", sel, ok)
	}
}

// TestResolveSelection_ReservedNameWins confirms a flag value equal to a reserved
// word (a format token or "stdin") never becomes a file, even though a same-named
// file might exist.
func TestResolveSelection_ReservedNameWins(t *testing.T) {
	for _, flag := range []string{"full", "json", "stdin"} {
		stubOutputEnv(t, "")
		dir := t.TempDir()
		sel, _ := ResolveSelectionFromOS(flag, true, dir, dir)
		if ref, ok := sel.AsTemplate(); ok && ref.Kind == TemplateFile {
			t.Errorf("%q: a reserved word must not become a file path", flag)
		}
	}
}

// TestResolveSelection_EmptyFlagSuppliedIsDegenerateTemplate confirms the 040
// presence change: a supplied empty/whitespace --output wins its rung by presence and
// classifies as a degenerate (empty-path) template file — NOT a token *FormatError,
// and NOT a fall-through. (The downstream read fails loud on the empty path; the env
// rung is never consulted.)
func TestResolveSelection_EmptyFlagSuppliedIsDegenerateTemplate(t *testing.T) {
	stubOutputEnv(t, "json") // a usable env value a (wrong) fall-through would pick up
	dir := t.TempDir()
	sel, err := ResolveSelectionFromOS("   ", true, dir, dir)
	if err != nil {
		t.Fatalf("a supplied empty flag should classify, not error at resolution: %v", err)
	}
	ref, ok := sel.AsTemplate()
	if !ok || ref.Kind != TemplateFile || ref.Path != "" {
		t.Errorf("a supplied empty flag should win as a degenerate empty TemplateFile, got %+v (ok=%v)", sel, ok)
	}
}

// TestResolveSelection_EnvNonTokenIsFormatError confirms a non-token value in the
// env rung returns a *FormatError naming the source — never a TemplateRef.
func TestResolveSelection_EnvNonTokenIsFormatError(t *testing.T) {
	stubOutputEnv(t, "./roles.tmpl")
	dir := t.TempDir()
	sel, err := ResolveSelectionFromOS("", false, dir, dir)
	var fe *FormatError
	if !errors.As(err, &fe) {
		t.Fatalf("a non-token env value should be a *FormatError, got %T (%v)", err, err)
	}
	if fe.Source != EnvVarOutput {
		t.Errorf("the error should name %s, named %q", EnvVarOutput, fe.Source)
	}
	if _, ok := sel.AsTemplate(); ok {
		t.Error("an env value must never resolve to a template")
	}
}

// TestResolveSelection_FileNonTokenIsFormatError confirms a non-token value in the
// .glassfrogrc output key returns a *FormatError naming the file — never a template.
func TestResolveSelection_FileNonTokenIsFormatError(t *testing.T) {
	stubOutputEnv(t, "")
	dir := t.TempDir()
	path := seedOutputRC(t, dir, "./roles.tmpl")
	sel, err := ResolveSelectionFromOS("", false, dir, dir)
	var fe *FormatError
	if !errors.As(err, &fe) {
		t.Fatalf("a non-token file value should be a *FormatError, got %T (%v)", err, err)
	}
	if fe.Source != path {
		t.Errorf("the error should name the file %q, named %q", path, fe.Source)
	}
	if _, ok := sel.AsTemplate(); ok {
		t.Error("a config value must never resolve to a template")
	}
}

// TestResolveSelection_PrecedenceUnchanged confirms the flag wins over env and file,
// and all-absent yields Full.
func TestResolveSelection_PrecedenceUnchanged(t *testing.T) {
	// Flag wins over a (valid) env and file.
	stubOutputEnv(t, "yaml")
	dir := t.TempDir()
	seedOutputRC(t, dir, "compact")
	sel, err := ResolveSelectionFromOS("json", true, dir, dir)
	if err != nil || sel.Format != FormatJSON {
		t.Errorf("the flag should win, resolved %+v (err %v)", sel, err)
	}

	// All absent → default Full.
	stubOutputEnv(t, "")
	emptyDir := t.TempDir()
	sel, err = ResolveSelectionFromOS("", false, emptyDir, emptyDir)
	if err != nil || sel.Format != FormatFull {
		t.Errorf("all-absent should yield Full, resolved %+v (err %v)", sel, err)
	}
}

// TestResolveSelection_FileReadErrorSurfaces confirms an unreadable .glassfrogrc at
// the file rung surfaces loudly (no fall-through to the default). A directory at the
// .glassfrogrc path makes the read fail deterministically across platforms.
func TestResolveSelection_FileReadErrorSurfaces(t *testing.T) {
	stubOutputEnv(t, "")
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, rcfile.FileName), 0o755); err != nil {
		t.Fatal(err)
	}
	_, err := ResolveSelectionFromOS("", false, dir, dir)
	var re *rcfile.ReadError
	if !errors.As(err, &re) {
		t.Fatalf("an rcfile read error should surface, got %T (%v)", err, err)
	}
}
