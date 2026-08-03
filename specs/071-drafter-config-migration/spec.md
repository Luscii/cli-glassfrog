# Specification: Drafter Config Migration

**Feature**: 071-drafter-config-migration
**Role**: Definer
**Tier**: 1 (zero setup)

---

## System Overview

Release Drafting (030) keeps a single draft GitHub Release current as changes land on `main` — the next semantic version resolved from pull-request labels, and the merged-PR titles grouped into seven note categories. It realizes that behavior through the release-drafter action, configured by `.github/release-drafter.yml`, and the label strings in that file are one leg of a three-file contract (with `.github/labeler.yml` and `.github/settings.yml`) that an in-repo guard asserts on every CI run.

release-drafter's new major reorganized its configuration schema: the label predicates, the exclusion list, and the semver resolver all move into a single `categories` list. The repo's config still expresses the contract in the superseded shape, so every drafting run emits schema-deprecation warnings. This feature realigns the config, the guard that parses it, and Release Drafting's live-contract documentation onto the current schema — **preserving every drafting behavior 030 specifies, changing only how that behavior is expressed.** It is a coordinated change, not a config edit: the guard reads the config structurally, so the two must move together, and — because an action major that predates the config's schema accepts it and silently ignores what it does not recognize — the pinned action version becomes a third thing the guard must hold in step.

---

## Behavioral Accord

### Drafting outcomes (preserved, not changed)

- When the drafter runs after the realignment, the draft it produces for a given set of merged pull requests is equivalent to the draft the superseded config produced: same proposed version, same note categories, same lines, same exclusions.
- When several included pull requests carry semver-bearing labels, the bump still resolves highest-wins — **breaking** forces major, else **features** forces minor, else **fixes** forces patch.
- When no included pull request carries a semver-bearing label, the bump is still **patch**, and that fallback is **declared in the config** rather than inherited from the tool's built-in behavior.
- When a merged pull request carries the exclusion label, it is still absent from the notes and ignored for the bump.

### Deprecation surface

- When the drafter runs against the realigned config, it emits no configuration-schema deprecation warnings.

### Contract guard

- When the guard runs, it still reaches the same four verdicts over the same three files: the seven category labels agree across all three; the semver buckets are exactly breaking/features/fixes; the exclusion label is present in all three; and the two catalog files each manage exactly the eight managed labels.
- When the guard reads the config, it derives every asserted value from the config's current shape. A config that still expresses the contract in the superseded shape fails the guard.
- When a category loses its label predicate, when the exclusion declaration is missing, when a semver-resolving entry carries the wrong increment, or when the declared patch fallback is absent, the guard fails and names the offending file and value.
- When the guard runs in CI, it reads the real config files from the repository, not only a fixture.

### Schema/version coupling

- The guard additionally holds the drafting workflow's pinned action version in step with the config's schema. It derives both sides from their own source — the pinned version from the workflow, the schema generation from the config's own shape — rather than pinning either as a literal.
- When the pinned action version would not understand the schema the config is written in, the guard fails in CI and names both the pinned version and the schema the config expects.
- The repository therefore never merges a state in which the two disagree. This matters because the mismatch has no runtime symptom: the drafting run reports success while producing miscategorized notes and a wrong bump.

### Documentation

- When Release Drafting's live-contract documents describe the configuration, they describe the current shape. Completed implementation records remain as written — they are history, not contract.

---

## User Scenarios

**In order to** stop every drafting run reporting that the release configuration is written against a superseded schema,
**as a** maintainer,
**I want** the configuration expressed in the schema the running action actually reads.

**In order to** keep trusting the draft release without re-reading it,
**as a** maintainer,
**I want** the realignment to move labels between configuration positions and nothing else, with the three-file label contract still asserted in CI as the standing check.

**In order to** never again discover a schema mismatch only by reading a wrong release draft,
**as a** maintainer,
**I want** CI to fail when the pinned action version and the configuration's schema disagree.

---

## Non-Behaviors

- This feature must not claim, attempt, or be reported as a fix for the untagged-release failure, in which a drafted release was published carrying a placeholder tag instead of a semantic version. **Why**: that failure lives in the release payload, not the config schema; the two coincided in time but no causal link has been established. Bundling an unproven fix into a behavior-preserving migration would make a later regression un-bisectable and would let a still-broken publish path be declared solved.
- The guard must not accept both the superseded and the current shape. **Why**: the config and the guard move in one change, so a dual-shape parser would be permanently untested dead code — and "accepts either" is precisely the looseness a change-detector exists to prevent.
- This feature must not rely on the tool's built-in fallback to supply the patch default. **Why**: 030 ADR-2 pins a patch bump when no semver-bearing label is present. An undeclared default is unguardable and would change silently on the tool's next major, with no diff and no failing test.
- This feature must not verify drafting behavior at runtime — no synthetic pull-request fixtures driven through the drafter in CI, and no gate on a drafting run. **Why**: the drafting workflow triggers only on merge to `main` and blocks nothing, so any pre-merge harness would have to re-implement the tool's routing and resolution logic in order to predict its output — duplicating the very logic the config delegates. Behavior preservation therefore rests on the structural contract the guard asserts, not on observed output. This is a deliberate limit, stated so nobody later mistakes a green guard for a verified draft.
- This feature must not add, remove, rename, or re-scope any managed label, nor change which pull requests receive which. **Why**: PR Administration (028) owns the label set; Release Drafting only reads it. Moving that ownership here would fork the contract the guard exists to hold.
- This feature must not change the notes' display text — section titles, the change line format, the header, or the replacers. **Why**: a diff in the rendered notes would mask whether a routing regression had occurred, and given no runtime verification exists (above), an unchanged rendering is the only remaining signal a reader has.
- This feature must not revisit whether dependency automation may raise this action's major version again. **Why**: the repository already answers it — automated majors for this action are blocked, and majors are taken by hand. This work performs a hand bump *within* that policy. Loosening or tightening the policy is a standing question about how the repository absorbs breaking tool changes, and settling it inside a behavior-preserving edit would smuggle a policy change into a migration.
- This feature must not rewrite completed implementation records in Release Drafting's task list. **Why**: those record what was done and when; editing them to match today's shape destroys the audit trail and misrepresents history as contract.

---

## Integration Boundaries

- **release-drafter action (external dependency)**: reads the config on every merge to `main`. Its major version determines which config schema is understood; a config written for a schema the running major predates is accepted and silently ignored rather than rejected, which is why the coupling is asserted in the repository instead of relied on at runtime.
- **Drafting workflow definition (edited, and a new guard input)**: the workflow file carrying the pinned action version. This work moves that version forward so it understands the realigned config, and the guard then reads it to reach the coupling verdict. Nothing else about the workflow is this feature's concern.
- **PR Administration (028) labels (input, unchanged)**: the seven category labels and the exclusion label. Read, never written.
- **`labeler.yml` and `settings.yml` (contract siblings)**: the other two legs of the label contract. Untouched by this feature, still asserted by the guard.
- **In-repo label-contract guard (CI)**: parses the drafter config structurally from disk and fails the build on drift. It is the reason this is a coordinated change, and it is the only pre-merge assurance this feature carries.
- **Release Drafting (030) specification artifacts**: the live-contract documents that describe the configuration shape; updated here. The completed task record is not.
- **Drafting run on `main` (post-merge, unguarded)**: the workflow's failure blocks nothing and it never runs pre-merge, so the first run after merge is where a mistake would surface — as a wrong draft, not as a failed job. No acceptance gate depends on it.

---

## Driving Scenarios

### Happy path

**Scenario: a feature merge still bumps minor and files under Features**
Given the last published release is `v1.2.0`
And the configuration has been realigned to the current schema
When a pull request labelled **features** merges to `main` and the drafter runs
Then the draft's proposed version is `v1.3.0`
And the pull request's title appears under the **Features** category.

**Scenario: a drafting run reports no schema deprecations**
Given the configuration has been realigned to the current schema
When the drafter runs
Then it completes without emitting any configuration-schema deprecation warning.

**Scenario: the declared fallback supplies the patch bump**
Given the last published release is `v1.2.0`
And no included merged pull request carries a semver-bearing label
When the drafter runs
Then the draft's proposed version is `v1.2.1`
And the patch fallback is the one declared in the configuration, not the tool's built-in default.

### Error scenarios

**Scenario: a category losing its label predicate fails the guard**
Given a category in the configuration no longer names any label
When the guard runs
Then it fails
And the message names the configuration file and the label that has gone missing from the contract.

**Scenario: removing the declared fallback fails the guard**
Given the declared patch fallback is deleted from the configuration
When the guard runs
Then it fails
And it does not pass merely because the tool would have fallen back to patch anyway.

### Edge cases

**Scenario: the pinned action version falling behind the config schema fails the guard**
Given the configuration is written in the current schema
And the drafting workflow pins an action version that predates that schema
When the guard runs
Then it fails and names both the pinned version and the schema the configuration expects
And the mismatch is caught before merge rather than by a drafting run, which would report success while producing miscategorized notes and a wrong bump.

**Scenario: exclusion survives the realignment**
Given a merged pull request carries the exclusion label
And the configuration has been realigned to the current schema
When the drafter runs
Then the pull request's title appears nowhere in the draft notes
And it does not contribute to the version bump.

---

## Validation Scenarios

> These are held out from the implementing agent for independent verification.

**Scenario: the artifact makes no claim about the untagged-release failure**
Given the specification and the eventual pull-request description
When read end to end
Then neither asserts nor implies that this change fixes the placeholder-tag publish failure.

**Scenario: no label is invented or dropped by the realignment**
Given every label named in the realigned configuration
When compared against PR Administration (028)'s managed set
Then the sets match exactly — the migration moves labels between config positions and introduces none.

**Scenario: the pre-existing assertions survive in number and strictness**
Given the guard's four label-contract verdicts before the change
When compared against the guard after the change
Then all four remain, each still failing on a missing value as loudly as on an extra one
And the coupling verdict is additional to them, not a replacement for any.

**Scenario: neither side of the coupling verdict is a hard-coded literal**
Given the guard's coupling check
When its two inputs are traced
Then the pinned version is read from the workflow and the schema generation is derived from the configuration's own shape
And neither is written into the guard as a fixed value that would need hand-editing on the next migration.

---

## Assumptions

- **The tool is named deliberately**: unlike 030, which specified Release Drafting tool-agnostically, this specification names release-drafter and its configuration schema directly. The schema *is* the subject of the change; abstracting it away would leave nothing to specify.
- **The current schema does provide a declarable fallback bump**: verified in the action's source — its own deprecation message directs the superseded `default` bump to a semver-resolving category carrying no condition, and the version-resolution logic treats a condition-less resolver entry as the fallback. (The published migration note omits this; the source and schema were read directly.)
- **The superseded shape degrades silently rather than failing**: verified in the previous major's source — its config validation admits unknown keys, so the current schema's structures would be accepted and ignored, yielding empty label sets, a patch-only resolver, and no exclusion. (This is what makes the coupling a correctness requirement rather than a tidiness one.)
- **Behavior equivalence is judged against 030's Behavioral Accord**, not against a byte-comparison of drafter output. Whitespace or ordering differences that leave version, categories, and membership identical are equivalent.
- **The realignment moves both the config and the pinned version**: `main` pins the action to the previous major and runs the superseded config against it, so the two currently agree. This work moves both forward together (see Clarifications). An earlier reading of this specification assumed the action was already on the new major; that ceased to be true when the pin-back landed, and the coupling requirement is what makes the version change part of this feature rather than a separate step.
- **Raising this action's major is a deliberate, hand-made change**: the repository now records that policy directly — dependency automation is blocked from raising it, and the recorded procedure for taking it to a new major is to bump it by hand and then watch a throwaway pre-release through the pipeline end to end. This work performs that hand bump. The post-merge observation that procedure calls for sits outside this feature's pre-merge assurance surface and does not change the Non-Behavior below.
- **No deprecation-warning count is pinned**: the acceptance condition is *zero* warnings, not a reduction from a specific number. Counting them invites a change-detector that drifts with the tool's own messaging.

---

## Ambiguity Warnings

None remaining — the coupling-enforcement mechanism, the assurance boundary, and the pending pin-back's disposition were all resolved during clarification (see Clarifications).

---

## Clarifications

### Session 2026-08-03

- **Coupling enforcement**: the config-schema/action-version mismatch is caught by extending the existing label-contract guard to read the drafting workflow's pinned version, failing CI on disagreement — not by review discipline. The failure mode is silent at runtime, so a human check was judged the wrong instrument. (Behavioral Accord — Schema/version coupling; new edge-case driving scenario; new validation scenario on deriving both sides from source.)
- **Assurance boundary**: behavior preservation rests on the structural contract the guard asserts. No runtime verification is added — no synthetic-pull-request harness in CI, and no acceptance gate on the post-merge drafting run. Recorded as an explicit non-behavior with its reasoning, and the "provably behavior-preserving" framing was removed from the user scenarios so the limit is not overstated. (Non-Behaviors; Integration Boundaries; User Scenarios.)
- **Pin-back disposition**: at clarification time a change returning the action to the previous major was open, and the decision recorded here was that this specification owns the action version and moves it forward. That change has since landed in full, along with a rule blocking automated major bumps and an unrelated semantic-version precondition on the build job (which stands on its own and is not touched here). The decision is unchanged by that landing — it only changes the starting point: this work now moves the pin forward from the previous major rather than leaving it where it stood. (Assumptions.)
- **Future major bumps**: left to the repository's own policy rather than settled here. At clarification time no such policy existed and the question was recorded as deliberately open; it has since been answered on `main` — automated majors for this action are blocked and majors are taken by hand. The clarification's intent is unchanged: this migration does not decide the policy, it operates inside it. (Non-Behaviors; Assumptions.)
