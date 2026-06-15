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

Grounded in the Glassfrog API v5 spec (`spec/glassfrog-api-v5.yaml`): each capability maps to one token-scoped `/me*` endpoint, and that operation is the entrypoint its spec should start from. Line numbers are a navigation hint against the current spec revision — confirm by `operationId` if the file has moved.

- Identity Read — a `me` command that prints the authenticated actor (person/agent + organization) the token resolves to. Spec: `GET /me` → `getMe` (`spec/glassfrog-api-v5.yaml:966`); supports `?include=roles` to embed the requester's roles
  + depends-on: Request Authentication
  + depends-on: Request Execution
- My Roles — list the roles the authenticated practitioner fills via a primary, non-discarded assignment (token-scoped, not the org-wide `GET /roles`). Spec: `GET /me/roles` → `listMyRoles` (`spec/glassfrog-api-v5.yaml:1003`); paginated
  + depends-on: Request Authentication
  + depends-on: Request Execution
- My Actions — list the actions owned by roles the practitioner fills. Spec: `GET /me/actions` → `listMyActions` (`spec/glassfrog-api-v5.yaml:1092`); paginated, `?status=` filter
  + depends-on: Request Authentication
  + depends-on: Request Execution
- My Projects — list the projects owned by roles the practitioner fills. Spec: `GET /me/projects` → `listMyProjects` (`spec/glassfrog-api-v5.yaml:1040`); paginated, `?status=` filter
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

Grounded in the Glassfrog API v5 spec (`spec/glassfrog-api-v5.yaml`): circles and accountabilities have no standalone endpoint — circles are read through the role tree, and accountabilities are returned inline on role reads (`Role` always carries `accountabilities`, `domains`, and `fillers`), while tree reads gate them behind `?include=accountabilities`. The per-role reads (domains, policies, projects) take a `required` role-id path param, so they depend on Role Reads for the ids they consume. Line numbers are a navigation hint against the current spec revision — confirm by `operationId`.

- Role Reads — list the organization's roles and read one by id: `GET /roles` → `listRoles` (`spec/glassfrog-api-v5.yaml:134`; paginated, optional `parent_role_id`/`person_id`/`has_subroles`/`tag` filters, embeds accountabilities/domains/fillers inline) and `GET /roles/{id}` → `getRole` (`spec/glassfrog-api-v5.yaml:201`; accountabilities/domains/fillers inline, `?include=assignments,subroles,parent_role,policies,notes,skills` for related resources). The org-wide role surface (vs. the token-scoped My Roles) and the source of the role ids the per-role reads consume
  + depends-on: Request Authentication
  + depends-on: Request Execution
- Organization Tree — read the circle hierarchy: `GET /tree` → `getOrgTree` (`spec/glassfrog-api-v5.yaml:682`; the whole org, no role id required), `GET /roles/{id}/tree` → `getRoleTree` (`spec/glassfrog-api-v5.yaml:597`), and `GET /roles/{id}/subroles` → `listSubroles` (`spec/glassfrog-api-v5.yaml:250`); `?include` embeds accountabilities/domains/members per node
  + depends-on: Request Authentication
  + depends-on: Request Execution
- Role Domains — list a role's domains and read one by id: `GET /roles/{id}/domains` → `listRoleDomains` (`spec/glassfrog-api-v5.yaml:2837`; requires role id) and `GET /domains/{id}` → `getDomain` (`spec/glassfrog-api-v5.yaml:2883`)
  + depends-on: Role Reads
  + depends-on: Request Authentication
  + depends-on: Request Execution
- Role Policies — list a role's policies and read one by id: `GET /roles/{id}/policies` → `listRolePolicies` (`spec/glassfrog-api-v5.yaml:2760`; requires role id) and `GET /policies/{id}` → `getPolicy` (`spec/glassfrog-api-v5.yaml:2806`)
  + depends-on: Role Reads
  + depends-on: Request Authentication
  + depends-on: Request Execution
- Role Projects — list a role's projects and read one by id: `GET /roles/{role_id}/projects` → `listRoleProjects` (`spec/glassfrog-api-v5.yaml:3274`; requires role id, optional `status`/`tag`/`include`) and `GET /projects/{id}` → `getProject` (`spec/glassfrog-api-v5.yaml:3502`)
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

Grounded in the Glassfrog API v5 spec (`spec/glassfrog-api-v5.yaml`): a tension is captured against a sensing role (the path role is the sensing role, stored as `impacted_role_id` for historical reasons; the requester is the sensing person). Reads and edits operate by tension id (`ten_`). `status` is auto-computed except explicit `archived` via PATCH; delete is a soft-delete where 404-after-204 is success. Line numbers are a navigation hint against the current spec revision — confirm by `operationId`.

- Tension Capture — capture a tension sensed by a role as the seed of a proposal: `POST /roles/{role_id}/tensions` → `createTension` (`spec/glassfrog-api-v5.yaml:2551`; body via `TensionInput` — body/label/status/meeting_type)
  + depends-on: Request Authentication
  + depends-on: Request Execution
- Tension Reads — list a role's tensions and read one by id: `GET /roles/{role_id}/tensions` → `listRoleTensions` (`spec/glassfrog-api-v5.yaml:2496`; paginated, `?status=unprocessed|processed|archived`) and `GET /tensions/{id}` → `getTension` (`spec/glassfrog-api-v5.yaml:2654`)
  + depends-on: Request Authentication
  + depends-on: Request Execution
- Subroles Tension Roll-up — list tensions across a role's direct subroles, one level (not transitive): `GET /roles/{role_id}/subroles/tensions` → `listSubrolesTensions` (`spec/glassfrog-api-v5.yaml:2599`; paginated, `?status=`; anchor must have subroles, leaf roles 404)
  + depends-on: Request Authentication
  + depends-on: Request Execution
- Tension Update — edit a tension's body, label, status, or meeting_type, including explicit archive: `PATCH /tensions/{id}` → `updateTension` (`spec/glassfrog-api-v5.yaml:2684`)
  + depends-on: Request Authentication
  + depends-on: Request Execution
- Tension Discard — soft-delete a tension, setting `discarded_at`: `DELETE /tensions/{id}` → `deleteTension` (`spec/glassfrog-api-v5.yaml:2730`; not cascaded to proposals; treat 404-after-204 as success)
  + depends-on: Request Authentication
  + depends-on: Request Execution

## Search
> Problem: Undiscoverable Governance — when working a tension, the operator can't find which roles, policies, or role-fillers are relevant without already knowing where to look; nothing lets them search the record by topic or relevance (affects: Practitioner, AI agent)

Grounded in the Glassfrog API v5 spec (`spec/glassfrog-api-v5.yaml`): a single cross-model full-text endpoint returns a uniform result shape (`SearchResult` — type, id, title, excerpt, rank, optional role_id) ranked by relevance, so results render uniformly and each result's id bridges into the matching read command. Line numbers are a navigation hint against the current spec revision — confirm by `operationId`.

- Cross-Model Search — a `search` command running a relevance-ranked full-text query across resource types (roles, notes, projects, actions, skills, actors, policies, domains): `GET /search` → `search` (`spec/glassfrog-api-v5.yaml:4156`); required `query` (websearch syntax), optional `types` scoping, paginated; each `SearchResult` carries the id the operator drills into via the matching read command
  + depends-on: Request Authentication
  + depends-on: Request Execution

## Actor Reads
> Problem: An Actor's Governance Footprint — given an actor, the operator can't see what they do: the roles they fill and the accountabilities, domains, and purposes those carry (affects: Practitioner, AI agent)

Grounded in the Glassfrog API v5 spec (`spec/glassfrog-api-v5.yaml`): `/people` and `/agents` are convenience aliases over a unified `/actors` endpoint (`?kind=human|agent`); `/actors` carries no feature gate, so agents are reachable through it (via `?kind=agent` or an `agt_` id), while the dedicated `/agents` alias is `ai_integration`-gated and stays deferred. Reads only — the actor-admin writes (`createActor`/`updateActor`/`deleteActor`, `createRoleAssignment`/`updateAssignment`/`deleteAssignment`) are out of scope. Line numbers are a navigation hint against the current spec revision — confirm by `operationId`.

- Actor Directory — list and find actors to identify whom you mean before drilling in: `GET /actors` → `listActors` (`spec/glassfrog-api-v5.yaml:771`; paginated, `?kind=human|agent`, `?role_id`, `?q`), with `/people` → `listPeople` (`spec/glassfrog-api-v5.yaml:1146`) as the human-filtered alias
  + depends-on: Request Authentication
  + depends-on: Request Execution
- Actor Read — read one actor by id (person or agent) with their roles embedded — the entry to an actor's governance footprint: `GET /actors/{id}` → `getActor` (`spec/glassfrog-api-v5.yaml:857`; accepts `per_`/`agt_`, `?include=roles,assignments`)
  + depends-on: Request Authentication
  + depends-on: Request Execution
- Actor Assignments — list the roles an actor fills as assignments (the inverse of Role Fillers, completing the footprint): `GET /actors/{actor_id}/assignments` → `listActorAssignments` (`spec/glassfrog-api-v5.yaml:1744`; default `include=role`, paginated)
  + depends-on: Request Authentication
  + depends-on: Request Execution

## Role Fillers
> Problem: Who to Contact for a Role — given a role relevant to a tension, the operator can't tell which actor fills it, so they don't know whom to reach out to (affects: Practitioner, AI agent)

Grounded in the Glassfrog API v5 spec (`spec/glassfrog-api-v5.yaml`): role assignments map actors to roles, so the read side answers "who fills this role" directly and one level down across the role's subroles. The subrole roll-up requires an expanded role (`has_subroles: true`) — leaf roles 404. Reads only — the assignment writes (`createRoleAssignment`/`updateAssignment`/`deleteAssignment`) are out of scope. Line numbers are a navigation hint against the current spec revision — confirm by `operationId`.

- Role Fillers — read which actors fill a given role, so the operator knows whom to contact about a tension: `GET /roles/{role_id}/assignments` → `listRoleAssignments` (`spec/glassfrog-api-v5.yaml:1644`; default `include=actor` embeds the full actor objects, paginated)
  + depends-on: Request Authentication
  + depends-on: Request Execution
- Subrole Filler Roll-up — read the actors filling a role's direct subroles (one level), to reach the surrounding circle when the role itself is vacant or shared: `GET /roles/{id}/subroles/actors` → `listSubrolesActors` (`spec/glassfrog-api-v5.yaml:321`; `?kind=human|agent`, paginated, requires `has_subroles`, leaf roles 404), with `/roles/{id}/subroles/people` → `listSubrolesPeople` (`spec/glassfrog-api-v5.yaml:379`) as the human-filtered alias
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

Grounded in the Glassfrog API v5 spec (`spec/glassfrog-api-v5.yaml`): proposals are the only sanctioned path to alter governance structure, and the whole write surface is **Premium-gated** (async proposals — every write returns `403` when not enabled). A proposal is anchored to a tension and carries `changes[]` — free-form governance commands (`ProposalChange` is `type` plus open properties, with no per-type schema in the spec), so the CLI passes them through as supplied and lets the server validate; typed per-type change builders are deferred as a separate problem (a future `/prelude:capture` candidate). The lifecycle is `draft → proposed_outside_meeting / escalated → accepted`, with `propose`/`withdraw` offered only when they appear in the proposal's `available_transitions`; responses are `no_objection` | `bring_to_meeting`, and `response_summary` is aggregate-only (no per-person attribution). Line numbers are a navigation hint against the current spec revision — confirm by `operationId`.

- Proposal Creation — create a draft proposal anchored to a tension, carrying caller-supplied governance `changes`: `POST /proposals` → `createProposal` (`spec/glassfrog-api-v5.yaml:3699`; Premium — 403 when async proposals disabled; body `CreateProposalRequest` — required `tension_id`, free-form `changes[]` passed through as supplied and server-validated)
  + depends-on: Request Authentication
  + depends-on: Request Execution
- Proposal Reads — list proposals and read one by id with its `changes`, aggregate `response_summary`, and the caller's `available_transitions`: `GET /proposals` → `listProposals` (`spec/glassfrog-api-v5.yaml:3622`; paginated, `?status`/`role_id`/`proposer_id`/`proposed_after`/`accepted_after`) and `GET /proposals/{id}` → `getProposal` (`spec/glassfrog-api-v5.yaml:3739`; no per-person response attribution)
  + depends-on: Request Authentication
  + depends-on: Request Execution
- Advance to Circulation — move a draft into circulation (`draft → proposed_outside_meeting`), auto-recording the proposer's implicit `no_objection` and setting the response deadline: `POST /proposals/{proposal_id}/propose` → `proposeProposal` (`spec/glassfrog-api-v5.yaml:3773`; Premium; allowed only when `propose` is in `available_transitions`)
  + depends-on: Request Authentication
  + depends-on: Request Execution
- Withdraw Proposal — return a circulating (`proposed_outside_meeting`/`escalated`) proposal to `draft` for re-editing, clearing existing responses and the proposed timestamps: `POST /proposals/{proposal_id}/withdraw` → `withdrawProposal` (`spec/glassfrog-api-v5.yaml:3829`; Premium; allowed only when `withdraw` is in `available_transitions`)
  + depends-on: Request Authentication
  + depends-on: Request Execution
- Response Recording — record a circle member's consent-window response: `POST /proposals/{proposal_id}/responses` → `createProposalResponse` (`spec/glassfrog-api-v5.yaml:3874`; Premium; body `CreateProposalResponseRequest` — `no_objection` | `bring_to_meeting`; one per person, 422 on a second)
  + depends-on: Request Authentication
  + depends-on: Request Execution

## Plan-Limit Signalling
> Problem: Unsignalled Plan Limits — plan/flag-gated endpoints (ai_integration, Premium) reject with 403 and no clear "not available on your plan" signal (affects: Practitioner)

Grounded in the Glassfrog API v5 spec (`spec/glassfrog-api-v5.yaml`): the `403` `Forbidden` response is a generic RFC 9457 `ProblemDetails` with no structured field marking it as a plan-gate, so gate-awareness comes from static spec metadata — the `x-feature-gate: ai_integration` vendor extension and the Premium-documented operations ("requires async proposals" / "restricted (non-premium) plans") — not from the response body. In scope, the gated surface that matters is the Premium async-proposal write path (relates to the Proposal Write-Flow problem); the `ai_integration` agent/skill endpoints stay deferred per PROJECT.md scope.

- Feature-Gate Recognition — identify when a `403` came from a known plan/feature-gated operation (the spec's `x-feature-gate: ai_integration` extension and the Premium-documented endpoints, chiefly the async-proposal write path) so a plan-limit rejection is distinguishable from a generic permission denial, since the `403` body is not contractually self-identifying
  + depends-on: API Error Extraction
- Plan-Limit Signal — surface a recognized plan-limit rejection as a clear, actionable "not available on your plan" diagnostic that names the gating feature (e.g. Premium async proposals) and the next step, rendered in the selected output format
  + depends-on: Feature-Gate Recognition
  + depends-on: Diagnostic Normalization

## Agent Operating Surface
> Problem: Unequipped Agent Operators — the AI agent driving the CLI has no packaged operating knowledge, so it rediscovers how to operate the CLI each session and can mis-drive it or run ungated writes (affects: AI agent, Practitioner)

A thin operator layer over the CLI — packaged operating knowledge, operator paths for governance work, and a write-safety guardrail — delivered as a repo-shipped plugin (its own marketplace). It adds no API capability of its own: every path is a guided composition of CLI capabilities that already exist, so its capabilities reach across solutions via `depends-on`. Reflects PROJECT scope "Agent operating surface" and honors VISION Exclusion 2 plus the "knowledge + guardrails, never capability" constraint. Holacracy-practice fluency was deliberately excluded (a separate plugin's concern).

- Operator Orientation — packaged knowledge of how to drive the CLI: command surface, output formats for parsing, pagination, exit-code reactions, and credential setup, so the agent operates it correctly without rediscovery
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
- Operating-Surface Packaging — package and distribute the surface so an agent environment installs and runs it (repo-shipped, its own marketplace), leaning on existing credential setup
  + depends-on: Operator Orientation
