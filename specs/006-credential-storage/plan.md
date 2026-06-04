# Plan: Credential Storage

**Feature**: 006-credential-storage
**Role**: Shaper
**Inputs**: spec.md (006-credential-storage), PROJECT.md, `.score/memory/DECISIONS.md` (read at 13 entries — Go self-contained, cobra registry, guard registration, explicit wiring, cobra resolution, outcome-category split, cobra help, built-ins hide, exit-code registry, RuntimeError, panic-recover, plus the three 005 credential precedents: `internal/auth` + shared read/write format module, `.glassfrogrc`/`GLASSFROG_TOKEN` `[ASSUMED]` contract, injected roots for hermetic tests), `.score/memory/LEARNINGS.md` (5 cobra/BDD findings — the shared-feature-file step-vocabulary rule applies; the cobra built-ins and Args-validator findings apply to the new command). Sibling 005 plan.md + interface-spec.md read for the shared file contract. No SOUL.md. 005 is at stage *Analyzed* — its `internal/auth` package is designed but not yet implemented in code.

---

## System Architecture

Credential Storage is the **write** half of Token Authentication and the first capability that *writes* to the filesystem and the first credential capability with a **command surface**. Where Discovery (005) is an internal resolver with no shell entry point, Storage is invoked by the operator ("store my token once") — so it registers a command in the cobra tree (guard-registered, explicitly wired, per 001 precedent) and that command calls a writer in `internal/auth`.

The parts split cleanly across the two existing layers:

- **`internal/auth` (writer side)** — Storage contributes the **writer** to the package Discovery's plan created. The writer persists a token into a `.glassfrogrc` file in exactly the structure Discovery reads, and **shares the credentials-file format module** with Discovery's reader so the two cannot drift (the 005 precedent: read and write share one format). The writer's contract is: given a target path and a token, validate any existing file (via the shared reader), replace only the `token` entry while preserving every other line, and write the result atomically with owner-only permissions.
- **`internal/cli` (command side)** — Storage contributes a **leaf command** (a `login`/`store`-style verb; exact name + flags are the interface skill's concern). The command owns everything CLI-shaped: resolving the token from the precedence chain (argument → piped stdin → `GLASSFROG_TOKEN` → interactive prompt), detecting interactivity (a TTY on stdin), selecting the target path (home by default, current directory on a flag), enforcing the existing-token guard (non-interactive needs an explicit overwrite signal; interactive confirms), and classifying a **code-free outcome** that Exit-Code Convention (004) maps. The token value never appears in output, prompts echo nothing, and errors name only paths.

```
glassfrog <store-verb>                         (internal/cli — new guard-registered leaf, wired in main)
 │  1. resolve token: arg → stdin(piped) → GLASSFROG_TOKEN → prompt(if TTY)   (input seam)
 │     · non-interactive + no token anywhere     → "no token to store" error (code-free outcome)
 │  2. reject blank/whitespace token (usable-token rule, shared with 005)
 │  3. choose target path: homeDir/.glassfrogrc (default) | CWD/.glassfrogrc (--flag)   (injected roots)
 │  4. existing token present?
 │     · interactive   → confirm; offer merge + location (cwd / home / both)
 │     · non-interactive → error UNLESS explicit overwrite signal; then merge target only
 │  5. call internal/auth writer ──────────────┐
 │  6. report written path (never the token)    │
 ▼                                              ▼
internal/auth  (shared with 005/007)      writeCredentials(path, token) → error
 ├─ readCredentialsFile(path) (005)         a. parse-validate existing file via shared reader
 │     (shared reader — the round-trip          (malformed, non-comment line w/o '=') → format error, NO write
 │      target; the format module both      b. line-preserving rewrite: replace token= value (append if absent),
 │      sides depend on)                         keep every other line + comments verbatim
 └─ writeCredentials(path, token) (006)     c. atomic write: temp file in same dir, chmod 0600, rename over target
```

---

## Architecture Decisions

### ADR-1: Storage's writer joins `internal/auth` and round-trips with Discovery's shared format module

**Context**: 005's plan (DECISIONS precedent) established `internal/auth` as the home for token concerns and reserved the *writer* for this spec, with the rule that the credentials-file read and write sides share one format module so they cannot drift. 005's interface-spec pins the exact `.glassfrogrc` structural contract. CONSTITUTION V (Composition) wants modular parts; the round-trip property is the contract that keeps Discovery able to read what Storage writes.

**Options considered**:
1. **Add `writeCredentials` to `internal/auth`, reusing the shared reader for parse-validation and round-trip tests.** The writer and reader live together; a test writes then reads back through Discovery's resolver. Single source of truth for the file shape.
2. **A separate writer package that re-implements the format.** Rejected: two implementations of one file shape is exactly the drift 005 forbade — a format tweak on one side silently breaks the other.

**Decision**: Option 1 — the writer is a function in `internal/auth` beside the reader, and the shared format understanding (the `key=value`, `#`-comment, `token` line rules from 005's interface-spec) is the one both sides obey. A round-trip test (write a token, resolve it back via the 005 reader) is the binding contract.

**Consequences**: Conforms to the active 005 precedent (followed, not diverged). Because 005 is not yet implemented, whichever of 005/006 lands first creates the shared module; the round-trip test pins the contract regardless of order (see Risks). The `.glassfrogrc` name and `GLASSFROG_TOKEN` remain `[ASSUMED]` — 006 conforms to 005's provisional contract rather than re-opening it; if either name changes, it changes once, in the shared `internal/auth` constants.

### ADR-2: Storage is the first credential command — a guard-registered cobra leaf, wired explicitly, calling the `internal/auth` writer

**Context**: Unlike Discovery (no command surface), Storage is operator-invoked. 001's precedents require commands to register through the fail-loud guard, be wired explicitly in `main` (no `init()` side effects), and each declare an `Args` validator (LEARNINGS: a leaf without one silently accepts stray positionals). The token may be supplied as a positional argument.

**Options considered**:
1. **A guard-registered leaf in `internal/cli` that gathers input and delegates the file write to `internal/auth`.** Keeps the command/transport concern in the command layer (where dispatch, help, exit-code already live) and the file concern in `auth` — mirrors how 007 will keep request-attachment with the API client, not in `auth`.
2. **Put the whole command, input-gathering and file write, inside `internal/auth`.** Rejected: drags cobra and stdin/TTY concerns into the filesystem package and couples the command tree to `auth`; 005 deliberately kept `auth` command-free.

**Decision**: Option 1 — a new leaf command in `internal/cli`, registered via `MustRegister` and wired in `main`, declaring an `Args` validator that matches its optional single token positional (`cobra.MaximumNArgs(1)`). It calls `internal/auth`'s writer. The exact command name, group placement, and flag names (location flag, overwrite signal) are **deferred to the interface skill** — this ADR fixes only that there *is* one guard-registered leaf delegating to the writer.

**Consequences**: Storage gets standard cobra help for free (003 precedent) once it sets a non-empty `Short`. The new leaf must carry an `Args` validator and the BDD test fixtures must mirror it (LEARNINGS: production/test parity). The command's outcome is code-free (ADR-5); `main`/004 map it. Interface pins the precise invocation surface.

### ADR-3: Token-source precedence and interactivity resolve behind an injected input seam

**Context**: The spec fixes precedence (argument → piped stdin → `GLASSFROG_TOKEN` → prompt) and defines interactivity as a TTY on stdin, with the prompt non-echoing. CONSTITUTION IV requires test-first; reading real stdin/TTY/`os.Args`/home inside the command would make tests non-hermetic and — given the secret — risk writing the developer's real `~/.glassfrogrc`. This extends 005's "inject the roots" precedent to the input side.

**Options considered**:
1. **Inject a small input context** — the candidate token sources (positional arg, a stdin reader, an `isTTY` flag, an env lookup) plus `startDir`/`homeDir` — into a pure resolution function; a thin production seam binds the real `os` values. Hermetic table-driven tests for every precedence rung and the TTY/no-TTY branches.
2. **Read `os.Stdin`, `term.IsTerminal`, `os.Getenv`, `os.UserHomeDir` directly in the command.** Rejected: forces tests to manipulate real process state and the real home directory — the exact non-hermetic trap 005 ADR-5 avoided.

**Decision**: Option 1 — a pure `resolveToken(sources)` returning the chosen token and its origin (or a "no token" result), and pure target-path selection from injected `startDir`/`homeDir`; the production seam supplies real stdin/TTY/env/dirs. Interactivity = the injected `isTTY` flag (production: a TTY check on stdin).

**Consequences**: Every precedence and interactivity branch is unit-tested without touching the terminal or the real home (a safety property given the secret). The prompt path (non-echoing read) is the one piece exercised only through the production seam + BDD; the resolution logic around it is pure. Detecting "piped stdin present" vs "interactive" reuses the same `isTTY` signal: stdin is a pipe → read it; stdin is a TTY and nothing else supplied → prompt.

### ADR-4: Merge is a line-preserving rewrite gated by parse-validation, written atomically with owner-only permissions

**Context**: The spec requires: replace only the `token` entry and leave every other key unchanged; if an existing file can't be parsed, report a format error rather than overwrite it; on any write error leave the filesystem unchanged; and (assumption) write a secret file with owner-only permissions.

**Options considered**:
1. **Parse-validate then line-preserving edit + atomic write.** Read the existing file through the shared reader (which errors on a malformed non-comment line without `=`) — a parse failure aborts with a format error and no write. On success, rewrite at the line level: replace the value on the existing `token=` line (or append a `token=` line if absent), preserving every other line including comments and ordering. Serialize to a temp file in the same directory, `chmod 0600`, and `rename` over the target (atomic; a mid-write failure leaves the original intact).
2. **Parse the whole file into a map, mutate `token`, re-serialize.** Rejected: drops comments and line ordering — a weaker reading of "leaves every other key unchanged" and a surprising rewrite of a hand-edited file.
3. **Truncate-and-write the target in place.** Rejected: a crash mid-write corrupts the credentials file, and it can't honor "filesystem unchanged on error."

**Decision**: Option 1. The shared reader is the parse gatekeeper (format error → no write); the rewrite is line-preserving; the write is atomic (temp + `rename`) with `0600` permissions `[ASSUMED]`. When the target file does not exist, the same path creates it (no existing lines to preserve) with `0600`.

**Consequences**: Comments and unrelated keys survive a re-store; a malformed file fails loud instead of being clobbered (CONSTITUTION III); a failed write never leaves a half-written credential. `0600` is `[ASSUMED]` pending confirmation and is platform-conditional (POSIX permission bits; on platforms without them the temp+rename still applies, permissions best-effort) — flagged in Risks. The atomic-rename temp file must be created in the **same directory** as the target so `rename` stays on one filesystem.

### ADR-5: Storage returns a code-free outcome; existing-token-without-override and no-token-non-interactive are reported errors; no exit codes here

**Context**: The spec's non-behavior forbids Storage from deciding the process exit code (004 owns that). It must distinguish: stored successfully, "no token to store" (non-interactive, nothing supplied), existing-token-without-an-overwrite-signal (non-interactive), a write error, and a format error. This mirrors 002/005's "the producer classifies a code-free category; the consumer/004 maps it."

**Options considered**:
1. **Return a code-free result (stored, with the written path) or a typed/categorized error; the command arm classifies into the existing `Outcome` enum and `main`/004 map to a code.** No exit code emitted in `auth` or in the command. "No token to store" and "existing token, no override" classify as `UsageError` (operator can fix the invocation); write/format failures classify as `RuntimeError` (internal/operational), reusing 004's existing categories.
2. **`os.Exit` inside the command on each outcome.** Rejected: duplicates 004's mapping, violates the spec non-behavior, and breaks the single-exit-site convention (004 precedent: codes flow through one mapper).

**Decision**: Option 1 — code-free outcomes, classified into the existing `Outcome` categories, mapped by the established `ExitCode(Outcome)` site. No new exit codes are introduced (the frozen 0/1/2/… convention already covers usage and internal-error). Secret hygiene is part of this contract: the success message reports the written path only; the no-echo prompt, stdin read, and every error string omit the token.

**Consequences**: Storage slots into the same outcome→code pipeline as dispatch, with no renumbering and no new producer category needed. The operator gets a finely-distinguished exit signal (usage vs internal) for free. A future `--token`-style flag or finer categories would extend at the established sites, not here.

---

## Integration Design

- **Credentials file format (specification boundary, shared with Discovery 005)**: `.glassfrogrc`, `key=value` lines, `token` key, `#`-comment and blank tolerant — exactly 005's pinned structural contract. Storage **writes** it; Discovery **reads** it; one shared format module. The round-trip test (write→resolve) is the contract guard. `[ASSUMED]` name pending the 005/006 reconciliation, now jointly held.
- **Standard input / terminal (input)**: a piped token is read from stdin; when stdin is a TTY and no other source supplied a token, a non-echoing prompt is shown. Interactivity is the TTY signal.
- **Environment (input)**: `GLASSFROG_TOKEN`, read as the third precedence source to persist. (Same constant Discovery reads; note it is *also* Discovery's runtime override — persisting it is an explicit operator action, not automatic.)
- **Filesystem (output)**: writes/creates `.glassfrogrc` in the home directory or the current working directory; atomic temp+rename; owner-only permissions; surfaces unwritable targets (write error) and unparseable existing files (format error) without partial writes.
- **Exit-Code Convention (004, downstream)**: receives the command's code-free `Outcome` category and maps it; Storage emits no code.

---

## Security Design

- **Secret never surfaces**: the token is never printed, logged, echoed at the prompt, or placed in any error or diagnostic. The success line names the written path; errors name paths only. A validation scenario asserts the token never appears in produced output (mirrors 005).
- **At-rest permissions**: a newly written credentials file is `0600` (owner read/write) `[ASSUMED]`; the temp file is created with the same restrictive mode *before* any token bytes are written, so the secret is never briefly world-readable. (POSIX permission bits; best-effort on platforms lacking them.)
- **Hermetic tests never write the real home**: per ADR-3, injected `homeDir`/`startDir` keep tests in temp directory trees; no test reads or writes the developer's actual `~/.glassfrogrc`.

---

## Cross-cutting Concerns

**Error handling (CONSTITUTION III — Fail Safe, Not Silent)**: an unwritable target or an unparseable existing file is a loud, typed error naming the path — never a silent overwrite, never a partial write (atomic rename guarantees all-or-nothing). "No token to store" (non-interactive, nothing supplied) and "existing token without an overwrite signal" are reported errors, not silent successes or no-ops.

**Testing strategy (CONSTITUTION IV — RED first)**: unit tests for (a) the writer — new-file create, merge-preserves-other-keys-and-comments, malformed-existing→format-error-no-write, atomic-failure-leaves-original, `0600` perms, and the **round-trip** with the 005 reader; (b) the pure token-source resolution — each precedence rung, blank-token rejection, non-interactive-no-token; (c) target-path selection — home default vs CWD flag with injected roots. The spec's driving scenarios become the godog outer loop over temp dirs with controlled stdin/TTY/`GLASSFROG_TOKEN`.

**BDD feature file (LEARNINGS — shared-file step vocabulary)**: 006's executable scenarios live in `features/unauthenticated-access/credential-storage.feature`, a solution-named file under the Token Authentication problem subdirectory (alongside 005's `credential-discovery.feature`, which the scenarios skill split out of the former flat `unauthenticated-access.feature`). Before adding steps, grep the existing `sc.Step(` registrations for that package and reuse exact step phrasing for assertions that already exist (e.g. file-state and source-precedence steps from 005); keep wording behavioral and tech-agnostic; reserve new bindings for genuinely new behavior (the write/merge/prompt assertions).

**Configuration**: the `.glassfrogrc` file name and `GLASSFROG_TOKEN` are the centralized `internal/auth` constants (005), reused — not re-declared. The `0600` mode and the overwrite-signal/location flag names are pinned by interface.

**Command surface conventions (001/003 precedents)**: the new leaf registers through the guard, is wired explicitly in `main`, declares an `Args` validator, and sets a non-empty `Short` for standard help. No package-global cobra toggles change.

---

## Implementation Strategy

Three phases, linear.

- **Phase 1 — Writer in `internal/auth`**: `writeCredentials(path, token) → error` — parse-validate the existing file via the shared reader (format error → no write), line-preserving `token=` replace/append, atomic temp+rename at `0600`. If 005's shared reader/format module does not yet exist in code, create it here to 005's interface-spec contract. RED-first unit tests including the **round-trip** against the reader, preserve-other-keys-and-comments, malformed→error, and the perms/atomicity properties. *Depends on: the shared format module (create if absent).*
- **Phase 2 — Token-source resolution, interactivity, and target-path selection**: pure `resolveToken(sources)` (arg → stdin → env → prompt precedence, blank rejection, non-interactive-no-token result) and pure target-path selection from injected `startDir`/`homeDir` (home default / CWD flag); plus the existing-token guard (non-interactive error-unless-override; the interactive confirm/location choice). The production seam binds real stdin/TTY/env/dirs and the non-echoing prompt. RED-first unit tests per branch. *Depends on: Phase 1 (writer is the sink).* 
- **Phase 3 — Command wiring + executable acceptance**: register the guard-validated leaf (with its `Args` validator and `Short`), wire it in `main`, classify outcomes into the existing `Outcome` categories. godog step definitions for the 006 driving scenarios (arg→home, stdin→CWD, env-persist, unwritable→write error, malformed→format error, blank-token, non-interactive-no-token, existing-token-no-override) in `features/unauthenticated-access/credential-storage.feature`, reusing existing step vocabulary. *Depends on: Phases 1–2.*

---

## Risks

- **Shared format module not yet implemented (005 at *Analyzed*)** (high likelihood, medium impact): 006's writer depends on a module 005 designed but hasn't built. If 006 lands first it must create the shared reader+constants to 005's interface-spec; if 005 lands first, 006 reuses them. Mitigation: treat the module as a Phase-1 deliverable conditional on absence, and make the round-trip test the contract that pins the shape either way — so whichever order, read and write agree.
- **Non-atomic or non-restrictive write leaks/corrupts the secret** (low likelihood, high impact): an in-place truncate that crashes mid-write, or a file briefly created world-readable, would corrupt or leak credentials. Mitigation: ADR-4's temp-in-same-dir + `chmod 0600`-before-write + `rename`; tests assert original-preserved-on-failure and `0600`.
- **Non-hermetic tests touch the real `~/.glassfrogrc`** (medium likelihood, high impact): a test that writes to the real home could clobber the developer's credentials or leak a token. Mitigation: ADR-3 injected roots; no test uses real `os` dirs; the production seam is exercised only over temp dirs in BDD.
- **Interactivity detection misfires in CI/pipes** (medium likelihood, medium impact): treating a piped or CI stdin as interactive would hang automation on a prompt; treating a real terminal as non-interactive would wrongly error "no token to store". Mitigation: a single injected `isTTY` signal drives both the prompt decision and the non-interactive error path; unit tests cover TTY/no-TTY × token-present/absent.
- **Existing-token override semantics under-specified for "both"** (low likelihood, low impact): the interactive "write to working / home / both" choice and the non-interactive single-target rule are pinned, but the exact override flag and confirm wording are interface-level. Mitigation: defer the precise surface to interface; the plan fixes the behavior (non-interactive: error unless override, then target-only), interface fixes the tokens.

---

## What This Plan Does Not Cover

- **The exact command name, group placement, flag names, and prompt/confirm wording** — the interface skill pins the invocation surface; this plan fixes only that a guard-registered leaf delegates to the writer.
- **Reading/resolving a token for use** — Discovery (005) owns resolution; Storage only writes. They share the format module.
- **Attaching the token to requests** — Request Authentication (007) owns header attachment and the active-identity line.
- **Exit-code numbers** — 004 owns the category→code mapping; Storage emits a code-free outcome category only.
- **Removing/clearing credentials (logout)** — out of scope this slice (spec non-behavior); a future spec adds it without reshaping the file contract.
- **Final `.glassfrogrc` name / `GLASSFROG_TOKEN` name / `0600` mode** — `[ASSUMED]`, jointly held with 005; reconcile before both ship.
