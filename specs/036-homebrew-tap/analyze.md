# Analyze: Homebrew Tap

**Feature**: 036-homebrew-tap
**Artifacts analyzed**: spec.md, plan.md, interface-spec.md, tasks.md, features/runtime-dependent-distribution/homebrew-tap.feature
**Checklist context**: loaded (9 checks, 9 pass, 0 fail)
**Checks**: 16 (15 pass, 1 fail)
**Generated**: 2026-06-12

---

## Summary

| Category | Checks | Pass | Fail | Severity |
|---|---|---|---|---|
| Consistency | 6 | 6 | 0 | P0 |
| Completeness | 6 | 6 | 0 | P1 |
| Coherence | 4 | 3 | 1 | P2 |
| **Total** | **16** | **15** | **1** | |

No P0 contradictions. Completeness is clean after T004's workflow guard closed the earlier coverage gap; one P2 coherence drift remains — the repository `LICENSE`-file traceability.

---

## Consistency: 6/6 passed

Passed: spec↔plan integration-boundary alignment (C1 — 022 upstream, GitHub Releases, the separate tap repo, GoReleaser config, Homebrew, siblings all reflected in plan); spec behavioral accord ↔ plan architecture (C2 — macOS+Linux install / currency / integrity ↔ formula + separate repo + stable-gated `tap` job); spec non-behaviors not architected by plan (C3 — no source build, no pre-release tracking, no commit to this repo's `main`, no Windows: plan honors each); plan ADRs ↔ interface contract (C4 — ADR-1 formula, ADR-2 separate tap repo, ADR-3 brew-publisher-only `tap` job, ADR-4 cross-repo token all materialized in interface's `brews` block, `tap` job, and the `release.disable`/`archives.mtime` refinements); plan System Architecture ↔ task scope (C5 — T001/T003/T004 build only what the plan describes; the `release.disable` + reproducible-`mtime` refinements trace plan Cross-cutting → interface → T003 with no contradiction); interface Surface ↔ feature steps (C6 — every Given/When/Then references an interface-defined surface: `brew install/upgrade`, `luscii/cli-glassfrog`, `glassfrog version`, `glassfrog_<version>_<os>_<arch>.tar.gz`, `checksums.txt`, the `brews` block, the tap repo's `Formula/glassfrog.rb`).

## Completeness: 6/6 passed

Passed: every spec driving + validation scenario has a Gherkin equivalent (K1 — all 10 map into homebrew-tap.feature); the interface is covered by the single Specification accord with each integration boundary addressed (K2 — Specification-touchpoint pattern, matching sibling 022); every plan phase has task decomposition (K3 — Phase 1→T001(+T002), Phase 2→T003, Phase 3→T004); every plan component has implementing tasks (K4 — `brews` block→T003, `tap` job→T004, repo+token→T001, refinements→T003); every spec user scenario has interface coverage (K6 — acquire macOS/Linux, upgrade-currency, trust-integrity); and every interface surface has downstream coverage (K5). On K5, the interface's Error Communication paths that `.feature` scenarios cannot exercise without a live tap — a missing/expired `HOMEBREW_TAP_TOKEN` and the `if: !prerelease` gate — are now covered structurally: **T004 extends the `internal/build` workflow guard** to assert the `tap` job's token-env wiring and gating. The `brew audit`/license path is covered by T002 plus the optional `brew audit` CI step the interface testing strategy names.

## Coherence: 3/4 passed

Passed: terminology consistent across the set (H1 — formula / tap / tap repo / archive / checksums / `brews` block; no stale "cask" survives the reframe; the tag-with-`v` vs asset-name-without-`v` distinction is reconciled in interface-spec.md as one rule); detail symmetry proportionate between spec↔plan and plan↔tasks (H2); task decomposition covers all plan phases structurally including the Phase 1 ∥ Phase 2 parallelism (H4).

### Failure

**P2 (coherence)** | **plan.md ↔ interface-spec.md ↔ tasks.md** (H3 — scope aligned across the set, no silently added/dropped capability)
plan.md **does** address licensing at the formula level — ADR-1's consequences, the testing strategy, and Risk #4 each name `brew audit`/`license` requirements. The narrow gap is that plan.md does not explicitly call out the **repository `LICENSE`-file decision** (whether to add a `LICENSE` so GoReleaser populates `brews.license`), which interface-spec.md raises as `[NEEDS INPUT]` and tasks.md T002 owns. Not silently added — fully traceable interface→tasks — but a one-line item in plan.md Phase 1 ("resolve the formula's license / add a `LICENSE`") would complete the plan→interface→tasks chain. Low impact: traceability hygiene; the work is already captured in T002.

---

## Checklist Correlation

One overlap remains after T004's workflow guard closed the VII/K5 pair:

- **Resolved**: the earlier Checklist P0 (VII Working Software, T004 test-free) ↔ Analyze P1 (K5, token/audit coverage) were a single root cause — closed by extending T004 to ship the `internal/build` workflow guard (asserting `needs: [publish]`, the `if: !prerelease` gate, the `HOMEBREW_TAP_TOKEN` env, brew-publisher-only). Both now pass.
- **Open (P2)**: Checklist's open `[NEEDS INPUT]` license note (T002) ↔ Analyze H3 — the repository `LICENSE`-file decision is traceable interface→tasks but not called out in plan.md. Resolving the license decision and adding a plan line closes both.

No analyze finding contradicts a checklist pass; the artifacts agree with one another. The only remaining gap is the shared `LICENSE`-file thread, not cross-artifact disagreement.

---

## Governance Notes

- **All 16 base checks ran** — every artifact in the relationship matrix (spec, plan, one interface file, one feature file, tasks) is present, so no checks were skipped.
- **Checklist context**: loaded and parsed (9 checks, 9 pass, 0 fail). Correlation performed above.
