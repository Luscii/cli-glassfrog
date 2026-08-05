# Interface Accord: Release Drafting — Specification

**Feature**: 030-release-drafting
**Role**: Crafter
**Touchpoint**: Specification
**Plan reference**: System Architecture (the `release-drafting.yml` draft job + `.github/release-drafter.yml` + the eighth label in `.github/labeler.yml`/`.github/settings.yml` + the `internal/build` config guard); ADR-1 (dedicated workflow, pinned release-drafter, `push:main`, separate from 029), ADR-2 (`version-resolver`, default patch, v0.1.0 first-release), ADR-3 (seven 028 categories → note sections), ADR-4 (eighth exclusion label via `negate`-over-noteworthy), ADR-5 (auto pre-release/latest from version), ADR-6 (`internal/build` label-contract guard)

---

## Surface

This feature is committed declarative config (a workflow + a release-drafter config) plus one coordinated label addition to 028's catalog/labeler and one Go config-guard test. **No runtime command is added to the CLI.** Its output surface is the **draft GitHub Release** that release-drafter maintains.

### Invocation

| Entry point | Trigger | Notes |
|---|---|---|
| `.github/workflows/release-drafting.yml` | GitHub event `push` to `main` | The single automated trigger. Every merge to `main` regenerates the draft. Not a required check — runs after the merge, gates nothing (spec §Non-Behaviors). |
| `.github/release-drafter.yml` | Read by the release-drafter action each run | Declarative config: categories (note sections, exclusion, and semver resolution — schema positions per 071) and templates. |
| Draft GitHub Release | Maintained by the action; **published by a maintainer by hand** | The output. Publishing it creates the tag and triggers 022's `release.yml`. 030 never publishes (spec §Stops at the draft). |

### `.github/workflows/release-drafting.yml` structure

| Element | Contract |
|---|---|
| `name` | `Release Drafting` |
| `on` | `push: { branches: [main] }` — no tags (022 owns tags; a tag push must not draft). |
| `permissions` | `contents: write` (maintain the draft release) + `pull-requests: read` (read merged-PR labels/titles). No `id-token`, no secrets beyond `GITHUB_TOKEN`. |
| `concurrency` | `group: ${{ github.workflow }}-${{ github.ref }}`, `cancel-in-progress: true` — a newer merge supersedes an in-flight draft run (latest tip wins). |
| Job `draft` | `runs-on: ubuntu-latest`. Step 1: pinned release-drafter (id `draft`, reads `.github/release-drafter.yml`). Step 2: set pre-release/latest from `steps.draft.outputs.major_version` (ADR-5). **No `actions/checkout` of any PR head** — release-drafter reads merged-PR metadata via the API; the workflow runs on trusted `main`. |

```yaml
name: Release Drafting

on:
  push:
    branches: [main]   # NOT tags — 022 owns the published-tag trigger.

# Least privilege: write the draft release + read merged-PR labels/titles.
permissions:
  contents: write
  pull-requests: read

# Latest tip wins: a newer merge supersedes an in-flight draft run.
concurrency:
  group: ${{ github.workflow }}-${{ github.ref }}
  cancel-in-progress: true

jobs:
  draft:
    runs-on: ubuntu-latest
    steps:
      - name: Draft the next release
        id: draft
        uses: release-drafter/release-drafter@v7.7.0   # exact patch; the major floor is guarded (071)
        with:
          config-name: release-drafter.yml
        env:
          GITHUB_TOKEN: ${{ github.token }}

      # ADR-5: status follows the version — pre-release while 0.x, latest at >=1.0.0.
      - name: Set pre-release/latest from version
        env:
          GH_TOKEN: ${{ github.token }}
        # [ASSUMED] exact gh invocation — verify draft is addressable by tag at impl time
        run: |
          if [ "${{ steps.draft.outputs.major_version }}" = "0" ]; then
            gh release edit "${{ steps.draft.outputs.tag_name }}" --draft --prerelease
          else
            gh release edit "${{ steps.draft.outputs.tag_name }}" --draft --latest --prerelease=false
          fi
```

### `.github/release-drafter.yml` structure

> **Schema positions per 071 (Drafter Config Migration).** The invariants below are still this accord's (030 ADR-2/ADR-3/ADR-4); 071 moved where they are declared: everything now lives under `categories` — the note sections carry their label in `when.labels`, the exclusion is a `pre-exclude` category, and the semver buckets plus the declared patch fallback are `version-resolver` categories. The superseded top-level `version-resolver`/`exclude-labels` keys no longer exist and are rejected by the guard by name.

| Element | Contract |
|---|---|
| `tag-template` / `name-template` | `v$RESOLVED_VERSION` — v-prefixed, consistent with 023's tag form that 022/`go install` read. |
| `version-template` | `$MAJOR.$MINOR.$PATCH`. |
| `categories` — note sections | The seven 028 labels → titled note sections, each label in the category's `when.labels` (ADR-3; positions per 071), in order Breaking → Internal. |
| `categories` — `version-resolver` entries | Three conditioned buckets (`major`→`[breaking]`, `minor`→`[features]`, `patch`→`[fixes]`) plus one condition-less entry with `semver-increment: "patch"` — the declared fallback. release-drafter resolves highest-wins natively (ADR-2; positions per 071). |
| `categories` — `pre-exclude` entry | `when.labels: [no-release-note]` — PRs carrying it leave the notes and the counted bump set (ADR-4; position per 071). |
| `change-template` | `- $TITLE (#$NUMBER) @$AUTHOR` — one note line per included PR (its title). |
| `template` | Release body containing `$CHANGES` (the categorized sections). |
| `prerelease` | Left at the release-drafter default (`false`); the workflow's post-step is authoritative for status (ADR-5). |
| `exclusive` | Deliberately omitted everywhere: the default (`false`) preserves the behaviour that a PR appears under every matching note section (071). |

```yaml
tag-template: "v$RESOLVED_VERSION"
name-template: "v$RESOLVED_VERSION"
version-template: "$MAJOR.$MINOR.$PATCH"

categories:
  # ADR-4: exclusion — auto-applied to spec/feature-only PRs (see labeler.yml).
  - type: "pre-exclude"
    when:
      labels: [no-release-note]

  # ADR-3: the seven 028 labels, exact strings.
  - title: "⚠ Breaking Changes"
    when:
      labels: [breaking]
  - title: "Features"
    when:
      labels: [features]
  - title: "Fixes"
    when:
      labels: [fixes]
  - title: "Documentation"
    when:
      labels: [docs]
  - title: "Infrastructure"
    when:
      labels: [infrastructure]
  - title: "Dependencies"
    when:
      labels: [dependencies]
  - title: "Internal"
    when:
      labels: [internal]

  # ADR-2: highest-wins semver resolution; the condition-less entry is the
  # declared patch fallback.
  - type: "version-resolver"
    semver-increment: "major"
    when:
      labels: [breaking]
  - type: "version-resolver"
    semver-increment: "minor"
    when:
      labels: [features]
  - type: "version-resolver"
    semver-increment: "patch"
    when:
      labels: [fixes]
  - type: "version-resolver"
    semver-increment: "patch"

change-template: "- $TITLE (#$NUMBER) @$AUTHOR"
template: |
  ## What's changed

  $CHANGES
```

Section titles (display text like "⚠ Breaking Changes") are `[ASSUMED]` and tunable; the **label strings** (`breaking`/`features`/`fixes`/`docs`/`infrastructure`/`dependencies`/`internal`) are the fixed 028 contract.

### Eighth managed label — `no-release-note` (ADR-4, coordinated 028 change)

A new managed label joins 028's catalog and labeler. release-drafter excludes purely by label, and `srvaroa/labeler`'s `files:` matcher is *any-file-matches* with no native confined-to mode (verified) — so "confined to `specs/`+`.feature`" is expressed as its complement with `negate: true`: the label is applied **iff no changed file is noteworthy**. "Noteworthy" is exactly the developer's rule — code, real docs (`docs/` or `README.md`), infra, deps, or the vendored API spec.

Added to `.github/labeler.yml` (the existing seven entries are untouched):

```yaml
  # no-release-note (030 ADR-4): applied iff NO changed file is noteworthy
  # (negate inverts the block). Noteworthy = code / real docs / infra / deps /
  # vendored API spec. A PR confined to specs/*.md and *.feature matches none,
  # so it gets this label and release-drafter excludes it. Patterns are
  # [ASSUMED]/tunable; the contract is "no noteworthy file → excluded".
  # NOTE: real-docs patterns are docs/ + README.md ONLY — NOT a broad .*\.md$,
  # which would (wrongly) treat Score spec markdown under specs/ as noteworthy.
  - label: no-release-note
    negate: true
    files:
      - ".*\\.go$"
      - "go\\.mod$"
      - "go\\.sum$"
      - "docs/.*"
      - "README\\.md$"
      - "\\.github/.*"
      - "\\.goreleaser\\.ya?ml$"
      - "\\.golangci\\.ya?ml$"
      - "scripts/.*"
      - "Makefile"
      - "Dockerfile.*"
      - "\\.pre-commit-config\\.yaml"
      - "spec/.*"
```

Added to `.github/settings.yml` `labels:` block (eighth entry; color `[ASSUMED]`):

```yaml
  - name: no-release-note
    color: "EDEDED"
    description: "Auto-excluded from release notes (changes confined to specs/ and .feature files)"
```

### Structural contract — `internal/build` label-contract config guard (ADR-6)

A Go test (joining the package's existing `.goreleaser`/`release.yml` guards; parses YAML via `sigs.k8s.io/yaml`, change-detector rigor) asserts the three config files agree:

| Assertion | Fails when |
|---|---|
| release-drafter changelog categories' `when` labels == the seven category labels in `labeler.yml` **and** `settings.yml` (positions per 071) | a category label is renamed/dropped/added in one file but not the others |
| the `version-resolver` categories' `major`/`minor`/`patch` buckets == exactly `breaking`/`features`/`fixes`, and the condition-less entry declares `semver-increment: patch` (positions per 071) | the semver buckets drift from the 028 semver-bearing labels, or the declared fallback is removed or drifts off patch |
| `no-release-note` present in `settings.yml` **and** `labeler.yml` **and** release-drafter's `pre-exclude` category (position per 071) | the exclusion label exists in one place but not the others |
| total managed set == eight (seven categories + exclusion) across `labeler.yml`/`settings.yml` | a managed label is added or removed without updating the contract |
| the config is on the current schema — superseded `version-resolver`/`exclude-labels` keys and category-level `labels` shorthands are rejected by name (071 ADR-4) | a config still expresses the contract at the superseded positions |
| the drafting workflow's pinned action major ≥ the floor the config schema requires (071's `internal/build/drafterschema.go` coupling guard) | the pinned major falls behind the schema, or the pinned ref carries no derivable major |

The guard runs inside the existing `go test ./...` matrix (024 pre-merge, 029 post-merge), so a desync reddens CI rather than silently mis-drafting. Exact Go symbol names / file constants are implementation-level.

---

## Interactions

**Drafting flow (end-to-end):** a PR merges to `main` → `push:main` starts `release-drafting.yml` → release-drafter reads every PR merged since the last *published* release and their 028-applied labels → drops PRs carrying `no-release-note` (the `pre-exclude` category) → resolves the bump highest-wins against the last published tag (declared patch fallback; no prior release ⇒ 0.0.0 base) → groups the remaining PR titles into the seven category sections (empty sections omitted) → writes/updates the single draft release with tag `v$RESOLVED_VERSION` → the post-step marks the draft pre-release (`major_version == 0`) or latest (`>= 1`) → the draft sits unpublished for a maintainer.

**Authoritative reconciliation:** release-drafter maintains one draft keyed to the unpublished release it owns; each run regenerates it, so re-runs converge rather than appending (spec edge-case "reconciliation converges"). `cancel-in-progress` ensures the final draft reflects the latest `main` tip.

**Hand-off to 022:** when a maintainer publishes the draft, the tag is created and `release.yml` (022, on `release: published`) builds and attaches the binaries+checksums. 022 honors the draft's notes and pre-release/latest status verbatim — it adds artifacts, never prose or status (022 ADR / DECISIONS). 030 owns notes/version/status; 022 owns binaries.

**Contract consumed from 028:** the eight label strings. 028 produces/syncs them (incl. the new `no-release-note` via the negate rule); 030 reads `breaking`/`features`/`fixes` for the bump, the four non-semver labels for categorization, and `no-release-note` for exclusion.

---

## Error Communication

| Condition | Behavior |
|---|---|
| No semver-bearing label across included PRs | The condition-less `version-resolver` category's `semver-increment: patch` applies (spec "default patch"; position per 071). Not an error. |
| Multiple semver-bearing labels across included PRs | Highest wins (breaking > features > fixes) — release-drafter native (ADR-2). Not an error. |
| PR carries `no-release-note` (and possibly other labels too, e.g. `docs`) | The `pre-exclude` category removes it from the notes **and** the counted bump set before categorization — exclusion wins over any category label it also holds. Not an error. |
| Spec/feature-only PR (confined to `specs/`+`.feature`) | The labeler's negate rule applies `no-release-note` (no noteworthy file changed) → excluded. Not an error. |
| First release, no published baseline | release-drafter computes from a `0.0.0` base; a feature/breaking first release → `v0.1.0`+ (spec floor). A *patch-only* first release computes `v0.0.1` (below the floor — see Consistency Notes / plan Risk). |
| Drafting run fails (action error, API hiccup) | The run goes red but blocks nothing — the workflow is not a required check and the merge already happened (spec §Non-Blocking). The previous draft is left intact; the next merge re-reconciles. |
| Draft never published | The draft has no git tag; 022 does not run until a maintainer publishes (spec §Stops at the draft). |

---

## Consistency Notes

- **Sibling boundary (028 PR Administration):** this accord is the **consumer** of the label contract 028 defines. The seven category labels are referenced by their exact 028 strings; the eighth (`no-release-note`) is *added* to 028's catalog/labeler here (ADR-4) — a coordinated change that widens 028's recorded "EXACTLY seven managed labels" to eight (announced divergence; recorded in DECISIONS, candidate for `/score:deprecate`). The `internal/build` config guard (ADR-6) closes 028's standing note that the label contract was "un-guarded until 030 ships."
- **Sibling boundary (022 Automated Release Pipeline):** clean seam — 030 maintains the draft (notes/version/status), 022 (`release.yml`, on `release: published`) honors them and attaches binaries. 022's interface already records that it "does not author or curate the release notes, nor maintain the draft" and "does not decide pre-release/latest status." The two never both write the same field.
- **Sibling boundary (029 Main-Branch Verification):** also on `push:main`, but kept a **separate** workflow (ADR-1) because 029 is `contents: read`/tests-only and 030 needs `contents: write`. Mirrors 028's split from 024 over divergent permissions.
- **Observed interaction — spec markdown also matches 028's `docs` label:** 028's `docs` matcher (`.*\.md$`) tags Score spec markdown under `specs/` as `docs`. That is harmless for release notes because `no-release-note` excludes the PR regardless of any category label it also carries (exclusion precedence). 028's `docs` matcher is left untouched (tightening it is out of 030's scope); the slight triage-label noise is a 028 concern, not a release-note one.
- **Convention — pinned action versions:** `release-drafter/release-drafter` is pinned to a concrete tag (022/024/028 "pin the toolchain, don't float `latest`"). Both developer-reference repos (ailign-cli, terraform-vitals-template) use this labeler + release-drafter shape.
- **Assumption — auto pre-release via `gh release edit` on a draft (`[ASSUMED]`):** ADR-5's post-step assumes a draft release is addressable by its `tag_name` and its pre-release flag is editable pre-publish. If not, the fallback is release-drafter's static `prerelease: true` with a documented flip at the 1.0.0 cut — still correct for the project's current 0.x phase (plan Risk).
- **Assumption — first-release v0.1.0 floor (`[ASSUMED]`):** native release-drafter from a 0.0.0 base yields the floor for any feature/breaking first release; a patch-only first release would compute `v0.0.1`. If the literal floor must hold in every case, pin it via an explicit initial-version config or a seed tag at implement (plan Risk).
- **Assumption — matcher patterns and label color (`[ASSUMED]`):** the noteworthy path regexes and the `no-release-note` color are tunable without changing the contract. The load-bearing contract is fixed: eight managed labels, highest-wins bump with patch default, seven category sections, exclusion by label, status derived from version, never a merge gate, never published automatically.
- **CONSTITUTION XII note:** the release-drafter action and `gh` are CI-host tools, not artifact dependencies — XII governs the produced binary's runtime, the same standing the project gives GoReleaser, golangci-lint, and srvaroa/labeler.
