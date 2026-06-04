# Checklist: Credential Storage

**Feature**: 006-credential-storage
**Checked against**: CONSTITUTION.md (no `accords/governance/done-*.md` present)
**Artifacts checked**: spec.md, plan.md, interface-cli.md, interface-spec.md, features/unauthenticated-access/credential-storage.feature, tasks.md
**Checks**: 10 (10 pass — 2 P1 raised in PR #12 review, resolved in this PR)
**Generated**: 2026-06-04 (updated after PR #12 review)

---

## Summary

10 checks: **10 pass, 0 open findings**. Constitution: 10/10. Done-criteria: not run (no accords). Cross-references: not run (no accords).

8 of 12 constitution principles produced applicable checks; 4 are N/A for this local, no-network credential writer (see Governance Notes). Unlike 005 (no operator-facing surface), 006 adds a command, so Action Transparency (II) is now applicable.

_Two P1 items were raised during PR #12 review and resolved in this PR — see **Resolved during review** below._

---

## Constitution Checks: 10/10 passed

### Resolved during review (2)

**P1 → resolved** | CONSTITUTION.md II (Action Transparency, NON-NEGOTIABLE): "every error MUST explain what went wrong **and the next step**"
→ **interface-cli.md § Error Communication** now pins a concrete next step in every error message (no-token → "supply a token via argument, stdin, or `GLASSFROG_TOKEN`"; write-error → "check write permission on the directory"; format-error → "fix or remove the malformed `.glassfrogrc`"; existing-token → "pass `--overwrite` to replace it"), and uses the canonical `Success`/`UsageError`/`RuntimeError` category names. The token value still never appears. *(PR #12, Copilot review.)*

**P1 → resolved** | CONSTITUTION.md IV (TDD): "user-facing behavior MUST have an executable acceptance scenario before the code"
→ the two previously-uncovered interactive behaviors now have driving scenarios in **spec.md § Driving Scenarios** ("Interactive prompt for a missing token", "Interactive confirmation chooses the write location") and matching `@wip` acceptance scenarios in **features/unauthenticated-access/credential-storage.feature**; the merge scenario now states its interactive (TTY) precondition explicitly. The interactive prompt and the existing-token confirmation (with the cwd/home/both location choice) are acceptance-covered. *(PR #12, Copilot review — correlates with analyze K5, also resolved.)*

### Passed (10/10)

**P0** | CONSTITUTION.md II (Action Transparency) — *report what was done + target, never the secret*
→ **interface-cli.md § Surface/Output**, **spec.md § Behavioral Accord > Writing**: a successful store reports the written path ("Stored credentials in `<path>`") and never the token; the target resource (the file) is always named. Every error names both the cause and a next step (resolved during review — see above).

**P0** | CONSTITUTION.md III (Fail Safe, Not Silent): "MUST validate a write before sending … MUST NOT leave [state] partially-applied"; anti-pattern "a failure condition reported as success"
→ **plan.md § ADR-4**, **interface-spec.md § Error Communication**, **feature scenarios**: the existing file is parse-validated before any write (malformed → format error, **no write**); the write is atomic (temp-in-same-dir + `rename`) so a mid-write failure leaves the original/absence intact; an unwritable target reports a write error with the filesystem unchanged. Pinned by "A malformed existing file fails the store loudly" and "An unwritable target fails the store loudly".

**P0** | CONSTITUTION.md IV (TDD): "Features MUST be built test-first (RED → GREEN)"
→ **tasks.md § T001–T005**: each mandates RED-first unit tests (writer: create/merge/malformed/atomic/perms/round-trip; resolution: precedence/blank/no-token/guard) before implementation, and the behavioral scenarios — both non-interactive and interactive (prompt, confirm-and-choose-location) — exist as `@wip` acceptance before code (T005 makes them executable; the 3 `@validation` scenarios stay held out).

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
