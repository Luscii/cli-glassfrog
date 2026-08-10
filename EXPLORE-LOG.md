# Explore Log

Session summaries from `/prelude:explore`. Each entry records what was explored, which problems and solutions looked promising or were set aside, and why. Newest sessions first.

Consumed by explore (previous session awareness) and capture (provenance detection). Owned by explore — no other skill writes to this file.

---

## Session — 2026-08-09

Evaluated an externally-written explore brief carrying five problem-shaped findings about the read surface's discovery-phase cost, measured empirically against the live org (27 reads, roughly 17 without signal). All five survived as real problems — four capture-ready as written, one reframed — with the brief's repo-facing claims verified against the code and the vendored spec before judging.

### Explored
- **Truncated Ranked Search** — `search` claims to walk "every page to completion by default" (verified at internal/cli/search.go:303) while the measured walk caps at 100 results, varies with page size, and signals nothing. Reads as a CONSTITUTION VI (Size-Aware) violation — arguably III as well ("a failure condition reported as success") — not merely a gap. Overlaps the tree's Silent Truncation; kept distinct (the contract actively asserts completeness) with a related-to, following the Oversized Content Unsignalled precedent. Narrow and reproducible enough to be a direct /score:specify candidate once captured.
- **Unenumerable Governance Corpus** — no read path enumerates the organization's policies; verified against the vendored spec (only `/policies/{id}` and role-scoped lists exist), so the gap is in the API itself. Solution space is constrained by PROJECT's "Bounded by the API surface" and CONSTITUTION I — the voiced org-scoped-list candidate is API-side and retained with that flag. Nuance: domains are closeable in principle by a full `GET /roles` walk (roles embed domains inline); the problem is sharpest for policies, where org-level records (`role_id: null`) hang off no role.
- **Vocabulary-Blind Search** — search matches literal text server-side (the CLI forwards the query verbatim), so ordinary synonyms return zero hits from a record that governs the topic — indistinguishable from "no governance exists", a confidently wrong verdict in exactly the constraint-discovery path. No constitutional violation (the CLI faithfully reports the API's correct zero); the harm traces to VISION principle 3's spirit. Candidates (near-misses, term index) are API-bounded — the same tension as Unenumerable Governance Corpus. Alternative placement noted: sub-problem of Undiscoverable Governance.
- **Indistinguishable Empty Results** — `policies <role-id>` returns "No policies." with exit 0 for every role when all governing policies are org-level and unreachable via any role-scoped read (13 zero-signal probes in the measured session). Structural cause verified against the spec. Weakest principle trace of the five: an empty list truthfully reports the API response (VIII-compliant, not an error), so III does not bite — assessed as a gap, not a violation. Largely a consequence of Unenumerable Governance Corpus in this record, though the ambiguity itself is generic (domains showed the same shape).
- **No Triage-Grade Output** — the brief's claim was half-true as measured: search's compact template is the triage rung ([type] id title rank, verified), while policy's compact drops the body. The durable problem is that `compact` has no cross-command contract — "one-line summary" in one command, "body deleted" in another — so an agent must know per command what compact discards. Traces to VISION principle 3 ("without guessing"). Reframed onto that contract inconsistency for capture. Cross-branch overlaps: Payloads Outsize the Work (the skill-side cost, with its Templated Read Projections candidate) and FEATURE-MODEL's Output Formatting capabilities (user templates already let a caller project any shape).

### Suggested for Capture
- **Truncated Ranked Search** — ready; verified in-repo evidence, likely constitutional violation, unambiguous correct behavior.
- **Unenumerable Governance Corpus** — ready; carries the API-boundedness tension explicitly.
- **Vocabulary-Blind Search** — ready; real actor harm, with the constrained solution space flagged.
- **Indistinguishable Empty Results** — ready as a gap, with no principle trace forced; its candidate travels with it.
- **No Triage-Grade Output** — ready after the reframe onto compact's inconsistent cross-command contract, accepted this session.

### Tensions
- **API-boundedness**: the candidates on Unenumerable Governance Corpus and Vocabulary-Blind Search need API support the spec does not define; the CLI must not invent endpoints (CONSTITUTION I, PROJECT "Bounded by the API surface"). The problems are recordable regardless of where the fix lives.
- **Rejected group tier**: the brief's proposed fourth tier ("reads that return a clean, plausible, incomplete answer with exit 0") splits constitutionally rather than unifying (Truncated Ranked Search lands on VI; the others carry no violation) — kept as prose in each problem's description, not as tree structure.
- **Placement**: all five under Client Foundation per the brief; Vocabulary-Blind Search could alternatively nest under Undiscoverable Governance — capture decides.

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
