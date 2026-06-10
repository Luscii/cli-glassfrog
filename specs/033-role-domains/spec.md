# Specification: Role Domains

**Feature**: 033-role-domains
**Role**: Definer
**Tier**: 1 (zero setup)

---

## System Overview

Role Domains is a **per-role read** in the Governance Reads slice. A **domain** is an area of control held by a role (e.g. *"Code review standards"*) — embedded on a role for convenience, but also addressable in its own right. This feature offers the addressable surface: list the domains a given role controls (`GET /roles/{id}/domains` → `listRoleDomains`, which takes a *required* role-id path parameter) and read a single domain by its own `dom_` id (`GET /domains/{id}` → `getDomain`), optionally embedding the policies scoped to that domain.

Where My Roles (012) and Role Reads (025) already surface a role's domains *inline* as part of the role, Role Domains answers "what does *this* role control, in detail" and "what is *this one* domain, with its governing policies" — the standalone projection 025 deliberately left to it. It depends on Role Reads (025) for the role ids its list consumes, and sits on the proven read chain rather than rebuilding it: it hands requests to **Request Execution (010)**, reads identity through **Request Authentication (007)**, walks the list through **Pagination (016)**, and lets **Output Format Selection (020)** render the result. Its own job is small: build the per-role-domains list or the single-domain request, apply the caller's search or `--include`, ask the seams to fetch, and produce the domain data as its result — never the raw nested payload by default.

---

## Behavioral Accord

### Invocation

- When the user requests a role's domains with a role id, the system reads that role's domains and produces them as a list result.
- When the user requests a single domain with a domain id, the system reads the domain with that id and produces it as a single-domain result.
- When the user requests a role's domains without supplying the required role id, the system rejects the invocation as a usage error and calls no API — the list is role-scoped and has no org-wide form.
- When the user passes an unknown flag, or more than one positional id where one is expected, the system rejects the invocation as a usage error and calls no API.

### List search

- When the user supplies a search term on the list, the system sends it as the `q` full-text query, narrowing the role's domains server-side; an empty or whitespace-only term is ignored (treated as no search), and a malformed query yields an empty list rather than an error — mirroring the API's documented `q` behavior.
- When the user supplies no search term, the system requests all of the role's domains.

### Related resources on the single read

- When the user supplies `--include policies` on the single read, the system sends `?include=policies` and embeds the policies scoped to that domain inline in the domain result.
- When the user supplies `--include` with a value outside the supported set (`policies`), the system rejects it as a usage error before issuing any request, naming the unsupported value and the supported set — mirroring how Identity Read (011) and Role Reads (025) validate their own `--include`.
- When the user supplies no `--include`, the system reads the domain without its policies.
- When the user supplies `--include` on the list (the role-scoped read), the system rejects it as a usage error — related-resource embedding is a single-read concern.

### Output

- When a read succeeds, the system produces the domain data as its result and lets Output Format Selection (020) render it in the effective format (`full` / `compact` / `json` / `yaml`); it neither fixes raw API JSON as its default nor defines its own format flag.
- When the role controls no domains, or no domain matches the search, the system produces an empty list result and exits successfully — an empty list is a valid answer, not an error.

### Completeness of the list

- When the role's domains span more than one page, the system walks every page through Pagination (016) by default and produces the complete set.
- When the user supplies the first-page opt-out flag, the system makes a single page request and, if more pages exist, produces the first page flagged incomplete with a clear "more exist" signal — so even the opted-out result is never silently truncated.
- When the walk cannot complete (a page fails mid-walk), the system produces the domains gathered so far, flagged incomplete with the cause, so a partial list is never mistaken for the whole.

### Failure

- When no usable token is available, the system surfaces the authentication fail-safe's refusal and exits non-zero with a not-authenticated outcome, reusing the shared not-authenticated message and pointing the operator at how to store a credential.
- When the request cannot reach or complete at the wire (connection, DNS, TLS, timeout), the system surfaces the transport failure by name and exits non-zero with the network-unavailable outcome.
- When the API answers with a non-2xx response — including a list whose role id does not exist or a single read whose domain id does not exist (typically `404`), or a malformed list query the API rejects (`400`) — the system reports that the read failed, naming the HTTP status, and exits non-zero. The command adds no interpretation of its own; the shared error handling (API Error Extraction, 015) classifies the status — a generic API-error outcome, or the permission (`401`/`403`) and rate-limit (`429`) outcomes it already distinguishes — and the command surfaces whichever results.
- Whatever the failure, the message names both what went wrong and a concrete next step (Action Transparency), and never includes the token.

---

## User Scenarios

**In order to** see exactly what a role is accountable to control,
**as an** AI agent operating the CLI on a practitioner's behalf,
**I want to** list a role's domains by its id.

**In order to** understand a single area of control together with the policies that govern it,
**as an** AI agent assembling context before acting,
**I want to** read one domain by its id with its policies embedded inline.

**In order to** find a particular area of control on a role with many domains,
**as a** practitioner exploring the org,
**I want to** search a role's domains by a full-text term.

**In order to** trust that I am seeing every domain a role controls,
**as a** practitioner reviewing a large circle,
**I want** the list to walk to completion, or to tell me plainly when it is incomplete.

---

## Non-Behaviors

- The system must not re-implement the *inline* embed of domains on a role. **Why**: Role Reads (025) and My Roles (012) already carry domains inline as a convenience view; Role Domains owns the *addressable standalone projection* (its own per-domain shape and the `?include=policies` embed). The two views coexist deliberately, and duplicating the inline view here removes this surface's reason to exist.
- The system must not offer an org-wide "all domains" list, nor read a role's domains without the required role id. **Why**: the API's domains list is role-scoped (a required `{id}` path parameter); there is no endpoint for every domain in the org, and inventing one would diverge from the spec contract.
- The system must not provide a *standalone* policy read. **Why**: `--include policies` here embeds policies *inline on the domain* as a convenience; the addressable per-policy projection belongs to Role Policies (#34). Forking a second policy read would drift from that contract.
- The system must not emit raw API JSON as a fixed default, nor define its own output-format flag. **Why**: Output Format Selection (020) resolves the format and dispatches to the 018/019 renderers; a private flag here would fork that contract.
- The system must not resolve the base URL or token, attach the `X-Auth-Token` header, decide the no-token fail-safe, type a non-2xx response, or choose its own exit codes. **Why**: Base URL Resolution (008), Credential Discovery (005), Request Authentication (007), API Error Extraction (015), and Exit-Code Convention (004) own those; a second path here would drift from their contracts.
- The system must not write to or mutate any domain. **Why**: Role Domains is a read surface; governance changes are made only through Proposals, which are out of scope here and reflect the read-only stance of the governance reads.

---

## Integration Boundaries

- **Glassfrog API v5 (`GET /roles/{id}/domains`, `GET /domains/{id}`)**: the system reads domains through these endpoints. Data flows inbound (domains in, nothing written). When an endpoint is unreachable or answers non-2xx, the command surfaces a transport or read failure and exits non-zero.
- **Request Execution (010)**: the seam the command hands each request to and reads the outcome from (success-with-body, non-2xx, transport error, decode error).
- **Request Authentication (007)**: supplies the authenticated transport and the no-token fail-safe whose refusal the command propagates.
- **Pagination (016)**: the walker the list read uses to assemble the complete set across pages (or a flagged-incomplete partial set when a page fails).
- **Output Format Selection (020)**: resolves the effective `--output` format and dispatches the produced result to the matching renderer; the command produces result data, not presentation.
- **Exit-Code Convention (004)**: the command maps its outcome to a process exit code through the established convention.
- **Role Reads (025)**: the source of the role ids the per-role-domains list consumes — a dependency for usable input, not a runtime seam.
- **User / AI agent (stdout/stderr)**: the rendered result is written to stdout on success; failure messages are written to stderr.

---

## Driving Scenarios

### Happy path

**Scenario: List a role's domains**
Given a valid token and a role id that exists in the org and controls several domains
When the user requests that role's domains
Then the system produces a list result of the role's domains
And the command exits successfully

**Scenario: Read a single domain by id**
Given a valid token and a domain id that exists in the org
When the user requests that single domain
Then the system produces a single-domain result carrying its description and controlling role
And the command exits successfully

**Scenario: Read a single domain with its policies embedded**
Given a valid token and an existing domain id
When the user requests that domain with `--include policies`
Then the system sends `include=policies` and embeds the domain's policies inline in the result
And the command exits successfully

**Scenario: Search a role's domains**
Given a valid token and a role id whose domains include one matching a search term
When the user requests that role's domains with the search term
Then the system sends `q` on the request and produces only the matching domains
And the command exits successfully

### Error scenarios

**Scenario: No usable token**
Given no usable token is available to the CLI
When the user requests a role's domains
Then the system surfaces the authentication fail-safe's refusal as a not-authenticated outcome
And the command exits non-zero, pointing the operator at how to store a credential
And no domain data is produced

**Scenario: The API cannot be reached**
Given a valid token but the API is unreachable (connection, DNS, TLS, or timeout)
When the user requests a role's domains
Then the system reports the transport failure by name on stderr
And the command exits with the network-unavailable code

**Scenario: A single read for an unknown domain id**
Given a valid token but a domain id that does not exist
When the user requests that single domain
Then the system reports that the read failed and names the HTTP status
And the command exits with the API-error code

**Scenario: An unsupported `--include` value is rejected without an API call**
Given a valid token and an existing domain id
When the user requests that domain with `--include nonsense`
Then the system rejects the invocation as a usage error, naming the unsupported value and the supported set
And no API call is made

### Edge cases

**Scenario: The role controls no domains**
Given a valid token and a role id that exists but controls no domains (or a search that matches none)
When the user requests that role's domains
Then the system produces an empty list result
And the command exits successfully

**Scenario: A role's domains span more than one page (default walk to completion)**
Given a role whose domains span more than one page of the API response
When the user requests that role's domains
Then the system walks every page through Pagination (016) and produces the complete set
And the command exits successfully

**Scenario: First-page opt-out stops at one page and signals more exist**
Given a role whose domains span more than one page of the API response
When the user requests that role's domains with the first-page opt-out flag
Then the system makes a single page request and produces the first page
And it surfaces a clear "more exist" incomplete signal
And the command exits successfully

**Scenario: `--include` is rejected on the role-scoped list**
Given a valid token and an existing role id
When the user requests that role's domains with `--include policies`
Then the system rejects the invocation as a usage error
And no API call is made

---

## Validation Scenarios

> These are held out from the implementing agent for independent verification.

**Scenario: Default output carries no raw API envelope**
Given a successful role-domains list run under the default human format
When the output is inspected
Then it shows the reshaped projection only
And it does not contain the raw `data`/`meta` JSON envelope or the unflattened domain objects

**Scenario: Embedded-policies view does not substitute for the standalone policy read**
Given a successful single-domain run with `--include policies`
When the result is inspected
Then the policies appear embedded inline on the domain
And no standalone per-policy projection (the Role Policies #34 surface) is produced here

**Scenario: List incompleteness is never silent**
Given a list run where Pagination (016) could not assemble every page
When the result is inspected
Then an explicit incomplete signal with its cause is present
And the partial list cannot be read as the complete set

---

## Assumptions

- **Command spelling** is deferred to interface design. 025 established that per-role reads cannot be children of `roles` (a positional id forecloses subcommands), so Role Domains needs its own command surface; the wrinkle here is that the *list* keys off a **role** id while the *single* read keys off a **domain** id. The behavior is fixed (a role-scoped list + a single-domain read); the exact command/flag spelling syncs with the CLI command convention at interface time. `[ASSUMED]`
- **`--include` semantics**: assumed Role Domains reuses Identity Read (011) / Role Reads (025)'s opt-in, reject-unknown `--include` handling, with the supported set drawn from the endpoint (`policies`). (Informed by specs 011 and 025.)
- **Search flag**: the *behavior* is fixed (sends the API's `q` full-text query; empty/whitespace-only ignored; malformed yields an empty list, not an error); the exact flag spelling syncs with the CLI flag convention at interface time. `[ASSUMED]`
- **First-page opt-out flag spelling**: the *behavior* (walk by default; opt out to a single first page that signals incompleteness) is fixed; the exact flag name syncs with the CLI flag convention at interface time. `[ASSUMED]`
- **Output rendering** is delegated to Output Format Selection (020); the built-in default format is `full` (020's default). (Informed by spec 020.)
- **Failure-to-exit-code mapping** reuses Request Execution's typed outcomes and the Exit-Code Convention (004) mapping rather than defining new codes. (Informed by specs 004 and 010.)

---

## Ambiguity Warnings

*None outstanding.* The scope (both reads), the search filter, and the command-spelling deferral were resolved in the defining conversation; what remains open is interface-level flag/command spelling, captured as assumptions for the interface step rather than behavioral ambiguities.
