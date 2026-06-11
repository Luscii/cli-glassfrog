package render

import (
	"strings"
	"testing"

	"github.com/Luscii/cli-glassfrog/internal/glassfrog"
)

// --- role-scoped projects list render (038): reuses the landed `projects` key --
//
// The role-addressable list (ProjectsView) renders through the SAME templates the
// 014 /me/projects projection uses (the templates range over .Data), so these
// goldens confirm parity — a ProjectsView produces byte-equivalent output to the
// MyProjectsResponse goldens in render_test.go.

func TestRender_ProjectsView_Full_Golden(t *testing.T) {
	view := ProjectsView{Data: []glassfrog.Project{
		{ID: "proj_1", Status: "current", Description: "Build it", RoleID: "role_1", Tags: []string{"t"}, HasSubProjects: true, HasActions: false},
		{ID: "proj_2", Status: "future", Description: "", RoleID: "", HasSubProjects: false, HasActions: true},
	}}
	want := "proj_1  [current]  Build it\n" +
		"  role: role_1   sub-projects: yes   actions: no   tags: t\n" +
		"proj_2  [future]  —\n" +
		"  role: —   sub-projects: no   actions: yes\n"
	assertRender(t, ResourceProjects, FormatFull, view, want)
}

func TestRender_ProjectsView_Empty_Golden(t *testing.T) {
	for _, format := range []Format{FormatFull, FormatCompact} {
		assertRender(t, ResourceProjects, format, ProjectsView{}, "no projects\n")
	}
}

func TestRender_ProjectsView_Compact_Golden(t *testing.T) {
	view := ProjectsView{Data: []glassfrog.Project{
		{ID: "proj_1", Status: "current", Description: "Build it"},
		{ID: "proj_2", Status: "future", Description: ""},
	}}
	want := "proj_1  [current]  Build it\n" +
		"proj_2  [future]  —\n"
	assertRender(t, ResourceProjects, FormatCompact, view, want)
}

// --- single project render (038) -------------------------------------------

// TestRender_ProjectFull_Golden pins the single full detail block: the header
// line, every labelled field, present timestamps/link, tags joined, and the
// presence flags.
func TestRender_ProjectFull_Golden(t *testing.T) {
	view := ProjectView{Project: glassfrog.Project{
		ID: "proj_0123", Status: "current", Description: "Ship the new onboarding flow",
		RoleID: "role_0123", ParentProjectID: "proj_root", Tags: []string{"q3", "growth"},
		HasSubProjects: true, HasActions: true,
		CreatedAt: "2024-01-02T03:04:05Z", UpdatedAt: "2024-05-06T07:08:09Z",
		Link: "https://example.com/p/123", Note: "Blocked on design review",
	}}
	want := "proj_0123  [current]\n" +
		"  Description:   Ship the new onboarding flow\n" +
		"  Role:          role_0123\n" +
		"  Parent:        proj_root\n" +
		"  Sub-projects:  yes\n" +
		"  Actions:       yes\n" +
		"  Tags:          q3, growth\n" +
		"  Created:       2024-01-02T03:04:05Z\n" +
		"  Updated:       2024-05-06T07:08:09Z\n" +
		"  Link:          https://example.com/p/123\n" +
		"  Note:          Blocked on design review\n"
	assertRender(t, ResourceProject, FormatFull, view, want)
}

// TestRender_ProjectFull_NullFields_Markers pins the explicit-absence markers for
// every nullable field: a null role_id (individual initiative), null parent (top
// level), empty description, empty tags, missing timestamps, null link/note — none
// renders as `<no value>` or a fabricated value.
func TestRender_ProjectFull_NullFields_Markers(t *testing.T) {
	view := ProjectView{Project: glassfrog.Project{
		ID: "proj_solo", Status: "scheduled",
		// role_id, parent, description, tags, timestamps, link, note all absent.
	}}
	got, err := Render(ResourceProject, FormatFull, view)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	for _, want := range []string{
		"  Description:   —",
		"  Role:          (individual initiative — no role)",
		"  Parent:        (top-level — no parent)",
		"  Sub-projects:  no",
		"  Actions:       no",
		"  Tags:          (none)",
		"  Created:       (unknown)",
		"  Updated:       (unknown)",
		"  Link:          (none)",
		"  Note:          (none)",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("a missing field must render its explicit-absence marker %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "<no value>") {
		t.Errorf("missingkey=error must never leak <no value>:\n%s", got)
	}
}

// TestRender_ProjectFull_NoteVerbatim pins CONSTITUTION VI: a long free-text note
// is rendered verbatim — neither truncated nor reflowed.
func TestRender_ProjectFull_NoteVerbatim(t *testing.T) {
	longNote := strings.Repeat("This is a long operator note that the renderer must not wrap or cut. ", 50)
	view := ProjectView{Project: glassfrog.Project{
		ID: "proj_long", Status: "current", Description: "x", Note: longNote,
	}}
	got, err := Render(ResourceProject, FormatFull, view)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if !strings.Contains(got, longNote) {
		t.Errorf("the full note must be rendered verbatim (not truncated/reflowed):\n%s", got)
	}
}

// TestRender_ProjectCompact_Golden pins the single compact line (detail omitted).
func TestRender_ProjectCompact_Golden(t *testing.T) {
	view := ProjectView{Project: glassfrog.Project{
		ID: "proj_0123", Status: "current", Description: "Ship the new onboarding flow",
		RoleID: "role_0123", Note: "ignored in compact",
	}}
	want := "proj_0123  [current]  Ship the new onboarding flow\n"
	assertRender(t, ResourceProject, FormatCompact, view, want)
}

// TestRender_ProjectCompact_EmptyDescriptionMarker pins the em-dash marker for a
// blank description in compact.
func TestRender_ProjectCompact_EmptyDescriptionMarker(t *testing.T) {
	view := ProjectView{Project: glassfrog.Project{ID: "proj_x", Status: "future", Description: ""}}
	want := "proj_x  [future]  —\n"
	assertRender(t, ResourceProject, FormatCompact, view, want)
}
