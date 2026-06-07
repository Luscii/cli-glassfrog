# Specification: My Roles

**Feature**: 012-my-roles
**Role**: Definer
**Tier**: 1 (zero setup)

---

## System Overview

My Roles is a self-service read command in the **Self-Service Reads** slice — the "what's mine" surface. It lets a practitioner (usually via an AI agent) list the roles they personally fill, without naming themselves: the token already scopes the call to one person in one organization. It maps to exactly one endpoint — `GET /me/roles` → `listMyRoles` (`spec/glassfrog-api-v5.yaml:1003`) — which returns the roles the requester fills through a *primary, non-discarded* assignment. This is deliberately narrower than the org-wide `GET /roles`: it answers "what do I hold here," the read half of VISION success #2 (an agent reads a practitioner's roles, then submits a proposal).

It is an endpoint command sitting on the proven transport chain: it hands a request to **Request Execution (010)** and reads identity through **Request Authentication (007)**, which attaches the token and owns the no-token fail-safe. It does not re-implement transport, identity, base-URL resolution, or exit-code mapping — those are owned upstream (010, 007, 008/009, 004). Its own job is small: build the `me roles` request, ask the seam to send it, and turn the outcome into a concise projection of the practitioner's roles or a named failure.

---

## Behavioral Accord

### Invocation

- When the user runs `glassfrog me roles` with no arguments, the system reads the roles the authenticated practitioner fills and reports them.
- When the user passes any positional argument or unknown flag to `me roles`, the system rejects the invocation as a usage error and does not call the API. **Note**: which command group owns and registers the `me` parent is a registration concern shared with Identity Read (011); this spec governs only the observable behavior of `me roles`.

### Output

- When the API returns the practitioner's roles, the system prints a concise, legible projection of each role — its name, its purpose, its accountabilities and its domains (each shown as its description text), and a minimal identifier (present so an agent can make follow-up calls, but kept unobtrusive) — rather than the raw nested API payload, and exits successfully. The role's fillers, tags, and classification flags are not part of the default projection.
- When the practitioner fills no roles, the system reports an empty result plainly (e.g. "no roles") and exits successfully — an empty list is a valid answer, not an error.

### Incomplete results

- When more roles exist than the single API response carried, the system prints the roles it received and clearly signals that the result is incomplete (more roles exist than shown), so a partial list is never mistaken for the whole.

### Failure

- When no usable token is available, the system surfaces the authentication fail-safe's refusal and exits non-zero with a not-authenticated outcome, pointing the operator at how to store a credential; it reuses the shared not-authenticated message rather than inventing its own.
- When the request cannot reach or complete at the wire (connection, DNS, TLS, timeout), the system surfaces the transport failure by name and exits with the runtime-failure code.
- When the API answers with a non-2xx response, the system reports that the read failed, naming the HTTP status, and exits with the runtime-failure code; it does not interpret which kind of API error it was.
- Whatever the failure, the message names both what went wrong and a concrete next step the operator can take (Action Transparency), and never includes the token.

---

## User Scenarios

**In order to** see which roles I hold so I can decide where to sense a tension or raise a proposal,
**as a** practitioner whose governance work the CLI serves,
**I want to** list my roles with one command, without having to identify myself.

**In order to** orient on a practitioner's responsibilities before acting on their behalf,
**as an** AI agent operating the CLI,
**I want** each role returned as a concise, parseable projection carrying its name, purpose, and identifier.

**In order to** trust that I am seeing every role I hold,
**as a** practitioner with many roles,
**I want** the command to tell me when the list it printed is incomplete rather than silently truncating it.

---

## Non-Behaviors

- The system must not list any roles other than those the authenticated practitioner fills, nor accept a person/actor selector. **Why**: My Roles is the token-scoped self-service read; org-wide and by-person role reads belong to the Governance Reads surface (`GET /roles`), and adding a selector here would blur that boundary and exceed the caller's intent.
- The system must not follow pagination or fetch additional pages to assemble a complete list. **Why**: Pagination (016) owns walking the API's paging and Request Execution (010) returns a single response; duplicating paging here would fork that contract. My Roles surfaces the first response and signals incompleteness instead.
- The system must not emit the raw API JSON as its default output, nor add an output-format flag. **Why**: the default is a reshaped/summarized projection consistent with Identity Read (011); a raw `--output json` mode is the Unconsumable Output capability's concern and will be introduced there, not invented here.
- The system must not resolve the base URL or token, attach the `X-Auth-Token` header, decide the no-token fail-safe, or choose its own exit codes. **Why**: Base URL Resolution (008), Credential Discovery (005), Request Authentication (007), and Exit-Code Convention (004) own those; a second path here would drift from their contracts.
- The system must not interpret a non-2xx response into a specific, meaningful API error. **Why**: API Error Extraction (015) owns turning a raw status and body into a typed error; My Roles only reports that the read failed and names the status.

---

## Integration Boundaries

- **Glassfrog API v5 (`GET /me/roles`)**: the system reads the requester's roles through this endpoint. Data flows inbound (roles in, nothing written). When the endpoint is unreachable or answers non-2xx, the command surfaces a transport or read failure and exits non-zero.
- **Request Execution (010)**: the seam the command hands its request to and reads the outcome from (success-with-body, non-2xx, transport error, decode error).
- **Request Authentication (007)**: supplies the authenticated transport and the no-token fail-safe whose refusal the command propagates.
- **Exit-Code Convention (004)**: the command maps its outcome to a process exit code through the established convention rather than choosing codes itself.
- **User / AI agent (stdout/stderr)**: the projection is written to stdout on success; failure messages are written to stderr.

---

## Driving Scenarios

### Happy path

**Scenario: List the roles I fill**
Given a valid token resolving to a practitioner who fills several roles
When the user runs `glassfrog me roles`
Then the system prints a concise projection of each role — name, purpose, identifier, and structural signals
And the command exits successfully

**Scenario: A projected role carries its essentials, not the raw payload**
Given a valid token and a practitioner filling at least one role
When the user runs `glassfrog me roles`
Then each printed role shows its name, its purpose, its accountabilities and its domains as description text, and a minimal identifier
And the role's fillers, tags, and classification flags are not shown in the default output
And the raw nested API objects are not dumped verbatim

**Scenario: The practitioner fills no roles**
Given a valid token resolving to a practitioner who fills no roles
When the user runs `glassfrog me roles`
Then the system reports an empty result plainly
And the command exits successfully

### Error scenarios

**Scenario: No usable token**
Given no usable token is available to the CLI
When the user runs `glassfrog me roles`
Then the system surfaces the authentication fail-safe's refusal as a not-authenticated outcome
And the command exits non-zero, pointing the operator at how to store a credential
And no role data is printed

**Scenario: The API cannot be reached**
Given a valid token but the API is unreachable (connection, DNS, TLS, or timeout)
When the user runs `glassfrog me roles`
Then the system reports the transport failure by name on stderr
And the command exits with the runtime-failure code

**Scenario: The API answers with a non-2xx status**
Given a valid token but the API returns a non-2xx response
When the user runs `glassfrog me roles`
Then the system reports that the read failed and names the HTTP status
And the command exits with the runtime-failure code

### Edge cases

**Scenario: More roles exist than one response carried**
Given a practitioner whose roles span more than one page of the API response
When the user runs `glassfrog me roles`
Then the system prints the roles from the response it received
And clearly signals that the result is incomplete
And the command exits successfully

**Scenario: Extra arguments are rejected without an API call**
Given a valid token
When the user runs `glassfrog me roles extra-argument`
Then the system rejects the invocation as a usage error
And no API call is made

---

## Validation Scenarios

> These are held out from the implementing agent for independent verification.

**Scenario: Default output contains no raw API envelope**
Given a successful `glassfrog me roles` run
When the output is inspected
Then it shows the reshaped projection only
And it does not contain the raw `data`/`meta` JSON envelope or the unflattened role objects

**Scenario: Incompleteness is never silent**
Given a run where the API response did not carry every role the practitioner fills
When the output is inspected
Then an explicit incomplete-result signal is present
And the partial list cannot be read as the complete set

---

## Assumptions

- **Command spelling** `glassfrog me roles`: assumed from the FEATURE-MODEL framing ("my roles") and the ROADMAP "/me, my roles" slice, mirroring the existing `auth login` group/leaf shape. (Informed by `FEATURE-MODEL.md` and the `internal/cli` command pattern.)
- **Failure-to-exit-code mapping**: assumed My Roles reuses Request Execution's typed outcomes and the Exit-Code Convention mapping (auth failure vs. runtime failure vs. usage error) rather than defining new codes. (Informed by spec 004 and spec 010.)
- **No locale handling**: assumed the command does not pass `?locale=`; role names render as the API returns them under the org/token default. (Informed by the spec's name-localization note; locale selection is not in the Now slice.)

---

## Ambiguity Warnings

1. **[NEEDS CLARIFICATION] Incompleteness signal form**: the spec requires that an incomplete result be signalled but does not fix *how* (a stderr note, a trailing line, a count like "showing N of more"). The Shaper should choose a form that is both agent-parseable and human-legible, consistent with the default projection.

---

## Clarifications

### Session 2026-06-07

- **Default projection content**: Per role, the projection shows name, purpose, the role's accountabilities and domains (each as its description text), and a minimal identifier (kept unobtrusive but present for agent follow-up calls). Fillers, tags, and classification flags are excluded from the default view. The precise field labelling still syncs with the reshaped-projection convention Identity Read (011) is establishing in parallel; the field *selection* above is fixed.
