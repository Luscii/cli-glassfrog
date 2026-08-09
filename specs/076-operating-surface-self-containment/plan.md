# Plan: Operating-Surface Self-Containment

**Feature**: 076-operating-surface-self-containment
**Role**: Shaper
**Inputs**: spec.md, PROJECT.md, DECISIONS.md (filtered precedent), LEARNINGS.md (filtered patterns), DEPRECATION.md (7 entries)

---

## System Architecture

Four parts land together, in two phases. Nothing here adds capability — the surface's behavior, fences, and gate posture are untouched (spec non-behavior 1).

1. **The swept operating surface** (`plugin/`, 21 of 26 files). Wording-only edits that remove every development-repository reference: spec-number ids become in-plugin component names (or are dropped where the prose name already sits beside them), the six single-source registry headers lose their `internal/build/*_guard_test.go` pointers and plan-ADR citations, and the two drafting reference files lose their spec-number provenance lines. The replacement vocabulary is fixed in the Sweep Design section so the sweep and the re-derived assertions cannot disagree.

2. **Re-derived repo-side assertions**. The BDD suites in `internal/build` and the six feature files under `features/unequipped-agent-operators/` currently assert that surface text *contains a spec number* (e.g. `containsFold(w.combined(), "065")`). Each assertion is re-derived from its property — "the artifact defers by a name resolvable where the operator stands" — to pin the in-plugin component name instead. Repo-side artifacts keep their own spec-number *comments*; only assertions against surface content change.

3. **The self-containment guard** (new, `internal/build`). Production source (`surfaceselfcontainment.go`) holds the walker and the deny lexicon; a guard test (`surface_self_containment_guard_test.go`) walks every file under `plugin/` — the checked set is derived by walking, never enumerated — and fails on any lexicon match, naming file, line, offending text, and the remedy. A companion BDD suite binds this spec's feature file, exercising detection against temp-dir fixture surfaces (never by mutating the real plugin). Naming avoids the existing `selfcontainment_test.go` (021's binary-linkage checks — a different self-containment).

4. **The constitutional principle**. CONSTITUTION.md gains Principle XIII (Self-Contained Operating Surface) in house form — statement, rationale, detection — via the Governance section's amendment process.

Control flow at merge time: the guard runs inside `go test ./...`, which both `ci.yml` (024) and `main-verify.yml` (029) already execute — so "the merge-gating verification run" needs no new CI wiring.

---

## Architecture Decisions

### ADR-1: The guard is an `internal/build` test, not a CI step or hook

**Context**: The spec requires a check in the merge-gating verification run that walks `plugin/` and fails on violations. The repo has three enforcement idioms: `internal/build` guard tests (021's config guard, every path spec's drift guard), standalone scripts under `scripts/`, and pre-commit hooks.

**Options considered**:
1. **`internal/build` guard test** — runs in `go test ./...`, so 024/029 gate merges with zero new wiring; sits beside the eight sibling artifact guards. Downside: enforcement only at test time, not at edit time.
2. **Dedicated CI step / script** — visible as its own check. Downside: new workflow wiring, a second enforcement idiom for the same artifact family, and it wouldn't run in a plain local `go test ./...`.

**Decision**: Option 1 — `internal/build` guard test. Silent conformance with the established guard pattern (021 ADR onwards; every operating-surface artifact guard lives there). Walker and lexicon live in production source, not the `_test.go` (LEARNINGS: drift-guard helpers belong in production source), so the BDD suite and the guard share one implementation.

**Consequences**: Enforcement is exactly as strong as the existing guards — a violation turns `go test ./...` red locally and in both merge gates. Anyone adding a plugin file learns the rule at the first test run.

### ADR-2: A checked-in two-family deny lexicon; the file set is walked, the lexicon is the contract

**Context**: "Development-repository reference" cannot be derived from repo state — the guard needs a definition. The drift-guard rule (DECISIONS/LEARNINGS: a guard must not hard-code the SoT) cuts here as: the *scanned set* must be derived (walk `plugin/`), while the *definition of forbidden* is a legitimate checked-in contract-fact, like 063's gated-commands registry.

**Options considered**:
1. **Single flat regex list** — simple. Downside: no structure for the two distinct failure modes (a resolvable pointer vs. a strict-ban prose mention), and per-pattern rationale gets lost.
2. **Two pattern families with per-entry property comments** — (a) *resolvable references*: spec-number ids (`\b0\d{2}\b`), repo paths (`specs/`, `internal/`, `features/`, `docs/`, `scripts/`, `.score/`, `spec/`), Go source tokens (`.go` filenames, `_test.go`), pipeline artifacts (`plan.md`, `tasks.md`, `spec.md`, `interface-*.md`, `ADR-<n>`), portfolio documents (FEATURE-MODEL, ROADMAP, BACKLOG, ISSUE-TREE, VISION, LEARNINGS, DECISIONS, STATUS); (b) *repo-machinery phrases* enforcing the strict ban: "drift guard", "drift tripwire", "source repository", "development repository", "plan ADR", and kin. Each entry carries a comment stating the property it protects and a concrete example, so the next editor re-derives instead of copying (the change-detector-property rule).
3. **Derive the lexicon from repo contents** (e.g. forbid any token that matches an existing repo path) — maximally automatic. Downside: the lexicon would change as the repo changes, making guard outcomes depend on unrelated commits, and prose mentions ("the source repository") are underivable anyway.

**Decision**: Option 2 — two families, property-commented, in `surfaceselfcontainment.go`. Word-boundary and path-shape anchoring keeps the known-safe tokens safe: `prp_0123` (no word boundary before `0`), version strings (`0.34.1` never forms `0\d{2}`), exit codes 0–7, "the v5 spec"/"specification" (only slash-form `specs/`/`spec/` and id-form numbers are matched), and `\bCI\b` never matches inside "CLI".

**Consequences**: The strict ban is enforced to the lexicon's reach, not to the reach of natural language — a novel prose mention of repo machinery could evade it (Risk 1). Extending the lexicon is a deliberate contract edit with a property comment, reviewed like a gated-commands change. A light positive check rides along: any `plugin/…`-shaped path mentioned in a surface file must exist, so in-surface references stay resolvable.

### ADR-3: Number-pinned assertions are re-derived to name-pins, never allowlisted or deleted

**Context**: The existing BDD suites assert spec numbers appear in surface text; the sweep removes those numbers, turning the suites red. Three ways out.

**Options considered**:
1. **Guard-side allowlist** (exempt the numbers the suites need) — keeps suites green without edits. Downside: the leak the feature exists to remove stays in the shipped surface. Rejected outright.
2. **Delete the assertions** — quick. Downside: destroys the property they protect (the artifact defers to the right path by an identifiable name) — spec non-behavior 6 forbids exactly this.
3. **Re-derive to name-pins** — `containsFold(text, "065")` becomes an assertion on "constraint-discovery" (the in-plugin skill/agent name from the Sweep Design mapping); feature-file step text updates in place to match.

**Decision**: Option 3. The property survives; the surface stops carrying repo ids. Feature scenario text diverges from the merged specs' User Scenario prose it was copied from — an announced divergence, recorded in the spec (non-behavior 2) and here: already-merged specs (the leaked 062–073 records and every other merged sibling) stay untouched, git history is the changelog.

**Consequences**: ~10 `internal/build` test files and 6 feature files change in the same phase as the sweep, keeping every suite green at each commit. Whitespace-collapsed matching (the operator-path BDD convention) is preserved — only the needle changes.

### ADR-4: The constitution amendment adds Principle XIII and establishes a version marker

**Context**: The spec requires the principle in house form via the Governance section's process ("documented justification and a version bump") — but CONSTITUTION.md currently carries no version marker to bump.

**Options considered**:
1. **Principle only, justification in the commit** — minimal. Downside: the governance clause's "version bump" stays unsatisfiable, for this and every future amendment.
2. **Principle + establish a version line** — add Principle XIII and introduce an explicit version marker (initial value acknowledging the pre-existing twelve principles, bumped by this amendment), with the justification recorded beside it.

**Decision**: Option 2. The amendment satisfies its own governance clause and makes the next amendment's bump mechanical. The principle's *Detection* paragraph names the observable: any file under the operating surface referencing the development repository fails the repository's verification run.

**Consequences**: CONSTITUTION.md becomes versioned; the spec's validation scenario ("documented justification and version bump") is satisfiable as written.

---

## Sweep Design

The single mapping both the sweep and the re-derived assertions use. Left column: the leaked id; right: the canonical in-plugin name (each resolves to a shipped file).

| Leaked id | In-plugin name (resolution) |
|---|---|
| 062 | the **orientation** skill (`skills/orientation/`) |
| 063 | the **write-safety gate** (`hooks/glassfrog-write-gate.sh` + `hooks/gated-commands.txt`) |
| 064 | the **governance-navigation** path (`skills/governance-navigation/`, agent `governance-navigator`) |
| 065 | the **constraint-discovery** path (`skills/constraint-discovery/`, agent `constraint-navigator`) |
| 066 | the **tension-processing** path (`skills/tension-processing/`, agent `tension-processor`) |
| 067 | the **proposal-drafting** path (`skills/proposal-drafting/`, agent `proposal-drafter`) |
| 068 | the **proposal-circulation** path (`skills/proposal-circulation/`, agent `proposal-circulator`) |
| 069 | the **proposal-impact-review** path (`skills/proposal-impact-review/`, agent `proposal-impact-reviewer`) — includes every "response side (069)" mention |
| 072 | the **change-set grammar facts** record (`skills/proposal-drafting/references/change-set-grammar-facts.md`) |
| 073 | the **circle-routing rule** record (`skills/proposal-drafting/references/circle-routing-rule.md`) |

Sweep shapes, by reference class:

- **Number beside its prose name** (the majority): drop the parenthetical — "the **Constraint Discovery Path** (065)" → "the **Constraint Discovery Path**". Word-boundary matching when sweeping (the symbol-rename rule), and reword any old-vs-new contrast sentences the drop strands.
- **Number standing alone** ("063 gates those writes regardless"): substitute the mapped name.
- **Registry headers** (`hooks/gated-commands.txt`, five `agents/*-commands.txt`): keep the in-plugin consumer list and the editing rule; delete the `internal/build/...` pointer lines, the "plan ADR-n" citations, and the "drift guard/tripwire turns CI red" consequences. The danger of hand-edits is restated through the surface's own consequence — the gate fail-closes an unrecognized command; an edited registry and the gate can disagree about what is gated. "Mirroring 063's gated-commands.txt" becomes "mirroring the write-safety gate's gated-commands.txt" (in-plugin path, kept).
- **Hook script** (`glassfrog-write-gate.sh`): drop the three "plan ADR" citations; the adjacent one-sentence rationales already carry the content.
- **Reference-file provenance lines** (072/073 records): remove the spec-number citations. The records' own `internal/build` guards pin their structure — those guard expectations are adjusted in the same commit (they are repo-side and may keep their own spec-number comments).

---

## Guard Design

- **Walk**: every regular file under `plugin/`, all extensions, comments included. Zero files or a missing directory is a loud failure (spec: no vacuous pass).
- **Match**: per line, against both lexicon families. Report every violation in a run (not first-only), each as `file:line: offending text — remedy`, the remedy being: *replace with the in-plugin component name (see the surface's own artifacts), or remove the reference*. The message is a specification — the remedy must be reachable, and it is: every current violation has a mapped name or is deletable.
- **Positive check**: tokens shaped like `plugin/<path>` in surface files must resolve to an existing file, keeping in-surface references real.
- **Testing**: the guard's detection behavior is proven against fixture surfaces in `t.TempDir()` (a violating file, an empty surface, a clean surface, each known-safe token class); the live pass over the real `plugin/` is the drift tripwire. BDD steps for the spec's scenarios drive the same production functions against fixtures — the error/edge scenarios never mutate the real plugin.

---

## Cross-cutting Concerns

**Testing strategy**: three layers — fixture unit tests for the lexicon (each family, each known-safe token), the live guard over `plugin/`, and the godog suite binding this spec's feature file (new file under `features/unequipped-agent-operators/`, own suite per the one-suite-per-feature convention). Content assertions use whitespace-collapsed copies per the operator-path BDD convention. After the sweep phase, the full `go test ./...` plus `gofmt -l .` and `golangci-lint run` gate the push (the gofmt-is-not-in-go-test lesson).

**Scenario execution is test-first within each phase, never deferred to a later one** (CONSTITUTION IV/VII). The feature file's behavioral scenarios split by what they read: the *surface-content* scenario reads the real swept surface and is therefore Phase 1's — its step definitions are written red against the unswept artifacts and go green when the sweep lands. The *detection* scenarios read fixture surfaces in `t.TempDir()`, depend on nothing the sweep does, and belong with the guard implementation in Phase 2. Neither phase lands step definitions for behavior implemented in the other, and no task is test-only: each pairs its own implementation with the tests that prove it, matching the sibling records' shape where step definitions land inside the implementing task.

**Error handling**: the guard is compile-time-deterministic — no I/O beyond the walk; any walk error is a test failure, never a skip (a skipped guard is CI theater).

**Configuration**: nothing configurable. The lexicon is deliberately hard-coded contract; the walk root is the repo-relative `plugin/` located via the existing `internal/build` repo-root helper the sibling guards use.

**Observability**: none needed beyond test output — the failure message carries file, line, text, remedy.

---

## Implementation Strategy

**Phase 1 — Conformance sweep + assertion re-derivation** (one PR-sized unit, possibly split by artifact family). Apply the Sweep Design mapping across the 21 plugin files; in the same commits, re-derive the number-pinned assertions in the `internal/build` suites and the six feature files, and adjust the record guards' pinned expectations for the two drafting reference records. The phase also opens this spec's godog suite and binds its surface-content scenario — written red against the unswept artifacts, green when the sweep lands — so the phase carries its own executable acceptance evidence rather than resting on a manual check. Every suite green at every commit; no guard exists yet, so ordering inside the phase is otherwise free.

**Phase 2 — Guard + constitution** (depends on Phase 1 only for the live tripwire; the fixture-driven detection scenarios depend on nothing the sweep does). Add `surfaceselfcontainment.go` + guard test + the detection scenarios' step definitions in the suite Phase 1 opened; amend CONSTITUTION.md (Principle XIII + version marker). TDD inside each task: fixture tests and step definitions first (red on a seeded violation), live tripwire last (green because Phase 1 landed). The guard implementation and the scenarios that prove it land together — never as a code-only task followed by a test-only one.

---

## Risks

1. **The strict ban outreaches any lexicon** — a future prose mention of repo machinery phrased novelly ("the checks upstream") slips through. Likelihood: medium over time. Impact: low (a wording leak, not a capability change). Mitigation: the two-family lexicon covers every observed violation class plus near variants; Principle XIII makes it a review obligation; the spec's validation scenario reads the surface end-to-end at validate time.
2. **False positives against future legitimate surface content** — e.g. a skill someday legitimately needing a `0NN`-shaped token. Likelihood: low. Impact: low-medium (blocked PR until lexicon edit). Mitigation: property comments on every entry make the intentional-relaxation path obvious and reviewable.
3. **Sweep collides with content-pinning guards** — the 072/073 record guards and the operating-surface packaging guard pin surface text that the sweep edits; the paired-parser normalization rule applies if any needle contains collapsed whitespace. Likelihood: high (known). Impact: low if handled in-phase. Mitigation: Phase 1 explicitly includes those guard-expectation updates; run the full suite per commit.
4. **Un-swept stragglers** — with ~150 references across 21 files, a stray `(0NN)` survives Phase 1 and turns the new guard red in Phase 2. Likelihood: medium. Impact: trivial to fix (the guard names it). Mitigation: run the Phase 2 lexicon as a local grep at the end of Phase 1 — the guard's patterns double as the sweep's completeness check.

---

## What This Plan Does Not Cover

- **Protocol-level contracts** — the guard's exact failure-message grammar, the lexicon's full pattern table, and the registry headers' post-sweep wording contract are the interface skill's concern (this feature has a specification boundary: it constrains a declarative artifact set).
- **Executable scenarios** — the scenarios skill turns the spec's driving scenarios into the feature file; the fixture-vs-live split above is guidance for it, not a substitute.
- **Task decomposition** — the tasks skill splits the two phases into PR-sized units with acceptance criteria.
- **Deliberately deferred** — nothing. The spec's one deferred technical point (check placement and token classification) is resolved here by ADR-1/ADR-2.
