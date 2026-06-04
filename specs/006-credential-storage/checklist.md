# Checklist: Credential Storage

**Feature**: 006-credential-storage
**Checked against**: CONSTITUTION.md (no `accords/governance/done-*.md` present)
**Artifacts checked**: spec.md, plan.md, interface-cli.md, interface-spec.md, features/unauthenticated-access/credential-storage.feature, tasks.md
**Checks**: 10 (8 pass, 2 P1 findings)
**Generated**: 2026-06-04

---

## Summary

10 checks: **8 pass, 2 P1 findings**. Constitution: 8/10. Done-criteria: not run (no accords). Cross-references: not run (no accords).

8 of 12 constitution principles produced applicable checks; 4 are N/A for this local, no-network credential writer (see Governance Notes). Unlike 005 (no operator-facing surface), 006 adds a command, so Action Transparency (II) is now applicable — and is where the single finding sits.

---

## Constitution Checks: 9/10 passed

### Findings

**P1** | CONSTITUTION.md II (Action Transparency, NON-NEGOTIABLE): "every error MUST explain what went wrong **and the next step**"
→ **interface-cli.md § Error Communication**: the error contract names the cause/path for every error, but only the existing-token row states an explicit next step ("`--overwrite` is required"). The `no token to store`, write-error, and format-error rows name the cause but leave the next step implicit.
*Why P1 not P0*: II's cause clause passes for every error (the **what** is always present); the gap is the **next-step** sub-clause, easily pinned in the message strings at implementation. Mechanically II is a MUST — recommend the Builder give each error message an explicit next step (e.g. write-error → "check permissions on `<path>`"; no-token → "supply a token via argument, stdin, or GLASSFROG_TOKEN") and add a step-asserting check to the @validation scenarios.

**P1** | CONSTITUTION.md IV (TDD): "user-facing behavior MUST have an executable acceptance scenario before the code"
→ **spec.md § Behavioral Accord > Token input / Existing credentials**, **interface-cli.md § Interactions**, **features/unauthenticated-access/credential-storage.feature**: two user-facing behaviors have **no** acceptance scenario — the non-echoing **interactive prompt** (TTY, no other source) and the **interactive existing-token confirmation** (offering merge + the working-dir/home/both location choice). All 9 behavioral scenarios cover the non-interactive paths.
*Why P1 not P0*: the primary operator is a non-interactive AI agent and every non-interactive path **is** acceptance-covered; the gap is the interactive (human-at-a-TTY) convenience sub-paths. Mechanically IV is a MUST — recommend either (a) adding godog scenarios that drive the prompt/confirm through the injected `isTTY` seam (T003/T005 already make interactivity injectable), or (b) explicitly deferring interactive-path verification to manual/validate with a noted justification in the spec and tasks. Correlates with analyze K5.

### Passed (8/10)

**P0** | CONSTITUTION.md II (Action Transparency) — *report what was done + target, never the secret*
→ **interface-cli.md § Surface/Output**, **spec.md § Behavioral Accord > Writing**: a successful store reports the written path ("Stored credentials in `<path>`") and never the token; the target resource (the file) is always named. The cause clause of every error is present (see finding for the next-step sub-clause).

**P0** | CONSTITUTION.md III (Fail Safe, Not Silent): "MUST validate a write before sending … MUST NOT leave [state] partially-applied"; anti-pattern "a failure condition reported as success"
→ **plan.md § ADR-4**, **interface-spec.md § Error Communication**, **feature scenarios**: the existing file is parse-validated before any write (malformed → format error, **no write**); the write is atomic (temp-in-same-dir + `rename`) so a mid-write failure leaves the original/absence intact; an unwritable target reports a write error with the filesystem unchanged. Pinned by "A malformed existing file fails the store loudly" and "An unwritable target fails the store loudly".

**P0** | CONSTITUTION.md IV (TDD): "Features MUST be built test-first (RED → GREEN)"
→ **tasks.md § T001–T005**: each mandates RED-first unit tests (writer: create/merge/malformed/atomic/perms/round-trip; resolution: precedence/blank/no-token/guard) before implementation, and the 9 **non-interactive** behavioral scenarios exist as `@wip` acceptance before code (T005 makes them executable; the 3 `@validation` scenarios stay held out). *(The interactive-path acceptance gap is the IV finding above.)*

**P0** | CONSTITUTION.md V (Composition over Monolith): "modular … adding … MUST NOT require changing unrelated ones"
→ **plan.md § ADR-1/ADR-2**: the writer joins `internal/auth` (file concern), the command is a new guard-registered leaf wired explicitly in `main` (no `init()`); adding it edits no existing command module.

**P0** | CONSTITUTION.md VII (Working Software): implementation paired with tests; validate and build
→ **tasks.md § T001–T005**: every task pairs implementation with tests; `go build ./...` and `go vet ./...` clean are explicit acceptance criteria.

**P0** | CONSTITUTION.md VIII (No Fabricated Data) — *applied to the stored credential*
→ **spec.md § edge "Supplied token is blank"**, **feature "A blank token is rejected"**: Storage persists only the token actually supplied; an empty/whitespace value is rejected rather than written as a fabricated credential.

**P0** | CONSTITUTION.md IX (Writes Require Explicit Intent): "No write … except as the direct result of an explicit write command"; conflict-table note "intent is expressed by the explicit write command/flag, not by an interactive prompt"
→ **plan.md § ADR-2/ADR-5**, **interface-cli.md § Interactions**: the file write occurs only via the explicit `auth login` command; in the non-interactive (AI-agent) path, overwrite intent is the explicit `--overwrite` flag (never a blocking prompt), and a missing token errors rather than hanging — directly honoring the constitution's write-intent-via-flag-not-prompt resolution. The interactive prompt/confirm is an additive convenience reserved for a human at a TTY.

**P0** | CONSTITUTION.md XII (Standalone Executable): "no language runtime … no other software … installed first"
→ **plan.md § ADR-1/ADR-4**, **go.mod**: the writer is hand-rolled over the Go standard library (`os`); no new dependency is added (the shared format module is reused from 005). The artifact stays a single self-contained binary.

---

## Governance Notes

- **No `done-*` accords found.** Constitution checks ran; done-criteria and cross-reference checks did not. Consider creating `accords/governance/done-{specify,plan,interface,scenarios,tasks}.md` to enable vertical quality checks for later specs. (Same gap noted for 005.)
- **Principle I (Spec Fidelity)**: N/A — `auth login` manages a local credential and invokes no Glassfrog API operation; the spec-operation binding applies to API-backed resource commands (roles, proposals, …). Credential storage is local CLI infrastructure for authentication, not an API surface.
- **Principle VI (Size-Aware)**: N/A — no API result sets or pagination. Single-token write, no truncation.
- **Principle X (Respect API Limits)**: N/A — no network calls.
- **Principle XI (Governance via Proposals)**: N/A — writing a local credentials file is not a governance-structure mutation; no governance command path exists here.

## Advisory Notes (not severity findings)

- **Secret hygiene has no dedicated principle**: "the token value never appears in output, logs, prompts, or error messages" is a genuine security property pinned by the validation scenario "The stored token value never appears in output" and the no-echo prompt, but it does not trace cleanly to a single constitution principle, so it is not a severity check. Recommend the reviewer confirm prompt/stdin reads and all error strings carry only the path. (Same treatment as 005.)
- **Shared `[ASSUMED]` contract with 005**: `.glassfrogrc` name/format, `GLASSFROG_TOKEN`, and the `0600` mode are provisional and jointly held with Credential Discovery. This is a cross-artifact/coordination concern (analyze's domain) — flagged so it is reconciled before 005 and 006 both ship.
