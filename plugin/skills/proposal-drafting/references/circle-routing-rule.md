> **Empirical record.** The two halves of this record carry different standing,
> and the split matters. That `proposal create` carries **no circle parameter**
> — the request takes an anchor tension and an optional change set, nothing
> more — is published contract, cited below rather than restated. Where the
> proposal consequently **lands** — the circle of the anchor tension's sensing
> role, the own-circle consequence, and the Circle Lead exception — is observed
> server behaviour and no part of the published v5 contract. Check the first
> half against the contract; treat the second as verified observation that may
> change when the server changes.

# Circle Routing Rule

- **Owner**: the proposal-drafting skill. This record lives in that skill's
  `references/` directory as the single source; any other skill that needs it
  consumes it through a symbolic link into its own directory, never a copy.
- **Contract citations**: `spec/glassfrog-api-v5.yaml` — cited by schema and
  property name only, never by line number and never by restated values.

## Contract citations

Where the published contract already speaks, this record cites it rather than
restating it. These are pointers into the contract, not restatements of it.

- **Premise**: `CreateProposalRequest.properties.proposal` requires only
  `tension_id`, optionally carries `changes`, and has **no circle property** —
  no request field says where the proposal should land.
- **Circle indicator**: `Role.has_subroles` resolves whether a target is a
  circle.
- **Root signal**: `Role.parent_role_id` is nullable; a null value is the
  root-circle signal the Rule section's Root circle field rests on.

## Rule

- **Mechanism**: *(observed)* a proposal inherits the circle of its anchor
  tension's sensing role — it lands in the circle containing that role, and in
  no other. The create carries no circle parameter (the premise above), so the
  anchor choice **is** the routing choice. This is the whole rule: a change to
  a role inside a circle anchors on a tension sensed in that circle and lands
  there — no separate case exists for it.
- **Own-circle consequence**: *(observed)* a change to a circle's own
  governance — the circle-role itself, or a domain or policy it holds — must
  be anchored in that circle's **parent** circle, on a tension sensed by a
  role the operator fills there.
- **Circle Lead exception**: *(observed; carried as recorded from LEARNINGS
  2026-08-05 F7, not independently re-verified)* where the operator fills the
  target circle's own Circle Lead role, the circle-role itself is a valid
  anchor site — the operator need not go to the parent circle to find one.
- **Root circle**: where the target circle's `Role.parent_role_id` is null,
  there is no parent circle to route to and the case is not resolved by this
  record — it names no default target, neither the circle itself nor any
  other circle.

## Classification test

- **Test**: classify from the change target alone. A change to a circle's own
  governance targets the circle-role itself or what that circle-role holds —
  its own domains or policies. A change to a role inside a circle targets one
  of the circle's sub-roles. The first routes through the parent (the
  Own-circle consequence); the second follows the Mechanism unchanged.
- **Resolved by**: `Role.has_subroles` resolves whether a target is a circle —
  a role expanded with sub-roles is a circle.
- **Parent resolution**: `Role.parent_role_id` names the containing role. The
  own-roles read (`me roles`) already carries it for every role the operator
  fills, so the role read is needed only when the target circle is not among
  them.

## Procedure

Run the reads in this order; each feeds the next:

```
me roles
tension list
roles
```

`me roles` establishes which roles the operator fills and each role's
`parent_role_id`; `tension list` establishes which tensions those roles sense
— the candidate anchors; `roles` resolves a target's classification and parent
only when the target circle is not among the operator's own roles.

- **Answer shape**: the target circle named by its `role_` id; each eligible
  anchor tension named by its `ten_` id.
- **All anchors named**: where several anchor tensions are eligible, all of
  them are named and none is chosen — choosing is the operator's act, not the
  procedure's.
- **Gap reporting**: where the operator fills a role in the target circle but
  no tension is sensed on it, report that no eligible anchor exists yet and
  name capture on that specific role in that specific circle as the step that
  closes the gap.
- **Uncertainty**: where an answer rests on the own-roles read, report **none
  found** in `me roles` — naming the read the search rested on — and mark the
  conclusion's completeness uncertain: `me roles` does not follow pagination,
  so an absence in it is an absence in what was read, never a settled absence.

A routing gap is reported, never enforced: nothing in this record refuses,
blocks, or delays a `proposal create` — the server remains the judge of what
it accepts.
