# Interface Accord: Proposal Impact Review Path — Specification

**Feature**: 069-proposal-impact-review-path
**Role**: Crafter
**Touchpoint**: Specification
**Plan reference**: plan.md System Architecture + ADR-1 (thin skill + subagent), ADR-2 (new `proposal-impact-reviewer` agent, no reuse of the circulator or drafter), ADR-3 (review in the subagent; the operator decision and the gated `respond` in the caller context — announced divergence from 067/068's in-subagent write locus), ADR-4 (compose ten shipped leaves; footprint honesty; reads inform, never decide), ADR-5 (drift guard + one-in-nine-out gate-membership invariant)
**Inputs**: spec.md, plan.md, PROJECT.md; frontmatter contracts grounded against the shipped sibling artifacts in this repository (`plugin/skills/proposal-circulation/SKILL.md`, `plugin/agents/proposal-circulator.md`, `plugin/agents/proposal-circulation-commands.txt`, `plugin/hooks/gated-commands.txt`) and the 064/065/066/067/068 accords

> The artifact *is* the interface: two declarative plugin components — a `proposal-impact-review` **skill** and a `proposal-impact-reviewer` **agent** — consumed by on-demand loading and delegation, not a runtime call. Protocol-level contracts are the two files' frontmatter, their required sections, the split write locus (review isolated in the agent; the operator decision and the single gated `respond` in the caller context), the shape of the impact picture the agent returns, and the single-source leaf list the drift guard consumes.

---

## Surface

### Invocation

Two entry points, one delegating to the other (mirroring 064–068) — plus one **caller-context write step** the skill owns (the ADR-3 divergence):

| Consumer | Entry point | Trigger |
|---|---|---|
| AI agent or human user | `plugin/skills/proposal-impact-review/SKILL.md` | The skill's frontmatter `description` matches the need "a circulating proposal awaits my consent response — what would it change for me?", "review a proposal's impact on my roles before answering", or "record my no-objection / bring-to-meeting" — loaded **on demand** |
| The skill (as caller) | `plugin/agents/proposal-impact-reviewer.md` | The skill instructs the caller to **delegate the review** to this subagent, passing the `prp_` id (or the "what's pending on me?" question); the subagent runs the read traversal in its own context and returns the impact picture. The subagent **never records the response** |
| The caller itself (following the skill) | `glassfrog proposal respond <prp-id> --response <value>` | Issued **in the caller's context**, only after the operator has explicitly chosen a value; 063's `PreToolUse` gate interposes the human confirmation exactly as it does anywhere |

No flags or arguments on the skill or agent — the proposal id (or pending-list question), and later the operator-chosen response value, are passed as natural-language input.

### Structural layout (required files)

```
plugin/
  .claude-plugin/
    plugin.json                              # manifest — untouched (auto-discovery)
  skills/
    proposal-impact-review/
      SKILL.md                               # when + workflow + the caller-context respond step (required)
  agents/                                    # reused component type (064 ADR-2)
    proposal-impact-reviewer.md              # pure-read impact reviewer subagent (required)
    proposal-impact-review-commands.txt      # single-source composed-leaf list (ADR-5)
    proposal-circulator.md                   # 068 — untouched sibling
    proposal-drafter.md                      # 067 — untouched sibling
    tension-processor.md                     # 066 — untouched sibling
    constraint-navigator.md                  # 065 — untouched sibling
    governance-navigator.md                  # 064 — untouched sibling
internal/build/
    proposal_impact_review_guard_test.go     # best-effort drift guard (companion, not part of the plugin package)
```

`plugin/skills/orientation/` (062), the sibling path skills (064–068), `plugin/agents/` siblings and their leaf lists, and `plugin/hooks/` (063 — including `gated-commands.txt`, which this feature *reads* in its guard but never edits) are untouched. `marketplace.json` remains absent (distribution is #70).

### `plugin.json` changes

None. Skills are auto-discovered from `skills/` and agents from `agents/` (confirmed by the five landed sibling agents). `version` MAY bump per the surface's pre-1.0 discretion. No existing field is rewritten.

### `SKILL.md` frontmatter

YAML, matching the 062/064–068 convention (`name` + `description` only):

| Field | Type | Required | Notes |
|---|---|---|---|
| `name` | string | yes | `proposal-impact-review` |
| `description` | string | yes | **The trigger.** States *when* to reach for it (a circulating proposal awaits the operator's consent response; the operator wants to see what a proposal would change for their own roles, actions, and projects before answering; the operator is ready to record `no_objection` or `bring_to_meeting`) and that recording the response is a gated write run through explicit confirmation, always the operator's own choice. Worded to fire on those needs and **not** on "advance / monitor / withdraw my own proposal" (that's 068), "assemble changes / create the draft" (that's 067), "am I allowed to do X" (that's 065), "capture/refine/retire a tension" (that's 066), or "understand the governance around a concern" (that's 064) |

### Required sections in `SKILL.md`

The body is thin — *when* + *workflow* + *delegation* + the decision-and-respond note, not a second copy of the steps:

| Section | Must convey |
|---|---|
| When to reach for it | The job: from a circulating proposal's `prp_` id (or "what's pending on me?"), see what the proposal would change and how that change lands on the operator's own governance — then, when the operator has decided, record their consent response; and the boundaries — not *circulation management* (advance/monitor/withdraw, 068), not *drafting* (067), not authority *judgment* (065), not tension *processing* (066), not governance *understanding* at large (064). The review stands on its own: it is useful even when no objection surfaces, and responding is its optional culmination |
| The workflow | The single-sourced steps: `prp_` id in hand (or delegate the pending-list question first and let the operator pick) → **delegate the review** to the `proposal-impact-reviewer` subagent → present the returned impact picture to the operator (what changes; where it intersects their roles/actions/projects; the current-vs-proposed read-backs; the footprint-completeness qualifier) → **the operator decides**: `no_objection`, `bring_to_meeting`, or not yet (a first-class exit — present the picture and stop) → on an explicit choice, narrate the proposal and the chosen value, then run `glassfrog proposal respond <prp-id> --response <value>` — a **gated** write 063 confirms, issued in the caller's context — and present the recorded response (`prr_` id, value, the parent proposal's status at the time of response; `accepted` means the response closed the consent window). The skill **never infers or defaults the value** from the review's content |
| Delegation | Instruction to run the `proposal-impact-reviewer` subagent for the review so the read fan-out and raw output stay out of the caller's context; and what the caller gets back (the impact picture). If the agent is absent, the workflow remains usable as guidance (documented degradation) |
| Decision-and-respond note | States that recording a consent response is a **governance write gated by the Write-Safety Guardrail (063)**: it always runs through the confirmed write flow, in the caller's context (the reviewer agent cannot and must not run it); the value rides inline on the command line (`--response no_objection` / `--response bring_to_meeting`) so the confirmation shows the complete payload; the responding person is the token's own identity; a declined confirmation means no response is recorded. The path's *only* write is this one — it never runs `proposal create` (067) or `proposal propose`/`proposal withdraw` (068), which 063 gates regardless. Points at orientation (062) for output/exit-code/pagination mechanics rather than restating them |

### `agents/proposal-impact-reviewer.md` frontmatter

YAML, matching the shipped sibling agents (`name`, `description`, `tools`, `model`):

| Field | Type | Required | Notes |
|---|---|---|---|
| `name` | string | yes | `proposal-impact-reviewer` |
| `description` | string | yes | Read-only proposal impact reviewer: given a circulating proposal's `prp_` id (or the pending-on-me question), reads the proposal and its change set back, draws the operator's own governance footprint through the `me` reads, reads back specifically-affected roles/domains/policies, and returns a synthesized impact picture — what would change, and where it intersects the operator's work — carrying the ids needed to act on it. Never records a response or performs any proposal write; never decides or recommends the operator's answer; never judges the proposer's authority. The proposal-impact-review skill delegates here |
| `tools` | string list | yes | **Read-posture grant**: `Bash, Read, Grep, Glob` — includes `Bash` (to invoke the `glassfrog` reads); **excludes `Write` and `Edit`** so the agent cannot mutate the workspace. The proposal-write fence is prompt-level, backstopped by 063 |
| `model` | string | optional | `inherit` unless a specific tier is chosen |

### Required sections in `agents/proposal-impact-reviewer.md`

| Section | Must convey |
|---|---|
| Identity & scope | Who it is (a proposal impact reviewer) and its hard limits: **zero writes** — it never runs `proposal respond` (this path's own write — that step belongs to the caller, after the operator decides), never `proposal create` (067), never `proposal propose`/`withdraw` (068), never a tension write (066); it computes **no objection verdict** and recommends no answer; it never rules on the proposer's authority (065) and never coaches Holacracy practice. **Reads inform, never gate**: it surfaces `available_transitions` as narration and never turns a snapshot into a client-side precondition (068's inherited invariant) |
| Workflow | Executes the review steps the skill names (references them; does not restate a divergent copy): ground the proposal (`proposal get <prp-id>`: status, change set, `response_summary`, `response_deadline`, `available_transitions`) → where the entry is "what's pending", situate instead (`proposal list`, paged through the **full** set — never a silent single-page cap) → draw the footprint (`me`, `me roles`, `me actions`, `me projects`) → for change elements that touch the footprint, read the affected governance back (`roles <id>`, `domains <id>`, `policies <id>`) for the current-vs-proposed picture → synthesize |
| Footprint honesty | `me roles` does **not** paginate: when more roles exist than one page, the CLI emits an incompleteness signal (stderr note in human formats; in-band pagination metadata in `json`/`yaml`) and exits 0. The reviewer must read that signal and carry it into the picture: `footprint_coverage` is tri-state, and a "does not touch your current governance" conclusion over an incomplete footprint must be stated as *"not in the roles visible to this read (list incomplete)"* — never an unqualified negative. When unsure whether a change touches the operator, **show it** (fail toward surfacing) |
| Composed reads | The exact `glassfrog` leaves it may call: `proposal get`, `proposal list`, `me`, `me roles`, `me actions`, `me projects`, `roles`, `domains`, `policies` — and only these (nine reads; the tenth registry leaf, `proposal respond`, is the skill's and never the agent's) |
| Output contract | Returns **only** the impact picture (below), never a dump of raw command output |

### Impact-picture output shape

The agent's return value — the contract the caller consumes. Field names/types are the contract; the rendering is the agent's prose:

| Element | Type | Notes |
|---|---|---|
| proposal | Proposal? | The target proposal **exactly as the server returned it**: `id` (`prp_…`), `status`, `response_summary`, `response_deadline`, `available_transitions`. Absent only when the grounding read itself failed, or in pending-list mode |
| changes | Change[]? | The proposal's change set drawn out: each element's type (e.g. `CreateRole`, `UpdateAccountability`) and the governance it targets (role/domain/policy id and name where the record carries one). Reflected from the record, never re-interpreted into a governance ruling |
| footprint | Footprint? | The operator's current governance as read: identity (from `me`), roles filled, actions held, projects carried — each with its id. Absent only when the footprint reads all failed |
| footprint_coverage | string | **Tri-state**: `complete` \| `incomplete` (the `me roles` incompleteness signal fired — the picture says so) \| `unknown` (a footprint read failed). Never a silent `complete` |
| intersections | Intersection[] | Where the change set lands on the operator: per affected element — the change, the operator's touched role/action/project, and the current-vs-proposed read-back (what the governance is today beside what the proposal would make it). Empty array = the honest no-impact case, qualified per `footprint_coverage` |
| pending | Proposal[]? | Pending-list mode only: the circulating proposals awaiting the operator — each `id`, `status`, `response_deadline`, one-line summary — so the operator can pick one to review |
| notes | string[] | Failure/qualifier notes (e.g. "affected-role read failed: 404 — shown from the change set only", "footprint incomplete: more roles exist than shown", "review incomplete: policies read failed", "no impact found in the roles visible to this read (list incomplete)") |

Every listed element carries the id needed to read it again or act on it — the picture bridges back into the CLI (spec accord). No verdict field exists: the picture contains **no recommended response and no objection ruling** — the operator judges. A contract shape, not a serialization format.

### Single-source leaf list (ADR-5)

The composed leaves are written **once** in `plugin/agents/proposal-impact-review-commands.txt`, consumed by the agent's "Composed reads" reference, the skill's respond step, and the drift guard — mirroring 063's `gated-commands.txt` and the 064/066/067/068 registry files. Contract: newline-delimited command paths (`proposal get`, `proposal list`, `proposal respond`, `me`, `me roles`, `me actions`, `me projects`, `roles`, `domains`, `policies`) — two-token `proposal <sub>` / `me <sub>` paths written like 063's registry lines so membership checks compare like with like; bare `me` and the top-level reads are single tokens (`domains`/`policies` are top-level commands, not `roles` subcommands — the 064/065 precedent). The file's comments state the invariant the guard enforces: every leaf resolves in the shipped CLI; `proposal respond` **is a member of** 063's gated set; the nine reads are **not**; and the write leaf belongs to the *skill's* step, never the agent's composed reads. The membership assertions are per-leaf contract-facts — a read/write swap across the gate cannot satisfy the guard by count alone. One file, three consumers, no duplicated list.

---

## Interactions

**Invocation-to-output flow**:

1. A caller has a circulating proposal's `prp_` id (or only the question "what's pending on me?"); the need matches the skill `description`; the host loads `SKILL.md` on demand.
2. The skill states the workflow and directs the caller to delegate the review to the `proposal-impact-reviewer` subagent, passing the `prp_` id or the pending-list question.
3. The subagent runs the review **in its own context**: `proposal get <prp-id>` to ground (or `proposal list`, full walk, for the pending picture) → `me` / `me roles` / `me actions` / `me projects` for the footprint (reading the `me roles` incompleteness signal) → `roles <id>` / `domains <id>` / `policies <id>` read-backs for the change-touched parts → synthesizes the impact picture.
4. The subagent returns **only** the impact picture — raw command output never leaves its context. In pending-list mode the operator picks a proposal and steps 2–3 repeat for it.
5. The caller presents the picture; **the operator decides**: `no_objection`, `bring_to_meeting`, or not yet. "Not yet" ends the path with the picture as its result — a first-class exit, not a failure.
6. On an explicit choice, the caller — following the skill, in its **own** context — narrates the proposal and the chosen value, then runs `glassfrog proposal respond <prp-id> --response <value>`. 063's `PreToolUse` hook interposes: the practitioner sees the complete command — the one-token value rides inline, nothing hidden — and confirms or declines. Declined → no response recorded.
7. The caller presents the recorded response exactly as returned: `prr_` id, the value, and the parent proposal's status at the time of response — `accepted` when this response closed the consent window (surfaced as the signal; acceptance is never computed client-side).

**Instructional model**: the skill tells the caller *when to review, to delegate, and how to carry the operator's decision across the gate*; the agent *performs* the ground/footprint/read-back traversal and *synthesizes* the picture. Neither adds a `glassfrog` subcommand. For per-command flags both defer to `glassfrog <cmd> --help` (062 ADR-3). For output/pagination/exit-code mechanics both defer to the orientation skill (062).

**Reviews inform, never decide (the split locus is the mechanism)**: the agent that draws the picture has no respond mode — the decision input (`--response <value>`) enters the flow only from the operator, in the caller's context, after the picture exists. The skill forbids inferring a value from the picture ("no objections found" is **not** an instruction to answer `no_objection`); the agent's output shape has no verdict field to smuggle one through.

**Single-sourced workflow**: the steps are written once (in the skill) and referenced by the agent — no second, divergent copy (plan Risk: family pattern).

---

## Error Communication

Specification artifacts fail as **constraint violations** and as **runtime review/respond outcomes**:

| Condition | Behavior |
|---|---|
| `SKILL.md` missing / no `description` | Skill not discoverable — the path is effectively absent; callers fall back to driving `proposal`/`me` commands by hand (the respond still gated by 063) |
| `proposal-impact-reviewer.md` missing / not registered | Skill's delegation target is absent; the review cannot run isolated (the workflow remains readable as guidance, degraded) |
| Agent `tools` grant omits `Bash` | Agent cannot invoke `glassfrog` at all — misconfiguration, not a safety win |
| Grounding read fails (`404` unknown/invisible proposal, network) | Agent surfaces the failure by name in `notes`; no fabricated proposal, no fabricated picture |
| A footprint or read-back fails mid-picture | Agent surfaces what failed and presents what it gathered, flagged incomplete (`footprint_coverage: unknown` when a footprint read failed) — never invented data, never an abandoned review (spec error scenario) |
| `me roles` signals incompleteness | `footprint_coverage: incomplete`; every no-impact conclusion qualified ("not in the roles visible to this read"); never an unqualified negative (plan ADR-4a — the tri-state rule) |
| The change set does not touch the operator | `intersections: []` with the honest no-impact statement, qualified per `footprint_coverage` — a load-bearing result, not an empty failure (spec edge case) |
| Operator declines to respond / defers | The picture is the result; no `respond` issued — a first-class exit (spec edge case) |
| Confirmation declined at the 063 gate | No response recorded; the skill reports the declined confirmation as the outcome, not an error |
| Respond rejected (`403` Premium refusal, `404`, `422` — e.g. the operator already responded) | The failure surfaces by name from the CLI's own diagnostics (the plan-gate `403` renders through 061's possibility-framed wording); nothing recorded, no retry — a retry is itself a gated write needing fresh confirmation (063 ADR-5) |
| Respond succeeds | The recorded response exactly as returned: `prr_` id, value, parent status (`accepted` when auto-acceptance triggered — surfaced, never computed) |
| Caller asks the agent to record the response | Named handoff — the agent refuses with "recording the response is the skill's caller-context step, after the operator decides"; no `proposal respond` is ever run by the agent |
| Drift guard fails (`internal/build` test red) | A registry leaf no longer resolves in the CLI, **or** `proposal respond` left 063's gated set, **or** a read entered it — the truthfulness/gate-posture contract is broken; fix the artifact (or the claim) |
| Drift guard coverage reduced/omitted | Permitted **only if stated**, never silent — the test names what it does not cover (LEARNINGS: no silent caps) |

**Gated-write nuance (plan ADR-3 — the announced divergence)**: unlike 067/068, the gated write does **not** run inside the subagent — the operator's decision structurally splits the workflow at the read/write seam, so the write runs in the caller's context, where 063's `PreToolUse` gate fires identically (the hook gates any `Bash` invocation of a registered proposal-write leaf, caller or subagent). The write boundary is **layered**: (1) the agent prompt forbids **all four** proposal writes — the strictest fence in the family, making "the thing that draws the picture cannot record the answer" structural; (2) the skill permits exactly one write, only on an explicit operator-chosen value, narrated before issuing; (3) 063's gate interposes the human confirmation and would gate `create`/`propose`/`withdraw` as backstop if guidance ever failed. A respond that executes without a confirmation prompt is itself a defect (the per-leaf gate-membership drift-guard assertion exists to catch the registry-side cause).

**Never-decide nuance (plan ADR-4)**: the inverse failure also matters — a picture that *recommends* an answer, computes an objection verdict, or a skill flow that auto-fills `--response` from the review's content has forked the consent judgment the operator owns. The prompt fence and the verdict-free output shape forbid it; a held-out validation scenario inspects for it.

Runtime HTTP error shapes, status rendering, optimistic-concurrency, and rate-limit handling are **N/A** here — they belong to the CLI the artifacts drive (015/017/031/032) and are surfaced through the exit-code reactions orientation (062) already documents.

---

## Consistency Notes

- **Sibling interface files**: none — this feature has only a specification touchpoint (it adds no `glassfrog` subcommand, API, UI, or event surface).
- **Follows the 064–068 skill+agent pattern** (plan ADR-1): thin skill delegating to an isolated subagent, `plugin/agents/` home, directory auto-discovery, single-sourced workflow. Silent conformance — recorded because the spec deferred the form.
- **New agent, not a reuse** (plan ADR-2): 068's circulator and 067's drafter prompts each forbid `proposal respond` — validate-pinned invariants this feature must not erode. The reviewer is a sibling with the complementary fence; 067/068 artifacts stay untouched. 069 closes the family — no later path inherits the reuse question.
- **Diverges from 067/068's write locus — announced** (plan ADR-3): 067 = one gated create in-subagent, payload inline; 068 = two gated bodyless transitions in-subagent; 069 = review in-subagent, the one gated `respond` in the **caller** context, because the operator's decision splits the workflow where an isolated subagent cannot ask (the 065 interaction-in-skill finding, extended to the write side). Same layered confirmation, same 063 gate, same inline-payload principle (a one-token `--response` value); the reviewer gains the family's strictest fence (zero proposal writes) in exchange.
- **Interaction lives in the skill** (065 ADR-1 precedent, silently conformed to and extended): 065 put the clarify-when-vague branch in the skill; 069 puts both the pick-a-pending-proposal moment and the response decision there. The agent stays non-interactive.
- **Footprint honesty single-sources the 012 caveat** (plan ADR-4a): `me roles` is the family's one non-paginating read (silently-incomplete + stderr note, exit 0); the tri-state `footprint_coverage` element and the qualified no-impact wording are the contract-level enforcement of the 065/#155 tri-state lesson — a judgment derived from a possibly-incomplete list is uncertain, never a confident negative.
- **Inherits 068's no-pre-gate invariant** (plan ADR-4b): `available_transitions` is narration; the respond is issued and the server's `422` surfaced plainly.
- **Single-sources the leaf list** (plan ADR-5) following the 063/064/066/067/068 file precedent — with the guard asserting the one-in-nine-out gate posture: `proposal respond`'s *membership* in the gated set and the nine reads' *absence* from it, per-leaf.
- **Depends on the orientation skill (062)** for output/pagination/exit-code mechanics and on 063's write gate for the confirmation — both single-sourced elsewhere, referenced not duplicated. Reads 063's `gated-commands.txt` in the guard; never edits it.
- **Hands off to sibling specs**: advancing/monitoring/withdrawing stay with Proposal Circulation (068); drafting with Proposal Drafting (067); authority questions with Constraint Discovery (065); deeper governance navigation with Governance Navigation (064). None is invoked or implemented here.
- **Frontmatter grounded against shipped repo artifacts** (068's circulator, 067's drafter, 066's processor, 065's constraint-navigator, 064's navigator) rather than external plugin examples — the in-repo siblings are the stronger precedent.
