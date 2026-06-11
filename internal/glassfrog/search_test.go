package glassfrog

import (
	"encoding/json"
	"testing"
)

// searchPageFixture is a representative GET /search page: the {data, meta} list
// envelope decoded via the generic Page[SearchResult]. It mixes all eight resource
// types across rows (a role/note/project/action/skill/actor/policy/domain stream),
// carries a meta.pagination block reporting a next page, and includes extra/unknown
// fields the decode must tolerate. It uses the API's snake_case names throughout so
// a missing JSON tag fails loud here. The note row has excerpt:null and no role_id
// (both nil); the role row populates both excerpt and role_id; the rows are listed
// in descending rank so decode-order-equals-response-order can be asserted.
const searchPageFixture = `{
  "data": [
    {"type": "role", "id": "role_0123", "title": "Onboarding Lead", "excerpt": "owns onboarding", "rank": 0.99, "role_id": "role_0123", "unexpected_row_field": "ignored"},
    {"type": "note", "id": "note_0456", "title": "Onboarding retro", "excerpt": null, "rank": 0.80},
    {"type": "project", "id": "proj_0789", "title": "Rebuild onboarding", "excerpt": "Q2 project", "rank": 0.70, "role_id": "role_0123"},
    {"type": "action", "id": "actn_0abc", "title": "Draft onboarding email", "excerpt": "todo", "rank": 0.60, "role_id": "role_0123"},
    {"type": "skill", "id": "skil_0def", "title": "Onboarding facilitation", "excerpt": null, "rank": 0.50},
    {"type": "actor", "id": "per_0aaa", "title": "Alice (onboarding buddy)", "excerpt": null, "rank": 0.40},
    {"type": "policy", "id": "plcy_0bbb", "title": "Onboarding policy", "excerpt": "rule", "rank": 0.30, "role_id": "role_0123"},
    {"type": "domain", "id": "dom_0ccc", "title": "Onboarding materials", "excerpt": null, "rank": 0.20, "role_id": "role_0123"}
  ],
  "meta": {
    "pagination": {
      "per_page": 100,
      "has_next_page": true,
      "next_cursor": "cursor_2",
      "unexpected_pagination_field": "ignored"
    }
  },
  "unexpected_top_level": {"anything": [1, 2, 3]}
}`

// TestSearchResultPageDecodesSnakeCaseFields pins every snake_case JSON tag on the
// SearchResult row, decoded through the generic Page[SearchResult] envelope, plus
// the reused Pagination. encoding/json is case-insensitive but does not bridge
// underscores, so an untagged RoleID would silently never bind to role_id — this
// feeds the real snake_case payload and asserts each field decoded, that the mixed
// type values are preserved, and that decode order equals response order.
func TestSearchResultPageDecodesSnakeCaseFields(t *testing.T) {
	var page Page[SearchResult]
	if err := json.Unmarshal([]byte(searchPageFixture), &page); err != nil {
		t.Fatalf("decoding the /search page fixture failed: %v", err)
	}

	// The generic Page[T] reuses the shared Pagination (012); a multi-page fixture
	// sets has_next_page true and carries the cursor.
	p := page.Meta.Pagination
	if !p.HasNextPage {
		t.Errorf("HasNextPage = false, want true (has_next_page tag must bind)")
	}
	if p.NextCursor != "cursor_2" {
		t.Errorf("NextCursor = %q, want %q (next_cursor tag must bind)", p.NextCursor, "cursor_2")
	}

	if len(page.Data) != 8 {
		t.Fatalf("data = %d results, want 8 (one per resource type)", len(page.Data))
	}

	// Decode order must equal the fixture (response) order — relevance is the answer,
	// never re-sorted (plan ADR-2).
	wantTypesInOrder := []string{"role", "note", "project", "action", "skill", "actor", "policy", "domain"}
	for i, want := range wantTypesInOrder {
		if page.Data[i].Type != want {
			t.Errorf("data[%d].type = %q, want %q (decode order must equal response order, and the type tag must bind)", i, page.Data[i].Type, want)
		}
	}

	// The first (role) row populates every field, including the nullable ones.
	first := page.Data[0]
	if first.ID != "role_0123" {
		t.Errorf("id = %q, want role_0123 (id tag must bind)", first.ID)
	}
	if first.Title != "Onboarding Lead" {
		t.Errorf("title = %q, want it populated (title tag must bind)", first.Title)
	}
	if first.Rank != 0.99 {
		t.Errorf("rank = %v, want 0.99 (rank tag must bind as a float)", first.Rank)
	}
	if first.Excerpt == nil || *first.Excerpt != "owns onboarding" {
		t.Errorf("excerpt = %v, want a populated pointer (excerpt tag must bind)", first.Excerpt)
	}
	if first.RoleID == nil || *first.RoleID != "role_0123" {
		t.Errorf("role_id = %v, want a populated pointer (role_id tag must bind)", first.RoleID)
	}
}

// TestSearchResultDecodesNullableFields pins the nullable contract: a row with
// excerpt:null and no role_id decodes with those fields nil (not an empty-string
// pointer, not an error), and a row with both present populates them. A pointer
// distinguishes absent from present so the render shows an explicit-absence marker
// rather than a fabricated value (CONSTITUTION VIII).
func TestSearchResultDecodesNullableFields(t *testing.T) {
	var page Page[SearchResult]
	if err := json.Unmarshal([]byte(searchPageFixture), &page); err != nil {
		t.Fatalf("decoding the /search page fixture failed: %v", err)
	}

	// The note row (index 1) has excerpt:null and omits role_id entirely.
	note := page.Data[1]
	if note.Excerpt != nil {
		t.Errorf("a null excerpt should decode to a nil pointer, got %v", *note.Excerpt)
	}
	if note.RoleID != nil {
		t.Errorf("an absent role_id should decode to a nil pointer, got %v", *note.RoleID)
	}

	// The role row (index 0) has both present.
	role := page.Data[0]
	if role.Excerpt == nil {
		t.Errorf("a present excerpt should decode to a non-nil pointer")
	}
	if role.RoleID == nil {
		t.Errorf("a present role_id should decode to a non-nil pointer")
	}
}

// TestSearchResultRankDecodesAsFloat pins that rank decodes as a float (a
// fractional relevance score), not truncated to an integer.
func TestSearchResultRankDecodesAsFloat(t *testing.T) {
	var result SearchResult
	if err := json.Unmarshal([]byte(`{"type":"role","id":"role_x","title":"T","rank":0.12345}`), &result); err != nil {
		t.Fatalf("decoding a single SearchResult failed: %v", err)
	}
	if result.Rank != 0.12345 {
		t.Errorf("rank = %v, want 0.12345 (must decode as a float, not truncated)", result.Rank)
	}
}

// TestSearchResultEmptyPageDecodes pins that an empty result set still decodes:
// data is empty, pagination reports no next page (a search matching nothing is a
// valid empty answer, not an error).
func TestSearchResultEmptyPageDecodes(t *testing.T) {
	var page Page[SearchResult]
	body := `{"data":[],"meta":{"pagination":{"per_page":100,"has_next_page":false,"next_cursor":""}}}`
	if err := json.Unmarshal([]byte(body), &page); err != nil {
		t.Fatalf("decoding an empty /search page failed: %v", err)
	}
	if len(page.Data) != 0 {
		t.Errorf("data = %d, want 0", len(page.Data))
	}
	if page.Meta.Pagination.HasNextPage {
		t.Errorf("HasNextPage = true, want false for an empty single page")
	}
}

// TestSearchResultToleratesUnknownFields pins forward-compatible decoding: the
// fixture carries unexpected fields at the top level, in pagination, and per row;
// the decode above succeeded — this pins the tolerance so a future strict decoder
// fails loud here.
func TestSearchResultToleratesUnknownFields(t *testing.T) {
	var page Page[SearchResult]
	if err := json.Unmarshal([]byte(searchPageFixture), &page); err != nil {
		t.Errorf("unknown fields should be ignored, decode failed: %v", err)
	}
}
