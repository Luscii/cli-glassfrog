# Risk: PR Administration

**Feature**: 028-pr-administration
**Round**: 1
**Date**: 2026-06-09
**Artifacts loaded**: spec.md, plan.md, interface-spec.md, PROJECT.md
**Acceptability matrix**: default 3×3 traffic-light — no project-level matrix found in PROJECT.md
**Regulatory bridge**: none — PROJECT.md declares no Regulatory Context (no IEC 14971 mapping)
**Degradation flags**: none — full upstream artifact set present

---

## Risk Register

| H-ID | Hazard | Source | Severity | Probability | Risk | Controls | Residual |
|---|---|---|---|---|---|---|---|
| H-1 | PR head code executed under `pull_request_target` → write-token / secret compromise (privilege escalation) | spec §Non-Behaviors; plan ADR-5 / Security Design | High | Low | Yellow | RC-1, RC-2, RC-3 | **Yellow** |
| H-2 | Authoritative sync removes a human/triage label outside the managed set | spec §Authoritative reconciliation / Non-Behaviors; plan ADR-2 | Medium | Low | Green | RC-4 | **Green** |
| H-3 | A labelling failure blocks or reddens an otherwise-mergeable PR | spec §Non-Blocking; plan Cross-cutting | Medium | Low | Green | RC-5, RC-6 | **Green** |
| H-4 | A managed label is missing/misnamed in the catalog → a stray label is auto-created or the wrong label applied | spec §Non-Behaviors (catalog pre-exists); interface Error Communication | Medium | Medium | Yellow | RC-7 | **Yellow** |
| H-5 | Label-name drift between this feature and Release Drafting (030) → wrong/missing semver bump or miscategorized notes | plan Risks; spec §Label sources (030 boundary) | Medium | Medium | Yellow | RC-7, RC-8 | **Yellow** |
| H-6 | Mislabelling from imperfect title/branch/path heuristics → wrong semver bump (e.g. a breaking change titled `fix:`) | spec edge "unrecognized not mislabelled"; plan Risks | Medium | Medium | Yellow | RC-8, RC-9, RC-10 | **Yellow** |
| H-7 | Third-party labeler action supply-chain compromise / behaviour drift → arbitrary code runs with the PR-write token | plan ADR-2 / Risks | High | Low | Yellow | RC-2, RC-11 | **Yellow** |
| H-8 | `continue-on-error` masks a *persistent* labelling failure → labels silently stop, feeding 030 a degraded/wrong bump | plan Cross-cutting; checklist P2-1 | Medium | Medium | Yellow | RC-12 | **Yellow** |

**No Red (unacceptable) residual risks.** Five Yellow (acceptable with documented justification), three Green.

---

## Per-Hazard Detail

### H-1 — PR-code execution under `pull_request_target`
- **Description**: `pull_request_target` runs in the base-repo context with a writable token. If the workflow checked out and ran fork-supplied code (build step, `npm install`, a repo script), untrusted code would execute with that token — the canonical privilege-escalation vector.
- **Severity High**: blast radius is the whole repository — token exfiltration, malicious writes to PRs, a foothold toward `main`.
- **Probability Low**: the design forecloses it — ADR-5 mandates **no `actions/checkout`** and no PR-supplied script; the job is the labeler step only.
- **Controls**: **RC-1** no checkout/execution of PR head (the labeler reads metadata + changed-file paths via the API). **RC-2** least privilege (`contents: read` + `pull-requests: write`, no `id-token`, no secrets exposed to the action). **RC-3** the labeler config is read from the **base** branch, so a fork cannot inject its own `labeler.yml`.
- **Residual Yellow**: High×Low. Accepted on the condition that the no-checkout invariant is preserved — this is a standing review rule, not a one-time choice (plan Risks). Any future step adding a checkout under this trigger is a security change.

### H-2 — Sync removes a label outside the managed set
- **Severity Medium** (loss of a maintainer's triage signal; recoverable). **Probability Low**.
- **Controls**: **RC-4** the labeler only adds/removes labels named in its own config (structural scoping) — labels outside the seven are never touched.
- **Residual Green**.

### H-3 — Labelling failure blocks a merge
- **Severity Medium** (a contributor blocked), **Probability Low**.
- **Controls**: **RC-5** the workflow is not added to branch protection (never a required check). **RC-6** `continue-on-error` on the label step so a flake doesn't surface as a blocking mark.
- **Residual Green**.

### H-4 — Missing/misnamed managed label
- **Severity Medium** (a misnamed *semver* label corrupts the 030 bump; a stray label clutters triage). **Probability Medium** — the catalog is declarative in `.github/settings.yml` and must match `labeler.yml`; a typo is still plausible (and not auto-cross-checked), and the labeler auto-creating a missing label (interface Error Communication) can mask a typo as a "new" stray label. A *new* sub-hazard: if the Probot Settings app is **not installed**, `settings.yml` is never reconciled and the labeler silently auto-creates labels with default colors — a degraded but non-failing catalog.
- **Controls**: **RC-7** `.github/settings.yml` is the single declarative source of truth for the names, reviewed in PRs and continuously reconciled by the Settings app (a hand-deleted/renamed managed label is restored — an improvement over an imperative script); the names must be identical across `settings.yml`, `labeler.yml`, and 030. (No automated cross-file guard today — see H-5.)
- **Residual Yellow**: accepted; mitigation is name discipline + PR review of `settings.yml` + app reconciliation, plus the deferred guard noted in H-5. Document the Settings-app install as a prerequisite so the not-installed sub-hazard is visible.

### H-5 — Label-name drift with Release Drafting (030)
- **Severity Medium** (wrong/missing bump → mis-versioned release, but caught by the human who publishes the 030 draft). **Probability Medium** — the seven names are an **un-guarded** cross-feature contract until 030 ships; they live in 2–3 places.
- **Controls**: **RC-7** names defined once here as the contract. **RC-8** human-in-the-loop review at release publish (030 drafts; a maintainer publishes); a config-guard test should be added **when 030 is built** (out of scope for 028).
- **Residual Yellow**: accepted, documented; the automated guard is explicitly deferred to 030 — flagged so it is not forgotten.

### H-6 — Mislabelling from imperfect heuristics
- **Severity Medium** (wrong bump, catchable at publish). **Probability Medium** (non-conventional titles / unprefixed branches happen).
- **Controls**: **RC-9** a no-match yields **no** managed label rather than a wrong one (conservative default; 030 applies its default bump). **RC-10** authoritative sync corrects the label when the title is later fixed. **RC-8** human review at publish.
- **Residual Yellow**: the dominant residual is under-labelling (no label → default patch when a minor/major was warranted), not wrong-labelling; the publish step is the backstop.

### H-7 — Third-party action supply-chain / behaviour drift
- **Severity High** (arbitrary code with the PR-write token). **Probability Low**.
- **Controls**: **RC-11** the action is pinned to a concrete version, not a floating tag, so an upstream release can't silently change behaviour. **RC-2** least privilege + no secrets bound the blast radius.
- **Residual Yellow**: accepted. *Enhancement*: pin to a full commit SHA (rather than a version tag) for a stronger supply-chain guarantee — recommended, not required.

### H-8 — `continue-on-error` masks persistent failure
- **Severity Medium** (silent label drift degrades 030 over time). **Probability Medium** (a malformed `labeler.yml` or a removed action tag fails deterministically and silently).
- **Controls**: **RC-12** per-event re-reconciliation retries transient failures (so flakes self-heal).
- **Residual Yellow**: accepted under the explicit non-blocking design. *Enhancement* (checklist P2-1): treat a deterministic **config-parse** failure differently from a flake so persistent breakage stays legible without re-introducing a merge block.

---

## Residual Risk Summary

- **0 Red**, **5 Yellow**, **3 Green**. No unacceptable risks; implementation may proceed.
- The two **High-severity** hazards (H-1 PR-code execution, H-7 supply chain) are reduced to **Yellow** by firm architectural controls (no-checkout, least privilege, pinned action, base-branch config). Their residual rests on **preserving** those controls — especially the no-checkout invariant (H-1).
- The **labelling-correctness** cluster (H-4/H-5/H-6/H-8) is uniformly **Medium×Medium → Yellow**, all rooted in the un-guarded label-name contract and heuristic imperfection. They are acceptable because (a) labels are administrative, not governance data, and (b) a human publishes the 030 draft release, providing a backstop before any version actually ships.

## Traceability Index

- **H-1** → spec §Non-Behaviors ("must not execute PR code"); plan ADR-5 / Security Design.
- **H-2** → spec §Authoritative reconciliation; plan ADR-2 (scoped sync).
- **H-3** → spec §Non-Blocking; plan Cross-cutting.
- **H-4** → spec §Non-Behaviors (catalog pre-exists); interface §Error Communication (missing-label auto-create).
- **H-5** → plan §Risks (label-name drift); spec §Label sources (030 boundary).
- **H-6** → spec Driving Scenarios edge ("an unrecognized change is not mislabelled"); plan §Risks.
- **H-7** → plan ADR-2 / §Risks (action behaviour drift).
- **H-8** → plan Cross-cutting (`continue-on-error`); checklist P2-1.
- **RC-1/RC-2/RC-3** → plan ADR-5 + Security Design (no checkout, least privilege, base-branch config).
- **RC-4** → plan ADR-2 / interface §`.github/labeler.yml` (managed-set scoping).
- **RC-5/RC-6** → spec §Non-Blocking; interface §`pr-administration.yml` (`continue-on-error`, not a required check).
- **RC-7** → ADR-4 / interface §Label catalog contract (`.github/settings.yml` — single declarative source of truth for names, reconciled by the Settings app).
- **RC-8** → plan §Risks (deferred 030 config-guard + publish-time human review).
- **RC-9/RC-10** → spec edge ("no managed label" on no match) + §Authoritative reconciliation (sync correction).
- **RC-11** → plan ADR-2 (pinned action version).
- **RC-12** → plan Cross-cutting (per-event re-reconciliation).
