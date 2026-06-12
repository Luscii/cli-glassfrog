package cli

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/Luscii/cli-glassfrog/internal/apiclient"
	"github.com/Luscii/cli-glassfrog/internal/output"
	"github.com/Luscii/cli-glassfrog/internal/rcfile"
	"github.com/Luscii/cli-glassfrog/internal/render"
)

// Diagnostic is the one normalized shape every failure family collapses into
// (spec: a cause, a category, and — where one exists — a next step). It is a
// pure value carrying exactly the three observable fields and no implementation
// detail: 004's ExitCode reads Category to map the process code, and 032
// (Output-Aware Failure Rendering) reads the whole value to render a failure in
// any --output format. Diagnose is its single producer.
type Diagnostic struct {
	// Category is the code-free Outcome category drawn from exitcode.go's fixed
	// taxonomy (UsageError, APIError, PermissionError, RateLimited,
	// NetworkUnavailable, RuntimeError). Diagnose never emits the exit code.
	Category Outcome
	// Cause is the human-meaningful, token-free explanation of what went wrong.
	// Always non-empty for a real failure; empty only for the nil/Success input.
	Cause string
	// NextStep is the recovery the caller can take, or "" when no reliable next
	// step exists. "" is the single, unambiguous "no next step" signal — there is
	// no separate presence flag (interface-spec).
	NextStep string
}

// Diagnose is the single, total normalizer for an API-client error: one
// errors.As chain matches the failure family and sets all three Diagnostic
// fields from the SAME matched value, so the category and the message are
// computed together and can never drift (the hazard the split
// classifyClientError / formatClientErrorMessage / clientErrorNextStep chains
// warned about — now one value). It never panics, never inspects an exit code,
// and never echoes the X-Auth-Token; an unrecognized error returns the
// RuntimeError fail-safe with the verbatim (token-free) error string and no
// next step (CONSTITUTION III — a failure can never become Success).
//
// It expects the error already refined by refineClientError for the API-detail
// arms (as reportClientError does), but an unrefined *ResponseError still
// classifies correctly via the status arm with the status-fallback cause.
//
// Discrimination order matters and mirrors the pre-consolidation chains:
// *AuthError is matched BEFORE *TransportError (007's fail-safe surfaces as a Do
// error and must not be mislabelled transport) and BEFORE the rcfile arms (a
// CredentialError wraps an rcfile error via Unwrap, so an unmatched rcfile arm
// would otherwise catch it as a base-URL UsageError). The refined *ProblemError
// arm precedes the bare *ResponseError arm because a *ProblemError wraps (Unwrap
// → *ResponseError); both branch on the same status, so the category is
// identical and only the cause's richness differs.
func Diagnose(err error) Diagnostic {
	if err == nil {
		return Diagnostic{Category: Success}
	}

	var authErr *apiclient.AuthError
	if errors.As(err, &authErr) {
		if authErr.Kind == apiclient.NoCredentials {
			return Diagnostic{
				Category: UsageError,
				Cause:    "not authenticated",
				NextStep: "run `glassfrog auth login` or set GLASSFROG_TOKEN",
			}
		}
		// CredentialError: a credentials file exists but could not be read/parsed.
		// The cause names the file, never the token.
		return Diagnostic{
			Category: RuntimeError,
			Cause:    authErr.Error(),
			NextStep: "fix or re-create the credentials file with `glassfrog auth login`",
		}
	}

	var transportErr *apiclient.TransportError
	if errors.As(err, &transportErr) {
		return Diagnostic{
			Category: NetworkUnavailable,
			Cause:    transportErr.Error(),
			NextStep: "check connectivity; the API may be unreachable",
		}
	}

	// The refined non-2xx (015 ADR-4). reportClientError refines a *ResponseError
	// into a *ProblemError before calling here, so the cause can surface the API's
	// own detail. The category comes from the status switch in this one arm — the
	// switch's default IS the generic-APIError bucket, so a 401/403/429 can never
	// fall through to it. Every field is response-side (status/detail/title), never
	// the X-Auth-Token.
	var problemErr *apiclient.ProblemError
	if errors.As(err, &problemErr) {
		return Diagnostic{
			Category: categoryForStatus(problemErr.StatusCode),
			Cause:    problemCause(problemErr),
			NextStep: nextStepForStatus(problemErr.StatusCode),
		}
	}
	var responseErr *apiclient.ResponseError
	if errors.As(err, &responseErr) {
		// Defensive fallback for an unrefined *ResponseError: name the status only.
		return Diagnostic{
			Category: categoryForStatus(responseErr.StatusCode),
			Cause:    fmt.Sprintf("the API returned a non-2xx response: status %d", responseErr.StatusCode),
			NextStep: nextStepForStatus(responseErr.StatusCode),
		}
	}

	var decodeErr *apiclient.DecodeError
	if errors.As(err, &decodeErr) {
		// A 2xx body that would not decode: the call succeeded at the wire but the
		// API returned an unreadable shape — an API-exchange problem (APIError → exit
		// 3), not a CLI-internal fault (031 ADR-2, superseding the prior decode →
		// RuntimeError(1) precedent; render-template failures stay RuntimeError(1)).
		// The cause/next-step wording is unchanged: name the shape mismatch and the
		// next step (report it). The underlying parse error is kept (path/cause only,
		// never the token) for diagnostics.
		return Diagnostic{
			Category: APIError,
			Cause:    "the API response did not match the expected shape",
			NextStep: fmt.Sprintf("this may be an API change; report it (%s)", decodeErr.Error()),
		}
	}

	// Base-URL configuration error from client construction: a malformed configured
	// value (*BaseURLError) or an unreadable/malformed .glassfrogrc the base-URL
	// resolver surfaced (*rcfile.ReadError / *rcfile.FormatError). All correctable
	// operator input → UsageError, naming the source + correction step. These sit
	// AFTER the AuthError arm so a credential-file rcfile error (wrapped in
	// *AuthError) keeps its credentials-file hint.
	var baseURLErr *apiclient.BaseURLError
	var rcReadErr *rcfile.ReadError
	var rcFormatErr *rcfile.FormatError
	if errors.As(err, &baseURLErr) || errors.As(err, &rcReadErr) || errors.As(err, &rcFormatErr) {
		return Diagnostic{
			Category: UsageError,
			Cause:    err.Error(),
			NextStep: "correct --base-url, GLASSFROG_BASE_URL, or the .glassfrogrc base_url",
		}
	}

	// Output Format Selection (020): a present-but-invalid --output / GLASSFROG_OUTPUT
	// / .glassfrogrc output value, surfaced as *output.FormatError before any request.
	// A correctable selection input → UsageError. The FormatError's own text already
	// names the source + value + supported list, so it stands as the cause verbatim.
	var formatErr *output.FormatError
	if errors.As(err, &formatErr) {
		return Diagnostic{Category: UsageError, Cause: err.Error()}
	}

	// User-Defined Template Output (035): a caller template that fails to parse
	// (pre-request) or to execute (an unguarded reference to an absent field/key
	// under missingkey=error, post-response) is the operator's own input — a usage
	// error, symmetric with the *output.FormatError arm above. Distinct from a
	// built-in *render.RenderError, which stays a code defect (RuntimeError(1), the
	// fail-safe below). The error's own text names the source + cause (token-free).
	var userTmplErr *render.UserTemplateError
	if errors.As(err, &userTmplErr) {
		return Diagnostic{Category: UsageError, Cause: err.Error()}
	}

	// Fail-safe: an unrecognized error is never silently a success. Surface its
	// verbatim (token-free by the apiclient contract) message with no next step;
	// Exit-Code Convention maps RuntimeError to the catch-all internal-error code 1
	// (CONSTITUTION III).
	return Diagnostic{Category: RuntimeError, Cause: err.Error()}
}

// categoryForStatus maps a non-2xx HTTP status to its code-free category: 401/403
// → PermissionError (code 4), 429 → RateLimited (code 5), everything else →
// APIError (the generic non-2xx, code 3). The default IS the generic bucket, so a
// 401/403/429 can never fall through to it (API Error Extraction 015, ADR-3).
func categoryForStatus(status int) Outcome {
	switch status {
	case http.StatusUnauthorized, http.StatusForbidden:
		return PermissionError
	case http.StatusTooManyRequests:
		return RateLimited
	default:
		return APIError
	}
}

// nextStepForStatus returns the per-class next-step hint for a non-2xx status
// (interface-spec Error Communication / CONSTITUTION II "…and the next step").
// The hint is status-derived only — it never echoes the token.
//
// 031 refines the permission and rate-limit hints (ADR-2 / interface-spec
// Next-step contract): 401 (an authentication failure) points at the configured
// token, while 403 (an authorization failure) points at the identity's role
// membership / permission — the two were previously a single combined hint. A
// 429 points at the rate-limit window resetting (the Retry-After /
// X-RateLimit-Reset headers already on the wrapped *ResponseError) rather than a
// bare "retry later".
func nextStepForStatus(status int) string {
	switch status {
	case http.StatusUnauthorized:
		return "verify the configured API token"
	case http.StatusForbidden:
		return "check that the configured identity has the required role membership / permission"
	case http.StatusTooManyRequests:
		return "wait for the rate-limit window to reset (per the `Retry-After` / `X-RateLimit-Reset` headers) and retry"
	default:
		return "the API rejected the read; check that the token has access and retry, or consult the status code"
	}
}

// problemCause renders the token-free cause for a refined non-2xx *ProblemError.
// It keys on DetailSynthesized (the provenance marker), NOT on Detail emptiness —
// the fallback always fills Detail. A synthesized detail keeps the generic
// "status N" wording (no synthesized text is presented as the API's own words);
// otherwise the API's own detail surfaces (prefixed with the title when the title
// adds context). All text is response-side (status/detail/title), never the token.
func problemCause(problemErr *apiclient.ProblemError) string {
	if problemErr.DetailSynthesized {
		return fmt.Sprintf("the API returned a non-2xx response: status %d", problemErr.StatusCode)
	}
	cause := problemErr.Detail
	if title := strings.TrimSpace(problemErr.Title); title != "" && title != problemErr.Detail {
		cause = fmt.Sprintf("%s: %s", title, problemErr.Detail)
	}
	return fmt.Sprintf("the API returned a non-2xx response: status %d: %s", problemErr.StatusCode, cause)
}

// renderDiagnostic composes the human-facing stderr line from a Diagnostic:
// the Cause alone when there is no next step, else "Cause — NextStep". This
// reproduces the pre-consolidation formatClientErrorMessage output exactly for
// every unchanged family (every current message was "<cause> — <next step>"),
// so the stderr surface does not drift. (032 will render the structured
// Diagnostic per --output instead of calling this; renderDiagnostic is the
// human-format fallback until then.)
func renderDiagnostic(d Diagnostic) string {
	if d.NextStep == "" {
		return d.Cause
	}
	return d.Cause + " — " + d.NextStep
}
