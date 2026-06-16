# Validate: Version Capture on Read

**Feature**: 052-version-capture-on-read
**Round**: 1 of 3
**Date**: 2026-06-14
**Verdict**: Ready
**Artifacts loaded**: spec.md, plan.md, tasks.md, interface-spec.md, features/clobbered-changes/version-capture-on-read.feature, PROJECT.md
**Implementation files**: 1 production change (`internal/apiclient/execute.go` — `Response.Version()`), 2 test files (`execute_test.go` unit tests, `version_capture_on_read_bdd_test.go` step definitions)

> guardian-agent.md not deployed — validated against SKILL.md alone (reduced character consistency, not a blocked skill).

---

## Conformance Summary

| Dimension | Status | Findings |
|---|---|---|
| Driving scenario coverage | ✓ Pass | 0 |
| Acceptance criteria | ✓ Pass | 0 |
| Interface contract conformance | ✓ Pass | 0 |
| Non-behavior absence | ✓ Pass | 0 |
| @wip lifecycle completion | ✓ Pass | 0 |
| **Validation scenarios** | ✓ Satisfied | 0 |

**Total**: 5 dimensions checked, 5 passed, 0 findings. 2 of 2 validation scenarios satisfied.

---

## Driving Scenario Coverage

**Status**: Pass (7 of 7 scenarios covered)

All seven driving scenarios from spec.md are concretized in the feature file, have identifiable code paths, and pass as executable BDD scenarios (`7 scenarios, 26 steps, 7 passed`).

| Scenario | Status | Implementation |
|---|---|---|
| Version captured from a single tension read | ✓ Covered | `execute.go:43` `Response.Version()` → `Header.Get("ETag")` |
| Mechanism is resource-agnostic | ✓ Covered | Accessor reads only the header; no resource-type input (BDD replays same ETag as a tension read) |
| Captured version does not change read output | ✓ Covered | Accessor derives from `Header`, writes nothing to the decoded body / render path |
| No version present on the response | ✓ Covered | `Header.Get` returns `""` for absent ETag; read still 2xx-decodes |
| Failed read captures nothing | ✓ Covered | Non-2xx → `*ResponseError`, no `*Response` exists to call `Version()` on (`execute.go:133-138`) |
| Collection read yields no per-resource version | ✓ Covered | Accessor is response-level only; list-walk path produces no per-item `*Response` |
| Version token is captured verbatim | ✓ Covered | `Header.Get` returns the raw value — no unquoting/normalization |

---

## Acceptance Criteria

**Status**: Pass (T001 checked — all 6 criteria met)

| Criterion (tasks.md T001) | Status | Evidence |
|---|---|---|
| ETag returned verbatim (no unquoting / no `W/` stripping / no normalization) | ✓ Met | `TestResponseVersionReturnsETagVerbatim`; `Version()` returns `Header.Get("ETag")` unaltered |
| Absent ETag → `""`; empty indistinguishable from absent | ✓ Met | `TestResponseVersionAbsentETagIsEmpty` |
| Header lookup case-insensitive (ETag/Etag/etag) | ✓ Met | `TestResponseVersionHeaderLookupIsCaseInsensitive` (net/http canonicalization) |
| `Execute` + existing reads byte-identical | ✓ Met | No `Execute` change; grep confirms **no non-test `.Version()` call site** — purely additive |
| No `If-Match` field on `Request`; nothing sends `If-Match` | ✓ Met | `Request` struct carries no `If-Match` field (only the deferral comment, client.go:42); no If-Match send introduced in `internal/apiclient` |
| Unit tests cover verbatim / absent / weak-validator / case-insensitivity | ✓ Met | All four `TestResponseVersion*` tests present and passing |

---

## Interface Contract Conformance

**Status**: Pass (1 of 1 surface conformant)

| Surface (interface-spec.md) | Status | Evidence |
|---|---|---|
| `func (r *Response) Version() string` | ✓ Conformant | Exact signature at `execute.go:43`; method on `*Response`, no new type |
| No new imports / no change to `Request`, `Execute`, `executor`, `RetryExecutor` | ✓ Conformant | Only an additive method + doc comment in `execute.go`; no import or type-shape change |
| Error communication (ETag → verbatim; no ETag → `""`; failed read → no receiver) | ✓ Conformant | Matches the table; `Version()` has no failure mode (`Header.Get` never errors) |

---

## Non-Behavior Absence

**Status**: Pass (0 of 6 non-behaviors violated)

| Non-behavior (spec.md) | Status | Evidence |
|---|---|---|
| Must not send `If-Match` / enforce concurrency | ✓ Absent | No If-Match send introduced in `internal/apiclient`; the pre-existing tension-command no-If-Match behavior (042–045) is untouched |
| Must not surface the version in any user-facing output | ✓ Absent | Accessor renders nothing; "rendered output unchanged" scenario passes; no non-test caller exists |
| Must not interpret / validate / normalize the token | ✓ Absent | Verbatim `Header.Get`; weak-validator preserved byte-for-byte |
| Must not capture from collection/list reads | ✓ Absent | Response-level accessor; walker path produces no per-resource `*Response` |
| Must not fail or alter a read when no version present | ✓ Absent | Absent ETag → `""`, read still succeeds and decodes |
| Must not persist the version across invocations | ✓ Absent | Pure derivation from in-memory `Header`; no disk/config write |

---

## Validation Scenario Results

**Status**: Satisfied (2 of 2 traced to implementation)

| Scenario | Status | Trace |
|---|---|---|
| Adding version capture changes no read contract | ✓ Satisfied | No `Execute` change, no call-site wiring (grep: zero non-test `.Version()` callers), full `go test ./...` green — no output/exit/diagnostic change |
| No If-Match header is sent by version capture | ✓ Satisfied | grep confirms `internal/apiclient` introduces no If-Match send and `Request` has no If-Match field; the capability adds only the read-side accessor |

These two scenarios correctly retain `@validation @wip` in the feature file — held out from the Builder, verified here independently.

---

## Verdict: Ready

All 5 conformance dimensions pass. Both validation scenarios are satisfied through independent inspection. The single task (T001) is checked and its acceptance criteria are fully met. The implementation is a faithful, behavior-preserving realization of the spec: a verbatim read-side `ETag` accessor with no consumption, no request-side change, and no user-facing surface — exactly the mechanism-only scope ADR-2 committed to.

## Next Steps

Implementation conforms to the specification. Suggest PR review and merge. The accessor is intentionally unused until **Guarded Writes (053)** consumes it via `If-Match`.
