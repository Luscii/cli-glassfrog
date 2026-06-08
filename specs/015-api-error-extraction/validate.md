# Validate: API Error Extraction

**Feature**: 015-api-error-extraction
**Round**: 1 of 3
**Date**: 2026-06-07
**Verdict**: Ready
**Artifacts loaded**: spec.md, plan.md (§ System Architecture + ADRs), tasks.md (3 of 3 tasks complete), interface-spec.md, features/opaque-failures/api-error-extraction.feature, PROJECT.md
**Implementation files**: `internal/apiclient/problem.go` (+ `problem_test.go`); `internal/cli/{dispatch,exitcode,clienterror,me}.go` (grown); `internal/cli/api_error_extraction_bdd_test.go` (new godog suite); test ripple in `internal/cli/{me,my_actions,my_projects,myroles}_test.go`, `{my_actions,my_projects,identity_read}_bdd_test.go`, and `features/self-service-reads/my-roles.feature`

> Note: `validate/agents/guardian-agent.md`, `references/context-engineering-review.md`, and `references/self-verification-checklist.md` are not deployed — applied the skill's own self-checks and the validate-template only.

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

**Total**: 5 dimensions checked, 5 passed, 0 findings; 4 of 4 validation scenarios satisfied.

---

## Driving Scenario Coverage

**Status**: Pass (8 of 8 spec driving scenarios + 4 architecture-informed consumer scenarios covered)

The capability splits across `internal/apiclient` (extraction) and `internal/cli` (mapping + presentation). Extraction scenarios trace to `ExtractProblem`; consumer scenarios to the grown shared read-surface helpers. All are executably pinned by the 12 non-`@validation` scenarios in the godog suite (`TestAPIErrorExtractionFeatures` — 12 scenarios, 45 steps, passing).

| Scenario | Status | Implementation |
|---|---|---|
| A valid Problem Details body is extracted | ✓ Covered | `apiclient/problem.go:94` `ExtractProblem` (parses type/title/detail, sets `DetailSynthesized=false`) |
| A 404 surfaces the API's own detail | ✓ Covered | `ExtractProblem` populates `Detail` from the body's `detail` member |
| Extension members are preserved without being promoted | ✓ Covered | `problemBody` (problem.go:74) decodes only the standard four; raw body reachable via `Unwrap()` → `*ResponseError.Body` |
| An empty body degrades to the HTTP status | ✓ Covered | `statusFallbackDetail` (problem.go:135), `DetailSynthesized=true`, raw body preserved on wrapped value |
| A non-JSON gateway body degrades gracefully | ✓ Covered | `json.Unmarshal` (problem.go:101) attempted regardless of header; degrades to fallback |
| The HTTP status wins over a disagreeing body status | ✓ Covered | `StatusCode = re.StatusCode` (problem.go:96); body `status` captured in `BodyStatus`, never promoted |
| A 429 is extracted without backoff | ✓ Covered | `ExtractProblem` (no sleep/retry); `classifyClientError` → `RateLimited` (clienterror.go:73); headers reachable |
| A 403 is carried generically | ✓ Covered | `ExtractProblem` carries detail as-is; `classifyClientError` → `PermissionError`; no plan-availability wording |
| The API detail appears in the failure message | ✓ Covered | `reportClientError` refines once (me.go:201); `formatClientErrorMessage` `*ProblemError` arm renders `Detail` (me.go:240) |
| An authorization failure exits with the permission code | ✓ Covered | `classifyClientError` 401/403 → `PermissionError`; `ExitCode` → 4 (exitcode.go:47) |
| A rate-limited response exits with the rate-limit code | ✓ Covered | `classifyClientError` 429 → `RateLimited`; `ExitCode` → 5 (exitcode.go:49) |
| A non-permission API error exits with the general API code | ✓ Covered | `classifyClientError` default → `APIError`; `ExitCode` → 3 |

---

## Acceptance Criteria

**Status**: Pass (3 of 3 tasks complete; every criterion met)

**T001** — `ProblemError` + `ExtractProblem`: a valid body yields `Type`/`Title`/`Detail` with `DetailSynthesized==false`; extension members surface only the standard fields with the raw body reachable via the wrapped `*ResponseError`; empty/non-JSON/HTML/member-missing bodies yield a status-derived fallback with `DetailSynthesized==true`, never nil/panic; parse is not Content-Type-gated; a disagreeing body `status` is captured in `BodyStatus` while `StatusCode==re.StatusCode`; `errors.As` discriminates `*ProblemError` and still matches the wrapped `*ResponseError`; RED-first table tests cover all rows; `go build`/`go vet` clean. Verified against `problem.go` + `problem_test.go` (9 table rows + wrap-reachability + discrimination + token-hygiene).

**T002** — consumer wiring: `classifyClientError` maps 401/403→`PermissionError`, 429→`RateLimited`, 500→`APIError`; the table test grows with these rows **and** the `*ProblemError` (wrapped) rows, keeping its `len`+comma-ok exhaustiveness guard; `ExitCode(PermissionError)==4` and `ExitCode(RateLimited)==5` pinned in `exitcode_test.go`; `reportClientError` refines once (guarded) and returns the `*ProblemError`; `formatClientErrorMessage` renders the detail when `DetailSynthesized==false`, the "status N" fallback when `true` (keyed on the flag, not emptiness), and the per-class next-step hint; messages are token-free; 011–014 tests pass. Verified against `dispatch.go`, `exitcode.go`, `clienterror.go`, `me.go` + tests.

**T003** — executable acceptance: 12 behavioral scenarios un-`@wip`'d and passing in a new suite whose `Paths` names only `api-error-extraction.feature`; the 4 `@validation` scenarios keep `@wip`; every `internal/cli` suite reports its own independent count; no real network/home touched; build/vet/suites clean. Verified by running each suite (APIErrorExtraction 12, TestFeatures 47, IdentityRead 11, MyActions 11, MyProjects 12, MyRoles 10).

---

## Interface Contract Conformance

**Status**: Pass (entry point, output contract, and all 5 consumer-contract changes conformant)

| Surface | Status | Implementation |
|---|---|---|
| `ExtractProblem(re *ResponseError) *ProblemError` | ✓ Conformant | problem.go:94 — exact signature, total, response-side only |
| `ProblemError` fields (`StatusCode`/`Type`/`Title`/`Detail`/`DetailSynthesized`/`BodyStatus *int`; `Body`/`Header` via wrapped; `Error()`/`Unwrap()`) | ✓ Conformant | problem.go:27–68 — all fields and accessors present, `BodyStatus` nil-able |
| `Outcome` + `PermissionError`/`RateLimited` (+ `String()`) | ✓ Conformant | dispatch.go:92/98, 114–117 |
| `ExitCode` + cases for codes 4/5 | ✓ Conformant | exitcode.go:47–50 — reuses reserved constants, no renumber |
| `classifyClientError` status split | ✓ Conformant | clienterror.go:68–78 — branch inside the `*ResponseError` arm |
| `formatClientErrorMessage` detail surfacing + next step | ✓ Conformant | me.go:240–256, `clientErrorNextStep` me.go:298 |
| `reportClientError` refine-once | ✓ Conformant | me.go:189 + `refineClientError` me.go:201 (double-refinement guard) |
| Status → Outcome → exit-code mapping (401/403→4, 429→5, else→3) | ✓ Conformant | matches the interface-spec table exactly |

The `Body`/`Header` "via the wrapped `*ResponseError`" accessor (no direct `ProblemError.Body()` method) is the interface-spec's documented design, not a gap.

---

## Non-Behavior Absence

**Status**: Pass (all 8 non-behaviors confirmed absent)

| Non-behavior | Status | Evidence |
|---|---|---|
| No requests / connections / transport read | ✓ Absent | `problem.go` has no `http.Client`/`.Do`/transport; `ExtractProblem` reads only `re` |
| No retry / backoff / sleep on 429 | ✓ Absent | no `time.`/`Sleep`/`Retry` in `problem.go`; the 429 path classifies only |
| No 403 → plan-availability translation | ✓ Absent | 403 hint is "check the token's access / membership"; no plan/premium/upgrade wording |
| Not conditioned on `Content-Type` | ✓ Absent | `ExtractProblem` never reads `re.Header`; `json.Unmarshal` runs regardless |
| No interpreting success / transport / decode outcomes | ✓ Absent | `ExtractProblem` takes only `*ResponseError`; `classifyClientError` leaves the Auth/Transport/Decode arms unchanged |
| No exit-code / print / message decision in the capability | ✓ Absent | `apiclient` carries the raw status; the consumer maps; `ExtractProblem` prints nothing |
| No re-decoding the error body into a success target | ✓ Absent | 010's `Execute` never decodes a non-2xx into `out`; 015 adds no such path |
| Body `status` never overrides HTTP status | ✓ Absent | `StatusCode = re.StatusCode`; `BodyStatus` is separate metadata |

---

## @wip Lifecycle Completion

**Status**: Pass

The 12 behavioral scenarios referenced by T003 have their `@wip` removed and pass executably. The 4 `@validation` scenarios ("Extraction reads only the outcome handed in", "No backoff occurs while interpreting a 429", "The produced status always equals the HTTP status", "Every non-2xx yields a typed error") correctly retain `@wip` — they are held out for this validation pass and are excluded by the `~@wip` filter. No stray `@wip` remains on an implemented scenario.

---

## Validation Scenario Results

**Status**: Satisfied (4 of 4 traced to implementation by independent inspection)

| Scenario | Status | Trace |
|---|---|---|
| Extraction reads only the outcome handed in | ✓ Satisfied | `ExtractProblem(re *ResponseError)` reads only `re.StatusCode`/`Body`/(never `Header`); no `os`/env/flag/credentials/request access anywhere in `problem.go` |
| Every non-2xx yields a typed error | ✓ Satisfied | `ExtractProblem` always returns a non-nil `*ProblemError`, never an `error`; empty/HTML/member-missing/blank-detail/nonstandard-status rows pin a non-empty `Detail` |
| The produced status always equals the HTTP status | ✓ Satisfied | `StatusCode = re.StatusCode` unconditionally; `BodyStatus` carries the disagreeing body `status` as metadata only (status-mismatch test row + bdd) |
| No backoff is observable for a 429 | ✓ Satisfied | no sleep/delay/retry in the interpretation path; rate-limit headers reachable unchanged via the wrapped `*ResponseError` (bdd asserts headers carried + no transport touched) |

---

## Verdict: Ready

All 3 tasks are checked. All 5 conformance dimensions pass with zero findings, and all 4 held-out validation scenarios trace to clear code paths. The `apiclient` extraction capability is pure, total, and response-side-only; the `internal/cli` consumer split fills the reserved codes 4/5 without renumbering and surfaces the API's own detail through the single `reportClientError` chokepoint. The cross-spec ripple (401/403/429 reclassification of landed reads 011–014) was handled by re-pointing arbitrary "generic non-2xx" samples to 500 and is recorded in LEARNINGS.md; the full suite is green. The implementation conforms to its specification.

---

## Next Steps

Implementation conforms to the specification. Suggest PR review and merge. The specification loop for 015 is closed.
