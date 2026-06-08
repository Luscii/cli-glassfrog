# Tasks: Structured Serialization

**Feature**: 018-structured-serialization
**Concretization**: Full context (plan + spec + interface + scenarios)
**Inputs**: plan.md, spec.md, interface-spec.md, features/unconsumable-output/structured-serialization.feature

---

## Dependency Graph

Phase 1: `internal/output` success serializers — `Format` + `RenderSuccess` (JSON + YAML) + `sigs.k8s.io/yaml` dep (1 task, no dependencies) [Shared]
Phase 2: Unified error envelope — `ErrorEnvelope`/`ErrorDetail` + `RenderError` (1 task, depends on T001) [US3]
Phase 3: Component-level acceptance — godog suite over the `output` package (1 task, depends on T001 + T002) [Shared]

3 tasks total | T001 startable immediately (018 is a no-dependency root) | Builder: pipeline

> **018 is a no-dependency root of Output Formatting.** It introduces a new pure leaf package `internal/output` and the only outside dependency the output surface adds (`sigs.k8s.io/yaml`). It builds on nothing from siblings: it captures the raw 2xx body via a `json.RawMessage` target (a valid `out` for the **existing** `Execute` — no 010 change) and reuses `*ResponseError.Body` and the `classifyClientError` taxonomy *by reference only* (the `kind` vocabulary), not by code change.
>
> **No CLI surface here.** 018 ships the encoders and the envelope shape — there is no `--output` flag and no command. The acceptance suite (T003) therefore runs at the **component level** over the `output` package functions, not end-to-end through a command. The flag, format selection, success/error routing, the typed-error→envelope *mapping*, and the invalid-selector bootstrap all land with **Output Format Selection (020)**; end-to-end `--output json` on a real command is verifiable then.
>
> **Downstream siblings extend this package.** Templated Human Rendering (019) adds template rendering to `internal/output`; 020 adds the selector + router. Their work is out of scope here.

---

## Branching Guidance

**Pipeline mode**: `spec/018-structured-serialization/base` → `spec/018-structured-serialization/task-1`, `…/task-2`, `…/task-3` (one task branch per T-id, merged back into the spec base). The base can be cut from current `main` — 018 depends on no unlanded sibling.

**Parallel-spec awareness**: none required. 018 is a no-dependency root; T001 needs nothing from other specs. Coordinate only with 019/020 on the shared `internal/output` package name once those specs begin (this spec creates the package — first-to-land-creates).

---

## Phase 1: `internal/output` success serializers [Shared]

- [x] **T001** [Shared] Create the `internal/output` leaf package with `Format` and `RenderSuccess` (JSON + YAML over raw bytes); add the `sigs.k8s.io/yaml` dependency — RED-first unit tests — 7 unit tests, `internal/output` clean of internal deps, `sigs.k8s.io/yaml v1.4.0` added
  - **Scope**: Create the new pure leaf package `internal/output` (no cobra, no transport, no domain types — importable by `internal/cli` without a cycle). Define `Format` (`JSON`, `YAML`). Implement `RenderSuccess(f Format, payload json.RawMessage) ([]byte, error)`: for `JSON`, validate and normalize the raw bytes into a single valid, consistently-indented JSON document (e.g. via `encoding/json`); for `YAML`, transform the raw JSON bytes with `sigs.k8s.io/yaml.JSONToYAML`. An empty/whitespace-only `payload` renders as a valid empty document, never an empty channel. Invalid JSON bytes (a 2xx contract violation) return a render error and **no** document — never a partial fragment. Add `sigs.k8s.io/yaml` to `go.mod`/`go.sum`. No exit codes, no flag, no command.
  - **Acceptance criteria**:
    - A raw JSON payload renders as a single valid JSON document carrying every field present in the input (including fields a typed `glassfrog` struct would drop, e.g. hypermedia links)
    - The same payload renders as a single valid YAML document; parsing the JSON and the YAML yields structurally equivalent data with no field in one missing from the other
    - A large integer value keeps its exact representation in both formats (no float64 coercion) — operating on bytes, not a generic decode
    - The same input renders an equivalent document each time (deterministic)
    - An empty/whitespace-only payload renders a valid empty document; the channel is never left empty
    - A non-JSON byte input returns a render error and emits no partial/invalid document
    - `internal/output` adds no import of `internal/cli`/`internal/apiclient`; `go build ./...` and `go vet ./...` are clean with the new dependency
  - **Dependencies**: None (no-dependency root; new leaf package).
  - **Plan reference**: Phase 1; ADR-1 (new `internal/output` leaf), ADR-2 (serialize raw bytes), ADR-3 (`sigs.k8s.io/yaml` `JSONToYAML`); Data Model Design
  - **Interface references**: interface-spec.md — `internal/output` (Surface: `Format`, `RenderSuccess`), the success document shape; Interactions (JSON≡YAML, number/field fidelity)
  - **Scenario references**: structured-serialization.feature: "A successful payload renders as a JSON document", "The JSON document preserves fields the projection drops", "A successful payload renders as a YAML document", "JSON and YAML encode identical data", "A large integer value keeps its exact representation", "An empty result renders as a valid document", "Structured success output is the raw payload, not the projection"
  - **Risk**: ⚠️ Serialize the raw bytes, never re-encode the tolerant typed `glassfrog` struct (it drops unknown fields — ADR-2). ⚠️ Operate on bytes for YAML (`JSONToYAML`) so JSON numbers keep precision — do not round-trip through a generic `any` with float64. ⚠️ Build the whole document in memory before any write so a render failure never leaves a partial document.

## Phase 2: Unified error envelope [US3]

- [x] **T002** [US3] Add `ErrorEnvelope`/`ErrorDetail` and `RenderError` to `internal/output` — RED-first unit tests over hand-built envelopes — 7 unit tests over hand-built envelopes; no classification, no apiclient import
  - **Scope**: In `internal/output`, define the unified `ErrorEnvelope` wrapping one `ErrorDetail` under an `error` key: `Message string` (always), `Kind string` (always — the lowercased taxonomy term), `Status int` (omitempty — HTTP status, non-2xx only), `Body json.RawMessage` (omitempty — the raw API error body verbatim, when present). Implement `RenderError(f Format, env ErrorEnvelope) ([]byte, error)` rendering the envelope in the active format (JSON marshal; YAML equivalent), with deterministic field order and a complete document or a render error — never a fragment. 018 owns the envelope **shape** and encoder only; it performs no classification. The typed-error→envelope *mapping* (kind from `classifyClientError`, status/body from `*ResponseError`) is **not** built here — it lands with 020 (ADR-4; interface-spec). Tests construct `ErrorEnvelope` values directly.
  - **Acceptance criteria**:
    - An envelope with `kind="api"`, a status, and a nested raw `body` renders as one valid JSON document carrying the raw body verbatim
    - A bodiless envelope (`kind="network"`, message only) renders with `status` and `body` absent (not null-keyed) — same top-level shape as the API-error case
    - The same envelope renders equivalently in JSON and YAML (same data, two encodings)
    - Each `kind` value (`api`, `network`, `usage`, `runtime`) renders without error
    - A token-bearing message/body is never produced by the renderer (the renderer adds nothing; a test asserts a supplied secret-free envelope stays secret-free and the renderer introduces no token)
    - A `RenderError` failure returns an error and emits no partial document
    - `internal/output` still imports neither `internal/cli` nor `internal/apiclient`
  - **Dependencies**: T001 (`Format`)
  - **Plan reference**: Phase 2; ADR-4 (unified envelope shape; `kind` from the `classifyClientError` taxonomy; mapping deferred to 020); Data Model Design; Cross-cutting (no partial documents, secret hygiene)
  - **Interface references**: interface-spec.md — `ErrorEnvelope`/`ErrorDetail`/`RenderError` (Surface), the error-envelope document shape, Error Communication (kind table)
  - **Scenario references**: structured-serialization.feature: "A non-2xx error renders as a structured envelope", "A bodiless failure still renders a structured envelope", "All failures share one envelope shape", "Error rendering performs no classification", "An invalid success body surfaces a render error, not a partial document" (render-error arm), "The token never appears in structured output"
  - **Risk**: ⚠️ `Status`/`Body` must be `omitempty` so bodiless failures share the exact top-level shape (not a differently-keyed variant). ⚠️ Do **not** add a `classifyClientError` arm, `Outcome`, or `ExitCode` here, and do **not** import `apiclient` — the mapping is 020's. ⚠️ Build the document fully before writing — no partial envelope on a render failure.

## Phase 3: Component-level acceptance [Shared]

- [ ] **T003** [Shared] Make the driving scenarios pass as executable component-level acceptance via a new godog suite over the `output` package
  - **Scope**: Add godog step definitions for `features/unconsumable-output/structured-serialization.feature` in a **new** godog suite (e.g. `TestStructuredSerializationFeatures`) whose `Paths` names **only** that feature file (LEARNINGS: a suite points at its own file, never the `features/` directory). Drive `RenderSuccess`/`RenderError` directly with raw-byte and hand-built-envelope fixtures — there is no command to invoke (no CLI surface until 020), so steps exercise the package functions. Remove `@wip` from the behavioral scenarios (the spec-derived + architecture-informed); keep the `@validation` scenarios `@wip` (held for validate). Step helpers return errors, never panic; capture any output with a temp file, not `os.Pipe` (PR #10 LEARNINGS). Reuse existing `internal/cli`/package step phrasings where an assertion already exists (grep `sc.Step(` first).
  - **Acceptance criteria**:
    - Every non-`@validation` scenario in structured-serialization.feature has an executable, passing step path driven through `RenderSuccess`/`RenderError`
    - `@wip` removed from those behavioral scenarios; the five `@validation` scenarios keep `@wip`
    - The new suite's `Paths` names only `structured-serialization.feature`; it runs and reports its own independent scenario count
    - No real network and no real home/filesystem are touched; `go build ./...`, `go vet ./...`, and the feature suite run clean
  - **Dependencies**: T001, T002
  - **Plan reference**: Phase 3; Cross-cutting (testing — component-level, hermetic, RED-first)
  - **Interface references**: interface-spec.md — Surface + Error Communication (the contracts the scenarios assert against)
  - **Scenario references**: structured-serialization.feature: all behavioral Rule-block scenarios (the five `@validation` scenarios stay held for validate)
  - **Risk**: ⚠️ Suite scoping — point the suite at its specific feature file, not the directory; verify it reports its own count. ⚠️ Drive the package functions directly — do not invent a CLI command or `--output` flag here (that is 020). ⚠️ Step helpers return errors, never panic; reuse shared step phrasings before writing new bindings (LEARNINGS).
