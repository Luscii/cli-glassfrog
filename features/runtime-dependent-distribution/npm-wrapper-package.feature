# Source: 037-npm-wrapper-package — Scenario: npx on a supported platform resolves and runs the binary

Feature: Runtime-Dependent Distribution — NPM Wrapper Package
  An npm package that makes the released `glassfrog` binary runnable through
  the Node toolchain. A main umbrella package (`@luscii-healthtech/glassfrog`)
  lists four platform-specific packages as optional dependencies; npm installs
  only the one matching the host OS and CPU, bundling its binary (the primary,
  offline-capable path). When no matching package is available, a postinstall
  fallback downloads the archive and checksums from the release, verifies the
  archive, and places the binary. A zero-dependency launcher execs the binary,
  passing arguments and exit codes straight through. It consumes the archives
  the Automated Release Pipeline (022) attaches and is one of several
  acquisition channels alongside the Install Script (027) and Homebrew (036).
  (affects: Practitioner, AI agent, Maintainer)

  Rule: Run the CLI once via npx without a separate install step
    # In order to run the CLI once in a Node-based agent environment without a separate install step,
    # as an AI agent or practitioner,
    # I want to invoke `npx glassfrog ...` and have the right platform binary resolved and executed.

    # Source: 037-npm-wrapper-package — Scenario: npx on a supported platform resolves and runs the binary
    @wip
    Scenario: npx resolves and runs the matching platform binary
      Given a published release with the umbrella and its four platform packages on the registry
      And a Linux x64 host with Node and npm
      When the operator runs "npx @luscii-healthtech/glassfrog --version"
      Then npm will resolve the Linux x64 platform package
      And the launcher will exec that bundled binary
      And the reported version will equal the installed package version

    # Source: 037-npm-wrapper-package — Scenario: unsupported platform is refused at install
    @wip
    Scenario: Unsupported platform is refused at install
      Given the package is installed on a host whose platform is not a supported target
      When the postinstall runs
      Then it will report a message naming the detected platform and the supported targets
      And it will leave no runnable "glassfrog" command
      And the install will fail with a non-zero status

    # Source: 037-npm-wrapper-package — Scenario: offline install using the bundled platform package
    @wip
    Scenario: Offline install uses the bundled platform package
      Given the matching platform package is available in the install cache
      And there is no network access to the release host
      When the package is installed
      Then the install will complete using the bundled binary
      And it will not attempt a fallback download

    # Source: 037-npm-wrapper-package — Proposed: launcher backstop when postinstall was skipped (plan ADR-4 / Risk R3)
    @wip
    Scenario: Launcher refuses clearly when no binary is installed
      Given the umbrella was installed with "--ignore-scripts" on a supported host
      And no matching platform package was installed
      When the operator runs "glassfrog"
      Then the launcher will report the detected platform and the supported targets
      And it will advise reinstalling without "--ignore-scripts"
      And it will exit with a non-zero status

    # Source: 037-npm-wrapper-package — Scenario: each supported platform resolves exactly its own binary
    @validation @wip
    Scenario: Each supported platform resolves exactly its own binary
      Given the umbrella and the four platform packages are published
      When the package is installed on each of the four supported targets in turn
      Then each install will resolve the binary for that exact platform
      And no install will resolve a binary for a different OS or architecture

  Rule: Provision a pinned version reproducibly in CI
    # In order to provision the CLI reproducibly in a Node-centric CI pipeline,
    # as a maintainer,
    # I want to `npm i -g glassfrog@<version>` and get the matching binary, pinned to a known version.

    # Source: 037-npm-wrapper-package — Scenario: pinned global install places the matching binary
    @wip
    Scenario: Pinned global install places the matching binary
      Given a "@luscii-healthtech/glassfrog@1.3.0" package is published alongside a newer "1.4.0"
      When the operator runs "npm i -g @luscii-healthtech/glassfrog@1.3.0" on a supported platform
      Then the install will resolve the "1.3.0" platform binary
      And running "glassfrog --version" will report "1.3.0"

    # Source: 037-npm-wrapper-package — Scenario: the placed binary's version matches the package and the release tag
    @validation @wip
    Scenario: The placed binary version matches the package and the release tag
      Given the package was installed at a specific version
      When the installed binary is run with "--version"
      Then the reported version will equal the installed npm package version
      And it will equal the resolved release's tag

  Rule: Trust the downloaded binary is authentic before it runs
    # In order to trust that the binary npm placed is the authentic released artifact even when it was downloaded rather than bundled,
    # as an operator,
    # I want the fallback download verified against the release's checksums file before it becomes runnable.

    # Source: 037-npm-wrapper-package — Scenario: fallback download verifies before placing the binary
    @wip
    Scenario: Fallback download verifies before placing the binary
      Given the matching platform package is not available from the registry for the host
      And the resolved release has the platform archive and checksums file attached
      When the package is installed
      Then the postinstall will download the matching archive and the checksums file
      And it will verify the archive against its checksums entry
      And it will place the binary only after the checksum matches

    # Source: 037-npm-wrapper-package — Scenario: checksum mismatch aborts the fallback install
    @wip
    Scenario: Checksum mismatch aborts the fallback install
      Given the fallback downloaded an archive that does not match its checksums entry
      When the postinstall runs
      Then it will stop before placing the binary
      And it will leave no runnable "glassfrog" command
      And the install will fail with a non-zero status naming the integrity failure

    # Source: 037-npm-wrapper-package — Proposed: fallback fails before placing when tar is missing (plan ADR-3 / Risk R4)
    @wip
    Scenario: Missing extractor fails before any binary is placed
      Given the fallback path was taken on a host with no "tar" extractor
      When the postinstall attempts to extract the archive
      Then it will stop before placing the binary
      And it will report the missing extractor
      And the install will fail with a non-zero status

    # Source: 037-npm-wrapper-package — Scenario: verification gates the fallback path
    @validation @wip
    Scenario: A corrupted fallback download never becomes runnable
      Given a deliberately corrupted archive whose checksum will not match, on the fallback path
      When the install runs to completion or failure
      Then at no point will a runnable "glassfrog" command appear
      And the install's exit status will be non-zero

  Rule: Script the CLI through npm with passthrough fidelity
    # In order to script the CLI from Node tooling the same way I would from a shell,
    # as an AI agent,
    # I want arguments and exit codes to pass straight through the wrapper to the underlying binary.

    # Source: 037-npm-wrapper-package — Scenario: exit code and arguments pass through unchanged
    @wip
    Scenario: Exit code and arguments pass through unchanged
      Given the CLI was installed through the npm wrapper
      And the underlying binary exits non-zero for a given command
      When the operator runs that command through the wrapper
      Then the wrapper will forward the arguments to the binary
      And it will exit with the binary's own exit code unchanged
