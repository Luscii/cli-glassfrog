package glassfrog

// Page is the generic {data, meta:{pagination}} list envelope every paginated
// read decodes into (ADR-2, 016). It supersedes the per-resource concrete
// envelopes (MyRolesResponse, …) named in 012/013's earlier plans: rather than a
// bespoke envelope type per resource, one generic shape carries the resource as
// the type argument T — Page[Role], Page[Action], Page[Project], … — so each
// read adds only its resource model, with zero per-resource envelope boilerplate.
// It reuses the shared Pagination (created by My Roles (012), the first paginated
// /me* read) rather than forking a second pagination type (DECISIONS §109).
//
// Decoding is tolerant of an absent meta.pagination: a body that carries no
// meta.pagination block decodes Meta.Pagination to its zero value
// (HasNextPage=false), which the walker (internal/paging) treats as a single
// complete page — the non-paginated-endpoint case (e.g. the org role tree).
type Page[T any] struct {
	Data []T  `json:"data"`
	Meta Meta `json:"meta"`
}

// Meta is the envelope's meta object. It is a named type (not an anonymous
// struct) so Page[T] reads cleanly and the meta shape can grow without rewriting
// the envelope; today it carries only the shared Pagination.
type Meta struct {
	Pagination Pagination `json:"pagination"`
}
