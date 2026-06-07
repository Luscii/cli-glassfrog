package glassfrog

// Action is the GET /me/actions list item: a next-action owned by a role the
// authenticated practitioner fills (spec/glassfrog-api-v5.yaml Action). The My
// Actions projection surfaces ID, Status, Description, RoleID, and Tags — the
// id is the machine-actionable handle an agent uses in follow-up calls. The
// remaining fields (IndividualInitiative, ParentProjectID, CreatedAt/UpdatedAt,
// Permissions, TriggerEvent, Note) decode so the type tracks the spec shape, but
// the projection does not render them (spec Non-Behaviors). Every field carries
// an explicit snake_case JSON tag — encoding/json is case-insensitive but does
// NOT bridge underscores, so an untagged RoleID would silently never bind to the
// API's role_id. Decoding is tolerant of unknown/extra fields (forward-compatible).
//
// Description and ParentProjectID are both nullable in the spec; a JSON null
// decodes to the Go zero value (empty string). They differ at the projection:
// Description is projected, so a null/empty Description renders as the em-dash
// placeholder; ParentProjectID is only decoded (not projected), so its null
// simply stays an empty string and is never rendered. The token is never a field
// here — it is a request header, not a response field, so secret hygiene holds by
// construction (CONSTITUTION II).
//
// Permissions is decoded as a free-form map rather than a named type: the My
// Actions read never projects it, and modelling it would couple this leaf schema
// to a Permissions shape no read command surfaces yet.
type Action struct {
	ID                   string         `json:"id"`
	Status               string         `json:"status"`
	Description          string         `json:"description"`
	RoleID               string         `json:"role_id"`
	Tags                 []string       `json:"tags"`
	IndividualInitiative bool           `json:"individual_initiative"`
	ParentProjectID      string         `json:"parent_project_id"`
	CreatedAt            string         `json:"created_at"`
	UpdatedAt            string         `json:"updated_at"`
	Permissions          map[string]any `json:"permissions"`
	TriggerEvent         string         `json:"trigger_event"`
	Note                 string         `json:"note"`
}

// MyActionsResponse is the GET /me/actions body: the {data: [Action], meta:
// {pagination}} list envelope. It references the shared named Pagination type
// (defined by My Roles 012, the first paginated /me* read) rather than a second
// pagination type — there is exactly one Pagination and one envelope shape across
// every list read (DECISIONS 2026-06-07). Decoding is tolerant of unknown/extra
// fields.
type MyActionsResponse struct {
	Data []Action `json:"data"`
	Meta struct {
		Pagination Pagination `json:"pagination"`
	} `json:"meta"`
}
