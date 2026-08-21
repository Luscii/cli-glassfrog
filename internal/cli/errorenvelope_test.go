package cli

import (
	"strings"
	"testing"

	"github.com/Luscii/cli-glassfrog/internal/apiclient"
	"github.com/Luscii/cli-glassfrog/internal/glassfrog"
	"github.com/Luscii/cli-glassfrog/internal/output"
)

// kind is the 1:1 map from a code-free Outcome to 018's envelope token (032
// ADR-4). The table pins every operational category's token; the guard below pins
// that the table and the allOutcomes list agree on the full set. Both lists are
// manually maintained and must be kept in sync with the Outcome enum: the guard
// catches a dropped or uncovered token (a row removed, or the two lists diverging),
// but it does NOT by itself detect a brand-new Outcome constant absent from both
// lists — kind()'s default keeps that case total instead. The sibling
// exitcode_test guard has the same manual-list shape (PR #10 LEARNINGS — the
// zero-valued / silent-default trap, mitigated here, not eliminated).
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
		{"stale-write", StaleWrite, "stale-write"},
		{"invalid-create", InvalidCreate, "invalid-create"},
	}

	covered := map[Outcome]bool{}
	for _, tc := range cases {
		if got := kind(tc.in); got != tc.want {
			t.Errorf("kind(%v) = %q, want %q", tc.in, got, tc.want)
		}
		covered[tc.in] = true
	}

	// Coverage guard (kept in sync with the Outcome enum by hand): the table must
	// exercise every Outcome named in allOutcomes. A row dropped from the table, or
	// the table and allOutcomes diverging, fails loud (the len check plus the comma-ok
	// half names the missing category). NOTE: allOutcomes is itself manually
	// maintained, so adding a new Outcome constant to the enum without also adding it
	// here will NOT fail this test — that new value would fall to kind()'s "runtime"
	// default. Keep this list current with dispatch.go's Outcome constants.
	allOutcomes := []Outcome{Success, UsageError, RuntimeError, NetworkUnavailable, APIError, PermissionError, RateLimited, StaleWrite, InvalidCreate}
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
// token "runtime", never an empty kind (032 ADR-4 — the safe fallback for any
// Outcome not yet given an explicit token, including a new enum constant the table
// above has not been updated to cover). Outcome(99) stands in for an unmapped value.
func TestKind_DefaultArmIsRuntime(t *testing.T) {
	if got := kind(Outcome(99)); got != "runtime" {
		t.Errorf("kind(unmapped) = %q, want %q (safe fallback)", got, "runtime")
	}
}

// errorEnvelopeFor maps a refined failure onto the envelope (032 ADR-2/ADR-4). The
// envelope is rendered to JSON so the assertions cover both the field values and
// the omitempty key-presence contract (a missing field is absent, never null-keyed).
func TestErrorEnvelopeFor(t *testing.T) {
	// envFor mirrors how reportFailure calls the mapper: it computes the diagnostic
	// once and hands that same value in alongside err (for the *ResponseError
	// status/body extraction).
	envFor := func(err error) output.ErrorEnvelope { return errorEnvelopeFor(Diagnose(err), err) }
	jsonOf := func(t *testing.T, env output.ErrorEnvelope) string {
		t.Helper()
		doc, err := output.RenderError(output.JSON, env)
		if err != nil {
			t.Fatalf("RenderError: %v", err)
		}
		return string(doc)
	}

	t.Run("next_step present when the diagnostic carries one", func(t *testing.T) {
		env := envFor(&apiclient.ResponseError{StatusCode: 403, Body: []byte(`{"detail":"Forbidden"}`)})
		if env.Error.NextStep == "" {
			t.Error("NextStep should be populated for a 403 (it carries a permission next step)")
		}
		doc := jsonOf(t, env)
		if !strings.Contains(doc, `"next_step"`) {
			t.Errorf("rendered envelope should carry the next_step key:\n%s", doc)
		}
	})

	t.Run("next_step omitted (not null) for the internal-error fallback", func(t *testing.T) {
		env := envFor(errSomethingUnexpected())
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
		env := envFor(&apiclient.ResponseError{StatusCode: 403, Body: []byte(body)})
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
		env := envFor(&apiclient.ResponseError{StatusCode: 500, Body: []byte(`<html>boom</html>`)})
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
		env := envFor(&apiclient.TransportError{})
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
		env := envFor(refined)
		if env.Error.Status != 403 {
			t.Errorf("Status = %d, want 403 after refinement", env.Error.Status)
		}
		if len(env.Error.Body) == 0 {
			t.Error("Body must survive the refinement — the *ProblemError unwraps to the *ResponseError")
		}
	})

	t.Run("feature key present and naming the gate for a recognized plan-limit 403 (061)", func(t *testing.T) {
		body := `{"type":"about:blank","title":"Forbidden","detail":"Forbidden"}`
		// A recognized plan-gate 403 reaches the mapper as the refined *ProblemError
		// wrapping a *ResponseError carrying the gated operation's method/path.
		refined := apiclient.ExtractProblem(&apiclient.ResponseError{
			StatusCode: 403, Method: "POST", Path: "/proposals/prp_0123/propose", Body: []byte(body),
		})
		env := envFor(refined)
		if env.Error.Feature != "Premium async proposals" {
			t.Errorf("Feature = %q, want %q", env.Error.Feature, "Premium async proposals")
		}
		if env.Error.Kind != "permission" {
			t.Errorf("Kind = %q, want permission (the 403 stays PermissionError)", env.Error.Kind)
		}
		doc := jsonOf(t, env)
		if !strings.Contains(doc, `"feature": "Premium async proposals"`) {
			t.Errorf("rendered envelope should carry the distinct feature key:\n%s", doc)
		}
		// Declaration order: message → next_step → feature → kind → status → body.
		order := []string{`"message"`, `"next_step"`, `"feature"`, `"kind"`, `"status"`, `"body"`}
		last := -1
		for _, key := range order {
			at := strings.Index(doc, key)
			if at < 0 {
				t.Fatalf("key %s missing from a recognized plan-limit envelope:\n%s", key, doc)
			}
			if at < last {
				t.Errorf("key %s is out of declaration order:\n%s", key, doc)
			}
			last = at
		}
	})

	t.Run("proposal_id and validation_alerts present for an invalid create (078)", func(t *testing.T) {
		env := envFor(&invalidCreateError{
			ProposalID: "prp_0123",
			Alerts: []glassfrog.ValidationAlert{
				{Severity: "error", Path: "name", Message: "Can't update the Cloud Foundations role during this meeting."},
				{Severity: "warning", Path: "changes/0", Message: "Second."},
			},
		})
		if env.Error.Kind != "invalid-create" {
			t.Errorf("Kind = %q, want invalid-create", env.Error.Kind)
		}
		if env.Error.ProposalID != "prp_0123" {
			t.Errorf("ProposalID = %q, want prp_0123 (never withheld on this failure)", env.Error.ProposalID)
		}
		want := []output.ValidationAlert{
			{Severity: "error", Path: "name", Message: "Can't update the Cloud Foundations role during this meeting."},
			{Severity: "warning", Path: "changes/0", Message: "Second."},
		}
		if len(env.Error.ValidationAlerts) != len(want) {
			t.Fatalf("ValidationAlerts carried %d entries, want %d", len(env.Error.ValidationAlerts), len(want))
		}
		for i := range want {
			if env.Error.ValidationAlerts[i] != want[i] {
				t.Errorf("ValidationAlerts[%d] = %+v, want %+v (server order, field-by-field copy)", i, env.Error.ValidationAlerts[i], want[i])
			}
		}
		// No exchange failed and no plan gate fired, so those keys do not apply.
		if env.Error.Status != 0 || len(env.Error.Body) != 0 || env.Error.Feature != "" {
			t.Errorf("status/body/feature must be absent for an invalid create: %+v", env.Error)
		}
		doc := jsonOf(t, env)
		// The server's own key spellings, so an agent parses these exactly as it
		// parses the success document's alerts.
		for _, key := range []string{`"proposal_id"`, `"validation_alerts"`, `"severity"`, `"path"`, `"message"`} {
			if !strings.Contains(doc, key) {
				t.Errorf("rendered envelope should carry %s:\n%s", key, doc)
			}
		}
		// `message` carries the cause and must not enumerate the alerts — they have
		// their own key beside it.
		if strings.Contains(env.Error.Message, "Can't update the Cloud Foundations role") {
			t.Errorf("message enumerates the alerts:\n%s", env.Error.Message)
		}
		// Declaration order for THIS failure's key set. The plan-limit subtest above
		// pins message → next_step → feature → kind → status → body; the two new keys
		// follow body, and a plan-limit envelope carries neither, so each failure's
		// order is asserted over its own key set rather than one shared list.
		order := []string{`"message"`, `"next_step"`, `"kind"`, `"proposal_id"`, `"validation_alerts"`}
		last := -1
		for _, key := range order {
			at := strings.Index(doc, key)
			if at < 0 {
				t.Fatalf("key %s missing from an invalid-create envelope:\n%s", key, doc)
			}
			if at < last {
				t.Errorf("key %s is out of declaration order:\n%s", key, doc)
			}
			last = at
		}
	})

	t.Run("validation_alerts absent (not an empty array) when the server attached none (078)", func(t *testing.T) {
		// Both no-alert decode shapes: a nil slice (the key was absent) and a non-nil
		// empty slice (the server stated an empty list). omitempty omits a zero-length
		// slice, so no contract may promise `[]` — and the failure still fires.
		for name, alerts := range map[string][]glassfrog.ValidationAlert{
			"nil-slice-absent-key": nil,
			"empty-slice-stated":   {},
		} {
			env := envFor(&invalidCreateError{ProposalID: "prp_0123", Alerts: alerts})
			if len(env.Error.ValidationAlerts) != 0 {
				t.Errorf("[%s] ValidationAlerts = %+v, want none", name, env.Error.ValidationAlerts)
			}
			if env.Error.ProposalID != "prp_0123" || env.Error.Kind != "invalid-create" {
				t.Errorf("[%s] the failure still carries kind + proposal_id: %+v", name, env.Error)
			}
			doc := jsonOf(t, env)
			if strings.Contains(doc, "validation_alerts") {
				t.Errorf("[%s] the validation_alerts key must be absent, never an empty array:\n%s", name, doc)
			}
		}
	})

	t.Run("proposal_id and validation_alerts omitted for every other failure (078 omitempty)", func(t *testing.T) {
		// Every failure family that is not an invalid create: an API rejection (what a
		// rejected create renders), a plan-limit 403, a transport failure, and the
		// internal-error fallback. All four envelopes are byte-identical to today.
		for _, err := range []error{
			apiclient.ExtractProblem(&apiclient.ResponseError{StatusCode: 422, Body: []byte(`{"detail":"nope"}`)}),
			apiclient.ExtractProblem(&apiclient.ResponseError{StatusCode: 403, Method: "POST", Path: "/proposals/prp_0123/propose"}),
			&apiclient.TransportError{},
			errSomethingUnexpected(),
		} {
			env := envFor(err)
			if env.Error.ProposalID != "" || len(env.Error.ValidationAlerts) != 0 {
				t.Errorf("%T carries invalid-create fields: %+v", err, env.Error)
			}
			doc := jsonOf(t, env)
			for _, key := range []string{"proposal_id", "validation_alerts"} {
				if strings.Contains(doc, key) {
					t.Errorf("the %s key must be absent (omitempty) for %T:\n%s", key, err, doc)
				}
			}
		}
	})

	t.Run("feature key omitted (not null) for every non-plan-limit failure (061 omitempty)", func(t *testing.T) {
		// A generic 403 (no recognized gate), a non-403, and a transport failure all
		// carry no feature.
		for _, err := range []error{
			apiclient.ExtractProblem(&apiclient.ResponseError{StatusCode: 403, Method: "GET", Path: "/roles/role_0123"}),
			apiclient.ExtractProblem(&apiclient.ResponseError{StatusCode: 404}),
			&apiclient.TransportError{},
		} {
			env := envFor(err)
			if env.Error.Feature != "" {
				t.Errorf("Feature = %q, want empty for %T", env.Error.Feature, err)
			}
			doc := jsonOf(t, env)
			if strings.Contains(doc, `"feature"`) {
				t.Errorf("the feature key must be absent (omitempty), not null-keyed:\n%s", doc)
			}
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
			env := errorEnvelopeFor(Diagnose(fam.err), fam.err)
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
