# Checklist: Cross-Model Search

**Feature**: 041-cross-model-search
**Checked against**: CONSTITUTION.md (12 principles)
**Artifacts checked**: spec.md, plan.md, interface-cli.md, features/undiscoverable-governance/cross-model-search.feature, tasks.md
**Checks**: 13 (13 pass, 0 fail)
**Generated**: 2026-06-11

---

## Summary

All 13 checks pass. Constitution: 13/13. (Done-criteria + cross-reference: skipped — no `done-*` accords; see Governance Notes.)

P0: 12/12 passed · P1: 1/1 passed · P2: 0.

---

## Constitution Checks: 13/13 passed

### Passed (13/13)

- **C-I.1 / C-I.2 — Spec Fidelity (P0)**: The single command maps to a defined v5 operation — `search`→`search` (`GET /search`, spec §4156). Flags map to documented query params only: query→`query` (§4165, websearch syntax), `--types`→`types` whose values match the spec enum exactly `{role,note,project,action,skill,actor,policy,domain}` (§4172-4178 / `SearchResult.type` §6135-6143), `--per-page`→`per_page` (§4182, min 1 max 100), the first-page opt-out and walk use the spec's `cursor`/`next_cursor` pagination (§4189, `Pagination` §5405). The query string is forwarded verbatim (no client rewrite). No invented endpoint, parameter, or behavior (plan ADR-1/ADR-3; interface Surface).

- **C-II.1 / C-II.2 — Action Transparency (P0, NON-NEGOTIABLE)**: Results are machine-parseable via `-o json`/`yaml` (020); each result is self-identifying by its `type` + `id` (and owning `role_id` where present), which is the operator's drill-in bridge. The interface Error Communication table gives every failure a cause **and** a next step, and states the token never appears in any message (interface Surface + Error Communication).

- **C-III.1 — Fail Safe, Not Silent (P0)**: No failure is reported as success — a malformed-query `400` and transport failures exit non-zero; a mid-walk failure renders the results gathered so far as a flagged partial with its cause on stderr and exits non-zero; an empty result is a genuine `No results.` success, not a hidden error (interface Interactions + Error Communication; spec Behavioral Accord § Failure/Completeness).

- **C-IV.1 — TDD (P0)**: Every task (T001–T003) specifies RED-first tests before implementation; T003 makes the driving scenarios executable acceptance, and the `cross-model-search.feature` scenarios exist before the code that satisfies them (tasks T001/T002/T003; CONSTITUTION IV).

- **C-V.1 — Composition over Monolith (P0)**: `search` is a new standalone command file (`internal/cli/search.go`); no unrelated command is edited — only `Assemble()` wiring lines are added, and `SearchResult` is a new schema file sharing nothing with existing types (plan System Architecture/ADR-2, tasks T001/T002). Adding `search` forces no change to another command.

- **C-VI.1 — Size-Aware, no silent truncation (P0)**: The result list walks to completion by default via `paging.All[SearchResult]`, signals "more results exist" on the `--first-page` opt-out (exit 0), and renders a flagged partial with its cause on a mid-walk failure (exit non-zero) — the relevance-ranked default was resolved to walk-by-default in clarify (plan ADR-4; spec Clarifications 2026-06-11; scenarios "A multi-page result walks to completion by default", "The first-page opt-out stops at one page and signals more", "A partial result set cannot be read as complete").

- **C-VII.1 — Working Software (P0)**: Every task pairs implementation with RED-first tests and asserts `go build`/`go vet` clean; no code-only or test-only increments (tasks T001–T003 acceptance criteria).

- **C-VIII.1 — No Fabricated Data (P0)**: A null `excerpt` renders as the explicit-absence marker `—` and a null `role_id` omits the `Role:` line — never invented text; the render preserves the API's relevance order exactly and never re-sorts, de-dups, or filters, so no synthesized ordering or value is presented as real (plan ADR-2; interface Surface/Consistency Notes; scenarios "The rendered order matches the API's relevance order"; spec Non-Behaviors).

- **C-IX.1 — Writes Require Explicit Intent (P0)**: `search` is a `GET`-only read; the spec Non-Behavior forbids any write, mutation, or capture from a search hit, and there is no auto-fetch per result. No read path issues a POST/PATCH/DELETE (spec Non-Behaviors; plan What This Plan Does Not Cover).

- **C-X.1 — Respect API Limits (P0)**: `429` backoff is reused via the landed `RetryExecutor` (017) on the page walk (plan Cross-cutting). The `If-Match`/`ETag` clause is N/A here — Cross-Model Search performs no updates (read-only; see Governance Notes).

- **C-XI.1 — Governance via Proposals (P0)**: Trivially satisfied — Cross-Model Search exposes no governance-mutating path; it is a read-only discovery surface (spec Non-Behaviors).

- **C-XII.1 — Standalone Executable (P0)**: The plan introduces no new runtime dependency — it composes Go stdlib + landed `internal/*` packages, so the produced binary stays self-contained (plan System Architecture/Cross-cutting; tasks "all hard dependencies landed").

- **C-II.3 — Action Transparency, structured failure form (P1)**: Failures route through the format-aware chokepoint (032) — structured `json`/`yaml` runs emit the error envelope to stdout, human runs the cause+next-step to stderr — so an agent operator parses success and failure the same way; the incompleteness note is the one deliberate stderr-in-every-format case (a partial `{data:[…]}` already occupies stdout). Reuses the landed `reportFailure`/`classifyClientError` path; adds no new `Outcome`/`ExitCode` (interface Error Communication; plan Cross-cutting).

## Done-Criteria Checks

Skipped — no `accords/governance/done-*.md` files exist (see Governance Notes).

## Cross-Reference Checks

Skipped — cross-reference checks derive from `done-*` accords, none of which are present.

---

## Governance Notes

- **`accords/governance/` is empty/absent**: no `done-specify.md`, `done-plan.md`, `done-interface.md`, `done-scenarios.md`, or `done-tasks.md`. Done-criteria and cross-reference checks were skipped. Consider creating these accords to enable artifact-level quality checks beyond the constitution. *(This is a project-wide infrastructure gap, not specific to 041 — every spec's checklist runs constitution-only.)*
- **CONSTITUTION X (`If-Match`/`ETag`)**: produced no applicable update-side check — Cross-Model Search is read-only. The 429-backoff half of X is checked (C-X.1).
- **CONSTITUTION XI (Proposals)**: produced only a trivial-satisfaction check — no governance mutation surface exists here.
- **No `guardian-agent.md` deployed**: checklist ran from SKILL.md alone (reduced character consistency, not a blocked skill).
