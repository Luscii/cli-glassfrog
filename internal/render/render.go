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

// PoliciesView is the data the `policies` templates (034) render: the per-role
// list of policies governing a role's interior. An empty Policies set renders the
// explicit `No policies.` line (a role with no policies is a valid empty answer,
// not an error). Each policy projects its title + id and its scope (RoleID/DomainID)
// with explicit-absence markers for the nullable scope fields.
type PoliciesView struct {
	Policies []glassfrog.Policy
}

// PolicyView is the data the `policy` templates (034) render: a single policy with
// its FULL body rendered verbatim — never truncated or reflowed (CONSTITUTION VI)
// — plus its scope (RoleID/DomainID) and timestamps (CreatedAt/UpdatedAt) with
// explicit-absence guards for the nullable fields. It is the first template to
// render a long free-text Body as primary content.
type PolicyView struct {
	Policy glassfrog.Policy
}

// ProjectsView is the data the role-scoped `projects` list templates (038)
// render: the projects owned by a role, walked to completion. It reuses the
// landed `projects` render key (014) unchanged — the templates range over .Data,
// so this view exposes the same Data field MyProjectsResponse does (014's
// /me/projects envelope), letting the role-addressable list share the projection
// without a second template. An empty Data set renders the explicit `no projects`
// line (a role that owns no projects, or a filter that matched none, is a valid
// empty answer, not an error).
type ProjectsView struct {
	Data []glassfrog.Project
}

// ProjectView is the data the singular `project` templates (038) render: a single
// project with its full detail — status, description, owning role (or the
// individual-initiative marker for a null role_id), parent (or the top-level
// marker), the has_sub_projects/has_actions presence signals, tags, timestamps,
// link, and note — with explicit-absence guards for every nullable field. It
// mirrors PolicyView{Policy}. The free-text description/note are rendered verbatim
// — never truncated or reflowed (CONSTITUTION VI).
type ProjectView struct {
	Project glassfrog.Project
}

// TensionView is the data the `tension` templates (042) render: a single created
// tension with its free-text Body rendered verbatim as primary content — never
// truncated or reflowed (CONSTITUTION VI), the Policy.Body / Project.note
// precedent — plus its status badge and the nullable Label/RoleID/SensedByID/
// MeetingType/ParentRoleID with explicit-absence guards, and the timestamps. It
// mirrors PolicyView{Policy} / ProjectView{Project}. The load-bearing output is the
// ten_ id (a later proposal references it as tension_id).
type TensionView struct {
	Tension glassfrog.Tension
}

// TensionsView is the data the role-scoped `tensions` list templates (043) render:
// the tensions a role carries, walked to completion. It mirrors ProjectsView's
// shape (a single .Data slice the templates range over) — the plural list sibling
// of the landed singular `tension` key (042), with roles reversed from 038 (042
// shipped the singular; 043 adds the plural). Each row projects the ten_ id, the
// status badge, the nullable Label (explicit-absence marker when blank), the
// free-text Body rendered VERBATIM in `full` — never truncated or reflowed
// (CONSTITUTION VI), matching projects.full and the singular tension.full — and the
// nullable sensing RoleID. An empty Data set renders the explicit `no tensions`
// line (a role carrying no tensions, or a --status filter that matched none, is a
// valid empty answer, not an error).
type TensionsView struct {
	Data []glassfrog.Tension
}

// TensionDiscardView is the data the `tension-discard` templates (045) render: the
// synthesized confirmation of a soft-deleted tension. Unlike TensionView/TensionsView
// (which project a server-decoded glassfrog.Tension), a successful discard carries NO
// body — `DELETE /tensions/{id}` returns 204 with nothing to decode — so the command
// constructs this view client-side from the id it was given (plan ADR-3). It exposes
// ONLY the discarded ten_ id: there is no server-owned field (no discarded_at) to
// project, because the bodyless response never provided one (the result must claim
// nothing the server did not return — spec validation scenario). The human projection
// is a single confirmation line (`<ten_…>  [discarded]`), identical in full/compact.
type TensionDiscardView struct {
	ID string
}

// ActorsView is the data the org-wide `actors` directory templates (048) render:
// the actors in the organization, walked to completion. It mirrors ProjectsView's
// shape (a single .Data slice the templates range over) — a flat homogeneous list
// where every row is the same glassfrog.Actor (unlike SearchView's heterogeneous
// type-badged rows). Each row projects the per_/agt_ id, the kind badge, and the
// name; the name is rendered verbatim — never truncated or reflowed (CONSTITUTION
// VI) — with a trim-empty absence guard (the repo's `eq (trimSpace .X) ""`
// convention) so a blank name shows the `—` marker rather than a fabricated value.
// An empty Data set renders the explicit `no actors` line (an org/filter that
// matched no actor is a valid empty answer, not an error). The reused
// glassfrog.Actor also carries created_at/updated_at, which the directory row does
// not project — discovery surfaces id/name/kind only (plan ADR-2/ADR-4).
type ActorsView struct {
	Data []glassfrog.Actor
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

// DomainsView is the data the `domains` templates (033) render: the gathered
// domains a role controls (walked to completion). The list has no ?include, so —
// unlike RoleView/SubrolesView — it carries no Requested set. An empty Domains
// slice renders the explicit `No domains.` line (a role that controls no domains,
// or a search that matched none, is a valid empty answer, not an error).
type DomainsView struct {
	Domains []glassfrog.Domain
}

// DomainView is the data the `domain` templates (033) render: the single domain
// read by id plus the set of ?include values the operator requested. Like
// RoleView, Requested is the only signal that lets the template omit an
// unrequested section yet show an explicit-absence marker for a
// requested-but-empty one — here the single optional embed is `policies` (033
// ADR-2). The Domain's RoleID is nullable; ControllingRole below dereferences it
// so the template prints the id or its explicit-absence marker, never a pointer
// literal or a fabricated id (CONSTITUTION VIII).
type DomainView struct {
	Domain    glassfrog.Domain
	Requested map[string]bool
}

// ControllingRole returns the domain's controlling role id, or "" when the
// nullable role_id is absent (nil or empty). The template treats "" as the
// explicit-absence case and renders the `(no controlling role)` marker — never a
// pointer literal (a *string prints its address under fmt) and never a fabricated
// id. Centralizing the deref here keeps the template free of pointer handling,
// the same shape NewTreeView used for the nullable name/purpose.
func (v DomainView) ControllingRole() string {
	if v.Domain.RoleID == nil || *v.Domain.RoleID == "" {
		return ""
	}
	return *v.Domain.RoleID
}

// SearchView is the data the `search` templates (041) render: the relevance-
// ordered heterogeneous cross-model result set, flattened into display Rows. It is
// the first render key over a deliberately mixed-type list — every row carries a
// `type` badge so the operator can tell a role hit from a policy hit and drill in
// via type + id. Row order IS the API's relevance order (the walker appends pages
// in sequence); the renderer never re-sorts, de-dups, or filters (041 plan ADR-2).
// An empty Rows set renders the explicit `No results.` line — zero matches is a
// valid empty answer, not an error.
type SearchView struct {
	Rows []SearchRow
}

// SearchRow is one search hit projected for rendering: the nullable Excerpt/RoleID
// *string fields of a glassfrog.SearchResult are dereferenced to plain strings
// (nil → ""), so the template renders strings and never prints a pointer or <nil>
// — the same shape NewTreeView used for the nullable name/purpose. The template
// then guards on trim-emptiness (the repo's `eq (trimSpace .X) ""` convention): a
// null/blank Excerpt renders as the `—` absence marker, and the `Role:` line is
// emitted only when RoleID is non-blank (it applies to a subset of types). Absent
// fields are never fabricated (041 plan ADR-2; CONSTITUTION VIII).
type SearchRow struct {
	Type    string
	ID      string
	Title   string
	Rank    float64
	Excerpt string
	RoleID  string
}

// NewSearchView flattens a relevance-ordered SearchResult slice into a SearchView
// by dereferencing the nullable excerpt/role_id (nil → ""), preserving the input
// order exactly (no re-sort/de-dup/filter — 041 plan ADR-2).
func NewSearchView(results []glassfrog.SearchResult) SearchView {
	rows := make([]SearchRow, 0, len(results))
	for _, r := range results {
		rows = append(rows, SearchRow{
			Type:    r.Type,
			ID:      r.ID,
			Title:   r.Title,
			Rank:    r.Rank,
			Excerpt: derefString(r.Excerpt),
			RoleID:  derefString(r.RoleID),
		})
	}
	return SearchView{Rows: rows}
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
	// ResourceDomains is the paginated role-scoped domains list (033):
	// GET /roles/{id}/domains rendered as a DomainsView (the gathered domains a
	// role controls). Plural — distinct from ResourceDomain (the single read).
	ResourceDomains Resource = "domains"
	// ResourceDomain is the single domain read (033): GET /domains/{id} rendered as
	// a DomainView (the Domain plus the requested ?include set). Singular —
	// distinct from ResourceDomains (the list).
	ResourceDomain Resource = "domain"
	// ResourcePolicies is the per-role policy list read (034):
	// GET /roles/{id}/policies rendered as a PoliciesView ([]glassfrog.Policy).
	// Plural, distinct from ResourcePolicy (the single standalone read).
	ResourcePolicies Resource = "policies"
	// ResourcePolicy is the single standalone policy read (034):
	// GET /policies/{id} rendered as a PolicyView (one glassfrog.Policy with its
	// full body). Singular, distinct from ResourcePolicies.
	ResourcePolicy Resource = "policy"
	// ResourceProject is the single standalone project read (038):
	// GET /projects/{id} rendered as a ProjectView (one glassfrog.Project with its
	// full detail). Singular, distinct from ResourceProjects (the list key reused
	// from 014).
	ResourceProject Resource = "project"
	// ResourceTension is the single-tension projection: the createTension 201
	// {data: Tension} (042) and the getTension {data: Tension} read (043) rendered
	// as a TensionView (one glassfrog.Tension with its verbatim body). Singular —
	// first added on 042's write path and reused unchanged by 043's `tension get`
	// read; its plural list sibling is ResourceTensions, below.
	ResourceTension Resource = "tension"
	// ResourceTensions is the role-scoped tension list read (043):
	// GET /roles/{id}/tensions rendered as a TensionsView ([]glassfrog.Tension).
	// Plural — the list sibling of the landed singular ResourceTension (042); the
	// plural/singular mirror of 038, with roles reversed (042 shipped the singular,
	// 043 adds the plural).
	ResourceTensions Resource = "tensions"
	// ResourceTensionDiscard is the synthesized soft-delete confirmation (045):
	// DELETE /tensions/{id} returns no body, so the command builds a
	// TensionDiscardView{ID} client-side and renders it through this key (plan
	// ADR-3). Distinct from ResourceTension (the decoded single-tension projection)
	// — the discard result carries only the id + a discarded marker, no server-owned
	// fields. The first render key over a wholly synthesized (un-decoded) result.
	ResourceTensionDiscard Resource = "tension-discard"
	// ResourceSearch is the cross-model search read (041): GET /search rendered as
	// a SearchView (the relevance-ordered heterogeneous result list, each row a
	// `type`-badged hit). The first render key over a deliberately mixed-type list
	// — distinct from every per-resource key, and NOT split per type (which would
	// break the ranked order — 041 plan ADR-2).
	ResourceSearch Resource = "search"
	// ResourceActors is the org-wide actor directory read (048): GET /actors
	// rendered as an ActorsView (a flat homogeneous list of glassfrog.Actor). The
	// first list render key keyed purely on filters (no positional subject). Distinct
	// from ResourceMe, which renders ONE actor inside the `me` document (actor + org
	// + membership + roles) — a different projection, not reused (plan ADR-4).
	ResourceActors Resource = "actors"

	FormatFull    Format = "full"
	FormatCompact Format = "compact"
)

// builtinResources and builtinFormats enumerate every key the engine ships, so
// the registry-exhaustiveness test can assert all (Resource × Format) templates
// resolve (a dropped or misnamed template fails loud, not silently at runtime —
// PR #10 LEARNINGS).
var (
	builtinResources = []Resource{ResourceMe, ResourceRoles, ResourceActions, ResourceProjects, ResourceOrgRoles, ResourceRole, ResourceTree, ResourceSubroles, ResourceDomains, ResourceDomain, ResourcePolicies, ResourcePolicy, ResourceProject, ResourceSearch, ResourceActors, ResourceTension, ResourceTensions, ResourceTensionDiscard}
	builtinFormats   = []Format{FormatFull, FormatCompact}
)

// templatesFS bundles every built-in template file (one per Resource × Format
// pair — see builtinResources/builtinFormats) at compile time, so no runtime file
// read is needed (CONSTITUTION XII self-containment holds). Each file is named
// <resource>.<format>.tmpl; the registry-exhaustiveness test asserts the parsed
// count matches len(builtinResources)*len(builtinFormats).
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
