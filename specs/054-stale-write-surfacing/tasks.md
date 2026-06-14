# Tasks: Stale-Write Surfacing

**Feature**: 054-stale-write-surfacing
**Concretization**: Full context (plan + spec + interface-cli + scenarios)
**Inputs**: plan.md, spec.md, interface-cli.md, features/clobbered-changes/stale-write-surfacing.feature

---

## Dependency Graph

Phase 1: Stale-Write Classification (1 task, no phase dependencies) — single-phase build

1 task total | 1 phase

---

## Branching Guidance

**Pipeline mode**: `spec/054-stale-write-surfacing/base` → `spec/054-stale-write-surfacing/task-1`

---

## Phase 1: Stale-Write Classification [Shared]

- [ ] **T001** [Shared] Add the `StaleWrite` Outcome + `codeStaleWrite = 7`, and classify `412` (category, cause, next step) in `internal/cli`, with unit tests — 10 scenarios (3 @validation held @wip), exit-code + diagnostic unit tests; no new command/flag/output
  - **Scope**: One reviewable change across three sibling files in `internal/cli`, plus tests:
    - `dispatch.go` — add `StaleWrite` to the `Outcome` enum (a new `iota` value after `RateLimited`) with a doc comment naming `412` / Stale-Write Surfacing (054) as its producer, and add its `String()` case.
    - `exitcode.go` — add `codeStaleWrite = 7` with a doc comment, and the `case StaleWrite: return codeStaleWrite` arm in `ExitCode`.
    - `diagnostic.go` — add a `case http.StatusPreconditionFailed:` arm to `categoryForStatus` returning `StaleWrite`; add a `412` arm to `nextStepForStatus` returning the re-read-for-current-version-then-retry hint; and make the synthesized cause `412`-aware (surface the API's own `detail`/`title` when present — unchanged `problemCause` behavior — and, when synthesized, name the precondition failure / changed-since-read instead of the bare generic "status 412" line).
    Adds no new command, flag, or output format; renders nothing itself (032 renders the unchanged three-field `Diagnostic`); classifies by the `412` status alone. The enum value, the code, the classification arms, and their pinning tests ship together (the code must not merge without the tests that pin its uniqueness and the no-drift guarantee).
  - **Acceptance criteria**:
    - A `412` outcome maps to `Category: StaleWrite` and the process exits with `codeStaleWrite = 7` — distinct from the generic API-error code `3`.
    - The `412` next step tells the operator to re-read the resource for its current version and retry the write — not the generic "check that the token has access" step.
    - The `412` cause surfaces the API's own `detail`/`title` when present; when the API supplied none, the cause is derived from the `412` status and names the precondition failure — never fabricated.
    - Classification is status-driven only: a `412` maps to `StaleWrite` regardless of the command, the resource, or whether an `If-Match` header was sent.
    - Every other status is unchanged — `404`/`500` keep `APIError`(3); `401`/`403` keep `PermissionError`(4); `429` keeps `RateLimited`(5) — with their prior cause and next step (the `default` arms are untouched).
    - `7` is a new, previously-unused code; no existing code (`0`–`6`) is renumbered or reassigned, and `7` is outside the shell-reserved range (`126`/`127`/`128+N`).
    - Unit tests cover: `412 *ProblemError` → `StaleWrite` category + code `7` + re-read/retry next step + API-detail cause; `412` with synthesized detail → status-derived precondition-failure cause; `404`/`500`/`401`/`403`/`429` unchanged (no drift); `exitcode_test.go` extended to pin `StaleWrite → 7`, code uniqueness (no category collision), no-shell-reserved, and no-renumber. The exit-code table test must use a `len` check + comma-ok lookups (not a value-only map index) so a dropped category fails loudly (zero-value-trap guard, LEARNINGS).
  - **Dependencies**: None (053 is landed and already produces the generic `*ResponseError{StatusCode: 412}`; this task reclassifies that surfaced `412` and does not touch the write path)
  - **Plan reference**: Phase 1 (single phase); ADR-1 (distinct `StaleWrite` Outcome → new exit code `7`, classified by `412` status at the existing registry sites); ADR-2 (the `412`-specific cause and re-read/retry next step at the existing `Diagnose` composition site)
  - **Scenario references**: features/clobbered-changes/stale-write-surfacing.feature: "A stale write surfaces under its own exit code", "The next step points the operator to re-read and retry", "Classification ignores whether a precondition was sent", "The cause names the stale write", "A 412 without readable detail derives its cause from the status", "Another non-2xx status is unaffected", "Adding the stale-write code renumbers no existing code", "A stale write is distinct from the generic bucket" (@validation), "The capability surfaces without recovering" (@validation), "No existing surfacing drifts" (@validation)
  - **Interface references**: interface-cli.md: the new `7` (stale-write) row in the published exit-code registry; the `412` diagnostic (cause + re-read/retry next step) in the Error Communication table
