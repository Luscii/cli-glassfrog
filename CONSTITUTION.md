# Glassfrog CLI Constitution

This document defines the enforceable principles that govern how the Glassfrog CLI is built. Each principle carries a detection mechanism: if a violation can't be observed, the principle is aspirational, not constitutional.

---

## Core Principles

### I. Spec Fidelity

**Every command MUST map to an operation defined in the Glassfrog API v5 spec. The CLI MUST NOT invent endpoints, parameters, or behaviors the spec does not define.**

*Rationale*: The spec is the contract. Divergence silently does the wrong thing to live governance data, and the whole project's value is being a faithful surface over the API.

*Detection*: The CLI's command and request surface can be diffed against `spec.yaml`. A command that calls an undefined operation, sends an undefined parameter, or relies on behavior the spec doesn't define is a violation. Contract tests assert each command's request shape matches a spec operation.

### II. Action Transparency (NON-NEGOTIABLE)

**The operator MUST always be able to tell what the CLI did and to which record. Every action MUST report the spec operation it invoked and the target resource, in machine-parseable form; every error MUST explain what went wrong and the next step.**

*Rationale*: The usual operator is an AI agent acting on a practitioner's behalf. Without legible, traceable actions, the agent acts on wrong assumptions and trust collapses.

*Detection*: An action whose output cannot be traced to a specific endpoint plus resource id is a violation. Output that only emits free-form human text with no structured form, or an error message lacking a cause and a next step, fails review.

### III. Fail Safe, Not Silent

**Errors MUST be obvious and recoverable, never hidden. The CLI MUST validate a write before sending it and MUST NOT leave governance in a partially-applied state.**

*Rationale*: A tool that breaks governance quietly is worse than no tool at all.

*Detection*: Swallowed errors, a multi-step write that applies partially with no rollback path, or a failure condition reported as success are violations.

*Anti-patterns*: swallowing errors; partially applied changes; auto-fixing without explicit intent; leaving the system in an inconsistent state.

### IV. Test-Driven Development (Red → Green → Refactor)

**Features MUST be built test-first: a failing test (RED) before implementation (GREEN), then refactor. User-facing behavior MUST have an executable acceptance scenario before the code that satisfies it.**

*Rationale*: TDD ensures we build it right; BDD acceptance scenarios ensure we build the right thing.

*Detection*: Code that lands without a preceding-then-passing test, or user-facing behavior with no acceptance scenario, is a violation.

### V. Composition over Monolith

**The CLI MUST be built from modular, independently-testable parts (per-resource command modules over a shared API client) that combine without entanglement. Adding a new command MUST NOT require changing unrelated ones.**

*Rationale*: The spec covers many resources; a monolith makes every addition risky and couples unrelated concerns.

*Detection*: A change to one resource's command that forces edits to unrelated commands, or hidden cross-module dependencies, is a violation.

*Anti-patterns*: tightly coupled command modules; hidden dependencies between resources; mixing concerns in one module.

### VI. Size-Aware by Design

**The CLI MUST handle large result sets and the organization tree within the API's pagination limits, and MUST NEVER silently truncate. When results are paged or capped, it MUST page through them or clearly signal the boundary.**

*Rationale*: Organizations and role trees can be large; silent truncation gives a false picture of governance.

*Detection*: Output that drops records beyond a page without indication, or a fetch that ignores `per_page` limits and assumes a single page is complete, is a violation.

### VII. Working Software

**Every commit and PR MUST include implementation together with its tests, and MUST validate and build. No code-only or test-only increments (except a RED test immediately followed by its GREEN implementation).**

*Rationale*: Non-validated code is not progress — it is technical debt. Keeping code and tests together makes every increment a verified, deliverable unit.

*Detection*: A commit or PR that fails lint/build/tests, or that ships implementation without tests (or vice versa) outside the RED→GREEN pair, is a violation.

### VIII. No Fabricated Data

**The CLI MUST present only data the API actually returned. It MUST NOT invent, guess, or fill placeholder values for fields the API did not provide.**

*Rationale*: Fabricated governance data misleads both the agent and the practitioner, and undermines spec fidelity.

*Detection*: Any output value not traceable to an API response — a synthesized, defaulted, or guessed value presented as real — is a violation.

### IX. Writes Require Explicit Intent

**No write or mutation MUST occur except as the direct result of an explicit write command. A read-shaped command (get/list/show) MUST NEVER mutate as a side effect.**

*Rationale*: An automated agent must not change live governance accidentally; the intent to write must be unambiguous.

*Detection*: Any read/list/get command path that issues a POST/PATCH/DELETE, or any mutation not gated behind an explicit write command, is a violation.

### X. Respect API Limits

**The CLI MUST honor the API's rate limits and concurrency model — backing off on `429` responses and using optimistic concurrency (`If-Match`/`ETag`) for updates rather than last-write-wins clobbering.**

*Rationale*: Ignoring limits gets the whole organization throttled; clobbering destroys concurrent governance work.

*Detection*: An update request that omits `If-Match` when an `ETag` is available, or a retry loop that ignores `429`/`Retry-After`, is a violation.

### XI. Governance via Proposals

**Changes to governance structure (roles, accountabilities, domains, policies) MUST go through a Proposal by default. The CLI MUST NOT expose any default command path that mutates governance structure directly. Direct governance management is permitted ONLY behind an explicit, clearly-named opt-in flag, and only when the caller holds the `role-manage-without-proposal` permission.**

*Rationale*: Governance changes that skip the proposal process bypass the organization's agreed way of changing itself, so the proposal path is the default and the norm. A narrow, explicit escape hatch is allowed for callers the API already trusts with `role-manage-without-proposal`, but it must be a deliberate, named choice — never a path an agent reaches by default.

*Detection*: Any governance-structure mutation reachable without the explicit opt-in flag — or any default command path that bypasses the `/proposals` flow — is a violation. The opt-in flag's presence on a command is the evidence of deliberate bypass; its absence on any governance-mutating path is a violation.

---

## When Principles Conflict

| Tension | Resolution | Reasoning |
|---|---|---|
| **Spec Fidelity vs Fail Safe / Respect API Limits** — the spec *permits* unconditional, last-write-wins updates by omitting `If-Match`. | Always use optimistic concurrency (send `If-Match`) even though the spec also allows omitting it. | Spec-permitted is not spec-required. Silently clobbering concurrent governance changes violates Fail Safe. Safety wins over the more permissive spec option. |
| **Writes Require Explicit Intent vs agent automation** — an AI-agent operator means no interactive confirmation prompts (Action Transparency favors non-interactive, automation-friendly behavior). | Intent is expressed by the explicit write command/flag itself, not by an interactive prompt. | Preserves both safety and automation: the command's existence is the intent, so an agent can act without a human in the loop while reads can never mutate. |

---

## Governance

This constitution is a living document and supersedes conflicting practices. Amendments require documented justification and a version bump, and follow the same rigor as code changes. Principles marked `[NEEDS DISCUSSION]` MUST be resolved before any code depends on the unresolved behavior. PR review checks changes against these principles; a violation blocks merging.
