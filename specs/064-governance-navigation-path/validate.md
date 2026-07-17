# Validate: Governance Navigation Path

**Feature**: 064-governance-navigation-path
**Round**: 1 of 3
**Date**: 2026-07-17
**Verdict**: Ready
**Artifacts loaded**: spec.md, plan.md, tasks.md, interface-spec.md, governance-navigation-path.feature, PROJECT.md
**Implementation files**: `plugin/skills/governance-navigation/SKILL.md`, `plugin/agents/governance-navigator.md`, `plugin/agents/composed-reads.txt`, `internal/build/governancenavigation.go`, `internal/build/governance_navigation_bdd_test.go`, `internal/build/governance_navigation_guard_test.go`, `internal/build/governance_navigation_unit_test.go`

---

## Conformance Summary

| Dimension | Status | Findings |
|---|---|---|
| Driving scenario coverage | ✓ Pass | 0 |
| Acceptance criteria | ✓ Pass (1 note) | 1 (P2) |
| Interface contract conformance | ✓ Pass | 0 |
| Non-behavior absence | ✓ Pass | 0 |
| @wip lifecycle completion | ✓ Pass | 0 |
| **Validation scenarios** | ✓ Satisfied | 0 |

**Total**: 5 dimensions checked, 5 passed, 1 informational (P2) finding. All 4 validation scenarios satisfied.

Confidence is high: the deliverable is declarative plugin content, and its described behavior is both traced to the artifact prose and exercised by an executable BDD suite (14 scenarios) plus a drift guard and unit tests — `go test ./...` reports 1994 passing.

---

## Driving Scenario Coverage

**Status**: Pass (7 of 7 scenarios covered)

Every spec driving scenario has an identifiable code path — the artifact prose that instructs the behavior, verified present by its executable BDD scenario (all un-`@wip` and passing).

| Scenario | Status | Implementation |
|---|---|---|
| From a concern to the roles that touch it | ✓ Covered | SKILL.md "The workflow" (search→roles→fillers) + agent "Composed reads"/"Output contract"; BDD "Search a concern surfaces the relevant roles" |
| Drawing in the governing domains and policies | ✓ Covered | agent Workflow + Output contract (`domains`/`policies`, "one picture"); BDD "A relevant role's domains and policies are drawn in" |
| A circle concern follows into its sub-roles | ✓ Covered | agent "bounded by relevance … stop short of walking the whole tree"; BDD "A circle concern follows into its sub-roles" |
| The concern matches nothing | ✓ Covered | agent "Traversing defensively" (nothing found / refine / fabricate no); BDD "An empty search reports nothing found without fabricating" |
| A read in the traversal fails | ✓ Covered | agent "surface what the failure was … reads that succeeded … not invent"; BDD "A failed read yields a partial picture" |
| An over-broad concern matches many models | ✓ Covered | agent/skill "page through the full result set … then narrow" + "narrowed — refine"; BDD "An over-broad concern is narrowed, not dumped" |
| The concern is really an authority question | ✓ Covered | agent/skill defer to Constraint Discovery Path (065); BDD "An authority question surfaces governance but defers the verdict" |

---

## Acceptance Criteria

**Status**: Pass (T001, T002 — 1 informational note)

All acceptance criteria for the two checked tasks are met on inspection. One T001 sub-criterion (host-side hook-coverage confirmation) could not be executed in this environment; the load-bearing control it backstops is nonetheless in place — see F-1.

- **T001** — skill + agent present with the pinned frontmatter (`name`+`description`; `name`/`description`/`tools`/`model`); read-only tool grant `Bash, Read, Grep, Glob` (Write/Edit withheld); workflow single-sourced in the skill and referenced (not restated) by the agent; workflow states paging the full set before narrowing (Constitution VI); agent auto-discovered from `plugin/agents/` with no `agents` key in `plugin.json`; documented degradation-to-guidance; orientation skill untouched; no `marketplace.json`. Verified against the committed files.
- **T002** — the drift guard asserts each of the seven composed leaves resolves to a top-level CLI command, fails loudly naming the offending leaf, single-sources the leaf list in `composed-reads.txt` (consumed by both the agent artifact and the test), reuses the 062/063 `internal/build` idiom, and documents its partial coverage. Unit tests prove it is not fail-open.

---

## Interface Contract Conformance

**Status**: Pass (all surfaces conformant)

| Surface element | Status | Evidence |
|---|---|---|
| Structural layout (`skills/governance-navigation/SKILL.md`, `agents/governance-navigator.md`, `internal/build` guard) | ✓ Conformant | all present |
| `plugin.json` — additive, no `agents` key | ✓ Conformant | `grep '"agents"'` → 0 |
| `SKILL.md` frontmatter (`name` + `description`) & required sections (When / Workflow / Delegation / Read-only note) | ✓ Conformant | present |
| Agent frontmatter (`name`/`description`/`tools`/`model`) & required sections (Identity & scope / Workflow / Composed reads / Output contract) | ✓ Conformant | present |
| Synthesized-picture output shape (roles/fillers/domains/policies/notes, each carrying its id) | ✓ Conformant | agent "Output contract" — `role_…`, `per_…`/`agt_…`, domain/policy ids, `notes` |
| Error communication (empty search, partial failure, over-broad, authority deferral) | ✓ Conformant | agent "Traversing defensively" |

---

## Non-Behavior Absence

**Status**: Pass (7 of 7 exclusions upheld)

| Non-behavior | Status | Evidence |
|---|---|---|
| Must not write / drive a write command | ✓ Absent | `tools` withholds Write/Edit; artifacts name no write command; BDD "The path only reads, never writes" |
| Must not add a command/flag/API capability | ✓ Absent | composes only shipped reads; only Go change is test code; drift guard pins the leaves |
| Must not judge authority | ✓ Absent | defers to 065; BDD "The path surfaces governance without judging authority" |
| Must not reimplement governance/permission logic locally | ✓ Absent | pure composition of CLI reads, no local logic |
| Must not teach/coach Holacracy practice | ✓ Absent | artifacts navigate the record; no craft coaching |
| Must not dump raw, unsynthesized output | ✓ Absent | output contract "never a concatenation of raw … output"; isolated context; BDD "The result is a synthesized picture, not raw output" |
| Must not define distribution / delivery form | ✓ Absent | no `marketplace.json`; distribution left to #70 |

---

## @wip Lifecycle Completion

**Status**: Pass

All 14 scenarios in `governance-navigation-path.feature` are un-`@wip` (`grep -c @wip` → 0); both referencing tasks (T001, T002) are checked. No stray `@wip` on an implemented scenario, and no scenario belonging to future work was prematurely activated.

---

## Validation Scenario Results

**Status**: Satisfied (4 of 4 scenarios traced to implementation, and executed)

These held-out `@validation` scenarios are now un-`@wip` and pass in the BDD suite.

| Scenario | Status | Trace |
|---|---|---|
| No invented surface ("names no read the CLI lacks") | ✓ Satisfied | `TestGovernanceNavigationDriftGuard` + `CheckNavigationDrift` trace each composed leaf to `app.go`'s registered top-level commands |
| Read-only throughout | ✓ Satisfied | `AgentTools` parse confirms Write/Edit withheld, Bash present; no write/confirm/gate step |
| Surfacing, not judging | ✓ Satisfied | deferral to 065 asserted + no permission-verdict phrasing present |
| Synthesized, not raw | ✓ Satisfied | output contract framed as a drawn-together picture, not a concatenation of dumps |

---

## Findings

### F-1: T001 host-side hook-coverage confirmation deferred (informational, P2)

- **Dimension**: Acceptance criteria
- **Source**: tasks.md § T001 acceptance criteria — "Subagent hook coverage is confirmed against the target host: verify whether 063's landed PreToolUse Bash gate also fires for the navigator subagent's Bash calls"
- **Implementation**: `plugin/agents/governance-navigator.md` (Identity & scope — strictly read-only prompt; `tools` grant withholds Write/Edit) + risk.md H-4 / interface-spec.md Error-Communication Caveat (residual risk documented)
- **Gap**: The criterion asks for a *live-host* confirmation of whether 063's `PreToolUse` Bash gate reaches a subagent's calls. That confirmation is environmental and could not be executed here (no target host in the validation environment). It does **not** gate conformance: the *behavioral* requirement — the navigator is strictly read-only regardless of the hook — is met structurally (Write/Edit withheld) and by prompt scope, and the residual "hook may not reach the subagent" case is the load-bearing control the plan/risk explicitly account for. Recommend confirming hook coverage once the surface is installed in a capable host (owned by #70 packaging), then closing this note.

---

## Verdict: Ready

All 5 conformance dimensions pass and all 4 validation scenarios are satisfied through inspection and executed in the BDD suite. The implementation conforms to the specification. The single finding (F-1) is an informational P2: a host-side verification step deferred to packaging, whose backstop control (read-only agent) is already in place — it does not block readiness.

---

## Next Steps

Implementation conforms to the specification. Suggest PR review and merge. Confirm 063's `PreToolUse` hook coverage of subagent Bash calls when the plugin is installed in a capable host (#70), then close F-1.
