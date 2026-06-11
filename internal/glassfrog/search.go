package glassfrog

// SearchResult is one row of the GET /search result set: a cross-type summary of
// a single governance record that matched the operator's full-text query
// (spec/glassfrog-api-v5.yaml SearchResult §6123). It is its OWN flat type — not a
// reuse or growth of Role/RoleDetail (011 ADR-1 only mandates sharing when the
// shape is the same): a SearchResult carries Rank and Excerpt that no resource
// projection has, and a Type that can be any of the eight resource kinds
// (role/note/project/action/skill/actor/policy/domain) in a single ranked list,
// so it is a discovery row, not a resource read (041 plan ADR-2).
//
// The list decodes via the EXISTING generic Page[SearchResult] envelope (016) —
// no 041-local envelope is defined. The walker appends pages in API order, so the
// decode order is the API's relevance order; the CLI never re-sorts, de-dups, or
// filters (041 plan ADR-2).
//
// Type is decoded as a plain string, NOT a constrained Go enum: the API owns the
// vocabulary, an unknown future type must decode rather than fail, and the CLI's
// --types validation (a separate client-side concern) is the place that rejects
// an out-of-set INPUT — a response value is rendered as received.
//
// Excerpt and RoleID are *string because both are nullable in the spec: a null
// excerpt (a row with no snippet) and an absent role_id (the field applies only to
// a subset of types) must decode cleanly to nil so the render shows an
// explicit-absence marker rather than a fabricated or empty value (CONSTITUTION
// VIII). A pointer distinguishes "absent" (nil) from a present empty string.
//
// Every field carries an explicit snake_case JSON tag — encoding/json is
// case-insensitive but does NOT bridge underscores, so an untagged RoleID would
// silently never bind to the API's role_id. Decoding is tolerant of unknown/extra
// fields (forward-compatible). The token is never a field here — it is an
// X-Auth-Token request header, not a response field, so secret hygiene holds by
// construction (CONSTITUTION II).
type SearchResult struct {
	Type    string  `json:"type"`
	ID      string  `json:"id"`
	Title   string  `json:"title"`
	Excerpt *string `json:"excerpt"`
	Rank    float64 `json:"rank"`
	RoleID  *string `json:"role_id"`
}
