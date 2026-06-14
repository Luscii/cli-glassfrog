package glassfrog

import (
	"encoding/json"
	"testing"
)

// actorBareDoc is a representative GET /actors/{id} body with NO embeds: the
// {data: ActorDetail} document carrying only the base Actor fields (the API
// omits roles/assignments when ?include was not requested). It carries an
// unknown top-level field the decode must tolerate, and uses the API's
// snake_case names so a missing JSON tag would fail loud here.
const actorBareDoc = `{
  "data": {
    "id": "per_0123456789abcdef0123456789abcdef",
    "name": "Alice Smith",
    "kind": "human",
    "created_at": "2024-01-02T03:04:05Z",
    "updated_at": "2024-02-03T04:05:06Z"
  },
  "unexpected_top_level": {"anything": [1, 2, 3]}
}`

// actorWithRolesDoc embeds the actor's governance footprint (?include=roles):
// each Role carries its FULL shape — purpose, accountabilities, domains — proving
// ActorDetail reuses the landed full Role type, not a minimal leaf.
const actorWithRolesDoc = `{
  "data": {
    "id": "agt_0123456789abcdef0123456789abcdef",
    "name": "Claude",
    "kind": "agent",
    "roles": [
      {
        "id": "role_0123456789abcdef0123456789abcdef",
        "type": "role",
        "name": "Marketing Lead",
        "purpose": "A market that knows us",
        "domains": [{"id": "dom_1", "description": "The marketing budget"}],
        "accountabilities": [{"id": "acc_1", "description": "Defining the campaign"}]
      },
      {"id": "role_00000000000000000000000000000001", "name": "Treasurer", "purpose": null}
    ]
  }
}`

// actorWithAssignmentsDoc embeds the actor's assignments (?include=assignments),
// reusing the landed Assignment leaf model.
const actorWithAssignmentsDoc = `{
  "data": {
    "id": "per_0123456789abcdef0123456789abcdef",
    "name": "Alice Smith",
    "kind": "human",
    "assignments": [
      {"id": "asgn_1", "actor_id": "per_0123456789abcdef0123456789abcdef", "role_id": "role_0123456789abcdef0123456789abcdef", "focus": "Campaigns"},
      {"id": "asgn_2", "actor_id": "per_0123456789abcdef0123456789abcdef", "role_id": "role_00000000000000000000000000000001"}
    ]
  }
}`

// TestActorDetailDecodesBareActor pins the document wrapper and the embedded
// Actor's snake_case tags, with the two optional embeds nil/empty when absent.
func TestActorDetailDecodesBareActor(t *testing.T) {
	var doc ActorDocument
	if err := json.Unmarshal([]byte(actorBareDoc), &doc); err != nil {
		t.Fatalf("decoding the bare /actors/{id} document failed: %v", err)
	}
	d := doc.Data
	if d.ID != "per_0123456789abcdef0123456789abcdef" {
		t.Errorf("ID = %q, want the per_ id", d.ID)
	}
	if d.Name != "Alice Smith" {
		t.Errorf("Name = %q, want \"Alice Smith\"", d.Name)
	}
	if d.Kind != "human" {
		t.Errorf("Kind = %q, want \"human\"", d.Kind)
	}
	if d.CreatedAt != "2024-01-02T03:04:05Z" || d.UpdatedAt != "2024-02-03T04:05:06Z" {
		t.Errorf("timestamps did not bind: created=%q updated=%q", d.CreatedAt, d.UpdatedAt)
	}
	if d.Roles != nil {
		t.Errorf("Roles should be nil with no embed, got %v", d.Roles)
	}
	if d.Assignments != nil {
		t.Errorf("Assignments should be nil with no embed, got %v", d.Assignments)
	}
}

// TestActorDetailDecodesRolesEmbed pins the ?include=roles footprint: each
// embedded role carries its full shape (purpose, accountabilities, domains),
// proving the landed full Role type is reused verbatim.
func TestActorDetailDecodesRolesEmbed(t *testing.T) {
	var doc ActorDocument
	if err := json.Unmarshal([]byte(actorWithRolesDoc), &doc); err != nil {
		t.Fatalf("decoding the roles-embedded document failed: %v", err)
	}
	d := doc.Data
	if d.ID != "agt_0123456789abcdef0123456789abcdef" || d.Kind != "agent" {
		t.Errorf("agent identity did not bind: id=%q kind=%q", d.ID, d.Kind)
	}
	if len(d.Roles) != 2 {
		t.Fatalf("Roles len = %d, want 2", len(d.Roles))
	}
	r0 := d.Roles[0]
	if r0.Name != "Marketing Lead" || r0.Purpose != "A market that knows us" {
		t.Errorf("role 0 name/purpose did not bind: %q / %q", r0.Name, r0.Purpose)
	}
	if len(r0.Domains) != 1 || r0.Domains[0].Description != "The marketing budget" {
		t.Errorf("role 0 domains did not bind: %v", r0.Domains)
	}
	if len(r0.Accountabilities) != 1 || r0.Accountabilities[0].Description != "Defining the campaign" {
		t.Errorf("role 0 accountabilities did not bind: %v", r0.Accountabilities)
	}
	if d.Assignments != nil {
		t.Errorf("Assignments should be nil when only roles were embedded, got %v", d.Assignments)
	}
}

// TestActorDetailDecodesAssignmentsEmbed pins the ?include=assignments embed,
// reusing the landed Assignment shape.
func TestActorDetailDecodesAssignmentsEmbed(t *testing.T) {
	var doc ActorDocument
	if err := json.Unmarshal([]byte(actorWithAssignmentsDoc), &doc); err != nil {
		t.Fatalf("decoding the assignments-embedded document failed: %v", err)
	}
	d := doc.Data
	if len(d.Assignments) != 2 {
		t.Fatalf("Assignments len = %d, want 2", len(d.Assignments))
	}
	a0 := d.Assignments[0]
	if a0.RoleID != "role_0123456789abcdef0123456789abcdef" || a0.Focus != "Campaigns" {
		t.Errorf("assignment 0 did not bind: role_id=%q focus=%q", a0.RoleID, a0.Focus)
	}
	if a0.ActorID != "per_0123456789abcdef0123456789abcdef" {
		t.Errorf("assignment 0 actor_id did not bind: %q", a0.ActorID)
	}
	if d.Roles != nil {
		t.Errorf("Roles should be nil when only assignments were embedded, got %v", d.Roles)
	}
}
