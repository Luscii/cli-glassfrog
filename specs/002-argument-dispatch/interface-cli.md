# Interface Accord: Argument Dispatch — CLI

**Feature**: 002-argument-dispatch
**Role**: Crafter
**Touchpoint**: CLI
**Plan reference**: System Architecture (dispatch entry + relied-on cobra defaults); ADR-1 (cobra resolution, exact match).

---

## Surface

This accord defines what the caller observes when they invoke `glassfrog <tokens>` — the resolution and error contract over the command set that Command Registration built. (That registered paths *exist* is 001's accord; how a typed invocation *resolves against them* is this one.)

| Invocation shape | Outcome |
|---|---|
| Exact path to a leaf (`glassfrog roles list`) | Routes to the leaf; its action runs. Category: success. |
| Exact path to a group, no further token (`glassfrog roles`) | Resolves to the group; routes to a help/listing outcome. Category: success. |
| No tokens (`glassfrog`) | Resolves to root; routes to a help/listing outcome. Category: success. |
| First token matches no registered command at its level (`glassfrog rolez`) | Unknown-command **usage error** (see Error Communication). |
| Unknown flag or unexpected positional arg (`glassfrog roles list --bogus`) | **Usage error**; the command does not run. |

Matching is **exact** at every level — no prefix or abbreviation resolves to a longer command (`glassfrog ro list` is an unknown command, not `roles list`).

---

## Interactions

- **Resolution walk**: dispatch matches tokens left to right, descending into groups by exact name. The deepest exact match wins; the first token that matches nothing at its current level is the point of failure.
- **Group vs leaf**: resolving to a group (or root) with no runnable action routes to the help/listing outcome rather than erroring — consistent with 001's bare-group-resolves-to-the-group behavior.
- **Best-effort suggestion**: on an unknown command, if a close registered name exists, the error may include a "did you mean …" hint. The hint is best-effort — present when a near match exists, absent otherwise; callers must not depend on it.

---

## Error Communication

- **Unknown command**: a message to standard error naming the unrecognized token and pointing to help (e.g. `unknown command "rolez" for "glassfrog"` + a `Run 'glassfrog --help' …` pointer), optionally with a "did you mean" suggestion. Outcome category: **usage error**.
- **Unknown flag / unexpected argument**: a message to standard error naming the offending flag/argument; the resolved command does **not** run. Outcome category: **usage error**.
- **Never silently ignored**: unexpected input always produces a usage error — it is never tolerated (resolves the spec's clarified decision).
- **Exit codes are out of scope**: this accord specifies the *outcome category* (success vs usage error), not the process exit code. Mapping the category to a code is **Exit-Code Convention**'s contract.

---

## Consistency Notes

- **001 (`interface-cli.md`)**: that accord guarantees registered paths are reachable and bare groups self-resolve; this accord defines how a typed invocation is matched against them and what an unmatched token does — the contract 001 explicitly deferred to dispatch.
- **Sibling (`interface-spec.md`, this spec)**: the outcome category surfaced here is produced by the `Run` entry defined there.
- **Help & Version (future, 003)**: renders the help/usage text this accord routes to; also owns whether cobra's built-in `help`/`completion` commands are kept or hidden.
- **Exit-Code Convention (future, 004)**: maps the success / usage-error category to a process exit code, and is where a distinct runtime-failure category (`RuntimeError`) is introduced when needed (deferred from this spec).
- **Assumption**: error messages go to standard error (stderr), separate from command output on stdout — conventional for CLIs and assumed here; not yet fixed in PROJECT.md.
