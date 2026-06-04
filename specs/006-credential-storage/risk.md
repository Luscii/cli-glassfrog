# Risk: Credential Storage

**Feature**: 006-credential-storage
**Round**: 1
**Date**: 2026-06-04
**Artifacts loaded**: spec.md, plan.md, interface-cli.md, interface-spec.md, PROJECT.md
**Acceptability matrix**: default 3×3 traffic-light — no project-level matrix found in PROJECT.md
**Regulatory bridge**: none — PROJECT.md defines no Regulatory Context

---

## Risk Register

| H-ID | Hazard | Source | Sev | Prob | Risk | Controls | Residual |
|---|---|---|---|---|---|---|---|
| H-1 | The token value leaks into output, logs, an error, or a prompt echo | spec Non-Behaviors; interface secret hygiene | High | Low | Yellow | RC-1, RC-2 | Green |
| H-2 | The credentials file is written group/world-readable, exposing the secret at rest | interface-spec at-rest; plan ADR-4 | High | Low | Yellow | RC-3 | Yellow |
| H-3 | A mid-write failure corrupts or destroys an existing credentials file | plan ADR-4; spec error handling | High | Low | Yellow | RC-4 | Green |
| H-4 | A malformed existing file is silently overwritten, discarding its other content | spec error "format error"; plan ADR-4 | Medium | Low | Green | RC-5 | Green |
| H-5 | A merge clobbers unrelated keys/comments in the file | spec edge "preserves other keys" | Medium | Low | Green | RC-6 | Green |
| H-6 | The interactive prompt hangs automation (non-interactive misdetected as a TTY) | spec interactivity; plan Risks | Medium | Medium | Yellow | RC-7 | Yellow |
| H-7 | An existing token is replaced without explicit intent | CONSTITUTION IX; spec existing-token | High | Low | Yellow | RC-8 | Green |
| H-8 | The writer produces a `.glassfrogrc` shape Discovery (005) cannot read (contract drift) | plan ADR-1/Risks; `[ASSUMED]` contract | High | Medium | Red | RC-9 | Yellow |
| H-9 | The token is written to the wrong location | spec location; plan ADR-3 | Medium | Low | Green | RC-10 | Green |
| H-10 | Tests write/read the developer's real `~/.glassfrogrc` (clobber/leak) | plan ADR-3/Risks | High | Low | Yellow | RC-11 | Yellow |

No residual risk remains **Red**. Four residual **Yellow** risks (H-2, H-6, H-8, H-10) are acceptable with the documented justifications below.

---

## Hazard Detail

### H-1 — Token leaks into produced output
**Severity High** — a leaked credential in a terminal, CI log, or transcript is a real compromise. **Probability Low** — the design routes the secret through several sinks (success line, error strings, the prompt, the stdin read), but each is explicitly constrained to omit the token.
**Controls**: **RC-1** the success message and all error strings name only the path, never the token; **RC-2** the interactive prompt is non-echoing and the stdin read is never echoed. Pinned by the validation scenario "The stored token value never appears in output".
**Residual Green** — comprehensive sink coverage plus a validation scenario.

### H-2 — File written too permissively
**Severity High** — a group/world-readable credentials file exposes the secret to other local users. **Probability Low** — the plan sets `0600` on the temp file before writing token bytes.
**Controls**: **RC-3** owner-only (`0600`) permissions set on the temp file *before* any token bytes, carried through the atomic rename; validation scenario "A newly stored file is readable only by its owner".
**Residual Yellow** — the `0600` mode is `[ASSUMED]` (pending confirmation) and POSIX-conditional; on platforms without permission bits the guarantee is best-effort. Acceptable with the noted platform caveat; confirm the `[ASSUMED]` mode during implementation.

### H-3 — Mid-write corruption
**Severity High** — losing or corrupting a stored credential breaks authentication and is hard to diagnose. **Probability Low** — the atomic temp-in-same-dir + `rename` makes a partial write impossible.
**Controls**: **RC-4** atomic write (temp file in the same directory, then `rename` over the target); test asserts the original/absence is preserved on a simulated failure.
**Residual Green**.

### H-4 — Malformed file overwritten
**Severity Medium** — discarding a user's other file content is destructive but recoverable. **Probability Low** — parse-validation gates the write.
**Controls**: **RC-5** the existing file is parse-validated via the shared reader before any write; a malformed file yields a format error and no write. Pinned by "A malformed existing file fails the store loudly".
**Residual Green**.

### H-5 — Merge clobbers unrelated keys
**Severity Medium** — silently dropping unrelated keys/comments surprises a hand-editor. **Probability Low** — line-preserving rewrite touches only the `token=` line.
**Controls**: **RC-6** line-preserving rewrite preserving every other line and comment; pinned by "Re-storing preserves other entries in the file".
**Residual Green**.

### H-6 — Prompt hangs automation
**Severity Medium** — a hung prompt stalls an AI-agent/CI run. **Probability Medium** — TTY detection can misfire across pipes, CI runners, and PTY-allocating wrappers.
**Controls**: **RC-7** a single injected `isTTY` signal drives both "read piped stdin" and "prompt only when interactive"; a non-interactive session with no token errors ("no token to store") instead of prompting.
**Residual Yellow** — correctness depends on accurate TTY detection at the production seam; acceptable given the explicit non-interactive-errors-not-prompts rule. *Note: the interactive prompt path itself currently lacks an acceptance scenario (see checklist IV / analyze K5) — closing that gap also exercises this control.*

### H-7 — Existing token replaced without intent
**Severity High** — silently replacing an active credential could swap the operator's identity. **Probability Low** — the design requires explicit intent to overwrite.
**Controls**: **RC-8** non-interactive sessions error unless `--overwrite` is given (intent via flag, per CONSTITUTION IX); interactive sessions confirm before replacing. Pinned by "A non-interactive overwrite requires the overwrite flag".
**Residual Green**.

### H-8 — Contract drift with Discovery (005)
**Severity High** — if the written shape diverges from what Discovery reads, stored credentials become unresolvable and authentication silently breaks. **Probability Medium** — 005 is not yet implemented, the `.glassfrogrc` format is `[ASSUMED]`, and two specs share one file shape.
**Controls**: **RC-9** one shared format module in `internal/auth` (no second implementation), with a write→read **round-trip test** as the binding contract; the `[ASSUMED]` name/format is reconciled with 005 before either ships. Pinned by the validation scenario "A stored token is resolved back unchanged".
**Residual Yellow** — the round-trip test reduces the initial-Red risk substantially, but cross-spec reconciliation is an open coordination action until both ship. Acceptable with that action tracked.

### H-9 — Wrong write location
**Severity Medium** — writing to an unexpected directory could store a token where it won't be found or where it's unintended. **Probability Low** — locations are explicit (home default; `--cwd`) and derived from injected roots.
**Controls**: **RC-10** injected `startDir`/`homeDir`; the location is the home file by default and the current-directory file only under the explicit `--cwd` flag.
**Residual Green**.

### H-10 — Tests touch the real home directory
**Severity High** — a test that writes the developer's real `~/.glassfrogrc` could clobber live credentials or leak a real token. **Probability Low** — the injected-roots design keeps tests in temp dirs, but it is a discipline that can be bypassed.
**Controls**: **RC-11** injected `startDir`/`homeDir` and stdin/TTY/env seam; all tests confine files to temp directories and never read/write real `os` dirs (plan ADR-3).
**Residual Yellow** — depends on test-authoring discipline (a `os.Getwd`/`os.UserHomeDir` call slipping into a test would bypass it); acceptable with the injected-roots convention and a review check, mirroring 005's same control.

---

## Residual Risk Summary

| Residual | Count | Hazards |
|---|---|---|
| Green | 6 | H-1, H-3, H-4, H-5, H-7, H-9 |
| Yellow | 4 | H-2, H-6, H-8, H-10 |
| Red | 0 | — |

The four Yellow residuals share a theme: they depend on a condition outside the core logic — platform permission support (H-2), accurate TTY detection (H-6), cross-spec contract reconciliation (H-8), and test-authoring discipline (H-10). Each has an assessment-level control; none is unacceptable. The two highest-leverage follow-ups are confirming the `0600` `[ASSUMED]` mode (H-2) and tracking the 005/006 contract reconciliation (H-8) before both ship.

---

## Traceability Index

**Hazards → source**:
- H-1 → spec.md § Non-Behaviors (token never printed/logged/echoed); interface-cli/-spec § secret hygiene
- H-2 → interface-spec.md § Surface (at-rest guarantees); plan.md ADR-4
- H-3 → plan.md ADR-4 (atomic write); spec.md § Behavioral Accord > Error handling
- H-4 → spec.md § Driving Scenarios (malformed merge); plan.md ADR-4
- H-5 → spec.md § Edge (merge preserves other keys)
- H-6 → spec.md § Behavioral Accord > Token input (interactivity); plan.md § Risks
- H-7 → CONSTITUTION.md IX; spec.md § Behavioral Accord > Existing credentials
- H-8 → plan.md ADR-1 + § Risks; the `[ASSUMED]` shared contract (DECISIONS, from 005)
- H-9 → spec.md § Behavioral Accord > Location selection; plan.md ADR-3
- H-10 → plan.md ADR-3 + § Risks

**Controls → architectural grounding**:
- RC-1, RC-2 → secret hygiene (plan § Security Design; interface Error Communication / Interactions)
- RC-3 → plan ADR-4 (0600 before bytes); interface-spec at-rest guarantees
- RC-4 → plan ADR-4 (temp-in-same-dir + rename)
- RC-5 → plan ADR-4 (parse-validate via shared reader)
- RC-6 → plan ADR-4 (line-preserving rewrite)
- RC-7 → plan ADR-3 (injected `isTTY` seam)
- RC-8 → plan ADR-2/ADR-5; CONSTITUTION IX (intent via `--overwrite` flag)
- RC-9 → plan ADR-1 (shared format module + round-trip test)
- RC-10, RC-11 → plan ADR-3 (injected `startDir`/`homeDir`)

Downstream: tasks.md acceptance criteria and `.feature` validation scenarios already realize RC-1 (no-output-leak), RC-3 (owner-only), RC-4 (atomicity), RC-5 (malformed-fails), RC-6 (merge-preserves), RC-8 (overwrite-required), and RC-9 (round-trip). RC-7 (TTY/prompt) is the control whose acceptance scenario is currently the open gap (checklist IV / analyze K5).
