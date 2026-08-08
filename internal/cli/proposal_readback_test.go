package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/Luscii/cli-glassfrog/internal/apiclient"
)

// proposalReadBackBody is a representative getProposal 200 body: the same
// {data: Proposal} shape as the create's 201, but carrying the two undeclared
// verdict fields (valid, validation_alerts) the read-back exists to obtain.
const proposalReadBackBody = `{"data":{"id":"prp_0123","type":"proposal","status":"draft","tension_id":"ten_0123","circle_id":"role_0123","proposer_id":"per_0123","changes":[{"id":"chg_1","type":"CreateRole","name":"Scribe"}],"response_summary":{"total":0,"no_objection":0,"bring_to_meeting":0},"expected_response_count":0,"received_response_count":0,"valid":false,"validation_alerts":[{"severity":"error","path":"name","message":"Can't update the Cloud Foundations role during this meeting."}],"available_transitions":[],"proposed_at":null,"response_deadline":null,"accepted_at":null,"created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"}}`

// newReadBackExecutor builds the same retrying executor the create hands the
// helper — a real client + RetryExecutor over the fake transport — so the
// helper's error mapping is exercised against the genuine error shapes the
// executor produces (TransportError / ResponseError / DecodeError), not
// hand-built ones.
func newReadBackExecutor(t *testing.T, tr http.RoundTripper) executor {
	t.Helper()
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}
	client, err := seam.newClient(seam.assemble("", false))
	if err != nil {
		t.Fatalf("newClient: %v", err)
	}
	var stderr bytes.Buffer
	return apiclient.NewRetryExecutor(client, apiclient.DefaultRetryPolicy, seam.sleep(), &stderr)
}

// TestReadBackProposalVerdict_Success pins the answered read-back: an empty
// reason, the decoded proposal (verdict fields bound), and the raw bytes for the
// machine path's verbatim emission.
func TestReadBackProposalVerdict_Success(t *testing.T) {
	tr := &tensionTransport{status: 200, body: proposalReadBackBody}
	exec := newReadBackExecutor(t, tr)

	p, raw, reason := readBackProposalVerdict(context.Background(), exec, "prp_0123")
	if reason != "" {
		t.Fatalf("a successful read-back must return an empty reason, got %q", reason)
	}
	if p.ID != "prp_0123" {
		t.Errorf("decoded proposal id = %q, want prp_0123", p.ID)
	}
	if p.Valid == nil || *p.Valid != false {
		t.Errorf("decoded Valid = %v, want non-nil false", p.Valid)
	}
	if len(p.ValidationAlerts) != 1 || p.ValidationAlerts[0].Severity != "error" {
		t.Errorf("decoded alerts mis-bound: %+v", p.ValidationAlerts)
	}
	if string(raw) != proposalReadBackBody {
		t.Errorf("raw bytes must be the server's document verbatim:\n got: %s\nwant: %s", raw, proposalReadBackBody)
	}
	if tr.calls != 1 {
		t.Errorf("a read-back is exactly one request, got %d", tr.calls)
	}
	if tr.lastMethod != http.MethodGet || !strings.HasSuffix(tr.lastPath, "/proposals/prp_0123") {
		t.Errorf("read-back must GET /proposals/{id}, got %s %s", tr.lastMethod, tr.lastPath)
	}
}

// TestReadBackProposalVerdict_FailureReasonsDistinct pins that a wire failure, a
// non-2xx, a post-retry 429, and an undecodable body each return a DISTINCT
// reason and never an error — the interface-cli reason vocabulary, one per
// failure family. The 429 reason names the exhausted request budget.
func TestReadBackProposalVerdict_FailureReasonsDistinct(t *testing.T) {
	cases := []struct {
		name string
		tr   *tensionTransport
		// wantPrefix pins the reason's failure-family shape; the wire family's
		// parenthetical carries a transport-specific cause, so the prefix is
		// the stable part.
		wantPrefix string
	}{
		{
			name:       "wire failure",
			tr:         &tensionTransport{netErr: errors.New("network unreachable")},
			wantPrefix: "the proposal could not be read back (",
		},
		{
			name:       "non-2xx",
			tr:         &tensionTransport{status: 404, body: `{"error":"not found"}`},
			wantPrefix: "the read-back was refused (",
		},
		{
			name:       "post-retry 429",
			tr:         &tensionTransport{status: 429, body: `{}`},
			wantPrefix: "the read-back was rate limited (the request budget was exhausted)",
		},
		{
			name:       "undecodable body",
			tr:         &tensionTransport{status: 200, body: `{"data": "not an object`},
			wantPrefix: "the read-back response could not be read",
		},
	}
	seen := map[string]bool{}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			exec := newReadBackExecutor(t, tc.tr)
			p, raw, reason := readBackProposalVerdict(context.Background(), exec, "prp_0123")
			if !strings.HasPrefix(reason, tc.wantPrefix) {
				t.Errorf("reason = %q, want prefix %q", reason, tc.wantPrefix)
			}
			if reason == "" {
				t.Fatal("a failed read-back must return a non-empty reason")
			}
			if p.ID != "" {
				t.Errorf("a failed read-back must return a zero proposal, got %+v", p)
			}
			if raw != nil {
				t.Errorf("a failed read-back must return nil raw bytes, got %s", raw)
			}
			if seen[reason] {
				t.Errorf("two failure families share the reason %q", reason)
			}
			seen[reason] = true
		})
	}
}

// TestReadBackProposalVerdict_ReasonTexts pins the exact reason wording for the
// deterministic families (the wire reason carries a transport-specific cause and
// is prefix-pinned above).
func TestReadBackProposalVerdict_ReasonTexts(t *testing.T) {
	tr := &tensionTransport{status: 429, body: `{}`}
	_, _, reason := readBackProposalVerdict(context.Background(), newReadBackExecutor(t, tr), "prp_0123")
	if reason != "the read-back was rate limited (the request budget was exhausted)" {
		t.Errorf("429 reason = %q", reason)
	}

	tr = &tensionTransport{status: 500, body: `oops`}
	_, _, reason = readBackProposalVerdict(context.Background(), newReadBackExecutor(t, tr), "prp_0123")
	if !strings.HasPrefix(reason, "the read-back was refused (") {
		t.Errorf("non-2xx reason = %q, want the refused shape", reason)
	}

	tr = &tensionTransport{netErr: errors.New("dial tcp: connection refused")}
	_, _, reason = readBackProposalVerdict(context.Background(), newReadBackExecutor(t, tr), "prp_0123")
	if !strings.HasPrefix(reason, "the proposal could not be read back (") {
		t.Errorf("wire reason = %q, want the could-not-read-back shape", reason)
	}
}

// TestReadBackProposalVerdict_EmptyIDShortCircuits pins ADR-6's no-fabricated-path
// rule: an empty id issues ZERO requests and returns the id-undeterminable reason.
func TestReadBackProposalVerdict_EmptyIDShortCircuits(t *testing.T) {
	tr := &tensionTransport{status: 200, body: proposalReadBackBody}
	exec := newReadBackExecutor(t, tr)

	p, raw, reason := readBackProposalVerdict(context.Background(), exec, "")
	if reason != "the created proposal's id could not be determined" {
		t.Errorf("reason = %q, want the id-undeterminable reason", reason)
	}
	if tr.calls != 0 {
		t.Errorf("an empty id must issue zero requests, got %d", tr.calls)
	}
	if p.ID != "" || raw != nil {
		t.Errorf("an empty id must return a zero proposal and nil raw, got %+v / %s", p, raw)
	}
}

// TestReadBackProposalVerdict_PathEscapesID pins that the path is built with
// url.PathEscape, matching `proposal get`: a malformed/adversarial id stays one
// opaque segment and cannot redirect the request or traverse the path.
func TestReadBackProposalVerdict_PathEscapesID(t *testing.T) {
	tr := &tensionTransport{status: 404, body: `{}`}
	exec := newReadBackExecutor(t, tr)

	_, _, reason := readBackProposalVerdict(context.Background(), exec, "prp_1/../secrets")
	if reason == "" {
		t.Fatal("a 404 read-back must return a reason")
	}
	if tr.calls != 1 {
		t.Fatalf("want one request, got %d", tr.calls)
	}
	if strings.Contains(tr.lastPath, "..") && !strings.Contains(tr.lastPath, "%2F") {
		t.Errorf("the id must be escaped to one opaque segment, got path %q", tr.lastPath)
	}
}

// TestReadBackProposalVerdict_ValidJSONWrongShape pins the local decode guard:
// a body that is valid JSON but not a {data: Proposal} document returns the
// undecodable reason and nil raw — the machine path must never emit a document
// the CLI could not read a proposal from.
func TestReadBackProposalVerdict_ValidJSONWrongShape(t *testing.T) {
	tr := &tensionTransport{status: 200, body: `{"data": [1, 2, 3]}`}
	exec := newReadBackExecutor(t, tr)

	_, raw, reason := readBackProposalVerdict(context.Background(), exec, "prp_0123")
	if reason != "the read-back response could not be read" {
		t.Errorf("reason = %q, want the undecodable reason", reason)
	}
	if raw != nil {
		t.Errorf("an unreadable body must return nil raw, got %s", raw)
	}
}

// TestReadBackProposalVerdict_RawIsVerbatim pins that the returned raw bytes are
// exactly the wire bytes — never re-marshalled — by comparing against the
// fixture byte-for-byte (the 018 verbatim contract the machine path relies on).
func TestReadBackProposalVerdict_RawIsVerbatim(t *testing.T) {
	// Key order and whitespace in this body would not survive a re-marshal.
	body := `{"data":{"updated_at":"t","id":"prp_0123",  "valid":true,"created_at":"t"}}`
	tr := &tensionTransport{status: 200, body: body}
	exec := newReadBackExecutor(t, tr)

	_, raw, reason := readBackProposalVerdict(context.Background(), exec, "prp_0123")
	if reason != "" {
		t.Fatalf("unexpected reason %q", reason)
	}
	if !bytes.Equal(raw, json.RawMessage(body)) {
		t.Errorf("raw must be the wire bytes verbatim:\n got: %s\nwant: %s", raw, body)
	}
}
