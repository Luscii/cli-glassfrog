package apiclient

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// ProblemError is the typed, code-free refinement of 010's generic
// *ResponseError into the API's own cause. It is produced by ExtractProblem
// (never constructed by a consumer) and carries the authoritative HTTP status,
// the RFC 9457 Problem Details members the body carried (type/title/detail), a
// provenance marker telling whether the detail is the API's own or a synthesized
// fallback, and the body's self-described status as metadata.
//
// It WRAPS the originating *ResponseError (Unwrap returns it), so
// errors.As(err, &ResponseError) still matches and the raw body / response
// headers stay reachable through the wrapped value — Request Execution's seam is
// refined, not replaced (ADR-1). Because it wraps, a consumer's error chain must
// match *ProblemError (or branch on the status) BEFORE the bare *ResponseError
// arm, or the wrapping swallows the more-specific type.
//
// It decides no exit code: the HTTP status is carried raw and the consuming
// command (internal/cli) maps it (producer-classifies / consumer-maps —
// 002/004/005/007/008/009/010/011).
type ProblemError struct {
	// StatusCode is the authoritative HTTP status, taken from the wrapped
	// *ResponseError. A disagreeing in-body status member never overrides it
	// (ADR-2); the body's status is captured in BodyStatus as metadata only.
	StatusCode int
	// Type is the RFC 9457 `type` member (a URI). Empty when the body wasn't a
	// parseable Problem Details object or omitted the member.
	Type string
	// Title is the RFC 9457 `title` member (a short summary). Empty when absent.
	Title string
	// Detail is the RFC 9457 `detail` member (the occurrence-specific
	// explanation) when the body parsed and carried a non-blank one; otherwise a
	// status-derived fallback. Always non-empty. Read DetailSynthesized to tell
	// the two apart — emptiness cannot, because the fallback always fills it.
	Detail string
	// DetailSynthesized is the provenance marker: false when Detail is the API's
	// own parsed detail, true when Detail is the status-derived fallback. The
	// consumer keys its message on this flag, not on Detail emptiness.
	DetailSynthesized bool
	// BodyStatus is the body's own RFC 9457 `status` member when the body parsed
	// and carried one; nil otherwise (nil-able so absence is unambiguous). It is
	// metadata only — never authoritative, never overrides StatusCode — present
	// so a consumer can observe a body-vs-HTTP status disagreement.
	BodyStatus *int

	// resp is the wrapped originating *ResponseError, exposed via Unwrap so the
	// raw body / headers stay reachable and errors.As(err, &ResponseError) still
	// matches.
	resp *ResponseError
}

// Error renders a token-free message naming the status and the Detail. Both are
// response-side values (the API's reply), so the request X-Auth-Token can never
// appear here.
func (e *ProblemError) Error() string {
	return fmt.Sprintf("the API returned a non-2xx response: status %d: %s", e.StatusCode, e.Detail)
}

// Unwrap returns the wrapped *ResponseError, so the raw body and response
// headers stay reachable and errors.As(err, &ResponseError) still matches the
// refined error.
func (e *ProblemError) Unwrap() error { return e.resp }

// problemBody mirrors the four standard RFC 9457 Problem Details members. Each
// is a pointer so a member's ABSENCE is distinguishable from its zero value —
// an absent `detail` (→ synthesize) differs from a present empty one, and an
// absent `status` (→ nil BodyStatus) differs from a body status of 0.
type problemBody struct {
	Type   *string `json:"type"`
	Title  *string `json:"title"`
	Status *int    `json:"status"`
	Detail *string `json:"detail"`
}

// ExtractProblem refines the generic non-2xx *ResponseError 010 produced into a
// typed *ProblemError. It is the pure, TOTAL extraction (ADR-1/2): it never
// returns an error and never panics, and it reads only the response-side
// *ResponseError (status, headers, body) — never the request token.
//
// It best-effort-decodes re.Body as an RFC 9457 Problem Details object, NOT
// gated on the response Content-Type (spec Non-Behavior). On a parseable body it
// surfaces the standard members (type/title/detail) and captures the body's own
// `status` as metadata; on an empty, non-JSON, HTML, or member-missing body it
// degrades to a status-derived fallback Detail with DetailSynthesized = true,
// leaving the raw body on the wrapped value. The StatusCode is always the
// authoritative HTTP status from re; a disagreeing in-body status is carried in
// BodyStatus but never promoted (ADR-2).
func ExtractProblem(re *ResponseError) *ProblemError {
	pe := &ProblemError{
		StatusCode: re.StatusCode,
		resp:       re,
	}

	var body problemBody
	if err := json.Unmarshal(re.Body, &body); err == nil {
		if body.Type != nil {
			pe.Type = *body.Type
		}
		if body.Title != nil {
			pe.Title = *body.Title
		}
		if body.Status != nil {
			s := *body.Status
			pe.BodyStatus = &s
		}
		// A present, non-blank `detail` is the API's own cause; a present-but-blank
		// one is degenerate and is treated as absent so the message is never empty.
		if body.Detail != nil && strings.TrimSpace(*body.Detail) != "" {
			pe.Detail = *body.Detail
		}
	}

	if pe.Detail == "" {
		// No usable API detail (empty / non-JSON / HTML / member-missing body):
		// derive the detail from the authoritative HTTP status and mark it
		// synthesized so the consumer renders the fallback wording rather than
		// presenting it as the API's own words (resolves No-Fabricated-Data).
		pe.Detail = statusFallbackDetail(re.StatusCode)
		pe.DetailSynthesized = true
	}

	return pe
}

// statusFallbackDetail derives a human-readable fallback detail from the HTTP
// status. http.StatusText covers the standard codes; a nonstandard status
// (StatusText returns "") still yields a non-empty detail so Detail is always
// usable.
func statusFallbackDetail(status int) string {
	if text := http.StatusText(status); text != "" {
		return text
	}
	return fmt.Sprintf("status %d", status)
}
