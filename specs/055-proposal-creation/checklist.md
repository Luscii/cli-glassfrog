# Checklist: Proposal Creation

**Feature**: 055-proposal-creation
**Checked against**: CONSTITUTION.md (12 principles). No `done-*` governance accords found.
**Artifacts checked**: spec.md, plan.md, interface-cli.md, interface-spec.md, tasks.md, features/proposal-write-flow/proposal-creation.feature, spec/glassfrog-api-v5.yaml (API contract reference)
**Mode**: Pre-implementation (artifacts only — no `proposal` command code exists yet, by design)
**Checks**: 24 (24 pass, 0 fail)
**Generated**: 2026-06-15

---

## Summary

All 24 checks pass. Constitution: 24/24. Done-criteria: not run (no `done-*` accords). Cross-references: 4/4 (reported informationally under Cross-Reference Checks).

| Severity | Count | Pass | Fail |
|---|---|---|---|
| P0 (blocking) | 22 | 22 | 0 |
| P1 (should fix) | 2 | 2 | 0 |
| P2 (consider) | 0 | 0 | 0 |
| **Total** | **24** | **24** | **0** |

Severity note: all twelve principles are framed `MUST` / `MUST NOT` / `NON-NEGOTIABLE` → P0. The two P1 checks are link-presence cross-reference checks (severity P1 by the cross-reference rule), reported here because no `done-*` accord generated them.

---

## Constitution Checks: 22/22 passed

All constitution checks pass. Each principle was translated into binary assertions against the pre-implementation artifact set and the API contract (`spec/glassfrog-api-v5.yaml`). Calibration was applied to Principles II, III, IV, and VII (broad imperatives) to make them binary against a spec/plan/interface/tasks set rather than running code; the calibrated assertions are stated inline below.

### Principle I — Spec Fidelity (P0, MUST / MUST NOT)

*Assertion 1*: The command maps to an operation the API spec defines.
→ **Pass**. spec.md §System Overview and §Integration Boundaries name `POST /proposals` (`createProposal`); `spec/glassfrog-api-v5.yaml:3699` defines `operationId: createProposal` at `POST /proposals`. Match.

*Assertion 2*: The request body shape matches the spec operation's schema.
→ **Pass**. interface-spec.md §Surface and plan ADR-4 pin `{proposal: {tension_id, changes}}`; `glassfrog-api-v5.yaml:7662` `CreateProposalRequest` is `{proposal: {tension_id (required), changes: array of ProposalChange}}`. Match — no invented parameter.

*Assertion 3*: The client invents no parameter or behavior the spec does not define (no `status`, no `proposer`, no `If-Match`).
→ **Pass**. spec.md §Non-Behaviors and interface-cli.md/interface-spec.md explicitly exclude a `--status` flag, a proposer field, and an `If-Match` header; the API request schema carries neither `status` nor `proposer`. The stricter-than-spec floor (non-empty `changes`) is a client-side tightening of a spec-permitted-but-optional field, not an invented parameter — blessed by the constitution's "Spec-permitted is not spec-required" conflict rule and justified in spec.md Assumptions + plan ADR-3. No undefined endpoint, parameter, or behavior.

### Principle II — Action Transparency (P0, NON-NEGOTIABLE)

Calibrated assertions for a pre-implementation artifact set:

*Assertion 1*: The created result is traceable to the spec operation and the target resource (a structured, machine-parseable form is specified).
→ **Pass**. spec.md §Output and interface-cli.md §Output specify the created proposal — with its `prp_` id, `draft` status, anchor `tension_id` — rendered through Output Format Selection (020), with `json`/`yaml` emitting the structured `{data}` document. The action targets `POST /proposals` and produces the `prp_` resource id.

*Assertion 2*: Every error names both the cause and a next step.
→ **Pass**. spec.md §Failure ("the message names both what went wrong and a concrete next step"), interface-cli.md §Error Communication (every row carries a cause + next step), and the feature file (e.g. "report 'not authenticated' and point to 'glassfrog auth login'") all specify cause-plus-next-step diagnostics.

### Principle III — Fail Safe, Not Silent (P0, MUST / MUST NOT)

*Assertion 1*: The write is validated before it is sent (fail-fast, no request on bad input).
→ **Pass**. spec.md §Input ("caught before any write"), plan §Data flow + ADR-3 ("fail-fast, no request"), interface-cli.md §Interactions (validation order 1–6, all pre-network), and the feature file's five "no request will be sent" scenarios all pin pre-send validation.

*Assertion 2*: No partial-apply / no swallowed errors / no failure-reported-as-success.
→ **Pass**. The operation is a single non-idempotent `POST` with no multi-step write to leave partial (plan §Cross-cutting). Failures route through `reportFailure`/`classifyClientError` and exit non-zero (interface-cli.md §Error Communication). The §133 retry note explicitly prevents a silent re-send (double-submit) on `429`.

### Principle IV — Test-Driven Development (P0, MUST)

Calibrated for pre-implementation: user-facing behavior must have an executable acceptance scenario authored before implementation, and tasks must be RED-first.

*Assertion 1*: User-facing behavior has executable acceptance scenarios.
→ **Pass**. features/proposal-write-flow/proposal-creation.feature exists with 16 scenarios across four Rule blocks, mapped to the spec's driving scenarios; T005 makes the behavioral scenarios executable acceptance.

*Assertion 2*: Tasks specify test-first (RED before GREEN).
→ **Pass**. tasks.md T004 states "RED-first unit tests for every branch"; T001/T002/T003 each carry test counts and acceptance criteria ahead of implementation; T005 is the executable-acceptance step.

### Principle V — Composition over Monolith (P0, MUST / MUST NOT)

*Assertion*: The feature is built from modular parts that add a command without changing unrelated ones.
→ **Pass**. plan §Components and tasks.md add new files only (`internal/cli/proposal.go`, `internal/glassfrog/proposal.go`, one new `internal/render` key) and consume shared packages unchanged. tasks.md §Branching Guidance confirms "no change to shared packages." ADR-1 reserves the `proposal` namespace for siblings without coupling.

### Principle VI — Size-Aware by Design (P0, MUST / NEVER)

*Assertion*: No silent truncation; pagination handled or boundary signalled where applicable.
→ **Pass (applicable, satisfied)**. A create is a single non-paged request producing one resource. interface-spec.md §Interactions ("One request, no walk … no pagination") and plan §Output state there is no result set to page. The change set is sent verbatim (interface-spec.md §Interactions "Verbatim change set"), so no element is dropped or truncated.

### Principle VII — Working Software (P0, MUST)

Calibrated for tasks: each task ships implementation with its tests and validates/builds.

*Assertion*: Every task pairs code with tests and asserts build/vet clean.
→ **Pass**. Each task (T001–T005) lists test counts and an acceptance criterion requiring `go build`/`go vet` clean; no code-only or test-only task. T005's acceptance criterion includes `go build ./...` / `go vet ./...` and the feature suites running clean.

### Principle VIII — No Fabricated Data (P0, MUST / MUST NOT)

*Assertion 1*: Output presents only API-returned data; nullable fields are not filled with invented values.
→ **Pass**. interface-cli.md §Output and interface-spec.md §`internal/render` require explicit-absence guards (`{{if}}…{{else}}(none){{end}}`) on nullable `tension_id`/`circle_id`/`proposer_id`, "never `<no value>` or an invented value" (tasks.md T003 acceptance). spec.md §Non-Behaviors forbids asserting a status the server owns.

*Assertion 2*: The created status is not synthesized by the client.
→ **Pass**. spec.md Assumptions ("[ASSUMED] the created proposal is always `draft`: the command does not assert the returned status … renders whatever the server returns") and §Non-Behaviors ("must not set, override, or interpret the … status").

### Principle IX — Writes Require Explicit Intent (P0, MUST / NEVER)

*Assertion*: The mutation occurs only as the direct result of an explicit write command; no read path mutates.
→ **Pass**. The mutation is gated behind the explicit `proposal create` write verb. spec.md §Non-Behaviors defers all read-shaped siblings (list/get) to separate specs; ADR-1 keeps `--changes` on the `create` leaf only, so a future read cannot carry it. No read/get/list path issues a `POST` here.

### Principle X — Respect API Limits (P0, MUST)

*Assertion 1*: `429` is honored, not ignored.
→ **Pass**. plan §Cross-cutting and interface-cli.md/interface-spec.md specify `429` → `RateLimited(5)` surfaced on first occurrence via the landed `RetryExecutor`; the feature file's "A rate-limited create is surfaced, not silently re-sent" scenario pins it.

*Assertion 2*: Optimistic concurrency (`If-Match`) used where an `ETag` is available.
→ **Pass (applicable, satisfied)**. A create has no prior `ETag` (spec.md §Non-Behaviors, plan §Cross-cutting, interface-spec.md §Consistency Notes), so `If-Match` is correctly omitted — the principle's `ETag`-available precondition does not hold for a create. This is consistent with the constitution's conflict-resolution note that concurrency guards apply to edits, not creates.

### Principle XI — Governance via Proposals (P0, MUST / MUST NOT)

*Assertion*: This governance change goes through the `/proposals` flow; no default path mutates governance structure directly.
→ **Pass (strongly affirming)**. This feature *is* the `/proposals` write path — the constitutionally-sanctioned default route for governance change. spec.md §System Overview frames it as "the anchor of the governance write path." It exposes no direct-mutation path and no `role-manage-without-proposal` escape hatch (out of scope, deferred). No violation; the feature embodies the principle.

### Principle XII — Standalone Executable (P0, MUST)

*Assertion*: The feature adds no new runtime/interpreter/external dependency beyond the binary + network to the API.
→ **Pass**. plan §Components and interface-spec.md add only Go stdlib + existing internal packages; the `--changes` source uses `os.Stat`/`os.ReadFile`/`os.Stdin` (host OS, no extra software). The only external dependency remains network access to the Glassfrog API.

---

## Done-Criteria Checks: not run

No `done-*` governance accords exist (no `accords/` directory in the repository). Done-criteria checks were not generated. See Governance Notes.

---

## Cross-Reference Checks: 4/4 passed

These link-presence checks (severity P1) are reported informationally — no `done-*` accord required them, but they verify the artifact set is internally wired. They check presence, not consistency (consistency is analyze's domain).

- **P1** | tasks.md → spec.md / interface artifacts: every task carries a "Plan reference" and "Interface references" field → **Pass** (T001–T005 all present).
- **P1** | tasks.md T004/T005 → feature file: scenario-reference fields name `proposal-creation.feature` scenarios → **Pass** (T004 lists 13 scenarios; T005 references the behavioral Rule-block scenarios).
- **P1** | feature file → spec.md: each scenario carries a `# Source: 055-proposal-creation — Scenario: …` provenance comment → **Pass** (all 16 scenarios traced; 2 marked `Proposed:`/architecture-informed with their rationale).
- **P1** | spec/plan/interface → API contract: `createProposal` / `POST /proposals` / `CreateProposalRequest` / `ProposalChange` references resolve in `spec/glassfrog-api-v5.yaml` → **Pass** (all four resolve at the cited shapes).

---

## Governance Notes

Informational — these are infrastructure gaps, not feature quality findings.

- **No `accords/` directory / no `done-*` accords found.** Done-criteria checks were not generated. Consider creating `accords/governance/done-specify.md`, `done-plan.md`, `done-interface.md`, `done-scenarios.md`, and `done-tasks.md` to enable per-artifact quality checks. Until then, checklist runs constitution checks only. (interface-cli.md §Consistency Notes and interface-spec.md §Consistency Notes both independently note "No `accords/` directory exists," consistent with this.)
- **Calibration applied** to Principles II, III, IV, VII — broad imperatives concretized against the spec/plan/interface/tasks set rather than running code (this is a pre-implementation run). Every calibrated assertion is binary and stated inline above.
- **Principles VI and X have applicability conditions** (pagination; `ETag`-gated concurrency) that do not arise for a single-resource create; both produced an applicable, satisfied check rather than being skipped, with the reason recorded.
- **API-contract source.** Principle I checks were evaluated against `spec/glassfrog-api-v5.yaml` (the in-repo API spec the constitution's "diff against `spec.yaml`" detection refers to). The file is named `glassfrog-api-v5.yaml`, not `spec.yaml`; the constitution's literal filename reference is slightly stale but resolves unambiguously.
