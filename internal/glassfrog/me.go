// Package glassfrog holds the API resource schema — plain JSON-tagged structs
// the read surface decodes API responses into. It is a leaf package: it imports
// nothing internal (no transport, no cobra, no exit codes), so both
// internal/cli (commands) and internal/apiclient (transport) may import it
// without a dependency cycle (ADR-1). Decoding is tolerant of unknown/extra
// fields — the structs surface only the fields the read commands project, and
// the API may add more without breaking a decode (forward-compatible).
//
// The token is never a field here: it is an X-Auth-Token request header, not a
// response field, so secret hygiene holds by construction (CONSTITUTION II).
package glassfrog

// MeResponse is the GET /me body: the authenticated actor bundled with its
// organization and membership, plus the roles it fills when ?include=roles was
// requested. Roles is populated only on that opt-in embed; it is empty/nil
// otherwise (the API omits the field). The full role shape (accountabilities,
// domains, assignments) is My Roles (012)'s growth of the shared Role type —
// 011 decodes only what the embed projects.
type MeResponse struct {
	Actor        Actor        `json:"actor"`
	Organization Organization `json:"organization"`
	Membership   Membership   `json:"membership"`
	Roles        []Role       `json:"roles"`
}

// Actor is a person or agent in the organization. ID carries the per_ (human) /
// agt_ (agent) prefix — the machine-actionable handle an agent uses in
// follow-up calls — and Kind is human|agent. CreatedAt/UpdatedAt decode but the
// me projection does not surface them.
type Actor struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Kind      string `json:"kind"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// Organization is the org an API key is scoped to. ID is the org_ handle.
type Organization struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Membership is the actor's membership in the organization. Only AccessLevel
// (admin|normal) is projected; ID/ActorID/OrganizationID decode but are not
// surfaced.
type Membership struct {
	ID             string `json:"id"`
	ActorID        string `json:"actor_id"`
	OrganizationID string `json:"organization_id"`
	AccessLevel    string `json:"access_level"`
}

// Role is the shared role shape. The ?include=roles embed (011) projects only
// ID (role_ / circle_ handle) and Name; My Roles (012) grew THIS SAME type with
// the full GET /me/roles shape — Purpose plus the role's Domains and
// Accountabilities — rather than defining a second role type (ADR-1, ADR-4).
// Decoding is tolerant of extra fields, so the minimal embed and the fuller
// /me/roles body share one type: the embed simply leaves the grown fields at
// their zero values. Every field carries an explicit snake_case JSON tag —
// encoding/json is case-insensitive but does NOT bridge underscores, so an
// untagged field would silently never bind to the API's snake_case name.
//
// Domains and Accountabilities each surface only the item's Description text;
// the API carries more per item, but the My Roles projection shows description
// only. Fillers, tags, and classification flags are deliberately absent — the
// projection never surfaces them (spec Non-Behaviors), so they are not fields.
// Domain and Accountability are distinct named types (not a shared one) — they
// are different governance concepts that will grow independently as the schema
// does, and naming them avoids forcing callers/tests to repeat an anonymous
// struct literal.
type Role struct {
	ID               string           `json:"id"`
	Name             string           `json:"name"`
	Purpose          string           `json:"purpose"`
	Accountabilities []Accountability `json:"accountabilities"`
	Domains          []Domain         `json:"domains"`
}

// Accountability is an ongoing activity a Role is accountable for. The My Roles
// projection surfaces only its Description; the API carries more, ignored here.
type Accountability struct {
	Description string `json:"description"`
}

// Domain is an area of authority a Role controls. The My Roles projection
// surfaces only its Description; the API carries more, ignored here.
type Domain struct {
	Description string `json:"description"`
}

// Pagination is the shared list-pagination model carried in a list read's
// meta.pagination (created here by My Roles (012), the first paginated /me*
// read, and reused verbatim by My Actions (013) and My Projects (014) — never a
// second pagination type; DECISIONS 2026-06-07). PerPage is the page size the
// API applied; HasNextPage reports that more results exist than this response
// carried; NextCursor is the ?cursor= value the next page would use. My Roles
// (012) decodes NextCursor but does not yet use it — following pagination is the
// deferred Pagination capability (016); 012 signals incompleteness only.
type Pagination struct {
	PerPage     int    `json:"per_page"`
	HasNextPage bool   `json:"has_next_page"`
	NextCursor  string `json:"next_cursor"`
}

// MyRolesResponse is the GET /me/roles body: the {data: [Role], meta:
// {pagination}} list envelope. It references the shared named Pagination type
// (not an anonymous struct) so the same envelope shape is reused by later list
// reads. Decoding is tolerant of unknown/extra fields.
type MyRolesResponse struct {
	Data []Role `json:"data"`
	Meta struct {
		Pagination Pagination `json:"pagination"`
	} `json:"meta"`
}
