# Source: 019-templated-human-rendering — Scenario: a failed read is not rendered through a template

Feature: Unconsumable Output — Templated Human Rendering
  In order to read a command's results as a human without parsing a machine format
  As a practitioner or an AI agent acting on their behalf
  I want command results rendered through named human templates (full and compact)

  Rule: Read results render as labelled human-readable text
    # In order to read a command's result as a human without parsing a machine format,
    # as a practitioner reviewing governance,
    # I want the CLI to render results as labelled, human-readable text.

    # Source: 019-templated-human-rendering — Scenario: a failed read is not rendered through a template
    @wip
    Scenario: Failed read is not rendered through a template
      Given a "my actions" read had failed with a transport error
      When the command reports the failure
      Then the error message will appear on stderr in its cause-plus-next-step form
      And nothing will be written to stdout

    # Source: 019-templated-human-rendering — Scenario: a missing field is omitted, never fabricated
    @wip
    Scenario: Absent embedded collection is omitted
      Given a "me" read without the roles embed
      When the result is rendered with the full template
      Then the rendered output will omit the roles section
      And no empty roles heading will be printed

    # Source: 019-templated-human-rendering — Scenario: an empty result set is legible, not blank
    @wip
    Scenario: Empty result set renders an explicit line
      Given a "my projects" read returned zero projects
      When the result is rendered with the full template
      Then stdout will show the explicit empty line "no projects"
      And no fabricated project row will appear

    # Source: 019-templated-human-rendering — Proposed: a render failure is buffered, never written partially (plan ADR-4)
    @wip
    Scenario: Render failure leaves stdout empty
      Given a built-in template that fails to execute for a "me" result
      When the command renders the result
      Then nothing will be written to stdout
      And the command will exit with the internal-error code

  Rule: Scan a long list with one line per record
    # In order to scan a long list of records quickly,
    # as an AI agent or practitioner triaging output,
    # I want a compact, one-line-per-record rendering.

    # Source: 019-templated-human-rendering — Scenario: compact renders a list one line per record
    @wip
    Scenario: Compact renders one line per role
      Given a "my roles" read returned three roles
      When the result is rendered with the compact template
      Then each role will appear on a single line
      And each line will surface the role's id

    # Source: 019-templated-human-rendering — Scenario: compact counts a nested collection that full expands
    @wip
    Scenario: Compact counts a nested collection
      Given a "me --include roles" read returned an actor filling three roles
      When the result is rendered with the compact template
      Then the actor will appear on a single line showing "roles=3"
      And the line will surface the actor's id

    # Source: 019-templated-human-rendering — Scenario: full and compact render the same record set
    @validation @wip
    Scenario: Full and compact cover the same records
      Given a "my actions" read returned five actions
      When the result is rendered with the full template and again with the compact template
      Then both renderings will account for the same five actions

  Rule: See every field of a record in full
    # In order to see everything a record carries when I need detail,
    # as an operator inspecting a single resource,
    # I want a full rendering that surfaces every field with its ids.

    # Source: 019-templated-human-rendering — Scenario: full preserves the identity projection
    @wip
    Scenario: Full preserves the identity projection
      Given a "me" read returned an actor, organization, and membership
      When the result is rendered with the full template
      Then stdout will show the actor's id, name, and kind, the organization's id and name, and the access level
      And the output will match the projection the command produced before this feature

    # Source: 019-templated-human-rendering — Scenario: full enumerates an embedded collection
    @wip
    Scenario: Full enumerates an embedded collection
      Given a "me --include roles" read whose response carried two roles
      When the result is rendered with the full template
      Then the identity fields will be rendered
      And each embedded role will be listed with its id and name

    # Source: 019-templated-human-rendering — Scenario: full is field-equivalent to the pre-feature projection
    @validation @wip
    Scenario: Full is field-equivalent to the pre-feature projection
      Given each shipped read "me", "my roles", "my actions", and "my projects"
      When their results are rendered with the full template
      Then the surfaced fields will match the projection produced before this feature
      And no field will be dropped or added

    # Source: 019-templated-human-rendering — Scenario: no template introduces a value absent from the source
    @validation @wip
    Scenario: No rendered value is absent from the source
      Given a result rendered with a built-in template
      When the rendered output is inspected
      Then every value shown will trace to a field the result carried
