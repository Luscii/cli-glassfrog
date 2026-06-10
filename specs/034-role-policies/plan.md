# Plan: Role Policies

**Feature**: 034-role-policies
**Role**: Shaper
**Inputs**: spec.md (034), PROJECT.md, DECISIONS.md (33 entries), LEARNINGS.md (background), DEPRECATION.md (1 entry)

---

## System Architecture

Role Policies adds the **addressable policy read surface** to the Governance Reads slice. It is, architecturally, a near-exact sibling of Role Reads (025) and Organization Tree (026): two thin cobra commands in `internal/cli` that build a request, hand it to the proven read chain, and render the result. It introduces no new package and no new transport, pagination, error, or output machinery — every seam it needs is landed.

Two commands, each `ExactArgs(1)`, guard-registered (001) and explicitly wired in `main`:

- **`glassfrog policies <role-id>`** → `GET /roles/{id}/policies` (`listRolePolicies`) — a **paginated list** of the policies governing a role's interior. Walks to completion through `paging.All[Policy]` (016) by default, with a `--first-page` opt-out and an optional `--query`/`-q` free-text filter sent as the endpoint's `q` parameter.
- **`glassfrog policy <pol-id>`** → `GET /policies/{id}` (`getPolicy`) — a **single policy** decoded from a `{data: Policy}` document, carrying the full body.

Data flow per invocation (identical to 025/026): the command validates its inputs locally where the API would otherwise silently mislead, resolves the connection context once (`AssembleFromOS`), builds the `*apiclient.Client` (010), issues the request (list → `paging.All`; single → one `Execute`), and hands the typed result to the shared render dispatch (`renderResult[T]`, 020) which selects raw-bytes JSON/YAML (018) or the human `policy`/`policies` templates (019) by the resolved `--output` format. Typed client errors route through the one shared `classifyClientError` chain (011/015) — no new `Outcome` category, no new exit code.

The only genuinely new artifacts are: the **`policies`/`policy` commands**, a small **growth of the shared `glassfrog.Policy` model**, the **generalization of the single-object envelope** to a generic `Document[T]`, and **four human-render templates** (`policy` / `policies` × `full` / `compact`).

---

## Architecture Decisions

### ADR-1: Expose two sibling commands `policies <role-id>` and `policy <pol-id>`, not a `role` group or children of `roles`

**Context**: 025 ADR-1 established that the optional-positional `roles <id>` shape forecloses `roles <id> <subcommand>` — cobra cannot distinguish a role id from a subcommand name — so the per-role reads (#33/#34/#38) "need a singular `role <id> …` group or a flag-based surface." 026 then chose two new sibling commands (`tree`, `subroles <id>`) over creating that group and explicitly left #34 "free to choose its own surface." Role Policies has two reads keyed on **different id kinds**: a `role_` id selects a per-role list, a `pol_` id selects one policy. The spec confirmed the surface with the developer.

**Options considered**:
1. **One command, optional positional** (the `roles`/`tree` shape) — `policies [id]`. Rejected: the two reads take *different* id kinds and hit *different* endpoints; an optional positional cannot carry that distinction, and it would re-foreclose any future `policies <id> <sub>`.
2. **A `role <role-id> policies` group** — a singular `role` parent hosting per-role subcommands. Rejected for now: it forces #33/#38 to land the same shared group in lockstep, and 026 deliberately did not create it. A heavier coordination than the feature needs.
3. **Two sibling top-level commands** — plural `policies <role-id>` (the role-scoped list) + singular `policy <pol-id>` (the standalone read), both `ExactArgs(1)`. Chosen.

**Decision**: Option 3. `policies` and `policy` are two guard-registered, explicitly-wired sibling commands, each requiring exactly one positional id. The plural/singular pair carries the role-id-vs-policy-id distinction in the command name, exactly as 026's `subroles <id>` sits beside `tree`.

A structural consequence makes this cleaner than 025's single-command branching: the **list-only flags (`--query`, `--first-page`, `--per-page`) are registered only on `policies`**. Passing any of them to `policy` is rejected by cobra's own unknown-flag handling as a `UsageError(2)` before assembly — so the spec's "`--query` on the single read is a usage error" needs *no hand-rolled cross-combo guard* (unlike 025, where one command had to branch on `len(args)` and guard filter+id / include-without-id combinations explicitly).

**Consequences**: Sets the per-role-read surface precedent #33 (Role Domains) and #38 (Role Projects) can follow verbatim — `domains <role-id>`/`domain <dom-id>`, `projects <role-id>`/`project <prj-id>`. No `role` group is created. Cost: two more top-level commands in the registry (accepted — the CLI is already command-per-read). The `--base-url` and `--output`/`-o` persistent root flags (011/020) are inherited by both, inert nowhere since both produce results.

### ADR-2: Grow the shared `glassfrog.Policy` model and generalize the single-object envelope to `Document[T]`

**Context**: The landed `glassfrog.Policy` (025, `roles.go:101`) is minimal — `ID`, `Title`, `Body` — because its only consumer so far is the *embedded-on-role* view (`roles <id> --include=policies`), which renders title + body. The standalone `getPolicy` read's reason to exist is the full, addressable policy: the spec schema marks `role_id`, `domain_id`, `created_at`, `updated_at` as present, and a reader fetching a policy on its own benefits from knowing which role/domain it governs and when it changed. Separately, 025 decodes the single role from a *named* `RoleDocument{Data RoleDetail}` and its own comment says "a later read that wants the same shape may generalize it" — 034 is that second single-object read.

**Options considered**:
1. **Keep `Policy` minimal; add a second fuller type** for the standalone read. Rejected: violates 011 ADR-1 (one shared schema type, grown not duplicated) and would fork the policy projection.
2. **Grow `Policy` in place** (add `RoleID`, `DomainID`, `CreatedAt`, `UpdatedAt`) and **generalize `RoleDocument` → generic `Document[T]`**. Chosen.
3. Grow `Policy` but keep a named `PolicyDocument` beside `RoleDocument`. Rejected: a second hand-written single-object envelope is exactly what `Page[T]` (016) eliminated for lists; the `RoleDocument` comment already invites the generalization, and 034 is the natural trigger.

**Decision**: Option 2. Grow `glassfrog.Policy` with the four spec fields (nullable `role_id`/`domain_id` modeled as plain strings, mirroring the existing nullable `Body`; tolerant decoding ignores anything else). The list read decodes the generic `glassfrog.Page[Policy]` (016); the single read decodes a new generic `glassfrog.Document[T]` with `Document[Policy]`, and the landed `RoleDocument` is refactored to `Document[RoleDetail]` (or aliased) in the same change — mirroring how 016 generalized the per-resource list envelopes into `Page[T]`.

**Consequences**: The growth is purely additive for 025 — its embedded render reads only `ID`/`Title`/`Body`, untouched by the new fields. The `Document[T]` refactor touches one landed call site (025's single-role decode) and its tests; small and mechanical, and it pays forward to every future single-object read (#38's `project`, a future `note`/`skill` read). The new `RoleID`/`DomainID` fields become explicit-absence render targets (019 guard: a role-level policy has a null `domain_id`).

### ADR-3: Validate nothing the API reports cleanly — `--query` is free text passed through; the id is passed through to a clean 404

**Context**: 025 ADR-4 set the input-handling principle: *validate closed-enum inputs locally* (where a wrong value makes the API silently return wrong results), but *pass free identifiers through* (where the API reports cleanly, e.g. a `404`). Role Policies has no closed-enum input. `--query` is free-text search; the `pol_`/`role_` id is a free identifier, and `getPolicy`/`listRolePolicies` document only `401`/`404` (`getPolicy`) and `400`/`401`/`404` (`listRolePolicies`) — a bad id yields a clean not-found, not silent-wrong-results.

**Options considered**:
1. **Pre-validate the id format** (`^pol_…$` / `^role_…$`) locally. Rejected: the API returns a clean `404`/`400` for a malformed id, so local validation only duplicates a check the API already does loudly — and risks rejecting a future id shape the API would accept (025 ADR-4's "pass through where it reports cleanly").
2. **Pass both the id and `--query` straight through**, with the only local guard being structural (list-only flags absent on `policy`, per ADR-1). Chosen.

**Decision**: Option 2. `--query`'s value is sent verbatim as the `q` parameter on `listRolePolicies` (no enum check — it's a search string). The positional id is sent into the path unvalidated; a malformed or unknown id surfaces as the API's non-2xx, classified by the shared chain (`APIError(3)` / `PermissionError(4)` for 401/403). The list-only-ness of `--query` is enforced structurally by ADR-1 (the flag does not exist on `policy`).

**Consequences**: No new validator function; the read carries no `validateInclude`/`validateStatus`-style local check (025/013) because it has nothing closed-enum to validate. An empty `--query` value (`--query ""`) is omitted, not sent as `q=`: `q` is sent only when the flag is `Changed()` **and** non-empty (the 026 `--depth` optional-flag discipline), so the spec's "no `--query` → every policy" holds for both the absent and empty-value cases. A `400` from the list endpoint on a degenerate query classifies cleanly as `APIError(3)`.

### ADR-4: Add `policy` and `policies` human-render templates; structured output is unaffected

**Context**: 019 renders human output from `//go:embed` templates per resource key in `internal/render`, with explicit-absence guards (`—`/`(none)`) and golden tests; 020 dispatches between machine (`internal/output`, raw bytes) and human (`internal/render`, typed structs) by resolved format. There is no `policy`/`policies` template yet (existing keys: `me`, `roles`/`org-roles`, `role`, `subroles`, `tree`, `actions`, `projects`).

**Decision**: Add four embedded templates — `policy.full` / `policy.compact` (single, full body) and `policies.full` / `policies.compact` (list, title + id per row) — plus their view structs (`PolicyView`, `PoliciesView`) in `internal/render`, registered in the same set the registry-exhaustiveness guard checks (every resource has both formats, PR #10 shape). The single-policy `full` template renders the full body and the scope (`role_id`/`domain_id`) and timestamps with explicit-absence guards for the nullable fields; `compact` renders title + id. The list templates render one row per policy (and the inherited empty-set line, e.g. `No policies.`) and the incomplete-walk stderr note lives on the command, not the template (025 ADR-3).

**Consequences**: Structured `json`/`yaml` output needs no change — it serializes the raw response bytes (018 ADR-2), so the new commands get correct machine output for free. The single-policy template is the first to render a long free-text `Body` field as the primary content (prior templates render short projected fields); the template must not truncate or reflow it (CONSTITUTION VI — present API data faithfully).

---

## Cross-cutting Concerns

**List completeness** (silent conformance to 025 ADR-3): `policies <role-id>` defaults to `paging.All[Policy]` and reduces to `(records, complete)`; the `--first-page` opt-out does a single `Execute` into `Page[Policy]`. A deliberate opt-out with more pages exits `0` with a one-line "more available — re-run without --first-page" stderr note; a mid-walk failure renders the partial `Records`, writes "incomplete — <cause>" to stderr, and exits non-zero via `classifyClientError(Stop)`. Never silently truncates (CONSTITUTION VI). Adds no `Outcome`/`ExitCode`.

**Error handling**: identical to every read since 011 — typed client errors (`TransportError`, `ResponseError`, `AuthError`, `DecodeError`) route through the single shared `classifyClientError` chain; the not-authenticated fail-safe (007) refuses at send time; messages name the failure + a next step and never include the token. No new exit codes (3–6 already populated by 011/015).

**Testing**: pure-unit tests for the grown `Policy` decode (incl. null `role_id`/`domain_id`) and the `Document[T]` generalization; golden tests for the four templates (incl. a role-level policy with null domain, and the empty-list line); a `internal/cli` godog suite over a new `features/governance-reads/role-policies.feature` driven by a fake transport returning canned pages, with a **transport tripwire** asserting no request is issued when `policy` is given a list-only flag (the unknown-flag `UsageError` path) — the 011/013/025 offline-BDD pattern.

**Configuration**: none new. Reuses `--base-url`/`--output`/`-o` (root persistent) and the `--first-page`/`--per-page` list flags (016/025).

---

## Implementation Strategy

Single phase — the feature is one cohesive read pair with no internal dependencies, and every seam below it is landed. Suggested ordering for task decomposition:

1. **Schema** — grow `glassfrog.Policy` (+`RoleID`/`DomainID`/`CreatedAt`/`UpdatedAt`); add generic `Document[T]` and refactor `RoleDocument`→`Document[RoleDetail]` (+ update 025's decode call site and tests). Pure, no command surface.
2. **Render** — add the four `policy`/`policies` templates + view structs + registry entries; golden tests.
3. **Commands** — `policies <role-id>` (list: `paging.All[Policy]` + `--query`/`--first-page`/`--per-page`, the 025 completeness logic) and `policy <pol-id>` (single: `Document[Policy]`), guard-registered and wired in `main`; route through `renderResult[T]` and `classifyClientError`.
4. **BDD** — the `role-policies.feature` suite covering the spec's driving scenarios + the structural list-only-flag tripwire.

Phases 1–2 are independent and can land in either order; 3 depends on both; 4 depends on 3.

---

## Risks

- **`Document[T]` refactor touches landed 025 code** (low likelihood, low impact): the single-role decode and its tests change. Mitigation: keep `RoleDocument` as a type alias for `Document[RoleDetail]` if a clean rename proves noisy, so 025's call sites and BDD stay byte-stable; the generalization is mechanical and covered by 025's existing tests.
- **Policy `body` may contain HTML** (medium likelihood, low impact): the schema notes the body "may contain HTML". The human template renders it as-is (`text/template`, no escaping — 019); this is faithful, but a very long HTML body is verbose in `full`. Mitigation: `compact` omits the body; no truncation in `full` (faithfulness over brevity — the operator chose `full`).
- **Empty `--query` semantics** (low likelihood, low impact): sending `q=` if a user passes `--query ""`. Mitigation: send `q` only when the flag is `Changed()` and non-empty (026 `--depth` optional-flag discipline), so `--query ""` behaves as no filter rather than a degenerate API call.

---

## What This Plan Does Not Cover

- **Protocol-level contracts** — exact command/flag spellings, the `Document[T]`/`PolicyView` field names, the request-descriptor shape, and the template text are the **interface** skill's concern (`/score:interface`). The names used here (`policies`/`policy`, `--query`/`-q`) are the developer-confirmed surface; treat them as the working contract, not a re-decision.
- **Executable scenarios** — the `.feature` file is the **scenarios** skill's output; the Driving Scenarios in spec.md are the source.
- **Task decomposition** — PR-sized units are the **tasks** skill's output; the Implementation Strategy above is the input.
- **A `role` command group** — deliberately not created (ADR-1); if #33/#38 later argue for consolidating the sibling commands under a group, that is a coordinated cross-spec change, not this plan's.
