package render

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/Luscii/cli-glassfrog/internal/glassfrog"
)

// --- full goldens: byte-equivalence with the pre-019 projections -----------
//
// Each full golden is the verbatim output the corresponding landed projection
// (formatMe / formatMeRoles / formatMeActions / formatMeProjects) produced before
// 019. Any drift here regresses shipped output (plan Risk "full drifts"), so the
// templates are pinned to the exact bytes.

func TestRender_MeFull_IdentityOnly_Golden(t *testing.T) {
	me := glassfrog.MeResponse{
		Actor:        glassfrog.Actor{ID: "per_abc", Name: "Alice Smith", Kind: "human"},
		Organization: glassfrog.Organization{ID: "org_abc", Name: "Acme"},
		Membership:   glassfrog.Membership{AccessLevel: "admin"},
	}
	want := "actor:        Alice Smith (human) per_abc\n" +
		"organization: Acme (org_abc)\n" +
		"access:       admin\n"
	assertRender(t, ResourceMe, FormatFull, me, want)
}

func TestRender_MeFull_WithRoles_Golden(t *testing.T) {
	me := glassfrog.MeResponse{
		Actor:        glassfrog.Actor{ID: "agt_abc", Name: "Claude", Kind: "agent"},
		Organization: glassfrog.Organization{ID: "org_abc", Name: "Acme"},
		Membership:   glassfrog.Membership{AccessLevel: "normal"},
		Roles: []glassfrog.Role{
			{ID: "role_1", Name: "Marketing Lead"},
			{ID: "role_2", Name: "Treasurer"},
		},
	}
	want := "actor:        Claude (agent) agt_abc\n" +
		"organization: Acme (org_abc)\n" +
		"access:       normal\n" +
		"roles:\n" +
		"  - Marketing Lead (role_1)\n" +
		"  - Treasurer (role_2)\n"
	assertRender(t, ResourceMe, FormatFull, me, want)
}

// An empty roles embed omits the section entirely (me's landed behavior — not a
// (none) marker, unlike roles full).
func TestRender_MeFull_EmptyRolesEmbed_OmitsSection(t *testing.T) {
	me := glassfrog.MeResponse{
		Actor:        glassfrog.Actor{ID: "per_abc", Name: "Alice", Kind: "human"},
		Organization: glassfrog.Organization{ID: "org_abc", Name: "Acme"},
		Membership:   glassfrog.Membership{AccessLevel: "admin"},
	}
	want := "actor:        Alice (human) per_abc\n" +
		"organization: Acme (org_abc)\n" +
		"access:       admin\n"
	assertRender(t, ResourceMe, FormatFull, me, want)
}

func TestRender_RolesFull_SingleRole_Golden(t *testing.T) {
	resp := glassfrog.MyRolesResponse{Data: []glassfrog.Role{
		roleWith("role_x", "Marketing Lead", "A market that knows us", []string{"Brand voice"}, []string{"Planning"}),
	}}
	want := "Marketing Lead (role_x)\n" +
		"  Purpose: A market that knows us\n" +
		"  Domains:\n" +
		"    - Brand voice\n" +
		"  Accountabilities:\n" +
		"    - Planning\n"
	assertRender(t, ResourceRoles, FormatFull, resp, want)
}

// Two roles are separated by exactly one blank line; the last block leaves no
// trailing blank line (the strings.Join(blocks, "\n") shape of the projection).
func TestRender_RolesFull_TwoRoles_BlankLineBetween_Golden(t *testing.T) {
	resp := glassfrog.MyRolesResponse{Data: []glassfrog.Role{
		roleWith("role_1", "Lead", "Reach", []string{"Voice"}, []string{"Plan"}),
		roleWith("role_2", "Rep", "Speak", nil, nil),
	}}
	want := "Lead (role_1)\n" +
		"  Purpose: Reach\n" +
		"  Domains:\n" +
		"    - Voice\n" +
		"  Accountabilities:\n" +
		"    - Plan\n" +
		"\n" +
		"Rep (role_2)\n" +
		"  Purpose: Speak\n" +
		"  Domains:\n" +
		"    (none)\n" +
		"  Accountabilities:\n" +
		"    (none)\n"
	assertRender(t, ResourceRoles, FormatFull, resp, want)
}

// A blank/absent purpose renders the landed explicit-absence marker, and empty
// sections render the (none) marker — never a fabricated value, never an empty
// field.
func TestRender_RolesFull_BlankPurposeAndEmptySections_Markers(t *testing.T) {
	resp := glassfrog.MyRolesResponse{Data: []glassfrog.Role{
		roleWith("role_x", "Empty", "   ", nil, nil),
	}}
	want := "Empty (role_x)\n" +
		"  Purpose: (no purpose set)\n" +
		"  Domains:\n" +
		"    (none)\n" +
		"  Accountabilities:\n" +
		"    (none)\n"
	assertRender(t, ResourceRoles, FormatFull, resp, want)
}

func TestRender_ActionsFull_Golden(t *testing.T) {
	resp := glassfrog.MyActionsResponse{Data: []glassfrog.Action{
		{ID: "actn_1", Status: "current", Description: "Do the thing", RoleID: "role_1", Tags: []string{"tag1", "tag2"}},
		{ID: "actn_2", Status: "future", Description: "", RoleID: "role_2"},
	}}
	want := "actn_1  [current]  Do the thing\n" +
		"  role: role_1   tags: tag1, tag2\n" +
		"actn_2  [future]  —\n" +
		"  role: role_2\n"
	assertRender(t, ResourceActions, FormatFull, resp, want)
}

func TestRender_ProjectsFull_Golden(t *testing.T) {
	resp := glassfrog.MyProjectsResponse{Data: []glassfrog.Project{
		{ID: "proj_1", Status: "current", Description: "Build it", RoleID: "role_1", Tags: []string{"t"}, HasSubProjects: true, HasActions: false},
		{ID: "proj_2", Status: "future", Description: "", RoleID: "", HasSubProjects: false, HasActions: true},
	}}
	want := "proj_1  [current]  Build it\n" +
		"  role: role_1   sub-projects: yes   actions: no   tags: t\n" +
		"proj_2  [future]  —\n" +
		"  role: —   sub-projects: no   actions: yes\n"
	assertRender(t, ResourceProjects, FormatFull, resp, want)
}

// --- empty result sets: the explicit per-command line (both formats) --------

func TestRender_EmptyResultSets_ExplicitLine(t *testing.T) {
	cases := []struct {
		name     string
		resource Resource
		data     any
		want     string
	}{
		{"roles full", ResourceRoles, glassfrog.MyRolesResponse{}, "No roles.\n"},
		{"actions full", ResourceActions, glassfrog.MyActionsResponse{}, "No actions.\n"},
		{"projects full", ResourceProjects, glassfrog.MyProjectsResponse{}, "no projects\n"},
	}
	for _, tc := range cases {
		for _, format := range []Format{FormatFull, FormatCompact} {
			t.Run(tc.name+" "+string(format), func(t *testing.T) {
				assertRender(t, tc.resource, format, tc.data, tc.want)
			})
		}
	}
}

// --- compact: one line per record, ids present, nested collections as counts -

func TestRender_RolesCompact_OneLinePerRecord_Golden(t *testing.T) {
	resp := glassfrog.MyRolesResponse{Data: []glassfrog.Role{
		roleWith("role_1", "Marketing Lead", "Reach", []string{"Voice"}, []string{"Plan", "Measure"}),
		roleWith("role_2", "Facilitator", "", nil, []string{"Run"}),
	}}
	want := "role_1  Marketing Lead  domains=1  accountabilities=2\n" +
		"role_2  Facilitator  domains=0  accountabilities=1\n"
	assertRender(t, ResourceRoles, FormatCompact, resp, want)
}

// The `role` full template falls back to the actor id when an embedded
// assignment's actor object is absent (Name empty) — the actor is optional on a
// role's ?include=assignments, and a blank name would otherwise render a stray
// "-  (per_x)". A present name renders "Name (per_x)".
func TestRender_RoleFull_AssignmentActorNameFallsBackToID(t *testing.T) {
	// Decode the detail from JSON so the test is decoupled from the embedded
	// Actor struct's shape: one assignment carries an actor object, one does not.
	var doc glassfrog.RoleDocument
	body := `{"data":{"id":"role_1","name":"Lead","purpose":"p","assignments":[
	  {"id":"asgn_1","actor_id":"per_noname"},
	  {"id":"asgn_2","actor_id":"per_named","actor":{"id":"per_named","name":"Alice Smith","kind":"human"}}
	]}}`
	if err := json.Unmarshal([]byte(body), &doc); err != nil {
		t.Fatalf("fixture decode failed: %v", err)
	}
	view := RoleView{Detail: doc.Data, Requested: map[string]bool{"assignments": true}}
	out, err := Render(ResourceRole, FormatFull, view)
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}
	if !strings.Contains(out, "- per_noname\n") {
		t.Errorf("an empty actor name should fall back to the bare actor id, got:\n%s", out)
	}
	if strings.Contains(out, "-  (per_noname)") {
		t.Errorf("a blank actor name must not render a stray double space:\n%s", out)
	}
	if !strings.Contains(out, "- Alice Smith (per_named)") {
		t.Errorf("a present actor name should render `Name (id)`, got:\n%s", out)
	}
}

// compact surfaces the actor's id and renders the embedded roles collection as a
// count (roles=N) rather than enumerating it the way full does.
func TestRender_MeCompact_NestedCollectionAsCount(t *testing.T) {
	me := glassfrog.MeResponse{
		Actor:        glassfrog.Actor{ID: "agt_abc", Name: "Claude", Kind: "agent"},
		Organization: glassfrog.Organization{ID: "org_abc", Name: "Acme"},
		Membership:   glassfrog.Membership{AccessLevel: "normal"},
		Roles: []glassfrog.Role{
			{ID: "role_1", Name: "Lead"},
			{ID: "role_2", Name: "Rep"},
			{ID: "role_3", Name: "Treasurer"},
		},
	}
	want := "agt_abc  Claude (agent)  org=org_abc  access=normal  roles=3\n"
	assertRender(t, ResourceMe, FormatCompact, me, want)
}

// full enumerates a list while compact condenses it, but both account for exactly
// the same record set — one compact line per full record (the validation
// "full and compact cover the same records" property, exercised here per ADR-4
// since compact is not CLI-reachable).
func TestRender_FullAndCompact_SameRecordCount(t *testing.T) {
	resp := glassfrog.MyActionsResponse{Data: []glassfrog.Action{
		{ID: "actn_1", Status: "current", Description: "A", RoleID: "role_1"},
		{ID: "actn_2", Status: "current", Description: "B", RoleID: "role_1"},
		{ID: "actn_3", Status: "future", Description: "C", RoleID: "role_2"},
	}}
	full, err := Render(ResourceActions, FormatFull, resp)
	if err != nil {
		t.Fatalf("full render failed: %v", err)
	}
	compact, err := Render(ResourceActions, FormatCompact, resp)
	if err != nil {
		t.Fatalf("compact render failed: %v", err)
	}
	for _, id := range []string{"actn_1", "actn_2", "actn_3"} {
		if !strings.Contains(full, id) {
			t.Errorf("full output dropped record %q:\n%s", id, full)
		}
		if !strings.Contains(compact, id) {
			t.Errorf("compact output dropped record %q:\n%s", id, compact)
		}
	}
	// compact is one line per record.
	if got := strings.Count(strings.TrimRight(compact, "\n"), "\n") + 1; got != 3 {
		t.Errorf("compact should be one line per record (3), got %d:\n%s", got, compact)
	}
}

// --- registry exhaustiveness (PR #10 LEARNINGS) ----------------------------
//
// Every (Resource × Format) pair must resolve to a parsed template. A len guard
// plus a per-pair lookup means a dropped or misnamed template fails loud here,
// not silently at runtime.

func TestRegistry_AllBuiltinsResolve(t *testing.T) {
	wantCount := len(builtinResources) * len(builtinFormats)
	var got int
	for _, parsed := range templates.Templates() {
		if strings.HasSuffix(parsed.Name(), ".tmpl") {
			got++
		}
	}
	if got != wantCount {
		t.Errorf("parsed %d built-in templates, want %d (a template was dropped or an extra added)", got, wantCount)
	}
	for _, resource := range builtinResources {
		for _, format := range builtinFormats {
			name := templateName(resource, format)
			if templates.Lookup(name) == nil {
				t.Errorf("missing built-in template %q", name)
			}
		}
	}
}

// --- RenderError: unknown key, no partial output ----------------------------

func TestRender_UnknownFormat_ReturnsRenderError(t *testing.T) {
	out, err := Render(ResourceMe, Format("verbose"), glassfrog.MeResponse{})
	if out != "" {
		t.Errorf("an unknown format must return the empty string, got %q", out)
	}
	var re *RenderError
	if !errors.As(err, &re) {
		t.Fatalf("expected a *RenderError, got %v", err)
	}
	if re.Resource != ResourceMe || re.Format != Format("verbose") {
		t.Errorf("RenderError should carry the keys, got %+v", re)
	}
}

func TestRender_UnknownResource_ReturnsRenderError(t *testing.T) {
	out, err := Render(Resource("widgets"), FormatFull, glassfrog.MeResponse{})
	if out != "" || err == nil {
		t.Fatalf("an unknown resource must return (\"\", error), got (%q, %v)", out, err)
	}
	var re *RenderError
	if !errors.As(err, &re) {
		t.Fatalf("expected a *RenderError, got %v", err)
	}
}

// RenderError carries only the keys and the engine cause — never request data.
func TestRenderError_MessageNamesKeys(t *testing.T) {
	re := &RenderError{Resource: ResourceMe, Format: FormatFull, Err: errors.New("boom")}
	msg := re.Error()
	for _, want := range []string{"me", "full", "boom"} {
		if !strings.Contains(msg, want) {
			t.Errorf("RenderError message should contain %q, got %q", want, msg)
		}
	}
	if !errors.Is(re, re.Err) {
		// Unwrap exposes the cause for errors.Is/As discrimination.
		if errors.Unwrap(re) != re.Err {
			t.Errorf("RenderError should unwrap to its cause")
		}
	}
}

// --- helpers ---------------------------------------------------------------

func assertRender(t *testing.T, resource Resource, format Format, data any, want string) {
	t.Helper()
	got, err := Render(resource, format, data)
	if err != nil {
		t.Fatalf("Render(%s, %s) returned error: %v", resource, format, err)
	}
	if got != want {
		t.Errorf("Render(%s, %s) mismatch:\n--- got ---\n%q\n--- want ---\n%q", resource, format, got, want)
	}
}

// --- tree render (026): recursion, depth indentation, depth-boundary marker --

func strptr(s string) *string { return &s }

// TestRender_TreeFull_RecursiveIndent_Golden pins the depth-indented recursive
// render: two spaces per level, id+name+flags per node, the purpose line, and a
// true leaf (no children, has_subroles=false) rendering with nothing indented and
// no marker.
func TestRender_TreeFull_RecursiveIndent_Golden(t *testing.T) {
	root := glassfrog.TreeNode{
		ID: "role_a", Name: strptr("Anchor"), Purpose: strptr("Run it"),
		HasSubroles: true, Flags: []string{"structural"},
		Children: []glassfrog.TreeNode{{
			ID: "role_b", Name: strptr("Marketing"), HasSubroles: false,
			Children: []glassfrog.TreeNode{},
		}},
	}
	want := "Anchor (role_a) [structural]\n" +
		"  Purpose: Run it\n" +
		"  Marketing (role_b)\n" +
		"    Purpose: (no purpose set)\n"
	assertRender(t, ResourceTree, FormatFull, NewTreeView(root, nil), want)
}

// TestRender_TreeFull_DepthBoundaryMarker pins that a node with has_subroles=true
// but no children in the result carries the depth-boundary marker (distinct from
// a true leaf), and never invents a descendant count.
func TestRender_TreeFull_DepthBoundaryMarker(t *testing.T) {
	root := glassfrog.TreeNode{
		ID: "role_a", Name: strptr("Anchor"), HasSubroles: true,
		Children: []glassfrog.TreeNode{
			{ID: "role_b", Name: strptr("Branch"), HasSubroles: true}, // capped: has subroles, none present
			{ID: "role_c", Name: strptr("Leaf"), HasSubroles: false},  // true leaf
		},
	}
	got, err := Render(ResourceTree, FormatFull, NewTreeView(root, nil))
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if !strings.Contains(got, "Branch (role_b)  (+ subroles below depth)") {
		t.Errorf("a capped branch must carry the depth-boundary marker:\n%s", got)
	}
	if strings.Contains(got, "Leaf (role_c)  (+ subroles below depth)") {
		t.Errorf("a true leaf must NOT carry the marker:\n%s", got)
	}
	// No invented count of omitted descendants.
	for _, banned := range []string{"subroles below depth)1", "(1 ", "descendants"} {
		if strings.Contains(got, banned) {
			t.Errorf("the marker must not invent a descendant count (%q):\n%s", banned, got)
		}
	}
}

// TestRender_TreeFull_IncludeSectionsGuarded pins per-node include sections:
// requested resources render (indented under the node); unrequested ones are
// omitted entirely; "members" maps to the node's Fillers.
func TestRender_TreeFull_IncludeSectionsGuarded(t *testing.T) {
	root := glassfrog.TreeNode{
		ID: "role_a", Name: strptr("Anchor"), HasSubroles: false,
		Accountabilities: []glassfrog.Accountability{{Description: "Holding the whole"}},
		Domains:          []glassfrog.Domain{{Description: "The company"}},
		Fillers:          []glassfrog.Actor{{ID: "per_1", Name: "Alice"}},
	}
	req := map[string]bool{"accountabilities": true, "members": true}
	got, err := Render(ResourceTree, FormatFull, NewTreeView(root, req))
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	for _, want := range []string{"  Accountabilities:", "    - Holding the whole", "  Members:", "    - Alice (per_1)"} {
		if !strings.Contains(got, want) {
			t.Errorf("requested include section missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "Domains:") {
		t.Errorf("an unrequested section (domains) must be omitted:\n%s", got)
	}
}

// TestRender_TreeCompact_DistinguishesCappedFromLeaf pins the compact form: a
// depth-capped node (children=0 has_subroles=yes) is distinct from a true leaf
// (children=0 has_subroles=no), with depth shown by indentation.
func TestRender_TreeCompact_DistinguishesCappedFromLeaf(t *testing.T) {
	root := glassfrog.TreeNode{
		ID: "role_a", Name: strptr("Anchor"), HasSubroles: true, Flags: []string{"structural"},
		Children: []glassfrog.TreeNode{
			{ID: "role_b", Name: strptr("Branch"), HasSubroles: true},
			{ID: "role_c", Name: strptr("Leaf"), HasSubroles: false},
		},
	}
	want := "role_a  Anchor  children=2  has_subroles=yes  flags=structural\n" +
		"  role_b  Branch  children=0  has_subroles=yes  flags=—\n" +
		"  role_c  Leaf  children=0  has_subroles=no  flags=—\n"
	assertRender(t, ResourceTree, FormatCompact, NewTreeView(root, nil), want)
}

// --- subroles render (026): list of children, empty marker -------------------

func TestRender_SubrolesFull_Empty_Golden(t *testing.T) {
	assertRender(t, ResourceSubroles, FormatFull, SubrolesView{}, "No subroles.\n")
}

func TestRender_SubrolesCompact_Empty_Golden(t *testing.T) {
	assertRender(t, ResourceSubroles, FormatCompact, SubrolesView{}, "No subroles.\n")
}

func TestRender_SubrolesFull_RendersEachChildWithGuardedIncludes(t *testing.T) {
	children := []glassfrog.RoleDetail{
		{
			Role:     glassfrog.Role{ID: "role_b", Name: "Press Officer", Purpose: "Press"},
			Policies: []glassfrog.Policy{{Title: "Two approvals"}},
		},
	}
	view := SubrolesView{Children: children, Requested: map[string]bool{"policies": true}}
	got, err := Render(ResourceSubroles, FormatFull, view)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	for _, want := range []string{"Press Officer (role_b)", "Purpose: Press", "  Policies:", "    - Two approvals"} {
		if !strings.Contains(got, want) {
			t.Errorf("subroles full missing %q:\n%s", want, got)
		}
	}
	// Subroles include not requested → its section is omitted.
	if strings.Contains(got, "  Subroles:") {
		t.Errorf("an unrequested include section must be omitted:\n%s", got)
	}
}

func TestRender_SubrolesCompact_OneLinePerChild_Golden(t *testing.T) {
	children := []glassfrog.RoleDetail{
		{Role: glassfrog.Role{ID: "role_b", Name: "Press Officer", HasSubroles: true}},
		{Role: glassfrog.Role{ID: "role_c", Name: "Treasurer", HasSubroles: false}},
	}
	want := "role_b  Press Officer  has_subroles=yes\n" +
		"role_c  Treasurer  has_subroles=no\n"
	assertRender(t, ResourceSubroles, FormatCompact, SubrolesView{Children: children}, want)
}

func roleWith(id, name, purpose string, domains, accountabilities []string) glassfrog.Role {
	r := glassfrog.Role{ID: id, Name: name, Purpose: purpose}
	for _, d := range domains {
		r.Domains = append(r.Domains, glassfrog.Domain{Description: d})
	}
	for _, a := range accountabilities {
		r.Accountabilities = append(r.Accountabilities, glassfrog.Accountability{Description: a})
	}
	return r
}
