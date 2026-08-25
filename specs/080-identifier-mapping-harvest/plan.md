# Plan: Identifier Mapping Harvest

**Feature**: 080-identifier-mapping-harvest
**Role**: Shaper
**Inputs**: spec.md (080), PROJECT.md, DECISIONS.md (174 entries), LEARNINGS.md (2026-08-05 V-facts and S-facts entries, 2026-08-08 W-facts entry), DEPRECATION.md (no relevant deprecations), internal/cli/proposal_reads.go, internal/glassfrog/proposal.go, internal/grammar, spec/glassfrog-api-v5.yaml (`ProposalChange` schema)

---

## System Architecture

The harvest is a new verb leaf on the existing `proposal` group — `proposal identifiers` — that reuses the 056 walked-list machinery end to end and inserts one new pure stage between the walk and the render: a projection that turns the gathered proposals' change sets into the identifier mapping. Nothing upstream of that stage is new, and nothing downstream of it is unusual; the novelty is concentrated in one package.

Flow, in order:

1. **Command layer** (`internal/cli`, new file beside `proposal_reads.go`): a `identifiers` leaf registered on the existing `proposal` group through the registration guard. It carries the same five server-side filters, `--first-page`, and `--per-page` that `proposal list` carries, resolved into a config struct mirroring `proposalsConfig`. Output-format resolution first, `--status` validation second, both pure and pre-request — the 056 error-precedence order unchanged.
2. **Walk** (existing `internal/paging`): the same `GET /proposals` walk `proposal list` performs, always as `paging.All[json.RawMessage]` — each record's raw bytes, exactly the machinery `proposal list`'s machine path already runs. Raw, not typed, because the projection does structural descent over decoded maps regardless, and a typed decode would let one malformed change element fail its whole page against the walk's stop-and-retain contract — capping the harvest permanently (risk H-4). The walk's ending shapes (complete, first-page opt-out, stopped-on-error-with-partial-set) map onto the spec's completeness states; per-record readability is the projection's dimension, below.
3. **Projection** (`internal/harvest`, new package): a pure function from the gathered `[]json.RawMessage` to a `Mapping` — it decodes each record itself, isolating faults at the record: a record that will not read as a proposal is skipped and counted (`Mapping.Unreadable`), every readable one is harvested. Then recursive structural descent through each change's free-form fields, identifier extraction and typing, label collection and tagging, deduplication, bounded provenance, and deterministic ordering. No I/O, no network, no clock; fully unit-testable on fixtures.
4. **Render** (`internal/render` + `internal/cli` output plumbing): machine formats serialize the mapping as a CLI-authored derived document in a `{data: …}` envelope; the human render is a new resource with the summary-vs-listing rule keyed on the mapping's own narrowed flag; operator templates address the same view struct. The existing incomplete-walk and more-pages stderr note idioms apply with harvest wording, *in addition to* the in-document completeness fields.

Two components are explicitly not touched: `proposal list` itself (the spec's non-behavior — the harvest reuses its private helpers where they need no change and copies-with-new-name where wording differs), and the filesystem (the projection and render layers have no write path; statelessness is structural, not disciplined).

---

## Architecture Decisions

### ADR-1: Descend structurally, not by a named nesting key

**Context**: The residue element types arrive only as children nested inside role changes. The contract states the nested-only rule but never names the key the children ride under — `ProposalChange` is `additionalProperties: true` with no per-type field schema, and the observed nesting key is empirical (F3's evidence is live payloads, not the spec). The spec's accord requires descent "to whatever depth they nest."

**Options considered**:
1. **Named-key descent** — recurse only into the observed children key. Precise and cheap, but hard-codes an unpublished key: if the API spells it differently on another change type (or grows a second nesting key), those identifiers silently vanish from the mapping, which is exactly the silent-incompleteness failure the spec's non-behaviors forbid.
2. **Structural descent** — recurse into every nested object and array in a change's fields, wherever found. Identifier keys are recognized wherever they occur; an object carrying a string `type` establishes a new enclosing change type for bare identifiers beneath it. Slightly more work per change, immune to key spelling.

**Decision**: Option 2 — structural descent. The walker visits every map and slice value recursively. Identifier extraction keys on the field-name grammar (ADR-3), not on position; the "current change type" context is updated whenever the walker enters an object whose `type` is a string. Depth is naturally unbounded, satisfying the accord without a depth constant to justify.

**Consequences**: No nesting-key drift risk, and new change types with new nesting shapes are harvested without code change. The cost: text-collection (ADR-4) must be scoped to the object that carries the identifier, not the whole subtree, or a parent role change's name would be recorded as a label on its child accountability. The walker therefore extracts per-object: identifiers and labels are read from the same map, and recursion hands children only the enclosing-type context. Recursion depth is bounded by construction, not by a cap in the walker: `encoding/json` refuses input nested deeper than 10000 levels (verified empirically on go1.26.6 — depth 10000 decodes, 10001 fails with `exceeded max depth`), so a value deeper than the walker comfortably handles can never exist in the decoded maps it consumes. That bound is a property of the decode path, not of the walker — it holds only while the projection consumes `encoding/json`-decoded `map[string]any`; a switch to a streaming or hand-rolled decoder removes it silently, so re-derive the bound before any such switch. Do not add a depth cap: it would be dead code guarding a condition the decoder already rejects (risk H-5).

### ADR-2: A new pure package `internal/harvest` owns the projection; the command stays a thin 056-shaped orchestrator

**Context**: The projection (typing, labels, dedup, ordering) is the feature's substance, and the spec demands determinism and statelessness that must be testable without a network. The repo's precedent is focused single-purpose packages (`paging`, `resolve`, `grammar`) consumed by a thin `internal/cli` layer.

**Options considered**:
1. **Project inside `internal/cli`** — keep it beside the command. No new package, but the pure logic gets entangled with config/render plumbing and the fixture tests inherit the seam machinery.
2. **Project inside `internal/glassfrog`** — beside the `Proposal` model. But that package is the API's response vocabulary; a derived projection that exists in no response would blur its "faithful decode" character.
3. **New `internal/harvest` package** — `Harvest([]glassfrog.Proposal) Mapping`, pure. One new package, but the boundary matches the novelty exactly.

**Decision**: Option 3 — `internal/harvest`. The package exposes the `Mapping` type and the `Harvest([]json.RawMessage) Mapping` function: it consumes each walked record's raw bytes and decodes them itself, because the projection is the layer that knows what "readable enough to harvest" means. A record that will not read as a proposal is skipped and counted (`Mapping.Unreadable`), and a value under an identifier key that is not an exact whole number is excluded and counted (`Mapping.Unrepresentable`) rather than truncated — neither is ever allowed to fail a page or the run — the typed `glassfrog.Proposal` decode is deliberately not in this path, since its strictness bought the harvest nothing and gave one malformed element page-wide blast radius (risk H-4). `Mapping` carries entries, per-type counts, and the two exclusion counts (unreadable records, unrepresentable identifiers) — but nothing about how the walk *ended*: walk-ending completeness (narrowed / first-page / stopped) is the command layer's knowledge, attached at render time; per-record readability and per-value representability are the projection's, reported from inside. Determinism lives here: entries sorted by (element type, number), labels in first-appearance order, all pinned by unit tests including a shuffled-input test.

**Consequences**: The Crafter and Verifier get a projection they can pin with table-driven fixtures (including the V10 string/integer case and the unknown-type case) with zero transport mocking. The command file stays a recognizable sibling of `proposal_reads.go`. One more package in the tree.

### ADR-3: Two typing tables in `internal/harvest`, drift-guarded against the grammar artifact's spec-derived change-type set

**Context**: Typing needs (a) a key-spelling table — the ten observed `…DatabaseId` spellings → element type — and (b) a change-type table — the 21-value contract enum → the element type a bare `databaseId` targets. (a) is empirical (V5); (b) is derivable from the contract, which the repo already carries as the generated `internal/grammar` artifact. Precedent: a drift guard must not hard-code the SoT — derive both sides.

**Options considered**:
1. **Hand tables, no guard** — simplest, but when the contract grows a 22nd change type the harvest silently types its identifiers unknown with nothing failing, and "unknown" degradation becomes invisible drift.
2. **Derive the change-type table structurally from the type name** (strip the `Create/Update/Remove/Move/Elect` prefix) — no table at all, but produces junk for the irregular members (`MoveItems`, `ElectRoleFiller`, `*LinkedRole`), and encodes a naming convention the contract never promises.
3. **Hand tables + a guard test that derives the expected key set from `grammar.Load()`** — the change-type table's keys must equal the grammar artifact's change-type set (which is itself generated from the vendored spec and already drift-guarded). A contract refresh that adds a change type fails the harvest's guard until the table gains a deliberate row.

**Decision**: Option 3. The key-spelling table is marked empirical in its comment (provenance: the 2026-08-05 walk, LEARNINGS V5) and is *not* guarded against anything — there is no SoT for it; an unlisted spelling falls through to the unknown-type path by design, which the spec makes visible rather than silent. The change-type table is guarded against `grammar.Load()`; a new enum member forces a human to decide its element type (or deliberately map it to unknown) rather than letting the default ride.

**Consequences**: The 21-row table is a maintenance point, but a *loud* one. Unknown-typed entries remain reachable in three honest ways: unlisted key spelling, an object with no `type` in scope, and a change type deliberately mapped to unknown. The guard reuses the existing grammar package rather than re-parsing the spec.

### ADR-4: Labels are a recognized naming-field set per change type plus an all-text tagged fallback, collected from the identifier's own object only

**Context**: The spec (clarified) requires: a recognized naming field per change type, tagged as recognized; every other text-valued key beside the identifier recorded as a label tagged with its key; nothing silently dropped. The naming-field vocabulary is empirical — the contract defines no per-type fields.

**Options considered**:
1. **Collect from the whole change subtree** — a parent's name would attach to nested children's identifiers; wrong ownership (see ADR-1 consequences).
2. **Collect from the identifier's own object** — the map that carries the identifier key contributes its string-valued fields: the recognized naming field (if present) tagged recognized, every other string field tagged with its own key. Skip only the structural fields (`id`, `type`, and the identifier keys themselves).

**Decision**: Option 2. The recognized set is a third small table in `internal/harvest` (change type → naming field name), seeded from live payloads during implementation and commented with per-row provenance; it shares ADR-3's guard (its keys must be a subset of the change-type set). Labels are deduplicated per entry by (text, key); each carries the most recent proposal (by the proposal's `created_at`, the one timestamp every proposal has) and a count of proposals that carried it.

**Consequences**: A rationale field does become a label — but a *tagged* one, which is the spec's explicit trade. The downstream matcher (081-era work) can weight recognized labels above fallback ones. Collecting only sibling fields keeps a role's name off its accountabilities' entries.

### ADR-5: Machine output is a CLI-authored derived document in the standard `{data: …}` envelope — an explicit, bounded divergence from the verbatim-server-document invariant

**Context**: Every machine output the CLI has shipped is a faithful server document (018, re-affirmed through 078). The mapping exists in no server response; the spec licenses a derived projection and requires completeness stated in every format.

**Options considered**:
1. **Emit the aggregated proposals plus a mapping side-channel** — preserves verbatim purity but makes the mapping the operator asked for the secondary artifact; absurd for the feature's purpose.
2. **Emit the mapping bare (no envelope)** — signals "not a server document" loudly, but breaks every consumer expectation the `{data: …}` convention has built, for no consumer gain.
3. **Emit `{data: <mapping object>}`** where the mapping object carries `complete`, the boundary cause when partial, per-type counts, and the entries array. The envelope is the CLI's output convention, not a server-fidelity claim; the document's own fields make its derived nature explicit.

**Decision**: Option 3. The verbatim invariant is *scoped*, not weakened: it governs documents that echo a server response, and this command emits none. The interface accord must state the derived-document class explicitly so no future reader "restores" verbatim behavior here, and the completeness fields are in the document (all formats) while the stderr notes (incomplete-walk, more-pages) remain the walk-path idioms they already are.

**Consequences**: First derived machine document in the CLI — a precedent future projection features (per-leaf projections in the backlog) will cite. Consumers parse one envelope everywhere. The interface skill owns the exact field contract.

### ADR-6: One render resource with the summary-vs-listing branch keyed on the mapping's narrowed flag

**Context**: The human render summarizes a whole-history walk and lists entries on a narrowed one (spec, clarified). The render layer's convention is one `Resource` per read result type with compact/full templates.

**Options considered**:
1. **Two resources** (summary resource, listing resource) chosen by the command — two template pairs, and an operator template author faces two shapes for one command.
2. **One resource, one view struct carrying entries + summary counts + a narrowed flag**; the template branches on the flag. One shape for template authors; the branch is whitespace-sensitive template work but the repo already does conditional templates (tree, role).

**Decision**: Option 2. The view always carries everything (entries included, even when the built-in template shows only the summary) so an operator template can render entries from a whole-history walk if the operator wants them — the built-in curation is the built-in renders' behavior, not a data restriction (the 075 template principle, applied silently as conformance).

**Consequences**: One `ResourceIdentifierMapping`, two templates, one view struct. The narrowed flag is set by the command layer from what it knows about how the walk was configured/ended — the projection stays ignorant of it (ADR-2).

---

## Cross-cutting Concerns

**Error handling**: No new Outcome and no new exit code. Pre-request failures (bad `--output`, bad `--status`) are the existing usage errors; walk failures classify exactly as `proposal list`'s do (clean failure before any page; partial-emit + classified non-zero after at least one page via the harvest-worded sibling of `reportIncompleteProposalsWalk`). An unreadable record and an unrepresentable identifier value are reported degradations, not failures: the run exits 0, the document carries each count and withholds the completeness claim, and a stderr note names each — non-zero would make the harvest permanently red for an org whose history carries one bad record, punishing without remedy, while silence would be the truncation CONSTITUTION VI forbids. The empty-history and empty-mapping cases are successes. The three-switch-site registry note (exitcode/errorenvelope/dispatch) is not triggered — nothing new to register.

**Determinism**: owned by `internal/harvest` (sort orders pinned by tests, including a shuffled-input fixture); the command layer adds no iteration-order-dependent behavior (per-type counts render in fixed element-type order).

**Statelessness**: structural — no component in the flow has a filesystem write path. The validation scenario's before/after filesystem check needs no special hooks.

**Testing strategy**: unit fixtures in `internal/harvest` carry the awkward cases (string/integer spelling, unknown type, one-number-two-types, unlabelled entry, nested descent, label tagging, dedup, provenance bounds, determinism). BDD scenarios drive the command surface over the fake transport with multi-page fixtures, mirroring the `proposal_reads_bdd_test.go` seam. The ADR-3/ADR-4 guard tests live with the harvest package. Template rendering gets the whitespace-collapsed assertion treatment where needed (operator-path precedent).

**Configuration**: none beyond the inherited persistent flags. No environment variable, no rc-file key — the spec's no-persistent-default posture for siblings applies trivially here since the command has no mode to persist.

---

## Implementation Strategy

**Phase 1 — the projection** (`internal/harvest`): types (`Mapping`, `Entry`, `Label`), the structural walker, the three tables with the grammar-derived guard tests, dedup/provenance/ordering, full fixture coverage. Pure Go, no dependency on cli/render. This phase is the feature's risk concentrated; it can merge alone.

**Phase 2 — the command and machine output** (`internal/cli`): the `identifiers` leaf on the proposal group (registration-guard pattern; group is already runnable-parent-safe), config struct + filter passthrough + validation ordering, the always-decode walk, completeness determination (complete / narrowed-by-filter / first-page / stopped-on-error), the derived `{data: …}` document, stderr notes, BDD wiring. Depends on Phase 1.

**Phase 3 — human render and reference docs** (`internal/render` + docs): `ResourceIdentifierMapping`, view struct, compact/full templates with the narrowed branch, `docs/reference` page, drift-sensitive template exclusions honored (templates ship with no trailing newline; keep them out of whitespace hooks). Depends on Phase 2 for the view's final field set, though template drafting can start against Phase 1's types.

Phases 2 and 3 could land as one PR if the tasks skill judges the diff reviewable; the spec-mandated boundary is only that Phase 1 precedes both.

---

## Risks

- **The naming-field table is guessed wrong** (medium likelihood, low impact): the recognized set is empirical, and a wrong row demotes labels to tagged-fallback status rather than losing them — degradation is visible in the output. Mitigation: seed the table from a live harvest during implementation, record provenance per row.
- **Walk cost on large histories** (low likelihood, medium impact): 1139 proposals ≈ a dozen pages today; the walk is the same one `proposal list` already performs, and the filters + `--first-page` are the pressure valves. No new mitigation needed; the stateless decision (spec-level) already accepted the repeat-walk cost.
- **float64 decode of identifiers** (was uncontrolled in disposal, H-3 — now designed out): the projection's decoded maps carry JSON numbers as float64; legacy ids are ≤8 digits and exact in float64, but the harvest normalizes through a checked exact-whole-number conversion and **excludes and counts** anything that fails it (`Mapping.Unrepresentable` → `boundary.unrepresentable_identifiers`) rather than truncating toward a plausible number or forcing an unkeyable entry. Residual: the exclusion fixture must exist (T001) or the disposal is asserted by nothing.
- **One malformed record capping the walk** (was the sharpest uncontrolled hazard, H-4 — now designed out): the raw walk plus per-record decode isolates the fault at the record, and the skip is counted and reported rather than swallowed. Residual: the skip-and-count fixture must exist (T001) or the isolation is asserted by nothing.
- **Derived-document precedent creep** (low likelihood, medium impact): ADR-5's divergence could be cited to erode the verbatim invariant on echo commands. Mitigation: the interface accord states the class boundary ("derived documents exist only where no server document does"); DECISIONS.md entry records it the same way.
