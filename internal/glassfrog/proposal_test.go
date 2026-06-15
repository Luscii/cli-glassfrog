package glassfrog

import (
	"encoding/json"
	"strings"
	"testing"
)

// proposalCreatedBody is a representative createProposal 201 body: the single-object
// {data: Proposal} envelope carrying the prp_ id, the server-set draft status, the
// anchor tension, the response summary, and the available transitions. Null nullable
// fields exercise the nullable-as-empty-string convention.
const proposalCreatedBody = `{"data":{
  "id":"prp_0123","type":"proposal","status":"draft",
  "tension_id":"ten_0123","circle_id":"role_0123","proposer_id":"per_0123",
  "changes":[{"id":"chg_1","type":"CreateRole","name":"Scribe","extra":{"nested":true}}],
  "response_summary":{"total":3,"no_objection":2,"bring_to_meeting":1},
  "expected_response_count":3,"received_response_count":2,
  "available_transitions":["propose","withdraw"],
  "proposed_at":null,"response_deadline":null,"accepted_at":null,
  "created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-02T00:00:00Z"
}}`

// TestProposal_Decode_BindsEveryField pins that a {data: Proposal} body decodes
// through the generic Document[Proposal] and every snake_case field binds.
func TestProposal_Decode_BindsEveryField(t *testing.T) {
	var doc Document[Proposal]
	if err := json.Unmarshal([]byte(proposalCreatedBody), &doc); err != nil {
		t.Fatalf("decode: %v", err)
	}
	p := doc.Data
	if p.ID != "prp_0123" || p.Type != "proposal" || p.Status != "draft" {
		t.Errorf("id/type/status mis-bound: %+v", p)
	}
	if p.TensionID != "ten_0123" || p.CircleID != "role_0123" || p.ProposerID != "per_0123" {
		t.Errorf("anchor/circle/proposer mis-bound: %+v", p)
	}
	if p.ExpectedResponseCount != 3 || p.ReceivedResponseCount != 2 {
		t.Errorf("response counts mis-bound: %+v", p)
	}
	if got := p.AvailableTransitions; len(got) != 2 || got[0] != "propose" || got[1] != "withdraw" {
		t.Errorf("available_transitions mis-bound: %v", got)
	}
	if p.CreatedAt != "2026-01-01T00:00:00Z" || p.UpdatedAt != "2026-01-02T00:00:00Z" {
		t.Errorf("timestamps mis-bound: %+v", p)
	}
}

// TestProposal_Decode_ResponseSummaryCounts pins the nested response_summary binds
// its three aggregate counts.
func TestProposal_Decode_ResponseSummaryCounts(t *testing.T) {
	var doc Document[Proposal]
	if err := json.Unmarshal([]byte(proposalCreatedBody), &doc); err != nil {
		t.Fatalf("decode: %v", err)
	}
	rs := doc.Data.ResponseSummary
	if rs.Total != 3 || rs.NoObjection != 2 || rs.BringToMeeting != 1 {
		t.Errorf("response_summary mis-bound: %+v", rs)
	}
}

// TestProposal_Decode_NullNullablesAreEmpty pins that null tension_id/circle_id/
// proposer_id and null timestamps decode to the empty string (the
// nullable-as-empty-string convention), never the literal "null".
func TestProposal_Decode_NullNullablesAreEmpty(t *testing.T) {
	body := `{"data":{"id":"prp_1","type":"proposal","status":"draft","tension_id":null,"circle_id":null,"proposer_id":null,"proposed_at":null,"response_deadline":null,"accepted_at":null,"created_at":"t","updated_at":"t"}}`
	var doc Document[Proposal]
	if err := json.Unmarshal([]byte(body), &doc); err != nil {
		t.Fatalf("decode: %v", err)
	}
	p := doc.Data
	for name, got := range map[string]string{
		"tension_id":        p.TensionID,
		"circle_id":         p.CircleID,
		"proposer_id":       p.ProposerID,
		"proposed_at":       p.ProposedAt,
		"response_deadline": p.ResponseDeadline,
		"accepted_at":       p.AcceptedAt,
	} {
		if got != "" {
			t.Errorf("null %s should decode to empty string, got %q", name, got)
		}
	}
}

// TestProposalChange_PreservesFreeFormKeys pins that a change element preserves its
// free-form command-specific keys (the CLI never interprets them) while lifting
// id/type for projection.
func TestProposalChange_PreservesFreeFormKeys(t *testing.T) {
	var doc Document[Proposal]
	if err := json.Unmarshal([]byte(proposalCreatedBody), &doc); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(doc.Data.Changes) != 1 {
		t.Fatalf("want 1 change, got %d", len(doc.Data.Changes))
	}
	ch := doc.Data.Changes[0]
	if ch.ID != "chg_1" || ch.Type != "CreateRole" {
		t.Errorf("id/type not lifted: %+v", ch)
	}
	if name, _ := ch.Fields["name"].(string); name != "Scribe" {
		t.Errorf("free-form key \"name\" not preserved: %v", ch.Fields)
	}
	if _, ok := ch.Fields["extra"].(map[string]any); !ok {
		t.Errorf("nested free-form key \"extra\" not preserved: %v", ch.Fields)
	}
}

// TestProposal_Decode_UnknownFieldsTolerated pins forward-compatible decoding: an
// unknown/extra top-level field decodes cleanly without error.
func TestProposal_Decode_UnknownFieldsTolerated(t *testing.T) {
	body := `{"data":{"id":"prp_1","type":"proposal","status":"draft","future_field":{"x":1},"created_at":"t","updated_at":"t"}}`
	var doc Document[Proposal]
	if err := json.Unmarshal([]byte(body), &doc); err != nil {
		t.Fatalf("unknown fields should decode cleanly, got %v", err)
	}
	if doc.Data.ID != "prp_1" {
		t.Errorf("id mis-bound with extra field present: %+v", doc.Data)
	}
}

// TestCreateProposalRequest_MarshalsVerbatim pins the request body: the nested
// {"proposal":{"tension_id":…,"changes":[…]}} envelope with the supplied changes
// carried byte-for-byte, and NO status/proposer keys.
func TestCreateProposalRequest_MarshalsVerbatim(t *testing.T) {
	changes := []json.RawMessage{
		json.RawMessage(`{"type":"CreateRole","name":"Scribe","keep":[1,2,3]}`),
		json.RawMessage(`{"type":"UpdateAccountability","text":"keep this"}`),
	}
	out, err := json.Marshal(NewCreateProposalRequest("ten_0123", changes))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	want := `{"proposal":{"tension_id":"ten_0123","changes":[{"type":"CreateRole","name":"Scribe","keep":[1,2,3]},{"type":"UpdateAccountability","text":"keep this"}]}}`
	if string(out) != want {
		t.Errorf("request body mismatch:\n got: %s\nwant: %s", out, want)
	}
	// No server-owned keys leak into the request.
	for _, forbidden := range []string{`"status"`, `"proposer"`} {
		if strings.Contains(string(out), forbidden) {
			t.Errorf("request body must not carry %s: %s", forbidden, out)
		}
	}
}
