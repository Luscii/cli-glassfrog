# Source: 032-output-aware-failure-rendering — Scenario: a permission failure renders as JSON on stdout

Feature: Opaque Failures — Output-Aware Failure Rendering
  When a command fails, the CLI already produces one normalized diagnostic (cause,
  category, next step) and has an effective --output format resolved. Output-Aware
  Failure Rendering wires them together: human formats (full/compact) keep the
  familiar cause-plus-next-step line on stderr, while structured formats (json/yaml)
  emit the unified error envelope on stdout, paired with the unchanged exit code — so
  a failure is as legible and parseable as a success under the same --output.
  (affects: AI agent, Practitioner)

  Rule: A command-execution failure is emitted as one structured envelope on the channel the agent already parses
    # In order to handle a failed call with the same parser I use for a successful one,
    # as an AI agent operating the CLI with --output json,
    # I want failures emitted as one structured error envelope on the channel I already parse.

    # Source: 032-output-aware-failure-rendering — Scenario: a permission failure renders as JSON on stdout
    Scenario: A permission failure renders as a JSON envelope on stdout
      Given a command run with --output json had failed with a 403 carrying a cause and a next step
      When the failure is rendered
      Then stdout will carry one unified error envelope as valid JSON
      And the envelope will carry the failure's message, kind, and originating status
      And stderr will not also carry the human cause-plus-next-step line

    # Source: 032-output-aware-failure-rendering — Scenario: a transport failure under json still emits the envelope
    Scenario: A transport failure under json emits the envelope with no status or body
      Given a command run with --output json had failed with a transport error carrying no API body
      When the failure is rendered
      Then stdout will carry the unified error envelope as valid JSON
      And the envelope will carry the message and kind "network"
      And the envelope will omit the status and body fields

    # Source: 032-output-aware-failure-rendering — Scenario: an API error body is carried verbatim into the structured render
    Scenario: An API error body is carried verbatim into the envelope
      Given a command run with --output json had failed with a non-2xx response carrying a JSON error body
      When the failure is rendered
      Then the raw error body will be nested verbatim within the envelope as structured data
      And the system will not re-classify or re-parse it

    # Source: 032-output-aware-failure-rendering — Scenario: the exit code is unchanged by the format
    Scenario: The exit code is identical across formats
      Given the same 403 permission failure had been rendered once under full and once under json
      When each invocation terminates
      Then both will terminate with the same exit code 4
      And only the rendered presentation will differ between the two

    # Source: 032-output-aware-failure-rendering — Scenario: a usage error keeps its plain-text form even under json
    Scenario: A usage error keeps its plain-text form under json
      Given an unknown command had been invoked with --output json
      When the usage error is reported
      Then it will keep its plain-text dispatch form rather than a structured envelope
      And the failure-render path will not wrap it, because it arose before a command executed in the resolved format

    # Source: 032-output-aware-failure-rendering — Scenario: a structured render that cannot complete writes nothing partial
    Scenario: A structured render that cannot complete writes nothing partial
      Given a structured failure render had been unable to produce a complete document
      When the failure is rendered
      Then nothing partial will be written to stdout
      And the invocation will map to the internal-error exit code 1

    # Source: 032-output-aware-failure-rendering — Scenario: failure parity across the four formats
    @validation @wip
    Scenario: A failure conveys the same facts across the four formats
      Given one failure carrying a cause and a next step
      When it is rendered once under each of full, compact, json, and yaml
      Then every render will convey the same cause and next step
      And the human formats will land on stderr while the structured formats land on stdout as one complete document each

    # Source: 032-output-aware-failure-rendering — Scenario: no secret leaks into any rendered failure
    @validation @wip
    Scenario: No secret appears in any rendered failure
      Given a failure had been rendered under each of the four formats
      When each rendered output is inspected
      Then the API token and the authentication header will appear in none of them

    # Source: 032-output-aware-failure-rendering — Proposed: a mid-walk failure with partial data keeps the incompleteness note on stderr (plan ADR-3)
    Scenario: A mid-walk failure with partial data keeps its incompleteness note on stderr under json
      Given a paginated read with --output json had rendered some pages then failed mid-walk
      When the failure is reported
      Then stdout will carry the partial data document
      And the incompleteness note will be written to stderr rather than a second document on stdout
      And the invocation will exit with the mid-walk failure's non-zero code

    # Source: 032-output-aware-failure-rendering — Proposed: an API body that is not valid JSON is omitted from the envelope (plan ADR-4)
    Scenario: An API error body that is not valid JSON is omitted from the envelope
      Given a command run with --output json had failed with a response whose error body is not valid JSON
      When the failure is rendered
      Then the envelope will still carry the message, kind, and status
      And the envelope will omit the body field rather than failing the render

  Rule: The next step is preserved in the structured failure render
    # In order to know what to do after a failure even when I asked for machine output,
    # as an AI agent driving a pipeline,
    # I want the next step preserved in the JSON/YAML failure render, not dropped when I switch away from the human format.

    # Source: 032-output-aware-failure-rendering — Scenario: the next step survives the structured render as a distinct field
    Scenario: The next step survives the structured render as a distinct field
      Given a command run with --output yaml had failed with a 429 whose next step is to wait for the reset window and retry
      When the failure is rendered
      Then the YAML document will convey that next step as its own distinct, parseable element
      And the cause will remain in its own element

    # Source: 032-output-aware-failure-rendering — Scenario: an internal-error fallback omits the next step in every format
    Scenario: An internal-error fallback omits the next step in every format
      Given a failure whose diagnostic is the internal-error fallback with no next step
      When it is rendered under json and under full
      Then neither render will fabricate a next step
      And the structured render will omit the distinct next-step field rather than null-keying it

  Rule: Human-format failures stay the familiar cause-plus-next-step line on stderr
    # In order to keep reading failures the way I do today when I have not opted into a machine format,
    # as a practitioner running the CLI by hand,
    # I want full/compact failures to stay the familiar cause-plus-next-step line on stderr.

    # Source: 032-output-aware-failure-rendering — Scenario: a human format keeps today's stderr line
    Scenario: A human format keeps today's stderr failure line
      Given a command run under the default full format had failed with a transport error
      When the failure is rendered
      Then the cause-plus-next-step line will be written to stderr exactly as the CLI does today
      And stdout will stay empty

    # Source: 032-output-aware-failure-rendering — Scenario: no implementation leakage in the artifact
    @validation @wip
    Scenario: The failure rendering exposes only observable behavior
      Given the produced specification
      When it is reviewed
      Then it will name only which channel, which shape, and which facts each format renders
      And it will prescribe no language, package layout, or function signature
