package render

import (
	"strings"
	"testing"

	"github.com/Luscii/cli-glassfrog/internal/glassfrog"
)

// --- assignments render (050): actor-scoped homogeneous assignment list -------
//
// The actor-end mirror of the `fillers` key: one render key over a flat
// homogeneous list — every row is the same Assignment shape — but each row leads
// with the FILLED ROLE (the role id + name + its purpose/parent context the
// actor-end default ?include=role embeds — plan ADR-2), then the assignment's
// governance context (focus + election). The full golden pins the per-row block;
// the compact golden pins the one-line-per-assignment shape. Both pin the `no
// assignments` empty line. All four nullable fields (role purpose/parent, focus,
// election) get an explicit-absence marker — never a fabricated value or an empty
// gap (CONSTITUTION VIII).

// assignmentWithRole builds an Assignment carrying the embedded role the actor-end
// read returns, plus the assignment's own focus/election.
func assignmentWithRole(roleID, roleName, purpose, parent, focus, elected string) glassfrog.Assignment {
	a := glassfrog.Assignment{Focus: focus, ElectedUntil: elected}
	a.Role.ID = roleID
	a.Role.Name = roleName
	a.Role.Purpose = purpose
	a.Role.ParentRoleID = parent
	return a
}

// mixedAssignmentsView is a fully-populated assignment (purpose + parent + focus +
// election) followed by a top-level role with none of those — exercising both the
// present and the absent case for all four nullable fields in one view.
func mixedAssignmentsView() AssignmentsView {
	return AssignmentsView{Data: []glassfrog.Assignment{
		assignmentWithRole("role_a", "Marketing Lead", "A market that knows us", "role_parent", "Keep the lights on", "2026-12-31"),
		assignmentWithRole("role_b", "General Company Circle", "", "", "", ""),
	}}
}

func TestRender_AssignmentsFull_Mixed_Golden(t *testing.T) {
	want := "role_a\n" +
		"  Role:           Marketing Lead\n" +
		"  Purpose:        A market that knows us\n" +
		"  Parent role:    role_parent\n" +
		"  Focus:          Keep the lights on\n" +
		"  Elected until:  2026-12-31\n" +
		"role_b\n" +
		"  Role:           General Company Circle\n" +
		"  Purpose:        (none)\n" +
		"  Parent role:    (top-level)\n" +
		"  Focus:          (none)\n" +
		"  Elected until:  (not an elected seat)\n"
	assertRender(t, ResourceAssignments, FormatFull, mixedAssignmentsView(), want)
}

func TestRender_AssignmentsCompact_Mixed_Golden(t *testing.T) {
	want := "role_a  Marketing Lead  — focus: Keep the lights on; elected until: 2026-12-31\n" +
		"role_b  General Company Circle  — focus: —; elected until: —\n"
	assertRender(t, ResourceAssignments, FormatCompact, mixedAssignmentsView(), want)
}

// An empty list (the actor fills no role) renders the explicit `no assignments`
// line under both human formats — a valid empty answer, not an error.
func TestRender_AssignmentsFull_Empty_Golden(t *testing.T) {
	assertRender(t, ResourceAssignments, FormatFull, AssignmentsView{}, "no assignments\n")
}

func TestRender_AssignmentsCompact_Empty_Golden(t *testing.T) {
	assertRender(t, ResourceAssignments, FormatCompact, AssignmentsView{Data: nil}, "no assignments\n")
}

// A blank (present-but-whitespace) or absent focus renders the `(none)` marker, a
// non-elected assignment renders `(not an elected seat)`, a role with no purpose
// renders `(none)`, and a top-level role renders `(top-level)` — never a fabricated
// value (CONSTITUTION VIII).
func TestRender_AssignmentsFull_AbsentFieldsShowMarkers(t *testing.T) {
	view := AssignmentsView{Data: []glassfrog.Assignment{
		assignmentWithRole("role_blank", "Top Role", "   ", "   ", "   ", ""),
	}}
	got, err := Render(ResourceAssignments, FormatFull, view)
	if err != nil {
		t.Fatalf("render returned error: %v", err)
	}
	for _, want := range []string{
		"Purpose:        (none)",
		"Parent role:    (top-level)",
		"Focus:          (none)",
		"Elected until:  (not an elected seat)",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("absent field must render its marker, missing %q:\n%s", want, got)
		}
	}
}

// Focus and purpose are free text — rendered verbatim, never truncated or reflowed
// (CONSTITUTION VI), like fillers.full / projects.full.
func TestRender_AssignmentsFull_FocusAndPurposeRenderedVerbatim(t *testing.T) {
	longFocus := "A very long focus statement that the renderer must never truncate or reflow across lines"
	longPurpose := "A purpose paragraph long enough that any truncation or reflow would be obvious to the reader"
	view := AssignmentsView{Data: []glassfrog.Assignment{
		assignmentWithRole("role_long", "Role", longPurpose, "role_parent", longFocus, ""),
	}}
	got, err := Render(ResourceAssignments, FormatFull, view)
	if err != nil {
		t.Fatalf("render returned error: %v", err)
	}
	for _, want := range []string{longFocus, longPurpose} {
		if !strings.Contains(got, want) {
			t.Errorf("free text must be rendered verbatim, missing %q:\n%s", want, got)
		}
	}
}

// The row leads with the role id and shows the filled role's name — the answer to
// "which roles does this actor fill?" (the filled-role-name scenario).
func TestRender_AssignmentsFull_LeadsWithRoleIDAndName(t *testing.T) {
	view := AssignmentsView{Data: []glassfrog.Assignment{
		assignmentWithRole("role_xyz", "Press Officer", "Press that lands", "role_parent", "", ""),
	}}
	got, err := Render(ResourceAssignments, FormatFull, view)
	if err != nil {
		t.Fatalf("render returned error: %v", err)
	}
	if !strings.HasPrefix(got, "role_xyz\n") {
		t.Errorf("the row must lead with the role id:\n%s", got)
	}
	if !strings.Contains(got, "Role:           Press Officer") {
		t.Errorf("the filled role name must be shown:\n%s", got)
	}
}
