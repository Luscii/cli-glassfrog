package render

import (
	"testing"

	"github.com/Luscii/cli-glassfrog/internal/glassfrog"
)

// fullProposal is a representative created proposal with every field populated, so
// the full golden pins each label and value.
func fullProposal() glassfrog.Proposal {
	return glassfrog.Proposal{
		ID:         "prp_1",
		Type:       "proposal",
		Status:     "draft",
		TensionID:  "ten_0123",
		CircleID:   "role_0123",
		ProposerID: "per_0123",
		Changes: []glassfrog.ProposalChange{
			{ID: "chg_1", Type: "CreateRole"},
			{ID: "chg_2", Type: "UpdateRole"},
		},
		ResponseSummary:      glassfrog.ResponseSummary{Total: 3, NoObjection: 2, BringToMeeting: 1},
		AvailableTransitions: []string{"propose", "withdraw"},
		ResponseDeadline:     "2026-06-22T12:00:00Z",
		CreatedAt:            "2026-01-01T00:00:00Z",
		UpdatedAt:            "2026-01-02T00:00:00Z",
	}
}

// TestRender_ProposalFull_Golden pins the full projection: the prp_ id + status
// header, the anchor/circle/proposer lines, the change count, the aggregate response
// counts, the available transitions, and the timestamps.
func TestRender_ProposalFull_Golden(t *testing.T) {
	want := "prp_1  [draft]\n" +
		"  Tension:        ten_0123\n" +
		"  Circle:         role_0123\n" +
		"  Proposer:       per_0123\n" +
		"  Changes:        2\n" +
		"  Responses:      2/3 no-objection, 1 bring-to-meeting\n" +
		"  Transitions:    propose, withdraw\n" +
		"  Deadline:       2026-06-22T12:00:00Z\n" +
		"  Created:        2026-01-01T00:00:00Z\n" +
		"  Updated:        2026-01-02T00:00:00Z\n"
	assertRender(t, ResourceProposal, FormatFull, ProposalView{Proposal: fullProposal()}, want)
}

// TestRender_ProposalFull_NullFieldsShowMarkers pins that every nullable field
// (tension_id, circle_id, proposer_id) renders its explicit-absence marker (none) —
// never <no value> or a blank — and absent timestamps render (unknown). A fresh draft
// with no changes still renders a zero count and zero response tallies.
func TestRender_ProposalFull_NullFieldsShowMarkers(t *testing.T) {
	p := glassfrog.Proposal{ID: "prp_1", Status: "draft"}
	want := "prp_1  [draft]\n" +
		"  Tension:        (none)\n" +
		"  Circle:         (none)\n" +
		"  Proposer:       (none)\n" +
		"  Changes:        0\n" +
		"  Responses:      0/0 no-objection, 0 bring-to-meeting\n" +
		"  Transitions:    (none)\n" +
		"  Deadline:       (none)\n" +
		"  Created:        (unknown)\n" +
		"  Updated:        (unknown)\n"
	assertRender(t, ResourceProposal, FormatFull, ProposalView{Proposal: p}, want)
}

// TestRender_ProposalCompact_Golden pins the one-line compact form: the prp_ id (the
// load-bearing handle), the status badge, and the change count.
func TestRender_ProposalCompact_Golden(t *testing.T) {
	want := "prp_1  [draft]  2 change(s)\n"
	assertRender(t, ResourceProposal, FormatCompact, ProposalView{Proposal: fullProposal()}, want)
}
