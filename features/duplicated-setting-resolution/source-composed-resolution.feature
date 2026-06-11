# Source: 039-source-composed-resolution — Scenario: Highest-precedence source wins

Feature: Duplicated Setting Resolution — Source-Composed Resolution
  Each configurable setting (token, base URL, output format) re-implements the
  same flag → env → .glassfrogrc → default precedence chain, so adding a setting
  copies the OS seam, the chain skeleton, the source enum, and the validation-error
  shape — and the copies can drift. A single composable resolver walks an ordered
  list of value sources, returns the first that yields with its provenance, and
  leaves value validation to the caller. (affects: Maintainer)

  Rule: Compose a resolver by listing reusable source constructors
    # In order to add a new configurable setting without copying the flag→env→file→default chain and its OS seam,
    # as a Maintainer,
    # I want to compose a resolver by listing reusable source constructors.

    # Source: 039-source-composed-resolution — Scenario: Highest-precedence source wins
    @wip
    Scenario: The highest-precedence source wins
      Given a resolver composed of a flag source, an environment source, a file source, and a trailing default
      And the flag had been supplied with a value
      When the setting is resolved
      Then it will return the flag's value
      And it will report the provenance as the flag

    # Source: 039-source-composed-resolution — Scenario: Resolution falls through empty sources
    @wip
    Scenario: Resolution falls through empty sources to the environment
      Given a resolver composed of a flag source, an environment source, a file source, and a trailing default
      And the flag had not been supplied
      And the environment variable had been set to a non-empty value
      When the setting is resolved
      Then it will skip the empty flag source
      And it will return the environment value
      And it will report the provenance as the environment variable

    # Source: 039-source-composed-resolution — Scenario: Trailing default backstops an otherwise-empty chain
    @wip
    Scenario: The trailing default backstops an empty chain
      Given a resolver composed of a flag source, an environment source, a file source, and a trailing default
      And no flag, environment variable, or ".glassfrogrc" key had supplied a value
      When the setting is resolved
      Then it will return the default value
      And it will report the provenance as the default

    # Source: 039-source-composed-resolution — Scenario: No source and no default yields a valid empty result
    @wip
    Scenario: An empty chain with no default yields a valid empty result
      Given a resolver composed of an environment source and a file source with no trailing default
      And neither the variable nor any ".glassfrogrc" had carried the key
      When the setting is resolved
      Then it will report the provenance as nothing-found
      And it will report no error

    # Source: 039-source-composed-resolution — Scenario: A list-valued source yields from its first matching input
    @wip
    Scenario: A list-valued flag source yields from its first present alias
      Given a flag source composed over the aliases "--output" and "-o"
      And "--output" had not been supplied but "-o" had been supplied with a value
      When the setting is resolved
      Then it will return the value from "-o"
      And it will report the provenance origin as "-o"

    # Source: 039-source-composed-resolution — Proposed: more than one Stdin source is a composition error (interface ADR-5 panic)
    @wip
    Scenario: Composing more than one stdin source fails as a wiring error
      Given a resolver composed with two stdin sources
      When the setting is resolved
      Then it will fail loudly as a composition error
      And it will not drain the stream for the first reader

    # Source: 039-source-composed-resolution — Scenario: The resolver names no concrete setting
    @validation @wip
    Scenario: The resolver is setting-agnostic
      Given the resolver and its source constructors
      When a reviewer reads their contract
      Then nothing will reference the token, base URL, or output format by name
      And the same mechanism will serve a fourth setting unchanged

  Rule: Receive the provenance of the winning value
    # In order to give a user a precise error that names where a bad value came from,
    # as a Maintainer,
    # I want to receive the provenance of the winning value alongside the value itself.

    # Source: 039-source-composed-resolution — Scenario: Malformed config file fails loud rather than falling through
    @wip
    Scenario: A malformed config file fails loud naming the file
      Given a resolver whose file source reads ".glassfrogrc"
      And no higher-precedence source had yielded
      And the nearest ".glassfrogrc" was unreadable or malformed
      When the setting is resolved
      Then it will surface a resolution error naming that file path
      And it will not fall through to a lower-precedence source

    # Source: 039-source-composed-resolution — Scenario: Provenance is rich enough to reproduce existing error labels
    @validation @wip
    Scenario: Provenance reproduces the existing source labels
      Given a resolved value from each kind of source
      When a caller phrases an error from the provenance alone
      Then it will reproduce the labels "--base-url", "GLASSFROG_BASE_URL", and the file path
      And the resolver will not have known which setting it served

    # Source: 039-source-composed-resolution — Scenario: No resolved value leaks into diagnostics
    @validation @wip
    Scenario: No resolved value leaks into the resolver's own output
      Given a resolution that returned a value including a token
      When the resolver's own output is inspected
      Then no resolved value will appear in it

  Rule: Keep OS access behind an injectable seam
    # In order to test resolution deterministically without touching the real environment, filesystem, or terminal,
    # as a Maintainer,
    # I want the resolver's OS access to sit behind an injectable seam.

    # Source: 039-source-composed-resolution — Scenario: STDIN read failure is surfaced uniformly
    @wip
    Scenario: A stdin read failure is surfaced uniformly
      Given a resolver whose highest-precedence source reads piped stdin through an injected reader
      And reading the piped stdin had failed
      When the setting is resolved
      Then it will surface the read error directly, aborting resolution the same way a config-file failure does
