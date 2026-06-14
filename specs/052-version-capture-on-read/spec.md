# Specification: Version Capture on Read

**Feature**: 052-version-capture-on-read
**Role**: Definer
**Tier**: 1 (zero setup)

---

## System Overview

Version Capture on Read is a **Maintainer-facing mechanism** and the dependency root of the *Optimistic Concurrency* solution (the **Clobbered Changes** problem). The Glassfrog API v5 returns an `ETag` header on a single-resource read — a fingerprint of that resource's current state. Sending it back on a later `PUT`/`PATCH` via `If-Match` lets the server refuse a stale write instead of silently overwriting it. Today the CLI throws that fingerprint away: the request-execution layer (010) already exposes the full response header set, but no read captures the `ETag`, so the value a guarded write would need is never retained. This capability closes that first gap — it captures the version that a single-resource read carries and makes it available, in-process, on the read's result.

It is a pure plumbing capability: it changes **no user-facing output** and sends **no precondition**. It captures and exposes; nothing more. The version it retains is consumed downstream by **Guarded Writes (053)**, which sends it via `If-Match` so the server can reject a stale write, and **Stale-Write Surfacing (054)**, which reports the resulting `412`. This spec deliberately stops at capture: it is setting-agnostic and resource-agnostic, retaining whatever opaque version token the server returned without interpreting it, so that any single-resource read — a tension today, a role tomorrow — can feed a future guarded write through one shared mechanism rather than per-command copies.

---

## Behavioral Accord

### Capture

- When the CLI performs a single-resource read and the response carries a version indicator (the `ETag` header), the system retains that version verbatim on the read's in-process result, available for a subsequent operation on the same resource to consume.
- When the response carries no version indicator, the system retains no version and the read proceeds and renders exactly as it does today.

### Fidelity

- When a version is captured, it is a faithful copy of what the server returned — the system does not parse, unquote, normalize, or otherwise interpret the token.

### Scope

- When the read returns a collection of resources (a list), the system captures no per-resource version, because a collection-level fingerprint cannot guard an individual resource's write.
- When a read fails (a non-success outcome such as not-found or unauthorized), the system captures no version and leaves existing diagnostic and exit-code handling unchanged.

---

## User Scenarios

**In order to** keep a foundation in place that lets later edits be guarded against intervening changes,
**as a** Maintainer building the Optimistic Concurrency capability,
**I want to** capture the version a read already carries, so a guarded write has a value to send without each write command re-deriving it.

**In order to** avoid silently clobbering a concurrent governance edit on my behalf,
**as a** practitioner whose work the CLI serves,
**I want to** have the resource version retained at read time, so the eventual write can detect that the resource changed under me.

---

## Non-Behaviors

- The system must not send an `If-Match` precondition or otherwise enforce optimistic concurrency. **Why**: enforcement is Guarded Writes (053); folding it in here would couple the capture mechanism to a write path and blur the roadmap's clean capture-then-send split.
- The system must not surface the captured version in any user-facing output — `--output json`, `yaml`, `full`, and `compact` render exactly as before. **Why**: per the chosen scope this is internal plumbing consumed in-process by a guarded write, not by the operator; adding a field or line would change a settled read contract for no operator benefit and would have to be designed and tested as its own change.
- The system must not interpret, validate, or normalize the version token. **Why**: the `ETag` is an opaque server fingerprint (it may be quoted or a weak validator); faithfully echoing it is the only behavior that keeps a later `If-Match` matching what the server expects — anything else risks a spurious mismatch.
- The system must not capture a version from collection/list reads. **Why**: the API's `ETag` on a collection response fingerprints the whole collection, not any one item, so retaining it would invite guarding a single-resource write with a value that cannot apply to it.
- The system must not fail or alter a read when no version is present. **Why**: many responses legitimately omit an `ETag`; absence is normal, not an error, and a read's success must not depend on a concurrency token it does not need.
- The system must not persist the captured version across CLI invocations (no caching to disk or config). **Why**: a version is only valid against the resource state read in this process; a persisted version would be stale by the next invocation and would defeat the very guard it exists to enable.

---

## Integration Boundaries

- **Glassfrog API v5** *(upstream)*: provides the `ETag` response header on single-resource reads of mutable resources. The CLI reads it; it never asks for it specially. When the header is absent, capture is simply empty.
- **Request Execution (010)** *(internal seam)*: already exposes the full response header set on its result. Version capture reads the `ETag` from that existing surface — no new transport behavior is required.
- **Guarded Writes (053)** *(downstream consumer)*: consumes the captured version, sending it via `If-Match` on an edit. This spec only makes the value available; it does not reach into the write path.
- **Single-resource read commands** *(consumers)*: any read that returns one resource (a tension today; a role, since its GET also carries an `ETag`) is a point where a version can be captured.

---

## Driving Scenarios

### Happy path

**Scenario: Version captured from a single tension read**
Given a single-resource read of a tension whose response includes an `ETag` header
When the CLI completes the read
Then the read's result carries the captured version
And that version equals the `ETag` value the server returned

**Scenario: Mechanism is resource-agnostic**
Given a single-resource read of a role whose response includes an `ETag` header
When the CLI completes the read
Then the read's result carries the captured version
And the capture behaves identically to the tension case, with no resource-specific handling

**Scenario: Captured version does not change read output**
Given a single-resource read whose response includes an `ETag` header
When the CLI renders the read in any output format
Then the rendered output is byte-for-byte what it was before version capture existed
And the captured version is present only on the in-process result, not in the rendered output

### Error scenarios

**Scenario: No version present on the response**
Given a single-resource read whose response omits the `ETag` header
When the CLI completes the read
Then no version is captured
And the read still succeeds and renders normally

**Scenario: Failed read captures nothing**
Given a single-resource read that the server rejects (for example, not-found or unauthorized)
When the CLI handles the failure
Then no version is captured
And the existing diagnostic message and exit code are unchanged

### Edge cases

**Scenario: Collection read yields no per-resource version**
Given a read that returns a list of resources whose response carries a collection-level `ETag`
When the CLI completes the read
Then no per-resource version is captured for any item in the list

**Scenario: Version token is captured verbatim**
Given a single-resource read whose `ETag` is a quoted or weak-validator token (for example `W/"abc123"`)
When the CLI captures the version
Then the captured value is exactly the token the server sent, including its quotes or weak-validator prefix
And the system performs no stripping, unquoting, or normalization

---

## Validation Scenarios

> These are held out from the implementing agent for independent verification.

**Scenario: Read contract is provably unchanged**
Given the read commands that exist before this capability
When version capture is added
Then a reader of the spec can confirm no user-facing output, exit code, or diagnostic changes — capture is additive and internal only

**Scenario: No precondition leaks into a request**
Given the capture mechanism
When any read or any existing write executes
Then no `If-Match` header is sent by anything this capability introduces, confirming the capture-then-send boundary with Guarded Writes (053) holds

---

## Assumptions

- **In-process consumption** *(technical)*: the captured version is consumed within the same CLI process that read it — the natural shape is a guarded write (053) that reads the resource, captures its version, and immediately writes with `If-Match`. (Informed by the chosen "internal seam only, no user-facing surface" scope plus the no-persistence non-behavior: with nothing surfaced and nothing persisted, the value cannot be carried across separate invocations, so the consuming write must perform its own read in-process. The exact read-then-write flow is finalized in 053.)
- **`ETag` is the sole version indicator** *(technical)*: the resource version lives only in the `ETag` response header, not in any body field — Glassfrog v5 resources carry no `version`/`rev` field. (Grounded in `spec/glassfrog-api-v5.yaml`'s "Optimistic Concurrency (ETags)" section.)
- **Capture is driven by header presence, not a mutability classification** *(technical)*: any single-resource read whose response carries an `ETag` yields a captured version, regardless of whether that resource is directly writable today. (Per the chosen scope; harmless for read-only resources and avoids maintaining a separate list of "mutable" types.)

---

## Ambiguity Warnings

_None — the capture-then-send boundary with Guarded Writes (053) and the internal-only scope are settled; the in-process consumption flow is documented as an assumption to be finalized in 053, not an open question for this spec._
