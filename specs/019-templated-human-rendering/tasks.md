# Tasks: Templated Human Rendering

**Feature**: 019-templated-human-rendering
**Concretization**: Full context (plan + spec + interface + scenarios)
**Inputs**: plan.md, spec.md, interface-spec.md, interface-cli.md, features/unconsumable-output/templated-human-rendering.feature

---

## Dependency Graph

Phase 1: `internal/render` engine + the eight built-in templates (1 task, no phase dependencies) [Shared]
Phase 2: Rewire the four reads to render through the seam with `full` (4 tasks, depend on Phase 1 — parallel with each other) [Shared]

5 tasks total | T001 gated on the open 018↔019 package-home reconciliation (`internal/render` vs `internal/output` — see DECISIONS.md / plan ADR-2); startable once that's resolved. T002–T005 parallel once T001 lands | Builder: pipeline

> Every task is `[Shared]`: the render engine and each read's `full`/`compact` templates serve all three user scenarios (read-as-human / scan-compact / see-everything-in-full) rather than decomposing per scenario.
>
> **Upstream dependency satisfied**: `internal/glassfrog` and all four reads are **landed on main** (STATUS: 011–014 Complete), so the result structs and their current projections exist both to render and to capture as `full` goldens.
>
> **Not purely additive**: T002–T005 each modify an existing read command file and **delete** its `formatXxx` projection (and that projection's pure unit tests, superseded by T001's goldens). The reads' existing godog suites must stay green — that is the no-regression gate for `full`.

---

## Branching Guidance

**Pipeline mode**: `spec/019-templated-human-rendering/base` → `spec/019-templated-human-rendering/task-1`, `…/task-2`, … (one task branch per T-id, merged back into the spec base).

**Parallel-spec awareness**: 019 is the human-rendering half of the Output Formatting cluster. **018 Structured Serialization** is the parallel sibling (JSON/YAML) on its own track — independent of 019. **020 Output Format Selection** and **029 User-Defined Template Output** are *downstream* of 019: 020 will add the `--output` flag that selects a `Format` and remove the hardcoded `FormatFull` at the call sites; 029 will register caller-supplied templates into the same engine. Neither is concurrent with this spec.

---

## Phase 1: `internal/render` engine + the eight built-in templates [Shared]

- [ ] **T001** [Shared] Add the `internal/render` package: the `text/template` engine, `Render` entry point, `Resource`/`Format` keys, `RenderError`, and the eight embedded built-in templates ({me,roles,actions,projects}×{full,compact}) — RED-first goldens + unit tests
  - **Scope**: Create a new package `internal/render` depending only on `internal/glassfrog` + stdlib (it must **not** import `internal/cli` or `internal/apiclient`). Embed eight template files via `//go:embed`, named `<resource>.<format>` for each `Resource × Format` pair; parse them into one `text/template` set (not `html/template`) with `Option("missingkey=error")` and a minimal `FuncMap` (nested-collection count, empty-set detection). Expose `Render(resource Resource, format Format, data any) (string, error)` that executes the named template into a `bytes.Buffer` and returns the string on success or `("", *RenderError{Resource, Format, Err})` on any failure — never partial output. Define the named-string types `Resource` (`ResourceMe`/`ResourceRoles`/`ResourceActions`/`ResourceProjects`) and `Format` (`FormatFull`/`FormatCompact`) as the single source of truth for keys. Author each `full` template **field-equivalent to the landed projection** (`formatMe`/`formatMeRoles`/`formatMeActions`/`formatMeProjects`) and each `compact` template as one line per record (ids always present; nested collections rendered as a count). Empty-result and absent-field rules per interface-cli.md.
  - **Acceptance criteria**:
    - `Render(ResourceMe, FormatFull, meResp)` is byte-equivalent to the pre-019 `formatMe` output, including the roles section only when `--include roles` was given and roles are present, and omitting it otherwise.
    - `Render(<resource>, FormatFull, resp)` is byte-equivalent to the corresponding landed projection for all four resources (four captured goldens).
    - `Render(<resource>, FormatCompact, resp)` renders one line per record with the record's id present; `me` compact shows `roles=<N>` (count) when roles were embedded.
    - An empty list result renders the explicit per-command line — `No roles.` / `No actions.` / `no projects` (wording inherited verbatim from the landed projections); `me` has no empty case.
    - An unknown `resource`/`format`, or a template-execution failure, returns a `*RenderError` and the empty string — never partial text.
    - A registry-exhaustiveness test asserts all eight `<resource>.<format>` templates resolve, with a `len`+comma-ok guard so a dropped/misnamed template fails loud (PR #10 LEARNINGS).
    - The package imports only `internal/glassfrog` + stdlib; `go build ./...` and `go vet ./...` are clean.
  - **Dependencies**: `internal/glassfrog` is landed on main, BUT T001 is **gated** on resolving the open 018↔019 package-home reconciliation (DECISIONS.md / plan ADR-2): 018 (landed) assumed human rendering extends `internal/output`, while this plan creates a separate `internal/render`. The package location must be settled before T001 creates it — otherwise parallel work risks two competing package homes. Once resolved, T001 has no further dependency.
  - **Plan reference**: Phase 1; ADR-1 (`text/template` engine), ADR-2 (`internal/render` package), ADR-3 (`missingkey=error` + FuncMap guards), ADR-4 (buffer-then-return); Cross-cutting (Testing).
  - **Interface references**: interface-spec.md — Surface (`Render`, `Resource`/`Format`, template set, `RenderError`); interface-cli.md — `full`/`compact` shapes and the empty-result table.
  - **Scenario references**: templated-human-rendering.feature: "Full preserves the identity projection", "Full enumerates an embedded collection", "Compact renders one line per role", "Compact counts a nested collection", "Empty result set renders an explicit line", "Full and compact cover the same records", "Full is field-equivalent to the pre-feature projection", "No rendered value is absent from the source"
  - **Risk**: ⚠️ Data-fidelity — `text/template` renders an absent field as a zero value / `<no value>`, which would fabricate output. Enforce with `Option("missingkey=error")` + `{{if}}` guards + goldens; highest-weight test area. ⚠️ `full` goldens must capture the current output exactly — any drift regresses shipped output.

## Phase 2: Rewire the four reads to render through the seam with `full` [Shared]

- [ ] **T002** [Shared] [P] Rewire `me` to render via `render.Render(ResourceMe, FormatFull, …)`; delete `formatMe`
  - **Scope**: In `internal/cli/me.go`, replace the `formatMe(me, includeRoles)` + `Fprint(stdout, …)` call with `render.Render(render.ResourceMe, render.FormatFull, me)`; write the returned string to stdout only on `err == nil`, and on a `*render.RenderError` return `RuntimeError` through the existing `Outcome`→`ExitCode` path. Remove the now-dead `formatMe` function and its pure unit tests (superseded by T001's `me` goldens). `runMe` orchestration, the injected transport seam, and error classification are otherwise unchanged.
  - **Acceptance criteria**:
    - `me` stdout is unchanged from pre-019 for success (including `--include roles` and the empty-roles case) — the identity-read godog suite stays green.
    - A render failure writes nothing to stdout and the command exits with code 1 (RuntimeError).
    - `formatMe` and its dedicated unit tests are removed; the token-never-in-output test still passes; `go build`/`vet` clean.
  - **Dependencies**: T001
  - **Plan reference**: Phase 2
  - **Interface references**: interface-cli.md — `me` full
  - **Scenario references**: templated-human-rendering.feature: "Full preserves the identity projection", "Full enumerates an embedded collection", "Absent embedded collection is omitted", "Render failure leaves stdout empty"

- [ ] **T003** [Shared] [P] Rewire `me roles` to render via `render.Render(ResourceRoles, FormatFull, …)`; delete `formatMeRoles`
  - **Scope**: In `internal/cli/me_roles.go`, replace `formatMeRoles(resp)` + write with `render.Render(render.ResourceRoles, render.FormatFull, resp)` → buffer → stdout, mapping a render error to `RuntimeError(1)`. Remove `formatMeRoles` and its helper `writeRoleSection` and their pure unit tests (superseded by T001's `roles` golden). The pagination incompleteness note path is unchanged.
  - **Acceptance criteria**:
    - `me roles` stdout is unchanged from pre-019 (populated and empty `No roles.` cases) — the `me_roles_bdd_test.go` godog suite stays green.
    - A render failure writes nothing to stdout and exits 1.
    - `formatMeRoles`/`writeRoleSection` and their unit tests are removed (or retained only if still referenced elsewhere); `go build`/`vet` clean.
  - **Dependencies**: T001
  - **Plan reference**: Phase 2
  - **Interface references**: interface-cli.md — `roles` full
  - **Scenario references**: templated-human-rendering.feature: "Compact renders one line per role" (compact verified in T001); "Full is field-equivalent to the pre-feature projection"

- [ ] **T004** [Shared] [P] Rewire `me actions` to render via `render.Render(ResourceActions, FormatFull, …)`; delete `formatMeActions`
  - **Scope**: In `internal/cli/me_actions.go`, replace `formatMeActions(resp)` + write with `render.Render(render.ResourceActions, render.FormatFull, resp)` → buffer → stdout, mapping a render error to `RuntimeError(1)`. Remove `formatMeActions` and its pure unit tests (superseded by T001's `actions` golden).
  - **Acceptance criteria**:
    - `me actions` stdout is unchanged from pre-019 (populated and empty `No actions.` cases) — the `me_actions_bdd_test.go` godog suite stays green.
    - A render failure writes nothing to stdout and exits 1.
    - `formatMeActions` and its unit tests are removed; `go build`/`vet` clean.
  - **Dependencies**: T001
  - **Plan reference**: Phase 2
  - **Interface references**: interface-cli.md — `actions` full
  - **Scenario references**: templated-human-rendering.feature: "Failed read is not rendered through a template", "Full and compact cover the same records"

- [ ] **T005** [Shared] [P] Rewire `me projects` to render via `render.Render(ResourceProjects, FormatFull, …)`; delete `formatMeProjects`
  - **Scope**: In `internal/cli/me_projects.go`, replace `formatMeProjects(resp)` + write with `render.Render(render.ResourceProjects, render.FormatFull, resp)` → buffer → stdout, mapping a render error to `RuntimeError(1)`. Remove `formatMeProjects` and its pure unit tests (superseded by T001's `projects` golden); remove now-unused helpers (`yesNo`, `noRoleMarker`) **only if** no other command still references them.
  - **Acceptance criteria**:
    - `me projects` stdout is unchanged from pre-019 (populated and empty `no projects` cases) — the `me_projects_bdd_test.go` godog suite stays green.
    - A render failure writes nothing to stdout and exits 1.
    - `formatMeProjects` and its unit tests are removed; any helper removed is confirmed unused elsewhere; `go build`/`vet` clean.
  - **Dependencies**: T001
  - **Plan reference**: Phase 2
  - **Interface references**: interface-cli.md — `projects` full
  - **Scenario references**: templated-human-rendering.feature: "Empty result set renders an explicit line"; "Full is field-equivalent to the pre-feature projection"
