package glassfrog

import "encoding/json"

// Proposal is the v5 Proposal schema: a governance change set raised against an
// anchor tension, and the anchor of the write path (the created proposal's prp_ id
// is the load-bearing handle a later step references to advance it to circulation).
// It is the response shape the createProposal 201 returns inside a {data: Proposal}
// envelope, decoded via the generic Document[Proposal] (034) — no per-resource
// envelope. The same shape is read by Proposal Reads (056); this model is shared
// between the two specs, created by whichever lands first (056 ADR-2 / 055 ADR-4).
//
// Every field carries an explicit snake_case JSON tag — encoding/json is
// case-insensitive but does NOT bridge underscores, so an untagged TensionID would
// silently never bind to the API's tension_id. Decoding is tolerant of unknown/extra
// fields (forward-compatible).
//
// The nullable fields (TensionID, CircleID, ProposerID, ProposedAt, ResponseDeadline,
// AcceptedAt) are modeled as plain strings — a JSON null decodes to the empty string,
// the nullable-as-empty-string convention the landed models use (Tension.RoleID,
// Policy.Body); the render guards explicit-absence on each rather than printing a
// blank. Status is server-computed (the create returns draft); the client never sends
// it. The proposer is derived from the token and never supplied by the client.
//
// The token is never a field here — it is an X-Auth-Token request header, not a
// response field, so secret hygiene holds by construction (CONSTITUTION II).
type Proposal struct {
	ID                    string           `json:"id"`
	Type                  string           `json:"type"`
	Status                string           `json:"status"`
	TensionID             string           `json:"tension_id"`
	CircleID              string           `json:"circle_id"`
	ProposerID            string           `json:"proposer_id"`
	Changes               []ProposalChange `json:"changes"`
	ResponseSummary       ResponseSummary  `json:"response_summary"`
	ExpectedResponseCount int              `json:"expected_response_count"`
	ReceivedResponseCount int              `json:"received_response_count"`

	// Valid is the server's own verdict on this proposal. It is a POINTER, not a
	// bool — a deliberate divergence from the model's nullable-as-empty-string
	// convention (TensionID, Tension.RoleID): an empty string stands in fine for
	// an absent id because no id is ever legitimately empty, whereas `false` is a
	// legitimate value of valid, so the absent case needs its own representation.
	// The field is NOT declared in spec/glassfrog-api-v5.yaml, so it may be
	// absent; nil means the server stated no verdict — it never means valid and
	// never means invalid. Observed carried by getProposal and NOT by
	// listProposals (074 probe).
	Valid *bool `json:"valid"`

	// ValidationAlerts carries the server's blocking and advisory alerts on this
	// proposal. Also undeclared in the vendored contract. A nil slice means the
	// key was absent or null; a non-nil empty slice means the server stated an
	// empty list — both mean "no alerts", and neither is a validity verdict on
	// its own (an entry carries its own severity).
	ValidationAlerts []ValidationAlert `json:"validation_alerts"`

	AvailableTransitions []string `json:"available_transitions"`
	ProposedAt           string   `json:"proposed_at"`
	ResponseDeadline     string   `json:"response_deadline"`
	AcceptedAt           string   `json:"accepted_at"`
	CreatedAt            string   `json:"created_at"`
	UpdatedAt            string   `json:"updated_at"`
}

// ValidationAlert is one entry of a proposal's validation_alerts. Undeclared in
// the v5 contract and observed live (074 probe): a three-key object carrying the
// severity, the element path the alert concerns, and the server's own message.
// Typed rather than a free-form map because all three keys are rendered; decoding
// stays forward-compatible, so an added key is ignored rather than fatal.
type ValidationAlert struct {
	Severity string `json:"severity"`
	Path     string `json:"path"`
	Message  string `json:"message"`
}

// ProposalChange is the RESPONSE decode of each element of a proposal's change set: a
// free-form governance command with an id and a type, plus open, command-specific
// keys the CLI never interprets (the actions.go Permissions precedent — 056 ADR-2).
// ID and Type are surfaced for projection; Fields preserves the WHOLE decoded object
// (id/type included) so every command-specific key rides through untouched and a
// later richer view (056) can render changes by type without a per-type schema.
//
// This is the RESPONSE shape. The createProposal REQUEST carries changes as
// []json.RawMessage (verbatim send — see CreateProposalBody), a deliberately distinct
// write-side type: the request never reshapes what the caller supplied, and the
// response never needs to.
type ProposalChange struct {
	ID     string
	Type   string
	Fields map[string]any
}

// UnmarshalJSON decodes a change element into its full key set (preserving every
// free-form command-specific key in Fields) and lifts the id/type out for
// projection. A non-object element fails to decode into the map, surfacing as a
// decode error rather than a silent zero value.
func (c *ProposalChange) UnmarshalJSON(data []byte) error {
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}
	c.Fields = m
	if id, ok := m["id"].(string); ok {
		c.ID = id
	}
	if t, ok := m["type"].(string); ok {
		c.Type = t
	}
	return nil
}

// ResponseSummary is the aggregate response_summary object on a Proposal (no
// per-person attribution): the total responses and the no-objection /
// bring-to-meeting tallies. On a fresh draft these are typically 0. Shared with
// Proposal Reads (056).
type ResponseSummary struct {
	Total          int `json:"total"`
	NoObjection    int `json:"no_objection"`
	BringToMeeting int `json:"bring_to_meeting"`
}

// CreateProposalRequest is the createProposal request body: the nested
// {"proposal": {"tension_id": …, "changes": [ …verbatim… ]}} envelope. This is 055's
// EXCLUSIVE net-new surface — 056 (reads) never needs it. There is NO proposer field
// (the server derives the proposer from the token) and NO status field (the server
// sets the created proposal to draft) — the input must not claim an identity or state
// the server owns. The changes slice is carried as []json.RawMessage so each supplied
// change is sent BYTE-FOR-BYTE above the type floor (ADR-3): the CLI neither reshapes
// nor drops any command-specific key. The token is never a field here.
type CreateProposalRequest struct {
	Proposal CreateProposalBody `json:"proposal"`
}

// CreateProposalBody is the inner object of CreateProposalRequest. TensionID is always
// serialized (the anchor is required); Changes is the validated, non-empty
// []json.RawMessage carried verbatim — no omitempty, because the command guarantees a
// non-empty array before marshalling (a changeless proposal is rejected pre-request).
type CreateProposalBody struct {
	TensionID string            `json:"tension_id"`
	Changes   []json.RawMessage `json:"changes"`
}

// NewCreateProposalRequest builds the create body from the validated inputs: the
// anchor tension id and the verbatim change slice (already parsed and floored by the
// command — non-empty, every element a typed object). It marshals to
// {"proposal":{"tension_id":…,"changes":[…verbatim…]}} with no added keys.
func NewCreateProposalRequest(tensionID string, changes []json.RawMessage) CreateProposalRequest {
	return CreateProposalRequest{Proposal: CreateProposalBody{
		TensionID: tensionID,
		Changes:   changes,
	}}
}

// ProposalVote is the v5 ProposalVote schema: a single consent-window response
// recorded against a circulating proposal — the response shape the
// createProposalResponse 201 returns inside a {data: ProposalVote} envelope,
// decoded via the generic Document[ProposalVote] (no per-resource envelope). It is
// a DISTINCT schema from Proposal (058 ADR-2): it carries no changes/response_summary
// and is never a grow of it. The recorded vote leads with its prr_ id and surfaces
// the parent proposal's status at response time — ProposalStatus reads `accepted`
// when this very response triggered auto-acceptance (the agent-visible signal that
// the consent window closed), which the CLI surfaces without computing acceptance.
//
// Every field carries an explicit snake_case JSON tag — encoding/json is
// case-insensitive but does NOT bridge underscores, so an untagged ProposalID would
// silently never bind to the API's proposal_id. Decoding is tolerant of unknown/extra
// fields (forward-compatible — the Tension/Proposal shape).
//
// ProposalID is nullable: a JSON null (or absent key) decodes to the empty string,
// the nullable-as-empty-string convention the landed models use (Proposal.TensionID,
// Tension.RoleID); the render guards explicit-absence rather than printing a blank.
//
// There is NO per-person attribution on the vote — no actor/person field — matching
// the API ("summary stats only on the proposal resource") and 056's ResponseSummary
// anti-attribution non-behavior. The responding person is derived from the token and
// is never a field here; the token itself rides 007's X-Auth-Token request header, so
// secret hygiene holds by construction (CONSTITUTION II).
type ProposalVote struct {
	ID             string `json:"id"`
	Type           string `json:"type"`
	ProposalID     string `json:"proposal_id"`
	ProposalStatus string `json:"proposal_status"`
	Value          string `json:"value"`
	CreatedAt      string `json:"created_at"`
	UpdatedAt      string `json:"updated_at"`
}

// ProposalResponseInput is the createProposalResponse request body: the nested
// {"response": {"value": …}} envelope (CreateProposalResponseRequest). This is 058's
// EXCLUSIVE net-new request surface — neither 055 (create) nor 056 (reads) needs it.
// There is NO person field (the server derives the responding person from the token —
// the recording carries no identity claim) and NO status field (the parent proposal's
// status is server-owned). The token is never a field here.
type ProposalResponseInput struct {
	Response ProposalResponseBody `json:"response"`
}

// ProposalResponseBody is the inner object of ProposalResponseInput. Value is always
// serialized (no omitempty): the command guarantees a non-empty, validated consent
// value before marshalling (an omitted/unsupported --response is rejected pre-request),
// so the body always carries response.value — the 042 TensionInputBody shape.
type ProposalResponseBody struct {
	Value string `json:"value"`
}

// NewProposalResponseInput builds the response body from the validated consent value
// (already checked against the closed response vocabulary by the command). It marshals
// to {"response":{"value":…}} with no added keys — no person, no status.
func NewProposalResponseInput(value string) ProposalResponseInput {
	return ProposalResponseInput{Response: ProposalResponseBody{Value: value}}
}
