# Checklist: PR Administration

**Feature**: 028-pr-administration
**Checked against**: CONSTITUTION.md (12 principles)
**Artifacts checked**: spec.md, plan.md, interface-spec.md, tasks.md, features/no-automated-pipeline/pr-administration.feature
**Checks**: 5 (5 pass, 0 fail) + 2 P2 considerations
**Generated**: 2026-06-09

---

## Summary

| Severity | Count | Pass | Fail |
|---|---|---|---|
| P0 (blocking) | 5 | 5 | 0 |
| P1 (should fix) | 0 | 0 | 0 |
| P2 (consider) | 2 | — | — |
| **Total** | **5 checks** | **5** | **0** |

---

## Constitution Checks: 5/5 passed

- **P0 | III (Fail Safe, Not Silent)** — *calibrated*: PR Administration's only "write" is labels on a PR (not governance data), and labelling is **non-blocking and self-healing**. spec §Non-Blocking + plan Cross-cutting + interface Error Communication require that a labelling failure never blocks or reddens a merge, and the per-event `concurrency` + re-reconciliation means a transient failure is retried on the next event — there is no partial-apply that misleads a consumer (the workflow is not a required check, so no one treats its status as a governance verdict). Labels outside the managed set are never touched, so a sync never corrupts human triage state. **PASS.** (See P2-1 for the observability nuance of `continue-on-error`.)
- **P0 | IV (Test-Driven Development / BDD)** — the user-facing labelling behaviours carry acceptance scenarios written before implementation: 11 scenarios in `pr-administration.feature` (3 `@validation`, 1 architecture-informed), each `# Source:`-traced to a spec scenario and preceding the workflow they describe; tasks.md T002/T003 reference them as verifying conditions and T003 removes `@wip` as they pass. As with sibling 024, a GitHub-Actions config feature is verified by inspection + a live-PR/fork exercise rather than a godog suite — the scenarios are the executable-style acceptance bar, exercised against a real PR. **PASS.**
- **P0 | V (Composition over Monolith)** — plan ADR-1 keeps PR Administration as an isolated declarative artifact set (`.github/settings.yml` labels block + `.github/labeler.yml` + `.github/workflows/pr-administration.yml`), deliberately **separate** from 024's `ci.yml` so neither widens the other's permissions or gates the other. It adds no Go command module and touches no existing one; the three files are independently legible parts. **PASS.**
- **P0 | VII (Working Software)** — the capability adds **no** Go code and **no** new Go tests; each task leaves a valid state (T001's `settings.yml` is a complete declarative catalog reconciled by the Settings app; T003 leaves a valid workflow). No code-only/test-only split. tasks.md notes the three files may collapse into one PR. **PASS.** (See P2-2: T002 in isolation is inert config.)
- **P0 | XII (Standalone Executable) — no new runtime dependency** — PR Administration introduces the `srvaroa/labeler` action and `gh` as **CI-host** tools only; plan Security Design / Cross-cutting and interface Consistency Notes (the "CONSTITUTION XII note") record that XII governs the produced binary's runtime, not the CI host — the same standing the project gives GoReleaser, golangci-lint, and `sigs.k8s.io/yaml`. It adds nothing to the distributed artifact. **PASS.**

### Principles with no applicable checks for this feature

Calibrated to zero checks — PR Administration adds no CLI command, issues no Glassfrog API call, and performs no governance mutation:

- **I (Spec Fidelity)**, **II (Action Transparency)**, **VIII (No Fabricated Data)** — no command/request surface and no API-sourced output to diff against `spec.yaml`.
- **VI (Size-Aware)**, **X (Respect API Limits)** — no Glassfrog pagination or rate-limited calls (GitHub Actions runs the labeller; the GH rate model is the labeler action's concern, not the CLI's).
- **IX (Writes Require Explicit Intent)**, **XI (Governance via Proposals)** — no governance-structure mutation path; the only mutation is PR labels in GitHub.

## Done-Criteria Checks

Not run — no `accords/governance/done-*.md` accords exist in this repository (see Governance Notes).

## Cross-Reference Checks

Not run — cross-reference checks derive from done-* accords, which are absent. (Informally: tasks.md T001–T003 each carry plan, interface, and scenario references; `pr-administration.feature` carries `# Source:` comments traced to spec scenarios.)

---

## P2 Considerations (advisory — not checks)

- **P2-1 | III observability of persistent labelling failure** — `continue-on-error: true` (interface §`pr-administration.yml`, plan Cross-cutting) is the right call for keeping a flake from blocking merge, but it also means a *persistently* failing labeller (a malformed `labeler.yml`, a removed action tag, a GitHub outage) shows the workflow green and silently stops labelling — and a maintainer relying on the labels for Release Drafting (030) wouldn't be alerted. This is acceptable under the explicit non-blocking design; consider whether a low-noise signal (e.g. not swallowing a *config-parse* failure, which is deterministic and not a flake) would preserve fail-safe legibility without re-introducing a merge block. Tension is real: spec §Non-Blocking vs III's "never hidden."
- **P2-2 | VII inert intermediate task** — T002 (`.github/labeler.yml`) merged on its own is inert until the workflow (T003) exists to run it, so a standalone T002 PR isn't a "working, deliverable unit" in the strict VII sense. Already mitigated by tasks.md's note that the files "may collapse into one PR" — recommend making that explicit by bundling T002+T003 (or all three) so every merged PR is a working increment.

---

## Governance Notes

- **No `accords/governance/` directory** in this repository, so done-criteria and cross-reference checks could not run — constitution checks only. Consider creating `accords/governance/done-{specify,plan,interface,scenarios,tasks}.md` to enable done-criteria quality checks across the pipeline (same note recorded for sibling 024).
- **Calibration**: principles III, IV, VII were calibrated to this CI-infrastructure feature (no Go code, no Glassfrog calls); V and XII applied concretely; the seven CLI-domain principles produced zero applicable checks (listed above) rather than being force-fit.
- The two P2 items are advisory considerations, kept **separate** from the check results per Guardian discipline — neither is a constitution violation.
