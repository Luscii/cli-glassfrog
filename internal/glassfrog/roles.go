package glassfrog

// Role is the shared role shape — the single canonical role type (011 ADR-1),
// grown rather than forked. The ?include=roles embed (011) projects only ID
// (role_ handle) and Name; My Roles (012) grew it with Purpose, Domains, and
// Accountabilities; Role Reads (025) grows it the rest of the way to the full
// GET /roles / GET /roles/{id} spec shape — Type, ParentRoleID, HasSubroles,
// Flags, Fillers, and Tags. Decoding stays tolerant of extra fields, so the
// minimal embed and the full list/detail bodies all share this one type: a
// caller that reads only the embed simply leaves the grown fields at their zero
// values. Every field carries an explicit snake_case JSON tag — encoding/json is
// case-insensitive but does NOT bridge underscores, so an untagged field would
// silently never bind to the API's snake_case name.
//
// ParentRoleID is a *string because the API field is nullable (null for an
// anchor role); a pointer distinguishes "no parent" (nil) from a present id.
// Fillers reuses the shared Actor type. Domains and Accountabilities each
// surface the item's Description (and id); the render projections show the
// description. The list decodes Page[Role] (016); the detail view is RoleDetail
// (below), which embeds this type.
type Role struct {
	ID               string           `json:"id"`
	Type             string           `json:"type"`
	Name             string           `json:"name"`
	Purpose          string           `json:"purpose"`
	ParentRoleID     *string          `json:"parent_role_id"`
	HasSubroles      bool             `json:"has_subroles"`
	Flags            []string         `json:"flags"`
	Accountabilities []Accountability `json:"accountabilities"`
	Domains          []Domain         `json:"domains"`
	Fillers          []Actor          `json:"fillers"`
	Tags             []string         `json:"tags"`
}

// Accountability is an ongoing activity a Role is accountable for. The
// projections surface its Description; the API carries an id too, ignored by the
// render but decoded so it is available to callers.
type Accountability struct {
	ID          string `json:"id"`
	Description string `json:"description"`
}

// Domain is an area of authority a Role controls (the spec's DomainRef on a
// Role). The projections surface its Description; the id is decoded but not
// rendered.
type Domain struct {
	ID          string `json:"id"`
	Description string `json:"description"`
}

// RoleDetail is the GET /roles/{id} body's data: a Role plus the optional
// related resources the read embeds when ?include= requests them (ADR-2). Each
// related field stays nil/empty unless its include value was requested, so the
// render templates guard every section (omit when unrequested, explicit-absence
// marker when requested-but-empty). Nested Subroles and ParentRole are plain
// Role (not RoleDetail) — the API embeds one level only, so there is no
// recursion. The leaf related models (Assignment, Policy, Note, SkillSummary)
// live here for reuse by the downstream per-role reads (#33/#34/#38), which this
// type's design sets the precedent for.
type RoleDetail struct {
	Role
	Assignments []Assignment   `json:"assignments"`
	Subroles    []Role         `json:"subroles"`
	ParentRole  *Role          `json:"parent_role"`
	Policies    []Policy       `json:"policies"`
	Notes       []Note         `json:"notes"`
	Skills      []SkillSummary `json:"skills"`
}

// RoleDocument is the single-object {data: RoleDetail} envelope GET /roles/{id}
// returns — the single-read counterpart to the paginated Page[T] (016). Role
// Reads (025) introduced it as a named type and its comment invited a later
// single-object read to generalize it; Role Policies (034) is that read, so it is
// now a type alias of the generic Document[T] (document.go). The alias keeps 025's
// decode call site and BDD byte-stable — `var doc RoleDocument` and `doc.Data`
// (a RoleDetail) read exactly as before — while the single policy read decodes
// the same generic envelope as Document[Policy].
type RoleDocument = Document[RoleDetail]

// Assignment maps an actor to a role (the spec's Assignment). ID, ActorID, and
// RoleID are always present; the focus/election/timestamp fields decode but the
// Role Reads projection surfaces only the actor reference. Reused by
// ?include=assignments here and the future standalone assignment reads.
type Assignment struct {
	ID           string `json:"id"`
	ActorID      string `json:"actor_id"`
	RoleID       string `json:"role_id"`
	Focus        string `json:"focus"`
	ElectedUntil string `json:"elected_until"`
	// Actor is the embedded actor when the API includes it (it does by default on
	// the assignments endpoints); on a role's ?include=assignments it may be
	// absent, leaving Name empty.
	Actor struct {
		ID   string `json:"id"`
		Name string `json:"name"`
		Kind string `json:"kind"`
	} `json:"actor"`
}

// Policy is a governance rule on a role's interior (the spec's Policy). Title is
// the projected field; Body carries the full text (may be HTML) and is decoded
// for callers that want it. Role Reads (025) introduced it minimal (ID/Title/Body)
// for the embedded ?include=policies view; Role Policies (034) grows it the rest
// of the way to the full GET /policies/{id} spec shape — RoleID, DomainID,
// CreatedAt, UpdatedAt — so the standalone read can show which role/domain a
// policy governs and when it changed. One canonical type, grown not forked (011
// ADR-1): 025's embedded render reads only ID/Title/Body and is untouched by the
// new fields. role_id and domain_id are nullable in spec.yaml — modeled as plain
// strings (empty = null), mirroring the existing nullable Body — so a role-level
// policy (null domain_id) or an org-level policy (null role_id) decodes to an
// empty string, never a panic, and the render guards explicit-absence on it.
type Policy struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Body      string `json:"body"`
	RoleID    string `json:"role_id"`
	DomainID  string `json:"domain_id"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// Note is a note on a role (the spec's Note: required title + body, no `text`
// field). Title is projected; Body and the role/timestamp fields decode for
// callers. Reused by future note reads.
type Note struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Body      string `json:"body"`
	RoleID    string `json:"role_id"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// SkillSummary is a skill as embedded in ?include=skills: name only, no Content
// (the API omits the full markdown body in summaries — fetch it via a future
// GET /skills/{id}). The render reflects "summary" so it never implies full
// content is present.
type SkillSummary struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
