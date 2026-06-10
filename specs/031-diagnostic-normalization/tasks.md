# Tasks: Diagnostic Normalization

**Feature**: 031-diagnostic-normalization
**Concretization**: Full context (plan + spec + interface + scenarios)
**Inputs**: plan.md, spec.md, interface-spec.md, features/opaque-failures/diagnostic-normalization.feature

---

## Dependency Graph

Phase 1: Diagnostic consolidation and refinements (2 tasks, no phase dependencies) — single-phase build [Shared]

2 tasks total | 1 phase | Builder: pipeline

T002 depends on T001 (the behavior changes apply to the consolidated `Diagnose`).

---

## Branching Guidance

**Pipeline mode**: `spec/031-diagnostic-normalization/base` → `spec/031-diagnostic-normalization/task-1`, `spec/031-diagnostic-normalization/task-2`

T001 lands first (pure consolidation, suite stays green); T002 branches from the integrated result.

---

## Phase 1: Diagnostic consolidation and refinements [Shared]

- [x] **T001** [Shared] Introduce the `Diagnostic` value and `Diagnose` normalizer (behavior-preserving consolidation) — 5 scenarios un-@wip'd (transport, permission-detail, api-error-no-detail, success, unrecognized); usage-from-dispatch left @wip (deferred to 032 per ADR-3, see LEARNINGS)
  - **Scope**: In `internal/cli`, add `Diagnostic{Category Outcome; Cause string; NextStep string}` and a total `Diagnose(err error) Diagnostic` that folds the existing `classifyClientError` (category), `formatClientErrorMessage` (cause), and `clientErrorNextStep` (next step) into one `errors.As` chain — verbatim, with **no** behavior change (decode still `RuntimeError`, the permission hint still combined, the rate-limit wording still "retry later"). Add an unexported `renderDiagnostic(Diagnostic) string` that composes `Cause` (when `NextStep == ""`) or `Cause + " — " + NextStep`. Refactor `reportClientError` to `err = refineClientError(err); d := Diagnose(err); print renderDiagnostic(d); return d.Category, err`. Switch the three category-only callers to `Diagnose(err).Category` (or retain `classifyClientError` as a one-line `Diagnose(err).Category` shim): `me.go:175`, `roles.go:215`, `subroles.go:237`.
  - **Acceptance criteria**:
    - A table-driven `Diagnose` test asserts `Category` equals the pre-change `classifyClientError(err)` for every family (auth no-creds/cred-error, transport, 401/403/429/other non-2xx, decode, base-URL, rcfile read/format, output.FormatError, fail-safe), guarded by a `len`+comma-ok exhaustiveness check so a dropped arm fails loudly.
    - `renderDiagnostic(Diagnose(err))` is byte-equivalent to the pre-change `formatClientErrorMessage(err)` for every family — pinned by golden capture before/after (capture stderr via a temp file, not `os.Pipe`).
    - `reportClientError` delegates to `Diagnose`; its stderr output and returned `Outcome` are unchanged; existing `me`/`roles`/`subroles` tests and `clienterror_test.go` pass untouched.
    - No `Diagnose` arm emits the `X-Auth-Token`: a token-never-in-output test covers `.Cause`, `.NextStep`, and the rendered line for every arm.
    - `go build` + `go vet` + full `go test ./...` are green (CONSTITUTION VII).
  - **Dependencies**: None
  - **Plan reference**: Phase (Implementation Strategy step 1), ADR-1: Consolidate into a single `Diagnostic` value
  - **Scenario references**: diagnostic-normalization.feature: "Every failure family produces the same diagnostic shape", "A diagnostic exposes only its observable fields", "No diagnostic output carries the auth token", "A successful outcome is never normalized", "An unrecognized failure falls back to the internal-error diagnostic"
  - **Interface references**: interface-spec.md: Surface (`Diagnostic`, `Diagnose`, `renderDiagnostic`, `classifyClientError` delegate); Error Communication (byte-equivalence + token-free invariants)

- [ ] **T002** [Shared] Apply the clarified behavior changes (decode→APIError, 401/403 split, 429 reset-window)
  - **Scope**: In the consolidated `Diagnose`: (a) reclassify a `*apiclient.DecodeError` from `RuntimeError` to `APIError` (cause/next-step wording unchanged); (b) split the permission next step — 401 → "verify the configured API token", 403 → "check that the configured identity has the required role membership / permission"; (c) refine the 429 next step to reference the reset window (`Retry-After` / `X-RateLimit-Reset`). Update `clienterror_test.go:38` (`decode-is-runtime` → `APIError`) and grep every other decode→`RuntimeError`/exit-1 assertion, updating together. Add a `/score:deprecate` candidate note recording that the prior decode-classification precedent is retired.
  - **Acceptance criteria**:
    - A `*DecodeError` yields `Diagnose(...).Category == APIError`, and `ExitCode` returns `3`; `clienterror_test.go:38` asserts `APIError`; a repo grep shows no surviving decode→exit-1 / decode→`RuntimeError` assertion.
    - A 401 and a 403 both classify as `PermissionError` (exit 4) but render distinct next steps (401 = token; 403 = membership/permission).
    - A 429 classifies as `RateLimited` (exit 5), renders a next step referencing the reset window, and triggers no wait or retry.
    - A render-template failure (019) and the fail-safe arm still classify as `RuntimeError` (exit 1) — regression guard that the decode change didn't widen.
    - The token-free invariant still holds for the three changed arms; `go build` + full `go test ./...` green.
  - **Dependencies**: T001
  - **Plan reference**: Implementation Strategy step 2, ADR-2: DIVERGENCE — decode → APIError(3); Diagnostic Composition (next-step refinements)
  - **Scenario references**: diagnostic-normalization.feature: "An undecodable 2xx body is normalized to a general API error", "A decode error exits with the general API code", "A 401 and a 403 carry distinct next steps", "A rate-limited response surfaced after retries carries a reset-window next step", "The most-specific category wins on an overlapping status"
  - **Interface references**: interface-spec.md: Status → Category → exit-code mapping (decode→3), Next-step contract (401/403/429)
  - **Risk**: ⚠️ Divergence from shipped precedent — decode reclassified 1→3 changes a published exit-code behavior; grep all decode→exit-1 assertions before changing and record the `/score:deprecate` entry so the retired precedent is explicit.
