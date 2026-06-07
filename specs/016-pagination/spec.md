# Specification: Pagination

**Feature**: 016-pagination
**Role**: Definer
**Tier**: 1 (zero setup)

---

## System Overview

Pagination is the **"page through them"** half of CONSTITUTION VI (*Size-Aware by Design* — the CLI must never silently truncate; it must page through results or clearly signal the boundary). It is a reusable **paging walker**: given a list request, it walks the Glassfrog API's cursor pagination and assembles the complete result set across every page, so list commands get the whole set without each re-implementing paging. It is the counterpart to the *signal-the-boundary* half that the self-service reads (012, 013, 014) use today — they print the first page and warn that more exist; the walker lets a command instead retrieve the rest.

Like its siblings API Error Extraction (015) and Rate-Limit Handling (017), Pagination is a **mechanism, not a command**. It sits on top of **Request Execution (010)**, calling that seam once per page and reading each decoded response's paging block to decide whether to continue. It does not re-implement transport, identity, base-URL resolution, exit codes, or error classification — those are owned upstream (010, 007, 008/009, 004) and by the sibling 015. Its own job is small and exact: issue page requests through the seam, concatenate the records they return, and stop — either complete, or partial and clearly flagged so a caller can never mistake it for the whole.

---

## Behavioral Accord

### Walking to completion

- When handed a list request (method, path, the caller's query parameters, and a decode target for the enveloped list response), the system issues the first request through Request Execution with a page size it controls, then reads the response's `meta.pagination` block.
- When a response reports `has_next_page` true and carries a usable `next_cursor`, the system re-issues the same request with the `cursor` query parameter set to that `next_cursor` — preserving the caller's other query parameters and the page size — and appends the new page's `data` to the records gathered so far, repeating until a page reports `has_next_page` false.
- When a response carries no `meta.pagination` block (an endpoint that does not paginate, such as the org role tree), the system treats the single response as the complete set and stops without issuing a cursor request.
- When the walk reaches its end (a page reports `has_next_page` false, or pagination is absent), the system reports a **complete** result carrying every record concatenated in the order the API returned them.

### Page size

- When issuing page requests, the system uses a page size that defaults to the API maximum to minimize the number of round-trips against the rate limit; an operator may override it (for now only via a command-line flag).
- When an out-of-range page size is sent, the system surfaces the API's rejection rather than assuming the server silently clamped the value — page size is the caller's input, and the API owns its bounds.

### Incomplete walks — never silently truncate

- When a page request fails mid-walk (a transport error, or a non-2xx response such as a `429` rate-limit), the system stops walking and reports the records gathered so far **flagged as incomplete**, carrying the failure that stopped it — so the caller keeps the partial set it did retrieve but can never read it as the whole.
- When a page reports `has_next_page` true but the cursor does not advance — `next_cursor` is absent, blank, or identical to the cursor just used — the system does not re-issue and does not loop; it stops and reports the records gathered so far flagged as incomplete, naming the malformed-paging boundary. (A repeated cursor is treated the same as a blank one: re-sending it would fetch the same page forever.)
- The system never reports a partial result as complete: every outcome states whether the set is complete, and every incomplete outcome carries the reason the walk stopped.

### Surfacing the outcome

- The system reports its outcome — a complete set, or a partial set flagged incomplete with its stopping cause — to the calling command; it does not decide the user-facing message or the process exit code.

---

## User Scenarios

**In order to** retrieve a list that spans more results than one API response carries, without re-implementing cursor walking in every command,
**as an** AI agent building a list command,
**I want** one walker I can hand a list request to and get back the complete set in API order.

**In order to** still get the records I *could* retrieve when a large read is cut short by a rate limit or a transport failure,
**as an** operator paging a big result set,
**I want** the partial set I gathered, clearly flagged as incomplete with the reason — not an all-or-nothing failure.

**In order to** trust that a printed list is the whole list,
**as a** practitioner reading my governance data,
**I want** the tool to never present a partial page as if it were complete.

---

## Non-Behaviors

- The system must not report a partial result as if it were complete. **Why**: that is exactly the silent truncation CONSTITUTION VI forbids — a partial governance picture read as the whole gives a false view. Every incomplete outcome is flagged with its stopping cause.
- The system must not re-implement transport, identity, or base-URL resolution. **Why**: Request Execution (010), Request Authentication (007), and Connection Context Assembly (009) own those; the walker composes 010 once per page and never duplicates the seam.
- The system must not interpret or classify a non-2xx page response into a typed API error. **Why**: API Error Extraction (015) owns turning a status and body into a meaningful error; the walker carries the failure that stopped it and lets 015 / the command interpret it.
- The system must not back off, sleep, or retry on `429`. **Why**: Rate-Limit Handling (017) owns backoff, and it lives inside the per-page send through 010; the walker only stops and flags the partial set when a page ultimately fails. Owning retry here would fork that contract.
- The system must not reorder, de-duplicate, or transform the records it gathers. **Why**: the CLI is a faithful surface (CONSTITUTION I) — it concatenates each page's `data` in the API's order; reordering or dedup would invent a view the API never returned.
- The system must not drop or rewrite the caller's other query parameters when re-issuing a page. **Why**: a `q` filter or `include` set defines the result being paged; only `per_page` and `cursor` are the walker's to set, and losing the rest would silently change what is being walked.
- The system must not fabricate a cursor or attempt to paginate an endpoint that returns no paging block. **Why**: the org role tree and other non-paginated endpoints return their full set in one response; inventing a cursor would send a meaningless request.
- The system must not decide the process exit code or the user-facing incompleteness message. **Why**: Exit-Code Convention (004) and the consuming command own classification and rendering; the walker reports the outcome (and whether it is complete) and lets the command signal per CONSTITUTION VI.
- The system must not prompt interactively. **Why**: the operator is usually a non-interactive AI agent; the page size arrives from a flag and a stopping condition is surfaced as a typed outcome, never solicited.

---

## Integration Boundaries

- **Request Execution (010 — upstream dependency)**: the walker composes 010, calling it once per page; 010 sends one authenticated request and returns the single decoded response (or a typed failure). The walker reads `meta.pagination` from each decoded response to decide whether to continue. *Note*: the v5 paging state lives in the response **body** at `meta.pagination`, not in a response header — 010's passing mention of a `Link` header is superseded by the spec's body-based cursor scheme, which 012 already reads (`meta.pagination.has_next_page`).
- **List commands (012 My Roles, 013 My Actions, 014 My Projects, and the future Governance Reads — downstream consumers)**: hand the walker a list request and receive the complete set, or a partial set flagged incomplete. Each command renders the records and, on an incomplete outcome, signals the boundary (CONSTITUTION VI) and chooses its exit code (004). Today these commands single-page and signal incompleteness themselves; adopting the walker is each command's own change, not this capability's.
- **Rate-Limit Handling (017 — sibling)**: when present, 017's backoff lives inside the per-page send through 010, so the walker sees either a resolved page or a final failure; the walker itself never sleeps or retries. A page that ultimately fails with `429` stops the walk and flags the partial set — the rate-limited large read this capability is meant to survive gracefully.
- **API Error Extraction (015 — sibling)**: interprets the non-2xx failure the walker surfaces as its stopping cause; the walker carries the raw failure, 015 types it.
- **Glassfrog API v5**: provides cursor pagination — `per_page` (1–500, default 100) and `cursor` query parameters, and a `meta.pagination` block (`per_page`, `has_next_page`, `next_cursor`) in each list response. `next_cursor` is present only when `has_next_page` is true; an over-max `per_page` returns `400`; some endpoints return no paging block.

---

## Driving Scenarios

### Happy path

**Scenario: a single page is the complete set**
Given a list request whose first response reports `has_next_page` false
When the walker runs
Then it reports a complete result carrying that page's records in API order
And issues no further request.

**Scenario: multiple pages assembled into a complete set**
Given a list endpoint with three pages of results
When the walker runs
Then it issues the first request, then re-issues with `cursor` set to each response's `next_cursor` until a page reports `has_next_page` false
And reports a complete result carrying all records from the three pages concatenated in API order.

**Scenario: a non-paginated endpoint returns in one response**
Given a list request to an endpoint that returns no `meta.pagination` block (the org role tree)
When the walker runs
Then it treats the single response as the complete set
And issues no cursor request.

### Error scenarios

**Scenario: a mid-walk page failure yields a partial set flagged incomplete**
Given a walk that has already gathered two pages
When the next page request returns a `429` rate-limit response (or a transport error)
Then the walker stops and reports the two pages it gathered, flagged as incomplete
And carries the failure that stopped it
And does not present the partial set as complete.

**Scenario: a first-page failure yields an empty set flagged incomplete**
Given a list request whose first page request fails (an out-of-range `per_page` returns `400`, or a transport error)
When the walker runs
Then it reports an empty set flagged as incomplete, carrying the failure
And does not report success.

### Edge cases

**Scenario: has_next_page true but a blank cursor does not loop**
Given a page that reports `has_next_page` true but carries an absent or blank `next_cursor`
When the walker inspects it
Then it does not re-issue with an empty cursor and does not loop
And stops, reporting the records gathered so far flagged as incomplete, naming the malformed-paging boundary.

**Scenario: has_next_page true but a repeated cursor does not loop**
Given a page that reports `has_next_page` true but carries a `next_cursor` identical to the cursor just used
When the walker inspects it
Then it does not re-issue the same cursor and does not loop
And stops, reporting the records gathered so far flagged as incomplete, naming the malformed-paging boundary.

**Scenario: an empty result set is a complete answer**
Given a list request whose first response carries `data: []` and `has_next_page` false
When the walker runs
Then it reports a complete result carrying no records.

**Scenario: the caller's other query parameters are preserved across pages**
Given a list request carrying a `q` filter and an `include` set
When the walker re-issues for the next page
Then each page request preserves the caller's `q` and `include`
And sets only `per_page` and `cursor`.

---

## Validation Scenarios

> These are held out from the implementing agent for independent verification.

**Scenario: records are never reordered or de-duplicated**
Given multiple pages whose `data` arrays are concatenated
When the complete result is inspected
Then the records appear in the same order the API returned them across pages
And no record is dropped, reordered, or merged.

**Scenario: no partial set is ever indistinguishable from a complete one**
Given any walk that stops before a page reports `has_next_page` false (a failure or a malformed cursor)
When the outcome is inspected
Then it is flagged incomplete and carries a stopping cause
And cannot be read as a complete result.

**Scenario: the walker re-resolves nothing**
Given the list request and the Request Execution seam are the only inputs
When the walk runs
Then the only transport, identity, and base URL used are the seam's
And the walker reads no flag, environment variable, or credentials file directly — the page size is supplied to it.

---

## Assumptions

- **Enveloped decode target** `[ASSUMED]`: the caller supplies a decode target shaped as the list envelope (`data` array plus `meta.pagination`), so the walker can read both the records and the paging state from each decoded response. The exact target shape is a planning detail; the behavior — read `data` and `meta.pagination` — is fixed.
- **Page-size default and override**: the walker defaults the page size to the API maximum (500) to minimize round-trips, and the operator may override it; for now the override is exposed only as a command-line flag (not env or config). The default value and the flag's name/shape are interface/planning details; the behavior (large default, caller-overridable, no silent clamp) is fixed.
- **Cursor parameter name** `[ASSUMED]`: the next page is requested by passing the prior response's `next_cursor` as the `cursor` query parameter, per the spec's defined `Cursor` parameter and the `Pagination` schema's "Pass as `?cursor=`" note. The spec's prose intro also mentions an `after` parameter; this is treated as a spec inconsistency resolved in favor of the defined `cursor` parameter, and is worth confirming against the live API.
- **Unbounded walk with a non-advancing-cursor guard**: the walk has no fixed maximum page count — it runs until the API reports `has_next_page` false — but a page claiming more results while the cursor fails to advance (a `next_cursor` that is absent, blank, **or identical to the one just used**) is treated as a malformed-paging boundary and stops the walk (flagged incomplete) rather than looping. The repeated-cursor case matters because an API that ignores an unrecognized `cursor` parameter (see the cursor-name assumption above) would otherwise return the same page — with the same non-blank cursor — indefinitely. A hard ceiling was considered and intentionally not imposed.
- **Partial-result outcome shape** `[ASSUMED]`: the names of the complete/partial outcome and the incompleteness flag are adjustable at planning without changing behavior; the behavior (records + complete flag + stopping cause) is fixed.
- **Composition with Rate-Limit Handling (017)**: when 017 exists, its backoff lives inside the per-page send through 010, so the walker sees either a resolved page or a final failure and never sleeps or retries itself. Documented so 016 and 017 do not both try to own retry.

---

## Ambiguity Warnings

_None remaining — the four behavioral forks were resolved during the defining conversation: (1) Pagination is a reusable walker mechanism, not a command; (2) a walk cut short returns the records gathered so far **flagged as incomplete** with its stopping cause, never an all-or-nothing failure and never a partial set passed off as complete; (3) the walk is unbounded but guards against a non-advancing cursor (fail loud, never loop), with no mandatory hard cap; (4) the page size defaults to the API maximum to conserve the rate budget and is overridable, for now only via a command-line flag. The remaining `[ASSUMED]` items are planning-time shape details and one spec-fidelity check (`cursor` vs `after`), not behavioral gaps._
