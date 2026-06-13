# Validate: Homebrew Tap

**Feature**: 036-homebrew-tap
**Round**: 1 of 3
**Date**: 2026-06-13
**Verdict**: Not Ready (partial)
**Artifacts loaded**: spec.md, plan.md, tasks.md, interface-spec.md, homebrew-tap.feature, PROJECT.md
**Implementation files**: `.goreleaser.yaml` (brews block + 022 refinements); `internal/build/config.go` (config-guard extension); `internal/build/config_guard_test.go`, `internal/build/formula_render_test.go`, `internal/build/homebrew_tap_bdd_test.go` (tests)

---

## Conformance Summary

| Dimension | Status | Findings |
|---|---|---|
| Driving scenario coverage | ◐ Partial (T003 scope only) | 0 |
| Acceptance criteria | ✓ Pass (T003 — the only checked task) | 0 |
| Interface contract conformance | ✓ Pass (T003 surfaces; tap-job surface deferred to T004) | 2 (low, intent-conformant) |
| Non-behavior absence | ✓ Pass | 0 |
| @wip lifecycle completion | ✓ Pass | 0 |
| **Validation scenarios** | ◐ Not yet traceable (depend on T004 + live tap) | 0 |

**Total**: 5 dimensions checked, all pass for the checked task; 2 low-severity, intent-conformant interface adaptations recorded. **1 of 4 tasks complete** — this is a progress checkpoint, not a failure.

---

## Task Completion

| Task | State | Notes |
|---|---|---|
| T001 — provision tap repo + `HOMEBREW_TAP_TOKEN` | ☐ Not done | Ops/credential work; developer-owned (cannot be done by the Builder) |
| T002 — LICENSE decision + `brews.license` | ☐ Not done | Developer-owned decision; `license` left unset (GoReleaser auto-detects from a `LICENSE`) |
| T003 — `brews` block + config-guard + offline render test | ☑ Done | Validated below |
| T004 — `tap` job in `release.yml` + workflow guard | ☐ Not done | Blocked on T001 (live repo+secret); also entangled with the `brews`-deprecation decision |

---

## Driving Scenario Coverage

**Status**: Partial — only scenarios referenced by the checked task (T003) are evaluated. The spec's seven brew-install/upgrade driving scenarios are all referenced by **T004** (unchecked) and are reported as *not yet implemented* — they describe live `brew install`/`brew upgrade` behaviour that the tap job (T004) plus a live tap produce.

| Scenario | Referenced by | Status | Implementation |
|---|---|---|---|
| Config-guard fails when the brews block is blanked or retargeted | T003 | ✓ Covered | `internal/build/config.go:checkBrews`; `config_guard_test.go` (blanked/retargeted/owner/formula-name drift cases); `homebrew_tap_bdd_test.go` |
| The published formula's checksums match the release's checksums file | T003 | ✓ Covered | `internal/build/formula_render_test.go:TestFormulaRender_OfflineShapeAndChecksums` (each rendered sha256 == snapshot `checksums.txt` entry) |
| Fresh install on macOS / Linux | T004 | ○ Not yet implemented | Tap job not built |
| Upgrade moves to the latest stable / no-op when current | T004 | ○ Not yet implemented | Tap job not built |
| A missing release asset fails the install clearly | T004 | ○ Not yet implemented | Tap job not built |
| Checksum mismatch refuses the install | T004 | ○ Not yet implemented | Tap job not built |
| A pre-release does not move the tap | T004 | ○ Not yet implemented | Tap job not built (stable-only `if`-gate lives in T004) |

---

## Acceptance Criteria

**Status**: Pass — T003 is the only checked task; all five of its acceptance criteria have implementation evidence.

| Criterion (T003) | Status | Evidence |
|---|---|---|
| Snapshot render produces `Formula/glassfrog.rb` with four platform url+sha256 blocks, install + test stanzas, populated version | ✓ Met | `TestFormulaRender_OfflineShapeAndChecksums` (asserts class, both OS blocks, 4 url+sha, `bin.install`, `test`, non-empty version) — passes |
| Each formula sha256 equals the matching `checksums.txt` line | ✓ Met | Same test cross-checks all four; verified green |
| `builds`/`ldflags` byte-unchanged; `release.disable: true`; archive mtime pinned | ✓ Met | `builds`/`ldflags` untouched in `.goreleaser.yaml`; `release.disable: true`; archive entry mtime pinned via `builds_info.mtime` (see F-1) |
| Config-guard fails loudly on missing/retargeted brews, non-disabled release, or unpinned mtime | ✓ Met | `checkBrews`/`checkRelease`/`checkArchives` + drift cases in `config_guard_test.go` — all pass |
| Implementation and tests ship in the same PR (CONSTITUTION VII) | ✓ Met | Same commit (`7e5df7f`) |

---

## Interface Contract Conformance

**Status**: Pass for the T003 surfaces (the `.goreleaser.yaml` `brews` section, the two 022 refinements, the config-guard extension, and the generated-formula structural contract). The `release.yml` `tap` job surface is **not yet implemented** (T004) — reported as deferred, not as a finding. Two low-severity, intent-conformant adaptations recorded (F-1, F-2).

| Interface surface | Status | Notes |
|---|---|---|
| `brews` fields (name, ids, repository.*, directory, homepage, description, install, test, skip_upload) | ✓ Conformant | All present; `license` intentionally omitted (auto-detect; T002's `[NEEDS INPUT]`) |
| Refinement: `release.disable: true` | ✓ Conformant | In effect; config-guard pins it |
| Refinement: pinned archive mtime | ◑ Conformant (adapted) | Realized as `builds_info.mtime` — see **F-1** |
| Config-guard extension (brews target + refinements) | ✓ Conformant | `checkBrews` + extended `checkArchives`/`checkRelease` |
| Generated `Formula/glassfrog.rb` structural contract | ✓ Conformant | Real GoReleaser output uses `on_macos`/`on_linux` + `Hardware::CPU` branches (interface noted "exact whitespace/order is GoReleaser's"); URL hard-contract satisfied — see **F-2** |
| `release.yml` `tap` job structure | ○ Deferred (T004) | Not yet implemented |

---

## Non-Behavior Absence

**Status**: Pass — no excluded behaviour is present in the T003 implementation.

| Non-behavior | Status | Evidence |
|---|---|---|
| Must not build / require a Go toolchain or source checkout | ✓ Absent | Formula uses `bin.install "glassfrog"` (installs the pre-built archive); no compile stanza |
| Must not track pre-releases | ✓ Absent | Nothing publishes yet; the authoritative stable-only `if`-gate is T004. `skip_upload: false` is set with the job-level gate as the contract — no pre-release tracking introduced |
| Must not commit the formula to this repo / push to its `main` | ✓ Absent | `brews.repository` targets the separate `Luscii/homebrew-cli-glassfrog`; nothing writes this repo's `main` |
| Must not author release notes, sign/notarize, or bump versions | ✓ Absent | `release.disable: true` stops GoReleaser from touching the GitHub release; no signing/version code added |
| Must not support platforms outside the four release targets | ✓ Absent | `brews.ids` draws from the four-target `glassfrog` build; rendered formula has exactly darwin/linux × amd64/arm64, no windows |

---

## Validation Scenario Results

**Status**: Not yet traceable — all three held-out @validation scenarios depend on the **tap job (T004)** and a live tap/release, which are not yet implemented. They remain `@validation @wip` (correctly held). None is a finding at this stage; they are revisited when T004 lands.

| Scenario | Status | Trace |
|---|---|---|
| The installed binary matches the release the formula points at | ○ Not yet traceable | Requires a live `brew install` from the tap (T004 + manual post-first-release validation) |
| A mismatched checksum never lets a binary reach PATH | ○ Not yet traceable | Homebrew install-time integrity behaviour; T004 + manual |
| A pre-release leaves the tap repository untouched | ○ Not yet traceable | Tap-job `if`-gate behaviour (T004) |

---

## @wip Lifecycle Completion

**Status**: Pass. The two scenarios referenced by the checked task T003 ("Config-guard fails when the brews block is blanked or retargeted", "The published formula's checksums match the release's checksums file") have had their `@wip` removed and are exercised by the suites. All other scenarios — the install/upgrade/pre-release driving scenarios (T004) and the three `@validation` scenarios — correctly retain `@wip`.

---

## Findings

### F-1: Archive mtime pinned via `builds_info.mtime`, not the interface's `archives.mtime`

- **Dimension**: Interface contract conformance
- **Source**: interface-spec.md § Refinements to 022's sections > "Reproducible archives — Pin `archives.mtime` (to the commit date)"
- **Implementation**: `.goreleaser.yaml` `archives[0].builds_info.mtime: "{{ .CommitDate }}"`; guard at `internal/build/config.go:checkArchives`
- **Gap**: The interface names a top-level `archives.mtime` field. The installed GoReleaser (v2.16.0, the `~> v2` pinned in `release.yml`) has no such field — its parser rejects it — so the archive-entry modification time is pinned under `builds_info.mtime` instead. **Intent met**: the offline render produces archives whose sha256 match `checksums.txt` (verified). Low severity; tooling-driven adaptation, documented in `.score/memory/LEARNINGS.md`.

### F-2: `brews.url_template` added (not in the interface's brews field list)

- **Dimension**: Interface contract conformance
- **Source**: interface-spec.md § `.goreleaser.yaml` — `brews` section (field table) and § Generated `Formula/glassfrog.rb` ("Hard contract" on asset URLs)
- **Implementation**: `.goreleaser.yaml` `brews[0].url_template`
- **Gap**: The interface's `brews` field table does not list `url_template`, but `release.disable: true` (the interface's own refinement) removes GoReleaser's default URL template, so the URLs must be pinned explicitly. The added `url_template` reproduces the interface's exact URL **hard contract** (`.../releases/download/{{ .Tag }}/{{ .ArtifactName }}`). Conformant in substance; the addition is the mechanism the contract requires. Low severity; documented in `.score/memory/LEARNINGS.md`.

---

## Observation (non-conformance — forward risk for T004)

Not a spec-conformance gap: the implementation **is** a Homebrew formula, exactly as ADR-1/spec require. But GoReleaser has **deprecated the `brews` (formula) publisher** in favour of `homebrew_casks` (soft since v2.10). `goreleaser release` still renders the formula (with a deprecation warning) and CI does **not** run `goreleaser check`, so nothing fails today — but `goreleaser check` exits non-zero on it, and a future `~> v2` bump could remove `brews` and break the T004 tap job. This conflicts with ADR-1's reason for choosing a formula over a cask (casks are macOS-only; the spec needs Linux). **Decide before T004 / the first live publish**: pin GoReleaser to a `brews`-supporting v2, or revisit ADR-1 (and flag for `/score:deprecate`). Recorded in `.score/memory/LEARNINGS.md`.

---

## Verdict: Not Ready (partial)

1 of 4 tasks complete (T003). 2 of the feature's T003-scoped scenarios covered; 0 of the 7 spec driving scenarios covered (all belong to T004); 0 of 3 @validation scenarios traceable yet (all depend on T004 + a live tap). For the **one** checked task, every conformance dimension passes: acceptance criteria met, interface surfaces conformant (two low-severity, intent-conformant adaptations), all non-behaviors absent, @wip lifecycle correct. This is a clean progress checkpoint — the implemented slice conforms; the remaining behaviour is genuinely not built yet, not built wrong.

T001 (provision tap repo + token) and T002 (LICENSE decision) are developer-owned and cannot be completed by the Builder. T004 (tap job + workflow guard) is blocked on T001 and entangled with the `brews`-deprecation decision above.

---

## Next Steps

- **T001 / T002** — developer actions: create `Luscii/homebrew-cli-glassfrog` + mint the scoped `HOMEBREW_TAP_TOKEN`; decide the LICENSE (or accept the auto-detect/omit).
- **Resolve the `brews`-deprecation question** (pin GoReleaser, or revisit ADR-1) before building T004.
- **T004** — once T001 lands and the deprecation is settled, implement the `tap` job + workflow guard via `/score:implement`, then **re-validate** (`/score:validate`). Re-validation will then cover the seven driving scenarios and the three held-out @validation scenarios (the latter partly via the documented manual post-first-release `brew install`).
