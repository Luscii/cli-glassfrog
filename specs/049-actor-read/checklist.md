# Checklist: Actor Read

**Feature**: 049-actor-read
**Checked against**: CONSTITUTION.md (12 principles)
**Artifacts checked**: spec.md, plan.md, interface-cli.md, features/actors-disconnected-from-governance/actor-read.feature, tasks.md
**Checks**: 13 (13 pass, 0 fail)
**Generated**: 2026-06-13

---

## Summary

All 13 checks pass. Constitution: 13/13. (Done-criteria + cross-reference: skipped — no `done-*` accords; see Governance Notes.)

P0: 12/12 passed · P1: 1/1 passed · P2: 0.

---

## Constitution Checks: 13/13 passed

### Passed (13/13)

- **C-I.1 / C-I.2 — Spec Fidelity (P0)**: The single read maps to a defined v5 operation — `actors <id>`→`getActor` (`GET /actors/{id}`, spec §855), reading the `Actor` schema (§6226). The `--include` flag maps to the documented `include` query param, validated locally against the spec's `getActor` enum exactly `{roles, assignments}` (spec.md Behavioral Accord § Related resources; plan ADR-3; interface `--include` row). The actor `id` is passed through verbatim on the path (no client rewrite; a malformed/unknown id is the API's clean `404` — plan ADR-3, 025 ADR-4). No invented endpoint, parameter, or behavior — an `agt_` read goes through the ungated unified `/actors/{id}`, never the deferred `ai_integration`-gated `/agents/{id}` alias (spec Non-Behavior; plan Cross-cutting/Permission scoping). `getActor`'s `PATCH`/`DELETE` siblings are excluded by Non-Behavior.

- **C-II.1 / C-II.2 — Action Transparency (P0, NON-NEGOTIABLE)**: Results are machine-parseable via `-o json`/`yaml` (020), emitting the `{data: …}` document; the actor is self-identifying by its `per_`/`agt_` `id` + `kind` badge. The interface Error Communication table gives every failure a cause **and** a next step (e.g. not-authenticated → "run `glassfrog auth login` or set GLASSFROG_TOKEN"; `404`/non-2xx → names the HTTP status + extracted detail + per-class next step), and states the token never appears in any message (interface Error Communication; spec Behavioral Accord § Failure: "names both what went wrong and a concrete next step … never includes the token").

- **C-III.1 — Fail Safe, Not Silent (P0)**: No failure is reported as success — a `404` for an unknown id and transport failures exit non-zero (interface Error Communication rows 3/6; spec § Failure). An unsupported `--include` value is a fail-fast `UsageError(2)` *before any request*, never silently dropped (the silent-wrong-results hazard plan ADR-3 validates against locally). The mode-separation guards (list filter with an id; `--include` with no id) likewise fail fast with no request. No swallowed errors; no partial-state risk (read-only).

- **C-IV.1 — TDD (P0)**: Every task (T001–T004) specifies RED-first tests before implementation — T001 unit-tests the `ActorDetail` decode, T002 golden + registry-guard tests, T003 "RED-first unit tests for every branch", and T004 makes the driving scenarios executable acceptance. The `actor-read.feature` scenarios exist before the code that satisfies them (tasks T001–T004; CONSTITUTION IV).

- **C-V.1 — Composition over Monolith (P0)**: Actor Read composes the landed read stack and adds only bounded, additive surface — a new `ActorDetail` type (`internal/glassfrog`, reusing the unchanged `Actor`/`Role`/`Assignment`), one new singular `actor` render key (`internal/render`), and a per-read `--include` validator. The one shared touch — *growing* 048's `actors` command from `cobra.NoArgs` to `cobra.MaximumNArgs(1)` — is an announced, bounded growth of an already-landed command (048 merged as #90; plan ADR-1); the 0-arg directory branch (048 `runActorsList`) is preserved verbatim, and no *unrelated* command is edited. If the `include.go` validator is parameterized, tasks T003 mandates preserving its existing message for current callers (no sibling regresses).

- **C-VI.1 — Size-Aware, no silent truncation (P0)**: `GET /actors/{id}` is a single resource; the `roles`/`assignments` embeds arrive inline in the one response, so there is no page walk to perform and nothing to truncate. The plan, interface, and a held-out validation scenario assert exactly one request is issued and no pagination cursor is followed — even when the embedded arrays are large (plan Cross-cutting "No pagination"; interface "Single request, no walk"; spec Validation Scenario "The single read issues no page walk"; feature "A single read issues exactly one request and no page walk"). `Page[T]`/`paging.All` are not instantiated; `--first-page`/`--per-page` do not apply. Conformance to the principle is by structural inapplicability, correctly asserted rather than assumed.

- **C-VII.1 — Working Software (P0)**: Every task pairs implementation with RED-first tests and asserts `go build`/`go vet` clean; no code-only or test-only increments (tasks T001–T004 acceptance criteria, each ending "`go build`/`go vet` clean" or equivalent).

- **C-VIII.1 — No Fabricated Data (P0)**: The render surfaces only data the API returns — each embed section (`Roles:`/`Assignments:`) and each nullable field (purpose/accountabilities/domains) is behind 019's explicit-absence guard (`(no purpose set)`/`(none)`), rendered **only when `?include`d**, never inventing a value (plan ADR-4; interface Output "guarded by an explicit-absence marker (019), never inventing a value"; tasks T002 Risk "never invent a value (019)"). An embedded role with no purpose renders the absence marker, not a blank.

- **C-IX.1 — Writes Require Explicit Intent (P0)**: Actor Read is a `GET`-only read; the spec Non-Behavior forbids creating, updating, or deleting the actor (the endpoint's `PATCH`/`DELETE` are out of scope). No read path issues a POST/PATCH/DELETE (spec Non-Behaviors; plan What This Plan Does Not Cover § Actor administration).

- **C-X.1 — Respect API Limits (P0)**: `429` backoff is reused via the landed `RetryExecutor`/shared classifier (017/015) on the single `Execute` (plan Cross-cutting/Error handling: "`429` → `RateLimited(5)` (017 retries, 015 classifies)"). The `If-Match`/`ETag` clause is N/A — Actor Read performs no updates (read-only; see Governance Notes).

- **C-XI.1 — Governance via Proposals (P0)**: Trivially satisfied — Actor Read exposes no governance-mutating path; it is a read-only drill-in (spec Non-Behaviors).

- **C-XII.1 — Standalone Executable (P0)**: The plan introduces no new runtime dependency — it composes Go stdlib + landed `internal/*` packages, so the produced binary stays self-contained (plan Implementation Strategy "Hard dependencies are all landed"; tasks T001–T004 import only `internal/glassfrog`/`internal/render`/`internal/cli` + stdlib, never `cli`/`apiclient` cross-imports where forbidden).

- **C-II.3 — Action Transparency, structured failure form (P1)**: Failures route through the format-aware chokepoint (032, `reportFailure`) — structured `json`/`yaml` failures emit the 018 unified error envelope to stdout, human (`full`/`compact`) failures write the cause+next-step to stderr — so an agent operator parses success and failure the same way. Reuses the landed `reportFailure`/`classifyClientError` path; adds no new `Outcome`/`ExitCode` (interface Error Communication "049 reuses `reportFailure` unchanged"; plan Cross-cutting/Error handling).

## Done-Criteria Checks

Skipped — no `accords/governance/done-*.md` files exist (see Governance Notes).

## Cross-Reference Checks

Skipped — cross-reference checks derive from `done-*` accords, none of which are present.

---

## Governance Notes

- **`accords/governance/` is empty/absent**: no `done-specify.md`, `done-plan.md`, `done-interface.md`, `done-scenarios.md`, or `done-tasks.md` (the `accords/` directory itself does not exist). Done-criteria and cross-reference checks were skipped. Consider creating these accords to enable artifact-level quality checks beyond the constitution. *(This is a project-wide infrastructure gap, not specific to 049 — every spec's checklist, including sibling 048's, runs constitution-only.)*
- **CONSTITUTION X (`If-Match`/`ETag`)**: produced no applicable update-side check — Actor Read is read-only. The 429-backoff half of X is checked (C-X.1).
- **CONSTITUTION XI (Proposals)**: produced only a trivial-satisfaction check — no governance mutation surface exists here.
- **CONSTITUTION VI (Size-Aware)**: applies by structural inapplicability — `GET /actors/{id}` is a single resource with inline embeds, so there is no page walk. The check (C-VI.1) confirms the artifacts correctly assert single-request / no-walk rather than silently assuming it; this is the inverse of 048's walk-to-completion obligation.
- **Spec line numbers** (`getActor` §855, `Actor` §6226) are a navigation hint against the current `spec/glassfrog-api-v5.yaml` revision — confirm by `operationId` (`getActor`) rather than line, per the FEATURE-MODEL convention.
