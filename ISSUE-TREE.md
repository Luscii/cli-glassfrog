# Glassfrog CLI — Issue Tree

Problems for the Glassfrog v5 CLI, decomposed into a project foundation, a shared client foundation, and the endpoint commands that are the project's endgoal. Bootstrapped from the 2026-06-03 explore session.

> How can practitioners — and the AI agents acting for them — read and change Glassfrog governance from the command line, faithfully to the v5 API?

---

* Project Foundation — prerequisites for the CLI to exist and run before any command can be built
  * No Runnable CLI — no project skeleton or command framework exists to build any command on
    + affects: Maintainer
  * Runtime-Dependent Distribution — the CLI can't run without a separately-installed runtime, so it won't run where operators need it
    + affects: Practitioner
    + affects: AI agent
    + affects: Maintainer
  * No Automated Pipeline — without CI/CD, every change is linted, tested, triaged, and released by hand, so regressions and inconsistencies can reach main and releases go out unguarded
    + affects: Maintainer
    + related-to: Runtime-Dependent Distribution
    + candidate: PR Checks Workflow — on pull request, run lint + tests and auto-apply administrative labels (pattern: template pull-request.yml)
    + candidate: Release Drafting on Merge — on merge to main, draft a GitHub release (release-drafter, label-driven semver) and re-run tests as a safety check (pattern: template draft-release.yml)
  * Shallow AI Reviews — Copilot reviews PRs without a configured project environment, so its feedback is poorly grounded and misses project-specific context
    + affects: Maintainer
    + candidate: Copilot Review Environment — configure the repo's Copilot environment (build/test/context setup) so its PR reviews are better-grounded
  * Spec Drift Goes Unnoticed — the vendored API contract can fall months behind the published one with nothing surfacing it, so the CLI gets built against a superseded contract and capabilities it could already offer stay invisible
    + affects: Maintainer
    + affects: AI agent
    + affects: Practitioner
    + related-to: No Automated Pipeline
    + candidate: Vendored-Spec Drift Check — re-fetch the published contract on a schedule or in CI and surface any difference from the vendored copy, since neither the contract's version field nor the vendor's own changelog reliably signals that it changed
* Client Foundation — cross-cutting concerns every command depends on
  * Unauthenticated Access — the CLI has no way to prove it's acting as a specific org + person, so Glassfrog can't authorize its calls
    + affects: AI agent
    + affects: Practitioner
  * Undefined Connection Settings — the CLI doesn't know which token, organization, or base URL to use, or where to read them from
    + affects: Practitioner
  * Duplicated Setting Resolution — each configurable setting (token, base URL, output format) re-implements the same flag→env→.glassfrogrc→default precedence chain, so adding a setting copies the OS seam, the chain skeleton, the Source enum, and the validation-error shape — and the copies can drift
    + affects: Maintainer
    + related-to: Undefined Connection Settings
    + related-to: Unconsumable Output
    + candidate: Composable setting resolver — resolveSetting([...sources]) takes an ordered list of source providers and returns the first that yields a value, where ARRAY ORDER IS PRECEDENCE (flag, env, file); sources are fromFlags(main, alias), fromEnv(envVar), fromFile(key) with the file source partially applied from retrieveFromSettingsFile(".glassfrogrc") so the file is bound once; a source yielding nothing is skipped, an optional trailing default source backstops the chain (absence stays valid where there is no default, e.g. token), and per-source validation collapses the per-type Source enums + validation errors into one shared shape; retrofit the 3 call sites (token 005, base-url 008, output 020), behavior-preserving with tests staying green; the .glassfrogrc file tier (rcfile.Resolve) is already shared and stays as the file source
  * Unconsumable Output — results aren't shaped for an AI agent to parse reliably or for a human to read
    + affects: AI agent
    + affects: Practitioner
    + candidate: --output flag — caller selects the output format per invocation: human-readable (full/compact), JSON, possibly YAML
  * Opaque Failures — when a call fails, the caller can't tell what went wrong or what to do next
    + affects: AI agent
    + affects: Practitioner
  * Unsignalled Plan Limits — plan/flag-gated endpoints (ai_integration, Premium) reject with 403 and no clear "not available on your plan" signal
    + affects: Practitioner
  * Silent Truncation — large result sets and the org tree get cut off without the caller knowing
    + affects: AI agent
    + affects: Practitioner
  * Oversized Content Unsignalled — the server reports when a saved record's content is too large to read back in one response, and nothing surfaces that report, so the caller can't tell that what it read is incomplete
    + affects: AI agent
    + affects: Practitioner
    + related-to: Silent Truncation
    + candidate: Content-Size Warning Surfacing — surface the server's content-size warnings on the records that carry them, so an agent knows a record's content exceeded what one response can return
  * Getting Throttled — unmanaged request volume trips the per-org hourly rate limit, throttling the whole organization
    + affects: Practitioner
  * Clobbered Changes — concurrent governance edits overwrite each other when writes skip version checks
    + affects: Practitioner
* Endpoint Commands — expose each in-scope v5 operation as a command (the endgoal)
  * Governance Reads — read roles, circles, accountabilities, domains, policies, and projects
    + affects: Practitioner
  * Self-Service Reads — read "what's mine": me, my roles, actions, and projects
    + affects: Practitioner
  * Tension Capture — record a tension as the entry point to a proposal
    + affects: Practitioner
    * Tension Details Rejected at Capture — supplying a tension's agenda label or meeting type while capturing it is refused outright, and the refusal names a length rule that doesn't apply, so nothing tells the operator that capturing bare and adding the label afterwards is the way through
      + affects: Practitioner
      + affects: AI agent
      + candidate: Bare Capture Then Label — either send the label and meeting type in the shape the server accepts at capture time, or retire the capture-time flags and document the capture-bare-then-update path, translating the misleading length error into a diagnostic that names the known capture-time limitation
  * Proposal Write-Flow — the multi-step governance write path: create → propose → respond → accepted
    + affects: Practitioner
    + related-to: Unsignalled Plan Limits
    * Unguided Change Construction — building a proposal means hand-writing each governance change command (`CreateRole`, `UpdateAccountability`, …) as free-form JSON, because the CLI offers no per-type structure or guidance for the change set, so constructing a valid `changes[]` is error-prone and demands prior knowledge of each command's shape
      + affects: Practitioner
      + affects: AI agent
      + candidate: Typed Change Builders — a CLI command (or flag set) per change type that shapes the proposal `changes[]` payload so each command's fields are explicit rather than hand-written free-form JSON
      + candidate: Verified Change-Set Grammar Reference — a grammar reference the drafting path loads before assembling: the part shapes the server accepts (a role-operation wrapper carrying `Create*`/`Update*`/`Remove*` children; a top-level `CreatePolicy` for a circle's own policy) and the shapes it rejects (top-level `Add*`; a role update self-targeting the circle from inside its own governance), so the shape is known before the write rather than discovered by refused round-trips
    * Success Reported for a Dead Proposal — the CLI reports a created proposal and its id as a success while the server has already marked the draft invalid with no available transitions, so the operator confirms a gated write that produced an object nothing can move forward and finds out only later
      + affects: Practitioner
      + affects: AI agent
      + candidate: Validity Read-Back on Create — after creating a proposal, read the draft back and surface its validity and any validation alerts before declaring success, so an invalid create reads as a failure carrying the alert text rather than a success carrying an id
    * Change Targets Unidentifiable from the CLI — a proposal's change can only name an existing role by its legacy numeric identifier, which no CLI read exposes, so every write touching existing governance sends the operator out to the web UI to read the number off a URL mid-flow
      + affects: Practitioner
      + affects: AI agent
      + related-to: Governance Reads
      + candidate: Numeric Identifier on Governance Reads — expose the legacy numeric identifier in read output wherever an API surface carries it
      + candidate: Ask for the Identifier Up Front — the drafting path asks the operator for the numeric identifier before assembling the change set, naming the web-UI URL pattern it can be read from, instead of failing into a not-found on the hashed id
    * No Way to Discard a Dead Draft — a draft the server has invalidated carries no available transitions, and the CLI offers no way to remove it, so clearing it means leaving the command line for the web UI
      + affects: Practitioner
      + candidate: Draft Discard — expose draft deletion as a gated write where the API carries it, and where it does not, document the web-UI-only cleanup in the circulation path
    * Lagging Counts Read as No Response — a proposal's response summary lags the operator's own recorded response, so unchanged counts read as "nobody has answered yet" and only a refused duplicate response settles whether the operator already responded
      + affects: Practitioner
      + affects: AI agent
      + candidate: Duplicate-Response Refusal as the Signal — document that the refusal of a second response is the definitive already-answered signal and that an unchanged response summary never means no response was recorded
    * Proposal Circle Not Choosable — a proposal carries no circle of its own and lands in the circle of the anchor tension's sensing role, so a change to a circle's own governance has to be proposed from the parent circle, anchored on a tension sensed by a role the operator fills there
      + affects: Practitioner
      + affects: AI agent
      + candidate: Circle Routing Pre-Check — before assembling a change set, establish which circle the change must land in, which tension can anchor it there, and whether the operator fills a sensing role in that circle
    * Proposals Touching My Roles Unfindable — proposals can't be filtered by what their changes affect, so finding which ones bear on the roles the operator fills means reading every proposal's change set
      + affects: Practitioner
      + affects: AI agent
      + related-to: Self-Service Reads
      + candidate: Affected-Element Proposal Filter — find proposals by the governance elements their changes target, rather than only by the circle the proposal itself sits in
  * Undiscoverable Governance — when working a tension, the operator can't find which roles, policies, or role-fillers are relevant without already knowing where to look; nothing lets them search the record by topic or relevance
    + affects: Practitioner
    + affects: AI agent
    + related-to: Tension Capture
    + candidate: Search — cross-model full-text search (`GET /search` → `search`) across roles, projects, actions, notes, skills, actors, policies, and domains, ranked by relevance, with a required `query` (websearch syntax) and an optional `types` filter; paginated
  * Actors Disconnected from Governance — when working a tension, the operator can't bridge people and the governance record in either direction: from a role to the actor to contact about it, or from an actor to the governance they hold
    + affects: Practitioner
    + affects: AI agent
    + related-to: Tension Capture
    + related-to: Undiscoverable Governance
    * Who to Contact for a Role — given a role relevant to a tension, the operator can't tell which actor fills it, so they don't know whom to reach out to
    * An Actor's Governance Footprint — given an actor, the operator can't see what they do: the roles they fill and the accountabilities, domains, and purposes those carry
    * Focus and Election Invisible — the operator can't see which focus a filler holds in a role, or whether they were elected and until when, so a role's real staffing can't be read from the tree
      + affects: Practitioner
      + affects: AI agent
      + candidate: Assignments on the Tree Read — widen the tree read's accepted per-node include set to the value that carries assignment focus and election data, which the API serves but the CLI's closed set currently rejects
  * Agent Capability Unverifiable — when an AI agent fills a role, nothing shows which skills it holds or which skills the role expects, so the practitioner can't tell whether the filler is equipped for the role it holds
    + affects: Practitioner
    + affects: AI agent
    + related-to: An Actor's Governance Footprint
    + candidate: Agent and Skill Reads — read agents, the skills they hold, and the skills a role expects, including the links in both directions; gated behind the ai_integration feature flag
  * Role Filling Requires Assembly — everything needed to fill a role is spread across separate reads the caller must stitch together, so an agent orienting to a role spends many calls and can still miss a part
    + affects: AI agent
    + affects: Practitioner
    + related-to: Governance Reads
    + related-to: Self-Service Reads
    + candidate: Role Context Read — read the single context document that carries everything needed to fill a role, for the caller and for an agent, rather than assembling it from per-resource reads
  * Organizational Conversation Out of Reach — discussion bearing on governance work happens in channels the CLI can't read or write, so an agent acting for a practitioner works blind to it and can't respond in place; a surface the API exposes ahead of the web app, so the need is anticipated rather than felt today
    + affects: Practitioner
    + affects: AI agent
    + candidate: Channel and Message Access — read channels, their messages, and per-caller read state, and author messages in place
* Unequipped Agent Operators — the AI agent driving the CLI has no packaged operating knowledge (command surface, auth setup, output parsing, exit-code handling, write-safety), so it rediscovers how to drive the CLI each session and can mis-drive it or run ungated writes
  + affects: AI agent
  + affects: Practitioner
  + related-to: Runtime-Dependent Distribution
  + candidate: Claude Code Plugin — repo-shipped plugin (its own marketplace) bundling a glassfrog-cli operator skill, a read-only governance-navigator agent, and a PreToolUse hook that gates writes; leans on existing `glassfrog auth login` for credentials
  * Nothing Learned Is Kept — the operating surface discards what live payloads already taught it, so every session re-derives the same identifier resolutions and a target resolved days earlier is declared unresolvable again
    + affects: AI agent
    + affects: Practitioner
    + related-to: Change Targets Unidentifiable from the CLI
    + candidate: Learned Identifier Mapping — harvest hashed-to-numeric identifier pairs from every live proposal payload the tools read and consult them before declaring a change target unresolvable, with content-matching against current governance texts as the documented fallback carrying the confidence flag it already reports
  * Non-Governance Writes Ungated — the write-safety gate is scoped to the governance record by design, so a write surface that isn't governance passes through without the confirmation every other write requires
    + affects: Practitioner
    + affects: AI agent
    + related-to: Organizational Conversation Out of Reach
    + candidate: Gate Scope Decision — decide deliberately whether the write-safety gate widens to cover non-governance writes or states its governance-only boundary explicitly, rather than letting the gap open by omission
  * Call Shapes Not Packaged — the operating surface pins which commands an agent may run but never how to call them, so every path run interrogates the CLI for each leaf's flags before it can do any work, and the flag knowledge is the one part of the composed surface nothing regenerates on change
    + affects: AI agent
    + affects: Maintainer
    + related-to: Nothing Learned Is Kept
    + candidate: Generated Call-Shape Snapshot — derive each composed leaf's call shape from the CLI binary being shipped into a co-located generated artifact the agents read instead of asking `--help`, with a drift guard that regenerates and fails on divergence, so flag knowledge becomes exactly as guarded as the command names already are
  * Payloads Outsize the Work — the operating surface's composed reads pull each record's whole payload into the agent's context even when a traversal uses only a few fields, so every path spends far more of its limited context on unused data than the work needs
    + affects: AI agent
    + related-to: Unconsumable Output
    + related-to: Call Shapes Not Packaged
    + candidate: Templated Read Projections — render the composed read leaves through the CLI's `-o`/`--output` template feature (035) to project each payload to the minimal shape a path needs before it enters the subagent's context, in place of raw `--output json`
