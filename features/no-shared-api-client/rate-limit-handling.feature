# Source: 017-rate-limit-handling — Scenario: a single 429 is honored and the retry succeeds

Feature: No Shared API Client — Rate-Limit Handling
  The Glassfrog API enforces a per-organization rolling rate limit and answers
  429 Too Many Requests with a Retry-After header when it is exceeded. Request
  Execution makes exactly one attempt and surfaces a 429 as a generic response
  error, leaving backoff to this capability. Rate-Limit Handling wraps the send
  seam: on a 429 for a safe read it honors the wait and re-attempts, bounded by
  caps on attempts and total wait, so a transient throttle becomes a brief pause
  rather than a failed command — and never an unbounded hang.
  (affects: AI agent, Practitioner, Maintainer)

  Rule: Transient throttles are ridden out automatically, within bounds
    # In order to have a command ride out a brief per-org throttle instead of failing the moment the API says "slow down,"
    # as a practitioner running a sequence of reads,
    # I want the CLI to honor the API's Retry-After and re-attempt automatically, within sane bounds.

    # Source: 017-rate-limit-handling — Scenario: a single 429 is honored and the retry succeeds
    @wip
    Scenario: A 429 is honored and the retry succeeds
      Given a retrying executor wrapping a client built from a complete connection context
      And the API would return a 429 with "Retry-After: 2" then a 200 response
      When a GET request is executed through the retrying executor
      Then the executor will wait about 2 seconds and re-attempt the request
      And the eventual 200 response will be returned

    # Source: 017-rate-limit-handling — Scenario: a non-429 outcome passes straight through
    @wip
    Scenario: A response that is not a 429 is returned without waiting
      Given a retrying executor wrapping a client built from a complete connection context
      And the API would return a 200 response
      When a GET request is executed through the retrying executor
      Then the 200 response will be returned
      And no wait will be imposed and only one attempt will be made

    # Source: 017-rate-limit-handling — Scenario: a 429 without Retry-After uses the fallback backoff
    @wip
    Scenario: A 429 without a usable Retry-After uses the fallback backoff
      Given a retrying executor wrapping a client built from a complete connection context
      And the API would return a 429 with no Retry-After then a 200 response
      When a GET request is executed through the retrying executor
      Then the executor will wait the fallback backoff interval and re-attempt
      And the eventual 200 response will be returned

    # Source: 017-rate-limit-handling — Scenario: a transport error is not retried
    @wip
    Scenario: A transport error is returned without a retry
      Given a retrying executor wrapping a client built from a complete connection context
      And the API could not be reached
      When a GET request is executed through the retrying executor
      Then a transport error will be returned
      And no wait will be imposed and only one attempt will be made

    # Source: 017-rate-limit-handling — Scenario: a non-429 non-2xx is passed through, not retried
    @wip
    Scenario: A non-429 non-2xx response is passed through without retrying
      Given a retrying executor wrapping a client built from a complete connection context
      And the API would return a 403 response
      When a GET request is executed through the retrying executor
      Then the 403 response error will be returned unchanged
      And no wait will be imposed and only one attempt will be made

    # Source: 017-rate-limit-handling — Scenario: a non-safe request is not auto-retried on a 429
    @wip
    Scenario: A write is not auto-retried on a 429
      Given a retrying executor wrapping a client built from a complete connection context
      And the API would return a 429 response
      When a POST request is executed through the retrying executor
      Then the 429 response error will be returned on the first occurrence
      And no wait will be imposed and the request will not be re-sent

    # Source: 017-rate-limit-handling — Scenario: every send goes through the 010 seam
    @validation @wip
    Scenario: Every attempt goes through the send seam
      Given a retrying executor wrapping a client built from a complete connection context
      When a request is handled across an initial attempt and one or more retries
      Then each attempt will be made through the client's send seam
      And the executor will not construct or send a request by itself

    # Source: 017-rate-limit-handling — Scenario: only 429 triggers a retry
    @validation @wip
    Scenario: Only a 429 triggers a retry
      Given a retrying executor wrapping a client built from a complete connection context
      When a success, a transport error, a decode error, and each non-429 status are each returned by the send seam
      Then a retry will be attempted only for a 429
      And every other outcome will be returned unchanged on the first attempt

  Rule: Waiting is bounded; the 429 surfaces once the cap is hit
    # In order to never have an automated run sleep for an unbounded stretch when the window is far from resetting,
    # as an operator (human or AI agent) driving the CLI,
    # I want the wait capped in both attempts and total time, with the 429 surfaced once the cap is hit.

    # Source: 017-rate-limit-handling — Scenario: caps reached — the 429 is surfaced unchanged
    @wip
    Scenario: The 429 is surfaced when attempts are exhausted
      Given a retrying executor wrapping a client built from a complete connection context
      And the API would return a 429 on every attempt
      When a GET request is executed through the retrying executor
      Then the executor will stop after the maximum number of attempts
      And the most recent 429 response error carrying its status, rate-limit headers, and body will be returned
      And the error will not be classified by failure kind

    # Source: 017-rate-limit-handling — Scenario: total wait is bounded regardless of Retry-After size
    @validation @wip
    Scenario: Total wait stays within the budget regardless of Retry-After size
      Given a retrying executor wrapping a client built from a complete connection context
      And the API would return 429s whose Retry-After values together exceed the total-wait budget
      When a GET request is executed through the retrying executor
      Then the executor will give up within the total-wait budget
      And it will not sleep beyond the cap
      And the last 429 response error will be returned

    # Source: 017-rate-limit-handling — Scenario: the surfaced 429 is the raw outcome, untyped
    @validation @wip
    Scenario: The surfaced 429 is the raw outcome, untyped
      Given a retrying executor that had exhausted its attempts on a 429
      When the outcome is surfaced
      Then it will carry the original 429 status, rate-limit headers, and body
      And no rate-limit-specific error type or message will have been synthesized

  Rule: Each wait is announced on stderr
    # In order to know the command is deliberately pausing rather than hung,
    # as an operator watching a command,
    # I want a short, non-secret note on stderr each time it waits before retrying.

    # Source: 017-rate-limit-handling — Scenario: a wait note is emitted to stderr before re-attempting
    @wip
    Scenario: A progress note is written to stderr before re-attempting
      Given a retrying executor wrapping a client built from a complete connection context
      And the API would return a 429 with "Retry-After: 5" then a 200 response
      When a GET request is executed through the retrying executor
      Then a progress note naming the wait and the next attempt will be written to standard error
      And the note will contain no token or secret
