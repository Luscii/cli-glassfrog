# Checklist: User-Defined Template Output

**Feature**: 035-user-defined-template-output
**Checked against**: CONSTITUTION.md (12 principles). No `accords/governance/done-*.md` found — done-criteria and cross-reference checks not generated.
**Artifacts checked**: spec.md, plan.md, interface-cli.md, interface-spec.md, features/unconsumable-output/user-defined-template-output.feature, tasks.md
**Checks**: 12 (12 pass, 0 fail) + 2 N/A principles
**Generated**: 2026-06-10

---

## Summary

All 12 applicable constitution checks pass. Constitution: 12/12 (principles X and XI produced no applicable checks — see Governance Notes).

This is a pre-implementation checklist: checks evaluate the artifact set (spec / plan / interface / scenarios / tasks), not running code.

---

## Constitution Checks: 12/12 passed

### Failures

None.

### Calibration note — II Action Transparency (NON-NEGOTIABLE)

Principle II was calibrated into three binary assertions for this feature, because a user template makes the *success* output operator-defined:
- **II-a** every template-source failure reports a cause + next step + the offending source → **pass** (interface-cli Error Communication names the file/`stdin` source and the parse/read/execute cause, exit 2).
- **II-b** template selection does not change the spec operation or resource the read invokes → **pass** (spec Non-Behavior "must not change which fields a command fetches"; interface "selection shapes presentation, not the request").
- **II-c** a machine-parseable, traceable output path is always retained → **pass** (`json`/`yaml`/`full` are unchanged and always available; a user template is an explicit operator opt-in to a custom shape). The case where an operator's own template omits ids/operation trace is operator-directed, not CLI fabrication — surfaced as an analyze completeness observation, not a constitution failure.

### Passed (12/12)

- **P0 | I. Spec Fidelity** → spec Non-Behaviors + plan: 035 adds no endpoint, parameter, or API behavior; it renders the result of reads that already map to spec operations. Does not change which fields are fetched.
- **P0 | II. Action Transparency** → interface-cli/spec Error Communication: see calibration note II-a/b/c — all three pass.
- **P0 | III. Fail Safe, Not Silent** → interface (read+parse before any request; buffer-then-write leaves stdout empty on failure; missing file / parse error / empty stdin → reported, no request). Read-only path, no partial governance state.
- **P0 | IV. Test-Driven Development** → user-defined-template-output.feature: 11 acceptance scenarios (`@wip`) exist before implementation; tasks.md acceptance criteria reference them.
- **P0 | V. Composition over Monolith** → plan: extends three existing packages along established seams (`internal/render` engine clone, `internal/output` selection, the `internal/cli` dispatch); `render`/`output` stay non-importing siblings. The shared `resolveFormat→resolveSelection` seam change touches the six reads uniformly (a shared seam, not hidden cross-resource coupling) — breadth noted in plan Risks, not a violation.
- **P0 | VI. Size-Aware by Design** → interface-spec/tasks T004: the pagination incompleteness note is preserved on the user-template path; 035 does not touch fetching or introduce truncation.
- **P0 | VII. Working Software** → tasks.md: every task's acceptance criteria require `go build`/`vet` clean and bundle tests/scenarios with the implementation.
- **P0 | VIII. No Fabricated Data** → spec Non-Behavior + plan ADR-2 + interface: the user template is parsed into a clone that keeps `Option("missingkey=error")`, so a missing key fails loud rather than emitting silent fake data; absence markers are structural, author-written, never invented data values.
- **P0 | IX. Writes Require Explicit Intent** → 035 renders successful *read* results only; it issues no POST/PATCH/DELETE and adds no mutation path.
- **P0 | XII. Standalone Executable** → plan: the engine is Go stdlib `text/template` (already embedded via 019); a template file or piped stdin is runtime operator *input*, not a pre-installed dependency. No new external dependency is introduced.

---

## Governance Notes

- **No `accords/governance/done-*.md` accords found.** Done-criteria and cross-reference checks were not generated. Consider creating `accords/governance/done-specify.md`, `done-plan.md`, `done-interface.md`, `done-scenarios.md`, and `done-tasks.md` to enable vertical quality checks on each artifact (this gap is project-wide, not specific to 035).
- **Principle X (Respect API Limits): no applicable checks.** 035 is a presentation feature — it changes no transport, retry, `If-Match`/`ETag`, or `429` behavior. Its only request-side effect is the fail-fast that *avoids* a doomed request on a bad template (never more requests).
- **Principle XI (Governance via Proposals): no applicable checks.** 035 is read-only; it exposes no governance-structure mutation path.
- **Principle II was calibrated** (see Calibration note). The success-output-traceability tension under a user template is operator-directed and is carried to analyze as a completeness observation, not a constitution failure.
