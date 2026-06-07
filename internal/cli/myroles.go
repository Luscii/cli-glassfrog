package cli

import (
	"fmt"
	"strings"

	"github.com/Luscii/cli-glassfrog/internal/glassfrog"
)

// formatMyRoles renders the reshaped `glassfrog me roles` projection (never raw
// JSON; --output json is the deferred Unconsumable Output capability). It is
// pure (MyRolesResponse → string) and surfaces only response-side fields, so the
// token never appears. One block per role, blocks separated by a blank line:
//
//	<Role Name> (role_…)
//	  Purpose: <purpose>          // (no purpose set) when null/empty
//	  Domains:                    // header always renders
//	    - <domain description>    //   (none) when the role has none
//	  Accountabilities:           // header always renders, Domains-before-Accountabilities
//	    - <accountability description>
//
// An empty list is the valid "you fill no roles" answer and renders exactly
// `No roles.` (interface-cli). The role's fillers, tags, and classification
// flags are never rendered (spec Non-Behaviors) — they are not even fields on
// the shared Role type.
func formatMyRoles(resp glassfrog.MyRolesResponse) string {
	if len(resp.Data) == 0 {
		return "No roles.\n"
	}

	blocks := make([]string, 0, len(resp.Data))
	for _, r := range resp.Data {
		var b strings.Builder
		fmt.Fprintf(&b, "%s (%s)\n", r.Name, r.ID)

		purpose := strings.TrimSpace(r.Purpose)
		if purpose == "" {
			purpose = "(no purpose set)"
		}
		fmt.Fprintf(&b, "  Purpose: %s\n", purpose)

		domains := make([]string, len(r.Domains))
		for i, d := range r.Domains {
			domains[i] = d.Description
		}
		accountabilities := make([]string, len(r.Accountabilities))
		for i, a := range r.Accountabilities {
			accountabilities[i] = a.Description
		}
		writeRoleSection(&b, "Domains", domains)
		writeRoleSection(&b, "Accountabilities", accountabilities)
		blocks = append(blocks, b.String())
	}
	// Each block already ends in "\n"; joining with "\n" inserts one blank line
	// between blocks and leaves no trailing blank line after the last.
	return strings.Join(blocks, "\n")
}

// writeRoleSection writes a role's Domains/Accountabilities section: the header
// always renders (uniform, agent-parseable structure), followed by the item
// descriptions or `    (none)` when the role has none.
func writeRoleSection(b *strings.Builder, header string, items []string) {
	fmt.Fprintf(b, "  %s:\n", header)
	if len(items) == 0 {
		b.WriteString("    (none)\n")
		return
	}
	for _, d := range items {
		fmt.Fprintf(b, "    - %s\n", d)
	}
}

// incomplete reports whether the API signalled that more roles exist than this
// (first) response carried — i.e. Meta.Pagination.HasNextPage. The command turns
// a true result into the single stderr incompleteness note (still exit 0); this
// slice never follows pagination (deferred to 016).
func incomplete(resp glassfrog.MyRolesResponse) bool {
	return resp.Meta.Pagination.HasNextPage
}
