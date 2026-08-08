# Source: 075-legacy-identifier-request — Scenario: A single role read carries its numeric identifier

Feature: Legacy Identifier Request
  A proposal's change can only name an existing role by its legacy numeric
  identifier — the number the web UI shows in a URL like /orgnav/roles/11079492
  and the change payloads carry as databaseId — and until now no CLI read
  exposed it, so every write touching existing governance sent the operator out
  to the browser mid-flow. The API grew an opt-in answer: an explicit
  --legacy-id flag on the six reads whose contract accepts it (roles, roles
  <id>, tree, actors, actors <id>, me) requests include_legacy_id=true and
  surfaces each resource's number beside its stable identifier.

  This file covers requesting the number and surfacing it. Its siblings:
  legacy-identifier-absence.feature covers what happens when the number is not
  there, and legacy-identifier-guard.feature covers the internal/build guard
  that anchors the flag surface to the vendored contract.
  (affects: Practitioner, AI agent)

  Rule: Target an existing role in a change without leaving the CLI

    # In order to target an existing role in a governance change without
    # breaking flow to read a number off a browser URL,
    # as a practitioner whose governance work the CLI serves,
    # I want to ask a role read for the numeric identifier the change payload
    # needs.

    # Source: 075-legacy-identifier-request — Scenario: A single role read carries its numeric identifier
    @wip
    Scenario: A single role read carries the legacy numeric identifier
      Given a role existed with legacy numeric identifier 14062695
      When the operator runs the roles read for that role with --legacy-id
      Then the output will carry legacy_id 14062695 beside the role's stable role_ identifier
      And that number will be the same number the web UI URL shows for the role

    # Source: 075-legacy-identifier-request — Scenario: The identity read curates the human render and echoes the structured output
    @wip
    Scenario: The identity read's structured output carries every number the response carried
      Given the caller was a human actor in an organization carrying legacy numbers on actor, organization and membership
      When the operator runs the me read with --legacy-id and structured output
      Then the structured document will carry legacy_id on the actor, the organization and the membership

    # Source: 075-legacy-identifier-request — Scenario: The identity read curates the human render and echoes the structured output
    @wip
    Scenario: The identity read's human render shows the actor and organization numbers
      Given the caller was a human actor in an organization carrying legacy numbers on actor, organization and membership
      When the operator runs the me read with --legacy-id and human output
      Then the human render will show the legacy numbers for the actor and the organization
      And the human render will not show the membership's legacy number

    # Source: 075-legacy-identifier-request — Scenario: A failing read is unaffected by the request
    @wip
    Scenario: A failing read is unaffected by the request
      Given a role identifier that did not exist
      When the operator runs the roles read for that identifier with --legacy-id
      Then the CLI will report the not-found exactly as it does without the flag
      And the exit code will be the one that read already produces

    # Source: 075-legacy-identifier-request — Scenario: The request is refused on a read that does not support it
    @wip
    Scenario: An unsupported read refuses the flag before any request
      Given the fillers read is excluded from the legacy-identifier contract
      When the operator runs the fillers read with --legacy-id
      Then the CLI will refuse with an error naming the --legacy-id option
      And it will exit with code 2
      And no request will reach the API

    # Source: 075-legacy-identifier-request — Scenario: Not asking leaves the read byte-for-byte as it was
    @wip
    Scenario: Without the flag the read is byte-identical to before
      Given a role existed with a legacy numeric identifier
      When the operator runs the roles read for that role without --legacy-id
      Then no field, line, or key for a legacy number will appear in any output format
      And the outbound request will carry no include_legacy_id parameter

  Rule: Assemble a change set in one pass instead of failing into a not-found

    # In order to assemble a change set in one pass instead of failing into a
    # not-found and going out to the web UI,
    # as an AI agent drafting a proposal on a practitioner's behalf,
    # I want to obtain a change target's numeric identifier from a read I am
    # already making.

    # Source: 075-legacy-identifier-request — Scenario: One tree read yields a whole subtree's numeric identifiers
    @wip
    Scenario: One tree read yields legacy numbers for a whole subtree
      Given a circle role with sub-roles nested several levels beneath it
      When the operator runs the tree read for that circle with --legacy-id
      Then every row in the rendered tree will carry its legacy number
      And rows at every depth will carry it, not only the root

    # Source: 075-legacy-identifier-request — Proposed: interface-cli.md § User templates (035 surface, no spec scenario)
    @wip
    Scenario: An operator template sees the membership number the built-in render omits
      Given the caller supplied an output template addressing the decoded me result
      When the operator runs the me read with --legacy-id and that template
      Then the template will be able to render the membership's legacy number
      And the built-in human render of the same read will still omit it

    # Source: 075-legacy-identifier-request — Scenario: The number is never used to address a resource
    @validation @wip
    Scenario: No outbound request addresses a resource by its legacy number
      Given the CLI's outbound requests across the read and write surface
      When every request path and identifier argument is inspected
      Then no request will address a resource by a legacy numeric identifier
      And every resource will be addressed by its stable identifier

  Rule: Tell two same-named roles apart when choosing a change target

    # In order to tell two same-named roles apart when choosing a change
    # target,
    # as an AI agent resolving an ambiguous target,
    # I want to see each candidate's numeric identifier next to its stable
    # identifier.

    # Source: 075-legacy-identifier-request — Scenario: Every role in a walked list carries its numeric identifier
    @wip
    Scenario: Every role in a walked list carries its legacy number
      Given an organization whose role list spanned more than one page
      When the operator runs the roles list with --legacy-id
      Then every role in the aggregated result will carry its legacy number beside its stable identifier
      And no page of the walk will be missing the number while another page has it

    # Source: 075-legacy-identifier-request — Proposed: interface-cli.md § per template (actors list rows, no spec scenario)
    @wip
    Scenario: The actor directory carries a legacy number for every actor
      Given an organization with several human actors
      When the operator runs the actors list with --legacy-id and compact output
      Then every actor row will carry a legacy_id segment holding that actor's number

    # Source: 075-legacy-identifier-request — Proposed: interface-cli.md § shared idioms (compact segment, no spec scenario)
    @wip
    Scenario Outline: Compact output of the role family carries the number as a segment
      Given a <subject> carrying a legacy number
      When the operator runs the <read> read with --legacy-id and compact output
      Then the rendered line will carry a legacy_id segment holding the number

      Examples:
        | subject  | read  |
        | role     | roles |
        | tree row | tree  |
