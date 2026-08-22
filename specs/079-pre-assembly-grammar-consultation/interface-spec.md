# Interface Accord: Pre-Assembly Grammar Consultation — Specification

**Feature**: 079-pre-assembly-grammar-consultation
**Role**: Crafter
**Touchpoint**: Specification
**Plan reference**: plan.md System Architecture (the component change map and the nine-step control flow), ADR-1 (in-place widening of the drafting path's artifacts), ADR-2 (two-phase decision points inside the agent), ADR-3 (`proposal grammar` joins the registry as an ordinary read, no guard code), ADR-4 (upstream truth sweep)
**Inputs**: spec.md, plan.md, PROJECT.md; grounded against the shipped artifacts this accord amends (`plugin/skills/proposal-drafting/SKILL.md`, `plugin/agents/proposal-drafter.md`, `plugin/agents/proposal-drafting-commands.txt`) and the sibling accords for the drafting path, the circle-routing record, and the grammar command

> The artifact *is* the interface: the same two declarative plugin components the drafting path already ships — its **skill** and its **drafter agent** — plus their single-source leaf registry, all amended in place. Nothing new is created under `plugin/`. The protocol-level contracts are the widened frontmatter descriptions, the gate's ordered workflow as the instructional surface, the eight-leaf fence, the grown `action` vocabulary, the new `consultation` element of the draft record, and the re-delegation input the two-phase returns rely on.

---

## Surface

### Invocation (unchanged entry points, widened trigger)

The two entry points and the delegation between them are the drafting path's own — this feature adds no entry point. What changes is what the descriptions promise:

| Artifact | Description requirement after 079 |
|---|---|
| `plugin/skills/proposal-drafting/SKILL.md` frontmatter | Must state the widened entry — the path takes the intended change (an anchor `ten_` id optionally in hand), **determines where the change lands and which anchors can route it there**, consults the grammar, and surfaces a recognized dead shape before the write — while keeping the existing trigger ("a ready tension should become a draft proposal") and every existing boundary sentence (not tension processing, not constraint discovery, not circulation, not the response side). |
| `plugin/agents/proposal-drafter.md` frontmatter | Must state the same widened entry in the agent voice (routes, consults, recognizes, two-phase returns) and keep the existing fences (never advances/responds/withdraws, never a tension write, never an authority verdict). |

The handoff sentence in the tension-processing skill's description stays complementary: it hands a ready tension *to* this path; nothing in 079 edits that artifact.

### Structural layout (edits in place — no new plugin files)

```
plugin/
  skills/proposal-drafting/
    SKILL.md                          # EDITED — workflow gains the gate; description widened
    references/
      change-set-grammar-facts.md     # untouched (rendered by the CLI; consulted through it)
      circle-routing-rule.md          # untouched (its procedure is what the routing step runs)
  agents/
    proposal-drafter.md               # EDITED — fence +1 leaf, output contract +1 element,
                                      #   action vocabulary +3, defensive entries +3
    proposal-drafting-commands.txt    # EDITED — +`proposal grammar`; routing-block annotation rewritten
  hooks/                              # untouched (grammar already in PROPOSAL_READS)
```

Repo-side (not part of the operating surface): a new BDD content-inspection suite in `internal/build`, the new feature file, and ADR-4's sweep edits. No `internal/build` production-guard code changes (plan ADR-3).

### The workflow contract (`SKILL.md` "The workflow" section)

The skill's workflow section remains the single source the agent executes. After 079 it must present these steps, in this order, each with its named reads:

| # | Step | Composed leaves | Contract |
|---|---|---|---|
| 1 | **Route** | `me roles`, `tension list`, `roles` | Run the routing record's procedure in its order; report the target circle and **every** eligible anchor, choosing none. A handed-in anchor that lands the change in the target circle settles step 1; one that does not, or no anchor in hand, is a two-phase return (see Interactions). |
| 2 | **Ground** | `tension get` | Unchanged — read the settled anchor. |
| 3 | **Situate** | `proposal list` | Unchanged — full walk, circle + `draft` status. |
| 4 | **Duplicate check** | `proposal get` | Unchanged — `surfaced-existing` return on a match. |
| 5 | **Consult** | `proposal grammar` | One invocation returns the whole reference; no credential, no request. On failure: note the gap in the consultation element and continue — never stop, never retry-loop. |
| 6 | **Assemble** | — | Unchanged — verbatim above the `type` floor. |
| 7 | **Match** | — | Compare the assembled set against the rendered dead shapes **with step 1's answer in hand**; a recognized match is a two-phase return carrying the fact's handle, shape, and symptom. No match → say so, implying nothing about validity. |
| 8 | **Confirm & create** | `proposal create` | Unchanged — inline `--changes`, the one gated write. |
| 9 | **Hand off** | — | Unchanged destination; the returned record now carries the consultation element. |

Prose requirements on the section: it names the record and the grammar command and **restates neither** (no change type, placement rule, shape text, or routing rule reproduced); it documents the relay loop (Interactions below); it keeps every phrase the drafting path's existing executable expectations pin (the six original step activities remain present).

### The fence contract (`proposal-drafter.md` "Composed commands" + registry)

The single-source registry lists exactly **eight** leaves; the agent's fence names all eight and no others:

```
tension get
proposal list
proposal get
proposal create
me roles
tension list
roles
proposal grammar
```

- `proposal grammar` enters the registry's read block with a comment naming it **the consultation read** — the grammar reference the gate consults before assembly — in the registry's existing annotation voice.
- The routing block's standing annotation ("no workflow step consults the record or runs these reads to route … a later change to this path adds the consultation. Do not infer a routing step from their presence.") is **rewritten**: the three reads are now the routing step's named reads, run by workflow step 1. The rewrite must leave no sentence claiming the reads are ahead of their use — that claim's falsity is a held-out validation scenario.
- The agent fence's matching note is rewritten in the same pass (the fence and registry describe the same posture; the drift guard's name-every-leaf check keeps them in step).
- Gate posture, unchanged and re-asserted by existing tests: `proposal create` remains the only registry member in the write gate's `gated-commands.txt`; the seven reads stay out of it; `grammar` is already in the gate script's recognized-read set, so consultation never prompts.

### Draft-record output shape (the drawn-together record)

The record keeps its six existing elements (`draft`, `anchor`, `situating`, `action`, `handoff`, `notes`) and gains one, present on **every** action path:

- **consultation** — what was consulted and what it surfaced, in three named parts:
  - **grammar** — consulted, or not consulted with the failure named (the read failed; assembly continued unconsulted).
  - **routing** — the determination's answer: the target circle's `role_` id and every eligible anchor's `ten_` id; the completeness hedge verbatim in posture (a search resting on the own-roles read reports "none found in `me roles`" and marks completeness uncertain); the incomplete-walk note when a read failed part-way; the record's decline when the target circle has no containing circle; or the capture-gap note when the eligible set is empty (naming capture on the specific role in the specific circle as the closing step, handed onward — never performed).
  - **match** — the recognized fact's handle (e.g. its `CSG-` id) with its shape and symptom as the rendering states them, or the explicit statement that no recorded shape matched (silence is not a signal; no-match implies nothing about validity).

The **`action`** vocabulary grows from four values to seven:

| `action` | Meaning | Draft created? |
|---|---|---|
| `created` | unchanged | yes |
| `surfaced-existing` | unchanged (duplicate check) | no |
| `declined` | unchanged (gate confirmation declined) | no |
| `none` | unchanged (create rejected / failure) | no |
| `surfaced-routing-mismatch` | the handed-in anchor lands the change outside the target circle; eligible anchors named | no — awaiting direction |
| `named-anchors` | routing answered and the anchor choice is the operator's: the eligible anchors are named (possibly none, with the capture-gap note) and none chosen | no — awaiting direction |
| `surfaced-dead-shape` | the assembled set matches a recorded dead shape; handle, shape, symptom carried | no — awaiting direction |

`named-anchors` deliberately covers both "choose among these" and "none exist yet" — the empty set with its capture-gap note is the same decision point (what the practitioner does next), so no fourth value is minted.

### Defensive-drafting contract (new entries in the agent)

Three entries join the agent's existing defensive list, in its voice: **the grammar read fails** (surface the failure, record grammar-not-consulted, continue — drafting is never withheld on it); **a routing read fails part-way** (name what failed, present the determination as incomplete, continue on what was established — invent nothing, abandon nothing); **direction is present** (a re-delegation carrying the practitioner's explicit direction acts on it — it does not re-surface the same decision; a decline is an outcome, not an error).

---

## Interactions

### The two-phase delegation loop

Every delegation runs the gate from step 1. Three outcomes end a run **before** the create, each a return-to-caller in the `surfaced-existing` shape the path already ships — the agent stops, returns the record with the awaiting-direction `action`, and the skill relays it to the practitioner:

1. **First delegation** — input: the intended change, an anchor `ten_` id optionally in hand.
2. **Return** (when a decision arises) — the record carries the decision's substance: the mismatch + eligible anchors, the named anchors (or the empty set + capture gap), or the recognized dead shape. The consultation element carries everything already established, so the practitioner decides on the record, not on a re-run.
3. **Re-delegation** — input carries the direction **explicitly**, because each delegation is a fresh context: the settled anchor's `ten_` id, and/or the proceed-past instruction naming the surfaced fact's handle. The re-delegated run repeats the gate from the top (the reads are cheap; a stale determination is worse than a repeated one) and, with direction present, acts on it rather than re-asking.
4. **Completion** — `created` (through the unchanged confirmed write flow), or any of the terminal non-created outcomes.

The loop never withholds: direction given is always acted on — the only thing that stops a create is the absence of the practitioner's direction (or the gate confirmation itself being declined, unchanged).

### Ordering and unconditionality

The nine steps run in order on every draft — no condition skips routing (the classification test's answer cannot gate its own run) and none skips the grammar consult (client-less, request-free). The match step (7) runs with the routing answer in hand: the anchor-dependent dead shape is recognizable only when both the change's target and the proposing circle are known.

---

## Error Communication

Consultation failures **degrade to reported gaps and continue**; decision points **return**; only the write's own failures end a run as failures — unchanged from the path's existing conduct:

| Condition | Conduct | Where it surfaces |
|---|---|---|
| Grammar read fails | Continue; assembly proceeds explicitly unconsulted | `consultation.grammar` names the failure; `notes` |
| Routing read fails part-way | Continue on what was established, flagged incomplete | `consultation.routing`; `notes` |
| Handed-in anchor routes elsewhere | Return, awaiting direction | `action: surfaced-routing-mismatch`; eligible anchors in `consultation.routing` |
| No anchor settled / choice open | Return, awaiting direction | `action: named-anchors` |
| No eligible anchor exists | Return; capture-gap note names the closing step (never performed) | `action: named-anchors` with the empty set; `consultation.routing` |
| Target circle has no containing circle | The record's decline reported; no target invented | `consultation.routing`; the run proceeds only on the practitioner's direction |
| Dead shape recognized | Return, awaiting direction; fact handle + shape + symptom | `action: surfaced-dead-shape`; `consultation.match` |
| No recorded shape matches | Stated explicitly; nothing implied about validity | `consultation.match` |
| Create rejected (403/404/422) | Unchanged — surfaced by name, nothing fabricated | `action: none`; `notes` |
| Gate confirmation declined | Unchanged — an outcome, not an error | `action: declined` |

Nothing in the table refuses, blocks, or delays the create — every row is a report or a return, and the server stays the sole judge of validity.

---

## Consistency Notes

- **What this accord supersedes in the drafting path's own accord**: the entry contract (anchor id as the sole starting point → intent, routed), the six-element record (→ seven), the seven-leaf fence (→ eight), and the four-value `action` vocabulary (→ seven). ADR-4's dated superseding note in that feature's spec points here; this accord is the current statement of those contracts.
- **What stays owned elsewhere**: the routing record's anatomy and content (the circle-routing accord — untouched); the grammar rendering's structure and provenance markings (the grammar-reference accord — this path consumes the command, and the command still tracks nothing about consultation; the tracking lives in the `consultation` element here); the write gate's registries and script (the guardrail accord — untouched).
- **One canonical list per enumeration**: the eight-leaf fence, the seven-element record, and the seven-value `action` vocabulary are each pinned once, in this accord's tables. The plugin artifacts state them in prose that the executable expectations hold to the registry; no second normative copy exists to drift.
- **Self-containment**: every sentence added to the plugin artifacts resolves in-surface or to the CLI — no spec numbers, repo paths, or development-repository mentions; the walking self-containment check enforces this on the edited files without a list update.
- **Deviation, deliberate**: the two-phase returns make `action` values that are *neither success nor failure* ("awaiting direction") — a shape no sibling path's record has needed. Recorded here so a reviewer does not "fix" them toward error semantics: a return is a report with a decision pending, and a decline after it is an outcome.
