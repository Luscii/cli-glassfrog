# Checklist: Install Script

**Feature**: 027-install-script
**Checked against**: CONSTITUTION.md (12 principles; done-* accords not present)
**Artifacts checked**: spec.md, plan.md, interface-spec.md, tasks.md, features/runtime-dependent-distribution/install-script.feature
**Checks**: 11 (9 pass, 2 fail)
**Generated**: 2026-06-09

---

## Summary

| Severity | Count | Pass | Fail |
|---|---|---|---|
| P0 (blocking) | 11 | 9 | 2 |
| P1 (should fix) | 0 | 0 | 0 |
| P2 (consider) | 0 | 0 | 0 |
| **Total** | **11** | **9** | **2** |

All 11 checks derive from constitution MUST / MUST NOT / NON-NEGOTIABLE principles, so every check is P0. Six principles (I, V, VI, IX, X, XI) produced zero applicable checks — they govern the CLI's Glassfrog-API command surface, which this shell installer has none of (see Governance Notes).

---

## Constitution Checks: 9/11 passed

### Failures

**P0** | CONSTITUTION.md II. Action Transparency (NON-NEGOTIABLE): "every error MUST explain what went wrong and the next step"
→ **interface-spec.md § Error Communication**: The contract specifies each failure's *cause* but commits to an actionable *next step* only for some — missing-tooling names "what satisfies it", and the not-on-PATH path prints the exact export line. Checksum mismatch ("message naming the integrity failure"), download failure ("stderr message"), unresolved-latest, and pinned-version-not-found are specified as cause-only. Action Transparency requires both halves for every error. Recommend the interface (and implementation) commit each error to a next step where one exists — e.g. mismatch/download-failure → "re-run to retry the download"; unsupported platform → "use a supported platform, or the Homebrew/npm channel".

**P0** | CONSTITUTION.md VII. Working Software: "Every commit and PR MUST include implementation together with its tests … No code-only or test-only increments"
→ **tasks.md § Phase 1 / Phase 2 (T001, T002)**: T001 ships `install.sh` as its own PR-sized unit; its tests live in T002 (Go exec-test) and T003 (shellcheck), separate PR-sized units that `depend on T001`. As decomposed, the T001 PR is a code-only increment. Recommend either folding the exec-test into T001 so the script and its tests land together, or having T001 deliver at least the unit tests for its pure functions (`detect_platform`, `asset_names`, checksum-line extraction) so no PR is test-free — leaving T002 for the integration harness.

### Passed (9/11)

- **II. Action Transparency** (success reporting): the installer reports what it did — interface-spec.md § Surface specifies success output of the install path and installed version; spec.md § Behavioral Accord ("reports where the binary was installed and which version was placed"). **Pass.**
- **III. Fail Safe, Not Silent** (3 checks): (a) validate-before-install — spec.md § Behavioral Accord and plan.md ADR-3 ("nothing touches the install dir until verification passes"); (b) no partial state — plan.md ADR-3 temp-dir→verify→`mv` with `EXIT`-trap cleanup, spec.md ("leaves no partially-written binary"); (c) no failure-as-success — interface-spec.md § Error Communication maps every failure to a non-zero exit (1/2/3). **Pass.**
- **IV. Test-Driven Development** (2 checks): (a) user-facing behavior has executable acceptance scenarios authored before the code — install-script.feature, 12 scenarios, all `@wip` (the pre-implementation RED layer); (b) the decomposition includes test work (T002 exec-test, T003 shellcheck) so implementation is not planned test-free at the suite level. **Pass.** (The PR-granularity concern is captured under VII above.)
- **VIII. No Fabricated Data**: the success output reports only the version actually resolved/installed — interface-spec.md § Surface and the validation scenario "The installed binary reports the resolved version" (installed `--version` equals the reported/resolved tag). **Pass.**
- **XII. Standalone Executable** (2 checks): (a) the installed artifact remains the self-contained binary — plan.md § System Architecture and § What This Plan Does Not Cover preserve that the installer adds no runtime dependency to the binary; (b) the installer's own host-tooling reliance (downloader/`tar`/sha256) is scoped to the shell installer and explicitly documented as not a XII concern — plan.md System Architecture and interface-spec.md § Consistency Notes. **Pass.**

---

## Done-Criteria Checks: not run

No `accords/governance/done-*.md` accords are present in the repository, so no done-criteria checks were generated. See Governance Notes.

---

## Cross-Reference Checks: not run

Cross-reference checks derive from done-* accords that require inter-artifact references; with no such accords present, none were generated. The artifacts do carry traceability (tasks.md references the feature file, interface, and plan; interface-spec.md cites spec 022's contract) — horizontal consistency of those links is `/score:analyze`'s domain.

---

## Governance Notes

- **No `accords/governance/` directory.** done-specify.md, done-plan.md, done-interface.md, done-scenarios.md, done-tasks.md are all absent. Consider creating `accords/governance/done-<skill>.md` for each to enable done-criteria and cross-reference quality checks. Until then, checklist coverage for this project is constitution-only.
- **Zero-applicable constitution principles** (governance surface this feature doesn't touch):
  - **I. Spec Fidelity** — no applicable checks: the installer invokes no Glassfrog API v5 operation; it fetches release artifacts from GitHub Releases, which the spec contract does not govern.
  - **V. Composition over Monolith** — no applicable checks: the feature adds no per-resource command module; it is a standalone script. (Its internal function decomposition is a plan concern, not the principle's command-module scope.)
  - **VI. Size-Aware by Design** — no applicable checks: no API result-set pagination or org-tree traversal is involved.
  - **IX. Writes Require Explicit Intent** — no applicable checks: the installer is not a read-shaped command; installing onto disk is its explicit, sole purpose, invoked deliberately.
  - **X. Respect API Limits** — no applicable checks against the Glassfrog API: the installer never calls it. (Related but non-constitutional: plan ADR-2 deliberately avoids GitHub's API rate limit via the `releases/latest` redirect.)
  - **XI. Governance via Proposals** — no applicable checks: the installer performs no governance-structure mutation.
