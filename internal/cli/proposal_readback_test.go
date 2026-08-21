package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/Luscii/cli-glassfrog/internal/apiclient"
	"github.com/Luscii/cli-glassfrog/internal/output"
	"github.com/Luscii/cli-glassfrog/internal/render"
)

// proposalReadBackBody is a representative getProposal 200 body: the same
// {data: Proposal} shape as the create's 201, but carrying the two undeclared
// verdict fields (valid, validation_alerts) the read-back exists to obtain.
//
// NOTE (078): its verdict is `valid: false`, so as a CREATE read-back it is now
// the body that FAILS the command with exit 8 — it is no longer a stand-in for a
// generic successful read-back. Tests that want the success path use
// proposalReadBackValidBody or proposalReadBackValidWithAlertBody below; this one
// is reserved for the helper's own tests (where no create runs) and for pinning
// the failure.
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
// a body that is valid JSON but not a readable {data: Proposal} document for the
// proposal that was asked for returns the undecodable reason and nil raw — the
// machine path must never emit a document the CLI could not read this proposal
// from, and the human path must never render one in place of the created draft.
//
// The last three cases decode CLEANLY into a zero (or foreign) Proposal, so a
// decode-error check alone would wave them through as an answered read-back.
func TestReadBackProposalVerdict_ValidJSONWrongShape(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{name: "data is not an object", body: `{"data": [1, 2, 3]}`},
		{name: "empty document", body: `{}`},
		{name: "empty data object", body: `{"data":{}}`},
		{name: "no data key at all", body: `{"meta":{"pagination":{}}}`},
		{name: "a different proposal", body: `{"data":{"id":"prp_9999","status":"draft","valid":true}}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tr := &tensionTransport{status: 200, body: tc.body}
			exec := newReadBackExecutor(t, tr)

			p, raw, reason := readBackProposalVerdict(context.Background(), exec, "prp_0123")
			if reason != "the read-back response could not be read" {
				t.Errorf("reason = %q, want the undecodable reason", reason)
			}
			if raw != nil {
				t.Errorf("an unreadable body must return nil raw, got %s", raw)
			}
			if p.ID != "" {
				t.Errorf("an unreadable body must return a zero proposal, got %+v", p)
			}
		})
	}
}

// TestRunProposalCreate_EmptyReadBackKeepsCreatedID is the regression for the
// spec's unqualified non-behavior — the created prp_ id is never withheld
// because the read-back failed. A 200 whose body decodes to no proposal is a
// failed read-back: the human body must still render the CREATED proposal (id
// and status intact), not the empty one the read-back returned.
func TestRunProposalCreate_EmptyReadBackKeepsCreatedID(t *testing.T) {
	for _, body := range []string{`{}`, `{"data":{}}`, `{"meta":{"x":1}}`, `{"data":{"id":"prp_9999"}}`} {
		t.Run(body, func(t *testing.T) {
			tr := &proposalSeqTransport{steps: []proposalSeqStep{
				{status: 201, body: proposalCreatedBody},
				{status: 200, body: body},
			}}
			seam := &fakeProposalSeam{fakeMeSeam: &fakeMeSeam{ctx: validMeContext(), transport: tr}}
			outcome, stdout, stderr := runProposalCreateOver(t, seam, proposalCreateConfig{
				tensionID:    "ten_0123",
				changesValue: `[{"type":"CreateRole","name":"Scribe"}]`,
			})
			if outcome != Success {
				t.Fatalf("outcome = %v, want Success\nstderr: %s", outcome, stderr)
			}
			if !strings.Contains(stdout, "prp_0123  [draft]") {
				t.Errorf("the CREATED proposal must render with its id and status:\n%s", stdout)
			}
			if strings.Contains(stdout, "prp_9999") {
				t.Errorf("a foreign proposal must never be reported as the created one:\n%s", stdout)
			}
			if !strings.Contains(stdout, "  Validity:       unavailable — the read-back response could not be read\n") {
				t.Errorf("the verdict must read as unavailable with its reason:\n%s", stdout)
			}
			if !strings.Contains(stderr, "could not read proposal prp_0123 back") {
				t.Errorf("the advisory must name the failed read-back:\n%s", stderr)
			}
		})
	}
}

// TestRunProposalCreate_MachineEmptyReadBackFallsBackToCreate is the machine-arm
// half of the same regression: an unreadable read-back must fall back to the
// create's document, never emit the empty one — which would drop the created
// prp_ id from stdout entirely, the exact loss the spec forbids.
func TestRunProposalCreate_MachineEmptyReadBackFallsBackToCreate(t *testing.T) {
	outcome, stdout, stderr := machineCreateOver(t, "json", []proposalSeqStep{
		{status: 201, body: proposalCreatedBody},
		{status: 200, body: `{"data":{}}`},
	})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success", outcome)
	}
	want, err := output.RenderSuccess(output.JSON, json.RawMessage(proposalCreatedBody))
	if err != nil {
		t.Fatalf("fixture render: %v", err)
	}
	if stdout != string(want) {
		t.Errorf("stdout must fall back to the create's document:\n got: %s\nwant: %s", stdout, want)
	}
	if !strings.Contains(stdout, "prp_0123") {
		t.Errorf("the created id must never be dropped from stdout:\n%s", stdout)
	}
	var advisory struct {
		VerdictSource verdictSource `json:"verdict_source"`
	}
	if err := json.Unmarshal([]byte(stderr), &advisory); err != nil {
		t.Fatalf("advisory decode: %v\nstderr: %s", err, stderr)
	}
	if advisory.VerdictSource.ReadBack {
		t.Error("read_back must be false when the read-back produced no readable proposal")
	}
	if advisory.VerdictSource.Reason != "the read-back response could not be read" {
		t.Errorf("reason = %q", advisory.VerdictSource.Reason)
	}
	if advisory.VerdictSource.ProposalID != "prp_0123" || advisory.VerdictSource.Remedy != "glassfrog proposal get prp_0123" {
		t.Errorf("the advisory must keep the id and name the remedy, got %+v", advisory.VerdictSource)
	}
}

// proposalSeqStep is one scripted exchange for proposalSeqTransport: a canned
// response, or a wire error.
type proposalSeqStep struct {
	status int
	body   string
	netErr error
}

// proposalSeqTransport is the two-exchange fake base transport the 074 create
// tests need: scripted responses per call (the POST's 201, then the read-back
// GET's reply), recording EVERY request's method and path. Calls beyond the
// script repeat the last step, so a retried read-back keeps seeing its scripted
// failure.
type proposalSeqTransport struct {
	calls   int
	methods []string
	paths   []string
	bodies  []string
	steps   []proposalSeqStep
}

func (s *proposalSeqTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	s.calls++
	s.methods = append(s.methods, req.Method)
	s.paths = append(s.paths, req.URL.Path)
	body := ""
	if req.Body != nil {
		b, _ := io.ReadAll(req.Body)
		body = string(b)
	}
	s.bodies = append(s.bodies, body)
	i := s.calls - 1
	if i >= len(s.steps) {
		i = len(s.steps) - 1
	}
	step := s.steps[i]
	if step.netErr != nil {
		return nil, step.netErr
	}
	return &http.Response{
		StatusCode: step.status,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(step.body)),
	}, nil
}

// proposalReadBackValidBody is a read-back 200 whose verdict is favourable: a
// stated valid, a present-and-empty alerts list, and the transitions as read at
// the same instant.
const proposalReadBackValidBody = `{"data":{"id":"prp_0123","type":"proposal","status":"draft","tension_id":"ten_0123","circle_id":"role_0123","proposer_id":"per_0123","changes":[{"id":"chg_1","type":"CreateRole","name":"Scribe"}],"response_summary":{"total":0,"no_objection":0,"bring_to_meeting":0},"expected_response_count":0,"received_response_count":0,"valid":true,"validation_alerts":[],"available_transitions":["propose","withdraw"],"proposed_at":null,"response_deadline":null,"accepted_at":null,"created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"}}`

// proposalReadBackValidWithAlertBody is a favourable read-back that nonetheless
// carries an alert. It is the *valid* state, not a state of its own (078 ADR-4's
// counting note), so it succeeds — which is what makes it the fixture for pinning
// a success-path rendering that includes an alert count, now that the not-valid
// body fails the create.
const proposalReadBackValidWithAlertBody = `{"data":{"id":"prp_0123","type":"proposal","status":"draft","tension_id":"ten_0123","circle_id":"role_0123","proposer_id":"per_0123","changes":[{"id":"chg_1","type":"CreateRole","name":"Scribe"}],"response_summary":{"total":0,"no_objection":0,"bring_to_meeting":0},"expected_response_count":0,"received_response_count":0,"valid":true,"validation_alerts":[{"severity":"warning","path":"name","message":"Advisory only."}],"available_transitions":["propose","withdraw"],"proposed_at":null,"response_deadline":null,"accepted_at":null,"created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"}}`

// TestRunProposalCreate_HumanVerdictStates pins the human full path for all four
// verdict states. Three are successes: each renders its state, exits 0 (the
// read-back's content or absence never changes the outcome for them, 074 ADR-2),
// and writes the stderr advisory derived from the same value. The fourth — an
// explicit `valid: false` — is the invalid-create FAILURE (078): stdout stays
// empty, stderr carries the diagnostic, no advisory is written, and it exits 8.
//
// The loop body branches on wantFailure rather than asserting one shape for every
// row: the success assertions (validity line, verdict-source line, id on stdout,
// advisory on stderr) are all false for the failure, so a shared body could not
// express both — and simply adding an expected-outcome column to a body that
// Fatalfs on `outcome != Success` and then asserts success-shaped output would
// still fail.
func TestRunProposalCreate_HumanVerdictStates(t *testing.T) {
	cases := []struct {
		name         string
		readBack     proposalSeqStep
		wantFailure  bool
		wantValidity string
		wantSource   string
		wantAdvisory string
		wantStderr   []string // the failure's diagnostic fragments
	}{
		{
			name:         "valid",
			readBack:     proposalSeqStep{status: 200, body: proposalReadBackValidBody},
			wantValidity: "  Validity:       valid\n",
			wantSource:   "  Verdict source: read-back of prp_0123 after create\n",
			wantAdvisory: "the validity verdict was read back from proposal prp_0123 after the create",
		},
		{
			name:        "not valid",
			readBack:    proposalSeqStep{status: 200, body: proposalReadBackBody},
			wantFailure: true,
			wantStderr: []string{
				"the server accepted the create but reports proposal prp_0123 not valid (read back after the create)",
				"  error name: Can't update the Cloud Foundations role during this meeting.",
				`review the alerts, check "glassfrog proposal grammar" for documented invalid shapes`,
			},
		},
		{
			name:         "not reported",
			readBack:     proposalSeqStep{status: 200, body: proposalCreatedBody},
			wantValidity: "  Validity:       not reported by the server\n",
			wantSource:   "  Verdict source: read-back of prp_0123 after create\n",
			wantAdvisory: "the validity verdict was read back from proposal prp_0123 after the create",
		},
		{
			name:         "unavailable",
			readBack:     proposalSeqStep{netErr: errors.New("network unreachable")},
			wantValidity: "  Validity:       unavailable — the proposal could not be read back (",
			wantSource:   "  Verdict source: none — the created proposal is reported from the create response\n",
			wantAdvisory: `could not read proposal prp_0123 back to obtain its validity verdict: the proposal could not be read back (`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tr := &proposalSeqTransport{steps: []proposalSeqStep{
				{status: 201, body: proposalCreatedBody},
				tc.readBack,
			}}
			seam := &fakeProposalSeam{fakeMeSeam: &fakeMeSeam{ctx: validMeContext(), transport: tr}}
			outcome, stdout, stderr := runProposalCreateOver(t, seam, proposalCreateConfig{
				tensionID:    "ten_0123",
				changesValue: `[{"type":"CreateRole","name":"Scribe"}]`,
			})

			if tc.wantFailure {
				if outcome != InvalidCreate || ExitCode(outcome) != 8 {
					t.Fatalf("outcome = %v (exit %d), want InvalidCreate/8\nstderr: %s", outcome, ExitCode(outcome), stderr)
				}
				if stdout != "" {
					t.Errorf("a human-format failure leaves stdout empty, got:\n%s", stdout)
				}
				for _, want := range tc.wantStderr {
					if !strings.Contains(stderr, want) {
						t.Errorf("stderr missing %q:\n%s", want, stderr)
					}
				}
				// The success-path advisory is not emitted on the failure (078 ADR-4):
				// kind + proposal_id subsume it.
				if strings.Contains(stderr, "the validity verdict was read back from proposal") {
					t.Errorf("no verdict advisory may accompany the failure:\n%s", stderr)
				}
				return
			}

			if outcome != Success {
				t.Fatalf("outcome = %v, want Success in every success verdict state\nstderr: %s", outcome, stderr)
			}
			if !strings.Contains(stdout, tc.wantValidity) {
				t.Errorf("stdout missing %q:\n%s", tc.wantValidity, stdout)
			}
			if !strings.Contains(stdout, tc.wantSource) {
				t.Errorf("stdout missing %q:\n%s", tc.wantSource, stdout)
			}
			if !strings.Contains(stdout, "prp_0123") {
				t.Errorf("the created id must be reported in every state:\n%s", stdout)
			}
			if !strings.Contains(stderr, tc.wantAdvisory) {
				t.Errorf("stderr advisory missing %q:\n%s", tc.wantAdvisory, stderr)
			}
		})
	}
}

// TestRunProposalCreate_InvalidDraftFullOutput pins the created-but-invalid path
// end to end in the human full format, INVERTED by 078: what used to render as a
// success document with a not-valid verdict line is now the failure. stdout stays
// empty, stderr carries the cause naming the created id, the alert with its
// severity and path, and the remedy — and the command exits 8.
//
// The success-shaped rendering this test used to assert (the Validity line, the
// Alerts block, the Transitions line) is still pinned for the three states that
// remain successes by TestRunProposalCreate_HumanVerdictStates and the
// alert-carrying valid fixture below.
func TestRunProposalCreate_InvalidDraftFullOutput(t *testing.T) {
	tr := &proposalSeqTransport{steps: []proposalSeqStep{
		{status: 201, body: proposalCreatedBody},
		{status: 200, body: proposalReadBackBody},
	}}
	seam := &fakeProposalSeam{fakeMeSeam: &fakeMeSeam{ctx: validMeContext(), transport: tr}}
	outcome, stdout, stderr := runProposalCreateOver(t, seam, proposalCreateConfig{
		tensionID:    "ten_0123",
		changesValue: `[{"type":"CreateRole","name":"Scribe"}]`,
	})
	if outcome != InvalidCreate || ExitCode(outcome) != 8 {
		t.Fatalf("outcome = %v (exit %d), want InvalidCreate/8\nstderr: %s", outcome, ExitCode(outcome), stderr)
	}
	if stdout != "" {
		t.Errorf("a human-format failure leaves stdout empty, got:\n%s", stdout)
	}
	for _, want := range []string{
		"the server accepted the create but reports proposal prp_0123 not valid (read back after the create)",
		"  error name: Can't update the Cloud Foundations role during this meeting.",
		"create a corrected proposal from the same tension",
	} {
		if !strings.Contains(stderr, want) {
			t.Errorf("stderr missing %q:\n%s", want, stderr)
		}
	}
	if strings.Contains(stderr, "the validity verdict was read back from proposal") {
		t.Errorf("no verdict advisory may accompany the failure:\n%s", stderr)
	}
	// Exactly the two exchanges: the failure classification issues no third request.
	if tr.calls != 2 {
		t.Errorf("the invalid path is exactly two exchanges (POST + one read), got %d", tr.calls)
	}
}

// TestRunProposalCreate_CompactCarriesVerdict pins the compact selection: one
// line carrying the id, status, change count, the compact validity label, and the
// alert count. It is driven by a VALID read-back carrying an alert — the valid
// state, which still succeeds (078 ADR-4's counting note) — because the not-valid
// body no longer reaches a success rendering at all. The alert count stays covered.
func TestRunProposalCreate_CompactCarriesVerdict(t *testing.T) {
	tr := &proposalSeqTransport{steps: []proposalSeqStep{
		{status: 201, body: proposalCreatedBody},
		{status: 200, body: proposalReadBackValidWithAlertBody},
	}}
	seam := &fakeProposalSeam{fakeMeSeam: &fakeMeSeam{ctx: validMeContext(), envOutput: "compact", transport: tr}}
	outcome, stdout, _ := runProposalCreateOver(t, seam, proposalCreateConfig{
		tensionID:    "ten_0123",
		changesValue: `[{"type":"CreateRole","name":"Scribe"}]`,
	})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success", outcome)
	}
	want := "prp_0123  [draft]  1 change(s)  0 responses  valid (1 alert)\n"
	if stdout != want {
		t.Errorf("compact line = %q, want %q", stdout, want)
	}
}

// TestRunProposalCreate_CompactFailsTheInvalidDraft pins that the compact format
// fails the invalid create exactly as full does (interface-cli: both human formats
// render the failure identically) — the compact projection is not consulted at all.
func TestRunProposalCreate_CompactFailsTheInvalidDraft(t *testing.T) {
	tr := &proposalSeqTransport{steps: []proposalSeqStep{
		{status: 201, body: proposalCreatedBody},
		{status: 200, body: proposalReadBackBody},
	}}
	seam := &fakeProposalSeam{fakeMeSeam: &fakeMeSeam{ctx: validMeContext(), envOutput: "compact", transport: tr}}
	outcome, stdout, stderr := runProposalCreateOver(t, seam, proposalCreateConfig{
		tensionID:    "ten_0123",
		changesValue: `[{"type":"CreateRole","name":"Scribe"}]`,
	})
	if outcome != InvalidCreate || ExitCode(outcome) != 8 {
		t.Fatalf("outcome = %v (exit %d), want InvalidCreate/8\nstderr: %s", outcome, ExitCode(outcome), stderr)
	}
	if stdout != "" {
		t.Errorf("compact stdout must stay empty on the failure, got:\n%s", stdout)
	}
	if !strings.Contains(stderr, "not valid (read back after the create)") {
		t.Errorf("stderr should carry the same diagnostic the full format renders:\n%s", stderr)
	}
}

// TestRunProposalCreate_PreChangeUserTemplateStillRenders pins ADR-4's promotion
// guarantee at the call site: a user template written against the pre-074 view
// (.Proposal.* field paths) still renders — no path is removed, renamed, or
// reshaped by the verdict's addition.
//
// It does NOT pin the rendered VALUES as unchanged, because they are not: where
// the read-back answered, the human arm renders the READ-BACK's proposal (plan
// § Verdict Assembly — the invalid-draft scenario needs the empty transition set
// only the read-back reports). So the projection here deliberately includes
// .Proposal.AvailableTransitions, the one field on which the two fixtures
// DISAGREE (create: ["propose"]; read-back: ["propose","withdraw"]), and each
// case asserts which document supplied it. A projection the two fixtures agree
// on — id, status, change count — renders identically whichever document is
// substituted, so it cannot fail when the promise breaks. That was this test's
// original defect (validate.md Round 2, F-1).
func TestRunProposalCreate_PreChangeUserTemplateStillRenders(t *testing.T) {
	// Every path below resolved before 074; none is verdict-aware.
	const pre074 = "{{.Proposal.ID}} {{.Proposal.Status}} {{len .Proposal.Changes}} {{len .Proposal.AvailableTransitions}}"

	cases := []struct {
		name     string
		readBack proposalSeqStep
		want     string
		why      string
	}{
		{
			name:     "read-back answered: values come from the read-back",
			readBack: proposalSeqStep{status: 200, body: proposalReadBackValidBody},
			want:     "prp_0123 draft 1 2",
			why:      "the read-back's two transitions, not the create's one",
		},
		{
			name:     "read-back failed: values fall back to the create response",
			readBack: proposalSeqStep{netErr: errors.New("dial tcp: network unreachable")},
			want:     "prp_0123 draft 1 1",
			why:      "the create's one transition — a failed read-back substitutes nothing",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tr := &proposalSeqTransport{steps: []proposalSeqStep{
				{status: 201, body: proposalCreatedBody},
				tc.readBack,
			}}
			seam := &fakeProposalSeam{fakeMeSeam: &fakeMeSeam{
				ctx:       validMeContext(),
				transport: tr,
				tmplFiles: map[string]string{"pre074.tmpl": pre074},
			}}
			outcome, stdout, stderr := runProposalCreateOver(t, seam, proposalCreateConfig{
				tensionID:     "ten_0123",
				changesValue:  `[{"type":"CreateRole","name":"Scribe"}]`,
				outputFlag:    "pre074.tmpl",
				outputPresent: true,
			})
			if outcome != Success {
				t.Fatalf("outcome = %v, want Success\nstderr: %s", outcome, stderr)
			}
			if stdout != tc.want {
				t.Errorf("pre-change template output = %q, want %q (%s)", stdout, tc.want, tc.why)
			}
		})
	}
}

// TestRunProposalCreate_HumanRenderFailureLeavesStdoutEmpty pins that a render
// failure on the new key still maps to the internal-error code with stdout left
// empty (buffer-then-write) — and no advisory is written for output that never
// appeared.
func TestRunProposalCreate_HumanRenderFailureLeavesStdoutEmpty(t *testing.T) {
	orig := renderFn
	defer func() { renderFn = orig }()
	renderFn = func(render.Resource, render.Format, any) (string, error) {
		return "", &render.RenderError{Resource: render.ResourceProposalCreated, Format: render.FormatFull, Err: errors.New("boom")}
	}

	tr := &proposalSeqTransport{steps: []proposalSeqStep{
		{status: 201, body: proposalCreatedBody},
		{status: 200, body: proposalReadBackValidBody},
	}}
	seam := &fakeProposalSeam{fakeMeSeam: &fakeMeSeam{ctx: validMeContext(), transport: tr}}
	outcome, stdout, _ := runProposalCreateOver(t, seam, proposalCreateConfig{
		tensionID:    "ten_0123",
		changesValue: `[{"type":"CreateRole","name":"Scribe"}]`,
	})
	if outcome != RuntimeError {
		t.Fatalf("outcome = %v, want RuntimeError", outcome)
	}
	if stdout != "" {
		t.Errorf("a render failure must leave stdout empty, got %q", stdout)
	}
}

// machineCreateOver drives a machine-format create over the scripted transport
// and returns (outcome, stdout, stderr).
func machineCreateOver(t *testing.T, format string, steps []proposalSeqStep) (Outcome, string, string) {
	t.Helper()
	tr := &proposalSeqTransport{steps: steps}
	seam := &fakeProposalSeam{fakeMeSeam: &fakeMeSeam{ctx: validMeContext(), envOutput: format, transport: tr}}
	outcome, stdout, stderr := runProposalCreateOver(t, seam, proposalCreateConfig{
		tensionID:    "ten_0123",
		changesValue: `[{"type":"CreateRole","name":"Scribe"}]`,
	})
	return outcome, stdout, stderr
}

// TestRunProposalCreate_MachineEmitsReadBackVerbatim pins ADR-5's stdout
// contract: on a successful read-back the emitted document is the READ-BACK's,
// carrying valid, validation_alerts, and available_transitions as the server
// sent them — and the bytes are the fixture's own, never re-marshalled (the
// document equals what the shared structured renderer produces from the
// fixture's exact bytes, key order preserved).
// It is driven by a valid read-back: the verbatim-emission contract governs the
// SUCCESS outcomes after 078 (its announced narrowing of 074 ADR-5), and the
// not-valid body now renders the failure envelope instead of any document.
func TestRunProposalCreate_MachineEmitsReadBackVerbatim(t *testing.T) {
	outcome, stdout, stderr := machineCreateOver(t, "json", []proposalSeqStep{
		{status: 201, body: proposalCreatedBody},
		{status: 200, body: proposalReadBackValidBody},
	})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success", outcome)
	}
	want, err := output.RenderSuccess(output.JSON, json.RawMessage(proposalReadBackValidBody))
	if err != nil {
		t.Fatalf("fixture render: %v", err)
	}
	if stdout != string(want) {
		t.Errorf("stdout must be the read-back's document from its exact bytes:\n got: %s\nwant: %s", stdout, want)
	}
	var advisory struct {
		VerdictSource struct {
			ReadBack   bool    `json:"read_back"`
			ProposalID string  `json:"proposal_id"`
			Reason     *string `json:"reason"`
			Remedy     *string `json:"remedy"`
		} `json:"verdict_source"`
	}
	if err := json.Unmarshal([]byte(stderr), &advisory); err != nil {
		t.Fatalf("the machine advisory must be a JSON document, got %q: %v", stderr, err)
	}
	vs := advisory.VerdictSource
	if !vs.ReadBack || vs.ProposalID != "prp_0123" {
		t.Errorf("advisory = %+v, want read_back true with the proposal id", vs)
	}
	if vs.Reason != nil || vs.Remedy != nil {
		t.Errorf("reason/remedy must be ABSENT when the read-back answered, got %+v", vs)
	}
}

// TestRunProposalCreate_MachineFourStateTable pins that the four verdict states
// are distinguishable in a machine format without prose. The three SUCCESS states
// are told apart by stdout's data.valid plus stderr's verdict_source.read_back, as
// 074 landed them. The fourth — an explicit `valid: false` — is now the
// invalid-create failure (078): stdout carries the failure envelope instead of a
// server document, and no advisory is written, so it is told apart by exit 8 and
// its envelope's kind. That is a stronger distinction than the one it replaced,
// not a weaker one.
//
// The loop body branches on wantFailure because the failure's stdout is not a
// {data: …} document at all: decoding it as one and looking for data.valid is
// meaningless, so the shared assertions cannot cover both shapes.
func TestRunProposalCreate_MachineFourStateTable(t *testing.T) {
	type parsed struct {
		validPresent bool
		validValue   bool
		readBack     bool
		hasReason    bool
	}
	cases := []struct {
		name        string
		readBack    proposalSeqStep
		wantFailure bool
		want        parsed
	}{
		{name: "valid", readBack: proposalSeqStep{status: 200, body: proposalReadBackValidBody},
			want: parsed{validPresent: true, validValue: true, readBack: true}},
		{name: "not valid", readBack: proposalSeqStep{status: 200, body: proposalReadBackBody},
			wantFailure: true},
		{name: "not reported", readBack: proposalSeqStep{status: 200, body: proposalCreatedBody},
			want: parsed{readBack: true}},
		{name: "unavailable", readBack: proposalSeqStep{netErr: errors.New("network unreachable")},
			want: parsed{hasReason: true}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			outcome, stdout, stderr := machineCreateOver(t, "json", []proposalSeqStep{
				{status: 201, body: proposalCreatedBody},
				tc.readBack,
			})

			if tc.wantFailure {
				if outcome != InvalidCreate || ExitCode(outcome) != 8 {
					t.Fatalf("outcome = %v (exit %d), want InvalidCreate/8\nstderr: %s", outcome, ExitCode(outcome), stderr)
				}
				var env struct {
					Error struct {
						Kind       string            `json:"kind"`
						ProposalID string            `json:"proposal_id"`
						Alerts     []json.RawMessage `json:"validation_alerts"`
					} `json:"error"`
				}
				if err := json.Unmarshal([]byte(stdout), &env); err != nil {
					t.Fatalf("stdout must be the failure envelope: %v\n%s", err, stdout)
				}
				if env.Error.Kind != "invalid-create" || env.Error.ProposalID != "prp_0123" || len(env.Error.Alerts) != 1 {
					t.Errorf("envelope = %+v, want kind invalid-create with the id and one alert", env.Error)
				}
				// No server document rides stdout beside the envelope, and no
				// success-path advisory rides stderr (078 ADR-4).
				if strings.Contains(stdout, `"data"`) {
					t.Errorf("no server proposal document may be emitted on the failure:\n%s", stdout)
				}
				if strings.Contains(stderr, "verdict_source") {
					t.Errorf("no verdict advisory may accompany the failure:\n%s", stderr)
				}
				return
			}

			if outcome != Success {
				t.Fatalf("outcome = %v, want Success in every success verdict state", outcome)
			}
			var doc struct {
				Data map[string]json.RawMessage `json:"data"`
			}
			if err := json.Unmarshal([]byte(stdout), &doc); err != nil {
				t.Fatalf("stdout must be one server document: %v", err)
			}
			validRaw, validPresent := doc.Data["valid"]
			if validPresent != tc.want.validPresent {
				t.Errorf("data.valid present = %v, want %v", validPresent, tc.want.validPresent)
			}
			if validPresent {
				var v bool
				if err := json.Unmarshal(validRaw, &v); err != nil || v != tc.want.validValue {
					t.Errorf("data.valid = %s, want %v", validRaw, tc.want.validValue)
				}
			}
			var advisory struct {
				VerdictSource struct {
					ReadBack bool    `json:"read_back"`
					Reason   *string `json:"reason"`
				} `json:"verdict_source"`
			}
			if err := json.Unmarshal([]byte(stderr), &advisory); err != nil {
				t.Fatalf("stderr advisory must be structured in a machine format: %v", err)
			}
			if advisory.VerdictSource.ReadBack != tc.want.readBack {
				t.Errorf("verdict_source.read_back = %v, want %v", advisory.VerdictSource.ReadBack, tc.want.readBack)
			}
			if (advisory.VerdictSource.Reason != nil) != tc.want.hasReason {
				t.Errorf("verdict_source.reason present = %v, want %v", advisory.VerdictSource.Reason != nil, tc.want.hasReason)
			}
		})
	}
}

// TestRunProposalCreate_MachineFailedReadBackFallsBack pins the fallback: on a
// failed read-back the emitted document is the CREATE's, unchanged from today,
// and the structured advisory states the verdict was unobtainable, why, and the
// remedy.
func TestRunProposalCreate_MachineFailedReadBackFallsBack(t *testing.T) {
	outcome, stdout, stderr := machineCreateOver(t, "json", []proposalSeqStep{
		{status: 201, body: proposalCreatedBody},
		{status: 429, body: `{}`},
	})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success — a rate-limited read-back is never the command's failure", outcome)
	}
	want, err := output.RenderSuccess(output.JSON, json.RawMessage(proposalCreatedBody))
	if err != nil {
		t.Fatalf("fixture render: %v", err)
	}
	if stdout != string(want) {
		t.Errorf("stdout must fall back to the create's document:\n got: %s\nwant: %s", stdout, want)
	}
	// 017's retry notices precede the advisory on stderr (a 429 GET is retried);
	// the advisory is the structured document that follows them.
	docStart := strings.Index(stderr, "{")
	if docStart < 0 {
		t.Fatalf("no structured advisory on stderr:\n%s", stderr)
	}
	var advisory struct {
		VerdictSource verdictSource `json:"verdict_source"`
	}
	if err := json.Unmarshal([]byte(stderr[docStart:]), &advisory); err != nil {
		t.Fatalf("advisory decode: %v\nstderr: %s", err, stderr)
	}
	vs := advisory.VerdictSource
	if vs.ReadBack {
		t.Error("read_back must be false when the read-back failed")
	}
	if vs.ProposalID != "prp_0123" {
		t.Errorf("proposal_id = %q, want prp_0123", vs.ProposalID)
	}
	if vs.Reason != "the read-back was rate limited (the request budget was exhausted)" {
		t.Errorf("reason = %q", vs.Reason)
	}
	if vs.Remedy != "glassfrog proposal get prp_0123" {
		t.Errorf("remedy = %q", vs.Remedy)
	}
}

// TestRunProposalCreate_MachineNoIDNoReadBack pins ADR-6's unliftable-id twin: a
// 2xx create body carrying no prp_ id emits the create's document, issues NO
// read-back request, and reports the id-undeterminable reason with no
// proposal_id and no remedy key (absent means not applicable, never "").
func TestRunProposalCreate_MachineNoIDNoReadBack(t *testing.T) {
	noIDBody := `{"data":{"type":"proposal","status":"draft"}}`
	tr := &proposalSeqTransport{steps: []proposalSeqStep{{status: 201, body: noIDBody}}}
	seam := &fakeProposalSeam{fakeMeSeam: &fakeMeSeam{ctx: validMeContext(), envOutput: "json", transport: tr}}
	outcome, stdout, stderr := runProposalCreateOver(t, seam, proposalCreateConfig{
		tensionID:    "ten_0123",
		changesValue: `[{"type":"CreateRole","name":"Scribe"}]`,
	})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success", outcome)
	}
	if tr.calls != 1 {
		t.Errorf("an id-less create body must issue no read-back, got %d calls", tr.calls)
	}
	want, _ := output.RenderSuccess(output.JSON, json.RawMessage(noIDBody))
	if stdout != string(want) {
		t.Errorf("stdout must be the create's document:\n got: %s\nwant: %s", stdout, want)
	}
	var advisory map[string]map[string]json.RawMessage
	if err := json.Unmarshal([]byte(stderr), &advisory); err != nil {
		t.Fatalf("advisory decode: %v", err)
	}
	vs := advisory["verdict_source"]
	if string(vs["read_back"]) != "false" {
		t.Errorf("read_back = %s, want false", vs["read_back"])
	}
	var reason string
	if err := json.Unmarshal(vs["reason"], &reason); err != nil || reason != "the created proposal's id could not be determined" {
		t.Errorf("reason = %s", vs["reason"])
	}
	for _, absent := range []string{"proposal_id", "remedy"} {
		if _, present := vs[absent]; present {
			t.Errorf("%s must be ABSENT when no id could be determined, got %s", absent, vs[absent])
		}
	}
}

// TestRunProposalCreate_MachineYAML pins the yaml selection: the emitted stdout
// document and the stderr advisory are both rendered in the selected format. It
// is driven by a valid read-back — the advisory is a success-path artifact and is
// not written on the invalid-create failure (078 ADR-4).
func TestRunProposalCreate_MachineYAML(t *testing.T) {
	outcome, stdout, stderr := machineCreateOver(t, "yaml", []proposalSeqStep{
		{status: 201, body: proposalCreatedBody},
		{status: 200, body: proposalReadBackValidBody},
	})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success", outcome)
	}
	want, err := output.RenderSuccess(output.YAML, json.RawMessage(proposalReadBackValidBody))
	if err != nil {
		t.Fatalf("fixture render: %v", err)
	}
	if stdout != string(want) {
		t.Errorf("yaml stdout must be the read-back's document:\n got: %s\nwant: %s", stdout, want)
	}
	for _, wantLine := range []string{"read_back: true", "proposal_id: prp_0123"} {
		if !strings.Contains(stderr, wantLine) {
			t.Errorf("yaml advisory missing %q:\n%s", wantLine, stderr)
		}
	}
}

// TestRunProposalCreate_MachineUndecodableCreateBody pins that an undecodable
// create body still classifies as an API error for the CREATE (031's decode
// classification, unchanged) — and no read-back is attempted for it.
func TestRunProposalCreate_MachineUndecodableCreateBody(t *testing.T) {
	tr := &proposalSeqTransport{steps: []proposalSeqStep{{status: 201, body: `{"data": not-json`}}}
	seam := &fakeProposalSeam{fakeMeSeam: &fakeMeSeam{ctx: validMeContext(), envOutput: "json", transport: tr}}
	outcome, _, _ := runProposalCreateOver(t, seam, proposalCreateConfig{
		tensionID:    "ten_0123",
		changesValue: `[{"type":"CreateRole","name":"Scribe"}]`,
	})
	if outcome != APIError {
		t.Fatalf("outcome = %v, want APIError for an undecodable create body", outcome)
	}
	if tr.calls != 1 {
		t.Errorf("no read-back may follow an undecodable create body, got %d calls", tr.calls)
	}
}

// TestVerdictSource_ProseLine pins the three advisory lines (interface-cli §
// stderr) and that they derive from the same value the structured form encodes.
func TestVerdictSource_ProseLine(t *testing.T) {
	obtained := newVerdictSource("prp_0123", "")
	if got, want := obtained.proseLine(), "the validity verdict was read back from proposal prp_0123 after the create"; got != want {
		t.Errorf("obtained prose = %q, want %q", got, want)
	}
	if !obtained.ReadBack || obtained.ProposalID != "prp_0123" || obtained.Reason != "" || obtained.Remedy != "" {
		t.Errorf("obtained structured form = %+v", obtained)
	}

	failed := newVerdictSource("prp_0123", "the read-back was rate limited (the request budget was exhausted)")
	wantFailed := `could not read proposal prp_0123 back to obtain its validity verdict: the read-back was rate limited (the request budget was exhausted); the proposal was created — run "glassfrog proposal get prp_0123" to read its verdict`
	if got := failed.proseLine(); got != wantFailed {
		t.Errorf("failed prose = %q, want %q", got, wantFailed)
	}
	if failed.ReadBack || failed.Remedy != "glassfrog proposal get prp_0123" {
		t.Errorf("failed structured form = %+v", failed)
	}

	noID := newVerdictSource("", "the created proposal's id could not be determined")
	wantNoID := "could not determine the created proposal's id from the create response, so no validity verdict was obtained; the create response is reported above"
	if got := noID.proseLine(); got != wantNoID {
		t.Errorf("no-id prose = %q, want %q", got, wantNoID)
	}
	if noID.ReadBack || noID.ProposalID != "" || noID.Remedy != "" {
		t.Errorf("the no-id advisory must name no proposal and no remedy, got %+v", noID)
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
