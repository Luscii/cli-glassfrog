package glassfrog

import (
	"encoding/json"
	"reflect"
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

// TestProposal_Decode_PageOfProposals pins that a {data:[…]} list body decodes
// through the generic Page[Proposal] — the shape the global `proposal list` walk
// (056) reads — binding each element's fields and the pagination meta. The same
// Proposal model the single read (Document[Proposal]) uses serves the list element.
func TestProposal_Decode_PageOfProposals(t *testing.T) {
	body := `{"data":[
	  {"id":"prp_1","type":"proposal","status":"draft","tension_id":"ten_1","circle_id":"role_1","proposer_id":"per_1","changes":[{"id":"chg_1","type":"CreateRole"}],"response_summary":{"total":1,"no_objection":1,"bring_to_meeting":0},"available_transitions":["propose"],"created_at":"t","updated_at":"t"},
	  {"id":"prp_2","type":"proposal","status":"proposed_outside_meeting","tension_id":null,"circle_id":null,"proposer_id":null,"changes":[],"response_summary":{"total":0,"no_objection":0,"bring_to_meeting":0},"created_at":"t","updated_at":"t"}
	],"meta":{"pagination":{"per_page":100,"has_next_page":false,"next_cursor":""}}}`
	var page Page[Proposal]
	if err := json.Unmarshal([]byte(body), &page); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(page.Data) != 2 {
		t.Fatalf("want 2 proposals, got %d", len(page.Data))
	}
	if page.Data[0].ID != "prp_1" || page.Data[0].Status != "draft" {
		t.Errorf("first proposal mis-bound: %+v", page.Data[0])
	}
	// The second element's null anchors decode to empty strings (the nullable-as-empty
	// convention), proving the list element shares the single-read decode posture.
	if p := page.Data[1]; p.TensionID != "" || p.CircleID != "" || p.ProposerID != "" {
		t.Errorf("null anchors on a list element should decode to empty strings: %+v", p)
	}
	// The {meta:{pagination}} block binds too — the walker reads HasNextPage/NextCursor
	// to decide whether to fetch another page, so Page[Proposal] must surface them.
	if pg := page.Meta.Pagination; pg.PerPage != 100 || pg.HasNextPage || pg.NextCursor != "" {
		t.Errorf("pagination meta mis-bound: %+v", pg)
	}
}

// TestResponseSummary_AggregateOnly pins the anti-attribution non-behavior at the
// type level (spec 056 non-behavior): ResponseSummary exposes ONLY the three
// aggregate counts — there is no field that could carry a per-person attribution
// (no actor/person id, no per-responder breakdown). A reflect-over-the-fields guard
// fails loud if a future edit adds an attribution-shaped field.
func TestResponseSummary_AggregateOnly(t *testing.T) {
	rt := reflect.TypeOf(ResponseSummary{})
	if rt.NumField() != 3 {
		t.Fatalf("ResponseSummary must expose exactly the three aggregate counts, got %d fields", rt.NumField())
	}
	want := map[string]bool{"Total": true, "NoObjection": true, "BringToMeeting": true}
	for i := 0; i < rt.NumField(); i++ {
		f := rt.Field(i)
		if !want[f.Name] {
			t.Errorf("ResponseSummary carries an unexpected field %q — no per-person attribution may be added", f.Name)
		}
		if f.Type.Kind() != reflect.Int {
			t.Errorf("aggregate count %q must be an int, got %s", f.Name, f.Type)
		}
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
