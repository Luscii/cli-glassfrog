package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/Luscii/cli-glassfrog/internal/output"
	"github.com/Luscii/cli-glassfrog/internal/render"
	"golang.org/x/term"
)

// selectionSeam is the narrow slice of every read command's seam that User-Defined
// Template Output (035) needs: resolve the output SELECTION (a built-in format or a
// user-template source — 020's resolveFormat widened, ADR-1) and read a selected
// template's text behind the seam (file/stdin I/O, ADR-4). Each command's full seam
// (meSeam, rolesSeam, …) declares both methods, so it satisfies this structurally and
// resolveRenderTarget can take any of them.
type selectionSeam interface {
	resolveSelection(flagValue string) (output.Selection, error)
	readTemplateSource(ref output.TemplateRef) (string, error)
}

// renderTarget is what a resolved selection becomes once any user template has been
// read and parsed (fail-fast, before the request): a built-in format, OR a prepared
// *render.UserTemplate. When tmpl != nil, format is DefaultFormat (Full) — a non-
// structured format — so the structured dispatch branches are never taken and a
// request FAILURE renders through the existing human cause-plus-next-step path (035
// renders successes only; failures keep today's form — plan Risks / 032 boundary).
type renderTarget struct {
	format output.OutputFormat
	tmpl   *render.UserTemplate // non-nil when a user template was selected
}

// resolveRenderTarget runs the resolve → read → parse fail-fast chain every
// --output-capable read shares (035 ADR-1/ADR-4): resolve the selection; if it is a
// user-template source, read its bytes and parse them into a *render.UserTemplate
// BEFORE any assembly or request. On any failure it writes to stderr and returns
// ok=false with the Outcome+error the caller returns directly (no request made). On
// success it returns the target (a built-in format, or a prepared user template) and
// ok=true. A resolution error (a present-but-invalid env/config selector,
// *output.FormatError) classifies as UsageError(2) via reportFormatResolutionError;
// a missing/unreadable file, an unparseable template, or empty/un-piped stdin all
// map to UsageError(2) via reportTemplateError.
func resolveRenderTarget(seam selectionSeam, outputFlag string, stderr io.Writer) (renderTarget, Outcome, error, bool) {
	sel, err := seam.resolveSelection(outputFlag)
	if err != nil {
		outcome, oerr := reportFormatResolutionError(stderr, err)
		return renderTarget{}, outcome, oerr, false
	}

	ref, ok := sel.AsTemplate()
	if !ok {
		// A built-in format — no template to read; render through the format path.
		return renderTarget{format: sel.Format}, Success, nil, true
	}

	// A user-template source: read + parse BEFORE assembly or any request (fail-fast).
	text, err := seam.readTemplateSource(ref)
	if err != nil {
		outcome, oerr := reportTemplateError(stderr, err)
		return renderTarget{}, outcome, oerr, false
	}
	tmpl, err := render.ParseUserTemplate(text)
	if err != nil {
		// Name the source on the typed parse error (render is source-agnostic) so the
		// stderr message identifies the file path or "stdin".
		nameTemplateSource(err, sourceLabel(ref))
		outcome, oerr := reportTemplateError(stderr, err)
		return renderTarget{}, outcome, oerr, false
	}
	tmpl.Source = sourceLabel(ref) // so a later execution error (post-response) names the origin
	return renderTarget{format: output.DefaultFormat, tmpl: tmpl}, Success, nil, true
}

// reportTemplateError writes a template-source failure to stderr and returns
// UsageError(2) — the single class for every operator-template failure caught before
// the request (a missing/unreadable file, an unparseable template, empty/un-piped
// stdin), mirroring how validateInclude returns a known-usage fail-fast directly
// (the category is known by construction at this call site, no classifier needed).
// The post-response EXECUTION failure is classified separately, inside writeHuman,
// through the *render.UserTemplateError classifier arm. The message is the error's
// own text, which names the source.
func reportTemplateError(stderr io.Writer, err error) (Outcome, error) {
	fmt.Fprintln(stderr, err.Error())
	return UsageError, err
}

// nameTemplateSource sets the operator-facing Source on a *render.UserTemplateError
// (in place — it is a pointer) when render returned one without a source, so the
// stderr message names the file path or "stdin". A no-op for any other error type.
func nameTemplateSource(err error, source string) {
	var ute *render.UserTemplateError
	if errors.As(err, &ute) && ute.Source == "" {
		ute.Source = source
	}
}

// sourceLabel is the operator-facing origin of a template source: the file path for
// a file, or "stdin" for the piped source.
func sourceLabel(ref output.TemplateRef) string {
	if ref.Kind == output.TemplateStdin {
		return "stdin"
	}
	return ref.Path
}

// writeHuman renders v through the active HUMAN renderer — a selected user template
// (035) or the built-in resource/format template (019) — and writes it to stdout,
// buffer-then-write. On a render failure nothing reaches stdout: a built-in
// *RenderError is a code defect (→ RuntimeError(1)); a *render.UserTemplateError is
// operator input (→ UsageError(2)). classifyClientError maps both — the
// UserTemplateError arm returns UsageError, an unmatched *RenderError falls to the
// RuntimeError fail-safe, preserving 019's mapping. Returns (Success, nil) on success.
func writeHuman(stdout, stderr io.Writer, tmpl *render.UserTemplate, resource render.Resource, format output.OutputFormat, v any) (Outcome, error) {
	var (
		text string
		rerr error
	)
	if tmpl != nil {
		text, rerr = tmpl.Render(v)
	} else {
		text, rerr = renderFn(resource, humanFormat(format), v)
	}
	if rerr != nil {
		fmt.Fprintln(stderr, rerr.Error())
		return classifyClientError(rerr), rerr
	}
	fmt.Fprint(stdout, text)
	return Success, nil
}

// maxPipedTemplateBytes caps how much piped stdin the `-o stdin` path reads. A
// template is larger than a token (006's 64 KiB token cap), but still bounded so the
// command never slurps an arbitrarily large pipe into memory. The cap and the
// overflow message ("piped template exceeds …") are template-specific so a large
// template never surfaces the token-worded error.
const maxPipedTemplateBytes = 1 << 20 // 1 MiB

// readTemplateSourceFrom is the pure source-read both seams share (035 ADR-4): a
// TemplateFile via readFile (production binds os.ReadFile, resolving a relative path
// against the cwd), a TemplateStdin via the injected bounded reader guarded by isTTY
// and an empty check (reusing the 006 readBoundedStdinN shape with a template cap and
// noun). Every failure is a usage-class error naming the source. Factoring the logic
// here keeps the production seam a thin binder and lets the test seam exercise the
// same logic over injected sources (no real network, no real ~/.glassfrogrc).
func readTemplateSourceFrom(ref output.TemplateRef, readFile func(string) ([]byte, error), isTTY bool, stdin io.Reader) (string, error) {
	switch ref.Kind {
	case output.TemplateStdin:
		if isTTY {
			return "", errors.New("-o stdin requires a template piped on standard input, but standard input is a terminal — pipe a template, e.g. `cat t.tmpl | glassfrog … -o stdin`")
		}
		text, err := readBoundedStdinN(stdin, maxPipedTemplateBytes, "template")
		if err != nil {
			return "", fmt.Errorf("could not read the template from stdin: %w", err)
		}
		if strings.TrimSpace(text) == "" {
			return "", errors.New("-o stdin: no template was piped to standard input (the pipe was empty)")
		}
		return text, nil
	case output.TemplateFile:
		b, err := readFile(ref.Path)
		if err != nil {
			return "", fmt.Errorf("could not read template file %q: %w", ref.Path, err)
		}
		return string(b), nil
	default:
		// Fail loud on an unknown/invalid TemplateKind rather than silently reading
		// it as a file — a future kind must be handled explicitly here.
		return "", fmt.Errorf("unknown template source kind %d", ref.Kind)
	}
}

// resolveSelection binds the real OS to Output Format Selection's resolver, widened
// to the 035 discriminated selection: a working directory that cannot be determined
// errors; a home directory that cannot be determined drops the home fallback (the
// ResolveBaseURLFromOS shape). It replaces productionSeam.resolveFormat (020) — every
// read command's seam now resolves a Selection.
func (productionSeam) resolveSelection(flagValue string) (output.Selection, error) {
	startDir, err := os.Getwd()
	if err != nil {
		return output.Selection{Format: output.DefaultFormat}, fmt.Errorf("could not determine the working directory: %w", err)
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "" // no home → skip the home fallback rather than fail
	}
	return output.ResolveSelectionFromOS(flagValue, startDir, homeDir)
}

// readTemplateSource is the single reader of the real filesystem / terminal for a
// selected user template (035 ADR-4): os.ReadFile for a file path (cwd-relative), the
// real os.Stdin behind the 006 bounded reader for the stdin marker, with the
// isTTY/empty guard. Tests bind a fake seam over injected sources, so the fail-fast
// cases are exercised without a real pipe or filesystem dependency.
func (productionSeam) readTemplateSource(ref output.TemplateRef) (string, error) {
	return readTemplateSourceFrom(ref, os.ReadFile, term.IsTerminal(int(os.Stdin.Fd())), os.Stdin)
}
