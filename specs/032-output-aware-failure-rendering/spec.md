# Specification: Output-Aware Failure Rendering

**Feature**: 032-output-aware-failure-rendering
**Role**: Definer
**Tier**: 1 (zero setup)

---

## System Overview

Output-Aware Failure Rendering is the **failure-side renderer** of the Output Formatting cluster and the presentation half of **Diagnostic Reporting** (problem: *Opaque Failures*). When a command fails, the CLI already produces two things its siblings own: a **normalized diagnostic** — a cause, a category, and (where one exists) a next step — from Diagnostic Normalization (031), and an **effective output format** — `full`, `compact`, `json`, or `yaml` — from Output Format Selection (020). This capability is the missing wire between them: it takes that one diagnostic and renders it *in the selected format*, so a failure is as legible and as parseable as a success under the very same `--output` the caller chose.

Today the gap is documented and deliberate: under `json`/`yaml` a command failure still prints today's human cause-plus-next-step text on stderr (020 Assumption; 019 kept it; 018 built the unified error envelope but nothing routes failures through it yet). This capability closes that gap. It renders the **human** formats (`full`/`compact`) exactly as the CLI does today — a cause-plus-next-step line on stderr — and renders the **structured** formats (`json`/`yaml`) as 018's unified error envelope on stdout, so an agent that requested JSON always parses one channel as JSON whether the call succeeded or failed. It owns *only* the choice of how to render a failure given a format; it does not classify the failure (031), select the format (020), define the envelope shape or the encoders (018), or decide the process exit code (004) — it pairs its rendering with the code those siblings already produce.

---

## Behavioral Accord

### Rendering a failure in the resolved format

- When a command execution fails and a normalized diagnostic is available, the system renders that one diagnostic in the output format already resolved for the invocation — the same `full`/`compact`/`json`/`yaml` a successful result would have used.
- When the resolved format is a human format (`full` or `compact`), the system writes the diagnostic as a cause-plus-next-step line to stderr, identical to today's failure output — the cause alone when there is no next step, otherwise the cause and the next step together. stdout stays empty.
- When the resolved format is a structured format (`json` or `yaml`), the system writes 018's unified error envelope to stdout as the sole content of that channel, so a consumer parsing stdout as JSON/YAML reads the failure with the same parser it uses for success, never a bare-text line. The human stderr line is not also emitted on this path — the envelope is the whole failure rendering.
- The rendered failure carries the same facts in every format: the cause and (where one exists) the next step that the diagnostic supplied. Selection changes only how the failure is presented, never which facts it contains — a structured render is never made less informative than the human one.

### Preserving the next step under every format

- When the diagnostic carries a next step, the system surfaces it under the structured formats too, not only under the human formats — a failure rendered as JSON still tells the caller what to do next, honoring the cause-plus-next-step contract regardless of format.
- When the structured formats render a next step, the system surfaces it as its own distinct, independently parseable element of the failure document — not folded into the human-readable cause text — so an agent can read the recovery action programmatically rather than parsing it back out of a prose message.
- When the diagnostic carries no next step (the internal-error fallback, or a general API error with no reliable recovery), the system renders the cause without inventing one, in whichever format is active — and under the structured formats the distinct next-step element is simply absent, never null-keyed or fabricated.

### Carrying through the structured failure facts

- When the failure originated from a typed API error, the structured render includes the failure facts that error carries — its kind, its originating HTTP status, and the raw API error body verbatim when one is present and is valid structured data — placed into 018's envelope; fields that do not apply to a given failure are absent. A body that is not valid structured data (not parseable JSON) is omitted from the envelope rather than included, so a malformed upstream body never suppresses the rest of the diagnostic.
- When the failure carries no API payload (a transport failure, a decode failure, or a local fail-safe refusal), the structured render still emits the same envelope shape carrying the facts available — its message and kind — rather than degrading to unstructured text or an empty channel.
- When any failure is rendered, the secret token and the authentication header never appear in the rendered output, under any format.

### Pairing with the exit code, never deciding it

- When the system renders a failure, the process still terminates with the exit code the category maps to under Exit-Code Convention (004); the rendering and the code are complementary outputs of the one failure and never disagree about which failure occurred.
- When a structured render itself cannot produce a complete document, the system writes nothing partial to stdout and the invocation maps to the internal-error code — a half-document is never emitted, on the failure path no less than the success path. (A raw API error body that is not valid JSON does not trigger this: it is omitted from the envelope, so the rest of the document still renders — see "Carrying through the structured failure facts".)

---

## User Scenarios

**In order to** handle a failed call with the same parser I use for a successful one, instead of special-casing a bare-text error,
**as an** AI agent operating the CLI with `--output json`,
**I want** failures emitted as one structured error envelope on the channel I already parse.

**In order to** know what to do after a failure even when I asked for machine output,
**as an** AI agent driving a pipeline,
**I want** the next step preserved in the JSON/YAML failure render, not dropped when I switch away from the human format.

**In order to** keep reading failures the way I do today when I have not opted into a machine format,
**as a** practitioner running the CLI by hand,
**I want** `full`/`compact` failures to stay the familiar cause-plus-next-step line on stderr.

---

## Non-Behaviors

- The system must not classify a failure or compose its cause, category, or next step. **Why**: Diagnostic Normalization (031) is the single normalizer; re-deriving any of those here would split one diagnostic contract across two capabilities and let the rendered failure disagree with the category that drives the exit code.
- The system must not select or resolve the output format. **Why**: Output Format Selection (020) owns the precedence chain and the effective format; this capability consumes the resolved format and would reopen 020's resolution contract if it re-derived it.
- The system must not define the unified error envelope's shape or implement the JSON/YAML encoders. **Why**: Structured Serialization (018) owns the envelope shape and the encoders; this capability maps the diagnostic's facts into that envelope and routes them through 018's renderer, so 018 can evolve the encoding without a second renderer drifting from it.
- The system must not emit or decide the process exit code. **Why**: Exit-Code Convention (004) is the single category→code mapper; this capability renders the failure and leaves the code to 004, so the two never disagree.
- The system must not render the invalid-selector usage error. **Why**: Output Format Selection (020) owns that case and renders it itself — it arises while resolving the format, before any valid format exists to render into, so 032 cannot render it in "the selected format." Rendering it here would require guessing a format the caller never validly selected.
- The system must not render usage errors (an unknown command, or an unknown/missing/invalid flag or positional argument) in the selected structured format; those keep today's plain-text dispatch form. **Why**: a usage error fails before a command executes — at or around the point the format is resolved — so the resolved format is not reliably available to render into, and the invalid-selector case shows usage failures and format selection are entangled. 032's surface is the *command-execution* failure path (transport, decode, API); folding usage errors in would blur that boundary and overlap 020's plain-text usage handling.
- The system must not render successful results. **Why**: the success dispatch already routes a result through the matching renderer (018/019); this capability is the failure-path counterpart and would fork the success contract if it touched it.
- The system must not retry, wait, re-parse a raw response body, or interpret a `403` as a plan-availability signal. **Why**: those belong to Rate-Limit Handling (017), API Error Extraction (015), and the *Unsignalled Plan Limits* problem respectively; a renderer that reached into them would duplicate work the diagnostic already carries.
- The system must not fabricate a next step (or any fact) the diagnostic did not supply, in any format. **Why**: CONSTITUTION VIII (No Fabricated Data) and II — a confidently wrong next step rendered as clean JSON is more dangerous than an honest omission.

---

## Integration Boundaries

- **Diagnostic Normalization (031)** *(upstream)*: supplies the one normalized diagnostic — cause, category, next step. This capability reads those facts and renders them; it never re-classifies.
- **Output Format Selection (020)** *(upstream)*: supplies the resolved effective format for the invocation. This capability renders the failure in that format. 020 establishes the selection; 032 renders failures into it.
- **Structured Serialization (018)** *(downstream renderer)*: provides the unified error envelope shape and the JSON/YAML encoders. This capability maps the diagnostic's facts (and the typed API error's status/body, when present) into the envelope and routes them through 018's encoder for the `json`/`yaml` formats.
- **Templated Human Rendering (019)** *(parallel surface)*: 019 owns success templates and explicitly does not render errors; this capability owns the human failure line (cause-plus-next-step on stderr) it preserves unchanged, so the human success/failure split stays clean.
- **API Error Extraction (015)** *(upstream of the facts)*: source of the typed API error whose kind, status, and raw body feed the structured envelope. This capability reads the already-typed error; it never re-parses the body.
- **Exit-Code Convention (004)** *(complementary output)*: maps the diagnostic's category to the process exit code. This capability renders the failure; 004 emits the code. The two are paired, neither subsumes the other.

---

## Driving Scenarios

### Happy path

**Scenario: a permission failure renders as JSON on stdout**
Given a command run with `--output json` fails with a `403` whose diagnostic carries a cause and a next step
When the failure is rendered
Then the system writes 018's unified error envelope to stdout as valid JSON
And the envelope carries the failure's message, kind, and originating status
And stderr is not also given the human cause-plus-next-step line.

**Scenario: a human format keeps today's stderr line**
Given a command run under the default `full` format fails with a transport error
When the failure is rendered
Then the system writes the cause-plus-next-step line to stderr exactly as the CLI does today
And stdout stays empty.

**Scenario: the next step survives the structured render as a distinct field**
Given a command run with `--output yaml` fails with a `429` whose diagnostic's next step is to wait for the rate-limit window and retry
When the failure is rendered
Then the YAML failure document conveys that next step as its own distinct, parseable element
And the cause remains in its own element, so an agent reads the recovery action without parsing it out of the cause text.

### Error scenarios

**Scenario: a transport failure under json still emits the envelope**
Given a command run with `--output json` fails with a transport error carrying no API body
When the failure is rendered
Then the system emits the same unified error envelope as valid JSON on stdout
And the envelope carries the message and kind, with the raw-body field absent
And the channel is never left empty or filled with bare text.

**Scenario: an API error body is carried verbatim into the structured render**
Given a command run with `--output json` fails with a non-2xx response carrying a JSON error body
When the failure is rendered
Then the raw error body is included verbatim within the envelope
And the system does not re-classify or re-parse it.

### Edge cases

**Scenario: the exit code is unchanged by the format**
Given the same `403` permission failure rendered once under `full` and once under `json`
When each invocation terminates
Then both terminate with the same exit code under Exit-Code Convention (004)
And only the rendered presentation differs between the two.

**Scenario: an internal-error fallback omits the next step in every format**
Given a failure whose diagnostic is the internal-error fallback with no next step
When it is rendered under `json` and under `full`
Then neither render fabricates a next step
And the structured render omits the distinct next-step field rather than null-keying or inventing one.

**Scenario: a usage error keeps its plain-text form even under json**
Given an unknown command is invoked with `--output json`
When the usage error is reported
Then it keeps today's plain-text dispatch form rather than a structured envelope
And 032 does not render it, because the failure arises before a command executes in the resolved format.

**Scenario: a structured render that cannot complete writes nothing partial**
Given a structured failure render cannot produce a complete document
When the failure is rendered
Then nothing partial is written to stdout
And the invocation maps to the internal-error exit code (004).

---

## Validation Scenarios

> These are held out from the implementing agent for independent verification.

**Scenario: failure parity across the four formats**
Given one failure carrying a cause and a next step
When it is rendered once under each of `full`, `compact`, `json`, and `yaml`
Then every render conveys the same cause and next step
And the human formats land on stderr while the structured formats land on stdout as one complete document each.

**Scenario: no secret leaks into any rendered failure**
Given a failure rendered under each of the four formats
When each rendered output is inspected
Then the API token and the authentication header appear in none of them.

**Scenario: no implementation leakage in the artifact**
Given the produced specification
When it is reviewed
Then it names only the observable rendering behavior (which channel, which shape, which facts) and prescribes no language, package layout, or function signature.

---

## Assumptions

- **Structured failures go to stdout; human failures stay on stderr** (decision, recorded for traceability): the structured error envelope is written to stdout — the channel an agent pipes into a parser — so success and failure are read from one channel, while human cause-plus-next-step diagnostics stay on stderr as today. (Informed by 018's "every output is structured under a structured format" contract and its plan routing `RenderError(...) → stdout`, and by 020's interim-gap Assumption naming exactly this as 032's job.)
- **Category-to-envelope-kind mapping** `[ASSUMED]`: the *behavior* is fixed — the structured render carries the failure's kind drawn from the same taxonomy 031 assigns and 018's envelope expects. The exact spelling of each kind token in the envelope is an interface-level detail pinned in plan/interface alongside 018's `ErrorDetail.Kind` and 031's category vocabulary, not a behavioral gap.
- **Distinct next-step element in the envelope** (decision, recorded for traceability): the structured render surfaces the next step as its own parseable element of 018's error envelope, distinct from the cause/`message`. This extends 018's current envelope (which carries only `message`) with a next-step element; the exact field name is an interface-level detail pinned alongside 018's envelope shape. (Decided in the 2026-06-10 clarification — the agent operator reads the recovery action programmatically.)
- **One failure in, one rendering out**: assumed the renderer handles a single terminal failure per invocation, mirroring 031's one-diagnostic-per-invocation shape and the CLI's one-invocation-one-outcome convention.

---

## Ambiguity Warnings

_None remaining — both behavioral forks from the initial draft were resolved in the 2026-06-10 clarification session: (1) the structured render surfaces the next step as its own distinct, parseable element of 018's envelope (not folded into `message`), so an agent reads the recovery action programmatically; and (2) 032's surface is the command-execution failure path (transport, decode, API) — usage errors and the invalid-selector error keep their plain-text form and are out of scope. See **Clarifications**. The remaining `[ASSUMED]` items (envelope kind-token spelling, next-step field name) are interface-level naming details, not behavioral gaps._

---

## Clarifications

### Session 2026-06-10

- **Next-step parseability under structured formats**: under `json`/`yaml` the next step is surfaced as its own distinct, independently parseable element of 018's unified error envelope, distinct from the cause/`message` — not folded into a prose string. This extends 018's current `message`-only envelope with a next-step element (exact field name is an interface detail). Chosen because the CLI's operator is usually an AI agent, so the recovery action must be machine-readable, fully realizing CONSTITUTION II (cause + next step) in structured form.
- **Scope of format-aware rendering**: 032 renders only *command-execution* failures — the transport, decode, and API errors that arise after a command runs and would otherwise print cause-plus-next-step on stderr. Usage errors (unknown command, or unknown/missing/invalid flag or positional) keep today's plain-text dispatch form and are out of scope, as does the invalid-selector usage error that Output Format Selection (020) already owns. Reasoning: a usage error fails before a command executes in the resolved format, so the format is not reliably available to render into; the invalid-selector case shows usage failures and format selection are entangled.
