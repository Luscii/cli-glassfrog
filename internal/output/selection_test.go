package output

import (
	"errors"
	"testing"

	"github.com/Luscii/cli-glassfrog/internal/rcfile"
)

// TestResolveSelection_FlagFormatTokens confirms a reserved format token at the
// flag rung resolves to the matching built-in OutputFormat (any casing), never a
// template.
func TestResolveSelection_FlagFormatTokens(t *testing.T) {
	cases := map[string]OutputFormat{
		"full": FormatFull, "compact": FormatCompact, "json": FormatJSON, "yaml": FormatYAML,
		"JSON": FormatJSON, "  Yaml ": FormatYAML,
	}
	for flag, want := range cases {
		sel, err := ResolveSelection(flag, "", "", "", false, nil)
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
		sel, err := ResolveSelection(flag, "", "", "", false, nil)
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
	sel, err := ResolveSelection("./roles.tmpl", "", "", "", false, nil)
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
// `-o ./x` — consistent with the flag-rung presence check and the reserved-token
// comparison (both trim).
func TestResolveSelection_FlagFileTrimsWhitespace(t *testing.T) {
	sel, err := ResolveSelection("  ./roles.tmpl  ", "", "", "", false, nil)
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
		sel, _ := ResolveSelection(flag, "", "", "", false, nil)
		if ref, ok := sel.AsTemplate(); ok && ref.Kind == TemplateFile {
			t.Errorf("%q: a reserved word must not become a file path", flag)
		}
	}
}

// TestResolveSelection_EnvNonTokenIsFormatError confirms a non-token value in the
// env rung still returns a *FormatError naming the source — never a TemplateRef.
func TestResolveSelection_EnvNonTokenIsFormatError(t *testing.T) {
	sel, err := ResolveSelection("", "./roles.tmpl", "", "", false, nil)
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
	sel, err := ResolveSelection("", "", "./roles.tmpl", "/work/.glassfrogrc", true, nil)
	var fe *FormatError
	if !errors.As(err, &fe) {
		t.Fatalf("a non-token file value should be a *FormatError, got %T (%v)", err, err)
	}
	if fe.Source != "/work/.glassfrogrc" {
		t.Errorf("the error should name the file, named %q", fe.Source)
	}
	if _, ok := sel.AsTemplate(); ok {
		t.Error("a config value must never resolve to a template")
	}
}

// TestResolveSelection_PrecedenceUnchanged confirms the flag wins over env and file,
// and all-absent yields Full.
func TestResolveSelection_PrecedenceUnchanged(t *testing.T) {
	// Flag wins over a (valid) env and file.
	sel, err := ResolveSelection("json", "yaml", "compact", "/work/.glassfrogrc", true, nil)
	if err != nil || sel.Format != FormatJSON {
		t.Errorf("the flag should win, resolved %+v (err %v)", sel, err)
	}
	// All absent → default Full.
	sel, err = ResolveSelection("", "", "", "", false, nil)
	if err != nil || sel.Format != FormatFull {
		t.Errorf("all-absent should yield Full, resolved %+v (err %v)", sel, err)
	}
}

// TestResolveSelection_FileReadErrorSurfaces confirms an unreadable .glassfrogrc at
// the file rung surfaces loudly (no fall-through), as under ResolveFormat.
func TestResolveSelection_FileReadErrorSurfaces(t *testing.T) {
	readErr := &rcfile.ReadError{Path: "/work/.glassfrogrc", Err: errors.New("permission denied")}
	_, err := ResolveSelection("", "", "", "", false, readErr)
	if !errors.Is(err, readErr) {
		t.Fatalf("an rcfile read error should surface, got %v", err)
	}
}
