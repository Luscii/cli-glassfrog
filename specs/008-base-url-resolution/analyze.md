# Analyze: Base URL Resolution

**Feature**: 008-base-url-resolution
**Artifacts analyzed**: spec.md, plan.md, interface-spec.md, features/undefined-connection-settings/base-url-resolution.feature, tasks.md
**Checklist context**: checklist.md present (9/9 pass) — correlated, not re-evaluated
**Findings**: 16 checks (16 pass, 0 open: 0 P0, 0 P1, 0 P2 — 2 findings raised and resolved in this pass)
**Generated**: 2026-06-04

---

## Summary

| Severity | Checks | Pass | Open findings |
|---|---|---|---|
| P0 (consistency / contradiction) | 6 | 6 | 0 |
| P1 (completeness / gap) | 6 | 6 | 0 |
| P2 (coherence / drift) | 4 | 4 | 0 |
| **Total** | **16** | **16** | **0** |

No contradictions. Two findings were raised on the first horizontal pass — one acceptance-coverage gap (K5, P1) and one terminology drift (H1, P2) — and both were resolved in this same pass (feature scenario added; wording corrected). Details retained below for transparency. No open findings; nothing blocks implementation.

---

## Consistency Checks (P0): 6/6 passed

All pass — no artifact pair makes incompatible claims:
- **C1** spec Integration Boundaries ↔ plan System Architecture: the plan's parts (internal/auth reuse, `internal/apiclient` resolver, flag/env/file/default, spec/glassfrog-api-v5.yaml default, connection-context downstream) cover every named boundary. **pass**
- **C2** spec Behavioral Accord ↔ plan System Architecture: the resolver serves the precedence/validation/always-a-value behaviors; none contradicted. **pass**
- **C3** spec Non-Behaviors ↔ plan System Architecture: the plan architects none of the excluded capabilities (no token resolution, no API call, no writes, no exit code, no normalization, no multi-endpoint) — ADR-1/ADR-3/ADR-4 keep them out. **pass**
- **C4** plan Architecture Decisions ↔ interface-spec Surface: the `BaseURL{Value, Source, Path}` shape, `Source` enum, `http(s)` validation, typed `BaseURLError`, and precedence in the accord reflect the plan's ADRs. **pass**
- **C5** plan System Architecture ↔ tasks Task Scope: T001/T002/T003 build exactly the plan's three parts; no task builds something the plan doesn't mention. **pass**
- **C6** interface-spec Surface ↔ feature Given/When/Then: every scenario step references a surface the accord defines (flag, `GLASSFROG_BASE_URL`, `.glassfrogrc` `base_url`, built-in default, source reporting, format/read errors). **pass**

---

## Completeness Checks (P1): 6/6 passed

### Finding (raised then resolved in this pass)

**P1 — RESOLVED** | K5: interface-spec.md § Error Communication → features/…/base-url-resolution.feature (Given/When/Then)
→ Originally: **two error surfaces the accord defines had no acceptance scenario** — (1) a **malformed `base_url` *value* in a file**, and (2) an **unparseable `.glassfrogrc`**. **Resolution:** the feature now includes "A malformed config-file value fails loudly naming the file" (a concretization of the spec's "A source supplies a malformed base URL"), closing surface (1) at the acceptance layer; tasks T003 enumerates it. Surface (2), an unparseable `.glassfrogrc`, is the shared 005 reader's `*FormatError` — already covered by the `internal/auth` reader's own tests (005), so a base-URL-specific acceptance scenario adds no coverage; acceptance coverage of the error surfaces is now adequate. K5 **passes**.

### Passed without a finding (K1–K4, K6)

- **K1** spec Driving Scenarios → feature: all 8 driving + 4 validation scenarios have Gherkin equivalents. **pass**
- **K2** spec Integration Boundaries → interface presence: the `.glassfrogrc`/config-input/`BaseURL` boundaries are covered by interface-spec.md; downstream (connection-context) and exit-code boundaries are justified-absence (deferred/downstream, stated in spec). **pass**
- **K3** plan Phases → tasks: 3 phases → 3 tasks (T001/T002/T003). **pass**
- **K4** plan Components → tasks Scope: each component (base_url file read / resolver+precedence+validation / godog acceptance) has an implementing task. **pass**
- **K6** spec User Scenarios → interface: all three user flows (flag override, default out-of-box, project-local precedence) have interface coverage. **pass**

---

## Coherence Checks (P2): 4/4 passed

### Finding (raised then resolved in this pass)

**P2 — RESOLVED** | H1: terminology drift across the artifact set
→ Originally: feature steps called the single `.glassfrogrc` a "**credentials file**" (inherited from 005's token vocabulary) while the rest of 008's artifacts say "config file" / ".glassfrogrc". **Resolution:** both occurrences were standardized — the empty-env step now reads "from the **current directory's file**" (matching the nearest-wins phrasing) and the env-wins step now reads "it will not read any **config file**". No terminology drift remains. H1 **passes**.

### Passed without a finding (H2–H4)

- **H2** detail symmetry (spec↔plan, plan↔tasks): proportionate; no artifact carries 3x+ detail on a shared topic. **pass**
- **H3** scope alignment (spec/interface/tasks): the same feature scope throughout; the deferred `--base-url` flag registration and the deferred connection-context half are stated consistently in all three; the proposed feature scenario adds coverage, not a capability. **pass**
- **H4** phase coverage (plan↔tasks): tasks' three phases map 1:1 to the plan's three phases with matching linear dependencies. **pass**

---

## Checklist Correlation

checklist.md (9/9 pass) carried two non-blocking notes; neither overlaps an analyze finding, but for the record:
- Checklist's **derived-default-host** note (Spec Fidelity nuance) is orthogonal to both analyze findings.
- Checklist's **shared-`parseCredentials` edit** coupling note relates to scope/coherence; analyze's H3 (scope alignment) found no silent capability drift from it — the extension is stated in plan ADR-3 and tasks T001. No contradiction.

---

## Governance Notes

- No checks were skipped — all six artifact types are present, so the full 16-check matrix ran.
- (Vertical done-criteria coverage is checklist's domain; analyze does not re-check it. The project-wide absence of `accords/` is recorded in checklist.md.)
