# Checklist: Identity Read

**Feature**: 011-identity-read
**Checked against**: CONSTITUTION.md (12 principles). No `accords/governance/done-*.md` accords present — done-criteria and cross-reference checks not generated (see Governance Infrastructure Notes).
**Artifacts checked**: spec.md, plan.md, interface-cli.md, interface-spec.md, features/self-service-reads/identity-read.feature, tasks.md
**Checks**: 16 (13 pass, 3 fail)
**Generated**: 2026-06-07

---

## Summary

| Severity | Count | Pass | Fail |
|---|---|---|---|
| P0 (blocking) | 9 | 9 | 0 |
| P1 (should fix) | 3 | 3 | 0 |
| P2 (consider) | 4 | 2 | 2 |
| **Total** | **16** | **14** | **2** |

No P0 failures. The P1 (Action Transparency error next-step) was **resolved during this guard session** — see "Resolution" below. Two P2 advisories remain (justified deferrals: projection parse-contract test; 429 backoff → 017). All in Action Transparency (II) and Respect API Limits (X), none re-shaping the spec.

**Resolution (P1 — II error next-step, addressed 2026-06-07)**: interface-cli.md § Error Communication now specifies a next step for each error class `me` owns — `CredentialError` ("fix or re-create with `glassfrog auth login`"), base-URL ("correct `--base-url`/env/file"), transport ("check connectivity"), in addition to the existing no-token message. The **non-2xx** next step (re-authenticate / back off) is correctly **deferred to API Error Extraction (015)** — 015 owns turning a non-2xx into a meaningful per-status message; 011 stays generic and surfaces the status code (spec fork 3). The check below is updated to passing.

---

## Constitution Checks: 14/16 passed

### Failures

**P2** | CONSTITUTION.md II (Action Transparency): "report … in machine-parseable form"
→ **interface-cli.md § Surface (Output)** / **spec.md Non-Behaviors**: The default output is a reshaped projection (labelled lines), with structured `--output json` deferred. The interface pins stable field set, order, and always-present ids, so the projection *is* parseable — but parseability relies on formatting convention rather than a structured contract, and no scenario pins the parse stability. Recommend a stability/parse-contract test (and note `--output json`, when it lands, hardens II). Advisory — not a violation given the pinned stable structure and the VISION agent-legible-output commitment.

**P2** | CONSTITUTION.md X (Respect API Limits): "backing off on `429` responses"
→ **plan.md § What This Plan Does Not Cover** / **interface-cli.md exit-code table**: `me` makes exactly one attempt and surfaces a `429` as a generic `APIError` (exit 3); backoff / `Retry-After` handling is deferred to Rate-Limit Handling (017). This is **not a 011 violation** (X's detection targets a retry loop that *ignores* `429`; `me` has no retry loop and surfaces the `429` honestly), but the system will not satisfy X's 429-backoff expectation until 017 lands. Cross-spec gap, recorded for traceability.

### Passed (14/16)

- **P1 | II Action Transparency (error next-step)** — *resolved this session*: interface-cli.md now specifies a next step for every error class `me` owns (no-token, `CredentialError`, base-URL, transport); the non-2xx next step is deferred to API Error Extraction (015) by design. (interface-cli.md § Error Communication)
- **P0 | I Spec Fidelity** — `me` maps to the real `getMe` operation (`spec/glassfrog-api-v5.yaml:966`); `--include roles` maps to the spec's `include` enum (only `roles` today); `validateInclude` rejects undefined targets, and `MeResponse` decodes the spec's documented shape. No invented endpoint, parameter, or behavior. (spec.md, interface-cli.md, interface-spec.md, plan.md ADR-1)
- **P0 | II Action Transparency (target traceability)** — the projection always surfaces the actor/organization ids (the machine-actionable handles), so output is traceable to the specific resource. (interface-cli.md, spec.md Behavioral Accord)
- **P0 | III Fail Safe** — every failure maps to a non-zero exit and a loud message; no swallowed errors; a 2xx that won't decode is a loud `DecodeError`, never a zero-valued projection; a non-2xx is never treated as success. (spec.md Failure accord, plan.md ADR-4, Cross-cutting)
- **P0 | IV Test-Driven Development** — tasks are RED-first (each task's acceptance criteria require failing-then-passing tests); the acceptance scenarios exist (`identity-read.feature`) before the command code; T005 makes them executable. (tasks.md T001–T005, features/…/identity-read.feature)
- **P0 | V Composition over Monolith** — `me` is a per-resource command over the shared API client; wiring is one `MustRegister` line; it edits no other command module. (The shared edits — the `Outcome`/`ExitCode` registry and the root `--base-url` flag — are the *sanctioned* extension points 004/010 reserved for the first consuming command, not unrelated-command coupling; the 003 help-test update is a consequence of the persistent flag.) See P2 note below. (plan.md ADR-2/ADR-3, tasks.md)
- **P0 | VIII No Fabricated Data** — `formatMe` renders only response-side fields; tolerant decode ignores unknown fields but synthesizes nothing; no defaulted/placeholder values are presented as real. (plan.md Data Model Design, Cross-cutting)
- **P0 | IX Writes Require Explicit Intent** — `me` is a pure read (`GET /me`); it issues no POST/PATCH/DELETE and mutates nothing. (spec.md, interface-cli.md)
- **P0 | XI Governance via Proposals** — `me` performs no governance-structure mutation; N/A by construction (read-only). (spec.md Non-Behaviors)
- **P0 | XII Standalone Executable** — adds no new runtime dependency; uses the existing Go stdlib (`net/http` via `apiclient`) and cobra; the artifact stays a self-contained binary. (plan.md, DECISIONS Go-self-contained precedent)
- **P1 | VII Working Software** — each task pairs implementation with tests and requires `go build`/`go vet` clean; the 010 dependency is gated so no task merges against non-compiling code. (tasks.md acceptance criteria, dependency graph)
- **P1 | II Action Transparency (operation legibility)** — for a self-invoked read the operator knows it ran `me`; the resource ids make the result traceable. (Passed; the broader "name the operation in output" is a VISION-level concern, not required for this read.)
- **P2 | VI Size-Aware by Design** — `/me` is a single resource (no pagination); the `--include roles` embed is the API's single-response array, rendered in full with no cap, so nothing is silently truncated. (spec.md Non-Behaviors, plan.md)
- **P2 | V (shared-file edits are the sanctioned extension)** — 011 modifies `dispatch.go`/`exitcode.go` (the single `Outcome`/`ExitCode` registry) and the root flag wiring; this is the explicitly-reserved extension point (004 ADR), not a monolith-coupling violation. (DECISIONS 004/010 entries)

---

## Governance Infrastructure Notes

*(separate from feature quality findings)*

- **No `accords/governance/done-*.md` accords exist.** Done-criteria and cross-reference checks were not generated — this checklist is constitution-only. Consider creating, to enable done-criteria gating in future runs:
  - `accords/governance/done-specify.md`, `done-plan.md`, `done-interface.md`, `done-scenarios.md`, `done-tasks.md`
  These would let checklist verify each artifact against its own done-criteria (and let cross-reference checks confirm tasks↔scenarios↔interface links), complementing the constitution checks. Their absence is a tooling gap, not a feature defect.
- **All constitution principles produced applicable checks** except none were dropped; II generated multiple checks (target traceability, machine-parseability, error next-step, operation legibility) because it is multi-clause and NON-NEGOTIABLE.

---

## Notes for the developer

- **The P1 (error next-step)** was resolved during this guard session: interface-cli.md now carries next-step messages for the error classes `me` owns, and the non-2xx next step is correctly assigned to API Error Extraction (015).
- The two remaining P2s are a robustness note (pin the projection's parse contract) and a known cross-spec deferral (429 backoff → 017), both accepted as justified deferrals.

No finding blocks implementation.
