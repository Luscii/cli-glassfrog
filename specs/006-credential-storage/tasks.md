# Tasks: Credential Storage

**Feature**: 006-credential-storage
**Concretization**: Full context (plan + spec + interface + scenarios)
**Inputs**: plan.md, spec.md, interface-cli.md, interface-spec.md, features/unauthenticated-access/credential-storage.feature

---

## Dependency Graph

Phase 1: Writer in `internal/auth` (1 task, no phase dependencies) [Shared]
Phase 2: Token-source resolution & interactivity (2 tasks, depends on Phase 1) [Shared]
Phase 3: Command wiring & executable acceptance (2 tasks, depends on Phase 2) [Shared]

5 tasks total | 0 phases parallelizable (linear chain) | Builder: pipeline

> Every task is `[Shared]`: storing a credential is infrastructure serving all three user scenarios (store-once / automation / project-local) rather than any single one. Scenario references below map each task to the specific scenarios it satisfies.

---

## Branching Guidance

**Pipeline mode**: `spec/006-credential-storage/base` → `spec/006-credential-storage/task-1`, `…/task-2`, … `…/task-5` (one task branch per T-id, merged back into the spec base).

**Role-based mode**: `spec/006-credential-storage/base` is the integration point; task branches as above. 005-credential-discovery is an in-progress sibling on the same `internal/auth` package and the shared `unauthenticated-access` problem — coordinate the shared format module (see T001) so the two specs do not create two implementations of the `.glassfrogrc` shape.

---

## Phase 1: Writer in `internal/auth` [Shared]

- [x] **T001** [Shared] Add `writeCredentials(path, token)` to `internal/auth` — line-preserving merge, atomic write, owner-only permissions — with RED-first unit tests including a round-trip against the reader — 12 unit tests; created shared format module (reader+constants) to 005's contract since 005 unimplemented; exported as `auth.WriteCredentials`/`auth.ReadCredentialsFile` (cross-package use in T004)
  - **Scope**: Add `writeCredentials(path, token) → error` to `internal/auth`. Parse-validate any existing file via the shared `readCredentialsFile` (a malformed non-comment line without `=` → format error, **no write**). On success, rewrite at the line level: replace the value on the existing `token=` line, or append a `token=` line if absent, preserving every other line and comment and their order; an absent target file is created. Serialize to a temp file **in the same directory**, `chmod 0600` before writing token bytes, then `rename` over the target (atomic; failure leaves the original/absence intact). If `internal/auth` and the shared reader/constants do not yet exist (005 not yet implemented), create them to 005's `interface-spec.md` contract. The token value never appears in any error or log.
  - **Acceptance criteria**:
    - Writing to an absent path creates `.glassfrogrc` with a `token=` line and `0600` permissions
    - Writing over a file with other keys/comments replaces only the `token` value and leaves every other line unchanged
    - An existing file with a non-blank, non-comment line without `=` returns a format error naming the path and is not overwritten
    - A simulated write failure leaves the original file (or its absence) unchanged; no temp file is left behind
    - **Round-trip**: a token written by `writeCredentials` is returned unchanged by `readCredentialsFile` for the same path
    - No error or log contains the token value; RED-first unit tests; `go build ./...` and `go vet ./...` clean
  - **Dependencies**: None
  - **Plan reference**: Phase 1 — Writer in `internal/auth`; ADR-1 (writer joins auth, round-trip), ADR-4 (line-preserving merge, atomic write, 0600)
  - **Interface references**: interface-spec.md — write-outcomes table, at-rest guarantees, round-trip contract
  - **Scenario references**: credential-storage.feature: "Re-storing preserves other entries in the file", "A malformed existing file fails the store loudly", "An unwritable target fails the store loudly", "A stored token is resolved back unchanged", "A newly stored file is readable only by its owner"
  - **Risk**: ⚠️ Atomicity & permissions — temp file must be same-directory (single-filesystem rename) and `0600` set before token bytes; test original-preserved-on-failure and owner-only perms. Hermetic: never write the real `~/.glassfrogrc`.

## Phase 2: Token-source resolution & interactivity [Shared]

- [ ] **T002** [Shared] Pure token-source precedence, blank rejection, target-path selection, and the existing-token guard — RED-first unit tests with injected sources
  - **Scope**: Add pure logic (injected sources, no real `os` reads): `resolveToken` choosing a single token in precedence order argument → piped stdin → `GLASSFROG_TOKEN` → prompt-marker, rejecting empty/whitespace-only values as unusable, and signalling the non-interactive-no-token case. Add target-path selection from injected `startDir`/`homeDir` (home default; current-directory file when `--cwd`). Add the existing-token guard: non-interactive with an existing token → error unless an overwrite signal is set; interactive → confirm/choose-location decision surfaced to the caller. All branches decided from injected inputs (`isTTY` flag, source set), returning a code-free outcome category (`UsageError` for no-token / no-override / blank; success otherwise) for 004 to map.
  - **Acceptance criteria**:
    - Precedence: argument beats piped stdin beats `GLASSFROG_TOKEN` beats prompt; first present wins
    - An empty/whitespace-only token from any source is rejected as unusable (no write requested)
    - Non-interactive (`isTTY` false) with no token from any source yields a "no token to store" usage outcome
    - Non-interactive with an existing token and no overwrite signal yields an error outcome and requests no write; with the overwrite signal it requests a single-target merge
    - Target path is the home file by default and the current-directory file under `--cwd`
    - All tests use injected sources/roots — none read the real stdin/TTY/env/home; RED-first; `go build`/`go vet` clean
  - **Dependencies**: T001
  - **Plan reference**: Phase 2 — Token-source resolution; ADR-3 (injected input seam), ADR-5 (code-free outcome)
  - **Interface references**: interface-cli.md — token-source precedence, existing-stored-token rules, Error Communication table
  - **Scenario references**: credential-storage.feature: "A token argument is stored to the home file", "A token in the environment is persisted", "A blank token is rejected", "A non-interactive store with no token is reported", "A non-interactive overwrite requires the overwrite flag"

- [ ] **T003** [Shared] Production input seam — bind real stdin / TTY detection / env / directories and the non-echoing prompt
  - **Scope**: Add the thin production seam that supplies real values to T002's pure logic: read piped stdin, detect whether stdin is a terminal, read `GLASSFROG_TOKEN`, resolve `os.Getwd`/`os.UserHomeDir`, and implement the non-echoing interactive prompt (no characters echoed; the prompt requests the token only when stdin is a TTY and no other source supplied one). The seam is the only place that reads these globals. The token read from stdin/prompt is never echoed or logged.
  - **Acceptance criteria**:
    - Piped stdin is read as the token without prompting; a TTY with no other source triggers the non-echoing prompt; a non-TTY with no source does not prompt
    - The prompt does not echo typed characters and the token never appears in output
    - The seam binds the real `os` values and is the only code reading them (T002 stays pure)
    - `go build ./...` / `go vet ./...` clean (the interactive prompt path is exercised via the BDD seam in T005)
  - **Dependencies**: T002
  - **Plan reference**: Phase 2 — production seam; ADR-3 (injected input seam)
  - **Interface references**: interface-cli.md — Interactivity (TTY) and piping/scripting
  - **Scenario references**: credential-storage.feature: "A piped token is stored to the current directory"

## Phase 3: Command wiring & executable acceptance [Shared]

- [ ] **T004** [Shared] Register the `auth` group and `auth login` leaf via the guard, wire in main, classify outcomes — delegating the write to `internal/auth`
  - **Scope**: Add the `auth` command group (non-runnable, non-empty `Short`) and the `auth login [TOKEN]` leaf (`Args: cobra.MaximumNArgs(1)`, non-empty `Short`, flags `--cwd` and `--overwrite`), registered through the `MustRegister` guard and wired explicitly in `main` (no `init()`). The leaf composes T003's seam → T002's resolution/guard → T001's writer, classifies the outcome into the existing `Outcome` categories (no `os.Exit`, no new codes — 004 maps), writes the success line naming the path (never the token) to stdout, and write/format/usage errors to stderr.
  - **Acceptance criteria**:
    - `glassfrog auth login` resolves; `auth` is a non-runnable group with a summary; the leaf accepts at most one positional and rejects extras as a usage error
    - `--cwd` targets the current-directory file; `--overwrite` is the non-interactive override signal
    - Success prints the written path and never the token; errors go to stderr and map to the documented categories (no-token / no-override / blank → usage; write / format → internal) via the existing `ExitCode` site
    - No package-global cobra toggle is changed; `go build`/`go vet` clean
  - **Dependencies**: T003
  - **Plan reference**: Phase 3 — Command wiring; ADR-2 (guard-registered leaf delegating to writer), ADR-5 (outcome categories)
  - **Interface references**: interface-cli.md — Surface (`auth login`, flags), output, Error Communication table
  - **Scenario references**: credential-storage.feature: all 14 scenarios (command-level behavior)

- [ ] **T005** [Shared] Executable acceptance — godog step definitions for the 006 scenarios over temp dirs and controlled stdin / TTY / `GLASSFROG_TOKEN`
  - **Scope**: Add godog step definitions for the scenarios in `features/unauthenticated-access/credential-storage.feature` (all three Rule blocks), driving the command/production seam against temp directory trees with a controlled token source (argument, piped stdin, `GLASSFROG_TOKEN`) and a controlled `isTTY` signal, asserting written file contents, reported path, preserved keys, `0600` permissions, the round-trip with the reader, and the no-token-in-output property. Reuse existing 005 step vocabulary for file-state Givens and "naming that file" assertions; reserve new bindings for the write/merge/prompt steps. Remove `@wip` from passing behavioral scenarios; keep the three `@validation` scenarios `@wip` (held out for validate).
  - **Acceptance criteria**:
    - Every non-`@validation` 006 scenario (arg→home / env-persist / interactive-prompt / blank-rejected / no-token-non-interactive / overwrite-required / piped→cwd / interactive-location-choice / unwritable / malformed / merge-preserves) has an executable, passing path — interactive paths driven through the injected `isTTY` seam
    - `@wip` removed from those scenarios; the three `@validation` scenarios (round-trip / token-never-in-output / owner-only-readable) keep `@wip`
    - Steps confine all files to temp dirs and control stdin/TTY/`GLASSFROG_TOKEN` without touching the real home directory; the suite asserts no token value appears in captured output
    - New step bindings reuse existing step text where the assertion already exists; `go build ./...`, `go vet ./...`, and the feature suite run clean
  - **Dependencies**: T004
  - **Plan reference**: Phase 3 — Executable acceptance; Cross-cutting Concerns (testing strategy, BDD shared-file step vocabulary)
  - **Scenario references**: credential-storage.feature: all 006 Rule-block scenarios
  - **Risk**: ⚠️ Test isolation — stdin/TTY/env/CWD/home manipulation must be hermetic and not leak between scenarios; restore per scenario and confine files to temp dirs.
