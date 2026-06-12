# Plan: PR Administration

**Feature**: 028-pr-administration
**Role**: Shaper
**Inputs**: spec.md, PROJECT.md, `.score/memory/DECISIONS.md` (esp. 024's `.github/workflows/ci.yml` + maintainer-script precedent, 022's release workflow, 021/022's GitHub Actions + pinned-action discipline), LEARNINGS.md / DEPRECATION.md (no conflicting precedent for CI labelling), sibling 024's plan.md (the PR-event CI workflow this one runs beside), and two reference repos the developer supplied (ailign-cli and the terraform-vitals-template — both `srvaroa/labeler` + `release-drafter`)

---

## System Architecture

PR Administration ships **no application code** — like 022 and 024, it is CI infrastructure. It introduces no Go package and registers no cobra command. It is three declarative artifacts plus one maintainer-run setup step:

1. `.github/labeler.yml` — the signal→label mapping: which PR title/branch patterns and which changed-file globs map to each of the seven managed categories.
2. `.github/workflows/pr-administration.yml` — the workflow that runs a pinned labeler action against that config on every pull request.
3. `.github/settings.yml` — a declarative `labels:` block (Probot "Settings" GitHub App) defining the seven label definitions, continuously reconciled by the app.

The workflow maps one-to-one onto the spec's Behavioral Accord:

```
pull_request_target  (opened | reopened | synchronize | edited)         ── spec §Trigger
   runs in the BASE-repo context → token can be granted pull-requests: write
   ⇒ fork PRs are labelled too (spec C1), with NO checkout of PR head code
        │   concurrency group keyed on ${{ github.workflow }}-<PR number>, cancel-in-progress
        ▼
[label job]   (ubuntu-latest, permissions: pull-requests: write + contents: read)
   pinned labeler action reads PR title + head branch + changed-file list
        │
        ├─ title/branch signal  → breaking | features | fixes   (semver-bearing)  ── spec §Label sources
        │                         (+ docs | internal from docs:/chore: types)
        ├─ changed-files signal → docs | infrastructure | dependencies | internal ── spec §Label sources
        │
        ▼   authoritative sync, scoped to the managed set                 ── spec §Authoritative reconciliation
   apply matching managed labels; remove managed labels that no longer match;
   never touch labels outside the managed set (human triage labels untouched)
        │
        ▼   NOT a required status check (absent from branch protection)   ── spec §Non-blocking
   labelling success/failure has no bearing on mergeability (024 owns the gate)

   managed label set  ──(consumed downstream)──▶  Release Drafting (030):
        breaking→major · features→minor · fixes→patch  +  note categories
```

The label set is the **contract this feature produces for Release Drafting (030)**: 030 reads these exact category labels off merged PRs to compute the semver bump and to group release notes. 028 is the producer; 030 owns resolution (which semver label wins, the default bump, render-vs-exclude). The workflow runs *beside* 024's `ci.yml` on the same PR events but is a wholly separate file with a different trigger and a different permission posture — see ADR-1.

---

## Architecture Decisions

### ADR-1: A separate labelling workflow on `pull_request_target`, not an extension of `ci.yml`

**Context**: 024's `ci.yml` triggers on `pull_request` with `permissions: contents: read` — deliberately least-privilege and fork-safe (a fork PR validates under a read-only token). PR Administration must *write* labels (`pull-requests: write`) and must label fork PRs (spec C1). A fork PR under the standard `pull_request` event gets a read-only token and cannot be labelled.

**Options**:
- (a) Add a label job to `ci.yml`. Rejected: would force `ci.yml` to carry `pull-requests: write`, eroding the read-only posture that makes the gate fork-safe; and `pull_request` still can't label forks.
- (b) Separate workflow on `pull_request`. Rejected: still cannot label fork PRs (read-only token) — fails C1.
- (c) Separate workflow on `pull_request_target`. **Chosen.** `pull_request_target` runs in the base-repo context with a token that can be granted write scope, so fork PRs are labelled. Isolation from `ci.yml` keeps the verification gate read-only and the labeller's write scope contained to its own file.

**Decision**: A new `.github/workflows/pr-administration.yml` on `pull_request_target`, independent of `ci.yml`.

**Consequences**: Two PR workflows run on overlapping events with different permissions — clearer than one workflow with mixed scope. `pull_request_target` is security-sensitive (see ADR-5). Neither workflow gates the other; PR Administration is never a required check (spec §Non-blocking).

### ADR-2: Drive labelling with a pinned third-party labeler action + committed config, not a hand-rolled script

**Context**: Spec A3 requires labels from **both** the PR title/branch (semver-bearing categories) **and** the changed-file paths (non-semver categories), with authoritative sync (B1).

**Options**:
- (a) `actions/labeler` (official). Rejected: matches on changed files and base/head branch but has no PR-**title** matcher — cannot derive the conventional-commit semver categories A3 requires.
- (b) `srvaroa/labeler` (pinned tag). **Chosen.** A single config expresses title, branch, and file matchers together, and supports authoritative sync (it removes the labels it manages when they no longer match, and only ever touches labels named in its own config). This is exactly A3+B1, and it is the shape both reference repos already use.
- (c) Hand-rolled `actions/github-script`. Rejected: re-implements a maintained action; more surface to get the sync/scoping semantics wrong.

**Decision**: Use `srvaroa/labeler` pinned to a concrete version (not a floating tag), configured by `.github/labeler.yml`. Pinning follows 024's golangci-lint / action-version discipline so an upstream release can't silently change labelling behaviour.

**Consequences**: The signal→label mapping lives declaratively in `.github/labeler.yml`. "Scoped to the managed set" (spec §Authoritative reconciliation) is satisfied structurally — the action only operates on labels named in its config. Exact action version and config syntax are interface-level.

### ADR-3: One managed label family — the seven categories — with breaking/features/fixes doubling as the semver signal; no parallel `version:*` namespace

**Context**: FEATURE-MODEL fixes seven release-note categories (breaking / features / fixes / docs / infrastructure / dependencies / internal) and states Release Drafting computes the semver bump from PR labels. One reference repo (ailign) uses a parallel `version:*` + `type:*` scheme; the other (tf-vitals) mixes `version: …` with bare category labels.

**Options**:
- (a) Two families: `version:{major,minor,patch}` for the bump + category labels for notes. Rejected for this repo: redundant — 030 would reconcile two families, and a contributor could desync them.
- (b) **Chosen**: a single managed family of the seven category labels; `breaking`→major, `features`→minor, `fixes`→patch carry the semver intent directly, which 030's version-resolver maps to a bump. The other four are note-only categories.

**Decision**: The managed set is exactly the seven categories; the three semver-bearing ones double as the bump signal. No `version:*` labels.

**Consequences**: Simpler contract for 030 and for triage. A PR may carry several category labels (e.g. `features` + `dependencies`); resolving multiple semver-bearing labels into one bump is 030's job (spec §Label sources). The exact label *strings* are interface-level, but they are a stable contract 030 depends on — changing one is a coordinated 028↔030 change.

> **Superseded in part (2026-06-12, by 030-release-drafting ADR-4 — see `.score/memory/DEPRECATION.md`):** the "managed set is *exactly* the seven categories" closedness no longer holds as a forward constraint — 030's design introduces an **eighth** managed label, `no-release-note` (an exclusion marker, to be applied by the labeler to PRs confined to `specs/`+`.feature`), so the managed *set* **will become eight once 030 task T001 lands it**. The seven release-note **categories** are unchanged (the eighth is not a category), and the "no parallel `version:*` namespace" decision still stands. 030's `internal/build` config guard will re-pin the eight-label invariant. Until T001 lands, the repo's managed set is still seven — the `.github/settings.yml`/`.github/labeler.yml` files and their "seven managed labels" header comments correctly still read seven.

### ADR-4: Provision the label catalog declaratively via `.github/settings.yml` (Probot Settings app), not a `gh` script or on-the-fly creation

**Context**: Spec §Non-Behaviors: PR Administration applies/removes labels but does not own the label *definitions*; they are assumed to pre-exist. The definitions need a home that is version-controlled and low-maintenance. 024 set a "maintainer-run setup script" precedent (`scripts/setup-branch-protection.sh`), but a script is imperative — someone must remember to re-run it, and drift is not reconciled.

**Options**:
- (a) `scripts/setup-labels.sh` using `gh label create … --force`, run once by a maintainer (mirrors 024's script). Self-contained, but imperative: re-running is manual and a hand-deleted/renamed label is not restored.
- (b) **Chosen**: `.github/settings.yml` consumed by the Probot "Settings" GitHub App. The label catalog is declarative config-as-code — edited as YAML, version-controlled, reviewed in PRs, and **continuously reconciled** by the app (a renamed/deleted managed label is restored). This is the convention already used in the team's other repos (ailign), so the app is an established, not a new, dependency.
- (c) Have the workflow create labels on the fly. Rejected: violates the spec non-behavior (the workflow would own the catalog) and would need broader permissions.

**Decision**: Define the seven managed labels in `.github/settings.yml`'s `labels:` block (Probot Settings-app schema: `name`/`color`/`description`), labels-only.

**Consequences**: Accepted cost — the Probot "Settings" GitHub App must be installed on the repo/org (a one-time, out-of-repo setup; already established for the team via ailign). Benefit: a declarative, reviewable, self-reconciling label catalog with no script to remember. The catalog stays **outside** the labelling workflow (spec non-behavior preserved). A labels-only `settings.yml` leaves unlisted labels untouched (no prune), so human triage labels survive. Forward-looking: 024's branch-protection script could later converge into the same `settings.yml` (`branches:` block) to unify repo config-as-code — out of scope for 028, noted only as a direction.

### ADR-5: The labelling workflow never checks out or executes pull-request head code

**Context**: `pull_request_target` (ADR-1) runs with a writable token **in the base-repo context**. The well-known hazard is checking out the PR head and running its code (build, test, `npm install`, a custom script) — that executes untrusted fork code with a privileged token. Spec §Non-Behaviors forbids executing PR code precisely to make fork labelling safe.

**Options**:
- (a) `actions/checkout` the PR head for richer file inspection. Rejected: classic `pull_request_target` privilege-escalation vector.
- (b) **Chosen**: no checkout at all. The labeler reads PR title, head branch name, and the changed-file list via the GitHub API — metadata only, never file contents or code.

**Decision**: The workflow contains no `actions/checkout` of PR head code and runs no PR-supplied script. The job is the labeler action and nothing else.

**Consequences**: Labelling is safe on fork PRs (C1) while honouring the no-execute non-behavior. File-based labels match on **paths**, not contents — sufficient for the seven categories. This is a load-bearing safety invariant: any future step that adds a checkout under this trigger must be reviewed as a security change.

---

## Security Design

`pull_request_target` is the only architecturally security-sensitive choice. The posture:

- **Least privilege**: `permissions: pull-requests: write` (to label) + `contents: read`; nothing else. No `id-token`, no `contents: write`.
- **No untrusted execution** (ADR-5): no checkout of PR head, no PR-supplied scripts, no build/test of fork code. The labeler reads metadata only.
- **Pinned action** (ADR-2): a concrete version, so a compromised/changed upstream tag can't alter behaviour silently.
- **No secrets**: the default `GITHUB_TOKEN` (write-scoped to PRs) suffices; the workflow exposes no repository secrets to the labeller, so even the action itself sees no sensitive material.

---

## Cross-cutting Concerns

- **Non-blocking** (spec §Non-Blocking): the workflow is **not** added to branch protection, so a labelling failure can never block merge (024's `ci-success` remains the sole required check). The label step additionally tolerates transient API hiccups (`continue-on-error`-style) so a flake doesn't surface as a confusing red mark on the PR.
- **Latest-state semantics**: a `concurrency` group keyed on `${{ github.workflow }}-${{ github.event.pull_request.number }}` (same shape as 024's `ci.yml`, so the workflow name namespaces the group) with `cancel-in-progress` means rapid `edited`/`synchronize` events reconcile to the latest state without racing — consistent with 024's per-PR concurrency pattern and with B1 sync.
- **Trigger activity types**: `opened, reopened, synchronize, edited`. `edited` is required because the PR **title** drives the semver-bearing labels — a title edit must re-reconcile (spec edge case "editing the title reconciles the labels").
- **Contract with 030**: the seven label strings are the seam Release Drafting consumes. Until 030 exists there is no automated guard that the names stay in sync; flagged as a risk. The label set should be defined once and treated as stable.

---

## Implementation Strategy

Small surface; logical phases (the tasks skill may collapse these into one or two PRs):

- **Phase 1 — Label catalog**: `.github/settings.yml` `labels:` block (Probot Settings-app schema) defining the seven labels with colors/descriptions. Foundation: the labels must exist before labelling has anything to apply. Reconciled by the Settings app once installed; the app is a documented one-time repo/org prerequisite.
- **Phase 2 — Signal mapping**: `.github/labeler.yml` — title/branch patterns for breaking/features/fixes (conventional-commit + branch-prefix) and path globs for docs/infrastructure/dependencies/internal, with sync enabled. Depends on the names fixed in Phase 1.
- **Phase 3 — Workflow**: `.github/workflows/pr-administration.yml` on `pull_request_target` (opened/reopened/synchronize/edited), least-privilege permissions, concurrency group, pinned `srvaroa/labeler`, no checkout (ADR-5), non-blocking.

Verification is by inspection plus a live PR exercising: a feature-typed title → `features`; a docs-only diff → `docs`; a title edit feature→fix → `features` removed, `fixes` added (sync); a fork PR labelled; a hand-added out-of-set label left untouched.

---

## Risks

- **`pull_request_target` misuse** (high impact, low likelihood given ADR-5): the privilege-escalation vector is real but fully mitigated by the no-checkout invariant. Mitigation is a review rule, not just a one-time choice — any added checkout under this trigger is a security change.
- **Label-name drift between 028 and 030** (medium): the seven strings are an un-guarded cross-feature contract until 030 ships. Mitigation: define them once here as the source of truth; 030 must consume these exact names; consider a config-guard test when 030 is built (out of scope for 028).
- **Labeler action behaviour drift** (low): pinned version mitigates; a deliberate bump is a reviewable PR that itself runs the gate.
- **Mislabelling from imperfect heuristics** (low): a non-conventional title or unrecognized path yields no managed label rather than a wrong one (spec edge case); 030's default bump covers the no-label case. Acceptable — labelling is administrative, not a gate.
