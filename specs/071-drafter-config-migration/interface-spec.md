# Interface Accord: Drafter Config Migration — Specification

**Feature**: 071-drafter-config-migration
**Role**: Crafter
**Touchpoint**: Specification
**Plan reference**: System Architecture (the restructured `.github/release-drafter.yml` + the reshaped `internal/build/labelcontract.go` + the new `internal/build/drafterschema.go`); ADR-1 (hard-switch, no dual-shape parser), ADR-2 (condition-less `version-resolver` category as the declared patch fallback), ADR-3 (coupling verdict in a sibling file sharing the parse), ADR-4 (superseded shape rejected explicitly), ADR-5 (pinned major derived, schema floor a named constant), ADR-6 (`exclusive` left at default), ADR-7 (`when` mapping form only)

---

## Surface

This feature is committed declarative config plus two Go guard files in the test-only `internal/build` package. **No runtime command is added to the CLI, and no drafting behavior changes.** Every contract below is a *position* change under an unchanged invariant.

### Invocation

| Entry point | Trigger | Notes |
|---|---|---|
| `.github/release-drafter.yml` | Read by the release-drafter action on each drafting run | The migrated file. Structure below. |
| `.github/workflows/release-drafting.yml` | GitHub `push` to `main` | **Edited** by this feature in exactly two places — the drafting step's pinned `uses:` ref moves to a major at or above the schema floor, and the comment above it is rewritten. The same pinned ref then becomes a **guard input**. Nothing else in the workflow changes. |
| `go test ./internal/build/...` | 024 pre-merge CI, 029 post-merge | Runs both guards. The only pre-merge assurance this feature carries. |

### `.github/release-drafter.yml` — target structure

Top-level keys. Everything not listed as *changed* is byte-identical to the current file.

| Key | State | Contract |
|---|---|---|
| `tag-template` | unchanged | `"v$RESOLVED_VERSION"` |
| `name-template` | unchanged | `"v$RESOLVED_VERSION"` |
| `version-template` | unchanged | `"$MAJOR.$MINOR.$PATCH"` |
| `version-resolver` | **removed** | Absorbed into `categories` (four entries below). Its presence is a guard violation (ADR-4). |
| `exclude-labels` | **removed** | Absorbed into a `pre-exclude` category. Its presence is a guard violation (ADR-4). |
| `categories` | **restructured** | One list carrying three entry kinds. Detail below. |
| `change-template` | unchanged | `"- $TITLE (#$NUMBER) @$AUTHOR"` |
| `template` | unchanged | The `## What's changed` / `$CHANGES` body |
| `replacers` | unchanged | The conventional-commit prefix stripper |

**`categories` entry kinds.** Twelve entries in three groups, written in this order for readability. Only the seven changelog entries' *relative* order is display-significant (it sets note section order); the other groups are position-independent.

| Group | Count | `type` | Required fields | Forbidden fields |
|---|---|---|---|---|
| Exclusion | 1 | `"pre-exclude"` | `when.labels: [no-release-note]` | `title`, `semver-increment`, `exclusive`, `collapse-after` |
| Note sections | 7 | *omitted* (defaults to `changelog`) | `title`, `when.labels: [<one category label>]` | `labels`, `label` (superseded shorthands) |
| Semver resolution | 3 | `"version-resolver"` | `semver-increment` ∈ {`major`,`minor`,`patch`}, `when.labels: [<one semver label>]` | `title`, `collapse-after` |
| Declared fallback | 1 | `"version-resolver"` | `semver-increment: "patch"`, **no `when` key at all** | `title`, `collapse-after`, `when` |

The forbidden-field lists are not stylistic. The action emits a warning for a `title` or a non-default `collapse-after` on a non-changelog category, and for a `semver-increment` other than `patch` on a `pre-exclude` category — any of which breaks the zero-deprecation-warnings accord. `exclusive` on a `pre-exclude` category raises a hard error.

`exclusive` is **omitted everywhere**. Its default (`false`) is the behavior-preserving value: a pull request lands in every matching note section, as it does today (plan ADR-6).

**Example** — the fallback entry, whose absence of `when` is the whole contract:

```yaml
  - type: "version-resolver"
    semver-increment: "patch"   # 030 ADR-2: the bump when no semver-bearing
                                # label is present. No `when` — a condition-less
                                # version-resolver entry IS the fallback.
```

### `internal/build/labelcontract.go` — exported contract

**Unchanged, in name and meaning**: `ReleaseDrafterFileName`, `LabelerFileName`, `SettingsFileName`, `CategoryLabels`, `SemverBuckets`, `NoReleaseNoteLabel`, `ManagedLabelCount`, `LoadLabelContract`, `CheckLabelContract`.

`ResolverDefault` keeps its name and its value (`"patch"`). Its *subject* moves: it now pins the `semver-increment` of the condition-less `version-resolver` category rather than a `version-resolver.default` key. The name encodes the property, not the position — its doc comment must say so, or the next migration will rename it and lose the trace to 030 ADR-2.

**New category-type vocabulary** (the strings the parse branches on):

| Constant | Value |
|---|---|
| `CategoryTypeChangelog` | `"changelog"` |
| `CategoryTypePreExclude` | `"pre-exclude"` |
| `CategoryTypeVersionResolver` | `"version-resolver"` |

**Reshaped types**:

| Type | Field | Type | Contract |
|---|---|---|---|
| `ReleaseDrafterConfig` | `Categories` | `[]DrafterCategory` | The single contract source. |
| | `LegacyVersionResolver` | pointer/map, `json:"version-resolver"` | **Rejection detector only** (ADR-4). Never read for contract purposes. Non-nil ⇒ violation. |
| | `LegacyExcludeLabels` | `[]string`, `json:"exclude-labels"` | Rejection detector only. Non-empty ⇒ violation. |
| `DrafterCategory` | `Title` | `string` | Display text. Not asserted. |
| | `Type` | `string` | Empty string means `changelog` — the action's own default. |
| | `SemverIncrement` | `string` | Read only for `version-resolver` entries. |
| | `When` | `*DrafterWhen` | `nil` distinguishes "no condition" (the fallback) from "empty condition". Pointer, not value — a value type cannot express the difference. |
| | `LegacyLabels` / `LegacyLabel` | `[]string` / `string` | Rejection detectors for the superseded category shorthands. |
| `DrafterWhen` | `Labels` | `[]string` | The primary form. |
| | `Label` | `string` | The singular shorthand; folded in with `Labels`. |

`VersionResolver` and `ResolverBucket` are **removed as contract-source types** — nothing reads a top-level `version-resolver` block for assertions any more.

**Derivation helpers** (unexported; named here because the drift cases target their behavior):

| Helper | Derives |
|---|---|
| `drafterCategoryLabels` | Labels from **changelog categories only** — `when.labels` + `when.label`. Version-resolver and pre-exclude entries are excluded, or the semver labels would triple-count and `no-release-note` would read as an unexpected category label. |
| `drafterSemverBuckets` | `map[increment][]label` from `version-resolver` categories **that have a `when`**. |
| `drafterFallbackIncrement` | The `semver-increment` of the `version-resolver` category with **no `when`**; empty string when absent. |
| `drafterExcludedLabels` | Labels from `pre-exclude` categories' `when`. |
| `drafterLegacyShape` | The ADR-4 rejections. |

### `internal/build/drafterschema.go` — exported contract

| Symbol | Kind | Contract |
|---|---|---|
| `DraftingWorkflowFileName` | `const string` | `".github/workflows/release-drafting.yml"`. Distinct from `WorkflowFileName` (022's `release.yml`) — the two must not be confused. |
| `DrafterActionRepo` | `const string` | `"release-drafter/release-drafter"`. The step is located by this `uses:` prefix. |
| `DrafterSchemaMinMajor` | `const int` | `7`. The checked-in contract fact: the lowest action major that understands the config schema this feature adopts. Its comment carries the property and the reason, following `GoReleaserVersion`'s precedent in `workflow.go`. |
| `LoadDrafterSchemaCoupling` | `func() (ReleaseDrafterConfig, Workflow, error)` | Reads the drafter config and the drafting workflow from the repository root. Reuses `RepoRoot()`, `loadYAML`, and `workflow.go`'s `Workflow` type. |
| `CheckDrafterSchemaCoupling` | `func(ReleaseDrafterConfig, Workflow) []string` | Returns violations; empty means the pinned major satisfies the schema's floor. |

Takes **both** inputs deliberately: the required floor is derived from the config's own shape, not assumed. A config on the superseded schema yields "cannot determine the required floor" rather than a silent pass.

---

## Interactions

**Drafting run** (unchanged in outcome): merge to `main` → the action reads `.github/release-drafter.yml` → pre-exclude categories filter the merged pull requests → changelog categories route each surviving pull request into every matching note section → version-resolver categories resolve the bump highest-wins, falling back to the condition-less entry → the draft is reconciled.

**Guard run**: `go test ./internal/build/...` → `LoadLabelContract` parses the three config files and `CheckLabelContract` returns the four label verdicts → `LoadDrafterSchemaCoupling` parses the config and the drafting workflow and `CheckDrafterSchemaCoupling` returns the coupling verdict. Both are pure functions over parsed structures; both are exercised twice — once against the real committed files, once against in-memory fixtures mutated one thing at a time.

**Pinned-major derivation**: locate the first step whose `Uses` begins with `DrafterActionRepo + "@"` → take the ref after `@` → require a leading `v` followed by digits → parse those digits as the major. `v7`, `v7.7`, and `v7.7.0` all yield `7`. A branch name, a bare tag without a leading `vN`, or a commit SHA yields **no major and a violation** — "cannot determine the pinned major" is a finding, never a pass.

**Instructional surface.** `.github/release-drafter.yml`'s header comment is part of the contract, not decoration: it is where the label strings are named as the cross-feature contract with `settings.yml`/`labeler.yml`, and where the guard is pointed at. It must be rewritten alongside the structure — its current text names `version-resolver`, `categories`, and `exclude-labels` at their superseded positions and cites "030 ADR-2/ADR-3/ADR-4" only. The rewritten comment names the new positions, keeps the 030 ADR citations (the invariants are still 030's), and adds the 071 citation for the schema. It must also state that `exclusive` is deliberately omitted, so nobody adds it later believing it preserves behavior.

---

## Error Communication

Violations are strings, one per problem, each naming the file, the section, and the offending value — the convention 030 ADR-6 established. Message wording for the four pre-existing verdicts is **unchanged**; only their derivation moved.

| Condition | Reported as |
|---|---|
| A changelog category's `when` names a label outside the seven | `unexpected release-drafter.yml category label declared: "<label>"` |
| A category label is absent from every changelog category's `when` | `required release-drafter.yml category label missing: "<label>"` |
| A `version-resolver` bucket drifts off its single semver label | `unexpected/required version-resolver <increment> label ...` |
| The condition-less `version-resolver` category is absent or carries the wrong increment | Names `version-resolver default`, the expected `"patch"`, and what was found — including the absent case, which must read as a missing declaration, not as an empty string mismatch |
| `no-release-note` is absent from the `pre-exclude` category | Names the `pre-exclude` category and the label (replaces the `exclude-labels must contain` wording, which no longer describes a real position) |
| The config still carries `version-resolver`, `exclude-labels`, or category-level `labels`/`label` | Names the **schema**, not the symptom: the config is on the superseded schema and must be migrated. This message must not be mistakable for "a label is missing." |
| The pinned major is below `DrafterSchemaMinMajor` | Names the workflow file, the pinned ref, the derived major, and the required floor |
| The pinned ref yields no major (branch, SHA, non-`vN` tag) | Names the workflow file and the ref, and states that the major could not be determined |
| No step in the workflow uses the drafter action | A violation — the guard's input is missing, which is not a pass |
| `when` is written in the list form (ADR-7) | A **parse** error from `LoadLabelContract`, not a violation string: `parsing .github/release-drafter.yml: ...`. Loud and file-named, but shaped differently from the other failures — noted so a test asserts against the loader's error, not the violation slice |
| A config file is missing or unparseable | Unchanged: `reading/parsing <file>: <err>` from `loadYAML` |

**Degradation**: none. Every condition above fails; no optional input is defaulted or skipped. The guards have no partial-success mode.

---

## Consistency Notes

**No sibling interface files.** This feature has one touchpoint. There is no `accords/` directory in this repository, so no cross-accord pattern check applies.

**Relationship to 030's interface accord.** `specs/030-release-drafting/interface-spec.md` documents the config at its superseded positions and is a live-contract document — it is updated by this feature's Phase 3, not superseded by this file. Where the two overlap, 030 owns *what* the invariants are (seven categories, highest-wins, patch fallback, the exclusion label) and this accord owns *where they now live*.

**Conforms to the `internal/build` guard family.** Same `sigs.k8s.io/yaml` parse path, same change-detector rigor (a missing value fails as loudly as an extra one), same real-file-plus-fixture test pairing, same "name the file and the value" message convention. The one new shape is the anchor type: cross-file agreement between a workflow's pinned dependency version and a committed config's schema generation.

**A godog suite sits above both guards** (plan ADR-8), driving this feature's acceptance scenarios through the same exported functions with a `~@wip` filter — the pattern `release_bdd_test.go` already uses in this package. It defines no surface of its own and adds no assurance beyond the guards; it exists so the scenarios and the verdicts cannot drift apart unnoticed. Five scenarios execute; eight are held, four for `/score:validate` and four because they assert drafter output this feature does not verify. Nothing in the contracts above changes because of it.

**One deliberate deviation from the package's own idiom.** `workflow.go` provides `StringOrSlice` for GitHub keys genuinely written both ways in this repository. `DrafterWhen` does *not* follow it (ADR-7): nothing here writes the list form, so a tolerant unmarshaller would ship an untested arm. The consequence — a list-form config fails at parse rather than as a violation — is recorded in Error Communication rather than smoothed over.
