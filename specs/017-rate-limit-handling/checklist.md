# Checklist: Rate-Limit Handling

**Feature**: 017-rate-limit-handling
**Checked against**: CONSTITUTION.md (12 principles). No `accords/governance/done-*.md` accords present — done-criteria and cross-reference checks not generated (see Governance Infrastructure Notes).
**Artifacts checked**: spec.md, plan.md, interface-spec.md, features/no-shared-api-client/rate-limit-handling.feature, tasks.md
**Checks**: 15 (13 pass, 2 fail)
**Generated**: 2026-06-07

---

## Summary

| Severity | Count | Pass | Fail |
|---|---|---|---|
| P0 (blocking) | 11 | 11 | 0 |
| P1 (should fix) | 0 | 0 | 0 |
| P2 (consider) | 4 | 2 | 2 |
| **Total** | **15** | **13** | **2** |

No P0 or P1 failures. **Principle X (Respect API Limits) — the principle this feature exists to satisfy — passes**: 017 is the capability that closes the 429-backoff gap 011's checklist recorded as a P2 cross-spec deferral. Two P2 advisories remain, both in Action Transparency (II) / Respect API Limits (X), with **distinct owners**: (1) the exit-code-5 deferral is a cross-spec deferral to API Error Extraction (015); (2) the progress-note format is 017's own future hardening. Neither re-shapes the spec or blocks implementation.

---

## Constitution Checks: 13/15 passed

### Calibration notes

Four broad MUST principles were calibrated to this feature before evaluation:
- **II (Action Transparency)** → "the operator can tell the CLI is deliberately waiting on a rate limit (not hung); the final surfaced outcome carries a cause and a next step." Multi-clause, NON-NEGOTIABLE → multiple checks.
- **III (Fail Safe)** → "a capped-out 429 is surfaced loud (never swallowed/looped-forever); the wait is always bounded; no write is partially applied."
- **V (Composition)** → "the retry layer is a shared `apiclient` helper; wiring it into a read is the consuming side, not unrelated-command coupling."
- **X (Respect API Limits)** → "the retry loop honors `429` + `Retry-After` and does not ignore them; writes are not auto-retried." (The `If-Match`/`ETag` clause concerns updates — out of scope for this read-retry capability; see Governance Notes.)

### Failures

**P2** | CONSTITUTION.md II (Action Transparency) + X (Respect API Limits): "every error MUST explain what went wrong and the next step" / "honor the API's rate limits"
→ **plan.md § ADR-5** / **interface-spec.md § Error Communication**: A capped-out `429` is surfaced as a **generic** `*ResponseError` and maps to `APIError` (exit 3), **not** the reserved rate-limit exit code 5 — the `429 → RateLimited(5)` classification is deferred to API Error Extraction (015) by design (ADR-5; mirrors the spec Non-Behavior "must not classify the 429"). The stderr retry notes still make the rate-limiting legible to the operator, so transparency is partially served, but **exit-code-level** rate-limit classification will not exist until 015 lands. Cross-spec deferral, recorded for traceability — **not a 017 violation** (X's detection targets a loop that *ignores* `429`; 017's loop honors it). Same shape as 011's accepted X deferral, now one layer closer.

**P2** | CONSTITUTION.md II (Action Transparency): "report … in machine-parseable form"
→ **spec.md § Observability** / **interface-spec.md § Interactions (progress note)**: The per-wait progress note is free-form stderr text and its exact format is `[ASSUMED]` (deferred to implementation). II's machine-parseable clause concerns the *action's response output* — here the `me` projection (011's concern), which is unaffected — so this is **advisory, not a violation**: the note is supplementary diagnostics, and the operator may be human or agent (spec). If watching agents are expected to parse the retry signal programmatically, a structured note would harden II. Consider pinning the note's stability in a test.

### Passed (13/15)

- **P0 | X Respect API Limits (429 backoff)** — 017's retry loop **honors** `429` + `Retry-After` (integer seconds per `spec/glassfrog-api-v5.yaml:5339`), with a bounded fallback when absent, and never ignores them. This is the exact behavior X's detection requires and the gap 011's checklist deferred here. (spec.md Behavioral Accord "Reacting to a 429", plan.md ADR-2, feature: "A 429 is honored and the retry succeeds")
- **P0 | I Spec Fidelity** — 017 reacts only to the API's documented rate-limit contract (`429`, `Retry-After`, `X-RateLimit-*`; `spec/glassfrog-api-v5.yaml:87`); `parseRetryAfter` reads the spec's `integer`-seconds `RetryAfter` schema. No invented endpoint, parameter, or behavior; it sends only through 010's existing seam. (spec.md System Overview, plan.md ADR-2, interface-spec.md Internal helpers)
- **P0 | II Action Transparency (deliberate-pause legibility)** — a non-secret stderr note on each wait tells the operator the CLI is rate-limited and pausing (not hung), naming the wait and next attempt; no interactive prompt. The CLI's retrying action is legible. (spec.md Behavioral Accord "Observability while waiting", interface-spec.md, feature: "A progress note is written to stderr before re-attempting")
- **P0 | III Fail Safe, Not Silent** — a capped-out `429` is surfaced loud as the unchanged `*ResponseError` (never swallowed); the wait is always bounded by both `MaxAttempts` and `MaxTotalWait` (never an infinite loop/hang); non-429 outcomes pass through unmasked; a nil seam fails fast. No write is auto-retried, so no partial-apply. (spec.md Behavioral Accord "Bounding the wait" + Non-Behaviors, plan.md ADR-2/ADR-4, Cross-cutting)
- **P0 | IV Test-Driven Development** — T001/T002 are RED-first unit tests; the acceptance scenarios exist (`rate-limit-handling.feature`) before the code; T004 makes them executable (with a recording fake-sleep). (tasks.md T001–T004, feature file)
- **P0 | V Composition over Monolith** — the retry executor is a shared, independently-testable `apiclient` helper; the one edit to landed code (routing 011's `runMe` send through it, T003) is the **consuming-side wiring**, not unrelated-command coupling — future reads adopt the same wrapper at their own send site. `apiclient` still never imports `internal/cli`. (plan.md ADR-1/ADR-5, tasks.md T003, Cross-spec note)
- **P0 | VII Working Software** — each task pairs implementation with tests and requires `go build`/`go vet` clean; the 010/011 dependencies are landed so no task builds against non-compiling code. (tasks.md acceptance criteria, dependency graph)
- **P0 | VIII No Fabricated Data** — 017 surfaces the raw API `429` (status/headers/body) and synthesizes no API data; the progress note is timing diagnostics, not governance data; ADR-5 forbids rewriting the error body. (spec.md Non-Behaviors, interface-spec.md Error Communication)
- **P0 | IX Writes Require Explicit Intent** — 017 auto-retries only safe (`GET`/`HEAD`) requests; a `429` on a non-safe method is surfaced on first occurrence and never re-sent (ADR-3), so retry never introduces or repeats a mutation. (spec.md Non-Behaviors, plan.md ADR-3, feature: "A write is not auto-retried on a 429")
- **P0 | XI Governance via Proposals** — N/A by construction: 017 is a transport-layer retry with no command and no governance-structure mutation. (spec.md — no command surface)
- **P0 | XII Standalone Executable** — adds no new runtime dependency; uses Go stdlib only (`time`, `strconv`, `net/http`, `io`) within `internal/apiclient`; the artifact stays a self-contained binary. (plan.md — additive, stdlib only; DECISIONS Go-self-contained precedent)
- **P2 | VI Size-Aware by Design** — N/A by construction: 017 neither paginates nor caps result sets (that is Pagination, 016); it returns whatever the final attempt returns and drops no records. Each page request under 016 composes with 017's per-request retry. (plan.md Integration Design; "Does Not Cover")
- **P2 | II Action Transparency (operation legibility)** — the underlying read's operation/target legibility is 011's concern and unchanged; 017 adds only the retry note. Passed — no new traceability obligation for a transport-layer retry. (plan.md Integration Design)

---

## Governance Infrastructure Notes

*(separate from feature quality findings)*

- **No `accords/governance/done-*.md` accords exist.** Done-criteria and cross-reference checks were not generated — this checklist is constitution-only (same state as 010/011). Consider creating `done-specify.md`, `done-plan.md`, `done-interface.md`, `done-scenarios.md`, `done-tasks.md` to enable done-criteria gating and tasks↔scenarios↔interface link checks in future runs. Their absence is a tooling gap, not a feature defect.
- **Principle X (Respect API Limits), `If-Match`/`ETag` clause**: produced no applicable check for this feature. The optimistic-concurrency half concerns **updates**; 017 is read-retry and auto-retries no writes (ADR-3). That clause lands with a future write/proposal spec, not here.
- **Principle XI**: N/A (no governance mutation). Recorded as passed-by-construction, not dropped.

---

## Notes for the developer

- **Principle X passes** — 017 delivers the 429-backoff behavior the constitution requires and that 011's checklist deferred here. This is the headline result.
- The two P2 advisories have **distinct owners** — neither is a MUST violation, both are recorded for traceability:
  - **(1) capped-out 429 exits 3, not the reserved 5** → a justified **cross-spec deferral to API Error Extraction (015)**, which types the non-2xx and adds code 5 (ADR-5).
  - **(2) free-form progress-note format** → **017's own future hardening**, *not* a 015 concern (015 types API responses, not the stderr note). The follow-up is to pin the note's stability/structure in a test (see the P2 finding above); owner is this capability / a later 017 hardening pass.
- No finding blocks implementation.
