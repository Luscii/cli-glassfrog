package glassfrog

// Domain is an area of control a Role holds — the shared, single canonical
// domain type (011 ADR-1), grown rather than forked (033 ADR-2). It started life
// (012/025/026) as the inline-on-Role / inline-on-TreeNode projection carrying
// only ID and Description; Role Domains (033) grows it the rest of the way to the
// full getDomain / listRoleDomains spec shape — Type, RoleID, CreatedAt,
// UpdatedAt — plus an optional Policies slice that the single read embeds only
// when ?include=policies was requested. Growth is additive: the embeds on Role
// (025) and TreeNode (026) render only Description and are unperturbed by the new
// fields, which simply decode and sit at their zero values there (the 012→025
// forward-compatible growth pattern).
//
// RoleID is a *string because the spec field is nullable (a domain need not be
// bound to a controlling role); a pointer distinguishes "no controlling role"
// (nil) from a present id, so the render can show its explicit-absence marker
// rather than a fabricated or empty id (CONSTITUTION VIII). Policies reuses the
// landed Policy leaf model — never a second policy type (011 ADR-1 / 025
// precedent) — and stays nil/empty unless the embed was requested, exactly as
// Role carries its optional Domains/Accountabilities; the render guards the
// section so an absent embed never invents a value (019). Every field carries an
// explicit snake_case JSON tag — encoding/json is case-insensitive but does NOT
// bridge underscores, so an untagged field would silently never bind to the
// API's snake_case name. Decoding stays tolerant of unknown/extra fields (011),
// so the minimal embed and the full standalone bodies all share this one type.
type Domain struct {
	ID          string   `json:"id"`
	Type        string   `json:"type"`
	Description string   `json:"description"`
	RoleID      *string  `json:"role_id"`
	CreatedAt   string   `json:"created_at"`
	UpdatedAt   string   `json:"updated_at"`
	Policies    []Policy `json:"policies"`
}

// DomainDocument is the single-object {data: Domain} envelope GET /domains/{id}
// returns — the single-read counterpart to the paginated Page[T] (016) and the
// sibling of RoleDocument (025). The role-scoped list (GET /roles/{id}/domains)
// decodes the existing generic Page[Domain] instead, so no 033-local list
// envelope is defined.
type DomainDocument struct {
	Data Domain `json:"data"`
}
