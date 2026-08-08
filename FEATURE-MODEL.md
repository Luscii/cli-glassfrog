# Glassfrog CLI — Feature Model

This document captures the solutions and capabilities for the Glassfrog CLI, organized by the problems they solve. Each solution addresses a specific problem; capabilities within a solution describe what needs to be built. The format is semantic markdown — headings for solutions, blockquotes for problems, dashes for capabilities, pluses for relationships.

---

## CLI Skeleton
> Problem: No Runnable CLI — no project skeleton or command framework exists to build any command on (affects: Maintainer)

- Command Registration — define and plug in subcommands so new commands attach without touching unrelated ones
- Argument Dispatch — parse the invocation arguments and route them to the correct registered command
  + depends-on: Command Registration
- Help & Version — --help / usage output, a discoverable listing of registered commands, and --version
  + depends-on: Command Registration
- Exit-Code Convention — standardized process exit codes (success / usage error / runtime error) for CI and agent consumption
  + depends-on: Argument Dispatch

## Token Authentication
> Problem: Unauthenticated Access — the CLI has no way to prove it's acting as a specific org + person, so Glassfrog can't authorize its calls (affects: AI agent, Practitioner)

- Credential Storage — a login-style command that accepts a token and writes it to a credentials file in the home or working directory (npmrc-style)
- Credential Discovery — locate and read the credentials file at call time, with the working-directory file taking precedence over the home-directory one (nearest-wins)
- Request Authentication — attach the resolved token as the X-Auth-Token header on outgoing API calls so Glassfrog authorizes them as the specific org + person
  + depends-on: Credential Discovery

## Connection Configuration
> Problem: Undefined Connection Settings — the CLI doesn't know which token or base URL to use, or where to read them from (affects: Practitioner)

- Base URL Resolution — resolve the effective base URL by precedence (command flag > environment variable > config file > built-in default), read the value from the same npmrc-style credentials file that Credential Discovery locates (nearest-wins), resolving its value independently of the token
- Connection Context Assembly — combine the resolved base URL with the discovered token into the single connection context each request uses
  + depends-on: Base URL Resolution
  + depends-on: Credential Discovery

## Self-Service Reads
> Problem: Self-Service Reads — read "what's mine": me, my roles, actions, and projects (affects: Practitioner)

Grounded in the Glassfrog API v5 spec (`spec/glassfrog-api-v5.yaml`): each capability maps to one token-scoped `/me*` endpoint, and that operation is the entrypoint its spec should start from. Operations are cited by `operationId`, which is stable across spec revisions — line numbers shift on every refresh and are deliberately omitted.

- Identity Read — a `me` command that prints the authenticated actor (person/agent + organization) the token resolves to. Spec: `GET /me` → `getMe`; supports `?include=roles` to embed the requester's roles
  + depends-on: Request Authentication
  + depends-on: Request Execution
- My Roles — list the roles the authenticated practitioner fills via a primary, non-discarded assignment (token-scoped, not the org-wide `GET /roles`). Spec: `GET /me/roles` → `listMyRoles`; paginated
  + depends-on: Request Authentication
  + depends-on: Request Execution
- My Actions — list the actions owned by roles the practitioner fills. Spec: `GET /me/actions` → `listMyActions`; paginated, `?status=` filter
  + depends-on: Request Authentication
  + depends-on: Request Execution
- My Projects — list the projects owned by roles the practitioner fills. Spec: `GET /me/projects` → `listMyProjects`; paginated, `?status=` filter
  + depends-on: Request Authentication
  + depends-on: Request Execution

## API Client
> Problem: No Shared API Client — the CLI can resolve a connection context but has no shared way to issue a request and apply the API's response, error, paging, and rate-limit conventions, so every endpoint command would reinvent transport plumbing (affects: AI agent, Practitioner, Maintainer)

- Request Execution — send an authenticated request through the resolved connection context to the Glassfrog API and return the parsed response or a typed transport error; the single seam every endpoint command calls through
  + depends-on: Connection Context Assembly
  + depends-on: Request Authentication
- API Error Extraction — interpret a non-2xx Glassfrog response into a typed error carrying the API's status and error detail, so callers receive a structured cause rather than a raw response
  + depends-on: Request Execution
- Pagination — follow the API's paging to retrieve a complete result set or signal the boundary, so list commands never silently truncate
  + depends-on: Request Execution
- Rate-Limit Handling — honor the API's per-org rate limit (429 + rate-limit headers) with backoff so request volume stays within limits
  + depends-on: Request Execution

## Output Formatting
> Problem: Unconsumable Output — results aren't shaped for an AI agent to parse reliably or for a human to read (affects: AI agent, Practitioner)

- Structured Serialization — render a command's result data as machine-readable JSON and YAML for agent consumption (VISION principle 3)
- Templated Human Rendering — render human-readable output through a template mechanism, with built-in `full` and `compact` templates; the template seam is what later admits caller-supplied templates
- Output Format Selection — an `--output` flag selecting the format per invocation (full | compact | json | yaml), with a default when omitted, dispatching the result to the matching renderer
  + depends-on: Structured Serialization
  + depends-on: Templated Human Rendering
- User-Defined Template Output — accept a caller-supplied template file as the output format, rendered through the same template mechanism (future — lower priority per developer)
  + depends-on: Templated Human Rendering

## Self-Contained Distribution
> Problem: Runtime-Dependent Distribution — the CLI can't run without a separately-installed runtime, so it won't run where operators need it (affects: Practitioner, AI agent, Maintainer)

- Self-Contained Executable Build — cross-compile a single dependency-free binary per supported platform (macOS amd64/arm64, Linux amd64/arm64) that runs with no separately-installed runtime (CONSTITUTION XII)
- Version Embedding — inject the release version at build time, falling back to Go module build info for `go install` source builds, so `--version` always reports a meaningful version
  + depends-on: Self-Contained Executable Build
- Automated Release Pipeline — when a GitHub Release is published, build the platform binaries, package them as archives with a checksums file, and attach them to that release — automated, entirely within this repository
  + depends-on: Self-Contained Executable Build
  + depends-on: Release Drafting (trigger source: the published release a maintainer publishes; a hand-published release also works)
- Install Script — a POSIX one-liner (hosted in this repo) that detects OS/arch, downloads the matching archive from Releases, verifies its checksum, and installs onto PATH; the primary path for Linux (and macOS) laptops and CI
  + depends-on: Automated Release Pipeline
- Homebrew Tap — a GoReleaser-published Homebrew formula in a dedicated tap repository (`Luscii/homebrew-cli-glassfrog`), tracking stable releases, so macOS and Linux users can `brew install` / `brew upgrade` (a formula installs the pre-built binary on both platforms with no source build; the dedicated tap repo lets GoReleaser publish the formula directly to it, so nothing is committed to this repo's protected main)
  + depends-on: Automated Release Pipeline
- NPM Wrapper Package — an npm package that resolves and installs the correct platform binary (platform-specific optional dependencies with a postinstall fallback), so Node-based agent environments can `npx` / `npm i -g`
  + depends-on: Automated Release Pipeline

## GitHub Actions Pipeline
> Problem: No Automated Pipeline — without CI/CD, every change is linted, tested, triaged, and released by hand, so regressions and inconsistencies can reach main and releases go out unguarded (affects: Maintainer)

- PR Validation — on pull request, run lint and the test suite so changes are verified before merge
- PR Administration — auto-apply administrative labels to pull requests for triage and release-note categorization
- Main-Branch Verification — on merge to main, re-run the test suite as a post-merge safety check
- Release Drafting — on merge to main, maintain a draft GitHub release (release-drafter; not published): compute the next semver tag from PR labels, accumulate merged-PR titles into label-driven note categories (breaking / features / fixes / docs / infrastructure / dependencies / internal) excluding noise-labelled PRs, and set the draft's pre-release vs latest status — so a maintainer publishes it to trigger Self-Contained Distribution's Automated Release Pipeline, which consumes the published release to build and attach binaries (the pipeline honors the pre-release/latest status set here rather than deciding it)
  + depends-on: PR Administration

## Diagnostic Reporting
> Problem: Opaque Failures — when a call fails, the caller can't tell what went wrong or what to do next (affects: AI agent, Practitioner)

- Diagnostic Normalization — collapse transport failures, typed API errors, and usage errors into one consistent, actionable diagnostic carrying a cause, a category, and the next step the caller can take to resolve it (where one exists)
  + depends-on: API Error Extraction
  + depends-on: Request Execution
- Output-Aware Failure Rendering — render the diagnostic in the selected `--output` format (human-readable cause + next-step on stderr for full/compact; a structured error envelope for json/yaml), paired with the conventional exit code, so failures are as legible and parseable as successful output
  + depends-on: Diagnostic Normalization
  + depends-on: Output Format Selection

## Governance Reads
> Problem: Governance Reads — read roles, circles, accountabilities, domains, policies, and projects (affects: Practitioner)

Grounded in the Glassfrog API v5 spec (`spec/glassfrog-api-v5.yaml`): circles and accountabilities have no standalone endpoint — circles are read through the role tree, and accountabilities are returned inline on role reads (`Role` always carries `accountabilities`, `domains`, and `fillers`), while tree reads gate them behind `?include=accountabilities`. The per-role reads (domains, policies, projects) take a `required` role-id path param, so they depend on Role Reads for the ids they consume. Operations are cited by `operationId`, which is stable across spec revisions — line numbers shift on every refresh and are deliberately omitted.

- Role Reads — list the organization's roles and read one by id: `GET /roles` → `listRoles` (paginated, optional `parent_role_id`/`person_id`/`has_subroles`/`tag` filters, embeds accountabilities/domains/fillers inline) and `GET /roles/{id}` → `getRole` (accountabilities/domains/fillers inline, `?include=assignments,subroles,parent_role,policies,notes,skills` for related resources). The org-wide role surface (vs. the token-scoped My Roles) and the source of the role ids the per-role reads consume
  + depends-on: Request Authentication
  + depends-on: Request Execution
- Organization Tree — read the circle hierarchy: `GET /tree` → `getOrgTree` (the whole org, no role id required), `GET /roles/{id}/tree` → `getRoleTree`, and `GET /roles/{id}/subroles` → `listSubroles`; `?include` embeds accountabilities/domains/members per node
  + depends-on: Request Authentication
  + depends-on: Request Execution
- Role Domains — list a role's domains and read one by id: `GET /roles/{id}/domains` → `listRoleDomains` (requires role id) and `GET /domains/{id}` → `getDomain`
  + depends-on: Role Reads
  + depends-on: Request Authentication
  + depends-on: Request Execution
- Role Policies — list a role's policies and read one by id: `GET /roles/{id}/policies` → `listRolePolicies` (requires role id) and `GET /policies/{id}` → `getPolicy`
  + depends-on: Role Reads
  + depends-on: Request Authentication
  + depends-on: Request Execution
- Role Projects — list a role's projects and read one by id: `GET /roles/{role_id}/projects` → `listRoleProjects` (requires role id, optional `status`/`tag`/`include`) and `GET /projects/{id}` → `getProject`
  + depends-on: Role Reads
  + depends-on: Request Authentication
  + depends-on: Request Execution

## Composable Setting Resolver
> Problem: Duplicated Setting Resolution — each configurable setting (token, base URL, output format) re-implements the same flag→env→.glassfrogrc→default precedence chain, so adding a setting copies the OS seam, the chain skeleton, the Source enum, and the validation-error shape, and the copies can drift (affects: Maintainer)

- Source-Composed Resolution — `resolveSetting([...sources])` walks an ordered source list (array order is precedence), returns the first source that yields a value, skips empty sources, and backstops with an optional trailing default (absence stays valid where there is no default, e.g. token); sources are `fromFlags` / `fromEnv` / `fromFile` / `fromStdin` constructors that each also accept a list and walk it until one yields (e.g. `fromFlags(["--output","-o"])`); the file source is partially applied from the shared `.glassfrogrc` reader, and every source returns one shared result shape carrying the value, its provenance, and a uniform validation error
- Resolution Call-Site Retrofit — migrate the three existing resolution sites (token, base URL, output format) onto the resolver, behavior-preserving with existing tests staying green
  + depends-on: Source-Composed Resolution

## Tension Management
> Problem: Tension Capture — record a tension as the entry point to a proposal (affects: Practitioner)

Grounded in the Glassfrog API v5 spec (`spec/glassfrog-api-v5.yaml`): a tension is captured against a sensing role (the path role is the sensing role, stored as `impacted_role_id` for historical reasons; the requester is the sensing person). Reads and edits operate by tension id (`ten_`). `status` is auto-computed except explicit `archived` via PATCH; delete is a soft-delete where 404-after-204 is success. Operations are cited by `operationId`, which is stable across spec revisions — line numbers shift on every refresh and are deliberately omitted.

- Tension Capture — capture a tension sensed by a role as the seed of a proposal: `POST /roles/{role_id}/tensions` → `createTension` (body via `TensionInput` — body/label/status/meeting_type)
  + depends-on: Request Authentication
  + depends-on: Request Execution
- Tension Reads — list a role's tensions and read one by id: `GET /roles/{role_id}/tensions` → `listRoleTensions` (paginated, `?status=unprocessed|processed|archived`) and `GET /tensions/{id}` → `getTension`
  + depends-on: Request Authentication
  + depends-on: Request Execution
- Subroles Tension Roll-up — list tensions across a role's direct subroles, one level (not transitive): `GET /roles/{role_id}/subroles/tensions` → `listSubrolesTensions` (paginated, `?status=`; anchor must have subroles, leaf roles 404)
  + depends-on: Request Authentication
  + depends-on: Request Execution
- Tension Update — edit a tension's body, label, status, or meeting_type, including explicit archive: `PATCH /tensions/{id}` → `updateTension`
  + depends-on: Request Authentication
  + depends-on: Request Execution
- Tension Discard — soft-delete a tension, setting `discarded_at`: `DELETE /tensions/{id}` → `deleteTension` (not cascaded to proposals; treat 404-after-204 as success)
  + depends-on: Request Authentication
  + depends-on: Request Execution

## Search
> Problem: Undiscoverable Governance — when working a tension, the operator can't find which roles, policies, or role-fillers are relevant without already knowing where to look; nothing lets them search the record by topic or relevance (affects: Practitioner, AI agent)

Grounded in the Glassfrog API v5 spec (`spec/glassfrog-api-v5.yaml`): a single cross-model full-text endpoint returns a uniform result shape (`SearchResult` — type, id, title, excerpt, rank, optional role_id) ranked by relevance, so results render uniformly and each result's id bridges into the matching read command. Operations are cited by `operationId`, which is stable across spec revisions — line numbers shift on every refresh and are deliberately omitted.

- Cross-Model Search — a `search` command running a relevance-ranked full-text query across resource types (roles, notes, projects, actions, skills, actors, policies, domains): `GET /search` → `search`; required `query` (websearch syntax), optional `types` scoping, paginated; each `SearchResult` carries the id the operator drills into via the matching read command
  + depends-on: Request Authentication
  + depends-on: Request Execution

## Actor Reads
> Problem: An Actor's Governance Footprint — given an actor, the operator can't see what they do: the roles they fill and the accountabilities, domains, and purposes those carry (affects: Practitioner, AI agent)

Grounded in the Glassfrog API v5 spec (`spec/glassfrog-api-v5.yaml`): `/people` and `/agents` are convenience aliases over a unified `/actors` endpoint (`?kind=human|agent`); `/actors` carries no feature gate, so agents are reachable through it (via `?kind=agent` or an `agt_` id), while the dedicated `/agents` alias is `ai_integration`-gated and stays deferred. Reads only — the actor-admin writes (`createActor`/`updateActor`/`deleteActor`, `createRoleAssignment`/`updateAssignment`/`deleteAssignment`) are out of scope. Operations are cited by `operationId`, which is stable across spec revisions — line numbers shift on every refresh and are deliberately omitted.

- Actor Directory — list and find actors to identify whom you mean before drilling in: `GET /actors` → `listActors` (paginated, `?kind=human|agent`, `?role_id`, `?q`), with `/people` → `listPeople` as the human-filtered alias
  + depends-on: Request Authentication
  + depends-on: Request Execution
- Actor Read — read one actor by id (person or agent) with their roles embedded — the entry to an actor's governance footprint: `GET /actors/{id}` → `getActor` (accepts `per_`/`agt_`, `?include=roles,assignments`)
  + depends-on: Request Authentication
  + depends-on: Request Execution
- Actor Assignments — list the roles an actor fills as assignments (the inverse of Role Fillers, completing the footprint): `GET /actors/{actor_id}/assignments` → `listActorAssignments` (default `include=role`, paginated)
  + depends-on: Request Authentication
  + depends-on: Request Execution

## Role Fillers
> Problem: Who to Contact for a Role — given a role relevant to a tension, the operator can't tell which actor fills it, so they don't know whom to reach out to (affects: Practitioner, AI agent)

Grounded in the Glassfrog API v5 spec (`spec/glassfrog-api-v5.yaml`): role assignments map actors to roles, so the read side answers "who fills this role" directly and one level down across the role's subroles. The subrole roll-up requires an expanded role (`has_subroles: true`) — leaf roles 404. Reads only — the assignment writes (`createRoleAssignment`/`updateAssignment`/`deleteAssignment`) are out of scope. Operations are cited by `operationId`, which is stable across spec revisions — line numbers shift on every refresh and are deliberately omitted.

- Role Fillers — read which actors fill a given role, so the operator knows whom to contact about a tension: `GET /roles/{role_id}/assignments` → `listRoleAssignments` (default `include=actor` embeds the full actor objects, paginated)
  + depends-on: Request Authentication
  + depends-on: Request Execution
- Subrole Filler Roll-up — read the actors filling a role's direct subroles (one level), to reach the surrounding circle when the role itself is vacant or shared: `GET /roles/{id}/subroles/actors` → `listSubrolesActors` (`?kind=human|agent`, paginated, requires `has_subroles`, leaf roles 404), with `/roles/{id}/subroles/people` → `listSubrolesPeople` as the human-filtered alias
  + depends-on: Request Authentication
  + depends-on: Request Execution

## Optimistic Concurrency
> Problem: Clobbered Changes — concurrent governance edits overwrite each other when writes skip version checks (affects: Practitioner)

Grounded in the Glassfrog API v5 spec (`spec/glassfrog-api-v5.yaml`) "Optimistic Concurrency (ETags)" section: GET responses for mutable resources carry an `ETag` header (a hash of current state); sending it back on a PUT/PATCH via `If-Match` makes the server accept the write only if the resource is unchanged, otherwise rejecting it (`412 Precondition Failed`). When `If-Match` is omitted the update proceeds unconditionally (last-write-wins) — which is the gap this solution closes. A Client-Foundation mechanism the write commands (Tension Update, Tension Discard, Proposal Write-Flow) opt into; retrofitting each write call-site is per-command work, not a capability here.

- Version Capture on Read — surface the resource version (`ETag`) from a read so a later edit can be guarded against intervening changes
- Guarded Writes — send the captured version via `If-Match` on an edit so the server refuses the write if the resource changed since it was read, rather than silently overwriting
  + depends-on: Version Capture on Read
- Stale-Write Surfacing — when a guarded write is refused (`412 Precondition Failed`), report the clobber distinctly so the operator knows the resource changed under them and can re-read before retrying
  + depends-on: Guarded Writes

## Governance Proposals
> Problem: Proposal Write-Flow — the multi-step governance write path: create → propose → respond → accepted (affects: Practitioner)

Grounded in the Glassfrog API v5 spec (`spec/glassfrog-api-v5.yaml`): proposals are the only sanctioned path to alter governance structure, and the whole write surface is **Premium-gated** (async proposals — every write returns `403` when not enabled). A proposal is anchored to a tension and carries `changes[]` — free-form governance commands (`ProposalChange` is `type` plus open properties, with no per-type schema in the spec), so the CLI passes them through as supplied and lets the server validate; per-type change construction is its own problem (Unguided Change Construction), addressed by the **Verified Change-Set Grammar Reference** and **Typed Change Builders** solutions below. The lifecycle is `draft → proposed_outside_meeting / escalated → accepted`, with `propose`/`withdraw` offered only when they appear in the proposal's `available_transitions`; responses are `no_objection` | `bring_to_meeting`, and `response_summary` is aggregate-only (no per-person attribution). Operations are cited by `operationId`, which is stable across spec revisions — line numbers shift on every refresh and are deliberately omitted.

- Proposal Creation — create a draft proposal anchored to a tension, carrying caller-supplied governance `changes`: `POST /proposals` → `createProposal` (Premium — 403 when async proposals disabled; body `CreateProposalRequest` — required `tension_id`, free-form `changes[]` passed through as supplied and server-validated)
  + depends-on: Request Authentication
  + depends-on: Request Execution
- Proposal Reads — list proposals and read one by id with its `changes`, aggregate `response_summary`, and the caller's `available_transitions`: `GET /proposals` → `listProposals` (paginated, `?status`/`role_id`/`proposer_id`/`proposed_after`/`accepted_after`) and `GET /proposals/{id}` → `getProposal` (no per-person response attribution)
  + depends-on: Request Authentication
  + depends-on: Request Execution
- Advance to Circulation — move a draft into circulation (`draft → proposed_outside_meeting`), auto-recording the proposer's implicit `no_objection` and setting the response deadline: `POST /proposals/{proposal_id}/propose` → `proposeProposal` (Premium; allowed only when `propose` is in `available_transitions`)
  + depends-on: Request Authentication
  + depends-on: Request Execution
- Withdraw Proposal — return a circulating (`proposed_outside_meeting`/`escalated`) proposal to `draft` for re-editing, clearing existing responses and the proposed timestamps: `POST /proposals/{proposal_id}/withdraw` → `withdrawProposal` (Premium; allowed only when `withdraw` is in `available_transitions`)
  + depends-on: Request Authentication
  + depends-on: Request Execution
- Response Recording — record a circle member's consent-window response: `POST /proposals/{proposal_id}/responses` → `createProposalResponse` (Premium; body `CreateProposalResponseRequest` — `no_objection` | `bring_to_meeting`; one per person, 422 on a second)
  + depends-on: Request Authentication
  + depends-on: Request Execution

## Plan-Limit Signalling
> Problem: Unsignalled Plan Limits — plan/flag-gated endpoints (ai_integration, Premium) reject with 403 and no clear "not available on your plan" signal (affects: Practitioner)

Grounded in the Glassfrog API v5 spec (`spec/glassfrog-api-v5.yaml`): the `403` `Forbidden` response is a generic RFC 9457 `ProblemDetails` with no structured field marking it as a plan-gate, so gate-awareness comes from static spec metadata — the `x-feature-gate: ai_integration` vendor extension and the Premium-documented operations ("requires async proposals" / "restricted (non-premium) plans") — not from the response body. The Premium async-proposal write path is the gated surface with the sharpest consequence (relates to the Proposal Write-Flow problem). The `ai_integration` surface is no longer hypothetical: PROJECT.md admitted agents, skills, and role context into scope on 2026-08-05, so this capability's `ai_integration` arm now has real consumers rather than standing ready for a deferred one.

- Feature-Gate Recognition — identify when a `403` came from a known plan/feature-gated operation (the spec's `x-feature-gate: ai_integration` extension and the Premium-documented endpoints, chiefly the async-proposal write path) so a plan-limit rejection is distinguishable from a generic permission denial, since the `403` body is not contractually self-identifying
  + depends-on: API Error Extraction
- Plan-Limit Signal — surface a recognized plan-limit rejection as a clear, actionable "not available on your plan" diagnostic that names the gating feature (e.g. Premium async proposals) and the next step, rendered in the selected output format
  + depends-on: Feature-Gate Recognition
  + depends-on: Diagnostic Normalization

## Agent Operating Surface
> Problem: Unequipped Agent Operators — the AI agent driving the CLI has no packaged operating knowledge, so it rediscovers how to operate the CLI each session and can mis-drive it or run ungated writes (affects: AI agent, Practitioner)

A thin operator layer over the CLI — packaged operating knowledge, operator paths for governance work, and a write-safety guardrail — delivered as a repo-shipped plugin (its own marketplace). It adds no API capability of its own: every path is a guided composition of CLI capabilities that already exist, so its capabilities reach across solutions via `depends-on`. Reflects PROJECT scope "Agent operating surface" and honors VISION Exclusion 2 plus the "knowledge + guardrails, never capability" constraint. Holacracy-practice fluency was deliberately excluded (a separate plugin's concern).

- Operator Orientation — the Claude plugin definition (manifest + orientation skill content) plus packaged knowledge of how to drive the CLI: output formats for parsing, pagination, exit-code reactions, credential setup, and write-safety guidance, so the agent operates it correctly without rediscovery; a composed leaf's call shape comes from the generated call-shape artifact rather than the CLI's own help, which orientation keeps for usage-error recovery and credential setup only
- Write-Safety Guardrail — enforce governance integrity at the operator layer: gate every command that writes to the governance record (tension capture, proposals, responses) behind explicit confirmation, and handle a stale-write refusal (412) by re-reading and re-confirming, never blind retry (VISION principle 2)
  + depends-on: Operator Orientation
  + depends-on: Stale-Write Surfacing
- Tension Processing Path — a guided path from a sensed tension to a captured one: articulate it, locate the sensing role, choose tactical vs governance, and capture it
  + depends-on: Operator Orientation
  + depends-on: Write-Safety Guardrail
  + depends-on: Tension Capture
  + depends-on: Cross-Model Search
- Governance Navigation Path — a read-only traversal to work a tension: find the relevant roles, policies, domains, and who fills them, returning a synthesized picture rather than raw dumps
  + depends-on: Operator Orientation
  + depends-on: Cross-Model Search
  + depends-on: Role Fillers
  + depends-on: Role Reads
- Constraint Discovery Path — given something the operator wants to do, surface the domains and policies that govern it (whether it falls under another role's domain or is shaped by a policy) so they know if it's within their authority or needs permission or a proposal; surfaces the governing governance, never reimplements permission rules locally
  + depends-on: Operator Orientation
  + depends-on: Cross-Model Search
  + depends-on: Role Domains
  + depends-on: Role Policies
- Proposal Drafting Path — from a tension, assemble the governance changes and create the draft proposal
  + depends-on: Operator Orientation
  + depends-on: Write-Safety Guardrail
  + depends-on: Proposal Creation
  + depends-on: Agent-Facing Grammar Reference
- Proposal Circulation Path — advance a draft into circulation, track the consent window's responses to acceptance, and withdraw to re-edit when needed
  + depends-on: Proposal Drafting Path
  + depends-on: Write-Safety Guardrail
  + depends-on: Advance to Circulation
  + depends-on: Proposal Reads
- Proposal Impact Review Path — given proposals others are circulating, find the ones touching the roles the practitioner fills, assess how their changes affect those roles, and record a response
  + depends-on: Operator Orientation
  + depends-on: Write-Safety Guardrail
  + depends-on: Proposal Reads
  + depends-on: Response Recording
  + depends-on: My Roles
- Operating-Surface Packaging — distribute the plugin Operator Orientation defines so an agent environment discovers, installs, and runs it (repo-shipped, its own marketplace), leaning on existing credential setup; the plugin definition itself lives in Operator Orientation, so this is distribution only
  + depends-on: Operator Orientation

## Verified Change-Set Grammar Reference
> Problem: Unguided Change Construction — building a proposal means hand-writing each governance change command as free-form JSON, because nothing carries per-type structure or guidance for the change set, so constructing a valid `changes[]` is error-prone and demands prior knowledge of each command's shape (affects: Practitioner, AI agent)

The operating-surface answer to the problem: put the verified grammar where the assembling agent reads it, rather than in the CLI. This layer claims no spec authority and rejects nothing locally — the server stays the judge of what is valid — which is what keeps it inside VISION Exclusion 2 ("no local governance logic") while still removing the prior knowledge the assembler currently has to supply from nowhere. Sits alongside the Agent Operating Surface's other packaged knowledge and is consumed by its Proposal Drafting Path.

- Agent-Facing Grammar Reference — render the recorded grammar facts as the reference an assembling agent loads: the accepted part shapes, and the rejected shapes with the symptom each produces, in a form consulted before a change set is built
  + depends-on: Change-Set Grammar Facts
- Pre-Assembly Grammar Consultation — the drafting path consults the reference before assembling, and surfaces a recognized dead shape before the write rather than after it, so the grammar is applied rather than merely shipped
  + depends-on: Agent-Facing Grammar Reference
  + depends-on: Proposal Drafting Path
  + depends-on: Circle Routing Rule

## Typed Change Builders
> Problem: Unguided Change Construction — building a proposal means hand-writing each governance change command as free-form JSON, because nothing carries per-type structure or guidance for the change set, so constructing a valid `changes[]` is error-prone and demands prior knowledge of each command's shape (affects: Practitioner, AI agent)

The CLI-side answer to the same problem: shape the payload through explicit named fields instead of hand-written JSON. The spec-fidelity tension originally recorded here has largely dissolved — `ProposalChange.type` is now a published enum of twenty-one change types, and the rule that the accountability and domain types are nested-only is stated in the contract, so builders for those types assert nothing the spec does not define (LEARNINGS 2026-08-05, S5). `ProposalChange` remains `additionalProperties: true` with no per-type field schema, so the *fields* within a part still come from observed behaviour; the builders stay scoped accordingly. Local pre-validation is deliberately not a capability: the server remains the judge of validity.

- Change-Type Builders — a per-change-type command or flag set that shapes one `changes[]` part through explicit named fields instead of hand-written JSON, scoped to the change types the recorded grammar facts verify rather than every type the API might accept
  + depends-on: Change-Set Grammar Facts
- Change-Set Assembly — compose the built parts into the `changes[]` array a proposal create consumes, nesting child edits inside their role-operation wrapper
  + depends-on: Change-Type Builders
  + depends-on: Proposal Creation

## Validity Read-Back on Create
> Problem: Success Reported for a Dead Proposal — the CLI reports a created proposal and its id as a success while the server has already marked the draft invalid with no available transitions, so the operator confirms a gated write that produced an object nothing can move forward and finds out only later (affects: Practitioner, AI agent)

Grounded in observed server behaviour recorded in `.score/memory/LEARNINGS.md` (2026-08-05, F6): a change set that self-targets the circle from inside its own governance is accepted at create and returned with an id, while the server has already set `valid: false`, attached a validation alert, and left `available_transitions` empty. The create response alone therefore cannot tell a caller whether the write succeeded in any useful sense. This solution closes the gap by **asking the server rather than judging locally** — the read-back reports the server's own verdict, so VISION Exclusion 2 ("no local governance logic") is untouched. The gated-write confirmation preceding the create is unchanged; what changes is what "created" is allowed to mean.

- Post-Create Validity Read — after a create returns, read the draft back and surface its validity flag and any validation alerts alongside the returned id, so the caller sees the server's own verdict on what was just written
  + depends-on: Proposal Creation
  + depends-on: Proposal Reads
- Invalid-Create Outcome — render a created-but-invalid proposal as a failure carrying the alert text rather than a success carrying a `prp_` id, in the selected output format
  + depends-on: Post-Create Validity Read
  + depends-on: Diagnostic Normalization
  + depends-on: Output-Aware Failure Rendering
  + depends-on: Exit-Code Convention

## Legacy Identifier on Reads
> Problem: Change Targets Unidentifiable from the CLI — a proposal's change can only name an existing role by its legacy numeric identifier, which no CLI read exposes, so every write touching existing governance sends the operator out to the web UI to read the number off a URL mid-flow (affects: Practitioner, AI agent)

The API grew the answer: `?include_legacy_id=true` adds an integer `legacy_id` to each resource on the role and actor reads, verified live to be the same number the web-UI URL shows and the change payloads carry (`.score/memory/LEARNINGS.md`, 2026-08-05, S1/S4). It also resolves both matching limits the harvest route could not — it distinguishes same-named roles, and it labels elements no proposal ever named. **Carries a retirement clock**: the parameter is a declared v3→v5 transition bridge that retires with the v3 API and is documented as something not to build durable integrations on (S3). So it is the right answer *now*, and the surface consuming it should degrade to asking rather than assume it persists.

- Legacy Identifier Request — opt into the transition-only `legacy_id` on the role and actor reads and surface it alongside the stable identifier, so a change target's numeric identifier is obtainable without leaving the CLI, while keeping the field's opt-in, nullable, and time-limited nature visible rather than presenting it as a permanent part of the contract
  + depends-on: Role Reads
  + depends-on: Actor Read

## Identifier Mapping from Proposal History
> Problem: Change Targets Unidentifiable from the CLI — a proposal's change can only name an existing role by its legacy numeric identifier, which no CLI read exposes, so every write touching existing governance sends the operator out to the web UI to read the number off a URL mid-flow (affects: Practitioner, AI agent)

The complement to `Legacy Identifier on Reads`, not a substitute for it — scoped to the residue that route cannot serve. `include_legacy_id` reaches roles, actors, and `/me` only; nested embeds are excluded and no standalone accountability, domain, or policy endpoint exists, so those elements' numeric identifiers stay unreadable (S2) — verified by a live read whose six accountabilities carried none. What does carry them is the organization's own proposal history: `proposal list` returns `changes` inline, so one walked call harvests every numeric identifier ever authored, most labelled and with rename history intact (V5/V6). The mapping is therefore **bootstrapped, not learned**. Reads only; the server remains the judge of whether a resolved target is correct.

- Identifier Mapping Harvest — walk the organization's proposal history and extract the accountability, domain, and policy numeric identifiers that no read exposes, with their labels and rename history, keyed across the identifier spellings the change payloads use
  + depends-on: Proposal Reads
- Identifier Resolution by Content Match — resolve a hashed accountability, domain, or policy to its numeric identifier by matching its description text against the harvested mapping — reporting the match and its confidence, and declining to choose when the element carries no label or the label maps to several identifiers
  + depends-on: Identifier Mapping Harvest
  + depends-on: Role Domains
  + depends-on: Role Policies

## Ask for the Identifier Up Front
> Problem: Change Targets Unidentifiable from the CLI — a proposal's change can only name an existing role by its legacy numeric identifier, which no CLI read exposes, so every write touching existing governance sends the operator out to the web UI to read the number off a URL mid-flow (affects: Practitioner, AI agent)

The unconditional fallback, at the operating-surface layer: if the identifier cannot be resolved, ask for it *before* assembling rather than discovering its absence as a not-found mid-write. Recorded fact F4 names the web-UI URL pattern (`/orgnav/roles/<number>`) the operator can read it from, so the ask is answerable. This is a sequencing change inside the drafting path, not new API capability, and it rides on the same pre-assembly gate that consults the change-set grammar rather than introducing its own. **Deliberately not coupled to either identifier solution.** `Legacy Identifier on Reads` covers roles and actors but carries a retirement clock (S3), and `Identifier Mapping from Proposal History` covers the accountability/domain/policy residue only by inference, declining ambiguous matches by design. So the ask remains the floor beneath both — and keeping it independent means the cheap unconditional fix is neither blocked behind the richer ones nor lost when the transition bridge retires.

- Identifier Prompt Before Assembly — the drafting path asks the operator for a change target's numeric identifier before assembling the change set, naming the web-UI URL pattern it can be read from, rather than failing into a not-found on the hashed id
  + depends-on: Proposal Drafting Path
  + depends-on: Pre-Assembly Grammar Consultation

## Circle Routing Pre-Check
> Problem: Proposal Circle Not Choosable — a proposal carries no circle of its own and lands in the circle of the anchor tension's sensing role, so a change to a circle's own governance has to be proposed from the parent circle, anchored on a tension sensed by a role the operator fills there (affects: Practitioner, AI agent)

Grounded in recorded fact F7: `proposal create` takes no circle parameter, so a proposal lands in the circle of its anchor tension's sensing role. The consequence is counter-intuitive and cost a wasted draft — a change to a circle's own governance must be proposed from the **parent** circle, anchored on a tension sensed by a role the operator fills there (the circle-role itself works when the operator is Circle Lead). Recorded as one capability rather than two: the check runs inside the pre-assembly gate that already exists for the change-set grammar, so this solution contributes the routing content, not a second gate.

- Circle Routing Rule — establish which circle a change must land in and which tension can anchor it there, given that a proposal inherits its anchor tension's sensing role's circle, so a change to a circle's own governance must be proposed from the parent circle by a role the operator fills there
  + depends-on: My Roles
  + depends-on: Tension Reads

## Bare Capture Then Label
> Problem: Tension Details Rejected at Capture — supplying a tension's agenda label or meeting type while capturing it is refused outright, and the refusal names a length rule that doesn't apply, so nothing tells the operator that capturing bare and adding the label afterwards is the way through (affects: Practitioner, AI agent)

Grounded in recorded fact F1: `tension create` refuses whenever `--label` or `--meeting-type` accompany it, and the refusal cites a length rule that does not apply (*"Agenda label is too short (minimum is 0 characters)"*); bare create succeeds and `tension update --label` then works, while `--meeting-type` is refused even via update. Whether an accepted capture-time shape exists at all is unsettled, so the first capability is written to accept either resolution — send the working shape, or retire a flag that can only ever refuse. A flag that always fails is worse than an absent one, because it reads as supported.

- Capture-Time Detail Flags — supply a tension's agenda label and meeting type at capture in the shape the server accepts, or where no accepted shape exists, remove the flag rather than offering one that always refuses
  + depends-on: Tension Capture
  + depends-on: Tension Update
- Capture-Then-Update Guidance — when a detail cannot be set at capture, name the working path in the refusal itself rather than passing through a length-rule message that misdescribes the cause
  + depends-on: Diagnostic Normalization
  + depends-on: Capture-Time Detail Flags

## Generated Call-Shape Snapshot
> Problem: Call Shapes Not Packaged — the operating surface pins which commands an agent may run but never how to call them, so every path run interrogates the CLI for each leaf's flags before it can do any work (affects: AI agent, Maintainer)

Closes the gap between guarded command names and unguarded call shapes. The composed-leaf registries pin which commands each path may run and the build guards verify those names still resolve, but nothing has ever asserted *how* to call them — so the agents ask the CLI at runtime, on every run. Because the plugin ships from the same repo as the CLI source, the shape can be derived from the binary being shipped and verified by the same build that verifies the names. This *extends* the surface's own invariant — no artifact asserts what no guard checks — rather than weakening it: the artifact exists only because a guard regenerates and verifies it, so a hand-maintained flags document would be a regression, not a shortcut. Grounded in a live tension-processing run on 2026-08-07 that spent three of eight tool uses on per-leaf help calls before doing any work. Six questions are left for the specification to settle: how the artifact reaches an agent (read at run time, or inlined into each agent at build time); whether it is one file per path or one shared file; whether the **orientation** skill's own prose is in scope or stays prose for now; what in the CLI's output is signal rather than rendering noise, and whether generating from command metadata avoids the question entirely; what an agent does when the artifact is missing or unreadable; and whether the per-run cost observed on one agent repeats across the other five. The originating brief that argues each of them is recorded on PR #187.

- Call-Shape Generation — derive each composed leaf's invocation shape from the CLI being shipped and write it into a co-located generated artifact, normalized to carry flags, arguments, and defaults rather than rendering noise
  + produces: co-located call-shape artifact (per-path or shared — open)
- Call-Shape Drift Guard — regenerate the artifact from the shipped CLI during the build and fail on any divergence, so a flag change that skips regeneration turns the build red, without entangling the names-only registries or the gated/ungated disjointness check they feed
  + depends-on: Call-Shape Generation
  + consumes: co-located call-shape artifact
- Snapshot-Backed Leaf Invocation — the path agents take each leaf's flags from the generated artifact instead of interrogating the CLI per run, with a stated behaviour when the artifact is unreadable, and the artifact stays legible to a practitioner following a path by hand
  + depends-on: Call-Shape Generation
  + consumes: co-located call-shape artifact
- Discovery-Guidance Retirement — retire the per-leaf "ask the CLI for its flags" instruction across the plugin's declarative artifacts while preserving the `--help` guidance that is not per-leaf discovery: usage-error recovery, credential setup, and the flag-not-subcommand fact
  + depends-on: Snapshot-Backed Leaf Invocation
  + depends-on: Operator Orientation

## Shared
- Change-Set Grammar Facts — the single recorded source of the change-set shapes the published contract does **not** carry: the own-circle policy shape, and the self-targeting role update the server accepts at create but returns invalid with no transitions, each with the symptom it produces — narrowed once the spec began enumerating the change types and stating the nested-only rule, which this cites rather than restates
  + used-by: Verified Change-Set Grammar Reference, Typed Change Builders
