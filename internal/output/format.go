package output

import (
	"fmt"
	"os"
	"strings"

	"github.com/Luscii/cli-glassfrog/internal/rcfile"
)

// OutputFormat is the four-valued *selection* vocabulary Output Format Selection
// (020) resolves the --output flag, the GLASSFROG_OUTPUT env var, and the
// .glassfrogrc output key to. It is distinct from Format (the machine-encoder enum
// JSON/YAML this package already owns for 018): OutputFormat additionally carries
// the human full/compact members (019), and is what selection produces before the
// dispatch routes each member to its renderer. internal/cli maps the human members
// onto internal/render's format names (ADR-3), so this package never imports
// internal/render.
//
// FormatFull is the zero value, so a bare OutputFormat{} (and the built-in default)
// is Full — the standing CLI projection (019).
type OutputFormat int

const (
	// FormatFull is the labelled human projection (019) — the built-in default and
	// the standing output preserved when no source selects a format.
	FormatFull OutputFormat = iota
	// FormatCompact is the denser one-line-per-record human rendering (019), made
	// reachable from the command line for the first time by 020.
	FormatCompact
	// FormatJSON selects 018's JSON encoder (Format.JSON).
	FormatJSON
	// FormatYAML selects 018's YAML encoder (Format.YAML).
	FormatYAML
)

// DefaultFormat is the built-in default at the end of the precedence chain — Full,
// preserving 019's standing projection. Selection always yields a format, so the
// chain never produces a "no format" outcome.
const DefaultFormat = FormatFull

// Selection-source constants, centralized here as the single source of truth so the
// registered flag, the resolver rungs, and the .glassfrogrc reader cannot drift —
// mirroring apiclient.FlagBaseURL / EnvVarBaseURL / baseURLKey (008).
const (
	// FlagOutput is the --output long flag name (precedence rung 1, highest). The
	// -o short alias is registered at the flag site (internal/cli/root.go).
	FlagOutput = "output"

	// EnvVarOutput is the environment variable carrying the format (rung 2).
	EnvVarOutput = "GLASSFROG_OUTPUT"

	// outputKey is the .glassfrogrc key carrying the format (rung 3), read through
	// the shared internal/rcfile walk. It is the third key after token (005) and
	// base_url (008), read independently of both.
	outputKey = "output"
)

// supportedFormats is the operator-facing list of the four valid tokens, named in
// every parse/resolution error so a rejection always says what is accepted.
const supportedFormats = "full, compact, json, yaml"

// ParseFormat matches s against exactly the four tokens (full / compact / json /
// yaml), case-insensitively and ignoring surrounding whitespace, so JSON, Json, and
// jSON all select FormatJSON. Any other value — including the empty string — returns
// a non-nil error naming the offending value and the supported set; callers must
// check the error before using the returned format (the zero value is FormatFull).
func ParseFormat(s string) (OutputFormat, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "full":
		return FormatFull, nil
	case "compact":
		return FormatCompact, nil
	case "json":
		return FormatJSON, nil
	case "yaml":
		return FormatYAML, nil
	default:
		return DefaultFormat, fmt.Errorf("unsupported output format %q — supported: %s", s, supportedFormats)
	}
}

// String returns the lowercase token for f, used in messages and tests.
func (f OutputFormat) String() string {
	switch f {
	case FormatFull:
		return "full"
	case FormatCompact:
		return "compact"
	case FormatJSON:
		return "json"
	case FormatYAML:
		return "yaml"
	default:
		return "unknown"
	}
}

// IsStructured reports whether f routes to the machine branch (internal/output's
// JSON/YAML encoders) rather than the human branch (internal/render's full/compact
// templates). It is the dispatch's decode-target switch (018 ADR-2): a structured
// format captures the verbatim 2xx bytes, a human format decodes the typed struct.
func (f OutputFormat) IsStructured() bool {
	return f == FormatJSON || f == FormatYAML
}

// MachineFormat maps a structured OutputFormat onto 018's Format (FormatJSON →
// JSON, FormatYAML → YAML), returning ok=false for the human formats. It keeps the
// json/yaml mapping inside internal/output; the human → internal/render mapping
// lives in internal/cli (ADR-3), so neither renderer package imports the other.
func (f OutputFormat) MachineFormat() (Format, bool) {
	switch f {
	case FormatJSON:
		return JSON, true
	case FormatYAML:
		return YAML, true
	default:
		return 0, false
	}
}

// FormatError reports that a non-empty selection source carried a value that is not
// one of the four valid tokens. Source names where it came from (--output, the env
// var, or the .glassfrogrc path) and Value is the offending value — both safe to
// display: the token is never in scope on a format-resolution path. internal/cli
// maps it to UsageError(2) (ADR-4), symmetric with a malformed --base-url.
type FormatError struct {
	// Source is the operator-facing label of the rung that supplied Value:
	// "--output", "GLASSFROG_OUTPUT", or the resolved .glassfrogrc path.
	Source string
	// Value is the offending value, verbatim.
	Value string
}

func (e *FormatError) Error() string {
	return fmt.Sprintf("unsupported output value %q from %s — supported: %s", e.Value, e.Source, supportedFormats)
}

// getenv is the production seam for the env rung — a package var so a test can
// exercise ResolveFormatFromOS over a controlled environment, never the developer's
// real one. The pure core (ResolveFormat) takes the env value as an argument and
// touches no global, so most tests bypass this entirely (the 005 inject split).
var getenv = os.Getenv

// ResolveFormatFromOS is the thin production seam over ResolveFormat: it binds the
// real GLASSFROG_OUTPUT lookup and the internal/rcfile nearest-wins walk (over the
// output key) to the pure core's injected sources, given the --output flag value
// and the start/home directories the caller supplies (the 005 inject-roots split).
// Tests call ResolveFormat directly with hand-built sources, so no suite reads the
// real ~/.glassfrogrc.
func ResolveFormatFromOS(flagValue, startDir, homeDir string) (OutputFormat, error) {
	envValue := getenv(EnvVarOutput)
	fileValue, filePath, fileFound, fileErr := rcfile.Resolve(startDir, homeDir, outputKey)
	return ResolveFormat(flagValue, envValue, fileValue, filePath, fileFound, fileErr)
}

// ResolveFormat is the pure precedence core (ADR-1), mirroring apiclient.baseurl:
// it consults the --output flag, then GLASSFROG_OUTPUT, then the .glassfrogrc output
// key, then the built-in default, over sources its caller has already fetched.
//
//   - flagValue / envValue are the rung-1 / rung-2 raw values; a whitespace-only
//     value is treated as absent and falls through (the base-URL convention).
//   - fileValue / filePath / fileFound / fileErr are the result of the rcfile walk
//     over the output key: fileErr (an unreadable/unparseable .glassfrogrc) fails
//     loud naming the file with NO fall-through; fileFound carries a usable value.
//
// The first source that yields a value wins. A present-but-invalid value at ANY
// source returns a *FormatError naming that source and value — never a fall-through
// to a lower rung (the 008 present-but-malformed rule). When every source is absent
// the built-in DefaultFormat (Full) backstops the chain, so resolution always
// yields a format unless a source erred. The returned format is meaningful only
// when err == nil; on error the zero value (FormatFull) is returned as a
// placeholder.
func ResolveFormat(flagValue, envValue, fileValue, filePath string, fileFound bool, fileErr error) (OutputFormat, error) {
	// Rung 1: the --output flag. A non-empty value short-circuits all else.
	if strings.TrimSpace(flagValue) != "" {
		f, err := ParseFormat(flagValue)
		if err != nil {
			return DefaultFormat, &FormatError{Source: "--" + FlagOutput, Value: flagValue}
		}
		return f, nil
	}

	// Rung 2: GLASSFROG_OUTPUT. Same present/absent/invalid rules, uniformly.
	if strings.TrimSpace(envValue) != "" {
		f, err := ParseFormat(envValue)
		if err != nil {
			return DefaultFormat, &FormatError{Source: EnvVarOutput, Value: envValue}
		}
		return f, nil
	}

	// Rung 3: the nearest .glassfrogrc output value. An rcfile read/format error
	// fails loud (no fall-through to the default); an absent value is skipped inside
	// the walk (fileFound == false).
	if fileErr != nil {
		return DefaultFormat, fileErr
	}
	if fileFound {
		f, err := ParseFormat(fileValue)
		if err != nil {
			return DefaultFormat, &FormatError{Source: filePath, Value: fileValue}
		}
		return f, nil
	}

	// Rung 4: the built-in default backstops the chain — always a format.
	return DefaultFormat, nil
}
