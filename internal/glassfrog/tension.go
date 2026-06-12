package glassfrog

// Tension is the v5 Tension schema: an issue or gap a role-filler senses, and the
// required seed of a governance proposal (the created tension's ten_ id is the
// load-bearing handle a later proposal references as tension_id). It is the
// response shape the createTension 201 returns inside a {data: Tension} envelope,
// decoded via the generic Document[Tension] (034) — no per-resource envelope.
//
// Every field carries an explicit snake_case JSON tag — encoding/json is
// case-insensitive but does NOT bridge underscores, so an untagged RoleID would
// silently never bind to the API's role_id. Decoding is tolerant of unknown/extra
// fields (forward-compatible).
//
// The optional fields (RoleID, SensedByID, Label, MeetingType, ParentRoleID) are
// modeled as plain strings — a JSON null decodes to the empty string, the
// nullable-as-empty-string convention the landed models use (Policy.Body,
// Project.RoleID); the render guards explicit-absence on each rather than printing
// a blank. Body is modeled the same way (it is nullable in the v5 schema, so a wire
// null decodes to empty), but it is the required, non-empty primary content of a
// captured tension (the command rejects an empty --body), so the render shows it
// verbatim rather than behind an absence guard. Status is server-computed
// (unprocessed/processed/archived); the client never sends it. The sensing person
// (SensedByID) is derived from the token and never supplied by the client.
//
// The token is never a field here — it is an X-Auth-Token request header, not a
// response field, so secret hygiene holds by construction (CONSTITUTION II).
type Tension struct {
	ID           string `json:"id"`
	Type         string `json:"type"`
	Body         string `json:"body"`
	Status       string `json:"status"`
	RoleID       string `json:"role_id"`
	SensedByID   string `json:"sensed_by_id"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
	Label        string `json:"label"`
	MeetingType  string `json:"meeting_type"`
	ParentRoleID string `json:"parent_role_id"`
}

// TensionInput is the createTension request body: the nested
// {"tension": {"body": …, "label"?: …, "meeting_type"?: …}} envelope. Body is
// always serialized; Label and MeetingType use omitempty so an absent flag sends
// no field at all (the command sets them only when the flag was provided AND
// non-empty). There is NO status field (the server auto-computes it) and NO
// sensed_by field (the server derives the sensing person from the token) — the
// input must not claim a state or identity the server owns.
type TensionInput struct {
	Tension TensionInputBody `json:"tension"`
}

// TensionInputBody is the inner object of TensionInput. omitempty on Label and
// MeetingType keeps an absent optional field off the wire; Body has no omitempty
// because the command guarantees it is non-empty (a bodyless tension is rejected
// before any request).
type TensionInputBody struct {
	Body        string `json:"body"`
	Label       string `json:"label,omitempty"`
	MeetingType string `json:"meeting_type,omitempty"`
}

// NewTensionInput builds the create body from the validated inputs: body is
// required (the caller has already rejected an empty/whitespace-only body), label
// and meeting-type are passed only when the caller supplies a non-empty value (an
// empty string is omitted by the omitempty tags). It marshals to
// {"tension":{"body":…[,"label":…][,"meeting_type":…]}}.
func NewTensionInput(body, label, meetingType string) TensionInput {
	return TensionInput{Tension: TensionInputBody{
		Body:        body,
		Label:       label,
		MeetingType: meetingType,
	}}
}
