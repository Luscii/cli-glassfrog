# Plan: Operator Orientation

**Feature**: 062-operator-orientation
**Role**: Shaper
**Inputs**: spec.md, PROJECT.md, DECISIONS.md (targeted precedent grep — §175/§203/§309/§316 config-drift guards, §399 write-safety-guardrail deferral, §318 generated-vs-committed packaging), DEPRECATION.md (no relevant entries)

---

## System Architecture

Operator Orientation is not a Go feature — it adds **no code to the CLI**. It introduces a new, separate top-level artifact: a **Claude plugin** that packages operating knowledge as skill content an agent consults on demand. The CLI (`internal/cli`, `internal/output`, `internal/auth`, `internal/paging`, …) is unchanged; the plugin *describes* its existing surface.

Three parts:

1. **Plugin package** — a dedicated top-level `plugin/` directory holding a Claude plugin manifest (`plugin/.claude-plugin/plugin.json`) and a `plugin/skills/` directory. The manifest makes the directory a well-formed, installable plugin; it does **not** publish or distribute it (that is #70). The directory is laid out so the Write-Safety Guardrail (#63) and the operator paths (#64–#69) drop in later as additional skills under `plugin/skills/` (and, for the guardrail, possibly `plugin/hooks/`) without restructuring.

2. **Orientation skill** — a single skill (`plugin/skills/glassfrog-operator/SKILL.md`, working name) carrying the hand-authored, cross-cutting operating knowledge: how to select a structured output format and what shape to expect, how to page, what each exit code means and the reaction, how credentials are supplied, and the write-safety *expectation* (as guidance, not enforcement). For per-command/per-flag detail it points the agent at the CLI's own `glassfrog help` / `--help`, rather than cataloguing it — keeping the drift surface minimal.

3. **Drift guard** — a best-effort `internal/build` config-drift test that anchors the small set of *stable facts* the skill states against their source of truth in the CLI, so the skill cannot silently drift as the CLI evolves. This is the project's existing config-guard pattern (§175/§203/§309/§316) applied to a new artifact.

Data flow: there is no runtime flow inside this feature. The agent (the consumer) reads the skill content on demand and then drives the *real* CLI, which talks to the Glassfrog API as it already does. Orientation sits entirely upstream of any execution, as static knowledge.

---

## Architecture Decisions

### ADR-1: Home the plugin in a dedicated top-level `plugin/` directory with the standard Claude layout

**Context**: The orientation knowledge ships as a Claude plugin (spec; developer-confirmed). The repo is primarily a Go CLI with sibling packaging channels (`npm/`, `install.sh`, GoReleaser tap). The plugin needs a home that keeps it separate from Go code and leaves room for #70's marketplace and for future skills.

**Options considered**:
1. **Root-level `.claude-plugin/` + `skills/`** — the whole repo becomes the plugin. Zero nesting, but conflates a Go CLI repo with a plugin root, pollutes the repo root, and gives the marketplace (#70) nowhere natural to sit.
2. **Dedicated top-level `plugin/` directory** — `plugin/.claude-plugin/plugin.json` + `plugin/skills/`. One extra level; cleanly separated from Go code; #70's marketplace manifest can sit alongside (e.g. `plugin/.claude-plugin/marketplace.json` or a sibling) without touching this spec's files.

**Decision**: Option 2 — dedicated `plugin/` directory, standard Claude plugin layout. The plugin is a committed, first-class repo artifact distinct from the compiled CLI.

**Consequences**: A new top-level directory enters the repo. Distribution (#70) extends *this* directory rather than restructuring it. The exact manifest schema and field contracts are interface-level.

### ADR-2: Ship one orientation skill now; structure the plugin to grow

**Context**: The spec deferred skill decomposition to shaping. The Agent Operating Surface will eventually hold the guardrail (#63) and several operator paths (#64–#69).

**Options considered**:
1. **One orientation skill** — all cross-cutting knowledge in a single `skills/glassfrog-operator` skill. Simplest; one description-triggered unit; matches the on-demand consumption model.
2. **Several fine-grained skills now** (output / pagination / exit-codes / credentials) — finer triggering, but fragments tightly-related knowledge, multiplies frontmatter/descriptions, and pre-commits a decomposition before the paths that would justify it exist.

**Decision**: Option 1 — a single orientation skill now. The `plugin/skills/` directory is the extension point: #63 and #64–#69 arrive as *sibling* skills (and the guardrail may add `plugin/hooks/`), so growth is additive, not a restructure.

**Consequences**: One skill to author and keep truthful. Future skills are independent additions. If the single skill grows unwieldy as paths land, a later spec can split it — cheaply, because consumers trigger by description, not by a fixed file set.

### ADR-3: Hand-author committed skill content; defer per-command detail to the CLI's own help

**Context**: The skill must stay truthful to the shipped CLI (spec: drift is a defect). The clarify session resolved that orientation packages cross-cutting knowledge only and points at the CLI's built-in help for per-command/flag detail. Precedent §318 (037 npm) *generates* packaging from source rather than committing scaffolding.

**Options considered**:
1. **Generate the skill from the CLI** (e.g. from `--help` dumps / command registry) — drift-proof for command detail, but tooling-heavy, and most orientation content (parsing strategy, exit-code *reactions*, write-safety guidance) is judgement that has no machine source to generate from.
2. **Hand-author, committed; defer per-command detail to `glassfrog help`** — the cross-cutting knowledge is written prose; the volatile per-command catalogue is never duplicated, so the drift surface shrinks to a handful of stable facts.

**Decision**: Option 2. Unlike the npm channel (pure mechanical repackaging, so generation wins), orientation is mostly judgement; generation has little to bite on. Committed hand-authored content, with command detail single-sourced in the CLI's help.

**Consequences**: The skill is editable prose reviewers can read. The residual drift surface — output-format names, exit-code meanings, the credential command, pagination — is small and stable, which is what makes ADR-4 feasible.

### ADR-4: Guard drift with a best-effort `internal/build` config-drift test over the stable anchors

**Context**: The developer wanted a drift check "if feasible," accepting partial or none. The CLI exposes stable anchors the skill references: the `supportedFormats` constant (`internal/output/format.go` — `full, compact, json, yaml`), the `Outcome`→`ExitCode` mapping (`internal/cli/exitcode.go`, 0–7 — including `codeStaleWrite=7`, the 412 anchor the orientation references), the `auth login` command (`internal/cli/authcmd.go`), and `internal/paging`. The repo already guards config drift this way (§175 `.goreleaser` matrix, §203/§309 label catalog, §316 brews target).

**Options considered**:
1. **No automated guard** — rely on review. Cheapest; but the spec calls drift a defect, and review misses it.
2. **Full prose-vs-CLI semantic check** — parse the skill and verify every claim. Infeasible: most claims are judgement, not machine-checkable.
3. **Best-effort anchor test** — a test in `internal/build` that asserts the *enumerable* facts the skill names still match their CLI source (the four format tokens appear and are exactly the supported set; the exit-code numbers/labels the skill cites match `ExitCode`; `auth login` still exists). Skips the unenumerable prose.

**Decision**: Option 3 — a focused, best-effort anchor test, scoped to what is mechanically checkable. Explicitly partial, per the spec assumption; a gap in coverage is acceptable, a false sense of total coverage is not.

**Consequences**: The most regression-prone facts (format names, exit codes) are pinned; the rest stays a review concern. The test lives in `internal/build` beside the existing config guards. If anchoring proves harder than expected during implementation, shipping with reduced or no automated coverage is acceptable (spec) — but the reduction must be stated, not silent (LEARNINGS: no silent caps).

### ADR-5: Define the plugin so it *can* be installed; leave distribution to #70

**Context**: This spec partially absorbs the former #70 — the plugin definition lands here, the marketplace/publishing/install flow stays in #70 (spec non-behavior + clarify scope note).

**Options considered**:
1. **Include a marketplace manifest + install flow here** — one-shot installable surface, but re-absorbs the half the developer explicitly split out and couples content to a delivery mechanism that should evolve independently.
2. **Manifest only; no marketplace/publishing** — the plugin is a well-formed, locally-installable package; how it is discovered and distributed is #70.

**Decision**: Option 2. Produce a valid `plugin.json` manifest and skill content; produce **no** marketplace manifest, publishing workflow, or install instructions.

**Consequences**: The plugin is complete enough to install from a local path but not yet *distributed*. #70 extends `plugin/` with the marketplace and (following the §316/§318 channel pattern) the publishing path. A validation scenario asserts no distribution machinery leaked in.

---

## Plugin Structure (Specification Boundary)

This feature's boundary is a **specification boundary** — it produces a declarative artifact (a plugin + skill) that an external consumer (the Claude Code plugin host / an agent) reads. At architectural level:

- **What it produces**: a plugin manifest and one skill document, plus a drift-guard test.
- **What the consumer expects**: a manifest the plugin host can load (identity + skill discovery), and a skill whose frontmatter description triggers on-demand loading when the agent faces an unknown about driving the CLI.
- **Invocation surface**: none at runtime in the CLI sense — the "invocation" is the host loading the skill and the agent reading it.

Protocol-level detail — the exact `plugin.json` field set, the SKILL.md frontmatter contract (name/description and what makes the description trigger well), and the precise section layout of the orientation content — is the **interface** skill's concern (`interface-spec.md`).

---

## Cross-cutting Concerns

- **Truthfulness / drift** — the dominant cross-cutting concern, handled by ADR-3 (defer volatile detail to `--help`) + ADR-4 (anchor test). Drift is treated as a defect, not a difference.
- **Testing strategy** — two kinds: (1) the `internal/build` anchor test (ADR-4); (2) structural/static checks that the produced artifact is well-formed (manifest parses; skill exists; no distribution machinery present; no write-gating logic present). There is no runtime behavior to integration-test — the consumer is an external agent.
- **Configuration** — none. The plugin adds no config to the CLI and introduces no credential mechanism (it points at the existing `auth login`).
- **Boundary discipline** — the skill must not assert any command/flag the CLI lacks (no invented surface) and must not implement gating. These are review + validation-scenario concerns, reinforced by the anchor test for the enumerable part.

---

## Implementation Strategy

**Phase 1 — Plugin scaffold + orientation content.** Create `plugin/.claude-plugin/plugin.json` and `plugin/skills/glassfrog-operator/SKILL.md`. Author the cross-cutting orientation content (output formats, pagination, exit-code reactions, credential setup, write-safety guidance; per-command detail deferred to `glassfrog help`). This phase is self-contained and is the bulk of the work; it depends only on the interface contract for the manifest/frontmatter shape.

**Phase 2 — Drift guard.** Add the best-effort anchor test in `internal/build` asserting the skill's enumerable facts match their CLI source (formats, exit codes, `auth login`). Depends on Phase 1 (the content must exist to anchor against). If anchoring proves infeasible within reasonable effort, reduce scope and state the reduction.

The phases are PR-separable (content first, guard second); tasks decomposes the specifics.

---

## Risks

- **R1 — Orientation drifts from the CLI** (medium likelihood, medium impact). A command/flag/format changes and the prose goes stale. *Mitigation*: ADR-3 shrinks the surface (per-command detail lives in `--help`); ADR-4 pins the enumerable anchors. Residual prose drift remains a review concern — accepted per the spec.
- **R2 — Plugin manifest / skill-trigger conventions are host-version-specific** (medium likelihood, low–medium impact). The `plugin.json` schema and what makes a skill description trigger reliably depend on the Claude Code plugin host's current conventions. *Mitigation*: interface pins the schema against the host's documented format at design time; treat it as an external contract that may need revisiting if the host changes.
- **R3 — Scope creep into enforcement or paths** (low likelihood, medium impact). Tempting to add a little gating or a navigation helper. *Mitigation*: spec non-behaviors + a validation scenario; #63 and #64–#69 are named owners.
- **R4 — "Oriented" is not runtime-observable** (certain, low impact). Success is the agent *having* the knowledge, which can't be asserted by running the CLI. *Mitigation*: verify via content/structure checks and the anchor test, not behavioral execution; scenarios are framed around consulting the skill, not a CLI exit.

---

## What This Plan Does Not Cover

- **Exact manifest & frontmatter contracts** — `plugin.json` fields and the SKILL.md frontmatter/section contract are the **interface** skill's output (`interface-spec.md`).
- **Marketplace, publishing, install flow** — deferred to **#70** (ADR-5).
- **Write-safety enforcement** — the **Write-Safety Guardrail (#63)**; orientation only describes the expectation.
- **Operator paths** — navigation / tension / proposal journeys are **#64–#69**, future sibling skills.
- **Executable scenarios** and **task decomposition** — the **scenarios** and **tasks** skills.
