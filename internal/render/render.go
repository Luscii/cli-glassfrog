// Package render maps a read command's decoded result into human-readable text
// through named text/template built-ins. It is the human half of the Output
// Formatting cluster (the machine JSON/YAML sibling is a separate concern), and
// the load-bearing seam Output Format Selection (020) and User-Defined Template
// Output (029) build on: 020 selects a built-in template by name, 029 registers
// caller-supplied templates into the same engine.
//
// It ships two built-in templates per result type — full (field-equivalent to
// each read's pre-019 projection) and compact (a denser, one-line-per-record
// variant) — embedded as files via //go:embed. It depends only on
// internal/glassfrog (the result structs it renders) and the stdlib; it must
// never import internal/cli or internal/apiclient (it owns no commands, no
// transport, and no exit codes — the same "lower layers never import cli"
// layering apiclient follows).
//
// Rendering operates on response-side result structs only — the token is an
// X-Auth-Token request header, never a result field — so the secret-never-emitted
// rule holds by construction (continuing 011).
package render

import (
	"bytes"
	"embed"
	"fmt"
	"strings"
	"text/template"

	"github.com/Luscii/cli-glassfrog/internal/glassfrog"
)

// RoleView is the data the `role` templates (025) render: the single role's
// RoleDetail plus the set of ?include values the operator requested. The API
// returns the same empty array for an unrequested resource and a
// requested-but-empty one, so Requested is the only signal that lets the
// template omit an unrequested section yet show an explicit-absence marker for a
// requested-but-empty one (interface-cli; ADR-2). A template reads a flag with
// `{{if index .Requested "policies"}}` — index returns false for an absent key,
// so an unrequested section is simply skipped.
type RoleView struct {
	Detail    glassfrog.RoleDetail
	Requested map[string]bool
}

// TreeView is the data the `tree` templates (026) render: the recursive
// hierarchy flattened into pre-order Rows (each carrying its depth, so the
// template shows nesting purely through `indent`), plus the set of ?include
// values the operator requested (the same omit-unrequested / mark-empty signal
// RoleView carries). Flattening the recursion in Go keeps the templates a flat
// range — text/template has no native depth-passing recursion — while the row
// order is exactly a pre-order tree walk, so reading top-to-bottom reconstructs
// the hierarchy.
type TreeView struct {
	Rows      []TreeRow
	Requested map[string]bool
}

// TreeRow is one node in the flattened tree. Name and Purpose are dereferenced
// from the node's nullable *string fields (nil → ""), so the template renders
// plain strings and never prints a pointer or <nil>. BelowDepth is the
// depth-boundary signal: true when the API reports the node HAS subroles but the
// result carries none (a --depth cut or an API-withheld branch), so the template
// marks it distinctly from a true leaf without inventing a descendant count
// (interface-cli; 026 ADR-2). ChildCount is the number of children present IN THE
// RESULT (compact's `children=N`), which is 0 for a depth-boundary node.
type TreeRow struct {
	Depth            int
	ID               string
	Name             string
	Purpose          string
	HasSubroles      bool
	BelowDepth       bool
	ChildCount       int
	Flags            []string
	Accountabilities []glassfrog.Accountability
	Domains          []glassfrog.Domain
	Fillers          []glassfrog.Actor
}

// NewTreeView flattens a TreeNode hierarchy into a TreeView by a pre-order walk,
// carrying depth and dereferencing the nullable name/purpose. requested is the
// closed ?include set the run validated; the template guards each per-node
// section on it.
func NewTreeView(root glassfrog.TreeNode, requested map[string]bool) TreeView {
	var rows []TreeRow
	var walk func(n glassfrog.TreeNode, depth int)
	walk = func(n glassfrog.TreeNode, depth int) {
		rows = append(rows, TreeRow{
			Depth:            depth,
			ID:               n.ID,
			Name:             derefString(n.Name),
			Purpose:          derefString(n.Purpose),
			HasSubroles:      n.HasSubroles,
			BelowDepth:       n.HasSubroles && len(n.Children) == 0,
			ChildCount:       len(n.Children),
			Flags:            n.Flags,
			Accountabilities: n.Accountabilities,
			Domains:          n.Domains,
			Fillers:          n.Fillers,
		})
		for _, c := range n.Children {
			walk(c, depth+1)
		}
	}
	walk(root, 0)
	return TreeView{Rows: rows, Requested: requested}
}

// derefString returns the pointed-to string, or "" for a nil pointer — so a null
// JSON field renders as an empty/absent value, never a pointer literal.
func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// SubrolesView is the data the `subroles` templates (026) render: the gathered
// immediate-child RoleDetails plus the requested ?include set. It mirrors
// RoleView's omit-unrequested / mark-empty guard, applied per child. An empty
// Children set renders the explicit `No subroles.` line (a leaf role is a valid
// empty answer, not an error).
type SubrolesView struct {
	Children  []glassfrog.RoleDetail
	Requested map[string]bool
}

// Resource names a read result type. Its constants are the single source of
// truth for the resource half of a template key: the read commands pass them,
// the template names derive from them (<resource>.<format>), and 020 maps its
// --output flag value onto a Format. No call site spells a key as a bare literal.
type Resource string

// Format names a built-in template variant. full is the standing CLI output
// (byte-equivalent to each read's pre-019 projection); compact is built and
// unit-verified but reachable from no operator surface until 020 wires --output.
type Format string

const (
	ResourceMe       Resource = "me"
	ResourceRoles    Resource = "roles"
	ResourceActions  Resource = "actions"
	ResourceProjects Resource = "projects"
	// ResourceOrgRoles is the org-wide role list (025): GET /roles rendered as
	// []glassfrog.Role. Distinct from ResourceRoles (the token-scoped `me roles`
	// list), which is untouched.
	ResourceOrgRoles Resource = "org-roles"
	// ResourceRole is the single org role read (025): GET /roles/{id} rendered as
	// a RoleView (the RoleDetail plus the requested ?include set). Singular,
	// distinct from ResourceRoles.
	ResourceRole Resource = "role"
	// ResourceTree is the recursive org-tree read (026): GET /tree or
	// GET /roles/{id}/tree rendered as a TreeView (the hierarchy flattened into
	// depth-carrying rows plus the requested ?include set). The first non-flat
	// render key — depth shows as leading indentation.
	ResourceTree Resource = "tree"
	// ResourceSubroles is the paginated immediate-children read (026):
	// GET /roles/{id}/subroles rendered as a SubrolesView (the gathered child
	// RoleDetails plus the requested ?include set). Distinct from ResourceRole
	// (singular) and ResourceOrgRoles (the whole-org flat list).
	ResourceSubroles Resource = "subroles"

	FormatFull    Format = "full"
	FormatCompact Format = "compact"
)

// builtinResources and builtinFormats enumerate every key the engine ships, so
// the registry-exhaustiveness test can assert all (Resource × Format) templates
// resolve (a dropped or misnamed template fails loud, not silently at runtime —
// PR #10 LEARNINGS).
var (
	builtinResources = []Resource{ResourceMe, ResourceRoles, ResourceActions, ResourceProjects, ResourceOrgRoles, ResourceRole, ResourceTree, ResourceSubroles}
	builtinFormats   = []Format{FormatFull, FormatCompact}
)

// templatesFS bundles the eight built-in template files at compile time, so no
// runtime file read is needed (CONSTITUTION XII self-containment holds). Each
// file is named <resource>.<format>.tmpl.
//
//go:embed templates/*.tmpl
var templatesFS embed.FS

// funcMap provides only the helpers the data-fidelity rules need that template
// syntax can't express inline. The helpers are pure and token-free.
var funcMap = template.FuncMap{
	// trimSpace mirrors strings.TrimSpace, so a template can detect a blank
	// field (and render its landed explicit-absence marker) the same way the
	// pre-019 projections did.
	"trimSpace": strings.TrimSpace,
	// join renders a string slice (a record's tags) the way the projections did.
	"join": func(items []string, sep string) string { return strings.Join(items, sep) },
	// indent returns the leading whitespace for a tree node at the given depth —
	// two spaces per level — so the recursive `tree` templates show depth by
	// indentation (026 ADR-2). A negative depth clamps to none.
	"indent": func(depth int) string {
		if depth < 0 {
			depth = 0
		}
		return strings.Repeat("  ", depth)
	},
}

// templates is the single parsed set of all built-ins. text/template (not
// html/template): CLI output is plain text, so no HTML auto-escaping is wanted.
// Option("missingkey=error") makes a truly-missing map key fail loud rather than
// rendering as <no value>, backstopping a typo'd key (the data-fidelity guard,
// ADR-3). A parse failure is a build-time defect, so template.Must panics it at
// package init rather than surfacing it per render.
var templates = template.Must(
	template.New("render").
		Funcs(funcMap).
		Option("missingkey=error").
		ParseFS(templatesFS, "templates/*.tmpl"),
)

// RenderError is the typed failure Render returns: an unknown resource/format
// key (no matching built-in) or a template execution error (a missing key under
// missingkey=error, a FuncMap helper error, or a type mismatch). It is
// errors.As-discriminable, and Err wraps the underlying cause. It carries no
// token and no request data — only the keys and the engine's error. A built-in
// RenderError is a code defect, so the consuming read maps it to RuntimeError(1)
// (interface-cli / ADR-4), like 011's undecodable-body handling.
type RenderError struct {
	Resource Resource
	Format   Format
	Err      error
}

func (e *RenderError) Error() string {
	return fmt.Sprintf("render %s.%s: %v", e.Resource, e.Format, e.Err)
}

func (e *RenderError) Unwrap() error { return e.Err }

// templateName derives the embedded template's name from a resource/format pair.
func templateName(resource Resource, format Format) string {
	return fmt.Sprintf("%s.%s.tmpl", resource, format)
}

// Render executes the built-in template named <resource>.<format> against data,
// into an in-memory buffer, and returns the rendered text on success. It returns
// ("", *RenderError) on any failure — an unknown resource/format, or a template
// execution error — never partial output (buffer-then-return, ADR-4): the caller
// writes the returned string to stdout only when err == nil, so a render failure
// never leaves partial bytes on stdout.
//
// Render is pure over its inputs: no I/O, no network, no token. The same
// (resource, format, data) always yields the same string. data is the read's
// decoded result value (glassfrog.MeResponse / MyRolesResponse / MyActionsResponse
// / MyProjectsResponse); a type the template doesn't expect surfaces as a
// *RenderError, never a silent zero-value render.
func Render(resource Resource, format Format, data any) (string, error) {
	name := templateName(resource, format)
	t := templates.Lookup(name)
	if t == nil {
		return "", &RenderError{Resource: resource, Format: format, Err: fmt.Errorf("no built-in template %q", name)}
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", &RenderError{Resource: resource, Format: format, Err: err}
	}
	return buf.String(), nil
}
