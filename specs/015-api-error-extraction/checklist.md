# Checklist: API Error Extraction

**Feature**: 015-api-error-extraction
**Checked against**: CONSTITUTION.md (done-* accords not found — see Governance Notes)
**Artifacts checked**: spec.md, plan.md, interface-spec.md, features/opaque-failures/api-error-extraction.feature, tasks.md
**Checks**: 13 (11 pass, 2 fail — both P1)
**Generated**: 2026-06-07

---

## Summary

13 checks, **11 pass, 2 fail (both P1, no P0)**. Constitution: 11/13. Done-criteria: not checked (no accords). Cross-reference: not checked (no accords).

The two P1 findings both sit on the **operator-facing surface** this slice improves, not on the extraction capability:
1. **II (Action Transparency)** — the new API-error message surfaces the API's *cause* (the `detail`) but the artifacts specify **no next-step hint**, which II's "every error MUST explain what went wrong **and the next step**" calls for — and the sibling `formatClientErrorMessage` arms (auth/transport/base-URL) all name a next step.
2. **VIII (No Fabricated Data)** — the status-derived **fallback `Detail`** occupies the same `Detail` field as an API-provided detail with no provenance marker, so a consumer can't tell "the API said this" from "we derived this from the status."

Four principles were calibrated to a non-2xx-interpretation seam with no command surface (II, III, VI, X); see Calibration notes. One principle (XI Governance via Proposals) produced no applicable check. A cross-spec **sequencing observation** on X (429 backoff still lands in 017) and a cross-spec **exit-code change** observation (401/403 move 3→4 for the landed reads 011–014) are recorded in Governance Notes for analyze to weigh — neither is a 015 artifact defect.

---

## Constitution Checks: 11/13 passed

### Calibration notes

- **II. Action Transparency** — no command surface in this slice; "report the spec operation + target resource, machine-parseable" is calibrated to "the produced `ProblemError` is a typed, `errors.As`-able value carrying the authoritative status + extracted `type`/`title`/`detail`, and the consuming command surfaces the `detail` to the operator." The *operation + resource id* line stays with the consuming command (which knows it called e.g. `GET /me`). The clause split into two binary checks: **(a) names the cause** and **(b) names the next step** — (a) passes, (b) fails (P1 below).
- **III. Fail Safe, Not Silent** — "validate a write / no partial state" is N/A (015 interprets a read-path error; it owns no write). The live concern is met and is the heart of the slice: `ExtractProblem` is **total** (always a typed error, never nil, never panics — even on a junk body), so a non-2xx is never swallowed or reported as success; the fail-soft *parse* degrades to a status fallback rather than dropping the error. The HTTP status is authoritative over a disagreeing body `status` (deterministic, no silent ambiguity).
- **VI. Size-Aware by Design** — no result-set paging surface here; calibrated to "no silent truncation of the error." `ProblemError` **preserves the full raw body** (extension members included) via the wrapped `ResponseError`; nothing is dropped. Paging across pages is Pagination (016).
- **X. Respect API Limits** — the detection target ("a retry loop that ignores `429`/`Retry-After`") cannot trip: 015 **never retries or backs off**; it extracts a `429` into a typed error like any other non-2xx and leaves backoff to Rate-Limit Handling (017). `If-Match`/`ETag` is an update concern, N/A. (Cross-spec sequencing of 017 — see Governance Notes.)

### Failed (2 — both P1)

- **P1** | CONSTITUTION II (Action Transparency): API-error message names a **next step** — **fail**. interface-spec.md § Error Communication defines the non-2xx message as the API's `detail` (or the `"status N"` fallback) with **no next-step hint**; plan ADR-4 and tasks T002 likewise specify only detail-surfacing. II requires "every error MUST explain what went wrong **and the next step**," and the sibling arms in `formatClientErrorMessage` already model this (`NoCredentials` → "run `glassfrog auth login`"; transport → "check connectivity"; base-URL → "correct --base-url …"). The `detail` satisfies "what went wrong"; the "next step" half is absent. *Most actionable where the class has an obvious step:* the new `PermissionError` (401/403) message could name "check the token's access / membership," and a `404` could hint "verify the resource id." **Recommendation (for the Crafter/Builder, not applied here):** specify a next-step hint for the API-error message in interface-spec.md (at least for the permission class), so the new arm matches its siblings and fully satisfies II. *Severity P1, not P0:* the cause **is** surfaced (a strict improvement over 011's bare "status N"), and a universal next step is not meaningful for every status — so this is an incomplete-satisfaction refinement, not an outright transparency failure.

- **P1** | CONSTITUTION VIII (No Fabricated Data): the **fallback `Detail` is indistinguishable from an API-provided `detail`** — **fail**. plan ADR-2 / interface-spec.md (`Detail` field) / tasks T001 specify that, when the body isn't parseable, `ExtractProblem` fills `Detail` with a value "derived from the status (e.g. `http.StatusText`)". The status itself is real API data, but the synthesized reason phrase is placed in the **same `Detail` field** that otherwise carries the API's own words, with **no marker of provenance**. A consumer (or the operator reading the message) cannot tell "the API said this" from "the CLI derived this from the status" — VIII's concern ("MUST NOT … fill placeholder values … present a synthesized value as real"). **Recommendation (for the Crafter/Builder, not applied here):** distinguish the two — e.g. a boolean on `ProblemError` indicating the detail was synthesized, or message phrasing that frames the fallback as a status label rather than the API's words (e.g. "no error detail provided (HTTP 502 Bad Gateway)"). *Severity P1, not P0:* the value is a deterministic rendering of a **real** status (not invented governance/API data), so it is fabrication-adjacent rather than a hard VIII breach — but the same-field, no-marker design is a real ambiguity worth closing.

### Passed (11/13)

- **P0** | CONSTITUTION I (Spec Fidelity): interprets only the documented error contract — **pass**. spec.md System Overview + interface-spec.md: 015 parses the RFC 9457 Problem Details shape (`type`/`title`/`status`/`detail`, `application/problem+json`) that `spec/glassfrog-api-v5.yaml` documents as the shape of "all 4xx and 5xx responses." It invents no endpoint, parameter, or behavior; it reads what the spec says the API returns. Strongly aligned (it implements the spec's own error contract).
- **P0** | CONSTITUTION II (Action Transparency): names the **cause** in machine-parseable form — **pass**. interface-spec.md Output contract (`ProblemError{StatusCode, Type, Title, Detail}`, `errors.As`-able) + Error Communication: the produced error carries the API's `detail`/`title` and the authoritative status, and the consuming command surfaces the `detail` — a clear improvement over 011's bare "status N". (The next-step half is the P1 above.)
- **P0** | CONSTITUTION III (Fail Safe, Not Silent): nothing swallowed or mis-reported as success — **pass** (calibrated). spec Behavioral Accord ("never fail to produce a typed error") + plan ADR-2 + Cross-cutting: `ExtractProblem` is total and fail-soft on parse only; every non-2xx yields a typed error mapped to a non-zero exit code; status authority is deterministic. Pinned by the `@validation` scenario "Every non-2xx yields a typed error."
- **P0** | CONSTITUTION IV (TDD): test-first + executable acceptance — **pass**. tasks T001–T003 each specify RED-first unit tests; features/opaque-failures/api-error-extraction.feature exists with executable acceptance scenarios (Phase 3 / T003); 4 `@validation` scenarios held out.
- **P0** | CONSTITUTION V (Composition over Monolith): additive, no unrelated edits — **pass**. plan System Architecture + tasks: Phase 1 is purely additive in `internal/apiclient` (new file); Phase 2 grows 011's **designated shared** read-surface helpers (`classifyClientError`/`formatClientErrorMessage`/`reportClientError`, the single extension point 012–017 reuse) rather than touching unrelated command modules. `apiclient` still does not import `internal/cli` (correct direction); the consumer maps the status. Mirrors how 013's `validateStatus` extended the shared surface.
- **P0** | CONSTITUTION VI (Size-Aware by Design): no silent truncation — **pass** (calibrated). plan ADR-1/ADR-2 + interface-spec.md: `ProblemError` wraps `ResponseError`, preserving the full raw body (extension members included); nothing is dropped.
- **P0** | CONSTITUTION VII (Working Software): impl + tests + build per increment — **pass**. tasks bundle implementation with RED-first tests; acceptance criteria require `go build ./...` and `go vet ./...` clean.
- **P0** | CONSTITUTION VIII (No Fabricated Data): extracted fields trace to the API — **pass**. plan ADR-1 + interface-spec.md: `Type`/`Title`/`Detail` come from the body when present; the raw body is carried verbatim; the authoritative status is the real HTTP status. No defaulting of *extracted* values. (The status-*derived fallback* detail is the separate P1 above.)
- **P0** | CONSTITUTION IX (Writes Require Explicit Intent): no mutation — **pass** (calibrated). 015 is read-path error interpretation with no command path; it issues no request at all (it interprets an outcome already received) and has no read→write side effect.
- **P0** | CONSTITUTION X (Respect API Limits): no retry loop ignoring 429 — **pass** (calibrated). spec Non-Behaviors + plan: 015 never retries or backs off; it surfaces a `429` as a typed error carrying its rate-limit headers so 017 can honor them. (Cross-spec sequencing — Governance Notes.)
- **P0** | CONSTITUTION XII (Standalone Executable): no new dependency — **pass**. plan/tasks/interface use only the standard library (`encoding/json`, `net/http` for `StatusText`, `errors`) and existing internal packages (`internal/apiclient`, `internal/cli`); no third-party dependency is introduced.

---

## Governance Notes

*(Separate from feature quality findings.)*

- **No `done-*` accords found** (`accords/governance/` does not exist). Done-criteria and cross-reference checks were not generated. Consider creating `accords/governance/done-{specify,plan,interface,scenarios,tasks}.md` to enable layered done-criteria checks across the pipeline. (Same project-level gap recorded for 008/009/010 — infrastructure, not a 015 finding.)
- **XI. Governance via Proposals** produced no applicable check — 015 is non-2xx error interpretation: no command surface, mutates no governance. The proposal-gating lives on the governance-write commands, not the error layer.
- **Cross-spec sequencing — X (Respect API Limits) and the 429 backoff** *(observation, not a 015 defect; carried to analyze/risk)*: 015 maps a `429` to the generic `APIError`(3) and leaves the `429`→rate-limit(5) split and the `Retry-After` backoff to Rate-Limit Handling (017), which is Should-tier and ranks after the Must-tier reads. So the one-attempt-no-backoff window 010's checklist already noted persists through 015 — expected, since 015's scope explicitly excludes 429 backoff. Same roadmap consideration: pull 017 forward, or accept the behavior for the initial slice.
- **Cross-spec exit-code change — 401/403 move from `APIError`(3) to `PermissionError`(4)** *(observation, not a 015 defect; carried to analyze)*: tasks T002 changes the exit code for 401/403 responses on the **landed, Complete** reads (011–014) from 3 to 4. This is intended — it fills the code 004 reserved, and is a published code, not a renumber — but it is a behavioral change to shipped capabilities that a caller scripting on exit 3 would observe. Worth a deliberate confirmation (the developer's call); analyze should confirm the 011 artifacts / exitcode.go comments and the 015 interface-spec agree that this is the planned split, not a contradiction.
