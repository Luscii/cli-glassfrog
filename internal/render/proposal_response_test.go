package render

import (
	"testing"

	"github.com/Luscii/cli-glassfrog/internal/glassfrog"
)

// fullProposalVote is a representative recorded response with every field populated, so
// the full golden pins each label and value.
func fullProposalVote() glassfrog.ProposalVote {
	return glassfrog.ProposalVote{
		ID:             "prr_0123",
		Type:           "proposal_response",
		ProposalID:     "prp_0123",
		ProposalStatus: "proposed_outside_meeting",
		Value:          "no_objection",
		CreatedAt:      "2026-01-01T00:00:00Z",
		UpdatedAt:      "2026-01-02T00:00:00Z",
	}
}

// TestRender_ProposalResponseFull_Golden pins the full recorded-response projection: the
// prr_ id + recorded marker header, the recorded value, the anchoring proposal_id, the
// parent proposal_status (the auto-acceptance signal), and the recorded timestamp.
func TestRender_ProposalResponseFull_Golden(t *testing.T) {
	want := "prr_0123  recorded\n" +
		"  Response:        no_objection\n" +
		"  Proposal:        prp_0123\n" +
		"  Proposal status: proposed_outside_meeting\n" +
		"  Recorded:        2026-01-01T00:00:00Z\n"
	assertRender(t, ResourceProposalResponse, FormatFull, ProposalVoteView{ProposalVote: fullProposalVote()}, want)
}

// TestRender_ProposalResponseFull_Accepted pins that an `accepted` parent status renders
// legibly in the full form — the load-bearing auto-acceptance signal ("this response
// closed the consent window").
func TestRender_ProposalResponseFull_Accepted(t *testing.T) {
	v := fullProposalVote()
	v.ProposalStatus = "accepted"
	want := "prr_0123  recorded\n" +
		"  Response:        no_objection\n" +
		"  Proposal:        prp_0123\n" +
		"  Proposal status: accepted\n" +
		"  Recorded:        2026-01-01T00:00:00Z\n"
	assertRender(t, ResourceProposalResponse, FormatFull, ProposalVoteView{ProposalVote: v}, want)
}

// TestRender_ProposalResponseFull_NullProposalIDShowsMarker pins that a null/absent
// proposal_id renders its explicit-absence marker (none) — never <no value> or a blank.
// A missing created_at renders the (unknown) marker.
func TestRender_ProposalResponseFull_NullProposalIDShowsMarker(t *testing.T) {
	v := glassfrog.ProposalVote{ID: "prr_1", Value: "bring_to_meeting", ProposalStatus: "draft"}
	want := "prr_1  recorded\n" +
		"  Response:        bring_to_meeting\n" +
		"  Proposal:        (none)\n" +
		"  Proposal status: draft\n" +
		"  Recorded:        (unknown)\n"
	assertRender(t, ResourceProposalResponse, FormatFull, ProposalVoteView{ProposalVote: v}, want)
}

// TestRender_ProposalResponseCompact_Golden pins the one-line compact form: the prr_ id,
// the recorded value, and the parent proposal status in brackets (legible when accepted).
func TestRender_ProposalResponseCompact_Golden(t *testing.T) {
	want := "prr_0123  no_objection  [proposed_outside_meeting]\n"
	assertRender(t, ResourceProposalResponse, FormatCompact, ProposalVoteView{ProposalVote: fullProposalVote()}, want)

	v := fullProposalVote()
	v.ProposalStatus = "accepted"
	assertRender(t, ResourceProposalResponse, FormatCompact, ProposalVoteView{ProposalVote: v}, "prr_0123  no_objection  [accepted]\n")
}
