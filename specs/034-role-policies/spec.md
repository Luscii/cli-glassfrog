# Specification: Role Policies

**Feature**: 034-role-policies
**Role**: Definer
**Tier**: 1 (zero setup)

---

## System Overview

Role Policies is the **policy read surface** in the Governance Reads slice. It offers two reads: list the policies governing a role's interior (`glassfrog policies <role-id>` → `GET /roles/{id}/policies` → `listRolePolicies`) and read a single policy by its id (`glassfrog policy <pol-id>` → `GET /policies/{id}` → `getPolicy`). A policy is a governance rule on a role's interior — a first-class governance element alongside accountabilities and domains — so this capability gives an agent or practitioner an addressable way to find the policies on any role and to fetch the full body of any one of them.

It is the per-role/standalone counterpart that **Role Reads (025) deliberately deferred to it**: 025 embeds policies *inline on a role* as a convenience view (`roles <id> --include=policies`), but the addressable standalone reads — each with its own projection — belong here. It sits on the proven read chain rather than rebuilding it: it hands requests to **Request Execution (010)**, reads identity through **Request Authentication (007)**, walks the list through **Pagination (016)**, lets **Output Format Selection (020)** render the result, and maps outcomes through **Exit-Code Convention (004)** and **API Error Extraction (015)**. It reuses the existing `glassfrog.Policy` model grown in 025 rather than defining its own.

---

## Behavioral Accord

### Invocation

- When the user runs `glassfrog policies <role-id>`, the system reads the policies governing that role's interior and produces them as a list result.
- When the user runs `glassfrog policy <pol-id>`, the system reads the single policy with that id and produces it as a single-policy result.
- When the user omits the required id on either command, passes more than one positional id, or passes an unknown flag, the system rejects the invocation as a usage error and calls no API.

### Search filter (list)

- When the user supplies `--query <text>` (or `-q <text>`) on `policies <role-id>`, the system sends it as the `q` query parameter, narrowing the list server-side to policies matching the free-text search; the role-id path scope still applies.
- When the user supplies no `--query`, the system requests every policy governing the role.
- When the user supplies `--query` on the single read (`policy <pol-id>`), the system rejects the invocation as a usage error — the search filter applies only to the per-role list.

### Output

- When a read succeeds, the system produces the policy data as its result and lets Output Format Selection (020) render it in the effective format (`full` / `compact` / `json` / `yaml`); it neither fixes raw API JSON as its default nor defines its own format flag.
- When the role has no policies, or no policy matches the supplied `--query`, the system produces an empty list result and exits successfully — an empty list is a valid answer, not an error.

### Completeness of the list

- When a role's policies span more than one page, the system walks every page through Pagination (016) by default and produces the complete set.
- When the user supplies the first-page opt-out flag, the system makes a single page request and, if more pages exist, produces the first page flagged incomplete with a clear "more exist" signal — so even the opted-out result is never silently truncated.
- When the walk cannot complete (a page fails mid-walk), the system produces the policies gathered so far, flagged incomplete with the cause, so a partial list is never mistaken for the whole.

### Failure

- When no usable token is available, the system surfaces the authentication fail-safe's refusal and exits non-zero with a not-authenticated outcome, reusing the shared not-authenticated message and pointing the operator at how to store a credential.
- When the request cannot reach or complete at the wire (connection, DNS, TLS, timeout), the system surfaces the transport failure by name and exits non-zero with the network-unavailable outcome.
- When the API answers with a non-2xx response — including a read whose role id or policy id does not exist (typically `404`) — the system reports that the read failed, naming the HTTP status, and exits non-zero. The command adds no interpretation of its own; the shared error handling (API Error Extraction, 015) classifies the status — a generic API-error outcome, or the permission (`401`/`403`) and rate-limit (`429`) outcomes it already distinguishes — and the command surfaces whichever results.
- Whatever the failure, the message names both what went wrong and a concrete next step (Action Transparency), and never includes the token.

---

## User Scenarios

**In order to** see every governance rule on a role before I act inside it,
**as an** AI agent operating the CLI on a practitioner's behalf,
**I want to** list the policies governing any role by its id with one command.

**In order to** read the full text of a specific policy I found referenced elsewhere,
**as a** practitioner reviewing the governance around my work,
**I want to** fetch a single policy by its id and see its full body.

**In order to** find the relevant policy in a circle that has many,
**as an** AI agent assembling context,
**I want to** narrow the role's policy list with a free-text search.

**In order to** trust I am seeing every policy on a role,
**as a** practitioner in a large circle,
**I want** the list to walk to completion, or to tell me plainly when it is incomplete.

---

## Non-Behaviors

- The system must not split the two reads into one command with an optional positional id the way `roles` does. **Why**: the two reads take *different* id kinds (a `role_` id selects a per-role list; a `pol_` id selects one policy) with different meanings, and cobra cannot tell a role id from a subcommand name (the constraint that already kept the per-role reads off `roles <id> …`). Two distinct commands — plural `policies <role-id>` for the role-scoped list, singular `policy <pol-id>` for the standalone read — keep each invocation unambiguous.
- The system must not embed policies inline on a role result, nor accept a `role-id` on the single read to "find its policies". **Why**: the embedded-on-the-role view is Role Reads' `--include=policies` (025); this capability owns only the *addressable* reads. Conflating them would fork the projection 025 already provides.
- The system must not provide standalone reads of other related resources (domains, projects, notes, skills). **Why**: each is its own capability — Role Domains (#33), Role Projects (#38), Organization Tree subroles (#26). Role Policies owns the policy surface alone.
- The system must not emit raw API JSON as a fixed default, nor define its own output-format flag. **Why**: Output Format Selection (020) resolves the format and dispatches to the 018/019 renderers; a private flag here would fork that contract.
- The system must not resolve the base URL or token, attach the `X-Auth-Token` header, decide the no-token fail-safe, type a non-2xx response, or choose its own exit codes. **Why**: Base URL Resolution (008), Credential Discovery (005), Request Authentication (007), API Error Extraction (015), and Exit-Code Convention (004) own those; a second path here would drift from their contracts.
- The system must not write to or mutate any policy or role. **Why**: Role Policies is a read surface; governance changes are made only through Proposals, which are out of scope here and reflect the read-only stance of the governance reads.

---

## Integration Boundaries

- **Glassfrog API v5 (`GET /roles/{id}/policies`, `GET /policies/{id}`)**: the system reads policies through these endpoints. Data flows inbound (policies in, nothing written). When an endpoint is unreachable or answers non-2xx, the command surfaces a transport or read failure and exits non-zero.
- **Request Execution (010) / Request Authentication (007) / Pagination (016)**: the command builds the request and hands it to these seams to authenticate, execute, and (for the list) walk pages; it does not re-implement transport, auth, or paging.
- **Output Format Selection (020)**: receives the policy data as the command's result and renders it in the effective format. The command produces structured data, never pre-rendered output.

---

## Driving Scenarios

### Happy path

**Scenario: List the policies on a role**
Given a valid stored credential and a role id that has two policies
When the user runs `glassfrog policies <role-id>`
Then the system reads `GET /roles/{id}/policies`
And produces both policies as a list result
And exits successfully.

**Scenario: Read a single policy with its full body**
Given a valid stored credential and an existing policy id
When the user runs `glassfrog policy <pol-id>`
Then the system reads `GET /policies/{id}`
And produces the single policy including its full body
And exits successfully.

**Scenario: Narrow a role's policies with a search**
Given a role whose policies include one titled "All PRs require two approvals"
When the user runs `glassfrog policies <role-id> --query approvals`
Then the system sends `q=approvals` on `GET /roles/{id}/policies`
And produces only the matching policies as a list result.

### Error scenarios

**Scenario: Policy id does not exist**
Given a valid stored credential and a `pol_` id that no policy has
When the user runs `glassfrog policy <pol-id>`
Then the API answers `404`
And the system reports the read failed, naming the HTTP status
And exits non-zero.

**Scenario: No usable credential**
Given no stored credential and none in the environment
When the user runs `glassfrog policies <role-id>`
Then the system surfaces the not-authenticated refusal without calling the API
And exits non-zero
And the message points the operator at how to store a credential.

### Edge cases

**Scenario: Role has no policies**
Given a valid stored credential and a role with no policies
When the user runs `glassfrog policies <role-id>`
Then the system produces an empty list result
And exits successfully.

**Scenario: Search flag on the single read is rejected**
Given a valid stored credential
When the user runs `glassfrog policy <pol-id> --query approvals`
Then the system rejects the invocation as a usage error before calling the API
And exits with the usage-error code.

**Scenario: Paginated list with first-page opt-out**
Given a role whose policies span more than one page
When the user runs `glassfrog policies <role-id>` with the first-page opt-out flag
Then the system makes a single page request
And produces the first page flagged incomplete with a clear "more exist" signal.

---

## Validation Scenarios

> These are held out from the implementing agent for independent verification.

**Scenario: The two commands never collide on id kind**
Given the policy reads are exposed as `policies <role-id>` and `policy <pol-id>`
When a `role_` id is passed to `policy` (or a `pol_` id to `policies`)
Then the system either reports the API's `404`/`400` for the wrong id kind, or rejects it — never silently reads the wrong resource — so the plural/singular split stays unambiguous in practice.

**Scenario: Output is structured, not pre-rendered**
Given any successful policy read
When the result reaches Output Format Selection (020)
Then the command supplied structured policy data and defined no format flag of its own, so all four formats (`full` / `compact` / `json` / `yaml`) render from the same result.

---

## Assumptions

- **First-page opt-out flag is the shared one**: the list reuses the same first-page opt-out flag and "more exist" signal established by the earlier list reads (016 / 025 / 026), not a new per-command flag. (Consistency across every list surface in the CLI.)
- **`Policy` model is reused, not redefined**: the `glassfrog.Policy` type grown in 025 already carries `id`, `title`, `body`, `role_id`, `domain_id`, and timestamps — the same schema both endpoints return — so no new leaf model is needed. (Reflects the 025 decision that the per-role specs reuse `Policy`.)
- **Body present on both reads**: the spec schema marks `body` required on the `Policy` returned by both the list and the single read, so the list is not a body-less summary projection. The renderers (019) decide how much body to show per format; the command does not truncate. ([ASSUMED] — if the live API returns body-less list items despite the schema, the render guards already mark explicit absence rather than inventing a value.)

---

## Ambiguity Warnings

_None — the feature's behavior follows the established 025/026 read pattern, and the two open boundary questions (command surface and the `--query` filter) were resolved during specification._
