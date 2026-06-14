# Checklist: Subrole Filler Roll-up

**Feature**: 051-subrole-filler-roll-up
**Checked against**: CONSTITUTION.md (12 principles)
**Artifacts checked**: spec.md, plan.md, interface-cli.md, tasks.md, features/who-to-contact-for-a-role/subrole-filler-roll-up.feature
**Checks**: 14 (14 pass, 0 fail)
**Generated**: 2026-06-14

---

## Summary

All 14 checks pass. Constitution: 14/14 (one principle — XI Governance via Proposals — produced no applicable checks for this read-only feature; see Governance Notes).

No done-criteria or cross-reference checks were generated — no `accords/governance/done-*.md` accords are deployed (see Governance Notes).

---

## Constitution Checks: 14/14 passed

### Failures

None.

### Passed (14/14)

- **P0 | I. Spec Fidelity** → spec.md, plan.md ADR-1/ADR-2, interface-cli.md: the command maps to a real spec operation — `GET /roles/{id}/subroles/actors` (`listSubrolesActors`), verified present in `spec/glassfrog-api-v5.yaml:319`. The `kind` enum (`human`/`agent`), `per_page`, and cursor are spec-defined parameters; `--first-page`/`--per-page` are client controls over those params. Only `--kind` is surfaced — `--role-id`/`--query` are deliberately omitted because the endpoint accepts neither (no invented parameter). The "one level, not transitive" roll-up matches the endpoint's documented semantics ("One level of nesting").
- **P0 | II. Action Transparency** → interface-cli.md § Output/Error Communication: success renders machine-parseable structured output (json/yaml) traceable to the endpoint; the full exit-code table maps every condition to a cause + next step; the token never appears. Errors name what went wrong and the next step.
- **P0 | III. Fail Safe, Not Silent** → spec.md § Failure, interface-cli.md, plan.md ADR-3: no swallowed errors — the leaf-anchor `404` is surfaced (not hidden as an empty success), a mid-walk failure is flagged incomplete with a non-zero exit, and an unsupported `--kind` fails fast. Write-validation/partial-state clauses are N/A (read-only).
- **P0 | IV. Test-Driven Development** → tasks.md T001 ("RED-first unit tests for every branch"), T002 (BDD suite), the `.feature` file authored before code with `@wip` scenarios held for the implementing agent. User-facing behavior has executable acceptance scenarios.
- **P0 | V. Composition over Monolith** → plan.md ADR-1, tasks.md Branching Guidance: adds a **distinct** read leaf and reuses shared seams (`Actor` model, `actors` render, `validateKind`, paging, error chain) without editing unrelated commands. ADR-1's distinct-command decision means it touches **neither** `actors.go` (049 grows it) **nor** the `assignments` command (050) — a clean addition with no cross-module entanglement.
- **P0 | VI. Size-Aware by Design** → spec.md § Completeness, interface-cli.md § Interactions, plan.md Cross-cutting: walks every page via `paging.All` by default; `--first-page` opt-out emits a "more exist" signal; a mid-walk failure renders the partial set flagged incomplete. The `kind` filter rides every page. Explicitly never silently truncates.
- **P0 | VII. Working Software** → tasks.md acceptance criteria require `go build`/`go vet` clean and passing tests; T001 (impl) and T002 (BDD) ship as verified units, not code-only/test-only increments.
- **P0 | VIII. No Fabricated Data** → spec.md Non-Behaviors, plan.md ADR-2/ADR-3, interface-cli.md: reuses 048's `actors` render with explicit-absence guards (no invented values); the leaf-`404` explicitly adds no fabricated "no sub-roles" message; rows are bare actor records (id/name/kind) with **no** fabricated `focus`/`elected_until` (those belong to 047's assignment shape); the command interprets/summarizes nothing.
- **P0 | IX. Writes Require Explicit Intent** → spec.md Non-Behaviors, plan.md: a pure `GET`; the spec forbids create/invite/update/delete of actors and assignments. No mutation path.
- **P0 | X. Respect API Limits** → plan.md Cross-cutting/Error Communication: a `429` routes through the shared chain and 017 may auto-retry a safe `GET`. `If-Match`/`ETag` is N/A (no writes).
- **P0 | XII. Standalone Executable** → plan.md, tasks.md: adds only Go code to existing packages within the established self-contained binary; introduces no language runtime or external software dependency.
- **P0 | I + VIII (one-level fidelity)** → spec.md validation scenario, interface-cli.md § Interactions, tasks.md T001 risk: the roll-up issues exactly the one paginated read and never recurses into grand-child roles — neither inventing a transitive closure (I) nor fabricating descendant data (VIII).
- **P0 | I + VIII (actor shape, not assignment shape)** → spec.md Non-Behaviors + validation scenario, interface-cli.md § Output, tasks.md T001: the rows surface the endpoint's bare `Actor` shape and project no `focus`/`elected_until` — faithfully representing what `/subroles/actors` returns (I) without fabricating assignment fields the response does not carry (VIII).
- **P0 | V (distinct command, not a subcommand of `actors`)** → plan.md ADR-1, interface-cli.md § Surface/Consistency Notes: keeping the roll-up a separate role-keyed leaf (vs. a `subroles` subcommand of the positional-bearing `actors`) avoids the runnable-parent-with-children entanglement and the ordering coupling to 049/050 — composition without entanglement (V).

## Done-Criteria Checks

Not run — no `accords/governance/done-*.md` accords are deployed.

## Cross-Reference Checks

Not run — cross-reference checks derive from done-* accords, which are absent.

---

## Governance Notes

- **No `accords/governance/done-*.md` accords deployed**: done-criteria and cross-reference checks could not be generated. Consider creating `done-specify.md`, `done-plan.md`, `done-interface.md`, `done-scenarios.md`, and `done-tasks.md` to enable artifact-level quality checks. (This is a project-wide infrastructure gap, not specific to 051 — every spec in this repo is checked constitution-only.)
- **Constitution principle XI (Governance via Proposals)**: no applicable checks for this feature. 051 is a read-only roll-up over a `GET` endpoint and exposes no governance-structure mutation path, so there is nothing for the proposal-gating principle to constrain.
- **guardian-agent.md not loaded**: the Guardian agent definition is not present in this deployment; checks were generated and evaluated per the checklist process directly.
