# Validate: Output-Aware Failure Rendering

**Feature**: 032-output-aware-failure-rendering
**Round**: 1 of 3
**Date**: 2026-06-10
**Verdict**: Ready
**Artifacts loaded**: spec.md, plan.md, tasks.md, interface-spec.md, interface-cli.md, features/opaque-failures/output-aware-failure-rendering.feature, PROJECT.md
**Implementation files**: 14 files in `internal/cli` + `internal/output` — the additive envelope field (`internal/output/error.go`), the cli-side mapping (`internal/cli/errorenvelope.go`), the format-aware chokepoint + test seam (`internal/cli/me.go`), 11 threaded call-site files (`render.go`, `roles.go`, `domain.go`, `domains.go`, `tree.go`, `subroles.go`, `policies.go`, `me_roles.go`, `me_projects.go`, `me_actions.go`), and the godog step definitions (`internal/cli/output_aware_failure_rendering_bdd_test.go`)

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

**Total**: 5 dimensions checked, 5 passed, 0 findings. 3 of 3 validation scenarios satisfied.

---

## Driving Scenario Coverage

**Status**: Pass (9 of 9 scenarios covered)

Every driving scenario in spec.md traces to an identifiable code path, and each is pinned by a now-un-`@wip`'d feature scenario.

| Scenario (spec.md) | Status | Implementation |
|---|---|---|
| A permission failure renders as JSON on stdout | ✓ Covered | `internal/cli/me.go` `reportFailure` (structured branch → `errorEnvelopeFor` → stdout); status/kind/message from `errorenvelope.go` |
| A human format keeps today's stderr line | ✓ Covered | `reportFailure` human branch → `renderDiagnostic(d)` to stderr; stdout untouched |
| The next step survives the structured render as a distinct field | ✓ Covered | `output.ErrorDetail.NextStep` (`json:"next_step,omitempty"`); `errorEnvelopeFor` maps `d.NextStep` |
| A transport failure under json still emits the envelope | ✓ Covered | `errorEnvelopeFor` — no `*ResponseError` → no status/body; `kind` = "network"; envelope still rendered |
| An API error body is carried verbatim into the structured render | ✓ Covered | `errorEnvelopeFor` sets `Body = json.RawMessage(re.Body)` when valid; nested as structured data by 018's `RenderError` |
| The exit code is unchanged by the format | ✓ Covered | `reportFailure` returns `d.Category` in both branches; 004's `ExitCode` maps it |
| An internal-error fallback omits the next step in every format | ✓ Covered | `omitempty` on `NextStep`; fallback `Diagnose` carries no next step → key absent |
| A usage error keeps its plain-text form even under json | ✓ Covered | usage errors route through `dispatch.go`, never `reportFailure` (verified: no `reportFailure` reference in dispatch.go) |
| A structured render that cannot complete writes nothing partial | ✓ Covered | `reportFailure` buffer-then-write: on `renderErrorFn` error, stdout stays empty, returns `RuntimeError` |

---

## Acceptance Criteria

**Status**: Pass (both tasks checked; all criteria met)

**T001 — `next_step` field + Diagnostic→ErrorEnvelope mapping**

| Criterion | Status | Evidence |
|---|---|---|
| Table-driven `kind` test with exhaustiveness guard (future `Outcome` fails loudly) | ✓ Met | `errorenvelope_test.go` `TestKind_Table` (covered-set + len guard) + `TestKind_DefaultArmIsRuntime` |
| `errorEnvelopeFor`: next_step present/absent; status/body present for JSON-body API error; body omitted when non-JSON (message/kind/status remain); status/body absent for transport; body survives `*ResponseError`→`*ProblemError` refinement | ✓ Met | `errorenvelope_test.go` `TestErrorEnvelopeFor` (all six sub-cases present) |
| Token-free test over every family | ✓ Met | `TestErrorEnvelopeFor_TokenFree` (7 families × {JSON, YAML}) |
| Existing 018 envelope tests pass untouched (omitempty additive) | ✓ Met | `internal/output` suite green; `next_step` omitempty changes no existing document |
| `go build` + `go vet` + full `go test ./...` green | ✓ Met | Full suite: PASS |

**T002 — format-aware chokepoint + threading**

| Criterion | Status | Evidence |
|---|---|---|
| json/yaml → one valid envelope on stdout, nothing on stderr; full/compact → cause-plus-next-step on stderr, stdout empty (BDD over four formats) | ✓ Met | feature scenarios "permission failure…", "transport failure…", "human format keeps…", and the parity validation trace; `reportFailure` branches |
| Returned `Outcome` (exit code) identical to pre-032 chokepoint per family; regression renders same 403 under full + json, same exit 4 | ✓ Met | feature scenario "exit code is identical across formats"; `reportFailure` returns `d.Category` in both branches |
| Structured render that cannot complete → stdout empty, `RuntimeError(1)`, token-free stderr | ✓ Met | feature scenario "structured render that cannot complete"; `renderErrorFn` seam |
| Mid-walk partial under json keeps partial `{data:[…]}` on stdout + note on stderr; usage error under `--output json` keeps plain-text | ✓ Met | feature scenarios "mid-walk failure…" and "usage error keeps its plain-text form…"; `reportIncompleteWalk` unchanged |
| Compiler enforces completeness (no old `reportClientError` form); existing me/roles/subroles/tree tests pass (human output byte-stable) | ✓ Met | zero `reportClientError` survivors in non-test code; `TestReportFailure_*` + read-command suites green |
| `@wip` scenarios un-`@wip`'d as they pass; build + vet + full test green | ✓ Met | 11 scenarios un-`@wip`'d; full suite PASS |

---

## Interface Contract Conformance

**Status**: Pass (all surfaces conformant)

**interface-spec.md** (Go symbols):

| Surface | Status | Evidence |
|---|---|---|
| `ErrorDetail.NextStep string` tag `json:"next_step,omitempty"`, declaration order `Message, NextStep, Kind, Status, Body` | ✓ Conformant | `internal/output/error.go` — exact field order and tag |
| `reportFailure(stdout, stderr io.Writer, format output.OutputFormat, err error) (Outcome, error)` | ✓ Conformant | `internal/cli/me.go` — exact signature |
| `errorEnvelopeFor(err error) output.ErrorEnvelope` | ✓ Conformant | `internal/cli/errorenvelope.go` |
| `kind(Outcome) string` — usage/runtime/network/api/permission/rate-limit + default "runtime" | ✓ Conformant | `internal/cli/errorenvelope.go` `kind` |
| Consumed-unchanged symbols (`Diagnose`, `renderDiagnostic`, `refineClientError`, `MachineFormat`, `RenderError`, `*ResponseError`) | ✓ Conformant | all consumed, none redefined |
| 018 leaf invariant: `internal/output` imports no transport | ✓ Conformant | no `apiclient` import in `internal/output` (only comment references) |

**interface-cli.md** (operator-facing surface):

| Surface | Status | Evidence |
|---|---|---|
| No new flag/command | ✓ Conformant | reuses `--output`; no command added |
| Structured failure document fields: `message` always, `next_step` when present, `kind` always, `status` non-2xx only, `body` valid-JSON only | ✓ Conformant | `errorEnvelopeFor` + `omitempty` tags |
| Human failure line unchanged (cause alone, else cause — next step); stdout empty | ✓ Conformant | `reportFailure` human branch / `renderDiagnostic` |
| Exit-code table (1/3/4/5/6) unchanged | ✓ Conformant | no `ExitCode` change; `d.Category` returned unchanged |

---

## Non-Behavior Absence

**Status**: Pass (no excluded behavior present)

| Non-behavior (spec.md) | Status | Evidence |
|---|---|---|
| Must not classify / compose cause, category, next step | ✓ Absent | `errorEnvelopeFor` reads `Diagnose` output (031); `kind` maps an existing `Outcome`, does not classify |
| Must not select / resolve the output format | ✓ Absent | `reportFailure` consumes the `format` parameter via `MachineFormat`; no resolution |
| Must not define the envelope shape or implement encoders | ✓ Absent (see note) | reuses `output.ErrorEnvelope`/`RenderError`; the additive `next_step` field is the spec-sanctioned extension (see note below) |
| Must not emit / decide the exit code | ✓ Absent | returns an `Outcome`; never calls `ExitCode` or emits a code |
| Must not render the invalid-selector usage error | ✓ Absent | `reportFormatResolutionError` (020) unchanged — stderr plain text |
| Must not render usage errors in the structured format | ✓ Absent | usage errors route through `dispatch.go`, never `reportFailure` |
| Must not render successful results | ✓ Absent | `reportFailure` is failure-only; success uses `RenderSuccess` |
| Must not retry, wait, re-parse the body, or interpret 403 as a plan signal | ✓ Absent | `errorEnvelopeFor` carries `re.Body` verbatim as `json.RawMessage`; `json.Valid` is a syntactic gate, not a re-parse; no retry/wait/plan logic |
| Must not fabricate a next step or any fact | ✓ Absent | `NextStep` mapped from `d.NextStep` only; `omitempty` omits when absent — never invented |

**Note (envelope-shape extension)**: T001 adds the `NextStep` field to `output.ErrorDetail` in `internal/output`. This is not a non-behavior violation: spec.md § Assumptions ("Distinct next-step element in the envelope") and § Clarifications (2026-06-10) explicitly sanction *extending 018's envelope with a next-step element*, and ADR-2 / interface-spec.md place the field **declaration** in 018's home (`internal/output`) while the **population/mapping** lives in `internal/cli`. The 018 leaf invariant (no transport import, no classification) is preserved. Envelope ownership stays with 018; 032 maps facts into it.

---

## @wip Lifecycle Completion

**Status**: Pass

The feature file's 11 behavioral scenarios (referenced by the two checked tasks) have had their `@wip` tags removed and pass under the suite's `~@wip` filter. The 3 remaining `@wip` tags are all on `@validation`-tagged scenarios — correctly held out for this validate pass (not referenced by any checked task as Builder work):

- `@validation @wip` — "A failure conveys the same facts across the four formats"
- `@validation @wip` — "No secret appears in any rendered failure"
- `@validation @wip` — "The failure rendering exposes only observable behavior"

---

## Validation Scenario Results

**Status**: Satisfied (3 of 3 scenarios traced to implementation)

| Scenario | Status | Trace |
|---|---|---|
| Failure parity across the four formats | ✓ Satisfied | `reportFailure`: full/compact → `renderDiagnostic` (cause + next step) to stderr; json/yaml → `errorEnvelopeFor` (`message` + `next_step`) to stdout as one complete `RenderError` document. Same facts across all four; human→stderr, structured→stdout |
| No secret leaks into any rendered failure | ✓ Satisfied | Human path renders token-free `Diagnose` output; structured path carries only response-side fields and `RenderError` adds nothing. `TestErrorEnvelopeFor_TokenFree` asserts the sentinel token absent across every family × {JSON, YAML} |
| No implementation leakage in the artifact | ✓ Satisfied | spec.md names only channels (stdout/stderr), shapes (envelope, distinct next-step element), and facts (cause/next-step/kind/status/body); no language, package layout, or signature — those live in plan/interface |

---

## Verdict: Ready

All 5 conformance dimensions pass with zero findings. All 3 held-out validation scenarios are satisfied through independent inspection (one — no-secret-leaks — additionally pinned by a token-free unit test). Both tasks are checked, the full Go suite is green, `go vet` and `golangci-lint` are clean. The implementation conforms to its specification: command-execution failures render as 018's envelope on stdout under `json`/`yaml` and as the unchanged cause-plus-next-step line on stderr under `full`/`compact`, the next step is a distinct parseable element, the exit code is unchanged across formats, and the usage-error / invalid-selector / partial-walk / success paths are all left untouched as the non-behaviors require.

---

## Next Steps

Implementation conforms to the specification. Suggest PR review and merge. The specification loop is closed for 032.
