# Plan: Legacy Identifier Request

**Feature**: 075-legacy-identifier-request
**Role**: Shaper
**Inputs**: spec.md (075), PROJECT.md, DECISIONS.md (157 entries), LEARNINGS.md (2026-08-05 refresh entries S1–S11, V1–V11), DEPRECATION.md (8 entries), vendored `spec/glassfrog-api-v5.yaml` (IncludeLegacyId parameter and its six `$ref` sites), landed code in `internal/cli`, `internal/glassfrog`, `internal/paging`, `internal/render`

---

## System Architecture

The capability is a thin opt-in threaded through four existing layers, plus one guard. No new package, no new command.

**Request side** (`internal/cli`): a command-local `--legacy-id` boolean flag on the four cobra leaves that own the six supported reads — `roles` (list + single branches), `tree`, `actors` (list + single branches), `me`. When set, the command adds `include_legacy_id=true` to the request's `url.Values`. For walked lists, `paging.All` already deep-clones `req.Query` per page (the 033 `q`-param precedent), so every page of a walk carries the parameter with no paging change.

**Model side** (`internal/glassfrog`): the typed models the human/template path decodes grow a nullable pointer field for the number — `Role`, `Actor`, `Organization`, `Membership`, and `TreeNode`. Absent and `null` both decode to nil, which is sufficient because the render layer knows whether the flag was passed (see ADR-3); no tri-state decoding is needed.

**Output side**: the structured path (`json`/`yaml`) needs **zero changes** — it serializes the raw response bytes (018), so the faithful-echo accord (key exactly where the API put it, integer or explicit null, and no key wherever the response omitted one) is satisfied by construction the moment the parameter is sent. That is also why it needed no revision when the embed behaviour turned out to vary per read. The human path (`internal/render` templates) gains conditional sections: each affected template shows the number beside the stable identifier when it was requested, renders the settled absence idiom when requested-but-nil, names the agent backing on agent-backed actors, and — on the reads whose embeds omit the number — states that once per embedded group. A requested-bit threads from each command's flag state into the render data (the existing `Requested`-map pattern on `role.full`). User templates address the decoded typed structs, so they see the new fields wherever the structured output carries the number — including `Membership`, which the built-in human render deliberately omits.

**Guard side** (`internal/build`): a best-effort drift guard derives the set of operations that reference the `IncludeLegacyId` parameter from the vendored `spec/glassfrog-api-v5.yaml` and cross-checks it against the CLI's registration set. This is the capability's retirement tripwire: when a spec refresh drops or widens the parameter, CI fails and forces the revisit the spec's retirement-clock constraint demands.

Flow: flag → query param → (structured: raw bytes out unchanged) / (human: typed decode → template with requested-bit) → exit codes and diagnostics untouched.

---

## Architecture Decisions

### ADR-1: `--legacy-id` registers command-local on the four supported leaves; refusal on unsupported reads is cobra's unknown-flag error

**Context**: The spec requires the option on exactly the six supported reads and a pre-request refusal that names the option everywhere else. The six reads live on four cobra commands (`roles` and `actors` each carry list + single branches, both branches contract-supported, so no cross-branch `validateRolesFlags` entry is needed). `--base-url` and `--output` are persistent root flags by precedent; sibling read-shaping flags (`--include`, `--first-page`, `--per-page`) are command-local.

**Options considered**:
1. **Persistent root flag + per-command allowlist validation** — one registration, but every unsupported command must actively reject it, meaning a new validation table, new tests per command, and a fail-open risk on every future command that forgets the check.
2. **Command-local registration on exactly the four leaves** — cobra rejects `--legacy-id` anywhere else with `unknown flag: --legacy-id`, which is already a `UsageError(2)` before any request and names the option verbatim.

**Decision**: Option 2 — command-local registration. The refusal accord is satisfied by the framework's existing behavior; the supported set is expressed by where the flag exists, not by a parallel allowlist.

Registration uses `Flags()`, not `PersistentFlags()` — critical on `me`, whose subcommands (`me roles`, `me actions`, `me projects`) are contract-excluded and must keep rejecting the flag. The flag's help text carries the retirement caveat (transition-only, will be withdrawn, not a stable identifier) — the single caveat surface the spec's clarifications fixed.

**Consequences**: Adding a future supported read means one flag registration plus the guard-set update (ADR-4 makes forgetting it a CI failure). No new validation code anywhere. The unknown-flag message shape is cobra's, not ours — acceptable because it names the flag and exits 2 pre-request, which is exactly what the accord requires.

### ADR-2: Faithful echo by construction — the parameter is the only request-side change; no output-curation layer exists on the structured path

**Context**: The spec's governing principle: structured output is a faithful echo of the response shape (key exactly where the API carried it, no synthesis, no filtering, no added reason fields); the human render curates. Landed decision 018 serializes raw response bytes for `json`/`yaml`, never re-encoded structs.

**Options considered**:
1. **Decode-and-re-encode with explicit legacy handling** — gives the CLI a place to normalize or annotate, but breaks 018's fidelity contract and *creates* the curation layer the spec forbids.
2. **Send the parameter; let 018's raw-bytes path do the rest** — zero structured-output code; fidelity is inherited, not implemented.

**Decision**: Option 2. The structured half of the Surfacing accord, the membership number on `me`, the omitted key on embeds, and the explicit-null-vs-no-key distinction all fall out of the raw bytes. The only structured-path work is scenario coverage proving it.

**Consequences**: The `Membership` number reaches structured consumers with no membership render work. If a future spec ever wants curated structured output for this field, it must first overturn 018 — which is the right friction. Test effort concentrates on the human path, where all the actual behavior lives.

### ADR-3: Typed models grow a nullable `LegacyID` pointer; the human render distinguishes states via a requested-bit in the render data, not tri-state decoding

**Context**: The human render must distinguish "not requested" (render unchanged) from "requested, absent/null" (settled absence idiom, agent-backing reason on actors). JSON decoding into Go collapses absent and `null` into one nil. The `me roles` tri-state lesson (065) warns against deriving a boolean the data cannot support — here the disambiguating fact (was the flag passed?) is *caller state*, not response state.

**Options considered**:
1. **Tri-state decode** (`json.RawMessage` or custom unmarshal distinguishing absent from null) — faithful to the wire but pointless: the human render's branch is keyed on the flag, which the command already knows; the structured path never decodes typed structs at all.
2. **Nullable pointer field + a requested-bit threaded into render data** — `LegacyID *int64` on `Role`, `Actor`, `Organization`, `Membership`, `TreeNode`; each affected render call site passes the flag state alongside the decoded value, following the `Requested` include-map pattern already on `role.full`.

**Decision**: Option 2. The pointer covers present-vs-nil; the requested-bit covers shown-vs-hidden; the actor's `Kind` (already decoded) covers the agent-backing reason. Nothing new is inferred from absence.

`*int64` rather than `*int`: the contract types it `integer` and every observed value is an 8-digit integer (LEARNINGS W5) — the wide type is free headroom.

**Decode tolerance** (developer decision, 2026-08-08): the field decodes from **either a JSON integer or a JSON string**, via a small custom unmarshaller scoped to this one field. Rationale: V10 records the sibling spelling `databaseId` arriving as a string at one level and an integer in nested children *of the same payload*, and this field is explicitly transitional and therefore less likely to be tightly maintained upstream. Failing an entire read because the operator asked for an *optional* extra is the worse outcome. This is cheap insurance, not a fix — no string spelling was observed anywhere across the six probed reads (W5). The tolerance is scoped to `legacy_id`; it softens no other decoding. A value that is neither integer nor string still fails the decode, so a genuine contract break is still loud.

**Grounding note**: `TreeNode` is included on **observed** evidence, not contract evidence. The vendored schema omits `legacy_id` entirely, but the live tree read returns it on all 1336 nodes to depth 6 (LEARNINGS W1) — the schema is defective. This is recorded here because a future reader comparing the model to the contract will find the field unsupported and be tempted to remove it.

**Consequences**: Five model structs change (additive; decode-only, no encode path exists for them). Every affected render data struct gains a bit or map entry, which touches settled render call sites — mitigated by the byte-identical-when-not-requested scenario pinning the default render. User templates gain access to the fields on all five models, honoring the spec's template assumption without template-specific code.

### ADR-4: An `internal/build` drift guard derives the `IncludeLegacyId` operation set from the vendored spec and cross-checks the CLI's registration set — the capability's retirement tripwire

**Context**: The number is a declared transition bridge (S3) that retires with the v3 API, and S7 established that neither `info.version` nor the vendor changelog signals drift — only diffing the vendored file does. The spec assigns retirement detection to "the check that diffs the vendored contract" and forbids any runtime signal. 072 landed the citation-integrity pattern: derive BOTH sides at test time, fail CI on disagreement.

**Options considered**:
1. **No guard — rely on the refresh workflow's manual diff** — zero code, but the 08-05 refresh proved a three-month drift goes unnoticed precisely when nobody diffs; the spec's Integration Boundaries name a sibling check this option leaves not existing for this parameter.
2. **A guard deriving the parameter's `$ref` sites from `spec/glassfrog-api-v5.yaml` and asserting they match the operations behind the CLI's `--legacy-id` registration** — extends the landed 062/072 config-drift family; both sides derived, no hard-coded SoT copy.

**Decision**: Option 2. On retirement (parameter removed), the guard fails at the refresh PR and forces the deliberate removal decision. On widening (new operations gain the parameter), the guard fails and surfaces the coverage question instead of leaving the CLI silently narrower than the contract.

The guard derives the spec side by parsing the vendored YAML for `IncludeLegacyId` references per operation, and the CLI side from a declared contract-fact list adjacent to the registrations (the 067-boundary nuance: the *sets* stay derived; the mapping of operation-id → command is a checked-in contract fact, since the YAML cannot know cobra names).

**The guard anchors the parameter, deliberately NOT the response schema.** An invariant asserting "every mapped operation's response schema declares `legacy_id`" is the obvious-looking strengthening, and it is **wrong**: `TreeNode` omits the field while the runtime returns it (LEARNINGS W1), so such an invariant would fail permanently on an upstream defect this repo cannot fix, and the only way to go green would be to drop a working read. A guard whose sole passing state is the removal of correct behavior is worse than no guard. Response-schema capability is therefore tracked as an **observed-exception register** carrying probe evidence rather than as a hard assertion — see interface-spec.md invariant 5.

**Consequences**: A spec refresh that touches this parameter cannot merge without acknowledging it — the retirement clock has an alarm. Cost: YAML-parsing test code in `internal/build` (precedented; the citation-integrity guard already parses this file). The guard is best-effort (062 wording): it protects coverage claims, not runtime behavior — and W1/W2 are the standing proof that those two are independent facts.

---

## Render Design

The human-path work, per template family (protocol-level line shapes belong to interface; this fixes which templates change and what each must express):

Which groups get an embed note is settled **per read by observation** (LEARNINGS W2/W3), never by a global rule:

- **`roles` list (`roles.full`/`roles.compact`)**, **single (`role.full`/`role.compact`)**: number beside the stable id on each role that is the read's subject. On `role.full`, the embedded groups get the once-per-group embed note — verified: `subroles`, `assignments`, `fillers`, `accountabilities`, and `domains` all come back without the field while the role itself carries it (W3).
- **`tree` (`tree.full`/`tree.compact`)**: number on every row, at every depth. Each row is a subject of the tree read, not an embed — verified on 1336 nodes to depth 6 (W1), which is the grounding, since the response schema omits the field.
- **`actors` list (`actors.full`/`actors.compact`)**, **single (`actor.full`/`actor.compact`)**: number beside the stable id; nil + `kind=agent` renders the agent-backing reason; nil + human renders the plain absence idiom. Embedded Roles/Assignments groups get the once-per-group note (same endpoint family as `role.full`; not separately probed — the note's wording states what the read carries rather than what exists, so it stays true either way).
- **`me` (`me.full`/`me.compact`)**: numbers for the caller's actor and the organization; membership number deliberately not rendered (structured carries it). **The `?include=roles` group gets NO embed note and renders the number on each embedded role** — verified: `me`'s embedded roles carry `legacy_id` while `getRole`'s `subroles` do not, for the identical `Role` schema (W2). An embed note here would have been a false statement in rendered output.

Templates branch on the requested-bit so the not-requested render stays byte-identical — that invariant is pinned by an existing-output regression scenario.

---

## Cross-cutting Concerns

**Error handling**: none added. The flag changes no failure path — `reportClientError`/`Diagnose` (031/032) are untouched; the Failure accord group is satisfied by not wiring anything into it. The only new error surface is cobra's pre-request unknown-flag `UsageError(2)` on unsupported reads, which already exists.

**Configuration**: deliberately none. The spec's non-behavior forbids env/rcfile persistence for this flag; nothing registers in `internal/resolve`. The flag is per-invocation cobra state only.

**Testing strategy**: BDD scenarios from the spec's driving scenarios (feature files via the scenarios skill); unit tests per command for query-param assembly and flag registration; render tests for each changed template (requested/not-requested × present/null × human/agent); a structured-fidelity test asserting raw-byte pass-through with the key present, null, and absent; the ADR-4 guard as its own `internal/build` test. The four held-out validation scenarios stay held out.

**Observability**: nothing — no diagnostic, no stderr note, per the spec's clarified silence on retirement.

---

## Implementation Strategy

**Phase 1 — opt-in plumbing and structured fidelity**: flag registration on the four leaves (help text with retirement caveat), query-param assembly on all six read paths (list walks inherit per-page carry), the five model fields, and the structured-path scenarios. After this phase the structured half of the accord is fully done and human output is unchanged (flag accepted, number decoded but not yet rendered — phase boundary is internal, not shippable alone).

**Phase 2 — human render** (depends on Phase 1): requested-bit threading into the affected render data structs, template changes per the Render Design section, absence/agent-reason/embed-note rendering, byte-identical-when-not-requested regression pin.

**Phase 3 — retirement tripwire** (independent of Phase 2, needs Phase 1's registration set to exist): the `internal/build` guard deriving spec-side `$ref` sites and cross-checking the CLI-side contract-fact list.

Phases 2 and 3 can proceed in parallel once Phase 1 lands; a single PR carrying all three is acceptable at this feature's size — the phases exist so tasks can slice cleanly, not because separate PRs are required.

---

## Risks

- **The bridge retires mid-flight or early** (likelihood: low near-term — tied to v3 API retirement; impact: low): reads keep succeeding by design, numbers stop arriving, and the ADR-4 guard fails on the refresh that records it. No runtime mitigation needed — that is the specified behavior.
- **Live behavior diverges from the contract** (likelihood: **materialised** — three instances found; impact: was medium, now largely retired): this risk was rated medium and then **realised during `/score:guard --pre`**, exactly where predicted. All six reads were probed (LEARNINGS W1–W6): the tree read returns the field its schema omits (W1), `me`'s embedded roles carry it while `getRole`'s embeds do not (W2) — which falsified the blanket embed note this plan originally specified — and all five `getRole` embed families were confirmed clear (W3). Structured output absorbed every surprise for free, as the faithful-echo principle predicted. Residual: the agent-backed-null behavior is still contract-only and unverifiable here (W6, no agent actors in the org), so the absence-reason text is fixture-tested; and `actor.full`'s embed groups were not separately probed, mitigated by wording the note as what the read carries rather than what exists.
- **Render-data threading regresses settled templates** (likelihood: medium; impact: medium — the renders are contract for agent consumers): every affected template is touched. Mitigation: the byte-identical-when-not-requested scenario pins the default render per resource; render unit tests cover the new branches.
- **The guard over-triggers on refresh noise** (likelihood: low; impact: low — CI friction only): the parameter's `$ref` sites could move textually without semantic change. Mitigation: derive by parsed structure (operation → parameter refs), not by line grep — the 072 guard already established the parsing approach on this file.

---

## What This Plan Does Not Cover

- **Protocol-level contracts** — exact flag help wording, template line shapes, the absence idiom's exact rendering, the embed-note text, and the guard's contract-fact file format are the interface skill's concern (this feature has CLI + specification boundaries, so interface runs next).
- **Executable scenarios** — the scenarios skill translates the spec's driving scenarios into feature files; the held-out validation scenarios stay out of them.
- **Task decomposition** — the tasks skill slices the three phases into PR-sized units.
- **Orientation-skill updates** — none needed: 062 defers per-command flag detail to `glassfrog <command> --help`, which this feature's help text satisfies; no operator-path skill composes these reads in a way that changes.
- **Consuming the number** — the drafting path's use of the number as a change target, and any fallback-prompt behavior, live in #77/#82's specs, not here.
- **The accountability/domain/policy residue** — stays with the harvest route (#78), per the Feature Model's complement split.
