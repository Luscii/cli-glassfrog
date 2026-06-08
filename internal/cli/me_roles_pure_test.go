package cli

import (
	"testing"

	"github.com/Luscii/cli-glassfrog/internal/glassfrog"
)

// The `me roles` full projection is now rendered through internal/render (019);
// its byte-equivalence with the pre-019 formatMeRoles output (blocks separated by
// a blank line, (no purpose set) / (none) absence markers, the No roles. empty
// line) is pinned by that package's goldens (TestRender_RolesFull_*,
// TestRender_EmptyResultSets_ExplicitLine). The end-to-end success path stays
// covered by the me-roles BDD/unit suites.

// TestIncomplete pins the pagination incompleteness signal the command still
// owns (orthogonal to rendering): more-roles-exist → the stderr note, exit 0.
func TestIncomplete(t *testing.T) {
	var withNext glassfrog.MyRolesResponse
	withNext.Meta.Pagination.HasNextPage = true
	if !incomplete(withNext) {
		t.Error("incomplete should be true when HasNextPage is set")
	}

	var complete glassfrog.MyRolesResponse
	complete.Meta.Pagination.HasNextPage = false
	if incomplete(complete) {
		t.Error("incomplete should be false when HasNextPage is unset")
	}
}
