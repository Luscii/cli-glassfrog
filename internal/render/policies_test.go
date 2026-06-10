package render

import (
	"strings"
	"testing"

	"github.com/Luscii/cli-glassfrog/internal/glassfrog"
)

// --- policies list render (034) --------------------------------------------

// TestRender_PoliciesFull_Empty_Golden pins the inherited empty-set line: a role
// with no policies renders exactly `No policies.` (a valid empty answer).
func TestRender_PoliciesFull_Empty_Golden(t *testing.T) {
	assertRender(t, ResourcePolicies, FormatFull, PoliciesView{}, "No policies.\n")
}

func TestRender_PoliciesCompact_Empty_Golden(t *testing.T) {
	assertRender(t, ResourcePolicies, FormatCompact, PoliciesView{}, "No policies.\n")
}

// TestRender_PoliciesFull_Golden pins the list full block: one block per policy
// (title + id, role/domain scope, body), blocks separated by a blank line, with an
// explicit-absence marker for a null domain and the `(no body set)` marker for a
// blank body.
func TestRender_PoliciesFull_Golden(t *testing.T) {
	view := PoliciesView{Policies: []glassfrog.Policy{
		{ID: "pol_1", Title: "Two approvals", Body: "Body one", RoleID: "role_0123", DomainID: "dom_1"},
		{ID: "pol_2", Title: "Spending limit", Body: "", RoleID: "role_0123", DomainID: ""},
	}}
	want := "Two approvals (pol_1)\n" +
		"  Role:   role_0123\n" +
		"  Domain: dom_1\n" +
		"  Body:\n" +
		"Body one\n" +
		"\n" +
		"Spending limit (pol_2)\n" +
		"  Role:   role_0123\n" +
		"  Domain: (whole-role — no domain)\n" +
		"  Body:\n" +
		"    (no body set)\n"
	assertRender(t, ResourcePolicies, FormatFull, view, want)
}

// TestRender_PoliciesFull_OrgLevelNullRole pins the null-role_id explicit-absence
// marker (an org-level policy), never `<no value>` or an invented value.
func TestRender_PoliciesFull_OrgLevelNullRole(t *testing.T) {
	view := PoliciesView{Policies: []glassfrog.Policy{
		{ID: "pol_o", Title: "Org rule", Body: "b", RoleID: "", DomainID: ""},
	}}
	got, err := Render(ResourcePolicies, FormatFull, view)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if !strings.Contains(got, "Role:   (org-level — no role)") {
		t.Errorf("a null role_id must render the org-level marker:\n%s", got)
	}
	if strings.Contains(got, "<no value>") {
		t.Errorf("missingkey=error must never leak <no value>:\n%s", got)
	}
}

// TestRender_PoliciesCompact_Golden pins the compact convention (resource id
// first, double-space-separated key=value fragments) and the em-dash marker for a
// null domain.
func TestRender_PoliciesCompact_Golden(t *testing.T) {
	view := PoliciesView{Policies: []glassfrog.Policy{
		{ID: "pol_1", Title: "Two approvals", RoleID: "role_0123", DomainID: "dom_1"},
		{ID: "pol_2", Title: "Spending limit", RoleID: "role_0123", DomainID: ""},
	}}
	want := "pol_1  Two approvals  role=role_0123  domain=dom_1\n" +
		"pol_2  Spending limit  role=role_0123  domain=—\n"
	assertRender(t, ResourcePolicies, FormatCompact, view, want)
}

// --- policy single render (034) --------------------------------------------

// TestRender_PolicyFull_Golden pins the single full block: title + id, role/domain
// scope, timestamps, and the body. Domain is null here → its explicit-absence
// marker.
func TestRender_PolicyFull_Golden(t *testing.T) {
	view := PolicyView{Policy: glassfrog.Policy{
		ID: "pol_1", Title: "All PRs require two approvals",
		Body:      "<p>Line one</p>\n<p>Line two</p>",
		RoleID:    "role_0123",
		DomainID:  "",
		CreatedAt: "2024-01-02T03:04:05Z",
		UpdatedAt: "2024-05-06T07:08:09Z",
	}}
	want := "All PRs require two approvals (pol_1)\n" +
		"  Role:    role_0123\n" +
		"  Domain:  (whole-role — no domain)\n" +
		"  Created: 2024-01-02T03:04:05Z\n" +
		"  Updated: 2024-05-06T07:08:09Z\n" +
		"  Body:\n" +
		"<p>Line one</p>\n<p>Line two</p>\n"
	assertRender(t, ResourcePolicy, FormatFull, view, want)
}

// TestRender_PolicyFull_BodyVerbatim pins CONSTITUTION VI: a long HTML body is
// rendered verbatim — neither truncated nor reflowed. The exact body bytes must
// appear in the output unchanged.
func TestRender_PolicyFull_BodyVerbatim(t *testing.T) {
	longBody := "<h1>Policy</h1>\n" + strings.Repeat("<p>A very long paragraph that the renderer must not wrap or cut.</p>\n", 50)
	view := PolicyView{Policy: glassfrog.Policy{
		ID: "pol_long", Title: "Long one", Body: longBody,
		RoleID: "role_0123", DomainID: "dom_1",
		CreatedAt: "2024-01-02T03:04:05Z", UpdatedAt: "2024-01-02T03:04:05Z",
	}}
	got, err := Render(ResourcePolicy, FormatFull, view)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if !strings.Contains(got, longBody) {
		t.Errorf("the full body must be rendered verbatim (not truncated/reflowed):\n%s", got)
	}
}

// TestRender_PolicyFull_MissingTimestamps pins the explicit-absence guards on the
// nullable timestamp fields — never `<no value>`.
func TestRender_PolicyFull_MissingTimestamps(t *testing.T) {
	view := PolicyView{Policy: glassfrog.Policy{
		ID: "pol_x", Title: "No times", Body: "b", RoleID: "role_0123", DomainID: "dom_1",
	}}
	got, err := Render(ResourcePolicy, FormatFull, view)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	for _, want := range []string{"Created: (unknown)", "Updated: (unknown)"} {
		if !strings.Contains(got, want) {
			t.Errorf("a missing timestamp must render its guard %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "<no value>") {
		t.Errorf("missingkey=error must never leak <no value>:\n%s", got)
	}
}

// TestRender_PolicyCompact_Golden pins the single compact line (body omitted).
func TestRender_PolicyCompact_Golden(t *testing.T) {
	view := PolicyView{Policy: glassfrog.Policy{
		ID: "pol_1", Title: "All PRs require two approvals", Body: "ignored in compact",
		RoleID: "role_0123", DomainID: "",
	}}
	want := "pol_1  All PRs require two approvals  role=role_0123  domain=—\n"
	assertRender(t, ResourcePolicy, FormatCompact, view, want)
}
