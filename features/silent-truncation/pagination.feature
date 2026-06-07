# Source: 016-pagination — Scenario: multiple pages assembled into a complete set

Feature: Silent Truncation — Pagination
  Large result sets and the org tree get cut off without the caller knowing.
  Pagination is the walker that turns the single-response API-client seam into a
  complete, multi-page result set: it follows the API's cursor until the last
  page, and when a walk is cut short it returns the records it did gather,
  clearly flagged incomplete with the reason — so a partial list is never
  mistaken for the whole.
  (affects: AI agent, Practitioner)

  Rule: One walker returns the complete set in API order
    # In order to retrieve a list that spans more results than one API response carries, without re-implementing cursor walking in every command,
    # as an AI agent building a list command,
    # I want one walker I can hand a list request to and get back the complete set in API order.

    # Source: 016-pagination — Scenario: a single page completes the set
    @wip
    Scenario: A single page completes the set
      Given a list endpoint whose first response reported no further pages
      When a list command walks the endpoint
      Then the walker will return a complete result carrying that page's records in API order
      And it will issue no further request

    # Source: 016-pagination — Scenario: multiple pages assembled into a complete set
    @wip
    Scenario: Multiple pages are assembled into a complete set
      Given a list endpoint with three pages of results
      When a list command walks the endpoint
      Then the walker will request each next page using the prior response's cursor
      And it will return a complete result carrying all records from the three pages in API order

    # Source: 016-pagination — Scenario: a non-paginated endpoint returns in one response
    @wip
    Scenario: A non-paginated endpoint returns in one response
      Given a list endpoint whose response carried no pagination block
      When a list command walks the endpoint
      Then the walker will treat the single response as the complete result
      And it will issue no cursor request

    # Source: 016-pagination — Scenario: an empty result set is a complete answer
    @wip
    Scenario: An empty result set is a complete answer
      Given a list endpoint whose first response carried no records and reported no further pages
      When a list command walks the endpoint
      Then the walker will return a complete result carrying no records

    # Source: 016-pagination — Scenario: the caller's other query parameters are preserved across pages
    @wip
    Scenario: The caller's query parameters are preserved across pages
      Given a list request carrying a search filter and an include set
      When the walker requests each subsequent page
      Then every page request will preserve the caller's search filter and include set
      And it will set only the page-size and cursor parameters

    # Source: 016-pagination — Scenario: records are never reordered or de-duplicated
    @validation @wip
    Scenario: Records are returned in API order without reordering or de-duplication
      Given a walk that gathered records across multiple pages
      When the complete result is inspected
      Then the records will appear in the order the API returned them across pages
      And no record will be dropped, reordered, or merged

    # Source: 016-pagination — Scenario: the walker re-resolves nothing
    @validation @wip
    Scenario: The walker re-resolves nothing
      Given a list request and the request seam are the walker's only inputs
      When a list command walks the endpoint
      Then the walker will use only the seam's transport, identity, and base URL
      And it will read no flag, environment variable, or credentials file itself

  Rule: A cut-short walk returns the partial set, flagged incomplete
    # In order to still get the records I could retrieve when a large read is cut short by a rate limit or a transport failure,
    # as an operator paging a big result set,
    # I want the partial set I gathered, clearly flagged as incomplete with the reason — not an all-or-nothing failure.

    # Source: 016-pagination — Scenario: a mid-walk page failure yields a partial set flagged incomplete
    @wip
    Scenario: A mid-walk page failure yields a partial set flagged incomplete
      Given a walk that had already gathered two pages
      And the next page request fails with a 429 rate-limit response
      When the walker continues the walk
      Then it will stop and return the two pages already gathered, flagged incomplete
      And the result will carry the failure that stopped it
      And the partial set will not be presented as complete

    # Source: 016-pagination — Scenario: a first-page failure yields an empty set flagged incomplete
    @wip
    Scenario: A first-page failure yields an empty set flagged incomplete
      Given a list endpoint whose first page request fails
      When a list command walks the endpoint
      Then the walker will return an empty result flagged incomplete carrying the failure
      And it will not report success

    # Source: 016-pagination — Scenario: has_next_page true but no usable cursor does not loop
    @wip
    Scenario: A page promising more results without a usable cursor does not loop
      Given a page that reported a further page but carried no usable cursor
      When the walker inspects the page
      Then it will not re-issue a request with an empty cursor and will not loop
      And it will return the records gathered so far flagged incomplete, naming the malformed-paging boundary

    # Source: 016-pagination — Scenario: a mid-walk page failure yields a partial set flagged incomplete
    # [proposed by skill — from plan Cross-cutting: a cancelled request context propagates as the stop cause]
    @wip
    Scenario: A cancelled request context stops the walk with the partial set
      Given a walk in progress that had gathered one page
      And the request context is cancelled before the next page
      When the walker continues the walk
      Then it will stop and return the records gathered so far, flagged incomplete
      And the result will carry the cancellation as the failure that stopped it

  Rule: A partial set is never presented as complete
    # In order to trust that a printed list is the whole list,
    # as a practitioner reading my governance data,
    # I want the tool to never present a partial page as if it were complete.

    # Source: 016-pagination — Scenario: no partial set is ever indistinguishable from a complete one
    @validation @wip
    Scenario: A partial result is never indistinguishable from a complete one
      Given a walk that stopped before the API reported its last page
      When the result is inspected
      Then it will be flagged incomplete and carry a stopping cause
      And it will not be readable as a complete result
