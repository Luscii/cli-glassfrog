# Plan: Structured Serialization

**Feature**: 018-structured-serialization
**Role**: Shaper
**Inputs**: spec.md (018-structured-serialization); PROJECT.md; CONSTITUTION.md; `.score/memory/DECISIONS.md` (relevant precedent: API models live in `internal/glassfrog`, decoded into shared schema types, tolerant of unknown/extra fields — 011 ADR-1; the read surface classifies 010's typed errors through one `classifyClientError` into the `Outcome` enum, producer-classifies/consumer-maps — 011 ADR-3 / 002 / 004; pure `run<Read>`+`format<Read>` projection behind an injected seam, the reshaped projection is the default and `--output json` a deferred cross-cutting flag — 011 ADR-5; the first capability to need a shared home creates it, siblings reuse — 005/006/007 + the `glassfrog` envelope; 010's `Execute(reqCtx, req, out any)` send seam decodes a 2xx into `out` and surfaces a generic `*ResponseError{StatusCode, Header, Body []byte}` on non-2xx); `.score/memory/LEARNINGS.md` (inject seams / fail-fast on nil; a godog suite points at its OWN feature file; pure functions unit-tested; temp-file stderr capture; no-silent-failure in helpers — PR #10/#20); `.score/memory/DEPRECATION.md` (no entry touches the output surface). No SOUL.md.

---

## System Architecture

Structured Serialization is the **machine-rendering branch** of the Output Formatting solution. It is a new pure leaf package, `internal/output`, that turns a command's result into a JSON or YAML document — and renders failures as one unified error envelope in the same format. It owns no flag, no transport, no command surface: it is a transformation, imported by `internal/cli` the way `internal/glassfrog` (schema) and `internal/apiclient` (transport) are, with no import cycle.

The defining constraint is fidelity. Fork B (spec) requires the **raw API payload, verbatim — all fields intact**. But `Execute` decodes a 2xx body into the *tolerant* typed `glassfrog` structs (011 ADR-1 — unknown fields ignored) and discards the raw bytes, so re-encoding a struct would silently drop fields and risk number-precision loss. The design therefore serializes the **raw response bytes**, never the typed struct.

```
[ Output Format Selection (020) / command surface ]
        │  active Format (json | yaml)        ── selection & routing are 020's
        │
   success ─ decode 2xx body into json.RawMessage   ── verbatim, no struct (ADR-2)
        │        (a valid `out` for the existing Execute — no 010 change)
        ▼
   output.RenderSuccess(format, raw)  ── JSON: normalize bytes · YAML: yaml.JSONToYAML(raw)
        │
        ▼  one complete document ──────────────────────────────────► stdout
   error ─ typed err ─(cli maps via classifyClientError taxonomy)─► output.ErrorEnvelope
        │        { message · kind · status? · body? }  (raw body from ResponseError.Body)
        ▼
   output.RenderError(format, envelope) ── same active format ─────► stdout
```

`internal/output` ships three things: a `Format` enum (`JSON`, `YAML` — the machine formats 018 supports), `RenderSuccess(Format, json.RawMessage)`, and the `ErrorEnvelope` type with `RenderError(Format, ErrorEnvelope)`. Selection of the format, the decode-target choice, and the typed-error→envelope mapping happen at the `internal/cli` / 020 boundary; 018's renderers are pure encoders of the values they are handed. This is the same package that Templated Human Rendering (019) will extend with template rendering and Output Format Selection (020) will extend with the `--output` flag and its router.

---

## Architecture Decisions

### ADR-1: A new `internal/output` leaf package owns the machine serializers and the unified error envelope

**Context**: 011 ADR-5 deferred `--output json` as a cross-cutting capability; the human `format<Read>` projections live in `internal/cli`. Serialization is cross-cutting across every command and will be joined by 019 (human templates) and 020 (selection). It needs a home that `internal/cli` can import without a cycle, holding no cobra and no transport.

**Options considered**:
1. **New `internal/output` leaf package** — pure, no cobra/transport, imported by `cli`. Mirrors the `glassfrog`/`apiclient` leaf pattern and the "first-to-land creates the shared home" precedent (005/006/007).
2. **Inside `internal/cli`** alongside the `format<Read>` projections — keeps rendering together, but entangles a cross-cutting capability with command wiring and gives 019/020 no clean seam.
3. **Inside `internal/glassfrog`** — wrong layer; `glassfrog` is schema-only, and serialization is not a domain model.

**Decision**: Option 1. `internal/output` is created by this spec as a pure leaf; 019 adds template rendering, 020 adds the selector and flag. CONSTITUTION VI (modular, independently-testable parts) favors a dedicated, hermetic package over folding rendering into `cli`.

**Consequences**: One-directional dependency `cli → output`. 019/020 extend the same package without touching `cli` command bodies beyond their own wiring. The exact package API (function names, the `Format` spelling) is interface-level. *First-to-land-creates precedent applied; sets the home for the whole Output Formatting solution.*

### ADR-2: Serialize the raw response bytes, not the tolerant typed structs

**Context**: Fork B requires the raw payload verbatim, all fields intact. `Execute` decodes the 2xx body into typed `glassfrog` structs that ignore unknown/extra fields (011 ADR-1) and does not retain the raw bytes; decoding JSON numbers into a generic `any` would coerce them to `float64` and lose precision for large integer ids/values.

**Options considered**:
1. **Re-encode the typed struct** — simplest, but lossy: drops every field the struct doesn't model and risks number precision. Violates fork B. Rejected.
2. **Decode into generic `any`/`map[string]any`** — preserves field presence but needs `json.Number` to avoid float64 precision loss, plus a custom encode path for both formats. More code, a real precision-edge risk.
3. **Capture the raw 2xx body verbatim via a `json.RawMessage` decode target and serialize the bytes** — JSON: normalize/indent the bytes; YAML: `yaml.JSONToYAML(raw)`. The error body (`ResponseError.Body`) is already raw and nests verbatim.

**Decision**: Option 3. When a structured format is active, the command decodes the 2xx body into `json.RawMessage` — a valid `out` for the **existing** `Execute`, so 010 is unchanged — instead of the typed struct; the typed struct path is retained for the human projection (019). `output.RenderSuccess` renders the bytes: JSON via stdlib normalization, YAML via `sigs.k8s.io/yaml.JSONToYAML`.

**Consequences**: Full fidelity, no precision loss, and "JSON ≡ YAML" holds by construction because YAML is derived from the same JSON bytes (spec validation scenario). No transport change. The decode-target choice (raw vs typed, keyed on the active format) is wiring that 020 owns; 018 only renders. Determinism: stdlib JSON normalization and struct-ordered envelope marshalling are stable for a given input.

### ADR-3: YAML via `sigs.k8s.io/yaml` (`JSONToYAML` on raw bytes) — new build dependency [developer-approved]

**Context**: Go has no stdlib YAML. Fork B (raw fidelity) and the same-data guarantee both favor transforming the raw JSON bytes rather than a typed value. CONSTITUTION XII (self-contained executable) governs **runtime** installs on the operator's machine — a Go module statically links into the binary and is not a runtime dependency, so adding one is compliant.

**Options considered**:
1. **`sigs.k8s.io/yaml`** — exposes `JSONToYAML([]byte) ([]byte, error)`, converting the raw JSON bytes directly to YAML; preserves every field and avoids float64 precision loss, and makes YAML a faithful transform of the JSON. Pulls `gopkg.in/yaml.v2` transitively.
2. **`gopkg.in/yaml.v3`** — one idiomatic direct dependency, but raw fidelity requires decoding JSON→`any` with `json.Number` and manually handling number/`any` encoding — more code and a precision-edge risk.
3. **Hand-roll YAML** — no dependency, but correct YAML emission (quoting, special characters, multiline) is genuinely hard; high defect risk. Rejected.

**Decision**: Option 1 (`sigs.k8s.io/yaml`), approved by the developer. `JSONToYAML` keeps YAML a byte-faithful transform of the raw JSON.

**Consequences**: One new `go.mod` entry (plus `yaml.v2` transitive), statically linked — XII-compliant. PROJECT.md's stack should record it. The library's `YAMLToJSON` is unused. This is the only outside dependency 018 introduces.

### ADR-4: The unified error envelope; `kind` reuses the `classifyClientError` taxonomy — no *build* dependency on 015

**Context**: Clarify resolved that failures render as one **unified error envelope** that 018 defines, in the active format, including for bodiless failures (transport, fail-safe refusal). API Error Extraction (015) has since **landed**: it added the typed `*ProblemError` (wrapping `*ResponseError`) and widened `classifyClientError` to split non-2xx by status — `Outcome` is now Success / UsageError / RuntimeError / NetworkUnavailable / APIError plus **PermissionError (401/403→4)** and **RateLimited (429→5)**. `*ResponseError` still carries `StatusCode` + raw `Body`, and `Execute(out any)` is unchanged. The question is whether 018's envelope should *own its shape* or be coupled to 015's classified-error type.

**Options considered**:
1. **Define the envelope shape in 018; populate it from the classifier taxonomy + `*ProblemError` detail at the consuming boundary** — 018 owns a stable shape independent of 015's internal types; `kind` aligns with the exit-code taxonomy the agent already sees.
2. **Couple 018's envelope to 015's `*ProblemError` type directly** — would drag `internal/apiclient` into `internal/output` and tie the output shape to classification internals; rejected.
3. **Emit only a message, no kind/status** — under-serves the agent operator and weakens the "responses as expected" intent. Rejected.

**Decision**: Option 1. 018 owns the envelope **shape** — `{ error: { message, kind, status?, body? } }`. The mapping typed-error→envelope reuses the `classifyClientError` discrimination (kind from its arms, now incl. `permission`/`rate-limit`; `status` + raw `body` from `*ResponseError`; `message` from the token-free error string, optionally enriched from 015's `*ProblemError` detail). For bodiless failures, `status`/`body` are omitted. The mapping helper lives at the `cli`/020 boundary (next to the taxonomy and 015's `*ProblemError`), so 018's `RenderError` stays a pure encoder of a plain struct and `internal/output` need not import `internal/apiclient`.

**Consequences**: 018's envelope shape is independent of 015's classification internals; with 015 landed, the mapping can already carry its `*ProblemError` detail into `body`/`message`, and the `kind` slot absorbs the widened taxonomy without a shape change. Exact field names are interface-level. *Sets the error-envelope precedent for the output surface; composes with 015 (no divergence) and 020 (routing).*

---

## Data Model Design

All types live in `internal/output` and are plain data — no transport, no cobra, no domain coupling.

- **`Format`** — an enum of the machine formats 018 supports: `JSON`, `YAML`. (020 extends the *selection* vocabulary with the human `full`/`compact` formats; the json/yaml identifiers are 018's.)
- **Success input** — `json.RawMessage`: the raw 2xx body captured verbatim (ADR-2). `RenderSuccess(Format, json.RawMessage) ([]byte, error)` normalizes it as JSON or transforms it via `JSONToYAML`. An empty/absent body renders as a valid empty document, never an empty channel (spec edge case).
- **`ErrorEnvelope` / `ErrorDetail`** — `Message string` (always present), `Kind string` (always — the taxonomy term: `usage` / `runtime` / `network` / `api`, mapped from `classifyClientError`), `Status int` (omitempty — the HTTP status, present only for `*ResponseError`), `Body json.RawMessage` (omitempty — the raw API error body verbatim when present). `RenderError(Format, ErrorEnvelope) ([]byte, error)`.

Number fidelity: both render paths operate on bytes (`JSONToYAML`, stdlib JSON normalization), so no JSON number is ever coerced to `float64`. Secret hygiene: the token is a request header, never a response field (011 ADR-1), so raw passthrough cannot carry it; the envelope `message` derives from typed errors that already exclude the token.

---

## Integration Design

- **Output Format Selection (020 — downstream dependent)**: selects the active `Format`, chooses the decode target (`json.RawMessage` when structured, the typed struct for human output), routes success and error through `internal/output`, and owns the invalid-selector bootstrap (e.g. `--output=xml`). Import direction `cli → output`.
- **Templated Human Rendering (019 — parallel sibling)**: extends `internal/output` with template rendering over the typed projection. Parallel branch; neither depends on the other.
- **API Error Extraction (015 — landed composition boundary)**: supplies the widened `Outcome` taxonomy (`permission`/`rate-limit`) and the `*ProblemError` detail the (020-owned) mapping carries into the envelope's `body`/`message`; the envelope shape (ADR-4) does not change. No build dependency either direction (`internal/output` does not import `internal/apiclient`).
- **Request Execution (010 — upstream, unchanged)**: `Execute` already accepts `json.RawMessage` as `out` (raw 2xx capture) and already surfaces the raw error `Body` on `*ResponseError`. No change to 010.
- **`internal/cli` error taxonomy (011/002/004)**: the typed-error→envelope mapping reuses `classifyClientError`'s `errors.As` chain rather than reimplementing discrimination.
- **`sigs.k8s.io/yaml`**: the YAML encoder (`JSONToYAML`); `encoding/json` is the JSON encoder/normalizer.

---

## Cross-cutting Concerns

- **Secret hygiene (CONSTITUTION II)**: covered by construction (token never a response field; envelope message from token-free typed errors). A unit test asserts the token never appears in any rendered document, success or error.
- **No partial documents**: a render failure (e.g. a 2xx body that is not valid JSON, which the contract says should not happen) must surface a typed render error that the command maps to `RuntimeError` — never a truncated or half-written document (spec Non-Behavior). The renderer builds the full document in memory before any write.
- **Determinism**: stdlib JSON normalization and struct-ordered envelope marshalling are stable; YAML derives deterministically from the JSON bytes.
- **Output discipline**: the structured document is the sole content on stdout (the channel a consumer parses); the existing more-available note and diagnostics ride stderr (012 convention). Whether that note should itself become structured under a machine format is 020's call, noted below — 018 does not change stderr behavior.
- **Testing (CONSTITUTION IV)**: `internal/output` is pure and tested hermetically with hand-built inputs (raw byte fixtures, constructed envelopes) — no network. RED-first. Because 018 has no CLI surface until 020 wires `--output`, the driving scenarios run at the **component level** over the `output` package functions (a godog suite pointed at its own feature file, the 011 pattern), not end-to-end through a command. End-to-end `--output json` on a real command lands with 020.

---

## Implementation Strategy

**Phase 1 — `internal/output` success serializers.** Create the package; add `sigs.k8s.io/yaml` to `go.mod`. Define `Format`; implement `RenderSuccess` (JSON normalization + `JSONToYAML`), including the empty-body case. Pure unit tests: raw fidelity (a field the typed struct would drop survives), JSON≡YAML equivalence, number precision, determinism. *(Depends on nothing.)*

**Phase 2 — the unified error envelope.** Define `ErrorEnvelope`/`ErrorDetail` and `RenderError` in `internal/output`. Unit tests over hand-built envelopes: API error (status + raw body nested), bodiless failure (message only), each `kind`, secret-absence, and no-partial-document on a render failure. The typed-error→envelope *mapping* (kind from `classifyClientError`, status/body from `*ResponseError`) is the consuming surface's and lands with 020 — not built here (ADR-4; interface-spec). *(Depends on Phase 1's `Format`.)*

**Phase 3 — component-level scenarios.** Author `features/unconsumable-output/structured-serialization.feature` from the driving scenarios and wire a hermetic godog suite over the `output` package. *(Depends on Phases 1–2.)*

Phases 1 and 2 are largely parallel after `Format` exists; Phase 3 follows both.

---

## Risks

- **Envelope shape vs 015's classification** (resolved — 015 has landed): 015 widened the taxonomy (`permission`/`rate-limit`) and added `*ProblemError` *without* requiring an envelope shape change — the `kind` slot absorbs the new categories and `body`/`message` absorb the extracted detail. *Mitigation*: ADR-4 owns the shape in 018; 015's detail flows in via the 020 mapping, not a shape change.
- **No CLI surface at 018 → a coverage gap until 020** (medium likelihood, low impact): serialization behavior isn't observable through the binary until `--output` exists. *Mitigation*: thorough pure unit tests plus component-level godog over the package; end-to-end deferred to 020 and noted for the Verifier.
- **A 2xx body that is not valid JSON** (very low likelihood — contract violation): normalization/`JSONToYAML` would fail. *Mitigation*: surface a typed render error → `RuntimeError`, never a partial document.
- **New dependency** `sigs.k8s.io/yaml` (low impact): small, well-maintained, statically linked (XII-compliant), developer-approved. *Mitigation*: record it in PROJECT.md's stack.

---

## What This Plan Does Not Cover

- **The `--output` flag** — its vocabulary, default, per-invocation selection, decode-target wiring, and the invalid-selector bootstrap → Output Format Selection (020).
- **Human `full`/`compact` templates** → Templated Human Rendering (019).
- **Richer API error extraction / messages** → API Error Extraction (015); 018 nests the raw body and a taxonomy `kind`.
- **Exact envelope field names and the package's exact public API** → the interface skill.
- **Executable scenario text and step definitions** → the scenarios skill; **task decomposition** → the tasks skill.
- **Whether the stderr more-available note becomes structured under a machine format** → 020's routing decision.
