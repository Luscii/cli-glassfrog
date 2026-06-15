package apiclient

import (
	"net/http"
	"testing"
)

// RecognizeFeatureGate's contract is a table over (method, path, status) triples
// (060 ADR-2/4). Each row pins one recognition path: a gated 403 names its gate,
// a non-403 on a gated op is GateNone, a non-gated op is GateNone, and the
// path-template edge cases (segment count, literal-vs-wildcard, trailing query,
// trailing slash) match exactly on shape. The recognizer reads only the three
// primitives — never a token, never a body — and is total: any input is a clean
// classification, never a panic or error.
func TestRecognizeFeatureGate_Table(t *testing.T) {
	cases := []struct {
		name   string
		method string
		path   string
		status int
		want   Gate
	}{
		// --- the four gated 403s, the {proposal_id} wildcard matching a concrete id ---
		{"create-proposal-403", http.MethodPost, "/proposals", 403, GatePremiumAsyncProposals},
		{"advance-403", http.MethodPost, "/proposals/prp_0123/propose", 403, GatePremiumAsyncProposals},
		{"withdraw-403", http.MethodPost, "/proposals/prp_0123/withdraw", 403, GatePremiumAsyncProposals},
		{"responses-403", http.MethodPost, "/proposals/prp_0123/responses", 403, GatePremiumAsyncProposals},

		// --- a non-403 status on a gated operation is never recognized ---
		{"create-proposal-422", http.MethodPost, "/proposals", 422, GateNone},
		{"create-proposal-412", http.MethodPost, "/proposals", 412, GateNone},
		{"create-proposal-200", http.MethodPost, "/proposals", 200, GateNone},

		// --- a non-gated operation on a 403 is never recognized ---
		{"role-read-403", http.MethodGet, "/roles/role_0123", 403, GateNone},
		{"unrelated-post-403", http.MethodPost, "/tensions", 403, GateNone},

		// --- path-template matching is exact on shape ---
		{"segment-count-too-many", http.MethodPost, "/proposals/a/b/propose", 403, GateNone},
		{"literal-segment-mismatch", http.MethodPost, "/proposals/x/propose", 403, GatePremiumAsyncProposals}, // propose IS a gated literal
		{"literal-segment-wrong-tail", http.MethodPost, "/proposals/x/promote", 403, GateNone},
		{"wrong-method-on-gated-path", http.MethodGet, "/proposals", 403, GateNone},
		{"trailing-query-ignored", http.MethodPost, "/proposals?dry_run=true", 403, GatePremiumAsyncProposals},
		{"trailing-slash", http.MethodPost, "/proposals/prp_0123/propose/", 403, GatePremiumAsyncProposals},

		// --- totality: malformed / unknown inputs yield GateNone, never a panic ---
		{"empty-everything", "", "", 0, GateNone},
		{"unknown-method", "WAT", "/proposals", 403, GateNone},
		{"garbage-path", http.MethodPost, "///", 403, GateNone},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := RecognizeFeatureGate(tc.method, tc.path, tc.status); got != tc.want {
				t.Errorf("RecognizeFeatureGate(%q, %q, %d) = %v, want %v", tc.method, tc.path, tc.status, got, tc.want)
			}
		})
	}
}

// The ai_integration gate kind is MODELED but UNREGISTERED today (060 ADR-3):
// GateAIIntegration must exist in the type, but no registry row may carry it, so
// the modeled-but-unreachable boundary cannot drift silently.
func TestGateAIIntegrationModeledButUnregistered(t *testing.T) {
	// It exists in the type — distinct from GateNone and the Premium gate.
	if GateAIIntegration == GateNone || GateAIIntegration == GatePremiumAsyncProposals {
		t.Fatal("GateAIIntegration must be a distinct modeled gate kind")
	}
	for _, op := range gatedOperations {
		if op.gate == GateAIIntegration {
			t.Errorf("no registry row may carry GateAIIntegration today, but %s %s does", op.method, op.pathTemplate)
		}
	}
}

// A change-detector over the static registry: it asserts the registry holds
// EXACTLY the expected set of gated rows — the expected count AND each expected
// (method, path-template, gate) row present — so a dropped row or a row mutated
// to a zero-valued (GateNone) gate fails loudly. The expected set is built as a
// map keyed by (method, path-template); pairing a length check with a comma-ok
// lookup avoids the map-zero-value trap where a missing key would silently read
// as a zero value (LEARNINGS).
func TestGatedRegistry_ChangeDetector(t *testing.T) {
	expected := map[[2]string]Gate{
		{http.MethodPost, "/proposals"}:                         GatePremiumAsyncProposals,
		{http.MethodPost, "/proposals/{proposal_id}/propose"}:   GatePremiumAsyncProposals,
		{http.MethodPost, "/proposals/{proposal_id}/withdraw"}:  GatePremiumAsyncProposals,
		{http.MethodPost, "/proposals/{proposal_id}/responses"}: GatePremiumAsyncProposals,
	}

	if len(gatedOperations) != len(expected) {
		t.Fatalf("registry has %d rows, want exactly %d", len(gatedOperations), len(expected))
	}

	got := make(map[[2]string]Gate, len(gatedOperations))
	for _, op := range gatedOperations {
		got[[2]string{op.method, op.pathTemplate}] = op.gate
	}
	for key, wantGate := range expected {
		gotGate, ok := got[key]
		if !ok {
			t.Errorf("expected gated row %s %s is missing from the registry", key[0], key[1])
			continue
		}
		if gotGate != wantGate {
			t.Errorf("gated row %s %s carries gate %v, want %v", key[0], key[1], gotGate, wantGate)
		}
	}
}

// pathMatchesTemplate's edge cases, pinned directly (060 ADR-2, Risks):
// segment-count equality, wildcard substitution, literal exactness, query
// stripping, and surrounding-slash tolerance.
func TestPathMatchesTemplate(t *testing.T) {
	cases := []struct {
		name     string
		path     string
		template string
		want     bool
	}{
		{"exact-literal", "/proposals", "/proposals", true},
		{"wildcard-matches-one-segment", "/proposals/prp_0123/propose", "/proposals/{proposal_id}/propose", true},
		{"segment-count-fewer", "/proposals/prp_0123", "/proposals/{proposal_id}/propose", false},
		{"segment-count-more", "/proposals/a/b/propose", "/proposals/{proposal_id}/propose", false},
		{"literal-tail-differs", "/proposals/prp_0123/withdraw", "/proposals/{proposal_id}/propose", false},
		{"wildcard-rejects-empty-segment", "/proposals//propose", "/proposals/{proposal_id}/propose", false},
		{"query-stripped", "/proposals?x=1", "/proposals", true},
		{"trailing-slash-tolerated", "/proposals/", "/proposals", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := pathMatchesTemplate(tc.path, tc.template); got != tc.want {
				t.Errorf("pathMatchesTemplate(%q, %q) = %v, want %v", tc.path, tc.template, got, tc.want)
			}
		})
	}
}
