# Risk: Write-Safety Guardrail

**Feature**: 063-write-safety-guardrail
**Round**: 1
**Generated**: 2026-06-16
**Artifacts loaded**: spec.md, plan.md, interface-spec.md, PROJECT.md
**Acceptability matrix**: default 3×3 traffic-light (no project-level matrix in PROJECT.md)
**Degradation flags**: none — full upstream set present. PROJECT.md declares no Regulatory Context, so no IEC 14971 bridge is included.

> Domain note: this is a **governance-integrity guardrail**. The dominant hazard class is a *false negative* — a governance write that reaches the API without the required human confirmation. False positives (gating a read) are friction, not integrity loss. Severity ratings weight integrity over availability accordingly.

---

## Risk Register

| H-ID | Hazard | Source | Sev | Prob | Pre-control | Controls | Residual |
|---|---|---|---|---|---|---|---|
| H-1 | Recognizer fails to identify a governance write (chaining, alias, env-prefix, quoting, wrapper) → ungated write reaches the API | plan R1 / ADR-3; interface Interactions | High | Med | Red | RC-1, RC-2, RC-3 | **Yellow** |
| H-2 | Hook fails *open* on internal error or malformed stdin → identified proposal write proceeds ungated | interface Error Communication; plan R1 | High | Low | Yellow | RC-2, RC-4 | **Yellow** |
| H-3 | Plugin-host hook contract drifts (PreToolUse schema / `permissionDecision` protocol changes) → hook silently stops gating | plan R2; interface Consistency Notes | High | Med | Red | RC-5 | **Yellow** |
| H-4 | Enforcement absent — plugin/hook not installed or host lacks PreToolUse → writes fall back to ungated guidance-only | plan R3; spec Integration Boundaries | High | Med | Red | RC-6, RC-7 | **Yellow** |
| H-5 | Gated-command registry drifts from the CLI's `proposal` surface → new/renamed write leaf ships ungated | plan ADR-4 / R4 | High | Med | Red | RC-8, RC-2 | **Green** |
| H-6 | Blind retry after a `412` stale-write clobbers a concurrent change | spec § stale-write accord; plan ADR-5 | High | Low | Yellow | RC-9, RC-10 | **Green** |
| H-7 | False-positive gating of reads / tension edits → agent friction, pressure to disable the hook | spec § Non-Behaviors | Med | Low | Green | RC-1, RC-11 | **Green** |
| H-8 | Human rubber-stamps the confirmation → gate becomes ceremonial, an unintended write is approved | spec § User Scenarios (human-in-loop); plan ADR-2 | Med | Med | Yellow | RC-12 | **Yellow** |

No residual risk is **Red**. Four High-severity hazards (H-1, H-3, H-4 + the registry path H-5) remain **Yellow** (acceptable with documented justification) because their full mitigation depends on factors partly outside the artifact (exotic command forms, an external host contract, installation, human attention).

---

## Hazard Detail

**H-1 — Recognizer evasion / mis-parse.** *Severity High*: an unrecognized governance write reaches the API with no confirmation — the exact integrity breach the guardrail exists to prevent (an unapproved governance change). *Probability Medium*: command strings vary (chaining `&&`/`;`, pipes, aliases, absolute paths, `VAR=val` prefixes, wrappers). Controls: **RC-1** resolve the `glassfrog` token and subcommand path rather than substring-matching (interface Interactions); **RC-2** fail-closed within the `proposal` namespace so an unrecognized `proposal` subcommand is gated by default (plan ADR-3); **RC-3** recognizer unit tests over command-string variants (tasks T002). Residual **Yellow** — exotic-invocation evasion is explicitly accepted and noted (plan R1), not assumed away.

**H-2 — Fail-open on error.** *Severity High* (ungated write), *Probability Low* (errors are infrequent). Controls: **RC-2** fail-closed for identified proposal writes; **RC-4** a `glassfrog proposal` write that can't be fully parsed errs to `ask`, never to silent allow (interface Error Communication). Residual **Yellow** — the script must never block *unrelated* Bash on error, so a narrow fail-open window for genuinely unparseable non-proposal commands is accepted by design.

**H-3 — Host contract drift.** *Severity High* (silent loss of enforcement), *Probability Medium* (the Claude Code host evolves). Control: **RC-5** pin the registration schema and decision protocol against the host's documented format at implementation, and treat it as an external contract to revisit (plan R2). Residual **Yellow** — an external dependency the feature cannot fully control; partial by nature.

**H-4 — Enforcement absent.** *Severity High* (no gate at all), *Probability Medium* (depends on install + host capability). Controls: **RC-6** documented fallback to 062's guidance-only behavior with nothing in the CLI broken (spec Integration Boundaries); **RC-7** distribution that installs the hook (Operating-Surface Packaging #70). Residual **Yellow, accepted** — the guardrail strengthens a *present* host and cannot retrofit absent infrastructure; this is a known architectural boundary, not a defect. *This is the dominant accepted residual — worth the developer's explicit acknowledgment.*

**H-5 — Registry/CLI drift.** *Severity High*, *Probability Medium* pre-control. Controls: **RC-8** the best-effort `internal/build` drift tripwire fails the build when the `proposal` subcommand surface changes without the registry (plan ADR-4, tasks T003); **RC-2** fail-closed within the `proposal` namespace catches an unregistered new leaf at runtime by default. Residual **Green** — the two controls compound (build-time tripwire + runtime fail-closed).

**H-6 — Blind-retry clobber.** *Severity High*, *Probability Low*. Controls: **RC-9** the retry is itself a gated proposal write requiring fresh confirmation (plan ADR-5); **RC-10** 062's orientation guidance to re-read for the current version before retrying. Residual **Green**.

**H-7 — False-positive gating.** *Severity Medium* (friction, possible pressure to disable), *Probability Low*. Controls: **RC-1** precise recognition; **RC-11** the registry lists only the four proposal-write leaves, so reads and tension edits are never matched. Residual **Green**.

**H-8 — Rubber-stamped confirmation.** *Severity Medium* (a defeated gate approves an unintended write), *Probability Medium* (confirmation fatigue is real). Control: **RC-12** the `ask` `systemMessage` names the command, target id, and effect so the human can review meaningfully rather than approving a bare prompt (interface Surface). Residual **Yellow** — human attention is partly outside the system's control; the legible message is the system-side control.

---

## Controls Index

| RC-ID | Control (assessment level) | Mitigates |
|---|---|---|
| RC-1 | Resolve the `glassfrog` token + subcommand path (not substring matching) | H-1, H-7 |
| RC-2 | Fail-closed within the `proposal` namespace (unrecognized `proposal` subcommand → gate) | H-1, H-2, H-5 |
| RC-3 | Recognizer unit tests over command-string variants | H-1 |
| RC-4 | On parse failure of an identified `glassfrog proposal` command, err to `ask` (never silent allow) | H-2 |
| RC-5 | Pin the host hook contract at implementation; treat as external contract to revisit | H-3 |
| RC-6 | Documented fallback to 062 guidance-only with no CLI breakage when the hook is absent | H-4 |
| RC-7 | Distribution installs the hook (#70 Operating-Surface Packaging) | H-4 |
| RC-8 | Best-effort `internal/build` drift tripwire over the CLI `proposal` surface | H-5 |
| RC-9 | Retry after `412` is itself a re-gated write requiring fresh confirmation | H-6 |
| RC-10 | 062 orientation guidance: re-read for current version before retry | H-6 |
| RC-11 | Registry lists only the four proposal-write leaves | H-7 |
| RC-12 | Confirmation `systemMessage` names command, target id, and effect | H-8 |

## Traceability Index

- **H-1, H-2** → plan.md § Risks (R1), § ADR-3; interface-spec.md § Interactions, § Error Communication
- **H-3** → plan.md § Risks (R2); interface-spec.md § Consistency Notes
- **H-4** → plan.md § Risks (R3); spec.md § Integration Boundaries
- **H-5** → plan.md § ADR-4, § Risks (R4); tasks.md T003
- **H-6** → spec.md § Behavioral Accord (stale-write); plan.md § ADR-5
- **H-7** → spec.md § Non-Behaviors
- **H-8** → spec.md § User Scenarios; plan.md § ADR-2
- **RC-1–RC-12** → grounded in plan ADR-2/ADR-3/ADR-4/ADR-5, interface Surface/Interactions/Error Communication, tasks T002/T003

## Residual Risk Summary

8 hazards, 12 controls. **0 Red**, **5 Yellow** (H-1, H-2, H-3, H-4, H-8 — acceptable with the documented justifications above), **3 Green** (H-5, H-6, H-7). The Yellow residuals cluster on factors partly outside the artifact: external command-string forms (H-1/H-2), the external host contract (H-3), installation/host capability (H-4), and human attention (H-8). H-4 (enforcement absent outside a hook-supporting host) is the dominant accepted residual and should be acknowledged explicitly — it is an architectural boundary, not a fixable defect.
