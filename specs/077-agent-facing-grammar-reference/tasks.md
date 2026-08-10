# Tasks: Agent-Facing Grammar Reference

**Feature**: 077-agent-facing-grammar-reference
**Concretization**: Full context
**Inputs**: `plan.md`, `spec.md`, `interface-cli.md`, `features/unguided-change-construction/agent-facing-grammar-reference.feature`, `PROJECT.md`

---

## Dependency Graph

```
Phase 1: Grammar data foundation (3 tasks, no dependencies) [Shared]
Phase 2: The command and its rendering (3 tasks, depends on Phase 1) [US1, US2, US3]

6 tasks total | T002/T003 parallel after T001 | Builder: pipeline mode, single builder
```

Phase 1 builds the seam: the generator, the committed artifact, the embedding package, and the guard that pins them. Phase 2 puts the command on it. Within Phase 1, T002 (embed + accessor) and T003 (drift guard) are `[P]` — both depend only on T001's artifact and touch different packages. Phase 2 is sequential: the command renders through the resource, the docs describe the command.

**Story labels** (from spec.md User Scenarios, in order):
- **US1** — the assembler gets the sayable types, placements, and dead shapes in one read, before building
- **US2** — every rendered shape carries its provenance, so contract and observation are never confused
- **US3** — the grammar is consultable on any machine with the CLI, before credentials, plugin or not

---

## Branching Guidance

**Pipeline mode**: `spec/077-agent-facing-grammar-reference/base` → `spec/077-agent-facing-grammar-reference/task-N`. Two PR-sized units are natural: Phase 1 as one PR (T001–T003 land together — the artifact without its guard is unpinned), Phase 2 as the second (T004–T006; T005's hook edit must ride with the command — see the task).

---

## Scenario Disposition

Fifteen scenarios exist in `agent-facing-grammar-reference.feature`. Twelve are implemented against by the tasks below; **three carry `@validation` and are held out for the validate stage — no task executes them**:

| Scenario | Disposition |
|---|---|
| An assembler reads the full grammar in one invocation | T005 |
| A change set offered for checking is refused as usage, not judged | T005 |
| Accepted-but-invalid stays distinct from accepted in the rendering | T005 |
| The full format presents the vocabulary, the nesting rule, and every fact | T004 |
| The compact format condenses each type and fact to one line | T004 |
| The contract vocabulary renders even with no live residue | T004 |
| No invocation judges a change set | `@validation` — held for validate |
| Every rendered shape carries its provenance token | T005 |
| A contract refresh that outruns the rendering fails the build | T003 |
| A retired fact leaves the rendering with the record | T005 (exercises T001's regeneration) |
| The rendered vocabulary matches the contract exactly | `@validation` — held for validate |
| Each rendered fact renders from the record alone | `@validation` — held for validate |
| The grammar renders with no credentials and no network | T005 |
| The write gate passes the grammar read ungated | T005 (the hook edit) |
| Repeated invocations render identical structured output | T005 |

**One interface surface deliberately carries no scenario**: the exit-1 corrupt-embed decode path (`interface-cli.md` § Error Communication). Reaching it requires a build the drift guard makes impossible — a scenario would have to fabricate a corrupt binary to exercise a state that cannot ship. It is covered instead by T002's unit test (the accessor returns an error rather than panicking or rendering empty). Recorded here so the absence reads as a decision, not an oversight.

---

## Phase 1: Grammar data foundation [Shared]

- [x] **T001** [Shared] Generator and the committed grammar artifact — wrapper pair derived from contract prose (self-checking against the enum), 12 derivation unit tests
  - **Scope**: A dev-time generator (its own small `main` package, invoked through a `go:generate` directive in `internal/grammar`) that derives the committed artifact from both sources mechanically, plus the committed artifact itself in `internal/grammar/`. Nothing hand-maintained: the change-type vocabulary via `internal/build.LoadSpecChangeTypes()`, the facts via `internal/build.ReadGrammarFactsRecord()` + `ParseGrammarFactsRecord()`.
  - **Acceptance criteria**:
    - The artifact is the `{generated, grammar}` envelope from interface-cli.md: `generated` carries a do-not-edit marker naming the regeneration step; `grammar` carries `change_types` and `facts` exactly per the accord's field tables.
    - `change_types` holds every enum member; nested-only entries (and only they) carry `wrappers: ["CreateRole", "UpdateRole"]`; every entry carries `provenance: "published-contract"`.
    - `facts` holds CSG-1 and CSG-2 with `id`, `title`, `shape`, `disposition`, `symptom`, `provenance: "empirical-observation"`; the record's Evidence and lineage fields are excluded.
    - Generation is byte-deterministic: `change_types` alphabetical by type, `facts` in the record's live-facts manifest order; a record with an empty manifest yields `facts: []` with the key present.
    - The wrapper-pair derivation decision is made and recorded in a code comment: derived from the contract's `ProposalChange` description prose, or — if prose-parsing proves brittle — a checked-in contract fact set-compare-guarded against the vendored spec (interface-cli.md Consistency Notes; plan ADR-2 discipline).
  - **Dependencies**: None
  - **Plan reference**: Phase 1; ADR-1, ADR-2, ADR-3
  - **Interface references**: interface-cli.md: "The rendered structure", "The embedded artifact"

- [x] **T002** [Shared] [P] The `internal/grammar` package — embed and typed accessor — 13 accessor unit tests incl. the corrupt-embed decode path
  - **Scope**: `//go:embed` of the committed artifact and a typed accessor that decodes once and returns the `grammar` payload (never the envelope) for the render layer and the command.
  - **Acceptance criteria**:
    - The accessor's types mirror the accord's field tables; the envelope's `generated` marker is not reachable through it.
    - A decode failure returns an error (T005 classifies it as the CLI-internal fault); it does not panic.
    - Unit tests cover the accessor against the committed artifact, including field completeness and ordering.
  - **Dependencies**: T001
  - **Plan reference**: Phase 1; ADR-1, ADR-3
  - **Interface references**: interface-cli.md: "The rendered structure"

- [x] **T003** [Shared] [P] Drift guard — regenerate-and-compare plus invariants — 1 scenario, 11 guard tests (one red case per divergence class)
  - **Scope**: An `internal/build` guard (helpers in production source, tests beside them, house convention) that regenerates the artifact in-memory via the same derivation functions and byte-compares against the committed file, plus the five named invariants: decodes, non-empty `change_types`, fact ids equal the record's live-facts manifest, a provenance token on every entry, the generated marker present.
  - **Acceptance criteria**:
    - Each divergence class has a red-case test: hand-edited artifact, record edited without regeneration, vendored-spec enum change without regeneration, missing marker, manifest/fact mismatch.
    - Every failure message names which half diverged and names the regeneration step as the remedy — and the remedy satisfies all sibling invariants at once (reachable-remedy discipline).
    - The guard runs under `go test ./...` with no new CI wiring.
  - **Dependencies**: T001
  - **Plan reference**: Phase 1; ADR-5; Risks R2, R3
  - **Scenario references**: agent-facing-grammar-reference.feature: "A contract refresh that outruns the rendering fails the build"

## Phase 2: The command and its rendering [US1, US2, US3]

- [x] **T004** [US1, US2] Grammar render resource and human-format templates — 3 scenarios, 10 render tests (golden full/compact + empty-residue variants)
  - **Scope**: A grammar resource in `internal/render` with `grammar.full.tmpl` and `grammar.compact.tmpl` in the embedded template set, rendering the accessor's structure.
  - **Acceptance criteria**:
    - `full` presents the vocabulary with each type's placement, the nesting rule stated once, every fact with title/shape/disposition/symptom, and visible provenance marking separating contract from observation.
    - `compact` presents every type with its placement class condensed, and each fact as one line carrying id, disposition, and title.
    - An empty `facts` list renders the explicit "no empirical residue is currently recorded" statement in both human formats — the section never silently disappears.
    - Templates carry no trailing newline (house convention); golden tests pin both formats, including the empty-residue variant.
    - **Test-first**: the three human-format scenarios listed for T004 in the Scenario Disposition table have their step definitions written and observed **red** before the templates that satisfy them, then green. The golden tests are likewise written before the template bodies they pin.
  - **Dependencies**: T002
  - **Plan reference**: Phase 2; ADR-4
  - **Interface references**: interface-cli.md: "The human formats (`full` / `compact`)"
  - **Scenario references**: agent-facing-grammar-reference.feature: "The full format presents the vocabulary, the nesting rule, and every fact", "The compact format condenses each type and fact to one line", "The contract vocabulary renders even with no live residue"

- [x] **T005** [US1, US2, US3] The `proposal grammar` command, its conduct, and the gate's recognized-read edit — 8 scenarios (12 of 12 non-@validation now green), 11 unit tests; accord boundary on "malformed credential file" recorded in LEARNINGS.md
  - **Scope**: The client-less cobra leaf under the `proposal` group — no positional arguments, no command-local flags, no seam, no token or base-url resolution — rendering the accessor's structure through the selected format (`full`/`compact` via T004; `json`/`yaml` serialize the structure directly; user templates apply over it). Bundled with **both** guardrail edits the new leaf requires, on two different anchors (plan § Integration Design): `expectedProposalSurface` in `internal/build/writesafetyguardrail.go` gains `"grammar"` (without it `TestWriteSafetyRegistryDriftGuard` fails the build — this is the CI forcing function), and `PROPOSAL_READS` in `plugin/hooks/glassfrog-write-gate.sh` gains `grammar` (without it the gate over-asks on a read at runtime — no build failure, so this one is on the builder and the scenario below). The gated registry (`gated-commands.txt`) is **not** touched: it lists writes only.
  - **Acceptance criteria**:
    - Succeeds (exit 0) with no credential file present, with a malformed one, and with no network — token resolution is never invoked.
    - Any positional argument fails as a usage error, exit 2, with cobra's usage text and no validity language; unknown flags likewise; exit codes 3–7 are unproducible (no request path exists).
    - `--output json` emits the `grammar` structure verbatim — top-level `change_types` and `facts`, `facts` present as `[]` when empty; `yaml` is the same document; repeated runs are byte-identical.
    - `--base-url` parses and is inert; help text states the command informs and never validates.
    - `TestWriteSafetyRegistryDriftGuard` is green with `"grammar"` in `expectedProposalSurface`, and the gate script classifies `proposal grammar` as a recognized read (passes ungated rather than asking). Both are required — the first is enforced by CI, the second only by the gate scenario.
    - The surface edit introduces no development-repository reference (CONSTITUTION.md XIII): adding one word to `PROPOSAL_READS` keeps `plugin/` self-contained, and the surface-walk guard stays green.
    - **Test-first**: the scenarios listed for T005 in the Scenario Disposition table have their step definitions written and observed **red** before the command that satisfies them, then green.
    - The `@wip` scenarios listed for T005 in the Scenario Disposition table pass; the three `@validation` scenarios remain held out.
  - **Dependencies**: T004
  - **Plan reference**: Phase 2; ADR-4; Integration Design ("Command registration"); Risk R4
  - **Interface references**: interface-cli.md: "The command", "Interactions", "Error Communication"
  - **Scenario references**: agent-facing-grammar-reference.feature: "An assembler reads the full grammar in one invocation", "A change set offered for checking is refused as usage, not judged", "Accepted-but-invalid stays distinct from accepted in the rendering", "The contract vocabulary renders even with no live residue", "Every rendered shape carries its provenance token", "A retired fact leaves the rendering with the record", "The grammar renders with no credentials and no network", "The write gate passes the grammar read ungated", "Repeated invocations render identical structured output"

- [ ] **T006** [Shared] Reference documentation
  - **Scope**: The command's page under `docs/reference/`, following the house reference-doc conventions.
  - **Acceptance criteria**:
    - Documents the command, its formats, the structured output's keys and token vocabularies, and the exit-code envelope (0/2/1, with codes 3–7 stated as unproducible).
    - Cross-cutting flag facts (`--output` chain, template behavior) state the shared conventions uniformly with the sibling pages, verified against source.
    - States the informs-never-validates boundary and the provenance marking.
  - **Dependencies**: T005
  - **Plan reference**: Phase 2; Cross-cutting Concerns ("Documentation")
  - **Interface references**: interface-cli.md: all sections
