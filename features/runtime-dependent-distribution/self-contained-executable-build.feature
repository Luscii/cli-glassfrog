# Source: 021-self-contained-executable-build — Scenario: a produced binary runs on a clean environment

Feature: Runtime-Dependent Distribution — Self-Contained Executable Build
  The CLI can't run without a separately-installed runtime, so it won't run
  where operators need it. Self-Contained Executable Build cross-compiles a
  single dependency-free `glassfrog` binary per supported platform (macOS and
  Linux, amd64 and arm64) from one source tree, and provides a per-target check
  that a produced binary runs on a clean host with only the OS and network —
  proven on at least the host target, with cross-target breadth deferred to 022
  — satisfying CONSTITUTION XII.
  Packaging and publishing (022) and version embedding (023) build on it.
  (affects: Practitioner, AI agent, Maintainer)

  Rule: Run on a bare host with only network access
    # In order to run the CLI in environments we don't control without first installing a runtime,
    # as an AI agent or practitioner,
    # I want a binary that runs on a bare host with only network access to the API.

    # Source: 021-self-contained-executable-build — Scenario: a produced binary runs on a clean environment
    @wip
    Scenario: A produced binary runs on a clean host
      Given a binary had been produced for a target platform
      When it runs on a clean host of that platform with only the OS and network present
      Then it will execute successfully
      And it will be able to reach the Glassfrog API

    # Source: 021-self-contained-executable-build — Scenario: the self-containment check catches a runtime dependency
    @wip
    Scenario: The self-containment check rejects a binary with a runtime dependency
      Given a produced binary required a separately-installed dependency
      When the self-containment check runs against it on a clean host of its target
      Then the check will fail and name the missing-dependency violation
      And the binary will not be treated as self-contained

    # Source: 021-self-contained-executable-build — Scenario: self-containment is per-target, not universal
    @wip
    Scenario: A binary runs only on its own target's host
      Given a binary had been produced for the macOS arm64 target
      When it is taken to a Linux host or a macOS amd64 host
      Then it will not be expected to run there
      And the self-containment guarantee will hold only on a clean host of its own target

    # Source: 021-self-contained-executable-build — Scenario: the artifact's only external need is the API
    @validation @wip
    Scenario: A produced binary needs only the API at runtime
      Given a produced binary ran on a clean host of its target
      When its external dependencies are examined
      Then its only unmet external dependency will be network access to the Glassfrog API
      And no runtime, interpreter, or installed library will be required

    # Source: 021-self-contained-executable-build — Proposed: config-guard fails when CGO_ENABLED is not 0 (plan ADR-2)
    @wip
    Scenario: Config drift to enabled cgo is rejected
      Given the build configuration had been changed to enable cgo
      When the config-guard check runs
      Then it will fail
      And it will report that cgo must remain disabled

  Rule: Ship to every supported platform from one source tree
    # In order to ship the CLI to every supported operator platform from one source tree,
    # as a maintainer,
    # I want a single release build that cross-compiles all four target binaries.

    # Source: 021-self-contained-executable-build — Scenario: release build produces the full matrix from one source tree
    @wip
    Scenario: The release build produces all four target binaries
      Given the repository source tree
      When the cross-platform release build runs
      Then one executable will be produced for each of macOS amd64, macOS arm64, Linux amd64, and Linux arm64
      And all four will come from the same single source tree

    # Source: 021-self-contained-executable-build — Scenario: a failed target fails the whole release build
    @wip
    Scenario: A failed target fails the whole release build
      Given the release build was producing the four target binaries
      When one target fails to build
      Then the release build will fail as a whole
      And it will emit no partial set of binaries

    # Source: 021-self-contained-executable-build — Scenario: cross-compilation from a foreign host
    @wip
    Scenario: Foreign-target binaries build from any host
      Given a maintainer was working on a macOS arm64 host
      When the release build runs
      Then the Linux amd64 binary will be produced
      And producing it will not require running on a Linux or amd64 host

    # Source: 021-self-contained-executable-build — Scenario: the matrix is exactly the four declared targets
    @validation @wip
    Scenario: The matrix is exactly the four supported targets
      Given the output of a release build
      When the produced binaries are enumerated
      Then there will be exactly one binary for each of the four supported targets
      And no binary will be produced for an unsupported platform such as Windows

    # Source: 021-self-contained-executable-build — Scenario: one build entry point, no second path
    @validation @wip
    Scenario: Maintainer and pipeline share one build entry point
      Given the build a maintainer invokes locally and the build the pipeline invokes
      When both are traced
      Then they will resolve to the same release-build entry point
      And the binaries produced by each cannot diverge

    # Source: 021-self-contained-executable-build — Proposed: config-guard fails on a target outside the four (plan ADR-2)
    @wip
    Scenario: An unsupported target in the build config is rejected
      Given the build configuration had declared a Windows target
      When the config-guard check runs
      Then it will fail
      And it will name the unsupported target

  Rule: Build a runnable binary for my own platform
    # In order to check a change quickly on my own machine,
    # as a maintainer,
    # I want a local build that produces a runnable binary for my own platform.

    # Source: 021-self-contained-executable-build — Scenario: local build produces a runnable host binary
    @wip
    Scenario: The local build produces a runnable host binary
      Given a maintainer was on a supported host platform
      When they run the local development build
      Then a single glassfrog executable will be produced for their own OS and architecture
      And it will run on their machine without any separately-installed runtime
