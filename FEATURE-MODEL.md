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

- Base URL Resolution — resolve the effective base URL by precedence (command flag > environment variable > config file > built-in default), read the value from the same npmrc-style credentials file that Credential Discovery locates (nearest-wins), resolving its value independently of the token
- Connection Context Assembly — combine the resolved base URL with the discovered token into the single connection context each request uses
  + depends-on: Base URL Resolution
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
  + depends-on: Connection Context Assembly
  + depends-on: Request Authentication
- API Error Extraction — interpret a non-2xx Glassfrog response into a typed error carrying the API's status and error detail, so callers receive a structured cause rather than a raw response
  + depends-on: Request Execution
- Pagination — follow the API's paging to retrieve a complete result set or signal the boundary, so list commands never silently truncate
  + depends-on: Request Execution
- Rate-Limit Handling — honor the API's per-org rate limit (429 + rate-limit headers) with backoff so request volume stays within limits
  + depends-on: Request Execution

## Output Formatting
> Problem: Unconsumable Output — results aren't shaped for an AI agent to parse reliably or for a human to read (affects: AI agent, Practitioner)

- Structured Serialization — render a command's result data as machine-readable JSON and YAML for agent consumption (VISION principle 3)
- Templated Human Rendering — render human-readable output through a template mechanism, with built-in `full` and `compact` templates; the template seam is what later admits caller-supplied templates
- Output Format Selection — an `--output` flag selecting the format per invocation (full | compact | json | yaml), with a default when omitted, dispatching the result to the matching renderer
  + depends-on: Structured Serialization
  + depends-on: Templated Human Rendering
- User-Defined Template Output — accept a caller-supplied template file as the output format, rendered through the same template mechanism (future — lower priority per developer)
  + depends-on: Templated Human Rendering

## Self-Contained Distribution
> Problem: Runtime-Dependent Distribution — the CLI can't run without a separately-installed runtime, so it won't run where operators need it (affects: Practitioner, AI agent, Maintainer)

- Self-Contained Executable Build — cross-compile a single dependency-free binary per supported platform (macOS amd64/arm64, Linux amd64/arm64) that runs with no separately-installed runtime (CONSTITUTION XII)
- Version Embedding — inject the release version at build time, falling back to Go module build info for `go install` source builds, so `--version` always reports a meaningful version
  + depends-on: Self-Contained Executable Build
- Automated Release Pipeline — on a version tag, build the platform binaries, package them as archives with a checksums file, and publish to GitHub Releases — automated, entirely within this repository
  + depends-on: Self-Contained Executable Build
- Install Script — a POSIX one-liner (hosted in this repo) that detects OS/arch, downloads the matching archive from Releases, verifies its checksum, and installs onto PATH; the primary path for Linux (and macOS) laptops and CI
  + depends-on: Automated Release Pipeline
- Homebrew Tap — a GoReleaser-published Homebrew cask committed within this repository (no separate tap repo), so macOS and Linux users can `brew install` / `brew upgrade`
  + depends-on: Automated Release Pipeline
- NPM Wrapper Package — an npm package that resolves and installs the correct platform binary (platform-specific optional dependencies with a postinstall fallback), so Node-based agent environments can `npx` / `npm i -g`
  + depends-on: Automated Release Pipeline

## GitHub Actions Pipeline
> Problem: No Automated Pipeline — without CI/CD, every change is linted, tested, triaged, and released by hand, so regressions and inconsistencies can reach main and releases go out unguarded (affects: Maintainer)

- PR Validation — on pull request, run lint and the test suite so changes are verified before merge
- PR Administration — auto-apply administrative labels to pull requests for triage and release-note categorization
- Main-Branch Verification — on merge to main, re-run the test suite as a post-merge safety check
- Release Drafting — on merge to main, maintain a draft GitHub release with a label-driven semver bump and accumulated notes (release-drafter; not published); adjacent to Self-Contained Distribution's Automated Release Pipeline, which consumes the published tag to build and publish binaries
  + depends-on: PR Administration
