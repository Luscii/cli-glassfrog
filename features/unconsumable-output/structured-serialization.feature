# Source: 018-structured-serialization — Scenario: serialize a successful read as JSON

Feature: Unconsumable Output — Structured Serialization
  Results aren't shaped for an AI agent to parse reliably. Structured
  Serialization renders a command's result as machine-readable JSON or YAML —
  the raw API payload verbatim on success, and one unified error envelope in the
  same format on failure — so the agent operator can act on output without
  scraping human text or breaking on a bare-text error. The renderer owns no
  flag and no command; selection arrives with Output Format Selection (020).
  (affects: AI agent, Practitioner)

  Rule: Act on a command's full result as machine-readable JSON
    # In order to act on a command's full result without scraping human-formatted text,
    # as an AI agent operating the CLI on a practitioner's behalf,
    # I want command output as machine-readable JSON.

    # Source: 018-structured-serialization — Scenario: serialize a successful read as JSON
    @wip
    Scenario: A successful payload renders as a JSON document
      Given a command had produced a successful API payload
      And the JSON format was active
      When the result is rendered
      Then the renderer will emit a single valid JSON document carrying the raw payload
      And the document will be the only content on the output channel

    # Source: 018-structured-serialization — Scenario: full fidelity, not the human projection
    @wip
    Scenario: The JSON document preserves fields the projection drops
      Given a successful payload contained fields the human projection omits, such as hypermedia links
      And the JSON format was active
      When the result is rendered
      Then the rendered document will contain those fields verbatim

    # Source: 018-structured-serialization — Scenario: empty result is a valid document, not an empty channel
    @wip
    Scenario: An empty result renders as a valid document
      Given a successful response carried an empty collection
      And a structured format was active
      When the result is rendered
      Then the renderer will emit a valid document representing the empty result
      And the output channel will not be left empty

    # Source: 018-structured-serialization — Scenario: the secret never appears under structured output
    @wip
    Scenario: The token never appears in structured output
      Given a structured format was active
      When the renderer produces any document, on success or failure
      Then the token value will never appear in the document

    # Proposed (architecture-informed): plan ADR-2/ADR-3 — the bytes path preserves number precision (no float64 coercion)
    @wip
    Scenario: A large integer value keeps its exact representation
      Given a successful payload contained a large integer identifier
      And a structured format was active
      When the result is rendered
      Then the rendered value will keep its exact representation
      And it will not be rounded to a floating-point approximation

    # Source: 018-structured-serialization — Scenario: reshaping is absent
    @validation @wip
    Scenario: Structured success output is the raw payload, not the projection
      Given a payload whose human projection would summarize or drop fields
      When it is rendered in a structured format
      Then the document will carry the raw payload
      And it will not carry the reshaped projection

  Rule: Consume the same result as YAML
    # In order to consume output in whichever structured form my tooling prefers,
    # as an AI agent,
    # I want the same result available as YAML, identical in content to the JSON form.

    # Source: 018-structured-serialization — Scenario: serialize the same result as YAML
    @wip
    Scenario: A successful payload renders as a YAML document
      Given a command had produced a successful API payload
      And the YAML format was active
      When the result is rendered
      Then the renderer will emit a single valid YAML document carrying the same data
      And no field will be added or dropped relative to the JSON form

    # Source: 018-structured-serialization — Scenario: JSON and YAML are the same data in two encodings
    @validation @wip
    Scenario: JSON and YAML encode identical data
      Given any successful payload
      When it is rendered once as JSON and once as YAML
      Then parsing both will yield structurally equivalent data
      And neither encoding will carry a field the other lacks

  Rule: Handle failures in the same structured format
    # In order to handle failures without my parser breaking on a bare-text error,
    # as an AI agent that requested a structured format,
    # I want errors emitted in that same format — never as plain text.

    # Source: 018-structured-serialization — Scenario: non-2xx with an API error body, in the active format
    @wip
    Scenario: A non-2xx error renders as a structured envelope
      Given the API had returned a non-2xx response carrying an error body
      And the JSON format was active
      When the failure is rendered
      Then the renderer will emit a unified error envelope as valid JSON
      And the envelope will carry the raw error body verbatim
      And it will not classify or interpret the error

    # Source: 018-structured-serialization — Scenario: transport failure with no API payload
    @wip
    Scenario: A bodiless failure still renders a structured envelope
      Given a transport failure had occurred with no API body
      And the JSON format was active
      When the failure is rendered
      Then the renderer will emit the same unified error envelope as valid JSON
      And the envelope will carry the available failure facts without a raw body

    # Proposed (architecture-informed): plan Cross-cutting — a render failure never yields a partial document
    @wip
    Scenario: An invalid success body surfaces a render error, not a partial document
      Given a 2xx body that was not valid JSON
      And a structured format was active
      When the result is rendered
      Then the renderer will surface a render error
      And it will emit no partial or invalid document

    # Source: 018-structured-serialization — Scenario: the channel is always a complete, valid document
    @validation @wip
    Scenario: Every outcome yields one complete document
      Given a structured format was active
      When any outcome is rendered, whether success, API error, or transport failure
      Then the channel will contain exactly one complete, parseable document
      And it will never contain a fragment or bare text

    # Source: 018-structured-serialization — Scenario: this capability classifies nothing
    @validation @wip
    Scenario: Error rendering performs no classification
      Given a non-2xx API response had been classified upstream
      When the error envelope is rendered
      Then the document will reflect the classified facts it was handed
      And the renderer will apply no status-specific interpretation of its own

    # Source: 018-structured-serialization — Scenario: every failure shares one error shape
    @validation @wip
    Scenario: All failures share one envelope shape
      Given an API error, a transport failure, and a fail-safe refusal were each rendered under a structured format
      When their documents are inspected
      Then all three will share the same top-level error-envelope shape
      And fields that do not apply to a failure will be absent rather than renamed
