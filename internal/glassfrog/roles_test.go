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

// orgRolesPageFixture is a representative GET /roles page: the generic
// {data, meta:{pagination}} envelope (Page[Role]) carrying the grown Role fields
// — type, parent_role_id, has_subroles, flags, fillers, tags — alongside the
// already-pinned purpose/domains/accountabilities. The first role is a
// sub-role (non-null parent, has_subroles true); the second is an anchor role
// (null parent_role_id, empty arrays). Extra/unknown fields are present so the
// decode's tolerance is exercised here too.
const orgRolesPageFixture = `{
  "data": [
    {
      "id": "role_0123456789abcdef0123456789abcdef",
      "type": "role",
      "name": "Marketing Lead",
      "purpose": "A market that knows us",
      "parent_role_id": "role_aaaa000000000000000000000000aaaa",
      "has_subroles": true,
      "flags": ["structural", "elected"],
      "domains": [{"id": "dom_1", "description": "The marketing budget"}],
      "accountabilities": [{"id": "acct_1", "description": "Defining the campaign"}],
      "fillers": [
        {"id": "per_x", "name": "Alice Smith", "kind": "human"},
        {"id": "agt_y", "name": "Claude", "kind": "agent"}
      ],
      "tags": ["marketing", "q2"],
      "unexpected_role_field": true
    },
    {
      "id": "role_00000000000000000000000000000001",
      "type": "role",
      "name": "Anchor Circle",
      "purpose": null,
      "parent_role_id": null,
      "has_subroles": false,
      "flags": [],
      "domains": [],
      "accountabilities": [],
      "fillers": [],
      "tags": []
    }
  ],
  "meta": {"pagination": {"per_page": 500, "has_next_page": true, "next_cursor": "c2"}}
}`

// TestPageRoleDecodesGrownFields pins that a GET /roles page decodes into the
// generic Page[Role] (016) with the grown Role fields populated and the shared
// Pagination read — every new snake_case tag must bind.
func TestPageRoleDecodesGrownFields(t *testing.T) {
	var page Page[Role]
	if err := json.Unmarshal([]byte(orgRolesPageFixture), &page); err != nil {
		t.Fatalf("decoding the /roles page fixture failed: %v", err)
	}
	if len(page.Data) != 2 {
		t.Fatalf("data = %d roles, want 2", len(page.Data))
	}
	if !page.Meta.Pagination.HasNextPage || page.Meta.Pagination.NextCursor != "c2" {
		t.Errorf("pagination not read: %+v", page.Meta.Pagination)
	}

	lead := page.Data[0]
	if lead.Type != "role" {
		t.Errorf("type = %q, want role (type tag must bind)", lead.Type)
	}
	if lead.ParentRoleID == nil || *lead.ParentRoleID != "role_aaaa000000000000000000000000aaaa" {
		t.Errorf("parent_role_id = %v, want the parent id (must decode to a non-nil pointer)", lead.ParentRoleID)
	}
	if !lead.HasSubroles {
		t.Errorf("has_subroles = false, want true (has_subroles tag must bind)")
	}
	if len(lead.Flags) != 2 || lead.Flags[0] != "structural" {
		t.Errorf("flags = %v, want [structural elected]", lead.Flags)
	}
	if len(lead.Fillers) != 2 || lead.Fillers[0].Name != "Alice Smith" || lead.Fillers[1].Kind != "agent" {
		t.Errorf("fillers = %+v, want Alice (human) + Claude (agent)", lead.Fillers)
	}
	if len(lead.Tags) != 2 || lead.Tags[0] != "marketing" {
		t.Errorf("tags = %v, want [marketing q2]", lead.Tags)
	}

	// The anchor role: a null parent_role_id decodes to a nil pointer (distinct
	// from an empty-string id), and the empty arrays decode without error.
	anchor := page.Data[1]
	if anchor.ParentRoleID != nil {
		t.Errorf("a null parent_role_id should decode to nil, got %v", anchor.ParentRoleID)
	}
	if anchor.HasSubroles {
		t.Errorf("has_subroles = true, want false for the leaf anchor")
	}
	if len(anchor.Flags) != 0 || len(anchor.Fillers) != 0 || len(anchor.Tags) != 0 {
		t.Errorf("anchor should have empty flags/fillers/tags, got %+v", anchor)
	}
}

// roleDetailFixture is a representative GET /roles/{id} body: the single-object
// {data: RoleDetail} envelope with every ?include resource present — assignments,
// subroles, parent_role, policies, notes, skills — so each related field's tag is
// pinned. SkillSummary carries no content.
const roleDetailFixture = `{
  "data": {
    "id": "role_0123456789abcdef0123456789abcdef",
    "type": "role",
    "name": "Marketing Lead",
    "purpose": "A market that knows us",
    "parent_role_id": "role_aaaa000000000000000000000000aaaa",
    "has_subroles": true,
    "flags": ["structural"],
    "domains": [{"id": "dom_1", "description": "The marketing budget"}],
    "accountabilities": [{"id": "acct_1", "description": "Defining the campaign"}],
    "fillers": [{"id": "per_x", "name": "Alice Smith", "kind": "human"}],
    "tags": ["marketing"],
    "assignments": [{"id": "asgn_1", "actor_id": "per_x", "role_id": "role_0123456789abcdef0123456789abcdef", "focus": "EU", "actor": {"id": "per_x", "name": "Alice Smith", "kind": "human"}}],
    "subroles": [{"id": "role_sub00000000000000000000000000sub", "type": "role", "name": "Press Officer", "parent_role_id": "role_0123456789abcdef0123456789abcdef", "has_subroles": false}],
    "parent_role": {"id": "role_aaaa000000000000000000000000aaaa", "type": "role", "name": "General Company Circle"},
    "policies": [{"id": "pol_1", "title": "All PRs require two approvals", "body": "The full body"}],
    "notes": [{"id": "rnot_1", "title": "How to run the deploy", "body": "markdown", "role_id": "role_0123456789abcdef0123456789abcdef"}],
    "skills": [{"id": "skill_1", "type": "skill", "name": "Code Review Standards"}],
    "unexpected_detail_field": [1, 2, 3]
  }
}`

// TestRoleDocumentDecodesDetail pins the single-read envelope: RoleDocument.Data
// is a RoleDetail that exposes the embedded Role fields AND each requested
// related resource. SkillSummary has no Content field by construction.
func TestRoleDocumentDecodesDetail(t *testing.T) {
	var doc RoleDocument
	if err := json.Unmarshal([]byte(roleDetailFixture), &doc); err != nil {
		t.Fatalf("decoding the /roles/{id} fixture failed: %v", err)
	}
	d := doc.Data

	// Embedded Role fields are reachable directly via the embed.
	if d.Name != "Marketing Lead" || d.ID != "role_0123456789abcdef0123456789abcdef" {
		t.Errorf("embedded Role not exposed: %+v", d.Role)
	}
	if d.Purpose != "A market that knows us" || !d.HasSubroles {
		t.Errorf("embedded Role fields not bound: purpose=%q hasSubroles=%v", d.Purpose, d.HasSubroles)
	}

	if len(d.Assignments) != 1 || d.Assignments[0].ActorID != "per_x" || d.Assignments[0].Actor.Name != "Alice Smith" {
		t.Errorf("assignments not bound: %+v", d.Assignments)
	}
	if len(d.Subroles) != 1 || d.Subroles[0].Name != "Press Officer" {
		t.Errorf("subroles not bound (must be plain Role): %+v", d.Subroles)
	}
	if d.ParentRole == nil || d.ParentRole.Name != "General Company Circle" {
		t.Errorf("parent_role not bound: %+v", d.ParentRole)
	}
	if len(d.Policies) != 1 || d.Policies[0].Title != "All PRs require two approvals" {
		t.Errorf("policies not bound (title must project): %+v", d.Policies)
	}
	if len(d.Notes) != 1 || d.Notes[0].Title != "How to run the deploy" {
		t.Errorf("notes not bound (title must project): %+v", d.Notes)
	}
	if len(d.Skills) != 1 || d.Skills[0].Name != "Code Review Standards" {
		t.Errorf("skills not bound: %+v", d.Skills)
	}
}

// TestRoleDocumentOmittedIncludesStayEmpty pins that when no ?include was
// requested the related fields decode to nil/empty (not an error), and a null
// parent_role decodes to a nil pointer — the render guards on exactly these.
func TestRoleDocumentOmittedIncludesStayEmpty(t *testing.T) {
	body := `{"data": {"id": "role_0123456789abcdef0123456789abcdef", "type": "role", "name": "Bare Role", "purpose": null, "parent_role_id": null, "has_subroles": false, "flags": [], "domains": [], "accountabilities": [], "fillers": [], "tags": [], "parent_role": null}}`
	var doc RoleDocument
	if err := json.Unmarshal([]byte(body), &doc); err != nil {
		t.Fatalf("decoding a bare /roles/{id} body failed: %v", err)
	}
	d := doc.Data
	if d.Assignments != nil || d.Subroles != nil || d.Policies != nil || d.Notes != nil || d.Skills != nil {
		t.Errorf("unrequested include fields should stay nil, got %+v", d)
	}
	if d.ParentRole != nil {
		t.Errorf("a null parent_role should decode to nil, got %+v", d.ParentRole)
	}
}
