# Specification: Actor Read

**Feature**: 049-actor-read
**Role**: Definer
**Tier**: 1 (zero setup)

---

## System Overview

Actor Read is the **single-actor drill-in** of the Actor Reads slice. It offers one read: read a single actor — person or agent — by id (`glassfrog actors <id>` → `GET /actors/{id}` → `getActor`), optionally embedding the roles they fill and/or their assignments. Where Actor Directory (048) is the flat *discovery* read that turns a name or a role into an addressable `per_`/`agt_` id, Actor Read is the *drill-in* that id feeds: it is the entry to an actor's **governance footprint** — the roles they fill and the accountabilities, domains, and purposes those roles carry.

It is the per-id read that mirrors `glassfrog roles <id>` (the single-read half of Role Reads, 025) on the actor surface, and it surfaces the same `glassfrog.Actor` model that Identity Read (011) grew and Actor Directory (048) reuses — 011 reads the token-scoped `GET /me`, Actor Read reads any actor by id. It sits on the proven read chain rather than rebuilding it: it hands requests to **Request Execution (010)**, reads identity through **Request Authentication (007)**, lets **Output Format Selection (020)** render the result, and maps outcomes through **Exit-Code Convention (004)** and **API Error Extraction (015)**. `GET /actors/{id}` returns a single resource, so — unlike the directory — there is no page walk to perform. It sits ahead of Actor Assignments (050), the standalone, paginated list of the roles an actor fills.

---

## Behavioral Accord

### Invocation

- When the user runs `glassfrog actors <id>` with a single positional id, the system reads the single actor with that id and produces it as a single-actor result.
- When the id carries a `per_` (human) or `agt_` (agent) prefix, the system passes it through unchanged on the path and the API resolves it — the same command reads either kind of actor.
- When the user runs `glassfrog actors` with no positional id, that is the directory list (Actor Directory, 048), not this read — Actor Read owns only the single-id form.
- When the user passes more than one positional id, or passes an unknown flag, the system rejects the invocation as a usage error and calls no API.

### Related resources

- When the user supplies `--include` with one or more of the supported related resources — `roles`, `assignments` — the system sends them as the `?include=` query and embeds the returned related resources inline in the actor result.
- When the user supplies `--include roles`, the embedded roles are the actor's governance footprint: each role carries its name, purpose, accountabilities, and domains inline, so one read shows what the actor does.
- When the user supplies `--include` with a value outside that set, the system rejects it as a usage error before issuing any request, naming the unsupported value and the supported set — mirroring how Identity Read (011) and Role Reads (025) validate their own `--include`.
- When the user supplies no `--include`, the system reads the bare actor — id, name, kind, and timestamps — with no embedded resources.

### Output

- When a read succeeds, the system produces the actor data as its result and lets Output Format Selection (020) render it in the effective format (`full` / `compact` / `json` / `yaml`); it neither fixes raw API JSON as its default nor defines its own format flag.
- When the read succeeds, the command completes with the success result, so the operator (or a wrapping agent) can tell the call worked without inspecting output.

### Failure

- When no usable token is available, the system surfaces the authentication fail-safe's refusal and exits non-zero with a not-authenticated outcome, reusing the shared not-authenticated message and pointing the operator at how to store a credential.
- When the request cannot reach or complete at the wire (connection, DNS, TLS, timeout), the system surfaces the transport failure by name and exits non-zero with the network-unavailable outcome.
- When the API answers with a non-2xx response — including a single read whose id does not exist (typically `404`) — the system reports that the read failed, naming the HTTP status, and exits non-zero. The command adds no interpretation of its own; the shared error handling (API Error Extraction, 015) classifies the status — a generic API-error outcome, or the permission (`401`/`403`) and rate-limit (`429`) outcomes it already distinguishes — and the command surfaces whichever results.
- Whatever the failure, the message names both what went wrong and a concrete next step (Action Transparency), and never includes the token.

---

## User Scenarios

**In order to** see what an actor actually does — the roles they fill and the accountabilities, domains, and purposes those carry — once I know their id,
**as an** AI agent operating the CLI on a practitioner's behalf,
**I want to** read one actor with their roles embedded.

**In order to** drill into an id I found in the directory (048) or as a role's filler (047),
**as an** AI agent assembling context before acting,
**I want to** read a single actor — person or agent — by their `per_`/`agt_` id with one command.

**In order to** tell "no such actor" apart from "the network failed" when a read fails,
**as an** operator diagnosing a failed run,
**I want** the command to surface the status-versus-transport distinction the shared seams already draw.

---

## Non-Behaviors

- The single read must not add a list or search of actors of its own. **Why**: listing/searching the directory (`GET /actors`) is Actor Directory (048), the flat discovery read; Actor Read is the per-id drill-in. The grown `actors` command still serves 048's list at the no-id (0-arg) form, but the single-read (1-arg) branch introduces no new list/search surface — folding one in would blur the find-then-read split the slice is built on.
- The system must not provide a *standalone* assignments listing. **Why**: `--include assignments` embeds assignments *inline on the actor* as a convenience view; the addressable, paginated list of the roles an actor fills belongs to Actor Assignments (050). The two views coexist deliberately — embedded-on-the-actor vs. addressable-on-its-own — exactly as Role Reads (025) embeds related resources while the standalone per-resource reads live in their own specs.
- The system must not route an `agt_` read through the `ai_integration`-gated `/agents/{id}` alias. **Why**: `GET /actors/{id}` is ungated and resolves both `per_` and `agt_` ids, so agents are reachable through it without the feature flag; the dedicated `/agents/{id}` alias stays deferred (mirroring how Actor Directory, 048, reaches agents through `/actors?kind=agent`).
- The system must not follow pagination. **Why**: `GET /actors/{id}` is a single resource; any embedded `roles`/`assignments` arrays are returned inline in the one response, so there is no page walk — paging (016) owns the list reads that need it.
- The system must not create, update, or delete the actor. **Why**: Actor Read is a read surface; the endpoint's `PATCH` (`updateActor`) and `DELETE` (`deleteActor`) are actor administration, out of scope for the read-only Actor Reads slice.
- The system must not emit raw API JSON as a fixed default, nor define its own output-format flag. **Why**: Output Format Selection (020) resolves the format and dispatches to the 018/019 renderers; a private flag here would fork that contract.
- The system must not resolve the base URL or token, attach the `X-Auth-Token` header, decide the no-token fail-safe, type a non-2xx response, or choose its own exit codes. **Why**: Base URL Resolution (008), Credential Discovery (005), Request Authentication (007), API Error Extraction (015), and Exit-Code Convention (004) own those; a second path here would drift from their contracts.

---

## Integration Boundaries

- **Glassfrog API v5 (`GET /actors/{id}`)**: the system reads one actor through this endpoint. It accepts a `per_`/`agt_` path id and an `include` query (`roles`, `assignments`) and returns a single `{data: <actor>}` object — the API's Actor schema, which carries the optional `roles`/`assignments` embeds when `include` is set (decoded into the `ActorDetail` wrapper, not the base Go `glassfrog.Actor`) — not a paginated envelope. Data flows inbound (one actor in, nothing written); it answers `401` for an unusable token and `404` for an unknown id. When the endpoint is unreachable or answers non-2xx, the command surfaces a transport or read failure and exits non-zero.
- **Request Execution (010) / Request Authentication (007)**: the command builds the request and hands it to these seams to authenticate and execute; it does not re-implement transport or auth, and propagates the no-token fail-safe rather than deciding it.
- **Output Format Selection (020)**: receives the actor data as the command's result and renders it in the effective format. The command produces structured data, never pre-rendered output.
- **Exit-Code Convention (004) / API Error Extraction (015)**: the command maps its outcome to a process exit code through 004 and lets 015 classify a non-2xx status; it adds no interpretation of its own.
- **Identity Read (011) — sibling**: established the `glassfrog.Actor` model (`id`, `name`, `kind`, timestamps) and the reshaped-projection output this command reuses. 011 reads the token-scoped `GET /me` actor; this command reads any actor by id. The model is shared; the surfaces are distinct.
- **Actor Directory (048) — sibling**: owns `glassfrog actors` (the list/discovery read); Actor Read adds the `<id>` single-read form on the same command. The ids this command consumes come from 048's directory (or a role's fillers, 047).
- **Actor Assignments (050) — sibling**: owns the standalone, paginated list of the roles an actor fills; Actor Read's `--include assignments` embed is the inline convenience view, not that addressable surface.

---

## Driving Scenarios

### Happy path

**Scenario: Read a single actor by id**
Given a valid stored credential and an actor id that exists in the org
When the user runs `glassfrog actors <id>`
Then the system reads `GET /actors/{id}`
And produces a single-actor result carrying the actor's id, name, and kind
And exits successfully.

**Scenario: Read an agent by its `agt_` id**
Given a valid stored credential and an `agt_` id for an agent in the org
When the user runs `glassfrog actors agt_<...>`
Then the system reads `GET /actors/{id}` with the `agt_` id passed through
And produces the agent as a single-actor result
And exits successfully.

**Scenario: Read an actor with their governance footprint embedded**
Given a valid stored credential and an existing actor id
When the user runs `glassfrog actors <id> --include roles`
Then the system sends `include=roles` and embeds the actor's roles inline
And the result shows each role's name, purpose, accountabilities, and domains
And exits successfully.

### Error scenarios

**Scenario: No usable credential**
Given no stored credential and none in the environment
When the user runs `glassfrog actors <id>`
Then the system surfaces the not-authenticated refusal without calling the API
And exits non-zero
And the message points the operator at how to store a credential.

**Scenario: A single read for an unknown id**
Given a valid stored credential and an actor id that does not exist
When the user runs `glassfrog actors <unknown-id>`
Then the API answers `404`
And the system reports the read failed, naming the HTTP status
And exits non-zero.

### Edge cases

**Scenario: Unsupported `--include` value is rejected before any request**
Given a valid stored credential and an existing actor id
When the user runs `glassfrog actors <id> --include nonsense`
Then the system rejects it as a usage error, naming the unsupported value and the supported set
And issues no request
And exits with the usage-error code.

**Scenario: An agent read does not require the gated alias**
Given a token without the `ai_integration` feature and an `agt_` id
When the user runs `glassfrog actors agt_<...>`
Then the system reads the agent through `GET /actors/{id}` (the ungated unified endpoint)
And never routes through the deferred, `ai_integration`-gated `/agents/{id}` alias.

**Scenario: Read an actor with their assignments embedded**
Given a valid stored credential and an actor id who fills several roles
When the user runs `glassfrog actors <id> --include assignments`
Then the system sends `include=assignments` and embeds the assignments inline on the actor result
And exits successfully.

---

## Validation Scenarios

> These are held out from the implementing agent for independent verification.

**Scenario: Agent drill-in does not route through the gated alias**
Given a token without the `ai_integration` feature
When the user reads an agent by its `agt_` id
Then the system reaches the agent through `GET /actors/{id}` (the ungated unified endpoint)
And never routes through the deferred, `ai_integration`-gated `/agents/{id}` alias.

**Scenario: Output is structured, not pre-rendered**
Given any successful single-actor read
When the result reaches Output Format Selection (020)
Then the command supplied structured actor data and defined no format flag of its own, so all four formats (`full` / `compact` / `json` / `yaml`) render from the same result.

**Scenario: The single read issues no page walk**
Given a successful `glassfrog actors <id> --include assignments` run whose actor fills many roles
When the request traffic is inspected
Then exactly one request is issued to `GET /actors/{id}`
And no pagination cursor is followed, even though the embedded assignments are a list.

**Scenario: A non-2xx status is surfaced, not classified**
Given the API answers `GET /actors/{id}` with a `404`
When the command runs
Then the system surfaces the read-failed outcome carrying the status
And does not turn it into a specific, interpreted message of its own (API Error Extraction, 015, owns classification).

---

## Assumptions

- **Command shape** `glassfrog actors <id>` `[ASSUMED]`: Actor Read adds the single positional-id form to the same `actors` command Actor Directory (048) registers, mirroring how Role Reads (025) pairs `glassfrog roles` (list) with `glassfrog roles <id>` (single). The exact command registration is an interface/001 detail; the behavior — one actor read by id — is fixed.
- **`--include` semantics**: reuses Identity Read (011) / Role Reads (025) opt-in, reject-unknown `--include` handling, validated against the spec's `getActor` include enum (`roles`, `assignments`) before any request. If the spec later adds include targets, the accepted set tracks the spec; this is a planning-adjustable detail, not a behavioral change. `[ASSUMED]` (the exact flag shape — repeatable vs comma-separated — is pinned at interface time).
- **`Actor` model is reused, not redefined**: the `glassfrog.Actor` type grown in Identity Read (011) and reused by Actor Directory (048) carries the base identity fields (`id`, `name`, `kind`, `created_at`, `updated_at`). The `getActor` response is that actor plus the optional `roles`/`assignments` arrays the include populates, so the single read decodes a `{data: <actor>}` document (one object, not the `{data: [...], meta}` list envelope), reusing the base `Actor` for identity. Because `glassfrog.Actor` itself has no embed fields, those arrays ride on a single-read decode type rather than the bare `Actor`; the exact type (the plan names it `ActorDetail`, embedding `Actor` + the two arrays) is a planning detail.
- **Projection field set**: the default projection surfaces the actor's id, name, and kind; when `--include roles` is given, it lists each embedded role's footprint facts (name, purpose, accountabilities, domains); when `--include assignments` is given, it lists the embedded assignments. The visual layout is a planning detail consistent with the agent-legible-output principle and Identity Read (011)'s projection.

---

## Ambiguity Warnings

_None — the feature follows the established 025 (single-read) / 048 (actor model + ungated-agent reasoning) / 011 (projection) pattern, and the boundary questions specific to Actor Read were resolved during specification: it is the `<id>` single-read form on the same `actors` command (the no-id list is 048); `--include {roles, assignments}` embeds related resources inline (the standalone, paginated assignments list is Actor Assignments, 050); and an `agt_` read goes through the ungated `/actors/{id}`, never the deferred `ai_integration`-gated `/agents/{id}` alias._

---

## Clarifications

### Session 2026-06-13

- **Command shape and the 048 boundary**: Actor Read is the `<id>` single-read form on the same `glassfrog actors` command that Actor Directory (048) registers for the list — mirroring `glassfrog roles` / `glassfrog roles <id>` in Role Reads (025). 048 today rejects a positional argument; Actor Read adds the single-read form, so the no-id form remains the directory and the one-id form becomes the drill-in.
- **Include set and the 050 boundary**: `--include` accepts `{roles, assignments}` (the `getActor` enum), embedding related resources inline as a convenience view. The standalone, addressable, paginated list of the roles an actor fills is Actor Assignments (050); the embedded-on-the-actor and addressable-on-its-own views coexist deliberately, as in Role Reads (025).
