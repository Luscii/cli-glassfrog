package glassfrog

import (
	"encoding/json"
	"testing"
)

// TestPageDecodesEnvelopeAndPagination pins the generic Page[T] envelope: the
// data array decodes into []T in API order, and the shared meta.pagination block
// binds via its snake_case tags. Page[Role] stands in for any resource — the
// envelope shape is identical regardless of T.
func TestPageDecodesEnvelopeAndPagination(t *testing.T) {
	var page Page[Role]
	if err := json.Unmarshal([]byte(myRolesFixture), &page); err != nil {
		t.Fatalf("decoding the list envelope failed: %v", err)
	}

	if len(page.Data) != 2 {
		t.Fatalf("Data length = %d, want 2", len(page.Data))
	}
	if page.Data[0].Name != "Marketing Lead" {
		t.Errorf("Data[0].Name = %q, want %q (data must decode in API order)", page.Data[0].Name, "Marketing Lead")
	}
	if page.Data[1].Name != "Treasurer" {
		t.Errorf("Data[1].Name = %q, want %q (data must decode in API order)", page.Data[1].Name, "Treasurer")
	}

	p := page.Meta.Pagination
	if p.PerPage != 25 {
		t.Errorf("PerPage = %d, want 25 (per_page tag must bind)", p.PerPage)
	}
	if !p.HasNextPage {
		t.Errorf("HasNextPage = false, want true (has_next_page tag must bind)")
	}
	if p.NextCursor != "abc" {
		t.Errorf("NextCursor = %q, want %q (next_cursor tag must bind)", p.NextCursor, "abc")
	}
}

// TestPageDecodesAbsentPaginationAsComplete is the load-bearing non-paginated
// case: a body with no meta.pagination block decodes without error, leaving
// Pagination at its zero value (HasNextPage=false). The walker reads that as a
// single complete page — a non-paginated endpoint (e.g. the org role tree)
// completes in one response (ADR-2/ADR-5).
func TestPageDecodesAbsentPaginationAsComplete(t *testing.T) {
	const noMeta = `{"data": [{"id": "role_1", "name": "Solo"}]}`

	var page Page[Role]
	if err := json.Unmarshal([]byte(noMeta), &page); err != nil {
		t.Fatalf("decoding a body with no meta.pagination must not error: %v", err)
	}
	if len(page.Data) != 1 {
		t.Fatalf("Data length = %d, want 1", len(page.Data))
	}
	if page.Meta.Pagination.HasNextPage {
		t.Errorf("HasNextPage = true, want false for an absent meta.pagination (the complete-in-one-page case)")
	}
	if page.Meta.Pagination.NextCursor != "" {
		t.Errorf("NextCursor = %q, want empty for an absent meta.pagination", page.Meta.Pagination.NextCursor)
	}
}
