# Checklist: Role Domains

**Feature**: 033-role-domains
**Checked against**: CONSTITUTION.md (12 principles)
**Artifacts checked**: spec.md, plan.md, interface-cli.md, features/governance-reads/role-domains.feature, tasks.md
**Checks**: 14 (14 pass, 0 fail)
**Generated**: 2026-06-10

---

## Summary

All 14 checks pass. Constitution: 14/14. (Done-criteria + cross-reference: skipped — no `done-*` accords; see Governance Notes.) One non-blocking observation (nullable `role_id` render) was raised and is now **resolved** (PR #70 review — see Governance Notes).

---

## Constitution Checks: 14/14 passed

### Passed (14/14)

- **C-I.1 — Spec Fidelity, commands map to operations (P0)**: Both commands map to defined v5 operations — `domains <role-id>`→`listRoleDomains` (`spec/glassfrog-api-v5.yaml` `/roles/{id}/domains`), `domain <dom-id>`→`getDomain` (`/domains/{id}`) (plan System Architecture + interface-cli Surface). No invented endpoint.
- **C-I.2 — Spec Fidelity, parameters map to documented params (P0)**: Every flag maps to a documented query/path param: `--query`/`-q`→`q` (the endpoint's documented full-text param), `--include`→`include` with the value set drawn **exactly** from the `getDomain` enum (`{policies}`), `--first-page`/`--per-page`→the shared `Cursor`/`PerPage` pagination params. The grown `Domain` fields (`type`, `role_id`, `created_at`, `updated_at`, `policies`) are exactly the spec's `Domain` schema. No invented parameter or behavior; the `q` "empty/whitespace ignored, malformed→empty" semantics mirror the spec's documented `q` behavior.
- **C-II.1 — Action Transparency, machine-parseable + traceable (P0, NON-NEGOTIABLE)**: Results are machine-parseable via `-o json`/`yaml` (020); the resource is identified by the `dom_…`/`role_…` ids in the projection, traceable to the invoked endpoint. (interface-cli Surface + Interactions.)
- **C-II.2 — Action Transparency, errors name cause + next step (P0, NON-NEGOTIABLE)**: The interface Error Communication table gives every failure a cause **and** a next step, and states the token never appears in any message.
- **C-III.1 — Fail Safe, Not Silent (P0)**: No failure is reported as success — both reads exit non-zero on error; a `domains` mid-walk failure renders the gathered partial flagged incomplete with its cause on stderr, exiting non-zero (interface Interactions + Error Communication; plan ADR-3).
- **C-IV.1 — TDD (P0)**: Every task (T001–T004) specifies RED-first tests before implementation; T004 makes the driving scenarios executable acceptance, and the `role-domains.feature` scenarios exist before the code that satisfies them (the spec-derived scenarios carry `@wip`, removed as they pass).
- **C-V.1 — Composition over Monolith (P0)**: `domains` and `domain` are two new sibling command files; the design composes the landed read stack and adds only `Assemble()` wiring lines; the shared `Domain` is grown **additively** and `Policy` is reused — no unrelated command is edited (plan ADR-1/ADR-2, tasks T002/T003).
- **C-VI.1 — Size-Aware, no silent truncation (P0)**: The `domains` list walks to completion by default, signals on the `--first-page` opt-out, and renders a flagged partial on mid-walk failure (CONSTITUTION VI, plan ADR-3, scenarios). The `q` search composes with the walk (every page carries `q`), so a filtered list is also walked whole. The `domain` single read returns one unpaginated document.
- **C-VII.1 — Working Software (P0)**: Every task pairs implementation with RED-first tests and asserts `go build`/`go vet` clean; no code-only or test-only increments.
- **C-VIII.1 — No Fabricated Data (P0)**: The `domain` render guards its `Policies:` section (omit-when-unrequested, explicit-absence marker when requested-but-empty — 019 pattern), never inventing a value; an empty `domains` list renders `No domains.` rather than a synthesized entry; a null `role_id` renders the `(no controlling role)` explicit-absence marker (pinned in interface-cli.md § Surface, PR #70). (See Governance Notes.)
- **C-IX.1 — Writes Require Explicit Intent (P0)**: Both commands are `GET` reads; the spec Non-Behavior forbids any write/mutation of a domain. No read path issues a POST/PATCH/DELETE.
- **C-X.1 — Respect API Limits, 429 backoff (P0)**: 429 backoff is reused via the landed `RetryExecutor` (017) on the `domains` walk and the single `Execute` (plan Cross-cutting + tasks T002/T003). The `If-Match`/`ETag` clause is N/A — Role Domains performs no updates (see Governance Notes).
- **C-XI.1 — Governance via Proposals (P0)**: Trivially satisfied — Role Domains exposes no governance-mutating path; it is a read-only surface (spec Non-Behavior; plan ADR-1).
- **C-XII.1 — Standalone Executable (P0)**: The plan introduces no new runtime dependency — it composes Go stdlib + landed `internal/*` packages, so the produced binary stays self-contained.

## Done-Criteria Checks

Skipped — no `accords/governance/done-*.md` files exist (see Governance Notes).

## Cross-Reference Checks

Skipped — cross-reference checks derive from `done-*` accords, none of which are present.

---

## Governance Notes

- **`accords/governance/` is empty/absent**: no `done-specify.md`, `done-plan.md`, `done-interface.md`, `done-scenarios.md`, or `done-tasks.md`. Done-criteria and cross-reference checks were skipped. Consider creating these accords to enable artifact-level quality checks beyond the constitution. *(Project-wide infrastructure gap, not specific to 033 — every spec's checklist runs constitution-only, as noted in 026's checklist.)*
- **Nullable `role_id` render (observation — RESOLVED, PR #70)**: the spec's `Domain.role_id` is nullable, and the plan models it `*string` (T001); the original `domain` render examples showed a concrete `Role: role_0456…` without specifying how a **null** controlling role renders. No fabrication risk (a null field renders absent, not invented — C-VIII always passed), but the contract didn't pin the explicit-absence treatment. **Resolved in PR #70 review**: interface-cli.md § Surface now specifies the `(no controlling role)` marker (the repo's `<value | (no X set)>` convention) for both the `full` and `compact` `domain` renders, and T003 acceptance pins it. Mirrors risk H-4 / RC-4.
- **CONSTITUTION X (`If-Match`/`ETag`)**: produced no applicable update-side check — Role Domains is read-only. The 429-backoff half of X is checked (C-X.1).
- **CONSTITUTION XI (Proposals)**: produced only a trivial-satisfaction check — no governance mutation surface exists here.
