# Risk: Identifier Mapping Harvest

**Feature**: 080-identifier-mapping-harvest
**Round**: 3
**Date**: 2026-08-23
**Artifacts loaded**: spec.md, plan.md, interface-cli.md, PROJECT.md; feature files consulted for the out-of-sequence coverage note (this run followed scenarios and tasks rather than preceding them, so it has more input than a first run usually does)
**Acceptability matrix**: default 3×3 traffic light — no project-level matrix found in PROJECT.md
**Regulatory bridge**: none — PROJECT.md declares no Regulatory Context

> Register history: the register was reviewed with the developer before this artifact was written. Two hazards were raised as uncontrolled (H-4, H-5); the developer directed a remedy, which was applied to plan.md (ADR-1, ADR-2), spec.md, interface-cli.md, the completeness feature file, and tasks.md before this round was recorded. The register below reflects the post-remedy state; the pre-remedy state is preserved in each hazard's detail.

---

## Risk Register

| H | Hazard | Source | Sev | Prob | Controls | Residual |
|---|---|---|---|---|---|---|
| H-1 | A harvested number addresses the wrong governance element in a later write | spec § Non-Behaviors (synthesis, collapsing) | High | Low | RC-2, RC-3, RC-9 | 🟡 Yellow |
| H-2 | A partial mapping is consumed as exhaustive → confident false "no such identifier" | spec § Non-Behaviors; accord § Walking | High | Low | RC-4, RC-5 | 🟡 Yellow |
| H-3 | An unrepresentable identifier value has no defined disposal | interface § `number`; plan § Risks | Med | Low | RC-3, RC-16 | 🟢 Green |
| H-4 | One malformed record caps the whole walk | plan § System Architecture (walk contract) | High | Low | RC-14 | 🟢 Green |
| H-5 | Deeply nested payload exhausts the stack during structural descent | plan ADR-1 (unbounded descent) | Med | Low | RC-15 | 🟢 Green |
| H-6 | The empirical key-spelling table goes stale → growing unknown residue | plan ADR-3 | Low | Med | RC-6, RC-8 | 🟢 Green |
| H-7 | The naming-field table is wrong → labels demoted, matching weakened | plan ADR-4, § Risks | Low | Med | RC-7 | 🟢 Green |
| H-8 | A harvested identifier names a since-deleted element | spec § Non-Behaviors (no liveness check) | Med | High | RC-11 | 🟡 Yellow |
| H-9 | A cache is added later to avoid the expensive re-walk | spec § Non-Behaviors (no store) | High | Low | RC-1 | 🟡 Yellow |
| H-10 | Flag drift from `proposal list` breaks operator knowledge transfer | interface § Consistency Notes | Low | Low | RC-17 | 🟢 Green |
| H-11 | The derived-document precedent is cited to reshape an echo command | plan ADR-5, § Risks | Med | Low | RC-13 | 🟢 Green |
| H-12 | Repeat full walks contribute to org-wide rate-limit exhaustion | spec § Integration Boundaries; CONSTITUTION X | Med | Low | RC-12 | 🟢 Green |
| H-13 | A parent role's name is attributed to a nested child's entry | plan ADR-1 § Consequences | Med | Low | RC-10 | 🟢 Green |

**No Red residuals, and no uncontrolled hazards.** Every hazard in the register now carries at least one control. H-3 was uncontrolled in round 1 and controlled from round 2; H-10 was uncontrolled through round 2 and controlled as of round 3.

---

## Hazard Detail

### H-1 — A harvested number addresses the wrong governance element in a later write

The worst outcome this problem area exists to prevent: a wrong number silently targets the wrong element in a proposal. **Severity High** — a governance write lands on the wrong record (spec § System Overview names this explicitly). **Probability Low** — three independent mistakes would each be required: a synthesized/derived number (forbidden and fenced), a collapsed two-type entry (forbidden, typed keying), or a truncated conversion (checked). **Controls**: RC-2 (typed (element type, number) keying; no merge, no rank), RC-3 (checked exact-integer conversion), RC-9 (the harvest sends no number back — consumption sits behind the drafting path's write gate). **Residual Yellow, accepted**: the CLI cannot control what a downstream consumer does with a correct-but-misread mapping; the server remains the judge of every write.

### H-2 — A partial mapping is consumed as exhaustive

A consumer concludes an identifier does not exist when it merely was not walked to. **Severity High** — a confidently wrong "no such identifier" mid-draft. **Probability Low** — four ending shapes all mark the document. **Controls**: RC-4 (in-document composed completeness: `complete` + four boundary dimensions, in every format), RC-5 (render legibility: summary states coverage; stderr notes). **Residual Yellow, accepted**: honesty is enforced in the artifact; trusting it is the consumer's act.

### H-3 — An unrepresentable identifier value has no defined disposal *(remedied this session)*

**Pre-remedy**: `number` is contractually an integer; a non-integral value was detected (RC-3) but its disposal was undefined — the artifacts said "unknown handling" without naming a shape the contract could hold, and the covering scenario asserted only a negative, so it passed under either reading and tested neither. Uncontrolled in round 1 (checklist III.4). **Remedy applied** (developer-directed): the value is excluded, never truncated and never carried as a null-numbered entry, and the exclusion is counted in `boundary.unrepresentable_identifiers`, which withholds the completeness claim. A companion non-behavior records why this does not contradict the rule against dropping an untypeable identifier — that rule forbids quietness, not exclusion. **Severity Medium** — one identifier absent from the mapping, reported; never a wrong write. **Probability Low** — the disposal is now specified, positively asserted by a scenario, and pinned by a T001 fixture. **Controls**: RC-3 (detects), RC-16 (disposes and reports). **Residual Green.**

### H-4 — One malformed record caps the whole walk *(remedied this session)*

**Pre-remedy**: the typed page decode let one malformed change element fail its page, and the walk's stop-and-retain contract capped the harvest there — permanently, undetectably, in the feature whose value is that one walk bootstraps the whole mapping. Uncontrolled at review time. **Remedy applied** (developer-directed): the walk is raw (`paging.All[json.RawMessage]`), the projection decodes per record, skips and counts the unreadable (`Mapping.Unreadable`), the document carries `unreadable_records` as a fourth boundary dimension withholding the completeness claim, a stderr note names the count, and the run exits 0. Pinned by a T001 fixture and a `@wip` scenario. **Controls**: RC-14. **Residual Green** — worst case is now "some records skipped, counted, reported; mapping otherwise whole."

### H-5 — Deep nesting exhausts the stack *(dissolved by verification this session)*

ADR-1 chose unbounded structural descent. Verified empirically on the repo's toolchain (go1.26.6): `encoding/json` refuses input nested deeper than 10000 levels (10000 decodes, 10001 fails `exceeded max depth`), and a walker of ADR-1's shape survived a max-depth payload. The recursion is bounded by construction — no payload deep enough to matter can exist in the maps the walker consumes. **Controls**: RC-15 (the bound, recorded in ADR-1 with its re-derivation condition: it holds only while the projection consumes `encoding/json`-decoded maps; a decoder switch removes it silently). A depth cap was deliberately NOT added — dead code guarding a rejected condition. **Residual Green.**

### H-6 — The key-spelling table goes stale

An unlisted spelling's identifiers land in `unknown` rather than their type. **Severity Low** — visible degradation, not loss: the entry survives with `origins`. **Probability Medium** — ten spellings are observed, not contractual. **Controls**: RC-6 (unknown path keeps and marks the identifier), RC-8 (the guarded change-type table forces a deliberate row on every contract refresh, a natural review moment for the spelling table beside it). **Residual Green.**

### H-7 — The naming-field table is wrong

A wrong row demotes labels from recognized to tagged-fallback — weaker match evidence, nothing lost. **Controls**: RC-7 (all-text tagged fallback records everything either way). **Residual Green** — degradation is visible in the output and correctable by a one-row edit with provenance.

### H-8 — A harvested identifier names a since-deleted element

The mapping is historical by construction; a resolver may hand a dead target to a write. **Severity Medium** — a failed or invalid write, surfaced by the server (the invalid-create outcome exists for exactly this reporting). **Probability High** — organizations rename and remove elements continuously; a bootstrap over years of history is guaranteed to carry dead entries. **Controls**: RC-11 (the posture: no liveness verdict is offered, provenance is visible, and the spec's non-behavior says why — the server is the judge). **Residual Yellow, accepted with justification**: harm requires the server to *accept* a stale target, and the server's validation plus the read-back verdict are the authoritative backstops; a local liveness check is the governance logic VISION Exclusion 2 forbids.

### H-9 — A cache is added later to avoid the re-walk

The stateless decision makes every run pay the full walk, which is standing pressure to persist. **Severity High** — a store is a second source of truth with stale-number authority (VISION Exclusion 4). **Probability Low** — the boundary is written in four places. **Controls**: RC-1 (statelessness is structural — no component has a write path — and the no-store non-behavior names where persistence must be settled instead). **Residual Yellow, accepted**: a future spec can still decide to persist; the control ensures it cannot happen as drift.

### H-10 — Flag drift from `proposal list` *(remedied this session)*

**Pre-remedy**: the interface declared parity load-bearing and named a defect condition, and nothing verified it — two flag sets maintained independently drift by default, so the Green rating reflected low severity rather than adequacy. **Remedy applied** (developer-directed, closing analyze K5): a parity assertion derives both flag sets from the two constructed cobra commands and fails on any divergence in name, value type, or default. Neither side is hard-coded, so a rename or drop on either command trips it — the derive-both-sides discipline the `internal/build` drift guards use, applied in-package because both commands are constructible there. **Severity Low** — operator friction, not data harm. **Probability Low** — drift now fails the build rather than reaching an operator. **Controls**: RC-17. **Residual Green**, now on adequacy as well as severity.

### H-11 — The derived-document precedent erodes the verbatim invariant

**Controls**: RC-13 (the class boundary — *derived documents exist only where no server document does* — stated in the accord's Consistency Notes and recorded as a DECISIONS.md entry future reviews will surface). **Residual Green.**

### H-12 — Rate-limit contribution

One sequential walk on the existing retrying executor, no concurrency, 429 classified and surfaced (CONSTITUTION X's shape). Filters and `--first-page` are the operator's pressure valves. **Controls**: RC-12. **Residual Green.**

### H-13 — A parent's name attributed to a nested child

Would poison the downstream matcher with wrong-ownership labels. **Controls**: RC-10 (own-object label collection, ADR-4; pinned by a T002 fixture asserting the parent's name is NOT on the child's entry). **Residual Green.**

---

## Controls Index

| RC | Control (assessment level) | Grounding | Mitigates |
|---|---|---|---|
| RC-1 | Statelessness is structural: no component in the flow has a filesystem write path; persistence is named as a separate capability's decision | plan § Cross-cutting; spec § Non-Behaviors | H-9 |
| RC-2 | Entries are keyed (element type, number); no merge, rank, or choice between same-numbered types | spec accord § Typing; interface § entries | H-1 |
| RC-3 | Identifier normalization is a checked exact-integer conversion; non-integral values never truncate into plausible numbers | plan § Risks; tasks T001 | H-1, H-3 (detection only) |
| RC-4 | Completeness is a composed in-document contract (`complete` + narrowed/first_page/stopped/unreadable_records) present in every format | interface § field contract | H-2 |
| RC-5 | Human render states coverage; stderr notes name partiality and skipped records | interface § Human render, § Error Communication | H-2 |
| RC-6 | Untypeable identifiers are kept as `unknown` with origins — visible residue, never a silent drop or guess | spec § Non-Behaviors; interface § origins | H-6 |
| RC-7 | Unrecognised text keys are recorded as tagged labels, never dropped | spec accord § What an entry carries | H-7 |
| RC-8 | The change-type table is drift-guarded against the spec-derived grammar artifact; every contract refresh forces a deliberate row | plan ADR-3; tasks T001 | H-6 |
| RC-9 | The harvest sends no harvested number back to the API; consumption belongs to the gated drafting path | spec § Non-Behaviors; validation scenario | H-1 |
| RC-10 | Labels are collected from the identifier's own object only | plan ADR-4; tasks T002 fixture | H-13 |
| RC-11 | No liveness verdict is offered; provenance is visible; the server judges every target | spec § Non-Behaviors | H-8 |
| RC-12 | One sequential walk on the existing retrying executor; no added concurrency; 429 surfaces classified | plan § Cross-cutting Concerns | H-12 |
| RC-13 | The derived-document class boundary is stated in the accord and recorded in DECISIONS.md | interface § Consistency Notes; DECISIONS.md | H-11 |
| RC-14 | Per-record decode isolation with counted, reported loss: raw walk, projection-owned decode, `unreadable_records` boundary dimension, fixture + scenario | plan ADR-2 (revised); interface § boundary; tasks T001/T003 | H-4 |
| RC-15 | The decode path bounds recursion depth: `encoding/json` rejects nesting beyond 10000 (verified go1.26.6); recorded in ADR-1 with the condition under which it must be re-derived | plan ADR-1 (revised) | H-5 |
| RC-17 | Flag parity is asserted, not merely declared: both flag sets are derived from the two constructed commands and compared on name, value type, and default; neither side is hard-coded | interface § Consistency Notes; tasks T003 | H-10 |
| RC-16 | Counted exclusion of unrepresentable values: no entry, no truncation, no unkeyable entry; the count is a boundary dimension that withholds the completeness claim, with the no-dropping rule's boundary stated so the two rules stay complementary | spec accord § Typing + § Non-Behaviors; interface § `number`, § boundary; tasks T001 fixture | H-3 |

---

## Residual Risk Summary

13 hazards: 9 Green, 4 Yellow, 0 Red, 0 uncontrolled. Every Yellow carries a documented acceptance rationale above, and each rests on a boundary the CLI genuinely cannot cross — what a downstream consumer does with a correct mapping (H-1), whether a consumer trusts an honest completeness claim (H-2), whether the server accepts a stale target (H-8), and whether a future spec decides to persist (H-9). No rating in this register is now contingent on an unresolved finding elsewhere.

## Scenario Coverage Note (out-of-sequence bonus)

This run followed scenarios/tasks, so coverage could be checked once: H-2 (four completeness scenarios + one validation), H-4 (the new unreadable-record scenario + T001 fixture), H-13 (T002 fixture) are covered; H-3 is covered by the round-2 positive-assertion scenario plus a T001 fixture; H-10 is covered by the T003 parity assertion (RC-17). A formal test-gap analysis belongs to a round-2 re-run after implementation.

---

## Improvement Summary (round 2 vs round 1)

Previous: 13 hazards, 15 controls, 0 Red, 5 Yellow, 8 Green — two uncontrolled (H-3, H-10). Current: 13 hazards, 16 controls, 0 Red, 4 Yellow, 9 Green — one uncontrolled (H-10).

- **Newly controlled**: H-3, by RC-16 (counted exclusion of unrepresentable values). Probability drops Medium → Low because the disposal is specified, positively asserted, and fixture-pinned rather than left to the implementer; residual Yellow → Green.
- **Unchanged**: every other hazard, rating, and control. No hazard was added or removed this round.
- **Still uncontrolled**: H-10 (flag parity with `proposal list` is declared load-bearing by the accord and verified by nothing). Green by matrix on low severity, not by adequacy.

---

## Improvement Summary (round 3 vs round 2)

Previous: 13 hazards, 16 controls, 0 Red, 4 Yellow, 9 Green — one uncontrolled (H-10). Current: 13 hazards, 17 controls, 0 Red, 4 Yellow, 9 Green — none uncontrolled.

- **Newly controlled**: H-10, by RC-17 (derived-both-sides flag-parity assertion). Probability drops Medium → Low because drift now fails the build rather than reaching an operator; the Green rating now rests on adequacy as well as low severity.
- **Unchanged**: every other hazard, rating, and control. No hazard was added or removed this round.
- **Across three rounds**: two hazards were designed out entirely (H-4 by per-record decode isolation, H-5 by verifying the stdlib bound already existed) and two moved from uncontrolled to controlled (H-3, H-10). The four remaining Yellows are all accepted on stated grounds, not pending decisions.
