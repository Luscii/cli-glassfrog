# Plan: Stale-Write Surfacing

**Feature**: 054-stale-write-surfacing
**Role**: Shaper
**Inputs**: `specs/054-stale-write-surfacing/spec.md`, PROJECT.md, `.score/memory/DECISIONS.md` (precedent: 004 single `ExitCode(Outcome)` registry / never-renumber / new category → new unused code; 011 added `APIError`+`NetworkUnavailable` to the live enum; 015 split `APIError` into permission/rate-limit by status at `categoryForStatus`; 031 cause/next-step composition), `.score/memory/DEPRECATION.md`, `.score/memory/LEARNINGS.md`

---

## System Architecture

Stale-Write Surfacing is a small, **additive classification change** in the `internal/cli` package: it branches the `412 Precondition Failed` outcome out of the generic non-2xx bucket into its own failure category with its own process exit code, a re-read-and-retry next step, and a cause that names the stale write. It is the surface half of the Optimistic Concurrency split — **Guarded Writes (053)** already produces the `412` (a refused `If-Match` write returns the generic `*ResponseError{StatusCode: 412}`); today that `412` falls through `categoryForStatus`'s `default` arm to `APIError` (exit `3`), indistinguishable from a `404` or `500`, and inherits the misleading generic next step (*"check that the token has access"*).

The seam already exists and the precedent is exact. The failure pipeline is a producer-classifies / registry-maps chain: **API Error Extraction (015)** types the `412` into a `*ProblemError`; **Diagnostic Normalization (031)** classifies it via `categoryForStatus(status)` and composes the cause/next-step in `Diagnose` (`diagnostic.go`); **Exit-Code Convention (004)** maps the resulting `Outcome` to a process code via the single `ExitCode` registry (`exitcode.go`); **Output-Aware Failure Rendering (032)** renders the resulting `Diagnostic` per `--output`. 015 already set the precedent this capability follows: it split `APIError` into `PermissionError`(4)/`RateLimited`(5) by adding `401/403`/`429` arms to `categoryForStatus` and registering their codes — without renumbering. 054 is the identical move for `412`: one new `Outcome` value (`StaleWrite`), one new registry code (`7`), one new `categoryForStatus` arm, one new `nextStepForStatus` arm, and a `412`-aware cause. It is purely additive — every other status keeps its exact category, code, cause, and next step, and the rendering layer (032) needs no change because the `Diagnostic` keeps its three-field shape.

Consumption is upstream and already landed: 053 produces the `412`; 054 does not perform the re-read or retry it recommends (that is each write command's own retrofit), does not render (032 owns that), and does not emit the code itself (004's registry owns that). This plan delivers the category, the code, the classification, the cause/next-step, and their tests only.

---

## Architecture Decisions

### ADR-1: Surface the `412` as a distinct `StaleWrite` category mapped to a new exit code `7`

**Context**: The spec requires a `412` to be *machine-distinct* — its own failure category carrying its own process exit code — so an agent operator can branch on `$?` to detect a clobber and re-read/retry, rather than seeing the generic API-error code `3` it shares with every other uncategorized non-2xx today. This was the one decision resolved during defining (a distinct exit code, not text-only). The codebase models every failure class as an `Outcome` value (`dispatch.go`) mapped to a code by the single `ExitCode` registry (`exitcode.go`); the published convention is `0`–`6`.

**Options considered**:
1. **New `StaleWrite` Outcome + new registry code `7`** — mirrors how 015 added `PermissionError`(4)/`RateLimited`(5): one enum value, one constant, one `ExitCode` case, classified by status at `categoryForStatus`. Machine-distinct (`$?` == 7 means clobber), fully within 004's extension rule. Disadvantage: extends the published exit-code contract beyond the original `0`–`6` band (the first code to do so).
2. **Keep `APIError`(3), distinguish by text only** — give `412` a distinct cause/next-step but leave the code at `3`. Disadvantage: not machine-distinct — an agent cannot tell a clobber from a `404`/`500` via `$?`, under-delivering the spec's primary (agent-branches-on-`$?`) user scenario. Rejected at defining.
3. **A separate signal outside the exit-code registry** (e.g. a marker on stderr) — Disadvantage: forks the one outcome signal 004 owns, and an agent would have to parse text — exactly what the exit-code convention exists to avoid.

**Decision**: Option 1. Add `StaleWrite` to the `Outcome` enum in `dispatch.go` (a new `iota` value after `RateLimited`) with its `String()` case, add `codeStaleWrite = 7` to `exitcode.go` with its `ExitCode` case, and add a `case http.StatusPreconditionFailed: return StaleWrite` arm to `categoryForStatus` in `diagnostic.go`. Classification is **status-driven only** — it depends solely on the surfaced `412`, never on whether this CLI sent an `If-Match`, which command failed, or which resource it targeted — exactly mirroring the `401/403/429` arms, so the capability stays decoupled from 053 and every write command. Code `7` is the next previously-unused value, extending the contiguous operational band (`3`–`6` → `7`); no existing code is renumbered or reassigned, so a consumer's existing branch on any current code keeps its meaning.

**Consequences**: `412` becomes distinguishable in `$?`. The change is the same shape a reviewer already knows from 015's split, at the same three sites. `7` is the first code added beyond 004's originally-published `0`–`6` set — but this is precisely the extension mechanism 004 designed (new category → single registry site → new unused code → never renumber), so it is anticipated growth, not a contract break. `exitcode_test.go`'s uniqueness / no-shell-reserved / no-renumber pins must be extended to cover `7`. The `ExitCode` default arm still returns `1`, so an unmapped future category can never exit `0`.

### ADR-2: Compose a `412`-specific cause and next step at the existing `Diagnose` composition site

**Context**: The spec requires the cause to identify the failure as a precondition failure caused by the resource changing since it was read, and the next step to tell the operator to re-read the resource for its current version and retry — replacing the generic `default` next step (*"the API rejected the read; check that the token has access…"*), which is wrong for a stale write (it was a write, not a read; the token is fine). 031 composes the cause via `problemCause` (which surfaces the API's own `detail` when present and keys on `DetailSynthesized` for the fallback) and the next step via `nextStepForStatus(status)`.

**Options considered**:
1. **Add a `412` arm to `nextStepForStatus`, and make the synthesized cause `412`-aware** — the next step says re-read + retry; the cause surfaces the API's own `detail` when present (unchanged `problemCause` behavior) and, when the API supplied none, uses a `412`-specific status-derived fallback naming the precondition failure instead of the bare "status 412". Mirrors the per-status `nextStepForStatus` arms 031 already has for `401/403/429`.
2. **Distinct category + code only; leave cause and next step generic** — Disadvantage: the operator still reads the misleading "check that the token has access" step and a cause that doesn't name the clobber, failing the spec's cause/next-step accord and CONSTITUTION II (explain what went wrong *and the next step*).

**Decision**: Option 1. Add `case http.StatusPreconditionFailed:` to `nextStepForStatus` returning a re-read-and-retry hint (the operator re-reads the resource to obtain its current version, then retries the write). For the cause: keep surfacing the API's own `detail`/`title` when present (the existing `problemCause` non-synthesized path is unchanged — the spec says the API's own words win), and when the body carried no readable detail (`DetailSynthesized`), produce a `412`-specific status-derived cause naming the precondition failure / resource-changed-since-read rather than the bare generic "status 412" line. No cause or next step is fabricated — the API's detail is preferred, and the fallback is derived from the well-defined `412` semantics, never invented. Exact wording is interface/Builder detail within this contract.

**Consequences**: A stale write reads as a stale write end to end — distinct code, a cause that names the precondition failure, and an actionable re-read/retry step — while every other status's cause and next step are untouched (the `default` arms are unchanged; only a new `412` arm is added). The defensive bare-`*ResponseError` arm in `Diagnose` (the unrefined path that `reportClientError` does not normally reach) still gets the correct category and next step via the shared `categoryForStatus`/`nextStepForStatus` switches; only its cause stays the generic status line, which is acceptable for that already-defensive fallback.

---

## Cross-cutting Concerns

**Error handling**: None introduced. This is classification of an already-typed failure — no new I/O, no new failure mode. The `412` continues to arrive as the existing `*ProblemError`/`*ResponseError`; 054 only changes which category/code/cause/next-step it maps to.

**Testing**: Unit tests in `internal/cli`. `diagnostic_test.go`: a `412` `*ProblemError` → `Diagnose` returns `Category: StaleWrite`, the re-read/retry next step, and a cause that surfaces the API `detail` when present / names the precondition failure when synthesized; assert a `404` and a `500` still map to `APIError` with the generic next step (no drift). `exitcode_test.go`: extend the pins so `StaleWrite → 7`, `7` is unique (no category collision), `7` is not shell-reserved (`126/127/128+N`), and no existing code (`0`–`6`) is renumbered. A table-driven exit-code test must guard against the missing-key-equals-zero-value trap — assert the full category set with comma-ok lookups, not a value-only index (LEARNINGS / user review precedent). Because 054 changes no CLI command and no output rendering, the spec's driving scenarios exercise the **classification seam** (status → category/code/cause/next-step) as mechanism-level Gherkin, with a guard-style assertion that no other status's surfacing changes.

**Configuration**: Nothing configurable — the `412` status, its category, and its code are fixed by the API contract and the published exit-code convention.

---

## Implementation Strategy

**Single phase.** One reviewable change across three sibling files in `internal/cli`, plus tests:
1. `dispatch.go` — add `StaleWrite` to the `Outcome` enum (new `iota` value after `RateLimited`) with a doc comment naming `412`/Stale-Write Surfacing as its producer, and its `String()` case.
2. `exitcode.go` — add `codeStaleWrite = 7` with a doc comment, and the `case StaleWrite: return codeStaleWrite` arm in `ExitCode`.
3. `diagnostic.go` — add the `http.StatusPreconditionFailed` arms to `categoryForStatus` (→ `StaleWrite`) and `nextStepForStatus` (→ re-read/retry), and the `412`-aware synthesized cause.
4. Tests — extend `exitcode_test.go` (uniqueness / no-renumber / no-shell-reserved pins to cover `7`) and `diagnostic_test.go` (the `412` classification + cause/next-step, and the no-drift assertions for other statuses).

One PR; no dependencies on other in-flight specs (053 is landed and already produces the `412`); no existing call site changes beyond the three classification sites and their pinning tests.

---

## Risks

- **Extending the published exit-code convention beyond `0`–`6`** — `7` is the first code added outside 004's originally-published band; a consumer assuming "codes are `0`–`6`" could be surprised. *Likelihood*: low. *Impact*: low — the convention's own extension rule (new category → new unused code, never renumber) anticipates exactly this, and every existing code keeps its meaning. *Mitigation*: ADR-1 takes the next unused code and renumbers nothing; the interface accord publishes `7` in the canonical registry table; `exitcode_test.go` pins uniqueness and the no-renumber guarantee.
- **The `412` arm drifts another status's surfacing** — a careless edit to `categoryForStatus`/`nextStepForStatus`/`problemCause` could change a `404`/`500`/`401` path. *Likelihood*: low. *Impact*: medium. *Mitigation*: the arms are additive `case` labels on existing switches (the `default` is untouched); `diagnostic_test.go` asserts the unchanged statuses explicitly.
- **Table-driven exit-code test misses a dropped category** (the zero-value trap) — a value-only map index passes silently if `StaleWrite` were dropped and its expected code were the zero value. *Likelihood*: low (code `7` ≠ 0, so this specific entry is safe, but the test pattern matters). *Impact*: medium. *Mitigation*: assert the full set with `len` + comma-ok lookups, per the LEARNINGS precedent — fail loudly on a missing key.
- **Cause over-reach** — synthesizing a `412` cause that asserts more than the status supports (e.g. naming *who* changed the resource). *Likelihood*: low. *Impact*: low. *Mitigation*: the cause surfaces the API's own detail when present and otherwise states only the precondition-failed / changed-since-read semantics the `412` status defines — no fabrication (CONSTITUTION VIII).

---

## What This Plan Does Not Cover

- **The per-command read-then-write retrofit** (Tension Update, Tension Discard, Proposal write-flow) that actually produces a guarded write — per-command work outside this capability; 054 surfaces the `412` whatever command produced it.
- **Rendering the stale-write diagnostic per `--output`** — Output-Aware Failure Rendering (032) already renders any `Diagnostic`; the new category needs no rendering change because the value keeps its three-field shape.
- **The published-contract record** (the new code `7`, the `StaleWrite` category, the re-read/retry next-step text, the cause provenance) — the interface skill records it in `interface-cli.md`.
- **Executable scenario placement** — the scenarios skill turns the spec's classification-seam driving scenarios into Gherkin under `features/clobbered-changes/`.
- **Any new CLI command, flag, or output format** — excluded by the spec's non-behaviors; 054 is an internal classification change surfaced through the existing diagnostic/exit-code/rendering pipeline.
