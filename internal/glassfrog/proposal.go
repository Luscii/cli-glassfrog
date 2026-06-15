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
	AvailableTransitions  []string         `json:"available_transitions"`
	ProposedAt            string           `json:"proposed_at"`
	ResponseDeadline      string           `json:"response_deadline"`
	AcceptedAt            string           `json:"accepted_at"`
	CreatedAt             string           `json:"created_at"`
	UpdatedAt             string           `json:"updated_at"`
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
