# Risk: Actor Read

**Feature**: 049-actor-read
**Round**: 1
**Date**: 2026-06-13
**Artifacts loaded**: spec.md, plan.md, interface-cli.md, tasks.md, PROJECT.md
**Degradation flags**: No Regulatory Context in PROJECT.md — IEC 14971 bridge omitted. No project risk-acceptability matrix — using the default 3×3 traffic-light matrix.

---

## Risk Register

| H-ID | Hazard | Source | Severity | Probability | Controls | Residual |
|---|---|---|---|---|---|---|
| H-1 | Growing 048's `actors` from `cobra.NoArgs` to `MaximumNArgs(1)` regresses the 0-arg directory branch → `glassfrog actors` (no id) breaks or changes behavior | plan ADR-1 + § Risks (048 coordination); tasks T003; CONSTITUTION V | High | Low | RC-1 | Yellow |
| H-2 | A large embedded `roles`/`assignments` array tempts a page walk on the single read → an unintended second request / cursor follow on a single resource | spec § Non-Behaviors (no pagination); plan § Cross-cutting; interface § Interactions; CONSTITUTION VI | Medium | Low | RC-2 | Green |
| H-3 | A mode-separation cross-combo is silently accepted instead of rejected (a list filter sent with an id → wrong/ignored scope; `--include` without an id silently dropped) | plan ADR-1; interface § Surface (mode separation); CONSTITUTION III | Medium | Low | RC-3 | Green |
| H-4 | An unsupported `--include` value reaches the API and is silently dropped → an actor returned with no embed, indistinguishable from "this actor has no roles" | spec § Related resources; plan ADR-3; CONSTITUTION I/III | Medium | Low | RC-4 | Green |
| H-5 | A nullable embed field (a role's purpose / empty accountabilities / domains) renders as a placeholder or invented value → fabricated governance footprint | plan ADR-4; interface § Output; CONSTITUTION VIII | Medium | Low | RC-5 | Green |
| H-6 | A `404` for an unknown id is reported as success or an empty result rather than a failure → "no such actor" mistaken for a clean read | spec § Failure; interface § Error Communication; CONSTITUTION III | Medium | Low | RC-6 | Green |
| H-7 | An `agt_` read is routed through the `ai_integration`-gated `/agents/{id}` alias → agent drill-in fails (403) for orgs without the feature, instead of reading the ungated `/actors/{id}` | spec § Non-Behaviors; plan § Permission scoping; CONSTITUTION I | Medium | Low | RC-7 | Green |
| H-8 | The API token leaks into stdout or an error line | plan § Error handling (never read `ctx.Cred.Token`); tasks T003; CONSTITUTION II | High | Low | RC-8 | Yellow |
| H-9 | The read-shaped `actors <id>` mutates as a side effect (a stray non-GET, or an unintended `?include`/write reaching the actor's `PATCH`/`DELETE` surface) | spec § Non-Behaviors; plan § What This Plan Does Not Cover; CONSTITUTION IX | High | Low | RC-9 | Yellow |
| H-10 | The single read surfaces an actor (or footprint) beyond the caller's membership → over-exposure of people/governance | spec § Failure; PROJECT.md Constraints (single org + person); CONSTITUTION I | Medium | Low | RC-10 | Green |
| H-11 | The optional-positional command shape forecloses `actors <id> assignments`, mis-shaping the deferred Actor Assignments (050) surface | plan ADR-1 § Consequences + § Risks (050 foreclosure) | Low | Low | RC-11 | Green |

No residual risk is Red. Three Yellows (H-1, H-8, H-9) — each acceptable with its documented control (see detail).

---

## Hazard Detail

### H-1 — Growing 048's command regresses the directory branch
This is the one genuinely new structural move: 048's **already-landed** `actors` command (merged as #90 on `main`) has its `Args` widened from `cobra.NoArgs` to `cobra.MaximumNArgs(1)` and its `RunE` branched on `len(args)`. If the growth disturbed the 0-arg path, the directory list (048) would break or change behavior — a regression of a shipped read introduced by an unrelated sibling, the entanglement CONSTITUTION V guards. **Severity High** (a working, validated command regresses); **Probability Low** — the plan mandates the directory branch is *preserved verbatim* (the existing `runActorsList`), the growth only widens the arg validator and adds a `len(args)==1` branch, and a held scenario re-asserts the 0-arg list still works (plan ADR-1; tasks T003 acceptance "`glassfrog actors` with no id still lists the directory and exits 0").
- **RC-1**: The 0-arg branch reuses 048's `runActorsList` unchanged — `RunE` branches, it does not rewrite; the arg validator only widens `NoArgs` → `MaximumNArgs(1)`. The directory-still-lists scenario ("The command with no id still lists the directory") is an executable guard. 048 is merged on `main` (#90), so this is an edit against a stable landed command, not a cross-spec race (plan ADR-1 § Consequences; § Risks first bullet; tasks T003/T004).
- **Residual: Yellow** (High×Low) — acceptable: the high severity reflects the consequence of a regression *were* it to occur; the directory branch is preserved verbatim and re-asserted by an executable scenario, holding probability Low.

### H-2 — A walk tempted by a large embedded array
`GET /actors/{id}` is a single resource, but `--include roles`/`--include assignments` can return large inline arrays — and an actor "who fills many roles" is exactly the held-out validation case. If the embedded array were mistaken for a paginated collection and a cursor followed (or a second request issued), the read would stop being the single, bounded drill-in the spec promises, and could partial-truncate a footprint without signalling (the CONSTITUTION VI silent-truncation hazard, in the embed dimension). **Severity Medium** (a wrong or partial footprint, or needless extra traffic); **Probability Low** — the plan instantiates no `Page[T]`/`paging.All` on this path and issues exactly one `Execute`; a held validation scenario records the traffic and asserts a single request even with many embedded roles.
- **RC-2**: The single read issues exactly one `Execute` into a `{data: ActorDetail}` document and follows no cursor; `Page[T]`/`paging.All` are not instantiated and `--first-page`/`--per-page` do not apply; the validation scenario "A single read issues exactly one request and no page walk" records the transport and asserts one request to `/actors/per_abc` with no cursor followed, and tasks T003/T004 carry a transport tripwire for it (spec § Non-Behaviors "must not follow pagination"; plan § Cross-cutting; interface § Interactions "single request, no walk").
- **Residual: Green** (Medium×Low).

### H-3 — A silently-accepted mode-separation cross-combo
The grown command carries two disjoint modes on one surface: list filters (`--kind`/`--role-id`/`--query`/`--first-page`/`--per-page`) are list-only; `--include` is single-only. If a filter supplied *with* an id were accepted (instead of rejected) it would be silently ignored (wrong scope, no signal); if `--include` supplied *without* an id were accepted, the embed request would be dropped against the directory list — either way a flag the operator set has no effect, with no error (the Fail-Safe-not-silent hazard). **Severity Medium** (a silently-ineffective flag → an agent assembles context it didn't actually request); **Probability Low** — the plan adds explicit pre-assembly cross-combo guards that fail fast as `UsageError(2)`, each pinned by a scenario.
- **RC-3**: The mode-separation guards run before any assembly: a list filter with an id, or `--include` with no id, is a fail-fast `UsageError(2)` with a transport tripwire asserting no request issued; the scenarios "A list filter combined with an id is rejected" and "A footprint include with no id is rejected" pin both directions, and the interface § Surface mode-separation table enumerates every combo (plan ADR-1; interface § Surface/Interactions; tasks T003 acceptance).
- **Residual: Green** (Medium×Low).

### H-4 — Unsupported `--include` reaches the API
`getActor`'s `include` is a closed enum (`roles`, `assignments`); the API silently ignores an unsupported value, returning the actor with no embed. Passed through, a typo'd `--include` would yield an embed-less actor indistinguishable from an actor that genuinely has no roles — the silent-wrong-results hazard 025 ADR-4 validates against locally. **Severity Medium** (a misleading "empty" footprint read as real); **Probability Low** — the CLI validates `--include` locally, reject-unknown, before any request.
- **RC-4**: A per-read `--include` validator rejects any value outside `{roles, assignments}` as `UsageError(2)` before assembly, naming the value and the alphabetically-sorted supported set (`assignments, roles`), with a transport tripwire; it is a *per-read set distinct from the role include set* (026 two-validator precedent), so it never accepts cross-read values the actor endpoint drops; omitting `--include` sends no param (plan ADR-3; interface § Surface/Consistency Notes; scenario "An unsupported include is rejected as a usage error").
- **Residual: Green** (Medium×Low).

### H-5 — Fabricated absent footprint field
The footprint render is the richest single-actor projection so far — each embedded role's name/purpose/accountabilities/domains, plus assignments — and any of those fields can be empty (a role with no purpose, no accountabilities, no domains). If an empty field rendered as a placeholder or invented string, the operator would read fabricated governance data. **Severity Medium**; **Probability Low** — the render uses 019's explicit-absence guards and prints absence markers, never blanks or invented values.
- **RC-5**: The `actor.full`/`actor.compact` templates render only the fields the API returned, each embed section present only when `?include`d and each nullable field behind an explicit-absence guard (`(no purpose set)`/`(none)`); no synthesized value is presented as real, and `name`/`purpose` are rendered verbatim (never truncated/reflowed); golden tests pin the bare actor, the footprint, and the absent-embed case (plan ADR-4; interface § Output; CONSTITUTION VIII; tasks T002 acceptance "an embedded role with no purpose renders the absence marker, not a blank").
- **Residual: Green** (Medium×Low).

### H-6 — Unknown id (`404`) misreported
A `404` for an unknown id must surface as a failure naming the status, never as a success or a misleading empty result — and the spec's third user scenario is precisely telling "no such actor" apart from "the network failed." **Severity Medium**; **Probability Low** — the shared classifier routes the non-2xx to a non-zero exit, and the id is passed through unvalidated so the API's own `404` is the single not-found signal.
- **RC-6**: A `404` flows through the shared `classifyClientError` → `APIError(3)`, naming the HTTP status on stderr and exiting non-zero; the transport-vs-status distinction is the shared seams' (`NetworkUnavailable(6)` vs `APIError(3)`); the actor id is sent verbatim (no local id validation that could mask the API's clean not-found — 025 ADR-4); scenarios "An unknown id fails with the API status" and the held "A non-2xx status is surfaced, not reinterpreted" pin it (spec § Failure; interface § Error Communication; CONSTITUTION III).
- **Residual: Green** (Medium×Low).

### H-7 — Agent drill-in routed through the gated alias
`/agents/{id}` is `ai_integration`-gated and deferred; the unified `GET /actors/{id}` is ungated and resolves both `per_` and `agt_` ids. If an `agt_` read were implemented against `/agents/{id}` rather than `/actors/{id}`, agent drill-in would 403 for orgs without the feature — a capability the spec deliberately keeps reachable. **Severity Medium** (a working read becomes a hard failure for some orgs); **Probability Low** — the spec Non-Behavior and plan fix the single-`/actors/{id}`-endpoint shape with the id passed through on the path.
- **RC-7**: The command reads only the unified `GET /actors/{id}`, passing the `per_`/`agt_` id through verbatim; it never routes through `/agents/{id}` or `/people`. The scenarios "An agent read reaches the ungated unified endpoint" (behavioral) and the held "An agent drill-in never touches the gated alias" (tripwire) assert that `agt_` reads `/actors/{id}` and no request reaches the gated alias, even with a token lacking `ai_integration` (spec § Non-Behaviors; plan § Permission scoping; tasks T003 acceptance).
- **Residual: Green** (Medium×Low).

### H-8 — Token leakage
The `actors <id>` command sits on the live request path where `X-Auth-Token` is most exposed. **Severity High** (secret disclosure, CONSTITUTION II); **Probability Low** — the command never reads the token; auth is owned by 007's replay thunk and 005's discovery, reused unchanged.
- **RC-8**: The read never reads `ctx.Cred.Token`; the diagnostic chain (031/032) names a cause + next step and never includes the token; tasks T003 acceptance pins "no token in any output" across stdout and stderr on every branch (plan § Error handling "never read `ctx.Cred.Token`"; interface § Error Communication "the token value never appears in any message"; CONSTITUTION II).
- **Residual: Yellow** (High×Low) — acceptable with the no-token-path discipline + the output assertion.

### H-9 — Read command mutates as a side effect
`actors <id>` is a drill-in read; a path that issued a POST/PATCH/DELETE — or reached the endpoint's actor-administration verbs (`updateActor` PATCH / `deleteActor` DELETE) — would violate Writes-Require-Explicit-Intent and could touch live people/governance data. **Severity High** (an accidental mutation of governance/people data); **Probability Low** — the command is GET-only by construction; the spec Non-Behavior forbids create/update/delete, and only a `?include` query (no body, no write verb) is ever attached.
- **RC-9**: The command issues only `GET /actors/{id}` with an optional `?include`; no POST/PATCH/DELETE path exists on it; the spec Non-Behavior forbids creating/updating/deleting the actor, and `updateActor`/`deleteActor` are explicitly out of scope; a contract/acceptance test pins the request method/shape (spec § Non-Behaviors; plan § What This Plan Does Not Cover; CONSTITUTION IX).
- **Residual: Yellow** (High×Low) — acceptable: read-only by construction, GET-only request shape pinned by test; the high severity reflects the consequence *were* it to occur, not its likelihood.

### H-10 — Over-exposure beyond membership
The read returns an actor (and, embedded, the roles/accountabilities/domains they carry); the hazard is surfacing an actor or footprint the caller's membership shouldn't see. **Severity Medium**; **Probability Low** — the API enforces permissions per the token's membership server-side (PROJECT.md single-org-+-person constraint); the CLI renders what the API returned and second-guesses nothing.
- **RC-10**: The command issues only the defined `GET /actors/{id}` operation (Spec Fidelity I); the server is the single authority on visibility; the CLI adds no client-side filtering and invents nothing; a contract/acceptance test pins the request shape (spec § Failure; PROJECT.md Constraints; CONSTITUTION I).
- **Residual: Green** (Medium×Low).

### H-11 — 050 surface foreclosed by the optional-positional shape
cobra cannot distinguish an actor id from a subcommand name under an optional positional, so `actors <id> assignments` is impossible — Actor Assignments (050, `GET /actors/{id}/assignments`) must take a flag-based or separate-command surface. The hazard is purely cross-spec design input: if 050 were later designed assuming `actors <id> assignments`, that surface would not be buildable. **Severity Low** (a constraint on a *future* sibling's design, not a runtime fault of 049); **Probability Low** — the plan records the foreclosure explicitly as 050's design input, and the same pattern already governed #33/#34/#38 under 025 ADR-1.
- **RC-11**: The foreclosure is recorded as a cross-spec precedent in plan ADR-1 § Consequences and § Risks (050 foreclosure); 050 resolves its own standalone surface, distinct from this spec's `--include assignments` embed; a `/score:deprecate` candidate is noted to retire 048 ADR-1's `cobra.NoArgs` sub-decision. No control is needed on *this* spec — the item is carried for traceability of the foreclosure decision.
- **Residual: Green** (Low×Low, accepted) — not a hazard to 049's own behavior; recorded so 050's designer inherits the constraint deliberately, not by surprise.

---

## Residual Risk Summary

11 hazards, 11 controls, **0 Red**. Three Yellows: H-1 (directory-branch regression) is the spec's one genuinely new structural risk — growing 048's already-landed command (#90) — controlled to Low by preserving 048's `runActorsList` verbatim and re-asserting the 0-arg list with an executable scenario; H-8 (token leak) and H-9 (read-mutates) are the inherent read-surface pair shared with every sibling read (048 H-2/H-10), each fully controlled (no-token-path discipline + output assertion; GET-only construction + request-shape test) with High severity reflecting consequence-if-it-occurred, not likelihood. The data-integrity, scope, and routing Greens (H-3 mode-separation guards, H-4 local `--include` validation, H-5 no fabrication, H-6 clean-`404` surfacing, H-7 ungated `/actors/{id}` only, H-10 server-side membership) are controlled by the plan's ADR-1/ADR-3/ADR-4 decisions and pinned by scenarios. Unlike the 048 directory, there is **no walk-throttling or partial-as-complete Yellow** — the single resource issues exactly one request; the only walk-shaped concern (H-2, a large embedded array tempting a cursor follow) is controlled to Green by the no-`Page[T]` construction and a single-request tripwire. H-11 (050 foreclosure) is a recorded cross-spec design input, not a 049 runtime hazard. No hazard is unacceptable; nothing requires resolution before implementation — every hazard at Medium or above is controlled by a mitigation already present in the plan and tasks.

## Traceability Index

| ID | Traces to |
|---|---|
| H-1 | plan ADR-1 + § Risks (048 coordination); tasks T003; CONSTITUTION V |
| H-2 | spec § Non-Behaviors (no pagination); plan § Cross-cutting; interface § Interactions; CONSTITUTION VI |
| H-3 | plan ADR-1; interface § Surface (mode separation); CONSTITUTION III |
| H-4 | spec § Related resources; plan ADR-3; CONSTITUTION I/III |
| H-5 | plan ADR-4; interface § Output; CONSTITUTION VIII |
| H-6 | spec § Failure; interface § Error Communication; CONSTITUTION III |
| H-7 | spec § Non-Behaviors; plan § Permission scoping; CONSTITUTION I |
| H-8 | plan § Error handling (token never read); tasks T003; CONSTITUTION II |
| H-9 | spec § Non-Behaviors; plan § What This Plan Does Not Cover; CONSTITUTION IX |
| H-10 | spec § Failure; PROJECT.md Constraints; CONSTITUTION I |
| H-11 | plan ADR-1 § Consequences + § Risks (050 foreclosure) |
| RC-1 | plan ADR-1 (runActorsList preserved verbatim; 048 landed #90); tasks T003/T004; scenario (directory still lists) |
| RC-2 | spec § Non-Behaviors; plan § Cross-cutting (no Page[T]); interface § Interactions; validation scenario (one request, no walk); tasks T003/T004 |
| RC-3 | plan ADR-1; interface § Surface (mode-separation table); scenarios (filter-with-id rejected / include-without-id rejected); tasks T003 |
| RC-4 | plan ADR-3; interface § Surface/Consistency Notes (per-read validator); scenario (unsupported include rejected); 026 two-validator precedent |
| RC-5 | plan ADR-4; interface § Output (absence guards); CONSTITUTION VIII; tasks T002 golden tests (absent-embed marker) |
| RC-6 | spec § Failure; interface § Error Communication; CONSTITUTION III; scenarios (unknown id fails / non-2xx surfaced not reinterpreted); 025 ADR-4 (id passed through) |
| RC-7 | spec § Non-Behaviors; plan § Permission scoping; scenarios (ungated endpoint / never the gated alias); tasks T003 |
| RC-8 | plan § Error handling (never read ctx.Cred.Token); tasks T003 ("no token in any output"); CONSTITUTION II |
| RC-9 | spec § Non-Behaviors; plan § What This Plan Does Not Cover; CONSTITUTION IX; request-shape acceptance test |
| RC-10 | spec § Failure; PROJECT.md membership enforcement; CONSTITUTION I contract test |
| RC-11 | plan ADR-1 § Consequences + § Risks (050 foreclosure recorded as cross-spec input); /score:deprecate candidate (retire 048 ADR-1 NoArgs) |
