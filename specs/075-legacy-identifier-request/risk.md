# Risk: Legacy Identifier Request

**Feature**: 075-legacy-identifier-request
**Round**: 1 (amended same day after the live probe the analysis called for)
**Date**: 2026-08-08
**Artifacts loaded**: spec.md, plan.md, interface-cli.md, interface-spec.md, PROJECT.md
**Acceptability matrix**: Default 3×3 traffic light

> ⚠ Using default risk acceptability matrix — no project-level matrix found in PROJECT.md. (PROJECT.md has no Regulatory Context, so no IEC 14971 bridge is included.)

> ⚠ **Out-of-sequence invocation.** Risk normally runs between interface and scenarios; here it was triggered by `/score:guard --pre` after tasks and scenarios were already produced. This is *more* input than a first run usually has, not less, so scenario-coverage observations appear against individual hazards. They are noted as context, not as a Step 7 re-run gap analysis.

This feature is **read-only** — no write, no mutation, no governance change (the whole surface is a query parameter on six existing `GET` reads). That removes the entire "governance silently corrupted" band that dominates the write-path specs. What replaces it is a subtler and, in this problem space, more relevant hazard class: **the feature exists to stop an operator being told a role's numeric identifier is unavailable, and its dominant failure mode is telling them exactly that, authoritatively, when it is not true.** A false "this role has no legacy id" is worse than the pre-feature state, because before the feature the operator knew to go to the web UI; after it, a confident absence marker invites them to conclude the number does not exist.

The second organising fact is that the underlying API facility is a **declared, dated bridge** (LEARNINGS S3: *"retires when the v3 API retires. Do not build durable integrations on it"*). Every hazard about durability, caching, and silent capability loss traces to that single upstream property, which the CLI cannot control — only degrade gracefully against.

---

## Risk Register

> **Amendment (post-probe).** The analysis below called for live-probing before implementing (RC-1). The developer directed the probe; it ran against all six operations. **H-1 is closed in the keep direction** — the tree read returns the field its schema omits — and **H-12 is closed with an exception that falsified a rendering accord**. **H-2's posture was decided** (tolerate both spellings). Per-hazard amendments are marked ▸ inline; the register rows and the summary reflect the post-probe state. Evidence: LEARNINGS 2026-08-08, W1–W6.

| H | Hazard | Source | Sev | Prob | Level | Controls | Residual |
|---|---|---|---|---|---|---|---|
| H-1 | Tree read reports every role's number as absent, though the numbers exist | interface-cli.md § per template (`tree.*`); plan ADR-3 | High | ~~High~~ **nil** | ~~Red~~ **closed** | RC-1 ✔, RC-2 ✔, RC-3 ✔ | **Green** |
| H-2 | An optional convenience field's type drift fails an otherwise-successful read | interface-cli.md § Error Communication; plan ADR-3 | Medium | Low | Green | RC-4 ✔, RC-5 ✔ | **Green** |
| H-3 | A number the response did not carry is synthesized or inferred, and a later write targets the wrong governance element | spec.md non-behavior 4 | High | Low | Yellow | RC-6, RC-7 | Green |
| H-4 | The bridge retires and nobody notices; every read silently loses the capability | spec.md non-behavior 10, § Integration Boundaries; plan § Risks | Medium | Medium | Yellow | RC-8, RC-9 | Yellow |
| H-5 | Structured output gains curation, so a machine consumer can no longer tell what the API returned | spec.md non-behavior 5; plan ADR-2 | Medium | Low | Green | RC-10, RC-11 | Green |
| H-6 | The flag is offered on a contract-excluded read, looks honoured, and returns nothing | spec.md non-behavior 2 | Medium | Low | Green | RC-12, RC-13 | Green |
| H-7 | Consumers build durable dependence on the number and break when it retires | spec.md non-behaviors 3 and 7, US4 | Medium | Medium | Yellow | RC-14, RC-15, RC-16 | Yellow |
| H-8 | Render-data threading regresses a settled human render that agent consumers parse | plan ADR-3 § Consequences, § Risks | Medium | Medium | Yellow | RC-17, RC-18 | Green |
| H-9 | A page of a walked list loses the parameter, yielding a partly-numbered result | spec.md § Behavioral Accord (Requesting); § Integration Boundaries (Pagination) | Medium | Low | Green | RC-19 | Green |
| H-10 | The opt-in becomes a persistent default via env or config, indistinguishable from always-on | spec.md non-behavior 3 | Low | Low | Green | RC-20 | Green |
| H-11 | The retirement caveat weakens in help text while the guard still passes | interface-spec.md invariant 4 | Low | Medium | Green | RC-21 | Green |
| H-12 | The embed note asserts embeds carry no number when one of them does | interface-cli.md § shared idioms; LEARNINGS S2 | Medium | ~~Low~~ **materialised** | Green | RC-22 ✔, RC-23 ✔ | **Green** (fixed) |

**Unacceptable (Red) before controls: 1** — H-1, now **closed** by the probe RC-1 called for. **Residual Red: 0**, unconditionally.

**Two hazards materialised rather than being avoided**, which is the outcome that justified running this analysis:
- **H-1** resolved *opposite* to the schema evidence: the tree read works and the vendored `TreeNode` schema is defective (W1). Deciding from the contract alone would have dropped a fully working read.
- **H-12** was rated Low probability and **was already live in the artifacts**: interface-cli.md specified an embed note on `me`'s embedded roles, which do in fact carry the number (W2). The note would have been a false statement in rendered output. Fixed in the accord, the interface, and the feature files.

---

## Hazard Detail

### H-1 — The tree read reports every role's number as absent, though the numbers exist — **Red**

**Description**: `getRoleTree` references the `IncludeLegacyId` parameter, but `TreeNode` — the schema its `200` returns — declares no `legacy_id` property and no `additionalProperties`. If the response genuinely omits the field, then `tree --legacy-id` renders the settled absence marker on **every row**, for roles whose numbers are independently obtainable (verified: `role_4d8f01d9…` → `14062695` via `getRole`, LEARNINGS S1). An operator or agent reading a subtree sees uniform absence and reasonably concludes the numbers do not exist — the precise false conclusion this feature was built to prevent, now delivered with the authority of a successful read.

**Severity — High**: the failure is *misinformation*, not unavailability. spec.md § System Overview states the capability's purpose as making the number obtainable "without leaving the CLI"; a uniformly-absent tree actively argues against going to look. It also degrades US3 (distinguishing same-named roles), where the tree is the natural read for comparing siblings.

**Probability — High**: this is not a rare edge. If `TreeNode` omits the field at runtime, it manifests on *every* tree read with the flag. The parameter's own description scopes it to *"the standard resource list/show endpoints and `/me`"* and excludes *"nested `?include=` embeds"*; the API overview additionally excludes aggregation endpoints. A recursive `children` tree is neither a standard list/show nor free of nesting, which makes omission the more likely runtime behavior and the `getRoleTree` `$ref` the likelier contract defect. Compounding: `getRoleTree` was **never live-probed** — S1's `curl` verification covered `getRole` only, and S2's coverage list was derived from `$ref` enumeration.

**Controls**:
- **RC-1** — Resolve the tree read's coverage before implementing it: either live-probe `getRoleTree?include_legacy_id=true` and keep it on observed evidence, or drop `tree` from the supported set and record the contract self-contradiction alongside U1–U3 (LEARNINGS 2026-08-05). The vendored spec is not edited either way — the standing triage precedent for this file.
- **RC-2** — Response-schema capability is asserted mechanically, not just parameter coverage: the guard checks that every mapped operation's response schema declares `legacy_id` directly or through `allOf` composition. interface-spec.md's invariant 1 as written checks `$ref` sites only, so it passes with `getRoleTree` on both sides while the schema remains incapable — the guard cannot currently catch this hazard.
- **RC-3** — Absence is never rendered in a form that asserts non-existence in the system of record. The idiom distinguishes "this read does not carry it" from "this resource has none" — the same distinction interface-cli.md already makes for embedded resources.

**▸ Amendment — CLOSED, Green.** RC-1 was executed: `GET /roles/{id}/tree?include_legacy_id=true` returns `legacy_id` on **all 1336 nodes to depth 6, every value a non-null integer** (LEARNINGS W1). The hazard's premise — that the field does not arrive — is false. `TreeNode`'s omission is an upstream schema defect, the fourth instance of the U1–U3 class, recorded rather than fixed per the standing precedent for this vendored file. Probability drops to nil; the tree read stays in scope on observed evidence.

**RC-2 was reshaped by what the probe found**, and this is the more durable lesson. As originally written ("the guard asserts every mapped operation's response schema declares `legacy_id`") the control would have been **actively harmful**: `TreeNode` omits the field permanently, so the guard could never go green, the vendored file cannot be edited, and the only passing state would have been deleting a working read — a guard whose sole remedy is the removal of correct behavior. It is now an **observed-exception register** (interface-spec.md invariant 5): a mapped operation must either have a schema that declares the field *or* a register entry carrying its probe evidence, with a subset rule so a stale exception fails loudly. That preserves the control's value — a *newly* mapped operation with an incapable schema still fails until someone probes it — without blocking on an upstream defect.

**Scenario coverage**: "One tree read yields legacy numbers for a whole subtree" now additionally asserts that rows at *every depth* carry the number, matching what was observed.

### H-2 — An optional convenience field's type drift fails an otherwise-successful read — Yellow

**Description**: plan ADR-3 decodes `legacy_id` into `*int64` and accepts that a non-integer fails the typed decode (`APIError`, exit 3). So a read the operator would otherwise have got in full fails entirely — because they asked for an optional extra. LEARNINGS V10 records this identifier family being loosely typed elsewhere in the same API: `databaseId` is a **string** at the `UpdateRole` level and an **integer** in nested children, *in the same payload*.

**Severity — Medium**: fail-loud, not misleading — the operator is blocked but not deceived, and the structured path is unaffected (018 raw bytes never decode the typed struct). The blast radius is the human and template paths of whichever read was requested.

**Probability — Medium**: the contract declares `integer` and the live probe returned an integer, so today it holds. The rating is driven by the recorded looseness in the sibling spelling, on a field that is explicitly transitional and therefore less likely to be tightly maintained upstream.

**Controls**:
- **RC-4** — The decode posture for this field is a deliberate, recorded decision rather than an inherited default: fail-loud (a contract break is surfaced) is defensible under Principle III and preserves Principle VIII, but the alternative — tolerating both spellings for this one transitional field — trades strictness for not failing a read over an optional extra. The developer chooses; the choice is written down.
- **RC-5** — Whichever posture is chosen, the *stable* read is never made worse by asking: if a tolerant posture is chosen, the tolerance is scoped to this field alone and does not soften decoding elsewhere.

**▸ Amendment — DECIDED, Green.** The developer chose to **tolerate both spellings** for this one field ("let's keep it light"). The decode accepts a JSON integer or a JSON string and yields the same value; a value that is neither still fails loudly, so a genuine contract break stays visible. The probe found **no string spelling anywhere** across roles, the full tree, 164 actors, and `me` — every observed value is an integer (W5) — so this is cheap insurance against a transitional field's looseness rather than a fix for an observed defect. Probability of a read failing over the optional extra drops to Low, and the residual is Green.

**Scenario coverage**: now covered — `Scenario Outline: A legacy number is accepted in either spelling` (absence.feature, 2 examples), owned by T003.

### H-3 — A synthesized or inferred number targets the wrong governance element — Yellow → Green

**Description**: the number's consumer is a proposal change payload. A fabricated, defaulted, or content-matched value would silently address a *different* role, accountability, or person than intended — governance changed against the wrong target.

**Severity — High**: this is the worst outcome in the problem space, and it lands on live governance through a downstream consumer.

**Probability — Low**: strongly designed out. LEARNINGS V3 establishes that the stable identifiers are UUID v4 and encode nothing, so nothing *can* be derived; plan ADR-2 means the structured path carries raw bytes with no synthesis point; the `*int64` decode fails rather than coercing to `0`.

**Controls**:
- **RC-6** — No value is presented that the response did not carry: no synthesis, no derivation, no defaulting, and no zero-value substitution for absence (absence uses the settled non-numeric idioms).
- **RC-7** — Content-matched resolution stays out of this capability. Inferring a number from labels is a separate capability (#78/#79) with its own declining-on-ambiguity design; folding it in here would let an inferred value render identically to a returned one.

**Residual — Green**.

**Scenario coverage**: the `@validation` scenario "Structured output carries the legacy_id key exactly where the response did" asserts no synthesized number, and "No outbound request addresses a resource by its legacy number" covers the misuse direction.

### H-4 — The bridge retires and nobody notices — Yellow

**Description**: spec.md deliberately forbids any runtime retirement signal (non-behavior 10), because the CLI cannot distinguish "retired" from "these resources have none". So on retirement, every `--legacy-id` read succeeds, renders uniform absence, and says nothing. The capability disappears silently and operators fall back to the web UI without knowing why.

**Severity — Medium**: capability loss, not misinformation — and a documented, expected end state for a dated bridge. The unconditional fallback (#82, asking the operator for the number) is designed to remain the floor beneath it.

**Probability — Medium**: certain eventually; timing unknown and not in the project's control.

**Controls**:
- **RC-8** — Detection is assigned to a mechanical check over the vendored contract rather than to a human noticing. LEARNINGS S7 establishes why this is the only workable instrument: `info.version` did not move across 15 new operations, and the vendor's own changelog listed neither this parameter nor the change-type enum.
- **RC-9** — The failure mode is designed to be non-breaking: reads keep working, absence is legitimate, and no exit code or diagnostic changes. Retirement degrades the feature to the pre-feature state rather than to a broken one.

**Residual — Yellow**: accepted with justification. RC-8 makes retirement *detectable at the refresh PR*; it cannot make it detectable at runtime, and spec.md's clarification records why attempting that would produce false alarms on working reads.

**Note**: interface-spec.md states this residue explicitly — the guard cannot see the API dropping the field behaviorally while the contract still declares it. That gap is real, stated, and unaddressed by design.

### H-5 — Structured output gains curation — Green

**Description**: if the structured path ever decodes and re-encodes, or filters the membership number, or adds a reason field beside a null, a machine consumer loses the ability to tell what the API actually returned.

**Severity — Medium** (agent consumers are the primary audience per PROJECT.md § Actors). **Probability — Low**: plan ADR-2 means there is no code path to curate — the structured branch hands raw bytes to the serializer (`internal/cli/render.go` decodes into `json.RawMessage`). Curation would require overturning landed decision 018.

**Controls**: **RC-10** — the structured document mirrors the response shape exactly, with no key added or removed. **RC-11** — the curated/faithful split is asymmetric by design: curation is permitted only in the human render, where a reader can see it, and never in the machine path.

**Residual — Green**. Covered by two `@validation` scenarios.

### H-6 — The flag is offered where the API ignores it — Green

**Description**: the API does not reject unrecognized request parameters, so a flag offered on an excluded read would appear honoured and return nothing — a silent lie, worse than a refusal.

**Severity — Medium**, **Probability — Low**: plan ADR-1 makes the supported set *identical to* where the flag is registered, so there is no allowlist to fall out of sync. Rejection is cobra's default unknown-flag path, classified `UsageError` → exit 2 before any request (`internal/cli/dispatch.go` records it as deliberately left on).

**Controls**: **RC-12** — the supported set is expressed by registration, not by a parallel validator. **RC-13** — the "nowhere else" property is asserted mechanically across the whole command tree, including the `me` subcommands, which are contract-excluded and share their parent's namespace.

**Residual — Green**. RC-13's local-not-inherited requirement is the one to watch: registering with `PersistentFlags()` instead of `Flags()` on `me` would silently offer the flag to `me roles`, `me actions`, and `me projects`.

### H-7 — Consumers build durable dependence on a dated bridge — Yellow

**Description**: the number is sanctioned but time-limited. Anything that treats it as a stable identifier — a stored mapping, a read keyed on it, a config-set default that makes it always-on — breaks on the retirement date.

**Severity — Medium**, **Probability — Medium**: the pressure is real. The number is *more convenient* than the stable identifier for anyone coming from v3, and nothing at runtime pushes back.

**Controls**: **RC-14** — the caveat is stated where the operator meets the option, and its presence is mechanically anchored. **RC-15** — the number is never accepted as input and never addresses a resource, so no request path can depend on it. **RC-16** — no persistence: nothing caches it across invocations and nothing makes it a config-set default.

**Residual — Yellow**: accepted. The controls prevent the *CLI* depending on it; they cannot prevent a downstream consumer doing so, which is why US4 exists and why the caveat's placement is guarded.

### H-8 — Render-data threading regresses a settled human render — Yellow → Green

**Description**: twelve templates and five render-data paths change. The human renders are a parsed contract for agent consumers, so a stray segment, an alignment shift, or a lost line is a consumer-visible break — on reads that have nothing to do with this feature when the flag is absent.

**Severity — Medium**, **Probability — Medium**: the surface is broad and every affected template is touched.

**Controls**: **RC-17** — the not-requested render is pinned as byte-identical to its pre-feature output, per resource, so the default path cannot drift. **RC-18** — the new branches are covered per template across the requested × present/absent matrix, rather than exercised only through the happy path.

**Residual — Green**, given RC-17. Note the adjacent, non-render instance of the same shape: adding a field to five structs re-aligns sibling `gofmt` columns, which the repository has already paid a triage round for (PR #164) — checklist C8 records that only one of the five tasks currently carries a lint gate.

### H-9 — A page of a walked list loses the parameter — Green

**Severity — Medium** (a partly-numbered result reads as "some of these roles have numbers", which is misinformation of the H-1 kind), **Probability — Low**: `internal/paging/paging.go` deep-clones `req.Query` per page, so the parameter rides the existing walk mechanism rather than needing per-page reconstruction — the same path the `q` search parameter already uses.

**Controls**: **RC-19** — uniformity across a walk is a stated property, not an incidental one: no page carries the parameter while another does not.

**Residual — Green**. Covered by "Every role in a walked list carries its legacy number".

### H-10 — The opt-in becomes a persistent default — Green

**Severity — Low** (it degrades the design's intent rather than producing wrong data), **Probability — Low**: nothing registers in the setting resolver, and the flag is per-invocation cobra state. The hazard is a future edit adding env/rcfile support by analogy with `--base-url` and `--output`.

**Controls**: **RC-20** — the exclusion from the resolver is recorded as deliberate, with its reason (a persisted default is indistinguishable from always-on), so a future editor reads the intent rather than inferring an omission.

**Residual — Green**.

### H-11 — The retirement caveat weakens while the guard passes — Green

**Description**: interface-spec.md's invariant 4 checks the help constant's identity plus two word stems rather than exact prose, deliberately, so wording can improve without breaking CI. A rewrite could keep the stems and lose the meaning.

**Severity — Low**, **Probability — Medium** (prose drifts; stems are cheap to preserve accidentally).

**Controls**: **RC-21** — the guard pins the *property* the text must convey, and the property is written into the guard's own failure message so an editor re-derives it rather than pattern-matching the check. This is the deliberate middle ground between freezing prose (which false-fails on a legitimate improvement) and checking nothing.

**Residual — Green**. Accepted: the alternative — asserting exact prose — trades a real failure mode for a more annoying one.

### H-12 — The embed note asserts embeds carry no number when one does — Green

**Description**: the human render states once per embedded group that embeds do not carry the number. LEARNINGS S2 verified this live for *role accountabilities* only; the other embed families (`me?include=roles`, actor assignments, role subroles/fillers) were not probed. If any embed does carry the field, the note is false and the structured output — which faithfully echoes whatever arrived — would contradict the human render on the same read.

**Severity — Medium** (self-contradiction between two output formats of one read is corrosive to trust in both), **Probability — Low** (the contract excludes embeds explicitly in two places, and the one probed family conformed).

**Controls**: **RC-22** — the embed families are observed before the note's wording is fixed, rather than the note being generalized from one verified family. **RC-23** — the note's wording is scoped to what this read carries rather than asserting what exists, so it stays true even if an embed unexpectedly carries the field.

**▸ Amendment — MATERIALISED and fixed, Green.** RC-22 was executed and the hazard turned out to be **already live in the artifacts**, at Low rated probability. The probe found the embed behavior is **per-endpoint, not per-schema** (W2): `GET /me?include=roles` carries `legacy_id` on every embedded role, while `GET /roles/{id}?include=subroles` does not — the same `Role` schema, identical key sets otherwise, opposite behavior on the same call. interface-cli.md had specified an embed note on `me.full`'s `roles:` group, which would have printed a false statement. W3 separately widened the *confirmed* exclusion from one family to all five of `getRole`'s embeds (`subroles`, `assignments`, `fillers`, `accountabilities`, `domains`), so those notes are correct.

Fixed in four places: the spec's Absence accord now has a bullet for each direction and names the determination as per-read observed fact; interface-cli.md's `me.full` row renders the number on embedded roles and carries no note; the embed-note rule states what *this read* carries (RC-23, now load-bearing rather than precautionary); and absence.feature gained "A read whose embeds carry the number renders it and states nothing".

**The structured path required no change at all** — ADR-2's faithful echo mirrors whatever arrived rather than asserting a rule, so it was correct before and after the discovery. That is the second contract surprise the principle has absorbed for free, and the strongest argument for it.

**Residual — Green**, on observed evidence for six of seven embed families. `actor.full`'s groups were not separately probed; RC-23's wording keeps the note true either way.

---

## Residual Risk Summary

Post-probe state:

| Residual | Count | Hazards |
|---|---|---|
| Red | 0 | — |
| Yellow | 2 | H-4 (silent retirement), H-7 (durable dependence) |
| Green | 10 | H-1, H-2, H-3, H-5, H-6, H-8, H-9, H-10, H-11, H-12 |

Both remaining Yellows are **inherent to the upstream facility** and cannot be engineered away. They trace to LEARNINGS S3 — the API declares the field temporary — so the CLI's honest posture is graceful degradation plus a visible caveat, not prevention. H-4's detection is delegated to a build-time guard because runtime detection would produce false alarms on working reads; H-7's controls prevent the *CLI* depending on the number but cannot stop a downstream consumer.

Three hazards moved on evidence rather than on argument:
- **H-1** Red → Green: the premise was false. The tree read works; the schema is defective (W1).
- **H-2** Yellow → Green: the developer chose tolerance, and no string spelling exists in practice (W5).
- **H-12** Green (precautionary) → Green (fixed): the hazard was already live in the artifacts and is now corrected (W2).

**What the analysis was worth**: H-1 and H-12 were both found by probing rather than reasoning, and they resolved in *opposite* directions — one said "the contract understates what you get", the other "the contract overstates what you can assume". Neither would have been caught by any artifact review, and one of them (H-12) had already been written into a rendered-output accord as a false statement.

**One residual worth naming plainly**: the agent-backed-null behavior is still contract-only (W6 — the organization has no agent actors, list complete). The absence-reason rendering it drives is fixture-tested, and the claim should be re-probed the first time an `agt_` actor appears.

---

## Traceability Index

**Hazards → source**

| H | Source section |
|---|---|
| H-1 | interface-cli.md § Human output — per template (`tree.compact`/`tree.full`); plan ADR-3; vendored spec `TreeNode` |
| H-2 | interface-cli.md § Error Communication; plan ADR-3; LEARNINGS V10 |
| H-3 | spec.md § Non-Behaviors 4; LEARNINGS V3 |
| H-4 | spec.md § Non-Behaviors 10, § Integration Boundaries; plan § Risks; LEARNINGS S3/S7 |
| H-5 | spec.md § Non-Behaviors 5; plan ADR-2; DECISIONS §145 (018 raw bytes) |
| H-6 | spec.md § Non-Behaviors 2; plan ADR-1 |
| H-7 | spec.md § Non-Behaviors 3 and 7, § User Scenarios (US4); LEARNINGS S3 |
| H-8 | plan ADR-3 § Consequences, § Risks |
| H-9 | spec.md § Behavioral Accord (Requesting); § Integration Boundaries (Pagination) |
| H-10 | spec.md § Non-Behaviors 3 |
| H-11 | interface-spec.md invariant 4 |
| H-12 | interface-cli.md § Human output — shared idioms; LEARNINGS S2 |

**Controls → architectural grounding**

| RC | Grounded in |
|---|---|
| RC-1 | LEARNINGS 2026-08-05 U1–U3 discipline (probe before specifying; never edit the vendored copy) |
| RC-2 | plan ADR-4 / interface-spec.md invariants — extends parameter-coverage to response-schema capability |
| RC-3 | spec.md § Behavioral Accord (Absence); interface-cli.md § shared idioms |
| RC-4, RC-5 | plan ADR-3 (`*int64` and its decode-failure consequence) |
| RC-6, RC-7 | spec.md § Non-Behaviors 4 and 6; FEATURE-MODEL complement split (#78/#79) |
| RC-8 | plan ADR-4; interface-spec.md invariant 1; LEARNINGS S7 |
| RC-9 | spec.md § Behavioral Accord (Absence, final bullet) |
| RC-10, RC-11 | plan ADR-2; spec.md § System Overview (faithful echo / curated human split) |
| RC-12, RC-13 | plan ADR-1; interface-spec.md invariants 2–3 |
| RC-14, RC-15, RC-16 | spec.md § Non-Behaviors 3, 6, 7; interface-spec.md invariant 4 |
| RC-17, RC-18 | plan § Render Design; interface-cli.md § per-template table |
| RC-19 | `internal/paging` per-page query clone; DECISIONS §247 (`q` carried on every page) |
| RC-20 | plan § Cross-cutting Concerns (Configuration: deliberately none) |
| RC-21 | interface-spec.md invariant 4 (property-not-prose check) |
| RC-22, RC-23 | plan § Risks (embed-family verification); LEARNINGS S2 |

---

## Degradation Flags

None on input — spec.md, plan.md, both interface files, and PROJECT.md were all present and complete.

Two limits on this analysis, stated rather than implied:

1. **No project acceptability matrix.** Severity/probability bands and the Red/Yellow/Green mapping are the default 3×3, not calibrated to this project's tolerance. A project-level matrix in PROJECT.md would make H-2's and H-7's Yellow classifications less a matter of the Guardian's judgment.
2. ~~**Live behavior unobserved for five of six operations.**~~ **Retired by the probe.** All six operations were probed directly (W1–W5): the tree read to full depth, all five `getRole` embed families, `me`'s three numbers and its embedded roles, and 164 actors. Two ratings that rested on contract-only evidence changed as a result. The one claim still unobserved is agent-backed nullability (W6), which is unverifiable in this organization.
