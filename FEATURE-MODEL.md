# Glassfrog CLI — Feature Model

This document captures the solutions and capabilities for the Glassfrog CLI, organized by the problems they solve. Each solution addresses a specific problem; capabilities within a solution describe what needs to be built. The format is semantic markdown — headings for solutions, blockquotes for problems, dashes for capabilities, pluses for relationships.

---

## CLI Skeleton
> Problem: No Runnable CLI — no project skeleton or command framework exists to build any command on (affects: Maintainer)

- Command Registration — define and plug in subcommands so new commands attach without touching unrelated ones
- Argument Dispatch — parse the invocation arguments and route them to the correct registered command
  + depends-on: Command Registration
- Help & Version — --help / usage output, a discoverable listing of registered commands, and --version
  + depends-on: Command Registration
- Exit-Code Convention — standardized process exit codes (success / usage error / runtime error) for CI and agent consumption
  + depends-on: Argument Dispatch

## Token Authentication
> Problem: Unauthenticated Access — the CLI has no way to prove it's acting as a specific org + person, so Glassfrog can't authorize its calls (affects: AI agent, Practitioner)

- Credential Storage — a login-style command that accepts a token and writes it to a credentials file in the home or working directory (npmrc-style)
- Credential Discovery — locate and read the credentials file at call time, with the working-directory file taking precedence over the home-directory one (nearest-wins)
- Request Authentication — attach the resolved token as the X-Auth-Token header on outgoing API calls so Glassfrog authorizes them as the specific org + person
  + depends-on: Credential Discovery

## Connection Configuration
> Problem: Undefined Connection Settings — the CLI doesn't know which token or base URL to use, or where to read them from (affects: Practitioner)

- Connection Resolution — resolve the effective base URL by precedence (command flag > environment variable > config file > built-in default), read the value from the same npmrc-style credentials file that Credential Discovery locates (nearest-wins), then combine it with the discovered token to form the single connection context each request uses
  + depends-on: Credential Discovery
