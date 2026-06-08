# Validate: Templated Human Rendering

**Feature**: 019-templated-human-rendering
**Round**: 1 of 3
**Date**: 2026-06-08
**Verdict**: Ready
**Artifacts loaded**: spec.md, plan.md (§ System Overview / Architecture Decisions), tasks.md (5 of 5 tasks complete), interface-spec.md, interface-cli.md, features/unconsumable-output/templated-human-rendering.feature, PROJECT.md
**Implementation files**: `internal/render/` (render.go + 8 embedded templates), `internal/cli/render.go` (the render bridge + `renderFn` seam), and the four rewired reads (`me.go`, `me_roles.go`, `me_actions.go`, `me_projects.go`)

---

## Conformance Summary

| Dimension | Status | Findings |
|---|---|---|
| Driving scenario coverage | ✓ Pass | 0 |
| Acceptance criteria | ✓ Pass | 0 |
| Interface contract conformance | ✓ Pass (1 advisory) | F-1 (advisory) |
| Non-behavior absence | ✓ Pass | 0 |
| @wip lifecycle completion | ✓ Pass | 0 |
| **Validation scenarios** | ✓ Satisfied (3 of 3) | 0 |

**Total**: 5 dimensions checked, 5 passed, 1 advisory finding (non-blocking). All 3 held-out @validation scenarios independently verified.

---

## Driving Scenario Coverage

**Status**: Pass (8 of 8 covered)

Every driving scenario (3 happy + 2 error + 2 edge + 1 plan-proposed) traces to an identifiable code path, and all 8 non-`@validation` scenarios run green in `TestTemplatedHumanRenderingFeatures`.

| Scenario | Status | Implementation |
|---|---|---|
| full preserves the identity projection | ✓ Covered | `internal/render/templates/me.full.tmpl` + `Render` |
| compact renders a list one line per record | ✓ Covered | `roles.compact.tmpl` |
| full enumerates an embedded collection | ✓ Covered | `me.full.tmpl` `{{range .Roles}}` |
| a failed read is not rendered through a template | ✓ Covered | `me_actions.go:runMeActions` — error path routes to `reportClientError`, never `renderResult` |
| a missing field is omitted, never fabricated | ✓ Covered | `me.full.tmpl` `{{if .Roles}}` (empty embed omits section) |
| an empty result set is legible, not blank | ✓ Covered | `projects.full.tmpl` `{{if not .Data}}no projects{{end}}` |
| compact counts a nested collection that full expands | ✓ Covered | `me.compact.tmpl` `roles={{len .Roles}}` |
| render failure leaves stdout empty (plan ADR-4) | ✓ Covered | `internal/cli/render.go:renderResult` (buffer-then-write → RuntimeError) |

---

## Acceptance Criteria

**Status**: Pass (5 of 5 tasks complete, all criteria met)

| Task | Status | Evidence |
|---|---|---|
| T001 — render engine + 8 templates | ✓ | `Render` returns `(string, *RenderError)`, never partial; 4 full goldens; compact one-line-per-record + `roles=<N>`; empty-set explicit lines; registry exhaustiveness (count + per-pair `Lookup`); package imports only stdlib; `go build`/`vet` clean |
| T002 — rewire `me`; delete `formatMe` | ✓ | `runMe` → `renderResult(ResourceMe, …)`; render fail → RuntimeError; `formatMe` + 3 tests removed; identity-read suite green; token-never-in-output test passes |
| T003 — rewire `me roles`; delete `formatMeRoles` | ✓ | `formatMeRoles`/`writeRoleSection` + pure tests removed; me-roles suite green (populated, `No roles.`, incompleteness note); `TestIncomplete` retained |
| T004 — rewire `me actions`; delete `formatMeActions` | ✓ | `formatMeActions` + 3 tests removed; me-actions suite green (populated, `No actions.`, more-available note) |
| T005 — rewire `me projects`; delete `formatMeProjects` | ✓ | `formatMeProjects` + `yesNo` + 4 tests removed; `noRoleMarker` retained (still test-referenced); me-projects suite green (populated, no-role marker, `no projects`, note) |

---

## Interface Contract Conformance

**Status**: Pass (1 advisory)

**interface-spec.md (Go API of `internal/render`)** — conformant:

| Surface element | Status | Evidence |
|---|---|---|
| `Render(resource Resource, format Format, data any) (string, error)` | ✓ | `render.go:Render` — exact signature, pure, buffer-then-return |
| `Resource` / `Format` named-string constants | ✓ | `ResourceMe/Roles/Actions/Projects`, `FormatFull/Compact` — single source of truth, no bare literals at call sites |
| Template set `<resource>.<format>`, `//go:embed`, single parse, `text/template`, `missingkey=error`, pure FuncMap | ✓ | `render.go` — 8 files embedded, parsed once with `Option("missingkey=error")` + `trimSpace`/`join` |
| `*RenderError{Resource, Format, Err}` + `Unwrap`, `errors.As`-discriminable, token-free | ✓ | `render.go:RenderError`; consuming reads map it to `RuntimeError(1)` |
| Registry exhaustiveness guard | ✓ | `TestRegistry_AllBuiltinsResolve` (count + per-pair lookup, PR #10) |

**interface-cli.md (rendered stdout)** — conformant on the surface 019 ships:

| Surface | Status | Evidence |
|---|---|---|
| Four reads render `full` via `render.Render(…, FormatFull, …)` | ✓ | all four `runXxx` |
| `full` field-equivalent to each pre-019 projection | ✓ | independently verified byte-for-byte against `origin/main` formatters (see Validation Scenarios) |
| Empty-line table (`No roles.` / `No actions.` / `no projects`) | ✓ | golden + suite assertions |
| `compact` not operator-selectable | ✓ | no `FormatCompact` reference in non-test `internal/cli` |
| Error communication (render fail → RuntimeError/1; read fail → existing codes) | ✓ | `renderResult` + unchanged `reportClientError` path |

**Advisory F-1** (non-blocking) — `me` compact emits `roles=<N>` unconditionally, whereas interface-cli.md's compact table says "`roles=<N>` present only when `--include roles` was given." See Findings; recorded for 020.

---

## Non-Behavior Absence

**Status**: Pass (6 of 6 excluded behaviors absent)

| Non-behavior | Status | Evidence |
|---|---|---|
| Must not read a flag / choose template / set default | ✓ Absent | `renderResult` hardcodes `FormatFull`; no flag read in the render path |
| Must not expose `compact` through any operator mechanism | ✓ Absent | no `FormatCompact` in non-test `internal/cli` |
| Must not emit JSON/YAML | ✓ Absent | `text/template` only; no `encoding/json`/yaml in `internal/render` |
| Must not load caller-supplied template files | ✓ Absent | only `//go:embed`; no `os.ReadFile`/`ParseFiles`/`ParseGlob` |
| Must not fabricate a data value | ✓ Absent | full == pre-019 output (verified); explicit markers (`—`/`(none)`/`(no purpose set)`) are structural, not data |
| Must not render error output through a template | ✓ Absent | error path routes to `reportClientError`; `renderResult` reached only after a successful `Execute` |
| Must not change which fields a read fetches | ✓ Absent | `apiclient.Request`/`req.Query` construction unchanged vs `origin/main` |

---

## @wip Lifecycle Completion

**Status**: Pass

The 8 scenarios referenced by checked tasks have had their `@wip` tags removed and run green. The 3 remaining `@wip` tags are all on `@validation` scenarios (held out for this skill) — correctly retained. No non-`@validation` scenario carries a stale `@wip`.

---

## Validation Scenario Results

**Status**: Satisfied (3 of 3) — independently verified

These held-out scenarios were verified by a throwaway probe that ran the **actual pre-019 projection functions extracted from `origin/main`** against `render.Render(FormatFull)` across an input matrix (empty lists, blank/whitespace fields, no-role projects, tags present/absent, multi-record, identity-only and empty-roles-embed), rather than trusting the Builder's own goldens.

| Scenario | Status | Trace |
|---|---|---|
| Full is field-equivalent to the pre-feature projection | ✓ Satisfied | render `full` byte-identical to `oldFormatMe`/`oldFormatMeRoles`/`oldFormatMeActions`/`oldFormatMeProjects` (from `origin/main`) across all matrix cases — no field dropped or added |
| No rendered value is absent from the source | ✓ Satisfied | corollary of the byte-equality above (the pre-019 code surfaced only API fields); compact reads the same struct fields; `missingkey=error` backstops typo'd keys |
| Full and compact render the same record set | ✓ Satisfied | five-action probe: both formats account for all five ids; compact is exactly 5 lines (one per record) |

---

## Findings

### F-1 (advisory, non-blocking): `me` compact emits `roles=<N>` unconditionally

- **Dimension**: Interface contract conformance
- **Source**: interface-cli.md § `compact` format > `me` row ("`roles=<N>` present only when `--include roles` was given")
- **Implementation**: `internal/render/templates/me.compact.tmpl` — `roles={{len .Roles}}` rendered on every `me` compact line
- **Gap**: The interface keys the `roles=<N>` segment on whether `--include roles` was passed. `render.Render` is pure over `glassfrog.MeResponse` (per interface-spec.md's signature) and receives no include signal; the API populates `Roles` only on the embed, so an empty `Roles` slice is indistinguishable between "embed not requested" and "embed requested, no roles." A `{{if .Roles}}` guard cannot separate them. The Builder resolved this by always emitting the count — which is consistent with spec.md's own accord ("compact renders the count, e.g. `roles=0`, on the record's line" for an empty embed) and satisfies the only scenario (`roles=3`).
- **Why non-blocking**: `compact` is reachable from **no operator surface** in 019 (deferred to 020), so no consumer can observe the divergence; the behavior conforms to the spec.md behavioral accord (validate's primary authority) and to the CLI surface 019 actually ships (`full` only); and the tension is already logged in `.score/memory/LEARNINGS.md` for 020 to resolve deliberately (pass the include signal into the renderer, or keep the uniform count).
- **Suggested action**: 020 (Output Format Selection), when it makes compact reachable, decides whether to suppress `roles=` absent `--include`. No 019 change required.

---

## Verdict: Ready

All five conformance dimensions pass and all three held-out `@validation` scenarios are independently satisfied — the most load-bearing one (field-equivalence of `full` with the pre-feature projection) verified byte-for-byte against the real `origin/main` code, not the Builder's goldens. The single finding (F-1) is a low-severity, non-blocking interface nuance on a surface 019 deliberately does not expose, is consistent with the spec's behavioral accord, and is already logged for 020. The implementation conforms to its specification.

---

## Next Steps

Implementation conforms to the specification. Suggest PR review and merge. When 020 Output Format Selection makes `compact` operator-reachable, revisit F-1 (the `me` compact `roles=<N>` conditional) — the decision and its constraint are recorded in `.score/memory/LEARNINGS.md`.
