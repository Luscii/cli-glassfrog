# Source: 006-credential-storage — Scenario: Store a token supplied as an argument to the home file

Feature: Unauthenticated Access — Credential Storage
  The CLI has no way to prove it's acting as a specific org + person, so
  Glassfrog can't authorize its calls. Credential Storage is the write side:
  it persists a token once to the credentials file that credential discovery
  later reads, so the operator does not re-supply it on every invocation.
  (affects: AI agent, Practitioner)

  Rule: Store my token once for later use
    # In order to run CLI commands later without re-supplying my token,
    # as a practitioner setting up the CLI for the first time,
    # I want to store my token once to a credentials file the CLI will find automatically.

    # Source: 006-credential-storage — Scenario: Store a token supplied as an argument to the home file
    Scenario: A token argument is stored to the home file
      Given no ".glassfrogrc" existed in the home directory
      And the operator supplied the token "gf_new_token" as a command argument
      When the CLI stores the credential
      Then it will create the home ".glassfrogrc" holding the token "gf_new_token"
      And it will report the path it wrote
      And the token value will not appear in the output

    # Source: 006-credential-storage — Scenario: Target location is not writable
    Scenario: An unwritable target fails the store loudly
      Given the home ".glassfrogrc" location could not be written
      And the operator supplied the token "gf_new_token" as a command argument
      When the CLI stores the credential
      Then it will report a write error naming that location
      And the filesystem will be unchanged afterward

    # Source: 006-credential-storage — Scenario: Existing credentials file cannot be parsed for a merge
    Scenario: A malformed existing file fails the store loudly
      Given the home ".glassfrogrc" held a line that was neither blank, a comment, nor a "key=value" pair
      And the operator supplied the token "gf_new_token" as a command argument
      When the CLI stores the credential
      Then it will report a format error naming that file
      And it will not overwrite the file

    # Source: 006-credential-storage — Scenario: Supplied token is blank
    Scenario: A blank token is rejected
      Given the operator supplied a token that was only whitespace
      When the CLI stores the credential
      Then it will reject the token as unusable
      And it will not write any file

    # Source: 006-credential-storage — Scenario: Merge preserves other keys in an existing file
    Scenario: Re-storing preserves other entries in the file
      Given the home ".glassfrogrc" held the token "gf_old_token" and an unrelated entry
      And the session was interactive (standard input is a terminal)
      And the operator confirmed replacing the token with "gf_new_token"
      When the CLI stores the credential
      Then it will replace only the token entry with "gf_new_token"
      And it will leave the unrelated entry unchanged

    # Source: 006-credential-storage — Scenario: Interactive prompt for a missing token
    Scenario: An interactive prompt requests a missing token
      Given the session was interactive (standard input is a terminal)
      And no token was supplied as an argument, on standard input, or in GLASSFROG_TOKEN
      And no ".glassfrogrc" existed in the home directory
      When the CLI stores the credential
      Then it will prompt for the token without echoing the typed characters
      And it will write the entered token to the home ".glassfrogrc"

    # Source: 006-credential-storage — Scenario: Stored token round-trips through Discovery
    @validation @wip
    Scenario: A stored token is resolved back unchanged
      Given the operator had stored the token "gf_new_token" to a location
      When the CLI resolves the credential from that location
      Then it will use the token "gf_new_token"

    # Source: 006-credential-storage — Scenario: The token value never appears in produced output
    @validation @wip
    Scenario: The stored token value never appears in output
      Given any store outcome
      When the produced output and error messages are inspected
      Then the token value will not appear in them

    # Source: 006-credential-storage — Scenario: A stored file is not world- or group-readable
    @validation @wip
    Scenario: A newly stored file is readable only by its owner
      Given the operator stored a token to a new ".glassfrogrc"
      When the file's permissions are inspected
      Then only the owning user will be able to read it

  Rule: Provision credentials in automation
    # In order to provision credentials inside automation without a terminal,
    # as an AI agent,
    # I want to pipe a token on standard input (or persist one already in the environment) and have it written deterministically.

    # Source: 006-credential-storage — Scenario: Persist a token already present in the environment
    Scenario: A token in the environment is persisted
      Given GLASSFROG_TOKEN was set to "gf_env_token"
      And the operator supplied no argument and piped nothing to standard input
      When the CLI stores the credential
      Then it will write the token "gf_env_token" to the home ".glassfrogrc"

    # Source: 006-credential-storage — Scenario: Non-interactive session with no token supplied
    Scenario: A non-interactive store with no token is reported
      Given the session had no terminal on standard input
      And no token was supplied as an argument, on standard input, or in GLASSFROG_TOKEN
      When the CLI stores the credential
      Then it will report that there is no token to store
      And it will not write any file

    # Source: 006-credential-storage — Scenario: Existing token, non-interactive, no overwrite signal
    Scenario: A non-interactive overwrite requires the overwrite flag
      Given the home ".glassfrogrc" already held the token "gf_old_token"
      And the operator supplied the token "gf_new_token" as a command argument
      And the session had no terminal on standard input
      And the --overwrite flag was not given
      When the CLI stores the credential
      Then it will report an error
      And it will leave the existing credentials unchanged

  Rule: Store a project-local token
    # In order to use a project-specific token while working in a particular directory,
    # as an operator who moves between projects,
    # I want to store a token into a current-directory credentials file that takes precedence over my home one.

    # Source: 006-credential-storage — Scenario: Store a token piped on standard input to the current directory
    Scenario: A piped token is stored to the current directory
      Given the token "gf_project_token" was piped to standard input
      And the operator supplied no token argument
      And the --cwd flag was given
      When the CLI stores the credential
      Then it will write the token "gf_project_token" to the current directory's ".glassfrogrc"

    # Source: 006-credential-storage — Scenario: Interactive confirmation chooses the write location
    Scenario: An interactive store confirms and chooses the write location
      Given the session was interactive (standard input is a terminal)
      And a ".glassfrogrc" in the current directory and one in the home directory each held a token
      And the operator supplied the token "gf_new_token" as a command argument
      When the CLI stores the credential
      Then it will confirm before changing the existing tokens
      And it will offer to write the current-directory file, the home file, or both
