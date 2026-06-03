# Risk: Credential Discovery

**Feature**: 005-credential-discovery
**Round**: 1
**Date**: 2026-06-03
**Artifacts loaded**: spec.md, plan.md, interface-spec.md, PROJECT.md
**Acceptability matrix**: Default 3×3 traffic light

> ⚠ Using default risk acceptability matrix — no project-level matrix found in PROJECT.md.

---

## Risk Register

| H-ID | Hazard | Source | Severity | Probability | Risk Level | Controls | Residual Risk |
|---|---|---|---|---|---|---|---|
| H-1 | Token value leaks into output, logs, or error messages | spec.md § Non-Behaviors | High | Medium | Red | RC-1 | Yellow |
| H-2 | Walk-up reads a `.glassfrogrc` planted in an ancestor directory → CLI acts as an unintended identity | plan.md § System Architecture (ADR-2 walk-up) | High | Medium | Red | RC-2 | Yellow |
| H-3 | Unreadable file silently falls through to a different/stale source | spec.md § Behavioral Accord > Error handling | Medium | Low | Green | RC-3 | Green |
| H-4 | Malformed file reported as "no credentials", masking a broken credential | spec.md § Behavioral Accord > Error handling | Medium | Low | Green | RC-3 | Green |
| H-5 | Empty/whitespace env var or blank value resolved as a real token | spec.md § Driving Scenarios (empty-env edge) | Medium | Medium | Yellow | RC-4 | Green |
| H-6 | Walk-up dedupe/ceiling bug — double-reads home, or fails to stop at root | plan.md § Risks (walk-up + home dedupe) | Medium | Medium | Yellow | RC-5 | Green |
| H-7 | Non-hermetic tests read the developer's real `~/.glassfrogrc`, leaking a token | plan.md § Risks (non-hermetic tests) | Medium | Low | Green | RC-6 | Green |
| H-8 | A tokenless nearest file halts the search, missing a valid home token | spec.md § Driving Scenarios (tokenless edge) | Low | Low | Green | RC-7 | Green |
| H-9 | The resolver accidentally writes/corrupts the credentials file | spec.md § Non-Behaviors | Medium | Low | Green | RC-8 | Green |

---

## Hazard Details

### H-1: Token leakage in output, logs, or errors

**Source**: spec.md § Non-Behaviors — "must not print, log, or otherwise expose the token value in plaintext output"; interface-spec.md § Error Communication.

**Description**: The resolved token is a credential that authenticates as a specific org + person. If it appears in an error message, a debug log, or stdout, it can be captured from a terminal, a CI transcript, or a log aggregator and used to impersonate the practitioner.

**Severity**: High — exposure of a live API token enables full impersonation within the caller's permissions against live governance.

**Probability**: Medium — error messages and logs are the natural place a value being processed leaks; a naive `fmt.Errorf("bad token %q", tok)` would expose it by default.

**Risk Level**: Red (High × Medium)

**Controls**:
- **RC-1**: Errors and diagnostics carry only the file *path*, never the token value; the resolved `Token` is excluded from all rendered output (only `Source`/`Path` are reportable).

**Residual Risk**: Yellow — with RC-1, the token is structurally kept out of output. Residual exposure depends on the consumer (Request Authentication 007) upholding the same discipline; pinned within this feature by the validation scenario "The token value never appears in output". Acceptable with documented justification: secret hygiene is a cross-capability contract carried forward to 007.

### H-2: Ancestor-planted credentials file injects an unintended identity

**Source**: plan.md § System Architecture / ADR-2 (env-first → walk-up → home); spec.md § Behavioral Accord > Resolution order.

**Description**: The walk-up ascends from the working directory through every parent to the filesystem root. A `.glassfrogrc` placed in a shared or untrusted ancestor directory (e.g. a common workspace root, a cloned repository, `/tmp` subtree) would be discovered and its token used — causing the CLI to authenticate and act on live governance as a *different* org + person than the operator intended.

**Severity**: High — acting on governance under the wrong identity is exactly the harm the authentication boundary exists to prevent.

**Probability**: Medium — running the CLI inside a directory nested under a shared parent is common, and this is the inherent, accepted tradeoff of the npm-style walk-up the feature deliberately chose.

**Risk Level**: Red (High × Medium)

**Controls**:
- **RC-2**: Deterministic, transparent precedence — the resolved `Source`/`Path` is reported so the operator (or agent) can see *which* file supplied the token, and an env-first `GLASSFROG_TOKEN` lets automation override file discovery entirely (bypassing the walk-up).

**Residual Risk**: Yellow — the walk-up tradeoff is intrinsic to the chosen model and cannot be fully eliminated without abandoning it. RC-2 reduces it by making the active source visible and offering an env override for untrusted directories. Acceptable with documented justification: this is the user-selected npm precedence model; the transparency of `Source`/`Path` makes a wrong-identity resolution detectable. Reconsider if a future requirement needs trust-boundary restrictions on which ancestors are searched.

### H-3: Unreadable file falls through silently

**Source**: spec.md § Behavioral Accord > Error handling; CONSTITUTION III (Fail Safe, Not Silent).

**Description**: If a `.glassfrogrc` that exists but cannot be read (permission denied) were silently skipped, resolution could fall through to a different, possibly stale token — or to "no credentials" — hiding a real configuration problem.

**Severity**: Medium — wrong-token or confusing-absence outcome; recoverable once surfaced.

**Probability**: Low — the plan mandates a typed read error naming the file, with no fall-through.

**Risk Level**: Green (Medium × Low)

**Controls**:
- **RC-3**: Fail-loud typed errors — an unreadable file returns a read error naming the path; a malformed file returns a format error; neither falls through nor is reported as absence.

**Residual Risk**: Green — pinned by the scenario "An unreadable credentials file fails loudly".

### H-4: Malformed file masked as absence

**Source**: spec.md § Behavioral Accord > Error handling; interface-spec.md § Error Communication.

**Description**: If an unparseable `.glassfrogrc` were treated as "no credentials found", a broken-but-present credential would be silently ignored, sending the operator down a confusing "not logged in" path instead of "fix your file".

**Severity**: Medium — misdirected diagnosis; recoverable.

**Probability**: Low — the plan mandates a distinct format error, not absence.

**Risk Level**: Green (Medium × Low)

**Controls**:
- **RC-3**: (shared) Fail-loud typed format error naming the path; never reported as absence.

**Residual Risk**: Green — pinned by the scenario "A malformed credentials file fails loudly".

### H-5: Blank value resolved as a token

**Source**: spec.md § Driving Scenarios (empty-env edge); plan.md § ADR-4 / Assumptions ("usable token").

**Description**: If an empty `GLASSFROG_TOKEN` or a whitespace-only `token=` value were accepted, the CLI would send an empty `X-Auth-Token` and fail authentication with a confusing downstream 401 rather than the clear "no credentials" outcome.

**Severity**: Medium — confusing failure, no data harm.

**Probability**: Medium — empty environment variables are common in CI and shell scripting.

**Risk Level**: Yellow (Medium × Medium)

**Controls**:
- **RC-4**: "Usable token" is defined as non-empty after trimming; an empty env var and whitespace-only values fall through rather than resolving as a token.

**Residual Risk**: Green — pinned by "An empty environment variable is ignored".

### H-6: Walk-up dedupe/ceiling bug

**Source**: plan.md § Risks (walk-up + home dedupe correctness).

**Description**: The ascent could read the home file twice (when home is an ancestor), loop without terminating, or mis-handle the filesystem root, producing a wrong result or a hang.

**Severity**: Medium — wrong resolution or process hang.

**Probability**: Medium — directory-walk termination and dedupe are classic off-by-one/edge sources.

**Risk Level**: Yellow (Medium × Medium)

**Controls**:
- **RC-5**: An explicit, de-duplicated candidate list with a filesystem-root ceiling; home is appended only if not already visited; unit tests cover the home-on-path and root-reached cases.

**Residual Risk**: Green — pinned by "A home file on the ascent path is read once" plus the resolver unit tests.

### H-7: Non-hermetic tests leak the real home token

**Source**: plan.md § Risks (non-hermetic tests touching the real home).

**Description**: A test that reads the developer's actual `~/.glassfrogrc` could surface a real token in CI logs or pass/fail nondeterministically depending on the developer's machine.

**Severity**: Medium — a real secret could appear in CI output.

**Probability**: Low — ADR-5 injects the roots so the pure resolver never reads globals.

**Risk Level**: Green (Medium × Low)

**Controls**:
- **RC-6**: The resolver takes injected `startDir`/`homeDir`; only the thin production seam reads `os` globals; all resolver and BDD tests run over temp directories.

**Residual Risk**: Green — grounded in ADR-5 and the T002/T003 acceptance criteria.

### H-8: Tokenless file halts the search

**Source**: spec.md § Driving Scenarios (tokenless edge).

**Description**: If a present-but-tokenless `.glassfrogrc` stopped the search, a valid token in a farther location (e.g. the home file) would be missed, producing a spurious "no credentials".

**Severity**: Low — recoverable; the operator sees absence and can act.

**Probability**: Low — the plan mandates skip-and-continue.

**Risk Level**: Green (Low × Low)

**Controls**:
- **RC-7**: A parseable-but-tokenless file is skipped and the search continues to the next source.

**Residual Risk**: Green — pinned by "A tokenless file is skipped for the next source".

### H-9: Accidental write to the credentials file

**Source**: spec.md § Non-Behaviors ("must not write, create, or modify any credentials file"); CONSTITUTION IX.

**Description**: A regression in which the resolver writes to or truncates the file it reads could clobber a stored token — Discovery is supposed to be strictly read-only.

**Severity**: Medium — could destroy a stored credential.

**Probability**: Low — the design is read-only; writing is Credential Storage's (006) concern.

**Risk Level**: Green (Medium × Low)

**Controls**:
- **RC-8**: Read-only resolver — performs no filesystem writes; pinned by the validation scenario "Resolution never writes to the filesystem".

**Residual Risk**: Green.

---

## Residual Risk Summary

| Level | Count | Hazards |
|---|---|---|
| Red (unacceptable) | 0 | — |
| Yellow (justified) | 2 | H-1, H-2 |
| Green (accepted) | 7 | H-3, H-4, H-5, H-6, H-7, H-8, H-9 |

**Unacceptable risks**: None. All residual risks are Yellow or Green after controls. The two Yellows are the secret-hygiene boundary (H-1, carried forward to Request Authentication 007) and the inherent ancestor-token tradeoff of the chosen npm walk-up model (H-2, mitigated by source transparency + env override) — both acceptable with the documented justifications above.

---

## Traceability Index

### Hazards

| ID | Source |
|---|---|
| H-1 | spec.md § Non-Behaviors |
| H-2 | plan.md § System Architecture (ADR-2) |
| H-3 | spec.md § Behavioral Accord > Error handling |
| H-4 | spec.md § Behavioral Accord > Error handling |
| H-5 | spec.md § Driving Scenarios (empty-env edge) |
| H-6 | plan.md § Risks |
| H-7 | plan.md § Risks |
| H-8 | spec.md § Driving Scenarios (tokenless edge) |
| H-9 | spec.md § Non-Behaviors |

### Controls

| ID | Mitigates | Grounding |
|---|---|---|
| RC-1 | H-1 | spec.md § Non-Behaviors / interface-spec.md § Error Communication — token excluded from output |
| RC-2 | H-2 | interface-spec.md § Surface (`Resolution` source/path) / plan.md § ADR-2 — env-first override + source transparency |
| RC-3 | H-3, H-4 | plan.md § Cross-cutting Concerns (error handling) / interface-spec.md § Error Communication |
| RC-4 | H-5 | plan.md § ADR-4 / Assumptions — usable-token definition |
| RC-5 | H-6 | plan.md § ADR-2 / Risks — de-duplicated candidate list + root ceiling |
| RC-6 | H-7 | plan.md § ADR-5 — injected filesystem roots |
| RC-7 | H-8 | spec.md § Behavioral Accord / interface-spec.md § Interactions — skip-and-continue |
| RC-8 | H-9 | spec.md § Non-Behaviors / interface-spec.md § Consistency Notes — read-only resolver |
