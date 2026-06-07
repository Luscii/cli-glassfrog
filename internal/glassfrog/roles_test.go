package glassfrog

import (
	"encoding/json"
	"testing"
)

// myRolesFixture is a representative GET /me/roles body: the {data, meta} list
// envelope with two full-shape roles (purpose + domains + accountabilities) and
// a meta.pagination block reporting a next page. It carries extra/unknown fields
// the decode must tolerate, and uses the API's snake_case names throughout so a
// missing JSON tag fails loud here.
const myRolesFixture = `{
  "data": [
    {
      "id": "role_0123456789abcdef0123456789abcdef",
      "name": "Marketing Lead",
      "purpose": "A market that knows us",
      "domains": [
        {"id": "dom_1", "description": "The marketing budget"},
        {"id": "dom_2", "description": "The brand guidelines"}
      ],
      "accountabilities": [
        {"id": "acc_1", "description": "Defining the quarterly campaign"},
        {"id": "acc_2", "description": "Reporting reach to the circle"},
        {"id": "acc_3", "description": "Maintaining the press list"}
      ],
      "fillers": [{"id": "per_x", "name": "ignored"}],
      "tags": ["ignored"]
    },
    {
      "id": "role_00000000000000000000000000000001",
      "name": "Treasurer",
      "purpose": null,
      "domains": [],
      "accountabilities": []
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

// TestMyRolesResponseDecodesSnakeCaseFields pins every snake_case JSON tag on the
// list envelope, Pagination, and the grown Role fields. encoding/json is
// case-insensitive but does not bridge underscores, so an untagged HasNextPage
// would silently never bind to has_next_page and incompleteness would read false
// — this test feeds the real snake_case payload and asserts each field decoded.
func TestMyRolesResponseDecodesSnakeCaseFields(t *testing.T) {
	var resp MyRolesResponse
	if err := json.Unmarshal([]byte(myRolesFixture), &resp); err != nil {
		t.Fatalf("decoding the /me/roles fixture failed: %v", err)
	}

	// Pagination: the snake_case tags must bind.
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

	// The role's full shape populated.
	if len(resp.Data) != 2 {
		t.Fatalf("data = %d roles, want 2", len(resp.Data))
	}
	lead := resp.Data[0]
	if lead.Name != "Marketing Lead" || lead.ID != "role_0123456789abcdef0123456789abcdef" {
		t.Errorf("role[0] = %+v, want Marketing Lead role_…", lead)
	}
	if lead.Purpose != "A market that knows us" {
		t.Errorf("purpose = %q, want it populated (purpose tag must bind)", lead.Purpose)
	}
	if len(lead.Domains) != 2 || lead.Domains[0].Description != "The marketing budget" {
		t.Errorf("domains = %+v, want 2 with description text (description tag must bind)", lead.Domains)
	}
	if len(lead.Accountabilities) != 3 || lead.Accountabilities[0].Description != "Defining the quarterly campaign" {
		t.Errorf("accountabilities = %+v, want 3 with description text", lead.Accountabilities)
	}

	// A null purpose decodes to the empty string; empty domains/accountabilities
	// arrays decode to empty slices.
	treasurer := resp.Data[1]
	if treasurer.Purpose != "" {
		t.Errorf("null purpose should decode to empty string, got %q", treasurer.Purpose)
	}
	if len(treasurer.Domains) != 0 || len(treasurer.Accountabilities) != 0 {
		t.Errorf("treasurer should have no domains/accountabilities, got %+v", treasurer)
	}
}

// An empty list still decodes: data is empty, pagination reports no next page.
func TestMyRolesResponseDecodesEmptyList(t *testing.T) {
	var resp MyRolesResponse
	body := `{"data":[],"meta":{"pagination":{"per_page":25,"has_next_page":false,"next_cursor":""}}}`
	if err := json.Unmarshal([]byte(body), &resp); err != nil {
		t.Fatalf("decoding an empty /me/roles body failed: %v", err)
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
// role; the decode above succeeded — this pins the tolerance so a future strict
// decoder fails loud here.
func TestMyRolesResponseToleratesUnknownFields(t *testing.T) {
	var resp MyRolesResponse
	if err := json.Unmarshal([]byte(myRolesFixture), &resp); err != nil {
		t.Errorf("unknown fields should be ignored, decode failed: %v", err)
	}
}
