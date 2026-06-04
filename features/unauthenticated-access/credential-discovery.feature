# Source: 005-credential-discovery — Scenario: Home-directory file as the final fallback

Feature: Unauthenticated Access — Credential Discovery
  The CLI has no way to prove it's acting as a specific org + person, so
  Glassfrog can't authorize its calls. Before any authenticated command can
  run, the CLI must resolve the caller's token — discovering it from the
  environment or a stored credentials file at call time. (affects: AI agent, Practitioner)

  Rule: Find and use a stored token automatically
    # In order to run CLI commands without re-supplying my token on every invocation,
    # as a practitioner who stored credentials once,
    # I want the CLI to find and use my stored token automatically.

    # Source: 005-credential-discovery — Scenario: Home-directory file as the final fallback
    Scenario: The home file is used when no project file exists
      Given GLASSFROG_TOKEN was not set
      And no ".glassfrogrc" existed in the current directory or any ancestor
      And a ".glassfrogrc" in the home directory held the token "gf_home_token"
      When the CLI resolves the credential
      Then it will use the token from the home file

    # Source: 005-credential-discovery — Scenario: A file is present but holds no token
    Scenario: A tokenless file is skipped for the next source
      Given a ".glassfrogrc" in the current directory existed with no token entry
      And a ".glassfrogrc" in the home directory held the token "gf_home_token"
      And GLASSFROG_TOKEN was not set
      When the CLI resolves the credential
      Then it will skip the tokenless file
      And it will use the token from the home file

    # Source: 005-credential-discovery — Scenario: No credentials anywhere
    Scenario: No credentials anywhere is reported as absence
      Given GLASSFROG_TOKEN was not set
      And no ".glassfrogrc" existed in the current directory, any ancestor, or the home directory
      When the CLI resolves the credential
      Then it will report that no credentials were found
      And it will not fabricate a token
      And it will not raise an error of its own

    # Source: 005-credential-discovery — Scenario: A credentials file exists but cannot be read
    Scenario: An unreadable credentials file fails loudly
      Given the nearest ".glassfrogrc" existed but could not be read
      And GLASSFROG_TOKEN was not set
      When the CLI resolves the credential
      Then it will report a read error naming that file
      And it will not fall through to another source

    # Source: 005-credential-discovery — Scenario: A credentials file cannot be parsed
    Scenario: A malformed credentials file fails loudly
      Given the nearest ".glassfrogrc" held a line that was neither blank, a comment, nor a "key=value" pair
      And GLASSFROG_TOKEN was not set
      When the CLI resolves the credential
      Then it will report a format error naming that file
      And it will not report that no credentials were found

    # Source: 005-credential-discovery — Scenario: Resolution is deterministic
    @validation @wip
    Scenario: Resolution is identical across repeated runs
      Given an unchanged environment and filesystem
      When the CLI resolves the credential twice
      Then both runs will return the same token from the same source

    # Source: 005-credential-discovery — Scenario: The token value never appears in produced output
    @validation @wip
    Scenario: The token value never appears in output
      Given any resolution outcome
      When the produced output and error messages are inspected
      Then the token value will not appear in them

    # Source: 005-credential-discovery — Scenario: Discovery performs no writes
    @validation @wip
    Scenario: Resolution never writes to the filesystem
      Given any starting filesystem state
      When the CLI resolves the credential
      Then the filesystem will be unchanged afterward
      And no credentials file will be created or modified

  Rule: Inject a token through the environment
    # In order to operate the CLI in automation without persisting a file to disk,
    # as an AI agent,
    # I want to supply the token through an environment variable that overrides any stored file.

    # Source: 005-credential-discovery — Scenario: Environment variable overrides any stored file
    Scenario: The environment token overrides a stored file
      Given GLASSFROG_TOKEN was set to "gf_env_token"
      And a ".glassfrogrc" in the home directory held the token "gf_home_token"
      When the CLI resolves the credential
      Then it will use the token from GLASSFROG_TOKEN
      And it will report the source as the environment
      And it will not read any credentials file

    # Source: 005-credential-discovery — Scenario: Environment variable set but empty
    Scenario: An empty environment variable is ignored
      Given GLASSFROG_TOKEN was set to an empty value
      And a ".glassfrogrc" in the current directory held the token "gf_file_token"
      When the CLI resolves the credential
      Then it will not treat the empty variable as a token
      And it will use the token from the credentials file

  Rule: Let a project-local file take precedence
    # In order to use a different token while working inside a particular project,
    # as an operator who moves between project directories,
    # I want a project-local credentials file to take precedence over my home-directory one.

    # Source: 005-credential-discovery — Scenario: Nearest credentials file wins over the home file
    Scenario: The nearest credentials file wins over the home file
      Given a ".glassfrogrc" in the current directory held the token "gf_project_token"
      And a ".glassfrogrc" in the home directory held the token "gf_home_token"
      And GLASSFROG_TOKEN was not set
      When the CLI resolves the credential
      Then it will use the token from the current directory's file

    # Source: 005-credential-discovery — Scenario: Walk-up finds an ancestor's credentials file
    Scenario: The search ascends to an ancestor's credentials file
      Given no ".glassfrogrc" existed in the current directory
      And a ".glassfrogrc" two directories above held the token "gf_ancestor_token"
      And no ".glassfrogrc" existed in the home directory
      And GLASSFROG_TOKEN was not set
      When the CLI resolves the credential from the current directory
      Then it will use the token from the ancestor file

    # Source: 005-credential-discovery — Proposed: A home file on the ascent path is read once (plan: walk-up + home dedupe risk)
    Scenario: A home file on the ascent path is read once
      Given the home directory was an ancestor of the current directory
      And the only ".glassfrogrc" lived in the home directory holding "gf_home_token"
      And GLASSFROG_TOKEN was not set
      When the CLI resolves the credential from the current directory
      Then it will use the token from the home file
      And it will report a File source with that file's path
