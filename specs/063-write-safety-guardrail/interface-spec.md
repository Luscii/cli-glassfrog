# Interface Accord: Write-Safety Guardrail — Specification

**Feature**: 063-write-safety-guardrail
**Role**: Crafter
**Touchpoint**: Specification
**Inputs**: spec.md, plan.md, PROJECT.md; the plugin-hook contracts are grounded against real installed Claude plugins under the developer's `~/.claude/plugins/` (the `security-guidance` plugin's `hooks/hooks.json` with a `matcher:"Bash"` hook, the `hookify` plugin's `PreToolUse` hook, and the official `plugin-dev` plugin's `hook-development` SKILL + `plugin-settings`/`hook-development` example scripts that emit the permission-decision JSON) — **external reference examples, not files present in this repository**

> The artifact *is* the interface: a Claude plugin **hook** (a `PreToolUse` gate over the `Bash` tool) plus its gated-command registry, added to the plugin 062 already established. There is no runtime call into the CLI — the host invokes the hook before the agent's shell command runs. Protocol-level contracts are the hook registration in `hooks.json`, the hook's stdin/stdout JSON, the registry file shape, and the gating reason string.

---

## Surface

### Invocation

The guardrail exposes **no CLI-style entry point** and adds no `glassfrog` command or flag. It is invoked by the plugin host as a hook:

| Consumer | Entry point | Trigger |
|---|---|---|
| Claude Code plugin host | `plugin/hooks/hooks.json` (`PreToolUse`, `matcher:"Bash"`) | Host fires the hook **before** any `Bash` tool call the agent makes |
| The gate script | `plugin/hooks/<gate-script>` (resolved via `${CLAUDE_PLUGIN_ROOT}`) | Run by the host on each matched `Bash` call; reads the call on stdin, emits a permission decision on stdout |

There are no flags or arguments — the hook's only input is the tool-call JSON on stdin.

### Structural layout (files this feature adds)

```
plugin/
  .claude-plugin/
    plugin.json                  # EXISTS (062). 063 adds the hooks wiring — either a top-level
                                 #   "hooks": "./hooks/hooks.json" key here, or the default-path file below.
  hooks/
    hooks.json                   # hook registration (PreToolUse → Bash)            [ADDED]
    <gate-script>                # the recognizer + decision emitter (working name)  [ADDED]
    <gated-commands-registry>    # the single source of gated proposal-write leaves [ADDED]
internal/build/
    write_safety_guardrail_guard_test.go   # best-effort drift tripwire (companion, not part of the plugin) [ADDED]
```

The gate script and registry names are `[ASSUMED]` working names — reversible (nothing keys on them but `hooks.json` and the drift test, both in-repo). The registry MAY be a standalone data file or a constant inside the script, provided the drift test can read the same source (plan ADR-4 single-sourcing).

### `hooks.json` schema (grounded on installed plugins)

```json
{
  "description": "Write-safety guardrail — gate governance writes (the proposal write path) behind explicit confirmation",
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Bash",
        "hooks": [
          {
            "type": "command",
            "command": "bash \"${CLAUDE_PLUGIN_ROOT}/hooks/<gate-script>\"",
            "timeout": 10
          }
        ]
      }
    ]
  }
}
```

| Field | Value | Notes |
|---|---|---|
| event | `PreToolUse` | Fires before the tool runs, so the decision gates execution (vs `PostToolUse`, which is after the fact) |
| `matcher` | `"Bash"` | Limits the hook to shell tool calls; non-`Bash` tools never reach the gate |
| `type` | `"command"` | **Deterministic** script gate. NOT `"prompt"` (which delegates the decision to an LLM) — enforcement must not itself depend on agent judgment |
| `command` | `${CLAUDE_PLUGIN_ROOT}`-rooted | Host expands `${CLAUDE_PLUGIN_ROOT}` to the installed plugin directory |
| `timeout` | seconds (e.g. `10`) | Bounded so a hung gate can't stall the agent |

### Hook stdin / stdout contract (grounded on `plugin-dev` hook-development)

**Stdin** (the tool call, JSON):

| Field | Type | Use |
|---|---|---|
| `tool_name` | string | The gate acts only when this is `"Bash"` |
| `tool_input.command` | string | The shell command line the gate parses |
| `tool_input.description` | string | Informational; not parsed for the decision |

**Stdout** (the decision, JSON):

```json
{
  "hookSpecificOutput": { "permissionDecision": "ask" },
  "systemMessage": "Governance write: `glassfrog proposal propose prp_01…` will advance this proposal into circulation. Confirm to proceed."
}
```

| Decision | When | Effect |
|---|---|---|
| `ask` | The command is a recognized governance write (a proposal write leaf), or an unrecognized `glassfrog … proposal` subcommand (fail-closed) | Host surfaces the command + `systemMessage` to the practitioner and runs it only on explicit approval |
| `allow` (or empty output / exit 0) | Reads, `tension` edits, and any non-`glassfrog` command | Command proceeds with no prompt |

`deny` is **not** used as the steady-state gate — the guardrail asks for human confirmation, it does not categorically refuse. (`deny` remains available for a host that cannot render `ask`; see Error Communication.)

### Gated-command registry (the proposal write path — Q2=B scope)

The registry enumerates exactly the four proposal-write leaves. Concrete command forms (arg names from the shipped CLI):

| Capability | Command form | Gated |
|---|---|---|
| Proposal Creation (055) | `glassfrog proposal create <tension-id>` | **yes** |
| Advance to Circulation (057) | `glassfrog proposal propose <prp-id>` | **yes** |
| Response Recording (058) | `glassfrog proposal respond <prp-id>` | **yes** |
| Withdraw Proposal (059) | `glassfrog proposal withdraw <prp-id>` | **yes** |
| Tension Capture / Update / Discard (042/044/045) | `glassfrog tension create\|update\|discard …` | no (operational, ungated) |
| All reads (`me`, `roles`, `proposal get/list`, `search`, …) | — | no |

---

## Interactions

**Invocation-to-decision flow**:

1. The agent is about to run a shell command; the host fires the `PreToolUse` hook with the call JSON on stdin.
2. The gate reads `tool_input.command`. If `tool_name` ≠ `Bash`, or the command does not invoke `glassfrog`, it emits `allow` (or nothing) and stops.
3. It resolves the `glassfrog` invocation token — tolerating a bare name, an absolute/relative path (`…/glassfrog`), and a leading `VAR=val` env prefix — then resolves the subcommand path (`proposal <leaf>`, `tension <leaf>`, a read, …).
4. **Decision**:
   - subcommand path ∈ registry (proposal create/propose/respond/withdraw) → `ask` with a `systemMessage` naming the command, the target id, and the effect.
   - command invokes `glassfrog proposal` but the leaf is **unrecognized** → `ask` (fail-closed: a future write leaf is gated by default until the registry is updated).
   - anything else (reads, `tension` edits, non-`glassfrog`) → `allow`.
5. On `ask`, the host presents the command to the practitioner; the write runs only on explicit approval, otherwise it is not sent and the governance record is unchanged.

**Stale-write re-confirmation** — no special path in the hook: a write refused as stale (`412`, exit `7`, surfaced by 054) is followed by the agent re-reading (per 062's orientation guidance) and **retrying**; the retry is itself a proposal-write `Bash` call, so step 1–5 run again and the host re-prompts. A blind retry therefore cannot bypass the gate. The hook neither performs nor verifies the re-read (it cannot inject state) — it guarantees the retry is re-confirmed.

**Recognition is deterministic and command-based**: the gate keys on the parsed command path against the static registry, never on the agent's stated intent or the command's flags. This mirrors 060's recognizer-plus-static-registry shape, one layer up (a hook over `Bash`, not a Go error-path interceptor) — there is no shared code between them.

---

## Error Communication

Hook artifacts fail as **constraint/operational conditions**, not runtime API errors:

| Condition | Behavior |
|---|---|
| `tool_name` ≠ `Bash` / command isn't `glassfrog` | `allow` — out of the gate's scope; never blocks unrelated shell work |
| Recognized proposal-write leaf | `ask` — the human-in-the-loop checkpoint (the guardrail's purpose) |
| Unrecognized `glassfrog … proposal` subcommand | `ask` — fail-closed *within the proposal namespace*, so no future write ships ungated by omission |
| Gate script error / malformed stdin | Fail safe: the gate must not block unrelated `Bash`. It allows non-`glassfrog` work; for a command it has positively identified as a `glassfrog proposal` write but cannot fully parse, it errs to `ask`. A host that treats a nonzero hook exit as a block (the `exit 2` + stderr convention seen in `plugin-dev` examples) is acceptable for the proposal-write case but must not be reached for ordinary commands |
| Registry missing/unreadable | The drift test (below) catches this in CI; at runtime the gate treats a `glassfrog proposal` write conservatively (`ask`) rather than silently allowing |
| Drift tripwire fails (`internal/build` test red) | The CLI's `proposal` subcommand surface changed without the registry — a potential silent-ungated-write; fix the registry (or confirm the change) |
| Drift coverage reduced/omitted | Permitted (plan ADR-4, partial by design) **only if stated**, never silent — the test names what it does not cover |
| Plugin / hook not installed, or host lacks `PreToolUse` | No enforcement: writes fall back to 062's guidance-only behavior. Nothing in the CLI breaks; the gate strengthens a present host, it cannot retrofit an absent one |

HTTP request/response shapes, status-code rendering, and rate-limit handling are **N/A** — the guardrail has no runtime API surface; those belong to the CLI it gates (015/017/031/032), and the `412`/exit-`7` surfacing belongs to 054.

---

## Consistency Notes

- **Sibling interface files**: none — this feature has only a specification touchpoint. It adds no `glassfrog` subcommand, so there is no CLI accord; no API/UI/events surface.
- **Extends 062's plugin** (plan ADR-1, DECISIONS §409): `plugin/.claude-plugin/plugin.json` already exists; 063 adds `plugin/hooks/` (the extension point 062 ADR-1 reserved — "the guardrail may add `plugin/hooks/`"). 062's orientation skill is unchanged; the `412` re-read guidance stays single-sourced there (plan ADR-5).
- **Grounded on installed plugins**, cited above as **external reference examples, not repo files**: the `matcher:"Bash"` hook shape and `${CLAUDE_PLUGIN_ROOT}` rooting follow `security-guidance`/`hookify`; the `permissionDecision` (`allow|deny|ask`) + `systemMessage` output and the `type:"command"` vs `type:"prompt"` distinction follow `plugin-dev`'s `hook-development` SKILL and example scripts.
- **`type:"command"`, not `type:"prompt"`**: a deterministic script gate is chosen over an LLM-judged prompt hook — the enforcement must not depend on the same agent judgment the guardrail exists to backstop.
- **Relation to 060's gate registry**: both are "recognizer + static registry," but 060 keys on HTTP method+path inside the Go error path (a `403` plan-gate), while this keys on the `Bash` command path inside a plugin hook (a pre-write gate). Different layers, different inputs, no shared code — noted so neither is mistaken for the other.
- **Host hook contract is version-specific** (plan R2): the registration schema and the permission-decision protocol track the Claude Code plugin host's current conventions; treat as an external contract to revisit if the host changes. The exact reason-field spelling (`systemMessage` vs a `permissionDecisionReason`) should be confirmed against the host version at implementation.
- **Distribution deferred to #70**: getting the plugin (and this hook) installed onto an agent's host is Operating-Surface Packaging; this accord assumes a locally-present plugin.
