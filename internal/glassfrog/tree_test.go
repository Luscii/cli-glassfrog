package glassfrog

import (
	"encoding/json"
	"testing"
)

// orgTreeFixture is a representative GET /tree body: the {data: TreeNode} wrapper
// with a nested hierarchy three levels deep, the anchor carrying a null
// parent_role_id and a structural flag, each ?include field present on the
// anchor, and unknown/extra fields the decode must tolerate. Snake_case names
// throughout so a missing JSON tag fails loud here.
const orgTreeFixture = `{
  "data": {
    "id": "role_anchor",
    "type": "circle",
    "name": "General Company Circle",
    "purpose": "Run the company",
    "parent_role_id": null,
    "has_subroles": true,
    "flags": ["structural", "elected"],
    "accountabilities": [{"id": "acc_1", "description": "Holding the whole"}],
    "domains": [{"id": "dom_1", "description": "The company"}],
    "fillers": [{"id": "per_1", "name": "Alice", "kind": "human"}],
    "unknown_field": "ignored",
    "children": [
      {
        "id": "role_mid",
        "type": "circle",
        "name": "Marketing",
        "purpose": null,
        "parent_role_id": "role_anchor",
        "has_subroles": true,
        "flags": ["linked"],
        "children": [
          {
            "id": "role_leaf",
            "type": "role",
            "name": "Press Officer",
            "purpose": "Press that lands",
            "parent_role_id": "role_mid",
            "has_subroles": false,
            "flags": [],
            "children": []
          }
        ]
      }
    ]
  }
}`

func TestTreeDocument_DecodesRecursiveHierarchy(t *testing.T) {
	var doc TreeDocument
	if err := json.Unmarshal([]byte(orgTreeFixture), &doc); err != nil {
		t.Fatalf("decode org tree: %v", err)
	}

	root := doc.Data
	if root.ID != "role_anchor" || root.Type != "circle" {
		t.Errorf("root id/type = %q/%q, want role_anchor/circle", root.ID, root.Type)
	}
	if root.Name == nil || *root.Name != "General Company Circle" {
		t.Errorf("root name = %v, want General Company Circle", root.Name)
	}
	if root.Purpose == nil || *root.Purpose != "Run the company" {
		t.Errorf("root purpose = %v, want Run the company", root.Purpose)
	}
	// The anchor's parent_role_id is null → nil pointer (no parent), distinct from
	// a present-but-empty id.
	if root.ParentRoleID != nil {
		t.Errorf("root parent_role_id = %v, want nil (anchor)", *root.ParentRoleID)
	}
	if !root.HasSubroles {
		t.Error("root has_subroles = false, want true")
	}
	if len(root.Flags) != 2 || root.Flags[0] != "structural" || root.Flags[1] != "elected" {
		t.Errorf("root flags = %v, want [structural elected]", root.Flags)
	}

	// Recursion: children nest to multiple levels.
	if len(root.Children) != 1 {
		t.Fatalf("root children = %d, want 1", len(root.Children))
	}
	mid := root.Children[0]
	if mid.ID != "role_mid" {
		t.Errorf("mid id = %q, want role_mid", mid.ID)
	}
	// A null purpose decodes as a nil pointer.
	if mid.Purpose != nil {
		t.Errorf("mid purpose = %v, want nil (null)", *mid.Purpose)
	}
	if mid.ParentRoleID == nil || *mid.ParentRoleID != "role_anchor" {
		t.Errorf("mid parent_role_id = %v, want role_anchor", mid.ParentRoleID)
	}
	if len(mid.Children) != 1 {
		t.Fatalf("mid children = %d, want 1", len(mid.Children))
	}
	leaf := mid.Children[0]
	if leaf.ID != "role_leaf" || leaf.HasSubroles {
		t.Errorf("leaf id/has_subroles = %q/%v, want role_leaf/false", leaf.ID, leaf.HasSubroles)
	}
}

func TestTreeNode_IncludeFieldsPopulateWhenPresent(t *testing.T) {
	var doc TreeDocument
	if err := json.Unmarshal([]byte(orgTreeFixture), &doc); err != nil {
		t.Fatalf("decode: %v", err)
	}
	root := doc.Data
	if len(root.Accountabilities) != 1 || root.Accountabilities[0].Description != "Holding the whole" {
		t.Errorf("accountabilities = %v, want one 'Holding the whole'", root.Accountabilities)
	}
	if len(root.Domains) != 1 || root.Domains[0].Description != "The company" {
		t.Errorf("domains = %v, want one 'The company'", root.Domains)
	}
	if len(root.Fillers) != 1 || root.Fillers[0].Name != "Alice" {
		t.Errorf("fillers = %v, want one 'Alice'", root.Fillers)
	}
}

func TestTreeNode_IncludeFieldsAbsentStayNil(t *testing.T) {
	// A node body with no include fields leaves them nil/empty (not requested).
	const body = `{"data": {"id": "role_x", "type": "role", "name": "X", "has_subroles": false, "children": []}}`
	var doc TreeDocument
	if err := json.Unmarshal([]byte(body), &doc); err != nil {
		t.Fatalf("decode: %v", err)
	}
	root := doc.Data
	if root.Accountabilities != nil || root.Domains != nil || root.Fillers != nil {
		t.Errorf("unrequested include fields should stay nil, got acc=%v dom=%v fil=%v",
			root.Accountabilities, root.Domains, root.Fillers)
	}
	if len(root.Children) != 0 {
		t.Errorf("leaf children = %d, want 0", len(root.Children))
	}
}

func TestTreeNode_LeafDecodesWithEmptyChildren(t *testing.T) {
	// A leaf node with an absent children key decodes to a nil (empty) Children
	// without error — and unknown fields are ignored.
	const body = `{"data": {"id": "role_y", "type": "role", "name": "Y", "has_subroles": false, "extra": {"nested": 1}}}`
	var doc TreeDocument
	if err := json.Unmarshal([]byte(body), &doc); err != nil {
		t.Fatalf("decode leaf with unknown fields: %v", err)
	}
	if len(doc.Data.Children) != 0 {
		t.Errorf("leaf children = %d, want 0", len(doc.Data.Children))
	}
}
