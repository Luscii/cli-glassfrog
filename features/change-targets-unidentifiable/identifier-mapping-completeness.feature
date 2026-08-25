# Source: 080-identifier-mapping-harvest — Scenario: A walked history yields the whole mapping in one pass

Feature: Identifier Mapping Completeness
  The harvest's mapping is only trustworthy if its coverage is legible: a
  consumer that treats a partial mapping as exhaustive concludes an identifier
  does not exist when it merely was not walked to. So the derived document
  carries its own completeness verdict in every format — complete only when
  the walk covered the whole history with no filter and no first-page opt-out
  — and the boundary dimensions (narrowed, first-page, stopped) report
  independently, never collapsed into one value. The human render summarizes
  a whole-history walk and lists entries only on a narrowed one, keyed on
  narrowing rather than size so the same invocation always renders the same
  way. This file covers how the walk bounds the mapping; its sibling
  identifier-mapping-harvest.feature covers what an entry carries.
  (affects: Practitioner, AI agent)

  Rule: A partial mapping is never read as an exhaustive one
    # In order to avoid trusting a mapping that silently covers only part of
    # the history,
    # as an AI agent consuming the harvest,
    # I want to be told whether the walk was complete, so a narrowed mapping
    # is never read as an exhaustive one.

    # Source: 080-identifier-mapping-harvest — Scenario: A walked history yields the whole mapping in one pass
    @wip
    Scenario: A whole-history walk reports the mapping complete
      Given a complete connection context with a stored token
      And the proposal history spans two pages
      When an agent runs "glassfrog proposal identifiers -o json"
      Then every page of the history will be read
      And the document will carry "complete" as true with every boundary dimension clear
      And the command will exit with code 0

    # Source: 080-identifier-mapping-harvest — Scenario: A walk that fails partway still yields what it gathered
    @wip
    Scenario: A mid-walk failure emits the partial mapping and exits non-zero
      Given a complete connection context with a stored token
      And the proposal history's second page fails with a server error
      When an agent runs "glassfrog proposal identifiers -o json"
      Then the mapping gathered from the first page will be emitted with "complete" as false and "stopped" naming the cause
      And a note on stderr will state the identifier mapping is partial
      And the command will exit with code 3

    # Source: 080-identifier-mapping-harvest — Scenario: An unreadable record is skipped and counted, not the end of the walk
    @wip
    Scenario: An unreadable record is skipped and counted, not the end of the walk
      Given a complete connection context with a stored token
      And the walked history carries one record that cannot be read as a proposal among readable proposals
      When an agent runs "glassfrog proposal identifiers -o json"
      Then the mapping will carry the identifiers from every readable proposal
      And the document will report "complete" as false with "unreadable_records" as 1
      And a note on stderr will name 1 skipped record
      And the command will exit with code 0

    # Source: 080-identifier-mapping-harvest — Scenario: A walk that fails immediately yields no mapping
    @wip
    Scenario: An immediate walk failure emits no mapping
      Given a connection context whose token the API refuses
      When an agent runs "glassfrog proposal identifiers -o json"
      Then no mapping document will be emitted
      And the diagnostic and exit code will be the ones the proposal list read produces

    # Source: 080-identifier-mapping-harvest — Scenario: A history carrying no identifiers is an empty mapping, not a failure
    @wip
    Scenario: A history carrying no identifiers is an empty mapping, not a failure
      Given a complete connection context with a stored token
      And a proposal history whose changes carry no numeric identifiers
      When an agent runs "glassfrog proposal identifiers"
      Then "no identifiers found in the proposal history" will be printed to stdout
      And the same harvest with json output will carry a total of 0 and an empty entries array
      And the command will exit with code 0

    # Source: 080-identifier-mapping-harvest — Scenario: An unsupported filter value is refused before any request
    @wip
    Scenario: An unsupported status filter is refused before any request
      Given a complete connection context with a stored token
      When an agent runs "glassfrog proposal identifiers --status pending"
      Then stderr will name the unsupported value "pending" and list the supported statuses
      And no request will reach the API
      And the command will exit with code 2

    # Source: 080-identifier-mapping-harvest — Scenario: A narrowed walk lists its entries where a whole-history walk summarizes
    @wip
    Scenario: A narrowed walk lists entries where a whole-history walk summarizes
      Given a complete connection context with a stored token
      And the proposal history carries identifiers in several proposals
      When an agent runs "glassfrog proposal identifiers --status accepted"
      Then the human render will state the walk was narrowed and list the entries
      And the same command without "--status" will render a summary with no entries listed
      And both commands will exit with code 0

    # Source: 080-identifier-mapping-harvest — Scenario: The same run twice produces the same output
    @wip
    Scenario: Two runs over an unchanged history emit identical output
      Given a complete connection context with a stored token
      And an unchanged proposal history
      When an agent runs "glassfrog proposal identifiers -o json" twice
      Then the two outputs will be byte-identical
      And the entries and the labels within each entry will appear in the same order both times

    # Source: 080-identifier-mapping-harvest — Scenario: A first-page opt-out reports partial even when that page was the whole history
    @wip
    Scenario: A first-page opt-out on a single-page history is partial and silent
      Given a complete connection context with a stored token
      And the proposal history fits on a single page
      When an agent runs "glassfrog proposal identifiers --first-page -o json"
      Then the document will carry "complete" as false with "first_page" as true
      And no note about more proposals will be written to stderr
      And the command will exit with code 0

    # Source: 080-identifier-mapping-harvest — Scenario: A first-page opt-out on a longer history says more proposals exist
    @wip
    Scenario: A first-page opt-out on a longer history notes that more proposals exist
      Given a complete connection context with a stored token
      And the proposal history spans more than one page
      When an agent runs "glassfrog proposal identifiers --first-page -o json"
      Then the document will carry "complete" as false with "first_page" as true
      And a note on stderr will state that more proposals exist than were harvested
      And the command will exit with code 0

    # Source: 080-identifier-mapping-harvest — Scenario: Nothing is written to disk
    @validation @wip
    Scenario: Nothing is written to disk
      Given a complete connection context with a stored token
      And the proposal history carries identifiers
      When an agent runs "glassfrog proposal identifiers -o json" to completion
      Then no mapping, index, or cache file will have been created or modified
      And a second run will read the history again rather than reading anything back

    # Source: 080-identifier-mapping-harvest — Scenario: The mapping's completeness claim can be trusted
    @validation @wip
    Scenario: Exactly the completed unfiltered walk claims completeness
      Given a complete connection context with a stored token
      And six harvest runs ending completed clean, filtered, opted out after the first page, stopped on an error, completed past an unreadable record, and completed past an unrepresentable value
      When each resulting output is inspected in every format
      Then only the clean completed unfiltered walk will report "complete" as true
      And each of the other five will report "complete" as false

    # Source: 080-identifier-mapping-harvest — Scenario: The human summary's counts account for every entry
    @validation @wip
    Scenario: The human summary's counts account for every entry
      Given a complete connection context with a stored token
      And a whole-history harvest whose human render shows a summary
      When the summary's counts are compared against the structured output of the same run
      Then every entry will be accounted for by exactly one element-type count
      And the unlabelled count will match the number of entries carrying no label
