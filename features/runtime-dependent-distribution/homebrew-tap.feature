# Source: 036-homebrew-tap — Scenario: fresh install on macOS

Feature: Runtime-Dependent Distribution — Homebrew Tap
  A Homebrew formula, published to a dedicated tap repository
  (`Luscii/homebrew-cli-glassfrog`), that installs the pre-built `glassfrog`
  release binary on macOS and Linux via `brew` — no Go toolchain, no source
  build, no manual download. GoReleaser authors the formula from the same
  archives and checksums the Automated Release Pipeline (022) attaches; on
  each published stable release a `tap` job pushes the updated
  `Formula/glassfrog.rb` to the tap repo, so nothing is committed to this
  repository's protected main. It is one of several acquisition channels
  alongside the Install Script (027) and the npm wrapper (037).
  (affects: Practitioner, AI agent, Maintainer)

  Rule: Acquire a working CLI through Homebrew on macOS or Linux
    # In order to install the CLI with the package manager I already use, on either my Mac or my Linux box,
    # as a practitioner or AI agent on macOS or Linux,
    # I want to tap the repository and brew install glassfrog, and get a working binary on PATH without a runtime or manual download.

    # Source: 036-homebrew-tap — Scenario: fresh install on macOS
    @wip
    Scenario: Fresh install on macOS
      Given a Mac on arm64 with no "glassfrog" binary installed
      And the latest stable release has the four platform archives and a checksums file attached
      And the tap's formula points at that release
      When the user runs "brew install luscii/cli-glassfrog/glassfrog"
      Then Homebrew will download the "glassfrog_<version>_darwin_arm64.tar.gz" archive
      And it will verify the archive against the formula's recorded checksum
      And it will place a working "glassfrog" binary on PATH
      And "glassfrog version" will report the installed release's version

    # Source: 036-homebrew-tap — Scenario: fresh install on Linux
    @wip
    Scenario: Fresh install on Linux
      Given a Linux amd64 host with Homebrew and no "glassfrog" binary installed
      And the tap "luscii/cli-glassfrog" has been added
      When the user runs "brew install glassfrog"
      Then Homebrew will resolve the "glassfrog_<version>_linux_amd64.tar.gz" archive
      And it will verify it against the formula's recorded checksum
      And it will install the binary on PATH

    # Source: 036-homebrew-tap — Scenario: expected release asset missing
    @wip
    Scenario: A missing release asset fails the install clearly
      Given the tap's formula references an archive that is not attached to the release
      When the user runs "brew install glassfrog"
      Then the install will fail on the missing download
      And no partial or wrong "glassfrog" binary will be placed

  Rule: Stay current with brew upgrade
    # In order to stay current with releases the way I update everything else,
    # as a user who already installed via brew,
    # I want to run brew upgrade and move to the latest stable glassfrog automatically.

    # Source: 036-homebrew-tap — Scenario: upgrade to the latest stable
    @wip
    Scenario: Upgrade moves to the latest stable
      Given "glassfrog" was installed via brew at an older stable release
      And a newer stable release has been published and the formula updated to it
      When the user runs "brew upgrade glassfrog"
      Then Homebrew will move the install to the newer stable release
      And "glassfrog version" will report the newer version

    # Source: 036-homebrew-tap — Scenario: already on the latest stable
    @wip
    Scenario: Upgrading when already current is a no-op
      Given the user is installed at the newest stable release
      When the user runs "brew upgrade glassfrog"
      Then nothing will be reinstalled
      And the installed version will be unchanged

    # Source: 036-homebrew-tap — Scenario: pre-release does not move the tap
    @wip
    Scenario: A pre-release does not move the tap
      Given the newest published release is marked as a pre-release
      And the newest stable release is an older one
      When the user runs "brew install" or "brew upgrade"
      Then Homebrew will resolve the older stable release rather than the pre-release
      And the tap repository's formula will still point at the stable release

    # Source: 036-homebrew-tap — Scenario: a pre-release leaves the tap repository untouched
    @validation @wip
    Scenario: A pre-release leaves the tap repository untouched
      Given a sequence of releases whose most recent is a pre-release
      When the tap repository's formula is inspected after that pre-release is published
      Then it will still reference the latest stable release rather than the pre-release

    # Source: 036-homebrew-tap — Proposed: the .goreleaser.yaml config-guard catches a missing or retargeted brews block (interface config-guard extension; plan config-drift guard)
    @wip
    Scenario: Config-guard fails when the brews block is blanked or retargeted
      Given the ".goreleaser.yaml" no longer has a "brews" entry targeting the "homebrew-cli-glassfrog" tap
      When the config-guard test runs in PR validation
      Then it will fail before any release is cut
      And it will name the missing or drifted brews configuration

  Rule: Trust the binary is authentic before it lands on PATH
    # In order to trust that brew installed the authentic released artifact,
    # as an operator,
    # I want Homebrew to verify the download against the checksum the release published before it lands on PATH.

    # Source: 036-homebrew-tap — Scenario: checksum mismatch refuses the install
    @wip
    Scenario: Checksum mismatch refuses the install
      Given an archive whose sha256 does not match the formula's recorded checksum
      When the user runs "brew install glassfrog"
      Then Homebrew will refuse the install and report the integrity failure
      And no "glassfrog" binary will be placed

    # Source: 036-homebrew-tap — Scenario: integrity gates the install
    @validation @wip
    Scenario: A mismatched checksum never lets a binary reach PATH
      Given a formula whose recorded checksum does not match the referenced archive
      When "brew install" runs to completion or failure
      Then at no point will a "glassfrog" binary appear on PATH from this channel

    # Source: 036-homebrew-tap — Scenario: the installed binary matches the release the formula points at
    @validation @wip
    Scenario: The installed binary matches the release the formula points at
      Given a fresh "brew install" from the tap
      When the installed binary is run with "glassfrog version"
      Then the reported version will equal the stable release the formula references
      And it will equal that release's tag

    # Source: 036-homebrew-tap — Proposed: each formula sha256 equals the release's checksums.txt entry (interface hard contract; plan Risk — reproducible archives)
    @wip
    Scenario: The published formula's checksums match the release's checksums file
      Given a stable release whose formula has been published to the tap
      When each archive recorded in the formula is checked against the release's "glassfrog_<version>_checksums.txt"
      Then every recorded sha256 will equal the matching checksums-file entry
