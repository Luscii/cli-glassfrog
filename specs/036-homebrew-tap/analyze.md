# Analyze: Homebrew Tap

**Feature**: 036-homebrew-tap
**Artifacts analyzed**: spec.md, plan.md, interface-spec.md, tasks.md, features/runtime-dependent-distribution/homebrew-tap.feature
**Checklist context**: loaded (9 checks, 8 pass, 1 fail)
**Checks**: 16 (14 pass, 2 fail)
**Generated**: 2026-06-12

---

## Summary

| Category | Checks | Pass | Fail | Severity |
|---|---|---|---|---|
| Consistency | 6 | 6 | 0 | P0 |
| Completeness | 6 | 5 | 1 | P1 |
| Coherence | 4 | 3 | 1 | P2 |
| **Total** | **16** | **14** | **2** | |

No P0 contradictions. One P1 completeness gap and one P2 coherence drift, both tied to the same two open threads as checklist (the missing T004 test and the unresolved license).

---

## Consistency: 6/6 passed

Passed: spec↔plan integration-boundary alignment (C1 — 022 upstream, GitHub Releases, the separate tap repo, GoReleaser config, Homebrew, siblings all reflected in plan); spec behavioral accord ↔ plan architecture (C2 — macOS+Linux install / currency / integrity ↔ formula + separate repo + stable-gated `tap` job); spec non-behaviors not architected by plan (C3 — no source build, no pre-release tracking, no commit to this repo's `main`, no Windows: plan honors each); plan ADRs ↔ interface contract (C4 — ADR-1 formula, ADR-2 separate tap repo, ADR-3 brew-publisher-only `tap` job, ADR-4 cross-repo token all materialized in interface's `brews` block, `tap` job, and the `release.disable`/`archives.mtime` refinements); plan System Architecture ↔ task scope (C5 — T001/T003/T004 build only what the plan describes; the `release.disable` + reproducible-`mtime` refinements trace plan Cross-cutting → interface → T003 with no contradiction); interface Surface ↔ feature steps (C6 — every Given/When/Then references an interface-defined surface: `brew install/upgrade`, `luscii/cli-glassfrog`, `glassfrog version`, `glassfrog_<version>_<os>_<arch>.tar.gz`, `checksums.txt`, the `brews` block, the tap repo's `Formula/glassfrog.rb`).

## Completeness: 5/6 passed

Passed: every spec driving + validation scenario has a Gherkin equivalent (K1 — all 10 map into homebrew-tap.feature); the interface is covered by the single Specification accord with each integration boundary addressed (K2 — Specification-touchpoint pattern, matching sibling 022); every plan phase has task decomposition (K3 — Phase 1→T001(+T002), Phase 2→T003, Phase 3→T004); every plan component has implementing tasks (K4 — `brews` block→T003, `tap` job→T004, repo+token→T001, refinements→T003); every spec user scenario has interface coverage (K6 — acquire macOS/Linux, upgrade-currency, trust-integrity).

### Failure

**P1 (completeness)** | **interface-spec.md § Error Communication ↔ homebrew-tap.feature + tasks.md T004** (K5 — every interface surface has downstream coverage)
The interface's Error Communication enumerates a **missing/expired `HOMEBREW_TAP_TOKEN`** failure (the `tap` job fails loudly without touching the release) and a **`brew audit`/style** failure. Neither has downstream coverage: no Gherkin scenario exercises them (the token-failure scenario was consciously dropped at the scenarios stage to stay under the file threshold), and — per checklist's P0 — T004 specifies no workflow-structural guard that would assert the token-env wiring and gating. These are CI/audit failure modes better covered by a structural workflow guard than by `.feature` scenarios, so the fix rides on the same T004 test the checklist flags. Recommend the T004 guard assert the token env + `if: !prerelease`; the audit path is addressed by T002 (license) plus the optional `brew audit` CI step the interface testing strategy names.

## Coherence: 3/4 passed

Passed: terminology consistent across the set (H1 — formula / tap / tap repo / archive / checksums / `brews` block; no stale "cask" survives the reframe; the tag-with-`v` vs asset-name-without-`v` distinction is reconciled in interface-spec.md as one rule); detail symmetry proportionate between spec↔plan and plan↔tasks (H2); task decomposition covers all plan phases structurally including the Phase 1 ∥ Phase 2 parallelism (H4).

### Failure

**P2 (coherence)** | **plan.md ↔ interface-spec.md ↔ tasks.md** (H3 — scope aligned across the set, no silently added/dropped capability)
The **LICENSE decision** appears in interface-spec.md (the `[NEEDS INPUT]` on `brews.license`) and tasks.md (T002), but neither spec.md nor plan.md mentions licensing. It is not *silently* added — it is a traceable, explicit interface gap that tasks picks up — but the chain is incomplete upstream. Recommend a one-line item in plan.md Phase 1 ("resolve the formula's license / add a LICENSE") so the full plan→interface→tasks trace is intact. Low impact: the task already captures the work; this is traceability hygiene.

---

## Checklist Correlation

Two overlaps, both pointing at the same two open threads — the horizontal and vertical views agree:

- **Checklist P0 (VII Working Software, tasks.md T004)** ↔ **Analyze P1 (K5)**: the same root cause. Checklist sees T004 shipping a workflow change test-free; analyze sees the interface's token/audit error paths having no downstream coverage. Both resolve with one fix — add a workflow-structural guard to T004 (asserting `needs: [publish]`, `if: !prerelease`, the `HOMEBREW_TAP_TOKEN` env, brew-publisher-only). Fixing T004's test closes both findings.
- **Checklist governance note (open `[NEEDS INPUT]` license, T002)** ↔ **Analyze P2 (H3)**: checklist flags the license as an open input; analyze flags that plan.md doesn't carry it. Resolving the license decision and adding a plan line closes both.

No analyze finding contradicts a checklist pass; the artifacts agree with one another. The only gaps are the two shared open threads, not cross-artifact disagreement.

---

## Governance Notes

- **All 16 base checks ran** — every artifact in the relationship matrix (spec, plan, one interface file, one feature file, tasks) is present, so no checks were skipped.
- **Checklist context**: loaded and parsed (9 checks, 8 pass, 1 fail). Correlation performed above.
