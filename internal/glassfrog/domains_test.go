package glassfrog

import (
	"encoding/json"
	"testing"
)

// getDomainFixture is a representative GET /domains/{id} body: the single-object
// {data: Domain} envelope carrying every always-present standalone field
// (type/role_id/created_at/updated_at) plus an embedded policies array (the
// ?include=policies case), in the API's snake_case names so a missing JSON tag
// fails loud here. It carries an extra/unknown field the decode must tolerate.
const getDomainFixture = `{
  "data": {
    "id": "dom_0123456789abcdef0123456789abcdef",
    "type": "domain",
    "description": "The marketing budget",
    "role_id": "role_aaaa000000000000000000000000aaaa",
    "created_at": "2024-01-01T00:00:00Z",
    "updated_at": "2024-02-02T00:00:00Z",
    "policies": [
      {"id": "pol_1", "title": "Spend under $10k needs no approval", "body": "The full body"},
      {"id": "pol_2", "title": "Quarterly review of vendors", "body": "More text"}
    ],
    "unexpected_domain_field": [1, 2, 3]
  }
}`

// TestDomainDocumentDecodesStandaloneFields pins the single-read envelope: every
// always-present standalone field binds via its snake_case tag, and the embedded
// policies (reusing the landed Policy) populate when ?include=policies was
// requested.
func TestDomainDocumentDecodesStandaloneFields(t *testing.T) {
	var doc DomainDocument
	if err := json.Unmarshal([]byte(getDomainFixture), &doc); err != nil {
		t.Fatalf("decoding the /domains/{id} fixture failed: %v", err)
	}
	d := doc.Data

	if d.ID != "dom_0123456789abcdef0123456789abcdef" {
		t.Errorf("id = %q, want the dom_ handle", d.ID)
	}
	if d.Type != "domain" {
		t.Errorf("type = %q, want %q", d.Type, "domain")
	}
	if d.Description != "The marketing budget" {
		t.Errorf("description = %q", d.Description)
	}
	if d.RoleID == nil || *d.RoleID != "role_aaaa000000000000000000000000aaaa" {
		t.Errorf("role_id not bound: %v", d.RoleID)
	}
	if d.CreatedAt != "2024-01-01T00:00:00Z" || d.UpdatedAt != "2024-02-02T00:00:00Z" {
		t.Errorf("timestamps not bound: created=%q updated=%q", d.CreatedAt, d.UpdatedAt)
	}
	if len(d.Policies) != 2 || d.Policies[0].Title != "Spend under $10k needs no approval" {
		t.Errorf("policies not bound (reusing Policy; title must project): %+v", d.Policies)
	}
}

// TestDomainDocumentNullRoleIDAndNoPolicies pins the two absence cases the render
// guards on: a null role_id decodes to a nil pointer (not "", not an error — the
// render shows its (no controlling role) marker), and an absent policies array
// (no ?include=policies) stays nil/empty rather than decoding to a fabricated
// value.
func TestDomainDocumentNullRoleIDAndNoPolicies(t *testing.T) {
	body := `{"data": {"id": "dom_0000000000000000000000000000ffff", "type": "domain", "description": "An unbound area", "role_id": null, "created_at": "2024-01-01T00:00:00Z", "updated_at": "2024-01-01T00:00:00Z"}}`
	var doc DomainDocument
	if err := json.Unmarshal([]byte(body), &doc); err != nil {
		t.Fatalf("decoding a null-role_id /domains/{id} body failed: %v", err)
	}
	d := doc.Data
	if d.RoleID != nil {
		t.Errorf("a null role_id should decode to nil, got %q", *d.RoleID)
	}
	if d.Policies != nil {
		t.Errorf("an unrequested policies embed should stay nil, got %+v", d.Policies)
	}
}

// roleDomainsPageFixture is a representative GET /roles/{id}/domains body: the
// paginated {data: [Domain], meta: {pagination}} envelope the list decodes into
// the existing generic Page[Domain] (016) — never a 033-local envelope. The
// pagination reports a next page so the walker keeps going. One domain carries a
// null role_id; neither carries a policies embed (the list never includes it).
const roleDomainsPageFixture = `{
  "data": [
    {"id": "dom_1", "type": "domain", "description": "The marketing budget", "role_id": "role_0123", "created_at": "2024-01-01T00:00:00Z", "updated_at": "2024-01-01T00:00:00Z"},
    {"id": "dom_2", "type": "domain", "description": "The brand guidelines", "role_id": null, "created_at": "2024-01-01T00:00:00Z", "updated_at": "2024-01-01T00:00:00Z"}
  ],
  "meta": {"pagination": {"per_page": 100, "has_next_page": true, "next_cursor": "page2"}}
}`

// TestRoleDomainsPageDecodes pins the list shape: GET /roles/{id}/domains decodes
// into Page[Domain] with Data populated and the pagination read — confirming the
// grown Domain rides the existing generic envelope without a bespoke list type.
func TestRoleDomainsPageDecodes(t *testing.T) {
	var page Page[Domain]
	if err := json.Unmarshal([]byte(roleDomainsPageFixture), &page); err != nil {
		t.Fatalf("decoding the /roles/{id}/domains fixture failed: %v", err)
	}
	if len(page.Data) != 2 {
		t.Fatalf("domains data = %d, want 2", len(page.Data))
	}
	if !page.Meta.Pagination.HasNextPage || page.Meta.Pagination.NextCursor != "page2" {
		t.Errorf("pagination not read: %+v", page.Meta.Pagination)
	}
	if page.Data[0].Description != "The marketing budget" {
		t.Errorf("first domain description = %q", page.Data[0].Description)
	}
	if page.Data[1].RoleID != nil {
		t.Errorf("second domain role_id = %v, want nil (null)", *page.Data[1].RoleID)
	}
}

// TestDomainInlineEmbedStillDecodes pins the additive-growth no-regression
// guarantee: the inline-on-Role projection (the original {id, description} shape
// 025/026 render) still decodes into the grown Domain unchanged — the new fields
// simply stay at their zero values when the embed omits them.
func TestDomainInlineEmbedStillDecodes(t *testing.T) {
	body := `{"id": "dom_inline", "description": "The marketing budget"}`
	var d Domain
	if err := json.Unmarshal([]byte(body), &d); err != nil {
		t.Fatalf("decoding the inline Domain embed failed: %v", err)
	}
	if d.ID != "dom_inline" || d.Description != "The marketing budget" {
		t.Errorf("inline embed not bound: %+v", d)
	}
	if d.Type != "" || d.RoleID != nil || d.Policies != nil {
		t.Errorf("grown fields should stay zero on the inline embed: %+v", d)
	}
}
