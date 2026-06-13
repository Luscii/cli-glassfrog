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

// TensionUpdateInput is the updateTension PATCH request body: the same nested
// {"tension": {…}} envelope as capture, but a SIBLING type rather than a reuse of
// TensionInput (plan ADR-1). Update is a true partial update: ALL FOUR inner fields
// use omitempty (including Status), so only the supplied fields ride the wire and
// the rest are left untouched server-side. Status IS present here — unlike capture,
// the API allows an explicit transition (notably `archived`) on PATCH and recomputes
// it on save — whereas capture's TensionInputBody deliberately has NO status field
// (creation cannot claim a server-owned status). Forking keeps capture's "no status
// field / Body non-omitempty" invariant byte-stable and load-bearing while giving
// update an honest partial-update contract. The token is never a field here.
type TensionUpdateInput struct {
	Tension TensionUpdateBody `json:"tension"`
}

// TensionUpdateBody is the inner object of TensionUpdateInput. Every field uses
// omitempty so an unsupplied field is dropped from the wire — the command resolves
// the send-set (presence + non-empty value) and a still-empty value is omitted,
// never sent as JSON null (no clear-to-null affordance — spec non-behavior). Status
// is present here (unlike capture's TensionInputBody) because update may set it.
type TensionUpdateBody struct {
	Body        string `json:"body,omitempty"`
	Label       string `json:"label,omitempty"`
	Status      string `json:"status,omitempty"`
	MeetingType string `json:"meeting_type,omitempty"`
}

// NewTensionUpdateInput builds the partial-update body from the already-resolved
// (presence-filtered, validated) field values: each value rides only when the
// command supplied it non-empty, and omitempty drops the rest. Because every
// supplied value is guaranteed non-empty by the command (a blank --body is rejected;
// --status/--meeting-type are closed-enum; --label rides only when non-empty),
// omitempty + plain strings faithfully expresses "send only what was supplied" with
// no pointer fields. Mirrors NewTensionInput (042) over the four-field,
// all-omitempty body. It marshals to {"tension":{ … only the non-empty fields … }}.
func NewTensionUpdateInput(body, label, status, meetingType string) TensionUpdateInput {
	return TensionUpdateInput{Tension: TensionUpdateBody{
		Body:        body,
		Label:       label,
		Status:      status,
		MeetingType: meetingType,
	}}
}
