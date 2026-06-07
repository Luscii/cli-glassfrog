# Glassfrog CLI — Roadmap

This roadmap sequences the problems in `ISSUE-TREE.md`. Its spine is VISION success criterion #2 — *an AI agent reads a practitioner's roles, then submits a proposal end-to-end* — front-loaded by the prerequisite that the CLI must exist at all. The foundation has landed (the CLI runs, authenticates, resolves its connection) and the first self-service reads plus client-hardening (pagination, rate limits) are in flight. With those wrapping up, Now broadens beyond the live work-front to take on the things that make the tool usable and shippable: agent-legible output, packaging it as a self-contained executable, and the CI/release pipeline that guards every change. Next shapes failure legibility and the full read surface; Later holds the governance write path and a review-quality enhancement. Sequence is order, not schedule.

The three top-level areas — *Project Foundation*, *Client Foundation*, and *Endpoint Commands* — are organizational parents; their sixteen child problems are what's sequenced below.

## Now

The live work-front (in-flight reads and client-hardening) plus the output, distribution, and pipeline work that makes the tool usable and shippable.

1. **No Runnable CLI** *(Project Foundation)* — nothing can be built or run until the CLI exists as a runnable skeleton with a command framework; hard prerequisite for every other problem. (addressed)
2. **Unauthenticated Access** *(Client Foundation)* — every command depends on a proven org + person identity; without it nothing reaches the API. (addressed)
3. **Undefined Connection Settings** *(Client Foundation)* — the CLI must know which token, organization, and base URL to use before any call; credential discovery/storage and connection resolution have both landed. (addressed)
4. **Self-Service Reads** *(Endpoint Commands)* — the smallest real vertical slice (`/me`, my roles, my actions, my projects); all four have now shipped end-to-end. (addressed)
5. **Silent Truncation** *(Client Foundation)* — pagination becomes load-bearing once reads return large sets or the org tree (CONSTITUTION VI); now in the Score pipeline. (in score)
6. **Getting Throttled** *(Client Foundation)* — rate-limit handling matters as read volume grows (CONSTITUTION X); now in the Score pipeline. (in score)
7. **Unconsumable Output** *(Client Foundation)* — make output agent-parseable and human-readable (VISION principle 3); the in-flight self-service reads produce output this shapes, so it lands alongside them.
8. **Runtime-Dependent Distribution** *(Project Foundation)* — ship the CLI as a standalone, dependency-free executable (CONSTITUTION XII); the release story, and a solution is already modeled.
9. **No Automated Pipeline** *(Project Foundation)* — a CI quality gate (lint + tests) plus release drafting that guards every change before it reaches main; clusters with *Runtime-Dependent Distribution* (drafting the version ↔ building and publishing the binary).

## Next

Shape failure legibility and broaden the read surface.

- **Opaque Failures** *(Client Foundation)* — fail legibly with a cause and a next step (CONSTITUTION II + III).
- **Governance Reads** *(Endpoint Commands)* — the full read surface (roles, circles, accountabilities, domains, policies, projects); extends the proven self-service slice.

## Later

The governance write path, its write-specific concerns, and a review-quality enhancement.

- **Shallow AI Reviews** *(Project Foundation)* — better-grounded Copilot PR reviews via a configured environment; a Maintainer-facing enhancement that pairs with the pipeline work.
- **Tension Capture** *(Endpoint Commands)* — recording a tension is where the write half of VISION success #2 begins.
- **Proposal Write-Flow** *(Endpoint Commands)* — the multi-step governance write path; the largest feature, and Premium-gated (related to *Unsignalled Plan Limits*).
- **Clobbered Changes** *(Client Foundation)* — optimistic concurrency only becomes relevant once writes exist.
- **Unsignalled Plan Limits** *(Client Foundation)* — feature-gating matters mostly for Premium / `ai_integration` endpoints, so it pairs with the write path.
