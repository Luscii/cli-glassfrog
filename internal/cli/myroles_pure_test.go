package cli

import (
	"strings"
	"testing"

	"github.com/Luscii/cli-glassfrog/internal/glassfrog"
)

// roleWith builds a Role with the given fields for the pure-renderer tests.
func roleWith(id, name, purpose string, domains, accountabilities []string) glassfrog.Role {
	r := glassfrog.Role{ID: id, Name: name, Purpose: purpose}
	for _, d := range domains {
		r.Domains = append(r.Domains, struct {
			Description string `json:"description"`
		}{Description: d})
	}
	for _, a := range accountabilities {
		r.Accountabilities = append(r.Accountabilities, struct {
			Description string `json:"description"`
		}{Description: a})
	}
	return r
}

func respWith(roles ...glassfrog.Role) glassfrog.MyRolesResponse {
	return glassfrog.MyRolesResponse{Data: roles}
}

// --- formatMyRoles ---------------------------------------------------------

func TestFormatMyRoles_FullRole(t *testing.T) {
	resp := respWith(roleWith(
		"role_0123456789abcdef0123456789abcdef", "Marketing Lead", "A market that knows us",
		[]string{"The marketing budget", "The brand guidelines"},
		[]string{"Defining the campaign", "Reporting reach", "Maintaining the press list"},
	))
	out := formatMyRoles(resp)

	for _, want := range []string{
		"Marketing Lead (role_0123456789abcdef0123456789abcdef)",
		"Purpose: A market that knows us",
		"Domains:",
		"- The marketing budget",
		"- The brand guidelines",
		"Accountabilities:",
		"- Defining the campaign",
		"- Maintaining the press list",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("projection missing %q:\n%s", want, out)
		}
	}

	// Domains must render before Accountabilities (interface-cli, matches the web UI).
	if strings.Index(out, "Domains:") > strings.Index(out, "Accountabilities:") {
		t.Errorf("Domains should render before Accountabilities:\n%s", out)
	}
}

func TestFormatMyRoles_NullPurpose(t *testing.T) {
	out := formatMyRoles(respWith(roleWith("role_x", "Treasurer", "", nil, nil)))
	if !strings.Contains(out, "Purpose: (no purpose set)") {
		t.Errorf("a null/empty purpose should render `(no purpose set)`:\n%s", out)
	}
	// A whitespace-only purpose is also treated as unset.
	out2 := formatMyRoles(respWith(roleWith("role_x", "Treasurer", "   ", nil, nil)))
	if !strings.Contains(out2, "Purpose: (no purpose set)") {
		t.Errorf("a whitespace-only purpose should render `(no purpose set)`:\n%s", out2)
	}
}

// A role with neither domains nor accountabilities still renders both headers,
// each followed by `(none)` — the structure is uniform for agent parsing.
func TestFormatMyRoles_NoDomainsNoAccountabilities(t *testing.T) {
	out := formatMyRoles(respWith(roleWith("role_x", "Treasurer", "Sound books", nil, nil)))
	if !strings.Contains(out, "Domains:") || !strings.Contains(out, "Accountabilities:") {
		t.Errorf("both headers should always render:\n%s", out)
	}
	if strings.Count(out, "(none)") != 2 {
		t.Errorf("an empty section should render `(none)` under each header (want 2):\n%s", out)
	}
}

func TestFormatMyRoles_MultipleRolesSeparatedByBlankLine(t *testing.T) {
	resp := respWith(
		roleWith("role_1", "Marketing Lead", "p1", []string{"d1"}, []string{"a1"}),
		roleWith("role_2", "Treasurer", "p2", nil, nil),
	)
	out := formatMyRoles(resp)
	if !strings.Contains(out, "Marketing Lead (role_1)") || !strings.Contains(out, "Treasurer (role_2)") {
		t.Errorf("both role blocks should render:\n%s", out)
	}
	if !strings.Contains(out, "\n\n") {
		t.Errorf("role blocks should be separated by a blank line:\n%s", out)
	}
	// No trailing blank line after the last block.
	if strings.HasSuffix(out, "\n\n") {
		t.Errorf("there should be no trailing blank line after the last block:\n%q", out)
	}
}

func TestFormatMyRoles_EmptyList(t *testing.T) {
	out := formatMyRoles(respWith())
	if strings.TrimRight(out, "\n") != "No roles." {
		t.Errorf("an empty role list should render exactly `No roles.`, got %q", out)
	}
}

// The projection must never surface fillers, tags, or classification flags. They
// are not fields on the shared Role, so the strongest guarantee is structural:
// the rendered output contains none of those labels for any input.
func TestFormatMyRoles_NeverShowsFillersTagsFlags(t *testing.T) {
	out := formatMyRoles(respWith(roleWith(
		"role_x", "Marketing Lead", "p", []string{"d1"}, []string{"a1"},
	)))
	for _, forbidden := range []string{"filler", "Filler", "tag", "Tag", "flag", "Flag"} {
		if strings.Contains(out, forbidden) {
			t.Errorf("projection must not surface %q:\n%s", forbidden, out)
		}
	}
}

// --- incomplete ------------------------------------------------------------

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
