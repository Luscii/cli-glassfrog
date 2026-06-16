# Source: 061-plan-limit-signal — Scenario: Advancing a draft on a non-Premium org surfaces an actionable plan-limit diagnostic

Feature: Unsignalled Plan Limits — Plan-Limit Signal
  A plan-gated endpoint rejects with a 403 that carries no field marking it as a
  plan limit, so a "not available on your plan" rejection reads as an ordinary
  permission error. Feature-Gate Recognition tags such a 403 as a possible plan
  limit naming the suspected gate; Plan-Limit Signal turns that classification
  into an actionable diagnostic — a cause that names the gating feature and frames
  it as a possibility, plus a next step to verify the plan — rendered in the
  selected output format, with the gating feature a distinct element under the
  structured formats and the exit code unchanged.
  (affects: Practitioner)

  Rule: A recognized plan-limit rejection is an actionable diagnostic naming the gating feature
    # In order to be told "this may not be on your plan" and what to check, instead of a bare permission error when a proposal write is rejected,
    # as a practitioner whose AI agent drives the CLI,
    # I want a recognized plan-limit 403 surfaced as an actionable diagnostic that names the gating feature.

    # Source: 061-plan-limit-signal — Scenario: Advancing a draft on a non-Premium org surfaces an actionable plan-limit diagnostic
    Scenario: A recognized 403 from advancing a draft names the gating feature and a next step
      Given the advance-to-circulation operation POST /proposals/prp_0123/propose had been rejected with HTTP status 403
      And the operation is a known plan-gated operation
      When the failure is rendered under the full format
      Then the diagnostic will name Premium async proposals as the gating feature
      And it will state the operation may not be available on the organization's plan
      And it will point the caller to verify the plan includes Premium async proposals

    # Source: 061-plan-limit-signal — Scenario: Creating a proposal surfaces the gate, framed as a possibility
    Scenario: A recognized 403 from creating a proposal names the gate as a possibility
      Given the create-proposal operation POST /proposals had been rejected with HTTP status 403
      And the operation is a known plan-gated operation
      When the failure is rendered
      Then the diagnostic will name Premium async proposals as the gating feature
      And it will frame the plan limit as a possibility, not a certainty

    # Source: 061-plan-limit-signal — Scenario: A 403 that was not recognized keeps the generic diagnostic
    Scenario: A non-recognized 403 keeps the generic permission diagnostic
      Given the role-read operation GET /roles/role_0123 had been rejected with HTTP status 403
      When the failure is rendered
      Then the diagnostic will be the generic permission denial
      And it will name no gating feature
      And the structured envelope will carry no feature element

    # Source: 061-plan-limit-signal — Scenario: A non-403 failure on a gated operation gets no plan-limit wording
    Scenario: A non-403 failure on a gated operation gets no plan-limit wording
      Given the create-proposal operation POST /proposals had been rejected with HTTP status 422 for an invalid change
      When the failure is rendered
      Then the diagnostic will carry no plan-limit wording
      And it will name no gating feature

    # Source: 061-plan-limit-signal — Scenario: The exit code is unchanged by the plan-limit wording
    Scenario: A recognized plan-limit 403 keeps the permission exit code across formats
      Given the advance-to-circulation operation POST /proposals/prp_0123/propose had been recognized as a plan-limit 403
      When the failure is rendered under each output format
      Then every invocation will terminate with the permission exit code 4
      And only the rendered presentation will differ between formats

    # Source: 061-plan-limit-signal — Scenario: A modeled ai_integration gate produces no message today
    Scenario: The ai_integration gate produces no plan-limit message today
      Given the ai_integration gate kind is modeled but reached by no command
      When commands are run
      Then no failure will render an ai_integration plan-limit message today
      And the signal will be ready to name that gate if such a command is later added

  Rule: The gating feature is surfaced as a distinct, parseable element under structured output
    # In order to branch on a plan limit programmatically rather than parse it out of prose,
    # as an AI agent operating the CLI with --output json,
    # I want the gating feature surfaced as a distinct, parseable element of the failure envelope.

    # Source: 061-plan-limit-signal — Scenario: The gating feature is a distinct, parseable element under json
    Scenario: The gating feature is a distinct feature element under json
      Given the create-proposal operation POST /proposals had been recognized as a plan-limit 403
      When the failure is rendered under the json format
      Then the error envelope on stdout will carry a distinct feature element naming Premium async proposals
      And the gate name will not be folded only into the message text

  Rule: The plan-limit wording is framed as a possibility, never a certain insufficiency
    # In order to trust the signal and not be sent to upgrade a plan that is already sufficient,
    # as an operator (human or agent),
    # I want the plan-limit wording framed as a possibility, never a certain insufficiency.

    # Source: 061-plan-limit-signal — Scenario: A genuine permission denial on a gated operation is still hedged, never asserted
    Scenario: A genuine permission denial on a gated operation is hedged, never asserted
      Given the advance-to-circulation operation POST /proposals/prp_0123/propose had been rejected with HTTP status 403
      And the rejection was a genuine permission denial unrelated to the plan
      When the failure is rendered
      Then the diagnostic will frame the plan limit as a possibility
      And it will note the rejection may instead be a permission issue
      And it will not state the plan is certainly insufficient

    # Source: 061-plan-limit-signal — Scenario: Possibility is preserved everywhere the signal renders
    @validation @wip
    Scenario: Every rendered plan-limit failure frames the limit as a possibility
      Given a recognized plan-limit 403 rendered under each output format
      When each rendered failure is inspected
      Then it will frame the plan limit as a possibility naming the gate
      And none will instruct the caller to upgrade as the sole remedy

    # Source: 061-plan-limit-signal — Scenario: No fabricated remedy detail in the rendered diagnostic
    @validation @wip
    Scenario: A rendered plan-limit failure invents no remedy detail
      Given a recognized plan-limit 403 rendered under each output format
      When each rendered failure is inspected
      Then it will name no plan price, no upgrade URL, and no plan name beyond the gating feature
      And the next step will be a verifiable action
