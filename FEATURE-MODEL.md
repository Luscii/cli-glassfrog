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

## Self-Service Reads
> Problem: Self-Service Reads — read "what's mine": me, my roles, actions, and projects (affects: Practitioner)

Grounded in the Glassfrog API v5 spec (`spec/glassfrog-api-v5.yaml`): each capability maps to one token-scoped `/me*` endpoint, and that operation is the entrypoint its spec should start from. Line numbers are a navigation hint against the current spec revision — confirm by `operationId` if the file has moved.

- Identity Read — a `me` command that prints the authenticated actor (person/agent + organization) the token resolves to. Spec: `GET /me` → `getMe` (`spec/glassfrog-api-v5.yaml:966`); supports `?include=roles` to embed the requester's roles
  + depends-on: Request Authentication
  + depends-on: Request Execution
- My Roles — list the roles the authenticated practitioner fills via a primary, non-discarded assignment (token-scoped, not the org-wide `GET /roles`). Spec: `GET /me/roles` → `listMyRoles` (`spec/glassfrog-api-v5.yaml:1003`); paginated
  + depends-on: Request Authentication
  + depends-on: Request Execution
- My Actions — list the actions owned by roles the practitioner fills. Spec: `GET /me/actions` → `listMyActions` (`spec/glassfrog-api-v5.yaml:1092`); paginated, `?status=` filter
  + depends-on: Request Authentication
  + depends-on: Request Execution
- My Projects — list the projects owned by roles the practitioner fills. Spec: `GET /me/projects` → `listMyProjects` (`spec/glassfrog-api-v5.yaml:1040`); paginated, `?status=` filter
  + depends-on: Request Authentication
  + depends-on: Request Execution

## API Client
> Problem: No Shared API Client — the CLI can resolve a connection context but has no shared way to issue a request and apply the API's response, error, paging, and rate-limit conventions, so every endpoint command would reinvent transport plumbing (affects: AI agent, Practitioner, Maintainer)

- Request Execution — send an authenticated request through the resolved connection context to the Glassfrog API and return the parsed response or a typed transport error; the single seam every endpoint command calls through
  + depends-on: Connection Resolution
  + depends-on: Request Authentication
- API Error Extraction — interpret a non-2xx Glassfrog response into a typed error carrying the API's status and error detail, so callers receive a structured cause rather than a raw response
  + depends-on: Request Execution
- Pagination — follow the API's paging to retrieve a complete result set or signal the boundary, so list commands never silently truncate
  + depends-on: Request Execution
- Rate-Limit Handling — honor the API's per-org rate limit (429 + rate-limit headers) with backoff so request volume stays within limits
  + depends-on: Request Execution
