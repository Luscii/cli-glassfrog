# Checklist: Proposal Reads

**Feature**: 056-proposal-reads
**Checked against**: CONSTITUTION.md (12 principles)
**Artifacts checked**: spec.md, plan.md, interface-cli.md, features/proposal-write-flow/proposal-reads.feature, tasks.md
**Checks**: 19 (19 pass, 0 fail)
**Generated**: 2026-06-15

---

## Summary

All 19 checks pass. Constitution: 19/19. Done-criteria: not run (no `done-*` accords). Cross-references: not run (no `done-*` accords).

Like sibling 043 (Tension Reads), this is a **read pair**, so several principles bind in their read-side form: **VI Size-Aware** is load-bearing (the global `proposal list` walk must never silently truncate) and **IX Writes Require Explicit Intent** binds as "a read MUST NEVER mutate" — reinforced here by `available_transitions` being *surfaced but never invoked*. **VIII No Fabricated Data** is unusually load-bearing for this feature: the aggregate-only `response_summary` (no per-person attribution) and the free-form `changes[]` (passed through, never interpreted) are both VIII commitments enforced at the type level. Principles X and XI pass on applicability grounds (calibration notes below). The one structural nuance is that 056 *creates* the `proposal` group and model shared with the concurrent Proposal Creation (055) — Calibration note 3 records why this is additive, not a Composition violation.

---

## Constitution Checks: 19/19 passed

### Passed

**P0** | I. Spec Fidelity — "Every command MUST map to a spec operation; no invented endpoints/parameters/behaviors"
→ **interface-cli.md / spec.md**: `proposal list` maps to `listProposals` (`GET /proposals`) and `proposal get <prp-id>` to `getProposal` (`GET /proposals/{id}`); the five list filters are sent as the spec's documented query parameters (`status`, `role_id`, `proposer_id`, `proposed_after`, `accepted_after`), `--status` validated against the spec enum (`draft`/`proposed_outside_meeting`/`escalated`/`accepted`/`draft_with_conflicts` — incl. `draft_with_conflicts`, verified against `spec/glassfrog-api-v5.yaml`); the `prp_` id is passed through to the path. No invented endpoint, parameter, or behavior. Pass (×2: operation mapping + no-invented-params).

**P0** | II. Action Transparency (NON-NEGOTIABLE) — "report the spec operation and target resource, machine-parseable; every error explains cause + next step"
→ **interface-cli.md § Output / Error Communication**: success produces the proposal data machine-parseable under `json`/`yaml` (the `prp_` id present); the Error Communication table gives every failure a named cause **and** a concrete next step, and never prints the token. The list also signals incompleteness explicitly on stderr (more-exist / mid-walk). Pass (×2: success traceability + error cause/next-step).

**P0** | III. Fail Safe, Not Silent — "errors obvious and recoverable, never hidden"
→ **spec.md § Behavioral Accord / interface-cli.md § Interactions**: `--status` is validated fail-fast **before any request** (transport tripwire asserts no call on rejection); a positional on `list` and a list flag on `get` are rejected pre-assembly; a mid-walk failure surfaces the partial set flagged incomplete on stderr and exits non-zero — never reported as a complete success; no swallowed errors. (The "validate a write / no partial state" clause is N/A — these are reads with no mutation.) Pass (×2: fail-fast input rejection + incomplete-never-as-success).

**P0** | IV. Test-Driven Development — "test-first; user-facing behavior has an executable acceptance scenario"
→ **tasks.md (T001–T006) / proposal-reads.feature**: every task mandates RED-first unit tests; T006 makes the driving scenarios executable acceptance; the feature file exists with `@wip` scenarios (4 `@validation` held for validate). Pass (×2: test-first mandate + acceptance scenarios present).

**P0** | V. Composition over Monolith — "modular per-resource modules; adding a command MUST NOT require changing unrelated ones"
→ **plan.md § System Architecture / tasks.md**: a new `internal/glassfrog/proposal.go`, a new proposal command file, two new render keys, and a proposal-status validator, all over the shared client. No edit to any *existing* command, model, render key, or the shared `status.go`. The only "shared" creation is the new `proposal` group/model, which is co-owned with the concurrent 055 under grow-not-duplicate — additive by construction. Pass — see Calibration note 3.

**P0** | VI. Size-Aware by Design — "handle large result sets within pagination limits; MUST NEVER silently truncate"
→ **spec.md § Completeness of the list / plan.md § Cross-cutting / interface-cli.md § Interactions**: `proposal list` walks every page via `paging.All` by default; `--first-page` opts out to one page and writes a "more proposals exist" stderr note (exit 0); a mid-walk failure renders the partial set with an explicit "incomplete — <cause>" stderr note and exits non-zero. Never silently truncates (the 025 ADR-3 completeness pattern). Pass (×2: full-walk default + opt-out/mid-walk boundary signalled). **Load-bearing for this read spec** (note: the list is *global*, so it is the whole-org proposal set being walked, not a role-scoped subset).

**P0** | VII. Working Software — "implementation together with its tests; validates and builds"
→ **tasks.md (T001–T006)**: each task bundles implementation with its tests and lists `go build`/`go vet` clean in acceptance — no code-only or test-only increment outside the RED→GREEN pair. Pass.

**P0** | VIII. No Fabricated Data — "present only data the API returned; no invented/placeholder values"
→ **spec.md § Non-Behaviors / interface-cli.md § Output / tasks.md T001-T002**: nullable fields render through explicit-absence guards (`(none)`, `no proposals`); `response_summary` exposes **aggregate counts only** — the model carries no per-person field, so attribution cannot be fabricated (a spec non-behavior + validation scenario); `changes[]` is rendered **verbatim** with no per-type schema invented (free-form `map[string]any` remainder); structured `json`/`yaml` emits the raw server bytes (faithful even for fields the model omits). Pass (×2: explicit-absence/no-placeholder + no-fabricated-attribution/changes). **Load-bearing for this feature.**

**P0** | IX. Writes Require Explicit Intent — "no mutation except via an explicit write command; a read MUST NEVER mutate"
→ **spec.md § Non-Behaviors / interface-cli.md**: both commands are reads issuing only `GET`; the spec's Non-Behaviors explicitly forbid create/advance/withdraw/respond; **`available_transitions` is surfaced but never invoked** (explicit non-behavior + validation scenario + T005 risk note); list filters on `get` are rejected. No `POST`/`PATCH`/`DELETE` on any read path. Pass (×2: GET-only reads + transitions-surfaced-not-invoked) — the read-side mirror of the principle (calibration note 1).

**P0** | X. Respect API Limits — "back off on 429; use If-Match/ETag for updates"
→ **plan.md § Cross-cutting**: the reads reuse the landed `RetryExecutor` (017), which honors `429`/`Retry-After`; both are `GET`s (safe to auto-retry). The `If-Match`/`ETag` clause targets **updates** — N/A to reads (nothing is mutated; no `If-Match` is sent). Pass — see Calibration note 2.

**P0** | XI. Governance via Proposals — "governance-structure mutations go through a Proposal; no default path mutating governance structure directly"
→ **spec.md / PROJECT.md**: reading proposals mutates no governance structure (roles/accountabilities/domains/policies) — it reads the proposal resource itself — so the opt-in-flag requirement does not apply. Pass on applicability grounds — see Calibration note 1.

**P0** | XII. Standalone Executable — "self-contained; no pre-installed dependencies beyond network"
→ **plan.md / tasks.md**: 056 adds only Go code (one model file, two render keys + templates, a validator, a group + two command leaves) — no language runtime, service, or library that must be installed first. Pass.

---

## Calibration Notes

1. **IX & XI bind in read form.** Proposal Reads performs no mutation, so IX ("a read MUST NEVER mutate") and XI ("no default path mutating governance structure directly") are satisfied by the GET-only command surface and the explicit Non-Behaviors. The distinctive read-side reinforcement is that `available_transitions` (the names of write actions the caller *could* invoke) is rendered as data but never acted on — the spec's non-behavior, a validation scenario, and a T005 risk note all guard this. The governance-mutation force of XI applies to the *write* flow (Proposal Creation 055 and the advance/withdraw/respond specs), not to these reads.

2. **X applies on the back-off half only.** The `If-Match`/`ETag` half of X governs *updates*; these are reads, so it is N/A (no `If-Match` is sent, correctly). The `429` back-off half is satisfied by reuse of the landed `RetryExecutor`. No optimistic-concurrency obligation arises for a read.

3. **V holds despite shared creation.** 056 *creates* the `proposal` group, the `glassfrog.Proposal` model, and both render keys — infrastructure co-owned with the concurrently-specified Proposal Creation (055). This is not a Composition violation: it adds new files and types and changes no unrelated existing command. The 055 coordination contract (plan ADR-1/ADR-2, tasks dependency-graph ⚠️) is explicitly grow-not-duplicate — the follower attaches leaves and grows shared types rather than editing unrelated modules. Adding the later proposal write verbs will likewise attach to the group without touching the reads.

---

## Governance Infrastructure Notes

- **No `done-*` accords found** (`accords/governance/` is absent). Done-criteria checks and cross-reference checks were not run — only constitution checks. Consider creating `accords/governance/done-{specify,plan,interface,scenarios,tasks}.md` to enable done-criteria quality checks across the pipeline. (Same gap noted for every sibling spec to date; not specific to 056.)
- **guardian-agent.md not loaded** (empty/unavailable in the deployed skill) — checks were generated and evaluated from the SKILL.md process directly. No effect on the binary check results.

---

## Result

19/19 constitution checks pass. No P0/P1/P2 failures. The spec, plan, interface, scenarios, and tasks are constitution-clean and ready to proceed to cross-artifact analysis (`/score:analyze`).
