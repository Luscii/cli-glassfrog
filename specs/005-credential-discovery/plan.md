# Plan: Credential Discovery

**Feature**: 005-credential-discovery
**Role**: Shaper
**Inputs**: spec.md (005-credential-discovery), PROJECT.md, `.score/memory/DECISIONS.md` (read at 10 entries — Go self-contained, cobra, guard, explicit wiring, cobra resolution, outcome category, cobra help, built-ins hide; this plan's post-step appends new credential-resolution precedent), `.score/memory/LEARNINGS.md` (3 cobra findings — not directly relevant; this feature does not touch the cobra tree). No SOUL.md.

---

## System Architecture

Credential Discovery is the first capability that steps outside the cobra command tree: it touches the **filesystem** and the **process environment** to answer one question — "what token are we operating as, right now, in this directory?" It is an internal resolver consumed by Request Authentication (007), not a registered command of its own (spec non-behavior: no CLI surface here). Per Composition (CONSTITUTION V), it lives in its own package rather than inside `internal/cli`.

The parts:

- **Credentials package** (`internal/auth`) — the new home for token concerns. Credential Discovery contributes the *resolver*; the sibling Credential Storage (006) will contribute the *writer* into the same package, and both share one **file-format reader/writer** so the read and write sides cannot drift. Request Authentication (007) is the consumer — it lives with the API client and calls the resolver.
- **Resolver** — a single function that, given a starting directory and a home directory, walks the precedence chain (environment variable → nearest `.glassfrogrc` up the directory tree → home `.glassfrogrc`) and returns a **Resolution**: the token plus the *source* it came from, or a "none found" outcome, or a typed error for a file that exists but can't be read or parsed.
- **File-format reader** — a minimal `key=value` parser shared with Credential Storage. Reads the `token` key out of an `.glassfrogrc` file; ignores blank lines and `#` comments; reports a parse failure rather than guessing.
- **Resolution outcome** — a small, code-free result (mirroring 002's outcome-category precedent): `Resolution{Token, Source, Path}`, where `Source` is an enum (`Environment`, `File`, or `None`) and `Path` carries the file path when `Source` is `File` (empty otherwise). Discovery classifies *what was found and from where*; it emits no exit code and prints no output (those are the consumer's, Exit-Code Convention's, and — for the operator-facing "acting as" line — Request Authentication's concerns).

```
internal/auth                         (new package; shared by 005/006/007)
 ├─ resolve(startDir, homeDir) → (Resolution, error)   (005: this feature)
 │     1. env GLASSFROG_TOKEN non-empty?  → Resolution{Token, Source: Environment}
 │     2. walk startDir → … → root: nearest .glassfrogrc with a usable token wins
 │     3. else homeDir/.glassfrogrc
 │     · file exists, parses, no token  → skip, continue chain
 │     · file unreadable / unparseable  → return typed error (fail loud)
 │     · nothing anywhere               → Resolution{Source: None}
 ├─ readCredentialsFile(path) → (token string, found bool, error)   (shared reader)
 └─ (006 Credential Storage adds the writer; 007 Request Authentication consumes Resolution)
        ▲
        │ 007: attaches Resolution.Token as the X-Auth-Token header,
        │      and reports Resolution.Source as the active identity (never the token)
```

---

## Architecture Decisions

### ADR-1: Credential resolution lives in its own `internal/auth` package

**Context**: Token Authentication is three capabilities — Storage (writes the file), Discovery (reads/resolves), Request Authentication (attaches the header). The spec forbids Discovery from registering a command or making API calls. CONSTITUTION V requires modular parts that combine without entanglement.

**Options considered**:
1. **A dedicated `internal/auth` package** holding the resolver and the shared file reader, consumed by the API-client/request layer. Storage later adds its writer beside the reader. Clear ownership; the read and write sides share one file-format module.
2. **Put resolution inside `internal/cli`** — colocated with dispatch. Rejected: couples a filesystem/identity concern to the command-tree layer, and Discovery has no command, so it doesn't belong there.

**Decision**: Option 1 — a new `internal/auth` package. Discovery contributes the resolver and the shared `.glassfrogrc` reader; Storage (006) adds the writer to the same package; Request Authentication (007) consumes the `Resolution`.

**Consequences**: A clean module boundary that 006 and 007 build on. The file-format reader is written here but is a *shared* artifact — Storage's writer must round-trip with it. *Precedent-setting: the auth package and the read/write-share-one-format rule are the contract the rest of Token Authentication builds on.*

### ADR-2: Resolution order is env-first, then a walk-up, then home — nearest usable token wins

**Context**: The spec fixes the precedence (environment variable → working-directory file via npmrc-style walk-up → home file) and the nearest-wins rule. The home directory may itself be an ancestor of the start directory.

**Options considered**:
1. **Env → walk-up(start→root) → home fallback**, taking the first *usable* token; a parseable-but-tokenless file is skipped and the chain continues. The home path is appended to the walk-up candidate list only if it was not already visited, so a home that sits on the ascent path is read once.
2. **Check only CWD then home** (no ascent). Rejected: the spec explicitly chose the npm walk-up so a token applies across a project subtree.
3. **Merge values across all files** (npm-style layered config). Rejected: a single token has nothing to merge; nearest-wins is the whole semantic.

**Decision**: Option 1. Build an ordered, de-duplicated candidate list — `start`, each ancestor up to the filesystem root, then `home` if not already present — and return the first candidate that yields a usable token. The environment variable short-circuits ahead of all files when non-empty.

**Consequences**: Deterministic resolution (Validation Scenario: same env+fs → same source every run). "Usable token" must be defined precisely (ADR-4) so a blank entry doesn't win. The home-dedupe avoids reading the same file twice and keeps the "found during walk-up at its position" behavior the spec's edge case describes.

### ADR-3: A minimal hand-rolled `key=value` reader is the shared `.glassfrogrc` format `[ASSUMED]`

**Context**: Discovery must read the same shape Credential Storage writes. The spec pins the format as `[ASSUMED]`: `.glassfrogrc`, npmrc-style `key=value`, token under a `token` key — to be reconciled with Storage (006). CONSTITUTION XII wants a self-contained binary; minimizing dependencies serves that.

**Options considered**:
1. **Hand-roll a tiny line reader** — split each non-blank, non-`#` line on the first `=`, trim, collect into a map, read `token`. ~30 lines, no dependency, trivially shared with Storage's writer.
2. **Pull in an INI/dotenv library**. Rejected: a third-party parser is far more surface than a single `token=` key needs, and adds a dependency for no benefit.

**Decision**: Option 1 — a hand-rolled reader, shared with Storage. Unknown keys are ignored (forward-compatible if config keys are added later). A line that is non-blank, non-comment, and has no `=` makes the file **unparseable** → a format error (ADR-4), not a silently-skipped line.

**Consequences**: Zero new dependencies; the format stays legible and hand-editable. Because the format is `[ASSUMED]` and *shared*, the interface step pins it as a contract and Storage (006) must conform — flagged for reconciliation in the combined handoff. The env var name (`GLASSFROG_TOKEN`) is likewise `[ASSUMED]` and centralized as a constant.

### ADR-4: Discovery returns a code-free Resolution; absence is a normal outcome, broken files are errors

**Context**: The consumer (007) needs to tell three things apart: a token was found (and from where), nothing was found anywhere, and a credential file exists but is broken. CONSTITUTION III (Fail Safe, Not Silent) forbids letting a broken credential masquerade as "no credentials". CONSTITUTION II wants the active identity reportable. 002's precedent is "the resolver classifies a code-free outcome; the consumer maps it."

**Options considered**:
1. **`Resolution{Token, Source, Path}` + `error`**, where `Source: None` is the normal "nothing found" outcome (no error) and `error` is reserved for read/parse failures of a file that *does* exist. Absence stays out of the error channel; broken files fail loud.
2. **Sentinel `ErrNoCredentials`** for absence. Rejected: absence is an expected, non-exceptional outcome the consumer routinely handles; modelling it as an error blurs it with genuine I/O/format failures and complicates the consumer's branching.
3. **Return only the bare token string**. Rejected: loses the source (needed for the operator-facing "acting as" line, CONSTITUTION II) and can't distinguish absence from a broken file.

**Decision**: Option 1. `Resolution` carries the `Token`, a `Source` enum (`Environment`, `File`, or `None`), and a `Path` (the file path when `Source` is `File`, empty otherwise); a read or format failure of an existing file returns a typed error naming the path. The token value is never placed in any error string or log (spec non-behavior; secret hygiene).

**Consequences**: 007 gets a clean three-way input (found / none / error) with no exit-code coupling, exactly as 002→004 did for dispatch. `Source` is safe to surface; the token is not. *Precedent-setting: the Resolution shape is the contract 007 consumes.*

### ADR-5: The resolver takes its start and home directories as inputs, not from process globals

**Context**: CONSTITUTION IV requires test-first development, and the behavior is almost entirely filesystem- and environment-dependent (walk-up, home fallback, permission/parse errors). Reading `os.Getwd`/`os.UserHomeDir` directly inside the core logic would force tests to mutate real process state.

**Options considered**:
1. **Inject `startDir` and `homeDir` (and read the env var through a tiny seam)** into the resolver; a thin entrypoint wrapper supplies the real `os` values in production. Hermetic, table-driven unit tests against temp directory trees; no global mutation.
2. **Read globals inside the resolver**. Rejected: makes tests non-hermetic (must `Chdir`, must override `HOME`), risks tests reading the developer's real `~/.glassfrogrc`, and couples the algorithm to process state.

**Decision**: Option 1 — dependency-injected roots. The pure resolver is fully exercised with temp dirs; a small production seam binds it to `os.Getwd`/`os.UserHomeDir`/`os.Getenv`.

**Consequences**: Tests never touch the real home directory or CWD (a real safety property given the secret involved). The seam is the only place that reads globals, keeping the algorithm pure and deterministic. *Precedent-setting: filesystem/env-dependent resolvers inject their roots for hermetic tests.*

---

## Integration Design

- **Credentials file format (specification boundary, shared with Credential Storage 006)**: `.glassfrogrc`, `key=value` lines, `token` key. Discovery reads it; Storage writes it; the reader is shared so the two cannot diverge. This is the contract the interface step pins and that 006 must conform to.
- **Environment (input)**: `GLASSFROG_TOKEN`; non-empty value short-circuits the file search. Empty/unset falls through.
- **Resolution (internal contract, consumed by Request Authentication 007)**: `Resolution{Token, Source, Path}` (`Source: None` when nothing is found), plus typed read/format errors. 007 attaches the `Token` as `X-Auth-Token` and reports `Source`/`Path` (never the `Token`) as the operator-facing active identity.
- **Filesystem (input)**: reads candidate files up the directory tree and at home; tolerates *missing* files (skip), surfaces *unreadable/unparseable* ones (error).

---

## Cross-cutting Concerns

**Error handling (CONSTITUTION III)**: a file that exists but is unreadable or unparseable returns a typed error naming the path — never a silent fall-through to a different source, and never reported as "no credentials". Absence (nothing found anywhere) is a normal `Source: None` result, not an error.

**Secret hygiene (spec non-behavior; CONSTITUTION II)**: the token value never appears in output, logs, or error messages. Only the `Source` (a file path or "environment") is reportable. The interface and tests assert this explicitly.

**Testing strategy (CONSTITUTION IV)**: RED-first. The reader and resolver are unit-tested against temp directory trees with injected roots (ADR-5); each precedence rung, the nearest-wins case, the home-as-ancestor dedupe, empty-env, tokenless-skip, unreadable, and unparseable cases get coverage. The spec's driving scenarios become the godog BDD outer loop, exercising the production seam over temp dirs with `GLASSFROG_TOKEN` set/unset.

**Configuration**: the file name (`.glassfrogrc`) and env var (`GLASSFROG_TOKEN`) are centralized constants in `internal/auth`, marked `[ASSUMED]` pending reconciliation with Credential Storage.

**No command surface**: Discovery registers nothing in the cobra tree and prints nothing — so the LEARNINGS findings about cobra built-ins do not apply here.

---

## Implementation Strategy

Three phases, linear.

- **Phase 1 — Shared `.glassfrogrc` reader**: `internal/auth` package with `readCredentialsFile(path) → (token, found, error)` — `key=value` parse, `#`/blank tolerance, `token` extraction, parse-failure on a non-comment line without `=`. RED-first unit tests (valid, tokenless-but-valid, comments/blanks, malformed). *Depends on: nothing.*
- **Phase 2 — Resolver + precedence**: `resolve(startDir, homeDir) → (Resolution, error)` implementing env-first short-circuit, walk-up candidate list with home-dedupe, nearest-usable-token-wins, tokenless-skip, and fail-loud read/format errors; plus the production seam binding `os.Getwd`/`os.UserHomeDir`/`os.Getenv`. RED-first unit tests per precedence rung and edge case (empty-env, home-as-ancestor, unreadable via a deliberately-unreadable temp file). *Depends on: Phase 1.*
- **Phase 3 — Executable acceptance**: godog step definitions for the 005 driving scenarios (env override, nearest-wins, walk-up to ancestor, home fallback, unreadable error, malformed error, no-credentials, empty-env, tokenless-skip), running the production seam against temp dirs and a controlled `GLASSFROG_TOKEN`. *Depends on: Phase 2.*

---

## Risks

- **Walk-up + home dedupe correctness** (medium likelihood, medium impact): the home-as-ancestor case and the "stop at filesystem root" ceiling are easy to get subtly wrong (double-read, infinite loop, off-by-one at root). Mitigation: build an explicit, de-duplicated candidate list and unit-test the home-on-path and root-reached cases directly (the spec's edge scenarios pin both).
- **Non-hermetic tests touching the real home** (medium likelihood, high impact): a test that reads the developer's actual `~/.glassfrogrc` could pass/fail nondeterministically or leak a real token. Mitigation: ADR-5's injected roots — the pure resolver never reads globals; only the thin production seam does, and it is exercised via temp dirs in the BDD layer.
- **Secret leakage into errors/logs** (low likelihood, high impact): a naive error like `bad token "abc123"` would print the secret. Mitigation: errors name only the *path*; a validation scenario asserts the token never appears in produced output.
- **Shared file-format contract drift with Credential Storage (006)** (medium likelihood, medium impact): Storage is not yet specified; if it writes a different shape, reads break. Mitigation: pin the format in the interface step, share one reader/writer module (ADR-1/3), and flag reconciliation in the handoff. The `[ASSUMED]` markers travel forward.
- **"Usable token" definition** (low likelihood, medium impact): treating a whitespace-only value as a token would resolve a blank credential. Mitigation: define usable as non-empty after trimming (spec assumption), tested with a blank-value fixture.

---

## What This Plan Does Not Cover

- **Writing the credentials file** — Credential Storage (006) owns the login-style write; Discovery only reads. They share the file-format module.
- **Attaching the token to requests** — Request Authentication (007) attaches `X-Auth-Token` and reports the active identity; Discovery only resolves the value and its source.
- **Exit codes and operator output** — Discovery emits neither; Exit-Code Convention (004) and the consuming command own those. Discovery returns a code-free Resolution.
- **Final file name / env var name** — pinned as `[ASSUMED]` (`.glassfrogrc`, `GLASSFROG_TOKEN`); reconcile with Credential Storage before both ship.
