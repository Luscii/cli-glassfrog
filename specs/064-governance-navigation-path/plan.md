# Plan: Governance Navigation Path

**Feature**: 064-governance-navigation-path
**Role**: Shaper
**Inputs**: `specs/064-governance-navigation-path/spec.md`, PROJECT.md, `.score/memory/DECISIONS.md` (loaded — precedent from 062/063 and the proposal-write specs), `.score/memory/LEARNINGS.md` (passive), `.score/memory/DEPRECATION.md` (no relevant entries for 064)

---

## System Architecture

Governance Navigation Path is the first **operator path** on the Agent Operating Surface. It adds no Go CLI code and no API capability — it is a pair of declarative plugin artifacts, hand-authored and committed under the existing top-level `plugin/` tree (the home 062 ADR-1 established), plus one best-effort Go drift-guard test that anchors the artifacts to the shipped command surface.

The path is delivered in **two cooperating pieces**:

- **A thin skill** — `plugin/skills/governance-navigation/SKILL.md`. This is the discoverable, description-triggered entry point (callable by an agent *or* a human user). It carries the *when* (reach for governance navigation to work a tension) and the *workflow* (the traversal steps: take a free-form concern → search to discover what it touches → read the relevant roles and who fills them → draw in the governing domains and policies → synthesize a picture, not raw dumps). It does not itself execute a long chain of reads in the caller's context; it delegates the traversal to the navigator subagent.

- **A read-only subagent** — `plugin/agents/governance-navigator.md`. This runs the workflow in its **own isolated context** with a **read-only tool grant**, then returns only the synthesized picture to the caller. Isolation is what structurally delivers the spec's core promise ("a synthesized picture rather than raw dumps"): the raw search/role/filler/domain/policy output stays in the subagent's context and never floods the caller's. The read-only tool grant withholds `Write`/`Edit`, so the subagent cannot mutate the workspace; governance-write prevention is layered — a tool grant cannot restrict which `glassfrog` subcommands `Bash` runs, so the prompt scopes execution to the read leaves and 063's `PreToolUse` hook gates any `glassfrog` write issued via `Bash` when present. "Surfacing, not judging" is likewise a prompt-level guardrail.

**Flow**: caller senses a governance concern → skill triggers (by description) and describes the workflow → skill delegates to the `governance-navigator` subagent → subagent composes the shipped reads (Cross-Model Search `search`, Role Reads `roles [id]`/`tree`, Role Fillers `fillers`/`subrole-actors`, and the role-interior `domains`/`policies`) in its own context → subagent returns the synthesized picture (roles, fillers, governing domains/policies), each element carrying the id that reads it again → caller receives the picture, not the dumps.

Everything the two artifacts name is behaviour the CLI already exposes. `plugin/agents/` is introduced here as a new component *type* under `plugin/`, extending the layout additively exactly as 062 ADR-2 anticipated for #64–#69 — not a restructure of what 062 shipped.

---

## Architecture Decisions

### ADR-1: Deliver the path as a thin skill delegating to a read-only navigator subagent (both, not one)

**Context**: The spec deferred the delivery form to shaping (spec Assumption 1). 062 ADR-2 projected that the operator paths would "arrive as sibling skills," but 063 already established the precedent that form follows the spec's *nature* (it became a hook because guidance cannot enforce). 064's core promise is "a synthesized picture rather than raw dumps," and its central non-behaviors are "must not write" and "surfacing, not judging." The developer chose both forms to get discoverability *and* context isolation.

**Options considered**:
1. **Skill only** — a `plugin/skills/` sibling to orientation. Discoverable and callable, consistent with 062's projection. But a skill is guidance the *caller* follows: the raw reads execute in the caller's context, so "synthesized not raw" rests on caller discipline, not structure, and read-only rests on the caller not choosing to write.
2. **Subagent only** — a `plugin/agents/` read-only agent. Structurally delivers synthesis-via-isolation and read-only-via-tool-grant, matches the ISSUE-TREE's "read-only governance-navigator agent." But an agent is less naturally a *human*-callable, description-triggered entry point, and the "when to reach for it + the workflow" knowledge has no natural home.
3. **Both — thin skill + subagent** — the skill owns *when* and the *workflow steps* and is the discoverable entry point; it delegates execution to the subagent, which owns isolated read-only execution and returns the synthesized picture.

**Decision**: Option 3 — both. The skill is the discoverable, callable surface carrying the when/workflow knowledge; the subagent is the isolated, read-only executor that returns the synthesized picture.

In practice: the skill's frontmatter `description` triggers it whenever a caller needs to understand the governance around a tension; its body describes the traversal workflow and instructs the caller to delegate to the `governance-navigator` subagent for execution. The subagent's frontmatter grants only read tools and its body runs the same workflow to completion in isolation, returning the synthesized picture. The two share one workflow definition — the skill *describes* it and points at the agent; the agent *executes* it — so there is a single source of truth for the steps, kept from drifting by writing the steps once in the skill and having the agent reference that workflow rather than restating it.

**Consequences**: Positive — discoverability (skill) + context isolation and structural read-only enforcement (agent); the raw dumps never reach the caller's context; the read-only and surfacing-not-judging non-behaviors become tool-grant guarantees. Negative — two artifacts to keep coherent (mitigated by single-sourcing the workflow steps in the skill and having the agent reference them), and the surface now has two component *types* (skills and agents), which #65–#69 may follow. Diverges from 062 ADR-2's "sibling skills" projection — an announced divergence: the paths that synthesize and must stay read-only are better served by an isolated read-only agent than by caller-followed guidance, the same reasoning that made 063 a hook.

### ADR-2: Extend `plugin/` additively with a new `plugin/agents/` component type — no restructure

**Context**: 062 ADR-1 established `plugin/` as the surface's home (`plugin/.claude-plugin/plugin.json` + `plugin/skills/`) and ADR-2 committed to additive growth. 064 introduces the first *agent*, so it needs a home for agent artifacts.

**Options considered**:
1. **`plugin/agents/governance-navigator.md`** — the conventional Claude-plugin location for agents, a sibling to `plugin/skills/`. Keeps the additive-growth contract and the surface in one tree.
2. **Fold the agent into the skill directory** — e.g. a file under `plugin/skills/governance-navigation/`. Keeps one path's artifacts together but breaks the plugin's conventional component layout and hides the agent from the host's agent discovery.

**Decision**: Option 1 — introduce `plugin/agents/` alongside `plugin/skills/`, registering the agent per the plugin's conventions.

This is silent conformance to 062 ADR-1/ADR-2's additive-growth home for everything except that `plugin/agents/` is new; the plugin manifest (`plugin.json`) is updated only as the host's agent-registration convention requires (interface-level detail). No existing artifact (the orientation skill, the manifest's existing fields) is moved or rewritten.

**Consequences**: The surface now has `skills/` and `agents/` subtrees; #65–#69 extend by adding sibling files, not restructuring. The exact manifest registration and agent frontmatter schema are the interface skill's concern. *Confirmed post-063:* 063's implementation landed `plugin/hooks/hooks.json` discovered from `plugin/hooks/` with **no** `hooks` array in `plugin.json` — concrete evidence that this plugin uses directory auto-discovery, so `plugin/agents/` needs no `agents` manifest key.

### ADR-3: Compose only already-shipped read commands; add no CLI code beyond a drift guard

**Context**: The "knowledge + guardrails, never capability" constraint (PROJECT) and the spec's non-behaviors forbid any new command, flag, or local governance logic. All the reads this path composes are shipped: Cross-Model Search (041), Role Reads (025), Role Fillers (047), Role Domains (033), Role Policies (034).

**Options considered**:
1. **Pure composition of shipped reads** — the skill/agent drive existing `glassfrog` commands only; the CLI is unchanged. Honors the constraint; zero API surface added.
2. **Add a convenience `navigate` command in the CLI** — one command that does the traversal. Rejected on sight: it would add API surface to the CLI, breaking "Bounded by the API surface" and VISION Exclusion 2, and duplicate composition logic the operator layer owns.

**Decision**: Option 1 — pure composition. The only Go change 064 makes is a test (ADR-4); no production CLI code changes.

**Consequences**: The path stays a faithful guide over the shipped surface. Per-command flags are never restated — the artifacts defer to `glassfrog <command> --help` (silent conformance to 062 ADR-3), keeping the drift surface to the *set of command leaves named*, not their flags.

### ADR-4: Guard artifact/CLI drift with a best-effort `internal/build` test pinning the composed command leaves

**Context**: 062 ADR-4 and 063 ADR-4 both anchor a plugin artifact's enumerable facts to their CLI source with a best-effort `internal/build` config-guard test, and LEARNINGS mandates that reduced/partial coverage be stated, not silent. 064's artifacts name a set of read-command leaves; if the CLI renames or drops one, the artifacts would silently drift.

**Options considered**:
1. **Best-effort `internal/build` drift test** — assert every command leaf the skill/agent name (`search`, `roles`, `tree`, `fillers`, `subrole-actors`, `domains`, `policies`) exists in the CLI's command registry. Pins the enumerable surface; leaves prose to review. Reuses the established home and idiom.
2. **No automated guard** — rely on review. Cheaper now; but drift is exactly the failure this surface is most prone to (062/063 both guarded it), and a silent-drift artifact reads as authoritative while being wrong.

**Decision**: Option 1 — a best-effort `internal/build` test pinning the named command leaves against the shipped command surface, mirroring 062/063.

Coverage is explicitly partial: it pins the *existence* of the command leaves the artifacts compose, not their flags (deferred to `--help`), not the synthesized-picture prose, and not parser robustness. The partial scope is stated in the test and the plan (no silent caps).

**Consequences**: A renamed/removed read command fails the build until the artifacts are updated. Establishes that operator *paths*, like the orientation and guardrail before them, carry a drift tripwire for the command leaves they name. *Pattern from 063 (landed):* 063 keeps its gated set in a single `plugin/hooks/gated-commands.txt` consumed by **both** the hook script and the drift test, so the two can't disagree. T002 should mirror this — single-source the composed-read leaf list in one file consumed by both the agent/skill artifact and the drift guard, rather than duplicating the list in prose and test.

### ADR-5: Read-only is layered — tool grant blocks workspace writes; prompt + 063 hook prevent governance writes; "surfacing, not judging" is prompt-level

**Context**: The spec's non-behaviors require the path never to write (deferring capture to 066 and drafting to the proposal paths) and never to judge authority (deferring that to Constraint Discovery 065). ADR-1's subagent form lets us make *workspace* mutation impossible and isolate the traversal; governance-write prevention still needs layering because a tool grant operates at the Claude-tool level (Bash, Write, Edit), not the `glassfrog`-subcommand level.

**Options considered**:
1. **Withhold `Write`/`Edit` and layer the rest** — grant the navigator `Bash` (needed to run `glassfrog` reads) plus read tools, but withhold `Write`/`Edit` so it cannot mutate the workspace; since `Bash` could still run a `glassfrog` write, scope the prompt to the read leaves and rely on 063's `PreToolUse` hook as the write backstop. Blocks workspace writes by construction; governance writes are prevented in layers.
2. **Rely on prompt instruction alone** — tell the agent not to write, with no tool restriction. Weaker: nothing structurally prevents even a workspace write.

**Decision**: Option 1 — the tool grant withholds `Write`/`Edit` (no workspace mutation), and governance-write prevention is layered: the prompt scopes execution to the read leaves and 063's `PreToolUse` hook gates any `glassfrog` write issued via `Bash` when present. A tool grant alone cannot restrict which `glassfrog` subcommands `Bash` runs, so "must not write" at the governance level is a *layered* guarantee, not a pure tool-grant one. The **boundary with 065** ("surface domains/policies, do not judge authority") *cannot* be tool-enforced either — reading a policy is legitimate for both paths — so it is enforced at the prompt level: the skill/agent describe surfacing the governing governance and explicitly hand any "am I allowed to do X?" judgment to the Constraint Discovery Path (065).

**Consequences**: Workspace mutation is structurally impossible (no `Write`/`Edit`); a governance write would require a `glassfrog` write command via `Bash`, which the prompt forbids and 063's hook gates when present — so the "must not write" non-behavior holds *in layers*, not from the tool grant alone. The softer boundary (judging) is a prose guardrail the artifacts state plainly. Defense is layered, not single-point; if 063's hook is absent, the prompt scope is the remaining governance-write control (and 064, being a read path, does not itself depend on 063 landing). The exact tool-grant syntax is interface-level.

---

## Cross-cutting Concerns

**Failure handling**: The subagent traverses defensively — a single failed read (e.g. a leaf role with no sub-role fillers to roll up, or a read that errors) does not abort the picture; the agent reports what failed and returns the picture assembled from the reads that succeeded (spec error scenarios). A search that matches nothing yields an explicit "nothing relevant found, refine the concern" rather than a fabricated picture. The agent relies on the orientation skill (062) for how to read exit codes and page results — it does not restate that knowledge (single-sourcing).

**No new observability/config**: This path adds no logging, config, or state — it composes reads. The only "configuration" is the read-only tool grant declared in the agent's frontmatter.

**Testing strategy**: (1) the best-effort `internal/build` drift guard (ADR-4) pinning the composed command leaves; (2) BDD scenarios (Tier 2) exercising the spec's driving scenarios against the artifacts' described behaviour — concern→picture, drawing in domains/policies, circle sub-role follow, empty-search, partial-failure, over-broad narrowing, and the authority-question hand-off to 065. Because the artifacts are declarative prose, scenario coverage verifies described behaviour and the drift guard verifies the named surface exists; there is no runtime Go path unique to 064 to unit-test beyond the guard.

---

## Implementation Strategy

Single phase — the feature is one coherent unit with no internal sequencing that isn't obvious:

1. **Author the skill and agent, register the agent, add the drift guard.** Write `plugin/skills/governance-navigation/SKILL.md` (when + workflow, delegating to the agent), write `plugin/agents/governance-navigator.md` (read-only grant + isolated execution returning the synthesized picture), update `plugin/.claude-plugin/plugin.json` per the host's agent-registration convention, and add the best-effort `internal/build` drift test. These land together because the drift test pins the command leaves the artifacts name, and the manifest registration is what makes the agent discoverable.

The tasks skill may still split this into PR-sized units (e.g. artifacts + registration in one, drift guard in another), but there is no cross-dependency forcing an order beyond "the artifacts must name the leaves the guard pins."

---

## Risks

- **Skill/agent drift from each other** (medium likelihood, low impact): two artifacts describing one workflow can diverge. *Mitigation*: single-source the workflow steps in the skill and have the agent reference that workflow rather than restating it (ADR-1); the drift guard does not cover this, so review must.
- **Host doesn't support subagents or read-only tool grants** (low likelihood, medium impact): the isolation and structural read-only enforcement depend on a plugin host that honours agent definitions and tool restrictions (external contract, like 062 R2 and 063's `PreToolUse` dependency). *Mitigation*: the skill remains useful as guidance even where the agent can't run isolated, and 063's guardrail still gates writes; #70 owns getting the surface installed in a capable host.
- **063's write hook may not cover the subagent's Bash** (low likelihood, medium impact): 063 landed as a `PreToolUse` matcher on `Bash`; if the host does not apply it to a *subagent's* Bash calls, the navigator's read-only guarantee rests on prompt scope + the `Write`/`Edit`-withheld grant, not on the 063 backstop (see risk.md H-4 / Post-063-Landing Note). *Mitigation*: T001 confirms subagent hook coverage against the target host and keeps the navigator's prompt strictly read-only regardless; the reads pass the gate ungated either way.
- **Boundary bleed into Constraint Discovery (065)** (medium likelihood, low impact): because 064 reads domains/policies, it is tempting to answer "can I do X?" *Mitigation*: ADR-5's prompt-level guardrail and a spec non-behavior; a validation scenario asserts no authority verdict appears.
- **Traversal-depth judgment** (low likelihood, low impact): "relevance-bounded" depth (spec Assumption 3) is judgment, not an algorithm, so a navigator could over- or under-traverse. *Mitigation*: the workflow describes narrowing over dumping; the over-broad-concern scenario pins the expected narrowing behaviour.

---

## What This Plan Does Not Cover

- **Protocol-level detail** — the exact `plugin.json` agent-registration fields, the skill and agent frontmatter schemas (name/description/tools), the precise read-only tool-grant syntax, and the shape of the synthesized picture the agent returns. These are the **interface** skill's concern (`interface-spec.md`).
- **Executable scenarios** — the Gherkin realizing the spec's driving/validation scenarios is the **scenarios** skill's concern.
- **Task decomposition** — PR-sized units and their acceptance criteria are the **tasks** skill's concern.
- **Distribution** — how the plugin (now carrying an agent) is published and installed remains Operating-Surface Packaging (#70); 064 only adds artifacts to `plugin/`.
- **The other operator paths (#65–#69)** — 064 establishes the skill+agent pattern and the `plugin/agents/` home; the other paths are separate specs that may reuse or diverge from it.
