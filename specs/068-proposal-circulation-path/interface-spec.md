# Interface Accord: Proposal Circulation Path — Specification

**Feature**: 068-proposal-circulation-path
**Role**: Crafter
**Touchpoint**: Specification
**Plan reference**: plan.md System Architecture + ADR-1 (thin skill + subagent), ADR-2 (new `proposal-circulator` agent, no reuse of 067's drafter), ADR-3 (both gated bodyless transitions inside the subagent, independently confirmed), ADR-4 (compose shipped proposal commands; reads inform, never gate), ADR-5 (drift guard + two-write gated-membership invariant)
**Inputs**: spec.md, plan.md, PROJECT.md; frontmatter contracts grounded against the shipped sibling artifacts in this repository (`plugin/skills/proposal-drafting/SKILL.md`, `plugin/agents/proposal-drafter.md`, `plugin/agents/proposal-drafting-commands.txt`, `plugin/hooks/gated-commands.txt`) and the 064/065/066/067 accords

> The artifact *is* the interface: two declarative plugin components — a `proposal-circulation` **skill** and a `proposal-circulator` **agent** — consumed by on-demand loading and delegation, not a runtime call. Protocol-level contracts are the two files' frontmatter, their required sections, the confirmation contract around the two gated transitions, the single-source leaf list the drift guard consumes, and the shape of the circulation record the agent returns.

---

## Surface

### Invocation

Two entry points, one delegating to the other (mirroring 064/065/066/067):

| Consumer | Entry point | Trigger |
|---|---|---|
| AI agent or human user | `plugin/skills/proposal-circulation/SKILL.md` | The skill's frontmatter `description` matches the need "a draft proposal is ready to circulate, a circulating proposal needs monitoring, or a circulating proposal needs pulling back to draft" — loaded **on demand** |
| The skill (as caller) | `plugin/agents/proposal-circulator.md` | The skill instructs the caller to **delegate** the circulation act to this subagent, passing the `prp_` id and the intent (advance / monitor / withdraw, as natural language); the subagent runs in its own context and returns the circulation record |

No flags or arguments on either — the proposal id and intent are passed as natural-language input to the agent.

### Structural layout (required files)

```
plugin/
  .claude-plugin/
    plugin.json                            # manifest — untouched (auto-discovery)
  skills/
    proposal-circulation/
      SKILL.md                             # when + workflow, delegates to the agent (required)
  agents/                                  # reused component type (064 ADR-2)
    proposal-circulator.md                 # gated-transition circulator subagent (required)
    proposal-circulation-commands.txt      # single-source composed-leaf list (ADR-5)
    proposal-drafter.md                    # 067 — untouched sibling
    tension-processor.md                   # 066 — untouched sibling
    constraint-navigator.md                # 065 — untouched sibling
    governance-navigator.md                # 064 — untouched sibling
internal/build/
    proposal_circulation_guard_test.go     # best-effort drift guard (companion, not part of the plugin package)
```

`plugin/skills/orientation/` (062), the sibling path skills (064/065/066/067), `plugin/agents/` siblings and their leaf lists, and `plugin/hooks/` (063 — including `gated-commands.txt`, which this feature *reads* in its guard but never edits) are untouched. `marketplace.json` remains absent (distribution is #70).

### `plugin.json` changes

None. Skills are auto-discovered from `skills/` and agents from `agents/` (confirmed by the four landed sibling agents). `version` MAY bump per the surface's pre-1.0 discretion. No existing field is rewritten.

### `SKILL.md` frontmatter

YAML, matching the 062/064/065/066/067 convention (`name` + `description` only):

| Field | Type | Required | Notes |
|---|---|---|---|
| `name` | string | yes | `proposal-circulation` |
| `description` | string | yes | **The trigger.** States *when* to reach for it (a draft proposal in hand ready to enter circulation; a circulating proposal to monitor for progress toward acceptance; a circulating proposal to withdraw back to draft for amendment) and that both transitions are gated writes run through explicit confirmation. Worded to fire on those needs and **not** on "assemble changes / create the draft" (that's 067), "record a no-objection / bring-to-meeting response" (that's 069), "capture/refine/retire a tension" (that's 066), "am I allowed to do X" (that's 065), or "understand the governance around a concern" (that's 064) |

### Required sections in `SKILL.md`

The body is thin — *when* + *workflow* + *delegation* + the gated-writes note, not a second copy of the steps:

| Section | Must convey |
|---|---|
| When to reach for it | The job: from a proposal's `prp_` id, advance the draft into circulation, monitor where the circulating proposal stands, or withdraw it back to draft; and the boundaries — not *drafting* (067), not *response recording* (069), not tension *processing* (066), not authority *judgment* (065), not governance *understanding* (064) |
| The workflow | The single-sourced steps: `prp_` id + intent → ground the act (`proposal get <prp-id>`: status, `response_summary`, `response_deadline`, `available_transitions`) → where the in-flight picture is relevant, situate (`proposal list`, paged through the **full** set — never a silent single-page cap) → by intent: **advance** (narrate the proposal and what `propose` will do, then `proposal propose <prp-id>` — a **gated** write 063 confirms), **monitor** (draw the picture together; no write), or **withdraw** (narrate, then `proposal withdraw <prp-id>` — the second **gated** write) → return the circulation record; after a withdraw, hand the `prp_` id back to the Proposal Drafting Path (067) for re-editing; consent responses belong to the response side (069). The reads **inform, never gate**: the snapshot of `available_transitions` is narration, never a client-side precondition — issue the transition and surface the server's `422` plainly |
| Delegation | Instruction to run the `proposal-circulator` subagent for execution so the monitoring walk and raw output stay out of the caller's context; and what the caller gets back (the circulation record). If the agent is absent, the workflow remains usable as guidance (documented degradation) |
| Gated-writes note | States that `proposal propose` and `proposal withdraw` are **governance writes gated by the Write-Safety Guardrail (063)**: each always runs through the confirmed write flow — the transitions are bodyless, so the confirmation shows the complete command — and each confirms **independently** (never batched or pre-authorized); a declined confirmation means no transition. The path's *only* writes are these two; it never runs `proposal create` (067) or `proposal respond` (069), which 063 gates regardless. Points at orientation (062) for output/exit-code/pagination mechanics rather than restating them |

### `agents/proposal-circulator.md` frontmatter

YAML, matching the shipped sibling agents (`name`, `description`, `tools`, `model`):

| Field | Type | Required | Notes |
|---|---|---|---|
| `name` | string | yes | `proposal-circulator` |
| `description` | string | yes | Gated-transition proposal circulator: given a proposal's `prp_` id and the intent, grounds the act in the proposal as the server returns it, situates against the circle's in-flight proposals where relevant, and advances the draft into circulation or withdraws the circulating proposal back to draft through the guardrail-confirmed bodyless transitions — or monitors without writing — returning a circulation record carrying the `prp_` id. Never creates a proposal or records a response; never pre-gates a transition on a read snapshot; never judges authority. The proposal-circulation skill delegates here |
| `tools` | string list | yes | **Write-capable-but-fenced grant**: `Bash, Read, Grep, Glob` — includes `Bash` (to invoke the `glassfrog` reads and the two gated transitions); **excludes `Write` and `Edit`** so the agent cannot mutate the workspace |
| `model` | string | optional | `inherit` unless a specific tier is chosen |

### Required sections in `agents/proposal-circulator.md`

| Section | Must convey |
|---|---|
| Identity & scope | Who it is (a proposal circulator) and its hard limits: exactly **two** writes — `proposal propose` and `proposal withdraw`, each always through 063's gate, each confirmed independently; **never** `proposal create` (067) or `proposal respond` (069); never a tension write (066); never an authority verdict (065); never coaching. **Reads inform, never gate**: it surfaces `available_transitions` but never turns the snapshot into a client-side precondition — it issues the transition and relays the server's refusal |
| Workflow | Executes the same single-sourced steps the skill names (references them; does not restate a divergent copy) |
| Confirmation contract | Before each transition: narrate the proposal (id, status, what the transition will do — for a withdraw, that the server clears `proposed_at`/`response_deadline` and deletes prior responses, as reflected in the returned record). The transitions are **bodyless** — the confirmed command line (`glassfrog proposal propose prp_…` / `glassfrog proposal withdraw prp_…`) *is* the complete payload; nothing is hidden. A declined confirmation is an outcome (`action: declined`), not an error. Two transitions in one session confirm twice |
| Composed commands | The exact `glassfrog` leaves it may call: `proposal get`, `proposal list` (reads) and `proposal propose`, `proposal withdraw` (the two gated writes) — and only these |
| Output contract | Returns **only** the circulation record (below), never a dump of raw command output |

### Circulation-record output shape

The agent's return value — the contract the caller consumes. Field names/types are the contract; the rendering is the agent's prose:

| Element | Type | Notes |
|---|---|---|
| proposal | Proposal? | The target proposal **exactly as the server last returned it**: `id` (`prp_…`), `status`, `response_summary`, `response_deadline`, `available_transitions`. After an advance: `proposed_outside_meeting` with the server-set deadline and the implicit `no_objection` in the summary. After a withdraw: back in `draft` with `proposed_at`/`response_deadline` cleared. Absent only when the grounding read itself failed |
| situating | Proposal[]? | The circle's in-flight proposals when the picture was drawn: each `id`, `status`, one-line summary. Absent when not relevant to the intent |
| action | string | What was done: `advanced` \| `monitored` \| `withdrawn` \| `declined` \| `none` |
| handoff | string? | After a withdraw: the `prp_` id to feed back to the Proposal Drafting Path (067) for re-editing. Absent otherwise |
| notes | string[] | Failure/decision notes (e.g. "advance rejected: 422 transition not allowed", "403 async proposals not enabled", "monitoring walk incomplete: page 2 failed", "confirmation declined — no transition", "consent responses are recorded via the response side (069)") |

Every listed element carries the id needed to read it again, advance it, or withdraw it (spec accord: the record bridges back into the CLI). No field is synthesized — acceptance is never computed client-side; the parent proposal's status is the signal, exactly as returned. A contract shape, not a serialization format.

### Single-source leaf list (ADR-5)

The composed leaves are written **once** in `plugin/agents/proposal-circulation-commands.txt`, consumed by both the agent's "Composed commands" reference and the drift guard — mirroring 063's `gated-commands.txt`, 064's `composed-reads.txt`, 066's `tension-processing-commands.txt`, and 067's `proposal-drafting-commands.txt`. Contract: newline-delimited two-token leaves (`proposal get`, `proposal list`, `proposal propose`, `proposal withdraw`), matching 063's registry format so membership checks compare like with like. The file's comments state the invariant the guard enforces: every leaf exists in the shipped CLI; `proposal propose` and `proposal withdraw` are **each a member of** 063's gated set; the two reads are **not**. The membership assertions are per-leaf contract-facts — a read/write swap across the gate cannot satisfy the guard by count alone. One file, two consumers, no duplicated list.

---

## Interactions

**Invocation-to-output flow**:

1. A caller has a proposal's `prp_` id (067's draft handoff, or a practitioner-identified proposal in either state) and an intent; the need matches the skill `description`; the host loads `SKILL.md` on demand.
2. The skill states the workflow and directs the caller to delegate to the `proposal-circulator` subagent, passing the `prp_` id and the intent.
3. The subagent runs the circulation act **in its own context**: `proposal get <prp-id>` to ground (status, `response_summary`, `response_deadline`, `available_transitions`) → where relevant, `proposal list` (full walk) to situate → for an advance or withdraw, narrates the proposal and the intended transition → runs the bodyless gated command (`proposal propose <prp-id>` or `proposal withdraw <prp-id>`).
4. 063's `PreToolUse` hook interposes: the practitioner sees the complete command — for a bodyless transition, the command line omits nothing — and confirms or declines. (Hook coverage inside subagent calls was empirically confirmed by 066; hook input carries `agent_id`.) Declined → no transition, `action: declined`. A session that advances and later withdraws confirms **each** transition independently.
5. The subagent returns **only** the circulation record — raw command output never leaves its context.
6. The caller receives the record and can act on any element by its id — after a withdraw, feeding the `prp_` id back to the Proposal Drafting Path (067) for re-editing; while circulating, pointing responders at the response side (069); or re-reading via `proposal get`.

**Instructional model**: the skill tells the caller *when to circulate and to delegate*; the agent *performs* the ground/situate/narrate/confirm/transition and *synthesizes* the record. Neither adds a `glassfrog` subcommand. For per-command flags both defer to `glassfrog proposal <sub> --help` (062 ADR-3). For output/pagination/exit-code mechanics both defer to the orientation skill (062).

**Reads inform, never gate**: the grounding read's `available_transitions` snapshot is narration for the proposer, never a client-side precondition. The agent issues the intended transition regardless of what the snapshot said and surfaces the server's `422` plainly — the server is the only transition authority (the 057/059 invariant, relayed not re-derived).

**Single-sourced workflow**: the steps are written once (in the skill) and referenced by the agent — no second, divergent copy (plan Risk 1).

---

## Error Communication

Specification artifacts fail as **constraint violations** and as **runtime circulation outcomes** the agent surfaces:

| Condition | Behavior |
|---|---|
| `SKILL.md` missing / no `description` | Skill not discoverable — the path is effectively absent; callers fall back to driving `proposal` commands by hand (still gated by 063) |
| `proposal-circulator.md` missing / not registered | Skill's delegation target is absent; circulation cannot run isolated (the workflow remains readable as guidance, degraded — plan Risk 6) |
| Agent `tools` grant omits `Bash` | Agent cannot invoke `glassfrog` at all — misconfiguration, not a safety win |
| Confirmation declined at the 063 gate | `action: declined`, no transition, a "confirmation declined" note — an outcome, not an error (spec edge case) |
| Transition rejected (`403` Premium refusal, `404` unknown proposal, `422` transition not allowed) | Agent surfaces the failure by name in `notes` with `action: none`; a `422` is a real refusal, **never absorbed as success** (the 057/059 invariant); no fabricated state |
| Grounding read fails | Agent surfaces the failure with `action: none` and no fabricated proposal |
| Monitoring list fails mid-walk | Agent surfaces the failure in `notes` and presents the proposals gathered so far, flagged incomplete — does not abort or invent (spec error scenario; 056 already flags the partial walk) |
| Caller asks the agent to record a response | Named handoff — "consent responses are the response side (069)" in `notes`; no `proposal respond` is ever run (spec edge case) |
| Withdraw succeeds | `action: withdrawn`, the returned proposal back in `draft` exactly as the server returned it (timestamps cleared, responses deleted server-side — reflected, not narrated as side-effect commentary), `handoff` set to the `prp_` id for 067 |
| Drift guard fails (`internal/build` test red) | A composed leaf no longer exists in the CLI, **or** `proposal propose` or `proposal withdraw` left 063's gated set, **or** a composed read entered it — the truthfulness/gated-invariant contract is broken; fix the artifact (or the claim) |
| Drift guard coverage reduced/omitted | Permitted **only if stated**, never silent — the test names what it does not cover (LEARNINGS: no silent caps) |

**Gated-write nuance (consistent with plan ADR-3)**: the `tools` grant gates *Claude tools*, not individual `glassfrog` subcommands — withholding `Write`/`Edit` blocks workspace mutation but the agent legitimately runs the two gated transitions through `Bash`. The write boundary is **layered**: (1) the agent prompt scopes execution to the four composed leaves and forbids every other proposal write; (2) 063's `PreToolUse` gate (`plugin/hooks/glassfrog-write-gate.sh`) interposes the human confirmation on `proposal propose` and `proposal withdraw` — and fires inside subagent Bash calls (confirmed by 066 against Claude Code) — and would gate `create`/`respond` as backstop if the prompt fence ever failed. Like 067 (and unlike 066, whose writes pass ungated by design), 068's writes are *supposed to be asked* — a transition that executes without a confirmation prompt is itself a defect (the per-leaf gated-membership drift-guard assertions exist to catch the registry-side cause).

**No-pre-gate nuance (plan ADR-4)**: the inverse failure also matters — an agent that *refuses* to issue a transition because a stale `available_transitions` snapshot didn't list it has forked the server's authority client-side. The prompt fence forbids it; a held-out validation scenario inspects for it; the surfaced `422` is the correct behavior when the server refuses.

Runtime HTTP error shapes, status rendering, optimistic-concurrency, and rate-limit handling are **N/A** here — they belong to the CLI the agent drives (015/017/031/032) and are surfaced through the exit-code reactions orientation (062) already documents. The plan-gate `403` renders through the CLI's own 061 diagnostic; the agent relays it without plan-aware interpretation of its own.

---

## Consistency Notes

- **Sibling interface files**: none — this feature has only a specification touchpoint (it adds no `glassfrog` subcommand, API, UI, or event surface).
- **Follows the 064/065/066/067 skill+agent pattern** (plan ADR-1): thin skill delegating to an isolated subagent, `plugin/agents/` home, directory auto-discovery, single-sourced workflow. Silent conformance — 064 carried the original divergence from 062 ADR-2. No 065-style interaction branch: the inputs are concrete on arrival, and the human interaction that matters is 063's structural confirmation.
- **New agent, not a reuse** (plan ADR-2, developer-confirmed): 067's `proposal-drafter` prompt forbids `proposal propose`/`respond`/`withdraw` — a validate-pinned invariant this feature must not erode. The circulator is a sibling with the inverse fence slice; 067's artifacts stay untouched. 069 decides its own form.
- **Extends 067's write posture, without the payload dimension** (plan ADR-3): 067 = one gated create with `--changes` inline so the confirmation shows the payload; 068 = two gated **bodyless** transitions whose command line *is* the payload — same in-subagent locus, same layered confirmation, no shell-quoting risk. Each transition confirms independently.
- **First path that reads `available_transitions` while invoking the transitions it advertises** (plan ADR-4): the 057/059 no-client-pre-gate invariant becomes a prompt fence + validation scenario here — reads inform, never gate.
- **Single-sources the leaf list** (plan ADR-5) following the 063/064/066/067 file precedent — with the guard asserting the two-in-two-out gate posture: both write leaves' *membership* in the gated set, and the reads' absence from it, per-leaf.
- **Depends on the orientation skill (062)** for output/pagination/exit-code mechanics and on 063's write gate for the confirmations — both single-sourced elsewhere, referenced not duplicated. Reads 063's `gated-commands.txt` in the guard; never edits it.
- **Hands off to sibling specs**: a withdrawn draft's `prp_` id feeds back to Proposal Drafting (067); consent responses go to the response side (069); authority questions go to Constraint Discovery (065). None is invoked or implemented here.
- **Frontmatter grounded against shipped repo artifacts** (067's drafter, 066's processor, 065's constraint-navigator, 064's navigator) rather than external plugin examples — the in-repo siblings are the stronger precedent.
