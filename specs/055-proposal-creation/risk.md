# Risk: Proposal Creation

**Feature**: 055-proposal-creation
**Round**: 1
**Date**: 2026-06-15
**Artifacts loaded**: spec.md, plan.md, interface-cli.md, interface-spec.md, PROJECT.md
**Acceptability matrix**: Default 3×3 traffic light

> ⚠ Using default risk acceptability matrix — no project-level matrix found in PROJECT.md. (PROJECT.md has no Regulatory Context, so no IEC 14971 bridge is included.)

This is the CLI's **second write** and the anchor of the governance write path. Unlike Tension Capture (042) — whose risk model rested on the fact that a tension is an *operational, recoverable* resource — Proposal Creation sits on the surface PROJECT.md singles out as the project's defining shape: *"a proposal is the only sanctioned path to alter governance structure"* (PROJECT.md § Domain / Constraint "Governance is proposal-gated"). That elevates the *correctness* of what gets created and *against what tension* it is anchored. The countervailing safety property is that a `create` produces a **`draft`** proposal: the change set is recorded but **not yet enacted** — it must be advanced to circulation and accepted by separate, deferred capabilities before it touches live governance. Most hazards here therefore land in the "wrong draft recorded / confusing failure" band, not "governance silently corrupted," and several are already mitigated by the design rather than left to the implementer.

The dominant *new* hazard relative to 042 is the **free-form `changes` pass-through**: the CLI sends a caller-supplied JSON array through verbatim above a one-key `type` floor (spec § Behavioral Accord; plan ADR-3), so the *content* of a governance change is never inspected client-side. This is a deliberate boundary (the deferred *Unguided Change Construction* problem), so the residual risk is about *where it is detected* (server `422`), not about preventing it here.

---

## Risk Register

| H-ID | Hazard | Source | Severity | Probability | Risk Level | Controls | Residual Risk |
|---|---|---|---|---|---|---|---|
| H-1 | Bad-but-typed change reaches the server (free-form verbatim pass-through, no per-key validation) | spec.md § Behavioral Accord / Non-Behaviors; plan.md ADR-3 / § Risks | Medium | Medium | Yellow | RC-1, RC-2 | Yellow (justified) |
| H-2 | Proposal anchored to the wrong tension (`<tension-id>` passed through unvalidated) | spec.md § Assumptions ([ASSUMED] id not validated); interface-cli.md § Surface | Medium | Low | Green | RC-3 | Green |
| H-3 | Inline JSON misclassified as a file (or a file as inline) by the source resolver | plan.md ADR-2 / § Risks; interface-cli.md § Change-set sourcing | Low | Low | Green | RC-4 | Green |
| H-4 | Empty / typeless / non-array change set slips through to create a meaningless proposal | spec.md § Input (empty/typeless rejection); plan.md ADR-3 | Medium | Low | Green | RC-5 | Green |
| H-5 | Double-submitted proposal from a retried non-idempotent `POST` (duplicate governance change requests) | plan.md § Cross-cutting (Non-idempotent retry, §133) / § Risks; interface-cli.md § Non-idempotent retry | Medium | Low | Green | RC-6 | Green (accepted) |
| H-6 | Premium `403` surfaces as an opaque permission error, not "enable async proposals" | plan.md § Cross-cutting (Premium gate) / § Risks; interface-cli.md § Premium gate | Low | Medium | Green | RC-7 | Green (accepted) |
| H-7 | API token leaked in output, diagnostics, or the request body | interface-spec.md § auth boundary; CONSTITUTION II / PROJECT.md § Constraints | High | Low | Yellow | RC-8 | Yellow (justified) |
| H-8 | `--changes stdin` reads a terminal or empty pipe, or a directory/unreadable file is read as a change set | interface-cli.md § Change-set sourcing; plan.md ADR-2 | Low | Low | Green | RC-9 | Green |
| H-9 | A non-2xx (`404`/`422`/`5xx`) mis-surfaced as success or as an opaque failure | interface-cli.md § Error Communication; interface-spec.md § Error Communication | Medium | Low | Green | RC-10 | Green |
| H-10 | Partial/corrupt output on a render failure after a successful create | plan.md § Cross-cutting (Output: buffer-then-write); interface-spec.md § Surface | Low | Low | Green | RC-11 | Green |

---

## Hazard Details

### H-1: Bad-but-typed change reaches the server
**Source**: spec.md § Behavioral Accord ("sends it through **verbatim**") and § Non-Behaviors ("must not validate the *value* of a change's `type`, nor any command-specific key"); plan.md ADR-3 / § Risks ("Verbatim pass-through ships a bad-but-typed change").
**Description**: The CLI validates only that each element is an object with a non-empty `type`. A change with a valid-looking `type` but wrong, missing, or malformed command-specific keys (e.g. a `CreateRole` lacking a `name`, or an `UpdateAccountability` pointing at a non-existent id) is sent through unread. Because a proposal is the project's only path to alter governance structure (PROJECT.md), the *content* of these commands is governance-load-bearing.
**Severity**: Medium — the malformed change is recorded in a **`draft`** proposal, not enacted; the server rejects it at create with a `422` (caught before the proposal exists) or, if structurally accepted, it surfaces when a later advance/accept step runs. The blast radius is bounded by the draft gate and by the API owning per-change validation. It is not Low (a wrong governance command is more consequential than a wrong operational field) nor High (nothing is enacted by this command).
**Probability**: Medium — the operator is an AI agent assembling free-form change JSON with no client-side schema to check against (typed builders are the deferred *Unguided Change Construction* capability), so a structurally-valid-but-semantically-wrong change is a realistic input, not an edge case.
**Risk Level**: Yellow (Medium × Medium).
**Controls**:
- **RC-1**: The server owns per-change validation — a rejected change/field returns `422`, surfaced cleanly through API Error Extraction (015) with the RFC 9457 detail naming what was wrong, exiting non-zero. The malformed change is caught *at create*, before any proposal exists.
- **RC-2**: The change set is carried **byte-for-byte** as `[]json.RawMessage`, so the CLI neither reshapes nor silently drops command keys — what the agent supplied is exactly what the server validates, making the `422` detail trustworthy and reproducible.
**Residual Risk**: Yellow (justified) — this is a deliberate design boundary (spec non-behavior; the verbatim pass-through above a floor). The residual exposure is that a *structurally well-formed but semantically wrong* change could be accepted into a draft and only surface at the advance/accept stage. That is acceptable because (a) nothing is enacted by `create`, (b) the draft is fully recoverable/withdrawable through deferred siblings, and (c) closing it client-side *is* the deferred Unguided Change Construction capability, explicitly out of scope. **Worth the implementer's attention**: do not add partial/ad-hoc per-`type` validation here — it would fork the deferred capability and give a false sense of coverage. The floor reads **only** `type`; every other key rides through untouched (validation scenario in the feature file pins this).

### H-2: Wrong anchor tension
**Source**: spec.md § Assumptions ("[ASSUMED] Tension-id format is not validated client-side"); interface-cli.md § Surface (the id "is **not** validated locally; the API resolves it").
**Description**: The positional `<tension-id>` is sent as `proposal.tension_id` without client-side shape or existence validation. A valid-but-unintended `ten_…` anchors the proposal to the wrong tension.
**Severity**: Medium — a proposal anchored to the wrong tension associates a governance change with the wrong sensed need; it is recorded in a draft (recoverable, withdrawable) and the created proposal echoes its `tension_id` for verification.
**Probability**: Low — the agent typically resolves the `ten_` id from a prior capture/read; an unknown id 404s cleanly rather than silently mis-anchoring, and a typo'd-but-valid id is uncommon.
**Risk Level**: Green (Medium × Low).
**Controls**:
- **RC-3**: The created proposal renders its anchor `tension_id` on both the human (`full`) and structured paths, so the operator can verify attribution immediately; an unknown/malformed id surfaces the API's `404`/`422` (H-9 chain) rather than a silent wrong anchor. Exactly one positional is required (`ExactArgs(1)`), so a missing or extra anchor is a pre-request usage error.
**Residual Risk**: Green — verification-on-output plus clean not-found make a silent mis-anchor unlikely; consistent with how 042/044/038 treat their ids (server owns id-shape validation).

### H-3: Source-resolver misclassification
**Source**: plan.md ADR-2 / § Risks ("Inline-vs-file misclassification"); interface-cli.md § Change-set sourcing.
**Description**: `resolveChangesSource` classifies a non-`stdin` `--changes` value as an existing-file read or, failing that, inline JSON. A value meant as inline JSON that happens to match an existing file path would be read as a file (or vice versa).
**Severity**: Low — both branches feed the *same* JSON parse + `type` floor, so a misread source still produces a validated (or cleanly rejected) change set; it does not produce a *wrong-but-valid* change silently.
**Probability**: Low — inline JSON begins with `[` (after trim), which is never a plausible bare file path; the classification is deterministic and the existence check uses a **regular-file** guard so a directory is not read as a source.
**Risk Level**: Green (Low × Low).
**Controls**:
- **RC-4**: The resolver order is fixed and documented (reserved `stdin` keyword → existing **regular** file → inline bytes), mirroring the 035 reserved-name-wins precedent; a file literally named `stdin` is still reachable as `./stdin`. The regular-file guard rejects a directory rather than reading it as a change set.
**Residual Risk**: Green — the rare collision (inline value that is also a real path) is parsed and validated identically, and is documented in the spec assumption.

### H-4: Empty / typeless change set creates a meaningless proposal
**Source**: spec.md § Input ("empty array … rejects", "object lacking a non-empty `type` … rejects"); plan.md ADR-3.
**Description**: A proposal with no changes, or with elements that are not objects / lack a `type`, would be a meaningless or server-rejected governance change request.
**Severity**: Medium — a changeless or typeless proposal is meaningless governance noise; recorded as a draft and reversible, but it pollutes the proposal list and wastes a write.
**Probability**: Low — caught by the client floor before any request.
**Risk Level**: Green (Medium × Low).
**Controls**:
- **RC-5**: The `type` floor validates **before any request** (fail-fast `UsageError(2)`, transport tripwire pins "no request sent"): the change set must be valid JSON, a non-empty array, and every element an object with a non-empty `type`. Missing `--changes` is itself a pre-request usage error naming the flag. The server's `422` is a backstop for anything that passes the floor.
**Residual Risk**: Green — the floor plus the no-request tripwire make a meaningless create unlikely; the floor is the only knowable-local check (§113 "validate what's knowable client-side").

### H-5: Double-submitted proposal on retry
**Source**: plan.md § Cross-cutting (Non-idempotent retry — §133 `isSafeMethod`) / § Risks ("Double-submit on retry"); interface-cli.md § Non-idempotent retry.
**Description**: `createProposal` is a non-idempotent `POST` with no idempotency key in the API. A retry — whether HTTP-level on a `429`, or an operator re-running after an ambiguous post-success network failure — could create a duplicate governance proposal.
**Severity**: Medium — duplicate **draft** proposals are recoverable (withdrawable via deferred siblings) and not enacted, but a duplicate governance change request is more consequential than a duplicate operational tension (042 rated this Low): it can confuse a circle's proposal list and the downstream advance flow.
**Probability**: Low — the auto-retry path is structurally closed (see RC-6); the only remaining window is a manual re-run after a post-success transport failure, which an agent can detect from the non-zero exit and (via deferred reads) the created proposal's existence.
**Risk Level**: Green (Medium × Low).
**Controls**:
- **RC-6**: HTTP-level auto-retry is suppressed for the non-idempotent `POST` — 017's `isSafeMethod` restricts `429` auto-retry to `GET`/`HEAD`, so a `POST` surfaces the `429` on first occurrence and is **never silently re-sent** (silent conformance to §133; a feature scenario pins "rate-limit surfaced, not retried, no duplicate"). A clear success/failure exit code lets the agent distinguish a completed create from a failed one.
**Residual Risk**: Green (accepted) — the post-success-network-failure window is inherent to a non-idempotent create with no idempotency key in the API; it is recoverable through the deferred proposal reads/withdraw. **Worth the implementer's attention**: do **not** add a client-side auto-retry on the create `POST` — doing so would convert this inherent edge case into systematic duplication of governance proposals. RC-6's POST-no-retry gate is load-bearing.

### H-6: Premium `403` reads as an opaque permission error
**Source**: plan.md § Cross-cutting (Premium gate, never pre-checked) / § Risks ("Premium `403` reads as a generic permission error"); interface-cli.md § Premium gate; PROJECT.md § Constraints (Premium-gated async proposals).
**Description**: The whole proposal write surface is Premium-gated; an org without async proposals enabled returns `403`. The command issues the request unconditionally and surfaces the `403` through the shared permission outcome, so the operator sees a generic permission denial rather than "enable async proposals / upgrade the plan."
**Severity**: Low — a confusing-but-correct failure; the create simply does not happen and the server's RFC 9457 detail string still surfaces the explanation.
**Probability**: Medium — any org without the Premium capability hits this on **every** create attempt, so for those orgs it is the common path, not an edge case.
**Risk Level**: Green (Low × Medium).
**Controls**:
- **RC-7**: A `403` classifies through API Error Extraction (015) as `PermissionError(4)` with the server's extracted RFC 9457 detail surfaced (e.g. "async proposals not enabled") and a non-zero exit — the operator gets the server's own explanation even though the CLI adds no Premium-specific interpretation.
**Residual Risk**: Green (accepted) — distinct plan-limit signalling is explicitly deferred to Feature-Gate Recognition (060) / Plan-Limit Signal (061); the spec non-behavior forbids a client-side Premium pre-check here. The surfaced detail string is the mitigation until those siblings land. **Worth the implementer's attention**: do not add a client-side Premium feature-check — a validation scenario pins "issues the request rather than pre-checking the gate."

### H-7: Token leakage
**Source**: interface-spec.md § Error Communication ("No secret anywhere"); CONSTITUTION II / PROJECT.md § Constraints (single `X-Auth-Token` per org+person).
**Description**: The write authenticates with the `X-Auth-Token`. A leak in output, a diagnostic, or the request body would expose the credential.
**Severity**: High — a leaked credential is a serious exposure that grants the holder the caller's full governance permissions.
**Probability**: Low — structurally controlled (see RC-8) and inherited from every landed read/write.
**Risk Level**: Yellow (High × Low).
**Controls**:
- **RC-8**: The token rides 007's transport header only — it is never a request-body field (the body is `{proposal:{tension_id, changes}}`, no credential) and never a model field; `runProposalCreate` never reads `ctx.Cred.Token`; the `Proposal` model and the `proposal` render view carry no token; `reportFailure`/the diagnostics never print it.
**Residual Risk**: Yellow (justified) — high-severity by nature, but the structural "token only in the transport header" invariant drives probability to Low. Acceptable with the documented control; the no-secret-anywhere invariant is asserted across the landed test suite.

### H-8: `stdin`/file source read failure
**Source**: interface-cli.md § Change-set sourcing ("rejects a terminal stdin or an empty pipe"; "rejected if unreadable or not a regular file"); plan.md ADR-2.
**Description**: `--changes stdin` against a terminal (no pipe) or an empty pipe, or a file source that is unreadable or a directory, would otherwise produce a confusing empty/garbage change set.
**Severity**: Low — a clean pre-request usage error; nothing is sent or created.
**Probability**: Low — the 035/006 TTY/empty fail-fast and the regular-file guard are reused, landed and tested.
**Risk Level**: Green (Low × Low).
**Controls**:
- **RC-9**: The reserved `stdin` arm reuses the bounded reader's TTY/empty fail-fast (piped-stdin-required, empty-pipe rejected); a file source is rejected if unreadable or not a regular file. Each failure is a `UsageError(2)` naming the source (for stdin, how to pipe), raised before assembly with no request.
**Residual Risk**: Green — the source-read failures are caught fail-fast with a source-named diagnostic.

### H-9: Mis-surfaced non-2xx response
**Source**: interface-cli.md § Error Communication (the full status table); interface-spec.md § Error Communication.
**Description**: A `404` (unknown anchor tension), `422` (rejected change/field), `5xx`, or a `2xx` body that does not match the expected shape — if mis-surfaced as success or as an opaque failure — would mislead the agent into thinking a proposal was created when it was not (or vice versa).
**Severity**: Medium — a create failure reported as success would mislead the downstream advance step into referencing a non-existent `prp_` id.
**Probability**: Low — reuses the landed, tested `reportFailure` → `classifyClientError`/`ExtractProblem` chain; no new outcome or interpretation is added.
**Risk Level**: Green (Medium × Low).
**Controls**:
- **RC-10**: Every non-2xx routes through `reportFailure` → `classifyClientError`/`ExtractProblem` (015), naming the HTTP status + RFC 9457 detail and exiting non-zero (`404`/`422`/`5xx` → `APIError(3)`, `401`/`403` → `PermissionError(4)`, `429` → `RateLimited(5)`); a `2xx` body that fails to decode is a `*DecodeError` → `APIError(3)`, never a false success. The command adds no interpretation of its own.
**Residual Risk**: Green — the failure mapping is inherited and frozen; no new outcome/exit-code case is introduced.

### H-10: Partial output on render failure
**Source**: plan.md § Cross-cutting (Output: "Buffer-then-write so a render failure leaves stdout empty and maps to `RuntimeError(1)`"); interface-spec.md § Surface.
**Description**: A successful create followed by a failed render of the created proposal could otherwise leave partial/corrupt bytes on stdout, which an agent might mis-parse.
**Severity**: Low — the proposal *was* created (recoverable as a draft); the risk is a confusing local output, not a wrong write.
**Probability**: Low — buffer-then-write is the landed output discipline.
**Risk Level**: Green (Low × Low).
**Controls**:
- **RC-11**: Output is buffered then written, so a render failure leaves stdout empty and maps to `RuntimeError(1)` rather than emitting a half-rendered proposal.
**Residual Risk**: Green — the create still happened server-side; the deferred reads provide the detection path, and the empty-stdout-on-failure discipline prevents a mis-parse.

---

## Residual Risk Summary

| Level | Count | Hazards |
|---|---|---|
| Red (unacceptable) | 0 | — |
| Yellow (justified) | 2 | H-1, H-7 |
| Green (accepted) | 8 | H-2, H-3, H-4, H-5, H-6, H-8, H-9, H-10 |

**Unacceptable risks**: None. All residual risks are Green, or one of the two justified Yellows (H-1 free-form pass-through — a deliberate, draft-bounded design boundary; H-7 token hygiene — controlled structurally).

**Already mitigated by the design** (no implementation action needed beyond conforming to the plan):
- **H-5 double-submit** — §133 `isSafeMethod` makes the `POST` non-retryable on `429`; the create cannot auto-duplicate.
- **H-1 bad-but-typed change** — verbatim `[]json.RawMessage` pass-through + server-owned `422`; the draft gate means nothing is enacted.
- **H-4 / H-8 fail-fast, no request** — the `type` floor and source-read guards reject before assembly, with a transport tripwire pinning "no request sent."
- **H-9 / H-7 / H-10** — inherited, frozen, landed chains (015 error mapping; 007 token-only-in-transport; buffer-then-write output).

**Needs the implementer's attention during build** (advisory, not blocking):
- **H-1** — resist adding partial per-`type` validation; the floor reads **only** `type`. Closing more is the deferred Unguided Change Construction capability.
- **H-5** — do **not** add a client-side auto-retry on the create `POST`; it would convert the inherent post-success-failure window into systematic duplicate governance proposals.
- **H-6** — do **not** add a client-side Premium pre-check; the command must always issue the request and surface the server `403`.

**Finding severities**: This skill assesses domain hazards and assigns risk levels (Red/Yellow/Green), not P0/P1/P2 finding severities. No P0/P1/P2 are assigned — that scale belongs to the checklist/analyze/validate Guardian skills, not risk. (checklist.md and analyze.md already exist for this spec and own that scale.)

---

## Traceability Index

### Hazards

| ID | Source |
|---|---|
| H-1 | spec.md § Behavioral Accord / Non-Behaviors; plan.md ADR-3, § Risks |
| H-2 | spec.md § Assumptions ([ASSUMED] id pass-through); interface-cli.md § Surface |
| H-3 | plan.md ADR-2, § Risks; interface-cli.md § Change-set sourcing |
| H-4 | spec.md § Behavioral Accord (Input); plan.md ADR-3 |
| H-5 | plan.md § Cross-cutting (Non-idempotent retry §133), § Risks; interface-cli.md § Non-idempotent retry |
| H-6 | plan.md § Cross-cutting (Premium gate), § Risks; interface-cli.md § Premium gate; PROJECT.md § Constraints |
| H-7 | interface-spec.md § Error Communication; CONSTITUTION II / PROJECT.md § Constraints |
| H-8 | interface-cli.md § Change-set sourcing; plan.md ADR-2 |
| H-9 | interface-cli.md § Error Communication; interface-spec.md § Error Communication |
| H-10 | plan.md § Cross-cutting (Output: buffer-then-write); interface-spec.md § Surface |

### Controls

| ID | Mitigates | Grounding |
|---|---|---|
| RC-1 | H-1 | API Error Extraction (015) `422` surfacing — server owns per-change validation (plan ADR-3) |
| RC-2 | H-1 | Verbatim `[]json.RawMessage` byte-for-byte body (interface-spec § Surface; plan ADR-3) |
| RC-3 | H-2 | Created proposal echoes anchor `tension_id` (interface-cli § Output); `ExactArgs(1)`; API `404`/`422` on bad id |
| RC-4 | H-3 | `resolveChangesSource` fixed order + regular-file guard (plan ADR-2; interface-cli § Change-set sourcing) |
| RC-5 | H-4 | Pre-request `type` floor + transport tripwire (plan ADR-3; interface § Interactions); server `422` backstop |
| RC-6 | H-5 | `isSafeMethod` POST-no-retry (§133; plan Cross-cutting; interface-cli § Non-idempotent retry); exit-code outcome (004) |
| RC-7 | H-6 | 015 `PermissionError(4)` + RFC 9457 detail (plan Cross-cutting; interface-cli § Premium gate) |
| RC-8 | H-7 | Token-only-in-transport invariant (007); body carries no credential; `reportFailure` token-free (032); never reads `ctx.Cred.Token` |
| RC-9 | H-8 | Reused 035/006 TTY/empty fail-fast + regular-file guard (plan ADR-2; interface-cli § Change-set sourcing) |
| RC-10 | H-9 | `reportFailure`/`classifyClientError`/`ExtractProblem` (015/032); `*DecodeError` → `APIError(3)` |
| RC-11 | H-10 | Buffer-then-write output discipline (plan Cross-cutting § Output) |
