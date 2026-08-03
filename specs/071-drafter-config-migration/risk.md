# Risk: Drafter Config Migration

**Feature**: 071-drafter-config-migration
**Round**: 1.5 — re-baselined the same day against #179's merge (see Residual Risk Summary)
**Generated**: 2026-08-03
**Artifacts loaded**: spec.md, plan.md (§ System Architecture, ADRs, Risks), interface-spec.md, PROJECT.md
**Degradation flags**: none — the full upstream set was available.
**Acceptability matrix**: default 3×3 traffic light. PROJECT.md defines no project-level risk acceptability matrix.
**Regulatory bridge**: not included. PROJECT.md declares no Regulatory Context.

---

## Risk Register

| ID | Hazard | Source | Sev | Prob | Level | Controls | Residual |
|---|---|---|---|---|---|---|---|
| H-1 | A current-schema config runs on an action major that predates it — accepted, silently ignored, wrong notes and wrong bump | spec § Integration Boundaries; plan § System Architecture | High | High | Red | RC-1, RC-2 | **Yellow** |
| H-2 | The declared patch fallback is deleted later as redundant, un-pinning 030 ADR-2 | spec § Non-Behaviors; plan ADR-2 | Med | Low | Green | RC-3, RC-4, RC-5 | **Green** |
| H-3 | `exclusive: true` is added on the strength of `dependabot.yml`'s incorrect comment, changing note placement for every multi-labelled PR | plan § Risks; plan ADR-6 | High | Med | Red | RC-6, RC-7, RC-8 | **Yellow** |
| H-4 | A routing regression ships because nothing verifies drafter output | spec § Non-Behaviors; plan § System Architecture | High | Low | Yellow | RC-8, RC-9, RC-10 | **Yellow** |
| H-5 | Residual deprecation warnings after merge go unread — the feature's headline outcome silently unmet | spec § Behavioral Accord > Deprecation surface | Low | Med | Green | RC-11 | **Green** |
| H-6 | The 030 doc sweep rewrites historical task records, destroying the audit trail | spec § Non-Behaviors; plan § Phase 3 | Med | Low | Green | RC-12, RC-13 | **Green** |
| H-7 | The version move and the config move are separated across commits or pull requests | plan § Migration Strategy, § Risks | High | Med | Red | RC-1, RC-2, RC-14 | **Yellow** |
| H-8 | A future major changes the schema again with no CI signal | plan ADR-5 | Med | Low | Green | RC-15, RC-19 | **Green** |
| H-9 | The change is read as fixing the untagged-release failure, so the real cause goes uninvestigated | spec § Non-Behaviors | Med | Med | Yellow | RC-16, RC-17 | **Yellow** |
| H-10 | A managed label is dropped or duplicated while moving between config positions | spec § Non-Behaviors; interface § target structure | High | Low | Yellow | RC-9, RC-18 | **Yellow** |

**0 Red · 6 Yellow · 4 Green**

---

## Hazard Detail

### H-1 — Current-schema config on a pre-schema action major

The defining hazard, and the reason the coupling guard exists. An action major that predates the config's schema does not reject it: verified in the previous major's source, config validation admits unknown keys, so `type`, `when`, and `semver-increment` are accepted and discarded. The result is empty label sets (no note routing), an absent resolver (every bump becomes patch — `breaking` and `features` stop mattering), and an absent exclusion (spec and planning churn appears in the notes).

**Severity High** — a wrong bump that reaches a published release cannot be unwound; a version number is consumed. Blast radius is every consumer of the release notes and the version.
**Probability High (pre-control)** — raised from Medium after #179 merged on 2026-08-03. `main` now pins the previous major, so the degraded state is no longer something an unrelated pull request might cause: it is what this feature produces by default if the config lands without the version move. The hazard sits on the change's own happy path, not off to one side of it.
**Controls**: RC-1, RC-2.
**Residual Yellow** — High severity × Low probability once RC-1 is in place. Accepted: the guard converts a silent runtime degradation into a red CI check, which is the strongest control available given that nothing exercises the drafter before merge.

### H-2 — The declared fallback is deleted as redundant

`plan` ADR-2 keeps the patch fallback declared even though the action's built-in fallback is also patch. That makes the declaration behaviorally inert *today* — which is precisely what invites a future reader to remove it as noise. Its removal would change no drafter output and would silently un-pin 030 ADR-2.

**Severity Medium** — no immediate harm; harm arrives only if the tool's built-in default ever changes, at which point label-less releases bump wrongly with no diff to point at.
**Probability Low** — three independent controls, one of which fails loudly.
**Controls**: RC-3, RC-4, RC-5.
**Residual Green.**

### H-3 — `exclusive: true` added on bad documentation

`.github/dependabot.yml`'s header states that release-drafter "assigns the FIRST matching category" and reasons from it that a `chore(deps)` pull request lands in Dependencies rather than Internal. Verified false against both majors — both push a pull request into every matching category. It is the repository's only prose about category routing, so an implementer doing this migration is likely to read it and "preserve" a behavior that never existed.

**Severity High** — silently changes note placement for every multi-labelled pull request, introduced by a change whose entire purpose is behavior preservation. It would also look like a fix rather than a regression.
**Probability Medium** — the misleading text is in the file most adjacent to this work and is not corrected by this feature.
**Controls**: RC-6, RC-7, RC-8.
**Residual Yellow** — accepted, with the note that all three controls are documentary. The strongest real fix (correcting the comment) is deliberately out of scope, so the residual stays yellow rather than green.

### H-4 — A routing regression ships undetected

`spec` § Non-Behaviors forecloses runtime verification: no synthetic-pull-request harness, no gate on the drafting run. The guards assert that the label *sets* still agree; nothing asserts that a labelled pull request still lands in the right section.

**Severity High** — wrong release notes reach users, and the bump could be wrong with them.
**Probability Low** — the change moves labels between positions without touching routing semantics, the `exclusive` default was verified equivalent across both majors, and display text is frozen.
**Controls**: RC-8, RC-9, RC-10.
**Residual Yellow** — accepted deliberately. The spec states this limit rather than hiding it, and the alternative (re-implementing the tool's routing in CI to predict its output) would duplicate the logic the config delegates.

### H-5 — Residual deprecation warnings go unread

The zero-warnings outcome is checked by nobody: CI cannot see it (the workflow is `push: main`-triggered) and the spec assigns no acceptance gate to the post-merge run.

**Severity Low** — a warning is cosmetic; the notes and version remain correct.
**Probability Medium** — no observer is named, so the default outcome is that nobody looks.
**Controls**: RC-11 (prevention only, not detection).
**Residual Green** on real-world impact. Recorded here because checklist F2 flags the same subject at P1 for a different reason — the spec presents it as a guarantee while resting on inspection. The risk is low; the honesty gap is the finding.

### H-6 — The doc sweep destroys 030's audit trail

Phase 3 updates six documents in a directory that also contains completed task records. A blanket find-and-replace across `specs/030-release-drafting/` would rewrite `- [x] Txxx` entries to match today's shape, misrepresenting history as contract.

**Severity Medium** — recoverable from git, but the loss is silent and may not be noticed for months.
**Probability Low** — T005 names the boundary explicitly and carries it as a risk line.
**Controls**: RC-12, RC-13.
**Residual Green.**

### H-7 — The version move and the config move get separated

Restated at round 1.5. This hazard was originally "PR #179 pins the action back below the schema floor," assessed at Low probability because either merge ordering was covered. #179 merged on 2026-08-03, so the ordering question is settled and the hazard has changed shape rather than closed: the pinned version is now something this change *edits*, and the exposure is the change splitting its two halves.

Three ways the split happens in practice: the version move is deferred to a follow-up pull request; T001 is committed in pieces and an intermediate commit is what gets reviewed or cherry-picked; or a later revert takes the config back without the version, or the version without the config.

**Severity High** — any of the three reproduces H-1 exactly.
**Probability Medium** — the two halves live in different files with no compiler or test relationship between them until T003 exists, and "revert the config" is the obvious first instinct if a draft comes out wrong.
**Controls**: RC-1, RC-2, RC-14.
**Residual Yellow** — RC-1 closes it once T003 lands, but note the gap it cannot close: within T001 itself, before the coupling guard exists, nothing mechanical enforces the pairing. That window is bounded by a single task in a single pull request, which is why the plan folds the version move into T001 rather than giving it its own task.

### H-8 — A future major changes the schema again with no CI signal

The coupling guard is one-directional by design: it catches config-newer-than-action and not action-newer-than-config.

Downgraded at round 1.5. The original assessment rated this Medium probability because no rule prevented an automatic major bump. #179 added one — dependency automation is now blocked from raising this action's major, and the recorded procedure is a deliberate hand bump followed by a throwaway pre-release watched end to end. The uncaught direction is therefore no longer reachable without a human deciding to go there.

**Severity Medium** — the repository would return to a familiar state: deprecation warnings and compatibility-mode operation, with output most likely still correct.
**Probability Low** — reaching it requires a deliberate hand bump that skips the recorded pre-release observation.
**Controls**: RC-15 (disclosure), RC-19 (the automation barrier).
**Residual Green.** Note the dependency: this rating rests entirely on the `ignore` rule staying in place. If it is ever removed — the plan lists that as an open question this feature does not settle — this hazard returns to Yellow and becomes the register's weakest-controlled entry again.

### H-9 — The change is read as fixing the untagged-release failure

The migration coincides in time with the placeholder-tag publish failure and touches the same subsystem. If it is described — or merely remembered — as the fix, the real cause goes uninvestigated and a still-broken publish path is treated as solved.

**Severity Medium** — v0.2.2-class publish failures recur, each costing a release cycle.
**Probability Medium** — the association is natural and the two changes are adjacent in the log.
**Controls**: RC-16, RC-17.
**Residual Yellow** — the controls are a spec non-behavior and a validation scenario, and checklist F1 establishes that the scenario cannot be executed. The control is therefore review-time only.

### H-10 — A label is dropped or duplicated in the position move

Twelve category entries replace three top-level structures; every one of the eight managed labels changes position.

**Severity High** — a dropped label removes a note section or a semver trigger.
**Probability Low** — four guard verdicts compare by set difference, and a missing label fails as loudly as an extra one.
**Controls**: RC-9, RC-18.
**Residual Yellow** — High severity keeps it off Green even at low probability. This is the hazard the existing guard is strongest against.

---

## Controls

| ID | Control | Grounding |
|---|---|---|
| RC-1 | A CI verdict holds the drafting workflow's pinned action major at or above the floor the config's schema requires, deriving both the pinned major and the floor's applicability rather than assuming them | plan ADR-3, ADR-5; interface § `drafterschema.go` |
| RC-2 | Config, guard, and docs ship as one pull request, so the repository never sits between schema states | spec § Schema/version coupling; tasks preamble |
| RC-3 | The guard asserts the fallback's presence and increment, failing when the declaration is absent rather than when a value mismatches | plan ADR-2; interface § Error Communication |
| RC-4 | A mandatory drift case covers deletion of the fallback specifically, because its removal changes no observable output | tasks T002 |
| RC-5 | `ResolverDefault`'s doc comment records the property it pins and its trace to 030 ADR-2, so the constant is not renamed or removed as cosmetic | interface § `labelcontract.go`; tasks T001 |
| RC-6 | The `exclusive` default is verified equivalent across both action majors from source, with the sources named, rather than inferred from repository prose | plan ADR-6 |
| RC-7 | The task that edits the config carries the misleading-documentation trap as an explicit risk line at the point of use | tasks T001 |
| RC-8 | The config's header comment states that `exclusive` is deliberately omitted, so a later reader does not add it | interface § Instructional surface |
| RC-9 | Four label-contract verdicts compare declared sets against the closed managed set by set difference, in all three files | plan § Migration Strategy; interface § Error Communication |
| RC-10 | Display text — titles, templates, replacers — is frozen byte-identical, preserving the only reader-visible signal that routing did not change | spec § Non-Behaviors; tasks T001 |
| RC-11 | The config is constructed to avoid every form the action warns about, enumerated as forbidden fields per entry kind | interface § target structure; tasks T001 |
| RC-12 | The doc sweep distinguishes live contract from historical record, leaving completed task entries untouched | spec § Non-Behaviors; tasks T005 |
| RC-13 | A whole-directory grep for the superseded keys confirms every surviving occurrence is deliberate history | tasks T005 |
| RC-14 | The pending pin-back's disposition is settled in the spec rather than left to merge-time sequencing | spec § Clarifications |
| RC-15 | The guard's one-directional coverage is disclosed in the plan and the ADR rather than presented as complete | plan ADR-5, § Risks |
| RC-19 | Dependency automation is blocked from raising this action's major, and the recorded procedure for a hand bump requires watching a throwaway pre-release through the pipeline end to end | `.github/dependabot.yml` `ignore` rule and its comment (landed in #179); plan § Migration Strategy |
| RC-16 | A spec non-behavior forbids the artifact and the pull-request description from claiming or implying the fix | spec § Non-Behaviors |
| RC-17 | A validation scenario asserts the absence of the claim (review-time; see checklist F1) | feature file, `@validation` |
| RC-18 | A validation scenario pins set equality between the config's labels and 028's managed set | feature file, `@validation` |

---

## Residual Risk Summary

No hazard is Red after controls. Six are Yellow and accepted with the justifications above; four are Green.

Two Yellows deserve attention beyond their rating:

- **H-7** is the register's live one. It was Green before #179 merged and is Yellow after, because the pinned version moved from being inherited to being edited. Its residual gap is narrow but real: inside T001, before the coupling guard exists, nothing mechanical pairs the version move with the config move. The mitigation is structural rather than automated — one task, one pull request — so it depends on the decomposition being followed rather than on a check.
- **H-3** carries three controls, all documentary. The intervention that would actually remove it — correcting `.github/dependabot.yml`'s category-routing comment — is deliberately out of scope. That is a defensible scope boundary, but it means the control set cannot get stronger without crossing it.

**Round 1.5 re-baseline (2026-08-03).** #179 merged after this register was first written, moving `main` from the new major with a superseded config to the previous major with a superseded config, and adding an automation barrier on major bumps. Three ratings changed as a result: **H-1** probability Medium → High (the degradation is now on this change's default path, not off to one side), **H-7** restated and Green → Yellow (the hazard is now the change splitting its own halves), and **H-8** Yellow → Green (RC-19 added). No hazard was removed and one control was added. The remaining seven entries are unaffected — they concern the config's shape and the guard's derivation, neither of which #179 touched.

**Cross-reference**: H-5 and H-9 both have Guardian findings elsewhere (checklist F2 and F1 respectively). In both cases the real-world risk is modest while the artifact-level problem is real — the spec overstates a guarantee (H-5) and a control is weaker than the artifact implies (H-9). The findings and the ratings are consistent, not in tension.

---

## Traceability Index

| Hazard | Source section |
|---|---|
| H-1, H-7 | spec § Integration Boundaries (release-drafter action); plan § System Architecture, § Migration Strategy |
| H-2 | spec § Non-Behaviors (built-in fallback); plan ADR-2 |
| H-3 | plan ADR-6, § Risks |
| H-4 | spec § Non-Behaviors (no runtime verification) |
| H-5 | spec § Behavioral Accord > Deprecation surface |
| H-6 | spec § Non-Behaviors (task records); plan § Phase 3 |
| H-8 | plan ADR-5, § Risks; `.github/dependabot.yml` `ignore` rule (landed in #179) |
| H-9 | spec § Non-Behaviors (untagged-release failure) |
| H-10 | spec § Non-Behaviors (label set); interface § target structure |

| Control | Architectural grounding |
|---|---|
| RC-1, RC-15 | plan ADR-3, ADR-5 — `internal/build/drafterschema.go` |
| RC-2, RC-14 | plan § Migration Strategy, § Implementation Strategy |
| RC-3, RC-5, RC-9 | plan ADR-1, ADR-2 — `internal/build/labelcontract.go` |
| RC-4, RC-7, RC-11, RC-12, RC-13 | tasks T001, T002, T005 |
| RC-6, RC-8, RC-10 | plan ADR-6; interface § Instructional surface, § target structure |
| RC-16, RC-17, RC-18 | spec § Non-Behaviors; feature file `@validation` scenarios |
