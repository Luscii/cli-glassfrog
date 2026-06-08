# Validate: Structured Serialization

**Feature**: 018-structured-serialization
**Round**: 1 of 3
**Date**: 2026-06-08
**Verdict**: Ready
**Artifacts loaded**: spec.md, plan.md, tasks.md, interface-spec.md, features/unconsumable-output/structured-serialization.feature, PROJECT.md
**Implementation files**: 3 in `internal/output/` (output.go, error.go, structured_serialization_bdd_test.go) plus output_test.go, error_test.go; go.mod/go.sum (`sigs.k8s.io/yaml v1.4.0`)

---

## Conformance Summary

| Dimension | Status | Findings |
|---|---|---|
| Driving scenario coverage | ✓ Pass | 0 |
| Acceptance criteria | ✓ Pass | 0 |
| Interface contract conformance | ✓ Pass | 0 |
| Non-behavior absence | ✓ Pass | 0 |
| @wip lifecycle completion | ✓ Pass | 0 |
| **Validation scenarios** | ✓ Satisfied | 0 |

**Total**: 5 dimensions checked, 5 passed, 0 findings. 5 of 5 validation scenarios traced.

---

## Driving Scenario Coverage

**Status**: Pass (7 of 7 scenarios covered)

All driving scenarios from spec.md § Driving Scenarios are concretized as behavioral (`@wip`-removed) scenarios in the feature file and have identifiable code paths. The component-level godog suite (`TestStructuredSerializationFeatures`) reports 9 scenarios / 43 steps passing.

| Scenario (spec.md) | Status | Implementation |
|---|---|---|
| serialize a successful read as JSON | ✓ Covered | `output.go:86-93` `RenderSuccess` JSON arm (`json.Indent`) |
| serialize the same result as YAML | ✓ Covered | `output.go:94-99` `RenderSuccess` YAML arm (`yaml.JSONToYAML`) |
| full fidelity, not the human projection | ✓ Covered | `output.go:88-93` raw-bytes path; no struct re-encode (ADR-2) |
| non-2xx with an API error body | ✓ Covered | `error.go:48-61` `RenderError`; `Body json.RawMessage` nests verbatim |
| transport failure with no API payload | ✓ Covered | `error.go:30-34` `Status`/`Body` omitempty → bodiless envelope |
| empty result is a valid document | ✓ Covered | `output.go:79-81` empty/whitespace → `null` (never empty channel) |
| the secret never appears | ✓ Covered | token is never a response field; renderer adds nothing (`error.go:48`, `output.go:78`) |

---

## Acceptance Criteria

**Status**: Pass (3 of 3 tasks checked; all criteria met)

| Task | Status | Evidence |
|---|---|---|
| T001 — `Format` + `RenderSuccess` + `sigs.k8s.io/yaml` | ✓ Met | Field fidelity, JSON≡YAML, large-int precision, determinism, empty-doc, invalid-JSON render error all in `output_test.go`; `go list -deps ./internal/output/` shows no internal-package deps; build/vet clean |
| T002 — `ErrorEnvelope`/`ErrorDetail` + `RenderError` | ✓ Met | API envelope w/ status+verbatim body, bodiless omits status/body, JSON≡YAML, each kind, token-free stays token-free, invalid body → render error w/ no partial doc — all in `error_test.go`; no `apiclient` import |
| T003 — component-level godog suite | ✓ Met | `structured_serialization_bdd_test.go`: 9 behavioral scenarios pass; suite `Paths` names only its own feature file; 5 `@validation` kept `@wip`; hermetic (no network/home/fs) |

---

## Interface Contract Conformance

**Status**: Pass (5 of 5 surfaces conformant)

Compared `internal/output`'s Go surface against interface-spec.md § Surface.

| Surface symbol | Status | Implementation |
|---|---|---|
| `Format` enum (`JSON`, `YAML`) | ✓ Conformant | `output.go:35-43` |
| `RenderSuccess(f Format, payload json.RawMessage) ([]byte, error)` | ✓ Conformant | `output.go:78` — signature matches exactly |
| `ErrorEnvelope` (one `ErrorDetail` under `error` key) | ✓ Conformant | `error.go:17-19` — `json:"error"` |
| `ErrorDetail` (`Message` always, `Kind` always, `Status` omitempty, `Body` omitempty) | ✓ Conformant | `error.go:24-35` — field set and omitempty tags match |
| `RenderError(f Format, env ErrorEnvelope) ([]byte, error)` | ✓ Conformant | `error.go:48` — signature matches exactly |

External document shapes also match: the success document is the raw payload verbatim in the active format; the error envelope renders as `{ "error": { "message", "kind", "status"?, "body"? } }` with `status`/`body` absent (not null-keyed) for bodiless failures. The "Consumed unchanged" boundary holds — `internal/apiclient` and `internal/cli` are untouched on this branch (`Execute` accepts `*json.RawMessage` as `out` with no 010 change), and 018 imports neither (`go list -deps` clean).

---

## Non-Behavior Absence

**Status**: Pass (6 of 6 exclusions respected)

| Non-behavior (spec.md § Non-Behaviors) | Status | Evidence |
|---|---|---|
| must not parse/own the `--output` flag | ✓ Absent | No cobra/flag code in `internal/output` (the only `--output` mention is a forward-reference doc comment) |
| must not produce human `full`/`compact` rendering | ✓ Absent | No template rendering; package ships encoders only (019 owns the human branch) |
| must not classify/interpret/enrich an API error | ✓ Absent | `RenderError` encodes the `kind`/`status` it is handed; no `classifyClientError`, no `apiclient` import (mapping deferred to 020) |
| must not decide the process exit code | ✓ Absent | No `os.Exit`/`ExitCode` in the package |
| must not reshape/summarize/drop fields from a successful payload | ✓ Absent | `RenderSuccess` serializes raw bytes (`json.Indent` / `JSONToYAML`), never a typed struct (ADR-2) |
| must not emit the token or auth header | ✓ Absent | Renderer introduces no content; token is a request header, never a response field |
| must not emit a partial/truncated document | ✓ Absent | Both renderers build the whole document in memory and return `nil, err` on failure (`output.go:82-83,89-91`; `error.go:57-59,63-70`) |

---

## @wip Lifecycle Completion

**Status**: Pass

The 9 behavioral scenarios referenced by T003 had their `@wip` removed and pass executably. The 5 `@validation` scenarios retain `@validation @wip` — held out for this validation step, referenced by no checked task's behavioral set, so their `@wip` is correct, not a lifecycle gap.

---

## Validation Scenario Results

**Status**: Satisfied (5 of 5 scenarios traced to implementation)

These were held out from the Builder. Traced independently against the code (not assumed from the driving-scenario pass).

| Scenario | Status | Trace |
|---|---|---|
| JSON and YAML are the same data in two encodings | ✓ Satisfied | YAML is a transform of the same JSON bytes (`yaml.JSONToYAML`, `output.go:95`); equivalence holds by construction, not parallel encoders. Pinned by `TestRenderSuccess_YAMLIsValidAndCarriesSameData` and bdd `compareYAMLToJSON`. |
| the channel is always a complete, valid document | ✓ Satisfied | All four outcomes route through one `RenderSuccess`/`RenderError` call that returns a complete doc or `(nil, err)`; no partial write path exists. Bodiless + API + fail-safe (`kind:"runtime"`) all produce one envelope. |
| this capability classifies nothing | ✓ Satisfied | `RenderError` reads `Kind`/`Status` from the passed `ErrorEnvelope`; no status-specific branch. bdd `thenNotClassifyOrInterpret` asserts the rendered `kind`/`status` equal the supplied values exactly. |
| every failure shares one error shape | ✓ Satisfied | Single `ErrorEnvelope`/`ErrorDetail` type; `Status`/`Body` omitempty so inapplicable fields are absent, never renamed (`error.go:24-35`). |
| reshaping is absent | ✓ Satisfied | `RenderSuccess` operates on raw bytes; `TestRenderSuccess_JSONPreservesFieldsTheProjectionDrops` confirms a `_links` field the typed struct would drop survives verbatim. |

*Note*: the 5 `@validation` scenarios are not executed here — their step phrasings are intentionally unbound in the component suite (held-out). They are verified by inspection plus the equivalent already-passing behavioral/unit assertions cited above.

---

## Verdict: Ready

All 5 conformance dimensions pass with 0 findings. All 5 validation scenarios are satisfied through independent inspection. The implementation conforms to the specification.

Scope is correctly bounded to 018's contract: `internal/output` ships the `Format` enum, `RenderSuccess`, and the unified `ErrorEnvelope`/`RenderError` as pure encoders, with no flag, no command, no classification, and no typed-error→envelope mapping (all deferred to Output Format Selection 020). The plan-documented coverage limitation stands and is not a finding: there is no CLI surface until 020, so end-to-end `--output json` on a real command is not yet observable through the binary — verified here at the component level, as the plan and tasks prescribe.

---

## Next Steps

Implementation conforms to the specification. Suggest PR review and merge. The specification loop for 018 is closed; end-to-end verification of the structured path through a command follows with Output Format Selection (020).
