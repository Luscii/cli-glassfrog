# Validate: Proposal Drafting Path

**Feature**: 067-proposal-drafting-path
**Round**: 1 of 3
**Date**: 2026-07-18
**Verdict**: Ready
**Artifacts loaded**: spec.md, plan.md, tasks.md (2 of 2 tasks complete), interface-spec.md, features/unequipped-agent-operators/proposal-drafting-path.feature (16 scenarios), PROJECT.md
**Implementation files**: 6 — `plugin/skills/proposal-drafting/SKILL.md`, `plugin/agents/proposal-drafter.md`, `plugin/agents/proposal-drafting-commands.txt`, `internal/build/proposaldrafting.go`, `internal/build/proposal_drafting_bdd_test.go`, `internal/build/proposal_drafting_guard_test.go`

---

## Conformance Summary

| Dimension | Status | Findings |
|---|---|---|
| Driving scenario coverage | ✓ Pass | 0 |
| Acceptance criteria | ✓ Pass | 0 |
| Interface contract conformance | ✓ Pass | 0 |
| Non-behavior absence | ✓ Pass | 0 |
| @wip lifecycle completion | ✓ Pass | 0 |
| **Validation scenarios** | ✓ Satisfied (6 of 6) | 0 |

**Total**: 5 dimensions checked, 5 passed, 0 findings. Full test suite green at inspection time (2062 tests, 12 packages), including the 067 godog suite (16 scenarios) and the standalone drift guard + manifest test.

One advisory observation (non-blocking, not a spec-conformance gap) is recorded under **Observations** below.

---

## Driving Scenario Coverage

**Status**: Pass (8 of 8 spec driving scenarios covered)

The deliverable is declarative (skill + agent prose); coverage is traced to the artifact section that carries the behavior and the BDD scenario that pins it.

| Spec scenario | Status | Implementation |
|---|---|---|
| From a tension to a draft proposal | ✓ Covered | SKILL.md § The workflow (steps 1, 5, 6); agent § Output contract (`draft` element: `prp_…`, `status` `draft`); BDD "A ready tension becomes a created draft proposal" |
| Situating against proposals already in flight | ✓ Covered | SKILL.md § The workflow (step 2: `proposal list --role-id <circle> --status draft`, full-set paging); agent § Output contract (situating element, "drawn together", "already circulating", "deliberate addition"); BDD "A draft is situated against the proposals already in flight" |
| Sourcing the change set from a file | ✓ Covered | SKILL.md § The workflow (step 4: verbatim above the `type` floor, no typed builders); agent § Identity & scope ("no typed per-change validator"); BDD "A file-held change set is passed through verbatim" |
| A create is rejected | ✓ Covered | agent § Drafting defensively ("surface the API failure by name", `403`/`404`/`422`, "creates nothing", "fabricate no `prp_` id"); BDD "A rejected create fabricates no id" |
| A situating read fails | ✓ Covered | agent § Drafting defensively ("surface what the failure was", "gathered so far, flagged incomplete", no invention/abandonment); BDD "A failed situating walk yields a partial picture" |
| The create must be confirmed before it crosses the boundary | ✓ Covered | SKILL.md § Gated-write note (inline `--changes`, declined = nothing created); agent § Confirmation contract; BDD "An unconfirmed create leaves the record untouched" (+ real gate-script exercise) |
| A matching draft is already in flight | ✓ Covered | SKILL.md § The workflow (step 3); agent `surfaced-existing` action, "let the practitioner decide", "silently create a duplicate"; BDD "A matching in-flight draft is surfaced instead of duplicated" |
| The draft is ready to circulate | ✓ Covered | agent § Output contract (handoff element → 068); § Drafting defensively ("never advance, withdraw, or circulate"); BDD "A created draft is handed off without being advanced" |

All eight are additionally pinned by passing godog scenarios in `internal/build/proposal_drafting_bdd_test.go`.

---

## Acceptance Criteria

**Status**: Pass (2 of 2 tasks, all criteria met)

**T001** (checked):
- SKILL.md frontmatter `name: proposal-drafting` + `description` present; the description states the *when* (a well-formed tension ready to become a governance change; assemble the changes and create the draft through a confirmed gated write, returning the `prp_` id) and explicitly excludes the 066 ("capturing, refining, or retiring a tension"), 065 ("does not judge whether an action is allowed or needs a proposal"), 064 ("does not explain the governance around a concern"), and 068/069 ("never advances, responds to, or withdraws a circulating proposal") surfaces.
- Body carries when / single-sourced workflow (ground → situate full-walk → duplicate check → assemble verbatim → surface & gated create → handoff) / delegation / gated-write note; defers to `glassfrog proposal <sub> --help` + `glassfrog tension get --help` and to orientation (062) for output/pagination/exit-code/quoting mechanics.
- Full-set paging before duplicate judgment stated (Constitution VI: never a silent single-page cap); situating narrows by circle + `draft` status and the artifacts state `proposal list` offers no tension filter (spec Assumption 5).
- Gated-write note states the create is a 063-gated governance write, change set passed inline so the confirmation shows the exact payload, declined = nothing created, and an unwieldy change set is surfaced not smuggled through a hidden file (plan ADR-3).
- Agent auto-discovered from `plugin/agents/` — `plugin.json` unchanged (`git diff 6904183..HEAD` names no `plugin/.claude-plugin/plugin.json`; pinned by `TestProposalDraftingKeepsManifestAutoDiscovered` and the BDD registration scenario); degradation to guidance documented in SKILL.md § Delegation.
- Agent frontmatter: `name`, `description`, `tools: Bash, Read, Grep, Glob` — `Bash` present, `Write`/`Edit` absent (inspection + BDD guardrail scenario); body carries Identity & scope (one write: `proposal create`, always gated; never propose/respond/withdraw; never a tension write; never an authority verdict; no typed validator), Workflow by reference, Confirmation contract (inline `--changes`, `action: declined`), Composed commands (the leaves the registry lists — four at 067's landing, seven after 073's routing-read widening; re-validated 2026-08-08, see the Re-validation addendum), and the draft-record Output contract (draft/anchor/situating/action/handoff/notes, ids per element; `action` ∈ created/surfaced-existing/declined/none).
- The one write is `proposal create`; no other proposal-write phrase appears as a command the path runs (the fence names propose/respond/withdraw only to disclaim them — pinned structurally by the composed-set ∩ gated-set = {create} assertion); the created `prp_` id handed to 068, authority questions to 065; workflow single-sourced in the skill, agent references it.
- Siblings untouched: `git diff 6904183..HEAD` names no file under `plugin/skills/orientation/`, `plugin/skills/governance-navigation/`, `plugin/skills/constraint-discovery/`, `plugin/skills/tension-processing/`, `plugin/agents/{governance-navigator,constraint-navigator,tension-processor}.md`, their leaf lists, or `plugin/hooks/`; no `marketplace.json` added.

**T002** (checked):
- `TestProposalDraftingDriftGuard` reads the three registries from source — composed set from `proposal-drafting-commands.txt` (not a hard-coded copy), live surfaces from `newTensionCommand` / `newProposalCommand` (`Use:` tokens resolved; 073 widened the resolution to four-way, adding `LiveTopLevelCommands` / `LiveMeSubcommands` for its routing reads — re-validated 2026-08-08), 063's gated set from `gated-commands.txt`. The one gated write is an explicit contract anchor (`ProposalDraftingGatedWrite = "proposal create"`), pinned like 063's `expectedProposalSurface` and cross-checked against the composed list: it cannot be derived as "the single composed leaf 063 gates" because a situating read shares the `proposal` group with the create, so that derivation would pass if a read were swapped in for the create (PR #160 review, Copilot).
- Gated-membership asserted both directions, anchored on the write leaf: the create anchor must be a composed leaf AND a member of 063's gated set (else it ships unconfirmed), and every other composed leaf (the situating reads) must be absent from the gated set (else situating would prompt). A create dropped from the registry — or swapped for a read — fails; a read pulled into the gated set fails. Empirically confirmed to bite on both regressions (including the read-swap) and pass healthy.
- Existence spans both groups: `tension get` against the tension surface, `proposal list/get/create` against the proposal surface — each resolves (`get <ten-id>`, `list`, `get <prp-id>`, `create <tension-id>`). (After 073's widening, existence spans four surfaces — `me roles` against the me surface, `roles` against the top-level surface, `tension list` against the tension surface — re-validated 2026-08-08.)
- Fails loudly naming the offending leaf; uncovered scope (flags, draft-record prose, confirmation wording, gate-parser robustness) stated in the test comment, not omitted silently.
- Reuses the `internal/build` config-guard home and idiom (mirrors `tension_processing_guard_test.go`; asserts membership where 066 asserted disjointness). Manifest auto-discovery / pure-data-plugin invariants pinned by `TestProposalDraftingKeepsManifestAutoDiscovered`.

---

## Interface Contract Conformance

**Status**: Pass — all specification-touchpoint surfaces present as contracted (`interface-spec.md`).

| Interface surface | Status | Evidence |
|---|---|---|
| Structural layout (skill dir, agent, single-source leaf list; plugin.json untouched; no marketplace.json) | ✓ Conformant | Files present at the contracted paths; `plugin.json` and siblings untouched (git diff) |
| `SKILL.md` frontmatter (`name`, `description` trigger with the four exclusions) | ✓ Conformant | Frontmatter inspected; exclusions worded per interface table |
| `SKILL.md` required sections (When / The workflow / Delegation / Gated-write note) | ✓ Conformant | All four headings present with contracted content |
| `proposal-drafter.md` frontmatter (`name`, `description`, fenced `tools`, `model`) | ✓ Conformant | `tools: Bash, Read, Grep, Glob` — Bash in, Write/Edit out |
| Agent required sections (Identity & scope / Workflow / Confirmation contract / Composed commands / Output contract) | ✓ Conformant | All five present; workflow by reference, not a divergent copy |
| Confirmation contract (narrate anchor + change set; inline `--changes`; declined = outcome) | ✓ Conformant | agent § Confirmation contract |
| Draft-record output shape (draft/anchor/situating/action/handoff/notes, ids per element) | ✓ Conformant | agent § Output contract; BDD "The result is a synthesized draft record, not raw output" |
| Single-source leaf list (two-token leaves, one file two consumers) | ✓ Conformant | `proposal-drafting-commands.txt` consumed by agent § Composed commands + the drift guard |
| Error Communication (declined / create rejected / partial situating / duplicate / handoff / unwieldy change set / drift-guard red) | ✓ Conformant | agent § Drafting defensively + SKILL.md § Gated-write note; drift-guard red on missing leaf or gate-membership violation |

---

## Non-Behavior Absence

**Status**: Pass — every spec § Non-Behaviors exclusion is honored.

| Non-behavior | Status | Evidence |
|---|---|---|
| No advance/withdraw/respond/circulate | ✓ Absent | agent forbids all three; composed set excludes them (composed ∩ gated = {create}, pinned by BDD "The path stops at the created draft") |
| No typed per-change constructor / no type-value or key validation | ✓ Absent | SKILL.md step 4 + agent "no typed per-change validator"; BDD "The path assembles the change set without typed construction" |
| No capture/refine/discard of the anchor tension | ✓ Absent | agent "never perform a tension write"; the composed set's tension leaves are reads only (`tension get` at 067's landing; `tension list` joined as a routing read with 073 — re-validated 2026-08-08) |
| No ungated create | ✓ Absent | Gated-membership invariant + real gate-script "ask" on `proposal create`; BDD "The path routes its one write through the guardrail" |
| No authority judgment | ✓ Absent | agent + SKILL disclaim ruling; hand to 065; BDD "The path drafts without judging authority or coaching" (verdict-phrase denylist clean) |
| No new command/flag/capability; no local governance/permission/validation logic | ✓ Absent | Composes only shipped leaves; drift guard pins existence; deliverable adds no CLI code (plugin tree is pure data) |
| No coaching; no raw unsynthesized dumps | ✓ Absent | agent "does not advise on governance craft or coach"; drawn-together record, "not a concatenation" |
| Spec must not define distribution / delivery form | ✓ Honored | Delivery form (skill+agent) is the shaping decision (ADR-1); no `marketplace.json`; distribution deferred to #70 |

---

## @wip Lifecycle Completion

**Status**: Pass — zero `@wip` tags remain in `proposal-drafting-path.feature` (all 16 scenarios were referenced by the two checked tasks and are now active; `grep -c @wip` → 0). The six `@validation` tags are retained by design (they mark the held-out scenarios; they are not `@wip`).

---

## Validation Scenario Results

**Status**: Satisfied (6 of 6). Traced independently against the implementation.

| @validation scenario (spec.md) | Status | Trace |
|---|---|---|
| No invented surface | ✓ Satisfied | `CheckProposalDraftingDrift` asserts each composed leaf resolves on the live CLI surface; `TestProposalDraftingDriftGuard` green |
| The gated create is routed through the guardrail | ✓ Satisfied | BDD runs 063's real gate script on `proposal create` → "ask"; gated-membership invariant pins create ∈ gated, reads ∉ gated |
| Assembly, not typed construction | ✓ Satisfied | Artifacts state verbatim above the `type` floor, no typed constructor, no key validation; BDD pins the phrases |
| Drafting only, no further transition | ✓ Satisfied | composed ∩ gated = {create} (no propose/respond/withdraw run); hands `prp_` to 068 |
| Drafting, not judging or coaching | ✓ Satisfied | Authority + coaching disclaimers present; authority-verdict denylist clean; defers to 065 |
| Synthesized, not raw | ✓ Satisfied | agent § Output contract "drawn-together … never a concatenation of raw … output"; all six record elements carry ids |

---

## Observations (advisory — non-blocking, not a conformance gap)

- **Drift-check unit test parity.** 066 shipped a dedicated unit test (`TestCheckTensionProcessingDrift`) exercising the pure `CheckTensionProcessingDrift` function across its failure sub-cases, independent of the filesystem. 067's `CheckProposalDraftingDrift` is covered by the BDD drift scenario and the standalone guard against the live registries, and its bite on both regression directions was confirmed empirically during implementation (a temporary in-package test, since removed). Nothing in spec/interface/tasks requires a standalone unit test of the check function, so this is not a conformance finding — but adding one would match the sibling idiom and give the pure logic durable, filesystem-independent regression protection. Owner's call.

---

## Verdict

**Ready.** The implementation conforms to its specification. All 2 tasks are checked; all 8 driving scenarios trace to artifact sections and passing godog scenarios; all interface surfaces are present as contracted; every non-behavior is honored (with the gated-membership boundary pinned structurally against 063's real gate script); zero `@wip` tags remain; all 6 held-out `@validation` scenarios are satisfied by independent trace. The full test suite is green (2062 tests). One advisory observation (drift-check unit-test parity) is recorded above and does not affect the verdict.

**Handoff**: Suggest PR review and merge. The specification loop is closed.

---

## Re-validation addendum — 2026-08-08 (073 T006: the composed surface widened)

**Trigger**: 073 (Circle Routing Rule) changed this validation's pinned surface at the developer's direction (073 plan ADR-3, an announced divergence from the "wiring added by the spec that adds the consumption" precedent): `plugin/agents/proposal-drafting-commands.txt` and the drafter agent's `## Composed commands` fence gained the three routing reads (`me roles`, `tension list`, `roles`), and `CheckProposalDraftingDrift` widened to a four-way leaf resolution taking `LiveTopLevelCommands()` and `LiveMeSubcommands()` as additional inputs.

**What was re-run**: `TestProposalDraftingDriftGuard`, the 067 BDD suite (`proposal_drafting_bdd_test.go` — all 16 scenarios), and 073's gate-posture scenario ("Widening the composed surface leaves the gate posture unchanged"), all green over the widened surface.

**Assertions updated by re-deriving the property, not by pasting the new list** (each edit is marked "re-validated 2026-08-08" in place above):

- *"Composed commands (the four leaves)"* — the property is **the fence permits exactly the leaves the registry lists and no others**; the count was incidental. Re-derived over the seven-leaf registry: the fence's "only these … and it is these and no others" wording is unchanged and drift check (d) confirms the agent names every composed leaf.
- *Existence "spans both groups"* — the property is **every composed leaf resolves on the shipped CLI**; the two-group shape was incidental. Re-derived: existence now spans four surfaces via the four-way resolution, with the unanchorable-default arm reporting rather than skipping.
- *"composed set holds only `tension get` (a read)"* — the property is **the composed set carries no tension write**; the single-leaf shape was incidental. Re-derived: the tension leaves are `get` and `list`, both reads.

**Invariants that must not change, confirmed unchanged**:

- No `proposal propose`/`respond`/`withdraw` and no tension write anywhere in the composed registry or as a runnable command in the agent fence (the fence names them only to disclaim them, as before).
- `proposal create` remains the **sole** member of 063's gated set among the composed leaves, and all six reads are absent from it — the gated-membership invariant re-asserted in both directions over seven leaves rather than assumed (pinned by the drift guard and by 073's gate-posture scenario).
- The drafter's *workflow* is unchanged: no step consults the routing record or runs the routing reads — their presence is capability granted ahead of #77's consultation, annotated as such in both the registry and the fence (073 plan R1/R5).

**Drift found**: none beyond the deliberate widening. The four stale evidence texts above pinned the four-leaf list verbatim; each encoded a property that still holds, so the verdict is unaffected.

**Outcome**: 067's STATUS row stays `Complete`. The row itself is unchanged (no new 067 skill run — this addendum is the record of the re-validation, performed by 073 T006).

---

## Re-validation addendum — 2026-08-22 (079 T003: the pre-assembly gate wired in)

**Trigger**: 079 (Pre-Assembly Grammar Consultation) changed this validation's pinned surfaces per its plan ADR-1 (in-place widening of the drafting path's shipped artifacts): the skill's workflow became the nine-step gate order (route → ground → situate → duplicate check → consult → assemble → match → confirm & create → hand off) with the two-phase relay documented; both frontmatter descriptions widened to the routed entry; the drafter agent's fence grew to eight leaves (`proposal grammar` joined as the consultation read), its output contract gained the `consultation` element (grammar / routing / match parts), its `action` vocabulary grew to seven values (`surfaced-routing-mismatch`, `named-anchors`, `surfaced-dead-shape` joined), and three defensive entries were added; the registry gained the `proposal grammar` line and its routing-block annotation was rewritten as the routing step's named reads. The corresponding superseding note sits on this spec's Entry accord and its entry assumption.

**What was re-run**: the full 067 BDD suite (`proposal_drafting_bdd_test.go`), `TestProposalDraftingDriftGuard`, and 073's gate-posture scenario — all green over the widened surface with **zero guard-code edits** (079 plan ADR-3: `proposal grammar` resolves through the existing `proposal <sub>` arm of the four-way leaf resolution, and the read-side gate-membership check already covers every non-create leaf).

**Disposition of the pinned surfaces** (assertions hold by re-derived property, not by pasted lists):

- *Workflow steps* — the six original step activities (ground, situate, duplicate check, assemble, confirm & create, hand off) remain present with their pinned phrases intact; the three gate steps joined around them. The 067 content assertions are presence-based, so the widening is additive where they reach.
- *Composed commands* — the property is **the fence permits exactly the leaves the registry lists and no others**; re-derived over eight leaves, drift check (d) confirms the agent names every leaf.
- *Output contract* — the six pinned elements (draft/anchor/situating/action/handoff/notes) are unchanged; `consultation` joined as a seventh, present on every action path. The four pinned `action` values are unchanged; three awaiting-direction values joined (a report with a decision pending — deliberately neither success nor failure, per 079's interface accord).
- *Frontmatter descriptions* — the trigger sentence and every boundary sentence retained; the routed entry added (pinned by 079's widened-descriptions scenario).

**Invariants that must not change, confirmed unchanged**: `proposal create` remains the sole gated composed leaf; all seven reads are absent from the gated set (`proposal grammar` is in the gate script's recognized proposal reads, so consultation never prompts); no circulation write and no tension write anywhere in the registry or the fence.

**Superseded evidence**: the 2026-08-08 addendum's third invariant — "the drafter's *workflow* is unchanged: no step consults the routing record or runs the routing reads" — described the pre-079 state and anticipated exactly this change ("capability granted ahead of the consultation"). 079 is that later change; the consulted state is now the pinned one (079's two BDD suites in `internal/build`).

**Outcome**: 067's STATUS row stays `Complete`. This addendum is the record of the re-validation, performed by 079 T003.
