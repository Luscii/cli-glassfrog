# Analyze: Identifier Mapping Harvest

**Feature**: 080-identifier-mapping-harvest
**Artifacts analyzed**: spec.md, plan.md, interface-cli.md, tasks.md, features/change-targets-unidentifiable/identifier-mapping-harvest.feature (12 scenarios), features/change-targets-unidentifiable/identifier-mapping-completeness.feature (13 scenarios)
**Checklist context**: checklist.md present (round 4: 30 checks, 0 fail) — correlated, not re-evaluated
**Checks**: 16 (16 pass, 0 fail)
**Generated**: 2026-08-23 (round 4 — re-derived after the K5/H3 remedy)

> Scope note: the full artifact set is present, so all 16 base relationship checks run with no skips. One interface file and two feature files, so the interface- and scenario-scaled checks run once each, with scenario checks covering all 22 scenarios across both files.

> Verification note: cross-artifact claims were counted, not inferred — runnable (`@wip`, non-validation) scenarios are 9 in the harvest file and 10 in the completeness file (19 total); the spec-to-feature title diff is empty in both directions (24 titles); the only flags appearing in any scenario step are `--status`, `--first-page`, and `--help`. Round-3 re-checks: C6 was re-evaluated for the new `unrepresentable_identifiers` field (used in one scenario step, defined in the boundary contract — consistent); H1 was re-evaluated for the term *unrepresentable*, which holds its meaning across spec, plan, interface, tasks, and features. Round-2 re-checks: C4 was re-evaluated against the revised ADR-2 and the interface's raw-walk Interactions step (both now say `paging.All[json.RawMessage]` with the projection decoding per record — consistent); C6 was re-evaluated for the new `unreadable_records` field (used in one scenario step, defined in the document contract — consistent).

---

## Summary

| Severity | Count | Pass | Fail |
|---|---|---|---|
| P0 (consistency — contradiction) | 6 | 6 | 0 |
| P1 (completeness — gap) | 6 | 5 | 1 |
| P2 (coherence — drift) | 4 | 3 | 1 |

**Verdict: all checks pass.** Consistency: 6/6. Completeness: 6/6. Coherence: 4/4.

The artifacts tell the same story, every accord promise has a downstream realization, and the two invariants the accord declares are now both detectable. Findings from rounds 1–3 are retained below with their resolutions.

---

## Consistency (P0) — 6/6 pass

| ID | Pair | Result |
|---|---|---|
| C1 | spec § Integration Boundaries ↔ plan § System Architecture | ✓ The API, pagination, and output-serialization boundaries each map to a named component; the three sibling/downstream boundaries (Identifier Resolution, Legacy Identifier Request, Identifier Prompt) are relationships, not components, and the plan correctly architects none of them |
| C2 | spec § Behavioral Accord ↔ plan § System Architecture | ✓ Every accord group has an architectural home — descent (ADR-1), typing and labels (ADR-3/ADR-4), the derived document (ADR-5), the summary-vs-listing branch (ADR-6). No architectural decision contradicts a stated behavior |
| C3 | spec § Non-Behaviors ↔ plan § System Architecture | ✓ The plan architects none of the 13 excluded capabilities, and makes two of them structural rather than disciplinary: no component has a filesystem write path, and `proposal list` is named as untouched. ADR-4's own-object label collection is what enforces the no-enclosing-context non-behavior |
| C4 | plan § Architecture Decisions ↔ interface-cli.md § Surface | ✓ ADR-5's derived document, ADR-6's single resource, and ADR-3's guarded tables each appear in the accord with matching claims. ADR-2 places completeness in the command layer and the accord attaches it at the document's top level — compatible, not contradictory |
| C5 | plan § System Architecture ↔ tasks § Task Scope | ✓ Each of the four components (command layer, walk, projection, render) is claimed by exactly one task; no task builds anything the plan does not name |
| C6 | interface-cli.md § Surface ↔ feature Given/When/Then steps | ✓ Every field referenced in a step (`element_type`, `number`, `labels`, `key`, `recognized`, `last_proposal_id`, `proposal_count`, `origins`, `complete`, `boundary`, `first_page`, `stopped`) is defined in the document contract; every command, flag, and exit code used in a step is defined in the Surface and Error Communication sections |

---

## Completeness (P1) — 5/6 pass

| ID | Trace | Result |
|---|---|---|
| K1 | spec § Driving Scenarios → features | ✓ All 16 driving scenarios have Gherkin equivalents (incl. the round-3 unrepresentable-value and empty-mapping scenarios); title diff empty in both directions |
| K2 | spec § Integration Boundaries → interface files | ✓ The CLI is the only external-facing boundary and has `interface-cli.md`; the remaining five are labelled *(upstream)*, *(internal)*, *(downstream consumer)*, or *(sibling)* in the spec, which is the justified-absence note |
| K3 | plan § Implementation Strategy → tasks | ✓ All three plan phases have task sections with matching names and dependency structure |
| K4 | plan § System Architecture components → tasks | ✓ All four components have implementing tasks |
| K5 | interface-cli.md § Surface → feature coverage | ✓ *(resolved round 4 — see below)* |
| K6 | spec § User Scenarios → interface-cli.md | ✓ All four user scenarios have surface realizations — US1 the command and document, US2 the labels array, US3 the provenance fields, US4 the completeness contract |

### ✓ K5 — RESOLVED (round 4): every interface surface now has coverage or a derived assertion

**Resolution**: item 1 gained a **flag-parity assertion** in T003 that derives both flag sets from the two constructed cobra commands and fails on any divergence in name, value type, or default — neither side hard-coded, so a rename or drop on either command trips it. The accord's Consistency Notes now say so, closing the *"declares a defect condition nothing can detect"* gap. Item 2 was closed in round 3 (empty-mapping scenario). Item 3 gained *"A first-page opt-out on a longer history notes that more proposals exist"*, so the note is now asserted **present** as well as absent — the absence assertion alone had passed trivially against an implementation that never emitted the note. The rounds 1–3 finding follows for the record.

**Rounds 1–3 finding (for the record)**:

**Artifacts**: interface-cli.md § Surface (flags table), § Consistency Notes, § Human render; features/change-targets-unidentifiable/*.feature; tasks.md T003.

**Assertion**: every surface the interface defines has scenario coverage.

**Found** — two uncovered surfaces remain (item 2 was closed in round 3):

1. **Five of the seven flags appear in no scenario.** Only `--status` and `--first-page` are exercised (verified: those two plus `--help` are the only flags in any step). `--role-id`, `--proposer-id`, `--proposed-after`, `--accepted-after`, and `--per-page` have none. Individually this is thin coverage of pass-through values — but the interface raises it above that: *"Flag parity with `proposal list` is deliberate and load-bearing: the five filters keep their exact names, value shapes, and pass-through semantics so an operator's knowledge transfers; drift between the two flag sets is a defect against this accord."* Nothing verifies that invariant. T003 names parity in its **Scope** but not in its **Acceptance criteria**, and no scenario, no fixture, and no guard compares the two flag sets. The accord declares a defect condition that nothing can detect — and the sibling capability's guard (`internal/build`'s derived-both-sides coverage guard for `--legacy-id`) is the established pattern for exactly this shape.

2. **RESOLVED in round 3 — the empty mapping now has a scenario.** Both halves the interface pins (the `total: 0` / empty-`entries` document and the `no identifiers found in the proposal history` render at exit 0) are exercised by *"A history carrying no identifiers is an empty mapping, not a failure"*, and T003 references it. *(Closed jointly with checklist IV.1, which found the same gap from the accord side.)*

3. **The more-exist stderr note is never asserted present.** The architecture-informed scenario asserts its **absence** when the first page was everything; no scenario asserts it appears when the API reports a next page. The absence assertion alone passes trivially against an implementation that never emits the note at all.

**Remedy (outstanding)**: add a parity assertion to T003's acceptance criteria — deriving both flag sets from the live command tree rather than hard-coding either, on the `--legacy-id` coverage-guard precedent — plus one `@wip` scenario for a `--first-page` run where more pages exist.

---

## Coherence (P2) — 3/4 pass

| ID | Scope | Result |
|---|---|---|
| H1 | Terminology across the set | ✓ The five load-bearing concepts (legacy numeric identifier, element type, label, provenance, narrowed/partial/complete) hold their names across all five artifacts. The prose/field split — British *"unrecognised"* in prose against the `recognized` field name — is an identifier-versus-prose distinction, not drift |
| H2 | Detail symmetry | ✓ spec (287 lines) → plan (134) → tasks (87) tapers as expected; no shared topic is treated 3x more deeply in one artifact than its neighbour |
| H3 | Scope alignment across spec + interface + tasks | ✓ *(resolved round 4 — see below)* |
| H4 | Phase coverage, plan ↔ tasks | ✓ Phase names, ordering, and dependency structure match. The plan's note that Phases 2 and 3 *"could land as one PR"* is permissive; tasks keeping them separate is compatible |

### ✓ H3 — RESOLVED (round 4): both scope misalignments closed

**Item 2 resolution (round 4)**: the spec's first-page bullet was split so each consequence carries its own condition — the partial marking is unconditional (the operator bounded the walk), the note is conditional on the page not having been the whole history. The surface reading now matches the interface and both scenarios, with no reliance on the reader knowing the sibling's note discipline. Both cases gained a driving scenario, which also absorbed the last architecture-informed proposal into the accord. The rounds 1–3 detail follows.

**Rounds 1–3 detail (for the record)** *(round 2: item 1 resolved)*

**Item 1 — RESOLVED in round 2.** T003's first acceptance criterion is now stated structurally (*"every runnable scenario across both feature files passes and loses `@wip`, except the one human-render scenario T004 owns"*), which cannot go stale against the Scenario inventory; the inventory itself was re-counted against the files and matches (9+8 runnable, 3+3 held). The round-1 finding is preserved below for the record.

**Item 1 (round 1, resolved) — T003's runnable-scenario count was wrong three ways, leaving three scenarios unclaimed.**

**Artifacts**: tasks.md T003 (first acceptance criterion), T001 and T002 (Scenario references), both feature files.

T003's first acceptance criterion reads: *"The 12 runnable non-human-render scenarios pass and lose `@wip`: the 9 listed under T001/T002 plus [five named]."* Counted against the files:

| Quantity | Stated | Actual |
|---|---|---|
| Scenarios listed under T001 + T002 | 9 | 10 (5 under T001, 5 under T002) |
| Criterion's own arithmetic | 12 | 9 + 5 = 14, so the sentence contradicts itself |
| Runnable scenarios T003 must flip | 12 | 15 (16 runnable total, minus the one human-render scenario T004 owns) |

The consequence is a genuine scope drop, not a typo: a Builder satisfying the criterion literally flips 12 and leaves three runnable scenarios tagged `@wip` with no task claiming them. Because the runners filter `~@wip`, those three would never execute and nothing downstream would notice — the same silent-omission shape the `@wip` discipline exists to prevent. The tasks.md § Scenario inventory block is correct (9 + 7, 3 + 3 held); only T003's criterion disagrees with it.

**Remedy (applied in round 2)**: the structural phrasing was adopted.

**Item 2 (resolved round 4) — The spec's first-page bullet reads two ways; the interface and the scenario pick one.**

**Artifacts**: spec.md § Behavioral Accord → Walking the history (third bullet); interface-cli.md § Error Communication; identifier-mapping-completeness.feature § "The first-page opt-out marks the mapping partial even when the first page was everything".

The spec bullet reads: *"When the operator opts out of the walk after the first page, the mapping is reported as partial and the CLI notes that more proposals exist — exactly as the proposal list read does."* Read as a flat conjunction, both the partial marking and the note are unconditional. The interface splits them — the mapping is always partial, the note fires *"only if the API reported a next page"* — and the architecture-informed scenario asserts the note's absence on a single-page history.

The trailing qualifier resolves the ambiguity correctly (`proposal list`'s note is conditional on `HasNextPage`, verified in `runProposalListFirstPage`), so this is not a contradiction — the artifacts agree once the qualifier is applied. But the resolution depends on the reader already knowing the sibling's behavior, and the bullet's surface reading contradicts the scenario it governs.

**Remedy**: split the spec bullet so the two consequences carry their own conditions, rather than relying on the qualifier to redistribute them.

---

## Checklist Correlation

| Analyze finding | Checklist finding | Relationship |
|---|---|---|
| K5 item 2 (empty mapping has no scenario coverage) | ✗ IV.1 (empty-mapping behavior has no acceptance scenario) | Same gap, found from opposite directions — checklist traced it from the accord, analyze from the interface surface. One scenario closes both |
| — | ✗ III.4 (non-integral value has no representable outcome) | No analyze counterpart: the artifacts are *consistent* in routing the case to "unknown handling"; they are consistently underspecified. A horizontal check cannot catch agreement on an unresolved point |

Checklist's 26 passing checks were not re-evaluated.

---

## Governance Notes

- **No checks skipped.** The full artifact set is present; all 16 base relationship checks ran.
- **One check type has no matrix row for the pair it would need.** Both H3 items concern relationships the matrix does not model directly — tasks ↔ features (item 1) and spec ↔ interface (item 2). They are recorded under H3's scope-alignment assertion, which is the closest traceable row, rather than as invented checks. A matrix carrying an explicit features↔tasks completeness row would have caught item 1 as a P1 gap rather than a P2 drift.

---

## Improvement Summary (round 2 vs round 1)

Previous: 0 P0, 1 P1 (K5, three items), 1 P2 (H3, two items). Current: 0 P0, 1 P1 (K5, three items — unchanged), 1 P2 (H3, one item).

- **Resolved**: H3 item 1 (T003's stale scenario arithmetic) — restated structurally as a side effect of the H-4 edit touching the same criterion; noted transparently rather than silently.
- **Re-verified after the amendment**: C4 and C6 against the revised walk contract and the new boundary dimension; K1 against the widened driving-scenario set; the tasks inventory against the recounted files.
- **Unchanged**: K5's three uncovered surfaces (flag-parity invariant, empty mapping, more-exist note positive case) and H3 item 2 (the first-page bullet's two readings) — neither was in the H-4/H-5 remedy's scope.

---

## Improvement Summary (round 3 vs round 2)

Previous: 0 P0, 1 P1 (K5, three items), 1 P2 (H3, one item). Current: 0 P0, 1 P1 (K5, two items), 1 P2 (H3, one item).

- **Resolved**: K5 item 2 (the empty mapping now has coverage on both halves the interface pins) — closed jointly with checklist IV.1.
- **Re-verified after the amendment**: C6 against the new `unrepresentable_identifiers` field; H1 against the new term; K1 against the widened driving-scenario set (16); the tasks inventory against the recounted files (9+9 runnable).
- **Unchanged and outstanding**: K5's flag-parity invariant (the accord declares a defect condition nothing can detect) and its more-exist-note positive case; H3 item 2 (the first-page accord bullet's two readings). None was in the III.4/IV.1 remedy's scope.

---

## Improvement Summary (round 4 vs round 3)

Previous: 0 P0, 1 P1 (K5, two items), 1 P2 (H3, one item). Current: 0 P0, 0 P1, 0 P2 — all 16 checks pass.

- **Resolved**: K5 item 1 (the flag-parity invariant is now derived-both-sides and detectable), K5 item 3 (the more-exist note is asserted present, not only absent), H3 item 2 (the first-page bullet split so each consequence carries its own condition).
- **Re-verified after the amendment**: K1 against the widened driving-scenario set (18 driving, 24 total spec scenarios); C6 against the two new first-page scenarios; the tasks inventory against the recounted files (9+10 runnable).
- **Cumulative across four rounds**: 5 findings raised, 5 resolved. Two were closed jointly with checklist findings (the empty mapping; nothing else overlapped), and two were closed as side effects of remedies aimed elsewhere — noted at the time rather than silently absorbed.
