# Source: 074-post-create-validity-read — Scenario: A valid draft reports its verdict alongside its id

Feature: Post-Create Validity Read
  The CLI reports a created proposal and its id as a success while the server has
  already marked the draft invalid with no available transitions, so the operator
  confirms a gated write that produced an object nothing can move forward and
  finds out only later. This capability closes that gap by asking the server
  rather than judging locally: once `glassfrog proposal create` succeeds, the CLI
  reads the created draft back and surfaces the server's own verdict — the
  `valid` flag, every `validation_alerts` entry, and the transitions available on
  it — alongside the returned `prp_` id. Validity, status, alerts, and
  transitions are four independent dimensions and none is ever derived from
  another. A read-back that fails never withholds the created id and never
  becomes the command's failure; rendering an invalid create as a failure exit is
  the separate Invalid-Create Outcome capability.
  (affects: Practitioner, AI agent)

  Rule: Surface the server's verdict in the create result
    # In order to find out at the moment of the write that a confirmed governance change produced an object nothing can move forward,
    # as a practitioner whose agent created a proposal on my behalf,
    # I want to see the server's own verdict on the draft in the result of the create I approved.

    # Source: 074-post-create-validity-read — Scenario: A valid draft reports its verdict alongside its id
    Scenario: A valid created draft reports its verdict with its id
      Given a complete connection context with a stored token
      And the tension "ten_0123" exists
      And the created proposal reads back as valid with no validation alerts
      When an agent runs "glassfrog proposal create ten_0123 --changes '[{\"type\":\"CreatePolicy\",\"name\":\"Deploy windows\"}]'"
      Then the created proposal will be printed with its "prp_" id and "draft" status
      And the result will report the validity as "valid"
      And the create will have been followed by exactly one read of the created proposal
      And the command will exit with code 0

    # Source: 074-post-create-validity-read — Scenario: A created-but-invalid draft surfaces the server's refusal
    @deprecate
    Scenario: A created-but-invalid draft surfaces the server's refusal
      Given a complete connection context with a stored token
      And the tension "ten_0123" exists
      And the created proposal reads back as not valid with the alert "Can't update the Cloud Foundations role during this meeting." and no available transitions
      When an agent runs "glassfrog proposal create ten_0123 --changes '[{\"type\":\"UpdateRole\",\"databaseId\":\"14067864\"}]'"
      Then the created proposal will be printed with its "prp_" id
      And the result will report the validity as "not valid"
      And the result will carry the server's alert message with its severity and path
      And the result will report that no transitions are available
      And the command will exit with code 0

    # Source: 074-post-create-validity-read — Scenario: The create itself is rejected
    Scenario: A rejected create is reported without a read-back
      Given a complete connection context with a stored token
      And the proposals endpoint rejects the create
      When an agent runs "glassfrog proposal create ten_0123 --changes '[{\"type\":\"CreateRole\"}]'"
      Then stderr will report that the create failed and name the HTTP status
      And no read of any proposal will be attempted
      And the command will exit with a non-zero API-error code

    # Source: 074-post-create-validity-read — Scenario: The server reports no verdict at all
    Scenario: A draft the server states no verdict on is reported as unreported
      Given a complete connection context with a stored token
      And the created proposal reads back carrying no validity field
      When an agent runs "glassfrog proposal create ten_0123 --changes '[{\"type\":\"CreateRole\",\"name\":\"Scribe\"}]'"
      Then the result will report that the server stated no verdict on the draft
      And the result will describe the draft as neither valid nor not valid
      And the command will exit with code 0

    # Source: 074-post-create-validity-read — Scenario: A valid draft with no available transitions
    Scenario: A valid draft with no transitions keeps the two facts distinct
      Given a complete connection context with a stored token
      And the created proposal reads back as valid with no available transitions
      When an agent runs "glassfrog proposal create ten_0123 --changes '[{\"type\":\"CreateRole\",\"name\":\"Scribe\"}]'"
      Then the result will report the validity as "valid"
      And the result will report that no transitions are available
      And neither fact will be restated as the other
      And the command will exit with code 0

    # Source: 074-post-create-validity-read — Scenario: A status that disagrees with the verdict
    Scenario: A conflicted status and a favourable verdict are both reported as given
      Given a complete connection context with a stored token
      And the created proposal reads back with status "draft_with_conflicts" and as valid
      When an agent runs "glassfrog proposal create ten_0123 --changes '[{\"type\":\"CreateRole\",\"name\":\"Scribe\"}]'"
      Then the result will report the status "draft_with_conflicts" as the server gave it
      And the result will report the validity as "valid"
      And neither will be adjusted to agree with the other
      And the command will exit with code 0

    # Source: 074-post-create-validity-read — Scenario: A valid draft carrying an alert reports both as they stand
    Scenario: A valid draft carrying an advisory alert reports both facts
      Given a complete connection context with a stored token
      And the created proposal reads back as valid with one alert of severity "warning"
      When an agent runs "glassfrog proposal create ten_0123 --changes '[{\"type\":\"CreateRole\",\"name\":\"Scribe\"}]'"
      Then the result will report the validity as "valid"
      And the result will carry the alert with its severity, path, and message
      And the alert's presence will not be reported as an unfavourable verdict
      And the command will exit with code 0

    # Source: 074-post-create-validity-read — Scenario: Provenance of the reported result is legible
    @validation @wip
    Scenario: The reported result names the read that produced the verdict
      Given a reader of the create result who was not present for the write
      When they read a create result carrying a verdict
      Then the result will name the created proposal the verdict was read back from
      And it will be legible that a read after the create produced it

    # Source: 074-post-create-validity-read — Scenario: No local validity derivation appears anywhere
    @validation @wip
    Scenario: No verdict is derived from the change set, status, transitions, or alerts
      Given the implemented create command and its verdict rendering
      When they are inspected for any local derivation of validity
      Then every reported verdict will be attributed to the server's own fields
      And no verdict will be computed from the change set, the status, the transition set, or the presence of alerts

    # Source: 074-post-create-validity-read — Scenario: No undeclared field is presented as contract
    @validation @wip
    Scenario: The verdict fields are never presented as published contract
      Given the "valid" and "validation_alerts" fields are absent from the vendored v5 contract
      When the command's help, output, and documentation are inspected
      Then neither field will be described as part of the published contract

  Rule: Carry the verdict in the output an agent already parses
    # In order to decide the next step without issuing a second command to find out whether the write meant anything,
    # as an AI agent driving the governance write path,
    # I want to read the created draft's validity, its alerts, and its available transitions out of the create output I already parse.

    # Source: 074-post-create-validity-read — Scenario: An agent parses the verdict out of machine-readable output
    @deprecate
    Scenario: Structured output carries the verdict alongside the created id
      Given a complete connection context with a stored token
      And the created proposal reads back as not valid with one validation alert
      When an agent runs "glassfrog proposal create ten_0123 --changes '[{\"type\":\"UpdateRole\",\"databaseId\":\"14067864\"}]' --output json"
      Then the structured result will contain the created proposal's "prp_" id
      And the structured result will carry "valid", "validation_alerts", and "available_transitions"
      And the advisory will be rendered in the selected machine format, carrying "read_back" as true
      And the command will exit with code 0

    # Source: 074-post-create-validity-read — Proposed: plan ADR-5 amendment (the advisory is format-aware, so no state needs prose)
    Scenario: An unobtainable verdict is machine-readable in a machine format
      Given a complete connection context with a stored token
      And the create succeeds but the read of the created proposal cannot reach the server
      When an agent runs "glassfrog proposal create ten_0123 --changes '[{\"type\":\"CreateRole\",\"name\":\"Scribe\"}]' --output json"
      Then the structured result will contain the created proposal's "prp_" id
      And the advisory will be rendered in the selected machine format, carrying "read_back" as false
      And the advisory will carry the reason and the remedy naming "glassfrog proposal get"
      And no part of the four verdict states will require reading prose to identify
      And the command will exit with code 0

    # Source: 074-post-create-validity-read — Proposed: interface-cli.md § "stdout — human compact format" (a defined surface with no scenario)
    @deprecate
    Scenario: The compact line carries the validity token
      Given a complete connection context with a stored token
      And the created proposal reads back as not valid with one validation alert
      When an agent runs "glassfrog proposal create ten_0123 --changes '[{\"type\":\"UpdateRole\",\"databaseId\":\"14067864\"}]' --output compact"
      Then the compact line will carry the created "prp_" id, its status, and the change count
      And it will carry the validity token and the alert count
      And the command will exit with code 0

    # Source: 074-post-create-validity-read — Proposed: plan ADR-4 § Consequences (the view type changes under an existing user template)
    Scenario: A user template written before the verdict still renders
      Given a complete connection context with a stored token
      And a user template referencing only the proposal fields the create rendered before the verdict existed
      When an agent creates a proposal with that template selected
      Then every field path in the template will still resolve
      And the template's output will carry the read-back's values through those same paths
      And the command will exit with code 0

    # Source: 074-post-create-validity-read — Proposed: plan Risk 3 (the shared singular proposal template is the drift surface ADR-4 routes around)
    Scenario: The sibling proposal commands render no verdict
      Given a complete connection context with a stored token
      When an agent reads, advances, and withdraws a proposal whose read carries a validity field
      Then none of those results will render a validity, alert, or verdict-source line
      And their output will be unchanged from before the create gained its verdict

  Rule: Keep the created proposal's id when the verdict cannot be obtained
    # In order to keep the handle on a proposal that really was created when the follow-up read fails,
    # as an AI agent,
    # I want to still receive the prp_ id, together with an explicit statement that the verdict is unknown.

    # Source: 074-post-create-validity-read — Scenario: The read-back cannot reach the server
    Scenario: An unreachable read-back still reports the created id
      Given a complete connection context with a stored token
      And the create succeeds but the read of the created proposal cannot reach the server
      When an agent runs "glassfrog proposal create ten_0123 --changes '[{\"type\":\"CreateRole\",\"name\":\"Scribe\"}]'"
      Then the created proposal will be printed with its "prp_" id
      And the result will report that the verdict could not be obtained and name the cause
      And the result will describe the draft as neither valid nor not valid
      And the command will exit with code 0

    # Source: 074-post-create-validity-read — Scenario: The read-back exhausts the hour's request budget
    Scenario: A rate-limited read-back reports the exhausted budget with the created id
      Given a complete connection context with a stored token
      And the create succeeds but the read of the created proposal is rate-limited after its retries
      When an agent runs "glassfrog proposal create ten_0123 --changes '[{\"type\":\"CreateRole\",\"name\":\"Scribe\"}]'"
      Then the created proposal will be printed with its "prp_" id
      And the result will report that the verdict could not be obtained because the request budget was exhausted
      And the command will exit with code 0

    # Source: 074-post-create-validity-read — Proposed: plan ADR-6 consequence (a 2xx create body from which no prp_ id can be lifted)
    Scenario: A create response carrying no id yields no read-back
      Given a complete connection context with a stored token
      And the create answers with a success body carrying no "prp_" id
      When an agent runs "glassfrog proposal create ten_0123 --changes '[{\"type\":\"CreateRole\",\"name\":\"Scribe\"}]'"
      Then no read of any proposal will be attempted
      And the result will report that the created proposal's id could not be determined
      And the create response will still be reported

    # Source: 074-post-create-validity-read — Scenario: Every verdict state is distinguishable in every output format
    @validation @wip @deprecate
    Scenario: Every verdict state is distinguishable in every output format
      Given each output format the create itself composes — the two human formats and the two machine formats
      When the server states a favourable verdict, states an unfavourable verdict, states no verdict, and the read-back fails
      Then all four outcomes will be distinguishable from one another in that format without inference
      And in a machine format none of the four will require reading human prose to identify
      And under a caller-authored template the four states will be available to be distinguished rather than rendered
