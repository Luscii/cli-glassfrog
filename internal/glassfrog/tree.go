package glassfrog

// TreeNode is the recursive circle-hierarchy node the org-tree reads decode
// (getOrgTree GET /tree, getRoleTree GET /roles/{id}/tree — spec §5571). It is a
// DIFFERENT shape from Role/RoleDetail (026 ADR-2), so it is its own type rather
// than a grown Role: it carries the recursion (Children []TreeNode), a
// constrained Flags enum (structural/elected/linked), and a Type discriminator,
// none of which belong on the flat Role. The two tree reads return ONE TreeNode
// in a single, unpaginated document — there is no Page envelope here.
//
// Name, Purpose, and ParentRoleID are *string because the API field is nullable:
// a pointer distinguishes "field absent / null" (nil) from "present but empty"
// (a non-nil pointer to ""). ParentRoleID is null on the anchor (root) node.
//
// The optional Accountabilities/Domains/Fillers fields populate only when the
// matching ?include value is requested (accountabilities/domains/members) and
// stay nil otherwise; they reuse the existing leaf projections (Accountability,
// Domain, Actor — 011/012/025) rather than forking tree-local copies. Every field
// carries an explicit snake_case JSON tag — encoding/json is case-insensitive but
// does not bridge underscores, so an untagged field would silently never bind.
// Decoding is tolerant of unknown/extra fields (011): a node body that carries
// fields this projection drops simply ignores them.
type TreeNode struct {
	ID               string           `json:"id"`
	Type             string           `json:"type"`
	Name             *string          `json:"name"`
	Purpose          *string          `json:"purpose"`
	ParentRoleID     *string          `json:"parent_role_id"`
	HasSubroles      bool             `json:"has_subroles"`
	Flags            []string         `json:"flags"`
	Children         []TreeNode       `json:"children"`
	Accountabilities []Accountability `json:"accountabilities"`
	Domains          []Domain         `json:"domains"`
	Fillers          []Actor          `json:"fillers"`
}

// TreeDocument is the single-object {data: TreeNode} envelope the tree reads
// return — the unpaginated counterpart to Page[T] (016) and a sibling of
// RoleDocument (025). It is named (not a generic Document[T]) for the same reason
// RoleDocument is: a later read that wants the same wrapper may generalize it.
type TreeDocument struct {
	Data TreeNode `json:"data"`
}
