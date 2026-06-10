# Deprecations

Retired decisions, patterns, and features. Each entry records what was superseded, what replaced it, and where it originated. Managed by the deprecate skill via memory-protocol.md.

---

- [decision] [ASSUMED] Request Authentication ↔ Connection Configuration composition seam — how 007 obtains the credential (re-resolve from Discovery vs. read from the connection context) → connection context is the single resolution point; 010 replays it into 007's AuthTransport (from 007-request-authentication, 2026-06-06)
  Superseded by 009-connection-context-assembly ADR-2: ConnectionContext carries the resolved credential, and Request Execution (010) wires AuthTransport with a resolver that replays the context's cached credential rather than re-resolving from Discovery — so 007 reads the token from the context. Resolves the seam 007 flagged and 008 carried open. (The package-name half of 007's [ASSUMED] seam was already fixed by 008: internal/apiclient.)

- [decision] Undecodable 2xx body → `RuntimeError` (exit 1, internal error) → `APIError` (exit 3, general API error) (from 011-identity-read, 2026-06-10)
  Superseded by 031-diagnostic-normalization ADR-2 (developer-confirmed clarification): a `*apiclient.DecodeError` — a 2xx response whose body would not decode — is an API-exchange problem (the API answered in an unreadable shape), not a CLI-internal fault, so it now classifies `APIError`/3. The cause/next-step wording ("the API response did not match the expected shape — … report it") is unchanged. Render-template failures (019, `*RenderError`) stay `RuntimeError`/1, so the split sharpens "the API's fault" (3) vs. "our fault" (1) rather than blurring it. Originated as the `classifyClientError` decode arm in 011 and was echoed by 012/013/014's "undecodable 2xx → RuntimeError(1)" scenarios and by the decode→1 rows in 011/012/014/025/026/034 interface-cli.md and docs/reference/*.md — those prose references remain at the old wording and should be reconciled when those features are next touched or their docs regenerated (see LEARNINGS 2026-06-10).
