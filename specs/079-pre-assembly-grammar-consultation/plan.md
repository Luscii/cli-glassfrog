# Plan: Pre-Assembly Grammar Consultation

**Feature**: 079-pre-assembly-grammar-consultation
**Role**: Shaper
**Inputs**: spec.md, PROJECT.md, `.score/memory/DECISIONS.md` (172 entries), `.score/memory/LEARNINGS.md`, `.score/memory/DEPRECATION.md` (12 entries), the landed drafting-path artifacts (`plugin/skills/proposal-drafting/SKILL.md`, `plugin/agents/proposal-drafter.md`, `plugin/agents/proposal-drafting-commands.txt`), both reference records under `plugin/skills/proposal-drafting/references/`, the landed grammar command surface (`internal/cli/proposal_grammar.go`, `PROPOSAL_READS` in `plugin/hooks/glassfrog-write-gate.sh`, `expectedProposalSurface`), and the guards `internal/build/proposaldrafting.go` / `internal/build/circleroutingrule.go`. No SOUL.md at the project root.

---

## System Architecture

This feature is a **declarative operating-surface change**: it adds no Go CLI code, no command, no flag. Every behavioral requirement lands as edits to the three drafting-path artifacts the plugin already ships, plus the executable expectations that keep those artifacts truthful. The CLI side of the gate — the `proposal grammar` read, its ungated classification in the write-gate's `PROPOSAL_READS`, its membership in `expectedProposalSurface` — landed with the grammar-reference feature and is consumed here unchanged.

**The components and what changes in each:**

| Component | Today | After 079 |
|---|---|---|
| `plugin/skills/proposal-drafting/SKILL.md` | Six-step workflow starting from a handed-in `ten_` id; consults nothing | The gate's ordered workflow: route → ground → situate → duplicate check → consult grammar → assemble → match → confirm & create → hand off, with the two-phase decision-point relay documented |
| `plugin/agents/proposal-drafter.md` | Fenced to seven leaves; routing reads present but annotated inert; six-value output record | Fenced to eight leaves (`proposal grammar` joins); routing reads are the routing step's reads; output record gains the `consultation` element; `action` vocabulary grows the decision-point returns; defensive entries for a failed grammar read, an incomplete routing walk, and proceed-past direction |
| `plugin/agents/proposal-drafting-commands.txt` | Seven leaves; routing block annotated "no workflow step consults … a later change adds the consultation" | Eight leaves; the routing block's annotation rewritten to describe the reads as the routing step's named reads — 079 *is* the later change the annotation promised |
| `internal/build` executable expectations | `CheckProposalDraftingDrift` (four-way leaf resolution, gate-membership posture, agent-names-every-leaf); 067's BDD content-inspection suite | The drift guard needs **no code change** (see ADR-3); 067's BDD suite keeps passing because its assertions are presence-based; a new BDD content-inspection suite binds 079's feature file |
| `features/` + guard bookkeeping | 073's guard feature carries a held `@validation` scenario asserting the record ships unconsulted | New feature file from 079's driving scenarios (directory named by the scenarios skill, by problem); the stale 073 scenario retired; re-validation addenda for 067 and 073 (ADR-4) |

**Control flow after the change** (the workflow the skill single-sources and the agent executes):

```
intent (+ optional handed-in ten_ id)
  │
  1. Route            me roles → tension list → roles (record's procedure, in its order)
  │                     → target circle + every eligible anchor
  │                     handed-in anchor lands elsewhere? → RETURN (two-phase, practitioner decides)
  │                     no anchor handed in?              → RETURN with the named anchors (choice is the operator's)
  2. Ground           tension get <settled anchor>
  3. Situate          proposal list --role-id <circle> --status draft  (full walk)
  4. Duplicate check  existing match? → RETURN surfaced-existing (unchanged from today)
  5. Consult          proposal grammar   (client-less read; failure → note the gap, continue)
  6. Assemble         changes[] (verbatim above the type floor, unchanged)
  7. Match            assembled set vs the rendered dead shapes, routing answer in hand
  │                     recognized? → RETURN (two-phase; proceed only on explicit direction)
  8. Create           proposal create <ten-id> --changes '<inline JSON>'  (the one gated write, unchanged)
  9. Hand off         draft record + consultation element → circulation
```

Every `RETURN` is the same non-interactive-agent shape the duplicate check already uses: the agent stops before the create, returns the drawn-together record with an `action` naming why, the skill relays it, and a re-delegation carries the practitioner's explicit direction (settled anchor, or proceed-past instruction) as input. Nothing in the loop refuses — a return is a report, and the re-delegated run acts on the direction it is given.

---

## Architecture Decisions

### ADR-1: The gate lands as an in-place widening of the drafting path's shipped artifacts, not a sibling skill or a new agent

**Context**: Every prior operator-path feature (governance navigation through impact review) arrived as a new sibling skill + agent. 079 is different in kind: its charter is wiring *inside* an existing path — the portfolio places the pre-assembly gate inside the Proposal Drafting Path, and the routing record's ownership was settled on exactly that ground ("the consuming pre-assembly gate lives inside the drafting path").

**Options considered**:
1. **New sibling skill/agent (`pre-assembly-consultation`)** — matches the additive-sibling projection for paths. But the gate is not a path: it has no independent entry, no independent handoff, and a second agent touching the drafting flow would fork the write workflow across two prompts — the drafter would still need editing to call it, so the sibling buys a second artifact set without removing the in-place edit.
2. **Edit the drafting path's skill + agent + registry in place** — the gate becomes workflow steps of the path that owns it. One workflow, one fence, one registry; the single-source discipline (workflow lives in the skill, agent references it) is preserved.

**Decision**: Option 2 — in-place widening. The precedent for editing 067's shipped artifacts is already set: the routing-rule feature added the three routing reads to this registry and this fence, explicitly "ahead of a workflow that uses them — a later change to this path adds the consultation." 079 is that later change. One-agent-per-path stands: no new agent is minted, and the `proposal-drafter` remains the only write-capable drafting executor.

**Consequences**: The edits touch validate-pinned surfaces of the drafting path (its agent fence and workflow), so the drafting path gets a re-validation addendum in the same change (the routing-rule feature did exactly this when it widened the same fence — ADR-4 carries the bookkeeping). The skill's and agent's frontmatter descriptions must be updated for the widened entry — they currently promise "given a well-formed anchor tension's ten_ id", which ceases to be the whole entry contract.

### ADR-2: Routing and recognition run inside the drafter agent, two-phase on every practitioner decision point (developer-confirmed)

**Context**: The spec has three practitioner decision points — a handed-in anchor that routes elsewhere, choosing among eligible anchors, and proceeding past a surfaced dead shape. Subagents in this surface are non-interactive by established precedent (the constraint-discovery feature placed its interactive step in the skill for exactly this reason): an agent cannot pause mid-run for the practitioner. And the routing reads produce exactly the raw payload bulk (own roles, tension lists, role resolution) the agent-isolation pattern exists to keep out of the caller's context.

**Options considered**:
1. **Two-phase inside the agent** — the agent runs the routing reads and the grammar consult in its isolated context; whenever a decision arises it stops before the create and returns the record with an `action` naming the decision needed; the skill relays and re-delegates with explicit direction. Extends the shipped `surfaced-existing` duplicate-check shape.
2. **Routing in the skill (caller context)** — the interactive precedent from constraint discovery; the agent receives a settled anchor. But the routing reads' raw output then lands in the caller's context, and the dead-shape decision point (which arises *after* assembly, inside the agent) still needs the two-phase return — so this option pays the context cost and still needs option 1's machinery.
3. **Split** — classification in the skill, reads in a one-off agent pass. Two delegation round-trips on every draft, for no isolation gain over option 1.

**Decision**: Option 1 — two-phase inside the agent (developer-confirmed). The `action` vocabulary grows three values alongside the existing four: **`surfaced-routing-mismatch`** (handed-in anchor lands the change outside the target circle; eligible anchors named), **`named-anchors`** (no anchor handed in, or none settled; every eligible anchor named, none chosen — choosing is the operator's act), and **`surfaced-dead-shape`** (the assembled set matches a recorded dead shape; fact handle, shape, and symptom carried). A re-delegation's input carries the practitioner's direction explicitly — the settled anchor `ten_` id, and/or the proceed-past instruction naming the surfaced fact — because each delegation is a fresh context; the skill documents this relay, and the re-delegated run repeats the gate from the top (the reads are cheap, the grammar read is free, and a stale determination is worse than a repeated one).

**Consequences**: The agent's output contract and defensive-drafting sections grow; the interface accord pins the new `action` values and the re-delegation input shape. A practitioner's decline after any return is an outcome, not an error — same as the declined gate confirmation today. The gate's never-withhold fence is structural: the only thing that stops a create is the absence of the practitioner's direction, and direction given is always acted on.

### ADR-3: `proposal grammar` joins the composed-leaf registry as a read; no guard code changes anywhere

**Context**: The drafting path may invoke only the leaves its registry names. The grammar read is not among them. Three executable expectations watch this surface: the drafting drift guard (leaf existence via four-way resolution, gate-membership posture, agent-names-every-leaf), the routing-rule guard (named reads present in the registry — one direction only, by design), and the write gate's own `PROPOSAL_READS` classification with its checked-in `expectedProposalSurface`.

**Options considered**:
1. **Add the leaf and extend the guards to know about "consultation reads"** — a new category in the registry grammar and the drift check. Invents a delimiter the one-directional routing check deliberately avoided, for no invariant gained.
2. **Add the leaf as an ordinary composed read; change no guard code** — `proposal grammar` resolves through the existing `proposal <sub>` arm of the four-way leaf resolution; the read-side gate-membership check already asserts every non-create leaf stays out of the gated set; check (d) already forces the agent fence to name it; the routing guard's one-directional record↔registry check is untouched by a registry line the record does not name.

**Decision**: Option 2 — ordinary read, zero guard code. Verified against the landed code: the leaf resolves (grammar is in the live proposal surface and in `expectedProposalSurface` since the grammar-reference feature), it is ungated (`PROPOSAL_READS=" list get grammar "` landed with that same feature, so consulting never prompts — the spec's write-gate boundary holds by construction), and it is absent from `gated-commands.txt`. The registry line carries a comment naming it the consultation read, in the same voice as the existing annotations, and the routing block's "no workflow step consults" annotation is rewritten — the capability-ahead-of-use debt the routing-rule feature knowingly took on is retired here, which is the annotation's own script.

**Consequences**: The gate-membership invariant is re-asserted by existing tests the moment the registry line lands (a failing state is unrepresentable without tripping the drift check). The one thing no existing test pins is the *annotation* flip — 079's own BDD content inspection pins it (the spec's "inert-reads note no longer misdescribes the path" validation scenario).

### ADR-4: Upstream truth sweep in the same change — retire the stale "ships unconsulted" validation scenario, amend the drafting-path spec's entry statements, and record both re-validation addenda

**Context**: 079 falsifies statements that other, landed features pinned as true. (a) The routing-rule guard feature file holds a `@validation @wip` scenario — "No workflow step consults the record or runs its reads to route … the drafting path's workflow will be unchanged" — whose premise ("Given this capability landed on its own") dissolves when 079 lands. (b) The drafting-path spec's Entry accord and its "entry is an existing tension id" assumption state the narrower contract 079 widens; 079's spec explicitly hands this amendment sweep to shaping. (c) The edits touch surfaces two prior validations pinned.

**Options considered**:
1. **Leave the stale artifacts** — the scenario's Given scopes it historically, and spec files are records of their time. But this repository's own history treats stale forward-pointers and falsified annotations as defects to sweep in the landing change (recorded repeatedly in LEARNINGS), and the scenario's text would sit beside a registry that now says the opposite.
2. **Sweep in the same change** — retire the scenario (delete, with a source comment noting the consultation landed and where); amend the drafting-path spec's Entry section and assumption with a dated note pointing at 079 (the same planning-driven-amendment shape the routing-rule feature used on its own spec); append re-validation addenda to the drafting path's and the routing rule's validate records.

**Decision**: Option 2 — sweep in the same change. Runner-safe by inspection: the guard suite runs with `~@wip`, no Go step binds the retired scenario, so deletion cannot break a test. The 067 amendment is a **dated superseding note, not a rewrite** — the original text stays legible as what was true before 079, and the note names what changed and which spec changed it (the cross-branch-anchor discipline: the note is self-contained, no dangling references).

**Consequences**: Three specs' artifacts move in 079's PR (079's own, plus addenda/notes in 067's and 073's). That is the honest shape — the alternative leaves the repository asserting a workflow that no longer exists. Portfolio files (FEATURE-MODEL, BACKLOG, ROADMAP status lines) are deliberately *not* swept here; status reconciliation is the prioritize stage's job.

---

## Integration Design

- **`proposal grammar` (consultation read)**: invoked once per delegation, before assembly. Client-less and credential-free, so the only failure mode is local (a corrupt install); on failure the agent notes "grammar not consulted" in the consultation element and continues — never a stop, never a retry loop. The agent asks for structured output the way the orientation knowledge already prescribes for machine consumption; the exact invocation phrasing lives in the interface accord.
- **Routing reads (`me roles`, `tension list`, `roles`)**: run in the record's procedure order, inside the agent. The completeness hedge the record prescribes (own-roles read does not paginate — an absence is an absence in what was read) is carried into the consultation element's routing answer verbatim in posture, not in copied text.
- **The routing record and the grammar rendering**: consulted, never restated. The skill and agent name the record and the command; neither reproduces a change type, a placement rule, a shape, or the routing rule (079's no-copy validation scenario pins this). The record's content contracts stay owned by their features — 079 changes neither file.
- **Write gate**: untouched. The create remains the one confirmed write; the three routing reads and the grammar read are all outside `gated-commands.txt` and (for the proposal-group leaves) inside `PROPOSAL_READS`, so consultation never prompts. No hook edit, no registry edit on the gate side.
- **Output contract (the drawn-together record)**: gains one element, **`consultation`**, present on every action path: whether the grammar was read (or the failure that prevented it), the routing determination's answer (target circle `role_` id, eligible anchor `ten_` ids, the completeness hedge when it applies, or the incompleteness note), and the dead-shape match result (the fact handle, or the explicit "no recorded shape matched"). Field-level pinning is the interface accord's job.
- **Operating-surface self-containment**: every new sentence in the plugin artifacts resolves in-surface or to the CLI — the self-containment check walks these files and fails the build on a development-repository reference, so the new prose is written under that ban from the start.

---

## Cross-cutting Concerns

**Error handling**: consultation failures degrade to reported gaps, never to stops (spec accord). The two-phase returns are outcomes with dedicated `action` values, not errors. The create's own failure handling (403/404/422 surfaced by name, nothing fabricated) is unchanged.

**Testing strategy**: three layers, all existing patterns. (1) BDD content inspection — a new `internal/build` godog suite binds 079's feature file and asserts the artifacts' load-bearing phrases: the workflow order, the two-phase action values, the consultation element, the rewritten annotations, the absence of restated record content; whitespace-normalized comparison per the established operator-path BDD discipline, `~@wip` filter. (2) The existing drift guards re-assert leaf existence and gate-membership the moment the registry changes — no new guard code (ADR-3). (3) Held-out `@validation` scenarios (unconditional-and-ordered consultation, nothing-withholds, no-copy, annotation-flip, result-legibility) stay `@wip` for the validate stage. Full `gofmt -l .` + `go test ./...` before push; the 067 suite must stay green against the edited artifacts — its assertions are presence-based, so the edits are additive where those pins reach.

**Configuration**: none. Nothing new is configurable; the gate is unconditional by spec.

**Observability**: the consultation element *is* the observability — the gate is falsifiable from the returned record, which is the spec's third user scenario.

---

## Implementation Strategy

**Phase 1 — the wired gate (one PR-sized unit)**: edit the three plugin artifacts together (skill workflow + descriptions, agent fence/workflow/output-contract/defensive sections, registry line + annotation rewrite), and land the new BDD content-inspection suite with the feature file's non-`@wip` scenarios passing. These cannot split: the drift guard's check (d) fails a registry line the agent does not name, and the annotation rewrite is only true once the workflow consults.

**Phase 2 — the truth sweep (small, depends on Phase 1)**: retire the stale validation scenario in the routing-rule guard feature (comment noting the consultation landed), add the dated superseding note to the drafting-path spec's Entry/assumption, append the re-validation addenda to the drafting path's and routing rule's validate records.

Phase 2 rides in the same PR as Phase 1 (the sweep is only honest alongside the change that forces it) but is a separable commit for review legibility.

---

## Risks

1. **Editing pinned prose breaks the drafting path's existing BDD suite** — likelihood medium, impact low (caught locally). The 067 content assertions are presence-based (`contains` on phrases like "page through the full result set", the six output-record element names); the mitigation is additive editing — keep every currently-pinned phrase, add the new steps and elements around them — and running the full `internal/build` suite before push.
2. **The widened descriptions over- or under-trigger the skill** — likelihood low, impact medium. The skill's description is its discovery surface; it must now say the path routes and consults without losing the "ready tension → created draft" trigger. Mitigation: the description edit is reviewed against the sibling paths' boundary sentences (each names what it is *not* for), and the tension-processing handoff sentence is updated in the same pass so the two skills' descriptions stay complementary.
3. **Two-phase re-delegation loses state between runs** — likelihood medium if undocumented, impact medium (a re-run that forgets the direction re-surfaces the same decision, looping). Mitigation: the skill's relay documentation makes the re-delegation input explicit and mandatory (settled anchor id, proceed-past instruction naming the fact), and the agent's defensive section says what to do when direction is present: act on it, do not re-ask.
4. **Recognition overreach** — likelihood low, impact high if it happens. A prompt-level matcher can drift into judging change sets the record does not name. Mitigation: the fence language mirrors the record's own closed posture (named match, fact handle, nothing implied on no-match — pinned by the no-match scenario), and the no-local-validator non-behavior is carried into the agent's hard-limits section.

---

## What This Plan Does Not Cover

- **Field-level contracts** — the consultation element's exact fields, the new `action` strings' exact spellings, the re-delegation input shape, and the registry line's comment wording are the interface accord's to pin (`/score:interface` next).
- **Executable scenarios** — the feature file (directory chosen by problem, not pre-named here) and scenario placement are the scenarios skill's.
- **Task decomposition** — the two phases above are sized for the tasks skill to cut into PR-sized units with acceptance criteria.
- **Identifier asks, typed builders, portfolio status sweeps** — explicitly out (spec non-behaviors and ADR-4's consequence); each has its own home.
