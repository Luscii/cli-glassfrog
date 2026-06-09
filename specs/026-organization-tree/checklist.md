# Checklist: Organization Tree

**Feature**: 026-organization-tree
**Checked against**: CONSTITUTION.md (12 principles)
**Artifacts checked**: spec.md, plan.md, interface-cli.md, features/governance-reads/organization-tree.feature, tasks.md
**Checks**: 15 (15 pass, 0 fail)
**Generated**: 2026-06-09 (round 2 — re-run after clarify + propagation)

---

## Summary

All 15 checks pass. Constitution: 15/15. (Done-criteria + cross-reference: skipped — no `done-*` accords; see Governance Notes.)

---

## Changes Since Previous Run

**Previous (round 1)**: 0 P0, 1 P1 fail (1 failure)
**Current (round 2)**: 0 P0, 0 P1 (0 failures)

**Resolved**:
- ~~P1: CONSTITUTION VI — interface-cli.md § Surface: depth-capped node indistinguishable from a leaf~~ → fixed. The clarify session added the depth-boundary behavior to spec.md (Behavioral Accord + Clarifications), and propagation added the render marker to interface-cli.md (`(+ subroles below depth)` / `has_subroles=yes`, risk RC-3), a driving + validation scenario to organization-tree.feature, and a T003 acceptance criterion. The signalling clause is now in the contract.

---

## Constitution Checks: 15/15 passed

### Passed (15/15)

- **C-VI.2 — Size-Aware, clearly signal the boundary (P1)** *(newly resolved)*: A depth-capped node (`has_subroles: true`, empty `children`) now renders a distinct boundary marker (`(+ subroles below depth)` in `full`, `has_subroles=yes` in `compact`) vs. a true leaf — driven by the API's `has_subroles` boolean, with no invented descendant count. Specified in spec.md (Behavioral Accord § Output + Clarifications), interface-cli.md § Surface, organization-tree.feature (driving + validation), and tasks T003 acceptance.

- **C-I.1 / C-I.2 — Spec Fidelity (P0)**: Every command maps to a defined v5 operation — `tree`→`getOrgTree`/`getRoleTree`, `subroles`→`listSubroles` (plan System Architecture + interface, with spec line refs). Flags map to documented query params only: `--depth`→`depth`, `--include` values match the spec enums exactly per read (`{accountabilities,domains,members}` for the tree §655-658; the `getRole` set for subroles §283-289). No invented endpoint, parameter, or behavior.
- **C-II.1 / C-II.2 — Action Transparency (P0, NON-NEGOTIABLE)**: Results are machine-parseable via `-o json`/`yaml` (020); the resource is identified by the `role_…` ids in the projection. The interface Error Communication table gives every failure a cause **and** a next step, and states the token never appears in any message.
- **C-III.1 — Fail Safe, Not Silent (P0)**: No failure is reported as success — tree errors and subroles mid-walk failures exit non-zero; an incomplete subroles set is rendered as a flagged partial with its cause on stderr (interface Interactions + Error Communication).
- **C-IV.1 — TDD (P0)**: Every task (T001–T005) specifies RED-first tests before implementation; T005 makes the driving scenarios executable acceptance, and the `organization-tree.feature` scenarios exist before the code that satisfies them.
- **C-V.1 — Composition over Monolith (P0)**: `tree` and `subroles` are new sibling command files; the `roles` stub is untouched and only `Assemble()` wiring lines are added (plan ADR-1, tasks T003/T004). No unrelated command is edited.
- **C-VI.1 — Size-Aware, no silent truncation (P0)**: The subroles list walks to completion by default, signals on the `--first-page` opt-out, and renders a flagged partial on mid-walk failure (CONSTITUTION VI, plan ADR-3, scenarios). The tree reads return the full unpaginated tree by default (the API is not paged); `--depth` is a user-requested cap, not silent truncation.
- **C-VII.1 — Working Software (P0)**: Every task pairs implementation with RED-first tests and asserts `go build`/`go vet` clean; no code-only or test-only increments.
- **C-VIII.1 — No Fabricated Data (P0)**: The render uses explicit-absence markers and guarded sections, never inventing a value — the interface states a leaf node renders "nothing indented … never a 'no children' line invented as data."
- **C-IX.1 — Writes Require Explicit Intent (P0)**: All three reads are `GET`; the spec Non-Behavior forbids any write/mutation. No read path issues a POST/PATCH/DELETE.
- **C-X.1 — Respect API Limits (P0)**: 429 backoff is reused via the landed `RetryExecutor` (017) on the subroles walk (plan Cross-cutting). The `If-Match`/`ETag` clause is N/A here — Organization Tree performs no updates (see Governance Notes).
- **C-XI.1 — Governance via Proposals (P0)**: Trivially satisfied — Organization Tree exposes no governance-mutating path; it is a read-only surface.
- **C-XII.1 — Standalone Executable (P0)**: The plan introduces no new runtime dependency — it composes Go stdlib + landed `internal/*` packages, so the produced binary stays self-contained.

## Done-Criteria Checks

Skipped — no `accords/governance/done-*.md` files exist (see Governance Notes).

## Cross-Reference Checks

Skipped — cross-reference checks derive from `done-*` accords, none of which are present.

---

## Governance Notes

- **`accords/governance/` is empty/absent**: no `done-specify.md`, `done-plan.md`, `done-interface.md`, `done-scenarios.md`, or `done-tasks.md`. Done-criteria and cross-reference checks were skipped. Consider creating these accords to enable artifact-level quality checks beyond the constitution. *(This is a project-wide infrastructure gap, not specific to 026 — every spec's checklist runs constitution-only.)*
- **CONSTITUTION X (`If-Match`/`ETag`)**: produced no applicable update-side check for this feature — Organization Tree is read-only. The 429-backoff half of X is checked (C-X.1).
- **CONSTITUTION XI (Proposals)**: produced only a trivial-satisfaction check — no governance mutation surface exists here.
