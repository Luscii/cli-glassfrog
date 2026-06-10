# Checklist: Output-Aware Failure Rendering

**Feature**: 032-output-aware-failure-rendering
**Checked against**: CONSTITUTION.md (12 principles)
**Artifacts checked**: spec.md, plan.md, interface-cli.md, interface-spec.md, features/opaque-failures/output-aware-failure-rendering.feature, tasks.md
**Checks**: 15 (15 pass, 0 fail)
**Generated**: 2026-06-10

---

## Summary

All 15 checks pass. Constitution: 15/15. (Done-criteria and cross-reference checks not generated — no `accords/` directory exists in this project.)

---

## Calibration

Two broad MUST principles were calibrated to binary, feature-specific assertions before evaluation:

- **II. Action Transparency** → for this feature: (a) under every resolved format the rendered failure conveys the diagnostic's cause and (where one exists) its next step; (b) the structured formats emit a machine-parseable document — 018's envelope on stdout — carrying `next_step` as its own parseable field; (c) the rendering pairs with the unchanged 004 exit code (the machine success/failure signal).
- **III. Fail Safe, Not Silent** → for this feature: (a) a command-execution failure is never silent — every format emits the failure (a stderr line, or one stdout envelope), never an empty channel; (b) a structured render that cannot complete writes nothing partial and maps to `RuntimeError(1)`, never a half-document or exit 0; (c) the partial-walk incompleteness signal is preserved (stderr note + non-zero exit) under structured formats, never dropped.

---

## Constitution Checks: 15/15 passed

### Passed (15/15)

**P0** | CONSTITUTION I (Spec Fidelity): invents no endpoint, parameter, or behavior
→ **spec.md § Non-Behaviors + System Overview**: 032 renders already-produced failures and makes no API request; it adds no command and no spec operation. Pass.

**P0** | CONSTITUTION II (Action Transparency): every error explains what went wrong, in every format
→ **spec.md § Behavioral Accord (Rendering a failure) + interface-cli.md § structured failure document**: human formats keep the cause-plus-next-step line on stderr; structured formats carry the cause as `message`. Every format conveys the cause. Pass.

**P0** | CONSTITUTION II: the next step survives every format
→ **spec.md § Preserving the next step + interface-cli.md/interface-spec.md (`next_step`)**: the next step is surfaced under structured formats as its own parseable `next_step` field, distinct from the cause; absence (internal-error fallback) is an omitted field, not a fabrication. Pass.

**P0** | CONSTITUTION II: machine-parseable failure emission
→ **spec.md § Carrying through the structured failure facts + interface-cli.md § channel split**: under `json`/`yaml` the failure is one 018 envelope on stdout — the same channel an agent parses for success — never bare text interleaved into a structured stream. Pass.

**P0** | CONSTITUTION II: the failure pairs with a machine-readable exit code
→ **spec.md § Pairing with the exit code + interface-cli.md § Error Communication table**: rendering returns `d.Category`; 004 maps it; the exit code is identical across formats and derived from the same `Diagnose` value, so rendering and code cannot disagree. Pass.

**P0** | CONSTITUTION III (Fail Safe): a failure is never silent
→ **spec.md § Behavioral Accord + feature: "A transport failure under json emits the envelope with no status or body"**: every format emits the failure; the structured path never leaves the channel empty. Pass.

**P0** | CONSTITUTION III: no partial/half document on a render that cannot complete
→ **spec.md § edge "a structured render that cannot complete" + plan.md § ADR-4 (buffer-then-write) + feature: "A structured render that cannot complete writes nothing partial"**: nothing partial reaches stdout; the outcome maps to `RuntimeError(1)`. Pass.

**P0** | CONSTITUTION III: a failure is never reported as success
→ **interface-spec.md § Error Communication + interface-cli.md § Error Communication table**: `reportFailure` returns the diagnostic's non-success `Outcome` for every family; the exit code is non-zero. Pass.

**P0** | CONSTITUTION IV (TDD): user-facing behavior has executable acceptance scenarios before code
→ **features/opaque-failures/output-aware-failure-rendering.feature**: 15 `@wip` scenarios (9 spec-derived, 3 validation, 2 architecture-informed) exist before implementation. Pass.

**P0** | CONSTITUTION IV: tasks are test-first and build-verified
→ **tasks.md T001/T002 acceptance criteria**: each task specifies tests (table-driven `kind`, `errorEnvelopeFor` unit tests, token-free assertion, cross-format exit-code regression, BDD un-`@wip`) and `go build` + `go vet` + full `go test ./...` green. Pass.

**P0** | CONSTITUTION V (Composition over Monolith): adding doesn't force unrelated changes
→ **plan.md § ADR-1 + interface-spec.md § Interactions**: 032 extends the single failure chokepoint and adds a pure mapper; the success path, pagination walk, 018 encoders, and 019 templates are untouched. The ~19 call-site edits are a uniform, compiler-enforced signature threading of one shared helper (the endorsed shared-client pattern), not changes to unrelated command logic. Pass.

**P0** | CONSTITUTION VI (Size-Aware, never silently truncate): the partial-walk signal is preserved
→ **plan.md § ADR-3 + spec.md edge + feature: "A mid-walk failure with partial data keeps its incompleteness note on stderr under json"**: under structured formats a mid-walk failure still writes the partial document to stdout, the incompleteness note to stderr, and a non-zero exit — never a silent truncation. Pass.

**P0** | CONSTITUTION VII (Working Software): implementation ships with its tests and builds
→ **tasks.md T001/T002**: acceptance criteria require implementation with its tests and a green build/suite in the same unit. Pass.

**P0** | CONSTITUTION VIII (No Fabricated Data): no invented cause, next step, or body
→ **spec.md § Non-Behaviors (must not fabricate) + plan.md § ADR-4 (body included only when valid JSON) + feature: "An internal-error fallback omits the next step in every format"**: an absent next step is omitted, not invented; the raw body is rendered verbatim or omitted, never fabricated. Pass.

**P0** | CONSTITUTION IX (Writes Require Explicit Intent): rendering is read-only
→ **spec.md § Non-Behaviors + System Overview**: 032 is a pure failure-side renderer — it makes no request and mutates nothing. Pass.

---

## Governance Infrastructure Notes

- **No `accords/governance/` directory** exists in this project, so done-criteria checks (done-plan, done-interface, done-scenarios, done-tasks) and cross-reference checks were not generated. Consider creating `accords/governance/done-*.md` to enable done-criteria quality checks across the pipeline. (Same standing gap noted for 031.)
- **Principles with zero applicable checks for this feature** (noted, not failures):
  - **X. Respect API Limits** — 032 performs no retry/backoff and no `If-Match`/`ETag` write; it renders an already-surfaced `429` diagnostic (the spec Non-Behaviors forbid reintroducing waits). No applicable check.
  - **XI. Governance via Proposals** — 032 exposes no governance mutation path; it is a read-side failure renderer. No applicable check.
  - **XII. Standalone Executable** — 032 adds no external runtime dependency; it reuses `internal/output` (whose `sigs.k8s.io/yaml` was already vendored by 018) and stdlib. Trivially satisfied; no feature-specific check.
