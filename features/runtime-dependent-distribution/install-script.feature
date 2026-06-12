# Source: 027-install-script — Scenario: fresh install on a supported platform

Feature: Runtime-Dependent Distribution — Install Script
  A POSIX shell one-liner, hosted in this repository, that turns a clean
  Linux or macOS host into one with a working `glassfrog` binary. It detects
  the host OS and architecture, resolves a release (the latest stable by
  default, or a pinned version), downloads the matching archive and the
  checksums file from GitHub Releases, verifies the archive's integrity, and
  installs the binary into a per-user directory on PATH — no runtime, no
  package manager, no sudo. It consumes the archives the Automated Release
  Pipeline (022) attaches and is one of several acquisition channels
  alongside Homebrew (036) and the npm wrapper (037).
  (affects: Practitioner, AI agent, Maintainer)

  Rule: Acquire a working CLI on a bare host with one command
    # In order to get a working CLI on a fresh Linux or macOS machine without installing a runtime or package manager,
    # as an AI agent or practitioner,
    # I want to run a single command that detects my platform, downloads the right binary, verifies it, and puts it on my PATH.

    # Source: 027-install-script — Scenario: fresh install on a supported platform
    Scenario: Fresh install on a supported platform
      Given a clean Linux amd64 host with no "glassfrog" binary installed
      And the latest stable release has the four platform archives and a checksums file attached
      When the operator runs the install script with no configuration
      Then it will download the "glassfrog_<version>_linux_amd64.tar.gz" archive and the checksums file
      And it will verify the archive against the checksums file
      And it will install the binary into a per-user directory on PATH without sudo
      And it will report the install location and the installed version

    # Source: 027-install-script — Scenario: unsupported platform is refused
    Scenario: Unsupported platform is refused
      Given the install script is run on a host whose platform is not a supported target
      When platform detection completes
      Then it will stop with a message naming the detected platform and the supported targets
      And it will install nothing
      And it will exit with a non-zero status

    # Source: 027-install-script — Scenario: install directory not on PATH
    Scenario: Install directory not on PATH still installs with guidance
      Given the chosen install directory is not present in the operator's PATH
      When the script finishes installing the binary
      Then it will report the install location
      And it will print the exact line to add the directory to PATH
      And it will not modify any shell profile or environment file

    # Source: 027-install-script — Proposed: missing required tooling fails before any download (interface Error Communication; plan tooling detection)
    Scenario: A host missing required tooling fails before any download
      Given a host that has neither a usable downloader nor a sha256 utility
      When the operator runs the install script
      Then it will stop before downloading anything
      And it will report which tool category is missing and what satisfies it
      And it will exit with a usage error status

  Rule: Provision unattended — pin the version and the location, no sudo
    # In order to provision the CLI reproducibly in CI without interactive privilege prompts,
    # as a maintainer writing a pipeline,
    # I want to run the install script unattended, optionally pinning a version and choosing an install directory, with no sudo required.

    # Source: 027-install-script — Scenario: install a pinned version
    Scenario: Install a pinned version
      Given a published release "v1.3.0" exists alongside a newer "v1.4.0"
      When the operator runs the script requesting version "v1.3.0"
      Then it will resolve the "v1.3.0" release rather than the latest
      And it will download, verify, and install the "v1.3.0" binary

    # Source: 027-install-script — Scenario: custom install directory
    Scenario: Install into a custom directory
      Given the operator sets a writable custom install directory
      When the script runs
      Then it will install the binary into that directory instead of the default
      And it will report that location

    # Source: 027-install-script — Proposed: a pinned version that does not exist fails clearly (interface Error Communication)
    Scenario: A pinned version that does not exist fails clearly
      Given the operator requests a version that has no published release
      When the script attempts to download the matching archive
      Then it will stop with a message naming the requested version
      And it will install nothing
      And it will exit with a non-zero status

    # Source: 027-install-script — Scenario: latest resolution excludes pre-releases
    @validation @wip
    Scenario: Latest resolution installs the newest stable, not a pre-release
      Given the newest release is marked as a pre-release and an older release is the newest stable one
      When the script runs with no version pinned
      Then it will resolve and install the older stable release rather than the pre-release
      And a pre-release will be installed only when its version is explicitly pinned

    # Source: 027-install-script — Scenario: the installed binary reports the resolved version
    @validation @wip
    Scenario: The installed binary reports the resolved version
      Given a fresh install resolving the latest stable release
      When the installed binary is run with "--version"
      Then the reported version will equal the version the script said it installed
      And it will equal the resolved release's tag

  Rule: Trust the binary is authentic before it lands on PATH
    # In order to trust that the binary I just installed is the authentic released artifact,
    # as an operator,
    # I want the download verified against the release's checksums file before it lands on my PATH.

    # Source: 027-install-script — Scenario: checksum mismatch aborts the install
    Scenario: Checksum mismatch aborts the install
      Given the downloaded archive does not match its entry in the checksums file
      When the script runs
      Then it will stop before installing
      And no "glassfrog" binary will be written to the target location
      And it will exit with a non-zero status naming the integrity failure

    # Source: 027-install-script — Scenario: verification gates the install
    @validation @wip
    Scenario: A corrupted download never reaches the install directory
      Given a deliberately corrupted archive whose checksum will not match
      When the script runs to completion or failure
      Then at no point will a binary appear at the target location
      And the script's exit status will be non-zero

  Rule: Upgrade or roll back in place by re-running
    # In order to move to a new version (or roll back to a known-good one),
    # as an operator who already has the CLI installed,
    # I want to re-run the script and have it replace the existing binary in place.

    # Source: 027-install-script — Scenario: re-running upgrades in place
    Scenario: Re-running upgrades in place
      Given "glassfrog" "v1.3.0" is already installed at the target location
      And "v1.4.0" is the latest stable release
      When the operator re-runs the script with no version pinned
      Then it will install "v1.4.0" over the existing binary at the same location
      And the installed binary will report "v1.4.0"
