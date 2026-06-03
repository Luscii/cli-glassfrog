# Interface Accord: Exit-Code Convention — CLI

**Feature**: 004-exit-code-convention
**Role**: Crafter
**Touchpoint**: CLI
**Plan reference**: System Architecture (the process-exit boundary); ADR-1 (pure category→code registry), ADR-2 (published 0–6 constants), ADR-3 (RuntimeError → 1), ADR-4 (panic-recover → 1).

---

## Surface

This accord defines the **process exit code** a caller observes in `$?` after any `glassfrog` invocation. The exit code is the only *outcome* signal this capability emits — it produces no error or outcome message of its own (error text belongs to cobra and the resolved command). The one exception is the panic safety net, which writes a crash diagnostic to stderr so an unrecovered crash stays diagnosable (see Error Communication).

The published, frozen convention — the canonical category↔code registry:

| Code | Category | Emitted when | Producer (today) |
|---|---|---|---|
| `0` | Success | A command completed its work, or a group/root/`--version`/help resolved to a help/listing outcome. | Argument Dispatch (002) |
| `1` | Internal error (safety net) | A resolved command's own action failed, or the CLI terminated for any reason matching no more-specific category (incl. an unexpected internal failure / panic). | Argument Dispatch (`RuntimeError`); panic-recover |
| `2` | Usage error | Unknown command, or an unknown/missing flag or unexpected positional argument — nothing ran. | Argument Dispatch (002) |
| `3` | API error | The API returned an error not covered by a more specific category. | *reserved — future API client* |
| `4` | Permission / authorization error | The API rejected the caller's auth or membership (incl. premium-gated rejection). | *reserved — future API client* |
| `5` | Rate-limited | The API reported the rate limit was exceeded. | *reserved — future API client* |
| `6` | Network-unavailable | The API could not be reached at all (connection, DNS, or timeout). | *reserved — future API client* |

Codes `3`–`6` are **published now but not yet produced** — no API client exists in the skeleton. They are reserved so an agent can rely on the full convention today; their producer (the future API client) classifies into these categories, and the registry maps them. A single invocation always exits with **exactly one** code.

---

## Interactions

- **Reading the result**: the caller (a CI runner or an AI agent) inspects `$?` (the process exit status) immediately after invocation. No parsing of stdout/stderr is required to determine the outcome class.
- **Producer-classifies model**: the producer of an outcome labels its category; this capability maps category→code and never re-derives the category from an error or an HTTP status. Argument Dispatch supplies `Success`/`UsageError`/`RuntimeError`; the future API client supplies the operational categories.
- **Most-specific category wins**: when more than one category could apply, the producer selects the most specific one before the mapping — a rate-limited outcome exits `5`, not `3`.
- **Scripting / branching example** (illustrative, not a mandated UX):
  ```
  glassfrog roles list
  case $? in
    0) : ;;          # success
    2) : ;;          # fix the invocation
    5) sleep … ;;    # rate-limited — back off
    4) : ;;          # permission — escalate
  esac
  ```
- **Extension**: a new outcome category is added at the single registry site, taking a new previously-unused code; existing codes are **never renumbered**, so a consumer's existing branch keeps its meaning across releases.

---

## Error Communication

- **This capability renders no text.** It contributes only the numeric code. The human/machine-readable error message is written by cobra (usage errors, a resolved command's `RunE` error) or by Argument Dispatch (the synthesized nested-unknown-subcommand message) — see 002's accord.
- **Never zero on failure**: any non-success termination exits non-zero. An unmapped or future category falls through to the safety-net code `1`, never `0`.
- **Panic safety net**: an unrecovered internal failure (panic) is mapped to `1`, not Go's default status `2` (which would collide with the usage code). The recover path still writes the panic value/stack to stderr so the crash stays diagnosable (Action Transparency).
- **No shell-reserved codes**: codes `126`, `127`, and `128 + N` (signal range) are never assigned to a CLI category, so `$?` is never ambiguous against shell/signal semantics.

---

## Consistency Notes

- **Argument Dispatch (`002/interface-cli.md`)**: that accord explicitly deferred "maps the `Success`/`UsageError` category to a process exit code, and is where a distinct runtime-failure category (`RuntimeError`) is introduced." This accord fulfils both: `Success`→`0`, `UsageError`→`2`, and the new `RuntimeError`→`1`.
- **Help & Version (003)**: help/listing/`--version` are `Success` outcomes → `0`. 003 owns the rendered text; this accord owns only the accompanying code.
- **Future API client (the producer of `3`–`6`)**: classifies HTTP/network results into the operational categories and references this registry for the code; it must not invent codes outside the published set, and adds any new category at the single registry site.
- **PROJECT.md / CONSTITUTION**: distinct codes per failure class realise Action Transparency II (the operator can always tell what happened from `$?`); the never-zero-on-failure and panic→`1` rules realise Fail Safe III (a failure is never reported as success).
- **stderr assumption** (inherited from 002): error text goes to stderr, command output to stdout; the exit code is orthogonal to both. Not yet fixed in PROJECT.md.
