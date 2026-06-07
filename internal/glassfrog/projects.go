package glassfrog

// Project is the GET /me/projects list item: a project owned by a role the
// authenticated practitioner fills (spec/glassfrog-api-v5.yaml Project, list
// view). The My Projects projection surfaces ID, Status, Description, RoleID,
// the HasSubProjects/HasActions presence signals, and Tags — the id is the
// machine-actionable handle an agent uses in follow-up calls. The remaining
// fields (IndividualInitiative, ParentProjectID, CreatedAt/UpdatedAt, Link, Note)
// decode so the type tracks the spec shape, but the projection does not render
// them (spec Non-Behaviors). Every field carries an explicit snake_case JSON tag
// — encoding/json is case-insensitive but does NOT bridge underscores, so an
// untagged RoleID would silently never bind to the API's role_id. Decoding is
// tolerant of unknown/extra fields (forward-compatible).
//
// RoleID is nullable in the spec — null for non-role-owned (individual-initiative)
// projects. A JSON null decodes to the Go zero value (empty string); the
// projection turns that empty RoleID into an explicit no-role marker rather than
// printing a blank owning role. Description, ParentProjectID, Link, and Note are
// likewise nullable; Description is projected (a null/empty Description renders as
// the em-dash placeholder), the others are only decoded.
//
// The sub_projects/actions embed arrays are deliberately NOT modelled: the
// /me/projects operation offers no ?include parameter (ADR-2), so they never
// arrive on this read. HasSubProjects/HasActions are projected instead as
// presence signals — the reader learns children exist without fetching them.
//
// The token is never a field here — it is an X-Auth-Token request header, not a
// response field, so secret hygiene holds by construction (CONSTITUTION II).
type Project struct {
	ID                   string   `json:"id"`
	Status               string   `json:"status"`
	Description          string   `json:"description"`
	RoleID               string   `json:"role_id"`
	Tags                 []string `json:"tags"`
	HasSubProjects       bool     `json:"has_sub_projects"`
	HasActions           bool     `json:"has_actions"`
	IndividualInitiative bool     `json:"individual_initiative"`
	ParentProjectID      string   `json:"parent_project_id"`
	CreatedAt            string   `json:"created_at"`
	UpdatedAt            string   `json:"updated_at"`
	Link                 string   `json:"link"`
	Note                 string   `json:"note"`
}

// MyProjectsResponse is the GET /me/projects body: the {data: [Project], meta:
// {pagination}} list envelope. It references the shared named Pagination type
// (defined by My Roles 012, the first paginated /me* read, and reused by My
// Actions 013) rather than a second pagination type — there is exactly one
// Pagination and one envelope shape across every list read (DECISIONS
// 2026-06-07). Decoding is tolerant of unknown/extra fields.
type MyProjectsResponse struct {
	Data []Project `json:"data"`
	Meta struct {
		Pagination Pagination `json:"pagination"`
	} `json:"meta"`
}
