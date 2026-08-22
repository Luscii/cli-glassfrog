# Source: 079-pre-assembly-grammar-consultation — Scenario: The target circle and its eligible anchors are established first

Feature: Pre-Assembly Routing Application
  A proposal carries no circle of its own and lands in the circle of its
  anchor tension's sensing role, so a change to a circle's own governance must
  be proposed from the parent circle. The routing record states that rule;
  this is its application — before an anchor is settled, the drafting path
  determines where the change lands and which anchors can route it there, and
  reports rather than enforces what it finds.

  Rule: Establish the target circle and eligible anchors before an anchor is settled
    # In order to not spend a confirmed governance write on a proposal that
    # lands in a circle which cannot decide it,
    # as a practitioner served through the agent,
    # I want the target circle and the anchors eligible to reach it
    # established before an anchor is settled on.

    # Source: 079-pre-assembly-grammar-consultation — Scenario: The target circle and its eligible anchors are established first
    @wip
    Scenario: Routing names the target circle and every eligible anchor, choosing none
      Given an intended change and no anchor settled on
      When the drafter runs the recorded routing procedure's reads in their order
      Then it will report the target circle's role_ id and every eligible anchor's ten_ id
      And it will return action named-anchors, choosing none — the choice is the practitioner's

    # Source: 079-pre-assembly-grammar-consultation — Scenario: A handed-in anchor routes the change elsewhere
    @wip
    Scenario: A mismatched handed-in anchor is reported, not drafted on silently
      Given a handed-in anchor whose determination landed the change outside the target circle
      When the drafter evaluates the handed-in anchor
      Then it will return action surfaced-routing-mismatch naming the eligible anchors that reach the target circle
      And drafting will proceed on the handed-in anchor where the practitioner directs it — the mismatch is reported, not enforced

    # Source: 079-pre-assembly-grammar-consultation — Scenario: No eligible anchor exists yet
    @wip
    Scenario: An empty eligible set names capture as the closing step
      Given a target circle where the operator filled a role but no tension was sensed on it
      When the routing determination reports
      Then it will return action named-anchors with an empty eligible set
      And the capture-gap note will name capture on that specific role in that specific circle as the step that closes the gap, handed onward rather than performed

    # Source: 079-pre-assembly-grammar-consultation — Scenario: A routing read fails part-way
    @wip
    Scenario: An incomplete routing walk continues flagged, inventing nothing
      Given a routing determination whose reads failed before the procedure completed
      When the drafter reports its answer
      Then it will name what failed and present the determination as incomplete in the consultation element
      And it will continue on what was established, neither inventing the unread part nor abandoning it

    # Source: 079-pre-assembly-grammar-consultation — Proposed: spec accord (no containing circle) + interface-spec.md error communication — the record's decline is reported, no target invented
    @wip
    Scenario: A root circle's missing parent is declined, not resolved
      Given a change to the governance of a circle whose parent_role_id was null
      When the routing determination reports
      Then the routing part of the consultation element will carry the record's decline that no target is resolved for that case
      And no target circle will be invented or chosen in its place

    # Source: 079-pre-assembly-grammar-consultation — Proposed: interface-spec.md invocation contract — the widened descriptions state the routed entry
    @wip
    Scenario: Both artifact descriptions state the routed entry
      Given the proposal-drafting skill and the proposal-drafter agent after the gate landed
      When their frontmatter descriptions are read
      Then each description will state that the path determines where the change lands before an anchor is settled on
      And every boundary sentence the descriptions carried before will still be present

    # Source: 079-pre-assembly-grammar-consultation — Scenario: Nothing withholds a write locally
    @validation @wip
    Scenario: Every surfaced finding leaves the decision with the practitioner
      Given the wired workflow and everything consultation can surface
      When it is inspected for a refusal, block, filter, delay, or withheld draft applied before the server sees the create
      Then none will be present — every surfaced finding will leave the decision with the practitioner
