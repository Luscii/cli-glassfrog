# Specification: Tension Discard

**Feature**: 045-tension-discard
**Role**: Definer
**Tier**: 1 (zero setup)

---

## System Overview

Tension Discard is the **soft-delete operation for an existing tension** — the fourth member of the `tension` command family, after Tension Capture (042) wrote one, Tension Reads (043) surfaced them, and Tension Update (044) edited their fields. It retires a tension already on the record: `glassfrog tension discard <ten-id>` → `DELETE /tensions/{id}` (`deleteTension`), which the API implements as a **soft-delete** (it sets `discarded_at`, it does not erase). Where capture creates the proposal seed, reads expose it, and update corrects it, discard is the explicit retirement the family deliberately deferred from update — the `archived` status transition (044) marks a tension done; discard takes it off the active record entirely.

It continues the verb grammar the family opened: `tension create` (042), `tension list` / `tension get` (043), `tension update` (044), and now `tension discard <ten-id>` — a sub-verb under `tension`, not a new top-level noun. It sits on the proven chain rather than rebuilding it: it hands the request to **Request Execution (010)**, reads identity through **Request Authentication (007)**, lets **Output Format Selection (020)** render its result, and maps outcomes through **Exit-Code Convention (004)** and **API Error Extraction (015)**.

Two facts of the endpoint shape its behavior. First, a successful delete returns **`204` with no body** — there is no server payload to render, so the command **synthesizes** a small structured result (the discarded `ten_` id and a discarded marker) and hands that to Output Format Selection, keeping the family's "produce structured data" contract even though the wire carries nothing. Second, the delete is **not REST-strict idempotent**: the first `DELETE` of a live tension returns `204`, but a subsequent call returns `404` — and an already-discarded tension is indistinguishable from one that never existed, so a single CLI call cannot tell the two apart. Following the API's own guidance ("treat `404`-following-`204` as success"), discard treats **`404` as success**: the end-state is identical (the tension is gone), and the command stays retry-safe for the agents that drive it. The acknowledged cost is that discarding a mistyped or non-existent id reports success rather than surfacing the typo. The two successes mean different things, though — the command deleted a live tension (`204`) versus found one already gone (`404`) — so discard surfaces an **advisory note on the diagnostic channel** saying which happened, while exiting zero either way; the machine-readable result stays uniform.

It is deliberately scoped to *removing one tension*: it does not hard-delete, it does not cascade to the tension's proposals (the server leaves those in place), it offers no restore, and — like update (044) — it sends no `If-Match` concurrency guard (Clobbered Changes is a separate capability).

---

## Behavioral Accord

### Invocation

- When the user runs `glassfrog tension discard <ten-id>`, the system soft-deletes the named tension and produces a discarded-tension result.
- When the user omits the required `<ten-id>` positional, passes more than one positional id, or passes an unknown flag, the system rejects the invocation as a usage error and calls no API.
- Discard takes no editable-field flags — it removes a tension whole; there is nothing to edit. Only the root persistent flags (`--base-url`, `--output`/`-o`) inherited from the command tree apply.

### Output

- When the delete succeeds (the API answers `204`), the system synthesizes a result carrying the discarded `ten_` id and a discarded marker — it has no server payload to echo — and lets Output Format Selection (020) render it in the effective format (`full` / `compact` / `json` / `yaml`); it neither fixes raw API JSON as its default nor defines its own format flag.
- When the API answers `404`, the system treats the outcome as success and produces the same discarded-tension result: a `404` means the tension is gone (already discarded, or never existed), which is the end-state discard exists to reach. The command cannot distinguish an already-discarded tension from a never-existed one, so it does not try to.
- Whichever success the API returned, the system surfaces a short advisory note on the diagnostic channel (stderr) stating whether it discarded a live tension (`204`) or found it already gone (`404`) — the machine-readable result on stdout is identical for both. The note is advisory, not an error, and does not change the exit code.
- When the discard succeeds (`204` or `404`), the system exits successfully.

### Failure

- When no usable token is available, the system surfaces the authentication fail-safe's refusal and exits non-zero with a not-authenticated outcome, reusing the shared not-authenticated message and pointing the operator at how to store a credential.
- When the request cannot reach or complete at the wire (connection, DNS, TLS, timeout), the system surfaces the transport failure by name and exits non-zero with the network-unavailable outcome.
- When the API answers with a non-2xx response **other than `404`** — notably a refused permission (`401`/`403`) or a throttle (`429`) — the system reports that the discard failed, naming the HTTP status, and exits non-zero. The command adds no interpretation of its own; the shared error handling (API Error Extraction, 015) classifies the status — a generic API-error outcome, or the permission and rate-limit outcomes it already distinguishes — and the command surfaces whichever results.
- Whatever the failure, the message names both what went wrong and a concrete next step (Action Transparency), and never includes the token.

---

## User Scenarios

**In order to** retire a tension that no longer belongs on the active record,
**as a** practitioner managing my governance backlog,
**I want to** remove it by id with one command.

**In order to** clean up after a tension I captured by mistake,
**as an** AI agent operating the CLI on a practitioner's behalf,
**I want to** discard it and have a re-run of the same command stay safe rather than fail.

**In order to** feed the discard outcome into a downstream automation,
**as an** AI agent driving the CLI,
**I want** the discard to produce a structured result I can parse as JSON, even though the API returns no body.

---

## Non-Behaviors

- The system must not create, list, get, or update tensions. **Why**: Tension Discard owns the soft-delete alone, matching the project's one-capability-per-spec pattern. Capture is Tension Capture (042); the reads are Tension Reads (043); field edits are Tension Update (044). A command that strayed into those would fork their contracts.
- The system must not hard-delete a tension or cascade deletion to its associated proposals. **Why**: the endpoint is a soft-delete — it sets `discarded_at` and explicitly leaves associated proposals in place ("delete those separately if needed"). The command removes exactly the one tension named and claims no authority over its proposals or its underlying storage.
- The system must not treat a `404` as a not-found failure. **Why**: the API documents the delete as not REST-strict idempotent — a `404` following a prior `204` is success, and a single call cannot tell an already-discarded tension from a never-existed one. Treating `404` as the success end-state keeps the command retry-safe for the agents that drive it; the accepted cost is that a mistyped id reports success rather than surfacing the typo.
- The system must not prompt for confirmation or require a `--force`/`--yes` flag before discarding. **Why**: the CLI is built for non-interactive, agent-driven use; an interactive guard would break that contract, and the caller already controls whether to issue the command. The soft-delete is recoverable in spirit (the record is marked, not erased), so a blocking prompt adds friction without a matching safety gain.
- The system must not offer a way to restore or un-discard a tension. **Why**: the API exposes no un-delete operation; a restore affordance would be speculative scope with no endpoint behind it. Reversal, if it ever exists, is its own capability.
- The system must not send an `If-Match` precondition or otherwise guard against concurrent edits. **Why**: optimistic concurrency is **Clobbered Changes**, a separate Client-Foundation capability relevant across every write, not this one command. Discard deletes unconditionally — exactly as the API behaves when `If-Match` is omitted — and opts into the shared guard when Clobbered Changes lands rather than growing its own.
- The system must not emit raw API JSON as a fixed default, nor define its own output-format flag. **Why**: Output Format Selection (020) resolves the format and dispatches to the 018/019 renderers; a private flag here would fork that contract.
- The system must not resolve the base URL or token, attach the `X-Auth-Token` header, decide the no-token fail-safe, type a non-2xx response, or choose its own exit codes. **Why**: Base URL Resolution (008), Credential Discovery (005), Request Authentication (007), API Error Extraction (015), and Exit-Code Convention (004) own those; a second path here would drift from their contracts.

---

## Integration Boundaries

- **Glassfrog API v5 (`DELETE /tensions/{id}`, `deleteTension`)**: the system soft-deletes an existing tension. The path `id` is the tension's `ten_` id; the request carries no body. A live tension returns `204` (no body, success); a subsequent delete returns `404` (the tension is already gone — treated as success). The delete sets `discarded_at` and does **not** cascade to associated proposals. A `401`/`403` means the caller may not delete it; a `429` means throttled. When the endpoint is unreachable, the command surfaces a transport failure and exits non-zero.
- **Request Execution (010) / Request Authentication (007)**: the command builds the bodyless `DELETE`, hands it to these seams to authenticate and execute, and does not re-implement transport or auth. Request Execution already drains the bodyless `2xx` response; the command reads only the outcome status, not a payload.
- **Output Format Selection (020) / Structured Serialization (018) / Templated Human Rendering (019)**: the command hands its synthesized discarded-tension result to the output seam, which renders it in the effective format. The command produces structured data, never pre-rendered output — even though the data is synthesized client-side rather than echoed from the server.
- **Exit-Code Convention (004) / API Error Extraction (015)**: the command maps the success (`204`/`404`), the non-2xx (other than `404`), transport, and not-authenticated outcomes to exit codes and messages through these seams.
- **Tension Capture (042) / Tension Reads (043) / Tension Update (044) — siblings**: 042 established the `tension` command, the verb grammar, and the `Tension` model shape; 043 and 044 extended it with reads and edits. Discard continues that grammar by attaching one more leaf — `discard` — and reuses the discarded tension's `ten_` id as the load-bearing reference in its synthesized result.

---

## Driving Scenarios

### Happy path

**Scenario: Discard a live tension**
Given a stored credential and an existing, live `ten_` id
When the user runs `glassfrog tension discard ten_<id>`
Then the system issues `DELETE /tensions/ten_<id>` with no body
And the API answers `204`
And the system produces a discarded-tension result naming the id
And surfaces an advisory note that it discarded the tension
And exits successfully.

**Scenario: Re-discarding an already-gone tension stays safe**
Given a stored credential and a `ten_` id that has already been discarded
When the user runs `glassfrog tension discard ten_<id>`
Then the API answers `404`
And the system treats the outcome as success
And produces the same discarded-tension result naming the id
And surfaces an advisory note that the tension was already gone
And exits successfully.

**Scenario: Discard result rendered as JSON**
Given a stored credential and an existing `ten_` id
When the user runs `glassfrog tension discard ten_<id> -o json`
Then the system synthesizes the discarded-tension result
And Output Format Selection renders it as JSON
And exits successfully.

### Error scenarios

**Scenario: Missing tension id is rejected before any request**
Given a stored credential
When the user runs `glassfrog tension discard` with no positional id
Then the system rejects the invocation as a usage error naming the required `<ten-id>`
And sends no request.

**Scenario: No credential surfaces the not-authenticated outcome**
Given no usable token is available
When the user runs `glassfrog tension discard ten_<id>`
Then the system surfaces the not-authenticated refusal
And exits non-zero
And the message points the operator at how to store a credential.

**Scenario: A refused permission fails loudly**
Given a stored credential lacking permission to delete the tension
When the user runs `glassfrog tension discard ten_<id>`
Then the API answers `403`
And the system reports that the discard failed, naming the HTTP status
And exits non-zero.

### Edge cases

**Scenario: More than one positional id is a usage error**
Given a stored credential
When the user runs `glassfrog tension discard ten_<a> ten_<b>`
Then the system rejects the invocation as a usage error
And sends no request.

**Scenario: A transport failure surfaces network-unavailable**
Given a stored credential and an unreachable API endpoint
When the user runs `glassfrog tension discard ten_<id>`
Then the system surfaces the transport failure by name
And exits non-zero with the network-unavailable outcome.

---

## Validation Scenarios

> These are held out from the implementing agent for independent verification.

**Scenario: Discard exposes no read or write verb of its own**
Given the `tension discard` command
When a reviewer exercises it for any create-, list-, get-, or update-style behavior
Then the command offers none — it issues only a `DELETE` and renders only a discard result
And its help text describes removal alone.

**Scenario: The synthesized result claims nothing the server did not return**
Given a successful discard (`204` or `404`, both bodyless)
When a reviewer inspects the rendered result
Then it carries the discarded `ten_` id the caller supplied and a discarded marker
And it does not fabricate server-owned fields (e.g. a `discarded_at` timestamp) that no response body provided.

**Scenario: The `404` path leaks no not-found error**
Given a `ten_` id the API answers `404` for
When the user runs `glassfrog tension discard ten_<id>`
Then no not-found failure message reaches the user
And the advisory note (the tension was already gone) is informational, not an error
And the outcome is the success result and a zero exit code.

---

## Assumptions

- **Synthesized result shape**: The success result carries the discarded `ten_` id and a discarded marker, and no server-provided fields, because both the `204` and `404` responses are bodyless. (Informed by the endpoint returning no payload; the exact field set is an interface-design detail for the next stage. Technical default.)
- **Positional id form**: The `<ten-id>` is the tension's `ten_` id (`^ten_[0-9a-f]{32}$`), matching how `tension get`/`update` (043/044) take it. (Informed by the API path-parameter pattern. Technical default.)
- **`404` is the only success-bearing non-2xx**: Every non-2xx status other than `404` is a failure routed through API Error Extraction (015). (Informed by the endpoint's documented responses: `204`, `401`, `403`, `404`. Technical default.)

---

## Ambiguity Warnings

None remaining — the one open question (distinguishing a `204` discard from a `404` already-gone outcome) was resolved during clarification (see Clarifications below).

---

## Clarifications

### Session 2026-06-13

- **`404` handling (idempotency)**: discard treats a `404` as success, not a not-found failure. The API documents the delete as not REST-strict idempotent and a single CLI call cannot distinguish an already-discarded tension from a never-existed one, so the command reaches for the success end-state (the tension is gone) and stays retry-safe. The accepted cost — a mistyped id reports success — is recorded as a non-behavior.
- **Bodyless success output**: a successful delete returns `204` with no body, so the command synthesizes a small structured result (the discarded `ten_` id and a discarded marker) and renders it through Output Format Selection (020), preserving the family's "produce structured data" contract even though the wire carries no payload.
- **Distinguishing `204` from `404`**: the success result on stdout is identical for both, but the command surfaces a short **advisory note on the diagnostic channel (stderr)** stating whether it discarded a live tension (`204`) or found one already gone (`404`). The note is informational, not an error, and does not change the zero exit code.
