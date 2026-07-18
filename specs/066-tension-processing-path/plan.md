# Plan: Tension Processing Path

**Feature**: 066-tension-processing-path
**Role**: Shaper
**Inputs**: `specs/066-tension-processing-path/spec.md`, PROJECT.md, `.score/memory/DECISIONS.md` (loaded — precedent from 062/063/064 and the tension command family 042–046), `.score/memory/LEARNINGS.md` (passive — drift-guard single-sourcing, presence-not-value, BDD step-fidelity), `.score/memory/DEPRECATION.md` (no relevant entries for 066)

---

## System Architecture

Tension Processing Path is the second **operator path** on the Agent Operating Surface, the write-side counterpart to the read-only Governance Navigation Path (064). Like 064 it adds no Go CLI code and no API capability — it is a pair of declarative plugin artifacts, hand-authored and committed under the existing top-level `plugin/` tree (the home 062 ADR-1 established; the `plugin/agents/` type 064 ADR-2 added), plus one best-effort Go drift-guard test that anchors the artifacts to the shipped command surface.

The path is delivered in **two cooperating pieces**, mirroring 064:

- **A thin skill** — `plugin/skills/tension-processing/SKILL.md`. The discoverable, description-triggered entry point (callable by an agent *or* a human user). It carries the *when* (reach for tension processing to turn a voiced tension into a well-formed record) and the *workflow* (voiced tension → situate against the tensions already sensed on the role and rolled up from its sub-roles → capture on the sensing role → refine or retire → hand the `ten_` id to the Proposal Drafting Path 067). It does not run the command chain in the caller's context; it delegates execution to the tension-processor subagent.

- **A tension-processor subagent** — `plugin/agents/tension-processor.md`. This runs the workflow in its **own isolated context**, then returns only the drawn-together tension record to the caller. Isolation is what structurally delivers the spec's "synthesized record rather than raw dumps" promise: the situating reads (existing tensions + sub-role roll-up) stay in the subagent's context and never flood the caller's. Unlike 064's read-only navigator, this subagent **writes** — it must run `glassfrog tension create/update/discard` — so it is *not* read-only. Its boundary is enforced in layers: the tool grant withholds `Write`/`Edit` (no *workspace* mutation), the prompt scopes execution to the operational *tension* leaves only, and it must never reach a proposal-write command — 063's `PreToolUse` hook is the backstop that would gate any proposal write regardless.

**Flow**: caller voices a tension → skill triggers (by description) and describes the workflow → skill delegates to the `tension-processor` subagent → subagent composes the shipped tension commands (Tension Reads `tension list`/`tension get`, Subroles Roll-up `tension subroles`, Tension Capture `tension create`, Tension Update `tension update`, Tension Discard `tension discard`) in its own context → subagent returns the synthesized tension record, each element carrying the `ten_`/role id that reads it again or feeds 067 → caller receives the record, ready to hand onward, not the raw dumps.

Everything the two artifacts name is behaviour the CLI already exposes. The path extends `plugin/` additively — a sibling skill and a sibling agent — exactly as 062 ADR-2 anticipated for #64–#69, not a restructure of what 062/064 shipped.

---

## Architecture Decisions

### ADR-1: Deliver the path as a thin skill delegating to a tension-processor subagent (both, mirroring 064)

**Context**: The spec deferred the delivery form to shaping (spec Assumption 1: "a plugin skill, a tension-processing agent, or both"). 064 established the first operator path as a thin skill + subagent (064 ADR-1), diverging from 062 ADR-2's "sibling skills" projection because a synthesizing path is better served by an isolated executor. 066 shares 064's "drawn-together record, not raw dumps" promise. The developer confirmed mirroring 064's both-form.

**Options considered**:
1. **Skill only** — a `plugin/skills/` sibling. Discoverable and callable, but the situating reads and writes execute in the *caller's* context, so "synthesized not raw" rests on caller discipline, not structure.
2. **Subagent only** — a `plugin/agents/` executor. Structurally delivers synthesis-via-isolation but is not a natural human-callable, description-triggered entry point, and the "when + workflow" knowledge has no home.
3. **Both — thin skill + subagent** — the skill owns *when* and the *workflow steps* and is the discoverable entry point; it delegates execution to the subagent, which owns isolated execution and returns the synthesized record.

**Decision**: Option 3 — both, mirroring 064. The skill's frontmatter `description` triggers whenever a caller needs to work a tension into a record; its body describes the workflow and delegates to the `tension-processor` subagent. The subagent runs the workflow to completion in isolation and returns the drawn-together record. The two share **one** workflow definition — the skill *describes* it and points at the agent; the agent *executes* it — a single source of truth for the steps, kept from drifting by writing the steps once in the skill and having the agent reference that workflow rather than restating it.

**Consequences**: Positive — discoverability (skill) + context isolation (agent); the raw command output never reaches the caller. Negative — two artifacts to keep coherent (mitigated by single-sourcing the workflow in the skill). This is **silent conformance** to 064 ADR-1's both-form pattern (064 already carried the divergence from 062 ADR-2; 066 follows the established path pattern, it does not re-diverge). The surface now has a second consumer of the `plugin/agents/` type — evidence the #65–#69 paths can share it rather than each minting a bespoke shape.

### ADR-2: Home the artifacts additively — reuse `plugin/skills/` and the `plugin/agents/` type; no restructure

**Context**: 062 ADR-1 established `plugin/` as the surface's home and ADR-2 committed to additive growth; 064 ADR-2 added `plugin/agents/` as a component type. 066 needs a home for a second skill and a second agent.

**Options considered**:
1. **`plugin/skills/tension-processing/SKILL.md` + `plugin/agents/tension-processor.md`** — conventional sibling locations, reusing the types 062/064 established. Keeps the additive-growth contract.
2. **Fold the agent into the skill directory** — breaks the plugin's conventional component layout and hides the agent from the host's agent discovery (rejected on the same grounds as 064 ADR-2 option 2).

**Decision**: Option 1 — sibling skill and sibling agent, registered per the plugin's conventions. Silent conformance to 062 ADR-1/ADR-2 and 064 ADR-2.

**Consequences**: No existing artifact is moved or rewritten. The agent is discovered by directory convention from `plugin/agents/` — no `plugin.json` `agents` key is needed (063's landed `hooks.json` and 064 both confirmed this plugin uses directory auto-discovery). Exact manifest/frontmatter schema is the interface skill's concern.

### ADR-3: The subagent is write-capable but fenced to operational tension leaves — the material divergence from 064's read-only navigator

**Context**: 066 is a *write* path — the subagent must run `glassfrog tension create/update/discard`. 063 deliberately leaves operational tension edits **ungated** (063's Behavioral Accord: "an operational tension edit … the guardrail does not gate it"), so these writes execute without confirmation — correctly. But the subagent must never cross into a *proposal* write (that is 067/068, gated by 063). 064 ADR-5 made its navigator read-only via a `Write`/`Edit`-withheld grant plus a prompt scoped to read leaves; 066 cannot be read-only without defeating its purpose.

**Options considered**:
1. **Withhold `Write`/`Edit`; grant `Bash`; prompt-scope to the tension leaves; rely on 063's hook as the proposal-write backstop** — no workspace mutation possible; the ungated tension writes execute; the proposal boundary is layered (prompt scope + 063 hook).
2. **Read-only agent that only *proposes* the write commands for the caller to run** — keeps the agent read-only but pushes execution back to the caller, losing the isolation benefit and making the path indistinguishable from the caller running the commands itself.
3. **Unrestricted `Bash`, prompt guidance only** — weakest; nothing structurally prevents a workspace write.

**Decision**: Option 1. The tool grant withholds `Write`/`Edit` (no *workspace* mutation); `Bash` is permitted for the operational tension commands; the prompt scopes execution to `tension create/update/discard` plus the situating reads (`tension list`/`get`/`subroles`) and forbids any `proposal …` command; 063's `PreToolUse` hook gates any proposal write the subagent might erroneously attempt.

**Consequences**: Workspace mutation is structurally impossible; the operational tension writes run ungated (the correct behaviour per 063); the proposal boundary holds *in layers* (prompt scope + 063 hook), not from the tool grant alone — a tool grant cannot restrict which `glassfrog` subcommands `Bash` runs. This is an **announced divergence** from 064 ADR-5: 064's path is read-only so it withholds all write capability; 066 is a write path, so "read-only" is replaced by "only the ungated tension writes, never a proposal write." The exact tool-grant syntax is interface-level.

### ADR-4: Compose only already-shipped tension commands; add no CLI code beyond a drift guard

**Context**: The "knowledge + guardrails, never capability" constraint (PROJECT) and the spec's non-behaviors forbid any new command, flag, or local governance logic. Every command this path composes is shipped: Tension Capture (042), Tension Reads (043), Subroles Tension Roll-up (046), Tension Update (044), Tension Discard (045).

**Options considered**:
1. **Pure composition of shipped commands** — the skill/agent drive existing `glassfrog tension …` commands only; the CLI is unchanged.
2. **Add a convenience `tension process` command in the CLI** — rejected on sight: it would add API surface, breaking "Bounded by the API surface" and VISION Exclusion 2, and duplicate composition logic the operator layer owns.

**Decision**: Option 1 — pure composition. The only Go change is a test (ADR-5); no production CLI code changes. Silent conformance to 064 ADR-3 and 062 ADR-3 (per-command flags are never restated — the artifacts defer to `glassfrog tension <sub> --help`).

**Consequences**: The path stays a faithful guide over the shipped surface; the drift surface is the *set of tension command leaves named*, not their flags.

### ADR-5: Guard artifact/CLI drift with a best-effort `internal/build` test pinning the composed tension leaves — single-sourced, and asserting the ungated invariant against 063

**Context**: 062/063/064 each anchor a plugin artifact's enumerable facts to their CLI source with a best-effort `internal/build` config-guard test, and LEARNINGS mandates single-sourcing an enumerable list (063's `gated-commands.txt` feeds both its hook and its guard) and that reduced coverage be stated, not silent. 066's artifacts name a set of tension command leaves; if the CLI renames or drops one, the artifacts would silently drift. Additionally, 066's ungated-writes design depends on those leaves being **absent** from 063's gated proposal-write set — if a tension leaf ever entered 063's registry, 066's writes would start prompting (or the reverse: a proposal leaf leaking into 066's set would be wrongly executed by the subagent).

**Options considered**:
1. **Best-effort `internal/build` drift test, single-sourced, with an ungated-invariant assertion** — assert every tension leaf the skill/agent name (`tension list`, `tension get`, `tension subroles`, `tension create`, `tension update`, `tension discard`) exists in the CLI's command registry, single-source that list in one file consumed by both the artifacts' reference and the guard, and additionally assert those leaves are **not** members of 063's `gated-commands.txt`. Pins the surface *and* the ungated invariant.
2. **Existence-only guard, no 063 cross-check** — cheaper, but leaves the ungated invariant (the crux of ADR-3's correctness) unguarded, exactly the silent-drift failure this surface is prone to.
3. **No automated guard** — rely on review; rejected, drift is the failure mode 062/063/064 all guarded.

**Decision**: Option 1 — a best-effort `internal/build` test pinning the named tension leaves against the shipped command registry and asserting they are disjoint from 063's gated proposal-write set, with the composed-leaf list single-sourced in one file, mirroring 063/064.

Coverage is explicitly partial: it pins the *existence* of the tension leaves the artifacts compose and their *absence* from 063's gated set — not their flags (deferred to `--help`), not the synthesized-record prose, not parser robustness. The partial scope is stated in the test and this plan (no silent caps).

**Consequences**: A renamed/removed tension command fails the build until the artifacts are updated; a future change that pulls a tension leaf into 063's gated set (or a proposal leaf into 066's composed set) fails the build, protecting the ungated-writes design. Establishes that operator write-paths carry a drift tripwire for both the leaves they name *and* their gate-membership invariant. The exact single-source file format is interface-level.

---

## Cross-cutting Concerns

**Failure handling**: The subagent processes defensively — a single failed situating read (e.g. the sub-role roll-up errors) does not abort the record; the agent reports what failed and returns the record assembled from the reads that succeeded (spec error scenarios). A capture rejected as a usage or API error (unknown sensing role, blank body) is surfaced by name and fabricates no `ten_` id. The agent relies on the orientation skill (062) for how to read exit codes and page results — it does not restate that knowledge (single-sourcing).

**Proposal boundary**: The subagent never runs a proposal-write command; handoff to the Proposal Drafting Path (067) is *returning the ready `ten_` id*, not executing a draft. If the subagent ever attempted a proposal write, 063's `PreToolUse` hook would gate it — the layered backstop behind ADR-3's prompt scope. Authority judgment ("does this need a proposal?") is likewise out of scope (065), enforced at the prompt level.

**No new observability/config**: This path adds no logging, config, or state — it composes commands. The only "configuration" is the subagent's tool grant declared in its frontmatter and the single-source leaf-list file (ADR-5).

**Testing strategy**: (1) the best-effort `internal/build` drift guard (ADR-5) pinning the composed tension leaves and the ungated-invariant cross-check against 063; (2) BDD scenarios (Tier 2) exercising the spec's driving/validation scenarios against the artifacts' described behaviour — voiced-tension→capture, situating against existing/rolled-up tensions, refine, capture-rejected, partial situating failure, duplicate-already-sensed, ready-for-governance hand-off to 067, and moot→discard. Because the artifacts are declarative prose, scenario coverage verifies described behaviour and the drift guard verifies the named surface and the gate invariant; there is no runtime Go path unique to 066 beyond the guard.

---

## Implementation Strategy

Single phase — one coherent unit, mirroring 064:

1. **Author the skill and agent, add the drift guard.** Write `plugin/skills/tension-processing/SKILL.md` (when + workflow, delegating to the agent), write `plugin/agents/tension-processor.md` (write-capable-but-fenced grant + isolated execution returning the synthesized tension record), single-source the composed-leaf list, and add the best-effort `internal/build` drift test (leaf existence + disjoint-from-063 assertion). The agent is discovered by directory convention from `plugin/agents/` — no `plugin.json` edit is needed. These land together because the drift test pins the leaves the artifacts name, and the agent must be present under `plugin/agents/` for its delegation to resolve.

The tasks skill may split this into PR-sized units (e.g. artifacts + registration in one, drift guard in another), but there is no cross-dependency forcing an order beyond "the artifacts must name the leaves the guard pins."

---

## Risks

- **Skill/agent drift from each other** (medium likelihood, low impact): two artifacts describing one workflow can diverge. *Mitigation*: single-source the workflow steps in the skill and have the agent reference them (ADR-1); the drift guard does not cover this, so review must.
- **063's write hook may not cover the subagent's Bash** (low likelihood, medium impact): 063 landed as a `PreToolUse` matcher on `Bash`; if the host does not apply it to a *subagent's* Bash calls, the proposal-write fence rests on the prompt scope alone. This matters more for 066 than for read-only 064, because 066's subagent legitimately runs writes. *Mitigation*: T001 confirms subagent hook coverage against the target host and keeps the subagent's prompt strictly scoped to the tension leaves regardless; the ungated tension writes pass the gate either way.
- **Ungated-invariant coupling with 063** (low likelihood, medium impact): 066's ungated-writes correctness depends on its tension leaves staying out of 063's gated set. A future edit to either registry could silently break it. *Mitigation*: ADR-5's drift guard asserts the two sets are disjoint.
- **Host doesn't support subagents or tool grants** (low likelihood, medium impact): isolation and the withheld-`Write`/`Edit` grant depend on a host that honours agent definitions (external contract, like 064 R2). *Mitigation*: the skill remains useful as guidance where the agent can't run isolated; #70 owns getting the surface into a capable host.
- **Boundary bleed into proposal drafting (067) or authority (065)** (medium likelihood, low impact): once a tension is well-formed it is tempting to draft the proposal or rule on whether one is needed. *Mitigation*: ADR-3's prompt-level fence, spec non-behaviors, and a validation scenario asserting no proposal write and no authority verdict appear.

---

## What This Plan Does Not Cover

- **Protocol-level detail** — the skill/agent frontmatter schemas (name/description/tools), the precise write-capable-but-fenced tool-grant syntax, the single-source leaf-list file format, and the shape of the synthesized tension record the agent returns. These are the **interface** skill's concern (`interface-spec.md`).
- **Executable scenarios** — the Gherkin realizing the spec's driving/validation scenarios is the **scenarios** skill's concern.
- **Task decomposition** — PR-sized units and their acceptance criteria are the **tasks** skill's concern.
- **Distribution** — how the plugin (now carrying a second agent) is published and installed remains Operating-Surface Packaging (#70); 066 only adds artifacts to `plugin/`.
- **Reuse of the write subagent by 067/068** — the developer noted that Proposal Drafting (067) and Proposal Circulation (068), also write-paths, may reuse this subagent rather than each minting their own ("an agent per skill would be overkill"), but confirmed that is not yet decided. 066 scopes `tension-processor` to tension processing and does **not** pre-generalize it into a shared write executor; whether 067/068 extend or reuse it is their spec's decision.
