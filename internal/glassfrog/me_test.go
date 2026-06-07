package glassfrog

import (
	"encoding/json"
	"testing"
)

// meFixture is a representative GET /me body without the roles embed: actor,
// organization, and membership, plus extra/unknown fields the API may carry
// that decoding must tolerate.
const meFixture = `{
  "actor": {
    "id": "per_0123456789abcdef0123456789abcdef",
    "type": "actor",
    "name": "Alice Smith",
    "kind": "human",
    "created_at": "2024-01-01T00:00:00Z",
    "updated_at": "2024-02-02T00:00:00Z",
    "unexpected_actor_field": "ignored"
  },
  "organization": {
    "id": "org_0123456789abcdef0123456789abcdef",
    "name": "Acme"
  },
  "membership": {
    "id": "mem_0123456789abcdef0123456789abcdef",
    "type": "membership",
    "actor_id": "per_0123456789abcdef0123456789abcdef",
    "organization_id": "org_0123456789abcdef0123456789abcdef",
    "access_level": "admin",
    "created_at": "2024-01-01T00:00:00Z",
    "updated_at": "2024-02-02T00:00:00Z"
  },
  "unexpected_top_level": {"anything": [1, 2, 3]}
}`

// meWithRolesFixture is a GET /me?include=roles body: the same identity plus an
// embedded roles array (the minimal id+name shape, with extra role fields the
// decode must ignore).
const meWithRolesFixture = `{
  "actor": {"id": "agt_0123456789abcdef0123456789abcdef", "name": "Claude", "kind": "agent"},
  "organization": {"id": "org_0123456789abcdef0123456789abcdef", "name": "Acme"},
  "membership": {"access_level": "normal"},
  "roles": [
    {"id": "role_0123456789abcdef0123456789abcdef", "name": "Marketing Lead", "purpose": "ignored", "has_subroles": false},
    {"id": "role_00000000000000000000000000000001", "name": "Treasurer"}
  ]
}`

func TestMeResponseDecodesIdentity(t *testing.T) {
	var me MeResponse
	if err := json.Unmarshal([]byte(meFixture), &me); err != nil {
		t.Fatalf("decoding the /me fixture failed: %v", err)
	}

	if me.Actor.ID != "per_0123456789abcdef0123456789abcdef" {
		t.Errorf("actor id = %q, want the per_ handle", me.Actor.ID)
	}
	if me.Actor.Name != "Alice Smith" {
		t.Errorf("actor name = %q, want %q", me.Actor.Name, "Alice Smith")
	}
	if me.Actor.Kind != "human" {
		t.Errorf("actor kind = %q, want human", me.Actor.Kind)
	}
	if me.Organization.ID != "org_0123456789abcdef0123456789abcdef" || me.Organization.Name != "Acme" {
		t.Errorf("organization = %+v, want Acme org_…", me.Organization)
	}
	if me.Membership.AccessLevel != "admin" {
		t.Errorf("access level = %q, want admin", me.Membership.AccessLevel)
	}
	// Decoded-but-unprojected fields still populate (they are part of the schema).
	if me.Membership.ActorID == "" || me.Actor.CreatedAt == "" {
		t.Errorf("decoded-but-unprojected fields should still populate: %+v / %q", me.Membership, me.Actor.CreatedAt)
	}
}

func TestMeResponseWithoutRolesEmbedLeavesRolesEmpty(t *testing.T) {
	var me MeResponse
	if err := json.Unmarshal([]byte(meFixture), &me); err != nil {
		t.Fatalf("decoding the /me fixture failed: %v", err)
	}
	if len(me.Roles) != 0 {
		t.Errorf("Roles should be empty when ?include=roles was not requested, got %d", len(me.Roles))
	}
}

func TestMeResponseWithRolesEmbedPopulatesRoles(t *testing.T) {
	var me MeResponse
	if err := json.Unmarshal([]byte(meWithRolesFixture), &me); err != nil {
		t.Fatalf("decoding the /me?include=roles fixture failed: %v", err)
	}

	if me.Actor.Kind != "agent" || me.Actor.ID != "agt_0123456789abcdef0123456789abcdef" {
		t.Errorf("agent actor = %+v, want kind agent with an agt_ id", me.Actor)
	}
	if len(me.Roles) != 2 {
		t.Fatalf("Roles = %d, want 2", len(me.Roles))
	}
	if me.Roles[0].ID != "role_0123456789abcdef0123456789abcdef" || me.Roles[0].Name != "Marketing Lead" {
		t.Errorf("role[0] = %+v, want Marketing Lead role_…", me.Roles[0])
	}
	if me.Roles[1].Name != "Treasurer" {
		t.Errorf("role[1].Name = %q, want Treasurer", me.Roles[1].Name)
	}
}

// Unknown/extra JSON fields must not fail the decode (forward-compatible with
// API additions). The identity fixture carries unexpected_actor_field and
// unexpected_top_level; the roles fixture carries extra role fields. All three
// decoded cleanly above — this test pins the tolerance explicitly so a future
// switch to a strict decoder (DisallowUnknownFields) fails loud here.
func TestMeResponseToleratesUnknownFields(t *testing.T) {
	for name, fixture := range map[string]string{
		"identity":    meFixture,
		"with-roles":  meWithRolesFixture,
		"extra-actor": `{"actor":{"id":"per_x","name":"X","kind":"human","brand_new":42},"organization":{"id":"org_x","name":"O"},"membership":{"access_level":"normal"}}`,
	} {
		var me MeResponse
		if err := json.Unmarshal([]byte(fixture), &me); err != nil {
			t.Errorf("%s: unknown fields should be ignored, decode failed: %v", name, err)
		}
	}
}
