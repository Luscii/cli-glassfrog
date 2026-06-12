package render

import (
	"strings"
	"testing"

	"github.com/Luscii/cli-glassfrog/internal/glassfrog"
)

// --- role-scoped tensions list render (043): the new plural `tensions` key -----
//
// The plural list sibling of the landed singular `tension` key (042). These
// goldens pin both templates, the empty-set line, the explicit-absence markers for
// the nullable label/role_id, and the verbatim body (CONSTITUTION VI).

// TestRender_TensionsView_Full_Golden pins the full list block per tension: the
// id + status + label header, the verbatim body on its own indented line, and the
// indented sensing-role line.
func TestRender_TensionsView_Full_Golden(t *testing.T) {
	view := TensionsView{Data: []glassfrog.Tension{
		{ID: "ten_1", Status: "unprocessed", Label: "Roadmap drift", Body: "We ship faster than we update the roadmap.", RoleID: "role_0123"},
		{ID: "ten_2", Status: "processed", Label: "", Body: "Some other tension", RoleID: ""},
	}}
	want := "ten_1  [unprocessed]  Roadmap drift\n" +
		"  We ship faster than we update the roadmap.\n" +
		"  sensing role: role_0123\n" +
		"ten_2  [processed]  —\n" +
		"  Some other tension\n" +
		"  sensing role: —\n"
	assertRender(t, ResourceTensions, FormatFull, view, want)
}

// TestRender_TensionsView_Compact_Golden pins the one-line-per-tension compact
// form: id, status badge, and the label (falling back to the body when no label).
func TestRender_TensionsView_Compact_Golden(t *testing.T) {
	view := TensionsView{Data: []glassfrog.Tension{
		{ID: "ten_1", Status: "unprocessed", Label: "Roadmap drift", Body: "We ship faster than we update the roadmap."},
		{ID: "ten_2", Status: "processed", Label: "", Body: "Some other tension"},
	}}
	want := "ten_1  [unprocessed]  Roadmap drift\n" +
		"ten_2  [processed]  Some other tension\n"
	assertRender(t, ResourceTensions, FormatCompact, view, want)
}

// TestRender_TensionsView_Empty_Golden pins the explicit `no tensions` empty line
// in both formats (an empty list is a valid answer, not <no value> or a blank).
func TestRender_TensionsView_Empty_Golden(t *testing.T) {
	for _, format := range []Format{FormatFull, FormatCompact} {
		assertRender(t, ResourceTensions, format, TensionsView{}, "no tensions\n")
	}
}

// TestRender_TensionsView_Full_NullFieldsMarkers pins the explicit-absence markers
// for a null/blank label and a null role_id — neither renders as <no value> or a
// blank.
func TestRender_TensionsView_Full_NullFieldsMarkers(t *testing.T) {
	view := TensionsView{Data: []glassfrog.Tension{
		{ID: "ten_solo", Status: "unprocessed", Body: "a tension"},
	}}
	got, err := Render(ResourceTensions, FormatFull, view)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	for _, want := range []string{
		"ten_solo  [unprocessed]  —\n",
		"  a tension\n",
		"  sensing role: —\n",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("a missing field must render its explicit-absence marker %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "<no value>") {
		t.Errorf("missingkey=error must never leak <no value>:\n%s", got)
	}
}

// TestRender_TensionsView_BodyVerbatim pins CONSTITUTION VI: a long free-text body
// is rendered verbatim in the full list — neither truncated nor reflowed.
func TestRender_TensionsView_BodyVerbatim(t *testing.T) {
	body := strings.Repeat("This is a long tension body the renderer must not wrap or cut. ", 50)
	view := TensionsView{Data: []glassfrog.Tension{
		{ID: "ten_long", Status: "unprocessed", Body: body, RoleID: "role_1"},
	}}
	got, err := Render(ResourceTensions, FormatFull, view)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if !strings.Contains(got, body) {
		t.Errorf("the full body must be rendered verbatim (not truncated/reflowed):\n%s", got)
	}
}
