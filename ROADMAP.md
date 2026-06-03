# Glassfrog CLI — Roadmap

This roadmap sequences the problems in `ISSUE-TREE.md`. Its spine is VISION success criterion #2 — *an AI agent reads a practitioner's roles, then submits a proposal end-to-end*. So the order proves the **read** chain first (with the client foundation every command depends on), broadens the read surface, and only then takes on the **governance write path**, which is the most complex and is Premium-gated. Sequence is order, not schedule.

The two top-level areas in the issue tree — *Client Foundation* and *Endpoint Commands* — are organizational parents; their twelve child problems are what's sequenced below.

## Now

Stand up one faithful read command, end-to-end.

1. **Unauthenticated Access** *(Client Foundation)* — every command depends on a proven org + person identity; without it nothing reaches the API. Grounds in PROJECT's single-org+person constraint.
2. **Undefined Connection Settings** *(Client Foundation)* — the CLI must know which token, organization, and base URL to use before any call; pairs naturally with authentication.
3. **Self-Service Reads** *(Endpoint Commands)* — the smallest real vertical slice (`/me`, my roles) that exercises the whole chain; the read half of VISION success #2.
4. **Unconsumable Output** *(Client Foundation)* — the first command must be agent-parseable and human-readable to be usable (VISION principle 3, CONSTITUTION II). The dual-output candidate note rides along.
5. **Opaque Failures** *(Client Foundation)* — the command must fail legibly with a cause and a next step (CONSTITUTION II + III).

## Next

Broaden the read surface and harden it for scale.

- **Governance Reads** *(Endpoint Commands)* — the full read surface (roles, circles, accountabilities, domains, policies, projects); extends the proven self-service slice.
- **Silent Truncation** *(Client Foundation)* — pagination becomes load-bearing once reads return large sets or the org tree (CONSTITUTION VI).
- **Getting Throttled** *(Client Foundation)* — rate-limit handling matters as read volume grows (CONSTITUTION X).

## Later

The governance write path and its write-specific concerns.

- **Tension Capture** *(Endpoint Commands)* — recording a tension is where the write half of VISION success #2 begins.
- **Proposal Write-Flow** *(Endpoint Commands)* — the multi-step governance write path; the largest feature, and Premium-gated (related to *Unsignalled Plan Limits*).
- **Clobbered Changes** *(Client Foundation)* — optimistic concurrency only becomes relevant once writes exist.
- **Unsignalled Plan Limits** *(Client Foundation)* — feature-gating matters mostly for Premium / `ai_integration` endpoints, so it pairs with the write path.
