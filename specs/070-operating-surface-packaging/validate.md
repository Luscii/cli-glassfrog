# Validate: Operating-Surface Packaging

**Feature**: 070-operating-surface-packaging
**Round**: 1 of 3
**Date**: 2026-07-21
**Verdict**: Ready
**Artifacts loaded**: spec.md, plan.md, tasks.md, interface-spec.md, features/unequipped-agent-operators/operating-surface-packaging.feature, PROJECT.md
**Implementation files**: 5 (`.claude-plugin/marketplace.json`, `plugin/skills/glassfrog-setup/SKILL.md`, `internal/build/operatingsurfacepackaging.go`, `internal/build/operating_surface_packaging_guard_test.go`, `internal/build/operating_surface_packaging_bdd_test.go`) + docs (README.md, `docs/guides/agent-operators/how-to-install-the-operating-surface.md`)

---

## Conformance Summary

| Dimension | Status | Findings |
|---|---|---|
| Driving scenario coverage | ✓ Pass | 0 |
| Acceptance criteria | ✓ Pass | 0 |
| Interface contract conformance | ✓ Pass | 0 |
| Non-behavior absence | ✓ Pass | 0 |
| @wip lifecycle completion | ✓ Pass | 0 |
| **Validation scenarios** | ✓ Satisfied | 0 |

**Total**: 5 dimensions checked, 5 passed, 0 findings. 4 of 4 validation scenarios satisfied.

Supplementary test execution: `TestOperatingSurfacePackagingFeatures` (13 scenarios), `TestMarketplaceConsistencyGuard`, `TestSetupSkillDriftGuard` — 16 tests pass. Full `go test ./...` green (2119), `gofmt -l .` clean.

---

## Driving Scenario Coverage

**Status**: Pass (9 of 9 driving scenarios covered)

| Scenario | Status | Implementation |
|---|---|---|
| Marketplace add lists the glassfrog plugin | ✓ Covered | `.claude-plugin/marketplace.json`; `FindMarketplacePlugin` (operatingsurfacepackaging.go); BDD `thenManifestAtPath`/`thenListsInstallableEntry` |
| Install brings the plugin's surface into the environment | ✓ Covered | `MarketplaceSourcePluginManifest`; BDD `thenSurfaceAvailable` traces skills/agents/hooks under the resolved `./plugin` source |
| Marketplace entry drift is a defect | ✓ Covered | `MarketplaceSourcePluginManifest` (error on unresolvable source) + `CheckMarketplaceEntryConsistency`; BDD `thenTreatedAsDefect`/`thenNotTolerable` |
| Guard fails when a version pin appears on the marketplace entry | ✓ Covered | `MarketplacePluginEntry.UnmarshalJSON` (HasVersion) + `CheckMarketplaceEntryConsistency`; BDD `thenReportsSingleSourced` |
| A sibling plugin is one appended entry | ✓ Covered | `appendSibling` re-parse through `ParseMarketplaceManifest`; BDD `thenListedAsAdditionalEntry`/`thenNoRestructuring` |
| Ready environment is reported ready | ✓ Covered | SKILL.md Ready-report section; BDD `thenReportsReady` (conditioned on both checks) |
| Missing CLI routes to the install channels | ✓ Covered | SKILL.md Missing-CLI section; BDD `thenReportsCLIMissing`/`thenDirectsToChannels` |
| Failing credential routes to the CLI's own setup | ✓ Covered | SKILL.md Failing-credential section; BDD `thenGuidesXAuthTokenSetup` |
| Setup re-checks after a fix instead of assuming success | ✓ Covered | SKILL.md re-check discipline; BDD `thenRechecksPresenceAfterFix`/`thenFailingRecheckRoutesBack` |

---

## Acceptance Criteria

**Status**: Pass (3 of 3 checked tasks; all criteria met)

| Task | Status | Evidence |
|---|---|---|
| T001 — marketplace manifest + consistency guard | ✓ Met | Manifest has `name:"glassfrog"`, `owner:{name:"Luscii"}`, `$schema`, `plugins[]`; single entry `name:"glassfrog"`, `source:"./plugin"`, `description` verbatim-equal to `plugin.json`, no `version`. `operatingsurfacepackaging.go` exports path constant + parse helpers (production source). Guard reads both sides from disk, existence-based lookup, resolution + identity + no-version assertions; partial-coverage stated in the test's COVERAGE comment. |
| T002 — document the install flow | ✓ Met | README "Install the agent operating surface" (marketplace add → plugin install), CLI stated as prerequisite pointing at the three existing channels. Guide `how-to-install-the-operating-surface.md` carries the journey. Command names match the shipped surface. |
| T003 — glassfrog-setup skill + facts guard | ✓ Met | SKILL.md frontmatter `name: glassfrog-setup` + triggering description with an explicit orientation-territory exclusion; presence check (`glassfrog --version`) + auth check (`glassfrog me`), two failure classes kept distinct, re-check after fix, installs/stores nothing, boundary note. Guard extension anchors the three channel names + the `me` leaf to their in-repo sources; frontmatter name/description asserted non-empty. |

---

## Interface Contract Conformance

**Status**: Pass (marketplace schema, SKILL.md frontmatter + sections, guard contract all conformant)

| Surface | Status | Evidence |
|---|---|---|
| `marketplace.json` schema (`$schema`, `name`, `owner`, `plugins`; entry `name`/`source`/`description`, no `version`) | ✓ Conformant | Manifest content matches every pinned field; version deliberately absent |
| `SKILL.md` frontmatter (`name` + `description` only; fires on provisioning, excludes orientation territory) | ✓ Conformant | `SetupSkillName()`-matched name; description enumerates the five pinned provisioning triggers and the orientation-territory exclusion |
| Required SKILL.md sections (presence, auth, missing-CLI fix, failing-credential fix, ready, boundary) | ✓ Conformant | All six sections present with the pinned content; channel invocations quoted, npm coordinate sourced from `npm/package.json` |
| Guard contract (existence lookup, source resolution, identity equality, no-version, frontmatter, enumerable facts, stated partiality) | ✓ Conformant | `CheckMarketplaceEntryConsistency` + `CheckSetupSkillDrift`; both guards verified red-on-injected-drift during implementation |

**Observation (benign deviation, not a finding)**: interface-spec lists three path constants for `operatingsurfacepackaging.go` including `plugin/.claude-plugin/plugin.json`. The file declares `MarketplaceManifestPath` and `SetupSkillPath` and *reuses* the sibling `OrientationManifestPath` for the plugin-manifest path rather than re-declaring it. This is consistent with the family's single-source discipline (a second constant for the same path would be a duplicate source of truth); the guard's substantive behavior — reading the plugin manifest via a named constant — is present.

---

## Non-Behavior Absence

**Status**: Pass (7 of 7 excluded behaviors absent)

| Non-behavior | Status | Evidence |
|---|---|---|
| No duplicated orientation content / operator paths / write-safety hook | ✓ Absent | No operator-path skill name or orientation enumeration in either packaging artifact (`whenInspectedForOperatingSurface` asserts this); setup defers reference to orientation |
| No new API capability, command, or flag | ✓ Absent | No CLI Go code added; setup instructs only existing commands (`--version`, `me`, `auth login`), each resolved against the live registry |
| Setup skill does not reimplement/bundle/fork CLI install | ✓ Absent | "never installs, bundles, or places the binary itself"; routes to the three channels |
| Setup skill introduces no own credential mechanism | ✓ Absent | "no credential mechanism of its own", "stores nothing"; routes to `glassfrog auth login` |
| Marketplace not locked to a single plugin | ✓ Absent | List-shaped; `thenNoRestructuring`/`thenAdmitsAdditionalEntries` confirm append-only growth |
| Setup skill does not enforce/gate/block writes | ✓ Absent | No gating language in SKILL.md (grep clean for gate/PreToolUse/block); journey is check→fix→verify only |
| No Holacracy coaching | ✓ Absent | Provisioning content only; no governance-practice guidance |

---

## Validation Scenario Results

**Status**: Satisfied (4 of 4 scenarios traced to implementation)

| Scenario | Status | Trace |
|---|---|---|
| Marketplace entry matches the plugin it ships | ✓ Satisfied | `MarketplaceSourcePluginManifest("./plugin")` resolves; `CheckMarketplaceEntryConsistency` equates name + description; BDD `thenSourcePointsAtRealPlugin`/`thenIdentityMatches` |
| Packaging adds no operating surface of its own | ✓ Satisfied | `whenInspectedForOperatingSurface`: manifest keys limited to the distribution set, zero unresolved command leaves, no restated orientation enumerations, no operator-path skill names; `thenOperatingFactsLiveInPlugin` confirms orientation + setup live in the distributed plugin |
| Setup leaves the CLI self-contained | ✓ Satisfied | `thenOnlyPointsAtExistingFixes` (three channels + X-Auth-Token) + `thenInstallsNothingStoresNothing` |
| Marketplace shape admits additional entries | ✓ Satisfied | `thenAdmitsAdditionalEntries` — `plugins` is a JSON array; an appended sibling re-parses through the same parser |

**Independence caveat**: per the 062–069 family convention, the `@validation` scenarios run through the same build-side suite (the `~@wip` filter executes everything not tagged `@wip`), so the Builder authored their step definitions and un-@wip'd them. Their evidence is real (the traces and the passing suite), but the *held-out* independence the validation tag implies is weaker here than the ideal — this is a structural property of the family's single-suite approach, not a 070-specific gap. It is recorded so the "independent verification" claim is not overread.

---

## Verdict: Ready

All 3 tasks are checked. All 5 conformance dimensions pass with zero findings. All 4 validation scenarios trace to identifiable code paths and pass in the executable suite. The implementation conforms to its specification: the repo-root marketplace manifest distributes the glassfrog plugin with a consistency guard that keeps the entry truthful to `plugin.json`; the `glassfrog-setup` skill instructs the presence/auth journey and routes each failure to the CLI's existing channels without adding capability; and the two `internal/build` guards anchor the enumerable facts to their in-repo sources. The one interface deviation (reusing `OrientationManifestPath` rather than re-declaring the plugin-manifest path) is a benign, family-consistent choice, not a conformance gap.

---

## Next Steps

Implementation conforms to the specification. Suggest PR review and merge.

One out-of-band item, unchanged from the implement handoff: the plan's manual smoke test (add the repo as a marketplace in a real Claude Code session and install the plugin once) is the only check no inspection or guard can substitute for, since the host's marketplace-schema acceptance is deliberately outside the guard's coverage (plan Risk 1). Worth doing before or alongside merge.
