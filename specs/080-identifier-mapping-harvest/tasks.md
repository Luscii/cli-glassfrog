# Tasks: Identifier Mapping Harvest

**Feature**: 080-identifier-mapping-harvest
**Concretization**: Full context
**Inputs**: plan.md, spec.md, interface-cli.md, features/change-targets-unidentifiable/identifier-mapping-harvest.feature, features/change-targets-unidentifiable/identifier-mapping-completeness.feature

---

## Dependency Graph

Phase 1: The projection (2 tasks, no phase dependencies; within the phase T002 depends on T001) [US1, US2, US3]
Phase 2: The command and machine output (1 task, depends on Phase 1) [Shared]
Phase 3: Human render and reference docs (1 task, depends on Phase 2) [US4]

4 tasks total | phases strictly sequential (T002 → T003 → T004 chain; only T001 unblocked at start) | Builder: implement (BDD outer loop)

## Branching Guidance

**Role-based mode**: `spec/080-identifier-mapping-harvest/base` as integration point; task branches `spec/080-identifier-mapping-harvest/task-1` … `…/task-4` in T-order. Two sibling specs await implementation work (075-legacy-identifier-request: Analyzed, full artifact set; 078-invalid-create-outcome: Needs fixes). 080 creates `internal/harvest/` (new, uncontested) and edits `internal/cli` proposal files and `internal/render` — 075's implementation would touch role/actor/me read files and its own feature files in the same `features/change-targets-unidentifiable/` directory; different files, no shared state, but coordinate if landing in the same window. Phase 1 can merge alone (plan: the risk concentrated, independently reviewable).

## Scenario inventory

**`identifier-mapping-harvest.feature`** (12 — what an entry carries): 9 runnable (`@wip`), 3 held (`@validation @wip`).
**`identifier-mapping-completeness.feature`** (13 — how the walk bounds the mapping): 10 runnable (`@wip`), 3 held (`@validation @wip`).
No architecture-informed scenarios remain: both proposals were absorbed into the behavioral accord during the guard rounds, which is the correct migration for a proposal that survives review.
Held scenarios stay `@wip` for /score:validate; the runners' `~@wip` filter never executes them. Runnable scenarios exercising projection logic (T001/T002) become executable only once T003 wires the command — their `@wip` tags are removed in T003, not in the phase that lands the logic.

---

## Phase 1: The projection [US1, US2, US3]

- [ ] **T001** [US1] Create `internal/harvest`: types, structural walker, identifier extraction and typing, and the grammar-derived drift guard
  - **Scope**: New package `internal/harvest` only. `Harvest([]json.RawMessage) Mapping` — the projection consumes each walked record's raw bytes and decodes them itself, skipping and counting any record that will not read as a proposal (`Mapping.Unreadable`), so one malformed element never fails a page or the run (plan ADR-2, risk H-4); identifier normalization is a checked exact-whole-number conversion that **excludes and counts** (`Mapping.Unrepresentable`) anything failing it, never truncating toward a plausible number and never forming an unkeyable entry (risk H-3). `Mapping`/`Entry`/`Label` types (entry keyed by element type + number; `origins` populated only for unknown); the recursive structural walker over a change's decoded fields (descends every nested map/slice, updates the enclosing-change-type context on any object whose `type` is a string — never keyed on a nesting key name, plan ADR-1); identifier extraction from the `…DatabaseId` key-spelling family with normalization of string and numeric spellings through a checked exact-integer conversion (non-integral or out-of-range → unknown handling, never truncation); the two typing tables — key-spelling → element type (empirical, per-row provenance comments, deliberately unguarded) and change-type → element type (all 21 contract members, per-row provenance) — with the guard test deriving the change-type table's expected key set from `grammar.Load()` (plan ADR-3); the closed snake_case element-type vocabulary from interface-cli.md with `unknown` as the never-drop/never-guess fallback.
  - **Acceptance criteria**:
    - Table-driven fixtures pass for: a nested accountability change typed by its own type not its wrapper; a bare `databaseId` typed from the enclosing change type; an unrecognised key spelling → unknown with origins carrying key + change type; one number under two element types → two entries; `"11079492"` (string) and `11079492` (number) → one entry
    - A fixture with `12345.5` under an identifier key yields `Unrepresentable == 1`, **no** entry for it, and no `12345` entry anywhere — the exclusion is asserted positively, not only as the absence of a truncated value (risk H-3)
    - A fixture whose record set contains one element that will not read as a proposal yields `Unreadable == 1` and every identifier from the readable records — the skip-and-count isolation is asserted, not assumed (plan § Risks)
    - The guard test fails if the change-type table's key set differs from the grammar artifact's change-type set (both sides derived, neither hard-coded)
    - Element-type vocabulary is exactly `accountability, circle, domain, person, policy, role, role_note, unknown`
    - No file in `internal/harvest` imports `net/http`, `os`, or any cli/render package; no filesystem write path exists
    - `go test ./internal/harvest/...`, `gofmt -l .` clean, `golangci-lint run ./...` clean
  - **Dependencies**: None
  - **Plan reference**: Phase 1; ADR-1 (structural descent), ADR-2 (pure package), ADR-3 (typing tables + guard)
  - **Scenario references** (logic pinned here by unit fixtures; BDD `@wip` removal happens in T003): identifier-mapping-harvest.feature: "A nested accountability change yields a typed accountability entry", "An untypeable identifier is kept with its element type reported as unknown", "One number seen as two element types yields two entries", "String and integer spellings of one identifier merge into one entry", "A non-integral value under an identifier key routes to unknown rather than truncating"
  - **Interface references**: interface-cli.md: The derived machine document (entry field contract, element_type vocabulary, origins rule)
  - **Risk**: ⚠️ float64 normalization (plan Risks) — the checked-conversion fixture is the pin

- [ ] **T002** [US2] Complete the projection: label collection, bounded provenance, dedup, counts, and deterministic ordering
  - **Scope**: `internal/harvest` only. The recognized naming-field table (change type → naming field; empirical rows seeded from live payloads with per-row provenance; keys guarded as a subset of the change-type set, plan ADR-4); label collection from the identifier's own object only — recognized field tagged `recognized`, every other string-valued field a label tagged with its own key, structural fields (`id`, `type`, identifier keys) skipped; label dedup per entry by (text, key); provenance bounded to most-recent proposal by the proposal's `created_at` plus a proposal count; `counts` (total, unlabelled, by_element_type with every vocabulary member present); deterministic ordering — entries by element type alphabetically then number ascending, labels in first-appearance-in-walk order.
  - **Acceptance criteria**:
    - Fixtures pass for: a renamed identifier carrying both labels, neither marked current; text under an unrecognised key kept as a label tagged with that key, distinguishable from a recognized one; an identifier with no text beside it → empty labels array, nothing synthesized; a label carried by three proposals → `last_proposal_id` of the most recently created + `proposal_count` 3; a parent role change's name NOT attached to its nested child's entry
    - A shuffled-input fixture produces byte-identical serialized output to the ordered-input run
    - `counts` reconcile against `entries` in every fixture (each entry in exactly one element-type count; unlabelled count matches empty-labels entries)
    - The naming-field guard fails on a table key outside the grammar artifact's change-type set
    - `go test ./internal/harvest/...`, `gofmt -l .`, `golangci-lint run ./...` clean
  - **Dependencies**: T001
  - **Plan reference**: Phase 1; ADR-4 (label vocabulary + own-object collection)
  - **Scenario references** (logic pinned here; BDD `@wip` removal in T003): identifier-mapping-harvest.feature: "A renamed identifier carries every name it ever had", "Text under an unrecognised key survives as a label tagged with its key", "An identifier no change ever named appears without a label", "Each label names the most recent proposal that carried it and how many did"; identifier-mapping-completeness.feature: "Two runs over an unchanged history emit identical output"
  - **Interface references**: interface-cli.md: The derived machine document (labels contract, counts contract, ordering)

## Phase 2: The command and machine output [Shared]

- [ ] **T003** [Shared] Register `proposal identifiers`, wire the walk and completeness determination, emit the derived document, and flip the projection scenarios
  - **Scope**: `internal/cli` (new file beside `proposal_reads.go` plus registration wiring) and the BDD step definitions. The `identifiers` verb leaf on the existing `proposal` group (registration guard, `cobra.NoArgs`); the config struct mirroring `proposalsConfig` with the five filters + `--first-page` + `--per-page` (names, help wording, and pass-through semantics in flag parity with `proposal list` — interface-cli.md Surface table); output-format-first then `--status`-validation ordering, both pure and pre-request; the raw walk (`paging.All[json.RawMessage]`, the machinery `proposal list`'s machine path already runs); completeness determination merging the walk configuration and ending with the projection's two exclusion counts (`narrowed` / `first_page` / `stopped` / `unreadable_records` / `unrepresentable_identifiers` as composed dimensions, `complete` iff all clear); the `{data: …}` derived document per the interface field contract; the four harvest-worded stderr note constants; failure classification identical to `proposal list`'s (partial-emit + classified non-zero after ≥1 page; clean failure before). `proposal list` itself untouched — shared helpers reused only where unchanged. Removes `@wip` from every runnable json-path scenario across both feature files.
  - **Acceptance criteria**:
    - Every runnable (`@wip`, non-`@validation`) scenario across both feature files passes and loses `@wip`, except the one human-render scenario T004 owns ("A narrowed walk lists entries where a whole-history walk summarizes") — stated structurally so the criterion cannot go stale against the Scenario inventory
    - A **flag-parity assertion** derives both flag sets from the two constructed cobra commands — `proposal list` and `proposal identifiers` — and fails when the five filters plus `--first-page`/`--per-page` differ in name, value type, or default. Neither side is hard-coded: the test reads both live command trees, so a rename or drop on either command trips it (the derive-both-sides discipline of the `internal/build` drift guards, applied in-package because both commands are constructible there). This is what makes the accord's *"drift between the two flag sets is a defect against this accord"* detectable rather than merely declared
    - A tripwire-transport test pins no request on `--status` rejection; the more-exist note appears only when the API reports a next page under `--first-page`
    - The document always carries `complete`, `boundary` (all five dimensions), `counts`, `entries` — verified against the interface example shape in a decode test
    - No new Outcome, exit code, or envelope kind; `proposal list`'s files show no behavioral diff (its existing tests unchanged and green)
    - `go test ./...`, `gofmt -l .`, `golangci-lint run ./...` clean
  - **Dependencies**: T002
  - **Plan reference**: Phase 2; ADR-5 (derived document), ADR-6 (narrowed flag set by the command layer)
  - **Scenario references**: identifier-mapping-completeness.feature: "A whole-history walk reports the mapping complete", "A mid-walk failure emits the partial mapping and exits non-zero", "An immediate walk failure emits no mapping", "An unsupported status filter is refused before any request", "An unreadable record is skipped and counted, not the end of the walk", "A history carrying no identifiers is an empty mapping, not a failure", "A first-page opt-out on a single-page history is partial and silent", "A first-page opt-out on a longer history notes that more proposals exist"; identifier-mapping-harvest.feature: "The proposal list read is unchanged by the harvest" (held — logic verified here, scenario stays `@validation @wip`)
  - **Interface references**: interface-cli.md: the command + flags table; The derived machine document; Interactions; Error Communication
  - **Risk**: ⚠️ scope creep into `proposal list` — the non-behavior is a hard boundary; copy-with-new-name over edit when wording differs

## Phase 3: Human render and reference docs [US4]

- [ ] **T004** [US4] Add `ResourceIdentifierMapping` with the summary-vs-listing render, wire operator templates, flip the human-render scenarios, and write the reference doc
  - **Scope**: `internal/render` (new resource, view struct, `identifier-mapping.compact.tmpl` + `identifier-mapping.full.tmpl` with the narrowed branch), the human-path wiring in the T003 command file, and `docs/reference/proposal-identifiers.md`. Whole-history walk renders the summary only (per-type counts with zero rows omitted, unlabelled count, completeness line, and each non-zero exclusion count); narrowed walk renders summary + entry listing (element type, number, labels with recognized-or-key tags, absence idiom for unlabelled, origins for unknown); empty mapping renders the existing "none" idiom; the branch keys on the view's narrowed state, never on entry count. The view carries entries always — including on a whole-history walk — so operator templates see the full mapping (075 template principle).
  - **Acceptance criteria**:
    - "A narrowed walk lists entries where a whole-history walk summarizes" passes and loses `@wip`; "The human summary's counts account for every entry" logic verified (scenario stays `@validation @wip`)
    - A template test proves an operator template can render entries from a whole-history walk
    - Templates ship with no trailing newline and are excluded from whitespace fixer hooks alongside the existing `internal/render/templates/*.tmpl` exclusions; human assertions use the whitespace-collapsed helper convention
    - `docs/reference/proposal-identifiers.md` documents the command, flags, derived-document fields, completeness semantics, and exit behavior, citing the 0–7(+8) convention rather than restating it
    - `go test ./...`, `gofmt -l .`, `golangci-lint run ./...` clean
  - **Dependencies**: T003
  - **Plan reference**: Phase 3; ADR-6 (one resource, narrowed-keyed branch)
  - **Scenario references**: identifier-mapping-completeness.feature: "A narrowed walk lists entries where a whole-history walk summarizes", "The human summary's counts account for every entry" (held)
  - **Interface references**: interface-cli.md: Human render; Operator templates
