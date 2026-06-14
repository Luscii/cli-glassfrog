# Plan: Version Capture on Read

**Feature**: 052-version-capture-on-read
**Role**: Shaper
**Inputs**: `specs/052-version-capture-on-read/spec.md`, PROJECT.md, `.score/memory/DECISIONS.md` (precedent: 042 narrow-field, 039→040 mechanism/retrofit), `.score/memory/DEPRECATION.md`, `.score/memory/LEARNINGS.md`

---

## System Architecture

Version Capture on Read is a single, small **read-side mechanism** in the `apiclient` package: a way to read the resource version (the `ETag` response header) off a single-resource read's result and hand it back verbatim, so a later guarded write can supply it as `If-Match`. It owns no setting of its own, sends nothing, and renders nothing.

The seam already exists. `apiclient.Client.Execute(reqCtx, req, out) (*Response, error)` returns a `*Response{StatusCode, Header}`, and that `*Response` already propagates up through the `executor` interface and the `RetryExecutor` wrapper to every command — single-read commands simply discard it today (`if _, err := exec.Execute(...)`). The `ETag` is therefore already in scope at the consumer; what is missing is a *named, tested* way to extract it that future write commands can depend on without each re-deriving `Header.Get("ETag")` with its own assumptions about the header name and fidelity. This mirrors the duplication that **Source-Composed Resolution (039)** centralized.

The mechanism is one accessor on the read result: `apiclient.Response.Version() string` — returns the `ETag` header verbatim, or empty string when the response carries no `ETag`. It is purely derived from the `Header` the `*Response` already holds: it adds no stored state, changes nothing in `Execute`, sends no header, and alters no rendered output. Its single contract is "give me the captured version of the resource this read returned, exactly as the server stated it." The list-walk path (the pagination walker / `aggregateRawData`) has no version seam and is untouched, so collection reads yield no per-resource version by construction. Consumption is deferred: **Guarded Writes (053)** will call this accessor on its in-process pre-write read and send the value as `If-Match`; this plan delivers the accessor and its tests only.

---

## Architecture Decisions

### ADR-1: Capture the version as a verbatim accessor on `apiclient.Response`

**Context**: The spec requires capturing the resource version (the `ETag`) on a single-resource read and making it available in-process, with **no user-facing output change** and **no interpretation** of the token. `Response` already carries the full `http.Header`; the question is where the captured version lives and in what form.

**Options considered**:
1. **Accessor on `Response` (`Version() string`)** — a derived getter reading `Header.Get("ETag")`. Zero stored state, `Execute` stays byte-identical, lazy and caller-driven. Disadvantage: the consumer must hold the `*Response` (it does — Execute returns it).
2. **New `Version`/`ETag` field on `Response`, populated in `Execute`** — eager capture. Disadvantage: adds stored state and an `Execute` code change for a value that is trivially derivable from `Header`, against the codebase's anti-speculation idiom.
3. **Surface on a `glassfrog` domain wrapper around the decoded body** — attach the version to the rendered model. Disadvantage: the `ETag` is transport metadata, not a body field; a domain wrapper invites rendering it, which the spec's non-behavior forbids.

**Decision**: Option 1 — an accessor on `Response`. `func (r *Response) Version() string` returns `r.Header.Get("ETag")`. `Header.Get` is case-insensitive (canonicalized), so `ETag`/`Etag`/`etag` all match. The value is returned **verbatim** — no unquoting, no weak-validator (`W/"…"`) stripping, no normalization — because a later `If-Match` must echo exactly what the server sent or risk a spurious mismatch. Absent header → empty string, which the consumer reads as "no version captured." The accessor is the seam contract 053 depends on; centralizing the header name and the verbatim guarantee here keeps future guarded writes from each re-deriving them.

**Consequences**: `Execute` and all existing reads stay byte-identical (the accessor is purely additive and only invoked by a future consumer). Transport metadata stays in the transport layer; the `glassfrog` domain package and the render path never see the version, satisfying the no-output non-behavior for free. The empty-string sentinel means "absent" is not distinguishable from "present but empty," which is acceptable: an empty `ETag` is not a usable precondition and 053 treats both as "nothing to guard with."

### ADR-2: Deliver the mechanism only — wire no call site, defer consumption and the `If-Match` request field to 053

**Context**: The roadmap splits Optimistic Concurrency into capture (052) → send (053) → surface-refusal (054). 052 could either (a) ship just the accessor, or (b) also retrofit the existing single-read commands (`tension get`, `role get`) to capture the version now.

**Options considered**:
1. **Mechanism only; 053 wires it** — deliver the accessor + tests; existing reads keep discarding `*Response`. Matches the 039 (resolver) → 040 (call-site retrofit) precedent.
2. **Retrofit existing single reads now** — have `tension get`/`role get` capture the version after `Execute`. Disadvantage: each CLI invocation is one process that exits immediately after rendering; with nothing consuming the value and nothing surfaced (non-behavior) and nothing persisted (non-behavior), the captured version evaporates unused — dead state, against the anti-speculation idiom.

**Decision**: Option 1. This spec adds the `Version()` accessor and its tests and nothing else. The only sensible consumer is an in-process read-then-write, which is **Guarded Writes (053)**: 053 will perform its pre-write read, call `resp.Version()`, and send the result as `If-Match`. Consistent with the **042 narrow-field precedent** ("generalize `ContentType`→`Header`/add `If-Match` when a real consumer lands"), 052 does **not** add an `If-Match` field to `apiclient.Request` — that request-side change belongs to 053, the consumer that grounds it.

**Consequences**: 052 is a tiny, zero-risk, behavior-preserving addition with no externally observable change. The accessor is unused until 053 lands — an intentional foundation, not dead code, with its next consumer named on the roadmap. The clean capture-then-send boundary (052 reads, 053 writes) is preserved, so neither spec has to reason about the other's surface.

---

## Cross-cutting Concerns

**Error handling**: None introduced. The accessor cannot fail — `Header.Get` returns `""` for an absent header. A *failed* read returns an error from `Execute` and no `*Response`, so there is nothing to call `Version()` on; existing diagnostic and exit-code handling is untouched.

**Testing**: Unit tests on the accessor against a constructed `Response`: (1) `ETag` present → returned verbatim; (2) `ETag` absent → empty string; (3) weak-validator/quoted token (`W/"abc"`, `"abc"`) → returned byte-for-byte with quotes/prefix intact; (4) header-name case-insensitivity. Because 052 changes no CLI command and no output, the spec's driving scenarios exercise the **seam** (the accessor contract), not a command — the scenarios skill places them as mechanism-level Gherkin, and a guard-style assertion confirms no `If-Match` is sent by anything this spec introduces.

**Configuration**: Nothing configurable — the version source (`ETag`) is fixed by the API contract.

---

## Implementation Strategy

**Single phase.** Add `func (r *Response) Version() string` to `internal/apiclient` (alongside the `Response` type in `execute.go`) with a doc comment stating the verbatim contract, the empty-on-absence sentinel, and that 053 is the intended consumer. Add the unit tests above. One PR; no dependencies on other in-flight specs, and no existing call site changes.

---

## Risks

- **Foundation lands before its consumer** — the accessor is unused until 053. *Likelihood*: certain (by design). *Impact*: low — it is a few derived lines with no behavior change, and a reviewer might flag "unused." *Mitigation*: the doc comment names 053 as the consumer and the roadmap dependency (`053 → requires 052`) records the intent; the tests prove the contract independently of any consumer.
- **A future guarded write strips or normalizes the token** — if 053 (or a maintainer) unquotes the value before sending `If-Match`, the server may reject a valid precondition. *Likelihood*: low. *Impact*: medium (spurious 412s). *Mitigation*: the verbatim guarantee is fixed and tested *here*, at the capture point, so the value 053 receives is already correct to forward unchanged.

---

## What This Plan Does Not Cover

- **Sending `If-Match` / guarded writes** — Guarded Writes (053). Includes the `apiclient.Request` `If-Match` field (deferred here per the 042 narrow-field precedent) and the read-then-write flow that consumes `Version()`.
- **Surfacing a refused write (`412`)** — Stale-Write Surfacing (054).
- **The seam's protocol-level contract** (accessor signature, return semantics, the no-`If-Match` guarantee) — the interface skill records it in `interface-spec.md`.
- **Executable scenario placement** — the scenarios skill turns the spec's seam-level driving scenarios into Gherkin.
- **Any user-facing output, flag, or command change** — explicitly excluded by the spec's non-behaviors; 052 is internal plumbing only.
