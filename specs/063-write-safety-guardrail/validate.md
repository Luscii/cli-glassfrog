# Validate: Write-Safety Guardrail

**Feature**: 063-write-safety-guardrail
**Round**: 1 of 3
**Date**: 2026-07-17
**Verdict**: Ready
**Artifacts loaded**: spec.md, plan.md, tasks.md (3 of 3 complete), interface-spec.md, PROJECT.md, features/unequipped-agent-operators/write-safety-guardrail.feature
**Implementation files**: 6 — plugin hook artifacts (`plugin/hooks/hooks.json`, `plugin/hooks/gated-commands.txt`, `plugin/hooks/glassfrog-write-gate.sh`) + build-side companions (`internal/build/writesafetyguardrail.go`, `write_safety_guardrail_guard_test.go`, `write_safety_guardrail_bdd_test.go`)

---

## Conformance Summary

| Dimension | Status | Findings |
|---|---|---|
| Driving scenario coverage | ✓ Pass | 0 |
| Acceptance criteria | ✓ Pass | 0 |
| Interface contract conformance | ✓ Pass | 0 |
| Non-behavior absence | ✓ Pass | 0 |
| @wip lifecycle completion | ✓ Pass | 0 |
| **Validation scenarios** | ✓ Satisfied | 0 |

**Total**: 5 dimensions checked, 5 passed, 0 findings; 5 of 5 validation scenarios satisfied (independently verified against the real gate + shipped CLI).

---

## Driving Scenario Coverage

**Status**: Pass (8 of 8 driving scenarios covered)

Every driving scenario in spec.md is concretized in the feature file and traces to a code path in the gate script (`plugin/hooks/glassfrog-write-gate.sh`) or its modelled host loop. All 11 non-`@validation` feature scenarios execute green in the `internal/build` BDD suite (which drives the real script).

| Driving scenario (spec.md) | Status | Implementation |
|---|---|---|
| Confirming a proposal write before it runs | ✓ Covered | `classify_segment` → `ask` naming command/target/effect |
| A read passes through ungated | ✓ Covered | non-`proposal` glassfrog path → `allow` (empty output) |
| Performing exactly the confirmed write | ✓ Covered | gate only classifies; the host runs the byte-identical command (BDD `thenNoBroaden`) |
| Confirmation withheld | ✓ Covered | decision `ask`; modelled host does not send without confirmation |
| Stale-write refusal triggers re-read and re-confirm | ✓ Covered | retry is itself a proposal write → re-gated to `ask` (ADR-5) |
| Re-confirmation withheld after a stale-write re-read | ✓ Covered | withheld re-confirm → not retried, resource unchanged |
| An operational tension edit passes through ungated | ✓ Covered | `tension create/update/discard` → `allow` (empirically confirmed) |
| A non-stale-write failure is not treated as a clobber | ✓ Covered | gate is `PreToolUse` only, reads no `$?`/outcome — no recovery path to invoke |

---

## Acceptance Criteria

**Status**: Pass (3 of 3 tasks complete, all criteria met)

| Task | Status | Evidence |
|---|---|---|
| T001 — registry + PreToolUse hook registration | ✓ Met | `hooks.json` parses; PreToolUse/`matcher:"Bash"`; single `type:"command"` (never `prompt`) rooted at `${CLAUDE_PLUGIN_ROOT}` via `bash`, `timeout:10`; registry lists exactly the 4 write leaves, no reads/tension; wired via default hooks path (no manifest `hooks` key), keeping 062's `ManifestDemandsNoSetup` true (`TestGateRegistrationWellFormed`, `TestGatedRegistryListsExactlyTheProposalWrites`, `TestGuardrailKeepsManifestSetupFree`) |
| T002 — gate script (recognizer + emitter) | ✓ Met | recognized write leaf → `ask` naming command/target/effect; unrecognized `proposal` subcommand → `ask` (fail-closed); reads/tension/non-glassfrog → `allow`; malformed stdin fails safe; decision keys only on parsed command path; pure bash (no jq/sed/grep) |
| T003 — drift tripwire in `internal/build` | ✓ Met | asserts each registry leaf exists on the CLI proposal surface + full surface matches the checked-in expectation; names the offending command; documents partial coverage (`TestWriteSafetyRegistryDriftGuard`) |

---

## Interface Contract Conformance

**Status**: Pass (all surface contracts conformant)

| Contract (interface-spec.md) | Status | Implementation |
|---|---|---|
| `hooks.json` schema (PreToolUse, matcher Bash, type command, `${CLAUDE_PLUGIN_ROOT}`-rooted, bounded timeout) | ✓ Conformant | `plugin/hooks/hooks.json` matches field-for-field |
| Hook stdin contract (`tool_name`, `tool_input.command`) | ✓ Conformant | `extract_scalar_string`/`extract_command` read exactly these |
| Hook stdout contract (`hookSpecificOutput.permissionDecision` + reason) | ✓ Conformant | emits `permissionDecision` plus both `permissionDecisionReason` and top-level `systemMessage` (interface flagged the reason spelling as host-version-specific; emitting both is the robust reading) |
| Decision table (write → ask; unrecognized proposal → ask; reads/tension/non-glassfrog → allow) | ✓ Conformant | empirically verified across all command classes |
| Gated-command registry (the 4 proposal-write leaves) | ✓ Conformant | `gated-commands.txt` single-sourced, read by script + drift test |

---

## Non-Behavior Absence

**Status**: Pass (0 of 8 non-behaviors violated)

| Non-behavior (spec.md) | Status | Evidence |
|---|---|---|
| Must not gate read-only commands | ✓ Absent | reads → `allow` |
| Must not gate operational tension edits | ✓ Absent | `tension create/update/discard` → `allow` |
| Must not add/remove/modify any CLI command, flag, or capability | ✓ Absent | diff touches only `plugin/`, `internal/build/`, `features/`, `specs/` — no `internal/cli`/`cmd`; plugin tree carries no Go code |
| Must not re-validate/reimplement governance logic locally | ✓ Absent | gate only classifies the command path; issues no request, decides no correctness |
| Must not blindly retry a stale-write refusal | ✓ Absent | gate holds no retry/auto-send/`$?` logic; a retry re-enters the gate |
| Must not auto-confirm or self-authorize a write | ✓ Absent | writes emit `ask` only — the gate never force-`allow`s a write; confirmation is the human's |
| Must not coach Holacracy / judge governance merits | ✓ Absent | message states command/target/effect; no soundness/advice vocabulary present |
| Spec must not define distribution/packaging | ✓ Absent | no install/distribution machinery added (deferred to #70) |

---

## @wip Lifecycle Completion

**Status**: Pass

All 11 non-`@validation` scenarios referenced by the checked tasks have had `@wip` removed and execute in the BDD suite. The 5 remaining `@wip` tags are exclusively on `@validation` scenarios — the intentional held-out marker (consistent with the 062 precedent and this pipeline's convention that `@validation` scenarios are retained for the Guardian, not run by the Builder's suite). No non-validation scenario retains `@wip`. Not a lifecycle defect.

---

## Validation Scenario Results

**Status**: Satisfied (5 of 5 traced independently to implementation)

Each was verified against the *real* gate script and, for VS2, the freshly-built CLI binary — not assumed from the driving-scenario pass.

| Scenario | Status | Trace |
|---|---|---|
| No proposal-write path bypasses confirmation | ✓ Satisfied | all four write leaves (create/propose/respond/withdraw) → `permissionDecision:"ask"`; an unrecognized `proposal` subcommand also `ask` (fail-closed) — no leaf yields a write-without-asking path |
| Guardrail names no command the CLI lacks | ✓ Satisfied | built `glassfrog` binary; `proposal --help` exposes create/propose/respond/withdraw; the drift tripwire enforces this continuously |
| Confirmation states the change, not its governance merits | ✓ Satisfied | message = "Governance write: `…` will advance/create/record/withdraw … Confirm to proceed"; scan for should/recommend/wise/sound/correct → none |
| No retry without an interposed re-confirmation | ✓ Satisfied | gate has no retry/auto-send/`$?` reference; re-running a refused write re-gates to `ask`, so a retry cannot be sent without fresh confirmation |
| Tension edits stay outside the gate | ✓ Satisfied | `tension create/update/discard` → `allow`; the gate scopes to the proposal write path only |

---

## Verdict: Ready

All 5 conformance dimensions pass and all 5 validation scenarios are satisfied through independent inspection and empirical exercise of the artifacts. The implementation delivers exactly what the specification promised: an operator-layer `PreToolUse` gate that routes every proposal-path governance write to explicit human confirmation, fail-closed within the `proposal` namespace, while reads and operational tension edits pass ungated — adding no CLI command, flag, or capability, re-validating nothing, and interposing no blind retry. The best-effort drift tripwire keeps the gated registry truthful to the CLI's proposal surface, with its partial coverage stated rather than implied.

One item is correctly surfaced-not-silently-actioned (already documented in plan.md ADR-1 and the implement handoff, not a validate finding): the spec's `[ASSUMED] delivered as a skill` was resolved to *a hook* during shaping; and FEATURE-MODEL's broader gated-set enumeration ("tension capture, proposals, responses") was deliberately narrowed to the proposal write path (spec Assumptions / Clarifications). Both are noted for the developer's reconciliation, outside validate's conformance scope.

---

## Next Steps

Implementation conforms to the specification. Suggest PR review and merge (`gh pr create --base main`). The specification loop is closed.

Optional follow-through for the developer (not blockers): reconcile FEATURE-MODEL's gated-set enumeration with the narrowed proposal-path scope, per the spec's own Assumptions note.
