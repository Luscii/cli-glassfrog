# Explore Log

Session summaries from `/prelude:explore`. Each entry records what was explored, which problems and solutions looked promising or were set aside, and why. Newest sessions first.

Consumed by explore (previous session awareness) and capture (provenance detection). Owned by explore — no other skill writes to this file.

---

## Session — 2026-08-10

Explored a measurement brief on the traversal cost of the `plugin/` operator paths, produced by running the constraint-discovery path end-to-end against a live org (27 reads, ~70k tokens, ~10 min for an answer whose load-bearing content came from ~10 reads). Five problems surfaced, all under the existing `* Unequipped Agent Operators` tier; every claim was verified against the surface files on `origin/main`, and all five fit VISION/PROJECT/CONSTITUTION. Two problems already in the tree were sharpened rather than re-captured.

### Explored

**The five new problems**
- **Traversal Runs in Lockstep** — the operator paths issue independent reads one after another, so a traversal's wall-clock is the sum of its reads rather than the depth of its dependency chain. Verified: the constraint-discovery workflow is a numbered sequential pipeline and nothing in the skill or agent marks any read as independent. Distinct from every existing sibling in the tier — those govern what enters the context; this one costs no tokens at all and is the part the practitioner feels. The candidate (naming which composed reads are independent) is knowledge, not capability, so it sits inside PROJECT's "knowledge + guardrails, never capability" constraint. Flagged for the fix, not for capture: a wide concurrent fan-out should state its relationship to CONSTITUTION X (rate limits) rather than assume burst shape is immaterial.
- **Void Steps Traversed Anyway** — a prescribed traversal step can be structurally incapable of yielding anything in a given record, and the path cannot notice or carry that forward. Verified: step 4 prescribes `policies <role-id>` unconditionally per role in view, with no early-exit and no record-shape awareness; the measured session got `No policies.` on thirteen consecutive roles because every policy in that org is org-level. Two flags carried to capture: whether it widens *Nothing Learned Is Kept* or stands beside it (the brief's argument for separate holds — that problem re-derives a *value*, this one executes a *step whose yield is structurally zero*, and a cache does not supply the early-exit), and a VISION Exclusion 4 conflict on the cross-session half of its candidate.
- **Clarification Gates Searchability, Not Scope** — the clarify step only asks whether the action is specific enough to search for, so dimensions that would prune most of the traversal go unasked and surface only as a closing note after the superset has been traversed. Verified: the clarify branch is scoped in the skill to an action "too vaguely to search for the governance that would constrain it" — a searchability gate with no notion of a pruning dimension. Fix placement is right by the surface's own design: interaction lives in the skill, the agent stays non-interactive.
- **Completeness Mandated Before Relevance** — the paths require paging a result set to completion before narrowing, so maximum payload is pulled in order to discard most of it, and the mandate is unsatisfiable on the entry-point read that cannot enumerate at all. Verified: the mandate appears in near-identical words across all six agents and four skills, so it is surface-wide rather than constraint-discovery-specific. The constitutional trace the brief asked for resolves the tension: CONSTITUTION VI has two arms — page through them **or clearly signal the boundary** — and the surface encodes only the first, which makes this a gap in the surface rather than a principle to weaken.
- **Traversal Coverage Unreported** — the returned picture says whether it was narrowed but never how much it covered or what it cost, so a shallow answer reads identically to an exhaustive one. Verified: the navigator's output contract specifies notes for narrowing, failure, and uncertainty, and nothing for coverage or cost; the same notes skeleton exists in all six agents. Grounds in CONSTITUTION VI's second arm and is load-bearing for the resolution of *Completeness Mandated Before Relevance*. The one fix of the five that lives in the agent's output contract rather than the skill.

**Already in the tree — sharpened, not re-captured**
- **Payloads Outsize the Work** — the brief's question was whether the retained *Templated Read Projections* candidate covers "don't hydrate a triage read at all". Reading the feature model: mostly yes — a projection on the `search` leaf keeping only id and title *is* the hit list, and projections replace `-o compact` wholesale, so the inverse trap the same session hit (compact deleting a policy's body) never bites. What the existing candidate does not settle is one template *per leaf* versus *per use* — a triage pass and a hydration pass over the same leaf want different projections. Recorded as an enrichment note on the existing problem, not a new problem.
- **Nothing Learned Is Kept** — whether it widens to cover record-shape facts (as distinct from identifier resolutions) is the same capture decision carried by *Void Steps Traversed Anyway*. Nothing further separable from that decision surfaced.

### Suggested for Capture
- **Traversal Runs in Lockstep** — ready; problem verified against the surface, actors named (Practitioner, AI agent), candidate note retained and inside the operating-layer constraint.
- **Void Steps Traversed Anyway** — ready; problem verified on thirteen consecutive empty reads. Carries two open flags for the capture conversation: widen-versus-separate against *Nothing Learned Is Kept*, and the persistence conflict on its candidate note.
- **Clarification Gates Searchability, Not Scope** — ready; problem verified in the skill's own scoping language, consequence observed in one round-trip. Carries a guard on its candidate: the pruning dimensions must derive from the record, since a canned scoping checklist would drift toward VISION Exclusion 1.
- **Completeness Mandated Before Relevance** — ready; mandate verified surface-wide and traced to CONSTITUTION VI. Carries the cross-brief dependency on the CLI-side half.
- **Traversal Coverage Unreported** — ready; absence verified in the output contract, grounded in CONSTITUTION VI's second arm. Carries a design note preferring proportion-dropped over raw read counts, since a read count means nothing to a practitioner.
- **Payloads Outsize the Work** — enrichment only, not a new problem: add the per-leaf-versus-per-use question to the existing candidate note.

### Tensions
- **Early-exit pulls against the completeness principle.** *Void Steps Traversed Anyway* wants a path that stops after several empties; *Completeness Mandated Before Relevance* defends a principle in which narrowing is a choice over the complete set. The sixth role could genuinely hold a policy. The two fixes need designing together rather than independently.
- **Record-shape persistence versus VISION Exclusion 4.** A record-shape note that persists across sessions is a local store of org-derived governance data, colliding with Exclusion 4 (no data store or cache) and PROJECT's "Live, stateless client" constraint. The session-local early-exit half is clean. The project has navigated this edge before — *Identifier Mapping from Proposal History* is deliberately "bootstrapped, not learned" — so the cross-session half wants that same treatment or a deliberate exception. Flagged, not resolved.
- **Cross-brief collision points — since resolved.** This brief is one of two from the same session; problems 2 and 4 are deliberately stated from both sides in both. The companion capture landed on main while this one was being written, so the reconciliation is settled rather than pending: both CLI-side halves became dedicated Client Foundation problems — *Indistinguishable Empty Results* for the empty read that cannot say "none reachable", and *Truncated Ranked Search* for the walk that claims completion and caps. The prediction here that the search-cap finding would attach to the existing *Silent Truncation* rather than become its own problem was wrong; the companion session kept it distinct on the grounds that the contract actively asserts completeness, and linked the two. Both halves of each pair are kept, linked by `related-to`, because the fixes live on different surfaces — a CLI that cannot signal is a separate matter from a path that never notices.
- **Fix placement follows the surface's own arrangement.** `constraint-navigator.md` explicitly defers the workflow to the skill and keeps no divergent copy, so the four workflow-level problems land in `SKILL.md` and the agent inherits them; only *Traversal Coverage Unreported* is an output-contract change that lives in the agent. Verified against both files. A fix that edits both has already broken the arrangement.
- **Surface-wide reach asserted for three of five.** The measured session exercised one path, but the completeness mandate and the notes skeleton were confirmed present across all six agents and four skills, so *Traversal Runs in Lockstep*, *Completeness Mandated Before Relevance*, and *Traversal Coverage Unreported* are surface-wide rather than constraint-discovery-specific.
- **None is ready for specification.** Each involves a judgment about how much autonomy the traversal gets, so each wants the capture conversation before any pipeline step.

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
