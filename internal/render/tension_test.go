package render

import (
	"strings"
	"testing"

	"github.com/Luscii/cli-glassfrog/internal/glassfrog"
)

// fullTension is a representative created tension with every field populated, so
// the full golden pins each label and value.
func fullTension() glassfrog.Tension {
	return glassfrog.Tension{
		ID:           "ten_1",
		Type:         "tension",
		Body:         "We ship faster than we update the roadmap.",
		Status:       "unprocessed",
		RoleID:       "role_0123",
		SensedByID:   "per_0123",
		CreatedAt:    "2026-01-01T00:00:00Z",
		UpdatedAt:    "2026-01-02T00:00:00Z",
		Label:        "Roadmap drift",
		MeetingType:  "governance",
		ParentRoleID: "role_9",
	}
}

// TestRender_TensionFull_Golden pins the full projection: the id + status header,
// the verbatim body, and every detail line (label, sensing role, sensed by,
// meeting type, parent role, timestamps).
func TestRender_TensionFull_Golden(t *testing.T) {
	want := "ten_1  [unprocessed]\n" +
		"  Body:          We ship faster than we update the roadmap.\n" +
		"  Label:         Roadmap drift\n" +
		"  Sensing role:  role_0123\n" +
		"  Sensed by:     per_0123\n" +
		"  Meeting type:  governance\n" +
		"  Parent role:   role_9\n" +
		"  Created:       2026-01-01T00:00:00Z\n" +
		"  Updated:       2026-01-02T00:00:00Z\n"
	assertRender(t, ResourceTension, FormatFull, TensionView{Tension: fullTension()}, want)
}

// TestRender_TensionFull_NullFieldsShowMarkers pins that every nullable field
// (label, role_id, sensed_by_id, meeting_type, parent_role_id) renders its
// explicit-absence marker (none) — never <no value> or a blank — and absent
// timestamps render (unknown). The body (required, present) renders verbatim.
func TestRender_TensionFull_NullFieldsShowMarkers(t *testing.T) {
	ten := glassfrog.Tension{ID: "ten_1", Status: "unprocessed", Body: "a tension"}
	want := "ten_1  [unprocessed]\n" +
		"  Body:          a tension\n" +
		"  Label:         (none)\n" +
		"  Sensing role:  (none)\n" +
		"  Sensed by:     (none)\n" +
		"  Meeting type:  (none)\n" +
		"  Parent role:   (none)\n" +
		"  Created:       (unknown)\n" +
		"  Updated:       (unknown)\n"
	assertRender(t, ResourceTension, FormatFull, TensionView{Tension: ten}, want)
}

// TestRender_TensionCompact_Golden pins the one-line compact form: id, status
// badge, and the verbatim body. The ten_ id is always present (the load-bearing
// handle).
func TestRender_TensionCompact_Golden(t *testing.T) {
	want := "ten_1  [unprocessed]  We ship faster than we update the roadmap.\n"
	assertRender(t, ResourceTension, FormatCompact, TensionView{Tension: fullTension()}, want)
}

// TestRender_TensionBodyVerbatim pins that a long, multi-line free-text body is
// rendered verbatim — neither truncated nor reflowed (CONSTITUTION VI) — in both
// formats.
func TestRender_TensionBodyVerbatim(t *testing.T) {
	body := strings.Repeat("This sentence is part of a deliberately long tension body. ", 40) +
		"\nIt also spans\nmultiple lines."
	ten := fullTension()
	ten.Body = body
	for _, format := range []Format{FormatFull, FormatCompact} {
		got, err := Render(ResourceTension, format, TensionView{Tension: ten})
		if err != nil {
			t.Fatalf("render %s: %v", format, err)
		}
		if !strings.Contains(got, body) {
			t.Errorf("%s must render the body verbatim (no truncation/reflow):\n%s", format, got)
		}
	}
}
