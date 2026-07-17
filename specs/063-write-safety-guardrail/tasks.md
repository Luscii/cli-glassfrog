# Tasks: Write-Safety Guardrail

**Feature**: 063-write-safety-guardrail
**Concretization**: Full context
**Inputs**: plan.md, spec.md, interface-spec.md, features/unequipped-agent-operators/write-safety-guardrail.feature

---

## Dependency Graph

Phase 1: Recognizer + registry + hook (2 tasks, no phase dependencies; intra-phase: T002 depends on T001) [Shared]
Phase 2: Drift tripwire (1 task, depends on Phase 1 — specifically on T001; parallelizable with T002) [Shared]

3 tasks total | T002 and T003 parallelizable after T001 | Builder: pipeline (single active spec)

---

## Branching Guidance

**Pipeline mode**: `spec/063-write-safety-guardrail/base` → `spec/063-write-safety-guardrail/task-1`, `spec/063-write-safety-guardrail/task-2`, `spec/063-write-safety-guardrail/task-3`

T002 depends on T001; T003 also depends only on T001, so once T001 lands on the spec base, T002 and T003 can be built in parallel and land independently.

---

## Phase 1: Recognizer + registry + hook [Shared]

- [x] **T001** [Shared] Add the gated-command registry and the PreToolUse hook registration — hooks.json + gated-commands.txt + build-side readers/validators; 3 guard tests; default-path discovery keeps 062's manifest setup-free (its 2 scenarios are @validation, held for validate)
  - **Scope**: Add `plugin/hooks/hooks.json` registering a `PreToolUse` hook with `matcher:"Bash"`, `type:"command"`, and a `${CLAUDE_PLUGIN_ROOT}`-rooted command pointing at the gate script, plus the single gated-command registry (a data file or a constant the gate script and the drift test both read) enumerating exactly the four proposal-write leaves: `proposal create`, `proposal propose`, `proposal respond`, `proposal withdraw`. Wires the gate in; no recognition logic yet. Adds no Go CLI code.
  - **Acceptance criteria**:
    - `plugin/hooks/hooks.json` parses as JSON and registers a `PreToolUse` entry with `matcher:"Bash"` and a single `type:"command"` hook whose `command` is rooted at `${CLAUDE_PLUGIN_ROOT}` with a bounded `timeout`; it is `type:"command"` (deterministic), never `type:"prompt"`
    - The plugin is wired to load the hook (either a top-level `"hooks":"./hooks/hooks.json"` key in the existing `plugin/.claude-plugin/plugin.json`, or the default `./hooks/hooks.json` path) without altering 062's manifest identity or its orientation skill
    - The registry lists exactly the four proposal-write leaves and contains no read or `tension` command; it is single-sourced so T002 and T003 read the same definition
    - The gate script's runtime is pinned to `bash` — the interpreter the `hooks.json` `command` invokes (`bash "${CLAUDE_PLUGIN_ROOT}/hooks/<gate-script>"`) and the one installed-plugin hook examples assume — introducing no other interpreter dependency (closes the checklist P2 on XII-adjacent runtime portability)
  - **Dependencies**: None
  - **Plan reference**: Phase 1; ADR-1 (operator-layer hook), ADR-3 (static registry), ADR-4 (single-sourced registry)
  - **Scenario references**: write-safety-guardrail.feature: "No proposal-write path bypasses confirmation", "Tension edits stay outside the gate"
  - **Interface references**: interface-spec.md — Surface (structural layout, `hooks.json` schema, gated-command registry)

- [ ] **T002** [Shared] Implement the gate script — recognizer plus permission-decision emitter
  - **Scope**: The `PreToolUse` gate script: read the tool-call JSON on stdin, and for a `Bash` call parse `tool_input.command`, resolve the `glassfrog` invocation token (bare name, absolute/relative path, leading `VAR=val` env prefix) and the subcommand path, classify against the registry, and emit the permission decision on stdout. No CLI code; lives under `plugin/hooks/`.
  - **Acceptance criteria**:
    - A recognized proposal-write leaf → `{"hookSpecificOutput":{"permissionDecision":"ask"}}` with a message naming the command, the target id, and the effect; a write is sent only after the practitioner confirms
    - An unrecognized `glassfrog … proposal` subcommand → `ask` (fail-closed within the `proposal` namespace)
    - Reads, `tension create/update/discard`, and any non-`glassfrog` command → `allow` (or empty output) with no prompt
    - On a malformed stdin or internal error the script fails safe: it never blocks unrelated `Bash`; a command it has positively identified as a `glassfrog proposal` write but cannot fully parse errs to `ask`
    - The decision keys only on the parsed command path against the registry — never on the agent's stated intent or the command's flags; the script re-validates no governance logic and adds no CLI surface
  - **Dependencies**: T001
  - **Plan reference**: Phase 1; ADR-2 (`ask` reuses the host checkpoint), ADR-3 (recognizer, fail-closed), ADR-5 (no re-read guidance; retry re-gated)
  - **Scenario references**: write-safety-guardrail.feature: "Gate a proposal write behind explicit confirmation", "Run only the write that was confirmed", "Decline a proposal write when confirmation is withheld", "Gate an unrecognized proposal subcommand fail-closed", "Re-confirm a retry after a stale-write refusal", "Hold off the retry when re-confirmation is withheld", "Leave a non-stale failure to normal handling", "Let a read run without confirmation", "Let a tension edit run without confirmation", "Confirmation states the change, not its governance merits", "Guardrail names no command the CLI lacks", "No retry without an interposed re-confirmation"
  - **Interface references**: interface-spec.md — Surface (hook stdin/stdout contract), Interactions (recognition rules, stale-write re-confirmation), Error Communication
  - **Risk**: R1 — command-string parsing robustness (chaining, quoting, aliases) has integrity stakes; cover with unit tests over command-string variants

---

## Phase 2: Drift tripwire [Shared]

- [ ] **T003** [Shared] Add the best-effort drift tripwire in `internal/build`
  - **Scope**: A new `internal/build` test anchoring the gated-command registry to the CLI's `proposal` subcommand surface, so a new or renamed proposal write command cannot silently ship ungated. Best-effort and explicitly partial; if an anchor proves infeasible, state the reduced coverage rather than dropping it silently.
  - **Acceptance criteria**:
    - Test asserts each registry leaf (`proposal create/propose/respond/withdraw`) still exists on the CLI's `proposal` command
    - Test asserts the CLI's `proposal` subcommand surface matches a checked-in expectation, failing when it grows or is renamed without the registry being updated
    - Test fails loudly and names the offending command when the surface diverges; any anchor deliberately left uncovered (e.g. the hook's parsing robustness, which it does not check) is documented in the test, not omitted silently
  - **Dependencies**: T001
  - **Plan reference**: Phase 2; ADR-4 (best-effort `internal/build` tripwire)
  - **Scenario references**: write-safety-guardrail.feature: "Drift tripwire fails when a gated leaf leaves the CLI"
  - **Interface references**: interface-spec.md — Error Communication (drift tripwire fails → CI red)
  - **Risk**: R4 — coverage is partial (enumerable surface only, not parser robustness); state the partiality, never imply total coverage
