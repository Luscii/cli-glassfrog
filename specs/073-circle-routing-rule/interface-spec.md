# Interface Accord: Circle Routing Rule — Specification

**Feature**: 073-circle-routing-rule
**Role**: Crafter
**Touchpoint**: Specification
**Plan reference**: plan.md System Architecture — "The record", "The composed surface widens here", "The guard" (ADR-1 sibling placement, ADR-2 form, ADR-3 composed-surface widening, ADR-4 guard coupling, ADR-5 retirement)

---

## Surface

### Invocation

N/A — the record has no entry point. It is consumed by reading: loaded by the pre-assembly gate (#77) when that lands, and parsed by the guard at CI. Nothing invokes it, and nothing consults it at landing (spec validation scenario "The content ships unconsulted").

### Configuration

N/A — no configurable parameters. The record's path is a shared constant in `internal/build` (see Guard Coupling).

### The record file

**Path**: `plugin/skills/proposal-drafting/references/circle-routing-rule.md` — owned by the proposal-drafting skill under the 072 ownership rule; a **sibling** to `change-set-grammar-facts.md` in the same `references/` directory (plan ADR-1), the directory's second member.

**Required file anatomy, in order**:

| # | Element | Structural requirement |
|---|---|---|
| 1 | Empirical marker | A leading blockquote (`> …`) before any heading, containing the phrase `Empirical record` and stating the **cite-versus-observe split**: that the absence of a circle parameter on proposal create is published contract, while *where the proposal consequently lands* is observed server behaviour and no part of the published v5 contract. This is the guard's marker anchor, and the split is what keeps VISION Principle 1 intact for a record whose two halves have different standing. |
| 2 | Document header | `# Circle Routing Rule` title, followed by two field lines: an **Owner** line naming the proposal-drafting skill and the symlink consumption rule, and a **Contract citations** line naming `spec/glassfrog-api-v5.yaml`. No live-facts manifest — the content is one rule, not an enumerable fact set (plan ADR-2); the named-reads block (row 6) is this record's declaration surface. |
| 3 | Contract-citations section | A `## ` section that **cites, never restates**: (a) the rule's premise, anchored as `CreateProposalRequest.properties.proposal`, stating that it requires only `tension_id` (with optional `changes`) and carries **no circle property**; (b) the circle indicator, anchored as `Role.has_subroles`; (c) the root signal, anchored as `Role.parent_role_id`, stating it is nullable. Anchors are schema/property names — never line numbers, never a pinned count. |
| 4 | Rule section | A `## ` section carrying **four required bold-labelled fields** (`- **<Label>**: …`). Labels are the machine anchors; prose around them is free. |
| 5 | Classification-test section | A `## ` section carrying **three required bold-labelled fields**. |
| 6 | Procedure section | A `## ` section carrying **four required bold-labelled fields** plus the **named-reads block**. |

**Rule section fields** (element 4):

| Field | Requirement |
|---|---|
| Mechanism | A proposal inherits the circle of its anchor tension's sensing role; the create carries no circle parameter, so the anchor choice *is* the routing choice. Marked as the observed half. |
| Own-circle consequence | A change to a circle's own governance must be anchored in that circle's **parent**, on a tension sensed by a role the operator fills there. |
| Circle Lead exception | Where the operator fills the circle's own Circle Lead role, the circle-role itself is a valid anchor site. Carried as recorded from LEARNINGS F7, not independently re-verified. |
| Root circle | Where the target circle has no containing circle, the record **states the limit and declines to name a target** — explicitly not defaulting to the circle itself or any other circle. Cites `Role.parent_role_id` being null as the signal. |

**Classification-test section fields** (element 5):

| Field | Requirement |
|---|---|
| Test | How to tell a change to a circle's own governance from a change to a role inside a circle, expressed so a consumer routes from the change target alone. |
| Resolved by | Names `Role.has_subroles` as what resolves whether a target is a circle. |
| Parent resolution | Names `Role.parent_role_id` as the containing role, and states that the own-roles read already carries it for roles the operator fills, so the role read is needed only when the target circle is not among them. |

**Procedure section fields** (element 6):

| Field | Requirement |
|---|---|
| Answer shape | The target circle named by its `role_` id; each eligible anchor tension named by its `ten_` id. |
| All anchors named | Where several anchors are eligible, all are named and none is chosen. |
| Gap reporting | Where the operator fills a role in the target circle with no tension sensed on it, name capture on that specific role as the step that closes the gap. |
| Uncertainty | Where an answer rests on the own-roles read, mark its completeness uncertain and name the read — because that read does not follow pagination, an absence in it is an absence in what was read. Never a settled absence. |

**Named-reads block** (element 6, the record's declaration surface and the guard's parse anchor):

A fenced code block inside the Procedure section, one read leaf per line, in the order the procedure runs them. **Token grammar is identical to the `plugin/agents/*-commands.txt` family** — a single token for a top-level command, two tokens for `<group> <sub>` — so the guard's set comparison against the drafting registry compares like with like rather than reconciling two spellings:

```
me roles
tension list
roles
```

Reading the block: `me roles` is the `roles` subcommand of `me`; `tension list` is the `list` subcommand of `tension`; `roles` is a **top-level** command, not a `roles` subcommand of anything. The record is the single source of which reads its procedure names — the guard hard-codes none of them.

### Composed-surface additions

Three coupled edits (plan ADR-3), all three required for the existing 067 guard to stay green:

| Target | Change |
|---|---|
| `plugin/agents/proposal-drafting-commands.txt` | The three read leaves appended, with a comment annotating them as **routing reads the path may run once #77 consults the record** — so a reader sees why they are present ahead of a workflow that uses them (plan R5). |
| `plugin/agents/proposal-drafter.md` `## Composed commands` | The three reads added to the fence list with the same annotation. The fence's "only these … and it is these and no others" wording is preserved; what changes is the list it governs. |
| `internal/build/proposaldrafting.go` | `CheckProposalDraftingDrift` gains `liveTop` and `liveMe` parameters and a four-way leaf resolution — top-level, `me <sub>`, `tension <sub>`/`proposal <sub>`, and an **unanchorable-default arm that reports rather than skips** — transplanted from 065's `CheckConstraintDrift`. Callers supply `LiveTopLevelCommands()` (064) and `LiveMeSubcommands()` (065); both already exist. |

The gated-membership assertions are unchanged in intent and must remain: `proposal create` stays the sole composed leaf in 063's gated set, and every other composed leaf — now seven, not four — stays absent from it.

### Guard coupling

| File | Contents |
|---|---|
| `internal/build/circleroutingrule.go` | The record's path constant; parsers deriving every side — record side (marker presence and its cite-versus-observe phrase, required section presence, required field labels per section and their non-emptiness, the named-reads block's leaves, the cited schema anchors) and spec side (`CreateProposalRequest.properties.proposal`'s property-name set, the presence of `has_subroles` and `parent_role_id` on `Role`). |
| `internal/build/circle_routing_rule_guard_test.go` | The guard test asserting the invariants in Error Communication. |

Every side is derived from its file at test time. The guard hard-codes **no read names, no property sets, and no schema field values** (plan ADR-4).

---

## Interactions

**Consumer flows** (all read-only; none ship in this feature):

- **Pre-assembly consultation** (#77): the drafting path loads the record before assembling a change set, establishes the target circle and eligible anchors by following the procedure, and surfaces a routing problem before the write. This is where the determination actually happens — the record supplies the rule and the steps, never the answer.
- **Second skill-consumer** (future, none planned): consumes via a symbolic link into its own skill directory, added by the spec that adds the consumption — never a copy (the 072 ownership rule).

**Maintenance flow** — retirement (plan ADR-5) is **whole-record** and a two-part act in one change:

1. **delete the record** — a superseded routing rule is not merely stale but actively wrong, because a consumer following it routes to a circle that cannot decide the change;
2. **record the supersession** via `/score:deprecate`, naming the spec revision that dissolved the premise.

The trigger is **premise dissolution** — proposal create gaining a circle parameter, which makes the anchor no longer determine the circle. Changes to a *consequence* (the Circle Lead exception alone, say) are ordinary edits, not retirements. There is no manifest to keep in step, so retirement cannot collide with the guard. The same flow lands this feature: one `/score:deprecate` entry supersedes LEARNINGS 2026-08-05 F7.

**Guard flow**: CI parses the record and the vendored spec on every build. A refresh that adds any property to the create request's `proposal` object — or drops either cited `Role` field — fails at the record, forcing re-derivation or retirement before the change lands.

---

## Error Communication

The guard is the record's only enforcement surface. Each violation fails the build loudly, naming the invariant, the offending element, **and which resolution path applies**, rather than leaving the reader to infer it (plan ADR-4). Never silently, never partially.

The conditions that check the record **against the published contract** — the premise tripwire and the classification anchors — admit exactly two legitimate answers: **re-derive against the changed contract, or retire the record**. Retirement is always whole-record (plan ADR-5); no condition is ever resolved by deleting a required element from the record, because a record missing a required element fails conditions 1, 2, or 4 instead. The remaining conditions check the record's internal consistency or its agreement with the shipped CLI, and their resolution paths differ accordingly.

| # | Condition | Failure message names | Resolution path named |
|---|---|---|---|
| 1 | A required section is absent | the missing section | Add the section; the record is incomplete, not merely terse |
| 2 | A required field label is missing or its value empty | the section and the field label | Supply the field |
| 3 | The empirical marker blockquote is absent or missing its required phrase | the marker requirement | Restore the marker |
| 4 | The named-reads block is absent or empty | that the record declares no reads | Declare the reads the procedure names; a procedure that names none is not a procedure |
| 5 | A named read does not resolve on the shipped CLI | the leaf and which surface was searched (top-level / `me` / `tension`) | Fix the record, or restore the command |
| 6 | A named read carries a command path the guard cannot anchor | the leaf and the supported forms | Extend the guard or fix the record — never skipped silently |
| 7 | A named read is absent from `proposal-drafting-commands.txt` | the leaf and the registry path | Add it to the registry (and the agent fence), or drop it from the procedure — the record must not name a read the path is forbidden to run |
| 8 | `CreateProposalRequest.properties.proposal`'s property set is not exactly `{tension_id, changes}` | both sets, so the added or removed property is readable from the failure | **Re-derive the rule against the new parameter, or retire the record** — a circle parameter dissolves the premise |
| 9 | `has_subroles` or `parent_role_id` is absent from the `Role` schema | the missing field and the section citing it | **Re-derive the citation against the new anchor, or retire the record** — if the contract can no longer distinguish a circle from a role, the classification test cannot be performed at all |

Condition 8 is the **premise tripwire** and is deliberately a set-equality on the whole property set rather than a search for circle-like key names: a circle parameter could arrive under any spelling, and a name-matching check would miss exactly the change that matters.

**Degradation**: none — the record has no optional inputs.

**Explicitly partial** (stated, not silent), three residues:

1. **Semantic drift is undetectable.** The server could change where a proposal lands while the request schema stays byte-identical. This is the one failure mode that would send an operator to the wrong circle with every check green; the empirical marker is what tells a consumer the landing behaviour is observed rather than contracted.
2. **The reverse direction of condition 7 is unchecked by design.** A registry leaf the record does not name is legitimate — the registry carries the drafting path's other composed leaves. Only the direction that matters (the record naming a read the path cannot run) is asserted; asserting set-equality would require inventing a routing-read delimiter in the registry and would make the guard a second source of truth for which reads the procedure names.
3. **Flags are unguarded.** `me roles`'s pagination limitation, which the Uncertainty field's hedging requirement rests on, has no machine anchor. The hedge is safe under either resolution — if the limitation were lifted, the hedge would merely be conservative.

---

## Consistency Notes

- **No sibling interface files** — one touchpoint. No `interface-cli.md` (the CLI gains no command and no flag) and no `interface-api.md` (no API surface is added). The composed-surface additions are artifact and guard edits, not a CLI contract change.
- **Deliberate divergence from the sibling accord's form**: 072's record carries a `Live facts` manifest and per-fact `Disposition` from a closed vocabulary. This record carries neither, because its content is one rule rather than a set of independently-retiring facts (plan ADR-2). The named-reads block is the structural analogue — the record's own declaration of the surface the guard checks. Flagged rather than silently differing, so a reader comparing the two siblings sees a reason instead of an inconsistency.
- **Extends the single-source + guard idiom** the `plugin/agents/*-commands.txt` family established, and deliberately reuses that family's **token grammar** for the named-reads block so the cross-check compares like with like — the same discipline that made 067's gated-membership check comparable to 063's registry.
- **The premise tripwire is a new anchor shape**: guarding the *absence* of a field by pinning a whole property set. Recorded in DECISIONS.md as generalizable — when a knowledge artifact's premise is that something is absent, pin the whole set rather than grepping for the field you expect to appear.
- **The cite-versus-observe split in the marker is 073-specific.** The landed 072 marker gives all its *facts* one standing — every one is observed behaviour — and treats contract citations as separate pointers alongside them. This record cannot do that: its two halves are both load-bearing content with different standing, because the missing circle parameter is contract while the landing is observation. So the marker must say which is which, or a consumer will trust the wrong half too far.
- **Condition 7 exists because of plan ADR-3.** Under the Shaper's original recommendation the record would only have asserted CLI existence; the developer's decision to widen the composed surface in this spec is what makes record↔registry agreement checkable here rather than in #77.
- **Terminology alias**: spec.md predominantly says "the content" (what the artifact states) where this accord says "the record" (the artifact itself). The spec declares the alias in its System Overview; both name the same object, so a spec non-behavior constraining "the content" is enforced here as a condition on "the record".
- **No `accords/` directory exists** in this repo; project conventions come from PROJECT.md and DECISIONS.md precedent, both honoured above.
