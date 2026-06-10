# Checklist: Diagnostic Normalization

**Feature**: 031-diagnostic-normalization
**Checked against**: CONSTITUTION.md (12 principles)
**Artifacts checked**: spec.md, plan.md, interface-spec.md, features/opaque-failures/diagnostic-normalization.feature, tasks.md
**Checks**: 14 (14 pass, 0 fail)
**Generated**: 2026-06-10

---

## Summary

All 14 checks pass. Constitution: 14/14. (Done-criteria and cross-reference checks not generated — no `accords/` directory exists in this project.)

---

## Calibration

Two broad MUST principles were calibrated to binary, feature-specific assertions before evaluation:

- **II. Action Transparency** → for this feature: (a) every normalized failure family yields a non-empty `Cause`; (b) every category with a known recovery yields a `NextStep`, and absence is the explicit `""` signal; (c) the diagnostic carries a `Category` drawn from one fixed vocabulary (the `Outcome` taxonomy), i.e. machine-branchable.
- **III. Fail Safe, Not Silent** → for this feature: (a) `Diagnose` is total and never maps a failure to `Success`/exit 0; (b) an unrecognized failure is surfaced (falls through to the internal-error safety net), never swallowed.

---

## Constitution Checks: 14/14 passed

### Passed (14/14)

**P0** | CONSTITUTION II (Action Transparency): every error explains what went wrong
→ **spec.md § Behavioral Accord (Composing the cause) + interface-spec.md § Surface**: every `Diagnose` arm sets a non-empty `Cause` (API detail, status fallback, named wire/shape failure, or token-free verbatim). Pass.

**P0** | CONSTITUTION II: every error states the next step
→ **spec.md § Composing the next step + interface-spec.md § Next-step contract**: each category with a known recovery attaches a `NextStep`; "no reliable next step" is the explicit `""` signal, not a fabricated one. Pass.

**P0** | CONSTITUTION II: machine-parseable / branchable form
→ **interface-spec.md § Output contract**: `Diagnostic.Category` is drawn from the fixed `Outcome` vocabulary, so a caller can branch on the kind of failure. Pass. (Machine-parseable *emission* per `--output` is 032's scope — spec Non-Behaviors; the structured value 031 produces satisfies the value-level requirement.)

**P0** | CONSTITUTION III (Fail Safe): a failure is never reported as success
→ **plan.md § Cross-cutting Concerns (Totality) + interface-spec.md § Interactions**: `Diagnose` is total; an unrecognized error maps to `RuntimeError` (exit 1), never `Success`. Pass.

**P0** | CONSTITUTION III (MUST NOT swallow errors)
→ **spec.md § Normalizing (unrecognized arm) + feature: "An unrecognized failure falls through to the safety net"**: an unrecognized failure is surfaced to 004's safety net (internal-error code + trace), never hidden. Pass.

**P0** | CONSTITUTION IV (TDD): user-facing behavior has an executable acceptance scenario
→ **features/opaque-failures/diagnostic-normalization.feature**: 15 `@wip` scenarios exist before implementation, covering every behavioral change. Pass.

**P0** | CONSTITUTION IV: tasks are test-first / build verified
→ **tasks.md T001/T002 acceptance criteria**: each task specifies tests (table-driven `Diagnose` test, golden byte-equivalence, token-free assertion, `clienterror_test.go:38` flip) and `go build` + full `go test ./...` green. Pass.

**P0** | CONSTITUTION V (Composition over Monolith): adding doesn't force unrelated changes
→ **plan.md § ADR-1 + interface-spec.md § Consumer contract changes**: the change is a single `internal/cli` consolidation; `classifyClientError` survives as a thin delegate so the three category-only callers (`me`/`roles`/`subroles`) stay zero-diff. No unrelated command is touched. Pass.

**P0** | CONSTITUTION VII (Working Software): impl + tests together, validates and builds
→ **tasks.md T001/T002**: acceptance criteria require implementation with its tests and a green build/suite in the same unit. Pass.

**P0** | CONSTITUTION VIII (No Fabricated Data): no invented/guessed values
→ **spec.md § Non-Behaviors + Composing the cause**: cause uses the API's own detail or a status-derived fallback (never invented); a next step is omitted rather than guessed when none reliably applies. Pass.

**P0** | CONSTITUTION X (Respect API Limits): honor rate limits, no rogue retry
→ **spec.md § Non-Behaviors + plan.md § Risks**: 031 adds no retry/backoff/sleep — it classifies a surfaced 429 as `RateLimited` and defers all retry to 017's bounded handler. Pass.

**P0** | CONSTITUTION XII (Standalone Executable): no new runtime dependency
→ **plan.md + interface-spec.md**: the work is pure `internal/cli` Go over stdlib `errors.As`; no new external/runtime dependency is introduced. Pass.

**P0** | CONSTITUTION IX (Writes Require Explicit Intent): no mutation side-effect
→ **spec.md § Reporting (the system reads only what it was handed; sends nothing)**: `Diagnose` is a pure value transform with no I/O or mutation. Pass.

**P0** | CONSTITUTION III (token safety, Fail-Safe corollary): failures never leak secrets
→ **interface-spec.md § Error Communication (token-free invariant) + feature: "No diagnostic output carries the auth token"**: every `Cause`/`NextStep`/rendered line is response/path/status only; pinned by a token-never-in-output test. Pass.

---

## Governance Notes

- **No `accords/` directory**: this project has no `accords/governance/done-*.md` files, so done-criteria and cross-reference checks were not generated — constitution checks only. Consider creating `accords/governance/done-{specify,plan,interface,scenarios,tasks}.md` to enable vertical done-criteria checks across the pipeline. (Consistent with all prior specs in this repo — no accords exist.)
- **CONSTITUTION I (Spec Fidelity)**: no applicable checks — 031 adds no command, request, or spec operation; it interprets errors from existing operations and sends nothing.
- **CONSTITUTION VI (Size-Aware by Design)**: no applicable checks — 031 handles no result sets or pagination.
- **CONSTITUTION XI (Governance via Proposals)**: no applicable checks — 031 mutates no governance structure.
- **CONSTITUTION II (machine-parseable emission)**: satisfied at the value level by `Diagnostic`; the per-`--output` machine-parseable *rendering* of failures is deferred to 032 (Output-Aware Failure Rendering) by design — a coherence dependency for analyze to confirm, not a 031 defect.
