# Specification: Guarded Writes

**Feature**: 053-guarded-writes
**Role**: Definer
**Tier**: 1 (zero setup)

---

## System Overview

Guarded Writes is a **Maintainer-facing mechanism** and the second step of the *Optimistic Concurrency* solution (the **Clobbered Changes** problem). The Glassfrog API v5 lets a write be made conditional: send a resource's version back on a `PUT`/`PATCH`/`DELETE` via the `If-Match` header and the server accepts the write only if the resource is unchanged, otherwise refusing it with `412 Precondition Failed`. When `If-Match` is omitted the write proceeds unconditionally — last-write-wins — which is the gap this capability closes. Version Capture on Read (052) already retains the version a single-resource read carried and exposes it in-process (`Response.Version()`); today nothing sends it back, so even a captured version can't guard anything. This capability is the send half: given a captured version, it attaches that version to a write request as an `If-Match` precondition so the server can reject a stale write instead of silently overwriting it.

It is a pure plumbing capability that completes the capture-then-send split: 052 captures, 053 sends. Like 052 it is **setting-agnostic and resource-agnostic** — it forwards whatever opaque version token it is given, on whatever write request it is given, so any write (a tension today, a role-bearing proposal tomorrow) can be guarded through one shared mechanism rather than per-command copies. It deliberately stops at sending the precondition: it does **not** decide which commands read-before-write (retrofitting each write call-site is per-command work), and it does **not** interpret the server's refusal — distinctly reporting the resulting `412` so the operator can re-read is **Stale-Write Surfacing (054)**.

---

## Behavioral Accord

### Precondition attachment

- When a caller issues a write request carrying a captured resource version, the system sends that version on the outbound request as an `If-Match` precondition, so the server accepts the write only if the resource is unchanged since it was read.
- When a write request carries no version (none was captured, or the read carried no version indicator), the system sends no `If-Match` precondition and the write proceeds unconditionally — last-write-wins, exactly as today.

### Fidelity

- When a version is sent, it is the exact token the read captured — the system does not unquote, strip a weak-validator (`W/…`) prefix, normalize, or otherwise re-shape it — so the precondition matches what the server expects byte-for-byte.

### Scope

- The system attaches the precondition based solely on whether a version is present on the write request, independent of the request's method, so any conditional write the caller marks (including a delete) is guarded the same way.
- When a guarded write is refused (a `412` or any other outcome), the system does not interpret, relabel, or specially handle that response — existing diagnostic and exit-code handling is unchanged; distinct surfacing of the refusal is a separate capability.

---

## User Scenarios

**In order to** complete the optimistic-concurrency foundation so an edit can actually be refused when the resource changed underneath it,
**as a** Maintainer building the Optimistic Concurrency capability,
**I want to** send a captured version as an `If-Match` precondition on a write, so the one shared mechanism turns a retained version into an enforced guard without each write command re-deriving how.

**In order to** avoid silently clobbering a concurrent governance edit made on my behalf,
**as a** practitioner whose work the CLI serves,
**I want to** have a write I make refused by the server when the resource changed since it was read, so my edit cannot quietly overwrite someone else's.

---

## Non-Behaviors

- The system must not perform the read that obtains the version. **Why**: capturing the version is Version Capture on Read (052); deciding when a command reads-before-write is per-command retrofit work. Folding either in would couple this mechanism to a read path and a command, blurring the roadmap's clean capture → send → surface split.
- The system must not interpret, relabel, or specially handle the server's refusal of a guarded write (`412 Precondition Failed`). **Why**: distinct surfacing of the clobber so the operator can re-read is Stale-Write Surfacing (054); handling it here would duplicate that capability and pre-empt its design.
- The system must not send an `If-Match` precondition when no version is present. **Why**: an absent version is the normal, expected case (many resources carry no `ETag`); sending an empty or fabricated precondition would invite a spurious `412` and break the unconditional last-write-wins path the API documents for an omitted `If-Match`.
- The system must not interpret, unquote, normalize, or validate the version token before sending it. **Why**: the `ETag` is an opaque server fingerprint (possibly quoted, possibly a weak validator); any re-shaping risks the server seeing a token it never issued and failing the precondition spuriously — verbatim forwarding is the only behavior that preserves the match 052 took care to capture.
- The system must not wire any specific production write command (Tension Update, Tension Discard, Proposal write-flow) onto this mechanism. **Why**: retrofitting each write call-site to read-then-guard is per-command work with its own behavior and tests; this spec ships the shared mechanism, mirroring the 052 (mechanism) → call-site (retrofit) split.
- The system must not change the outbound request of any existing write or read when no version is supplied. **Why**: the landed commands must stay byte-identical until a caller opts in; a mechanism that altered requests unconditionally would be an unrequested behavior change across the whole surface.

---

## Integration Boundaries

- **Glassfrog API v5** *(upstream)*: accepts the `If-Match` request header on conditional writes of mutable resources and refuses a stale write with `412 Precondition Failed`. The CLI sends the header when a version is present and otherwise omits it; it does not negotiate or probe for conditional-write support.
- **Version Capture on Read (052)** *(in-process, upstream)*: supplies the verbatim version token this capability forwards. When 052 captured nothing (empty), this capability sends no precondition.
- **Stale-Write Surfacing (054)** *(in-process, downstream)*: consumes the `412` outcome this capability's preconditioned writes can produce, and reports it distinctly. This spec produces the conditions for that `412` but does not handle it.

---

## Driving Scenarios

### Happy path

**Scenario: Captured version guards the write**
Given a write request carrying a version captured from the resource's prior read
When the request is sent
Then the outbound request carries an `If-Match` header equal to that version
And the server accepts the write only if the resource is unchanged.

**Scenario: No version falls through to an unconditional write**
Given a write request that carries no captured version
When the request is sent
Then the outbound request carries no `If-Match` header
And the write proceeds unconditionally, exactly as before this capability existed.

**Scenario: A delete is guarded the same way**
Given a delete request carrying a captured version
When the request is sent
Then the outbound request carries an `If-Match` header equal to that version
And the precondition is attached identically regardless of the request's method.

### Error scenarios

**Scenario: Server refuses a stale write**
Given a write request carrying a version that no longer matches the resource's current state
When the request is sent and the server responds with `412 Precondition Failed`
Then the system does not interpret or relabel the response
And the outcome flows through existing diagnostic and exit-code handling unchanged.

**Scenario: Empty captured version is not sent as a precondition**
Given a write request whose captured version is empty (the read carried no version indicator)
When the request is sent
Then no `If-Match` header is attached
And the write is not refused for a malformed precondition — empty is treated as "no version", not as a value to send.

### Edge cases

**Scenario: Weak-validator version is forwarded verbatim**
Given a write request carrying a captured version that is a quoted weak validator (a `W/…` token)
When the request is sent
Then the `If-Match` header carries the token byte-for-byte, including its quotes and `W/` prefix.

**Scenario: Precondition composes with an existing content type**
Given a write request that already carries a body media type and also carries a captured version
When the request is sent
Then the outbound request carries both its existing `Content-Type` and the new `If-Match` header
And neither header displaces or alters the other.

---

## Validation Scenarios

> These are held out from the implementing agent for independent verification.

**Scenario: The spec names no command it secretly retrofits**
Given the produced artifact
When it is read end to end
Then it describes only the shared send mechanism
And it commits to wiring no specific production write command, consistent with the mechanism/retrofit split it claims.

**Scenario: The 412 is owned downstream, not here**
Given the produced artifact
When its treatment of `412 Precondition Failed` is read
Then it produces the condition for that refusal but defers all interpretation and surfacing of it to Stale-Write Surfacing (054), with no overlap.

---

## Assumptions

- **Verbatim token from 052**: The version forwarded is exactly what `Response.Version()` returns — the `ETag` the read captured, with no transformation. (Informed by 052's pinned verbatim-fidelity contract, which exists precisely so this capability can forward unchanged.)
- **Empty-as-absent symmetry with 052**: An empty version and a never-captured version are indistinguishable to this capability — both mean "no precondition to send". (Mirrors 052's empty-as-absent sentinel so the capture and send halves agree.)
- **[ASSUMED] Narrow precondition field, not a general header bag**: The version is conveyed to the mechanism as a single narrow precondition input on the write request, mirroring the existing narrow content-type field rather than introducing a general header bag. (The existing request descriptor already notes `If-Match` as the deferred consumer that would justify generalizing; this capability is that consumer, but a narrow field is assumed sufficient until a second precondition header lands.)

---

## Ambiguity Warnings

_None. The capability's scope is fixed by the Optimistic Concurrency roadmap (052 captures, 053 sends, 054 surfaces) and by 052's pinned verbatim-fidelity and empty-as-absent contracts, leaving no open behavioral question at this layer. The one decision deferred outward — whether an operator should be signalled when a write is unguarded because no version was available — belongs to the per-command retrofit and to 054, not to this mechanism._
