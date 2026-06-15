package cli

import "github.com/spf13/cobra"

// Assemble builds the fully-wired root command: it creates the root and
// attaches every top-level command and group through the registration guard.
//
// This is the single, explicit wiring site (ADR-4). The entrypoint calls it
// and executes the result; commands never self-register via package init.
// Adding a command means one MustRegister line here plus the command's own
// constructor — no existing command is edited.
func Assemble() *cobra.Command {
	root := NewRootCommand()
	// Top-level commands and groups are wired here — one MustRegister line
	// each. Adding a command does not touch the others.
	MustRegister(root, newVersionCommand())
	// Role Reads (025): the org-wide `roles` command (list + single read). It
	// REPLACES the earlier stub `roles` group. productionSeam binds the real
	// transport (AssembleFromOS + NewClientFromOS) and clock; `roles` reads the
	// inherited persistent --base-url/--output flags.
	MustRegister(root, newRolesCommand(productionSeam{}))
	// Organization Tree (026): the `tree` command — the whole-org tree (no id) or
	// the subtree rooted at a role (one id), one unpaginated nested document. A
	// sibling of `roles`, not a child (ADR-1). productionSeam binds the real
	// transport + clock; `tree` reads the inherited persistent --base-url/--output.
	MustRegister(root, newTreeCommand(productionSeam{}))
	// Organization Tree (026): the `subroles <id>` command — a role's immediate
	// children, paginated and walked to completion by default (reusing 025's walk +
	// --first-page opt-out). Also a sibling of `roles` (ADR-1).
	MustRegister(root, newSubrolesCommand(productionSeam{}))
	// Role Domains (033): the `domains <role-id>` command — the domains a role
	// controls, paginated and walked to completion by default (reusing 025's walk +
	// --first-page opt-out), searchable with --query. A sibling of `roles`, not a
	// child (plan ADR-1). productionSeam binds the real transport + clock; it reads
	// the inherited persistent --base-url/--output.
	MustRegister(root, newDomainsCommand(productionSeam{}))
	// Role Domains (033): the `domain <dom-id>` command — one domain read by its
	// own id, optionally embedding its policies with --include policies. The
	// singular sibling of the plural `domains` list (plan ADR-1); unpaginated, so
	// it issues one request and rejects the list's walk/search flags. productionSeam
	// binds the real transport; it reads the inherited persistent --base-url/--output.
	MustRegister(root, newDomainCommand(productionSeam{}))
	// Role Policies (034): the `policies <role-id>` command — the addressable
	// per-role policy list, paginated and walked to completion by default (reusing
	// 025's walk + --first-page opt-out), narrowable with --query. A sibling of
	// `roles`, not a child (ADR-1). productionSeam binds the real transport + clock;
	// `policies` reads the inherited persistent --base-url/--output.
	MustRegister(root, newPoliciesCommand(productionSeam{}))
	// Role Policies (034): the `policy <pol-id>` command — the standalone single
	// policy read with its full body, decoded from a {data: Policy} document. A
	// sibling of `policies`, declaring no list flags so a list-only flag is a cobra
	// unknown-flag usage error (the structural list-only guard, ADR-1). Shares the
	// productionSeam; reads the inherited persistent --base-url/--output.
	MustRegister(root, newPolicyCommand(productionSeam{}))
	// Role Projects (038): the `projects <role-id>` command — the addressable
	// per-role project list, paginated and walked to completion by default (reusing
	// 025's walk + --first-page opt-out), narrowable with --query/--status/--tag. A
	// sibling of `roles`, not a child (ADR-1). productionSeam binds the real
	// transport + clock; `projects` reads the inherited persistent --base-url/--output.
	// Distinct from `me projects` (014): role-addressable, not token-scoped.
	MustRegister(root, newProjectsCommand(productionSeam{}))
	// Role Projects (038): the `project <proj-id>` command — the standalone single
	// project read with its full detail, decoded from a {data: Project} document. A
	// sibling of `projects`, declaring no list flags so a list-only flag is a cobra
	// unknown-flag usage error (the structural list-only guard, ADR-1). Shares the
	// productionSeam; reads the inherited persistent --base-url/--output.
	MustRegister(root, newProjectCommand(productionSeam{}))
	// Cross-Model Search (041): the `search <query>` command — one relevance-ranked
	// full-text query across all eight resource types (GET /search), walked to
	// completion by default (reusing 025/026's walk + --first-page opt-out,
	// parameterized on SearchResult). An org-wide, cross-type sibling — child of no
	// resource group (ADR-1). The required positional query is forwarded verbatim;
	// --types scopes it (reject-unknown over the closed 8-value set). productionSeam
	// binds the real transport + clock; it reads the inherited persistent
	// --base-url/--output.
	MustRegister(root, newSearchCommand(productionSeam{}))
	// Actor Directory (048): the `actors` command — the org-wide actor directory
	// (GET /actors), walked to completion by default (reusing 025's walk +
	// --first-page opt-out). The first read keyed purely on flags (cobra.NoArgs); its
	// subject is the whole organization, narrowed by the optional --kind/--role-id/
	// --query filters. --kind is validated locally (reject-unknown over {human,
	// agent}); --role-id/--query are passed through. An org-wide sibling — child of no
	// resource group (ADR-1). No `people`/`agents` command — --kind covers actor-kind
	// selection through the ungated /actors. productionSeam binds the real transport +
	// clock; it reads the inherited persistent --base-url/--output.
	MustRegister(root, newActorsCommand(productionSeam{}))
	// Role Fillers (047): the `fillers <role-id>` command — the role-scoped read of
	// who fills a role (GET /roles/{role_id}/assignments), walked to completion by
	// default (reusing 025's walk + --first-page opt-out). Each row leads with the
	// filling actor (per_/agt_ id + kind badge + name) then the assignment's focus and
	// election context, rendered through the new `fillers` key (plan ADR-2). A required
	// positional role id (cobra.ExactArgs(1)); no filter flags and no --include — the
	// endpoint offers none beyond the default include + pagination (ADR-3). No singular
	// sibling — the API exposes no GET /assignments/{id} (ADR-1). productionSeam binds
	// the real transport + clock; it reads the inherited persistent --base-url/--output.
	MustRegister(root, newFillersCommand(productionSeam{}))
	// Actor Assignments (050): the `assignments <actor-id>` command — the actor-scoped
	// read of the roles an actor fills (GET /actors/{actor_id}/assignments), walked to
	// completion by default (reusing 025's walk + --first-page opt-out). The actor-end
	// mirror of `fillers`: each row leads with the FILLED ROLE (its role_ id + name +
	// purpose/parent context the default ?include=role embeds) then the assignment's
	// focus and election, rendered through the new `assignments` key (plan ADR-2). A
	// required positional actor id (cobra.ExactArgs(1)); no filter flags and no
	// --include — the endpoint offers none beyond the default include + pagination
	// (ADR-3). No singular sibling — the API exposes no GET /assignments/{id} (ADR-1).
	// productionSeam binds the real transport + clock; it reads the inherited persistent
	// --base-url/--output.
	MustRegister(root, newAssignmentsCommand(productionSeam{}))
	// Subrole Filler Roll-up (051): the `subrole-actors <role-id>` command — the
	// cross-role roll-up of the actors filling the anchor role's DIRECT sub-roles
	// (GET /roles/{role_id}/subroles/actors), one level not transitive, walked to
	// completion by default (reusing 025's walk + --first-page opt-out). The actor-shaped
	// twin of `tension subroles` (046) and the cross-role counterpart of `actors
	// --role-id` (048): rows are bare actors (id/name/kind), rendered through the landed
	// `actors` key. A required positional anchor role id (cobra.ExactArgs(1)); only
	// --kind among the list filters (the endpoint offers no role_id/q — ADR-2). Its OWN
	// top-level read leaf, NOT a subcommand of the positional-bearing `actors` (ADR-1).
	// A leaf anchor's 404 is surfaced verbatim, distinct from an empty-200 success
	// (ADR-3). productionSeam binds the real transport + clock; it reads the inherited
	// persistent --base-url/--output.
	MustRegister(root, newSubroleActorsCommand(productionSeam{}))
	// Tension Capture (042): the `tension` group + `create` leaf — the CLI's first
	// write. `tension create <role-id>` POSTs a captured tension (the seed of a
	// governance proposal) and prints the created tension with its ten_ id. A
	// non-runnable group parenting one leaf (the auth/auth login shape, ADR-2),
	// built with its child before registration so the guard's ">=1 child" rule
	// holds. productionSeam binds the real transport + clock; the leaf reads the
	// inherited persistent --base-url/--output.
	MustRegister(root, newTensionCommand(productionSeam{}))
	// Proposal Creation (055): the `proposal` group + `create` leaf — the CLI's second
	// write and the anchor of the governance write path. `proposal create <tension-id>
	// --changes <src>` POSTs a draft proposal carrying a caller-supplied governance
	// change set (inline / file / piped stdin) and prints the created proposal with its
	// prp_ id. A non-runnable group parenting one leaf (the tension shape, ADR-1), built
	// with its child before registration so the guard's ">=1 child" rule holds. The
	// group is shared with Proposal Reads (056) under first-to-land-creates.
	MustRegister(root, newProposalCommand(productionSeam{}))
	// Credential Storage (006): the auth group + login leaf, delegating the file
	// write to internal/auth through the production input seam.
	MustRegister(root, newAuthCommand(productionSeam{}))
	// Identity Read (011): the `me` command — the first end-to-end API read.
	// productionSeam binds AssembleFromOS + NewClientFromOS (the real transport);
	// me reads the persistent --base-url flag the root declares. me is registered
	// as a leaf FIRST (while childless), so the guard sees a valid runnable leaf;
	// My Roles' `roles` child is attached AFTER, so the guard validates the child
	// and `me` becomes both runnable and a parent without tripping the
	// leaf-xor-group rule (which only fires at a command's own registration).
	meCmd := newMeCommand(productionSeam{})
	MustRegister(root, meCmd)
	// My Roles (012): the `roles` leaf under the runnable `me` command. `me roles`
	// sends GET /me/roles and prints the roles the practitioner fills. Reuses the
	// same productionSeam (assemble + newClient) and the inherited --base-url.
	MustRegister(meCmd, newMeRolesCommand(productionSeam{}))
	// My Actions (013): the `actions` leaf under `me`, a sibling of `roles`.
	// `me actions [--status <s>]` sends GET /me/actions and prints the actions the
	// practitioner's roles own, optionally filtered by status. (The 013 spec prose
	// says "my actions"; the implemented convention is `me actions` under `me`,
	// mirroring `me roles` — see .score/memory/LEARNINGS.md.) Reuses the same
	// productionSeam and the inherited --base-url.
	MustRegister(meCmd, newMeActionsCommand(productionSeam{}))
	// My Projects (014): the `projects` leaf under `me`, a sibling of `roles` and
	// `actions`. `me projects [--status <s>]` sends GET /me/projects and prints the
	// projects the practitioner's roles own, optionally filtered by status. (The
	// 014 spec prose says "my projects"; the implemented convention is `me projects`
	// under `me`, mirroring `me roles`/`me actions` — see .score/memory/LEARNINGS.md.)
	// Reuses the same productionSeam and the inherited --base-url; it adds no
	// --include flag (/me/projects offers no include — ADR-2).
	MustRegister(meCmd, newMeProjectsCommand(productionSeam{}))
	// Help & Version (003): tune the assembled root's help/version rendering —
	// unify --version with the version command, hide framework built-ins, keep
	// standard alphabetical listing. Applied after wiring so it configures the
	// final command set.
	configureHelpAndVersion(root)
	return root
}
