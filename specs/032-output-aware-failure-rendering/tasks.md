# Tasks: Output-Aware Failure Rendering

**Feature**: 032-output-aware-failure-rendering
**Concretization**: Full context (plan + spec + interface + scenarios)
**Inputs**: plan.md, spec.md, interface-cli.md, interface-spec.md, features/opaque-failures/output-aware-failure-rendering.feature

---

## Dependency Graph

Phase 1: Envelope mapping (1 task, no phase dependencies) [Shared]
Phase 2: Format-aware failure chokepoint (1 task, depends on Phase 1) [Shared]

2 tasks total | 1 phase parallelizable: no | Builder: pipeline

T002 depends on T001 (the chokepoint consumes the `errorEnvelopeFor` mapping and the `next_step` field T001 adds).

---

## Branching Guidance

**Pipeline mode**: `spec/032-output-aware-failure-rendering/base` → `spec/032-output-aware-failure-rendering/task-1`, `spec/032-output-aware-failure-rendering/task-2`

T001 lands first (pure additive mapping; the new helper is unused, so the suite stays green). T002 branches from the integrated result — its signature change makes the Go compiler enforce that every call site is threaded in one PR.

---

## Phase 1: Envelope mapping [Shared]

- [x] **T001** [Shared] Add the `next_step` envelope field and the pure Diagnostic→`ErrorEnvelope` mapping (no behavior change yet) — unit tests only (no `@wip` removal: the feature scenarios are end-to-end and pass once T002 wires the chokepoint); `kind` exhaustiveness + `errorEnvelopeFor` (next_step present/omitted, status/body present, body-omitted-when-invalid, transport no status/body, body-survives-refinement) + token-free over both structured formats
  - **Scope**: Two additive changes, no call-site churn. (1) In `internal/output`, add `NextStep string` with tag `json:"next_step,omitempty"` to the existing `ErrorDetail`, placed in declaration order `Message, NextStep, Kind, Status, Body` so the document reads message → next_step → kind → status → body. (2) In `internal/cli`, add an unexported total `kind(Outcome) string` mapper (`UsageError→"usage"`, `RuntimeError→"runtime"`, `NetworkUnavailable→"network"`, `APIError→"api"`, `PermissionError→"permission"`, `RateLimited→"rate-limit"`, defensive `default→"runtime"`) and `errorEnvelopeFor(err error) output.ErrorEnvelope` that refines once, calls `Diagnose`, and maps `Cause→Message`, `NextStep→NextStep`, `kind(Category)→Kind`, and — by extracting the wrapped `*apiclient.ResponseError` (`var re *apiclient.ResponseError; errors.As(err, &re)`) — `StatusCode→Status` and the raw body→`Body` **only when `json.Valid`** (otherwise omitted). The helper is added but not yet called.
  - **Acceptance criteria**:
    - A table-driven test over `kind` covers every `Outcome` value with a `len`+comma-ok (or switch-exhaustiveness) guard so a future added `Outcome` fails loudly rather than silently falling to the default.
    - `errorEnvelopeFor` unit tests assert: `next_step` present for a failure that carries one and **absent** (key omitted, not null) for the internal-error fallback; `status`/`body` present for a typed API error with a JSON body; `body` **omitted** when the API body is not valid JSON while `message`/`kind`/`status` remain; `status`/`body` absent for a transport failure; the body survives `*ResponseError`→`*ProblemError` refinement (reached via `errors.As`).
    - A token-free test asserts no `X-Auth-Token` value appears in any field of the produced envelope for every family.
    - Existing `internal/output` envelope tests (018) pass untouched — the `omitempty` field changes no existing document.
    - `go build` + `go vet` + full `go test ./...` green (CONSTITUTION VII).
  - **Dependencies**: None
  - **Plan reference**: Phase 1 (Implementation Strategy); ADR-2 (Diagnostic→envelope mapping; `next_step` field); ADR-4 (kind map; body-when-valid)
  - **Scenario references**: output-aware-failure-rendering.feature: "The next step survives the structured render as a distinct field", "An internal-error fallback omits the next step in every format", "An API error body is carried verbatim into the envelope", "An API error body that is not valid JSON is omitted from the envelope", "No secret appears in any rendered failure"
  - **Interface references**: interface-spec.md: Surface (`ErrorDetail.NextStep`, `errorEnvelopeFor`, `kind`); interface-cli.md: structured failure document field table

## Phase 2: Format-aware failure chokepoint [Shared]

- [x] **T002** [Shared] Make the single failure chokepoint format-aware and thread the resolved format to every command-execution failure site — 11 scenarios un-`@wip`'d (3 `@validation` held for validate); threaded 38 call sites across 11 files (incl. `domain.go`/`domains.go`/`policies.go` added by 033/034 after planning — drift noted in LEARNINGS, compiler-enforced via the signature change); added the `renderErrorFn` test seam (mirrors `renderFn`) for the render-cannot-complete path
  - **Scope**: Evolve `reportClientError(stderr, err)` into `reportFailure(stdout, stderr io.Writer, format output.OutputFormat, err error) (Outcome, error)`: refine once, `Diagnose`; when `format.MachineFormat()` is structured, write `output.RenderError(machineFmt, errorEnvelopeFor(err))` to **stdout** (buffer-then-write — on a render error, leave stdout empty and return `RuntimeError`); otherwise write `renderDiagnostic(d)` to **stderr** (unchanged). Return `d.Category` and the refined error. Update all ~19 clean-failure call sites (`render.go` ×2, `roles.go`, `subroles.go`, `tree.go`, `me.go`, `me_roles.go`, `me_projects.go`, `me_actions.go`) to pass `cfg.stdout` + the in-scope `format`. Leave `reportIncompleteWalk`/`reportIncompleteSubrolesWalk` unchanged (stderr note in every format — the partial structured document already occupies stdout). Do **not** touch the `dispatch.go` usage path or 020's invalid-selector path.
  - **Acceptance criteria**:
    - Under `json`/`yaml`, a command-execution failure writes one valid `{"error":{…}}` envelope to stdout and nothing to stderr; under `full`/`compact`, stderr keeps today's cause-plus-next-step line and stdout stays empty — pinned by BDD over the four formats.
    - The returned `Outcome` (and thus the exit code) is identical to the pre-032 chokepoint for every family — a regression test renders the same `403` under `full` and `json` and asserts the same exit code 4.
    - A structured render that cannot complete leaves stdout empty and maps to `RuntimeError(1)` (buffer-then-write); the render error message on stderr is token-free.
    - A mid-walk failure with partial data under `json` still writes the partial `{"data":[…]}` to stdout and the incompleteness note to stderr (the `reportIncompleteWalk` path is unchanged); a usage error under `--output json` keeps its plain-text dispatch form.
    - The compiler enforces completeness: no call site retains the old `reportClientError(stderr, err)` form. Existing `me`/`roles`/`subroles`/`tree` tests pass (human-path output byte-stable).
    - The feature file's `@wip` scenarios are un-`@wip`'d as they pass; `go build` + `go vet` + full `go test ./...` green.
  - **Dependencies**: T001
  - **Plan reference**: Phase 2 (Implementation Strategy); ADR-1 (format-aware chokepoint); ADR-3 (channel split + command-execution-only scope; partial-walk stays stderr)
  - **Scenario references**: output-aware-failure-rendering.feature: "A permission failure renders as a JSON envelope on stdout", "A transport failure under json emits the envelope with no status or body", "A human format keeps today's stderr failure line", "The exit code is identical across formats", "A usage error keeps its plain-text form under json", "A structured render that cannot complete writes nothing partial", "A mid-walk failure with partial data keeps its incompleteness note on stderr under json", "A failure conveys the same facts across the four formats"
  - **Interface references**: interface-spec.md: Surface (`reportFailure`), Interactions (call-site threading; out-of-scope symbols); interface-cli.md: Error Communication table
