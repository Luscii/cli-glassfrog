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
