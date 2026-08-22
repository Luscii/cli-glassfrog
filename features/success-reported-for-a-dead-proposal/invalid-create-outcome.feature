# Source: 078-invalid-create-outcome — Scenario: An invalid draft terminates the create as a failure

Feature: Invalid-Create Outcome
  The CLI reports a created proposal and its id as a success while the server has
  already marked the draft invalid, so an agent branching on the exit code and a
  CI job both read a dead write as a pass. Post-Create Validity Read surfaced the
  server's verdict in the create result; this capability makes that verdict cost
  the create its success: when the read-back carries an explicit unfavourable
  verdict, the create terminates as a failure with its own previously-unused exit
  code, still carrying the created `prp_` id and every validation alert the
  server attached. Only the server's explicit verdict trips it — a missing
  verdict, a failed read-back, and a valid draft carrying alerts all remain
  successes exactly as they land today.
  (affects: Practitioner, AI agent)

  Rule: Branch on the exit code instead of parsing output
    # In order to learn at the moment of the write that a confirmed governance change produced a dead object, without reading command output to find out,
    # as an AI agent driving the governance write path,
    # I want an accepted-but-invalid create to exit with its own failure code, so I can branch on $? and stop the sequence.

    # Source: 078-invalid-create-outcome — Scenario: An invalid draft terminates the create as a failure
    Scenario: An invalid draft fails the create with its own exit code
      Given a complete connection context with a stored token
      And the created proposal reads back as not valid with the alert "Can't update the Cloud Foundations role during this meeting." and no available transitions
      When an agent runs "glassfrog proposal create ten_0123 --changes '[{\"type\":\"UpdateRole\",\"databaseId\":\"14067864\"}]'"
      Then stderr will carry the created "prp_" id
      And stderr will carry each alert with its severity, path, and message
      And stderr will name creating a corrected proposal as the next step
      And the create will have been followed by exactly one read of the created proposal
      And the command will exit with code 8

    # Source: 078-invalid-create-outcome — Scenario: A machine-readable failure is fully structured
    Scenario: A machine-readable failure carries the structured envelope
      Given a complete connection context with a stored token
      And the created proposal reads back as not valid with one validation alert
      When an agent runs "glassfrog proposal create ten_0123 --changes '[{\"type\":\"UpdateRole\",\"databaseId\":\"14067864\"}]' --output json"
      Then stdout will carry the failure envelope with kind "invalid-create"
      And the envelope will carry "proposal_id" and "validation_alerts"
      And the envelope will carry the cause naming the created "prp_" id
      And the envelope will carry the remedy as its own field
      And no server proposal document will be emitted
      And no verdict advisory will be emitted
      And the command will exit with code 8

    # Source: 078-invalid-create-outcome — Proposed: interface-cli.md § "stderr — human formats" (both human formats render the failure identically)
    Scenario: The compact format fails the create the same way
      Given a complete connection context with a stored token
      And the created proposal reads back as not valid with one validation alert
      When an agent runs "glassfrog proposal create ten_0123 --changes '[{\"type\":\"UpdateRole\",\"databaseId\":\"14067864\"}]' --output compact"
      Then stderr will carry the created "prp_" id
      And stderr will carry each alert with its severity, path, and message
      And stdout will be empty
      And the command will exit with code 8

    # Source: 078-invalid-create-outcome — Proposed: interface-cli.md § stdout field table (an invalid draft with zero alerts still fails; the key is absent, not empty)
    Scenario: An invalid draft with no alerts still fails
      Given a complete connection context with a stored token
      And the created proposal reads back as not valid with no validation alerts
      When an agent runs "glassfrog proposal create ten_0123 --changes '[{\"type\":\"UpdateRole\",\"databaseId\":\"14067864\"}]' --output json"
      Then stdout will carry the failure envelope with kind "invalid-create"
      And the envelope will carry "proposal_id"
      And the envelope will not carry a "validation_alerts" key
      And the command will exit with code 8

    # Source: 078-invalid-create-outcome — Scenario: The failure keys on the verdict, not the transitions
    Scenario: The failure keys on the verdict, not the transitions
      Given a complete connection context with a stored token
      And the created proposal reads back as not valid with one validation alert and a non-empty transition set
      When an agent runs "glassfrog proposal create ten_0123 --changes '[{\"type\":\"UpdateRole\",\"databaseId\":\"14067864\"}]'"
      Then stderr will carry the created "prp_" id
      And the command will exit with code 8

    # Source: 078-invalid-create-outcome — Scenario: A missing verdict leaves the create a success
    Scenario: A missing verdict leaves the create a success
      Given a complete connection context with a stored token
      And the created proposal reads back carrying no validity field
      When an agent runs "glassfrog proposal create ten_0123 --changes '[{\"type\":\"CreateRole\",\"name\":\"Scribe\"}]'"
      Then the result will report that the server stated no verdict on the draft
      And no failure envelope will be emitted
      And the command will exit with code 0

    # Source: 078-invalid-create-outcome — Scenario: A failed read-back leaves the create a success
    Scenario: A failed read-back leaves the create a success
      Given a complete connection context with a stored token
      And the create succeeds but the read of the created proposal cannot reach the server
      When an agent runs "glassfrog proposal create ten_0123 --changes '[{\"type\":\"CreateRole\",\"name\":\"Scribe\"}]'"
      Then the created proposal will be printed with its "prp_" id
      And no failure envelope will be emitted
      And the command will exit with code 0

    # Source: 078-invalid-create-outcome — Scenario: The new code never renumbers an existing one
    Scenario: The invalid-create code is new and renumbers nothing
      Given the exit-code registry with its existing assigned codes
      When the invalid-create category is added to it
      Then the category will take a previously-unused code
      And every existing category will keep the code it had before

    # Source: 078-invalid-create-outcome — Scenario: A valid draft still succeeds (re-pin of the structured success path)
    Scenario: Structured output carries the verdict for a valid create
      Given a complete connection context with a stored token
      And the created proposal reads back as valid with one alert of severity "warning"
      When an agent runs "glassfrog proposal create ten_0123 --changes '[{\"type\":\"CreateRole\",\"name\":\"Scribe\"}]' --output json"
      Then the structured result will contain the created proposal's "prp_" id
      And the structured result will carry "valid", "validation_alerts", and "available_transitions"
      And the advisory will be rendered in the selected machine format, carrying "read_back" as true
      And the command will exit with code 0

    # Source: 078-invalid-create-outcome — Scenario: A valid draft carrying alerts still succeeds (re-pin of the compact token line)
    Scenario: The compact line carries the validity token for a valid create
      Given a complete connection context with a stored token
      And the created proposal reads back as valid with one alert of severity "warning"
      When an agent runs "glassfrog proposal create ten_0123 --changes '[{\"type\":\"CreateRole\",\"name\":\"Scribe\"}]' --output compact"
      Then the compact line will carry the created "prp_" id, its status, and the change count
      And it will carry the validity token and the alert count
      And the command will exit with code 0

    # Source: 078-invalid-create-outcome — Proposed: plan ADR-3 (the two envelope keys are omitempty; only this failure populates them)
    Scenario: Other failures carry no invalid-create envelope keys
      Given a complete connection context with a stored token
      And the proposals endpoint rejects the create
      When an agent runs "glassfrog proposal create ten_0123 --changes '[{\"type\":\"CreateRole\"}]' --output json"
      Then stdout will carry the failure envelope for the rejected create
      And the envelope will carry neither "proposal_id" nor "validation_alerts"
      And the command will exit with a non-zero API-error code

    # Source: 078-invalid-create-outcome — Scenario: The failure keys only on the explicit server verdict
    @validation @wip
    Scenario: No failure trigger exists besides the server's explicit verdict
      Given the implemented create command and its outcome classification
      When it is inspected for what raises the invalid-create failure
      Then the only trigger will be the server's explicit "valid" being false
      And no failure will be raised from the status, the transition set, the presence of alerts, or the change-set shape

    # Source: 078-invalid-create-outcome — Scenario: The new code is one-to-one and never reassigned
    @validation @wip
    Scenario: The registry stays one-to-one after the new code
      Given the exit-code registry after the invalid-create category is added
      When each category is matched to its code and back
      Then the invalid-create category will map to exactly one previously-unused code
      And no existing category's code will have changed

    # Source: 078-invalid-create-outcome — Scenario: The failure is distinguishable from every success state in a machine format
    @validation @wip
    Scenario: The failure is distinguishable from every success state
      Given a machine-readable output format
      When the server reports the draft not valid, reports it valid, reports it valid with alerts, reports no verdict, and the read-back fails, in turn
      Then the invalid-create failure will be distinguishable from each of the three success states without inference
      And a valid draft carrying alerts will be recognizable as the valid state rather than a state of its own
      And none of them will require reading human prose to identify

  Rule: Fail the build when the write produced a dead draft
    # In order to keep a build honest when a proposal it created can never move forward,
    # as a CI pipeline,
    # I want an invalid create to exit non-zero, so the job fails instead of passing on a dead draft.

    # Source: 078-invalid-create-outcome — Scenario: A valid draft still succeeds
    Scenario: A valid draft still succeeds
      Given a complete connection context with a stored token
      And the created proposal reads back as valid with no validation alerts
      When an agent runs "glassfrog proposal create ten_0123 --changes '[{\"type\":\"CreatePolicy\",\"name\":\"Deploy windows\"}]'"
      Then the created proposal will be printed with its "prp_" id and "draft" status
      And the result will report the validity as "valid"
      And the command will exit with code 0

    # Source: 078-invalid-create-outcome — Scenario: A valid draft carrying alerts still succeeds
    Scenario: A valid draft carrying alerts still succeeds
      Given a complete connection context with a stored token
      And the created proposal reads back as valid with one alert of severity "warning"
      When an agent runs "glassfrog proposal create ten_0123 --changes '[{\"type\":\"CreateRole\",\"name\":\"Scribe\"}]'"
      Then the result will report the validity as "valid"
      And the result will carry the alert with its severity, path, and message
      And no failure envelope will be emitted
      And the command will exit with code 0

  Rule: Keep the handle and the reasons on the failed create
    # In order to find and clean up the dead draft the write left behind,
    # as a practitioner whose agent created a proposal on my behalf,
    # I want the failure to still show me the created prp_ id and the server's reasons.

    # Source: 078-invalid-create-outcome — Proposed: interface-cli.md § "stdout — user template" (failures bypass template rendering)
    Scenario: A user template does not render the failure
      Given a complete connection context with a stored token
      And a user template referencing only proposal fields
      And the created proposal reads back as not valid with one validation alert
      When an agent creates a proposal with that template selected
      Then the template will not be rendered
      And stderr will carry the created "prp_" id
      And the command will exit with code 8

    # Source: 078-invalid-create-outcome — Scenario: The created id survives every failure path
    @validation @wip
    Scenario: The created id survives every failure path
      Given each path on which the create terminates as an invalid-create failure
      When the result is reported
      Then the created "prp_" id will be present in it
