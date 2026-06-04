# Tasks: Credential Discovery

**Feature**: 005-credential-discovery
**Concretization**: Full context (plan + spec + interface + scenarios)
**Inputs**: plan.md, spec.md, interface-spec.md, features/unauthenticated-access/credential-discovery.feature

---

## Dependency Graph

Phase 1: Shared `.glassfrogrc` reader (1 task, no phase dependencies) [Shared]
Phase 2: Resolver + precedence (1 task, depends on Phase 1) [Shared]
Phase 3: Executable acceptance (1 task, depends on Phase 2) [Shared]

3 tasks total | 0 phases parallelizable (linear chain) | Builder: pipeline

> Every task is `[Shared]`: credential resolution is infrastructure serving all three user scenarios (stored-token / env-override / project-local-precedence) rather than any single one.

---

## Branching Guidance

**Pipeline mode**: `spec/005-credential-discovery/base` → `spec/005-credential-discovery/task-1`, `…/task-2`, `…/task-3` (one task branch per T-id, merged back into the spec base).

---

## Phase 1: Shared `.glassfrogrc` reader [Shared]

- [x] **T001** [Shared] Create the `internal/auth` package and the shared `.glassfrogrc` reader — `readCredentialsFile(path)` parsing npmrc-style `key=value`, with RED-first unit tests — 7 unit tests (valid/trim/tokenless/whitespace-value/comments+blanks/malformed/missing); no findings
  - **Scope**: New `internal/auth` package. Add `readCredentialsFile(path) → (token string, found bool, err error)`: split each line on the first `=`, trim key/value, ignore blank lines and `#`-comment lines, read the `token` key, treat a whitespace-only value as no token (`found = false`). A non-blank, non-comment line without `=` is a parse error. Centralize the file name (`.glassfrogrc`) and `GLASSFROG_TOKEN` as constants in the package. Reader only — no walk-up, no env, no resolver yet.
  - **Acceptance criteria**:
    - `readCredentialsFile` returns the token and `found = true` for a file containing `token=gf_x`
    - A file that parses but has no `token` key (or a whitespace-only value) returns `found = false`, no error
    - `#` comments and blank lines are ignored; unknown keys are ignored without error
    - A non-blank, non-comment line without `=` returns a format error naming the path; the token value is never included in any error
    - RED-first unit tests cover valid / tokenless / comments+blanks / malformed; `go build ./...` and `go vet ./...` clean
  - **Dependencies**: None
  - **Plan reference**: Phase 1 — Shared `.glassfrogrc` reader; ADR-1 (package), ADR-3 (format)
  - **Interface references**: interface-spec.md — `.glassfrogrc` structural contract

## Phase 2: Resolver + precedence [Shared]

- [x] **T002** [Shared] Implement `resolve(startDir, homeDir)` with env-first / walk-up / home-fallback precedence and the production seam — RED-first unit tests over temp directory trees — 13 unit tests (env-first/empty-env/nearest-wins/walk-up/home-fallback/home-on-ascent/tokenless-skip/none/unreadable/malformed + 3 candidateDirs dedup+root cases); no findings
  - **Scope**: Add `resolve(startDir, homeDir) → (Resolution, error)` returning `Resolution{Token, Source, Path}` (`Source ∈ {Environment, File, None}`). Order: non-empty `GLASSFROG_TOKEN` short-circuits to `Environment`; else build a de-duplicated candidate list (`startDir`, each ancestor to the filesystem root, then `homeDir` if not already present) and return the first file with a usable token (nearest-wins); a parseable-but-tokenless file is skipped; an unreadable or unparseable file returns a typed error naming the path (no fall-through); nothing found → `Source: None`, no error. Add a thin production seam binding `os.Getwd` / `os.UserHomeDir` / `os.Getenv` (the only place that reads globals — ADR-5). The token value never appears in any error or log.
  - **Acceptance criteria**:
    - Env-first: non-empty `GLASSFROG_TOKEN` wins and no file is read; an empty/unset value falls through to files
    - Nearest-wins: a current-directory file beats a home file; walk-up reaches an ancestor file; home is the final fallback
    - A home directory that lies on the ascent path is read once (no double-processing); the walk-up stops at the filesystem root (no infinite loop)
    - A tokenless file is skipped and the search continues; an unreadable file and an unparseable file each return a typed error naming the path and do not report absence; nothing found returns `Source: None` with no error
    - All resolver unit tests use injected `startDir`/`homeDir` over temp directories — none read the real `~/.glassfrogrc`; RED-first; `go build`/`go vet` clean
  - **Dependencies**: T001
  - **Plan reference**: Phase 2 — Resolver + precedence; ADR-2 (order), ADR-4 (Resolution), ADR-5 (injected roots)
  - **Interface references**: interface-spec.md — resolution precedence, `Resolution` output contract, Error Communication table
  - **Scenario references**: unauthenticated-access/credential-discovery.feature: "The environment token overrides a stored file", "An empty environment variable is ignored", "The nearest credentials file wins over the home file", "The search ascends to an ancestor's credentials file", "A home file on the ascent path is read once", "The home file is used when no project file exists", "A tokenless file is skipped for the next source", "No credentials anywhere is reported as absence", "An unreadable credentials file fails loudly", "A malformed credentials file fails loudly"
  - **Risk**: ⚠️ Walk-up + home-dedupe correctness (double-read / root off-by-one) and test hermeticity — build an explicit de-duplicated candidate list and exercise the home-on-path and root-reached cases over temp dirs; never touch the real home directory.

## Phase 3: Executable acceptance [Shared]

- [x] **T003** [Shared] Make the 005 driving scenarios pass as executable acceptance via godog, exercising the production seam over temp dirs and a controlled `GLASSFROG_TOKEN` — new `internal/auth` godog suite, 10 behavioral scenarios pass / 3 `@validation` held @wip; scoped the `cli` suite to its own feature file (noted in LEARNINGS)
  - **Scope**: Add godog step definitions for the Credential Discovery scenarios in `features/unauthenticated-access/credential-discovery.feature` (all three Rule blocks), driving the production seam against temp directory trees and a set/unset `GLASSFROG_TOKEN`, asserting the resolved `(Token, Source, Path)` or the typed read/format error. Remove `@wip` from the passing behavioral scenarios; keep the three `@validation` scenarios `@wip` (held out for validate).
  - **Acceptance criteria**:
    - Every non-`@validation` 005 scenario (env override / empty-env / nearest-wins / walk-up / home-on-path / home-fallback / tokenless-skip / no-credentials / unreadable / malformed) has an executable, passing path
    - `@wip` removed from those scenarios; the three `@validation` scenarios (deterministic / token-never-in-output / no-writes) keep `@wip`
    - Steps set and unset `GLASSFROG_TOKEN` and build temp `.glassfrogrc` trees without touching the developer's real home directory; the suite asserts no token value appears in captured output
    - `go build ./...`, `go vet ./...`, and the feature suite run clean
  - **Dependencies**: T002
  - **Plan reference**: Phase 3 — Executable acceptance; Cross-cutting Concerns (testing strategy)
  - **Scenario references**: unauthenticated-access/credential-discovery.feature: all 005 Rule-block scenarios
  - **Risk**: ⚠️ Test isolation — environment-variable and CWD/home manipulation must be hermetic and not leak between scenarios; restore/override per scenario and confine all files to temp dirs.
