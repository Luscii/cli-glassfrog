# Backlog

> Generated: 2026-06-12T20:59:00 | Enriched: 2026-08-05 | Framework: MoSCoW | Items: 84

### 1. Command Registration

- **Score**: MoSCoW Must Have
- **Framework**: MoSCoW (Must Have)
- **Rationale**: Dependency root of the CLI Skeleton — every other capability requires it, so it builds first. Highest leverage at zero dependency cost.
- **Status**: specified:001-command-registration

### 2. Argument Dispatch

- **Score**: MoSCoW Must Have
- **Framework**: MoSCoW (Must Have)
- **Rationale**: Builds on Command Registration and sits on the critical path to Exit-Code Convention, so it precedes the parallelizable Help & Version.
- **Dependencies**: → requires: Command Registration
- **Status**: specified:002-argument-dispatch

### 3. Help & Version

- **Score**: MoSCoW Must Have
- **Framework**: MoSCoW (Must Have)
- **Rationale**: Builds on Command Registration but is off the critical path (nothing depends on it), so it follows Argument Dispatch; the two are parallelizable.
- **Dependencies**: → requires: Command Registration
- **Status**: specified:003-help-and-version

### 4. Exit-Code Convention

- **Score**: MoSCoW Must Have
- **Framework**: MoSCoW (Must Have)
- **Rationale**: End of the dependency chain — needs Argument Dispatch in place to emit standardized exit codes, so it builds last.
- **Dependencies**: → requires: Argument Dispatch
- **Status**: specified:004-exit-code-convention

### 5. Credential Discovery

- **Score**: MoSCoW Must Have
- **Framework**: MoSCoW (Must Have)
- **Rationale**: Dependency root of Token Authentication — Request Authentication can't resolve a token without it, so it builds first in the auth solution. Builds on the CLI Skeleton (not a declared FEATURE-MODEL edge).
- **Status**: specified:005-credential-discovery

### 6. Credential Storage

- **Score**: MoSCoW Must Have
- **Framework**: MoSCoW (Must Have)
- **Rationale**: Independent of Credential Discovery (writes the file Discovery reads); off the critical path within the solution, so it parallelizes with #5. The login command builds on the CLI Skeleton (not a declared FEATURE-MODEL edge).
- **Status**: specified:006-credential-storage

### 7. Request Authentication

- **Score**: MoSCoW Must Have
- **Framework**: MoSCoW (Must Have)
- **Rationale**: Proves identity on each API call; builds after Credential Discovery, which it consumes the resolved token from. Gates every self-service read (#11–#14) and Request Execution.
- **Dependencies**: → requires: Credential Discovery
- **Status**: specified:007-request-authentication

### 8. Base URL Resolution

- **Score**: MoSCoW Must Have
- **Framework**: MoSCoW (Must Have)
- **Rationale**: Resolves the effective endpoint by precedence (flag > env > config > default) — the base-URL portion of connection setup. Reuses Credential Discovery's `.glassfrogrc` file and nearest-wins walk, but resolves its value independently. Already specified as 008.
- **Status**: specified:008-base-url-resolution

### 9. Connection Context Assembly

- **Score**: MoSCoW Must Have
- **Framework**: MoSCoW (Must Have)
- **Rationale**: Combines the resolved base URL with the discovered token into the single connection context every request uses. Gates Request Execution, so it must be specified before the API client.
- **Dependencies**: → requires: Base URL Resolution; → requires: Credential Discovery
- **Status**: specified:009-connection-context-assembly

### 10. Request Execution

- **Score**: MoSCoW Must Have
- **Framework**: MoSCoW (Must Have)
- **Rationale**: The single seam every endpoint command calls through — sends the authenticated request and returns a parsed response or a typed transport error. Gates all reads, so it leads the API Client and precedes Identity Read.
- **Dependencies**: → requires: Connection Context Assembly; → requires: Request Authentication
- **Status**: specified:010-request-execution

### 11. Identity Read

- **Score**: MoSCoW Must Have
- **Framework**: MoSCoW (Must Have)
- **Rationale**: The smallest end-to-end read that proves the whole chain (`GET /me` → `getMe`); the first call a consumer makes to orient itself, so it leads the read surface. Now gated on Request Execution. Parallelizable with #12–#14 (no inter-read dependency).
- **Dependencies**: → requires: Request Authentication; → requires: Request Execution
- **Status**: specified:011-identity-read

### 12. My Roles

- **Score**: MoSCoW Must Have
- **Framework**: MoSCoW (Must Have)
- **Rationale**: Named in the ROADMAP Now slice ("/me, my roles"), so it follows Identity Read; the chain is already proven, making this an extension. Parallelizable with the other reads.
- **Dependencies**: → requires: Request Authentication; → requires: Request Execution
- **Status**: specified:012-my-roles

### 13. My Actions

- **Score**: MoSCoW Must Have
- **Framework**: MoSCoW (Must Have)
- **Rationale**: Extends the "what's mine" surface beyond the proven slice; no dependency on the other reads, so its position reflects priority, not build order.
- **Dependencies**: → requires: Request Authentication; → requires: Request Execution
- **Status**: specified:013-my-actions

### 14. My Projects

- **Score**: MoSCoW Must Have
- **Framework**: MoSCoW (Must Have)
- **Rationale**: Completes the "what's mine" surface; last of the parallelizable reads by priority, with no inter-read dependency.
- **Dependencies**: → requires: Request Authentication; → requires: Request Execution
- **Status**: specified:014-my-projects

### 15. API Error Extraction

- **Score**: MoSCoW Should Have
- **Framework**: MoSCoW (Should Have)
- **Rationale**: Turns a non-2xx response into a typed error carrying the API's status and detail; pairs with the Next-tier Opaque Failures problem, so it ranks below the Must-Have reads. Not required for the first read slice — Request Execution already surfaces transport and status failures.
- **Dependencies**: → requires: Request Execution
- **Status**: specified:015-api-error-extraction

### 16. Pagination

- **Score**: MoSCoW Should Have
- **Framework**: MoSCoW (Should Have)
- **Rationale**: Follows the API's paging so list commands never silently truncate; pairs with the Next-tier Silent Truncation problem. Not needed for `/me` (a single resource), so it ranks after the read slice.
- **Dependencies**: → requires: Request Execution
- **Status**: specified:016-pagination

### 17. Rate-Limit Handling

- **Score**: MoSCoW Should Have
- **Framework**: MoSCoW (Should Have)
- **Rationale**: Honors the per-org rate limit (429 + rate-limit headers) with backoff; pairs with the Next-tier Getting Throttled problem. Ranks after the Must-Have reads.
- **Dependencies**: → requires: Request Execution
- **Status**: specified:017-rate-limit-handling

### 18. Structured Serialization

- **Score**: MoSCoW Must Have
- **Framework**: MoSCoW (Must Have)
- **Rationale**: JSON/YAML machine-readable output — core for the AI-agent operator (VISION principle 3) and a no-dependency root of the Output Formatting solution, so it leads the fresh Now work.
- **Status**: specified:018-structured-serialization

### 19. Templated Human Rendering

- **Score**: MoSCoW Must Have
- **Framework**: MoSCoW (Must Have)
- **Rationale**: full/compact human-readable output via a template seam; no dependencies and on the critical path to Output Format Selection, so it parallelizes with Structured Serialization.
- **Status**: specified:019-templated-human-rendering

### 20. Output Format Selection

- **Score**: MoSCoW Must Have
- **Framework**: MoSCoW (Must Have)
- **Rationale**: The `--output` flag that dispatches to a renderer — builds after both renderers exist, so it follows Structured Serialization and Templated Human Rendering.
- **Dependencies**: → requires: Structured Serialization; → requires: Templated Human Rendering
- **Status**: specified:020-output-format-selection

### 21. Self-Contained Executable Build

- **Score**: MoSCoW Must Have
- **Framework**: MoSCoW (Must Have)
- **Rationale**: The dependency-free binary (CONSTITUTION XII) and dependency root of Self-Contained Distribution — everything in the solution builds on it, so it leads the distribution work.
- **Status**: specified:021-self-contained-executable-build

### 22. Automated Release Pipeline

- **Score**: MoSCoW Must Have
- **Framework**: MoSCoW (Must Have)
- **Rationale**: The ship mechanism (publish a release → build → attach binaries + checksums); gates every acquisition channel, so it follows the build and precedes the install paths. Triggered by a published release — normally drafted by Release Drafting, but a hand-published release works too — so the dependency on Release Drafting is soft, and the Must-tier ranking ahead of Should-tier Release Drafting is unchanged.
- **Dependencies**: → requires: Self-Contained Executable Build
- **Status**: specified:022-automated-release-pipeline

### 23. Version Embedding

- **Score**: MoSCoW Must Have
- **Framework**: MoSCoW (Must Have)
- **Rationale**: Injects the release version at build time with a `go install` build-info fallback so `--version` is correct across install methods; a release without it reports "dev", so it's non-negotiable for shipping.
- **Dependencies**: → requires: Self-Contained Executable Build
- **Status**: specified:023-version-embedding

### 24. PR Validation

- **Score**: MoSCoW Must Have
- **Framework**: MoSCoW (Must Have)
- **Rationale**: The core CI quality gate (lint + tests on pull request) — the heart of the pipeline solution, with no dependencies. As a Must-Have it ranks above the Should-Have pipeline and distribution channels.
- **Status**: specified:024-pr-validation

### 25. Role Reads

- **Score**: MoSCoW Must Have
- **Framework**: MoSCoW (Must Have)
- **Rationale**: The org-wide role surface (`GET /roles` + `GET /roles/{id}` with `?include`) and the dependency root of Governance Reads — the per-role reads consume the role ids it yields. Buildable now (deps shipped: 007, 010). As the highest-value Next-tier read, it leads the new Must-Have work.
- **Dependencies**: → requires: Request Authentication; → requires: Request Execution
- **Status**: specified:025-role-reads

### 26. Organization Tree

- **Score**: MoSCoW Must Have
- **Framework**: MoSCoW (Must Have)
- **Rationale**: The circle hierarchy read (`GET /tree`, `GET /roles/{id}/tree`, `GET /roles/{id}/subroles`) — core to reading governance structure. Buildable now (deps shipped: 007, 010); independent of Role Reads via the no-arg whole-org `getOrgTree`.
- **Dependencies**: → requires: Request Authentication; → requires: Request Execution
- **Status**: specified:026-organization-tree

### 27. Install Script

- **Score**: MoSCoW Should Have
- **Framework**: MoSCoW (Should Have)
- **Rationale**: The primary Linux/macOS/CI acquisition path (detect, download, checksum, install); consumes published release artifacts, so it follows the release pipeline. Should rather than Must — one of several channels.
- **Dependencies**: → requires: Automated Release Pipeline
- **Status**: specified:027-install-script

### 28. PR Administration

- **Score**: MoSCoW Should Have
- **Framework**: MoSCoW (Should Have)
- **Rationale**: Auto-applies administrative labels to PRs; no dependencies, but its labels feed Release Drafting's semver bump, so it precedes Release Drafting.
- **Status**: specified:028-pr-administration

### 29. Main-Branch Verification

- **Score**: MoSCoW Should Have
- **Framework**: MoSCoW (Should Have)
- **Rationale**: Re-runs the test suite on merge to main as a post-merge safety net; PR Validation already gates pre-merge, so this is important but not the primary gate.
- **Status**: specified:029-main-branch-verification

### 30. Release Drafting

- **Score**: MoSCoW Should Have
- **Framework**: MoSCoW (Should Have)
- **Rationale**: Maintains a label-driven draft release on merge; depends on PR Administration's labels for the semver bump. Adjacent to Automated Release Pipeline (which consumes the published tag), but a distinct, separately-buildable stage.
- **Dependencies**: → requires: PR Administration
- **Status**: specified:030-release-drafting

### 31. Diagnostic Normalization

- **Score**: MoSCoW Should Have
- **Framework**: MoSCoW (Should Have)
- **Rationale**: Collapses transport, typed-API, and usage failures into one consistent, actionable diagnostic (cause + category + next step) — the root of Diagnostic Reporting and the failure-legibility half of CONSTITUTION II + III. Buildable now (deps shipped: 015, 010).
- **Dependencies**: → requires: API Error Extraction; → requires: Request Execution
- **Status**: specified:031-diagnostic-normalization

### 32. Output-Aware Failure Rendering

- **Score**: MoSCoW Should Have
- **Framework**: MoSCoW (Should Have)
- **Rationale**: Renders the diagnostic in the selected `--output` (human cause+next-step on stderr; structured envelope for json/yaml), so failures are as legible as successes. Follows Diagnostic Normalization and is gated on Output Format Selection (#20).
- **Dependencies**: → requires: Diagnostic Normalization; → requires: Output Format Selection
- **Status**: specified:032-output-aware-failure-rendering

### 33. Role Domains

- **Score**: MoSCoW Should Have
- **Framework**: MoSCoW (Should Have)
- **Rationale**: Per-role domain reads (`GET /roles/{id}/domains` + `GET /domains/{id}`); a first-class governance element. Gated on Role Reads for the required role id.
- **Dependencies**: → requires: Role Reads; → requires: Request Authentication; → requires: Request Execution
- **Status**: specified:033-role-domains

### 34. Role Policies

- **Score**: MoSCoW Should Have
- **Framework**: MoSCoW (Should Have)
- **Rationale**: Per-role policy reads (`GET /roles/{id}/policies` + `GET /policies/{id}`); a first-class governance element. Gated on Role Reads for the required role id.
- **Dependencies**: → requires: Role Reads; → requires: Request Authentication; → requires: Request Execution
- **Status**: specified:034-role-policies

### 35. User-Defined Template Output

- **Score**: MoSCoW Could Have
- **Framework**: MoSCoW (Could Have)
- **Rationale**: Caller-supplied template files via the same template seam; developer-flagged as future / lower-priority, so it sits among the Could-Haves below the Must/Should new work.
- **Dependencies**: → requires: Templated Human Rendering
- **Status**: specified:035-user-defined-template-output

### 36. Homebrew Tap

- **Score**: MoSCoW Could Have
- **Framework**: MoSCoW (Could Have)
- **Rationale**: An additional acquisition channel beyond the primary install script; consumes published release artifacts. Could Have — convenient but not blocking the tool's reach.
- **Dependencies**: → requires: Automated Release Pipeline
- **Status**: specified:036-homebrew-tap

### 37. NPM Wrapper Package

- **Score**: MoSCoW Could Have
- **Framework**: MoSCoW (Could Have)
- **Rationale**: An additional channel for Node-based agent environments (`npx` / `npm i -g`); consumes published release artifacts. Could Have — extends reach but not core to shipping.
- **Dependencies**: → requires: Automated Release Pipeline
- **Status**: specified:037-npm-wrapper-package

### 38. Role Projects

- **Score**: MoSCoW Could Have
- **Framework**: MoSCoW (Could Have)
- **Rationale**: Per-role project reads (`GET /roles/{role_id}/projects` + `GET /projects/{id}`); the most operational (least governance-structural) of the governance reads, so it sits at the Could-Have tier. Gated on Role Reads for the required role id.
- **Dependencies**: → requires: Role Reads; → requires: Request Authentication; → requires: Request Execution
- **Status**: specified:038-role-projects

### 39. Source-Composed Resolution

- **Score**: MoSCoW Should Have
- **Framework**: MoSCoW (Should Have)
- **Rationale**: The composable resolver mechanism for the Next-tier Duplicated Setting Resolution problem (Maintainer-facing maintainability); dependency root of its solution with no declared FEATURE-MODEL edges, so it leads the refactor.
- **Status**: specified:039-source-composed-resolution

### 40. Resolution Call-Site Retrofit

- **Score**: MoSCoW Should Have
- **Framework**: MoSCoW (Should Have)
- **Rationale**: Migrates the three landed resolution sites (token, base URL, output) onto the resolver, behavior-preserving; builds after the resolver exists.
- **Dependencies**: → requires: Source-Composed Resolution
- **Status**: specified:040-resolution-call-site-retrofit

### 41. Cross-Model Search

- **Score**: MoSCoW Should Have
- **Framework**: MoSCoW (Should Have)
- **Rationale**: The `search` command (`GET /search`) for the Undiscoverable Governance problem — relevance-ranked cross-model discovery, high value for the agent operator. Buildable now (deps shipped: 007, 010); self-contained, no inter-feature edges.
- **Dependencies**: → requires: Request Authentication; → requires: Request Execution
- **Status**: specified:041-cross-model-search

### 42. Tension Capture

- **Score**: MoSCoW Could Have
- **Framework**: MoSCoW (Could Have)
- **Rationale**: Captures a tension (`POST /roles/{role_id}/tensions`) — the anchor of the Later write path and where the write half of VISION success #2 begins; highest-value tension capability. Buildable now (deps shipped: 007, 010).
- **Dependencies**: → requires: Request Authentication; → requires: Request Execution
- **Status**: specified:042-tension-capture

### 43. Tension Reads

- **Score**: MoSCoW Could Have
- **Framework**: MoSCoW (Could Have)
- **Rationale**: Lists a role's tensions and reads one by id (`GET /roles/{role_id}/tensions` + `GET /tensions/{id}`); the read surface for tensions, following the capture anchor.
- **Dependencies**: → requires: Request Authentication; → requires: Request Execution
- **Status**: specified:043-tension-reads

### 44. Tension Update

- **Score**: MoSCoW Could Have
- **Framework**: MoSCoW (Could Have)
- **Rationale**: Edits a tension's body/label/status/meeting_type (`PATCH /tensions/{id}`); a Later-tier write op, independent of the other tension capabilities.
- **Dependencies**: → requires: Request Authentication; → requires: Request Execution
- **Status**: specified:044-tension-update

### 45. Tension Discard

- **Score**: MoSCoW Could Have
- **Framework**: MoSCoW (Could Have)
- **Rationale**: Soft-deletes a tension (`DELETE /tensions/{id}`, 404-after-204 as success); a Later-tier write op, independent of the other tension capabilities.
- **Dependencies**: → requires: Request Authentication; → requires: Request Execution
- **Status**: specified:045-tension-discard

### 46. Subroles Tension Roll-up

- **Score**: MoSCoW Could Have
- **Framework**: MoSCoW (Could Have)
- **Rationale**: One-level roll-up of tensions across a role's direct subroles (`GET /roles/{role_id}/subroles/tensions`); the most peripheral tension read, so it sits last.
- **Dependencies**: → requires: Request Authentication; → requires: Request Execution
- **Status**: specified:046-subroles-tension-roll-up

### 47. Role Fillers

- **Score**: MoSCoW Should Have
- **Framework**: MoSCoW (Should Have)
- **Rationale**: Angle-1 anchor of the Who to Contact for a Role problem (`GET /roles/{role_id}/assignments`) — reads which actors fill a role so the operator knows whom to reach out to about a tension. Highest-value of the actor-reads wave; buildable now (deps shipped: 007, 010). Peer to Cross-Model Search (#41) at the Should tier.
- **Dependencies**: → requires: Request Authentication; → requires: Request Execution
- **Status**: specified:047-role-fillers

### 48. Actor Directory

- **Score**: MoSCoW Should Have
- **Framework**: MoSCoW (Should Have)
- **Rationale**: The discovery entry of Actor Reads (`GET /actors` / `listPeople`) — find an actor by name or role before drilling into their footprint. Self-contained, buildable now (deps shipped: 007, 010); ranks just below Role Fillers as the find-then-read primitive.
- **Dependencies**: → requires: Request Authentication; → requires: Request Execution
- **Status**: specified:048-actor-directory

### 49. Actor Read

- **Score**: MoSCoW Should Have
- **Framework**: MoSCoW (Should Have)
- **Rationale**: Angle-2 anchor of An Actor's Governance Footprint (`GET /actors/{id}` with `?include=roles`) — read one actor (person or agent) and the roles they fill. Completes the Should-tier core of the actor reads.
- **Dependencies**: → requires: Request Authentication; → requires: Request Execution
- **Status**: specified:049-actor-read

### 50. Actor Assignments

- **Score**: MoSCoW Could Have
- **Framework**: MoSCoW (Could Have)
- **Rationale**: Lists the roles an actor fills as assignments (`GET /actors/{actor_id}/assignments`) — the inverse of Role Fillers, completing the footprint, but overlapping Actor Read's `?include=roles`, so it sits at the Could tier like the more operational governance reads (#38).
- **Dependencies**: → requires: Request Authentication; → requires: Request Execution
- **Status**: specified:050-actor-assignments

### 51. Subrole Filler Roll-up

- **Score**: MoSCoW Could Have
- **Framework**: MoSCoW (Could Have)
- **Rationale**: One-level roll-up of actors filling a role's direct subroles (`GET /roles/{id}/subroles/actors` / `subroles/people`) — an escalation aid for reaching the surrounding circle; requires an expanded role (leaf roles 404). The most peripheral of the wave, mirroring Subroles Tension Roll-up (#46).
- **Dependencies**: → requires: Request Authentication; → requires: Request Execution
- **Status**: specified:051-subrole-filler-roll-up

### 52. Version Capture on Read

- **Score**: MoSCoW Could Have
- **Framework**: MoSCoW (Could Have)
- **Rationale**: Foundation of the Optimistic Concurrency solution (Clobbered Changes problem) — surfaces the `ETag` from a read so a later edit can be guarded; a Later/Client-Foundation safety mechanism that, per the roadmap, only becomes relevant once writes exist, so it ranks at the Could tier alongside the write wave it protects.
- **Status**: specified:052-version-capture-on-read

### 53. Guarded Writes

- **Score**: MoSCoW Could Have
- **Framework**: MoSCoW (Could Have)
- **Rationale**: Sends the captured version via `If-Match` so the server refuses a stale write instead of silently overwriting (last-write-wins today); the core of Optimistic Concurrency, built once Version Capture exists.
- **Dependencies**: → requires: Version Capture on Read
- **Status**: specified:053-guarded-writes

### 54. Stale-Write Surfacing

- **Score**: MoSCoW Could Have
- **Framework**: MoSCoW (Could Have)
- **Rationale**: Reports a refused write (`412 Precondition Failed`) distinctly so the operator knows the resource changed under them and can re-read before retrying; completes Optimistic Concurrency, only reachable once Guarded Writes can be refused.
- **Dependencies**: → requires: Guarded Writes
- **Status**: specified:054-stale-write-surfacing

### 55. Proposal Creation

- **Score**: MoSCoW Should Have
- **Framework**: MoSCoW (Should Have)
- **Rationale**: Anchor of the Governance Proposals solution and the fulfilment of VISION success criterion #2 (submit a valid proposal); highest-value new capability, buildable now. Should rather than Must — the CLI ships and functions on reads without it.
- **Dependencies**: → requires: Request Authentication; → requires: Request Execution
- **Status**: specified:055-proposal-creation

### 56. Proposal Reads

- **Score**: MoSCoW Should Have
- **Framework**: MoSCoW (Should Have)
- **Rationale**: Operating the write flow requires reading proposals with their changes, response summary, and available transitions; pairs with Proposal Creation as the core read/write pair, no inter-dependency.
- **Dependencies**: → requires: Request Authentication; → requires: Request Execution
- **Status**: specified:056-proposal-reads

### 57. Advance to Circulation

- **Score**: MoSCoW Should Have
- **Framework**: MoSCoW (Should Have)
- **Rationale**: The `propose` transition that moves a draft into circulation; without it a created proposal never reaches acceptance, so it ranks with the core write path. Premium-gated.
- **Dependencies**: → requires: Request Authentication; → requires: Request Execution
- **Status**: specified:057-advance-to-circulation

### 58. Response Recording

- **Score**: MoSCoW Should Have
- **Framework**: MoSCoW (Should Have)
- **Rationale**: Records a circle member's consent-window response (`no_objection` / `bring_to_meeting`), completing the path to auto-acceptance; core to VISION success #2. Premium-gated.
- **Dependencies**: → requires: Request Authentication; → requires: Request Execution
- **Status**: specified:058-response-recording

### 59. Withdraw Proposal

- **Score**: MoSCoW Could Have
- **Framework**: MoSCoW (Could Have)
- **Rationale**: Returns a circulating proposal to draft for re-editing; a recovery path off the happy path to accepted, so it sits below the core write-path capabilities. Premium-gated.
- **Dependencies**: → requires: Request Authentication; → requires: Request Execution
- **Status**: specified:059-withdraw-proposal

### 60. Feature-Gate Recognition

- **Score**: MoSCoW Could Have
- **Framework**: MoSCoW (Could Have)
- **Rationale**: Distinguishes a plan/feature-gated 403 from a generic permission denial using static spec gate metadata; the foundation of Plan-Limit Signalling, buildable now (dep API Error Extraction shipped).
- **Dependencies**: → requires: API Error Extraction
- **Status**: specified:060-feature-gate-recognition

### 61. Plan-Limit Signal

- **Score**: MoSCoW Could Have
- **Framework**: MoSCoW (Could Have)
- **Rationale**: Renders the recognized plan-limit rejection as an actionable "not available on your plan" diagnostic; legibility polish atop Feature-Gate Recognition, so it follows it.
- **Dependencies**: → requires: Feature-Gate Recognition; → requires: Diagnostic Normalization
- **Status**: specified:061-plan-limit-signal

### 62. Operator Orientation

- **Score**: MoSCoW Should Have
- **Framework**: MoSCoW (Should Have)
- **Rationale**: Root of the Agent Operating Surface — every path and the guardrail build on it, so it leads the surface; buildable now atop the shipped CLI. Should rather than Must — the CLI ships without the operating layer. Now also lands the Claude plugin definition (manifest + orientation skill content); its marketplace distribution stays with #70.
- **Status**: specified:062-operator-orientation

### 63. Write-Safety Guardrail

- **Score**: MoSCoW Should Have
- **Framework**: MoSCoW (Should Have)
- **Rationale**: The governance-integrity gate (VISION principle 2) every write path depends on; builds on Operator Orientation and Stale-Write Surfacing, so it follows the root before any write path.
- **Dependencies**: → requires: Operator Orientation; → requires: Stale-Write Surfacing
- **Status**: specified:063-write-safety-guardrail

### 64. Governance Navigation Path

- **Score**: MoSCoW Should Have
- **Framework**: MoSCoW (Should Have)
- **Rationale**: Core read traversal for working a tension; highest-value path and all its read deps are shipped, so it's buildable right after Operator Orientation.
- **Dependencies**: → requires: Operator Orientation; → requires: Cross-Model Search; → requires: Role Fillers; → requires: Role Reads
- **Status**: specified:064-governance-navigation-path

### 65. Constraint Discovery Path

- **Score**: MoSCoW Could Have
- **Framework**: MoSCoW (Could Have)
- **Rationale**: The "am I limited by a domain or policy?" read path; deps shipped, but narrower in use than general navigation, so it sits a tier below.
- **Dependencies**: → requires: Operator Orientation; → requires: Cross-Model Search; → requires: Role Domains; → requires: Role Policies
- **Status**: specified:065-constraint-discovery-path

### 66. Tension Processing Path

- **Score**: MoSCoW Could Have
- **Framework**: MoSCoW (Could Have)
- **Rationale**: First write path — capture a tension end-to-end; gated on the Write-Safety Guardrail, with Tension Capture and Cross-Model Search already shipped, so buildable once the guardrail exists.
- **Dependencies**: → requires: Operator Orientation; → requires: Write-Safety Guardrail; → requires: Tension Capture; → requires: Cross-Model Search
- **Status**: specified:066-tension-processing-path

### 67. Proposal Drafting Path

- **Score**: MoSCoW Could Have
- **Framework**: MoSCoW (Could Have)
- **Rationale**: Drafts a proposal from a tension; gated behind Proposal Creation (#55, still pending), so it follows the Now-tier proposal write-flow landing.
- **Dependencies**: → requires: Operator Orientation; → requires: Write-Safety Guardrail; → requires: Proposal Creation
- **Status**: specified:067-proposal-drafting-path

### 68. Proposal Circulation Path

- **Score**: MoSCoW Could Have
- **Framework**: MoSCoW (Could Have)
- **Rationale**: Advances and tracks a proposal to acceptance; builds on Proposal Drafting Path and the still-pending circulation/read capabilities, so it sequences after them.
- **Dependencies**: → requires: Proposal Drafting Path; → requires: Write-Safety Guardrail; → requires: Advance to Circulation; → requires: Proposal Reads
- **Status**: specified:068-proposal-circulation-path

### 69. Proposal Impact Review Path

- **Score**: MoSCoW Could Have
- **Framework**: MoSCoW (Could Have)
- **Rationale**: The consume/respond side — assess others' circulating proposals against the roles I fill and respond; gated on Response Recording (#58, pending), with My Roles already shipped.
- **Dependencies**: → requires: Operator Orientation; → requires: Write-Safety Guardrail; → requires: Proposal Reads; → requires: Response Recording; → requires: My Roles
- **Status**: specified:069-proposal-impact-review-path

### 70. Operating-Surface Packaging

- **Score**: MoSCoW Should Have
- **Framework**: MoSCoW (Should Have)
- **Rationale**: The distribution vehicle — its own repo-shipped marketplace that publishes and installs the plugin Operator Orientation defines; the plugin definition itself moved to #62, so this is distribution only. Still needs the surface to have content to ship, so it trails it but is required for any of it to reach an agent environment.
- **Dependencies**: → requires: Operator Orientation
- **Status**: specified:070-operating-surface-packaging

### 71. Change-Set Grammar Facts

- **Score**: MoSCoW Must Have
- **Framework**: MoSCoW (Must Have)
- **Rationale**: Root of the new dependency graph — both Unguided Change Construction solutions derive from it and nothing in either branch can be specified before it; narrowed by the spec refresh to the shapes the contract still does not carry, which makes it cheaper without moving it off the critical path.
- **Status**: pending

### 72. Circle Routing Rule

- **Score**: MoSCoW Must Have
- **Framework**: MoSCoW (Must Have)
- **Rationale**: An independent root whose dependencies are all specified; feeds the pre-assembly gate at #77, and routing must be settled before a change set can be assembled at all, so it precedes the grammar it shares that gate with.
- **Dependencies**: → requires: My Roles; → requires: Tension Reads
- **Status**: pending

### 73. Post-Create Validity Read

- **Score**: MoSCoW Must Have
- **Framework**: MoSCoW (Must Have)
- **Rationale**: Independent root with every dependency already specified, and it unblocks the outcome change at #76 that stops a dead proposal reading as a success.
- **Dependencies**: → requires: Proposal Creation; → requires: Proposal Reads
- **Status**: pending

### 74. Legacy Identifier Request

- **Score**: MoSCoW Must Have
- **Framework**: MoSCoW (Must Have)
- **Rationale**: Replaces the harvest at #78 as the Must-tier answer to the identifier gap: a query parameter on reads that already exist, verified live to return the number the write path needs — the cheapest removal of the blocker that cost a full drafter run. Carries a retirement clock, so it ships alongside the ask at #81 rather than instead of it.
- **Dependencies**: → requires: Role Reads; → requires: Actor Read
- **Status**: pending

### 75. Agent-Facing Grammar Reference

- **Score**: MoSCoW Must Have
- **Framework**: MoSCoW (Must Have)
- **Rationale**: The single deliverable that removes most of the write path's friction; follows the facts at #71 that it renders and precedes the gate at #77 that consults it.
- **Dependencies**: → requires: Change-Set Grammar Facts
- **Status**: pending

### 76. Invalid-Create Outcome

- **Score**: MoSCoW Must Have
- **Framework**: MoSCoW (Must Have)
- **Rationale**: Needs the read-back at #73 before an invalid create can be rendered as a failure, and it introduces a new outcome touching the exit-code registry, so it is specified after the read rather than folded into it.
- **Dependencies**: → requires: Post-Create Validity Read; → requires: Diagnostic Normalization; → requires: Output-Aware Failure Rendering; → requires: Exit-Code Convention
- **Status**: pending

### 77. Pre-Assembly Grammar Consultation

- **Score**: MoSCoW Must Have
- **Framework**: MoSCoW (Must Have)
- **Rationale**: Last of the Must tier because it consumes both #75 and #72; without it the reference ships unconsulted, which is the failure the tier exists to prevent.
- **Dependencies**: → requires: Agent-Facing Grammar Reference; → requires: Proposal Drafting Path; → requires: Circle Routing Rule
- **Status**: pending

### 78. Identifier Mapping Harvest

- **Score**: MoSCoW Should Have
- **Framework**: MoSCoW (Should Have)
- **Rationale**: Demoted from Must by the spec refresh: the read route at #74 now covers roles and actors, leaving this the accountability, domain, and policy residue no read exposes — real, but narrower and no longer the write path's blocker.
- **Dependencies**: → requires: Proposal Reads
- **Status**: pending

### 79. Identifier Resolution by Content Match

- **Score**: MoSCoW Should Have
- **Framework**: MoSCoW (Should Have)
- **Rationale**: Turns #78's harvested numbers into usable answers, so it follows directly; matching on accountability description text rather than role names, since names proved ambiguous and roles are now served by #74.
- **Dependencies**: → requires: Identifier Mapping Harvest; → requires: Role Domains; → requires: Role Policies
- **Status**: pending

### 80. Capture-Time Detail Flags

- **Score**: MoSCoW Should Have
- **Framework**: MoSCoW (Should Have)
- **Rationale**: An independent root, and confirmed still necessary — the refreshed spec leaves tension creation byte-identical, so the refusal was not fixed upstream. Follows the Must tier because the write path survives it.
- **Dependencies**: → requires: Tension Capture; → requires: Tension Update
- **Status**: pending

### 81. Capture-Then-Update Guidance

- **Score**: MoSCoW Should Have
- **Framework**: MoSCoW (Should Have)
- **Rationale**: Follows #80, whose resolution determines what the refusal should say — the diagnostic cannot be written before the flag's fate is settled.
- **Dependencies**: → requires: Diagnostic Normalization; → requires: Capture-Time Detail Flags
- **Status**: pending

### 82. Identifier Prompt Before Assembly

- **Score**: MoSCoW Should Have
- **Framework**: MoSCoW (Should Have)
- **Rationale**: Needs the gate at #77 to exist before a prompt can ride on it; deliberately independent of #74/#78/#79 so the unconditional floor is neither blocked behind them nor lost when the transition bridge retires.
- **Dependencies**: → requires: Proposal Drafting Path; → requires: Pre-Assembly Grammar Consultation
- **Status**: pending

### 83. Change-Type Builders

- **Score**: MoSCoW Could Have
- **Framework**: MoSCoW (Could Have)
- **Rationale**: The larger CLI investment against a problem #75 already solves, so it trails the reference despite sharing its root at #71; its spec-fidelity objection has weakened now that the change types are a published enum, but its cost has not.
- **Dependencies**: → requires: Change-Set Grammar Facts
- **Status**: pending

### 84. Change-Set Assembly

- **Score**: MoSCoW Could Have
- **Framework**: MoSCoW (Could Have)
- **Rationale**: Terminal item in the new graph — needs #83's builders before parts can be composed into a change set.
- **Dependencies**: → requires: Change-Type Builders; → requires: Proposal Creation
- **Status**: pending
