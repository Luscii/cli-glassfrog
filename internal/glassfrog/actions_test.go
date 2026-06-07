package glassfrog

import (
	"encoding/json"
	"testing"
)

// myActionsFixture is a representative GET /me/actions body: the {data, meta}
// list envelope with two actions and a meta.pagination block reporting a next
// page. It carries extra/unknown fields the decode must tolerate, and uses the
// API's snake_case names throughout so a missing JSON tag fails loud here.
const myActionsFixture = `{
  "data": [
    {
      "id": "actn_0123456789abcdef0123456789abcdef",
      "type": "action",
      "description": "Review PR #6818",
      "status": "current",
      "role_id": "role_0123456789abcdef0123456789abcdef",
      "individual_initiative": false,
      "parent_project_id": "proj_0123456789abcdef0123456789abcdef",
      "tags": ["marketing", "q2"],
      "created_at": "2026-01-01T00:00:00Z",
      "updated_at": "2026-01-02T00:00:00Z",
      "permissions": {"can_update": true},
      "trigger_event": "after the standup",
      "note": "blocked on review",
      "unexpected_action_field": "ignored"
    },
    {
      "id": "actn_00000000000000000000000000000001",
      "type": "action",
      "description": null,
      "status": "waiting",
      "role_id": "role_00000000000000000000000000000001",
      "individual_initiative": true,
      "parent_project_id": null,
      "tags": [],
      "created_at": "2026-01-03T00:00:00Z",
      "updated_at": "2026-01-04T00:00:00Z"
    }
  ],
  "meta": {
    "pagination": {
      "per_page": 25,
      "has_next_page": true,
      "next_cursor": "abc",
      "unexpected_pagination_field": "ignored"
    }
  },
  "unexpected_top_level": {"anything": [1, 2, 3]}
}`

// TestMyActionsResponseDecodesSnakeCaseFields pins every snake_case JSON tag on
// the list envelope, the reused Pagination, and the Action fields. encoding/json
// is case-insensitive but does not bridge underscores, so an untagged RoleID
// would silently never bind to role_id — this test feeds the real snake_case
// payload and asserts each field decoded.
func TestMyActionsResponseDecodesSnakeCaseFields(t *testing.T) {
	var resp MyActionsResponse
	if err := json.Unmarshal([]byte(myActionsFixture), &resp); err != nil {
		t.Fatalf("decoding the /me/actions fixture failed: %v", err)
	}

	// Pagination is the shared type reused from My Roles (012); a multi-page
	// fixture sets has_next_page true and carries the cursor.
	p := resp.Meta.Pagination
	if !p.HasNextPage {
		t.Errorf("HasNextPage = false, want true (has_next_page tag must bind)")
	}
	if p.NextCursor != "abc" {
		t.Errorf("NextCursor = %q, want %q (next_cursor tag must bind)", p.NextCursor, "abc")
	}
	if p.PerPage != 25 {
		t.Errorf("PerPage = %d, want 25 (per_page tag must bind)", p.PerPage)
	}

	if len(resp.Data) != 2 {
		t.Fatalf("data = %d actions, want 2", len(resp.Data))
	}

	// The projected subset of the first action.
	first := resp.Data[0]
	if first.ID != "actn_0123456789abcdef0123456789abcdef" {
		t.Errorf("id = %q, want the actn_ handle (id tag must bind)", first.ID)
	}
	if first.Status != "current" {
		t.Errorf("status = %q, want %q (status tag must bind)", first.Status, "current")
	}
	if first.Description != "Review PR #6818" {
		t.Errorf("description = %q, want it populated (description tag must bind)", first.Description)
	}
	if first.RoleID != "role_0123456789abcdef0123456789abcdef" {
		t.Errorf("role_id = %q, want the owning role handle (role_id tag must bind)", first.RoleID)
	}
	if len(first.Tags) != 2 || first.Tags[0] != "marketing" || first.Tags[1] != "q2" {
		t.Errorf("tags = %+v, want [marketing q2] (tags tag must bind)", first.Tags)
	}

	// Decoded-but-not-projected fields still bind from snake_case. The first
	// action's individual_initiative is false (the second's true is checked below).
	if first.IndividualInitiative {
		t.Errorf("individual_initiative = true, want false for the first action")
	}
	if first.ParentProjectID != "proj_0123456789abcdef0123456789abcdef" {
		t.Errorf("parent_project_id = %q, want it populated (parent_project_id tag must bind)", first.ParentProjectID)
	}
	if first.CreatedAt != "2026-01-01T00:00:00Z" || first.UpdatedAt != "2026-01-02T00:00:00Z" {
		t.Errorf("created_at/updated_at = %q/%q, want the timestamps (tags must bind)", first.CreatedAt, first.UpdatedAt)
	}
	if first.TriggerEvent != "after the standup" || first.Note != "blocked on review" {
		t.Errorf("trigger_event/note = %q/%q, want them populated (tags must bind)", first.TriggerEvent, first.Note)
	}

	// A null description decodes to the empty string; an empty tags array decodes
	// to an empty (non-nil-required) slice; individual_initiative true binds.
	second := resp.Data[1]
	if second.Description != "" {
		t.Errorf("null description should decode to empty string, got %q", second.Description)
	}
	if !second.IndividualInitiative {
		t.Errorf("individual_initiative = false, want true (tag must bind)")
	}
	if second.ParentProjectID != "" {
		t.Errorf("null parent_project_id should decode to empty string, got %q", second.ParentProjectID)
	}
	if len(second.Tags) != 0 {
		t.Errorf("empty tags should decode to an empty slice, got %+v", second.Tags)
	}
}

// An empty list still decodes: data is empty, pagination reports no next page.
func TestMyActionsResponseDecodesEmptyList(t *testing.T) {
	var resp MyActionsResponse
	body := `{"data":[],"meta":{"pagination":{"per_page":25,"has_next_page":false,"next_cursor":""}}}`
	if err := json.Unmarshal([]byte(body), &resp); err != nil {
		t.Fatalf("decoding an empty /me/actions body failed: %v", err)
	}
	if len(resp.Data) != 0 {
		t.Errorf("data = %d, want 0", len(resp.Data))
	}
	if resp.Meta.Pagination.HasNextPage {
		t.Errorf("HasNextPage = true, want false for a single complete page")
	}
}

// Unknown/extra JSON fields must not fail the decode (forward-compatible). The
// fixture carries unexpected fields at the top level, in pagination, and per
// action; the decode in TestMyActionsResponseDecodesSnakeCaseFields succeeded —
// this pins the tolerance so a future strict decoder fails loud here.
func TestMyActionsResponseToleratesUnknownFields(t *testing.T) {
	var resp MyActionsResponse
	if err := json.Unmarshal([]byte(myActionsFixture), &resp); err != nil {
		t.Errorf("unknown fields should be ignored, decode failed: %v", err)
	}
}
