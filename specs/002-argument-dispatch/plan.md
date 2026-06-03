# Plan: Argument Dispatch

**Feature**: 002-argument-dispatch
**Role**: Shaper
**Inputs**: spec.md (002-argument-dispatch), PROJECT.md, `.score/memory/DECISIONS.md` (4 entries — Go, cobra, guard, explicit wiring), `.score/memory/LEARNINGS.md` (cobra built-ins finding). No SOUL.md.

---

## System Architecture

Argument Dispatch is realized on the cobra command tree that Command Registration assembles (DECISIONS.md precedent: cobra's command tree *is* the registry). Dispatch is cobra's resolution + execution over that tree, wrapped by a thin **classification layer** that labels each invocation's outcome so Exit-Code Convention (004) can later map it to a process code without re-parsing — exactly the "dispatch classifies, Exit-Code encodes" split the spec draws.

The parts:

- **Dispatch entry** — a single function (`Run`) that takes the assembled root command and the invocation arguments, runs cobra's resolution/execution, and returns a **categorized outcome**. It replaces the bare `Assemble().Execute()` call currently in `main.go`.
- **Outcome category** — a small, code-free classification (`Success`, `UsageError`) derived from how the invocation resolved. Unknown command, unknown flag, or bad arguments → `UsageError`; a clean run (including a group/root resolving to help) → `Success`. A resolved command whose own action returns an error is surfaced via the returned error and left uncategorized for now — giving runtime failures a distinct category (a future `RuntimeError`) is **deferred to Exit-Code Convention (004)**, the capability that needs the distinction. This is the spec's "classify the outcome" made concrete (the spec names success and usage error); it carries no exit codes.
- **Relied-on cobra defaults** — exact matching (cobra's `EnablePrefixMatching` is left at its default of `false`), unknown-flag rejection (cobra's default), and best-effort "did you mean" suggestions (cobra's default) are framework behaviors dispatch relies on rather than re-implements.

```
main
 └─ cli.Assemble()            (001: builds the cobra tree via the guard)
 └─ cli.Run(root, args)       (002: this feature)
       ├─ cobra resolve+execute (exact match, reject unknown flags, suggest)
       └─ classify → Outcome{Success | UsageError}, err
            ▲
            │ today: main maps minimally (0 / non-zero)
            │ 004: Exit-Code Convention maps category → process code
```

---

## Architecture Decisions

### ADR-1: Route via cobra's built-in resolution, not a hand-rolled matcher

**Context**: The spec requires exact (non-prefix) matching, an unknown-command usage error with a best-effort suggestion, bare-group/root → help, and rejection of unexpected flags. Command Registration already builds a cobra tree (DECISIONS.md). cobra's `Execute` provides all of these resolution behaviors.

**Options considered**:
1. **Rely on cobra's resolution** — `Execute` over the assembled tree. Exact match, unknown-command errors + suggestions, unknown-flag rejection, and group/root help are cobra defaults. Minimal new code; the behavior must be verified to match the spec.
2. **Hand-roll a matcher** — walk the tree and resolve tokens ourselves. Full control, but re-implements well-trodden resolution and duplicates what cobra already does correctly, for no benefit.

**Decision**: Option 1 — rely on cobra's resolution. It already delivers the spec's matching, unknown-command, suggestion, and unknown-flag behaviors.

In practice: dispatch calls cobra over the root from `Assemble`. **`EnablePrefixMatching` (a cobra package-global) is left `false`** — this is what makes matching exact; turning it on would violate the exact-match non-behavior. Unknown-flag rejection and suggestions are left at their defaults.

**Consequences**: The spec's resolution behaviors come essentially free and stay consistent with 001's framework choice. The exact-match guarantee depends on a package-global staying `false` — a regression test must assert a prefix does **not** resolve. cobra auto-adds `help` and `completion` commands (see LEARNINGS.md); they are resolvable like any command — whether to keep or hide them is Help & Version's decision (003), not dispatch's. *Precedent-setting: later command-bearing specs dispatch through this same path.*

### ADR-2: Dispatch classifies each outcome into a code-free category

**Context**: The spec has dispatch *classify* an outcome as success or usage error, while Exit-Code Convention (004) owns the actual process codes. 004 does not exist yet, so dispatch must produce a classification with no consumer in place.

**Options considered**:
1. **Dispatch returns a typed outcome category** (`Success`/`UsageError`) — code-free; the entrypoint maps it minimally today, and Exit-Code Convention consumes it later.
2. **Defer all classification to Exit-Code Convention** — dispatch returns only cobra's raw error. But dispatch is the only layer that knows *why* a run failed (resolution/usage vs the command's own error); pushing that to 004 would force it to re-derive the category from an untyped error.
3. **Map straight to exit codes here** — violates the spec's non-behavior (dispatch must not emit codes).

**Decision**: Option 1 — dispatch returns a code-free outcome category. Dispatch is the right owner because it alone distinguishes a resolution/usage failure from a resolved command that ran. The category is deliberately minimal — **two values now** (`Success`/`UsageError`), matching what the spec names; a third (`RuntimeError`, for a resolved command's own failure) is **deferred to 004**, the capability that needs to tell runtime failures apart. 004 can extend the category without churn.

**Consequences**: 004 gets a clean, code-free input. Until 004 lands, the entrypoint maps the category minimally (success → 0, any error → non-zero) — a documented placeholder, not the final convention. Detecting `UsageError` from cobra requires care (cobra does not expose a typed "command not found"): a flag-error hook marks flag failures and unknown-command is detected at resolution. A resolved command's own error is returned as-is for the caller to surface; categorizing it distinctly is deferred to 004. *Precedent-setting: the outcome category is the contract Exit-Code Convention builds on.*

---

## Outcome Classification

The category is derived, not configured:

| Situation | Category |
|---|---|
| Resolved command's action runs and returns no error | `Success` |
| Group or root resolved with no subcommand (help/listing outcome) | `Success` |
| Token does not resolve to a registered command | `UsageError` |
| Unknown flag, or argument the resolved command rejects | `UsageError` |

> A resolved command whose own action errors is **not** given a distinct category yet — the error is returned for the caller to surface, and distinguishing it (a future `RuntimeError`) is deferred to Exit-Code Convention (004).

The category names *what kind* of outcome occurred; it deliberately says nothing about exit codes or messages (those are Exit-Code Convention's and Help & Version's).

---

## Cross-cutting Concerns

**Error handling**: dispatch never swallows. Usage errors surface cobra's message (unknown command/flag) plus the help pointer; the category travels with the returned error so the caller can act on it. A resolved command's own error passes through via the returned error; giving it a distinct category is deferred to 004.

**Testing strategy** (Constitution IV): dispatch is exercised by calling `Run` with argument slices against an assembled tree and asserting `(category, output)`. Each category and the exact-match / bare-group / unknown-command / unexpected-flag behaviors get coverage; the spec's driving scenarios become the BDD outer loop.

**Configuration**: none — matching strictness and flag handling are cobra defaults, fixed at build time.

**No consumer yet**: the outcome category has no caller until Exit-Code Convention (004). The entrypoint maps it minimally for now; this is called out so 004 replaces the placeholder rather than discovering it.

---

## Implementation Strategy

Three phases, linear.

- **Phase 1 — Dispatch entry + outcome category**: add `cli.Run(root, args) (Outcome, error)` that executes the assembled tree via cobra and returns a category; rewire `main.go` to use it (replacing bare `Assemble().Execute()`). Confirm prefix matching is off and unknown-flag rejection is on. *Depends on: nothing (001 code is in place).*
- **Phase 2 — Outcome classification**: implement the success / usage-error derivation (flag-error hook + unknown-command detection) with RED-first unit tests per category. A resolved command's own error is returned uncategorized (RuntimeError deferred to 004). *Depends on: Phase 1.*
- **Phase 3 — Executable acceptance**: godog step definitions for the 002 driving scenarios (routing, bare group, unknown command, exact-match-no-prefix, unexpected-flag rejection, empty invocation), turning them into executable acceptance. *Depends on: Phase 2.*

---

## Risks

- **Classifying usage errors from cobra's untyped errors** (medium likelihood, medium impact): cobra does not expose a typed "command not found", so detecting `UsageError` (unknown command / unknown flag / bad arg) relies on a flag-error hook and resolution-time detection. Mitigation: detect at the seams (resolve before/inside Execute; `SetFlagErrorFunc` sentinel) and pin with tests. A resolved command's own failure is left to the returned error; its distinct categorization is deferred to 004.
- **Exact-match depends on a package-global** (low likelihood, medium impact): cobra's `EnablePrefixMatching` is process-global; if any future code sets it `true`, the exact-match non-behavior breaks silently. Mitigation: a regression test asserting a prefix (`ro`) does not resolve to `roles`.
- **cobra built-ins are resolvable** (low likelihood, low impact): `glassfrog help` / `completion` resolve as commands, which interacts with the unknown-command contract. Mitigation: defer the keep/hide decision to Help & Version (003); note it, don't pre-empt.
- **Classification has no consumer yet** (low likelihood, low impact): building the category before Exit-Code Convention risks the wrong shape. Mitigation: keep it to three code-free values; 004 adapts.

---

## What This Plan Does Not Cover

- **Help/usage text rendering** — dispatch routes group/root/unknown to a help outcome; **Help & Version** produces the text.
- **Exit code mapping** — dispatch produces a category; **Exit-Code Convention** maps it to a process code. The entrypoint's current minimal mapping is a placeholder.
- **Per-command flags and arguments** — each command defines and validates its own flags; dispatch only routes and classifies. The protocol-level CLI surface (the dispatch contract) is the **interface** skill's concern; task decomposition is **tasks**'.
- **Keep/hide of cobra's built-in `help`/`completion`** — deferred to Help & Version (003).
