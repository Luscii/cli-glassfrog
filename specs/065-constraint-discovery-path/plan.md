# Plan: Constraint Discovery Path

**Feature**: 065-constraint-discovery-path
**Role**: Shaper
**Inputs**: `specs/065-constraint-discovery-path/spec.md`, PROJECT.md, `.score/memory/DECISIONS.md` (loaded — precedent from 062/063 and, decisively, 064 which set the read-path skill+agent pattern and pre-stated the 064→065 boundary), `.score/memory/LEARNINGS.md` (passive — no-silent-caps), `.score/memory/DEPRECATION.md` (no relevant entries for 065)

---

## System Architecture

Constraint Discovery Path is the second **operator path** on the Agent Operating Surface and the structural twin of the Governance Navigation Path (064). It adds no Go CLI code and no API capability — it is a pair of declarative plugin artifacts, hand-authored and committed under the existing top-level `plugin/` tree (the home 062 ADR-1 established, the `plugin/agents/` subtree 064 ADR-2 added), plus one best-effort Go drift-guard test that anchors the artifacts to the shipped read surface.

The path is delivered in **two cooperating pieces**, reusing the pattern 064 established (DECISIONS: "sets the skill+agent delivery pattern the other read paths (#65) may reuse"):

- **A thin skill** — `plugin/skills/constraint-discovery/SKILL.md`. The discoverable, description-triggered entry point (callable by an agent *or* a human user). It carries the *when* (reach for constraint discovery when you want to know whether an action is within your authority, falls under another role's domain, is shaped by a policy, or needs a proposal) and the *workflow* (take a free-form wanted action → if it is too vague to search, sharpen it with the operator → search to discover the domains and policies it touches → read the owning role and its domains/policies → characterize the authority situation drawn from the record → synthesize a picture, not raw dumps). The skill owns the one interactive step 064 did not have — the **clarify-when-vague** exchange (ADR-3) — then delegates the isolated read traversal to the navigator subagent.

- **A read-only subagent** — `plugin/agents/constraint-navigator.md`. Runs the traversal in its **own isolated context** with a **read-only tool grant**, then returns only the synthesized picture to the caller. Isolation delivers the spec's "synthesized picture rather than raw dumps" promise structurally: the raw search/role/domain/policy output stays in the subagent's context and never floods the caller's. The read-only tool grant withholds `Write`/`Edit`, so the subagent cannot mutate the workspace; the agent runs non-interactively (like 064's navigator). Its defining guardrail — **surface and characterize, never locally rule** (ADR-4) — is prompt-level, because reading a domain or policy is legitimate and cannot be tool-restricted.

**Flow**: caller has an action a practitioner wants to take → skill triggers (by description) and describes the workflow → if the action is too vague to locate its governing governance, the skill asks the operator to sharpen it (caller context) → skill delegates the well-formed action to the `constraint-navigator` subagent → subagent composes the shipped reads (Cross-Model Search `search`, Role Reads `roles [id]`/`tree`, Role Domains `domains`, Role Policies `policies`/`policy`, and the caller's own roles via `me roles`) in its own context → subagent returns the synthesized picture: the governing domains and binding policies, the role(s) that own them, each element carrying the id that reads it again, and a characterization of the authority situation (within your own authority / falls under another role's domain / shaped by a policy / a governance change that needs a proposal / the record does not clearly answer) → caller receives the picture, not the dumps.

Everything the two artifacts name is behaviour the CLI already exposes; `plugin/agents/` and `plugin/skills/` are extended by adding sibling files, exactly the additive growth 062 ADR-2 / 064 ADR-2 anticipated for #65–#69 — not a restructure.

---

## Architecture Decisions

### ADR-1: Deliver the path as a thin skill delegating to a read-only navigator subagent — reuse 064's pattern

**Context**: The spec deferred the delivery form to shaping (spec Assumption 1, "delivery form deferred to shaping"). 064 resolved the same deferral for the first read path by choosing a thin skill + read-only subagent, and its DECISIONS entry explicitly names 065 as a reuser of that pattern. 065's core promise ("a synthesized picture … not … a concatenation of … reads") and central non-behaviors ("must not write", "must not … rule on it") are the same shape as 064's.

**Options considered**:
1. **Skill only** — a `plugin/skills/` sibling. Discoverable and callable, but the raw reads execute in the *caller's* context, so "synthesized not raw" rests on caller discipline, not structure, and read-only rests on the caller not choosing to write.
2. **Subagent only** — a `plugin/agents/` read-only agent. Structurally delivers synthesis-via-isolation and read-only-via-tool-grant, but is a poor home for the *when*-to-reach-for-it and *workflow* knowledge and is not a natural human-callable, description-triggered entry point.
3. **Both — thin skill + subagent** — the skill owns *when* + the *workflow* and is the discoverable entry point; it delegates isolated execution to the subagent, which returns the synthesized picture.

**Decision**: Option 3 — both, conforming to 064 ADR-1. The skill (`plugin/skills/constraint-discovery/SKILL.md`) is the discoverable, callable surface carrying the when/workflow knowledge and the clarify-when-vague step; the subagent (`plugin/agents/constraint-navigator.md`) is the isolated, read-only, non-interactive executor that returns the synthesized picture. The two share one workflow definition — the skill *describes* it and points at the agent; the agent *executes* it — single-sourcing the steps in the skill so they cannot drift (review-guarded, as in 064). The `plugin/agents/` home is silent conformance to 064 ADR-2; the agent is discovered by directory convention (063's landed `hooks.json` and 064's navigator confirm this plugin auto-discovers, so no `plugin.json` component-array edit is required — an interface-level detail).

**Consequences**: Discoverability (skill) + context isolation and structural workspace read-only (agent), matching 064. This is conformance to a precedent 064 set for exactly this spec, not a new divergence — so it is recorded (it resolves a spec-deferred question) but claims no novelty. Negative — two artifacts to keep coherent (mitigated by single-sourcing the workflow in the skill). The one place 065 departs from 064 is the interactive clarify step (ADR-3), which is why the skill, not only the agent, carries workflow responsibility.

### ADR-2: Compose only already-shipped read commands; add no CLI code beyond a drift guard

**Context**: The "knowledge + guardrails, never capability" constraint (PROJECT) and the spec's non-behaviors forbid any new command, flag, or local governance logic. The reads this path composes are all shipped: Cross-Model Search (041) `search`, Role Reads (025) `roles`/`tree`, Role Domains (033) `domains`, Role Policies (034) `policies`/`policy`, and the self-service read `me roles` (the caller's own roles). `me roles` is what lets the characterization tell "your own authority" from "another role's domain" per the spec's Characterization accord — it surfaces the caller's own assignments from the record, not a computed verdict. (Role Fillers is deliberately *not* composed — 065's FEATURE-MODEL dependency list omits it; the picture names the owning role with its id, and reaching *who fills it* is a bridge into the CLI's own `fillers` command or the Governance Navigation Path (064), not part of this path's traversal. The FEATURE-MODEL's coarse dep list names only search/domains/policies; `roles`/`tree`/`me roles` are the shipped reads the spec's owning-role and own-vs-other behaviour additionally require — surfaced facts, not new capability.)

**Options considered**:
1. **Pure composition of shipped reads** — the skill/agent drive existing `glassfrog` commands only; the CLI is unchanged. Honors the constraint; zero API surface added.
2. **Add a convenience `constraints`/`authority` command in the CLI** — one command that does the traversal. Rejected on sight: it would add API surface (breaking "Bounded by the API surface" and VISION Exclusion 2) and, worse for this path, invite reimplementing permission logic locally — the exact thing the spec and VISION Exclusion 2 forbid.

**Decision**: Option 1 — pure composition (silent conformance to 064 ADR-3). The only Go change 065 makes is a test: a best-effort `internal/build` drift guard asserting every command leaf the artifacts name — `search`, `roles`, `tree`, `domains`, `policies`, `policy`, and `me roles` — exists in the CLI's command registry, mirroring 062 ADR-4 / 063 ADR-4 / 064 ADR-4. The leaf list is single-sourced (one machine-readable list the artifact and the drift test both consume, per 063/064) rather than duplicated in prose and test. Coverage is explicitly partial: it pins the *existence* of the named leaves, not their flags (deferred to `glassfrog <command> --help`, silent conformance to 062 ADR-3), not the synthesized-picture prose, and not parser robustness. The partial scope is stated in the test and plan — no silent caps (LEARNINGS).

**Consequences**: The path stays a faithful guide over the shipped surface; a renamed/removed read command fails the build until the artifacts are updated. `domains`, `policies`, and `policy` are **top-level** commands (`domains <id>`, `policies <role-id>`, `policy <pol-id>`) — not `roles` subcommands and there is no `roles get`; the drift guard pins the real leaves, avoiding the invented-leaf trap. Per-command flags are never restated.

### ADR-3: The clarify-when-vague step lives in the skill (caller context); the agent stays non-interactive

**Context**: The spec adds a behavior 064 never had: when a wanted action is too vague to locate the governance that would constrain it, the path asks the operator a clarifying question rather than guessing (spec Behavioral Accord "Entry"; developer preference for a structured ask-the-operator mechanism). But ADR-1's traversal runs inside an *isolated subagent*, and a subagent generally cannot prompt the human operator — asking mid-traversal has no reliable channel back to the caller.

**Options considered**:
1. **Clarify in the skill, before delegating** — the skill (which runs in the caller's context, where operator interaction is possible) detects a too-vague action and asks the operator to sharpen it, then delegates the *well-formed* action to the agent for the isolated traversal. The agent stays non-interactive, exactly like 064's navigator.
2. **Clarify inside the agent** — the subagent asks the operator mid-traversal. Rejected: an isolated subagent has no dependable interactive channel to the human; the question would either be swallowed or force the agent out of isolation, dissolving the synthesis-via-isolation guarantee ADR-1 rests on.

**Decision**: Option 1 — the skill owns the clarify-when-vague exchange in the caller's context and only delegates a searchable, well-formed action to the agent; the agent performs the read traversal non-interactively and returns the picture. The preferred interaction mechanism (an `ask_user_question`-style structured prompt) is a delivery detail for the interface skill; the plan fixes only that the *skill*, not the agent, asks, and that it asks rather than guesses.

**Consequences**: The interaction is placed where a channel to the operator actually exists, and the agent keeps 064's clean isolated-and-non-interactive shape. This is the one genuine architectural divergence from 064's navigator, and it slightly widens the skill's role (it carries a pre-traversal branch, not just when/workflow). It also sets a reusable pattern for later operator paths that need operator input: *interaction lives in the skill; isolated execution lives in the agent.* A vague action that the operator declines to sharpen ends at the skill with no traversal — no fabricated action is sent to the agent.

### ADR-4: Surface and characterize, never locally rule; fail toward surfacing under uncertainty — a prompt-level guarantee

**Context**: 065 is the destination of the authority hand-off 064 ADR-5 named ("the boundary with Constraint Discovery (065) — surface domains/policies, do NOT judge authority — … hands 'am I allowed to do X?' to 065"). Yet VISION Exclusion 2 and PROJECT's "never reimplements governance logic locally" forbid computing a permission verdict from local rules, and the spec's clarification reconciles this: 065 *surfaces* the governing domains and policies scoped to the action and *characterizes* the situation drawn from the record (own authority / another role's domain / policy-shaped / needs a proposal), but it does not compute an allow/deny verdict from its own logic, and when the record does not clearly answer it says so and surfaces what it found rather than fabricating a ruling. Like 064's surface-not-judge boundary, this cannot be tool-enforced — reading a policy is a legitimate read for any path.

**Options considered**:
1. **Prompt-level surface-and-characterize guardrail, fail-toward-surfacing** — the skill/agent prose instructs the navigator to draw the governing domains/policies together and name the authority situation *as read from the record*, to never compute a local permission verdict, and — under uncertainty — to state what is unclear and surface what it found rather than guess. Read-only is layered exactly as 064 (Write/Edit withheld → no workspace mutation; the path is read-only so it drives no `glassfrog` write, and 063's PreToolUse hook remains a backstop for any write issued via Bash when present).
2. **Encode a local permission model** — teach the artifacts Holacracy's authority rules and compute a verdict. Rejected outright: this is precisely VISION Exclusion 2 (Local governance logic) and the spec's central non-behavior; it would drift from the record the API is the source of truth for.

**Decision**: Option 1 — a prompt-level guarantee. The agent surfaces and characterizes from the record and never rules locally; under uncertainty it surfaces-and-says-so rather than fabricating a verdict. Read-only layering conforms to 064 ADR-5 (Write/Edit withheld; prompt scoped to read leaves; 063 hook as a backstop), though 065 being read-only rarely exercises the write backstop.

**Consequences**: 065 answers the question 064 hands it *without* becoming a local rules engine — the distinguishing discipline of this path. The guarantee is prose, so it is verified by a held-out validation scenario (no permission verdict computed from local logic; no fabricated ruling under uncertainty), not by a tool grant. Workspace mutation is structurally impossible (no Write/Edit). The boundary with 064 is the mirror image of 064's: 064 surfaces the governance around a *concern*; 065 scopes to a wanted *action* and surfaces what constrains it — neither rules, and neither reimplements the other's traversal.

---

## Cross-cutting Concerns

**Failure handling**: The subagent traverses defensively — a single failed read (e.g. a `policies` read errors) does not abort the picture; the agent reports what failed and returns the picture assembled from the reads that succeeded (spec error scenario). An ambiguous record — no domain plainly owns the action, or conflicting partial matches — yields an explicit "the record does not clearly answer" with what was found, never a fabricated authority ruling (spec error scenario + ADR-4). A search that matches nothing yields an explicit "nothing found, refine the action" rather than an invented picture. The agent relies on the orientation skill (062) for how to read exit codes and page results — it does not restate that knowledge (single-sourcing); the pagination-before-narrowing behavior (spec) is described, not reimplemented.

**No new observability/config**: This path adds no logging, config, or state — it composes reads. The only "configuration" is the read-only tool grant declared in the agent's frontmatter and the single-sourced composed-leaf list the drift guard consumes.

**Testing strategy**: (1) the best-effort `internal/build` drift guard (ADR-2) pinning the composed command leaves; (2) BDD scenarios (Tier 2) exercising the spec's driving scenarios against the artifacts' described behaviour — action-under-another-role's-domain, policy-shaped action, nothing-constrains-it, read-failure partial picture, record-does-not-clearly-answer, too-vague-so-ask, and over-broad narrowing — plus the held-out validation scenarios (no invented surface, read-only, surface-not-rule, no-fabricated-ruling, synthesized-not-raw). Because the artifacts are declarative prose, scenario coverage verifies described behaviour and the drift guard verifies the named surface exists; there is no runtime Go path unique to 065 beyond the guard.

---

## Implementation Strategy

Single phase — the feature is one coherent unit:

1. **Author the skill and agent, add the drift guard.** Write `plugin/skills/constraint-discovery/SKILL.md` (when + workflow + the clarify-when-vague step, delegating to the agent), write `plugin/agents/constraint-navigator.md` (read-only grant + isolated, non-interactive execution returning the synthesized picture with its characterization), single-source the composed-leaf list, and add the best-effort `internal/build` drift test. The agent is discovered by directory convention from `plugin/agents/` — no `plugin.json` component-array edit is needed (064's navigator and 063's `hooks.json` confirm auto-discovery). These land together because the drift test pins the command leaves the artifacts name and the agent must be present under `plugin/agents/` for the skill's delegation to resolve.

The tasks skill may still split this into PR-sized units (e.g. artifacts in one, drift guard in another), but there is no cross-dependency forcing an order beyond "the artifacts must name the leaves the guard pins."

---

## Risks

- **Boundary bleed into the record-as-rules-engine** (medium likelihood, medium impact): because 065 reads domains/policies to characterize authority, it is tempting to slide from *surfacing* the governing governance into *computing* a permission verdict from local Holacracy rules — exactly VISION Exclusion 2. *Mitigation*: ADR-4's prompt-level guardrail, a spec non-behavior, and a held-out validation scenario asserting no locally-computed verdict and no fabricated ruling under uncertainty.
- **Skill/agent drift from each other** (medium likelihood, low impact): two artifacts describing one workflow can diverge — more so than 064, since the skill now also owns the clarify branch. *Mitigation*: single-source the workflow steps in the skill and have the agent reference them (ADR-1); the clarify branch lives only in the skill (ADR-3), so it has no agent counterpart to drift against; review covers the prose the drift guard does not.
- **A subagent cannot ask the operator** (addressed by design): 064's navigator never needed to; 065 does. *Mitigation*: ADR-3 places the clarify step in the skill (caller context) so the isolated agent stays non-interactive — the risk is designed out rather than mitigated at runtime.
- **Host doesn't support subagents or read-only tool grants** (low likelihood, medium impact): isolation and structural workspace read-only depend on a plugin host that honours agent definitions and tool restrictions (external contract, like 062 R2 / 064). *Mitigation*: the skill remains useful as guidance even where the agent can't run isolated; #70 owns getting the surface installed in a capable host.
- **Artifact/CLI drift** (low likelihood, medium impact): a renamed/removed read leaf would silently invalidate the artifacts. *Mitigation*: the ADR-2 drift guard fails the build on any change to the named leaves; coverage is stated-partial (existence, not flags/prose).
- **Traversal-depth judgment** (low likelihood, low impact): "relevance-bounded" depth (spec Assumption) is judgment, not an algorithm. *Mitigation*: the workflow describes narrowing over dumping; the over-broad-action scenario pins the expected narrowing behaviour.

---

## What This Plan Does Not Cover

- **Protocol-level detail** — the skill and agent frontmatter schemas (name/description/tools), the precise read-only tool-grant syntax, the exact structured ask-the-operator mechanism for the clarify step, the single-sourced leaf-list file format, and the shape of the synthesized picture (fields, how the characterization is rendered). These are the **interface** skill's concern (`interface-spec.md`).
- **Executable scenarios** — the Gherkin realizing the spec's driving/validation scenarios is the **scenarios** skill's concern.
- **Task decomposition** — PR-sized units and their acceptance criteria are the **tasks** skill's concern.
- **Distribution** — how the plugin (carrying a second agent) is published and installed remains Operating-Surface Packaging (#70); 065 only adds artifacts to `plugin/`.
- **The other operator paths (#66–#69)** — 065 reuses 064's skill+agent pattern and adds the clarify-in-skill pattern; the write paths (#66–#69) still route through 063's guardrail and are separate specs.
