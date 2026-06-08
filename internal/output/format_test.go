package output

import (
	"errors"
	"testing"

	"github.com/Luscii/cli-glassfrog/internal/rcfile"
)

// --- T001: ParseFormat / IsStructured / MachineFormat / constants ----------

func TestParseFormat_AllTokensAndCasing(t *testing.T) {
	cases := []struct {
		in   string
		want OutputFormat
	}{
		{"full", FormatFull},
		{"compact", FormatCompact},
		{"json", FormatJSON},
		{"yaml", FormatYAML},
		// Mixed casing all select the same format (case-insensitive over the four).
		{"JSON", FormatJSON},
		{"Json", FormatJSON},
		{"jSON", FormatJSON},
		{"FULL", FormatFull},
		{"Compact", FormatCompact},
		{"YAML", FormatYAML},
		// Surrounding whitespace is ignored.
		{"  json  ", FormatJSON},
	}
	for _, tc := range cases {
		got, err := ParseFormat(tc.in)
		if err != nil {
			t.Errorf("ParseFormat(%q) unexpected error: %v", tc.in, err)
			continue
		}
		if got != tc.want {
			t.Errorf("ParseFormat(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

func TestParseFormat_InvalidReturnsError(t *testing.T) {
	for _, in := range []string{"xml", "", "   ", "jsonl", "ya ml", "JSO"} {
		if _, err := ParseFormat(in); err == nil {
			t.Errorf("ParseFormat(%q) expected a non-nil error, got nil", in)
		}
	}
}

func TestIsStructured(t *testing.T) {
	cases := map[OutputFormat]bool{
		FormatFull:    false,
		FormatCompact: false,
		FormatJSON:    true,
		FormatYAML:    true,
	}
	for f, want := range cases {
		if got := f.IsStructured(); got != want {
			t.Errorf("%v.IsStructured() = %v, want %v", f, got, want)
		}
	}
}

func TestMachineFormat(t *testing.T) {
	cases := []struct {
		in     OutputFormat
		want   Format
		wantOK bool
	}{
		{FormatJSON, JSON, true},
		{FormatYAML, YAML, true},
		{FormatFull, 0, false},
		{FormatCompact, 0, false},
	}
	for _, tc := range cases {
		got, ok := tc.in.MachineFormat()
		if ok != tc.wantOK {
			t.Errorf("%v.MachineFormat() ok = %v, want %v", tc.in, ok, tc.wantOK)
		}
		if ok && got != tc.want {
			t.Errorf("%v.MachineFormat() = %v, want %v", tc.in, got, tc.want)
		}
	}
}

func TestDefaultFormatIsFull(t *testing.T) {
	if DefaultFormat != FormatFull {
		t.Fatalf("DefaultFormat = %v, want FormatFull", DefaultFormat)
	}
	// The zero value of OutputFormat is FormatFull, so a bare default is Full.
	var zero OutputFormat
	if zero != FormatFull {
		t.Fatalf("zero OutputFormat = %v, want FormatFull", zero)
	}
}

func TestSelectionConstants(t *testing.T) {
	if FlagOutput != "output" {
		t.Errorf("FlagOutput = %q, want %q", FlagOutput, "output")
	}
	if EnvVarOutput != "GLASSFROG_OUTPUT" {
		t.Errorf("EnvVarOutput = %q, want %q", EnvVarOutput, "GLASSFROG_OUTPUT")
	}
	if outputKey != "output" {
		t.Errorf("outputKey = %q, want %q", outputKey, "output")
	}
}

// --- T002: ResolveFormat precedence + present-but-invalid + rcfile errors --

func TestResolveFormat_Precedence(t *testing.T) {
	// flag wins over env and file.
	if got, err := ResolveFormat("json", "yaml", "compact", "/work/.glassfrogrc", true, nil); err != nil || got != FormatJSON {
		t.Errorf("flag precedence: got %v, err %v; want FormatJSON, nil", got, err)
	}
	// env wins over file when the flag is absent.
	if got, err := ResolveFormat("", "yaml", "compact", "/work/.glassfrogrc", true, nil); err != nil || got != FormatYAML {
		t.Errorf("env precedence: got %v, err %v; want FormatYAML, nil", got, err)
	}
	// the file supplies the format when flag and env are absent.
	if got, err := ResolveFormat("", "", "compact", "/work/.glassfrogrc", true, nil); err != nil || got != FormatCompact {
		t.Errorf("file precedence: got %v, err %v; want FormatCompact, nil", got, err)
	}
	// all absent → the built-in default Full.
	if got, err := ResolveFormat("", "", "", "", false, nil); err != nil || got != FormatFull {
		t.Errorf("default: got %v, err %v; want FormatFull, nil", got, err)
	}
	// a whitespace-only flag/env value is treated as absent and falls through.
	if got, err := ResolveFormat("   ", "  ", "compact", "/work/.glassfrogrc", true, nil); err != nil || got != FormatCompact {
		t.Errorf("blank flag/env fall-through: got %v, err %v; want FormatCompact, nil", got, err)
	}
}

func TestResolveFormat_PresentButInvalidNamesSource(t *testing.T) {
	cases := []struct {
		name                      string
		flag, env, file, filePath string
		fileFound                 bool
		wantSource, wantValue     string
	}{
		{"flag", "xml", "json", "json", "/work/.glassfrogrc", true, "--output", "xml"},
		{"env", "", "xml", "json", "/work/.glassfrogrc", true, "GLASSFROG_OUTPUT", "xml"},
		{"file", "", "", "xml", "/work/.glassfrogrc", true, "/work/.glassfrogrc", "xml"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ResolveFormat(tc.flag, tc.env, tc.file, tc.filePath, tc.fileFound, nil)
			var fe *FormatError
			if !errors.As(err, &fe) {
				t.Fatalf("expected *FormatError, got %v", err)
			}
			if fe.Source != tc.wantSource {
				t.Errorf("Source = %q, want %q", fe.Source, tc.wantSource)
			}
			if fe.Value != tc.wantValue {
				t.Errorf("Value = %q, want %q", fe.Value, tc.wantValue)
			}
			// The message names the value and never falls through to a lower rung's
			// (valid) value.
			if msg := fe.Error(); msg == "" {
				t.Error("FormatError.Error() should be non-empty")
			}
		})
	}
}

func TestResolveFormat_InvalidLowerSourceDoesNotFallThrough(t *testing.T) {
	// A present-but-invalid env value must surface loudly even though the file holds
	// a valid value below it.
	_, err := ResolveFormat("", "xml", "json", "/work/.glassfrogrc", true, nil)
	var fe *FormatError
	if !errors.As(err, &fe) || fe.Source != "GLASSFROG_OUTPUT" {
		t.Fatalf("expected a *FormatError naming GLASSFROG_OUTPUT, got %v", err)
	}
}

func TestResolveFormat_SurfacesRcfileError(t *testing.T) {
	readErr := &rcfile.ReadError{Path: "/work/.glassfrogrc", Err: errors.New("permission denied")}
	_, err := ResolveFormat("", "", "", "", false, readErr)
	if !errors.Is(err, readErr) {
		t.Fatalf("expected the rcfile read error to surface, got %v", err)
	}

	fmtErr := &rcfile.FormatError{Path: "/work/.glassfrogrc"}
	_, err = ResolveFormat("", "", "", "", false, fmtErr)
	var rcFmt *rcfile.FormatError
	if !errors.As(err, &rcFmt) {
		t.Fatalf("expected the rcfile format error to surface, got %v", err)
	}
}
