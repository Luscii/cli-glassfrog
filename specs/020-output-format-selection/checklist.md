# Checklist: Output Format Selection

**Feature**: 020-output-format-selection
**Checked against**: CONSTITUTION.md (12 principles)
**Artifacts checked**: spec.md, plan.md, interface-spec.md, interface-cli.md, features/unconsumable-output/output-format-selection.feature, tasks.md
**Checks**: 12 (12 pass, 0 fail)
**Generated**: 2026-06-08

---

## Summary

All 12 checks pass. Constitution: 12/12 (3 principles — IX, X, XI — produced no applicable checks for a read-side, no-API-call selection feature; see Governance Notes).

---

## Constitution Checks: 12/12 passed

### Passed (12/12)

- **P0 | I. Spec Fidelity** → 020 adds no API operation and changes no request — selection is downstream of the read; the spec Non-Behavior "must not change which fields a command fetches" and interface-spec.md (resolution independent of `AssembleFromOS`) confirm the request is untouched. No endpoint/parameter/behavior is invented. **Pass.**
- **P0 | II. Action Transparency (machine-parseable output)** → 020 is the capability that makes the machine-parseable formats (`json`/`yaml`, via 018) selectable from the command line; the four-format selection vocabulary and the dispatch route success to the encoder (interface-cli.md Surface; interface-spec.md dispatch). It strengthens II rather than regressing it. **Pass.**
- **P0 | II. Action Transparency (output traceable)** → selection changes only the rendering, never which fields are present; ids/handles survive because the same result data feeds every renderer (spec Behavioral Accord "Dispatch"; validation scenario "Selection changes rendering only, not the fetched data"). **Pass.**
- **P0 | II. Action Transparency (errors name cause + next step)** → the invalid-selector usage error names the offending value and the supported set ("unsupported --output value 'xml' — supported: full, compact, json, yaml"); a present-but-invalid env/config value names its source; an unreadable `.glassfrogrc` names the file with the `--base-url`-symmetric correction (interface-cli.md Error Communication). **Pass.** *(Coherence note for analyze: under `json`/`yaml`, a command failure keeps cause-plus-next-step plain text on stderr until 032 — the error still explains cause + next step, so II holds; the structured-envelope error path is deferred. See analyze.)*
- **P0 | III. Fail Safe, Not Silent** → an invalid selector at any source fails fast before any request (no silent fall-through to a lower rung); a present-but-invalid value surfaces loudly naming its source; a render error is buffer-then-write → `RuntimeError(1)` with nothing partial on stdout (spec Behavioral Accord "Invalid selector"; plan ADR-4; interface-* Error Communication; scenarios "An unknown selector value fails before any request", "An invalid environment value fails, naming its source"). **Pass.**
- **P0 | IV. Test-Driven Development** → every user-facing behavior has an `@wip` acceptance scenario (output-format-selection.feature: json selection, default full, compact reachable, invalid-selector fail-fast, precedence); tasks are RED-first with testable acceptance criteria (T001/T002 pure-resolver tests, T006 fail-fast + default-byte-equivalence assertions). **Pass.**
- **P0 | V. Composition over Monolith** → `internal/output` (machine) and `internal/render` (human) stay non-importing siblings, bridged by a single generic `cli` dispatch; the `--output` flag is registered once (persistent on root) and the dispatch is shared, so adding a future read reuses it without touching unrelated commands (plan ADR-2/ADR-3; tasks T003/T005). **Pass.**
- **P0 | VI. Size-Aware by Design (no silent truncation)** → 020 renders whatever the read produced and never drops records; pagination and the partial-result incompleteness signal are owned upstream (016) and ride the stderr diagnostic channel, orthogonal to the selected format (interface-cli.md Error Communication). 020 introduces no truncation. **Pass.** *(Low-severity completeness note for analyze: 020's artifacts don't explicitly state how the size-boundary signal appears under `json`/`yaml`.)*
- **P0 | VII. Working Software** → each task ships implementation with its tests and requires `go build`/`go vet` clean (and the reads' suites green for the default `full` path); no code-only/test-only increment (tasks acceptance criteria across T001–T006). **Pass.**
- **P0 | VIII. No Fabricated Data** → 020 routes already-produced result data to a renderer and invents no value; the default `full` is a *format* default (config), not a fabricated *data* value; field fidelity is the renderers' (018 verbatim / 019 markers) concern (spec Non-Behaviors "must not change which fields a command fetches"). **Pass.**
- **P0 | XII. Standalone Executable** → 020 introduces no new runtime or external dependency — the resolver uses `internal/rcfile` + stdlib `os`, and the dispatch uses the existing `internal/output`/`internal/render` packages (plan ADR-1/ADR-3; tasks). **Pass.**
- **P0 | III/II (invalid selector before network)** → the format resolves before connection assembly and the request, so an invalid value never triggers a doomed API call (the `validateInclude` fail-fast shape), keeping the failure both safe and traceable (plan ADR-4; interface-spec.md "resolve before request"; tasks T006 tripwire-transport assertion). **Pass.**

---

## Governance Notes

- **No `accords/governance/done-*.md` accords exist.** Checklist ran constitution-only. Consider creating `done-specify.md`, `done-plan.md`, `done-interface.md`, `done-scenarios.md`, and `done-tasks.md` to enable done-criteria checks for every spec (this affects all specs, not just 020).
- **Guardian agent definition not found** at the expected path — ran the process without the character layer (reduced style consistency, not a blocked check).
- **Principle IX (Writes Require Explicit Intent)**: no applicable checks — output selection is read-side and has no mutation path.
- **Principle X (Respect API Limits)**: no applicable checks — format resolution makes no API call (offline, before the request) and 020 issues no request of its own.
- **Principle XI (Governance via Proposals)**: no applicable checks — 020 performs no governance mutation.
- **Calibration — Principle II ("machine-parseable form")**: for a selection feature, II was read directly — 020 *delivers* the machine-parseable formats (json/yaml) as operator-selectable, and preserves traceability (ids survive because selection changes only rendering). The one open thread is the deferred structured-error path (→ 032), recorded as a coherence note above and carried to analyze.
