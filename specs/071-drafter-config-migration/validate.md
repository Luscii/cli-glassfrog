# Validate: Drafter Config Migration

**Feature**: 071-drafter-config-migration
**Round**: 1 of 3
**Date**: 2026-08-03
**Verdict**: Ready
**Artifacts loaded**: spec.md, plan.md, tasks.md (5 of 5 tasks complete), interface-spec.md, features/no-automated-pipeline/drafter-config-contract.feature (13 scenarios), PROJECT.md
**Implementation files**: `.github/release-drafter.yml`, `.github/workflows/release-drafting.yml`, `internal/build/labelcontract.go` + `labelcontract_test.go`, `internal/build/drafterschema.go` + `drafterschema_test.go`, `internal/build/drafter_config_contract_bdd_test.go`, swept live-contract docs under `specs/030-release-drafting/`

---

## Conformance Summary

| Dimension | Status | Findings |
|---|---|---|
| Driving scenario coverage | ✓ Pass | 0 |
| Acceptance criteria | ✓ Pass | 0 |
| Interface contract conformance | ✓ Pass | 1 (P2 advisory) |
| Non-behavior absence | ✓ Pass | 0 |
| @wip lifecycle completion | ✓ Pass | 0 |
| **Validation scenarios** | ✓ Satisfied | 0 (one scenario partially inspectable pre-PR — see notes) |

**Total**: 5 dimensions checked, 5 passed, 1 advisory finding (P2)

Full quality gate at validation time: `gofmt -l .` empty, `go vet ./...` clean, `go test ./... -count=1` 2152 passed across 12 packages, `golangci-lint run` 0 issues.

---

## Driving Scenario Coverage

**Status**: Pass (7 of 7 spec driving scenarios covered)

Per spec § Non-Behaviors, drafting behavior is not verified at runtime; coverage for drafter-output scenarios is structural — the committed config declares the behavior and the guards pin the declaration. This is the spec's stated assurance boundary, not a gap.

| Scenario | Status | Implementation |
|---|---|---|
| a feature merge still bumps minor and files under Features | ✓ Covered (structural) | `version-resolver` category `minor`→`when.labels: [features]` + Features changelog category; declaration pinned by `CheckLabelContract` (buckets + category labels), exercised by `TestLabelContract_RealConfig` against the shipped file |
| a drafting run reports no schema deprecations | ✓ Covered (structural) | Config uses none of the warned-about forms — verified: no top-level `version-resolver`/`exclude-labels`, no category-level `labels`/`label`, no `title`/`collapse-after` on non-changelog entries, no `exclusive`, no `semver-increment` on the `pre-exclude` entry. First post-merge run is the observation point (spec: no acceptance gate on it) |
| the declared fallback supplies the patch bump | ✓ Covered | Condition-less `version-resolver` entry with `semver-increment: "patch"` present in the shipped config; `drafterFallbackIncrement` derives it and the guard fails on its absence as a missing declaration |
| a category losing its label predicate fails the guard | ✓ Covered (executable) | Drift case "a changelog category naming no label in its when" + executed BDD scenario; violation names file and missing label |
| removing the declared fallback fails the guard | ✓ Covered (executable) | Drift case "a deleted condition-less version-resolver fallback" + executed BDD scenario; message reads as an absent declaration |
| the pinned action version falling behind the config schema fails the guard | ✓ Covered (executable) | `CheckDrafterSchemaCoupling` below-floor drift case + executed BDD scenario; violation names the pinned ref, derived major, and required floor |
| exclusion survives the realignment | ✓ Covered (structural) | `pre-exclude` category `when.labels: [no-release-note]`; pinned by the guard's exclusion verdict against all three contract files |

---

## Acceptance Criteria

**Status**: Pass (all criteria met for all 5 checked tasks)

| Task | Status | Evidence |
|---|---|---|
| T001 — atomic migration | ✓ Met | Pin at `release-drafter/release-drafter@v7.7.0` (exact patch, major ≥ floor 7); pin comment keeps the empirical tag-loss record, removes the landed forward-pointer, and explicitly states the migration "is NOT known to fix" the tag loss; `dependabot.yml` untouched (verified: empty diff since base); config parsed structurally — 12 categories in the required groups (1 pre-exclude / 7 changelog without `type` / 3 conditioned version-resolver / 1 condition-less patch fallback), no forbidden fields anywhere; `tag-template`, `name-template`, `version-template`, `change-template`, `template`, `replacers`, and all seven titles byte-identical to the pre-migration file (diffed against git base 678c894); header comment names the new positions, keeps 030 ADR-2/3/4 citations, adds 071, states `exclusive` is deliberately omitted; `drafterCategoryLabels` gathers from changelog categories only; `ResolverDefault` keeps name and value with the 030 ADR-2 trace; `DrafterCategory.When` is `*DrafterWhen`; `VersionResolver`/`ResolverBucket` removed; `TestLabelContract_RealConfig` passes; drift table preserved with unchanged verdict wording (the exclusion message moved to the pre-exclude position, as the interface prescribes) |
| T002 — drift cases | ✓ Met | Seven additive cases: empty-`when` changelog category, deleted pre-exclude, wrong `semver-increment`, deleted fallback (asserted as absent declaration, with the redundancy rationale in the case comment), and the three superseded-shape forms asserted by schema name — the two pure-schema cases additionally assert the violation set contains no "missing" wording; harness uses substring matching over the joined violation set, no map-by-key lookups |
| T003 — coupling guard | ✓ Met | `DraftingWorkflowFileName` (distinct from 022's `WorkflowFileName`, contrast stated in its comment), `DrafterActionRepo`, `DrafterSchemaMinMajor = 7` with property-and-reason comment per the `GoReleaserVersion` precedent; loader reuses `RepoRoot`/`loadYAML`/`ReleaseDrafterConfig`/`Workflow` (the `json:"true"` quirk handled where it lives); check takes both inputs and derives the floor from the config's shape; `v7`/`v7.7`/`v7.7.0` all derive 7 (passing drift cases); underivable refs (SHA, branch, non-vN tag) and an absent drafter step each produce a named violation, never a pass; real-file test mirrors `TestLabelContract_RealConfig` |
| T004 — godog suite | ✓ Met | Suite runs `Tags: "~@wip"` with `Paths` naming only this feature file; exactly the five per-table scenarios cleared (verified: 8 `@wip` remain, 4 of them `@validation`); suite doc comment states the two hold reasons separately and records the drafter-output four as a boundary, not a backlog; no scenario step content modified (diff shows only 5 removed tag lines) |
| T005 — live-contract sweep | ✓ Met | 030's interface-spec.md/plan.md/validate.md describe the current positions with 071 pointers and retained 030 ADR citations; 030 spec.md verified tool-agnostic and unedited; analyze.md/risk.md name `version-resolver` only as surviving mechanism vocabulary; 030 tasks.md untouched (empty diff); directory-wide grep confirms every surviving `version-resolver`/`exclude-labels` occurrence is deliberate history or a rejection statement |

---

## Interface Contract Conformance

**Status**: Pass (all surfaces conformant; 1 advisory finding)

| Surface | Status | Notes |
|---|---|---|
| `.github/release-drafter.yml` target structure | ✓ Conformant | Twelve entries in the three groups with the specified required/forbidden fields (verified by parsing the shipped file); `exclusive` omitted everywhere; fallback entry carries no `when` key |
| `.github/workflows/release-drafting.yml` edit surface | ✓ Conformant | Exactly two changes — the pinned `uses:` ref and the rewritten comment; the rest of the workflow untouched |
| `internal/build/labelcontract.go` exported contract | ✓ Conformant | Unchanged names preserved; `CategoryTypeChangelog`/`CategoryTypePreExclude`/`CategoryTypeVersionResolver` as specified; reshaped `ReleaseDrafterConfig`/`DrafterCategory`/`DrafterWhen` match the field table including rejection-detector fields with do-not-clean-up comments; all five named derivation helpers present with the specified behavior |
| `internal/build/drafterschema.go` exported contract | ✓ Conformant | All five symbols with the specified values/signatures; pinned-major derivation follows the specified rules |
| Error communication | ✓ Conformant | All message classes verified against the table, including the schema-naming rejection (distinguishable from missing-label), the absent-declaration fallback wording, and the loader-level parse failure for list-form `when` (F-1 notes its test status) |

---

## Non-Behavior Absence

**Status**: Pass (all 8 non-behaviors absent)

| Non-behavior | Status | Evidence |
|---|---|---|
| No claim of fixing the untagged-release failure | ✓ Absent | Pin comment explicitly disclaims; commit messages disclaim; spec unchanged. The eventual PR description must carry the same restraint (see Validation Scenario Results) |
| Guard must not accept both shapes | ✓ Absent | Hard-switch: superseded positions parsed only into `Legacy*` rejection detectors, never read for contract values; presence fails by schema name |
| No reliance on the built-in fallback | ✓ Absent | Fallback declared in config; its deletion fails the guard (drift case verifies, with the action's identical built-in behavior noted as the reason only the guard can catch it) |
| No runtime verification of drafting | ✓ Absent | No synthetic-PR harness; the four drafter-output scenarios stay `@wip` as documentation-grade; no gate on the drafting run |
| No managed-label changes | ✓ Absent | `labeler.yml`/`settings.yml` untouched (empty diff since base); config label set == 028's managed set exactly (independently verified — nothing invented, nothing dropped) |
| No display-text changes | ✓ Absent | Templates, replacers, and all seven titles byte-identical (diffed against base) |
| No revisiting the dependency-automation policy | ✓ Absent | `dependabot.yml` untouched; the pin comment describes the bump as a hand bump within the standing policy |
| No rewriting 030's completed task records | ✓ Absent | `specs/030-release-drafting/tasks.md` has an empty diff since base |

---

## @wip Lifecycle Completion

**Status**: Pass

The five scenarios referenced as **execute** by checked task T004 have their `@wip` cleared; the eight held scenarios retain it (four `@validation`, four foreclosed drafter-output assertions). No other feature file was modified. The suite's `~@wip` filter makes the cleared tags load-bearing: 5 scenarios executed and passing in `TestDrafterConfigContractFeatures`.

---

## Validation Scenario Results

**Status**: Satisfied (4 of 4 traced independently; one has a surface that does not exist yet)

| Scenario | Status | Trace |
|---|---|---|
| The change claims no fix for the untagged-release failure | ✓ Satisfied (partially inspectable) | Spec § Non-Behaviors states it; the rewritten pin comment explicitly disclaims ("the schema migration is NOT known to fix it") while keeping the empirical record; all five commit messages avoid the claim. The scenario also names "the eventual pull-request description" — no PR exists at validation time, so that surface must honor the disclaimer when written |
| No label is invented or dropped by the realignment | ✓ Satisfied | Independently recomputed: the set of labels named anywhere in the realigned config equals 028's managed set in `settings.yml` and `labeler.yml` exactly (8 = 7 categories + exclusion; semver labels are reuses, not additions). The guard additionally pins this via the category/bucket/exclusion verdicts |
| The four label-contract assertions survive in number and strictness | ✓ Satisfied | All four verdicts present in `CheckLabelContract` with derivation moved and meaning intact: (a) three-file category-set agreement, (b) bucket sets + declared fallback, (c) exclusion presence in all three files, (d) managed-count of eight in both catalog files. Set-difference assertions preserved (missing fails as loudly as extra); the coupling verdict lives in a separate sibling function/file, additional to all four |
| Neither side of the coupling verdict is a hard-coded literal | ✓ Satisfied | Substitution probe executed at validation time: changing only the fixture's pinned ref flips the verdict (`v7.7.0`/`v8.0.1` pass, `v6.4.0` fails), and a superseded-shape config yields "cannot determine the schema floor" rather than a pass — both sides derived. `DrafterSchemaMinMajor` is the sanctioned checked-in contract fact (plan ADR-5): it maps schema generation → floor, cannot be derived from anything in the repository, and carries the property and reason in its comment |

---

## Findings

### F-1: ADR-7's list-form `when` rejection is verified but not pinned by a committed test

- **Dimension**: Interface contract conformance
- **Severity**: P2 (advisory)
- **Source**: interface-spec.md (071) § Error Communication — "`when` is written in the list form (ADR-7) | A **parse** error from `LoadLabelContract` … noted so a test asserts against the loader's error, not the violation slice"
- **Implementation**: `internal/build/labelcontract.go` — `DrafterWhen` is a struct type, so a list-form `when` fails `sigs.k8s.io/yaml` unmarshalling loudly (empirically confirmed at validation time with a throwaway probe: the parse errors as specified)
- **Gap**: The behavior exists and matches the contract, but no committed test asserts it, so a future widening of `DrafterWhen` (e.g. a tolerant unmarshaller) would not redden anything. tasks.md never required this test, so this is an advisory coverage note, not an acceptance failure. A one-case addition to `TestLabelContract_Drift`-adjacent parsing tests would close it.
- **Resolution** (PR #184 review round 1): closed — `TestLabelContract_ListFormWhenFailsParse` now pins the parse rejection. The same round also strengthened the ADR-4 detectors to fire on key presence (`json.RawMessage`), covering `exclude-labels: []`, bare-null keys, and empty category shorthands.

---

## Verdict: Ready

All 5 conformance dimensions pass. All 4 validation scenarios satisfied through independent inspection, including an executed substitution probe for the coupling verdict. The single finding is a P2 test-coverage advisory on a behavior that was empirically confirmed to work as specified. The implementation conforms to its specification.

Two forward notes ride with the verdict: (1) the eventual PR description must not claim or imply a fix for the untagged-release failure — the held-out scenario names that surface, and it does not exist yet; (2) per the repository's recorded hand-bump procedure, the first post-merge drafting run and a throwaway pre-release are the post-merge observation points; the spec deliberately hangs no acceptance gate on them.

---

## Next Steps

Implementation conforms to the specification. Suggest PR review and merge. Optionally close F-1 (one small parse-rejection test) before opening the PR — self-flagged advisory findings left open tend to be re-raised in review.
