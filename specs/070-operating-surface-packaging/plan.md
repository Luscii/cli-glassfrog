# Plan: Operating-Surface Packaging

**Feature**: 070-operating-surface-packaging
**Role**: Shaper
**Inputs**: spec.md, PROJECT.md, DECISIONS.md (operator-surface precedent §408–§459), LEARNINGS.md, DEPRECATION.md (no relevant deprecations), marketplace-name resolution from the developer

---

## System Architecture

Operating-Surface Packaging adds three small, committed artifacts around the plugin that 062–069 built — no Go CLI code beyond the established drift-guard pattern, and no change to the plugin's existing content:

1. **Marketplace manifest** — `.claude-plugin/marketplace.json` at the **repo root**: a hand-authored, committed Claude plugin marketplace named `glassfrog`, owned by Luscii, whose `plugins` list carries one entry today — the `glassfrog` plugin, sourced by relative path from the in-repo `plugin/` directory. This is the entire discovery/install surface: an agent environment runs `/plugin marketplace add Luscii/cli-glassfrog`, the host reads this manifest, and the plugin installs from the same repo checkout the marketplace lives in. The manifest is general in *shape* (a list), so a future sibling plugin (e.g. Holacracy practice, from its own repo) is one appended entry, never a restructure.

2. **`glassfrog-setup` skill** — `plugin/skills/glassfrog-setup/SKILL.md`: a new sibling skill added to the existing plugin per the additive-growth precedent (062 ADR-2). It is packaged operating knowledge, not code: it instructs the invoking agent to run a **presence check** (is the `glassfrog` binary on PATH and runnable?) and an **auth check** (does a low-cost authenticated identity read succeed?), and tells it what to do on each failure — point at the CLI's three existing install channels (install script, Homebrew tap, npm wrapper) when the binary is missing; walk the CLI's existing `X-Auth-Token` credential setup when the auth check fails. When both checks pass it reports the environment ready. It runs in the **caller context with no subagent** — unlike the operator paths, there is no multi-step record work to isolate.

3. **Consistency drift guard** — a new `internal/build/operatingsurfacepackaging.go` (path constants + parse helpers in production source, per the family convention) with guard tests extending the §417/§426/§438 best-effort pattern: parse the marketplace manifest and assert its glassfrog entry resolves to the committed plugin definition and matches its identity (`plugin/.claude-plugin/plugin.json` `name`), with both sides derived from the files — never a hard-coded expected value. A second best-effort guard anchors the setup skill's enumerable facts (the named install channels, the auth-check command leaf) to their in-repo sources.

Data flow at operate time is unchanged: the plugin host installs and surfaces the skills; the skills drive the separately-installed CLI; the CLI talks to the API. Packaging touches only the leftmost hop (getting the plugin into the host).

**Context loading strategy** (the skill is an instruction artifact): `glassfrog-setup` is description-triggered and self-contained for the *journey* (check → fix → verify), deferring *reference* knowledge outward — per-command detail to `glassfrog <command> --help`, cross-cutting credential/exit-code knowledge to the orientation skill. It restates neither.

---

## Architecture Decisions

### ADR-1: The marketplace is named `glassfrog` and lives at the repo root

**Context**: The marketplace name is the identity operators install against (`/plugin install glassfrog@<marketplace-name>`) and every future sibling plugin rides under it — renaming later breaks every documented install command. The spec calls for a general marketplace; a separate **private Luscii marketplace already exists**, so the org-level name is taken.

**Options considered**:
1. **`luscii`** — org-level identity. Collides with the existing private Luscii marketplace.
2. **`luscii-agents`** — narrower org identity. Still overlaps the org marketplace's territory and doesn't say what it carries.
3. **`glassfrog`** — the glassfrog-family marketplace, matching the repo and the plugin it ships. Reads slightly redundant at install time (`glassfrog@glassfrog`).

**Decision**: Option 3 — `glassfrog` (developer-resolved). The manifest lives at `.claude-plugin/marketplace.json` in the repo root — the location a Claude plugin host discovers when the repo is added as a marketplace — with `owner` Luscii.

**Consequences**: Install commands are stable and documentable (`/plugin marketplace add Luscii/cli-glassfrog`, then `/plugin install glassfrog@glassfrog`). The generality the spec requires must therefore live in the manifest's *shape*, not its name (ADR-2). A future Holacracy-practice plugin listed here is a glassfrog-adjacent offering under a glassfrog-named marketplace — acceptable, and it can equally move to the private Luscii marketplace if that fits better then.

### ADR-2: General in shape — one list entry now, additive growth, no version pin

**Context**: The spec's non-behavior forbids a marketplace "locked to a single plugin such that adding a sibling requires restructuring", while today exactly one plugin exists. The plugin manifest already carries its own `version` (0.1.0).

**Options considered**:
1. **Single-plugin manifest form** — flat manifest describing just the glassfrog plugin. Smallest, but adding a sibling restructures the file — exactly what the spec forbids.
2. **List-shaped manifest, one entry, source by relative path, no version pin** — `plugins: [{name: "glassfrog", source: "./plugin", …}]`. A sibling is one appended entry (in-repo path or cross-repo source); the installed version is whatever the checkout carries, single-sourced in `plugin.json`.
3. **List-shaped with a version-pinned entry** — adds a second place the version lives, which drifts from `plugin.json` on every bump.

**Decision**: Option 2. The entry names the plugin, points at `./plugin`, and carries a description consistent with `plugin.json`; version stays single-sourced in `plugin.json`.

**Consequences**: The "second plugin" edge-case scenario is satisfied structurally. No release-time step touches the manifest. The identity duplication that *does* exist (the `name` appears in both manifest entry and `plugin.json`) is exactly what ADR-5's guard anchors.

### ADR-3: `glassfrog-setup` ships in-plugin as a thin caller-context skill — no subagent

**Context**: The spec assumes the setup skill ships inside the plugin. The operator paths (064–069) each minted a subagent, one per path, and 069 explicitly closed that family. Setup is not an operator path — it does no governance-record work; it provisions the environment.

**Options considered**:
1. **Thin skill + subagent** — follows the paths' both-form. Wrong fit: nothing to isolate; the checks are two cheap commands and the fixes are interactive guidance the operator must see.
2. **Caller-context skill only** — the invoking agent runs the checks itself and walks the operator through fixes in-conversation.

**Decision**: Option 2. `plugin/skills/glassfrog-setup/SKILL.md`, description-triggered, added additively alongside the seven existing skills. This is a deliberate, announced divergence from the paths' thin-skill+subagent shape — the one-agent-per-path convention stays intact because setup is not a path.

**Consequences**: `plugin/agents/` is untouched; the closed agent family stays closed. Setup's guidance is inherently interactive (install instructions, credential entry), which caller context serves better than an isolated agent.

### ADR-4: Presence and auth checks are instructed knowledge, not shipped code

**Context**: The setup skill must check CLI presence and credential validity while the CLI stays self-contained and no capability is added (spec non-behaviors; PROJECT constraint "knowledge + guardrails, never capability").

**Options considered**:
1. **Ship a check script or a new CLI command** (`glassfrog doctor`) — executable, but adds surface: a script is a second distribution artifact to keep working everywhere; a CLI command is exactly the new capability the surface forswears.
2. **Instruct the checks in skill content** — the skill tells the agent to run an existing innocuous command for presence (e.g. `glassfrog --version`) and a low-cost authenticated identity read (`glassfrog me`) for auth, and to interpret failures via the CLI's exit-code convention (orientation's territory).

**Decision**: Option 2. Both checks reuse commands the CLI already exposes; the skill packages *how to interpret* their outcomes and *where to send the operator* on failure — the install channels (install script, Homebrew tap, npm wrapper — all shipped: 027/036/037) and the CLI's `X-Auth-Token` setup.

**Consequences**: Zero new runtime surface; the CLI remains the source of truth for whether a credential works. The named commands and channels become enumerable facts that can drift — anchored best-effort by ADR-5's guard. Exact command choice for the identity read is confirmed at implementation against the shipped CLI.

### ADR-5: The consistency guard is an `internal/build` test with both sides derived from source

**Context**: The spec makes marketplace↔plugin drift a defect. The family precedent (§417 orientation, §426 gate registry, §438 composed reads) guards committed artifacts with best-effort `internal/build` tests, and process memory warns: a drift guard must derive both sides — hard-coding the guarded value creates a second source of truth.

**Options considered**:
1. **No automated guard** — rely on review. Cheapest; but the spec explicitly promotes this consistency to a behavior, and manifest drift is silent (nothing else parses it in CI).
2. **`internal/build` guard test + production-source constants** — parse `.claude-plugin/marketplace.json`, locate the glassfrog entry, assert its `source` resolves to the committed plugin directory and its `name` matches `plugin.json`'s, both read from disk at test time. Extend the same best-effort approach to the setup skill's enumerable facts.
3. **Standalone CI script** — new mechanism outside the established Go-test gate; nothing else in the repo works this way.

**Decision**: Option 2 — `internal/build/operatingsurfacepackaging.go` (paths + parse helpers) with `operating_surface_packaging_guard_test.go`, running under the ordinary `go test ./...` CI gate.

**Consequences**: Drift fails CI, satisfying the "defect, not difference" accord. The guard stays best-effort: it verifies in-repo consistency (entry ↔ plugin definition, named facts ↔ their sources), not the host's marketplace schema — the host's evolution is a risk, not a testable invariant here.

---

## Cross-cutting Concerns

**Error handling**: Packaging artifacts have no runtime error path of their own — failures surface either at install time (host rejects/can't find the manifest: fixed by the guard keeping the manifest valid and consistent) or during setup (checks fail: the skill's whole purpose — each failure maps to a directed fix, never a dead end). The setup skill must keep the two failure classes distinct: *missing binary* → install channels; *failing auth* → credential setup.

**Testing strategy**: Three layers, all existing patterns — (1) the ADR-5 guard tests; (2) BDD scenarios in `features/unequipped-agent-operators/` asserting the committed artifacts' content (whitespace-normalized for content assertions, raw for structural ones, per the family's convention); (3) the spec's validation scenarios stay held out for independent verification. Manual smoke: add the repo as a marketplace in a real Claude Code session and install the plugin once at implementation time.

**Configuration**: None. Everything is committed and hand-authored; the only "configuration" is the operator's own credential, which stays entirely in the CLI's existing mechanism.

**Documentation**: README gains an install section for the agent operating surface (marketplace add + plugin install), beside the existing CLI install section; `docs/guides/agent-operators/` gets the same journey in guide form. Doc drift on command names is covered by review, not the guard (prose, not enumerable).

---

## Implementation Strategy

Two phases, independent in content, sequenced for review clarity:

**Phase 1 — Distribution vehicle**: the repo-root `.claude-plugin/marketplace.json` (ADR-1/-2), the `internal/build` packaging source file + marketplace-consistency guard (ADR-5), and the README/docs install section. Lands the discover/install behaviors and the consistency accord.

**Phase 2 — Setup skill**: `plugin/skills/glassfrog-setup/SKILL.md` (ADR-3/-4), its enumerable-facts guard (extending Phase 1's guard file), and the BDD coverage for the setup journey's content. Lands the presence-check/auth-check/ready behaviors.

Phase 2 does not depend on Phase 1 functionally (the skill ships inside the plugin either way), but Phase 1 first means the plugin is installable before setup content lands — the natural demo order.

---

## Risks

1. **Host marketplace-schema evolution** — Claude Code's marketplace manifest format may gain/rename fields. *Likelihood*: low-moderate; *impact*: install breaks until the manifest is touched up. *Mitigation*: keep the manifest minimal (name, owner, one entry); verify the exact field set against current Claude Code docs at implementation time; the guard checks internal consistency, so a host-schema change is a doc-check, not a test rewrite.
2. **Setup-skill fact drift** — the named install channels or the identity-read command could change ahead of the skill. *Likelihood*: low; *impact*: setup directs operators at a dead channel. *Mitigation*: ADR-5's best-effort guard anchors the enumerable facts; detail defers to CLI help and orientation, keeping the drift surface small (062's proven approach).
3. **Duplication seam with orientation** — setup and orientation both touch credentials; restating detail in setup creates a second drifting source. *Likelihood*: moderate at authoring time; *impact*: contradictory guidance. *Mitigation*: the boundary is explicit — setup owns the *journey* (check → fix → verify), orientation owns the *reference* (how credentials/exit codes work); setup links and never restates.
4. **Generality stays untested until a second plugin arrives** — the list-shaped manifest's extensibility is asserted, not exercised. *Likelihood*: n/a (structural); *impact*: low. *Mitigation*: the spec's validation scenario inspects the shape; adding the future entry is the real test, and ADR-2 keeps it a one-line change.

---

## What This Plan Does Not Cover

- **Structural contracts** — the marketplace manifest's exact field set, the setup skill's frontmatter/description wording, and the guard's checked-fact list are the interface skill's concern (specification boundaries: two declarative artifacts + a guard contract).
- **Executable scenarios** — the scenarios skill translates the spec's driving scenarios into `features/unequipped-agent-operators/` Gherkin.
- **Task decomposition** — the tasks skill turns the two phases into PR-sized units.
- **Plugin content changes** — nothing in 062–069's skills, agents, or hook is modified; the orientation skill is not extended (any future "setup exists" cross-reference from orientation is that skill's own additive change, not this feature's).
- **CLI distribution itself** — the install script, Homebrew tap, and npm wrapper are consumed as-is; packaging points at them and changes nothing about them.
