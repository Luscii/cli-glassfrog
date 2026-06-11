package render

import (
	"bytes"
	"fmt"
	"text/template"
)

// UserTemplateStage discriminates where a caller-supplied template failed:
//
//   - StageParse: a syntax error in the caller text, caught BEFORE any request
//     (fail-fast — ParseUserTemplate needs no data and does no I/O).
//   - StageExecute: the template parsed but referenced data the result did not
//     carry (a missing map key under the inherited Option("missingkey=error"), an
//     absent struct field, or a FuncMap helper error), caught AFTER a successful
//     response.
//
// Both stages map to UsageError(2) in internal/cli (ADR-3) — a user template is
// operator input, not a CLI defect — but the stage tells an operator whether a
// request was already spent.
type UserTemplateStage int

const (
	// StageParse is a parse-time failure (pre-request).
	StageParse UserTemplateStage = iota
	// StageExecute is an execution-time failure (post-response).
	StageExecute
)

// UserTemplateError is the typed failure ParseUserTemplate and (*UserTemplate).Render
// return. It is errors.As-discriminable and DISTINCT from *RenderError: a built-in
// *RenderError is a code defect (→ RuntimeError(1)), whereas a UserTemplateError is
// the operator's own template (→ UsageError(2), ADR-3). Source is the operator-facing
// origin (a file path, or "stdin"); it is set by the cli seam that knows where the
// text came from (render is a pure leaf with no notion of I/O origins) and is left
// empty by this package. Err wraps the underlying text/template cause. It carries no
// token and no request data (the token is an X-Auth-Token request header, never on
// the render path — continuing 011/019).
type UserTemplateError struct {
	Stage  UserTemplateStage
	Source string
	Err    error
}

func (e *UserTemplateError) Error() string {
	origin := e.Source
	if origin == "" {
		origin = "the supplied template"
	} else {
		origin = "template " + origin
	}
	switch e.Stage {
	case StageExecute:
		return fmt.Sprintf("%s failed during rendering: %v", origin, e.Err)
	default:
		return fmt.Sprintf("%s failed to parse: %v", origin, e.Err)
	}
}

func (e *UserTemplateError) Unwrap() error { return e.Err }

// userTemplateBase is a clone of the built-in set taken at package init — before any
// Render execution — and NEVER executed itself, so ParseUserTemplate can clone it
// safely regardless of how many built-in renders have run (text/template forbids
// cloning a set after it has executed). Each ParseUserTemplate clones THIS base, so
// every user template gets its own associated copy of the built-ins, sharing the
// package FuncMap and Option("missingkey=error") by construction (ADR-2) and able to
// compose a built-in via {{template "me.full.tmpl" .}}.
var userTemplateBase = template.Must(templates.Clone())

// UserTemplate is a caller-supplied template parsed into a clone of the built-in set.
// It is opaque: the caller parses it once (fail-fast, before any request) and renders
// the read's decoded result value through it once (after a successful response).
type UserTemplate struct {
	tmpl *template.Template
	// Source is the operator-facing origin (file path or "stdin"), set by the cli
	// seam after a successful parse so an execution error (which surfaces only at
	// Render time, post-response) can name where the template came from. Empty until
	// set; rendering does not depend on it.
	Source string
}

// ParseUserTemplate parses text into a clone of the built-in set and returns a usable
// *UserTemplate, or (nil, *UserTemplateError{Stage: StageParse}) on a syntax error. It
// performs NO I/O and needs NO data, so the caller can parse before assembling any
// connection or sending any request (the fail-fast window, ADR-4). The parsed template
// shares the package FuncMap (trimSpace/join/indent) and Option("missingkey=error") —
// the same data-fidelity guard the built-ins use — and may reference a built-in by
// name. No FuncMap helper exposes file/network/exec, so the data-only sandbox holds by
// construction (ADR-4): a user template can only project the data it is handed.
func ParseUserTemplate(text string) (*UserTemplate, error) {
	clone, err := userTemplateBase.Clone()
	if err != nil {
		return nil, &UserTemplateError{Stage: StageParse, Err: err}
	}
	parsed, err := clone.Parse(text)
	if err != nil {
		return nil, &UserTemplateError{Stage: StageParse, Err: err}
	}
	return &UserTemplate{tmpl: parsed}, nil
}

// Render executes the template against data into an in-memory buffer and returns the
// rendered text on success, or ("", *UserTemplateError{Stage: StageExecute}) on an
// execution error — NEVER partial output (buffer-then-return, mirroring Render's ADR-4
// contract). data is the read's decoded result value — the same value the built-in
// Render(resource, …) receives for that resource. Render is pure over data: no I/O, no
// network, no token. The returned error carries u.Source so the cli can name the origin.
func (u *UserTemplate) Render(data any) (string, error) {
	var buf bytes.Buffer
	if err := u.tmpl.Execute(&buf, data); err != nil {
		return "", &UserTemplateError{Stage: StageExecute, Source: u.Source, Err: err}
	}
	return buf.String(), nil
}
