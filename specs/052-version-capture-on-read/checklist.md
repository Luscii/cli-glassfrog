# Checklist: Version Capture on Read

**Feature**: 052-version-capture-on-read
**Checked against**: CONSTITUTION.md (12 principles)
**Artifacts checked**: spec.md, plan.md, interface-spec.md, features/clobbered-changes/version-capture-on-read.feature, tasks.md
**Checks**: 12 (12 pass, 0 fail)
**Generated**: 2026-06-14

---

## Summary

All 12 checks pass. Constitution: 12/12. (Done-criteria + cross-reference: skipped — no `done-*` accords; see Governance Notes.)

P0: 12/12 passed · P1: 0 · P2: 0.

Two principles pass by **correctly-asserted structural inapplicability** (VI Size-Aware, XI Governance-via-Proposals); one principle (X Respect API Limits) is the principle this feature exists to lay the foundation for — see the observations under Governance Notes.

---

## Constitution Checks: 12/12 passed

### Passed (12/12)

- **C-I — Spec Fidelity (P0)**: Version capture reads the `ETag` response header defined in the v5 spec's "Optimistic Concurrency (ETags)" section (`spec/glassfrog-api-v5.yaml` §50-66, header def §5315). It invents no endpoint, parameter, or behavior — it reads an existing field off a response the CLI already receives (plan ADR-1; interface-spec Surface). No new request shape; `Request` is untouched.

- **C-II — Action Transparency (P0, NON-NEGOTIABLE)**: 052 introduces no new action and no new output — existing reads stay byte-identical (`-o json`/`yaml`/`full`/`compact` unchanged), asserted by the spec non-behavior, plan ADR-1 consequences, and the validation scenario "Adding version capture changes no read contract." Nothing the CLI reports to the operator changes, so the existing traceability is preserved, not reduced. (The version is internal metadata for an in-process consumer, not an action result to report — the deliberate scope choice.)

- **C-III — Fail Safe, Not Silent (P0)**: The accessor introduces no write and swallows no error. It cannot fail (`Header.Get` returns `""` for an absent key); an absent `ETag` is a valid "nothing captured" outcome, not a hidden failure. A *failed* read returns its error and no `*Response`, so the accessor is never reached and existing diagnostics/exit codes (004/015) are untouched (plan Cross-cutting; interface Error Communication table).

- **C-IV — Test-Driven Development (P0)**: T001 ships the accessor together with unit tests covering every contract branch (present→verbatim, absent→`""`, weak-validator preserved, case-insensitive), and the `version-capture-on-read.feature` scenarios are authored now (`@wip`) as executable acceptance before the code that satisfies them (tasks T001; plan Cross-cutting Testing; spec Driving/Validation Scenarios).

- **C-V — Composition over Monolith (P0)**: The change is one additive accessor on the existing `apiclient.Response` type — no stored state, no `Execute` change, no edit to `Request`, the `executor` interface, `RetryExecutor`, or any command (plan ADR-1/ADR-2). It wires no call site, so no unrelated module is touched. Strongly conformant.

- **C-VI — Size-Aware, no silent truncation (P0)**: Inapplicable by structure, correctly asserted. Version capture concerns a single response's header, not result-set paging. Collection reads yield no per-resource version *by construction* — the pagination walker / `aggregateRawData` path has no version seam (plan System Architecture; spec non-behavior; feature "A list read retains no per-resource version"). Nothing is dropped or truncated.

- **C-VII — Working Software (P0)**: T001 pairs implementation with its tests as one reviewable unit (the accessor "must not merge without the tests that pin its contract") — no code-only or test-only increment (tasks T001 Scope/Acceptance).

- **C-VIII — No Fabricated Data (P0)**: The accessor returns only the `ETag` the API actually sent, **verbatim** — no unquoting, no weak-validator stripping, no normalization — and returns `""` (not a placeholder/guessed token) when the header is absent (plan ADR-1; interface-spec; feature "A weak-validator version is captured verbatim"). Strongly conformant — the verbatim/empty-sentinel contract is exactly No-Fabrication applied to a version token.

- **C-IX — Writes Require Explicit Intent (P0)**: 052 is read-side only. It issues no POST/PATCH/DELETE and sends no header; the non-behavior and validation scenario "No If-Match header is sent by version capture" assert nothing this capability introduces sends a precondition. No read path mutates as a side effect. Strongly conformant.

- **C-X — Respect API Limits (P0)**: No violation. X's detection targets "an update request that omits `If-Match` when an `ETag` is available" — 052 issues no updates. It is the *enabling foundation* the conflict-table resolution ("always use optimistic concurrency") depends on: it captures the `ETag` so a later guarded write (053) can send `If-Match`. The accessor sends nothing itself (ADR-2). See observation below on full satisfaction.

- **C-XI — Governance via Proposals (P0)**: Inapplicable, correctly. 052 exposes no command path and mutates no governance structure — it is an internal accessor. No default or opt-in mutation path is introduced. No applicable check beyond confirming the absence of any governance-mutating surface (there is none).

- **C-XII — Standalone Executable (P0)**: The change adds a few derived lines of Go reading an existing `net/http.Header` — no new import, runtime, or external dependency (plan System Architecture: "No new imports"; interface-spec Package). The distributed binary's dependency profile is unchanged.

---

## Governance Notes

- **No `done-*` accords**: `accords/governance/` is absent, so done-criteria and cross-reference checks were skipped (constitution-only run, consistent with siblings 049/050). Consider creating `accords/governance/done-{specify,plan,interface,scenarios,tasks}.md` to enable artifact-quality checks across the pipeline.

- **Principle X is necessary-but-not-sufficient at 052** (observation, not a finding): X is only *fully* satisfied once Guarded Writes (053) lands and the write commands (tension update 044, discard 045) actually send `If-Match`. 052 alone neither satisfies nor violates X for updates — it removes the blocker (no captured version) without changing today's last-write-wins behavior on writes. The standing X exposure on 044/045 is those specs' / 053's scope, unchanged by 052.

- **Unused-until-053 accessor** (observation, not a finding): the exported `Version()` method has no caller until 053. This is a deliberate roadmap foundation (plan Risk 1; doc-comment names 053 as consumer; BACKLOG `053 → requires 052`), not dead code — but a code reviewer may flag it on the implementing PR. It builds clean (Go does not error on unused exported methods).

- **Sibling task-phrasing consistency** (advisory): sibling tasks (e.g. 049 T001–T004) state "RED-first" and "`go build`/`go vet` clean" explicitly in acceptance criteria. T001 here mandates tests-with-implementation and the contract but does not use that exact phrasing. Not a constitutional gap (IV/VII pass), but adding the explicit RED-first ordering and build/vet-clean line would match the established task style.
