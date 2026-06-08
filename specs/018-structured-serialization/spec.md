# Specification: Structured Serialization

**Feature**: 018-structured-serialization
**Role**: Definer
**Tier**: 1 (zero setup)

---

## System Overview

Structured Serialization renders a command's result data as machine-readable **JSON** and **YAML** so the CLI's primary operator — an AI agent — can parse output reliably and decide what to do next (VISION principle 3). Every self-service read so far (Identity Read 011, My Roles 012, My Actions 013, My Projects 014) prints a *reshaped, summarized projection* for human eyes and explicitly defers a "raw or structured `--output json`" mode to a future capability. This is that capability.

It is the **serializer seam**: it owns the JSON and YAML encoders, the guarantee that — when a structured format is active — the *entire* output channel is that format, and the shape of the **unified error envelope** that failures are rendered into. It does **not** parse the flag that selects a format (Output Format Selection, 020, depends on this and routes output through it), does **not** produce human-readable `full`/`compact` output (Templated Human Rendering, 019, owns that parallel branch), and does **not** classify API errors (API Error Extraction, 015, turns a status and body into a typed error — 018 only renders the classified facts into its envelope). It is a no-dependency root of the Output Formatting solution: a transformation over data the commands already produce.

The data it serializes is the **raw API payload, verbatim** — all fields the API returned, not the human projection's summarized subset. The agent operator wants full fidelity; reshaping is the human renderer's job.

---

## Behavioral Accord

### Serialization

- When a command's result is rendered in JSON format, the system emits a single valid JSON document carrying the command's result data — the raw API payload as the API returned it, with all fields intact — not the reshaped, summarized projection used for human output.
- When the same result is rendered in YAML format, the system emits a single valid YAML document carrying the identical data, differing from the JSON form only in encoding: no field is added, dropped, or reshaped by the choice of format.
- When the result data is serialized, the document is equivalent for the same input each time, so a consumer can rely on a stable structure to parse against.

### Uniform format across outcomes

- When a structured format is active, the system renders *every* output the invocation produces in that format — successful payloads and errors alike — so a consumer that requested JSON always receives JSON and never a bare-text message interleaved into a structured stream.
- When any failure occurs under a structured format, the system emits the error as a single **unified error envelope** with a consistent top-level shape, regardless of the failure's origin: it carries a human-readable message, the failure's kind and originating status where those apply, and the raw API error body when one is present. Fields that do not apply to a given failure are absent. This one shape is what a consumer always parses on the error path.
- When the API returns a non-2xx response carrying an error body, the raw body is included within the envelope verbatim; the system does not classify or interpret the error — API Error Extraction (015) supplies the classified facts (kind, status) that 018 places into the envelope, while 018 owns the envelope's shape, not the classification.
- When a failure carries no API payload at all — a transport/wire failure, or the authenticated transport's fail-safe refusal — the system still emits the same unified error envelope, carrying the facts available (its message; no raw body), rather than degrading to unstructured text.

### Output discipline

- When structured output is produced, the serialized document is the only content written to the channel a consumer parses, so the consumer can read that channel directly without stripping decorative text.
- When any value is serialized, the secret token and authentication header never appear in the document, under success or failure.
- When output is produced, the system never writes a truncated or non-parseable fragment; the channel carries a complete, valid document or the structured-output contract is not met.

---

## User Scenarios

**In order to** act on a command's full result without scraping human-formatted text,
**as an** AI agent operating the CLI on a practitioner's behalf,
**I want** command output as machine-readable JSON.

**In order to** consume output in whichever structured form my tooling prefers,
**as an** AI agent,
**I want** the same result available as YAML, identical in content to the JSON form.

**In order to** handle failures without my parser breaking on a bare-text error,
**as an** AI agent that requested a structured format,
**I want** errors emitted in that same format — never as plain text.

---

## Non-Behaviors

- The system must not parse or own the `--output` flag, nor decide which format (if any) is active. **Why**: Output Format Selection (020) owns selecting the format per invocation and routing output through these encoders; defining the flag here would pre-empt that decision and fork the surface.
- The system must not produce the human-readable `full`/`compact` rendering or the default reshaped projection. **Why**: Templated Human Rendering (019) owns the human branch; 018 is the machine branch only, and the two consume the same result data independently.
- The system must not classify, interpret, or enrich an API error into a typed or meaningful error. **Why**: API Error Extraction (015) owns turning a raw status and body into a typed error; 018 only places the classified facts into its envelope and encodes them, so 015 can evolve its classification without changing 018's envelope contract.
- The system must not decide the process exit code. **Why**: Exit-Code Convention (004) owns the exit code; output format is orthogonal to the success/failure signal — an agent reads the structured document *and* the exit code as complementary outputs.
- The system must not reshape, summarize, re-key, or drop fields from a *successful* result's payload. **Why**: the structured mode's value is full fidelity (raw payload verbatim); reshaping is the human projection's job, and a lossy machine view would force agents to guess what was omitted. (The error path is the deliberate exception — failures are wrapped in the unified envelope, with any raw API error body preserved verbatim inside it.)
- The system must not emit the secret token or authentication header in any serialized output. **Why**: the secret-never-emitted rule that governs Credential Discovery (005), Storage (006), Authentication (007), and Request Execution (010) applies to every emission surface, and structured output is a new one.
- The system must not emit a partial, truncated, or otherwise non-parseable document, even on the error path. **Why**: reliable machine parsing is the entire point; a half-document defeats it as surely as bare text would.

---

## Integration Boundaries

- **Command result data (upstream)**: the data each command produces — for the reads, the raw API response body surfaced by Request Execution (010). 018 encodes this data; it does not fetch, reshape, or summarize it.
- **Output Format Selection (020 — downstream dependent)**: selects which structured format (if any) is active for an invocation and routes the command's output — success and error — through these encoders. 018 provides the encoders and the uniform-format contract; 020 chooses and dispatches. 020 also owns the case where the selector itself is invalid (e.g. `--output=xml`): 018's uniform-format guarantee applies only once a structured format is validly active.
- **Templated Human Rendering (019 — parallel sibling)**: owns human `full`/`compact` output over the same command result; the human branch to 018's machine branch. Both build on the result data; neither depends on the other.
- **API Error Extraction (015 — composition boundary)**: classifies a non-2xx response into a typed error (its kind, status, extracted detail). When a structured format is active, 018 places those classified facts and the raw error body into its unified error envelope and serializes it. 018 owns the envelope's shape and the encoding; it performs no classification. 015 does no serialization.
- **Exit-Code Convention (004)**: the process exit code is decided independently of output format; the structured document and the exit code are complementary signals an agent reads together.
- **Glassfrog API**: the origin of the raw payloads being serialized, reached only indirectly through Request Execution (010). 018 has no direct contact with the API.

---

## Driving Scenarios

### Happy path

**Scenario: serialize a successful read as JSON**
Given a command produced a successful API payload
And JSON format is active
When the result is rendered
Then the system emits a single valid JSON document containing the raw payload with all fields intact
And that document is the only content on the channel the consumer parses.

**Scenario: serialize the same result as YAML**
Given the same successful API payload
And YAML format is active
When the result is rendered
Then the system emits a single valid YAML document carrying the identical data
And no field is added, dropped, or reshaped relative to the JSON form.

**Scenario: full fidelity, not the human projection**
Given a payload containing fields the human projection summarizes away (e.g. hypermedia links, embedded sub-resources)
And JSON format is active
When the result is rendered
Then those fields are present verbatim in the document.

### Error scenarios

**Scenario: non-2xx with an API error body, in the active format**
Given the API returned a non-2xx response carrying an error body
And JSON format is active
When output is rendered
Then the system emits a unified error envelope as valid JSON, with the raw error body included verbatim within it
And does not classify or interpret the error itself
And the consumer can parse the channel as JSON without special-casing the failure.

**Scenario: transport failure with no API payload**
Given a wire/transport failure occurred and there is no API body to serialize
And JSON format is active
When output is rendered
Then the system emits the same unified error envelope as valid JSON, carrying the failure's available facts and no raw body
And does not fall back to bare, unstructured text.

### Edge cases

**Scenario: empty result is a valid document, not an empty channel**
Given a successful response whose result data is an empty collection
And a structured format is active
When the result is rendered
Then the system emits a valid document representing the empty result (e.g. an empty array)
And the channel is not left empty.

**Scenario: the secret never appears under structured output**
Given any outcome — success payload, serialized API error, or bodiless failure
And a structured format is active
When the produced document is inspected
Then the token value and the authentication header are absent from it.

---

## Validation Scenarios

> These are held out from the implementing agent for independent verification.

**Scenario: JSON and YAML are the same data in two encodings**
Given any successful payload
When it is rendered once as JSON and once as YAML
Then parsing both yields structurally equivalent data
And neither encoding carries a field the other lacks.

**Scenario: the channel is always a complete, valid document**
Given any outcome under a structured format (success, API error, transport failure, fail-safe refusal)
When the output channel is inspected
Then it contains exactly one complete, parseable document of the active format
And never a fragment, a mixed stream, or bare text.

**Scenario: this capability classifies nothing**
Given a non-2xx API response
When the error is serialized
Then the document reflects the classified facts it was handed, wrapped in 018's envelope
And 018 applies no status-specific interpretation of its own (that remains 015's).

**Scenario: every failure shares one error shape**
Given an API error, a transport failure, and a fail-safe refusal, each under a structured format
When their documents are inspected
Then all three share the same top-level error-envelope shape
And fields that do not apply to a given failure are absent rather than differently-named.

**Scenario: reshaping is absent**
Given a payload whose human projection would summarize or drop fields
When it is rendered in a structured format
Then the document carries the raw payload, not the projection.

---

## Assumptions

- **Output channel** `[ASSUMED]`: structured output is written to standard output (the channel an agent pipes into a parser), with any diagnostics kept off that channel. The behavior — one complete document, nothing decorative alongside it — is fixed; the exact channel mechanics are a planning detail.
- **Encoding normalization**: the *data* serialized is the raw payload verbatim (all fields), while the *encoding* is normalized to a valid JSON/YAML document (consistent indentation/quoting). Re-encoding is unavoidable for YAML and harmless for JSON; field content is preserved either way.
- **Two formats only**: the structured formats are JSON and YAML, matching the FEATURE-MODEL capability. Caller-supplied templates (User-Defined Template Output) and human `full`/`compact` are separate, out-of-scope capabilities.
- **Default remains the projection**: when no structured format is active, output stays the reshaped human projection the `/me*` reads already produce (019); 018 changes nothing about the default path.
- **Error-envelope field names** `[ASSUMED]`: the unified error envelope carries a human-readable message, the failure kind, the originating status (when applicable), and the raw API error body (when present). The *existence and consistency* of the envelope is fixed behavior (018 defines it); the exact field names and nesting are an interface-design detail pinned in plan/interface alongside 015's classified-error shape.

---

## Ambiguity Warnings

_None remaining — both behavioral forks were resolved during the defining conversation: (1) errors under a structured format are rendered as a single **unified error envelope** that 018 defines, with the raw API error body included verbatim when present and the same shape used for bodiless failures (transport, fail-safe refusal); 015 supplies the classified facts, 018 owns the envelope shape and encoding; and (2) 018's "every output is structured" guarantee is scoped to *when a structured format is validly active* — the invalid-selector bootstrap (e.g. `--output=xml`) belongs to Output Format Selection (020). The remaining `[ASSUMED]` items (output channel, error-envelope field names) are interface/planning-time shape details, not behavioral gaps._

---

## Clarifications

### Session 2026-06-07

- **Error-document shape**: errors are rendered as a single unified error envelope (a consistent top-level shape) that 018 defines, rather than echoing the raw API error body verbatim. The raw body is included *within* the envelope when present; bodiless failures use the same envelope with the facts available. 018 owns the envelope shape and encoding; API Error Extraction (015) supplies the classified facts (kind, status). The exact field names remain an interface detail.
- **Format-selection bootstrap**: 018's uniform-format guarantee applies only once a structured format is validly active. When the format selector itself is invalid (e.g. `--output=xml`), handling that usage error belongs to Output Format Selection (020), not 018.
