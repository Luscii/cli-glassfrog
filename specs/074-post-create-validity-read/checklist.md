# Checklist: Post-Create Validity Read

**Feature**: 074-post-create-validity-read
**Checked against**: CONSTITUTION.md (12 principles)
**Artifacts checked**: spec.md, plan.md, interface-cli.md, interface-spec.md, tasks.md, features/success-reported-for-a-dead-proposal/post-create-validity-read.feature
**Checks**: 18 (18 pass, 0 fail) + 1 P2 consideration
**Generated**: 2026-08-08 (round 2 — re-derived after the machine-format and scenario-coverage fixes)

> Source note: no `accords/governance/done-*.md` accords are deployed in this repo, so this checklist runs **constitution checks only** — the same standing as sibling specs 024/028/029/030/071/072. Done-criteria checks are skipped, not failed.

> Calibration note: this feature adds one server exchange and one render path to an existing write command. Eleven principles were calibrated to that shape; **VI (Size-Aware by Design)** produced zero applicable checks (see Governance Notes). Principle I required the most care: the feature reads two fields the vendored contract does not declare, which is a live tension with Spec Fidelity — resolved by CH-3, which tests whether the design survives those fields disappearing.

---

## Summary

| Severity | Count | Pass | Fail |
|---|---|---|---|
| P0 (blocking) | 18 | 18 | 0 |
| P1 (should fix) | 0 | 0 | 0 |
| P2 (consider) | 1 | — | — |
| **Total** | **18 checks** | **18** | **0** |

Every constitution principle in this repo is phrased as a MUST or MUST NOT, so mechanical severity inheritance puts all checks at P0. Severity here reflects the source's own priority, not per-finding impact assessment.

---

## Changes Since Previous Run

**Previous** (round 1): 2 P0 fail, 2 P2 considerations
**Current** (round 2): 0 P0 fail, 1 P2 consideration

**Resolved**:

- ~~**CH-4** | P0 | Principle II — in a machine format, `not reported` and `unavailable` both emitted a document with no `valid` key, so an agent parsing stdout could not tell "the server declined to state a verdict" from "the CLI never managed to ask." The only disambiguator was a prose stderr line, contradicting the spec's own accord ("without scraping human text") and its validation scenario.~~ → **fixed, and not by the route round 1 recommended.** Round 1's recommendation was to narrow the accord's claim to match the design. That was rejected on review: the accord stated the intent, and narrowing it would have preserved the gap the accord exists to close. The design was widened instead — the advisory is now **format-aware**, following the landed convention (032) that a diagnostic renders structurally when a machine format is selected. `verdict_source.read_back` answers the question the emitted document cannot ("did the CLI manage to ask?"), so all four states are machine-distinguishable with no prose parsing, and 018's verbatim contract is untouched because the reshaped document is the CLI's own diagnostic, never the server's. Verified against the four-case table now pinned in `interface-cli.md` and asserted by T006's criteria. ADR-5 (amended), the accord, the validation scenario, the feature file, `interface-cli.md`, `interface-spec.md`, tasks T005/T006, and the DECISIONS entry all moved together.
- ~~**CH-9** | P0 | Principle IV — the accord bullet for a valid draft carrying alerts had no acceptance scenario.~~ → fixed; *"A valid draft carrying an advisory alert reports both facts"* added under Rule 1, with T003 gaining the matching render criterion and T007's disposition table updated.
- ~~**P2-2** — the unavailable advisory named its cause but not its remedy.~~ → fixed; the read-back-failed advisory now names `glassfrog proposal get <prp_id>` in both renderings. The one case with no remedy (no id determinable) deliberately names none rather than pointing at a command the caller cannot run — recorded in the accord so a later reader does not "complete" the table by inventing one.

**Carried forward**: P2-1 (an invalid create still exits 0) — unchanged and still correct as scoped. See P2 Considerations and `risk.md` H-4.

**Not re-drawn**: round 1's calibrations are carried forward unchanged, so round 2 is measured against the same bar rather than a re-drawn one.

---

## Passing Checks

### Principle II — Action Transparency (2/2) — both now pass

- **CH-4** | Every action the command performs reports the operation invoked and the target resource in machine-parseable form, and the four verdict states are distinguishable in a machine format without reading human text. **PASS (round 2).** A stated verdict rides the emitted document structurally; the advisory rides stderr rendered in the selected format, carrying `read_back` (did the CLI ask?), `proposal_id`, and — when unavailable — `reason` and `remedy`. The four states resolve from `data.valid` plus `verdict_source.read_back` with no prose. The reshaped document is the CLI's own diagnostic; the server's document is still passed through byte-for-byte, so this satisfies II without spending 018.
- **CH-5** | Every error the command reports names a cause and a next step. PASS — the create's failure path is unchanged and routes through the landed diagnostic normalization, which supplies both. The read-back's non-fatal advisory is not an error message (the command exits 0), but it now names cause *and* remedy anyway, which is strictly better than the principle requires.

### Principle IV — Test-Driven Development (2/2) — both now pass

- **CH-9** | Every user-facing behavior stated in the spec's Behavioral Accord has a corresponding acceptance scenario. **PASS (round 2).** The valid-draft-with-alerts bullet now has *"A valid draft carrying an advisory alert reports both facts."* Re-walked all fourteen accord bullets against the nineteen scenarios: every bullet has at least one, and the newly-strengthened machine-format bullet has two (the structured-advisory scenario plus the held validation scenario).
- **CH-10** | Tasks sequence tests with implementation rather than after it. PASS — every task's acceptance criteria are assertions rather than descriptions, T001/T002 are test-first by construction, and T007 explicitly reconciles the tests whose expectations the behavior change invalidates.

### Principle I — Spec Fidelity (3/3)

- **CH-1** | Every request the feature issues maps to a published v5 operation. PASS — `POST /proposals` (`createProposal`, unchanged) and `GET /proposals/{id}` (`getProposal`, published and already exercised by 056). No new endpoint, no undefined parameter; the read-back sends no query parameters at all.
- **CH-2** | Neither undeclared field is presented as contract-defined. PASS — spec Non-Behavior 3 forbids it, `interface-spec.md`'s field comments mark both as undeclared and observed, and a validation scenario asserts no such claim appears in help, output, or documentation.
- **CH-3** | The feature does not *rely* on undeclared behavior: if `valid` and `validation_alerts` disappear upstream, it degrades rather than breaks. PASS — the tri-state design makes absence a first-class reported state ("not reported by the server"), T001 asserts the nil decode, and no code path errors or fabricates a verdict when the fields are gone. This is the check that resolves the Spec-Fidelity tension: the CLI *reports* undeclared data without *depending* on it.

### Principle III — Fail Safe, Not Silent (3/3)

- **CH-6** | No error is swallowed. PASS — the read-back's failure is deliberately non-fatal, but it is never silent: it surfaces in the rendered `Validity` line, in the stderr advisory, and with a named cause. T004's criteria require a distinct reason per failure class.
- **CH-7** | No failure condition is reported as success. PASS — an unfavourable verdict is reported *as unfavourable* in every format; nothing claims the draft is fine. That the exit code stays 0 is a scoped, named boundary rather than a silent pass, raised as P2-1.
- **CH-8** | Governance is never left partially applied. PASS — the feature adds a read, not a second write. No new mutation, so no partial-application path exists.

### Principle V — Composition over Monolith (1/1)

- **CH-11** | Adding this feature requires no change to unrelated commands. PASS — and the design chose the option that guarantees it: ADR-4 rejected conditional lines in the shared template precisely because they would have altered `proposal get`, `propose`, and `withdraw`. T003 pins `proposal`-keyed output byte-identical, so the guarantee is enforced by test rather than by intent.

### Principle VII — Working Software (1/1)

- **CH-12** | Every task lands implementation with its tests and requires a clean build and lint. PASS — T001 requires `gofmt -l .` clean (adding struct fields re-aligns sibling tags, and that re-alignment is required in the same commit), and T007 requires `go test ./...` and `gofmt -l .` both clean.

### Principle VIII — No Fabricated Data (1/1)

- **CH-13** | No output value is synthesized, defaulted, or guessed. PASS — this is the principle the feature is built around. An absent `valid` is never defaulted in either direction; the machine path emits server bytes it never reshapes; alerts render the server's own message, path, and severity; and the unavailable case explicitly carries no alerts because none were stated.

### Principle IX — Writes Require Explicit Intent (1/1)

- **CH-14** | No mutation occurs outside an explicit write command, and no read-shaped command mutates. PASS — the feature adds a *read inside a write command*, which is the safe direction. No read-shaped command gains any new behavior, and no new write is introduced anywhere.

### Principle X — Respect API Limits (2/2)

- **CH-15** | Rate limits are honored. PASS — the read-back is a GET and inherits the landed safe-method retry policy; the POST remains never-retried, so the doubled request count cannot double-submit a proposal. A post-retry 429 on the read-back is a reported reason, and one scenario covers it.
- **CH-16** | No update omits `If-Match` where an ETag is available. PASS — the feature issues no update, so no obligation arises. The read-back's captured ETag is discarded exactly as every sibling read discards its own; `propose` and `withdraw` likewise send no `If-Match`, so this is consistency, not neglect.

### Principle XI — Governance via Proposals (1/1)

- **CH-17** | No command path mutates governance structure directly. PASS — no governance mutation is added, and the gated `proposal create` leaf is unchanged. The plan explicitly records that the write-safety guardrail's contract facts are untouched because no command surface changed.

### Principle XII — Standalone Executable (1/1)

- **CH-18** | No new external dependency is introduced. PASS — the two new templates ride the existing `go:embed` bundle, and the new code uses only packages already imported. The distributed artifact still needs nothing but network access.

---

## P2 Considerations

**P2-1 | An invalid create still exits 0.** A machine caller that gates on exit codes cannot distinguish a live proposal from a dead one; only a caller reading stdout can. This is stated scope, not an oversight — spec Non-Behavior 2, ADR-2, and plan Risk 5 all name it, and the exit-code change is the dependent Invalid-Create Outcome capability (backlog #76), which touches both registry sites. Worth confirming the sibling follows closely: shipping this alone solves the human half of the problem and leaves the automation half open.

*(P2-2 from round 1 — the advisory naming its cause but not its remedy — is resolved. See Changes Since Previous Run.)*

---

## Governance Notes

**Principles producing zero applicable checks**:
- **VI (Size-Aware by Design)** — the feature issues one single-resource GET. It reads no list, requests no page, and cannot truncate anything. The one place pagination *could* have entered was rejected on other grounds: `interface-cli.md` records that the verdict is deliberately absent from `proposal list` because carrying it there would require a detail read per row. No truncation surface exists, so no check applies.

**Missing done-criteria accords**: `accords/governance/` is not deployed in this repo. Consider creating `done-specify.md`, `done-plan.md`, `done-interface.md`, `done-scenarios.md`, and `done-tasks.md` to enable done-criteria checks — the pipeline skills already self-verify against these criteria internally, so the accords would make the bar externally checkable rather than skill-internal. This is a repo-wide governance gap, not a finding against this feature.

**Structural mismatches**: none. All 12 principles carry a detection mechanism, which is what made mechanical evaluation possible here.
