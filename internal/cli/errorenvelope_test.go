package cli

import (
	"strings"
	"testing"

	"github.com/Luscii/cli-glassfrog/internal/apiclient"
	"github.com/Luscii/cli-glassfrog/internal/output"
)

// kind is the 1:1 map from a code-free Outcome to 018's envelope token (032
// ADR-4). The table pins every operational category's token; the exhaustiveness
// guard pins that the table covers every Outcome the failure path can carry, so a
// future category added to the enum without a token here diverges the produced set
// and fails loud rather than silently falling to the "runtime" default (PR #10
// LEARNINGS — the zero-valued / silent-default trap).
func TestKind_Table(t *testing.T) {
	cases := []struct {
		name string
		in   Outcome
		want string
	}{
		// Success never reaches the failure-render path, so it has no token of its
		// own; it shares the internal-error fallback rather than inventing "success".
		{"success-falls-to-runtime", Success, "runtime"},
		{"usage", UsageError, "usage"},
		{"runtime", RuntimeError, "runtime"},
		{"network", NetworkUnavailable, "network"},
		{"api", APIError, "api"},
		{"permission", PermissionError, "permission"},
		{"rate-limit", RateLimited, "rate-limit"},
	}

	covered := map[Outcome]bool{}
	for _, tc := range cases {
		if got := kind(tc.in); got != tc.want {
			t.Errorf("kind(%v) = %q, want %q", tc.in, got, tc.want)
		}
		covered[tc.in] = true
	}

	// Exhaustiveness guard: the table must exercise every Outcome the enum defines.
	// If a future edit adds a category but no row, this list and the covered set
	// diverge and the test fails loud (the comma-ok half names the missing category
	// explicitly rather than passing silently on the "runtime" default).
	allOutcomes := []Outcome{Success, UsageError, RuntimeError, NetworkUnavailable, APIError, PermissionError, RateLimited}
	if len(covered) != len(allOutcomes) {
		t.Errorf("kind table covers %d distinct outcomes, want %d (a category lost or gained coverage)", len(covered), len(allOutcomes))
	}
	for _, o := range allOutcomes {
		if !covered[o] {
			t.Errorf("no row covers %v — that category's kind token is untested", o)
		}
	}
}

// The defensive default: an unmapped/future category renders the internal-error
// token "runtime", never an empty kind (032 ADR-4 — the safe fallback that the
// exhaustiveness guard above forces a real category off of). Outcome(99) stands in
// for an unmapped value.
func TestKind_DefaultArmIsRuntime(t *testing.T) {
	if got := kind(Outcome(99)); got != "runtime" {
		t.Errorf("kind(unmapped) = %q, want %q (safe fallback)", got, "runtime")
	}
}

// errorEnvelopeFor maps a refined failure onto the envelope (032 ADR-2/ADR-4). The
// envelope is rendered to JSON so the assertions cover both the field values and
// the omitempty key-presence contract (a missing field is absent, never null-keyed).
func TestErrorEnvelopeFor(t *testing.T) {
	jsonOf := func(t *testing.T, env output.ErrorEnvelope) string {
		t.Helper()
		doc, err := output.RenderError(output.JSON, env)
		if err != nil {
			t.Fatalf("RenderError: %v", err)
		}
		return string(doc)
	}

	t.Run("next_step present when the diagnostic carries one", func(t *testing.T) {
		env := errorEnvelopeFor(&apiclient.ResponseError{StatusCode: 403, Body: []byte(`{"detail":"Forbidden"}`)})
		if env.Error.NextStep == "" {
			t.Error("NextStep should be populated for a 403 (it carries a permission next step)")
		}
		doc := jsonOf(t, env)
		if !strings.Contains(doc, `"next_step"`) {
			t.Errorf("rendered envelope should carry the next_step key:\n%s", doc)
		}
	})

	t.Run("next_step omitted (not null) for the internal-error fallback", func(t *testing.T) {
		env := errorEnvelopeFor(errSomethingUnexpected())
		if env.Error.NextStep != "" {
			t.Errorf("NextStep = %q, want empty for the internal-error fallback", env.Error.NextStep)
		}
		if env.Error.Kind != "runtime" {
			t.Errorf("Kind = %q, want runtime for the fallback", env.Error.Kind)
		}
		doc := jsonOf(t, env)
		if strings.Contains(doc, "next_step") {
			t.Errorf("the next_step key must be absent (omitempty), not null-keyed:\n%s", doc)
		}
	})

	t.Run("status and body present for a typed API error with a JSON body", func(t *testing.T) {
		body := `{"type":"about:blank","title":"Forbidden","detail":"nope"}`
		env := errorEnvelopeFor(&apiclient.ResponseError{StatusCode: 403, Body: []byte(body)})
		if env.Error.Status != 403 {
			t.Errorf("Status = %d, want 403", env.Error.Status)
		}
		if len(env.Error.Body) == 0 {
			t.Error("Body should carry the valid-JSON API body")
		}
		doc := jsonOf(t, env)
		// The body nests verbatim as structured data, not a quoted string.
		if !strings.Contains(doc, `"title": "Forbidden"`) {
			t.Errorf("rendered body should nest the API body verbatim:\n%s", doc)
		}
	})

	t.Run("body omitted when the API body is not valid JSON, message/kind/status remain", func(t *testing.T) {
		env := errorEnvelopeFor(&apiclient.ResponseError{StatusCode: 500, Body: []byte(`<html>boom</html>`)})
		if len(env.Error.Body) != 0 {
			t.Errorf("Body should be omitted for a non-JSON API body, got %q", env.Error.Body)
		}
		if env.Error.Message == "" || env.Error.Kind == "" || env.Error.Status != 500 {
			t.Errorf("message/kind/status must remain: %+v", env.Error)
		}
		doc := jsonOf(t, env)
		if strings.Contains(doc, "body") {
			t.Errorf("the body key must be absent for a non-JSON body:\n%s", doc)
		}
	})

	t.Run("status and body absent for a transport failure", func(t *testing.T) {
		env := errorEnvelopeFor(&apiclient.TransportError{})
		if env.Error.Status != 0 || len(env.Error.Body) != 0 {
			t.Errorf("a transport failure carries no status/body, got %+v", env.Error)
		}
		if env.Error.Kind != "network" {
			t.Errorf("Kind = %q, want network", env.Error.Kind)
		}
		doc := jsonOf(t, env)
		if strings.Contains(doc, "status") || strings.Contains(doc, "body") {
			t.Errorf("status/body keys must be absent for a transport failure:\n%s", doc)
		}
	})

	t.Run("body survives *ResponseError->*ProblemError refinement (reached via errors.As)", func(t *testing.T) {
		body := `{"detail":"nope"}`
		// Refine to the typed *ProblemError, exactly as reportFailure does before mapping.
		refined := apiclient.ExtractProblem(&apiclient.ResponseError{StatusCode: 403, Body: []byte(body)})
		env := errorEnvelopeFor(refined)
		if env.Error.Status != 403 {
			t.Errorf("Status = %d, want 403 after refinement", env.Error.Status)
		}
		if len(env.Error.Body) == 0 {
			t.Error("Body must survive the refinement — the *ProblemError unwraps to the *ResponseError")
		}
	})
}

// No rendered failure carries the API token, in any family or format. The
// diagnostic is token-free (031) and RenderError adds nothing (018), so the
// envelope is secret-free by construction; this pins that no future mapping edit
// smuggles a token in. The token is asserted absent from the rendered document in
// every structured format for a representative error from each family.
func TestErrorEnvelopeFor_TokenFree(t *testing.T) {
	families := []struct {
		name string
		err  error
	}{
		{"no-credentials", &apiclient.AuthError{Kind: apiclient.NoCredentials}},
		{"credential-error", &apiclient.AuthError{Kind: apiclient.CredentialError}},
		{"transport", &apiclient.TransportError{}},
		{"api-403", &apiclient.ResponseError{StatusCode: 403, Body: []byte(`{"detail":"Forbidden"}`)}},
		{"api-500-nonjson", &apiclient.ResponseError{StatusCode: 500, Body: []byte(`<html>boom</html>`)}},
		{"rate-limited-429", &apiclient.ResponseError{StatusCode: 429, Body: []byte(`{"detail":"slow down"}`)}},
		{"runtime-fallback", errSomethingUnexpected()},
	}
	for _, fam := range families {
		for _, f := range []output.Format{output.JSON, output.YAML} {
			env := errorEnvelopeFor(fam.err)
			doc, err := output.RenderError(f, env)
			if err != nil {
				t.Fatalf("[%s/%v] RenderError: %v", fam.name, f, err)
			}
			if strings.Contains(string(doc), meSecretToken) {
				t.Errorf("[%s/%v] token leaked into the rendered envelope:\n%s", fam.name, f, doc)
			}
		}
	}
}

// errSomethingUnexpected is the unrecognized failure that Diagnose maps to the
// RuntimeError fail-safe with no next step — the internal-error fallback family.
func errSomethingUnexpected() error {
	return &unexpectedTestError{}
}

type unexpectedTestError struct{}

func (e *unexpectedTestError) Error() string { return "something unexpected" }
