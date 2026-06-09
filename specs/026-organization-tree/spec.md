# Specification: Organization Tree

**Feature**: 026-organization-tree
**Role**: Definer
**Tier**: 1 (zero setup)

---

## System Overview

Organization Tree is the **circle-hierarchy read** in the Governance Reads slice. It answers "how is this organization structured?" by reading the role tree three ways: the whole org as one nested tree (`glassfrog tree` → `GET /tree` → `getOrgTree`), the subtree rooted at any role (`glassfrog tree <role-id>` → `GET /roles/{id}/tree` → `getRoleTree`), and the immediate children of a role (`glassfrog subroles <role-id>` → `GET /roles/{id}/subroles` → `listSubroles`). Where Role Reads (025) answers "what roles exist and what is *this* one", Organization Tree answers "what *contains* what" — the nesting that makes a circle a circle.

It sits on the proven read chain rather than rebuilding it: it hands requests to **Request Execution (010)**, reads identity through **Request Authentication (007)**, lets **Output Format Selection (020)** render the result, and maps outcomes through **Exit-Code Convention (004)**. Two completeness models live side by side because the API draws them that way: the tree reads return the **full nested tree in a single, unpaginated response** (bounded only by an optional `--depth`), while the subroles read is a **paginated flat list** that walks to completion like every other list in the CLI. The capability owns the addressable subroles read that Role Reads (025) deliberately deferred to it.

---

## Behavioral Accord

### Invocation

- When the user runs the whole-org tree read with no positional id, the system reads the organization's full role tree and produces it as a nested tree result.
- When the user runs the role-rooted tree read with a role id, the system reads the subtree rooted at (and including) that role and produces it as a nested tree result.
- When the user runs the subroles read with a role id, the system reads that role's immediate child roles and produces them as a list result.
- When the user passes an unknown flag, or more than one positional id, or omits the required role id on a read that needs one, the system rejects the invocation as a usage error and calls no API.

### Depth (tree reads)

- When the user supplies `--depth <n>` on either tree read, the system sends it as the `depth` query parameter, bounding the descendants returned (`0` = the root node alone, `1` = root plus direct children, and so on).
- When the user supplies no `--depth`, the system requests the full subtree.
- When the user supplies `--depth` on the subroles read, the system rejects the invocation as a usage error — depth bounds a recursive tree, and subroles returns only one level.

### Include (related resources per node)

- When the user supplies `--include` on a tree read with one or more of `accountabilities`, `domains`, `members`, the system sends them as the `?include=` query and the returned nodes carry those related resources inline.
- When the user supplies `--include` on the subroles read with one or more of `assignments`, `subroles`, `parent_role`, `policies`, `notes`, `skills`, the system sends them as the `?include=` query and embeds them inline on each child role.
- When the user supplies `--include` with a value outside the supported set **for the read being run**, the system rejects it as a usage error before issuing any request, naming the unsupported value and that read's supported set — even though the tree endpoints would silently ignore it server-side. The two reads validate against their own distinct sets.

### Output

- When a read succeeds, the system produces the tree or list data as its result and lets Output Format Selection (020) render it in the effective format (`full` / `compact` / `json` / `yaml`); it neither fixes raw API JSON as its default nor defines its own format flag.
- When a tree read returns a root role with no children (a leaf, or `--depth 0`), the system produces a single-node tree result and exits successfully — an empty `children` set is a valid answer, not an error.
- When a node reports it has subroles but its children are not present in the result — because `--depth` capped beneath it, or the API withheld them — the system marks the node as having subroles below the returned tree, distinct from a true leaf (a node that reports no subroles). It uses only the has-subroles signal the API provides and never invents a count of the omitted descendants — so a depth-bounded tree always shows where it stops and a capped branch is never mistaken for an empty one.
- When the subroles read returns no children (a leaf role), the system produces an empty list result and exits successfully.

### Completeness of the subroles list

- When a role's subroles span more than one page, the system walks every page through **Pagination (016)** by default and produces the complete set.
- When the user supplies the first-page opt-out flag, the system makes a single page request and, if more pages exist, produces the first page flagged incomplete with a clear "more exist" signal — so even the opted-out result is never silently truncated.
- When the walk cannot complete (a page fails mid-walk), the system produces the children gathered so far, flagged incomplete with the cause, so a partial list is never mistaken for the whole.

### Failure

- When no usable token is available, the system surfaces the authentication fail-safe's refusal and exits non-zero with a not-authenticated outcome, reusing the shared not-authenticated message and pointing the operator at how to store a credential.
- When the request cannot reach or complete at the wire (connection, DNS, TLS, timeout), the system surfaces the transport failure by name and exits non-zero with the network-unavailable outcome.
- When the API answers with a non-2xx response — including a tree or subroles read whose role id does not exist (typically `404`) — the system reports that the read failed, naming the HTTP status, and exits non-zero. The command adds no interpretation of its own; the shared error handling (API Error Extraction, 015) classifies the status, and the command surfaces whichever outcome results.
- Whatever the failure, the message names both what went wrong and a concrete next step (Action Transparency), and never includes the token.

---

## User Scenarios

**In order to** orient myself to an organization's whole governance structure in one call,
**as an** AI agent operating the CLI on a practitioner's behalf,
**I want to** read the entire role tree as a single nested document.

**In order to** focus on one branch of the org without pulling the whole tree,
**as a** practitioner exploring a particular circle,
**I want to** read the subtree rooted at a specific role, optionally bounding how deep it goes.

**In order to** see exactly what sits directly inside a circle,
**as an** AI agent navigating the hierarchy step by step,
**I want to** list a role's immediate subroles and trust the list is complete or plainly flagged incomplete.

**In order to** keep a large tree response manageable,
**as an** AI agent with a bounded context,
**I want to** cap the tree depth so I read only as far down as I need.

---

## Non-Behaviors

- The system must not paginate the tree reads or invent a depth-walk that stitches the tree together from multiple calls. **Why**: `GET /tree` and `GET /roles/{id}/tree` return the full subtree in one unpaginated response by design; `--depth` is the API's own size control. Faking pagination here would fork the contract and risk an inconsistent, partially-assembled tree.
- The system must not manage an ETag cache or send `If-None-Match` to obtain `304 Not Modified`. **Why**: conditional-request caching is a latency optimization, not part of a faithful first read surface; the CLI issues a plain read each time and leaves cache coordination out of scope until there is demand for it.
- The system must not pass unknown `--include` values through to the API on the strength of the tree endpoints' silent-ignore behavior, nor share one include set across the two reads. **Why**: silent-ignore hides typos and makes the supported set undiscoverable; CLI-side reject-unknown (per 011/025 and the project's "validate closed-enum inputs locally" stance) catches the mistake before any call. The tree and subroles reads expose genuinely different related-resource sets, so a single shared set would either reject valid values or accept impossible ones.
- The system must not emit raw API JSON as a fixed default, nor define its own output-format flag. **Why**: Output Format Selection (020) resolves the format and dispatches to the 018/019 renderers; a private flag here would fork that contract.
- The system must not resolve the base URL or token, attach the `X-Auth-Token` header, decide the no-token fail-safe, type a non-2xx response, or choose its own exit codes. **Why**: Base URL Resolution (008), Credential Discovery (005), Request Authentication (007), API Error Extraction (015), and Exit-Code Convention (004) own those; a second path here would drift from their contracts.
- The system must not write to or mutate any role, nor offer a way to restructure the tree. **Why**: Organization Tree is a read surface; the governance structure changes only through Proposals, reflecting the read-only stance of all governance reads.

---

## Integration Boundaries

- **Glassfrog API v5 (`GET /tree`, `GET /roles/{id}/tree`, `GET /roles/{id}/subroles`)**: the system reads the hierarchy through these endpoints. Data flows inbound (tree/roles in, nothing written). When an endpoint is unreachable or answers non-2xx, the command surfaces a transport or read failure and exits non-zero.
- **Request Execution (010)**: the seam the command hands each request to and reads the outcome from (success-with-body, non-2xx, transport error, decode error).
- **Request Authentication (007)**: supplies the authenticated transport and the no-token fail-safe whose refusal the command propagates.
- **Pagination (016)**: the walker the **subroles** read uses to assemble the complete set across pages (or a flagged-incomplete partial set when a page fails). The tree reads do not use it.
- **Output Format Selection (020)**: resolves the effective `--output` format and dispatches the produced result to the matching renderer; the command produces result data, not presentation.
- **Exit-Code Convention (004)**: the command maps its outcome to a process exit code through the established convention.
- **User / AI agent (stdout/stderr)**: the rendered result is written to stdout on success; failure messages are written to stderr.

---

## Driving Scenarios

### Happy path

**Scenario: Read the whole organization tree**
Given a valid token resolving to a member of an organization with a role hierarchy
When the user runs the whole-org tree read with no positional id
Then the system produces a nested tree result rooted at the org's anchor role
And the command exits successfully

**Scenario: Read the subtree rooted at a role**
Given a valid token and a role id that exists in the org
When the user runs the role-rooted tree read with that id
Then the system produces a nested tree result rooted at (and including) that role
And the command exits successfully

**Scenario: List a role's immediate subroles**
Given a valid token and a circle role id that has child roles
When the user runs the subroles read with that id
Then the system produces a list result of the role's direct children
And the command exits successfully

**Scenario: Bound the tree depth**
Given a valid token and an org with a deep hierarchy
When the user runs the whole-org tree read with `--depth 1`
Then the system sends `depth=1` and produces the anchor role plus only its direct children
And the command exits successfully

**Scenario: Read a tree with related resources on each node**
Given a valid token and an existing role id
When the user runs the role-rooted tree read with `--include accountabilities,domains`
Then the system sends `include=accountabilities,domains` and the returned nodes carry those resources inline
And the command exits successfully

**Scenario: List subroles with related resources embedded**
Given a valid token and a circle role id with child roles
When the user runs the subroles read with `--include assignments,policies`
Then the system sends `include=assignments,policies` and embeds those resources inline on each child role
And the command exits successfully

### Error scenarios

**Scenario: No usable token**
Given no usable token is available to the CLI
When the user runs any Organization Tree read
Then the system surfaces the authentication fail-safe's refusal as a not-authenticated outcome
And the command exits non-zero, pointing the operator at how to store a credential
And no tree data is produced

**Scenario: A tree read for an unknown role id**
Given a valid token but a role id that does not exist
When the user runs the role-rooted tree read with that id
Then the system reports that the read failed and names the HTTP status
And the command exits with the API-error code

**Scenario: An unsupported `--include` value is rejected without an API call**
Given a valid token and an existing role id
When the user runs the tree read with `--include nonsense`
Then the system rejects the invocation as a usage error, naming the unsupported value and the tree read's supported set
And no API call is made

### Edge cases

**Scenario: A leaf role's tree is a single node**
Given a valid token and a role id with no child roles
When the user runs the role-rooted tree read with that id
Then the system produces a single-node tree result with an empty children set
And the command exits successfully

**Scenario: A leaf role has no subroles**
Given a valid token and a leaf role id
When the user runs the subroles read with that id
Then the system produces an empty list result
And the command exits successfully

**Scenario: Subroles span more than one page (default walk to completion)**
Given a role whose subroles span more than one page of the API response
When the user runs the subroles read with that id
Then the system walks every page through Pagination (016) and produces the complete set
And the command exits successfully

**Scenario: `--depth` is rejected on the subroles read**
Given a valid token and an existing role id
When the user runs the subroles read with `--depth 2`
Then the system rejects the invocation as a usage error
And no API call is made

**Scenario: A depth-capped node is marked as having more below**
Given a valid token and an org whose hierarchy is deeper than one level
When the user runs the whole-org tree read with `--depth 1`
Then a direct child that itself contains subroles is marked as having subroles below the returned tree
And it is distinguishable from a true leaf, which is marked as having none

**Scenario: First-page opt-out on subroles stops at one page and signals more exist**
Given a circle role whose subroles span more than one page of the API response
When the user runs the subroles read with `--first-page`
Then the system makes a single page request and produces the first page of child roles
And it surfaces a clear "more exist" incomplete signal
And the command exits successfully

---

## Validation Scenarios

> These are held out from the implementing agent for independent verification.

**Scenario: Tree default output carries no raw API envelope**
Given a successful whole-org tree run under the default human format
When the output is inspected
Then it shows the reshaped nested projection only
And it does not contain the raw `data` JSON envelope

**Scenario: The CLI rejects unknown includes the API would have silently ignored**
Given a tree read invoked with an `--include` value outside `accountabilities,domains,members`
When the invocation is processed
Then it is rejected as a usage error before any request is issued
And no request with a silently-ignored include is sent

**Scenario: Subroles incompleteness is never silent**
Given a subroles run where Pagination (016) could not assemble every page
When the result is inspected
Then an explicit incomplete signal with its cause is present
And the partial list cannot be read as the complete set

**Scenario: A depth-capped node is distinguishable from a leaf**
Given a depth-bounded tree result where one boundary node has subroles below the cut and another is a true leaf
When the result is inspected
Then the boundary node is marked as having subroles below the returned tree
And the leaf is marked as having none
And neither carries an invented count of omitted descendants

---

## Assumptions

- **Command spelling** (`glassfrog tree`, `glassfrog tree <id>`, `glassfrog subroles <id>`): assumed from the FEATURE-MODEL "Organization Tree" framing and 025's note that these reads cannot be children of `roles` (a positional id forecloses subcommands). The exact command surface — separate `tree` / `subroles` leaves vs. a `tree --subroles` style flag — syncs with the CLI command convention at interface time. `[ASSUMED]`
- **`--include` reject-unknown semantics**: assumed Organization Tree reuses Identity Read (011) / Role Reads (025)'s opt-in, reject-unknown `--include` handling, validating each read against its own supported set. (Informed by specs 011 and 025 and the DECISIONS "validate closed-enum inputs locally" note.)
- **`--depth` flag**: the *behavior* (bound the tree; `0` = root only; omit = full subtree) is fixed and maps to the API's `depth` query parameter; the exact flag spelling syncs with the CLI flag convention at interface time. `[ASSUMED]`
- **Subroles completeness** reuses Role Reads (025)'s model: walk by default, first-page opt-out that signals incompleteness, partial-on-mid-walk-failure. The exact opt-out flag name is shared with 025 at interface time. (Informed by spec 025 and CONSTITUTION VI.)
- **Output rendering** is delegated to Output Format Selection (020); the built-in default format is `full` (020's default). (Informed by spec 020.)
- **Failure-to-exit-code mapping** reuses Request Execution's typed outcomes and the Exit-Code Convention (004) mapping rather than defining new codes. (Informed by specs 004 and 010.)

---

## Ambiguity Warnings

*None outstanding.* (Scope, `--include` validation, ETag/304 caching, and depth control were all resolved during the define conversation; the depth-boundary signal and subroles scenario coverage were resolved in the Clarifications below.)

---

## Clarifications

### Session 2026-06-09

- **Depth-boundary signalling**: When `--depth` caps a tree (or the API withholds children), a node that reports it has subroles but whose children are not in the result is marked as having subroles below the returned tree — distinct from a true leaf, which reports none. The system uses only the has-subroles signal the API provides and never invents a count of the omitted descendants. This closes the silent-truncation-at-the-boundary gap (CONSTITUTION VI): a depth-bounded tree always shows where it stops. (Surfaced by the pre-implementation guard: checklist P1 / risk H-3.)
- **Subroles scenario coverage**: The `subroles --include` behavior (embed related resources on each child; reject an unknown value before any request, against the subroles set) and the `subroles --first-page` opt-out (single page + "more exist" signal) were already in the Behavioral Accord but had no driving scenarios. Driving scenarios were added for both; subroles `--include` reject-unknown uses the same per-read local validation already exercised by the tree read's "unsupported include" scenario. (Surfaced by the guard: analyze K5 coverage gap.)
