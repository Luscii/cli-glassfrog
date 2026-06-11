package output

import (
	"strings"

	"github.com/Luscii/cli-glassfrog/internal/rcfile"
)

// reservedStdin is the fifth reserved --output flag word added by User-Defined
// Template Output (035): at the FLAG rung, "stdin" selects a user template read
// from a pipe rather than a file literally named "stdin". The four ParseFormat
// tokens (full/compact/json/yaml) PLUS this one are the reserved set; any other
// non-empty flag value is a template file path. Centralized here as the single
// source of truth shared with interface-cli's widened usage string.
const reservedStdin = "stdin"

// TemplateKind discriminates a user-template source (035): a file path on disk, or
// the template piped on standard input.
type TemplateKind int

const (
	// TemplateFile names a template read from a file path.
	TemplateFile TemplateKind = iota
	// TemplateStdin names a template read from piped standard input.
	TemplateStdin
)

// TemplateRef names a user-template source resolved from the --output flag (035).
// Path is the file path for TemplateFile and is empty (ignored) for TemplateStdin.
// It carries only the path/marker — this package performs NO file read and does not
// import internal/render; the actual read lives behind the cli seam (ADR-4).
type TemplateRef struct {
	Kind TemplateKind
	Path string
}

// Selection is what ResolveSelection yields: either a built-in OutputFormat or a
// user-template source. Template == nil means a built-in format (read Format);
// a non-nil Template means a user template was selected at the flag rung. It is the
// wider shape the per-command cli seam now returns, replacing the bare OutputFormat
// (020's selection vocabulary, extended — 032 consumes the same Selection for
// failure rendering).
type Selection struct {
	// Format is the built-in format; meaningful only when Template == nil.
	Format OutputFormat
	// Template, when non-nil, names the user-template source selected at the flag rung.
	Template *TemplateRef
}

// AsTemplate returns the TemplateRef and true when a user template was selected, or
// the zero TemplateRef and false for a built-in format. It is the discriminator the
// cli read uses to decide whether to read+parse a template before the request.
func (s Selection) AsTemplate() (TemplateRef, bool) {
	if s.Template == nil {
		return TemplateRef{}, false
	}
	return *s.Template, true
}

// ResolveSelectionFromOS is the thin production seam over ResolveSelection: it binds
// the real GLASSFROG_OUTPUT lookup and the internal/rcfile nearest-wins walk (over
// the output key) to the pure core's injected sources, mirroring ResolveFormatFromOS.
func ResolveSelectionFromOS(flagValue, startDir, homeDir string) (Selection, error) {
	envValue := getenv(EnvVarOutput)
	fileValue, filePath, fileFound, fileErr := rcfile.Resolve(startDir, homeDir, outputKey)
	return ResolveSelection(flagValue, envValue, fileValue, filePath, fileFound, fileErr)
}

// ResolveSelection is the pure precedence core for the discriminated selection
// (035 ADR-1). It reuses ResolveFormat's precedence (flag → env → file → default)
// but, at the FLAG rung only, classifies a non-empty value into a built-in format
// or a user-template source:
//
//   - a reserved format token (full/compact/json/yaml, any casing) → that OutputFormat
//   - "stdin" (any casing)                                          → TemplateRef{TemplateStdin}
//   - any other non-empty value                                     → TemplateRef{TemplateFile, value}
//
// Reserved names win — a value equal to a reserved word never becomes a file. The
// env and file rungs are UNCHANGED: they still parse the four tokens or yield a
// *FormatError (and the rcfile read error unchanged), and NEVER a TemplateRef — a
// template source is reachable only from the command line (ADR-1). When every source
// is absent the built-in default (Full) backstops the chain. On error the zero
// Selection (Full, no template) is returned as a placeholder.
func ResolveSelection(flagValue, envValue, fileValue, filePath string, fileFound bool, fileErr error) (Selection, error) {
	// Flag rung: a non-empty value short-circuits all lower rungs and may name a
	// user-template source (035). A whitespace-only value is treated as absent and
	// falls through (the base-URL / ResolveFormat convention).
	if strings.TrimSpace(flagValue) != "" {
		return classifyFlagSelection(flagValue), nil
	}

	// Env / file / default rungs: unchanged. Delegate to ResolveFormat with an empty
	// flag so it skips rung 1 — it yields one of the four tokens, a *FormatError
	// naming the env/file source, or the rcfile read error; never a TemplateRef.
	format, err := ResolveFormat("", envValue, fileValue, filePath, fileFound, fileErr)
	if err != nil {
		return Selection{Format: DefaultFormat}, err
	}
	return Selection{Format: format}, nil
}

// classifyFlagSelection classifies a non-empty --output flag value (035 ADR-1):
// reserved tokens win (format tokens, then the stdin marker), and anything else is a
// template file path. The file Path keeps the raw flag value (not lowercased — it is
// a filesystem path), while the reserved comparison is case-insensitive and
// whitespace-trimmed (matching ParseFormat).
func classifyFlagSelection(flagValue string) Selection {
	if format, err := ParseFormat(flagValue); err == nil {
		return Selection{Format: format}
	}
	if strings.ToLower(strings.TrimSpace(flagValue)) == reservedStdin {
		return Selection{Template: &TemplateRef{Kind: TemplateStdin}}
	}
	return Selection{Template: &TemplateRef{Kind: TemplateFile, Path: flagValue}}
}
