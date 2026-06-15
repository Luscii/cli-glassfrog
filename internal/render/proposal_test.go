package render

import (
	"testing"

	"github.com/Luscii/cli-glassfrog/internal/glassfrog"
)

// fullProposal is a representative proposal with every field populated, so the full
// golden pins each label and value. The changes carry free-form command-specific keys
// (beyond id/type) so the per-type rendering is exercised verbatim.
func fullProposal() glassfrog.Proposal {
	return glassfrog.Proposal{
		ID:         "prp_1",
		Type:       "proposal",
		Status:     "draft",
		TensionID:  "ten_0123",
		CircleID:   "role_0123",
		ProposerID: "per_0123",
		Changes: []glassfrog.ProposalChange{
			{ID: "chg_1", Type: "CreateRole", Fields: map[string]any{"id": "chg_1", "type": "CreateRole", "name": "Scribe"}},
			{ID: "chg_2", Type: "UpdateRole", Fields: map[string]any{"id": "chg_2", "type": "UpdateRole"}},
		},
		ResponseSummary:       glassfrog.ResponseSummary{Total: 3, NoObjection: 2, BringToMeeting: 1},
		ExpectedResponseCount: 3,
		ReceivedResponseCount: 2,
		AvailableTransitions:  []string{"propose", "withdraw"},
		ProposedAt:            "2026-01-03T00:00:00Z",
		ResponseDeadline:      "2026-01-10T00:00:00Z",
		AcceptedAt:            "",
		CreatedAt:             "2026-01-01T00:00:00Z",
		UpdatedAt:             "2026-01-02T00:00:00Z",
	}
}

// TestRender_ProposalFull_Golden pins the grown 056 full projection: the prp_ id +
// status header, the anchor/circle/proposer lines, the lifecycle timestamps, the
// aggregate response counts, the expected/received counts, the available transitions,
// and each change rendered BY TYPE with its free-form properties verbatim. The change
// with no extra properties (chg_2) renders only its type badge.
func TestRender_ProposalFull_Golden(t *testing.T) {
	want := "prp_1  [draft]\n" +
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
	assertRender(t, ResourceProposal, FormatFull, ProposalView{Proposal: fullProposal()}, want)
}

// TestRender_ProposalFull_NullFieldsShowMarkers pins that every nullable field
// (tension_id, circle_id, proposer_id, and the lifecycle timestamps) renders its
// explicit-absence marker (none) — never <no value> or a blank. A fresh draft with no
// changes still renders a zero count and zero response tallies, and an empty changes
// list renders the `Changes (0):` header with no rows.
func TestRender_ProposalFull_NullFieldsShowMarkers(t *testing.T) {
	p := glassfrog.Proposal{ID: "prp_1", Status: "draft"}
	want := "prp_1  [draft]\n" +
		"  Tension:        (none)\n" +
		"  Circle:         (none)\n" +
		"  Proposer:       (none)\n" +
		"  Proposed:       (none)\n" +
		"  Deadline:       (none)\n" +
		"  Accepted:       (none)\n" +
		"  Responses:      0 total — 0 no-objection, 0 bring-to-meeting\n" +
		"  Expected/recv:  0 / 0\n" +
		"  Transitions:    (none)\n" +
		"  Changes (0):\n"
	assertRender(t, ResourceProposal, FormatFull, ProposalView{Proposal: p}, want)
}

// TestRender_ProposalCompact_Golden pins the one-line compact form: the prp_ id (the
// load-bearing handle), the status badge, the change count, and the response total.
func TestRender_ProposalCompact_Golden(t *testing.T) {
	want := "prp_1  [draft]  2 change(s)  3 responses\n"
	assertRender(t, ResourceProposal, FormatCompact, ProposalView{Proposal: fullProposal()}, want)
}

// TestRender_ProposalsFull_Golden pins the plural list projection: one block per
// proposal (id + status, indented proposer, change count, and the one-line aggregate
// response summary), with a null proposer rendering the `—` absence marker.
func TestRender_ProposalsFull_Golden(t *testing.T) {
	data := []glassfrog.Proposal{
		fullProposal(),
		{ID: "prp_2", Status: "proposed_outside_meeting", ResponseSummary: glassfrog.ResponseSummary{Total: 0}},
	}
	want := "prp_1  [draft]\n" +
		"  proposer: per_0123\n" +
		"  changes: 2\n" +
		"  responses: 3 total (2 no-objection, 1 bring-to-meeting)\n" +
		"prp_2  [proposed_outside_meeting]\n" +
		"  proposer: —\n" +
		"  changes: 0\n" +
		"  responses: 0 total (0 no-objection, 0 bring-to-meeting)\n"
	assertRender(t, ResourceProposals, FormatFull, ProposalsView{Data: data}, want)
}

// TestRender_ProposalsCompact_Golden pins the one-line-per-proposal compact form.
func TestRender_ProposalsCompact_Golden(t *testing.T) {
	data := []glassfrog.Proposal{
		fullProposal(),
		{ID: "prp_2", Status: "accepted", Changes: []glassfrog.ProposalChange{{Type: "X"}}},
	}
	want := "prp_1  [draft]  2 change(s)\n" +
		"prp_2  [accepted]  1 change(s)\n"
	assertRender(t, ResourceProposals, FormatCompact, ProposalsView{Data: data}, want)
}

// TestRender_ProposalsEmpty_ShowsExplicitLine pins that an empty visible set renders
// the explicit `no proposals` line (not <no value>, not blank) in both formats — an
// empty list is a valid answer, not an error.
func TestRender_ProposalsEmpty_ShowsExplicitLine(t *testing.T) {
	for _, format := range []Format{FormatFull, FormatCompact} {
		assertRender(t, ResourceProposals, format, ProposalsView{Data: nil}, "no proposals\n")
	}
}
