# Source: 073-circle-routing-rule — the recorded routing content (rule, classification test, procedure)

Feature: Circle Routing Rule
  A proposal has no circle of its own. `proposal create` takes an anchor tension
  and nothing else, so the proposal lands in the circle of that tension's
  sensing role — the anchor choice is the routing choice. The consequence is
  counter-intuitive and cost a wasted draft: a change to a circle's own
  governance must be anchored in the parent circle, on a tension sensed by a
  role the operator fills there, with the circle-role itself working when the
  operator is Circle Lead. Circle Routing Rule is the recorded content for that
  question — a hand-authored record owned by the proposal-drafting skill at
  `plugin/skills/proposal-drafting/references/circle-routing-rule.md`, carrying
  the rule, the test that tells an own-circle change from a change to a role
  inside a circle, and the procedure that names which reads answer it. It
  records; it never routes.

  This file covers what the record says and how a consumer reads it. Its sibling
  `circle-routing-guard.feature` covers what the `internal/build` guard enforces
  about it and about the composed surface it names.
  (affects: Practitioner, AI agent)

  Rule: Read where the change must land before spending a gated write on it
    # In order to not spend a gated write on a proposal that lands in a circle
    # that cannot decide it,
    # as an AI agent about to assemble a change set,
    # I want to read which circle this change must land in and which of the
    # operator's tensions can anchor it there, named by id.

    # Source: 073-circle-routing-rule — Scenario: A change to a circle's own governance routes to the parent
    Scenario: An own-circle change routes to the parent circle
      Given the recorded routing content
      When it is consulted for a change to a circle's own domain or policy
      Then its Own-circle consequence field will state the change must be anchored in that circle's parent
      And its Mechanism field will state the proposal lands in the circle of whichever tension anchors it

    # Source: 073-circle-routing-rule — Proposed: interface-spec.md anatomy row 2 (document header)
    Scenario: The document header names its owner and the consumption rule
      Given the recorded routing content
      When its document header is read
      Then its Owner line will name the proposal-drafting skill as the owning skill
      And it will name symlink consumption as how any other skill would consume the record
      And its Contract citations line will name the published API specification

    # Source: 073-circle-routing-rule — Scenario: The content states how to tell the two cases apart
    Scenario: The classification test distinguishes the two cases
      Given the recorded routing content
      When its classification-test section is read
      Then its Test field will distinguish a change to a circle's own governance from a change to a role inside a circle
      And its "Resolved by" field will name "Role.has_subroles" as what resolves whether a target is a circle

    # Source: 073-circle-routing-rule — Scenario: The procedure names the reads and how to state the answer
    Scenario: The procedure names its reads in the order they run
      Given the recorded routing content
      When its named-reads block is read
      Then it will name "me roles", "tension list" and "roles" in the order the procedure runs them
      And its "Answer shape" field will require the target circle named by its role_ id and each eligible anchor named by its ten_ id

    # Source: 073-circle-routing-rule — Scenario: The Circle Lead exception is stated
    Scenario: The circle-role itself anchors when the operator is Circle Lead
      Given the recorded routing content
      When it is consulted for a circle's own governance where the operator fills that circle's Circle Lead role
      Then its "Circle Lead exception" field will state the circle-role itself is a valid anchor site
      And it will not send the operator to the parent circle to find one

    # Source: 073-circle-routing-rule — Scenario: An ordinary change needs no parent hop
    Scenario: A change to a role inside a circle routes without a parent hop
      Given the recorded routing content
      When it is consulted for a change to a role inside a circle rather than to the circle itself
      Then the stated Mechanism will route it to the circle containing that role
      And no separate case will have to be looked up for it

    # Source: 073-circle-routing-rule — Scenario: The target circle has no parent
    Scenario: A circle with no parent yields a stated limit, not a default
      Given the recorded routing content
      When it is consulted for a change to the governance of a circle whose "parent_role_id" is null
      Then its "Root circle" field will state there is no parent circle to route to and that the case is not resolved
      And it will name neither the circle itself nor any other circle as a default target

  Rule: Be told what to do about a missing anchor, not just that one is missing
    # In order to know what to do when I have no usable anchor rather than being
    # told only that I have none,
    # as a practitioner whose agent is drafting on my behalf,
    # I want to read a procedure that requires naming which role in which circle
    # to capture a tension on, and how certain that conclusion is.

    # Source: 073-circle-routing-rule — Scenario: The procedure prescribes what to say when no tension is sensed
    Scenario: A missing anchor is reported with the capture that would close it
      Given the recorded routing content
      When its "Gap reporting" field is read
      Then it will require reporting that no eligible anchor exists yet
      And it will require naming capture on that specific role in that specific circle as the step that closes the gap

    # Source: 073-circle-routing-rule — Scenario: An absence that cannot be proven must be reported as unproven
    Scenario: An unprovable absence is reported as none found, not none existing
      Given the recorded routing content
      When its Uncertainty field is read
      Then it will require reporting "none found" and naming the read the search rested on
      And it will require marking the conclusion uncertain because the own-roles read does not follow pagination

    # Source: 073-circle-routing-rule — Scenario: Nothing the content prescribes stops a write
    Scenario: A routing gap does not stop the operator writing anyway
      Given the recorded routing content was read end to end
      When it is inspected for what it would have a consumer refuse
      Then nothing it prescribes will refuse, block, or delay a proposal create
      And the server will remain the judge of what it accepts

  # --- Validation scenarios ---

  # Source: 073-circle-routing-rule — Scenario: The content carries routing only
  @validation @wip
  Scenario: Only routing is recorded, never change-set shape
    Given the landed record was read end to end
    When its content is enumerated
    Then everything it records will answer where a change lands or what can anchor it there
    And no change-set shape fact owned by the sibling grammar record will appear in it

  # Source: 073-circle-routing-rule — Scenario: Every unprovable absence is prescribed as hedged
  @validation @wip
  Scenario: No statement about a missing role asserts a settled absence
    Given every statement the record makes about finding none of the operator's roles in a circle
    When those statements are read
    Then each will require phrasing as none found in a named read with completeness marked uncertain
    And none will require asserting a settled absence
