# Plan: Drafter Config Migration

**Feature**: 071-drafter-config-migration
**Role**: Shaper
**Inputs**: `specs/071-drafter-config-migration/spec.md`, `PROJECT.md`, `.score/memory/DECISIONS.md` (465 lines), `.score/memory/DEPRECATION.md` (25 lines), `.score/memory/LEARNINGS.md` (913 lines, background). No SOUL.md. Additionally read for grounding: `.github/release-drafter.yml`, `.github/workflows/release-drafting.yml`, `internal/build/labelcontract.go`, `internal/build/labelcontract_test.go`, `internal/build/workflow.go`, and the release-drafter v6.4.0 / v7.7.0 sources.

---

## System Architecture

Three artifacts move together, all inside the repository — no runtime component is added and no CLI behavior changes. `internal/build` is a test-only guard package whose "system under test is the pipeline itself" (030 ADR-6 charter); everything here lives there or in `.github/`.

```
.github/release-drafter.yml ──parsed by──> internal/build/labelcontract.go
        (v7 schema)                          ReleaseDrafterConfig
                                                    │
.github/labeler.yml   ─────parsed by───────────────┤ CheckLabelContract
.github/settings.yml  ─────parsed by───────────────┘   (4 label verdicts)

.github/workflows/release-drafting.yml ──parsed by──> internal/build/drafterschema.go
        (`uses:` pinned ref)                            CheckDrafterSchemaCoupling
                                                        (1 version-floor verdict)
                                    ▲
                    reuses Workflow/Job/Step from internal/build/workflow.go
```

**Flow.** `.github/release-drafter.yml` is restructured so the seven category labels sit under each category's `when`, the exclusion moves to a `pre-exclude` category, and the three semver buckets plus the patch fallback become four `version-resolver` categories. `labelcontract.go`'s `ReleaseDrafterConfig` is reshaped to read those positions, and its four existing verdicts are re-derived from the new shape without changing what they assert. A new sibling, `drafterschema.go`, adds one further verdict: the drafting workflow's pinned action major must be at or above the floor the current config schema requires. Both guards run inside the existing `go test ./...` matrix (024 pre-merge, 029 post-merge), so a drift reddens CI rather than silently mis-drafting.

**What is deliberately not built.** Nothing evaluates drafter output. The spec forecloses a synthetic-pull-request harness (Non-Behaviors), so the architecture's whole assurance surface is structural parsing. The guards answer "is the contract still declared correctly" — never "did the draft come out right."

---

## Architecture Decisions

### ADR-1: Hard-switch the guard to the current schema; no dual-shape parser

**Context**: The config and the guard that parses it move in one change (spec System Overview). The guard could be widened to read labels from either the superseded position or the current one, easing any future rollback.

**Options considered**:
1. **Dual-shape parser** — read labels from `categories[].labels` *or* `categories[].when.labels`, whichever is populated. Tolerant of a partial migration; but after the coordinated change no config in the repository exercises the superseded arm, so it becomes permanently untested code, and "accepts either" is the exact looseness a change-detector exists to prevent.
2. **Hard-switch** — the parser reads only the current positions. Every drift is a failure; no dead arm.

**Decision**: Option 2 — hard-switch. The spec records this as a non-behavior, and it conforms to the change-detector rigor 030 ADR-6 established ("a missing label fails as loudly as an extra one").

In practice: `DrafterCategory` loses its direct `Labels` field as the source of contract truth and gains a `When` structure; `VersionResolver`/`ResolverBucket` stop being read from a top-level key and are derived from categories carrying `type: version-resolver`; the exclusion is derived from the `pre-exclude` category rather than `exclude-labels`.

**Consequences**: Rolling back the config alone would redden CI immediately — which is the point, since the config and action version are coupled. It also means the guard cannot help during a partial migration; there is no partial migration, by design. The old-shape *rejection* this implies is not automatic and is handled deliberately in ADR-4.

### ADR-2: The patch fallback becomes a condition-less `version-resolver` category

**Context**: 030 ADR-2 pins a patch bump when no included pull request carries a semver-bearing label, and the guard pins it as `ResolverDefault = "patch"`. The published migration note (PR #1558) documents no replacement for the superseded `version-resolver.default`.

**Options considered**:
1. **Drop the declaration** — rely on the action's built-in fallback, which resolves to patch when no version-resolver category matches. Zero config; but the ADR-2 property becomes implicit in the tool, unguardable, and would change silently on the tool's next major with no diff and no failing test.
2. **A `version-resolver` category with no `when` and `semver-increment: patch`** — the fallback stays declared in the config and therefore stays assertable.

**Decision**: Option 2, matching the spec's clarification. The replacement exists and was verified directly in the action's source rather than inferred: the deprecation warning in `parse-categories.ts` prescribes exactly this migration for `version-resolver.default`, and `resolve-version-increment.ts` implements it — version-resolver categories are evaluated with `emptyWhenBehavior: 'fallback'`, so a condition-less entry supplies the bump when nothing else matches.

Two properties of that code matter downstream and should not be re-derived by guesswork. First, a condition-less **version-resolver** category is not a changelog catch-all: `categorize-pull-requests.ts` searches only `type: changelog` entries for its uncategorized bucket, so the fallback entry never collects note lines. Second, the built-in fallback is also patch, so the declaration is behaviorally redundant *today* — its value is that it keeps the property visible and guarded.

**Consequences**: The guard keeps a `ResolverDefault`-equivalent assertion, now derived from a category rather than a key. Because the declaration is behaviorally redundant, a test that merely observes drafter output could never catch its removal — only the guard can, which is why the drift case for it is mandatory (spec: "removing the declared fallback fails the guard").

### ADR-3: The coupling verdict lives in a new `internal/build/drafterschema.go`, sharing the config parse

**Context**: The spec requires CI to fail when the pinned action version and the config's schema disagree. The existing guard already loads the drafter config from disk; `workflow.go` already parses workflow files into `Workflow`/`Job`/`Step` and already extracts pinned `uses:` refs.

**Options considered**:
1. **Widen `CheckLabelContract`** — add the workflow as a fourth input and a fifth verdict. One entry point; but it conflates two invariants over two different file sets, forces a signature change through both existing tests, and makes every label-contract failure message compete with a version message.
2. **A sibling file with its own loader and check** — `LoadDrafterSchemaCoupling` / `CheckDrafterSchemaCoupling`, reusing `RepoRoot()`, `loadYAML`, the `ReleaseDrafterConfig` type, and `workflow.go`'s `Workflow`/`Step` types.

**Decision**: Option 2. The package convention is one file per concern (`constraintdiscovery.go`, `writesafetyguardrail.go`, `operatororientation.go`); the label contract and the schema/version coupling are separate invariants that happen to share one input file. Sharing the *parse* — the same `ReleaseDrafterConfig`, the same `Workflow` types — satisfies the spec's "extend the existing guard" intent without fusing the two verdicts.

A new `DraftingWorkflowFileName = ".github/workflows/release-drafting.yml"` constant sits alongside the existing `WorkflowFileName` (which is `release.yml` and belongs to 022's guard — the two must not be confused).

**Consequences**: A separate exported check must be *called* by a test or it is dead weight. The task list must include its real-file test, mirroring `TestLabelContract_RealConfig`. Two guard failures can now name the drafter config; each message must say which invariant failed.

### ADR-4: Reject the superseded shape explicitly rather than failing to find labels in it

**Context**: Under ADR-1's hard-switch, a config still written in the superseded shape parses into a `ReleaseDrafterConfig` with empty label sets everywhere. The guard would fail — but with seven "required release-drafter.yml category label missing" messages that describe a symptom, not the cause.

**Options considered**:
1. **Accept the confusing failure** — it fails, which is what matters. Cheapest; but the reader is told labels vanished when in fact the whole file is on the wrong schema, and the obvious "fix" (re-adding `labels:`) is exactly wrong.
2. **Retain the superseded keys as rejection detectors** — keep `version-resolver` and `exclude-labels` (and the category-level `labels`/`label` shorthands) on the struct solely so their *presence* can be reported as "this config is on the superseded schema."

**Decision**: Option 2. This is not accepting both shapes — nothing is ever read *from* the superseded positions for contract purposes. They are parsed only to be refused, with a message naming the schema and the migration.

This is what makes the spec's accord bullet real: "A config that still expresses the contract in the superseded shape fails the guard." It also gives the coupling check in ADR-5 a clean precondition — once the guard passes, the config is unambiguously on the current schema.

**Consequences**: The struct carries fields that exist to be empty, which needs a comment saying so or a future reader will "clean them up" and remove the rejection. Their drift cases must assert the message names the schema, not just that a violation occurred.

### ADR-5: Derive the pinned major from the workflow; the schema floor is a checked-in contract fact

**Context**: The coupling verdict compares two things. The spec's validation scenario requires that neither be a hand-editable literal standing in for a derivable value.

**Options considered**:
1. **Pin the exact expected ref** — assert `uses:` equals a constant like `release-drafter/release-drafter@v7.7.0`. Simple; but it turns every routine patch bump into a guard failure and encodes the wrong property (the exact version, not the schema generation).
2. **Derive the major from the ref and compare against a named floor** — parse the major out of the pinned ref and require it to be at or above `DrafterSchemaMinMajor`.

**Decision**: Option 2. The pinned major is *derived* from the workflow; the mapping "the current config schema requires major N or later" is a contract fact that cannot be derived from anything in the repository, so it is a named constant whose comment states the property — following `GoReleaserVersion`'s precedent in `workflow.go`, where the pin's *reason* is written into the comment so the next person re-derives instead of copying.

Ref-parsing rules: the step is located by its `Uses` prefix, and only a `vN`-leading ref yields a major. A ref that is a branch, a tag without a leading `vN`, or a commit SHA yields no major and **fails loudly** — "cannot determine the pinned major" is a real finding, not a reason to pass.

**Consequences**: A routine patch or minor bump passes untouched; a major downgrade fails. The floor constant is hand-edited at the next schema migration, deliberately — that is the same act as writing the migration. Note the asymmetry this creates, stated plainly in Risks: the guard catches *config newer than action* (silent and catastrophic) and does not catch *action newer than config* (noisy, and degrades through the tool's compatibility mode). That asymmetry is intentional — it guards the failure mode that has no other signal.

### ADR-6: Leave category exclusivity at its default

**Context**: The current major introduces an `exclusive` flag on categories, defaulting to `false`. Whether the migrated config needs it set is a behavior-preservation question, and the repository's own documentation gives the wrong answer.

**Options considered**:
1. **Set `exclusive: true`** — would make each pull request land in only its first matching category. `.github/dependabot.yml`'s header comment asserts this is already how drafting works ("release-drafter assigns the FIRST matching category"), so this reads as the behavior-preserving choice.
2. **Omit `exclusive`** — leave the default.

**Decision**: Option 2, on verified evidence rather than on the comment. Both majors are non-exclusive: the previous major's `releases.js` loops every category over every pull request with an explicit note that a pull request matching several categories "is intended to 'duplicate' the pull request into each category," and the current major's `categorize-pull-requests.ts` breaks out of the category loop only `if (category.exclusive)`. The default therefore *is* the preserved behavior, and 030's own spec agrees ("A pull request carrying more than one managed label appears under each matching category").

**Consequences**: `.github/dependabot.yml`'s comment is wrong and its conclusion ("a gomod PR lands in **Dependencies**") does not follow from the stated reason. Correcting it is not in this feature's scope — it is 028/030 documentation about labeling, and the spec forbids touching label behavior — but it is a live trap for anyone doing this migration, so it is recorded in Risks and flagged for a separate correction.

### ADR-7: Parse `when` in its mapping form only

**Context**: The current schema allows a category's `when` to be either a single condition mapping or a list of them. The guard must decide which forms it understands.

**Options considered**:
1. **Accept both** — a custom unmarshaller normalizing to a slice, following `StringOrSlice`'s precedent in `workflow.go`. Tolerant of either authoring style; but the repository writes exactly one style, so the list arm ships untested unless a fixture is invented for it.
2. **Mapping form only** — the shape the migrated config uses; anything else yields no labels and fails the contract.

**Decision**: Option 1's precedent does not transfer: `StringOrSlice` exists because GitHub's `needs:` is genuinely written both ways *in this repository's own workflows*. Nothing here writes the list form. Take Option 2, and let a switch to the list form fail the guard — forcing a deliberate guard update is the correct outcome, not a false negative.

**Consequences**: A future need for multi-condition categories (say, path-based routing) requires widening the type first. The failure a list-form config produces is a label-set failure, which under ADR-4's reasoning is a symptom-level message; this is accepted because, unlike the whole-file schema case, the config author here has changed exactly the thing the message names.

---

## Migration Strategy

**Current state.** The pin-back landed on `main` (#179, 2026-08-03). `.github/workflows/release-drafting.yml`'s drafting step now pins `release-drafter/release-drafter@v6.4.0`, and `.github/dependabot.yml` carries an `ignore` rule blocking automated major bumps for that action. So the action and the config currently *agree* — both on the superseded schema — and the deprecation warnings have stopped.

That changes this feature's shape in one concrete way: **the workflow's pinned version is now something this change edits, not something it inherits.** Moving the config alone would create exactly the silent degradation the coupling verdict exists to catch. The version move is part of Phase 1, not a precondition of it.

Two things #179 left behind that this feature must reconcile rather than ignore:

- The drafting step carries a long comment explaining why the pin is on the previous major, and it ends with a forward-pointer to this work: *"Migrating to v7 is its own coordinated change: the config restructure plus internal/build/labelcontract.go, which parses `categories[*].labels`, `version-resolver.*` and `exclude-labels` at their v6 paths and will fail loudly when they move."* That comment becomes wrong the moment Phase 1 lands and must be rewritten with it.
- The `ignore` rule's own comment records the repository's procedure for taking a release-path action to a new major: bump by hand, then cut a throwaway pre-release and watch the pipeline end to end. This change *is* that hand bump. The pre-release observation is post-merge and outside this feature's pre-merge assurance surface, but it is now a documented expectation rather than an optional courtesy.

`#179` also added an unrelated semver-tag precondition to the build job, guarded by `TagPreconditionMarker`/`checkTagPrecondition` in `workflow.go`. It stands on its own and is not touched here — but note it as precedent for ADR-5: it derives a *class* (`exit[[:space:]]+[1-9][0-9]*`, any non-zero status) rather than pinning a literal, which is the same discipline the ref-major parse needs.

**Target config shape** (positions only — exact field spelling, ordering, and titles are interface-level):

| Contract element | Superseded position | Current position |
|---|---|---|
| Seven category labels | `categories[].labels` | `categories[].when.labels` |
| Exclusion label | top-level `exclude-labels` | a `type: pre-exclude` category's `when` |
| Three semver buckets | `version-resolver.{major,minor,patch}.labels` | three `type: version-resolver` categories with `semver-increment` + `when` |
| Patch fallback | `version-resolver.default` | a fourth `type: version-resolver` category with `semver-increment: patch` and **no** `when` |

Everything outside that table is untouched: `tag-template`, `name-template`, `version-template`, `change-template`, `template`, `replacers`, and every category `title`. The spec forbids changing display text, and holding it fixed is the only reader-visible signal that routing did not regress.

**Guard changes, verdict by verdict.** The four existing verdicts keep their meaning and their message wording; only their derivation moves. Category labels are gathered from each category's `when` instead of its `labels`. The semver buckets are gathered by grouping `type: version-resolver` categories by `semver-increment`. The exclusion check reads the `pre-exclude` category instead of `exclude-labels`. The managed-count checks over `labeler.yml` and `settings.yml` are untouched — those files do not change.

---

## Cross-cutting Concerns

**Failure messaging.** Every new or moved violation keeps the existing convention: name the file, the section, and the offending value, so a reviewer fixes drift without re-reading the configs. Two message classes are new — "this config is on the superseded schema" (ADR-4) and "the pinned major is below the floor the schema requires" (ADR-5) — and both must name the migration or the floor's reason, not just report a mismatch.

**Testing strategy.** The existing split is preserved and extended: a real-file change-detector (`TestLabelContract_RealConfig`) that parses the shipped `.github/` files, plus a table-driven drift suite over in-memory fixtures that mutate one thing each. The fixtures in `labelcontract_test.go` are rewritten to the current schema, and drift cases are added for every position that moved — a category losing its `when` labels, a missing `pre-exclude` category, a `version-resolver` category with the wrong `semver-increment`, the condition-less fallback deleted, and a config left in the superseded shape. The coupling check gets the same pair: a real-file test against the shipped workflow, and drift cases for a below-floor major and an underivable ref.

Two trap-avoidance notes for whoever writes those tests. The drift table must not assert by map lookup against a zero-valued expectation — a dropped entry would return the zero value and pass; the existing suite avoids this with set-difference and substring assertions, and the new cases must too. And the real-file tests are the only thing standing between a green build and a wrong draft, since nothing exercises the drafter itself.

**Configuration vs. hardcoding.** Exactly one value in this feature is hardcoded on purpose: `DrafterSchemaMinMajor`. Everything else — the pinned major, the label sets, the schema shape — is read from source. The constant's comment carries the property it stands for so the next migration re-derives it.

**Lint.** Reshaping `ReleaseDrafterConfig`'s struct fields re-aligns sibling field columns under gofmt, and CI runs `gofmt -l .` and `golangci-lint run` as a gate separate from `go test`. All three run before pushing; the formatting change belongs in the same commit as the edit that caused it.

---

## Implementation Strategy

All three phases ship as **one pull request**. The phases are build order inside it, not separate merges: the spec's coupling requirement means the config and the guard cannot land apart, and the repository's convention puts contract-doc updates in the same commit as the contract change.

**Phase 1 — the atomic migration.** Move the drafting workflow's pinned action version forward to a major that understands the current schema, and rewrite the pin comment that explains why it was held back; restructure `.github/release-drafter.yml` to the target shape; reshape `ReleaseDrafterConfig`/`DrafterCategory` and re-derive the four verdicts in `labelcontract.go`, including the ADR-4 rejection of the superseded shape; rewrite the `labelcontract_test.go` fixtures. The version move belongs here rather than in Phase 2 because it is one half of the coupling — landing the config without it is the degradation, not a step toward avoiding it. This is the only phase that can redden anything, and it is self-consistent only on completion.

**Phase 2 — the coupling guard.** Add `internal/build/drafterschema.go` with `DraftingWorkflowFileName`, `DrafterSchemaMinMajor`, the ref-major parse, `LoadDrafterSchemaCoupling`, and `CheckDrafterSchemaCoupling`; add its real-file test and drift cases. Depends on Phase 1 only for its precondition (ADR-4 guarantees the config is on the current schema by the time this runs), and is additive — it passes against the workflow as it stands.

**Phase 3 — the live-contract sweep.** Update spec 030's documents that describe the configuration's shape. `plan.md`, `interface-spec.md`, and `validate.md` carry the most references; `analyze.md` and `risk.md` carry a few; `spec.md`'s mentions are tool-agnostic behavioral prose about note categories and need no change — verify rather than assume. Completed `- [x] Txxx` entries in `tasks.md` are history and stay as written; only forward-looking contract statements there change. Depends on Phases 1-2 for the final shape.

---

## Risks

**The dependabot comment is a trap set for this exact change.** `.github/dependabot.yml`'s header states that release-drafter "assigns the FIRST matching category" and reasons from it that a `chore(deps)` pull request lands in Dependencies rather than Internal. Verified false against both majors (ADR-6). *Likelihood*: high that an implementer reads it, since it is the repository's only prose about category routing. *Impact*: setting `exclusive: true` to "preserve" it would silently change every multi-labelled pull request's note placement — a behavior regression introduced by a migration whose whole purpose is preserving behavior. *Mitigation*: ADR-6 records the verification with its sources; the comment's correction is flagged for a separate change and must not be folded in here.

**The guard is one-directional.** It catches a config newer than the pinned action, not the reverse. *Likelihood*: certain — it is a design property, not a defect. *Impact*: if a future major changes the schema again, the repository re-enters exactly today's state (deprecation warnings, compatibility mode) with no CI signal. *Mitigation*: accepted deliberately, because that direction degrades noisily and gracefully while the guarded direction degrades silently and catastrophically. Recorded here so the limit is not mistaken for an oversight.

**Zero deprecation warnings cannot be confirmed before merge.** The drafting workflow triggers only on merge to `main` and blocks nothing. *Likelihood*: certain. *Impact*: a residual warning from a shape the migration adopts — the tool warns on `conventional: {}`, for instance — would surface only after merge. *Mitigation*: the migrated config uses none of the warned-about forms; the first post-merge run is the observation point. Per the spec, no acceptance gate depends on it, so a residual warning is a follow-up, not a rollback.

**The pin-back landed first, so the version move is now inside the blast radius.** *Likelihood*: realized — #179 merged 2026-08-03. *Impact*: this feature must now move the pinned version and the config in the same change. If the two are split across commits or, worse, across pull requests, the intermediate state is precisely the silent degradation this feature exists to prevent — and unlike the pre-#179 situation, that state is now reachable by landing only the config. *Mitigation*: the version move is folded into Phase 1's single task rather than left to Phase 2, and the Phase 2 guard reddens CI on any subsequent attempt to separate them. Phase 2 must still not be deferred to a follow-up pull request.

**Moving the pin forward contradicts a comment that argues against it.** *Likelihood*: certain. *Impact*: #179's pin comment states the empirical case for staying on the previous major (v6 tagged correctly 3/3, v7 failed 1/1) and explicitly says migrating the config is *not* known to fix the tag loss. Leaving it in place while raising the pin would put a live argument against the change directly above the change. *Mitigation*: Phase 1 rewrites the comment. The rewrite must preserve the empirical record — it is the reason the pre-release observation is expected — while removing the now-stale forward-pointer to "a coordinated change" that has happened. This is the one place where "do not claim to fix the tag loss" has to be written down at the point a reader will most expect a claim.

---

## What This Plan Does Not Cover

- **Exact YAML field spelling, category ordering, and Go field names** — the structural contract for both the config file and the reshaped `ReleaseDrafterConfig` is `/score:interface`'s output.
- **Executable scenarios** — the spec's seven driving scenarios become feature files in `/score:scenarios`.
- **Task decomposition** — the three phases become PR-sized units in `/score:tasks`.
- **The untagged-release failure** — explicitly foreclosed by the spec. No decision here bears on it, and the pull request must say so.
- **Whether the `ignore` rule blocking automated major bumps for this action should be removed** — #179 answered the policy question this plan originally recorded as open: automated majors are blocked, and majors are taken by hand. This feature performs a hand bump *within* that policy and does not revisit it. Note the interaction: with the Phase 2 guard in place, a future automated major that broke the schema would fail CI, which weakens the case for keeping the rule — but that is a decision for whoever next touches the release-path pinning policy, not a consequence of this change.
- **Correcting `.github/dependabot.yml`'s category-routing comment** — a real defect found while planning, out of scope here (see Risks). Note that Phase 1 *does* rewrite a different comment in `release-drafting.yml`; the two are separate, and the category-routing one stays as it is.
- **The semver-tag precondition on the build job** — landed in #179 as `TagPreconditionMarker`/`checkTagPrecondition` in `workflow.go`. Unrelated to the config schema and independently useful. Not modified here.
- **The throwaway pre-release observation** — the repository's recorded procedure for a hand-made major bump. Post-merge, outside this plan's pre-merge scope, and deliberately not turned into an acceptance gate (see spec Non-Behaviors). Named here so it is not mistaken for an oversight.
