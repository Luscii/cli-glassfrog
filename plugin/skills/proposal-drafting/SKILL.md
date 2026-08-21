---
name: proposal-drafting
description: Turn a well-formed tension that is ready to become a governance change into a created draft proposal — grounding the draft in its anchor tension, situating it against the proposals already in flight in the circle, assembling the change set, and creating the draft through the guardrail-confirmed write with the change set shown inline — then returning a draft record carrying its prp_ id, ready to hand to circulation. Reach for this whenever a ready tension should become a draft proposal. It is not for capturing, refining, or retiring a tension (that is the Tension Processing Path), it does not judge whether an action is allowed or needs a proposal (that is the Constraint Discovery Path), it does not explain the governance around a concern (that is governance navigation), and it never advances or withdraws a circulating proposal (that is the Proposal Circulation Path) or records a response on one (that is the proposal-impact-review path).
---

# Drafting a ready tension into a created proposal

This skill takes a **well-formed anchor tension** — one already sensed and ready
to become a governance change — and turns it into a **created draft proposal**:
grounded in the anchor tension, situated against the proposals already in flight in
the circle, assembled into a change set, and created through the
**guardrail-confirmed write**, carrying the `prp_` id that feeds the Proposal
Circulation Path.

It composes only commands the `glassfrog` CLI already exposes. It adds no command,
flag, or capability of its own. Its **one** write is the gated `proposal create`;
every other write stays out of this path (see [Gated-write note](#gated-write-note)).

## When to reach for it

Reach for proposal drafting when a **well-formed tension is ready to become a
governance change** and you need to assemble the changes and create the draft
proposal, returning its `prp_` id through a confirmed gated write.

Do **not** reach for it to *capture, refine, or retire a tension* — that is the
**Tension Processing Path**; a ready tension is handed *to* this path, not
worked here. Do not use it to answer *"am I allowed to do X?"* or *"does this need
a proposal?"* — that authority verdict belongs to the **Constraint Discovery
Path**. Do not use it to *understand the governance around a concern* — that
traversal is the **governance-navigation** skill's job. And do not use it to
*advance or withdraw a circulating proposal* — that is the **Proposal Circulation
Path** — or to *record a response* on one — that is the **response side**, the
proposal-impact-review path. This path stops at the created draft.

## The workflow

This is the single source of the drafting steps — the `proposal-drafter` agent
runs exactly this workflow; it does not carry a second copy. From a well-formed
anchor tension's `ten_` id and the intended change:

1. **Ground** — `tension get <ten-id>` reads the anchor tension the draft is
   grounded in (its body and the `role_` id of the circle it belongs to), so the
   draft is anchored to a real, well-formed tension.
2. **Situate** — see what is already circulating before adding to it:
   `proposal list --role-id <circle> --status draft` surfaces the proposals
   already in flight in that circle. Where the in-flight list spans more than one
   page, **page through the full result set** (the default walk does this; see the
   orientation pagination guidance) **before judging duplicates** — the duplicate
   check is a judgment over the complete set, never a silent single-page cap. Use
   `proposal get <prp-id>` to inspect a candidate match. Situating narrows by
   **circle + `draft` status** because `proposal list` offers **no** tension
   filter — this path must not imply one.
3. **Check for a duplicate** — if the intended change matches a draft already
   circulating, **surface the existing proposal with its `prp_` id** and let the
   practitioner decide; do not silently create a duplicate draft.
4. **Assemble the change set** — build the `changes` JSON array: each element is
   an object with a **non-empty `type`**, and above that type floor the array is
   passed through **verbatim** — the path assembles and passes it, it does not
   interpret or validate any change's `type` value or command-specific keys, and
   it builds **no typed constructor**. (The CLI's create enforces the floor.)
5. **Surface & create** — narrate the anchor tension and the assembled change set
   for confirmation, then create the draft:
   `proposal create <ten-id> --changes '<inline JSON>'`. This is a **gated**
   governance write the Write-Safety Guardrail confirms — the change set is passed **inline** so the
   confirmation shows the exact payload (see [Gated-write note](#gated-write-note)).
6. **Hand off** — return the created draft record and hand its `prp_` id to the
   **Proposal Circulation Path**. Advancing or withdrawing the proposal is that
   path's job; recording a response is the response side's. Neither is this one's.

A create can also **fail because the server judged the draft invalid** — the write
was accepted and the draft it produced can never move forward. That is a distinct
failure, not a rejected create: the failure carries the created draft's `prp_` id
and the server's validation alerts. Do not re-run the same change set. Read the
alerts, revise, and create a new proposal from the same anchor tension; the dead
draft stays behind under the id the failure names.

For the exact flags of any one command, ask the CLI:
`glassfrog proposal <sub> --help` and `glassfrog tension get --help`.

## Delegation

Run the **`proposal-drafter`** agent to execute this workflow. It performs the
grounding read, the situating walk, and the one gated create in its **own isolated
context** and returns **only** the draft record — so the raw `tension
get`/`proposal list`/`proposal get` output stays out of your context and never
floods it. You pass the anchor `ten_` id and the intended change (natural language
and/or a prepared change set) as input; you get back the draft record (the created
draft, the anchor it was grounded in, the in-flight proposals it was situated
against, the action taken, the `prp_` handoff id, and notes), with every element
carrying the id needed to act on it next.

If the drafter agent is absent or unregistered, the workflow above still stands as
guidance you can **follow by hand** — the path degrades to guidance, and no CLI
command is broken by the agent's absence.

## Gated-write note

This path's **only** write is `proposal create`, and it is a **governance write
gated by the Write-Safety Guardrail**: the create always runs through the
**confirmed write flow**. The change set is passed **inline** (`--changes '<inline
JSON>'`) so the confirmation prompt displays the **exact payload** being
written — never a file path or `stdin` that would make the human confirm blind. A
**declined** confirmation means **no proposal is created**. A change set too
unwieldy for the command line is **surfaced to the caller**, never smuggled
through a hidden file — inline is the only source that keeps the confirmation
honest.

This path performs **no other write**. It never runs `proposal propose`,
`proposal respond`, or `proposal withdraw` — advancing and withdrawing belong to
the **Proposal Circulation Path**, recording a response to the **response side**
(the proposal-impact-review path), and the guardrail gates those writes
regardless; the ready `prp_` id is handed to circulation. It never performs a tension
write (that is the Tension Processing Path), and whether a tension *needs* a
proposal (or whether the practitioner is allowed to act) is a question for the
**Constraint Discovery Path**. This path drafts the proposal; it does not
rule on whether the governance record permits an action, and it does not advise on
governance craft or coach Holacracy practice.

That write fence holds in layers: the drafter agent's prompt is scoped to the seven
composed leaves and forbids any other `proposal` write, and the guardrail's `PreToolUse`
write gate (`plugin/hooks/glassfrog-write-gate.sh`) is the backstop that
interposes the human confirmation on `proposal create` — and would gate any other
proposal write regardless. In the target host, `PreToolUse` hooks fire for a
subagent's Bash calls too (the hook input carries an `agent_id` inside a subagent
call), so the gate reaches the drafter.
