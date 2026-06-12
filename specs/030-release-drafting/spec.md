# Specification: Release Drafting

**Feature**: 030-release-drafting
**Role**: Definer
**Tier**: 1 (zero setup)

---

## System Overview

Release Drafting is the **note-keeping stage** of the glassfrog-cli pipeline. As changes land on `main`, it maintains a single **draft GitHub Release** that always reflects everything merged since the last published release: the next semantic version, and the merged changes grouped into release-note categories. It never publishes — the draft sits ready until a maintainer publishes it, and that publish act is what triggers Automated Release Pipeline (022) to build and attach the binaries.

It is the seam between *what changed* and *shipping it*. PR Administration (028) is its upstream supplier: every pull request already carries the administrative labels 028 applied (the semver-bearing **breaking** / **features** / **fixes**, the non-semver **docs** / **infrastructure** / **dependencies** / **internal**, and an exclusion label). Release Drafting reads those labels on the merged pull requests to do the two things 028 deliberately left to it — **resolve the many labels into one semver bump**, and **group the merged-PR titles into the right note categories**. It owns the draft, the computed version, the notes, and the release's pre-release/latest status; 022 owns the binaries and consumes the published release. This keeps the project's release legible — a maintainer reviews an always-current draft rather than hand-assembling notes at release time.

---

## Behavioral Accord

### Trigger

- When a change is merged to `main`, Release Drafting runs and reconciles the draft release to reflect every pull request merged since the last published release.
- It does not run on pull-request events and computes nothing at pull-request time — the merge to `main` is the trigger. Until then, PR Administration (028) has already supplied the labels it will read.

### Version computation

- The next version is computed relative to the **last published release's tag**: the bump is applied to that baseline to produce the draft's proposed `vMAJOR.MINOR.PATCH` tag.
- When no published release exists yet (the first ever release), there is no baseline tag; the first draft proposes **`v0.1.0`**. Subsequent releases apply the normal bump to the last published tag.
- The bump is resolved from the semver-bearing labels carried across all included merged pull requests, **highest wins**: any **breaking** label forces a **major** bump; otherwise any **features** label forces a **minor** bump; otherwise any **fixes** label forces a **patch** bump.
- When no included pull request carries a semver-bearing label, the bump defaults to **patch**.

### Note categorization

- Each included merged pull request contributes one note line — its title — filed under the category its label names: **breaking**, **features**, **fixes**, **docs**, **infrastructure**, **dependencies**, or **internal**.
- A pull request carrying more than one managed label appears under each matching category; the version bump still resolves to the single highest semver-bearing label (above).
- Categories with no contributing pull requests are omitted from the notes.

### Exclusion

- A merged pull request marked with the **exclusion label** is omitted from the notes entirely and is ignored when resolving the bump.
- A merged pull request whose changes are confined to **specification and planning artifacts** — the `specs/` directory and `.feature` files — is likewise omitted and ignored for the bump. Only changes that touch shipped code or real documentation (the `docs/` directory or `README.md`) are noteworthy. **Why this lives here**: spec and feature-file churn is internal pipeline bookkeeping, not a user-facing change; surfacing it would pad every release's notes and could force spurious version bumps.

### Draft maintenance (authoritative reconciliation)

- The draft is regenerated authoritatively on every run: it converges on one draft that reflects the current full set of included merged pull requests since the last release, rather than appending duplicates across runs.
- The draft is set with its **pre-release vs latest** status from the proposed version: while the version is in the `0.x` range the draft is marked a **pre-release**; once it reaches `1.0.0` or higher the draft is marked **latest**. The maintainer who publishes it, and Automated Release Pipeline (022) which honors that status, both inherit the decision made here.

### Stops at the draft

- Release Drafting maintains the draft and never publishes it. The draft carries no tag until a maintainer publishes it; publishing is the maintainer's intentional act and the single trigger for 022.

---

## User Scenarios

**In order to** ship without hand-assembling release notes or guessing the next version,
**as a** maintainer,
**I want** an always-current draft release that already carries the computed version and categorized notes for everything merged since the last release.

**In order to** keep release notes about real, user-facing change,
**as a** maintainer,
**I want** spec/feature-file-only and explicitly-excluded pull requests left out of the notes and the version bump.

**In order to** decide when a version actually ships,
**as a** maintainer,
**I want** Release Drafting to stop at a draft I publish deliberately, rather than releasing automatically on merge.

---

## Non-Behaviors

- Release Drafting must not publish the release or create its git tag. **Why**: publishing is the maintainer's deliberate, versioned act and the single trigger for Automated Release Pipeline (022); auto-publishing on merge would ship every commit and flood Releases with un-reviewed versions.
- Release Drafting must not build, package, attach, or upload binaries or checksums. **Why**: Automated Release Pipeline (022) owns shipping artifacts, gated on a *published* release; this stage only curates the notes the published release will carry.
- Release Drafting must not run lint or tests, nor gate or block the merge. **Why**: PR Validation (024) owns the pre-merge gate and Main-Branch Verification (029) the post-merge net; drafting is administrative and runs after the merge has already happened.
- Release Drafting must not apply, remove, or manage pull-request labels. **Why**: PR Administration (028) owns the labels; Release Drafting only *reads* them. Mutating labels here would fork the labelling contract and double-own the managed set.
- Release Drafting must not run on pull-request events or compute a version/notes at pull-request time. **Why**: the draft reflects what has *merged*; the labels it reads are already on the pull request by merge time, and computing at PR time would draft from changes that may never land.
- Release Drafting must not decide a release's pre-release/latest status at *publish* time or re-mark a published release. **Why**: the status is set on the draft here and honored downstream by 022; the publishing decision stays in one place.

---

## Integration Boundaries

- **Merge to `main` (upstream / trigger source)**: a merge triggers a drafting run that reconciles the draft against all pull requests merged since the last published release.
- **PR Administration (028) labels (input)**: the semver-bearing (**breaking** / **features** / **fixes**), non-semver category (**docs** / **infrastructure** / **dependencies** / **internal**), and **exclusion** labels that 028 applied to each pull request. Release Drafting reads them; it never writes them. Release Drafting owns resolving the bump and grouping the categories — the concerns 028 explicitly deferred.
- **Merged pull-request metadata (input)**: each included pull request's title becomes a note line under its category; the set of changed files distinguishes spec/feature-only churn from noteworthy change.
- **Last published GitHub Release / tag (input)**: the baseline the bump is applied to when computing the draft's proposed version.
- **Draft GitHub Release (destination)**: the single draft this stage maintains — its version, categorized notes, and pre-release/latest status — left unpublished.
- **Automated Release Pipeline (022) — downstream**: triggered only when a maintainer publishes the draft. It consumes the published release (reads the tag for the version, honors the pre-release/latest status) and attaches the binaries and checksums; it does not touch the notes.

---

## Driving Scenarios

### Happy path

**Scenario: a feature merge bumps the draft to the next minor and files a note**
Given the last published release is `v1.2.0` and a pull request labelled **features** merges to `main`
When Release Drafting runs
Then the draft's proposed version is `v1.3.0`
And the pull request's title appears under the **Features** category of the draft notes
And the release remains a draft.

**Scenario: highest semver label wins across several merges**
Given several pull requests labelled **fixes**, **features**, and **breaking** have merged since the last published release `v1.2.0`
When Release Drafting runs
Then the draft's proposed version is `v2.0.0` (the **breaking** label forces a major bump)
And each pull request's title appears under its own category (Breaking / Features / Fixes).

**Scenario: a docs-only change drafts under Docs with a default patch bump**
Given the last published release is `v1.2.0` and a pull request changing only `README.md`, labelled **docs**, merges
When Release Drafting runs
Then the pull request's title appears under the **Docs** category
And the draft's proposed version is `v1.2.1` (no semver-bearing label → default patch bump).

### Error scenarios

**Scenario: the draft is never published automatically**
Given a pull request merges to `main` and the draft is updated
When Release Drafting completes
Then the release remains an unpublished draft with no tag created
And Automated Release Pipeline (022) does not run until a maintainer publishes it.

**Scenario: an excluded pull request affects neither notes nor version**
Given the last published release is `v1.2.0` and a pull request labelled **features** but also carrying the **exclusion label** merges
When Release Drafting runs
Then the pull request's title does not appear anywhere in the draft notes
And it does not contribute to the version bump.

### Edge cases

**Scenario: a spec-only pull request is omitted without a label**
Given a pull request whose changed files are confined to `specs/` and `.feature` files merges
When Release Drafting runs
Then the pull request's title does not appear in the draft notes
And it does not drive the version bump, even if it carries a semver-bearing label.

**Scenario: the first ever release proposes v0.1.0 as a pre-release**
Given no GitHub Release has been published yet and a pull request labelled **features** merges to `main`
When Release Drafting runs
Then the draft's proposed version is `v0.1.0`
And the draft is marked a pre-release (the version is in the `0.x` range).

**Scenario: reconciliation converges rather than duplicating**
Given a pull request was already reflected in the draft from an earlier run
When a later merge triggers Release Drafting again
Then the draft reflects the current full set of pull requests merged since the last release
And the earlier pull request's note line appears exactly once, not duplicated.

---

## Validation Scenarios

> These are held out from the implementing agent for independent verification.

**Scenario: the spec names no specific drafting tool or workflow syntax in its behaviors**
Given the Behavioral Accord and Non-Behaviors sections
When read end to end
Then they describe observable outcomes (draft state, version, categories, status) without prescribing a particular GitHub Action, workflow file, or configuration format.

**Scenario: every category the notes can produce traces to an 028 managed label**
Given the seven note categories in this spec
When compared against PR Administration (028)'s managed label set
Then the category names match exactly (breaking / features / fixes / docs / infrastructure / dependencies / internal) with no category invented here.

---

## Assumptions

- **Intended mechanism is release-drafter** [ASSUMED]: per FEATURE-MODEL, the project intends to realize this with the release-drafter GitHub Action. The behaviors here are specified tool-agnostically; selecting and configuring the action (its categories, version-resolver, and templates) is a plan-level decision.
- **Baseline is the last *published* release**: the version bump is computed against the most recent published (non-draft) release's tag, so re-runs before publishing keep proposing against the same baseline rather than compounding bumps. (Matches how 022 reads the tag from the *published* release.)
- **Exclusion label name is a plan detail**: the spec requires a single managed exclusion label supplied by 028; its concrete name (e.g. `skip-changelog`) is chosen during interface/plan, alongside 028's catalog.
- **Spec/feature-only detection mechanism is a plan detail**: whether the "confined to `specs/` and `.feature`" exclusion is realized as a label applied by 028 or a path filter evaluated by 030 is left to the plan; the behavioral guarantee is that such changes are omitted from notes and the bump.

---

## Ambiguity Warnings

None remaining — the pre-release/latest status rule was resolved during clarification (see Clarifications).

---

## Clarifications

### Session 2026-06-12

- **Pre-release vs latest status**: the draft's status derives from the proposed version — pre-release while the version is in the `0.x` range, latest once it reaches `1.0.0+`. This resolved the open Ambiguity Warning. (Behavioral Accord — Draft maintenance.)
- **First-release baseline**: when no published release exists yet, the first draft proposes `v0.1.0`; the normal bump applies to subsequent releases. Closed the bootstrap gap in version computation. (Behavioral Accord — Version computation; new edge-case driving scenario.)
