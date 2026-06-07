package cli

import (
	"strings"
	"testing"
)

// Each of the seven spec-sourced statuses passes validation.
func TestValidateStatus_SupportedValuesPass(t *testing.T) {
	for _, s := range []string{"archived", "cancelled", "completed", "current", "scheduled", "someday", "waiting"} {
		if err := validateStatus(s); err != nil {
			t.Errorf("status %q should be supported, got error %v", s, err)
		}
	}
}

// An empty value is the absent flag: no constraint, no error.
func TestValidateStatus_EmptyPasses(t *testing.T) {
	if err := validateStatus(""); err != nil {
		t.Errorf("an empty --status should pass (no filter), got %v", err)
	}
}

// An unsupported value is rejected with a usage error that names the offending
// value AND lists the supported set so an agent operator can self-correct.
func TestValidateStatus_UnsupportedRejectedNamingValueAndSet(t *testing.T) {
	err := validateStatus("done")
	if err == nil {
		t.Fatal("an unsupported --status value should be rejected")
	}
	msg := err.Error()
	if !strings.Contains(msg, `"done"`) {
		t.Errorf("the usage error should quote the unsupported value, got %q", msg)
	}
	for _, s := range []string{"archived", "cancelled", "completed", "current", "scheduled", "someday", "waiting"} {
		if !strings.Contains(msg, s) {
			t.Errorf("the usage error should list the supported status %q, got %q", s, msg)
		}
	}
}

// The supported set is reported in stable (sorted) order so the message is
// deterministic — the same input always yields the same text.
func TestValidateStatus_SupportedSetIsSorted(t *testing.T) {
	msg := validateStatus("bogus").Error()
	want := "archived, cancelled, completed, current, scheduled, someday, waiting"
	if !strings.Contains(msg, want) {
		t.Errorf("the supported set should appear sorted as %q, got %q", want, msg)
	}
}
