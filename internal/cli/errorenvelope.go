package cli

import (
	"encoding/json"
	"errors"

	"github.com/Luscii/cli-glassfrog/internal/apiclient"
	"github.com/Luscii/cli-glassfrog/internal/output"
)

// kind maps a code-free Outcome category onto 018's lowercased envelope `kind`
// token (032 ADR-4). It is a total 1:1 map: every operational category has a
// token, and the defensive default returns "runtime" (the internal-error token)
// so a future Outcome value can never render an empty kind — it falls to the safe
// fallback until it is given its own case here. The table test (errorenvelope_test.go)
// pins each token and must be kept in sync with the Outcome enum: it catches a
// dropped or uncovered token among the outcomes it lists, but — like the sibling
// exitcode_test guard — it cannot by itself detect a brand-new Outcome constant
// absent from its list. The default keeps kind total in that case rather than
// emitting an empty token (a safe fallback, not a silent catch-all — LEARNINGS PR #10).
//
// Success has no token: a Success never reaches the failure-render path, so it
// shares the internal-error fallback rather than inventing a "success" kind.
func kind(o Outcome) string {
	switch o {
	case UsageError:
		return "usage"
	case RuntimeError:
		return "runtime"
	case NetworkUnavailable:
		return "network"
	case APIError:
		return "api"
	case PermissionError:
		return "permission"
	case RateLimited:
		return "rate-limit"
	case StaleWrite:
		return "stale-write"
	default:
		return "runtime"
	}
}

// errorEnvelopeFor maps an already-computed diagnostic onto 018's unified error
// envelope (032 ADR-2/ADR-4). It is pure: it reads the failure, it neither writes
// nor renders. The caller passes the SAME Diagnostic it derives the exit-code
// category from (reportFailure computes Diagnose once and hands it here), so the
// rendered facts and the outcome are structurally guaranteed to come from one
// Diagnose value — they cannot drift, and Diagnose is never run twice on the
// failure path.
//
//   - Message  ← d.Cause      (the token-free human cause, 031)
//   - NextStep ← d.NextStep   (omitted when empty — the internal-error fallback
//     and bare general-API errors carry none)
//   - Feature  ← d.Feature    (the recognized plan-limit gate's display name, 061;
//     omitted when empty — every non-plan-limit failure carries none. Read from
//     the one Diagnostic, never re-recognized here — single classification site)
//   - Kind     ← kind(d.Category)
//   - Status   ← the wrapped *apiclient.ResponseError's StatusCode, when present
//   - Body     ← that response's raw body, but ONLY when it is valid JSON; a
//     non-JSON body is omitted so it can never fail the whole structured render
//     (RenderError nests Body as structured data and errors on non-JSON — ADR-4).
//     The diagnostic is more valuable than a malformed upstream body.
//
// err carries the typed-error chain for the status/body extraction. The caller
// refines err once before computing d (reportFailure does), so the wrapped
// *ResponseError is reachable via errors.As whether err is a bare *ResponseError or
// the refined *ProblemError that unwraps to it — the body survives the refinement.
// This helper lives in internal/cli, the only package importing both
// internal/output (the envelope) and internal/apiclient (the wrapped response), so
// internal/output stays transport-free (018 invariant).
//
// It never reads or emits the X-Auth-Token: every field is response-side
// (cause/next-step/status/body), and the diagnostic is already token-free — the
// envelope is secret-free by construction.
func errorEnvelopeFor(d Diagnostic, err error) output.ErrorEnvelope {
	detail := output.ErrorDetail{
		Message:  d.Cause,
		NextStep: d.NextStep,
		Feature:  d.Feature,
		Kind:     kind(d.Category),
	}
	var re *apiclient.ResponseError
	if errors.As(err, &re) {
		detail.Status = re.StatusCode
		if json.Valid(re.Body) {
			detail.Body = json.RawMessage(re.Body)
		}
	}
	return output.ErrorEnvelope{Error: detail}
}
