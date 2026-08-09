# Checklist: Legacy Identifier Request

**Feature**: 075-legacy-identifier-request
**Checked against**: CONSTITUTION.md (12 principles)
**Artifacts checked**: spec.md, plan.md, interface-cli.md, interface-spec.md, tasks.md, features/change-targets-unidentifiable/{legacy-identifier-request,legacy-identifier-absence,legacy-identifier-guard}.feature
**Checks**: 17 (17 pass, 0 fail) + 0 P2 considerations
**Generated**: 2026-08-08 (round 2 — re-derived after the round-1 findings were addressed)

> Source note: no `accords/governance/done-*.md` accords are deployed in this repo, so this checklist runs **constitution checks only**, plus cross-reference checks — the same standing as sibling specs 024/028/029/030/071/072/073.

> Calibration note: the round-1 calibrations are carried forward unchanged (II → "action" = a read invocation carrying the flag, "error" = the pre-request refusal plus existing failure paths; IV → "test" = godog scenario plus Go unit/render tests, "before" = authored during shape). Round 2 is measured against the same bar, not a re-drawn one. All twelve principles still produce at least one applicable check.

---

## Summary

| Severity | Count | Pass | Fail |
|---|---|---|---|
| P0 (blocking) | 13 | 13 | 0 |
| P1 (should fix) | 4 | 4 | 0 |
| P2 (consider) | 0 | — | — |
| **Total** | **17 checks** | **17** | **0** |

**Resolved since round 1**:

- ~~**C2 (P0)**: `getRoleTree` references `IncludeLegacyId` while `TreeNode` declares no `legacy_id`; plan ADR-3 and interface-cli.md committed to the field anyway~~ → **resolved by live probe, in the keep direction.** The read returns the field on all 1336 nodes to depth 6, every value a non-null integer (LEARNINGS 2026-08-08, W1). `TreeNode`'s omission is an upstream schema defect — the fourth instance of the U1–U3 class — recorded in LEARNINGS rather than fixed in the vendored copy, per the standing precedent for that file. plan ADR-3, interface-cli.md § The field, and the tree scenario now state that the field rests on **observed** evidence, so a future reader comparing model to schema does not remove it.
- ~~**C8 (P0)**: only T001 of five Go-touching tasks carried a lint/build gate~~ → fixed. All five now carry "`gofmt -l .` clean and `go test ./...` green before push", with the struct-realignment reason stated inline on T001, T003, and T004 (the tasks that add or reorder struct fields). tasks.md also carries a standing note recording the **verified absence of any pre-commit hook** in this repo, so the gate is not assumed to be automatic.
- ~~**X1 (P1)**: the retired-bridge scenario was owned by two tasks across a phase boundary~~ → fixed by splitting it into a structured-output scenario (T002, Phase 1) and a human-render scenario (T003, Phase 2), each atomically greenable. The identity-read scenario was split the same way. tasks.md gained a per-scenario **Scenario Disposition** table: 28 scenarios, 23 executed each owned by exactly one task, 5 held with a single stated reason (`@validation`).
- ~~**P2-1**: one scenario bundled two behaviors behind a conjunctive title~~ → fixed by the same split.
- ~~**P2-2**: US2 and US3 carried no task label~~ → fixed. tasks.md § Dependency Graph gained a **Story coverage** note naming which tasks serve each of the four user scenarios and why the `[Shared]` labels are the template's multi-story rule rather than an unassigned default.

**Found and fixed during round 2** (not a round-1 finding):

- The decode-tolerance criterion read "either integer or string", but a *non-numeric* string is a string — so `"abc"` would have been accepted and then yielded a silent nil, making breakage indistinguishable from legitimate absence. Narrowed to "an integer-bearing string", with the rejection path required to be tested, an explicit criterion on T003 that a non-numeric value must not render as the absence marker, and a new scenario covering it. This is the guard-message-as-specification pattern applied to an acceptance criterion: the stated intent was right and the wording admitted more than the intent.

---

## Constitution Checks

Thirteen checks, all P0. All pass. Detail is given where this feature is most likely to go wrong; the rest are recorded with source and evidence.

**C1 — Spec Fidelity (I): the requested parameter is real and sent unmodified** — PASS
Source: Principle I. The parameter is `#/components/parameters/IncludeLegacyId`, referenced by exactly six operations. The CLI sends `true` and nothing else.

**C2 — Spec Fidelity (I): every surfaced field is one the API actually returns** — PASS *(was the round-1 P0 failure)*
Source: Principle I ("MUST NOT invent endpoints, parameters, or behaviors the spec does not define"). The tree read's field is now grounded in observation (W1) rather than in the `$ref` enumeration that originally justified it. Fidelity is preserved in the sense the principle protects: the CLI surfaces what the API returns and invents nothing. Where the published contract contradicts observed behavior, the observation is recorded and carried into the work — the U1–U3 precedent — and the vendored artifact is left untouched. If GlassFrog later makes the response match `TreeNode` by dropping the field, the CLI degrades to explicit absence, which is already the designed behavior, so nothing breaks in either direction.

**C3 — Action Transparency (II): every surfaced number is traceable to a resource, and the refusal names what it rejected** — PASS
Source: Principle II (calibrated). The number sits on the resource carrying the stable id, so every value traces to operation + resource id. The refusal is cobra's unknown-flag rejection, classified `UsageError` → exit 2 (`internal/cli/dispatch.go:149` records it as deliberately left on) — landed behavior shared by every flag, not new surface.

**C4 — Fail Safe, Not Silent (III): absence is never an error, and breakage never looks like absence** — PASS
Source: Principle III and its anti-patterns (swallowing errors; a failure reported as success). Three absence causes are legitimate, so reporting them as success is correct. The round-2 narrowing is what makes this check pass cleanly rather than nominally: a non-numeric value now fails loudly instead of decoding to nil and rendering as the same explicit-absence marker a legitimate null produces. Deliberate silence on *retirement* is not a hidden error — it is an absent optional field, with detection delegated to a build-time guard.

**C5 — Test-Driven Development (IV): every user-facing behavior has an executable scenario, written before implementation** — PASS
Source: Principle IV (calibrated). 28 scenarios across three files, all `@wip`, authored before any implementation task. Every accord group has coverage, including the two behaviors discovered during the probe (per-read embed behavior; decode tolerance and its rejection path). Ownership is now verifiable rather than asserted — see X1.

**C6 — Composition over Monolith (V): new invariants land in a sibling guard; no unrelated command is edited** — PASS
Source: Principle V. `internal/build/legacyidcoverage.go` is a sibling of 072's `grammarfacts.go` and 073's `circleroutingrule.go`, following the 071 separate-invariant-separate-file precedent. Command-local registration (ADR-1) means a future supported read touches one registration, not a shared allowlist.

**C7 — Size-Aware by Design (VI): walked reads carry the parameter on every page; nothing silently truncates** — PASS
Source: Principle VI. `internal/paging/paging.go:155` deep-clones `req.Query` per page. T002 requires uniformity across the walk, and the walked-list scenario asserts no page is missing the number while another has it. The tree read's depth coverage is now asserted explicitly ("rows at every depth, not only the root"), matching what the probe observed to depth 6.

**C8 — Working Software (VII): every task making Go changes requires lint and build gates** — PASS *(was the round-1 P0 failure)*
Source: Principle VII. All five tasks carry the gate. The standing note in tasks.md records the verified state — **no `.pre-commit-config.yaml`, no husky/lefthook, and `.git/hooks/` holds only samples; linting exists solely as CI's `lint:` job** — so the gate is stated per task rather than left to a routine that is not installed. The three field-adding tasks carry the PR #164 reason inline.

**C9 — No Fabricated Data (VIII): no value is synthesized, and absence is rendered as absence** — PASS
Source: Principle VIII. Structured output is 018's raw response bytes, so there is no synthesis point by construction. The decode tolerance widens the accepted *spelling* of a value the API sent, not the set of values presented — `"14062695"` and `14062695` are the same datum, and anything that is not an integer in either spelling fails rather than defaulting. Absence uses the settled non-numeric idioms, never `0`.

**C10 — Writes Require Explicit Intent (IX): no read path mutates** — PASS
Source: Principle IX. A `GET` query parameter only; all six operations are `GET`; no body, method, or `If-Match` change; no CLI input accepts a legacy id; the guard is a test and issues no requests.

**C11 — Respect API Limits (X): the feature adds no request and changes no concurrency or retry behavior** — PASS
Source: Principle X. The number rides requests the CLI already makes — no per-resource follow-up, no read-back, no second call. A design that resolved each number through a separate call would have multiplied request volume and violated this principle; that this one does not is the substance of the check.

**C12 — Governance via Proposals (XI): nothing shipped mutates governance structure** — PASS
Source: Principle XI. Read-only throughout; no opt-in bypass flag; no `/proposals` path altered.

**C13 — Standalone Executable (XII): no new runtime dependency** — PASS
Source: Principle XII. The guard parses the vendored spec with `sigs.k8s.io/yaml v1.6.0`, already required and already used by `internal/build/config.go` and `grammarfacts.go` (which also defines `VendoredSpecPath`). The guard ships in no binary.

---

## Cross-Reference Checks

Four checks, all P1. All pass.

**X1 — Every executed scenario is owned by exactly one task** — PASS *(was the round-1 P1 failure)*
tasks.md § Scenario Disposition lists all 28 scenarios. **23 executed**, each with exactly one owning task (T002 ×7, T003 ×7, T004 ×5, T005 ×4). **5 held**, all `@validation` for `/score:validate` — one reason, stated, not a mixed set. Counts reconcile against the files: 12 + 10 + 6 scenarios; 1 + 2 + 2 `@validation`. Phase locality is asserted and holds: every structured-output assertion sits in Phase 1, every human-render assertion in Phase 2, and the two scenarios that straddled the boundary were split one half per phase. T001's absence from the ownership column is stated as deliberate (model-layer change, unit-tested, exercised through T003).

**X2 — Every task carries a plan traceability reference** — PASS
T001 (Phase 1, ADR-3), T002 (Phase 1, ADR-1, ADR-2), T003 and T004 (Phase 2, Render Design), T005 (Phase 3, ADR-4).

**X3 — Every interface surface has an owning task** — PASS
interface-cli.md: The flag / The request / Structured output → T002; The field, incl. decode tolerance and the `TreeNode` note → T001; the twelve per-template rows → T003 (roles, role, tree) and T004 (actors, actor, me); User templates → T004. interface-spec.md: five invariants + the invariant-5 design note + Error Communication → T005.

**X4 — Every ADR has at least one implementing task** — PASS
ADR-1 → T002; ADR-2 → T002; ADR-3 → T001, T003, T004; ADR-4 → T005.

---

## Governance Notes

**Missing done-* accords** — `accords/` does not exist in this repo. Consider creating `accords/governance/done-specify.md`, `done-plan.md`, `done-interface.md`, `done-scenarios.md`, `done-tasks.md`. Round 1's X1 and P2-1 were both defects a `done-scenarios` accord would have caught as first-class checks rather than as a cross-reference finding and a P2 observation.

**No pre-commit tooling** — reported here as project infrastructure, not a feature finding. Linting exists only in CI, so a contributor can commit and push a gofmt violation and learn about it from a red PR check. This repo has already paid one triage round for exactly that (PR #164). Installing a pre-commit hook that runs `gofmt -l` and `go vet` would move the signal left for every future spec; that is a repo-wide workflow decision, so it is surfaced rather than actioned here.

**Principles producing zero applicable checks** — none.

**Verification posture** — five of the six reads' runtime behavior is now observed rather than inferred (LEARNINGS W1–W5). The one remaining contract-only claim is agent-backed nullability (W6): the organization has no `agt_` actors and the list is complete, so the absence-reason rendering is fixture-tested and the claim is marked `[ASSUMED]` in spec.md. Worth re-probing when an agent actor first appears.
