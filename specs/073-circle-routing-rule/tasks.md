# Tasks: Circle Routing Rule

**Feature**: 073-circle-routing-rule
**Concretization**: Full context
**Inputs**: plan.md, spec.md, interface-spec.md, features/proposal-circle-not-choosable/circle-routing-rule.feature, features/proposal-circle-not-choosable/circle-routing-guard.feature

---

## Dependency Graph

Phase 1: The record, its guard, and the supersession (3 tasks, no phase dependencies)
Phase 2: The composed-surface widening and 067 re-validation (3 tasks, gated on **T001 only** — not on all of Phase 1)

6 tasks total | 2 phases | Builder: implement skill (BDD outer loop)

T001 is unblocked. T002 and T003 each depend on T001 and are independent of each other. T004 depends on T001 (the record must declare its named reads first) and on **nothing else in Phase 1** — it does not wait for T002's guard or T003's deprecation entry. T005 depends on T002 and T004. T006 depends on T004.

The phase boundary is a grouping of concerns, not a gate: once T001 lands, T004 can proceed in parallel with T002 and T003. Only T005 genuinely spans the two phases.

**Story coverage**: T001 is `[Shared]` because it serves both US1 (read where the change must land before spending a gated write) and US2 (be told which role to capture a tension on, and how certain that is) — the record carries the rule fields for one and the gap-reporting fields for the other. T002–T006 serve US3 (build the pre-assembly gate against one routing authority). No task serves US1 or US2 alone.

---

## Branching Guidance

**Pipeline mode**: `spec/073-circle-routing-rule/base` → `spec/073-circle-routing-rule/task-1` … `…/task-6`.

Phase 1 is one coherent PR-sized change in T-order, matching 072's shape for its sibling record. **Phase 2 should be its own PR** — it edits a validate-pinned 067 surface (the drafter agent's command fence) and reworks that path's drift check, so keeping it separable from the new artifact makes the review of the fence change legible rather than buried under a new file. T004 must land as a single commit: the registry, the fence, and the widened drift check are mutually dependent and CI is red if any one lands alone.

---

## Scenario Disposition

Per-scenario execution disposition across both feature files (24 scenarios). Tasks execute only the scenarios listed against them. The hold set splits by reason — `@validation`-held versus process-inexecutable — rather than being claimed as one group.

**`circle-routing-rule.feature`** (12 — what the record says):

| Scenario | Disposition |
|---|---|
| An own-circle change routes to the parent circle | executed by T001 |
| The document header names its owner and the consumption rule | executed by T001 |
| The classification test distinguishes the two cases | executed by T001 |
| The procedure names its reads in the order they run | executed by T001 |
| The circle-role itself anchors when the operator is Circle Lead | executed by T001 |
| A change to a role inside a circle routes without a parent hop | executed by T001 |
| A circle with no parent yields a stated limit, not a default | executed by T001 |
| A missing anchor is reported with the capture that would close it | executed by T001 |
| An unprovable absence is reported as none found, not none existing | executed by T001 |
| A routing gap does not stop the operator writing anyway | executed by T001 — a content-inspection assertion that the record prescribes no refusal; it needs no guard, because the claim is about what the record says |
| Only routing is recorded, never change-set shape | `@validation` — held for `/score:validate` |
| No statement about a missing role asserts a settled absence | `@validation` — held for `/score:validate` |

**`circle-routing-guard.feature`** (12 — what the guard enforces and what the composed surface agrees on):

| Scenario | Disposition |
|---|---|
| A circle parameter on the create request fails the build | executed by T002 (fixture: a spec copy with an added property) |
| A structurally incomplete record fails the guard *(Outline, 3 examples)* | executed by T002 (fixtures: a record missing a section, a field label, and the named-reads block) — covers conditions 1, 2, and 4 |
| A dropped Role field fails at the section citing it | executed by T002 (fixture) |
| A named read the CLI no longer exposes fails the build | executed by T002 (fixture) |
| An unanchorable command path is reported rather than skipped | executed by T002 (fixture: a three-token leaf) |
| A record without its empirical marker fails the build | executed by T002 (fixture) |
| A named read missing from the composed registry fails the build | executed by **T005** — condition 7 does not exist until the record↔registry cross-check is added, so the task that turns it green owns it |
| Widening the composed surface leaves the gate posture unchanged | executed by **T004** — asserts the gated-membership posture survives the widening, which is T004's own change |
| Premise dissolution retires the whole record, not one field | **inexecutable by automation** — asserts a future maintenance event against a real contract change that has not happened. Not unmechanized, though: the trigger half is exercised against a fixture by the premise-tripwire scenario above, and the supersession half is carried by T003 and the `@validation` scenarios below |
| The three routing reads appear in both the registry and the agent fence | `@validation` — held for `/score:validate` |
| No workflow step consults the record or runs its reads to route | `@validation` — held for `/score:validate` |
| Nothing the feature ships can refuse a change set locally | `@validation` — held for `/score:validate` |

Totals: **18 executed** (T001 10, T002 6, T004 1, T005 1), **5 held for validate**, **1 process-inexecutable**.

Every element interface-spec.md defines now has either a scenario or an explicit disposition above — the record's full anatomy including the document header, and all nine guard conditions.

---

## Phase 1: The record, its guard, and the supersession

- [x] **T001** [Shared] Author the circle-routing record under the proposal-drafting skill — 10 scenarios, record-side parsers landed in `internal/build/circleroutingrule.go` (T002 extends the same file with the guard)
  - **Scope**: Create `plugin/skills/proposal-drafting/references/circle-routing-rule.md` — the `references/` directory's second member, sibling to `change-set-grammar-facts.md` — with the full anatomy interface-spec.md pins, in order: empirical-marker blockquote carrying the cite-versus-observe split, document header (Owner, Contract citations), contract-citations section, rule section, classification-test section, and procedure section with its named-reads block. No other plugin file changes in this task: the registry, the drafter agent, and drafting's SKILL.md stay untouched here, so the record lands before anything claims to run its reads.
  - **Acceptance criteria**:
    - The marker states that the *absent circle parameter* is published contract while *where the proposal lands* is observed behaviour — the split, not a single blanket claim
    - The rule section carries all four required fields: Mechanism, Own-circle consequence, Circle Lead exception, Root circle. The Root circle field states the limit and declines to name a target, naming neither the circle itself nor any other
    - The classification-test section carries Test, "Resolved by" (naming `Role.has_subroles`), and "Parent resolution" (naming `Role.parent_role_id`, and that the own-roles read already carries it for roles the operator fills)
    - The procedure section carries Answer shape (`role_` id for the circle, `ten_` id per anchor), "All anchors named", Gap reporting (capture on the *specific* role), and Uncertainty (hedge naming the read, grounded in the own-roles read not following pagination)
    - The named-reads block is a fenced block listing `me roles`, `tension list`, `roles` in procedure order, using the `*-commands.txt` token grammar (single token = top-level, two tokens = `<group> <sub>`)
    - The document header carries an **Owner** line naming the proposal-drafting skill and the symlink consumption rule, and a **Contract citations** line naming the vendored spec — the Owner line is what carries the ownership rule into the artifact itself
    - Contract citations are by schema and property name only — no line numbers, no restated values, no pinned count
    - Nothing in the record prescribes refusing, filtering, or pre-validating a change set
    - Step definitions land for this task's ten scenarios (godog; content assertions against a whitespace-collapsed copy, structural checks against the raw file, per the operator-path BDD convention); `@wip` removed as each passes
  - **Dependencies**: None
  - **Plan reference**: Phase 1 (Implementation Strategy), ADR-1: sibling placement under proposal-drafting, ADR-2: structured markdown with a named-reads block
  - **Scenario references**: circle-routing-rule.feature — all ten non-`@validation` scenarios
  - **Interface references**: interface-spec.md § Surface — The record file (anatomy rows 1–6, the three field tables, the named-reads block)

- [x] **T002** [US3] Add the routing guard with its premise tripwire and citation anchors — 6 scenarios (incl. 3-example Outline), conditions 1–6/8–9 in `CheckCircleRoutingRule`, three residues stated in its comment
  - **Scope**: `internal/build/circleroutingrule.go` (record path constant; parsers deriving every side — marker presence and its split phrase, required section presence, required field labels and non-emptiness, the named-reads block's leaves, cited schema anchors; and the spec side — the property-name set of `CreateProposalRequest.properties.proposal`, and the presence of `has_subroles` / `parent_role_id` on `Role`) plus `internal/build/circle_routing_rule_guard_test.go`. Named-read resolution reuses `LiveTopLevelCommands()` (064), `LiveMeSubcommands()` (065), and `LiveTensionSubcommands()` (066) — no new extractors. Derivation helpers in production source, assertions in the test.
  - **Acceptance criteria**:
    - Conditions 1–6 and 8–9 from interface-spec.md fail loudly, each naming the invariant, the offending element, **and which resolution path applies**. Condition 7 (record↔registry agreement) is T005's.
    - The **premise tripwire** asserts set-equality of the create request's whole `proposal` property set against `{tension_id, changes}` — not a search for circle-like key names — and its failure names both sets plus the two resolution paths (re-derive the rule, or retire the record)
    - Named-read resolution has a four-way shape with an **unanchorable-default arm that reports rather than skips**, so a command path the guard cannot anchor fails loudly
    - Every side is derived from its file at test time — no hard-coded read names, property sets, or schema field values anywhere in the guard
    - The three explicitly-partial residues from interface-spec.md are stated in the guard's comment: semantic drift is undetectable, condition 7's reverse direction is unchecked by design, and the reads' flags are unguarded
    - Step definitions land for this task's six scenarios; `@wip` removed as each passes
    - `gofmt -l .` clean and `go test ./...` green before push
  - **Dependencies**: T001
  - **Plan reference**: Phase 1 (Implementation Strategy), ADR-4: sibling guard, everything derived, premise tripwire
  - **Scenario references**: circle-routing-guard.feature: "A circle parameter on the create request fails the build", "A structurally incomplete record fails the guard" (Outline, 3 examples — conditions 1, 2, 4), "A dropped Role field fails at the section citing it", "A named read the CLI no longer exposes fails the build", "An unanchorable command path is reported rather than skipped", "A record without its empirical marker fails the build"
  - **Interface references**: interface-spec.md § Surface — Guard coupling; § Error Communication (conditions 1–6, 8–9 with resolution paths, and the three stated residues)
  - **Risk**: ⚠️ Largest task in the decomposition — two parser sides and eight of the nine conditions. Deliberately not split: the natural seam (record-side consistency versus contract-side citation integrity) runs through the same two files.

- [x] **T003** [US3] Record the LEARNINGS F7 supersession via /score:deprecate — 1 DEPRECATION.md entry; whole-record retirement mechanics and the no-manifest contrast with the sibling stated
  - **Scope**: One `/score:deprecate` entry in `.score/memory/DEPRECATION.md` recording that LEARNINGS 2026-08-05 F7 is superseded by the record at `plugin/skills/proposal-drafting/references/circle-routing-rule.md`. No edit to the LEARNINGS entry itself — it already carries the forward pointer naming this capability, and git history is the changelog.
  - **Acceptance criteria**:
    - The entry names the superseded fact (F7), the superseding record path, and the origin (`073-circle-routing-rule`)
    - The entry states the go-forward retirement mechanics as **whole-record** and its trigger as **premise dissolution** (a circle parameter appearing on proposal create), distinguishing that from a consequence-level change, which is an ordinary edit
    - The entry notes there is no manifest to keep in step, so retirement cannot collide with the guard — the contrast with the sibling record's three-part retirement is stated, so a maintainer does not apply the wrong flow
    - No second copy of the routing content is introduced anywhere
  - **Dependencies**: T001
  - **Plan reference**: Phase 1 (Implementation Strategy), ADR-5: whole-record retirement on premise dissolution
  - **Scenario references**: circle-routing-guard.feature: "Premise dissolution retires the whole record, not one field" (process-inexecutable — this task establishes the recorded mechanics the scenario describes); the supersession half of the `@validation` set
  - **Interface references**: interface-spec.md § Interactions — Maintenance flow

---

## Phase 2: The composed-surface widening and 067 re-validation

- [ ] **T004** [US3] Widen the drafting path's composed surface to the three routing reads
  - **Scope**: Three coupled edits that must land as one commit, because the existing 067 guard is red if any lands alone. (1) `plugin/agents/proposal-drafting-commands.txt` gains `me roles`, `tension list`, `roles`, with a comment annotating them as routing reads the path may run once #77 consults the record. (2) `plugin/agents/proposal-drafter.md` `## Composed commands` gains the same three with the same annotation, preserving the fence's "only these … and it is these and no others" wording. (3) `internal/build/proposaldrafting.go` — `CheckProposalDraftingDrift` gains `liveTop` and `liveMe` parameters and a four-way leaf resolution (top-level, `me <sub>`, `tension`/`proposal <sub>`, unanchorable-default), transplanted from 065's `CheckConstraintDrift`; the guard test supplies `LiveTopLevelCommands()` and `LiveMeSubcommands()`.
  - **Acceptance criteria**:
    - All three edits in one commit; `go test ./...` green — in particular `TestProposalDraftingDriftGuard`, which fails on the registry addition alone (check (a) rejects `me roles` on the group whitelist and `roles` on arity, check (d) requires the agent to name every composed leaf)
    - The widened resolution keeps an unanchorable-default arm that reports rather than skips — no silent caps
    - The gated-membership assertions still hold over the now-seven composed leaves: `proposal create` remains the sole member of 063's gated registry and all six reads remain absent from it
    - The annotation in both the registry and the fence states *why* the reads are present ahead of a workflow that uses them, so a reader does not infer a routing step the path does not perform
    - `gofmt -l .` clean before push — editing `CheckProposalDraftingDrift`'s signature re-aligns its callers
    - Step definitions land for this task's scenario; `@wip` removed when it passes
  - **Dependencies**: T001
  - **Plan reference**: Phase 2 (Implementation Strategy), ADR-3: announced divergence — composed surface widens ahead of the consultation
  - **Scenario references**: circle-routing-guard.feature: "Widening the composed surface leaves the gate posture unchanged"
  - **Interface references**: interface-spec.md § Surface — Composed-surface additions
  - **Risk**: ⚠️ Edits a validate-pinned 067 surface. The fence's permitted-command list grows by three reads the drafter's workflow never uses — capability granted ahead of its use (plan R1), accepted at the developer's direction. Carries T006.

- [ ] **T005** [US3] Cross-check the record's named reads against the composed registry
  - **Scope**: Add condition 7 to `internal/build/circleroutingrule.go` and its guard test — every read the record's named-reads block declares must appear in `plugin/agents/proposal-drafting-commands.txt`. One direction only.
  - **Acceptance criteria**:
    - A named read absent from the registry fails, naming the leaf and the registry path, and naming both resolution paths (add it to the registry and fence, or drop it from the procedure)
    - The reverse direction is **not** asserted, and the guard's comment states why: the registry legitimately carries the drafting path's other composed leaves, and asserting set-equality would require inventing a routing-read delimiter in the registry and would make the guard a second source of truth for which reads the procedure names
    - The comparison is like-with-like — both sides parsed into the same `<group> <sub>` / single-token grammar, never string-matched across two spellings
    - No read name is hard-coded; the record remains the single source of which reads its procedure names
    - Step definitions land for this task's scenario; `@wip` removed when it passes
    - `gofmt -l .` clean and `go test ./...` green before push — this task adds a parser to an existing file, the edit shape that silently re-aligns neighbouring gofmt columns
  - **Dependencies**: T002, T004
  - **Plan reference**: Phase 2 (Implementation Strategy), ADR-4 invariant group 2 (record↔registry agreement half)
  - **Scenario references**: circle-routing-guard.feature: "A named read missing from the composed registry fails the build"
  - **Interface references**: interface-spec.md § Error Communication (condition 7 and its stated reverse-direction partiality)

- [ ] **T006** [US3] Re-validate 067 against the widened drafter fence
  - **Scope**: Re-run 067's validation against the changed `plugin/agents/proposal-drafter.md` command fence and re-examine its STATUS row. This is not a re-implementation — it is confirming that 067's pinned invariants still hold, and recording where they now read differently. No new behaviour.
  - **Acceptance criteria**:
    - 067's validation assertions about the drafter's composed-command fence are re-run against the widened list, and any that pinned the four-leaf list verbatim are updated to the seven-leaf list — **by re-deriving the property each assertion encodes** (the fence permits exactly the path's composed leaves and no others), not by pasting the new list over the old expectation
    - 067's prompt-fence invariants that must *not* change are confirmed unchanged: no `proposal propose`/`respond`/`withdraw`, no tension write, and `proposal create` still the one gated write run only through the confirmed flow
    - The outcome is recorded — either 067's STATUS row stays `Complete` with the re-validation noted, or it moves and the reason is stated
    - Any drift found between 067's shipped artifacts and its validation record is reported rather than silently reconciled
  - **Dependencies**: T004
  - **Plan reference**: Phase 2 (Implementation Strategy), Cross-cutting Concerns — 067 re-validation
  - **Scenario references**: none directly — this task verifies a landed spec's invariants rather than satisfying a 073 scenario
  - **Risk**: ⚠️ Work this spec creates for a landed spec. If skipped, 067's `Complete` status would rest on assertions about text that changed.
