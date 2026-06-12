# Checklist: Tension Reads

**Feature**: 043-tension-reads
**Checked against**: CONSTITUTION.md (12 principles)
**Artifacts checked**: spec.md, plan.md, interface-cli.md, features/tension-capture/tension-reads.feature, tasks.md
**Checks**: 17 (17 pass, 0 fail)
**Generated**: 2026-06-12

---

## Summary

All 17 checks pass. Constitution: 17/17. Done-criteria: not run (no `done-*` accords). Cross-references: not run (no `done-*` accords).

Unlike sibling 042 (a write), this is a **read pair**, so two principles bind differently: **VI Size-Aware** is now load-bearing (the `tension list` walk must never silently truncate) and **IX Writes Require Explicit Intent** binds as "a read MUST NEVER mutate". Principles X and XI pass on applicability grounds (calibration notes below).

---

## Constitution Checks: 17/17 passed

### Passed

**P0** | I. Spec Fidelity — "Every command MUST map to a spec operation; no invented endpoints/parameters/behaviors"
→ **interface-cli.md / spec.md**: `tension list <role-id>` maps to `listRoleTensions` (`GET /roles/{role_id}/tensions`) and `tension get <ten-id>` to `getTension` (`GET /tensions/{id}`); the one filter `--status` is sent as the spec's `?status=` enum (`unprocessed`/`processed`/`archived`); ids are passed through to the path. No invented endpoint, parameter, or behavior. Pass (×2: operation mapping + no-invented-params).

**P0** | II. Action Transparency (NON-NEGOTIABLE) — "report the spec operation and target resource, machine-parseable; every error explains cause + next step"
→ **interface-cli.md § Output / Error Communication**: success produces the tension data machine-parseable under `json`/`yaml` (the `ten_` id present); the Error Communication table gives every failure a named cause **and** a concrete next step, and never prints the token. The list also signals incompleteness explicitly on stderr (more-exist / mid-walk). Pass (×2: success traceability + error cause/next-step).

**P0** | III. Fail Safe, Not Silent — "errors obvious and recoverable, never hidden"
→ **spec.md § Behavioral Accord / interface-cli.md § Interactions**: `--status` is validated fail-fast **before any request** (transport tripwire asserts no call on rejection); a mid-walk failure surfaces the partial set flagged incomplete on stderr and exits non-zero — never reported as a complete success; no swallowed errors. (The "validate a write / no partial state" clause is N/A — these are reads with no mutation.) Pass (×2: fail-fast input rejection + incomplete-never-as-success).

**P0** | IV. Test-Driven Development — "test-first; user-facing behavior has an executable acceptance scenario"
→ **tasks.md (T001–T005) / tension-reads.feature**: every task mandates RED-first unit tests; T005 makes the driving scenarios executable acceptance; the feature file exists with `@wip` scenarios (3 `@validation` held for validate). Pass (×2: test-first mandate + acceptance scenarios present).

**P0** | V. Composition over Monolith — "modular per-resource modules; adding a command MUST NOT require changing unrelated ones"
→ **plan.md § System Architecture / tasks.md**: a new `internal/cli/tension_reads.go` + one plural render key + a tension-status validator, over the shared client. The only shared touch is **extending 042's `newTensionCommand` to `MustRegister` the two read leaves** — same-resource (the `tension` group) and additive; it changes no unrelated command and does not alter `create`'s behavior. Pass — see Calibration note 3.

**P0** | VI. Size-Aware by Design — "handle large result sets within pagination limits; MUST NEVER silently truncate"
→ **spec.md § Completeness of the list / plan.md § Cross-cutting / interface-cli.md § Interactions**: `tension list` walks every page via `paging.All` by default; `--first-page` opts out to one page and writes a "more tensions exist" stderr note (exit 0); a mid-walk failure renders the partial set with an explicit "incomplete — <cause>" stderr note and exits non-zero. Never silently truncates (the 025 ADR-3 completeness pattern). Pass (×2: full-walk default + opt-out/mid-walk boundary signalled). **This is the load-bearing read-spec check** (N/A for 042's create).

**P0** | VII. Working Software — "implementation together with its tests; validates and builds"
→ **tasks.md (T001–T005)**: each task bundles implementation with its tests and lists `go build`/`go vet` clean in acceptance — no code-only or test-only increment outside the RED→GREEN pair. Pass.

**P0** | VIII. No Fabricated Data — "present only data the API returned; no invented/placeholder values"
→ **interface-cli.md § Output / tasks.md T001**: the new plural `tensions` render key uses explicit-absence guards (`(none)`, `no tensions`) for nullable/empty fields and renders only API-returned tensions; the `body` is rendered verbatim, never truncated or invented; the single read reuses 042's `tension` key unchanged. Pass.

**P0** | IX. Writes Require Explicit Intent — "no mutation except via an explicit write command; a read MUST NEVER mutate"
→ **spec.md § Non-Behaviors / interface-cli.md**: both commands are reads issuing only `GET`; the spec's Non-Behaviors explicitly forbid create/update/discard/subroles-rollup and any mutation; `--status` on `get` is rejected. No `POST`/`PATCH`/`DELETE` on any read path. Pass — this is the read-side mirror of the principle (calibration note 1).

**P0** | X. Respect API Limits — "back off on 429; use If-Match/ETag for updates"
→ **plan.md § Cross-cutting**: the reads reuse the landed `RetryExecutor` (017), which honors `429`/`Retry-After`; both are `GET`s (safe to auto-retry per `isSafeMethod`). The `If-Match`/`ETag` clause targets **updates** — N/A to reads (nothing is mutated). Pass — see Calibration note 2.

**P0** | XI. Governance via Proposals — "governance-structure mutations go through a Proposal; no default path mutating governance structure directly"
→ **spec.md / PROJECT.md**: reading tensions mutates no governance structure (roles/accountabilities/domains/policies), so the opt-in-flag requirement does not apply. Pass on applicability grounds — see Calibration note 1.

**P0** | XII. Standalone Executable — "self-contained; no pre-installed dependencies beyond network"
→ **plan.md / tasks.md**: 043 adds only Go code (one render key + templates, a validator, two command leaves) — no language runtime, service, or library that must be installed first. Pass.

---

## Governance Infrastructure Notes

*(Separate from the feature quality findings above.)*

- **No `done-*` accords found** — `accords/governance/` does not exist, so **done-criteria checks and cross-reference checks were not generated**. This is a **project-wide** condition (no spec in the repo has these accords), not specific to 043. Consider creating `accords/governance/done-{specify,plan,interface,scenarios,tasks}.md` to unlock done-criteria checks for every spec; until then, checklist evaluates constitution principles only.
- **`agents/guardian-agent.md` not deployed** — the checklist ran on its SKILL.md process alone (reduced character consistency, not a blocked skill).

## Calibration Notes

1. **Principles IX and XI read as the inverse of 042.** Where 042 (a write) satisfied IX by *being* an explicit write command, 043 satisfies IX by being read-only — two `GET`s that mutate nothing, with the spec's Non-Behaviors forbidding any mutation. Likewise XI's governance-mutation set is empty for a read. Both recorded as passes; if a future tension *edit* spec (044/045) reused this surface, IX/XI would re-engage on the write paths.
2. **Principle X applied to reads.** The binding part for `GET`s is 429 back-off — satisfied via the landed `RetryExecutor` (a `GET` is safely auto-retried). `If-Match`/`ETag` governs *updates*; reads carry no write to guard, so that clause is N/A.
3. **Principle V — the one shared touch is same-resource.** 043 extends 042's `newTensionCommand` to register the `list`/`get` leaves alongside `create`. This is intra-resource (the `tension` group) and additive — it forces no change to any unrelated command and does not alter `create`'s behavior — so it satisfies "adding a command MUST NOT require changing unrelated ones". (Coordination with sibling tension specs 044/045/046, which also attach to this group, is noted in plan/tasks.)
