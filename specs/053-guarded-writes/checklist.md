# Checklist: Guarded Writes

**Feature**: 053-guarded-writes
**Checked against**: CONSTITUTION.md (12 principles)
**Artifacts checked**: spec.md, plan.md, interface-spec.md, features/clobbered-changes/guarded-writes.feature, tasks.md
**Checks**: 12 (12 pass, 0 fail)
**Generated**: 2026-06-14

---

## Summary

All 12 checks pass. Constitution: 12/12. (Done-criteria + cross-reference: skipped — no `done-*` accords; see Governance Notes.)

P0: 12/12 passed · P1: 0 · P2: 0.

Two principles pass by **correctly-asserted structural inapplicability** (VI Size-Aware, XI Governance-via-Proposals). One principle (X Respect API Limits) is the one this feature exists to advance — 053 builds the `If-Match` send path that X requires; see the observations under Governance Notes.

---

## Constitution Checks: 12/12 passed

### Passed (12/12)

- **C-I — Spec Fidelity (P0)**: `If-Match` is the precondition header defined in the v5 spec's "Optimistic Concurrency (ETags)" section; sending it on a `PUT`/`PATCH`/`DELETE` is spec-defined behavior (FEATURE-MODEL §202; plan System Architecture). 053 invents no endpoint, parameter, or behavior — it sets a standard HTTP precondition header on write requests the CLI already issues. No new request *shape*; `Request` gains one optional field (`IfMatch`), not a new operation (plan ADR-1; interface-spec Surface).

- **C-II — Action Transparency (P0, NON-NEGOTIABLE)**: 053 introduces no new action and no new output — every landed read and write stays byte-identical (the `IfMatch` field zero-values to `""`, so no existing call site sets it and no outbound request gains an `If-Match` header), asserted by the spec non-behavior, plan ADR-1/ADR-2, and the validation scenario "Guarded Writes wires no production write command." Nothing the CLI reports to the operator changes. (The `If-Match` header is request metadata, not an action result to report — and no command sends it yet.)

- **C-III — Fail Safe, Not Silent (P0)**: Setting a header cannot fail (`Header.Set` has no error path). 053 introduces no write of its own and swallows no error; a refused guarded write (`412`) is surfaced through the existing generic `*ResponseError` path, not hidden and not reported as success (plan Cross-cutting; interface Error Communication table). 053 *advances* Fail Safe — it is the mechanism by which a stale write is refused rather than silently clobbering a concurrent edit (spec System Overview).

- **C-IV — Test-Driven Development (P0)**: T001 ships the `IfMatch` field + the `Execute` send together with unit tests covering every contract branch (non-empty → verbatim header, empty → no header, weak-validator preserved, method-agnostic on `DELETE`, composes with `ContentType`), and the `guarded-writes.feature` scenarios are authored now (`@wip`) as executable acceptance before the code that satisfies them (tasks T001; plan Cross-cutting Testing; spec Driving/Validation Scenarios).

- **C-V — Composition over Monolith (P0)**: The change is one additive field on the existing `apiclient.Request` plus one conditional block in the existing `Execute` seam — no edit to `Response`, the `executor` interface, `RetryExecutor`, `NewClient`, `buildURL`, or any command (plan ADR-1/ADR-2). It wires no call site, so no unrelated module is touched. Mirrors 042's narrow `ContentType` field exactly. Strongly conformant.

- **C-VI — Size-Aware, no silent truncation (P0)**: Inapplicable by structure, correctly asserted. 053 concerns a single write request's precondition header, not result-set paging. No collection walk, no `per_page` handling, nothing dropped or truncated (plan System Architecture; spec scope).

- **C-VII — Working Software (P0)**: T001 pairs implementation with its tests as one reviewable unit (the send "must not merge without the tests that pin its contract") — no code-only or test-only increment (tasks T001 Scope/Acceptance).

- **C-VIII — No Fabricated Data (P0)**: 053 sends only the version token a read actually captured (052's `Response.Version()`), **verbatim** — no quoting, unquoting, weak-validator stripping, or normalization — and sets *no* header (not a guessed/placeholder token) when the version is empty (plan ADR-1; interface-spec; feature "A weak-validator version is forwarded verbatim"). It presents nothing to the operator. The verbatim-forwarding + empty-as-absent contract is exactly No-Fabrication applied to a precondition token. Strongly conformant.

- **C-IX — Writes Require Explicit Intent (P0)**: 053 introduces no command and no read path that issues a `POST`/`PATCH`/`DELETE`; the `IfMatch` field is set only by a caller on a write request it explicitly constructs, and a read leaves it empty (no `If-Match` sent on reads). The mechanism only changes a header on writes that are already gated behind explicit write commands — it adds no new mutation path and no read-side mutation (plan ADR-2; interface Interactions "Caller-driven, conditional"). Strongly conformant.

- **C-X — Respect API Limits (P0)**: No violation, and this is the principle 053 advances. X requires "using optimistic concurrency (`If-Match`/`ETag`) for updates rather than last-write-wins clobbering"; the conflict table resolves to "always use optimistic concurrency (send `If-Match`)." 053 *builds the send path* X needs — 052 captured the `ETag`, 053 sends it as `If-Match`. X's detection targets "an update request that omits `If-Match` when an `ETag` is available"; 053 issues no update request (it wires no command — ADR-2), so it cannot violate X. It moves X from *blocked* (no send path existed) to *enabled* (the send path exists, awaiting per-command wiring). See observation below on full satisfaction.

- **C-XI — Governance via Proposals (P0)**: Inapplicable, correctly. 053 exposes no command path and mutates no governance structure — it is an internal request-side field. No default or opt-in governance-mutation path is introduced; the method-agnostic `If-Match` send applies only to writes a caller already constructs and introduces no governance-mutating surface of its own (plan ADR-2; spec Non-Behaviors).

- **C-XII — Standalone Executable (P0)**: The change adds one struct field and a few lines setting an existing `net/http.Header` — no new import, runtime, or external dependency (plan System Architecture: "No new imports"; interface-spec Package). The distributed binary's dependency profile is unchanged.

---

## Governance Notes

- **No `done-*` accords**: `accords/governance/` is absent, so done-criteria and cross-reference checks were skipped (constitution-only run, consistent with siblings 049/050/052). Consider creating `accords/governance/done-{specify,plan,interface,scenarios,tasks}.md` to enable artifact-quality checks across the pipeline.

- **Principle X reaches "enabled" at 053, "fully satisfied" only after the retrofits** (observation, not a finding): X is *fully* satisfied for a given update only once that write command actually populates `IfMatch` from `Version()`. 053 delivers the shared send path but wires no command (ADR-2, per the FEATURE-MODEL "retrofitting each write call-site is per-command work, not a capability here"). The standing X exposure on the existing write commands (tension update 044, tension discard 045) — updates that currently omit `If-Match` — is those specs'/their retrofits' scope, neither created nor closed by 053. 053 removes the *last shared blocker*: after 052 (capture) + 053 (send), a retrofit is a pure call-site change.

- **Unused-until-retrofit field** (observation, not a finding): the `Request.IfMatch` field has no caller until a write command retrofits onto it. This is a deliberate roadmap foundation (plan Risk 1; doc-comment names the consuming write commands; DECISIONS `053 → requires 052`, retrofits → require 053), not dead code — but a code reviewer may flag the unused field on the implementing PR. It builds clean (Go does not error on an unused struct field).

- **Sibling task-phrasing consistency** (advisory): sibling tasks (e.g. 049 T001–T004) state "RED-first" and "`go build`/`go vet` clean" explicitly in acceptance criteria. T001 here mandates tests-with-implementation and the contract but does not use that exact phrasing. Not a constitutional gap (IV/VII pass), but adding the explicit RED-first ordering and build/vet-clean line would match the established task style — carried forward from 052's checklist (the same advisory).
