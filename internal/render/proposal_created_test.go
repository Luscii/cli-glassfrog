package render

import (
	"strings"
	"testing"

	"github.com/Luscii/cli-glassfrog/internal/glassfrog"
)

func boolPtr(b bool) *bool { return &b }

// TestNewProposalVerdict_FourStates_FullLabels pins the `full` block's Validity
// label for each of the four verdict states: a non-nil true, a non-nil false, a
// nil flag, and a non-empty unavailable reason. The nil case maps to its own
// label — never to valid and never to not valid (ADR-3).
func TestNewProposalVerdict_FourStates_FullLabels(t *testing.T) {
	cases := []struct {
		name   string
		valid  *bool
		reason string
		want   string
	}{
		{name: "valid", valid: boolPtr(true), want: "valid"},
		{name: "not valid", valid: boolPtr(false), want: "not valid"},
		{name: "not reported", valid: nil, want: "not reported by the server"},
		{name: "unavailable", valid: nil, reason: "the proposal could not be read back (network unreachable)", want: "unavailable — the proposal could not be read back (network unreachable)"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			v := NewProposalVerdict(tc.valid, nil, tc.reason, "prp_1")
			if v.Validity != tc.want {
				t.Errorf("Validity = %q, want %q", v.Validity, tc.want)
			}
		})
	}
}

// TestNewProposalVerdict_FourStates_CompactLabels pins the compact vocabulary for
// the same four states, produced by the SAME call — the two vocabularies have one
// source and cannot drift.
func TestNewProposalVerdict_FourStates_CompactLabels(t *testing.T) {
	cases := []struct {
		name   string
		valid  *bool
		reason string
		want   string
	}{
		{name: "valid", valid: boolPtr(true), want: "valid"},
		{name: "not valid", valid: boolPtr(false), want: "not valid"},
		{name: "not reported", valid: nil, want: "validity not reported"},
		{name: "unavailable", valid: nil, reason: "any reason", want: "validity unavailable"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			v := NewProposalVerdict(tc.valid, nil, tc.reason, "prp_1")
			if v.Compact != tc.want {
				t.Errorf("Compact = %q, want %q", v.Compact, tc.want)
			}
		})
	}
}

// TestNewProposalVerdict_CompactAlertCount pins that the compact label appends the
// alert count whenever the server stated at least one alert, in EITHER validity
// state — a favourable verdict carrying an advisory alert stays visible — while
// the full Validity label never carries a count (the full block renders the
// alerts themselves).
func TestNewProposalVerdict_CompactAlertCount(t *testing.T) {
	one := []glassfrog.ValidationAlert{{Severity: "warning", Path: "name", Message: "advisory"}}
	two := append([]glassfrog.ValidationAlert{{Severity: "error", Path: "x", Message: "m"}}, one...)

	cases := []struct {
		name        string
		valid       *bool
		alerts      []glassfrog.ValidationAlert
		wantCompact string
	}{
		{name: "not valid with one alert", valid: boolPtr(false), alerts: one, wantCompact: "not valid (1 alert)"},
		{name: "valid with one alert", valid: boolPtr(true), alerts: one, wantCompact: "valid (1 alert)"},
		{name: "not valid with two alerts", valid: boolPtr(false), alerts: two, wantCompact: "not valid (2 alerts)"},
		{name: "valid with none", valid: boolPtr(true), alerts: nil, wantCompact: "valid"},
		{name: "valid with present-and-empty", valid: boolPtr(true), alerts: []glassfrog.ValidationAlert{}, wantCompact: "valid"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			v := NewProposalVerdict(tc.valid, tc.alerts, "", "prp_1")
			if v.Compact != tc.wantCompact {
				t.Errorf("Compact = %q, want %q", v.Compact, tc.wantCompact)
			}
			if strings.Contains(v.Validity, "alert") {
				t.Errorf("full Validity label must never carry an alert count, got %q", v.Validity)
			}
		})
	}
}

// TestNewProposalVerdict_ReasonWins pins that a non-empty unavailable reason wins
// over any flag value and carries no alerts — nothing is claimed that the server
// did not state, so neither label ever shows an alert count in that state.
func TestNewProposalVerdict_ReasonWins(t *testing.T) {
	alerts := []glassfrog.ValidationAlert{{Severity: "error", Path: "p", Message: "m"}}
	v := NewProposalVerdict(boolPtr(true), alerts, "the read-back was refused (server error)", "prp_1")
	if v.Validity != "unavailable — the read-back was refused (server error)" {
		t.Errorf("reason must win over the flag, got Validity %q", v.Validity)
	}
	if v.Compact != "validity unavailable" {
		t.Errorf("compact must show unavailable with no alert count, got %q", v.Compact)
	}
	if v.Alerts != nil {
		t.Errorf("an unavailable verdict must carry no alerts, got %v", v.Alerts)
	}
}

// TestNewProposalVerdict_Source pins the provenance value: the read-back and the
// proposal id when a verdict was obtained, and an explicit statement that none was
// obtained otherwise.
func TestNewProposalVerdict_Source(t *testing.T) {
	obtained := NewProposalVerdict(boolPtr(true), nil, "", "prp_0123")
	if obtained.Source != "read-back of prp_0123 after create" {
		t.Errorf("obtained Source = %q", obtained.Source)
	}
	unavailable := NewProposalVerdict(nil, nil, "some reason", "prp_0123")
	if unavailable.Source != "none — the created proposal is reported from the create response" {
		t.Errorf("unavailable Source = %q", unavailable.Source)
	}
}

// createdProposalBody is the shared full body the delegated proposal.full.tmpl
// renders for fullProposal() — the byte-for-byte prefix every proposal-created
// full rendering must start with (one source for the body, ADR-4).
const createdProposalBody = "prp_1  [draft]\n" +
	"  Tension:        ten_0123\n" +
	"  Circle:         role_0123\n" +
	"  Proposer:       per_0123\n" +
	"  Proposed:       2026-01-03T00:00:00Z\n" +
	"  Deadline:       2026-01-10T00:00:00Z\n" +
	"  Accepted:       (none)\n" +
	"  Responses:      3 total — 2 no-objection, 1 bring-to-meeting\n" +
	"  Expected/recv:  3 / 2\n" +
	"  Transitions:    propose, withdraw\n" +
	"  Changes (2):\n" +
	"    - [CreateRole] {\"name\":\"Scribe\"}\n" +
	"    - [UpdateRole]\n"

// TestRender_ProposalCreatedFull_ValidGolden pins the full projection of a valid
// verdict: the shared body (rendered by the invoked proposal.full.tmpl, not
// restated), the Validity line aligned to the existing 16-column label field, no
// Alerts block (none stated), and the Verdict source line.
func TestRender_ProposalCreatedFull_ValidGolden(t *testing.T) {
	verdict := NewProposalVerdict(boolPtr(true), nil, "", "prp_1")
	view := ProposalCreatedView{ProposalView: ProposalView{Proposal: fullProposal()}, Verdict: verdict}
	want := createdProposalBody +
		"  Validity:       valid\n" +
		"  Verdict source: read-back of prp_1 after create\n"
	assertRender(t, ResourceProposalCreated, FormatFull, view, want)
}

// TestRender_ProposalCreatedFull_NotValidWithAlertGolden pins the invalid case:
// the Alerts block appears only when the server stated at least one alert, each
// line carrying its severity, path, and the server's message verbatim.
func TestRender_ProposalCreatedFull_NotValidWithAlertGolden(t *testing.T) {
	alerts := []glassfrog.ValidationAlert{{Severity: "error", Path: "name", Message: "Can't update the Cloud Foundations role during this meeting."}}
	verdict := NewProposalVerdict(boolPtr(false), alerts, "", "prp_1")
	view := ProposalCreatedView{ProposalView: ProposalView{Proposal: fullProposal()}, Verdict: verdict}
	want := createdProposalBody +
		"  Validity:       not valid\n" +
		"  Alerts (1):\n" +
		"    - [error] name: Can't update the Cloud Foundations role during this meeting.\n" +
		"  Verdict source: read-back of prp_1 after create\n"
	assertRender(t, ResourceProposalCreated, FormatFull, view, want)
}

// TestRender_ProposalCreatedFull_ValidWithAlertRendersBothFacts pins that a
// VALID verdict carrying an alert renders both facts — the validity as `valid`
// and the alert with its severity, path, and message — so alert presence never
// reads as an unfavourable verdict.
func TestRender_ProposalCreatedFull_ValidWithAlertRendersBothFacts(t *testing.T) {
	alerts := []glassfrog.ValidationAlert{{Severity: "warning", Path: "changes[0]", Message: "advisory only"}}
	verdict := NewProposalVerdict(boolPtr(true), alerts, "", "prp_1")
	view := ProposalCreatedView{ProposalView: ProposalView{Proposal: fullProposal()}, Verdict: verdict}
	want := createdProposalBody +
		"  Validity:       valid\n" +
		"  Alerts (1):\n" +
		"    - [warning] changes[0]: advisory only\n" +
		"  Verdict source: read-back of prp_1 after create\n"
	assertRender(t, ResourceProposalCreated, FormatFull, view, want)
}

// TestRender_ProposalCreatedFull_FourStatesDistinct pins that all four verdict
// states render distinct Validity lines in the full format — the unavailable
// state carrying its reason and the no-verdict source statement.
func TestRender_ProposalCreatedFull_FourStatesDistinct(t *testing.T) {
	states := map[string]ProposalVerdict{
		"  Validity:       valid\n":                      NewProposalVerdict(boolPtr(true), nil, "", "prp_1"),
		"  Validity:       not valid\n":                  NewProposalVerdict(boolPtr(false), nil, "", "prp_1"),
		"  Validity:       not reported by the server\n": NewProposalVerdict(nil, nil, "", "prp_1"),
		"  Validity:       unavailable — the proposal could not be read back (network unreachable)\n": NewProposalVerdict(nil, nil, "the proposal could not be read back (network unreachable)", "prp_1"),
	}
	seen := map[string]bool{}
	for wantLine, verdict := range states {
		view := ProposalCreatedView{ProposalView: ProposalView{Proposal: fullProposal()}, Verdict: verdict}
		out, err := Render(ResourceProposalCreated, FormatFull, view)
		if err != nil {
			t.Fatalf("render: %v", err)
		}
		if !strings.Contains(out, wantLine) {
			t.Errorf("full output missing %q:\n%s", wantLine, out)
		}
		if seen[out] {
			t.Errorf("two verdict states rendered identically:\n%s", out)
		}
		seen[out] = true
	}
	unavailable := NewProposalVerdict(nil, nil, "some reason", "prp_1")
	view := ProposalCreatedView{ProposalView: ProposalView{Proposal: fullProposal()}, Verdict: unavailable}
	out, err := Render(ResourceProposalCreated, FormatFull, view)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if !strings.Contains(out, "  Verdict source: none — the created proposal is reported from the create response\n") {
		t.Errorf("unavailable state must state no verdict was obtained:\n%s", out)
	}
}

// TestRender_ProposalCreatedCompact_Golden pins the compact one-liner: the shared
// compact line (via the invoked proposal.compact.tmpl, trailing newline trimmed)
// plus the compact verdict label — never the full block's Validity, and never any
// transitions (absent from compact, as today).
func TestRender_ProposalCreatedCompact_Golden(t *testing.T) {
	alerts := []glassfrog.ValidationAlert{{Severity: "error", Path: "name", Message: "refused"}}
	cases := []struct {
		name    string
		verdict ProposalVerdict
		want    string
	}{
		{name: "valid", verdict: NewProposalVerdict(boolPtr(true), nil, "", "prp_1"),
			want: "prp_1  [draft]  2 change(s)  3 responses  valid\n"},
		{name: "not valid with alert", verdict: NewProposalVerdict(boolPtr(false), alerts, "", "prp_1"),
			want: "prp_1  [draft]  2 change(s)  3 responses  not valid (1 alert)\n"},
		{name: "valid with alert", verdict: NewProposalVerdict(boolPtr(true), alerts, "", "prp_1"),
			want: "prp_1  [draft]  2 change(s)  3 responses  valid (1 alert)\n"},
		{name: "not reported", verdict: NewProposalVerdict(nil, nil, "", "prp_1"),
			want: "prp_1  [draft]  2 change(s)  3 responses  validity not reported\n"},
		{name: "unavailable", verdict: NewProposalVerdict(nil, nil, "network unreachable", "prp_1"),
			want: "prp_1  [draft]  2 change(s)  3 responses  validity unavailable\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			view := ProposalCreatedView{ProposalView: ProposalView{Proposal: fullProposal()}, Verdict: tc.verdict}
			assertRender(t, ResourceProposalCreated, FormatCompact, view, tc.want)
		})
	}
}

// TestRender_ProposalKeyed_UnchangedByVerdictFields is the guard against verdict
// lines leaking into `proposal get`, `propose`, and `withdraw` (ADR-4's drift
// surface): a proposal whose model DOES carry a verdict (the read endpoint
// returns the fields) still renders byte-identically to today under the shared
// `proposal` key — no Validity, Alerts, or Verdict source line in either format.
func TestRender_ProposalKeyed_UnchangedByVerdictFields(t *testing.T) {
	p := fullProposal()
	p.Valid = boolPtr(false)
	p.ValidationAlerts = []glassfrog.ValidationAlert{{Severity: "error", Path: "name", Message: "refused"}}

	wantFull := createdProposalBody
	assertRender(t, ResourceProposal, FormatFull, ProposalView{Proposal: p}, wantFull)

	wantCompact := "prp_1  [draft]  2 change(s)  3 responses\n"
	assertRender(t, ResourceProposal, FormatCompact, ProposalView{Proposal: p}, wantCompact)
}

// TestProposalCreatedView_FieldPromotion pins that the embedded ProposalView's
// field paths still resolve on the created view — the promotion a pre-074 user
// template (035) depends on.
func TestProposalCreatedView_FieldPromotion(t *testing.T) {
	view := ProposalCreatedView{ProposalView: ProposalView{Proposal: fullProposal()}}
	if view.Proposal.ID != "prp_1" {
		t.Errorf(".Proposal.ID must resolve through the embedded ProposalView, got %q", view.Proposal.ID)
	}
	if view.Proposal.Status != "draft" {
		t.Errorf(".Proposal.Status must resolve through the embedded ProposalView, got %q", view.Proposal.Status)
	}
}
