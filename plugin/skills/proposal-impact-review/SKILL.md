---
name: proposal-impact-review
description: See what a circulating proposal would change for the operator's own roles, actions, and projects before answering it — delegating the review to an isolated reviewer that returns a drawn-together impact picture with no verdict — and, once the operator has explicitly chosen a no-objection or bring-to-meeting, record that consent response through the guardrail-confirmed write, returning the recorded response with its prr_ id. Reach for this whenever a circulating proposal awaits the operator's consent response, the operator wants to see what a proposal would change for their own work before answering, or the operator is ready to record their own explicitly chosen no_objection or bring_to_meeting — a gated write, always the operator's choice, never inferred from the review. It is not for advancing, monitoring, or withdrawing the operator's own proposal (that is the Proposal Circulation Path), not for assembling changes or creating the draft (that is the Proposal Drafting Path), it does not judge whether an action is allowed or needs a proposal (that is the Constraint Discovery Path), it is not for capturing, refining, or retiring a tension (that is the Tension Processing Path), and it does not explain the governance around a concern (that is governance navigation).
---

# Reviewing a circulating proposal's impact before answering it

This skill takes a **circulating proposal** — carrying its `prp_` id, or only the
question *"what's pending on me?"* — and shows the operator what the proposal
would change for **their own roles, actions, and projects** before they answer
it: a drawn-together impact picture relating the change set to the operator's
footprint. The review **stands on its own** — it is a useful result even when no
objection surfaces — and responding is its optional culmination: when the
operator has **decided**, the path records their `no_objection` or
`bring_to_meeting` through the guardrail-confirmed write and returns the
recorded response.

It composes only commands the `glassfrog` CLI already exposes. It adds no
command, flag, or capability of its own. Its **one** write is the gated
`proposal respond`, issued in the caller's context after the operator decides;
every other write stays out of this path (see
[Decision-and-respond note](#decision-and-respond-note)).

## When to reach for it

Reach for proposal impact review when a **circulating proposal awaits the
operator's consent response** and the job is to see what it would change and how
that change lands on the operator's own governance — and then, when the operator
has decided, to **record their explicitly chosen response**.

Do **not** reach for it to *advance, monitor, or withdraw* a proposal — those
circulation acts belong to the **Proposal Circulation Path** (068). Do not use
it to *assemble changes or create the draft* — that is the **Proposal Drafting
Path** (067). Do not use it to answer *"am I allowed to do X?"* or *"does this
need a proposal?"* — that authority verdict belongs to the **Constraint
Discovery Path** (065). Do not use it to *capture, refine, or retire a
tension* — that is the **Tension Processing Path** (066). And do not use it to
*understand the governance around a concern* at large — that traversal is the
**governance-navigation** skill's job (064).

## The workflow

This is the single source of the review-and-respond steps — the
`proposal-impact-reviewer` agent runs exactly the review steps named here; it
does not carry a second copy. From a circulating proposal's `prp_` id (or the
pending-list question):

1. **Target** — with a `prp_` id in hand, proceed. With only *"what's pending on
   me?"*, delegate the pending-list question first: the reviewer situates via
   `proposal list` (paged through the **full** set — never a silent single-page
   cap) and returns the circulating proposals awaiting the operator, so the
   operator picks one to review.
2. **Review (delegated)** — delegate the review to the
   **`proposal-impact-reviewer`** subagent. It grounds the proposal
   (`proposal get <prp-id>`: status, change set, `response_summary`,
   `response_deadline`, `available_transitions`), draws the operator's footprint
   (`me`, `me roles`, `me actions`, `me projects`), reads back the affected
   governance where the change set touches that footprint (`roles <role-id>`,
   `domains <role-id>`, `policies <role-id>`), and synthesizes the impact picture.
3. **Present the picture** — what the proposal changes; where it intersects the
   operator's roles, actions, and projects; the current-vs-proposed read-backs;
   and the footprint-coverage qualifier (see the reviewer's footprint honesty —
   an incomplete `me roles` read qualifies every no-impact conclusion).
4. **The operator decides** — `no_objection`, `bring_to_meeting`, or **not
   yet** — a first-class exit: present the picture, record no response, and
   stop. The review is a **useful result on its own**; nothing obliges an
   answer.
5. **Record the response (on an explicit choice only)** — narrate the proposal
   and the chosen value, then run
   `glassfrog proposal respond <prp-id> --response <value>` — a **gated**
   governance write 063 confirms, issued **in the caller's context** (see
   [Decision-and-respond note](#decision-and-respond-note)) — and present the
   recorded response: its `prr_` id, the value, and the parent proposal's
   status at the time of response. A parent status of `accepted` means this
   response completed the expected set and closed the consent window — a signal
   surfaced from the record, never computed client-side. A recorded
   `bring_to_meeting` persists on the proposal and blocks auto-acceptance; the
   path then **stops** — advancing or withdrawing the proposal stays with the
   **Proposal Circulation Path** (068).

For the exact flags of any one command, ask the CLI:
`glassfrog <cmd> --help` (e.g. `glassfrog proposal respond --help`).

## Delegation

Run the **`proposal-impact-reviewer`** agent to execute the review. It performs
the grounding read, the footprint reads, and the affected-governance read-backs
in its **own isolated context** and returns **only** the impact picture — so the
raw `proposal get`/`me`/`roles` output stays out of your context and never
floods it. You pass the `prp_` id (or the pending-list question) as input; you
get back the impact picture (the proposal exactly as the server returned it, the
change set drawn out, the operator's footprint with its coverage qualifier, the
intersections with current-vs-proposed read-backs, and notes), with every
element carrying the id needed to act on it. The reviewer **never records the
response** — recording is this skill's caller-context step, after the operator
decides.

If the reviewer agent is absent or unregistered, the workflow above still
stands as guidance you can **follow by hand** — the path degrades to guidance,
and no CLI command is broken by the agent's absence.

## Decision-and-respond note

Recording a consent response is a **governance write gated by the Write-Safety
Guardrail (063)**: it always runs through the **confirmed write flow**, and it
is issued **in the caller's context** — the reviewer agent cannot and must not
run it. The one-token value rides inline on the command line
(`--response no_objection` / `--response bring_to_meeting`) so the confirmation
shows the **complete payload** — nothing hidden. The responding person is the
**token's own identity**. A **declined** confirmation is an outcome, not an
error: **no response is recorded**, and the path says so.

The value is always the **operator's own choice**: this skill never infers or
defaults the value from the review's content — *"no objections found"* is **not
an instruction to answer** `no_objection`. The picture informs; the operator
judges.

If the respond is rejected — a `403` plan refusal, a `404` unknown proposal, a
`422` the server did not allow (including an already-recorded response) — the
path surfaces the **API failure by name** from the CLI's own diagnostics and
**records nothing**: it never treats a **non-2xx** as success and fabricates no
state the **record does not contain**. It adds no retry — a retry is itself a
gated write needing fresh confirmation.

This path's **only** write is `proposal respond`. It never runs
`proposal create` — drafting is the **Proposal Drafting Path** (067) — and never
`proposal propose` or `proposal withdraw` — advancing and withdrawing are the
**Proposal Circulation Path**'s acts (068); 063 gates those writes regardless.
It never performs a tension write (that is the Tension Processing Path, 066),
it does not rule on whether the change is within the proposer's authority (that
is the Constraint Discovery Path, 065), and it does not advise on how to weigh
an objection or coach Holacracy practice.

For output shapes, pagination mechanics, and exit-code reactions, see the
orientation skill (062) — this path does not restate them.
