# Glassfrog CLI — Roadmap

This roadmap sequences the problems in `ISSUE-TREE.md`. Its spine is VISION success criterion #2 — *an AI agent reads a practitioner's roles, then submits a proposal end-to-end* — now front-loaded by the prerequisite that the CLI must exist at all: nothing can be built or run until there's a runnable tool. So the order is: get a runnable CLI, give it identity and a first read command, then broaden reads and harden the tool (including standalone packaging), and finally take on the governance write path. Sequence is order, not schedule.

The three top-level areas in the issue tree — *Project Foundation*, *Client Foundation*, and *Endpoint Commands* — are organizational parents; their fourteen child problems are what's sequenced below.

## Now

Get a runnable CLI that can authenticate and serve one real read command.

1. **No Runnable CLI** *(Project Foundation)* — nothing can be built or run until the CLI exists as a runnable skeleton with a command framework; hard prerequisite for every other problem. (addressed)
2. **Unauthenticated Access** *(Client Foundation)* — every command depends on a proven org + person identity; without it nothing reaches the API. (addressed)
3. **Undefined Connection Settings** *(Client Foundation)* — the CLI must know which token, organization, and base URL to use before any call; pairs with authentication. The token half rides on Token Authentication; the base URL half (Connection Resolution) is modeled but unbuilt. (in score)
4. **Self-Service Reads** *(Endpoint Commands)* — the smallest real vertical slice (`/me`, my roles) that exercises the whole chain; the read half of VISION success #2. (in score)

## Next

Harden the tool for real use and broaden the read surface.

- **Runtime-Dependent Distribution** *(Project Foundation)* — package the CLI as a standalone, dependency-free executable (CONSTITUTION XII). Deferred from Now: development can proceed against a runnable CLI before packaging is solved, but this must land before the tool ships to operators.
- **Unconsumable Output** *(Client Foundation)* — make output agent-parseable and human-readable (VISION principle 3, CONSTITUTION II). The dual-output candidate note rides along. The Now self-service slice needs basic output; this is where it's properly shaped.
- **Opaque Failures** *(Client Foundation)* — fail legibly with a cause and a next step (CONSTITUTION II + III).
- **Governance Reads** *(Endpoint Commands)* — the full read surface (roles, circles, accountabilities, domains, policies, projects); extends the proven self-service slice.
- **Silent Truncation** *(Client Foundation)* — pagination becomes load-bearing once reads return large sets or the org tree (CONSTITUTION VI).
- **Getting Throttled** *(Client Foundation)* — rate-limit handling matters as read volume grows (CONSTITUTION X).

## Later

The governance write path and its write-specific concerns.

- **Tension Capture** *(Endpoint Commands)* — recording a tension is where the write half of VISION success #2 begins.
- **Proposal Write-Flow** *(Endpoint Commands)* — the multi-step governance write path; the largest feature, and Premium-gated (related to *Unsignalled Plan Limits*).
- **Clobbered Changes** *(Client Foundation)* — optimistic concurrency only becomes relevant once writes exist.
- **Unsignalled Plan Limits** *(Client Foundation)* — feature-gating matters mostly for Premium / `ai_integration` endpoints, so it pairs with the write path.
