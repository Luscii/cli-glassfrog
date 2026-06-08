# Source: 022-automated-release-pipeline — Scenario: publishing a release attaches all platform archives and a checksums file

Feature: Runtime-Dependent Distribution — Automated Release Pipeline
  When a GitHub Release is published, the pipeline builds the supported
  platform binaries, packages each as a `tar.gz` archive, generates a single
  sha256 checksums file, and attaches the whole set to that release —
  automatically and entirely within this repository. It is triggered by
  publishing a release (drafted by Release Drafting, or created by hand),
  derives the version from the release's tag, honors the release's notes and
  pre-release/latest status rather than deciding them, and is atomic: any
  target build or cross-target self-containment failure aborts the release
  with nothing attached. Build (021) and version embedding (023) are upstream;
  the acquisition channels (install script, Homebrew, npm) consume what it
  attaches.
  (affects: Practitioner, AI agent, Maintainer)

  Rule: Ship every platform artifact automatically on publish
    # In order to ship a new version without manually building and uploading artifacts,
    # as a maintainer,
    # I want to publish a release and have every platform archive and a checksums file built and attached automatically.

    # Source: 022-automated-release-pipeline — Scenario: publishing a release attaches all platform archives and a checksums file
    Scenario: Publishing a release attaches all platform archives and a checksums file
      Given a release for tag "v1.4.0" had been drafted
      When a maintainer publishes the release
      Then the pipeline will build the macOS amd64, macOS arm64, Linux amd64, and Linux arm64 binaries
      And it will attach four "tar.gz" archives and one checksums file to the "v1.4.0" release

    # Source: 022-automated-release-pipeline — Scenario: a build failure aborts the whole release
    Scenario: A build failure aborts the whole release
      Given a release for tag "v1.4.0" had been published
      And the build for one target platform fails
      When the pipeline runs
      Then it will abort
      And it will attach no archives and no checksums file

    # Source: 022-automated-release-pipeline — Scenario: routine activity without a published release triggers no build
    Scenario: Routine activity without a published release triggers no build
      Given commits had been merged to main and a draft release was updated
      When no release is published
      Then the pipeline will not run
      And no archives or checksums file will be built or attached

    # Source: 022-automated-release-pipeline — Scenario: re-running for an already-published release converges on one artifact set
    Scenario: Re-running for an already-published release converges on one artifact set
      Given a "v1.4.0" release already had the artifact set attached
      When the pipeline runs again for the "v1.4.0" release
      Then it will converge on a single attached artifact set
      And it will not create duplicate or conflicting archives

    # Source: 022-automated-release-pipeline — Scenario: a hand-created published release is handled identically
    Scenario: A hand-created published release is handled identically
      Given a maintainer had created and published a "v1.4.0" release by hand without the draft-release flow
      When the pipeline runs
      Then it will build and attach the full artifact set exactly as for a drafted release

    # Source: 022-automated-release-pipeline — Proposed: a cross-target self-containment failure aborts the release (plan ADR-3)
    @wip
    Scenario: A self-containment verification failure aborts the release
      Given the four target archives had been built for a published "v1.4.0" release
      And one target binary fails the self-containment check on its own platform
      When the pipeline runs
      Then it will abort before attaching anything
      And the release will receive no archives and no checksums file

    # Source: 022-automated-release-pipeline — Scenario: the attached matrix is exactly the four supported targets
    @validation @wip
    Scenario: The attached matrix is exactly the four supported targets
      Given a published release with artifacts attached
      When the attached archives are enumerated
      Then there will be exactly one archive for each of macOS amd64, macOS arm64, Linux amd64, and Linux arm64
      And no archive will exist for any other platform such as Windows

  Rule: Verify a download against published checksums
    # In order to verify that a downloaded binary is authentic and intact,
    # as an AI agent or practitioner acquiring the CLI,
    # I want a checksums file attached alongside the archives.

    # Source: 022-automated-release-pipeline — Scenario: a consumer verifies a download against the checksums file
    @wip
    Scenario: A consumer verifies a download against the checksums file
      Given a published release had four archives and a checksums file attached
      When a consumer downloads one archive and the checksums file
      Then the archive's checksum will match its entry in the checksums file
      And the consumer will be able to confirm the download is intact before installing

    # Source: 022-automated-release-pipeline — Scenario: every attached archive has a matching checksum entry
    @validation @wip
    Scenario: Every attached archive has a matching checksum entry
      Given a published release with artifacts attached
      When the checksums file is compared against the attached archives
      Then every archive will have exactly one matching entry
      And every entry will correspond to an attached archive

  Rule: Honor the published release's pre-release and latest status
    # In order to circulate a release candidate before promoting it to stable,
    # as a maintainer,
    # I want a release published as a pre-release to receive the same artifacts without its status changing.

    # Source: 022-automated-release-pipeline — Scenario: publishing a pre-release attaches artifacts without changing its status
    Scenario: Publishing a pre-release attaches artifacts without changing its status
      Given a release for tag "v1.4.0-rc.1" had been published and marked as a pre-release
      When the pipeline runs
      Then it will build and attach the same artifact set
      And the release will remain marked as a pre-release

    # Source: 022-automated-release-pipeline — Scenario: the release's pre-release/latest status is preserved
    @validation @wip
    Scenario: The release's pre-release and latest status is preserved
      Given a release published as a pre-release and a release published as the latest
      When the pipeline attaches artifacts to each
      Then the pre-release will stay a pre-release
      And the latest will stay the latest
