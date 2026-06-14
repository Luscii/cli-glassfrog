# Tasks: Version Capture on Read

**Feature**: 052-version-capture-on-read
**Concretization**: Full context (plan + spec + interface-spec + scenarios)
**Inputs**: plan.md, spec.md, interface-spec.md, features/clobbered-changes/version-capture-on-read.feature

---

## Dependency Graph

Phase 1: Capture Accessor (1 task, no phase dependencies) — single-phase build

1 task total | 1 phase

---

## Branching Guidance

**Pipeline mode**: `spec/052-version-capture-on-read/base` → `spec/052-version-capture-on-read/task-1`

---

## Phase 1: Capture Accessor [Shared]

- [ ] **T001** [Shared] Add the `Response.Version()` version-capture accessor with its unit tests
  - **Scope**: One reviewable change in `internal/apiclient` (alongside the `Response` type in `execute.go`): add a derived accessor that returns the resource version carried by a read response — the `ETag` header, verbatim, with `""` as the "no version captured" sentinel. Adds no stored field, no `Execute` change, no change to `Request`, the `executor` interface, or `RetryExecutor`; wires no existing read call site and sends no header. The accessor plus its unit tests ship together (the accessor must not merge without the tests that pin its contract).
  - **Acceptance criteria**:
    - A read response carrying an `ETag` returns that value **verbatim** from the accessor — no unquoting, no weak-validator (`W/"…"`) prefix stripping, no normalization.
    - A response carrying no `ETag` returns `""`; an empty `ETag` is indistinguishable from absent (both `""`).
    - Header lookup is case-insensitive (`ETag`/`Etag`/`etag` all match).
    - `Execute` and all existing read commands stay byte-identical — no user-facing output, exit-code, or diagnostic change (the accessor is purely additive and invoked by no existing call site).
    - No `If-Match` field is added to `apiclient.Request`, and nothing this change introduces sends an `If-Match` header.
    - Unit tests cover: `ETag` present → verbatim; `ETag` absent → `""`; quoted weak-validator token → preserved byte-for-byte; header-name case-insensitivity.
  - **Dependencies**: None
  - **Plan reference**: Phase 1 (single phase), ADR-1: verbatim accessor on `apiclient.Response`; ADR-2: mechanism-only, defer consumption + the `If-Match` request field to 053
  - **Scenario references**: features/clobbered-changes/version-capture-on-read.feature: "A single-resource read retains its version", "A response without an ETag retains no version", "A rejected read retains no version", "Version capture is resource-agnostic", "Capturing a version leaves rendered output unchanged", "A list read retains no per-resource version", "A weak-validator version is captured verbatim", "Adding version capture changes no read contract" (@validation), "No If-Match header is sent by version capture" (@validation)
  - **Interface references**: interface-spec.md: `func (r *Response) Version() string`
