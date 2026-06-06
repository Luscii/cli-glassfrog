# Checklist: Connection Context Assembly

**Feature**: 009-connection-context-assembly
**Checked against**: CONSTITUTION.md (done-* accords not found — see Governance Notes)
**Artifacts checked**: spec.md, plan.md, interface-spec.md, features/undefined-connection-settings/connection-context-assembly.feature, tasks.md
**Checks**: 10 (10 pass, 0 fail)
**Generated**: 2026-06-06

---

## Summary

All 10 checks pass. Constitution: 10/10. Done-criteria: not checked (no accords). Cross-reference: not checked (no accords).

Three constitution principles (VI Size-Aware, X Respect API Limits, XI Governance via Proposals) produced no applicable checks for this feature — it registers no command, makes no API call, and mutates no governance. See Governance Notes.

---

## Constitution Checks: 10/10 passed

### Calibration notes

- **II. Action Transparency** — this slice has no command surface and returns a code-free value; "report what it did" is calibrated to "the `ConnectionContext` carries each part's `Source`/`Path` and an aggregate readiness (`Complete()`/`Problems()`) in a structured, machine-readable form", and "errors explain cause" to "`Problems()` and the carried typed errors name the offending part (base-URL source/path, credential path)". The operator-facing *next-step* message and exit code are owned by the consuming command (deferred, code-free split — consistent with 007/008). *Secret-hygiene nuance:* the context holds the token, so transparency must not leak it — the value-receiver redacting `String()` and the secret-free `Problems()` satisfy this; that the token never appears is a cross-artifact invariant (analyze touchpoint), enforced by design here, not a separate constitution check.
- **III. Fail Safe, Not Silent** — calibrated to a read-only aggregator: "validate writes / no partial state" is N/A (no writes); the live concern is "no swallowed error and no problem reported as success." `Assemble` carries both sub-outcomes (no short-circuit), `Complete()` is false whenever any part is missing/errored, and a nil resolver fails loud (panic, no nil-default).
- **VIII. No Fabricated Data** — the slice presents no API-response data; the applicable target is its own spec Non-Behavior "must not fabricate a token" — absence is carried as absence (`Cred.Source == None`), never a synthesized token/URL.

### Passed (10/10)

- **P0** | CONSTITUTION I (Spec Fidelity): invents no endpoints, parameters, or behaviors — **pass**. spec.md Non-Behaviors + plan ADR-1/ADR-2: the slice defines no command and makes no API call; it combines 008's `BaseURL` (already spec-traced — the default URL is 008's, not re-owned here) and 005's `auth.Resolution`. Assembly only — nothing spec-defined is invented.
- **P0** | CONSTITUTION II (Action Transparency): reports what it assembled in machine-parseable form — **pass**. interface-spec.md Output contract `ConnectionContext{BaseURL, BaseURLErr, Cred, CredErr}` + Readiness accessors (`Complete()`, `Problems()`); each part carries its `Source`/`Path`.
- **P0** | CONSTITUTION II (Action Transparency): incomplete/error outcomes name the cause — **pass**. interface-spec.md § Error Communication + spec Behavioral Accord: `Problems()` names each missing/errored part; carried errors name the base-URL source/path and the credential path (path-only). The token never appears (redacting `String()`, secret-free `Problems()`).
- **P0** | CONSTITUTION III (Fail Safe, Not Silent): problems surface loudly, never swallowed or reported as complete — **pass**. spec Behavioral Accord (carry-forward) + plan ADR-1: `Assemble` carries both sub-outcomes without short-circuit; `Complete()` is false on any missing/errored part; a nil resolver panics (fail-fast). No write/partial-state path (read-only assembly).
- **P0** | CONSTITUTION IV (TDD): test-first + executable acceptance — **pass**. tasks T001–T003 each specify RED-first unit tests; features/undefined-connection-settings/connection-context-assembly.feature exists with executable acceptance scenarios (Phase 3 / T003); 4 `@validation` scenarios held out.
- **P0** | CONSTITUTION V (Composition over Monolith): modular, no unrelated edits — **pass**. plan System Architecture + tasks: purely additive (`context.go` in `internal/apiclient`), changes no existing file; consumes 005/008 via injected resolver seams; 007's `AuthTransport` is untouched (the replay wiring is deferred to 010). Adding this capability touches no unrelated command.
- **P0** | CONSTITUTION VII (Working Software): impl + tests + build per increment — **pass**. tasks bundle implementation with RED-first tests per task; acceptance criteria require `go build ./...` and `go vet ./...` clean.
- **P0** | CONSTITUTION VIII (No Fabricated Data): no invented/defaulted values — **pass**. spec Non-Behaviors ("must not fabricate a token") + plan ADR-1: absence is carried as `Cred.Source == None`, never a synthesized token; the base URL is passed through verbatim from 008 (no normalization, no fabricated host).
- **P0** | CONSTITUTION IX (Writes Require Explicit Intent): no mutation — **pass**. spec Non-Behaviors ("must not write, create, or modify any file") + validation scenario "Assembly performs no writes and no network call"; assembly is read-only and makes no API call.
- **P0** | CONSTITUTION XII (Standalone Executable): no new dependency — **pass**. plan/tasks use only the standard library (`sync`, `fmt`, `net/http` for the existing transport contract) and existing internal packages (`internal/auth`, the landed `BaseURL`); no third-party dependency is introduced.

---

## Governance Notes

*(Separate from feature quality findings.)*

- **No `done-*` accords found** (`accords/governance/` is empty). Done-criteria and cross-reference checks were not generated. Consider creating `accords/governance/done-{specify,plan,interface,scenarios,tasks}.md` to enable layered done-criteria checks across the pipeline. (Same gap recorded for 008 — a project-level infrastructure item, not a 009 finding.)
- **Three principles produced no applicable checks** for this feature, each because the capability has no command surface, makes no API call, and mutates no governance:
  - **VI. Size-Aware by Design** — no result sets or pagination (no API call).
  - **X. Respect API Limits** — no API request issued (429/`If-Match` belong to Request Execution, 010).
  - **XI. Governance via Proposals** — no governance-mutating path; no command.
- **Secret-hygiene** (token never emitted) is enforced by design (redacting `String()`, secret-free `Problems()`) rather than a CONSTITUTION principle; it is recorded here as a cross-artifact invariant for analyze to confirm, not a constitution check.
