# Glassfrog CLI — Vision & Constraints

---

## Identity

Glassfrog CLI is a command-line tool that covers the Glassfrog API v5. It makes it easy to get information out of Glassfrog — roles, circles, accountabilities, and the rest of a Holacracy governance record — and to act on it by making proposals and adjusting things through the API. Glassfrog is a Holacracy governance platform; this CLI is a faithful, exhaustive surface over its published v5 API.

It exists so that working with Glassfrog does not require hand-writing HTTP calls or navigating the web UI for every task. Its distinctive pull is that it is built to be driven by AI agents: an agent such as Claude can read a practitioner's governance state and submit valid proposals on their behalf, with the CLI translating intent into spec-correct API actions.

---

## Audience

**Primary**: Holacracy practitioners and circle members — the people whose governance work the tool serves. In practice they rarely operate the CLI by hand; they work through an AI agent (such as Claude) that drives the CLI for them. They need to see their roles, circles, and the state of governance, and to make proposals or adjustments without learning the API themselves. The AI agent is the CLI's direct operator; the practitioner is who it ultimately serves.

**Secondary**: Developers and integrators who script directly against the CLI for automation or bulk work. [THIN — may need revisiting]

---

## Principles

### 1. Spec fidelity
The CLI must always perform actions exactly as defined in the published Glassfrog API v5 specification. The spec is the contract; the CLI's job is to honor it, not to reinterpret it.

*This means*: every command maps to behavior the v5 spec defines, and the CLI is kept aligned with the spec as the authoritative reference.
*This means we won't*: invent endpoints, parameters, or behaviors the spec doesn't define, or quietly diverge from how the spec says an action works.

### 2. Governance integrity
Changes to a Holacracy record go through proper governance — proposals — by default, rather than back-door mutations that bypass process. The tool respects how governance is meant to change. A narrow, deliberate exception exists for callers the platform already trusts to manage governance directly, but it is never the default and never silent.

*This means*: write operations are framed as proposals submitted through governance by default, so an agent acting on a practitioner's behalf changes things the way the organization expects; any direct governance management is reachable only through an explicit, clearly-named opt-in and the appropriate permission.
*This means we won't*: offer shortcuts that silently or by default bypass the governance process — a direct governance change always requires a deliberate, explicit choice, never something an agent reaches without asking.

### 3. Agent-legible output
Output is structured and predictable so an AI agent can reliably parse it and decide what to do next. The primary operator is a machine, and the interface is shaped for that.

*This means*: command output is machine-readable and consistent enough for an agent to act on without guessing.
*This means we won't*: rely on free-form, human-only formatting that an agent would have to scrape or interpret heuristically.

### 4. Transparent about actions
Every action maps clearly to a documented spec endpoint, so the user or agent always knows exactly what was done to which record. There are no hidden effects.

*This means*: a command's effect is traceable to the specific spec operation it performs and the record it touches.
*This means we won't*: bundle implicit side effects into commands or obscure which API action a command actually performs.

---

## Exclusions

### 1. A Holacracy coach
The CLI is a faithful API surface, not a facilitator. It will not advise on governance practice, interpret tensions, or teach Holacracy.

*Why it's tempting*: the data is all about governance, so it would feel natural to layer guidance on top — but that conflates a transport tool with a coaching role and pulls the project away from being a clean API surface.

### 2. Local governance logic
The CLI will not reimplement Glassfrog's rules or validation locally. The API, per the spec, is the source of truth for what is valid.

*Why it's tempting*: validating locally could give faster feedback, but it would duplicate logic that lives in Glassfrog and inevitably drift from it, violating spec fidelity.

### 3. A web UI replacement
The CLI will not try to mirror every rich visual or interactive feature of the Glassfrog web app.

*Why it's tempting*: once you cover the API it's easy to imagine reproducing the whole product — but the value here is command-line and agent access, not recreating the UI.

### 4. A data store or sync engine
The CLI will not maintain its own database, cache, or two-way sync of Glassfrog data. It is a live client.

*Why it's tempting*: caching could speed up reads and enable offline work, but a local store introduces staleness and a second source of truth, which conflicts with being a faithful live surface over the API.

---

## Constraints

**Spec conformance**: The CLI must conform to the Glassfrog API v5 specification (the published `spec.yaml`). The spec is the authoritative definition of every action the CLI performs; behavior that contradicts the spec is a defect, not a feature.

**Live, stateless client**: The CLI operates against the live Glassfrog API with no persistent local store of governance data. Every read reflects current state and every write goes to the live API, because the project explicitly excludes maintaining its own database, cache, or sync engine.

**Bounded by the API surface**: The CLI can only offer what the v5 API exposes. It cannot provide capabilities the API does not define, because it is a surface over that API rather than an independent system.

---

## Success Criteria

1. **Every in-scope endpoint in the v5 spec is reachable through the CLI.** Nothing within the CLI's defined scope is inaccessible from the command line. (What's in scope, out, and deferred — e.g. actor administration and multi-org are out — is defined in PROJECT.md.)

2. **An AI agent can complete a real task end-to-end** — read a practitioner's roles and submit a valid proposal — without the human needing any knowledge of the API.

3. **The CLI stays provably correct against the spec.** When the spec changes, the drift is caught, so the CLI's behavior can be shown to still match the published specification.
