package render

import (
	"strings"
	"testing"

	"github.com/Luscii/cli-glassfrog/internal/glassfrog"
)

// --- fillers render (047): role-scoped homogeneous assignment list ----------
//
// One render key over a flat homogeneous list — every row is the same Assignment
// shape (reused as-is, plan ADR-2; like actors, unlike search's heterogeneous
// rows). Each row leads with the FILLING actor (the per_/agt_ id + kind badge +
// name — the "whom to contact?" answer), then the assignment's governance context
// (focus + election). The full golden pins the per-row block; the compact golden
// pins the one-line-per-filler shape. Both pin the `no fillers` empty line. The
// nullable focus/elected_until each get an explicit-absence marker — never a
// fabricated value (CONSTITUTION VIII). A person and an agent filler are visually
// distinguished by id prefix and kind badge.

func assignment(actorID, name, kind, focus, elected string) glassfrog.Assignment {
	a := glassfrog.Assignment{Focus: focus, ElectedUntil: elected}
	a.Actor.ID = actorID
	a.Actor.Name = name
	a.Actor.Kind = kind
	return a
}

// mixedFillersView is a person (with a focus + an election) followed by an agent
// (with neither) — exercising both id prefixes, both kind badges, and both the
// present and absent focus/election in one view.
func mixedFillersView() FillersView {
	return FillersView{Data: []glassfrog.Assignment{
		assignment("per_0123", "Alice Smith", "human", "Keep the lights on", "2026-12-31"),
		assignment("agt_0456", "Claude", "agent", "", ""),
	}}
}

func TestRender_FillersFull_MixedKinds_Golden(t *testing.T) {
	want := "per_0123  [human]\n" +
		"  Name:           Alice Smith\n" +
		"  Focus:          Keep the lights on\n" +
		"  Elected until:  2026-12-31\n" +
		"agt_0456  [agent]\n" +
		"  Name:           Claude\n" +
		"  Focus:          (none)\n" +
		"  Elected until:  (not an elected seat)\n"
	assertRender(t, ResourceFillers, FormatFull, mixedFillersView(), want)
}

func TestRender_FillersCompact_MixedKinds_Golden(t *testing.T) {
	want := "per_0123  [human]  Alice Smith  — focus: Keep the lights on; elected until: 2026-12-31\n" +
		"agt_0456  [agent]  Claude  — focus: —; elected until: —\n"
	assertRender(t, ResourceFillers, FormatCompact, mixedFillersView(), want)
}

// An empty list (the role is filled by no actor) renders the explicit `no fillers`
// line under both human formats — a valid empty answer, not an error.
func TestRender_FillersFull_Empty_Golden(t *testing.T) {
	assertRender(t, ResourceFillers, FormatFull, FillersView{}, "no fillers\n")
}

func TestRender_FillersCompact_Empty_Golden(t *testing.T) {
	assertRender(t, ResourceFillers, FormatCompact, FillersView{Data: nil}, "no fillers\n")
}

// A present focus + election (the capacity context) is projected under the default
// human format; an absent one shows its explicit-absence marker. This pins the
// scenario "A filler shows its focus and election expiry" / "Focus and election are
// projected, not dropped".
func TestRender_FillersFull_FocusAndElectionProjected(t *testing.T) {
	view := FillersView{Data: []glassfrog.Assignment{
		assignment("per_0123", "Alice Smith", "human", "Keep the lights on", "2026-12-31"),
	}}
	got, err := Render(ResourceFillers, FormatFull, view)
	if err != nil {
		t.Fatalf("render returned error: %v", err)
	}
	for _, want := range []string{"Keep the lights on", "2026-12-31"} {
		if !strings.Contains(got, want) {
			t.Errorf("focus/election must be projected, missing %q:\n%s", want, got)
		}
	}
}

// A blank (present-but-whitespace) or absent focus renders the `(none)` marker, and
// a non-elected filling renders `(not an elected seat)` — never a fabricated value
// or an empty gap (CONSTITUTION VIII).
func TestRender_FillersFull_AbsentFocusAndElectionShowMarkers(t *testing.T) {
	view := FillersView{Data: []glassfrog.Assignment{
		assignment("per_blank", "Bob", "human", "   ", ""),
	}}
	got, err := Render(ResourceFillers, FormatFull, view)
	if err != nil {
		t.Fatalf("render returned error: %v", err)
	}
	if !strings.Contains(got, "Focus:          (none)") {
		t.Errorf("a blank focus must render the (none) marker:\n%s", got)
	}
	if !strings.Contains(got, "Elected until:  (not an elected seat)") {
		t.Errorf("a non-elected filling must render the (not an elected seat) marker:\n%s", got)
	}
}

// A focus is free text — rendered verbatim, never truncated or reflowed
// (CONSTITUTION VI), like projects.full and tensions.full.
func TestRender_FillersFull_FocusRenderedVerbatim(t *testing.T) {
	long := "A very long focus statement that the renderer must never truncate or reflow across lines"
	view := FillersView{Data: []glassfrog.Assignment{
		assignment("per_long", "Alice", "human", long, ""),
	}}
	got, err := Render(ResourceFillers, FormatFull, view)
	if err != nil {
		t.Fatalf("render returned error: %v", err)
	}
	if !strings.Contains(got, long) {
		t.Errorf("the focus must be rendered verbatim, never truncated:\n%s", got)
	}
}

// A blank or absent actor name renders the `—` marker rather than a fabricated
// value — the repo's trim-emptiness convention (CONSTITUTION VIII), as actors.
func TestRender_FillersFull_BlankNameShowsMarker(t *testing.T) {
	view := FillersView{Data: []glassfrog.Assignment{
		assignment("per_blank", "   ", "human", "", ""),
	}}
	got, err := Render(ResourceFillers, FormatFull, view)
	if err != nil {
		t.Fatalf("render returned error: %v", err)
	}
	if !strings.Contains(got, "Name:           —") {
		t.Errorf("a blank name must render the — marker:\n%s", got)
	}
}
