package cli

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/Luscii/cli-glassfrog/internal/apiclient"
	"github.com/Luscii/cli-glassfrog/internal/glassfrog"
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
	// Feature is the recognized gating feature's display name (e.g. "Premium
	// async proposals") when this failure is a recognized plan-limit 403, or ""
	// otherwise (061 ADR-3). Set once here by Diagnose; both renderers read it —
	// the human line carries it inside Cause prose, the structured envelope
	// surfaces it as its own parseable element (errorEnvelopeFor). It never
	// changes the Category or the exit code.
	Feature string
	// ProposalID is the created draft's prp_ id on an invalid-create failure
	// (078), or "" otherwise — the Feature pattern above, applied to the second
	// family that carries a fact of its own. Never withheld on that failure: it
	// is the handle the caller needs to find what the dead write left behind.
	ProposalID string
	// ValidationAlerts are the alerts the server attached to the invalid draft
	// (078), in the server's own order, or nil for every other failure. They are
	// carried as their own element rather than folded into Cause: Cause becomes
	// the envelope's `message`, and enumerating them there would duplicate the
	// dedicated `validation_alerts` key. A nil/empty slice is the honest "the
	// server attached none" — an invalid draft with no alerts still fails.
	ValidationAlerts []glassfrog.ValidationAlert
}

// invalidCreateError is the typed failure value `glassfrog proposal create`
// raises when the server accepted the create and the read-back reported the
// created draft not valid (078 ADR-2). It is deliberately an error for a
// COMPLETED exchange — both the POST and the read-back succeeded — so that the
// failure travels 031's single Diagnose chain and 032's single format-aware
// render chokepoint instead of growing a second classification site. That
// oddity is confined to this seam; nothing outside internal/cli sees it.
//
// Error() is token-free (it names only the created id, which is response-side),
// so the fail-safe arm at the bottom of Diagnose would still surface a safe
// message if this type's own arm were ever removed.
type invalidCreateError struct {
	ProposalID string
	Alerts     []glassfrog.ValidationAlert
}

func (e *invalidCreateError) Error() string {
	return fmt.Sprintf("the created proposal %s is not valid", e.ProposalID)
}

// invalidCreateCause names the verdict and its provenance (078 ADR-5): the
// server accepted the write, and the not-valid verdict came from reading the
// created draft back — not from the create's own response. Token-free, and it
// deliberately does NOT enumerate the alerts; those are their own element.
func invalidCreateCause(id string) string {
	return fmt.Sprintf("the server accepted the create but reports proposal %s not valid (read back after the create)", id)
}

// invalidCreateNextStep is the remedy (078 ADR-5), built around what the CLI can
// and cannot do: there is no draft-discard command, so it names the web UI for
// cleanup rather than a command the caller cannot run, and it points at the
// grammar reference for the accepted-but-invalid shapes that are documented.
func invalidCreateNextStep() string {
	return `review the alerts, check "glassfrog proposal grammar" for documented invalid shapes, and create a corrected proposal from the same tension; the invalid draft can be deleted in the GlassFrog web UI`
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
//
// The *invalidCreateError arm (078) sits first because its placement is free: the
// type wraps nothing and is wrapped by nothing, so no other arm can shadow it and
// it can shadow no other arm. It reads first because it is the one family that is
// not an exchange failure at all.
func Diagnose(err error) Diagnostic {
	if err == nil {
		return Diagnostic{Category: Success}
	}

	var invalidCreateErr *invalidCreateError
	if errors.As(err, &invalidCreateErr) {
		return Diagnostic{
			Category:         InvalidCreate,
			Cause:            invalidCreateCause(invalidCreateErr.ProposalID),
			NextStep:         invalidCreateNextStep(),
			ProposalID:       invalidCreateErr.ProposalID,
			ValidationAlerts: invalidCreateErr.Alerts,
		}
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
		diag := Diagnostic{
			Category: categoryForStatus(problemErr.StatusCode),
			Cause:    problemCause(problemErr),
			NextStep: nextStepForStatus(problemErr.StatusCode),
		}
		// Plan-Limit Signal (061): a 403 from a known plan-gated operation is
		// refined to possibility-framed plan-limit wording naming the gate. Reach
		// the wrapped *ResponseError (the same unwrap path errorEnvelopeFor uses)
		// for the request's method/path and let Feature-Gate Recognition (060)
		// classify the operation. The Category stays PermissionError and the exit
		// code is unchanged (ADR-2); a GateNone result — a non-gated 403, or no
		// reachable *ResponseError — leaves the generic permission diagnostic
		// above exactly as it is today. categoryForStatus/nextStepForStatus are
		// untouched: this refines only Cause/NextStep/Feature. A recognized gate
		// with an empty display name (a future gate registered without a
		// featureGateDisplayName entry — which TestFeatureGateDisplayName_Exhaustive
		// guards against) also falls back to the generic diagnostic rather than
		// rendering broken "requires the  feature" prose (defense-in-depth).
		if problemErr.StatusCode == http.StatusForbidden {
			var re *apiclient.ResponseError
			if errors.As(err, &re) {
				if gate := apiclient.RecognizeFeatureGate(re.Method, re.Path, problemErr.StatusCode); gate != apiclient.GateNone {
					if name := featureGateDisplayName(gate); name != "" {
						diag.Feature = name
						diag.Cause = planLimitCause(name)
						diag.NextStep = planLimitNextStep(name)
					}
				}
			}
		}
		return diag
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
// → PermissionError (code 4), 429 → RateLimited (code 5), 412 → StaleWrite (code
// 7), everything else → APIError (the generic non-2xx, code 3). The default IS
// the generic bucket, so a 401/403/412/429 can never fall through to it (API
// Error Extraction 015, ADR-3; Stale-Write Surfacing 054, ADR-1). Classification
// is status-driven only — the 412 maps to StaleWrite regardless of the command,
// the resource, or whether an If-Match header was sent.
func categoryForStatus(status int) Outcome {
	switch status {
	case http.StatusUnauthorized, http.StatusForbidden:
		return PermissionError
	case http.StatusTooManyRequests:
		return RateLimited
	case http.StatusPreconditionFailed:
		return StaleWrite
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
	case http.StatusPreconditionFailed:
		// 412: a guarded write was refused because the resource changed since it was
		// read. Point the operator at the recovery — re-read for the current version,
		// then retry — not the generic "check that the token has access" step, which
		// is wrong for a stale write (Stale-Write Surfacing 054, ADR-2).
		return "re-read the resource for its current version, then retry the write"
	default:
		return "the API rejected the read; check that the token has access and retry, or consult the status code"
	}
}

// featureGateDisplayName maps a recognized apiclient.Gate kind to its
// human-prose display name for the operator-facing plan-limit diagnostic (061
// ADR-3). It is total: every Gate kind has a case. This is DISTINCT from 060's
// Gate.String() (kebab-case, for logs/%v) — the operator diagnostic uses the
// human-prose form. GateAIIntegration is mapped for readiness even though no
// command reaches it today (060 ADR-3); GateNone maps to "" (no gate). 061 owns
// this name and the wording composed from it; 060 stays a code-free classifier.
//
// The exhaustiveness guard test (diagnostic_test.go, LEARNINGS PR #10 shape)
// fails loud if a new non-None Gate kind is added without a display name here,
// rather than letting it silently fall through to "".
func featureGateDisplayName(g apiclient.Gate) string {
	switch g {
	case apiclient.GateNone:
		return ""
	case apiclient.GatePremiumAsyncProposals:
		return "Premium async proposals"
	case apiclient.GateAIIntegration:
		return "AI Integration"
	default:
		return ""
	}
}

// planLimitCause composes the possibility-framed plan-limit cause naming the
// gating feature (061 ADR-3 / spec decision 1A). It states the operation *may
// not be* available on the organization's plan and notes the same 403 may
// instead be a permission issue — never asserting the plan is certainly
// insufficient. It invents no plan name, price, or upgrade URL; the display name
// (from the recognized gate) is the only added specific (CONSTITUTION VIII).
func planLimitCause(feature string) string {
	return fmt.Sprintf("this operation requires the %s feature, which your organization's plan may not include; a 403 may instead mean your identity lacks permission", feature)
}

// planLimitNextStep composes the possibility-framed next step: a verifiable
// action, never an instruction to upgrade as the sole remedy (061 ADR-3 / 060
// ADR-4 / CONSTITUTION VIII).
func planLimitNextStep(feature string) string {
	return fmt.Sprintf("verify your organization's plan includes %s, or that your identity has permission for this operation", feature)
}

// problemCause renders the token-free cause for a refined non-2xx *ProblemError.
// It keys on DetailSynthesized (the provenance marker), NOT on Detail emptiness —
// the fallback always fills Detail. A synthesized detail keeps the generic
// "status N" wording (no synthesized text is presented as the API's own words);
// otherwise the API's own detail surfaces (prefixed with the title when the title
// adds context). All text is response-side (status/detail/title), never the token.
func problemCause(problemErr *apiclient.ProblemError) string {
	if problemErr.DetailSynthesized {
		if problemErr.StatusCode == http.StatusPreconditionFailed {
			// 412 with no readable API detail: derive a stale-write cause from the
			// status rather than the bare generic "status 412" line — name the
			// precondition failure / changed-since-read the status defines. Status-
			// derived, never invented (Stale-Write Surfacing 054, ADR-2; CONSTITUTION
			// VIII). When the API DID supply a detail/title, the non-synthesized path
			// below surfaces it verbatim — the API's own words win.
			return fmt.Sprintf("the guarded write was refused: the resource changed since it was read (precondition failed, status %d)", problemErr.StatusCode)
		}
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
//
// Invalid-Create Outcome (078) adds the one multi-line shape: when the Diagnostic
// carries validation alerts, each renders on its own two-space-indented
// "<severity> <path>: <message>" line between the cause and the next step
// (interface-cli § "stderr — human formats"), in the server's order. Keeping them
// out of Cause is what stops them being duplicated into the machine envelope's
// `message`, which is Cause verbatim. A failure carrying no alerts — which is
// every other family, and an invalid draft the server attached none to — renders
// the single-line "cause — next step" form exactly as before: no alert block, no
// stray blank line.
//
// A strings.Builder rather than repeated concatenation: the alert count is
// server-controlled and unbounded, and `line += …` per alert reallocates, so the
// copying is quadratic in the number of alerts. It also reads better — one exit
// instead of three.
func renderDiagnostic(d Diagnostic) string {
	var b strings.Builder
	b.WriteString(d.Cause)
	for _, a := range d.ValidationAlerts {
		fmt.Fprintf(&b, "\n  %s %s: %s", a.Severity, a.Path, a.Message)
	}
	if d.NextStep != "" {
		if len(d.ValidationAlerts) > 0 {
			// The alert block ends the line it is on, so the next step opens one of
			// its own. The " — " separator is kept either way, so the cause/next-step
			// pair stays the same recognizable shape it has in every other failure.
			b.WriteString("\n — ")
		} else {
			b.WriteString(" — ")
		}
		b.WriteString(d.NextStep)
	}
	return b.String()
}
