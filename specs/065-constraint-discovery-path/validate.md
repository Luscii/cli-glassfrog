# Validate: Constraint Discovery Path

**Feature**: 065-constraint-discovery-path
**Round**: 1 of 3
**Date**: 2026-07-18
**Verdict**: Ready
**Artifacts loaded**: spec.md, plan.md, tasks.md (2 of 2 tasks complete), interface-spec.md, features/unequipped-agent-operators/constraint-discovery-path.feature (17 scenarios), PROJECT.md
**Implementation files**: 7 — `plugin/skills/constraint-discovery/SKILL.md`, `plugin/agents/constraint-navigator.md`, `plugin/agents/constraint-discovery-composed-reads.txt`, `internal/build/constraintdiscovery.go`, `internal/build/constraint_discovery_bdd_test.go`, `internal/build/constraint_discovery_guard_test.go`, `internal/build/constraint_discovery_unit_test.go`

**Independence note**: this validation ran in the same session that implemented the feature (pipeline mode). Separation is procedural, not structural: every check below was re-traced against the committed artifacts and fresh test runs (`go test ./internal/build/ -count=1`: 21 tests pass; full suite: 2017 tests, 12 packages), not taken from the Builder's memory. The @validation scenarios were concretized into the executable feature file during the scenarios stage and were therefore visible to the Builder; the held-out property rests on the spec's Validation Scenarios section, and each is independently re-traced below.

---

## Conformance Summary

| Dimension | Status | Findings |
|---|---|---|
| Driving scenario coverage | ✓ Pass | 0 |
| Acceptance criteria | ✓ Pass | 0 |
| Interface contract conformance | ✓ Pass | 0 |
| Non-behavior absence | ✓ Pass | 0 |
| @wip lifecycle completion | ✓ Pass | 0 |
| **Validation scenarios** | ✓ Satisfied | 0 |

**Total**: 5 dimensions checked, 5 passed, 0 findings

---

## Driving Scenario Coverage

**Status**: Pass (7 of 7 spec driving scenarios covered)

The deliverable is declarative plugin content, so "code path" means the artifact section that carries the behavior plus the executable BDD scenario that pins it (all green in `internal/build/constraint_discovery_bdd_test.go`).

| Spec scenario | Status | Implementation |
|---|---|---|
| A wanted action falls under another role's domain | ✓ Covered | SKILL.md § The workflow (steps 2–4) + agent § Output contract characterization part 1 ("held by another role… needing its permission or a proposal"); BDD "A wanted action under another role's domain is surfaced with its owner" |
| A wanted action is shaped by a policy | ✓ Covered | Agent § Output contract characterization part 2 ("the constraint to observe and drawn together with any governing domain"); BDD "A policy that shapes the action is surfaced as a constraint to observe" |
| A wanted action that nothing in the record constrains | ✓ Covered | Agent § Output contract ("no domain in view governs it… not a \"you are permitted\" verdict") + § Traversing defensively "Nothing found"; BDD "An unconstrained action surfaces the absence without asserting permission" |
| A read in the discovery fails | ✓ Covered | Agent § Traversing defensively "A read fails" (surface the failure, return the picture from the reads that succeeded, do not invent); BDD "A failed read yields a partial picture" |
| The record does not clearly answer | ✓ Covered | Agent § Output contract + § Traversing defensively "An ambiguous record" ("the record does not clearly answer", never fabricate an authority ruling); BDD "An ambiguous record is reported as unclear, not resolved by a guess" |
| The wanted action is too vague to locate its governance | ✓ Covered | SKILL.md § Clarify when the action is too vague (skill asks via structured ask mechanism before delegating; decline → stop; agent never invoked on a guess); BDD "A too-vague action is clarified by the skill before any traversal" |
| An over-broad action matches many models | ✓ Covered | SKILL.md § The workflow pagination paragraph + agent § "An over-broad action" (page the full result set, then most relevant, with a narrowed note); BDD "An over-broad action is narrowed, not dumped" |

The three feature-file scenarios proposed by the interface stage beyond the spec's driving set (own-role authority, incomplete-roles uncertainty, registration/degradation) are likewise covered and green.

---

## Acceptance Criteria

**Status**: Pass (2 of 2 tasks, all criteria met)

**T001** (skill + agent + registry):

| Criterion | Evidence |
|---|---|
| SKILL.md exists, frontmatter `name: constraint-discovery` + `description` stating *when* + synthesized picture, worded off 064/062 triggers | Frontmatter present; description names the four authority questions, the synthesized picture, and explicitly excludes tension-work/who-fills (064) and CLI mechanics (062) |
| Body carries when / workflow / clarify / delegation / surface-not-rule + read-only note, deferring to `glassfrog <command> --help` and orientation (062) | All five sections present; § Boundaries defers mechanics to orientation and flags to `--help` |
| Clarify-when-vague lives in the skill, via the host's structured ask, before delegating; decline stops the path; agent never invoked on a guess | § Clarify when the action is too vague (ADR-3), asserted by BDD too-vague scenario |
| Paging-capable reads (`search`, `roles`, `domains`, `policies`) page the full set before narrowing | § The workflow closing paragraph + agent "Page before narrowing" |
| `me roles` does not paginate → `owned_by_caller` **uncertain**, never a definite false | SKILL.md step 5 + agent "`me roles` is not paged" + tri-state contract; BDD incomplete-roles scenario |
| Agent registered by auto-discovery; documented degradation | No `agents` key added (`plugin.json` untouched; `ManifestDemandsNoSetup` asserted in BDD); SKILL.md § Delegation degradation paragraph |
| Subagent hook coverage note carried from 064 | SKILL.md § Boundaries: 065 drives no write, does not depend on 063's `PreToolUse` gate; prompt strictly read-only |
| Agent frontmatter `name`/`description`/read-only `tools` incl. `Bash`, excl. `Write`/`Edit`, no interactive tool | `tools: Bash, Read, Grep, Glob` (verified); "You never ask the operator" in § Identity & scope |
| Composed characterization: domain finding (own / other+id / absence ≠ permitted) + policies compose even in own domain + governance change → proposal + "record does not clearly answer"; every element carries its id | Agent § Output contract, three numbered parts + ambiguity outcome; ids per element (`role_…`, `pol_…`) |
| Workflow single-sourced in the skill; agent references it; clarify branch only in the skill | Agent § Workflow: "defined once in the `constraint-discovery` skill… do not keep a divergent copy"; clarify has no agent counterpart |
| Read-only, no write command named, no permission rules, no fabricated ruling; 062/064 artifacts untouched; no marketplace.json | `git diff 5260af0..HEAD --stat` touches no orientation/governance-navigation/governance-navigator files, no `plugin.json`, no `marketplace.json` |

**T002** (drift guard):

| Criterion | Evidence |
|---|---|
| Guard asserts `search`, `roles`, `tree`, `domains`, `policies`, `policy` (top-level) and `me roles` (subcommand of `me`) resolve in the CLI registry | `TestConstraintDiscoveryDriftGuard` via `LiveTopLevelCommands` + `LiveMeSubcommands`; green |
| Fails loudly naming the offending leaf | `CheckConstraintDrift` findings name the leaf; `TestCheckConstraintDrift` proves not-fail-open for top-level, `me` subcommand, unanchorable path, and prose drift |
| Both sides source-derived, no hard-coded list | Leaves from `constraint-discovery-composed-reads.txt`; live surface parsed from `internal/cli/app.go` Assemble + `Use:` tokens |
| Deliberately-uncovered facts documented, not silent | Guard-test COVERAGE / NOT COVERED comment (flags, picture/characterization prose, read-vs-write classification, parser robustness) |
| Reuses the `internal/build` config-guard home/idiom (064 model) | Same package, same joinDrift/registry/anchor idiom as `governance_navigation_guard_test.go` |
| Single-sourced leaf list consumed by both artifact and test | Agent § Composed reads names the registry file as authoritative; guard reads the same file; drift check (c) pins the agent naming every leaf |

---

## Interface Contract Conformance

**Status**: Pass (specification touchpoint; all pinned contracts conformant)

| Contract (interface-spec.md) | Status |
|---|---|
| Structural layout: `plugin/skills/constraint-discovery/SKILL.md`, `plugin/agents/constraint-navigator.md`, guard in `internal/build/` | ✓ Conformant (guard file named `constraint_discovery_guard_test.go`, matching the sibling naming convention) |
| `plugin.json`: no `skills`/`agents` key added, no field rewritten | ✓ Conformant (file untouched; BDD asserts `ManifestDemandsNoSetup`) |
| SKILL.md frontmatter: `name` + `description` only, description = trigger with 064/062 exclusions | ✓ Conformant |
| SKILL.md required sections (when / workflow / clarify-when-vague / delegation / surface-not-rule + read-only) | ✓ Conformant |
| Agent frontmatter: `name`, `description`, read-only `tools` (Bash in, Write/Edit out, no ask tool), `model: inherit` | ✓ Conformant |
| Agent required sections (identity & scope / workflow-by-reference / composed reads / output contract) | ✓ Conformant |
| Synthesized-picture shape: `action`, `domains` (id, `role_id`, `role_name`, brief, tri-state `owned_by_caller`), `policies` (`pol_…`, `role_id`, brief), composed `characterization`, `notes` | ✓ Conformant |
| Interactions: clarify → delegate → isolated traversal → picture only; single-sourced workflow | ✓ Conformant |
| Error communication rows (too-vague clarify/stop, empty search, partial failure, record-unclear, over-broad narrowing, missing-agent degradation, drift-guard red, stated-partial coverage, governance-write nuance) | ✓ Conformant |

---

## Non-Behavior Absence

**Status**: Pass (7 of 7 non-behaviors absent)

| Non-behavior | Inspection result |
|---|---|
| Must not write / drive any write command | No write leaf in the registry or artifacts; `tools` withholds Write/Edit; agent carries "no confirm or gate step"; asserted by BDD read-only scenario |
| Must not add any command, flag, or API capability | Diff contains no `internal/cli` change — only plugin data files and `internal/build` test/guard code |
| Must not reimplement permission rules / compute allow-deny locally | Artifacts disclaim it and carry none of the verdict phrasings; BDD verdict-phrase sweep (blacklist) green |
| Must not fabricate an authority ruling under uncertainty | "the record does not clearly answer" is a first-class outcome; "never fabricate an authority ruling" pinned by BDD |
| Must not judge/interpret the tension's substance or coach Holacracy | Neither artifact interprets tensions or advises practice; the skill's description scopes to constraint surfacing |
| Must not dump raw, unsynthesized output | Isolation-by-subagent + "never a concatenation of raw, unsynthesized command output"; BDD synthesized-not-raw scenario |
| Spec must not fix distribution / delivery form | No `marketplace.json`; delivery form was resolved in plan ADR-1, not in spec.md (unchanged) |

---

## @wip Lifecycle Completion

**Status**: Pass (0 remaining @wip tags)

All 17 scenarios in `constraint-discovery-path.feature` are un-@wip'd and execute in `TestConstraintDiscoveryFeatures` (T001: 16, T002: 1). No scenario referenced by a checked task retains @wip; only tag lines changed in the feature file (verified via diff — no Given/When/Then content was modified).

---

## Validation Scenario Results

**Status**: Satisfied (5 of 5 scenarios traced to implementation)

| Scenario | Status | Trace |
|---|---|---|
| No invented surface | ✓ Satisfied | Drift guard green: all 7 registry leaves resolve against the shipped CLI (`policy` top-level at `app.go` Assemble; `me roles` via the `me` subcommand surface) and the agent names each |
| Read-only throughout | ✓ Satisfied | `tools: Bash, Read, Grep, Glob` (no Write/Edit, no ask tool); artifacts state "no write, confirm, or gate step"; no write leaf named anywhere |
| Surfacing, not ruling | ✓ Satisfied | Independent re-read of both artifacts finds no permission verdict computed from local logic; explicit disclaimers plus the BDD blacklist sweep over verdict phrasings ("you are allowed to", "permission granted", …) |
| No fabricated ruling under uncertainty | ✓ Satisfied | Agent § Identity: "state what is unclear and surface what it found — assert no permitted-or-forbidden verdict you cannot ground in the record"; ambiguity yields "the record does not clearly answer" with what was found |
| Synthesized, not raw | ✓ Satisfied | Output contract returns only the drawn-together picture; raw output structurally confined to the subagent's isolated context |

---

## Verdict: Ready

All 5 conformance dimensions pass with zero findings. All 5 validation scenarios are satisfied by independent trace and by execution (the suite runs them as part of the 17 green scenarios). Both tasks' acceptance criteria are met with fresh evidence. The implementation conforms to its specification.

---

## Next Steps

Implementation conforms to the specification. Suggest PR review and merge.
