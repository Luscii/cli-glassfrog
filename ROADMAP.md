# Glassfrog CLI — Roadmap

This roadmap sequences the problems in `ISSUE-TREE.md`. Its spine is VISION success criterion #2 — *an AI agent reads a practitioner's roles, then submits a proposal end-to-end* — front-loaded by the prerequisite that the CLI must exist at all. The foundation has landed (the CLI runs, authenticates, resolves its connection), the self-service reads ship end-to-end, and the client-hardening that was in flight — pagination and rate-limit handling — has now landed too. With that hardening done, the live work-front is what makes the tool usable, shippable, and legible: agent-parseable output, the standalone executable, the CI/release pipeline, the broadening read surface, and failure legibility. Next holds a maintainability refactor of the settings layer; Later holds the governance write path and a review-quality enhancement. Sequence is order, not schedule.

The three top-level areas — *Project Foundation*, *Client Foundation*, and *Endpoint Commands* — are organizational parents; their seventeen child problems are what's sequenced below.

## Now

The landed foundation and client-hardening, plus the live work-front: output, distribution, pipeline, the broadening read surface, and failure legibility.

1. **No Runnable CLI** *(Project Foundation)* — nothing can be built or run until the CLI exists as a runnable skeleton with a command framework; hard prerequisite for every other problem. (addressed)
2. **Unauthenticated Access** *(Client Foundation)* — every command depends on a proven org + person identity; without it nothing reaches the API. (addressed)
3. **Undefined Connection Settings** *(Client Foundation)* — the CLI must know which token, organization, and base URL to use before any call; credential discovery/storage and connection resolution have both landed. (addressed)
4. **Self-Service Reads** *(Endpoint Commands)* — the smallest real vertical slice (`/me`, my roles, my actions, my projects); all four have shipped end-to-end. (addressed)
5. **Silent Truncation** *(Client Foundation)* — pagination is load-bearing once reads return large sets or the org tree (CONSTITUTION VI); the pagination spec has landed. (addressed)
6. **Getting Throttled** *(Client Foundation)* — rate-limit handling matters as read volume grows (CONSTITUTION X); the rate-limit spec has landed. (addressed)
7. **Runtime-Dependent Distribution** *(Project Foundation)* — ship the CLI as a standalone, dependency-free executable (CONSTITUTION XII); the self-contained build has landed. (addressed)
8. **Unconsumable Output** *(Client Foundation)* — make output agent-parseable and human-readable (VISION principle 3); structured serialization and format selection have landed, with templated and failure-aware rendering following. (in score)
9. **No Automated Pipeline** *(Project Foundation)* — a CI quality gate (lint + tests) plus release drafting that guards every change before it reaches main; clusters with *Runtime-Dependent Distribution* (drafting the version ↔ building and publishing the binary). (in score)
10. **Governance Reads** *(Endpoint Commands)* — the full read surface (roles, circles, accountabilities, domains, policies, projects); role, domain, and policy reads have landed, with the organization tree following. (in score)
11. **Opaque Failures** *(Client Foundation)* — fail legibly with a cause and a next step (CONSTITUTION II + III); error extraction and diagnostic normalization have landed, with output-aware failure rendering following. (in score)

## Next

A maintainability refactor of the settings layer, now that its call sites have landed.

- **Duplicated Setting Resolution** *(Client Foundation)* — collapse the per-setting flag→env→.glassfrogrc→default chain into one composable resolver before more settings copy the seam; relates to *Undefined Connection Settings* and *Unconsumable Output*, whose landed call sites it would retrofit.

## Later

The governance write path, its write-specific concerns, and a review-quality enhancement.

- **Shallow AI Reviews** *(Project Foundation)* — better-grounded Copilot PR reviews via a configured environment; a Maintainer-facing enhancement that pairs with the pipeline work.
- **Tension Capture** *(Endpoint Commands)* — recording a tension is where the write half of VISION success #2 begins.
- **Proposal Write-Flow** *(Endpoint Commands)* — the multi-step governance write path; the largest feature, and Premium-gated (related to *Unsignalled Plan Limits*).
- **Clobbered Changes** *(Client Foundation)* — optimistic concurrency only becomes relevant once writes exist.
- **Unsignalled Plan Limits** *(Client Foundation)* — feature-gating matters mostly for Premium / `ai_integration` endpoints, so it pairs with the write path.
