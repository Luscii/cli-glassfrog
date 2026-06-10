package glassfrog

import (
	"encoding/json"
	"testing"
)

// policyDocumentFixture is a representative GET /policies/{id} body: the
// single-object {data: Policy} envelope carrying the full policy spec shape —
// id/title/body plus the grown role_id/domain_id/created_at/updated_at — with an
// unexpected sibling field present to prove tolerant decoding.
const policyDocumentFixture = `{
  "data": {
    "id": "pol_0123456789abcdef0123456789abcdef",
    "title": "All PRs require two approvals",
    "body": "<p>Every pull request <strong>must</strong> carry two approvals.</p>",
    "role_id": "role_0123456789abcdef0123456789abcdef",
    "domain_id": "dom_0123456789abcdef0123456789abcdef",
    "created_at": "2024-01-02T03:04:05Z",
    "updated_at": "2024-05-06T07:08:09Z",
    "unexpected_policy_field": [1, 2, 3]
  }
}`

// TestPolicyDocumentDecodesFullShape pins the single-read envelope: Document[Policy]
// exposes Data with every grown field bound, and an unknown sibling field is
// ignored (tolerant decoding).
func TestPolicyDocumentDecodesFullShape(t *testing.T) {
	var doc Document[Policy]
	if err := json.Unmarshal([]byte(policyDocumentFixture), &doc); err != nil {
		t.Fatalf("decoding the /policies/{id} fixture failed: %v", err)
	}
	p := doc.Data
	if p.ID != "pol_0123456789abcdef0123456789abcdef" {
		t.Errorf("id not bound: %q", p.ID)
	}
	if p.Title != "All PRs require two approvals" {
		t.Errorf("title not bound: %q", p.Title)
	}
	if p.Body != "<p>Every pull request <strong>must</strong> carry two approvals.</p>" {
		t.Errorf("body not bound verbatim (HTML preserved): %q", p.Body)
	}
	if p.RoleID != "role_0123456789abcdef0123456789abcdef" {
		t.Errorf("role_id not bound: %q", p.RoleID)
	}
	if p.DomainID != "dom_0123456789abcdef0123456789abcdef" {
		t.Errorf("domain_id not bound: %q", p.DomainID)
	}
	if p.CreatedAt != "2024-01-02T03:04:05Z" || p.UpdatedAt != "2024-05-06T07:08:09Z" {
		t.Errorf("timestamps not bound: created=%q updated=%q", p.CreatedAt, p.UpdatedAt)
	}
}

// TestPolicyNullScopeDecodesEmpty pins the nullable scope: a policy with null
// role_id and null domain_id decodes to empty strings without error (mirroring the
// nullable Body), never a panic — the render guards explicit-absence on exactly
// these empties.
func TestPolicyNullScopeDecodesEmpty(t *testing.T) {
	body := `{"data": {"id": "pol_x", "title": "Org rule", "body": null, "role_id": null, "domain_id": null, "created_at": "2024-01-02T03:04:05Z", "updated_at": "2024-01-02T03:04:05Z"}}`
	var doc Document[Policy]
	if err := json.Unmarshal([]byte(body), &doc); err != nil {
		t.Fatalf("decoding a null-scope policy failed: %v", err)
	}
	p := doc.Data
	if p.RoleID != "" || p.DomainID != "" || p.Body != "" {
		t.Errorf("null fields should decode to empty strings, got role=%q domain=%q body=%q", p.RoleID, p.DomainID, p.Body)
	}
	if p.ID != "pol_x" || p.Title != "Org rule" {
		t.Errorf("present fields should still bind: %+v", p)
	}
}

// policiesPageFixture is a representative GET /roles/{id}/policies body: the
// paginated {data: [Policy], meta: {pagination}} envelope decoded as the generic
// Page[Policy] (016) — no 034-local list envelope. One policy is role-level (null
// domain_id), the meta reports a next page so the walker keeps going.
const policiesPageFixture = `{
  "data": [
    {"id": "pol_1", "title": "Two approvals", "body": "b1", "role_id": "role_0123", "domain_id": "dom_1", "created_at": "2024-01-01T00:00:00Z", "updated_at": "2024-01-01T00:00:00Z"},
    {"id": "pol_2", "title": "Spending limit", "body": "b2", "role_id": "role_0123", "domain_id": null, "created_at": "2024-02-01T00:00:00Z", "updated_at": "2024-02-01T00:00:00Z"}
  ],
  "meta": {"pagination": {"per_page": 100, "has_next_page": true, "next_cursor": "page2"}}
}`

// TestPoliciesPageDecodesPage pins the paginated list shape: Page[Policy] binds
// Data and reads Meta.Pagination, and a role-level policy (null domain_id) decodes
// to an empty string.
func TestPoliciesPageDecodesPage(t *testing.T) {
	var page Page[Policy]
	if err := json.Unmarshal([]byte(policiesPageFixture), &page); err != nil {
		t.Fatalf("decoding the /roles/{id}/policies fixture failed: %v", err)
	}
	if len(page.Data) != 2 {
		t.Fatalf("policies data = %d, want 2", len(page.Data))
	}
	if !page.Meta.Pagination.HasNextPage || page.Meta.Pagination.NextCursor != "page2" {
		t.Errorf("pagination not read: %+v", page.Meta.Pagination)
	}
	if page.Data[0].Title != "Two approvals" || page.Data[0].DomainID != "dom_1" {
		t.Errorf("first policy not bound: %+v", page.Data[0])
	}
	if page.Data[1].DomainID != "" {
		t.Errorf("role-level policy (null domain_id) should decode to empty string, got %q", page.Data[1].DomainID)
	}
}

// TestRoleDocumentStillDecodesViaAlias pins that the RoleDocument alias of
// Document[RoleDetail] keeps 025's single-role decode byte-stable: `var doc
// RoleDocument` decodes a {data: RoleDetail} body and exposes Data.Role unchanged.
func TestRoleDocumentStillDecodesViaAlias(t *testing.T) {
	body := `{"data": {"id": "role_0123", "type": "role", "name": "Marketing Lead", "purpose": "Win"}}`
	var doc RoleDocument
	if err := json.Unmarshal([]byte(body), &doc); err != nil {
		t.Fatalf("decoding via the RoleDocument alias failed: %v", err)
	}
	if doc.Data.Name != "Marketing Lead" || doc.Data.ID != "role_0123" {
		t.Errorf("alias decode did not bind the embedded Role: %+v", doc.Data.Role)
	}
}
