package glassfrog

import (
	"encoding/json"
	"strings"
	"testing"
)

// tensionDocumentFixture is a representative createTension 201 body: the
// {data: Tension} document, using the API's snake_case names throughout so a
// missing JSON tag fails loud here, and carrying extra/unknown fields the decode
// must tolerate. Every nullable field is populated so each snake_case tag is
// pinned (the null variants are covered separately below).
const tensionDocumentFixture = `{
  "data": {
    "id": "ten_0123456789abcdef0123456789abcdef",
    "type": "tension",
    "body": "We ship faster than we update the roadmap.",
    "status": "unprocessed",
    "role_id": "role_0123456789abcdef0123456789abcdef",
    "sensed_by_id": "per_0123456789abcdef0123456789abcdef",
    "created_at": "2026-01-01T00:00:00Z",
    "updated_at": "2026-01-02T00:00:00Z",
    "label": "Roadmap drift",
    "meeting_type": "governance",
    "parent_role_id": "role_00000000000000000000000000000009",
    "unexpected_tension_field": "ignored"
  },
  "unexpected_top_level": {"anything": [1, 2, 3]}
}`

// TestTensionDocumentDecodesSnakeCaseFields pins every snake_case JSON tag on the
// Tension model decoded through the generic Document[Tension] envelope (034).
// encoding/json does not bridge underscores, so an untagged RoleID/SensedByID/etc.
// would silently never bind — this feeds the real snake_case payload and asserts
// each field decoded.
func TestTensionDocumentDecodesSnakeCaseFields(t *testing.T) {
	var doc Document[Tension]
	if err := json.Unmarshal([]byte(tensionDocumentFixture), &doc); err != nil {
		t.Fatalf("decoding the {data: Tension} fixture failed: %v", err)
	}
	ten := doc.Data
	cases := map[string]struct{ got, want string }{
		"id":             {ten.ID, "ten_0123456789abcdef0123456789abcdef"},
		"type":           {ten.Type, "tension"},
		"body":           {ten.Body, "We ship faster than we update the roadmap."},
		"status":         {ten.Status, "unprocessed"},
		"role_id":        {ten.RoleID, "role_0123456789abcdef0123456789abcdef"},
		"sensed_by_id":   {ten.SensedByID, "per_0123456789abcdef0123456789abcdef"},
		"created_at":     {ten.CreatedAt, "2026-01-01T00:00:00Z"},
		"updated_at":     {ten.UpdatedAt, "2026-01-02T00:00:00Z"},
		"label":          {ten.Label, "Roadmap drift"},
		"meeting_type":   {ten.MeetingType, "governance"},
		"parent_role_id": {ten.ParentRoleID, "role_00000000000000000000000000000009"},
	}
	for tag, c := range cases {
		if c.got != c.want {
			t.Errorf("%s = %q, want %q (the %s tag must bind)", tag, c.got, c.want, tag)
		}
	}
}

// TestTensionDecodesNullsToEmptyStrings pins the nullable-as-empty-string
// convention (Policy.Body / Project.RoleID precedent): a null body, role_id,
// sensed_by_id, label, meeting_type, and parent_role_id each decode to the empty
// string (which the render turns into an explicit-absence marker), never a panic.
func TestTensionDecodesNullsToEmptyStrings(t *testing.T) {
	body := `{"data":{"id":"ten_1","type":"tension","body":null,"status":"unprocessed","role_id":null,"sensed_by_id":null,"created_at":"t","updated_at":"t","label":null,"meeting_type":null,"parent_role_id":null}}`
	var doc Document[Tension]
	if err := json.Unmarshal([]byte(body), &doc); err != nil {
		t.Fatalf("decoding a tension with null nullable fields failed: %v", err)
	}
	ten := doc.Data
	for tag, got := range map[string]string{
		"body":           ten.Body,
		"role_id":        ten.RoleID,
		"sensed_by_id":   ten.SensedByID,
		"label":          ten.Label,
		"meeting_type":   ten.MeetingType,
		"parent_role_id": ten.ParentRoleID,
	} {
		if got != "" {
			t.Errorf("null %s should decode to empty string, got %q", tag, got)
		}
	}
}

// TestTensionToleratesUnknownFields pins forward-compatible decoding: the fixture
// carries unexpected fields at the top level and on the tension; the decode above
// succeeded — this guards the tolerance so a future strict decoder fails loud here.
func TestTensionToleratesUnknownFields(t *testing.T) {
	var doc Document[Tension]
	if err := json.Unmarshal([]byte(tensionDocumentFixture), &doc); err != nil {
		t.Errorf("unknown fields should be ignored, decode failed: %v", err)
	}
}

// TestNewTensionInputBodyOnly pins that a body-only input marshals to
// {"tension":{"body":…}} — no label, no meeting_type, and never a status or
// sensed_by key (the server owns those).
func TestNewTensionInputBodyOnly(t *testing.T) {
	got := mustMarshal(t, NewTensionInput("a tension", "", ""))
	want := `{"tension":{"body":"a tension"}}`
	if got != want {
		t.Errorf("body-only marshal = %s, want %s", got, want)
	}
	for _, forbidden := range []string{"status", "sensed_by", "label", "meeting_type"} {
		if strings.Contains(got, forbidden) {
			t.Errorf("body-only marshal must not emit %q, got %s", forbidden, got)
		}
	}
}

// TestNewTensionInputWithLabelAndMeetingType pins that a label + meeting-type
// input emits all three fields under the tension envelope.
func TestNewTensionInputWithLabelAndMeetingType(t *testing.T) {
	got := mustMarshal(t, NewTensionInput("a tension", "Roadmap drift", "governance"))
	want := `{"tension":{"body":"a tension","label":"Roadmap drift","meeting_type":"governance"}}`
	if got != want {
		t.Errorf("full marshal = %s, want %s", got, want)
	}
}

// --- TensionUpdateInput (partial-update body, all-omitempty incl. status) ---

// TestNewTensionUpdateInputOnlySuppliedFields pins the partial-update contract: a
// body + status input marshals to {"tension":{"body":…,"status":…}} only — the
// unsupplied label/meeting_type are dropped by omitempty, and status IS emitted
// (unlike capture, which has no status field at all).
func TestNewTensionUpdateInputOnlySuppliedFields(t *testing.T) {
	got := mustMarshal(t, NewTensionUpdateInput("b", "", "archived", ""))
	want := `{"tension":{"body":"b","status":"archived"}}`
	if got != want {
		t.Errorf("partial marshal = %s, want %s", got, want)
	}
	for _, forbidden := range []string{"label", "meeting_type"} {
		if strings.Contains(got, forbidden) {
			t.Errorf("an unsupplied field must be omitted, got %q in %s", forbidden, got)
		}
	}
}

// TestNewTensionUpdateInputEmptyOmitsAll pins that an all-empty input marshals to
// {"tension":{}} — every field is omitempty. The command guarantees this case never
// reaches the marshaller (the at-least-one-field precondition), but the type's omit
// behavior is pinned here.
func TestNewTensionUpdateInputEmptyOmitsAll(t *testing.T) {
	got := mustMarshal(t, NewTensionUpdateInput("", "", "", ""))
	want := `{"tension":{}}`
	if got != want {
		t.Errorf("empty marshal = %s, want %s", got, want)
	}
}

// TestNewTensionUpdateInputEachFieldUnderSnakeCaseKey pins that a value supplied for
// each of body/label/status/meeting_type appears under the correct snake_case key
// (encoding/json does not bridge underscores).
func TestNewTensionUpdateInputEachFieldUnderSnakeCaseKey(t *testing.T) {
	got := mustMarshal(t, NewTensionUpdateInput("a tension", "Roadmap drift", "processed", "governance"))
	want := `{"tension":{"body":"a tension","label":"Roadmap drift","status":"processed","meeting_type":"governance"}}`
	if got != want {
		t.Errorf("full marshal = %s, want %s", got, want)
	}
}

// TestCaptureInputUnchangedByUpdateFork pins plan ADR-1's byte-stable invariant:
// adding TensionUpdateInput must not have touched capture's wire output — a body-only
// create still marshals to {"tension":{"body":…}} with no status field.
func TestCaptureInputUnchangedByUpdateFork(t *testing.T) {
	got := mustMarshal(t, NewTensionInput("a tension", "", ""))
	want := `{"tension":{"body":"a tension"}}`
	if got != want {
		t.Errorf("capture input must stay byte-stable, got %s want %s", got, want)
	}
	if strings.Contains(got, "status") {
		t.Errorf("capture must keep its no-status invariant, got %s", got)
	}
}

func mustMarshal(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	return string(b)
}
