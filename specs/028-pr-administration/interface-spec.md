# Interface Accord: PR Administration — Specification

**Feature**: 028-pr-administration
**Role**: Crafter
**Touchpoint**: Specification
**Plan reference**: System Architecture (the `pr-administration.yml` label job + `.github/labeler.yml` + `.github/settings.yml` labels block); ADR-1 (`pull_request_target`, separate from `ci.yml`), ADR-2 (pinned `srvaroa/labeler` + committed config), ADR-3 (seven-category label family, semver-bearing), ADR-4 (`settings.yml` label catalog via Probot Settings app), ADR-5 (no checkout of PR head)

---

## Surface

This feature is three committed declarative artifacts. No runtime command is added to the CLI.

### Invocation

| Entry point | Trigger | Notes |
|---|---|---|
| `.github/workflows/pr-administration.yml` | GitHub event `pull_request_target`, activity types `opened`, `reopened`, `synchronize`, `edited` | The single automated trigger. `synchronize` covers pushes to the PR head (changed-files signal); `edited` covers title/branch edits (the semver-bearing signal). `pull_request_target` runs in the **base-repo** context, so the token can carry `pull-requests: write` and **fork PRs are labelled** (spec C1). No `branches:` filter — labelling applies to every PR regardless of base. |
| `.github/settings.yml` (`labels:` block) | Reconciled by the Probot "Settings" GitHub App on push to the default branch | Declarative source of truth for the seven managed label definitions. Created/updated as config-as-code, reviewed in PRs, and continuously reconciled (a renamed/deleted managed label is restored). One-time prerequisite: the Settings app installed on the repo/org. |

### `.github/workflows/pr-administration.yml` structure

| Element | Contract |
|---|---|
| `name` | `PR Administration` |
| `on` | `pull_request_target: { types: [opened, reopened, synchronize, edited] }` — no `branches:` filter. |
| `permissions` | `contents: read` + `pull-requests: write` — the minimum to read PR metadata and write labels. No `id-token`, no `contents: write`, no secrets exposed to the action. |
| `concurrency` | `group: ${{ github.workflow }}-${{ github.event.pull_request.number }}`, `cancel-in-progress: true` — a new event cancels the superseded run, so labels reconcile to the latest PR state (B1). |
| Job `label` | `runs-on: ubuntu-latest`. A single step running the pinned labeler action. **No `actions/checkout`** — the labeler reads PR title, head branch, and the changed-file list via the API; PR head code is never fetched or executed (ADR-5). |

```yaml
name: PR Administration

on:
  pull_request_target:
    types: [opened, reopened, synchronize, edited]

# Least privilege: read repo metadata + write PR labels. Nothing else.
# pull_request_target gives a base-context token so fork PRs are labelled too.
permissions:
  contents: read
  pull-requests: write

# Reconcile to the latest PR state; a new edit/push cancels the superseded run.
concurrency:
  group: ${{ github.workflow }}-${{ github.event.pull_request.number }}
  cancel-in-progress: true

jobs:
  label:
    runs-on: ubuntu-latest
    steps:
      # No checkout (ADR-5): the labeler reads PR title, head branch, and the
      # changed-file list via the API. It loads .github/labeler.yml from the
      # BASE repo's default branch, so a fork cannot supply its own config.
      - name: Apply administrative labels
        uses: srvaroa/labeler@v1.14.0   # [ASSUMED] exact patch — pin at impl time
        continue-on-error: true          # labelling is administrative; a flake must never mark the PR
        env:
          GITHUB_TOKEN: ${{ github.token }}
```

### `.github/labeler.yml` structure

`srvaroa/labeler` config (`version: 1`). For **each** label in the config, the action evaluates its matchers: a match **adds** the label, a non-match **removes** it if present — and it only ever touches labels named here. That per-label evaluate→add/remove is exactly the spec's authoritative sync (B1), scoped to the managed set by construction (human labels outside this file are never touched).

The seven managed labels and their signals (ADR-3 — `breaking`/`features`/`fixes` double as the semver bump signal 030 resolves; the exact regexes are `[ASSUMED]` and tunable without changing the contract):

| Label | Title signal (conventional-commit) | Branch signal | Changed-files signal | semver |
|---|---|---|---|---|
| `breaking` | `^\w+(\([^)]+\))?!:` (a `!` after the type) | `^(major\|breaking)/` | — | major |
| `features` | `^feat(\([^)]+\))?:` | `^(feat\|feature)/` | — | minor |
| `fixes` | `^fix(\([^)]+\))?:` | `^(fix\|bug\|hotfix)/` | — | patch |
| `docs` | `^docs(\([^)]+\))?:` | — | `.*\.md$`, `docs/.*` | — |
| `infrastructure` | `^(ci\|build)(\([^)]+\))?:` | — | `\.github/workflows/.*`, `\.github/[^/]+\.ya?ml$`, `\.goreleaser\.ya?ml$`, `\.golangci\.ya?ml$`, `scripts/.*`, `Makefile`, `Dockerfile.*`, `\.pre-commit-config\.yaml` | — |
| `dependencies` | `^\w+\(deps\)` | `^dependabot/` | `go\.mod$`, `go\.sum$` | — |
| `internal` | `^(chore\|refactor\|perf\|test\|style)(\([^)]+\))?:` | `^(chore\|refactor\|cleanup)/` | — | — |

```yaml
version: 1
labels:
  - label: breaking
    title: "^\\w+(\\([^)]+\\))?!:"
    branch: "^(major|breaking)/"
  - label: features
    title: "^feat(\\([^)]+\\))?:"
    branch: "^(feat|feature)/"
  - label: fixes
    title: "^fix(\\([^)]+\\))?:"
    branch: "^(fix|bug|hotfix)/"
  - label: docs
    title: "^docs(\\([^)]+\\))?:"
    files:
      - ".*\\.md$"
      - "docs/.*"
  - label: infrastructure
    title: "^(ci|build)(\\([^)]+\\))?:"
    files:
      - "\\.github/workflows/.*"
      - "\\.github/[^/]+\\.ya?ml$"
      - "\\.goreleaser\\.ya?ml$"
      - "\\.golangci\\.ya?ml$"
      - "scripts/.*"
      - "Makefile"
      - "Dockerfile.*"
      - "\\.pre-commit-config\\.yaml"
  - label: dependencies
    title: "^\\w+\\(deps\\)"
    branch: "^dependabot/"
    files:
      - "go\\.mod$"
      - "go\\.sum$"
  - label: internal
    title: "^(chore|refactor|perf|test|style)(\\([^)]+\\))?:"
    branch: "^(chore|refactor|cleanup)/"
```

A PR matching no signal receives **no** managed label (spec edge case "an unrecognized change is not mislabelled"); Release Drafting (030) applies its default bump at release time. A PR may match several labels at once (e.g. `features` + `dependencies`); 030 resolves which semver label wins.

### Label catalog contract (`.github/settings.yml` `labels:` block — ADR-4)

The seven label **definitions** live declaratively in `.github/settings.yml` and are reconciled by the Probot "Settings" GitHub App; the workflow assumes they exist (spec §Non-Behaviors). A labels-only file leaves unlisted labels untouched (no prune), so human triage labels survive. The block is the single source of truth for the label strings, colors, and descriptions. The **names** are the cross-feature contract Release Drafting (030) consumes — renaming one is a coordinated 028↔030 change.

| Label | Color `[ASSUMED]` | Description |
|---|---|---|
| `breaking` | `B60205` | Breaking change — major version bump |
| `features` | `0E8A16` | New feature — minor version bump |
| `fixes` | `FBCA04` | Bug fix — patch version bump |
| `docs` | `0075CA` | Documentation change |
| `infrastructure` | `D93F0B` | CI, build, tooling, and project-config change |
| `dependencies` | `0366D6` | Dependency manifest change |
| `internal` | `CFD3D7` | Internal change (chore/refactor/test) |

```yaml
# .github/settings.yml — consumed by the Probot "Settings" GitHub App.
# The labels block is the source of truth for the seven managed PR labels;
# the names are the contract Release Drafting (030) reads. Labels-only (no
# repository:/branches: blocks), so the app manages only these labels and
# leaves other (human triage) labels in place.
labels:
  - name: breaking
    color: "B60205"
    description: "Breaking change — major version bump"
  - name: features
    color: "0E8A16"
    description: "New feature — minor version bump"
  - name: fixes
    color: "FBCA04"
    description: "Bug fix — patch version bump"
  - name: docs
    color: "0075CA"
    description: "Documentation change"
  - name: infrastructure
    color: "D93F0B"
    description: "CI, build, tooling, and project-config change"
  - name: dependencies
    color: "0366D6"
    description: "Dependency manifest change"
  - name: internal
    color: "CFD3D7"
    description: "Internal change (chore/refactor/test)"
```

---

## Interactions

**Labelling flow (end-to-end):** A contributor opens / reopens / pushes to / edits the title of a pull request (from any branch, fork or not) → the `pull_request_target` event starts `pr-administration.yml` → the `label` job loads `.github/labeler.yml` from the base repo's default branch and evaluates each of the seven labels against the PR's title, head branch, and changed-file list → matching labels are added, previously-applied managed labels that no longer match are removed (B1 sync), labels outside the seven are left untouched → the resulting labels sit on the PR for a maintainer to read and for Release Drafting (030) to consume on merge.

**Re-reconcile on update:** `synchronize` re-runs on each head push (recomputes changed-files signals); `edited` re-runs on a title or base/branch edit (recomputes the semver-bearing signals). `cancel-in-progress` cancels the superseded run so the final labels reflect the latest state — e.g. editing a title from `feat:` to `fix:` removes `features` and adds `fixes`.

**Fork safety:** because the config is read from the **base** repo and PR head code is never checked out or executed (ADR-5), a fork PR is labelled on the same terms as an internal one with no untrusted-code exposure.

**Contract handed to 030:** Release Drafting reads `breaking`/`features`/`fixes` to compute the semver bump (major/minor/patch) and groups merged-PR titles under the seven category headings. 028 produces and syncs the labels; 030 owns resolution (which semver label wins, the default-when-none bump, and which categories are rendered vs excluded as noise).

---

## Error Communication

| Condition | Behavior |
|---|---|
| No signal matches | No managed label is applied (the PR is left as-is for that run); 030's default bump covers the no-label case. Not an error. |
| Multiple signals match | All matching managed labels are applied; resolving multiple semver labels into one bump is 030's job. Not an error. |
| Stale label after a title/branch edit | The next run removes the no-longer-matching managed label and applies the correct one (B1). |
| Hand-applied label outside the managed set | Untouched — the labeler only adds/removes labels named in `.github/labeler.yml`. |
| Transient labels-API failure / action error | `continue-on-error: true` keeps the failure from surfacing as a blocking red mark; the workflow is not a required check, so the merge is never blocked (spec §Non-Blocking). The next triggering event re-reconciles. |
| Missing label definition | If a managed label is not yet reconciled by the Settings app (app not installed, or `settings.yml` not yet applied), the labeler creates it on apply with a default color; `settings.yml` remains the source of truth for the intended color/description, and the app normalizes it on the next reconcile. |
| Fork PR | Labelled identically to an internal PR; no checkout or execution of PR head code (ADR-5). |

---

## Consistency Notes

- **Sibling boundary (024 PR Validation):** 024's `ci.yml` is a **separate** workflow on `pull_request` with `contents: read` only — the read-only, fork-safe merge gate. PR Administration is deliberately a distinct file on `pull_request_target` with `pull-requests: write` (ADR-1) so the gate's least-privilege posture is never widened. Neither workflow gates the other; PR Administration is **not** added to branch protection (024's `ci-success` stays the sole required check), so labelling never blocks merge — 024's own interface-spec already records that PR Validation "neither reads nor writes PR labels."
- **Sibling boundary (030 Release Drafting):** the seven label strings are the contract 030 consumes; this accord is their definition point. There is no automated cross-feature guard until 030 ships — 030 should consume these exact names and may add a config-guard test then (out of scope for 028, recorded as a plan Risk). **Update (2026-06-12):** 030's design introduces an **eighth** managed label, `no-release-note` (exclusion; to be applied here by the labeler to PRs confined to `specs/`+`.feature`), so the managed *set* **will become eight once 030 task T001 lands it** — the seven note **categories** stay unchanged. 030's `internal/build` config guard is the cross-feature guard this note anticipated, and it will re-pin the eight-label invariant. Until T001 lands, the managed set is still seven. See 028 plan ADR-3 (superseded-in-part note) and `.score/memory/DEPRECATION.md`.
- **Convention — pinned action versions:** matches 022/024's "pin the toolchain, don't float `latest`" discipline — `srvaroa/labeler` is pinned to a concrete tag so an upstream release can't silently change labelling. Both developer-supplied reference repos (ailign-cli, terraform-vitals-template) use this same labeler + release-drafter shape; the label *taxonomy* here is this repo's seven FEATURE-MODEL categories rather than either repo's scheme (ADR-3).
- **Convention — declarative label catalog (`settings.yml`):** the label definitions live in `.github/settings.yml` (Probot Settings app), matching the team convention in ailign. This diverges from 024's imperative `scripts/setup-branch-protection.sh` in favour of config-as-code that is reviewed in PRs and self-reconciling. The Settings app is a one-time repo/org prerequisite; 024's branch-protection step could later converge into the same file's `branches:` block (out of scope here).
- **Assumption — matcher regexes and label colors (`[ASSUMED]`):** the conventional-commit/branch patterns, path globs, and label colors are tunable in a follow-up without changing the contract. The load-bearing contract is fixed: seven managed categories (plus the forthcoming `no-release-note` exclusion label that 030 task T001 will add — see the 030 update above), both title/branch and changed-files signals, authoritative sync scoped to the managed set, every PR (incl. forks) labelled, never a merge gate.
- **Assumption — labeler removal default:** B1 sync relies on `srvaroa/labeler`'s per-label evaluate→add/remove behavior being active by default; if the pinned version requires an explicit removal/`sync` flag, the implementation sets it. The behavioral contract (stale managed labels are removed) is fixed; the config key is an implementation knob.
- **CONSTITUTION XII note:** the labeler action and `gh` are CI-host tools, not artifact dependencies — XII governs the produced binary's runtime, the same standing the project gives GoReleaser and golangci-lint.
