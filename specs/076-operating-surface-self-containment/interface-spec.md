# Interface Accord: Operating-Surface Self-Containment — Specification

**Feature**: 076-operating-surface-self-containment
**Role**: Crafter
**Touchpoint**: Specification
**Plan reference**: the specification boundary — the constrained declarative artifact set (`plugin/`), the guard that enforces it (plan ADR-1/ADR-2), the registry-header wording (Sweep Design), and the constitutional principle (ADR-4)

---

## Surface

### 1. Reference rules for surface files

The normative contract every file under `plugin/` satisfies — the artifact set is the interface; these rules are its shape.

**Permitted reference classes**:

| Class | Form | Constraint |
|---|---|---|
| In-surface component | Its in-plugin name ("the constraint-discovery path", "the write-safety gate") or a `plugin/`-relative path (`plugin/hooks/gated-commands.txt`) | A `plugin/…`-shaped path must resolve to an existing file (checked; see Error Communication row 3). Names must be the component's own — the plan's Sweep Design table is the id→name mapping. |
| The `glassfrog` CLI | Command and flag names as the CLI exposes them | Unrestricted — the surface exists to drive it. |
| The operating world | The GlassFrog API and its published specification (as concepts, e.g. "the v5 spec"), Holacracy terms, operator-machine paths needed for setup (e.g. credential file locations) | Unrestricted — self-containment is repository-independence, not world-independence. |

**Forbidden: the deny lexicon.** Two families. Matching is per line, over every regular file walked under `plugin/` (all extensions, comment lines included). Each entry ships with a property comment and a concrete example in the guard source; the table below is that contract.

**Family A — resolvable references** (a pointer an operator without the repository cannot follow):

| Pattern | Case | Property it protects | Example violation |
|---|---|---|---|
| `\b0\d{2}\b` | — | No development-spec catalog exists where the operator stands; spec numbers are repo ids, not surface names | `(067)`, `063's gated set` |
| `\bspecs?/` | insensitive | Repo spec-directory and vendored-spec paths don't ship | `specs/067-…`, `spec/glassfrog-api-v5.yaml` |
| `\b(internal\|features\|docs\|scripts)/` | insensitive | Repo source/test/docs trees don't ship | `internal/build/…_guard_test.go` |
| `\b\.score/` | insensitive | Pipeline memory doesn't ship | `.score/memory/LEARNINGS.md` |
| `\b[\w-]+\.go\b` (includes `_test.go`) | insensitive | Go source is repo implementation, invisible to the operator | `write_safety_guardrail_guard_test.go` |
| `\b(plan\|tasks\|spec)\.md\b`, `\binterface-[\w-]+\.md\b` | insensitive | Pipeline artifacts don't ship | `plan.md ADR-4` |
| `\bADR-\d+\b` | sensitive | Design history lives in the repo, not the surface | `(plan ADR-5)` |
| `FEATURE-MODEL`, `ROADMAP`, `BACKLOG`, `ISSUE-TREE`, `LEARNINGS`, `DECISIONS`, `DEPRECATION`, `PROJECT.md`, `VISION.md`, `CONSTITUTION.md`, `STATUS.md` | **sensitive** (exact tokens) | Portfolio documents don't ship; case-sensitivity keeps ordinary words ("decisions", "learnings") legal | `supersedes LEARNINGS 2026-08-05, F5` |

**Family B — repo-machinery phrases** (the strict ban: the development repository is not acknowledged even pathlessly):

| Pattern | Case | Property it protects | Example violation |
|---|---|---|---|
| `drift (guard\|tripwire)` | insensitive | Enforcement machinery is repo-side; the surface never names its own watchers | `the drift tripwire turns CI red` |
| `(source\|development\|parent) repositor(y\|ies)` | insensitive | The repo is not acknowledged in any form | `a build-time guard in the source repository` |
| `\bCI\b` | sensitive | Merge gating is repo machinery (`\b` keeps "CLI" safe; case keeps prose safe) | `turns CI red` |

**Known-safe tokens** — non-matching by construction, each pinned as a guard fixture so a lexicon edit cannot silently widen: `prp_0123` / `ten_…` / `role_…` example ids (no word boundary after `_`), version strings (`0.34.1`), exit codes `0`–`7`, `--per-page 500`, "the v5 spec" / "specification" (only slash-form paths and id-form numbers match), "CLI".

The lexicon is the strict ban's *reach*, not its *definition* — the definition is Principle XIII; novel phrasings beyond these patterns are a review obligation (plan Risk 1).

### 2. Guard surface

| File (all in `internal/build`) | Responsibility |
|---|---|
| `surfaceselfcontainment.go` | Production source: the surface walker, the two-family lexicon (each entry: pattern, property comment, example), the scan producing violations, the `plugin/…`-path resolution check. Shared by the guard test and the BDD suite. |
| `surface_self_containment_guard_test.go` | The live tripwire: walks the real `plugin/` from the repo root (located via the same repo-root helper the sibling guards use) and fails on any violation or on a missing/empty surface. |
| `surface_self_containment_bdd_test.go` | godog suite for this feature's file under `features/unequipped-agent-operators/`, carrying two step families. **Detection steps** drive the production functions against `t.TempDir()` fixture surfaces only — never against the real `plugin/`, so a seeded violation is never introduced into the checkout. **Surface-content steps** read the real swept surface read-only, asserting its text names in-plugin components. The invariant across both: the suite never writes to `plugin/`, and never runs the walker against it — the live pass belongs to the guard test above. |

Invocation: `go test ./...` — no new command, flag, or CI wiring; 024's `ci.yml` and 029's `main-verify.yml` already run it. Configuration surface: none — deliberately (plan Cross-cutting Concerns); the walk root and lexicon are hard-coded contract.

### 3. Registry-header instructional contract

Post-sweep shape of the seven instructional artifacts (`hooks/gated-commands.txt`, the five `agents/*-commands.txt`, `hooks/glassfrog-write-gate.sh`). Each header keeps, in surface-only terms:

- **What the file is** — the registry's role as the single source of its command set.
- **The in-plugin consumer list** — every consumer named by `plugin/`-relative path (e.g. the gate script, the agent artifact that composes the leaves).
- **The editing rule** — what a line is, what belongs and what never does (reads vs. gated writes), and what to do when the CLI's surface grows.
- **The fail-closed consequence, stated through the surface's own behavior** — e.g. "an unrecognized `proposal` subcommand is asked, not waved through; an edited registry and the gate can disagree about what is gated" — never through repo machinery.

And contains nothing the lexicon matches: no guard-test paths, no plan-ADR citations, no "turns CI red" consequences, no spec-number ids. The hook script keeps its one-sentence rationales where an ADR citation sat beside them; only the citation goes.

### 4. Constitution Principle XIII structural contract

- Heading `### XIII. Self-Contained Operating Surface`, following XII.
- Three parts in sibling form: a **bold statement** (every reference inside `plugin/` resolves within the surface or to the CLI it drives; the development repository is not referenced in any form; pointers flow repository → surface only), a *Rationale* paragraph, and a *Detection* paragraph naming the observable: any surface file referencing the development repository fails the repository's verification run.
- **Version marker**: the Governance section gains an explicit version line — initial value `1.0` acknowledging the pre-existing twelve principles, bumped to `1.1` by this amendment — with the amendment's justification recorded adjacent (date, what was added, why). Future amendments bump it mechanically.

---

## Interactions

**Operator reading the surface** (the contract's beneficiary): every handoff, boundary, and deferral resolves in place — by in-plugin name or resolving `plugin/` path — with only the plugin and the CLI installed. No workflow step requires the repository.

**Developer editing or adding a surface file**: write → `go test ./...` → the guard reports every violation in the run (not first-only) with file, line, matched text, family, property, and remedy → fix per remedy → green. New files are covered automatically; there is no registration step.

**Developer extending or relaxing the lexicon** (deliberate contract edit, reviewed like a gated-commands change): add or narrow a pattern *with* its property comment and concrete example, add the corresponding fixture (a violating sample for a new pattern; a known-safe sample for a relaxation), and keep every existing fixture green — a relaxation that un-matches a current violation class is a red fixture, not a merge.

**Amending the constitution**: edit CONSTITUTION.md per its Governance section — bump the version marker, record the justification adjacent. This feature's amendment both establishes the marker and performs the first bump.

**Sweep execution** (one-time, Phase 1): apply plan.md's Sweep Design mapping and header shapes; the Family A/B patterns double as the sweep's completeness check (plan Risk 4) — run them as a local grep before Phase 2 lands the guard.

---

## Error Communication

**Violation report grammar** (one line per violation; the test failure aggregates all of them plus a count):

```
plugin/<path>:<line>: forbidden reference "<matched text>" (family <A|B>: <property>). Remedy: replace with the in-plugin component name, or remove the reference.
```

The message is a specification: the stated remedy must be reachable for every condition it accompanies, and it is — every Family A/B match either has a mapped in-plugin name (plan's Sweep Design) or is deletable prose.

**Condition table** (each row's remedy is checked against its siblings — no remedy satisfies one row by tripping another):

| # | Condition | Guard behavior | Remedy stated |
|---|---|---|---|
| 1 | A surface line matches a Family A pattern | Fail; report per the grammar | Replace with the in-plugin component name, or remove the reference |
| 2 | A surface line matches a Family B phrase | Fail; report per the grammar | Reword to state the rule through the surface's own consequences, or remove the mention |
| 3 | A `plugin/…`-shaped path in a surface file does not resolve to an existing file | Fail; report the dangling path | Correct the path to the existing in-surface file, or remove the reference |
| 4 | `plugin/` is missing or the walk finds zero files | Fail loudly ("surface missing or empty — nothing to verify") | Restore the surface checkout; this condition is never answered with a lexicon edit |
| 5 | Legitimate new content trips a pattern (false positive) | Identical to row 1/2 — the guard cannot distinguish; the developer judges | If the reference genuinely serves the operator, narrow the pattern via the deliberate lexicon edit flow (property comment + known-safe fixture); the edit must keep rows 1–4's existing fixtures green |

Row 5's remedy edits the guard, not the surface — the only condition where that is the right direction, and the fixture rule is what keeps it from quietly weakening rows 1 and 2.

**Degradation**: none. The guard has no warning tier, no skip path, and no partial pass — a walk error is a failure, not a skip.

---

## Consistency Notes

- **Single touchpoint** — no sibling interface files for this feature.
- **No `accords/` directory exists** — the closest standing patterns are conventions from process memory, followed here: the 063 registry pattern (the *checked set* derived by walking, the *definition of forbidden* a checked-in contract-fact, mirroring how gated-commands.txt anchors the gate); property-commented pinned values (a guard's literal encodes a property — the comment is the property); whitespace-collapsed content assertions for the re-derived name-pins (the operator-path BDD convention).
- **Announced divergence** (carried from plan ADR-3 / spec non-behavior 2): re-derived feature-scenario step text diverges from the merged specs' Connextra prose it was copied from; already-merged specs (the leaked 062–073 records and every other merged sibling) stay untouched, git history is the changelog.
- **Naming**: `surface_self_containment_*` is deliberately distinct from 021's `selfcontainment_test.go` (binary linkage) — two different self-containments, two different names.
- **Direction rule made concrete here**: this accord, the guard, and its fixtures all reference `plugin/` freely; nothing in `plugin/` references any of them — the accord itself is an artifact the surface must never cite.
