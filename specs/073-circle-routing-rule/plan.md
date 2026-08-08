# Plan: Circle Routing Rule

**Feature**: 073-circle-routing-rule
**Role**: Shaper
**Inputs**: spec.md (as amended during this shape run — see its Clarifications), PROJECT.md, DECISIONS.md (477 lines — the 072 ownership/guard precedent and the 063–069 composed-leaf family), LEARNINGS.md (2026-08-05 F7 evidence; the `me roles` pagination lesson), DEPRECATION.md (25 lines, none bearing), vendored `spec/glassfrog-api-v5.yaml` (`CreateProposalRequest`, `Role` — verified during planning). No SOUL.md. Fresh planning run — no prior plan.md.

---

## System Architecture

The feature produces one committed knowledge artifact and one guard, and widens one shipped operator path's composed surface. No runtime code, no CLI change, no new API capability.

**The record** — `plugin/skills/proposal-drafting/references/circle-routing-rule.md`, a hand-authored committed markdown file carrying the routing rule and the procedure for applying it. Three content groups: the **rule** (a proposal inherits the circle of its anchor tension's sensing role; own-circle governance routes to the parent; the Circle Lead exception; the root-circle decline), the **classification test** (whether a target is a circle, and what resolves it), and the **procedure** (which reads to run in what order, how to name the answer, and how to report and hedge a gap). A file-level empirical marker states that the landing behaviour is observed, never v5-contract reference.

**What the record cites versus what it observes** — the distinction is load-bearing and both halves are mechanically checkable:

- *Published contract, cited by schema and property name*: `CreateProposalRequest.properties.proposal` requires only `tension_id` and carries no circle property — the rule's premise. `Role.has_subroles` is the circle indicator; `Role.parent_role_id` is nullable, which is the root-circle signal.
- *Empirical, marked as such*: that the proposal consequently **lands** in the sensing role's circle, the own-circle refusal, and the Circle Lead exception (LEARNINGS F7). The contract says the parameter is absent; only observation says where the proposal goes.

**Ownership** — the Proposal Drafting skill owns it, silent conformance to the ownership rule 072 established (single skill owner by best artifact↔charter match, usually the primary consumer). Charter fit is exact and the primary consumer is the same: the pre-assembly gate (#77) lives inside the drafting path. A **sibling file** to `change-set-grammar-facts.md`, not a section within it (ADR-1) — shape and routing stay structurally separate, which is what makes the spec's no-restatement non-behavior enforceable rather than aspirational. No symlink: the primary consumer is local.

**The composed surface widens here** (ADR-3, developer-directed) — the three reads the procedure names (`me roles`, `tension list`, `roles`) enter `plugin/agents/proposal-drafting-commands.txt` and the drafter agent's `## Composed commands` fence, and 067's drift check widens to resolve them. This is a change to a validate-pinned 067 surface and diverges from the recorded "wiring added by the spec that adds the consumption, never preemptively" precedent; it is announced, not silent, and carries 067 re-validation with it.

**The guard** — a sibling best-effort `internal/build` guard deriving every side from source. Four invariant groups: the record's internal consistency; its named reads resolving on the shipped CLI; a **premise tripwire** against the vendored spec (the create request's property set); and classification-anchor integrity (`has_subroles`, `parent_role_id` still present). The guard hard-codes no read names, no property sets, and no schema values.

**Data flow**: LEARNINGS F7 (provisional) → the record (canonical, this spec) → consumed later by #77, which wires consultation into the drafting workflow. The LEARNINGS copy is superseded via `/score:deprecate` in the same change.

---

## Architecture Decisions

### ADR-1: A sibling record — `references/circle-routing-rule.md` — not a routing section inside the grammar facts record

**Context**: Both artifacts are owned by the same skill, consumed by the same downstream gate (#77), and both are curated empirical knowledge about building a correct proposal. The spec forbids the routing content restating the grammar record's shape facts. 072 established `references/` and the ownership rule.

**Options considered**:
1. **A routing section inside `change-set-grammar-facts.md`** — one artifact for the gate to load; one guard. But the no-restatement boundary becomes a prose convention inside a single file rather than a structural fact, the two retire on different triggers (per-fact absorption versus whole-premise dissolution), and 072's manifest invariant — "the manifest set-equals the actual fact sections" — would have to grow an exception for content that is not a fact.
2. **Sibling file `references/circle-routing-rule.md`** — the no-restatement boundary is structural, the guards stay separate invariants over separate file pairs (071's precedent, which 072 also followed), and each artifact retires on its own trigger. Costs the gate a second file to load.
3. **A new shared subdirectory grouping write-path knowledge** — tidier taxonomy, but invents a second placement convention one spec after `references/` was established, for two files.

**Decision**: Option 2 (developer-confirmed). Silent conformance to 072 ADR-1 on ownership and placement; the sibling-versus-section question is what this ADR resolves.

**Consequences**: Two artifacts under one owner, each with its own guard and its own retirement trigger. #77 loads two files rather than one — a real but small cost, and it is the same skill directory. The `references/` directory gains its second member, confirming it as the convention rather than a one-off.

### ADR-2: Structured markdown with a named-reads block — no per-fact manifest, because the content is one rule rather than an enumerable fact set

**Context**: 072's record carries a self-declared manifest of live fact IDs so that legitimate per-fact retirement passes the guard instead of tripping it. That machinery exists because 072's membership shrinks over time. This record's content is one rule with one procedure: it does not shrink fact by fact, it dissolves entirely if its premise changes.

**Options considered**:
1. **Adopt the manifest pattern for uniformity** — consistent with the sibling; but a manifest over a single logical rule is ceremony, and it would invite splitting one rule into pseudo-facts to populate it.
2. **Structured markdown with required named sections plus an explicit named-reads block** — the guard anchors on section labels for internal consistency, and the enumerable, genuinely-guardable surface becomes the **named reads** (which must resolve on the CLI and agree with the drafting registry) and the **cited schema anchors** (which must still exist). The record stays the single source of which reads the procedure names.
3. **Free prose** — most natural to author, nothing to anchor, structure reverse-engineered later.

**Decision**: Option 2. Field vocabulary and exact section shapes are the interface skill's contract to pin. Contract citations are by schema and property name — never line numbers (they shift on every refresh), never restated values.

The named-reads block is this record's analogue of 072's manifest: the *record* declares which reads its procedure names, and the guard checks that declaration against the CLI and against the drafting registry. The guard never hard-codes the read names, so adding or dropping a read from the procedure is a single deliberate edit in one place.

**Consequences**: The record reads as knowledge and greps as data. Uniformity with the sibling is partial by design, and the reason is written down so the difference is not read as an oversight.

### ADR-3: ANNOUNCED DIVERGENCE — the composed surface widens in this spec, ahead of the consultation that uses it

**Context**: The procedure names three reads the drafter does not currently compose. `plugin/agents/proposal-drafting-commands.txt` is the single source of "which leaves the drafter composes", consumed by both the drafter agent's fence (*"only these … and it is these and no others"*) and 067's drift check. The recorded 072 precedent is explicit: wiring is "added by the spec that adds the consumption, never preemptively."

**Options considered**:
1. **Leave the registry untouched; #77 adds the reads when it wires them** — preserves the recorded precedent and 067's validate-pinned fence; 073's guard asserts only that the named reads resolve as real CLI commands, claiming no composition. The Shaper's recommendation.
2. **Widen now** — the composed surface matches what the content names from the moment the content lands, so there is no window in which the record names reads the path is not permitted to run.
3. **A separate registry owned by 073** — avoids touching 067, but creates a second source of truth for the same path's composed surface, which is the anti-pattern the registry exists to prevent.

**Decision**: Option 2, at the developer's explicit direction after the collision and its costs were surfaced. Recorded as an **announced divergence** from the 072 precedent named above, not as silent conformance.

Concretely, three coupled edits that must land together:
1. the three reads join `proposal-drafting-commands.txt`, annotated as routing reads the path may run once #77 consults the record;
2. the drafter agent's `## Composed commands` fence gains them, so drift check (d) — *the agent must name every composed leaf* — stays green;
3. `CheckProposalDraftingDrift` widens its leaf resolution from `tension <sub>`/`proposal <sub>` to also accept `me <sub>` and top-level commands, transplanting 065's `CheckConstraintDrift` four-way switch including its unanchorable-default arm, and taking `LiveMeSubcommands()` and `LiveTopLevelCommands()` as additional inputs. Both extractors already exist (065, 064) — no new extraction code.

The gated-membership invariant is unaffected and must be re-asserted rather than assumed: the three additions are reads, so `proposal create` remains the only composed leaf in 063's gated set and the read side stays absent from it.

**Consequences**: The drafter agent's permitted surface grows by three reads its workflow does not yet use — a capability granted ahead of its use (R1). 067 must be re-validated, because its fence is a validate-pinned invariant and the fence text changes. In exchange, the record never names a read the path is forbidden to run, and #77 becomes a workflow edit only. The divergence is from a recorded precedent, so the handoff raises formal deprecation as a question for the developer rather than deciding it.

### ADR-4: A sibling `internal/build` guard with a premise tripwire — everything derived, including the reads

**Context**: Two things could silently falsify this record. The API could grow a circle parameter on proposal create, dissolving the rule's premise entirely. Or the Role schema could drop the fields the classification test and the root-circle decline rest on. Neither is visible without a guard, and S7 established that `info.version` and the vendor changelog are unusable drift signals.

**Options considered**:
1. **No guard — review discipline** — cheapest; a refresh that adds a circle parameter would leave the record confidently teaching a rule the API no longer has, and pass CI.
2. **Widen 072's citation-integrity guard** — same neighbourhood and same anchor type, but a different invariant over a different file pair; 071's precedent says that gets a sibling.
3. **Sibling guard, four derived invariant groups** — as below.

**Decision**: Option 3, in `internal/build/circleroutingrule.go` plus its guard test.

1. **Internal consistency** — required sections present and non-empty (rule, classification test, procedure, named reads), the empirical marker present, and the root-circle decline stated. A record missing any of these is incomplete, not merely terse.
2. **Named-read integrity** — every read in the record's named-reads block resolves on the shipped CLI (top-level via `LiveTopLevelCommands()`, `me <sub>` via `LiveMeSubcommands()`, `tension <sub>` via `LiveTensionSubcommands()`), with an unanchorable-default arm that reports rather than skips; and the block set-equals the read leaves 073 added to the drafting registry, so the record and the composed surface cannot disagree about what the procedure names.
3. **Premise tripwire** — the property set of `CreateProposalRequest.properties.proposal`, derived from the vendored spec, must equal `{tension_id, changes}`. Any added property fails the build. This is deliberately a set-equality on the *whole* property set rather than a search for circle-like key names: a circle parameter could arrive under any spelling, and a name-matching check would miss it. The failure message names the resolution path — re-derive the rule against the new parameter, or retire the record (ADR-5).
4. **Classification-anchor integrity** — `has_subroles` and `parent_role_id` still exist on the `Role` schema, since the classification test and the root-circle decline cite them by name.

Nothing is hard-coded: not the read names, not the property set, not the field names. The record and the vendored spec are each parsed at test time.

**Explicitly partial, stated not silent**: the guard cannot detect *semantic* drift — the server changing where a proposal lands while the request schema stays identical is invisible to it, and that is precisely the empirical half of the record. It also does not check the reads' flags (deferred to `--help`, the family convention). Both residues ride in the guard's comment.

**Consequences**: A refresh that adds a circle parameter, or drops either Role field, fails the build at the record and forces re-derivation or retirement. The named-read invariant makes ADR-3's widening self-checking in both directions. The guard joins the family as a second instance of the knowledge↔vendored-contract anchor type 072 introduced, extended with leaf-existence from the 063–069 line.

### ADR-5: Retirement is whole-record, triggered by premise dissolution; landing supersedes the LEARNINGS copy

**Context**: The spec requires that a fact absorbed or invalidated by the contract does not linger. Unlike 072, this record has no per-fact granularity to retire — the rule stands or falls on its premise.

**Options considered**:
1. **Whole-record retirement on premise dissolution** — if proposal create gains a circle parameter, the routing rule is not merely stale, it is wrong: the anchor no longer determines the circle. The record is deleted and a `/score:deprecate` entry records what superseded it. Partial rewrites are possible for the *consequences* (the Circle Lead exception could change independently) and are ordinary edits, not retirements.
2. **In-file retired markers** — dead content accumulates exactly where an agent reads, and a superseded routing rule is more dangerous than a superseded shape fact, because a consumer following it routes to a circle that cannot decide the change.

**Decision**: Option 1. Retirement is a two-part act in one change: delete the record, add the `/score:deprecate` entry naming the spec revision that dissolved the premise. Git history is the changelog — the vendored-spec discipline, and 072's.

On landing, one `/score:deprecate` entry records that LEARNINGS 2026-08-05 F7 is superseded by this record; the LEARNINGS entry already carries the forward pointer naming this capability.

**Consequences**: The record stays live-only. Because retirement deletes rather than edits, it cannot collide with the guard the way 072's early fact-ID design would have — there is no manifest to keep in step. The `/score:deprecate` step stays process discipline the guard cannot see, so the held-out validation scenario carries that check.

---

## Cross-cutting Concerns

**Error handling**: None at runtime — the feature ships no executable behavior. The guard is the only code; it fails loudly with which invariant broke and which of the two resolution paths applies (re-derive against the changed contract, or retire), never partially.

**Testing strategy**: The guard is the mechanical test, a standard `internal/build` Go test compiled into CI. BDD scenarios from the spec's driving scenarios assert the record's content and structure by content inspection, following the operator-path convention — whitespace-normalized copy for content assertions, raw copy for structural checks, with drift-guard helpers in production source rather than `_test.go`. The composed-surface scenario asserts registry and agent agreement, which the widened 067 check also enforces.

**067 re-validation**: ADR-3 changes a validate-pinned surface. 067's own validation assertions about the drafter's fence ("these and no others") must be re-run against the widened fence, and its STATUS row re-examined. This is work this spec creates for a landed spec, and it belongs in the task list, not in a reviewer's memory.

**Configuration**: Nothing configurable. The record's path is a constant shared by the guard and future consumers, single-sourced in `internal/build` alongside the family's other path constants.

**VISION guardrails as structure**: Exclusion 2 (no local governance logic) holds structurally — there is no code path that could refuse a write, because there is no code path. Exclusion 1 (no governance coaching) is a content boundary the BDD scenarios check. Principle 1 (spec fidelity) is carried by the empirical marker the guard asserts present, and sharpened here by the cite-versus-observe split: the contract half is checked against the contract, the observed half is marked as observed.

---

## Implementation Strategy

Two phases. They are separable because the record's guard does not depend on the registry until phase 2 adds the agreement invariant.

1. **The record, its guard, and the supersession** — author `plugin/skills/proposal-drafting/references/circle-routing-rule.md` (rule, classification test, procedure, named-reads block, empirical marker, contract citations per ADR-2); add the sibling `internal/build` guard with invariant groups 1, 3, and 4 plus the CLI-existence half of group 2; record the LEARNINGS F7 supersession via `/score:deprecate` (ADR-5). BDD scenarios from the spec's driving scenarios ride with it.

2. **The composed-surface widening and 067 re-validation** — add the three reads to `proposal-drafting-commands.txt` and the drafter agent's fence; widen `CheckProposalDraftingDrift` to 065's four-way resolution with the two existing live-surface extractors; add the record↔registry agreement half of invariant group 2; re-validate 067 against the changed fence. The two spec scenarios covering the composed surface land here.

Phase 2's real gate is the **record**, not all of phase 1: the record must declare its named reads before anything can agree with them, so the composed-surface widening waits on the record alone and can run in parallel with the guard and the supersession entry. Only the record↔registry cross-check genuinely needs both phases. The phase split groups concerns for review; it is not a serialization boundary.

---

## Risks

- **R1 — The drafter's fence now permits three reads its workflow never uses.** A widened fence is a real expansion of what the subagent may do, granted ahead of the consultation that justifies it. Likelihood certain (it is the design); impact low-moderate — they are reads, absent from 063's gated set, and the agent's workflow gives it no reason to run them. Mitigation: the registry lines and the fence entries are annotated as routing reads pending #77's consultation, so a reader sees the reason rather than inferring a capability the path exercises. Accepted at the developer's direction, not solved.
- **R2 — 067 re-validation is real work this spec creates elsewhere.** Changing a validate-pinned surface invalidates the validation that pinned it. Likelihood certain; impact moderate if forgotten, because 067's Complete status would then rest on assertions about text that changed. Mitigation: it is a task in phase 2, and the handoff names it.
- **R3 — Semantic drift is invisible to the guard.** The server could change where a proposal lands while `CreateProposalRequest` stays byte-identical; the record would then teach a wrong rule with every mechanical check green. Likelihood low, impact high — this is the one failure mode that would send an operator to the wrong circle with confidence. Mitigation: the empirical marker tells a consumer the landing behaviour is observed rather than contracted, and the guard's comment names this residue explicitly. No in-repo check can do better.
- **R4 — The record names reads whose flags are unguarded.** `me roles` in particular carries the pagination limitation the record's hedging requirement rests on; nothing mechanically checks that the limitation still holds. Likelihood low (removing pagination limits is additive and would only make the hedge conservative), impact low. Mitigation: stated in the guard's partial-coverage comment; the hedge is safe under either resolution.
- **R5 — A reader could take the widened composed surface as evidence the drafter routes.** The registry says "which leaves the drafter composes", and three routing reads sitting in it imply a routing step that does not exist until #77. Likelihood moderate; impact low (a wrong inference about capability, not a wrong write). Mitigation: the annotation in both the registry and the fence, and the "ships unconsulted" validation scenario that pins the workflow as unchanged.

---

## What This Plan Does Not Cover

- **The record's exact section vocabulary, field labels, and file-level contract** — the interface skill pins the structural contract the guard and #77 code against.
- **Executable scenarios** — the scenarios skill turns the spec's driving scenarios into .feature files.
- **The consultation itself** — #77 (Pre-Assembly Grammar Consultation) points the drafting workflow at this record and the grammar record, and performs the determination. This plan defines the content it consults and grants the reads it will run; it wires nothing.
- **General vendored-spec drift detection** — the same narrowing 072 recorded: this guard watches only its own citations and premise, not the contract file at large. That remains a separate roadmap item.
- **The identifier problem** — resolving a change target's numeric identifier is #74/#78/#79/#82's work. Routing establishes *which circle and which anchor*, never *which numeric target*.
