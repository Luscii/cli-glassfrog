# Specification: Cross-Model Search

**Feature**: 041-cross-model-search
**Role**: Definer
**Tier**: 1 (zero setup)

---

## System Overview

Cross-Model Search is the discovery read for the **Undiscoverable Governance** problem: when an operator is working a tension, they often can't find which roles, policies, projects, or role-fillers are relevant without already knowing where to look. This capability adds a `search` command (`glassfrog search <query>` → `GET /search` → `search`) that runs a single relevance-ranked full-text query across every searchable resource type — roles, notes, projects, actions, skills, actors, policies, and domains — and returns a uniform, ranked result set. It is the entry point a practitioner or AI agent uses to go from "something about onboarding" to the specific records they then read directly.

The API answers from one endpoint with one uniform result shape: each `SearchResult` carries a `type`, an `id`, a `title`, an optional `excerpt`, a relevance `rank`, and an optional owning `role_id`. Because every result is self-describing in this way, the CLI renders them all the same and each result's `type` + `id` is the bridge the operator uses to drill into the matching read command (e.g. a `role` result's id into the role read). Cross-Model Search sits on the proven read chain rather than rebuilding it: it hands requests to **Request Execution (010)**, reads identity through **Request Authentication (007)**, walks pages through **Pagination (016)**, lets **Output Format Selection (020)** render the result, and maps outcomes through **Exit-Code Convention (004)** and **API Error Extraction (015)**.

---

## Behavioral Accord

### Invocation

- When the user runs the search command with a query, the system sends that query verbatim as the `query` parameter and produces the ranked results as a list result.
- When the user runs the search command with no query (the required term missing or empty), the system rejects the invocation as a usage error and calls no API.
- When the user passes an unknown flag or more than one positional query argument, the system rejects the invocation as a usage error and calls no API.
- When the query contains websearch operators (quoted phrases, `or`, `-` exclusions), the system forwards the whole string unmodified — it does not parse, rewrite, escape, or validate the query's internal syntax; the API owns websearch interpretation.

### Type scoping

- When the user supplies `--types` with one or more of the supported values (`role`, `note`, `project`, `action`, `skill`, `actor`, `policy`, `domain`), the system sends them as the comma-separated `types` query and the results are limited to those types.
- When the user supplies no `--types`, the system requests all types (the API default) and does not send the parameter.
- When the user supplies `--types` with a value outside the supported set, the system rejects the invocation as a usage error before issuing any request, naming the unsupported value and the supported set — it validates the closed enum locally rather than relying on the API to reject it.

### Output

- When a search succeeds, the system produces the result set as its result and lets Output Format Selection (020) render it in the effective format (`full` / `compact` / `json` / `yaml`); it neither fixes raw API JSON as its default nor defines its own format flag.
- When the system produces results, it preserves the API's relevance ordering exactly and never re-sorts or re-ranks them — the most relevant result the API returned first stays first.
- When a result carries each field, the system surfaces the result's `type`, `id`, `title`, `excerpt`, `rank`, and `role_id` as the API returned them, so the operator can read the match and drill into the matching read command via the result's `type` + `id`.
- When a result's `excerpt` or `role_id` is absent (null), the system renders it as absent and never substitutes invented text — a missing excerpt is shown as missing, not fabricated.
- When the search matches nothing, the system produces an empty list result and exits successfully — zero matches is a valid answer, not an error.

### Completeness

- When the results span more than one page, the system walks every page through **Pagination (016)** by default and produces the complete relevance-ordered set.
- When the user supplies the first-page opt-out flag, the system makes a single page request and, if more pages exist, produces the first page flagged incomplete with a clear "more exist" signal — so even the opted-out result is never silently truncated.
- When the walk cannot complete (a page fails mid-walk), the system produces the results gathered so far, flagged incomplete with the cause, so a partial set is never mistaken for the whole.

### Failure

- When no usable token is available, the system surfaces the authentication fail-safe's refusal and exits non-zero with a not-authenticated outcome, reusing the shared not-authenticated message and pointing the operator at how to store a credential.
- When the request cannot reach or complete at the wire (connection, DNS, TLS, timeout), the system surfaces the transport failure by name and exits non-zero with the network-unavailable outcome.
- When the API answers with a non-2xx response — including a malformed query the API rejects (typically `400`) — the system reports that the search failed, naming the HTTP status, and exits non-zero. The command adds no interpretation of its own; API Error Extraction (015) classifies the status and the command surfaces whichever outcome results.
- Whatever the failure, the message names both what went wrong and a concrete next step (Action Transparency), and never includes the token.

---

## User Scenarios

**In order to** find the roles, policies, and projects relevant to a tension without already knowing where they live,
**as an** AI agent operating the CLI on a practitioner's behalf,
**I want to** run one relevance-ranked query across every governance resource type and get a uniform ranked list.

**In order to** narrow a noisy search to just the kinds of records I care about,
**as an** AI agent assembling context for a decision,
**I want to** scope the query to specific resource types.

**In order to** go from a search hit to the full record,
**as an** AI agent navigating governance,
**I want to** each result to carry the type and id I need to drill into the matching read command.

**In order to** trust that I have seen everything that matched,
**as an** AI agent with a bounded context,
**I want to** the result set to be complete by default or plainly flagged incomplete.

---

## Non-Behaviors

- The system must not parse, rewrite, escape, or validate the internal syntax of the query string. **Why**: the API runs websearch full-text search; any client-side rewriting would change what the operator asked for and could silently alter matches. The CLI is a faithful surface — it forwards the query verbatim.
- The system must not re-rank, re-sort, de-duplicate, or filter the results it received. **Why**: relevance ranking is the API's job and the ordering is the answer; a client-side re-sort would present a different, unfaithful ranking and mislead the operator about what matched best.
- The system must not auto-fetch the underlying resource for each result to enrich it. **Why**: search returns a self-describing summary (type, id, title, excerpt, rank); fanning out a read per result would multiply API calls, hit rate limits, and blur the line between "search" and "read". The result's type + id is the bridge the operator uses to drill in deliberately.
- The system must not invent a `title` or `excerpt` when the API omits one, nor count or summarize omitted results. **Why**: fabricated content gives a false picture of governance (No Fabricated Data); a null excerpt is shown as absent.
- The system must not resolve the base URL or token, attach the `X-Auth-Token` header, decide the no-token fail-safe, type a non-2xx response, or choose its own exit codes. **Why**: Base URL Resolution (008), Credential Discovery (005), Request Authentication (007), API Error Extraction (015), and Exit-Code Convention (004) own those; a second path here would drift from their contracts.
- The system must not emit raw API JSON as a fixed default, nor define its own output-format flag. **Why**: Output Format Selection (020) resolves the format and dispatches to the renderers; a private flag here would fork that contract.
- The system must not write, mutate, or capture anything from a search hit. **Why**: search is a pure read; governance changes only through Proposals, and operational writes are out of this capability's scope.

---

## Integration Boundaries

- **Glassfrog API v5 (`GET /search`)**: the system runs the full-text query through this endpoint. Data flows inbound (ranked results in, nothing written). When the endpoint is unreachable or answers non-2xx, the command surfaces a transport or search failure and exits non-zero.
- **Request Execution (010)**: the seam the command hands each request to and reads the outcome from (success-with-body, non-2xx, transport error, decode error).
- **Request Authentication (007)**: supplies the authenticated transport and the no-token fail-safe whose refusal the command propagates.
- **Pagination (016)**: the walker the command uses to assemble the complete relevance-ordered set across pages (or a flagged-incomplete partial set when a page fails or the operator opts out).
- **Output Format Selection (020)**: resolves the effective `--output` format and dispatches the produced result to the matching renderer; the command produces result data, not presentation.
- **API Error Extraction (015) / Exit-Code Convention (004)**: classify a non-2xx response and map the command's outcome to a process exit code through the established conventions.
- **User / AI agent (stdout/stderr)**: the rendered result is written to stdout on success; failure messages are written to stderr.

---

## Driving Scenarios

### Happy path

**Scenario: Search across all resource types**
Given a valid token resolving to a member of an organization
When the user runs the search command with the query `onboarding`
Then the system sends `query=onboarding` with no type scope and produces a relevance-ordered list of matching results
And the command exits successfully

**Scenario: Scope a search to specific types**
Given a valid token and an organization with matching roles and projects
When the user runs the search command with the query `budget` and `--types role,project`
Then the system sends `query=budget` and `types=role,project` and the results contain only role and project hits
And the command exits successfully

**Scenario: Each result carries the bridge into a read command**
Given a valid token and a query that matches at least one role
When the user runs the search command with that query
Then each produced result carries its `type`, `id`, `title`, `excerpt`, and `rank`
And a role result also carries its `role_id`, so the operator can drill into the matching read command

**Scenario: A multi-word websearch query is forwarded verbatim**
Given a valid token
When the user runs the search command with the query `"strategy review" -archived`
Then the system forwards the whole query string unmodified as the `query` parameter
And the command exits successfully

### Error scenarios

**Scenario: No usable token**
Given no usable token is available to the CLI
When the user runs the search command with any query
Then the system surfaces the authentication fail-safe's refusal as a not-authenticated outcome
And the command exits non-zero, pointing the operator at how to store a credential
And no results are produced

**Scenario: A query the API rejects as malformed**
Given a valid token but a query the API cannot process
When the user runs the search command with that query
Then the system reports that the search failed and names the HTTP status (typically `400`)
And the command exits with the API-error code

### Edge cases

**Scenario: A search that matches nothing**
Given a valid token and a query that matches no resources
When the user runs the search command with that query
Then the system produces an empty list result
And the command exits successfully

**Scenario: An unsupported `--types` value is rejected without an API call**
Given a valid token
When the user runs the search command with `--types nonsense`
Then the system rejects the invocation as a usage error, naming the unsupported value and the supported set
And no API call is made

**Scenario: A missing query is a usage error**
Given a valid token
When the user runs the search command with no query argument
Then the system rejects the invocation as a usage error
And no API call is made

**Scenario: Results span more than one page (default walk to completion)**
Given a query whose matches span more than one page of the API response
When the user runs the search command with that query
Then the system walks every page through Pagination (016) and produces the complete relevance-ordered set
And the command exits successfully

**Scenario: First-page opt-out stops at one page and signals more exist**
Given a query whose matches span more than one page of the API response
When the user runs the search command with the first-page opt-out flag
Then the system makes a single page request and produces the first page of relevance-ordered results
And it surfaces a clear "more exist" incomplete signal
And the command exits successfully

---

## Validation Scenarios

> These are held out from the implementing agent for independent verification.

**Scenario: The query is forwarded byte-for-byte**
Given a search invoked with a query containing websearch operators
When the outbound request is inspected
Then the `query` parameter equals the operator's input exactly
And no client-side rewriting, escaping, or normalization was applied

**Scenario: The rendered order matches the API's relevance order**
Given a successful multi-result search
When the produced result order is compared against the API response order
Then they are identical
And no client-side re-sort or de-duplication occurred

**Scenario: Search default output carries no raw API envelope**
Given a successful search run under the default human format
When the output is inspected
Then it shows the reshaped result projection only
And it does not contain the raw `data` JSON envelope

**Scenario: Incompleteness is never silent**
Given a search where Pagination (016) could not assemble every page
When the result is inspected
Then an explicit incomplete signal with its cause is present
And the partial set cannot be read as the complete set

---

## Assumptions

- **Command spelling** (`glassfrog search <query>`, query as a required positional): assumed from the FEATURE-MODEL "Cross-Model Search" framing and the sibling read commands' shape. The exact command and flag surface — positional query vs. a `--query` flag, the `--types` flag name — syncs with the CLI command convention at interface time. `[ASSUMED]`
- **`--types` reject-unknown semantics**: assumed Cross-Model Search reuses the Identity Read (011) / Role Reads (025) / Organization Tree (026) opt-in, reject-unknown handling for closed-enum inputs, validating each value against the spec's eight-value type set locally before any call. (Informed by specs 011/025/026 and the DECISIONS "validate closed-enum inputs locally" stance.)
- **Completeness model** reuses the sibling list reads' model: walk by default, first-page opt-out that signals incompleteness, partial-on-mid-walk-failure flagged — required by CONSTITUTION VI (never silently truncate). The exact opt-out flag name is shared with the other paginated reads at interface time. (Informed by specs 016/026 and CONSTITUTION VI; the walk-by-default vs. first-page-default question was resolved in Clarifications — see below.)
- **Output rendering** is delegated to Output Format Selection (020); the built-in default format is `full` (020's default). The uniform per-type rendering of a `SearchResult` and whether the rendering surfaces a concrete drill-in command string per result is a presentation decision for the interface/render layer. (Informed by spec 020.)
- **Permission scoping**: the API returns only results the caller's membership permits and the CLI does not second-guess that; types the org cannot access (e.g. skills/actors behind `ai_integration`) simply return no hits rather than an error. (Informed by PROJECT.md actor notes and Constraint "single org + person per key".)

---

## Ambiguity Warnings

*None outstanding.* (The default page behavior for a relevance-ranked search — the one open question after specify — was resolved in the Clarifications below: `search` walks all pages by default with a first-page opt-out, staying symmetric with the other list reads.)

---

## Clarifications

### Session 2026-06-11

- **Default page behavior**: `search` walks all pages by default and offers a first-page opt-out that signals "more exist," matching every other paginated list read (subroles, org tree) rather than defaulting to a single page. The alternative — first-page-by-default with an opt-in `--all` — was considered for the relevance-ranked, bounded-context case but rejected in favor of cross-command symmetry and the strongest reading of CONSTITUTION VI (complete by default, never silently truncated). The Completeness accord and the driving scenarios (default walk-to-completion + first-page opt-out) already reflect this; no accord change was needed.
