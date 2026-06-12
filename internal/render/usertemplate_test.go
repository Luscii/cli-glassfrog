package render

import (
	"errors"
	"strings"
	"testing"

	"github.com/Luscii/cli-glassfrog/internal/glassfrog"
)

// TestParseUserTemplate_Valid parses a well-formed caller template into a usable
// *UserTemplate with no error — the parse path needs no data and no I/O.
func TestParseUserTemplate_Valid(t *testing.T) {
	ut, err := ParseUserTemplate(`{{.Actor.Name}}`)
	if err != nil {
		t.Fatalf("valid template should parse, got %v", err)
	}
	if ut == nil {
		t.Fatal("a valid template should return a usable *UserTemplate, got nil")
	}
}

// TestParseUserTemplate_SyntaxError returns a *UserTemplateError{StageParse} and a
// nil template for a malformed template — discriminable via errors.As.
func TestParseUserTemplate_SyntaxError(t *testing.T) {
	ut, err := ParseUserTemplate(`{{.Unclosed`)
	if ut != nil {
		t.Fatalf("a syntax error should return a nil template, got %v", ut)
	}
	var ute *UserTemplateError
	if !errors.As(err, &ute) {
		t.Fatalf("a parse failure should be a *UserTemplateError, got %T (%v)", err, err)
	}
	if ute.Stage != StageParse {
		t.Errorf("a syntax error should carry StageParse, got %v", ute.Stage)
	}
}

// TestUserTemplate_Render_Success renders the result value through the template.
func TestUserTemplate_Render_Success(t *testing.T) {
	ut, err := ParseUserTemplate(`actor={{.Actor.Name}}`)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	got, err := ut.Render(glassfrog.MeResponse{Actor: glassfrog.Actor{Name: "Alice"}})
	if err != nil {
		t.Fatalf("render should succeed, got %v", err)
	}
	if got != "actor=Alice" {
		t.Errorf("unexpected render: %q", got)
	}
}

// TestUserTemplate_Render_GuardedAbsence renders the author's explicit-absence
// marker for a present-but-empty embedded collection — no fabricated value, no
// execution error (a guarded struct field that is empty takes the else branch).
func TestUserTemplate_Render_GuardedAbsence(t *testing.T) {
	ut, err := ParseUserTemplate(`{{if .Roles}}has-roles{{else}}—{{end}}`)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	got, err := ut.Render(glassfrog.MeResponse{}) // Roles nil/empty
	if err != nil {
		t.Fatalf("a guarded absence should not error, got %v", err)
	}
	if got != "—" {
		t.Errorf("expected the explicit-absence marker, got %q", got)
	}
}

// TestUserTemplate_Render_MissingStructField surfaces an unguarded reference to an
// absent struct field as a *UserTemplateError{StageExecute} with empty output
// (buffer-then-return) — never a fabricated value.
func TestUserTemplate_Render_MissingStructField(t *testing.T) {
	ut, err := ParseUserTemplate(`{{.NoSuchField}}`)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	got, err := ut.Render(glassfrog.MeResponse{Actor: glassfrog.Actor{Name: "Alice"}})
	if got != "" {
		t.Errorf("an execution failure must write no partial output, got %q", got)
	}
	var ute *UserTemplateError
	if !errors.As(err, &ute) {
		t.Fatalf("an execution failure should be a *UserTemplateError, got %T (%v)", err, err)
	}
	if ute.Stage != StageExecute {
		t.Errorf("an execution failure should carry StageExecute, got %v", ute.Stage)
	}
}

// TestUserTemplate_Render_MissingMapKey confirms the inherited
// Option("missingkey=error") guard applies to user templates: a truly-missing map
// key fails loud (StageExecute) rather than rendering silent fake data.
func TestUserTemplate_Render_MissingMapKey(t *testing.T) {
	ut, err := ParseUserTemplate(`{{.absent}}`)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	_, err = ut.Render(map[string]any{"present": "x"})
	var ute *UserTemplateError
	if !errors.As(err, &ute) || ute.Stage != StageExecute {
		t.Fatalf("a missing map key should fail loud as StageExecute, got %T (%v)", err, err)
	}
}

// TestUserTemplate_FuncMapAvailable confirms the package FuncMap
// (trimSpace/join/indent) is available to a user template and unchanged.
func TestUserTemplate_FuncMapAvailable(t *testing.T) {
	ut, err := ParseUserTemplate(`{{trimSpace "  hi  "}}|{{join .Tags ","}}|{{indent 2}}x`)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	got, err := ut.Render(map[string]any{"Tags": []string{"a", "b"}})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if got != "hi|a,b|    x" {
		t.Errorf("FuncMap helpers produced %q", got)
	}
}

// TestUserTemplate_ComposesBuiltin confirms the clone shares the built-in set: a
// user template may invoke a built-in by name, and the built-in's rendering is
// reproduced.
func TestUserTemplate_ComposesBuiltin(t *testing.T) {
	ut, err := ParseUserTemplate(`{{template "me.full.tmpl" .}}`)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if ut.tmpl.Lookup("me.full.tmpl") == nil {
		t.Fatal("the clone should share the built-in named templates")
	}
	data := glassfrog.MeResponse{
		Actor:        glassfrog.Actor{Name: "Alice", Kind: "human", ID: "per_x"},
		Organization: glassfrog.Organization{Name: "Acme", ID: "org_x"},
		Membership:   glassfrog.Membership{AccessLevel: "admin"},
	}
	composed, err := ut.Render(data)
	if err != nil {
		t.Fatalf("composing a built-in should render, got %v", err)
	}
	builtin, err := Render(ResourceMe, FormatFull, data)
	if err != nil {
		t.Fatalf("built-in render: %v", err)
	}
	if composed != builtin {
		t.Errorf("composed output should match the built-in:\n got %q\nwant %q", composed, builtin)
	}
}

// TestUserTemplateError_DistinctFromRenderError confirms the two typed errors do
// not cross-match under errors.As — a user-template failure (UsageError) is never
// mistaken for a built-in defect (RuntimeError) and vice versa.
func TestUserTemplateError_DistinctFromRenderError(t *testing.T) {
	var ute error = &UserTemplateError{Stage: StageParse, Err: errors.New("x")}
	var re error = &RenderError{Resource: ResourceMe, Format: FormatFull, Err: errors.New("y")}

	var asRender *RenderError
	if errors.As(ute, &asRender) {
		t.Error("a *UserTemplateError must not match *RenderError")
	}
	var asUser *UserTemplateError
	if errors.As(re, &asUser) {
		t.Error("a *RenderError must not match *UserTemplateError")
	}
}

// TestUserTemplateError_NamesSource confirms the error message names the source
// once the cli seam has set it.
func TestUserTemplateError_NamesSource(t *testing.T) {
	e := &UserTemplateError{Stage: StageExecute, Source: "./roles.tmpl", Err: errors.New("boom")}
	if !strings.Contains(e.Error(), "./roles.tmpl") {
		t.Errorf("the error should name the source, got %q", e.Error())
	}
}
