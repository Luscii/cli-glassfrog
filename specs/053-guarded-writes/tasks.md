# Tasks: Guarded Writes

**Feature**: 053-guarded-writes
**Concretization**: Full context (plan + spec + interface-spec + scenarios)
**Inputs**: plan.md, spec.md, interface-spec.md, features/clobbered-changes/guarded-writes.feature

---

## Dependency Graph

Phase 1: If-Match Send (1 task, no phase dependencies) — single-phase build

1 task total | 1 phase

---

## Branching Guidance

**Pipeline mode**: `spec/053-guarded-writes/base` → `spec/053-guarded-writes/task-1`

---

## Phase 1: If-Match Send [Shared]

- [ ] **T001** [Shared] Add the `Request.IfMatch` field and its conditional `If-Match` send in `Execute`, with unit tests — 9 scenarios (2 @validation held @wip), 5 send unit tests; no LEARNINGS, no drift
  - **Scope**: One reviewable change in `internal/apiclient`: add a narrow `IfMatch string` field to the `Request` struct (`client.go`, beside `ContentType`), and in `Execute` (`execute.go`), immediately after the `ContentType` header block, set the `If-Match` header to `req.IfMatch` **only when non-empty**. Adds no new type; no change to `Response`, the `executor` interface, `RetryExecutor`, `NewClient`, or `buildURL`; wires no production write command and interprets no response. The field plus the send plus their unit tests ship together (the send must not merge without the tests that pin its contract).
  - **Acceptance criteria**:
    - A request with a non-empty `IfMatch` produces an outbound `If-Match` header equal to that value **verbatim** — no quoting, unquoting, weak-validator (`W/"…"`) handling, or normalization.
    - A request with an empty/unset `IfMatch` produces **no** `If-Match` header; the write proceeds unconditionally (last-write-wins) — empty is indistinguishable from unset, and is not an error.
    - The send is method-agnostic: an `IfMatch` set on a `DELETE` produces the header just as a `PUT`/`PATCH` does (depends only on the field being non-empty).
    - `If-Match` and `Content-Type` are independent: a request with both `IfMatch` and `ContentType` set produces both headers, neither displacing the other.
    - Every landed read and write stays byte-identical — the field zero-values to `""`, so no existing call site sets it and no outbound request gains an `If-Match` header. No new `Outcome`, exit code, or diagnostic; a refused write (`412`) rides the existing generic `*ResponseError` path unchanged (distinct surfacing is 054).
    - Unit tests cover: `IfMatch` non-empty → header present verbatim; `IfMatch` empty → no header; quoted weak-validator token → preserved byte-for-byte; `IfMatch` on a `DELETE` → header present; `IfMatch` + `ContentType` → both headers present and independent.
  - **Dependencies**: None (052 `Response.Version()` is already landed; this task adds the request-side counterpart and does not call `Version()` itself)
  - **Plan reference**: Phase 1 (single phase), ADR-1: narrow `Request.IfMatch` field sent as `If-Match` only when non-empty; ADR-2: send mechanism only, defer the per-command retrofit and the `412` surfacing
  - **Scenario references**: features/clobbered-changes/guarded-writes.feature: "A captured version guards the write", "A stale guarded write is refused untouched", "A write without a version is sent unconditionally", "A delete is guarded the same way as an update", "An empty captured version sends no precondition", "A weak-validator version is forwarded verbatim", "An If-Match precondition composes with a content type", "Guarded Writes wires no production write command" (@validation), "The 412 refusal is left for downstream surfacing" (@validation)
  - **Interface references**: interface-spec.md: `Request.IfMatch string` field; the conditional `If-Match` set in `Execute`
