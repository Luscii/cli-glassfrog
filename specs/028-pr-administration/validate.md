# Validate: PR Administration

**Feature**: 028-pr-administration
**Round**: 1 of 3
**Date**: 2026-06-09
**Verdict**: Ready
**Artifacts loaded**: spec.md, plan.md (§ System Architecture + ADRs), tasks.md (3 of 3 tasks complete), interface-spec.md, features/no-automated-pipeline/pr-administration.feature, PROJECT.md
**Implementation files**: 3 declarative artifacts — `.github/settings.yml` (label catalog), `.github/labeler.yml` (signal→label mapping), `.github/workflows/pr-administration.yml` (labelling workflow). No Go package, no command (CI infrastructure, like 022/024).

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

**Total**: 5 dimensions checked, 5 passed, 3 validation scenarios satisfied, 0 findings.

---

## Driving Scenario Coverage

**Status**: Pass (8 of 8 behavioral scenarios covered)

Every driving scenario (concretized as the 8 non-`@validation` scenarios) traces to an identifiable code path across the three artifacts.

| Scenario | Status | Implementation |
|---|---|---|
| A feature pull request is labelled from its title | ✓ Covered | `labeler.yml` `features.title: ^feat(\(…\))?:` + workflow runs the labeler |
| A docs-only change is labelled from its changed files | ✓ Covered | `labeler.yml` `docs.files: .*\.md$`, `docs/.*`; the three semver labels carry **no** `files` matcher, so none is forced |
| A pull request carries every matching label | ✓ Covered | Per-label evaluation adds all matches (e.g. `features` + `dependencies`) |
| An unrecognized change is not mislabelled | ✓ Covered | No catch-all label; a no-match PR receives no managed label |
| Editing the title reconciles the labels | ✓ Covered † | `edited` trigger + per-label remove-on-no-match |
| A labelling failure does not block the merge | ✓ Covered | `continue-on-error: true` + workflow absent from branch protection |
| A fork pull request is labelled on the same terms | ✓ Covered | `pull_request_target` (base-context token) + no checkout |
| Rapid successive title edits reconcile to the latest title | ✓ Covered | `concurrency` group + `cancel-in-progress: true` + `edited` trigger |

† **Framework-behavior note** (not a finding): the sync scenarios (title-edit reconcile; multi-label removal) rest on `srvaroa/labeler@v1.14.0`'s per-label evaluate→add/remove default. This is the action's documented behavior and the basis the interface-spec records (Consistency Notes — "labeler removal default"); it is a reasonable framework assumption, confirmable only by a live run.

---

## Acceptance Criteria

**Status**: Pass (3 of 3 checked tasks, all criteria met)

| Task | Criteria | Evidence |
|---|---|---|
| T001 `settings.yml` | 7 labels (name/color/description), labels-only, names == labeler/030 contract, YAML valid, prerequisite documented | Labels-only (`labels:` is the sole top-level key); 7 names exact; each entry has name+color+description; Probot Settings-app prerequisite documented in header |
| T002 `labeler.yml` | feat→features / docs→docs / multi→multiple / none→none; sync removes stale; out-of-set untouched; schema-valid; names exact | `version: 1`, 7 entries with the interface-spec matchers; regexes spot-checked against scenario inputs; names parity with `settings.yml` confirmed |
| T003 workflow | 4 trigger types incl. forks; perms exactly `contents: read` + `pull-requests: write`; no secrets; no checkout; concurrency cancels superseded; non-blocking; actionlint clean | All invariants confirmed by inspection; `actionlint` clean; no `secrets.*`; single labeler step, no `actions/checkout` |

---

## Interface Contract Conformance

**Status**: Pass (3 of 3 surfaces conformant)

| Surface | Status | Note |
|---|---|---|
| `pr-administration.yml` structure | ✓ Conformant | `name`, `on.pull_request_target.types` (4, no `branches:`), `permissions`, `concurrency` group+cancel, `label` job (ubuntu-latest, single pinned step, no checkout, `GITHUB_TOKEN: ${{ github.token }}`, `continue-on-error`) — all match the interface table |
| `labeler.yml` mapping | ✓ Conformant | `version: 1` + all 7 `title`/`branch`/`files` matchers match the interface mapping table **byte-for-byte** (machine-diffed) |
| `settings.yml` catalog | ✓ Conformant | All 7 `name`/`color`/`description` rows match the interface catalog table **byte-for-byte** (machine-diffed) |

The interface-spec marked the action pin `[ASSUMED] v1.14.0 — pin at impl time`; the implementation pins `srvaroa/labeler@v1.14.0`, confirmed as the current latest release tag — a real, resolvable version, consistent with the repo's tag-pinning discipline (022/024).

---

## Non-Behavior Absence

**Status**: Pass (7 of 7 exclusions absent)

| Non-behavior (spec § Non-Behaviors) | Status | Evidence |
|---|---|---|
| Must not run lint/tests or report pass/fail | ✓ Absent | Workflow has a single labeler step — no test/lint job |
| Must not block/gate/fail a merge | ✓ Absent | Not in branch protection (only `ci-success` is required, per `scripts/setup-branch-protection.sh`); `continue-on-error: true` |
| Must not compute semver / draft notes / set release status | ✓ Absent | Only applies/removes labels; no version or notes logic |
| Must not build/package/publish/release | ✓ Absent | No build, package, or publish steps |
| Must not check out / run / execute PR code | ✓ Absent | No `actions/checkout`, no PR-supplied `run` step (ADR-5) |
| Must not touch labels outside the managed set | ✓ Absent | Labeler evaluates only the 7 labels named in `labeler.yml`; `settings.yml` is labels-only (no prune) |
| Labelling path must not own/maintain the label catalog | ✓ Absent ‡ | The workflow never creates label definitions; the catalog lives in `settings.yml` (Probot Settings app), decoupled from the labelling path |

‡ **Documented edge, not a finding**: the interface-spec's Error Communication table ("Missing label definition") accepts that if a managed label is not yet reconciled, the GitHub labels API creates it on first apply with a default color, which the Settings app then normalizes. This is an intrinsic API fallback, not the labelling path *owning* the catalog (source of truth, colors, and lifecycle remain in `settings.yml`/the app). The spec's behavioral guarantee — "labelling itself does not own or change [the catalog]" — holds. This was adjudicated at the interface layer; flagged here only for traceability.

---

## @wip Lifecycle Completion

**Status**: Pass

The 8 behavioral scenarios (referenced by checked tasks T002/T003) have had `@wip` removed. The 3 `@validation` scenarios correctly retain `@validation @wip` — T003 explicitly held them for this validate step; they are not referenced as implementable by any task. No `@wip` remains on a scenario that should have been implemented. (Validate is read-only on `.feature` files, so the `@validation @wip` tags are inspected, not removed.)

---

## Validation Scenario Results

**Status**: Satisfied (3 of 3 traced to implementation, independently of the driving-scenario pass)

| Scenario | Status | Trace |
|---|---|---|
| Reconciliation never touches labels outside the managed set | ✓ Satisfied | `srvaroa/labeler` evaluates/adds/removes only labels named in `labeler.yml` (the 7); `settings.yml` is labels-only so the catalog app never prunes. A hand-applied triage label outside the 7 is untouched by construction. |
| Labelling is not a required check | ✓ Satisfied | `pr-administration.yml` is absent from `scripts/setup-branch-protection.sh`; the sole required context declared there is `ci-success` (ruleset `require-ci-success`). `continue-on-error` further prevents a blocking red mark. |
| Fork labelling reads metadata only | ✓ Satisfied | No `actions/checkout` and no PR-supplied `run`; the single step is the labeler, which reads PR title/branch/changed-file list via the API and loads config from the base repo. Head code is never fetched or executed. |

---

## Verdict: Ready

All 5 conformance dimensions pass and all 3 held-out validation scenarios are satisfied. The implementation conforms to the specification: the seven-category managed label family, the dual title/branch + changed-files signals, authoritative sync scoped to the managed set, every-PR-including-forks labelling with no code execution, and the non-blocking posture are all present and contract-exact against the interface accord.

**Basis and caveat**: this is inspection-based conformance (the validate baseline), supported by machine diffs of the two config files against the interface tables, `actionlint` on the workflow, and `go test ./...` (no regression). Per the plan and tasks.md self-trigger caveat, `pull_request_target` runs the **base-branch** workflow, so the introducing PR will not exercise the labeller on itself — the runtime behaviors marked † (per-label sync) and the live fork run are confirmable only on the **first PR after merge to `main`**. This does not lower the verdict: conformance is established structurally; the live run is real-world confirmation of an already-conformant configuration.

---

## Next Steps

Implementation conforms to the specification. Suggest PR review and merge. After merge, confirm on the next PR that labels apply and reconcile as specified (the self-trigger caveat means this PR cannot self-verify). The specification loop for 028 is closed.
