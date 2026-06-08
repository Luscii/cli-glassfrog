# Checklist: Structured Serialization

**Feature**: 018-structured-serialization
**Checked against**: CONSTITUTION.md (12 principles). No `accords/governance/done-*.md` accords present — done-criteria and cross-reference checks not generated (see Governance Infrastructure Notes).
**Artifacts checked**: spec.md, plan.md, interface-spec.md, features/unconsumable-output/structured-serialization.feature, tasks.md
**Checks**: 14 (12 pass, 2 fail)
**Generated**: 2026-06-08

---

## Summary

| Severity | Count | Pass | Fail |
|---|---|---|---|
| P0 (blocking) | 10 | 10 | 0 |
| P1 (should fix) | 0 | 0 | 0 |
| P2 (consider) | 4 | 2 | 2 |
| **Total** | **14** | **12** | **2** |

No P0 or P1 failures. **Principle II (Action Transparency) — the principle this feature exists to satisfy — passes**: 018 delivers the machine-parseable structured output II's detection requires ("output that only emits free-form human text with no structured form … fails review"), realizing the `--output json/yaml` mode every `/me*` read so far deferred. **Principle XII (Standalone Executable) passes** despite the new `sigs.k8s.io/yaml` dependency — a statically-linked Go module is compiled into the binary, not a runtime install. Two P2 advisories remain with **distinct owners**: (1) the error envelope's human-readable "next step" deepens once Output Format Selection (020) wires 015's (now-landed) `*ProblemError` detail into the envelope; (2) end-to-end user-facing `--output` acceptance also lands with 020 — 018's behavior is scenario-covered at the component grain. Neither re-shapes the spec or blocks implementation.

---

## Constitution Checks: 12/14 passed

### Calibration notes

Three broad MUST principles were calibrated to this feature before evaluation:
- **II (Action Transparency, NON-NEGOTIABLE)** → "the serialized output is machine-parseable (JSON/YAML); the error envelope carries a cause (message) and a discriminable category (kind, status); the *operation/target* legibility of any given command remains that command's concern (020), not 018's." Multi-clause → multiple checks.
- **III (Fail Safe)** → "a serialization/render failure is surfaced loud (a typed render error → RuntimeError), never swallowed and never emitted as a partial or invalid document."
- **V (Composition)** → "`internal/output` is a standalone, independently-testable leaf imported one-directionally; 018 edits no existing command or package (not even 010)."

### Failures

**P2** | CONSTITUTION.md II (Action Transparency): "every error MUST explain what went wrong and the next step"
→ **interface-spec.md § Error Communication** / **plan.md § ADR-4**: The unified error envelope carries `message` (the cause), `kind`, and `status` — a discriminable category an agent can act on (e.g. `kind=usage` ⇒ fix invocation). A richer human-readable **next step** in the message now exists in **API Error Extraction (015, landed #44)** as `*ProblemError` detail; surfacing it into the envelope is **Output Format Selection (020)**'s mapping, since 018 owns the envelope *shape* (incl. the `body`/`message` slot), not the typed-error→envelope population. The cause is present today and the kind/status give the operator an actionable signal, so transparency is served — the next-step *richness* awaits 020's wiring, recorded for traceability, **not a 018 violation**.

**P2** | CONSTITUTION.md IV (Test-Driven Development): "user-facing behavior MUST have an executable acceptance scenario before the code that satisfies it"
→ **tasks.md § T003** / **plan.md § Cross-cutting (testing)**: 018 has **no CLI surface** (no `--output` flag, no command — that is 020), so its acceptance scenarios run at the **component level** over the `internal/output` functions (T003 godog suite), RED-first, before the code — IV *is* satisfied at that grain. The end-to-end *user-facing* grain (running `glassfrog … --output json` and observing the document) is only reachable once 020 wires the flag and routing, so that acceptance lands with **020**. Recorded for traceability — the behavior is scenario-covered before code; only the e2e grain is deferred. **Not a violation.**

### Passed (10 P0 + 2 P2)

- **P0 | II Action Transparency (machine-parseable form)** — 018 *is* the structured-output capability II's detection demands: it renders a command's result as machine-readable JSON/YAML, the literal "structured form" whose absence II says "fails review". The error path is structured too (the unified envelope), so failures are parseable, not free-form text. (spec.md System Overview + Behavioral Accord, interface-spec.md Surface, feature: "A successful payload renders as a JSON document")
- **P0 | I Spec Fidelity** — 018 invents no endpoint, parameter, or behavior; it makes no API call. It serializes the raw API payload **verbatim** (the maximally faithful surface — no reshaping, no field loss) and the raw API error body verbatim. The error envelope is the CLI's own error *representation* (II's domain), not a claimed API behavior. (spec.md Non-Behaviors, plan.md ADR-2)
- **P0 | III Fail Safe, Not Silent** — a render failure (e.g. a 2xx body that is not valid JSON) is surfaced as a typed render error → `RuntimeError`, never swallowed; the renderer builds the whole document in memory so it never emits a partial/invalid fragment (explicit Non-Behavior + ADR-2 + task risk). No writes, so no partial-apply. (spec.md Non-Behaviors, plan.md Cross-cutting, feature: "An invalid success body surfaces a render error, not a partial document")
- **P0 | IV Test-Driven Development** — T001/T002 are RED-first unit tests; the acceptance scenarios exist (`structured-serialization.feature`) before the code; T003 makes the behavioral scenarios executable over the package. (tasks.md T001–T003, feature file)
- **P0 | V Composition over Monolith** — `internal/output` is a new pure leaf, independently testable, imported one-directionally by `cli`; 018 edits **no** existing command and not even 010 (it captures raw bytes via a `json.RawMessage` target the existing `Execute` already accepts). Adding it forces no change to unrelated commands. (plan.md ADR-1/ADR-2, tasks.md — additive)
- **P0 | VII Working Software** — each task pairs implementation with tests and requires `go build`/`go vet` clean; 018 is a no-dependency root, so no task builds against non-compiling sibling code. (tasks.md acceptance criteria, dependency graph)
- **P0 | VIII No Fabricated Data** — 018 emits the raw API payload verbatim (the least-fabricated possible output) and never reshapes or defaults a success field. The envelope's synthesized fields (message/kind for a bodiless failure) are the CLI's own *error metadata*, clearly not API governance data presented as real — the same distinction as 017's diagnostic note. (spec.md Non-Behaviors, interface-spec.md, plan.md ADR-2/ADR-4)
- **P0 | IX Writes Require Explicit Intent** — N/A by construction, recorded as passed: 018 is pure serialization with no API calls, no command, and no mutation/side effects. (spec.md System Overview — a transformation)
- **P0 | XI Governance via Proposals** — N/A by construction, recorded as passed: 018 has no command surface and mutates no governance structure. (spec.md — no command surface)
- **P0 | XII Standalone Executable** — passes despite the new `sigs.k8s.io/yaml` dependency: XII's detection targets *runtime installs on the user's machine* ("running the distributed artifact on a clean environment … succeeds"). A Go module is statically linked into the binary, so a clean host + network still runs it with nothing pre-installed. ADR-3 and the DECISIONS Go-self-contained precedent address this explicitly. (plan.md ADR-3, .score/memory/DECISIONS.md 018 entry)
- **P2 | VI Size-Aware by Design** — passes / N/A-leaning: 018 neither paginates nor caps (that is Pagination, 016) and **never truncates** — it emits the full payload verbatim. Notably, structured success output carries the raw `meta.pagination` (incl. `has_next_page`) verbatim, so an agent parsing the document sees incompleteness directly — truncation is never silent in machine mode. (spec.md Non-Behaviors, plan.md "Does Not Cover")
- **P2 | X Respect API Limits** — N/A by construction, recorded as passed: 018 makes no API request and performs no update, so neither the `429`-backoff clause (017's) nor the `If-Match`/`ETag` clause (a future write spec's) applies. (plan.md Integration Design — no direct API contact)

---

## Governance Infrastructure Notes

*(separate from feature quality findings)*

- **No `accords/governance/done-*.md` accords exist.** Done-criteria and cross-reference checks were not generated — this checklist is constitution-only (same state as 010/011/017). Consider creating `done-specify.md`, `done-plan.md`, `done-interface.md`, `done-scenarios.md`, `done-tasks.md` to enable done-criteria gating and tasks↔scenarios↔interface link checks in future runs. Their absence is a tooling gap, not a feature defect.
- **Principles IX, X, XI**: produced N/A-by-construction results for this feature (no writes, no API requests, no command/governance surface). Recorded as passed-by-construction, not dropped.
- **Principle X, `If-Match`/`ETag` clause**: no applicable check — concerns updates; 018 issues no requests at all.

---

## Notes for the developer

- **Principle II passes — this is the headline.** 018 delivers the machine-parseable structured output the constitution's NON-NEGOTIABLE transparency principle requires, closing the `--output json/yaml` deferral 011/014 carried.
- **The dependency question (XII) is settled**: `sigs.k8s.io/yaml` compiles into the self-contained binary; no runtime install. Record it in PROJECT.md's stack (plan handoff).
- The two P2 advisories have **distinct owners**, neither a MUST violation, both recorded for traceability:
  - **(1) error-envelope "next step" richness** → **Output Format Selection (020)** wires 015's (landed) `*ProblemError` detail into the envelope; the envelope already carries message + kind + status and the slot for it.
  - **(2) end-to-end `--output` acceptance** → lands with **Output Format Selection (020)**; 018's behavior is scenario-covered at the component grain (T003).
- No finding blocks implementation.
