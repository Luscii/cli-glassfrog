# Glassfrog CLI — Roadmap

This roadmap sequences the problems in `ISSUE-TREE.md`. Its spine is VISION success criterion #2 — *an AI agent reads a practitioner's roles, then submits a proposal end-to-end* — front-loaded by the prerequisite that the CLI must exist at all. The read half of that spine has now broadly landed: the foundation runs and authenticates, the self-service and governance reads have shipped (org tree still following), cross-model search is done, and the actor-governance reads have landed. With the read surface mature, the live work-front is what makes the tool usable, shippable, and legible — agent-parseable output, the CI/release pipeline — and the project has crossed into the **write half**: tension capture has substantially shipped, leaving *Proposal Write-Flow* as the remaining frontier. Next holds a maintainability refactor of the settings layer (in Score), the now-landed search, and the newly-scoped agent operating surface — packaging how an agent drives the CLI, sequenced to follow the write path it wraps. The **write path is now the active frontier**: Now holds the optimistic-concurrency guard (Clobbered Changes — version capture, guarded writes, and stale-write surfacing have all landed), the *Proposal Write-Flow* it protects (now modeled), and the *Unsignalled Plan Limits* signal that write path needs (now modeled). Later holds a deferred refinement of the write path (typed change construction) and a Maintainer-facing review-quality enhancement. Sequence is order, not schedule.

The three organizational parent areas — *Project Foundation*, *Client Foundation*, and *Endpoint Commands* — contribute the child problems sequenced below; a fourth root problem, *Unequipped Agent Operators* (the agent operating surface), is itself sequenced directly.

## Now

The landed foundation and client-hardening, the broad read surface (self-service, governance, search, and actor reads), the output/pipeline/failure-legibility work-front, and the write path now the active frontier — tension capture landed, the optimistic-concurrency guard landed, and the proposal write-flow plus its plan-gating signal ahead.

1. **No Runnable CLI** *(Project Foundation)* — nothing can be built or run until the CLI exists as a runnable skeleton with a command framework; hard prerequisite for every other problem. (addressed)
2. **Unauthenticated Access** *(Client Foundation)* — every command depends on a proven org + person identity; without it nothing reaches the API. (addressed)
3. **Undefined Connection Settings** *(Client Foundation)* — the CLI must know which token, organization, and base URL to use before any call; credential discovery/storage and connection resolution have both landed. (addressed)
4. **Self-Service Reads** *(Endpoint Commands)* — the smallest real vertical slice (`/me`, my roles, my actions, my projects); all four have shipped end-to-end. (addressed)
5. **Silent Truncation** *(Client Foundation)* — pagination is load-bearing once reads return large sets or the org tree (CONSTITUTION VI); the pagination spec has landed. (addressed)
6. **Getting Throttled** *(Client Foundation)* — rate-limit handling matters as read volume grows (CONSTITUTION X); the rate-limit spec has landed. (addressed)
7. **Runtime-Dependent Distribution** *(Project Foundation)* — ship the CLI as a standalone, dependency-free executable (CONSTITUTION XII); the self-contained build and install script have landed (Homebrew tap and npm wrapper channels still in flight). (addressed)
8. **Opaque Failures** *(Client Foundation)* — fail legibly with a cause and a next step (CONSTITUTION II + III); error extraction, diagnostic normalization, and output-aware failure rendering have all landed. (addressed)
9. **Unconsumable Output** *(Client Foundation)* — make output agent-parseable and human-readable (VISION principle 3); structured serialization, format selection, and user-defined templates have landed, with templated human rendering following. (in score)
10. **No Automated Pipeline** *(Project Foundation)* — a CI quality gate (lint + tests) plus release drafting that guards every change before it reaches main; release drafting, PR administration, and main-branch verification have landed, with PR validation following. Clusters with *Runtime-Dependent Distribution* (drafting the version ↔ building and publishing the binary). (in score)
11. **Governance Reads** *(Endpoint Commands)* — the full read surface (roles, circles, accountabilities, domains, policies, projects); role, domain, policy, and project reads have landed, with the organization tree following. (in score)
12. **Tension Capture** *(Endpoint Commands)* — where the write half of VISION success #2 begins; capture, reads, discard, and subrole roll-up have landed, with tension update following. (in score)
13. **Actors Disconnected from Governance** *(Endpoint Commands)* — read the actor↔governance link in both directions: who fills a role (whom to contact) and an actor's governance footprint (roles, accountabilities, domains, purposes); role-fillers, the actor read + assignments, and subrole filler roll-up have all landed. Relates to *Tension Capture*. (addressed)
14. **Clobbered Changes** *(Client Foundation)* — optimistic concurrency for governance writes; version capture on read, guarded writes (`If-Match`), and stale-write (`412`) surfacing have all landed. Lands ahead of the *Proposal Write-Flow* it protects. (addressed)
15. **Proposal Write-Flow** *(Endpoint Commands)* — the multi-step governance write path (create → propose → respond → accepted); the largest feature and the remaining frontier of VISION success #2, modeled as the **Governance Proposals** solution (creation / reads / advance / withdraw / respond, `changes[]` pass-through) and ready to prioritize. (in score)
16. **Unsignalled Plan Limits** *(Client Foundation)* — surface a clear "not available on your plan" signal for Premium / `ai_integration` endpoints (403); the plan-gating signal the proposal write path needs (pairs with *Proposal Write-Flow*), modeled as the **Plan-Limit Signalling** solution. (in score)

## Next

A maintainability refactor of the settings layer, now that its call sites have landed; cross-model search, which has itself already landed; and the newly-scoped agent operating surface that equips an agent to drive the CLI.

- **Duplicated Setting Resolution** *(Client Foundation)* — collapse the per-setting flag→env→.glassfrogrc→default chain into one composable resolver before more settings copy the seam; the call-site retrofit has landed, with the composable source resolution following. Relates to *Undefined Connection Settings* and *Unconsumable Output*, whose landed call sites it retrofits. (in score)
- **Undiscoverable Governance** *(Endpoint Commands)* — cross-model search by topic/relevance over the record (roles, policies, role-fillers); extends the read surface beyond direct lookups and supports working a tension (relates to *Tension Capture*). (addressed)
- **Unequipped Agent Operators** *(Agent Operating Surface)* — the AI agent driving the CLI has no packaged operating knowledge, so it rediscovers how to operate the CLI each session and can mis-drive it or run ungated writes; strategically core to VISION success #2, but it packages a surface (especially the write path) still being completed in Now, so it follows rather than competes. Relates to *Runtime-Dependent Distribution*.

## Later

A deferred refinement of the proposal write path, and a Maintainer-facing review-quality enhancement.

- **Unguided Change Construction** *(Endpoint Commands)* — hand-assembling a proposal's free-form `changes[]` is error-prone and demands prior knowledge of each command's shape; a deferred refinement of the now-active write path (candidate: typed change builders), consciously pushed out of the first cut since `changes[]` ships as pass-through. Relates to *Proposal Write-Flow*.
- **Shallow AI Reviews** *(Project Foundation)* — better-grounded Copilot PR reviews via a configured environment; a Maintainer-facing enhancement that pairs with the pipeline work.
