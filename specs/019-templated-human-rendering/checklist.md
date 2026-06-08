# Checklist: Templated Human Rendering

**Feature**: 019-templated-human-rendering
**Checked against**: CONSTITUTION.md (12 principles)
**Artifacts checked**: spec.md, plan.md, interface-spec.md, interface-cli.md, features/unconsumable-output/templated-human-rendering.feature, tasks.md
**Checks**: 13 (13 pass, 0 fail)
**Generated**: 2026-06-08

---

## Summary

All 13 checks pass. Constitution: 13/13 (2 principles — X, XI — produced no applicable checks for a read-side rendering feature; see Governance Notes).

---

## Constitution Checks: 13/13 passed

### Passed (13/13)

- **P0 | I. Spec Fidelity** → `internal/render` introduces no request surface — it renders already-decoded result structs and calls no API operation (interface-spec.md: "no command and no entry point", library only). No endpoint/parameter is invented. **Pass.**
- **P0 | II. Action Transparency (output traceable)** → both `full` and `compact` always surface each record's id — "ids always present" / "The id values are always surfaced" (spec Behavioral Accord; interface-cli.md shapes). Output stays traceable to a resource + id. **Pass.**
- **P0 | II. Action Transparency (calibrated — machine-parseable form)** → see calibration note. `full` preserves the landed reads' projection verbatim (field-equivalent), and those Complete reads satisfy II via ids-as-handles; structured JSON/YAML is the 018 sibling, not regressed here. **Pass.**
- **P0 | II. Action Transparency (errors)** → error output is explicitly **not** templated; it keeps its existing cause-plus-next-step form on stderr (spec Non-Behavior; interface-cli.md Error Communication). **Pass.**
- **P0 | III. Fail Safe, Not Silent** → render is buffer-then-write: on a render error nothing reaches stdout and the command exits `RuntimeError(1)`; no error is swallowed and no partial output is emitted (plan ADR-4; interface-spec.md; interface-cli.md error table; feature scenario "Render failure leaves stdout empty"). **Pass.**
- **P0 | IV. Test-Driven Development** → tasks are RED-first: T001 acceptance criteria are goldens + unit tests + a registry exhaustiveness guard; T002–T005 keep the landed godog suites green and remove superseded unit tests; user-facing behavior has `@wip` acceptance scenarios in the feature file. **Pass.**
- **P0 | V. Composition over Monolith** → rendering is a new modular `internal/render` package with per-resource templates; adding a future read adds templates without touching unrelated commands (plan ADR-2; the four read rewires are independent, parallelizable tasks). **Pass.**
- **P0 | VI. Size-Aware by Design (no silent truncation)** → render emits every record in the result set (`full` enumerates; `compact` is one line per record) and renders an explicit per-command empty line for zero records — it never drops records. Pagination/truncation is upstream (016), not render's concern. **Pass.**
- **P0 | VII. Working Software** → each task ships implementation with its tests and requires `go build`/`go vet` clean and the BDD suites green (tasks acceptance criteria). No code-only/test-only increment. **Pass.**
- **P0 | VIII. No Fabricated Data** → `Option("missingkey=error")` + `{{if}}` guards + no literal data defaults; the clarified rule forbids inventing a *data value* the API didn't return while permitting explicit emptiness markers; pinned by the "No rendered value is absent from the source" validation scenario and the `full` goldens (spec Behavioral Accord + Non-Behaviors after the 2026-06-08 clarification; plan ADR-3). **Pass.**
- **P0 | IX. Writes Require Explicit Intent** → render is a pure data→string function on read results; it issues no POST/PATCH/DELETE and has no mutation path (interface-spec.md; plan). **Pass.**
- **P0 | XII. Standalone Executable** → the engine is Go stdlib `text/template` and templates are bundled via `//go:embed`; no new runtime or external dependency is introduced (plan ADR-1). **Pass.**
- **P0 | III/VIII (registry integrity)** → a `len`+comma-ok registry-exhaustiveness test asserts all eight `<resource>.<format>` templates resolve, so a dropped/misnamed template fails loud rather than rendering wrong/empty (plan Cross-cutting; interface-spec.md; tasks T001). **Pass.**

---

## Governance Notes

- **No `accords/governance/done-*.md` accords exist.** Checklist ran constitution-only. Consider creating `done-specify.md`, `done-plan.md`, `done-interface.md`, `done-scenarios.md`, and `done-tasks.md` to enable done-criteria checks for every spec (this affects all specs, not just 019).
- **Guardian agent definition not found** at the expected path — ran the process without the character layer (reduced style consistency, not a blocked check).
- **Principle X (Respect API Limits)**: no applicable checks — `internal/render` makes no API calls (no `429`/`If-Match` surface).
- **Principle XI (Governance via Proposals)**: no applicable checks — render performs no governance mutation.
- **Calibration — Principle II ("machine-parseable form")**: for a human-rendering feature, II was read as "output remains traceable to a resource + id." `full`/`compact` both surface ids unconditionally, and `full` is field-equivalent to the landed reads (Complete, already conformant); structured JSON/YAML output is the 018 Structured Serialization sibling. 019 preserves traceability and does not regress II.
