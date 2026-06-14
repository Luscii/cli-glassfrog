package glassfrog

// ActorDetail is the GET /actors/{id} body's data: an Actor plus the optional
// related resources the single read embeds when ?include= requests them — the
// RoleDetail shape (025 ADR-2) applied to the actor. It embeds the unchanged
// landed Actor (011, reused as-is by the directory 048 and `me` 011) and adds two
// optional embed slices that decode from the `roles`/`assignments` JSON keys. Each
// stays nil/empty unless its include value was requested (the API omits the field
// otherwise), so the render templates guard every embed section (omit when
// unrequested, explicit-absence marker when requested-but-empty — 019).
//
// The embedded Role (its FULL landed shape — purpose, accountabilities, domains)
// and Assignment are the landed 025 types reused verbatim (roles.go), NOT new leaf
// models: an actor's governance footprint is assembled entirely from already-landed
// types. Actor, Role, and Assignment are unchanged (011 ADR-1 — grow the schema,
// don't duplicate; the embed fields live on the detail type exactly as RoleDetail
// carries Role's related resources). Decoding stays tolerant of unknown/extra
// fields (the package contract).
type ActorDetail struct {
	Actor
	Roles       []Role       `json:"roles"`
	Assignments []Assignment `json:"assignments"`
}

// ActorDocument is the single-object {data: ActorDetail} envelope GET /actors/{id}
// returns — the single-read counterpart to the paginated Page[Actor] the directory
// (048) decodes. It is a type alias of the generic Document[T] (document.go), the
// same envelope Role Reads' RoleDocument and the single policy read use, so the
// single-actor read adds only its resource model with no per-resource envelope
// boilerplate.
type ActorDocument = Document[ActorDetail]
