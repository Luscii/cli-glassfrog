# Checklist: Credential Discovery

**Feature**: 005-credential-discovery
**Checked against**: CONSTITUTION.md (no `accords/governance/done-*.md` present)
**Artifacts checked**: spec.md, plan.md, interface-spec.md, features/unauthenticated-access/credential-discovery.feature, tasks.md
**Checks**: 8 (8 pass, 0 fail)
**Generated**: 2026-06-03 (re-validated 2026-06-04 after the scenarios migration relocated the feature file to `features/unauthenticated-access/credential-discovery.feature` — findings unchanged)

---

## Summary

All 8 checks pass. Constitution: 8/8. Done-criteria: not run (no accords). Cross-references: not run (no accords).

7 of 12 constitution principles produced applicable checks; 5 are N/A for this internal, read-only, no-network credential resolver (see Governance Notes).

---

## Constitution Checks: 8/8 passed

### Failures

None.

### Passed (8/8)

**P0** | CONSTITUTION.md III (Fail Safe, Not Silent): "Errors MUST be obvious and recoverable, never hidden" / anti-pattern "swallowing errors … a failure condition reported as success"
→ **spec.md § Behavioral Accord > Error handling**, **interface-spec.md § Error Communication**, **features/unauthenticated-access/credential-discovery.feature**: an unreadable or unparseable `.glassfrogrc` returns a typed error naming the path — never a silent fall-through and never reported as "no credentials found". Pinned by scenarios "An unreadable credentials file fails loudly" and "A malformed credentials file fails loudly".

**P0** | CONSTITUTION.md IV (TDD): "Features MUST be built test-first (RED → GREEN)"
→ **tasks.md § T001, T002**: both mandate RED-first unit tests (reader: valid/tokenless/comments/malformed; resolver: each precedence rung + edge cases over temp dirs) before implementation.

**P0** | CONSTITUTION.md IV: "user-facing behavior MUST have an executable acceptance scenario before the code"
→ **features/unauthenticated-access/credential-discovery.feature**, **tasks.md § T003**: 10 behavioral `@wip` scenarios (env override / empty-env / nearest-wins / walk-up / home-on-path / home-fallback / tokenless-skip / no-credentials / unreadable / malformed) exist before implementation; tasks.md § T003 turns them into executable acceptance.

**P0** | CONSTITUTION.md V (Composition over Monolith): "modular, independently-testable parts … adding … MUST NOT require changing unrelated ones"
→ **plan.md § ADR-1 / System Architecture**: resolution lives in a new `internal/auth` package, separate from `internal/cli`; it registers no command and edits no existing command module. The reader/resolver are exercised in isolation via injected roots (ADR-5).

**P0** | CONSTITUTION.md VII (Working Software): implementation paired with tests; "validate and build"
→ **tasks.md § T001, T002, T003**: each task pairs implementation with tests; `go build ./...` and `go vet ./...` clean are explicit acceptance criteria.

**P0** | CONSTITUTION.md VIII (No Fabricated Data): "MUST NOT invent, guess, or fill placeholder values" (applied to the resolved credential value)
→ **spec.md § Behavioral Accord > Absence**, **non-behavior** ("must not … invent a token"), **scenario "No credentials anywhere is reported as absence"**: when no source yields a usable token, Discovery reports `Source: None` and does not fabricate a token.

**P0** | CONSTITUTION.md IX (Writes Require Explicit Intent): "A read-shaped command … MUST NEVER mutate as a side effect"
→ **spec.md § Non-Behaviors** ("must not write, create, or modify any credentials file"), **interface-spec.md § Consistency Notes**, **validation scenario "Resolution never writes to the filesystem"**: Discovery is strictly a reader; writing is Credential Storage's (006) concern.

**P0** | CONSTITUTION.md XII (Standalone Executable): "no language runtime … no other software … that must be installed first"
→ **plan.md § ADR-3 / Cross-cutting Concerns**: the `.glassfrogrc` reader is hand-rolled (no INI/dotenv dependency); resolution uses only the Go standard library (`os`). The artifact remains a single self-contained binary.

---

## Governance Notes

- **No `done-*` accords found.** Constitution checks ran; done-criteria and cross-reference checks did not. Consider creating, to enable vertical quality checks for later specs:
  - `accords/governance/done-specify.md`, `done-plan.md`, `done-interface.md`, `done-scenarios.md`, `done-tasks.md`
- **Principle I (Spec Fidelity)**: N/A — Credential Discovery invokes no Glassfrog API operation; it resolves a local credential. Spec-operation binding is deferred to endpoint-command specs (and to Request Authentication 007, which attaches the token to API calls).
- **Principle II (Action Transparency)**: N/A for the operator-facing clause — Discovery performs no API-backed action and targets no record, and surfaces nothing to the operator directly (Request Authentication 007 composes the "acting as" line and the error's next step). The source-reporting (`Source`/`Path`, never the token) and secret-hygiene properties align with II's machine-legibility spirit — see Advisory Notes.
- **Principle VI (Size-Aware)**: N/A — no API result sets or pagination. Nearest-wins selects one file; it is not truncation of a result set.
- **Principle X (Respect API Limits)**: N/A — no network calls.
- **Principle XI (Governance via Proposals)**: N/A — no governance-mutating command path.

## Advisory Notes (not severity findings)

These passed (or are out of scope) but are worth the developer's attention:

- **II — secret hygiene has no dedicated principle**: "the token value never appears in output, logs, or error messages" (spec non-behavior; interface Error Communication) is a genuine security property but does not trace cleanly to any single constitution principle, so it is not raised as a severity check. It is pinned by the validation scenario "The token value never appears in output" — recommend the reviewer confirm error strings carry only the path.
- **Shared `[ASSUMED]` contract with Credential Storage (006)**: the `.glassfrogrc` name/format and `GLASSFROG_TOKEN` are provisional and shared with an unspecified sibling. This is a cross-artifact/coordination concern (analyze's domain and the shape handoff), not a constitution check — flagged so it is reconciled before 006 and 005 both ship.
