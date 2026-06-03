# Explore Log

Session summaries from `/prelude:explore`. Each entry records what was explored, which problems and solutions looked promising or were set aside, and why. Newest sessions first.

Consumed by explore (previous session awareness) and capture (provenance detection). Owned by explore — no other skill writes to this file.

---

## Session — 2026-06-03

Decomposed the CLI's problem space into a two-tier issue tree — a shared "client foundation" of cross-cutting concerns, and an "endpoint commands" tier (the endgoal) sliced by resource/use-case. Twelve problems surfaced, all fitting VISION/PROJECT/CONSTITUTION; four were promoted from MECE-gap flags. All twelve looked ready to capture.

### Explored

**Tier 1 — Client foundation**
- **Authentication** — how the CLI obtains/uses the `X-Auth-Token` for one org + person. Grounds in PROJECT constraint "single org + person per key". Well-defined.
- **Configuration** — where the token + org/base-URL come from (env/file/flag). Promoted from the auth↔config overlap flag; developer chose to split it out.
- **Output Rendering** — how output is structured so an AI agent and a human can both consume it. The voiced "JSON + human-readable" was translated to this problem and retained as a candidate note (dual output), not committed work. Maps to VISION principle 3 and CONSTITUTION II.
- **Error Handling** — surface RFC 9457 failures legibly with cause + next step, never silent. Grounds in CONSTITUTION II + III.
- **Feature-Gating** — handle `403` for `ai_integration`/Premium ("not on your plan") as a distinct shape from normal errors. Promoted gap.
- **Pagination** — traverse large result sets / the org tree without silent truncation. Grounds in CONSTITUTION VI (Size-Aware).
- **Rate Limiting** — honor the per-org hourly limit and back off on `429`. Grounds in CONSTITUTION X.
- **Optimistic Concurrency** — use `If-Match`/`ETag` on writes, no last-write-wins clobber. Promoted gap; CONSTITUTION X's conflict-table resolution made it load-bearing, distinct from rate-limiting.

**Tier 2 — Endpoint commands** (sliced by resource/use-case; in-scope per PROJECT)
- **Governance Reads** — roles, circles, accountabilities, domains, policies, projects. PROJECT In Scope.
- **Self-Service Reads** — `me`, my roles/actions/projects. PROJECT In Scope.
- **Tension Capture** — record a tension; the entry point to a proposal. PROJECT In Scope.
- **Proposal Write-Flow** — the multi-step create → propose → respond → accepted governance write path. PROJECT In Scope; qualitatively different from a single GET/PATCH, which is why it's its own problem rather than "just an endpoint".

### Suggested for Capture
- **Authentication** — ready; clear problem + actor, grounded in PROJECT.
- **Configuration** — ready; bounded, adjacent to auth.
- **Output Rendering** — ready; carries the dual-output candidate note.
- **Error Handling** — ready; grounded in two principles.
- **Feature-Gating** — ready; distinct response shape.
- **Pagination** — ready; grounded in CONSTITUTION VI.
- **Rate Limiting** — ready; grounded in CONSTITUTION X.
- **Optimistic Concurrency** — ready; load-bearing per the conflict table.
- **Governance Reads** — ready; in-scope deliverable.
- **Self-Service Reads** — ready; in-scope deliverable.
- **Tension Capture** — ready; in-scope, feeds proposals.
- **Proposal Write-Flow** — ready; in-scope, multi-step.

### Tensions
- **Intentional overlaps** the developer accepted (splitting judged clarifying, not confusing): feature-gating ↔ error-handling; optimistic-concurrency ↔ rate-limiting; configuration ↔ authentication.
- **Cross-link**: the proposal write-flow is itself Premium feature-gated (async proposals → `403`), so Feature-Gating and Proposal Write-Flow touch directly.
- **Scope guard**: the endpoint tier must stay within PROJECT's In Scope — actor administration and multi-org are Out; AI-agent/skill endpoints and standalone operational writes are Deferred. Wanting one of those is a PROJECT scope change, not a new endpoint problem.
