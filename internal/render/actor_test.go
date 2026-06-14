package render

import (
	"testing"

	"github.com/Luscii/cli-glassfrog/internal/glassfrog"
)

// actorDetail builds an ActorDetail with the base identity fields; the embeds are
// set by the individual tests.
func actorDetail(id, name, kind string) glassfrog.ActorDetail {
	return glassfrog.ActorDetail{Actor: glassfrog.Actor{ID: id, Name: name, Kind: kind}}
}

// --- single actor: identity line (no embeds) -------------------------------

func TestRender_ActorFull_BareActor_Golden(t *testing.T) {
	v := ActorDetailView{Detail: actorDetail("per_abc", "Alice", "human")}
	want := "per_abc  [human]\n" +
		"  Name:  Alice\n"
	assertRender(t, ResourceActor, FormatFull, v, want)
}

func TestRender_ActorCompact_BareActor_Golden(t *testing.T) {
	v := ActorDetailView{Detail: actorDetail("per_abc", "Alice", "human")}
	want := "per_abc  [human]  Alice\n"
	assertRender(t, ResourceActor, FormatCompact, v, want)
}

// A blank name renders the explicit-absence marker, not a blank — never a
// fabricated value (CONSTITUTION VIII).
func TestRender_ActorFull_BlankNameShowsMarker_Golden(t *testing.T) {
	v := ActorDetailView{Detail: actorDetail("agt_def", "  ", "agent")}
	want := "agt_def  [agent]\n" +
		"  Name:  —\n"
	assertRender(t, ResourceActor, FormatFull, v, want)
}

// --- roles footprint embed -------------------------------------------------

// The roles footprint prints each role's name, purpose, accountabilities, and
// domains; a role with no purpose / empty sections renders the absence markers,
// not blanks.
func TestRender_ActorFull_RolesFootprint_Golden(t *testing.T) {
	d := actorDetail("per_abc", "Alice", "human")
	d.Roles = []glassfrog.Role{
		roleWith("role_x", "Marketing Lead", "A market that knows us", []string{"The marketing budget"}, []string{"Defining the campaign"}),
		roleWith("role_y", "Treasurer", "   ", nil, nil),
	}
	v := ActorDetailView{Detail: d, Requested: map[string]bool{"roles": true}}
	want := "per_abc  [human]\n" +
		"  Name:  Alice\n" +
		"  Roles:\n" +
		"    Marketing Lead (role_x)\n" +
		"      Purpose: A market that knows us\n" +
		"      Domains:\n" +
		"        - The marketing budget\n" +
		"      Accountabilities:\n" +
		"        - Defining the campaign\n" +
		"    Treasurer (role_y)\n" +
		"      Purpose: (no purpose set)\n" +
		"      Domains:\n" +
		"        (none)\n" +
		"      Accountabilities:\n" +
		"        (none)\n"
	assertRender(t, ResourceActor, FormatFull, v, want)
}

// A requested-but-empty roles embed renders its section with the explicit-absence
// marker (a real "this actor fills no role" answer), distinct from an unrequested
// embed (omitted entirely).
func TestRender_ActorFull_RolesRequestedButEmpty_ShowsMarker_Golden(t *testing.T) {
	v := ActorDetailView{Detail: actorDetail("per_abc", "Alice", "human"), Requested: map[string]bool{"roles": true}}
	want := "per_abc  [human]\n" +
		"  Name:  Alice\n" +
		"  Roles:\n" +
		"    (none)\n"
	assertRender(t, ResourceActor, FormatFull, v, want)
}

// --- assignments embed -----------------------------------------------------

func TestRender_ActorFull_AssignmentsEmbed_Golden(t *testing.T) {
	d := actorDetail("per_abc", "Alice", "human")
	d.Assignments = []glassfrog.Assignment{
		{RoleID: "role_x", Focus: "Campaigns"},
		{RoleID: "role_y"},
	}
	v := ActorDetailView{Detail: d, Requested: map[string]bool{"assignments": true}}
	want := "per_abc  [human]\n" +
		"  Name:  Alice\n" +
		"  Assignments:\n" +
		"    - role_x  Campaigns\n" +
		"    - role_y  —\n"
	assertRender(t, ResourceActor, FormatFull, v, want)
}

// --- absent embed: unrequested sections are omitted, not printed empty ------

// Even when the decoded ActorDetail carries roles/assignments, an embed NOT in
// the Requested set is omitted entirely (the API returns the same empty value for
// unrequested and requested-but-empty, so Requested is the only signal — 019).
func TestRender_ActorFull_UnrequestedEmbedsOmitted_Golden(t *testing.T) {
	d := actorDetail("per_abc", "Alice", "human")
	d.Roles = []glassfrog.Role{roleWith("role_x", "Marketing Lead", "p", nil, nil)}
	d.Assignments = []glassfrog.Assignment{{RoleID: "role_x"}}
	v := ActorDetailView{Detail: d} // Requested nil → nothing requested
	want := "per_abc  [human]\n" +
		"  Name:  Alice\n"
	assertRender(t, ResourceActor, FormatFull, v, want)
}

// --- compact carries embed counts when requested ---------------------------

func TestRender_ActorCompact_WithEmbedCounts_Golden(t *testing.T) {
	d := actorDetail("agt_def", "Claude", "agent")
	d.Roles = []glassfrog.Role{
		roleWith("role_x", "A", "p", nil, nil),
		roleWith("role_y", "B", "p", nil, nil),
	}
	d.Assignments = []glassfrog.Assignment{{RoleID: "role_x"}, {RoleID: "role_y"}, {RoleID: "role_z"}}
	v := ActorDetailView{Detail: d, Requested: map[string]bool{"roles": true, "assignments": true}}
	want := "agt_def  [agent]  Claude  roles=2  assignments=3\n"
	assertRender(t, ResourceActor, FormatCompact, v, want)
}

// A compact render with no embeds requested carries the bare identity line only
// (no count suffix).
func TestRender_ActorCompact_NoEmbeds_Golden(t *testing.T) {
	v := ActorDetailView{Detail: actorDetail("agt_def", "Claude", "agent")}
	want := "agt_def  [agent]  Claude\n"
	assertRender(t, ResourceActor, FormatCompact, v, want)
}
