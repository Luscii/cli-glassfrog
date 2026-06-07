# Analyze: Rate-Limit Handling

**Feature**: 017-rate-limit-handling
**Artifacts analyzed**: spec.md, plan.md, interface-spec.md, features/no-shared-api-client/rate-limit-handling.feature, tasks.md
**Checklist context**: checklist.md present — correlation applied (0 P0, 0 P1, 2 P2)
**Findings**: 16 checks (16 pass, 0 fail) — one P2 coherence drift was resolved during this guard session
**Generated**: 2026-06-07

---

## Summary

| Category | Severity | Checks | Pass | Fail |
|---|---|---|---|---|
| Consistency | P0 | 6 | 6 | 0 |
| Completeness | P1 | 6 | 6 | 0 |
| Coherence | P2 | 4 | 4 | 0 |
| **Total** | | **16** | **16** | **0** |

No P0 contradictions, no P1 gaps. One P2 coherence finding (H1 — executor name drift between the plan/DECISIONS sketch and the interface-pinned surface) was **resolved this session** — see H1 below. The artifact set tells one coherent story.

---

## Consistency: 6/6 passed (P0)

All consistency checks pass — the artifacts make compatible claims.

- **C1** spec Integration Boundaries ↔ plan System Architecture: the plan's components/Integration Design cover every named boundary (010 upstream dependency, 011 consumer, 015/016 downstream siblings, 004 downstream, the operator stderr surface). PASS.
- **C2** spec Behavioral Accord ↔ plan architecture: the plan's `RetryExecutor` serves every behavior — pass-through of non-429, react-to-429 (honor `Retry-After`), bound-the-wait, safe-only eligibility, and the stderr note. None contradicted. PASS.
- **C3** spec Non-Behaviors ↔ plan: the plan architects none of the excluded capabilities — no proactive throttling, no unbounded wait, no write-retry (ADR-3), no 429 classification (ADR-5), no re-sending by itself (wraps `Execute`), no secret in notes, no exit-code decision. PASS.
- **C4** plan ADRs ↔ interface-spec Surface: the interface reflects every ADR — ADR-1 (retry above `Execute`), ADR-2 (`RetryPolicy` caps + `Retry-After`), ADR-3 (`isSafeMethod`), ADR-4 (injected `sleep`+writer, fail-fast), ADR-5 (no classification; surfaced 429 = `*ResponseError`). PASS.
- **C5** plan System Architecture ↔ tasks Scope: every task builds a plan-named component (T001 policy+helpers, T002 executor, T003 read-path wiring, T004 acceptance); no task introduces unplanned work. PASS.
- **C6** interface-spec ↔ feature steps: every Gherkin step references a surface the interface defines (the retrying executor, `GET`/`POST` requests, the 429/200/403/transport outcomes, `Retry-After`, the progress note, the surfaced `*ResponseError`). PASS.

---

## Completeness: 6/6 passed (P1)

- **K1** spec Driving Scenarios → feature: all 8 driving + 4 validation spec scenarios have Gherkin equivalents (verified title-by-title via the `# Source:` comments). PASS.
- **K2** spec Integration Boundaries → interface presence: the specification boundary (the package API) has interface-spec.md; the stderr operator surface is covered in interface-spec's Interactions/Error Communication; the upstream/downstream/sibling boundaries (010/011/015/016/004) are covered by their own specs and need no own interface file here (justified). PASS.
- **K3** plan phases → tasks: all three plan phases decompose into tasks (Phase 1→T001, Phase 2→T002, Phase 3→T003+T004). PASS.
- **K4** plan components → task scope: every component (`RetryPolicy`/`DefaultRetryPolicy`, `isSafeMethod`, `parseRetryAfter`, `RetryExecutor`/`NewRetryExecutor`, the read-path wiring, the godog acceptance) has an implementing task. PASS.
- **K5** interface-spec surfaces → feature coverage: every behavioral surface has scenario coverage — retry-then-success, no-wait passthrough, fallback backoff (`parseRetryAfter` false path), attempts-cap, total-wait-cap, safe-method gate (`POST` not retried), transport/non-429 passthrough, the progress note, the every-attempt-via-seam and surfaced-429-untyped validations. The `NewRetryExecutor` nil-seam **fail-fast** is a wiring-bug precondition covered by a unit test (T002 acceptance), not a behavioral scenario — consistent with how 010 treated its nil-base panic. `*AuthError` passthrough is covered by the "only a 429 triggers a retry" validation scenario's "every other outcome returned unchanged" clause (017 adds no AuthError-specific behavior). PASS.
- **K6** spec User Scenarios → interface coverage: all three user scenarios have interface coverage — ride-out-throttle → the `Execute` retry loop; bounded-wait → `RetryPolicy` caps + give-up; stderr-note → the Interactions progress note. PASS.

---

## Coherence: 4/4 passed (P2)

### Resolved this session

**P2 (resolved)** | H1 (Terminology — plan.md / `.score/memory/DECISIONS.md` ↔ interface-spec.md / tasks.md): the executor was named inconsistently across the set.
→ interface-spec.md pinned the decorator **`RetryExecutor`** built by **`NewRetryExecutor`** (with `(*RetryExecutor).Execute` mirroring `(*Client).Execute`), and tasks.md adopted that name — but plan.md (9 occurrences) and the durable DECISIONS.md entry (2 occurrences) still carried the plan's provisional free-function sketch name **`ExecuteWithRetry`**. This is the signature/name-drift class LEARNINGS flags (2026-06-07 #29: "when interface/tasks pin a signature, the DECISIONS.md entry and the plan prose must reflect the **pinned** surface — DECISIONS.md is the durable contract future specs inherit"). **Resolved**: plan.md and both DECISIONS.md entries were reconciled to `RetryExecutor`/`NewRetryExecutor` (the pseudocode header now reads `(*RetryExecutor).Execute`); a re-grep confirms no `ExecuteWithRetry` remains in any 017 artifact or the memory file. H1 passes.

### Passed (4/4)

- **H1** Terminology — *resolved above*: key concepts (`RetryExecutor`/`NewRetryExecutor`, `RetryPolicy`, `Retry-After`, "safe method", "progress note", "bounded wait", "surfaced unchanged") are now named consistently across all artifacts. PASS.
- **H2** Detail symmetry: spec↔plan and plan↔tasks are proportionate; no artifact is 3×+ more detailed than its neighbor on a shared topic. PASS.
- **H3** Scope alignment: spec, interface, and tasks describe the same feature scope. The exit-code-deferral (ADR-5), the caps, and the injected-seam wiring are consistent across the set — no capability is silently added or dropped. PASS.
- **H4** Phase coverage: tasks cover the plan's three-phase structure (ordering, the 010/011 dependencies, Phase 3's internal parallelism); tasks reference no non-existent phase, and every plan phase has corresponding tasks. PASS.

---

## Checklist Correlation

- The **H1 finding** (executor name drift, resolved this session) does **not** overlap either checklist finding — it is a coherence/terminology concern, not a constitution check.
- No analyze finding overlaps the two remaining checklist P2s (the exit-code-5 deferral to 015; the free-form progress-note format — the latter being 017's own future hardening, not a 015 concern). Those are vertical (constitution) advisories with no horizontal-consistency counterpart — and the artifacts agree *with each other* on the one cross-spec deferral they do make: spec Non-Behaviors, plan ADR-5, and interface Error Communication all consistently defer 429 *classification* to 015.

---

## Governance Notes

- All 16 relationship checks ran — every artifact was present, so no checks were skipped.
- Interface checks scaled ×1 (one interface file, interface-spec.md); scenario checks ran against the single feature file (rate-limit-handling.feature).
