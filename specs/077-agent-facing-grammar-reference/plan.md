# Plan: Agent-Facing Grammar Reference

**Feature**: 077-agent-facing-grammar-reference
**Role**: Shaper
**Inputs**: spec.md, PROJECT.md, FEATURE-MODEL.md (amended grammar-reference block), `.score/memory/DECISIONS.md` (167 entries), `.score/memory/LEARNINGS.md`, `.score/memory/DEPRECATION.md` (9 entries), `internal/build/grammarfacts.go` (072's parser + spec-derivation machinery), `internal/render/render.go`, `internal/cli` command conventions, the 072 record at `plugin/skills/proposal-drafting/references/change-set-grammar-facts.md`

---

## System Architecture

The capability is a pipeline with a hard seam in the middle: everything left of the seam runs **in the repository at development time**, everything right of it runs **in the shipped binary with no repository access**. The seam is a generated, committed artifact.

```
  repo sources (dev time)              │  shipped binary (run time)
                                       │
  spec/glassfrog-api-v5.yaml ─┐        │
    LoadSpecChangeTypes()     ├─ generator ─→ committed grammar artifact
  plugin/.../change-set-      │  (go:generate)   (internal/grammar/, JSON,
    grammar-facts.md ─────────┘        │           marked generated)
    ParseGrammarFactsRecord()          │      │
                                       │      ▼ //go:embed
  internal/build drift guard           │  grammar package (typed accessor)
  (regenerate in-memory,               │      │
   byte-compare, invariants)           │      ▼
                                       │  grammar command (internal/cli,
                                       │   client-less: no seam, no token,
                                       │   no request)
                                       │      │
                                       │      ▼
                                       │  internal/render (new grammar
                                       │   resource; full/compact templates;
                                       │   json/yaml serialize the structure)
```

**Components:**

- **Generator** — a small dev-time `main` package invoked through a `go:generate` directive in the grammar package. It derives the artifact from both sources *mechanically*, reusing 072's existing functions: the change-type vocabulary and nested-only membership via `LoadSpecChangeTypes()` (parses the vendored contract), and the empirical facts via `ReadGrammarFactsRecord()` + `ParseGrammarFactsRecord()` (the record's one parser). It hand-maintains nothing.
- **Committed grammar artifact** — one JSON file in the grammar package: the vocabulary entries (type + placement class, provenance `published contract`) and the fact entries (id, title, shape, disposition, symptom, provenance `empirical observation`) — the field set the interface accord pins, with the wrong shape named inside the symptom rather than carried as its own field, exactly as the record encodes it. Header-marked as generated; never hand-edited.
- **Grammar package** (`internal/grammar`) — `//go:embed`s the artifact and exposes a typed accessor. The binary carries no YAML or markdown parsing for this feature; the record's format is a generation-time concern only.
- **Grammar command** (`internal/cli`) — a cobra read command that is *client-less*: it builds no API client, resolves no token and no base URL, and touches only the output-format resolution chain. It hands the embedded structure to render. Cobra argument validation rejects positionals, so a change set offered for judgment fails as a usage error before any code of ours runs.
- **Render integration** (`internal/render`) — a new grammar resource with `full`/`compact` templates in the embedded template set; `json`/`yaml` serialize the embedded structure directly. User template files (`-o <file>`) apply over the same structure, as they do for every read.
- **Drift guard** (`internal/build`) — regenerates the artifact in-memory using the same derivation functions and byte-compares against the committed file, plus structural invariants. Runs in `go test ./...`, so the existing merge gate covers it with no new wiring (the 024/029 pattern, same as 072/075/076).

**Data flow at run time**: embed → accessor → render. No network, no credentials, no filesystem beyond what cobra and output resolution already do. The spec's Conduct accord falls out of the architecture rather than being enforced by checks.

---

## Architecture Decisions

### ADR-1: The binary carries the grammar as a generated, committed, embedded artifact; the record stays at its 072 home

**Context**: `go:embed` cannot reach outside a package's directory tree, and the record lives under `plugin/`. The model discussion sketched relocating the record into the CLI tree with a plugin-side symlink back (072 ADR-1's reserved consumer pattern) — but 076's developer-confirmed decision bans it: pointers flow repository → surface *only*, and a symlink at the plugin path targeting `internal/...` is a surface → repository pointer that dangles on a machine with only the plugin and trips 076's walk-derived guard.

**Options considered**:
1. **Relocate the record + plugin symlink** — one physical file. Violates 076's strict ban; also reworks 072's path constant, guards, and ownership rule.
2. **Relocate the record + guarded byte-copy back into the plugin** — avoids the symlink but leaves two hand-visible full copies and still reworks 072.
3. **Generated, committed, embedded artifact + regeneration drift guard** — the record stays put; a generator derives a structured artifact from both sources; the guard makes drift impossible. Zero rework of 072, zero 076 exposure.
4. **Embed the whole vendored contract** — 290KB in the binary for one enum and one rule; disproportionate, and the contract file is repo-only by design (integration boundary: consumers never need it).

**Decision**: Option 3 — generated artifact (resolve-phase, developer-confirmed). The record remains the only file a human edits; the artifact is machine-produced, header-marked generated, and any divergence — hand-edit, record edit without regeneration, contract refresh — turns the merge-gating run red.

**Single-source reading, recorded deliberately**: the spec's Sync accord ("no second copy of a fact's text exists **to drift** from the record") and the held-out validation scenario ("no fact's text is **hand-maintained** outside the record") are satisfied on the drift-impossibility property: the artifact does contain fact text, but it is a generated projection that the guard pins byte-exact to the record — there is exactly one editable source, and the second life a hand-copy would live (diverging quietly) cannot occur. This reading lives here, in the plan, the way 072's plan recorded its deliberate narrowing; a validator reading the spec alone should be pointed here.

*Wording note (validation round 1, F-3)*: this ADR was written against the scenario's original phrasing, "no fact's text lives a second life outside it", which reads as a ban on the second copy itself rather than on hand-maintenance. The scenarios stage had already concretized it correctly ("renders from the record **through the generated artifact**"; "hand-maintained"); spec.md was aligned to match after round 1, and the quote above follows the amended text. The decision this ADR records is unchanged.

**Consequences**: the fact-retirement flow (072's three-part act) gains a fourth mechanical step — regenerate the artifact — enforced by the guard rather than remembered by the editor. The record itself gains **no** new repository references (076 ban); the remedy lives in the guard's failure message and repo-side docs.

### ADR-2: Both artifact halves are derived mechanically — nothing hand-maintained

**Context**: the drift-guard discipline in process memory is explicit: a guard derives both sides from source; hard-coding a guarded value creates a second source of truth. 072 already ships the two derivation functions.

**Options considered**:
1. **Hand-written Go table for the vocabulary, guarded by set-compare** — the 067-style checked-in contract fact. Works, but creates a second hand-edit site for content that is fully derivable, and every contract refresh needs a hand edit.
2. **Mechanical derivation via `LoadSpecChangeTypes()` and `ParseGrammarFactsRecord()`** — the generator and the guard call the same functions, so the parsers cannot diverge from the guard's reading of the same files (the paired-parsers discipline).

**Decision**: Option 2. `internal/build`'s derivation functions gain the generator as a second consumer; their contract stays owned by 072's guard.

**Consequences**: a vendored-spec refresh that changes the enum or the nested-only description flows into the artifact by regeneration alone. The dependency runs generator → `internal/build` (dev-time only); the shipped binary never imports the derivation code.

### ADR-3: The artifact is structured at generation time — the binary parses no record format

**Context**: the command could embed the record verbatim and parse it at run time, or embed a structure parsed at generation time.

**Options considered**:
1. **Embed the record markdown, parse at run time** — the rendering provably consumes the record's bytes, but it pulls the record parser (and its YAML dependency) into the shipped binary, and a record-format quirk becomes a runtime failure on an operator's machine.
2. **Parse at generation, embed structured JSON** — the binary decodes one JSON document; a record-format problem fails generation or the guard, in the repository, before merge.

**Decision**: Option 2. Failures move to where the repository and its guards are; the runtime surface is a decode of a build-guaranteed document.

**Consequences**: the artifact's structure becomes a real contract between generator, guard, and command — the interface stage pins its field vocabulary. The record's per-fact **Evidence** and lineage fields are maintenance metadata; whether they render is an interface decision, not lost by this ADR (the generator can carry them).

### ADR-4: Rendering goes through the existing Resource × Format machinery

**Context**: the Conduct accord requires the reference to render "the way other reads do." The house machinery is `Render(resource, format, data)` with embedded per-resource `full`/`compact` templates; `json`/`yaml` for API reads serialize the response.

**Options considered**:
1. **Bespoke printing in the command** — simplest, but agents would face one command whose output conventions match nothing else, and user templates (`-o <file>`) would need special handling.
2. **New grammar resource in `internal/render`** — a template pair in the embedded set; `json`/`yaml` serialize the embedded structure directly (there is no API response — the structure *is* the source data, in the pass-through spirit).

**Decision**: Option 2. The grammar becomes one more resource to the render layer; format selection, user templates, and agent parsing behave identically to every read.

**Consequences**: the `json` output shape is the artifact structure — a contract agents will code against, pinned at interface. Template files follow the house convention (no trailing newline; golden tests).

### ADR-5: The guard is regenerate-and-compare plus structural invariants, in `internal/build`

**Context**: 072's guard family (and 075's, 076's) runs as `internal/build` tests under `go test ./...`, which the merge gate already executes. The guard's failure message is a specification — its remedy must be reachable.

**Options considered**:
1. **Byte-equality only** — catches every divergence but reports it opaquely.
2. **Byte-equality plus invariants** — equality against the in-memory regeneration, and independent structural checks: the artifact parses, the vocabulary is non-empty, the fact ids equal the record's live-facts manifest, every entry carries a provenance marker, and the generated header is present. Invariants make the failure nameable (which half diverged, which fact is missing) instead of "bytes differ."

**Decision**: Option 2. Each failure names the offending half and the remedy — run the generator — and the remedy cannot collide with a sibling invariant (the regeneration satisfies all of them by construction).

**Consequences**: a record edit, a contract refresh, or a hand-edit of the artifact each turn the run red with a message naming the regeneration step. The guard lives beside 072's, sharing its derivation functions but pinning a different file.

---

## Integration Design

- **Vendored contract and grammar record → generator**: generation-time only. The seam guarantees the spec's availability accords: no consumer of the binary ever needs `spec/glassfrog-api-v5.yaml` or the plugin file. Both stay exactly where they are; 072's `GrammarFactsPath` constant and guards are untouched.
- **Fact retirement choreography** (extends 072's recorded three-part act): (1) delete the fact's section from the record, (2) drop its id from the live-facts manifest, (3) record the `/score:deprecate` entry — and now (4) regenerate the artifact. Steps 1–3 unaccompanied by 4 fail this feature's guard; step 4 alone (hand-edit without record change) fails it too. The "no live residue" edge scenario falls out: a record with an empty manifest generates an artifact with an empty facts list, and the rendering states that explicitly. Note: 072's guard deletes the record file when the *last* fact retires — at that point regeneration fails loudly (source missing), which is correct: serving a grammar with no record is a decision for that future change, not a silent fallback.
- **Command registration**: one new read leaf in `internal/cli` under the proposal group — `glassfrog proposal grammar` (developer-chosen at interface resolve; the grammar is about proposals). The plan-level constraint holds: it is its **own leaf**, never a flag — a flag on a gated write would trip the write-gate hook's positional matching, and a flag on the bare `proposal` group would invert the CLI's structure (groups have no action; flags modify actions, they don't select them). Consequence of the placement, verified against the landed guardrail: adding a `proposal` leaf requires **two** coordinated edits in Phase 2, on two different anchors.
  1. `expectedProposalSurface` in `internal/build/writesafetyguardrail.go` gains `"grammar"`. This is what turns CI red: `TestWriteSafetyRegistryDriftGuard` compares the CLI's live proposal surface against that checked-in expectation and fails on any added leaf, *read or write* ("the CLI's proposal command grew or renamed subcommand … reclassify it (read or write?)"). The forcing function is this Go-side expectation, not the shell script.
  2. `PROPOSAL_READS` in `plugin/hooks/glassfrog-write-gate.sh` gains `grammar`. This is runtime conduct, not a build gate: an unrecognized `proposal` subcommand fail-closes to *ask*, so omitting this edit produces a spurious confirmation prompt on every gated machine — safe friction, silently wrong UX, and no red build.
  The gated registry (`plugin/hooks/gated-commands.txt`) is **not** touched — it lists writes only, and this is a read.
- **Downstream consumers (not built here)**: Pre-Assembly Grammar Consultation points the drafting path at the command; Typed Change Builders scope to the shapes the same artifact verifies. Both consume the command/artifact as-is; nothing in this plan reserves structure for them beyond ADR-3's real contract.

---

## Cross-cutting Concerns

**Failure handling**: the runtime failure surface is deliberately near-empty. Usage errors (unexpected positional, unknown flag) are cobra's, classified through the existing dispatch as usage errors. The embedded artifact failing to decode is build-guaranteed-impossible; if it ever happens (corrupt build), it classifies as the CLI-internal runtime fault through the existing Outcome machinery — never as an API or network category, since no request exists. No plan-gate, no 403, no pagination, no stale-write paths apply.

**Configuration**: only the output-format resolution chain participates (flag → env → `.glassfrogrc` → default). Token and base URL are never resolved — not "resolved and unused," never resolved, so a malformed credential file cannot block the command.

**Testing strategy**: unit tests on the accessor and generator derivation; golden render tests per format (house pattern — templates are whitespace-sensitive, no trailing newline); guard tests in `internal/build` (equality + each invariant, including the hand-edit and stale-artifact cases); BDD wiring comes from the scenarios stage. The drift-guard helpers live in production source, not `_test.go` files, per house convention.

**Documentation**: the command gets its reference page under `docs/reference/` like every command; the artifact's generated header names the generator and warns against hand edits.

---

## Implementation Strategy

**Phase 1 — Grammar data foundation.** The generator (derivation via `internal/build`'s functions), the committed artifact, the `internal/grammar` package with embed + typed accessor, and the drift guard with its invariants. Exit criterion: the artifact regenerates byte-identically in CI, and every guard invariant has a red-case test.

**Phase 2 — The command and its rendering.** The client-less cobra command (`proposal grammar`), the render resource with `full`/`compact` templates, `json`/`yaml` serialization, usage-error conduct, credential-free conduct, the two guardrail edits (`expectedProposalSurface` and `PROPOSAL_READS`, per Integration Design), and the reference documentation. Depends on Phase 1 (renders the accessor's structure). Exit criterion: the spec's driving scenarios pass against the shipped command.

Two phases, each a PR-sized unit; no parallelism worth engineering between them.

---

## Risks

- **R1 — The single-source reading is challenged at validation.** *(Materialized in round 1 as F-3, resolved as predicted.)* The held-out scenario originally said "no fact's text lives a second life outside it"; the generated artifact contains fact text. Likelihood moderate, impact low (a wording dispute, not a defect). Mitigation: ADR-1 records the drift-impossibility reading explicitly, the artifact is header-marked generated, and the guard's hand-edit rejection is itself tested. If validation still reads the scenario literally, the resolution is a spec wording touch-up, not an architecture change. **Outcome**: validation traced every rendered field as byte-identical to the record and confirmed the guard rejects a hand edit, then flagged the literal wording. spec.md was amended to the scenarios stage's already-correct phrasing ("hand-maintained"); no architecture changed.
- **R2 — The nested-only derivation parses contract prose.** `LoadSpecChangeTypes()` extracts the nested-only list from the `ProposalChange` description text; a refresh that rewords it breaks derivation. Likelihood low, impact low — generation and 072's guard fail together, at the refresh PR, which is the designed signal (the S7 lesson: the vendor's own change signals are unusable). Mitigation: shared derivation function means one fix; the failure names the derivation site.
- **R3 — A record edit merges without regeneration.** Impossible past CI (the guard runs in the merge gate); the residual risk is developer friction mid-change. Mitigation: the guard's failure message names the exact regeneration command; the retirement choreography is recorded here and in the guard's comments — deliberately not in the record itself, which must stay free of repository references (076).
- **R4 — The chosen name sits inside the gate's fail-close scope.** `proposal grammar` is an unrecognized `proposal` subcommand to the shipped gate script, which asks rather than waves through — so shipping the command without the `PROPOSAL_READS` edit produces a spurious confirmation prompt on every gated machine. Likelihood of shipping that way: low but **not** zero, and the reason is worth stating precisely. The CI forcing function is `expectedProposalSurface` (Go side): it fails the build on the added leaf and its message tells the builder to reclassify it. That guarantees the leaf is *classified*, but nothing mechanically verifies the shell script was also updated — a builder could satisfy CI by adding `"grammar"` to the expectation alone and ship the over-gating. Mitigation: both edits are named in the same task with the asymmetry stated (T005), and the feature file's gate scenario asserts the ungated pass, so the omission is caught by an acceptance scenario rather than by CI alone.

---

## What This Plan Does Not Cover

- **The command's name, flags, and output field contract** — the interface stage pins the CLI touchpoint and the artifact/output structure (ADR-3/ADR-4 make it a real contract).
- **Executable scenarios** — the scenarios stage turns the spec's driving scenarios into .feature files.
- **Task decomposition** — the tasks stage cuts the two phases into PR-sized units.
- **Consultation wiring and builders** — Pre-Assembly Grammar Consultation (#77) and Typed Change Builders (#83) consume this feature; nothing here builds or reserves their behavior.
- **076's conformance sweep** — the record's existing prose (its pathless "the guard" mentions) is 076's to sweep; this feature adds no new violations and does not fix old ones.
- **The template-fields sibling** — the CLI-served field listing candidate recorded in the issue tree shares this plan's mechanism family but is its own problem, deliberately untouched.
