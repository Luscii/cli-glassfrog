# Checklist: Circle Routing Rule

**Feature**: 073-circle-routing-rule
**Checked against**: CONSTITUTION.md (12 principles)
**Artifacts checked**: spec.md, plan.md, interface-spec.md, tasks.md, features/proposal-circle-not-choosable/circle-routing-rule.feature, features/proposal-circle-not-choosable/circle-routing-guard.feature
**Checks**: 11 (11 pass, 0 fail) + 0 P2 considerations
**Generated**: 2026-08-08 (round 2 — re-derived after the round-1 findings were addressed)

> Source note: no `accords/governance/done-*.md` accords are deployed in this repo, so this checklist runs **constitution checks only** — the same standing as sibling specs 024/028/029/030/071/072. Done-criteria checks are skipped, not failed.

> Calibration note: this feature ships **no runtime code** — one committed knowledge artifact, one `internal/build` guard, and edits to a shipped operator path's composed surface. Ten principles were calibrated to that shape; two produced zero applicable checks (see Governance Notes). The round-1 calibrations are carried forward unchanged, so round 2 is measured against the same bar rather than a re-drawn one.

---

## Summary

| Severity | Count | Pass | Fail |
|---|---|---|---|
| P0 (blocking) | 11 | 11 | 0 |
| P1 (should fix) | 0 | 0 | 0 |
| P2 (consider) | 0 | — | — |
| **Total** | **11 checks** | **11** | **0** |

---

## Changes Since Previous Run

**Previous** (round 1): 1 P0 fail, 2 P2 considerations
**Current** (round 2): 0 P0 fail, 0 P2 considerations

**Resolved**:
- ~~C8 (P0): tasks.md T005 had no `gofmt`/`go test` acceptance criterion~~ → fixed. T005 now carries "`gofmt -l .` clean and `go test ./...` green before push" with the reason stated inline (the task adds a parser to an existing file — the edit shape that silently re-aligns neighbouring gofmt columns). Verified mechanically: `gofmt`/`go test` now appear against all three Go-touching tasks (T002, T004, T005) and against no others.
- ~~P2-1: three guard conditions (missing section, missing field label, absent named-reads block) had no named scenario~~ → fixed. `circle-routing-guard.feature` gained `Scenario Outline: A structurally incomplete record fails the guard` with three examples, one per condition. The disposition table names it against T002 and marks which conditions it covers.
- ~~P2-2: no task carried [US1] or [US2]~~ → fixed. tasks.md § Dependency Graph gained a **Story coverage** note stating that T001 is `[Shared]` because it serves US1 and US2, naming which fields serve which, and that T002–T006 serve US3. The `[Shared]` label is preserved per the template's multi-story rule; the note supplies the traceability the label alone did not.

---

## Constitution Checks

All 11 pass. Detail is given for the checks whose subject matter is where this feature is most likely to go wrong; the rest are recorded with their source and evidence.

**C1 — Spec Fidelity (I): contract citations anchor to real schema elements, and the observed half is marked as observed** — PASS
Source: Principle I. Artifacts: spec.md § Behavioral Accord, interface-spec.md anatomy rows 1 and 3, guard conditions 8–9. The three cited anchors were verified present in the vendored spec during planning (`CreateProposalRequest.properties.proposal` requires only `tension_id`; `Role.has_subroles`; nullable `Role.parent_role_id`). The empirical marker must state the cite-versus-observe split rather than a blanket claim, and conditions 8–9 make both anchors mechanically checkable.

**C2 — Spec Fidelity (I): no invented command, flag, or parameter** — PASS
Source: Principle I. Artifacts: interface-spec.md § Surface (Invocation: N/A; no `interface-cli.md`), tasks.md T004. The feature adds no command and no flag. The three composed-surface additions name commands the CLI already ships, verified against source: `me roles` (`internal/cli/me_roles.go:91`), `tension list` (`internal/cli/tension_reads.go:280`), `roles` (`internal/cli/roles.go:549`).

**C3 — Action Transparency (II): every guard failure names cause and next step** — PASS
Source: Principle II ("every error MUST explain what went wrong and the next step"). Artifact: interface-spec.md § Error Communication — all nine conditions carry a "Resolution path named" column; plan ADR-4 states the requirement so the interface is not its only home. Condition 8 names both property sets so the drift is readable from the failure itself. The round-2 Scenario Outline extends scenario-level coverage of this property to conditions 1, 2, and 4.

**C4 — Fail Safe, Not Silent (III): no silent skips, and every coverage reduction is stated** — PASS
Source: Principle III (anti-pattern: swallowing errors). Artifacts: interface-spec.md condition 6 (unanchorable command path reports rather than skips) and the three explicitly-stated residues; tasks.md T002 requires the residues in the guard's comment.

**C5 — Test-Driven Development (IV): scenarios precede implementation and every executed scenario has an owner** — PASS
Source: Principle IV. Artifacts: both feature files (written during shape, before implementation); tasks.md § Scenario Disposition. All 24 scenarios carry a disposition; the 18 executed ones are owned by exactly one task each (T001 10, T002 6, T004 1, T005 1), and each owning task's criteria require step definitions with `@wip` removed as each passes. The hold set splits by reason — 5 `@validation`, 1 process-inexecutable — rather than being claimed as one group. Counts reconcile against the files: 12 + 12 scenarios, 2 + 3 `@validation`.

**C6 — Composition over Monolith (V): new invariants land in a sibling guard, not a widened existing one** — PASS
Source: Principle V. Artifacts: plan ADR-4 (option 2, widening 072's guard, considered and rejected on the 071 separate-invariant-separate-file precedent); interface-spec.md § Guard coupling names `internal/build/circleroutingrule.go` and its own test file. The widening of `CheckProposalDraftingDrift` in T004 is leaf-*resolution* accommodating new registry content, not a new invariant added to an existing guard's assertions.

**C7 — Size-Aware by Design (VI): an unprovable absence is never presented as settled** — PASS
Source: Principle VI ("MUST NEVER silently truncate … MUST page through them or clearly signal the boundary"). Artifacts: spec.md § Behavioral Accord (Reporting a gap) and its second non-behavior; interface-spec.md Uncertainty field; scenario "An unprovable absence is reported as none found, not none existing"; validation scenario "No statement about a missing role asserts a settled absence". Grounded in verified source — `internal/glassfrog/me.go` shows the own-roles read decodes the next-page cursor without following it — so the hedge is required by the artifacts rather than assumed.

**C8 — Working Software (VII): every task making Go changes requires lint and build gates** — PASS *(was the round-1 P0 failure)*
Source: Principle VII. Artifact: tasks.md. All three Go-touching tasks now carry the gate: T002, T004 (`gofmt` called out specifically because the signature change re-aligns callers), and T005. T001, T003, and T006 make no Go changes and correctly carry no gate.

**C9 — No Fabricated Data (VIII): the guard derives every side from source** — PASS
Source: Principle VIII. Artifacts: plan ADR-4; interface-spec.md § Guard coupling ("hard-codes no read names, no property sets, and no schema field values"); tasks.md T002 and T005 acceptance criteria. The record remains the single source of which reads its procedure names; T005 explicitly forbids hard-coding them.

**C10 — Writes Require Explicit Intent (IX): the composed-surface additions are reads and the gate posture is asserted unchanged** — PASS
Source: Principle IX. Artifacts: plan ADR-3; interface-spec.md § Composed-surface additions; scenario "Widening the composed surface leaves the gate posture unchanged"; tasks.md T004 criteria. All three additions are reads. The gated-membership invariant is re-asserted rather than assumed: `proposal create` remains the sole composed leaf in 063's gated registry and all six reads remain absent from it.

**C11 — Governance via Proposals (XI): nothing shipped mutates governance or pre-empts the server** — PASS
Source: Principle XI. Artifacts: spec.md non-behaviors 1–2 and 4; validation scenario "Nothing the feature ships can refuse a change set locally"; plan § Cross-cutting Concerns ("no code path that could refuse a write, because there is no code path").

---

## Governance Notes

Separate from feature quality findings.

**Missing done-* accords** — `accords/` does not exist in this repo. Consider creating:
- `accords/governance/done-specify.md`, `done-plan.md`, `done-interface.md`, `done-scenarios.md`, `done-tasks.md`

Each would enable the corresponding skill's done-criteria checks, which are currently skipped for every spec in this repo, not just this one.

**Principles producing zero applicable checks** (2 of 12):
- **X. Respect API Limits** — the feature issues no requests. The record *names* reads that a future consumer will issue, but nothing in this feature calls the API, so there is no `If-Match`, `429`, or `Retry-After` behaviour to check. The record's hedging requirement about the non-paginating own-roles read is checked under C7 (Principle VI), where it belongs.
- **XII. Standalone Executable** — the feature changes no build, no distribution artifact, and no runtime dependency.

**Calibration record** — the ten calibrated principles were interpreted against a no-runtime-code feature: "command" reads as "the artifacts and the guard", "error" reads as "guard failure", and "test" reads as "godog scenario plus `internal/build` guard test". This is the same calibration basis 072 used for its sibling record, so the two are measured against the same bar.
