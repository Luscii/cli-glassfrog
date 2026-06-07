# Checklist: API Error Extraction

**Feature**: 015-api-error-extraction
**Checked against**: CONSTITUTION.md (done-* accords not found — see Governance Notes)
**Artifacts checked**: spec.md, plan.md, interface-spec.md, features/opaque-failures/api-error-extraction.feature, tasks.md
**Checks**: 13 (13 pass, 0 fail)
**Generated**: 2026-06-07 (round 1); **updated 2026-06-07 (round 2 — PR #37 triage)**

---

## Summary

> **Round 2 (PR #37 triage):** the two round-1 P1 findings are now **resolved by design**. The interface/plan/tasks gained a **`DetailSynthesized`** provenance marker (VIII), a **`BodyStatus`** metadata field, and a **next-step hint** on the API-error message (II). Updated tallies: **13 pass, 0 fail.**

13 checks, **13 pass, 0 fail (no P0, no P1)** after round 2. Constitution: 13/13. Done-criteria: not checked (no accords). Cross-reference: not checked (no accords).

The two round-1 P1 findings — both on the **operator-facing surface** this slice improves — were addressed in the triage round:
1. **II (Action Transparency)** — the API-error message now appends a **next-step hint** (≥ for 401/403: "check the token's access / membership"), matching the sibling `formatClientErrorMessage` arms and satisfying II's "what went wrong **and the next step**".
2. **VIII (No Fabricated Data)** — `ProblemError` now carries **`DetailSynthesized`**, so the status-derived fallback is never presented as the API's own words; the consumer keys its message on the flag.

Four principles were calibrated to a non-2xx-interpretation seam with no command surface (II, III, VI, X); see Calibration notes. One principle (XI Governance via Proposals) produced no applicable check. A cross-spec **sequencing observation** on X (429 backoff still lands in 017) and a cross-spec **exit-code change** observation (401/403 move 3→4 for the landed reads 011–014) are recorded in Governance Notes for analyze to weigh — neither is a 015 artifact defect.

---

## Constitution Checks: 13/13 passed

### Calibration notes

- **II. Action Transparency** — no command surface in this slice; "report the spec operation + target resource, machine-parseable" is calibrated to "the produced `ProblemError` is a typed, `errors.As`-able value carrying the authoritative status + extracted `type`/`title`/`detail`, and the consuming command surfaces the `detail` to the operator." The *operation + resource id* line stays with the consuming command (which knows it called e.g. `GET /me`). The clause split into two binary checks: **(a) names the cause** and **(b) names the next step** — (a) passes; (b) passes after the round-2 fix added a next-step hint to the API-error message (see the round-2 resolution below).
- **III. Fail Safe, Not Silent** — "validate a write / no partial state" is N/A (015 interprets a read-path error; it owns no write). The live concern is met and is the heart of the slice: `ExtractProblem` is **total** (always a typed error, never nil, never panics — even on a junk body), so a non-2xx is never swallowed or reported as success; the fail-soft *parse* degrades to a status fallback rather than dropping the error. The HTTP status is authoritative over a disagreeing body `status` (deterministic, no silent ambiguity).
- **VI. Size-Aware by Design** — no result-set paging surface here; calibrated to "no silent truncation of the error." `ProblemError` **preserves the full raw body** (extension members included) via the wrapped `ResponseError`; nothing is dropped. Paging across pages is Pagination (016).
- **X. Respect API Limits** — the detection target ("a retry loop that ignores `429`/`Retry-After`") cannot trip: 015 **never retries or backs off**; it extracts a `429`, classifies it to `RateLimited`(5), and leaves the retry/backoff to Rate-Limit Handling (017). `If-Match`/`ETag` is an update concern, N/A. (Cross-spec sequencing of 017 — see Governance Notes.)

### Resolved in round 2 (were P1, now pass)

- **II (Action Transparency): API-error message names a next step — ✅ resolved (round 2).** Round 1 found the message surfaced the cause (`detail`) but no next step. The triage round added a **next-step hint** to the `formatClientErrorMessage` API-error arm (interface-spec.md § Error Communication; plan § System Architecture / ADR-4; tasks T002), at minimum for the `PermissionError` (401/403) class ("check the token's access / membership"), matching the sibling arms (`NoCredentials` → "run `glassfrog auth login`"; transport → "check connectivity"; base-URL → "correct --base-url …"). Both halves of II ("what went wrong **and** the next step") are now satisfied.

- **VIII (No Fabricated Data): fallback `Detail` distinguishable from API-provided detail — ✅ resolved (round 2).** Round 1 found the status-derived fallback shared the `Detail` field with API-provided detail, with no provenance marker. The triage round added **`DetailSynthesized bool`** to `ProblemError` (interface-spec.md Output contract; plan ADR-1/ADR-2; tasks T001): `false` = the API's own detail, `true` = the status-derived fallback. The consumer keys its message on the flag, so a synthesized value is never presented as the API's words. VIII is satisfied for both extracted and fallback paths.

### Passed (the other 11; with II and VIII above now also passing → 13/13)

- **P0** | CONSTITUTION I (Spec Fidelity): interprets only the documented error contract — **pass**. spec.md System Overview + interface-spec.md: 015 parses the RFC 9457 Problem Details shape (`type`/`title`/`status`/`detail`, `application/problem+json`) that `spec/glassfrog-api-v5.yaml` documents as the shape of "all 4xx and 5xx responses." It invents no endpoint, parameter, or behavior; it reads what the spec says the API returns. Strongly aligned (it implements the spec's own error contract).
- **P0** | CONSTITUTION II (Action Transparency): names the **cause** in machine-parseable form — **pass**. interface-spec.md Output contract (`ProblemError{StatusCode, Type, Title, Detail}`, `errors.As`-able) + Error Communication: the produced error carries the API's `detail`/`title` and the authoritative status, and the consuming command surfaces the `detail` — a clear improvement over 011's bare "status N". (The next-step half is now also satisfied — see the round-2 resolution above.)
- **P0** | CONSTITUTION III (Fail Safe, Not Silent): nothing swallowed or mis-reported as success — **pass** (calibrated). spec Behavioral Accord ("never fail to produce a typed error") + plan ADR-2 + Cross-cutting: `ExtractProblem` is total and fail-soft on parse only; every non-2xx yields a typed error mapped to a non-zero exit code; status authority is deterministic. Pinned by the `@validation` scenario "Every non-2xx yields a typed error."
- **P0** | CONSTITUTION IV (TDD): test-first + executable acceptance — **pass**. tasks T001–T003 each specify RED-first unit tests; features/opaque-failures/api-error-extraction.feature exists with executable acceptance scenarios (Phase 3 / T003); 4 `@validation` scenarios held out.
- **P0** | CONSTITUTION V (Composition over Monolith): additive, no unrelated edits — **pass**. plan System Architecture + tasks: Phase 1 is purely additive in `internal/apiclient` (new file); Phase 2 grows 011's **designated shared** read-surface helpers (`classifyClientError`/`formatClientErrorMessage`/`reportClientError`, the single extension point 012–017 reuse) rather than touching unrelated command modules. `apiclient` still does not import `internal/cli` (correct direction); the consumer maps the status. Mirrors how 013's `validateStatus` extended the shared surface.
- **P0** | CONSTITUTION VI (Size-Aware by Design): no silent truncation — **pass** (calibrated). plan ADR-1/ADR-2 + interface-spec.md: `ProblemError` wraps `ResponseError`, preserving the full raw body (extension members included); nothing is dropped.
- **P0** | CONSTITUTION VII (Working Software): impl + tests + build per increment — **pass**. tasks bundle implementation with RED-first tests; acceptance criteria require `go build ./...` and `go vet ./...` clean.
- **P0** | CONSTITUTION VIII (No Fabricated Data): extracted fields trace to the API — **pass**. plan ADR-1 + interface-spec.md: `Type`/`Title`/`Detail` come from the body when present; the raw body is carried verbatim; the authoritative status is the real HTTP status. No defaulting of *extracted* values. (The status-*derived fallback* detail is now marked by `DetailSynthesized` — see the round-2 resolution above — so it is never presented as the API's words.)
- **P0** | CONSTITUTION IX (Writes Require Explicit Intent): no mutation — **pass** (calibrated). 015 is read-path error interpretation with no command path; it issues no request at all (it interprets an outcome already received) and has no read→write side effect.
- **P0** | CONSTITUTION X (Respect API Limits): no retry loop ignoring 429 — **pass** (calibrated). spec Non-Behaviors + plan: 015 never retries or backs off; it surfaces a `429` as a typed error carrying its rate-limit headers so 017 can honor them. (Cross-spec sequencing — Governance Notes.)
- **P0** | CONSTITUTION XII (Standalone Executable): no new dependency — **pass**. plan/tasks/interface use only the standard library (`encoding/json`, `net/http` for `StatusText`, `errors`) and existing internal packages (`internal/apiclient`, `internal/cli`); no third-party dependency is introduced.

---

## Governance Notes

*(Separate from feature quality findings.)*

- **No `done-*` accords found** (`accords/governance/` does not exist). Done-criteria and cross-reference checks were not generated. Consider creating `accords/governance/done-{specify,plan,interface,scenarios,tasks}.md` to enable layered done-criteria checks across the pipeline. (Same project-level gap recorded for 008/009/010 — infrastructure, not a 015 finding.)
- **XI. Governance via Proposals** produced no applicable check — 015 is non-2xx error interpretation: no command surface, mutates no governance. The proposal-gating lives on the governance-write commands, not the error layer.
- **Cross-spec sequencing — X (Respect API Limits) and the 429 backoff** *(observation, not a 015 defect; carried to analyze/risk)*: 015 classifies a `429` to `RateLimited`(5) but leaves the `Retry-After` backoff to Rate-Limit Handling (017), which is Should-tier and ranks after the Must-tier reads. So the one-attempt-no-backoff window 010's checklist already noted persists through 015 — expected, since 015's scope is classification, not backoff. Same roadmap consideration: pull 017 forward, or accept the behavior for the initial slice.
- **Cross-spec 429→rate-limit(5) ownership** *(surfaced by the `main` merge; resolved PR #37)*: landed 017 does the 429 retry/backoff but its Non-Behavior defers the 429→rate-limit(5) *classification* to 015 ("Code 5 lands when 015's producer exists"). **Resolved by folding the 429→`RateLimited`(5) split into 015** — 015 classifies (exit 5), 017 handles (retry); 015 never backs off. Code 5 now has its producer; no orphaned reserved code.
- **Cross-spec exit-code change — 401/403→`PermissionError`(4) and 429→`RateLimited`(5)** *(observation, not a 015 defect; carried to analyze)*: tasks T002 changes the exit code for 401/403 and 429 responses on the **landed, Complete** reads (011–014) from 3 to 4 / 3 to 5. Intended — it fills the codes 004 reserved, published codes, not a renumber — but it is a behavioral change to shipped capabilities that a caller scripting on exit 3 would observe. Worth a deliberate confirmation (the developer's call); analyze should confirm the 011 artifacts / exitcode.go comments and the 015 interface-spec agree this is the planned split, not a contradiction.
