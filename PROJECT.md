# Glassfrog CLI — Project Definition

---

## Domain

The Glassfrog CLI operates in the world of **Holacracy governance** as recorded in **Glassfrog**. Glassfrog is the system of record; the CLI is a faithful command-line surface over its v5 API. An organization's structure lives as a tree of **roles** (a **circle** is simply a role that contains sub-roles — v5 is role-centric, so circle references resolve to a supported role). Each role carries **accountabilities** (ongoing activities) and **domains** (areas it controls), and may have **policies**, **projects**, **actions**, **metrics**, and **checklist items**. People and AI agents — collectively **actors** — fill roles through **assignments**.

The defining shape of this domain is *how governance changes*. The governance structure itself (roles, accountabilities, domains, policies) is **read-only** through direct endpoints. The only way to change it is to raise a **tension** and submit a **proposal** — a governance change request whose `changes` carry commands like `CreateRole` or `UpdateAccountability`. A proposal moves from `draft` into circulation and is accepted when circle members respond without objection. This means the project's domain has a clean split: a broad **read** surface over the whole organization, and a **propose** surface that is the sanctioned path to alter governance. Operational items (tensions, actions, assignments, metrics, notes) can be edited directly, but governance flows through proposals — the API enforces the same discipline the project's vision commits to.

The CLI's distinctive context is that its operator is usually an **AI agent** acting on a **practitioner's** behalf, so the domain is expressed in machine-legible terms that map one-to-one onto spec operations.

---

## Vocabulary

| Term | Definition | Also known as |
|---|---|---|
| Glassfrog | The Holacracy governance platform the CLI talks to; the system of record. | — |
| Holacracy | The self-management/governance method Glassfrog implements. | — |
| Organization | The org an API key is scoped to. One key = one org + one person. | Org |
| Role | A position in the holacratic structure; holds accountabilities and domains. Read-only via direct endpoints. | — |
| Circle | A role that contains sub-roles. v5 is role-centric, so a circle is represented by its *supported role* (`circle_id` resolves to a `role_` id). | Supported role |
| Accountability | An ongoing activity of a role. Embedded as a property of Role — not a standalone resource. | — |
| Domain | An area of control held by a role. Embedded in Role. | — |
| Policy | A governance rule on a role's interior. | — |
| Tension | An issue or gap sensed by a role; the required seed of a proposal (`tension_id`). | — |
| Proposal | A governance change request carrying `changes`; the only path to alter governance structure. States: `draft → proposed_outside_meeting / escalated → accepted`. | Governance proposal |
| Change | A single governance command inside a proposal (e.g. `CreateRole`, `UpdateAccountability`). | ProposalChange, governance command |
| Response | A circle member's reaction to a circulating proposal: `no_objection` or `bring_to_meeting`. Aggregated anonymously. | Vote (the `ProposalVote` schema), proposal response |
| Actor | A person or AI agent that can fill roles. Identified by `per_` (person) or `agt_` (agent) prefix. | — |
| Practitioner | A human actor whose governance work the CLI serves; typically operates via an AI agent. | Person, circle member |
| AI Agent | An `agt_` actor; the CLI's usual direct operator. Agent/Skill endpoints require the `ai_integration` feature flag. | Agent |
| Assignment | The mapping of an actor to a role. | — |
| Context | A hypermedia document with everything needed to fill a role. | — |
| Spec | The published Glassfrog API v5 OpenAPI specification (`spec.yaml`); the authoritative contract for every CLI action. | spec.yaml |

---

## Actors

**Practitioner / circle member** — A human who holds roles in the organization. The person the CLI ultimately serves: they want to see their roles and the governance around them, raise tensions, and have proposals submitted on their behalf. They usually do not operate the CLI by hand.

**AI agent** — The CLI's direct operator (an `agt_` actor, e.g. Claude). It drives commands to read governance state and submit proposals, translating a practitioner's intent into spec-correct API calls. Output is shaped for this actor to parse reliably.

**Glassfrog API** *(system actor)* — The remote system of record. It owns all governance state, enforces permissions per the caller's membership, validates every change, and is the single source of truth the CLI never second-guesses.

---

## Scope

### In Scope
- **Reading the governance structure**: roles, circles, accountabilities, domains, policies, projects across the organization tree.
- **Personal / self-service reads**: `me`, my roles, my actions, my projects — the "what's mine" surface.
- **Actor governance reads**: resolving which actor fills a role (whom to contact about it) and an actor's governance footprint — the roles, accountabilities, domains, and purposes they hold. Reading actors through the governance lens, not actor administration.
- **Governance proposals (the write path)**: creating a proposal from a tension, viewing proposals and their `changes`/response summary, advancing a draft into circulation (`propose`), withdrawing, and recording a response (`no_objection` / `bring_to_meeting`).
- **Capturing tensions** as the entry point to a proposal.

### Out of Scope
- **Actor administration** — inviting, updating, or deleting people, and managing assignments. Administrative, not part of the practitioner-facing surface.
- **Multi-organization operation** — an API key is scoped to a single org + person; the CLI will not try to span organizations.
- **Holacracy coaching / governance advice** — interpreting tensions, advising on governance practice, or teaching Holacracy. The CLI is a faithful API surface, not a facilitator. (Reflects VISION Exclusion 1.)
- **Web-UI feature parity** — mirroring the rich visual or interactive features of the Glassfrog web app. The CLI's value is command-line and agent access, not recreating the UI. (Reflects VISION Exclusion 3.)

### Deferred
- **AI-agent-specific resources** (the `/agents` and `/skills` endpoints) — gated behind the `ai_integration` feature flag.
  *Condition to revisit*: when the target organization has `ai_integration` enabled and agent/skill management is needed.
- **Standalone operational writes** — direct create/update of actions, metrics, checklist items, notes, and strategy as day-to-day task management (beyond the tension capture that feeds a proposal).
  *Condition to revisit*: once the read + propose core is solid and there is demand to manage operational work items from the CLI.

---

## Constraints

**Spec is the contract**: Every CLI action must conform to the Glassfrog API v5 specification. The spec defines the available endpoints, fields, and behaviors; the CLI implements them faithfully and treats any divergence as a defect.

**Single org + person per key**: Authentication is an `X-Auth-Token` API key scoped to one organization and one person, with permissions following that person's membership. The CLI operates within that single identity and cannot exceed the caller's permissions.

**Governance is proposal-gated**: The API does not allow direct mutation of governance structure (roles, accountabilities, domains, policies); those change only through an accepted proposal. The CLI works within this — it cannot offer a direct "edit role" that bypasses a proposal.

**Optimistic concurrency**: Mutable resources expose `ETag` headers and accept `If-Match` for optimistic locking; omitting it is last-write-wins. The CLI must work within this concurrency model when writing.

**Premium-gated async proposals**: Creating and circulating proposals out of meeting requires the async-proposals (Premium) capability and returns 403 when not enabled. Availability of the proposal write path depends on the organization's plan.

**Rate limits**: The API enforces a per-organization rolling 1-hour rate limit that varies by plan. The CLI must operate within these limits.
