package glassfrog

import (
	"encoding/json"
	"testing"
)

// myProjectsFixture is a representative GET /me/projects body: the {data, meta}
// list envelope with two projects and a meta.pagination block reporting a next
// page. It carries extra/unknown fields the decode must tolerate, and uses the
// API's snake_case names throughout so a missing JSON tag fails loud here. The
// second project has a null role_id (a non-role-owned project) and has_* booleans
// false; the first has them true.
const myProjectsFixture = `{
  "data": [
    {
      "id": "proj_0123456789abcdef0123456789abcdef",
      "type": "project",
      "description": "Rebuild onboarding flow",
      "status": "current",
      "role_id": "role_0123456789abcdef0123456789abcdef",
      "individual_initiative": false,
      "has_sub_projects": true,
      "has_actions": true,
      "parent_project_id": "proj_00000000000000000000000000000009",
      "tags": ["marketing", "q2"],
      "created_at": "2026-01-01T00:00:00Z",
      "updated_at": "2026-01-02T00:00:00Z",
      "link": "https://example.test/p/1",
      "note": "kickoff scheduled",
      "unexpected_project_field": "ignored"
    },
    {
      "id": "proj_00000000000000000000000000000001",
      "type": "project",
      "description": null,
      "status": "scheduled",
      "role_id": null,
      "individual_initiative": true,
      "has_sub_projects": false,
      "has_actions": false,
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

// TestMyProjectsResponseDecodesSnakeCaseFields pins every snake_case JSON tag on
// the list envelope, the reused Pagination, and the Project fields. encoding/json
// is case-insensitive but does not bridge underscores, so an untagged HasSubProjects
// would silently never bind to has_sub_projects — this test feeds the real
// snake_case payload and asserts each field decoded.
func TestMyProjectsResponseDecodesSnakeCaseFields(t *testing.T) {
	var resp MyProjectsResponse
	if err := json.Unmarshal([]byte(myProjectsFixture), &resp); err != nil {
		t.Fatalf("decoding the /me/projects fixture failed: %v", err)
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
		t.Fatalf("data = %d projects, want 2", len(resp.Data))
	}

	// The projected subset of the first project.
	first := resp.Data[0]
	if first.ID != "proj_0123456789abcdef0123456789abcdef" {
		t.Errorf("id = %q, want the proj_ handle (id tag must bind)", first.ID)
	}
	if first.Status != "current" {
		t.Errorf("status = %q, want %q (status tag must bind)", first.Status, "current")
	}
	if first.Description != "Rebuild onboarding flow" {
		t.Errorf("description = %q, want it populated (description tag must bind)", first.Description)
	}
	if first.RoleID != "role_0123456789abcdef0123456789abcdef" {
		t.Errorf("role_id = %q, want the owning role handle (role_id tag must bind)", first.RoleID)
	}
	if !first.HasSubProjects {
		t.Errorf("has_sub_projects = false, want true (tag must bind)")
	}
	if !first.HasActions {
		t.Errorf("has_actions = false, want true (tag must bind)")
	}
	if len(first.Tags) != 2 || first.Tags[0] != "marketing" || first.Tags[1] != "q2" {
		t.Errorf("tags = %+v, want [marketing q2] (tags tag must bind)", first.Tags)
	}

	// Decoded-but-not-projected fields still bind from snake_case.
	if first.IndividualInitiative {
		t.Errorf("individual_initiative = true, want false for the first project")
	}
	if first.ParentProjectID != "proj_00000000000000000000000000000009" {
		t.Errorf("parent_project_id = %q, want it populated (parent_project_id tag must bind)", first.ParentProjectID)
	}
	if first.CreatedAt != "2026-01-01T00:00:00Z" || first.UpdatedAt != "2026-01-02T00:00:00Z" {
		t.Errorf("created_at/updated_at = %q/%q, want the timestamps (JSON tags must bind)", first.CreatedAt, first.UpdatedAt)
	}
	if first.Link != "https://example.test/p/1" || first.Note != "kickoff scheduled" {
		t.Errorf("link/note = %q/%q, want them populated (JSON tags must bind)", first.Link, first.Note)
	}

	// The second project is non-role-owned: a null role_id decodes to the empty
	// string (the projection turns that into the no-role marker). A null
	// description decodes to empty; has_* decode as false; tags empty.
	second := resp.Data[1]
	if second.RoleID != "" {
		t.Errorf("null role_id should decode to empty string (no owning role), got %q", second.RoleID)
	}
	if second.Description != "" {
		t.Errorf("null description should decode to empty string, got %q", second.Description)
	}
	if second.HasSubProjects || second.HasActions {
		t.Errorf("has_sub_projects/has_actions = %v/%v, want false/false", second.HasSubProjects, second.HasActions)
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

// A single complete page decodes with HasNextPage false (no further page).
func TestMyProjectsResponseDecodesSinglePage(t *testing.T) {
	var resp MyProjectsResponse
	body := `{"data":[{"id":"proj_0123456789abcdef0123456789abcdef","status":"current","description":"x","role_id":"role_0123456789abcdef0123456789abcdef","has_sub_projects":false,"has_actions":false,"tags":[]}],"meta":{"pagination":{"per_page":25,"has_next_page":false,"next_cursor":""}}}`
	if err := json.Unmarshal([]byte(body), &resp); err != nil {
		t.Fatalf("decoding a single-page /me/projects body failed: %v", err)
	}
	if len(resp.Data) != 1 {
		t.Fatalf("data = %d, want 1", len(resp.Data))
	}
	if resp.Meta.Pagination.HasNextPage {
		t.Errorf("HasNextPage = true, want false for a single complete page")
	}
}

// An empty list still decodes: data is empty, pagination reports no next page.
func TestMyProjectsResponseDecodesEmptyList(t *testing.T) {
	var resp MyProjectsResponse
	body := `{"data":[],"meta":{"pagination":{"per_page":25,"has_next_page":false,"next_cursor":""}}}`
	if err := json.Unmarshal([]byte(body), &resp); err != nil {
		t.Fatalf("decoding an empty /me/projects body failed: %v", err)
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
// project; the decode in TestMyProjectsResponseDecodesSnakeCaseFields succeeded —
// this pins the tolerance so a future strict decoder fails loud here.
func TestMyProjectsResponseToleratesUnknownFields(t *testing.T) {
	var resp MyProjectsResponse
	if err := json.Unmarshal([]byte(myProjectsFixture), &resp); err != nil {
		t.Errorf("unknown fields should be ignored, decode failed: %v", err)
	}
}
