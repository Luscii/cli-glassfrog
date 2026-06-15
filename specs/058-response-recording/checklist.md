# Checklist: Response Recording

**Feature**: 058-response-recording
**Inputs**: spec.md, plan.md, interface-cli.md, features/proposal-write-flow/response-recording.feature, tasks.md; CONSTITUTION.md
**Checks**: 16 (16 pass, 0 fail) | 4 principles N/A for this feature
**Generated**: 2026-06-15T12:50:00

---

## Summary

| Severity | Pass | Fail | Total |
|---|---|---|---|
| P0 | 8 | 0 | 8 |
| P1 | 8 | 0 | 8 |
| P2 | 0 | 0 | 0 |
| **Total** | **16** | **0** | **16** |

**By category**: Constitution 16/16. Done-criteria 0/0 (no `accords/governance/done-*.md` present). Cross-reference 0/0 (no done-* accords to derive link checks from).

**Verdict**: **pass** — all 16 applicable constitution checks pass. No P0/P1/P2 failures.

---

## Constitution Checks

### I. Spec Fidelity (P0)

- **C-I.1** ✅ P0 — *spec.md System Overview / Integration Boundaries; plan ADR-2; interface Surface* — The command maps to a defined v5 operation: `POST /proposals/{proposal_id}/responses` (`createProposalResponse`). No invented endpoint.
- **C-I.2** ✅ P0 — *spec Input; plan Data Model; interface `--response` flag* — The request body matches `CreateProposalResponseRequest` (`{response:{value}}`) and the two values (`no_objection`, `bring_to_meeting`) are exactly the spec enum — no invented parameter or value. The `prr_`/`proposal_status` response fields trace to the `ProposalVote` schema.

### II. Action Transparency — NON-NEGOTIABLE (P0)

- **C-II.1** ✅ P0 — *spec Output; interface Output + Error Communication* — Success produces the recorded response (its `prr_` id + parent `proposal_status`) in machine-parseable form (`-o json`/`yaml` emit the `{data: ProposalVote}` document); the target resource (the `prp_` proposal, the responses endpoint) is traceable.
- **C-II.2** ✅ P0 — *spec Failure; interface Error Communication table* — Every failure row names a cause **and** a next step, and the token never appears in any message. Failures route through the format-aware `reportFailure` chokepoint (032).

### III. Fail Safe, Not Silent (P0→P1)

- **C-III.1** ✅ P0 — *spec Behavioral Accord (Input); plan ADR-1; tasks T003/T004* — The write is validated before it is sent: `--response` presence + closed-enum is checked fail-fast (`UsageError(2)`, no request). A 422/403/404 is surfaced, never swallowed.
- **C-III.2** ✅ P1 — *plan Cross-cutting; interface Interactions* — Recording is a single atomic `POST`; there is no multi-step write that could leave a partial state. The `422` (second response) is surfaced, not folded into success.

### IV. Test-Driven Development — RED→GREEN (P0)

- **C-IV.1** ✅ P0 — *tasks T001–T005 (RED-first unit tests); T005 BDD* — Every task names tests-first; user-facing behavior has executable acceptance scenarios in `response-recording.feature` (15 scenarios), with `@wip`/`@validation` discipline.
- **C-IV.2** ✅ P1 — *features/proposal-write-flow/response-recording.feature* — The acceptance scenarios exist before the implementing code (the feature file is written; T005 makes them pass).

### V. Composition over Monolith (P1)

- **C-V.1** ✅ P1 — *plan System Architecture; tasks T001/T002/T004* — A new `respond` verb leaf + an appended `ProposalVote` model + one new render key; no edits to unrelated commands, the shared `status.go`, or the transport seams (only consumes landed fields).

### VI. Size-Aware by Design — *N/A for this feature*

- **C-VI** ⊘ N/A — Response Recording is a single-resource write with no list or paginated result; there is nothing to page or truncate. No applicable checks. (Pagination/size-awareness is exercised by the read specs, e.g. Proposal Reads 056.)

### VII. Working Software (P1)

- **C-VII.1** ✅ P1 — *tasks T001–T005 acceptance criteria* — Each task pairs implementation with its tests and asserts `go build`/`go vet` clean; no code-only or test-only increment is implied.

### VIII. No Fabricated Data (P0)

- **C-VIII.1** ✅ P0 — *plan ADR-2; tasks T002 (explicit-absence guards); spec Output* — The render path uses explicit-absence guards for the nullable `proposal_id` and surfaces `proposal_status` verbatim; structured output passes raw server bytes through (`output.RenderSuccess`). No synthesized/placeholder value.

### IX. Writes Require Explicit Intent (P0)

- **C-IX.1** ✅ P0 — *spec System Overview; interface Surface; plan ADR-1* — Recording is reachable only through the explicit `proposal respond` write command (`ExactArgs(1)` + required `--response`); no read-shaped path issues this `POST`. Intent is the command itself (per the constitution's conflict-resolution: no interactive prompt needed).

### X. Respect API Limits (P1)

- **C-X.1** ✅ P1 — *plan Cross-cutting (non-idempotent retry); interface Interactions* — A `429` is surfaced as `RateLimited(5)`; the `POST` is **not** auto-retried (017 `isSafeMethod`), which honors the limit without risking a double-record for a non-idempotent write.
- **C-X.2** ✅ P1 — *plan ADR-3; spec Non-Behaviors* — Optimistic concurrency (`If-Match`/`ETag`) applies to *updates*; recording a response is an append-**create** with no prior `ETag`, so omitting `If-Match` is correct, not last-write-wins clobbering. Deliberately justified in ADR-3 and pinned by a BDD tripwire (header absent). *(See governance note 2.)*

### XI. Governance via Proposals (P0)

- **C-XI.1** ✅ P0 — *spec System Overview; PROJECT.md domain* — Recording a response **participates in** the proposal consent flow (it is how circulating proposals reach acceptance); it does **not** mutate governance structure directly, so no opt-in escape-hatch flag is required or present. The command operates on the `/proposals` flow, never bypassing it.

### XII. Standalone Executable — *N/A for this feature*

- **C-XII** ⊘ N/A — Response Recording introduces no language runtime or external dependency; the self-contained-binary guarantee is owned by the build/distribution specs (021/036/037), not exercised here. No applicable checks.

---

## Governance Infrastructure Notes

*(Separate from feature quality findings.)*

1. **No `accords/governance/done-*.md` accords found.** Checklist ran constitution checks only. Consider creating `accords/governance/done-specify.md`, `done-plan.md`, `done-interface.md`, `done-scenarios.md`, and `done-tasks.md` to enable done-criteria and cross-reference checks across the pipeline. (This is a project-wide gap, not specific to 058 — the prior specs ran under the same condition.)
2. **Principle X / If-Match interaction is a recurring judgment call for write specs.** Principle X reads "use `If-Match`/`ETag` for updates"; append-creates (Tension Capture 042, Response Recording 058) have no prior version and correctly omit it. The conflict table in CONSTITUTION.md addresses *updates*; an explicit note that creates are out of scope would remove the per-spec re-litigation. Advisory only.
3. **`guardian-agent.md` was not found** at the checklist skill's expected path — ran with SKILL.md alone (reduced character consistency, not a blocked skill).

---

## Calibration Summary

No principles required calibration — all 12 are concrete thresholds or prohibited/anti-patterns with observable detection mechanisms. Four principles (VI Size-Aware, XII Standalone) produced N/A for this single-write feature; X and XI produced passing checks with the contextual notes above.
