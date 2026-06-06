# Deprecations

Retired decisions, patterns, and features. Each entry records what was superseded, what replaced it, and where it originated. Managed by the deprecate skill via memory-protocol.md.

---

- [decision] [ASSUMED] Request Authentication ↔ Connection Configuration composition seam — how 007 obtains the credential (re-resolve from Discovery vs. read from the connection context) → connection context is the single resolution point; 010 replays it into 007's AuthTransport (from 007-request-authentication, 2026-06-06)
  Superseded by 009-connection-context-assembly ADR-2: ConnectionContext carries the resolved credential, and Request Execution (010) wires AuthTransport with a resolver that replays the context's cached credential rather than re-resolving from Discovery — so 007 reads the token from the context. Resolves the seam 007 flagged and 008 carried open. (The package-name half of 007's [ASSUMED] seam was already fixed by 008: internal/apiclient.)
