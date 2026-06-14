# Specification: Stale-Write Surfacing

**Feature**: 054-stale-write-surfacing
**Role**: Definer
**Tier**: 1 (zero setup)

---

## System Overview

Stale-Write Surfacing is a **Maintainer- and operator-facing capability** and the third and final step of the *Optimistic Concurrency* solution (the **Clobbered Changes** problem). Version Capture on Read (052) retains the resource version a read carried; Guarded Writes (053) sends it back as an `If-Match` precondition, so the server refuses a stale write with `412 Precondition Failed` instead of silently overwriting a concurrent change. That refusal is the signal this capability exists to make legible: 053 deliberately produces the `412` but does not interpret it, handing all surfacing of it here.

Today a `412` is invisible as a clobber. It flows through the existing failure pipeline — API Error Extraction (015) types it, Diagnostic Normalization (031) classifies it — and lands in the **generic non-2xx bucket** (general API error, exit code `3`), indistinguishable from a `404` or `500`. Worse, its generic next step points the operator at the wrong cause (*"check that the token has access"* — but the token is fine; the resource changed underneath). This capability branches the `412` out of that generic bucket into its **own distinct surfacing**: a distinct failure **category** carrying its **own process exit code**, a **cause** that names the stale-write, and a **next step** that tells the operator to re-read the resource for its current version and retry the write.

It is the consume half that completes the capture → send → surface split: 052 captures, 053 sends, 054 surfaces. Like its siblings it is **resource-agnostic and command-agnostic** — it classifies purely by the surfaced HTTP status (`412`), exactly as 015/031 classify `401`/`403`/`429`, so it is decoupled from 053 and from any specific write command. It deliberately stops at making the refusal legible: it does **not** perform the re-read, retry, or any recovery; it does **not** render the diagnostic (Output-Aware Failure Rendering, 032, owns that); and it does **not** decide *which* commands read-before-write (that is per-command retrofit work).

---

## Behavioral Accord

### Classifying the stale write

- When a failure carries the HTTP status `412 Precondition Failed`, the system classifies it under a distinct **stale-write** category — its own member of the fixed failure taxonomy — rather than the generic general-API-error category that every other non-`401`/`403`/`429` non-2xx falls into.
- The system classifies on the surfaced HTTP status alone, independent of which command failed, which resource it targeted, or whether this CLI sent an `If-Match` precondition — consistent with how the permission and rate-limited categories are assigned by status.

### A distinct process exit code

- When the outcome is classified as stale-write, the process exits with a new, previously-unused code reserved for this category in the single canonical category→code registry, so an operator branching on `$?` can detect a clobber specifically and react (re-read and retry) without parsing any text.
- The new code is added only in the registry, takes a previously-unused value, and renumbers no existing code — a consumer's existing branch on any current code keeps its meaning unchanged.

### Composing the cause and next step

- When the refusal is surfaced, the cause identifies it as a precondition failure caused by the resource changing since it was read — the operator reads *why* the write was refused, not a bare status number. When the API supplied its own error detail, that detail is surfaced (consistent with how every other typed API error surfaces the API's own words); when it did not, the cause is derived from the `412` status rather than invented.
- The next step tells the operator to re-read the resource to obtain its current version and retry the write — the actionable recovery for a clobber — replacing the misleading generic "check that the token has access" step that the `412` previously inherited from the generic bucket.

### Staying in its lane

- The system does not perform the re-read, retry, back off, or take any other recovery action; it classifies and explains, and the operator (or a future per-command flow) decides whether to re-read and retry.
- The system does not render or print the diagnostic in any `--output` format, nor emit the exit code itself; it supplies the category, cause, and next step into the existing pipeline that renders (032) and maps the code (004).
- The system changes the surfacing of no other HTTP status: every status that is not `412` keeps the exact category, cause, next step, and exit code it has today.

---

## User Scenarios

**In order to** detect that my guarded write was clobbered and react automatically — re-read the resource and retry — instead of treating it as an indistinct API error,
**as an** AI agent driving a write on a practitioner's behalf,
**I want** a `412 Precondition Failed` surfaced under its own exit code and with a next step that tells me to re-read and retry, so I can branch on `$?` without parsing text.

**In order to** understand why an edit I made was refused,
**as a** practitioner whose work the CLI serves,
**I want** the refusal explained as "the resource changed since you read it — re-read and retry" rather than a generic API error, so I know my change was protected, not broken.

**In order to** complete the Optimistic Concurrency capability so a refused write is finally legible end to end,
**as a** Maintainer building the Optimistic Concurrency solution,
**I want** the `412` that Guarded Writes (053) can produce surfaced distinctly through the one shared diagnostic pipeline, so capture → send → surface is whole.

---

## Non-Behaviors

- The system must not perform the re-read, retry, back off, or otherwise recover from the stale write. **Why**: it makes the refusal legible so the operator (or a future per-command read-then-retry flow) can act; silently retrying here would hide the very clobber the code exists to surface and would re-introduce the unbounded-recovery hazard the failure pipeline keeps out of the classifier.
- The system must not render, print, or format the diagnostic in any `--output` format. **Why**: rendering per `--output` is Output-Aware Failure Rendering's (032) job; this capability supplies the category, cause, and next step into the one diagnostic shape 032 renders — duplicating rendering here would split one contract across two owners.
- The system must not emit or decide the process exit code itself. **Why**: Exit-Code Convention (004) is the single category→code mapper; this capability assigns the stale-write category and registers its code there, and a second emitter would risk two paths disagreeing on the code.
- The system must not change the surfacing of any HTTP status other than `412` — no other status's category, cause, next step, or exit code changes. **Why**: this is an additive branch off the generic bucket; altering any other status would be an unrequested behavior change across the whole failure surface.
- The system must not renumber, reassign, or reuse any existing exit code; the stale-write code is a new, previously-unused value. **Why**: the exit-code convention is a published, frozen contract — renumbering would silently change the meaning of a consumer's existing branch on `$?`, the exact hazard 004's no-renumber rule forbids.
- The system must not condition its classification on whether this CLI sent an `If-Match` header, on the failed command, or on the targeted resource. **Why**: classifying by surfaced status alone keeps the capability decoupled from Guarded Writes (053) and from every write command, mirroring how `401`/`403`/`429` are classified — coupling it to a command or a sent header would make one shared mechanism into per-command logic.
- The system must not fabricate a cause or next step the failure does not support. **Why**: CONSTITUTION VIII (No Fabricated Data) and II — when the API supplied detail it is surfaced; when it did not, the cause is `412`-status-derived, never an invented specific reason.

---

## Integration Boundaries

- **Guarded Writes (053)** *(in-process, upstream)*: produces the `412 Precondition Failed` outcome when a captured version no longer matches the resource. This capability consumes that surfaced `412` and reports it distinctly; it does not reach into the write path that produced it.
- **API Error Extraction (015)** *(upstream)*: types the non-2xx `412` into a structured API error carrying the authoritative status `412`, the API's `detail`/`title` when present, and the raw body. This capability reads the typed `412` and never re-parses the body.
- **Diagnostic Normalization (031)** *(the seam extended)*: owns the status→category classifier and the cause/next-step composition. This capability adds the `412` branch there — a new category and a distinct next step — alongside the existing `401`/`403`/`429` arms, leaving every other status's arm untouched.
- **Exit-Code Convention (004)** *(the registry extended)*: the single category→code map. This capability registers the stale-write category against a new, previously-unused code, under 004's no-renumber extension rule.
- **Output-Aware Failure Rendering (032)** *(downstream)*: renders the normalized diagnostic — including this capability's stale-write diagnostic — per the selected `--output`. This capability produces the value, not the rendering; 032 needs no change because the diagnostic keeps the same three-field shape.

---

## Driving Scenarios

### Happy path

**Scenario: A stale write surfaces under its own category and exit code**
Given a guarded write the server refused with `412 Precondition Failed`
When the failure is surfaced
Then it is classified under the distinct stale-write category
And the process exits with the new code reserved for stale-write, not the generic API-error code
And an operator can tell a clobber apart from any other API error from `$?` alone.

**Scenario: The next step points to re-read and retry**
Given a `412 Precondition Failed` outcome
When the failure is surfaced
Then the next step tells the operator to re-read the resource for its current version and retry the write
And it is not the generic "check that the token has access" step the `412` previously inherited.

**Scenario: The cause names the stale write**
Given a `412` outcome whose API body carries a `detail`
When the failure is surfaced
Then the cause surfaces the API's own detail
And the failure is identified as a precondition failure from the resource changing since it was read.

### Error scenarios

**Scenario: A 412 with no readable detail derives its cause from the status**
Given a `412` outcome with no `detail` or `title` the API could supply
When the failure is surfaced
Then the cause is derived from the `412` status rather than invented
And the stale-write category, exit code, and re-read next step are still assigned.

**Scenario: Another non-2xx is unaffected**
Given a `404 Not Found` outcome (a non-`401`/`403`/`429` status that shares the generic bucket with `412` today)
When the failure is surfaced
Then it keeps the generic general-API-error category and exit code `3`
And only `412` is branched out — every other status's surfacing is unchanged.

### Edge cases

**Scenario: Classification ignores whether this CLI sent a precondition**
Given a `412` surfaced on a request this CLI did not attach an `If-Match` to
When the failure is surfaced
Then it is still classified as stale-write purely from the `412` status
And the classification does not depend on the command, the resource, or a sent header.

**Scenario: Adding the stale-write code renumbers no existing code**
Given the published set of exit codes (`0`–`6`)
When the stale-write category is registered
Then it takes a new, previously-unused code
And every existing category keeps the exact code it had before.

---

## Validation Scenarios

> These are held out from the implementing agent for independent verification.

**Scenario: 412 is distinct from the generic bucket**
Given a `412` outcome and a `500` outcome
When each is surfaced
Then the `412` carries the stale-write category and its own exit code while the `500` keeps the generic general-API-error category and code `3`, confirming the `412` is no longer indistinguishable from any other non-2xx.

**Scenario: The capability surfaces but does not recover**
Given a `412` outcome
When it is surfaced
Then no re-read, retry, sleep, or back-off occurs — only classification, a cause, a next step, and a code are produced.

**Scenario: No existing surfacing drifts**
Given the surfacing of `401`, `403`, `404`, `429`, and `500` before this capability
When the `412` branch is added
Then each of those statuses keeps its prior category, exit code, cause, and next step unchanged.

---

## Assumptions

- **`412` is the sole stale-write signal** *(technical)*: optimistic-concurrency refusal arrives only as HTTP `412 Precondition Failed` (the status the API documents for a failed `If-Match`), so classifying by that status alone is sufficient. (Grounded in the project's "Optimistic concurrency" constraint and 053's behavioral accord, which names `412` as the refusal a guarded write produces.)
- **[ASSUMED] Category and code names**: the failure category is referred to as "stale-write" and its exit code as the next unused value (`7`, following the contiguous operational band `3`–`6`). The exact category identifier and the numeric value are planning-time details settled against the existing registry; the behavior — a distinct category mapping to a new, previously-unused, never-renumbered code — is fixed.
- **Cause provenance mirrors the existing pipeline** *(technical)*: a `412` cause surfaces the API's own `detail`/`title` when present and otherwise a `412`-status-derived explanation, identical in provenance to how every other typed API error composes its cause today — only the category and next step are distinct to `412`.

---

## Ambiguity Warnings

_None. The capability's scope is fixed by the Optimistic Concurrency roadmap (052 captures, 053 sends, 054 surfaces) and by the existing failure pipeline it extends at two established seams (031's status→category classifier and 004's category→code registry). The one decision that was open — whether "distinct surfacing" includes a distinct process exit code or only distinct diagnostic text — was resolved during defining in favor of a new category and a new exit code (the architecturally-consistent reading that serves the agent-branches-on-`$?` model the CLI is built around). The remaining `[ASSUMED]` items are naming and numeric details settled at planning against the registry, not open behavioral questions._
