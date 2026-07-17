# Source: 063-write-safety-guardrail — Scenario: Confirming a proposal write before it runs

Feature: Write-Safety Guardrail
  The AI agent driving the CLI can run a governance write with nothing between
  its decision and the write, so it can alter governance the practitioner never
  approved or blindly retry a clobbered write. The Write-Safety Guardrail is the
  enforcing capability of the Agent Operating Surface: a Claude Code PreToolUse
  hook (in `plugin/hooks/`) over the `Bash` tool that recognizes a governance
  write on the proposal write path and routes it to an explicit human-confirmation
  prompt before it runs, while reads and operational tension edits pass through
  ungated. It adds no CLI command, flag, or capability and re-validates nothing —
  the API stays the source of truth. The `412` re-read guidance stays in Operator
  Orientation; this capability enforces that the retry is re-confirmed, not blindly
  re-sent. A best-effort drift tripwire in `internal/build` keeps the gated-command
  registry truthful to the CLI's proposal surface.
  (affects: AI agent, Practitioner)

  Rule: Keep a human decision in the loop on every governance write
    # In order to keep a deliberate human decision in the loop on every change to
    # the governance record,
    # as a practitioner whose AI agent acts on my behalf,
    # I want to personally confirm each proposal-path write before the agent sends it,
    # so the agent never alters governance without my explicit go-ahead.

    # Source: 063-write-safety-guardrail — Scenario: Confirming a proposal write before it runs
    Scenario: Gate a proposal write behind explicit confirmation
      Given the guardrail hook was active over the Bash tool
      And the agent was about to run "glassfrog proposal propose prp_0123"
      When the PreToolUse hook evaluates the command
      Then the hook will return permissionDecision "ask"
      And its message will name the command, the target "prp_0123", and that it advances the proposal into circulation
      And the write will not be sent until the practitioner explicitly confirms it

    # Source: 063-write-safety-guardrail — Scenario: Performing exactly the confirmed write
    Scenario: Run only the write that was confirmed
      Given the practitioner explicitly confirmed "glassfrog proposal create ten_0456"
      When the agent executes the confirmed write
      Then exactly the confirmed draft proposal will be created from tension "ten_0456"
      And the agent will not broaden, substitute, or bundle any additional write into the action

    # Source: 063-write-safety-guardrail — Scenario: Confirmation withheld
    Scenario: Decline a proposal write when confirmation is withheld
      Given the agent was about to run "glassfrog proposal create ten_0456"
      And the hook returned permissionDecision "ask"
      When the practitioner does not confirm the write
      Then the command will not run
      And the governance record will remain unchanged

    # Source: 063-write-safety-guardrail — Scenario: No ungated governance write
    @validation @wip
    Scenario: No proposal-write path bypasses confirmation
      Given the produced guardrail hook and its gated-command registry
      When every proposal-write leaf the registry covers is traced
      Then each will be reachable only after an explicit human confirmation
      And none will have a path that writes without asking

    # Source: 063-write-safety-guardrail — Scenario: No invented surface
    @validation @wip
    Scenario: Guardrail names no command the CLI lacks
      Given the gated-command registry
      When every command it names is checked against the shipped CLI
      Then each one will exist
      And the guardrail will add no command, flag, or capability of its own

    # Source: 063-write-safety-guardrail — Scenario: Guardrail, not coach
    @validation @wip
    Scenario: Confirmation states the change, not its governance merits
      Given the message the hook surfaces for a governance write
      When its content is inspected
      Then it will state what changes — the command, target, and effect
      And it will nowhere advise whether the change is governance-sound

    # Source: 063-write-safety-guardrail — architecture-informed (plan R1 / interface Error Communication; proposed by skill)
    Scenario: Gate an unrecognized proposal subcommand fail-closed
      Given the agent invoked "glassfrog proposal <new-write-leaf> prp_0123" that the registry did not list
      When the PreToolUse hook evaluates the command
      Then the hook will return permissionDecision "ask"
      And a future proposal write will be gated by default until the registry is updated

  Rule: Re-read and re-confirm a stale write, never blindly retry
    # In order to avoid clobbering a concurrent change when a write is refused as stale,
    # as a practitioner,
    # I want the agent to re-read and show me the current state and ask again before
    # retrying, rather than blindly re-sending the stale write.

    # Source: 063-write-safety-guardrail — Scenario: Stale-write refusal triggers re-read and re-confirm
    Scenario: Re-confirm a retry after a stale-write refusal
      Given a confirmed write was refused as a stale write with exit code 7
      When the agent re-reads the resource for its current version and retries the write
      Then the retry will itself be a proposal write the hook gates again
      And the practitioner will be asked to confirm against the now-current state before it is sent

    # Source: 063-write-safety-guardrail — Scenario: Re-confirmation withheld after a stale-write re-read
    Scenario: Hold off the retry when re-confirmation is withheld
      Given a stale-write refusal had prompted a re-read of the current state
      When the practitioner does not re-confirm against that current state
      Then the agent will not retry the write
      And the resource will remain as the concurrent change last set it

    # Source: 063-write-safety-guardrail — Scenario: A non-stale-write failure is not treated as a clobber
    Scenario: Leave a non-stale failure to normal handling
      Given a confirmed write failed with a permission outcome rather than the stale-write category
      When the agent observes the outcome
      Then it will not invoke the re-read and re-confirm recovery
      And the failure will flow through the CLI's normal failure handling unchanged

    # Source: 063-write-safety-guardrail — Scenario: No blind retry on a clobber
    @validation @wip
    Scenario: No retry without an interposed re-confirmation
      Given the guardrail's stale-write handling
      When it is inspected for a retry
      Then no retry will occur without an interposed re-read and a fresh confirmation

  Rule: Let reads and operational tension edits pass ungated
    # In order to keep reads and operational tension edits fast and frictionless,
    # as an AI agent,
    # I want read-only commands and tension edits to pass through ungated,
    # so only governance writes through the proposal path carry the confirmation cost.

    # Source: 063-write-safety-guardrail — Scenario: A read passes through ungated
    Scenario: Let a read run without confirmation
      Given the guardrail hook was active over the Bash tool
      And the agent was about to run "glassfrog roles --output json"
      When the PreToolUse hook evaluates the command
      Then the hook will not require confirmation
      And the read will proceed immediately

    # Source: 063-write-safety-guardrail — Scenario: An operational tension edit passes through ungated
    Scenario: Let a tension edit run without confirmation
      Given the agent was about to run "glassfrog tension create role_0123 --body 'onboarding flow unclear'"
      When the PreToolUse hook evaluates the command
      Then the hook will not require confirmation
      And the tension will be captured immediately

    # Source: 063-write-safety-guardrail — Scenario: Operational edits are not gated
    @validation @wip
    Scenario: Tension edits stay outside the gate
      Given the guardrail's command coverage
      When "glassfrog tension create", "glassfrog tension update", and "glassfrog tension discard" are traced
      Then none of them will require confirmation
      And the gate will cover the proposal write path only

    # Source: 063-write-safety-guardrail — architecture-informed (plan R3; proposed by skill)
    Scenario: Fall back to ungated guidance when the hook is absent
      Given the guardrail hook was not installed in the agent's host
      When the agent runs "glassfrog proposal propose prp_0123"
      Then the write will proceed under Operator Orientation's guidance only with no enforcement
      And nothing in the CLI will break

    # Source: 063-write-safety-guardrail — architecture-informed (plan ADR-4 / interface drift tripwire; proposed by skill)
    @wip
    Scenario: Drift tripwire fails when a gated leaf leaves the CLI
      Given the registry gated a proposal-write leaf the CLI's proposal surface no longer exposed
      When the internal/build drift tripwire runs
      Then the tripwire will fail
      And it will report that the proposal subcommand surface changed without the registry
