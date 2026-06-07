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

// Role is the minimal role shape the ?include=roles embed projects: ID (role_ /
// circle_ handle) and Name. My Roles (012) grows THIS type to the full role
// shape — never a second role type (ADR-1).
type Role struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
