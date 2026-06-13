package output

import (
	"strings"

	"github.com/Luscii/cli-glassfrog/internal/resolve"
)

// reservedStdin is the fifth reserved --output flag word added by User-Defined
// Template Output (035): at the FLAG rung, "stdin" selects a user template read
// from a pipe rather than a file literally named "stdin". The four ParseFormat
// tokens (full/compact/json/yaml) PLUS this one are the reserved set; any other
// non-empty flag value is a template file path. Centralized here as the single
// source of truth for this package's flag-rung resolution; the CLI's `-o` help
// prose (internal/cli/root.go) names the token independently, the same way it
// hand-writes the four format tokens.
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

// ResolveSelectionFromOS is the single composing entry for the discriminated
// selection (035 widened by the 040 retrofit). It composes the precedence walk —
// the --output flag, then GLASSFROG_OUTPUT, then the .glassfrogrc output key, then
// the built-in default — onto the one shared resolve walk (039), binding the real
// GLASSFROG_OUTPUT lookup (the getenv seam) and the internal/rcfile nearest-wins
// walk (resolve.FromFile over the output key); startDir/homeDir are injected so the
// walk is hermetic (ADR-4). It replaces the former 6-arg pre-fetched-source pure
// core, which no longer fits now that resolve fetches env/file itself (mirrors base
// URL's composing core).
//
// The flag rung is PRESENCE-based (ADR-2): a supplied --output (flagPresent, cobra
// Changed()) wins its rung even with an empty/whitespace value — its act of being
// typed is the signal. An unsupplied flag falls through to the environment; the env
// and file rungs keep their non-empty-after-trim yield rule, so a whitespace-only
// GLASSFROG_OUTPUT / file value still falls through.
//
// The winner is interpreted at the call site (ADR-3), keyed off its provenance:
//
//   - a FLAG winner → classifyFlagSelection: a reserved format token
//     (full/compact/json/yaml, any casing) → that OutputFormat; "stdin" (any casing)
//     → TemplateRef{TemplateStdin}; any other non-empty value → TemplateRef{
//     TemplateFile, path}. A non-token flag value is therefore NOT an error — it
//     selects a user template (035). An empty/whitespace flag classifies as a
//     degenerate empty template selection that fails loud downstream.
//   - an ENV/FILE winner → ParseFormat: one of the four tokens, or a *FormatError
//     naming the source via Provenance.Origin (GLASSFROG_OUTPUT or the file path).
//     A template is NEVER reachable here — templates are flag-only (035 ADR-1).
//   - the DEFAULT winner → Selection{DefaultFormat} (Full), valid by construction and
//     never re-validated.
//
// A resolution error (an unreadable/unparseable .glassfrogrc) surfaces verbatim
// before any parse, with NO fall-through to the default. On error the zero Selection
// (Full, no template) is returned as a placeholder.
func ResolveSelectionFromOS(flagValue string, flagPresent bool, startDir, homeDir string) (Selection, error) {
	res, err := resolve.Resolve(
		resolve.FromFlags(resolve.Flag{Name: "--" + FlagOutput, Present: flagPresent, Value: flagValue}),
		resolve.FromEnv(getenv, EnvVarOutput),
		resolve.FromFile(startDir, homeDir, outputKey),
		resolve.Default(FormatFull.String()), // "full"; classified back to DefaultFormat
	)
	if err != nil {
		return Selection{Format: DefaultFormat}, err // unreadable/unparseable .glassfrogrc → fail loud, no fall-through
	}

	switch res.Provenance.Kind {
	case resolve.KindFlag:
		// Flag rung: token → format, "stdin" → stdin template, else → template file
		// path (035). A non-token flag value is NOT an error.
		return classifyFlagSelection(res.Value), nil
	case resolve.KindEnv, resolve.KindFile:
		// Env/file rung: only the four tokens; a non-token fails loud naming the
		// source (templates are flag-only).
		format, perr := ParseFormat(res.Value)
		if perr != nil {
			return Selection{Format: DefaultFormat}, &FormatError{Source: res.Provenance.Origin, Value: res.Value}
		}
		return Selection{Format: format}, nil
	default:
		// Default rung (KindDefault): the built-in default backstops the chain,
		// valid by construction and never re-validated.
		return Selection{Format: DefaultFormat}, nil
	}
}

// classifyFlagSelection classifies a non-empty --output flag value (035 ADR-1):
// reserved tokens win (format tokens, then the stdin marker), and anything else is a
// template file path. The file Path is whitespace-trimmed — surrounding whitespace
// is insignificant, consistent with the flag-rung presence check (ResolveSelection
// uses strings.TrimSpace to decide non-emptiness) and the reserved-token comparison,
// so `-o " ./t.tmpl "` resolves to the same file as `-o ./t.tmpl`. It is NOT
// lowercased — case is significant for a filesystem path.
func classifyFlagSelection(flagValue string) Selection {
	if format, err := ParseFormat(flagValue); err == nil {
		return Selection{Format: format}
	}
	trimmed := strings.TrimSpace(flagValue)
	if strings.ToLower(trimmed) == reservedStdin {
		return Selection{Template: &TemplateRef{Kind: TemplateStdin}}
	}
	return Selection{Template: &TemplateRef{Kind: TemplateFile, Path: trimmed}}
}
