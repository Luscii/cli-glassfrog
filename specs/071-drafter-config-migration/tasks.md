# Tasks: Drafter Config Migration

**Feature**: 071-drafter-config-migration
**Concretization**: Full context (plan + spec + interface + scenarios)
**Inputs**: plan.md, spec.md, interface-spec.md, features/no-automated-pipeline/drafter-config-contract.feature

---

> **One pull request.** All three phases ship together. The config schema and the guard that parses it cannot land apart (spec — Schema/version coupling), and the repository's convention puts contract-doc updates in the same commit as the contract change. The phases below are build order inside that PR, not separate merges.
>
> **No app code.** Like 030, this feature's only Go is in the test-only `internal/build` guard package. Nothing in the CLI changes, and no runtime behavior is added.
>
> **Nothing verifies drafter output.** The spec forecloses a synthetic-pull-request harness, and `release-drafting.yml` triggers only on `push: main` and blocks nothing. The guards are the entire pre-merge assurance surface. Three scenarios in the feature file are behavior-preservation statements that no test can execute — see T004.
>
> **Current state to build from.** #179 merged on 2026-08-03. `.github/workflows/release-drafting.yml` now pins `release-drafter/release-drafter@v6.4.0`, and `.github/dependabot.yml` blocks automated major bumps for it. Action and config currently agree — both on the superseded schema. **A workflow edit is therefore required**, and it belongs in T001 alongside the config: landing the config without moving the pin *is* the silent degradation this feature exists to prevent. T003's guard reddens CI on any later attempt to separate them, which is why T003 must not be deferred to a follow-up PR.
>
> **A comment argues against this change.** The drafting step's pin comment makes the empirical case for staying on the previous major and ends with a forward-pointer to "a coordinated change" — this one. T001 rewrites it. Keep the empirical record (it is why the pre-release observation is expected); remove the stale forward-pointer; do not let the rewrite imply the tag loss is fixed.
>
> **Lint is a separate gate.** Reshaping `ReleaseDrafterConfig`'s fields re-aligns sibling struct columns under gofmt. Run `gofmt -l .`, `go vet ./...`, `go test ./... -count=1`, and `golangci-lint run` before pushing — `go test` alone will not catch the formatting change.

## Dependency Graph

Phase 1: The Atomic Migration (2 tasks, no dependencies) [US1]
Phase 2: The Coupling Guard (2 tasks, depends on Phase 1) [US3]
Phase 3: The Live-Contract Sweep (1 task, depends on Phases 1, 2) [Shared]

5 tasks total | 0 phases parallelizable | Builder: pipeline (single spec)

## Branching Guidance

**Pipeline mode**: `spec/071-drafter-config-migration/base` (cut from current `main`) → `spec/071-drafter-config-migration/task-1` … `task-5`.

Fetch `origin/main` before cutting the base — the drafter action version on `main` is both an input to T003 and something T001 edits, and it moved once already (#179).

Within Phase 1, T002 is additive test work over the shape T001 lands. T002 and T003 touch different files (`labelcontract_test.go` versus two new files) and can run in parallel once T001 is in. All five collapse into one PR at the end.

---

## Phase 1: The Atomic Migration [US1]

- [ ] **T001** [US1] Move the pinned action version forward, restructure `.github/release-drafter.yml` to the current schema, and reshape the guard to read it
  - **Scope**: The coupled change — all three halves of it. Raise the drafting step's pinned action major to one that understands the current schema and rewrite the pin comment that argues for holding it back. Move the seven category labels into each changelog category's `when`, the exclusion into a `pre-exclude` category, and the three semver buckets plus the patch fallback into four `version-resolver` categories. In `internal/build/labelcontract.go`, reshape `ReleaseDrafterConfig`/`DrafterCategory`, add `DrafterWhen` and the three category-type constants, retain the superseded keys as rejection detectors, and re-derive the four verdicts from the new positions. Rewrite the `validDrafterYAML` fixture in `labelcontract_test.go` to the new shape so the existing drift table still exercises what it did before. Version, config, and guard are one task because any split leaves either `go test` red or the drafter silently miscategorising — they are not independently reviewable.
  - **Acceptance criteria**:
    - `.github/workflows/release-drafting.yml`'s drafting step pins a `release-drafter/release-drafter` major at or above the schema floor T003 declares, at an exact patch version (the repo's action-pinning discipline).
    - That step's pin comment is rewritten: the empirical tag-loss record is **kept** (it is why a post-merge pre-release observation is expected), the forward-pointer to "a coordinated change … which will fail loudly when they move" is removed as landed, and the new text does not state or imply that this change fixes the tag loss.
    - `.github/dependabot.yml` is **not** modified. The `ignore` rule blocking automated majors stays; this is a hand bump within that policy, not a change to it.
    - `.github/release-drafter.yml` carries twelve `categories` entries: one `pre-exclude`, seven changelog (no `type` key), three `version-resolver` with a `when`, one `version-resolver` with `semver-increment: "patch"` and **no `when` key**.
    - No top-level `version-resolver` or `exclude-labels` key remains, and no category carries `labels`/`label`.
    - No `title` or `collapse-after` on any `pre-exclude` or `version-resolver` entry; no `exclusive` anywhere; no `semver-increment` on the `pre-exclude` entry. Each of these would emit a deprecation or misuse warning.
    - `tag-template`, `name-template`, `version-template`, `change-template`, `template`, `replacers`, and all seven category `title` values are byte-identical to before.
    - The file's header comment is rewritten: it names the new positions, keeps the 030 ADR citations, adds 071, and states that `exclusive` is deliberately omitted.
    - `drafterCategoryLabels` gathers labels from **changelog categories only** — including version-resolver entries would triple-count the semver labels and read `no-release-note` as an unexpected category label.
    - `ResolverDefault` keeps its name and value `"patch"`; its doc comment says it now pins the condition-less `version-resolver` category's increment and still traces to 030 ADR-2.
    - `DrafterCategory.When` is a pointer, so "no condition" is distinguishable from "empty condition".
    - `VersionResolver` and `ResolverBucket` are removed as contract-source types.
    - `TestLabelContract_RealConfig` passes against the shipped files; the pre-existing drift cases still fail for the same reasons, with unchanged message wording for the four verdicts.
    - `gofmt -l .` is empty and `golangci-lint run` is clean.
  - **Dependencies**: None
  - **Plan reference**: Phase 1: the atomic migration; ADR-1 (hard-switch), ADR-2 (condition-less fallback), ADR-4 (reject the superseded shape), ADR-6 (`exclusive` omitted), ADR-7 (`when` mapping form only); Migration Strategy
  - **Interface references**: interface-spec.md: `.github/release-drafter.yml` — target structure; `internal/build/labelcontract.go` — exported contract
  - **Scenario references**: drafter-config-contract.feature: "A drafting run reports no schema deprecations"
  - **Risk**: ⚠️ Do **not** set `exclusive: true`. `.github/dependabot.yml`'s header comment claims the drafter "assigns the FIRST matching category" — false against both action majors, which push a pull request into every matching category. Setting it to "preserve" that would silently change note placement. Correcting the comment is out of scope (plan Risks).

- [ ] **T002** [US2] [P] Add drift cases for every position the migration moved
  - **Scope**: Extend `TestLabelContract_Drift`'s table in `internal/build/labelcontract_test.go`. Additive test work only — no production change.
  - **Acceptance criteria**:
    - A changelog category whose `when` names no label fails, and the violation names the file and the missing label.
    - A missing `pre-exclude` category fails, naming the `pre-exclude` category and `no-release-note`.
    - A `version-resolver` category carrying the wrong `semver-increment` fails, naming the bucket and both labels.
    - **Deleting** the condition-less `version-resolver` category fails, and the message reads as an absent declaration rather than an empty-string mismatch. This case is mandatory: the action's built-in fallback is also patch, so removing the declaration changes no observable output — only the guard can catch it.
    - A config still carrying `version-resolver`, `exclude-labels`, or category-level `labels`/`label` fails with a message naming the **schema**, distinguishable from a missing-label message.
    - Assertions use set-difference and substring matching, not map-by-key lookup — a lookup against a zero-valued expectation would pass on a dropped entry, which is exactly the omission these cases exist to catch.
  - **Dependencies**: T001
  - **Plan reference**: Phase 1: the atomic migration; Cross-cutting Concerns > Testing strategy
  - **Interface references**: interface-spec.md: Error Communication
  - **Scenario references**: drafter-config-contract.feature: "A category losing its label predicate fails the guard"; "Removing the declared fallback fails the guard"; "A configuration left on the superseded schema is rejected by name"

---

## Phase 2: The Coupling Guard [US3]

- [ ] **T003** [US3] Add `internal/build/drafterschema.go` and its guard test
  - **Scope**: The new invariant: the drafting workflow's pinned action major must be at or above the floor the config's schema requires. Production code and its tests together, following the package's one-file-per-concern convention.
  - **Acceptance criteria**:
    - `DraftingWorkflowFileName`, `DrafterActionRepo`, and `DrafterSchemaMinMajor` are declared, with `DrafterSchemaMinMajor`'s comment carrying the property and reason (the `GoReleaserVersion` precedent in `workflow.go`), not just the number.
    - `DraftingWorkflowFileName` is visibly distinguished from the existing `WorkflowFileName`, which is 022's `release.yml`.
    - `LoadDrafterSchemaCoupling` reuses `RepoRoot()`, `loadYAML`, `ReleaseDrafterConfig`, and `workflow.go`'s `Workflow` type — it does not re-declare workflow structures. Note that `Workflow.On` is tagged `json:"true"` because YAML coerces the `on:` key; that quirk is already handled and documented there.
    - `CheckDrafterSchemaCoupling` takes **both** the config and the workflow, deriving the required floor from the config's shape rather than assuming it.
    - The pinned major is derived from the step whose `Uses` begins with `DrafterActionRepo + "@"`. `v7`, `v7.7`, and `v7.7.0` all yield `7`.
    - A ref with no derivable major — a branch, a commit SHA, a tag without a leading `vN` — produces a violation stating the major could not be determined. It must not default to zero and must not pass.
    - No step using the drafter action is a violation, not a pass.
    - A real-file test mirrors `TestLabelContract_RealConfig`: it calls the loader and asserts the shipped workflow and config pass. Without it the exported check is never invoked and the guard is dead weight.
    - Drift cases cover a below-floor major, an underivable ref, and an absent drafter step, each asserting the violation names the workflow file and the offending value.
  - **Dependencies**: T001
  - **Plan reference**: Phase 2: the coupling guard; ADR-3 (sibling file sharing the parse), ADR-5 (derived major, named floor)
  - **Interface references**: interface-spec.md: `internal/build/drafterschema.go` — exported contract; Interactions > Pinned-major derivation
  - **Scenario references**: drafter-config-contract.feature: "A pinned action version behind the configuration schema fails the guard"; "A pinned reference with no derivable major fails rather than passes"
  - **Risk**: ⚠️ Must ship in the same PR as T001. Since #179 landed, `main` sits on the previous major, so T001 moves the pin and the config together — this guard is what stops a later change from separating them again. Deferring it leaves exactly the silent-degradation window this feature exists to close.

- [ ] **T004** [US3] Wire the feature file to a godog runner, and record which scenarios stay unexecuted
  - **Scope**: `features/no-automated-pipeline/*.feature` currently has no godog runner, while `runtime-dependent-distribution/*` and `unequipped-agent-operators/*` do (see `internal/build/release_bdd_test.go` for the pattern). Add a runner over `drafter-config-contract.feature` covering the scenarios the guards make observable, and clear their `@wip` tags. **This task is not named in plan.md** — it closes a hole the scenarios step opened, and is listed rather than left implicit so the decision is visible.
  - **Acceptance criteria**:
    - A godog suite in `internal/build` runs `drafter-config-contract.feature`, following the existing runner pattern in the package.
    - Every scenario under "Express the configuration in the schema the running action reads" and "Fail CI when the pinned action version and the configuration schema disagree" is executed against the T001–T003 guard functions, and its `@wip` tag is removed.
    - The guard-observable scenarios under "Move labels between configuration positions and nothing else" — the two failure cases and the three `@validation` ones — are likewise executed where the assertion is structural.
    - The three behavior-preservation scenarios ("A feature merge still bumps the draft to the next minor", "The declared fallback supplies the patch bump", "The exclusion survives the realignment") **keep `@wip` and stay unexecuted**, and the runner's tag filter excludes them. They assert drafter output, which the spec forecloses verifying. A comment in the runner states this, so the tags read as a deliberate boundary rather than unfinished work — the same documentation-grade status as the neighbouring scenarios in `release-drafting.feature`.
    - No scenario is reworded into a config-shape assertion to make it executable. Restating "a features PR bumps minor" as "the config declares a minor resolver for `features`" would be a shape check wearing a behavior label.
  - **Dependencies**: T001, T003
  - **Plan reference**: Phase 2: the coupling guard; Cross-cutting Concerns > Testing strategy; spec Non-Behaviors (no runtime verification)
  - **Scenario references**: drafter-config-contract.feature (all thirteen scenarios — ten executed, three deliberately not)

---

## Phase 3: The Live-Contract Sweep [Shared]

- [ ] **T005** [Shared] Update spec 030's live-contract documents to the new positions
  - **Scope**: Sweep `specs/030-release-drafting/` for statements describing the configuration's shape. Update the live contract; leave history alone.
  - **Acceptance criteria**:
    - `plan.md`, `interface-spec.md`, and `validate.md` — which carry the most references — describe the current positions.
    - `analyze.md` and `risk.md` are checked and updated where they name the superseded keys.
    - `spec.md` is **verified, not assumed**: its mentions of "categories" are tool-agnostic behavioral prose about note sections and should need no change. Confirm rather than edit reflexively.
    - Completed `- [x] Txxx` entries in `tasks.md` are left exactly as written — they record what was done, not what the contract is now. Only forward-looking contract statements in that file change.
    - Each updated document points at 071 for the schema while keeping its 030 ADR citations, since the invariants are still 030's.
    - No count, list, or enumeration of configuration keys is left stale — grep the whole directory for `version-resolver` and `exclude-labels` and confirm every surviving occurrence is deliberate history.
  - **Dependencies**: T001, T002, T003, T004
  - **Plan reference**: Phase 3: the live-contract sweep
  - **Risk**: ⚠️ The distinction between live contract and historical record is the whole difficulty here. A blanket find-and-replace across `specs/030-release-drafting/` would rewrite completed task records and destroy the audit trail (spec Non-Behaviors).
