# Source: 023-version-embedding — Scenario: a release build reports the stamped version

Feature: Runtime-Dependent Distribution — Version Embedding
  On its own the binary has no real version to report, so a shipped release
  would announce itself as a development placeholder. Version Embedding gets a
  meaningful version into the binary: the release build stamps the supplied tag
  version through the one build entry point, and at runtime the binary resolves
  which version to report by a fixed precedence — an embedded version wins;
  otherwise the version Go recorded in the binary's module build info is used
  verbatim; otherwise a clear development placeholder. So `--version` names the
  build actually running, whichever way the CLI was obtained. Build (021) is
  upstream; the release pipeline (022) supplies the version from the published
  tag; Help & Version (003) renders the value.
  (affects: Practitioner, AI agent, Maintainer)

  Rule: Confirm the exact published build from the reported version
    # In order to confirm I am operating the exact published build before I trust its behavior,
    # as an AI agent operating the CLI,
    # I want --version to report the real release version that was shipped, not a development placeholder.

    # Source: 023-version-embedding — Scenario: a pre-release version is embedded verbatim
    @wip
    Scenario: A pre-release version is reported verbatim
      Given a build had the release version "v1.4.0-rc.1" supplied
      When the produced binary is asked for its version
      Then it will report "v1.4.0-rc.1"
      And the pre-release suffix will be preserved exactly

    # Source: 023-version-embedding — Scenario: an embedded version overrides recorded build info
    @wip
    Scenario: An embedded version wins over recorded build info
      Given a binary had the version "v1.4.0" supplied at build time
      And the same binary carried module build info recording a different version
      When it is asked for its version
      Then it will report "v1.4.0"
      And the recorded build info will be ignored in favor of the embedded version

    # Source: 023-version-embedding — Scenario: the resolution precedence holds in order
    @validation @wip
    Scenario: The version is resolved by a fixed precedence
      Given the three version sources are an embedded version, recorded build info, and a placeholder
      When the version is resolved for a binary carrying any combination of them
      Then an embedded version will be reported when present
      And otherwise recorded build info will be reported when present
      And otherwise the placeholder will be reported

    # Source: 023-version-embedding — Scenario: the version determination never reaches out at runtime
    @validation @wip
    Scenario: Version determination needs no network or VCS at runtime
      Given a produced binary ran on a clean host with no network and no git checkout
      When it is asked for its version
      Then it will resolve and report a version without any runtime lookup
      And it will not fail for lack of network or VCS access

  Rule: Get a meaningful version from a source install
    # In order to get a meaningful version even when I installed the CLI from source,
    # as a practitioner or maintainer who ran go install,
    # I want the version to reflect the module version Go recorded for that install.

    # Source: 023-version-embedding — Scenario: a tagged source install reports the recorded module version
    @wip
    Scenario: A tagged source install reports the recorded module version
      Given the CLI was installed from a tagged module version "v1.3.2" with no version supplied to the build
      When the installed binary is asked for its version
      Then it will report "v1.3.2" from the module build info Go recorded

    # Source: 023-version-embedding — Scenario: an untagged source install reports the pseudo-version verbatim
    @wip
    Scenario: An untagged source install reports the pseudo-version verbatim
      Given the CLI was installed from an untagged commit so Go recorded a pseudo-version "v0.0.0-20260101120000-abc123def456"
      When the installed binary is asked for its version
      Then it will report "v0.0.0-20260101120000-abc123def456" verbatim
      And the reported value will identify the exact commit

    # Source: 023-version-embedding — Scenario: a plain local build reports Go's development marker
    @wip
    Scenario: A plain local build reports Go's development marker
      Given a binary was produced by a plain local build with no version supplied
      And Go recorded the module version as "(devel)"
      When it is asked for its version
      Then it will report "(devel)" verbatim

    # Source: 023-version-embedding — Scenario: no embedded version and no build info reports the placeholder
    @wip
    Scenario: A build with no embedded version and no build info reports the placeholder
      Given a binary was built with no version supplied and with no module build info available
      When it is asked for its version
      Then it will report a clear, non-empty development placeholder
      And it will not report an empty string

    # Source: 023-version-embedding — Scenario: version output is never empty
    @validation @wip
    Scenario: Version output is never empty for any build or install path
      Given the CLI was obtained by a release build, a tagged source install, an untagged source install, a plain local build, or a build with no module info
      When the version is requested
      Then the output will be a non-empty string in every case

  Rule: Ship releases that self-report their version through one build entry point
    # In order to ship releases whose binaries self-report their version with no extra manual step,
    # as a maintainer,
    # I want the release version to be stamped into every binary through the one build entry point the pipeline already invokes.

    # Source: 023-version-embedding — Scenario: a release build reports the stamped version
    @wip
    Scenario: A release build reports the stamped version
      Given a build had the release version "v1.4.0" supplied
      When the produced binary is asked for its version via "--version"
      Then it will report "v1.4.0"
      And the "version" command will report the same "v1.4.0"

    # Source: 023-version-embedding — Scenario: no formatting or render logic leaks into version resolution
    @validation @wip
    Scenario: Version resolution produces a value and leaves formatting to Help & Version
      Given the version-resolution surface is inspected
      When its responsibilities are traced
      Then it will produce a version value
      And it will not decide how "--version" is printed
      And it will not add commit or date build metadata to the value

    # Source: 023-version-embedding — Proposed: a blanked injection seam is caught by a config assertion (plan Risks; interface Error Communication)
    @wip
    Scenario: A blanked version-injection seam is caught before release
      Given the build configuration had its version-injection seam blanked so no version is stamped
      When the build-config assertion runs
      Then it will fail
      And it will report that the configuration no longer injects the version variable
