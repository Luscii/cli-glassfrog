package apiclient

import (
	"errors"
	"net/http"
	"strings"
	"testing"
)

// intptr is a small helper for the nil-able BodyStatus expectations.
func intptr(i int) *int { return &i }

// ExtractProblem's contract is a table over crafted *ResponseError values
// (interface-spec Output contract + ADR-2). Each row pins one degradation /
// extraction path; together they cover valid / extension-bearing / empty / HTML
// / member-missing / status-mismatch bodies — the totality and HTTP-authoritative
// disciplines the spec forks resolved.
func TestExtractProblem_Table(t *testing.T) {
	cases := []struct {
		name string
		re   *ResponseError
		// expectations on the produced *ProblemError
		wantStatus      int
		wantType        string
		wantTitle       string
		wantDetail      string
		wantSynthesized bool
		wantBodyStatus  *int
	}{
		{
			name:            "valid-problem-details",
			re:              &ResponseError{StatusCode: 404, Body: []byte(`{"type":"about:blank","title":"Not Found","status":404,"detail":"No such role"}`)},
			wantStatus:      404,
			wantType:        "about:blank",
			wantTitle:       "Not Found",
			wantDetail:      "No such role",
			wantSynthesized: false,
			wantBodyStatus:  intptr(404),
		},
		{
			name:            "404-surfaces-api-detail",
			re:              &ResponseError{StatusCode: 404, Body: []byte(`{"detail":"Not Found"}`)},
			wantStatus:      404,
			wantDetail:      "Not Found",
			wantSynthesized: false,
			wantBodyStatus:  nil,
		},
		{
			name:            "extension-members-not-promoted",
			re:              &ResponseError{StatusCode: 422, Body: []byte(`{"title":"Unprocessable","detail":"validation failed","status":422,"errors":[{"field":"name"}],"trace_id":"abc123"}`)},
			wantStatus:      422,
			wantTitle:       "Unprocessable",
			wantDetail:      "validation failed",
			wantSynthesized: false,
			wantBodyStatus:  intptr(422),
		},
		{
			name:            "empty-body-degrades",
			re:              &ResponseError{StatusCode: 500, Body: []byte(``)},
			wantStatus:      500,
			wantDetail:      http.StatusText(500),
			wantSynthesized: true,
			wantBodyStatus:  nil,
		},
		{
			name:            "html-gateway-body-degrades",
			re:              &ResponseError{StatusCode: 502, Body: []byte(`<html><body>502 Bad Gateway</body></html>`)},
			wantStatus:      502,
			wantDetail:      http.StatusText(502),
			wantSynthesized: true,
			wantBodyStatus:  nil,
		},
		{
			name:            "member-missing-degrades",
			re:              &ResponseError{StatusCode: 400, Body: []byte(`{"foo":"bar"}`)},
			wantStatus:      400,
			wantDetail:      http.StatusText(400),
			wantSynthesized: true,
			wantBodyStatus:  nil,
		},
		{
			name:            "blank-detail-degrades",
			re:              &ResponseError{StatusCode: 503, Body: []byte(`{"detail":"   "}`)},
			wantStatus:      503,
			wantDetail:      http.StatusText(503),
			wantSynthesized: true,
			wantBodyStatus:  nil,
		},
		{
			name:            "body-status-disagrees-http-authoritative",
			re:              &ResponseError{StatusCode: 403, Body: []byte(`{"status":401,"detail":"Forbidden"}`)},
			wantStatus:      403,
			wantDetail:      "Forbidden",
			wantSynthesized: false,
			wantBodyStatus:  intptr(401),
		},
		{
			name:            "nonstandard-status-still-nonempty-detail",
			re:              &ResponseError{StatusCode: 599, Body: []byte(``)},
			wantStatus:      599,
			wantDetail:      "status 599",
			wantSynthesized: true,
			wantBodyStatus:  nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			pe := ExtractProblem(tc.re)
			if pe == nil {
				t.Fatal("ExtractProblem must never return nil")
			}
			if pe.StatusCode != tc.wantStatus {
				t.Errorf("StatusCode = %d, want %d", pe.StatusCode, tc.wantStatus)
			}
			if pe.Type != tc.wantType {
				t.Errorf("Type = %q, want %q", pe.Type, tc.wantType)
			}
			if pe.Title != tc.wantTitle {
				t.Errorf("Title = %q, want %q", pe.Title, tc.wantTitle)
			}
			if pe.Detail != tc.wantDetail {
				t.Errorf("Detail = %q, want %q", pe.Detail, tc.wantDetail)
			}
			if pe.DetailSynthesized != tc.wantSynthesized {
				t.Errorf("DetailSynthesized = %v, want %v", pe.DetailSynthesized, tc.wantSynthesized)
			}
			switch {
			case tc.wantBodyStatus == nil && pe.BodyStatus != nil:
				t.Errorf("BodyStatus = %d, want nil", *pe.BodyStatus)
			case tc.wantBodyStatus != nil && pe.BodyStatus == nil:
				t.Errorf("BodyStatus = nil, want %d", *tc.wantBodyStatus)
			case tc.wantBodyStatus != nil && pe.BodyStatus != nil && *pe.BodyStatus != *tc.wantBodyStatus:
				t.Errorf("BodyStatus = %d, want %d", *pe.BodyStatus, *tc.wantBodyStatus)
			}
		})
	}
}

// The raw body (RFC 9457 extension members included) and the response headers
// stay reachable through the wrapped *ResponseError: errors.As must still match
// it, and Body/Header must be the originals (the seam is refined, not replaced).
func TestExtractProblem_WrapsResponseErrorReachable(t *testing.T) {
	hdr := http.Header{"X-Request-Id": []string{"req-123"}}
	raw := []byte(`{"title":"Unprocessable","detail":"bad","errors":[{"field":"name"}]}`)
	re := &ResponseError{StatusCode: 422, Header: hdr, Body: raw}

	pe := ExtractProblem(re)

	var unwrapped *ResponseError
	if !errors.As(error(pe), &unwrapped) {
		t.Fatal("errors.As(*ProblemError, &ResponseError) should match the wrapped value")
	}
	if string(unwrapped.Body) != string(raw) {
		t.Errorf("wrapped Body = %q, want the raw body %q (extension members must remain available)", unwrapped.Body, raw)
	}
	if unwrapped.Header.Get("X-Request-Id") != "req-123" {
		t.Errorf("wrapped Header lost the response headers: %v", unwrapped.Header)
	}
}

// errors.As must discriminate *ProblemError itself (a consumer branches on it
// before the bare *ResponseError arm).
func TestExtractProblem_DiscriminableAsProblemError(t *testing.T) {
	var err error = ExtractProblem(&ResponseError{StatusCode: 429, Body: []byte(`{"detail":"slow down"}`)})
	var pe *ProblemError
	if !errors.As(err, &pe) {
		t.Fatal("errors.As should discriminate *ProblemError")
	}
	if pe.StatusCode != 429 {
		t.Errorf("StatusCode = %d, want 429", pe.StatusCode)
	}
}

// Secret hygiene: a token-shaped value can only ride the REQUEST header, never
// the response side ExtractProblem reads — so no produced field, the Error()
// string, or the fallback can carry it (mirrors 010's token-never-in-output
// test). Crafted bodies that maliciously embed a token-shaped value are not
// rendered by Error() (it surfaces only status + Detail), and Detail here is the
// synthesized fallback.
func TestExtractProblem_NoTokenInOutput(t *testing.T) {
	const token = "gf_live_secret123"
	// Even if a body were to echo something token-shaped, Error() renders only the
	// status and Detail; on a non-Problem-Details body the Detail is synthesized.
	re := &ResponseError{StatusCode: 500, Body: []byte(`<html>` + token + `</html>`)}
	pe := ExtractProblem(re)
	if strings.Contains(pe.Error(), token) {
		t.Fatalf("the token leaked into ProblemError.Error(): %q", pe.Error())
	}
	if strings.Contains(pe.Detail, token) {
		t.Fatalf("the token leaked into the synthesized Detail: %q", pe.Detail)
	}
}
