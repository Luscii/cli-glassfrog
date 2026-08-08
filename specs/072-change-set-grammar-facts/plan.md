# Plan: Change-Set Grammar Facts

**Feature**: 072-change-set-grammar-facts
**Role**: Shaper
**Inputs**: spec.md, PROJECT.md, DECISIONS.md (471 lines — operating-surface precedent §408–§465), LEARNINGS.md (2026-08-05 write-path fidelity entries: F5/F6 evidence, S5 spec-refresh narrowing), DEPRECATION.md (6 entries, none bearing), vendored `spec/glassfrog-api-v5.yaml` (ProposalChange schema, verified during planning). No SOUL.md. Fresh planning run — no prior plan.md.

---

## System Architecture

The feature produces one committed knowledge artifact and one guard — no runtime code, no CLI change, no new API surface.

**The record** — `plugin/skills/proposal-drafting/references/change-set-grammar-facts.md`, a hand-authored, committed markdown file carrying exactly the two empirical change-set shapes the published contract does not: the own-circle policy shape (top-level `CreatePolicy`, no `UpdateRole` wrapper — LEARNINGS F5) and the self-targeting role update the server accepts at create but returns invalid (`valid: false`, blocking alert, no transitions — LEARNINGS F6). Each fact carries stable, labelled fields — shape, disposition (accepted / rejected / accepted-but-invalid), symptom, evidence, provenance — plus a file-level empirical marker stating these are observed behaviors, never v5-contract reference. The record also declares a **manifest of its live fact IDs**, so the record — not the guard — stays the single source of truth for *which* facts exist. Where the published contract already carries a shape (the enumerated change types under `ProposalChange.properties.type`; the nested-only accountability/domain types named in that schema's description), the record **cites** the contract by schema and property name — never by line number, never restating the values, and never pinning a count a refresh would silently falsify.

**Ownership and consumption** — the Proposal Drafting skill owns the artifact, under the reference-ownership rule this plan establishes (developer-confirmed, intended to recur): **a single skill owns each shared reference artifact, chosen for the best artifact↔charter match — usually the primary consumer; any other skill consumes through a symbolic link into its own directory** — one source, no copies. The plugin host packages skill-scoped files only (a free-standing `plugin/knowledge/` is not natively supported), and drafting is the charter-fit owner: the facts exist to inform change-set assembly, which is the drafting path's own step. This spec ships the record inert — it creates no consumer symlinks and does not edit drafting's SKILL.md or the drafter agent (wiring consultation is #75/#77's work; the validate-pinned 067 fences are untouched).

**The guard** — a sibling best-effort `internal/build` guard (config-guard family, §417/§427/§439 idiom) that derives every side from source. Two invariant groups: (1) the record's **internal consistency** — its declared manifest set-equals its actual fact sections, the record is non-empty, every fact's required fields are present and non-empty, dispositions come from the closed vocabulary, and the empirical marker is present; (2) a **citation-integrity / retirement tripwire** against the vendored spec — every change type the record cites must exist under `ProposalChange.properties.type`, and the record's nested-only citation must set-equal the nested-only set named in that schema's description (currently the six accountability/domain types, `CreatePolicy` absent). A spec refresh that moves either fails the build and forces re-derivation or retirement, and the failure says which. The guard hard-codes no fact IDs, no enum values, and no type names.

**Data flow**: LEARNINGS F5/F6 (provisional) → the record (canonical, this spec) → consumed later by #75 (renders/wires it into the drafting path) and #83 (scopes typed builders to it). The LEARNINGS copy is superseded via `/score:deprecate` in the same change.

---

## Architecture Decisions

### ADR-1: The Proposal Drafting skill owns the record in its `references/`; the reference-ownership rule is single-skill-owner by charter match, symlink consumption for everyone else

**Context**: The spec requires a single recorded source consumable by future consumers (#75 operating-surface reference, #77 consultation, #83 Go builders) with no second copy to drift. The plugin host packages skill-scoped files only, so the record must live inside some skill's directory. FEATURE-MODEL's parent solution directs the grammar to "where the assembling agent reads it" — the plugin.

**Options considered**:
1. **`plugin/knowledge/` — a new plugin component type** — cleanest ownership story on paper, additive per 062 ADR-2. Invalid: not natively supported by the plugin host (developer-confirmed); files outside skill directories are not reliably packaged/loadable.
2. **`plugin/skills/orientation/references/` + consumer symlinks** — orientation as a generic knowledge home. Works within the host constraint, but only as *custody, not charter*: orientation's scope is cross-cutting CLI-driving knowledge (§412/§464) while the grammar is write-path API-behavior knowledge — a fudge that invites unowned accretion under orientation.
3. **`plugin/skills/proposal-drafting/references/` + symlinks for any later second consumer** — the primary consumer owns it. Charter fit is exact: the facts exist to inform change-set assembly, the drafting path's own step (the LEARNINGS entry names drafting's step 4 as where the facts bear). #75/#77 become local edits to one skill; #83 reads the repo file from Go, indifferent to skill placement.
4. **Repo-level doc (`docs/reference/`)** — wrong audience (Diataxis user docs, generated from pipeline artifacts) and forces #75 to copy content into the plugin, creating the drift the spec forbids.

**Decision**: Option 3 (developer-confirmed). Proposal Drafting owns `plugin/skills/proposal-drafting/references/change-set-grammar-facts.md`.

This resolves under a general rule the developer asked to record for recurrence (see DECISIONS.md): **for shared reference artifacts, a single skill is the owner, chosen for the best artifact↔skill-charter match — usually the primary consumer; other skills consume the artifact through symbolic links** into their own directories. Symlinks are the *reserved* consumption mechanism here, not shipped now: the primary path needs none (the record is already local to drafting), and a later second skill-consumer (e.g., the impact reviewer reading change sets back) adds its symlink in the spec that adds the consumption.

In practice: this spec adds the file (and `references/` — the shipped plugin's first). Drafting's SKILL.md and the drafter agent are not edited; the record is inert until #75/#77 wire consultation, and 067's validate-pinned prompt fences are untouched.

**Consequences**: One source, host-compatible, charter-clean — no custody stretch to police. The `references/` convention enters the shipped plugin. Symlink resolution becomes a host-environment expectation only when a second skill-consumer ships (risk R3 transfers there). The ownership rule is recorded as reusable precedent, closing this discussion for future shared artifacts.

### ADR-2: Hand-authored structured markdown with stable field labels — not a YAML data file, not free prose

**Context**: Three consumers with different needs: #75 wants prose-legible content an agent reads; #83 and the guard want mechanical anchors. §414 sets the content pattern for operating-surface knowledge: hand-authored committed prose, package judgement. There are exactly two facts.

**Options considered**:
1. **Structured markdown** — per-fact `###` sections with labelled fields (Shape / Disposition / Symptom / Evidence / Provenance) and a file-level empirical-marker header. Prose-legible and greppable; the guard anchors on labels, not layout.
2. **YAML/JSON data file (+ rendering later)** — machine-first, strongest parsing story; but two facts of judgement-laden prose don't justify a data schema, and §414's precedent is prose. Over-engineering at this scale.
3. **Free prose** — most natural to author; nothing for the guard or #83 to anchor on; structure would be reverse-engineered later.

**Decision**: Option 1 — structured markdown. Citations to the published contract are by **schema and property name** (`ProposalChange.properties.type.enum`; the nested-only sentence in `ProposalChange`'s description) — never line numbers (they shift on every refresh) and never restated values. The exact field vocabulary and section shapes — including the manifest's form and where it sits (ADR-3) — are the interface skill's contract to pin.

**Consequences**: The record reads as knowledge and greps as data. If #83 later needs richer machine semantics, a structured front-matter block can be added additively; nothing here blocks that.

### ADR-3: A sibling best-effort `internal/build` guard — internal-consistency invariants plus a citation-integrity retirement tripwire, everything derived

**Context**: The spec's Maintenance accord requires retirement when the contract absorbs a fact, and its cite-don't-restate rule requires citations that stay true. Nothing mechanically watches either without a guard. The `internal/build` family (§417, §427, §439, §465, §470) is the repo's established shape for this; 071 set the precedent that a new invariant over a new file pair is a **sibling** guard, never a widened existing one.

**Options considered**:
1. **No guard — review discipline only** — cheapest; but a spec refresh that absorbs or contradicts a cited shape would pass CI silently, exactly the drift S7 proved invisible (`info.version` never moved).
2. **Widen 062's orientation drift guard** — same file neighborhood; but a different invariant over a different file pair (record ↔ vendored spec), which the 071 precedent says gets its own sibling.
3. **Sibling guard, everything derived** — parse the record for its manifest, cited types, and nested-only citation list; parse the vendored spec for the actual enum and nested-only description set; assert record-side ⊆ spec-side (enum), set-equality (nested-only), and the record's internal consistency (manifest ≡ fact sections, non-empty, required fields, closed disposition vocabulary, empirical marker).

**Decision**: Option 3. Every side is derived from its file at test time — the guard hard-codes no enum values, no type names, and **no fact IDs** (the drift-guard-must-not-hardcode-the-SoT discipline, applied to all three).

The fact-ID point is the one that took a correction. An earlier shape had the guard require `CSG-1` and `CSG-2` by name, which made the guard a *second* source of truth for which facts exist and — worse — put it in direct collision with ADR-4: deleting an absorbed fact would fail the build the retirement was supposed to satisfy. Instead the **record declares its own manifest** of live fact IDs and the guard asserts manifest ≡ actual fact sections (plus non-empty: a record with no facts left is a record to delete, not to keep). A deliberate retirement updates both halves and passes; a fact section deleted without its manifest entry — or a manifest entry with no section — fails loudly. The record stays the single source of truth; the guard checks that it agrees with itself.

**Failure messages name the resolution path.** A guard that fires under this design has two legitimate answers — re-derive the citation, or retire the fact — and the message says which applies rather than leaving the reader to infer it from this plan. (A guard's message is a specification; making it state the next step is the same discipline the repo applies to CLI diagnostics.)

**Explicitly partial, stated not silent**: the guard cannot detect the spec *semantically* absorbing CSG-2 (accepted-but-invalid is behavior prose, not a schema key) — that residue stays with the refresh-diff review the LEARNINGS S7 action established.

**Consequences**: A refresh that changes the enum or the nested-only set fails the build at the record, forcing re-derivation or retirement — the Maintenance accord gets a mechanical tripwire for its checkable half, and ADR-4's retirement flow now passes through it rather than colliding with it. The manifest is a self-declaration, so it catches partial edits rather than proving intent (see R4). New guard file joins the family; the partial-coverage statement rides in the guard's comment.

### ADR-4: Retirement removes the fact and records supersession in DEPRECATION.md; landing supersedes the LEARNINGS copy the same way

**Context**: The spec requires that a contract-absorbed fact retires with its supersession recorded, and that the LEARNINGS F5/F6 copy is superseded on landing rather than left to drift.

**Options considered**:
1. **Remove + `/score:deprecate` entry** — the record carries only live facts; git history is the changelog (the same discipline applied to the vendored spec); DEPRECATION.md carries what retired, what replaced it, where it originated.
2. **In-file `retired` status marker** — self-documenting in one place, but dead entries accumulate exactly where an agent reads, and "never carried in two places" erodes as retired shapes linger beside live ones.

**Decision**: Option 1 (recommended during resolve, unobjected). On landing, one `/score:deprecate` entry records that LEARNINGS 2026-08-05 F5/F6 are superseded by the record; the LEARNINGS entry itself already carries the forward pointer.

Future contract absorption is a **deliberate three-part act**, and all three parts belong to the same change:

1. delete the fact section from the record;
2. drop its ID from the record's manifest (ADR-3) — without this the guard fails, which is the point: retirement is explicit, never a side effect of an edit;
3. add the `/score:deprecate` entry naming the spec revision that absorbed the shape.

When the last fact retires, the record itself goes with it — an empty record is deleted, not kept as a shell (the guard's non-empty invariant enforces this).

**Consequences**: The record stays minimal and live-only, and retirement now *passes* the guard instead of tripping it — ADR-3 and this ADR describe one flow rather than two that collide. Historical shapes are recoverable via git and DEPRECATION.md, not by scanning the record past dead entries. Coupling step 3 to steps 1–2 stays process discipline: the guard can see the manifest agree with the sections, but it cannot see whether a deprecation entry was written, so the held-out validation scenario carries that check.

---

## Cross-cutting Concerns

**Error handling**: None at runtime — the feature ships no executable behavior. The guard is the only code, and it fails loudly with which invariant broke (manifest mismatch, missing field, unrecognized disposition, uncited type, nested-only mismatch, absent marker) and which resolution path applies, never partially.

**Testing strategy**: The guard *is* the test — a standard `internal/build` Go test compiled into CI. BDD scenarios (from the spec's driving scenarios) assert the record's content and structure via content inspection, following the operator-path convention: whitespace-normalized for content assertions, raw for structural checks (LEARNINGS/memory: operator-path BDD normalization), with drift-guard helpers in production source, not `_test.go`.

**Configuration**: Nothing configurable. The record's path is a constant the guard and future consumers share; single-sourced in `internal/build` alongside the family's other path constants.

**VISION guardrails as structure**: Exclusion 2 (no local governance logic) is satisfied structurally — there is no code path that could pre-reject a change set, because there is no code path. Principle 1 (spec fidelity) is carried by the empirical marker the guard asserts present.

---

## Implementation Strategy

Single phase — the artifact and its guard are one coherent PR-sized change:

1. **The record + guard + supersession** — author `plugin/skills/proposal-drafting/references/change-set-grammar-facts.md` (two facts, fields per ADR-2, empirical marker, manifest, contract citations); add the sibling `internal/build` guard (ADR-3: internal-consistency invariants plus the citation tripwire, nothing hard-coded); record the LEARNINGS F5/F6 supersession via `/score:deprecate` (ADR-4). BDD scenarios from the spec's driving scenarios ride with it.

No dependencies beyond what is landed. The tasks skill may split the deprecation bookkeeping from the artifact+guard if PR shape favors it, but nothing orders them apart.

---

## Risks

- **R1 — Semantic absorption is invisible to the guard.** If a future spec refresh documents CSG-2's accepted-but-invalid behavior in prose the guard doesn't parse, the fact overstays. Likelihood low (S7 showed the vendor documents late), impact low (an overstayed true fact misleads no one — it just duplicates). Mitigation: the refresh workflow's diff review (LEARNINGS S7 action) plus the guard's comment naming this residue.
- **R2 — The record reads as authority despite the marker.** An agent consuming the rendered reference (#75) could treat empirical shapes as contract. Impact: misplaced trust if the server changes. Mitigation: the empirical marker is a guarded structural invariant, and #75's rendering inherits the marking requirement through the spec's validation scenarios.
- **R3 — Symlink resolution varies by host environment.** The reserved consumption pattern (ADR-1) assumes a consumer-skill symlink resolves where the plugin is installed; a checkout with `core.symlinks=false` (some Windows setups) materializes symlinks as path-text files. Likelihood low for this repo's agent environments (macOS/Linux), impact deferred — **this spec ships no symlink**, and the primary consumer needs none (the record is local to drafting); the risk transfers to whichever spec first adds a second skill-consumer, recorded here so it is weighed there, not rediscovered.
- **R4 — The manifest is a self-declaration, not proof of intent.** The guard catches a *partial* retirement (section deleted, manifest entry left behind, or the reverse) but not a thorough-but-wrong one: an edit that removes both halves passes. Likelihood low, impact moderate (a live fact silently leaves the record). No in-repo scheme can do better — the guard can only check that the record agrees with itself. Mitigation: retirement takes a visible second edit that shows in the diff, ADR-4 requires a deprecation entry alongside it, and the held-out validation scenario checks that the supersession was recorded. Accepted, not solved.

---

## What This Plan Does Not Cover

- **The record's exact field vocabulary, section shapes, and file-level contract** — the interface skill pins the structural contract (interface-spec.md) that the guard, #75, and #83 code against.
- **Executable scenarios** — the scenarios skill turns the spec's driving scenarios into .feature files.
- **Consumer wiring** — #75 (Agent-Facing Grammar Reference: shaping the record for pre-assembly consultation), #77 (Pre-Assembly Grammar Consultation: pointing drafting's workflow at it), and #83 (Typed Change Builders) are separate backlog items; this plan defines the ownership rule and reserved symlink pattern they inherit but builds none of them.
- **The drift check for the vendored spec itself** (S7's diff-the-file argument) — a separate roadmap item; this guard watches only the record's citations, not general spec drift.

**One deliberate narrowing of the spec's deferral, recorded rather than left implicit.** spec.md § Ambiguity Warnings defers "the drift-detection mechanism that triggers retirement" to a separate spec-drift capability. This plan includes a drift mechanism anyway — ADR-3's citation tripwire — on a narrower reading than that sentence states: **general** vendored-spec drift detection (diffing the contract file across refreshes, per S7) remains the separate capability and is out of scope here, while **citation integrity of this record** is inseparable from shipping the record at all, because a record whose citations have gone stale is a record that misinforms the assembler it exists to serve. The spec's deferral is superseded to that extent and no further. Flagged because the distinction lives in this plan, not in the spec sentence — a reader of the spec alone would not derive it.
