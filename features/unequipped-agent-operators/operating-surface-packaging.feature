# Source: 070-operating-surface-packaging — Scenario: Discovering the plugin through the marketplace

Feature: Operating-Surface Packaging
  The Agent Operating Surface exists as a committed Claude plugin (the manifest,
  orientation, operator paths, and write-safety hook that 062–069 built), but
  nothing publishes it — an agent environment cannot discover or install it, and
  a fresh environment has no guided path from "plugin installed" to "CLI present
  and authenticated". Operating-Surface Packaging is the distribution vehicle:
  a repo-shipped Claude marketplace at `.claude-plugin/marketplace.json` (named
  `glassfrog`, general in shape — one entry today, siblings appended later)
  plus a `glassfrog-setup` skill that presence-checks the CLI and auth-checks
  the credential, pointing at the CLI's existing install channels and
  `X-Auth-Token` setup when either is missing. Distribution only: it adds no
  operating knowledge, no command, and no API capability, and an
  `internal/build` consistency guard keeps the marketplace entry truthful to
  the plugin it ships.
  (affects: AI agent, Practitioner)

  Rule: Install the operating surface from the repo, not by hand-copied files
    # In order to get the operating surface into my agent's environment without
    # hand-copying files or rediscovering where it lives,
    # as a practitioner (or whoever provisions the agent),
    # I want to add this repo as a Claude marketplace and install the glassfrog
    # plugin from it.

    # Source: 070-operating-surface-packaging — Scenario: Discovering the plugin through the marketplace
    Scenario: Marketplace add lists the glassfrog plugin
      Given an agent environment had a Claude plugin host
      When the environment adds "Luscii/cli-glassfrog" as a plugin marketplace
      Then the host will find the marketplace manifest at ".claude-plugin/marketplace.json"
      And the marketplace will list the "glassfrog" plugin as an installable entry

    # Source: 070-operating-surface-packaging — Scenario: Installing and running the plugin
    Scenario: Install brings the plugin's surface into the environment
      Given the repository had been added as a plugin marketplace
      When the environment installs the "glassfrog" plugin from it
      Then the plugin's skills, agents, and write-safety hook will become available in that environment

    # Source: 070-operating-surface-packaging — Scenario: The marketplace entry drifts from the plugin
    Scenario: Marketplace entry drift is a defect
      Given the marketplace listed the "glassfrog" plugin
      When the entry no longer resolves to a matching plugin definition in the repo
      Then the mismatch will be treated as a defect to fix
      And it will not be accepted as a tolerable difference

    # Source: 070-operating-surface-packaging — Scenario: The marketplace matches what it ships
    @validation
    Scenario: Marketplace entry matches the plugin it ships
      Given the marketplace entry for the "glassfrog" plugin
      When the named plugin and its "./plugin" source are resolved against the repo
      Then the source will point at the real plugin definition
      And the entry's identity will match the plugin manifest's

    # Source: 070-operating-surface-packaging — Scenario: Distribution only — no new surface
    @validation
    Scenario: Packaging adds no operating surface of its own
      Given the packaging artifacts produced by this feature
      When they are inspected for orientation content, operator paths, commands, or API capability
      Then none will be present
      And every operating fact will still live in the plugin the marketplace distributes

    # Source: 070-operating-surface-packaging — architecture-informed (interface guard contract, ADR-2; proposed by skill)
    Scenario: Guard fails when a version pin appears on the marketplace entry
      Given the "glassfrog" marketplace entry carried a "version" key
      When the internal/build consistency guard runs
      Then the guard will fail
      And it will report that the plugin version is single-sourced in the plugin manifest

  Rule: Go from installed plugin to ready-to-drive through a guided setup
    # In order to start driving the CLI immediately after installing the plugin,
    # as an AI agent,
    # I want to run a setup skill that confirms the CLI is installed and I am
    # authenticated, and tells me exactly what to do when either is missing.

    # Source: 070-operating-surface-packaging — Scenario: A ready environment reported ready
    Scenario: Ready environment is reported ready
      Given the plugin was installed
      And the "glassfrog" CLI was present in the environment
      And a working credential was configured
      When the operator invokes the setup skill
      Then the setup skill will confirm the CLI presence and the authenticated identity
      And it will report the environment ready to drive the CLI

    # Source: 070-operating-surface-packaging — Scenario: The CLI is not installed
    Scenario: Missing CLI routes to the install channels
      Given the plugin was installed
      And the "glassfrog" CLI was not present in the environment
      When the operator invokes the setup skill
      Then the setup skill will report the CLI as missing
      And it will direct the operator to the install script, Homebrew tap, and npm wrapper channels
      And it will not attempt to install or bundle the binary itself

    # Source: 070-operating-surface-packaging — Scenario: No working credential is configured
    Scenario: Failing credential routes to the CLI's own setup
      Given the "glassfrog" CLI was present in the environment
      And no working credential was configured
      When the operator invokes the setup skill
      Then the auth check will fail
      And the setup skill will guide the operator through the CLI's existing X-Auth-Token setup
      And it will introduce no credential mechanism of its own

    # Source: 070-operating-surface-packaging — Scenario: The CLI stays self-contained
    @validation
    Scenario: Setup leaves the CLI self-contained
      Given the produced setup skill content
      When it is inspected for how it handles a missing CLI or a missing credential
      Then it will only point at the CLI's existing install channels and credential setup
      And it will install no binary and store no credential of its own

    # Source: 070-operating-surface-packaging — architecture-informed (interface setup journey; proposed by skill)
    Scenario: Setup re-checks after a fix instead of assuming success
      Given the setup skill had directed the operator to an install channel for a missing CLI
      When the operator completes the fix
      Then the setup skill will run the presence check again before moving to the auth check
      And a failing re-check will route back to the fix, never to a ready report

  Rule: Carry a future sibling plugin without new distribution machinery
    # In order to add a second operating-surface plugin later without standing
    # up new distribution machinery,
    # as a maintainer,
    # I want the marketplace to be a general one the glassfrog plugin is simply
    # the first entry of.

    # Source: 070-operating-surface-packaging — Scenario: A second plugin is added to the marketplace
    Scenario: A sibling plugin is one appended entry
      Given the marketplace shipped with the "glassfrog" plugin as its only entry
      When a sibling operating-surface plugin is later added
      Then it will be listed as an additional entry in the plugins list
      And the marketplace will require no restructuring to carry it

    # Source: 070-operating-surface-packaging — Scenario: The marketplace is general, not glassfrog-locked
    @validation
    Scenario: Marketplace shape admits additional entries
      Given the marketplace manifest
      When it is inspected for whether it can carry more than one plugin
      Then its plugins list will admit additional entries without restructuring
