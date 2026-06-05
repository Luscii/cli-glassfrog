# Checklist: Base URL Resolution

**Feature**: 008-base-url-resolution
**Checked against**: CONSTITUTION.md (done-* accords not found — see Governance Notes)
**Artifacts checked**: spec.md, plan.md, interface-spec.md, features/undefined-connection-settings/base-url-resolution.feature, tasks.md
**Checks**: 9 (9 pass, 0 fail)
**Generated**: 2026-06-04

---

## Summary

All 9 checks pass. Constitution: 9/9. Done-criteria: not checked (no accords). Cross-reference: not checked (no accords).

Four constitution principles (VI Size-Aware, VIII No Fabricated Data, X Respect API Limits, XI Governance via Proposals) produced no applicable checks for this feature — it registers no command and makes no API call. See Governance Notes.

---

## Constitution Checks: 9/9 passed

### Calibration notes

- **II. Action Transparency** — this slice has no command surface and returns a code-free outcome; "report what it did" is calibrated to "reports the resolved `Source` (and `Path`)", and "errors explain cause" to "typed error names the offending source". The operator-facing *next-step* message is owned by the consuming command (deferred, consistent with 007's code-free split) — not in scope for this slice.
- **I. Spec Fidelity** — calibrated to the one API-derived value this slice owns: the built-in default base URL must trace to the spec, since the slice defines no commands/parameters.
- **VIII. No Fabricated Data** — the slice presents no API-response data, so the principle's literal target (invented response fields) does not apply; the *derived default host* is recorded as a Spec-Fidelity nuance below, not a fabrication finding.

### Passed (9/9)

- **P0** | CONSTITUTION I (Spec Fidelity): built-in default base URL traces to the spec — **pass**. plan.md § Integration Design and interface-spec.md (default row) adopt `https://glassfrog.com/api/v5` — the `/api/v5` path from `spec/glassfrog-api-v5.yaml`'s `servers` block, the host *inferred* from `info.contact.url` (not a normative OpenAPI base). The slice invents no endpoints/parameters (resolution only). *Nuance:* the host is inferred (relative server URL), flagged in plan Risks / risk H-1 — not a violation.
- **P0** | CONSTITUTION II (Action Transparency): reports the resolved source — **pass**. interface-spec.md `BaseURL{Value, Source, Path}` with a `Source` enum (`Flag`/`Environment`/`File`/`Default`) and `Path` for the file case.
- **P0** | CONSTITUTION II (Action Transparency): error outcomes name the cause — **pass**. interface-spec.md § Error Communication: a malformed value yields a typed `BaseURLError` naming the source; unreadable/unparseable files reuse 005's typed errors naming the path.
- **P0** | CONSTITUTION III (Fail Safe, Not Silent): broken inputs fail loud, never silently — **pass**. spec.md Behavioral Accord + plan ADR-4: malformed value and unreadable/unparseable file surface typed errors with no silent fall-through to a lower source or the default. (Write-validation/partial-state aspect N/A — read-only resolution.)
- **P0** | CONSTITUTION IV (TDD): test-first + executable acceptance — **pass**. tasks T001–T003 each specify RED-first unit tests; features/undefined-connection-settings/base-url-resolution.feature exists with executable acceptance scenarios (Phase 3 / T003).
- **P0** | CONSTITUTION V (Composition over Monolith): modular, no unrelated edits — **pass**. plan ADR-1 (resolution in `internal/apiclient`) and ADR-3 (reuse 005's one shared `.glassfrogrc` parser, no second reader) keep concerns modular; no command is added. *Coupling note:* T001 generalizes the shared `parseCredentials` (used by 005/006/007) — a deliberate extension with the token path preserved by test, not a hidden dependency (an analyze/risk touchpoint).
- **P0** | CONSTITUTION VII (Working Software): impl + tests + build per increment — **pass**. tasks bundle implementation with RED-first tests per task; acceptance criteria require `go build ./...` and `go vet ./...` clean.
- **P0** | CONSTITUTION IX (Writes Require Explicit Intent): no mutation — **pass**. spec.md Non-Behaviors ("must not write/create/modify any config file") + validation scenario "Resolution never writes to the filesystem"; resolution is read-only.
- **P0** | CONSTITUTION XII (Standalone Executable): no new dependency — **pass**. plan ADR-3 reuses the hand-rolled stdlib parser (no INI/dotenv lib); ADR-4's URL validation uses stdlib `net/url`. No third-party dependency introduced.

---

## Done-Criteria Checks: not checked

No `accords/governance/done-*.md` accords exist in the project — done-criteria and cross-reference checks could not be generated. See Governance Notes.

---

## Governance Notes

- **No `accords/` directory**: done-criteria accords are absent, so checklist ran constitution checks only. Consider creating `accords/governance/done-specify.md`, `done-plan.md`, `done-interface.md`, `done-scenarios.md`, and `done-tasks.md` to enable vertical done-criteria checks and cross-reference (link-presence) checks for every spec. (This gap is project-wide, not specific to 008.)
- **Principle VI (Size-Aware by Design)**: no applicable checks — Base URL Resolution handles no result sets or org tree.
- **Principle VIII (No Fabricated Data)**: no applicable checks — the slice presents no API-response data. (Derived default host noted under Spec Fidelity.)
- **Principle X (Respect API Limits)**: no applicable checks — no API call in this slice (429/If-Match belong to the request/response path).
- **Principle XI (Governance via Proposals)**: no applicable checks — no governance-structure mutation.
