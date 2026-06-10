package glassfrog

// Document is the generic {data: T} single-object envelope a single-resource read
// decodes into — the single-read counterpart to the paginated Page[T] (016). It
// generalizes Role Reads' named RoleDocument the way Page[T] generalized the
// per-resource list envelopes: rather than a bespoke {data: X} type per
// single-object read, one generic shape carries the resource as the type argument
// T — Document[RoleDetail] (025, via the RoleDocument alias), Document[Policy]
// (034, the single policy read), and every future single-object read (a project, a
// note, a skill) — so each read adds only its resource model, with zero
// per-resource envelope boilerplate.
//
// Decoding is tolerant of unknown/extra fields (encoding/json ignores them), so a
// body that carries siblings of "data" decodes cleanly and only Data is bound.
type Document[T any] struct {
	Data T `json:"data"`
}
