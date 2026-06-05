# Risk: Base URL Resolution

**Feature**: 008-base-url-resolution
**Round**: 1
**Generated**: 2026-06-04
**Artifacts loaded**: spec.md, plan.md, interface-spec.md, PROJECT.md
**Matrix**: default 3×3 traffic-light — no project-level acceptability matrix found in PROJECT.md
**Degradation flags**: none — spec, plan, and interface all present. PROJECT.md has no Regulatory Context, so no IEC 14971 bridge is included.

---

## Risk Register

| H-ID | Hazard | Source | Severity | Probability | Pre-control | Controls | Residual |
|---|---|---|---|---|---|---|---|
| H-1 | The built-in default endpoint is mis-derived, so every out-of-the-box request targets the wrong host | plan § Integration Design / Risks (derived host) | High | Medium | **Red** | RC-1, RC-2 | **Yellow** |
| H-2 | The token leaks through the shared `.glassfrogrc` reader on a base-URL path | spec § Non-Behaviors; plan ADR-3 | High | Low | Yellow | RC-3 | **Green** |
| H-3 | Generalizing the shared `parseCredentials` regresses 005's token resolution | plan ADR-3; tasks T001; checklist (V coupling note) | High | Low | Yellow | RC-4 | **Green** |
| H-4 | A malformed base URL is accepted/passed through, routing requests to a bad endpoint | spec § Error scenarios; plan ADR-4 | Medium | Low | Green | RC-5 | **Green** |
| H-5 | A broken `.glassfrogrc` is mistaken for absence → silent fallback to the default (wrong endpoint, no signal) | spec § Error handling | Medium | Low | Yellow | RC-6 | **Green** |
| H-6 | The `base_url` key contract drifts from Credential Storage (006), so a stored base URL is never read | plan § Integration Design / Risks (`[ASSUMED]` key) | Medium | Medium | Yellow | RC-7 | **Yellow** |
| H-7 | Silent URL normalization masks a misconfiguration or breaks a deliberate endpoint | spec § Non-Behaviors (no normalization) | Low | Low | Green | RC-8 | **Green** |

No residual **Red**. Two residual **Yellow** (H-1, H-6) — acceptable with the documented justifications below.

---

## Hazard Detail

### H-1 — Mis-derived default endpoint
**Severity: High** — the default backstops the chain, so when nothing is configured *every* request goes to the default host; a wrong host misroutes all out-of-the-box traffic (and, combined with the deferred connection-context half, would send the token there). **Probability: Medium** — `spec/glassfrog-api-v5.yaml` declares a *relative* server `url: /api/v5`, so the host (`https://glassfrog.com`) is inferred from `info.contact.url`, not stated as a server. The inference is plausible but unconfirmed (the live API may be served from `api.glassfrog.com` or similar).
- **RC-1**: the default is a single centralized constant in `internal/apiclient` (the `/api/v5` path from the spec, the host inferred from `info.contact.url`), so a correction is one-line; confirm the host with the spec owner if the relative-server-URL convention is load-bearing (plan Risks).
- **RC-2**: resolution reports `Source: Default`, so the operator can see the default was used rather than an explicit setting (interface § Surface).
- **Residual: Yellow** — accepted with justification: blast radius is high but the value is a single reviewed constant, the fallback is transparently reported, and no request is sent in this slice (resolution is offline). Re-confirm the host before the connection-context half ships.

### H-2 — Token leak through the shared reader
**Severity: High** (secret exposure) — the `.glassfrogrc` holds the token; a base-URL reader that returned or logged the whole parsed file could leak it. **Probability: Low** — ADR-3 specifies a narrow seam returning only `base_url`.
- **RC-3**: the `auth` base-URL seam returns only the `base_url` value (and path), never the token; a test asserts the token never appears in the value or any error from this path (tasks T001 acceptance + risk note).
- **Residual: Green**.

### H-3 — Regression in the shared `parseCredentials`
**Severity: High** — `parseCredentials` is consumed by 005/006/007; a faulty generalization could break token resolution for the whole auth stack. **Probability: Low** — the change is additive (capture a second key) and guarded by existing tests.
- **RC-4**: 005's token path is preserved; existing 005 unit tests and 006's write→read round-trip stay green; the token reader's exposed surface is unchanged (plan ADR-3, tasks T001).
- **Residual: Green**.

### H-4 — Malformed base URL routes to a bad endpoint
**Severity: Medium** — a bad endpoint fails requests at the connection-context/request layer (read-time); it does not corrupt governance. **Probability: Low** — ADR-4 validates each non-empty value as an absolute `http(s)` URL and fails loud naming the source.
- **RC-5**: `http(s)` validation at resolution; a malformed value yields a typed `BaseURLError` naming the source with no fall-through (interface § Error Communication; ADR-4). *Coverage:* the malformed case is exercised at the acceptance layer for flag, env, **and file** sources (feature scenarios "A malformed flag value fails loudly", "A malformed environment value names the environment variable", "A malformed config-file value fails loudly naming the file") and required at the unit layer by tasks T002 ("malformed-per-source incl. file"). The remaining error surface, an unparseable `.glassfrogrc`, is the shared 005 reader's `*FormatError`, already covered by the `internal/auth` reader's tests.
- **Residual: Green**.

### H-5 — Broken file mistaken for absence → silent wrong default
**Severity: Medium** — a silent fallback to the default on a broken file would hide a misconfiguration. **Probability: Low** — read/parse errors fail loud.
- **RC-6**: an unreadable/unparseable `.glassfrogrc` surfaces 005's typed read/format error and does **not** fall through to the default (interface § Error Communication; ADR-4).
- **Residual: Green**.

### H-6 — `base_url` key contract drift with Credential Storage (006)
**Severity: Medium** — a key mismatch means a stored base URL is silently never read, and the default is used instead. **Probability: Medium** — 006 is unbuilt and the `base_url` key is `[ASSUMED]`, not yet reconciled.
- **RC-7**: `base_url` is a centralized `[ASSUMED]` constant in `internal/auth`; the read side uses the one shared parser; reconcile the key with Credential Storage before either capability ships (plan ADR-3, Integration Design).
- **Residual: Yellow** — accepted with justification: no writer consumes the key yet (006 unbuilt), and the shared-constant + shared-parser design makes reconciliation a single-point change. Track until 006 reconciles.

### H-7 — Silent URL normalization
**Severity: Low**, **Probability: Low** — the spec forbids normalization.
- **RC-8**: the resolved value is passed through verbatim — no trailing-slash/scheme rewriting (spec § Non-Behaviors; interface § Surface).
- **Residual: Green**.

---

## Residual Risk Summary

7 hazards, 8 controls. After controls: 5 Green, 2 Yellow, 0 Red.
- **H-1 (Yellow)**: confirm the derived default host before the connection-context half sends real traffic.
- **H-6 (Yellow)**: reconcile the `base_url` key with Credential Storage (006) before either ships.

Neither Yellow blocks this slice (offline resolution, no request sent, no writer of the key yet). Both are pre-existing `[ASSUMED]`/derivation items already flagged in plan Risks — risk formalizes them with controls and acceptance rationale.

---

## Traceability Index

**Hazards → source**: H-1 → plan § Integration Design/Risks · H-2 → spec § Non-Behaviors, plan ADR-3 · H-3 → plan ADR-3, tasks T001 · H-4 → spec § Error scenarios, plan ADR-4 · H-5 → spec § Error handling · H-6 → plan § Integration Design/Risks · H-7 → spec § Non-Behaviors.

**Controls → grounding**: RC-1/RC-2 → ADR-1/Integration Design (single default constant; Source reporting) · RC-3 → ADR-3 (secret-safe seam) · RC-4 → ADR-3/T001 (preserve token path) · RC-5 → ADR-4 (http(s) validation, typed error) · RC-6 → ADR-4/Error Communication (fail-loud, no fall-through) · RC-7 → ADR-3/Integration Design (shared `[ASSUMED]` constant) · RC-8 → spec Non-Behaviors (verbatim pass-through).

Downstream: tasks may reference RC-N in acceptance criteria (e.g. RC-3 → T001 secret-hygiene test; RC-5 → T002 validation tests + the flag/env/file malformed acceptance scenarios in T003).
