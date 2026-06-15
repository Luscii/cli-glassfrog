# Source: 060-feature-gate-recognition — Scenario: Advancing a draft on a non-Premium org is recognized as a plan limit

Feature: Unsignalled Plan Limits — Feature-Gate Recognition
  Plan- and feature-gated endpoints reject with a 403 that carries no field
  marking it as a plan limit, so a "not available on your plan" rejection looks
  identical to an ordinary permission denial. Feature-Gate Recognition uses the
  spec's static gate metadata to identify when a 403 came from a known
  plan/feature-gated operation — chiefly the Premium async-proposal write path —
  so the rejection is distinguishable from a generic permission denial. It
  expresses possibility, never certainty, and renders nothing itself.
  (affects: Practitioner)

  Rule: Plan-gated 403s are recognized as distinct from ordinary permission denials
    # In order to be told "this isn't on your plan" instead of a bare permission error when a proposal write is rejected,
    # as a practitioner whose AI agent drives the CLI,
    # I want plan-gated 403s recognized as distinct from ordinary permission denials.

    # Source: 060-feature-gate-recognition — Scenario: Advancing a draft on a non-Premium org is recognized as a plan limit
    Scenario: A 403 from advancing a draft is recognized as a possible plan limit
      Given the advance-to-circulation operation POST /proposals/prp_0123/propose had been rejected with HTTP status 403
      When the failure is checked for a feature gate
      Then it will be recognized as a possible plan-limit rejection
      And the suspected gate will be named as Premium async proposals

    # Source: 060-feature-gate-recognition — Scenario: Creating a proposal on a non-Premium org is recognized as a plan limit
    Scenario: A 403 from creating a proposal is recognized as a possible plan limit
      Given the create-proposal operation POST /proposals had been rejected with HTTP status 403
      When the failure is checked for a feature gate
      Then it will be recognized as a possible plan-limit rejection naming Premium async proposals

    # Source: 060-feature-gate-recognition — Scenario: Recording a response on a non-Premium org is recognized as a plan limit
    Scenario: A 403 from recording a response is recognized as a possible plan limit
      Given the record-response operation POST /proposals/prp_0123/responses had been rejected with HTTP status 403
      When the failure is checked for a feature gate
      Then it will be recognized as a possible plan-limit rejection naming Premium async proposals

    # Source: 060-feature-gate-recognition — Scenario: A 403 from a non-gated read is not a plan limit
    Scenario: A 403 from a non-gated operation is not recognized as a plan limit
      Given the role-read operation GET /roles/role_0123 had been rejected with HTTP status 403
      When the failure is checked for a feature gate
      Then it will not be recognized as a plan-limit rejection
      And it will remain a generic permission denial

    # Source: 060-feature-gate-recognition — Scenario: A non-403 failure from a gated operation is not a plan limit
    Scenario: A non-403 failure from a gated operation is not recognized as a plan limit
      Given the create-proposal operation POST /proposals had been rejected with HTTP status 422 for an invalid change
      When the failure is checked for a feature gate
      Then it will not be recognized as a plan-limit rejection

    # Source: 060-feature-gate-recognition — Scenario: Recognition ignores body content when identifying the gate
    Scenario: Recognition keys on the operation and status, not the response body
      Given the create-proposal operation POST /proposals had been rejected with HTTP status 403
      And the response body described an unrelated cause
      When the failure is checked for a feature gate
      Then it will be recognized as a possible plan-limit rejection
      And the body content will not have affected whether the gate was recognized

  Rule: The downstream receives a possible-plan-limit tag with the suspected gate named
    # In order to word a plan-limit diagnostic that names the right gate and never falsely tells someone to upgrade,
    # as the downstream Plan-Limit Signal capability,
    # I want to receive a recognized rejection tagged as a possible plan limit with the suspected gate named.

    # Source: 060-feature-gate-recognition — Scenario: A genuine permission denial on a gated operation is still flagged as possible
    Scenario: A permission denial on a gated operation is flagged as possible, not confirmed
      Given the advance-to-circulation operation POST /proposals/prp_0123/propose had been rejected with HTTP status 403
      And the rejection was a genuine permission denial unrelated to the plan
      When the failure is checked for a feature gate
      Then it will be recognized as a possible plan-limit rejection, not a confirmed one
      And recognition will make no claim of certainty about the cause

    # Source: 060-feature-gate-recognition — Scenario: A modeled ai_integration gate has no reachable command today
    Scenario: The ai_integration gate is modeled but no operation triggers it today
      Given the ai_integration gate kind is modeled
      And no operation in the recognized set carries the ai_integration gate
      When operations are checked for a feature gate
      Then no failure will be recognized as an ai_integration plan limit today
      And recognition will be ready to name that gate if such an operation is later added

    # Source: 060-feature-gate-recognition — Scenario: Possibility is expressed everywhere recognition is described
    @validation @wip
    Scenario: A recognized rejection is always expressed as a possibility
      Given a 403 recognized from a gated operation
      When the recognition result is inspected
      Then it will be framed as a possible or suspected plan limit
      And it will never assert that the 403 is certainly a plan limit

    # Source: 060-feature-gate-recognition — Scenario: The capability names no user-facing diagnostic wording
    @validation @wip
    Scenario: Recognition produces only a classification, not a rendered message
      Given a 403 recognized from a gated operation
      When the recognition output is inspected
      Then it will carry only a classification naming the suspected gate
      And it will contain no rendered "not available on your plan" message
