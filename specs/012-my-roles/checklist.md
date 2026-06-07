# Checklist: My Roles

**Feature**: 012-my-roles
**Checked against**: CONSTITUTION.md (12 principles)
**Artifacts checked**: spec.md, plan.md, interface-cli.md, features/self-service-reads/my-roles.feature, tasks.md
**Checks**: 14 (12 pass, 2 fail)
**Generated**: 2026-06-07 (round 3 — after conforming to 011)

---

## Summary

| Severity | Count | Pass | Fail |
|---|---|---|---|
| P0 (blocking) | 9 | 9 | 0 |
| P1 (should fix) | 5 | 3 | 2 |
| P2 (consider) | 0 | 0 | 0 |
| **Total** | **14** | **12** | **2** |

---

## Changes Since Previous Run

**Round 1**: 1 P0, 2 P1 (3 failures) → **Round 2**: 0 P0, 2 P1 (P0 resolved) → **Round 3**: 0 P0, 2 P1 (unchanged)

Round 3 re-derives against the artifacts after they were **conformed to Identity Read (011)** (auth fail-safe→2/1 via the shared `classifyClientError`; `roles` subcommand under the runnable `me`; grown `internal/glassfrog.Role`). The conformance did **not** change any check outcome:
- The round-2 P0 fix (II. error next-step) stays resolved — the cause+next-step rule persists in interface-cli § Error Communication and spec § Failure.
- The two P1s (II machine-parseable output; X `429` backoff) are unchanged and remain open by design.
- No previously-passing check regressed; the auth→2/1 mapping mirrors the shipped `auth login` (006), so no new Action-Transparency or Exit-Code finding arises.

---

## Constitution Checks: 12/14 passed

### Failures

**P1** | CONSTITUTION.md II. Action Transparency: "Every action MUST report the spec operation it invoked and the target resource, in machine-parseable form"
→ **interface-cli.md § Surface / spec.md § Output**: The default output is a reshaped, line-oriented projection — structured, but not a formal machine-parseable format, and it does not name the invoked operation (`GET /me/roles`). The formal agent-parseable form (`--output json`) is deliberately deferred to the Unconsumable Output capability. Confirm the line-oriented projection is sufficient for the Now slice, or that this gap is consciously accepted until Unconsumable Output lands. *(Open by design — developer's call.)*

**P1** | CONSTITUTION.md X. Respect API Limits: "backing off on `429` responses"
→ **plan.md ADR-3 / interface-cli.md § Error Communication**: A `429` is surfaced generically as `APIError`→code 3 (now with a "wait before retrying" next step) but with no `Retry-After`/backoff; backoff is deferred to Rate-Limit Handling (017). No retry loop exists (one bounded attempt), so the specific anti-pattern X names is absent. Confirm the deferral to 017 is acceptable for the first read slice. *(Open by design — developer's call.)*

### Passing

**P0** | I. Spec Fidelity → `me roles` maps to `GET /me/roles` (`listMyRoles`, spec.yaml:1003); no invented endpoints, parameters, or behaviors. PASS.

**P0** | II. Action Transparency (error transparency) → every error condition in interface-cli.md § Error Communication now names both a cause and a concrete next step, and never includes the token; the rule is also stated in spec.md § Failure. PASS. *(was P0 fail in round 1)*

**P0** | III. Fail Safe, Not Silent → read-only; every fork fails loud; an empty list is an explicit success. PASS.

**P0** | IV. Test-Driven Development → tasks.md mandates RED-first tests (T003/T004) + a godog suite (T005); acceptance scenarios exist in `my-roles.feature` ahead of code. PASS.

**P0** | V. Composition over Monolith → additive command consuming the shared `apiclient` seam; one `MustRegister` line; no unrelated command edited. PASS.

**P0** | VI. Size-Aware by Design → first-page read clearly signals the boundary on `has_next_page`; never silently truncates. PASS.

**P0** | VIII. No Fabricated Data → projection renders only API-returned fields; `(no purpose set)`/`(none)`/`No roles.` are explicit absence indicators. PASS.

**P0** | IX. Writes Require Explicit Intent → only `GET /me/roles`; no mutation on any path. PASS.

**P0** | XI. Governance via Proposals → no governance-structure mutation surface. PASS.

**P0** | XII. Standalone Executable → no new external/runtime dependency. PASS.

**P1** | VII. Working Software → tasks pair implementation with tests per unit (RED→GREEN). PASS.

---

## Governance Infrastructure Notes

- **No `accords/governance/done-*.md` accords found.** Done-criteria and cross-reference checks were not generated — this run is constitution-only. Consider creating `accords/governance/done-*.md` to enable per-artifact done-criteria checks. (Consistent with prior specs in this project.)
