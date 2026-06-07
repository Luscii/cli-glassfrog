# Analyze: My Projects

**Feature**: 014-my-projects
**Artifacts analyzed**: spec.md, plan.md, interface-cli.md, interface-spec.md, features/self-service-reads/my-projects.feature, tasks.md
**Checklist context**: loaded — 18/18 checks pass (0 failures)
**Checks**: 21 (21 pass, 0 fail)
**Generated**: 2026-06-07

---

## Summary

All 21 checks pass. Consistency: 8/8. Completeness: 9/9. Coherence: 4/4.

| Category | Checks | Pass | Fail |
|---|---|---|---|
| Consistency (P0) | 8 | 8 | 0 |
| Completeness (P1) | 9 | 9 | 0 |
| Coherence (P2) | 4 | 4 | 0 |
| **Total** | **21** | **21** | **0** |

Check counts reflect scaling: two interface files (interface-cli.md, interface-spec.md) multiply the interface-related checks (C4, C6, K2, K5, K6); one feature file.

---

## Consistency: 8/8 passed

### Passed (8/8)

- **C1** spec.md § Integration Boundaries ↔ plan.md § System Architecture/Integration Design — every boundary spec names (010 Request Execution, 007 Request Authentication, the `GET /me/projects` API, 011/012/013 siblings, 016 Pagination, 004 Exit-Code) appears in the plan's architecture with a compatible role. No contradiction.
- **C2** spec.md § Behavioral Accord ↔ plan.md § System Architecture — every behavior (Listing, Filtering, Pagination boundary, Empty result, Error handling) is served by the architecture (single `Execute` call, reused `validateStatus` before request, `HasNextPage` signal, empty-list success, `classifyClientError`). No behavior is contradicted.
- **C3** spec.md § Non-Behaviors ↔ plan.md § System Architecture — the plan architects none of the excluded capabilities: no page-walking (deferred to 016), no `?include` embedding (ADR-2 declines it), no `--output json`, no raw-payload rendering, no mutation, no token reading. The exclusions and the architecture agree.
- **C4** plan.md § Architecture Decisions ↔ interface-cli.md § Surface — ADR-1 (reuse `Pagination`/envelope + `validateStatus`), ADR-2 (no `--include`), ADR-3 (guard-registered leaf + injected seam + pure trio) are all reflected: no `--include` flag, `--status` validated by the shared validator, the reshaped projection.
- **C4** plan.md § Architecture Decisions ↔ interface-spec.md § Surface — the same ADRs are reflected in the Go surface (`newMyProjectsCommand`/`runMyProjects`/`formatMyProjects`/`myProjectsSeam`, `Project` added to `internal/glassfrog`, reused symbols declared "Consumed unchanged").
- **C5** plan.md § System Architecture ↔ tasks.md § Task Scope — every task builds a plan-named component: T001 → `glassfrog.Project`; T002 → the command leaf + pure trio + seam; T003 → wiring + godog. No task builds something the plan does not mention.
- **C6** interface-cli.md § Surface ↔ my-projects.feature steps — scenario steps reference only surfaces the CLI accord defines (the my-projects endpoint, the `--status` filter, projection fields id/status/description/role, the "more results available" signal, the no-role marker, the `--base-url` problem). No step uses an undefined surface.
- **C6** interface-spec.md § Surface ↔ my-projects.feature steps — the package-level outcomes the steps assert (transport fail-safe refusal, decode failure → internal error, transport failure outcome, non-2xx surfaced-not-classified, exactly-one-request) all map to the pinned `runMyProjects`/seam/`Execute` surface and the reused error mapping.

---

## Completeness: 9/9 passed

### Passed (9/9)

- **K1** spec.md § Driving Scenarios → my-projects.feature — every driving scenario has a Gherkin equivalent: list-with-no-filter, filter-by-supported-status, more-results-than-one-page, no-usable-token, API-non-2xx, no-matching-projects, invalid-status-rejected-before-request. The four Validation Scenarios are present and held as `@validation @wip`. Architecture-informed scenarios (network failure, undecodable 200, malformed base URL, null-role marker) are additional coverage, not gaps.
- **K2** spec.md § Integration Boundaries → interface-cli.md — the externally-facing CLI boundary has its interface file; reused upstream boundaries (010/007/011/012/016/004) are documented as consumed-unchanged dependencies, a justified absence (no new surface to pin). PASS.
- **K2** spec.md § Integration Boundaries → interface-spec.md — the Go package-API boundary (the `Project` model + command internals) has its interface file; the reused symbols are explicitly listed under "Consumed unchanged (not defined here)". PASS.
- **K3** plan.md § Implementation Strategy/Phases → tasks.md — Phase 1 → T001; Phase 2 → T002 + T003. Every plan phase has task decomposition; the plan's note that there are two phases (no separate validator task) matches tasks' three-task graph.
- **K4** plan.md § System Architecture/Components → tasks.md § Task Scope — every component has an implementing task: `Project` (T001); `newMyProjectsCommand`/`runMyProjects`/`formatMyProjects`/`myProjectsSeam` (T002); the single `MustRegister` wiring + godog suite (T003). Reused components correctly have no build task.
- **K5** interface-cli.md § Surface → my-projects.feature — the command, `--status` flag (supported + unsupported paths), the projection (incl. empty-list and no-role), the "more available" signal, and the `--base-url` error path all have scenario coverage.
- **K5** interface-spec.md § Surface → my-projects.feature — the pure-trio outcomes, the validate-first tripwire, exactly-one-request, decode/transport/auth branches all have scenario coverage (behavioral + held `@validation`).
- **K6** spec.md § User Scenarios → interface-cli.md — all three user scenarios (see my projects, filter by status, signal-more-results) have CLI interface coverage.
- **K6** spec.md § User Scenarios → interface-spec.md — the same three user-facing flows trace to the package-level surface that realizes them (`runMyProjects` orchestration, `formatMyProjects` projection + signal).

---

## Coherence: 4/4 passed

### Passed (4/4)

- **H1** Terminology across all artifacts — key concepts use consistent names everywhere: the shared `/me*` "projection", the "more results available" signal, the reused `validateStatus`, the `Project` resource, the "no-role marker" for null `role_id`, and the `has_sub_projects`/`has_actions` presence booleans. No concept is renamed across artifacts.
- **H2** Detail symmetry across spec↔plan and plan↔tasks — detail is proportionate; no artifact carries 3x+ more detail on a shared topic than its neighbor. (The dependency/sequencing story is described at compatible depth in plan § Implementation Strategy and tasks § Dependency Graph.)
- **H3** Scope alignment across spec.md + interface-*.md + tasks.md — the capability set is identical across all three: one read command, one `--status` filter, first-page projection with a signal, no `--include`, no `--output json`. Nothing is added or dropped silently. The intentional reuse (no new validator/exit-code/pagination type) is described consistently everywhere as reuse, not as a missing capability.
- **H4** Phase coverage plan.md + tasks.md — the plan's two phases map structurally onto tasks' Phase 1 (T001) and Phase 2 (T002 → T003), with matching dependencies (T002 depends on T001 and on 010/011/012/013 being implemented; T003 depends on T002). Tasks reference no phase absent from the plan.

---

## Checklist Correlation

Checklist correlation: no overlapping findings between checklist and analyze results. The checklist found 0 failures across 18 constitution checks; analyze found 0 failures across 21 cross-artifact checks. The vertical (constitution/done-criteria) and horizontal (cross-artifact) passes both clear.

---

## Governance Notes

- **No `done-*` accords**: checklist ran constitution-only; this does not affect analyze, which checks cross-artifact relationships independently of done-criteria.
- **Cross-spec dependency sequencing (informational, not a cross-artifact finding)**: every artifact consistently states that 014 reuses 011 (`internal/glassfrog`, `classifyClientError`, `Outcome`/`ExitCode` codes 3/6, `--base-url`), 012 (`my` parent, `Pagination`, list envelope, "more available" signal), and 013 (`validateStatus` + status set, projection/seam shape), and that 011/012/013 are shaped but not yet on main while 010 is landed. tasks.md and plan.md agree on the "first status-filtered read to land creates the shared `validateStatus`, the other reuses" fallback. This is an internally-consistent build-ordering constraint, not an artifact contradiction — analyze records it so the Builder treats the 011/012/013 base as a hard prerequisite for T002/T003.
- **Checklist context**: loaded — 18/18 pass, 0 failures.
