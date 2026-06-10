package cli

// classifyClientError turns an API-client error into a code-free Outcome
// category. Since Diagnostic Normalization (031) it is a thin delegate over the
// single Diagnose normalizer — classification, cause, and next step are computed
// together in one errors.As chain (diagnostic.go) so the category and the
// message can never drift. This delegate is retained for the three callers that
// need only the category and render their own message: the me.go output-format
// path (reportFormatResolutionError) and the roles.go / subroles.go
// partial-result paths (reportIncompleteWalk / reportIncompleteSubrolesWalk).
//
// The mapping it exposes (now Diagnose's, unchanged in substance):
//
//   - nil                                   → Success
//   - *apiclient.AuthError{NoCredentials}   → UsageError
//   - *apiclient.AuthError{CredentialError} → RuntimeError
//   - *apiclient.TransportError             → NetworkUnavailable
//   - *apiclient.ProblemError / *apiclient.ResponseError → 401/403 → PermissionError,
//     429 → RateLimited, every other non-2xx → APIError
//   - *apiclient.DecodeError                → APIError (031 ADR-2 — was RuntimeError)
//   - base-URL / rcfile / *output.FormatError → UsageError
//   - anything else                         → RuntimeError (the fail-safe; never Success)
func classifyClientError(err error) Outcome {
	return Diagnose(err).Category
}
