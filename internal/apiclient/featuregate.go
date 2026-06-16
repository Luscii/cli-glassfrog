package apiclient

import (
	"net/http"
	"strings"
)

// Gate is a code-free classification naming a plan/feature gate that a known
// gated operation sits behind. It models the gate *kinds* the spec carries,
// independent of how many operations carry each one (060 ADR-3).
//
// A non-None Gate returned by RecognizeFeatureGate is a SUSPICION, never a
// verdict: it means the failed 403 is *consistent with* a known plan gate, not
// that the plan limit is confirmed. The 403 body is not self-identifying, so a
// genuine permission denial on a gated operation is indistinguishable from a
// plan-gate rejection — recognition can only ever say a 403 *may be* a plan
// limit (060 ADR-4). Downstream wording (Plan-Limit Signal, #61) must phrase the
// diagnostic as possibility ("may not be available on your plan"), never as
// certainty.
type Gate int

const (
	// GateNone means no known feature gate was recognized — the operation/status
	// pair does not match a registered gated row. It is the total/pure fallback.
	GateNone Gate = iota
	// GatePremiumAsyncProposals is the gate behind the Premium async-proposal
	// write path (creating, advancing, withdrawing a proposal, and recording a
	// response). These operations reject with 403 on an org without the Premium
	// plan.
	GatePremiumAsyncProposals
	// GateAIIntegration is the gate behind the agent/skill operations carrying the
	// `x-feature-gate: ai_integration` extension. It is MODELED so the recognizer
	// can name it, but no registry row references it today — no CLI command
	// reaches those endpoints yet (060 ADR-3; PROJECT "Deferred"). When such a
	// command lands, recognition extends by adding registry rows, not by changing
	// this type.
	GateAIIntegration
)

// String renders the gate as a stable, legible name (kebab-case, matching the
// package's sibling enums AuthErrorKind and BaseURLSource), so %v in test
// failures, logs, and the downstream #61 diagnostic reads a name rather than a
// bare integer.
func (g Gate) String() string {
	switch g {
	case GateNone:
		return "none"
	case GatePremiumAsyncProposals:
		return "premium-async-proposals"
	case GateAIIntegration:
		return "ai-integration"
	default:
		return "unknown"
	}
}

// gatedOperation is one row of the static gated-operation registry: an HTTP
// method, a path template (where a {…} segment is a single-segment wildcard),
// and the gate that operation sits behind.
type gatedOperation struct {
	method       string
	pathTemplate string
	gate         Gate
}

// gatedOperations is the static gated-operation registry: a literal,
// self-documenting transcription of the spec's gate metadata (060 ADR-2). It
// holds ONLY the four reachable Premium async-proposal writes — recognition keys
// on the operation, not the command, so the withdraw/responses rows are correct
// the moment those commands issue the request even though the commands need not
// exist yet.
//
// Entries are EXPLICIT per-operation rows, never a `POST /proposals*` prefix
// rule, so a future non-gated POST under /proposals is not mis-recognized (060
// ADR-2). GateAIIntegration is deliberately absent (060 ADR-3) — pinned by a
// guard test.
var gatedOperations = []gatedOperation{
	{method: http.MethodPost, pathTemplate: "/proposals", gate: GatePremiumAsyncProposals},
	{method: http.MethodPost, pathTemplate: "/proposals/{proposal_id}/propose", gate: GatePremiumAsyncProposals},
	{method: http.MethodPost, pathTemplate: "/proposals/{proposal_id}/withdraw", gate: GatePremiumAsyncProposals},
	{method: http.MethodPost, pathTemplate: "/proposals/{proposal_id}/responses", gate: GatePremiumAsyncProposals},
}

// RecognizeFeatureGate reports whether a failed operation is consistent with a
// known plan/feature gate. It is PURE and TOTAL: it reads only the supplied
// method/path/status — never the token, never the response body — performs no
// I/O, never panics, and never returns an error.
//
// It returns the registered gate when status == 403 AND the operation
// (method + path) matches a registered row; GateNone otherwise. A non-403 status
// from a gated operation, any status from an unregistered operation, and an
// unknown or malformed method/path/status all yield GateNone.
//
// A non-None result is a SUSPICION, not a verdict — see the Gate type doc. The
// 403 body is not self-identifying (060 ADR-4), so a recognized 403 is only ever
// *consistent with* the named gate, never a confirmed plan limit.
func RecognizeFeatureGate(method, path string, status int) Gate {
	if status != http.StatusForbidden {
		return GateNone
	}
	for _, op := range gatedOperations {
		if op.method == method && pathMatchesTemplate(path, op.pathTemplate) {
			return op.gate
		}
	}
	return GateNone
}

// pathMatchesTemplate reports whether a concrete request path matches a path
// template. It compares segment-wise after stripping any query string: a {…}
// template segment matches any one concrete segment, a literal segment must
// match exactly, and the segment counts must be equal. The query string is
// ignored. It is total — any input yields a clean true/false.
func pathMatchesTemplate(path, template string) bool {
	// A trailing query string never participates in matching.
	if i := strings.IndexByte(path, '?'); i >= 0 {
		path = path[:i]
	}

	pathSegs := strings.Split(strings.Trim(path, "/"), "/")
	tmplSegs := strings.Split(strings.Trim(template, "/"), "/")
	if len(pathSegs) != len(tmplSegs) {
		return false
	}

	for i, tmpl := range tmplSegs {
		if isTemplateSegment(tmpl) {
			// A wildcard matches any one non-empty concrete segment.
			if pathSegs[i] == "" {
				return false
			}
			continue
		}
		if pathSegs[i] != tmpl {
			return false
		}
	}
	return true
}

// isTemplateSegment reports whether a template segment is a {…} placeholder.
func isTemplateSegment(seg string) bool {
	return len(seg) >= 2 && seg[0] == '{' && seg[len(seg)-1] == '}'
}
