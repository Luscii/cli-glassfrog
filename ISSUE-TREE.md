# Glassfrog CLI — Issue Tree

Problems for the Glassfrog v5 CLI, decomposed into a shared client foundation and the endpoint commands that are the project's endgoal. Bootstrapped from the 2026-06-03 explore session.

> How can practitioners — and the AI agents acting for them — read and change Glassfrog governance from the command line, faithfully to the v5 API?

---

* Client Foundation — cross-cutting concerns every command depends on
  * Unauthenticated Access — the CLI has no way to prove it's acting as a specific org + person, so Glassfrog can't authorize its calls
    + affects: AI agent
    + affects: Practitioner
  * Undefined Connection Settings — the CLI doesn't know which token, organization, or base URL to use, or where to read them from
    + affects: Practitioner
  * Unconsumable Output — results aren't shaped for an AI agent to parse reliably or for a human to read
    + affects: AI agent
    + affects: Practitioner
    + candidate: Dual output — emit JSON for agents and a human-readable format for people
  * Opaque Failures — when a call fails, the caller can't tell what went wrong or what to do next
    + affects: AI agent
    + affects: Practitioner
  * Unsignalled Plan Limits — plan/flag-gated endpoints (ai_integration, Premium) reject with 403 and no clear "not available on your plan" signal
    + affects: Practitioner
  * Silent Truncation — large result sets and the org tree get cut off without the caller knowing
    + affects: AI agent
    + affects: Practitioner
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
  * Proposal Write-Flow — the multi-step governance write path: create → propose → respond → accepted
    + affects: Practitioner
    + related-to: Unsignalled Plan Limits
