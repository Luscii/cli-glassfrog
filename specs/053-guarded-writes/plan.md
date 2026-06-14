# Plan: Guarded Writes

**Feature**: 053-guarded-writes
**Role**: Shaper
**Inputs**: `specs/053-guarded-writes/spec.md`, PROJECT.md, `.score/memory/DECISIONS.md` (precedent: 042 narrow-field `ContentType`, 052 verbatim/empty-as-absent capture, 039→040 mechanism/retrofit), `.score/memory/DEPRECATION.md`, `.score/memory/LEARNINGS.md`

---

## System Architecture

Guarded Writes is a single, small **request-side mechanism** in the `apiclient` package: a way to carry a captured resource version on a write request and have `Execute` send it as the `If-Match` precondition, so the server refuses a stale write (`412 Precondition Failed`) instead of overwriting it last-write-wins. It is the send half of the Optimistic Concurrency split — **Version Capture on Read (052)** already exposes the version verbatim via `Response.Version()`; this capability is the missing path that puts that value back on the wire.

The seam already exists and the precedent is exact. `apiclient.Request` carries a **narrow `ContentType` field** that `Execute` sets as the `Content-Type` header *only when non-empty* (042 ADR-1, `execute.go:128`); a bodyless read leaves it empty and the request stays byte-identical. 042 deliberately deferred a sibling `If-Match` field "until a real consumer lands" — Guarded Writes is that consumer. The mechanism is therefore one more narrow field, `Request.IfMatch string`, and one more conditional header-set in `Execute` mirroring the `ContentType` block: when `IfMatch` is non-empty, set the `If-Match` header to its value **verbatim**; when empty, send nothing and the write proceeds unconditionally. It is purely additive — `Execute`'s existing branches, every landed read, and every landed write stay byte-identical until a caller populates the new field. Consumption is deferred: the per-command read-then-write retrofit (Tension Update, Tension Discard, Proposal write-flow) is per-command work the FEATURE-MODEL places outside this capability, and distinct handling of the resulting `412` is **Stale-Write Surfacing (054)**. This plan delivers the field, the send, and their tests only.

---

## Architecture Decisions

### ADR-1: Carry the version as a narrow `Request.IfMatch` field, sent by `Execute` as `If-Match` only when non-empty

**Context**: The spec requires sending a captured version as an `If-Match` precondition on a write, **verbatim**, and sending **no** precondition when no version is present — without changing the outbound request of any existing read or write. `Request` already models per-call header needs with the narrow `ContentType` field; the question is how the version reaches the wire and in what shape.

**Options considered**:
1. **Narrow `IfMatch` field on `Request`, set by `Execute` when non-empty** — mirrors `ContentType` exactly: one field, one guarded `Header.Set`. Zero stored state, every existing call site stays byte-identical (the field defaults to `""`), method-agnostic by construction. Disadvantage: a second single-purpose field rather than a general header container.
2. **General `Header http.Header` bag on `Request`** — one extensible field for any header. Disadvantage: over-generalizes for a single new header against the codebase's anti-speculation idiom (042 explicitly chose the narrow field and deferred the bag); invites callers to set transport headers the layer means to own (e.g. `X-Auth-Token` lives in 007's `AuthTransport`).
3. **A dedicated conditional-write method/wrapper** (`ExecuteIfMatch`) — a parallel send path. Disadvantage: forks the single `Execute` seam (010 ADR-1) every command calls through, duplicating the build-and-send logic for one header.

**Decision**: Option 1. Add `IfMatch string` to `apiclient.Request`. In `Execute`, immediately after the `ContentType` block, add `if req.IfMatch != "" { httpReq.Header.Set("If-Match", req.IfMatch) }`. The value is sent **verbatim** — no quoting, unquoting, weak-validator (`W/"…"`) handling, or normalization — because the precondition must echo the server's token byte-for-byte (the value 052 took care to capture unchanged) or risk a spurious `412`. Empty `IfMatch` sets no header, so the write proceeds unconditionally, exactly as today — the empty-as-absent symmetry with 052's `Version()` sentinel. The set is **method-agnostic**: it depends only on the field being non-empty, so a `DELETE` (Tension Discard) is guarded the same way as a `PUT`/`PATCH`; the caller is responsible for only populating `IfMatch` on requests it intends to guard.

**Consequences**: `Execute` stays a single send seam; the addition is two lines symmetric with the `ContentType` handling a reviewer already knows. Every landed read and write is byte-identical until a caller sets `IfMatch` (the field zero-values to `""`). The narrow field resolves 042's deferred "add `If-Match` when a real consumer lands" rather than opening the general header bag — that generalization waits for a *second* request header to justify it. `If-Match` and `Content-Type` are independent fields, so a JSON-bodied guarded write carries both with neither displacing the other.

### ADR-2: Deliver the send mechanism only — wire no production write command, defer the `412` to 054

**Context**: The Optimistic Concurrency roadmap splits the solution into capture (052) → send (053) → surface-refusal (054), and the FEATURE-MODEL states plainly that the write commands "opt into" this Client-Foundation mechanism and that "retrofitting each write call-site is per-command work, not a capability here." 053 could either (a) ship just the field + send, or (b) also retrofit `tension update`/`tension discard` to read-then-guard now.

**Options considered**:
1. **Mechanism only; each command retrofits separately** — deliver `Request.IfMatch`, the `Execute` send, and tests; no production command is wired. Matches the 039 (resolver mechanism) → 040 (call-site retrofit) and 052 (capture mechanism, no call site) precedents.
2. **Retrofit the existing write commands now** — have Tension Update/Discard perform a pre-write read, call `resp.Version()`, and pass it as `IfMatch`. Disadvantage: each retrofit is its own behavioral change (an extra read per write, the read-failure path, whether an unguarded write should be signalled) with its own scenarios and tests — bundling them inflates this spec and couples the shared mechanism to specific commands, exactly the split the FEATURE-MODEL draws.

**Decision**: Option 1. This spec adds the `IfMatch` field, the conditional send in `Execute`, and their tests, and nothing else. It interprets nothing on the way back: a refused guarded write returns the existing generic `*ResponseError{StatusCode: 412, …}` that every non-2xx already produces (`execute.go:151`), surfaced through the landed diagnostic and exit-code path (004/015) — distinctly reporting the clobber so the operator can re-read is **Stale-Write Surfacing (054)**, which refines that `412`. The read-then-write flow that supplies `IfMatch` belongs to each write command's own retrofit.

**Consequences**: 053 is a tiny, low-risk, behavior-preserving addition with no externally observable change until a command opts in — an intentional foundation, not dead code, with its consumers named on the roadmap. The clean capture (052) → send (053) → surface (054) boundary is preserved, so no spec has to reason about another's surface. The mechanism produces the `412` condition without owning its presentation, leaving 054 a clean seam to build on.

---

## Cross-cutting Concerns

**Error handling**: None introduced. Setting a header cannot fail. A guarded write the server refuses comes back as the existing generic `*ResponseError` (status `412`), already carried with its status, headers, and body for downstream refinement; 053 neither classifies nor relabels it. Existing transport/decode/non-2xx handling is untouched.

**Testing**: Unit tests on `Execute` against a fake transport that captures the outbound `*http.Request`: (1) `IfMatch` non-empty → outbound carries `If-Match` equal to the value verbatim; (2) `IfMatch` empty → no `If-Match` header present; (3) weak-validator/quoted token (`W/"abc"`, `"abc"`) → sent byte-for-byte; (4) `IfMatch` set on a `DELETE` → header present (method-agnostic); (5) `IfMatch` + `ContentType` both set → both headers present and independent. Because 053 changes no CLI command and no output, the spec's driving scenarios exercise the **seam** (the request-side send contract), placed as mechanism-level Gherkin, with a guard-style assertion that no existing read/write becomes non-byte-identical.

**Configuration**: Nothing configurable — the precondition header (`If-Match`) and its source (the captured version) are fixed by the API contract.

---

## Implementation Strategy

**Single phase.** Add `IfMatch string` to the `Request` struct in `internal/apiclient/client.go` with a doc comment stating the verbatim contract, the empty-as-absent (unconditional) sentinel, the method-agnostic behavior, and that the write commands (via their own retrofits) are the intended consumers. Add the conditional `If-Match` set in `Execute` (`execute.go`) immediately after the `ContentType` block. Add the unit tests above. One PR; no dependencies on other in-flight specs beyond 052 (already landed); no existing call site changes.

---

## Risks

- **Mechanism lands before any command consumes it** — the `IfMatch` field is unset by every existing caller until a write command retrofits. *Likelihood*: certain (by design). *Impact*: low — a single derived field and a two-line guarded set, no behavior change; a reviewer might flag "unused field." *Mitigation*: the doc comment names the consuming write commands and the roadmap dependency (`053 → requires 052`, write-command retrofits → require 053); the tests prove the send contract independently of any consumer (mirrors 052 ADR-2 / the 039→040 split).
- **A consumer normalizes the token before setting `IfMatch`** — if a future retrofit unquotes or strips the value, the server may reject a valid precondition with a spurious `412`. *Likelihood*: low. *Impact*: medium. *Mitigation*: `Execute` forwards `IfMatch` straight to `Header.Set` with no transformation, and 052 fixed the verbatim guarantee at the capture point, so a consumer that simply threads `Version()` → `IfMatch` is correct by default; the verbatim test pins it here too.
- **Premature header-bag generalization** — pressure to add a general `Header` bag instead of the narrow field. *Likelihood*: low. *Impact*: low. *Mitigation*: ADR-1 keeps the narrow field per the 042 precedent; the bag waits for a genuine *second* request header to justify it.

---

## What This Plan Does Not Cover

- **The read-then-write retrofit of each write command** (Tension Update 044, Tension Discard 045, Proposal write-flow) — per-command work that consumes `Version()` and populates `IfMatch`, explicitly outside this capability per the FEATURE-MODEL.
- **Surfacing a refused write (`412 Precondition Failed`) distinctly** — Stale-Write Surfacing (054).
- **The seam's protocol-level contract** (the `IfMatch` field semantics, the verbatim/empty-as-absent send guarantee, the no-interpretation-of-`412` boundary) — the interface skill records it in `interface-spec.md`.
- **Executable scenario placement** — the scenarios skill turns the spec's seam-level driving scenarios into Gherkin.
- **Any user-facing output, flag, or command change** — excluded by the spec's non-behaviors; 053 is internal request-side plumbing only.
