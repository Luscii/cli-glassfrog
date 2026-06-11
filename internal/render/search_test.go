package render

import (
	"strings"
	"testing"

	"github.com/Luscii/cli-glassfrog/internal/glassfrog"
)

// --- search render (041): heterogeneous relevance-ordered list -------------
//
// One render key over a deliberately mixed-type list. The full golden pins the
// per-row `type` badge, the rank, the always-present Excerpt line with its `—`
// absence marker, the Role line emitted only when role_id is present, the
// blank-line block separator, and the relevance (input) order preserved exactly.

func sptr(s string) *string { return &s }

// twoTypeMixedView is a role hit (every field populated) followed by a note hit
// (null excerpt, no role_id) — exercising both nullable-as-absent paths in one
// view, in descending-rank (relevance) order.
func twoTypeMixedView() SearchView {
	return NewSearchView([]glassfrog.SearchResult{
		{Type: "role", ID: "role_0123", Title: "Onboarding Lead", Rank: 0.99, Excerpt: sptr("owns onboarding"), RoleID: sptr("role_0123")},
		{Type: "note", ID: "note_0456", Title: "Onboarding retro", Rank: 0.8, Excerpt: nil, RoleID: nil},
	})
}

func TestRender_SearchFull_MixedTypes_Golden(t *testing.T) {
	want := "[role] Onboarding Lead (role_0123)  rank 0.99\n" +
		"  Excerpt: owns onboarding\n" +
		"  Role: role_0123\n" +
		"\n" +
		"[note] Onboarding retro (note_0456)  rank 0.8\n" +
		"  Excerpt: —\n"
	assertRender(t, ResourceSearch, FormatFull, twoTypeMixedView(), want)
}

func TestRender_SearchCompact_MixedTypes_Golden(t *testing.T) {
	want := "[role]  role_0123  Onboarding Lead  rank=0.99\n" +
		"[note]  note_0456  Onboarding retro  rank=0.8\n"
	assertRender(t, ResourceSearch, FormatCompact, twoTypeMixedView(), want)
}

// An empty result set renders the explicit `No results.` line (zero matches is a
// valid empty answer, not an error) under both human formats.
func TestRender_SearchFull_Empty_Golden(t *testing.T) {
	assertRender(t, ResourceSearch, FormatFull, NewSearchView(nil), "No results.\n")
}

func TestRender_SearchCompact_Empty_Golden(t *testing.T) {
	assertRender(t, ResourceSearch, FormatCompact, NewSearchView(nil), "No results.\n")
}

// A null excerpt renders as the `—` marker (never invented text); a blank
// (present-but-whitespace) excerpt is treated the same way (the repo's
// trim-emptiness convention). A null role_id omits the Role line entirely.
func TestRender_SearchFull_NullableFieldsRenderAsAbsent(t *testing.T) {
	view := NewSearchView([]glassfrog.SearchResult{
		{Type: "policy", ID: "plcy_1", Title: "Spend policy", Rank: 0.5, Excerpt: nil, RoleID: nil},
		{Type: "actor", ID: "per_1", Title: "Alice", Rank: 0.4, Excerpt: sptr("   "), RoleID: sptr("")},
	})
	got, err := Render(ResourceSearch, FormatFull, view)
	if err != nil {
		t.Fatalf("render returned error: %v", err)
	}
	// Both rows show the em-dash excerpt marker and omit the Role line.
	if strings.Count(got, "Excerpt: —") != 2 {
		t.Errorf("both a null and a blank excerpt should render the — marker:\n%s", got)
	}
	if strings.Contains(got, "Role:") {
		t.Errorf("a null/blank role_id must omit the Role line entirely:\n%s", got)
	}
}

// NewSearchView preserves input (relevance) order exactly — never re-sorts or
// re-orders by rank, type, or id (plan ADR-2).
func TestNewSearchView_PreservesInputOrder(t *testing.T) {
	// Deliberately NOT rank-sorted: a low-rank row precedes a high-rank one, so a
	// re-sort would reorder them.
	view := NewSearchView([]glassfrog.SearchResult{
		{Type: "domain", ID: "dom_low", Title: "Low", Rank: 0.1},
		{Type: "role", ID: "role_high", Title: "High", Rank: 0.9},
	})
	if view.Rows[0].ID != "dom_low" || view.Rows[1].ID != "role_high" {
		t.Errorf("input order must be preserved, got %q then %q", view.Rows[0].ID, view.Rows[1].ID)
	}
}
