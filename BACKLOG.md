# Backlog

> Generated: 2026-06-06T09:49:36 | Framework: MoSCoW | Items: 17

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
- **Rationale**: The base-URL half of Connection Resolution — resolves the effective endpoint by precedence (flag > env > config > default). Reuses Credential Discovery's `.glassfrogrc` file and nearest-wins walk, but resolves its value independently. Already specified as 008.
- **Decomposed from**: Connection Resolution
- **Status**: specified:008-base-url-resolution

### 9. Connection Context Assembly

- **Score**: MoSCoW Must Have
- **Framework**: MoSCoW (Must Have)
- **Rationale**: The remaining half of Connection Resolution — combines the resolved base URL with the discovered token into the single connection context every request uses. Gates Request Execution, so it must be specified before the API client.
- **Dependencies**: → requires: Base URL Resolution; → requires: Credential Discovery
- **Decomposed from**: Connection Resolution
- **Status**: pending

### 10. Request Execution

- **Score**: MoSCoW Must Have
- **Framework**: MoSCoW (Must Have)
- **Rationale**: The single seam every endpoint command calls through — sends the authenticated request and returns a parsed response or a typed transport error. Gates all reads, so it leads the API Client and precedes Identity Read. (FEATURE-MODEL declares depends-on Connection Resolution; mapped here to its Connection Context Assembly half after the #8 decomposition.)
- **Dependencies**: → requires: Connection Context Assembly; → requires: Request Authentication
- **Status**: pending

### 11. Identity Read

- **Score**: MoSCoW Must Have
- **Framework**: MoSCoW (Must Have)
- **Rationale**: The smallest end-to-end read that proves the whole chain (`GET /me` → `getMe`); the first call a consumer makes to orient itself, so it leads the read surface. Now gated on Request Execution. Parallelizable with #12–#14 (no inter-read dependency).
- **Dependencies**: → requires: Request Authentication; → requires: Request Execution
- **Status**: pending

### 12. My Roles

- **Score**: MoSCoW Must Have
- **Framework**: MoSCoW (Must Have)
- **Rationale**: Named in the ROADMAP Now slice ("/me, my roles"), so it follows Identity Read; the chain is already proven, making this an extension. Parallelizable with the other reads.
- **Dependencies**: → requires: Request Authentication; → requires: Request Execution
- **Status**: pending

### 13. My Actions

- **Score**: MoSCoW Must Have
- **Framework**: MoSCoW (Must Have)
- **Rationale**: Extends the "what's mine" surface beyond the proven slice; no dependency on the other reads, so its position reflects priority, not build order.
- **Dependencies**: → requires: Request Authentication; → requires: Request Execution
- **Status**: pending

### 14. My Projects

- **Score**: MoSCoW Must Have
- **Framework**: MoSCoW (Must Have)
- **Rationale**: Completes the "what's mine" surface; last of the parallelizable reads by priority, with no inter-read dependency.
- **Dependencies**: → requires: Request Authentication; → requires: Request Execution
- **Status**: pending

### 15. API Error Extraction

- **Score**: MoSCoW Should Have
- **Framework**: MoSCoW (Should Have)
- **Rationale**: Turns a non-2xx response into a typed error carrying the API's status and detail; pairs with the Next-tier Opaque Failures problem, so it ranks below the Must-Have reads. Not required for the first read slice — Request Execution already surfaces transport and status failures.
- **Dependencies**: → requires: Request Execution
- **Status**: pending

### 16. Pagination

- **Score**: MoSCoW Should Have
- **Framework**: MoSCoW (Should Have)
- **Rationale**: Follows the API's paging so list commands never silently truncate; pairs with the Next-tier Silent Truncation problem. Not needed for `/me` (a single resource), so it ranks after the read slice.
- **Dependencies**: → requires: Request Execution
- **Status**: pending

### 17. Rate-Limit Handling

- **Score**: MoSCoW Should Have
- **Framework**: MoSCoW (Should Have)
- **Rationale**: Honors the per-org rate limit (429 + rate-limit headers) with backoff; pairs with the Next-tier Getting Throttled problem. Ranks after the Must-Have reads.
- **Dependencies**: → requires: Request Execution
- **Status**: pending
