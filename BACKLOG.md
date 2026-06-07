# Backlog

> Generated: 2026-06-07T21:59:25 | Framework: MoSCoW | Items: 31

### 1. Command Registration

- **Score**: MoSCoW Must Have
- **Framework**: MoSCoW (Must Have)
- **Rationale**: Dependency root of the CLI Skeleton — every other capability requires it, so it builds first. Highest leverage at zero dependency cost.
- **Status**: specified:001-command-registration

### 2. Argument Dispatch

- **Score**: MoSCoW Must Have
- **Framework**: MoSCoW (Must Have)
- **Rationale**: Builds on Command Registration and sits on the critical path to Exit-Code Convention, so it precedes the parallelizable Help & Version.
- **Dependencies**: → requires: Command Registration
- **Status**: specified:002-argument-dispatch

### 3. Help & Version

- **Score**: MoSCoW Must Have
- **Framework**: MoSCoW (Must Have)
- **Rationale**: Builds on Command Registration but is off the critical path (nothing depends on it), so it follows Argument Dispatch; the two are parallelizable.
- **Dependencies**: → requires: Command Registration
- **Status**: specified:003-help-and-version

### 4. Exit-Code Convention

- **Score**: MoSCoW Must Have
- **Framework**: MoSCoW (Must Have)
- **Rationale**: End of the dependency chain — needs Argument Dispatch in place to emit standardized exit codes, so it builds last.
- **Dependencies**: → requires: Argument Dispatch
- **Status**: specified:004-exit-code-convention

### 5. Credential Discovery

- **Score**: MoSCoW Must Have
- **Framework**: MoSCoW (Must Have)
- **Rationale**: Dependency root of Token Authentication — Request Authentication can't resolve a token without it, so it builds first in the auth solution. Builds on the CLI Skeleton (not a declared FEATURE-MODEL edge).
- **Status**: specified:005-credential-discovery

### 6. Credential Storage

- **Score**: MoSCoW Must Have
- **Framework**: MoSCoW (Must Have)
- **Rationale**: Independent of Credential Discovery (writes the file Discovery reads); off the critical path within the solution, so it parallelizes with #5. The login command builds on the CLI Skeleton (not a declared FEATURE-MODEL edge).
- **Status**: specified:006-credential-storage

### 7. Request Authentication

- **Score**: MoSCoW Must Have
- **Framework**: MoSCoW (Must Have)
- **Rationale**: Proves identity on each API call; builds after Credential Discovery, which it consumes the resolved token from. Gates every self-service read (#11–#14) and Request Execution.
- **Dependencies**: → requires: Credential Discovery
- **Status**: specified:007-request-authentication

### 8. Base URL Resolution

- **Score**: MoSCoW Must Have
- **Framework**: MoSCoW (Must Have)
- **Rationale**: Resolves the effective endpoint by precedence (flag > env > config > default) — the base-URL portion of connection setup. Reuses Credential Discovery's `.glassfrogrc` file and nearest-wins walk, but resolves its value independently. Already specified as 008.
- **Status**: specified:008-base-url-resolution

### 9. Connection Context Assembly

- **Score**: MoSCoW Must Have
- **Framework**: MoSCoW (Must Have)
- **Rationale**: Combines the resolved base URL with the discovered token into the single connection context every request uses. Gates Request Execution, so it must be specified before the API client.
- **Dependencies**: → requires: Base URL Resolution; → requires: Credential Discovery
- **Status**: specified:009-connection-context-assembly

### 10. Request Execution

- **Score**: MoSCoW Must Have
- **Framework**: MoSCoW (Must Have)
- **Rationale**: The single seam every endpoint command calls through — sends the authenticated request and returns a parsed response or a typed transport error. Gates all reads, so it leads the API Client and precedes Identity Read.
- **Dependencies**: → requires: Connection Context Assembly; → requires: Request Authentication
- **Status**: specified:010-request-execution

### 11. Identity Read

- **Score**: MoSCoW Must Have
- **Framework**: MoSCoW (Must Have)
- **Rationale**: The smallest end-to-end read that proves the whole chain (`GET /me` → `getMe`); the first call a consumer makes to orient itself, so it leads the read surface. Now gated on Request Execution. Parallelizable with #12–#14 (no inter-read dependency).
- **Dependencies**: → requires: Request Authentication; → requires: Request Execution
- **Status**: specified:011-identity-read

### 12. My Roles

- **Score**: MoSCoW Must Have
- **Framework**: MoSCoW (Must Have)
- **Rationale**: Named in the ROADMAP Now slice ("/me, my roles"), so it follows Identity Read; the chain is already proven, making this an extension. Parallelizable with the other reads.
- **Dependencies**: → requires: Request Authentication; → requires: Request Execution
- **Status**: specified:012-my-roles

### 13. My Actions

- **Score**: MoSCoW Must Have
- **Framework**: MoSCoW (Must Have)
- **Rationale**: Extends the "what's mine" surface beyond the proven slice; no dependency on the other reads, so its position reflects priority, not build order.
- **Dependencies**: → requires: Request Authentication; → requires: Request Execution
- **Status**: specified:013-my-actions

### 14. My Projects

- **Score**: MoSCoW Must Have
- **Framework**: MoSCoW (Must Have)
- **Rationale**: Completes the "what's mine" surface; last of the parallelizable reads by priority, with no inter-read dependency.
- **Dependencies**: → requires: Request Authentication; → requires: Request Execution
- **Status**: specified:014-my-projects

### 15. API Error Extraction

- **Score**: MoSCoW Should Have
- **Framework**: MoSCoW (Should Have)
- **Rationale**: Turns a non-2xx response into a typed error carrying the API's status and detail; pairs with the Next-tier Opaque Failures problem, so it ranks below the Must-Have reads. Not required for the first read slice — Request Execution already surfaces transport and status failures.
- **Dependencies**: → requires: Request Execution
- **Status**: specified:015-api-error-extraction

### 16. Pagination

- **Score**: MoSCoW Should Have
- **Framework**: MoSCoW (Should Have)
- **Rationale**: Follows the API's paging so list commands never silently truncate; pairs with the Next-tier Silent Truncation problem. Not needed for `/me` (a single resource), so it ranks after the read slice.
- **Dependencies**: → requires: Request Execution
- **Status**: specified:016-pagination

### 17. Rate-Limit Handling

- **Score**: MoSCoW Should Have
- **Framework**: MoSCoW (Should Have)
- **Rationale**: Honors the per-org rate limit (429 + rate-limit headers) with backoff; pairs with the Next-tier Getting Throttled problem. Ranks after the Must-Have reads.
- **Dependencies**: → requires: Request Execution
- **Status**: specified:017-rate-limit-handling

### 18. Structured Serialization

- **Score**: MoSCoW Must Have
- **Framework**: MoSCoW (Must Have)
- **Rationale**: JSON/YAML machine-readable output — core for the AI-agent operator (VISION principle 3) and a no-dependency root of the Output Formatting solution, so it leads the fresh Now work.
- **Status**: pending

### 19. Templated Human Rendering

- **Score**: MoSCoW Must Have
- **Framework**: MoSCoW (Must Have)
- **Rationale**: full/compact human-readable output via a template seam; no dependencies and on the critical path to Output Format Selection, so it parallelizes with #18.
- **Status**: pending

### 20. Output Format Selection

- **Score**: MoSCoW Must Have
- **Framework**: MoSCoW (Must Have)
- **Rationale**: The `--output` flag that dispatches to a renderer — builds after both renderers exist, so it follows #18 and #19.
- **Dependencies**: → requires: Structured Serialization; → requires: Templated Human Rendering
- **Status**: pending

### 21. Self-Contained Executable Build

- **Score**: MoSCoW Must Have
- **Framework**: MoSCoW (Must Have)
- **Rationale**: The dependency-free binary (CONSTITUTION XII) and dependency root of Self-Contained Distribution — everything in the solution builds on it, so it leads the distribution work.
- **Status**: pending

### 22. Automated Release Pipeline

- **Score**: MoSCoW Must Have
- **Framework**: MoSCoW (Must Have)
- **Rationale**: The ship mechanism (tag → build → publish to GitHub Releases); gates every acquisition channel, so it follows the build and precedes the install paths.
- **Dependencies**: → requires: Self-Contained Executable Build
- **Status**: pending

### 23. Version Embedding

- **Score**: MoSCoW Must Have
- **Framework**: MoSCoW (Must Have)
- **Rationale**: Injects the release version at build time with a `go install` build-info fallback so `--version` is correct across install methods; a release without it reports "dev", so it's non-negotiable for shipping.
- **Dependencies**: → requires: Self-Contained Executable Build
- **Status**: pending

### 24. Install Script

- **Score**: MoSCoW Should Have
- **Framework**: MoSCoW (Should Have)
- **Rationale**: The primary Linux/macOS/CI acquisition path (detect, download, checksum, install); consumes published release artifacts, so it follows the release pipeline. Should rather than Must — one of several channels.
- **Dependencies**: → requires: Automated Release Pipeline
- **Status**: pending

### 25. PR Validation

- **Score**: MoSCoW Must Have
- **Framework**: MoSCoW (Must Have)
- **Rationale**: The core CI quality gate (lint + tests on pull request) — the heart of the pipeline solution, with no dependencies, so it leads the pipeline work.
- **Status**: pending

### 26. PR Administration

- **Score**: MoSCoW Should Have
- **Framework**: MoSCoW (Should Have)
- **Rationale**: Auto-applies administrative labels to PRs; no dependencies, but its labels feed Release Drafting's semver bump, so it precedes #28.
- **Status**: pending

### 27. Main-Branch Verification

- **Score**: MoSCoW Should Have
- **Framework**: MoSCoW (Should Have)
- **Rationale**: Re-runs the test suite on merge to main as a post-merge safety net; PR Validation already gates pre-merge, so this is important but not the primary gate.
- **Status**: pending

### 28. Release Drafting

- **Score**: MoSCoW Should Have
- **Framework**: MoSCoW (Should Have)
- **Rationale**: Maintains a label-driven draft release on merge; depends on PR Administration's labels for the semver bump. Adjacent to Automated Release Pipeline (which consumes the published tag), but a distinct, separately-buildable stage.
- **Dependencies**: → requires: PR Administration
- **Status**: pending

### 29. User-Defined Template Output

- **Score**: MoSCoW Could Have
- **Framework**: MoSCoW (Could Have)
- **Rationale**: Caller-supplied template files via the same template seam; developer-flagged as future / lower-priority, so it sits among the Could-Haves at the end of the backlog.
- **Dependencies**: → requires: Templated Human Rendering
- **Status**: pending

### 30. Homebrew Tap

- **Score**: MoSCoW Could Have
- **Framework**: MoSCoW (Could Have)
- **Rationale**: An additional acquisition channel beyond the primary install script; consumes published release artifacts. Could Have — convenient but not blocking the tool's reach.
- **Dependencies**: → requires: Automated Release Pipeline
- **Status**: pending

### 31. NPM Wrapper Package

- **Score**: MoSCoW Could Have
- **Framework**: MoSCoW (Could Have)
- **Rationale**: An additional channel for Node-based agent environments (`npx` / `npm i -g`); consumes published release artifacts. Could Have — extends reach but not core to shipping.
- **Dependencies**: → requires: Automated Release Pipeline
- **Status**: pending
