# Source: 008-base-url-resolution — Scenario: Flag overrides every other source

Feature: Undefined Connection Settings — Base URL Resolution
  The CLI doesn't know which base URL to use, or where to read it from.
  Before any API call can be addressed, the CLI must resolve one effective
  base URL — from a command flag, an environment variable, the same
  .glassfrogrc it reads the token from, or a built-in default — and it must
  always produce a value. (affects: Practitioner)

  Rule: Override the endpoint with a flag
    # In order to point the CLI at a non-production Glassfrog endpoint for a single command,
    # as an operator testing against a staging environment,
    # I want to pass the endpoint as a flag that overrides any stored or default value.

    # Source: 008-base-url-resolution — Scenario: Flag overrides every other source
    Scenario: The flag overrides every other source
      Given the base-URL flag was set to "https://staging.example.com/api/v5"
      And GLASSFROG_BASE_URL was set to "https://env.example.com/api/v5"
      And a ".glassfrogrc" in the current directory held the base URL "https://file.example.com/api/v5"
      When the CLI resolves the base URL
      Then it will use the base URL from the flag
      And it will report the source as the flag
      And it will consult no other source

    # Source: 008-base-url-resolution — Scenario: A source supplies a malformed base URL
    Scenario: A malformed flag value fails loudly
      Given the base-URL flag was set to "api.glassfrog.com" with no scheme
      When the CLI resolves the base URL
      Then it will report a format error naming the flag
      And it will not fall through to another source

  Rule: Work out of the box with a built-in default
    # In order to run the CLI against the real Glassfrog API without configuring anything first,
    # as a practitioner who just installed the tool,
    # I want a sensible built-in default base URL so commands work out of the box.

    # Source: 008-base-url-resolution — Scenario: Built-in default when nothing is configured
    Scenario: The built-in default is used when nothing is configured
      Given the base-URL flag was not set
      And GLASSFROG_BASE_URL was not set
      And no ".glassfrogrc" held a base URL in the current directory, any ancestor, or the home directory
      When the CLI resolves the base URL
      Then it will use the built-in default base URL
      And it will report the source as the built-in default

    # Source: 008-base-url-resolution — Scenario: Resolution always yields a value
    @validation @wip
    Scenario: Resolution always yields a value
      Given the base-URL flag was not set
      And GLASSFROG_BASE_URL was not set
      And no ".glassfrogrc" held a base URL anywhere in the search path
      When the CLI resolves the base URL
      Then it will produce the built-in default base URL
      And it will never report that no base URL was found

    # Source: 008-base-url-resolution — Scenario: Resolution is deterministic
    @validation @wip
    Scenario: Resolution is identical across repeated runs
      Given an unchanged flag value, environment, and filesystem
      When the CLI resolves the base URL twice
      Then both runs will return the same base URL from the same source

    # Source: 008-base-url-resolution — Scenario: Resolution performs no writes
    @validation @wip
    Scenario: Resolution never writes to the filesystem
      Given any starting filesystem state
      When the CLI resolves the base URL
      Then the filesystem will be unchanged afterward
      And no ".glassfrogrc" will be created or modified

    # Source: 008-base-url-resolution — Scenario: Resolution makes no network call
    @validation @wip
    Scenario: Resolution makes no network call
      Given any resolution outcome
      When the CLI resolves the base URL
      Then no outbound connection will be made
      And no API call will be made

  Rule: Let a project-local config take precedence
    # In order to use a project-specific endpoint across a whole working tree,
    # as an operator who moves between project directories,
    # I want a project-local config file's base URL to take precedence over my home-directory one.

    # Source: 008-base-url-resolution — Scenario: Nearest config file wins over the home file
    Scenario: The nearest config file wins over the home file
      Given the base-URL flag was not set
      And GLASSFROG_BASE_URL was not set
      And a ".glassfrogrc" in the current directory held the base URL "https://project.example.com/api/v5"
      And a ".glassfrogrc" in the home directory held the base URL "https://home.example.com/api/v5"
      When the CLI resolves the base URL
      Then it will use the base URL from the current directory's file

    # Source: 008-base-url-resolution — Scenario: Environment variable wins over config file and default
    Scenario: The environment variable wins over a config file
      Given the base-URL flag was not set
      And GLASSFROG_BASE_URL was set to "https://env.example.com/api/v5"
      And a ".glassfrogrc" in the current directory held the base URL "https://file.example.com/api/v5"
      When the CLI resolves the base URL
      Then it will use the base URL from GLASSFROG_BASE_URL
      And it will not read any config file

    # Source: 008-base-url-resolution — Scenario: A config file is present but holds no base URL
    Scenario: A config file with no base URL is skipped
      Given the base-URL flag was not set
      And GLASSFROG_BASE_URL was not set
      And a ".glassfrogrc" in the current directory existed with no base URL entry
      And a ".glassfrogrc" in the home directory held the base URL "https://home.example.com/api/v5"
      When the CLI resolves the base URL
      Then it will skip the file with no base URL
      And it will use the base URL from the home file

    # Source: 008-base-url-resolution — Scenario: Environment variable set but empty
    Scenario: An empty environment variable is ignored
      Given the base-URL flag was not set
      And GLASSFROG_BASE_URL was set to an empty value
      And a ".glassfrogrc" in the current directory held the base URL "https://file.example.com/api/v5"
      When the CLI resolves the base URL
      Then it will not treat the empty variable as a base URL
      And it will use the base URL from the current directory's file

    # Source: 008-base-url-resolution — Scenario: A config file exists but cannot be read
    Scenario: An unreadable config file fails loudly
      Given the base-URL flag was not set
      And GLASSFROG_BASE_URL was not set
      And the nearest ".glassfrogrc" existed but could not be read
      When the CLI resolves the base URL
      Then it will report a read error naming that file
      And it will not fall through to the built-in default

    # Source: 008-base-url-resolution — Scenario: A source supplies a malformed base URL
    Scenario: A malformed config-file value fails loudly naming the file
      Given the base-URL flag was not set
      And GLASSFROG_BASE_URL was not set
      And a ".glassfrogrc" in the current directory held the base URL "api.glassfrog.com" with no scheme
      When the CLI resolves the base URL
      Then it will report a format error naming that file
      And it will not fall through to another source

    # Source: 008-base-url-resolution — Proposed: A malformed value names its source for any source (interface: error names flag / env / file)
    Scenario: A malformed environment value names the environment variable
      Given the base-URL flag was not set
      And GLASSFROG_BASE_URL was set to "ftp://glassfrog.com/api/v5"
      When the CLI resolves the base URL
      Then it will report a format error naming GLASSFROG_BASE_URL
      And it will not fall through to another source
