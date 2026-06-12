# Source: 040-resolution-call-site-retrofit — Scenario: Token resolved from the environment

Feature: Duplicated Setting Resolution — Resolution Call-Site Retrofit
  The token, base URL, and output-format resolvers each re-implement the same
  flag → env → .glassfrogrc → default chain. This slice migrates all three onto
  the one composable resolver (039), mapping its provenance back onto each site's
  existing output type so consumers are untouched — a behaviour-preserving change
  at every public surface, save one deliberate fix: an explicitly-supplied empty
  or whitespace --base-url / --output is now honoured by presence and fails loud
  instead of being silently ignored. (affects: Maintainer, Practitioner)

  Rule: Express each setting's resolution as a composition of shared resolve sources
    # In order to maintain one precedence implementation instead of three drifting copies,
    # as a Maintainer,
    # I want to express each setting's resolution as a composition of shared resolve sources.

    # Source: 040-resolution-call-site-retrofit — Scenario: Token resolved from the environment
    @wip
    Scenario: The token resolver returns the environment value through the composed walk
      Given GLASSFROG_TOKEN had been set to a non-empty value
      When the token is resolved
      Then it will return that token with the source reported as the environment
      And it will not read any ".glassfrogrc"

    # Source: 040-resolution-call-site-retrofit — Scenario: Output selection falls through to the built-in default
    @wip
    Scenario: The output selection falls through the composed chain to the built-in default
      Given no "--output" flag, no GLASSFROG_OUTPUT, and no ".glassfrogrc" output key
      When the output selection is resolved
      Then it will return the built-in default "full"
      And it will report no source on success (output surfaces provenance only on a format error)

    # Source: 040-resolution-call-site-retrofit — Scenario: Base URL falls through flag→env→file→default
    @wip
    Scenario: The base URL falls through an unsupplied flag and unset environment to the file
      Given the "--base-url" flag had not been supplied and GLASSFROG_BASE_URL was unset
      And the nearest ".glassfrogrc" carried "base_url = https://team.example.com/api/v5"
      When the base URL is resolved
      Then it will return that file value with the source reported as the file and its path

    # Source: 040-resolution-call-site-retrofit — Scenario: Unparseable .glassfrogrc on the output walk fails loud
    @wip
    Scenario: An unparseable config file on the output walk fails loud without using the default
      Given no "--output" flag and no GLASSFROG_OUTPUT
      And the nearest ".glassfrogrc" was malformed
      When the output format is resolved
      Then it will surface the typed config read error naming that file
      And it will not fall through to the built-in default

    # Source: 040-resolution-call-site-retrofit — Scenario: No precedence skeleton remains at a call site
    @validation @wip
    Scenario: No call site re-implements the precedence skeleton
      Given the retrofit is complete
      When a reviewer inspects the token, base-URL, and output-format resolvers
      Then each will express precedence as a composition of resolve sources
      And none will re-implement the env-trim, file-walk, or source-ordering skeleton by hand

  Rule: Honour an explicitly-supplied flag by its presence
    # In order to get a loud, immediate error when I explicitly pass an empty or malformed --base-url or --output rather than having it silently ignored,
    # as a Practitioner,
    # I want the flag I supplied to be honoured as the winning source by presence, not quietly dropped when its value is blank.

    # Source: 040-resolution-call-site-retrofit — Scenario: Base URL resolved from a supplied flag
    @wip
    Scenario: A supplied base-URL flag wins its rung
      Given "--base-url https://example.com/api/v5" had been supplied
      When the base URL is resolved
      Then it will return that value with the source reported as the flag
      And it will not consult the environment or any ".glassfrogrc"

    # Source: 040-resolution-call-site-retrofit — Scenario: Malformed base URL from the flag fails loud
    @wip
    Scenario: A malformed base-URL flag fails loud without consulting lower sources
      Given "--base-url not-a-url" had been supplied
      When the base URL is resolved
      Then it will fail with a usage error naming "--base-url"
      And no lower-precedence source will be consulted

    # Source: 040-resolution-call-site-retrofit — Scenario: Explicitly empty --base-url is honoured by presence and fails loud
    @wip
    Scenario: An explicitly empty base-URL flag is honoured by presence and fails loud
      Given "--base-url" had been supplied with an empty or whitespace value
      When the base URL is resolved
      Then the flag will win its rung by its presence
      And it will fail with a usage error naming "--base-url"
      And it will not fall through to the environment

    # Source: 040-resolution-call-site-retrofit — Scenario: Whitespace-only GLASSFROG_OUTPUT is treated as absent and falls through
    @wip
    Scenario: A whitespace-only output environment value is treated as absent and falls through
      Given the "--output" flag had not been supplied and GLASSFROG_OUTPUT was set to whitespace only
      And no ".glassfrogrc" output key was present
      When the output format is resolved
      Then it will treat the environment value as absent
      And it will return the built-in default "full"

    # Source: 040-resolution-call-site-retrofit — Proposed: presence is detected wherever the flag sits on the command path (plan: Changed() on inherited persistent flags)
    @wip
    Scenario: An empty base-URL flag fails loud regardless of its position on the command path
      Given "glassfrog --base-url \"\" me" and "glassfrog me --base-url \"\"" are both invoked
      When each command resolves the base URL
      Then both will detect the flag as supplied and fail with the same usage error

  Rule: Leave every public output surface unchanged for consumers
    # In order to keep relying on the token, base-URL, and output-format outputs exactly as before,
    # as a downstream consumer (Request Authentication, connection assembly, the read commands),
    # I want the retrofit to leave every public output type, provenance enum, and typed error unchanged.

    # Source: 040-resolution-call-site-retrofit — Scenario: No token anywhere is a normal empty outcome
    @wip
    Scenario: No token anywhere remains a normal empty outcome
      Given GLASSFROG_TOKEN was unset and no ".glassfrogrc" carried a token key
      When the token is resolved
      Then it will report the source as none with no error
      And the resolution will carry no token value

    # Source: 040-resolution-call-site-retrofit — Scenario: Public surfaces are byte-for-byte stable for consumers
    @validation @wip
    Scenario: The public output types and typed errors are unchanged
      Given a consumer that reads the token, base-URL, and output-format outputs
      When it reads their fields, provenance members, and typed-error messages after the retrofit
      Then each will be unchanged from before the retrofit

    # Source: 040-resolution-call-site-retrofit — Scenario: The flag-semantics change is the only observable behaviour difference
    @validation @wip
    Scenario: The presence change is the only observable behaviour difference
      Given the same environment and ".glassfrogrc" as before the retrofit
      When every documented resolution path is exercised
      Then the only differing observable outcome will be that an explicitly-supplied empty or whitespace "--base-url" or "--output" now fails loud instead of falling through
