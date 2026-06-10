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
// fallback until it is given its own case here. The exhaustiveness test
// (errorenvelope_test.go) fails loudly when a new Outcome is added without a
// token, so the default is a safety net, not a silent catch-all (LEARNINGS PR #10).
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
	default:
		return "runtime"
	}
}

// errorEnvelopeFor maps a refined failure onto 018's unified error envelope (032
// ADR-2/ADR-4). It is pure: it reads the failure, it neither writes nor renders.
//
//   - Message  ← Diagnose(err).Cause      (the token-free human cause, 031)
//   - NextStep ← Diagnose(err).NextStep   (omitted when empty — the internal-error
//     fallback and bare general-API errors carry none)
//   - Kind     ← kind(Diagnose(err).Category)
//   - Status   ← the wrapped *apiclient.ResponseError's StatusCode, when present
//   - Body     ← that response's raw body, but ONLY when it is valid JSON; a
//     non-JSON body is omitted so it can never fail the whole structured render
//     (RenderError nests Body as structured data and errors on non-JSON — ADR-4).
//     The diagnostic is more valuable than a malformed upstream body.
//
// The caller refines err once before calling (reportFailure does), so the wrapped
// *ResponseError is reachable via errors.As whether err is a bare *ResponseError or
// the refined *ProblemError that unwraps to it — the body survives the refinement.
// This helper lives in internal/cli, the only package importing both
// internal/output (the envelope) and internal/apiclient (the wrapped response), so
// internal/output stays transport-free (018 invariant).
//
// It never reads or emits the X-Auth-Token: every field is response-side
// (cause/next-step/status/body), and Diagnose is already token-free — the envelope
// is secret-free by construction.
func errorEnvelopeFor(err error) output.ErrorEnvelope {
	d := Diagnose(err)
	detail := output.ErrorDetail{
		Message:  d.Cause,
		NextStep: d.NextStep,
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
