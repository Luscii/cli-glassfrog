# Specification: Role Reads

**Feature**: 025-role-reads
**Role**: Definer
**Tier**: 1 (zero setup)

---

## System Overview

Role Reads is the **org-wide role surface** in the Governance Reads slice. It offers two reads: list every role in the organization (`glassfrog roles` → `GET /roles` → `listRoles`) and read a single role by id (`glassfrog roles <id>` → `GET /roles/{id}` → `getRole`). Where the token-scoped My Roles (012) answers "what do *I* hold here" without anyone naming themselves, Role Reads answers "what roles exist in this org, and what is *this* one" — and it deliberately accepts a person / parent / tag selector, because that org-wide reach is its reason to exist.

It is the **dependency root of Governance Reads**: the per-role reads — Role Domains (#33), Role Policies (#34), Role Projects (#38) — take a *required* role-id path parameter, and the ids they consume come from here. It sits on the proven read chain rather than re-building it: it hands requests to **Request Execution (010)**, reads identity through **Request Authentication (007)**, walks the list through **Pagination (016)**, and lets **Output Format Selection (020)** render the result. Its own job is small: build the `roles` (list) or `roles <id>` (single) request, apply the caller's filters or `--include`, ask the seams to fetch, and produce the role data as its result — never the raw nested payload by default.

---

## Behavioral Accord

### Invocation

- When the user runs `glassfrog roles` with no positional id, the system reads the organization's roles and produces them as a list result.
- When the user runs `glassfrog roles <id>`, the system reads the single role with that id and produces it as a single-role result.
- When the user passes an unknown flag, or more than one positional id, the system rejects the invocation as a usage error and calls no API.

### List filters

- When the user supplies `--parent <role-id>`, `--person <person-id>`, `--has-subroles`, or `--tag <tag>`, the system sends each as the matching `GET /roles` query parameter (`parent_role_id`, `person_id`, `has_subroles`, `tag`), narrowing the list server-side; multiple filters combine.
- When the user supplies no filter, the system requests every role in the organization.
- When the user supplies any list filter together with a positional id (the single read), the system rejects the invocation as a usage error — filters apply only to the list.

### Related resources on the single read

- When the user supplies `--include` on `roles <id>` with one or more of the supported related resources — `assignments`, `subroles`, `parent_role`, `policies`, `notes`, `skills` — the system sends them as the `?include=` query and embeds the returned related resources inline in the role result.
- When the user supplies `--include` with a value outside that set, the system rejects it as a usage error before issuing any request, naming the unsupported value and the supported set — mirroring how Identity Read (011) validates its own `--include`.
- When the user supplies no `--include`, the system reads the role with only the always-inline accountabilities, domains, and fillers.
- When the user supplies `--include` on the list (no id), the system rejects it as a usage error — related-resource embedding is a single-role concern.

### Output

- When a read succeeds, the system produces the role data as its result and lets Output Format Selection (020) render it in the effective format (`full` / `compact` / `json` / `yaml`); it neither fixes raw API JSON as its default nor defines its own format flag.
- When the org has no roles, or no role matches the supplied filters, the system produces an empty list result and exits successfully — an empty list is a valid answer, not an error.

### Completeness of the list

- When the organization's roles span more than one page, the system walks every page through Pagination (016) by default and produces the complete set.
- When the user supplies the first-page opt-out flag, the system makes a single page request and, if more pages exist, produces the first page flagged incomplete with a clear "more exist" signal (the *signal-the-boundary* behavior the self-service reads use) — so even the opted-out result is never silently truncated.
- When the walk cannot complete (a page fails mid-walk), the system produces the roles gathered so far, flagged incomplete with the cause, so a partial list is never mistaken for the whole.

### Failure

- When no usable token is available, the system surfaces the authentication fail-safe's refusal and exits non-zero with a not-authenticated outcome, reusing the shared not-authenticated message and pointing the operator at how to store a credential.
- When the request cannot reach or complete at the wire (connection, DNS, TLS, timeout), the system surfaces the transport failure by name and exits non-zero with the network-unavailable outcome.
- When the API answers with a non-2xx response — including a single read whose id does not exist (typically `404`) — the system reports that the read failed, naming the HTTP status, and exits non-zero with the generic API-error outcome; it does not interpret which kind of API error it was.
- Whatever the failure, the message names both what went wrong and a concrete next step (Action Transparency), and never includes the token.

---

## User Scenarios

**In order to** navigate the organization's governance structure and drill into any role I find,
**as an** AI agent operating the CLI on a practitioner's behalf,
**I want to** list every role and read one by its id with one command each.

**In order to** find the roles under a particular circle, held by a particular person, or carrying a tag,
**as a** practitioner exploring the org,
**I want to** filter the role list by parent, person, tag, or whether a role has subroles.

**In order to** see a role together with its related resources in a single call,
**as an** AI agent assembling context before acting,
**I want to** request a single role with its assignments, subroles, parent, policies, notes, or skills embedded inline.

**In order to** trust that I am seeing every role in the org,
**as a** practitioner in a large organization,
**I want** the list to walk to completion, or to tell me plainly when it is incomplete.

---

## Non-Behaviors

- The system must not scope the list to the token's own roles, nor drop the person / parent / tag selector. **Why**: that is My Roles (012), the token-scoped self-service read; Role Reads is the org-wide surface, and conflating the two removes the org view's reason to exist.
- The system must not provide *standalone* reads of related resources (`GET /policies/{id}`, `GET /domains/{id}`, a standalone subroles or projects listing). **Why**: `--include` here embeds related resources *inline on the role* as a convenience view; the addressable standalone reads, each with their own projection, belong to Role Policies (#34), Role Domains (#33), Organization Tree subroles (#26), and Role Projects (#38). The two views coexist deliberately — embedded-on-the-role vs. addressable-on-its-own — and Role Reads owns only the former.
- The system must not emit raw API JSON as a fixed default, nor define its own output-format flag. **Why**: Output Format Selection (020) resolves the format and dispatches to the 018/019 renderers; a private flag here would fork that contract.
- The system must not resolve the base URL or token, attach the `X-Auth-Token` header, decide the no-token fail-safe, type a non-2xx response, or choose its own exit codes. **Why**: Base URL Resolution (008), Credential Discovery (005), Request Authentication (007), API Error Extraction (015), and Exit-Code Convention (004) own those; a second path here would drift from their contracts.
- The system must not write to or mutate any role. **Why**: Role Reads is a read surface; governance changes are made only through Proposals, which are out of scope here and reflect the read-only stance of the governance reads.

---

## Integration Boundaries

- **Glassfrog API v5 (`GET /roles`, `GET /roles/{id}`)**: the system reads roles through these endpoints. Data flows inbound (roles in, nothing written). When an endpoint is unreachable or answers non-2xx, the command surfaces a transport or read failure and exits non-zero.
- **Request Execution (010)**: the seam the command hands each request to and reads the outcome from (success-with-body, non-2xx, transport error, decode error).
- **Request Authentication (007)**: supplies the authenticated transport and the no-token fail-safe whose refusal the command propagates.
- **Pagination (016)**: the walker the list read uses to assemble the complete set across pages (or a flagged-incomplete partial set when a page fails).
- **Output Format Selection (020)**: resolves the effective `--output` format and dispatches the produced result to the matching renderer; the command produces result data, not presentation.
- **Exit-Code Convention (004)**: the command maps its outcome to a process exit code through the established convention.
- **User / AI agent (stdout/stderr)**: the rendered result is written to stdout on success; failure messages are written to stderr.

---

## Driving Scenarios

### Happy path

**Scenario: List the organization's roles**
Given a valid token resolving to a member of an organization with several roles
When the user runs `glassfrog roles`
Then the system produces a list result of the org's roles
And the command exits successfully

**Scenario: Read a single role by id**
Given a valid token and a role id that exists in the org
When the user runs `glassfrog roles <id>`
Then the system produces a single-role result carrying that role's name, purpose, accountabilities, domains, and fillers
And the command exits successfully

**Scenario: Filter the list by parent circle**
Given a valid token and a parent role id
When the user runs `glassfrog roles --parent <parent-id>`
Then the system sends `parent_role_id` on the request and produces only the roles under that parent
And the command exits successfully

**Scenario: Read a single role with related resources embedded**
Given a valid token and an existing role id
When the user runs `glassfrog roles <id> --include policies,subroles`
Then the system sends `include=policies,subroles` and embeds those related resources inline in the role result
And the command exits successfully

### Error scenarios

**Scenario: No usable token**
Given no usable token is available to the CLI
When the user runs `glassfrog roles`
Then the system surfaces the authentication fail-safe's refusal as a not-authenticated outcome
And the command exits non-zero, pointing the operator at how to store a credential
And no role data is produced

**Scenario: The API cannot be reached**
Given a valid token but the API is unreachable (connection, DNS, TLS, or timeout)
When the user runs `glassfrog roles`
Then the system reports the transport failure by name on stderr
And the command exits with the network-unavailable code

**Scenario: A single read for an unknown id**
Given a valid token but a role id that does not exist
When the user runs `glassfrog roles <unknown-id>`
Then the system reports that the read failed and names the HTTP status
And the command exits with the API-error code

**Scenario: An unsupported `--include` value is rejected without an API call**
Given a valid token and an existing role id
When the user runs `glassfrog roles <id> --include nonsense`
Then the system rejects the invocation as a usage error, naming the unsupported value and the supported set
And no API call is made

### Edge cases

**Scenario: The organization has no roles**
Given a valid token resolving to an org with no roles (or a filter that matches none)
When the user runs `glassfrog roles`
Then the system produces an empty list result
And the command exits successfully

**Scenario: Roles span more than one page (default walk to completion)**
Given an org whose roles span more than one page of the API response
When the user runs `glassfrog roles`
Then the system walks every page through Pagination (016) and produces the complete set
And the command exits successfully

**Scenario: First-page opt-out stops at one page and signals more exist**
Given an org whose roles span more than one page of the API response
When the user runs `glassfrog roles` with the first-page opt-out flag
Then the system makes a single page request and produces the first page
And it surfaces a clear "more exist" incomplete signal
And the command exits successfully

**Scenario: A list filter is rejected on the single read**
Given a valid token and an existing role id
When the user runs `glassfrog roles <id> --tag some-tag`
Then the system rejects the invocation as a usage error
And no API call is made

---

## Validation Scenarios

> These are held out from the implementing agent for independent verification.

**Scenario: Default output carries no raw API envelope**
Given a successful `glassfrog roles` run under the default human format
When the output is inspected
Then it shows the reshaped projection only
And it does not contain the raw `data`/`meta` JSON envelope or the unflattened role objects

**Scenario: Embedded-include view does not substitute for the standalone reads**
Given a successful `glassfrog roles <id> --include policies` run
When the result is inspected
Then the policies appear embedded inline on the role
And no standalone per-policy projection (the Role Policies #34 surface) is produced here

**Scenario: List incompleteness is never silent**
Given a list run where Pagination (016) could not assemble every page
When the result is inspected
Then an explicit incomplete signal with its cause is present
And the partial list cannot be read as the complete set

---

## Assumptions

- **Command spelling** `glassfrog roles` / `glassfrog roles <id>`: assumed from the FEATURE-MODEL "Role Reads" framing and the chosen positional-id shape, mirroring the existing command groups. (Informed by `FEATURE-MODEL.md` and the `internal/cli` command pattern.)
- **`--include` semantics**: assumed Role Reads reuses Identity Read (011)'s opt-in, reject-unknown `--include` handling and the same supported-set validation. (Informed by spec 011.)
- **Filter flag names** (`--parent`, `--person`, `--has-subroles`, `--tag`): assumed to map to the API's `parent_role_id` / `person_id` / `has_subroles` / `tag`; exact flag spellings and whether `--has-subroles` is a presence flag or takes a boolean value sync with the CLI flag convention at interface time. `[ASSUMED]`
- **Output rendering** is delegated to Output Format Selection (020); the built-in default format is `full` (020's default). (Informed by spec 020.)
- **Failure-to-exit-code mapping** reuses Request Execution's typed outcomes and the Exit-Code Convention (004) mapping rather than defining new codes. (Informed by specs 004 and 010.)
- **First-page opt-out flag spelling**: the *behavior* (walk by default; opt out to a single first page that signals incompleteness) is fixed; the exact flag name syncs with the CLI flag convention at interface time. `[ASSUMED]`

---

## Ambiguity Warnings

*None outstanding.* (The list completeness strategy, formerly open here, is now resolved — see Clarifications.)

---

## Clarifications

### Session 2026-06-08

- **List completeness strategy**: `glassfrog roles` walks every page to completion through Pagination (016) **by default**, and exposes a **first-page opt-out flag** that limits the read to a single page — surfacing the first page with a clear "more exist" incomplete signal when more pages exist (never silently truncated). This pairs the default *page-through* half of CONSTITUTION VI with the *signal-the-boundary* half on opt-out. The exact opt-out flag name is deferred to interface design; the behavior is fixed.
