# Specification: Subrole Filler Roll-up

**Feature**: 051-subrole-filler-roll-up
**Role**: Definer
**Tier**: 1 (zero setup)

---

## System Overview

Subrole Filler Roll-up is the **one-level cross-role read for fillers** — it rolls up the actors filling an anchor role's **direct sub-roles** in one read: `glassfrog subrole-actors <role-id>` → `GET /roles/{id}/subroles/actors` → `listSubrolesActors`. It answers "who is staffing the circles inside this one?" — the view a circle lead or facilitator wants when a role is vacant, shared, or unfamiliar and they need to reach the surrounding circle, without fetching each child role's fillers one at a time.

The roll-up is **one level only** — the anchor's direct children, not a transitive closure of the whole subtree. The anchor must be an expanded role (a circle, `has_subroles: true`); the API returns `404` for a leaf role, which the command surfaces as a plain read failure rather than interpreting. It returns **actor** records (people and agents) — the same shape `GET /actors` returns — not the assignment relationship: this is the roll-up counterpart of Actor Directory (048), where Role Fillers (047) is the assignment-shaped read for a single role. It continues the subroles roll-up grammar Subroles Tension Roll-up (046) grew (`<resource> subroles <role-id>`), and it carries the same locally-validated `--kind human|agent` filter Actor Directory (048) established over the shared actor shape.

It sits on the proven read chain rather than rebuilding it: it hands requests to **Request Execution (010)**, reads identity through **Request Authentication (007)**, walks the list through **Pagination (016)**, lets **Output Format Selection (020)** render the result, and maps outcomes through **Exit-Code Convention (004)** and **API Error Extraction (015)**. It reuses the existing `glassfrog.Actor` model (grown in Identity Read (011), reused by Actor Directory (048)) rather than defining its own.

---

## Behavioral Accord

### Invocation

- When the user runs `glassfrog subrole-actors <role-id>`, the system reads the actors filling that anchor role's direct sub-roles and produces them as a list result.
- When the user omits the required `<role-id>`, passes more than one positional id, or passes an unknown flag, the system rejects the invocation as a usage error and calls no API.

### Filter

- When the user supplies `--kind <value>`, the system validates the value against the actor kind vocabulary (`human`, `agent`) before issuing any request; a supported value is sent as the `kind` query parameter, narrowing the roll-up to actors of that kind.
- When the user supplies a `--kind` value outside that vocabulary, the system rejects it as a usage error before issuing any request — naming the unsupported value and the supported set — and sends no request (mirroring how Actor Directory (048) validates `--kind`).
- When the user supplies no filter, the system requests every actor filling the direct sub-roles.

### Output

- When the read succeeds, the system produces the actor data as its result — each row carrying the filling actor's id, name, and kind — and lets Output Format Selection (020) render it in the effective format (`full` / `compact` / `json` / `yaml`); it neither fixes raw API JSON as its default nor defines its own format flag.
- When the anchor's sub-roles carry no fillers, or none match the supplied kind, the system produces an empty list result and exits successfully — an empty list is a valid answer, not an error.

### Completeness of the list

- When the rolled-up actors span more than one page, the system walks every page through Pagination (016) by default and produces the complete set.
- When the user supplies the first-page opt-out flag, the system makes a single page request and, if more pages exist, produces the first page flagged incomplete with a clear "more exist" signal — so even the opted-out result is never silently truncated.
- When the walk cannot complete (a page fails mid-walk), the system produces the actors gathered so far, flagged incomplete with the cause, so a partial list is never mistaken for the whole.

### Failure

- When no usable token is available, the system surfaces the authentication fail-safe's refusal and exits non-zero with a not-authenticated outcome, reusing the shared not-authenticated message and pointing the operator at how to store a credential.
- When the request cannot reach or complete at the wire (connection, DNS, TLS, timeout), the system surfaces the transport failure by name and exits non-zero with the network-unavailable outcome.
- When the API answers with a non-2xx response — including an anchor that is a leaf role (no sub-roles) or an unknown role id, both typically `404` — the system reports that the read failed, naming the HTTP status, and exits non-zero. It adds no interpretation of its own; the shared error handling (API Error Extraction, 015) classifies the status — a generic API-error outcome, or the permission (`401`/`403`) and rate-limit (`429`) outcomes it already distinguishes — and the command surfaces whichever results.
- Whatever the failure, the message names both what went wrong and a concrete next step (Action Transparency), and never includes the token.

---

## User Scenarios

**In order to** see who is staffing the circles inside this one when the role itself is vacant or shared,
**as a** practitioner facilitating a circle,
**I want to** roll up the actors filling a circle's direct sub-roles with one command.

**In order to** reach the surrounding circle without fetching each child role's fillers one at a time,
**as an** AI agent operating the CLI on a practitioner's behalf,
**I want to** read the sub-roles' fillers in a single roll-up by the anchor role's id.

**In order to** tell automation apart from people before I decide whom to contact,
**as an** AI agent assembling context,
**I want to** narrow the roll-up to just humans or just agents.

**In order to** trust I am seeing every actor staffing the sub-roles,
**as a** practitioner in a large circle,
**I want** the roll-up to walk to completion, or to tell me plainly when it is incomplete.

---

## Non-Behaviors

- The system must not roll up actors beyond the anchor's **direct** sub-roles. **Why**: the endpoint (`listSubrolesActors`) is a one-level roll-up, not a transitive closure of the whole subtree; recursing into grand-children would invent a behavior the API does not offer and silently change what "subroles" means.
- The system must not list the actors filling the anchor role **itself**. **Why**: that is the actors filling a single role — covered by Actor Directory's `actors --role-id <id>` (048) and the assignment-shaped Role Fillers `fillers <role-id>` (047), backed by different endpoints. This command surfaces only the children's fillers; merging the two would blur which level is being staffed.
- The system must not return the **assignment** relationship (its `focus` and `elected_until`). **Why**: the subroles endpoint returns bare **actor** records (cross-linked assignments excluded), the same shape `GET /actors` returns — not the `Assignment` resource. Projecting focus/election here would invent fields the response does not carry; the assignment-shaped read is Role Fillers (047), scoped to a single role.
- The system must not expose a separate `people` (or `agents`) subroles command. **Why**: the `/roles/{id}/subroles/people` endpoint is just an alias for `/roles/{id}/subroles/actors?kind=human`, and `--kind human|agent` selects either through the one endpoint with no capability lost — exactly the decision Actor Directory (048) made for `/people` vs `/actors`. A second command would fork the roll-up surface.
- The system must not special-case the leaf-role `404` into a "this role has no sub-roles" message or an empty-list success. **Why**: the API itself answers `404` for a leaf anchor; the CLI is a faithful surface (VISION Exclusion 1) and passes the status through the shared error handling, distinct from the genuine empty-list success returned when sub-roles exist but carry no fillers.
- The system must not create, invite, update, or remove actors or assignments. **Why**: this is a read surface alone, matching the read-only stance of the Actor Reads slice; staffing changes are actor administration, which PROJECT.md places out of scope.
- The system must not emit raw API JSON as a fixed default, nor define its own output-format flag. **Why**: Output Format Selection (020) resolves the format and dispatches to the 018/019 renderers; a private flag here would fork that contract.
- The system must not resolve the base URL or token, attach the `X-Auth-Token` header, decide the no-token fail-safe, type a non-2xx response, or choose its own exit codes. **Why**: Base URL Resolution (008), Credential Discovery (005), Request Authentication (007), API Error Extraction (015), and Exit-Code Convention (004) own those; a second path here would drift from their contracts.
- The system must not interpret, summarize, or advise on the actors it rolls up. **Why**: the CLI is a faithful API surface, not a Holacracy facilitator (VISION Exclusion 1); it produces the actor data and lets the operator reason about it.

---

## Integration Boundaries

- **Glassfrog API v5 (`GET /roles/{id}/subroles/actors`)**: the system reads the direct sub-roles' fillers through this endpoint. It accepts an optional `kind` (enum) query filter and is paginated (`{data: [Actor], meta: {pagination}}`). Data flows inbound (actors in, nothing written). A `404` means the anchor is a leaf role (no sub-roles) or the role id does not exist. When the endpoint is unreachable or answers non-2xx, the command surfaces a transport or read failure and exits non-zero.
- **Request Execution (010) / Request Authentication (007) / Pagination (016)**: the command builds the request and hands it to these seams to authenticate, execute, and walk pages; it does not re-implement transport, auth, or paging.
- **Output Format Selection (020) / Structured Serialization (018) / Templated Human Rendering (019)**: receives the rolled-up actor data as the command's result and renders it in the effective format. The command produces structured data, never pre-rendered output.
- **Exit-Code Convention (004) / API Error Extraction (015)**: the command maps the success and the non-2xx / transport / not-authenticated outcomes to exit codes and messages through these seams.
- **Subroles Tension Roll-up (046) — sibling**: established the one-level `<resource> subroles <role-id>` roll-up grammar, the leaf-role-`404`-as-read-failure stance, and the direct-children-only boundary this read follows.
- **Actor Directory (048) — sibling**: established the `glassfrog.Actor` model reuse, the locally-validated `--kind human|agent` filter, and the decision not to fork a separate `people`/`agents` command. 048 lists actors filling a single role (`GET /actors?role_id=`); this rolls them up across the anchor's sub-roles.

---

## Driving Scenarios

### Happy path

**Scenario: Roll up the actors filling a circle's direct sub-roles**
Given a valid stored credential and an anchor role whose direct sub-roles are filled by several actors
When the user runs `glassfrog subrole-actors <role-id>`
Then the system reads `GET /roles/{id}/subroles/actors`
And produces the sub-roles' fillers as a list result, each carrying its id, name, and kind
And exits successfully.

**Scenario: Narrow the roll-up to agents**
Given an anchor role whose sub-roles are filled by both people and agents
When the user runs `glassfrog subrole-actors <role-id> --kind agent`
Then the value is accepted as a supported kind
And the system sends `kind=agent` on `GET /roles/{id}/subroles/actors`
And produces only the agents as a list result.

**Scenario: Roll-up walks every page to completion**
Given an anchor role whose sub-role fillers span more than one page
When the user runs `glassfrog subrole-actors <role-id>`
Then the system walks every page through Pagination (016)
And produces the complete set of sub-role fillers
And exits successfully.

### Error scenarios

**Scenario: Anchor is a leaf role**
Given a valid stored credential and a role that has no sub-roles
When the user runs `glassfrog subrole-actors <role-id>`
Then the API answers `404`
And the system reports the read failed, naming the HTTP status
And exits non-zero
And adds no "this role has no sub-roles" interpretation of its own.

**Scenario: No usable credential**
Given no stored credential and none in the environment
When the user runs `glassfrog subrole-actors <role-id>`
Then the system surfaces the not-authenticated refusal without calling the API
And exits non-zero
And the message points the operator at how to store a credential.

### Edge cases

**Scenario: Sub-roles exist but carry no fillers**
Given a valid stored credential and an anchor whose sub-roles are unfilled
When the user runs `glassfrog subrole-actors <role-id>`
Then the system produces an empty list result
And exits successfully.

**Scenario: Unsupported kind value is rejected before any request**
Given a valid stored credential
When the user runs `glassfrog subrole-actors <role-id> --kind robot`
Then the system rejects it as a usage error, naming the unsupported value and the supported set (`agent`, `human`)
And issues no request
And exits with the usage-error code.

**Scenario: Paginated roll-up with first-page opt-out**
Given an anchor whose sub-role fillers span more than one page
When the user runs `glassfrog subrole-actors <role-id>` with the first-page opt-out flag
Then the system makes a single page request
And produces the first page flagged incomplete with a clear "more exist" signal.

---

## Validation Scenarios

> These are held out from the implementing agent for independent verification.

**Scenario: The roll-up is one level only**
Given an anchor role with direct sub-roles that themselves contain grand-child roles staffed by actors
When the user runs `glassfrog subrole-actors <role-id>`
Then only the direct sub-roles' fillers are read through `GET /roles/{id}/subroles/actors`
And the command makes no attempt to recurse into grand-child roles.

**Scenario: A leaf-role 404 is a failure, not an empty success**
Given a leaf anchor role for which the API answers `404`
When the command runs
Then the outcome is the shared non-2xx read failure naming the status and a non-zero exit
And it is distinct from the empty-list success returned when sub-roles exist but carry no fillers.

**Scenario: An unsupported kind costs no request**
Given a `--kind` value outside the actor kind vocabulary
When the command runs
Then the system rejects it as a usage error before assembling the connection or sending a request
And a transport tripwire confirms no request was issued.

**Scenario: The result is actor-shaped, not assignment-shaped**
Given a successful roll-up of an anchor whose sub-role fillers hold focuses and elected seats
When the result is rendered
Then each row carries the actor's id, name, and kind sourced from `GET /roles/{id}/subroles/actors`
And no `focus` or `elected_until` field is projected — that is Role Fillers' (047) assignment-shaped read.

**Scenario: Output is structured, not pre-rendered**
Given any successful roll-up read
When the result reaches Output Format Selection (020)
Then the command supplied structured actor data and defined no format flag of its own, so all four formats (`full` / `compact` / `json` / `yaml`) render from the same result.

---

## Assumptions

- **`Actor` model is reused, not redefined**: the `glassfrog.Actor` type grown in Identity Read (011) and reused by Actor Directory (048) already carries `id`, `name`, `kind`, and timestamps — the same schema this endpoint returns in each `data` element (the spec notes "Same response shape as `GET /actors`") — so no new leaf model is needed. The list returns the shared `{data: [...], meta: {pagination}}` envelope. (Reflects the per-list reuse pattern of 048.)
- **Kind vocabulary tracks the spec enum, validated locally**: `--kind` is validated against the actor kind set (`human`, `agent`) before any request — the same shape Actor Directory (048) validates `--kind` against, and the values `listSubrolesActors` accepts. The accepted set tracks the vendored spec (`spec/glassfrog-api-v5.yaml`); whether validation shares the 048 helper is a planning detail.
- **First-page opt-out flag is the shared one**: the roll-up reuses the same first-page opt-out flag and "more exist" signal established by the earlier list reads (016 / 025 / 026 / 033 / 034 / 038 / 046 / 048), not a new per-command flag. (Consistency across every list surface in the CLI.)
- **Command grammar is a distinct role-keyed leaf**: the command rolls up an actor-shaped result keyed on a role id. The exact spelling is pinned at the interface stage as the distinct top-level leaf `subrole-actors <role-id>` (plan ADR-1) — **not** an `actors subroles` subcommand, because `actors` is a runnable, positional-bearing command and hosting a subcommand under it would force a runnable-parent-with-children shape the codebase avoids. The behavior — one role-scoped, read-only, one-level roll-up of the actors filling an anchor's direct sub-roles, `--kind`-filterable — is fixed regardless of the surface spelling.
- **[ASSUMED] Role-id format is not validated client-side**: the read requires exactly one positional id but lets the API reject an unknown or malformed id (typically `404`), rather than enforcing the `^role_…` pattern locally. (Mirrors how Subroles Tension Roll-up (046), Role Fillers (047), and Actor Directory (048) leave id-shape validation to the server.)

---

## Ambiguity Warnings

_None — the feature is the actor-shaped twin of Subroles Tension Roll-up (046) over a single new endpoint (`listSubrolesActors`), and every design call has a settled sibling precedent: the one-level / leaf-`404`-as-read-failure / direct-children-only stance from 046, and the `glassfrog.Actor` reuse, the locally-validated `--kind human|agent` filter, and the no-separate-`people`-command decision from Actor Directory (048). The two boundary questions specific to this endpoint were resolved during specification: (1) it returns bare **actor** records (the `/subroles/actors` shape, cross-linked assignments excluded), not the assignment-shaped read of Role Fillers (047); and (2) the `/subroles/people` alias is reached through `--kind human`, not a forked command. The exact command spelling is pinned at the interface stage (as in 046/047/048) — for this feature, the distinct top-level leaf `subrole-actors <role-id>` (plan ADR-1)._
