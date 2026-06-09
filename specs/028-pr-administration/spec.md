# Specification: PR Administration

**Feature**: 028-pr-administration
**Role**: Definer
**Tier**: 1 (zero setup)

---

## System Overview

PR Administration is the **labelling stage** of the glassfrog-cli pipeline. On every pull request, it reads what the pull request *is about* — from its title, its head branch name, and the files it changes — and applies a managed set of **administrative labels** that classify the change. The labels serve two consumers at once: a maintainer triaging the pull request at a glance, and Release Drafting (030), which reads them to compute the next semver bump and to file the change under the right release-note category.

It is deliberately narrow: it **only labels**. It does not run lint or tests, does not gate the merge, does not compute a version, does not draft notes, and does not build or publish anything. PR Validation (024) owns the merge-blocking quality gate on the same pull-request events; Release Drafting (030) owns the semver bump and the release notes that consume these labels; Automated Release Pipeline (022) owns build-and-publish. PR Administration owns the one thing they all depend on for categorization: putting the right, current labels on the pull request.

The managed label set mirrors the seven release-note categories the pipeline recognizes: **breaking**, **features**, **fixes**, **docs**, **infrastructure**, **dependencies**, and **internal**. The first three (breaking / features / fixes) also carry semver intent (major / minor / patch) that Release Drafting reads.

---

## Behavioral Accord

### Trigger

- When a pull request is opened, labelling runs and applies the labels its current state warrants.
- When a pull request is updated in a way that can change its classification — new commits pushed to the head (changed files), or its title or head branch edited — labelling re-runs and reconciles the labels to the new state.
- When a closed pull request is reopened, labelling runs against its current state.
- Labelling runs for **every** pull request, including those opened from forks by external contributors. It classifies the pull request by reading its metadata and the list of changed files only; it never checks out or executes the pull request's code (see Non-Behaviors).

### Label sources

Labels are derived from two kinds of signal, combined:

- **Title / branch signal** — the pull request's title (a conventional-commit-style type, optionally with a breaking marker) and/or its head branch name (a recognized prefix). This is the source for the **semver-bearing** categories:
  - a breaking marker on the type, or a breaking-intent branch → **breaking** (major)
  - a feature type, or a feature-intent branch → **features** (minor)
  - a fix type, or a fix-intent branch → **fixes** (patch)
  - the title/branch signal may also indicate a non-semver category (e.g. a docs- or chore-typed title → **docs** / **internal**).
- **Changed-files signal** — the set of paths the pull request modifies. This is the source for the **non-semver** categories:
  - documentation paths → **docs**
  - build/CI and project-tooling paths → **infrastructure**
  - dependency-manifest paths → **dependencies**
  - and so on for the remaining managed categories.

When several signals match, the pull request carries **all** the matching labels from the managed set. A pull request can hold more than one managed label (e.g. **features** from its title and **docs** from a changed `README`). Resolving multiple semver-bearing labels into a single bump, and the default bump when none is present, are Release Drafting's (030) concern, not this feature's.

### Authoritative reconciliation (sync)

- The labels PR Administration manages always reflect the pull request's **current** state. When a re-run finds that a previously-applied managed label no longer matches (for example, the title was edited from a feature type to a fix type), it **removes** the stale label and applies the now-correct one.
- Reconciliation is scoped to the **managed set only**. Labels outside the managed set — human triage labels, hold/needs-work flags, anything a maintainer added by hand — are never added, removed, or disturbed.

### Non-blocking

- The labelling result is **administrative, not a gate**. Whether labelling succeeds, fails, or is delayed has no bearing on whether the pull request may merge. A labelling failure must not block or redden the merge — the merge gate is PR Validation's (024), and it depends on code, not labels.

---

## User Scenarios

**In order to** triage a pull request and know what kind of change it is at a glance,
**as a** maintainer,
**I want** every pull request automatically labelled by what it changes and what it is.

**In order to** produce a correct semver bump and well-categorized release notes without hand-labelling every merge,
**as a** maintainer running Release Drafting,
**I want** each pull request to already carry accurate, current category and version labels.

**In order to** have my contribution categorized correctly even though I work from a fork,
**as an** external contributor or AI agent submitting a change,
**I want** my pull request labelled on the same terms as an internal one.

---

## Non-Behaviors

- PR Administration must not run lint or tests, and must not report a pass/fail quality status. **Why**: PR Validation (024) owns the pre-merge gate; duplicating verification here would fork that contract and slow labelling.
- PR Administration must not block, gate, or fail a merge. **Why**: labelling is administrative; the merge verdict depends on code (024), not on labels, and a labelling hiccup must never trap an otherwise-mergeable pull request.
- PR Administration must not compute the semver version, draft or edit release notes, or decide a release's pre-release/latest status. **Why**: Release Drafting (030) owns the bump and the notes; this feature only supplies the labels 030 reads. Resolving which semver label wins is therefore 030's job.
- PR Administration must not build, package, publish, or release anything. **Why**: Automated Release Pipeline (022) owns shipping, gated on a published release — not on a pull request.
- PR Administration must not check out, run, or otherwise execute a pull request's code. **Why**: labelling every pull request — including forks — requires write access in the base-repository context; reading only the pull request's metadata and changed-file list (never its code) is what keeps that safe from untrusted contributions.
- PR Administration must not add, remove, or modify labels outside its managed set. **Why**: human triage labels and manual flags are the maintainer's; an authoritative sync that reached beyond its own labels would silently undo deliberate human labelling.
- PR Administration's **labelling path must not create or maintain the label definitions** (their names, colors, descriptions) as a side effect of labelling a pull request. **Why**: at PR time the labelling behaviour only applies and removes labels — it never mutates the catalog. The label definitions are provisioned separately, as declarative configuration applied outside the labelling path (a maintainer-managed catalog), so the runtime labelling behaviour and the catalog's lifecycle stay decoupled. (The catalog mechanism is a plan/interface detail; the behavioural guarantee is that labelling itself does not own or change it.)

---

## Integration Boundaries

- **GitHub pull-request events (upstream / trigger source)**: opening, updating (push to head, title or branch edit), and reopening a pull request triggers a labelling run. The run reads the pull request's title, head branch, and changed-file list.
- **GitHub pull-request labels (destination)**: labelling applies and removes labels from the managed set on the pull request. Applying labels requires write access to the pull request even for fork-originated pull requests; that access is exercised without trusting or executing pull-request code.
- **Label catalog (system actor / setup)**: the managed labels are assumed to be defined in the repository ahead of time by a maintainer setup step. PR Administration consumes that catalog; it does not produce it.
- **Release Drafting (030) — downstream consumer**: reads the labels this feature applies to merged pull requests to compute the next semver bump and to group changes into release-note categories. PR Administration is the upstream supplier of those labels; 030 owns how they are resolved and rendered.
- **PR Validation (024) — sibling**: runs on the same pull-request events but owns an independent concern — the merge-blocking lint/test gate. The two do not depend on each other and neither gates the other.

---

## Driving Scenarios

### Happy path

**Scenario: a feature pull request is labelled from its title**
Given a pull request whose title declares a feature-type change
When labelling runs
Then the pull request carries the **features** label (minor semver intent)
And the label is available for Release Drafting (030) to read.

**Scenario: a docs-only change is labelled from its changed files**
Given a pull request that changes only documentation files
When labelling runs
Then the pull request carries the **docs** label
And no semver-bearing label is forced onto it by this feature.

**Scenario: a pull request carries multiple matching labels**
Given a pull request whose title declares a feature and which also changes a dependency manifest
When labelling runs
Then the pull request carries both the **features** and **dependencies** labels
And resolving them into a single semver bump is left to Release Drafting (030).

### Error scenarios

**Scenario: a labelling failure does not block the merge**
Given a pull request that is otherwise mergeable
When the labelling run fails or cannot complete
Then the merge is not blocked or reddened by PR Administration
And the merge gate remains solely PR Validation's (024) to decide.

**Scenario: an unrecognized change is not mislabelled**
Given a pull request whose title, branch, and changed files match no managed signal
When labelling runs
Then PR Administration applies no managed label rather than inventing one
And Release Drafting's (030) default bump applies at release time.

### Edge cases

**Scenario: editing the title reconciles the labels (sync)**
Given a pull request previously labelled **features** from a feature-type title
When the title is edited to a fix-type change
Then labelling removes **features** and applies **fixes**
And the pull request no longer carries a label that contradicts its current state.

**Scenario: a fork pull request is labelled on the same terms**
Given a pull request opened from a fork by an external contributor
When labelling runs
Then the pull request is classified and labelled like an internal one
And its code is never checked out or executed to do so.

---

## Validation Scenarios

> These are held out from the implementing agent for independent verification.

**Scenario: reconciliation never touches labels outside the managed set**
Given a pull request a maintainer has hand-labelled with a triage label outside the managed set
When labelling re-runs and reconciles its managed labels
Then the hand-applied triage label is left untouched
And only labels within the managed set are added or removed.

**Scenario: labelling is not a required check**
Given the repository's merge protection
When it is inspected
Then PR Administration's labelling is not configured as a required, merge-blocking status
And only the verification gate (024) blocks merge.

**Scenario: fork labelling reads metadata only**
Given a labelling run on a fork-originated pull request
When the run is inspected
Then it derives labels from the pull request's title, branch, and changed-file list
And it does not check out or execute the pull request's head code.

---

## Assumptions

- **Managed label set** `[ASSUMED]`: the seven categories (breaking, features, fixes, docs, infrastructure, dependencies, internal) are fixed by the pipeline's release-note contract (FEATURE-MODEL). The exact label *strings*, colors, and descriptions — and the precise path globs and title/branch patterns that map to each — are plan/interface details; the behavioral requirement is that these seven categories are recognized from the combined title/branch and changed-files signals.
- **Semver intent** `[ASSUMED]`: breaking → major, features → minor, fixes → patch. PR Administration applies the label; the bump itself is computed by Release Drafting (030).
- **Label-source precedence**: the behavioral rule is that title/branch drives the semver-bearing categories and changed-files drives the non-semver categories, and that all matching managed labels are applied. The exact conflict-resolution and any default-when-nothing-matches at *bump* time belong to Release Drafting (030).
- **Label definitions pre-exist**: the managed labels are created once by a maintainer setup step (mirroring how PR Validation's required-gate branch protection is applied once outside the workflow). PR Administration assumes they exist.
- **CI provider is GitHub Actions** (technical default): the repository lives on GitHub and ships releases through GitHub (022), so the pull-request trigger, the labelling mechanism, and the fork-safe write access run on GitHub Actions. The behavioral requirements — label every pull request including forks, sync to current state, never gate the merge, never execute PR code — are independent of that default.
- **Scope is the seven release-note categories**: other label families a project might want (PR size, domain/area, hold flags) are out of scope for this feature and may be added later; recorded here so the narrow scope is explicit.

---

## Ambiguity Warnings

_None remaining — the three behavioral forks were resolved during the defining conversation: (1) labels are derived from **both** the title/branch signal (semver-bearing categories) **and** the changed-files signal (non-semver categories); (2) labelling is **authoritative** — it reconciles its managed labels to the pull request's current state, removing stale ones, while never touching labels outside the managed set; and (3) **every** pull request is labelled, including forks, by reading metadata and changed files only and never executing pull-request code. The remaining `[ASSUMED]` items (exact label strings, path globs, title/branch patterns, the labelling mechanism, and label-definition setup) are plan/interface-level details, not behavioral gaps._
