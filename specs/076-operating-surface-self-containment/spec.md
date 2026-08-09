# Specification: Operating-Surface Self-Containment

**Feature**: 076-operating-surface-self-containment
**Role**: Definer
**Tier**: 1 (zero setup)

---

## System Overview

The agent operating surface (`plugin/`) ships to machines that have the plugin and the `glassfrog` CLI — and nothing else. Yet today 21 of its 26 files reach outside it: spec numbers used as cross-reference ids ("hand the verdict to the Constraint Discovery Path (065)"), development-repository paths in the single-source registry headers (`internal/build/..._guard_test.go`), and design citations (plan ADRs). To an operator without the repository, every one of these is a dangling pointer; to the surface's own charter — knowledge and guardrails, workflow-oriented and lightweight — they are development residue.

Operating-Surface Self-Containment is the standing rule plus its enforcement: every reference inside the surface resolves within the surface or to the CLI it drives, with the ban strict — the development repository is not acknowledged at all, not even as a pathless prose mention of its machinery. The capability has three parts that land together: the **conformance sweep** that rewords the existing artifacts, the **detection check** that fails the repository's merge-gating verification run on any violation (present or future), and the **constitutional principle** that future authoring inherits. Repo-side executable expectations that today pin the leaked spec numbers are re-derived to pin the in-surface names — the property they protect (a handoff resolvable where the operator stands) is preserved, not deleted.

---

## Behavioral Accord

### The rule

- When any file under the operating surface refers to another component, the reference resolves within the surface — a skill, agent, hook, registry, or reference file, named by its in-surface name or path — or to the `glassfrog` CLI.
- When a file hands work onward (a deferral, a boundary, a handoff), it names the receiving in-surface component; no spec number or other development-repository identifier is used as a reference id.
- When a file states an editing rule or rationale (the single-source registry headers), it states the rule in terms of the surface's own consumers; it does not mention the development repository or its machinery — no spec ids, no source or test paths, no plan/ADR citations, and no pathless prose like "a drift guard in the source repository".
- When a file refers to the world the surface operates in — the GlassFrog API and its published specification, Holacracy, credential locations on the operator's machine — that reference remains permitted; the ban is on the development repository, not on the operating world.

### Detection

- When the repository's merge-gating verification run executes, every file under the operating surface is checked for development-repository references; the checked set is derived by walking the surface, so a file added by a future feature is covered without any list being updated.
- When a violation is present, the run fails, naming the file, the offending text, and the remedy: replace the reference with the in-surface component name, or remove it.
- When the operating surface is missing or contains no files, the check fails loudly rather than passing vacuously.
- When a numeric or spec-like token is not a development-repository reference — a concrete example id (`prp_0123`), a version string, an exit code, the API specification the CLI implements — it does not trip the check.

### The conforming surface

- When this capability lands, every existing file under the surface conforms: spec-number ids are replaced by in-surface component names, repository paths and ADR citations are removed, and every cross-reference resolves for an operator who has only the plugin and the CLI.
- When a registry header explains why hand-edits are dangerous, it does so through the surface's own consequences (the gate fail-closes an unrecognized command) without invoking repository machinery.

### Re-derived expectations

- When a repo-side executable expectation previously asserted that surface text contains a spec number, it now asserts the in-surface name of the same component — the deferral property survives; no assertion is satisfied by deletion.
- When repo-side artifacts reference surface files (guards naming the registries they pin), those references are untouched: pointers flow repository → surface, never surface → repository.

### The standing accord

- When CONSTITUTION.md is read after this capability lands, it carries the self-containment principle in its house form — statement, rationale, detection — with the amendment following the constitution's own governance (documented justification, version bump), so specification and quality-gate stages author future surface artifacts against the rule.

---

## User Scenarios

**In order to** follow every handoff and boundary in the surface on a machine that has only the plugin and the CLI,
**as an** AI agent operating the glassfrog CLI,
**I want to** find each cross-reference resolvable in place, with no repository or spec catalog needed to decode it.

**In order to** keep future path features from leaking development residue into the shipped surface,
**as the** developer,
**I want** the merge-gating verification run to fail on any surface file that references the development repository, including files that do not exist yet.

**In order to** author new surface artifacts against the rule instead of rediscovering it in review,
**as the** developer running the specification pipeline,
**I want** the principle recorded in CONSTITUTION.md with its detection mechanism.

---

## Non-Behaviors

- The capability must not change what the surface does — no skill workflow, agent fence, command registry membership, or gate posture changes. **Why**: this is a wording-and-enforcement change; altering behavior under it would smuggle a capability change past the scenario coverage that pinned the current behavior.
- The capability must not rewrite any already-merged spec — neither the 062–073 records whose numbers the surface leaked nor any other merged sibling — to match the reworded artifacts. **Why**: merged specs are historical records — git history is the changelog; the living pins are the executable expectations, and those are re-derived here. Where a feature scenario's step text quoted a spec number, the step text changes while the merged spec's prose stays — an announced divergence from the verbatim-copy convention, recorded in this spec.
- The capability must not restrict repository artifacts from referencing the surface. **Why**: guards must name what they pin; the direction rule (repository → surface only) is the mechanism that lets enforcement exist without the surface acknowledging it.
- The capability must not ban references to the CLI's operating world. **Why**: the setup skill must name credential locations, and the paths must name the GlassFrog API and Holacracy concepts — self-containment means repository-independence, not world-independence; banning the world would gut the surface's purpose.
- The capability must not enumerate the files it checks. **Why**: a hard-coded file list silently exempts every file added after it — exactly the future leak the check exists to catch; the set is derived by walking the surface.
- The capability must not satisfy a failing repo-side assertion by deleting it. **Why**: each pinned spec number encodes a property — the artifact defers by a resolvable name; re-derive the assertion from the property, or the guard goes green while the thing it protects is gone.

---

## Integration Boundaries

- **Agent operating surface (`plugin/`)**: the artifact constrained and swept — skills, agents, hooks, registries, reference files, manifest. Wording changes only.
- **Merge-gating verification run**: where detection lives; a violation anywhere under the surface turns the run red before merge.
- **Existing per-path guards and feature suites**: the repo-side expectations that today assert spec numbers appear in surface text; their assertions are re-derived to the in-surface names. Their own repository-side references to surface files are unchanged.
- **CONSTITUTION.md**: gains the standing principle via its amendment process; downstream, the specification and quality-gate stages read it when authoring future surface artifacts.

---

## Driving Scenarios

### Happy path

**Scenario: A conforming surface passes verification**
Given every file under the operating surface references only in-surface components and the `glassfrog` CLI
When the merge-gating verification run executes
Then the self-containment check passes
And every cross-reference in the surface resolves without the development repository.

**Scenario: A handoff reads by name where the operator stands**
Given a skill that defers an authority question to another path
When its text is read on a machine with only the plugin and the CLI installed
Then the handoff names the receiving in-surface skill or agent
And no spec number or repository artifact is needed to follow it.

**Scenario: A future file is covered without registration**
Given a later feature adds a new file under the operating surface
When the merge-gating verification run executes
Then the new file is checked for development-repository references
And no list or configuration was updated to include it.

### Error scenarios

**Scenario: A spec-number reference turns the run red**
Given a surface file that gains a spec number used as a reference id
When the merge-gating verification run executes
Then the run fails, naming the file and the offending text
And the failure states the remedy: replace the reference with the in-surface component name, or remove it.

**Scenario: A repository mention fails even without a path**
Given a surface file that gains a mention of repository machinery — a test path, a plan ADR citation, or the pathless phrase "the source repository's drift guard"
When the merge-gating verification run executes
Then the run fails on that mention
And the strict ban is enforced: the development repository is not acknowledged in any form.

### Edge cases

**Scenario: Non-reference tokens do not false-positive**
Given surface files carrying a concrete example id like `prp_0123`, a version string, exit codes, and mentions of the GlassFrog API specification the CLI implements
When the merge-gating verification run executes
Then none of these trips the self-containment check
And the operating-world references remain intact.

**Scenario: An empty surface is a failure, not a pass**
Given the operating surface directory is missing or contains no files
When the merge-gating verification run executes
Then the self-containment check fails loudly
And it does not report success over a vacuously clean set.

---

## Validation Scenarios

> These are held out from the implementing agent for independent verification.

**Scenario: The sweep preserved every handoff**
Given the reworded surface and the pre-sweep artifacts
When each replaced spec-number reference is compared with its replacement
Then each replacement names the same in-surface component the number pointed at
And no deferral, boundary, or handoff was dropped to achieve conformance.

**Scenario: Re-derived assertions kept the property**
Given the repo-side expectations that previously asserted spec numbers in surface text
When the re-derived expectations are read
Then each asserts the in-surface name of the same component
And none was deleted or weakened to an assertion that any text at all is present.

**Scenario: The surface reads lightweight and workflow-oriented**
Given the swept surface read end to end
When its files are inspected for development residue
Then no file explains repository mechanics, design history, or enforcement machinery
And registry headers state their editing rules through the surface's own consequences.

**Scenario: The constitutional principle is in house form**
Given CONSTITUTION.md after the amendment
When the new principle is read
Then it carries a statement, a rationale, and a detection section like its siblings
And the amendment carries the documented justification and version bump the governance section requires.

**Scenario: Pointer direction is intact**
Given the repository artifacts that reference surface files
When their references are checked after the sweep
Then repository → surface references still resolve
And no surface file references the repository in any form.

---

## Assumptions

- **Every leaked id maps to an in-surface component**: investigation confirmed each spec-number reference resolves to something shipped in the surface (062 → the orientation skill, 063 → the write-gate hook and its registry, 064–069 → the five path skills and their agents, 072/073 → the two drafting reference files) — so the sweep is renaming, not restructuring. (Grep-verified this session: 21 files, ~150 references.)
- **The repo-side pins are known**: the assertions that must be re-derived live in the build-guard BDD suites and six feature files under `features/unequipped-agent-operators/`, which assert spec numbers appear in surface text. (Grep-verified this session.)
- **Strict form chosen deliberately**: the developer chose the strict ban over a resolvable-reference ban to keep the surface lightweight and workflow-oriented; registry headers lose their repository mentions entirely and lean on the surface's own fail-closed consequences. (Confirmed during specification.)
- **[ASSUMED] Check placement is a shaping decision**: the behavioral requirement is only that the check runs in the merge-gating verification and walks the surface; where it lives and how tokens are classified is deferred to `/score:plan`.

---

## Ambiguity Warnings

_None. The three open decisions — one spec or two, the accord's home, and strict vs resolvable-reference ban — were resolved with the developer during specification. The remaining open point (check placement and token classification) is technical, deferred to `/score:plan`._
